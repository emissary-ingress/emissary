{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "ambassador.name" }}
    {{- if contains "ambassador" .Release.Name }}
        {{- .Release.Name | trunc 63 | trimSuffix "-" }}
    {{- else }}
        {{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
    {{- end }}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "ambassador.fullname" }}
    {{- if .Values.fullnameOverride }}
        {{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
    {{- else }}
        {{- $name := default .Chart.Name .Values.nameOverride }}
        {{- if or (contains $name .Release.Name) (contains "ambassador" .Release.Name) }}
            {{- .Release.Name | trunc 63 | trimSuffix "-" }}
        {{- else }}
            {{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
        {{- end }}
    {{- end }}
{{- end -}}

{{/*
The base set of labels for all resources.
*/}}
{{- define "ambassador.labels" }}
    {{- $deploymentTool := .Values.deploymentTool | default .Release.Service }}
    {{- if eq $deploymentTool "Helm" }}
        {{- "\n" }}helm.sh/chart: {{ include "ambassador.chart" . }}
    {{- end }}
    {{- "\n" }}app.kubernetes.io/instance: {{ .Release.Name }}
    {{- "\n" }}app.kubernetes.io/part-of: {{ .Release.Name }}
    {{- "\n" }}app.kubernetes.io/managed-by: {{ $deploymentTool }}
{{- end -}}

{{/*
Set the image that should be used for ambassador.
Use fullImageOverride if present,
Then if the image repository is explicitly set, use "repository:image"
*/}}
{{- define "ambassador.image" }}
    {{- $repo := .Values.image.repository | default "docker.io/emissaryingress/emissary" }}
    {{- .Values.image.fullImageOverride | default (printf "%s:%s" $repo .Values.image.tag) }}
{{- end -}}

{{/*
Set the image that should be used for the canary deployment.
disabled if fullImageOverride is present
*/}}
{{- define "ambassador.canaryImage" }}
    {{- if not .Values.image.fullImageOverride }}
        {{- $repo := .Values.canary.image.repository | default .Values.image.repository | default "docker.io/emissaryingress/emissary" }}
        {{- if and $repo .Values.canary.image.tag }}
            {{- printf "%s:%s" $repo .Values.canary.image.tag }}
        {{- end }}
    {{- end }}
{{- end -}}

{{/*
Create chart namespace based on override value.
*/}}
{{- define "ambassador.namespace" }}
    {{- .Values.namespaceOverride | default .Release.Namespace }}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "ambassador.chart" }}
    {{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "ambassador.serviceAccountName" }}
    {{ $default := "default" }}
    {{- if .Values.serviceAccount.create }}
        {{ $default = include "ambassador.fullname" . }}
    {{- end }}
    {{- .Values.serviceAccount.name | default $default }}
{{- end -}}

{{/*
Create the name of the RBAC to use
*/}}
{{- define "ambassador.rbacName" }}
    {{- .Values.rbac.nameOverride | default (include "ambassador.fullname" .) }}
{{- end -}}

{{/*
Define the http port of the Ambassador service
*/}}
{{- define "ambassador.servicePort" }}
    {{- range .Values.service.ports }}
        {{- if eq .name "http" }}
            {{- .port }}
        {{- end }}
    {{- end }}
{{- end -}}
