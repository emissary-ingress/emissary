# Consul for Service Discovery

Ambassador supports using [Consul](https://consul.io) for service discovery. In this configuration, Consul keeps track of all endpoints. Ambassador synchronizes with Consul, and uses this endpoint information for routing purposes. This architecture is particularly useful when deploying Ambassador in environments where Kubernetes is not the only platform (e.g., you're running VMs).

## Configuration

FIXME: Add endpoints & namespaces to RBAC

1. Use the image `quay.io/datawire/ambassador:flynn-dev-watt-f16a585`.
1. Set `AMBASSADOR_ENABLE_ENDPOINTS` in your environment. 
2. Create a `configmap` to configure Ambassador:

```
kind: ConfigMap
apiVersion: v1
metadata:
  name: foo
  annotations:
    "getambassador.io/consul-resolver": "true"
data:
  consulAddress: "consul-server:8500"
  datacenter: "dc1"
  service: "qotm-consul"
```

The name can be any value. but the `consulAddress` must be correct for your Consul, and the `service` name is important.

3. Deploy the ConfigMap to the cluster:

```
kubectl create configmap consul-sd --from-file=bar-service.yaml
```

3. Deploy the Ambassador image above.
4. Register a service `qotm-consul` endpoint with Consul. You can exec into the Consul pod to do this.

```
kubectl exec -it consul-pod -- /bin/bash
```

```
curl -X PUT -d '{"Datacenter": "dc1", "Node": "qotm","Address": "10.39.251.30","Service": {"Service": "abc", "Port": 80}}' http://127.0.0.1:8500/v1/catalog/register
```

5. Add a `Mapping` that includes

```
---
apiVersion: v1
kind: Service
metadata:
  name: consul-search
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v1
      kind: Mapping
      name: consul_search_mapping
      prefix: /consul/
      service: abc
      load_balancer: 
        policy: round_robin
spec:
  ports:
  - name: http
    port: 80
```

(or use the `ring_hash` LB, whatever, the point is to turn on endpoint routing)

6. Try out your `Mapping`.
7. Try registering and deregistering `bar` endpoints through Consul.

## TODO

Still todo:

* Support TLS
* Switch from ConfigMap to CRD