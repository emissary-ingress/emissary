# Running Ambassador

This section is intended for operators running Ambassador, and covers various aspects of deploying and configuring Ambassador in production.

## Ambassador and Kubernetes

Ambassador relies on Kubernetes for reliability, availability, and scalability. This means that features such as Kubernetes readiness and liveness probes, rolling updates, and the Horizontal Pod Autoscaling should be utilized to manage Ambassador.

## Default configuration

The default configuration of Ambassador includes default [resource limits](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#resource-requests-and-limits-of-pod-and-container), as well as [readiness and liveness probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/). These values should be adjusted for your specific environment.

The default configuration also includes a `statsd` sidecar for collecting and forwarding StatsD statistics to your metrics infrastructure. If you are not collecting metrics, you should delete the `statsd` sidecar.

## Namespaces

Ambassador supports multiple namespaces within Kubernetes. To make this work correctly, you need to set the `AMBASSADOR_NAMESPACE` environment variable in Ambassador's container. By far the easiest way to do this is using Kubernetes' downward API (this is included in the YAML files from `getambassador.io`):

```yaml
env:
- name: AMBASSADOR_NAMESPACE
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace          
```

Given that `AMBASSADOR_NAMESPACE` is set, Ambassador [mappings](reference/mappings) can operate within the same namespace, or across namespaces. **Note well** that mappings will have to explicitly include the namespace with the service to cross namespaces; see the [mapping](reference/mappings) documentation for more information.

If you only want Ambassador to only work within a single namespace, set `AMBASSADOR_SINGLE_NAMESPACE` as an environment variable.

## Multiple Ambassadors in One Cluster

If you need to run multiple Ambassadors in one cluster, but you don't want to restrict a given Ambassador to a single namespace, you can assign each Ambassador a unique `AMBASSADOR_ID` using the environment:

```yaml
env:
- name: AMBASSADOR_ID
  value: ambassador-1
```

and then the Ambassador will only use YAML objects that include an appropriate `ambassador_id` attribute. For example, if Ambassador is given the ID `ambassador-1` as above, then of these YAML objects, only the first two will be used:

```yaml
---
apiVersion: ambassador/v0
kind:  Mapping
name:  mapping_used_1
ambassador_id: ambassador-1
prefix: /demo1/
service: demo1
---
apiVersion: ambassador/v0
kind:  Mapping
name:  mapping_used_2
ambassador_id: [ "ambassador-1", "ambassador-2" ]
prefix: /demo2/
service: demo2
---
apiVersion: ambassador/v0
kind:  Mapping
name:  mapping_skipped_1
prefix: /demo3/
service: demo3
---
apiVersion: ambassador/v0
kind:  Mapping
name:  mapping_skipped_2
ambassador_id: ambassador-2
prefix: /demo4/
service: demo4
```

The list syntax (shown in `mapping_used_2` above) permits including a given object in the configuration for multiple Ambassadors. In this case `mapping_used_2` will be included in the configuration for `ambassador-1` and also for `ambassador-2`.

**Note well that _any_ object can have an `ambassador_id` included** so, for example, it is _fully supported_ to use `ambassador_id` to qualify the `ambassador Module`, `TLS`, and `AuthService` objects.

If no `AMBASSADOR_ID` is assigned to an Ambassador, it will use the ID `default`. If no `ambassador_id` is present in a YAML object, it will also use the ID `default`.

## Reconfiguration Timing Configuration

Ambassador is constantly watching for changes to the service annotations. When changes are observed, Ambassador generates a new Envoy configuration and restarts the Envoy handling the heavy lifting of routing. Three environment variables provide control over the timing of this reconfiguration:

- `AMBASSADOR_RESTART_TIME` (default 15) sets the minimum number of seconds between restarts. No matter how often services are changed, Ambassador will never restart Envoy more frequently than this.

- `AMBASSADOR_DRAIN_TIME` (default 5) sets the number of seconds that the Envoy will wait for open connections to drain on a restart. Connections still open at the end of this time will be summarily dropped.

- `AMBASSADOR_SHUTDOWN_TIME` (default 10) sets the number of seconds that Ambassador will wait for the old Envoy to clean up and exit on a restart. **If Envoy is not able to shut down in this time, the Ambassador pod will exit.** If this happens, it is generally indicative of issues with restarts being attempted too often.

These environment variables can be set much like `AMBASSADOR_NAMESPACE`, above.

## Diagnostics

If Ambassador is not routing your services as you'd expect, your first step should be the Ambassador Diagnostics service. This is exposed on port 8877 by default. You'll need to use `kubectl port-forward` for access, e.g.,

```shell
kubectl port-forward ambassador-xxxx-yyy 8877
```

where you'll have to fill in the actual pod name of one of your Ambassador pods (any will do). Once you have that, you'll be able to point a web browser at

`http://localhost:8877/ambassador/v0/diag/`

for the diagnostics overview.

![Diagnostics](/images/diagnostics.png)

 Some of the most important information - your Ambassador version, how recently Ambassador's configuration was updated, and how recently Envoy last reported status to Ambassador - is right at the top. The diagnostics overview can show you what it sees in your configuration map, and which Envoy objects were created based on your configuration.

If needed, you can get JSON output from the diagnostic service, instead of HTML:

`curl http://localhost:8877/ambassador/v0/diag/?json=true`

## Troubleshooting

If the diagnostics service does not provide sufficient information, Kubernetes and Envoy provide additional debugging information.

If Ambassador isn't working at all, start by looking at the data from the following:

* `kubectl describe pod <ambassador-pod>` will give you a list of all events on the Ambassador pod
* `kubectl logs <ambassador-pod> ambassador` will give you a log from Ambassador itself

If you need additional help, feel free to join our [Gitter channel](https://gitter.im/datawire/ambassador) with the above information (along with your Kubernetes manifest).

You can also increase the debug of Envoy through the button in the diagnostics panel. Turn on debug logging, issue a request, and capture the log output from the Ambassador pod using `kubectl logs` as described above.
