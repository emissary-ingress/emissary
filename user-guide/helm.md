# Installing Ambassador with Helm

[Helm](https://helm.sh) is a package manager for Kubernetes. Ambassador is available as a Helm chart if you use Helm for package management. To install with Helm:

1. Install the Ambassador Chart:

   ```
   helm upgrade --install --wait my-release stable/ambassador
   ```
   
   For details on how to configure the chart, see the [official chart documentation](https://hub.helm.sh/charts/stable/ambassador) 


2. Jump to [step 3](/user-guide/getting-started#3-creating-your-first-service) of the Ambassador tutorial to create your first service.

## Notes

- By default, the Helm chart configures Ambassador to [run as non-root](/reference/running#running-as-non-root) and listen on port 8080. Remember to configure the `service_port` in the [Ambassador Module](/reference/modules) if enabling TLS termination in Ambassador.

   To configure Ambassador to run as root, remove `runAsUser: 8888` from the `securityContext` in your `values.yaml`. Removing `service_port: 8080` from the Ambassador `Module` will tell Ambassador to listen on default HTTP and HTTPS ports.

- The value of `ambassador.id` should not be changed unless you wish to run multiple Ambassadors in your cluster. See notes on [AMBASSADOR_ID](/reference/running#ambassador_id) f you set it something other than `default`. 
