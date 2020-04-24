# Edgectl Install: Error processing downloaded information

The installer retrieves information required to install AES over the Internet. Some of that information seems to have been corrupted in transit.

## What's next?

1. Restore Internet connectivity. Perhaps there is a web proxy interfering with access to the Internet.

   Try `curl -ISsf https://www.getambassador.io/` to verify that your computer can reach important websites. Successfully reaching the Internet will show something like

   ```shell
   HTTP/2 200
   cache-control: public, max-age=0, must-revalidate
   content-length: 0
   content-type: text/html; charset=UTF-8
   [...]
   ```

   The exact results are not important as long as "200" shows up as the return code in the first line, which it does in this example.

   A result that looks like

   ```shell
   curl: (7) Failed to connect to 167.170.215.127 port 443: Network is unreachable
   ```

   indicates that there is something preventing you from reaching the Internet.

2. Run the installer again:
   ```shell
   edgectl install
   ```

Don't worry: it is safe to run the installer repeatedly on a Kubernetes cluster.
