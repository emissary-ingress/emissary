# Linkerd 2 Integration

[Linkerd 2](https://www.linkerd.io) is a zero-config and ultra-lightweight service mesh. Ambassador Edge Stack natively supports Linkerd 2 for service discovery, end-to-end TLS (including mTLS between services), and (with Linkerd 2.8) multicluster operation.

## Architecture

Linkerd 2 is designed for simplicity, security, and performance. In the cluster, it runs a control plane in its own namespace and then injects sidecar proxy containers in every Pod that should be meshed.

Ambassador Edge Stack itself also needs to be interwoven or "meshed" with Linkerd 2, and then configured to add special Linkerd headers to requests that tell Linkerd 2 where to forward them. This ie because mTLS between services is automatically handled by the control plane and the proxies. Istio and Consul allow Ambassador to initiate mTLS connections to upstream services by grabbing a certificate from a Kubernetes Secret. However, Linkerd 2 does not work this way, so Ambassador must rely on Linkerd 2 for mTLS connections to upstream services. This means we want Linkerd 2 to inject its sidecar into Ambassador's pods, but not Istio and Consul.

Through that setup, Ambassador Edge Stack terminates external TLS as the gateway and traffic is then immediately wrapped into mTLS by Linkerd 2 again. Thus we have a full end-to-end TLS encryption chain.

## Getting started

In this guide, you will use Linkerd 2 Auto-Inject to mesh a service and use Ambassador Edge Stack to dynamically route requests to that service based on Linkerd 2's service discovery data. If you already have Ambassador Edge Stack installed, you will just need to install Linkerd 2 and deploy your service.

Setting up Linkerd 2 requires to install three components. The first is the CLI on your local machine, the second is the actual Linkerd 2 control plane in your Kubernetes Cluster. Finally, you have to inject your services' Pods with Linkerd Sidecars to mesh them.

1. Install and configure Linkerd 2 [instructions](https://linkerd.io/2/getting-started/). Follow the guide until Step 3. That should give you the CLI on your machine and all required pre-flight checks.

    In a nutshell, these steps boil down to the following:

    ```bash
    # install linkerd cli tool
    curl -sL https://run.linkerd.io/install | sh
    # add linkerd to your path
    export PATH=$PATH:$HOME/.linkerd2/bin
    # verify installation
    linkerd version
    ```

2. Now it is time to install Linkerd 2 itself. To do so execute the following command:

    ```bash
    linkerd install --ha | kubectl apply -f -
    ```

    This will install Linkerd 2 in high-availability mode for the control plane. This means the controller and other components are started multiple times. Since Linkerd 2.5 it is also made sure the components are split across different nodes, if possible.

    Note that this simple command automatically enables mTLS by default and registers the AutoInject Webhook with your Kubernetes API Server. You now have a production-ready Linkerd 2 setup rolled out into your cluster!

3. Deploy Ambassador Edge Stack.

   **Note:** If this is your first time deploying Ambassador Edge Stack, reviewing the Ambassador Edge Stack [getting started](../../tutorials/getting-started) is strongly recommended.

   ```
   kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-rbac.yaml
   ```
   
   If you're on GKE, or haven't previously created the Ambassador Edge Stack service, please see [the quick start guide](../../tutorials/getting-started).

4. Configure Ambassador Edge Stack to add Linkerd 2 Headers to requests.

    ```yaml
    ---
    apiVersion: getambassador.io/v2
    kind: Module
    metadata:
      name: ambassador
    spec:
      config:
        add_linkerd_headers: true
    ```

    This will tell Ambassador Edge Stack to add additional headers to each request forwarded to Linkerd 2 with information about where to route this request to. This is a general setting. You can also set `add_linkerd_headers` per [Mapping](../../topics/using/mappings#mapping-configuration).

## Routing to Linkerd 2 Services

You'll now register a demo application with Linkerd 2, and show how Ambassador Edge Stack can route to this application using endpoint data from Linkerd 2.

1. Enable [AutoInjection](https://linkerd.io/2/features/proxy-injection/) on the Namespace you are about to deploy to:
    ```yaml
    apiVersion: v1
    kind: Namespace
    metadata:
      name: default # change this to your namespace if you're not using 'default'
      annotations:
        linkerd.io/inject: enabled
    ```

    Save the above to a file called `namespace.yaml` and run `kubectl apply -f namespace.yaml`. This will enable the namespace to be handled by the AutoInjection Webhook of Linkerd 2. Every time something is deployed to that namespace, the deployment is passed to the AutoInject Controller and injected with the Linkerd 2 proxy sidecar automatically.

2. Deploy the QOTM demo application.

    ```yaml
    ---
    apiVersion: extensions/v1beta1
    kind: Deployment
    metadata:
      name: qotm
    spec:
      replicas: 1
      strategy:
        type: RollingUpdate
      template:
        metadata:
          labels:
            app: qotm
        spec:
          containers:
          - name: qotm
            image: docker.io/datawire/qotm:$qotmVersion$
            ports:
            - name: http-api
              containerPort: 5000
            env:
            - name: POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
            readinessProbe:
              httpGet:
                path: /health
                port: 5000
              initialDelaySeconds: 30
              periodSeconds: 3
            resources:
              limits:
                cpu: "0.1"
                memory: 100Mi
    ```

    Save the above to a file called `qotm.yaml` and deploy it with

    ```
    kubectl apply -f qotm.yaml
    ```

    Watch via `kubectl get pod -w` as the Pod is created. Note that it starts with `0/2` containers automatically, as it has been auto-injected by the Linkerd 2 Webhook.

3. Verify the QOTM pod has been registered with Linkerd 2. You can verify the QOTM pod is registered correctly by accessing the Linkerd 2 Dashboard.

   ```shell
   linkerd dashboard
   ```

   Your browser should automatically open the correct URL. Otherwise, note the output from the above command and open that in a browser of your choice.

4. Create a `Mapping` for the `qotm-Linkerd2` service.

   ```yaml
   ---
   apiVersion: getambassador.io/v2
   kind: Mapping
   metadata:
     name: linkerd2-qotm
   spec:
     prefix: /qotm-linkerd2/
     service: qotm-linkerd2
   ```

Save the above YAML to a file named `qotm-mapping.yaml`, and apply it with:
```
kubectl apply -f qotm-mapping.yaml
``` 
to apply this configuration to your Kubernetes cluster. Note that in the above config there is nothing special to make it work with Linkerd 2. The general config for Ambassador Edge Stack already adds Linkerd Headers when forwarding requests to the service mesh.

1. Send a request to the `qotm-Linkerd2` API.

   ```shell
   curl -L http://$AMBASSADOR_IP/qotm-Linkerd2/

   {"hostname":"qotm-749c675c6c-hq58f","ok":true,"quote":"The last sentence you read is often sensible nonsense.","time":"2019-03-29T22:21:42.197663","version":"1.7"}
   ```

Congratulations! You're successfully routing traffic to the QOTM application, the location of which is registered in Linkerd 2. The traffic to Ambassador Edge Stack is not TLS secured, but from Ambassador Edge Stack to the QOTM an automatic mTLS connection is being used.

If you now [configure TLS termination](../../topics/running/tls) in Ambassador Edge Stack, you have an end-to-end secured connection.

## Multicluster Operation

Linkerd 2.8 can support [multicluster operation](https://linkerd.io/2/features/multicluster/), where the Linkerd mesh transparently bridges from one cluster to another, allowing seamless access between the two. This works using the Linkerd "[service mirror controller](https://linkerd.io/2020/02/25/multicluster-kubernetes-with-service-mirroring/#step-1-service-discovery)" to discover services in the target cluster, and expose (mirror) them in the source cluster. Requests to mirrored services in the source cluster are transparently proxied via the Ambassador in the target cluster to the appropriate target service, using Linkerd's [automatic mTLS](https://linkerd.io/2/features/automatic-mtls/) to protect the requests in flight between clusters.

### Initial Multicluster Setup

1. Install Linkerd and Ambassador.
2. Manually inject the Ambassador deployment with Linkerd (even if you have AutoInject enabled):

  ```
  kubectl -n ambassador get deploy ambassador -o yaml | \
    linkerd inject \
    --skip-inbound-ports 80,443 \
    --require-identity-on-inbound-ports 4183 - | \
    kubectl apply -f -
  ```

   (It's important to require identity on the gateway port so that automatic mTLS works, but it's also important to let Ambassador handle its own ports. AutoInject can't handle this on its own.)

3. Configure Ambassador as normal for your application. Don't forget to set `add_linkerd_headers: true`!

At this point, your Ambassador installation should work fine with multicluster Linkerd as a source cluster: you can configure Linkerd to bridge to a target cluster, and all should be well.

### Using the Cluster as a Target Cluster

Allowing the Ambassador installation to serve as a target cluster requires explicitly giving permission for Linkerd to mirror services from the cluster, and explicitly telling Linkerd to use the Ambassador as the target gateway. 

1. Configure the target cluster Ambassador to allow insecure routing.

When Ambassador is running in a Linkerd mesh, Linkerd provides transport security, so connections coming in from the Linkerd in the source cluster will always be HTTP when they reach Ambassador. Therefore, the `Host` CRDs corresponding to services that you'll be accessing from the source cluster *must* be configured to `Route` insecure requests. More information on this topic is available in the [`Host` documentation](../../topics/running/host-crd); an example might be

```yaml
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: linkerd-host
spec:
  hostname: host.example.com
  acmeProvider:
    authority: none
  requestPolicy:
    insecure:
      action: Route
```

2. Configure the target cluster Ambassador to support Linkerd health checks.

Multicluster Linkerd does its own health checks beyond what Kubernetes does, so a `Mapping` is needed to allow Linkerd's health checks to succeed:

```yaml
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: public-health-check
  namespace: ambassador
spec:
  prefix: /-/ambassador/ready
  rewrite: /ambassador/v0/check_ready
  service: localhost:8877
  bypass_auth: true
```

When configuring Ambassador, Kubernetes is usually configured to run health checks directly against port 8877 -- however, that port is not meant to be exposed outside the cluster. The `Mapping` permits accessing the health check endpoint without directly exposing the port.

(The actual prefix in the `Mapping` is not terribly important, but it needs to match the metadata supplied to the service mirror controller, below.)

3. Configure the target cluster Ambassador for the service mirror controller.

This requires changes to the Ambassador's `deployment` and `service`. **For all of these commands, you will need to make sure your Kubernetes context is set to talk to the target cluster.**

- In the `deployment`, you need the `config.linkerd.io/enable-gateway` `annotation`:

```bash
$ kubectl -n ambassador patch deploy ambassador -p='
spec:
    template:
        metadata:
            annotations:
                config.linkerd.io/enable-gateway: "true"
'
```

- In the `service`, you need to provide appropriate named `port` definitions:

   - `mc-gateway` needs to be defined as `port` 4143
   - `mc-probe` needs to be defined as `port` 80, `targetPort` 8080 (or wherever Ambassador is listening)

```bash
$ kubectl -n ambassador patch svc ambassador --type='json' -p='[
        {"op":"add","path":"/spec/ports/-", "value":{"name": "mc-gateway", "port": 4143}},
        {"op":"replace","path":"/spec/ports/0", "value":{"name": "mc-probe", "port": 80, "targetPort": 8080}}
    ]'
```

- Finally, the `service` also needs its own set of `annotation`s:

```bash
$ kubectl -n ambassador patch svc ambassador -p='
metadata:
    annotations:
        mirror.linkerd.io/gateway-identity: ambassador.ambassador.serviceaccount.identity.linkerd.cluster.local
        mirror.linkerd.io/multicluster-gateway: "true"
        mirror.linkerd.io/probe-path: -/ambassador/ready
        mirror.linkerd.io/probe-period: "3"
'
```

(Here, the value of `mirror.linkerd.io/probe-path` must match the `prefix` using for the probe `Mapping` above.)

4. Configure individual exported services.

```bash
$ kubectl -n $namespace patch svc $service -p='
metadata:
    annotations:
        mirror.linkerd.io/gateway-name: ambassador
        mirror.linkerd.io/gateway-ns: 
'
```

This annotation will tell Linkerd that the given service can be reached via the Ambassador in the `ambassador` namespace.

5. Verify that all is well from the source cluster.

**For all of these commands, you'll need to set your Kubernetes context for the _source_ cluster.**

- First, check to make that the clusters are correctly linked:

```bash
$ linkerd check --multicluster
```

- Next, make sure that the Ambassador gateway shows up when listing active gateways:

```bash
$ linkerd multicluster gateways
```

At this point, all should be well!

## More information

For more about Ambassador Edge Stack's integration with Linkerd 2, read the [service discovery configuration](../../topics/running/resolvers) documentation.
