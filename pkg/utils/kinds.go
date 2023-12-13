package utils

import (
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	DeploymentKind  = apps.SchemeGroupVersion.WithKind("Deployment")
	StatefulSetKind = apps.SchemeGroupVersion.WithKind("StatefulSet")

	ReplicaSetKind = apps.SchemeGroupVersion.WithKind("ReplicaSet")

	PodKind = corev1.SchemeGroupVersion.WithKind("Pod")
)

func IsSupportedKind(kind schema.GroupVersionKind) bool {
	return (DeploymentKind.Group == kind.Group && DeploymentKind.Kind == kind.Kind) ||
		(StatefulSetKind.Group == kind.Group && StatefulSetKind.Kind == kind.Kind) ||
		(ReplicaSetKind.Group == kind.Group && ReplicaSetKind.Kind == kind.Kind)
}