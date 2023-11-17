# Developing Emissary-ingress

Welcome to the Emissary-ingress Community!

Thank you for contributing, we appreciate small and large contributions and look forward to working with you to make Emissary-ingress better.

This document is intended for developers looking to contribute to the Emissary-ingress project. In this document you will learn how to get your development environment setup and how to contribute to the project. Also, you will find more information about the internal components of Emissary-ingress and other questions about working on the project.

> Looking for end user guides for Emissary-ingress? You can check out the end user guides at <https://www.getambassador.io/docs/emissary/>.

After reading this document if you have questions we encourage you to join us on our [Slack channel](https://d6e.co/slack) in the [#emissary-dev](https://datawire-oss.slack.com/archives/CB46TNG83) channel.

- [Code of Conduct](../Community/CODE_OF_CONDUCT.md)
- [Governance](../Community/GOVERNANCE.md)
- [Maintainers](../Community/MAINTAINERS.md)

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Development Setup](#development-setup)
  - [Step 1: Install Build Dependencies](#step-1-install-build-dependencies)
  - [Step 2: Clone Project](#step-2-clone-project)
  - [Step 3: Configuration](#step-3-configuration)
  - [Step 4: Building](#step-4-building)
  - [Step 5: Push](#step-5-push)
  - [Step 6: Deploy](#step-6-deploy)
  - [Step 7: Dev-loop](#step-7-dev-loop)
  - [What should I do next?](#what-should-i-do-next)
- [Contributing](#contributing)
  - [Submitting a Pull Request (PR)](#submitting-a-pull-request-pr)
  - [Pull Request Review Process](#pull-request-review-process)
  - [Rebasing a branch under review](#rebasing-a-branch-under-review)
  - [Fixup commits during PR review](#fixup-commits-during-pr-review)
- [Development Workflow](#development-workflow)
  - [Branching Strategy](#branching-strategy)
  - [Backport Strategy](#backport-strategy)
    - [What if I need a patch to land in a previous supported version?](#what-if-i-need-a-patch-to-land-in-a-previous-supported-version)
    - [What if my patch is only for a previous supported version?](#what-if-my-patch-is-only-for-a-previous-supported-version)
    - [What if I'm still not sure?](#what-if-im-still-not-sure)
  - [Merge Strategy](#merge-strategy)
    - [What about merge commit strategy?](#what-about-merge-commit-strategy)
- [Contributing to the Docs](#contributing-to-the-docs)
- [Advanced Topics](#advanced-topics)
  - [Running Emissary-ingress internals locally](#running-emissary-ingress-internals-locally)
    - [Setting up diagd](#setting-up-diagd)
    - [Changing the ambassador root](#changing-the-ambassador-root)
    - [Getting envoy](#getting-envoy)
    - [Shutting up the pod labels error](#shutting-up-the-pod-labels-error)
    - [Extra credit](#extra-credit)
  - [Debugging and Developing Envoy Configuration](#debugging-and-developing-envoy-configuration)
    - [Ambassador Dump](#ambassador-dump)
  - [Making changes to Envoy](#making-changes-to-envoy)
    - [1. Preparing your machine](#1-preparing-your-machine)
    - [2. Setting up your workspace to hack on Envoy](#2-setting-up-your-workspace-to-hack-on-envoy)
    - [3. Hacking on Envoy](#3-hacking-on-envoy)
    - [4. Building and testing your hacked-up Envoy](#4-building-and-testing-your-hacked-up-envoy)
    - [5. Test Devloop](#5-test-devloop)
    - [6. Protobuf changes](#6-protobuf-changes)
    - [7. Finalizing your changes](#7-finalizing-your-changes)
    - [8. Final Checklist](#8-final-checklist)
  - [Developing Emissary-ingress (Maintainers-only advice)](#developing-emissary-ingress-maintainers-only-advice)
    - [Updating license documentation](#updating-license-documentation)
    - [Upgrading Python dependencies](#upgrading-python-dependencies)
- [FAQ](#faq)
  - [How do I find out what build targets are available?](#how-do-i-find-out-what-build-targets-are-available)
  - [How do I develop on a Mac with Apple Silicon?](#how-do-i-develop-on-a-mac-with-apple-silicon)
  - [How do I develop on Windows using WSL?](#how-do-i-develop-on-windows-using-wsl)
  - [How do I test using a private Docker repository?](#how-do-i-test-using-a-private-docker-repository)
  - [How do I change the loglevel at runtime?](#how-do-i-change-the-loglevel-at-runtime)
  - [Can I build from a docker container instead of on my local computer?](#can-i-build-from-a-docker-container-instead-of-on-my-local-computer)
  - [How do I clear everything out to make sure my build runs like it will in CI?](#how-do-i-clear-everything-out-to-make-sure-my-build-runs-like-it-will-in-ci)
  - [My editor is changing `go.mod` or `go.sum`, should I commit that?](#my-editor-is-changing-gomod-or-gosum-should-i-commit-that)
  - [How do I debug "This should not happen in CI" errors?](#how-do-i-debug-this-should-not-happen-in-ci-errors)
  - [How do I run Emissary-ingress tests?](#how-do-i-run-emissary-ingress-tests)
  - [How do I type check my python code?](#how-do-i-type-check-my-python-code)
  - [How do I get the source code for a release?](#how-do-i-get-the-source-code-for-a-release)

## Development Setup

This section provides the steps for getting started developing on Emissary-ingress. There are a number of prerequisites that need to be setup. In general, our tooling tries to detect any missing requirements and provide a friendly error message. If you ever find that this is not the case please file an issue.

> **Note:** To enable developers contributing on Macs with Apple Silicon, we ensure that the artifacts are built for `linux/amd64`
> rather than the host `linux/arm64` architecture. This can be overriden using the `BUILD_ARCH` environment variable. Pull Request are welcome :).

### Step 1: Install Build Dependencies

Here is a list of tools that are used by the build system to generate the build artifacts, packaging them up into containers, generating  crds, helm charts and for running tests.

- git
- make
- docker (make sure you can run docker commands as your dev user without sudo)
- bash
- rsync
- golang - `go.mod` for current version
- python (>=3.10.9)
- kubectl
- a kubernetes cluster (you need permissions to create resources, i.e. crds, deployments, services, etc...)
- a Docker registry
- bsdtar (Provided by libarchive-tools on Ubuntu 19.10 and newer)
- gawk
- jq
- helm

### Step 2: Clone Project

If you haven't already then this would be a good time to clone the project running the following commands:

```bash
# clone to your preferred folder
git clone https://github.com/emissary-ingress/emissary.git

# navigate to project
cd emissary
```

### Step 3: Configuration

You can configure the build system using environment variables, two required variables are used for setting the container registry and the kubeconfig used.

> **Important**: the test and build system perform destructive operations against your cluster. Therefore, we recommend that you
> use a development cluster. Setting the DEV_KUBECONFIG variable described below ensures you don't accidently perform actions on a production cluster.

Open a terminal in the location where you cloned the repository and run the following commands:

```bash
# set container registry using `export DEV_REGISTRY=<your-registry>
# note: you need to be logged in and have permissions to push
# Example:
export DEV_REGISTRY=docker.io/parsec86

# set kube config file using `export DEV_KUBECONFIG=<dev-kubeconfig>`
# your cluster needs the ability to read from the configured container registry
export DEV_KUBECONFIG="$HOME/.kube/dev-config.yaml"

```

### Step 4: Building

The build system for this project leverages `make` and multi-stage `docker` builds to produce the following containers:

- `emissary.local/emissary` - single deployable container for Emissary-ingress
- `emissary.local/kat-client` - test client container used for testing
- `emissary.local/kat-server` - test server container used for testing

Using the terminal session you opened in step 2, run the following commands

>

```bash
# This will pull and build the necessary docker containers and produce multiple containers.
# If this is the first time running this command it will take a little bit while the base images are built up and cached.
make images

# verify containers were successfully created, you should also see some of the intermediate builder containers as well
docker images | grep emissary.local
```

*What just happened?*

The build system generated a build container that pulled in envoy, the build dependencies, built various binaries from within this project and packaged them into a single deployable container. More information on this can be found in the [Architecture Document](ARCHITECTURE.md).

### Step 5: Push

Now that you have successfully built the containers its time to push them to your container registry which you setup in step 2.

In the same terminal session you can run the following command:

```bash
# re-tags the images and pushes them to your configured container registry
# docker must be able to login to your registry and you have to have push permissions
make push

# you can view the newly tag images by running
docker images | grep <your -registry>

# alternatively, we have two make targets that provide information as well
make env

# or in a bash export friendly format
make export
```

### Step 6: Deploy

Now its time to deploy the container out to your Kubernetes cluster that was configured in step 2. Hopefully, it is already becoming apparent that we love to leverage Make to handle the complexity for you :).

```bash
# generate helm charts and K8's Configs with your container swapped in and apply them to your cluster
make deploy

# check your cluster to see if emissary is running
# note: kubectl doesn't know about  DEV_KUBECONFIG so you may need to ensure KUBECONFIG is pointing to the correct cluster
kubectl get pod -n ambassador
```

🥳 If all has gone well then you should have your development environment setup for building and testing Emissary-ingress.

### Step 7: Dev-loop

Now that you are all setup and able to deploy a development container of Emissary-ingress to a cluster, it is time to start making some changes.

Lookup an issue that you want to work on, assign it to yourself and if you have any questions feel free to ping us on slack in the #emissary-dev channel.

Make a change to Emissary-ingress and when you want to test it in a live cluster just re-run

`make deploy`

This will:

- recompile the go binary
- rebuild containers
- push them to the docker registry
- rebuild helm charts and manifest
- reapply manifest to cluster and re-deploy Emissary-ingress to the cluster

> *Do I have to run the other make targets `make images` or `make push` ?*
> No you don't have to because `make deploy` will actually run those commands for you. The steps above were meant to introduce you to the various make targets so that you aware of them and have options when developing.

### What should I do next?

Now that you have your dev system up and running here are some additional content that we recommend you check out:

- [Emissary-ingress Architecture](ARCHITECTURE.md)
- [Contributing Code](#contributing)
- [Contributing to Docs](#contributing-to-the-docs)
- [Advanced Topics](#advanced-topics)
- [Faq](#faq)

## Contributing

This section goes over how to contribute code to the project and how to get started contributing. More information on how we manage our branches can be found below in [Development Workflow](#development-workflow).

Before contributing be sure to read our [Code of Conduct](../Community/CODE_OF_CONDUCT.md) and [Governance](../Community/GOVERNANCE.md) to get an understanding of how our project is structured.

### Submitting a Pull Request (PR)

> If you haven't set up your development environment then please see the [Development Setup](#development-setup) section.

When submitting a Pull Request (PR) here are a set of guidelines to follow:

1. Search for an [existing issue](https://github.com/emissary-ingress/emissary/issues) or create a [new issue](https://github.com/emissary-ingress/emissary/issues/new/choose).

2. Be sure to describe your proposed change and any open questions you might have in the issue. This allows us to collect historical context around an issue, provide feedback on the proposed solution and discuss what versions a fix should target.

3. If you haven't done so already create a fork of the respository and clone it locally

   ```shell
   git clone <your-fork>
   ```

4. Cut a new patch branch from `master`:

   ```shell
   git checkout master
   git checkout -b my-patch-branch master
   ```

5. Make necessary code changes.

   - Make sure you include test coverage for the change, see [How do I run Tests](#how-do-i-run-emissary-ingress-tests)
   - Ensure code linting is passing by running `make lint`
   - Code changes must have associated documentation updates.
      - Make changes in <https://github.com/datawire/ambassador-docs> as necessary, and include a reference to those changes the pull request for your code changes.
      - See [Contributing to Docs](#contributing-to-the-docs) for more details.

   > Smaller pull requests are easier to review and can get merged faster thus reducing potential for merge conflicts so it is recommend to keep them small and focused.

6. Commit your changes using descriptive commit messages.
   - we **require** that all commits are signed off so please be sure to commit using the `--signoff` flag, e.g. `git commit --signoff`
   - commit message should summarize the fix and motivation for the proposed fix. Include issue # that the fix looks to address.
   - we are "ok" with multiple commits but we may ask you to squash some commits during the PR review process

7. Push your branch to your forked repository:

   > It is good practice to make sure your change is rebased on the latest master to ensure it will merge cleanly so if it has been awhile since you rebased on upstream you should do it now to ensure there are no merge conflicts

   ```shell
   git push origin my-patch-branch
   ```

8. Submit a Pull Request from your fork targeting upstream `emissary/master`.

Thanks for your contribution! One of the [Maintainers](../Community/MAINTAINERS.md) will review your PR and discuss any changes that need to be made.

### Pull Request Review Process

This is an opportunity for the Maintainers to review the code for accuracy and ensure that it solves the problem outlined in the issue. This is an iterative process and meant to ensure the quality of the code base. During this process we may ask you to break up Pull Request into smaller changes, squash commits, rebase on master, etc...

Once you have been provided feedback:

1. Make the required updates to the code per the review discussion
2. Retest the code and ensure linting is still passing
3. Commit the changes and push to Github
   - see [Fixup Commits](#fixup-commits-during-pr-review) below
4. Repeat these steps as necessary

Once you have **two approvals** then one of the Maintainers will merge the PR.

:tada: Thank you for contributing and being apart of the Emissary-ingress Community!

### Rebasing a branch under review

Many times the base branch will have new commits added to it which may cause merge conflicts with your open pull request. First, a good rule of thumb is to make pull request small so that these conflicts are less likely to occur but this is not always possible when have multiple people working on similiar features. Second, if it is just addressing commit feedback a `fixup` commit is also a good option so that the reviewers can see what changed since their last review.

If you need to address merge conflicts then it is preferred that you use **Rebase** on the base branch rather than merging base branch into the feature branch. This ensures that when the PR is merged that it will cleanly replay on top of the base branch ensuring we maintain a clean linear history.

To do a rebase you can do the following:

```shell
# add emissary.git as a remote repository, only needs to be done once
git remote add upstream https://github.com/emissary-ingress/emissary.git

# fetch upstream master
git fetch upstream master

# checkout local master and update it from upstream master
git checkout master
git pull -ff upstream master

# rebase patch branch on local master
git checkout my-patch-branch
git rebase -i master
```

Once the merge conflicts are addressed and you are ready to push the code up you will need to force push your changes because during the rebase process the commit sha's are re-written and it has diverged from what is in your remote fork (Github).

To force push a branch you can:

```shell
git push head --force-with-lease
```

> Note: the `--force-with-lease` is recommended over `--force` because it is safer because it will check if the remote branch had new commits added during your rebase. You can read more detail here: <https://itnext.io/git-force-vs-force-with-lease-9d0e753e8c41>

### Fixup commits during PR review

One of the major downsides to rebasing a branch is that it requires force pushing over the remote (Github) which then marks all the existing review history outdated. This makes it hard for a reviewer to figure out whether or not the new changes addressed the feedback.

One way you can help the reviewer out is by using **fixup** commits. Fixup commits are special git commits that append `fixup!` to the subject of a commit. `Git` provides tools for easily creating these and also squashing them after the PR review process is done.

Since this is a new commit on top of the other commits, you will not lose your previous review and the new commit can be reviewed independently to determine if the new changes addressed the feedback correctly. Then once the reviewers are happy we will ask you to squash them so that we when it is merged we will maintain a clean linear history.

Here is a quick read on it: <https://jordanelver.co.uk/blog/2020/06/04/fixing-commits-with-git-commit-fixup-and-git-rebase-autosquash/>

TL;DR;

```shell
# make code change and create new commit
git commit --fixup <sha>

# push to Github for review
git push

# reviewers are happy and ask you to do a final rebase before merging
git rebase -i --autosquash master

# final push before merging
git push --force-with-lease
```

## Development Workflow

This section introduces the development workflow used for this repository. It is recommended that both Contributors, Release Engineers and Maintainers familiarize themselves with this content.

### Branching Strategy

This repository follows a trunk based development workflow. Depending on what article you read there are slight nuances to this so this section will outline how this repository interprets that workflow.

The most important branch is `master` this is our **Next Release** version and it should always be in a shippable state. This means that CI should be green and at any point we can decided to ship a new release from it. In a traditional trunk based development workflow, developers are encouraged to land partially finished work daily and to keep that work hidden behind feature flags. This repository does **NOT** follow that and instead if code lands on master it is something we are comfortable with shipping.

We ship release candidate (RC) builds from the `master` branch (current major) and also from `release/v{major.minor}` branches (last major version) during our development cycles. Therefore, it is important that it remains shippable at all times!

When we do a final release then we will cut a new `release/v{major.minor}` branch. These are long lived release branches which capture a snapshot in time for that release. For example here are some of the current release branches (as of writing this):

- release/v3.2
- release/v3.1
- release/v3.0
- release/v2.4
- release/v2.3
- release/v1.14

These branches contain the codebase as it was at that time when the release was done. These branches have branch protection enabled to ensure that they are not removed or accidently overwritten. If we needed to do a security fix or bug patch then we may cut a new `.Z` patch release from an existing release branch. For example, the `release/v2.4` branch is currently on `2.4.1`.

As you can see we currently support mutliple major versions of Emissary-ingress and you can read more about our [End-of-Life Policy](https://www.getambassador.io/docs/emissary/latest/about/aes-emissary-eol/).

For more information on our current RC and Release process you can find that in our [Release Wiki](https://github.com/emissary-ingress/emissary/wiki).

### Backport Strategy

Since we follow a trunk based development workflow this means that the majority of the time your patch branch will be based off from `master` and that most Pull Request will target `master`.

This ensures that we do not miss bug fixes or features for the "Next" shippable release and simplifies the mental-model for deciding how to get started contributing code.

#### What if I need a patch to land in a previous supported version?

Let's say I have a bug fix for CRD round trip conversion for AuthService, which is affecting both `v2.y` and `v3.y`.

First within the issue we should discuss what versions we want to target. This can depend on current cycle work and any upcoming releases we may have.

The general rules we follow are:

1. land patch in "next" version which is `master`
2. backport patch to any `release/v{major}.{minor}` branches

So, let's say we discuss it and say that the "next" major version is a long ways away so we want to do a z patch release on our current minor version(`v3.2`) and we also want to do a z patch release on our last supported major version (`v2.4`).

This means that these patches need to land in three separate branches:

1. `master` - next release
2. `release/v3.2` - patch release
3. `release/v2.4` - patch release

In this scenario, we first ask you to land the patch in the `master` branch and then provide separate PR's with the commits backported onto the `release/v*` branches.

> Recommendation: using the `git cherry-pick -x` will add the source commit sha to the commit message. This helps with tracing work back to the original commit.

#### What if my patch is only for a previous supported version?

Although, this should be an edge case, it does happen where the code has diverged enough that a fix may only be relevant to an existing supported version. In these cases we may need to do a patch release for that older supported version.

A good example, if we were to find a bug in the Envoy v2 protocol configuration we would only want to target the v2 release.

In this scenario, the base branch that we would create our feature branch off from would be the latest `minor` version for that release. As of writing this, that would be the `release/v2.4` branch. We would **not** need to target master.

But, let's say during our fix we notice other things that need to be addressed that would also need to be fixed in `master`. Then you need to submit a **separate Pull Request** that should first land on master and then follow the normal backporting process for the other patches.

#### What if I'm still not sure?

This is what the issue discussions and disucssion in Slack are for so that we can help guide you so feel free to ping us in the `#emissary-dev` channel on Slack to discuss directly with us.

### Merge Strategy

> The audience for this section is the Maintainers but also beneficial for Contributors so that they are familiar with how the project operates.

Having a clean linear commit history for a repository makes it easier to understand what is being changed and reduces the mental load for new comers to the project.

To maintain a clean linear commit history the following rules should be followed:

First, always rebase patch branch on to base branch. This means **NO** merge commits from merging base branch into the patch branch. This can be accomplished using git rebase.

```shell
# first, make sure you pull latest upstream changes
git fetch upstream
git checkout master
git pull -ff upstream/master

# checkout patch branch and rebase interactive
# you may have merge conflicts you need to resolve
git checkout my-patch-branch
git rebase -i master
```

> Note: this does rewrite your commit shas so be aware when sharing branches with co-workers.

Once the Pull Request is reviewed and has **two approvals** then a Maintainer can merge. Maintainers should follow prefer the following merge strategies:

1. rebase and merge
2. squash merge

When `rebase and merge` is used your commits are played on top of the base branch so that it creates a clean linear history. This will maintain all the commits from the Pull Request. In most cases this should be the **preferred** merge strategy.

When a Pull Request has lots of fixup commits, or pr feedback fixes then you should ask the Contributor to squash them as part of the PR process.

If the contributor is unable to squash them then using a `squash merge` in some cases makes sense. **IMPORTANT**, when this does happen it is important that the commit messages are cleaned up and not just blindly accepted the way proposed by Github. Since it is easy to miss that cleanup step, this should be used less frequently compared to `rebase and merge`.

#### What about merge commit strategy?

> The audience for this section is the Maintainers but also beneficial for Contributors so that they are familiar with how the project operates.

When maintaining a linear commit history, each commit tells the story of what was changed in the repository. When using `merge commits` it
adds an additional commit to the history that is not necessary because the commit history and PR history already tell the story.

Now `merge commits` can be useful when you are concerned with not rewriting the commit sha. Based on the current release process which includes using `rel/v` branches that are tagged and merged into `release/v` branches we must use a `merge commit` when merging these branches. This ensures that the commit sha a Git Tag is pointing at still exists once merged into the `release/v` branch.

## Contributing to the Docs

The Emissary-ingress community will all benefit from having documentation that is useful and correct. If you have found an issue with the end user documentation, then please help us out by submitting an issue and/or pull request with a fix!

The end user documentation for Emissary-ingress lives in a different repository and can be found at <https://github.com/datawire/ambassador-docs>.

See this repository for details on how to contribute to either a `pre-release` or already-released version of Emissary-ingress.

## Advanced Topics

This section is for more advanced topics that provide more detailed instructions. Make sure you go through the Development Setup and read the Architecture document before exploring these topics.

### Running Emissary-ingress internals locally

The main entrypoint is written in go. It strives to be as compatible as possible
with the normal go toolchain. You can run it with:

```bash
go run ./cmd/busyambassador entrypoint
```

Of course just because you can run it this way does not mean it will succeed.
The entrypoint needs to launch `diagd` and `envoy` in order to function, and it
also expect to be able to write to the `/ambassador` directory.

#### Setting up diagd

If you want to hack on diagd, its easiest to setup a virtualenv with an editable
copy and launch your `go run` from within that virtualenv. Note that these
instructions depend on the virtualenvwrapper
(<https://virtualenvwrapper.readthedocs.io/en/latest/>) package:

```bash
# Create a virtualenv named venv with all the python requirements
# installed.
python3 -m venv venv
. venv/bin/activate
# If you're doing this in Datawire's apro.git, then:
cd ambassador
# Update pip and install dependencies
pip install --upgrade pip
pip install orjson    # see below
pip install -r builder/requirements.txt
# Created an editable installation of ambassador:
pip install -e python/
# Check that we do indeed have diagd in our path.
which diagd
# If you're doing this in Datawire's apro.git, then:
cd ..
```

(Note: it shouldn't be necessary to install `orjson` by hand. The fact that it is
at the moment is an artifact of the way Ambassador builds currently happen.)

#### Changing the ambassador root

You should now be able to launch ambassador if you set the
`ambassador_root` environment variable to a writable location:

   ambassador_root=/tmp go run ./cmd/busyambassador entrypoint

#### Getting envoy

If you do not have envoy in your path already, the entrypoint will use
docker to run it. At the moment this is untested for macs which probably
means it is broken since localhost communication does not work by
default on macs. This can be made to work as soon an intrepid volunteer
with a mac reaches out to me (<rhs@datawire.io>).

#### Shutting up the pod labels error

An astute observe of the logs will notice that ambassador complains
vociferously that pod labels are not mounted in the ambassador
container. To reduce this noise, you can:

```bash
mkdir /tmp/ambassador-pod-info && touch /tmp/ambassador-pod-info/labels
```

#### Extra credit

When you run ambassador locally it will configure itself exactly as it
would in the cluster. That means with two caveats you can actually
interact with it and it will function normally:

1. You need to run `telepresence connect` or equivalent so it can
   connect to the backend services in its configuration.

2. You need to supply the host header when you talk to it.

### Debugging and Developing Envoy Configuration

Envoy configuration is generated by the ambassador compiler. Debugging
the ambassador compiler by running it in kubernetes is very slow since
we need to push both the code and any relevant kubernetes resources
into the cluster. The following sections will provide tips for improving
this development experience.

#### Ambassador Dump

The `ambassador dump` tool is also useful for debugging and hacking on
the compiler. After running `make shell`, you'll also be able to use
the `ambassador` CLI, which can export the most import data structures
that Ambassador works with as JSON.  It works from an input which can
be either a single file or a directory full of files in the following
formats:

- raw Ambassador resources; or
- an annotated Kubernetes resources like you'll find in `/tmp/k8s-AmbassadorTest.yaml` after running `make test`; or
- a `watt` snapshot like you'll find in the `$AMBASSADOR_CONFIG_BASE_DIR/snapshots/snapshot.yaml` (which is a JSON file, I know, it's misnamed).

Given an input source, running

```bash
ambassador dump --ir --xds [$input_flags] $input > test.json
```

will dump the Ambassador IR and v2 Envoy configuration into `test.json`. Here
`$input_flags` will be

- nothing for raw Ambassador resources;
- `--k8s` for Kubernetes resources; or
- `--watt` for a `watt` snapshot.

You can get more information with

```bash
ambassador dump --help
```

### Making changes to Envoy

Emissary-ingress is built on top of Envoy and leverages a vendored version of Envoy (*we track upstream very closely*). This section will go into how to make changes to the Envoy that is packaged with Emissary-ingress.

This is a bit more complex than anyone likes, but here goes:

#### 1. Preparing your machine

Building and testing Envoy can be very resource intensive.  A laptop
often can build Envoy... if you plug in an external hard drive, point
a fan at it, and leave it running overnight and most of the next day.
At Ambassador Labs, we'll often spin up a temporary build machine in GCE, so
that we can build it very quickly.

As of Envoy 1.15.0, we've measure the resource use to build and test
it as:

> | Command            | Disk Size | Disk Used | Duration[1] |
> |--------------------|-----------|-----------|-------------|
> | `make update-base` | 450G      |  12GB     | ~11m        |
> | `make check-envoy` | 450G      | 424GB     | ~45m        |
>
> [1] On a "Machine type: custom (32 vCPUs, 512 GB memory)" VM on GCE,
> with the following entry in its `/etc/fstab`:
>
> ```bash
> tmpfs:docker  /var/lib/docker  tmpfs  size=450G  0  0
> ```

If you have the RAM, we've seen huge speed gains from doing the builds
and tests on a RAM disk (see the `/etc/fstab` line above).

#### 2. Setting up your workspace to hack on Envoy

1. From your `emissary.git` checkout, get Emissary-ingress's current
   version of the Envoy sources, and create a branch from that:

   ```shell
   make $PWD/_cxx/envoy
   git -C _cxx/envoy checkout -b YOUR_BRANCHNAME
   ```
2. To build Envoy in FIPS mode, set the following variable:

   ```shell
   export FIPS_MODE=true
   ```

   It is important to note that while building Envoy in FIPS mode is
   required for FIPS compliance, additional steps may be necessary.
   Emissary does not claim to be FIPS compliant or certified.
   See [here](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/security/ssl#fips-140-2) for more information on FIPS and Envoy.

> _NOTE:_ FIPS_MODE is NOT supported by the emissary-ingress maintainers but we provide this for developers as convience 

#### 3. Hacking on Envoy

Modify the sources in `./_cxx/envoy/`. or update the branch and/or `ENVOY_COMMIT` as necessary in `./_cxx/envoy.mk`

#### 4. Building and testing your hacked-up Envoy

> See `./_cxx/envoy.mk` for the full list of targets.

Multiple Phony targets are provided so that developers can run the steps they are interested in when developing, here are few of the key ones:

- `make update-base`: will perform all the steps necessary to verify, build envoy, build docker images, push images to the container repository and compile the updated protos.

- `make build-envoy`: will build the envoy binaries using the same build container as the upstream Envoy project. Build outputs are mounted to the  `_cxx/envoy-docker-build` directory and Bazel will write the results there.

- `make build-base-envoy-image`: will use the release outputs from building envoy to generate a new `base-envoy` container which is then used in the main emissary-ingress container build.

- `make push-base-envoy`: will push the built container to the remote container repository.

- `make check-envoy`: will use the build docker container to run the Envoy test suite against the currently checked out envoy in the `_cxx/envoy` folder.

- `make envoy-shell`: will run the envoy build container and open a bash shell session. The `_cxx/envoy` folder is volume mounted into the container and the user is set to the `envoybuild` user in the container to ensure you are not running as root to ensure hermetic builds.

#### 5. Test Devloop

Running the Envoy test suite will compile all the test targets. This is a slow process and can use lots of disk space. 

The Envoy Inner Devloop for build and testing:

- You can make a change to Envoy code and run the whole test by just calling `make check-envoy`
- You can run a specific test instead of the whole test suite by setting the `ENVOY_TEST_LABEL` environment variable. 
  - For example, to run just the unit tests in `test/common/network/listener_impl_test.cc`, you should run:

   ```shell
   ENVOY_TEST_LABEL='//test/common/network:listener_impl_test' make check-envoy
   ```

- Alternatively, you can run `make envoy-shell` to get a bash shell into the Docker container that does the Envoy builds and you are free to interact with `Bazel` directly.

Interpreting the test results:

- If you see the following message, don't worry, it's harmless; the tests still ran:

   ```text
   There were tests whose specified size is too big. Use the --test_verbose_timeout_warnings command line option to see which ones these are.
   ```

   The message means that the test passed, but it passed too
   quickly, and Bazel is suggesting that you declare it as smaller.
   Something along the lines of "This test only took 2s, but you
   declared it as being in the 60s-300s ('moderate') bucket,
   consider declaring it as being in the 0s-60s ('short')
   bucket".

   Don't be confused (as I was) in to thinking that it was saying
   that the test was too big and was skipped and that you need to
   throw more hardware at it.

- **Build or test Emissary-ingress** with the usual `make` commands, with
  the exception that you MUST run `make update-base` first whenever
  Envoy needs to be recompiled; it won't happen automatically.  So
  `make test` to build-and-test Emissary-ingress would become
  `make update-base && make test`, and `make images` to just build
  Emissary-ingress would become `make update-base && make images`.

The Envoy changes with Emissary-ingress:

- Either run `make update-base` to build, and push a new base container and then you can run `make test` for the Emissary-ingress test suite.
- If you do not want to push the container you can instead:
   - Build Envoy - `make build-envoy`
   - Build container - `make build-base-envoy-image` 
   - Test Emissary - `make test`


#### 6. Protobuf changes

If you made any changes to the Protocol Buffer files or if you bumped versions of Envoy then you 
should make sure that you are re-compiling the Protobufs so that they are available and checked-in
to the emissary.git repository.

```sh
make compile-envoy-protos
```

This will copy over the raw proto files, compile and copy the generated go code over to emisary-ignress repository.

#### 7. Finalizing your changes

> NOTE: we are no longer accepting PR's in `datawire/envoy.git`.

If you have custom changes then land them in your custom envoy repository and update the `ENVOY_COMMIT` and `ENVOY_DOCKER_REPO` variable in `_cxx/envoy.mk` so that the image will be pushed to the correct repository.

Then run `make update-base` does all the bits so assuming that was successful then are all good.

**For maintainers:**

You will want to make sure that the image is pushed to the backup container registries:

```shell
# upload image to the mirror in GCR
SHA=GET_THIS_FROM_THE_make_update-base_OUTPUT
TAG="envoy-0.$SHA.opt"
docker pull "docker.io/emissaryingress/base-envoy:envoy-0.$TAG.opt"
docker tag "docker.io/emissaryingress/base-envoy:$TAG" "gcr.io/datawire/ambassador-base:$TAG"
docker push "gcr.io/datawire/ambassador-base:$TAG"
```

#### 8. Final Checklist

**For Maintainers Only**

Here is a checklist of things to do when bumping the `base-envoy` version:

- [ ] The image has been pushed to...
  - [ ] `docker.io/emissaryingress/base-envoy`
  - [ ] `gcr.io/datawire/ambassador-base`
- [ ] The `datawire/envoy.git` commit has been tagged as `datawire-$(git describe --tags --match='v*')`
      (the `--match` is to prevent `datawire-*` tags from stacking on each other).
- [ ] It's been tested with...
  - [ ] `make check-envoy`

The `check-envoy-version` CI job will double check all these things, with the exception of running
the Envoy tests. If the `check-envoy-version` is failing then double check the above, fix them and
re-run the job.

### Developing Emissary-ingress (Maintainers-only advice)

At the moment, these techniques will only work internally to Maintainers. Mostly
this is because they require credentials to access internal resources at the
moment, though in several cases we're working to fix that.

#### Updating license documentation

When new dependencies are added or existing ones are updated, run
`make generate` and commit changes to `DEPENDENCIES.md` and
`DEPENDENCY_LICENSES.md`

#### Upgrading Python dependencies

Delete `python/requirements.txt`, then run `make generate`.

If there are some dependencies you don't want to upgrade, but want to
upgrade everything else, then

 1. Remove from `python/requirements.txt` all of the entries except
    for those you want to pin.
 2. Delete `python/requirements.in` (if it exists).
 3. Run `make generate`.

> **Note**: If you are updating orjson you will need to also update `docker/base-python/Dockerfile` before running `make generate` for the new version. orjson uses rust bindings and the default wheels on PyPI rely on glibc. Because our base python image is Alpine based, it is built from scratch using rustc to build a musl compatable version.

 > :warning: You may run into an error when running `make generate` where it can't detect the licenses for new or upgraded dependencies, which is needed so that so that we can properly generate DEPENDENCIES.md and DEPENDENCY_LICENSES.md. If that is the case, you may also have to update `build-aux/tools/src/py-mkopensource/main.go:parseLicenses` for any license changes then run `make generate` again.

## FAQ

This section contains a set of Frequently Asked Questions that may answer a question you have. Also, feel free to ping us in Slack.

### How do I find out what build targets are available?

Use `make help` and `make targets` to see what build targets are
available along with documentation for what each target does.

### How do I develop on a Mac with Apple Silicon?

To ensure that developers using a Mac with Apple Silicon can contribute, the build system ensures
the build artifacts are `linux/amd64` rather than the host architecture. This behavior can be overriden
using the `BUILD_ARCH` environment variable (e.g. `BUILD_ARCH=linux/arm64 make images`).

### How do I develop on Windows using WSL?

- [WSL 2](https://learn.microsoft.com/en-us/windows/wsl/)
- [Docker Desktop for Windows](https://docs.docker.com/desktop/windows/wsl/)
- [VS Code](https://code.visualstudio.com/)

### How do I test using a private Docker repository?

If you are pushing your development images to a private Docker repo,
then:

```sh
export DEV_USE_IMAGEPULLSECRET=true
export DOCKER_BUILD_USERNAME=...
export DOCKER_BUILD_PASSWORD=...
```

and the test machinery should create an `imagePullSecret` from those Docker credentials such that it can pull the images.

### How do I change the loglevel at runtime?

```console
curl localhost:8877/ambassador/v0/diag/?loglevel=debug
```

Note: This affects diagd and Envoy, but NOT the AES `amb-sidecar`.
See the AES `DEVELOPING.md` for how to do that.

### Can I build from a docker container instead of on my local computer?

If you want to build within a container instead of setting up dependencies on your local machine then you can run the build within a docker container and leverage "Docker in Docker" to build it.

1. `docker pull docker:latest`
2. `docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -it docker:latest sh`
3. `apk add --update --no-cache bash build-base go curl rsync python3 python2 git libarchive-tools gawk jq`
4. `git clone https://github.com/emissary-ingress/emissary.git && cd emissary`
5. `make images`

Steps 0 and 1 are run on your machine, and 2 - 4 are from within the docker container. The base image is a "Docker in Docker" image, ran with `-v /var/run/docker.sock:/var/run/docker.sock` in order to connect to your local daemon from the docker inside the container. More info on Docker in Docker [here](https://hub.docker.com/_/docker).

The images will be created and tagged as defined above, and will be available in docker on your local machine.

### How do I clear everything out to make sure my build runs like it will in CI?

Use `make clobber` to completely remove all derived objects, all cached artifacts, everything, and get back to a clean slate. This is recommended if you change branches within a clone, or if you need to `make generate` when you're not *certain* that your last `make generate` was using the same Envoy version.

Use `make clean` to remove derived objects, but *not* clear the caches.

### My editor is changing `go.mod` or `go.sum`, should I commit that?

If you notice this happening, run `make go-mod-tidy`, and commit that.

(If you're in Ambassador Labs, you should do this from `apro/`, not
`apro/ambassador/`, so that apro.git's files are included too.)

### How do I debug "This should not happen in CI" errors?

These checks indicate that some output file changed in the middle of a
run, when it should only change if a source file has changed.  Since
CI isn't editing the source files, this shouldn't happen in CI!

This is problematic because it means that running the build multiple
times can give different results, and that the tests are probably not
testing the same image that would be released.

These checks will show you a patch showing how the output file
changed; it is up to you to figure out what is happening in the
build/test system that would cause that change in the middle of a run.
For the most part, this is pretty simple... except when the output
file is a Docker image; you just see that one image hash is different
than another image hash.

Fortunately, the failure showing the changed image hash is usually
immediately preceded by a `docker build`.  Earlier in the CI output,
you should find an identical `docker build` command from the first time it
ran.  In the second `docker build`'s output, each step should say
`---> Using cache`; the first few steps will say this, but at some
point later steps will stop saying this; find the first step that is
missing the `---> Using cache` line, and try to figure out what could
have changed between the two runs that would cause it to not use the
cache.

If that step is an `ADD` command that is adding a directory, the
problem is probably that you need to add something to `.dockerignore`.
To help figure out what you need to add, try adding a `RUN find
DIRECTORY -exec ls -ld -- {} +` step after the `ADD` step, so that you
can see what it added, and see what is different on that between the
first and second `docker build` commands.

### How do I run Emissary-ingress tests?

- `export DEV_REGISTRY=<your-dev-docker-registry>` (you need to be logged in and have permission to push)
- `export DEV_KUBECONFIG=<your-dev-kubeconfig>`

If you want to run the Go tests for `cmd/entrypoint`, you'll need `diagd`
in your `PATH`. See the instructions below about `Setting up diagd` to do
that.

| Group           | Command                                                                |
| --------------- | ---------------------------------------------------------------------- |
| All Tests       | `make test`                                                            |
| All Golang      | `make gotest`                                                          |
| All Python      | `make pytest`                                                          |
| Some/One Golang | `make gotest GOTEST_PKGS=./cmd/entrypoint GOTEST_ARGS="-run TestName"` |
| Some/One Python | `make pytest PYTEST_ARGS="-k TestName"`                                |

Please note the python tests use a local cache to speed up test
results. If you make a code update that changes the generated envoy
configuration, those tests will fail and you will need to update the
python test cache.

Note that it is invalid to run one of the `main[Plain.*]` Python tests
without running all of the other `main[Plain*]` tests; the test will
fail to run (not even showing up as a failure or xfail--it will fail
to run at all).  For example, `PYTEST_ARGS="-k WebSocket"` would match
the `main[Plain.WebSocketMapping-GRPC]` test, and that test would fail
to run; one should instead say `PYTEST_ARGS="-k Plain or WebSocket"`
to avoid breaking the sub-tests of "Plain".

### How do I type check my python code?

Ambassador uses Python 3 type hinting and the `mypy` static type checker to
help find bugs before runtime. If you haven't worked with hinting before, a
good place to start is
[the `mypy` cheat sheet](https://mypy.readthedocs.io/en/latest/cheat_sheet_py3.html).

New code must be hinted, and the build process will verify that the type
check passes when you `make test`. Fair warning: this means that
PRs will not pass CI if the type checker fails.

We strongly recommend using an editor that can do realtime type checking
(at Datawire we tend to use PyCharm and VSCode a lot, but many many editors
can do this now) and also running the type checker by hand before submitting
anything:

- `make lint/mypy` will check all the Ambassador code

Ambassador code should produce *no* warnings and *no* errors.

If you're concerned that the mypy cache is somehow wrong, delete the
`.mypy_cache/` directory to clear the cache.

### How do I get the source code for a release?

The current shipping release of Ambassador lives on the `master`
branch. It is tagged with its version (e.g. `v0.78.0`).

Changes on `master` after the last tag have not been released yet, but
will be included in the next release of Ambassador.
