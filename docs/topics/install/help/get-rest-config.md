# Edgectl Install: Can't communicate with the Kubernetes cluster

Installation of Ambassador Edge Stack (AES) requires a Kubernetes cluster. The installer was unable to access your Kubernetes cluster using `kubectl`.

## Details

The installer uses the current `kubectl` configuration and context to access your Kubernetes cluster.

* If you are using a non-default Kubernetes configuration file, remember to set your `KUBECONFIG` environment variable.
* Use `kubectl config current-context` and `kubectl config set-context` to view or set the current context from among the contexts defined in the configuration file
* Use `kubectl version` to see version information for the cluster specified by the current configuration and context

See the [Kubernetes documentation about `kubectl`](https://kubernetes.io/docs/reference/kubectl/overview/) for more information.

## If you don't have a Kubernetes cluster...

If you don't have a Kubernetes cluster, various cloud providers make it easy to create one, and local options allow you to try Kubernetes on your own workstation (with limitions). See the [Kubernetes documentation on Getting Started](https://kubernetes.io/docs/setup/) for a number of local cluster options and [this crowdsourced list on GitHub](https://github.com/ramitsurana/awesome-kubernetes#publicprivate-cloud) for a links to managed providers.
