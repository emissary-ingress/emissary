---
layout: doc
weight: 3
title: "TLS and Auth"
categories: about
---

[Transport Layer Security](https://en.wikipedia.org/wiki/Transport_Layer_Security) (TLS) is the most standard mechanism for encrypting data on the Web. We **strongly** recommend using it for Ambassador (and it's required to use TLS client certificate authentication)... but unfortunately, it can be a bit of a challenge to set up correctly.

The challenges mostly arise because TLS relies extensively on certificates (which are often misunderstood) and on the DNS (which is oddly easy to get wrong).

# TLS Handshake

The basic TLS handshake goes like this:

1. The client connects to the server, and they agree on which encryption mechanism they'll use.
2. The client tells the server which hostname it's trying to talk to.
3. The server presents a certificate vouching for its identity (the _server cert_)
4. The client may drop the connection if it doesn't trust the server cert.
5. The client may present a certificate vouching for _its_ identity (the _client cert_).
6. The server may drop the connection if it doesn't trust the client cert.
7. Assuming all is well, the client and server proceed to exchange encrypted data.

# Certificates

Certificates in TLS obey the [X.509](https://en.wikipedia.org/wiki/X.509) standard. Most of the X. standards are big, complex, sprawling things, and X.509 is no exception -- fortunately, the basics of certificates don't have to be complex. A certificate is basically like an ID: it's a way of asserting that the identity of the entity holding the certificate is something you can rely on.

In the real world, an ID includes some identifying information - a name and a photo, for example - and we trust IDs because they're made to be hard to forge, and they're issued by agencies like the DMV that, at least in theory, can vouch for whether the ID is valid and matches its holder. 

In TLS, certificates have exactly the same properties. The identifying information is called the _Subject Name_. The certificate also contains an _Issuer Name_, which tells you who's prepared to vouch for the certificate. Being hard to forge is a matter of public-key cryptography.

Both the Subject Name and the Issuer Name are [X.509 Distinguished Names](https://www.ibm.com/support/knowledgecenter/en/SSFKSJ_7.5.0/com.ibm.mq.sec.doc/q009860_.htm). Distinguished Names can contain a ridiculous amount of data, but for certificates the important bits are:

- several items giving the physical location of the entity named
- the organization and organization unit (think "department" or "group")
- the Common Name of the entity
- the email address of the entity

Common Name (the `CN`) is the most important:

- For cert identifying a host, the `CN` **must** be the DNS name of the host. The only exception is the so-called _wildcard_: this looks like e.g. `*.example.com` and means "any host in the `example.com` domain".

- For cert identifying a person (usually a client cert), the `CN` **must** be the name of the person.

## The Trust Chain

## Chains vs Certs

# Managing the DNS

TLS is extremely closely tied to the DNS, so in addition to all the cert stuff, you **must** get the DNS correct for TLS to work. This means that you **must** set up the Ambassador service in your Kubernetes cluster with a stable DNS name. The easiest way to do this is:

1. Create the Ambassador service in Kubernetes.
2. Use `kubectl describe service ambassador` to get the external IP or hostname of the Ambassador service (Ambassador's _external appearance_).
3. Create a `CNAME` or `A` record for Ambassador's external appearance.
4. **Do not delete Ambassador's Kubernetes service**, even if you need to delete its Kubernetes deployment.

## Create the Service

The simplest way to create the Ambassador service is

```
kubectl apply -f ambassador-https.yaml
```

This YAML file will create a service with type `LoadBalancer`, which will in turn ask Kubernetes to create an L4 load balancer that will later be used to talk to Ambassador. This load balancer _should_ persist until the service is deleted, independent of the Ambassador deployment, so creating it first and not deleting it allows you to acquire and retain the stable external appearance to which can associate a DNS record.

## Determining the External Appearance

If you're running Ambassador on AWS, its external appearance is likely to be a hostname. If you're running on GKE, you're more likely to find an IP address. In either case,

```
kubectl describe service ambassador
```

will show you what's up.

## Creating the DNS Record

Actually getting a record into DNS for your setup is, sadly, beyond the scope of this article -- check with your local DNS administrator for advice.

If you _are_ the local DNS administrator, you can use either a `CNAME` or an `A` record. It's not critical to have a matching `PTR` for a `CNAME` -- just make sure that the `A` record the `CNAME` points to _does_ have a correct `PTR`. 

## Do Not Delete the Service

Again: don't delete Ambassador's Kubernetes service unless you really want the L4 load balancer behind Ambassador's external appearance to go away. If you do, you'll have to re-point the DNS when you recreate the service.



Once created, you'll be able to set up your DNS to associate a DNS name with this service, which will let you request the cert. Sadly, setting up your DNS and requesting a cert are a bit outside the scope of this README -- if you don't know how to do this, check with your local DNS administrator! (If you _are_ the domain admin and are just hunting a CA recommendation, check out [Let's Encrypt](https://letsencrypt.org/).)

Once you have the cert, you can run

```
sh scripts/push-cert $FULLCHAIN_PATH $PRIVKEY_PATH
```

where `$FULLCHAIN_PATH` is the path to a single PEM file containing the certificate chain for your cert (including the certificate for your Ambassador and all relevant intermediate certs -- this is what Let's Encrypt calls `fullchain.pem`), and `$PRIVKEY_PATH` is the path to the corresponding private key. `push-cert` will push the cert into Kubernetes secret storage, for Ambassador's later use.

### Without TLS

If you really, really cannot use TLS, you can do

```
kubectl apply -f ambassador-http.yaml
```

for HTTP-only access.

### Using TLS for Client Auth

If you want to use TLS client-certificate authentication, you'll need to tell Ambassador about the CA certificate chain to use to validate client certificates. This is also best done before starting Ambassador. Get the CA certificate chain - including all necessary intermediate certificates - and use `scripts/push-cacert` to push it into a Kubernetes secret:

```
sh scripts/push-cacert $CACERT_PATH
```