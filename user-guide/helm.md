# Installing Ambassador with Helm

[Helm](https://helm.sh) is a package manager for Kubernetes. Ambassador is available as a Helm chart if you use Helm for package management. To install with Helm:

1. Install the Ambassador Chart:

   ```
   helm install -n ambassador stable/ambassador
   ```
   
   Details on how to configure Ambassador using the helm chart can be found the in the Helm chart [README](https://github.com/helm/charts/tree/master/stable/ambassador).


2. Jump to [step 3](/user-guide/getting-started#3-creating-your-first-service) of the Ambassador tutorial to create your first service.
