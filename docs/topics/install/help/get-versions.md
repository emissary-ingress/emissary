# Edgectl Install: Can't communicate with the Kubernetes cluster

Installation of AES requires a Kubernetes cluster. The installer was unable to access your cluster using `kubectl`.

## Details

The installer uses the current `kubectl` configuration and context to access your cluster.

* Set the environment variable `KUBECONFIG` to use a configuration file other than the default
* Use `kubectl config current-context` and `kubectl config set-context` to view or set the current context from among the contexts defined in the configuration file
* Use `kubectl version` to see version information for the cluster specified by the current configuration and context

See the [Kubernetes documentation about `kubectl`](https://kubernetes.io/docs/reference/kubectl/overview/) for more information.

## Why Kubernetes?

See the [Kubernetes Overview](https://kubernetes.io/docs/concepts/overview/what-is-kubernetes/) for more information on what Kubernetes is and why you may wish to use it.

If you don't have a Kubernetes cluster, various cloud providers make it easy to create one, and local options allow you to try Kubernetes on your own workstation (with limitions). See the [Kubernetes documentation on Getting Started](https://kubernetes.io/docs/setup/) for a number of local cluster options and [this crowdsourced list on GitHub](https://github.com/ramitsurana/awesome-kubernetes#publicprivate-cloud) for a links to managed providers.
