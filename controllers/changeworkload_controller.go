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
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/resource"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils/native"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/apis/app.ops.cloud.alipay.com/v1alpha1"
)

const (
	changeWorkLoadWorkCount = 5
)

// ChangeWorkloadReconciler reconciles a ChangeWorkload object
type ChangeWorkloadReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=app.ops.cloud.alipay.com,resources=changeworkloads,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=app.ops.cloud.alipay.com,resources=changeworkloads/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.ops.cloud.alipay.com,resources=changeworkloads/finalizers,verbs=update
//+kubebuilder:rbac:groups="apps",resources=replicasets,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ChangeWorkload object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ChangeWorkloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	workload := &v1alpha1.ChangeWorkload{}
	namespacedName := req.NamespacedName
	resourceType := utils.GetResourceTypeFromNamespaceName(namespacedName)

	switch {
	case utils.IsPodResourceType(resourceType):
		if changeWorkload, err := r.podReconcile(ctx, utils.GetPodRequestFromPodNameSpaceName(namespacedName)); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		} else if changeWorkload != nil {
			workload = changeWorkload
		} else {
			return ctrl.Result{}, nil
		}
	case utils.IsChangePodResourceType(resourceType):
		if changeWorkload, err := r.changePodReconcile(ctx, utils.GetPodRequestFromPodNameSpaceName(namespacedName)); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		} else if changeWorkload != nil {
			workload = changeWorkload
		} else {
			return ctrl.Result{}, nil
		}
	default:
		if err := r.Get(ctx, req.NamespacedName, workload); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	switch workload.Status.Status {
	case v1alpha1.Init:
		return r.initChangeWorkloadHandle(ctx, workload)
	case v1alpha1.Success:
		return r.successChangeWorkloadHandle(ctx, workload)
	case v1alpha1.Running:
		return r.runningChangeWorkloadHandle(ctx, workload)
	case v1alpha1.Suspend:
		return r.suspendChangeWorkloadHandle(ctx, workload)
	case v1alpha1.TimeOutPreThreshold:
		return r.waitTimeoutChangeWorkloadHandle(ctx, workload)
	default:
		return ctrl.Result{}, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChangeWorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.setChangeWorkloadFieldIndex(mgr); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ChangeWorkload{}).
		// Set the maximum number of concurrency
		WithOptions(controller.Options{MaxConcurrentReconciles: changeWorkLoadWorkCount}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: r.handleUpdateEvent,
			CreateFunc: r.handleCreateEvent,
		}).
		// 监听 Pod 事件
		Watches(&source.Kind{Type: &v1.Pod{}}, handler.EnqueueRequestsFromMapFunc(watchPodEventHandler)).
		// 监听 ChangePod 事件
		Watches(&source.Kind{Type: &v1alpha1.ChangePod{}}, handler.EnqueueRequestsFromMapFunc(watchChangePodEventHandler)).
		Complete(r)
}

// setChangeWorkloadFieldIndex 设置changeWorkload的索引
// setChangeWorkloadFieldIndex sets the index of changeWorkload
func (r *ChangeWorkloadReconciler) setChangeWorkloadFieldIndex(mgr ctrl.Manager) error {
	indexer := mgr.GetFieldIndexer()
	// 设置changeWorkload的status索引
	// Set the status index of changeWorkload
	if err := indexer.IndexField(context.Background(), &v1alpha1.ChangeWorkload{}, utils.ChangeWorkloadFieldStatus, func(rawObj client.Object) []string {
		obj, ok := rawObj.(*v1alpha1.ChangeWorkload)
		if !ok {
			return nil
		}
		return []string{obj.Status.Status}
	}); err != nil {
		return err
	}
	return nil
}

func (r *ChangeWorkloadReconciler) handleUpdateEvent(e event.UpdateEvent) bool {
	switch newObj := e.ObjectNew.(type) {
	case *v1.Pod:
		return r.podHandleEvent(newObj)
	case *v1alpha1.ChangePod:
		return r.changePodUpdateHandleEvent(e.ObjectOld.(*v1alpha1.ChangePod), newObj)
	case *v1alpha1.ChangeWorkload:
		return r.changeWorkloadHandleEvent(newObj)
	default:
		return false
	}
}

