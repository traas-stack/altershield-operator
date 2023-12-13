package controllers

import (
	"github.com/traas-stack/altershield-operator/pkg/constants"
	"github.com/traas-stack/altershield-operator/pkg/utils"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ handler.EventHandler = &enqueueRequestForWorkload{}

type enqueueRequestForWorkload struct {
	reader client.Reader
	scheme *runtime.Scheme

	kind schema.GroupVersionKind
}

func (w *enqueueRequestForWorkload) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
}

func (w *enqueueRequestForWorkload) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
}

func (w *enqueueRequestForWorkload) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
}

func (w *enqueueRequestForWorkload) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	klog.Infof("update event for obj %v %v/%v", evt.ObjectNew.GetObjectKind(),
		evt.ObjectNew.GetNamespace(), evt.ObjectNew.GetName())
	w.handleEvent(q, evt.ObjectNew)
}

func (w *enqueueRequestForWorkload) handleEvent(q workqueue.RateLimitingInterface, obj client.Object) {
	changeDefense, err := utils.FetchChangeDefense(w.reader, w.kind, obj)
	if err != nil {
		klog.Errorf("unable to get ChangeDefense for workload %v %v/%v: %v",
			obj.GetObjectKind().GroupVersionKind().String(), obj.GetNamespace(), obj.GetName(), err)
		return
	}
	if changeDefense != nil {
		klog.Infof("workload %v %v/%v and reconcile ChangeDefense %v/%v",
			obj.GetObjectKind().GroupVersionKind().String(), obj.GetNamespace(), obj.GetName(),
			changeDefense.Namespace, changeDefense.Name)
		nsn := types.NamespacedName{
			Namespace: changeDefense.GetNamespace(),
			Name: changeDefense.GetName(),
		}
		q.Add(reconcile.Request{NamespacedName: nsn})
	}
}

type enqueueRequestForChangeDefenseExecution struct {
	reader client.Reader
	scheme *runtime.Scheme
}

func (w *enqueueRequestForChangeDefenseExecution) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
}

func (w *enqueueRequestForChangeDefenseExecution) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
}

func (w *enqueueRequestForChangeDefenseExecution) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
}

func (w *enqueueRequestForChangeDefenseExecution) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	klog.Infof("update event for obj %v %v/%v", evt.ObjectNew.GetObjectKind(),
		evt.ObjectNew.GetNamespace(), evt.ObjectNew.GetName())
	w.handleEvent(q, evt.ObjectNew)
}

func (w *enqueueRequestForChangeDefenseExecution) handleEvent(q workqueue.RateLimitingInterface, obj client.Object) {
	changeDefenseName := utils.GetLabel(obj, constants.LabelChangeDefense)
	if changeDefenseName == "" {
		return
	}
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      changeDefenseName,
	}})
}