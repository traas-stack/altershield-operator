package utils

import (
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