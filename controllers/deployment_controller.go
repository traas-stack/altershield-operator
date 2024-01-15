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

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/apis/app.ops.cloud.alipay.com/v1alpha1"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/resource"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils/native"
)

const (
	deploymentWorkCount = 5
)

// DeploymentReconciler reconciles a OpsConfigInfo object
type DeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OpsConfigInfo object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.1/pkg/reconcile
func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("DeploymentReconciler Reconcile")
	deployment := &v1.Deployment{}
	if err := r.Get(ctx, req.NamespacedName, deployment); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	labels := deployment.Labels
	defensed := isDefensed(labels)
	if defensed {
		return ctrl.Result{}, nil
	}
	// 创建或者获取workload
	// create or get workload
	_, err := r.getOrCreateChangeWorkload(ctx, deployment)
	if err != nil {
		logger.Error(err, "DeploymentReconciler getOrCreateChangeWorkload error", utils.LogDeploymentResource, utils.GetResource(deployment))
		return ctrl.Result{}, err
	}
	// 给deployment打上防御标签
	// add defensed label to deployment
	if err = r.defenseProcessedDeployment(ctx, deployment); err != nil {
		logger.Error(err, "DeploymentReconciler defenseProcessedDeployment error", utils.LogDeploymentResource, utils.GetResource(deployment))
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Deployment{}).
		Owns(&v1alpha1.ChangeWorkload{}).
		// Set the maximum number of concurrency
		WithOptions(controller.Options{MaxConcurrentReconciles: deploymentWorkCount}).
		// Settings only handle pods that are created or updated and whose label exists CheckStatusLabel=true
		WithEventFilter(predicate.Funcs{
			UpdateFunc: r.handleUpdateEvent,
			CreateFunc: r.handleCreateEvent,
		}).
		Complete(r)
}

// handleUpdateEvent handles the event of updating a Pod
func (r *DeploymentReconciler) handleUpdateEvent(e event.UpdateEvent) bool {
	labels := e.ObjectNew.GetLabels()
	return r.handleEvent(labels)
}

// handleCreateEvent handles the event of creating a new Pod
func (r *DeploymentReconciler) handleCreateEvent(e event.CreateEvent) bool {
	labels := e.Object.GetLabels()
	return r.handleEvent(labels)
}

// handleEvent handles the event of updating or creating a Pod
func (r *DeploymentReconciler) handleEvent(labels map[string]string) bool {
	admissionWebhook := isAdmissionWebhook(labels)
	defenseStatus := isDefensed(labels)
	return admissionWebhook && !defenseStatus
}

// getOrCreateChangeWorkload 根据deployment获取或者创建workload
// getOrCreateChangeWorkload get or create workload by deployment
func (r *DeploymentReconciler) getOrCreateChangeWorkload(ctx context.Context, deployment *v1.Deployment) (workload *v1alpha1.ChangeWorkload, err error) {
	logger := log.FromContext(ctx).WithName("getOrCreateChangeWorkload")
	// 1、根据版本label获取workload
	// 1. get workload by version label
	workload, err = r.getChangeWorkloadByDeployment(ctx, deployment)
	if err != nil && errors.IsNotFound(err) {
		// create workload
		if workload, err = r.createNewChangeWorkload(ctx, deployment); err != nil {
			logger.Error(err, "getOrCreateChangeWorkload create new workload instance error", utils.LogDeploymentResource, utils.GetResource(deployment))
			return workload, err
		}
		return workload, nil
	}
	return workload, err
}

// getChangeWorkloadByDeployment 通过deployment版本拼装的workload名称获取workload
// getChangeWorkloadByDeployment get workload by deployment
func (r *DeploymentReconciler) getChangeWorkloadByDeployment(ctx context.Context, deployment *v1.Deployment) (workload *v1alpha1.ChangeWorkload, err error) {
	logger := log.FromContext(ctx).WithName("getChangeWorkloadByDeployment")
	workload = &v1alpha1.ChangeWorkload{}
	if err = r.Get(ctx, client.ObjectKey{Name: native.GetChangeWorkloadNameByDeployment(deployment),
		Namespace: deployment.Namespace}, workload); err != nil {
		logger.Error(err, "get workload error", utils.LogDeploymentResource, utils.GetResource(deployment))
	}
	return
}

// createNewChangeWorkload 新的Deployment类型ChangeWorkload并创建
// createNewChangeWorkload create new ChangeWorkload
func (r *DeploymentReconciler) createNewChangeWorkload(ctx context.Context, deployment *v1.Deployment) (workload *v1alpha1.ChangeWorkload, err error) {
	logger := log.FromContext(ctx).WithName("createNewChangeWorkload")
	// create workload
	changeWorkloadFactory := resource.NativeChangeWorkloadFactory{Deployment: deployment, Replicas: deployment.Spec.Replicas}
	instance := changeWorkloadFactory.NewInstance()
	workload, ok := instance.(*v1alpha1.ChangeWorkload)
	if !ok {
		err = fmt.Errorf("createNewChangeWorkload NewInstance workload type error")
		logger.Error(err, "createNewChangeWorkload NewInstance workload type error", utils.LogDeploymentResource, utils.GetResource(deployment))
		return workload, err
	} else {
		if err = controllerutil.SetControllerReference(deployment, workload, r.Scheme); err != nil {
			logger.Error(err, "DeploymentReconciler SetControllerReference error", utils.LogDeploymentResource, utils.GetResource(deployment), utils.LogChangeWorkloadResource, utils.GetResource(workload))
		}
		if err = r.Create(ctx, workload); err != nil {
			logger.Error(err, "create workload error", utils.LogDeploymentResource, utils.GetResource(deployment))
			return workload, err
		}
	}
	return workload, nil
}

// defenseProcessedPod 给deployment打上defense-status=Processed标记
// defenseProcessedPod sets the label defense-status=Processed for the deployment
func (r *DeploymentReconciler) defenseProcessedDeployment(ctx context.Context, deployment *v1.Deployment) (err error) {
	logger := log.FromContext(ctx).WithName("defenseProcessedDeployment")
	patch := client.MergeFrom(deployment.DeepCopy())
	deployment.Labels[utils.DefenseStatusLabel] = utils.DefenseStatusLabelProcessed
	delete(deployment.Labels, utils.IgnoredSuspendLabel)
	logger.Info("defense processed deployment", utils.LogPodResource, utils.GetResource(deployment))
	if err = r.Patch(ctx, deployment, patch); err != nil {
		logger.Error(err, "update deployment label error", utils.LogPodResource, utils.GetResource(deployment))
		return err
	}
	return
}

// isAdmissionWebhook 当前pod是否是admission-webhook管理的
// isAdmissionWebhook is admission webhook
func isAdmissionWebhook(labels map[string]string) bool {
	_, isAdmissionWebhook := labels[native.AdmissionWebhookVersionLabel]
	return isAdmissionWebhook
}

// isDefensed 当前否已经被管控
// isDefensed is defensed
func isDefensed(labels map[string]string) bool {
	_, isDefensed := labels[utils.DefenseStatusLabel]
	return isDefensed
}
