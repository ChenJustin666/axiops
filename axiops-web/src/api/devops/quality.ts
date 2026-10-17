import request from '@/utils/http'

const DEVOPS_QUALITY_BASE = '/devops/v1/quality'

export interface QualityScanPlanItem {
  stepId?: string
  stageId?: string
  tool: string
  toolMode: string
  toolBindingId?: string
  targetType?: string
  targetName?: string
  targetParamCode?: string
  reportSource?: string
  reportPath?: string
  reportFormat: string
  required?: boolean
  enforce?: boolean
  parser?: string
}

export interface QualityScanReport {
  id: string
  projectId: string
  projectName: string
  projectCode: string
  systemId: string
  systemName: string
  systemCode: string
  environmentId: string
  environmentName: string
  environmentCode: string
  pipelineId: string
  runId: string
  stageId: string
  stepId: string
  tool: string
  toolMode: string
  toolBindingId: string
  targetType: string
  targetName: string
  imageRef: string
  reportSource: string
  reportFormat: string
  reportPath: string
  parser: string
  bucket: string
  objectKey: string
  size: number
  contentType: string
  sha256: string
  status: string
  rejectReason: string
  gateStatus: string
  summaryJson: string
  batchId: string
  entryId: string
  sourceType: string
  archiveEntryPath: string
  originalFileName: string
  current: boolean
  supersededBy: string
  createdAt: string
}

export interface QualityReportUploadEntryResult {
  id: string
  entryPath: string
  status: string
  reason: string
  reportId: string
  objectKey: string
  sha256: string
  size: number
}

export interface QualityReportUploadResponse {
  batchId: string
  reportIds: string[]
  acceptedCount: number
  rejectedCount: number
  ignoredCount: number
  entries: QualityReportUploadEntryResult[]
}

export interface QualityReportUploadBatch {
  id: string
  projectId: string
  projectName: string
  projectCode: string
  systemId: string
  systemName: string
  systemCode: string
  environmentId: string
  environmentName: string
  environmentCode: string
  pipelineId: string
  pipelineName: string
  runId: string
  stageId: string
  stepId: string
  tool: string
  reportFormat: string
  parser: string
  sourceType: string
  originalFileName: string
  bucket: string
  objectKey: string
  size: number
  contentType: string
  sha256: string
  status: string
  acceptedCount: number
  rejectedCount: number
  ignoredCount: number
  message: string
  createdAt: string
}

export interface QualityReportUploadEntry {
  id: string
  batchId: string
  runId: string
  stageId: string
  stepId: string
  tool: string
  reportFormat: string
  parser: string
  entryPath: string
  originalFileName: string
  bucket: string
  objectKey: string
  size: number
  contentType: string
  sha256: string
  status: string
  reason: string
  reportId: string
  createdAt: string
}

export interface QualityScanIssue {
  id: string
  reportId: string
  projectId: string
  systemId: string
  runId: string
  tool: string
  targetType: string
  issueType: string
  severity: string
  title: string
  description: string
  filePath: string
  line: number
  component: string
  packageName: string
  imageRef: string
  imageDigest: string
  cveId: string
  installedVersion: string
  fixedVersion: string
  ruleId: string
  status: string
  rawJson: string
  createdAt: string
}

export interface ListQualityScanReportParams {
  page?: number
  pageSize?: number
  projectId?: string
  systemId?: string
  pipelineId?: string
  runId?: string
  tool?: string
  targetType?: string
  status?: string
  gateStatus?: string
}

export interface ListQualityScanIssueParams {
  page?: number
  pageSize?: number
  reportId?: string
  projectId?: string
  systemId?: string
  runId?: string
  tool?: string
  targetType?: string
  issueType?: string
  severity?: string
  status?: string
}

export interface ListQualityReportUploadBatchParams {
  page?: number
  pageSize?: number
  projectId?: string
  systemId?: string
  pipelineId?: string
  runId?: string
  tool?: string
  status?: string
}

export interface ListQualityReportUploadEntryParams {
  page?: number
  pageSize?: number
  batchId: string
  runId?: string
  status?: string
}

export interface QualityDashboardOverview {
  reportCount: number
  passedCount: number
  failedCount: number
  rejectedCount: number
  issueCount: number
  criticalCount: number
  highCount: number
  gatePassRate: number
}

export interface QualityDashboardMetric {
  id: string
  name: string
  code: string
  reportCount: number
  issueCount: number
  criticalCount: number
  highCount: number
  gatePassRate: number
}

export interface QualityDashboardResponse {
  overview: QualityDashboardOverview
  projectRanks: QualityDashboardMetric[]
  systemRanks: QualityDashboardMetric[]
  toolRanks: QualityDashboardMetric[]
}

