# Upgrading to Ambassador Pro

If you are already using Ambassador open source, upgrading to using Ambassador Pro is straight-forward. In this demo we will walk-through integrating Ambassador Pro into your currently running Ambassador instance and show how quickly you can secure your APIs with JWT authentication.

## 1. Install Ambassador Pro Resources

   Ambassador Pro relies on several Custom Resource Definition (CRDs) for configuration as well are requires a redis instance for rate limiting.

   We have published these resources for download at https://www.getambassador.io/yaml/ambassador-pro/upgrade.yaml or you can easily install them using `kubectl`.

   ```
   kubectl apply -f https://www.getambassador.io/yaml/ambassador/pro/upgrade.yaml
   ```

## 2. Modify Ambassador Deployment

   Ambassador Pro is typically deployed as a sidecar to Ambassador allowing Ambassador to communicate with Pro services locally.

   To upgrade your current Ambassador instance to Ambassador Pro, you will need to edit Ambassador's deployment YAML. A full deployment will look something like this:

   ```yaml
   ---
   apiVersion: extensions/v1beta1
   kind: Deployment
   metadata:
     name: ambassador
   spec:
     replicas: 3
     selector:
       matchLabels:
         service: ambassador
     template:
       metadata:
         annotations:
           sidecar.istio.io/inject: "false"
           "consul.hashicorp.com/connect-inject": "false"
         labels:
           service: ambassador
       spec:
         affinity:
           podAntiAffinity:
             preferredDuringSchedulingIgnoredDuringExecution:
             - weight: 100
               podAffinityTerm:
                 labelSelector:
                   matchLabels:
                     service: ambassador
                 topologyKey: kubernetes.io/hostname
         serviceAccountName: ambassador
         containers:
         - name: ambassador
           image: quay.io/datawire/ambassador:%version%
           resources:
             limits:
               cpu: 1
               memory: 400Mi
             requests:
               cpu: 200m
               memory: 100Mi
           env:
           - name: AMBASSADOR_NAMESPACE
             valueFrom:
               fieldRef:
                 fieldPath: metadata.namespace
           ports:
           - name: http
             containerPort: 80
           - name: https
             containerPort: 443
           - name: admin
             containerPort: 8877
           livenessProbe:
             httpGet:
               path: /ambassador/v0/check_alive
               port: 8877
             initialDelaySeconds: 30
             periodSeconds: 3
           readinessProbe:
             httpGet:
               path: /ambassador/v0/check_ready
               port: 8877
             initialDelaySeconds: 30
             periodSeconds: 3
         - name: ambassador-pro
           image: quay.io/datawire/ambassador_pro:amb-sidecar-%aproVersion%
           ports:
           - name: ratelimit-grpc
             containerPort: 8081
           - name: ratelimit-debug
             containerPort: 6070
           - name: auth-http
             containerPort: 8082
           env:
           - name: REDIS_SOCKET_TYPE 
             value: tcp
           - name: REDIS_URL 
             value: ambassador-pro-redis:6379
           - name: AMBASSADOR_LICENSE_KEY 
             value: ""
         restartPolicy: Always
   ```

   As you can see, the only difference between this deployment and the default deployment [here](https://www.getambassador.io/yaml/ambassador/ambassador-no-rbac.yaml), is the addition of the `ambassador-pro` container. Adding this container to your Ambassador deployment and applying the YAML will install Ambassador Pro.


   ```yaml
         ...
         - name: ambassador-pro
           image: quay.io/datawire/ambassador_pro:amb-sidecar-%aproVersion%
           ports:
           - name: ratelimit-grpc
             containerPort: 8081
           - name: ratelimit-debug
             containerPort: 6070
           - name: auth-http
             containerPort: 8082
           env:
           - name: REDIS_SOCKET_TYPE 
             value: tcp
           - name: REDIS_URL 
             value: ambassador-pro-redis:6379
           - name: AMBASSADOR_LICENSE_KEY 
             value: ""
   ```

   **Note:** Make sure to put your license key in the `AMBASSADOR_LICENSE_KEY` environment variable.

## 3. Configure Additional Ambassador Pro Services

Ambassador Pro has many more features such as rate limiting, OAuth integration, and more.

### Enabling Rate limiting

For more information on configuring rate limiting, consult the [Advanced Rate Limiting tutorial ](/user-guide/advanced-rate-limiting) for information on configuring rate limits.

### Enabling Single Sign-On

 For more information on configuring the OAuth filter, see the [Single Sign-On with OAuth and OIDC](/user-guide/oauth-oidc-auth) documentation.

### Enabling Service Preview

Service Preview requires a command-line client, `apictl`. For instructions on configuring Service Preview, see the [Service Preview tutorial](/docs/dev-guide/service-preview).

### Enabling Consul Connect integration

Ambassador Pro's Consul Connect integration is deployed as a separate Kubernetes service. For instructions on deploying Consul Connect, see the [Consul Connect integration guide](/user-guide/consul-connect-ambassador).
