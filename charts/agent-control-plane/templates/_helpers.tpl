{{- define "agent-control-plane.name" -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "agent-control-plane.fullname" -}}
{{- if contains .Chart.Name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "agent-control-plane.controllerName" -}}
{{- printf "%s-controller-manager" (include "agent-control-plane.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "agent-control-plane.gatewayName" -}}
{{- printf "%s-gateway" (include "agent-control-plane.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "agent-control-plane.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "agent-control-plane.controllerName" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "agent-control-plane.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | quote }}
app.kubernetes.io/name: {{ include "agent-control-plane.name" . | quote }}
app.kubernetes.io/instance: {{ .Release.Name | quote }}
app.kubernetes.io/part-of: "agent-control-plane"
app.kubernetes.io/managed-by: {{ .Release.Service | quote }}
{{- end -}}

{{- define "agent-control-plane.controllerLabels" -}}
{{ include "agent-control-plane.labels" . }}
app.kubernetes.io/component: "controller-manager"
{{- end -}}
