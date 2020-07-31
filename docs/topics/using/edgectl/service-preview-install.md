# Service Preview Quick Start

Service Preview is installed as an addon to the Ambassador Edge Stack.

There are two main mechanisms for installing Service Preview

- [`edgectl install`](#install-with-edgectl) will boot strap your cluster with the Ambassador Edge Stack and Service Preview and only works when doing a fresh install of Ambassador.

- [`Manual YAML Install`](#install-with-yaml) will work with an existing installation of Ambassador Edge Stack.

## Prerequisites

- [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/) access to a Kubernetes cluster
- The Ambassador Edge Stack [Installed](../../../../tutorials/getting-started/)
- [`edgectl`](../edge-control#installing-edge-control) client

## Install with Edgectl

If you are a new user, or you are looking to start using Ambassador Edge Stack with Service Preview on a fresh installation, the `edgectl install` command will get you up and running in no time with a pre-configured Traffic Manager and Traffic Agent supported by automatic sidecar injection.

Run the following command to let `edgectl` bootstrap your cluster with Ambassador and Service Preview

```sh
$ edgectl install
```

## Install with YAML

Two extra Deployments are required to install Service Preview:

- The [Traffic Manager](#1--install-the-traffic-manager) responsible for managing communication between your Kubernetes Cluster and your local machine
- The [Ambassador Injector](#3--install-the-ambassador-injector) which automates injecting the Traffic Agent sidecar responsible for routing requests to either the container in the cluster or on your local machine

### 1. Install the Traffic Manager

The Traffic Manager is what is responsible for managing communications between your Kubernetes cluster and your local machine.

Deploy the Traffic Manager with `kubectl`

```
kubectl apply -f https://www.getambassador.io/yaml/traffic-manager.yaml
```

The above will deploy:

- `ServiceAccount`, `ClusterRole`, and `ClusterRoleBinding` named `traffic-manager` to grant the Traffic Manager the necessary RBAC permissions
- A `Service` and `Deployment` named `telepresence-proxy` which is the name for the Traffic Manager in the cluster

See the [Traffic Manager reference](service-preview-reference#traffic-manager) for more information on this deployment.

The traffic manager is now installed in the Ambassador namespace in your cluster and is ready to connect your cluster to your local machine.

### 2. Connect to your Cluster

Now that you installed the Traffic Manager, you can connect to your cluster using `edgectl`.

First, start the daemon on your local machine to prime your local machine for connecting to your cluster

```sh
$ sudo edgectl daemon

Launching Edge Control Daemon v1.6.1 (api v1)
```

The daemon is now running and your local machine is ready to connect to your laptop. See the [`edgectl daemon` reference](edge-control#edgectl-daemon) for more information on how `edgectl` stages your local machine for connecting to your cluster.

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
In the last month:
    1 unique developers have connected, licensed for 5 developers
    0 unique CI runs have connected
```

### 3. Install the Ambassador Injector

Services in your cluster opt-in to using Service Preview by injecting the Traffic Agent sidecar. Service Preview includes an automatic sidecar injection feature which simplifies the process of injecting the Traffic Agent as sidecars to your services.

Install the Ambassador Injector in the `ambassador` namespace with `kubectl`:

```sh
kubectl apply -f https://getambassador.io/yaml/ambassador-injector.yaml
```

This installs the Ambassador Injector with a `MutatingWebhookConfiguration` that allows it to inject the sidecar in newly created pods.

### 4. Inject the Traffic Agent Sidecar

The Traffic Agent sidecar is required in order to intercept requests to a service and route them to your local machine.

At the moment, you can see that no sidecars are currently available with `edgectl`:

```sh
edgectl intercept available

No interceptable deployments
```

The Traffic Agent sidecar needs to be added to any service that you would like to use with Service Preview.

With the automatic injector, we can simply add it to our services by annotating the pod with `getambassador.io/inject-traffic-agent: enabled`.

First, you need to create the RBAC resources required for the Traffic Agent to run in the namespace you want to intercept.

```
kubectl apply -f https://getambassador.io/yaml/traffic-agent-rbac.yaml
```

Then, apply the helloworld service manifest that is annotated to inject the Traffic Agent.

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
      targetPort: http              # Application port
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
      containers:                   # Application container
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
edgectl intercept available

Found 1 interceptable deployment(s):
   1. hello in namespace default
```

Take a look at the [Traffic Agent reference](service-preview-reference#traffic-agent) for more information on how to connect your services to Service Preview.

Service Preview is now installed in your cluster and ready to intercept traffic sent to the helloworld service! 

## Next Steps

Now that you have Service Preview installed, let's see how you can use it to intercept traffic sent to services in your Kubernetes cluster!

Take a look at the [Service Preview Quickstart](service-preview-quickstart) to get Service Preview working for the helloworld service we installed!
