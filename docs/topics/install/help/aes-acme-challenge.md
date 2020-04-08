# Edgectl Install: AES Failed to Respond to Queries
 
The installer could not verify that Ambassador is answering queries. This could happen if AES took longer than expected to start up, or if the AES load balancer is not reachable from this host.

## What's next?

1. Verify that the load balancer address is reachable from this host
2. Run the installer again:
   ```shell
   edgectl install
   ```

It is safe to run the installer repeatedly on a cluster.
