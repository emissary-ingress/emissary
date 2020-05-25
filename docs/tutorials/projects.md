# Getting started with the Project CRD

This feature is in beta, your feedback is appreciated.

In this guide, we'll walk you through using Ambassador Edge Stack's [Project CRD](../../topics/using/projects/). At the end of this you will have launched your own microservice in less time than it takes to microwave popcorn. Not only that, but this service will meet the most important standards of **production-readiness**. It will be:

* **Secure:** Protected with TLS, Authentication, and Rate Limiting
* **Robust:** Complete with the usual -ilities: scalability, reliability, availability
* **Agile:** Can be updated *quickly* and *frequently* without disrupting users!

## Before You Begin

You will need:

* A [Github](https://github.com) account.
* A [working installation of Ambassador Edge Stack...](../getting-started/)
  * With TLS configured. (Needed for the github webhook used to sync source changes to your cluster.)
  * And access to the AES admin console: https://$YOUR_HOST/edge_stack/admin/.

## Quick Start

1. First, let's enable the `ProjectController` for your AES installation. This will give you access to the `Projects` tab in the `Edge Policy Console`. If you already see the `Projects` tab then you can skip this step:

```
kubectl apply -f https://getambassador-preview.netlify.app/yaml/projects.yaml &&  \
kubectl wait --for condition=established --timeout=90s crd -lproduct=aes && \
kubectl apply -f - <<EOF && kubectl patch deployment ambassador -n ambassador -p "{\"spec\":{\"template\":{\"metadata\":{\"annotations\":{\"date\":\"`date +'%s'`\"}}}}}"
apiVersion: getambassador.io/v2
kind: ProjectController
metadata:
  labels:
    projects.getambassador.io/ambassador_id: default
  name: projectcontroller
  namespace: ambassador
EOF
```

2. Run `edgectl login $YOUR_HOST` and click on the `Projects` tab.

3. Create an HTTP service implementation in your own new Github repo with our [quick start project generator](https://github.com/datawire/project-template/generate).

4. Click on Projects -> Add, you will be directed to enter the name, namespace, host, and url path prefix for your project: ![Add Project](../../images/project-create.png)

5. You will also need to supply a github token: ![Github Token](../../images/project-create-github-token.png)
   Make sure you select the repo scope for your token: ![Repo Scope](../../images/project-create-repo-scope.png)


6. As soon as you enter a valid access token, you will see the "github repo" field populate with all the github repos granted access by that token. Choose your newly created repo from step 3 and click Save: ![Github Repo](../../images/project-create-github-repo.png)

You will see the Project resource automatically build and deploy the code in your newly created repo. Building the first time will take a few minutes. Subsequent builds will be much faster due to caching of the docker image layers in the build. To follow the progress of building, you can click on the "build" link to see your build logs streamed in realtime:

![Project Build Logs](../../images/project-build-logs.png)

When your build and deploy succeeds, the project will show the master branch as "Deployed" and the "build", "logs", and "url" links will all be green:

![Project Deployed](../../images/project-deployed.png)

Click the "url" link to visit your newly deployed microservice:

![Project URL](../../images/project-url.png)

## Viewing Server Logs

You can use the Projects Tab to access the server logs for your project. Click on the "log" link next to the master deployment of your project, and you will see realtime log output from your server:

![Project Server Logs](../../images/project-server-logs.png)

## Making Updates

We are going to update our project by creating a pull-request on Github. The Project resource will automatically build and stage the PR'ed version of our service so that we can make sure it works the way we anticipated before updating our production deployment.

1. Go to your git repo in your browser and click on server.js:

![Server.js](../../images/project-server.js.png)

2. Click on the edit icon:

![Edit Server.js](../../images/project-server.js-edit.png)

3. Change "Hello World!" to "Hello Update!":

![Hello Update](../../images/project-update.png)

4. Select "Create a new branch for this commit..." and then click "Propose file change":

![New Branch](../../images/project-update-pr.png)

5. Create the Pull Request:

![New Branch](../../images/project-update-pr-create.png)


6. Now go to the Projects Tab, you will see the Pull Request you just made being built and staged. When the status reaches Deployed, you can click on the preview url link:

![PR Build](../../images/project-update-url.png)

7. You will see the updated version of your code running at a staging deployment. Note the preview URL. Every PR gets published at its own preview URL so you can have as many simultaneous PRs as you like and test them however you like before merging:

![PR Preview](../../images/project-update-preview.png)

8. Once we are satisfied with our change we can go back to github and merge our PR:

![PR Merge](../../images/project-update-merge.png)

9. After master finishes building and reaches the Deployed state, we can click on the URL:

![Merged URL](../../images/project-update-merged-url.png)

10. And we can see our updated service running in production!

![Merged](../../images/project-update-merged.png)

## What’s Next?

Read more about [using Projects](../../topics/using/projects/), including how to use Ambassador Edge Stack's powerful [Authentication](../../topics/using/filters/) and [Rate Limiting](../../topics/using/rate-limits/) features to secure your service.

The Ambassador Edge Stack has a comprehensive range of [features](/features/) to support the requirements of any edge microservice.

To learn more about how the Ambassador Edge Stack works, along with use cases,
best practices, and more, check out the [Welcome page](/docs/) or read the
[Ambassador Story](/about/why-ambassador).

For a custom configuration, you can install the Ambassador Edge Stack [manually](/user-guide/manual-install).
