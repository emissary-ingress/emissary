# Edgectl Install: AES Failed to Respond to Queries
 
The installer could not verify that Ambassador Edge Stack (AES) is answering queries. This can happen if AES took longer than expected to start up, or if the load balancer in front of AES is not reachable from this host (the computer on which the installer is running, e.g., your laptop).

## What's next?

1. Verify that the load balancer address is reachable from this host
2. Run the installer again:
   ```shell
   edgectl install
   ```

Don't worry: it is safe to run the installer repeatedly on a cluster.
