# Installing the Dev Portal

To install the Dev Portal you will need:

1. A Docker image for the Dev Portal.
2. Load the relevant Kubernetes resources.

## Temporary hack: Building Docker images in minikube

In the future, once this is merged, the Docker image should be available from quay.io, but in the short term you need to load it into your Kubernetes cluster somehow.
You can e.g. build and push to any public Docker repository you have access to and edit the Kubernetes yaml appropriately.

I've been testing with minikube, where you can build directly into the Docker registry there.
In the root of your `apro` checkout:

```
$ make build
$ eval $(minikube docker-env)
$ cd docker/dev-portal-server
$ export KUBECONFIG=~/.kube/config
$ docker build -t quay.io/ambassador/ambassador_pro:dev-portal-server-0.4.0 .
```

This tag matches what's in the Kubernetes YAML file.


## Loading the relevant Kubernetes resources

Assuming you have standard Ambassador install already, i.e. Service `ambassador.default`, in the root of your `apro` checkout you just need to:

```
$ kubectl apply -f docs/dev-portal/devportal-rbac.yaml
```

In practice, customers will need edit a few environment variables in there, but out-of-the-box you get something working.

## Accessing your new Dev Portal

By default Ambassador routes `/docs/` to the Dev Portal.
So go to the root public URL of Ambassador and then tack on `/docs/` to the end.
