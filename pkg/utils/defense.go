package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/traas-stack/altershield-operator/apis/app.ops.cloud.alipay.com/v1alpha1"
	"github.com/traas-stack/altershield-operator/pkg/constants"
	types "github.com/traas-stack/altershield-operator/pkg/types"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"k8s.io/utils/integer"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

func GetOwnerWorkload(r client.Reader, pod *v1.Pod) (client.Object, error) {
	if pod == nil {
		return nil, nil
	}
	var curr client.Object = pod
	var topLevelOwner client.Object
	for ;; {
		ownerRef := metav1.GetControllerOf(curr)
		if ownerRef == nil {
			break
		}
		ownerGVK := schema.FromAPIVersionAndKind(ownerRef.APIVersion, ownerRef.Kind)
		if !IsSupportedKind(ownerGVK) {
			return nil, nil
		}
		ownerKey := apitypes.NamespacedName{Namespace: pod.GetNamespace(), Name: ownerRef.Name}
		ownerObj := GetEmptyWorkloadObject(ownerGVK)
		err := r.Get(context.TODO(), ownerKey, ownerObj)
		if err != nil {
			if client.IgnoreNotFound(err) != nil {
				return nil, err
			}
			break
		}
		curr = ownerObj
		if len(curr.GetAnnotations()[constants.AnnotationLatestDefenseExec]) > 0 {
			topLevelOwner = curr
		}
	}
	return topLevelOwner, nil
}

// GetEmptyWorkloadObject return specific object based on the given gvk
func GetEmptyWorkloadObject(gvk schema.GroupVersionKind) client.Object {
	switch gvk {
	case ReplicaSetKind:
		return &apps.ReplicaSet{}
	case DeploymentKind:
		return &apps.Deployment{}
	case StatefulSetKind:
		return &apps.StatefulSet{}
	default:
		return nil
	}
}

func IsPodOwner(r client.Reader, workload client.Object, pod *v1.Pod) (bool, error) {
	podOwnerWorkload, err := GetOwnerWorkload(r, pod)
	if err != nil {
		return false, err
	}
	if podOwnerWorkload == nil {
		return false, nil
	}
	klog.Infof("pod %v owner workload is %v", klog.KObj(pod), klog.KObj(podOwnerWorkload))
	return podOwnerWorkload.GetUID() == workload.GetUID(), nil
}

func ListOwnedPods(r client.Reader, workload client.Object) (pods []*v1.Pod, err error){
	var podsSelector labels.Selector
	switch workloadObj := workload.(type) {
	case *apps.Deployment:
		podsSelector, err = metav1.LabelSelectorAsSelector(workloadObj.Spec.Selector)
	case *apps.StatefulSet:
		podsSelector, err = metav1.LabelSelectorAsSelector(workloadObj.Spec.Selector)
	default:
		return nil, fmt.Errorf("unsupported workload type %v", workload.GetObjectKind().GroupVersionKind().String())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pods label selector of workload %v/%v",
			workload.GetNamespace(), workload.GetName())
	}
	if podsSelector.Empty() {
		return nil, fmt.Errorf("do not support workload %v/%v with empty label selector",
			workload.GetNamespace(), workload.GetName())
	}

	totalPodList := &v1.PodList{}
	if err = r.List(context.TODO(),
		totalPodList,
		&client.ListOptions{
			LabelSelector: podsSelector,
			Namespace: workload.GetNamespace(),
		},
	); err != nil {
		return nil, err
	}
	pods = make([]*v1.Pod, 0, len(totalPodList.Items))
	for i := range totalPodList.Items {
		pod := &totalPodList.Items[i]
		if pod.Status.Phase == v1.PodFailed || pod.Status.Phase == v1.PodSucceeded {
			continue
		}
		owned, err := IsPodOwner(r, workload, pod)
		if err != nil {
			return nil, err
		} else if !owned {
			continue
		}
		pods = append(pods, pod)
	}
	return pods, nil
}

func GetLatestTemplateRevisionOfWorkload(r client.Reader, workload client.Object) (string, error) {
	switch workloadObj := workload.(type) {
	case *apps.Deployment:
		// List all ReplicaSets belonging to the deployment
		rsList := &apps.ReplicaSetList{}
		rsSelector, err := metav1.LabelSelectorAsSelector(workloadObj.Spec.Selector)
		if err != nil {
			return "", err
		}
		if err = r.List(context.TODO(),
			rsList,
			&client.ListOptions{
				LabelSelector: rsSelector,
			}); err != nil {
			return "", err
		}
		var latestRS *apps.ReplicaSet
		var latestCreationTime time.Time
		for i := range rsList.Items {
			curr := &rsList.Items[i]
			if curr.CreationTimestamp.After(latestCreationTime) {
				latestCreationTime = curr.CreationTimestamp.Time
				latestRS = curr
			}
		}
		if latestRS == nil {
			return "", fmt.Errorf("not found replicasets for workload %v/%v", workload.GetNamespace(), workload.GetName())
		}
		revision := latestRS.Labels["pod-template-hash"]
		if revision == "" {
			return "", fmt.Errorf("not found revision recorded in latest replicaset %v/%v", latestRS.Namespace, latestRS.Name)
		}
		return revision, nil
	case *apps.StatefulSet:
		return workloadObj.Status.UpdateRevision, nil
	default:
		return "", fmt.Errorf("unsupported workload type %v", workload.GetObjectKind().GroupVersionKind().String())
	}
}

