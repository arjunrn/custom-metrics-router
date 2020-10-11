module github.com/arjunrn/custom-metrics-router

go 1.15

require (
	github.com/hashicorp/go-multierror v1.1.0 // indirect
	github.com/kubernetes-sigs/custom-metrics-apiserver v0.0.0-20201023134757-8a652aad2cb2
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	k8s.io/api v0.18.9
	k8s.io/apimachinery v0.18.9
	k8s.io/client-go v0.18.2
	k8s.io/klog v1.0.0
	k8s.io/metrics v0.18.2
	sigs.k8s.io/controller-tools v0.4.0
)
