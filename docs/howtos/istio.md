# Ambassador and Istio

The Ambassador Edge Stack is a feature-rich ingress controller than easily handles getting requests from your clients to your backend services. It exposes powerful security and access control functionality that makes it easy to control and observe which clients and accessing your services and how they are doing so.

Istio, is a feature-rich service mesh that gives you fine-grained control and observability over requests that travel from service-to-service in your cluster.

This guide will explain how to take advantage of both Ambassador and Istio to have complete control and observability over how requests are made in your cluster. 

## Prerequisites

- A Kubernetes cluster version 1.15 and above
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

## Install Istio

[Istio installation](https://istio.io/docs/setup/getting-started/) is outside of the scope of this document. Ambassador will integrate with any version of Istio from any installation method.

## Install Ambassador

Select your Istio version below for instructions on how to install Ambassador.

- [Istio 1.5 and above](#istio-1.5-and-above)
- [Istio 1.4 and below](#istio-1.4-and-below)

### Istio 1.5 and Above

Istio 1.5 introduced [istiod](https://istio.io/docs/ops/deployment/architecture/#istiod) which moved Istio away from a microservice architecture and towards a single control plane process. 

Due to this change, acquiring mTLS certificates relies on the presence of an `istio-proxy` to get them from the control plane. Because we do not want this proxy to handle route, we will manually add it to our Ambassador `Deployment`.

Below is the standard AES deployment YAML found at https://getambassador.io/yaml/aes.yaml with the `istio-proxy` sidecar added:

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
+        - mountPath: /etc/istio-certs/
+          name: istio-certs
+      - name: istio-proxy
+        # Use the same version as your Istio installation
+        image: istio/proxyv2:{{ISTIO_VERSION}}
+        args:
+        - proxy
+        - sidecar
+        - --domain
+        - $(POD_NAMESPACE).svc.cluster.local
+        - --serviceCluster
+        - istio-proxy-ambassador
+        - --discoveryAddress
+        - istio-pilot.istio-system.svc:15012
+        - --connectTimeout
+        - 10s
+        - --statusPort
+        - "15020"
+        - --trust-domain=cluster.local
+        - --controlPlaneBootstrap=false
+        env:
+        - name: OUTPUT_CERTS
+          value: "/etc/istio-certs"
+        - name: JWT_POLICY
+          value: third-party-jwt
+        - name: PILOT_CERT_PROVIDER
+          value: istiod
+        - name: CA_ADDR
+          value: istiod.istio-system.svc:15012
+        - name: ISTIO_META_MESH_ID
+          value: cluster.local
+        - name: POD_NAME
+          valueFrom:
+            fieldRef:
+              fieldPath: metadata.name
+        - name: POD_NAMESPACE
+          valueFrom:
+            fieldRef:
+              fieldPath: metadata.namespace
+        - name: INSTANCE_IP
+          valueFrom:
+            fieldRef:
+              fieldPath: status.podIP
+        - name: SERVICE_ACCOUNT
+          valueFrom:
+            fieldRef:
+              fieldPath: spec.serviceAccountName
+        - name: HOST_IP
+          valueFrom:
+            fieldRef:
+              fieldPath: status.hostIP
+        - name: ISTIO_META_POD_NAME
+          valueFrom:
+            fieldRef:
+              apiVersion: v1
+              fieldPath: metadata.name
+        - name: ISTIO_META_CONFIG_NAMESPACE
+          valueFrom:
+            fieldRef:
+              apiVersion: v1
+              fieldPath: metadata.namespace
+        imagePullPolicy: IfNotPresent
+        readinessProbe:
+          failureThreshold: 30
+          httpGet:
+            path: /healthz/ready
+            port: 15020
+            scheme: HTTP
+          initialDelaySeconds: 1
+          periodSeconds: 2
+          successThreshold: 1
+          timeoutSeconds: 1
+        volumeMounts:
+        - mountPath: /var/run/secrets/istio
+          name: istiod-ca-cert
+        - mountPath: /etc/istio/proxy
+          name: istio-envoy
+        - mountPath: /etc/istio-certs/
+          name: istio-certs
+        - mountPath: /var/run/secrets/tokens
+          name: istio-token
+        securityContext:
+          runAsUser: 0
      volumes:
+      - name: istio-certs
+        emptyDir:
+          medium: Memory
+      - name: istiod-ca-cert
+        configMap:
+          defaultMode: 420
+          name: istio-ca-root-cert
+      - emptyDir:
+          medium: Memory
+        name: istio-envoy
+      - name: istio-token
+        projected:
+          defaultMode: 420
+          sources:
+          - serviceAccountToken:
+              audience: istio-ca
+              expirationSeconds: 43200
+              path: istio-token
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

Adding the `istio-proxy` container and volumes will allow Ambassador to get the mTLS certificates from Istio for sending requests upstream. 

**Make sure the `istio-proxy` is the same version as your Istio installation**

Deploy the YAML above with `kubectl apply` to install Ambassador with the `istio-proxy` sidecar.

After installing Ambassador, we need to stage the Istio mTLS certificates for use.

Simply create a `TLSContext` object to load the TLS certificates the `istio-proxy` got from Istio for Ambassador to use on requests upstream.

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

You now have Ambassador installed and staged to do mTLS with upstream services.

### Istio 1.4 and Below

There is no change in how you install Ambassador when running Istio 1.4 and below. See the [getting started](../tutorials/getting-started) page to install Ambassador.

After installing Ambassador, we need to stage the Istio mTLS certificates for use.

Simply create a `TLSContext` object to load the `istio.default` secret from the Ambassador namespace for Ambassador to use on requests upstream.

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

You now have Ambassador installed and staged to do mTLS with upstream services.

## Routing to Services

Now we will have Ambassador route to services in our Kubernetes cluster.

1. Install the [bookinfo sample application](https://istio.io/docs/examples/bookinfo/)

2. 



Now we will show how you can use Ambassador to route to services in the Istio service mesh.

1. Label the default namespace for [automatic sidecar injection](https://istio.io/docs/setup/additional-setup/sidecar-injection/#automatic-sidecar-injection)

   ```
   kubectl label namespace default istio-injection=enabled
   ```

   This will tell Istio to automatically inject the `istio-proxy` sidecar container into pods in this namespace.



2. Install the quote example service:

   ```
   kubectl apply -n default -f https://getambassador.io/yaml/backends/quote.yaml
   ```

   Wait for the pod to start and see that there are two containers: the `quote` application and the `istio-proxy` sidecar.

3. Send a request to the service

   The above `kubectl apply` installed the following `Mapping` which configured Ambassador to route traffic with URL prefix `/backend/` to the `quote` service.

   ```yaml
   apiVersion: getambassador.io/v2
   kind: Mapping
   metadata:
     name: quote-backend
   spec:
     prefix: /backend/
     service: quote
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

   While you may not be able to tell, Ambassador received the above request and forwarded it on to the quote service where the `istio-proxy` intercepted it and forwarded it on to the application.

## Mutual TLS (mTLS)

Istio defaults to PERMISSIVE mTLS that does not require authentication between containers in the cluster. Configuring STRICT mTLS will require all connections within the cluster be encrypted. We have already staged Ambassador with the necessary TLS certificates to support this during [installation](#install-ambassador).

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

   This will enforce authentication between all containers in the mesh.

   Now, if you send a request to the quote service, you will see the request fails because we are not sending an encrypted request.

   ```bash
   $ curl -k https://{{AMBASSADOR_HOST}}/backend/
   upstream connect error or disconnect/reset before headers. reset reason: connection termination
   ```

2. Configure Ambassador to use mTLS certificates

   Since we already staged Ambassador to use the mTLS certificates above, we can simply add that `TLSContext` to the `Mapping` to the quote service.

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