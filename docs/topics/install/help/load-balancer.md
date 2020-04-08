# Edgectl Install: No address for the load balancer service
 
The installer could not retrieve an address (DNS name or IP address) for the load balancer that enables traffic from outside the Kubernetes cluster to reach Ambassador. 

## What's next?

* Provisioning a load balancer might be taking longer than expected.
  1. Check whether a load balancer address has appeared:
     ```shell
     kubectl get -n ambassador service ambassador -o "go-template={{range .status.loadBalancer.ingress}}{{or .ip .hostname}}{{end}}"
     ```
  2. Run the installer again:
     ```shell
     edgectl install
     ```
* If your Kubernetes cluster doesn't support load balancers, you'll need to expose AES in some other way. See the [Bare Metal Installation Guide](https://www.getambassador.io/docs/latest/topics/install/bare-metal/) for some options.
