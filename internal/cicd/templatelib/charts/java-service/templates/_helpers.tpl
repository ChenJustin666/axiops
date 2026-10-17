{{- define "sinozo.application" -}}
{{- default .Release.Name .Values.application.name -}}
{{- end -}}
{{- define "sinozo.service" -}}
{{- default (printf "%s-svc" .Release.Name) .Values.service.name -}}
{{- end -}}
{{- define "sinozo.labels" -}}
app.kubernetes.io/name: {{ .Release.Name | quote }}
app.kubernetes.io/instance: {{ .Release.Name | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service | quote }}
{{- end -}}
{{- define "sinozo.selector" -}}
app.kubernetes.io/instance: {{ .Release.Name | quote }}
{{- end -}}
{{- define "sinozo.javaOptions" -}}
{{- $opts := regexReplaceAll "\\s+" (trim .Values.java.opts) " " -}}
{{- if .Values.skywalking.enabled -}}
{{- $opts = printf "%s -javaagent:/skywalking/agent/skywalking-agent.jar" $opts -}}
{{- end -}}
{{- if .Values.java.heapDump.enabled -}}
{{- $opts = printf "%s -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=%s/heapdump.hprof" $opts .Values.java.heapDump.mountPath -}}
{{- if .Values.java.heapDump.exitOnOutOfMemory -}}{{- $opts = printf "%s -XX:+ExitOnOutOfMemoryError" $opts -}}{{- end -}}
{{- end -}}
{{- trim $opts -}}
{{- end -}}
