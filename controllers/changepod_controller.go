/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	opsClient "gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/client"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils/native"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/apis/app.ops.cloud.alipay.com/v1alpha1"
)

const (
	changePodWorkCount = 20
)

// ChangePodReconciler reconciles a ChangePod object
type ChangePodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=app.ops.cloud.alipay.com,resources=changepods,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=app.ops.cloud.alipay.com,resources=changepods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.ops.cloud.alipay.com,resources=changepods/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ChangePod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ChangePodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// 获取changePod
	// Get changePod
	changePod := v1alpha1.ChangePod{}
	if err := r.Get(ctx, req.NamespacedName, &changePod); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// 获取changeWorkload
	// Get changeWorkload
	changeWorkload := v1alpha1.ChangeWorkload{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: changePod.Namespace, Name: changePod.Spec.ChangeWorkloadId}, &changeWorkload); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// 如果是初始化状态，直接执行初始化
	// If it is the initialization state, execute the initialization directly
	switch changePod.Status.Status {
	case v1alpha1.ExecuteInit:
		return r.executeInitChangePodHandle(ctx, &changePod, &changeWorkload)
	case v1alpha1.PreWait:
		return r.preWaitChangePodHandle(ctx, &changePod)
	case v1alpha1.PostWait:
		return r.postWaitChangePodHandle(ctx, &changePod)
	case v1alpha1.PostFinish:
		return r.postFinishChangePodHandle(ctx, &changePod)
	case v1alpha1.PreTimeout, v1alpha1.PostTimeout, v1alpha1.PreFailed, v1alpha1.PostFailed:
		return r.timeoutOrFailedChangePodHandle(ctx, &changePod)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChangePodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.setChangePodFieldIndex(mgr); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ChangePod{}).
		// Set the maximum number of concurrency
		WithOptions(controller.Options{MaxConcurrentReconciles: changePodWorkCount}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: r.handleUpdateEvent,
			CreateFunc: r.handleCreateEvent,
		}).
		Complete(r)
}

// setChangePodFieldIndex 设置changePod的索引
// setChangePodFieldIndex sets the index of changePod
func (r *ChangePodReconciler) setChangePodFieldIndex(mgr ctrl.Manager) error {
	indexer := mgr.GetFieldIndexer()
	// 设置changePod的索引
	// set the index of changePod
	if err := indexer.IndexField(context.Background(), &v1alpha1.ChangePod{}, utils.ChangePodFieldStatus, func(rawObj client.Object) []string {
		obj, ok := rawObj.(*v1alpha1.ChangePod)
		if !ok {
			return nil
		}
		return []string{obj.Status.Status}
	}); err != nil {
		return err
	}
	if err := indexer.IndexField(context.Background(), &v1alpha1.ChangePod{}, utils.ChangePodFieldChangePodId, func(rawObj client.Object) []string {
		obj, ok := rawObj.(*v1alpha1.ChangePod)
		if !ok {
			return nil
		}
		return []string{obj.Status.ChangePodId}
	}); err != nil {
		return err
	}
	if err := indexer.IndexField(context.Background(), &v1alpha1.ChangePod{}, utils.ChangePodFieldChangeWorkloadId, func(rawObj client.Object) []string {
		obj, ok := rawObj.(*v1alpha1.ChangePod)
		if !ok {
			return nil
		}
		return []string{obj.Spec.ChangeWorkloadId}
	}); err != nil {
		return err
	}
	return nil
}

// handleUpdateEvent handles the event of updating a Pod
func (r *ChangePodReconciler) handleUpdateEvent(e event.UpdateEvent) bool {
	changePod, ok := e.ObjectNew.(*v1alpha1.ChangePod)
	if !ok {
		return false
	}
	return r.handleEvent(changePod)
}

// handleCreateEvent handles the event of creating a new Pod
func (r *ChangePodReconciler) handleCreateEvent(e event.CreateEvent) bool {
	changePod, ok := e.Object.(*v1alpha1.ChangePod)
	if !ok {
		return false
	}
	return r.handleEvent(changePod)
}

// handleEvent handles the event of updating or creating a Pod
func (r *ChangePodReconciler) handleEvent(changePod *v1alpha1.ChangePod) bool {
	return changePod.Status.Status != v1alpha1.ExecuteDone
}

