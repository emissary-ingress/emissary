# Knative Integration

[Knative](https://knative.dev/) is a popular Kubernetes-based platform for managing serverless workloads with 3 main components:
- Build - Source-to-container build orchestration
- Eventing - Management and delivery of events
- Serving - Request-driven compute that can scale to zero

We will be focusing on Knative Serving which builds on Kubernetes to support deploying and serving of serverless applications and functions.

Ambassador can watch for changes in Knative configuration in your Kubernetes cluster and set up routing accordingly.

**Note:** Knative was originally built with Istio handling cluster networking. This integration lets us replace Istio with Ambassador which will dramatically reduce the operational overhead of running Knative.

## Getting started

#### Prerequisites

- Knative requires a Kubernetes cluster v1.11 or newer with the MutatingAdmissionWebhook admission controller enabled. kubectl v1.10 is also required. This guide assumes that you’ve already created a Kubernetes cluster which you’re comfortable installing alpha software on.

- Ambassador should be installed in your cluster. Follow one of the [installation guides](https://www.getambassador.io/user-guide/install) for instructions on installing Ambassador.

#### Installation

1. Install Knative:

   Knative is installed from remote YAML manifests. Check the [Knative install documentation](https://knative.dev/docs/install/knative-with-ambassador/) to install the most recent version of Knative.

   **Note:** You can safely ignore the `no matches for kind "Gateway" in version "networking.istio.io/v1alpha3"` warnings during the installation since we will be using Ambassador instead of the Istio gateway.
   
2. Configure Ambassador to listen for Knative Services

    After Knative is installed, we need to tell Ambassador to start looking for any Knative service we create.

    This is done by setting the `AMBASSADOR_KNATIVE_SUPPORT` environment variable to `"true"` in the Ambassador deployment.

    ```yaml
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: ambassador
    spec:
    ...
            env:
            - name: AMBASSADOR_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: AMBASSADOR_KNATIVE_SUPPORT
              value: "true"
    ...
    ```
   

3. Deploy an example Knative Service:

    Now that we have Knative installed, we can use it to build serverless applications.

    Copy the YAML below into a file called `helloworld-go.yaml` and apply it with `kubectl`

    ```yaml
    apiVersion: serving.knative.dev/v1alpha1
    kind: Service
    metadata:
      name: helloworld-go
      namespace: default
    spec:
      template:
        spec:
          containers:
          - image: gcr.io/knative-samples/helloworld-go
            env: 
            - name: TARGET
              value: Ambassador is Awesome!
    ```

    ```
    kubectl apply -f helloworld-go.yaml
    ```
   
   Knative automatically creates an `Ingress`/`ClusterIngress` resource from this `Knative Service`. Ambassador can then use that to register a route to the `helloworld-go` application.
   
5. Send a request 

    Knative applications are exposed via an automatically assigned `Host` header. By default, this `Host` header takes the form of `{service-name}.{namespace}.example.com`.

    You can verify the value of the `Host` header of the `helloworld-go` `Knative Service` created above by grabbing the `EXTERNAL-IP` from the Kubernetes service

    ```
    $ kubectl get service helloworld-go

    NAME            TYPE           CLUSTER-IP   EXTERNAL-IP                         PORT(S)   AGE
    helloworld-go   ExternalName   <none>       helloworld-go.default.example.com   <none>    3m
    ```

    We can now use this value and the `EXTERNAL-IP` of Ambassador's Kubernetes service to route to the application:

    ```
    curl -H “Host: helloworld-go.default.example.com” <ambassador IP>
    ```

We have now installed Knative with Ambassador handling traffic to our serverless applications. See the [Knative documentation](https://knative.dev/umentation/) for more information on what else Knative can do.
