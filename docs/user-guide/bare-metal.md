# Deploying Ambassador on a Bare Metal Kubernetes Installation
---
In cloud environments, provisioning a readily available network load balancer with Ambassador is the best option for handling ingress into your Kubernetes cluster. When running kubernetes on a bare-metal setup, where network load balancers are not available by default, we need to consider different options for exposing Ambassador. 
##Exposing Ambassador via NodePort
The simplest way to expose an application in Kubernetes is via a `NodePort` service. In this configuration, we create the [Ambassador service](/user-guide/getting-started#2-defining-the-ambassador-service) and identify `type: NodePort` instead of `LoadBalancer`. Kubernetes will then create a service and assign that service a port to be exposed externally and direct traffic to Ambassador via the defined `port`. 

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
    targetPort: 80
    nodePort: 30036  # Optional: Define the port you would like exposed
    protocol: TCP
  selector:
    service: ambassador
```
Using a `NodePort` leaves Ambassador isolated from the host network, allowing the Kubernetes service to handle routing to Ambassador pods. You can drop-in this yaml to replace the `LoadBalancer` service in the [YAML installation guide](/user-guide/getting-started) and use `http://<External-Node-IP>:<NodePort>/` as the host for requests. 

## Exposing Ambassador via Host Network
When running Ambassador on a bare-metal install of Kubernetes, you have the option to configure Ambassador pods to use the network of the host they are running on. This method allows you to bind Ambassador directly to port 80 or 443 so you won't need to identify the port in requests. 

i.e `http://<External-Node-IP>:<NodePort>/` becomes `http://<External-Node-IP>/` 

This can be configured by setting `hostNetwork: true` in the Ambassador deployment. `dnsPolicy: ClusterFirstWithHostNet` will also need to set to tell Ambassador to use *KubeDNS* when attempting to resolve mappings. 

```diff
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: ambassador
spec:
  replicas: 1
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
        image: quay.io/datawire/ambassador:0.40.0
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
This configuration does not require an Ambassador service be defined so you can remove that service if you have defined one. 

**Note:** Before configuring Ambassador with this method, consider some of the functionality that is lost by bypassing the Kubernetes service including only having one Ambassador able to bind to port 80 or 443 per node and losing any load balancing that is typically performed by Kubernetes services. Join our [slack channel](https://join.slack.com/t/datawire-oss/shared_invite/enQtMzcwMDEwMTc5ODQ3LTE1NmIzZTFmZWE0OTQ1NDc2MzE2NTkzMDAzZWM0MDIxZTVjOGIxYmRjZjY3N2M2Mjk4NGI5Y2Q4NGY4Njc1Yjg) to ask any questions you have regarding running Ambassador on a bare-metal installation.
