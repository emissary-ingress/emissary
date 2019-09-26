# Consul Integration

[Consul](https://www.consul.io) is a widely used service mesh. Ambassador natively supports Consul for service discovery and end-to-end TLS (including mTLS between services). This capability is particularly useful when deploying Ambassador in so-called hybrid clouds, where applications are deployed on bare metal, VMs, and Kubernetes. In this environment, Ambassador can securely route over TLS to any application regardless of where it is deployed.

## Architecture Overview

In this architecture, Consul serves as the source of truth for your entire data center, tracking available endpoints, service configuration, and secrets for TLS encryption. New applications and services automatically register themselves with Consul using the Consul agent or API. When a request is sent through Ambassador, Ambassador sends the request to an endpoint based on the data in Consul.

![ambassador-consul](/doc-images/consul-ambassador.png)

## Getting started

In this guide, you will register a service with Consul and use Ambassador to dynamically route requests to that service based on Consul's service discovery data. If you already have Ambassador installed, you will just need to configure the `ConsulResolver` in step 3.

1. Install and configure Consul ([instructions](https://www.consul.io/docs/platform/k8s/index.html)). Consul can be deployed anywhere in your data center. 

   **Note** 
   - If you are using the [Consul Helm Chart](https://www.consul.io/docs/platform/k8s/helm.html) for installation, you must install the latest version of both the [Chart](https://github.com/hashicorp/consul-helm) and the Consul binary itself (configurable via the [`values.yaml`](https://www.consul.io/docs/platform/k8s/helm.html#configuration-values-) file). `git checkout` the most recent tag in the [consul-helm](https://github.com/hashicorp/consul-helm) repository to get the latest version of the Consul Helm chart.
   - If you would like to enable end-to-end TLS between all of your APIs in Kubernetes, you will need to set `connectInject.enabled: true` and `client.grpc: true` in the values.yaml file.

2. Deploy Ambassador. Note: If this is your first time deploying Ambassador, reviewing the [Ambassador quick start](/user-guide/getting-started) is strongly recommended.

   ```
   kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-rbac.yaml
   ```

   If you're on GKE, or haven't previously created the Ambassador service, please see the Quick Start.

3. Configure Ambassador to look for services registered to Consul by creating the `ConsulResolver`:

    ```yaml
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: ambassador
      annotations:
        getambassador.io/config: |
          ---
          apiVersion: ambassador/v1
          kind: ConsulResolver
          name: consul-dc1
          address: consul-server.default.svc.cluster.local:8500
          datacenter: dc1
    spec:
      type: LoadBalancer
      selector:
        service: ambassador
      ports:
      - port: 80
        targetPort: 8080
    ```

    This will tell Ambassador that Consul is a service discovery endpoint. Save the configuration to a file (e.g., `ambassador-service.yaml`, and apply this configuration with `kubectl apply -f ambassador-service.yaml`. For more information about resolver configuration, see the [resolver reference documentation](/reference/core/resolvers). (If you're using Consul deployed elsewhere in your data center, make sure the `address` points to your Consul FQDN or IP address).

## Routing to Consul Services

You'll now register a demo application with Consul, and show how Ambassador can route to this application using endpoint data from Consul. To simplify this tutorial, you'll deploy the application in Kubernetes, although in practice this application can be deployed anywhere in your data center (e.g., on VMs or bare metal). 

1. Deploy the QOTM demo application. The QOTM application contains code to automatically register itself with Consul, using the CONSUL_IP and POD_IP environment variables specified within the QOTM container spec.

    ```yaml
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: qotm
    spec:
      replicas: 1
      strategy:
        type: RollingUpdate
      selector:
        matchLabels:
          app: qotm
      template:
        metadata:
          labels:
            app: qotm
          annotations:
            "consul.hashicorp.com/connect-inject": "false"
        spec:
          containers:
          - name: qotm
            image: datawire/qotm:%qotmVersion%
            ports:
            - name: http-api
              containerPort: 5000
            env:
            - name: CONSUL_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.hostIP
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

    Save the above to a file called `qotm.yaml` and run `kubectl apply -f qotm.yaml`. This will register the QOTM pod as a Consul service with the name `qotm-consul` and the IP address of the QOTM pod. 

2. Verify the QOTM pod has been registered with Consul. You can verify the QOTM pod is registered correctly by accessing the Consul UI.

   ```shell
   kubectl port-forward service/consul-ui 8500:80
   ```

   Go to http://localhost:8500 from a web browser and you should see a service named `qotm-consul`. 

3. Create a `Mapping` for the `qotm-consul` service. 

   ```yaml
   ---
   apiVersion: v1
   kind: Service
   metadata:
     name: qotm-consul
     annotations:
       getambassador.io/config: |
         ---
         apiVersion: ambassador/v1
         kind: Mapping
         name: consul_qotm_mapping
         prefix: /qotm-consul/
         service: qotm-consul
         resolver: consul-dc1
         load_balancer: 
           policy: round_robin
   spec:
     ports:
     - name: http
       port: 80
   ```
Save the above YAML to a file named `qotm-mapping.yaml`, and use `kubectl apply -f qotm-mapping.yaml` to apply this configuration to your Kubernetes cluster. Note that in the above config:
   - `resolver` must be set to the `ConsulResolver` that you created in the previous step
   - `load_balancer` must be set to configure Ambassador to route directly to the QOTM application endpoint(s) that are retrieved from Consul.

4. Send a request to the `qotm-consul` API.

   ```shell
   curl http://$AMBASSADOR_IP/qotm-consul/

   {"hostname":"qotm-749c675c6c-hq58f","ok":true,"quote":"The last sentence you read is often sensible nonsense.","time":"2019-03-29T22:21:42.197663","version":"1.7"}
   ```

Congratulations! You're successfully routing traffic to the QOTM application, the location of which is registered in Consul.

## Encrypted TLS

Ambassador can also use certificates stored in Consul to originate encrypted TLS connections from Ambassador to the Consul service mesh. This requires the use of the Ambassador Consul connector. The following steps assume you've already set up Consul for service discovery, as detailed above.

1. The Ambassador Consul connector retrieves the TLS certificate issued by the Consul CA and stores it in a Kubernetes secret for Ambassador to use. Deploy the Ambassador Consul Connector with `kubectl`:

   ```
   kubectl apply -f https://www.getambassador.io/yaml/consul/ambassador-consul-connector.yaml
   ```
   
This will install into your cluster:
   - RBAC resources.
   - The Consul connector service.
   - A `TLSContext` named `ambassador-consul` to load the `ambassador-consul-connect` secret into Ambassador. 

2. Deploy a new version of the demo application, and configure it to inject the Consul sidecar proxy by setting `"consul.hashicorp.com/connect-inject"` to `true`. Note that in this version of the configuration, you do not have to configure environment variables for the location of the Consul server:

    ```yaml
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: qotm-mtls
    spec:
      replicas: 1
      strategy:
        type: RollingUpdate
      selector:
        matchLabels:
          app: qotm
      template:
        metadata:
          labels:
            app: qotm
          annotations:
            "consul.hashicorp.com/connect-inject": "true"
        spec:
          containers:
          - name: qotm
            image: datawire/qotm:%qotmVersion%
            ports:
            - name: http-api
              containerPort: 5000
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
   Copy this YAML in a file called `qotm-consul-mtls.yaml` and apply it to your cluster with `kubectl apply -f qotm-consul-mtls.yaml`.

   This will deploy a demo application called `qotm-mtls` with the Connect sidecar proxy. The Connect proxy will register the application with Consul, require TLS to access the application, and expose other [Consul Service Segmentation](https://www.consul.io/segmentation.html) features.

3. Verify the `qotm-mtls` application is registered in Consul by accessing the Consul UI on http://localhost:8500/ after running:

   ```
   kubectl port-forward service/consul-ui 8500:80
   ```

   You should see a service registered as `qotm-proxy`.

4. Create a `Mapping` to route to the `qotm-mtls-proxy` service in Consul

    ```yaml
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: qotm-consul-mtls
      annotations:
        getambassador.io/config: |
          ---
          apiVersion: ambassador/v1
          kind: Mapping
          name: consul_qotm_tls_mapping
          prefix: /qotm-consul-tls/
          service: qotm-proxy
          resolver: consul-dc1
          tls: ambassador-consul
          load_balancer:
            policy: round_robin
    spec:
      ports:
      - name: http
        port: 80
    ```
    - `resolver` must be set to the `ConsulResolver` created when configuring Ambassador
    - `tls` must be set to the `TLSContext` storing the Consul mTLS certificates (e.g. `ambassador-consul`)
    - `load_balancer` must be set to configure Ambassador to route directly to the application endpoint(s) that are retrieved from Consul

    Copy this YAML to a file named `qotm-consul-mtls-svc.yaml` and apply it to your cluster with `kubectl apply -f qotm-consul-mtls-svc.yaml`.

5. Send a request to the `/qotm-consul-tls/` API.

   ```
   curl $AMBASSADOR_IP/qotm-consul-tls/

   {"hostname":"qotm-6c6dc4f67d-hbznl","ok":true,"quote":"A principal idea is omnipresent, much like candy.","time":"2019-04-17T19:27:54.758361","version":"1.7"}
   ```

## More information

For more about Ambassador's integration with Consul, read the [service discovery configuration](/reference/core/resolvers) documentation.

See the [TLS documentation](/reference/core/tls#mtls-consul) for information on configuring the Ambassador Consul connector.
