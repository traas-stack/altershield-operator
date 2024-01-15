package client

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/apis/app.ops.cloud.alipay.com/v1alpha1"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
)

func TestG2(t *testing.T) {
	var bizExeOrderId = fmt.Sprintf("testOrder%d", time.Now().Unix())
	request := OpsCloudChangeExecOrderSubmitRequest{
		BizExecOrderId:     bizExeOrderId,
		Platform:           Platform,
		ChangeSceneKey:     utils.ChangeSceneKeyRollingUpdate,
		ChangeApps:         []string{"app1"},
		ChangeParamJson:    "{}",
		ChangePhases:       []string{utils.ChangePhase},
		ChangeScenarioCode: utils.ChangeScenarioCode,
		ChangeTitle:        "小程序云变更测试",
		ChangeUrl:          "http://xxx.xxx",
		Creator:            utils.DefaultCreator,
		ChangeContents:     DefaultChangeContents,
		TldcTenantCode:     utils.DefaultTldcTenantCode,
	}
	result, err := SubmitChangeExecOrder(request)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(result)

	var podInfos []string
	podInfo := v1alpha1.PodSummary{
		App:       "topscloud",
		Hostname:  "topscloud-60-ea133-0",
		Workspace: "default",
		Pod:       "topscloudprodgzcr60a0-pbl4b-74hk6",
		Ip:        "10.2.31.121",
		Namespace: "topscloud",
	}
	marshal, err := json.Marshal(podInfo)
	if err != nil {
		fmt.Println(err.Error())
	}
	podInfos = append(podInfos, string(marshal))

	// submit pre check
	request1 := OpsCloudChangeExecBatchStartNotifyRequest{
		ChangePhase:              utils.ChangePhase,
		Executor:                 utils.DefaultCreator,
		EffectiveTargetType:      "pass.pod",
		EffectiveTargetLocations: podInfos,
		Platform:                 Platform,
		ChangeSceneKey:           utils.ChangeSceneKeyRollingUpdate,
		TldcTenantCode:           utils.DefaultTldcTenantCode,
		BizExecOrderId:           bizExeOrderId,
	}
	notify, err := SubmitChangeStartNotify(request1)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(notify)

	if notify.Domain == nil {
		// 空指针处理，可以返回错误信息或者使用默认值
		fmt.Println(fmt.Errorf("notify.Domain is nil"))
	} else {
		m := notify.Domain.(map[string]interface{})
		// 接下来的操作
		// submit post check
		request2 := OpsCloudChangeFinishNotifyRequest{
			NodeId:         m["nodeId"].(string),
			Success:        true,
			ServiceResult:  "{}",
			Platform:       Platform,
			ChangeSceneKey: utils.ChangeSceneKeyRollingUpdate,
			BizExecOrderId: bizExeOrderId,
			TldcTenantCode: utils.DefaultTldcTenantCode,
		}
		finishNotify, err := SubmitChangeFinishNotify(request2)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println(finishNotify)
	}

}
