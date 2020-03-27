# Upgrading Ambassador Edge Stack

Since Ambassador Edge Stack's configuration is entirely stored in Kubernetes resources, no special process is necessary to upgrade Ambassador Edge Stack. If you're using the YAML files supplied by Datawire, you'll be able to upgrade simply by repeating the following `kubectl apply` commands.

First, determine if Kubernetes has RBAC enabled:

```shell
kubectl cluster-info dump --namespace kube-system | grep authorization-mode
```

If you see something like `--authorization-mode=Node,RBAC` in the output, then RBAC is enabled.

If RBAC is enabled:

```shell
kubectl apply -f https://www.getambassador.io/yaml/aes-crds.yaml
```

If RBAC is not enabled:

```shell
kubectl apply -f https://www.getambassador.io/yaml/aes.yaml
```

This will trigger a rolling upgrade of Ambassador Edge Stack.

If you're using your own YAML, check the Datawire YAML to be sure of other changes, but at minimum, you'll need to change the pulled `image` for the Ambassador Edge Stack container and redeploy.

## Helm

Helm makes upgrades easy by managing all the changes made to the installation files automatically. Assuming you followed the instructions for [installing Ambassador via Helm](/user-guide/helm/), the upgrade is very simple:

1. Grab the latest version of the Chart

```shell
helm repo update datawire
```

2. Upgrade your Ambassador release

```shell
helm upgrade ambassador datawire/ambassador -n ambassador
```
