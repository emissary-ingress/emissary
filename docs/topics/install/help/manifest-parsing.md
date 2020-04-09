# Edgectl Install: Error processing downloaded information

The installer retrieves information required to install AES over the Internet. Some of that information seems to have been corrupted in transit.

## What's next?

1. Restore Internet connectivity. Perhaps there is a web proxy interfering with access to the Internet.

   Try `curl -ISsf https://www.getambassador.io/` to verify that your computer can reach important websites. This command will show "200" in the first few lines upon success.

2. Run the installer again:
   ```shell
   edgectl install
   ```

Don't worry: it is safe to run the installer repeatedly on a cluster.