// handleCreateEvent handles the event of creating a new Pod
func (r *ChangeWorkloadReconciler) handleCreateEvent(e event.CreateEvent) bool {
	switch newObj := e.Object.(type) {
	case *v1.Pod:
		return r.podHandleEvent(newObj)
	case *v1alpha1.ChangePod:
		return r.changePodCreateHandleEvent(newObj)
	case *v1alpha1.ChangeWorkload:
		return r.changeWorkloadHandleEvent(newObj)
	default:
		return false
	}
}

// changeWorkloadHandleEvent handles the event of updating or creating a Pod
func (r *ChangeWorkloadReconciler) changeWorkloadHandleEvent(workload *v1alpha1.ChangeWorkload) bool {
	isInitStatus := isInitStatusChangeWorkload(workload)
	isFailedStatus := isFailedStatusChangeWorkload(workload)
	// 如果是初始化状态或者失败状态，不处理
	// if it is an initialization status or a failed status, it is not processed
	if isInitStatus || isFailedStatus {
		return false
	}
	return true
}

// initChangeWorkloadHandle 处理init状态
// initChangeWorkloadHandle handles the init status
func (r *ChangeWorkloadReconciler) initChangeWorkloadHandle(ctx context.Context, changeWorkload *v1alpha1.ChangeWorkload) (ctrl.Result, error) {
	//logger := log.FromContext(ctx).WithName("initChangeWorkloadHandle")
	// 单机单元测试时，不需要调用ops-cloud接口
	//_, err := opsClient.SubmitChangeExecOrder(buildChangeChangeWorkloadRequest(*changeWorkload))
	//if err != nil {
	//	logger.Error(err, "changePod to SubmitChangeFinishNotify failed", utils.LogChangeWorkloadResource, utils.GetResource(changeWorkload))
	//	changeWorkload.Status.Status = v1alpha1.Failed
	//	return ctrl.Result{}, r.updateWorkloadStatus(ctx, changeWorkload)
	//}
	changeWorkload.Status.Status = v1alpha1.Running
	return ctrl.Result{}, r.updateWorkloadStatus(ctx, changeWorkload)
}

// successChangeWorkloadHandle 处理success状态的changeWorkload
// successChangeWorkloadHandle handles the success status of changeWorkload
func (r *ChangeWorkloadReconciler) successChangeWorkloadHandle(ctx context.Context, workload *v1alpha1.ChangeWorkload) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("successChangeWorkloadHandle")
	r.deleteOldChangeWorkload(ctx, workload)
	_, err := r.ensureInitChangePodCreatedIfNecessary(ctx, workload, false)
	if err != nil {
		logger.Error(err, "ensureInitChangePodCreatedIfNecessary error", utils.LogChangeWorkloadResource, utils.GetResource(workload))
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, r.syncChangeWorkloadStatus(ctx, workload)
}

// runningChangeWorkloadHandle 处理running状态的changeWorkload
// runningChangeWorkloadHandle handles the running status of changeWorkload
func (r *ChangeWorkloadReconciler) runningChangeWorkloadHandle(ctx context.Context, workload *v1alpha1.ChangeWorkload) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("RunningChangeWorkloadHandle")
	_, err := r.ensureInitChangePodCreatedIfNecessary(ctx, workload, true)
	if err != nil {
		logger.Error(err, "ensureInitChangePodCreatedIfNecessary error", utils.LogChangeWorkloadResource, utils.GetResource(workload))
		return ctrl.Result{}, err
	}
	// Sync current workload status
	return ctrl.Result{}, r.syncChangeWorkloadStatus(ctx, workload)
}

// suspendChangeWorkloadHandle 处理suspend状态的changeWorkload
// suspendChangeWorkloadHandle handles the suspend status of changeWorkload
func (r *ChangeWorkloadReconciler) suspendChangeWorkloadHandle(ctx context.Context, workload *v1alpha1.ChangeWorkload) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("suspendChangeWorkloadHandle")
	_, err := r.ensureInitChangePodCreatedIfNecessary(ctx, workload, false)
	if err != nil {
		logger.Error(err, "ensureInitChangePodCreatedIfNecessary error", utils.LogChangeWorkloadResource, utils.GetResource(workload))
		return ctrl.Result{}, err
	}
	// Sync current workload status
	return ctrl.Result{}, r.syncChangeWorkloadStatus(ctx, workload)
}

