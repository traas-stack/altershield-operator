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
	altershield "github.com/traas-stack/altershield-operator/pkg/altershield/client"
	"github.com/traas-stack/altershield-operator/pkg/constants"
	"github.com/traas-stack/altershield-operator/pkg/utils"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sync"

	v1alpha1 "github.com/traas-stack/altershield-operator/apis/app.ops.cloud.alipay.com/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const controllerChangeDefense = "CHANGEDEFENSE"

// ChangeDefenseReconciler reconciles a ChangeDefense object
type ChangeDefenseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	AsClient *altershield.AltershieldClient
}

//+kubebuilder:rbac:groups=app.ops.cloud.alipay.com,resources=changedefenses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.ops.cloud.alipay.com,resources=changedefenses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.ops.cloud.alipay.com,resources=changedefenses/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ChangeDefense object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *ChangeDefenseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// get ChangeDefense object
	changeDefense := &v1alpha1.ChangeDefense{}
	err := r.Get(context.TODO(), req.NamespacedName, changeDefense)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	klog.Infof("start reconcile ChangeDefense %v", klog.KObj(changeDefense))
	if !changeDefense.DeletionTimestamp.IsZero() {
		// TODO: handle deletion, delete executions, then remove finalizer
		klog.Infof("ChangeDefense %v to be deleted", klog.KObj(changeDefense))
		return ctrl.Result{}, err
	}

	changeDefenseExecution, err := r.ensureChangeDefenseExecution(ctx, changeDefense)
	if err != nil {
		klog.Errorf("failed to ensure ChangeDefenseExecution for ChangeDefense %v: %v",
			klog.KObj(changeDefense), err)
		return ctrl.Result{Requeue: true}, err
	}

	if err = r.updateStatus(ctx, changeDefense, changeDefenseExecution); err != nil {
		klog.Errorf("failed to update ChangeDefense %v status: %v",
			klog.KObj(changeDefense), err)
		return ctrl.Result{Requeue: true}, err
	}
	return ctrl.Result{}, nil
}

func (r *ChangeDefenseReconciler) updateStatus(ctx context.Context,
	changeDefense *v1alpha1.ChangeDefense, changeDefenseExecution *v1alpha1.ChangeDefenseExecution) error {
	if changeDefenseExecution == nil {
		return nil
	}
	execStatus := changeDefenseExecution.Status.DefenseStatus
	if execStatus.CurrentBatch != len(changeDefense.Spec.DefenseStrategy.Workload.Steps) {
		return nil
	}
	newStatus := changeDefense.Status.DeepCopy()
	if (execStatus.Phase == v1alpha1.DefensePhasePassed) || (execStatus.Phase == v1alpha1.DefensePhaseSkipped) {
		newStatus.Phase = v1alpha1.DefensePhasePassed
	}
	return utils.UpdateChangeDefenseStatus(r.Client, ctx, changeDefense, newStatus)
}

