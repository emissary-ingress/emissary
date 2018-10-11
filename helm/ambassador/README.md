# Ambassador

Ambassador is an open source, Kubernetes-native [microservices API gateway](https://www.getambassador.io/about/microservices-api-gateways) built on the [Envoy Proxy](https://www.envoyproxy.io/).

## TL;DR;

```console
$ helm repo add datawire https://www.getambassador.io/helm
$ helm install datawire/ambassador
```

## Introduction

This chart bootstraps an [Ambassador](https://www.getambassador.io) deployment on
a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.7+

## Installing the Chart

To install the chart with the release name `my-release`:

```console
$ helm install --name my-release datawire/ambassador
```

The command deploys Ambassador API gateway on the Kubernetes cluster in the default configuration.
The [configuration](#configuration) section lists the parameters that can be configured during installation.

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```console
$ helm delete --purge my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

The following tables lists the configurable parameters of the Ambassador chart and their default values.

| Parameter                       | Description                                | Default                                                    |
| ------------------------------- | ------------------------------------------ | ---------------------------------------------------------- |
| `image.repository` | Image | `quay.io/datawire/ambassador`
| `image.tag` | Image tag | `0.35.0`
| `image.pullPolicy` | Image pull policy | `IfNotPresent`
| `image.imagePullSecrets` | Image pull secrets | None
| `daemonSet` | If `true `, Create a daemonSet. By default Deployment controller will be created | `false` 
| `replicaCount`  | Number of Ambassador replicas  | `1`
| `resources` | CPU/memory resource requests/limits | None
| `rbac.create` | If `true`, create and use RBAC resources | `true`
| `serviceAccount.create` | If `true`, create a new service account | `true`
| `serviceAccount.name` | Service account to be used | `ambassador`
| `namespace.single` | Set the `AMBASSADOR_SINGLE_NAMESPACE` environment variable | `false`
| `namespace.name` | Set the `AMBASSADOR_NAMESPACE` environment variable | `metadata.namespace`
| `ambassador.id` | Set the identifier of the Ambassador instance | none
| `service.enableHttp` | if port 80 should be opened for service | `true`
| `service.enableHttps` | if port 443 should be opened for service | `true`
| `service.targetPorts.http` | Sets the targetPort that maps to the service's cleartext port | `80`
| `service.targetPorts.https` | Sets the targetPort that maps to the service's TLS port | `443`
| `service.type` | Service type to be used | `LoadBalancer`
| `service.nodePort` | If explicit Nodeport is required | None
| `service.loadBalancerIP` | IP address to assign (if cloud provider supports it) | `""`
| `service.annotations` | Annotations to apply to Ambassador service | none
| `service.loadBalancerSourceRanges` | Passed to cloud provider load balancer if created (e.g: AWS ELB) | none
| `adminService.create` | If `true`, create a service for Ambassador's admin UI | `true`
| `adminService.type` | Ambassador's admin service type to be used | `ClusterIP`
| `exporter.image` | Prometheus exporter image | `prom/statsd-exporter:v0.6.0`
| `timing.restart` | The minimum number of seconds between Envoy restarts | none
| `timing.drain` | The number of seconds that the Envoy will wait for open connections to drain on a restart | none
| `timing.shutdown` | The number of seconds that Ambassador will wait for the old Envoy to clean up and exit on a restart | none

Make sure the configured `service.targetPorts.http` and `service.targetPorts.https` ports match your Ambassador Module's `service_port` and `redirect_cleartext_from` configurations. 

If you intend to use `service.annotations`, remember to include the annotation key, for example:

```
service:
  type: LoadBalancer
  port: 80
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v0
      kind: Module
      name:  ambassador
      config:
        diagnostics:
          enabled: false
        redirect_cleartext_from: 80
        service_port: 443
```

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```console
$ helm upgrade --install --wait my-release \
    --set adminService.type=NodePort \
    datawire/ambassador
```

Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example,

```console
$ helm upgrade --install --wait my-release -f values.yaml datawire/ambassador
```