// waitTimeoutChangeWorkloadHandle 处理waitTimeout状态的changeWorkload
// waitTimeoutChangeWorkloadHandle handles the waitTimeout status of changeWorkload
func (r *ChangeWorkloadReconciler) waitTimeoutChangeWorkloadHandle(ctx context.Context, workload *v1alpha1.ChangeWorkload) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("WaitTimeoutChangeWorkloadHandle")
	newChangePod, err := r.ensureInitChangePodCreatedIfNecessary(ctx, workload, false)
	if err != nil {
		logger.Error(err, "ensureInitChangePodCreatedIfNecessary error", utils.LogChangeWorkloadResource, utils.GetResource(workload))
		return ctrl.Result{}, err
	}
	if newChangePod != nil {
		logger.Info("Created a new changePod", utils.LogChangeWorkloadResource, utils.GetResource(workload), utils.LogPodResource, utils.GetResource(newChangePod))
		workload.Status.Status = v1alpha1.Running
		return ctrl.Result{}, r.updateWorkloadStatus(ctx, workload)
	}
	return ctrl.Result{}, r.syncChangeWorkloadStatus(ctx, workload)
}

// ensureInitChangePodCreatedIfNecessary 验证是否有init状态的changepod,不存在时判断是否达到条件，如果达到创建init的changePod
// ensureInitChangePodCreatedIfNecessary verify whether there is an init status changePod, if not, determine whether the conditions are met, and if so, create an init changePod
func (r *ChangeWorkloadReconciler) ensureInitChangePodCreatedIfNecessary(ctx context.Context, workload *v1alpha1.ChangeWorkload, reachedThreshold bool) (changePod *v1alpha1.ChangePod, err error) {
	// Check if there is an init status changePod, if so, do nothing and return the number of existing changePods
	exist, changePodNum, err := r.isExistInitStatusChangePod(ctx, workload)
	if err != nil || exist {
		return changePod, err
	}
	var ifNecessary bool
	if reachedThreshold {
		// Check if preparing pod number reaches the threshold, if so, create a new changePod
		ifNecessary = r.isDefensePreparingPodsThresholdReached(ctx, workload)
	} else {
		// Check if there are preparing pods, if so, create a new changePod
		ifNecessary = r.isDefensePreparingPodsExist(ctx, workload)
	}
	if ifNecessary {
		// Create new changePod
		newChangePod, err := r.createNewChangePod(ctx, workload, changePodNum)
		if err == nil {
			r.patchWorkloadEntryTime(ctx, workload)
		}
		return newChangePod, err
	}
	return changePod, err
}

// 判断是否存在init状态的changePod，如果存在则不进行后续操作。如果不存在则返回现有的changePod个数
// judge whether there is init status changePod, if there is, do not perform subsequent operations. If not, return the number of existing changePods
func (r *ChangeWorkloadReconciler) isExistInitStatusChangePod(ctx context.Context, workload *v1alpha1.ChangeWorkload) (exist bool, changePodNum int, err error) {
	// 1、获取当前workload下的所有changePod
	// 1、get all changePods under current workload
	changePods, err := r.getAllChangePodsByWorkload(ctx, workload)
	if err != nil {
		return false, utils.NumberZero, err
	}
	// 2、判断是否存在init状态的changePod
	// 2、judge whether there is init status changePod
	for _, changePod := range changePods {
		if changePod.Status.Status == v1alpha1.ExecuteInit {
			exist = true
		}
	}
	return exist, len(changePods), nil
}

