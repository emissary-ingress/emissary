# Consul for Service Discovery

Ambassador supports using [Consul](https://consul.io) for service discovery. In this configuration, Consul keeps track of all endpoints. Ambassador synchronizes with Consul, and uses this endpoint information for routing purposes. This architecture is particularly useful when deploying Ambassador in environments where Kubernetes is not the only platform (e.g., you're running VMs).

## Configuration Example:

**Note:** This integration is available starting with Ambassador `0.53.0`. For now, the development image of this integration is here: `quay.io/datawire/ambassador:flynn-dev-watt-f16a585`.

In this example, we will demo using Consul Service Discovery to expose APIs to Ambassador. For simplicity, we will do this with a Kubernetes Service.

1. Create the QoTM API and service:

    ```yaml
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: qotm
    spec:
      selector:
        app: qotm
      ports:
      - port: 80
        name: http-qotm
        targetPort: http-api
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
            image: datawire/qotm:1.4
            ports:
            - name: http-api
              containerPort: 5000
            env:
            - name: REQUEST_LIMIT
              value: "5"
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

2. Get the IP address of the QOTM service:

   ```shell
   kubectl get svc qotm

   NAME   TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
   qotm   ClusterIP   10.27.251.205   <none>        80/TCP    5h
   ```

3. Register the `qotm-service` endpoint with Consul

   - Use `kubectl exec` to start a shell in the Consul server pod running in your cluster

      ```
      kubectl exec -it consul-server-0 -- sh
      ```

   - Register a service with Consul using the Consul CLI

     ```
     consul services register -name=qotm-consul -address=10.27.251.205 -port=80
     ```

4. Create the `ConfigMap` to expose `qotm-consul` to Ambassador

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
      service: "qotm-consul"
    ```

    ```
    kubectl apply -f consul-cm.yaml
    ```

    Note that the `ConfigMap` will be replaced with a CRD for GA.

5. Set `AMBASSADOR_ENABLE_ENDPOINTS` to `true` in the Ambassador deployment and deploy Ambassador.

6. Create a `Mapping` for the `qotm-consul` service. Make sure you specify the `load_balancer` annotation to configure Ambassador to route directly to the endpoint(s) from Consul.

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
         service: qotm-consul
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

7. Send a request to the `qotm-consul` API.

   ```shell
   curl http://$AMBASSADORURL/qotm-consul/

   {"hostname":"qotm-749c675c6c-hq58f","ok":true,"quote":"The last sentence you read is often sensible nonsense.","time":"2019-03-29T22:21:42.197663","version":"1.3"}
   ```