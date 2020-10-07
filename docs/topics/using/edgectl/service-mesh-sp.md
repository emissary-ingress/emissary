# Service Preview and Service Meshes

The Service Preview `traffic-agent` runs as a sidecar to the container running your application. Service Preview will run alongside a service mesh without any changes needed to be made on your part.

Below is some information on how Service Preview runs alongside different service mesh implementations.

## Service Preview and Istio

Istio is an open source Envoy-based service mesh implementation that transparently intercepts traffic to your containers and adds observability, security, and layer 7 routing to your services. 

Service Preview can intercept services in an Istio service mesh with a couple of configuration details.

1. Ensure the `istio-proxy` is not injected in Service Preview components

Istio does some complicated networking to ensure that it can transparently intercept all traffic to your service. While this is great for making it easy to add a service mesh to your applications, it interferes with Service Preview's ability to intercept and proxy traffic to your local machine. To ensure that Istio will not automatically inject the `istio-proxy` in Service Preview services, the Traffic Manager, Ambassador Injector, and teleproxy services must be annotated with annotated with `sidecar.istio.io/inject: 'false'`.

The Traffic Manager and Ambassador Injector have this annotation by default. You must manually set this in the `teleproxy` pod that is created after running `edgectl connect` for the first time. You can do this with the following `kubectl` command:

```sh
kubectl annotation po teleproxy sidecar.istio.io/inject='false'
```

2. Ensure the `traffic-agent` can run alongside another Envoy proxy

The `traffic-agent` is powered by Envoy proxy. Envoy, by default, does not like to run on the same host as other Envoy proxies. Since Istio is also powered by Envoy, you need to ensure that the `traffic-agent` knows how to run in the same pod as the `istio-proxy`.

This is easily done by setting `AMBASSADOR_ENVOY_BASE_ID: "1"` in the `traffic-agent` environment. Make sure this is set when injecting the `traffic-agent`.

> **IMPORTANT**
> At the moment, the Ambassador Injector does not automatically set `AMBASSADOR_ENVOY_BASE_ID`. You will need to manually inject the `traffic-agent` when intercepting a service in your Istio mesh.
> 
> Future versions of the Ambassador Injector will support setting `AMBASSADOR_ENVOY_BASE_ID`.

