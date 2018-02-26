## Upgrading Ambassador

Since Ambassador's configuration is entirely stored in annotations or a `ConfigMap`, no special process is necessary to upgrade Ambassador. If you're using the YAML files supplied by Datawire, you'll be able to upgrade simply by repeating

```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-rbac.yaml
```

or

```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-no-rbac.yaml
```

as appropriate for your cluster. This will trigger a rolling upgrade of Ambassador.

If you're using your own YAML, check the Datawire YAML to be sure of other changes, but at minimum, you'll need to change the pulled `image` for the Ambassador container and redeploy.
