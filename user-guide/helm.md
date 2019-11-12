<div style="border: thick solid red">
<!-- TODO: fix red bordered text -->
This method of installation has not been tested and is not supported at this time.
</div>

# Installing Ambassador Edge Stack with Helm

```Note: These instructions do not work with Minikube.```

[Helm](https://helm.sh) is a package manager for Kubernetes. Ambassador Edge Stack is available as a Helm chart if you use Helm for package management. To install with Helm:

1. Install the Ambassador Edge Stack Chart:

   ```
   helm install -n ambassador stable/ambassador
   ```
   
   Details on how to configure the chart, see the [official chart documentation](https://hub.helm.sh/charts/stable/ambassador)


2. Jump to [step 3](/user-guide/getting-started#3-creating-your-first-service) of the Ambassador Edge Stack tutorial to create your first service.
