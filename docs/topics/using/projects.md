# Introduction to the `Project` resource

Ambassador is designed around a [declarative, self-service management model](../../concepts/gitops-continuous-delivery). The `Project` resource takes self-service, declarative configuration, and gitops to the next level by enabling developers to stage and deploy code with nothing more than a github repository. See [The Project Quickstart](../../tutorials/projects) to setup your first `Project`.

## The `Project` Resource

A `Project` resource requires the following configuration:

| Required attribute        | Description               |
| :------------------------ | :------------------------ |
| `name`                    | A string identifying the `Project` |
| `host`                    | The hostname at which you would like to publish your `Project` |
| `prefix`                  | The URL prefix at which you would like to publish your `Project` |[resource](#resources) |
| `githubToken`             | A token that has access to the repo with your source code |
| `githubRepo`              | The `owner/name` of the github repo that holds your source code. The repo must have a Dockerfile located in the root that builds your server and runs it on port 8080. |

**Note:** The `Edge Policy Console` provides a streamlined self-service workflow for creating a `Project` resource.

## How it Works

The `Project Controller` publishes each `Project` by:

1. Building and deploying the default (usually master) branch of the git repo.
2. Building and staging each open PR at a preview URL.

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
+----------ProjectCommit_3 -----------> 3234abc... <--+ | | |
| +--------ProjectCommit_2 -----------> 2234abc... <----+ | |
| | +------ProjectCommit_1 -----------> 1234abc... <------+ |
| | | +----ProjectCommit_0 -----------> 0234abc... <--------+
| | | |
| | | |
| | | +--> https://<host>/<prefix>/
| | +----> https://<host>/.preview/<prefix>/1234abc.../
| +------> https://<host>/.preview/<prefix>/2234abc.../
+--------> https://<host>/.preview/<prefix>/3234abc.../
```

## The `ProjectCommit` resource

The `Project` resource accomplishes its goals by delegating to the `ProjectCommit` resource which in turn manages other kubernetes resources. The `Project Controller` will create a `ProjectCommit` for every git commit that is to be staged or deployed:

```
$ kubectl get projectcommits
NAME              PROJECT   REF                 REV         STATUS         AGE
foo-07d04b...     foo       refs/heads/master   07d04b...   Deployed       2d
foo-19f77a...     foo       refs/pull/8/head    19f77a...   Building       25s
foo-3d88e5...     foo       refs/pull/12/head   3d88e5...   Deployed       4h
foo-5664e9...     foo       refs/pull/11/head   5664e9...   Deploying      65s
```

Each `ProjectCommit` will create a `Job` to perform the build, and a `Deployment` + `Service` + [`Mapping`](#mapping) to publish the commit:

```
Project
   | 
   +---> ProjectCommit_1
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
   +---> ProjectCommit_N
```

A `ProjectCommit` progresses through different phases in its lifecycle as it attempts to build and run code. The `phase` of a `ProjectCommit` (stored in the status.phase field) tells us exactly what it is doing and what has happened:

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
Deployed. Every `ProjectCommit` will progress through its lifecycle
until it reaches one of these stages.

The resources managed by each project are prefixed by the project name and the commit sha so that you can easily drill down and examine e.g. the build or server logs using `kubectl`. The `Edge Policy Console` provides a live streaming in-browser terminal for viewing build and server logs as well as the full state of each `ProjectCommit` resource.

## Adding Authentication to your `Project`

Make sure you have configured at least one working [authentication Filter](filters). The [HOWTO section](../../howtos/) has numerous dentailed entries on integrating with specific IDPs.

The following `FilterPolicy` will enable authentication for your `Project`'s production deployment:

```
apiVersion: getambassador.io/v2
kind: FilterPolicy
metadata:
  name: foo
  namespace: default
spec:
  rules:
  - host: <your-hostname>
    path: <your-project-prefix>** # e.g. /foo/**
    filters:
    - name: <your-filter-name>
      namespace: <your-filter-namespace>
```

You can apply the following `FilterPolicy` to enable authentication for your `Project`'s preview deploys. Note that you can use a different authentication filter for previews, and in fact you can omit the project-specific portion of the path if you wish to lock down previews for all `Projects`:

```
apiVersion: getambassador.io/v2
kind: FilterPolicy
metadata:
  name: foo
  namespace: default
spec:
  rules:
  - host: <your-hostname>
    path: /.previews/<your-project-prefix>** # e.g. /.previews/foo/*
    filters:
    - name: <your-filter-name>
      namespace: <your-filter-namespace>
```
