---
   description: In this guide, we'll walk through the process of deploying Ambassador Edge Stack in Kubernetes for ingress routing.
---
# Quick Start Installation Guide

In this guide, we'll walk you through installing and configuring the Ambassador Edge Stack in your Kubernetes cluster. Within a few minutes, your cluster will be routing HTTPS requests from the Internet to a backend service. You'll also have a sense of how the Ambassador Edge Stack is managed.

## Before You Begin

The Ambassador Edge Stack is designed to run in Kubernetes for production. The most essential requirements are:

* Kubernetes 1.11 or later
* The `kubectl` command-line tool
* Edge Control

## Quick Install (Recommended!)

The Ambassador Edge Stack is typically deployed to Kubernetes from the command line. If you don't have Kubernetes, you should use our [Docker](../../topics/install/docker) image to deploy the Ambassador Edge Stack locally.

When you use Edge Control on your publicly
accessible cluster, it will:

1. Install the Ambassador Edge Stack
2. Generate a domain name for you to access the Edge Policy Console and complete
   advanced configuration
3. Obtain a TLS certificate for that domain name
4. Configure automatic TLS and HTTPS using that certificate

**To install the Ambassador Edge Stack:**

1. Download the `edgectl` file for your operating system:

   * [MacOS](https://metriton.datawire.io/downloads/darwin/edgectl)
   * [Linux](https://metriton.datawire.io/downloads/linux/edgectl)
   * [Windows](https://metriton.datawire.io/downloads/windows/edgectl.exe)
   * or use a [curl command](/reference/edgectl-download).

   If using macOS, you may encounter a security block. To change this you need to enable permissions to download files outside of the app store. To change this:

     * Go to **System Preferences > Security & Privacy**.
     * Click the **Open Anyway** button.
     * Click the **Open** button.

2. Move the file into your PATH (for Windows users, move it into the Windows System path). For Linux and MacOS,
   * Ensure the file is executable with the command `chmod a+x edgectl`
   * You can view your PATH with `echo $PATH`
   * Move the file into a directory on your path. For example: `sudo mv edgectl /usr/local/bin/`

3. Now, run the following command: `edgectl install`

    Your terminal will show you something similar to the following as the installer provisions
    a load balancer, configures TLS, and provides you with an `edgestack.me` subdomain:

    ```
    $ edgectl install
    -> Installing the Ambassador Edge Stack $version$.
    Downloading images. (This may take a minute.)
    -> Provisioning a cloud load balancer. (This may take a minute, depending on
    your cloud provider.)
    Your AES installation's IP address is 4.3.2.1
    -> Automatically configuring TLS

    Please enter an email address. We'll use this email address to notify you prior
    to domain and certificate expiration. We also share this email address with
    Let's Encrypt to acquire your certificate for TLS.
    ```

    Minikube users will see something similar to the following:

    ```
    $ edgectl install
    -> Installing the Ambassador Edge Stack $version$.
    Downloading images. (This may take a minute.)
    -> Local cluster detected. Not configuring automatic TLS.

    Congratulations! You've successfully installed the Ambassador Edge Stack in
    your Kubernetes cluster. However, we cannot connect to your cluster from the
    Internet, so we could not configure TLS automatically.

    Determine the IP address and port number of your Ambassador service.
    (e.g., minikube service -n ambassador ambassador)

    The following command will open the Edge Policy Console once you accept a
    self-signed certificate in your browser.

    $ edgectl login -n ambassador IP_ADDRESS:PORT

    See https://www.getambassador.io/user-guide/getting-started/
    ```

4. Provide an email address as required by the ACME TLS certificate provider, Let's Encrypt. Then you will see something similar to the following:

    ```shell
    Email address [john@example.com]:

    -> Acquiring DNS name random-word-3421.edgestack.me
    -> Obtaining a TLS certificate from Let's Encrypt
    -> TLS configured successfully

    Congratulations! You've successfully installed the Ambassador Edge Stack in
    your Kubernetes cluster. Visit random-word-3421.edgestack.me to access your
    Edge Stack installation and for additional configuration.
    ```

    The `random-word-3421.edgestack.me` is a provided subdomain that allows the
    Ambassador Edge Stack to automatically provision TLS and HTTPS for a domain
    name, so you can get started right away.

Your new [Edge Policy Console](../../topics/using/edge-policy-console) will open
automatically in your browser at the provided URL or IP address. **Note that the provided `random-word.edgestack.me` domain name will expire after 90 days**.

## Create a Mapping

In a typical configuration workflow, Custom Resource Definitions (CRDs) are used to define the intended behavior of Ambassador Edge Stack. In this example, we'll deploy a sample service and create a `Mapping` resource. Mappings allow you to associate parts of your domain with different URLs, IP addresses, or prefixes.

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

2. Deploy the `quote` service to the cluster by typing the command `kubectl apply -f quote.yaml`.

3. Now, create a `Mapping` configuration that tells Ambassador to route all traffic from `/backend/` to the `quote` service. Copy the following YAML and save it to a file called `quote-backend.yaml`.

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
best practices, and more, check out the [documentation](/docs/).

For a custom configuration, you can install the Ambassador Edge Stack with our [standard YAML](../../topics/install/yaml-install) or [Helm](../../topics/install/helm).