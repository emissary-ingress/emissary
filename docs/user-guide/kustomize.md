# Installing Ambassador with Kustomize

[Kustomize](https://kustomize.io) traverses a Kubernetes manifest to add, remove or update configuration options without forking. Ambassador provides a base manifest that you can use to deploy to your cluster with an optional step of overlaying with additional configuration. To install with Kustomize:

1. Create a `kustomization.yaml` in your `config` folder:

   ```yaml
   apiVersion: kustomize.config.k8s.io/v1beta1
   kind: Kustomization

   resources:
   # use this if you don't need rbac
   # github.com/datawire/ambassador/docs/yaml/kustomize/base?ref=%version%
   - github.com/datawire/ambassador/docs/yaml/kustomize/rbac?ref=%version%
   ```

   `kubectl apply -k config`

   For details on how to use kustomize, see the [official kubectl documentation](https://kubectl.docs.kubernetes.io/pages/app_customization/introduction.html)


2. Jump to [step 3](/user-guide/getting-started#3-creating-your-first-service) of the Ambassador tutorial to create your first service.