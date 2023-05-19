package runnable

import (
	"context"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/apis/app.ops.cloud.alipay.com/v1alpha1"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
)

func ChangePodTimeoutValidRun() {
	go func() {
		logger := utils.NewLogger()
		for {
			time.Sleep(1 * time.Second)
			// 获取集群中所有的node，并且label中status为PRE_AOP或者POST_AOP的
			// get all node and label status is PRE_AOP or POST_AOP
			preSubmittedChangePods := v1alpha1.ChangePodList{}
			postSubmittedChangePods := v1alpha1.ChangePodList{}
			if err := utils.App.Client.List(context.Background(), &preSubmittedChangePods, client.MatchingFields{utils.ChangePodFieldStatus: v1alpha1.PreSubmitted}); err != nil {
				logger.Error(err, "get PreSubmitted node list error")
			}
			if err := utils.App.Client.List(context.Background(), &postSubmittedChangePods, client.MatchingFields{utils.ChangePodFieldStatus: v1alpha1.PostSubmitted}); err != nil {
				logger.Error(err, "get PostSubmitted node list error")
			}

			// 遍历order，如果status中EntryTimeUnix超过1分钟，将order状态置为超时// TODO 后续敲定时间
			// traverse order, if status EntryTimeUnix is more than 1 minute, set order status to timeout
			for _, node := range preSubmittedChangePods.Items {
				if node.Status.Status == v1alpha1.PreSubmitted && time.Now().Unix()-node.Status.PreSubmitTimeUnix > int64(node.Status.PreTimeoutThreshold) {
					// 将order状态置为超时
					// set order status to timeout
					node.Status.Status = v1alpha1.PreTimeout
					node.Status.UpdateTime = utils.GetNowTime()
					node.Status.UpdateTimeUnix = time.Now().Unix()
					if err := utils.App.Client.Status().Update(context.Background(), &node); err != nil {
						logger.Error(err, "ExeNodeTimeoutValidRun: PreSubmitted update node status error")
					} else {
						logger.Info("ExeNodeTimeoutValidRun:PreSubmitted update node status to pre timeout success", utils.LogChangePodResource, utils.GetResource(&node))
					}
				}
			}
			for _, node := range postSubmittedChangePods.Items {
				if node.Status.Status == v1alpha1.PostSubmitted && time.Now().Unix()-node.Status.PostSubmitTimeUnix > int64(node.Status.PostTimeoutThreshold) {
					// 将order状态置为超时
					// set order status to timeout
					node.Status.Status = v1alpha1.PostTimeout
					node.Status.UpdateTime = utils.GetNowTime()
					node.Status.UpdateTimeUnix = time.Now().Unix()
					if err := utils.App.Client.Status().Update(context.Background(), &node); err != nil {
						logger.Error(err, "ExeNodeTimeoutValidRun: PostSubmitted update node status error")
					} else {
						logger.Info("ExeNodeTimeoutValidRun:PostSubmitted update node status to post timeout success", utils.LogChangePodResource, utils.GetResource(&node))
					}
				}
			}
		}
	}()
}
