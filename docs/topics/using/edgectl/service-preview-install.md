# Service Preview Installation

One of the challenges in adopting Kubernetes and microservices is the development and testing workflow. Creating and maintaining a full development environment with many microservices and their dependencies is complex and hard.

Service Preview addresses this challenge by connecting your CI system or local development infrastructure to the Kubernetes cluster, and dynamically routing specific requests to your local environment.

## Prerequisites

- [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/) access to a Kubernetes cluster
- The Ambassador Edge Stack [Installed](../../../tutorials/getting-started)
- [`edgectl`](../edge-control#installing-edge-control) client

## Install

Service Preview requires two Deployments to run:

- The [Traffic Manager](#1--install-the-traffic-manager) responsible for managing communication between your Kubernetes Cluster and your local machine
- The [Traffic Agent](#3--install-the-traffic-agent-sidecar) sidecar responsible for routing requests to either the container in the cluster or on your local machine

### 1. Install the Traffic Manager

The Traffic Manager is what is responsible for managing communications between your Kubernetes cluster and your local machine.

Deploy the Traffic Manager with `kubectl`

```
kubectl apply -f https://www.getambassador.io/yaml/traffic-manager.yaml
```

The above will deploy:

- `ServiceAccount`, `ClusterRole`, and `ClusterRoleBinding` named `traffic-manager` to grant the Traffic Manager the necessary RBAC permissions
- A `Service` and `Deployment` named `telepresence-proxy` which is the name for the Traffic Manager in the cluster

See the [Traffic Manager reference](../service-preview-reference#traffic-manager) for more information on this deployment.

The traffic manager is now installed in the Ambassador namespace in your cluster and is ready to connect your cluster to your local machine.

### 2. Connect to your Cluster

Now that you installed the Traffic Manager, you can connect to your cluster using `edgectl`.

First, start the daemon on your local machine to prime your local machine for connecting to your cluster

```sh
sudo edgectl daemon
```

The daemon is now running and your local machine is ready to connect to your laptop. See the [`edgectl daemon` reference](../edge-control#edgectl-daemon) for more information on how `edgectl` stages your local machine for connecting to your cluster.

After starting the daemon, you are ready to connect to the Traffic Manager.

Connect your local machine to the cluster with `edgectl`:

```sh
edgectl connect
```

`edgectl` will now attempt to connect to the Traffic Manager in your cluster and bridge your cluster and local networks.

Verify that you are connected to your cluster:

```sh
edgectl status
```

You should see output saying you are connected:
```
Connected
  Context:       default (https://34.72.18.227)
  Proxy:         ON (networking to the cluster is enabled)
  Interceptable: 0 deployments
  Intercepts:    0 total, 0 local
In the last month:
    1 unique developers have connected, licensed for 5 developers
    0 unique CI runs have connected
```

### 3. Install the Traffic Agent Sidecar

The Traffic Agent sidecar is required in order to intercept requests to a service and route them to your local machine.

At the moment, you can see that no sidecars are currently available with `edgectl`:

```sh
edgectl intercept available
```

```
No interceptable deployments
```

The Traffic Agent sidecar needs to be added to any service that you would like to use with Service Preview.

We can add it to the example quote of the moment service by applying a manifest that has the Traffic Agent sidecar injected.

First, you need to create the RBAC resources required for the Traffic Agent to run in the namespace you want to intercept.

```
kubectl apply -f https://getambassador.io/yaml/traffic-agent-rbac.yaml
```

Then, apply the updated quote of the moment service manifest that has the Traffic Agent injected.

```
kubectl apply -f https://getambassador.io/yaml/backends/quote-service-preview.yaml
```

After applying the above manifest, you can see that there is now an available service to intercept.

```
edgectl intercept available
```

```
Found 1 interceptable deployment(s):
   1. quote in namespace default
```

Take a look at the [Traffic Agent reference](../service-preview-reference#traffic-agent) for more information on how to connect your services to Service Preview.

Service Preview is now installed in your cluster and ready to intercept traffic sent to the quote of the moment service! See the next section of a quick example of how to get Service Preview to intercept traffic sent to the quote service!

## Intercepting Traffic