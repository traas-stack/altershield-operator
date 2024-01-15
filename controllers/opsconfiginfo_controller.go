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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"

	appv1alpha1 "gitlab.alipay-inc.com/common_release/altershieldoperator/apis/app.ops.cloud.alipay.com/v1alpha1"
)

// OpsConfigInfoReconciler reconciles a OpsConfigInfo object
type OpsConfigInfoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=app.ops.cloud.alipay.com,resources=opsconfiginfoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.ops.cloud.alipay.com,resources=opsconfiginfoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.ops.cloud.alipay.com,resources=opsconfiginfoes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OpsConfigInfo object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.1/pkg/reconcile
func (r *OpsConfigInfoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// get OpsConfigInfo
	opsConfigInfo := &appv1alpha1.OpsConfigInfo{}
	if err := r.Get(ctx, req.NamespacedName, opsConfigInfo); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// Determine if Namespace is utils.altershieldoperator Namespace, if not, delete
	if opsConfigInfo.Namespace != utils.AlterShieldOperatorNamespace {
		if err := r.Delete(ctx, opsConfigInfo); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	// Determine the spec.type of ops ConfigInfo
	switch opsConfigInfo.Spec.Type {
	case utils.ConfigTypeIsBranch:
		return r.configTypeIsBranchHandel(ctx, opsConfigInfo)
	case utils.ConfigTypeIsBlockingUp:
		return r.configTypeIsBlockingUpHandel(ctx, opsConfigInfo)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpsConfigInfoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha1.OpsConfigInfo{}).
		Complete(r)
}

func (r *OpsConfigInfoReconciler) configTypeIsBranchHandel(ctx context.Context, opsConfigInfo *appv1alpha1.OpsConfigInfo) (ctrl.Result, error) {
	// Determine if name is ConfigNameIsBranch, and if not, delete it
	logger := log.FromContext(ctx)
	if opsConfigInfo.Name != utils.ConfigNameIsBranch {
		if err := r.Delete(ctx, opsConfigInfo); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	} else {
		enable := opsConfigInfo.Spec.Enable
		utils.ConfigIsBatchChannel <- enable
		if enable {
			// Determine whether it is int type, and if it is strong, it is converted to int
			count, err := strconv.Atoi(opsConfigInfo.Spec.Content)
			if err != nil {
				logger.Error(err, "configTypeIsBranchHandel strconv.Atoi error")
				utils.ConfigBatchCountChannel <- utils.NumberOne
			} else {
				utils.ConfigBatchCountChannel <- count
			}
			return ctrl.Result{}, nil
		} else {
			utils.ConfigBatchCountChannel <- utils.NumberOne
			return ctrl.Result{}, nil
		}
	}
}

func (r *OpsConfigInfoReconciler) configTypeIsBlockingUpHandel(ctx context.Context, opsConfigInfo *appv1alpha1.OpsConfigInfo) (ctrl.Result, error) {
	// Determine if name is ConfigNameIsBlockingUp, and if not, delete it
	if opsConfigInfo.Name != utils.ConfigNameIsBlockingUp {
		if err := r.Delete(ctx, opsConfigInfo); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	} else {
		enable := opsConfigInfo.Spec.Enable
		utils.ConfigIsBlockingUpChannel <- enable
		return ctrl.Result{}, nil
	}
}
