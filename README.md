# Custom Metrics Router

Router to support multiple metrics adapters in Kubernetes


## Running the router

### Build and push the image

```bash
IMAGE=custom-metrics-router VERSION=v0.1.0 make build.push
```

Here `IMAGE` should be the set to an image name which to which location
the built image will be subsequently pushed. `VERSION` can also be set
accordingly, or a value will be generated from the git commit.

### Deploy the router

```bash
kubectl create ns custom-metrics
kubectl apply -f deploy/metricsrouter.io_custommetricsssources.yaml
kubectl apply -f deploy/rbac.yaml
```

In the `deploy/deployment.yaml`file modify the `image` field with the value
of the docker image from the previous step.

```bash
kubectl apply -f deploy/deployment.yaml
kubectl apply -f deploy/service.yaml
```

The next step registers the router as the default API service for
`custom.metrics.k8s.io` and `external.metrics.k8s.io`. Before this step
ensure that there is no existing service for that API group or the existing
service register has lower priority.

```bash
kubectl apply -f deploy/apiservice.yaml
```

Modify the file `deploy/example.yaml` to point to an existing custom or external
metrics provider

### Testing the metrics router.

```bash
kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta2/"
kubectl get --raw "/apis/external.metrics.k8s.io/v1beta2/"
```