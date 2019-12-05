# Upgrade from the API Gateway to the Ambassador Edge Stack

If you currently have the open source API Gateway version of Ambassador, you can upgrade to the Ambassador Edge Stack with a few simple commands.

**Prerequistes**:

* You must have properly installed Ambassador previously following [these](/user-guide/install-ambassador-oss) instructions.
* You must have TLS configured and working properly on your Ambassador instance following [these](/user-guide/tls-termination/) instructions.

**To upgrade your instance of Ambassador**:

1. [Install Redis](/user-guide/upgrade-from-oss-to-edge-stack#1-install-redis)
2. [Create an Empty Secret CRD](user-guide/upgrade-from-oss-to-edge-stack#2-create-an-empty-secret-crd)
3. [Create a New Deployment](/user-guide/upgrade-from-oss-to-edge-stack#3-create-a-new-deployment)
4. [Delete the Old Deployment](/user-guide/upgrade-from-oss-to-edge-stack#4-delete-the-old-deployment)
5. [Update CRDs](/user-guide/upgrade-from-oss-to-edge-stack#5-update-crds)
6. [Update RBAC Permissions](/user-guide/upgrade-from-oss-to-edge-stack#6-update-rbac-permissions)
7. [Apply the aes Configuration](/user-guide/upgrade-from-oss-to-edge-stack#7-apply-the-aes-configuration)
8. [What's Next?](/user-guide/upgrade-from-oss-to-edge-stack#8-whats-next)

## 1. Install Redis

Before you can upgrade, you'll need to install the Redis deployment. Use the following command:

```
cat << EOF | kubectl apply -n default -f -
---
apiVersion: apps/v1
kind: Deployment
metadata:
 labels:
   product: aes
 name: ambassador-redis-aes
spec:
 replicas: 1
 selector:
   matchLabels:
     product: aes
     service: ambassador-redis
 template:
   metadata:
     labels:
       product: aes
       service: ambassador-redis
   spec:
     containers:
       - image: redis:5.0.1
         name: redis
     restartPolicy: Always
---
apiVersion: v1
kind: Service
metadata:
 labels:
   product: aes
 name: ambassador-redis
spec:
 ports:
 - port: 6379
   targetPort: 6379
 selector:
   product: aes
   service: ambassador-redis
 type: ClusterIP
EOF
```

## 2. Create an Empty Secret CRD

Next, use the following command to create an empty secret CRD to use in your k8s cluster:

```
cat <<EOF | kubectl apply -f -
---
apiVersion: v1
kind: Secret
metadata:
  name: ambassador-edge-stack
data:
  license-key: "" # This secret is just a placeholder, it is mounted as a volume and refreshed when changed
EOF
```

## 3. Create a New Deployment

Create the new `aes` deployment in the same namespace in as your current deployment; in the following file, replace `default` with your own namespace. Use the following command:

```
cat <<EOF | kubectl apply -n default -f
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
      service: ambassador
  template:
    metadata:
      annotations:
        consul.hashicorp.com/connect-inject: 'false'
        sidecar.istio.io/inject: 'false'
      labels:
        service: ambassador
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchLabels:
                  service: ambassador
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
          value: https://ambassador.default.svc.cluster.local
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
```

## 4. Delete the Old Deployment

Use the following command to delete your deployment of the API Gateway:

```
kubectl delete deployment -n default ambassador
```

## 5. Update CRDs

Use the following command to update your CRDs:

```
kubectl apply -f https://deploy-preview-91--datawire-ambassador.netlify.com/yaml/aes-crds.yaml
```

## 6. Update RBAC Permissions

Use the following command to update your RBAC permissions:

```
cat<<EOF | kubectl apply -f -
---
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
EOF
```

## 7. Apply the `aes` Configuration

Finally, apply the remaining `aes` configurations using the following command: 

```
cat<<EOF | kubectl apply -n default -f -
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

## 8. What's Next?

Now that you have the Ambassador Edge Stack up and running, check out the [Getting Started](/user-guide/getting-started) guide for recommendations on what to do next and take full advantage of its features.
