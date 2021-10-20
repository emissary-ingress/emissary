changequote(`«', `»')
---
apiVersion: v1
kind: Service
metadata:
  labels:
    service: ambassador-admin
  name: ambassador-admin
spec:
  type: NodePort
  ports:
  - name: ambassador-admin
    port: 8877
    targetPort: 8877
  selector:
    service: ambassador
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ambassador
rules:
- apiGroups: [""]
  resources: [ "endpoints", "namespaces", "secrets", "services" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "getambassador.io" ]
  resources: [ "*" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "getambassador.io" ]
  resources: [ "mappings/status" ]
  verbs: ["update"]
- apiGroups: [ "apiextensions.k8s.io" ]
  resources: [ "customresourcedefinitions" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "networking.internal.knative.dev" ]
  resources: [ "clusteringresses", "ingresses" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "networking.internal.knative.dev" ]
  resources: [ "ingresses/status", "clusteringresses/status" ]
  verbs: ["update"]
- apiGroups: [ "extensions", "networking.k8s.io" ]
  resources: [ "ingresses", "ingressclasses" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "extensions", "networking.k8s.io"]
  resources: [ "ingresses/status" ]
  verbs: ["update"]
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ambassador
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ambassador
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ambassador
subjects:
- kind: ServiceAccount
  name: ambassador
  namespace: default
include(../../../manifests/emissary/ambassador-crds.yaml)
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: ambassador-statsd-config
data:
  exporterConfiguration: ''
---
apiVersion: apps/v1
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
      volumes:
      - name: stats-exporter-mapping-config
        configMap:
          name: ambassador-statsd-config
          items:
          - key: exporterConfiguration
            path: mapping-config.yaml
      containers:
      - name: ambassador
        image: docker.io/datawire/ambassador:$version$
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
        - name: HOST_IP
          valueFrom:
            fieldRef:
              fieldPath: status.hostIP
        - name: STATSD_ENABLED
          value: "true"
        - name: STATSD_HOST
          value: "localhost"
        ports:
        - name: http
          containerPort: 8080
        - name: https
          containerPort: 8443
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
        securityContext:
          allowPrivilegeEscalation: false
      - name: statsd-sink
        image: prom/statsd-exporter:v0.8.1
        ports:
        - name: metrics
          containerPort: 9102
        - name: listener
          containerPort: 8125
        args: ["--statsd.listen-udp=:8125", "--statsd.mapping-config=/statsd-exporter/mapping-config.yaml"]
        volumeMounts:
        - name: stats-exporter-mapping-config
          mountPath: /statsd-exporter/
          readOnly: true
      restartPolicy: Always
      securityContext:
        runAsUser: 8888
