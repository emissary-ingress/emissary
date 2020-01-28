# Self-Service Routing and Deployment Control

Traditionally, API Gateways have focused on operators as the primary user. Ambassador Edge Stack considers both developers and operators to be first-class citizens.

## Decentralized, declarative configuration

The Ambassador Edge Stack model is a decentralized, declarative configuration model. What does this mean?

* Decentralized. Ambassador Edge Stack is designed so that lots of developers can individually configure a specific aspect of its configuration (usually a route). Ambassador Edge Stack then aggregates these individual bits of configuration into a master configuration for the gateway.

* Declarative. In Ambassador Edge Stack, the user declares the desired end state of the configuration. Ambassador Edge Stack then figures out how to achieve that desired end state. If the desired end state is already in effect, no change happens. This is a contrast from an imperative model (most frequently seen as a REST API configuration), which forces the user to understand *how* to configure the gateway.

## Ambassador Edge Stack configuration in practice

In a typical Ambassador Edge Stack deployment, each service is owned by a developer or development team. This team writes the code, tests, and deploys the service. To deploy this service, a team must create a Kubernetes manifest that specifies the desired end state of the service. For example, the `my-service` service could be defined as below:

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

Because a Kubernetes `service` is the fundamental abstraction by which new services are exposed to other services and end-users, Ambassador Edge Stack extends the `service` with a custom mapping. For example:

```yaml
---
apiVersion: getambassador.io/v2
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

With this approach, there is no centralized Ambassador Edge Stack configuration file -- the routing configuration for Ambassador Edge Stack is associated with each service. This offers numerous benefits:

* Agility: Service owners can change their Ambassador Edge Stack configuration without worrying about other end users or going through a central operations function.
* Organizational scalability: Configuring individual routes in Ambassador Edge Stack is the responsibility of service owners, instead of a centralized team.
* Maintainability: If a service is deleted, the route disappears with the service. All of the machinery used to manage Kubernetes manifests can be used with Ambassador Edge Stack without modification.

## Ingress resources

You can use Ambassador Edge Stack as an [Ingress Controller](../../reference/core/ingress-controller).
