package controllers

import (
	"github.com/traas-stack/altershield-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ handler.EventHandler = &enqueueRequestForPod{}

type enqueueRequestForPod struct {
	reader client.Reader
	scheme *runtime.Scheme
}

func (w *enqueueRequestForPod) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	klog.Infof("create event for obj %v", klog.KObj(evt.Object))
	pod, ok := evt.Object.(*corev1.Pod)
	if !ok {
		return
	}
	w.handleEvent(q, pod)
}

func (w *enqueueRequestForPod) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	klog.Infof("delete event for obj %v", klog.KObj(evt.Object))

	pod, ok := evt.Object.(*corev1.Pod)
	if !ok {
		return
	}
	w.handleEvent(q, pod)
}

func (w *enqueueRequestForPod) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
}

func (w *enqueueRequestForPod) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	klog.Infof("update event for obj %v %v/%v", evt.ObjectNew.GetObjectKind(),
		evt.ObjectNew.GetNamespace(), evt.ObjectNew.GetName())

	pod, ok := evt.ObjectNew.(*corev1.Pod)
	if !ok {
		return
	}
	w.handleEvent(q, pod)
}

func (w *enqueueRequestForPod) handleEvent(q workqueue.RateLimitingInterface, pod *corev1.Pod) {
	klog.Infof("Get for pod %v", klog.KObj(pod))

	workload, err := utils.GetOwnerWorkload(w.reader, pod)
	if err != nil {
		klog.Errorf("unable to get owner workload of pod %v: %v", klog.KObj(pod), err)
		return
	}
	if workload == nil {
		return
	}
	defenseExecutionBrief, err := utils.GetLatestDefenseExecutionBrief(workload)
	if err != nil {
		klog.Errorf("failed to get defense execution brief for workload %v: %v",
			klog.KObj(workload), err)
		return
	}

	if defenseExecutionBrief == nil {
		return
	}
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: pod.Namespace,
		Name:      defenseExecutionBrief.BuildChangeDefenseExecutionName(),
	}})
}
