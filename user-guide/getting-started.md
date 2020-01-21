---
    description: In this guide, we'll walk through the process of deploying Ambassador Edge Stack in Kubernetes for ingress routing.
---
# Quick Start Installation Guide

In this guide, we'll walk you through installing and configuring the Ambassador Edge Stack in your Kubernetes cluster. Within a few minutes, your cluster will be routing HTTPS requests from the Internet to a backend service. You'll also have a sense of how the Ambassador Edge Stack is managed.

## Before You Begin

The Ambassador Edge Stack is designed to run in Kubernetes for production. The most essential requirements are:

* Kubernetes 1.11 or later
* The `kubectl` command-line tool

## Install the Ambassador Edge Stack

The Ambassador Edge Stack is typically deployed to Kubernetes from the command line. If you don't have Kubernetes, you should use our [Docker](../../about/quickstart) to deploy the Ambassador Edge Stack locally.


1. In your terminal, run the following command:

    ```bash
    kubectl apply -f https://www.getambassador.io/yaml/aes-crds.yaml && \
    kubectl wait --for condition=established --timeout=90s crd -lproduct=aes && \
    kubectl apply -f https://www.getambassador.io/yaml/aes.yaml && \
    kubectl -n ambassador wait --for condition=available --timeout=90s deploy -lproduct=aes
    ```

2. Determine the IP address of your cluster by running the following command:

    ```bash
    kubectl get -n ambassador service ambassador -o 'go-template={{range .status.loadBalancer.ingress}}{{print .ip "\n"}}{{end}}'
    ```

    Your load balancer may take several minutes to provision your IP address. Repeat the provided command until you get an IP address.

    Note: If you are a **Minikube user**, Minikube does not natively support load balancers. Instead, use `minikube service list`. You should see something similar to the following:

    ```bash
    (⎈ |minikube:ambassador)$ minikube service list
    |-------------|------------------|--------------------------------|
    |  NAMESPACE  |       NAME       |              URL               |
    |-------------|------------------|--------------------------------|
    | ambassador  | ambassador       | http://192.168.64.2:31230      |
    |             |                  | http://192.168.64.2:31042      |
    | ambassador  | ambassador-admin | No node port                   |
    | ambassador  | ambassador-redis | No node port                   |
    | default     | kubernetes       | No node port                   |
    | kube-system | kube-dns         | No node port                   |
    |-------------|------------------|--------------------------------|
    ```

    Use any of the URLs listed next to `ambassador` to access the Ambassador Edge Stack.

3. Navigate to `http://<your-IP-address>` and click through the certificate warning for access the Edge Policy Console interface. The certificate warning appears because, by default, the Ambassador Edge Stack uses a self-signed certificate for HTTPS.
    * Chrome users should click **Advanced > Proceed to website**.
    * Safari users should click **Show details > visit the website** and provide your password.

4. To login to the [Edge Policy Console](../../about/edge-policy-console), download and install `edgectl`, the command line tool Edge Control, by following the provided instructions on the page. The Console lists the correct command to run, and provides download links for the edgectl binary.

The Edge Policy Console must authenticate your session using a Kubernetes Secret in your cluster. Edge Control accesses that secret using `kubectl`, then sends a URL to your browser that contains the corresponding session key. This extra step ensures that access to the Edge Policy Console is just as secure as access to your Kubernetes cluster.

For more information, see [Edge Control](../../reference/edge-control).

## Configure TLS Termination and Automatic HTTPS

**The Ambassador Edge Stack enables TLS termination by default using a self-signed certificate. See the [Host CRD](/reference/host-crd) for more information about disabling TLS.** If you have the ability to update your DNS, Ambassador can automatically configure a valid TLS certificate for you, eliminating the TLS warning. If you do not have the ability to update your DNS, skip to the next section, "Create a Mapping."

1. Update your DNS so that your domain points to the IP address for your cluster. 

