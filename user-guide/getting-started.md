# Getting Started with Ambassador

Ambassador is an API Gateway for microservices. We'll demo the very basics here.


## 1. The QOTM Service


We'll start by deploying a sample "Quote of the Moment" service (`qotm`) to demo the gateway.

```shell
kubectl apply -f https://www.getambassador.io/yaml/demo/demo-qotm.yaml
```

## 2. Configure Routes

In Ambassador, we configure routes with a YAML file. Create a file called `mapping-qotm.yaml` with the following contents:

```yaml
---
apiVersion: ambassador/v0
kind: Mapping
name: qotm_mapping
prefix: /qotm/
service: qotm
```

## 3. Create a ConfigMap

Ambassador expects to find its configuration in a Kubernetes `ConfigMap` named `ambassador-config`. You can create that from the `mapping-qotm.yaml` file with

```shell
kubectl create configmap ambassador-config --from-file mapping-qotm.yaml
```

## 4. Start Ambassador

At this point we can start an HTTP-only Ambassador (obviously, in the real world, you'd use TLS):

```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador.yaml
```

## 5. Test It Out!

To test things out, we'll need the external IP for Ambassador (it might take some time for this to be available):

```shell
kubectl get svc ambassador
```

which should, eventually, give you something like

```
NAME         CLUSTER-IP      EXTERNAL-IP     PORT(S)        AGE
ambassador   10.11.12.13     35.36.37.38     80:31656/TCP   1m
```

which we can use to fetch a Quote of the Moment:

```shell
curl 35.36.37.38/qotm/
```

This should be routed to the `qotm` service per the mapping we created at the start, and return something like

```json
{
  "hostname": "qotm-2399866569-9q4pz",
  "msg": "QotM health check OK",
  "ok": true,
  "time": "2017-09-15T04:09:51.897241",
  "version": "1.1"
}
```

That's the basics. For more:

- Join us on [Gitter](https://gitter.im/datawire/ambassador);
- Take a [deeper dive into this demo](demo-in-detail.md); or
- Dig into more about [running Ambassador](running.md).