// createNewChangePod 创建新的changePod
// createNewChangePod create new changePod
func (r *ChangeWorkloadReconciler) createNewChangePod(ctx context.Context, changeWorkload *v1alpha1.ChangeWorkload, changePodNum int) (changePod *v1alpha1.ChangePod, err error) {
	logger := log.FromContext(ctx).WithName("createNewChangePod")
	changePodFactory := resource.NativeChangePodFactory{ChangeWorkload: changeWorkload, ChangePodNum: changePodNum}
	instance := changePodFactory.NewInstance()
	changePod, ok := instance.(*v1alpha1.ChangePod)
	if !ok {
		logger.Error(err, "createNewChangeWorkload NewInstance workload error", utils.LogChangeWorkloadResource, utils.GetResource(changeWorkload))
		return changePod, err
	} else {
		if err = controllerutil.SetControllerReference(changeWorkload, changePod, r.Scheme); err != nil {
			logger.Error(err, "createNewChangePod SetControllerReference error", utils.LogChangeWorkloadResource, utils.GetResource(changeWorkload), utils.LogChangePodResource, utils.GetResource(changePod))
		}
		if err = r.Create(ctx, changePod); err != nil {
			logger.Error(err, "create changePod error", utils.LogChangeWorkloadResource, utils.GetResource(changeWorkload))
			return changePod, err
		}
		logger.Info("create changePod success", utils.LogChangeWorkloadResource, utils.GetResource(changeWorkload), utils.LogChangePodResource, utils.GetResource(changePod))
	}
	return changePod, nil
}

// updateWorkloadStatus 更新workload的状态
// updateWorkloadStatus update workload status
func (r *ChangeWorkloadReconciler) updateWorkloadStatus(ctx context.Context, workload *v1alpha1.ChangeWorkload) error {
	logger := log.FromContext(ctx).WithName("updateWorkloadStatus")
	workload.Status.UpdateTime = utils.GetNowTime()
	workload.Status.UpdateTimeUnix = time.Now().Unix()
	if err := r.Status().Update(ctx, workload); err != nil {
		logger.Error(err, "update workload status error", utils.LogChangeWorkloadResource, utils.GetResource(workload))
		return err
	}
	return nil
}

// addOrRemoveDeploymentSuspendLabel 添加或删除deployment的suspend label
// addOrRemoveDeploymentSuspendLabel add or remove deployment suspend label
func (r *ChangeWorkloadReconciler) addOrRemoveDeploymentSuspendLabel(ctx context.Context, deployment *appsv1.Deployment, add bool) error {
	logger := log.FromContext(ctx).WithName("addOrRemoveDeploymentSuspendLabel")
	patch := client.MergeFrom(deployment.DeepCopy())
	if add && utils.ConfigIsBlockingUp() {
		if _, ok := deployment.Labels[utils.SuspendLabel]; ok {
			return nil
		}
		deployment.Labels[utils.SuspendLabel] = strconv.FormatInt(time.Now().Unix(), utils.NumberTen)
	} else {
		if _, ok := deployment.Labels[utils.SuspendLabel]; !ok {
			return nil
		}
		delete(deployment.Labels, utils.SuspendLabel)
	}
	if err := r.Patch(ctx, deployment, patch); err != nil {
		logger.Error(err, "add or remove deployment suspend label error", utils.LogDeploymentResource, utils.GetResource(deployment))
		return err
	}
	return nil
}

// patchWorkloadEntryTime 更新workload的批次时间
// patchWorkloadEntryTime update workload entry time
func (r *ChangeWorkloadReconciler) patchWorkloadEntryTime(ctx context.Context, workload *v1alpha1.ChangeWorkload) {
	logger := log.FromContext(ctx).WithName("patchWorkloadEntryTime")
	deepCopy := workload.DeepCopy()
	workload.Status.EntryTime = utils.GetNowTime()
	workload.Status.EntryTimeUnix = time.Now().Unix()
	if err := r.Status().Patch(ctx, workload, client.MergeFrom(deepCopy)); err != nil {
		logger.Error(err, "patch workload entry time error", utils.LogChangeWorkloadResource, utils.GetResource(workload))
	}
}

// isDefensePreparingPodsThresholdReached 获取当前workload下有finished label并且没有defense label的pod，如果个数大于等于阈值则返回true
// isDefensePreparingPodsThresholdReached get pod list by four tuple, then filter pod with finished label and without defense label, if the number of pod is greater than or equal to threshold, return true
func (r *ChangeWorkloadReconciler) isDefensePreparingPodsThresholdReached(ctx context.Context, changeWorkload *v1alpha1.ChangeWorkload) bool {
	logger := log.FromContext(ctx).WithName("isDefensePreparingPodsThresholdReached")
	if podArray, err := r.getFinishedWithoutDefensedPodsByWorkload(ctx, changeWorkload); err != nil {
		logger.Error(err, "get pod error", utils.LogChangeWorkloadResource, utils.GetResource(changeWorkload))
		return false
	} else {
		return len(podArray) >= changeWorkload.Spec.CountThreshold
	}
}

