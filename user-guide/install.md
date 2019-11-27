# Quick Start Installation Guide

The Ambassador Edge Stack provides a comprehensive, self-service edge stack in the Kubernetes cluster with a decentralized deployment model and a declarative paradigm. So what does that mean, exactly?

**Self service** is the ability for an application developer to independently complete configurations without bothering their Operations team. For the Ambassador Edge Stack, this means app-devs can configure the traffic at the “edge” of their application 

**Declarative** in k8s land means that developers can create configuration files that declare the ideal end state, and Ambassador does the work to achieve it so you don’t need to worry about the control flow. This means less work overall.

**Custom Resource Definitions** are Kubernetes objects that define Mappings that route your services. The Ambassador Edge Stack is configured using CRDs, which you can create with a text editor or on the Edge Policy Console interface. And once you create them, the Edge Policy Console supports a full round-trip creation and editing CRDs.

The Ambassador Edge Stack enables GitOps-style management of your application, in addition to the easy to use Edge Policy Console interface that visually displays all of your configurations.

**To start using the Ambassador Edge Stack and its features right away:**

1. Install the Ambassador Edge Stack
2. Add Hosts and TLS
3. Create Mappings

## Working Requirements

The Ambassador Edge Stack is designed to run in Kubernetes for production. The most essential requirements are:

* Kubernetes Cluster 1.11 or later
* Kubectl

You can find additional requirements and recommendations for a successful installation and deployment on the [Product Requirements] page.

## Deploy the Ambassador Edge Stack to Kubernetes

The Ambassador Edge Stack is deployed via a YAML file, which automatically configures [these fifteen Custom Resource Definitions [(CRDs)](../reference/core/crds.md).

**To deploy the Ambassador Edge Stack:**

1. In a command line tool, run the following command:

    ```
    kubectl apply -f secret.yaml && \
    kubectl apply -f https://deploy-preview-91--datawire-ambassador.netlify.com/yaml/aes-crds.yaml && \
    kubectl wait --for condition=established --timeout=60s crd -lproduct=aes && \
    kubectl apply -f https://deploy-preview-91--datawire-ambassador.netlify.com/yaml/aes.yaml && \
    kubectl -n ambassador wait --for condition=available --timeout=60s deploy -lproduct=aes
    ```

2. Determine your IP address by running the following command:

```
kubectl get -n ambassador service ambassador -o 'go-template={{range .status.loadBalancer.ingress}}{{print .ip "\n"}}{{end}}'
```

Your load balancer may not provision your IP address automatically. You can repeat the provided command as necessary until you get an IP address.

Note: If you are a **Minikube user**, Minikube does not natively support load balancers. Instead, use the following command: `minikube service list`

You should see something similar to the following:

```
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

4. Navigate to `http://<your-host-name>/edge_stack/admin/` and follow the instructions to install the `edgectl` executable file in order to use the Edge Policy Console, the admin interface.

* Note that you must click through the certificate warning for access to the instructions.

## Add Hosts and Configure TLS

You now have access to the Edge Policy Console, where you can start to secure your application with automatic HTTPS, configure TLS with ease, and create CRDs and mappings independently of your Ops teams.

To secure your application with HTTPS, you must first add a Host to your Ambassador Edge Stack configuration.

**To do so:**

1. From the left menu, go to the **Hosts** tab and then click the **(+)** button on the right.
A configuration table appears with your IP address prepopulated in the “Resource” field. You can change this, but we recommend using your hostname in this field.

* Note: If you want to use automatic TLS during this section, your host must be an FQDN. Otherwise, you will see an error message indicating that your host does not qualify for the ACME Certificate. You can continue without TLS configuration.

2. In the “Namespace” field, add an existing namespace. We recommend that it matches the namespace of your Kubeconfig context, or leaving it set to “Default.”
3. Read and check the box to agree to the Terms of Service.
4. Enter your email address to receive your TLS certificate.

Your hostname will appear in a pending state as the Ambassador Edge Stack configures automatic TLS. In the “Status” field, you will see the TLS status change. If you receive an error that your hostname does not qualify for ACME management, you can still configure TLS following [these instructions].

To upgrade from the evaluation mode, [sign up for a free community license] today.

## Create Mappings

Before you can create `Mappings`, you should verify that all of the CRDs were successfully installed using `Kubectl`, the command-line tool that allows you to control Kubernetes.

Because Ambassador is built on top of Kubernetes, it’s more than likely that you already have `kubectl` installed on your machine. However, if you don’t, you can install it following [these directions](https://kubernetes.io/docs/tasks/tools/install-kubectl/).

To verify that the CRDs were installed, run the command: `kubectl get mappings`

Kubernetes will print the Custom Resource Definitions that were installed with the Ambassador Edge Stack.

Next, you can start to create your own Mappings. You can do this in the Edge Policy Console, or via the command line.

**To configure Mappings in the Edge Policy Console:**

1. Navigate to your Edge Policy Console.
2. From the left menu, click the **Mappings** tab.
3. On the right hand side, click the **(+)** button to add a mapping.
4. In the empty table, provide information to configure your mapping. 
5. Click the **Save** button.
6. In your command line, if you run the  `kubectl get mappings` again, you will see that your new Mapping has been adding to the CRDs of the Ambassador Edge Stack.

## What’s Next?

To learn more about how the Ambassador Edge Stack works, along with use cases, best practices, and more, check out the [Product Overview](/content/docs/index.md]!
