package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/apis/app.ops.cloud.alipay.com/v1alpha1"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
)

// watchChangePodEventHandler handles the event of watching a ChangePod
func watchChangePodEventHandler(object client.Object) []reconcile.Request {
	// 获取pod
	changePod, ok := object.(*v1alpha1.ChangePod)
	if !ok || changePod == nil {
		return nil
	} else {
		if changePod.Status.Status != v1alpha1.ExecuteInit {
			return []reconcile.Request{{NamespacedName: utils.GetChangePodNameSpaceNameFromChangePod(*changePod)}}
		}
	}
	return nil
}

// changePodCreateHandleEvent handles the event of creating a ChangePod
func (r *ChangeWorkloadReconciler) changePodCreateHandleEvent(changePod *v1alpha1.ChangePod) bool {
	return changePod.Status.Status == v1alpha1.ExecuteDone
}

// changePodUpdateHandleEvent handles the event of updating a ChangePod
func (r *ChangeWorkloadReconciler) changePodUpdateHandleEvent(oldChangePod *v1alpha1.ChangePod, newChangePod *v1alpha1.ChangePod) bool {
	if oldChangePod.Status.Status == v1alpha1.ExecuteInit && newChangePod.Status.Status == v1alpha1.PreWait {
		return true
	}
	if newChangePod.Status.Status == v1alpha1.ExecuteDone {
		return true
	}
	return false
}

// changePodReconcile reconciles a ChangePod object
func (r *ChangeWorkloadReconciler) changePodReconcile(ctx context.Context, req ctrl.Request) (*v1alpha1.ChangeWorkload, error) {
	logger := log.FromContext(ctx).WithName("changePodReconcile")
	changePod := &v1alpha1.ChangePod{}
	if err := r.Get(ctx, req.NamespacedName, changePod); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	workload := &v1alpha1.ChangeWorkload{}
	changeWorkloadName := changePod.Spec.ChangeWorkloadId
	namespace := changePod.Namespace
	if err := r.Get(ctx, types.NamespacedName{Name: changeWorkloadName, Namespace: namespace}, workload); err != nil {
		logger.Error(err, "get workload error", utils.LogChangePodResource, utils.GetResource(changePod))
		return nil, err
	}
	return workload, nil
}