// isDefensePreparingPodsExist 获取当前workload下有finished label并且没有defense label的pod，如果存在则返回true
// isDefensePreparingPodsExist get pod list by four tuple, then filter pod with finished label and without defense label, if the number of pod is greater than or equal to threshold, return true
func (r *ChangeWorkloadReconciler) isDefensePreparingPodsExist(ctx context.Context, changeWorkload *v1alpha1.ChangeWorkload) bool {
	logger := log.FromContext(ctx).WithName("isDefensePreparingPodsExist")
	if podArray, err := r.getFinishedWithoutDefensedPodsByWorkload(ctx, changeWorkload); err != nil {
		logger.Error(err, "get pod error", utils.LogChangeWorkloadResource, utils.GetResource(changeWorkload))
		return false
	} else {
		return len(podArray) > utils.NumberZero
	}
}

// getFinishedPodsByWorkload 通过workload获取所有的finished的pod
// getFinishedPodsByWorkload get all finished pod by workload
func (r *ChangeWorkloadReconciler) getFinishedPodsByWorkload(ctx context.Context, workload *v1alpha1.ChangeWorkload) (podArray []v1.Pod, err error) {
	// 过滤掉没有finished label的pod
	// filter pod without finished label
	if array, err := r.getAllPodsByWorkload(ctx, workload); err != nil {
		return nil, err
	} else {
		return filterPodWithoutFinishedLabel(array), nil
	}
}

// getFinishedWithoutDefensedPodsByWorkload 通过workload获取所有的finished的pod
// getFinishedWithoutDefensedPodsByWorkload get all finished pod by workload
func (r *ChangeWorkloadReconciler) getFinishedWithoutDefensedPodsByWorkload(ctx context.Context, workload *v1alpha1.ChangeWorkload) (podArray []v1.Pod, err error) {
	// 过滤掉没有finished label的pod
	// filter pod without finished label
	if array, err := r.getAllPodsByWorkload(ctx, workload); err != nil {
		return nil, err
	} else {
		return filterPodWithoutFinishedOrDefensedLabel(array), nil
	}
}

// getAllPodsByWorkload 通过workload获取所有的finished的pod
// getAllPodsByWorkload get all finished pod by workload
func (r *ChangeWorkloadReconciler) getAllPodsByWorkload(ctx context.Context, workload *v1alpha1.ChangeWorkload) (podArray []v1.Pod, err error) {
	return r.getAllPodsByVersion(ctx, workload.Namespace, workload.Labels[native.AdmissionWebhookVersionLabel])
}

// getAllPodMapByWorkload 通过workload获取所有的pod,返回的是一个map，key是pod的name，value是pod
// getAllPodMapByWorkload get all pod by workload, return a map, key is pod name, value is pod
func (r *ChangeWorkloadReconciler) getAllPodMapByWorkload(ctx context.Context, workload *v1alpha1.ChangeWorkload) (podMap map[string]v1.Pod, err error) {
	podMap = make(map[string]v1.Pod)
	podArray, err := r.getAllPodsByVersion(ctx, workload.Namespace, workload.Labels[native.AdmissionWebhookVersionLabel])
	if err != nil {
		return podMap, err
	}
	for _, pod := range podArray {
		podMap[pod.Name] = pod
	}
	return podMap, nil
}

// getAllPodsByVersion 获取全部的有version的label的pod
// getAllPodsByVersion get all pod with version label
func (r *ChangeWorkloadReconciler) getAllPodsByVersion(ctx context.Context, ns string, webhookVersionLabel string) (finishedPodArray []v1.Pod, err error) {
	logger := log.FromContext(ctx).WithName("getAllPodsByVersion")
	podList := &v1.PodList{}
	if err = r.List(ctx, podList, client.InNamespace(ns), client.MatchingLabels{native.AdmissionWebhookVersionLabel: webhookVersionLabel}); err != nil {
		logger.Error(err, "get pod list error", "webhookVersionLabel", webhookVersionLabel)
		return
	}
	return podList.Items, nil
}

