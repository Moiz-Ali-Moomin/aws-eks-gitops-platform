{{/*
Expand the name of the chart.
*/}}
{{- define "admin-backoffice-service.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "admin-backoffice-service.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "admin-backoffice-service.labels" -}}
helm.sh/chart: {{ include "admin-backoffice-service.chart" . }}
{{ include "admin-backoffice-service.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "admin-backoffice-service.selectorLabels" -}}
app.kubernetes.io/name: {{ include "admin-backoffice-service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Chart name and version
*/}}
{{- define "admin-backoffice-service.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Construct full image reference.
*/}}
{{- define "image" -}}
{{ .Values.global.awsAccountId }}.dkr.ecr.{{ .Values.global.awsRegion | default "us-east-1" }}.amazonaws.com/ecommerce-repo/{{ .Chart.Name }}:{{ .Values.image.tag }}
{{- end -}}

{{/*
Construct IRSA role ARN from global.awsAccountId.
*/}}
{{- define "serviceAccountRoleArn" -}}
arn:aws:iam::{{ .Values.global.awsAccountId }}:role/{{ .Chart.Name }}-irsa
{{- end -}}
