# The Ambassador API Gateway

**The Ambassador Edge stack is now available and includes additional functionality beyond the current Ambassador API Gateway.** These features including automatic HTTPS, the Edge Policy Console UI, OAuth/OpenID Connect authentication support, integrated rate limiting, a developer portal, and [more](/edge-stack-faq/).

If you still want to use just the Ambassador API Gateway, don't worry! You can follow the directions below to install it. Throughout the documentation, you'll see product tags at the top of the page, so you know what features apply to the Ambassador API Gateway.

## Install the Ambassador API Gateway

* [Kubernetes YAML](#kubernetes-yaml)
* [Helm](#helm)

## Kubernetes YAML

In this tutorial, we'll walk through the process of deploying the Ambassador API Gateway in Kubernetes for ingress routing. The Ambassador API Gateway provides all the functionality of a traditional ingress controller (i.e., path-based routing) while exposing many additional capabilities such as authentication, URL rewriting, CORS, rate limiting, and automatic metrics collection (the [mappings reference](../../using/intro-mappings) contains a full list of supported options). Note that the Ambassador Edge Stack can be used as an [Ingress Controller](../../running/ingress-controller).

For more background on Kubernetes ingress, [read this blog post](https://blog.getambassador.io/kubernetes-ingress-nodeport-load-balancers-and-ingress-controllers-6e29f1c44f2d).

The Ambassador API Gateway is designed to allow service authors to control how their service is published to the Internet. We accomplish this by permitting a wide range of annotations on the *service*, which Ambassador reads to configure its Envoy Proxy.

Below, we'll configure Ambassador to map `/httpbin/` to `httpbin.org`.

### 1. Deploying the Ambassador API Gateway

The following steps deploy Ambassador in the default namespace. 

**Note:** If you're using Google Kubernetes Engine, you'll need to grant permissions to the account that will be setting up the Ambassador API Gateway. To do this, get your official GKE username, and then grant `cluster-admin` role privileges to that username:

```shell
kubectl create clusterrolebinding my-cluster-admin-binding --clusterrole=cluster-admin --user=$(gcloud info --format="value(config.account)")
```

Then, you can deploy the Ambassador API Gateway. Start by installing CRDs required by Ambassador:

```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-crds.yaml
```

Then, apply the RBAC configuration with:

```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-rbac.yaml
```

We recommend downloading the YAML files and exploring the content. You will see that an `ambassador-admin` NodePort Service is created (which provides an Ambassador ODD Diagnostic web UI), along with an ambassador ClusterRole, ServiceAccount, and ClusterRoleBinding. An Ambassador Deployment is also created.

When not installing the Ambassador API Gateway into the default namespace you must update the namespace used in the `ClusterRoleBinding`.

For production configurations, we recommend you download these YAML files as your starting point, and customize them accordingly.

### 2. Defining the Ambassador Service

The Ambassador service is deployed as a Kubernetes Service that references the ambassador Deployment you deployed previously. Create the following YAML and put it in a file called`ambassador-service.yaml`.

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: ambassador
spec:
  type: LoadBalancer
  externalTrafficPolicy: Local
  ports:
   - port: 80
     targetPort: 8080
  selector:
    service: ambassador
```

Deploy this service with `kubectl`:

```shell
$ kubectl apply -f ambassador-service.yaml
```

The YAML above creates a Kubernetes service for Ambassador of type `LoadBalancer`, and configures the `externalTrafficPolicy` to propagate [the original source IP](https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/#preserving-the-client-source-ip) of the client. All HTTP traffic will be evaluated against the routing rules you create. Note that if you're not deploying in an environment where `LoadBalancer` is a supported type (such as minikube), you'll need to change this to a different type of service, e.g., `NodePort`.

If you have a static IP provided by your cloud provider you can set as `loadBalancerIP`.

### 3. The Diagnostics Service in Kubernetes

the Ambassador API Gateway includes an integrated diagnostics service to help with troubleshooting.

By default, this is exposed to the internet at the URL `http://{{AMBASSADOR_HOST}}/ambassador/v0/diag/`. Go to that URL from a web browser to view the diagnostic UI.

You can change the default so it is not exposed externally by default by setting `diagnostics.enabled: false` in the [ambassador `Module`](../../running/ambassador).

```yaml
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
spec:
  config:
    diagnostics:
      enabled: false
```

After applying this `Module`, to view the diagnostics UI, we'll need to get the name of one of the Ambassador pods:

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

will then let us view the diagnostics at `http://localhost:8877/ambassador/v0/diag/`.

### 4. Enable HTTPS

The versatile HTTPS configuration of the Ambassador API Gateway lets it support various HTTPS use cases whether simple or complex.

See the [TLS HOWTO](../../../howtos/tls-termination) to quickly enable HTTPS support for your applications.

**Note that the Ambassador Edge Stack automatically enables HTTPs.** Read more about its configuration on the [Host CRD](../../running/host-crd) page.

## Helm

In the following instructions, we'll install the open-source Ambassador API
Gateway with Helm. 

Although the [Helm chart](https://github.com/datawire/ambassador-chart) installs
the Ambassador Edge Stack by default, the Ambassador API Gateway is still
available for installation for both Helm 2 and Helm 3.

With Helm 2, you must enable CRD creation with the `crd-install` hook that is
included in the CRD manifests. When installing with Helm 3, the following
message will be output to `stderr`:

```bash
manifest_sorter.go:175: info: skipping unknown hook: "crd-install"
```

Because this hook is required for Helm 2 support, it **IS NOT AN ERROR AND CAN BE SAFELY IGNORED**.

**To get started on Helm:**
1. Add the Datawire repo to your Helm repositories
2. Install the Ambassador API Gateway


### 1. Add the Datawire repo to your Helm repositories

```
helm repo add datawire https://www.getambassador.io
```

### 2. Install Ambassador API Gateway

The Ambassador Edge Stack is installed by default. To install the Ambassador API
Gateway instead,  change the `image` to point to the OSS image and set
`enableAES: false` in the `values.yaml` file.

For example:

```yaml
image:
  repository: quay.io/datawire/ambassador
  tag: $version$
enableAES: false
```

Then, install the chart using the `values.yaml` file:

```
helm install ambassador datawire/ambassador -f values.yaml
```

You can also install the chart with the `--set` flag:

```
helm install ambassador datawire/ambassador --set image.repository=quay.io/datawire/ambassador --set image.tag=$version$ --set enableAES=false
```

## Want More?

For more features, check out the latest build of the [Ambassador Edge Stack](../).
