package runnable

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
)

func DeploymentStatusPauseRun() {
	go func() {
		logger := utils.NewLogger()
		for {
			// 查询指的deployment
			deployment := &appsv1.Deployment{}
			if err := utils.App.Client.Get(context.Background(), client.ObjectKey{
				Namespace: "default",
				Name:      "sleep",
			}, deployment); err != nil {
				logger.Error(err, "get deployment error")
			} else {
				deployment.Spec.Paused = true
				if err := utils.App.Client.Update(context.Background(), deployment); err != nil {
					logger.Error(err, "update deployment error")
				}
			}
		}
	}()
}
