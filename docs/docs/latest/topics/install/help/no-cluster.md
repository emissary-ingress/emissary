# edgectl install: Can't communicate with cluster
 
Ambassador's `edgectl install` uses `kubectl` to communicate with Kubernetes.  

Ambassador was unable to communicate with your cluster using your `kubectl` context.  Double-check that your
context is set correctly.  use `kubectl cluster-infl dump` to get more information on your cluster, or
re-install your cluster if needed.

