# Installing Ambassador Pro
---

Ambassador Pro is a commercial version of Ambassador that includes integrated SSO, flexible rate limiting, and more. In this tutorial, we'll walk through the process of installing Ambassador Pro in Kubernetes.

### 1. Create the Ambassador Pro registry credentials secret.
Your credentials to pull the image from the Ambassador Pro registry were given in the sign up email. If you have lost this email, please contact us at support@datawire.io.

```
kubectl create secret docker-registry ambassador-pro-registry-credentials --docker-server=quay.io --docker-username=<CREDENTIALS USERNAME> --docker-password=<CREDENTIALS PASSWORD> --docker-email=<YOUR EMAIL>
```
- `<CREDENTIALS USERNAME>`: Username given in sign up email
- `<CREDNETIALS PASSWORD>`: Password given in sign up email
- `<YOUR EMAIL>`: Your email address

### 2. Download the Ambassador Pro Deployment File 
Ambassador Pro is deployed as a sidecar to Ambassador. In addition, Ambassador Pro also relies on a Redis instance for its rate limit service and a couple of Custom Resource Definitions (CRDs) for configuration. The default configuration for Ambassador Pro is available at https://www.getambassador.io/yaml/ambassador/pro/ambassador-pro.yaml. Download this file locally:

```
curl -O "https://www.getambassador.io/yaml/ambassador/pro/ambassador-pro.yaml"
```

Next, ensure the `namespace` field in the `ClusterRoleBinding` is configured correctly for your particular deployment. If you are not installing Ambassador into the `default` namespace, you will need to update this file accordingly.

**Note:** Ambassador 0.40.2 and below does not support v1 `AuthService` configurations. If you are using a lower version of Ambassador, replace the `AuthService` in the downloaded YAML with:

```
      ---
      apiVersion: ambassador/v0
      kind: AuthService
      name: authentication
      auth_service: ambassador-pro
      allowed_headers:
      - "Client-Id"
      - "Client-Secret"
      - "Authorization"
```

### 3. License Key

In the `ambassador-pro.yaml` file, update all the `AMBASSADOR_LICENSE_KEY` environment variable fields with the license key that is supplied as part of your trial email.

**Note:** The Ambassador Pro will not start without your license key.

### 4. Single Sign-On

Ambassador Pro's authentication service requires a URL for your authentication provider. This will be the URL Ambassador Pro will direct to for authentication. 

If you are using Auth0, this will be the name of the tenant you created (e.g `datawire-ambassador`). To create an Auth0 tenant, go to auth0.com and sign up for a free account. Once you have created an Auth0 tenant, the full `AUTH_PROVIDER_URL` is `https://<auth0-tenant-name>.auth0.com`. 

You can also find this as the domain for your application.

![](/images/Auth0_domain_clientID.png)

Add this as the `AUTH_PROVIDER_URL` in your Ambassador Pro deployment manifest.

```
  env:
  # Auth provider's absolute url: {scheme}://{host}
    - name: AUTH_PROVIDER_URL
      value: https://datawire-ambassador.auth0.com
```

Next, you will need to configure a tenant resource for Ambassador Pro to authenticate against. Details on how to configure this can be found in the [Single Sign-On with OAuth & OIDC](/user-guide/oauth-oidc-auth#configure-your-authentication-tenants) documentation.

**Note:** The Ambassador Pro will not start without this value configured.

### 6. Deploy Ambassador Pro

Once you have fully configured Ambassador Pro, deploy the your configuration:

```
kubectl apply -f ambassador-pro.yaml
```

Verify that Ambassador Pro is running:

```
kubectl get pods | grep ambassador
ambassador-79494c799f-vj2dv        2/2       Running            0         1h
ambassador-pro-redis-dff565f78-88bl2   1/1       Running            0         1h
```

### 7. Create Ambassador Pro Services

Once Ambassador Pro is deployed, you will need to create services to expose the Pro resources to Ambassador. Create these services with `kubectl`:

**SSO Authentication Service**
```bash
kubectl apply -f https://www.getambassador.io/yaml/ambassador/pro/ambassador-pro-auth.yaml
```

**Rate Limiting Service***
```bash
kubectl apply -f https://www.getambassador.io/yaml/ambassador/pro/ambassador-pro-ratelimit.yaml

### More

For more details on Ambassador Pro, see:

* [Single Sign-On with OAuth and OIDC](/user-guide/oauth-oidc-auth) for information about configuring SSO
* [Advanced Rate Limiting](/user-guide/advanced-rate-limiting) for information on configuring rate limiting

