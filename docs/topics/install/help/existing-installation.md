# Edgectl Install: Existing installation detected

The installer detected that an existing version of Ambassador is already installed in your Kubernetes cluster. To avoid causing damage to your current setup, the installer has stopped. 

The Edgectl Installer is designed for easy first-time installs, but not for upgrading or downgrading an existing installation. We recommend that you use the [Ambassador Operator](../../aes-operator/) to automatically manage those Day 2 operations (upgrades, etc).

## What's next?

* Perhaps your installation is ready to go, having been installed in a different manner. Try `edgectl login` to access the Edge Policy Console running on your existing installation.

* Is `kubectl` talking to the cluster you intended?
  * Set the environment variable `KUBECONFIG` to use a configuration file other than the default
  * Use `kubectl config current-context` and `kubectl config set-context` to view or set the current context from among the contexts defined in the configuration file
  * Use `kubectl version` to see version information for the cluster specified by the current configuration and context
  * Once `kubectl` refers to the intended cluster, you can run the installer again with `edgectl install`
