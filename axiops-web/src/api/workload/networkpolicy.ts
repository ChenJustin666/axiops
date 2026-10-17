/**
 * NetworkPolicy API 接口定义
 */

import request from '@/utils/http'

const BASE_PATH = '/workload/v1/networkpolicy'

// ==================== 类型定义 ====================

/**
 * NetworkPolicy 标签选择器表达式
 */
export interface NPMatchExpression {
  key?: string
  operator?: string // In, NotIn, Exists, DoesNotExist
  values?: string[]
}

/**
 * NetworkPolicy 端口信息
 */
export interface NPNetworkPolicyPortInfo {
  protocol?: string // TCP, UDP, SCTP
  port?: number
  endPort?: number
  namedPort?: string
}

/**
 * NetworkPolicy IP 块信息
 */
export interface NPIPBlockInfo {
  cidr: string
  except?: string[]
}

/**
 * NetworkPolicy 对端信息
 */
export interface NPNetworkPolicyPeerInfo {
  podSelector?: Record<string, string>
  podSelectorExprs?: NPMatchExpression[]
  namespaceSelector?: Record<string, string>
  namespaceSelectorExprs?: NPMatchExpression[]
  ipBlock?: NPIPBlockInfo
}

/**
 * NetworkPolicy 入站规则信息
 */
export interface NPIngressRuleInfo {
  ports?: NPNetworkPolicyPortInfo[]
  from?: NPNetworkPolicyPeerInfo[]
}

/**
 * NetworkPolicy 出站规则信息
 */
export interface NPEgressRuleInfo {
  ports?: NPNetworkPolicyPortInfo[]
  to?: NPNetworkPolicyPeerInfo[]
}

/**
 * NetworkPolicy 列表项
 */
export interface NetworkPolicyListItem {
  name: string
  namespace: string
  age: string
  creationTimestamp: number
  labels?: Record<string, string>
  annotations?: Record<string, string>
}

/**
 * NetworkPolicy 列表响应
 */
export interface NetworkPolicyListResponse {
  total: number
  items: NetworkPolicyListItem[]
}

/**
 * NetworkPolicy 详情
 */
export interface NetworkPolicyDetail {
  name: string
  namespace: string
  podSelector: Record<string, string>
  podSelectorMatchExpressions?: NPMatchExpression[]
  ingressRules?: NPIngressRuleInfo[]
  egressRules?: NPEgressRuleInfo[]
  policyTypes?: string[]
  labels?: Record<string, string>
  annotations?: Record<string, string>
  age: string
  creationTimestamp: number
}

/**
 * NetworkPolicy YAML 创建/更新请求
 */
export interface NetworkPolicyYamlRequest {
  clusterUuid: string
  namespace: string
  yamlStr: string
}

/**
 * NetworkPolicy 受影响的 Pod
 */
export interface NetworkPolicyAffectedPod {
  name: string
  namespace: string
  labels: Record<string, string>
  matched: boolean
  reason: string
}

/**
 * NetworkPolicy 受影响的 Pod 响应
 */
export interface NetworkPolicyAffectedPodsResponse {
  networkPolicyName: string
  networkPolicyNamespace: string
  affectedPods: NetworkPolicyAffectedPod[]
  totalAffected: number
}

/**
 * NetworkPolicy 列表请求参数
 */
export interface NetworkPolicyListRequest {
  clusterUuid: string
  namespace?: string
  search?: string
  labelSelector?: string
}

/**
 * NetworkPolicy 名称请求参数
 */
export interface NetworkPolicyNameRequest {
  clusterUuid: string
  namespace?: string
  name?: string
}

// ==================== API 接口 ====================

/**
 * 获取 NetworkPolicy 列表
 * @param params 查询参数
 * @returns NetworkPolicy 列表
 */
export async function getNetworkPolicyListApi(params: NetworkPolicyListRequest) {
  return request.get<NetworkPolicyListResponse>({
    url: BASE_PATH,
    params
  })
}

/**
 * 获取 NetworkPolicy 详情
 * @param params 查询参数
 * @returns NetworkPolicy 详情
 */
export async function getNetworkPolicyDetailApi(params: NetworkPolicyNameRequest) {
  return request.get<NetworkPolicyDetail>({
    url: `${BASE_PATH}/detail`,
    params
  })
}

/**
 * 获取 NetworkPolicy YAML
 * @param params 查询参数
 * @returns YAML 字符串
 */
export async function getNetworkPolicyYamlApi(params: NetworkPolicyNameRequest) {
  return request.get<string>({
    url: `${BASE_PATH}/yaml`,
    params
  })
}

/**
 * 创建 NetworkPolicy
 * @param data 创建参数
 * @returns 操作结果
 */
export async function createNetworkPolicyApi(data: NetworkPolicyYamlRequest) {
  return request.post<string>({
    url: BASE_PATH,
    data
  })
}

/**
 * 更新 NetworkPolicy
 * @param data 更新参数
 * @returns 操作结果
 */
export async function updateNetworkPolicyApi(data: NetworkPolicyYamlRequest) {
  return request.put<string>({
    url: BASE_PATH,
    data
  })
}

/**
 * 删除 NetworkPolicy
 * @param params 删除参数
 * @returns 操作结果
 */
export async function deleteNetworkPolicyApi(params: NetworkPolicyNameRequest) {
  return request.del<string>({
    url: BASE_PATH,
    params
  })
}

/**
 * 获取受 NetworkPolicy 影响的 Pod
 * @param params 查询参数
 * @returns 受影响的 Pod 列表
 */
export async function getNetworkPolicyAffectedPodsApi(params: NetworkPolicyNameRequest) {
  return request.get<NetworkPolicyAffectedPodsResponse>({
    url: `${BASE_PATH}/affected-pods`,
    params
  })
}
