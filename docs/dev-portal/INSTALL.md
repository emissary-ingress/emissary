# Installing the Dev Portal

To install the Dev Portal you will need:

1. A Docker image for the Dev Portal.
2. A Docker image for the Internal Access service.
2. Load the relevant Kubernetes resources.

## Temporary hack: Building Docker images in minikube

In the future, once this is merged, the Docker image should be available from quay.io, but in the short term you need to load it into your Kubernetes cluster somehow.
You can e.g. build and push to any public Docker repository you have access to and edit the Kubernetes yaml appropriately.

I've been testing with minikube, where you can build directly into the Docker registry there.
In the root of your `apro` checkout:

```
$ make docker/dev-portal-server/dev-portal-server
$ make docker/apro-internal-access/apro-internal-access
$ eval $(minikube docker-env)
$ cd docker/dev-portal-server
$ export KUBECONFIG=~/.kube/config
$ docker build -t quay.io/ambassador/ambassador_pro:dev-portal-server-0.4.0 .
$ cd ../apro-internal-access
$ docker build -t quay.io/ambassador/ambassador_pro:apro-internal-access-0.4.0 .
```

These tag matches what's in the Kubernetes YAML files.


## Loading the relevant Kubernetes resources

Assuming you have standard Ambassador and Ambassador Pro install already, i.e. Service `ambassador.default`, in the root of your `apro` checkout you just need to:

First, edit `docs/dev-portal/devportal-rbac.yaml` so it has an appropriate Ambassador Pro license key (you can get one from `k8s-env.sh` in this repo).

Then:

```
$ kubectl apply -f docs/dev-portal/internal.yaml
$ kubectl apply -f docs/dev-portal/devportal-rbac.yaml
```

In practice, customers will need edit a few environment variables in there, but out-of-the-box you get something working.

## Accessing your new Dev Portal

By default Ambassador routes `/docs/` to the Dev Portal.
So go to the root public URL of Ambassador and then tack on `/docs/` to the end.
