import request from '@/utils/http'

// 定义基础路径
const MONITORING_BASE_PATH = '/manager/v1/monitoring'

// ========== 类型定义 ==========

/** 获取 Alertmanager 配置请求 */
export interface GetAlertmanagerConfigRequest {
  clusterUuid: string // 集群UUID (路径参数)
  namespace?: string // 命名空间，默认 monitoring
  crdType?: 'configmap' | 'secret' // 资源类型，默认 secret
  name?: string // 资源名称，默认 alertmanager-main
}

/** 获取 Prometheus 配置请求 */
export interface GetPrometheusConfigRequest {
  clusterUuid: string // 集群UUID (路径参数)
  namespace?: string // 命名空间，默认 monitoring
  name?: string // 资源名称，默认 k8s
}

/** 配置 Alertmanager 请求 */
export interface SetAlertmanagerConfigRequest {
  clusterUuid: string // 集群UUID (路径参数)
  namespace?: string // 命名空间，默认 monitoring
  crdType?: 'configmap' | 'secret' // 资源类型，默认 secret
  name?: string // 资源名称，默认 alertmanager-main
}

/** 监控配置响应 */
export interface MonitoringConfigResponse {
  config: string
}

// ========== API 接口 ==========

/**
 * 获取 Alertmanager 配置
 * @param params 请求参数
 * @returns 配置内容
 */
export async function getAlertmanagerConfigApi(params: GetAlertmanagerConfigRequest) {
  const { clusterUuid, ...queryParams } = params
  return request.get<MonitoringConfigResponse>({
    url: `${MONITORING_BASE_PATH}/alertmanager/${clusterUuid}/config`,
    params: queryParams
  })
}

/**
 * 获取 Prometheus 配置
 * @param params 请求参数
 * @returns 配置内容
 */
export async function getPrometheusConfigApi(params: GetPrometheusConfigRequest) {
  const { clusterUuid, ...queryParams } = params
  return request.get<MonitoringConfigResponse>({
    url: `${MONITORING_BASE_PATH}/prometheus/${clusterUuid}/config`,
    params: queryParams
  })
}

/**
 * 配置 Alertmanager
 * @param params 请求参数
 * @returns 操作结果
 */
export async function setAlertmanagerConfigApi(params: SetAlertmanagerConfigRequest) {
  const { clusterUuid, ...bodyParams } = params
  return request.post<string>({
    url: `${MONITORING_BASE_PATH}/alertmanager/${clusterUuid}/config`,
    data: bodyParams
  })
}
