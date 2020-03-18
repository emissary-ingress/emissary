# Install with Google Kubernetes Engine (GKE) Ingress 

Google offers a [L7 load balancer](https://cloud.google.com/kubernetes-engine/docs/concepts/ingress) to 
leverage network services such as managed SSL certificates, SSL offloading or the Google content delivery network. 
A L7 load balancer in front of Ambassador can be configured by hand or by using the ingress-gce resource. Using the 
ingress resource also allows you to create google managed SSL certificates through kubernetes.

With this setup, HTTPS will be terminated at the Google load balancer. The load balancer will be created and configured by 
the ingress-gce resource. The load balancer consists of a set of 
[forwarding rules](https://cloud.google.com/load-balancing/docs/forwarding-rule-concepts#https_lb) and a set of
[backend service](https://cloud.google.com/load-balancing/docs/backend-service). 
In this setup the ingress resource creates two forwarding rules, one for HTTP and one for HTTPS. The HTTPS
forwarding rule has the SSL certificates attached. In addition one backend service will be created to point to
a list of instance groups at a static port. This will be the NodePort of the Ambassador service. 

With this setup the load balancer terminates HTTPS and then directs the traffic to the Ambassador service 
via the NodePort. Ambassador is then doing all the routing to the other internal/external services. 

# Overview of steps

1. Install and configure the ingress with the HTTP(S) load balancer
2. Install Ambassador
3. Configure and connect Ambassador to ingress
4. Create an SSL certificate and enable HTTPS
5. Configure Ambassador to do HTTP -> HTTPS redirection

Ambassador will be running as NodePort service. Health checks will be configured to go to the ambassador-admin service. Ingress and Ambassador need to run in their own namespace.

## 0. Ambassador Edge Stack

This guide will install Ambassador API gateway. You can also install Ambassador Edge Stack. Please note:
- The ingress and the ambassador service need to run in the same namespace
- The Ambassador service needs to be of type `NodePort` and not `LoadBalancer`. Also remove the line with `externalTrafficPolicy: Local`
- Ambassador-Admin needs to be of type `NodePort` instead of `ClusterIP` since it needs to be available for health checks
 
## 1 . Install and configure ingress with the HTTP(S) load balancer

Create a GKE cluster through the web console. Use the release channel. When the cluster
is up and running follow [this tutorial from google](https://cloud.google.com/kubernetes-engine/docs/tutorials/http-balancer) to configure 
an ingress and a L7 load balancer. After you have completed these steps you will have a running L7 load balancer
and one services. 

## 2. Install Ambassador

Follow the first section of [installation of Ambassador API](../install/install-ambassador-oss) guide to install Ambassador API.
Stop before defining the ambassador service.

Ambassador needs to be deployed as `NodePort` instead of `LoadBalancer` to work with the L7 load balancer and the ingress.

Save the yaml below in ambassador.yaml and apply with `kubectl apply -f ambassador.yaml`
```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: ambassador
spec:
  type: NodePort
  ports:
   - port: 8080
     targetPort: 8080
  selector:
    service: ambassador
```

You will now have a ambassador service running next to your ingress.

## 3.  Configure and connect ambassador to the ingress

You need to change the ingress for it to send traffic to ambassador. Assuming you have followed the tutorial, you should
have a file named basic-ingress.yaml. Change it to point to ambassador instead of web:

```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: basic-ingress
spec:
  backend:
    serviceName: ambassador
    servicePort: 8080
```

Now let's connect the other service from the tutorial to ambassador by specifying a mapping:

```yaml
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: web
  namespace: default
spec:
  prefix: /
  service: web:8080
```

All traffic will now go to ambassador and from ambassador to the web service. You should be able to hit your load balancer and get the output. It may take some time until the load balancer infrastructure has rolled out all changes and you might see gateway errors during that time.
As a side note: right now all traffic will go to the `web` service, including the load balancer health check.

## 4. Create an SSL certificate and enable HTTPS

Read up on [managed certificates on GKE](https://cloud.google.com/kubernetes-engine/docs/how-to/managed-certs). You need
a DNS name and point it to the external IP of the load balancer. Going forward it is assumed that this DNS name
is www.example.com .

certificate.yaml:
```yaml 
apiVersion: networking.gke.io/v1beta1
kind: ManagedCertificate
metadata:
  name: www-example-com
spec:
  domains:
    - www.example.com
```

Modify the ingress from before:
```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: basic-ingress
  annotations:
    networking.gke.io/managed-certificates: www-example-com
spec:
  backend:
    serviceName: ambassador
    servicePort: 8080
```

Please wait (5-15 minutes) until the certificate is created and all edge servers have the certificates ready. 
`kubectl describe ManagedCertificate` will show you the status or go to the web console to view the load balancer.

You should now be able to access the web service via https://www.example.com

## 5. Configure Ambassador to do HTTP -> HTTPS redirection

### Redirecting the health check
The google load balancer depends on a successful health check to route traffic to your nodes. By default the
health check will send a HTTP request to `/` and expect `HTTP 200` to be returned. Otherwise the node is not healthy and if there
are no healthy nodes the load balancer will return a 500 error.

The health check definition needs to be changed. Ambassador will respond with a HTTP redirect after enabling HTTP->HTTPS redirection.
The health check would fail. This is where the `ambassador-admin` service is coming into play. It can be queried at `/ambassador/v0/check_ready`
and will return the correct HTTP 200 answer.

- determine the node port from the service ambassador-admin: `kubectl get service`. The NodePort is a port number between 30000 and 32767.
- Open the cloud console http://console.cloud.google.com and select Network Services/Load Balancing.
- Click on the load balancer
- In the Backend Section you will find a link to the health check
- Click it and edit the health check
  - Change the port to the NodePort from ambassador-admin
  - Change the Request path to `/ambassador/v0/check_ready`
  - Optional: change the check interval to their defaults (10 seconds check interval, 5 s timeout, unhealthy 3 tries)

Now the service health is determined by contacting ambassador-admin service

### Enabling HTTP -> HTTPS 

- Configure Ambassador to [redirect traffic from HTTP to HTTPS](../running/tls/cleartext-redirection/#protocol-based-redirection). 
- you need to restart Ambassador to effect the changes

The result should be that http://www.example.com will redirect to https://www.example.com. 

You can now add more services by specifying the hostname in the mapping.
