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

package v1

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils/native"
)

// log is for logging in this package.
var deploymentlog = logf.Log.WithName("deployment-resource")

type DeploymentValidator struct {
	recorder record.EventRecorder
	decoder  *admission.Decoder
}

type DeploymentMutator struct {
	recorder record.EventRecorder
	decoder  *admission.Decoder
}

// InjectDecoder injects the decoder into the admission webhook
func (v *DeploymentValidator) InjectDecoder(scheme *runtime.Scheme) error {
	decoder, err := admission.NewDecoder(scheme)
	if err != nil {
		return err
	}
	v.decoder = decoder
	return nil
}

func (m *DeploymentMutator) InjectRecorder(recorder record.EventRecorder) {
	m.recorder = recorder
}

// InjectDecoder injects the decoder into the admission webhook
func (m *DeploymentMutator) InjectDecoder(scheme *runtime.Scheme) error {
	decoder, err := admission.NewDecoder(scheme)
	if err != nil {
		return err
	}
	m.decoder = decoder
	return nil
}

func (v *DeploymentValidator) InjectRecorder(recorder record.EventRecorder) {
	v.recorder = recorder
}

// Handle validates the Deployment object
func (v *DeploymentValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	deployment := &v1.Deployment{}

	err := v.decoder.DecodeRaw(req.Object, deployment)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	operation := req.Operation
	switch operation {
	case admissionv1.Create:
		if err := v.ValidateCreate(*deployment); err != nil {
			return admission.Denied(err.Error())
		}
		return admission.Allowed("")
	case admissionv1.Update:
		oldDeployment := &v1.Deployment{}
		err := v.decoder.DecodeRaw(req.OldObject, oldDeployment)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		if err := v.ValidateUpdate(*deployment, *oldDeployment); err != nil {
			return admission.Denied(err.Error())
		}
		// v.recorder.Event(deployment, "Normal", "Updated", "Deployment updated")
		return admission.Allowed("")
	case admissionv1.Delete:
		return admission.Allowed("")
	}
	return admission.Allowed("")
}

//+kubebuilder:webhook:path=/mutate-apps-v1-deployment,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps,resources=deployments,verbs=create;update,versions=v1,name=mdeployment.kb.io,admissionReviewVersions=v1

// Handle validates the Deployment object
func (m *DeploymentMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	deployment := &v1.Deployment{}

	err := m.decoder.Decode(req, deployment)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	// get the hash of the Deployment object
	hash, err := getDeploymentTemplateHash(*deployment)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	// if the hash hasn't changed, admit the object
	oldHash := deployment.Labels[native.AdmissionWebhookVersionLabel]
	if oldHash == hash {
		return admission.Allowed("")
	}
	delete(deployment.Labels, utils.DefenseStatusLabel)
	// set the hash as an annotation on the Deployment object
	deployment.Labels[native.AdmissionWebhookVersionLabel] = hash
	// set the hash as an annotation on the Deployment spec.template
	deployment.Spec.Template.Labels[native.AdmissionWebhookVersionLabel] = hash
	patch, err := json.Marshal(deployment)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	// the object has been validated, so we can safely admit it.
	return admission.PatchResponseFromRaw(req.Object.Raw, patch)
}

// DeploymentWebhook is an Admission Webhook for Deployment objects
type DeploymentWebhook struct {
	Validator *DeploymentValidator
	Mutator   *DeploymentMutator
}

// SetupWebhookWithManager SetupWebhook registers the Admission Webhook
func (w *DeploymentWebhook) SetupWebhookWithManager(mgr manager.Manager) error {

	w.Validator = &DeploymentValidator{}
	w.Mutator = &DeploymentMutator{}
	if err := w.Validator.InjectDecoder(mgr.GetScheme()); err != nil {
		return err
	}
	w.Validator.InjectRecorder(mgr.GetEventRecorderFor("deployment-validator"))
	if err := w.Mutator.InjectDecoder(mgr.GetScheme()); err != nil {
		return err
	}
	w.Mutator.InjectRecorder(mgr.GetEventRecorderFor("deployment-mutator"))

	mgr.GetWebhookServer().Register("/validate-apps-v1-deployment", &admission.Webhook{
		Handler: admission.HandlerFunc(w.Validator.Handle),
	})
	mgr.GetWebhookServer().Register("/mutate-apps-v1-deployment", &admission.Webhook{
		Handler: admission.HandlerFunc(w.Mutator.Handle),
	})

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-apps-v1-deployment,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps,resources=deployments,verbs=create;update,versions=v1,name=vdeployment.kb.io,admissionReviewVersions=v1

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *DeploymentValidator) ValidateCreate(r v1.Deployment) error {
	deploymentlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *DeploymentValidator) ValidateUpdate(r v1.Deployment, old v1.Deployment) error {
	deploymentlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	if _, ok := r.Labels[utils.IgnoredSuspendLabel]; ok {
		return nil
	}
	if _, ok := old.Labels[utils.SuspendLabel]; ok {
		return fmt.Errorf("deployment %s is suspended", old.Name)
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *DeploymentValidator) ValidateDelete(r v1.Deployment) error {
	deploymentlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

// getTemplateHash 计算 PodTemplateSpec 的 32 长度的 hash 值
func getDeploymentTemplateHash(deployment v1.Deployment) (string, error) {
	// 获取template，将label中的version字段删除，生成一个hash值，作为新的version
	deploymentTemplate := deployment.Spec.Template
	delete(deploymentTemplate.Labels, native.AdmissionWebhookVersionLabel)
	//记录日志
	logger := utils.NewLogger().WithName("getHash")
	//将deploymentTemplate转换为json格式
	jsonBytes, err := json.Marshal(deploymentTemplate)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Error marshalling object: %v", err))
		return "", err
	}
	//计算json格式的hash值
	hashBytes := md5.Sum(jsonBytes)
	return hex.EncodeToString(hashBytes[:]), nil
}
