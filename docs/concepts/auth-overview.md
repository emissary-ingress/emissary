# OAuth and OIDC Overview

The implementation of an effective authentication strategy is vital to any application's security solution, as it is a key part of determining a user's identity, and stopping bad actors from masquerading as others, particularly within parts of your system that access sensitive data.

Typically with web applications, the authentication is implemented at the edge, either via an API/edge gateway or via a top-level request filter within your application framework. It is also increasingly common for applications to use external identity providers -- such as Google, GitHub, or Facebook -- typically via an Identity hub like [Auth0](https://auth0.com/), [Keycloak](https://www.keycloak.org/) or [Okta](https://www.okta.com/) that provides authentication-as-a-service, rather than taking on the high cost (and risk) of maintaining their own identity database.

This article is focused on implementing authentication at the edge with the Kubernetes-native Ambassador Edge Stack and shows an example of how to integrate this with third-party identity providers.

## How Ambassador Edge Stack Integrates with OAuth and OIDC

This is what the authentication process looks like at a high level when using Ambassador Edge Stack with Auth0 as an identity provider. The use case is an end-user accessing a secured app service.

![Ambassador Authentication OAuth/OIDC](../../doc-images/ambassador_oidc_flow.jpg)

There is quite a bit happening in this diagram, and so it will be useful to provide an overview of all of the moving parts.

# OpenID, OAuth, IdPs, OIDC, Oh my!

In software development, we are generally not shy about using lots of acronyms, and the authentication space is no different. There are quite a few acronyms to learn, but the underlying concepts are surprisingly simple. Here's a cheat sheet:

* OpenID: is an [open standard](https://openid.net/) and [decentralized authentication protocol](https://en.wikipedia.org/wiki/OpenID). OpenID allows users to be authenticated by co-operating sites, referred to as "relying parties" (RP) using a third-party authentication service. End-users can create accounts by selecting an OpenID identity provider (such as Auth0, Okta, etc), and then use those accounts to sign onto any website that accepts OpenID authentication.
* Open Authorization (OAuth): an open standard for [token-based authentication and authorization](https://oauth.net/) on the Internet. OAuth provides to clients a "secure delegated access" to server or application resources on behalf of an owner, which means that although you won't manage a user's authentication credentials, you can specify what they can access within your application once they have been successfully authenticated. The current latest version of this standard is OAuth 2.0.
* Identity Provider (IdP): an entity that [creates, maintains, and manages identity information](https://en.wikipedia.org/wiki/Identity_provider) for user accounts (also referred to "principals") while providing authentication services to external applications (referred to as "relying parties") within a distributed network, such as the web.
* OpenID Connect (OIDC): is an [authentication layer that is built on top of OAuth 2.0](https://openid.net/connect/), which allows applications to verify the identity of an end-user based on the authentication performed by an IdP, using a well-specified RESTful HTTP API with JSON as a data format. Typically an OIDC implementation will allow you to obtain basic profile information for a user that successfully authenticates, which in turn can be used for implementing additional security measures like Role-based Access Control (RBAC).
* JSON Web Token (JWT): is a [JSON-based open standard for creating access tokens](https://jwt.io/), such as those generated from an OAuth authentication. JWTs are compact, web-safe (or URL-safe), and are often used in the context of implementing single sign-on (SSO) within federated applications and organizations. Additional profile information, claims, or role-based information can be added to a JWT, and the token can be passed from the edge of an application right through the application's service call stack.

If you look back at the authentication process diagram, the function of the entities involved should now be much clearer.

## Why Use an Identity Hub like Auth0 or Keycloak?

Using an identity hub or broker allows you to support many IdPs without having to code individual integrations with them. For example, [Auth0](https://auth0.com/docs/identityproviders) and [Keycloak](https://www.keycloak.org/docs/latest/server_admin/index.html#social-identity-providers) both offer support for using Google and GitHub as an IdP.

An identity hub sits between your application and the IdP that authenticates your users, which not only adds a level of abstraction so that your application (and Ambassador Edge Stack) is isolated from any changes to each provider's implementation, but it also allows your users to chose which provider they use to authenticate (and you can set a default, or restrict these options).

The Auth0 docs provide a guide for adding social IdP "[connections](https://auth0.com/docs/identityproviders)" to your Auth0 account, and the Keycloak docs provide a guide for adding social identity "[brokers](https://www.keycloak.org/docs/latest/server_admin/index.html#social-identity-providers)".

## Learn More With the Ambassador Edge Stack and Auth0 Tutorial

You can learn more from the [Single Sign-On with OAuth & OIDC](../../user-guide/oauth-oidc-auth) tutorial, which also contains a full walkthrough of how to configure Ambassador Edge Stack with Auth0.
