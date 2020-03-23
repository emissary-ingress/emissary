# Installation Demo

## Create a Mapping

In a typical configuration workflow, Custom Resource Definitions (CRDs) are used to define the intended behavior of Ambassador Edge Stack. In this demo, we'll deploy a sample service and create a `Mapping` resource. Mappings allow you to associate parts of your domain with different URLs, IP addresses, or prefixes.

1. We'll start by deploying the `quote` service. Save the below configuration into a file named `quote.yaml`. This is a basic configuration that tells Kubernetes to deploy the `quote` container and create a Kubernetes `service` that points to the `quote` container.

  ```yaml
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
          image: quay.io/datawire/quote:$quoteVersion$
          ports:
          - name: http
            containerPort: 8080
  ```

2. Deploy the `quote` service to the cluster by typing the command `kubectl apply -f quote.yaml`

3. Now, create a `Mapping` configuration that tells Ambassador to route all traffic from `/backend/` to the `quote` service. Copy the following YAML and save it to a file called `quote-backend.yaml`

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

4. Apply the configuration to the cluster by typing the command `kubectl apply -f quote-backend.yaml`

5. Test the configuration by typing `curl -Lk https://<hostname>/backend/` or
   `curl -Lk https://<IP address>/backend/`

You should see something similar to the following:

  ```
  (⎈ |rdl-1:default)$ curl -Lk https://aes.ri.k36.net/backend/
  {
   "server": "idle-cranberry-8tbb6iks",
   "quote": "Non-locality is the driver of truth. By summoning, we vibrate.",
   "time": "2019-12-11T20:10:16.525471212Z"
  }
  ```

## A Single Source of Configuration

In the Ambassador Edge Stack, Kubernetes serves as the single source of configuration. Changes made on the command line (via `kubectl`) are reflected in the Edge Policy Console, and vice versa.

1. To see this in action, navigate to the **Mappings** tab. You'll see an entry for the `quote-backend` Mapping that was just created on the command line.

2. Type `kubectl get hosts` to see the `Host` resource that was created:

```  
  (⎈ |rdl-1:default)$ kubectl get hosts
 NAME                          HOSTNAME                      STATE   PHASE COMPLETED   PHASE PENDING   AGE
 blackbird-123.edgestack.me    blackbird-123.edgestack.me    Ready                                     158m
 ```

## Developer Onboarding

The Quote service we just deployed publishes its API as a Swagger document. This API is automatically detected by the Ambassador Edge Stack and published.

1. In the Edge Policy Console, navigate to the **APIs** tab. You'll see the documentation there for internal use.

2. Navigate to `https://<hostname>/docs/` or `https://<IP address>/docs/` to see the externally visible Developer Portal (make sure you include the trailing `/`). This is a fully customizable portal that you can share with third parties who need information about your APIs.

## What’s Next?

The Ambassador Edge Stack has a comprehensive range of [features](/features/) to support the requirements of any edge microservice.

To learn more about how the Ambassador Edge Stack works, along with use cases,
best practices, and more, check out the [Welcome page](/docs/) or read the
[Ambassador Story](/docs/latest/about/why-ambassador).

For a custom configuration, you can install the Ambassador Edge Stack [manually](/topics/install/yaml-install.md).