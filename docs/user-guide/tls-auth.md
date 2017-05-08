---
layout: doc
weight: 3
title: "TLS and Auth"
categories: user-guide
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

The standard doesn't _require_ the server to always present a certificate, but it has become a de facto requirement in current practice. On the other hand, clients usually _don't_ present certificates unless their user has explicitly told them to do so. This is partly because of the annoyance of generating certificates, and partly because the Web grew up with an idea that you shouldn't need to say who you are without a good reason to do so, and while servers nearly always have the good reason of reassuring their clients that they've found the correct server, things are different for clients.

# Certificates

Certificates in TLS obey the X.509 standard, which is part of the gargantuan sprawling suite of standards collectively called X.500. If you really, really want to, you can read about X.509 on [Wikipedia](https://en.wikipedia.org/wiki/X.509) -- but the fundamental ideas of certificates are simple enough, even if the standard is not.

A certificate is basically like an ID: it's a way of asserting that the identity of the entity holding the certificate is something you can rely on. To follow what's up with them, you need to understand... some stuff.

## Certificates are based on public-key cryptography.

Certs have a public part and a private part. If you own a cert, you'll show everyone the public part, but you need to keep the private part secret.

This matters a lot in two ways: first, in the handshake above, whenever we say "present a certificate", there's some heavy crypto math going on there. You need both the public and private parts to be able to do that math: a microservice that you talk to with HTTPS needs to have access to the private part of its cert, and that access needs to be secure.

Second, one certificate can _sign_ another, which is a way to say that the thing that owns the signing cert is willing to vouch for the signed cert being valid. This is the root of trust in X.509, and again, you need both parts to sign -- but you only need the public parts to verify that the signature is valid.

## Certificates contain identifying information about their holder.

This is called the `Subject` of the certificate in X.509-speak. It's a thing called a Distinguished Name (a `DN`). Here's a valid DN for me:

```
C=US, ST=Massachusetts, L=Boston, O=Datawire, Inc., OU=Ambassador, CN=Flynn/emailAddress=flynn@datawire.io
```

* my Common Name (`CN`) is Flynn, and my email address is `flynn@datawire.io`
* my Organization (`O`) is Datawire, Inc.
* my Organizational Unit (`OU`) is Ambassador -- think "group" or "department" or "project", as appropriate for your situation
* I'm in the United States (Country, `C`), state (`ST`) of Massachusetts, locality (`L`) Boston.
   * It's not "City" because you might need to use something larger or smaller to be meaningful, depending on where you are. In the US, cities are common though.

Here's a DN identifying a server in my Ambassador cluster:

```
C=US, ST=Massachusetts, L=Boston, O=Datawire, Inc., OU=Ambassador, CN=ambassador.test.datawire.io/emailAddress=flynn@datawire.io
```

The only difference is the Common Name: for a host, the `CN` **must** be the fully-qualified DNS name of the host. The only exception is the so-called _wildcard_, which looks like `CN=*.test.datawire.io` and means "any host in the test.datawire.io domain".

## Certificates exist in a strict hierarchy of trust.

X.509 mandates a model where every certificate is _issued_ by a _certifying authority_ (a _CA_). A CA has a certificate that identifies it (of course), and "issuing" means:

* The CA uses its certificate to sign the cert that it's issuing.
* The `Issuer` of the cert being issued gets set to the `Subject` of the CA's cert.

This means that every cert will have both a `Subject` and an `Issuer`... including the CA's cert, which must have been issued by some CA, which has a cert which must have been issued by some CA, which has a cert... and we break this infinite recursion using a cert which has issued itself, and there has its `Subject` equal to its `Issuer`. This is, unsurprisingly, called a _self-signed_ cert.

At the top of the trust hierarchy are some self-signed certificates that belong to a relatively small number of organizations out there in the world whose function is to be trustworthy certifying authorities. These are called the "Trusted Root CAs", and their certificates are included in OS distributions.

* Modern TLS implementations consider a certificate valid if and _only_ if:
   * the cert is correctly constructed
   * the cert's Subject matches the entity that presented the cert
   * the Issuer's signatures match not just for the cert in question, but for _every_ cert in the trust chain, all the way up to a trusted root CA.

In practice, this means that to make TLS work, you can either get a certificate directly from a trusted root CA, or you can create your own CA _and_ get the root CA cert for your own CA onto _every_ system that you work with. Thankfully, [Let's Encrypt](https://www.letsencrypt.org/) makes it quite easy to use a trusted root CA now.

## Chains vs Certs

If every certificate in the world had to be directly signed by a trusted root CA, things would get unwieldy quickly. In practice, your cert will be signed by one or more _intermediate CA certs_ that form a chain of signatures up to the root CA. To work with these things, your software needs to be set up to present not just its own cert, but all necessary intermediate CA certs.

Fortunately, modern software lets you just concatenate the public parts of all the relevant certificates and have the whole lot of them presented at once.

# Managing the DNS

TLS is extremely closely tied to the DNS, so in addition to all the cert stuff, you **must** get the DNS correct for TLS to work. For Ambassador, this means that you **must** set up the Ambassador service in your Kubernetes cluster with a stable DNS name as discussed in [Running Ambassador](running.md).

Mishandling the DNS is a very common source of problems with TLS, so be careful here! To recap a couple of points discussed in [Running Ambassador](running.md): it's not important whether you use a `CNAME` or an `A` record, but it _is_ important that your cert's `CN` match the record you use, and that there's a valid `PTR` in play.

## Do Not Delete the Service

Again: don't delete Ambassador's Kubernetes service unless you really want the L4 load balancer behind Ambassador's external appearance to go away. If you do, you'll have to re-point the DNS when you recreate the service.

# Using TLS for Client Auth

If you want to use TLS client-certificate authentication, you **must** enable TLS, since the client cert will be promoted only as part of the TLS handshake.

**NOTE WELL** that once enabled, client certs are _mandatory_ for all Ambassador services. We'll be improving on this in a later release.
