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
        image: quay.io/datawire/ambassador:0.35.0
        name: ambassador
     restartPolicy: Always
     securityContext:
       runAsUser: 8888
     serviceAccountName: ambassador
```

* Set the `service_port` element in the ambassador Module to 8080 (cleartext) or 8443 (TLS). This is the port that Ambassador will use to listen to incoming traffic. Note that any port number above 1024 will work; Ambassador will use 8080/8443 as its defaults in the future.

* Make sure that incoming traffic to Ambassador is configured to route to the `service_port`. If you're using the default Ambassador configuration, this means configuring the `targetPort` to point to the `service_port` above.

* If you are using `redirect_cleartext_from`, change the value of this field to point to your cleartext port (e.g., 8080) and set `service_port` to be your TLS port (e.g., 8443).

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

## Multiple Ambassadors in One Cluster

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
      apiVersion: ambassador/v0
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

Ambassador will then only use YAML objects that include an appropriate `ambassador_id` attribute. For example, if Ambassador is given the ID `ambassador-1` as above, then of these YAML objects, only the first two will be used:

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


