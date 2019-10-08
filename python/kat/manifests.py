KAT_CLIENT_POD = """
---
apiVersion: v1
kind: Pod
metadata:
  name: kat
  labels:
    backend: kat
spec:
  containers:
  - name: backend
    image: {environ[KAT_CLIENT_DOCKER_IMAGE]}
    imagePullPolicy: {environ[KAT_IMAGE_PULL_POLICY]}
"""

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
    image: {environ[KAT_SERVER_DOCKER_IMAGE]}
    imagePullPolicy: {environ[KAT_IMAGE_PULL_POLICY]}
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
        image: {environ[KAT_SERVER_DOCKER_IMAGE]}
        imagePullPolicy: {environ[KAT_IMAGE_PULL_POLICY]}
        # ports:
        # (ports)
        env:
        - name: INCLUDE_EXTAUTH_HEADER
          value: "yes"
        # (envs)
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
    image: {environ[KAT_SERVER_DOCKER_IMAGE]}
    imagePullPolicy: {environ[KAT_IMAGE_PULL_POLICY]}
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
    image: {environ[KAT_SERVER_DOCKER_IMAGE]}
    imagePullPolicy: {environ[KAT_IMAGE_PULL_POLICY]}
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
    image: {environ[KAT_SERVER_DOCKER_IMAGE]}
    imagePullPolicy: {environ[KAT_IMAGE_PULL_POLICY]}
    ports:
    - containerPort: 8080
    env:
    - name: BACKEND
      value: {self.path.k8s}
    - name: KAT_BACKEND_TYPE
      value: grpc_echo
"""

CRDS = """
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: authservices.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: authservices
    singular: authservice
    kind: AuthService
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: consulresolvers.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: consulresolvers
    singular: consulresolver
    kind: ConsulResolver
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: kubernetesendpointresolvers.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: kubernetesendpointresolvers
    singular: kubernetesendpointresolver
    kind: KubernetesEndpointResolver
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: kubernetesserviceresolvers.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: kubernetesserviceresolvers
    singular: kubernetesserviceresolver
    kind: KubernetesServiceResolver
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: mappings.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: mappings
    singular: mapping
    kind: Mapping
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: modules.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: modules
    singular: module
    kind: Module
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: ratelimitservices.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: ratelimitservices
    singular: ratelimitservice
    kind: RateLimitService
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: tcpmappings.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: tcpmappings
    singular: tcpmapping
    kind: TCPMapping
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: tlscontexts.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: tlscontexts
    singular: tlscontext
    kind: TLSContext
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: tracingservices.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: tracingservices
    singular: tracingservice
    kind: TracingService
"""

RBAC_CLUSTER_SCOPE = """
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: {self.path.k8s}
rules:
- apiGroups: [""]
  resources: [ "services", "secrets", "namespaces", "endpoints" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "apiextensions.k8s.io" ]
  resources: [ "customresourcedefinitions" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "getambassador.io" ]
  resources: [ "*" ]
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
  resources: [ "services", "secrets", "namespaces", "endpoints" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "apiextensions.k8s.io" ]
  resources: [ "customresourcedefinitions" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "getambassador.io" ]
  resources: [ "*" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "extensions" ]
  resources: [ "ingresses" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "extensions" ]
  resources: [ "ingresses/status" ]
  verbs: ["update"]
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
  labels:
    app.kubernetes.io/component: ambassador-service
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
  securityContext:
    runAsUser: 8888
  serviceAccountName: {self.path.k8s}
  restartPolicy: Always
  volumes:
    - name: scratchpad
      emptyDir:
        medium: Memory
        sizeLimit: "45Mi"
    - name: ambassador-pod-info
      downwardAPI:
        items:
        - path: "labels"
          fieldRef:
            fieldPath: metadata.labels
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
    - name: AMBASSADOR_CONFIG_BASE_DIR
      value: "/tmp/ambassador"
    securityContext:
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true  
      {capabilities_block}
    livenessProbe:
      httpGet:
        path: /ambassador/v0/check_alive
        port: 8877
      initialDelaySeconds: 30
      periodSeconds: 10
      failureThreshold: 30
    readinessProbe:
      httpGet:
        path: /ambassador/v0/check_ready
        port: 8877
      initialDelaySeconds: 30
      periodSeconds: 10
      failureThreshold: 30
    volumeMounts:
      - mountPath: /tmp/
        name: scratchpad
      - name: ambassador-pod-info
        mountPath: /tmp/ambassador-pod-info
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

KNATIVE_SERVING_CRDS = """
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
    serving.knative.dev/release: "v0.7.1"
  name: certificates.networking.internal.knative.dev
spec:
  additionalPrinterColumns:
  - JSONPath: .status.conditions[?(@.type=="Ready")].status
    name: Ready
    type: string
  - JSONPath: .status.conditions[?(@.type=="Ready")].reason
    name: Reason
    type: string
  group: networking.internal.knative.dev
  names:
    categories:
    - all
    - knative-internal
    - networking
    kind: Certificate
    plural: certificates
    shortNames:
    - kcert
    singular: certificate
  scope: Namespaced
  subresources:
    status: {}
  version: v1alpha1

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
    serving.knative.dev/release: "v0.7.1"
  name: clusteringresses.networking.internal.knative.dev
spec:
  additionalPrinterColumns:
  - JSONPath: .status.conditions[?(@.type=='Ready')].status
    name: Ready
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].reason
    name: Reason
    type: string
  group: networking.internal.knative.dev
  names:
    categories:
    - all
    - knative-internal
    - networking
    kind: ClusterIngress
    plural: clusteringresses
    singular: clusteringress
  scope: Cluster
  subresources:
    status: {}
  version: v1alpha1

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
  name: images.caching.internal.knative.dev
spec:
  group: caching.internal.knative.dev
  names:
    categories:
    - all
    - knative-internal
    - caching
    kind: Image
    plural: images
    shortNames:
    - img
    singular: image
  scope: Namespaced
  subresources:
    status: {}
  version: v1alpha1

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
    serving.knative.dev/release: "v0.7.1"
  name: ingresses.networking.internal.knative.dev
spec:
  additionalPrinterColumns:
  - JSONPath: .status.conditions[?(@.type=='Ready')].status
    name: Ready
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].reason
    name: Reason
    type: string
  group: networking.internal.knative.dev
  names:
    categories:
    - all
    - knative-internal
    - networking
    kind: Ingress
    plural: ingresses
    shortNames:
    - ing
    singular: ingress
  scope: Namespaced
  subresources:
    status: {}
  version: v1alpha1

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
    serving.knative.dev/release: "v0.7.1"
  name: podautoscalers.autoscaling.internal.knative.dev
spec:
  additionalPrinterColumns:
  - JSONPath: .status.conditions[?(@.type=='Ready')].status
    name: Ready
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].reason
    name: Reason
    type: string
  group: autoscaling.internal.knative.dev
  names:
    categories:
    - all
    - knative-internal
    - autoscaling
    kind: PodAutoscaler
    plural: podautoscalers
    shortNames:
    - kpa
    singular: podautoscaler
  scope: Namespaced
  subresources:
    status: {}
  version: v1alpha1

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
    serving.knative.dev/release: "v0.7.1"
  name: serverlessservices.networking.internal.knative.dev
spec:
  additionalPrinterColumns:
  - JSONPath: .spec.mode
    name: Mode
    type: string
  - JSONPath: .status.serviceName
    name: ServiceName
    type: string
  - JSONPath: .status.privateServiceName
    name: PrivateServiceName
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].status
    name: Ready
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].reason
    name: Reason
    type: string
  group: networking.internal.knative.dev
  names:
    categories:
    - all
    - knative-internal
    - networking
    kind: ServerlessService
    plural: serverlessservices
    shortNames:
    - sks
    singular: serverlessservice
  scope: Namespaced
  subresources:
    status: {}
  version: v1alpha1
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
    serving.knative.dev/release: "v0.7.1"
  name: configurations.serving.knative.dev
spec:
  additionalPrinterColumns:
  - JSONPath: .status.latestCreatedRevisionName
    name: LatestCreated
    type: string
  - JSONPath: .status.latestReadyRevisionName
    name: LatestReady
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].status
    name: Ready
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].reason
    name: Reason
    type: string
  group: serving.knative.dev
  names:
    categories:
    - all
    - knative
    - serving
    kind: Configuration
    plural: configurations
    shortNames:
    - config
    - cfg
    singular: configuration
  scope: Namespaced
  subresources:
    status: {}
  versions:
  - name: v1alpha1
    served: true
    storage: true

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
    serving.knative.dev/release: "v0.7.1"
  name: revisions.serving.knative.dev
spec:
  additionalPrinterColumns:
  - JSONPath: .status.serviceName
    name: Service Name
    type: string
  - JSONPath: .metadata.labels['serving\\.knative\\.dev/configurationGeneration']
    name: Generation
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].status
    name: Ready
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].reason
    name: Reason
    type: string
  group: serving.knative.dev
  names:
    categories:
    - all
    - knative
    - serving
    kind: Revision
    plural: revisions
    shortNames:
    - rev
    singular: revision
  scope: Namespaced
  subresources:
    status: {}
  versions:
  - name: v1alpha1
    served: true
    storage: true

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
    serving.knative.dev/release: "v0.7.1"
  name: routes.serving.knative.dev
spec:
  additionalPrinterColumns:
  - JSONPath: .status.url
    name: URL
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].status
    name: Ready
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].reason
    name: Reason
    type: string
  group: serving.knative.dev
  names:
    categories:
    - all
    - knative
    - serving
    kind: Route
    plural: routes
    shortNames:
    - rt
    singular: route
  scope: Namespaced
  subresources:
    status: {}
  versions:
  - name: v1alpha1
    served: true
    storage: true

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
    serving.knative.dev/release: "v0.7.1"
  name: services.serving.knative.dev
spec:
  additionalPrinterColumns:
  - JSONPath: .status.url
    name: URL
    type: string
  - JSONPath: .status.latestCreatedRevisionName
    name: LatestCreated
    type: string
  - JSONPath: .status.latestReadyRevisionName
    name: LatestReady
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].status
    name: Ready
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].reason
    name: Reason
    type: string
  group: serving.knative.dev
  names:
    categories:
    - all
    - knative
    - serving
    kind: Service
    plural: services
    shortNames:
    - kservice
    - ksvc
    singular: service
  scope: Namespaced
  subresources:
    status: {}
  versions:
  - name: v1alpha1
    served: true
    storage: true
"""

