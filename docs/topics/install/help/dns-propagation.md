# Edgectl Install: DNS Failed to Propagate to This Host

The installer has acquired a DNS name for your Kubernetes cluster but is unable to see that name from this computer (your developer machine).

The EdgeStack.me DNS server is serving a DNS name for you. The associated DNS record must propagate through intermediate DNS servers on the Internet to the ones serving your local network so that your laptop can see your new DNS name. This usually takes a few minutes, but in some circumstances it may take longer.

## What's next?

Run the installer again:

```shell
edgectl install
```

Don't worry: it is safe to run the installer repeatedly on a Kubernetes cluster.

If running the installer again does not work, please reach out to us on [Slack](http://d6e.co/slack).
