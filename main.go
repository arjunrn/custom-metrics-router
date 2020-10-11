package main

import (
	"flag"
	"os"

	basecmd "github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/cmd"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/arjunrn/custom-metrics-router/controller"
	"github.com/arjunrn/custom-metrics-router/pkg/clientset"
	"github.com/arjunrn/custom-metrics-router/pkg/provider"
	"github.com/arjunrn/custom-metrics-router/pkg/routes"
)

type RoutedAdapter struct {
	basecmd.AdapterBase
}

func main() {
	cmd := &RoutedAdapter{}
	cmd.Flags().AddGoFlagSet(flag.CommandLine) // make sure you get the klog flags
	err := cmd.Flags().Parse(os.Args)
	if err != nil {
		klog.Fatalf("failed to parse flags: %v", err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", cmd.RemoteKubeConfigFile)
	if err != nil {
		panic(err.Error())
	}
	// create the clientSet
	clientSet, err := clientset.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		panic(err)
	}
	cachedDiscoveryClient := memory.NewMemCacheClient(discoveryClient)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)
	customRoutes := routes.New(mapper)

	c := controller.NewController(clientSet, customRoutes)
	stopCh := make(chan struct{})
	go c.Run(stopCh)
	defer close(stopCh)

	routedProvider := provider.NewRoutedProvider(customRoutes)
	cmd.WithCustomMetrics(routedProvider)
	cmd.WithExternalMetrics(routedProvider)

	if err := cmd.Run(wait.NeverStop); err != nil {
		klog.Fatalf("unable to run custom metrics routedProvider: %v", err)
	}
}
