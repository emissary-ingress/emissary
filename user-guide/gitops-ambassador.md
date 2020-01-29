# Implementing GitOps with Ambassador Edge Stack

Because all of the Ambassador Edge Stack's configuration is described via annotations in Kubernetes YAML files, it is very easy to implement a "GitOps" style workflow -- in fact, if a team is already following this way of working for deploying applications and configurations, no additional machinery or set up should be required.

## Continuous Delivery with Ambassador Edge Stack and GitOps

"[GitOps](https://www.weave.works/technologies/gitops/)" is the name given by the Weaveworks team for how they use developer tooling to drive operations and to implement continuous delivery. GitOps is implemented by using the Git distributed version control system (DVCS) as a single source of truth for declarative infrastructure and applications.

## How Does GitOps Work?

Every developer within a team can issue pull requests against a Git repository, and when merged, a "diff and sync" tool detects a difference between the intended and actual state of the system. Tooling can then be triggered to update and synchronize the infrastructure to the intended state.

Using the GitOps practices, automated build/delivery pipelines detect and roll out changes to infrastructure when changes are made to Git. This practice does not enforce specific tools or products, and instead only requires certain functionality (such as the "diff and sync") to be provided by the chosen tools. As the Ambassador Edge Stack relies on declarative configuration, it is fully compatible with integration into a GitOps workflow.

## Developer Workflow with GitOps

The Datawire interpretation of the guidelines for Weaveworks' implementation of GitOps, which uses containers and Kubernetes for deployment, includes:

1. Everything within the software system that can be described as code must be stored in Git: By using Git as the source of truth, it is possible to observe a cluster and compare it with the desired state. The goal is to describe and version control all aspects of a system: code, configuration, routing, security policies, rate limiting, and monitoring/alerting.
2. The `kubectl` Kubernetes CLI tool should not be used directly: As a general rule, it is not a good practice to deploy directly to the cluster using kubectl (in the same regard as it is not recommended to manually deploy locally built binaries to production).
    * The Weaveworks team argue that many people let their CI tool drive deployment, and by doing this they are not practicing good separation of concerns,
    * Deploying all changes (code and config) via a pipeline allows verification and validation, for example, a pipeline can check for potential route naming collisions, or an invalid security policy
3. Automate the "diff and sync" of codified required state within git and the associated actual state of the system: As soon as the continually executed "diff" process detects that either an automated process merges an engineer's changeset or the cluster state deviates from the current specification, a "sync" should be triggered to converge the actual state to what is specified within the git-based single source of truth.
    * Weavework uses a Kubernetes controller that follows an "[operator pattern](https://coreos.com/blog/introducing-operators.html)": By extending the functionality offered by Kubernetes, using a custom [controller](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) that follows the operator pattern, the cluster can be configured to always stay in sync with the Git-based 'source of truth'.
    * The Weaveworks team uses "diff" and "sync" tools such as the open-source [kubediff,](https://github.com/weaveworks/kubediff) as well as internal tools like "terradiff" and "ansiblediff" (for Terraform and Ansible, respectively), that compare the intended state cluster state with the actual state.
    * The [AppDirect engineering team](https://blog.getambassador.io/fireside-chat-with-alex-gervais-accelerating-appdirect-developer-workflow-with-ambassador-7586597b1c34) writes Ambassador Edge Stack configurations within each team's Kubernetes service YAML manifests. These are stored in git and follow the same review/approval process as any other code unit, and the CD pipeline listens on changes to the git repo and applies the diff to Kubernetes