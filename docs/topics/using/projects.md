# Introduction to the `Project` resource

## This feature is in BETA. Please [tell me](mailto:rhs@datawire.io?subject=Project%20CRD%20UX%20Feedback) (lead developer of the feature) about your experience.

Please note that the Project functionality is disabled by default. See [The Project Quickstart](../../../tutorials/projects/) for instructions on enabling the Project Controller and setting up your first `Project`.

Ambassador is designed around a [declarative, self-service management model](../../concepts/gitops-continuous-delivery). The `Project` resource takes self-service, declarative configuration, and gitops to the next level by enabling developers to stage and deploy code with nothing more than a github repository. 

## The `Project` Resource

A `Project` resource requires the following configuration:

| Required attribute        | Description               |
| :------------------------ | :------------------------ |
| `name`                    | A string identifying the `Project` |
| `host`                    | The hostname at which you would like to publish your `Project` |
| `prefix`                  | The URL prefix at which you would like to publish your `Project` |[resource](#resources) |
| `githubToken`             | A token that has access to the repo with your source code |
| `githubRepo`              | The `owner/name` of the github repo that holds your source code. The repo must have a Dockerfile located in the root that builds your server and runs it on port 8080. |

The `Edge Policy Console` provides a streamlined self-service workflow for creating a `Project` resource, but you can define projects just like any other kubernetes resource:

```
---
apiVersion: getambassador.io/v2
kind: Project
metadata:
  name: <your-project-name>
  namespace: <your-project-namespace>
spec:
  host: <your-hostname>
  prefix: <your-project-prefix> # e.g. /foo/
  githubRepo: <your-github-org>/<your-github-repo>
  githubToken: <your-github-api-token>
```

## `Project` Repositories

The `Project` Controller expects `Project` repositories to have a `Dockerfile` in the root of the referenced github repo:

```
<root>
  |
  +-- Dockerfile                  // Required: tells the controller how to build your project
  |
  +-- project-revision.yaml.tmpl  // Optional: allows customization of kubernetes resources
  |
  +-- ...
```

This `Dockerfile` will be used to build and deploy the `Project`. The `Dockerfile` MUST include an `EXPOSE 8080` instruction, and the server's code MUST be written to bind to port 8080:

```
FROM <your-base-image>
...
RUN <your-build-instructions>
...
EXPOSE 8080 # PLEASE NOTE: Your code must bind to port 8080 also!!
CMD <your-server>
```

The `project-revision.yaml.tmpl` is optional. When present, the contents of this file are evaluated as a [golang template](https://golang.org/pkg/text/template/), and then used to deploy each revision. See [Customizing Project Deployment](../project-customization) for more details.

## How it Works

The `Project Controller` publishes each `Project` by:

1. Building and deploying the default (usually master) branch of the git repo.
2. Building and staging each open PR at a preview URL. Note that for security reasons, the `Project Controller` will only build PRs whose base branch is in the repo itself. This prevents third party PRs from being used for nefarious purposes.

The `Project Controller` automatically registers a webhook so that it is notified whenever PRs are opened or closed or code changes are pushed. This allows the `Project Controller` to automatically and continuously reconcile the cluster state with the git repo and rebuild/restage each PR as well as rebuild/redeploy the default branch as needed.

For example, if the foo `Project` points to a repo with 3 open feature branches, the foo `Project` will stage 3 commits and deploy 1 commit as depicted below:

```
.          <<Project>>    <====>     <<Repo>>
               foo             github.com/octocat/foo
                |                        |
                |                  +-----+-----+
                |                  |           |
                |                  . PRs       . Branches
                |                  .           .
                |                  .           .
                |                  .        master ---------+
                |                PR#1 ---> feature-1 -----+ |
                |                PR#2 ---> feature-2 ---+ | |
                |                PR#3 ---> feature-3 -+ | | |
                |                                     | | | | Commits
               \|/                                    | | | |
+----------ProjectRevision_3----------> 3234abc... <--+ | | |
| +--------ProjectRevision_2----------> 2234abc... <----+ | |
| | +------ProjectRevision_1----------> 1234abc... <------+ |
| | | +----ProjectRevision_0----------> 0234abc... <--------+
| | | |
| | | |
| | | +--> https://<host>/<prefix>/
| | +----> https://<host>/.preview/<prefix>/1234abc.../
| +------> https://<host>/.preview/<prefix>/2234abc.../
+--------> https://<host>/.preview/<prefix>/3234abc.../
```

## The `ProjectRevision` resource

The `Project` resource accomplishes its goals by delegating to the `ProjectRevision` resource which in turn manages a number of other kubernetes resources. The `Project Controller` will create a `ProjectRevision` for every git commit that is to be staged or deployed:

```
$ kubectl get projectrevisions
NAME              PROJECT   REF                 REV         STATUS         AGE
foo-07d04b...     foo       refs/heads/master   07d04b...   Deployed       2d
foo-19f77a...     foo       refs/pull/8/head    19f77a...   Building       25s
foo-3d88e5...     foo       refs/pull/12/head   3d88e5...   Deployed       4h
foo-5664e9...     foo       refs/pull/11/head   5664e9...   Deploying      65s
```

Each `ProjectRevision` will create a `Job` to perform the build, and a `Deployment` + `Service` + [`Mapping`](#mapping) to publish the commit:

```
Project
   | 
   +---> ProjectRevision_1
   |       |
   |       +-> Job (builds and pushes the image)
   |       |
   |       |
   |       +---> Deployment (runs the image)
   |       |        /|\
   |       |         |
   |       +-----> Service
   |       |           /|\
   |       |            |
   |       +--------> Mapping (publishes the image at /.preview/... or /<prefix>)
   |
   +...
   |
   +---> ProjectRevision_N
```

A `ProjectRevision` progresses through different phases in its lifecycle as it attempts to build and run your server. The `phase` of a `ProjectRevision` (stored in the status.phase field) tells us exactly what it is doing and what has happened:

| Phase        | Description               |
| :------------| :------------------------ |
| Received     | The initial state of a ProjecCommit when it is first created. |
| BuildQueued  | Waiting for other builds to finish. |
| Building     | The build is in progress. |
| BuildFailed  | The build has failed. This is a terminal state. |
| Deploying    | The commit has been succesfully built and the deploy is in progress.
| DeployFailed | The commit has been succesfully built but the deploy has failed. This is a terminal state. |
| Deployed     | The commit has been succesfully built and deployed. This is a terminal state.

There are three terminal states: `BuildFailed`, `DeployFailed`, and
Deployed. Every `ProjectRevision` will progress through its lifecycle
until it reaches one of these stages.

The resources managed by each project are prefixed by the project name and the commit sha so that you can easily drill down and examine e.g. the build or server logs using `kubectl`. The `Edge Policy Console` provides a live streaming in-browser terminal for viewing build and server logs as well as the full state of each `ProjectRevision` resource.
