# Knative Integration

[Knative](https://knative.dev/) is a popular Kubernetes-based platform for managing serverless workloads with 3 main components:
- Build - Source-to-container build orchestration
- Eventing - Management and delivery of events
- Serving - Request-driven compute that can scale to zero

We will be focusing on Knative Serving which builds on Kubernetes and Istio to support deploying and serving of serverless applications and functions.

Replacing Istio with Ambassador can potentially reduce the operational overhead of running Knative.

Ambassador can watch for changes in Knative configuration in your Kubernetes cluster and set up routing accordingly.

## Getting started

#### Prerequisites

Knative requires a Kubernetes cluster v1.11 or newer with the MutatingAdmissionWebhook admission controller enabled. kubectl v1.10 is also required. This guide assumes that you’ve already created a Kubernetes cluster which you’re comfortable installing alpha software on.

#### Installation

1. Install Knative:

   ```
   kubectl apply -f https://github.com/knative/serving/releases/download/v0.7.1/serving.yaml
   ```
   
2. Install Ambassador::
   
   ```
   kubectl apply -f https://getambassador.io/yaml/ambassador/ambassador-rbac.yaml
   kubectl apply -f https://getambassador.io/yaml/ambassador/ambassador-service.yaml
   ```
   
3. Set `AMBASSADOR_KNATIVE_SUPPORT: "true"` in `ambassador` deployment. Ambassador will only watch for Knative resources when this environment variable is set.

4. Deploy an example Knative Service:

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
            value: "Ambassador is Awesome"
   ```
   
5. Send a request to Ambassador with the hostname assigned to the Knative Service:
   ```
   curl -H “Host: <hostname>” <ambassador IP>
   ```

**Note**:
Knative integration in Ambassador is a very recent and an experimental feature, please use at your discretion.
