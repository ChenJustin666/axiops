/**
 * 资源选择器 API 接口定义
 * 用于智能选择 Pod 标签、Namespace 标签等
 */

import request from '@/utils/http'

const BASE_PATH = '/workload/v1/selector'

// ==================== 类型定义 ====================

/**
 * 工作负载资源列表请求
 */
export interface WorkloadResourceListRequest {
  clusterUuid: string
  namespace: string
  type: 'deployment' | 'statefulset' | 'daemonset' | 'cronjob'
}
/**
 * 工作负载名称项
 */
export interface WorkloadNameItem {
  name: string
  namespace: string
  labels?: Record<string, string>
  createdAt?: number
}

/**
 * 工作负载资源列表响应
 */
export interface WorkloadResourceListResponse {
  total: number
  items: WorkloadNameItem[]
}

/**
 * Pod 选择器请求
 */
export interface PodSelectorRequest {
  clusterUuid: string
  namespace: string
  type: 'deployment' | 'statefulset' | 'daemonset' | 'cronjob'
  name: string
}

/**
 * Pod 选择器响应
 */
export interface PodSelectorResponse {
  matchLabels: Record<string, string>
  labels: Record<string, string>
}

/**
 * Namespace 标签请求
 */
export interface NamespaceLabelsRequest {
  clusterUuid: string
  namespace: string
}

/**
 * Namespace 标签响应
 */
export interface NamespaceLabelsResponse {
  labels: Record<string, string>
}

// ==================== API 接口 ====================

/**
 * 获取工作负载资源列表
 */
export async function getWorkloadResourceListApi(
  params: WorkloadResourceListRequest
): Promise<WorkloadResourceListResponse> {
  return request.get({
    url: `${BASE_PATH}/workload-list`,
    params
  })
}

/**
 * 获取工作负载 Pod 选择器
 */
export async function getWorkloadPodSelectorApi(
  params: PodSelectorRequest
): Promise<PodSelectorResponse> {
  return request.get({
    url: `${BASE_PATH}/pod-selector`,
    params
  })
}

/**
 * 获取 Namespace 标签
 */
export async function getNamespaceLabelsApi(
  params: NamespaceLabelsRequest
): Promise<NamespaceLabelsResponse> {
  return request.get({
    url: `${BASE_PATH}/namespace-labels`,
    params
  })
}

// ==================== 工作负载明细列表（环境视图） ====================

/**
 * 工作负载明细列表请求
 */
export interface WorkloadDetailListRequest {
  clusterUuid: string
  namespace: string
  type: 'deployment' | 'statefulset' | 'daemonset' | 'cronjob' | 'job'
  search?: string
}

/**
 * 工作负载明细行
 */
export interface WorkloadDetailItem {
  name: string
  namespace: string
  kind: string
  ready: string
  desiredReplicas: number
  readyReplicas: number
  images: string[]
  schedule?: string
  suspend?: boolean
  status?: string
  completions?: string
  labels?: Record<string, string>
  createdAt: number
}

/**
 * 工作负载明细列表响应
 */
export interface WorkloadDetailListResponse {
  total: number
  items: WorkloadDetailItem[]
}

/**
 * 获取工作负载明细列表（环境视图，按 namespace + kind 返回每行明细）
 */
export async function getWorkloadDetailListApi(
  params: WorkloadDetailListRequest
): Promise<WorkloadDetailListResponse> {
  return request.get({
    url: `${BASE_PATH}/workload-detail-list`,
    params
  })
}
