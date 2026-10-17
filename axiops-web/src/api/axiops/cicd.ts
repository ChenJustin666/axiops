import request from '@/utils/http'
import { serviceNameError } from '@/utils/service-name'

export interface ServiceEnvironmentConfig {
  helm?: HelmEnvironmentSettings
  adjustments?: string[]
  environment_yaml_revision_id?: number
  manifest?: string
  final_values?: string
  config_digest?: string
  current_deployment?: {
    snapshot_id: number
    run_id: number
    cluster_id: number
    namespace: string
    images: Record<string, string>
  }
  assets?: import('./service-assets').ServiceAssets
  environment_id: number
  service_id: number
  revision_id: number
  latest_revision_id: number
  bound: boolean
  definition: string
  format: string
  image_ref: string
  replicas: number | null
  variables: Array<{
    key: string
    value: string
    secret: boolean
    origin: string
    credential_id?: number
  }>
  token: string
}
export interface SaveServiceEnvironmentConfig {
  template?: { id: number; revision: number }
  helm_credentials?: Record<string, { credential_id: number; key: string } | null>
  helm?: HelmEnvironmentSettings
  definition?: string
  reset_definition?: boolean
  assets?: import('./service-assets').ServiceAssets
  source?: { environment_id: number; token: string }
  targets: Array<{ environment_id: number; token: string }>
  revision_id: number
  variables: Record<string, string | null>
}
export interface HelmEnvironmentSettings {
  source?: HelmValuesSource
  shared_values?: string
  shared_revision?: number
  credential_keys?: Record<string, string>
  values: string
  image: { repository_path: string; tag_path: string; ref_path: string }
}
export interface HelmValuesSource {
  code_source_id: number
  repo_path: string
  ref: string
  path: string
  commit_id: string
  base: string
}
export function validateHelmChartApi(
  serviceId: number,
  files: Array<{ path: string; content: string }>
) {
  return request.post<{ definition: string }>({
    url: `/cicd/v1/services/${serviceId}/helm-chart/validate`,
    data: { files }
  })
}
export function readHelmValuesApi(
  projectId: number,
  serviceId: number,
  source: Partial<HelmValuesSource>
) {
  return request.post<HelmValuesSource>({
    url: `/cicd/v1/projects/${projectId}/helm-values/read`,
    data: { service_id: serviceId, source },
    timeout: 130000
  })
}
export interface HelmSharedValues {
  values: string
  revision: number
  services: Array<{
    service_id: number
    name: string
    token: string
    changed: boolean
    before?: string
    after?: string
  }>
  saved: boolean
  validated: boolean
}
export function getHelmSharedValuesApi(environmentId: number) {
  return request.get<HelmSharedValues>({
    url: `/cicd/v1/environments/${environmentId}/helm-shared-values`
  })
}
export function saveHelmSharedValuesApi(
  environmentId: number,
  data: { values: string; revision: number; tokens: Record<number, string> },
  preview = false
) {
  return request.post<HelmSharedValues>({
    url: `/cicd/v1/environments/${environmentId}/helm-shared-values${preview ? '?preview=1' : ''}`,
    data,
    timeout: 130000
  })
}
export interface InitialHelmService {
  service_id: number
  revision_id: number
  helm: HelmEnvironmentSettings
}
export interface ServiceEnvironmentConfigResult {
  saved: boolean
  validated: boolean
  configs: ServiceEnvironmentConfig[]
}
export function getServiceEnvironmentConfigApi(environmentId: number, serviceId: number) {
  return request.get<ServiceEnvironmentConfig>({
    url: `/cicd/v1/environments/${environmentId}/services/${serviceId}/config`
  })
}
export async function createConfiguredServiceApi(
  environmentId: number,
  data: {
    name: string
    code: string
    definition: string
    variables: Record<string, string>
    assets?: import('./service-assets').ServiceAssets
  },
  preview = false
) {
  const result = await request.post<ServiceEnvironmentConfigResult & { service_id: number }>({
    url: `/cicd/v1/environments/${environmentId}/services${preview ? '?preview=1' : ''}`,
    data
  })
  const config = result?.configs?.[0]
  if (
    result?.saved !== !preview ||
    result?.validated !== true ||
    !Number.isInteger(result?.service_id) ||
    result.service_id <= 0 ||
    result.configs?.length !== 1 ||
    config?.environment_id !== environmentId ||
    config?.service_id !== result.service_id ||
    !config.bound ||
    config.revision_id <= 0 ||
    typeof config.manifest !== 'string' ||
    !config.manifest.trim() ||
    !/^[a-f0-9]{64}$/.test(config.token || '')
  )
    throw new Error('服务创建接口返回的配置校验结果不完整，请重新加载确认')
  return result
}
export async function saveServiceEnvironmentConfigApi(
  serviceId: number,
  data: SaveServiceEnvironmentConfig,
  preview = false
) {
  const result = await request.post<ServiceEnvironmentConfigResult>({
    url: `/cicd/v1/services/${serviceId}/environment-config${preview ? '?preview=1' : ''}`,
    data,
    timeout: 130000
  })
  const targets = new Set(data.targets.map((target) => target.environment_id))
  if (
    result?.saved !== !preview ||
    result?.validated !== true ||
    !Array.isArray(result?.configs) ||
    result.configs.length !== targets.size ||
    !result.configs.every(
      (config) =>
        config &&
        config.service_id === serviceId &&
        config.bound === true &&
        Number.isInteger(config.revision_id) &&
        config.revision_id > 0 &&
        typeof config.definition === 'string' &&
        Array.isArray(config.variables) &&
        typeof config.token === 'string' &&
        /^[a-f0-9]{64}$/.test(config.token) &&
        targets.delete(config.environment_id)
    ) ||
    targets.size > 0
  ) {
    throw new Error(
      preview
        ? '配置预览返回不完整，无法确认差异，请重新加载后重试。'
        : '无法确认配置保存结果，请重新加载核对目标环境后再操作。'
    )
  }
  return result
}

