# edgectl install: Can't get client and server versions
 
Ambassador's `edgectl install` uses `kubectl` to communicate with Kubernetes.  

## The Problem

Ambassador was unable to get client and server version information from Kubernetes by calling `kubectl version`.
This process failed.

## How to Resolve It

...
