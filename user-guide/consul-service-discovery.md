# Consul Integration

[Consul](https://www.consul.io) is a widely used service mesh. Ambassador natively supports Consul for end-to-end TLS and service discovery. This capability is particularly useful when deploying Ambassador in so-called hybrid clouds, where applications are deployed in VMs, bare metal, and Kubernetes. In this environment, Ambassador can securely route to any application regardless where it is deployed over TLS.

## Getting started

**Note:** This integration is not yet shipping. For now, the development image of this integration is here: `quay.io/datawire/ambassador:flynn-dev-watt-8922add`.

In this guide, we will register a service with Consul and use Ambassador to dynamically route requests to that service based on Consul's service discovery data.

1. Install and configure Consul ([instructions](https://www.consul.io/docs/platform/k8s/index.html)). Consul can be deployed anywhere in your data center.

2. Download the standard Ambassador deployment YAML file:

   ```
   curl -o ambassador-rbac.yaml https://www.getambassador.io/yaml/ambassador/ambassador-rbac.yaml
   ```

3. Edit the deployment and set `AMBASSADOR_ENABLE_ENDPOINTS` to `true`:

   ```
   ...
    containers:
    - name: ambassador
      image: quay.io/datawire/ambassador:%version%
      resources:
        limits:
          cpu: 1
          memory: 400Mi
        requests:
          cpu: 200m
          memory: 100Mi
      env:
      - name: AMBASSADOR_NAMESPACE
        valueFrom:
          fieldRef:
            fieldPath: metadata.namespace
      - name: AMBASSADOR_ENABLE_ENDPOINTS
        value: true
      ports:
   ...
   ```
   
   This will enable [endpoint load balancing](/reference/core/load-balancer) in Ambassador, and is required for Consul.

4. Deploy Ambassador. Note: If this is your first time deploying Ambassador, reviewing the [Ambassador quick start](/user-guide/getting-started) is strongly recommended.

   ```
   kubectl apply -f ambassador-rbac.yaml
   ```

   If you're on GKE, or haven't previously created the Ambassador service, please see the Quick Start.

   Note: For now, you'll need to install using https://github.com/datawire/ambassador-docs/tree/consul-sd/yaml/consul/ambassador-consul-sd.yaml, which adds the necessary RBAC permissions.

5. Deploy the QOTM test service. This service will automatically register itself with Consul when deployed.

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

    Note: If reading this on GitHub, use version 1.6 for QOTM.

    ```
    kubectl apply -f qotm.yaml
    ```

    This will register the QOTM pod with Consul with the name `{QOTM_POD_NAME}-consul` and the IP address of the QOTM pod. 

6. Verify the QOTM pod has been registered with Consul. You can verify the QOTM pod is registered correctly by accessing the Consul UI.

   ```shell
   kubectl port-forward service/consul-ui 8500:80
   ```

   Go to http://localhost:8500 from a web browser and you should see a service named `qotm-XXXXXXXXXX-XXXXX-consul`. 


7. Create the `ConfigMap` to expose `qotm-consul` to Ambassador:

    ```yaml
    kind: ConfigMap
    apiVersion: v1
    metadata:
      name: consul-sd
      annotations:
        "getambassador.io/consul-resolver": "true"
    data:
      consulAddress: "consul-server:8500"
      datacenter: "dc1"
      service: "qotm-XXXXXXXXXX-XXXXX-consul"
    ```

    ```
    kubectl apply -f consul-cm.yaml
    ```

    Note that the `ConfigMap` will be replaced with a CRD for GA.


8. Create a `Mapping` for the `qotm-consul` service. Make sure you specify the `load_balancer` annotation to configure Ambassador to route directly to the endpoint(s) from Consul.

   ```yaml
   ---
   apiVersion: v1
   kind: Service
   metadata:
     name: consul-sd
     annotations:
       getambassador.io/config: |
         ---
         apiVersion: ambassador/v1
         kind: Mapping
         name: consul_qotm_mapping
         prefix: /qotm-consul/
         service: qotm-XXXXXXXXXX-XXXXX-consul
         load_balancer: 
           policy: round_robin
   spec:
     ports:
     - name: http
       port: 80
   ```

   ```
   kubectl apply -f consul-sd.yaml
   ```

9. Send a request to the `qotm-consul` API.

   ```shell
   curl http://$AMBASSADORURL/qotm-consul/

   {"hostname":"qotm-749c675c6c-hq58f","ok":true,"quote":"The last sentence you read is often sensible nonsense.","time":"2019-03-29T22:21:42.197663","version":"1.3"}
   ```

   Congratulations! You're successfully routing traffic to the QOTM service, which is registered in Consul.

## Encrypted TLS

