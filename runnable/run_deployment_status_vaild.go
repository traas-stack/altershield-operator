package runnable

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils/native"
)

var logger = utils.NewLogger().WithName("DeploymentStatusRollBackRun")
var timeout = 2 * time.Minute

func DeploymentStatusRollBackRun() {
	go func() {
		for {
			time.Sleep(1 * time.Second)
			// 查询所有被 webhook 管控的 Namespace
			// Query all namespaces controlled by the webhook
			allWebhookNamespace := getAllWebhookNamespace()
			for _, ns := range allWebhookNamespace.Items {
				// 查询状态不正常超过 1 分钟的 Pod
				// Query pods with abnormal status for more than 1 minute
				pods := getNonRunningPods(ns.Name)
				for _, pod := range pods {
					logger.Info("Found non-running pod", utils.LogPodResource, utils.GetResource(&pod))
					// 查询该 Pod 所属的 ReplicaSet
					// Query the ReplicaSet to which the pod belongs
					ownsReplicaSet := getPodOwnsReplicaSet(&pod)

					if ownsReplicaSet.Name != "" {
						// 查询该 ReplicaSet 所属的 Deployment
						// Query the Deployment to which the ReplicaSet belongs
						ownsDeployment := getReplicaSetOwnsDeployment(ownsReplicaSet)
						if ownsDeployment.Name != "" {
							// 查询该 Deployment 状态是否正常
							if checkDeploymentStatus(ownsDeployment) {
								return
							} else {
								logger.Info("Found non-running deployment", utils.LogDeploymentResource, utils.GetResource(ownsDeployment))

								normalReplicaSet := getNormalReplicaSetByDeployment(ownsDeployment, ownsReplicaSet)
								if normalReplicaSet.Name == "" {
									logger.Info("Not found non-running deployment's running replicaSet", utils.LogDeploymentResource, utils.GetResource(ownsDeployment))
									return
								}
								logger.Info("Found non-running deployment's running replicaSet", utils.LogReplicaSetResource, utils.GetResource(&normalReplicaSet))
								// 将这个replicaset的template的spec赋值给deployment的spec
								delete(normalReplicaSet.Spec.Template.Labels, "pod-template-hash")
								ownsDeployment.Spec.Template = normalReplicaSet.Spec.Template
								// 更新deployment
								if err := utils.App.Client.Update(context.Background(), ownsDeployment); err != nil {
									logger.Error(err, "update deployment error")
								}
								logger.Info("update deployment success", utils.LogDeploymentResource, utils.GetResource(ownsDeployment))
							}
						}
					}
				}
			}
		}
	}()
}

// getNonRunningPods 获取所有状态不正常的 Pod，相同的 ReplicaSet 只会返回一个
// getNonRunningPods get all pods with abnormal status, and only one ReplicaSet will be returned
func getNonRunningPods(ns string) []corev1.Pod {
	podList := &corev1.PodList{}
	if err := utils.App.Client.List(context.Background(), podList, client.InNamespace(ns)); err != nil {
		return nil
	}
	// 创建一个map，key为pod的owns，value为pod
	// Create a map, the key is the owns of the pod, and the value is the pod
	mapPods := make(map[string]corev1.Pod)
	for _, pod := range podList.Items {
		// 如果pod的label中不包含AdmissionWebhookVersionLabel，说明不是webhook管控的pod，不需要处理
		// if the label of the pod does not contain AdmissionWebhookVersionLabel, it means that it is not a pod controlled by the webhook and does not need to be processed
		if _, ok := pod.Labels[native.AdmissionWebhookVersionLabel]; !ok {
			continue
		}
		// 如果 Pod 的状态不是 Running，并且 Pod 的启动时间在指定时间之前
		// If the status of the Pod is not Running, and the startup time of the Pod is before the specified time
		if pod.Status.Phase != native.PodStatusRunning && pod.Status.Phase != native.PodStatusTerminating && pod.Status.StartTime.Before(&metav1.Time{Time: time.Now().Add(-timeout)}) {
			ownerReferences := pod.OwnerReferences
			for _, ownerReference := range ownerReferences {
				if ownerReference.Kind == native.ReplicaSetKind {
					mapPods[ownerReference.Name] = pod
				}
			}
		}
	}
	var pods []corev1.Pod
	// 此时mapPods中的value就是状态不正常的pod，并且owns都是不同的
	// At this time, the value in mapPods is the pod with abnormal status, and the owns are different
	for _, value := range mapPods {
		pods = append(pods, value)
	}
	return pods
}

