# Edgectl Install: Unable to talk to an AES Pod
 
The installer failed to communicate with an Ambassador Edge Stack (AES) Pod in your Kubernetes cluster to validate the installation.

## Investigation

* What is the status of the associated Deployment?
  * `kubectl -n ambassador get deploy ambassador`
  * `kubectl -n ambassador describe deploy ambassador`
* What Pods exist in the ambassador namespace?
  * `kubectl -n ambassador get po`
* If there is an Ambassador Pod, what do the logs tell us?
  * `kubectl -n ambassador logs <name of the pod>`

## What's next?

Please get in touch on [Slack](http://d6e.co/slack).
