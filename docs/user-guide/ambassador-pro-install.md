# Installing Ambassador Pro
---

Ambassador Pro is a commercial version of Ambassador that includes integrated SSO, flexible rate limiting, and more. In this tutorial, we'll walk through the process of installing Ambassador Pro in Kubernetes.

### 1. Install and Configure Ambassador
This guide assumes you have Ambassador installed and configured. If this is not the case, follow the [Ambassador installation guide](/user-guide/getting-started) to get Ambassador running before continuing.

### 2. Create the Ambassador Pro registry credentials secret.
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
curl -O "https://www.getambassador.io/yaml/ambassador/pro/ambassador-pro-auth.yaml"
```

Next, ensure the `namespace` field in the `ClusterRoleBinding` is configured correctly for your particular deployment. If you are not installing Ambassador into the `default` namespace, you will need to update this file accordingly.

### 4. Single-Sign On

Ambassador Pro's authentication service requires some additional information about your Identity Provider. This is done by configuring environment variables in the deployment manifest. 

```
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: ambassador-pro-auth
spec:
  replicas: 1
  selector:
    matchLabels:
      service: ambassador-pro-auth
  template:
    metadata:
      labels:
        service: ambassador-pro-auth
    spec:
      serviceAccountName: ambassador-pro
      containers:
      - name: ambassador-pro
        image: quay.io/datawire/ambassador-pro:0.0.6
        ports:
        - containerPort: 8080
        env:
#         Configure to your callback URL
          - name: AUTH_CALLBACK_URL
            value: ""
#         Configure to your Auth0 domain
          - name: AUTH_DOMAIN
            value: ""
#         Configure to your Auth0 API Audience
          - name: AUTH_AUDIENCE
            value: ""
#         Configure to your Auth0 Application client ID
          - name: AUTH_CLIENT_ID
            value: ""
#          Uncomment if you want the Auth0 management API to validate your configurations
#          - name: AUTH_CLIENT_SECRET
#            value: <CLIENT SECRET>
      imagePullSecrets:
      - name: ambassador-pro-registry-credentials
```

Configure the `AUTH_CALLBACK_URL`, `AUTH_DOMAIN`, `AUTH_AUDIENCE`, and `AUTH_CLIENT_ID` variables by following the [Single Sign-On with OAuth and OIDC](/user-guide/oauth-oidc-auth) guide.

After configuring the authentication, you will need to configure an `AuthService` for Ambassador. You can do this by updating the Ambassador Pro Kubernetes service, like the below:

```
---
apiVersion: v1
kind: Service
metadata:
  name: ambassador-pro
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v0
      kind: RateLimitService
      name: ambassador-pro
      service: "ambassador-pro:8081"
      ---
      apiVersion: ambassador/v0
      kind:  AuthService
      name:  authentication
      auth_service: ambassador-pro
      allowed_headers:
        - "Authorization"
        - "Client-Id"
        - "Client-Secret"
...
```

(For the sake of brevity, the full Kubernetes service is not duplicated above.)

### 5. Deploy Ambassador Pro

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

Restart Ambassador once Pro is deployed so it will update the `AuthService` and `RateLimitService` configuration. You can do this by deleting the Ambassador pods and letting the deployment redeploy the pods.

### More
For more details on Ambassador Pro, see:

* [Single Sign-On with OAuth and OIDC](/user-guide/oauth-oidc-auth) for information about configuring SSO
* [Advanced Rate Limiting](/user-guide/advanced-rate-limiting) for information on configuring rate limiting