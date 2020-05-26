# Customizing Project Deployment

## This feature is in BETA. Your feedback is greatly appreciated!

The project controller deploys each project revision using a default set of resources that include:

 - a Deployment
 - a Service
 - a Mapping

If you want to customize how these resources are defined or even deploy your application using a different set of resources, then you can define a `project-revision.yaml.tmpl` file in your repo.

When writing this file there are a number of things that are helpful to keep in mind:

- The file is a [golang template](https://golang.org/pkg/text/template/) that is expected to produce Kubernetes yaml.

- The template is supplied variables about the project and revision in its environment. See the reference at the end of this document for a complete list of variables.

- The Kubernetes resources defined in the file will be applied for every revision, and since multiple revisions may exist simultaneously, it is important to template the resource names with a value that is unique to the revision. A simple way to do this is use the revision name in your templates.

- The name of the image built from the Dockerfile is also supplied to the template along with a pull secret for accessing the registry that holds the image. You MUST remember to include the pull secret in your manifests or they will not work.

- The project controller tracks each resource produced by your template and will clean them all up when a revision is removed (i.e. when you close your PR), so don't worry if you make mistakes.

- Any namespaces that are omitted will be defaulted to the revision namespace.

The default resources are defined as follows:

```
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: '{{.Revision.Name}}'
spec:
  ambassador_id:
  - '{{.AmbassadorID}}'
  prefix: '{{.Revision.Prefix}}'
  service: '{{.Revision.Name}}'

---
apiVersion: v1
kind: Service
metadata:
  name: '{{.Revision.Name}}'
spec:
  ports:
  - port: 80
    protocol: TCP
    targetPort: 8080
  selector:
    projects.getambassador.io/ambassador_id: '{{.AmbassadorID}}'
    projects.getambassador.io/revision-uid: '{{.Revision.UID}}'

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: '{{.Revision.Name}}'
spec:
  selector:
    matchLabels:
      projects.getambassador.io/ambassador_id: '{{.AmbassadorID}}'
      projects.getambassador.io/revision-uid: '{{.Revision.UID}}'
  strategy: {}
  template:
    metadata:
      labels:
        projects.getambassador.io/ambassador_id: '{{.AmbassadorID}}'
        projects.getambassador.io/revision-uid: '{{.Revision.UID}}'
        projects.getambassador.io/service: "true"
    spec:
      containers:
      - name: app
        image: '{{.Revision.Image}}'
        env:
        - name: AMB_PROJECT_PREVIEW
          value: '{{.Revision.IsPreview}}'
        - name: AMB_PROJECT_REPO
          value: '{{.Project.Repo}}'
        - name: AMB_PROJECT_REF
          value: '{{.Revision.Ref}}'
        - name: AMB_PROJECT_REV
          value: '{{.Revision.Rev}}'
      imagePullSecrets:
      - name: '{{.Revision.Name}}'
```

## Template Variable Reference

| Template Variable         | Description               |
| :------------------------ | :------------------------ |
| `.AmbassadorID`           | The ambassador ID of the project controller. |
| `.Project`                | All project level info is grouped under this struct. |
| `.Project.Name`           | The project name. |
| `.Project.Namespace`      | The project namespace. |
| `.Project.UID`            | The UID of the project resource. |
| `.Project.Prefix`         | The project prefix. |
| `.Project.Repo`           | The project repo. |
| `.Revision`               | All revision level info is grouped under this struct. |
| `.Revision.Name`          | The name of the revision. |
| `.Revision.Namespace`     | The namespace of the revision. |
| `.Revision.UID`           | The UID of the revision resource. |
| `.Revision.IsPreview`     | A boolean indicating if this is a preview or production revision. |
| `.Revision.Ref`           | The git ref of the revision. |
| `.Revision.Rev`           | The git hash of the revision. |
| `.Revision.Prefix`        | The Prefix of the revision. This is the same as the project prefix for production revisions and a preview prefix for preview revisions |
| `.Revision.Image`         | The image built for this revision. |
| `.Revision.PullSecret`    | The name of the pull secret for the image revision. |
