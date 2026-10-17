import request from '@/utils/http'

// 定义基础路径
const ALERT_DASHBOARD_BASE_PATH = '/manager/v1/alert/dashboard'

// ===================== 告警仪表盘通用类型 =====================

/** 告警仪表盘过滤条件 */
export interface AlertDashboardFilter {
  clusterUuid?: string
  projectId?: number
  workspaceId?: number
}

// ===================== 告警总览统计 =====================

/** 获取告警总览统计请求 */
export interface GetAlertOverviewRequest {
  clusterUuid?: string
  projectId?: number
  workspaceId?: number
  startTime?: number
  endTime?: number
}

/** 告警总览统计响应 */
export interface GetAlertOverviewResponse {
  totalCount: number
  firingCount: number
  resolvedCount: number
  todayNewCount: number
  todayResolvedCount: number
  avgDuration: number
  resolvedRate: number
  compareYesterday: number
}

// ===================== 告警级别统计 =====================

/** 获取告警级别统计请求 */
export interface GetAlertSeverityStatsRequest {
  clusterUuid?: string
  projectId?: number
  workspaceId?: number
  startTime?: number
  endTime?: number
}

/** 单个级别的统计信息 */
export interface SeverityStatItem {
  severity: 'critical' | 'warning' | 'info' | 'none'
  severityCn: string
  totalCount: number
  firingCount: number
  resolvedCount: number
  percentage: number
}

/** 告警级别统计响应 */
export interface GetAlertSeverityStatsResponse {
  items: SeverityStatItem[]
  totalCount: number
}

// ===================== 告警趋势分析 =====================

/** 获取告警趋势请求 */
export interface GetAlertTrendRequest {
  clusterUuid?: string
  projectId?: number
  workspaceId?: number
  days?: number
}

/** 单日趋势数据 */
export interface TrendDataPoint {
  date: string
  newCount: number
  resolvedCount: number
  firingCount: number
}

/** 告警趋势响应 */
export interface GetAlertTrendResponse {
  dataPoints: TrendDataPoint[]
  totalNew: number
  totalResolved: number
}

// ===================== 告警维度统计 =====================

/** 维度类型 */
export type DimensionType = 'cluster' | 'project' | 'workspace'

/** 获取告警维度统计请求 */
export interface GetAlertDimensionStatsRequest {
  clusterUuid?: string
  projectId?: number
  workspaceId?: number
  dimension: DimensionType
  page?: number
  pageSize?: number
}

/** 单个维度项的统计信息 */
export interface DimensionStatItem {
  dimensionId: string
  dimensionName: string
  totalCount: number
  firingCount: number
  resolvedCount: number
  criticalCount: number
  warningCount: number
  infoCount: number
}

/** 告警维度统计响应 */
export interface GetAlertDimensionStatsResponse {
  items: DimensionStatItem[]
  total: number
}

// ===================== 告警排行榜 =====================

/** 排行类型 */
export type RankingType = 'cluster' | 'project' | 'workspace' | 'rule' | 'instance'

/** 获取告警排行榜请求 */
export interface GetAlertTopRankingRequest {
  clusterUuid?: string
  projectId?: number
  workspaceId?: number
  startTime?: number
  endTime?: number
  rankingType: RankingType
  topN?: number
  severity?: 'critical' | 'warning' | 'info'
  status?: 'firing' | 'resolved'
}

/** 单个排行项 */
export interface RankingItem {
  rank: number
  itemId: string
  itemName: string
  alertCount: number
  firingCount: number
  criticalCount: number
  warningCount: number
  percentage: number
  extraInfo: string
}

/** 告警排行榜响应 */
export interface GetAlertTopRankingResponse {
  items: RankingItem[]
  totalAlerts: number
}

// ===================== 告警实时状态 =====================

/** 获取告警实时状态请求 */
export interface GetAlertRealtimeStatusRequest {
  clusterUuid?: string
  projectId?: number
  workspaceId?: number
}

/** 告警实时状态响应 */
export interface GetAlertRealtimeStatusResponse {
  firingCount: number
  criticalFiringCount: number
  warningFiringCount: number
  infoFiringCount: number
  last5MinNewCount: number
  last5MinResolved: number
  updateTimestamp: number
}

// ===================== 告警汇总报告 =====================

/** 获取告警汇总报告请求 */
export interface GetAlertSummaryReportRequest {
  clusterUuid?: string
  projectId?: number
  workspaceId?: number
  startTime?: number
  endTime?: number
}

/** 告警汇总报告响应 */
export interface GetAlertSummaryReportResponse {
  overview: GetAlertOverviewResponse
  severityStats: GetAlertSeverityStatsResponse
  trend: GetAlertTrendResponse
  topClusters: RankingItem[]
  topProjects: RankingItem[]
  topRules: RankingItem[]
}

// ===================== API 接口 =====================

/**
 * 获取告警总览统计
 */