export type DeliveryType = 'k8s_yaml' | 'helm' | 'host'

export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export type InitializationStatus = 'pending' | 'in_progress' | 'completed'

export interface AxiOpsProject {
  id: number
  name: string
  code: string
  description: string
  status: string
  delivery_type: DeliveryType | 'unclassified'
  delivery_status: string
  owner_user_id: number
  initialization_status: InitializationStatus
  initialization_step: number
  initialized_at: string | null
  created_at: string
  updated_at: string
}

export interface CreateProjectInput {
  name: string
  code: string
  description: string
  delivery_type: DeliveryType
}

export interface UpdateProjectInput {
  name?: string
  description?: string
  status?: string
  owner_user_id?: number
  initialization_status?: InitializationStatus
  initialization_step?: number
}

export interface AxiOpsService {
  id: number
  project_id: number
  name: string
  code: string
  description: string
  delivery_type: DeliveryType
  source_type: 'git' | 'manual' | 'template'
  source_ref?: {
    helm_repository?: {
      code_source_id: number
      repo_path: string
      ref: string
      chart_path: string
      commit_id: string
      revision_id: number
    }
  }
  repo_url: string
  repo_branch: string
  language: string
  framework: string
  image_name: string
  code_source_id: number
  registry_id?: number
  current_revision_id: number
  component_count: number
  status: string
  updated_at: string
}

export interface CreateServiceInput {
  name: string
  code: string
  description: string
  source_type: 'git' | 'manual' | 'template'
  repo_url: string
  repo_branch: string
  language: string
  framework: string
  image_name: string
  code_source_id?: number
  registry_id?: number
  repository_id?: string
}

export type UpdateServiceInput = Partial<Omit<CreateServiceInput, 'code'>> & {
  status?: 'active' | 'inactive'
}

export interface AxiOpsCredential {
  id: number
  name: string
  type: string
  description: string
  project_id: number
  created_at: string
  updated_at: string
  keys?: string[]
  token?: string
}

export interface AxiOpsCodeSource {
  id: number
  project_id: number
  name: string
  type: string
  url: string
  credential_id: number
  webhook_credential_id: number
  status: string
  created_at: string
  updated_at: string
}

export interface AxiOpsRepository {
  id: string
  name: string
  full_name: string
  clone_url: string
  default_branch: string
  private: boolean
}

export interface AxiOpsBranch {
  name: string
  commit_id?: string
}

export interface AxiOpsRepositoryFile {
  name: string
  path: string
  type: 'file' | 'dir'
  size: number
}

export interface ServiceImportRequest {
  code_source_id: number
  repo_path: string
  ref: string
  paths: string[]
  chart_path?: string
  spec_path?: string
  expected_commit?: string
  expected_revision_id?: number
  accept_local_changes?: boolean
}

export interface ServiceImportPreview {
  commit_id: string
  content_hash: string
  files?: Array<{ path: string; content: string }>
  current_revision_id?: number
  local_changes?: boolean
  resource_count?: number
  component_count: number
  components?: string[]
  images?: string[]
  resources?: Array<{
    api_version: string
    kind: string
    name: string
    namespace?: string
    images?: string[]
  }>
  chart_path?: string
  metadata?: { name: string; version: string; app_version?: string; description?: string }
  file_count?: number
  template_count?: number
  default_value_keys?: string[]
  name?: string
  os?: string
  artifact_source?: string
  timeout_seconds?: number
  batch_size?: number
  batch_percent?: number
  interval_seconds?: number
  scripts?: Array<{ phase: string; path: string; sha256: string; bytes: number }>
}

export interface AxiOpsServiceRevision {
  id: number
  service_id: number
  revision_number: number
  source_commit: string
  definition_format: string
  content_hash: string
  definition?: string
  created_at: string
}

export interface AxiOpsBuildConfig {
  dockerfile_source?: 'repository' | 'inline'
  dockerfile_content?: string
  id: number
  service_id: number
  template_type: string
  build_mode: 'dockerfile' | 'jar_image' | 'artifact' | string
  build_image: string
  build_command: string
  dockerfile_path: string
  build_context: string
  image_repo: string
  cache_enabled: boolean
  cache_dir: string
  cache_strategy: 'emptydir' | 'pvc' | 'registry' | string
  cache_pvc_name: string
  timeout_seconds: number
  branch_rules?: string
  env_vars?: string
  resource_cpu: string
  resource_memory: string
  provider: string
  jenkins_job_name: string
  host_agent_id: number
}

export interface SaveBuildConfigInput {
  dockerfile_source?: 'repository' | 'inline'
  dockerfile_content?: string
  service_binding?: {
    code_source_id?: number
    repo_url?: string
    repo_branch?: string
    registry_id?: number
  }
  template_type: string
  build_mode?: 'dockerfile' | 'jar_image' | 'artifact' | string
  build_image: string
  build_command: string
  dockerfile_path: string
  build_context: string
  image_repo: string
  cache_enabled: boolean
  cache_dir: string
  cache_strategy?: 'emptydir' | 'pvc' | 'registry' | string
  cache_pvc_name?: string
  timeout_seconds: number
  branch_rules: { includes: string[]; excludes: string[] }
  env_vars: Array<{ key: string; value: string }>
  resource_cpu: string
  resource_memory: string
  provider: string
}

