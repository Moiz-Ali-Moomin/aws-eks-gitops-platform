{{/*
Expand the name of the chart.
*/}}
{{- define "attribution-service.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "attribution-service.fullname" -}}
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
{{- define "attribution-service.labels" -}}
helm.sh/chart: {{ include "attribution-service.chart" . }}
{{ include "attribution-service.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "attribution-service.selectorLabels" -}}
app.kubernetes.io/name: {{ include "attribution-service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Chart name and version
*/}}
{{- define "attribution-service.chart" -}}
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
