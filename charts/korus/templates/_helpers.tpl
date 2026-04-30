{{- define "korus.name" -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "korus.fullname" -}}
{{- if contains .Chart.Name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "korus.controllerName" -}}
{{- printf "%s-controller-manager" (include "korus.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "korus.gatewayName" -}}
{{- printf "%s-gateway" (include "korus.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "korus.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "korus.controllerName" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "korus.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | quote }}
app.kubernetes.io/name: {{ include "korus.name" . | quote }}
app.kubernetes.io/instance: {{ .Release.Name | quote }}
app.kubernetes.io/part-of: "korus"
app.kubernetes.io/managed-by: {{ .Release.Service | quote }}
{{- end -}}

{{- define "korus.controllerLabels" -}}
{{ include "korus.labels" . }}
app.kubernetes.io/component: "controller-manager"
{{- end -}}
