# Cert-Manager and Ambassador Edge Stack

**Note:** This document assumes cert-manager  v0.15 or greater.   This document has been updated to use CRD standards specified in v0.15.  [Legacy CRD support](https://cert-manager.io/docs/installation/upgrading/upgrading-0.14-0.15/) was removed in cert-manager v0.15, see their [upgrading](https://cert-manager.io/docs/installation/upgrading/) document for more info.

---

Creating and managing certificates in Kubernetes is made simple with Jetstack's [cert-manager](https://github.com/jetstack/cert-manager). Cert-manager will automatically create and renew TLS certificates and store them in Kubernetes secrets for easy use in a cluster. Ambassador will automatically watch for secret changes and reload certificates upon renewal.

**Note:** Ambassador Edge Stack will automatically create and renew TLS certificates with the HTTP-01 challenge. You should use cert-manager if you need support for the DNS-01 challenge and/or wildcard certificates.

## Install Cert-Manager

There are many different ways to [install cert-manager](https://docs.cert-manager.io/en/latest/getting-started/install.html). For simplicity, we will use Helm v3.

1. Create the cert-manager CRDs.
    ```shell
    kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.crds.yaml
    ```

2. Add the `jetstack` Helm repository.
    ```shell
    helm repo add jetstack https://charts.jetstack.io && helm repo update
    ```

3. Install cert-manager.

    ```shell
    kubectl create ns cert-manager
    helm install cert-manager --namespace cert-manager jetstack/cert-manager
    ```

## Issuing Certificates

cert-manager issues certificates from a CA such as [Let's Encrypt](https://letsencrypt.org/). It does this using the ACME protocol which supports various challenge mechanisms for verifying ownership of the domain.

### Issuer

An `Issuer` or `ClusterIssuer` identifies which Certificate Authority cert-manager will use to issue a certificate. `Issuer` is a namespaced resource allowing you to use different CAs in each namespace, a `ClusterIssuer` is used to issue certificates in any namespace. Configuration depends on which ACME [challenge](#challenge) you are using.

### Certificate

A [Certificate](https://cert-manager.readthedocs.io/en/latest/reference/certificates.html) is a namespaced resource that references an `Issuer` or `ClusterIssuer` for issuing certificates. `Certificate`s define the DNS name(s) a key and certificate should be issued for, as well as the secret to store those files (e.g. `ambassador-certs`). Configuration depends on which ACME [challenge](#challenge) you are using.

By duplicating issuers, certificates, and secrets one can support multiple domains with [SNI](../../topics/running/tls/sni).

## Challenge

cert-manager supports two kinds of ACME challenges that verify domain ownership in different ways: HTTP-01 and DNS-01.

### DNS-01 Challenge

The DNS-01 challenge verifies domain ownership by proving you have control over its DNS records. Issuer configuration will depend on your [DNS provider](https://cert-manager.readthedocs.io/en/latest/tasks/acme/configuring-dns01/index.html#supported-dns01-providers). This example uses [AWS Route53](https://cert-manager.readthedocs.io/en/latest/tasks/acme/configuring-dns01/route53.html).

1. Create the IAM policy specified in the cert-manager [AWS Route53](https://cert-manager.readthedocs.io/en/latest/tasks/acme/configuring-dns01/route53.html) documentation.

2. Note the `accessKeyID` and create a `secret` named `prod-route53-credentials-secret` in the cert-manager namespace that has a key value: `secret-access-key` from your AWS IaM credentials.

3. Create and apply a `ClusterIssuer`. 

    ```yaml
    ---
    apiVersion: cert-manager.io/v1alpha2
    kind: ClusterIssuer
    metadata:
      name: letsencrypt-prod
    spec:
      acme:
        email: email@example.com
        server: https://acme-v02.api.letsencrypt.org/directory
        privateKeySecretRef:
          name: letsencrypt-prod
        solvers:
        - selector:
            dnsZones:
              - "myzone.route53.com"
          dns01:
            route53:
              region: us-east-1
              accessKeyID: {accessKeyID}
              hostedZoneID: {Hosted Zone ID} # optional, allows you to reduce the scope of permissions in Amazon IAM
              secretAccessKeySecretRef:
                name: prod-route53-credentials-secret
                key: secret-access-key
    ```

4. Create and apply a `Certificate`. 

    ```yaml
    ---
    apiVersion: cert-manager.io/v1alpha2
    kind: Certificate
    metadata:
      name: myzone.route53.com
      # cert-manager will put the resulting Secret in the same Kubernetes 
      # namespace as the Certificate. You should create the certificate in 
      # whichever namespace you want to configure a Host.
    spec:
      secretName: ambassador-certs
      issuerRef:
        name: letsencrypt-prod
        kind: ClusterIssuer
      commonName: myzone.route53.com
      dnsNames:
      - myzone.route53.com
    ```

5. Verify the secret is created.

    ```shell
    $ kubectl get secrets -n ambassador
    NAME                     TYPE                                  DATA      AGE
    ambassador-certs         kubernetes.io/tls                     2         1h
    ```

### HTTP-01 Challenge

The HTTP-01 challenge verifies ownership of the domain by sending a request for a specific file on that domain. cert-manager accomplishes this by sending a request to a temporary pod with the prefix `/.well-known/acme-challenge/`. To perform this challenge:

1. Create and apply a `ClusterIssuer`.

    ```yaml
    ---
    apiVersion: cert-manager.io/v1alpha2
    kind: ClusterIssuer
    metadata:
      name: letsencrypt-prod
    spec:
      acme:
        email: email@example.com
        server: https://acme-v02.api.letsencrypt.org/directory
        privateKeySecretRef:
          name: letsencrypt-prod
        solvers:
        - http01:
            ingress:
              class: nginx
          selector: {}
    ```

2. Create and apply a `Certificate`.

    ```yaml
    ---
    apiVersion: cert-manager.io/v1alpha2
    kind: Certificate
    metadata:
      name: ambassador-certs
      # cert-manager will put the resulting Secret in the same Kubernetes 
      # namespace as the Certificate. You should create the certificate in 
      # whichever namespace you want to configure a Host.
      namespace: ambassador
    spec:
      secretName: ambassador-certs
      issuerRef:
        name: letsencrypt-prod
        kind: ClusterIssuer
      dnsNames:
      - example.com
    ```

3. Apply both the `ClusterIssuer` and `Certificate`

    After applying both of these YAML manifests, you will notice that cert-manager has spun up a temporary pod named `cm-acme-http-solver-xxxx` but no certificate has been issued. Check the cert-manager logs and you will see a log message that looks like this:

    ```shell
    $ kubectl logs cert-manager-756d6d885d-v7gmg
    ...
    Preparing certificate default/ambassador-certs with issuer
    Calling GetOrder
    Calling GetAuthorization
    Calling HTTP01ChallengeResponse
    Cleaning up old/expired challenges for Certificate default/ambassador-certs
    Calling GetChallenge
    wrong status code '404'
    Looking up Ingresses for selector certmanager.k8s.io/acme-http-domain=161156668,certmanager.k8s.io/acme-http-token=1100680922
    Error preparing issuer for certificate default/ambassador-certs: http-01 self check failed for domain "example.com
    ```

4. Create a Mapping for the `/.well-known/acme-challenge/` route.

    cert-manager uses an `Ingress` to issue the challenge to `/.well-known/acme-challenge/` that is incompatible with Ambassador. We will need to create a `Mapping` so the cert-manager can reach the temporary pod.
 
    ```yaml
    ---
    apiVersion: getambassador.io/v2
    kind: Mapping
    metadata:
      name: acme-challenge-mapping
    spec:
      prefix: /.well-known/acme-challenge/
      rewrite: ""
      service: acme-challenge-service

    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: acme-challenge-service
    spec:
      ports:
      - port: 80
        targetPort: 8089
      selector:
        acme.cert-manager.io/http01-solver: "true"
    ```

    Apply the YAML and wait a couple of minutes. cert-manager will retry the challenge and issue the certificate.

5. Verify the secret is created:

    ```shell
    $ kubectl get secrets
    NAME                     TYPE                                  DATA      AGE
    ambassador-certs         kubernetes.io/tls                     2         1h
    ambassador-token-846d5   kubernetes.io/service-account-token   3         2h
    ```
