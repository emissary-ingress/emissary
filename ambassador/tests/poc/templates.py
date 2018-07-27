BACKEND = """
---
kind: Service
apiVersion: v1
metadata:
  name: %(name)s
spec:
  selector:
    deployment: %(name)s
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %(name)s
  labels:
    deployment: %(name)s
spec:
  replicas: 1
  selector:
    matchLabels:
      deployment: %(name)s
  template:
    metadata:
      labels:
        deployment: %(name)s
    spec:
      containers:
      - name: backend
        image: rschloming/backend
        ports:
        - containerPort: 8080
"""

AMBASSADOR = """
---
apiVersion: v1
kind: Service
metadata:
  name: ambassador-%(name)s
spec:
  type: NodePort
  ports:
   - port: 80
  selector:
    service: ambassador-%(name)s
---
apiVersion: v1
kind: Service
metadata:
  labels:
    service: ambassador-%(name)s-admin
  name: ambassador-%(name)s-admin
spec:
  type: NodePort
  ports:
  - name: ambassador-%(name)s-admin
    port: 8877
    targetPort: 8877
  selector:
    service: ambassador-%(name)s
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: ambassador-%(name)s
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
  name: ambassador-%(name)s
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: ambassador-%(name)s
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ambassador-%(name)s
subjects:
- kind: ServiceAccount
  name: ambassador-%(name)s
  namespace: default
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: ambassador-%(name)s
spec:
  replicas: 1
  template:
    metadata:
      annotations:
        sidecar.istio.io/inject: "false"
      labels:
        service: ambassador-%(name)s
    spec:
      serviceAccountName: ambassador-%(name)s
      containers:
      - name: ambassador
        image: quay.io/datawire/ambassador:0.35.3
#        resources:
#          limits:
#            cpu: 1
#            memory: 400Mi
#          requests:
#            cpu: 200m
#            memory: 100Mi
        env:
        - name: AMBASSADOR_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: AMBASSADOR_ID
          value: %(name)s
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
