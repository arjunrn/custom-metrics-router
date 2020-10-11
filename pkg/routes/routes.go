package routes

import (
	"fmt"
	"sync"
	"time"

	"github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/provider"
	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/arjunrn/custom-metrics-router/pkg/metricsclient"
)

type serviceKey struct {
	Name      string
	Namespace string
}

type ServiceProperties struct {
	priority            int
	customMetricInfos   map[provider.CustomMetricInfo]struct{}
	externalMetricInfos map[provider.ExternalMetricInfo]struct{}
	client              *metricsclient.Client
}

type Routes struct {
	lock              sync.RWMutex
	serviceProperties map[serviceKey]ServiceProperties
	customMetrics     map[provider.CustomMetricInfo]*MetricServiceList
	externalMetrics   map[provider.ExternalMetricInfo]*MetricServiceList
	mapper            meta.RESTMapper
}

func New(mapper meta.RESTMapper) *Routes {
	return &Routes{
		serviceProperties: make(map[serviceKey]ServiceProperties),
		customMetrics:     make(map[provider.CustomMetricInfo]*MetricServiceList),
		externalMetrics:   make(map[provider.ExternalMetricInfo]*MetricServiceList),
		mapper:            mapper,
	}
}

// TODO anaik: Refactor so that the old client can be reused when nothing changes.
func (r *Routes) AddService(name, namespace string, port int32, priority int, insecureTLSSkipVerify bool, creationTimestamp time.Time, customMetrics, externalMetrics bool) error {
	client, err := metricsclient.NewClient(insecureTLSSkipVerify, name, namespace, port, r.mapper)
	if err != nil {
		return err
	}

	r.lock.Lock()
	defer r.lock.Unlock()
	key := serviceKey{Name: name, Namespace: namespace}
	customMetricInfos := make(map[provider.CustomMetricInfo]struct{})
	if customMetrics {
		customMetricInfos, err = client.ListCustomMetricInfos()
		if err != nil {
			return fmt.Errorf("failed to list custom metric api resources: %v", err)
		}
	}
	if serviceProperties, ok := r.serviceProperties[key]; ok {
		oldMetricInfos := getOldCustomMetricInfos(serviceProperties.customMetricInfos, customMetricInfos)
		for _, outdated := range oldMetricInfos {
			r.customMetrics[outdated].RemoveService(name, namespace)
		}
	}
	for mInfo := range customMetricInfos {
		var ok bool
		if _, ok = r.customMetrics[mInfo]; !ok {
			r.customMetrics[mInfo] = NewMetricServiceList()
		}
		serviceList := r.customMetrics[mInfo]
		serviceList.AddService(name, namespace, creationTimestamp, priority)
	}

	externalMetricInfos := make(map[provider.ExternalMetricInfo]struct{})
	if externalMetrics {
		externalMetricInfos, err = client.ListExternalMetrics()
		if err != nil {
			return fmt.Errorf("failed to list external metric api resources: %v", err)
		}
	}
	if serviceProperties, ok := r.serviceProperties[key]; ok {
		oldMetricInfos := getOldExternalMetricInfos(serviceProperties.externalMetricInfos, externalMetricInfos)
		for _, outdated := range oldMetricInfos {
			r.externalMetrics[outdated].RemoveService(name, namespace)
		}
	}
	for mInfo := range externalMetricInfos {
		var ok bool
		if _, ok = r.externalMetrics[mInfo]; !ok {
			r.externalMetrics[mInfo] = NewMetricServiceList()
		}
		serviceList := r.externalMetrics[mInfo]
		serviceList.AddService(name, namespace, creationTimestamp, priority)
	}
	r.serviceProperties[key] = ServiceProperties{
		priority:            priority,
		client:              client,
		customMetricInfos:   customMetricInfos,
		externalMetricInfos: externalMetricInfos,
	}
	return nil
}

func getOldCustomMetricInfos(old map[provider.CustomMetricInfo]struct{}, new map[provider.CustomMetricInfo]struct{}) []provider.CustomMetricInfo {
	var outdated []provider.CustomMetricInfo
	for info := range old {
		if _, ok := new[info]; !ok {
			outdated = append(outdated, info)
		}
	}
	return outdated
}

func getOldExternalMetricInfos(old map[provider.ExternalMetricInfo]struct{}, new map[provider.ExternalMetricInfo]struct{}) []provider.ExternalMetricInfo {
	var outdated []provider.ExternalMetricInfo
	for info := range old {
		if _, ok := new[info]; !ok {
			outdated = append(outdated, info)
		}
	}
	return outdated
}

func (r *Routes) RemoveService(name, namespace string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	key := serviceKey{Name: name, Namespace: namespace}
	for k, v := range r.customMetrics {
		empty := v.RemoveService(name, namespace)
		if empty {
			delete(r.customMetrics, k)
		}
	}
	delete(r.serviceProperties, key)
}

func (r *Routes) GetMetricsBackend(info provider.CustomMetricInfo) (*metricsclient.Client, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	var services *MetricServiceList
	var metricsService ServiceProperties
	var ok bool
	if services, ok = r.customMetrics[info]; !ok {
		return nil, fmt.Errorf("metric %s is not provided by any metrics backend", info.Metric)
	}
	service, err := services.GetBestMetricService()
	if err != nil {
		return nil, fmt.Errorf("not backend for metric: %v", info.Metric)
	}
	if metricsService, ok = r.serviceProperties[serviceKey{
		Name:      service.Name,
		Namespace: service.Namespace,
	}]; !ok {
		return nil, fmt.Errorf("properties for metric service %s/%s is missing", service.Namespace, service.Name)
	}
	return metricsService.client, nil
}
func (r *Routes) GetExternalMetricsBackend(info provider.ExternalMetricInfo) (*metricsclient.Client, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	var services *MetricServiceList
	var metricsService ServiceProperties
	var ok bool
	if services, ok = r.externalMetrics[info]; !ok {
		return nil, fmt.Errorf("metric %s is not provided by any metrics backend", info.Metric)
	}
	service, err := services.GetBestMetricService()
	if err != nil {
		return nil, fmt.Errorf("not backend for metric: %v", info.Metric)
	}
	if metricsService, ok = r.serviceProperties[serviceKey{
		Name:      service.Name,
		Namespace: service.Namespace,
	}]; !ok {
		return nil, fmt.Errorf("properties for metric service %s/%s is missing", service.Namespace, service.Name)
	}
	return metricsService.client, nil
}

func (r *Routes) ListAllCustomMetrics() []provider.CustomMetricInfo {
	r.lock.RLock()
	defer r.lock.RUnlock()
	infos := make([]provider.CustomMetricInfo, len(r.customMetrics))
	count := 0
	for k := range r.customMetrics {
		infos[count] = k
		count++
	}
	return infos
}

func (r *Routes) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	r.lock.RLock()
	defer r.lock.RUnlock()
	infos := make([]provider.ExternalMetricInfo, len(r.externalMetrics))
	count := 0
	for k := range r.externalMetrics {
		infos[count] = k
		count++
	}
	return infos
}
