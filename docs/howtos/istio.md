# Ambassador and Istio

Ambassador Edge Stack and Istio: Edge Proxy and Service Mesh together in one. The Edge Stack is deployed at the edge of your network and routes incoming traffic to your internal services (aka "north-south" traffic). [Istio](https://istio.io/) is a service mesh for microservices, and is designed to add application-level Layer (L7) observability, routing, and resilience to service-to-service traffic (aka "east-west" traffic). Both Istio and the Ambassador Edge Stack are built using [Envoy](https://www.envoyproxy.io).

Ambassador Edge Stack and Istio can be deployed together on Kubernetes. In this configuration, incoming traffic from outside the cluster is first routed through the Ambassador Edge Stack, which then routes the traffic to Istio-powered services. The Ambassador Edge Stack handles authentication, edge routing, TLS termination, and other traditional edge functions.

This allows the operator to have the best of both worlds: a high performance, modern edge service (Ambassador Edge Stack) combined with a state-of-the-art service mesh (Istio). While Istio has introduced a [Gateway](https://istio.io/docs/tasks/traffic-management/ingress/#configuring-ingress-using-an-istio-gateway) abstraction, the Ambassador Edge Stack still has a much broader feature set for edge routing than Istio. For more on this topic, see our blog post on [API Gateway vs Service Mesh](https://blog.getambassador.io/api-gateway-vs-service-mesh-104c01fa4784).

This guide will explain how to take advantage of both Ambassador and Istio to have complete control and observability over how requests are made in your cluster. 

## Prerequisites

- A Kubernetes cluster version 1.15 and above
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

## Install Istio

[Istio installation](https://istio.io/docs/setup/getting-started/) is outside of the scope of this document. Ambassador will integrate with any version of Istio from any installation method.

## Install Ambassador

There a number of installation options for Ambassador. See the [getting started](../../tutorials/getting-started) for the full list of installation options and instructions.

## Integrate Ambassador and Istio

Ambassador integrates with Istio in three ways:

* Uses Istio [mutual TLS (mTLS)](#mutual-tls) certificates for end-to-end encryption
* Integrates with Prometheus for centralized metrics collection
* Integrates with Istio distributed tracing for end-to-end observability

Integrating Ambassador and Istio allows you to take advantage of the edge routing capabilities of Ambassador while maintaining the end-to-end security and observability that makes Istio so powerful.

### Mutual TLS

The process of collecting mTLS certificates is different depending on your Istio version. Select your Istio version below for instructions on how to integrate Ambassador with Istio.

- [Istio 1.5 and above](#integrating-ambassador-with-istio-15-and-above)
- [Istio 1.4 and below](#integrating-ambassador-with-istio-14-and-below)

#### Integrating Ambassador with Istio 1.5 and Above

Istio 1.5 introduced [istiod](https://istio.io/docs/ops/deployment/architecture/#istiod) which moved Istio towards a single control plane process.

Below we will update the deployment of Ambassador to add the `istio-proxy` sidecar so we can get Istio's mTLS certificates straight from the Istio control plane.

```diff
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    product: aes
  name: ambassador
  namespace: ambassador
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
        app.kubernetes.io/managed-by: getambassador.io
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
      - name: aes
        image: docker.io/datawire/aes:$version$
        imagePullPolicy: Always
        env:
        - name: AMBASSADOR_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: REDIS_URL
          value: ambassador-redis:6379
        - name: AMBASSADOR_URL
          value: https://ambassador.ambassador.svc.cluster.local
        - name: POLL_EVERY_SECS
          value: '60'
        - name: AMBASSADOR_INTERNAL_URL
          value: https://127.0.0.1:8443
        - name: AMBASSADOR_ADMIN_URL
          value: http://127.0.0.1:8877
        - name: AMBASSADOR_SINGLE_NAMESPACE
          value: ''
        # Necessary to run the istio-proxy sidecar
        - name: AMBASSADOR_ENVOY_BASE_ID
          value: "1"
        livenessProbe:
          httpGet:
            path: /ambassador/v0/check_alive
            port: 8877
          periodSeconds: 3
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
        resources:
          limits:
            cpu: 1000m
            memory: 600Mi
          requests:
            cpu: 200m
            memory: 300Mi
        securityContext:
          allowPrivilegeEscalation: false
        volumeMounts:
        - mountPath: /tmp/ambassador-pod-info
          name: ambassador-pod-info
        - mountPath: /.config/ambassador
          name: ambassador-edge-stack-secrets
          readOnly: true
        - mountPath: /etc/istio-certs/
          name: istio-certs
      - name: istio-proxy
        # Use the same version as your Istio installation
        image: istio/proxyv2:{{ISTIO_VERSION}}
        args:
        - proxy
        - sidecar
        - --domain
        - $(POD_NAMESPACE).svc.cluster.local
        - --serviceCluster
        - istio-proxy-ambassador
        - --discoveryAddress
        - istio-pilot.istio-system.svc:15012
        - --connectTimeout
        - 10s
        - --statusPort
        - "15020"
        - --trust-domain=cluster.local
        - --controlPlaneBootstrap=false
        env:
        - name: OUTPUT_CERTS
          value: "/etc/istio-certs"
        - name: JWT_POLICY
          value: third-party-jwt
        - name: PILOT_CERT_PROVIDER
          value: istiod
        - name: CA_ADDR
          value: istiod.istio-system.svc:15012
        - name: ISTIO_META_MESH_ID
          value: cluster.local
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: INSTANCE_IP
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
        - name: SERVICE_ACCOUNT
          valueFrom:
            fieldRef:
              fieldPath: spec.serviceAccountName
        - name: HOST_IP
          valueFrom:
            fieldRef:
              fieldPath: status.hostIP
        - name: ISTIO_META_POD_NAME
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: metadata.name
        - name: ISTIO_META_CONFIG_NAMESPACE
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: metadata.namespace
        imagePullPolicy: IfNotPresent
        readinessProbe:
          failureThreshold: 30
          httpGet:
            path: /healthz/ready
            port: 15020
            scheme: HTTP
          initialDelaySeconds: 1
          periodSeconds: 2
          successThreshold: 1
          timeoutSeconds: 1
        volumeMounts:
        - mountPath: /var/run/secrets/istio
          name: istiod-ca-cert
        - mountPath: /etc/istio/proxy
          name: istio-envoy
        - mountPath: /etc/istio-certs/
          name: istio-certs
        - mountPath: /var/run/secrets/tokens
          name: istio-token
        securityContext:
          runAsUser: 0
      volumes:
      - name: istio-certs
        emptyDir:
          medium: Memory
      - name: istiod-ca-cert
        configMap:
          defaultMode: 420
          name: istio-ca-root-cert
      - emptyDir:
          medium: Memory
        name: istio-envoy
      - name: istio-token
        projected:
          defaultMode: 420
          sources:
          - serviceAccountToken:
              audience: istio-ca
              expirationSeconds: 43200
              path: istio-token
      - downwardAPI:
          items:
          - fieldRef:
              fieldPath: metadata.labels
            path: labels
        name: ambassador-pod-info
      - name: ambassador-edge-stack-secrets
        secret:
          secretName: ambassador-edge-stack
      restartPolicy: Always
      securityContext:
        runAsUser: 8888
      serviceAccountName: ambassador
      terminationGracePeriodSeconds: 0
```

**Make sure the `istio-proxy` is the same version as your Istio installation**

Deploy the YAML above with `kubectl apply` to install Ambassador with the `istio-proxy` sidecar.

After applying the updated Ambassador deployment above to your cluster, we need to stage the Istio mTLS certificates for use.

We do this with a `TLSContext` that loads the mTLS certificates from the `istio-proxy` for use when sending requests upstream.

```bash
$ kubectl apply -f - <<EOF
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: istio-upstream
  namespace: ambassador
spec:
  cert_chain_file: /etc/istio-certs/cert-chain.pem
  private_key_file: /etc/istio-certs/key.pem
  cacert_chain_file: /etc/istio-certs/root-cert.pem
  alpn_protocols: istio
EOF
```

Ambassador is now integrated with Istio for end-to-end encryption.

#### Integrating Ambassador with Istio 1.4 and Below

With Istio 1.4 and below, Istio stores it's mTLS certificates as a Kubernetes `Secret` in each namespace.

We can read these certificates from the `istio.default` `Secret` in the Ambassador namespace with a `TLSContext`.

```bash
$ kubectl apply -f - <<EOF
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: istio-upstream
  namespace: ambassador
spec:
  secret: istio.default
  secret_namespacing: false
  alpn_protocols: istio
EOF
```

Ambassador is now integrated with Istio for end-to-end encryption.

### Integrating Prometheus Metrics Collection

Istio installs by default with a Prometheus deployment for collecting metrics from different resources in your cluster. 

We can integrate Ambassador with the same Prometheus to give us a single metrics endpoint.

Istio's Prometheus deployment is configured using a `ConfigMap`. To add Ambassador as a Metrics endpoint, we need to update this `ConfigMap` and restart Prometheus.

1. Export the current Prometheus configuration.

   * If you installed Istio with `istioctl`, you can get the YAML that was installed with

      ```bash
      istioctl manifest generate > istio.yaml
      ```
      This will export all of the YAML configuration used by Istio to a file named `istio.yaml` so you can update the `ConfigMap` there.

   * If you did not install with `istioctl`, you can export the YAML of the current `ConfigMap` in your cluster with `kubectl`:

      ```bash
      kubectl get -n istio-system configmap prometheus  -o yaml > prom-cm.yaml
      ```
   
      You should now have a `ConfigMap` that looks something like this:
   
      ```yaml
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: prometheus
        namespace: istio-system
        labels:
          app: prometheus
          release: istio
      data:
        prometheus.yml: |-
          global:
            scrape_interval: 15s
          scrape_configs:
          ...
      ```

2. Update the `Prometheus` `ConfigMap` to add Ambassador as a scraping endpoint

   To configure Prometheus to get metrics from Ambassador, add the following config under `scrape_configs` in the `Prometheus` `ConfigMap` we exported above and apply it with `kubectl`.

   ```yaml
   ...
   data:
     prometheus.yml: |-
       global:
         scrape_interval: 15s
       scrape_configs:
       # Ambassador scraping.
       - job_name: 'ambassador'
         kubernetes_sd_configs:
         - role: endpoints
           namespaces:
             names:
             - ambassador
         relabel_configs:
         - source_labels: [__meta_kubernetes_service_name,    __meta_kubernetes_endpoint_port_name]
           action: keep
           regex: ambassador-admin;ambassador-admin
         - source_labels: [__meta_kubernetes_endpoint_address_target_name]
           action: replace
           target_label: pod_name
       
       # Mixer scrapping. Defaults to Prometheus and mixer on same namespace.
       ...
   ```

3. Restart and access the Prometheus UI

   The Prometheus pod must be restarted to start with the new configuration. 

   ```bash
   kubectl delete pod -n istio-system $(kubectl get pods -n istio-system | grep prometheus | awk '{print $1}')
   ```

   After the pod restarts you can port-forward the Prometheus `Service` to access the Prometheus UI.

   ```bash
   kubectl port-forward -n istio-system svc/prometheus 9090:9090
   ```

   You can now access the UI at http://localhost:9090/

### Integrating Distributed Tracing

Enabling the [tracing component](https://istio.io/docs/tasks/observability/distributed-tracing/overview/) in Istio gives you the power to observe how a request behaves at each point in your application.

Since Istio will propagate the tracing headers automatically, integrating Ambassador with the Istio Jaeger deployment will give you end-to-end observability of requests throughout your cluster.

To do so, simply create a [`TracingService`](../../topics/running/services/tracing-service) and point it at the `zipkin` `Service` in the istio-system namespace.

```yaml
---
apiVersion: getambassador.io/v2
kind:  TracingService
metadata:
  name:  tracing
  namespace: ambassador
spec:
  service: "zipkin.istio-system:9411"
  driver: zipkin
  config: {}
  tag_headers:
    - ":authority"
    - ":path"
```

After applying this configuration with `kubectl apply`, restart the Ambassador pod for the configuration to take effect.

```bash
kubectl delete po -n ambassador {{AMBASSADOR_POD_NAME}}
```

You can now access the tracing service UI to see Ambassador is now one of the services.

## Routing to Services

Above, we integrated Ambassador with Istio to take advantage of end-to-end encryption and observability offered by Istio while leveraging the feature-rich edge routing capabilities of Ambassador.

Now we will show how you can use Ambassador to route to services in the Istio service mesh.

1. Label the default namespace for [automatic sidecar injection](https://istio.io/docs/setup/additional-setup/sidecar-injection/#automatic-sidecar-injection)

   ```bash
   kubectl label namespace default istio-injection=enabled
   ```

   This will tell Istio to automatically inject the `istio-proxy` sidecar container into pods in this namespace.

2. Install the quote example service

   ```bash
   kubectl apply -n default -f https://getambassador.io/yaml/backends/quote.yaml
   ```

   Wait for the pod to start and see that there are two containers: the `quote` application and the `istio-proxy` sidecar.

   ```bash
   $ kubectl get po -n default 

   NAME                     READY   STATUS    RESTARTS   AGE
   quote-6bc6b6bd5d-jctbh   2/2     Running   0          91m
   ```

3. Route to the service

   Traffic routing in Ambassador is configured with the [`Mapping`](../../topics/using/intro-mappings) resource. This is a powerful configuration object that lets you configure different routing rules for different services. 

   The above `kubectl apply` installed the following basic `Mapping` which has configured Ambassador to route traffic with URL prefix `/backend/` to the `quote` service.

   ```yaml
   apiVersion: getambassador.io/v2
   kind: Mapping
   metadata:
     name: quote-backend
   spec:
     prefix: /backend/
     service: quote
   ```

   Since we have integrated Ambassador with Istio, we can tell it to use the mTLS certificates to encrypt requests to the quote service.

   Simply do that by updating the above `Mapping` with the following one.

   ```bash
   $ kubectl apply -f - <<EOF
   apiVersion: getambassador.io/v2
   kind: Mapping
   metadata:
     name: quote-backend
   spec:
     prefix: /backend/
     service: quote
     tls: istio-upstream
   EOF
   ```

   Send a request to the quote service using curl:

   ```bash
   $ curl -k https://{{AMBASSADOR_HOST}}/backend/

   {
       "server": "bewitched-acai-5jq7q81r",
       "quote": "A late night does not make any sense.",
       "time": "2020-06-02T10:48:45.211178139Z"
   }
   ```

   While the majority of the work being done is transparent to the user, you have successfully sent a request to Ambassador which routed it to the quote service in the default namespace. It was then intercepted by the `istio-proxy` which authenticated the request from Ambassador and exported various metrics and finally forwarded it on to the  quote service.

## Enforcing Authentication Between Containers

Istio defaults to PERMISSIVE mTLS that does not require authentication between containers in the cluster. Configuring STRICT mTLS will require all connections within the cluster be encrypted.

1. Configure Istio in [STRICT mTLS](https://istio.io/docs/tasks/security/authentication/authn-policy/#globally-enabling-istio-mutual-tls-in-strict-mode) mode.

   ```bash
   $ kubectl apply -f - <<EOF
   apiVersion: security.istio.io/v1beta1
   kind: PeerAuthentication
   metadata:
     name: default
     namespace: istio-system
   spec:
     mtls:
       mode: STRICT   
   EOF
   ```

   This will enforce authentication for all containers in the mesh.

   We can test this by removing the `tls` configuration from the quote-backend `Mapping`
   and sending a request.

   ```bash
   $ kubectl apply -f - <<EOF
   apiVersion: getambassador.io/v2
   kind: Mapping
   metadata:
     name: quote-backend
   spec:
     prefix: /backend/
     service: quote
   EOF
   ```

   ```
   $ curl -k https://{{AMBASSADOR_HOST}}/backend/
   
   upstream connect error or disconnect/reset before headers. reset reason: connection termination
   ```


2. Configure Ambassador to use mTLS certificates

   As we have demonstrated above we can tell Ambassador to use the mTLS certificates from Istio to authenticate with the `istio-proxy` in the quote pod.

   ```bash
   $ kubectl apply -f - <<EOF
   ---
   apiVersion: getambassador.io/v2
   kind: Mapping
   metadata:
     name: quote-backend
   spec:
     prefix: /backend/
     service: quote
     tls: istio-upstream
   EOF
   ```

   Now Ambassador will use the Istio mTLS certificates when routing to the `quote` service. 

   ```bash
   $ curl -k https://{{AMBASSADOR_HOST}}/backend/
   {
       "server": "bewitched-acai-5jq7q81r",
       "quote": "Non-locality is the driver of truth. By summoning, we vibrate.",
       "time": "2020-06-02T11:06:53.854468941Z"
   }
   ```

## Grafana

The Istio [Grafana addon](https://istio.io/docs/tasks/observability/metrics/using-istio-dashboard/) integrates a Grafana dashboard with the Istio Prometheus deployment to visualize the metrics collected there.

The metrics Ambassador adds to the list will appear in the Istio dashboard but we can add an Ambassador dashboard as well. We're going to use the Ambassador dashboard on [Grafana's](https://grafana.com/) website under entry [4689](https://grafana.com/dashboards/4698) as a starting point.

First, let's start the port-forwarding for Istio's Grafana service:

```sh
$ kubectl -n istio-system port-forward $(kubectl -n istio-system get pod -l app=grafana -o jsonpath='{.items[0].metadata.name}') 3000:3000 &
```

Now, open Grafana tool by accessing: `http://localhost:3000/`

To install the Ambassador Edge Stack Dashboard:

* Click on Create
* Select Import
* Enter number 4698

Now we need to adjust the Dashboard Port to reflect our Ambassador Edge Stack configuration:

* Open the Imported Dashboard
* Click on Settings in the Top Right corner
* Click on Variables
* Change the port to 80 (according to the ambassador service port)

Next, adjust the Dashboard Registered Services metric:

* Open the Imported Dashboard
* Find Registered Services
* Click on the down arrow and select Edit
* Change the Metric to:

```yaml
envoy_cluster_manager_active_clusters{job="ambassador"}
```

Now let's save the changes:

* Click on Save Dashboard in the Top Right corner

## FAQ

### How to Test Istio Certificate Rotation

Istio mTLS certificates, by default, will be valid for a max of 90 days but will be rotated every day.

Ambassador will watch and update the mTLS certificates as they rotate so you will not need to worry about certificate expiration. 

To test that Ambassador is properly rotating certificates you can shorten the TTL of the Istio certificates so you can verify that Ambassador is using the new certificates.

In **Istio 1.5 and above**, you can configure that by setting the following environment variables in the `istiod` container.

```yaml
env:
- name: DEFAULT_WORKLOAD_CERT_TTL
  value: 30m
- name: MAX_WORKLOAD_CERT_TTL
  value: 1h
```

In **Istio 1.4 and below**, you can configure this by passing the following arguments to the `istio-citadel` container

```yaml
      containers:
      - name: citadel
        ...
        args:
          - --workload-cert-ttl=1h # Lifetime of certificates issued to workloads in Kubernetes.
          - --max-workload-cert-ttl=48h # Maximum lifetime of certificates issued to workloads by Citadel.
```

This will make the certificates Istio issues expire in one hour so testing certificate rotation is much easier.
