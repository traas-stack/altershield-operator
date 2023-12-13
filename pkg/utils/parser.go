package utils

import "sigs.k8s.io/controller-runtime/pkg/client"

func GetAnnotation(obj client.Object, key string) string {
	if obj == nil {
		return ""
	}
	if len(obj.GetAnnotations()) == 0 {
		return ""
	}
	return obj.GetAnnotations()[key]
}

func GetLabel(obj client.Object, key string) string {
	if obj == nil {
		return ""
	}
	if len(obj.GetLabels()) == 0 {
		return ""
	}
	return obj.GetLabels()[key]
}
