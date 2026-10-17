import request from '@/utils/http'
import { useUserStore } from '@/store/modules/user'

const DEVOPS_PIPELINE_BASE = '/devops/v1/pipeline'
const DEVOPS_PROJECT_BASE = '/devops/v1/project'

export interface StepParamOption {
  label: string
  value: string
}

export interface CompoundParamField {
  name: string
  code: string
  defaultValue?: string
  paramType: string
  mode: string
  required?: boolean
  readonly?: boolean
  description?: string
  selectList?: StepParamOption[]
  allowCustom?: boolean
  runtimeMode?: string
  runtimeConfig?: boolean
  config?: StepParamConfig
}

export interface StepParamConfig {
  options?: StepParamOption[]
  channelGroupCode?: string
  channelTypeFilter?: string
  channelParamCode?: string
  channelBindingId?: string
  mappingField?: string
  voucherModel?: string
  credentialId?: string
  credentialMode?: 'field' | 'jsonData' | 'jsonBase64' | 'jenkinsCredentialId'
  credentialSourceParamCode?: string
  configTypeId?: string
  configTypeCode?: string
  valueMode?: 'string' | 'text' | 'credentialsSecretFile' | 'base64' | 'json' | 'jsonBase64'
  projectParamCode?: string
  componentParamCode?: string
  provider?: string
  validateRemote?: boolean
  compoundFields?: CompoundParamField[]
  fullRow?: boolean
  renderMode?: 'normal' | 'inventory'
  dependencyParamCodes?: StepParamDependency[]
}

export interface StepParamDependency {
  field: string
  paramCode: string
}

export interface StepArtifactConfig {
  enabled?: boolean
  type?: string
  name?: string
  path?: string
  required?: boolean
  contentType?: string
}

export type TektonParamType = 'string' | 'array' | 'object'

export interface TektonPropertySpec {
  type?: TektonParamType
}

export interface DevopsTektonTaskParam {
  name: string
  type?: TektonParamType
  defaultValue?: string
  description?: string
  enum?: string[]
  properties?: Record<string, TektonPropertySpec>
  required?: boolean
}

export interface DevopsTektonTaskResult {
  name: string
  type?: TektonParamType
  description?: string
  value?: string
  properties?: Record<string, TektonPropertySpec>
}

export interface DevopsTektonWorkspace {
  name: string
  description?: string
  optional?: boolean
  readOnly?: boolean
  mountPath?: string
}

export interface DevopsStepParam {
  name: string
  code: string
  defaultValue?: string
  paramType: string
  mode: string
  required?: boolean
  readonly?: boolean
  description?: string
  selectList?: StepParamOption[]
  allowCustom?: boolean
  channelGroupCode?: string
  mappingField?: string
  voucherModel?: string
  sortOrder?: number
  runtimeMode?: string
  runtimeConfig?: boolean
  config?: StepParamConfig
}

