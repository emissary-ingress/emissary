---
title: Single Sign-On using GitHub and Keycloak
category: Security
reading_time: 5 minutes
technologies_used: Ambassador Pro
---

Ambassador Pro integrates with the [Keycloak IDP](https://keycloak.org). Using Keycloak, we can set up Single Sign-On with other OAuth providers such as GitHub. In this tutorial, we'll configure Ambassador Pro to use GitHub credentials for Single Sign-On to the `httpbin` service.

1. <install-ambassador-pro/> // InstallAmbassadorPro component

2. Create an OAuth application in GitHub.
   * Click on your Profile photo, then choose Settings.
   * Click on Developer Settings.
   * Click on "Register a New Application".
     * The Name can be any value.
     * The Homepage URL should be set to your domain name, or you can use `https://example.com` if you're just testing.
     * The Authorization callback uRL should be `https://${AMBASSADOR_IP}/auth/realms/demo/broker/github/endpoint`.
3. Edit your `env.sh` and add the `CLIENT_ID` and `CLIENT_SECRET` from GitHub:

   ```
   CLIENT_ID=<Client ID from GitHub>
   CLIENT_SECRET=<Client Secret from GitHub>
   ```
4. Get the `External-IP` for your Ambassador service `kubectl get svc ambassador`
5. Replace the `${AMBASSADOR_IP}` values in [api-auth-with-github/00-tenant.yaml](00-tenant.yaml)
6. Run `make apply-api-auth`.
7. Go to `https://${AMBASSADOR_IP}/httpbin/headers` in your browser and you will be asked to login. Select the `GitHub` option.

Behind the scenes, we've created an [OAuth authentication Filter](/reference/filter-reference) for requests to `httpbin/headers`. Note that requests to other `httpbin` URLs, e.g., `httpbin/ip` do not require authentication. With this approach, we give users fine-grained control over which specific hosts and paths require authentication, and support using different authentication schemes for different URLs.

## Summary
To quickly enable Single Sign-On for your application, get started with a [free 14-day trial of Ambassador Pro](https://www.getambassador.io/pro/free-trial), or [contact sales](https://www.getambassador.io/contact) today.
