# Quick Start Installation Guide

The Ambassador Edge Stack provides a comprehensive, self-service edge stack in the Kubernetes cluster with a decentralized deployment model and a declarative paradigm. So what does that mean, exactly?

* **Comprehensive** allows you to take all the different pieces of your cluster, application, traffic, security, etc, and conveniently manage them all in one place, instead of having to individually configure them all, or their interactions with one another.
* **Self service** is the ability for an application developer to independently complete configurations without bothering their Operations team. For the Ambassador Edge Stack, this means app-devs can configure the traffic at the “edge” of their application.

In other words, **the Ambassador Edge Stack is an all-in-one edge stack management tool** that allows developers and platform engineers alike to skip the red tape and get things done without stepping on each other’s toes.

The Ambassador Edge Stack enables GitOps-style management of your application, in addition to the easy to use Edge Policy Console interface that visually displays all of your configurations.

To start using the Ambassador Edge Stack and its features right away:

1. [Install the Ambassador Edge Stack](/user-guide/install#install-the-ambassador-edge-stack)
2. [Add Hosts for automatic HTTPS](/user-guide/install#add-hosts-and-configure-tls)
3. [Create and Verify Mappings](/user-guide/install#create-mappings)
4. [What's Next?](/user-guide/install#whats-next)

## Before You Begin

The Ambassador Edge Stack is designed to run in Kubernetes for production. The most essential requirements are:

* Kubernetes Cluster 1.11 or later
* `Kubectl`

Find additional requirements and recommendations for a successful installation and deployment on the [Product Requirements](/user-guide/product-requirements) page.

## Install the Ambassador Edge Stack

The Ambassador Edge Stack is deployed to Kubernetes from the command line via our custom manifest in a YAML file.

**To deploy the Ambassador Edge Stack:**

1. In a command line tool, run the following command:

    ```bash
    kubectl apply -f secret.yaml && \
    kubectl apply -f https://deploy-preview-91--datawire-ambassador.netlify.com/yaml/aes-crds.yaml && \
    kubectl wait --for condition=established --timeout=60s crd -lproduct=aes && \
    kubectl apply -f https://deploy-preview-91--datawire-ambassador.netlify.com/yaml/aes.yaml && \
    kubectl -n ambassador wait --for condition=available --timeout=60s deploy -lproduct=aes
    ```

2. Determine your IP address by running the following command:

    ```bash
    kubectl get -n ambassador service ambassador -o 'go-template={{range .status.loadBalancer.ingress}}{{print .ip "\n"}}{{end}}'
    ```

Your load balancer may take several minutes to provision your IP address. Repeat the provided command until you get an IP address.

Note: If you are a **Minikube user**, Minikube does not natively support load balancers. Instead, use the following command: `minikube service list`

You should see something similar to the following:

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

3. Once you have an IP address, navigate to it in your browser. Optionally assign a DNS name to your IP address from step 2 using a provider of your choice, such as gandi.net. Or, continue using the IP address instead.

4. Navigate to `http://<your-host-name>/` or `http://<your-IP-address>` and click through the certificate warning for access to the instructions. 
    * Chrome users should click **Advanced > Proceed to website**. 
    * Safari users should click **Show details > visit the website** and provide your password.

5. Follow the instructions to install the `edgectl` executable file.

## Add Hosts and Configure TLS

You now have access to the Edge Policy Console, where you can start to secure your application with automatic HTTPS, configure TLS with ease, and create CRDs and mappings independently of your Ops teams.

To secure your application with HTTPS, you must first add a Host to your Ambassador Edge Stack Configuration. 

**To do so:**

1. Navigate to your Edge Policy Console. From the left menu, go to the **Hosts** tab and then click the **(+)** button on the right.
2. A configuration table appears with your IP address prepopulated in the “Resource” field. You can change this, but we recommend using your hostname in this field.

* Note: If you want to use automatic TLS during this section, your host must be an FQDN. Otherwise, you will see an error message indicating that your host does not qualify for the ACME Certificate. You can continue without TLS configuration.

3. To the right of the “Host” field, enter an existing namespace for the host. We recommend that it matches the namespace of your Kubeconfig context, or leaving it set to “Default.”
4. Read and check the box to agree to the Terms of Service.
5. Enter your email address to receive your TLS certificate.

Your hostname will appear in a pending state as the Ambassador Edge Stack configures automatic TLS. In the “Status” field, you will see the TLS status change. If you receive an error that your hostname does not qualify for ACME management, you can still configure TLS following [these instructions](/reference/core/tls).

To upgrade from evaluation mode, sign up for a free community license today from your Edge Policy Console.

## Create Mappings

Mappings allow you to associate parts of your domain with different URLs, IP addresses, or prefixes. Create your own Mappings in the administrative interface, the Edge Policy Console, to map out your own application.

To show you how powerful the Ambassador Edge Stack is, follow the instructions to create a mapping to the site `httpbin.org`.

**To do so:**

1. Navigate to your Edge Policy Console.
2. From the left menu, click the **Mappings** tab.
3. On the right hand side, click the **(+)** button to add a mapping.
4. In the “Mapping” field, enter a name that follows the naming conventions found [here](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names), such as `examplemapping`.
5. To the right of the “Mapping” field, enter an existing namespace for your mapping, or leave it as `Default.`
6. In the “Prefix URL” field, enter `/httpbin/`
7. In the “Service” field, enter `httpbin.org:80`
8. Click the **Save** button. Your new mapping will appear in the URL table below, and as a block in the Mappings section.
9. To see how quickly the Mappings were applied, run the following command: `curl https://<hostname>/httpbin/` to print the HTML of the page.

### Verify Mappings

After you create Mappings, you can also verify that all of the CRDs were successfully installed using the command `kubectl get mappings`. `Kubectl` is the command-line tool that allows you to control Kubernetes.

Because Ambassador is built on top of Kubernetes, it’s more than likely that you already have `kubectl` installed on your machine. However, if you don’t, you can install it following [these directions](https://kubernetes.io/docs/tasks/tools/install-kubectl/).

Kubernetes will print the Custom Resource Definitions that were installed with the Ambassador Edge Stack, and you’ll see that your new Mapping has been adding to the CRDs of the Ambassador Edge Stack.

## What’s Next?

To learn more about how the Ambassador Edge Stack works, along with use cases, best practices, and more, check out the [Ambassador](/about/why-ambassador) story.
