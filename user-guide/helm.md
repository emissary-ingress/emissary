# Installing Ambassador with Helm

[Helm](https://helm.sh) is a package manager for Kubernetes. Ambassador is available as a Helm chart if you use Helm for package management. To install with Helm:

1. Install the Ambassador Chart:

   ```
   helm upgrade --install --wait my-release stable/ambassador
   ```
   
   For details on how to configure the chart, see the [official chart documentation](https://hub.helm.sh/charts/stable/ambassador) 


2. Jump to [step 3](/user-guide/getting-started#3-creating-your-first-route) of the Ambassador tutorial to create your first route.
