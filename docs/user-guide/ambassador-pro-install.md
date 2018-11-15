# Installing Ambassador Pro
---

Ambassador Pro adds built in support for single sign-on with OAuth and OIDC. In this tutorial, we'll walk through the process of installing Ambassador Pro in Kubernetes for adding single sign-on with OAuth and OIDC. 

### 1. Install and Configure Ambassador
This guide assumes you have Ambassador installed and configured correctly apart from the Authentication service. If this is not the case, follow the [Ambassador installation guide](/user-guide/getting-started) to get Ambassador running before continuing.

### 2. Create the Ambassador Pro registry credentials secret.
Your credentials to pull the image from the Ambassador Pro registry were given in the signup email. If you have lost this email, please contact us at support@datawire.io.

```
kubectl create secret docker-registry ambassador-pro-registry-credentials --docker-server=quay.io --docker-username=<CREDENTIALS USERNAME> --docker-password=<CREDENTIALS PASSWORD> --docker-email=<YOUR EMAIL>
```

### 3. Download the Ambassador Pro Deployment File 
Ambassador Pro plugins run externally to Ambassador. You will need to download the Ambassador Pro Authentication service deployment in order to change the default values and sync it with your Auth0 application. Download Ambassador Pro Authentication from https://www.getambassador.io/yaml/ambassador/pro/ambassador-pro-auth.yaml or with:

```
wget "https://www.getambassador.io/yaml/ambassador/pro/ambassador-pro-auth.yaml"
```
**Note:** This file is configured to deploy in the default namespace. Ensure the `namespace` field in the `ClusterRoleBinding` is configured correctly.

### 4. Define the Ambassador Pro Authentication service
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

### 5. Configure and Deploy Ambassador Pro Authentication
You will need to connect Ambassador Pro Authentication with you OAuth/OIDC Identify Provider. This is done by configuring environment variables in the deployment manifest. Configure the `AUTH_CALLBACK_URL`, `AUTH_DOMAIN`, `AUTH_AUDIENCE`, and `AUTH_CLIENT_ID` variables by following the [Single Sign-On with OAuth and OIDC](/user-guide/oauth-oidc-auth) guide.

### More
For more details on installing Ambassador Pro and the Authentication service, read the documentation on configuring [Single Sign-On with OAuth and OIDC](/user-guide/oauth-oidc-auth) and [Access Controls](/reference/services/access-control). You can also email us at support@datawire.io for more info.