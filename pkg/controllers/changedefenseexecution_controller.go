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

package controllers

import (
	"context"
	"fmt"
	v1alpha1 "github.com/traas-stack/altershield-operator/apis/app.ops.cloud.alipay.com/v1alpha1"
	ac "github.com/traas-stack/altershield-operator/pkg/altershield/client"
	"github.com/traas-stack/altershield-operator/pkg/constants"
	"github.com/traas-stack/altershield-operator/pkg/types"
	"github.com/traas-stack/altershield-operator/pkg/utils"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strconv"
	"time"
)

const controllerChangeDefenseExecution = "CHANGEDEFENSEEXEC"

// ChangeDefenseExecutionReconciler reconciles a ChangeDefenseExecution object
type ChangeDefenseExecutionReconciler struct {
	client.Client

	Scheme       *runtime.Scheme
	AsClient     *ac.AltershieldClient
}

//+kubebuilder:rbac:groups=app.ops.cloud.alipay.com,resources=changedefenseexecutions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.ops.cloud.alipay.com,resources=changedefenseexecutions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.ops.cloud.alipay.com,resources=changedefenseexecutions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ChangeDefenseExecution object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *ChangeDefenseExecutionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.Infof("reconcile ChangeDefenseExecution request %v", req.String())

	// get ChangeDefenseExecution object
	changeDefenseExec := &v1alpha1.ChangeDefenseExecution{}
	err := r.Get(context.TODO(), req.NamespacedName, changeDefenseExec)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	klog.Infof("start reconcile ChangeDefenseExecution %v", klog.KObj(changeDefenseExec))
	if !changeDefenseExec.DeletionTimestamp.IsZero() {
		klog.Infof("ChangeDefenseExecution %v to be deleted", klog.KObj(changeDefenseExec))
		return ctrl.Result{}, err
	}

	// find target resource for change defense
	workloadRef := changeDefenseExec.Spec.Target.ObjectRef
	workloadInfo, err := r.getWorkloadInfo(ctx, changeDefenseExec.Namespace, workloadRef)
	if err != nil {
		klog.Errorf("failed to get workload info for ChangeDefenseExecution %v: %v",
			klog.KObj(changeDefenseExec), err)
		return ctrl.Result{Requeue: true}, err
	}
	latestExecution, err := utils.GetLatestDefenseExecutionBrief(workloadInfo.Obj)
	if err != nil {
		klog.Error("failed to get latest exeuction brief for workload %v: %v",
			klog.KObj(workloadInfo.Obj), err)
		return ctrl.Result{Requeue: true}, err
	}
	if latestExecution.DefenseExecID != changeDefenseExec.Spec.ID {
		klog.Infof("not latest defense execution, abort")
		return ctrl.Result{}, nil
	}

	var (
		newStatus = getInitializedStatus(changeDefenseExec)
		result ctrl.Result
	)

	switch newStatus.DefenseStatus.Phase {
	case v1alpha1.DefensePhaseInitial:
		result, err = r.processInitial(changeDefenseExec, newStatus, workloadInfo)
	case v1alpha1.DefensePhaseProgressing:
		result ,err = r.processProgressing(changeDefenseExec, newStatus, workloadInfo)
	case v1alpha1.DefensePhaseObserving:
		result, err = r.processObserving(changeDefenseExec, newStatus, workloadInfo)
	case v1alpha1.DefensePhasePassed:
		fallthrough
	case v1alpha1.DefensePhaseSkipped:
		result, err = r.proceed(changeDefenseExec, newStatus)
	case v1alpha1.DefensePhaseFailed:
	}
	if err != nil {
		klog.Errorf("failed to process ChangeDefenseExecution %v in phase %v: %v",
			klog.KObj(changeDefenseExec), changeDefenseExec.Status.DefenseStatus.Phase, err)
		return ctrl.Result{Requeue: true}, err
	}

	if err = utils.UpdateChangeDefenseExecutionStatus(r.Client, ctx, changeDefenseExec, newStatus); err != nil {
		return ctrl.Result{Requeue: true}, fmt.Errorf("failed to update ChangeDefenseExecution %v status: %v",
			klog.KObj(changeDefenseExec), err)
	}
	return result, nil
}

