# Linkerd2 Integration

[Linkerd2](https://www.linkerd.io) is a zero-config and ultra-lightweight service mesh. Ambassador Edge Stack natively supports Linkerd2 for service discovery and end-to-end TLS (including mTLS between services).

## Architecture

Linkerd2 is designed for simplicity, security and performance. In the cluster it runs a control plane in its own namespace and then injects sidecar proxy containers in every Pod that should be meshed. mTLS between services is automatically handled by the control plane and the proxies.

Ambassador Edge Stack itself also needs to be meshed and then configured to add special linkerd headers to requests so as to tell Linkerd2 where to forward them.

Through that setup, Ambassador Edge Stack terminates external TLS as the gateway and traffic is then immediately wrapped into mTLS by Linkerd2 again. Thus we have a full end-to-end TLS encryption chain.

## Getting started

In this guide, you will use Linkerd2 Auto-Inject to mesh a service and use Ambassador Edge Stack to dynamically route requests to that service based on Linkerd2's service discovery data. If you already have Ambassador Edge Stack installed, you will just need to install Linkerd2 and deploy your service.

Setting up Linkerd2 requires to install three components. The first is the CLI on your local machine, the second is the actual Linkerd2 control plane in your Kubernetes Cluster. Finally you have to inject your services' Pods with Linkerd Sidecars to mesh them.

1. Install and configure Linkerd2 ([instructions](https://linkerd.io/2/getting-started/)). Follow the guide until Step 3. That should give you the CLI on your machine and all required pre-flight checks.

    In a nutshell these steps boil down to the following:

    ```bash
    # install linkerd cli tool
    curl -sL https://run.linkerd.io/install | sh
    # add linkerd to your path
    export PATH=$PATH:$HOME/.linkerd2/bin
    # verify installation
    linkerd version
    ```

2. Now it is time to install Linkerd2 itself. To do so execute the following command:

    ```bash
    linkerd install --ha | kubectl apply -f -
    ```

    This will install Linkerd2 in high-availability mode for the control plane. This means the controller and other components are started multiple times. Since Linkerd2 2.5 it is also made sure the components are split across different nodes, if possible.

    Note that this simple command automatically enables mTLS by default and registers the AutoInject Webhook with your Kubernetes API Server. You now have a production ready Linkerd2 setup rolled out into your cluster!

3. Deploy Ambassador Edge Stack.

   **Note:** If this is your first time deploying Ambassador Edge Stack, reviewing the [Ambassador Edge Stack quick start](/user-guide/getting-started) is strongly recommended.

   ```
   kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-rbac.yaml
   ```

   If you're on GKE, or haven't previously created the Ambassador Edge Stack service, please see the Quick Start.

4. Configure Ambassador Edge Stack to add Linkerd2 Headers to requests.

    ```yaml
    ---
    apiVersion: getambassador.io/v1
    kind: Module
    metadata:
      name: ambassador
    spec:
      config:
        add_linkerd_headers: true
    ```

    This will tell Ambassador Edge Stack to add additional headers to each request forwarded to Linkerd2 with information about where to route this request to. This is a general setting. You can also set `add_linkerd_headers` per [Mapping](https://www.getambassador.io/reference/mappings#mapping-configuration).

## Routing to Linkerd2 Services

You'll now register a demo application with Linkerd2, and show how Ambassador Edge Stack can route to this application using endpoint data from Linkerd2.

1. Enable [AutoInjection](https://linkerd.io/2/features/proxy-injection/) on the Namespace you are about to deploy to:
    ```yaml
    apiVersion: v1
    kind: Namespace
    metadata:
      name: default # change this to your namespace if you're not using 'default'
      annotations:
        linkerd.io/inject: enabled
    ```
    Save the above to a file called `namespace.yaml` and run `kubectl apply -f namespace.yaml`. This will enable the namespace to be handled by the AutoInjection Webhook of Linkerd2. Every time something is deployed to that namespace, the deployment is passed to the AutoInject Controller and injected with the Linkerd2 proxy sidecar automatically.

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
            image: datawire/qotm:$qotmVersion$
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
    Watch via `kubectl get pod -w` as the Pod is created. Note that it starts with `0/2` containers automatically, as it has been auto-injected by the Linkerd2 Webhook.

3. Verify the QOTM pod has been registered with Linkerd2. You can verify the QOTM pod is registered correctly by accessing the Linkerd2 Dashboard.

   ```shell
   linkerd dashboard
   ```

   You browser should automatically open the correct URL. Otherwise note the output from the above command and open that in a browser of your choice.

4. Create a `Mapping` for the `qotm-Linkerd2` service.

   ```yaml
   ---
   apiVersion: getambassador.io/v1
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
to apply this configuration to your Kubernetes cluster. Note that in the above config there is nothing special to make it work with Linkerd2. The general config for Ambassador Edge Stack already adds Linkerd Headers when forwarding requests to the service mesh.

1. Send a request to the `qotm-Linkerd2` API.

   ```shell
   curl http://$AMBASSADOR_IP/qotm-Linkerd2/

   {"hostname":"qotm-749c675c6c-hq58f","ok":true,"quote":"The last sentence you read is often sensible nonsense.","time":"2019-03-29T22:21:42.197663","version":"1.7"}
   ```

Congratulations! You're successfully routing traffic to the QOTM application, the location of which is registered in Linkerd2. The traffic to Ambassador Edge Stack is not TLS secured, but from Ambassador Edge Stack to the QOTM an automatic mTLS connection is being used.

If you now [configure TLS termination in Ambassador Edge Stack](/reference/core/tls), you have an end-to-end secured connection.

## More information

For more about Ambassador Edge Stack's integration with Linkerd2, read the [service discovery configuration](/reference/core/resolvers) documentation.
