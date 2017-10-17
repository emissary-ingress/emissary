# Getting Started with Ambassador

Ambassador is an API Gateway for microservices. We'll demo the very basics here.

## 1. Running the Demo

You can use `docker run` to start Ambassador with a default configuration that can route to some Datawire cloud services as a demonstration:

```shell
docker run -it -p 8080:80 --name=ambassador --rm datawire/ambassador:{VERSION}
```

will start an Ambassador running in Docker that you can talk to on `localhost` port 8080.

## 2. Ambassador's Diagnostics

Ambassador provides live diagnostics viewable with a web browser. While this would normally not be exposed to the public network, the Docker demo makes the diagnostics available at

`http://localhost:8080/ambassador/v0/diag/`

Some of the most important information - your Ambassador version, how recently Ambassador's configuration was updated, and how recently Envoy last reported status to Ambassador - is right at the top. The diagnostics overview can show you what it sees in your configuration map, and which Envoy objects were created based on your configuration.

## 3. The Quote of the Moment service

Since Ambassador is an API gateway, its primary purpose is to provide access to microservices. The demo is preconfigured with a mapping that connects the `/qotm/` resource to the "Quote of the Moment" service -- a demo service that supplies (usually surreal) quotations.

You can see the mapping by clicking the `mapping-qotm.yaml` link from the diagnostic overview, or by opening

`http://localhost:8080/ambassador/v0/diag/mapping-qotm.yaml`

To _use_ the mapping to get a Quote of the Moment, just

```shell
curl http://localhost:8080/qotm/
```

This will be routed to the `qotm` service at `demo.getambassador.io`, and should return something like

```json
{
  "hostname": "qotm-1827164760-180sg",
  "ok": true,
  "quote": "Utter nonsense is a storyteller without equal.",
  "time": "2017-10-16T18:53:13.265612",
  "version": "1.1"
}
```

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

## 5. Running the Demo in Kubernetes

Since the Ambassador Docker image contains its configuration already, running it in Kubernetes is simple. You can easily deploy it with the following YAML:

```shell
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
  replicas: 1
  template:
    metadata:
      labels:
        service: ambassador
    spec:
      containers:
      - name: ambassador
        image: datawire/ambassador:0.14.2-draft
        imagePullPolicy: Always
        resources:
          limits:
            cpu: 1
            memory: 400Mi
          requests:
            cpu: 200m
            memory: 100Mi
      restartPolicy: Always
``` 

To test things out, we'll need the external IP for Ambassador (it might take some time for this to be available):

```shell
kubectl get svc ambassador
```

which should, eventually, give you something like

```
NAME         CLUSTER-IP      EXTERNAL-IP     PORT(S)        AGE
ambassador   10.11.12.13     35.36.37.38     80:31656/TCP   1m
```

All the tests from above should work fine, but instead of `localhost:8080`, you'll use the `EXTERNAL-IP` from the `kubectl` output. For example, using the `EXTERNAL-IP` from above, you could grab a a Quote of the Moment:

```shell
curl 35.36.37.38/qotm/
```

or see the diagnostic overview at `http://35.36.37.38/ambassador/v0/diag/`.

## 6. Using Your Own Configuration

Ambassador's configuration is supplied by YAML files installed in `/etc/ambassador-config` inside its container. You can check out the "[Configuring Ambassador](reference/configuration.md)" reference for the gory details, but here's the one-file version of the default configuration to get started with:

```yaml
---
apiVersion: ambassador/v0
kind:  Module
name:  authentication
config:
  auth_service: "demo.getambassador.io"
  path_prefix: "/auth/v0"
  allowed_headers:
  - "x-extauth-required"
  - "x-authenticated-as"
  - "x-qotm-session"
---
apiVersion: ambassador/v0
kind:  Mapping
name:  diag_mapping
prefix: /ambassador/
rewrite: /ambassador/
service: 127.0.0.1:8877
---
apiVersion: ambassador/v0
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
rewrite: /qotm/
service: demo.getambassador.io
```

The easiest way to run Ambassador with a custom configuration is simply to use our Docker image as a base, but to replace our default configuration with your own. If you put the above YAML into a file named `config.yaml`, the following `Dockerfile` would do the trick:

```shell
FROM datawire/ambassador:{VERSION}
RUN rm /etc/ambassador-config/*
COPY config.yaml /etc/ambassador-config
```

Of course, this isn't very flexible: you'll need to rebuild the image for any configuration change. With Kubernetes, you can do better by mounting the configuration from a `ConfigMap`. 

Using `config.yaml` from above, we can create a `ConfigMap` by running

```shell
kubectl create configmap ambassador-config --from-file config.yaml
```

and then update the Ambassador deployment with the following YAML to mount the config from our newly-created `ConfigMap`:

```yaml
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: ambassador
spec:
  replicas: 3
  template:
    metadata:
      labels:
        service: ambassador
    spec:
      containers:
      - name: ambassador
        image: datawire/ambassador:0.14.2-draft
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

That's the basics. For more:

- Join us on [Gitter](https://gitter.im/datawire/ambassador);
- Learn how to [add authentication](auth-tutorial.md) to existing services; or
- Dig into more about [running Ambassador](running.md).
