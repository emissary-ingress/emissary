# Ambassador

The Ambassador Edge Stack is a self-service, comprehensive edge stack that is Kubernetes-native and built on [Envoy Proxy](https://www.envoyproxy.io/).

## TL;DR;

```console
$ helm repo add datawire https://getambassador.io
$ helm install ambassador datawire/ambassador
```

## Introduction

This chart bootstraps an [Ambassador](https://www.getambassador.io) deployment on
a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.11+

## Add this Helm repository to your Helm client

```console
helm repo add datawire https://getambassador.io
```

## Installing the Chart

To install the chart with the release name `my-release`:

```console
$ kubectl create namespace ambassador
$ helm install my-release datawire/ambassador -n ambassador
```

The command deploys Ambassador Edge Stack in the ambassador namespace on the Kubernetes cluster in the default configuration.

It is recommended to use the ambassador namespace for easy upgrades.

The [configuration](#configuration) section lists the parameters that can be configured during installation.

### Ambassador Edge Stack Installation

This chart defaults to installing The Ambassador Edge Stack with all of its configuration objects.

- A Redis instance
- `AuthService` resource for enabling authentication
- `RateLimitService` resource for enabling rate limiting
- `Mapping`s for internal request routing

If installing alongside another deployment of Ambassador, some of these resources can cause configuration errors since only one `AuthService` or `RateLimitService` can be configured at a time.

If you already have one of these resources configured in your cluster, please see the [configuration](#configuration) section below for information on how to disable them in the chart.

### Ambassador OSS Installation

This chart can still be used to install Ambassador OSS.

To install OSS, change the `image` to use the OSS image and set `enableAES: false` to skip the install of any AES resources.

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```console
$ helm uninstall my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Changelog

Notable chart changes are listed in the [CHANGELOG](./CHANGELOG.md)

## Configuration

The following tables lists the configurable parameters of the Ambassador chart and their default values.

| Parameter                                          | Description                                                                                                                                                              | Default                                                                                             |
|----------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------|
| `nameOverride`                                     | Override the generated chart name. Defaults to .Chart.Name.                                                                                                              |                                                                                                     |
| `fullnameOverride`                                 | Override the generated release name. Defaults to .Release.Name.                                                                                                          |                                                                                                     |
| `namespaceOverride`                                | Override the generated release namespace. Defaults to .Release.Namespace.                                                                                                |                                                                                                     |
| `adminService.create`                              | If `true`, create a service for Ambassador's admin UI                                                                                                                    | `true`                                                                                              |
| `adminService.nodePort`                            | If explicit NodePort for admin service is required                                                                                                                       | `true`                                                                                              |
| `adminService.type`                                | Ambassador's admin service type to be used                                                                                                                               | `ClusterIP`                                                                                         |
| `adminService.annotations`                         | Annotations to apply to Ambassador admin service                                                                                                                         | `{}`                                                                                                |
| `adminService.loadBalancerIP`                      | IP address to assign (if cloud provider supports it)                                                                                                                     | `""`                                                                                                |
| `adminService.loadBalancerSourceRanges`            | Passed to cloud provider load balancer if created (e.g: AWS ELB)                                                                                                         | None                                                                                                |
| `ambassadorConfig`                                 | Config thats mounted to `/ambassador/ambassador-config`                                                                                                                  | `""`                                                                                                |
| `crds.enabled`                                     | If `true`, enables CRD resources for the installation.                                                                                                                   | `true`                                                                                              |
| `crds.create`                                      | If `true`, Creates CRD resources                                                                                                                                         | `true`                                                                                              |
| `crds.keep`                                        | If `true`, if the ambassador CRDs should be kept when the chart is deleted                                                                                               | `true`                                                                                              |
| `daemonSet`                                        | If `true`, Create a DaemonSet. By default Deployment controller will be created                                                                                          | `false`                                                                                             |
| `test.enabled`                                     | If `true`, Create test Pod to verify the Ambassador service works correctly (Only created on `helm test`)                                                                | `true`                                                                                              |
| `test.image`                                       | Image to use for the test Pod                                                                                                                                            | `busybox`                                                                                           |
| `hostNetwork`                                      | If `true`, uses the host network, useful for on-premise setups                                                                                                           | `false`                                                                                             |
| `dnsPolicy`                                        | Dns policy, when hostNetwork set to ClusterFirstWithHostNet                                                                                                              | `ClusterFirst`                                                                                      |
| `env`                                              | Any additional environment variables for ambassador pods                                                                                                                 | `{}`                                                                                                |
| `envRaw`                                           | Additional environment variables in raw YAML format                                                                                                                      | `{}`                                                                                                |
| `image.pullPolicy`                                 | Ambassador image pull policy                                                                                                                                             | `IfNotPresent`                                                                                      |
| `image.repository`                                 | Ambassador image                                                                                                                                                         | `docker.io/datawire/aes`                                                                            |
| `image.tag`                                        | Ambassador image tag                                                                                                                                                     | `1.14.2`                                                                                             |
| `imagePullSecrets`                                 | Image pull secrets                                                                                                                                                       | `[]`                                                                                                |
| `namespace.name`                                   | Set the `AMBASSADOR_NAMESPACE` environment variable                                                                                                                      | `metadata.namespace`                                                                                |
| `scope.singleNamespace`                            | Set the `AMBASSADOR_SINGLE_NAMESPACE` environment variable and create namespaced RBAC if `rbac.enabled: true`                                                            | `false`                                                                                             |
| `podAnnotations`                                   | Additional annotations for ambassador pods                                                                                                                               | `{}`                                                                                                |
| `deploymentAnnotations`                            | Additional annotations for ambassador DaemonSet/Deployment                                                                                                               | `{}`                                                                                                |
| `podLabels`                                        | Additional labels for ambassador pods                                                                                                                                    |                                                                                                     |
| `deploymentLabels`                                 | Additional labels for ambassador DaemonSet/Deployment                                                                                                                    |                                                                                                     |
| `affinity`                                         | Affinity for ambassador pods                                                                                                                                             | `{}`                                                                                                |
| `topologySpreadConstraints`                        | Topology Spread Constraints for Ambassador pods. Stable since 1.19.                                                                                                      | `[]`                                                                                                |
| `nodeSelector`                                     | NodeSelector for ambassador pods                                                                                                                                         | `{}`                                                                                                |
| `priorityClassName`                                | The name of the priorityClass for the ambassador DaemonSet/Deployment                                                                                                    | `""`                                                                                                |
| `rbac.create`                                      | If `true`, create and use RBAC resources                                                                                                                                 | `true`                                                                                              |
| `rbac.podSecurityPolicies`                         | pod security polices to bind to                                                                                                                                          |                                                                                                     |
| `rbac.nameOverride`                                | Overrides the default name of the RBAC resources                                                                                                                         | ``                                                                                                  |
| `replicaCount`                                     | Number of Ambassador replicas                                                                                                                                            | `3`                                                                                                 |
| `resources`                                        | CPU/memory resource requests/limits                                                                                                                                      | `{ "limits":{"cpu":"1000m","memory":"600Mi"},"requests":{"cpu":"200m","memory":"300Mi"}}`           |
| `securityContext`                                  | Set security context for pod                                                                                                                                             | `{ "runAsUser": "8888" }`                                                                           |
| `security.podSecurityContext`                      | Set the security context for the Ambassador pod                                                                                                                          | `{ "runAsUser": "8888" }`                                                                           |
| `security.containerSecurityContext`                | Set the security context for the Ambassador container                                                                                                                    | `{ "allowPrivilegeEscalation": false }`                                                             |
| `security.podSecurityPolicy`                       | Create a PodSecurityPolicy to be used for the pod.                                                                                                                       | `{}`                                                                                                |
| `progressDeadlines.ambassador`                     | Configures progressDeadlineSeconds for the Ambassador deployment                                                                                                         |`600`                                                                                              |
| `progressDeadlines.agent`                          | Configures progressDeadlineSeconds for the Ambassador-Agent deployment                                                                                                   | `600`                                                                                              |
| `restartPolicy`                                    | Set the `restartPolicy` for pods                                                                                                                                         | ``                                                                                                  |
| `terminationGracePeriodSeconds`                    | Set the `terminationGracePeriodSeconds` for the pod. Defaults to 30 if unset.                                                                                            | ``                                                                                                  |
| `initContainers`                                   | Containers used to initialize context for pods                                                                                                                           | `[]`                                                                                                |
| `sidecarContainers`                                | Containers that share the pod context                                                                                                                                    | `[]`                                                                                                |
| `livenessProbe.initialDelaySeconds`                | Initial delay (s) for Ambassador pod's liveness probe                                                                                                                    | `30`                                                                                                |
| `livenessProbe.periodSeconds`                      | Probe period (s) for Ambassador pod's liveness probe                                                                                                                     | `3`                                                                                                 |
| `livenessProbe.failureThreshold`                   | Failure threshold for Ambassador pod's liveness probe                                                                                                                    | `3`                                                                                                 |
| `readinessProbe.initialDelaySeconds`               | Initial delay (s) for Ambassador pod's readiness probe                                                                                                                   | `30`                                                                                                |
| `readinessProbe.periodSeconds`                     | Probe period (s) for Ambassador pod's readiness probe                                                                                                                    | `3`                                                                                                 |
| `readinessProbe.failureThreshold`                  | Failure threshold for Ambassador pod's readiness probe                                                                                                                   | `3`                                                                                                 |
| `service.annotations`                              | Annotations to apply to Ambassador service                                                                                                                               | `""`                                                                                                |
| `service.externalTrafficPolicy`                    | Sets the external traffic policy for the service                                                                                                                         | `""`                                                                                                |
| `service.nameOverride`                             | Sets the name of the service                                                                                                                                             | `ambassador.fullname`                                                                               |
| `service.ports`                                    | List of ports Ambassador is listening on                                                                                                                                 | `[{"name": "http","port": 80,"targetPort": 8080},{"name": "https","port": 443,"targetPort": 8443}]` |
| `service.loadBalancerIP`                           | IP address to assign (if cloud provider supports it)                                                                                                                     | `""`                                                                                                |
| `service.loadBalancerSourceRanges`                 | Passed to cloud provider load balancer if created (e.g: AWS ELB)                                                                                                         | None                                                                                                |
| `service.sessionAffinity`                          | Sets the session affinity policy for the service                                                                                                                         | `""`                                                                                                |
| `service.sessionAffinityConfig`                    | Sets the session affinity config for the service                                                                                                                         | `""`                                                                                                |
| `service.type`                                     | Service type to be used                                                                                                                                                  | `LoadBalancer`                                                                                      |
| `service.externalIPs`                              | External IPs to route to the ambassador service                                                                                                                          | `[]`                                                                                                |
| `serviceAccount.create`                            | If `true`, create a new service account                                                                                                                                  | `true`                                                                                              |
| `serviceAccount.name`                              | Service account to be used                                                                                                                                               | `ambassador`                                                                                        |
| `volumeMounts`                                     | Volume mounts for the ambassador service                                                                                                                                 | `[]`                                                                                                |
| `volumes`                                          | Volumes for the ambassador service                                                                                                                                       | `[]`                                                                                                |
| `enableAES`                                        | Create the [AES configuration objects](#ambassador-edge-stack-installation)                                                                                              | `true`                                                                                              |
| `createDevPortalMappings`                          | Expose the dev portal on `/docs/` and `/documentation/`                                                                                                                  | `true`                                                                                              |
| `licenseKey.value`                                 | Ambassador Edge Stack license. Empty will install in evaluation mode.                                                                                                    | ``                                                                                                  |
| `licenseKey.createSecret`                          | Set to `false` if installing mutltiple Ambassdor Edge Stacks in a namespace.                                                                                             | `true`                                                                                              |
| `licenseKey.secretName`                            | Name of the secret to store Ambassador license key in.                                                                                                                   | ``                                                                                                  |
| `licenseKey.annotations`                           | Annotations to attach to the license-key-secret.                                                                                                                         | {}                                                                                                  |
| `redisURL`                                         | URL of redis instance not created by the release                                                                                                                         | `""`                                                                                                |
| `redisEnv`                                         | (**DEPRECATED:** Use `envRaw`) Set env vars that control how Ambassador interacts with redis.                                                                            | `""`                                                                                                |
| `redis.create`                                     | Create a basic redis instance with default configurations                                                                                                                | `true`                                                                                              |
| `redis.annotations`                                | Annotations for the redis service and deployment                                                                                                                         | `""`                                                                                                |
| `redis.resources`                                  | Resource requests for the redis instance                                                                                                                                 | `""`                                                                                                |
| `redis.nodeSelector`                               | NodeSelector for redis pods                                                                                                                                              | `{}`                                                                                                |
| `redis.affinity`                                   | Affinity for redis pods                                                                                                                                                  | `{}`                                                                                                |
| `redis.tolerations`                                | Tolerations for redis pods                                                                                                                                               | `{}`                                                                                                |
| `authService.create`                               | Create the `AuthService` CRD for Ambassador Edge Stack                                                                                                                   | `true`                                                                                              |
| `authService.optional_configurations`              | Config options for the `AuthService` CRD                                                                                                                                 | `""`                                                                                                |
| `rateLimit.create`                                 | Create the `RateLimit` CRD for Ambassador Edge Stack                                                                                                                     | `true`                                                                                              |
| `registry.create`                                  | Create the `Project` registry.                                                                                                                                           | `false`                                                                                             |
| `autoscaling.enabled`                              | If true, creates Horizontal Pod Autoscaler                                                                                                                               | `false`                                                                                             |
| `autoscaling.minReplicas`                          | If autoscaling enabled, this field sets minimum replica count                                                                                                            | `2`                                                                                                 |
| `autoscaling.maxReplicas`                          | If autoscaling enabled, this field sets maximum replica count                                                                                                            | `5`                                                                                                 |
| `autoscaling.metrics`                              | If autoscaling enabled, configure hpa metrics                                                                                                                            |                                                                                                     |
| `podDisruptionBudget`                              | Pod disruption budget rules                                                                                                                                              | `{}`                                                                                                |
| `resolvers.endpoint.create`                        | Create a KubernetesEndpointResolver                                                                                                                                      | `false`                                                                                             |
| `resolvers.endpoint.name`                          | If creating a KubernetesEndpointResolver, the resolver name                                                                                                              | `endpoint`                                                                                          |
| `resolvers.consul.create`                          | Create a ConsulResolver                                                                                                                                                  | `false`                                                                                             |
| `resolvers.consul.name`                            | If creating a ConsulResolver, the resolver name                                                                                                                          | `consul-dc1`                                                                                        |
| `resolvers.consul.spec`                            | If creating a ConsulResolver, additional configuration                                                                                                                   | `{}`                                                                                                |
| `module`                                           | Configure and manage the Ambassador Module from the Chart                                                                                                                | `{}`                                                                                                |
| `prometheusExporter.enabled`                       | DEPRECATED: Prometheus exporter side-car enabled                                                                                                                         | `false`                                                                                             |
| `prometheusExporter.pullPolicy`                    | DEPRECATED: Image pull policy                                                                                                                                            | `IfNotPresent`                                                                                      |
| `prometheusExporter.repository`                    | DEPRECATED: Prometheus exporter image                                                                                                                                    | `prom/statsd-exporter`                                                                              |
| `prometheusExporter.tag`                           | DEPRECATED: Prometheus exporter image                                                                                                                                    | `v0.8.1`                                                                                            |
| `prometheusExporter.resources`                     | DEPRECATED: CPU/memory resource requests/limits                                                                                                                          | `{}`                                                                                                |
| `metrics.serviceMonitor.enabled`                   | Create ServiceMonitor object (`adminService.create` should be to `true`)                                                                                                 | `false`                                                                                             |
| `metrics.serviceMonitor.interval`                  | Interval at which metrics should be scraped                                                                                                                              | `30s`                                                                                               |
| `metrics.serviceMonitor.scrapeTimeout`             | Timeout after which the scrape is ended                                                                                                                                  | `30s`                                                                                               |
| `metrics.serviceMonitor.selector`                  | Label Selector for Prometheus to find ServiceMonitors                                                                                                                    | `{ prometheus: kube-prometheus }`                                                                   |
| `servicePreview.enabled`                           | If true, install Service Preview components: traffic-manager & traffic-agent (`enableAES` needs to also be to `true`)                                                    | `false`                                                                                             |
| `servicePreview.trafficManager.image.repository`   | Ambassador Traffic-manager image                                                                                                                                         | Same value as `image.repository`                                                                    |
| `servicePreview.trafficManager.image.tag`          | Ambassador Traffic-manager image tag                                                                                                                                     | Same value as `image.tag`                                                                           |
| `servicePreview.trafficManager.serviceAccountName` | Traffic-manager Service Account to be used                                                                                                                               | `traffic-manager`                                                                                   |
| `servicePreview.trafficAgent.image.repository`     | Ambassador Traffic-agent image                                                                                                                                           | Same value as `image.repository`                                                                    |
| `servicePreview.trafficAgent.image.tag`            | Ambassador Traffic-agent image tag                                                                                                                                       | Same value as `image.tag`                                                                           |
| `servicePreview.trafficAgent.injector.enabled`     | If true, install the ambassador-injector                                                                                                                                 | `true`                                                                                              |
| `servicePreview.trafficAgent.injector.crtPEM`      | TLS certificate for the Common Name of <ambassador-injector>.<namespace>.svc                                                                                             | Auto-generated, valid for 365 days                                                                  |
| `servicePreview.trafficAgent.injector.keyPEM`      | TLS private key for the Common Name of <ambassador-injector>.<namespace>.svc                                                                                             | Auto-generated, valid for 365 days                                                                  |
| `servicePreview.trafficAgent.port`                 | Traffic-agent listening port number when injected with ambassador-injector                                                                                               | `9900`                                                                                              |
| `servicePreview.trafficAgent.serviceAccountName`   | Label Selector for Prometheus to find ServiceMonitors                                                                                                                    | `traffic-agent`                                                                                     |
| `servicePreview.trafficAgent.singleNamespace`      | If `true`, installs the traffic-agent ServiceAccount and Role in the current installation namespace; Otherwise uses a global ClusterRole applied to every ServiceAccount | `true`                                                                                              |
| `agent.enabled`                                    | If `true`, installs the ambassador-agent Deployment, ServiceAccount and ClusterRole in the ambassador namespace                                                                     | `true`                                                                                              |
| `agent.cloudConnectionToken`                       | API token for reporting snapshots to the [Service Catalog](https://app.getambassador.io/cloud/catalog/); If empty, agent will not report snapshots                       | `""`                                                                                                |
| `agent.rpcAddress`                                 | Address of the ambassador Service Catalog rpc server.                                                                                                                    | `https://app.getambassador.io/`                                                                     |
| `agent.image.repository`                           | Image repository for the ambassador-agent deployment. Defaults to value of `image.repository`                                                                            | Same value as `image.repository`                                                                    |
| `agent.image.tag`                                  | Image tag for the ambassador-agent deployment. Defaults to value of `image.tag`                                                                                          | Same value as `image.tag`                                                                           |

**NOTE:** Make sure the configured `service.http.targetPort` and `service.https.targetPort` ports match your [Ambassador Module's](https://www.getambassador.io/reference/modules/#the-ambassador-module) `service_port` and `redirect_cleartext_from` configurations.

### The Ambasssador Edge Stack

The Ambassador Edge Stack provides a comprehensive, self-service edge stack in
the Kubernetes cluster with a decentralized deployment model and a declarative
paradigm.

By default, this chart will install the latest image of The Ambassador Edge
Stack which will replace your existing deployment of Ambassador with no changes
to functionality.

### CRDs

This helm chart includes the creation of the core CRDs Ambassador uses for
configuration.

The `crds` flags (Helm 2 only) let you configure how a release manages crds.
- `crds.create` Can only be set on your first/master Ambassador release.
- `crds.enabled` Should be set on all releases using Ambassador CRDs
- `crds.keep` Configures if the CRDs are deleted when the master release is
  purged. This value is only checked for the master release and can be set to
  any value on secondary releases.

### Security

Ambassador takes security very seriously. For this reason, the YAML installation will default with a couple of basic security policies in place.

The `security` field of the `values.yaml` file configures these default policies and replaces the `securityContext` field used earlier.

The defaults will configure the pod to run as a non-root user and prohibit privilege escalation and outline a `PodSecurityPolicy` to ensure these conditions are met.



```yaml
security:
  # Security Context for all containers in the pod.
  # https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#podsecuritycontext-v1-core
  podSecurityContext:
    runAsUser: 8888
  # Security Context for the Ambassador container specifically
  # https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#securitycontext-v1-core
  containerSecurityContext:
    allowPrivilegeEscalation: false
  # A basic PodSecurityPolicy to ensure Ambassador is running with appropriate security permissions
  # https://kubernetes.io/docs/concepts/policy/pod-security-policy/
  #
  # A set of reasonable defaults is outlined below. This is not created by default as it should only
  # be created by a one Release. If you want to use the PodSecurityPolicy in the chart, create it in
  # the "master" Release and then leave it unset in all others. Set the `rbac.podSecurityPolicies`
  # in all non-"master" Releases.
  podSecurityPolicy: {}
    # # Add AppArmor and Seccomp annotations
    # # https://kubernetes.io/docs/concepts/policy/pod-security-policy/#apparmor
    # annotations:
    # spec:
    #   seLinux:
    #     rule: RunAsAny
    #   supplementalGroups:
    #     rule: 'MustRunAs'
    #     ranges:
    #       # Forbid adding the root group.
    #       - min: 1
    #         max: 65535
    #   fsGroup:
    #     rule: 'MustRunAs'
    #     ranges:
    #       # Forbid adding the root group.
    #       - min: 1
    #         max: 65535
    #   privileged: false
    #   allowPrivilegeEscalation: false
    #   runAsUser:
    #     rule: MustRunAsNonRoot
```

### Annotations

Ambassador is configured using Kubernetes Custom Resource Definitions (CRDs). If you are unable to use CRDs, Ambassador can also be configured using annotations on services. The `service.annotations` section of the values file contains commented out examples of [Ambassador Module](https://www.getambassador.io/reference/core/ambassador) and a global [TLSContext](https://www.getambassador.io/reference/core/tls) configurations which are typically created in the Ambassador service.

If you intend to use `service.annotations`, remember to include the `getambassador.io/config` annotation key as above.

### Prometheus Metrics

Using the Prometheus Exporter has been deprecated and is no longer recommended. You can now use `metrics.serviceMonitor.enabled` to create a `ServiceMonitor` from the chart if the [Prometheus Operator](https://github.com/coreos/prometheus-operator) has been installed on your cluster.

Please see Ambassador's [monitoring with Prometheus](https://www.getambassador.io/user-guide/monitoring/) docs for more information on using the `/metrics` endpoint for metrics collection.

### Specifying Values

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```console
$ helm install --wait my-release \
    --set adminService.type=NodePort \
    datawire/ambassador
```

Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example,

```console
$ helm install --wait my-release -f values.yaml datawire/ambassador
```

---

# Upgrading

## To 6.0.0

Introduces Ambassador Edge Stack being installed by default.

### Breaking changes

Ambassador Pro support has been removed in 6.0.0. Please [upgrade to the Ambassador Edge Stack](https://www.getambassador.io/user-guide/helm).

## To 5.0.0

### Breaking changes

**Note** If upgrading an existing helm 2 installation no action is needed, previously installed CRDs will not be modified.

- Helm 3 support for CRDs was added. Specifically, the CRD templates were moved to non-templated files in the `/crds` directory, and to keep Helm 2 support they are globbed from there by `/templates/crds.yaml`. However, because Helm 3 CRDs are not templated, the labels for new installations have necessarily changed

## To 4.0.0

The 4.0.0 chart contains a number of changes to the way Ambassador Pro is installed.

- Introduces the performance tuned and certified build of open source Ambassador, Ambassador core
- The license key is now stored and read from a Kubernetes secret by default
- Added `.Values.pro.licenseKey.secret.enabled` `.Values.pro.licenseKey.secret.create` fields to allow multiple releases in the same namespace to use the same license key secret.
- Introduces the ability to configure resource limits for both Ambassador Pro and it's redis instance
- Introduces the ability to configure additional `AuthService` options (see [AuthService documentation](https://www.getambassador.io/reference/services/auth-service/))
- The ambassador-pro-auth `AuthService` and ambassador-pro-ratelimit `RateLimitService` and now created as CRDs when `.Values.crds.enabled: true`
- Fixed misnamed selector for redis instance that failed in an edge case
- Exposes annotations for redis deployment and service

### Breaking changes

The value of `.Values.pro.image.tag` has been shortened to assume `amb-sidecar` (and `amb-core` for Ambassador core)
`values.yaml`
```diff
<3.0.0>
  image:
    repository: quay.io/datawire/ambassador_pro
-    tag: amb-sidecar-0.6.0

<4.0.0+>
  image:
    repository: quay.io/datawire/ambassador_pro
+    tag: 0.7.0
```

Method for creating a Kubernetes secret to hold the license key has been changed

`values.yaml`
```diff
<3.0.0>
-    secret: false
<4.0.0>
+    secret:
+      enabled: true
+      create: true
```

## To 3.0.0

### Service Ports

The way ports are assigned has been changed for a more dynamic method.

Now, instead of setting the port assignments for only the http and https, any port can be open on the load balancer using a list like you would in a standard Kubernetes YAML manifest.

`pre-3.0.0`
```yaml
service:
  http:
    enabled: true
    port: 80
    targetPort: 8080
  https:
    enabled: true
    port: 443
    targetPort: 8443
```

`3.0.0`
```yaml
service:
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: https
    port: 443
    targetPort: 8443
```

This change has also replaced the `.additionalTCPPorts` configuration. Additional TCP ports can be created the same as the http and https ports above.

### Annotations and `service_port`

The below Ambassador `Module` annotation is no longer being applied by default.

```yaml
getambassador.io/config: |
  ---
  apiVersion: ambassador/v1
  kind: Module
  name: ambassador
  config:
    service_port: 8080
```
This was causing confusion with the `service_port` being hard-coded when enabling TLS termination in Ambassador.

Ambassador has been listening on port 8080 for HTTP and 8443 for HTTPS by default since version `0.60.0` (chart version 2.2.0).

### RBAC and CRDs

A `ClusterRole` and `ClusterRoleBinding` named `{{release name}}-crd` will be created to watch for the Ambassador Custom Resource Definitions. This will be created regardless of the value of `scope.singleNamespace` since CRDs are created the cluster scope.

`rbac.namespaced` has been removed. For namespaced RBAC, set `scope.singleNamespace: true` and `rbac.enabled: true`.

`crds.enabled` will indicate that you are using CRDs and will create the rbac resources regardless of the value of `crds.create`. This allows for multiple deployments to use the CRDs.

## To 2.0.0

### Ambassador ID

ambassador.id has been removed in favor of setting it via an environment variable in `env`. `AMBASSADOR_ID` defaults to `default` if not set in the environment. This is mainly used for [running multiple Ambassadors](https://www.getambassador.io/reference/running#ambassador_id) in the same cluster.

| Parameter       | Env variables   |
| --------------- | --------------- |
| `ambassador.id` | `AMBASSADOR_ID` |

## Migrating from `datawire/ambassador` chart (chart version 0.40.0 or 0.50.0)

Chart now runs ambassador as non-root by default, so you might need to update your ambassador module config to match this.

### Timings

Timings values have been removed in favor of setting the env variables using `env´

| Parameter         | Env variables              |
| ----------------- | -------------------------- |
| `timing.restart`  | `AMBASSADOR_RESTART_TIME`  |
| `timing.drain`    | `AMBASSADOR_DRAIN_TIME`    |
| `timing.shutdown` | `AMBASSADOR_SHUTDOWN_TIME` |

### Single namespace

| Parameter          | Env variables                 |
| ------------------ | ----------------------------- |
| `namespace.single` | `AMBASSADOR_SINGLE_NAMESPACE` |

### Renamed values

Service ports values have changed names and target ports have new defaults.

| Previous parameter          | New parameter              | New default value |
| --------------------------- | -------------------------- | ----------------- |
| `service.enableHttp`        | `service.http.enabled`     |                   |
| `service.httpPort`          | `service.http.port`        |                   |
| `service.httpNodePort`      | `service.http.nodePort`    |                   |
| `service.targetPorts.http`  | `service.http.targetPort`  | `8080`            |
| `service.enableHttps`       | `service.https.enabled`    |                   |
| `service.httpsPort`         | `service.https.port`       |                   |
| `service.httpsNodePort`     | `service.https.nodePort`   |                   |
| `service.targetPorts.https` | `service.https.targetPort` | `8443`            |

### Exporter sidecar

Pre version `0.50.0` ambassador was using socat and required a sidecar to export statsd metrics. In `0.50.0` ambassador no longer uses socat and doesn't need a sidecar anymore to export its statsd metrics. Statsd metrics are disabled by default and can be enabled by setting environment `STATSD_ENABLED`, this will (in 0.50) send metrics to a service named `statsd-sink`, if you want to send it to another service or namespace it can be changed by setting `STATSD_HOST`

If you are using prometheus the chart allows you to enable a sidecar which can export to prometheus see the `prometheusExporter` values.