// IsConsistentWithRevision return true iff pod is match the revision
func IsConsistentWithRevision(pod *v1.Pod, revision string) bool {
	if pod.Labels[apps.DefaultDeploymentUniqueLabelKey] != "" &&
		strings.HasSuffix(revision, pod.Labels[apps.DefaultDeploymentUniqueLabelKey]) {
		return true
	}

	if pod.Labels[apps.ControllerRevisionHashLabelKey] != "" &&
		strings.HasSuffix(revision, pod.Labels[apps.ControllerRevisionHashLabelKey]) {
		return true
	}
	return false
}

func GetReadyPodsOfLatestRevision(r client.Reader, workload client.Object) ([]*v1.Pod, error) {
	return getPodsOfLatestRevisionInner(r, workload, true)
}

func GetPodsOfLatestRevision(r client.Reader, workload client.Object) ([]*v1.Pod, error) {
	return getPodsOfLatestRevisionInner(r, workload, false)
}

func getPodsOfLatestRevisionInner(r client.Reader, workload client.Object, checkReady bool) ([]*v1.Pod, error) {
	pods, err := ListOwnedPods(r, workload)
	if err != nil {
		return nil, err
	}

	workloadLatestRevision, err := GetLatestTemplateRevisionOfWorkload(r, workload)
	klog.Infof("workload %v latest template revision is %v", klog.KObj(workload), workloadLatestRevision)
	if err != nil {
		return nil, err
	}
	latestPods := make([]*v1.Pod, 0)
	for _, pod := range pods {
		if IsConsistentWithRevision(pod, workloadLatestRevision) &&
			pod.DeletionTimestamp.IsZero() &&
			(!checkReady || IsPodReady(pod)) {
			latestPods = append(latestPods, pod)
		}
	}
	return latestPods, nil
}

// IsPodReady returns true if a pod is ready; false otherwise.
func IsPodReady(pod *v1.Pod) bool {
	return IsPodReadyConditionTrue(pod.Status)
}

// IsPodReadyConditionTrue returns true if a pod is ready; false otherwise.
func IsPodReadyConditionTrue(status v1.PodStatus) bool {
	condition := GetPodReadyCondition(status)
	return condition != nil && condition.Status == v1.ConditionTrue
}

// GetPodReadyCondition extracts the pod ready condition from the given status and returns that.
// Returns nil if the condition is not present.
func GetPodReadyCondition(status v1.PodStatus) *v1.PodCondition {
	_, condition := GetPodCondition(&status, v1.PodReady)
	return condition
}

// GetPodCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func GetPodCondition(status *v1.PodStatus, conditionType v1.PodConditionType) (int, *v1.PodCondition) {
	if status == nil {
		return -1, nil
	}
	return GetPodConditionFromList(status.Conditions, conditionType)
}

// GetPodConditionFromList extracts the provided condition from the given list of condition and
// returns the index of the condition and the condition. Returns -1 and nil if the condition is not present.
func GetPodConditionFromList(conditions []v1.PodCondition, conditionType v1.PodConditionType) (int, *v1.PodCondition) {
	if conditions == nil {
		return -1, nil
	}
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return i, &conditions[i]
		}
	}
	return -1, nil
}

func GetLatestDefenseExecutionBrief(workload client.Object) (*types.DefenseExecutionBrief, error) {
	latestExecRaw := GetAnnotation(workload, constants.AnnotationLatestDefenseExec)
	if latestExecRaw == "" {
		return nil, nil
	}
	var latestExec types.DefenseExecutionBrief
	if err := json.Unmarshal([]byte(latestExecRaw), &latestExec); err != nil {
		return nil, fmt.Errorf("failed to parse latest defense execution for target %v: %v",
			klog.KObj(workload), err)
	}
	return &latestExec, nil
}

// NewRSReplicasLimit return a limited replicas of new RS calculated via partition.
func GetBatchReplicasBound(partition intstr.IntOrString, replicas int) int {
	replicaLimit, _ := intstr.GetScaledValueFromIntOrPercent(&partition, replicas, true)
	replicaLimit = integer.IntMax(integer.IntMin(replicaLimit, replicas), 0)
	if replicas > 1 && partition.Type == intstr.String && partition.String() != "100%" {
		replicaLimit = integer.IntMin(replicaLimit, replicas-1)
	}
	return replicaLimit
}