// syncChangeWorkloadStatus 同步workload的状态
// syncChangeWorkloadStatus sync workload status
func (r *ChangeWorkloadReconciler) syncChangeWorkloadStatus(ctx context.Context, workload *v1alpha1.ChangeWorkload) error {
	logger := log.FromContext(ctx).WithName("syncChangeWorkloadStatus")
	// 查询workload下的所有finished的changePod
	// query all finished changePod under workload
	changePods, err := r.getFinishedChangePodsByWorkload(ctx, workload)
	if err != nil {
		logger.Error(err, "get changePod error", utils.LogChangeWorkloadResource, utils.GetResource(workload))
		return err
	}
	// 获取workload下所有的pod
	// get all pod under workload
	allPodMapByWorkload, err := r.getAllPodMapByWorkload(ctx, workload)
	if err != nil {
		logger.Error(err, "get pod error", utils.LogChangeWorkloadResource, utils.GetResource(workload))
		return err
	}
	if utils.IsNotEmpty(changePods) {
		// 创建两个数组，分别存储成功与失败的changePod
		// create two array, one for success and the other for failure
		successChangePods := make([]v1alpha1.PodSummary, 0)
		failureChangePods := make([]v1alpha1.PodSummary, 0)
		for _, changePod := range changePods {
			switch changePod.Status.Message {
			case v1alpha1.PreTimeout, v1alpha1.PreFailed, v1alpha1.PostFailed, v1alpha1.PostTimeout:
				successChangePods = append(successChangePods, changePod.Spec.PodInfos...)
			case v1alpha1.PostFinish:
				passPods, failedPods := getPassAndFailedPodsByPostFinishChangePod(&changePod)
				successChangePods = append(successChangePods, passPods...)
				failureChangePods = append(failureChangePods, failedPods...)
			}
		}
		// 更新workload的状态
		// update workload status
		passPods := utils.RemoveDuplicatePod(successChangePods, allPodMapByWorkload)
		failPods := utils.RemoveDuplicatePod(failureChangePods, allPodMapByWorkload)
		if utils.IsPodSummarySliceEqual(passPods, workload.Status.DefenseCheckPassPods) && utils.IsPodSummarySliceEqual(failPods, workload.Status.DefenseCheckFailPods) {
			return nil
		}
		workload.Status.DefenseCheckPassPods = passPods
		workload.Status.DefenseCheckFailPods = failPods
		if err := r.validateChangeWorkloadSuccessOrSuspend(ctx, workload); err != nil {
			return err
		}
	}
	return nil
}

// validateChangeWorkloadSuccess 验证workload是否已经完成，完成是指workload中成功的pod数量去重后等于replicas，并且均为成功状态，这时证明所有的pod都已经完成了
// validateChangeWorkloadSuccess validate whether workload is success, success means the number of success pod is equal to replicas after remove duplicate, and all of them are success status, then all pod is success
func (r *ChangeWorkloadReconciler) validateChangeWorkloadSuccessOrSuspend(ctx context.Context, workload *v1alpha1.ChangeWorkload) error {
	deployment, err := r.getDeploymentByWorkload(workload)
	if err != nil {
		return err
	}
	replicas := int(*deployment.Spec.Replicas)
	// 获取目前全部的有finished的label的pod
	// get all pod with finished label
	finishedPods, err := r.getFinishedPodsByWorkload(ctx, workload)
	if err != nil {
		return err
	}
	// 如果去重后的数组长度等于replicas
	// if the length of array after remove duplicate is equal to replicas
	isAllPodFinished := len(finishedPods) >= replicas
	// 判断changeWorkload中pass的pod数量是否等于replicas
	// judge whether the number of pass pod in changeWorkload is equal to replicas
	isAllPodPass := len(workload.Status.DefenseCheckPassPods) == replicas
	if isAllPodFinished && isAllPodPass {
		workload.Status.Status = v1alpha1.Success
	}
	// 判断changeWorkload中fail的pod数量是否不为空
	// judge whether the number of fail pod in changeWorkload is not empty
	isPodFail := len(workload.Status.DefenseCheckFailPods) > utils.NumberZero
	if isPodFail {
		workload.Status.Status = v1alpha1.Suspend
	}
	if err := r.updateWorkloadStatus(ctx, workload); err != nil {
		return err
	}
	if err := r.addOrRemoveDeploymentSuspendLabel(ctx, deployment, isPodFail); err != nil {
		return err
	}
	return nil
}

