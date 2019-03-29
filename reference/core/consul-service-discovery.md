# Consul for Service Discovery

Ambassador supports using [Consul](https://consul.io) for service discovery. In this configuration, Consul keeps track of all endpoints. Ambassador synchronizes with Consul, and uses this endpoint information for routing purposes. This architecture is particularly useful when deploying Ambassador in environments where Kubernetes is not the only platform (e.g., you're running VMs).

## Configuration

**Note:** This integration is available starting with Ambassador `0.53.0`.

1. Set `AMBASSADOR_ENABLE_ENDPOINTS` in your environment. 

2. Configure Ambassador to use Consul Service Discovery with a `ConfigMap`:

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
      service: "consul-service"
    ```
    - `consulAddress`: The hostname and port Ambassador will use to reach Consul
    - `datacenter`: Consul data center where to `service` is located.
    - `service`: The Consul service you would like to route to.

3. Deploy the ConfigMap to the cluster:

   ```
   kubectl apply -f  consul-cm.yaml
   ```

4. Deploy or restart Ambassador so it picks up the configuration change.

5. Configure a `Mapping` to route to the `service` exposed by the `ConfigMap` above.

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
         name: consul_search_mapping
         prefix: /consul/
         service: consul-service
         load_balancer: 
           policy: round_robin
   spec:
     ports:
     - name: http
       port: 80
   ```

Requests to http://{AMBASSADORURL}/consul/ will now be routed to the service registered in Consul.


## Example

In this example, we will demo using Consul Service Discovery to expose APIs to Ambassador with a Kubernetes service.

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

2. Get the IP address of the qotm service

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

5. Set `AMBASSADOR_ENABLE_ENDPOINTS` to `true` in the Ambassador deployment and deploy Ambassador

6. Create a `Mapping` for the `qotm-consul` service

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
         name: consul_search_mapping
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