# {{ .Project.ShortName }}

[{{ .Project.Name }}]({{ .Project.URL }}) - {{ .Project.Description }}

## TL;DR;

```console
$ helm repo add {{ .Repository.Name }} {{ .Repository.URL }}
$ helm repo update
$ helm install {{ .Release.Name }} --devel {{ .Repository.Name }}/{{ .Chart.Name }} -n {{ .Release.Namespace }}{{ with .Chart.Version }} --version={{.}}{{ end }}
```

## Introduction

This chart deploys {{ .Project.App }} on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

This chart is used to install the 2.0 release line of {{ .Project.App }}. 

Versions in the older 1.0 release line of Emissary Ingress and Ambassador Edge Stack share a
single chart that can be found in this repository under the branch for the specific release, e.g.
[the release/v1.14 branch] for the latest 1.14 chart, [release/v1.13] for 1.13, and so on.

> Note that for 1.0 releases, the `enableAES` helm value is used to control installing Edge Stack or
  Emissary Ingress.

As of version 2.0, Emissary Ingress and Ambassador Edge Stack have separate charts. The helm chart
for Ambassador Edge Stack 2.0 lives in the [Ambassador Edge Stack chart repository].

See the [Ambassador Edge Stack FAQ] for more information about the differences between Emissary
Ingress and Ambassador Edge Stack.

[the release/v1.14 branch]: https://github.com/emissary-ingress/emissary/tree/release/v1.14/charts/ambassador 
[release/v1.13]: https://github.com/emissary-ingress/emissary/tree/release/v1.13/charts/ambassador 
[Ambassador Edge Stack chart repository]: https://github.com/datawire/edge-stack/tree/main/charts/edge-stack
[Ambassador Edge Stack FAQ]: https://www.getambassador.io/docs/edge-stack/latest/about/faq/#whats-the-difference-between-ossproductname-and-aesproductname

## Prerequisites
{{ range .Prerequisites }}
- {{ . }}
{{- end }}

## Installing the Chart

To install the chart with the release name `{{ .Release.Name }}`:

```console
$ helm install {{ .Release.Name }} --devel {{ .Repository.Name }}/{{ .Chart.Name }} -n {{ .Release.Namespace }}{{ with .Chart.Version }} --version={{.}}{{ end }}
```

The command deploys {{ .Project.App }} on the Kubernetes cluster in the default configuration. The [configuration](#configuration) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `{{ .Release.Name }}`:

```console
$ helm delete {{ .Release.Name }} -n {{ .Release.Namespace }}
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Changelog

Notable chart changes are listed in the [CHANGELOG](./CHANGELOG.md)

{{ if .Chart.Values -}}
## Configuration

The following table lists the configurable parameters of the `{{ .Chart.Name }}` chart and their default values.

{{ .Chart.Values }}

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example:

```console
$ helm install {{ .Release.Name }} --devel {{ .Repository.Name }}/{{ .Chart.Name }} -n {{ .Release.Namespace }}{{ with .Chart.Version }} --version={{.}}{{ end }} --set {{ .Chart.ValuesExample }}
```

Alternatively, a YAML file that specifies the values for the parameters can be provided while
installing the chart. For example:

```console
$ helm install {{ .Release.Name }} --devel {{ .Repository.Name }}/{{ .Chart.Name }} -n {{ .Release.Namespace }}{{ with .Chart.Version }} --version={{.}}{{ end }} --values values.yaml
```
{{- end }}
