import request from '@/utils/http'

const DEVOPS_CHANNEL_BASE = '/devops/v1/channel'
const DEVOPS_CHANNEL_GROUP_BASE = '/devops/v1/channel/group'
const DEVOPS_CHANNEL_TYPE_BASE = '/devops/v1/channel/type'
const DEVOPS_CREDENTIAL_BASE = '/devops/v1/credential'
const DEVOPS_HOST_BASE = '/devops/v1/host'
const DEVOPS_JENKINS_AGENT_BASE = '/devops/v1/pipeline/jenkins-agent'

export interface DevopsChannelGroup {
  id: string
  name: string
  code: string
  description: string
  sortOrder: number
  isSystem: boolean
  status: number
  groupType: string
  allowedChannelTypes: string[]
  icon: string
  iconColor: string
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface DevopsChannelGroupForm {
  name: string
  code?: string
  description?: string
  sortOrder: number
  isSystem?: boolean
  status: number
  groupType?: string
  allowedChannelTypes?: string[]
  icon?: string
  iconColor?: string
}

export interface DevopsChannelType {
  id: string
  code: string
  name: string
  groupCode: string
  credentialTypes: string[]
  configSchema: string
  testStrategy: string
  icon: string
  iconColor: string
  connectionMode: string
  metadataStrategy: string
  mappingFields: string
  isSystem: boolean
  status: number
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface DevopsChannel {
  id: string
  groupId: string
  name: string
  code: string
  channelType: string
  endpoint: string
  description: string
  globalCredentialId: string
  credentialId: string
  config: string
  labels: string
  healthStatus: string
  lastCheckAt: number
  lastCheckMessage: string
  metadata: string
  status: number
  isSystem: boolean
  authType: string
  username: string
  password: string
  token: string
  insecureSkipTls: boolean
  icon: string
  iconColor: string
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface DevopsChannelForm {
  groupId: string
  name: string
  code?: string
  channelType: string
  endpoint: string
  description?: string
  globalCredentialId?: string
  credentialId?: string
  config?: string
  labels?: string
  authType: string
  username?: string
  password?: string
  token?: string
  insecureSkipTls: boolean
  icon?: string
  iconColor?: string
  status: number
}

export interface DevopsCredential {
  id: string
  name: string
  code: string
  credentialType: string
  username: string
  password: string
  token: string
  privateKey: string
  passphrase: string
  kubeconfig: string
  secretText: string
  certificate: string
  jsonData: string
  description: string
  status: number
  isSystem: boolean
  scope: string
  projectId: string
  channelGroupCode: string
  channelType: string
  jenkinsSyncStatus: string
  jenkinsSyncMessage: string
  jenkinsCredentialId: string
  jenkinsLastSyncAt: number
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface DevopsCredentialUsage {
  resourceType: string
  resourceName: string
  resourceCode: string
  resourceId: string
  relation: string
}

export interface DevopsCredentialForm {
  name: string
  code?: string
  credentialType: string
  username?: string
  password?: string
  token?: string
  privateKey?: string
  passphrase?: string
  kubeconfig?: string
  secretText?: string
  certificate?: string
  jsonData?: string
  description?: string
  status: number
  isSystem?: boolean
  scope?: string
  projectId?: string
  channelGroupCode?: string
  channelType?: string
}

export interface DevopsHost {
  id: string
  name: string
  ip: string
  port: number
  credentialId: string
  labels: string
  description: string
  healthStatus: string
  lastCheckAt: number
  lastCheckMessage: string
  metadata: string
  status: number
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface DevopsJenkinsAgentContainer {
  name: string
  image?: string
}

export interface DevopsJenkinsAgent {
  id: string
  channelId: string
  channelName: string
  name: string
  code: string
  agentType: 'static' | 'dynamic' | 'pod'
  matchMode: 'name' | 'label'
  matchValue: string
  cloud: string
  podYaml: string
  containers: DevopsJenkinsAgentContainer[]
  status: number
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface DevopsJenkinsAgentForm {
  channelId: string
  name: string
  code?: string
  agentType: 'static' | 'dynamic'
  matchMode: 'name' | 'label'
  matchValue?: string
  cloud?: string
  podYaml?: string
  containers?: DevopsJenkinsAgentContainer[]
  status: number
}

export interface DevopsJenkinsAgentSearchParams {
  page?: number
  pageSize?: number
  channelId?: string
  buildChannelBindingId?: string
  name?: string
  agentType?: string
  status?: number
}

export interface DevopsJenkinsAgentListResponse {
  items: DevopsJenkinsAgent[]
  total: number
}

export interface DevopsHostForm {
  name: string
  ip: string
  port: number
  credentialId?: string
  labels?: string
  description?: string
  status: number
}

export interface DevopsChannelGroupSearchParams {
  page?: number
  pageSize?: number
  name?: string
  code?: string
  status?: number
  groupType?: string
}

export interface DevopsChannelSearchParams {
  page?: number
  pageSize?: number
  name?: string
  code?: string
  groupId?: string
  channelType?: string
  status?: number
}

export interface DevopsChannelTypeForm {
  code: string
  name: string
  groupCode: string
  credentialTypes?: string[]
  configSchema?: string
  testStrategy?: string
  icon?: string
  iconColor?: string
  connectionMode?: string
  metadataStrategy?: string
  mappingFields?: string
  isSystem?: boolean
  status: number
}

export interface DevopsChannelTypeSearchParams {
  page?: number
  pageSize?: number
  name?: string
  code?: string
  groupCode?: string
  status?: number
}

export interface DevopsCredentialSearchParams {
  page?: number
  pageSize?: number
  name?: string
  code?: string
  credentialType?: string
  status?: number
  scope?: string
  projectId?: string
  buildChannelBindingId?: string
  channelGroupCode?: string
  channelType?: string
}

export interface DevopsHostSearchParams {
  page?: number
  pageSize?: number
  name?: string
  ip?: string
  status?: number
}

export interface DevopsChannelGroupListResponse {
  items: DevopsChannelGroup[]
  total: number
}

export interface DevopsChannelListResponse {
  items: DevopsChannel[]
  total: number
}

export interface DevopsChannelTypeListResponse {
  items: DevopsChannelType[]
  total: number
}

export interface DevopsCredentialListResponse {
  items: DevopsCredential[]
  total: number
}

export interface DevopsCredentialUsageResponse {
  items: DevopsCredentialUsage[]
  total: number
}

export interface DevopsHostListResponse {
  items: DevopsHost[]
  total: number
}

export interface TestDevopsChannelRequest {
  id?: string
  groupId?: string
  name?: string
  code?: string
  channelType: string
  endpoint: string
  authType: string
  username?: string
  password?: string
  token?: string
  insecureSkipTls: boolean
  config?: string
  credentialId?: string
}

export interface TestDevopsHostRequest {
  id?: string
  name?: string
  ip: string
  port: number
  credentialId?: string
}

export interface TestDevopsChannelResponse {
  success: boolean
  healthStatus: string
  message: string
  checkedAt: number
  metadata: string
}

export function createDevopsChannelGroupApi(data: DevopsChannelGroupForm) {
  return request.post<{ id: string }>({
    url: DEVOPS_CHANNEL_GROUP_BASE,
    data
  })
}

export function updateDevopsChannelGroupApi(id: string, data: DevopsChannelGroupForm) {
  return request.put<string>({
    url: `${DEVOPS_CHANNEL_GROUP_BASE}/${id}`,
    data
  })
}

export function deleteDevopsChannelGroupApi(id: string) {
  return request.del<string>({
    url: `${DEVOPS_CHANNEL_GROUP_BASE}/${id}`
  })
}

export function listDevopsChannelGroupApi(params: DevopsChannelGroupSearchParams = {}) {
  return request.get<DevopsChannelGroupListResponse>({
    url: `${DEVOPS_CHANNEL_GROUP_BASE}/list`,
    params
  })
}

export function listDevopsChannelTypeApi(params: DevopsChannelTypeSearchParams = {}) {
  return request.get<DevopsChannelTypeListResponse>({
    url: `${DEVOPS_CHANNEL_TYPE_BASE}/list`,
    params
  })
}

export function createDevopsChannelTypeApi(data: DevopsChannelTypeForm) {
  return request.post<{ id: string }>({
    url: DEVOPS_CHANNEL_TYPE_BASE,
    data
  })
}

export function updateDevopsChannelTypeApi(id: string, data: DevopsChannelTypeForm) {
  return request.put<string>({
    url: `${DEVOPS_CHANNEL_TYPE_BASE}/${id}`,
    data
  })
}

export function deleteDevopsChannelTypeApi(id: string) {
  return request.del<string>({
    url: `${DEVOPS_CHANNEL_TYPE_BASE}/${id}`
  })
}

export function getDevopsChannelTypeApi(id: string) {
  return request.get<DevopsChannelType>({
    url: `${DEVOPS_CHANNEL_TYPE_BASE}/${id}`
  })
}

export function createDevopsCredentialApi(data: DevopsCredentialForm) {
  return request.post<{ id: string }>({
    url: DEVOPS_CREDENTIAL_BASE,
    data
  })
}

export function updateDevopsCredentialApi(id: string, data: DevopsCredentialForm) {
  return request.put<string>({
    url: `${DEVOPS_CREDENTIAL_BASE}/${id}`,
    data
  })
}

export function deleteDevopsCredentialApi(id: string) {
  return request.del<string>({
    url: `${DEVOPS_CREDENTIAL_BASE}/${id}`
  })
}

export function listDevopsCredentialApi(params: DevopsCredentialSearchParams = {}) {
  return request.get<DevopsCredentialListResponse>({
    url: `${DEVOPS_CREDENTIAL_BASE}/list`,
    params
  })
}

export function getDevopsCredentialUsageApi(id: string) {
  return request.get<DevopsCredentialUsageResponse>({
    url: `${DEVOPS_CREDENTIAL_BASE}/${id}/usage`
  })
}

export function createDevopsHostApi(data: DevopsHostForm) {
  return request.post<{ id: string }>({
    url: DEVOPS_HOST_BASE,
    data
  })
}

export function updateDevopsHostApi(id: string, data: DevopsHostForm) {
  return request.put<string>({
    url: `${DEVOPS_HOST_BASE}/${id}`,
    data
  })
}

export function deleteDevopsHostApi(id: string) {
  return request.del<string>({
    url: `${DEVOPS_HOST_BASE}/${id}`
  })
}

export function listDevopsHostApi(params: DevopsHostSearchParams = {}) {
  return request.get<DevopsHostListResponse>({
    url: `${DEVOPS_HOST_BASE}/list`,
    params
  })
}

export function testDevopsHostApi(id: string) {
  return request.post<TestDevopsChannelResponse>({
    url: `${DEVOPS_HOST_BASE}/${id}/test`
  })
}

export function testDevopsHostConfigApi(data: TestDevopsHostRequest) {
  return request.post<TestDevopsChannelResponse>({
    url: `${DEVOPS_HOST_BASE}/test`,
    data
  })
}

export function createDevopsChannelApi(data: DevopsChannelForm) {
  return request.post<{ id: string }>({
    url: DEVOPS_CHANNEL_BASE,
    data
  })
}

export function updateDevopsChannelApi(id: string, data: DevopsChannelForm) {
  return request.put<string>({
    url: `${DEVOPS_CHANNEL_BASE}/${id}`,
    data
  })
}

export function deleteDevopsChannelApi(id: string) {
  return request.del<string>({
    url: `${DEVOPS_CHANNEL_BASE}/${id}`
  })
}

export function listDevopsChannelApi(params: DevopsChannelSearchParams) {
  return request.get<DevopsChannelListResponse>({
    url: `${DEVOPS_CHANNEL_BASE}/list`,
    params
  })
}

export function testDevopsChannelApi(data: TestDevopsChannelRequest) {
  return request.post<TestDevopsChannelResponse>({
    url: `${DEVOPS_CHANNEL_BASE}/test`,
    data
  })
}

export function testExistingDevopsChannelApi(id: string) {
  return request.post<TestDevopsChannelResponse>({
    url: `${DEVOPS_CHANNEL_BASE}/${id}/test`
  })
}

export function syncDevopsTektonChannelStepsApi(id: string) {
  return request.post<string>({
    url: `${DEVOPS_CHANNEL_BASE}/${id}/tekton/step/sync`
  })
}

export function createDevopsJenkinsAgentApi(data: DevopsJenkinsAgentForm) {
  return request.post<{ id: string }>({
    url: DEVOPS_JENKINS_AGENT_BASE,
    data
  })
}

export function updateDevopsJenkinsAgentApi(id: string, data: DevopsJenkinsAgentForm) {
  return request.put<string>({
    url: `${DEVOPS_JENKINS_AGENT_BASE}/${id}`,
    data
  })
}

export function deleteDevopsJenkinsAgentApi(id: string) {
  return request.del<string>({
    url: `${DEVOPS_JENKINS_AGENT_BASE}/${id}`
  })
}

export function listDevopsJenkinsAgentApi(params: DevopsJenkinsAgentSearchParams = {}) {
  return request.get<DevopsJenkinsAgentListResponse>({
    url: `${DEVOPS_JENKINS_AGENT_BASE}/list`,
    params
  })
}
