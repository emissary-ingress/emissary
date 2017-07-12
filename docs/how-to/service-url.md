# Ambassador Microservice Access

Access to your microservices through Ambassador is via port 443 (if you configured TLS) or port 80 (if not). This port is exposed with a Kubernetes service; we'll use `$AMBASSADORURL` as shorthand for the base URL through this port.

If you're using TLS, you can set it by hand with something like

```shell
export AMBASSADORURL=https://your-domain-name
```

where `your-domain-name` is the name you set up when you requested your certs. **Do not include a trailing `/`**, or the examples in this document won't work.

Without TLS, if you have a domain name, great, do the above.

If you're using AWS, GKE, or Minikube, you may be able to use the commands below -- **note that these will only work since we already know we're using HTTP**:

```shell
# AWS (for Ambassador using HTTP)
AMBASSADORURL=http://$(kubectl get service ambassador --output jsonpath='{.status.loadBalancer.ingress[0].hostname}')

# GKE (for Ambassador using HTTP)
AMBASSADORURL=http://$(kubectl get service ambassador --output jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Minikube (for Ambassador using HTTP)
AMBASSADORURL=$(minikube service --url ambassador)
```

Otherwise, look at the `LoadBalancer Ingress` line of `kubectl describe service ambassador` (or use `minikube service --url ambassador` on Minikube) and set `$AMBASSADORURL` based on that. Again, **do not include a trailing `/`**, or the examples in this document won't work.

After that, you can access your microservices by using URLs based on `$AMBASSADORURL` and the URL prefixes defined for your mappings. For example, with first the `user` mapping from above in effect:

```shell
curl $AMBASSADORURL/v1/user/health
```

would be relayed to the `usersvc` as simply `/health`;
