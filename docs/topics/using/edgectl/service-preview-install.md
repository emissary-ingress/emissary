# Service Preview Quick Start

Service Preview is installed as an addon to the Ambassador Edge Stack.

## Prerequisites

- [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/) access to a Kubernetes cluster
- [`edgectl`](../edge-control#installing-edge-control) client

## Install

There are three method for installing Service Preview.

### [<img class="Ambassador's OpenSource Blackbird" src="../../../images/features-page-bird.svg"/> Install with Edgectl](#install-with-edgectl)

If you are installing Service Preview and Ambassador Edge Stack for the first time, `edgectl` will automatically bootstrap and integrate both tools in your cluster.

### [<img class="k8s-logo" src="../../../images/kubernetes.png"/> Install with YAML](install with yaml)

The YAML installation method will walk you through a step-by-step deployment of all the resources necessary for installing Service Preview alongside the Ambassador Edge Stack. The YAML installation method is the most common approach to install Ambassador Edge Stack, especially in production environments, with our default, customizable manifest.

### [<img class="helm-logo" src="../../../images/helm-navy.png"/> Install with Helm](#install-with-helm)

Helm is a popular Kubernetes package manager. The Ambassador helm chart allows you to install Service Preview alongside the Ambassador Edge Stack.

---

## <img class="Ambassador's OpenSource Blackbird" src="../../../images/features-page-bird.svg"/> Install with Edgectl

If you are a new user, or you are looking to start using Ambassador Edge Stack with Service Preview on a fresh installation, the `edgectl install` command will get you up and running in no time with a pre-configured Traffic Manager and Traffic Agent supported by automatic sidecar injection.

### 1. Install the Traffic Manager and Ambassador Injector Alongside the Ambassador Edge Stack 

The Traffic Manager is what is responsible for managing communications between your Kubernetes cluster and your local machine.

Services in your cluster opt-in to using Service Preview by injecting the Traffic Agent sidecar. Service Preview includes an automatic sidecar injection feature which simplifies the process of injecting the Traffic Agent as sidecars to your services.

Run the following command to let `edgectl` bootstrap your cluster with Ambassador, the Traffic Manager, and Ambassador Injector:

```sh
$ edgectl install
```

### 2. Connect to your Cluster

Now that you installed the Traffic Manager, you can connect to your cluster using `edgectl`.

First, start the daemon on your local machine to prime your local machine for connecting to your cluster

```sh
$ sudo edgectl daemon

Launching Edge Control Daemon v1.6.1 (api v1)
```

The daemon is now running and your local machine is ready to connect to your laptop. See the [`edgectl daemon` reference](../edge-control#edgectl-daemon) for more information on how `edgectl` stages your local machine for connecting to your cluster.

After starting the daemon, you are ready to connect to the Traffic Manager.

Connect your local machine to the cluster with `edgectl`:

```sh
$ edgectl connect

Connecting to traffic manager in namespace ambassador...
Connected to context default (https://34.72.18.227)
```

`edgectl` will now attempt to connect to the Traffic Manager in your cluster and bridge your cluster and local networks.

Verify that you are connected to your cluster:

```sh
$ edgectl status

Connected
  Context:       default (https://34.72.18.227)
  Proxy:         ON (networking to the cluster is enabled)
  Interceptable: 0 deployments
  Intercepts:    0 total, 0 local
```

### 3. Inject the Traffic Agent Sidecar

The Traffic Agent sidecar is required in order to intercept requests to a service and route them to your local machine.

At the moment, you can see that no sidecars are currently available with `edgectl`:

```sh
$ edgectl intercept available

No interceptable deployments
```

The Traffic Agent sidecar needs to be added to any service that you would like to use with Service Preview.

With the automatic injector, we can simply add it to our services by annotating the pod with `getambassador.io/inject-traffic-agent: enabled`.

First, you need to create the RBAC resources required for the Traffic Agent to run in the namespace you want to intercept.

The following will create the required resources in the default namespace. If you would like to run Service Preview in another namespace, you need to download and edit the YAML and

* change the namespace of the `ServiceAcount` and `Secret`
* edit the `ClusterRoleBinding` to reference the `traffic-agent` `ServiceAccount` in the appropriate namespace


Create the RBAC resources with `kubectl`:

```
kubectl apply -f https://getambassador.io/yaml/traffic-agent-rbac.yaml
```

Then, apply the `Hello` service manifest that is annotated to inject the Traffic Agent.

```sh
kubectl apply -f - <<EOF
---
apiVersion: v1
kind: Service
metadata:
  name: hello
  namespace: default
  labels:
    app: hello
spec:
  selector:
    app: hello
  ports:
    - protocol: TCP
      port: 80
      targetPort: http
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: hello
  namespace: default
  labels:
    app: hello
spec:
  prefix: /hello/
  service: hello:80
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello
  namespace: default
  labels:
    app: hello
spec:
  replicas: 1
  selector:
    matchLabels:
      app: hello
  template:
    metadata:
      annotations:
        getambassador.io/inject-traffic-agent: enabled # Enable automatic Traffic Agent sidecar injection
      labels:
        app: hello
    spec:
      containers:
        - name: hello
          image: docker.io/datawire/hello-world:latest
          ports:
            - name: http
              containerPort: 8000 
EOF
 
service/hello created
mapping.getambassador.io/hello created
deployment.apps/hello created
```

After applying the above manifest, you can see that there is now an available service to intercept.

```
$ edgectl intercept available

Found 1 interceptable deployment(s):
   1. hello in namespace default
```

Take a look at the [Traffic Agent reference](../service-preview-reference#traffic-agent) for more information on how to connect your services to Service Preview.

Service Preview is now installed in your cluster and ready to intercept traffic sent to the `Hello` service! 

### Next Steps

Now that you have Service Preview installed, let's see how you can use it to intercept traffic sent to services in your Kubernetes cluster!

Take a look at the [Service Preview Tutorial](../service-preview-tutorial) to get Service Preview working for the `Hello` service we installed!

---

## <img class="k8s-logo" src="../../../images/kubernetes.png"/> Install with YAML

Downloading and installing our published Kubernetes YAML gives you full control over the installation of Service Preview. This is the most popular approach for running Service Preview in production and in CI.

### 1. Install the Ambassador Edge Stack

Service Preview runs alongside the Ambassador Edge Stack.

[Install Ambassador Edge Stack](https://www.getambassador.io/docs/latest/topics/install/yaml-install/) if you do not already have it running.


### 2. Install the Traffic Manager and Ambassador Injector

The Traffic Manager is what is responsible for managing communications between your Kubernetes cluster and your local machine.

Services in your cluster opt-in to using Service Preview by injecting the Traffic Agent sidecar. Service Preview includes an automatic sidecar injection feature which simplifies the process of injecting the Traffic Agent as sidecars to your services.

Deploy the Traffic Manager and Ambassador Injector in the `ambassador` namespace with `kubectl`:

```
kubectl apply -f https://getambassador.io/yaml/traffic-manager.yaml
kubectl apply -f https://getambassador.io/yaml/ambassador-injector.yaml
```

The above will deploy:

- `ServiceAccount`, `ClusterRole`, and `ClusterRoleBinding` named `traffic-manager` to grant the Traffic Manager the necessary RBAC permissions.
- A `Service` and `Deployment` named `telepresence-proxy` which is the name for the Traffic Manager in the cluster.
- The Ambassador Injector with a `MutatingWebhookConfiguration` that allows injection of the Traffic Agent sidecar in newly created pods.

See the [Traffic Manager reference](../service-preview-reference#traffic-manager) for more information on this deployment.

The traffic manager is now installed in the Ambassador namespace in your cluster and is ready to connect your cluster to your local machine.

### 3. Connect to your Cluster

Now that you installed the Traffic Manager, you can connect to your cluster using `edgectl`.

First, start the daemon on your local machine to prime your local machine for connecting to your cluster

```sh
$ sudo edgectl daemon

Launching Edge Control Daemon v1.6.1 (api v1)
```

The daemon is now running and your local machine is ready to connect to your laptop. See the [`edgectl daemon` reference](../edge-control#edgectl-daemon) for more information on how `edgectl` stages your local machine for connecting to your cluster.

After starting the daemon, you are ready to connect to the Traffic Manager.

Connect your local machine to the cluster with `edgectl`:

```sh
$ edgectl connect

Connecting to traffic manager in namespace ambassador...
Connected to context default (https://34.72.18.227)
```

`edgectl` will now attempt to connect to the Traffic Manager in your cluster and bridge your cluster and local networks.

Verify that you are connected to your cluster:

```sh
$ edgectl status

Connected
  Context:       default (https://34.72.18.227)
  Proxy:         ON (networking to the cluster is enabled)
  Interceptable: 0 deployments
  Intercepts:    0 total, 0 local
```

### 5. Inject the Traffic Agent Sidecar

The Traffic Agent sidecar is required in order to intercept requests to a service and route them to your local machine.

At the moment, you can see that no sidecars are currently available with `edgectl`:

```sh
$ edgectl intercept available

No interceptable deployments
```

The Traffic Agent sidecar needs to be added to any service that you would like to use with Service Preview.

With the automatic injector, we can simply add it to our services by annotating the pod with `getambassador.io/inject-traffic-agent: enabled`.

First, you need to create the RBAC resources required for the Traffic 

The following will create the required resources in the default namespace. If you would like to run Service Preview in another namespace, you need to download and edit the YAML and

* change the namespace of the `ServiceAcount` and `Secret`
* edit the `ClusterRoleBinding` to reference the `traffic-agent` `ServiceAccount` in the appropriate namespace


Create the RBAC resources with `kubectl`:

```
kubectl apply -f https://getambassador.io/yaml/traffic-agent-rbac.yaml

Then, apply the `Hello` service manifest that is annotated to inject the Traffic Agent.

```sh
kubectl apply -f - <<EOF
---
apiVersion: v1
kind: Service
metadata:
  name: hello
  namespace: default
  labels:
    app: hello
spec:
  selector:
    app: hello
  ports:
    - protocol: TCP
      port: 80
      targetPort: http
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: hello
  namespace: default
  labels:
    app: hello
spec:
  prefix: /hello/
  service: hello:80
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello
  namespace: default
  labels:
    app: hello
spec:
  replicas: 1
  selector:
    matchLabels:
      app: hello
  template:
    metadata:
      annotations:
        getambassador.io/inject-traffic-agent: enabled # Enable automatic Traffic Agent sidecar injection
      labels:
        app: hello
    spec:
      containers:
        - name: hello
          image: docker.io/datawire/hello-world:latest
          ports:
            - name: http
              containerPort: 8000 
EOF
 
service/hello created
mapping.getambassador.io/hello created
deployment.apps/hello created
```

After applying the above manifest, you can see that there is now an available service to intercept.

```
$ edgectl intercept available

Found 1 interceptable deployment(s):
   1. hello in namespace default
```

Take a look at the [Traffic Agent reference](../service-preview-reference#traffic-agent) for more information on how to connect your services to Service Preview.

Service Preview is now installed in your cluster and ready to intercept traffic sent to the `Hello` service! 

## Next Steps

Now that you have Service Preview installed, let's see how you can use it to intercept traffic sent to services in your Kubernetes cluster!

Take a look at the [Service Preview Tutorial](../service-preview-tutorial) to get Service Preview working for the `Hello` service we installed!

---

## <img class="helm-logo" src="../../../images/helm-navy.png"/> Install with Helm

Helm is a popular package manager for Kubernetes software. The Ambassador helm chart contains a lot of configuration options that make it easy to deploy and upgrade a custom configuration of Ambassador Edge Stack.

The Ambassador chart also contains configurations for installing Service Preview alongside Ambassador Edge Stack.

Downloading and installing our published Kubernetes YAML gives you full control over the installation of Service Preview. This is the most popular approach for running Service Preview in production and in CI.

### 1. Install the Traffic Manager and Ambassador Injector Alongside the Ambassador Edge Stack 

The Traffic Manager is what is responsible for managing communications between your Kubernetes cluster and your local machine.

Services in your cluster opt-in to using Service Preview by injecting the Traffic Agent sidecar. Service Preview includes an automatic sidecar injection feature which simplifies the process of injecting the Traffic Agent as sidecars to your services.

These services are available to be deployed in the helm chart. 

Install Service Preview alongside the Ambassador Edge Stack with the following `values.yaml` options:

```yaml
servicePreview:
  enabled: true
```

Create the Ambassador namespace if it is not already created:

```sh
$ kubectl create namespace ambassador
```

Upgrade or install your release of the Ambassador Edge Stack with the Traffic Manager and Ambassador Injector

```sh
$ helm upgrade --install ambassador -n ambassador datawire/ambassador -f values.yaml
```


### 2. Connect to your Cluster

Now that you installed the Traffic Manager, you can connect to your cluster using `edgectl`.

First, start the daemon on your local machine to prime your local machine for connecting to your cluster

```sh
$ sudo edgectl daemon

Launching Edge Control Daemon v1.6.1 (api v1)
```

The daemon is now running and your local machine is ready to connect to your laptop. See the [`edgectl daemon` reference](../edge-control#edgectl-daemon) for more information on how `edgectl` stages your local machine for connecting to your cluster.

After starting the daemon, you are ready to connect to the Traffic Manager.

Connect your local machine to the cluster with `edgectl`:

```sh
$ edgectl connect

Connecting to traffic manager in namespace ambassador...
Connected to context default (https://34.72.18.227)
```

`edgectl` will now attempt to connect to the Traffic Manager in your cluster and bridge your cluster and local networks.

Verify that you are connected to your cluster:

```sh
$ edgectl status

Connected
  Context:       default (https://34.72.18.227)
  Proxy:         ON (networking to the cluster is enabled)
  Interceptable: 0 deployments
  Intercepts:    0 total, 0 local
```

### 3. Inject the Traffic Agent Sidecar

The Traffic Agent sidecar is required in order to intercept requests to a service and route them to your local machine.

At the moment, you can see that no sidecars are currently available with `edgectl`:

```sh
$ edgectl intercept available

No interceptable deployments
```

The Traffic Agent sidecar needs to be added to any service that you would like to use with Service Preview.

With the automatic injector, we can simply add it to our services by annotating the pod with `getambassador.io/inject-traffic-agent: enabled`.

First, you need to create the RBAC resources required for the Traffic 

The following will create the required resources in the default namespace. If you would like to run Service Preview in another namespace, you need to download and edit the YAML and

* change the namespace of the `ServiceAcount` and `Secret`
* edit the `ClusterRoleBinding` to reference the `traffic-agent` `ServiceAccount` in the appropriate namespace


Create the RBAC resources with `kubectl`:

```
kubectl apply -f https://getambassador.io/yaml/traffic-agent-rbac.yaml
```

Then, apply the `Hello` service manifest that is annotated to inject the Traffic Agent.

```sh
kubectl apply -f - <<EOF
---
apiVersion: v1
kind: Service
metadata:
  name: hello
  namespace: default
  labels:
    app: hello
spec:
  selector:
    app: hello
  ports:
    - protocol: TCP
      port: 80
      targetPort: http
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: hello
  namespace: default
  labels:
    app: hello
spec:
  prefix: /hello/
  service: hello:80
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello
  namespace: default
  labels:
    app: hello
spec:
  replicas: 1
  selector:
    matchLabels:
      app: hello
  template:
    metadata:
      annotations:
        getambassador.io/inject-traffic-agent: enabled # Enable automatic Traffic Agent sidecar injection
      labels:
        app: hello
    spec:
      containers:
        - name: hello
          image: docker.io/datawire/hello-world:latest
          ports:
            - name: http
              containerPort: 8000 
EOF
 
service/hello created
mapping.getambassador.io/hello created
deployment.apps/hello created
```

After applying the above manifest, you can see that there is now an available service to intercept.

```
$ edgectl intercept available

Found 1 interceptable deployment(s):
   1. hello in namespace default
```

Take a look at the [Traffic Agent reference](../service-preview-reference#traffic-agent) for more information on how to connect your services to Service Preview.

Service Preview is now installed in your cluster and ready to intercept traffic sent to the `Hello` service! 

## Next Steps

Now that you have Service Preview installed, let's see how you can use it to intercept traffic sent to services in your Kubernetes cluster!

Take a look at the [Service Preview Tutorial](../service-preview-tutorial) to get Service Preview working for the `Hello` service we installed!
