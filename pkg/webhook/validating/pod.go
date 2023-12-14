package validating

import (
	"context"
	"fmt"
	"github.com/traas-stack/altershield-operator/apis/app.ops.cloud.alipay.com/v1alpha1"
	utils "github.com/traas-stack/altershield-operator/pkg/utils"
	webhookutils "github.com/traas-stack/altershield-operator/pkg/webhook/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var podGVK = gvkConverter(corev1.SchemeGroupVersion.WithKind("Pod"))

func init() {
	deleteHandlers[podGVK] = PodDeleteValidation
}

func PodDeleteValidation(ctx context.Context, req *admission.Request, handler *Handler) admission.Response {
	return validate(ctx, req, handler, true)
}

func validate(ctx context.Context, req *admission.Request, handler *Handler, allowLatest bool) admission.Response {
	pod := &corev1.Pod{}
	err := handler.Decoder.DecodeRaw(req.OldObject, pod)
	if err != nil {
		return webhookutils.AdmissionErroredWithLog(http.StatusBadRequest, err)
	}

	workloadObj, err := utils.GetOwnerWorkload(handler.Client, pod)
	if err != nil {
		return webhookutils.AdmissionErroredWithLog(http.StatusBadRequest, err)
	}
	if workloadObj == nil {
		return admission.Allowed("no owner workload for change defense, validation pass")
	}
	latestRevision, err := utils.GetLatestTemplateRevisionOfWorkload(handler.Client, workloadObj)
	if err != nil {
		return webhookutils.AdmissionErroredWithLog(http.StatusBadRequest, err)
	}
	if utils.IsConsistentWithRevision(pod, latestRevision) == allowLatest {
		return admission.Allowed("validation pass")
	}

	latestDefenseExecModel, err := utils.GetLatestDefenseExecutionBrief(workloadObj)
	if err != nil {
		return webhookutils.AdmissionErroredWithLog(http.StatusBadRequest, err)
	}
	if latestDefenseExecModel == nil {
		return admission.Allowed("validation pass")
	}

	// get ChangeDefenseExecution object
	latestDefenseExec := &v1alpha1.ChangeDefenseExecution{}
	if err = handler.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: workloadObj.GetNamespace(),
		Name:      latestDefenseExecModel.BuildChangeDefenseExecutionName(),
	}, latestDefenseExec); err != nil {
		if errors.IsNotFound(err) {
			return admission.Allowed("validation pass")
		}
		return webhookutils.AdmissionErroredWithLog(http.StatusBadRequest, err)
	}

	defensePhase := latestDefenseExec.Status.DefenseStatus.Phase
	if webhookutils.InDefenseProgress(defensePhase) {
		return webhookutils.AdmissionDeniedWithLog(fmt.Sprintf("current defense execution %v phase is %v",
			latestDefenseExecModel.BuildChangeDefenseExecutionName(), defensePhase))
	}
	return admission.Allowed("validation pass")
}