KNATIVE_SERVING_071 = """
---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    serving.knative.dev/release: "v0.7.1"
  name: knative-serving

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    networking.knative.dev/certificate-provider: cert-manager
    serving.knative.dev/controller: "true"
    serving.knative.dev/release: "v0.7.1"
  name: knative-serving-certmanager
rules:
- apiGroups:
  - certmanager.k8s.io
  resources:
  - certificates
  verbs:
  - get
  - list
  - create
  - update
  - delete
  - patch
  - watch

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    autoscaling.knative.dev/metric-provider: custom-metrics
    serving.knative.dev/release: "v0.7.1"
  name: custom-metrics-server-resources
rules:
- apiGroups:
  - custom.metrics.k8s.io
  resources:
  - '*'
  verbs:
  - '*'

---
aggregationRule:
  clusterRoleSelectors:
  - matchLabels:
      serving.knative.dev/controller: "true"
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    serving.knative.dev/release: "v0.7.1"
  name: knative-serving-admin
rules: []
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    serving.knative.dev/controller: "true"
    serving.knative.dev/release: "v0.7.1"
  name: knative-serving-core
rules:
- apiGroups:
  - ""
  resources:
  - pods
  - namespaces
  - secrets
  - configmaps
  - endpoints
  - services
  - events
  - serviceaccounts
  verbs:
  - get
  - list
  - create
  - update
  - delete
  - patch
  - watch
- apiGroups:
  - ""
  resources:
  - endpoints/restricted
  verbs:
  - create
- apiGroups:
  - apps
  resources:
  - deployments
  - deployments/finalizers
  verbs:
  - get
  - list
  - create
  - update
  - delete
  - patch
  - watch
- apiGroups:
  - admissionregistration.k8s.io
  resources:
  - mutatingwebhookconfigurations
  verbs:
  - get
  - list
  - create
  - update
  - delete
  - patch
  - watch
- apiGroups:
  - apiextensions.k8s.io
  resources:
  - customresourcedefinitions
  verbs:
  - get
  - list
  - create
  - update
  - delete
  - patch
  - watch
- apiGroups:
  - autoscaling
  resources:
  - horizontalpodautoscalers
  verbs:
  - get
  - list
  - create
  - update
  - delete
  - patch
  - watch
- apiGroups:
  - serving.knative.dev
  - autoscaling.internal.knative.dev
  - networking.internal.knative.dev
  resources:
  - '*'
  - '*/status'
  - '*/finalizers'
  verbs:
  - get
  - list
  - create
  - update
  - delete
  - deletecollection
  - patch
  - watch
- apiGroups:
  - caching.internal.knative.dev
  resources:
  - images
  verbs:
  - get
  - list
  - create
  - update
  - delete
  - patch
  - watch

---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    serving.knative.dev/release: "v0.7.1"
  name: controller
  namespace: knative-serving

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    autoscaling.knative.dev/metric-provider: custom-metrics
    serving.knative.dev/release: "v0.7.1"
  name: custom-metrics:system:auth-delegator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:auth-delegator
subjects:
- kind: ServiceAccount
  name: controller
  namespace: knative-serving

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    autoscaling.knative.dev/metric-provider: custom-metrics
    serving.knative.dev/release: "v0.7.1"
  name: hpa-controller-custom-metrics
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: custom-metrics-server-resources
subjects:
- kind: ServiceAccount
  name: horizontal-pod-autoscaler
  namespace: kube-system

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    serving.knative.dev/release: "v0.7.1"
  name: knative-serving-controller-admin
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: knative-serving-admin
subjects:
- kind: ServiceAccount
  name: controller
  namespace: knative-serving

---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  labels:
    autoscaling.knative.dev/metric-provider: custom-metrics
    serving.knative.dev/release: "v0.7.1"
  name: custom-metrics-auth-reader
  namespace: kube-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: extension-apiserver-authentication-reader
subjects:
- kind: ServiceAccount
  name: controller
  namespace: knative-serving

---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: activator
    serving.knative.dev/release: "v0.7.1"
  name: activator-service
  namespace: knative-serving
spec:
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: 8012
  - name: http2
    port: 81
    protocol: TCP
    targetPort: 8013
  - name: metrics
    port: 9090
    protocol: TCP
    targetPort: 9090
  selector:
    app: activator
  type: ClusterIP

---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: controller
    serving.knative.dev/release: "v0.7.1"
  name: controller
  namespace: knative-serving
spec:
  ports:
  - name: metrics
    port: 9090
    protocol: TCP
    targetPort: 9090
  selector:
    app: controller

---
apiVersion: v1
kind: Service
metadata:
  labels:
    role: webhook
    serving.knative.dev/release: "v0.7.1"
  name: webhook
  namespace: knative-serving
spec:
  ports:
  - port: 443
    targetPort: 8443
  selector:
    role: webhook

---
apiVersion: caching.internal.knative.dev/v1alpha1
kind: Image
metadata:
  labels:
    serving.knative.dev/release: "v0.7.1"
  name: queue-proxy
  namespace: knative-serving
spec:
  image: gcr.io/knative-releases/github.com/knative/serving/cmd/queue@sha256:89fb5a1d2d9c0abd10ce3135c02f9e9ffbf93087a3ece7481615a0f9d9209713

---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    serving.knative.dev/release: "v0.7.1"
  name: activator
  namespace: knative-serving
spec:
  selector:
    matchLabels:
      app: activator
      role: activator
  template:
    metadata:
      annotations:
        cluster-autoscaler.kubernetes.io/safe-to-evict: "false"
      labels:
        app: activator
        role: activator
        serving.knative.dev/release: "v0.7.1"
    spec:
      containers:
      - args:
        - -logtostderr=false
        - -stderrthreshold=FATAL
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: SYSTEM_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: CONFIG_LOGGING_NAME
          value: config-logging
        - name: CONFIG_OBSERVABILITY_NAME
          value: config-observability
        - name: METRICS_DOMAIN
          value: knative.dev/serving
        image: gcr.io/knative-releases/github.com/knative/serving/cmd/activator@sha256:864c0dc5e8d8eeee6162f448ae6452ab53f53642536a4720d59b6bc2402df01f
        livenessProbe:
          httpGet:
            httpHeaders:
            - name: k-kubelet-probe
              value: activator
            path: /healthz
            port: 8012
        name: activator
        ports:
        - containerPort: 8012
          name: http1-port
        - containerPort: 8013
          name: h2c-port
        - containerPort: 9090
          name: metrics-port
        readinessProbe:
          httpGet:
            httpHeaders:
            - name: k-kubelet-probe
              value: activator
            path: /healthz
            port: 8012
        resources:
          limits:
            cpu: 200m
            memory: 600Mi
          requests:
            cpu: 20m
            memory: 60Mi
        securityContext:
          allowPrivilegeEscalation: false
        volumeMounts:
        - mountPath: /etc/config-logging
          name: config-logging
        - mountPath: /etc/config-observability
          name: config-observability
      serviceAccountName: controller
      volumes:
      - configMap:
          name: config-logging
        name: config-logging
      - configMap:
          name: config-observability
        name: config-observability

---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: autoscaler
    serving.knative.dev/release: "v0.7.1"
  name: autoscaler
  namespace: knative-serving
spec:
  ports:
  - name: http
    port: 8080
    protocol: TCP
    targetPort: 8080
  - name: metrics
    port: 9090
    protocol: TCP
    targetPort: 9090
  - name: custom-metrics
    port: 443
    protocol: TCP
    targetPort: 8443
  selector:
    app: autoscaler

---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    serving.knative.dev/release: "v0.7.1"
  name: autoscaler
  namespace: knative-serving
spec:
  replicas: 1
  selector:
    matchLabels:
      app: autoscaler
  template:
    metadata:
      annotations:
        cluster-autoscaler.kubernetes.io/safe-to-evict: "false"
      labels:
        app: autoscaler
        serving.knative.dev/release: "v0.7.1"
    spec:
      containers:
      - args:
        - --secure-port=8443
        - --cert-dir=/tmp
        env:
        - name: SYSTEM_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: CONFIG_LOGGING_NAME
          value: config-logging
        - name: CONFIG_OBSERVABILITY_NAME
          value: config-observability
        - name: METRICS_DOMAIN
          value: knative.dev/serving
        image: gcr.io/knative-releases/github.com/knative/serving/cmd/autoscaler@sha256:026860790fe07bf3dcd42fe2c0a21c7c15ef59f4cb772b6e369f927620f6c0ec
        livenessProbe:
          httpGet:
            httpHeaders:
            - name: k-kubelet-probe
              value: autoscaler
            path: /healthz
            port: 8080
        name: autoscaler
        ports:
        - containerPort: 8080
          name: websocket
        - containerPort: 9090
          name: metrics
        - containerPort: 8443
          name: custom-metrics
        readinessProbe:
          httpGet:
            httpHeaders:
            - name: k-kubelet-probe
              value: autoscaler
            path: /healthz
            port: 8080
        resources:
          limits:
            cpu: 300m
            memory: 400Mi
          requests:
            cpu: 30m
            memory: 40Mi
        securityContext:
          allowPrivilegeEscalation: false
        volumeMounts:
        - mountPath: /etc/config-autoscaler
          name: config-autoscaler
        - mountPath: /etc/config-logging
          name: config-logging
        - mountPath: /etc/config-observability
          name: config-observability
      serviceAccountName: controller
      volumes:
      - configMap:
          name: config-autoscaler
        name: config-autoscaler
      - configMap:
          name: config-logging
        name: config-logging
      - configMap:
          name: config-observability
        name: config-observability

---
apiVersion: v1
data:
  _example: |
    ################################
    #                              #
    #    EXAMPLE CONFIGURATION     #
    #                              #
    ################################

    # This block is not actually functional configuration,
    # but serves to illustrate the available configuration
    # options and document them in a way that is accessible
    # to users that `kubectl edit` this config map.
    #
    # These sample configuration options may be copied out of
    # this example block and unindented to be in the data block
    # to actually change the configuration.

    # The Revision ContainerConcurrency field specifies the maximum number
    # of requests the Container can handle at once. Container concurrency
    # target percentage is how much of that maximum to use in a stable
    # state. E.g. if a Revision specifies ContainerConcurrency of 10, then
    # the Autoscaler will try to maintain 7 concurrent connections per pod
    # on average. A value of 70 is chosen because the Autoscaler panics
    # when concurrency exceeds 2x the desired set point. So we will panic
    # before we reach the limit.
    # For legacy and backwards compatibility reasons, this value also accepts
    # fractional values in (0, 1] interval (i.e. 0.7 ⇒ 70%).
    # Thus minimal percentage value must be greater than 1.0, or it will be
    # treated as a fraction.
    # TODO(#2016): Set to 70%.
    container-concurrency-target-percentage: "100"

    # The container concurrency target default is what the Autoscaler will
    # try to maintain when the Revision specifies unlimited concurrency.
    # Even when specifying unlimited concurrency, the autoscaler will
    # horizontally scale the application based on this target concurrency.
    #
    # A value of 100 is chosen because it's enough to allow vertical pod
    # autoscaling to tune resource requests. E.g. maintaining 1 concurrent
    # "hello world" request doesn't consume enough resources to allow VPA
    # to achieve efficient resource usage (VPA CPU minimum is 300m).
    container-concurrency-target-default: "100"

    # When operating in a stable mode, the autoscaler operates on the
    # average concurrency over the stable window.
    stable-window: "60s"

    # When observed average concurrency during the panic window reaches
    # panic-threshold-percentage the target concurrency, the autoscaler
    # enters panic mode. When operating in panic mode, the autoscaler
    # scales on the average concurrency over the panic window which is
    # panic-window-percentage of the stable-window.
    panic-window-percentage: "10.0"

    # Absolute panic window duration.
    # Deprecated in favor of panic-window-percentage.
    # Existing revisions will continue to scale based on panic-window
    # but new revisions will default to panic-window-percentage.
    panic-window: "6s"

    # The percentage of the container concurrency target at which to
    # enter panic mode when reached within the panic window.
    panic-threshold-percentage: "200.0"

    # Max scale up rate limits the rate at which the autoscaler will
    # increase pod count. It is the maximum ratio of desired pods versus
    # observed pods.
    max-scale-up-rate: "10"

    # Scale to zero feature flag
    enable-scale-to-zero: "true"

    # Tick interval is the time between autoscaling calculations.
    tick-interval: "2s"

    # Dynamic parameters (take effect when config map is updated):

    # Scale to zero grace period is the time an inactive revision is left
    # running before it is scaled to zero (min: 30s).
    scale-to-zero-grace-period: "30s"
kind: ConfigMap
metadata:
  labels:
    serving.knative.dev/release: "v0.7.1"
  name: config-autoscaler
  namespace: knative-serving

---
apiVersion: v1
data:
  _example: |
    ################################
    #                              #
    #    EXAMPLE CONFIGURATION     #
    #                              #
    ################################

    # This block is not actually functional configuration,
    # but serves to illustrate the available configuration
    # options and document them in a way that is accessible
    # to users that `kubectl edit` this config map.
    #
    # These sample configuration options may be copied out of
    # this block and unindented to actually change the configuration.

    # IssuerRef is a reference to the issuer for this certificate.
    # IssuerRef should be either `ClusterIssuer` or `Issuer`.
    # Please refer `IssuerRef` in https://github.com/jetstack/cert-manager/blob/master/pkg/apis/certmanager/v1alpha1/types_certificate.go
    # for more details about IssuerRef configuration.
    issuerRef: |
      kind: ClusterIssuer
      name: letsencrypt-issuer

    # solverConfig defines the configuration for the ACME certificate provider.
    # The solverConfig should be either dns01 or http01.
    # Please refer `SolverConfig` in https://github.com/jetstack/cert-manager/blob/master/pkg/apis/certmanager/v1alpha1/types_certificate.go
    # for more details about ACME configuration.
    solverConfig: |
      dns01:
        provider: cloud-dns-provider
kind: ConfigMap
metadata:
  labels:
    networking.knative.dev/certificate-provider: cert-manager
    serving.knative.dev/release: "v0.7.1"
  name: config-certmanager
  namespace: knative-serving

---
apiVersion: v1
data:
  _example: |
    ################################
    #                              #
    #    EXAMPLE CONFIGURATION     #
    #                              #
    ################################

    # This block is not actually functional configuration,
    # but serves to illustrate the available configuration
    # options and document them in a way that is accessible
    # to users that `kubectl edit` this config map.
    #
    # These sample configuration options may be copied out of
    # this example block and unindented to be in the data block
    # to actually change the configuration.

    # revision-timeout-seconds contains the default number of
    # seconds to use for the revision's per-request timeout, if
    # none is specified.
    revision-timeout-seconds: "300"  # 5 minutes

    # max-revision-timeout-seconds contains the maximum number of
    # seconds that can be used for revision-timeout-seconds.
    # This value must be greater than or equal to revision-timeout-seconds.
    # If omitted, the system default is used (600 seconds).
    max-revision-timeout-seconds: "600"  # 10 minutes

    # revision-cpu-request contains the cpu allocation to assign
    # to revisions by default.  If omitted, no value is specified
    # and the system default is used.
    revision-cpu-request: "400m"  # 0.4 of a CPU (aka 400 milli-CPU)

    # revision-memory-request contains the memory allocation to assign
    # to revisions by default.  If omitted, no value is specified
    # and the system default is used.
    revision-memory-request: "100M"  # 100 megabytes of memory

    # revision-cpu-limit contains the cpu allocation to limit
    # revisions to by default.  If omitted, no value is specified
    # and the system default is used.
    revision-cpu-limit: "1000m"  # 1 CPU (aka 1000 milli-CPU)

    # revision-memory-limit contains the memory allocation to limit
    # revisions to by default.  If omitted, no value is specified
    # and the system default is used.
    revision-memory-limit: "200M"  # 200 megabytes of memory

    # container-name-template contains a template for the default
    # container name, if none is specified.  This field supports
    # Go templating and is supplied with the ObjectMeta of the
    # enclosing Service or Configuration, so values such as
    # {{.Name}} are also valid.
    container-name-template: "user-container"
kind: ConfigMap
metadata:
  labels:
    serving.knative.dev/release: "v0.7.1"
  name: config-defaults
  namespace: knative-serving

---
apiVersion: v1
data:
  _example: |
    ################################
    #                              #
    #    EXAMPLE CONFIGURATION     #
    #                              #
    ################################

    # This block is not actually functional configuration,
    # but serves to illustrate the available configuration
    # options and document them in a way that is accessible
    # to users that `kubectl edit` this config map.
    #
    # These sample configuration options may be copied out of
    # this example block and unindented to be in the data block
    # to actually change the configuration.

    # List of repositories for which tag to digest resolving should be skipped
    registriesSkippingTagResolving: "ko.local,dev.local"
  queueSidecarImage: gcr.io/knative-releases/github.com/knative/serving/cmd/queue@sha256:89fb5a1d2d9c0abd10ce3135c02f9e9ffbf93087a3ece7481615a0f9d9209713
kind: ConfigMap
metadata:
  labels:
    serving.knative.dev/release: "v0.7.1"
  name: config-deployment
  namespace: knative-serving

---
apiVersion: v1
data:
  _example: |
    ################################
    #                              #
    #    EXAMPLE CONFIGURATION     #
    #                              #
    ################################

    # This block is not actually functional configuration,
    # but serves to illustrate the available configuration
    # options and document them in a way that is accessible
    # to users that `kubectl edit` this config map.
    #
    # These sample configuration options may be copied out of
    # this example block and unindented to be in the data block
    # to actually change the configuration.

    # Default value for domain.
    # Although it will match all routes, it is the least-specific rule so it
    # will only be used if no other domain matches.
    example.com: |

    # These are example settings of domain.
    # example.org will be used for routes having app=nonprofit.
    example.org: |
      selector:
        app: nonprofit

    # Routes having domain suffix of 'svc.cluster.local' will not be exposed
    # through Ingress. You can define your own label selector to assign that
    # domain suffix to your Route here, or you can set the label
    #    "serving.knative.dev/visibility=cluster-local"
    # to achieve the same effect.  This shows how to make routes having
    # the label app=secret only exposed to the local cluster.
    svc.cluster.local: |
      selector:
        app: secret
kind: ConfigMap
metadata:
  labels:
    serving.knative.dev/release: "v0.7.1"
  name: config-domain
  namespace: knative-serving

---
apiVersion: v1
data:
  _example: |
    ################################
    #                              #
    #    EXAMPLE CONFIGURATION     #
    #                              #
    ################################

    # This block is not actually functional configuration,
    # but serves to illustrate the available configuration
    # options and document them in a way that is accessible
    # to users that `kubectl edit` this config map.
    #
    # These sample configuration options may be copied out of
    # this example block and unindented to be in the data block
    # to actually change the configuration.

    # Delay after revision creation before considering it for GC
    stale-revision-create-delay: "24h"

    # Duration since a route has been pointed at a revision before it should be GC'd
    # This minus lastpinned-debounce be longer than the controller resync period (10 hours)
    stale-revision-timeout: "15h"

    # Minimum number of generations of revisions to keep before considering for GC
    stale-revision-minimum-generations: "1"

    # To avoid constant updates, we allow an existing annotation to be stale by this
    # amount before we update the timestamp
    stale-revision-lastpinned-debounce: "5h"
kind: ConfigMap
metadata:
  labels:
    serving.knative.dev/release: "v0.7.1"
  name: config-gc
  namespace: knative-serving

---
apiVersion: v1
data:
  _example: |
    ################################
    #                              #
    #    EXAMPLE CONFIGURATION     #
    #                              #
    ################################

    # This block is not actually functional configuration,
    # but serves to illustrate the available configuration
    # options and document them in a way that is accessible
    # to users that `kubectl edit` this config map.
    #
    # These sample configuration options may be copied out of
    # this example block and unindented to be in the data block
    # to actually change the configuration.

    # Common configuration for all Knative codebase
    zap-logger-config: |
      {{
        "level": "info",
        "development": false,
        "outputPaths": ["stdout"],
        "errorOutputPaths": ["stderr"],
        "encoding": "json",
        "encoderConfig": {{
          "timeKey": "ts",
          "levelKey": "level",
          "nameKey": "logger",
          "callerKey": "caller",
          "messageKey": "msg",
          "stacktraceKey": "stacktrace",
          "lineEnding": "",
          "levelEncoder": "",
          "timeEncoder": "iso8601",
          "durationEncoder": "",
          "callerEncoder": ""
        }}
      }}

    # Log level overrides
    # For all components except the autoscaler and queue proxy,
    # changes are be picked up immediately.
    # For autoscaler and queue proxy, changes require recreation of the pods.
    loglevel.controller: "info"
    loglevel.autoscaler: "info"
    loglevel.queueproxy: "info"
    loglevel.webhook: "info"
    loglevel.activator: "info"
kind: ConfigMap
metadata:
  labels:
    serving.knative.dev/release: "v0.7.1"
  name: config-logging
  namespace: knative-serving

---
apiVersion: v1
data:
    clusteringress.class: "ambassador.ingress.networking.knative.dev"

    # domainTemplate specifies the golang text template string to use
    # when constructing the Knative service's DNS name. The default
    # value is "{{.Name}}.{{.Namespace}}.{{.Domain}}". And those three
    # values (Name, Namespace, Domain) are the only variables defined.
    #
    # Changing this value might be necessary when the extra levels in
    # the domain name generated is problematic for wildcard certificates
    # that only support a single level of domain name added to the
    # certificate's domain. In those cases you might consider using a value
    # of "{{.Name}}-{{.Namespace}}.{{.Domain}}", or removing the Namespace
    # entirely from the template. When choosing a new value be thoughtful
    # of the potential for conflicts - for example, when users choose to use
    # characters such as `-` in their service, or namespace, names.
    # {{.Annotations}} can be used for any customization in the go template if needed.
    # We strongly recommend keeping namespace part of the template to avoid domain name clashes
    # Example '{{.Name}}-{{.Namespace}}.{{ index .Annotations "sub"}}.{{.Domain}}'
    # and you have an annotation {{"sub":"foo"}}, then the generated template would be {{Name}}-{{Namespace}}.foo.{{Domain}}
    domainTemplate: "{{.Name}}.{{.Namespace}}.{{.Domain}}"

    # tagTemplate specifies the golang text template string to use
    # when constructing the DNS name for "tags" within the traffic blocks
    # of Routes and Configuration.  This is used in conjunction with the
    # domainTemplate above to determine the full URL for the tag.
    tagTemplate: "{{.Name}}-{{.Tag}}"

    # Controls whether TLS certificates are automatically provisioned and
    # installed in the Knative ingress to terminate external TLS connection.
    # 1. Enabled: enabling auto-TLS feature.
    # 2. Disabled: disabling auto-TLS feature.
    autoTLS: "Disabled"

    # Controls the behavior of the HTTP endpoint for the Knative ingress.
    # It requires autoTLS to be enabled.
    # 1. Enabled: The Knative ingress will be able to serve HTTP connection.
    # 2. Disabled: The Knative ingress ter will reject HTTP traffic.
    # 3. Redirected: The Knative ingress will send a 302 redirect for all
    # http connections, asking the clients to use HTTPS
    httpProtocol: "Enabled"
kind: ConfigMap
metadata:
  labels:
    serving.knative.dev/release: "v0.7.1"
  name: config-network
  namespace: knative-serving

---
apiVersion: v1
data:
  _example: |
    ################################
    #                              #
    #    EXAMPLE CONFIGURATION     #
    #                              #
    ################################

    # This block is not actually functional configuration,
    # but serves to illustrate the available configuration
    # options and document them in a way that is accessible
    # to users that `kubectl edit` this config map.
    #
    # These sample configuration options may be copied out of
    # this example block and unindented to be in the data block
    # to actually change the configuration.

    # logging.enable-var-log-collection defaults to false.
    # The fluentd daemon set will be set up to collect /var/log if
    # this flag is true.
    logging.enable-var-log-collection: false

    # logging.revision-url-template provides a template to use for producing the
    # logging URL that is injected into the status of each Revision.
    # This value is what you might use the the Knative monitoring bundle, and provides
    # access to Kibana after setting up kubectl proxy.
    logging.revision-url-template: |
      http://localhost:8001/api/v1/namespaces/knative-monitoring/services/kibana-logging/proxy/app/kibana#/discover?_a=(query:(match:(kubernetes.labels.knative-dev%2FrevisionUID:(query:'${{REVISION_UID}}',type:phrase))))

    # If non-empty, this enables queue proxy writing request logs to stdout.
    # The value determines the shape of the request logs and it must be a valid go text/template.
    # It is important to keep this as a single line. Multiple lines are parsed as separate entities
    # by most collection agents and will split the request logs into multiple records.
    #
    # The following fields and functions are available to the template:
    #
    # Request: An http.Request (see https://golang.org/pkg/net/http/#Request)
    # representing an HTTP request received by the server.
    #
    # Response:
    # struct {{
    #   Code    int       // HTTP status code (see https://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml)
    #   Size    int       // An int representing the size of the response.
    #   Latency float64   // A float64 representing the latency of the response in seconds.
    # }}
    #
    # Revision:
    # struct {{
    #   Name          string  // Knative revision name
    #   Namespace     string  // Knative revision namespace
    #   Service       string  // Knative service name
    #   Configuration string  // Knative configuration name
    #   PodName       string  // Name of the pod hosting the revision
    #   PodIP         string  // IP of the pod hosting the revision
    # }}
    #
    logging.request-log-template: '{{"httpRequest": {{"requestMethod": "{{{{.Request.Method}}}}", "requestUrl": "{{{{js .Request.RequestURI}}}}", "requestSize": "{{{{.Request.ContentLength}}}}", "status": {{{{.Response.Code}}}}, "responseSize": "{{{{.Response.Size}}}}", "userAgent": "{{{{js .Request.UserAgent}}}}", "remoteIp": "{{{{js .Request.RemoteAddr}}}}", "serverIp": "{{{{.Revision.PodIP}}}}", "referer": "{{{{js .Request.Referer}}}}", "latency": "{{{{.Response.Latency}}}}s", "protocol": "{{{{.Request.Proto}}}}"}}, "traceId": "{{{{index .Request.Header "X-B3-Traceid"}}}}"}}'

    # metrics.backend-destination field specifies the system metrics destination.
    # It supports either prometheus (the default) or stackdriver.
    # Note: Using stackdriver will incur additional charges
    metrics.backend-destination: prometheus

    # metrics.request-metrics-backend-destination specifies the request metrics
    # destination. If non-empty, it enables queue proxy to send request metrics.
    # Currently supported values: prometheus, stackdriver.
    metrics.request-metrics-backend-destination: prometheus

    # metrics.stackdriver-project-id field specifies the stackdriver project ID. This
    # field is optional. When running on GCE, application default credentials will be
    # used if this field is not provided.
    metrics.stackdriver-project-id: "<your stackdriver project id>"

    # metrics.allow-stackdriver-custom-metrics indicates whether it is allowed to send metrics to
    # Stackdriver using "global" resource type and custom metric type if the
    # metrics are not supported by "knative_revision" resource type. Setting this
    # flag to "true" could cause extra Stackdriver charge.
    # If metrics.backend-destination is not Stackdriver, this is ignored.
    metrics.allow-stackdriver-custom-metrics: "false"
kind: ConfigMap
metadata:
  labels:
    serving.knative.dev/release: "v0.7.1"
  name: config-observability
  namespace: knative-serving

---
apiVersion: v1
data:
  _example: |
    ################################
    #                              #
    #    EXAMPLE CONFIGURATION     #
    #                              #
    ################################

    # This block is not actually functional configuration,
    # but serves to illustrate the available configuration
    # options and document them in a way that is accessible
    # to users that `kubectl edit` this config map.
    #
    # These sample configuration options may be copied out of
    # this example block and unindented to be in the data block
    # to actually change the configuration.
    #
    # If true we enable adding spans within our applications.
    enable: "false"

    # URL to zipkin collector where traces are sent.
    zipkin-endpoint: "http://zipkin.istio-system.svc.cluster.local:9411/api/v2/spans"

    # Enable zipkin debug mode. This allows all spans to be sent to the server
    # bypassing sampling.
    debug: "false"

    # Percentage (0-1) of requests to trace
    sample-rate: "0.1"
kind: ConfigMap
metadata:
  labels:
    serving.knative.dev/release: "v0.7.1"
  name: config-tracing
  namespace: knative-serving

---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    serving.knative.dev/release: "v0.7.1"
  name: controller
  namespace: knative-serving
spec:
  replicas: 1
  selector:
    matchLabels:
      app: controller
  template:
    metadata:
      labels:
        app: controller
        serving.knative.dev/release: "v0.7.1"
    spec:
      containers:
      - env:
        - name: SYSTEM_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: CONFIG_LOGGING_NAME
          value: config-logging
        - name: CONFIG_OBSERVABILITY_NAME
          value: config-observability
        - name: METRICS_DOMAIN
          value: knative.dev/serving
        image: gcr.io/knative-releases/github.com/knative/serving/cmd/controller@sha256:36e48772b4a38d4790c4b72d3e05c5552b3b083709ba6bf3f355af0c4ebb216a
        name: controller
        ports:
        - containerPort: 9090
          name: metrics
        resources:
          limits:
            cpu: 1000m
            memory: 1000Mi
          requests:
            cpu: 100m
            memory: 100Mi
        securityContext:
          allowPrivilegeEscalation: false
        volumeMounts:
        - mountPath: /etc/config-logging
          name: config-logging
      serviceAccountName: controller
      volumes:
      - configMap:
          name: config-logging
        name: config-logging

---
apiVersion: apiregistration.k8s.io/v1beta1
kind: APIService
metadata:
  labels:
    autoscaling.knative.dev/metric-provider: custom-metrics
    serving.knative.dev/release: "v0.7.1"
  name: v1beta1.custom.metrics.k8s.io
spec:
  group: custom.metrics.k8s.io
  groupPriorityMinimum: 100
  insecureSkipTLSVerify: true
  service:
    name: autoscaler
    namespace: knative-serving
  version: v1beta1
  versionPriority: 100

---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    networking.knative.dev/certificate-provider: cert-manager
    serving.knative.dev/release: "v0.7.1"
  name: networking-certmanager
  namespace: knative-serving
spec:
  replicas: 1
  selector:
    matchLabels:
      app: networking-certmanager
  template:
    metadata:
      labels:
        app: networking-certmanager
    spec:
      containers:
      - env:
        - name: SYSTEM_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: CONFIG_LOGGING_NAME
          value: config-logging
        - name: CONFIG_OBSERVABILITY_NAME
          value: config-observability
        - name: METRICS_DOMAIN
          value: knative.dev/serving
        image: gcr.io/knative-releases/github.com/knative/serving/cmd/networking/certmanager@sha256:0868e623602dfa736092baf15c71930dff67a5eec0d89a689496525b32bdad08
        name: networking-certmanager
        ports:
        - containerPort: 9090
          name: metrics
        resources:
          limits:
            cpu: 1000m
            memory: 1000Mi
          requests:
            cpu: 100m
            memory: 100Mi
        securityContext:
          allowPrivilegeEscalation: false
        volumeMounts:
        - mountPath: /etc/config-logging
          name: config-logging
      serviceAccountName: controller
      volumes:
      - configMap:
          name: config-logging
        name: config-logging

---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    serving.knative.dev/release: "v0.7.1"
  name: webhook
  namespace: knative-serving
spec:
  replicas: 1
  selector:
    matchLabels:
      app: webhook
      role: webhook
  template:
    metadata:
      annotations:
        cluster-autoscaler.kubernetes.io/safe-to-evict: "false"
        sidecar.istio.io/inject: "false"
      labels:
        app: webhook
        role: webhook
        serving.knative.dev/release: "v0.7.1"
    spec:
      containers:
      - env:
        - name: SYSTEM_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: CONFIG_LOGGING_NAME
          value: config-logging
        image: gcr.io/knative-releases/github.com/knative/serving/cmd/webhook@sha256:76e726d1f3f015623513224c3787793f0e71294f8df9e6dca46dc92f31bec1c3
        name: webhook
        resources:
          limits:
            cpu: 200m
            memory: 200Mi
          requests:
            cpu: 20m
            memory: 20Mi
        securityContext:
          allowPrivilegeEscalation: false
        volumeMounts:
        - mountPath: /etc/config-logging
          name: config-logging
      serviceAccountName: controller
      volumes:
      - configMap:
          name: config-logging
        name: config-logging
"""

