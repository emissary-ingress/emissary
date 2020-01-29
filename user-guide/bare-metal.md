# Install with Bare Metal

In cloud environments, provisioning a readily available network load balancer with Ambassador is the best option for handling ingress into your Kubernetes cluster. When running Kubernetes on a bare metal setup, where network load balancers are not available by default, we need to consider different options for exposing Ambassador. 

## Exposing Ambassador via NodePort

The simplest way to expose an application in Kubernetes is via a `NodePort` service. In this configuration, we create the [Ambassador service](../getting-started#2-defining-the-ambassador-service) and identify `type: NodePort` instead of `LoadBalancer`. Kubernetes will then create a service and assign that service a port to be exposed externally and direct traffic to Ambassador via the defined `port`.

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: ambassador
spec:
  type: NodePort
  ports:
  - name: http
    port: 8088
    targetPort: 8080
    nodePort: 30036  # Optional: Define the port you would like exposed
    protocol: TCP
  selector:
    service: ambassador
```

Using a `NodePort` leaves Ambassador isolated from the host network, allowing the Kubernetes service to handle routing to Ambassador pods. You can drop-in this YAML to replace the `LoadBalancer` service in the [YAML installation guide](../getting-started) and use `http://<External-Node-IP>:<NodePort>/` as the host for requests.

## Exposing Ambassador via Host Network

When running Ambassador on a bare metal install of Kubernetes, you have the option to configure Ambassador pods to use the network of the host they are running on. This method allows you to bind Ambassador directly to port 80 or 443 so you won't need to identify the port in requests. 

i.e `http://<External-Node-IP>:<NodePort>/` becomes `http://<External-Node-IP>/`

This can be configured by setting `hostNetwork: true` in the Ambassador deployment. `dnsPolicy: ClusterFirstWithHostNet` will also need to set to tell Ambassador to use *KubeDNS* when attempting to resolve mappings.

```diff
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ambassador
spec:
  replicas: 1
  selector:
    matchLabels:
      service: ambassador
  template:
    metadata:
      annotations:
        sidecar.istio.io/inject: "false"
      labels:
        service: ambassador
    spec:
+     hostNetwork: true
+     dnsPolicy: ClusterFirstWithHostNet
      serviceAccountName: ambassador
      containers:
      - name: ambassador
        image: quay.io/datawire/ambassador:$version$
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
```

This configuration does not require a defined Ambassador service, so you can remove that service if you have defined one.

**Note:** Before configuring Ambassador with this method, consider some of the functionality that is lost by bypassing the Kubernetes service including only having one Ambassador able to bind to port 8080 or 8443 per node and losing any load balancing that is typically performed by Kubernetes services.

Join our [Slack channel](https://d6e.co/slack) to ask any questions you have regarding running Ambassador on a bare metal installation.
