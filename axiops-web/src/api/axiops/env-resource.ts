import request from '@/utils/http'

// 环境资源视图 API —— 与部署同源，按环境（cluster_id + namespace）只读查询 K8s 资源
// 后端：GET /cicd/v1/environments/:id/{workloads|pods|services|ingresses|pvcs|configs}

export type WorkloadKind = 'deployment' | 'statefulset' | 'daemonset' | 'cronjob' | 'job'

export interface EnvWorkloadItem {
  restart_allowed?: boolean
  name: string
  namespace: string
  kind: string
  ready: string
  desired_replicas: number
  ready_replicas: number
  images: string[]
  schedule?: string
  suspend?: boolean
  status?: string
  controller_status?: string
  status_reasons?: { reason: string; pods: number }[]
  completions?: string
  labels?: Record<string, string>
  created_at: number
}

export interface EnvRestartPreview {
  token: string
  environment_id: number
  environment_name: string
  cluster_id: number
  namespace: string
  service_id: number
  kind: string
  name: string
  is_production: boolean
  images: string[]
}

export function previewEnvRestartApi(environmentId: number, kind: string, name: string) {
  return request.get<EnvRestartPreview>({
    url: `/cicd/v1/environments/${environmentId}/restart-preview`,
    params: { kind, name }
  })
}

export function submitEnvRestartApi(
  environmentId: number,
  data: {
    kind: string
    name: string
    expected_token: string
    request_key: string
  }
) {
  return request.post<{ id: number }>({
    url: `/cicd/v1/environments/${environmentId}/restarts`,
    data
  })
}

export interface EnvPodItem {
  name: string
  namespace: string
  phase: string
  status?: string
  ready: string
  restarts: number
  node_name: string
  pod_ip: string
  images: string[]
  containers: string[]
  created_at: number
}

export interface EnvServiceItem {
  name: string
  namespace: string
  type: string
  cluster_ip: string
  external_ip?: string[]
  ports: string[]
  created_at: number
}

export interface EnvIngressItem {
  name: string
  namespace: string
  class?: string
  hosts: string[]
  address?: string[]
  created_at: number
}

export interface EnvPVCItem {
  name: string
  namespace: string
  status: string
  capacity: string
  access_modes: string
  storage_class: string
  volume_name: string
  created_at: number
}

export interface EnvConfigItem {
  name: string
  namespace: string
  kind: string
  type?: string
  keys: number
  created_at: number
}

/** 工作负载明细列表（按 kind） */
export function listEnvWorkloadsApi(
  environmentId: number,
  type: WorkloadKind,
  params: { search?: string } = {}
) {
  return request.get<EnvWorkloadItem[]>({
    url: `/cicd/v1/environments/${environmentId}/workloads`,
    params: { type, ...params }
  })
}

/** 容器组（Pod）列表 */
export function listEnvPodsApi(environmentId: number, params: { search?: string } = {}) {
  return request.get<EnvPodItem[]>({
    url: `/cicd/v1/environments/${environmentId}/pods`,
    params
  })
}

/** Service 列表 */
export function listEnvServicesApi(environmentId: number) {
  return request.get<EnvServiceItem[]>({
    url: `/cicd/v1/environments/${environmentId}/services`
  })
}

/** Ingress 列表 */
export function listEnvIngressesApi(environmentId: number) {
  return request.get<EnvIngressItem[]>({
    url: `/cicd/v1/environments/${environmentId}/ingresses`
  })
}

/** PVC 列表 */
export function listEnvPVCsApi(environmentId: number) {
  return request.get<EnvPVCItem[]>({
    url: `/cicd/v1/environments/${environmentId}/pvcs`
  })
}

/** ConfigMap / Secret 列表 */
export function listEnvConfigsApi(environmentId: number, type: 'configmap' | 'secret') {
  return request.get<EnvConfigItem[]>({
    url: `/cicd/v1/environments/${environmentId}/configs`,
    params: { type }
  })
}

export interface EnvResourceDetail {
  namespace: string
  name: string
  kind: string
  yaml: string
  redacted: boolean
  events: Array<{
    resource_kind?: string
    resource_name?: string
    type: string
    reason: string
    message: string
    count: number
    time: number
  }>
  events_error?: string
}
export function getEnvResourceDetailApi(environmentId: number, kind: string, name: string) {
  return request.get<EnvResourceDetail>({
    url: `/cicd/v1/environments/${environmentId}/resource-detail`,
    params: { kind, name }
  })
}
export function getEnvPodLogsApi(
  environmentId: number,
  name: string,
  container: string,
  previous = false
) {
  return request.get<{ logs: string }>({
    url: `/cicd/v1/environments/${environmentId}/pod-logs`,
    params: { name, container, previous: previous ? '1' : '0' }
  })
}
