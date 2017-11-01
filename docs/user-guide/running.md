# Running Ambassador

The simplest way to run Ambassador is **not** to build it! Instead, just use the YAML files published at https://www.getambassador.io, and start by deciding whether you want to use TLS or not. (If you want more information on TLS, check out our [TLS Overview](../reference/tls-auth.md).) It's possible to switch this later, but it's a pain, and may well involve mucking about with your DNS and such to do it, so it's better to decide up front.

## Upgrading Ambassador

Since Ambassador's configuration is entirely stored in its ConfigMap, no special process is necessary to upgrade Ambassador. If you're using the YAML files supplied by Datawire, you'll be able to upgrade simply by repeating (for HTTPS)

```
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-proxy.yaml
```

or (for HTTP)

```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador.yaml
```

to trigger a rolling upgrade of Ambassador.

If you're using your own YAML, check the Datawire YAML to be sure of other changes, but at minimum, you'll need to change the pulled `image` for the Ambassador container and redeploy.

## Diagnostics

Ambassador provides a diagnostics overview on port 8877 by default. This is deliberately not exposed to the outside world; you'll need to use `kubectl port-forward` for access, something like

```shell
kubectl port-forward ambassador-xxxx-yyy 8877
```

where, obviously, you'll have to fill in the actual pod name of one of your Ambassador pods (any will do).

Once you have that, you'll be able to point a web browser at

`http://localhost:8877/ambassador/v0/diag/`

for the diagnostics overview. Some of the most important information - your Ambassador version, how recently Ambassador's configuration was updated, and how recently Envoy last reported status to Ambassador - is right at the top. The diagnostics overview can show you what it sees in your configuration map, and which Envoy objects were created based on your configuration.

If needed, you can get JSON output from the diagnostic service, instead of HTML:

`curl http://localhost:8877/ambassador/v0/diag/?json=true`
