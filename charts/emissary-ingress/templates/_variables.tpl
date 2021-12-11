{{- /* $chartName : Chart name ***********************************/ }}
  {{- /* (replaces {{ include "ambassador.name" . }}) */ }}
  {{- $chartName := .Values.nameOverride | default .Chart.Name }}
  {{- if len $chartName | gt 63 }}
    {{- fail "$chartName is too long" }}
  {{- end }}

{{- /* $instanceName : Instance of the chart *********************/ }}
  {{- /* (replaces {{ include "ambassador.fullname" . }}) */ }}
  {{- $instanceName := .Values.fullnameOverride | default .Release.Name }}
  {{- if not ($instanceName | contains $chartName) }}
    {{- $instanceName = printf "%s-%s" $chartName $instanceNAme }}
  {{- end }}
  {{- if len $instanceName | gt 63 }}
    {{- fail "$instanceName is too long" }}
  {{- end }}

{{- /* $image : Full image name **********************************/ }}
  {{- /* (replaces {{ include "ambassador.image" . }}) */ }}
  {{- $image := .Values.image.fullImageOverride | default (printf "%s:%s" .Values.image.repository .Values.image.tag) }}

{{- /* $canaryImage : Full image name for canaries****************/ }}
  {{- /* (replaces {{ include "ambassador.canaryImage" . }}) */ }}
  {{- $_canaryImageRepo := .Values.canary.image.repository | default .Values.image.repository }}
  {{- $_canaryImageTag := .Values.canary.image.tag | default .Values.image.tag }}
  {{- $canaryImage := .Values.canary.image.fullImageOverride | default (printf "%s:%s" $_canaryImageRepo $_canaryImageTag ) }}

{{- /* $namespace : The namespace to install in ****************/ }}
  {{- /* (replaces {{ include "ambassador.namespace" . }}) */ }}
  {{- $namespace := .Values.namespaceOverride | default .Release.Namespace }}

{{- /* $serviceAccountName *************************************/ }}
  {{- /* (replaces {{ include "ambassador.serviceAccountName" . }}) */ }}
  {{- $serviceAccountName := .Values.serviceAccount.name | default (ternary .Values.serviceAccount.create $instanceName "default") }}

{{- /* $rbacName ***********************************************/ }}
  {{- /* (replaces {{ include "ambassador.rbacName" . }}) */ }}
  {{- $rbacName := .Values.rbac.nameOverride | default $instanceName }}

{{- /* end */ -}}