// executeInitChangePodHandle 处理初始化状态的changePod
// executeInitChangePodHandle handles the changePod in the initialization state
func (r *ChangePodReconciler) executeInitChangePodHandle(ctx context.Context, changePod *v1alpha1.ChangePod, workload *v1alpha1.ChangeWorkload) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("preWaitChangePodHandle")
	logger.Info("change pod execute init", utils.LogChangePodResource, utils.GetResource(changePod))

	// judge whether changePod has been written to pod
	podInfos := changePod.Spec.PodInfos
	if utils.IsNotEmpty(podInfos) {
		// If it has been written to pod, mark pod with defensed label
		return ctrl.Result{}, r.markDefensedPods(ctx, podInfos)
	} else {
		// 通过ChangePod获取所有的finished的pod，如果没有则删除changePod
		// Get all finished pods through ChangePod, if not, delete ChangePod
		podArray, err := r.getFinishedPodsWithoutDefenseByChangePodAndDeleteIfNotExist(ctx, changePod)
		if err != nil {
			logger.Error(err, "get finished without defensed pods error", utils.LogChangePodResource, utils.GetResource(changePod))
			return ctrl.Result{}, err
		}

		// get threshold, if threshold is greater than the number of pods, select the threshold number of pods to write to changePod, otherwise write all pods to changePod
		isGreaterThanThreshold := len(podArray) > workload.Spec.CountThreshold
		if isGreaterThanThreshold {
			podArray = podArray[:workload.Spec.CountThreshold]
		}
		changePod.Spec.PodInfos = []v1alpha1.PodSummary{}
		for _, pod := range podArray {
			changePod.Spec.PodInfos = append(changePod.Spec.PodInfos, getPreparingPodSummary(&pod, changePod.Labels[native.DeploymentNameLabel]))
		}

		// update changePod
		if err := r.Update(ctx, changePod); err != nil {
			logger.Error(err, "update changePod error", utils.LogChangePodResource, utils.GetResource(changePod))
			return ctrl.Result{}, err
		}

		// mark pod with defensed label
		for _, pod := range podArray {
			if err := r.defenseProcessedPod(ctx, &pod); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	setChangePodPreWaitStatus(changePod)
	logger.Info("change pod init to pre wait", utils.LogChangePodResource, utils.GetResource(changePod))
	// update changePod status
	return ctrl.Result{}, r.updateChangePodStatus(ctx, changePod)
}

// preWaitChangePodHandle 处理变更前置等待的changePod
// preWaitChangePodHandle handles changePod of pre wait
func (r *ChangePodReconciler) preWaitChangePodHandle(ctx context.Context, changePod *v1alpha1.ChangePod) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("preWaitChangePodHandle")
	logger.Info("change pod pre wait", utils.LogChangePodResource, utils.GetResource(changePod))
	// 提交变更开始通知并获取nodeId
	// submit change start notify and get nodeId
	if nodeId, err := submitChangeStartNotifyAndGetNodeId(changePod); err != nil {
		logger.Error(err, "failed to submit change start notify for change pod", utils.LogChangePodResource, utils.GetResource(changePod))
		// 更新为失败
		// update to failure
		setChangePodPreFailedStatus(changePod)
		logger.Info("change pod pre wait to pre failed", utils.LogChangePodResource, utils.GetResource(changePod))
		return ctrl.Result{}, r.updateChangePodStatus(ctx, changePod)
	} else {
		changePod.Status.ChangePodId = nodeId
		// 更新为preSubmitted状态
		// update to preSubmitted status
		setChangePodPreSubmittedStatus(changePod)
		logger.Info("change pod pre wait to pre submitted", utils.LogChangePodResource, utils.GetResource(changePod))
		return ctrl.Result{}, r.updateChangePodStatus(ctx, changePod)
	}
}

