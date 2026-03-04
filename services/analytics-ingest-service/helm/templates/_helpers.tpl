{{/*
Expand the name of the chart.
*/}}
{{- define "analytics-ingest-service.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "analytics-ingest-service.fullname" -}}
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
{{- define "analytics-ingest-service.labels" -}}
helm.sh/chart: {{ include "analytics-ingest-service.chart" . }}
{{ include "analytics-ingest-service.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "analytics-ingest-service.selectorLabels" -}}
app.kubernetes.io/name: {{ include "analytics-ingest-service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Chart name and version
*/}}
{{- define "analytics-ingest-service.chart" -}}
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
