package callback

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/apis/app.ops.cloud.alipay.com/v1alpha1"
	opscloudclient "gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/client"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
)

func CheckCallBackHandler(c *gin.Context) {
	logger := utils.NewLogger().WithName("CheckCallBackHandler")
	var data opscloudclient.OpsCloudChangeCheckCallbackWrapperRequest
	if err := c.ShouldBindJSON(&data); err != nil {
		logger.Error(err, "CheckCallBackHandler:bind json error")
		c.JSON(http.StatusBadRequest, utils.GetCommonCallbackErr(err))
		return
	}
	// 前置后置复用一个接口
	if data.ChangeCheckType != opscloudclient.OpsCloudChangeCheckTypeEnumBatch {
		return
	}
	logger.Info("CheckCallBackHandler:callback data", "data", data)
	// 获取node信息
	// get node info
	changePod := v1alpha1.ChangePod{}
	changePodList := v1alpha1.ChangePodList{}
	nodeId := data.CallbackRequest.NodeId
	orderId := data.CallbackRequest.BizExecOrderId
	// TODO 返回只有NODE ID
	logger.Info("CheckCallBackHandler:get node info", "nodeId", nodeId, "orderId", orderId)
	if err := utils.App.Client.List(context.Background(), &changePodList,
		client.MatchingFields{utils.ChangePodFieldChangePodId: nodeId}); err != nil {
		logger.Error(err, "CheckCallBackHandler:get node error")
		return
	}
	if utils.IsNotEmpty(changePodList.Items) {
		changePod = changePodList.Items[utils.NumberZero]
	} else {
		logger.Info("CheckCallBackHandler:node not found")
		c.JSON(http.StatusOK, utils.GetCommonCallbackSuccess())
	}
	if data.CallbackRequest.DefenseStageEnum == opscloudclient.DefenseStageEnumPre {
		// 如果当前状态不是PRE_AOP，则不做任何操作
		// if node status is not PRE_AOP, do nothing
		if changePod.Status.Status == v1alpha1.PreSubmitted {
			// 使用patch的方式更新node的status
			// patch node status
			patch := client.MergeFrom(changePod.DeepCopy())
			changePod.Status.Status = v1alpha1.PostWait
			changePod.Status.UpdateTime = utils.GetNowTime()
			changePod.Status.UpdateTimeUnix = time.Now().Unix()
			if err := utils.App.Client.Status().Patch(context.Background(), &changePod, patch); err != nil {
				logger.Error(err, "DefenseStageEnumPre:patch node status error")
				c.JSON(http.StatusInternalServerError, utils.GetCommonCallbackErr(err))
				return
			}

		}
		c.JSON(http.StatusOK, utils.GetCommonCallbackSuccess())
	} else if data.CallbackRequest.DefenseStageEnum == opscloudclient.DefenseStageEnumPost {
		// 如果当前状态不是POST_AOP状态，则不做任何操作
		// if node status is not POST_AOP, do nothing
		if changePod.Status.Status == v1alpha1.PostSubmitted {
			// 使用patch的方式更新node的status
			// patch node status
			patch := client.MergeFrom(changePod.DeepCopy())
			changePod.Status.Status = v1alpha1.PostFinish
			// TODO 根据返回结果 更新status
			changePod.Status.PodResults = changePod.Spec.PodInfos
			for i := range changePod.Status.PodResults {
				changePod.Status.PodResults[i].Verdict = data.CallbackRequest.Verdict.Verdict
				changePod.Status.PodResults[i].Message = data.CallbackRequest.Verdict.Msg
			}
			changePod.Status.UpdateTime = utils.GetNowTime()
			changePod.Status.UpdateTimeUnix = time.Now().Unix()
			if err := utils.App.Client.Status().Patch(context.Background(), &changePod, patch); err != nil {
				logger.Error(err, "DefenseStageEnumPost:patch node status error")
				c.JSON(http.StatusInternalServerError, utils.GetCommonCallbackErr(err))
				return
			}
		} else {
			logger.Info("DefenseStageEnumPost:node status is not POST_AOP")
			c.JSON(http.StatusOK, utils.GetCommonCallbackSuccess())
			return
		}
	}
}

func LiveTest(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Hello world"})
}
