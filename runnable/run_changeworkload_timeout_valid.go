package runnable

import (
	"context"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/apis/app.ops.cloud.alipay.com/v1alpha1"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
)

// ChangeWorkloadTimeoutValidRun Batch pod handling
func ChangeWorkloadTimeoutValidRun() {
	go func() {
		logger := utils.NewLogger()
		for {
			time.Sleep(1 * time.Second)
			// 获取集群中所有的order，并且label中status为0或者1的
			// get all order and label status is 0 or 1
			changeWorkloadList := v1alpha1.ChangeWorkloadList{}
			listOptions := []client.ListOption{
				client.MatchingFields{
					utils.ChangeWorkloadFieldStatus: v1alpha1.Running,
				},
			}
			if err := utils.App.Client.List(context.Background(), &changeWorkloadList, listOptions...); err != nil {
				logger.Error(err, "get new or check record list error")
				continue
			}
			// 遍历order，如果status中EntryTimeUnix超过1分钟，将order状态置为超时// TODO 后续敲定时间
			// traverse order, if status EntryTimeUnix is more than 1 minute, set order status to timeout
			for _, order := range changeWorkloadList.Items {
				if order.Status.Status != v1alpha1.Running {
					continue
				}
				if utils.IsNotEmpty(order.Status.DefensePreparingPods) && time.Now().Unix()-order.Status.EntryTimeUnix > int64(order.Spec.WaitTimeThreshold) {
					// 将order状态置为超时
					// set order status to timeout
					order.Status.Status = v1alpha1.TimeOutPreThreshold
					order.Status.UpdateTime = utils.GetNowTime()
					order.Status.UpdateTimeUnix = time.Now().Unix()
					if err := utils.App.Client.Status().Update(context.Background(), &order); err != nil {
						logger.Error(err, "OpsCheckRecordTimeOutRun:update record status error")
					} else {
						logger.Info("OpsCheckRecordTimeOutRun:update record status to timeout success", utils.LogChangeWorkloadResource, utils.GetResource(&order))
					}
				}
			}
		}
	}()
}
