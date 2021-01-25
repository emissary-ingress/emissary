# Dev Portal Tutorial

In this tutorial, you will access and explore some of the key features of the Dev Portal.

## Prerequisites

You must have [Ambassador Edge Stack installed](../getting-started/) in your 
Kubernetes cluster. This tutorial assumes you have deployed the `quote` app and
`Mapping` from the [Edge Stack tutorial](../getting-started/).


  ```
  export AMBASSADOR_LB_ENDPOINT=$(kubectl -n ambassador get svc ambassador -o "go-template={{range .status.loadBalancer.ingress}}{{or .ip .hostname}}{{end}}")
  ```


## Edge Policy Console

First you are going to log in to the Edge Policy Console to explore some of its
features. The console is a web-based interface that can be used to configure and
monitor Ambassador. 

1. Initially the console is accessed from the load balancer's hostname or public
address (depending on your Kubernetes environment). Run the command below to
find that endpoint.

  ```
  kubectl -n ambassador get svc ambassador -o "go-template={{range .status.loadBalancer.ingress}}{{or .ip .hostname}}{{end}}"
  ```

1. In your browser, navigate to `http://<load-balancer-endpoint>` and follow the
prompts to bypass the TLS warning. 

  > [A `Host` resource is created in production](../../topics/running/host-crd)
to use your own registered domain name instead of the load balancer endpoint to 
access the console and your `Mapping` endpoints.

1. The next page will prompt you to log in to the console using `edgectl`, the 
Ambassador CLI. The page provides instructions on how to install `edgectl` for 
all OSes and log in.

1. Once logged in, click on the **Mappings** tab in the Edge Policy Console. 
Scroll down to find an entry for the `quote-backend` `Mapping`.

As you can see, the console lists the `Mapping` that you created in the Edge Stack tutorial. This
information came from Ambassador polling the Kubernetes API. In 
Ambassador, Kubernetes serves as the single source of truth 
around cluster configuration. Changes made via `kubectl` are reflected in the 
Edge Policy Console and vice versa.  Try the following to see this in action.

1. Click **Edit** next to the `quote-backend` entry.

1. Change the **Prefix URL** from `/backend/` to `/quoteme/`.

1. Click **Save**.

1. Run `kubectl get mappings --namespace ambassador`. You will see the 
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

## Developer API Documentation

The `quote` service you just deployed publishes its API as an 
[OpenAPI (formally Swagger)](https://swagger.io/solutions/getting-started-with-oas/)
document. Ambassador automatically detects and publishes this documentation. 
This can help with internal and external developer onboarding by serving as a 
single point of reference for of all your microservice APIs.

1. In the Edge Policy Console, navigate to the **APIs** tab. You'll see the 
OpenAPI documentation there for the "Quote Service API." Click **GET** to
expand out the documentation.

1. Navigate to `https://<load-balancer-endpoint>/docs/` to see the 
publicly visible Developer Portal. Make sure you include the trailing `/`. 
This is a fully customizable portal that you can share with third parties who 
need information about your APIs.

