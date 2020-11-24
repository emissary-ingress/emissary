# Ambassador Tutorial

In this article, you will explore some of the key features of the Ambassador
Edge Stack by walking through an example workflow and exploring the 
Edge Policy Console.

## Prerequisites

You must have [Ambassador Edge Stack installed](../getting-started/) in your 
Kubernetes cluster.

## Routing Traffic from the Edge

Ambassador empowers developers with self-service functionality for managing traffic from the cluster’s edge to Kubernetes services.  Like any other Kubernetes object, Custom Resource Definitions (CRDs) are used to declaratively define Ambassador’s desired state.

The workflow you're going to build uses a sample deployment and the `Mapping` CRD, which is the core resource that you will use with Ambassador to manage your edge. It enables you to route requests by host and URL path from the edge of your cluster to Kubernetes services.

1. Copy the configuration below and save it to a file named `quote.yaml` so that you can deploy these resources to your cluster. This basic configuration deploys the `quote` pod and a service to expose the pod on port 80.

  ```yaml
  ---
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: quote
    namespace: ambassador
  spec:
    replicas: 1
    selector:
      matchLabels:
        app: quote
    strategy:
      type: RollingUpdate
    template:
      metadata:
        labels:
          app: quote
      spec:
        containers:
        - name: backend
          image: docker.io/datawire/quote:$quoteVersion$
          ports:
          - name: http
            containerPort: 8080
  ---
  apiVersion: v1
  kind: Service
  metadata:
    name: quote
    namespace: ambassador
  spec:
    ports:
    - name: http
      port: 80
      targetPort: 8080
    selector:
      app: quote
  ```

1. Apply the configuration to the cluster with the command `kubectl apply -f quote.yaml`.

1. Copy the configuration below and save it to 
a file called `quote-backend.yaml` to deploy a `Mapping` to your cluster. This `Mapping` tells Edge Stack to route all traffic 
from `/backend/` to the `quote` service. 

  ```yaml
  ---
  apiVersion: getambassador.io/v2
  kind: Mapping
  metadata:
    name: quote-backend
    namespace: ambassador
  spec:
    prefix: /backend/
    service: quote
  ```

1. Apply the configuration to the cluster with the command `kubectl apply -f quote-backend.yaml`

1. Store the Ambassador `LoadBalancer` address to a local environment variable.  You will use this variable to test accessing your pod.

  ```
  export AMBASSADOR_LB_ENDPOINT=$(kubectl -n ambassador get svc ambassador -o "go-template={{range .status.loadBalancer.ingress}}{{or .ip .hostname}}{{end}}")
  ```

1. Test the configuration by accessing the service through the Ambassador load 
balancer.

  ```
  $ curl -Lk "https://$AMBASSADOR_LB_ENDPOINT/backend/"
  {
   "server": "idle-cranberry-8tbb6iks",
   "quote": "Non-locality is the driver of truth. By summoning, we vibrate.",
   "time": "2019-12-11T20:10:16.525471212Z"
  }
  ```

Success, you have created your first Ambassador `Mapping`, routing a
request from your cluster's edge to a service! This is a diagram of the 
architecture you have created:

![Ambassador Edge Stack routing traffic to a service](../../images/quote-diagram.png)

## Edge Policy Console

The Edge Policy Console is a web-based interface used to configure and monitor 
Ambassador. You are going to log in to the console next in order to explore some of it's features.

1. Initially the console is accessed from the IP address of your load balancer. Run this command to retrieve that IP address from Kubernetes.

  ```
  kubectl get -n ambassador service ambassador -o "go-template={{range .status.loadBalancer.ingress}}{{or .ip .hostname}}{{end}}"
  ```

1. In your browser, navigate to `http://<your-IP-address>` and follow the prompts to bypass the TLS warning. 

  > [A `Host` resource is created in production](../../topics/running/host-crd)
to use your own registered domain name instead of an IP address to access the console and your `Mapping` endpoints.

1. The next page will prompt you to login to the console using `edgectl`, the Ambassador CLI. The page provides instructions to install `edgectl` for all OSes.

1. Click the **Mappings** tab in your Edge Policy Console. Scroll down to find an 
entry for the `quote-backend` `Mapping` that you created in your terminal
with `kubectl`.

As you can see, the console exposes the `Mapping` that you created earlier. In 
Ambassador, Kubernetes serves as the single source of truth 
around cluster configuration. Changes made via `kubectl` are reflected in the 
Edge Policy Console and vice versa.  Try the following to see this in action.

1. Click **Edit** next to the `quote-backend` entry.

1. Change the **Prefix URL** from `/backend/` to `/quoteme/`.

1. Click **Save**.

1. Run `kubectl get mappings --namespace ambassador`.  You will see the 
`quote-backend` `Mapping` has the updated prefix listed. Try to access the 
endpoint again via `curl` with the updated prefix.

  ```
  $ kubectl get mappings --namespace ambassador
  NAME            PREFIX      SERVICE   STATE   REASON
  quote-backend   /quoteme/   quote
   
  $ curl -Lk "https://${AMBASSADOR_LB_ENDPOINT}/quoteme/"
  {
      "server": "snippy-apple-ci10n7qe",
      "quote": "A principal idea is omnipresent, much like candy.",
      "time": "2020-11-18T17:15:42.095153306Z"
  }
  ```

1. Change the prefix back to `/backend/` so that you can later use the `Mapping` 
with other tutorials.

## Developer Portal

The Quote service you just deployed publishes its API as a Swagger document. 
Ambassador automatically detects and publishes this documentation, helping with 
internal and external developer onboarding.

1. In the Edge Policy Console, navigate to the **APIs** tab. You'll see the 
Swagger documentation there for internal use.

1. Navigate to `https://<your-IP-address>/docs/` to see the 
externally visible Developer Portal. Make sure you include the trailing `/`. 
This is a fully customizable portal that you can share with third parties who 
need information about your APIs.

## Next Steps

Further explore some of the concepts you learned about in this article: 
* [`Mapping` resource](../../topics/using/intro-mappings/): Routes traffic from 
the edge of your cluster to a Kubernetes service.
* [`Host` resource](../../topics/running/host-crd/): Sets the hostname by which 
Edge Stack will be reachable and for provisioning TLS certificates.
* [Edge Policy Console](../../topics/using/edge-policy-console/): A web-based 
interface used to configure and monitor Edge Stack
* [Developer Portal](https://www.getambassador.io/docs/pre-release/topics/using/dev-portal/): 
Publishes an API catalog and Swagger API documentation.

The Ambassador Edge Stack has a comprehensive range of [features](/features/) to
support the requirements of any edge microservice.

Learn more about [how developers use Edge Stack](../../topics/using/) to manage 
edge policies.

Learn more about [how site reliability engineers and operators run Edge Stack](../../topics/running/) 
in production environments.

To learn how Edge Stack works, use cases, best practices, and more, check out 
the [docs home](../../) or read the [Ambassador Story](../../about/why-ambassador).

For a custom configuration, you can install Edge Stack 
[manually](../../topics/install/yaml-install).
