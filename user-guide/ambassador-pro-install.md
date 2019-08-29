# Installing Ambassador Pro
---

Ambassador Pro is a commercial version of Ambassador that includes integrated Single Sign-On, powerful rate limiting, custom filters, and more. Ambassador Pro also uses a certified version of Ambassador OSS that undergoes additional testing and validation. In this tutorial, we'll walk through the process of installing Ambassador Pro in Kubernetes and show the JWT filter in action.

Information about open source code used in Ambassador Pro can be found in `/*.opensource.tar.gz` files in each Docker image.

## 1. Clone the Ambassador Pro configuration repository
Ambassador Pro consists of a series of modules that communicate with Ambassador. The core Pro module is typically deployed as a sidecar to Ambassador. This means it is an additional process that runs on the same pod as Ambassador. Ambassador communicates with the Pro sidecar locally. Pro thus scales in parallel with Ambassador. Ambassador Pro also relies on a Redis instance for its rate limit service and several Custom Resource Definitions (CRDs) for configuration.

For this installation, we'll start with a standard set of Ambassador Pro configuration files.

```
git clone https://github.com/datawire/pro-ref-arch
```

## 2. License Key

Copy `env.sh.example` to `env.sh`, and add your specific license key to the `env.sh` file.

This license key will be loaded into a Kubernetes secret that will be referenced by Ambassador.

**Note:** Ambassador Pro will not start without a valid license key.

## 3. Deploy Ambassador Pro

If you're on GKE, first, create the following `ClusterRoleBinding`:

```
kubectl create clusterrolebinding my-cluster-admin-binding --clusterrole=cluster-admin --user=$(gcloud info --format="value(config.account)")
```

Then, deploy Ambassador Pro:

```
make apply-ambassador
```

This `make` command will use `kubectl` to deploy Ambassador Pro and a basic test configuration to the cluster.

Verify that Ambassador Pro is running:

```
kubectl get pods | grep ambassador
ambassador-79494c799f-vj2dv            2/2       Running            0         1h
ambassador-pro-redis-dff565f78-88bl2   1/1       Running            0         1h
```

**Note:** If you are not deploying in a cloud environment that supports the `LoadBalancer` type, you will need to change the `ambassador/ambassador-service.yaml` to a different service type (e.g., `NodePort`).

