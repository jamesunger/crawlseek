{{/* Generate the full name of the chart */}}
{{- define "seeker.fullname" -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Generate the name of the chart */}}
{{- define "seeker.name" -}}
{{ .Chart.Name | quote }}
{{- end -}}

{{/* Generate the labels for the resources */}}
{{- define "seeker.labels" -}}
app: {{ include "seeker.name" . }}
release: {{ .Release.Name }}
{{- end -}}
