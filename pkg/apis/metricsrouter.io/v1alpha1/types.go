package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen=true
type Service struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Port      int32  `json:"port"`
}

// +kubebuilder:validation:Enum=CustomMetrics;ExternalMetrics
type MetricType string

const (
	CustomMetricsType   = "CustomMetrics"
	ExternalMetricsType = "ExternalMetrics"
)

// +k8s:deepcopy-gen=true
type CustomMetricsSourceSpec struct {
	Service               Service      `json:"service"`
	InsecureSkipTLSVerify bool         `json:"insecureSkipTLSVerify"`
	Priority              int          `json:"priority"`
	MetricTypes           []MetricType `json:"metricTypes"`
}

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:deepcopy-gen=true
type CustomMetricsSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CustomMetricsSourceSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true
type CustomMetricsSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CustomMetricsSource `json:"items"`
}
