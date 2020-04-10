# Edgectl Install: Existing installation detected

The installer detected that Ambassador is already installed in your Kubernetes cluster.
To avoid causing damage to your current setup, the installer has quit. The installer does
not support upgrades or downgrades at this time.

## What's next?

* Perhaps your installation is ready to go, having been installed in a different manner.
Try `edgectl login` to access the Edge Policy Console running on your existing installation.

* Is `kubectl` talking to the cluster you intended?
  * Set the environment variable `KUBECONFIG` to use a configuration file other than the default
  * Use `kubectl config current-context` and `kubectl config set-context` to view or set the current context from among the contexts defined in the configuration file
  * Use `kubectl version` to see version information for the cluster specified by the current configuration and context
  * Once `kubectl` refers to the intended cluster, you can run the installer again with `edgectl install`

* If you are **absolutely certain** it's safe to do so, you can delete your existing installation
  * Depending on your current installation:
    * Use `helm uninstall RELEASE_NAME` to delete an existing Helm installation.
    * Use `kubectl delete all -l service=ambassador` to delete existing Ambassador OSS installation. 
    * Use `kubectl delete all -l product=aes` to delete existing AES.
    This will also delete all the Ambassador resources shown in the prior step.
  * Use `kubectl delete namespace ambassador` to delete the Kubernetes Services and Deployments for Ambassador
  * Restart the installation with `edgectl install`
