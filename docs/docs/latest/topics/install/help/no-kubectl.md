# edgectl install: kubectl must be installed
 
Ambassador's `edgectl install` uses `kubectl` to communicate with Kubernetes.  

## The Problem

For some reason it was unable to be found.

## How to Resolve It

Be sure that you have the latest release of Kubernetes installed, `kubectl` is in your PATH, and that
it is executable.
