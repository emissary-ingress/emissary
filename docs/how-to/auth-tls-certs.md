# Auth with TLS Client Certificates

If you want to use TLS client-certificate authentication, you'll need to tell Ambassador about the CA certificate chain to use to validate client certificates. Get the CA certificate chain - including all necessary intermediate certificates - and create a Kubernetes secret with it:

```shell
kubectl create secret generic ambassador-cacert --from-file=fullchain.pem=$CACERT_PATH
```

Once that's done, you can tell Ambassador to pay attention to client certs using the `client` block of the `ambassador` module. Note that it is necessary to enable [TLS termination](../how-to/tls-termination.md) as well, or client certificates cannot be checked -- the simplest useful way to enable client certificate authentication is therefore something like:

```yaml
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  tls:
    # The 'server' block configures TLS termination. 'enabled' is the only required element.
    server:
      enabled: True
    client:
      enabled: True
      cert_required: True
```

where the `cert_required` line tells Ambassador to _require_ a client certificate with a valid signature in order to proceed.

If desired, you can override the path to the certificate chain as well:

```yaml
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  tls:
    # The 'server' block configures TLS termination. 'enabled' is the only required element.
    server:
      enabled: True
    client:
      enabled: True
      cert_required: True
      cacert_chain_file: /etc/cacert/fullchain.pem
```

The default is `/etc/cacert/fullchain.pem`, which is how the `ConfigMap` mounts are defined in the `ambassador-proxy.yaml` which Datawire provides. If you choose to move it, you'll need to edit the maps!
