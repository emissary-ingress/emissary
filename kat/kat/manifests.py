BACKEND_SERVICE = """
---
kind: Service
apiVersion: v1
metadata:
  name: {self.path.k8s}
spec:
  selector:
    backend: {self.path.k8s}
  ports:
  - name: http
    protocol: TCP
    port: 80
    targetPort: 8080
  - name: https
    protocol: TCP
    port: 443
    targetPort: 8443
"""

BACKEND = BACKEND_SERVICE + """
---
apiVersion: v1
kind: Pod
metadata:
  name: {self.path.k8s}
  labels:
    backend: {self.path.k8s}
spec:
  containers:
  - name: backend
    image: quay.io/datawire/kat-backend:11
    ports:
    - containerPort: 8080
    env:
    - name: BACKEND
      value: {self.path.k8s}
"""

SUPERPOD_POD = """
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: superpod
spec:
  replicas: 1
  selector:
    matchLabels:
      backend: superpod
  template:
    metadata:
      labels:
        backend: superpod
    spec:
      containers:
      - name: backend
        image: quay.io/datawire/kat-backend:11
        # ports:
        # {ports}
        env:
        - name: INCLUDE_EXTAUTH_HEADER
          value: "yes"
        # {envs} 
"""

AUTH_BACKEND = """
---
kind: Service
apiVersion: v1
metadata:
  name: {self.path.k8s}
spec:
  selector:
    backend: {self.path.k8s}
  ports:
  - name: http
    protocol: TCP
    port: 80
    targetPort: 8080
  - name: https
    protocol: TCP
    port: 443
    targetPort: 8443
---
apiVersion: v1
kind: Pod
metadata:
  name: {self.path.k8s}
  labels:
    backend: {self.path.k8s}
spec:
  containers:
  - name: backend
    image: quay.io/datawire/kat-backend:11
    ports:
    - containerPort: 8080
    env:
    - name: BACKEND
      value: {self.path.k8s}
    - name: INCLUDE_EXTAUTH_HEADER
      value: "yes" 
"""

GRPC_AUTH_BACKEND = """
---
kind: Service
apiVersion: v1
metadata:
  name: {self.path.k8s}
spec:
  selector:
    backend: {self.path.k8s}
  ports:
  - name: http
    protocol: TCP
    port: 80
    targetPort: 8080
  - name: https
    protocol: TCP
    port: 443
    targetPort: 8443
--- 
apiVersion: v1
kind: Pod
metadata:
  name: {self.path.k8s}
  labels:
    backend: {self.path.k8s}
spec:
  containers:
  - name: backend
    image: quay.io/datawire/kat-backend:11
    ports:
    - containerPort: 8080
    env:
    - name: BACKEND
      value: {self.path.k8s}
    - name: KAT_BACKEND_TYPE
      value: grpc_auth
"""

GRPC_ECHO_BACKEND = """
---
kind: Service
apiVersion: v1
metadata:
  name: {self.path.k8s}
spec:
  selector:
    backend: {self.path.k8s}
  ports:
  - name: http
    protocol: TCP
    port: 80
    targetPort: 8080
  - name: https
    protocol: TCP
    port: 443
    targetPort: 8443
--- 
apiVersion: v1
kind: Pod
metadata:
  name: {self.path.k8s}
  labels:
    backend: {self.path.k8s}
spec:
  containers:
  - name: backend
    image: quay.io/datawire/kat-backend:11
    ports:
    - containerPort: 8080
    env:
    - name: BACKEND
      value: {self.path.k8s}
    - name: KAT_BACKEND_TYPE
      value: grpc_echo
"""

RBAC_CLUSTER_SCOPE = """
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: {self.path.k8s}
rules:
- apiGroups: [""]
  resources:
  - configmaps
  - services
  - secrets
  - namespaces
  - endpoints
  verbs: ["get", "list", "watch"]
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {self.path.k8s}
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: {self.path.k8s}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: {self.path.k8s}
subjects:
- kind: ServiceAccount
  name: {self.path.k8s}
  namespace: {self.namespace}
"""

# The actual namespace attribute will be added by the KAT harness.
RBAC_NAMESPACE_SCOPE = """
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: Role
metadata:
  name: {self.path.k8s}
rules:
- apiGroups: [""]
  resources:
  - configmaps
  - services
  - secrets
  - namespaces
  - endpoints
  verbs: ["get", "list", "watch"]
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {self.path.k8s}
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: RoleBinding
metadata:
  name: {self.path.k8s}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: {self.path.k8s}
subjects:
- kind: ServiceAccount
  name: {self.path.k8s}
  namespace: {self.namespace}
"""

AMBASSADOR = """
---
apiVersion: v1
kind: Service
metadata:
  name: {self.path.k8s}
spec:
  type: NodePort
  ports:
  - name: http
    protocol: TCP
    port: 80
    targetPort: 8080
  - name: https
    protocol: TCP
    port: 443
    targetPort: 8443
  {extra_ports}
  selector:
    service: {self.path.k8s}
---
apiVersion: v1
kind: Service
metadata:
  labels:
    service: {self.path.k8s}-admin
  name: {self.path.k8s}-admin
spec:
  type: NodePort
  ports:
  - name: {self.path.k8s}-admin
    port: 8877
    targetPort: 8877
  selector:
    service: {self.path.k8s}
---
apiVersion: v1
kind: Pod
metadata:
  name: {self.path.k8s}
  annotations:
    sidecar.istio.io/inject: "false"
  labels:
    service: {self.path.k8s}
spec:
  serviceAccountName: {self.path.k8s}
  containers:
  - name: ambassador
    image: {image}
    env:
    {envs}
    - name: AMBASSADOR_NAMESPACE
      valueFrom:
        fieldRef:
          fieldPath: metadata.namespace
    - name: AMBASSADOR_ID
      value: {self.path.k8s}
    - name: AMBASSADOR_SNAPSHOT_COUNT
      value: "0"
    livenessProbe:
      httpGet:
        path: /ambassador/v0/check_alive
        port: 8877
      initialDelaySeconds: 120
      periodSeconds: 3
    readinessProbe:
      httpGet:
        path: /ambassador/v0/check_ready
        port: 8877
      initialDelaySeconds: 120
      periodSeconds: 3
  restartPolicy: Always
"""

HTTPBIN = """
---
kind: Service
apiVersion: v1
metadata:
  name: {self.path.k8s}
spec:
  selector:
    pod: {self.path.k8s}
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80
---
apiVersion: v1
kind: Pod
metadata:
  name: {self.path.k8s}
  labels:
    pod: {self.path.k8s}
spec:
  containers:
  - name: backend
    image: kennethreitz/httpbin
    ports:
    - containerPort: 80
"""