func (r *ChangeDefenseReconciler) ensureChangeDefenseExecution(
	ctx context.Context, changeDefense *v1alpha1.ChangeDefense) (*v1alpha1.ChangeDefenseExecution, error) {
	// find target resource for change defense
	defenseTargetRef := changeDefense.Spec.Target.ObjectRef
	defenseTarget := &unstructured.Unstructured{}
	defenseTarget.SetGroupVersionKind(schema.FromAPIVersionAndKind(
		defenseTargetRef.APIVersion, defenseTargetRef.Kind),
	)
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: changeDefense.Namespace,
		Name:      defenseTargetRef.Name,
	}, defenseTarget); err != nil {
		return nil, fmt.Errorf("failed to get target object %v of ChangeDefense %v: %v",
			defenseTargetRef.Name, klog.KObj(changeDefense), err)
	}

	// get latest defense execution information recorded in target object
	latestExec, err := utils.GetLatestDefenseExecutionBrief(defenseTarget)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest defense execution brief for target %v: %v",
			klog.KObj(defenseTarget), err)
	}
	if latestExec == nil {
		klog.Infof("no related execution for %v", klog.KObj(defenseTarget))
		return nil, nil
	}

	// recycle stale executions
	currentDefenseExecs, err := r.getCurrentDefenseExecutions(ctx, changeDefense)
	if err != nil {
		return nil, fmt.Errorf("failed to get defense executions for target %v from altershield: %v",
			klog.KObj(defenseTarget), err)
	}
	var toRemove []*v1alpha1.ChangeDefenseExecution
	for _, ele := range currentDefenseExecs {
		if latestExec.DefenseExecID != ele.Spec.ID {
			toRemove = append(toRemove, ele)
		}
	}
	var wg sync.WaitGroup
	errChan := make(chan error)
	for _, changeDefenseExecution := range toRemove {
		wg.Add(1)
		go func(cde *v1alpha1.ChangeDefenseExecution) {
			defer wg.Done()
			if err := r.Delete(ctx, cde); client.IgnoreNotFound(err) != nil {
				errChan <- err
			}
		}(changeDefenseExecution)
	}
	wg.Wait()
	close(errChan)
	if len(errChan) > 0 {
		err = <- errChan
		return nil, fmt.Errorf("failed to recycle stale executions: %v", err)
	}

	// create ChangeDefenseExecution if not exists
	changeDefenseExecution := &v1alpha1.ChangeDefenseExecution{
		ObjectMeta: v1.ObjectMeta{
			Namespace: changeDefense.Namespace,
			Name: latestExec.BuildChangeDefenseExecutionName(),
		},
	}
	if _, err = controllerutil.CreateOrUpdate(ctx, r.Client, changeDefenseExecution, func() (err error) {
		if !changeDefenseExecution.DeletionTimestamp.IsZero() {
			return fmt.Errorf("ChangeDefenseExecution object %v being deleted", klog.KObj(changeDefenseExecution))
		}
		changeDefenseExecution.Spec.ID = latestExec.DefenseExecID
		changeDefenseCopy := changeDefense.DeepCopy()
		changeDefenseExecution.Spec.DefenseStrategy = changeDefenseCopy.Spec.DefenseStrategy
		changeDefenseExecution.Spec.Target = changeDefenseCopy.Spec.Target
		changeDefenseExecution.Spec.RiskPolicy = changeDefenseCopy.Spec.RiskPolicy
		if changeDefenseExecution.Labels == nil {
			changeDefenseExecution.Labels = make(map[string]string)
		}
		changeDefenseExecution.Labels[constants.LabelChangeDefense] = changeDefense.Name
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to create or update change defense execution: %v", err)
	}

	return changeDefenseExecution, nil
}

func (r *ChangeDefenseReconciler) getCurrentDefenseExecutions(
	ctx context.Context, changeDefense *v1alpha1.ChangeDefense) ([]*v1alpha1.ChangeDefenseExecution, error) {
	cdeList := &v1alpha1.ChangeDefenseExecutionList{}
	if err := r.List(ctx, cdeList,
		client.InNamespace(changeDefense.Namespace),
		client.MatchingLabels{
			constants.LabelChangeDefense: changeDefense.Name,
		},
	); err != nil {
		return nil, err
	}
	cdes := make([]*v1alpha1.ChangeDefenseExecution, len(cdeList.Items))
	for ii := 0; ii < len(cdeList.Items); ii++ {
		cdes[ii] = &cdeList.Items[ii]
	}
	return cdes, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChangeDefenseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerChangeDefense).
		For(&v1alpha1.ChangeDefense{}).
		Watches(&source.Kind{Type: &apps.Deployment{}}, &enqueueRequestForWorkload{
			reader: mgr.GetCache(),
			scheme: r.Scheme,
			kind: utils.DeploymentKind,
		}).
		Watches(&source.Kind{Type: &apps.StatefulSet{}}, &enqueueRequestForWorkload{
			reader: mgr.GetCache(),
			scheme: r.Scheme,
			kind: utils.StatefulSetKind,
		}).
		Watches(&source.Kind{Type: &v1alpha1.ChangeDefenseExecution{}}, &enqueueRequestForChangeDefenseExecution{
			reader: mgr.GetCache(),
			scheme: r.Scheme,
		}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 100,
			RecoverPanic:            true,
		}).
		Complete(r)
}