// getDeploymentByWorkload 通过workload获取deployment
// getDeploymentByWorkload get deployment by workload
func (r *ChangeWorkloadReconciler) getDeploymentByWorkload(workload *v1alpha1.ChangeWorkload) (deployment *appsv1.Deployment, err error) {
	// 通过workload的ownerReference获取deployment
	// get deployment by workload ownerReference
	deployment = &appsv1.Deployment{}
	ownerReferences := workload.OwnerReferences
	for _, ownerReference := range ownerReferences {
		if ownerReference.Kind == native.DeploymentKind {
			if err := utils.App.Client.Get(context.Background(), client.ObjectKey{
				Namespace: workload.Namespace,
				Name:      ownerReference.Name,
			}, deployment); err != nil {
				return nil, err
			}
			break
		}
	}
	return deployment, nil
}

// getFinishedChangePodsByWorkload 通过workload获取所有的finished的changePod
// getFinishedChangePodsByWorkload get all finished changePod by workload
func (r *ChangeWorkloadReconciler) getFinishedChangePodsByWorkload(ctx context.Context, workload *v1alpha1.ChangeWorkload) (changePods []v1alpha1.ChangePod, err error) {
	// 1、获取当前workload下的所有changePod
	// 1、get all changePods under current workload
	changePods, err = r.getAllChangePodsByWorkload(ctx, workload)
	if err != nil {
		return changePods, err
	}
	// 2、过滤掉状态不是ExecutedDone的changePod
	// 2、filter changePods which status is not ExecutedDone
	executedDoneChangePods := make([]v1alpha1.ChangePod, 0)
	for _, changePod := range changePods {
		if changePod.Status.Status == v1alpha1.ExecuteDone {
			executedDoneChangePods = append(executedDoneChangePods, changePod)
			continue
		}
	}
	return executedDoneChangePods, nil
}

// getAllChangePodsByWorkload 通过workload获取所有的changePod
// getAllChangePodsByWorkload get all changePod by workload
func (r *ChangeWorkloadReconciler) getAllChangePodsByWorkload(ctx context.Context, workload *v1alpha1.ChangeWorkload) (changePods []v1alpha1.ChangePod, err error) {
	// 1、获取当前workload下的所有changePod
	// 1、get all changePods under current workload
	changePodList := &v1alpha1.ChangePodList{}
	if err := r.List(ctx, changePodList, client.InNamespace(workload.Namespace), client.MatchingFields{utils.ChangePodFieldChangeWorkloadId: workload.Name}); err != nil {
		return changePods, err
	}
	return changePodList.Items, nil
}

// deleteOldChangeWorkload 删除旧的changeWorkload通过deployment的name
// deleteOldChangeWorkload delete old changeWorkload by deployment name
func (r *ChangeWorkloadReconciler) deleteOldChangeWorkload(ctx context.Context, changeWorkload *v1alpha1.ChangeWorkload) {
	logger := log.FromContext(ctx).WithName("deleteOldChangeWorkload")
	// 根据changeWorkload先获取deployment
	// get deployment by changeWorkload
	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: changeWorkload.Namespace, Name: changeWorkload.Labels[native.DeploymentNameLabel]}, deployment); err != nil {
		logger.Error(err, "get deployment error", utils.LogChangeWorkloadResource, utils.GetResource(changeWorkload))
		return
	}
	// 判断deployment的version是否与changeWorkload的version一致
	// check if deployment version is same as changeWorkload version
	if deployment.Labels[native.AdmissionWebhookVersionLabel] != changeWorkload.Labels[native.AdmissionWebhookVersionLabel] {
		return
	}

	// 获取所有与该 Deployment 相关的 ChangeWorkload 资源
	// get all changeWorkload resource related to deployment
	selector := client.MatchingLabels{native.DeploymentNameLabel: deployment.Name}
	changeWorkloadList := &v1alpha1.ChangeWorkloadList{}
	if err := r.List(ctx, changeWorkloadList, client.InNamespace(changeWorkload.Namespace), selector); err != nil {
		logger.Error(err, "list workload error", utils.LogChangeWorkloadResource, utils.GetResource(changeWorkload))
		return
	}
	// 删除所有与该 Deployment 相关的 ChangeWorkload 资源 (除了当前正在处理的 ChangeWorkload)
	// delete all changeWorkload resource related to deployment (except current changeWorkload)
	for _, workload := range changeWorkloadList.Items {
		if workload.Name != changeWorkload.Name {
			if err := r.Delete(ctx, &workload); err != nil {
				logger.Error(err, "delete workload error", utils.LogChangeWorkloadResource, utils.GetResource(&workload))
			}
		}
	}
}

