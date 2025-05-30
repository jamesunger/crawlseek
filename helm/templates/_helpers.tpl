{{/* Generate the full name of the chart */}}
{{- define "frontend.fullname" -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Generate the name of the chart */}}
{{- define "frontend.name" -}}
{{ .Chart.Name | quote }}
{{- end -}}

{{/* Generate the labels for the resources */}}
{{- define "frontend.labels" -}}
app: {{ include "frontend.name" . }}
release: {{ .Release.Name }}
{{- end -}}
