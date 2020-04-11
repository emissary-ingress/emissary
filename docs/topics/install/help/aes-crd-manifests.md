# Edgectl Install: Can't reach the Internet

If Internet connectivity is not available, the installer cannot proceed.

## What's next?

1. Restore Internet connectivity. Perhaps there is a VPN or firewall preventing access to the Internet.

   Try `curl -ISsf https://www.getambassador.io/` to verify that your computer can reach important websites. This command will show "200" in the first few lines upon success.

2. Run the installer again:
   ```shell
   edgectl install
   ```

Don't worry: it is safe to run the installer repeatedly on a Kubernetes cluster.