// postWaitChangePodHandle 处理变更后置等待的changePod
// postWaitChangePodHandle handles the changePod of post wait
func (r *ChangePodReconciler) postWaitChangePodHandle(ctx context.Context, changePod *v1alpha1.ChangePod) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("postWaitChangePodHandle")
	logger.Info("change pod post wait", utils.LogChangePodResource, utils.GetResource(changePod))
	// 提交变更结束通知
	// submit change finish notify
	if _, err := opsClient.SubmitChangeFinishNotify(buildChangeFinishNotifyRequest(*changePod)); err != nil {
		logger.Error(err, "failed to submit change finish notify for change pod", utils.LogChangePodResource, utils.GetResource(changePod))
		// 更新为失败
		// update to failure
		setChangePodPostFailedStatus(changePod)
		logger.Info("change pod post wait to post failed", utils.LogChangePodResource, utils.GetResource(changePod))
		return ctrl.Result{}, r.updateChangePodStatus(ctx, changePod)
	} else {
		// 更新为postSubmitted状态
		// update to postSubmitted status
		setChangePodPostSubmittedStatus(changePod)
		logger.Info("change pod post wait to post submitted", utils.LogChangePodResource, utils.GetResource(changePod))
		return ctrl.Result{}, r.updateChangePodStatus(ctx, changePod)
	}
}

// timeoutOrFailedChangePodHandle 处理变更超时或失败的changePod
// timeoutOrFailedChangePodHandle handles changePod that has timeout or failed
func (r *ChangePodReconciler) postFinishChangePodHandle(ctx context.Context, changePod *v1alpha1.ChangePod) (ctrl.Result, error) {
	return r.handleFinishedChangePod(ctx, changePod)
}

// timeoutOrFailedChangePodHandle 处理变更超时或失败的changePod
// timeoutOrFailedChangePodHandle handles changePod that has timeout or failed
func (r *ChangePodReconciler) timeoutOrFailedChangePodHandle(ctx context.Context, changePod *v1alpha1.ChangePod) (ctrl.Result, error) {
	return r.handleFinishedChangePod(ctx, changePod)
}

// handleFinishedChangePod 处理变更完成的changePod，其中完成包括超时和失败
// handleFinishedChangePod handles changePod that has finished, including timeout and failure
func (r *ChangePodReconciler) handleFinishedChangePod(ctx context.Context, changePod *v1alpha1.ChangePod) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("change pod timeout or failed or finish", utils.LogChangePodResource, utils.GetResource(changePod), "ChangePodStatus", changePod.Status.Status)
	changePod.Status.Message = changePod.Status.Status
	// 设置changePod为结束状态
	// set changePod to EXECUTE_DONE status
	setChangePodDoneStatus(changePod)
	logger.Info("change pod timeout or failed or finish to done", utils.LogChangePodResource, utils.GetResource(changePod), "oldChangePodStatus", changePod.Status.Status)
	return ctrl.Result{}, r.updateChangePodStatus(ctx, changePod)
}

// updateChangePodStatus 更新changePod状态
// updateChangePodStatus updates the changePod status
func (r *ChangePodReconciler) updateChangePodStatus(ctx context.Context, changePod *v1alpha1.ChangePod) error {
	if err := r.Status().Update(ctx, changePod); err != nil {
		logger := log.FromContext(ctx).WithName("updateChangePodStatus")
		logger.Error(err, "update changePod status failed", utils.LogChangePodResource, utils.GetResource(changePod))
		return err
	}
	return nil
}

// Mark pods with defensed label if they have been written to podInfos
func (r *ChangePodReconciler) markDefensedPods(ctx context.Context, podInfos []v1alpha1.PodSummary) error {
	logger := log.FromContext(ctx).WithName("defenseProcessedPod")
	for _, podInfo := range podInfos {
		pod := &v1.Pod{}
		if err := r.Get(ctx, types.NamespacedName{Namespace: podInfo.Namespace, Name: podInfo.Pod}, pod); err != nil {
			logger.Error(err, "get pod error", utils.LogPodResource, utils.GetResource(pod))
			return err
		}
		if err := r.defenseProcessedPod(ctx, pod); err != nil {
			return err
		}
	}
	return nil
}

// defenseProcessedPod 给pod打上defense-status=Processed标记
// defenseProcessedPod set pod label defense-status=Processed
func (r *ChangePodReconciler) defenseProcessedPod(ctx context.Context, pod *v1.Pod) (err error) {
	logger := log.FromContext(ctx).WithName("defenseProcessedPod")
	// 先判断是否已经打过标记
	// first check if the label has been set
	if pod.Labels[utils.DefenseStatusLabel] == utils.DefenseStatusLabelProcessed {
		return
	}
	podCopy := pod.DeepCopy()
	pod.Labels[utils.DefenseStatusLabel] = utils.DefenseStatusLabelProcessed
	logger.Info("defense processed pod", utils.LogPodResource, utils.GetResource(pod))
	if err = r.Patch(ctx, pod, client.MergeFrom(podCopy)); err != nil {
		logger.Error(err, "defense processed pod failed", utils.LogPodResource, utils.GetResource(pod))
		return
	}
	return
}

