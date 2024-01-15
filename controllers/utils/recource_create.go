package utils

import (
	"strconv"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/apis/app.ops.cloud.alipay.com/v1alpha1"
)

func NewOpsConfigInfoBatchFunc() *v1alpha1.OpsConfigInfo {
	opsConfigInfo := v1alpha1.OpsConfigInfo{}
	opsConfigInfo.Name = ConfigNameIsBranch
	opsConfigInfo.Namespace = AlterShieldOperatorNamespace
	opsConfigInfo.Spec.Type = ConfigTypeIsBranch
	opsConfigInfo.Spec.Content = strconv.Itoa(NumberTen)
	opsConfigInfo.Spec.Remark = "Enabling Batch Protection"
	opsConfigInfo.Spec.Enable = false
	return &opsConfigInfo
}

func NewOpsConfigInfoBlockFunc() *v1alpha1.OpsConfigInfo {
	opsConfigInfo := v1alpha1.OpsConfigInfo{}
	opsConfigInfo.Name = ConfigNameIsBlockingUp
	opsConfigInfo.Namespace = AlterShieldOperatorNamespace
	opsConfigInfo.Spec.Type = ConfigTypeIsBlockingUp
	opsConfigInfo.Spec.Remark = "Enabling Blockade"
	opsConfigInfo.Spec.Enable = true
	return &opsConfigInfo
}
