package metricsclient

import (
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"

	"github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/provider"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	cachedDiscovery "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/rest"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	customMetricsAPI "k8s.io/metrics/pkg/apis/custom_metrics/v1beta1"
	"k8s.io/metrics/pkg/apis/custom_metrics/v1beta2"
	"k8s.io/metrics/pkg/apis/external_metrics"
	externalMetricsAPI "k8s.io/metrics/pkg/apis/external_metrics/v1beta1"
	cmClient "k8s.io/metrics/pkg/client/custom_metrics"
	emClient "k8s.io/metrics/pkg/client/external_metrics"
)

var (
	tokenFile  = pflag.String("token-file", "/var/run/secrets/kubernetes.io/serviceaccount/token", "path to token file")
	rootCAFile = pflag.String("root-ca-faile", "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt", "path to root CA file")
)

type Client struct {
	customMetricsClient   cmClient.CustomMetricsClient
	externalMetricsClient emClient.ExternalMetricsClient
	discoveryClient       discovery.CachedDiscoveryInterface
	mapper                meta.RESTMapper
	namespace             string
	name                  string
}

// InClusterConfig returns a config object which uses the service account
// kubernetes gives to pods. It's intended for clients that expect to be
// running inside a pod running on kubernetes. It will return ErrNotInCluster
// if called from a process not running in a kubernetes environment.
func InClusterConfig(host, port string, insecure bool) (*rest.Config, error) {
	token, err := ioutil.ReadFile(*tokenFile)
	if err != nil {
		return nil, err
	}
	var tlsClientConfig rest.TLSClientConfig
	if insecure {
		tlsClientConfig.Insecure = true
	} else {
		if _, err := certutil.NewPool(*rootCAFile); err != nil {
			klog.Errorf("Expected to load root CA config from %s, but got err: %v", *rootCAFile, err)
		} else {
			tlsClientConfig.CAFile = *rootCAFile
		}
	}

	return &rest.Config{
		// TODO: switch to using cluster DNS.
		Host: "https://" + net.JoinHostPort(host, port),
		// Host:            "https://localhost:6443",
		TLSClientConfig: tlsClientConfig,
		BearerToken:     string(token),
		BearerTokenFile: *tokenFile,
	}, nil
}

func NewClient(insecureTLSSkipVerify bool, name, namespace string, port int32, mapper meta.RESTMapper) (*Client, error) {
	host := fmt.Sprintf("%s.%s", name, namespace)
	config, err := InClusterConfig(host, strconv.Itoa(int(port)), insecureTLSSkipVerify)
	if err != nil {
		return nil, fmt.Errorf("failed to generate rest config for %s: %v", host, err)
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %v", err)
	}
	cachedClient := cachedDiscovery.NewMemCacheClient(discoveryClient)
	customMetricsClient := cmClient.NewForConfig(config, mapper, cmClient.NewAvailableAPIsGetter(discoveryClient))
	externalMetricsClient, err := emClient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create external metrics client: %v", err)
	}

	return &Client{
		name:                  name,
		namespace:             namespace,
		customMetricsClient:   customMetricsClient,
		externalMetricsClient: externalMetricsClient,
		discoveryClient:       cachedClient,
		mapper:                mapper,
	}, err
}

func (c *Client) ListCustomMetricInfos() (map[provider.CustomMetricInfo]struct{}, error) {
	resources, err := c.discoveryClient.ServerResourcesForGroupVersion(customMetricsAPI.SchemeGroupVersion.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get resource for %s: %v", customMetricsAPI.SchemeGroupVersion, err)

	}
	metricInfos := make(map[provider.CustomMetricInfo]struct{})
	for _, r := range resources.APIResources {
		parts := strings.SplitN(r.Name, "/", 2)
		if len(parts) != 2 {
			klog.Warningf("provider %s/%s returned a malformed metrics with name %s", c.namespace, c.name, r.Name)
			continue
		}
		resource, err := c.mapper.ResourceFor(schema.GroupVersionResource{Resource: parts[0]})
		if err != nil {
			klog.Warningf("failed to get group version for resource %s from provider %s/%s", parts[0], c.namespace, c.name)
			continue
		}
		info := provider.CustomMetricInfo{
			GroupResource: schema.GroupResource{Group: resource.Group, Resource: resource.Resource},
			Namespaced:    r.Namespaced, Metric: parts[1],
		}
		metricInfos[info] = struct{}{}
	}
	return metricInfos, nil
}

