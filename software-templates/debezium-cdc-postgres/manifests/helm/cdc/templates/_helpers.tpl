{{- define "cdc.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "cdc.labels" -}}
app.kubernetes.io/name: {{ include "cdc.name" . }}
app.kubernetes.io/part-of: rhads-demo
app.kubernetes.io/component: cdc
backstage.io/kubernetes-id: {{ include "cdc.name" . }}
{{- end }}

{{- define "cdc.postgresHost" -}}
{{- printf "%s-postgres.%s.svc" (include "cdc.name" .) (.Values.namespace | default .Release.Namespace) -}}
{{- end }}
