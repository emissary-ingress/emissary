# Edgectl Install: Installer Failed to Retrieve Host Resource

After creating a new Host resource using `kubectl apply` the installer was unable to retrieve the Host resource from your Kubernetes cluster. This is unexpected.

## What's next?

If this appears to be an intermittent failure, try running the installer again:

```shell
edgectl install
```

Don't worry: it is safe to run the installer repeatedly on a cluster.

If running the installer again does not work, please reach out to us on [Slack](http://d6e.co/slack).