// filterPodWithoutFinishedLabel 过滤掉没有finished label的pod
// filterPodWithoutFinishedLabel filter pod without finished label
func filterPodWithoutFinishedLabel(items []v1.Pod) []v1.Pod {
	var result []v1.Pod
	for _, item := range items {
		finished := isFinished(&item)
		if finished {
			result = append(result, item)
		}
	}
	return result
}

// filterPodWithoutFinishedOrDefensedLabel	返回是finished但是不是defensed的pod
// filterPodWithoutFinishedOrDefensedLabel return finished but not defensed pod
func filterPodWithoutFinishedOrDefensedLabel(items []v1.Pod) []v1.Pod {
	var result []v1.Pod
	for _, item := range items {
		finished := isFinished(&item)
		defensed := isDefensed(item.Labels)
		if finished && !defensed {
			result = append(result, item)
		}
	}
	return result
}

// isInitStatusChangeWorkload 当前changeWorkload是否是初始化状态
// isInitStatusChangeWorkload is init status
func isInitStatusChangeWorkload(changeWorkload *v1alpha1.ChangeWorkload) bool {
	return changeWorkload.Status.Status == v1alpha1.Init
}

// isFailedStatusChangeWorkload 当前changeWorkload是否是失败状态
// isFailedStatusChangeWorkload is failed status
func isFailedStatusChangeWorkload(changeWorkload *v1alpha1.ChangeWorkload) bool {
	return changeWorkload.Status.Status == v1alpha1.Failed
}

// getPassAndFailedPodsByPostFinishChangePod 通过changePod获取pass和failed的pod
// getPassAndFailedPodsByPostFinishChangePod get pass and failed pod by changePod
func getPassAndFailedPodsByPostFinishChangePod(changePod *v1alpha1.ChangePod) (passPods []v1alpha1.PodSummary, failedPods []v1alpha1.PodSummary) {
	passPods = make([]v1alpha1.PodSummary, utils.NumberZero)
	failedPods = make([]v1alpha1.PodSummary, utils.NumberZero)
	if changePod.Status.PodResults != nil {
		for _, podResult := range changePod.Status.PodResults {
			if podResult.Verdict == utils.ChangePodVerdictPass {
				passPods = append(passPods, podResult)
			} else {
				failedPods = append(failedPods, podResult)
			}
		}
	}
	return passPods, failedPods
}

// buildChangeChangeWorkloadRequest 构建changeWorkload的请求
// buildChangeChangeWorkloadRequest build changeWorkload request
//func buildChangeChangeWorkloadRequest(workload v1alpha1.ChangeWorkload) opsClient.OpsCloudChangeExecOrderSubmitRequest {
//	request := opsClient.OpsCloudChangeExecOrderSubmitRequest{
//		BizExecOrderId:     workload.Spec.ChangeWorkloadId,
//		Platform:           opsClient.Platform,
//		ChangeSceneKey:     utils.ChangeSceneKeyRollingUpdate,
//		ChangeApps:         []string{workload.Spec.AppName},
//		ChangePhases:       []string{utils.ChangePhase},
//		ChangeTitle:        fmt.Sprintf("小程序云发布-RollingUpdate-%s", workload.Spec.AppName),
//		Creator:            utils.DefaultCreator,
//		ChangeParamJson:    "{}",
//		ChangeUrl:          "http://test.cn",
//		TldcTenantCode:     utils.DefaultTldcTenantCode,
//		ChangeContents:     opsClient.DefaultChangeContents,
//		ChangeScenarioCode: utils.ChangeScenarioCode,
//	}
//	return request
//}