export interface DevopsStepCategory {
  id: string
  name: string
  code: string
  description: string
  icon: string
  iconColor: string
  sortOrder: number
  status: number
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface DevopsStepCategoryForm {
  name: string
  code?: string
  description?: string
  icon?: string
  iconColor?: string
  sortOrder: number
  status: number
}

export interface DevopsStepTemplate {
  id: string
  name: string
  code: string
  icon: string
  iconColor: string
  description: string
  type: string
  categoryId: string
  categoryName: string
  engineType: string
  engineChannelGroupCode: string
  engineChannelType: string
  stageContent: string
  params: DevopsStepParam[]
  taskParams?: DevopsTektonTaskParam[]
  taskResults?: DevopsTektonTaskResult[]
  taskWorkspaces?: DevopsTektonWorkspace[]
  artifactConfig?: StepArtifactConfig
  status: number
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
  lastRunId?: string
  lastRunBuildNumber?: number
  lastRunBuildUrl?: string
  lastRunStartedAt?: string
}

export interface DevopsStepTemplateForm {
  name: string
  code?: string
  icon?: string
  iconColor?: string
  description?: string
  type: string
  categoryId: string
  stageContent: string
  params: DevopsStepParam[]
  taskParams?: DevopsTektonTaskParam[]
  taskResults?: DevopsTektonTaskResult[]
  taskWorkspaces?: DevopsTektonWorkspace[]
  artifactConfig?: StepArtifactConfig
  status: number
}

export interface PipelineTemplateStep {
  id?: string
  stepId: string
  stepCode?: string
  stepName?: string
  nodeName?: string
  containerName?: string
  stepType?: string
  icon?: string
  iconColor?: string
  parentNodeId?: string
  branchType?: 'next' | 'parallel'
  sortOrder?: number
  enabled?: boolean
  x?: number
  y?: number
  paramValues?: string
  artifactConfig?: StepArtifactConfig
}

export interface DevopsPipelineTemplate {
  id: string
  name: string
  code: string
  icon: string
  iconColor: string
  description: string
  scope: string
  projectId: string
  engineType: string
  steps: PipelineTemplateStep[]
  tektonDagConfig?: string
  tektonRunPolicy?: string
  tektonTriggerConfig?: string
  tektonPrunerPolicyRef?: string
  tektonPipelineYamlSnapshot?: string
  status: number
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface DevopsPipelineTemplateForm {
  name: string
  code?: string
  icon?: string
  iconColor?: string
  description?: string
  scope: string
  projectId?: string
  engineType: string
  steps: PipelineTemplateStep[]
  tektonDagConfig?: string
  tektonRunPolicy?: string
  tektonTriggerConfig?: string
  tektonPrunerPolicyRef?: string
  tektonPipelineYamlSnapshot?: string
  status: number
}

export interface ListDevopsStepCategoryParams {
  page?: number
  pageSize?: number
  name?: string
  code?: string
  status?: number
}

export interface ListDevopsStepTemplateParams {
  page?: number
  pageSize?: number
  name?: string
  code?: string
  categoryId?: string
  engineType?: string
  type?: string
  status?: number
}

export interface ListDevopsPipelineTemplateParams {
  page?: number
  pageSize?: number
  name?: string
  code?: string
  scope?: string
  projectId?: string
  engineType?: string
  status?: number
}

export interface ListDevopsStepCategoryResponse {
  items: DevopsStepCategory[]
  total: number
}

export interface ListDevopsStepTemplateResponse {
  items: DevopsStepTemplate[]
  total: number
}

export type TektonStepImageType = 'image' | 'param'

export interface TektonStepImageItem {
  id: string
  name: string
  code: string
  image: string
  imageType: TektonStepImageType
  paramName?: string
}

export interface ListTektonStepImageParams {
  page?: number
  pageSize?: number
  name?: string
  code?: string
  categoryId?: string
  status?: number
}

export interface UpdateTektonStepImageItem {
  id: string
  image?: string
  newImage: string
  imageType: TektonStepImageType
  paramName?: string
}

export interface BatchUpdateTektonStepImageResponse {
  total: number
  updated: number
}

export interface ListDevopsPipelineTemplateResponse {
  items: DevopsPipelineTemplate[]
  total: number
}

export interface CredentialTypeOption {
  credentialType: string
  name: string
  mappingFields: string[]
  defaultMappingField: string
}

export interface ChannelGroupOption {
  code: string
  name: string
}

export interface ChannelMappingFieldOption {
  groupCode?: string
  field: string
  name: string
  kind?: 'default' | 'dynamic' | 'address' | 'composite' | 'option' | 'output' | string
  uiControl?:
    | 'select'
    | 'input'
    | 'addressPicker'
    | 'cascadePicker'
    | 'deployConfig'
    | 'readonly'
    | 'radio'
    | 'output'
    | string
  provider?: string
  dependencies?: ChannelMappingFieldDependency[]
  allOf?: ChannelMappingFieldDependency[]
  anyOf?: ChannelMappingFieldDependency[]
  allowManualInput?: boolean
  addressBuilder?: string
  required?: boolean
  channelTypes?: string[]
  valueType?: string
  outputMode?: string
  outputTemplate?: string
  credentialPolicy?: string
  fallbackAddressFields?: string[]
  resolveEndpoint?: boolean
  options?: StepParamOption[]
  sortOrder?: number
}

export interface ChannelMappingFieldDependency {
  field: string
  source: 'channel' | 'param' | string
  name?: string
  required?: boolean
}

export interface PipelineTemplateVariable {
  stepId: string
  stepNodeId?: string
  stepCode: string
  stepName: string
  name: string
  code: string
  sourceCode?: string
  paramType: string
  mode: string
  defaultValue: string
  currentValue: string
  required: boolean
  readonly: boolean
  description: string
  selectList?: StepParamOption[]
  allowCustom?: boolean
  channelGroupCode?: string
  mappingField?: string
  voucherModel?: string
  sortOrder?: number
  runtimeMode?: string
  runtimeConfig?: boolean
  config?: StepParamConfig
}

export interface DevopsSystem {
  id: string
  projectId: string
  name: string
  code: string
  description: string
  ownerUserIds: number[]
  status: number
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface DevopsSystemForm {
  projectId: string
  name: string
  code?: string
  description?: string
  ownerUserIds?: number[]
  status: number
}

export interface ListDevopsSystemParams {
  page?: number
  pageSize?: number
  projectId?: string
  name?: string
  code?: string
  status?: number
}

export interface ListDevopsSystemResponse {
  items: DevopsSystem[]
  total: number
}

export interface DevopsPipelineEnvironment {
  id: string
  name: string
  code: string
  description: string
  icon: string
  iconColor: string
  sortOrder: number
  status: number
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface DevopsPipelineEnvironmentForm {
  name: string
  code?: string
  description?: string
  icon?: string
  iconColor?: string
  sortOrder: number
  status: number
}

export interface ListDevopsPipelineEnvironmentParams {
  page?: number
  pageSize?: number
  name?: string
  code?: string
  status?: number
}

export interface ListDevopsPipelineEnvironmentResponse {
  items: DevopsPipelineEnvironment[]
  total: number
}

export interface PipelineParam {
  name: string
  code: string
  sourceCode?: string
  stepNodeId?: string
  defaultValue?: string
  currentValue?: string
  paramType: string
  mode: string
  required?: boolean
  readonly?: boolean
  description?: string
  selectList?: StepParamOption[]
  allowCustom?: boolean
  channelGroupCode?: string
  mappingField?: string
  voucherModel?: string
  sortOrder?: number
  runtimeMode?: string
  runtimeConfig?: boolean
  config?: StepParamConfig
}

export interface JenkinsAgent {
  type: string
  label?: string
  dockerImage?: string
  dockerArgs?: string
  raw?: string
  id?: string
  name?: string
  agentType?: string
  matchMode?: string
  matchValue?: string
  cloud?: string
  podYaml?: string
  containers?: JenkinsAgentContainer[]
}

export interface JenkinsAgentContainer {
  name: string
  image?: string
}

export interface JenkinsAgentOption {
  id: string
  name: string
  displayName: string
  label: string
  offline: boolean
  temporarilyOffline: boolean
  labels: string[]
  agentType: string
  matchMode: string
  matchValue: string
  cloud: string
  podYaml: string
  containers: JenkinsAgentContainer[]
}

export interface ListJenkinsAgentParams {
  projectId?: string
  systemId?: string
  environmentId?: string
  buildChannelBindingId: string
}

export interface ListJenkinsAgentResponse {
  items: JenkinsAgentOption[]
}

export interface JenkinsTool {
  name: string
  type: string
  version?: string
}

export interface JenkinsOption {
  name: string
  value?: string
}

export interface JenkinsTrigger {
  type: string
  spec: string
  enabled: boolean
}

export interface PipelineScanPlanItem {
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

export interface DevopsPipeline {
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
  name: string
  code: string
  engineType: string
  templateId: string
  buildChannelBindingId: string
  buildChannelName: string
  steps: PipelineTemplateStep[]
  params: PipelineParam[]
  agent: JenkinsAgent
  tools: JenkinsTool[]
  options: JenkinsOption[]
  triggers: JenkinsTrigger[]
  jobName: string
  jobFullName: string
  tektonNamespace?: string
  tektonPipelineName?: string
  triggerMode: string
  syncStatus: string
  lastRunStatus: string
  status: number
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
  lastRunId?: string
  lastRunBuildNumber?: number
  lastRunBuildUrl?: string
  lastRunStartedAt?: string
  scanEnabled?: boolean
  scanMode?: string
  rejectUnexpectedReports?: boolean
  enforce?: boolean
  scanItems?: PipelineScanPlanItem[]
  tektonDagConfig?: string
  tektonRunPolicy?: string
  tektonTriggerConfig?: string
  tektonPrunerPolicyRef?: string
  tektonPipelineYamlSnapshot?: string
  tektonTemplateId?: string
  tektonTemplateSnapshot?: string
  tektonParamBindings?: string
  tektonWorkspaceBindings?: string
  tektonResourceBindings?: string
}

export interface DevopsPipelineForm {
  projectId: string
  systemId: string
  environmentId: string
  name: string
  code?: string
  engineType: string
  templateId?: string
  buildChannelBindingId: string
  steps: PipelineTemplateStep[]
  params: PipelineParam[]
  agent: JenkinsAgent
  tools: JenkinsTool[]
  options: JenkinsOption[]
  triggers: JenkinsTrigger[]
  triggerMode: string
  scanEnabled?: boolean
  scanMode?: string
  rejectUnexpectedReports?: boolean
  enforce?: boolean
  scanItems?: PipelineScanPlanItem[]
  tektonDagConfig?: string
  tektonRunPolicy?: string
  tektonTriggerConfig?: string
  tektonPrunerPolicyRef?: string
  tektonTemplateId?: string
  tektonTemplateSnapshot?: string
  tektonParamBindings?: string
  tektonWorkspaceBindings?: string
  tektonResourceBindings?: string
  status: number
}

export interface ListDevopsPipelineParams {
  page?: number
  pageSize?: number
  projectId?: string
  systemId?: string
  environmentId?: string
  name?: string
  code?: string
  engineType?: string
  syncStatus?: string
  lastRunStatus?: string
  triggerMode?: string
  status?: number
}

export interface ListDevopsPipelineResponse {
  items: DevopsPipeline[]
  total: number
}

export interface DevopsPipelineRun {
  id: string
  projectId: string
  pipelineId: string
  pipelineName: string
  pipelineCode: string
  projectCode: string
  projectName: string
  systemId: string
  systemCode: string
  systemName: string
  environmentId: string
  environmentCode: string
  environmentName: string
  engineType: string
  jenkinsJobName: string
  jenkinsJobFullName: string
  jenkinsBuildNumber: number
  jenkinsQueueId: string
  jenkinsBuildUrl?: string
  buildId?: number
  tektonNamespace: string
  tektonPipelineName: string
  tektonPipelineRunName: string
  triggerType: string
  triggerUserId: number
  triggerUsername: string
  status: string
  startedAt: string
  finishedAt: string
  durationSeconds: number
  errorMessage: string
  logStorageType: string
  logObjectKey: string
  logSize: number
  logArchivedAt: string
}

export interface DevopsPipelineRunYamlSnapshot {
  runId: string
  pipelineId: string
  engineType: string
  tektonNamespace: string
  tektonPipelineName: string
  tektonPipelineRunName: string
  tektonPipelineYamlSnapshot: string
  tektonPipelineRunYamlSnapshot: string
  pipelineSnapshot: string
}

export interface ListDevopsPipelineRunParams {
  page?: number
  pageSize?: number
  projectId?: string
  pipelineId?: string
  status?: string
}

export type TektonRuntimeResourceType =
  | 'task'
  | 'clusterTask'
  | 'serviceAccount'
  | 'storageClass'
  | 'configMap'

export interface TektonRuntimeResource {
  name: string
  namespace: string
  kind: string
  apiVersion: string
  description: string
  params: string[]
  workspaces: string[]
  results: string[]
  provisioner: string
  reclaimPolicy: string
  volumeBindingMode: string
  allowVolumeExpansion: boolean
  secrets: string[]
  imagePullSecrets: string[]
  createdAt: number
  yaml: string
  keys: string[]
}

export interface ListTektonRuntimeResourceParams {
  resourceType: TektonRuntimeResourceType
  projectId: string
  systemId?: string
  environmentId?: string
  buildChannelBindingId: string
  keyword?: string
}

export interface DevopsPipelineRunStage {
  id: string
  runId: string
  pipelineId: string
  nodeId: string
  stageName: string
  stageType: string
  status: string
  startedAt: string
  finishedAt: string
  durationSeconds: number
  jenkinsNodeId: string
  tektonTaskRunName: string
  tektonPodName: string
  containerName: string
  containerNames?: string[]
  tektonTaskRuns?: TektonTaskRunSnapshot[]
  logObjectKey: string
  errorMessage: string
}

export interface TektonTaskRunSnapshot {
  name: string
  pipelineTaskName: string
  podName: string
  containerName: string
  containerNames?: string[]
  status: string
  startedAt: string
  finishedAt: string
  durationSeconds: number
  errorMessage: string
}

export interface DevopsPipelineRunInputParam {
  name: string
  type: string
  description: string
  defaultValue: string
  options: StepParamOption[]
}

export interface DevopsPipelineRunInput {
  id: string
  name: string
  message: string
  params: DevopsPipelineRunInputParam[]
}

export interface DynamicParamOption {
  label: string
  value: string
  metadata?: string
}

export interface DynamicParamOptionMetadata {
  [key: string]: any
}

export interface DynamicSelectOption {
  label: string
  value: string
  metadata?: DynamicParamOptionMetadata
}

export interface ListDynamicParamOptionsParams {
  projectId?: string
  systemId?: string
  environmentId?: string
  paramType: string
  channelBindingId?: string
  projectValue?: string
  componentValue?: string
  configTypeId?: string
  configTypeCode?: string
  mappingField?: string
  dependencyValues?: string
  keyword?: string
  page?: number
  pageSize?: number
}

export interface DevopsPipelineRunArtifact {
  id: string
  runId: string
  stageId: string
  stepId: string
  name: string
  type: string
  bucket: string
  objectKey: string
  size: number
  contentType: string
  status: string
  errorMessage: string
  url: string
  createdAt: string
}

export interface PipelineDashboardOverview {
  buildCount: number
  successCount: number
  failureCount: number
  abortedCount: number
  totalDurationSeconds: number
  successRate: number
  failureRate: number
  avgDurationSeconds: number
  projectCount: number
  systemCount: number
  environmentCount: number
  pipelineCount: number
  channelCount: number
}

export interface PipelineDashboardMetric {
  id: string
  name: string
  code: string
  buildCount: number
  successCount: number
  failureCount: number
  abortedCount: number
  totalDurationSeconds: number
  successRate: number
  failureRate: number
  avgDurationSeconds: number
  lastRunAt: string
}

export interface PipelineDashboardTrend {
  date: string
  buildCount: number
  successCount: number
  failureCount: number
  durationSeconds: number
}

export interface PipelineDashboardResponse {
  overview: PipelineDashboardOverview
  trends: PipelineDashboardTrend[]
  projectRanks: PipelineDashboardMetric[]
  systemRanks: PipelineDashboardMetric[]
  pipelineRanks: PipelineDashboardMetric[]
  environmentRanks: PipelineDashboardMetric[]
  channelRanks: PipelineDashboardMetric[]
}

export interface JenkinsCredentialSyncRecord {
  credentialId: string
  credentialName: string
  credentialCode: string
  credentialType: string
  jenkinsCredentialId: string
  syncStatus: string
  syncMessage: string
  lastSyncAt: number
  referenceCount: number
}

export interface ListJenkinsCredentialSyncParams {
  projectId: string
  buildChannelBindingId: string
  page?: number
  pageSize?: number
}

export interface ListJenkinsCredentialSyncResponse {
  items: JenkinsCredentialSyncRecord[]
  total: number
}

export interface JenkinsCredentialReference {
  pipelineId: string
  pipelineName: string
  pipelineCode: string
  paramCode: string
  paramName: string
  referenceType: string
}

export interface JenkinsCredentialReferenceParams {
  projectId: string
  buildChannelBindingId: string
  credentialId: string
}

export interface JenkinsCredentialReferenceResponse {
  items: JenkinsCredentialReference[]
  total: number
}

export function createDevopsStepCategoryApi(data: DevopsStepCategoryForm) {
  return request.post<{ id: string }>({ url: `${DEVOPS_PIPELINE_BASE}/category`, data })
}

export function updateDevopsStepCategoryApi(id: string, data: DevopsStepCategoryForm) {
  return request.put<string>({ url: `${DEVOPS_PIPELINE_BASE}/category/${id}`, data })
}

export function deleteDevopsStepCategoryApi(id: string) {
  return request.del<string>({ url: `${DEVOPS_PIPELINE_BASE}/category/${id}` })
}

export function getDevopsStepCategoryApi(id: string) {
  return request.get<DevopsStepCategory>({ url: `${DEVOPS_PIPELINE_BASE}/category/${id}` })
}

export function listDevopsStepCategoryApi(params: ListDevopsStepCategoryParams = {}) {
  return request.get<ListDevopsStepCategoryResponse>({
    url: `${DEVOPS_PIPELINE_BASE}/category/list`,
    params
  })
}

export function createDevopsJenkinsStepApi(data: DevopsStepTemplateForm) {
  return request.post<{ id: string }>({ url: `${DEVOPS_PIPELINE_BASE}/jenkins/step`, data })
}

export function updateDevopsJenkinsStepApi(id: string, data: DevopsStepTemplateForm) {
  return request.put<string>({ url: `${DEVOPS_PIPELINE_BASE}/jenkins/step/${id}`, data })
}

export function deleteDevopsJenkinsStepApi(id: string) {
  return request.del<string>({ url: `${DEVOPS_PIPELINE_BASE}/jenkins/step/${id}` })
}

export function getDevopsJenkinsStepApi(id: string) {
  return request.get<DevopsStepTemplate>({ url: `${DEVOPS_PIPELINE_BASE}/jenkins/step/${id}` })
}

export function listDevopsJenkinsStepApi(params: ListDevopsStepTemplateParams = {}) {
  return request.get<ListDevopsStepTemplateResponse>({
    url: `${DEVOPS_PIPELINE_BASE}/jenkins/step/list`,
    params
  })
}

export function updateDevopsJenkinsStepStatusApi(id: string, status: number) {
  return request.put<string>({
    url: `${DEVOPS_PIPELINE_BASE}/jenkins/step/${id}/status`,
    data: { status }
  })
}

export function createDevopsTektonStepApi(data: DevopsStepTemplateForm) {
  return request.post<{ id: string }>({ url: `${DEVOPS_PIPELINE_BASE}/tekton/step`, data })
}

export function updateDevopsTektonStepApi(id: string, data: DevopsStepTemplateForm) {
  return request.put<string>({ url: `${DEVOPS_PIPELINE_BASE}/tekton/step/${id}`, data })
}

export function deleteDevopsTektonStepApi(id: string) {
  return request.del<string>({ url: `${DEVOPS_PIPELINE_BASE}/tekton/step/${id}` })
}

export function getDevopsTektonStepApi(id: string) {
  return request.get<DevopsStepTemplate>({ url: `${DEVOPS_PIPELINE_BASE}/tekton/step/${id}` })
}

export function listDevopsTektonStepApi(params: ListDevopsStepTemplateParams = {}) {
  return request.get<ListDevopsStepTemplateResponse>({
    url: `${DEVOPS_PIPELINE_BASE}/tekton/step/list`,
    params
  })
}

export function listDevopsTektonStepImageApi(params: ListTektonStepImageParams = {}) {
  return request.get<{ items: TektonStepImageItem[]; total: number }>({
    url: `${DEVOPS_PIPELINE_BASE}/tekton/step/image/list`,
    params
  })
}

export function batchUpdateDevopsTektonStepImageApi(items: UpdateTektonStepImageItem[]) {
  return request.put<BatchUpdateTektonStepImageResponse>({
    url: `${DEVOPS_PIPELINE_BASE}/tekton/step/image/batch`,
    data: { items }
  })
}

export function listDevopsTektonRuntimeResourceApi(
  params: ListTektonRuntimeResourceParams,
  options: { showErrorMessage?: boolean } = {}
) {
  return request.get<{ items: TektonRuntimeResource[]; total: number }>({
    url: `${DEVOPS_PIPELINE_BASE}/tekton/runtime-resource/list`,
    params,
    showErrorMessage: options.showErrorMessage
  })
}

export function updateDevopsTektonStepStatusApi(id: string, status: number) {
  return request.put<string>({
    url: `${DEVOPS_PIPELINE_BASE}/tekton/step/${id}/status`,
    data: { status }
  })
}

export function createDevopsPipelineTemplateApi(data: DevopsPipelineTemplateForm) {
  return request.post<{ id: string }>({ url: `${DEVOPS_PIPELINE_BASE}/template`, data })
}

export function updateDevopsPipelineTemplateApi(id: string, data: DevopsPipelineTemplateForm) {
  return request.put<string>({ url: `${DEVOPS_PIPELINE_BASE}/template/${id}`, data })
}

export function deleteDevopsPipelineTemplateApi(id: string) {
  return request.del<string>({ url: `${DEVOPS_PIPELINE_BASE}/template/${id}` })
}

export function getDevopsPipelineTemplateApi(id: string) {
  return request.get<DevopsPipelineTemplate>({ url: `${DEVOPS_PIPELINE_BASE}/template/${id}` })
}

export function listDevopsPipelineTemplateApi(params: ListDevopsPipelineTemplateParams = {}) {
  return request.get<ListDevopsPipelineTemplateResponse>({
    url: `${DEVOPS_PIPELINE_BASE}/template/list`,
    params
  })
}

export function updateDevopsPipelineTemplateStatusApi(id: string, status: number) {
  return request.put<string>({
    url: `${DEVOPS_PIPELINE_BASE}/template/${id}/status`,
    data: { status }
  })
}

export function saveDevopsPipelineTemplateStepsApi(id: string, steps: PipelineTemplateStep[]) {
  return request.put<string>({
    url: `${DEVOPS_PIPELINE_BASE}/template/${id}/steps`,
    data: { steps }
  })
}

export function previewDevopsPipelineTemplateApi(id: string) {
  return request.get<{ pipeline: string }>({
    url: `${DEVOPS_PIPELINE_BASE}/template/${id}/preview`
  })
}

export function listDevopsPipelineTemplateVariablesApi(id: string) {
  return request.get<{ items: PipelineTemplateVariable[]; total: number }>({
    url: `${DEVOPS_PIPELINE_BASE}/template/${id}/variables`
  })
}

export function listDevopsCredentialTypeOptionsApi() {
  return request.get<{ items: CredentialTypeOption[] }>({
    url: `${DEVOPS_PIPELINE_BASE}/options/credential-types`
  })
}

export function listJenkinsCredentialSyncApi(params: ListJenkinsCredentialSyncParams) {
  return request.get<ListJenkinsCredentialSyncResponse>({
    url: `${DEVOPS_PIPELINE_BASE}/jenkins/credential-sync/list`,
    params
  })
}

export function listJenkinsCredentialReferencesApi(params: JenkinsCredentialReferenceParams) {
  return request.get<JenkinsCredentialReferenceResponse>({
    url: `${DEVOPS_PIPELINE_BASE}/jenkins/credential-sync/references`,
    params
  })
}

export function deleteJenkinsCredentialSyncApi(params: JenkinsCredentialReferenceParams) {
  return request.del<string>({
    url: `${DEVOPS_PIPELINE_BASE}/jenkins/credential-sync`,
    params
  })
}

export function listDevopsChannelGroupOptionsApi() {
  return request.get<{ items: ChannelGroupOption[] }>({
    url: `${DEVOPS_PIPELINE_BASE}/options/channel-groups`
  })
}

export interface StepChannelParamMappingItem {
  id: string
  paramType: string
  paramName: string
  groupCode: string
  channelTypeFilter: string
  description: string
  sortOrder: number
  status: number
  isSystem: boolean
}

export interface StepChannelParamMappingForm {
  paramType?: string
  paramName: string
  groupCode: string
  channelTypeFilter?: string
  description?: string
  sortOrder: number
  status: number
}

export interface ListStepChannelParamMappingParams {
  paramType?: string
  groupCode?: string
  status?: number
  page?: number
  pageSize?: number
}

export function listStepChannelParamMappingApi(params: ListStepChannelParamMappingParams = {}) {
  return request.get<{ items: StepChannelParamMappingItem[]; total: number }>({
    url: `${DEVOPS_PIPELINE_BASE}/options/step-channel-param-mapping`,
    params
  })
}

export function createStepChannelParamMappingApi(data: StepChannelParamMappingForm) {
  return request.post<{ id: string }>({
    url: `${DEVOPS_PIPELINE_BASE}/options/step-channel-param-mapping`,
    data
  })
}

export function updateStepChannelParamMappingApi(id: string, data: StepChannelParamMappingForm) {
  return request.put<string>({
    url: `${DEVOPS_PIPELINE_BASE}/options/step-channel-param-mapping/${id}`,
    data
  })
}

export function deleteStepChannelParamMappingApi(id: string) {
  return request.del<string>({
    url: `${DEVOPS_PIPELINE_BASE}/options/step-channel-param-mapping/${id}`
  })
}

export function listDevopsChannelMappingFieldOptionsApi(
  channelGroupCode?: string,
  channelType?: string,
  channelTypeId?: string
) {
  return request.get<{ items: ChannelMappingFieldOption[] }>({
    url: `${DEVOPS_PIPELINE_BASE}/options/channel-mapping-fields`,
    params: { channelGroupCode, channelType, channelTypeId }
  })
}

export function listDevopsDynamicParamOptionsApi(params: ListDynamicParamOptionsParams) {
  return request.get<{ items: DynamicParamOption[]; total: number }>({
    url: `${DEVOPS_PIPELINE_BASE}/options/dynamic-params`,
    params
  })
}

export function createDevopsSystemApi(data: DevopsSystemForm) {
  return request.post<{ id: string }>({ url: `${DEVOPS_PROJECT_BASE}/system`, data })
}

export function updateDevopsSystemApi(id: string, data: DevopsSystemForm) {
  return request.put<string>({ url: `${DEVOPS_PROJECT_BASE}/system/${id}`, data })
}

export function deleteDevopsSystemApi(id: string) {
  return request.del<string>({ url: `${DEVOPS_PROJECT_BASE}/system/${id}` })
}

export function getDevopsSystemApi(id: string) {
  return request.get<DevopsSystem>({ url: `${DEVOPS_PROJECT_BASE}/system/${id}` })
}

export function listDevopsSystemApi(params: ListDevopsSystemParams = {}) {
  return request.get<ListDevopsSystemResponse>({
    url: `${DEVOPS_PROJECT_BASE}/system/list`,
    params
  })
}

export function createDevopsPipelineEnvironmentApi(data: DevopsPipelineEnvironmentForm) {
  return request.post<{ id: string }>({ url: `${DEVOPS_PIPELINE_BASE}/environment`, data })
}

export function updateDevopsPipelineEnvironmentApi(
  id: string,
  data: DevopsPipelineEnvironmentForm
) {
  return request.put<string>({ url: `${DEVOPS_PIPELINE_BASE}/environment/${id}`, data })
}

export function deleteDevopsPipelineEnvironmentApi(id: string) {
  return request.del<string>({ url: `${DEVOPS_PIPELINE_BASE}/environment/${id}` })
}

export function getDevopsPipelineEnvironmentApi(id: string) {
  return request.get<DevopsPipelineEnvironment>({
    url: `${DEVOPS_PIPELINE_BASE}/environment/${id}`
  })
}

export function listDevopsPipelineEnvironmentApi(params: ListDevopsPipelineEnvironmentParams = {}) {
  return request.get<ListDevopsPipelineEnvironmentResponse>({
    url: `${DEVOPS_PIPELINE_BASE}/environment/list`,
    params
  })
}

export function createDevopsPipelineApi(data: DevopsPipelineForm) {
  return request.post<{ id: string }>({
    url: `${DEVOPS_PIPELINE_BASE}/instance`,
    data,
    timeout: 60000
  })
}

export function updateDevopsPipelineApi(id: string, data: DevopsPipelineForm) {
  return request.put<string>({
    url: `${DEVOPS_PIPELINE_BASE}/instance/${id}`,
    data,
    timeout: 60000
  })
}

export function deleteDevopsPipelineApi(id: string) {
  return request.del<string>({ url: `${DEVOPS_PIPELINE_BASE}/instance/${id}` })
}

export function getDevopsPipelineApi(id: string) {
  return request.get<DevopsPipeline>({ url: `${DEVOPS_PIPELINE_BASE}/instance/${id}` })
}

export function listDevopsPipelineApi(params: ListDevopsPipelineParams = {}) {
  return request.get<ListDevopsPipelineResponse>({
    url: `${DEVOPS_PIPELINE_BASE}/instance/list`,
    params
  })
}

export function listDevopsPipelineJenkinsAgentApi(params: ListJenkinsAgentParams) {
  return request.get<ListJenkinsAgentResponse>({
    url: `${DEVOPS_PIPELINE_BASE}/options/jenkins-agents`,
    params
  })
}

export function previewDevopsPipelineApi(id: string) {
  return request.get<{ pipeline: string }>({
    url: `${DEVOPS_PIPELINE_BASE}/instance/${id}/preview`
  })
}

export function runDevopsPipelineApi(
  id: string,
  data: { params?: string; workspaces?: string; triggerType?: string }
) {
  return request.post<{ id: string }>({
    url: `${DEVOPS_PIPELINE_BASE}/instance/${id}/run`,
    data
  })
}

export function stopDevopsPipelineRunApi(id: string) {
  return request.post<string>({ url: `${DEVOPS_PIPELINE_BASE}/run/${id}/stop` })
}

export function deleteDevopsPipelineRunApi(id: string) {
  return request.del<string>({ url: `${DEVOPS_PIPELINE_BASE}/run/${id}` })
}

export function listDevopsPipelineRunApi(params: ListDevopsPipelineRunParams = {}) {
  return request.get<{ items: DevopsPipelineRun[]; total: number }>({
    url: `${DEVOPS_PIPELINE_BASE}/run/list`,
    params
  })
}

export function listDevopsPipelineRunStageApi(runId: string) {
  return request.get<{ items: DevopsPipelineRunStage[] }>({
    url: `${DEVOPS_PIPELINE_BASE}/run/${runId}/stage/list`
  })
}

export function getDevopsPipelineRunYamlSnapshotApi(runId: string) {
  return request.get<DevopsPipelineRunYamlSnapshot>({
    url: `${DEVOPS_PIPELINE_BASE}/run/${runId}/yaml-snapshot`
  })
}

export function listDevopsPipelineRunArtifactApi(runId: string) {
  return request.get<{ items: DevopsPipelineRunArtifact[] }>({
    url: `${DEVOPS_PIPELINE_BASE}/run/${runId}/artifact/list`
  })
}

export function listDevopsPipelineRunInputApi(runId: string) {
  return request.get<{ items: DevopsPipelineRunInput[] }>({
    url: `${DEVOPS_PIPELINE_BASE}/run/${runId}/input/list`
  })
}

export function proceedDevopsPipelineRunInputApi(
  runId: string,
  inputId: string,
  data: { params?: string } = {}
) {
  return request.post<string>({
    url: `${DEVOPS_PIPELINE_BASE}/run/${runId}/input/${inputId}/proceed`,
    data
  })
}

export function abortDevopsPipelineRunInputApi(runId: string, inputId: string) {
  return request.post<string>({
    url: `${DEVOPS_PIPELINE_BASE}/run/${runId}/input/${inputId}/abort`
  })
}

export function getDevopsPipelineDashboardApi(params: { projectId?: string } = {}) {
  return request.get<PipelineDashboardResponse>({
    url: `${DEVOPS_PIPELINE_BASE}/dashboard`,
    params
  })
}

export interface DevopsPipelineRunLogParams {
  stageId?: string
  full?: boolean
  taskRunName?: string
  podName?: string
  containerName?: string
}

export function getDevopsPipelineRunLogApi(runId: string, params: DevopsPipelineRunLogParams = {}) {
  return request.get<{ content: string; archived: boolean; source: string; changed?: boolean }>({
    url: `${DEVOPS_PIPELINE_BASE}/run/${runId}/log`,
    params
  })
}

export function createDevopsPipelineRunLogStream(
  runId: string,
  params: DevopsPipelineRunLogParams = {}
) {
  const query = new URLSearchParams()
  query.append('stream', 'true')
  if (params.stageId) query.append('stageId', params.stageId)
  if (params.full) query.append('full', 'true')
  if (params.taskRunName) query.append('taskRunName', params.taskRunName)
  if (params.podName) query.append('podName', params.podName)
  if (params.containerName) query.append('containerName', params.containerName)
  const { accessToken } = useUserStore()
  if (accessToken) query.append('token', accessToken)

  const baseUrl = String(import.meta.env.VITE_API_URL || '').replace(/\/$/, '')
  return new EventSource(`${baseUrl}${DEVOPS_PIPELINE_BASE}/run/${runId}/log?${query.toString()}`)
}
