package provider

import (
	"fmt"

	"github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/provider"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/arjunrn/custom-metrics-router/pkg/routes"
)

type FullMetricsProvider interface {
	provider.CustomMetricsProvider
	provider.ExternalMetricsProvider
}

type routedMetricsProvider struct {
	customMetricRoutes *routes.Routes
}

func NewRoutedProvider(customMetricRoutes *routes.Routes) FullMetricsProvider {
	return &routedMetricsProvider{
		customMetricRoutes: customMetricRoutes,
	}
}

func (r routedMetricsProvider) GetMetricByName(name types.NamespacedName, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValue, error) {
	backend, err := r.customMetricRoutes.GetMetricsBackend(info)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics backend: %v", err)
	}
	return backend.GetMetricByName(name, info, metricSelector)
}

func (r routedMetricsProvider) GetMetricBySelector(namespace string, selector labels.Selector, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
	backend, err := r.customMetricRoutes.GetMetricsBackend(info)
	if err != nil {
		return nil, fmt.Errorf("failed to get backend: %v", err)
	}
	return backend.GetMetricBySelector(namespace, selector, info, metricSelector)
}

func (r routedMetricsProvider) ListAllMetrics() []provider.CustomMetricInfo {
	return r.customMetricRoutes.ListAllCustomMetrics()
}

func (r routedMetricsProvider) GetExternalMetric(namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	backend, err := r.customMetricRoutes.GetExternalMetricsBackend(info)
	if err != nil {
		return nil, fmt.Errorf("failed to get backend for external metric %s: %v", info.Metric, err)
	}
	return backend.GetExternalMetric(info.Metric, namespace, metricSelector)
}

func (r routedMetricsProvider) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	return r.customMetricRoutes.ListAllExternalMetrics()
}
