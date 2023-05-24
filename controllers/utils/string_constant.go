package utils

const (
	True                      = "true"
	False                     = "false"
	Enabled                   = "enabled"
	MetaMark                  = "--x--"
	Namespace                 = "namespace"
	ResourceTypePod           = "[**pod**]"
	ResourceTypeChangePod     = "[**changepod**]"
	LogPodResource            = "pod resource"
	LogDeploymentResource     = "deployment resource"
	LogReplicaSetResource     = "replicaSet resource"
	LogChangeWorkloadResource = "change workload resource"
	LogChangePodResource      = "change pod resource"
)

const (
	// AlterShieldOperatorNamespace 主命名空间
	AlterShieldOperatorNamespace = "altershieldoperator-system"

	// DefenseStatusLabel 变更后置标签-防控状态标签

	ChangePodVerdictPass        = "pass"
	DefenseStatusLabelProcessed = "processed"

	ConfigTypeIsBranch     = "isBranch"
	ConfigNameIsBranch     = "branch"
	ConfigTypeIsBlockingUp = "isBlockingUp"
	ConfigNameIsBlockingUp = "blocking"

	ChangePodFieldStatus           = "changePod.status"
	ChangePodFieldChangePodId      = "changePod.changePodId"
	ChangePodFieldChangeWorkloadId = "changePod.changeWorkloadId"
	ChangeWorkloadFieldStatus      = "changeWorkload.status"

	// StringRecordSpecEffectiveTargetType 管控端字段
	StringRecordSpecEffectiveTargetType = "paas.pod"

	// ChangeScenarioCode 变更管控交互
	ChangeScenarioCode          = "XX"         // default change scenario
	ChangePhase                 = "prod_phase" // default change phase is prod
	ChangeSceneKeyRollingUpdate = "com.alipay.cafed.cloudrun.rollingupdate"
	// DefaultTldcTenantCode 租户不能为空
	DefaultTldcTenantCode = "MAYITNSC"
	DefaultCreator        = "altershieldoperator"
)

// label
const (
	OperateFinishedLabel = "altershield.defense.antgroup.com/operate-finished"
	DefenseStatusLabel   = "altershield.defense.antgroup.com/defense-status"
	SuspendLabel         = "altershield.defense.antgroup.com/suspend"
	IgnoredSuspendLabel  = "altershield.defense.antgroup.com/ignored-suspend"
)

// webhook
const (
	ContentTypeHeader = "Content-Type"
	ContentTypeJSON   = "application/json"
)
