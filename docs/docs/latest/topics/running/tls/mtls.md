# Mutual TLS (mTLS)

Many oganizations have security concerns that require all network traffic 
throughout their cluster be encrypted. With traditional architectures,
this was not that complicated of a requirement since internal network traffic
was fairly minimal. With microservices, we are making many more requests over
the network that must all be authenticated and secured.

In order for services to authenticate with eachother, they will each need to 
provide a certificate and key that the other trusts before establishing a 
connection. This action of both the client and server providing and validating
certificates is referred to as mutual TLS. 

## mTLS with Ambassador

Since Ambassador is a reverse proxy acting as the entry point to your cluster,
Ambassador is acting as the client as it proxies requests to services upstream.

It is trivial to configure Ambassador to simply originate TLS connections as 
the clien to upstream services by setting 
`service: https://{{UPSTREAM_SERVICE}}` in the `Mapping` configuration. 
However, in order to do mTLS with services upstream, Ambassador must also 
have certificates to authenticate itself with the service. 

To do this, we can use the `TLSContext` object to get certificates from a 
Kubernetes `Secret` and use those to authenticate with the upstream service.

```yaml
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: upstream-context
spec:
  hosts: []
  secret: upstream-certs
```

We give it `host: []` since we do not want to use this to terminate TLS
connections from the client. We are just using this to load certificates for
requests upstream.

After loading the certificates, we can tell Ambassador when to use them by
setting the `tls` parameter in a `Mapping`:

```yaml
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: upstream-mapping
spec:
  prefix: /upstream/
  service: upstream-service
  tls: upstream-context
```

Now, when Ambassador proxies a request to `upstream-service`, it will provide
the certificates in the `upstream-certs` secret for authentication when 
encryting traffic.

## Service Mesh

As you can imagine, when you have many services in your cluster all 
authenticating with eachother, managing all of those certificats can become a
very big challenge.

For this reason, many organizations rely on a service mesh for their
service-to-service authentication and encryption. 

Ambassador integrates with multiple service meshes and makes it easy to
configure mTLS to upstream services for all of them. Click the links below to 
see how to configure Ambassador to do mTLS with any of these service meshes:

- [Consul Connect](../../howtos/consul#encrypted-tls)

- [Istio](../../howtos/istio#istio-mutual-tls)