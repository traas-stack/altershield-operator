package types

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ChangeCheckVerdict struct {
	Passed bool `json:"passed"`
	Msg     string `json:"msg"`
}

type DefenseExecutionBrief struct {
	// Generation is the expected spec generation of defense target object
	Generation int64 `json:"generation"`
	// DefenseExecID is the execution ID of change defense in altershield
	DefenseExecID     string `json:"defenseExecId"`
	ChangeDefenseName string `json:"changeDefenseName"`
}

func (d *DefenseExecutionBrief) BuildChangeDefenseExecutionName() string {
	return d.ChangeDefenseName + "-" + d.DefenseExecID
}

type ChangeDefenseExecution struct {
	ID string `json:"id"`
}

type WorkloadInfo struct {
	GVK schema.GroupVersionKind
	Obj client.Object
	Replicas int32
}
