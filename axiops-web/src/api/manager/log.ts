import request from '@/utils/http'

// 日志查询与下载 API 封装

const LOG_BASE_PATH = '/manager/v1/logging'

export type LogSourceType = 'container' | 'system'
export type LogQueryMode = 'platform' | 'loki' | 'es'
export type LogAlertSeverity = 'critical' | 'warning' | 'info'
export type LogAlertConditionType =
  | 'match_any'
  | 'count_gt'
  | 'count_gte'
  | 'count_lt'
  | 'count_lte'
  | 'no_data'
  | 'flatline'

// ========== 类型定义 ==========

/** 单条日志记录 */
export interface LogRecord {
  timestamp: number
  message: string
  backendType: string
  labels: Record<string, string>
  raw: string
}

/** 日志查询请求
 * 对外 HTTP 协议继续使用 camelCase，内部真实日志字段口径按后端处理。
 */
export interface LogQueryRequest {
  sourceType: LogSourceType
  logType?: LogSourceType
  clusterUuid: string
  projectUuid?: string
  namespace?: string
  application?: string
  resourceName?: string
  podName?: string
  containerName?: string
  host?: string
  podIp?: string
  hosts?: string[]
  logSourceType?: string
  level?: string
  queryMode?: LogQueryMode
  mode?: 'form' | 'code'
  expr?: string
  queryText?: string
  command?: string
  message?: string
  keyword?: string
  start: string
  end: string
  limit?: number
  direction?: 'forward' | 'backward'
  nextToken?: string
}

/** 日志查询响应 */
export interface LogQueryResponse {
  backendType: string
  queryMode?: string
  dataStream?: string
  indexPattern?: string
  nextToken?: string
  list: LogRecord[]
}

/** 日志下载请求（与查询基本一致） */
export interface LogDownloadRequest extends LogQueryRequest {
  limit?: number
}

export interface LogLabelKeysRequest {
  sourceType: LogSourceType
  clusterUuid: string
  queryMode: LogQueryMode
  workspaceId?: number
  namespace?: string
  application?: string
  resourceName?: string
  hosts?: string[]
}

export interface LogLabelKeysResponse {
  keys: string[]
}

export interface LogLabelValuesRequest extends LogLabelKeysRequest {
  labelKey: string
}

export interface LogLabelValuesResponse {
  values: string[]
}

export interface LogQueryModeOptionsResponse {
  loki: boolean
  elasticsearch: boolean
  rawQuery: boolean
  export: boolean
}

export interface LogClusterConfigItem {
  id: number
  clusterUuid: string
  appName: string
  appCode: string
  appType: number
  isDefault: boolean
  appUrl: string
  port: number
  protocol: string
  authEnabled: boolean
  authType: string
  tlsEnabled: boolean
  insecureSkipVerify: boolean
  status: number
  createdAt: number
  updatedAt: number
}

export interface LogClusterConfigResponse {
  clusterUuid: string
  defaultBackend: string
  backends: LogClusterConfigItem[]
}

export interface LogClusterSyncStatusResponse {
  clusterUuid: string
  totalRules: number
  enabledRules: number
  successRules: number
  failedRules: number
  pendingRules: number
  disabledRules: number
  healthStatus: string
  lastSyncError: string
  lastSyncAt: number
}

export interface LogAlertRule {
  id: number
  clusterUuid: string
  backendType: string
  projectUuid: string
  workspaceId: number
  namespace: string
  application: string
  resourceName: string
  name: string
  description: string
  queryText: string
  conditionType: LogAlertConditionType | string
  threshold: string
  window: string
  severity: LogAlertSeverity | string
  notifyChannels: string
  enabled: boolean
  backendRuleId: string
  lastSyncStatus: string
  lastSyncError: string
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
  evalInterval: string
  forDuration: string
  silenceWindow: string
}

export interface ListLogAlertRulesRequest {
  clusterUuid?: string
  workspaceId?: number
  projectUuid?: string
  namespace?: string
  application?: string
  resourceName?: string
  enabled?: boolean
  page?: number
  pageSize?: number
}

export interface ListLogAlertRulesResponse {
  data: LogAlertRule[]
  total: number
}

export interface CreateLogAlertRuleRequest {
  clusterUuid?: string
  workspaceId?: number
  projectUuid: string
  namespace: string
  application?: string
  resourceName?: string
  name: string
  description?: string
  queryText: string
  conditionType: LogAlertConditionType | string
  threshold: string
  window: string
  severity: LogAlertSeverity | string
  notifyChannels?: string
  enabled: boolean
  evalInterval?: string
  forDuration?: string
  silenceWindow?: string
}

export interface UpdateLogAlertRuleRequest extends CreateLogAlertRuleRequest {
  id: number
}

export interface LogAlertEngineStatsResponse {
  clusterUuid: string
  totalRules: number
  enabledRules: number
  disabledRules: number
  successRules: number
  failedRules: number
  pendingRules: number
  activeFiringEvents: number
  failedNotifyEvents: number
  deadNotifyEvents: number
  criticalRules: number
  warningRules: number
  infoRules: number
  hotRules: number
  warmRules: number
  coldRules: number
}

export interface ManualRetryLogAlertEventResponse {
  id: number
  notifyStatus: string
  retryCount: number
  notifyError: string
  nextRetryAt: number
}

export interface CloseLogAlertEventRequest {
  reason: string
}

export interface UpdateLogAlertEventReasonRequest {
  reason: string
}