By default, Ambassador Pro uses ports 8500-8503.  If for whatever
reason those assignments are problematic (perhaps you [set
`service_port`](/reference/running/#running-as-non-root) to one of
those), you can set adjust these by setting environment variables:

| Purpose                        | Variable         | Default |
| ---                            | ---              | ---     |
| Filtering AuthService (gRPC)   | `APRO_AUTH_PORT` | 8500    |
| RateLimitService (gRPC)        | `GRPC_PORT`      | 8501    |
| RateLimitService debug (HTTP)  | `DEBUG_PORT`     | 8502    |
| RateLimitService health (HTTP) | `PORT`           | 8503    |

If you have deployed Ambassador with
[`AMBASSADOR_NAMESPACE`, `AMBASSADOR_SINGLE_NAMESPACE`](/reference/running/#namespaces), or
[`AMBASSADOR_ID`](/reference/running/#ambassador_id)
set, you will also need to set them in the Pro container.

## 4. Configure JWT authentication

Now that you have Ambassador Pro running, we'll show a few features of Ambassador Pro. We'll start by configuring Ambassador Pro's JWT authentication filter.

```
make apply-jwt
```

This will configure the following `FilterPolicy`:

```
---
apiVersion: getambassador.io/v1beta2
kind: FilterPolicy
metadata:
  name: httpbin-filterpolicy
  namespace: default
spec:
  # everything defaults to private; you can create rules to make stuff
  # public, and you can create rules to require additional scopes
  # which will be automatically checked
  rules:
  - host: "*"
    path: /jwt-httpbin/*
    filters:
    - name: jwt-filter
  - host: "*"
    path: /httpbin/*
    filters: null
```

Get the External IP address of your Ambassador service:

```
AMBASSADOR_IP=$(kubectl get svc ambassador -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
```

We'll now test Ambassador Pro with the `httpbin` service. First, curl to the `httpbin` URL This URL is public, so it returns successfully without an authentication token.

```
$ curl -k https://$AMBASSADOR_IP/httpbin/ip # No authentication token
{
  "origin": "108.20.119.124, 35.194.4.146, 108.20.119.124"
}
```

Send a request to the `jwt-httpbin` URL, which is protected by the JWT filter. This URL is not public, so it returns a 401.

```
$ curl -i -k https://$AMBASSADOR_IP/jwt-httpbin/ip # No authentication token
HTTP/1.1 401 Unauthorized
content-length: 58
content-type: text/plain
date: Mon, 04 Mar 2019 21:18:17 GMT
server: envoy
```

Finally, send a request with a valid JWT to the `jwt-httpbin` URL, which will return successfully.

```
$ curl -k --header "Authorization: Bearer eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ." https://$AMBASSADOR_IP/jwt-httpbin/ip
{
  "origin": "108.20.119.124, 35.194.4.146, 108.20.119.124"
}
```

## 5. Configure additional Ambassador Pro services

Ambassador Pro has many more features such as rate limiting, OAuth integration, and more.

### Enabling Rate limiting

For more information on configuring rate limiting, consult the [Advanced Rate Limiting tutorial ](/user-guide/advanced-rate-limiting) for information on configuring rate limits.

### Enabling Single Sign-On

 For more information on configuring the OAuth filter, see the [Single Sign-On with OAuth and OIDC](/user-guide/oauth-oidc-auth) documentation.

### Enabling Service Preview

Service Preview requires a command-line client, `apictl`. For instructions on configuring Service Preview, see the [Service Preview tutorial](/docs/dev-guide/service-preview).

### Enabling Consul Connect integration

Ambassador Pro's Consul Connect integration is deployed as a separate Kubernetes service. For instructions on deploying Consul Connect, see the [Consul Connect integration guide](/user-guide/consul-connect-ambassador).


# Upgrading Ambassador Pro

Follow the steps below to upgrade Ambassador Pro.

**Note**: For simplicity, we recommend storing this license key in a Kubernetes secret that can be referenced by both the certified Ambassador and Ambassador Pro containers. You can do this with the following command.

```
kubectl create secret generic ambassador-pro-license-key --from-literal=key={{AMBASSADOR_PRO_LICENSE_KEY}}
```

1. Create the `ambassador-pro-license-key` secret using the command above.

2. Upgrade to the latest image of Ambassador Pro

    ```yaml
          - name: ambassador-pro
            image: quay.io/datawire/ambassador_pro:amb-sidecar-%aproVersion%
    ```

3. Change the image of the Ambassador container to use the certified version of Ambassador.

    ```diff
          containers:
          - name: ambassador
    -       image: quay.io/datawire/ambassador:%version%
    +       image: quay.io/datawire/ambassador_pro:amb-core-%aproVersion%
    ```

4. Add the `AMBASSADOR_PRO_LICENSE_KEY` environment variable to the Ambassador container and have it get its value from the secret created in step 1.

    ```yaml
            env:
            - name: AMBASSADOR_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: AMBASSADOR_LICENSE_KEY 
              valueFrom:
                secretKeyRef:
                  name: ambassador-pro-license-key
                  key: key
    ```
  
5. Ensure the `AMBASSADOR_LICENSE_KEY` in the Ambassador Pro container is also referencing the `ambassador-pro-license-key` secret.

    ```yaml
            env:
            - name: REDIS_SOCKET_TYPE 
              value: tcp
            - name: APP_LOG_LEVEL
              value: "info"
            - name: REDIS_URL 
              value: ambassador-pro-redis:6379
            - name: AMBASSADOR_LICENSE_KEY 
              valueFrom:
                secretKeyRef:
                  name: ambassador-pro-license-key
                  key: key
    ```

After making these changes, redeploy Ambassador to receive the performance and stability improvements that certified Ambassador brings. 