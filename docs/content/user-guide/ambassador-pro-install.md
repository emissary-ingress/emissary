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
- `<CREDENTIALS USERNAME>`: Username given in signup email
- `<CREDNETIALS PASSWORD>`: Password given in signup email
- `<YOUR EMAIL>`: Your email address

### 3. Download the Ambassador Pro Deployment File 
Ambassador Pro plugins run externally to Ambassador. You will need to download the Ambassador Pro Authentication service deployment in order to change the default values and integrate it with your Auth0 application. Download Ambassador Pro Authentication from https://www.getambassador.io/yaml/ambassador/pro/ambassador-pro-auth.yaml or with:

```
wget "https://www.getambassador.io/yaml/ambassador/pro/ambassador-pro-auth.yaml"
```
**Note:** This file is configured to deploy in the default namespace. Ensure the `namespace` field in the `ClusterRoleBinding` is configured correctly.

```
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: ambassador-pro
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ambassador-pro
subjects:
- kind: ServiceAccount
  name: ambassador-pro
# Ensure Your namespace is configured correctly
  namespace: default 
```

### 4. Configure and Deploy Ambassador Pro Authentication
You will need to connect Ambassador Pro Authentication with your OAuth/OIDC Identify Provider. This is done by configuring environment variables in the deployment manifest.

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

### More
For more details on installing Ambassador Pro and the Authentication service, read the documentation on configuring [Single Sign-On with OAuth and OIDC](/user-guide/oauth-oidc-auth) and [Access Controls](/reference/services/access-control). You can also email us at support@datawire.io for more info.