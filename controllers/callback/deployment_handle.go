package callback

import (
	"context"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/apis/app.ops.cloud.alipay.com/v1alpha1"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils/native"
)

type SuspendDeployment struct {
	DeploymentName    string        `json:"deploymentName"`
	Namespace         string        `json:"namespace"`
	NowReplicaSet     v1.ReplicaSet `json:"nowReplicaSet"`
	HasSuccessVersion bool          `json:"hasSuccessVersion"`
	SuccessReplicaSet v1.ReplicaSet `json:"successReplicaSet"`
	SuccessVersion    string        `json:"successVersion"`
}

// GetSuspendDeployment GetDeploymentStatus get suspend status of deployment
func GetSuspendDeployment(c *gin.Context) {
	logger := utils.NewLogger().WithName("GetSuspendDeployment")
	namespace := c.Query("namespace")
	logger.Info("GetSuspendDeployment", "namespace", namespace)
	deploymentList := v1.DeploymentList{}

	if err := utils.App.Client.List(c, &deploymentList, client.InNamespace(namespace), client.HasLabels{utils.SuspendLabel}); err != nil {
		logger.Error(err, "GetSuspendDeployment: get deployment list error")
		c.JSON(500, utils.GetCommonCallbackErr(err))
		return
	}
	if utils.IsNotEmpty(deploymentList.Items) {
		suspendDeployments := make([]SuspendDeployment, utils.NumberZero)
		for _, deployment := range deploymentList.Items {
			suspendDeployment, err := getSuspendDeployment(deployment)
			if err != nil {
				logger.Error(err, "GetSuspendDeployment: get suspend deployment error")
				c.JSON(500, utils.GetCommonCallbackErr(err))
				return
			}
			suspendDeployments = append(suspendDeployments, suspendDeployment)
		}
		success := utils.GetCommonCallbackSuccess()
		success["deployments"] = suspendDeployments
		c.JSON(200, success)
		return
	}
	c.JSON(200, utils.GetCommonCallbackSuccess())
}

// DeploymentRollback deployment rollback to previous version
func DeploymentRollback(c *gin.Context) {
	//logger := utils.NewLogger().WithName("DeploymentRollback")

}

func getSuspendDeployment(deployment v1.Deployment) (SuspendDeployment, error) {
	suspendDeployment := SuspendDeployment{
		DeploymentName: deployment.Name,
		Namespace:      deployment.Namespace,
	}
	// 获取当前deployment的replicaset
	// Get the current deployment replica set
	nowReplicaSet, err := getReplicaSet(deployment.Name, deployment.Namespace, deployment.Labels[native.AdmissionWebhookVersionLabel])
	if err != nil {
		return SuspendDeployment{}, err
	}
	suspendDeployment.NowReplicaSet = nowReplicaSet
	// 获取deployment下的所有workload
	// Get all workloads under deployment
	workloadList := v1alpha1.ChangeWorkloadList{}
	if err := utils.App.Client.List(context.Background(), &workloadList, client.InNamespace(deployment.Namespace), client.MatchingLabels{native.DeploymentNameLabel: deployment.Name}); err != nil {
		return SuspendDeployment{}, err
	}
	// 判断workload中是否有成功的版本
	// Determine if there is a successful version in the workload
	if utils.IsNotEmpty(workloadList.Items) {
		for _, workload := range workloadList.Items {
			if workload.Status.Status == v1alpha1.Success {
				suspendDeployment.HasSuccessVersion = true
				suspendDeployment.SuccessVersion = workload.Labels[native.AdmissionWebhookVersionLabel]
				oldReplicaSet, err := getReplicaSet(deployment.Name, deployment.Namespace, workload.Labels[native.AdmissionWebhookVersionLabel])
				if err != nil {
					return SuspendDeployment{}, err
				}
				suspendDeployment.SuccessReplicaSet = oldReplicaSet
				break
			}
		}
	}
	return suspendDeployment, nil
}

func getReplicaSet(deploymentName string, namespace string, version string) (v1.ReplicaSet, error) {
	replicaSetList := v1.ReplicaSetList{}
	if err := utils.App.Client.List(context.Background(), &replicaSetList, client.InNamespace(namespace), client.MatchingLabels{native.AdmissionWebhookVersionLabel: version}); err != nil {
		return v1.ReplicaSet{}, err
	}
	if utils.IsNotEmpty(replicaSetList.Items) {
		for _, replicaSet := range replicaSetList.Items {
			if replicaSet.OwnerReferences[utils.NumberZero].Name == deploymentName {
				return replicaSet, nil
			}
		}
	}
	return v1.ReplicaSet{}, nil
}
