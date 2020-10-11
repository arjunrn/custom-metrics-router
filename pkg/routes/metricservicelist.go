package routes

import (
	"fmt"
	"sort"
	"time"
)

type MetricsAPIService struct {
	Name      string
	Namespace string
	Created   time.Time
	Priority  int
}

type MetricServiceList []MetricsAPIService

func NewMetricServiceList() *MetricServiceList {
	serviceList := make(MetricServiceList, 0)
	return &serviceList
}

func (m MetricServiceList) Len() int {
	return len(m)
}

func (m MetricServiceList) Less(i, j int) bool {
	if m[i].Priority < m[j].Priority {
		return true
	}
	if m[i].Priority > m[j].Priority {
		return false
	}

	if m[i].Created.Before(m[j].Created) {
		return true
	}
	return false
}

func (m MetricServiceList) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m *MetricServiceList) AddService(name, namespace string, created time.Time, priority int) {
	found := -1
	for i, s := range *m {
		if s.Name == name && s.Namespace == namespace {
			found = i
			break
		}
	}
	service := MetricsAPIService{
		Name:      name,
		Namespace: namespace,
		Created:   created,
		Priority:  priority,
	}
	if found != -1 {
		(*m)[found] = service
	} else {
		*m = append(*m, service)
	}
	sort.Sort(m)
}

func (m *MetricServiceList) RemoveService(namespace, name string) bool {
	found := -1
	for i, s := range *m {
		if s.Name == name && s.Namespace == namespace {
			found = i
		}
	}
	if found != -1 {
		*m = append((*m)[:found], (*m)[found+1:]...)
		sort.Sort(m)
	}
	if m.Len() > 0 {
		return true
	}
	return false
}

func (m *MetricServiceList) GetBestMetricService() (*MetricsAPIService, error) {
	if m.Len() == 0 {
		return nil, fmt.Errorf("no metric backend for metric")
	}
	service := (*m)[0]
	return &service, nil
}
