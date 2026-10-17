import request from '@/utils/http'
import type { ServiceEnvironmentConfig } from './cicd'

export interface ServiceAssets {
  replicas?: number | null
  command: string[]
  args: string[]
  resources?: {
    cpu_request: string
    cpu_limit: string
    memory_request: string
    memory_limit: string
  } | null
  workload_kind: string
  workload_name: string
  container: string
  files: Array<{
    name: string
    filename: string
    content: string
    mount_path: string
    existing_config_map: string
  }>
  secrets: Array<{
    name: string
    group?: string
    credential_id: number
    credential_key: string
    existing_secret: string
    key: string
    env_name: string
    mount_path: string
  }>
  routes: Array<{
    name: string
    host: string
    path: string
    path_type?: 'Prefix' | 'Exact' | 'ImplementationSpecific'
    service: string
    port: number
    class: string
    tls_secret: string
    tls_enabled?: boolean
    tls_hosts?: string[]
    annotations?: Record<string, string>
    labels?: Record<string, string>
  }>
  volumes: Array<{
    name: string
    existing_claim: string
    capacity: string
    storage_class: string
    access_mode: string
    mount_path: string
    read_only: boolean
  }>
}
export const emptyServiceAssets = (): ServiceAssets => ({
  replicas: null,
  command: [],
  args: [],
  workload_kind: '',
  workload_name: '',
  container: '',
  files: [],
  secrets: [],
  routes: [],
  volumes: []
})
export async function saveServiceAssetsApi(
  environmentId: number,
  serviceId: number,
  data: { token: string; assets: ServiceAssets },
  preview: boolean
) {
  const result = await request.post<{
    saved: boolean
    validated: boolean
    config: ServiceEnvironmentConfig
    manifest: string
    secret_count: number
  }>({
    url: `/cicd/v1/environments/${environmentId}/services/${serviceId}/assets${preview ? '?preview=1' : ''}`,
    data
  })
  if (
    result?.saved !== !preview ||
    result?.validated !== true ||
    result?.config?.environment_id !== environmentId ||
    result?.config?.service_id !== serviceId ||
    !result?.config?.token ||
    typeof result.manifest !== 'string'
  ) {
    throw new Error('无法确认配置结果，请重新加载核对；不会自动重试保存')
  }
  return result
}
