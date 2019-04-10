# Consul for Service Discovery

Ambassador supports using [Consul](https://consul.io) for service discovery. In this configuration, Consul keeps track of all endpoints. Ambassador synchronizes with Consul, and uses this endpoint information for routing purposes. This architecture is particularly useful when deploying Ambassador in environments where Kubernetes is not the only platform (e.g., you're running VMs).

## Configuration Example:

**Note:** This integration is not yet shipping. For now, the development image of this integration is here: `quay.io/datawire/ambassador:flynn-dev-watt-8922add`.

In this example, we will demo using Consul Service Discovery to expose APIs to Ambassador. For simplicity, we have created a QoTM API that automatically registers itself as service with Consul.

1. Install Ambassador with the YAML here: https://github.com/datawire/ambassador-docs/tree/consul-sd/yaml/consul/ambassador-consul-sd.yaml

   This will install Ambassador with the image above with the proper RBAC permissions and configure the Ambassador service. 

2. Create the QoTM API (if you're reading this in GitHub, use version 1.6 for `qotmVersion` below:)

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

    ```
    kubectl apply -f qotm.yaml
    ```

    This will register the qotm pod with Consul with the name `{QOTM_POD_NAME}-consul` and the IP address of the qotm pod. 

2. Verify the QOTM pod has been registered with Consul.

   You can verify the qotm pod is registered correctly by accessing the Consul UI.

   ```shell
   kubectl port-forward service/consul-ui 8500:80
   ```

   Go to http://localhost:8500 from a web browser and you should see a service named `qotm-XXXXXXXXXX-XXXXX-consul`. 


3. Create the `ConfigMap` to expose `qotm-consul` to Ambassador

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

4. Set `AMBASSADOR_ENABLE_ENDPOINTS` to `true` in the Ambassador deployment and deploy Ambassador.

5. Create a `Mapping` for the `qotm-consul` service. Make sure you specify the `load_balancer` annotation to configure Ambassador to route directly to the endpoint(s) from Consul.

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

6. Send a request to the `qotm-consul` API.

   ```shell
   curl http://$AMBASSADORURL/qotm-consul/

   {"hostname":"qotm-749c675c6c-hq58f","ok":true,"quote":"The last sentence you read is often sensible nonsense.","time":"2019-03-29T22:21:42.197663","version":"1.3"}
   ```