KNATIVE_SERVING_080 = """
---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    serving.knative.dev/release: "v0.8.0"
  name: knative-serving

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    autoscaling.knative.dev/metric-provider: custom-metrics
    serving.knative.dev/release: "v0.8.0"
  name: custom-metrics-server-resources
rules:
- apiGroups:
  - custom.metrics.k8s.io
  resources:
  - '*'
  verbs:
  - '*'

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    rbac.authorization.k8s.io/aggregate-to-admin: "true"
    serving.knative.dev/release: "v0.8.0"
  name: knative-serving-namespaced-admin
rules:
- apiGroups:
  - serving.knative.dev
  - networking.internal.knative.dev
  - autoscaling.internal.knative.dev
  resources:
  - '*'
  verbs:
  - '*'

---
aggregationRule:
  clusterRoleSelectors:
  - matchLabels:
      serving.knative.dev/controller: "true"
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    serving.knative.dev/release: "v0.8.0"
  name: knative-serving-admin
rules: []
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    serving.knative.dev/controller: "true"
    serving.knative.dev/release: "v0.8.0"
  name: knative-serving-core
rules:
- apiGroups:
  - ""
  resources:
  - pods
  - namespaces
  - secrets
  - configmaps
  - endpoints
  - services
  - events
  - serviceaccounts
  verbs:
  - get
  - list
  - create
  - update
  - delete
  - patch
  - watch
- apiGroups:
  - ""
  resources:
  - endpoints/restricted
  verbs:
  - create
- apiGroups:
  - apps
  resources:
  - deployments
  - deployments/finalizers
  verbs:
  - get
  - list
  - create
  - update
  - delete
  - patch
  - watch
- apiGroups:
  - admissionregistration.k8s.io
  resources:
  - mutatingwebhookconfigurations
  verbs:
  - get
  - list
  - create
  - update
  - delete
  - patch
  - watch
- apiGroups:
  - apiextensions.k8s.io
  resources:
  - customresourcedefinitions
  verbs:
  - get
  - list
  - create
  - update
  - delete
  - patch
  - watch
- apiGroups:
  - autoscaling
  resources:
  - horizontalpodautoscalers
  verbs:
  - get
  - list
  - create
  - update
  - delete
  - patch
  - watch
- apiGroups:
  - serving.knative.dev
  - autoscaling.internal.knative.dev
  - networking.internal.knative.dev
  resources:
  - '*'
  - '*/status'
  - '*/finalizers'
  verbs:
  - get
  - list
  - create
  - update
  - delete
  - deletecollection
  - patch
  - watch
- apiGroups:
  - caching.internal.knative.dev
  resources:
  - images
  verbs:
  - get
  - list
  - create
  - update
  - delete
  - patch
  - watch

---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    serving.knative.dev/release: "v0.8.0"
  name: controller
  namespace: knative-serving

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    autoscaling.knative.dev/metric-provider: custom-metrics
    serving.knative.dev/release: "v0.8.0"
  name: custom-metrics:system:auth-delegator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:auth-delegator
subjects:
- kind: ServiceAccount
  name: controller
  namespace: knative-serving

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    autoscaling.knative.dev/metric-provider: custom-metrics
    serving.knative.dev/release: "v0.8.0"
  name: hpa-controller-custom-metrics
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: custom-metrics-server-resources
subjects:
- kind: ServiceAccount
  name: horizontal-pod-autoscaler
  namespace: kube-system

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    serving.knative.dev/release: "v0.8.0"
  name: knative-serving-controller-admin
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: knative-serving-admin
subjects:
- kind: ServiceAccount
  name: controller
  namespace: knative-serving

---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  labels:
    autoscaling.knative.dev/metric-provider: custom-metrics
    serving.knative.dev/release: "v0.8.0"
  name: custom-metrics-auth-reader
  namespace: kube-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: extension-apiserver-authentication-reader
subjects:
- kind: ServiceAccount
  name: controller
  namespace: knative-serving

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
    serving.knative.dev/release: "v0.8.0"
  name: certificates.networking.internal.knative.dev
spec:
  additionalPrinterColumns:
  - JSONPath: .status.conditions[?(@.type=="Ready")].status
    name: Ready
    type: string
  - JSONPath: .status.conditions[?(@.type=="Ready")].reason
    name: Reason
    type: string
  group: networking.internal.knative.dev
  names:
    categories:
    - knative-internal
    - networking
    kind: Certificate
    plural: certificates
    shortNames:
    - kcert
    singular: certificate
  scope: Namespaced
  subresources:
    status: {{}}
  version: v1alpha1

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
    serving.knative.dev/release: "v0.8.0"
  name: clusteringresses.networking.internal.knative.dev
spec:
  additionalPrinterColumns:
  - JSONPath: .status.conditions[?(@.type=='Ready')].status
    name: Ready
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].reason
    name: Reason
    type: string
  group: networking.internal.knative.dev
  names:
    categories:
    - knative-internal
    - networking
    kind: ClusterIngress
    plural: clusteringresses
    singular: clusteringress
  scope: Cluster
  subresources:
    status: {{}}
  versions:
  - name: v1alpha1
    served: true
    storage: true

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
  name: images.caching.internal.knative.dev
spec:
  group: caching.internal.knative.dev
  names:
    categories:
    - knative-internal
    - caching
    kind: Image
    plural: images
    shortNames:
    - img
    singular: image
  scope: Namespaced
  subresources:
    status: {{}}
  version: v1alpha1

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
    serving.knative.dev/release: "v0.8.0"
  name: ingresses.networking.internal.knative.dev
spec:
  additionalPrinterColumns:
  - JSONPath: .status.conditions[?(@.type=='Ready')].status
    name: Ready
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].reason
    name: Reason
    type: string
  group: networking.internal.knative.dev
  names:
    categories:
    - knative-internal
    - networking
    kind: Ingress
    plural: ingresses
    shortNames:
    - ing
    singular: ingress
  scope: Namespaced
  subresources:
    status: {{}}
  versions:
  - name: v1alpha1
    served: true
    storage: true

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
    serving.knative.dev/release: "v0.8.0"
  name: metrics.autoscaling.internal.knative.dev
spec:
  additionalPrinterColumns:
  - JSONPath: .status.conditions[?(@.type=='Ready')].status
    name: Ready
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].reason
    name: Reason
    type: string
  group: autoscaling.internal.knative.dev
  names:
    categories:
    - knative-internal
    - autoscaling
    kind: Metric
    plural: metrics
    singular: metric
  scope: Namespaced
  subresources:
    status: {{}}
  version: v1alpha1

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
    serving.knative.dev/release: "v0.8.0"
  name: podautoscalers.autoscaling.internal.knative.dev
spec:
  additionalPrinterColumns:
  - JSONPath: .status.conditions[?(@.type=='Ready')].status
    name: Ready
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].reason
    name: Reason
    type: string
  group: autoscaling.internal.knative.dev
  names:
    categories:
    - knative-internal
    - autoscaling
    kind: PodAutoscaler
    plural: podautoscalers
    shortNames:
    - kpa
    - pa
    singular: podautoscaler
  scope: Namespaced
  subresources:
    status: {{}}
  versions:
  - name: v1alpha1
    served: true
    storage: true

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
    serving.knative.dev/release: "v0.8.0"
  name: serverlessservices.networking.internal.knative.dev
spec:
  additionalPrinterColumns:
  - JSONPath: .spec.mode
    name: Mode
    type: string
  - JSONPath: .status.serviceName
    name: ServiceName
    type: string
  - JSONPath: .status.privateServiceName
    name: PrivateServiceName
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].status
    name: Ready
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].reason
    name: Reason
    type: string
  group: networking.internal.knative.dev
  names:
    categories:
    - knative-internal
    - networking
    kind: ServerlessService
    plural: serverlessservices
    shortNames:
    - sks
    singular: serverlessservice
  scope: Namespaced
  subresources:
    status: {{}}
  versions:
  - name: v1alpha1
    served: true
    storage: true

---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: activator
    serving.knative.dev/release: "v0.8.0"
  name: activator-service
  namespace: knative-serving
spec:
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: 8012
  - name: http2
    port: 81
    protocol: TCP
    targetPort: 8013
  - name: metrics
    port: 9090
    protocol: TCP
    targetPort: 9090
  selector:
    app: activator
  type: ClusterIP

---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: controller
    serving.knative.dev/release: "v0.8.0"
  name: controller
  namespace: knative-serving
spec:
  ports:
  - name: metrics
    port: 9090
    protocol: TCP
    targetPort: 9090
  selector:
    app: controller

---
apiVersion: v1
kind: Service
metadata:
  labels:
    role: webhook
    serving.knative.dev/release: "v0.8.0"
  name: webhook
  namespace: knative-serving
spec:
  ports:
  - port: 443
    targetPort: 8443
  selector:
    role: webhook

---
apiVersion: caching.internal.knative.dev/v1alpha1
kind: Image
metadata:
  labels:
    serving.knative.dev/release: "v0.8.0"
  name: queue-proxy
  namespace: knative-serving
spec:
  image: gcr.io/knative-releases/knative.dev/serving/cmd/queue@sha256:e0654305370cf3bbbd0f56f97789c92cf5215f752b70902eba5d5fc0e88c5aca

---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    serving.knative.dev/release: "v0.8.0"
  name: activator
  namespace: knative-serving
spec:
  selector:
    matchLabels:
      app: activator
      role: activator
  template:
    metadata:
      annotations:
        cluster-autoscaler.kubernetes.io/safe-to-evict: "false"
      labels:
        app: activator
        role: activator
        serving.knative.dev/release: "v0.8.0"
    spec:
      containers:
      - args:
        - -logtostderr=false
        - -stderrthreshold=FATAL
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: SYSTEM_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: CONFIG_LOGGING_NAME
          value: config-logging
        - name: CONFIG_OBSERVABILITY_NAME
          value: config-observability
        - name: METRICS_DOMAIN
          value: knative.dev/serving
        image: gcr.io/knative-releases/knative.dev/serving/cmd/activator@sha256:88d864eb3c47881cf7ac058479d1c735cc3cf4f07a11aad0621cd36dcd9ae3c6
        livenessProbe:
          httpGet:
            httpHeaders:
            - name: k-kubelet-probe
              value: activator
            path: /healthz
            port: 8012
        name: activator
        ports:
        - containerPort: 8012
          name: http1-port
        - containerPort: 8013
          name: h2c-port
        - containerPort: 9090
          name: metrics-port
        readinessProbe:
          httpGet:
            httpHeaders:
            - name: k-kubelet-probe
              value: activator
            path: /healthz
            port: 8012
        resources:
          limits:
            cpu: 1000m
            memory: 600Mi
          requests:
            cpu: 300m
            memory: 60Mi
        securityContext:
          allowPrivilegeEscalation: false
      serviceAccountName: controller
      terminationGracePeriodSeconds: 300
---
apiVersion: autoscaling/v2beta1
kind: HorizontalPodAutoscaler
metadata:
  name: activator
  namespace: knative-serving
spec:
  maxReplicas: 20
  metrics:
  - resource:
      name: cpu
      targetAverageUtilization: 100
    type: Resource
  minReplicas: 1
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: activator

---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    autoscaling.knative.dev/autoscaler-provider: hpa
    serving.knative.dev/release: "v0.8.0"
  name: autoscaler-hpa
  namespace: knative-serving
spec:
  replicas: 1
  selector:
    matchLabels:
      app: autoscaler-hpa
  template:
    metadata:
      labels:
        app: autoscaler-hpa
        serving.knative.dev/release: "v0.8.0"
    spec:
      containers:
      - env:
        - name: SYSTEM_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: CONFIG_LOGGING_NAME
          value: config-logging
        - name: CONFIG_OBSERVABILITY_NAME
          value: config-observability
        - name: METRICS_DOMAIN
          value: knative.dev/serving
        image: gcr.io/knative-releases/knative.dev/serving/cmd/autoscaler-hpa@sha256:a7801c3cf4edecfa51b7bd2068f97941f6714f7922cb4806245377c2b336b723
        name: autoscaler-hpa
        ports:
        - containerPort: 9090
          name: metrics
        resources:
          limits:
            cpu: 1000m
            memory: 1000Mi
          requests:
            cpu: 100m
            memory: 100Mi
        securityContext:
          allowPrivilegeEscalation: false
      serviceAccountName: controller

---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: autoscaler
    serving.knative.dev/release: "v0.8.0"
  name: autoscaler
  namespace: knative-serving
spec:
  ports:
  - name: http
    port: 8080
    protocol: TCP
    targetPort: 8080
  - name: metrics
    port: 9090
    protocol: TCP
    targetPort: 9090
  - name: custom-metrics
    port: 443
    protocol: TCP
    targetPort: 8443
  selector:
    app: autoscaler

---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    serving.knative.dev/release: "v0.8.0"
  name: autoscaler
  namespace: knative-serving
spec:
  replicas: 1
  selector:
    matchLabels:
      app: autoscaler
  template:
    metadata:
      annotations:
        cluster-autoscaler.kubernetes.io/safe-to-evict: "false"
      labels:
        app: autoscaler
        serving.knative.dev/release: "v0.8.0"
    spec:
      containers:
      - args:
        - --secure-port=8443
        - --cert-dir=/tmp
        env:
        - name: SYSTEM_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: CONFIG_LOGGING_NAME
          value: config-logging
        - name: CONFIG_OBSERVABILITY_NAME
          value: config-observability
        - name: METRICS_DOMAIN
          value: knative.dev/serving
        image: gcr.io/knative-releases/knative.dev/serving/cmd/autoscaler@sha256:aeaacec4feedee309293ac21da13e71a05a2ad84b1d5fcc01ffecfa6cfbb2870
        livenessProbe:
          httpGet:
            httpHeaders:
            - name: k-kubelet-probe
              value: autoscaler
            path: /healthz
            port: 8080
        name: autoscaler
        ports:
        - containerPort: 8080
          name: websocket
        - containerPort: 9090
          name: metrics
        - containerPort: 8443
          name: custom-metrics
        readinessProbe:
          httpGet:
            httpHeaders:
            - name: k-kubelet-probe
              value: autoscaler
            path: /healthz
            port: 8080
        resources:
          limits:
            cpu: 300m
            memory: 400Mi
          requests:
            cpu: 30m
            memory: 40Mi
        securityContext:
          allowPrivilegeEscalation: false
      serviceAccountName: controller

---
apiVersion: v1
data:
  _example: |
    ################################
    #                              #
    #    EXAMPLE CONFIGURATION     #
    #                              #
    ################################

    # This block is not actually functional configuration,
    # but serves to illustrate the available configuration
    # options and document them in a way that is accessible
    # to users that `kubectl edit` this config map.
    #
    # These sample configuration options may be copied out of
    # this example block and unindented to be in the data block
    # to actually change the configuration.

    # The Revision ContainerConcurrency field specifies the maximum number
    # of requests the Container can handle at once. Container concurrency
    # target percentage is how much of that maximum to use in a stable
    # state. E.g. if a Revision specifies ContainerConcurrency of 10, then
    # the Autoscaler will try to maintain 7 concurrent connections per pod
    # on average.
    # Note: this limit will be applied to container concurrency set at every
    # level (ConfigMap, Revision Spec or Annotation).
    # For legacy and backwards compatibility reasons, this value also accepts
    # fractional values in (0, 1] interval (i.e. 0.7 ⇒ 70%).
    # Thus minimal percentage value must be greater than 1.0, or it will be
    # treated as a fraction.
    container-concurrency-target-percentage: "70"

    # The container concurrency target default is what the Autoscaler will
    # try to maintain when the Revision specifies unlimited concurrency.
    # Even when specifying unlimited concurrency, the autoscaler will
    # horizontally scale the application based on this target concurrency.
    container-concurrency-target-default: "100"

    # The target burst capacity specifies the size of burst in concurrent
    # requests that the system operator expects the system will receive.
    # Autoscaler will try to protect the system from queueing by introducing
    # Activator in the request path if the current spare capacity of the
    # service is less than this setting.
    # If this setting is 0, then Activator will be in the request path only
    # when the revision is scaled to 0.
    # If this setting is > 0 and container-concurrency-target-percentage is
    # 100% or 1.0, then activator will always be in the request path.
    # -1 denotes unlimited target-burst-capacity and activator will always
    # be in the request path.
    # Other negative values are invalid.
    target-burst-capacity: "0"

    # When operating in a stable mode, the autoscaler operates on the
    # average concurrency over the stable window.
    stable-window: "60s"

    # When observed average concurrency during the panic window reaches
    # panic-threshold-percentage the target concurrency, the autoscaler
    # enters panic mode. When operating in panic mode, the autoscaler
    # scales on the average concurrency over the panic window which is
    # panic-window-percentage of the stable-window.
    panic-window-percentage: "10.0"

    # Absolute panic window duration.
    # Deprecated in favor of panic-window-percentage.
    # Existing revisions will continue to scale based on panic-window
    # but new revisions will default to panic-window-percentage.
    panic-window: "6s"

    # The percentage of the container concurrency target at which to
    # enter panic mode when reached within the panic window.
    panic-threshold-percentage: "200.0"

    # Max scale up rate limits the rate at which the autoscaler will
    # increase pod count. It is the maximum ratio of desired pods versus
    # observed pods.
    max-scale-up-rate: "1000.0"

    # Scale to zero feature flag
    enable-scale-to-zero: "true"

    # Tick interval is the time between autoscaling calculations.
    tick-interval: "2s"

    # Dynamic parameters (take effect when config map is updated):

    # Scale to zero grace period is the time an inactive revision is left
    # running before it is scaled to zero (min: 30s).
    scale-to-zero-grace-period: "30s"
kind: ConfigMap
metadata:
  labels:
    serving.knative.dev/release: "v0.8.0"
  name: config-autoscaler
  namespace: knative-serving

---
apiVersion: v1
data:
  _example: |
    ################################
    #                              #
    #    EXAMPLE CONFIGURATION     #
    #                              #
    ################################

    # This block is not actually functional configuration,
    # but serves to illustrate the available configuration
    # options and document them in a way that is accessible
    # to users that `kubectl edit` this config map.
    #
    # These sample configuration options may be copied out of
    # this example block and unindented to be in the data block
    # to actually change the configuration.

    # revision-timeout-seconds contains the default number of
    # seconds to use for the revision's per-request timeout, if
    # none is specified.
    revision-timeout-seconds: "300"  # 5 minutes

    # max-revision-timeout-seconds contains the maximum number of
    # seconds that can be used for revision-timeout-seconds.
    # This value must be greater than or equal to revision-timeout-seconds.
    # If omitted, the system default is used (600 seconds).
    max-revision-timeout-seconds: "600"  # 10 minutes

    # revision-cpu-request contains the cpu allocation to assign
    # to revisions by default.  If omitted, no value is specified
    # and the system default is used.
    revision-cpu-request: "400m"  # 0.4 of a CPU (aka 400 milli-CPU)

    # revision-memory-request contains the memory allocation to assign
    # to revisions by default.  If omitted, no value is specified
    # and the system default is used.
    revision-memory-request: "100M"  # 100 megabytes of memory

    # revision-cpu-limit contains the cpu allocation to limit
    # revisions to by default.  If omitted, no value is specified
    # and the system default is used.
    revision-cpu-limit: "1000m"  # 1 CPU (aka 1000 milli-CPU)

    # revision-memory-limit contains the memory allocation to limit
    # revisions to by default.  If omitted, no value is specified
    # and the system default is used.
    revision-memory-limit: "200M"  # 200 megabytes of memory

    # container-name-template contains a template for the default
    # container name, if none is specified.  This field supports
    # Go templating and is supplied with the ObjectMeta of the
    # enclosing Service or Configuration, so values such as
    # {{.Name}} are also valid.
    container-name-template: "user-container"
kind: ConfigMap
metadata:
  labels:
    serving.knative.dev/release: "v0.8.0"
  name: config-defaults
  namespace: knative-serving

---
apiVersion: v1
data:
  _example: |
    ################################
    #                              #
    #    EXAMPLE CONFIGURATION     #
    #                              #
    ################################

    # This block is not actually functional configuration,
    # but serves to illustrate the available configuration
    # options and document them in a way that is accessible
    # to users that `kubectl edit` this config map.
    #
    # These sample configuration options may be copied out of
    # this example block and unindented to be in the data block
    # to actually change the configuration.

    # List of repositories for which tag to digest resolving should be skipped
    registriesSkippingTagResolving: "ko.local,dev.local"
  queueSidecarImage: gcr.io/knative-releases/knative.dev/serving/cmd/queue@sha256:e0654305370cf3bbbd0f56f97789c92cf5215f752b70902eba5d5fc0e88c5aca
kind: ConfigMap
metadata:
  labels:
    serving.knative.dev/release: "v0.8.0"
  name: config-deployment
  namespace: knative-serving

---
apiVersion: v1
data:
  _example: |
    ################################
    #                              #
    #    EXAMPLE CONFIGURATION     #
    #                              #
    ################################

    # This block is not actually functional configuration,
    # but serves to illustrate the available configuration
    # options and document them in a way that is accessible
    # to users that `kubectl edit` this config map.
    #
    # These sample configuration options may be copied out of
    # this example block and unindented to be in the data block
    # to actually change the configuration.

    # Default value for domain.
    # Although it will match all routes, it is the least-specific rule so it
    # will only be used if no other domain matches.
    example.com: |

    # These are example settings of domain.
    # example.org will be used for routes having app=nonprofit.
    example.org: |
      selector:
        app: nonprofit

    # Routes having domain suffix of 'svc.cluster.local' will not be exposed
    # through Ingress. You can define your own label selector to assign that
    # domain suffix to your Route here, or you can set the label
    #    "serving.knative.dev/visibility=cluster-local"
    # to achieve the same effect.  This shows how to make routes having
    # the label app=secret only exposed to the local cluster.
    svc.cluster.local: |
      selector:
        app: secret
kind: ConfigMap
metadata:
  labels:
    serving.knative.dev/release: "v0.8.0"
  name: config-domain
  namespace: knative-serving

---
apiVersion: v1
data:
  _example: |
    ################################
    #                              #
    #    EXAMPLE CONFIGURATION     #
    #                              #
    ################################

    # This block is not actually functional configuration,
    # but serves to illustrate the available configuration
    # options and document them in a way that is accessible
    # to users that `kubectl edit` this config map.
    #
    # These sample configuration options may be copied out of
    # this example block and unindented to be in the data block
    # to actually change the configuration.

    # Delay after revision creation before considering it for GC
    stale-revision-create-delay: "24h"

    # Duration since a route has been pointed at a revision before it should be GC'd
    # This minus lastpinned-debounce be longer than the controller resync period (10 hours)
    stale-revision-timeout: "15h"

    # Minimum number of generations of revisions to keep before considering for GC
    stale-revision-minimum-generations: "1"

    # To avoid constant updates, we allow an existing annotation to be stale by this
    # amount before we update the timestamp
    stale-revision-lastpinned-debounce: "5h"
kind: ConfigMap
metadata:
  labels:
    serving.knative.dev/release: "v0.8.0"
  name: config-gc
  namespace: knative-serving

---
apiVersion: v1
data:
  _example: |
    ################################
    #                              #
    #    EXAMPLE CONFIGURATION     #
    #                              #
    ################################

    # This block is not actually functional configuration,
    # but serves to illustrate the available configuration
    # options and document them in a way that is accessible
    # to users that `kubectl edit` this config map.
    #
    # These sample configuration options may be copied out of
    # this example block and unindented to be in the data block
    # to actually change the configuration.

    # Common configuration for all Knative codebase
    zap-logger-config: |
      {{
        "level": "info",
        "development": false,
        "outputPaths": ["stdout"],
        "errorOutputPaths": ["stderr"],
        "encoding": "json",
        "encoderConfig": {{
          "timeKey": "ts",
          "levelKey": "level",
          "nameKey": "logger",
          "callerKey": "caller",
          "messageKey": "msg",
          "stacktraceKey": "stacktrace",
          "lineEnding": "",
          "levelEncoder": "",
          "timeEncoder": "iso8601",
          "durationEncoder": "",
          "callerEncoder": ""
        }}
      }}

    # Log level overrides
    # For all components except the autoscaler and queue proxy,
    # changes are be picked up immediately.
    # For autoscaler and queue proxy, changes require recreation of the pods.
    loglevel.controller: "info"
    loglevel.autoscaler: "info"
    loglevel.queueproxy: "info"
    loglevel.webhook: "info"
    loglevel.activator: "info"
kind: ConfigMap
metadata:
  labels:
    serving.knative.dev/release: "v0.8.0"
  name: config-logging
  namespace: knative-serving

---
apiVersion: v1
data:
  _example: |
    clusteringress.class: "ambassador.ingress.networking.knative.dev"

    certificate.class: "cert-manager.certificate.networking.internal.knative.dev"

    # domainTemplate specifies the golang text template string to use
    # when constructing the Knative service's DNS name. The default
    # value is "{{.Name}}.{{.Namespace}}.{{.Domain}}". And those three
    # values (Name, Namespace, Domain) are the only variables defined.
    #
    # Changing this value might be necessary when the extra levels in
    # the domain name generated is problematic for wildcard certificates
    # that only support a single level of domain name added to the
    # certificate's domain. In those cases you might consider using a value
    # of "{{.Name}}-{{.Namespace}}.{{.Domain}}", or removing the Namespace
    # entirely from the template. When choosing a new value be thoughtful
    # of the potential for conflicts - for example, when users choose to use
    # characters such as `-` in their service, or namespace, names.
    # {{.Annotations}} can be used for any customization in the go template if needed.
    # We strongly recommend keeping namespace part of the template to avoid domain name clashes
    # Example '{{.Name}}-{{.Namespace}}.{{ index .Annotations "sub"}}.{{.Domain}}'
    # and you have an annotation {{"sub":"foo"}}, then the generated template would be {{Name}}-{{Namespace}}.foo.{{Domain}}
    domainTemplate: "{{.Name}}.{{.Namespace}}.{{.Domain}}"

    # tagTemplate specifies the golang text template string to use
    # when constructing the DNS name for "tags" within the traffic blocks
    # of Routes and Configuration.  This is used in conjunction with the
    # domainTemplate above to determine the full URL for the tag.
    tagTemplate: "{{.Name}}-{{.Tag}}"

    # Controls whether TLS certificates are automatically provisioned and
    # installed in the Knative ingress to terminate external TLS connection.
    # 1. Enabled: enabling auto-TLS feature.
    # 2. Disabled: disabling auto-TLS feature.
    autoTLS: "Disabled"

    # Controls the behavior of the HTTP endpoint for the Knative ingress.
    # It requires autoTLS to be enabled.
    # 1. Enabled: The Knative ingress will be able to serve HTTP connection.
    # 2. Disabled: The Knative ingress ter will reject HTTP traffic.
    # 3. Redirected: The Knative ingress will send a 302 redirect for all
    # http connections, asking the clients to use HTTPS
    httpProtocol: "Enabled"
kind: ConfigMap
metadata:
  labels:
    serving.knative.dev/release: "v0.8.0"
  name: config-network
  namespace: knative-serving

---
apiVersion: v1
data:
  _example: |
    ################################
    #                              #
    #    EXAMPLE CONFIGURATION     #
    #                              #
    ################################

    # This block is not actually functional configuration,
    # but serves to illustrate the available configuration
    # options and document them in a way that is accessible
    # to users that `kubectl edit` this config map.
    #
    # These sample configuration options may be copied out of
    # this example block and unindented to be in the data block
    # to actually change the configuration.

    # logging.enable-var-log-collection defaults to false.
    # The fluentd daemon set will be set up to collect /var/log if
    # this flag is true.
    logging.enable-var-log-collection: false

    # logging.revision-url-template provides a template to use for producing the
    # logging URL that is injected into the status of each Revision.
    # This value is what you might use the the Knative monitoring bundle, and provides
    # access to Kibana after setting up kubectl proxy.
    logging.revision-url-template: |
      http://localhost:8001/api/v1/namespaces/knative-monitoring/services/kibana-logging/proxy/app/kibana#/discover?_a=(query:(match:(kubernetes.labels.serving-knative-dev%2FrevisionUID:(query:'${{REVISION_UID}}',type:phrase))))

    # If non-empty, this enables queue proxy writing request logs to stdout.
    # The value determines the shape of the request logs and it must be a valid go text/template.
    # It is important to keep this as a single line. Multiple lines are parsed as separate entities
    # by most collection agents and will split the request logs into multiple records.
    #
    # The following fields and functions are available to the template:
    #
    # Request: An http.Request (see https://golang.org/pkg/net/http/#Request)
    # representing an HTTP request received by the server.
    #
    # Response:
    # struct {{
    #   Code    int       // HTTP status code (see https://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml)
    #   Size    int       // An int representing the size of the response.
    #   Latency float64   // A float64 representing the latency of the response in seconds.
    # }}
    #
    # Revision:
    # struct {{
    #   Name          string  // Knative revision name
    #   Namespace     string  // Knative revision namespace
    #   Service       string  // Knative service name
    #   Configuration string  // Knative configuration name
    #   PodName       string  // Name of the pod hosting the revision
    #   PodIP         string  // IP of the pod hosting the revision
    # }}
    #
    logging.request-log-template: '{{"httpRequest": {{"requestMethod": "{{{{.Request.Method}}}}", "requestUrl": "{{{{js .Request.RequestURI}}}}", "requestSize": "{{{{.Request.ContentLength}}}}", "status": {{{{.Response.Code}}}}, "responseSize": "{{{{.Response.Size}}}}", "userAgent": "{{{{js .Request.UserAgent}}}}", "remoteIp": "{{{{js .Request.RemoteAddr}}}}", "serverIp": "{{{{.Revision.PodIP}}}}", "referer": "{{{{js .Request.Referer}}}}", "latency": "{{{{.Response.Latency}}}}s", "protocol": "{{{{.Request.Proto}}}}"}}, "traceId": "{{{{index .Request.Header "X-B3-Traceid"}}}}"}}'

    # metrics.backend-destination field specifies the system metrics destination.
    # It supports either prometheus (the default) or stackdriver.
    # Note: Using stackdriver will incur additional charges
    metrics.backend-destination: prometheus

    # metrics.request-metrics-backend-destination specifies the request metrics
    # destination. If non-empty, it enables queue proxy to send request metrics.
    # Currently supported values: prometheus, stackdriver.
    metrics.request-metrics-backend-destination: prometheus

    # metrics.stackdriver-project-id field specifies the stackdriver project ID. This
    # field is optional. When running on GCE, application default credentials will be
    # used if this field is not provided.
    metrics.stackdriver-project-id: "<your stackdriver project id>"

    # metrics.allow-stackdriver-custom-metrics indicates whether it is allowed to send metrics to
    # Stackdriver using "global" resource type and custom metric type if the
    # metrics are not supported by "knative_revision" resource type. Setting this
    # flag to "true" could cause extra Stackdriver charge.
    # If metrics.backend-destination is not Stackdriver, this is ignored.
    metrics.allow-stackdriver-custom-metrics: "false"
kind: ConfigMap
metadata:
  labels:
    serving.knative.dev/release: "v0.8.0"
  name: config-observability
  namespace: knative-serving

---
apiVersion: v1
data:
  _example: |
    ################################
    #                              #
    #    EXAMPLE CONFIGURATION     #
    #                              #
    ################################

    # This block is not actually functional configuration,
    # but serves to illustrate the available configuration
    # options and document them in a way that is accessible
    # to users that `kubectl edit` this config map.
    #
    # These sample configuration options may be copied out of
    # this example block and unindented to be in the data block
    # to actually change the configuration.
    #
    # If true we enable adding spans within our applications.
    enable: "false"

    # URL to zipkin collector where traces are sent.
    zipkin-endpoint: "http://zipkin.istio-system.svc.cluster.local:9411/api/v2/spans"

    # Enable zipkin debug mode. This allows all spans to be sent to the server
    # bypassing sampling.
    debug: "false"

    # Percentage (0-1) of requests to trace
    sample-rate: "0.1"
kind: ConfigMap
metadata:
  labels:
    serving.knative.dev/release: "v0.8.0"
  name: config-tracing
  namespace: knative-serving

---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    serving.knative.dev/release: "v0.8.0"
  name: controller
  namespace: knative-serving
spec:
  replicas: 1
  selector:
    matchLabels:
      app: controller
  template:
    metadata:
      labels:
        app: controller
        serving.knative.dev/release: "v0.8.0"
    spec:
      containers:
      - env:
        - name: SYSTEM_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: CONFIG_LOGGING_NAME
          value: config-logging
        - name: CONFIG_OBSERVABILITY_NAME
          value: config-observability
        - name: METRICS_DOMAIN
          value: knative.dev/serving
        image: gcr.io/knative-releases/knative.dev/serving/cmd/controller@sha256:3b096e55fa907cff53d37dadc5d20c29cea9bb18ed9e921a588fee17beb937df
        name: controller
        ports:
        - containerPort: 9090
          name: metrics
        resources:
          limits:
            cpu: 1000m
            memory: 1000Mi
          requests:
            cpu: 100m
            memory: 100Mi
        securityContext:
          allowPrivilegeEscalation: false
      serviceAccountName: controller

---
apiVersion: apiregistration.k8s.io/v1beta1
kind: APIService
metadata:
  labels:
    autoscaling.knative.dev/metric-provider: custom-metrics
    serving.knative.dev/release: "v0.8.0"
  name: v1beta1.custom.metrics.k8s.io
spec:
  group: custom.metrics.k8s.io
  groupPriorityMinimum: 100
  insecureSkipTLSVerify: true
  service:
    name: autoscaler
    namespace: knative-serving
  version: v1beta1
  versionPriority: 100

---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    serving.knative.dev/release: "v0.8.0"
  name: webhook
  namespace: knative-serving
spec:
  replicas: 1
  selector:
    matchLabels:
      app: webhook
      role: webhook
  template:
    metadata:
      annotations:
        cluster-autoscaler.kubernetes.io/safe-to-evict: "false"
      labels:
        app: webhook
        role: webhook
        serving.knative.dev/release: "v0.8.0"
    spec:
      containers:
      - env:
        - name: SYSTEM_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: CONFIG_LOGGING_NAME
          value: config-logging
        - name: CONFIG_OBSERVABILITY_NAME
          value: config-observability
        - name: METRICS_DOMAIN
          value: knative.dev/serving
        image: gcr.io/knative-releases/knative.dev/serving/cmd/webhook@sha256:c2076674618933df53e90cf9ddd17f5ddbad513b8c95e955e45e37be7ca9e0e8
        name: webhook
        ports:
        - containerPort: 9090
          name: metrics-port
        resources:
          limits:
            cpu: 200m
            memory: 200Mi
          requests:
            cpu: 20m
            memory: 20Mi
        securityContext:
          allowPrivilegeEscalation: false
      serviceAccountName: controller

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
    serving.knative.dev/release: "v0.8.0"
  name: configurations.serving.knative.dev
spec:
  additionalPrinterColumns:
  - JSONPath: .status.latestCreatedRevisionName
    name: LatestCreated
    type: string
  - JSONPath: .status.latestReadyRevisionName
    name: LatestReady
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].status
    name: Ready
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].reason
    name: Reason
    type: string
  group: serving.knative.dev
  names:
    categories:
    - all
    - knative
    - serving
    kind: Configuration
    plural: configurations
    shortNames:
    - config
    - cfg
    singular: configuration
  scope: Namespaced
  subresources:
    status: {{}}
  versions:
  - name: v1alpha1
    served: true
    storage: true

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
    serving.knative.dev/release: "v0.8.0"
  name: revisions.serving.knative.dev
spec:
  additionalPrinterColumns:
  - JSONPath: .metadata.labels['serving\\.knative\\.dev/configuration']
    name: Config Name
    type: string
  - JSONPath: .status.serviceName
    name: K8s Service Name
    type: string
  - JSONPath: .metadata.labels['serving\\.knative\\.dev/configurationGeneration']
    name: Generation
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].status
    name: Ready
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].reason
    name: Reason
    type: string
  group: serving.knative.dev
  names:
    categories:
    - all
    - knative
    - serving
    kind: Revision
    plural: revisions
    shortNames:
    - rev
    singular: revision
  scope: Namespaced
  subresources:
    status: {{}}
  versions:
  - name: v1alpha1
    served: true
    storage: true

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
    serving.knative.dev/release: "v0.8.0"
  name: routes.serving.knative.dev
spec:
  additionalPrinterColumns:
  - JSONPath: .status.url
    name: URL
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].status
    name: Ready
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].reason
    name: Reason
    type: string
  group: serving.knative.dev
  names:
    categories:
    - all
    - knative
    - serving
    kind: Route
    plural: routes
    shortNames:
    - rt
    singular: route
  scope: Namespaced
  subresources:
    status: {{}}
  versions:
  - name: v1alpha1
    served: true
    storage: true

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    knative.dev/crd-install: "true"
    serving.knative.dev/release: "v0.8.0"
  name: services.serving.knative.dev
spec:
  additionalPrinterColumns:
  - JSONPath: .status.url
    name: URL
    type: string
  - JSONPath: .status.latestCreatedRevisionName
    name: LatestCreated
    type: string
  - JSONPath: .status.latestReadyRevisionName
    name: LatestReady
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].status
    name: Ready
    type: string
  - JSONPath: .status.conditions[?(@.type=='Ready')].reason
    name: Reason
    type: string
  group: serving.knative.dev
  names:
    categories:
    - all
    - knative
    - serving
    kind: Service
    plural: services
    shortNames:
    - kservice
    - ksvc
    singular: service
  scope: Namespaced
  subresources:
    status: {{}}
  versions:
  - name: v1alpha1
    served: true
    storage: true
"""
