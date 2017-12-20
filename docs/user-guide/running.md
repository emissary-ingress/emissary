# Running Ambassador

The simplest way to run Ambassador is **not** to build it! Instead, just use the YAML files published at https://www.getambassador.io, and start by deciding whether you want to use TLS or not. (If you want more information on TLS, check out our [TLS Overview](../how-to/tls-termination.md).) It's possible to switch this later, but it's a pain, and may well involve mucking about with your DNS and such to do it, so it's better to decide up front.

### Creating the Ambassador Service With TLS

You'll need to follow the steps in the [Ambassador TLS Termination](/how-to/tls-termination.md) guide to configure TLS certificates, including using

```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-https.yaml
```

to create the HTTPS Ambassador service.

### Creating the Ambassador Service Without TLS

**We recommend using TLS**, but if for some reason you can't, you create the HTTP-only Ambassador service with

```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-http.yaml
```

### Deploying Ambassador After Creating the Service

Once the Ambassador service is creating, to actually deploy Ambassador you can use

```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-rbac.yaml
```

if Kubernetes Role Based Access Control (RBAC) is enabled, or

```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-no-rbac.yaml
```

if not.

## Diagnostics

Ambassador provides a diagnostics overview on port 8877 by default. This is deliberately not exposed to the outside world; you'll need to use `kubectl port-forward` for access, something like

```shell
kubectl port-forward ambassador-xxxx-yyy 8877
```

where you'll have to fill in the actual pod name of one of your Ambassador pods (any will do). Once you have that, you'll be able to point a web browser at

`http://localhost:8877/ambassador/v0/diag/`

for the diagnostics overview.

![Diagnostics](/images/diagnostics.png)

 Some of the most important information - your Ambassador version, how recently Ambassador's configuration was updated, and how recently Envoy last reported status to Ambassador - is right at the top. The diagnostics overview can show you what it sees in your configuration map, and which Envoy objects were created based on your configuration.

If needed, you can get JSON output from the diagnostic service, instead of HTML:

`curl http://localhost:8877/ambassador/v0/diag/?json=true`

## Debugging

If you're running into an issue and the diagnostics service does not provie sufficient information, you can increase the debug level of Envoy. To do so:

* get a shell on your Ambassador pod with `kubectl exec`
* Turn Envoy’s debug logging on with `curl localhost:8001/logging?level=debug`
* Issue your request
* Turn Envoy’s logging back to normal with `curl localhost:8001/logging?level=warning`
* View Envoy's logs with `kubectl logs`

Envoy’s debug logging is very verbose. You can do `localhost:8001/logging?level=debug; sleep 5; curl localhost:8001/logging?level=warning` and then issue the request right after pressing RETURN on that.
