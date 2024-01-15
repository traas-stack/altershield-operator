package client

import (
	"github.com/traas-stack/altershield-operator/pkg/types"
	"strconv"
)

const (
	DefaultPlatform           = "kubenetes"
	DefaultChangeScene        = "com.alipay.alitershield.kubenetes.rollingupdate"
	DefaultChangePhase        = "prod_phase"
	DefaultChangeScenarioCode = "XX"
	DefaultChangeURL          = "http://xxx.xxx"
	DefaultCreator            = "operator"
	DefaultExecutor           = "system"
	DefaultTenant             = "default"
	DefaultIdentity			  = "altershield-operator"
)

type ChangeTargetType string

var (
	ChangeTargetTypePod ChangeTargetType = "pass.pod"
)

type AltershieldResponse struct {
	Success    bool        `json:"success"`
	ResultCode string      `json:"resultCode"`
	Msg        string      `json:"msg"`
	Domain     map[string]interface{} `json:"domain"`
}

// AltershieldChangeExecOrderSubmitRequest submit change execute order request
type SubmitChangeExecOrderRequest struct {
	BizExecOrderID     string          `json:"bizExecOrderId"`
	Platform           string          `json:"platform"`
	ChangeSceneKey     string          `json:"changeSceneKey"`
	ChangeApps         []string        `json:"changeApps"`
	ChangeParamJSON    string          `json:"changeParamJson"`
	ChangePhases       []string        `json:"changePhases"`
	ChangeScenarioCode string          `json:"changeScenarioCode"`
	ChangeTitle        string          `json:"changeTitle"`
	ChangeURL          string          `json:"changeUrl"`
	Creator            string          `json:"creator"`
	ChangeContents     []ChangeContent `json:"changeContents"`
	TenantCode 		   string        `json:"tenantCode"`
}

func NewSubmitChangeExecOrderRequest(executionID, appName string) *SubmitChangeExecOrderRequest {
	return &SubmitChangeExecOrderRequest{
		BizExecOrderID: executionID,
		Platform:       DefaultPlatform,
		ChangeSceneKey: DefaultChangeScene,
		ChangeApps:         []string{
			appName,
		},
		ChangeParamJSON:    "{}",
		ChangePhases:       []string{
			DefaultChangePhase,
		},
		ChangeScenarioCode: DefaultChangeScenarioCode,
		ChangeTitle:        appName + "-" + executionID,
		ChangeURL:          DefaultChangeURL,
		Creator:            DefaultCreator,
		ChangeContents:     []ChangeContent{
			{
				ContentType: ChangeContentType{
					TypeName: string(ChangeTargetTypePod),
				},
				InstanceName: "",
			},
		},
		TenantCode: DefaultTenant,
	}
}

func NewSubmitChangeExecOrderStartNotifyRequest(executionID string) *SubmitChangeExecOrderStartNotifyRequest {
	return &SubmitChangeExecOrderStartNotifyRequest{
		Executor:       DefaultExecutor,
		TenantCode:     DefaultTenant,
		Platform:       DefaultPlatform,
		ChangeSceneKey: DefaultChangeScene,
		BizExecOrderID: executionID,
	}
}

type ChangeContent struct {
	ContentType  ChangeContentType `json:"contentType"`
	InstanceName string            `json:"instanceName"`
}

type ChangeContentType struct {
	TypeName string `json:"typeName"`
}

type SubmitChangeExecOrderResponse struct {
	OrderID string `json:"orderId"`
	URL string `json:"url"`
}

type SubmitChangeExecOrderStartNotifyRequest struct {
	Executor string `json:"executor"`
	TenantCode string `json:"tenantCode"`
	Platform string `json:"platform"`
	ChangeSceneKey string `json:"changeSceneKey"`
	BizExecOrderID string `json:"bizExecOrderId"`
}

type SubmitChangeExecOrderStartNotifyResponse struct {
	NodeID string `json:"nodeId"`
	SkipCheck bool `json:"skipCheck"`
	URL string `json:"url"`
}

type SubmitChangeExecBatchStartNotifyRequest struct {
	ChangePhase string `json:"changePhase"`
	IsLastBatchInChangePhaseTag string "isLastBatchInChangePhaseTag"
	TotalBatchNumInChangePhase int `json:"totalBatchNumInChangePhase"`
	BatchNo int `json:"batchNo"`
	Executor string `json:"executor"`
	TotalBatchNum string `json:"totalBatchNum"`
	EffectiveTargetType string `json:"effectiveTargetType"`
	EffectiveTargetLocations []string `json:"effectiveTargetLocations"`
	ExtInfo map[string]string `json:"extInfo"`
	BizExecOrderID string `json:"bizExecOrderId"`
	Platform string `json:"platform"`
	ChangeSceneKey string `json:"changeSceneKey"`
	TenantCode string `json:"tenantCode"`
}

type SubmitChangeExecBatchStartNotifyResponse struct {
	NodeID string `json:"nodeId"`
	SkipCheck bool `json:"skipCheck"`
	URL string `json:"url"`
}

func NewSubmitChangeExecBatchStartNotifyRequest(executionID string, batch, totalBatches int, info *types.WorkloadInfo) *SubmitChangeExecBatchStartNotifyRequest {
	return &SubmitChangeExecBatchStartNotifyRequest{
		ChangePhase:                 DefaultChangePhase,
		IsLastBatchInChangePhaseTag: "not_last",
		TotalBatchNumInChangePhase:  1,
		BatchNo:                     batch,
		Executor:                    DefaultExecutor,
		TotalBatchNum:               strconv.Itoa(totalBatches),
		EffectiveTargetType:         string(ChangeTargetTypePod),
		EffectiveTargetLocations:    []string{},
		ExtInfo:                     map[string]string{
			"namespace": info.Obj.GetNamespace(),
			"workloadName": info.Obj.GetName(),
			"workloadType": info.GVK.Kind,
		},
		BizExecOrderID:              executionID,
		Platform:                    DefaultPlatform,
		ChangeSceneKey:              DefaultChangeScene,
		TenantCode:                  DefaultTenant,
	} 
}

type SubmitChangeFinishNotifyRequest struct {
	NodeID string `json:"nodeId"`
	Success bool `json:"success"`
	ServiceResult string `json:"serviceResult"`
	BizExecOrderID string `json:"bizExecOrderId"`
	Platform string `json:"platform"`
	ChangeSceneKey string `json:"changeSceneKey"`
	TenantCode string `json:"tenantCode"`
}

type SubmitChangeFinishNotifyResponse struct {
	NodeID string `json:"nodeId"`
	SkipCheck bool `json:"skipCheck"`
	URL string `json:"url"`
}

func NewSubmitChangeFinishNotifyRequest(executionID string, nodeID string) *SubmitChangeFinishNotifyRequest {
	return &SubmitChangeFinishNotifyRequest{
		NodeID:         nodeID,
		Success:        true,
		ServiceResult:  "{}",
		BizExecOrderID: executionID,
		Platform:       DefaultPlatform,
		ChangeSceneKey: DefaultChangeScene,
		TenantCode:     DefaultTenant,
	}
}

type ChangeStartCheckRequest struct {

}

type ChangeStartCheckResponse struct {

}

type SubmitChangeBatchPostCheckRequest struct {

}

type SubmitChangeBatchPostCheckResponse struct {

}

type QueryChangeBatchPostCheckRequest struct {

}

type QueryChangeBatchPostCheckResponse struct {

}

type ChangeBatchPreCheckRequest struct {

}

type ChangeBatchPreCheckResponse struct {

}

