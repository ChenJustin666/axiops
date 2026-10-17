{{- if and .Values.nacos.enabled (empty (trim .Values.nacos.serverAddr)) -}}
{{- fail "nacos.enabled=true 时必须填写 nacos.serverAddr" -}}
{{- end -}}
{{- if and .Values.nacos.auth.enabled (not .Values.nacos.enabled) -}}
{{- fail "开启 Nacos 认证前必须启用 Nacos" -}}
{{- end -}}
{{- if and .Values.nacos.enabled .Values.nacos.auth.enabled (empty .Values.nacos.auth.secretName) -}}
{{- fail "启用 Nacos 认证时必须选择 nacos.auth.secretName" -}}
{{- end -}}
{{- if .Values.skywalking.enabled -}}
{{- if not .Values.java.enabled -}}{{- fail "SkyWalking Java Agent 需要 java.enabled=true" -}}{{- end -}}
{{- if empty (trim .Values.skywalking.backendService) -}}{{- fail "请填写 skywalking.backendService（OAP host:11800）" -}}{{- end -}}
{{- end -}}
{{- if and .Values.java.heapDump.enabled (not .Values.java.enabled) -}}{{- fail "HeapDump 需要 java.enabled=true" -}}{{- end -}}
{{- if and .Values.probes.enabled (eq .Values.probes.type "http") (not (hasPrefix "/" .Values.probes.path)) -}}{{- fail "HTTP 探针路径必须以 / 开头" -}}{{- end -}}
{{- if .Values.ingress.enabled -}}
{{- if not .Values.service.enabled -}}{{- fail "Ingress 需要启用当前服务的 Service" -}}{{- end -}}
{{- if not .Values.ingress.items -}}{{- fail "启用 Ingress 后至少需要一项路由" -}}{{- end -}}
{{- $names := dict -}}
{{- range .Values.ingress.items -}}
{{- if hasKey $names .name -}}{{- fail "Ingress 路由名称不能重复" -}}{{- end -}}
{{- $_ := set $names .name true -}}
{{- $tls := default dict .tls -}}
{{- if $tls.enabled -}}
{{- if and (eq .className "nginx") (empty $tls.secretName) -}}{{- fail "Nginx TLS 请填写 tls.secretName" -}}{{- end -}}
{{- range .hosts -}}{{- if empty .host -}}{{- fail "TLS 路由必须填写域名，不能使用空 host" -}}{{- end -}}{{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}
