import request from '@/utils/http'

// 定义基础路径
const BASE_URL = '/manager/v1/resource/dashboard'

// ==================== 类型定义 ====================

/** 筛选条件 */
export interface FilterCondition {
  clusterUuid: string
  clusterName: string
  projectId: number
  projectName: string
  filteredClusterCount: number
  filteredProjectCount: number
  filteredWorkspaceCount: number
}

/** 顶部统计卡片 - 8个卡片 */
export interface DashboardSummaryCards {
  // 数量统计
  clusterTotalCount: number
  projectTotalCount: number
  workspaceTotalCount: number

  // CPU物理分配率 = limit / physical * 100
  cpuPhysical: number // 物理CPU总量
  cpuLimit: number // 已分配配额（不是超分）
  cpuPhysicalRate: number // 物理分配率

  // 内存物理分配率
  memPhysical: number
  memLimit: number
  memPhysicalRate: number

  // GPU物理分配率
  gpuPhysical: number
  gpuLimit: number
  gpuPhysicalRate: number

  // 存储分配率
  storagePhysical: number
  storageLimit: number
  storagePhysicalRate: number

  // Pod分配率
  podPhysical: number
  podLimit: number
  podPhysicalRate: number
}

/** 资源概览项 - 三层数据 */
export interface ResourceOverviewItem {
  physical: number // 物理资源总量
  limit: number // 分配总数（原始配额）
  capacity: number // 超分总数（超分后容量）
  unit: string
}

/** 资源分配概览 */
export interface ResourceAllocationOverview {
  cpu: ResourceOverviewItem // 物理数、分配总数、超分总数
  mem: ResourceOverviewItem
  gpu: ResourceOverviewItem
  storage: ResourceOverviewItem // 存储总数、分配总数（无超分）
  pod: ResourceOverviewItem // Pod总数、分配总数（无超分）
}

/** 超分收益项 */
export interface OversellSavingItem {
  limitTotal: number // 原始配额
  capacityTotal: number // 超分后容量
  savingAmount: number // 节约量 = capacity - limit
  savingRate: number // 超分率
  unit: string
}

/** 超分收益 */
export interface ResourceOversellSavings {
  cpu: OversellSavingItem
  mem: OversellSavingItem
  gpu: OversellSavingItem
}

/** 获取资源仪表盘汇总数据请求 */
export interface GetResourceDashboardSummaryRequest {
  clusterUuid?: string
  projectId?: number
}

/** 获取资源仪表盘汇总数据响应 */
export interface GetResourceDashboardSummaryResponse {
  currentFilter: FilterCondition
  summaryCards: DashboardSummaryCards
  allocationOverview: ResourceAllocationOverview
  oversellSavings: ResourceOversellSavings
}

/** 集群排行项 */
export interface ClusterRankingItem {
  rank: number
  clusterUuid: string
  clusterName: string
  environment: string
  region: string
  provider: string
  projectCount: number
  workspaceCount: number
  cpuCapacity: number
  cpuLimit: number
  cpuAllocated: number
  cpuAllocationRate: number
  cpuOversellRate: number
  memCapacityGib: number
  memLimitGib: number
  memAllocatedGib: number
  memAllocationRate: number
  memOversellRate: number
  gpuCapacity: number
  gpuLimit: number
  gpuAllocated: number
  gpuAllocationRate: number
  gpuOversellRate: number
  storageLimitGib: number
  storageAllocatedGib: number
  storageAllocationRate: number
  podsLimit: number
  podsAllocated: number
  podsAllocationRate: number
  overallUsageRate: number
}

/** 获取集群资源排行请求 */
export interface GetClusterResourceRankingRequest {
  clusterUuid?: string
  projectId?: number
  sortBy?: string
  topN?: number
}

/** 获取集群资源排行响应 */
export interface GetClusterResourceRankingResponse {
  items: ClusterRankingItem[]
  total: number
}

/** 项目排行项 */
export interface ProjectRankingItem {
  rank: number
  projectId: number
  projectName: string
  projectUuid: string
  isSystem: boolean
  clusterCount: number
  workspaceCount: number
  primaryClusterUuid: string
  primaryClusterName: string
  cpuLimit: number
  cpuCapacity: number
  cpuAllocated: number
  cpuUsageRate: number
  memLimitGib: number
  memCapacityGib: number
  memAllocatedGib: number
  memUsageRate: number
  gpuLimit: number
  gpuCapacity: number
  gpuAllocated: number
  gpuUsageRate: number
  storageLimitGib: number
  storageAllocatedGib: number
  storageUsageRate: number
  podsLimit: number
  podsAllocated: number
  podsUsageRate: number
  overallUsageRate: number
}

