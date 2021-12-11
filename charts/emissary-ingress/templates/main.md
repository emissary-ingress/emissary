# main-ish
## main Service selector
```
{{- if .Values.service.selector }}
  {{ toYaml .Values.service.selector | nindent 4 }}
{{- else }}
  app.kubernetes.io/name: {{ include "ambassador.name" . }}
  app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- if not (and .Values.canary.enabled (not .Values.canary.mixPods)) }}
  profile: main
{{- end }}
```
## main-canary Service selector
```
{{- if .Values.service.selector }}
  {{ toYaml .Values.service.selector | nindent 4 }}
{{- else }}
  app.kubernetes.io/name: {{ include "ambassador.name" . }}
  app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

profile: canary
```
## admin Service selector
```
{{- if .Values.service.selector }}
  {{ toYaml .Values.service.selector | nindent 6 }}
{{- else }}
  app.kubernetes.io/name: {{ include "ambassador.name" . }}
  app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
```
## main Pod labels
```
{{- if .Values.service.selector }}
  {{ toYaml .Values.service.selector | nindent 8 }}
{{- end }}

{{- if ne .Values.deploymentTool "getambassador.io" }}
  app.kubernetes.io/name: {{ include "ambassador.name" . }}
  app.kubernetes.io/part-of: {{ .Release.Name }}
  app.kubernetes.io/instance: {{ .Release.Name }}
  product: aes
{{- end }}

{{- if .Values.deploymentTool }}
  app.kubernetes.io/managed-by: {{ .Values.deploymentTool }}
{{- else }}
  app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

profile: main

{{- if .Values.podLabels }}
  {{- toYaml .Values.podLabels | nindent 8 }}
{{- end }}
```
## main-canary Pod labels
```
{{- if .Values.service.selector }}
  {{ toYaml .Values.service.selector | nindent 8 }}
{{- end }}

{{- if ne .Values.deploymentTool "getambassador.io" }}
  app.kubernetes.io/name: {{ include "ambassador.name" . }}
  app.kubernetes.io/part-of: {{ .Release.Name }}
  app.kubernetes.io/instance: {{ .Release.Name }}
  product: aes
{{- end }}

{{- if .Values.deploymentTool }}
  app.kubernetes.io/managed-by: {{ .Values.deploymentTool }}
{{- else }}
  app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

profile: canary

{{- if .Values.podLabels }}
  {{- toYaml .Values.podLabels | nindent 8 }}
{{- end }}
```