export async function getAlertOverviewApi(params?: GetAlertOverviewRequest) {
  return request.get<GetAlertOverviewResponse>({
    url: `${ALERT_DASHBOARD_BASE_PATH}/overview`,
    params
  })
}

/**
 * 获取告警级别统计
 */
export async function getAlertSeverityStatsApi(params?: GetAlertSeverityStatsRequest) {
  return request.get<GetAlertSeverityStatsResponse>({
    url: `${ALERT_DASHBOARD_BASE_PATH}/severity`,
    params
  })
}

/**
 * 获取告警趋势分析
 */
export async function getAlertTrendApi(params?: GetAlertTrendRequest) {
  return request.get<GetAlertTrendResponse>({
    url: `${ALERT_DASHBOARD_BASE_PATH}/trend`,
    params
  })
}

/**
 * 获取告警维度统计
 */
export async function getAlertDimensionStatsApi(params: GetAlertDimensionStatsRequest) {
  return request.get<GetAlertDimensionStatsResponse>({
    url: `${ALERT_DASHBOARD_BASE_PATH}/dimension`,
    params
  })
}

/**
 * 获取告警排行榜
 */
export async function getAlertTopRankingApi(params: GetAlertTopRankingRequest) {
  return request.get<GetAlertTopRankingResponse>({
    url: `${ALERT_DASHBOARD_BASE_PATH}/ranking`,
    params
  })
}

/**
 * 获取告警实时状态
 */
export async function getAlertRealtimeStatusApi(params?: GetAlertRealtimeStatusRequest) {
  return request.get<GetAlertRealtimeStatusResponse>({
    url: `${ALERT_DASHBOARD_BASE_PATH}/realtime`,
    params
  })
}

/**
 * 获取告警汇总报告
 */
export async function getAlertSummaryReportApi(params?: GetAlertSummaryReportRequest) {
  return request.get<GetAlertSummaryReportResponse>({
    url: `${ALERT_DASHBOARD_BASE_PATH}/report`,
    params
  })
}

// ===================== 辅助函数 =====================

/**
 * 格式化持续时长
 */
export function formatDuration(seconds: number): string {
  if (seconds < 60) {
    return `${Math.round(seconds)}秒`
  } else if (seconds < 3600) {
    return `${Math.round(seconds / 60)}分钟`
  } else if (seconds < 86400) {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.round((seconds % 3600) / 60)
    return minutes > 0 ? `${hours}小时${minutes}分钟` : `${hours}小时`
  } else {
    const days = Math.floor(seconds / 86400)
    const hours = Math.round((seconds % 86400) / 3600)
    return hours > 0 ? `${days}天${hours}小时` : `${days}天`
  }
}

/**
 * 获取级别对应的颜色
 */
export function getSeverityColor(severity: string): string {
  const colorMap: Record<string, string> = {
    critical: 'var(--art-error)',
    warning: 'var(--art-warning)',
    info: 'var(--art-info)',
    none: 'var(--art-gray-400)'
  }
  return colorMap[severity] || 'var(--art-gray-400)'
}

/**
 * 获取级别对应的背景色
 */
export function getSeverityBgColor(severity: string): string {
  const colorMap: Record<string, string> = {
    critical: 'var(--art-error)',
    warning: 'var(--art-warning)',
    info: 'var(--art-info)',
    none: 'var(--art-gray-400)'
  }
  return colorMap[severity] || 'var(--art-gray-400)'
}

/**
 * 获取级别对应的中文名
 */
export function getSeverityCnName(severity: string): string {
  const nameMap: Record<string, string> = {
    critical: '严重',
    warning: '警告',
    info: '信息',
    none: '无级别'
  }
  return nameMap[severity] || severity
}

/**
 * 获取状态对应的颜色
 */
export function getStatusColor(status: string): string {
  const colorMap: Record<string, string> = {
    firing: 'var(--art-error)',
    resolved: 'var(--art-success)'
  }
  return colorMap[status] || 'var(--art-gray-400)'
}

/**
 * 获取状态对应的中文名
 */
export function getStatusCnName(status: string): string {
  const nameMap: Record<string, string> = {
    firing: '触发中',
    resolved: '已恢复'
  }
  return nameMap[status] || status
}

/**
 * 格式化百分比变化
 */
export function formatPercentageChange(value: number): string {
  const sign = value >= 0 ? '+' : ''
  return `${sign}${value.toFixed(1)}%`
}

/**
 * 获取维度类型对应的中文名
 */
export function getDimensionCnName(dimension: DimensionType): string {
  const nameMap: Record<DimensionType, string> = {
    cluster: '集群',
    project: '项目',
    workspace: '工作空间'
  }
  return nameMap[dimension] || dimension
}

/**
 * 获取排行类型对应的中文名
 */
export function getRankingTypeCnName(rankingType: RankingType): string {
  const nameMap: Record<RankingType, string> = {
    cluster: '集群',
    project: '项目',
    workspace: '工作空间',
    rule: '告警规则',
    instance: '告警实例'
  }
  return nameMap[rankingType] || rankingType
}
