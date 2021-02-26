# Service Preview Reference

The following is a reference for the various components of Service Preview. 

See [Service Preview Quick Start](../service-preview-install) for detailed installation instructions.

### Traffic Manager

The Traffic Manager is the central point of communication between Traffic Agents in the cluster and Edge Control Daemons on developer workstations.

The following YAML is the basic Traffic Manager installation manifests that is available for download at [https://getambassador.io/yaml/traffic-manager.yaml](/yaml/traffic-manager.yaml).

```yaml
# This is traffic-manager.yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: traffic-manager
  namespace: ambassador
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: traffic-manager
rules:
  - apiGroups: [""]
    resources: ["namespaces", "services", "pods", "secrets"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: traffic-manager
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: traffic-manager
subjects:
  - kind: ServiceAccount
    name: traffic-manager
    namespace: ambassador
---
apiVersion: v1
kind: Service
metadata:
  name: telepresence-proxy
  namespace: ambassador
spec:
  type: ClusterIP
  clusterIP: None
  selector:
    app: telepresence-proxy
  ports:
    - name: sshd
      protocol: TCP
      port: 8022
    - name: api
      protocol: TCP
      port: 8081
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: telepresence-proxy
  namespace: ambassador
  labels:
    app: telepresence-proxy
spec:
  replicas: 1
  selector:
    matchLabels:
      app: telepresence-proxy
  template:
    metadata:
      labels:
        app: telepresence-proxy
    spec:
      containers:
      - name: telepresence-proxy
        image: docker.io/datawire/aes:$version$
        command: [ "traffic-manager" ]
        ports:
          - name: sshd
            containerPort: 8022
        env:
          - name: AMBASSADOR_NAMESPACE
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
          - name: REDIS_URL
            value: ambassador-redis:6379
        volumeMounts:
          - mountPath: /tmp/ambassador-pod-info
            name: ambassador-pod-info
      restartPolicy: Always
      serviceAccountName: traffic-manager
      terminationGracePeriodSeconds: 0
      volumes:
      - downwardAPI:
          items:
          - fieldRef:
              fieldPath: metadata.labels
            path: labels
        name: ambassador-pod-info
```

The Traffic Manager needs to be able to watch resources in the cluster so it is aware of what services are interceptable by Service Preview. The default is to provide a cluster-wide scope for this as shown above so you can run Service Preview in any namespace.

It also requires the ability to read your Ambassador Edge Stack license key from the `ambassador-edge-stack` `Secret`.

#### Traffic Manager Options

- __Remove permission to read `Secret`s__

   If you do not wish to grant read privileges on `Secrets` to the `traffic-manager` `ServiceAccount`, you may mount the `ambassador-edge-stack` secret containing the license key in an extra volume and reference it using the `AMBASSADOR_LICENSE_FILE` environment variable:

   ```yaml
       # [...]
       env:
       - name: AMBASSADOR_LICENSE_FILE
         value: /.config/ambassador/license-key
       # [...]
       volumeMounts:
       - mountPath: /.config/ambassador
         name: ambassador-edge-stack-secrets
         readOnly: true
     # [...]
     volumes:
     - name: ambassador-edge-stack-secrets
       secret:
         secretName: ambassador-edge-stack
   ```

- __Run with namespace scope__

   You can run the Traffic Agent without cluster-wide permissions if you only want to use service preview in a single namespace. 
   
   To do so, you will need use the following manifest which modifies the deployment to run only in the `ambassador` namespace.

   ```yaml
   ---
   apiVersion: v1
   kind: ServiceAccount
   metadata:
     name: traffic-manager
     namespace: ambassador
   ---
   apiVersion: rbac.authorization.k8s.io/v1beta1
   kind: Role
   metadata:
     name: traffic-manager
   rules:
     - apiGroups: [""]
       resources: ["namespaces", "services", "pods", "secrets"]
       verbs: ["get", "list", "watch"]
   ---
   apiVersion: rbac.authorization.k8s.io/v1beta1
   kind: RoleBinding
   metadata:
     name: traffic-manager
   roleRef:
     apiGroup: rbac.authorization.k8s.io
     kind: Role
     name: traffic-manager
   subjects:
     - kind: ServiceAccount
       name: traffic-manager
       namespace: ambassador
   ---
   apiVersion: v1
   kind: Service
   metadata:
     name: telepresence-proxy
     namespace: ambassador
   spec:
     type: ClusterIP
     clusterIP: None
     selector:
       app: telepresence-proxy
     ports:
       - name: sshd
         protocol: TCP
         port: 8022
       - name: api
         protocol: TCP
         port: 8081
   ---
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: telepresence-proxy
     namespace: ambassador
     labels:
       app: telepresence-proxy
   spec:
     replicas: 1
     selector:
       matchLabels:
         app: telepresence-proxy
     template:
       metadata:
         labels:
           app: telepresence-proxy
       spec:
         containers:
         - name: telepresence-proxy
           image: docker.io/datawire/aes:$version$
           command: [ "traffic-manager" ]
           ports:
             - name: sshd
               containerPort: 8022
           env:
             - name: AMBASSADOR_SINGLE_NAMESPACE
               value: "true"
             - name: AMBASSADOR_NAMESPACE
               valueFrom:
                 fieldRef:
                   fieldPath: metadata.namespace
             - name: REDIS_URL
               value: ambassador-redis:6379
           volumeMounts:
             - mountPath: /tmp/ambassador-pod-info
               name: ambassador-pod-info
         restartPolicy: Always
         serviceAccountName: traffic-manager
         terminationGracePeriodSeconds: 0
         volumes:
         - downwardAPI:
             items:
             - fieldRef:
                 fieldPath: metadata.labels
               path: labels
           name: ambassador-pod-info
   ```

### Traffic Agent

Any pod running in a cluster with a Traffic Manager can opt in to intercept functionality by including the Traffic Agent container.

#### Configuring RBAC

Since the Traffic Agent is built on Ambassador Edge Stack, it needs a subset of the same RBAC permissions that Ambassador does. The easiest way to provide this is to create a `ServiceAccount` in your service's namespace, bound to the `traffic-agent` `Role` or `ClusterRole`.

The following YAML is the basic Traffic Agent RBAC configuration manifests that is available for download at [https://getambassador.io/yaml/traffic-agent-rbac.yaml](/yaml/traffic-agent-rbac.yaml).

```yaml
# This is traffic-agent-rbac.yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: traffic-agent
  namespace: default
  labels:
    product: aes
---
## After creating the ServiceAccount, create a service-account-token for traffic-agent with a matching name.
## Since the ambassador-injector will use this token name, it must be deterministic and not auto-generated.
apiVersion: v1
kind: Secret
metadata:
  name: traffic-agent
  namespace: default
  annotations:
    kubernetes.io/service-account.name: traffic-agent
type: kubernetes.io/service-account-token
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: traffic-agent
rules:
  - apiGroups: [""]
    resources: [ "namespaces", "services", "secrets" ]
    verbs: ["get", "list", "watch"]
  - apiGroups: [ "getambassador.io" ]
    resources: [ "*" ]
    verbs: ["get", "list", "watch", "update"]
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: traffic-agent
  labels:
    product: aes
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: traffic-agent
subjects:
  - name: traffic-agent
    namespace: default
    kind: ServiceAccount
```

If you want to include the Traffic Agent with multiple services, they can all use the same `ServiceAccount` name, as long as it exists in every namespace.

##### RBAC Options

- __Run with namespace scope__

   You can reduce the scope of the Traffic Agent if you only want to run Service Preview in a single namespace.

   To do so, create the following RBAC roles instead of the cluster-scoped ones above:

   ```yaml
   # This is traffic-agent-rbac.yaml
   ---
   apiVersion: v1
   kind: ServiceAccount
   metadata:
     name: traffic-agent
     namespace: default
     labels:
       product: aes
   ---
   ## After creating the ServiceAccount, create a service-account-token for traffic-agent with a matching name.
   ## Since the ambassador-injector will use this token name, it must be deterministic and not auto-generated.
   apiVersion: v1
   kind: Secret
   metadata:
     name: traffic-agent
     namespace: default
     annotations:
       kubernetes.io/service-account.name: traffic-agent
   type: kubernetes.io/service-account-token
   ---
   apiVersion: rbac.authorization.k8s.io/v1beta1
   kind: Role
   metadata:
     name: traffic-agent
   rules:
     - apiGroups: [""]
       resources: [ "namespaces", "services", "secrets" ]
       verbs: ["get", "list", "watch"]
     - apiGroups: [ "getambassador.io" ]
       resources: [ "*" ]
       verbs: ["get", "list", "watch", "update"]
   ---
   apiVersion: rbac.authorization.k8s.io/v1beta1
   kind: RoleBinding
   metadata:
     name: traffic-agent
     labels:
       product: aes
   roleRef:
     apiGroup: rbac.authorization.k8s.io
     kind: Role
     name: traffic-agent
   subjects:
     - name: traffic-agent
       namespace: default
       kind: ServiceAccount
   ```

- __Give permission to all `ServiceAccount`s in the Cluster__

   Alternatively, if you already have specific `ServiceAccount`s defined for each of your pod, you may grant all of them the additional `traffic-agent` permissions:

   ```yaml
   ---
   apiVersion: rbac.authorization.k8s.io/v1beta1
   kind: ClusterRoleBinding
   metadata:
     name: traffic-agent
     labels:
       product: aes
   roleRef:
     apiGroup: rbac.authorization.k8s.io
     kind: ClusterRole
     name: traffic-agent
   subjects:
     - name: system:serviceaccounts
       kind: Group
       apiGroup: rbac.authorization.k8s.io
   ``` 

#### Automatic Traffic Agent Sidecar Injection with Ambassador Injector

The Ambassador Injector automatically injects the Traffic Agent sidecar into services that you want to use Service Preview with.

It does this with a [Mutating Admission Webhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook) that runs when pods are created in the cluster.

The Ambassador Injector can be installed in your cluster in the Ambassador namespace from this YAML manifest: [https://getambassador.io/yaml/ambassador-injector.yaml](/yaml/ambassador-injector.yaml).

This works well for most usecase but there are a couple of important points to make sure the Ambassador Injector is able to function properly.

- TLS certificates are required for the Ambassador Injector to authenticate with Kubernetes. The `ambassador-injector.yaml` provides some default certificates that can be used in development environments but this should be replaced when running in production.

- The port the container is listening on must be defined in the Pod template. The Injector will automatically detect container ports with the name `http` or `https` and use those ports to know how to route to the container.

   ```yaml
   spec:
     containers:                   # Application container
       - name: hello
         image: docker.io/datawire/hello-world:latest
         ports:
           - name: http
             containerPort: 8000   # Application port
   ```

Take a look at the following for a more detailed look at what is included in [https://getambassador.io/yaml/ambassador-injector.yaml](/yaml/ambassador-injector.yaml):

```yaml
# This is ambassador-injector.yaml
---
kind: Secret
apiVersion: v1
metadata:
  name: ambassador-injector-tls
  namespace: ambassador
type: Opaque
data:
  crt.pem: $CRT_PEM_BASE64$
  key.pem: $KEY_PEM_BASE64$
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ambassador-injector
  namespace: ambassador
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: ambassador-injector
      app.kubernetes.io/instance: ambassador
  template:
    metadata:
      labels:
        app.kubernetes.io/name: ambassador-injector
        app.kubernetes.io/instance: ambassador
    spec:
      containers:
        - name: webhook
          image: docker.io/datawire/aes:$version$
          command: [ "aes-injector" ]
          env:
            - name: TRAFFIC_AGENT_IMAGE                # Mandatory. The Traffic Agent is included in the AES image.
              value: docker.io/datawire/aes:$version$
            - name: TRAFFIC_AGENT_SERVICE_ACCOUNT_NAME # Optional. The Injector can configure the sidecar to use a specific ServiceAccount and service-account-token. if unspecified, the original Pod ServiceAccount is used.
              value: traffic-agent
            - name: TRAFFIC_AGENT_AGENT_LISTEN_PORT    # Optional. The port on which the Traffic Agent will listen. Defaults to "9900".
              value: "9900"
            - name: AGENT_MANAGER_NAMESPACE            # Optional. Namespace for contacting the Traffic Manager. Defaults to "ambassador".
              value: ambassador
          ports:
            - containerPort: 8443
              name: https
          livenessProbe:
            httpGet:
              path: /healthz
              port: https
              scheme: HTTPS
          volumeMounts:
            - mountPath: /var/run/secrets/tls
              name: tls
              readOnly: true
      volumes:
        - name: tls
          secret:
            secretName: ambassador-injector-tls
---
apiVersion: v1
kind: Service
metadata:
  name: ambassador-injector
  namespace: ambassador
spec:
  type: ClusterIP
  selector:
    app.kubernetes.io/name: ambassador-injector
    app.kubernetes.io/instance: ambassador
  ports:
    - name: ambassador-injector
      port: 443
      targetPort: https
---
apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: ambassador-injector-webhook-config
webhooks:
  - name: ambassador-injector.getambassador.io
    clientConfig:
      service:
        name: ambassador-injector
        namespace: ambassador
        path: "/traffic-agent"
      caBundle: $CA_BUNDLE_BASE64$
    failurePolicy: Ignore
    rules:
      - operations: ["CREATE"]
        apiGroups: [""]
        apiVersions: ["v1"]
        resources: ["pods"]
```

#### Manual Traffic Agent Sidecar Configuration

Each service that you want to work with Service Preview requires the Traffic Agent sidecar. This is typically managed by the Ambassador Injector.

The following is information on how to manually configure the Traffic Agent as a sidecar to your service.

You'll need to modify the YAML for each microservice to include the Traffic Agent. We'll start with a set of manifests for a simple microservice:

```yaml
# This is hello.yaml
---
apiVersion: v1
kind: Service
metadata:
  name: hello
  namespace: default
  labels:
    app: hello
spec:
  selector:
    app: hello
  ports:
    - protocol: TCP
      port: 80
      targetPort: http              # Application port
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello
  namespace: default
  labels:
    app: hello
spec:
  replicas: 1
  selector:
    matchLabels:
      app: hello
  template:
    metadata:
      labels:
        app: hello
    spec:
      containers:                   # Application container
        - name: hello
          image: docker.io/datawire/hello-world:latest
          ports:
            - name: http
              containerPort: 8000   # Application port
```

In order to run the sidecar:
- you need to include the Traffic Agent container in the microservice pod;
- you need to switch the microservice's `Service` definition to point to the Traffic Agent's listening port (using named ports such as `http` or `https` allow us to change the `Pod` definition without changing the `Service` definition); and
- you need to tell the Traffic Agent how to set up for the microservice, using environment variables.

Here is a modified set of manifests that includes the Traffic Agent (with notes below):

```yaml
# This is hello-intercept.yaml
---
apiVersion: v1
kind: Service
metadata:
  name: hello
  namespace: default
  labels:
    app: hello
spec:
  selector:
    app: hello
  ports:
    - protocol: TCP
      port: 80
      targetPort: http              # Traffic Agent listen port (note 1)
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello
  namespace: default
  labels:
    app: hello
spec:
  replicas: 1
  selector:
    matchLabels:
      app: hello
  template:
    metadata:
      labels:
        app: hello
    spec:
      containers:
        - name: hello               # Application container (note 2)
          image: docker.io/datawire/hello-world:latest
          ports:
            - containerPort: 8000   # Application port
        - name: traffic-agent       # Traffic Agent container (note 3)
          image: docker.io/datawire/aes:$version$ # (note 4)
          ports:
            - name: http
              containerPort: 9900   # Traffic Agent listen port
          env:
          - name: AGENT_SERVICE     # Name to use for intercepting (note 5)
            value: hello
          - name: AGENT_PORT        # Port on which to talk to the microservice (note 6)
            value: "8000"
          - name: AGENT_MANAGER_NAMESPACE # Namespace for contacting the Traffic Manager (note 7)
            value: ambassador
          - name: AMBASSADOR_NAMESPACE # Namespace in which this microservice is running (note 8)
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
          - name: AMBASSADOR_SINGLE_NAMESPACE # Traffic Agent container can run in a single-namespace (note 9)
            value: "true"
          - name: AGENT_LISTEN_PORT # Port on which to listen for connections (note 10)
            value: "9900"
      serviceAccountName: traffic-agent # The pod runs with traffic-agent RBAC
```

Key points include:

- **Note 1**: The `Service` now points to the Traffic Agent’s listen port (named `http`, 9900) instead of the application’s port (8000).
- **Note 2**: The microservice's application container is actually unchanged.
- **Note 3**: The Traffic Agent's container has been added.
- **Note 4**: The Traffic Agent is included in the AES image.
- **Note 5**: The `AGENT_SERVICE` environment variable is mandatory. It sets the name that the Traffic Agent will report to the Traffic Manager for this microservice: you will have to provide this name to intercept this microservice.
- **Note 6**: The `AGENT_PORT` environment variable is mandatory. It tells the Traffic Agent the local port on which the microservice is listening.
- **Note 7**: The `AGENT_MANAGER_NAMESPACE` environment variable tells the Traffic Agent the namespace in which it will be able to find the Traffic Manager. If not present, it defaults to the `ambassador` namespace.
- **Note 8**: The `AMBASSADOR_NAMESPACE` environment variable is mandatory. It lets the Traffic Agent tell the Traffic Manager the namespace in which the microservice is running. 
- **Note 9**: The `AMBASSADOR_SINGLE_NAMESPACE` environment variable tells the Traffic Agent to watch resources only in its current namespace. This allows the `traffic-agent` `ServiceAccount` to only have `Role` permissions instead of a cluster-wide `ClusterRole`.
- **Note 10**: The `AGENT_LISTEN_PORT` environment variable tells the Traffic Agent the port on which to listen for incoming connections. The `Service` must point to this port (see Note 1). If not present, it defaults to port 9900.

#### TLS Support

If other microservices in the cluster expect to speak TLS to this microservice, tell the Traffic Agent to terminate TLS:
- Set the `getambassador.io/inject-terminating-tls-secret` pod annotation, or the `AGENT_TLS_TERM_SECRET` environment variable if injecting the sidecar manually, to the name of a Kubernetes Secret that contains a TLS certificate
- The Traffic Agent will terminate TLS on its listen port (named `https` instead of `http`; 9900 by default) using the named certificate
- The Traffic Agent will not accept cleartext communication when configured to terminate TLS

If this microservice expects incoming requests to speak TLS, tell the Traffic Agent to originate TLS:
- Set the `getambassador.io/inject-originating-tls-secret` pod annotation, or the `AGENT_TLS_ORIG_SECRET` environment variable if injecting the sidecar manually, to the name of a Kubernetes Secret that contains a TLS certificate
- The Traffic Agent will use that certificate originate HTTPS requests to the application

### Ambassador Edge Stack

To enable Preview URLs, you must first enable preview URL processing in one or more Host resources. Ambassador Edge Stack uses Host resources to configure various aspects of a given host. Enabling preview URLs is as simple as adding the `previewUrl` section and setting `enabled` to `true`:

```yaml
# This is minimal-host-preview-url.yaml
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: minimal-host
spec:
  hostname: host.example.com
  acmeProvider:
    email: julian@example.com
  previewUrl:
    enabled: true
    type: Path
```

**Note**: If you already had an active Edge Control Daemon connection to the cluster, you must reconnect to the cluster for the Edge Control Daemon to detect the change to the Host resource. This limitation will be removed in the future.

## What's Next?

See how [Edge Control commands can be used in action](../service-preview-tutorial) to establish outbound connectivity with a remote Kubernetes cluster and intercept inbound requests.
