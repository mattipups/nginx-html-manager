{{- define "html.image" -}}
{{- if .digest -}}{{ .repository }}@{{ .digest }}{{- else -}}{{ .repository }}:{{ .tag }}{{- end -}}
{{- end -}}
{{- define "html.claim" -}}
{{- default (printf "%s-data" .Release.Name) .Values.persistence.existingClaim -}}
{{- end -}}
