{{- if .Values.ingress.enabled -}}
{{- if or (not .Values.service.enabled) (has .Values.workload.kind (list "Job" "CronJob")) -}}
{{- fail "Ingress requires an enabled Service and a long-running workload" -}}
{{- end -}}
{{- if not .Values.ingress.hosts -}}{{- fail "Ingress requires at least one host and path" -}}{{- end -}}
{{- end -}}
{{- if .Values.autoscaling.enabled -}}
{{- if not (has .Values.workload.kind (list "Deployment" "StatefulSet")) -}}{{- fail "HPA supports Deployment or StatefulSet only" -}}{{- end -}}
{{- if gt (int .Values.autoscaling.minReplicas) (int .Values.autoscaling.maxReplicas) -}}{{- fail "HPA minReplicas must not exceed maxReplicas" -}}{{- end -}}
{{- if and (not .Values.autoscaling.targetCPUUtilizationPercentage) (not .Values.autoscaling.targetMemoryUtilizationPercentage) -}}{{- fail "HPA needs at least one resource metric" -}}{{- end -}}
{{- if and .Values.autoscaling.targetCPUUtilizationPercentage (not (dig "requests" "cpu" "" .Values.resources)) -}}{{- fail "CPU HPA requires resources.requests.cpu" -}}{{- end -}}
{{- if and .Values.autoscaling.targetMemoryUtilizationPercentage (not (dig "requests" "memory" "" .Values.resources)) -}}{{- fail "Memory HPA requires resources.requests.memory" -}}{{- end -}}
{{- end -}}
{{- if and .Values.pdb.enabled (has .Values.workload.kind (list "Job" "CronJob")) -}}{{- fail "PDB is for long-running workloads" -}}{{- end -}}
{{- if and .Values.rbac.create (or (not .Values.serviceAccount.create) (not .Values.rbac.rules)) -}}{{- fail "RBAC requires an explicitly created ServiceAccount and rules" -}}{{- end -}}
{{- if and .Values.persistence.enabled (not .Values.persistence.existingClaim) (eq .Values.workload.kind "Deployment") (or (gt (int .Values.replicaCount) 1) (and .Values.autoscaling.enabled (gt (int .Values.autoscaling.maxReplicas) 1))) (not (has "ReadWriteMany" .Values.persistence.accessModes)) -}}{{- fail "A shared multi-replica Deployment PVC needs ReadWriteMany; use StatefulSet for per-replica claims" -}}{{- end -}}
{{- if and .Values.persistence.enabled (eq .Values.workload.kind "DaemonSet") (not .Values.persistence.existingClaim) (not (has "ReadWriteMany" .Values.persistence.accessModes)) -}}{{- fail "A shared DaemonSet PVC needs ReadWriteMany or an explicitly reviewed existing claim" -}}{{- end -}}

{{- range $key,$secret := .Values.secrets -}}
{{- $data := deepCopy (default dict $secret.stringData) -}}
{{- range $field := $secret.fields -}}{{- $_ := set $data $field.key $field.value -}}{{- end -}}
{{- $type := default "Opaque" $secret.type -}}
{{- if and (eq $type "kubernetes.io/tls") (or (not (hasKey $data "tls.crt")) (not (hasKey $data "tls.key"))) -}}{{- fail "TLS Secret requires tls.crt and tls.key credential fields" -}}{{- end -}}
{{- if and (eq $type "kubernetes.io/basic-auth") (not (or (hasKey $data "username") (hasKey $data "password"))) -}}{{- fail "Basic auth Secret requires username or password" -}}{{- end -}}
{{- if and (eq $type "kubernetes.io/ssh-auth") (not (hasKey $data "ssh-privatekey")) -}}{{- fail "SSH Secret requires ssh-privatekey" -}}{{- end -}}
{{- end -}}
