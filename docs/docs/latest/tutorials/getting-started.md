---
   description: In this guide, we'll walk through the process of deploying Ambassador Edge Stack in Kubernetes for ingress routing.
---

import DetectOS from './DetectOS'

<DetectOS/>

# Quick Start Installation Guide

In just four minutes, your cluster will be routing HTTPS requests from the
Internet to a backend service.

The Ambassador Edge Stack is deployed to Kubernetes via YAML for MacOS, Linux, and
Windows. For other options, such as Docker, read the [detailed instructions] (/docs/latest/topics/install/).

<details class="os-instructions" data-os="mac">
<summary class="heading">

### Install on MacOS
<img class="os-logo" src="/../../docs/latest/images/apple.png"/>

</summary>

1. Download the `edgectl` file [here](https://metriton.datawire.io/downloads/darwin/edgectl) or download it with a curl command:

    ```shell
    sudo curl -fL https://metriton.datawire.io/downloads/darwin/edgectl -o /usr/local/bin/edgectl && sudo chmod a+X /usr/local/bin/edgectl
    ```

    If you decide to download the file, you may encounter a security block. To change this:
    * Go to **System Preferences > Security & Privacy > General**.
    * Click the **Open Anyway** button.
    * On the new dialog, click the **Open** button.

2. Run the installer with `./edgectl install`

</details>

<details class="os-instructions" data-os="linux">
<summary class="heading">

### Install on Linux
<img class="os-logo" src="/../../docs/latest/images/linux.png"/>
</summary>

1. Download the `edgectl` file
   [here](https://metriton.datawire.io/downloads/linux/edgectl) or download it with a curl
   command:

    ```shell
    sudo curl -fL https://metriton.datawire.io/downloads/linux/edgectl -o /usr/local/bin/edgectl && sudo chmod a+x /usr/local/bin/edgectl
    ```
2. Run the installer with `./edgectl install`

</details>

<details class="os-instructions" data-os="windows">
<summary class="heading">

### Install on Windows
<img class="os-logo" src="/../../docs/latest/images/windows.png"/>

</summary>

1. Download the `edgectl` file
   [here](https://metriton.datawire.io/downloads/windows/edgectl.exe).
2. Run the installer with `edgectl.exe install`

</details>

## Installation

Your terminal will show you something similar to the following as the installer provisions
a load balancer, configures TLS, and provides you with an `edgestack.me` subdomain:


<div class="codeblockInstall">
$ <span class="userInputText">edgectl install</span><br/>
-> Installing the Ambassador Edge Stack $version$.<br/>
Downloading images. (This may take a minute.)<br/>

-> Provisioning a cloud load balancer. (This may take a minute, depending on
your cloud provider.)<br/>

Your AES installation's IP address is 4.3.2.1
-> Automatically configuring TLS<br/>

Please enter an email address. We'll use this email address to notify you prior
to domain and certificate expiration. We also share this email address with
Let's Encrypt to acquire your certificate for TLS.
</div>


Provide an email address as required by the ACME TLS certificate provider, Let's
Encrypt. Then your terminal will print something similar to the following:

<div class="codeblockInstall">
Email address [<span class="userInputText">john@example.com</span>]:<br/>

-> Acquiring DNS name random-word-3421.edgestack.me 
-> Obtaining a TLS certificate from Let's Encrypt 
-> TLS configured successfully

Congratulations! You've successfully installed the Ambassador Edge Stack in
your Kubernetes cluster. Visit <span class="userGuidanceText">random-word-3421.edgestack.me</span> to access your
Edge Stack installation and for additional configuration.
</div>

The `random-word-3421.edgestack.me` is a provided subdomain that allows the
Ambassador Edge Stack to automatically provision TLS and HTTPS for a domain
name, so you can get started right away.

Your new [Edge Policy Console](/docs/latest/topics/using/edge-policy-console) will open
automatically in your browser at the provided URL or IP address. 

![AES success](/../../docs/latest/images/aes-success.png)

### Minikube

Minikube users will see something similar to the following:

<div class="codeblockInstall">
$ <span class="userInputText">edgectl install</span><br/>
-> Installing the Ambassador Edge Stack $version$.<br/>
Downloading images. (This may take a minute.)<br/>

-> Local cluster detected. Now configuring automatic TLS.<br/>

Congratulations! You've successfully installed the Ambassador Edge Stack in
your Kubernetes cluster. However, we cannot connect to your cluster from the
Internet, so we could not configure TLS automatically.<br/>

Determine the IP address and port number of your Ambassador service.
(e.g., <span class="userInputText">minikube service -n ambassador ambassador</span>)<br/>

The following command will open the Edge Policy Console once you accept a
self-signed certificate in your browser.
$ <span class="userInputText">edgectl login -n ambassador IP_ADDRESS:PORT</span>

See <a class="not-active">https://www.getambassador.io/user-guide/getting-started/
</a></div>

## Installation Success

Congratulations, you've installed the Ambassador Edge Stack! Take advantage of
the quick start demo by [creating a mapping](/docs/latest/tutorials/quickstart-demo) on
your cluster using the Ambassador Edge Stack.

### What’s Next?

The Ambassador Edge Stack has a comprehensive range of [features](/features/) to
support the requirements of any edge microservice. To learn more about how the
Ambassador Edge Stack works, along with use cases, best practices, and more,
check out the [Welcome page](/docs/latest/) or read the [Ambassador
Story](/docs/latest/about/why-ambassador).

For a custom configuration, you can install the Ambassador Edge Stack [manually](/docs/latest/topics/install/yaml-install).

**Note that the provided `random-word-3421.edgestack.me` domain name will expire after 90 days**.
