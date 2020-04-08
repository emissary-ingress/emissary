# `edgectl install`: Installer Failed to Retrieve Host Resource

Ambassador's `edgectl install` uses `kubectl` to communicate with Kubernetes.  

## The Problem

After creating a new Host resource using `kubectl apply` with the Host manifest, the installer was unable
to retrieve the Host resource from your cluster.  This is unexpected.

## How to Resolve It

...