export interface LogAlertEvent {
  id: number
  ruleId: number
  ruleName: string
  backendType: string
  clusterUuid: string
  clusterName: string
  projectUuid: string
  projectName: string
  workspaceId: number
  workspaceName: string
  namespace: string
  application: string
  resourceName: string
  podName: string
  containerName: string
  host: string
  eventFingerprint: string
  hitCount: number
  severity: string
  eventStatus: string
  notifyStatus: string
  notifyError: string
  retryCount: number
  nextRetryAt: number
  sampleTimestamp: number
  sampleMessage: string
  sampleRaw: string
  sampleLabels: Record<string, string>
  closeReason: string
  closedBy: string
  closedAt: number
  manualReason: string
  manualReasonBy: string
  manualReasonAt: number
  createdAt: number
  updatedAt: number
}

export interface SearchLogAlertEventsRequest {
  clusterUuid?: string
  ruleId?: number
  eventStatus?: string
  notifyStatus?: string
  page?: number
  pageSize?: number
}

export interface SearchLogAlertEventsResponse {
  data: LogAlertEvent[]
  total: number
}

/**
 * 查询日志
 */
export async function queryLogsApi(params: LogQueryRequest) {
  const resp = await request.post<LogQueryResponse>({
    url: `${LOG_BASE_PATH}/query`,
    data: params
  })
  return resp
}

/**
 * 下载日志（返回 Blob，用于触发浏览器下载）
 */
export async function downloadLogsApi(params: LogDownloadRequest) {
  const resp = await request.post<Blob>({
    url: `${LOG_BASE_PATH}/download`,
    data: params,
    responseType: 'blob',
    showErrorMessage: true
  })
  return resp
}

/**
 * 查询日志 labels key 列表
 */
export async function listLogLabelKeysApi(params: LogLabelKeysRequest) {
  const resp = await request.post<LogLabelKeysResponse>({
    url: `${LOG_BASE_PATH}/labels/keys`,
    data: params
  })
  return resp
}

/**
 * 查询日志 labels value 列表
 */
export async function listLogLabelValuesApi(params: LogLabelValuesRequest) {
  const resp = await request.post<LogLabelValuesResponse>({
    url: `${LOG_BASE_PATH}/labels/values`,
    data: params
  })
  return resp
}

export async function getLogQueryModeOptionsApi(clusterUuid: string, namespace?: string) {
  return request.get<LogQueryModeOptionsResponse>({
    url: `${LOG_BASE_PATH}/query-mode/options`,
    params: { clusterUuid, namespace }
  })
}

export async function getLogClusterConfigApi(clusterUuid: string) {
  return request.get<LogClusterConfigResponse>({
    url: `${LOG_BASE_PATH}/cluster-config`,
    params: { clusterUuid }
  })
}

export async function getLogClusterSyncStatusApi(clusterUuid: string) {
  return request.get<LogClusterSyncStatusResponse>({
    url: `${LOG_BASE_PATH}/cluster-sync-status`,
    params: { clusterUuid }
  })
}

export async function listLogAlertRulesApi(params: ListLogAlertRulesRequest) {
  return request.get<ListLogAlertRulesResponse>({
    url: `${LOG_BASE_PATH}/alerts`,
    params
  })
}

export async function getLogAlertRuleApi(id: number) {
  return request.get<LogAlertRule>({
    url: `${LOG_BASE_PATH}/alerts/${id}`
  })
}

export async function createLogAlertRuleApi(data: CreateLogAlertRuleRequest) {
  return request.post<LogAlertRule>({
    url: `${LOG_BASE_PATH}/alerts`,
    data
  })
}

export async function updateLogAlertRuleApi(
  id: number,
  data: Omit<UpdateLogAlertRuleRequest, 'id'>
) {
  return request.put<LogAlertRule>({
    url: `${LOG_BASE_PATH}/alerts/${id}`,
    data
  })
}

export async function enableLogAlertRuleApi(id: number) {
  return request.post<string>({
    url: `${LOG_BASE_PATH}/alerts/${id}/enable`
  })
}

export async function disableLogAlertRuleApi(id: number) {
  return request.post<string>({
    url: `${LOG_BASE_PATH}/alerts/${id}/disable`
  })
}

export async function deleteLogAlertRuleApi(id: number) {
  return request.del<string>({
    url: `${LOG_BASE_PATH}/alerts/${id}`
  })
}

export async function getLogAlertEngineStatsApi(clusterUuid?: string) {
  return request.get<LogAlertEngineStatsResponse>({
    url: `${LOG_BASE_PATH}/alert-engine/stats`,
    params: { clusterUuid }
  })
}

export async function manualRetryLogAlertEventApi(id: number) {
  return request.post<ManualRetryLogAlertEventResponse>({
    url: `${LOG_BASE_PATH}/events/${id}/retry`
  })
}

export async function closeLogAlertEventApi(id: number, data: CloseLogAlertEventRequest) {
  return request.post<LogAlertEvent>({
    url: `${LOG_BASE_PATH}/events/${id}/close`,
    data
  })
}

export async function updateLogAlertEventReasonApi(
  id: number,
  data: UpdateLogAlertEventReasonRequest
) {
  return request.post<LogAlertEvent>({
    url: `${LOG_BASE_PATH}/events/${id}/reason`,
    data
  })
}

export async function deleteLogAlertEventApi(id: number) {
  return request.del<string>({
    url: `${LOG_BASE_PATH}/events/${id}`
  })
}

export async function searchLogAlertEventsApi(params: SearchLogAlertEventsRequest) {
  return request.get<SearchLogAlertEventsResponse>({
    url: `${LOG_BASE_PATH}/events`,
    params
  })
}
