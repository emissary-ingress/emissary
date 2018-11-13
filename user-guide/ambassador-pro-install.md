# Installing Ambassador Pro
---

Ambassador Pro adds built in support for single sign-on with OAuth and OIDC. In this tutorial, we'll walk through the process of installing Ambassador Pro in Kubernetes for adding single sign-on with OAuth and OIDC.

### 1. Create the Ambassador Pro registry credentials secret.
Your credentials to pull the image from the Ambassador Pro registry were given in the signup email. If you have lost this email, please contact us at support@datawire.io.

```bash
kubectl create secret docker-registry ambassador-pro-registry-credentials --docker-server=quay.io --docker-username=<CREDENTIALS USERNAME> --docker-password=<CREDENTIALS PASSWORD> --docker-email=<YOUR EMAIL>
```

### 2. Download the Ambassador Pro Deployment File 
It is recommended to deploy Ambassador Pro as a replacement for your currently running Ambassador with the Authentication service running as a sidecar to Ambassador. Download this deployment file and re-configure any changes you have made to the default Ambassador deployment such as changing the number of replicas or adding environment variables.

```
wget "https://www.getambassador.io/yaml/ambassador/ambassador-pro.yaml"
```

**Note:** It is also possible to install the Ambassador Pro Authentication service as a separate deployment from Ambassador.

```
wget "https://www.getambassador.io/yaml/ambassador/pro/ambassador-pro-auth.yaml"
```

### 3. Define the Ambassador Pro Authentication service

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
      kind:  AuthService
      name:  authentication
      auth_service: ambassador-pro
      allowed_headers:
       - "Authorization"
       - "Client-Id"
       - "Client-Secret"
spec:
  selector:
    name: ambassador-pro
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: ClusterIP
```

### 4. Configure Ambassador Pro Authentication
The Ambassador Pro Authentication service is configured with environment variables set in the deployment file. Configure the `AUTH_CALLBACK_URL`, `AUTH_DOMAIN`, `AUTH_AUDIENCE`, and `AUTH_CLIENT_ID` variables by following the [Single Sign-On with OAuth and OIDC](/user-guide/oauth-oidc-auth) guide.

### 5. Apply the Ambassador Pro Deployment

```
kubectl apply -f ambassador-pro.yaml
```

### More
For more details on installing Ambassador Pro and the Authentication service, read the documentation on configuring [Single Sign-On with OAuth and OIDC](/user-guide/oauth-oidc-auth) and [Access Controls](/reference/services/access-control.md). You can also email us at support@datawire.io for more info.