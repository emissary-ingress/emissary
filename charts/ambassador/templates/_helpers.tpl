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
{{- else if hasKey .Values.image "repository"  -}}
{{- if .Values.enableAES }}
{{- printf "%s:%s" .Values.image.repository .Values.image.aesTag -}}
{{- else }}
{{- printf "%s:%s" .Values.image.repository .Values.image.tag -}}
{{- end -}}
{{- else if .Values.enableAES -}}
{{- printf "%s:%s" "docker.io/datawire/aes" .Values.image.aesTag -}}
{{- else -}}
{{- if .Values.image.ossRepository }}
{{- printf "%s:%s" .Values.image.ossRepository .Values.image.tag -}}
{{- else }}
{{- printf "%s:%s" "docker.io/datawire/ambassador" .Values.image.tag -}}
{{- end -}}
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
