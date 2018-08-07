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
  - protocol: TCP
    port: 80
    targetPort: 8080
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
    image: rschloming/backend:3
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
  name: ambassador-{self.name.k8s}
spec:
  type: NodePort
  ports:
   - port: 80
  selector:
    service: ambassador-{self.name.k8s}
---
apiVersion: v1
kind: Service
metadata:
  labels:
    service: ambassador-{self.name.k8s}-admin
  name: ambassador-{self.name.k8s}-admin
spec:
  type: NodePort
  ports:
  - name: ambassador-{self.name.k8s}-admin
    port: 8877
    targetPort: 8877
  selector:
    service: ambassador-{self.name.k8s}
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: ambassador-{self.name.k8s}
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
  name: ambassador-{self.name.k8s}
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: ambassador-{self.name.k8s}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ambassador-{self.name.k8s}
subjects:
- kind: ServiceAccount
  name: ambassador-{self.name.k8s}
  namespace: default
---
apiVersion: v1
kind: Pod
metadata:
  name: ambassador-{self.name.k8s}
  annotations:
    sidecar.istio.io/inject: "false"
  labels:
    service: ambassador-{self.name.k8s}
spec:
  serviceAccountName: ambassador-{self.name.k8s}
  containers:
  - name: ambassador
    image: quay.io/datawire/ambassador:0.35.3
#    resources:
#      limits:
#        cpu: 1
#        memory: 400Mi
#      requests:
#        cpu: 200m
#        memory: 100Mi
    env:
    - name: AMBASSADOR_NAMESPACE
      valueFrom:
        fieldRef:
          fieldPath: metadata.namespace
    - name: AMBASSADOR_ID
      value: {self.name.k8s}
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
  - name: statsd
    image: quay.io/datawire/statsd:0.35.3
  restartPolicy: Always
"""
