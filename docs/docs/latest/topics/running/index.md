# Running Ambassador in Production

This section of the documentation is designed for operators and site reliability engineers who are managing the deployment of Ambassador. Learn more below:

* *Global Configuration:* The [Ambassador module](ambassador) is used to set system-wide configuration.
* *Exposing Ambassador to the Internet:* [Host CRD](host-crd) defines how Ambassador is exposed to the outside world, managing TLS, domains, and such.
* *Load Balancing:* Ambassador supports a number of different [load balancing strategies](load-balancer) as well as different ways to configure [service discovery](resolvers)
* [Gzip Compression](gzip)
* *Deploying Ambassador:* On [Amazon Web Services](ambassador-with-aws) | [Google Cloud](ambassador-with-gke) | [general security and operational notes](running), including running multiple Ambassadors on a cluster
* *TLS/SSL:* [Simultaneously Routing HTTP and HTTPS](cleartext-redirection#cleartext-routing) | [HTTP -> HTTPS Redirection](cleartext-redirection#http---https-redirection) | [Mutual TLS](tls/mtls) | [TLS origination](tls/origination)
* *Monitoring* [Integrating with Prometheus, DataDog, and other monitoring systems](statistics)
* *Extending Ambassador* Ambassador can be extended with custom plug-ins that connect via HTTP/gRPC interfaces. [Custom Authentication](services/auth-service) | [The External Auth protocol](services/ext_authz) | [Custom Logging](services/log-service) | [Rate Limiting](services/rate-limit-service) | [Distributed Tracing](services/tracing-service)
* *Troubleshooting:* [Diagnostics](diagnostics) | [Debugging](debugging))
* *Ingress:* Ambassador can function as an [Ingress Controller](ingress-controller)