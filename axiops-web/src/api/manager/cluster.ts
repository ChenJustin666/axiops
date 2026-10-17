import request from '@/utils/http'

// 定义基础路径
const CLUSTER_BASE_PATH = '/manager/v1/cluster'

// ========== 类型定义 ==========

/** 集群基本信息 */
export interface Cluster {
  id: number
  name: string
  uuid: string
  avatar: string
  environment: string
  clusterType: string
  version: string
  status: number
  healthStatus: number
  cpuUsage: number
  memoryUsage: number
  podUsage: number
  storageUsage: number
  createdAt: number
}

/** 集群详细信息 */
export interface ClusterDetail {
  id: number
  name: string
  avatar: string
  uuid: string
  description: string
  clusterType: string
  environment: string
  region: string
  zone: string
  datacenter: string
  provider: string
  isManaged: number
  nodeLb: string
  masterLb: string
  ingressDomain: string
  status: number
  healthStatus: number
  lastSyncAt: number
  version: string
  gitCommit: string
  platform: string
  versionBuildAt: number
  clusterCreatedAt: number
  costCenter: string
  businessUnit: string
  ownerTeam: string
  ownerEmail: string
  priority: number
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

/** 添加集群请求 - multipart/form-data 格式 */
export interface AddClusterRequest {
  name: string
  avatar?: string
  allowCreate: boolean
  description?: string
  clusterType: 'standard' | 'edge' | 'serverless'
  environment: string
  region?: string
  zone?: string
  datacenter?: string
  provider: 'aws' | 'azure' | 'gcp' | 'alibaba' | 'tencent' | 'huawei' | 'self-hosted'
  isManaged?: number
  nodeLb?: string
  masterLb?: string
  ingressDomain?: string
  authType: 'kubeconfig' | 'token' | 'certificate' | 'incluster'
  apiServerHost?: string
  kubeFile?: string
  token?: string
  caCert?: string
  caFile?: string
  clientCert?: string
  certFile?: string
  clientKey?: string
  keyFile?: string
  insecureSkipVerify?: number
  // ========== 费用配置字段 ==========
  priceConfigId: number // 价格配置ID（必填）
  billingStartTime?: number // 计费开始时间（秒级时间戳，可选）
  // ========== Prometheus 配置字段（可选） ==========
  enablePrometheus?: number // 是否配置 Prometheus：0-否，1-是
  prometheusUrl?: string // Prometheus 地址
  prometheusPort?: number // Prometheus 端口
  prometheusProtocol?: string // 协议：http/https
  prometheusAuthEnabled?: number // 是否启用认证：0-否，1-是
  prometheusAuthType?: string // 认证类型：none/basic/token/apikey
  prometheusUsername?: string // Basic 认证用户名
  prometheusPassword?: string // Basic 认证密码
  prometheusToken?: string // Token 认证
  prometheusAccessKey?: string // API Key
  prometheusAccessSecret?: string // API Secret
  prometheusTlsEnabled?: number // 是否启用 TLS：0-否，1-是
  prometheusInsecureSkipVerify?: number // 是否跳过证书验证：0-否，1-是
  prometheusCaCert?: string // CA 证书
  prometheusClientCert?: string // 客户端证书
  prometheusClientKey?: string // 客户端密钥
}

/** 更新集群请求 - JSON 格式 */
export interface UpdateClusterRequest {
  id: number
  name?: string
  description?: string
  clusterType?: 'standard' | 'edge' | 'serverless'
  environment?: string
  region?: string
  zone?: string
  datacenter?: string
  provider?: 'aws' | 'azure' | 'gcp' | 'alibaba' | 'tencent' | 'huawei' | 'self-hosted'
  isManaged?: number
  nodeLb?: string
  masterLb?: string
  ingressDomain?: string
}

/** 修改集群头像请求 */
export interface ClusterUpdateAvatarRequest {
  id: number
  avatar?: string | File
}

/** 更新集群认证信息请求 */
export interface UpdateClusterAuthInfoRequest {
  id: number
  authType: 'kubeconfig' | 'token' | 'certificate' | 'incluster'
  apiServerHost?: string
  kubeFile?: string
  token?: string
  caCert?: string
  caFile?: string
  clientCert?: string
  certFile?: string
  clientKey?: string
  keyFile?: string
  insecureSkipVerify?: number
}

/** 搜索集群请求 */
export interface SearchClusterRequest {
  page?: number
  pageSize?: number
  orderStr?: string
  isAsc?: boolean
  name?: string
  uuid?: string
  environment?: string
}

/** 搜索集群响应 */
export interface SearchClusterResponse {
  items: Cluster[]
  total: number
}

/** 集群审计日志 */
export interface ClusterAudit {
  id: number
  clusterUuid: string
  operator: string
  auditType: string
  auditContent: string
  color: number
  createdAt: number
}

/** 搜索集群审计日志请求 */
export interface SearchClusterAuditRequest {
  page?: number
  pageSize?: number
  orderStr?: string
  isAsc?: boolean
  clusterUuid: string
}

/** 搜索集群审计日志响应 */
export interface SearchClusterAuditResponse {
  items: ClusterAudit[]
  total: number
}

/** 集群认证信息 */
export interface ClusterAuthInfo {
  id: number
  clusterUuid: string
  authType: string
  apiServerHost: string
  kubeFile: string
  token: string
  caCert: string
  caFile: string
  clientCert: string
  certFile: string
  clientKey: string
  keyFile: string
  insecureSkipVerify: number
  createdAt: number
  updatedAt: number
}

/** 集群资源信息 */
export interface ClusterResourceInfo {
  id: number
  clusterUuid: string
  cpuPhysicalCapacity: number
  cpuAllocatedTotal: number
  cpuCapacityTotal: number
  memPhysicalCapacity: number
  memAllocatedTotal: number
  memCapacityTotal: number
  storagePhysicalCapacity: number
  storageAllocatedTotal: number
  gpuPhysicalCapacity: number
  gpuAllocatedTotal: number
  gpuCapacityTotal: number
  gpuUsedTotal: number
  podsPhysicalCapacity: number
  podsAllocatedTotal: number
}

/** 测试集群连通性请求 */
export interface TestClusterConnectivityRequest {
  authType: 'kubeconfig' | 'token' | 'certificate' | 'incluster'
  apiServerHost?: string
  kubeFile?: string
  token?: string
  caCert?: string
  caFile?: string
  clientCert?: string
  certFile?: string
  clientKey?: string
  keyFile?: string
  insecureSkipVerify?: number
}

/** 修改集群资源的存储容量请求 */
export interface UpdateClusterStorageCapacityRequest {
  resourceId: number
  storagePhysicalCapacity: number
}

/** 集群网络配置信息 */
export interface ClusterNetwork {
  id: number
  clusterUuid: string
  clusterCidr: string
  serviceCidr: string
  nodeCidrMaskSize: number
  dnsDomain: string
  dnsServiceIp: string
  dnsProvider: string
  cniPlugin: string
  cniVersion: string
  proxyMode: string
  ingressController: string
  ingressClass: string
  ipv6Enabled: boolean
  dualStackEnabled: boolean
  mtuSize: number
  enableNodePort: boolean
  nodePortRange: string
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface LogQueryModeOptionsResponse {
  loki: boolean
  elasticsearch: boolean
  rawQuery: boolean
  export: boolean
}

/**
 * 添加集群
 */
export async function addClusterApi(data: AddClusterRequest) {
  const formData = new FormData()
  Object.keys(data).forEach((key) => {
    const value = data[key as keyof AddClusterRequest]
    if (value !== undefined && value !== null) {
      formData.append(key, String(value))
    }
  })

  return request.post<string>({
    url: `${CLUSTER_BASE_PATH}`,
    data: formData,
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}

/**
 * 更新集群基本信息
 */
export async function updateClusterApi(data: UpdateClusterRequest) {
  const { id, ...params } = data
  return request.put<string>({
    url: `${CLUSTER_BASE_PATH}/${id}`,
    data: params
  })
}

/**
 * 修改集群头像
 */
export async function updateClusterAvatarApi(data: ClusterUpdateAvatarRequest) {
  const { id, avatar } = data
  const formData = new FormData()
  if (avatar) {
    if (typeof avatar === 'object' && avatar instanceof File) {
      formData.append('avatar', avatar)
    } else {
      formData.append('avatar', avatar as string)
    }
  }
  return request.put<string>({
    url: `${CLUSTER_BASE_PATH}/${id}/avatar`,
    data: formData,
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}

/**
 * 更新集群认证信息
 */
export async function updateClusterAuthInfoApi(data: UpdateClusterAuthInfoRequest) {
  const { id, ...params } = data
  return request.put<string>({
    url: `${CLUSTER_BASE_PATH}/${id}/auth`,
    data: params
  })
}

/**
 * 删除集群
 */
export async function deleteClusterApi(id: number) {
  return request.del<string>({
    url: `${CLUSTER_BASE_PATH}/${id}`,
    timeout: 600000
  })
}

/**
 * 获取集群详情
 */
export async function getClusterDetailApi(id: number) {
  return request.get<ClusterDetail>({
    url: `${CLUSTER_BASE_PATH}/${id}`
  })
}

/**
 * 搜索集群列表
 */
export async function searchClusterApi(params?: SearchClusterRequest) {
  return request.get<SearchClusterResponse>({
    url: `${CLUSTER_BASE_PATH}`,
    params
  })
}

export async function getClusterNamespaceListApi(clusterUuid: string) {
  return request.get<string[]>({
    url: `${CLUSTER_BASE_PATH}/${clusterUuid}/namespaces`
  })
}

/**
 * 获取集群审计日志
 */
export async function searchClusterAuditApi(params: SearchClusterAuditRequest) {
  return request.get<SearchClusterAuditResponse>({
    url: `${CLUSTER_BASE_PATH}/audit`,
    params
  })
}

/**
 * 获取集群认证信息
 */
export async function getClusterAuthInfoApi(id: number) {
  return request.get<ClusterAuthInfo>({
    url: `${CLUSTER_BASE_PATH}/${id}/auth`
  })
}

/**
 * 获取集群资源信息
 */
export async function getClusterResourceInfoApi(id: number) {
  return request.get<ClusterResourceInfo>({
    url: `${CLUSTER_BASE_PATH}/${id}/resource`
  })
}

/**
 * 测试集群连通性
 */
export async function testClusterConnectivityApi(data: TestClusterConnectivityRequest) {
  return request.post<string>({
    url: `${CLUSTER_BASE_PATH}/test`,
    data
  })
}

/**
 * 获取某个集群的Ingress域名列表
 */
export async function getClusterIngressDomainsApi(uuid: string) {
  return request.get<string[]>({
    url: `${CLUSTER_BASE_PATH}/${uuid}/ingress`
  })
}

/**
 * 设置集群的存储容量
 */
export async function updateClusterStorageCapacityApi(data: UpdateClusterStorageCapacityRequest) {
  const { resourceId, ...params } = data
  return request.put<string>({
    url: `${CLUSTER_BASE_PATH}/${resourceId}/storage`,
    data: params
  })
}

/**
 * 获取集群日志查询模式可用性
 */
export async function getLogQueryModeOptionsApi(clusterUuid: string, namespace?: string) {
  return request.get<LogQueryModeOptionsResponse>({
    url: '/manager/v1/logging/query-mode/options',
    params: { clusterUuid, namespace }
  })
}

/**
 * 获取集群网络配置
 */
export async function getClusterNetworkApi(clusterUuid: string) {
  return request.get<ClusterNetwork>({
    url: `${CLUSTER_BASE_PATH}/${clusterUuid}/network`
  })
}