func (c *Client) GetMetricByName(name types.NamespacedName, info provider.CustomMetricInfo, selector labels.Selector) (*custom_metrics.MetricValue, error) {
	var object *v1beta2.MetricValue

	var err error
	if info.Namespaced {
		object, err = c.customMetricsClient.NamespacedMetrics(name.Namespace).GetForObject(
			schema.GroupKind{Group: info.GroupResource.Group, Kind: info.GroupResource.Resource},
			name.Name, info.Metric, selector,
		)
	} else {
		object, err = c.customMetricsClient.RootScopedMetrics().GetForObject(
			schema.GroupKind{Group: info.GroupResource.Group, Kind: info.GroupResource.Resource},
			name.Name, info.Metric, selector,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get metric from backend: %v", err)
	}
	return &custom_metrics.MetricValue{
		DescribedObject: custom_metrics.ObjectReference{
			Kind:            object.DescribedObject.Kind,
			Namespace:       object.DescribedObject.Namespace,
			Name:            object.DescribedObject.Name,
			APIVersion:      object.DescribedObject.APIVersion,
			ResourceVersion: object.DescribedObject.ResourceVersion,
		},
		Metric: custom_metrics.MetricIdentifier{
			Name:     object.Metric.Name,
			Selector: object.Metric.Selector,
		},
		Timestamp:     object.Timestamp,
		WindowSeconds: object.WindowSeconds,
		Value:         object.Value,
	}, nil
}

func (c *Client) GetMetricBySelector(namespace string, selector labels.Selector, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
	var objects *v1beta2.MetricValueList
	var err error
	kind, err := c.mapper.ResourceSingularizer(info.GroupResource.Resource)
	if err != nil {
		return nil, fmt.Errorf("failed to singularize %s: %v", info.GroupResource.Resource, err)
	}
	klog.Infof("custom metric info: %#v", info)
	if info.Namespaced {
		objects, err = c.customMetricsClient.NamespacedMetrics(namespace).GetForObjects(
			schema.GroupKind{
				Group: info.GroupResource.Group,
				Kind:  kind,
			},
			selector, info.Metric, metricSelector,
		)
	} else {
		objects, err = c.customMetricsClient.RootScopedMetrics().GetForObjects(
			schema.GroupKind{
				Group: info.GroupResource.Group,
				Kind:  kind,
			},
			selector, info.Metric, metricSelector,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get metric from backend: %v", err)
	}
	values := make([]custom_metrics.MetricValue, len(objects.Items))
	for i, v := range objects.Items {
		values[i] = custom_metrics.MetricValue{
			DescribedObject: custom_metrics.ObjectReference{
				Kind:            v.DescribedObject.Kind,
				Namespace:       v.DescribedObject.Namespace,
				Name:            v.DescribedObject.Name,
				APIVersion:      v.DescribedObject.APIVersion,
				ResourceVersion: v.DescribedObject.ResourceVersion,
			},
			Metric: custom_metrics.MetricIdentifier{
				Name:     v.Metric.Name,
				Selector: v.Metric.Selector,
			},
			Timestamp:     v.Timestamp,
			WindowSeconds: v.WindowSeconds,
			Value:         v.Value,
		}
	}
	return &custom_metrics.MetricValueList{
		Items: values,
	}, nil
}

func (c *Client) ListExternalMetrics() (map[provider.ExternalMetricInfo]struct{}, error) {
	infos := make(map[provider.ExternalMetricInfo]struct{})
	resources, err := c.discoveryClient.ServerResourcesForGroupVersion(externalMetricsAPI.SchemeGroupVersion.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get resource for %s: %v", externalMetricsAPI.SchemeGroupVersion, err)
	}
	for _, r := range resources.APIResources {
		info := provider.ExternalMetricInfo{
			Metric: r.Name,
		}
		infos[info] = struct{}{}
	}
	return infos, nil
}

func (c *Client) GetExternalMetric(name, namespace string, selector labels.Selector) (*external_metrics.ExternalMetricValueList, error) {
	result, err := c.externalMetricsClient.NamespacedMetrics(namespace).List(name, selector)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics for external metric %s/%s: %v", namespace, name, err)
	}
	valueList := &external_metrics.ExternalMetricValueList{
		Items: make([]external_metrics.ExternalMetricValue, len(result.Items)),
	}
	for i, m := range result.Items {
		valueList.Items[i] = external_metrics.ExternalMetricValue{
			TypeMeta:      metav1.TypeMeta{Kind: m.Kind, APIVersion: m.APIVersion},
			MetricName:    m.MetricName,
			MetricLabels:  m.MetricLabels,
			Timestamp:     m.Timestamp,
			WindowSeconds: m.WindowSeconds,
			Value:         m.Value,
		}
	}
	return valueList, nil
}
