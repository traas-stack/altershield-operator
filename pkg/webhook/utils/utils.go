package utils

import (
	"github.com/traas-stack/altershield-operator/apis/app.ops.cloud.alipay.com/v1alpha1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func AdmissionErroredWithLog(code int32, err error) admission.Response {
	klog.Errorf( "admission error: %v", err)
	return admission.Errored(code, err)
}

func AdmissionDeniedWithLog(reason string) admission.Response {
	klog.Errorf("admission denied: %v", reason)
	return admission.Denied(reason)
}

func InDefenseProgress(phase v1alpha1.DefensePhase) bool {
	switch phase {
	case v1alpha1.DefensePhaseInitial:
		fallthrough
	case v1alpha1.DefensePhasePreCheck:
		fallthrough
	case v1alpha1.DefensePhaseObserving:
		fallthrough
	case v1alpha1.DefensePhasePostCheck:
		fallthrough
	case v1alpha1.DefensePhaseFailed:
		return true
	default:
		return false
	}
}