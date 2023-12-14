package routers

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/traas-stack/altershield-operator/apis/app.ops.cloud.alipay.com/v1alpha1"
	altershield "github.com/traas-stack/altershield-operator/pkg/altershield/client"
	utils2 "github.com/traas-stack/altershield-operator/pkg/utils"
	"k8s.io/klog/v2"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CallbackHandler struct {
	client.Client
}

func (handler *CallbackHandler) CheckCallBackHandler(c *gin.Context) {
	request := &altershield.ChangeCheckCallbackWrapperRequest{}
	if err := c.ShouldBindJSON(request); err != nil {
		klog.Errorf("failed to unmarshal check callback request: %v", err)
		c.JSON(http.StatusBadRequest, GetCommonCallbackErr(err))
		return
	}
	klog.Infof("get check callback request: %v", func () string {
		raw, _ := json.Marshal(request)
		return string(raw)
	}())

	orderID := request.CallbackRequest.BizExecOrderID
	changeDefense, err := utils2.GetChangeDefenseByExecutionID(handler, orderID)
	if err != nil {
		klog.Errorf("failed to get change defense by exec id in callback: %v", err)

		c.JSON(http.StatusBadRequest, GetCommonCallbackErr(err))
		return
	}
	if changeDefense == nil {
		c.JSON(http.StatusOK, GetCommonCallbackSuccess())
		return
	}
	callbackContent := &request.CallbackRequest
	switch request.ChangeCheckType {
	case altershield.ChangeCheckTypeOrder:
		newStatus := changeDefense.Status.DeepCopy()
		if altershield.DefenseStageTypePre == callbackContent.DefenseStageEnum {
			switch callbackContent.Verdict.Verdict {
			case altershield.DefenseVerdictFail:
				newStatus.Phase = v1alpha1.DefensePhaseFailed
			case altershield.DefenseVerdictPass:
				newStatus.Phase = v1alpha1.DefensePhaseProgressing
			}
		}
		if err = utils2.UpdateChangeDefenseStatus(handler.Client, context.TODO(), changeDefense, newStatus); err != nil {
			klog.Errorf("failed to update ChangeDefenseStatus in callback: %v", err)
			c.JSON(http.StatusBadRequest, GetCommonCallbackErr(err))
			return
		}
	case altershield.ChangeCheckTypeBatch:
		changeDefenseExecution, err := utils2.GetChangeDefenseExecutionByID(handler, orderID)
		if err != nil {
			c.JSON(http.StatusBadRequest, GetCommonCallbackErr(err))
			return
		}
		if changeDefenseExecution == nil {
			c.JSON(http.StatusOK, GetCommonCallbackSuccess())
			return
		}

		nodeID := request.CallbackRequest.NodeID
		newStatus := changeDefenseExecution.Status.DeepCopy()
		if nodeID != newStatus.DefenseStatus.NodeID {
			klog.Infof("callback node id %v not identical to status %v, ignored",
				nodeID, newStatus.DefenseStatus.NodeID)
			c.JSON(http.StatusOK, GetCommonCallbackSuccess())
			return
		}
		if (altershield.DefenseStageTypePre == callbackContent.DefenseStageEnum) &&
			(newStatus.DefenseStatus.Phase == v1alpha1.DefensePhasePreCheck) {
			switch callbackContent.Verdict.Verdict {
			case altershield.DefenseVerdictPass:
				newStatus.DefenseStatus.Phase = v1alpha1.DefensePhaseProgressing
				newStatus.DefenseStatus.Verdict = callbackContent.Verdict.Msg
			case altershield.DefenseVerdictFail:
				newStatus.DefenseStatus.Phase = v1alpha1.DefensePhaseFailed
				newStatus.DefenseStatus.Verdict = callbackContent.Verdict.Msg
			}
		} else if (altershield.DefenseStageTypePost == callbackContent.DefenseStageEnum) &&
			(newStatus.DefenseStatus.Phase == v1alpha1.DefensePhasePostCheck) {
			switch callbackContent.Verdict.Verdict {
			case altershield.DefenseVerdictPass:
				newStatus.DefenseStatus.Phase = v1alpha1.DefensePhasePassed
				newStatus.DefenseStatus.Verdict = callbackContent.Verdict.Msg
			case altershield.DefenseVerdictFail:
				newStatus.DefenseStatus.Phase = v1alpha1.DefensePhaseFailed
				newStatus.DefenseStatus.Verdict = callbackContent.Verdict.Msg
			}
		}
		if err = utils2.UpdateChangeDefenseExecutionStatus(
			handler.Client, context.TODO(), changeDefenseExecution, newStatus); err != nil {
			klog.Errorf("failed to update ChangeDefenseExecutionStatus in callback: %v", err)
			c.JSON(http.StatusBadRequest, GetCommonCallbackErr(err))
			return
		}
	}
	c.JSON(http.StatusOK, GetCommonCallbackSuccess())
}
