import request from '@/utils/http'
import { useUserStore } from '@/store/modules/user'

const INSPECTION_BASE_PATH = '/manager/v1/inspection'

export type InspectionScopeType = 'global' | 'cluster' | string
export type InspectionScheduleType = 'manual' | 'cron' | string
export type InspectionCheckType = 'builtin' | 'promql' | 'sql' | string
export type InspectionSeverity = 'critical' | 'warning' | 'info' | string
export type InspectionStatus = 'running' | 'success' | 'failed' | 'partial' | string
export type InspectionHealthLevel = 'healthy' | 'warning' | 'critical' | 'unknown' | string

export interface InspectionTemplate {
  id: number
  name: string
  code: string
  description: string
  scopeType: InspectionScopeType
  enabled: boolean
  isBuiltin: boolean
  version: number
  configJson: string
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface InspectionTemplatePayload {
  name: string
  code: string
  description?: string
  scopeType?: InspectionScopeType
  enabled?: boolean
  configJson?: string
}

export interface SearchInspectionTemplateRequest {
  name?: string
  scopeType?: InspectionScopeType
  enabled?: boolean
  page?: number
  pageSize?: number
  orderField?: string
  isAsc?: boolean
}

export interface SearchInspectionTemplateResponse {
  data: InspectionTemplate[]
  total: number
}

export interface InspectionGroup {
  id: number
  templateId: number
  groupCode: string
  groupName: string
  description: string
  enabled: boolean
  orderNum: number
  configJson: string
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface InspectionGroupPayload {
  templateId?: number
  groupCode: string
  groupName: string
  description?: string
  enabled?: boolean
  orderNum?: number
  configJson?: string
}

export interface InspectionGroupTransferPayload {
  targetTemplateId: number
  groupCode: string
  groupName: string
  itemCodePrefix?: string
  operation: 'copy' | 'move'
}

export interface InspectionGroupTransferResponse {
  group: InspectionGroup
  itemCount: number
}

export interface SearchInspectionGroupRequest {
  templateId?: number
  groupName?: string
  enabled?: boolean
  page?: number
  pageSize?: number
  orderField?: string
  isAsc?: boolean
}

export interface SearchInspectionGroupResponse {
  data: InspectionGroup[]
  total: number
}

export interface InspectionItem {
  id: number
  templateId: number
  groupId: number
  itemCode: string
  itemName: string
  category: string
  checkType: InspectionCheckType
  targetType: string
  severity: InspectionSeverity
  weight: number
  enabled: boolean
  timeoutSec: number
  orderNum: number
  promql: string
  operator: string
  threshold: string
  unit: string
  configJson: string
  advice: string
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface InspectionItemPayload {
  templateId?: number
  groupId?: number
  itemCode: string
  itemName: string
  category?: string
  checkType?: InspectionCheckType
  targetType?: string
  severity?: InspectionSeverity
  weight?: number
  enabled?: boolean
  timeoutSec?: number
  orderNum?: number
  promql?: string
  operator?: string
  threshold?: string
  unit?: string
  configJson?: string
  advice?: string
}

export interface SearchInspectionItemRequest {
  templateId?: number
  groupId?: number
  category?: string
  checkType?: InspectionCheckType
  enabled?: boolean
  page?: number
  pageSize?: number
  orderField?: string
  isAsc?: boolean
}

export interface SearchInspectionItemResponse {
  data: InspectionItem[]
  total: number
}

export interface InspectionTask {
  id: number
  name: string
  description: string
  templateId: number
  scopeType: InspectionScopeType
  clusterUuid: string
  scheduleType: InspectionScheduleType
  cronExpr: string
  enabled: boolean
  maxConcurrency: number
  timeoutSec: number
  lastRunAt: number
  nextRunAt: number
  lastRecordId: number
  lastStatus: InspectionStatus
  lastError: string
  configJson: string
  prometheusEnabled: boolean
  prometheusEndpoint: string
  prometheusAuthEnabled: boolean
  prometheusAuthType: string
  prometheusUsername: string
  prometheusPassword: string
  prometheusToken: string
  prometheusTlsEnabled: boolean
  prometheusInsecureSkipVerify: boolean
  prometheusCaCert: string
  prometheusClientCert: string
  prometheusClientKey: string
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface InspectionTaskPayload {
  name: string
  description?: string
  templateId?: number
  scopeType?: InspectionScopeType
  clusterUuid?: string
  scheduleType?: InspectionScheduleType
  cronExpr?: string
  enabled?: boolean
  maxConcurrency?: number
  timeoutSec?: number
  configJson?: string
  prometheusEnabled?: boolean
  prometheusEndpoint?: string
  prometheusAuthEnabled?: boolean
  prometheusAuthType?: string
  prometheusUsername?: string
  prometheusPassword?: string
  prometheusToken?: string
  prometheusTlsEnabled?: boolean
  prometheusInsecureSkipVerify?: boolean
  prometheusCaCert?: string
  prometheusClientCert?: string
  prometheusClientKey?: string
}

export interface SearchInspectionTaskRequest {
  name?: string
  scopeType?: InspectionScopeType
  clusterUuid?: string
  scheduleType?: InspectionScheduleType
  enabled?: boolean
  page?: number
  pageSize?: number
  orderField?: string
  isAsc?: boolean
}

export interface SearchInspectionTaskResponse {
  data: InspectionTask[]
  total: number
}

export interface RunInspectionTaskRequest {
  clusterUuid?: string
}

export interface RunInspectionTaskResponse {
  records: InspectionRecord[]
}

export interface InspectionRunLog {
  time: number
  level: 'info' | 'warning' | 'error' | string
  message: string
}

export interface InspectionRecord {
  id: number
  recordNo: string
  taskId: number
  templateId: number
  triggerType: string
  scopeType: InspectionScopeType
  clusterUuid: string
  clusterName: string
  status: InspectionStatus
  score: number
  healthLevel: InspectionHealthLevel
  totalCount: number
  successCount: number
  warningCount: number
  criticalCount: number
  failedCount: number
  summaryJson: string
  reportJson: string
  startedAt: number
  finishedAt: number
  durationMs: number
  errorMessage: string
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface SearchInspectionRecordRequest {
  taskId?: number
  clusterUuid?: string
  status?: InspectionStatus
  healthLevel?: InspectionHealthLevel
  startTime?: number
  endTime?: number
  page?: number
  pageSize?: number
  orderField?: string
  isAsc?: boolean
}

export interface SearchInspectionRecordResponse {
  data: InspectionRecord[]
  total: number
}

export interface InspectionResult {
  id: number
  recordId: number
  itemId: number
  itemCode: string
  itemName: string
  category: string
  targetType: string
  targetName: string
  severity: InspectionSeverity
  status: string
  score: number
  expected: string
  actual: string
  value: string
  unit: string
  message: string
  suggestion: string
  detailJson: string
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface InspectionReportResponse {
  record: InspectionRecord
  results: InspectionResult[]
  resultsTotal: number
}

export interface InspectionReportRequest {
  page?: number
  pageSize?: number
  keyword?: string
  status?: string
  severity?: string
}

export interface InspectionRecordFinishPayload {
  message?: string
}

export interface InspectionRecordFinishResponse {
  record: InspectionRecord
}

export interface InspectionCategoryStat {
  category: string
  totalCount: number
  successCount: number
  warningCount: number
  criticalCount: number
  failedCount: number
}

export interface InspectionDashboardRequest {
  clusterUuid?: string
  days?: number
}

export interface InspectionDashboardResponse {
  totalRecords: number
  successRecords: number
  failedRecords: number
  criticalRecords: number
  avgScore: number
  latestHealthLevel: InspectionHealthLevel
  latestRecordId: number
  categoryStats: InspectionCategoryStat[]
}

export function searchInspectionTemplatesApi(params: SearchInspectionTemplateRequest) {
  return request.get<SearchInspectionTemplateResponse>({
    url: `${INSPECTION_BASE_PATH}/templates`,
    params
  })
}

export function createInspectionTemplateApi(data: InspectionTemplatePayload) {
  return request.post<InspectionTemplate>({
    url: `${INSPECTION_BASE_PATH}/templates`,
    data
  })
}

export function updateInspectionTemplateApi(id: number, data: InspectionTemplatePayload) {
  return request.put<InspectionTemplate>({
    url: `${INSPECTION_BASE_PATH}/templates/${id}`,
    data
  })
}

export function deleteInspectionTemplateApi(id: number) {
  return request.del<string>({
    url: `${INSPECTION_BASE_PATH}/templates/${id}`
  })
}

export function searchInspectionGroupsApi(params: SearchInspectionGroupRequest) {
  return request.get<SearchInspectionGroupResponse>({
    url: `${INSPECTION_BASE_PATH}/groups`,
    params
  })
}

export function createInspectionGroupApi(data: InspectionGroupPayload) {
  return request.post<InspectionGroup>({
    url: `${INSPECTION_BASE_PATH}/groups`,
    data
  })
}

export function updateInspectionGroupApi(id: number, data: InspectionGroupPayload) {
  return request.put<InspectionGroup>({
    url: `${INSPECTION_BASE_PATH}/groups/${id}`,
    data
  })
}

export function deleteInspectionGroupApi(id: number) {
  return request.del<string>({
    url: `${INSPECTION_BASE_PATH}/groups/${id}`
  })
}

export function transferInspectionGroupApi(id: number, data: InspectionGroupTransferPayload) {
  return request.post<InspectionGroupTransferResponse>({
    url: `${INSPECTION_BASE_PATH}/groups/${id}/transfer`,
    data
  })
}

export function searchInspectionItemsApi(params: SearchInspectionItemRequest) {
  return request.get<SearchInspectionItemResponse>({
    url: `${INSPECTION_BASE_PATH}/items`,
    params
  })
}

export function createInspectionItemApi(data: InspectionItemPayload) {
  return request.post<InspectionItem>({
    url: `${INSPECTION_BASE_PATH}/items`,
    data
  })
}

export function updateInspectionItemApi(id: number, data: InspectionItemPayload) {
  return request.put<InspectionItem>({
    url: `${INSPECTION_BASE_PATH}/items/${id}`,
    data
  })
}

export function deleteInspectionItemApi(id: number) {
  return request.del<string>({
    url: `${INSPECTION_BASE_PATH}/items/${id}`
  })
}

export function searchInspectionTasksApi(params: SearchInspectionTaskRequest) {
  return request.get<SearchInspectionTaskResponse>({
    url: `${INSPECTION_BASE_PATH}/tasks`,
    params
  })
}

export function createInspectionTaskApi(data: InspectionTaskPayload) {
  return request.post<InspectionTask>({
    url: `${INSPECTION_BASE_PATH}/tasks`,
    data
  })
}

export function updateInspectionTaskApi(id: number, data: InspectionTaskPayload) {
  return request.put<InspectionTask>({
    url: `${INSPECTION_BASE_PATH}/tasks/${id}`,
    data
  })
}

export function deleteInspectionTaskApi(id: number) {
  return request.del<string>({
    url: `${INSPECTION_BASE_PATH}/tasks/${id}`
  })
}

export function runInspectionTaskApi(id: number, data?: RunInspectionTaskRequest) {
  return request.post<RunInspectionTaskResponse>({
    url: `${INSPECTION_BASE_PATH}/tasks/${id}/run`,
    data: data || {}
  })
}

export function getInspectionRecordLogStreamUrl(recordId: number) {
  const params = new URLSearchParams()
  const { accessToken } = useUserStore()
  if (accessToken) {
    params.append('token', accessToken)
  }
  const query = params.toString()
  const baseUrl = String(import.meta.env.VITE_API_URL || '').replace(/\/$/, '')
  return `${baseUrl}${INSPECTION_BASE_PATH}/records/${recordId}/logs/stream${query ? `?${query}` : ''}`
}

export function searchInspectionRecordsApi(params: SearchInspectionRecordRequest) {
  return request.get<SearchInspectionRecordResponse>({
    url: `${INSPECTION_BASE_PATH}/records`,
    params
  })
}

export function getInspectionReportApi(recordId: number, params?: InspectionReportRequest) {
  return request.get<InspectionReportResponse>({
    url: `${INSPECTION_BASE_PATH}/reports/${recordId}`,
    params
  })
}

export function finishInspectionRecordApi(recordId: number, data?: InspectionRecordFinishPayload) {
  return request.post<InspectionRecordFinishResponse>({
    url: `${INSPECTION_BASE_PATH}/records/${recordId}/finish`,
    data: data || {}
  })
}

export function deleteInspectionRecordApi(recordId: number) {
  return request.del<string>({
    url: `${INSPECTION_BASE_PATH}/records/${recordId}`
  })
}

export function exportInspectionReportApi(recordId: number) {
  return request.get<Blob>({
    url: `${INSPECTION_BASE_PATH}/reports/${recordId}/export`,
    responseType: 'blob',
    showErrorMessage: true
  })
}

export function getInspectionDashboardApi(params: InspectionDashboardRequest) {
  return request.get<InspectionDashboardResponse>({
    url: `${INSPECTION_BASE_PATH}/dashboard`,
    params
  })
}
