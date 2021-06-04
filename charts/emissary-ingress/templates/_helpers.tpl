{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "ambassador.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
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
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "ambassador.imagetag" -}}
{{- if .Values.image.fullImageOverride }}
    {{- .Values.image.fullImageOverride }}
{{- else }}
    {{- if hasKey .Values.image "tag" -}}
        {{- .Values.image.tag }}
    {{- else if .Values.enableAES }}
        {{- .Values.image.aesTag }}
    {{- else }}
        {{- .Values.image.ossTag }}
    {{- end -}}
{{- end -}}
{{- end -}}

{{/*
Set the image that should be used for ambassador.
Use fullImageOverride if present,
Then if the image repository is explicitly set, use "repository:image"
Otherwise, check if AES is enabled
Use AES image if AES is enabled, ambassador image if not
*/}}
{{- define "ambassador.image" -}}
{{- if .Values.image.fullImageOverride }}
    {{- .Values.image.fullImageOverride }}
{{- else }}
    {{- $repoName := "" }}
    {{- $imageTag := "" }}
    {{- if hasKey .Values.image "repository"  -}}
        {{- $repoName = .Values.image.repository }}
    {{- else if .Values.enableAES }}
        {{- $repoName = .Values.image.aesRepository }}
    {{- else }}
        {{- $repoName = .Values.image.ossRepository }}
    {{- end -}}
    {{- if hasKey .Values.image "tag" -}}
        {{- $imageTag = .Values.image.tag }}
    {{- else if .Values.enableAES }}
        {{- $imageTag = .Values.image.aesTag }}
    {{- else }}
        {{- $imageTag = .Values.image.ossTag }}
    {{- end -}}
    {{- printf "%s:%s" $repoName $imageTag -}}
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