// getFinishedPodsWithoutDefenseByChangePodAndDeleteIfNotExist 通过ChangePod获取所有的finished的pod，如果没有则删除changePod
// getFinishedPodsWithoutDefenseByChangePodAndDeleteIfNotExist get all finished pod by ChangePod and delete changePod if not exist
func (r *ChangePodReconciler) getFinishedPodsWithoutDefenseByChangePodAndDeleteIfNotExist(ctx context.Context, changePod *v1alpha1.ChangePod) (podArray []v1.Pod, err error) {
	logger := log.FromContext(ctx).WithName("getAllPodsByChangePod")
	// 过滤掉没有finished label的pod
	// filter pod without finished label
	if array, err := r.getAllPodsByChangePod(ctx, changePod); err != nil {
		return nil, err
	} else {
		podArray = filterPodWithoutFinishedOrDefensedLabel(array)
	}
	// judge whether the number of pods is empty, if empty, delete changePod
	if !utils.IsNotEmpty(podArray) {
		if err := r.Delete(ctx, changePod); err != nil {
			logger.Error(err, "delete changePod error", utils.LogChangePodResource, utils.GetResource(changePod))
			return podArray, err
		}
		return podArray, fmt.Errorf("no pod found by changePod %s, will delete changePod", changePod.Name)
	}
	return podArray, nil
}

// getAllPodsByChangePod 通过workload获取所有的finished的pod
// getAllPodsByChangePod get all finished pod by workload
func (r *ChangePodReconciler) getAllPodsByChangePod(ctx context.Context, changePod *v1alpha1.ChangePod) (podArray []v1.Pod, err error) {
	logger := log.FromContext(ctx).WithName("getAllPodsByChangePod")
	podList := &v1.PodList{}
	podArray = make([]v1.Pod, 0)
	if err = r.List(ctx, podList, client.InNamespace(changePod.Namespace),
		client.MatchingLabels{native.AdmissionWebhookVersionLabel: changePod.Labels[native.AdmissionWebhookVersionLabel]}); err != nil {
		logger.Error(err, "getAllPodsByChangePod get pod error", utils.LogChangePodResource, utils.GetResource(changePod))
		return podArray, err
	}
	return podList.Items, nil
}

// submitChangeStartNotifyAndGetNodeId 提交变更开始通知并获取nodeId
// submitChangeStartNotifyAndGetNodeId submits the change start notification and gets the nodeId
func submitChangeStartNotifyAndGetNodeId(changePod *v1alpha1.ChangePod) (string, error) {
	result, err := opsClient.SubmitChangeStartNotify(buildChangeStartNotifyRequest(*changePod))
	if err != nil {
		return "", err
	}
	domain, ok := result.Domain.(map[string]interface{})
	if !ok || domain["nodeId"] == nil {
		return "", fmt.Errorf("submitChangeStartNotifyAndGetNodeId get nodeId failed, result: %v", result)
	}
	return fmt.Sprint(domain["nodeId"]), nil
}

// submitChangeEndNotify 提交变更结束通知
// submitChangeEndNotify submits the change end notification
func setChangePodPreSubmittedStatus(changePod *v1alpha1.ChangePod) {
	setChangePodStatus(changePod, v1alpha1.PreSubmitted)
	changePod.Status.PreTimeoutThreshold = utils.ChangePodPreTimeoutThreshold
	changePod.Status.PreSubmitTime = utils.GetNowTime()
	changePod.Status.PreSubmitTimeUnix = time.Now().Unix()
}

// setChangePodSubmittedStatus 设置changePod为submitted状态
// setChangePodSubmittedStatus sets the changePod to the submitted state
func setChangePodPostSubmittedStatus(changePod *v1alpha1.ChangePod) {
	setChangePodStatus(changePod, v1alpha1.PostSubmitted)
	changePod.Status.PostTimeoutThreshold = utils.ChangePodPostTimeoutThreshold
	changePod.Status.PostSubmitTime = utils.GetNowTime()
	changePod.Status.PostSubmitTimeUnix = time.Now().Unix()
}