// getNonRunningDeployments 获取所有被webhook管控的namespace
// getNonRunningDeployments get all non-running deployments
func getAllWebhookNamespace() *corev1.NamespaceList {
	// 查询所有 Namespace
	namespaceList := &corev1.NamespaceList{}
	if err := utils.App.Client.List(context.Background(), namespaceList, client.MatchingLabels{
		native.AdmissionWebhookNamespaceLabel: utils.Enabled,
	}); err != nil {
		logger.Error(err, "get namespace list error")
	}
	return namespaceList
}

// getPodOwnsReplicaSet 获取 Pod 所属的 ReplicaSet
// getPodOwnsReplicaSet get the ReplicaSet that the Pod belongs to
func getPodOwnsReplicaSet(pod *corev1.Pod) *appsv1.ReplicaSet {
	replicaSet := &appsv1.ReplicaSet{}
	ownerReferences := pod.OwnerReferences
	for _, ownerReference := range ownerReferences {
		if ownerReference.Kind == native.ReplicaSetKind {
			if err := utils.App.Client.Get(context.Background(), client.ObjectKey{
				Namespace: pod.Namespace,
				Name:      ownerReference.Name,
			}, replicaSet); err != nil {
				logger.Error(err, "get replicaSet error")
			}
			break
		}
	}
	return replicaSet
}

// getReplicaSetOwnsDeployment 获取 ReplicaSet 所属的 Deployment
// getReplicaSetOwnsDeployment get the Deployment that the ReplicaSet belongs to
func getReplicaSetOwnsDeployment(replicaSet *appsv1.ReplicaSet) *appsv1.Deployment {
	deployment := &appsv1.Deployment{}
	ownerReferences := replicaSet.OwnerReferences
	for _, ownerReference := range ownerReferences {
		if ownerReference.Kind == native.DeploymentKind {
			if err := utils.App.Client.Get(context.Background(), client.ObjectKey{
				Namespace: replicaSet.Namespace,
				Name:      ownerReference.Name,
			}, deployment); err != nil {
				logger.Error(err, "get deployment error")
			}
			break
		}
	}
	return deployment
}

// checkDeploymentStatus 检查 Deployment 的状态是否正常
// checkDeploymentStatus check if the status of the Deployment is normal
func checkDeploymentStatus(deployment *appsv1.Deployment) bool {
	return deployment.Status.Replicas == deployment.Status.ReadyReplicas &&
		deployment.Status.ReadyReplicas == deployment.Status.AvailableReplicas &&
		deployment.Status.AvailableReplicas == deployment.Status.UpdatedReplicas
}

// getNormalReplicaSetByDeployment 获取 Deployment 的正常 ReplicaSet
// getNormalReplicaSetByDeployment get the normal ReplicaSet of Deployment
func getNormalReplicaSetByDeployment(deployment *appsv1.Deployment, ownsReplicaSet *appsv1.ReplicaSet) appsv1.ReplicaSet {
	replicaSets := &appsv1.ReplicaSetList{}
	normalReplicaSet := appsv1.ReplicaSet{}
	if err := utils.App.Client.List(context.Background(), replicaSets, client.InNamespace(deployment.Namespace), client.MatchingLabels(deployment.Spec.Selector.MatchLabels)); err != nil {
		logger.Error(err, "get non-running deployment's replicaSets error")
	}
	if len(replicaSets.Items) <= 1 {
		return normalReplicaSet
	}
	for _, replicaSet := range replicaSets.Items {
		if replicaSet.Name != ownsReplicaSet.Name && replicaSet.Status.Replicas != utils.NumberZero && replicaSet.Status.Replicas == replicaSet.Status.ReadyReplicas &&
			replicaSet.Status.ReadyReplicas == replicaSet.Status.AvailableReplicas {
			normalReplicaSet = replicaSet
			break
		}
	}
	return normalReplicaSet
}