export interface AxiOpsPlatformBuildSettings {
  execution_policy?: {
    log_retention_days?: number
    node_selector?: Record<string, string>
    tolerations?: Array<{ key: string; value: string; effect: string }>
    max_builds_per_node?: number
    workspace_size?: string
  }
  enabled: boolean
  cluster_id: number
  namespace: string
  builder_image: string
  build_command: string
  timeout_seconds: number
  resource_cpu: string
  resource_memory: string
  cache_pvc_name: string
  cache_dir: string
  job_retention_seconds: number
  updated_by?: number
  updated_at?: string
}

export type SavePlatformBuildSettingsInput = Omit<
  AxiOpsPlatformBuildSettings,
  'updated_by' | 'updated_at'
>

export interface AxiOpsConcurrencyPolicy {
  scope_type: string
  scope_id: number
  max_running: number
  queue_timeout_seconds: number
  inherited?: boolean
  updated_at?: string
}

export interface SaveConcurrencyPolicyInput {
  project_id: number
  max_running: number
  queue_timeout_seconds: number
}

export interface AxiOpsEnvironment {
  id: number
  project_id: number
  name: string
  code: string
  environment_type: 'k8s_namespace' | 'host_group'
  cluster_id: number
  namespace: string
  host_group_ref: string
  binding_status: string
  is_production: boolean
  deploy_approval: boolean
  status: string
  updated_at: string
}

export interface CreateEnvironmentInput {
  helm_services?: InitialHelmService[]
  name: string
  code: string
  cluster_id?: number
  namespace?: string
  host_group_ref?: string
  is_production: boolean
  deploy_approval: boolean
}

export interface AxiOpsVariableItem {
  id: number
  key: string
  value?: string
  credential_id?: number
  is_secret: boolean
}

export interface AxiOpsVariableSet {
  id: number
  project_id: number
  environment_id: number
  service_id: number
  name: string
  scope: string
  revision: number
  items: AxiOpsVariableItem[]
  updated_at: string
}

export interface UpdateVariableSetInput {
  revision: number
  items: Array<{
    key: string
    value: string
    credential_id: number
    is_secret: boolean
  }>
}

export interface AxiOpsEnvironmentServiceOverride {
  id: number
  project_environment_id: number
  service_id: number
  service_revision_id: number
  image_ref: string
  replicas?: number
  revision: number
  updated_at: string
}

export interface AxiOpsEnvironmentResource {
  id: number
  environment_id: number
  name: string
  kind: 'Ingress' | 'ConfigMap' | 'Secret' | 'PersistentVolumeClaim'
  manifest: string
  revision: number
  updated_at: string
}

export interface SaveEnvironmentResourceInput {
  id?: number
  name: string
  kind: AxiOpsEnvironmentResource['kind']
  manifest: string
  expected_revision?: number
}

export interface SaveEnvironmentServiceOverrideInput {
  expected_revision: number
  service_revision_id: number
  image_ref: string
  replicas: number | null
}

export interface AxiOpsResourceChange {
  api_version: string
  kind: string
  namespace: string
  name: string
  action: 'create' | 'update' | 'delete' | 'unchanged'
}

export interface AxiOpsDeploymentPreview {
  delivery_type: 'k8s_yaml' | 'helm'
  project_id: number
  environment_id: number
  service_id: number
  service_revision_id: number
  cluster_id: number
  namespace: string
  release_name?: string
  changes: AxiOpsResourceChange[]
  summary: { create: number; update: number; delete: number; unchanged: number }
  values?: Record<string, unknown>
}

export interface AxiOpsCluster {
  id: number
  name: string
  code: string
  provider: string
  region: string
  version: string
  status: string
  node_count: number
  cpu_total?: string
  memory_total?: string
  api_server?: string
  description?: string
  last_heartbeat?: string
  created_at?: string
  updated_at?: string
}

export interface CreateClusterInput {
  name: string
  code: string
  description?: string
  provider?: string
  region?: string
  api_server: string
  kubeconfig: string
}

export interface UpdateClusterInput {
  name?: string
  description?: string
  api_server?: string
  kubeconfig?: string
}

export type WorkflowNodeType =
  | 'build'
  | 'yaml_deploy'
  | 'helm_deploy'
  | 'host_deploy'
  | 'test'
  | 'approval'
  | 'notify'

export interface WorkflowNode {
  key: string
  name: string
  type: WorkflowNodeType
  config?: Record<string, unknown>
  depends_on?: string[]
  timeout_seconds?: number
  retry_limit?: number
}

export interface AxiOpsWorkflow {
  requires_environment?: boolean
  target_environment_id?: number
  id: number
  project_id: number
  name: string
  code: string
  description: string
  delivery_type: DeliveryType
  definition_version: number
  status: string
  nodes?: WorkflowNode[]
  node_count: number
  updated_at: string
}

export interface AxiOpsBuildRun {
  git_commit_url?: string
  git_changes?: AxiOpsGitChanges
  id: number
  project_id: number
  service_id: number
  build_number: number
  status: string
  git_branch: string
  git_commit?: string
  git_commit_short: string
  git_author?: string
  git_author_email?: string
  git_message?: string
  git_compare_url?: string
  image_url: string
  image_digest?: string
  artifact_url: string
  artifact_sha256?: string
  image_verification_status?: string
  artifact_verification_status?: string
  output_verification_error?: string
  job_name?: string
  pod_name?: string
  node_name?: string
  duration_seconds?: number
  error_message?: string
  created_at: string
}