2. In the Edge Policy Console, create a `Host` resource:
   * On the "Hosts" tab, click the **Add** button on the right.
   * Enter your hostname (domain) in the hostname field.
   * Check the "Use ACME to manage TLS" box.
   * Review the Terms of Service and check the box that you agree to the Terms of Service.
   * Enter the email address to be associated with your TLS certificate.
   * Click the **Save** button.
  
  You'll see the newly created `Host` resource appear in the UI with a status of "Pending." This will change to "Ready" once the certificate is fully provisioned. If you receive an error that your hostname does not qualify for ACME management, you can still configure TLS following [these instructions](../../reference/core/tls) or by reviewing configuration in the [Host CRD](/reference/host-crd).

3. Once the Host is ready, navigate to `https://<hostname>` in your browser. Note that the certificate warning has gone away. In addition, the Ambassador Edge Stack automatically will redirect HTTP connections to HTTPS.

## Create a Mapping

In a typical configuration workflow, Custom Resource Definitions (CRDs) are used to define the intended behavior of Ambassador Edge Stack. In this example, we'll deploy a sample service and create a `Mapping` resource. Mappings allow you to associate parts of your domain with different URLs, IP addresses, or prefixes.

1. We'll start by deploying the `quote` service. Save the below configuration into a file named `quote.yaml`. This is a basic configuration that tells Kubernetes to deploy the `quote` container and create a Kubernetes `service` that points to the `quote` container.

   ```
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
           image: quay.io/datawire/quote:0.2.7
           ports:
           - name: http
             containerPort: 8080
   ```

2. Deploy the `quote` service to the cluster by typing the command `kubectl apply -f quote.yaml`.

3. Now, create a `Mapping` configuration that tells Ambassador to route all traffic from `/backend/` to the `quote` service. Copy the YAML and save to a file called `quote-backend.yaml`.

   ```
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

4. Apply the configuration to the cluster by typing the command `kubectl apply -f quote-backend.yaml`.

5. Test the configuration by typing `curl -Lk https://<hostname>/backend/` or `curl -Lk https://<IP address>/backend/`. You should see something similar to the following:

   ```
   (⎈ |rdl-1:default)$ curl -Lk https://aes.ri.k36.net/backend/
   {
    "server": "idle-cranberry-8tbb6iks",
    "quote": "Non-locality is the driver of truth. By summoning, we vibrate.",
    "time": "2019-12-11T20:10:16.525471212Z"
   }
   ```

## A single source of configuration

1. In the Ambassador Edge Stack, Kubernetes serves as the single source of configuration. Changes made on the command line (via `kubectl`) are reflected in the UI, and vice versa. This enables a consistent configuration workflow. You can see this in action by navigating to the Mappings tab. You'll see an entry for the `quote-backend` Mapping that was just created on the command line.

2. If you configured TLS, you can type `kubectl get hosts` to see the `Host` resource that was created:

   ```
   (⎈ |rdl-1:default)$ kubectl get hosts
   NAME               HOSTNAME           STATE   PHASE COMPLETED   PHASE PENDING   AGE
   aes.ri.k36.net     aes.ri.k36.net     Ready                                    158m
   ```

## Developer Onboarding

The Quote service we just deployed publishes its API as a Swagger document. This API is automatically detected by the Ambassador Edge Stack and published.

1. In the Edge Policy Console, navigate to the **APIs** tab. You'll see the documentation there for internal use.

2. Navigate to `https://<hostname>/docs/` or `https://<IP address>/docs/` to see the externally visible Developer Portal (make sure you include the trailing `/`). This is a fully customizable portal that you can share with third parties who need to information about your APIs. 

## What’s Next?

The Ambassador Edge Stack has a comprehensive range of [features](/features/) to support the requirements of any edge microservice.

To learn more about how the Ambassador Edge Stack works, along with use cases, best practices, and more, check out the [Ambassador](../../about/why-ambassador) story.
