{{- define "axiops.labels" -}}
app.kubernetes.io/name: {{ .Release.Name | quote }}
app.kubernetes.io/instance: {{ .Release.Name | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service | quote }}
{{- end -}}
{{- define "axiops.selector" -}}
app.kubernetes.io/instance: {{ .Release.Name | quote }}
{{- end -}}
{{- define "axiops.account" -}}
{{- if .Values.serviceAccount.create -}}
{{- default .Release.Name .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}
{{- define "axiops.pvcSpec" -}}
accessModes: {{ toJson .Values.persistence.accessModes }}
{{- if ne .Values.persistence.storageClass nil }}
storageClassName: {{ .Values.persistence.storageClass | quote }}
{{- end }}
resources:
  requests:
    storage: {{ .Values.persistence.size | quote }}
{{- end -}}
{{- define "axiops.pod" -}}
{{- $hasConfigMount := false -}}
{{- range .Values.configMaps -}}{{- if .mountPath -}}{{- $hasConfigMount = true -}}{{- end -}}{{- end -}}
metadata:
  labels:
    {{- with .Values.podLabels }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
    {{- include "axiops.labels" . | nindent 4 }}
  {{- if or .Values.podAnnotations .Values.monitoring.enabled }}
  annotations:
    {{- with .Values.podAnnotations }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
    {{- if .Values.monitoring.enabled }}
    prometheus.io/scrape: "true"
    prometheus.io/path: {{ .Values.monitoring.path | quote }}
    prometheus.io/port: {{ .Values.monitoring.port | quote }}
    {{- end }}
  {{- end }}
spec:
  serviceAccountName: {{ include "axiops.account" . }}
  automountServiceAccountToken: {{ .Values.serviceAccount.automountServiceAccountToken }}
  {{- if has .Values.workload.kind (list "Job" "CronJob") }}
  restartPolicy: {{ .Values.job.restartPolicy }}
  {{- end }}
  terminationGracePeriodSeconds: {{ .Values.terminationGracePeriodSeconds }}
  {{- with .Values.imagePullSecrets }}
  imagePullSecrets: {{ toJson . }}
  {{- end }}
  {{- with .Values.podSecurityContext }}
  securityContext:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .Values.nodeSelector }}
  nodeSelector:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .Values.tolerations }}
  tolerations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .Values.affinity }}
  affinity:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .Values.topologySpreadConstraints }}
  topologySpreadConstraints:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .Values.initContainers }}
  initContainers:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  containers:
    - name: application
      image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
      imagePullPolicy: {{ .Values.image.pullPolicy }}
      ports:
        - name: http
          containerPort: {{ .Values.service.targetPort }}
      resources:
        {{- toYaml .Values.resources | nindent 8 }}
      {{- with .Values.securityContext }}
      securityContext:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- range $field := list "env" "envFrom" "command" "args" }}
      {{- with index $.Values $field }}
      {{ $field }}:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- end }}
      {{- range $entry := list (dict "field" "readinessProbe" "value" .Values.probes.readiness) (dict "field" "livenessProbe" "value" .Values.probes.liveness) (dict "field" "startupProbe" "value" .Values.probes.startup) }}
      {{- with $entry.value }}
      {{ $entry.field }}:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- end }}
      {{- if or .Values.persistence.enabled $hasConfigMount .Values.secretMounts .Values.volumeMounts }}
      volumeMounts:
        {{- if .Values.persistence.enabled }}
        - name: data
          mountPath: {{ .Values.persistence.mountPath | quote }}
          readOnly: {{ .Values.persistence.readOnly }}
        {{- end }}
        {{- range $key,$config := .Values.configMaps }}
        {{- with $config.mountPath }}
        - name: config-{{ $key }}
          mountPath: {{ . | quote }}
          readOnly: true
        {{- end }}
        {{- end }}
        {{- range $i,$secret := .Values.secretMounts }}
        - name: secret-{{ $i }}
          mountPath: {{ $secret.mountPath | quote }}
          readOnly: true
        {{- end }}
        {{- with .Values.volumeMounts }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      {{- end }}
    {{- with .Values.sidecars }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
  {{- if or (and .Values.persistence.enabled (or (ne .Values.workload.kind "StatefulSet") .Values.persistence.existingClaim)) $hasConfigMount .Values.secretMounts .Values.volumes }}
  volumes:
    {{- if and .Values.persistence.enabled (or (ne .Values.workload.kind "StatefulSet") .Values.persistence.existingClaim) }}
    - name: data
      persistentVolumeClaim:
        claimName: {{ default (printf "%s-data" .Release.Name) .Values.persistence.existingClaim }}
    {{- end }}
    {{- range $key,$config := .Values.configMaps }}
    {{- with $config.mountPath }}
    - name: config-{{ $key }}
      configMap:
        name: {{ printf "%s-%s" $.Release.Name $key }}
    {{- end }}
    {{- end }}
    {{- range $i,$secret := .Values.secretMounts }}
    - name: secret-{{ $i }}
      secret:
        secretName: {{ $secret.name | quote }}
        optional: {{ default false $secret.optional }}
        {{- with $secret.items }}
        items: {{ toJson . }}
        {{- end }}
    {{- end }}
    {{- with .Values.volumes }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
  {{- end }}
{{- end -}}
{{- define "axiops.jobSpec" -}}
backoffLimit: {{ .Values.job.backoffLimit }}
activeDeadlineSeconds: {{ .Values.job.activeDeadlineSeconds }}
ttlSecondsAfterFinished: {{ .Values.job.ttlSecondsAfterFinished }}
template:
  {{- include "axiops.pod" . | nindent 2 }}
{{- end -}}
