package clientset

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/arjunrn/custom-metrics-router/pkg/client/clientset/versioned/typed/metricsrouter.io/v1alpha1"

	metricsRouter "github.com/arjunrn/custom-metrics-router/pkg/client/clientset/versioned"
)

type Interface interface {
	kubernetes.Interface
	metricsRouter.Interface
}

type ClientSet struct {
	kubernetes.Interface
	metricsProvider metricsRouter.Interface
}

func (c *ClientSet) MetricsrouterV1alpha1() v1alpha1.MetricsrouterV1alpha1Interface {
	return c.metricsProvider.MetricsrouterV1alpha1()
}

func NewForConfig(kubeConfig *rest.Config) (Interface, error) {
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	metricsProvider, err := metricsRouter.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	return NewClientSet(kubeClient, metricsProvider), nil
}

func NewClientSet(kubeClient *kubernetes.Clientset, provider *metricsRouter.Clientset) Interface {
	return &ClientSet{
		Interface:       kubeClient,
		metricsProvider: provider,
	}
}
