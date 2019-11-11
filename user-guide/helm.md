<div style="border: thick solid red">
<!-- TODO: fix red bordered text -->
This method of installation has not been tested and is not supported at this time.
</div>

# Installing Ambassador Edge Stack with Helm

[Helm](https://helm.sh) is a package manager for Kubernetes. Ambassador Edge Stack is available as a Helm chart if you use Helm for package management. To install with Helm:

1. Install the Ambassador Edge Stack Chart:

   ```
   helm upgrade --install --wait my-release stable/ambassador
   ```

   For details on how to configure the chart, see the [official chart documentation](https://hub.helm.sh/charts/stable/ambassador)


2. Jump to [step 3](/user-guide/getting-started#3-creating-your-first-service) of the Ambassador Edge Stack tutorial to create your first service.

## Notes

- By default, the Helm chart configures Ambassador Edge Stack to [run as non-root](/reference/running#running-as-non-root) and listen on port 8080. Remember to configure the `service_port` in the [Ambassador Edge Stack Module](/reference/modules) if enabling TLS termination in Ambassador Edge Stack.

   To configure Ambassador Edge Stack to run as root, remove `runAsUser: 8888` from the `securityContext` in your `values.yaml`. Removing `service_port: 8080` from the Ambassador Edge Stack `Module` will tell Ambassador Edge Stack to listen on default HTTP and HTTPS ports.

- The value of `ambassador.id` should not be changed unless you wish to run multiple Ambassador Edge Stack in your cluster. See notes on [AMBASSADOR_ID](/reference/running#ambassador_id) f you set it something other than `default`.


