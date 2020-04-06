# edgectl install: Timed Out Waiting for the AES Pod to Respond with its Cluster ID
 
Ambassador's `edgectl install` uses `kubectl` to communicate with Kubernetes.  

## The Problem

The installer timed out while waiting for the AES pod to return its cluster ID.  Either the pod doesn't exist or there 
is a problem with your cluster.

## How to Resolve It

...
