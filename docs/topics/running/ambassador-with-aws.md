# Ambassador Edge Stack on AWS

The Ambassador Edge Stack is a platform agnostic Kubernetes API gateway. It will run in any distribution of Kubernetes whether it is managed by a cloud provider or on homegrown bare-metal servers.

This document serves as a reference for how different configuration options available when running Kubernetes in AWS. See [Installing Ambassador Edge Stack](../../install) for the various installation methods available.

## tl;dr Recommended Configuration:
There are lot of configuration options available to you when running Ambassador in AWS. While you should read this entire document to understand what is best for you, the following is the recommended configuration when running Ambassador in AWS:

It is recommended to terminate TLS at Ambassador so you can take advantage of all the TLS configuration options available in Ambassador including setting the allowed TLS versions, setting `alpn_protocol` options, enforcing HTTP -> HTTPS redirection, and [automatic certificate management](../host-crd) in the Ambassador Edge Stack.

When terminating TLS at Ambassador, you should deploy a L4 [Network Load Balancer (NLB)](#network-load-balancer-nlb) with the proxy protocol enabled to get the best performance out of your load balancer while still preserving the client IP address.

The following `Service` should be configured to deploy an NLB with cross zone load balancing enabled (see [NLB notes](#network-load-balancer-nlb) for caveat on the cross-zone-load-balancing annotation). You will need to configure the proxy protocol in the NLB manually in the AWS Console.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: ambassador
  namespace: ambassador
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
    service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled: "true"
spec:
  type: LoadBalancer
  ports:
  - name: HTTP
    port: 80
    targetPort: 8080
  - name: HTTPS
    port: 443
    targetPort: 8443
  selector:
    service: ambassador
```

   After deploying the `Service` above and manually enabling the proxy protocol you will need to deploy the following [Ambassador `Module`](../ambassador) to tell Ambassador to use the proxy protocol and then restart Ambassador for the configuration to take effect.

   ```yaml
   apiVersion: getambassador.io/v2
   kind: Module
   metadata:
     name: ambassador
     namespace: ambassador
   spec:
     config:
       use_proxy_proto: true
   ```

   Ambassador will now expect traffic from the load balancer to be wrapped with the proxy protocol so it can read the client IP address.

## AWS load balancer notes

AWS provides three types of load balancers:

### "Classic" Load Balancer (ELB)

The ELB is the first generation AWS Elastic Load Balancer. It is the default type of load balancer ensured by a `type: LoadBalancer` `Service` and routes directly to individual EC2 instances. It can be configured to run at layer 4 or layer 7 of the OSI model. See [What is a Classic Load Balancer](https://docs.aws.amazon.com/elasticloadbalancing/latest/classic/introduction.html) for more details.

* Ensured by default for a `type: LoadBalancer` `Service`
* Layer 4: TCP, TCP/SSL
   * Protocol support
      * HTTP(S)
      * Websockets
      * HTTP/2
   * Connection based load balacing
   * Cannot modify the request
* Layer 7: HTTP, HTTPS
   * Protocol support
      * HTTP(S)
   * Request based load balancing
   * Can modify the request (append to `X-Forwarded-*` headers)
* Can perform TLS termination

**Notes:** 
- While it has been superseded by the `Network Load Balancer` and `Application Load Balancer` the ELB offers the simplest way of provisioning an L4 or L7 load balancer in Kubernetes. 
- All  of the [load balancer annotations](#load-balancer-annotations) are respected by the ELB.
- If using the ELB for TLS termination, it is recommended to run in L7 mode so it can modify `X-Forwarded-Proto` correctly.

### Network Load Balancer (NLB)

The NLB is a second generation AWS Elastic Load Balancer. It can be ensure by a `type: LoadBalancer` `Service` using an annotation. It can only run at layer 4 of the OSI model and load balances based on connection allowing it to handle millions of requests per second. See [What is a Network Load Balancer](https://docs.aws.amazon.com/elasticloadbalancing/latest/network/introduction.html) for more details.

* Can be ensured by a `type: LoadBalancer` `Service`
* Layer 4: TCP, TCP/SSL
   * Protocol support
      * HTTP(S)
      * Websockets
      * HTTP/2
   * Connection based load balacing
   * Cannot modify the request
* Can perform TLS termination

**Notes:** 
- The NLB is the most efficient load balancer capable of handling millions of requests per second. It is recommended for streaming connections since it will maintain the connection stream between the client and Ambassador. 
- Most  of the [load balancer annotations](#load-balancer-annotations) are respected by the NLB. You will need to manually configure the proxy protocol and take an extra step to enable cross zone load balancing.
- Since it operates at L4 and cannot modify the request, you will need to tell Ambassador if it is terminating TLS or not (see [TLS termination](#tls-termination) notes below).

### Application Load Balancer (ALB)

The ALB is a second generation AWS Elastic Load Balancer. It cannot be ensured by a `type: LoadBalancer` `Service` and must be deployed and configured manually. It can only run at layer 7 of the OSI model and load balances based on request information allowing it to perform fine-grained routing to applications. See [What is a Application Load Balancer](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/introduction.html) for more details.

* Cannot be configured by a `type: LoadBalancer` `Service`
* Layer 7: HTTP, HTTPS
   * Protocol support
      * HTTP(S)
   * Request based load balancing
   * Can modify the request (append to `X-Forwarded-*` headers)
* Can perform TLS termination

**Notes:**  

- The ALB can perform routing based on the path, headers, host, etc.. Since Ambassador performs this kind of routing in your cluster, unless you are using the same load balancer to route to services outside of Kubernetes, the overhead of provisioning an ALB is often not worth the benefits. 
- If you would like to use an ALB, you will need to expose Ambassador with a `type: NodePort` service and manually configure the ALB to forward to the correct ports.
- None of the [load balancer annotations](#load-balancer-annotations) are respected by the ALB. You will need to manually configure all options.
- The ALB will properly set the `X-Forward-Proto` header if terminating TLS. See (see [TLS termination](#tls-termination) notes below).

## Load Balancer Annotations

Kubernetes on AWS exposes a mechanism to request certain load balancer configurations by annotating the `type: LoadBalancer` `Service`. You can view all of them in the [Kubernetes documentation](https://kubernetes.io/docs/concepts/cluster-administration/cloud-providers/#load-balancers). This document will go over the subset that is most relevant when deploying Ambassador Edge Stack.

- `service.beta.kubernetes.io/aws-load-balancer-ssl-cert`: 

    Configures the load balancer to use a valid certificate ARN to terminate TLS at the Load Balancer.
    
    Traffic from the client into the load balancer is encrypted but, since TLS is being terminated at the load balancer, traffic from the load balancer to Ambassador Edge Stack will be cleartext. You will need to configure Ambassador differently depending on if the load balancer is running in L4 or L7 (see [TLS termination](#tls-termination) notes below).

- `service.beta.kubernetes.io/aws-load-balancer-ssl-ports`:

    Configures which port the load balancer will be listening for SSL traffic on. Defaults to `"*"`.

    If you want to enable cleartext redirection, make sure to set this to `"443"` so traffic on port 80 will come in over cleartext.

- `service.beta.kubernetes.io/aws-load-balancer-backend-protocol`:

    Configures the ELB to operate in L4 or L7 mode. Can be set to `"tcp"`/`"ssl"` for an L4 listener or `"http"`/`"https"` for an L7 listener. Defaults to `"tcp"` or  `"ssl"` if `aws-load-balancer-ssl-cert` is set.

- `service.beta.kubernetes.io/aws-load-balancer-type: "nlb"`:

    When this annotation is set it will launch a [Network Load Balancer (NLB)](#network-load-balancer-nlb) instead of a classic ELB.
    
- `service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled`:

    Configures the load balancer to load balance across zones. For high availability, it is typical to deploy nodes across availability zones so this should be set to `"true"`.

    **Note:** You cannot configure this annotation and `service.beta.kubernetes.io/aws-load-balancer-type: "nlb"` at the same time. You must first deploy the `Service` with an NLB and then update it with the cross zone load balancing configuration.
    
- `service.beta.kubernetes.io/aws-load-balancer-proxy-protocol`:

    Configures the ELB to enable the proxy protocol. `"*"`, which enables the proxy protocol on all ELB backends, is the only acceptable value.

    The proxy protocol can be used to preserve the client IP address. 

    If setting this value, you need to make sure Ambassador is configured to use the proxy protocol (see [preserving the client IP address](#preserving-the-client-ip-address) below).

    **Note:** This annotation will not be recognized if `aws-load-balancer-type: "nlb"` is configured. Proxy protocol must be manually enabled for NLBs.

## TLS Termination

TLS termination is an important part of any modern web app. Ambassador exposes a lot of TLS termination configuration options that make it a powerful tool for managing encryption between your clients and microservices. Refer to the [TLS Termination](../tls) documentation for more information on how to configure TLS termination at Ambassador.

With AWS, the AWS Certificate Manager (ACM) makes it easy to configure TLS termination at an AWS load balancer using the annotations explained above.

This means that, when running Ambassador in AWS, you have the choice between terminating TLS at the load balancer using a certificate from the ACM or at Ambassador using a certificate stored as a `Secret` in your cluster.

The following documentation will cover the different options available to you and how to configure Ambassador and the load balancer to get the most of each.

### TLS Termination at Ambassador

Terminating TLS at Ambassador will guarantee you to be able to use all of the TLS termination options that Ambassador exposes including enforcing the minimum TLS version, setting the `alpn_protocols`, and redirecting cleartext to HTTPS. 

If terminating TLS at Ambassador, you can provision any AWS load balancer that you want with the following (default) port assignments:

```yaml
spec:
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: https
    port: 443
    targetPort: 8443
```

While terminating TLS at Ambassador makes it easier to expose more advanced TLS configuration options, it does have the drawback of not being able to use the ACM to manage certificates. You will have to manage your TLS certificates yourself or use the [automatic certificate management](../host-crd) available in the Ambassador Edge Stack to have Ambassador do it for you.

### TLS Termination at the Load Balancer

If you choose to terminate TLS at your Amazon load balancer you will be able to use the ACM to manage TLS certificates. This option does add some complexity to your Ambassador configuration, depending on which load balancer you are using.

Terminating TLS at the load balancer means that Ambassador will be receiving all traffic as un-encrypted cleartext traffic. Since Ambassador expects to be serving both encrypted and cleartext traffic by default, you will need to make the following configuration changes to Ambassador to support this:

#### L4 Load Balancer (Default ELB or NLB)

* **Load Balancer Service Configuration:**
   The following `Service` will deploy a L4 ELB with TLS termination configured at the load balancer:
   ```yaml
   apiVersion: v1
   kind: Service
   metadata:
     name: ambassador
     namespace: ambassador
     annotations:
       service.beta.kubernetes.io/aws-load-balancer-ssl-cert: {{ACM_CERT_ARN}}
       service.beta.kubernetes.io/aws-load-balancer-ssl-ports: "443"
   spec:
     type: LoadBalancer
     ports:
     - name: HTTP
       port: 80
       targetPort: 8080
     - name: HTTPS
       port: 443
       targetPort: 8080
     selector:
       service: ambassador
   ```

   Note that the `spec.ports` has been changed so both the HTTP and HTTPS ports forward to the cleartext port 8080 on Ambassador.

* **Host:**
   
   The `Host` configures how Ambassador handles encrypted and cleartext traffic. The following `Host` configuration will tell Ambassador to `Route` cleartext traffic that comes in from the load balancer:

   ```yaml
   apiVersion: getambassador.io/v2
   kind: Host
   metadata:
     name: ambassador
   spec:
     hostname: "*"
     selector:
       matchLabels:
         hostname: wildcard
     acmeProvider:
       authority: none
     requestPolicy:
       insecure:
         action: Route
   ```

**Important:**

Because L4 load balancers do not set `X-Forwarded` headers, Ambassador will not be able to distinguish between traffic that came in to the load balancer as encrypted or cleartext. Because of this, **HTTP -> HTTPS redirection is not possible when terminating TLS at a L4 load balancer**.

#### L7 Load Balancer (ELB or ALB)

* **Load Balancer Service Configuration (L7 ELB):**

   The following `Service` will deploy a L7 ELB with TLS termination configured at the load balancer:
   ```yaml
   apiVersion: v1
   kind: Service
   metadata:
     name: ambassador
     namespace: ambassador
     annotations:
       service.beta.kubernetes.io/aws-load-balancer-ssl-cert: {{ACM_CERT_ARN}}
       service.beta.kubernetes.io/aws-load-balancer-ssl-ports: "443"
       service.beta.kubernetes.io/aws-load-balancer-backend-protocol: "http"
   spec:
     type: LoadBalancer
     ports:
     - name: HTTP
       port: 80
       targetPort: 8080
     - name: HTTPS
       port: 443
       targetPort: 8080
     selector:
       service: ambassador
   ```

   Note that the `spec.ports` has been changed so both the HTTP and HTTPS ports forward to the cleartext port 8080 on Ambassador.

* **Host:**
   
   The `Host` configures how Ambassador handles encrypted and cleartext traffic. The following `Host` configuration will tell Ambassador to `Redirect` cleartext traffic that comes in from the load balancer:

   ```yaml
   apiVersion: getambassador.io/v2
   kind: Host
   metadata:
     name: ambassador
   spec:
     hostname: "*"
     selector:
       matchLabels:
         hostname: wildcard
     acmeProvider:
       authority: none
     requestPolicy:
       insecure:
         action: Redirect
   ```

* **Module:**

   Since a L7 load balancer will be able to append to `X-Forwarded` headers, we need to configure Ambassador to trust the value of these headers. The following `Module` will configure Ambassador to trust a single L7 proxy in front of Ambassador:

   ```yaml
   apiVersion: getambassador.io/v2
   kind: Module
   metadata:
     name: ambassador
     namespace: ambassador
   spec:
     config:
       xff_num_trusted_hops: 1
       use_remote_address: false
   ```

**Note:**

Ambassador uses the value of `X-Forwarded-Proto` to know if the request originated as encrypted or cleartext. Unlike L4 load balancers, L7 load balancers will set this header so HTTP -> HTTPS redirection is possible when terminating TLS at a L7 load balancer.

## Preserving the Client IP Address

Many applications will want to know the IP address of the connecting client. In Kubernetes, this IP address is often obscured by the IP address of the `Node` that is forwarding the request to Ambassador so extra configuration must be done if you need to preserve the client IP address.

In AWS, there are two options for preserving the client IP address.

1. Use a L7 Load Balancer that sets `X-Forwarded-For`

   A L7 load balancer will populate the `X-Forwarded-For` header with the IP address of the downstream connecting client. If your clients are connecting directly to the load balancer, this will be the IP address of your client.

   When using L7 load balancers, you must configure Ambassador to trust the value of `X-Forwarded-For` and not append its own IP address to it by setting `xff_num_trusted_hops` and `use_remote_address: false` in the [Ambassador `Module`](../ambassador):

   ```yaml
   apiVersion: getambassador.io/v2
   kind: Module
   metadata:
     name: ambassador
     namespace: ambassador
   spec:
     config:
       xff_num_trusted_hops: 1
       use_remote_address: false
   ```

   After configuring the above `Module`, you will need to restart Ambassador for the changes to take effect.

2. Use the proxy protocol

   The [proxy protocol](https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt) is a wrapper around an HTTP request that, like `X-Forwarded-For`, lists the IP address of the downstream connecting client but is able to be set by L4 load balancers as well.

   In AWS, you can configure ELBs to use the proxy protocol by setting the `service.beta.kubernetes.io/aws-load-balancer-proxy-protocol: "*"` annotation on the service. You must manually configure this on ALBs and NLBs.

   After configuring the load balancer to use the proxy protocol, you need to tell Ambassador to expect it on the request.

   ```yaml
   apiVersion: getambassador.io/v2
   kind: Module
   metadata:
     name: ambassador
     namespace: ambassador
   spec:
     config:
       use_proxy_proto: true
   ```

   After configuring the above `Module`, you will need to restart Ambassador for the changes to take effect.
   
