package controller

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	"github.com/arjunrn/custom-metrics-router/pkg/apis/metricsrouter.io/v1alpha1"
	"github.com/arjunrn/custom-metrics-router/pkg/client/informers/externalversions"
	alpha1 "github.com/arjunrn/custom-metrics-router/pkg/client/informers/externalversions/metricsrouter.io/v1alpha1"
	mrLister "github.com/arjunrn/custom-metrics-router/pkg/client/listers/metricsrouter.io/v1alpha1"
	"github.com/arjunrn/custom-metrics-router/pkg/clientset"
	"github.com/arjunrn/custom-metrics-router/pkg/routes"
)

type Controller struct {
	clientSet              clientset.Interface
	customRoutes           *routes.Routes
	queue                  workqueue.RateLimitingInterface
	informer               cache.SharedIndexInformer
	customMetricsLister    mrLister.CustomMetricsSourceLister
	customMetricsHasSynced func() bool
	customMetricsInformer  alpha1.CustomMetricsSourceInformer
}

func NewController(clientSet clientset.Interface, customRoutes *routes.Routes) *Controller {
	factory := externalversions.NewSharedInformerFactory(clientSet, time.Minute)
	customMetricsInformer := factory.Metricsrouter().V1alpha1().CustomMetricsSources()
	controller := &Controller{
		customRoutes: customRoutes,
		clientSet:    clientSet,
		queue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "metricsrouter"),
		informer:     customMetricsInformer.Informer(),
	}
	customMetricsInformer.Informer().AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueRoute,
		UpdateFunc: func(oldObj, newObj interface{}) {
			controller.enqueueRoute(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			controller.deleteRoute(obj)
		},
	}, time.Minute)
	controller.customMetricsInformer = customMetricsInformer
	controller.customMetricsLister = customMetricsInformer.Lister()
	controller.customMetricsHasSynced = customMetricsInformer.Informer().HasSynced
	return controller
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()
	go c.customMetricsInformer.Informer().Run(stopCh)
	klog.Infof("Starting metrics router controller")
	defer klog.Infof("Shutting down metrics router controller")

	if !cache.WaitForNamedCacheSync("metrics-router", stopCh, c.customMetricsHasSynced) {
		return
	}

	// start a single worker (we may wish to start more in the future)
	go wait.Until(c.worker, time.Second, stopCh)
	<-stopCh
}

func (c *Controller) enqueueRoute(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("unable to get key for object %+v: %v", obj, err))
		return
	}
	c.queue.AddRateLimited(key)
}

func (c *Controller) deleteRoute(obj interface{}) {
	provider := obj.(*v1alpha1.CustomMetricsSource)
	c.customRoutes.RemoveService(provider.Name, provider.Namespace)
	c.queue.Forget(obj)
}

func (c *Controller) updateRoutes(provider *v1alpha1.CustomMetricsSource) error {
	var (
		customMetrics, externalMetrics bool
	)
	for _, metricType := range provider.Spec.MetricTypes {
		if metricType == v1alpha1.CustomMetricsType {
			customMetrics = true
		}
		if metricType == v1alpha1.ExternalMetricsType {
			externalMetrics = true
		}
	}
	return c.customRoutes.AddService(
		provider.Spec.Service.Name,
		provider.Spec.Service.Namespace,
		provider.Spec.Service.Port,
		provider.Spec.Priority,
		provider.Spec.InsecureSkipTLSVerify,
		provider.ObjectMeta.CreationTimestamp.Time,
		customMetrics,
		externalMetrics,
	)
}

func (c *Controller) worker() {
	for c.processNextWorkItem() {
	}
	klog.Infof("custom metrics router controller worker shutting down")
}

func (c *Controller) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	deleted, err := c.reconcileKey(key.(string))
	if err != nil {
		utilruntime.HandleError(err)
	}
	if !deleted {
		c.queue.AddRateLimited(key)
	}

	return true
}

func (c *Controller) reconcileKey(key string) (deleted bool, err error) {
	service, err := c.customMetricsLister.Get(key)
	if errors.IsNotFound(err) {
		klog.Infof("Custom Metrics Source %s has been deleted", key)
		return true, nil
	}
	if err != nil {
		return false, err
	}

	return false, c.updateRoutes(service)
}
