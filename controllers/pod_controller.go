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
	"strconv"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils/native"
)

const (
	podWorkCount = 5
)

// PodReconciler reconciles a pod object
type PodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// 2、Pod新版本发布后
	// 2、pod new version publish
	pod := &v1.Pod{}
	if err := r.Get(ctx, req.NamespacedName, pod); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	finished := isFinished(pod)
	if finished {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, r.markAsFinished(ctx, pod)
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Pod{}).
		// Set the maximum number of concurrency
		WithOptions(controller.Options{MaxConcurrentReconciles: podWorkCount}).
		// Settings only handle pods that are created or updated and whose label exists CheckStatusLabel=true
		WithEventFilter(predicate.Funcs{
			UpdateFunc: r.handleUpdateEvent,
			CreateFunc: r.handleCreateEvent,
		}).
		Complete(r)
}

// handleUpdateEvent handles the event of updating a Pod
func (r *PodReconciler) handleUpdateEvent(e event.UpdateEvent) bool {
	newPod, newOk := e.ObjectNew.(*v1.Pod)
	oldPod, oldOk := e.ObjectOld.(*v1.Pod)
	if !newOk || !oldOk {
		return false
	}
	finished := isFinished(newPod)
	if finished {
		return false
	}
	running := isRunning(newPod) && !isRunning(oldPod)
	hasVersion := isHasVersion(newPod)
	return running && hasVersion
}

// handleCreateEvent handles the event of creating a new Pod
func (r *PodReconciler) handleCreateEvent(e event.CreateEvent) bool {
	pod, ok := e.Object.(*v1.Pod)
	if !ok {
		return false
	}
	finished := isFinished(pod)
	if finished {
		return false
	}
	running := isRunning(pod)
	hasVersion := isHasVersion(pod)
	return running && hasVersion
}

// markAsFinished 给pod打上操作完成标签
// markAsFinished pod mark as finished label
func (r *PodReconciler) markAsFinished(ctx context.Context, pod *v1.Pod) (err error) {
	logger := log.FromContext(ctx).WithName("markAsFinished")
	pod.Labels[utils.OperateFinishedLabel] = strconv.Itoa(int(time.Now().Unix()))
	logger.Info("pod finished", utils.LogPodResource, utils.GetResource(pod))
	if err = r.Update(ctx, pod); err != nil {
		if utils.IsObjectModifiedErr(err) {
			logger.Error(err, "update pod mark as finished label failed", utils.LogPodResource, utils.GetResource(pod))
		}
		return err
	}
	return
}

// isRunning 当前pod是否是运行中
// isRunning pod is running
func isRunning(pod *v1.Pod) bool {
	return pod.Status.Phase == v1.PodRunning
}

// isFinished 当前pod是否有操作完成标签
// isFinished pod is finished label
func isFinished(pod *v1.Pod) bool {
	_, finished := pod.Labels[utils.OperateFinishedLabel]
	return finished
}

// isHasVersion 当前pod是否已经打上版本标签
// isHasVersion pod is has version label
func isHasVersion(pod *v1.Pod) bool {
	_, hasVersion := pod.Labels[native.AdmissionWebhookVersionLabel]
	return hasVersion
}
