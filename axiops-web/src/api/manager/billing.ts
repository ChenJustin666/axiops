import request from '@/utils/http'

// 定义基础路径
const BILLING_BASE_PATH = '/manager/v1/billing'

// ===================== 收费配置表类型定义 =====================

/** 收费配置信息 */
export interface BillingPriceConfig {
  id: number
  configName: string
  description: string
  cpuPrice: number
  memoryPrice: number
  storagePrice: number
  gpuPrice: number
  podPrice: number
  managementFee: number
  isSystem: number
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

/** 新增收费配置请求 */
export interface BillingPriceConfigAddRequest {
  configName: string
  description?: string
  cpuPrice: number
  memoryPrice: number
  storagePrice: number
  gpuPrice: number
  podPrice: number
  managementFee: number
  isSystem?: number
}

/** 更新收费配置请求 */
export interface BillingPriceConfigUpdateRequest {
  id: number
  configName: string
  description?: string
  cpuPrice: number
  memoryPrice: number
  storagePrice: number
  gpuPrice: number
  podPrice: number
  managementFee: number
  isGenerateBilling?: boolean
}

/** 搜索收费配置请求 */
export interface BillingPriceConfigSearchRequest {
  configName?: string
}

/** 搜索收费配置响应 */
export interface BillingPriceConfigSearchResponse {
  data: BillingPriceConfig[]
}

/** 根据ID获取收费配置响应 */
export interface BillingPriceConfigGetByIdResponse {
  data: BillingPriceConfig
}

// ===================== 计费配置绑定表类型定义 =====================

/** 计费配置绑定信息 */
export interface BillingConfigBinding {
  id: number
  bindingType: string
  bindingClusterUuid: string
  bindingProjectId: number
  priceConfigId: number
  billingStartTime: number
  lastBillingTime: number
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

/** 新增计费配置绑定请求 */
export interface BillingConfigBindingAddRequest {
  bindingType: 'cluster' | 'project_cluster'
  bindingClusterUuid: string
  bindingProjectId?: number
  priceConfigId: number
  billingStartTime?: number
  isGenerateBilling?: boolean
}

/** 更新计费配置绑定请求 */
export interface BillingConfigBindingUpdateRequest {
  id: number
  bindingClusterUuid?: string
  bindingProjectId?: number
  priceConfigId: number
  isGenerateBilling?: boolean
}

/** 获取计费配置绑定请求 */
export interface BillingConfigBindingGetRequest {
  bindingType: 'cluster' | 'project_cluster'
  bindingClusterUuid: string
  bindingProjectId?: number
}

/** 获取计费配置绑定响应 */
export interface BillingConfigBindingGetResponse {
  data: BillingConfigBinding
}

// ===================== 账单仪表盘类型定义 =====================

/** 仪表盘统计卡片请求 */
export interface BillingDashboardStatsRequest {
  month?: string
  clusterUuid?: string
  projectId?: number
  startTime?: number // 新增：开始时间（秒级时间戳）
  endTime?: number // 新增：结束时间（秒级时间戳）
}

/** 仪表盘统计卡片响应 */
export interface BillingDashboardStatsResponse {
  totalCost: number
  totalCostMom: number
  resourceCost: number
  resourceCostMom: number
  managementCost: number
  managementCostMom: number
  statementCount: number
  projectCount: number
}

/** 费用构成项 */
export interface CostCompositionItem {
  costName: string
  costAmount: number
  percentage: number
}

/** 费用构成请求 */
export interface BillingCostCompositionRequest {
  month?: string
  clusterUuid?: string
  projectId?: number
  startTime?: number // 新增
  endTime?: number // 新增
}

/** 费用构成响应 */
export interface BillingCostCompositionResponse {
  items: CostCompositionItem[]
  totalCost: number
}

/** 费用趋势项 */
export interface CostTrendItem {
  month: string
  totalCost: number
  resourceCost: number
  managementCost: number
}

/** 费用趋势请求 */
export interface BillingCostTrendRequest {
  months?: number
  clusterUuid?: string
  projectId?: number
}

/** 费用趋势响应 */
export interface BillingCostTrendResponse {
  items: CostTrendItem[]
}

/** 项目费用项 */
export interface ProjectCostItem {
  projectId: number
  projectName: string
  projectUuid: string
  totalCost: number
}

/** 项目费用TOP请求 */
export interface BillingProjectTopRequest {
  month?: string
  clusterUuid?: string
  topN?: number
  startTime?: number // 新增
  endTime?: number // 新增
}

/** 项目费用TOP响应 */
export interface BillingProjectTopResponse {
  items: ProjectCostItem[]
}

/** 集群费用项 */
export interface ClusterCostItem {
  clusterUuid: string
  clusterName: string
  projectCount: number
  totalCost: number
}

/** 集群费用TOP请求 */
export interface BillingClusterTopRequest {
  month?: string
  topN?: number
  startTime?: number // 新增
  endTime?: number // 新增
}

/** 集群费用TOP响应 */
export interface BillingClusterTopResponse {
  items: ClusterCostItem[]
}

// ===================== 账单中心类型定义 =====================

/** 账单明细 */
export interface BillingStatement {
  id: number
  statementNo: string
  statementType: string
  billingStartTime: number
  billingEndTime: number
  billingHours: number
  clusterUuid: string
  clusterName: string
  projectId: number
  projectName: string
  projectUuid: string
  projectClusterId: number
  bindingId: number
  cpuCapacity: string
  memCapacity: string
  storageLimit: string
  gpuCapacity: string
  podsLimit: number
  workspaceCount: number
  applicationCount: number
  priceCpu: number
  priceMemory: number
  priceStorage: number
  priceGpu: number
  pricePod: number
  priceManagementFee: number
  cpuCost: number
  memoryCost: number
  storageCost: number
  gpuCost: number
  podCost: number
  managementFee: number
  resourceCostTotal: number
  totalAmount: number
  remark: string
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

/** 账单汇总统计 */
export interface BillingStatementSummary {
  totalCount: number
  totalAmount: number
  cpuCost: number
  memoryCost: number
  storageCost: number
  gpuCost: number
  podCost: number
  managementFee: number
}

/** 账单搜索请求 */
export interface BillingStatementSearchRequest {
  startTime?: number
  endTime?: number
  clusterUuid?: string
  projectId?: number
  statementType?: string
  page?: number
  pageSize?: number
  orderField?: string
  isAsc?: boolean
}

/** 账单搜索响应 */
export interface BillingStatementSearchResponse {
  summary: BillingStatementSummary
  list: BillingStatement[]
  total: number
}

/** 批量删除账单请求 */
export interface BillingStatementBatchDelRequest {
  beforeTime: number
  clusterUuid?: string
  projectId?: number
}

/** 批量删除账单响应 */
export interface BillingStatementBatchDelResponse {
  deletedCount: number
}

// ===================== 收费配置接口 =====================

/**
 * 新增收费配置
 */
export async function addBillingPriceConfigApi(data: BillingPriceConfigAddRequest) {
  return request.post<void>({
    url: `${BILLING_BASE_PATH}/price-config`,
    data
  })
}

/**
 * 更新收费配置
 */
export async function updateBillingPriceConfigApi(data: BillingPriceConfigUpdateRequest) {
  return request.put<void>({
    url: `${BILLING_BASE_PATH}/price-config`,
    data
  })
}

/**
 * 删除收费配置
 */
export async function deleteBillingPriceConfigApi(id: number) {
  return request.del<void>({
    url: `${BILLING_BASE_PATH}/price-config/${id}`
  })
}

/**
 * 根据ID获取收费配置
 */
export async function getBillingPriceConfigByIdApi(id: number) {
  return request.get<BillingPriceConfigGetByIdResponse>({
    url: `${BILLING_BASE_PATH}/price-config/${id}`
  })
}

/**
 * 搜索收费配置
 */
export async function searchBillingPriceConfigApi(params?: BillingPriceConfigSearchRequest) {
  return request.get<BillingPriceConfigSearchResponse>({
    url: `${BILLING_BASE_PATH}/price-config`,
    params
  })
}

// ===================== 计费配置绑定接口 =====================

/**
 * 新增计费配置绑定
 */
export async function addBillingConfigBindingApi(data: BillingConfigBindingAddRequest) {
  return request.post<void>({
    url: `${BILLING_BASE_PATH}/config-binding`,
    data
  })
}

/**
 * 更新计费配置绑定
 */
export async function updateBillingConfigBindingApi(data: BillingConfigBindingUpdateRequest) {
  return request.put<void>({
    url: `${BILLING_BASE_PATH}/config-binding`,
    data
  })
}

/**
 * 删除计费配置绑定
 */
export async function deleteBillingConfigBindingApi(id: number) {
  return request.del<void>({
    url: `${BILLING_BASE_PATH}/config-binding/${id}`
  })
}

/**
 * 获取计费配置绑定
 */
export async function getBillingConfigBindingApi(params: BillingConfigBindingGetRequest) {
  return request.get<BillingConfigBindingGetResponse>({
    url: `${BILLING_BASE_PATH}/config-binding`,
    params
  })
}

// ===================== 账单仪表盘接口 =====================

/**
 * 获取仪表盘统计数据
 */
export async function getBillingDashboardStatsApi(params?: BillingDashboardStatsRequest) {
  return request.get<BillingDashboardStatsResponse>({
    url: `${BILLING_BASE_PATH}/dashboard/stats`,
    params
  })
}

/**
 * 获取费用构成
 */
export async function getBillingCostCompositionApi(params?: BillingCostCompositionRequest) {
  return request.get<BillingCostCompositionResponse>({
    url: `${BILLING_BASE_PATH}/dashboard/composition`,
    params
  })
}

/**
 * 获取费用趋势
 */
export async function getBillingCostTrendApi(params?: BillingCostTrendRequest) {
  return request.get<BillingCostTrendResponse>({
    url: `${BILLING_BASE_PATH}/dashboard/trend`,
    params
  })
}

/**
 * 获取项目费用排行
 */
export async function getBillingProjectTopApi(params?: BillingProjectTopRequest) {
  return request.get<BillingProjectTopResponse>({
    url: `${BILLING_BASE_PATH}/dashboard/project-top`,
    params
  })
}

/**
 * 获取集群费用排行
 */
export async function getBillingClusterTopApi(params?: BillingClusterTopRequest) {
  return request.get<BillingClusterTopResponse>({
    url: `${BILLING_BASE_PATH}/dashboard/cluster-top`,
    params
  })
}

// ===================== 账单中心接口 =====================

/**
 * 搜索账单
 */
export async function searchBillingStatementApi(params?: BillingStatementSearchRequest) {
  return request.get<BillingStatementSearchResponse>({
    url: `${BILLING_BASE_PATH}/statement`,
    params
  })
}

/**
 * 删除单条账单
 */
export async function deleteBillingStatementApi(id: number) {
  return request.del<void>({
    url: `${BILLING_BASE_PATH}/statement/${id}`
  })
}

/**
 * 批量删除账单
 */
export async function batchDeleteBillingStatementApi(data: BillingStatementBatchDelRequest) {
  return request.post<BillingStatementBatchDelResponse>({
    url: `${BILLING_BASE_PATH}/statement/batch-delete`,
    data
  })
}

// ===================== 辅助函数 =====================

/**
 * 格式化金额（保留2位小数，添加千分位）
 */
export function formatAmount(amount: number): string {
  return amount.toLocaleString('zh-CN', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  })
}

/**
 * 格式化金额（带人民币符号）
 */
export function formatCurrency(amount: number): string {
  return `¥ ${formatAmount(amount)}`
}

/**
 * 格式化价格（保留4位小数）
 */
export function formatPrice(price: number): string {
  return price.toFixed(4)
}

/**
 * 格式化 CPU 价格
 */
export function formatCpuPrice(price: number): string {
  return `${formatPrice(price)} 元/核/小时`
}

/**
 * 格式化内存价格
 */
export function formatMemoryPrice(price: number): string {
  return `${formatPrice(price)} 元/GiB/小时`
}

/**
 * 格式化存储价格
 */
export function formatStoragePrice(price: number): string {
  return `${formatPrice(price)} 元/GiB/小时`
}

/**
 * 格式化 GPU 价格
 */
export function formatGpuPrice(price: number): string {
  return `${formatPrice(price)} 元/卡/小时`
}

/**
 * 格式化 Pod 价格
 */
export function formatPodPrice(price: number): string {
  return `${formatPrice(price)} 元/个/小时`
}

/**
 * 格式化管理费
 */
export function formatManagementFee(fee: number): string {
  return `${formatPrice(fee)} 元/小时`
}

/**
 * 格式化环比变化
 */
export function formatMomChange(value: number): string {
  const sign = value >= 0 ? '+' : ''
  return `${sign}${value.toFixed(1)}%`
}

/**
 * 获取环比变化类型
 */
export function getMomChangeType(value: number): 'success' | 'danger' | 'info' {
  if (value > 0) return 'danger'
  if (value < 0) return 'success'
  return 'info'
}

/**
 * 格式化月份显示
 */
export function formatMonth(month: string): string {
  const [year, mon] = month.split('-')
  return `${year}年${parseInt(mon)}月`
}

/**
 * 获取当前月份（格式：2025-01）
 */
export function getCurrentMonth(): string {
  const now = new Date()
  const year = now.getFullYear()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  return `${year}-${month}`
}

/**
 * 获取当月的起止时间戳（秒级）
 */
export function getCurrentMonthRange(): { startTime: number; endTime: number } {
  const now = new Date()
  const startOfMonth = new Date(now.getFullYear(), now.getMonth(), 1)
  const endOfMonth = new Date(now.getFullYear(), now.getMonth() + 1, 0, 23, 59, 59)
  return {
    startTime: Math.floor(startOfMonth.getTime() / 1000),
    endTime: Math.floor(endOfMonth.getTime() / 1000)
  }
}

/**
 * 获取指定月份的起止时间戳（秒级）
 */
export function getMonthRange(month: string): { startTime: number; endTime: number } {
  const [year, mon] = month.split('-').map(Number)
  const startOfMonth = new Date(year, mon - 1, 1)
  const endOfMonth = new Date(year, mon, 0, 23, 59, 59)
  return {
    startTime: Math.floor(startOfMonth.getTime() / 1000),
    endTime: Math.floor(endOfMonth.getTime() / 1000)
  }
}

/**
 * 获取指定年份的起止时间戳（秒级）
 */
export function getYearRange(year: string): { startTime: number; endTime: number } {
  const y = parseInt(year)
  const startOfYear = new Date(y, 0, 1)
  const endOfYear = new Date(y, 11, 31, 23, 59, 59)
  return {
    startTime: Math.floor(startOfYear.getTime() / 1000),
    endTime: Math.floor(endOfYear.getTime() / 1000)
  }
}

/**
 * 获取最近N个月份列表
 */
export function getRecentMonths(count: number): string[] {
  const months: string[] = []
  const now = new Date()
  for (let i = 0; i < count; i++) {
    const date = new Date(now.getFullYear(), now.getMonth() - i, 1)
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    months.push(`${year}-${month}`)
  }
  return months
}

/**
 * 费用构成颜色列表
 */
export const COST_COMPOSITION_COLORS = [
  '#3b82f6', // CPU费用 - 蓝色
  '#10b981', // 内存费用 - 绿色
  '#f59e0b', // 存储费用 - 橙色
  '#8b5cf6', // GPU费用 - 紫色
  '#06b6d4', // Pod费用 - 青色
  '#ec4899' // 管理费 - 粉色
]

/**
 * 获取费用构成颜色
 */
export function getCostCompositionColor(index: number): string {
  return COST_COMPOSITION_COLORS[index % COST_COMPOSITION_COLORS.length]
}

/** 立即生成账单请求 */
export interface BillingStatementGenerateRequest {
  clusterUuid?: string
  projectId?: number
}

/** 立即生成账单 */
export async function generateBillingStatementApi(data: BillingStatementGenerateRequest) {
  return request.post<string>({
    url: `${BILLING_BASE_PATH}/statement/generate`,
    data
  })
}

// 导出账单请求参数
export interface BillingStatementExportRequest {
  startTime?: number
  endTime?: number
  clusterUuid?: string
  projectId?: number
  statementType?: string
  orderField?: string
  isAsc?: boolean
}

// 导出账单Excel
export async function exportBillingStatementApi(params: BillingStatementExportRequest) {
  return request.get<string>({
    url: `${BILLING_BASE_PATH}/statement/export`,
    responseType: 'blob',
    timeout: 60000,
    params
  })
}
