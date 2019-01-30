# Installing Ambassador Pro
---

Ambassador Pro is a commercial version of Ambassador that includes integrated Single Sign-On, powerful rate limiting, and more. In this tutorial, we'll walk through the process of installing Ambassador Pro in Kubernetes.

## 1. Create the Ambassador Pro registry credentials secret.
Your credentials to pull the image from the Ambassador Pro registry were given in the sign up email. If you have lost this email, please contact us at support@datawire.io.

```
kubectl create secret docker-registry ambassador-pro-registry-credentials --docker-server=quay.io --docker-username=<CREDENTIALS USERNAME> --docker-password=<CREDENTIALS PASSWORD> --docker-email=<YOUR EMAIL>
```
- `<CREDENTIALS USERNAME>`: Username given in sign up email
- `<CREDENTIALS PASSWORD>`: Password given in sign up email
- `<YOUR EMAIL>`: Your email address

## 2. Download the Ambassador Pro Deployment File 
Ambassador Pro consists of a series of modules that communicate with Ambassador. The core Pro module is typically deployed as a sidecar to Ambassador. This means it is an additional process that runs on the same pod as Ambassador. Ambassador communicates with the Pro sidecar locally. Pro thus scales in parallel with Ambassador. Ambassador Pro also relies on a Redis instance for its rate limit service and several Custom Resource Definitions (CRDs) for configuration.

The full configuration for Ambassador Pro is available at https://www.getambassador.io/yaml/ambassador/pro/ambassador-pro.yaml. Download this file locally:

```
curl -O "https://www.getambassador.io/yaml/ambassador/pro/ambassador-pro.yaml"
```

Next, ensure the `namespace` field in the `ClusterRoleBinding` is configured correctly for your particular deployment. If you are not installing Ambassador into the `default` namespace, you will need to update this file accordingly.

**Note:** Ambassador 0.40.2 and below does not support v1 `AuthService` configurations. If you are using an older version of Ambassador, replace the `AuthService` in the downloaded YAML with:

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

## 3. License Key

In the `ambassador-pro.yaml` file, update the `AMBASSADOR_LICENSE_KEY` environment variable field with the license key that is supplied as part of your trial email.

**Note:** The Ambassador Pro will not start without your license key.

## 4. Deploy Ambassador Pro

Once you have fully configured Ambassador Pro, deploy your updated configuration. Note that the default configuration will also redeploy your current Ambassador configuration, so verify that you have the correct Ambassador version before deploying Pro.

```
kubectl apply -f ambassador-pro.yaml
```

Verify that Ambassador Pro is running:

```
kubectl get pods | grep ambassador
ambassador-79494c799f-vj2dv            2/2       Running            0         1h
ambassador-pro-redis-dff565f78-88bl2   1/1       Running            0         1h
```

## 5. Configure Ambassador Pro services

Ambassador should now be running, along with the Pro modules. To enable rate limiting and authentication, some additional configuration is required.

### Enabling Rate limiting

Deploy the Kubernetes service that enables rate limiting. You will then want to review the [Advanced Rate Limiting tutorial ](/user-guide/advanced-rate-limiting) for information on configuring rate limits.

```bash
kubectl apply -f https://www.getambassador.io/yaml/ambassador/pro/ambassador-pro-ratelimit.yaml
```

### Enabling Single Sign-On

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

Redeploy the Ambassador Pro deployment manifest, along with the Ambassador Pro auth service:

```bash
kubectl apply -f ambassador-pro.yaml
kubectl apply -f https://www.getambassador.io/yaml/ambassador/pro/ambassador-pro-auth.yaml
```

Finally, you will need to configure a tenant resource for Ambassador Pro to authenticate against. Details on how to configure this can be found in the [Single Sign-On with OAuth & OIDC](/user-guide/oauth-oidc-auth#configure-your-authentication-tenants) documentation.

### Enabling Service Preview

Service Preview requires a command-line client, `apictl`. For instructions on configuring Service Preview, see the [Service Preview tutorial](/docs/dev-guide/service-preview).

### Enabling Consul Connect integration

Ambassador Pro's Consul Connect integration is deployed as a separate Kubernetes service. For instructions on deploying Consul Connect, see the [Consul Connect integration guide](/user-guide/consul-connect-ambassador).
