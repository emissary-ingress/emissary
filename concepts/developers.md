# Self-Service Routing and Deployment Control

Traditionally, API Gateways have focused on operators as the primary user. Ambassador considers both developers and operators to be first-class citizens.

## Decentralized, declarative configuration

The Ambassador model is a decentralized, declarative configuration model. What does this mean?

* Decentralized. Ambassador is designed so that lots of developers can individually configure a specific aspect of Ambassador's configuration (usually a route). Ambassador then aggregates these individual bits of configuration into a master configuration for the gateway.

* Declarative. In Ambassador, the user declares the desired end state of the configuration. Ambassador then figures out how to achieve that desired end state. If the desired end state is already in effect, no change happens. This is a contrast from an imperative model (most frequently seen as a REST API configuration), which forces the user to understand *how* to configure the gateway.

## Ambassador configuration in practice

In a typical Ambassador deployment, each service is owned by a developer or development team. This team writes the code, tests, and deploys the service. In order to deploy this service, a team must create a Kubernetes manifest that specifies the desired end state of the service. For example, the `my-service` service could be defined as below:

```
kind: Service
apiVersion: v1
metadata:
  name: my-service
spec:
  selector:
    app: MyApp
  ports:
  - protocol: TCP
    port: 80
    targetPort: 9376
```

Because a Kubernetes `service` is the fundamental abstraction by which new services are exposed to other services and end users, Ambassador extends the `service` with custom annotations. For example:

```yaml
---
apiVersion: getambassador.io/v1
kind: Mapping
metadata:
  name: my-service
spec:
  prefix: /my-service/
  service: my-service
---
kind: Service
apiVersion: v1
metadata:
  name: my-service
spec:
  selector:
    app: MyApp
  ports:
  - protocol: TCP
    port: 80
    targetPort: 9376
```

With this approach, there is no centralized Ambassador configuration file -- the routing configuration for Ambassador is associated with each individual service. This offers numerous benefits:

* Agility: Service owners can change their Ambassador configuration without worrying about other end users or going through a central operations function.
* Organizational scalability: Configuring individual routes in Ambassador is the responsibility of service owners, instead of a centralized team.
* Maintainability: If a service is deleted, the route disappears with the service. All of the machinery used to manage Kubernetes manifests can be used with Ambassador without modification.

## Centralized configuration

It's possible as a migration strategy to start with a centralized Ambassador configuration that contains all the necessary mappings (you can create a dummy `service` for this). In general, though, this approach is not recommended.

## Ingress resources

Astute Kubernetes observers may note that this decentralized model is not unique to Ambassador. In fact, Ingress controllers and Ingress resources can be used in a similar way. In this approach, a Kubernetes service stays unmodified, and a separate Ingress resource can be added to the Kubernetes manifest. You'd end up with something that looks like:

```
kind: Service
apiVersion: v1
metadata:
  name: my-service
spec:
  selector:
    app: MyApp
  ports:
  - protocol: TCP
    port: 80
    targetPort: 9376
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: my-service-ingress
spec:
  backend:
    serviceName: my-service
    servicePort: 80
```

Ambassador has chosen not to support ingress resources for the following reasons:

* The ingress API is extremely limited, and supports only a small subset of Ambassador's features
* The ingress API has been in beta for a number of years, and there is the possibility of it being deprecated altogether
* Having a separate object that is associated with the `service` object
