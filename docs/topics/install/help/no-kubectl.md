# Edgectl Install: `kubectl` not found

Installation of Ambassador Edge Stack (AES) requires the `kubectl` command. The installer did not find `kubectl` in your shell PATH.

## What's next?

1. Install `kubectl`: see the [Kubernetes documentation on how to install `kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/).
2. Start the installer again:
   ```shell
   edgectl install
   ```

It is safe to run the installer repeatedly on a cluster.

## What is `kubectl`?

`kubectl` is a command line tool for controlling Kubernetes clusters. See the [Kubernetes documentation about `kubectl`](https://kubernetes.io/docs/reference/kubectl/overview/) for more information.
