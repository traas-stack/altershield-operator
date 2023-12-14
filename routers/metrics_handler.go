package routers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/traas-stack/altershield-operator/pkg/metric"
	metricprovider "github.com/traas-stack/altershield-operator/pkg/metric/provider"
	utils2 "github.com/traas-stack/altershield-operator/pkg/utils"
	k8sautoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	"net/http"
	"time"
)

type MetricHandler struct {
	metricProvider metricprovider.Interface
}

func NewMetricHandler(metricProvider metricprovider.Interface) *MetricHandler {
	return &MetricHandler{
		metricProvider: metricProvider,
	}
}

func (m *MetricHandler) Query(c *gin.Context) {
	request := &Query{}
	if err := c.ShouldBindJSON(request); err != nil {
		klog.Errorf("failed to unmarshal metric query request: %v", err)
		c.JSON(http.StatusBadRequest, GetCommonCallbackErr(err))
		return
	}
	klog.Infof("get metric query request: %v", func () string {
		raw, _ := json.Marshal(request)
		return string(raw)
	}())

	metricQuery, err := convertToInternalQuery(request)
	if err != nil {
		c.JSON(http.StatusBadRequest, GetCommonCallbackErr(
			fmt.Errorf("failed to convert to internal query: %v", err)))
		return
	}

	start := time.UnixMilli(request.Start)
	end := time.UnixMilli(request.End)
	step := time.Duration(request.Duration) * time.Millisecond
	series, err := m.metricProvider.Query(context.TODO(), metricQuery, start, end, step)
	if err != nil {
		c.JSON(http.StatusBadRequest, GetCommonCallbackErr(fmt.Errorf("failed to query metrics: %v", err)))
		return
	}

	c.JSON(http.StatusOK, series)
}

func convertToInternalQuery(in *Query) (out *metric.Query, err error) {
	out = &metric.Query{}

	switch in.Type {
	case QueryTypePodMetric:
		out.Type = metric.ObjectQueryType
		inputPodMetricQuery := in.PodMetricQuery
		if inputPodMetricQuery.Metric == nil || inputPodMetricQuery.Metric.Name == "" {
			return nil, fmt.Errorf("metric not specified")
		}
		var metricLs *metav1.LabelSelector
		if inputPodMetricQuery.Metric.Selector != "" {
			metricLs, err = metav1.ParseToLabelSelector(inputPodMetricQuery.Metric.Selector)
			if err != nil {
				return nil, fmt.Errorf("failed to parse metric selector %v: %v",
					inputPodMetricQuery.Metric.Selector, err)
			}
		}
		out.Object = &metric.ObjectQuery{
			GroupKind: schema.GroupKind{
				Group: utils2.PodKind.Group,
				Kind:  utils2.PodKind.Kind,
			},
			Namespace: inputPodMetricQuery.Namespace,
			Name:      inputPodMetricQuery.Name,
			Metric: k8sautoscalingv2.MetricIdentifier{
				Name:     inputPodMetricQuery.Metric.Name,
				Selector: metricLs,
			},
		}
	case QueryTypeWorkloadExternal:
		out.Type = metric.WorkloadExternalQueryType
		inputWorkloadExternalQuery := in.WorkloadExternalQuery
		gvk, err := parseWorkloadKind(inputWorkloadExternalQuery.Kind)
		if err != nil {
			return nil, err
		}
		if inputWorkloadExternalQuery.Metric == nil || inputWorkloadExternalQuery.Metric.Name == "" {
			return nil, fmt.Errorf("metric not specified")
		}
		var metricLs *metav1.LabelSelector
		if inputWorkloadExternalQuery.Metric.Selector != "" {
			metricLs, err = metav1.ParseToLabelSelector(inputWorkloadExternalQuery.Metric.Selector)
			if err != nil {
				return nil, fmt.Errorf("failed to parse metric selector %v: %v",
					inputWorkloadExternalQuery.Metric.Selector, err)
			}
		}
		out.WorkloadExternal = &metric.WorkloadExternalQuery{
			GroupKind: schema.GroupKind{
				Group: gvk.Group,
				Kind:  gvk.Kind,
			},
			Namespace: inputWorkloadExternalQuery.Namespace,
			Name:      inputWorkloadExternalQuery.Name,
			Metric: k8sautoscalingv2.MetricIdentifier{
				Name:     inputWorkloadExternalQuery.Metric.Name,
				Selector: metricLs,
			},
		}
	default:
		return nil, fmt.Errorf("unsupported query type: %v", in.Type)
	}
	return out, nil
}

type QueryType string

const (
	QueryTypePodMetric QueryType = "podMetric"
	QueryTypeWorkloadExternal  QueryType = "workloadExternal"
)

type MetricIdentifier struct {
	Name string `json:"name"`
	Selector string `json:"selector"`
}

type PodMetricQuery struct {
	Namespace string `json:"namespace"`
	Name string `json:"name"`
	Metric *MetricIdentifier `json:"metric"`
}

type WorkloadKind string

const (
	WorkloadKindDeployment = "Deployment"
	WorkloadKindStatefulSet = "StatefulSet"
)

func parseWorkloadKind(kind WorkloadKind) (schema.GroupVersionKind, error) {
	if kind == WorkloadKindDeployment {
		return utils2.DeploymentKind, nil
	} else if kind == WorkloadKindStatefulSet{
		return utils2.StatefulSetKind, nil
	}
	return schema.GroupVersionKind{}, fmt.Errorf("unsupported kind: %v", kind)
}

type WorkloadExternalQuery struct {
	Kind WorkloadKind `json:"kind"`
	Namespace string `json:"namespace"`
	Name string `json:"name"`
	Metric *MetricIdentifier `json:"metric"`
}

type Query struct {
	Type QueryType                               `json:"type"`
	PodMetricQuery *PodMetricQuery             `json:"podMetricQuery"`
	WorkloadExternalQuery *WorkloadExternalQuery `json:"workloadExternalQuery"`

	Start int64 `json:"start"`
	End int64 `json:"end"`
	Duration int64 `json:"duration"`
}
