# {{ .Project.ShortName }}

[{{ .Project.Name }}]({{ .Project.URL }}) - {{ .Project.Description }}

## Introduction

This chart deploys the Emissary-ingress CRDS and (optionally) conversion
webhook on a [Kubernetes](http://kubernetes.io) cluster using the
[Helm](https://helm.sh) package manager.

## Prerequisites
{{ range .Prerequisites }}
- {{ . }}
{{- end }}

## Installing the Chart

A typical installation will use the `emissary-system` namespace:

```console
helm install {{ .Release.Name }} \
     --namespace emissary-system --create-namespace \
     {{ .Repository.Name }}/{{ .Chart.Name }} \
     --version {{ .Chart.Version }} \
     --wait
```

The command deploys the Emissary-ingress CRDs on the Kubernetes cluster in the
default configuration. The [configuration](#configuration) section lists the
parameters that can be configured during installation.

## Changelog

Notable chart changes are listed in the [CHANGELOG](./CHANGELOG.md)

{{ if .Chart.Values -}}
## Configuration

The following table lists the configurable parameters of the `{{ .Chart.Name }}` chart and their default values.

{{ .Chart.Values }}

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example:

```console
helm install {{ .Release.Name }} \
     --namespace emissary-system --create-namespace \
     {{ .Repository.Name }}/{{ .Chart.Name }} \
     --version {{ .Chart.Version }} \
     --set {{ .Chart.ValuesExample }} \
     --wait
```

Alternatively, a YAML file that specifies the values for the parameters can be provided while
installing the chart. For example:


```console
helm install {{ .Release.Name }} \
     --namespace emissary-system --create-namespace \
     {{ .Repository.Name }}/{{ .Chart.Name }} \
     --version {{ .Chart.Version }} \
     --values values.yaml \
     --wait
```
{{- end }}
