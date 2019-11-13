# Consul Connect Integration with Ambassador Edge Stack

## Prerequisites

### Consul Connect

Installation and configuration of Consul Connect is outside of the scope of this document. Please refer to [Consul documentation](https://www.consul.io/docs/platform/k8s/index.html) for information on how to install and configure Consul Connect.

### Ambassador Edge Stack

Install and configure Ambassador. If you are using a cloud provider such as Amazon, Google, or Azure, you can type:

```
kubectl apply -f https://getambassador.io/yaml/ambassador/ambassador-rbac.yaml
kubectl apply -f https://getambassador.io/yaml/ambassador/ambassador-service.yaml
```

Note: If you are using GKE, you will need additional privileges:

```
kubectl create clusterrolebinding my-cluster-admin-binding --clusterrole=cluster-admin --user=$(gcloud info --format="value(config.account)")
```

For more detailed instructions on installing Ambassador Edge Stack, please see the [Ambassador Edge Stack installation guide](/user-guide/getting-started).

**Note:** If you have automatic sidecar injection enabled, ensure the `"consul.hashicorp.com/connect-inject":` annotation is set to `"false"` in the Ambassador Edge Stack deployment spec.

```yaml
spec:
  replicas: 1
  selector:
    matchLabels:
      service: ambassador
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        service: ambassador
      annotations:
        "consul.hashicorp.com/connect-inject": "false"
```

## 1. Install the Ambassador Pro Consul Connector

<div style="border: thick solid red"> </div>

Ambassador Pro integrates with Consul Connect via a sidecar service. This service does two things:

- Talks to Consul and registers Ambassador as a Consul Service
- Retrieves the TLS certificate issued by the Consul CA and stores it as a Kubernetes secret Ambassador will use to authenticate with upstream services.

Deploy the Ambassador Consul Connector via kubectl:

```
kubectl apply -f https://getambassador.io/yaml/consul/ambassador-consul-connector.yaml
```

## 2. Configure Ambassador

### Create the TLSContext
You will need to tell Ambassador to use the certificate issued by Consul for `mTLS` with upstream services. This is accomplished by configuring a `TLSContext` to store the secret.

  ```yaml
  ---
  apiVersion: getambassador.io/v1
  kind: TLSContext
  metadata:
    name: ambassador-consul
  spec:
    hosts: []
    secret: ambassador-consul-connect
  ```
  
### Configure Ambassador Mappings to use the TLSContext
Ambassador needs to be configured to originate TLS to upstream services. This is done by providing a `TLSContext` to your service `Mapping`.  

  ```yaml
  ---
  apiVersion: getambassador.io/v1
  kind: Mapping
  metadata:
    name: qotm-mapping
  spec:
    prefix: /qotm/
    tls: ambassador-consul
    service: https://qotm:443
  ```
  **Note:** All service mappings will need `tls: ambassador-consul` to authenticate with Connect-enabled upstream services.

## 3. Test the Ambassador Consul Connector
To test that the Ambassador Consul Connector is working, you will need to have a service running with a Connect Sidecar. The following configuration will create the QoTM service with a Connect sidecar.

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
        "consul.hashicorp.com/connect-inject": "true"
    spec:
      containers:
      - name: qotm
        image: datawire/qotm:$qotmVersion$
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
Put this YAML in a file called `qotm-deploy.yaml` and apply it with `kubectl`:

```
kubectl apply -f qotm-deploy.yaml
```

Now, you will need to configure a service for Ambassador to route requests to. The following service will:

- Create a `Mapping` to tell Ambassador to originate TLS using the `ambassador-consul` `TLSContext` configured earlier.
- Route requests to Ambassador to the Connect sidecar in the QoTM pod using the statically assigned Consul port: `20000`.

```yaml
---
apiVersion: getambassador.io/v1
kind: Mapping
metadata:
  name: qotm-mapping
spec:
  prefix: /qotm/
  tls: ambassador-consul
  service: https://qotm:443
---
apiVersion: v1
kind: Service
metadata:
  name: qotm
spec:
  type: NodePort
  selector:
    app: qotm
  ports:
  - port: 443
    name: https-qotm
    targetPort: 20000
```
Put this YAML in a file named `qotm-service.yaml` and apply it with `kubectl`.

```
kubectl apply -f qotm-service.yaml
```

Finally, test the service with cURL.

```
curl -v https://{AMBASSADOR-EXTERNAL-IP}/qotm/
```


