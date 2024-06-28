{{- define "k8s-deployment-scaler.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "k8s-deployment-scaler.fullname" -}}
{{- printf "%s" (include "k8s-deployment-scaler.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "k8s-deployment-scaler.chart" -}}
{{- .Chart.Name -}}
{{- end -}}

{{- define "k8s-deployment-scaler.labels" -}}
helm.sh/chart: {{ include "k8s-deployment-scaler.chart" . }}
{{ include "k8s-deployment-scaler.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "k8s-deployment-scaler.selectorLabels" -}}
app.kubernetes.io/name: {{ include "k8s-deployment-scaler.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}