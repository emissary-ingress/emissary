# Consul for Service Discovery

Ambassador supports using [Consul](https://consul.io) for service discovery. In this configuration, Consul keeps track of all endpoints. Ambassador synchronizes with Consul, and uses this endpoint information for routing purposes. This architecture is particularly useful when deploying Ambassador in environments where Kubernetes is not the only platform (e.g., you're running VMs).

## Configuration

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
  consulAddress: "consul:8500"
  datacenter: "dc1"
  service: "bar"
```

The name can be any value. but the `consulAddress` must be correct for your Consul, and the `service` name is important.

3. Deploy the ConfigMap to the cluster:

```
kubectl create configmap consul-sd --from-file=bar-service.yaml
```

3. Deploy the Ambassador image above.
4. Register a service `bar` endpoint with Consul.
5. Add a `Mapping` that includes

```service: bar
load_balancer: 
  policy: round_robin
```

(or use the `ring_hash` LB, whatever, the point is to turn on endpoint routing)

6. Try out your `Mapping`.
7. Try registering and deregistering `bar` endpoints through Consul.

## TODO

Still todo:

* Support TLS
* Switch from ConfigMap to CRD