# Edgectl Install: Unable to talk to an AES Pod
 
The installer failed to communicate with an Ambassador Edge Stack (AES) Pod in your Kubernetes cluster to validate the installation.

## What's next?

This is an unusual failure case, so we will have to help you on our Support channel. Please run these commands and then get in touch on [Slack](http://d6e.co/slack).

* What is the status of the associated Deployment?
  * `kubectl -n ambassador get deploy ambassador`
  * `kubectl -n ambassador describe deploy ambassador`
* What Pods exist in the `ambassador` namespace?
  * `kubectl -n ambassador get po`
* If there is an Ambassador Pod listed above (something like `ambassador-abcd1234-vwxyz`), what has it logged?
  * `kubectl -n ambassador logs <name of the pod>`

Those commands capture the state of things in your Kubernetes cluster. This information will allow us to start from a good spot to help you.

