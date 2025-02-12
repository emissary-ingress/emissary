# Emissary-ingress 3.10.0

Emissary 3.10 encompasses:

- all the changes made in this repo by Ambassador Labs after Emissary 3.9.1;
- updates to Go dependencies;
- updates to the included Envoy Proxy; and
- community updates (with huge thanks to [Ajay Choudhary] and [Luke Shumaker]!)

**Note well** that there is also an Ambassador Edge Stack 3.10.0, but
**Emissary 3.10 and Edge Stack 3.10 are not equivalent**. Their codebases have
diverged and will continue to do so.

[Ajay Choudhary]: https://github.com/ajaychoudhary-hotstar
[Luke Shumaker]: https://github.com/lukeshu

## Trying 3.10

The most current Emissary 3.10 is Emissary 3.10.0-rc.1. This is a **release
candidate**: it passes CI and is believed to be stable, but there will be at
least one more RC to update Python dependencies and pull in further community
changes, and we'd like more people to try it out before we call it a final
release.

By far the easiest way to try things is with Helm -- and note that Emissary
3.10 includes a Helm chart for CRDs.

### Installing if you're already running an earlier Emissary

First, install the CRDs and the conversion webhook:

```bash
helm install emissary-crds \
 --namespace emissary-system --create-namespace \
 oci://docker.io/dwflynn/emissary-crds-chart --version=3.10.0-rc.1 \
 --wait
```

This will install all the versions of the CRDs (v1, v2, and v3alpha1) and the
conversion webhook into the `emissary-system` namespace. Once that's done, you'll install Emissary itself:

```bash
helm install emissary \
 --namespace emissary --create-namespace \
 oci://docker.io/dwflynn/emissary-ingress --version=3.10.0-rc.1 \
 --wait
```

### Installing if you're starting fresh

If you're starting from scratch and you don't need to worry about older CRD
versions, you can do things differently:

```bash
helm install emissary-crds \
 --namespace emissary --create-namespace \
 oci://docker.io/dwflynn/emissary-crds-chart --version=3.10.0-rc.1 \
 --set enableLegacyVersions=false \
 --wait
```

This will install only v3alpha1 CRDs and skip the conversion webhook entirely.
It will create the `emissary` namespace for you, but there won't be anything
in it until you install Emissary itself:

```bash
helm install emissary \
 --namespace emissary \
 oci://docker.io/dwflynn/emissary-ingress --version=3.10.0-rc.1 \
 --set waitForApiext.enabled=false \
 --wait
```

### Using Emissary

In either case above, you should have a running Emissary behind the Service
named `emissary-emissary-ingress` in the `emissary` namespace. How exactly you connect to that Service will vary with your cluster provider, but you can start with

```bash
kubectl get svc -n emissary emissary-emissary-ingress
```

and that should get you started. Or, of course, you can use something like

```bash
kubectl port-forward -n emissary svc/emissary-emissary-ingress 8080:80
```

(after you configure a Listener!) and then talk to localhost:8080 with any
kind of cluster.

## Using Faces for a sanity check

[Faces Demo]: https://github.com/buoyantio/faces-demo

If you like, you can continue by using the [Faces Demo] as a quick sanity
check. First, install Faces itself using Helm:

```bash
helm install faces \
 --namespace faces --create-namespace \
 oci://ghcr.io/buoyantio/faces-chart --version 2.0.0-rc.1 \
 --wait
```

Next, you'll need to configure Emissary to route to Faces. First, we'll do the
basic configuration to tell Emissary to listen for HTTP traffic:

```bash
kubectl apply -f - <<EOF
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: ambassador-https-listener
spec:
  port: 8443
  protocol: HTTPS
  securityModel: XFP
  hostBinding:
    namespace:
      from: ALL
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: ambassador-http-listener
spec:
  port: 8080
  protocol: HTTP
  securityModel: XFP
  hostBinding:
    namespace:
      from: ALL
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: wildcard-host
spec:
  hostname: "*"
  requestPolicy:
    insecure:
      action: Route
EOF
```

(This actually supports both HTTPS and HTTP, but since we haven't set up TLS
certificates, we'll just stick with HTTP.)

Next, we need two Mappings:

| Prefix    | Routes to Service | in Namespace |
| --------- | ----------------- | ------------ |
| `/faces/` | `faces-gui`       | `faces`      |
| `/face/`  | `face`            | `faces`      |

```bash
kubectl apply -f - <<EOF
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: gui-mapping
  namespace: faces
spec:
  hostname: "*"
  prefix: /faces/
  service: faces-gui.faces
  rewrite: /
  timeout_ms: 0
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: face-mapping
  namespace: faces
spec:
  hostname: "*"
  prefix: /face/
  service: face.faces
  timeout_ms: 0
EOF
```

Once that's done, then you'll be able to access the Faces Demo at `/faces/`,
on whatever IP address or hostname your cluster provides for the
`emissary-emissary-ingress` Service. Or you can port-forward as above and
access it at `http://localhost:8080/faces/`.