func GetChangeDefenseExecutionByID(c client.Reader, executionID string) (*v1alpha1.ChangeDefenseExecution, error) {
	changeDefenseExecutionList := &v1alpha1.ChangeDefenseExecutionList{}
	if err := c.List(context.TODO(), changeDefenseExecutionList,
		client.MatchingFields{"spec.id": executionID}); err != nil {
		return nil, err
	}

	klog.Infof("GETDEFENSEEXECBYID: %v", changeDefenseExecutionList.Items)

	if len(changeDefenseExecutionList.Items) == 0 {
		return nil, nil
	}

	cde := &changeDefenseExecutionList.Items[0]
	cdeClone := cde.DeepCopy()
	return cdeClone, nil
}

func UpdateChangeDefenseStatus(apiClient client.Client, ctx context.Context,
	cd *v1alpha1.ChangeDefense, newStatus *v1alpha1.ChangeDefenseStatus) error {
	var err error
	defer func() {
		if err != nil {
			klog.Errorf("Failed to update ChangeDefense status %v: %v", klog.KObj(cd), err)
		}
	}()

	oldStatusBytes, _ := json.Marshal(cd.Status)
	newStatusBytes, _ := json.Marshal(newStatus)
	klog.Infof("STATUS CD UPDATE!!! cd %v, old status %v, new status %v",
		klog.KObj(cd), string(oldStatusBytes), string(newStatusBytes))

	// do not retry
	objectKey := client.ObjectKeyFromObject(cd)
	if !reflect.DeepEqual(cd.Status, *newStatus) {
		err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			clone := &v1alpha1.ChangeDefense{}
			getErr := apiClient.Get(ctx, objectKey, clone)
			if getErr != nil {
				return getErr
			}
			clone.Status = *newStatus.DeepCopy()
			return apiClient.Status().Update(ctx, clone)
		})
	}
	return err
}

func UpdateChangeDefenseExecutionStatus(
	apiClient client.Client, ctx context.Context,
	cde *v1alpha1.ChangeDefenseExecution, newStatus *v1alpha1.ChangeDefenseExecutionStatus) error {
	var err error
	defer func() {
		if err != nil {
			klog.Errorf("Failed to update ChangeDefenseExecution status %v: %v", klog.KObj(cde), err)
		}
	}()

	oldStatusBytes, _ := json.Marshal(cde.Status)
	newStatusBytes, _ := json.Marshal(newStatus)
	klog.Infof("STATUS UPDATE!!! cde %v, old status %v, new status %v",
		klog.KObj(cde), string(oldStatusBytes), string(newStatusBytes))

	// do not retry
	objectKey := client.ObjectKeyFromObject(cde)
	if !reflect.DeepEqual(cde.Status, *newStatus) {
		err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			clone := &v1alpha1.ChangeDefenseExecution{}
			getErr := apiClient.Get(ctx, objectKey, clone)
			if getErr != nil {
				return getErr
			}
			clone.Status = *newStatus.DeepCopy()
			return apiClient.Status().Update(ctx, clone)
		})
	}
	return err
}

func GetChangeDefenseByExecutionID(c client.Reader, executionID string) (*v1alpha1.ChangeDefense, error) {
	changeDefenseList := &v1alpha1.ChangeDefenseList{}
	if err := c.List(context.TODO(), changeDefenseList,
		client.MatchingFields{"status.currentExecutionID": executionID}); err != nil {
		return nil, err
	}

	klog.Infof("GETDEFENSEBYEXECID: %v", changeDefenseList.Items)

	if len(changeDefenseList.Items) == 0 {
		return nil, nil
	}

	cd := &changeDefenseList.Items[0]
	cdClone := cd.DeepCopy()
	return cdClone, nil
}

func FetchChangeDefense(c client.Reader, gvk schema.GroupVersionKind, obj client.Object) (*v1alpha1.ChangeDefense, error) {
	changeDefenseList := &v1alpha1.ChangeDefenseList{}
	if err := c.List(context.TODO(), changeDefenseList,
		&client.ListOptions{Namespace: obj.GetNamespace()}); err != nil {
		klog.Errorf("WorkloadHandler List ChangeDefense failed: %s", err.Error())
		return nil, err
	}
	for i := range changeDefenseList.Items {
		changeDefense := &changeDefenseList.Items[i]
		klog.Infof("CHANGE DEFENSE!!! %v", changeDefense.Name)
		if !changeDefense.DeletionTimestamp.IsZero() || changeDefense.Spec.Target.ObjectRef == nil {
			continue
		}
		ref := changeDefense.Spec.Target.ObjectRef
		gv, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			klog.Warningf("ParseGroupVersion changeDefense(%s/%s) ref failed: %s", changeDefense.Namespace, changeDefense.Name, err.Error())
			continue
		}
		if gvk.Group == gv.Group && gvk.Kind == ref.Kind && obj.GetName() == ref.Name {
			return changeDefense.DeepCopy(), nil
		}
	}
	return nil, nil
}