export interface QualityRunTypeSummary {
  targetType: string
  reportCount: number
  issueCount: number
  criticalCount: number
  highCount: number
  mediumCount: number
  lowCount: number
}

export interface QualityImageSummary {
  imageRef: string
  imageDigest: string
  reportCount: number
  issueCount: number
  criticalCount: number
  highCount: number
  fixableCount: number
}

export interface QualityRunSummaryResponse {
  runId: string
  gateStatus: string
  overview: QualityDashboardOverview
  staticCode: QualityRunTypeSummary
  image: QualityRunTypeSummary
  scanPlan: QualityScanPlanItem[]
  reports: QualityScanReport[]
  staticCodeIssues: QualityScanIssue[]
  imageIssues: QualityScanIssue[]
  missingReports: Array<{
    tool: string
    stepId: string
    reportFormat: string
    reportPath: string
  }>
  rejectedReports: QualityScanReport[]
  rejectedEntries: QualityReportUploadEntry[]
  images: QualityImageSummary[]
}

export function listQualityScanReportApi(params: ListQualityScanReportParams = {}) {
  return request.get<{ items: QualityScanReport[]; total: number }>({
    url: `${DEVOPS_QUALITY_BASE}/report/list`,
    params
  })
}

export function getQualityScanReportApi(id: string) {
  return request.get<QualityScanReport>({ url: `${DEVOPS_QUALITY_BASE}/report/${id}` })
}

export function uploadQualityScanReportApi(data: FormData) {
  return request.post<QualityReportUploadResponse>({
    url: `${DEVOPS_QUALITY_BASE}/report/upload`,
    data
  })
}

export function downloadQualityScanReportApi(id: string) {
  return request.get<Blob>({
    url: `${DEVOPS_QUALITY_BASE}/report/${id}/download`,
    responseType: 'blob'
  })
}

export function previewQualityScanReportApi(id: string) {
  return request.get<Blob>({
    url: `${DEVOPS_QUALITY_BASE}/report/${id}/preview`,
    responseType: 'blob'
  })
}

export function listQualityScanIssueApi(params: ListQualityScanIssueParams = {}) {
  return request.get<{ items: QualityScanIssue[]; total: number }>({
    url: `${DEVOPS_QUALITY_BASE}/issue/list`,
    params
  })
}

export function evaluateQualityGateApi(runId: string) {
  return request.post<{ runId: string; gateStatus: string; message: string }>({
    url: `${DEVOPS_QUALITY_BASE}/run/${runId}/gate/evaluate`
  })
}

export function getQualityDashboardApi(
  params: {
    projectId?: string
    systemId?: string
    targetType?: string
    startDate?: string
    endDate?: string
  } = {}
) {
  return request.get<QualityDashboardResponse>({
    url: `${DEVOPS_QUALITY_BASE}/dashboard`,
    params
  })
}

export function getQualityRunSummaryApi(runId: string) {
  return request.get<QualityRunSummaryResponse>({
    url: `${DEVOPS_QUALITY_BASE}/run/${runId}/summary`
  })
}

export function getQualityScanReportDownloadUrlApi(id: string) {
  return request.get<{ url: string }>({
    url: `${DEVOPS_QUALITY_BASE}/report/${id}/download-url`
  })
}

export function listQualityReportUploadBatchApi(params: ListQualityReportUploadBatchParams = {}) {
  return request.get<{ items: QualityReportUploadBatch[]; total: number }>({
    url: `${DEVOPS_QUALITY_BASE}/report/upload-batch/list`,
    params
  })
}

export function listQualityReportUploadEntryApi(params: ListQualityReportUploadEntryParams) {
  return request.get<{ items: QualityReportUploadEntry[]; total: number }>({
    url: `${DEVOPS_QUALITY_BASE}/report/upload-entry/list`,
    params
  })
}

export function downloadQualityReportUploadBatchApi(id: string) {
  return request.get<Blob>({
    url: `${DEVOPS_QUALITY_BASE}/report/upload-batch/${id}/download`,
    responseType: 'blob'
  })
}

export function downloadQualityReportUploadEntryApi(id: string) {
  return request.get<Blob>({
    url: `${DEVOPS_QUALITY_BASE}/report/upload-entry/${id}/download`,
    responseType: 'blob'
  })
}

export function previewQualityReportUploadEntryApi(id: string) {
  return request.get<Blob>({
    url: `${DEVOPS_QUALITY_BASE}/report/upload-entry/${id}/preview`,
    responseType: 'blob'
  })
}
