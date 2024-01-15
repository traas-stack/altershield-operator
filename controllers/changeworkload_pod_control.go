package controllers

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/apis/app.ops.cloud.alipay.com/v1alpha1"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils/native"
)

// watchPodEventHandler 监听pod的事件
func watchPodEventHandler(object client.Object) []reconcile.Request {
	// 获取pod
	pod, ok := object.(*v1.Pod)
	if !ok || pod == nil {
		return nil
	} else {
		// 如果pod的label中有finished标签，并且没有defensed标签，则返回该pod的changeWorkload的request
		// If the pod has the finished label and no defensed label, return the request of the changeWorkload of the pod
		finished := isFinished(pod)
		defensed := isDefensed(pod.Labels)
		if finished && !defensed {
			// 获取changeWorkload的request
			// Get the request of changeWorkload
			return []reconcile.Request{{NamespacedName: utils.GetPodNameSpaceNameFromPod(*pod)}}
		}
	}
	return nil
}

// podHandleEvent handles the event of updating a Pod
func (r *ChangeWorkloadReconciler) podHandleEvent(pod *v1.Pod) bool {
	finished := isFinished(pod)
	defensed := isDefensed(pod.Labels)
	if finished && !defensed {
		return true
	}
	return false
}

// podReconcile reconciles a Pod
func (r *ChangeWorkloadReconciler) podReconcile(ctx context.Context, req ctrl.Request) (*v1alpha1.ChangeWorkload, error) {
	logger := log.FromContext(ctx).WithName("podReconcile")
	pod := &v1.Pod{}
	if err := r.Get(ctx, req.NamespacedName, pod); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	notFinished := !isFinished(pod)
	if notFinished {
		return nil, nil
	}
	deployment, err := r.getDeploymentByPod(ctx, pod)
	if err != nil {
		logger.Error(err, "get deployment error", utils.LogPodResource, utils.GetResource(pod))
		return nil, err
	}
	workload, err := r.getChangeWorkloadByDeployment(ctx, deployment)
	if err != nil {
		logger.Error(err, "get workload error", utils.LogPodResource, utils.GetResource(pod), utils.LogDeploymentResource, utils.GetResource(deployment))
		return nil, err
	}
	return workload, nil
}

// getDeploymentByPod 获取Deployment通过pod
// getDeploymentByPod get Deployment by pod
func (r *ChangeWorkloadReconciler) getDeploymentByPod(ctx context.Context, pod *v1.Pod) (deployment *appsv1.Deployment, err error) {
	logger := log.FromContext(ctx).WithName("getDeploymentByPod")
	// 查询该 Pod 所属的 Deployment
	replicaSet, err := r.getReplicaSetByPod(pod)
	if err != nil {
		logger.Error(err, "getDeploymentByPod get replicaSet error", utils.LogPodResource, utils.GetResource(pod))
		return deployment, err
	}
	deployment, err = r.getDeploymentByReplicaSet(ctx, replicaSet)
	if err != nil {
		logger.Error(err, "getDeploymentByPod get deployment error", utils.LogPodResource, utils.GetResource(pod))
		return deployment, err
	}
	return deployment, nil
}

// getReplicaSetByPod 获取replicaSet通过pod
// getReplicaSetByPod get replicaSet by pod
func (r *ChangeWorkloadReconciler) getReplicaSetByPod(pod *v1.Pod) (replicaSet *appsv1.ReplicaSet, err error) {
	// 查询该 Pod 所属的 ReplicaSet
	replicaSet = &appsv1.ReplicaSet{}
	ownerReferences := pod.OwnerReferences
	for _, ownerReference := range ownerReferences {
		if ownerReference.Kind == native.ReplicaSetKind {
			if err := utils.App.Client.Get(context.Background(), client.ObjectKey{
				Namespace: pod.Namespace,
				Name:      ownerReference.Name,
			}, replicaSet); err != nil {
				return nil, err
			}
			break
		}
	}
	return replicaSet, err
}

// getReplicaSetByPod 获取Deployment通过replicaSet
// getReplicaSetByPod get Deployment by replicaSet
func (r *ChangeWorkloadReconciler) getDeploymentByReplicaSet(ctx context.Context, replicaSet *appsv1.ReplicaSet) (deployment *appsv1.Deployment, err error) {
	logger := log.FromContext(ctx).WithName("getReplicaSetByPod")
	// 查询该 Pod 所属的 ReplicaSet
	deployment = &appsv1.Deployment{}
	ownerReferences := replicaSet.OwnerReferences
	for _, ownerReference := range ownerReferences {
		if ownerReference.Kind == native.DeploymentKind {
			if err := utils.App.Client.Get(context.Background(), client.ObjectKey{
				Namespace: replicaSet.Namespace,
				Name:      ownerReference.Name,
			}, deployment); err != nil {
				logger.Error(err, "get deployment error", utils.LogReplicaSetResource, utils.GetResource(replicaSet))
				return nil, err
			}
			break
		}
	}
	return deployment, err
}

// getChangeWorkloadByPod 通过Deployment获取workload
// getChangeWorkloadByPod get workload by  Deployment
func (r *ChangeWorkloadReconciler) getChangeWorkloadByDeployment(ctx context.Context, deployment *appsv1.Deployment) (workload *v1alpha1.ChangeWorkload, err error) {
	logger := log.FromContext(ctx).WithName("getChangeWorkloadByDeployment")
	workload = &v1alpha1.ChangeWorkload{}
	if err = r.Get(ctx, client.ObjectKey{Name: native.GetChangeWorkloadNameByDeployment(deployment),
		Namespace: deployment.Namespace}, workload); err != nil {
		logger.Error(err, "get workload error", utils.LogDeploymentResource, utils.GetResource(deployment))
	}
	return
}
