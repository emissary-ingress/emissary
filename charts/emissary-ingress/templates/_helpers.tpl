{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "ambassador.name" -}}
{{- if contains "emissary" .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else if contains "ambassador" .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- default "emissary-ingress" .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "ambassador.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default "emissary-ingress" .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else if contains "ambassador" .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else if contains "emissary" .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
The base set of labels for all resources.
*/}}
{{- define "ambassador.labels" -}}
{{- $deploymentTool := .Values.deploymentTool | default .Release.Service }}
{{- if eq $deploymentTool "Helm" -}}
helm.sh/chart: {{ include "ambassador.chart" . }}
{{- end }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/part-of: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ $deploymentTool }}
{{- end -}}

{{/*
Set the image that should be used for ambassador.
Use fullImageOverride if present,
Then if the image repository is explicitly set, use "repository:image"
*/}}
{{- define "ambassador.image" -}}
{{- if .Values.image.fullImageOverride }}
{{- .Values.image.fullImageOverride }}
{{- else if hasKey .Values.image "repository"  -}}
{{- printf "%s:%s" .Values.image.repository .Values.image.tag -}}
{{- else -}}
{{- printf "%s:%s" "docker.io/emissaryingress/emissary" .Values.image.tag -}}
{{- end -}}
{{- end -}}

{{/*
Set the image that should be used for the canary deployment.
disabled if fullImageOverride is present
*/}}
{{- define "ambassador.canaryImage" -}}
{{- if .Values.image.fullImageOverride }}
{{- printf "%s" "" -}}
{{- else if and .Values.canary.image.repository .Values.canary.image.tag -}}
{{- printf "%s:%s" .Values.canary.image.repository .Values.canary.image.tag -}}
{{- else if .Values.canary.image.tag -}}
{{- if hasKey .Values.image "repository" -}}
{{- printf "%s:%s" .Values.image.repository .Values.canary.image.tag -}}
{{- else -}}
{{- printf "%s:%s" "docker.io/emissaryingress/emissary" .Values.canary.image.tag -}}
{{- end -}}
{{- else -}}
{{- printf "%s" "" -}}
{{- end -}}
{{- end -}}


{{/*
Create chart namespace based on override value.
*/}}
{{- define "ambassador.namespace" -}}
{{- if .Values.namespaceOverride -}}
{{- .Values.namespaceOverride -}}
{{- else -}}
{{- .Release.Namespace -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "ambassador.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "ambassador.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (include "ambassador.fullname" .) .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
Create the name of the RBAC to use
*/}}
{{- define "ambassador.rbacName" -}}
{{ default (include "ambassador.fullname" .) .Values.rbac.nameOverride }}
{{- end -}}

{{/*
Define the http port of the Ambassador service
*/}}
{{- define "ambassador.servicePort" -}}
{{- range .Values.service.ports -}}
{{- if (eq .name "http") -}}
{{ default .port }}
{{- end -}}
{{- end -}}
{{- end -}}
