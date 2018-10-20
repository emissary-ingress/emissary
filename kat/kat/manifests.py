BACKEND = """
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
    image: quay.io/datawire/kat-backend:5
    ports:
    - containerPort: 8080
    env:
    - name: BACKEND
      value: {self.path.k8s}
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
    targetPort: 80
  - name: https
    protocol: TCP
    port: 443
    targetPort: 443
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
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: {self.path.k8s}
rules:
- apiGroups: [""]
  resources:
  - services
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources:
  - configmaps
  verbs: ["create", "update", "patch", "get", "list", "watch"]
- apiGroups: [""]
  resources:
  - secrets
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
  namespace: default
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
    - name: AMBASSADOR_NAMESPACE
      valueFrom:
        fieldRef:
          fieldPath: metadata.namespace
    - name: AMBASSADOR_ID
      value: {self.path.k8s}
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
