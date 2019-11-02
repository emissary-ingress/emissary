# Upgrading Ambassador

Since Ambassador's configuration is entirely stored in annotations or a `ConfigMap`, no special process is necessary to upgrade Ambassador. If you're using the YAML files supplied by Datawire, you'll be able to upgrade simply by repeating the following `kubectl apply` commands.

First determine if Kubernetes has RBAC enabled:

```shell
kubectl cluster-info dump --namespace kube-system | grep authorization-mode
```

If you see something like `--authorization-mode=Node,RBAC` in the output, then RBAC is enabled.

If RBAC is enabled:
```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-rbac.yaml
```

If RBAC is not enabled:
```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-no-rbac.yaml
```

This will trigger a rolling upgrade of Ambassador Edge Stack.

If you're using your own YAML, check the Datawire YAML to be sure of other changes, but at minimum, you'll need to change the pulled `image` for the Ambassador Edge Stack container and redeploy.

## Upgrading to 0.50.0

Ambassador 0.50.0 adds the v1 API. All future Ambassador Edge Stack features will be created for the v1 API. Please refer to the [changelog](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md) for information on all new features.

While for the most part, upgrading from v0 to v1 is as simple as changing the `apiVersion` in the resource definition, there are a couple of breaking changes that need to be addressed. 

**Rate Limiting**
The [rate_limits](/reference/rate-limits/) `Mapping` attribute is replaced by `labels` in the v1 API. Ambassador 0.50.0 requires the v1 `Mapping` API for rate limtiing. See the [rate limiting tutorial](/user-guide/rate-limiting-tutorial#v1-api) and [rate limits](/reference/rate-limits/) documentation for more information. 

**AuthService**
The v1 `AuthService` API adds a number of features that require configuration changes. Please refer to the [authentication documentation](/reference/services/auth-service) for more information on how to upgrade.
