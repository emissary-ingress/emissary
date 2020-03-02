# Safely Testing in Production

Canary release is a technique to reduce the risk of introducing a new version of software to production by slowly rolling out the change to a small subset of users, before rolling it out to the entire infrastructure and making it available to everybody.

## Benefits of Canary Releases

This technique was rather gruesomely inspired by the fact that canary birds were once used in coal mines to alert miners when toxic gases reached dangerous levels — the gases would kill the canary before killing the miners, which provides a warning to get out of the mine tunnels immediately. As long as the canary kept singing, the miners knew that the air was free of dangerous gases. If a canary died, then this signaled an immediate evacuation.

This technique is called "canary" releasing because just like canaries that were once used in coal mining to alert miners when toxic gases reached dangerous levels, a small set of end-users selected for testing act as the canaries and are used to provide an early warning. Unlike the poor canaries of the past, obviously no users are physically hurt during a software release, but negative results from a canary release can be inferred from telemetry and metrics in relation to key performance indicators (KPIs).

Canary tests can be automated, and are typically run after testing in a pre-production environment has been completed. The canary release is only visible to a fraction of actual users, and any bugs or negative changes can be reversed quickly by either routing traffic away from the canary or by rolling-back the canary deployment.

![Canary release process overview](../../../doc-images/canary-release-overview.png)

## Basic Kubernetes Canary Releases: Deployments

The first approach to Kubernetes canary releases that many engineers either discover or read about is creating a Service backed by multiple Deployments with [several common selector labels and one "canary" label](https://kubernetes.io/docs/concepts/cluster-administration/manage-deployment/#canary-deployments). For example, the Kubernetes documentation suggests using multiple "track" labels, such as "stable" and "canary", in addition to the common identifying Service selector labels (e.g. "app: guestbook"). As the track labels are not used as a selector in the Service this allows two different (stable and canary) Deployments to be created that do not overlap.

Assuming that a round-robin load-balancing algorithm is being used within the Service, the percentage of traffic directed to the canary can be selected by altering the ratio of "stable" to "canary" Deployments. For example, creating nine replicas of the "track: stable" Deployment and a single replica of the "track: canary" Deployment enables approximately 10% of traffic to flow to the new canary Deployment.


```yaml
apiVersion: v1
kind: Service
metadata:
  name: payment-service
spec:
  selector:
    app: payment-app
...

apiVersion: apps/v2
kind: Deployment
metadata:  
    name: payment
     replicas: 9
     ...
     labels:
        app: payment-app
        track: stable
     ...
     image: payment-app:v3

apiVersion: apps/v1
kind: Deployment
metadata:
    name: payment-canary
     replicas: 1
     ...
     labels:
        app: payment-app
        track: canary
     ...
     image: payment-app:v4
```


The obvious downside to implementing Kubernetes canary releases like this is that if the application contained within the Deployment is large or resource intensive, it may not be practical to deploy multiple versions of this. In addition, creating fine-grained canary releases that only route 1% or 2% of traffic is challenging, and this is a standard use case for a canary release with an application that receives a reasonable amount of traffic.

## Flexible Kubernetes Canary Releases: Smart Routing with the Ambassador Edge Stack

A more effective approach to Kubernetes canary releases is to use some kind of smart proxy, load balancer or API gateway -- like the Ambassador Edge Stack. Dynamically routing traffic at the request level means that only one Deployment is required for each of the "stable" and "canary" versions of the application. Instead of relying on a round-robin load balancing implementation, the smart proxy can direct a specified percentage of Service requests to each Deployment. This is exactly how Ambassador Edge Stack can be configured to canary release applications.

Using the smart routing approach requires two Kubernetes Services to be created -- one for the stable version of the app e.g. "payment", and one for the canary version of the app e.g. "payment-canary"-- and associated Deployments can be created a configured as required (the Deployments are not involved with smart routing).

Ambassador Edge Stack itself is deployed as a Service, typically of type LoadBalancer (which by default uses the underlying platform implementation of a load balancer e.g. on AWS this is an ELB, on GCP a TCP/UDP Load Balancer, etc). With the application Services and the Ambassador Edge Stack Service deployed, all that is required to enable a canary release is the creation of appropriate Ambassador Edge Stack Mapping, which are defined as a Kubernetes custom resource definition (CRD).

The example below defines the CRDs. Note the "weight: 1" property in the payment-canary mapping, which tells Ambassador Edge Stack to route 1% of traffic to this service for the /payment/ route. By default, the remaining amount of traffic, 99% in this case, will be routed to the Mapping without a weight specified.

Ambassador Edge Stack Service config:

```yaml
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: payment
spec:
  prefix: /payment/
  service: payment-service
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: payment-canary
spec:
  prefix: /payment/
  service: payment-canary:8081
  weight: 1
```

And the payment-service Service definition

```yaml
apiVersion: v1   
kind: Service
metadata:
  name: payment-service
spec:
  selector:
    app: payment-app
    track: stable
...

apiVersion: v1   
kind: Service
metadata:
  name: payment-canary
spec:
  selector:
    app: payment-app
    track: canary
...


apiVersion: apps/v1
kind: Deployment
metadata:  
  name: payment
  replicas: 9
  ...
  labels:
    app: payment-app
    track: stable
  ...
  image: payment-app:v3


apiVersion: apps/v1
kind: Deployment
metadata:
  name: payment-canary
  replicas: 1
  ...
  labels:
    app: payment-app
    track: canary
  ...
  image: payment-app:v4

```

We've written more about canary releases on the [Ambassador Edge Stack blog](https://blog.getambassador.io/search?q=canary). To learn more about this pattern, you can [read more here](https://blog.getambassador.io/cloud-native-patterns-canary-release-1cb8f82d371a).
