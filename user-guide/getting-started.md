# Getting Started with Ambassador

Ambassador is a microservices API Gateway. We'll do a quick tour of Ambassador with a demo configuration, before walking through how to deploy Ambassador in Kubernetes with a custom configuration.

## 1. Running the demo configuration

By default, Ambassador uses a demo configuration to show some of its basic features. Get it running with Docker, and expose Ambassador on port 8080:

```shell
docker run -it -p 8080:80 --name=ambassador --rm datawire/ambassador:{VERSION} --demo
```

## 2. Ambassador's Diagnostics

Ambassador provides live diagnostics viewable with a web browser. While this would normally not be exposed to the public network, the Docker demo publishes the diagnostics service at the following URL:

`http://localhost:8080/ambassador/v0/diag/`

Some of the most important information - your Ambassador version, how recently Ambassador's configuration was updated, and how recently Envoy last reported status to Ambassador - is right at the top. The diagnostics overview can show you what it sees in your configuration map, and which Envoy objects were created based on your configuration.

## 3. The Quote of the Moment service

Since Ambassador is an API gateway, its primary purpose is to provide access to microservices. The demo is preconfigured with a mapping that connects the `/qotm/` resource to the "Quote of the Moment" service -- a demo service that supplies quotations. You can try it out here:

```shell
curl http://localhost:8080/qotm/
```

This request will route to the `qotm` service at `demo.getambassador.io`, and return a quote in a JSON object.

You can also see the mapping by clicking the `mapping-qotm.yaml` link from the diagnostic overview, or by opening

`http://localhost:8080/ambassador/v0/diag/mapping-qotm.yaml`

## 4. Authentication

On the diagnostic overview, you can also see that Ambassador is configured to do authentication -- click the `auth.yaml` link, or open

`http://localhost:8080/ambassador/v0/diag/auth.yaml`

for more here. Ambassador uses a demo authentication service at `demo.getambassador.io` to mediate access to the Quote of the Moment: simply getting a random quote is allowed without authentication, but to get a specific quote, you'll have to authenticate:

```shell
curl -v http://localhost:8080/qotm/quote/5
```

will return a 401, but

```shell
curl -v -u username:password http://localhost:8080/qotm/quote/5
```

will succeed. (Note that that's literally "username" and "password" -- the demo auth service is deliberately not very secure!)

Note that it's up to the auth service to decide what needs authentication -- teaming Ambassador with an authentication service can be as flexible or strict as you need it to be.

## 5. Configuring Ambassador

So far, we've used a demo configuration, and run everything in our local Docker instance. We'll now create a custom configuration for Ambassador that maps `/httpbin/` to `httpbin.org`. Create a `config.yaml` file with the following contents:

```yaml
---
apiVersion: ambassador/v0
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
rewrite: /qotm/
service: demo.getambassador.io
---
apiVersion: ambassador/v0
kind:  Mapping
name:  httpbin_mapping
prefix: /httpbin/
service: httpbin.org:80
host_rewrite: httpbin.org
```

(Note the `host_rewrite` attribute for the `httpbin_mapping` -- this forces the HTTP `Host` header, and is often a good idea when mapping to external services.)

We can deploy this configuration into a `ConfigMap` for Ambassador.

```shell
kubectl create configmap ambassador-config --from-file config.yaml
```

## 6. Deploying Ambassador in Kubernetes

Now, we need to deploy Ambassador in Kubernetes. Create a Kubernetes manifest in a file called `ambassador.yaml` that looks like the following:

```yaml
---
apiVersion: v1
kind: Service
metadata:
  labels:
    service: ambassador
  name: ambassador
spec:
  type: LoadBalancer
  ports:
  - name: ambassador
    port: 80
    targetPort: 80
  selector:
    service: ambassador
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: ambassador
spec:
  replicas: 2
  template:
    metadata:
      labels:
        service: ambassador
    spec:
      containers:
      - name: ambassador
        image: datawire/ambassador:{VERSION}
        imagePullPolicy: Always
        resources:
          limits:
            cpu: 1
            memory: 400Mi
          requests:
            cpu: 200m
            memory: 100Mi
        volumeMounts:
        - mountPath: /etc/ambassador-config
          name: config-map
      volumes:
      - name: config-map
        configMap:
          name: ambassador-config
      restartPolicy: Always
```

Then, deploy Ambassador into Kubernetes:

```
kubectl apply -f ambassador.yaml
```

To test things out, we'll need the external IP for Ambassador (it might take some time for this to be available):

```shell
kubectl get svc ambassador
```

which should, eventually, give you something like:

```
NAME         CLUSTER-IP      EXTERNAL-IP     PORT(S)        AGE
ambassador   10.11.12.13     35.36.37.38     80:31656/TCP   1m
```

You should now be able to use `curl` to both `qotm` and `httpbin` (don't forget the trailing `/`):

```shell
$ curl 35.36.37.38/qotm/
$ curl 35.36.37.38/httpbin/
```

Note that we did not expose the diagnostics port for Ambassador, since we don't want to expose it on the Internet. To view it, we'll need to get the name of one of the ambassador pods:

```
$ kubectl get pods
NAME                          READY     STATUS    RESTARTS   AGE
ambassador-3655608000-43x86   1/1       Running   0          2m
ambassador-3655608000-w63zf   1/1       Running   0          2m
```

Forwarding local port 8877 to one of the pods:

```
kubectl port-forward ambassador-3655608000-43x86 8877
```

will then let us view the diagnostics at http://localhost:8877/ambassador/v0/diag/.

## 7. Updating Ambassador configuration

To change the Ambassador configuration, you need to do two things:

1. Update your ConfigMap in Kubernetes. The following will regenerate and replace your `ambassador-config` ConfigMap using the current contents of your `conf.yaml` file:

```
kubectl create configmap ambassador-config --from-file conf.yaml -o yaml --dry-run | \
  kubectl replace -f -
```

2. Use Kubernetes to redeploy Ambassador, so that it can reread the ConfigMap:

```
kubectl patch deployment ambassador -p \
  "{\"spec\":{\"template\":{\"metadata\":{\"annotations\":{\"date\":\"`date +'%s'`\"}}}}}"
```

## 8. Next

We've just done a quick tour of some of the core features of Ambassador: diagnostics, routing, configuration, and authentication.

- Join us on [Gitter](https://gitter.im/datawire/ambassador);
- Learn how to [add authentication](auth-tutorial.md) to existing services; or
- Read about [configuring Ambassador](/reference/configuration.md).
