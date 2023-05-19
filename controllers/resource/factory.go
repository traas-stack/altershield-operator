package resource

import (
	"k8s.io/apimachinery/pkg/runtime"
)

type IFactory interface {
	NewInstance() runtime.Object
}