export interface AxiOpsTaskRun {
  retry_count?: number
  next_retry_at?: string
  retry_history?: { provider_run_id: string; error_message: string; finished_at: string }[]
  id: number
  workflow_run_id: number
  node_key: string
  node_type: WorkflowNodeType
  depends_on: string[]
  node_snapshot?: WorkflowNode
  status: string
  attempt: number
  provider_run_id: string
  log_ref: string
  log_text?: string
  error_message: string
  created_at: string
  started_at?: string
  finished_at?: string
}

export interface AxiOpsWorkflowRun {
  run_number?: number
  build_commits?: AxiOpsBuildCommitSummary[]
  id: number
  workflow_id: number
  project_id: number
  environment_id: number
  definition_version: number
  trigger_type: string
  operation_type: string
  trigger_user_id: number
  trigger_username?: string
  status: string
  error_message: string
  context_snapshot?: Record<string, unknown>
  created_at: string
  started_at?: string
  finished_at?: string
  git_branch?: string
  git_commit_short?: string
  tasks?: AxiOpsTaskRun[]
}

export interface AxiOpsBuildCommitSummary {
  build_id: number
  service_id: number
  branch: string
  commit: string
  subject: string
  author: string
  url: string
}

export interface AxiOpsGitChanges {
  status: 'complete' | 'first_build' | 'unavailable'
  head: string
  base_commit?: string
  base_build_id?: number
  base_build_number?: number
  commits: { commit: string; author: string; subject: string; authored_at: string; url?: string }[]
  files: { action: string; path: string }[]
  commits_truncated: boolean
  files_truncated: boolean
  compare_url?: string
}

export interface AxiOpsBuildLog {
  id: number
  build_run_id: number
  stage: string
  log_url: string
  log_text?: string
  line_count: number
  truncated: boolean
  started_at?: string
  finished_at?: string
  created_at: string
}

export interface CreateWorkflowInput {
  target_environment_id?: number
  name: string
  code: string
  description: string
  definition: {
    delivery_type: DeliveryType
    nodes: WorkflowNode[]
  }
}

export interface UpdateWorkflowInput {
  target_environment_id?: number
  expected_version: number
  name: string
  description: string
  definition: {
    delivery_type: DeliveryType
    nodes: WorkflowNode[]
  }
}

export interface WorkflowValidationIssue {
  field: string
  code: string
  message: string
}

export type NotificationChannel = 'feishu' | 'dingtalk' | 'wecom' | 'email' | 'webhook'

