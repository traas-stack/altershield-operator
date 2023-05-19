package native

import (
	appsv1 "k8s.io/api/apps/v1"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
)

const (
	// AdmissionWebhookVersionLabel version label for admission webhook
	AdmissionWebhookVersionLabel = "admission-webhook-altershield.antgroup.com/version"

	DeploymentNameLabel = "app.kubernetes.io/deployment-name"

	// AdmissionWebhookNamespaceLabel namespace label for admission webhook
	AdmissionWebhookNamespaceLabel = "admission-webhook-altershield"
)

const (
	DeploymentKind = "Deployment"
	ReplicaSetKind = "ReplicaSet"
)

const (
	PodStatusRunning     = "Running"
	PodStatusTerminating = "Terminating"
)

func GetChangeWorkloadNameByDeployment(deployment *appsv1.Deployment) string {
	return utils.CombineString(deployment.Name, deployment.Labels[AdmissionWebhookVersionLabel])
}
