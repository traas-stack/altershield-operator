/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mutating

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/traas-stack/altershield-operator/apis/app.ops.cloud.alipay.com/v1alpha1"
	altershield "github.com/traas-stack/altershield-operator/pkg/altershield/client"
	"github.com/traas-stack/altershield-operator/pkg/constants"
	"github.com/traas-stack/altershield-operator/pkg/types"
	"github.com/traas-stack/altershield-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime/schema"
	types2 "k8s.io/apimachinery/pkg/types"
	"net/http"
	"time"

	admissionv1 "k8s.io/api/admission/v1"
	apps "k8s.io/api/apps/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func init() {
	updateHandlers[gvkConverter(utils.DeploymentKind)] = WorkloadUpdateMutation
	updateHandlers[gvkConverter(utils.StatefulSetKind)] = WorkloadUpdateMutation
}

const (
	DefenseExecutionIDSuffixLength = 5
)

// GetTemplate return pod template spec for workload object
func getTemplateSpec(object client.Object) *corev1.PodTemplateSpec {
	switch o := object.(type) {
	case *apps.Deployment:
		return &o.Spec.Template
	case *apps.StatefulSet:
		return &o.Spec.Template
	default:
		return nil
	}
}

func shouldDefendChange(oldObj, newObj client.Object) bool {
	oldTemplate, newTemplate := getTemplateSpec(oldObj), getTemplateSpec(newObj)
	if (oldTemplate == nil) || (newTemplate == nil) {
		return false
	}

	oldTemplateClone, newTemplateClone := oldTemplate.DeepCopy(), newTemplate.DeepCopy()
	return !equality.Semantic.DeepEqual(oldTemplateClone, newTemplateClone)
}

// Handle handles admission requests.
func WorkloadUpdateMutation(ctx context.Context, req *admission.Request, handler *Handler) admission.Response {
	// ignore non-update or subresource requests
	if req.Operation != admissionv1.Update || req.SubResource != "" {
		return admission.Allowed("")
	}
	// only check native workload updates
	if req.Kind.Group != apps.SchemeGroupVersion.Group {
		return admission.Allowed("")
	}
	var gvk schema.GroupVersionKind
	var oldObj, newObj client.Object
	switch req.Kind.Kind {
	case utils.DeploymentKind.Kind:
		gvk = utils.DeploymentKind
		oldObj = &apps.Deployment{}
		if err := handler.Decoder.DecodeRaw(req.OldObject, oldObj); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		newObj = &apps.Deployment{}
		if err := handler.Decoder.DecodeRaw(req.Object, newObj); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
	case utils.StatefulSetKind.Kind:
		gvk = utils.StatefulSetKind
		oldObj = &apps.StatefulSet{}
		if err := handler.Decoder.DecodeRaw(req.OldObject, oldObj); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		newObj = &apps.StatefulSet{}
		if err := handler.Decoder.DecodeRaw(req.Object, newObj); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
	default:
		return admission.Allowed("")
	}
	changeDefense, err := utils.FetchChangeDefense(handler.Client, gvk, newObj)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if changeDefense == nil {
		return admission.Allowed("")
	}
	if !shouldDefendChange(oldObj, newObj) {
		return admission.Allowed("")
	}

	defenseExec, err := triggerDefenseExec(handler, newObj)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	_, err = defensePreCheck(handler, defenseExec)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	newStatus := changeDefense.Status.DeepCopy()
	newStatus.CurrentExecutionID = defenseExec.ID
	newStatus.Phase = v1alpha1.DefensePhasePreCheck
	if err = utils.UpdateChangeDefenseStatus(handler.Client, ctx, changeDefense, newStatus); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	verdict, err := getOrderPreCheckResult(ctx, handler.Client, defenseExec.ID, changeDefense)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if !verdict.Passed {
		return admission.Denied(verdict.Msg)
	}

	defenseExecBrief := &types.DefenseExecutionBrief{
		Generation: newObj.GetGeneration() + 1,
		DefenseExecID: defenseExec.ID,
		ChangeDefenseName: changeDefense.Name,
	}
	defenseExecBriefRaw, err := json.Marshal(defenseExecBrief)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	patchAnnotation(newObj, constants.AnnotationLatestDefenseExec, string(defenseExecBriefRaw))

	klog.Infof("patch change defense info %v for workload %v",
		string(defenseExecBriefRaw), klog.KObj(newObj))

	return generatePatchResponse(req.Object.Raw, newObj)
}

func getOrderPreCheckResult(ctx context.Context, apiClient client.Client,
	executionID string, changeDefense *v1alpha1.ChangeDefense) (*types.ChangeCheckVerdict, error) {
	timeout := time.After(9 * time.Second)
	ticker := time.Tick(3 * time.Second)
	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("get change defense order %v pre-check result timeout", klog.KObj(changeDefense))
		case <-ticker:
			changeDefenseCurr := &v1alpha1.ChangeDefense{}
			if err := apiClient.Get(ctx, types2.NamespacedName{
				Namespace: changeDefense.Namespace,
				Name:      changeDefense.Name,
			}, changeDefenseCurr); err != nil {
				klog.Errorf("failed to get change defense in order pre-check: %v", err)
				return nil, err
			}
			if executionID != changeDefenseCurr.Status.CurrentExecutionID {
				return nil, fmt.Errorf("%v covered by newer execution: %v",
					executionID, changeDefenseCurr.Status.CurrentExecutionID)
			}
			if changeDefenseCurr.Status.Phase != v1alpha1.DefensePhasePreCheck {
				return &types.ChangeCheckVerdict{
					Passed: changeDefenseCurr.Status.Phase == v1alpha1.DefensePhaseProgressing,
					Msg:     changeDefenseCurr.Status.Verdict,
				}, nil
			}
		}
	}
}

func generatePatchResponse(original []byte, obj client.Object) admission.Response {
	marshaled, err := json.Marshal(obj)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	return admission.PatchResponseFromRaw(original, marshaled)
}

func patchAnnotation(obj client.Object, key, value string) {
	if obj.GetAnnotations() == nil {
		obj.SetAnnotations(make(map[string]string))
	}
	obj.GetAnnotations()[key] = value
}

// TODO: invoke altershield
func triggerDefenseExec(handler *Handler, workloadObj client.Object) (*types.ChangeDefenseExecution, error) {
	defenseAppName := fmt.Sprintf("%v-%v", workloadObj.GetNamespace(), workloadObj.GetName())
	executionID := fmt.Sprintf("%v-%v-%v", defenseAppName, workloadObj.GetGeneration() + 1,
		utils.RandomString(DefenseExecutionIDSuffixLength))

	_, err := handler.altershieldClient.SubmitChangeExecOrder(altershield.NewSubmitChangeExecOrderRequest(executionID, defenseAppName))
	if err != nil {
		return nil, err
	}
	return &types.ChangeDefenseExecution{
		ID: executionID,
	}, nil
}

// TODO: invoke altershield
func defensePreCheck(handler *Handler, defenseExec *types.ChangeDefenseExecution) (nodeID string, err error) {
	resp, err := handler.altershieldClient.SubmitChangeExecOrderStartNotify(
		altershield.NewSubmitChangeExecOrderStartNotifyRequest(defenseExec.ID),
	)
	if err != nil {
		return "", err
	}

	return resp.NodeID, nil
}
