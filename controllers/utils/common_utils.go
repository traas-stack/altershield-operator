package utils

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/apis/app.ops.cloud.alipay.com/v1alpha1"
)

var (
	ConfigIsBatchChannel      = make(chan bool)
	ConfigBatchCountChannel   = make(chan int)
	ConfigIsBlockingUpChannel = make(chan bool)
	Env                       = EnvCache{cache: map[string]string{}}
)

func GetNowTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func CombineString(s1 string, s2 string) string {
	return s1 + MetaMark + s2
}

// RemoveDuplicatePod pod去重，并且去除已经被删除的pod
// RemoveDuplicatePod pod deduplication and remove pods that have been deleted
func RemoveDuplicatePod(arr []v1alpha1.PodSummary, podMap map[string]corev1.Pod) []v1alpha1.PodSummary {
	var uniqueArr []v1alpha1.PodSummary
	for _, v := range arr {
		// 不在podMap中出现，如果不在podMap中出现，说明该pod已经被删除，不需要再拼接
		// Not in podMap, if not in podMap, it means that the pod has been deleted and does not need to be spliced
		if _, ok := podMap[v.Pod]; !ok {
			continue
		}
		// 判断v是否在uniqueArr中出现过
		// Determine whether v has appeared in uniqueArr
		needAppend := true
		for _, u := range uniqueArr {
			if v.Pod == u.Pod {
				needAppend = false
				break
			}
		}
		// 如果需要被添加，将其添加到uniqueArr中
		// if need append, add it to uniqueArr
		if needAppend {
			uniqueArr = append(uniqueArr, v)
		}
	}
	return uniqueArr
}

// GetPodNameSpaceNameFromPod 从pod中获取NamespacedName
// GetPodNameSpaceNameFromPod get NamespacedName from pod
func GetPodNameSpaceNameFromPod(pod corev1.Pod) types.NamespacedName {
	return types.NamespacedName{
		Namespace: pod.GetNamespace(),
		Name:      ResourceTypePod + MetaMark + pod.GetName(),
	}
}

// GetChangePodNameSpaceNameFromChangePod 从changePod中获取NamespacedName
// GetChangePodNameSpaceNameFromChangePod get NamespacedName from changePod
func GetChangePodNameSpaceNameFromChangePod(changePod v1alpha1.ChangePod) types.NamespacedName {
	return types.NamespacedName{
		Namespace: changePod.GetNamespace(),
		Name:      ResourceTypeChangePod + MetaMark + changePod.GetName(),
	}
}

// GetPodRequestFromPodNameSpaceName 从pod类型的NamespacedName中获取PodRequest
// GetPodRequestFromPodNameSpaceName get PodRequest from NamespacedName of pod type
func GetPodRequestFromPodNameSpaceName(namespacedName types.NamespacedName) ctrl.Request {
	nameParts := strings.Split(namespacedName.Name, MetaMark)
	podName := strings.Join(nameParts[1:], MetaMark)
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespacedName.Namespace,
			Name:      podName,
		},
	}
}

// GetResourceTypeFromNamespaceName 从ChangeWorkloadRequest中获取资源类型
// GetResourceTypeFromNamespaceName get resource type from ChangeWorkloadRequest
func GetResourceTypeFromNamespaceName(request types.NamespacedName) string {
	return strings.Split(request.Name, MetaMark)[NumberZero]
}

// IsPodResourceType 判断资源类型是否为pod
// IsPodResourceType determine whether the resource type is pod
func IsPodResourceType(resourceType string) bool {
	return resourceType == ResourceTypePod
}

// IsChangePodResourceType 判断资源类型是否为pod
// IsChangePodResourceType determine whether the resource type is pod
func IsChangePodResourceType(resourceType string) bool {
	return resourceType == ResourceTypeChangePod
}

// Percent 数值，百分比，返回值为int类型表示百分比后的数值
// Percent value, percentage, the return value is of type int to indicate the value after the percentage
func Percent(value int, percent int) int {
	if percent == NumberZero || value == NumberZero {
		return NumberOne
	}
	if percent > NumberOneHundred {
		return value
	}
	result := value * percent / NumberOneHundred
	if result < NumberZero {
		return NumberOne
	}
	return result
}

// GetCommonCallbackErr 返回错误信息
// GetCommonCallbackErr return error message
func GetCommonCallbackErr(err error) gin.H {
	return gin.H{
		"code":     50001,
		"message":  "服务器内部错误",
		"detail":   "err:" + err.Error(),
		"solution": "请联系系统管理员解决问题",
	}
}

// GetCommonCallbackSuccess 返回成功信息
// GetCommonCallbackSuccess return success message
func GetCommonCallbackSuccess() gin.H {
	return gin.H{
		"code":    200,
		"message": "服务请求成功",
	}
}

// IsEmpty 数组判空校验
// IsEmpty array null check
func IsEmpty(arr interface{}) bool {
	if arr == nil {
		return true
	}
	switch reflect.TypeOf(arr).Kind() {
	case reflect.Slice, reflect.Array:
		return reflect.ValueOf(arr).Len() == NumberZero
	default:
		return true
	}
}

// IsNotEmpty 数组判空校验
// IsNotEmpty array null check
func IsNotEmpty(arr interface{}) bool {
	if arr == nil {
		return false
	}
	switch reflect.TypeOf(arr).Kind() {
	case reflect.Slice, reflect.Array:
		return reflect.ValueOf(arr).Len() != NumberZero
	default:
		return false
	}
}

// GetResource 获取资源名称
// GetResource get resource name
func GetResource(obj runtime.Object) string {
	switch o := obj.(type) {
	case *corev1.Pod:
		return fmt.Sprintf("%s/%s", o.Namespace, o.Name)
	case *appsv1.Deployment:
		return fmt.Sprintf("%s/%s", o.Namespace, o.Name)
	case *appsv1.ReplicaSet:
		return fmt.Sprintf("%s/%s", o.Namespace, o.Name)
	case *v1alpha1.ChangeWorkload:
		return fmt.Sprintf("%s/%s", o.Namespace, o.Name)
	case *v1alpha1.ChangePod:
		return fmt.Sprintf("%s/%s", o.Namespace, o.Name)
	default:
		return ""
	}
}

// IsPodSummarySliceEqual 验证PodSummary数组是否相等
// IsPodSummarySliceEqual verify whether the PodSummary array is equal
func IsPodSummarySliceEqual(a, b []v1alpha1.PodSummary) bool {
	if len(a) != len(b) {
		return false
	}

	if len(a) == 0 {
		return true
	}

	aMap := make(map[string]v1alpha1.PodSummary)
	for _, p := range a {
		aMap[p.Pod] = p
	}

	bMap := make(map[string]v1alpha1.PodSummary)
	for _, p := range b {
		bMap[p.Pod] = p
	}

	for k, v := range aMap {
		if bVal, ok := bMap[k]; !ok || bVal != v {
			return false
		}
	}

	return true
}
