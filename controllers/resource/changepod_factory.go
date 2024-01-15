package resource

import (
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/apis/app.ops.cloud.alipay.com/v1alpha1"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils/native"
)

type IChangePodFactory interface {
	IFactory
}

type NativeChangePodFactory struct {
	ChangeWorkload *v1alpha1.ChangeWorkload
	ChangePodNum   int
}

func (factory *NativeChangePodFactory) NewInstance() runtime.Object {
	changePod := v1alpha1.ChangePod{}
	changePod.Name = factory.ChangeWorkload.Name + utils.MetaMark + strconv.Itoa(factory.ChangePodNum+utils.NumberOne)
	changePod.Namespace = factory.ChangeWorkload.Namespace
	changePod.Spec.PodInfos = make([]v1alpha1.PodSummary, utils.NumberZero)
	changePod.Spec.ChangeWorkloadId = factory.ChangeWorkload.Name
	changePod.Spec.CreateTime = utils.GetNowTime()
	changePod.Spec.CreateTimeUnix = time.Now().Unix()
	changePod.Labels = make(map[string]string)
	changePod.Labels[native.DeploymentNameLabel] = factory.ChangeWorkload.Labels[native.DeploymentNameLabel]
	changePod.Labels[native.AdmissionWebhookVersionLabel] = factory.ChangeWorkload.Labels[native.AdmissionWebhookVersionLabel]
	return &changePod
}