func (r *ChangeDefenseExecutionReconciler) proceed(changeDefenseExec *v1alpha1.ChangeDefenseExecution,
	newStatus *v1alpha1.ChangeDefenseExecutionStatus) (ctrl.Result, error) {
	currentBatch := newStatus.DefenseStatus.CurrentBatch
	steps := changeDefenseExec.Spec.DefenseStrategy.Workload.Steps

	if currentBatch < len(steps) {
		newStatus.DefenseStatus.CurrentBatch++
		newStatus.DefenseStatus.Phase = v1alpha1.DefensePhaseInitial
		newStatus.DefenseStatus.LastTransitionTime = &metav1.Time{Time: time.Now()}
	}
	return ctrl.Result{}, nil
}

func (r *ChangeDefenseExecutionReconciler) processInitial(changeDefenseExec *v1alpha1.ChangeDefenseExecution,
	newStatus *v1alpha1.ChangeDefenseExecutionStatus, info *types.WorkloadInfo) (ctrl.Result, error) {
	currentBatch := newStatus.DefenseStatus.CurrentBatch
	steps := changeDefenseExec.Spec.DefenseStrategy.Workload.Steps

	// label target object to resume
	if err := r.resumeProgress(info.Obj); err != nil {
		klog.Errorf("failed to resume change progress: %v", err)
		return ctrl.Result{Requeue: true}, err
	}

	request := ac.NewSubmitChangeExecBatchStartNotifyRequest(changeDefenseExec.Spec.ID, currentBatch, len(steps), info)
	response, err := r.AsClient.SubmitChangeExecBatchStartNotify(request)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	newStatus.DefenseStatus.NodeID = response.NodeID
	newStatus.DefenseStatus.Phase = v1alpha1.DefensePhasePreCheck
	newStatus.DefenseStatus.LastTransitionTime = &metav1.Time{Time: time.Now()}
	return ctrl.Result{}, nil
}

func (r *ChangeDefenseExecutionReconciler) resumeProgress(in client.Object) error {
	if in.GetLabels() == nil {
		in.SetLabels(make(map[string]string))
	}
	in.GetLabels()[constants.LabelContinue] = strconv.FormatInt(time.Now().UnixMilli(), 10)
	return r.Update(context.TODO(), in)
}

func (r *ChangeDefenseExecutionReconciler) processProgressing(changeDefenseExec *v1alpha1.ChangeDefenseExecution,
	newStatus *v1alpha1.ChangeDefenseExecutionStatus, workloadInfo *types.WorkloadInfo) (ctrl.Result, error) {
	currentBatch := newStatus.DefenseStatus.CurrentBatch
	steps := changeDefenseExec.Spec.DefenseStrategy.Workload.Steps

	readyPods, err := utils.GetReadyPodsOfLatestRevision(r.Client, workloadInfo.Obj)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	if len(readyPods) >= utils.GetBatchReplicasBound(steps[currentBatch - 1].Partition, int(workloadInfo.Replicas)) {
		newStatus.DefenseStatus.Phase = v1alpha1.DefensePhaseObserving
		newStatus.DefenseStatus.LastTransitionTime = &metav1.Time{Time: time.Now()}
	}
	return ctrl.Result{}, nil
}

func (r *ChangeDefenseExecutionReconciler) processObserving(changeDefenseExec *v1alpha1.ChangeDefenseExecution,
	newStatus *v1alpha1.ChangeDefenseExecutionStatus, workloadInfo *types.WorkloadInfo) (ctrl.Result, error) {
	currentBatch := newStatus.DefenseStatus.CurrentBatch
	steps := changeDefenseExec.Spec.DefenseStrategy.Workload.Steps

	expectedTime := newStatus.DefenseStatus.LastTransitionTime.Add(
		time.Duration(utils.Int32IndirectOrZero(steps[currentBatch - 1].CheckAfterComplete)) * time.Second)
	if expectedTime.After(time.Now()) {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	updatedPods, err := utils.GetPodsOfLatestRevision(r.Client, workloadInfo.Obj)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}
	podInfoList := filterPodInfoList(updatedPods, newStatus.DefenseStatus.TargetStatus.PodBatches[:currentBatch - 1])

	newStatus.DefenseStatus.Phase = v1alpha1.DefensePhasePostCheck
	newStatus.DefenseStatus.LastTransitionTime = &metav1.Time{Time: time.Now()}
	if len(newStatus.DefenseStatus.TargetStatus.PodBatches) < currentBatch {
		newStatus.DefenseStatus.TargetStatus.PodBatches =
			append(newStatus.DefenseStatus.TargetStatus.PodBatches, v1alpha1.BatchPodInfo{})
	}
	newStatus.DefenseStatus.TargetStatus.PodBatches[currentBatch - 1] = v1alpha1.BatchPodInfo{Pods: podInfoList}
	request := ac.NewSubmitChangeFinishNotifyRequest(changeDefenseExec.Spec.ID, newStatus.DefenseStatus.NodeID)
	response, err := r.AsClient.SubmitChangeFinishNotify(request)
	if err != nil {
		klog.Errorf("failed to submit post check for batch %v: %v", currentBatch, err)
		return ctrl.Result{Requeue: true}, err
	}
	newStatus.DefenseStatus.NodeID = response.NodeID

	return ctrl.Result{}, nil
}

