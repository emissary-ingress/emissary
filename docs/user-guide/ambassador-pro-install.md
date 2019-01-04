# Installing Ambassador Pro
---

Ambassador Pro is a commercial version of Ambassador that includes integrated SSO, flexible rate limiting, and more. In this tutorial, we'll walk through the process of installing Ambassador Pro in Kubernetes.

### 1. Install and Configure Ambassador
Install and configure Ambassador. Ambassador Pro requires Ambassador version `0.50.0-rc2` and above.

Download Ambassador and upgrade the image to `quay.io/datawire/ambassador:0.50.0-rc2`.

```
curl -O https://www.getambassador.io/yaml/ambassador/ambassador-rbac.yaml
```

```diff
    spec:
      serviceAccountName: ambassador
      containers:
      - name: ambassador
-       image: quay.io/datawire/ambassador:0.40.2
+       image: quay.io/datawire/ambassador:0.50.0-rc2
```
Deploy Ambassador and the LoadBalancer service. 

```
kubectl apply -f ambassador-rbac.yaml
kubectl apply -f https://getambassador.io/yaml/ambassador/ambassador-service.yaml
```

Note: If you are using GKE, you will need additional privileges:

```
kubectl create clusterrolebinding my-cluster-admin-binding --clusterrole=cluster-admin --user=$(gcloud info --format="value(config.account)")
```

For more detailed instructions on installing Ambassador, please see the [Ambassador installation guide](/user-guide/getting-started).

### 2. Create the Ambassador Pro registry credentials secrets.
Your credentials to pull the image from the Ambassador Pro registry were given in the signup email. If you have lost this email, please contact us at support@datawire.io.

```
kubectl create secret docker-registry ambassador-pro-registry-credentials --docker-server=quay.io --docker-username=<CREDENTIALS USERNAME> --docker-password=<CREDENTIALS PASSWORD> --docker-email=<YOUR EMAIL>
```
- `<CREDENTIALS USERNAME>`: Username given in signup email
- `<CREDNETIALS PASSWORD>`: Password given in signup email
- `<YOUR EMAIL>`: Your email address

### 3. Download the Ambassador Pro Deployment File 
Ambassador Pro is deployed as an additional set of Kubernetes services that integrate with Ambassador. In addition, Ambassador Pro also relies on a Redis instance for its rate limit service. The default configuration for Ambassador Pro is available at https://www.getambassador.io/yaml/ambassador/pro/ambassador-pro.yaml. Download this file locally:

```
curl -O "https://www.getambassador.io/yaml/ambassador/pro/ambassador-pro.yaml"
```

Next, ensure the `namespace` field in the `ClusterRoleBinding` is configured correctly for your particular deployment. If you are not installing Ambassador into the `default` namespace, you will need to update this file accordingly.

### 4. License Key

In the `ambassador-pro.yaml` file, update the `AMBASSADOR_LICENSE_KEY` environment variable with the license key that is supplied as part of your trial email.

**Note:** The Ambassador Pro rate limit container will not properly start without your license key.

### 5. Single Sign-On

Ambassador Pro's authentication service requires a URL for your authentication provider. This will be the URL Ambassador Pro will direct to for authentication. If you are using Auth0, this URL with be the Domain of your Auth0 application. This can be found here:

![](/images/Auth0_domain_clientID.png)

Add this as the `AUTH_PROVIDER_URL` in your Ambassador Pro deployment manifest.

```
- name: auth
  env:
  # Auth provider's abolute url: {scheme}://{host}
    - name: AUTH_PROVIDER_URL
      value: https://datawire-ambassador.auth0.com
```

Next, you will need to configure a tenant for Ambassador Pro to authenticate against. Details on how to configure this can be found in the [Single Sign-On with OAuth & OIDC](/user-guide/oauth-oidc-auth#configure-your-authentication-tenants) documentation.

**Note:** The Ambassador Pro authentication container will not properly start without this set.

### 6. Deploy Ambassador Pro

Once you have fully configured Ambassador Pro, deploy the your configuration:

```
kubectl apply -f ambassador-pro.yaml
```

Verify that Ambassador Pro is running:

```
kubectl get pods | grep pro
ambassador-pro-79494c799f-vj2dv        2/2       Running            0         1h
ambassador-pro-redis-dff565f78-88bl2   1/1       Running            0         1h
```

### 6. Restart Ambassador

Restart Ambassador once Ambassador Pro is deployed so it will update the `AuthService` and `RateLimitService` configuration. You can do this by deleting the Ambassador pods and letting the deployment redeploy the pods.

### More

For more details on Ambassador Pro, see:

* [Single Sign-On with OAuth and OIDC](/user-guide/oauth-oidc-auth) for information about configuring SSO
* [Advanced Rate Limiting](/user-guide/advanced-rate-limiting) for information on configuring rate limiting