/** 获取项目资源排行请求 */
export interface GetProjectResourceRankingRequest {
  clusterUuid?: string
  projectId?: number
  sortBy?: string
  topN?: number
}

/** 获取项目资源排行响应 */
export interface GetProjectResourceRankingResponse {
  items: ProjectRankingItem[]
  total: number
}

/** 工作空间排行项 */
export interface WorkspaceRankingItem {
  rank: number
  workspaceId: number
  workspaceName: string
  workspaceUuid: string
  namespace: string
  projectId: number
  projectName: string
  clusterUuid: string
  clusterName: string
  cpuLimit: number
  cpuAllocated: number
  cpuUsageRate: number
  memLimitGib: number
  memAllocatedGib: number
  memUsageRate: number
  gpuLimit: number
  gpuAllocated: number
  gpuUsageRate: number
  storageLimitGib: number
  storageAllocatedGib: number
  storageUsageRate: number
  podsLimit: number
  podsAllocated: number
  podsUsageRate: number
  overallUsageRate: number
}

/** 获取工作空间资源排行请求 */
export interface GetWorkspaceResourceRankingRequest {
  clusterUuid?: string
  projectId?: number
  sortBy?: string
  topN?: number
}

/** 获取工作空间资源排行响应 */
export interface GetWorkspaceResourceRankingResponse {
  items: WorkspaceRankingItem[]
  total: number
}

// ==================== API 接口 ====================

/**
 * 获取资源仪表盘汇总数据
 */
export async function getResourceDashboardSummaryApi(params?: GetResourceDashboardSummaryRequest) {
  return request.get<GetResourceDashboardSummaryResponse>({
    url: `${BASE_URL}/summary`,
    params
  })
}

/**
 * 获取集群资源排行
 */
export async function getClusterResourceRankingApi(params?: GetClusterResourceRankingRequest) {
  return request.get<GetClusterResourceRankingResponse>({
    url: `${BASE_URL}/ranking/cluster`,
    params
  })
}

/**
 * 获取项目资源排行
 */
export async function getProjectResourceRankingApi(params?: GetProjectResourceRankingRequest) {
  return request.get<GetProjectResourceRankingResponse>({
    url: `${BASE_URL}/ranking/project`,
    params
  })
}

/**
 * 获取工作空间资源排行
 */
export async function getWorkspaceResourceRankingApi(params?: GetWorkspaceResourceRankingRequest) {
  return request.get<GetWorkspaceResourceRankingResponse>({
    url: `${BASE_URL}/ranking/workspace`,
    params
  })
}

// ==================== 辅助函数 ====================

/**
 * 格式化资源值
 */
export function formatResourceValue(value: number, unit: string, decimals: number = 0): string {
  return `${value.toFixed(decimals)} ${unit}`
}

/**
 * 获取分配率对应的颜色
 */
export function getAllocationRateColor(rate: number): string {
  if (rate >= 90) return 'var(--art-error)'
  if (rate >= 70) return 'var(--art-warning)'
  return 'var(--art-success)'
}

/**
 * 获取环境标签样式类
 */
export function getEnvironmentTagClass(env: string): string {
  const classMap: Record<string, string> = {
    production: 'env-prod',
    staging: 'env-staging',
    testing: 'env-test',
    development: 'env-dev'
  }
  return classMap[env] || 'env-default'
}

/**
 * 获取环境中文名
 */
export function getEnvironmentCnName(env: string): string {
  const nameMap: Record<string, string> = {
    production: '生产',
    staging: '预发',
    testing: '测试',
    development: '开发'
  }
  return nameMap[env] || env
}

/**
 * 格式化百分比
 */
export function formatPercentage(value: number, decimals: number = 1): string {
  return `${value.toFixed(decimals)}%`
}

/**
 * 格式化超分收益
 */
export function formatSavingAmount(value: number, unit: string): string {
  const sign = value >= 0 ? '+' : ''
  return `${sign}${value.toFixed(0)} ${unit}`
}