export interface AxiOpsNotificationSubscription {
  id: number
  project_id: number
  user_id: number
  name: string
  channel: NotificationChannel
  credential_id: number
  recipient: string
  on_success: boolean
  on_failure: boolean
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface CreateNotificationSubscriptionInput {
  user_id: number
  name: string
  channel: NotificationChannel
  credential_id: number
  recipient: string
  on_success: boolean
  on_failure: boolean
}

export interface AxiOpsNotificationDelivery {
  id: number
  subscription_id: number
  project_id: number
  subscription_name: string
  channel: NotificationChannel
  event_type: string
  event_id: number
  status: 'pending' | 'processing' | 'succeeded' | 'dead'
  attempts: number
  next_attempt_at: string
  last_error: string
  created_at: string
  updated_at: string
}

export function listProjectsApi(
  params: {
    keyword?: string
    status?: string
    page?: number
    page_size?: number
  },
  options: { silent?: boolean } = {}
) {
  return request.get<PageResult<AxiOpsProject>>({
    url: '/platform/v1/projects',
    params,
    showErrorMessage: options.silent !== true
  })
}

export function getProjectApi(id: number, options: { silent?: boolean } = {}) {
  return request.get<AxiOpsProject>({
    url: `/platform/v1/projects/${id}`,
    showErrorMessage: options.silent !== true
  })
}

export function createProjectApi(data: CreateProjectInput) {
  return request.post<{ id: number }>({ url: '/platform/v1/projects', data })
}

export function updateProjectApi(id: number, data: UpdateProjectInput) {
  return request.put<void>({ url: `/platform/v1/projects/${id}`, data })
}

export interface ProjectDeletionItem {
  kind: string
  name: string
  count: number
}

export interface ProjectDeletionCheck {
  project_id: number
  project_name: string
  can_delete: boolean
  items: ProjectDeletionItem[]
  blockers: ProjectDeletionItem[]
}

export function checkProjectDeletionApi(id: number) {
  return request.get<ProjectDeletionCheck>({ url: `/platform/v1/projects/${id}/deletion-check` })
}

export function deleteProjectApi(id: number, confirmName: string, permanent = false) {
  return request.del<void>({
    url: `/platform/v1/projects/${id}`,
    data: { confirm_name: confirmName, permanent }
  })
}

export function listServicesApi(
  projectId: number,
  params: { keyword?: string; status?: string; page?: number; page_size?: number } = {},
  options: { silent?: boolean } = {}
) {
  return request.get<PageResult<AxiOpsService>>({
    url: '/cicd/v1/services',
    params: { project_id: projectId, page: 1, page_size: 200, ...params },
    showErrorMessage: options.silent !== true
  })
}

// Project workbenches need the complete authorized service set. Fetch it in
// bounded pages so large projects do not silently stop at the first 200 rows.
export async function listAllServicesApi(
  projectId: number,
  params: { keyword?: string; status?: string } = {},
  options: { silent?: boolean } = {}
) {
  const pageSize = 500
  const first = await listServicesApi(
    projectId,
    { ...params, page: 1, page_size: pageSize },
    options
  )
  const list = [...(first.list || [])]
  const total = Math.min(first.total || list.length, 10000)
  for (let page = 2; list.length < total; page += 1) {
    const result = await listServicesApi(
      projectId,
      { ...params, page, page_size: pageSize },
      options
    )
    if (!result.list?.length) break
    list.push(...result.list)
  }
  return list.slice(0, total)
}

export function createServiceApi(projectId: number, data: CreateServiceInput) {
  const error = serviceNameError(data.name)
  if (error) return Promise.reject(new Error(error))
  return request.post<{ id: number }>({ url: `/cicd/v1/projects/${projectId}/services`, data })
}

export function updateServiceApi(id: number, data: UpdateServiceInput) {
  return request.put<void>({ url: `/cicd/v1/services/${id}`, data })
}

export function deleteServiceApi(id: number) {
  return request.del<void>({ url: `/cicd/v1/services/${id}` })
}

export interface ServiceDeletionCheck {
  can_delete: boolean
  can_purge: boolean
  status: string
  blockers: Array<{
    kind: 'environment' | 'workflow' | 'build' | 'run'
    id: number
    name: string
    revision: number
  }>
  deployment_history: number
}
export const checkServiceDeletionApi = (id: number) =>
  request.get<ServiceDeletionCheck>({ url: `/cicd/v1/services/${id}/deletion-check` })
export const restoreServiceApi = (id: number) =>
  request.post<void>({ url: `/cicd/v1/services/${id}/restore` })
export const purgeServiceApi = (id: number, confirmCode: string) =>
  request.post<void>({ url: `/cicd/v1/services/${id}/purge`, data: { confirm_code: confirmCode } })
export const renameServiceApi = (id: number, name: string) =>
  request.post<void>({ url: `/cicd/v1/services/${id}/rename`, data: { name } })
export const detachServiceEnvironmentApi = (id: number, environmentId: number, revision: number) =>
  request.del<void>({
    url: `/cicd/v1/services/${id}/environment-bindings/${environmentId}`,
    params: { revision }
  })

export function listCredentialsApi(params: { type?: string; project_id?: number } = {}) {
  return request.get<AxiOpsCredential[]>({ url: '/platform/v1/credentials', params })
}

export function createCredentialApi(data: {
  name: string
  type: string
  description: string
  data: string
  project_id: number
}) {
  return request.post<{ id: number }>({ url: '/platform/v1/credentials', data })
}

export function deleteCredentialApi(id: number) {
  return request.del<void>({ url: `/platform/v1/credentials/${id}` })
}

export function getCredentialApi(id: number) {
  return request.get<AxiOpsCredential>({ url: `/platform/v1/credentials/${id}` })
}
export function deriveSecretCredentialApi(
  id: number,
  data: { name: string; token: string; retain: string[]; values: Record<string, string> }
) {
  return request.post<{ id: number }>({ url: `/platform/v1/credentials/${id}/derive-secret`, data })
}

export function listCodeSourcesApi(projectId: number, options: { silent?: boolean } = {}) {
  return request.get<AxiOpsCodeSource[]>({
    url: '/cicd/v1/code-sources',
    params: { project_id: projectId },
    showErrorMessage: options.silent !== true
  })
}

export function createCodeSourceApi(
  projectId: number,
  data: { name: string; type: string; url: string; credential_id: number }
) {
  return request.post<{ id: number }>({ url: `/cicd/v1/projects/${projectId}/code-sources`, data })
}

export function createGlobalCodeSourceApi(data: {
  name: string
  type: string
  url: string
  credential_id: number
}) {
  return request.post<{ id: number }>({ url: '/cicd/v1/code-sources', data })
}

export function updateCodeSourceApi(
  id: number,
  data: {
    name?: string
    type?: string
    url?: string
    credential_id?: number
    status?: 'active' | 'inactive'
  }
) {
  return request.put<void>({ url: `/cicd/v1/code-sources/${id}`, data })
}

export function listRepositoriesApi(sourceId: number) {
  return request.get<AxiOpsRepository[]>({ url: `/cicd/v1/code-sources/${sourceId}/repositories` })
}

export function listBranchesApi(sourceId: number, repoPath: string) {
  return request.get<AxiOpsBranch[]>({
    url: `/cicd/v1/code-sources/${sourceId}/branches`,
    params: { repo_path: repoPath },
    showErrorMessage: false
  })
}

export function listRepositoryFilesApi(
  sourceId: number,
  params: { repo_path: string; ref: string; path?: string }
) {
  return request.get<AxiOpsRepositoryFile[]>({
    url: `/cicd/v1/code-sources/${sourceId}/repository-files`,
    params
  })
}

export function checkCodeSourceApi(id: number) {
  return request.post<{ configuration_valid: boolean }>({
    url: `/cicd/v1/code-sources/${id}/check`
  })
}

export function previewServiceImportApi(service: AxiOpsService, data: ServiceImportRequest) {
  const kind =
    service.delivery_type === 'helm' ? 'helm' : service.delivery_type === 'host' ? 'host' : 'yaml'
  return request.post<ServiceImportPreview>({
    url: `/cicd/v1/services/${service.id}/${kind}-import/preview`,
    ...(kind === 'helm' ? { timeout: 130000 } : {}),
    data
  })
}

export function importServiceRevisionApi(service: AxiOpsService, data: ServiceImportRequest) {
  const kind =
    service.delivery_type === 'helm' ? 'helm' : service.delivery_type === 'host' ? 'host' : 'yaml'
  return request.post<AxiOpsServiceRevision>({
    url: `/cicd/v1/services/${service.id}/${kind}-import`,
    ...(kind === 'helm' ? { timeout: 130000 } : {}),
    data
  })
}

export function listServiceRevisionsApi(serviceId: number, latest = false) {
  return request.get<AxiOpsServiceRevision[]>({
    url: `/cicd/v1/services/${serviceId}/revisions${latest ? '?latest=1' : ''}`
  })
}

export function createServiceRevisionApi(
  serviceId: number,
  data: {
    source_commit?: string
    definition: string
    expected_revision_id?: number
    template?: { id: number; revision: number }
  }
) {
  return request.post<AxiOpsServiceRevision>({
    url: `/cicd/v1/services/${serviceId}/revisions`,
    data
  })
}

export function getBuildConfigApi(serviceId: number) {
  return request.get<AxiOpsBuildConfig>({
    url: `/cicd/v1/services/${serviceId}/build-config`,
    showErrorMessage: false
  })
}

export function saveBuildConfigApi(serviceId: number, data: SaveBuildConfigInput) {
  return request.put<void>({ url: `/cicd/v1/services/${serviceId}/build-config`, data })
}

export function getPlatformBuildSettingsApi() {
  return request.get<AxiOpsPlatformBuildSettings>({ url: '/cicd/v1/build-platform-settings' })
}

export function savePlatformBuildSettingsApi(data: SavePlatformBuildSettingsInput) {
  return request.put<AxiOpsPlatformBuildSettings>({
    url: '/cicd/v1/build-platform-settings',
    data
  })
}

export function getConcurrencyPolicyApi(projectId = 0) {
  return request.get<AxiOpsConcurrencyPolicy>({
    url: '/cicd/v1/build-concurrency',
    params: { project_id: projectId }
  })
}

export function saveConcurrencyPolicyApi(data: SaveConcurrencyPolicyInput) {
  return request.put<AxiOpsConcurrencyPolicy>({ url: '/cicd/v1/build-concurrency', data })
}

export function listEnvironmentsApi(projectId: number, options: { silent?: boolean } = {}) {
  return request.get<AxiOpsEnvironment[]>({
    url: `/cicd/v1/projects/${projectId}/environments`,
    showErrorMessage: options.silent !== true
  })
}

export function createEnvironmentApi(
  projectId: number,
  data: CreateEnvironmentInput,
  preview = false
) {
  return request.post<{ id: number; saved?: boolean; configs?: ServiceEnvironmentConfig[] }>({
    url: `/cicd/v1/projects/${projectId}/environments${preview ? '?preview=1' : ''}`,
    data,
    timeout: 130000
  })
}

export function cloneEnvironmentApi(
  sourceEnvironmentId: number,
  data: {
    name: string
    code: string
    cluster_id: number
    namespace: string
    host_group_ref?: string
    is_production?: boolean
    deploy_approval?: boolean
  }
) {
  return request.post<{ id: number }>({
    url: `/cicd/v1/environments/${sourceEnvironmentId}/clone`,
    data
  })
}

export function getEnvironmentApi(environmentId: number) {
  return request.get<AxiOpsEnvironment>({
    url: `/cicd/v1/environments/${environmentId}`
  })
}

export function getEnvironmentVariableSetApi(environmentId: number) {
  return request.get<AxiOpsVariableSet>({
    url: `/cicd/v1/environments/${environmentId}/variable-set`
  })
}

export function getEnvironmentServiceVariableSetApi(environmentId: number, serviceId: number) {
  return request.get<AxiOpsVariableSet>({
    url: `/cicd/v1/environments/${environmentId}/services/${serviceId}/variable-set`,
    showErrorMessage: false
  })
}

export function updateEnvironmentVariableSetApi(
  variableSetId: number,
  data: UpdateVariableSetInput
) {
  return request.put<AxiOpsVariableSet>({
    url: `/cicd/v1/variable-sets/${variableSetId}`,
    data
  })
}

export function getEnvironmentServiceOverrideApi(environmentId: number, serviceId: number) {
  return request.get<AxiOpsEnvironmentServiceOverride>({
    url: `/cicd/v1/environments/${environmentId}/services/${serviceId}/override`,
    showErrorMessage: false
  })
}

export function saveEnvironmentServiceOverrideApi(
  environmentId: number,
  serviceId: number,
  data: SaveEnvironmentServiceOverrideInput
) {
  return request.put<AxiOpsEnvironmentServiceOverride>({
    url: `/cicd/v1/environments/${environmentId}/services/${serviceId}/override`,
    data
  })
}

export function listEnvironmentResourcesApi(environmentId: number) {
  return request.get<AxiOpsEnvironmentResource[]>({
    url: `/cicd/v1/environments/${environmentId}/resources`
  })
}

export function saveEnvironmentResourceApi(
  environmentId: number,
  data: SaveEnvironmentResourceInput
) {
  return data.id
    ? request.put<AxiOpsEnvironmentResource>({
        url: `/cicd/v1/environments/${environmentId}/resources`,
        data
      })
    : request.post<AxiOpsEnvironmentResource>({
        url: `/cicd/v1/environments/${environmentId}/resources`,
        data
      })
}

export function deleteEnvironmentResourceApi(environmentId: number, resourceId: number) {
  return request.del<void>({
    url: `/cicd/v1/environments/${environmentId}/resources/${resourceId}`
  })
}

export function previewEnvironmentResourcesApi(
  environmentId: number,
  params: { service_id: number; service_revision_id?: number; release_name?: string }
) {
  return request.get<AxiOpsDeploymentPreview>({
    url: `/cicd/v1/environments/${environmentId}/render-preview`,
    params
  })
}

export function listClustersApi(projectId?: number, options: { silent?: boolean } = {}) {
  return request.get<AxiOpsCluster[]>({
    url: '/platform/v1/clusters',
    params: { status: 'connected', ...(projectId ? { project_id: projectId } : {}) },
    showErrorMessage: options.silent !== true
  })
}

export function getClusterApi(id: number) {
  return request.get<AxiOpsCluster>({ url: `/platform/v1/clusters/${id}` })
}

export function createClusterApi(data: CreateClusterInput) {
  return request.post<{ id: number }>({ url: '/platform/v1/clusters', data })
}

export function updateClusterApi(id: number, data: UpdateClusterInput) {
  return request.put<void>({ url: `/platform/v1/clusters/${id}`, data })
}

export function testClusterConnectionApi(id: number) {
  return request.post<{ message: string }>({ url: `/platform/v1/clusters/${id}/test-connect` })
}

export function deleteClusterApi(id: number) {
  return request.del<void>({ url: `/platform/v1/clusters/${id}` })
}

export function listWorkflowsApi(projectId: number, options: { silent?: boolean } = {}) {
  return request.get<AxiOpsWorkflow[]>({
    url: `/cicd/v1/projects/${projectId}/workflows`,
    showErrorMessage: options.silent !== true
  })
}

export function getWorkflowApi(id: number) {
  return request.get<AxiOpsWorkflow>({ url: `/cicd/v1/workflows/${id}` })
}

export function validateWorkflowApi(
  id: number,
  definition: { delivery_type: DeliveryType; nodes: WorkflowNode[] }
) {
  return request.post<{ valid: boolean; issues: WorkflowValidationIssue[] }>({
    url: `/cicd/v1/workflows/${id}/validate`,
    data: definition
  })
}

export function updateWorkflowApi(id: number, data: UpdateWorkflowInput) {
  return request.put<AxiOpsWorkflow>({ url: `/cicd/v1/workflows/${id}`, data })
}

export function runWorkflowApi(
  id: number,
  data: {
    services?: Array<{
      service_id: number
      service_revision_id: number
      branch?: string
      source_build_id?: number
    }>
    execution_mode?: 'build' | 'existing_image' | 'config_only'
    source_snapshot_id?: number
    expected_config_digest?: string
    source_build_id?: number
    environment_id: number
    trigger_type?: string
    service_id?: number
    service_revision_id?: number
    select_workflow_service?: boolean
    use_environment_config?: boolean
    branch?: string
    deployment_strategy?: 'rolling' | 'canary' | 'bluegreen'
  }
) {
  return request.post<{ id: number }>({ url: `/cicd/v1/workflows/${id}/runs`, data })
}

export function listWorkflowRunsApi(
  id: number,
  params: { page?: number; page_size?: number } = {}
) {
  return request.get<PageResult<AxiOpsWorkflowRun>>({
    url: `/cicd/v1/workflows/${id}/runs`,
    params: { page: 1, page_size: 20, ...params }
  })
}

export function getWorkflowRunApi(id: number) {
  return request.get<AxiOpsWorkflowRun>({ url: `/cicd/v1/workflow-runs/${id}` })
}

export interface DeploymentVersion {
  service_name?: string
  snapshot_id: number
  run_id: number
  run_number: number
  workflow_id: number
  revision_id: number
  digest: string
  status: string
  created_at: string
  images: string[]
  helm_revision?: number
}

export interface RollbackService {
  service_id: number
  name: string
  provider: string
  versions: DeploymentVersion[]
}

export interface RollbackPreview {
  token: string
  origin_run_id: number
  workflow_id: number
  environment_id: number
  cluster_id: number
  namespace: string
  service_id: number
  service_name: string
  requires_approval: boolean
  current: DeploymentVersion
  target: DeploymentVersion
  current_manifest: string
  target_manifest: string
  blocked_reason: string
  warnings: string[]
}

export function getRollbackOptionsApi(runId: number) {
  return request.get<RollbackService[]>({
    url: `/cicd/v1/workflow-runs/${runId}/rollback-options`,
    showErrorMessage: false
  })
}

export function previewRollbackApi(runId: number, serviceId: number, snapshotId: number) {
  return request.get<RollbackPreview>({
    url: `/cicd/v1/workflow-runs/${runId}/rollback-preview`,
    params: { service_id: serviceId, snapshot_id: snapshotId },
    showErrorMessage: false
  })
}

export function submitRollbackApi(
  snapshotId: number,
  data: { origin_run_id: number; service_id: number; expected_token: string },
  requestKey: string
) {
  return request.post<{ workflow_run_id: number }>({
    url: `/cicd/v1/deployment-snapshots/${snapshotId}/rollback`,
    data,
    headers: { 'Idempotency-Key': requestKey },
    showErrorMessage: false
  })
}

export function cancelWorkflowRunApi(id: number) {
  return request.post<{ accepted: boolean }>({ url: `/cicd/v1/workflow-runs/${id}/cancel` })
}

export type DeploymentVerification = {
  id: number
  snapshot_id: number
  environment_id: number
  service_id: number
  build_id: number
  image_ref: string
  manifest_digest: string
  status: 'passed' | 'revoked'
  method: string
  note: string
  username: string
  revision: number
  verified_at: string
}
export type VerificationOption = {
  service_id: number
  name: string
  snapshot_id: number
  digest: string
  image_ref: string
  verification?: DeploymentVerification
  blocked_reason: string
}
export type ProductionCandidate = Omit<DeploymentVerification, 'status'> & {
  status: 'passed' | 'revoked' | ''
  name: string
  run_id: number
  workflow_id: number
  run_number: number
  build_number: number
  commit: string
  branch: string
  recommended: boolean
  built_at?: string
  commit_url?: string
  compare_url?: string
  source_name?: string
  deployment_status?: string
}
export type ProductionPolicy = {
  environment_id: number
  source_environment_id: number
  revision: number
}
export type ProductionOptions = {
  policy: ProductionPolicy
  source_name: string
  environment_id: number
  environment_name: string
  cluster_id: number
  namespace: string
  sources: { id: number; name: string }[]
  image_sources: { id: number; name: string; code?: string }[]
  services: {
    service_id: number
    name: string
    candidates: ProductionCandidate[]
    images: ProductionCandidate[]
  }[]
}
export type ProductionPreview = {
  token: string
  candidate: ProductionCandidate
  environment_id: number
  environment_name: string
  source_name: string
  cluster_id: number
  namespace: string
  current_image: string
  current_manifest: string
  target_manifest: string
  blocked_reason: string
  requires_approval: boolean
}
export type ProductionPick = { verification_id?: number; build_id?: number; expected_token: string }
export const getVerificationOptionsApi = (runId: number) =>
  request.get<VerificationOption[]>({
    url: `/cicd/v1/workflow-runs/${runId}/verification-options`,
    showErrorMessage: false
  })
export const saveVerificationApi = (
  runId: number,
  data: {
    snapshot_id: number
    expected_digest: string
    revision: number
    note: string
    action: 'pass' | 'revoke'
  }
) =>
  request.post<DeploymentVerification>({
    url: `/cicd/v1/workflow-runs/${runId}/verification`,
    data,
    showErrorMessage: false
  })
export const getProductionOptionsApi = (workflowId: number) =>
  request.get<ProductionOptions>({
    url: `/cicd/v1/workflows/${workflowId}/production-options`,
    showErrorMessage: false
  })
export const getProductionPreviewApi = (workflowId: number, verificationId: number) =>
  request.get<ProductionPreview>({
    url: `/cicd/v1/workflows/${workflowId}/production-preview`,
    params: { verification_id: verificationId },
    showErrorMessage: false
  })
export const getProductionImagePreviewApi = (workflowId: number, buildId: number) =>
  request.get<ProductionPreview>({
    url: `/cicd/v1/workflows/${workflowId}/production-preview`,
    params: { build_id: buildId },
    showErrorMessage: false
  })
export const saveProductionPolicyApi = (environmentId: number, data: ProductionPolicy) =>
  request.put<ProductionPolicy>({
    url: `/cicd/v1/project-environments/${environmentId}/production-policy`,
    data,
    showErrorMessage: false
  })
export const getProductionPolicyApi = (environmentId: number) =>
  request.get<ProductionPolicy>({
    url: `/cicd/v1/project-environments/${environmentId}/production-policy`,
    showErrorMessage: false
  })
export const submitProductionApi = (workflowId: number, picks: ProductionPick[], key: string) =>
  request.post<{ workflow_run_id: number }>({
    url: `/cicd/v1/workflows/${workflowId}/production-runs`,
    data: { picks },
    headers: { 'Idempotency-Key': key },
    showErrorMessage: false
  })

export function approveWorkflowRunApi(id: number) {
  return request.post<{ approved: boolean }>({ url: `/cicd/v1/workflow-runs/${id}/approve` })
}

export function getBuildRunApi(id: number) {
  return request.get<AxiOpsBuildRun & { stages_parsed?: unknown[]; logs?: AxiOpsBuildLog[] }>({
    url: `/cicd/v1/builds/${id}`,
    // A workflow can expose a build task before the build row is persisted;
    // the run drawer handles that race without surfacing a duplicate toast.
    showErrorMessage: false
  })
}

export function getBuildLogsApi(id: number) {
  return request.get<AxiOpsBuildLog[]>({
    url: `/cicd/v1/builds/${id}/logs`,
    showErrorMessage: false
  })
}

export function createWorkflowApi(projectId: number, data: CreateWorkflowInput) {
  return request.post<{ id: number }>({ url: `/cicd/v1/projects/${projectId}/workflows`, data })
}

export function listBuildRunsApi(
  projectId: number,
  options: {
    silent?: boolean
    service_id?: number
    status?: string
    page?: number
    page_size?: number
  } = {}
) {
  return request.get<PageResult<AxiOpsBuildRun>>({
    url: '/cicd/v1/builds',
    params: {
      project_id: projectId,
      page: options.page || 1,
      page_size: options.page_size || 200,
      service_id: options.service_id,
      status: options.status
    },
    showErrorMessage: options.silent !== true
  })
}

export function listNotificationSubscriptionsApi(projectId: number) {
  return request.get<AxiOpsNotificationSubscription[]>({
    url: '/cicd/v1/notification-subscriptions',
    params: { project_id: projectId }
  })
}

export function createNotificationSubscriptionApi(
  projectId: number,
  data: CreateNotificationSubscriptionInput
) {
  return request.post<{ id: number }>({
    url: `/cicd/v1/projects/${projectId}/notification-subscriptions`,
    data
  })
}

export function deleteNotificationSubscriptionApi(id: number, projectId: number) {
  return request.del<void>({
    url: `/cicd/v1/notification-subscriptions/${id}`,
    params: { project_id: projectId }
  })
}

export function listNotificationDeliveriesApi(
  projectId: number,
  params: { status?: string; page?: number; page_size?: number } = {}
) {
  return request.get<PageResult<AxiOpsNotificationDelivery>>({
    url: '/cicd/v1/notification-deliveries',
    params: { project_id: projectId, page: 1, page_size: 20, ...params }
  })
}