func filterPodInfoList(totalPods []*corev1.Pod, previousBatches []v1alpha1.BatchPodInfo) []v1alpha1.PodInfo {
	// get new pods
	previousPodSet := make(map[string]struct{})
	for _, batch := range previousBatches {
		for _, bpo := range batch.Pods {
			previousPodSet[bpo.UID] = struct{}{}
		}
	}

	var filteredPodInfoList []v1alpha1.PodInfo
	for _, pod := range totalPods {
		if _, ok := previousPodSet[string(pod.UID)]; ok {
			continue
		}
		filteredPodInfoList = append(filteredPodInfoList, v1alpha1.PodInfo{
			Name: pod.Name,
			IP:   pod.Status.PodIP,
			UID:  string(pod.UID),
		})
	}
	return filteredPodInfoList
}

func (r *ChangeDefenseExecutionReconciler) getWorkloadInfo(ctx context.Context, namespace string, workloadRef *v1alpha1.ObjectRef) (*types.WorkloadInfo, error) {
	gvk := schema.FromAPIVersionAndKind(workloadRef.APIVersion, workloadRef.Kind)
	switch {
	case (utils.DeploymentKind.Group == gvk.Group) && (utils.DeploymentKind.Kind == gvk.Kind):
		workload := &apps.Deployment{}
		if err := r.Get(ctx, client.ObjectKey{
			Namespace: namespace,
			Name:      workloadRef.Name,
		}, workload); err != nil {
			klog.Errorf("failed to get target deployment %v: %v", workloadRef.Name, err)
			return nil, err
		}
		return &types.WorkloadInfo{
			GVK:      utils.DeploymentKind,
			Obj:      workload,
			Replicas: utils.Int32IndirectOrZero(workload.Spec.Replicas),
		}, nil
	case (utils.StatefulSetKind.Group == gvk.Group) && (utils.StatefulSetKind.Kind == gvk.Kind):
		workload := &apps.StatefulSet{}
		if err := r.Get(ctx, client.ObjectKey{
			Namespace: namespace,
			Name:      workloadRef.Name,
		}, workload); err != nil {
			klog.Errorf("failed to get target statefulset %v: %v", workloadRef.Name, err)
			return nil, err
		}
		return &types.WorkloadInfo{
			GVK:      utils.StatefulSetKind,
			Obj:      workload,
			Replicas: utils.Int32IndirectOrZero(workload.Spec.Replicas),
		}, nil
	}
	return nil, fmt.Errorf("unsupported workload type: %v", gvk.String())
}

func getInitializedStatus(cde *v1alpha1.ChangeDefenseExecution) *v1alpha1.ChangeDefenseExecutionStatus {
	status := &cde.Status
	newStatus := status.DeepCopy()
	if len(status.DefenseStatus.Phase) == 0 {
		newStatus.DefenseStatus.CurrentBatch = 1
		newStatus.DefenseStatus.Phase = v1alpha1.DefensePhaseInitial
		newStatus.DefenseStatus.TargetStatus.PodBatches = make([]v1alpha1.BatchPodInfo, 0)
	}
	return newStatus
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChangeDefenseExecutionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerChangeDefenseExecution).
		For(&v1alpha1.ChangeDefenseExecution{}).
		Watches(&source.Kind{Type: &corev1.Pod{}}, &enqueueRequestForPod{
			reader: mgr.GetCache(),
			scheme: r.Scheme,
		}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 100,
			RecoverPanic:            true,
		}).
		Complete(r)
}
