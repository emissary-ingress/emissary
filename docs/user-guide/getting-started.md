---
   description: In this guide, we'll walk through the process of deploying Ambassador Edge Stack in Kubernetes for ingress routing.
---
# Quick Start Installation Guide

In just four minutes, your cluster will be routing HTTPS requests from the
Internet to a backend service.

The Ambassador Edge Stack is deployed to Kubernetes via YAML for MacOS, Linux, and
Windows. For other options, such as Docker, click [here](/user-guide/install).

### Install on MacOS

1. Download the `edgectl` file [here](https://metriton.datawire.io/downloads/darwin/edgectl) or download it with a curl command:

    ```shell
    sudo curl -fL https://metriton.datawire.io/downloads/darwin/edgectl -o /usr/local/bin/edgectl && sudo chmod a+X /usr/local/bin/edgectl
    ```

    If you decide to download the file, you may encounter a security block. To change this:
    * Go to **System Preferences > Security & Privacy > General**.
    * Click the **Open Anyway** button.
    * On the new dialog, click the **Open** button.

2. Run the installer with `./edgectl install`

### Install on *nix

1. Download the `edgectl` file
   [here](https://metriton.datawire.io/downloads/linux/edgectl) or download it with a curl
   command:

    ```shell
    sudo curl -fL https://metriton.datawire.io/downloads/linux/edgectl -o /usr/local/bin/edgectl && sudo chmod a+x /usr/local/bin/edgectl
    ```
2. Run the installer with `./edgectl install`

### Install on Windows

1. Download the `edgectl` file
   [here](https://metriton.datawire.io/downloads/windows/edgectl.exe).
2. Run the installer with `edgectl.exe install`

## Installation

Your terminal will print something similar to the following as it provisions a load balancer, configures TLS, and provides you with an `edgestack.me` subdomain:

```
$ edgectl install
-> Installing the Ambassador Edge Stack $version$.
-> Remote Kubernetes cluster detected.
-> Provisioning a cloud load balancer. (This may take 
a minute, depending on your cloud provider.)
-> Automatically configuring TLS.
Please enter an email address. We’ll use this email 
address to notify you prior to domain and certification 
expiration [None]: john@example.com.
```

Provide an email address as required by the ACME TLS certificate provider, Let's
Encrypt. Then your terminal will print something similar to the following:

```
-> Obtaining a TLS certificate from Let’s Encrypt.
Congratulations, you’ve successfully installed the 
Ambassador Edge Stack in your Kubernetes cluster. 
Visit https://random-word-3421.edgestack.me to access
your Edge Stack installation and for additional configuration
```

The `random-word-1234.edgestack.me` is a provided subdomain that allows the
Ambassador Edge Stack to automatically provision TLS and HTTPS for a domain
name, so you can get started right away.

Your new [Edge Policy Console](/about/edge-policy-console) will open
automatically in your browser at the provided URL or IP address. **Note that the provided `random-word.edgestack.me` domain name will expire after 90 days**.

![AES success](/../../doc-images/aes-success.png)

### Minikube

Minikube users will see something similar to the following:

```
$ edgectl install
-> Installing the Ambassador Edge Stack $version$.
-> Automatically configuring TLS.
-> Cluster is not publicly accessible. Please ensure 
your cluster is publicly accessible if you would like to 
use automatic TLS.
Congratulations, you’ve successfully installed the
 Ambassador Edge Stack in your Kubernetes cluster. 
 Visit http://192.168.64.2:31334 to access your Edge 
 Stack installation and for additional configuration.
 ```

## Installation Success

Congratulations, you've installed the Ambassador Edge Stack! Take advantage of
the quick start demo by [creating a mapping](/user-guide/quickstart-demo) on
your cluster using the Ambassador Edge Stack.

### What’s Next?

The Ambassador Edge Stack has a comprehensive range of [features](/features/) to
support the requirements of any edge microservice. To learn more about how the
Ambassador Edge Stack works, along with use cases, best practices, and more,
check out the [Welcome page](/docs/) or read the [Ambassador
Story](/about/why-ambassador).

For a custom configuration, you can install the Ambassador Edge Stack [manually](/user-guide/manual-install).
