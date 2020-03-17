---
   description: In this guide, we'll walk through using Ambassador Edge Stack to publish services.
---
# Publishing Services

In this guide, we'll walk you through using Ambassador Edge Stack to publish services. Ambassador Edge Stack provides two mechanisms you can use to publish services running in kubernetes:

1. The Mapping resource.
2. The Project resource.

The Mapping resource provides the maximum flexibility over routing, but requires you to build out a solution from scratch for how you are going to build, test, and use routing to safely update your services. The Project resource provides an out-of-the box solution that will build, test, and safely update your services directly from source code in a git repo.

We recommend you start with the Project resource as it is far quicker to get up and running. The Project resource is built on top of the Mapping resource, so you can easily add customized alternatives and/or extensions to the Project resource later if the need arises.

## Before You Begin

You will need:

* A working installation of Ambassador Edge Stack...
  * With TLS configured.
  * And access to the AES admin console (https://**$YOUR_HOST**/edge_stack/admin/).
* A Github account.
* git

## Quick Start

Publishing any github repo is simple, all you need is a Dockerfile in the root directory of the repository that exposes a service on port 8080. But to keep things simple, lets start out with a new repo based on the datawire/project-template repo.

1. Create a new repo from our quickstart template by going to https://github.com/datawire/project-template/generate (please note the **$OWNER** and **$REPO_NAME** you choose for later reference).
2. Go to https://**$YOUR_HOST**/edge_stack/admin/
3. Click on Projects -> Add
   - Fill in the name and namespace you would like to use for the Project CRD.
   - Enter **$YOUR_HOST** for the host. It is important that github can reach this host.
   - Choose a prefix where you would like to publish your project.
   - Enter **$OWNER/$REPO_NAME** for the github repo.
   - Enter a github token with **repo** scope. If you don't already have one, you can generate one by going to https://github.com/settings/tokens/new . Make sure you select the **repo** checkbox under "Select Scopes"

4. Click Save. The project master branch of the project will build and deploy at the configured prefix.

Building the first time will take a little while. Subsequent builds will be much faster. You can click click on the build link to see a live stream of the build logs. Once your build is published, continue to see how to update your service.

The Projects tab will provide a complete summary of all activity related to a project. It will be handy to keep it open and visible for the rest of this guide.

## Updating your service directly

Any code changes on your master branch will be automatically built and deployed. Let's make a change to see how this works:

1. Do a `git clone` of your new repo.
   - (Or click edit on the github UI.)
2. Edit server.js and change "Hello World!" to "Hello Master Branch!".
3. Commit and push your change directly to the master branch.
   - (Or click "Commit changes" in the github UI with the "Commit directly to the `master` branch" option chosen.)

Visit https://**$YOUR_HOST**/edge_stack/admin/ -> Projects and you will see a new build proceeding. When that build completes you will see your change published at your chosen prefix.

## Updating your service with a PR

Of course we'd like to be able to test our code before putting it into production, so let's create a PR instead of pushing directly to master:

1. Do a `git clone` of your new repo.
   - (Or click edit on the github UI.)
2. Edit server.js and change "Hello Master Branch!" to "Hello Pull Requests!".
3. Create a branch (`git checkout -b your-branch`) and PR.
   - (Or click "Propose file change" in the github UI with the "Create a new branch for this commit..." option chosen)
4. Visit the Projects tab or click on the status link in your github PR. You will see a build for master and a build for the PR, each with their own url. Click on the URL for the PR and you will see your changes at its preview url.

Every PR gets published at its own preview url, so you can have as many as you like and test them however you like before merging.

## Automating Tests

1. TODO

## What’s Next?

The Ambassador Edge Stack has a comprehensive range of [features](/features/) to support the requirements of any edge microservice.

To learn more about how the Ambassador Edge Stack works, along with use cases,
best practices, and more, check out the [Welcome page](/docs/) or read the
[Ambassador Story](/about/why-ambassador).

For a custom configuration, you can install the Ambassador Edge Stack [manually](/user-guide/manual-install).
