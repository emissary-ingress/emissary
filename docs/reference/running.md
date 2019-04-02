# Running Ambassador

This section is intended for operators running Ambassador, and covers various aspects of deploying and configuring Ambassador in production.

## Ambassador and Kubernetes

Ambassador relies on Kubernetes for reliability, availability, and scalability. This means that features such as Kubernetes readiness and liveness probes, rolling updates, and the Horizontal Pod Autoscaling should be utilized to manage Ambassador.

## Default configuration

The default configuration of Ambassador includes default [resource limits](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#resource-requests-and-limits-of-pod-and-container), as well as [readiness and liveness probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/). These values should be adjusted for your specific environment.

## Running as non-root

Starting with Ambassador 0.35, we support running Ambassador as non-root. This is the recommended configuration, and will be the default configuration in future releases. We recommend you configure Ambassador to run as non-root as follows:

* Have Kubernetes run Ambassador as non-root. This may happen by default (e.g., OpenShift) or you can set a `securityContext` in your Deployment as shown below in this abbreviated example:

```yaml
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: ambassador
spec:
  replicas: 1
  template:
    metadata:
      labels:
        service: ambassador
    spec:
      containers:
        image: quay.io/datawire/ambassador:0.50.0
        name: ambassador
     restartPolicy: Always
     securityContext:
       runAsUser: 8888
     serviceAccountName: ambassador
```

* Set the `service_port` element in the ambassador Module to 8080 (cleartext) or 8443 (TLS). This is the port that Ambassador will use to listen to incoming traffic. Note that any port number above 1024 will work; Ambassador will use 8080/8443 as its defaults in the future.

* Make sure that incoming traffic to Ambassador is configured to route to the `service_port`. If you're using the default Ambassador configuration, this means configuring the `targetPort` to point to the `service_port` above.

* If you are using `redirect_cleartext_from`, change the value of this field to point to your cleartext port (e.g., 8080) and set `service_port` to be your TLS port (e.g., 8443).

## Changing the configuration directory

While running, Ambassador needs to use a directory within its container for generated configuration data. 
Normally this is `/ambassador`, but in some situations - especially if running as non-root - it may be necessary to
change to a different directory. To do so, set the environment variable `AMBASSADOR_CONFIG_BASE_DIR` to the full 
pathname of the directory to use, as shown below in this abbreviated example:

```yaml
env:
- name: AMBASSADOR_CONFIG_BASE_DIR
  value: /tmp/ambassador-config
```

With `AMBASSADOR_CONFIG_BASE_DIR` set as above, Ambassador will create and use the directory `/tmp/ambassador-config`
for its generated data. (Note that, while the directory will be created if it does not exist, attempts to turn an
existing file into a directory will fail.)

## Running as daemonset

Ambassador can be deployed as daemonset to have one pod per node in Kubernetes cluster. This setup up is especially helpful when you have Kubernetes cluster running on bare metal or private cloud. 

* Ideal scenario could be when you are running containers on Kubernetes along side with your non containerized applications running exposed via VIP using BIG-IP or similar products. In such cases, east-west traffic is routed based on iRules to certain set of application pools consisting of application or web servers. In this setup, along side of traditonal application servers, two or more Ambassador pods can also be part of the application pools. In case of failure there is atleast one Ambassdor pod available to BIG-IP and can take care of routing traffic to kubernetes cluster.

* In manifest files `kind: Deployment` needs to be updated to `kind: DaemonSet`  and  `replicas` should be removed in `spec`section.

## Namespaces

Ambassador supports multiple namespaces within Kubernetes. To make this work correctly, you need to set the `AMBASSADOR_NAMESPACE` environment variable in Ambassador's container. By far the easiest way to do this is using Kubernetes' downward API (this is included in the YAML files from `getambassador.io`):

```yaml
env:
- name: AMBASSADOR_NAMESPACE
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace          
```

Given that `AMBASSADOR_NAMESPACE` is set, Ambassador [mappings](/reference/mappings) can operate within the same namespace, or across namespaces. **Note well** that mappings will have to explicitly include the namespace with the service to cross namespaces; see the [mapping](/reference/mappings) documentation for more information.

If you only want Ambassador to only work within a single namespace, set `AMBASSADOR_SINGLE_NAMESPACE` as an environment variable.

```
env:
- name: AMBASSADOR_NAMESPACE
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace 
- name: AMBASSADOR_SINGLE_NAMESPACE
  value: "true"
```

If you are using Ambassador Pro, if you set `AMBASSADOR_NAMESPACE` or `AMBASSADOR_SINGLE_NAMESPACE`, you will need to set them in **both** containers in the deployment.

## `AMBASSADOR_ID`

Ambassador supports running multiple Ambassadors in the same cluster, without restricting a given Ambassador to a single namespace. This is done with the `AMBASSADOR_ID` setting. In the Ambassador module, set the `ambassador_id`, e.g.,

```yaml
apiVersion: v1
kind: Service
metadata:
  name: ambassador
  namespace: ambassador-1
  labels:
    app: ambassador
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v1
      kind:  Module
      name:  ambassador
      ambassador_id: ambassador-1
```

Then, assign each Ambassador pod a unique `AMBASSADOR_ID` with the environment variable as part of your deployment:

```yaml
env:
- name: AMBASSADOR_ID
  value: ambassador-1
```

If you are using Ambassador Pro, if you set `AMBASSADOR_ID`, you will need to set it in **both** containers in the deployment.

Ambassador will then only use YAML objects that include an appropriate `ambassador_id` attribute. For example, if Ambassador is given the ID `ambassador-1` as above, then of these YAML objects, only the first two will be used:

```yaml
---
apiVersion: ambassador/v1
kind:  Mapping
name:  mapping_used_1
ambassador_id: ambassador-1
prefix: /demo1/
service: demo1
---
apiVersion: ambassador/v1
kind:  Mapping
name:  mapping_used_2
ambassador_id: [ "ambassador-1", "ambassador-2" ]
prefix: /demo2/
service: demo2
---
apiVersion: ambassador/v1
kind:  Mapping
name:  mapping_skipped_1
prefix: /demo3/
service: demo3
---
apiVersion: ambassador/v1
kind:  Mapping
name:  mapping_skipped_2
ambassador_id: ambassador-2
prefix: /demo4/
service: demo4
```

The list syntax (shown in `mapping_used_2` above) permits including a given object in the configuration for multiple Ambassadors. In this case `mapping_used_2` will be included in the configuration for `ambassador-1` and also for `ambassador-2`.

**Note well that _any_ object can and should have an `ambassador_id` included** so, for example, it is _fully supported_ to use `ambassador_id` to qualify the `ambassador Module`, `TLS`, and `AuthService` objects. You will need to set Ambassador_id in all resources you want to use for Ambassador.

If no `AMBASSADOR_ID` is assigned to an Ambassador, it will use the ID `default`. If no `ambassador_id` is present in a YAML object, it will also use the ID `default`.

## `AMBASSADOR_VERIFY_SSL_FALSE`

By default, Ambassador will verify the TLS certificates provided by the Kubernetes API. In some situations, the cluster may be deployed with self-signed certificates. In this case, set `AMBASSADOR_VERIFY_SSL_FALSE` to `true` to disable verifying the TLS certificates.

## Reconfiguration Timing Configuration

Ambassador is constantly watching for changes to the service annotations. When changes are observed, Ambassador generates a new Envoy configuration and restarts the Envoy handling the heavy lifting of routing. Three environment variables provide control over the timing of this reconfiguration:

- `AMBASSADOR_RESTART_TIME` (default 15) sets the minimum number of seconds between restarts. No matter how often services are changed, Ambassador will never restart Envoy more frequently than this.

- `AMBASSADOR_DRAIN_TIME` (default 5) sets the number of seconds that the Envoy will wait for open connections to drain on a restart. Connections still open at the end of this time will be summarily dropped.

- `AMBASSADOR_SHUTDOWN_TIME` (default 10) sets the number of seconds that Ambassador will wait for the old Envoy to clean up and exit on a restart. **If Envoy is not able to shut down in this time, the Ambassador pod will exit.** If this happens, it is generally indicative of issues with restarts being attempted too often.

These environment variables can be set much like `AMBASSADOR_NAMESPACE`, above.

## Configuration From the Filesystem

If desired, Ambassador can be configured from YAML files in the directory `$AMBASSADOR_CONFIG_BASE_DIR/ambassador-config` (by default, `/ambassador/ambassador-config`, which is empty in the images built by Datawire). You could volume mount an external configuration directory here, for example, or use a custom Dockerfile to build configuration directly into a Docker image.

Note well that while Ambassador will read its initial configuration from this directory, configuration loaded from Kubernetes annotations will _replace_ this initial configuration. If this is not what you want, you will need to set the environment variable `AMBASSADOR_NO_KUBEWATCH` so that Ambassador will not try to update its configuration from Kubernetes resources.

Also note that the YAML files in the configuration directory must contain Ambassador resources, not Kubernetes resources with annotations.

## Log levels and debugging

Ambassador and Ambassador Pro support more verbose debugging levels. If using Ambassador, the [diagnostics](diagnostics) service has a button to enable debug logging. Be aware that if you're running Ambassador on multiple pods, the debug log levels are not enabled for all pods -- they are configured on a per-pod basis.

If using Ambassador Pro, you can adjust the log level by setting the `APP_LOG_LEVEL` environment variable; from least verbose to most verbose, the valid values are `error`, `warn`/`warning`, `info`, `debug`, and `trace`; the default is `info`.

## Ambassador Update Checks (Scout)

Ambassador integrates Scout, a service that periodically checks with Datawire servers to advise of available updates. Scout also sends anonymized usage data and the Ambassador version. This information is important to us as we prioritize test coverage, bug fixes, and feature development. Note that Ambassador will run regardless of the status of Scout (i.e., our uptime has zero impact on your uptime.)

We do not recommend you disable Scout, since we use this mechanism to notify users of new release (including critical fixes and security issues). This check can be disabled by setting the environment
variable `SCOUT_DISABLE` to `1` in your Ambassador deployment.
  
Each Ambassador installation generates a unique cluster ID based on the UID of its Kubernetes namespace and
its Ambassador ID: the resulting cluster ID is a UUID which cannot be used to reveal the namespace name nor
Ambassador ID itself. Ambassador needs RBAC permission to get namespaces for this purpose, as shown in the 
default YAML files provided by Datawire; if not granted this permission it will generate a UUID based only on
the Ambassador ID. To disable cluster ID generation entirely, set the environment variable `AMBASSADOR_CLUSTER_ID`
to a UUID that will be used for the cluster ID.

Unless disabled, Ambassador will also report the following anonymized information back to Datawire:

| Attribute                 | Type  | Description               |
| :------------------------ | :---- | :------------------------ |
| `cluster_count` | int | total count of clusters in use |
| `cluster_grpc_count` | int | count of clusters using GRPC upstream |
| `cluster_http_count` | int | count of clusters using HTTP or HTTPS upstream |
| `cluster_routing_envoy_rh_count` | int | count of clusters routing using Envoy `ring_hash` |
| `cluster_routing_envoy_rr_count` | int | count of clusters routing using Envoy `round_robin` |
| `cluster_routing_kube_count` | int | count of clusters routing using Kubernetes |
| `cluster_tls_count` | int | count of clusters originating TLS |
| `custom_ambassador_id` | bool | has the `ambassador_id` been changed from 'default'? |
| `custom_diag_port` | bool | has the diag port been changed from 8877? |
| `custom_listener_port` | bool | has the listener port been changed from 80/443? |
| `diagnostics` | bool | is the diagnostics service enabled? |
| `endpoint_grpc_count` | int | count of endpoints to which Ambassador will originate GRPC |
| `endpoint_http_count` | int | count of endpoints to which Ambassador will originate HTTP or HTTPS |
| `endpoint_resource_total` | int | total count of `Endpoints` resources loaded from Kubernetes |
| `endpoint_routing` | bool | is endpoint routing enabled? |
| `endpoint_routing_envoy_rh_count` | int | count of endpoints being routed using Envoy `ring_hash` |
| `endpoint_routing_envoy_rr_count` | int | count of endpoints being routed using Envoy `round_robin` |
| `endpoint_routing_kube_count` | int | count of endpoints being routed using Kubernetes |
| `endpoint_tls_count` | int | count of endpoints to which Ambassador will originate TLS |
| `extauth` | bool | is extauth enabled? |
| `extauth_allow_body` | bool | will Ambassador send the body to extauth? |
| `extauth_host_count` | int | count of extauth hosts in use |
| `extauth_proto` | str | extauth protocol in use ('http', 'grpc', or `null` if not active) |
| `group_canary_count` | int | count of Mapping groups that include more than one Mapping |
| `group_count` | int | total count of Mapping groups in use (length of the route table) |
| `group_header_match_count` | int | count of groups using header matching (including `host` and `method`) |
| `group_host_redirect_count` | int | count of groups using host_redirect |
| `group_host_rewrite_count` | int | count of groups using host_rewrite |
| `group_http_count` | int | count of HTTP Mapping groups |
| `group_precedence_count` | int | count of groups that explicitly set the precedence of the group |
| `group_regex_header_count` | int | count of groups using regex header matching |
| `group_regex_prefix_count` | int | count of groups using regex prefix matching |
| `group_shadow_count` | int | count of groups using shadows |
| `group_tcp_count` | int | count of TCP Mapping groups |
| `listener_count` | int | count of active listeners (1 unless `redirect_cleartext_from` or TCP Mappings are in use) |
| `liveness_probe` | bool | are liveness probes enabled? |
| `ratelimit` | bool | is rate limiting in use? |
| `ratelimit_custom_domain` | bool | has the rate limiting domain been changed from 'ambassador'? |
| `ratelimit_data_plane_proto` | bool | is rate limiting using the data plane proto? |
| `readiness_probe` | bool | are readiness probes enabled? |
| `request_4xx_count` | int | lower bound for how many requests have gotten a 4xx response | 
| `request_5xx_count` | int | lower bound for how many requests have gotten a 5xx response | 
| `request_bad_count` | int | lower bound for how many requests have failed (either 4xx or 5xx) | 
| `request_elapsed` | float | seconds over which the request_ counts are valid | 
| `request_hr_elapsed` | string | human-readable version of `request_elapsed` (e.g. "3 hours 35 minutes 20 seconds" | 
| `request_ok_count` | int | lower bound for how many requests have succeeded (not a 4xx or 5xx) | 
| `request_total_count` | int | lower bound for how many requests were handled in total | 
| `statsd` | bool | is statsd enabled? |
| `tls_origination_count` | int | count of TLS origination contexts |
| `tls_termination_count` | int | count of TLS termination contexts |
| `tls_using_contexts` | bool | is the old TLS module in use? |
| `tls_using_module` | bool | are new TLSContext resources in use? |
| `tracing` | bool | is tracing in use? |
| `tracing_driver` | str | tracing driver in use ('zipkin', 'lightstep', or `null` if not active) |
| `use_proxy_proto` | bool | is the `PROXY` protocol in use? |
| `use_remote_address` | bool | is Ambassador honoring remote addresses? |
| `x_forwarded_proto_redirect` | bool | is Ambassador redirecting based on `X-Forwarded-Proto`? |
| `xff_num_trusted_hops` | int | what is the count of trusted hops for `X-Forwarded-For`? | 

The `request_*` counts are always incremental: they contain only information about the last `request_elapsed` seconds.
Additionally, they only provide a lower bound -- notably, if an Ambassador pod crashes or exits, no effort is made to
ship out a final update, so it's very easy for counts to never be reported.   

To completely disable feature reporting, set the environment variable `AMBASSADOR_DISABLE_FEATURES` to any non-empty
value.

