# apiext
## apiext Service selector
```
app.kubernetes.io/name: {{ include "ambassador.name" . }}-apiext
app.kubernetes.io/part-of: {{ .Release.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
```

## apiext Pod labels
```
product: aes
app.kubernetes.io/name: {{ include "ambassador.name" . }}-apiext
app.kubernetes.io/part-of: {{ .Release.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}

{{- if and (ne .Values.deploymentTool "getambassador.io") (ne .Values.deploymentTool "kat") }}
  helm.sh/chart: {{ include "ambassador.chart" . }}
  {{- if .Values.deploymentTool }}
    app.kubernetes.io/managed-by: {{ .Values.deploymentTool }}
  {{- else }}
    app.kubernetes.io/managed-by: {{ .Release.Service }}
  {{- end }}
{{- end }}
```
