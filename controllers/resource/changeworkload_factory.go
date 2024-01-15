package resource

import (
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/apis/app.ops.cloud.alipay.com/v1alpha1"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils/native"
)

type IChangeWorkloadFactory interface {
	IFactory
}

type NativeChangeWorkloadFactory struct {
	Deployment *appsv1.Deployment
	Replicas   *int32
}

func (factory *NativeChangeWorkloadFactory) NewInstance() runtime.Object {
	workload := v1alpha1.ChangeWorkload{}
	workload.Name = native.GetChangeWorkloadNameByDeployment(factory.Deployment)
	workload.Namespace = factory.Deployment.Namespace
	workload.Spec.ChangeWorkloadId = workload.Name
	workload.Spec.ServiceName = factory.Deployment.Name
	workload.Spec.Reversion = factory.Deployment.Labels[native.AdmissionWebhookVersionLabel]
	if utils.ConfigIsBatch() {
		replicas := *factory.Replicas
		workload.Spec.CountThreshold = utils.Percent(int(replicas), utils.ConfigBatchCount())
	} else {
		workload.Spec.CountThreshold = utils.NumberOne
	}
	workload.Spec.WaitTimeThreshold = utils.ChangeWorkloadWaitTimeThreshold
	workload.Spec.CreateTime = utils.GetNowTime()
	workload.Spec.CreateTimeUnix = time.Now().Unix()
	workload.Spec.AppName = factory.Deployment.Name
	workload.Labels = make(map[string]string)
	workload.Labels[native.DeploymentNameLabel] = factory.Deployment.Name
	workload.Labels[native.AdmissionWebhookVersionLabel] = factory.Deployment.Labels[native.AdmissionWebhookVersionLabel]
	return &workload
}
