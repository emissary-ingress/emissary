# Upgrade from the API Gateway to the Ambassador Edge Stack

If you currently have the open source API Gateway version of Ambassador, you can upgrade to the Ambassador Edge Stack with a few simple commands.

**Prerequisites**:

* You must have properly installed Ambassador previously following [these](/user-guide/install-ambassador-oss) instructions.
* You must have TLS configured and working properly on your Ambassador instance following [these](/user-guide/tls-termination/) instructions.

**To upgrade your instance of Ambassador**:

1. [Apply the Migration Manifest](/user-guide/upgrade-from-oss-to-edge-stack#1-apply-the-migration-manifest)
2. [Test the New Deployment](/user-guide/upgrade-from-oss-to-edge-stack#2-test-the-new-deployment)
3. [Redirect Traffic from Ambassador to AES](/user-guide/upgrade-from-oss-to-edge-stack#3-redirect-traffic-from-ambassador-to-aes)
4. [Delete the Old Deployment](/user-guide/upgrade-from-oss-to-edge-stack#4-delete-the-old-deployment)
5. [Update the CRDs](/user-guide/upgrade-from-oss-to-edge-stack#5-update-the-crds)
6. [Apply New Resources](/user-guide/upgrade-from-oss-to-edge-stack#6-apply-new-resources)
7. [Restart the Pods](/user-guide/upgrade-from-oss-to-edge-stack#7-restart-the-pods)
8. [What's Next?](/user-guide/upgrade-from-oss-to-edge-stack#8-whats-next)

## 1. Apply the Migration Manifest

First off, install AES alongside your Ambassador installation, so you can test your workload against the new deployment.

Note: Make sure you apply the manifests in the same namespace as your current Ambassador installation.

Use the following command to install AES:

```
cat << EOF | kubectl apply -n <namespace> -f -
---
########################################
# updated RBAC permissions
########################################
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: ambassador
  labels:
    product: aes
rules:
# Regular Ambassador access requirements
- apiGroups: [""]
  resources: [ "endpoints", "namespaces", "services" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "getambassador.io" ]
  resources: [ "*" ]
  verbs: ["get", "list", "watch", "update", "patch", "create", "delete" ]
- apiGroups: [ "apiextensions.k8s.io" ]
  resources: [ "customresourcedefinitions" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "networking.internal.knative.dev" ]
  resources: [ "clusteringresses", "ingresses" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "networking.internal.knative.dev" ]
  resources: [ "ingresses/status", "clusteringresses/status" ]
  verbs: ["update"]
- apiGroups: [ "extensions" ]
  resources: [ "ingresses" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "extensions" ]
  resources: [ "ingresses/status" ]
  verbs: ["update"]
# Ambassador Pro access requirements.  Also gets "create" for secrets.
- apiGroups: [""]
  resources: [ "secrets" ]
  verbs: ["get", "list", "watch", "create", "update"]
---
########################################
# redis deployment
########################################
apiVersion: v1
kind: Service
metadata:
  name: ambassador-redis
  labels:
    product: aes
spec:
  type: ClusterIP
  ports:
  - port: 6379
    targetPort: 6379
  selector:
    service: ambassador-redis
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ambassador-redis
  labels:
    product: aes
spec:
  replicas: 1
  selector:
    matchLabels:
      service: ambassador-redis
  template:
    metadata:
      labels:
        service: ambassador-redis
    spec:
      containers:
      - name: redis
        image: redis:5.0.1
      restartPolicy: Always

---
apiVersion: v1
kind: Secret
metadata:
 name: ambassador-edge-stack
data:
 license-key: ""
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    product: aes
  name: aes
spec:
  replicas: 1
  selector:
    matchLabels:
      product: aes
  template:
    metadata:
      annotations:
        consul.hashicorp.com/connect-inject: 'false'
        sidecar.istio.io/inject: 'false'
      labels:
        product: aes
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchLabels:
                  product: aes
              topologyKey: kubernetes.io/hostname
            weight: 100
      containers:
      - env:
        - name: AMBASSADOR_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: REDIS_URL
          value: ambassador-redis:6379
        - name: AMBASSADOR_URL
          value: https://ambassador/
        - name: POLL_EVERY_SECS
          value: '60'
        - name: AMBASSADOR_INTERNAL_URL
          value: https://127.0.0.1:8443
        - name: AMBASSADOR_ADMIN_URL
          value: http://127.0.0.1:8877
        image: quay.io/datawire-dev/aes:0.99.0-rc-latest
        imagePullPolicy: Always
        livenessProbe:
          httpGet:
            path: /ambassador/v0/check_alive
            port: 8877
          periodSeconds: 3
        name: aes
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 8443
          name: https
        - containerPort: 8877
          name: admin
        readinessProbe:
          httpGet:
            path: /ambassador/v0/check_ready
            port: 8877
          periodSeconds: 3
        volumeMounts:
        - mountPath: /tmp/ambassador-pod-info
          name: ambassador-pod-info
        - mountPath: /.config/ambassador
          name: ambassador-edge-stack-secrets
          readOnly: true
      imagePullSecrets:
      - name: aes-pull-secret
      restartPolicy: Always
      securityContext:
        runAsUser: 8888
      serviceAccountName: ambassador
      terminationGracePeriodSeconds: 0
      volumes:
      - downwardAPI:
          items:
          - fieldRef:
              fieldPath: metadata.labels
            path: labels
        name: ambassador-pod-info
      - name: ambassador-edge-stack-secrets
        secret:
          secretName: ambassador-edge-stack

---
apiVersion: v1
kind: Service
metadata:
  name: test-aes
  labels:
    product: aes
spec:
  type: LoadBalancer
  externalTrafficPolicy: Local
  ports:
  - name: http
    port: 80
    targetPort: http
  - name: https
    port: 443
    targetPort: https
  selector:
    product: aes
EOF
```

## 2. Test the New Deployment

At this point, you have Ambassador and AES running side by side in your cluster. AES is configured using the same configuration (Mappings, Modules, etc) as current Ambassador.

Get IP address to connect to AES by running the following command:
`kubectl get service test-aes -n <namespace>`

Test that AES is working properly.

## 3. Redirect Traffic from Ambassador to AES

Once you’re satisfied with the new deployment, update your current Ambassador service to point to AES.

Edit the current Ambassador deployment with `kubectl edit deployment -n <namespace> ambassador` and change the selector to "product: aes”.

## 4. Delete the Old Deployment

You can now safely delete the older Ambassador deployment and AES service.

```
kubectl delete deployment -n <namespace> ambassador
kubectl delete service -n <namespace> test-aes
```

## 5. Update CRDs

You should now apply the new CRDs that AES uses:
```
kubectl apply -f https://deploy-preview-91--datawire-ambassador.netlify.com/yaml/aes-crds.yaml
```

## 6. Apply the New Resources

Finally, apply the remaining AES manifests using the following command:

```
cat<<EOF | kubectl apply -n <namespace> -f -
---
apiVersion: getambassador.io/v2
kind: RateLimitService
metadata:
 name: ambassador-edge-stack-ratelimit
 labels:
   product: aes
spec:
 service: "127.0.0.1:8500"
---
apiVersion: getambassador.io/v2
kind: AuthService
metadata:
 name: ambassador-edge-stack-auth
 labels:
   product: aes
spec:
 proto: grpc
 status_on_error:
   code: 504
 auth_service: "127.0.0.1:8500"
 allow_request_body: false # setting this to 'true' allows Plugin and External filters to access the body, but has performance overhead
######################################################################
# Configure DevPortal
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
 # This Mapping name is referenced by convention, it's important to leave as-is.
 name: ambassador-devportal
 labels:
   product: aes
spec:
 prefix: /documentation/
 rewrite: "/docs/"
 service: "127.0.0.1:8500"
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
 # This Mapping name is what the demo uses. Sigh.
 name: ambassador-devportal-demo
 labels:
   product: aes
spec:
 prefix: /docs/
 rewrite: "/docs/"
 service: "127.0.0.1:8500"
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
 # This Mapping name is referenced by convention, it's important to leave as-is.
 name: ambassador-devportal-api
 labels:
   product: aes
spec:
 prefix: /openapi/
 rewrite: ""
 service: "127.0.0.1:8500"
---
apiVersion: getambassador.io/v2
kind: FilterPolicy
metadata:
 name: ambassador-internal-access-control
 labels:
   product: aes
spec:
 rules:
   - host: "*"
     path: "*/.ambassador-internal/*"
     filters:
       - name: ambassador-internal-access-control
---
apiVersion: getambassador.io/v2
kind: Filter
metadata:
 name: ambassador-internal-access-control
 labels:
   product: aes
spec:
 Internal: {}
EOF
```

## 6. Restart the Pods

Finally, restart the AES pods so they pick up the new RBAC permissions.

```
kubectl delete pods -l product=aes
```

## 7. What's Next?

Now that you have the Ambassador Edge Stack up and running, check out the [Getting Started](/user-guide/getting-started) guide for recommendations on what to do next and take full advantage of its features.
