{{/* vim: set filetype=mustache: */}}

{{/*
Expand the name of the chart.
*/}}
{{- define "newrelic.secret.name" -}}
{{- default "steadybit-extension-newrelic" .Values.newrelic.existingSecret -}}
{{- end -}}