// setChangePodDoneStatus 设置changePod为结束状态
// setChangePodDoneStatus sets the changePod to the end state
func setChangePodDoneStatus(changePod *v1alpha1.ChangePod) {
	setChangePodStatus(changePod, v1alpha1.ExecuteDone)
}

// setChangePodPreWaitStatus 设置changePod为preWait状态
// setChangePodPreWaitStatus sets the changePod to the preWait state
func setChangePodPreWaitStatus(changePod *v1alpha1.ChangePod) {
	setChangePodStatus(changePod, v1alpha1.PreWait)
}

// setChangePodPreFailedStatus 设置changePod为preFailed状态
// setChangePodPreFailedStatus sets the changePod to the preFailed state
func setChangePodPreFailedStatus(changePod *v1alpha1.ChangePod) {
	setChangePodStatus(changePod, v1alpha1.PreFailed)
}

// setChangePodPostFailedStatus 设置changePod为postFailed状态
// setChangePodPostFailedStatus sets the changePod to the postFailed state
func setChangePodPostFailedStatus(changePod *v1alpha1.ChangePod) {
	setChangePodStatus(changePod, v1alpha1.PostFailed)
}

// setChangePodStatus 设置changePod状态
// setChangePodStatus sets the changePod status
func setChangePodStatus(changePod *v1alpha1.ChangePod, status string) {
	changePod.Status.Status = status
	changePod.Status.UpdateTime = utils.GetNowTime()
	changePod.Status.UpdateTimeUnix = time.Now().Unix()
}

// getPreparingPodSummary 获取预校验的summary
// getPreparingPodSummary get preparing pod summary
func getPreparingPodSummary(pod *v1.Pod, deploymentName string) (podSummary v1alpha1.PodSummary) {
	podSummary = v1alpha1.PodSummary{
		App:       deploymentName,
		Hostname:  pod.Name,
		Workspace: "default",
		Pod:       pod.Name,
		Ip:        pod.Status.PodIP,
		Namespace: pod.Namespace,
	}
	return podSummary
}

// buildChangeStartNotifyRequest 构建变更开始通知请求
// buildChangeStartNotifyRequest builds the change start notification request
func buildChangeStartNotifyRequest(changePod v1alpha1.ChangePod) opsClient.OpsCloudChangeExecBatchStartNotifyRequest {
	var podInfos []string
	for _, pod := range changePod.Spec.PodInfos {
		marshal, err := json.Marshal(pod)
		if err != nil {
			return opsClient.OpsCloudChangeExecBatchStartNotifyRequest{}
		}
		podInfos = append(podInfos, string(marshal))
	}
	request := opsClient.OpsCloudChangeExecBatchStartNotifyRequest{
		ChangePhase:              utils.ChangePhase,
		Executor:                 utils.DefaultCreator,
		EffectiveTargetType:      utils.StringRecordSpecEffectiveTargetType,
		EffectiveTargetLocations: podInfos,
		Platform:                 opsClient.Platform,
		ChangeSceneKey:           utils.ChangeSceneKeyRollingUpdate,
		BizExecOrderId:           changePod.Spec.ChangeWorkloadId,
		TldcTenantCode:           utils.DefaultTldcTenantCode,
	}
	return request
}

// buildChangeFinishNotifyRequest 构建变更结束通知请求
// buildChangeFinishNotifyRequest builds the change finish notification request
func buildChangeFinishNotifyRequest(changePod v1alpha1.ChangePod) opsClient.OpsCloudChangeFinishNotifyRequest {
	request := opsClient.OpsCloudChangeFinishNotifyRequest{
		BizExecOrderId: changePod.Spec.ChangeWorkloadId,
		Success:        true,
		ServiceResult:  "{}",
		Platform:       opsClient.Platform,
		ChangeSceneKey: utils.ChangeSceneKeyRollingUpdate,
		NodeId:         changePod.Status.ChangePodId,
		TldcTenantCode: utils.DefaultTldcTenantCode,
	}
	return request
}
