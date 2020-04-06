# "How-to" guides

These guides are designed to help users quickly accomplish common tasks. The guides assume a certain level of understanding of Ambassador. Many of these guides are contributed by third parties; we welcome contributions via Pull Request at https://github.com/datawire/ambassador.

* Integrating with Service Mesh. Ambassador natively integrates with many service meshes.
  * [HashiCorp Consul](consul)
  * [Istio](istio)
  * [Linkerd](linkerd2)
* Distributed tracing. Ambassador natively supports a number of distributed tracing systems to enable developers to visualize request flow in microservice and service-oriented architectures.
  * [Datadog](tracing-datadog)
  * [Zipkin](tracing-zipkin)
* Identity providers. Ambassador integrates with a number of OAuth Identity Providers via OpenID Connect.
  * [Auth0](sso/auth0)
  * [Azure Active Directory](sso/azure)
  * [Google Identity](sso/google)
  * [Keycloak](sso/keycloak)
  * [Okta](sso/okta)
  * [Onelogin](sso/onelogin)
  * [Salesforce](sso/salesforce)
  * [UAA](sso/uaa)
* Monitoring. Ambassador integrates with a number of different monitoring/metrics providers.
  * [Prometheus](prometheus)
* [Developing Custom Filters](filter-dev-guide)
* Frameworks and Protocols. Ambassador supports a wide range of protocols and cloud-native frameworks.
  * [gRPC](grpc)
  * [Knative Serverless Framework](knative)
  * [WebSockets](websockets)
* Security. Ambassador supports a number of strategies for securing Kubernetes services.
  * [HTTPS and TLS termination](tls-termination)
  * [Certificate Manager](cert-manager) can be used to automatically obtain and renew TLS certificates; Ambassador Edge Stack natively integrates this functionality.
  * [Client Certificate Validation](client-cert-validation)
  * [Basic Authentication](basic-auth) is a tutorial on how to use the external authentication API to code your own authentication service.
  * [Basic Rate Limiting](rate-limiting-tutorial)
  * [Advanced Rate Limiting](advanced-rate-limiting)
  * [Single Sign-On with OAuth and OpenID Connect](oauth-oidc-auth)