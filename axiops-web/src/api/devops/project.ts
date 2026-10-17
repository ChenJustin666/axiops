import request from '@/utils/http'

const DEVOPS_PROJECT_BASE = '/devops/v1/project'

export interface DevopsProject {
  id: string
  name: string
  code: string
  portalProjectUuid: string
  description: string
  pipelineEngineType?: string
  defaultEngineChannelId: string
  status: number
  extraConfig: string
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface DevopsProjectForm {
  name: string
  code?: string
  description?: string
  defaultEngineChannelId: string
  buildChannelIds: string[]
  status: number
  extraConfig?: string
}

export interface DevopsProjectSearchParams {
  page?: number
  pageSize?: number
  name?: string
  code?: string
  status?: number
}

export interface DevopsProjectListResponse {
  items: DevopsProject[]
  total: number
}

export interface DevopsProjectChannelBinding {
  id: string
  projectId: string
  channelId: string
  channelGroupCode: string
  channelName: string
  channelCode: string
  channelType: string
  usageScope: string
  isDefault: boolean
  allowUseGlobalCredential: boolean
  projectCredentialId: string
  bindingConfig: string
  status: number
  healthStatus: string
  lastCheckAt: number
  lastCheckMessage: string
  metadata: string
  channelConfig: string
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface BindDevopsProjectChannelsForm {
  channelGroupCode?: string
  channelIds: string[]
  defaultChannelId?: string
  usageScope?: string
  allowUseGlobalCredential: boolean
  projectCredentialId?: string
  bindingConfig?: string
}

export interface AddDevopsProjectChannelForm {
  channelId: string
  isDefault?: boolean
  allowUseGlobalCredential: boolean
  projectCredentialId?: string
  bindingConfig?: string
}

export interface UpdateDevopsProjectChannelBindingForm {
  isDefault?: boolean
  allowUseGlobalCredential: boolean
  projectCredentialId?: string
  bindingConfig?: string
  status: number
}

export interface DevopsProjectChannelBindingSearchParams {
  page?: number
  pageSize?: number
  projectId?: string
  channelId?: string
  channelGroupCode?: string
  channelType?: string
  usageScope?: string
  status?: number
}

export interface DevopsProjectChannelBindingListResponse {
  items: DevopsProjectChannelBinding[]
  total: number
}

export interface TektonSecretBindingOption {
  id: string
  projectId: string
  channelId: string
  channelName: string
  channelCode: string
  namespace: string
  taskNamespace: string
  serviceAccountName: string
  status: number
  healthStatus: string
}

export interface TektonSecret {
  name: string
  type: string
  namespace: string
  bindingId: string
  channelName: string
  keys: string[]
  data?: Record<string, string>
  managedByKubeNova: boolean
  createdAt: number
}

export interface TektonSecretListParams {
  projectId: string
  bindingId?: string
  keyword?: string
}

export interface TektonSecretBindingListResponse {
  items: TektonSecretBindingOption[]
  total: number
}

export interface TektonSecretListResponse {
  items: TektonSecret[]
  total: number
}

export interface TektonPVC {
  name: string
  namespace: string
  bindingId: string
  channelName: string
  storageClassName: string
  storage: string
  accessModes: string[]
  volumeMode: string
  status: string
  createdAt: number
}

export interface TektonPVCListParams {
  projectId: string
  bindingId?: string
  keyword?: string
}

export interface TektonPVCListResponse {
  items: TektonPVC[]
  total: number
}

export interface TektonResource {
  name: string
  namespace: string
  bindingId: string
  channelName: string
  resourceType: string
  kind: string
  apiVersion: string
  status: string
  phase: string
  reason: string
  message: string
  keys: string[]
  storageClassName: string
  storage: string
  accessModes: string[]
  volumeMode: string
  secrets: string[]
  imagePullSecrets: string[]
  containerNames: string[]
  createdAt: number
  yaml: string
}

export interface TektonResourceListParams {
  projectId: string
  bindingId?: string
  resourceType: string
  keyword?: string
}

export interface TektonResourceGetParams {
  projectId: string
  bindingId: string
  resourceType: string
  name: string
}

export interface TektonResourceForm {
  projectId: string
  bindingId: string
  resourceType: string
  name: string
  yaml: string
}

export interface TektonResourceDeleteParams {
  projectId: string
  bindingId: string
  resourceType: string
  name: string
}

export interface TektonResourceLogParams extends TektonResourceDeleteParams {
  container?: string
  tailLines?: number
}

export interface TektonResourceLogResponse {
  content: string
  containers: string[]
}

export interface TektonResourceActionParams extends TektonResourceDeleteParams {
  action: 'cancel'
}

export interface TektonSecretForm {
  projectId: string
  bindingId: string
  name: string
  type: string
  data: Record<string, string>
}

export interface TektonSecretSyncForm {
  projectId: string
  sourceBindingId: string
  name?: string
  targetBindingIds: string[]
}

export interface TestDevopsProjectChannelResponse {
  success: boolean
  healthStatus: string
  message: string
  checkedAt: number
  metadata: string
}

export interface DevopsProjectMember {
  id: string
  projectId: string
  userId: number
  username: string
  nickname: string
  role: string
  status: number
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface DevopsProjectMemberInput {
  userId: number
  username: string
  nickname?: string
  role: string
}

export interface SetDevopsProjectMembersForm {
  members: DevopsProjectMemberInput[]
}

export interface DevopsProjectMemberSearchParams {
  page?: number
  pageSize?: number
  projectId?: string
  userId?: number
  role?: string
  status?: number
}

export interface DevopsProjectMemberListResponse {
  items: DevopsProjectMember[]
  total: number
}

export interface DevopsProjectMavenConfig {
  id: string
  projectId: string
  name: string
  code: string
  content: string
  description: string
  status: number
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface DevopsProjectMavenConfigForm {
  projectId: string
  name: string
  code?: string
  content: string
  description?: string
  status: number
}

export interface DevopsProjectMavenConfigSearchParams {
  page?: number
  pageSize?: number
  projectId?: string
  name?: string
  code?: string
  status?: number
}

export interface DevopsProjectMavenConfigListResponse {
  items: DevopsProjectMavenConfig[]
  total: number
}

export interface DevopsConfigType {
  id: string
  parentId: string
  name: string
  code: string
  storageType: string
  description: string
  sortOrder: number
  status: number
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
  children?: DevopsConfigType[]
}

export interface DevopsConfigTypeForm {
  parentId?: string
  name: string
  code?: string
  storageType: string
  description?: string
  sortOrder: number
  status: number
}

export interface DevopsConfigTypeSearchParams {
  page?: number
  pageSize?: number
  parentId?: string
  name?: string
  code?: string
  status?: number
}

export interface DevopsConfigTypeListResponse {
  items: DevopsConfigType[]
  total: number
}

export interface DevopsConfigTypeTreeResponse {
  items: DevopsConfigType[]
}

export interface DevopsProjectConfig {
  id: string
  projectId: string
  typeId: string
  typeCode: string
  typeName: string
  name: string
  code: string
  content: string
  description: string
  status: number
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface DevopsProjectConfigForm {
  projectId: string
  typeId?: string
  typeCode?: string
  name: string
  code?: string
  content: string
  description?: string
  status: number
}

export interface DevopsProjectConfigSearchParams {
  page?: number
  pageSize?: number
  projectId?: string
  typeId?: string
  typeCode?: string
  name?: string
  code?: string
  status?: number
}

export interface DevopsProjectConfigListResponse {
  items: DevopsProjectConfig[]
  total: number
}

export function createDevopsProjectApi(data: DevopsProjectForm) {
  return request.post<{ id: string }>({
    url: DEVOPS_PROJECT_BASE,
    data
  })
}

export function updateDevopsProjectApi(id: string, data: DevopsProjectForm) {
  return request.put<string>({
    url: `${DEVOPS_PROJECT_BASE}/${id}`,
    data
  })
}

export function deleteDevopsProjectApi(id: string) {
  return request.del<string>({
    url: `${DEVOPS_PROJECT_BASE}/${id}`
  })
}

export function listDevopsProjectApi(params: DevopsProjectSearchParams) {
  return request.get<DevopsProjectListResponse>({
    url: `${DEVOPS_PROJECT_BASE}/list`,
    params
  })
}

export function getDevopsProjectApi(id: string) {
  return request.get<DevopsProject>({
    url: `${DEVOPS_PROJECT_BASE}/${id}`
  })
}

export function bindDevopsProjectChannelsApi(id: string, data: BindDevopsProjectChannelsForm) {
  return request.put<string>({
    url: `${DEVOPS_PROJECT_BASE}/${id}/channel`,
    data
  })
}

export function addDevopsProjectChannelApi(id: string, data: AddDevopsProjectChannelForm) {
  return request.post<string>({
    url: `${DEVOPS_PROJECT_BASE}/${id}/channel/item`,
    data
  })
}

export function updateDevopsProjectChannelBindingApi(
  id: string,
  data: UpdateDevopsProjectChannelBindingForm
) {
  return request.put<string>({
    url: `${DEVOPS_PROJECT_BASE}/channel/${id}`,
    data
  })
}

export function deleteDevopsProjectChannelBindingApi(id: string) {
  return request.del<string>({
    url: `${DEVOPS_PROJECT_BASE}/channel/${id}`
  })
}

export function listDevopsProjectChannelBindingApi(
  params: DevopsProjectChannelBindingSearchParams
) {
  return request.get<DevopsProjectChannelBindingListResponse>({
    url: `${DEVOPS_PROJECT_BASE}/channel/list`,
    params
  })
}

export function testDevopsProjectChannelBindingApi(id: string) {
  return request.post<TestDevopsProjectChannelResponse>({
    url: `${DEVOPS_PROJECT_BASE}/channel/${id}/test`
  })
}

export function listDevopsTektonSecretBindingApi(projectId: string) {
  return request.get<TektonSecretBindingListResponse>({
    url: `${DEVOPS_PROJECT_BASE}/tekton/secret/binding/list`,
    params: { projectId }
  })
}

export function listDevopsTektonSecretApi(params: TektonSecretListParams) {
  return request.get<TektonSecretListResponse>({
    url: `${DEVOPS_PROJECT_BASE}/tekton/secret/list`,
    params
  })
}

export function listDevopsTektonPvcApi(params: TektonPVCListParams) {
  return request.get<TektonPVCListResponse>({
    url: `${DEVOPS_PROJECT_BASE}/tekton/pvc/list`,
    params
  })
}

export function listDevopsTektonResourceApi(params: TektonResourceListParams) {
  return request.get<{ items: TektonResource[]; total: number }>({
    url: `${DEVOPS_PROJECT_BASE}/tekton/resource/list`,
    params
  })
}

export function getDevopsTektonResourceApi(params: TektonResourceGetParams) {
  return request.get<TektonResource>({
    url: `${DEVOPS_PROJECT_BASE}/tekton/resource/detail/${encodeURIComponent(params.name)}`,
    params: {
      projectId: params.projectId,
      bindingId: params.bindingId,
      resourceType: params.resourceType
    }
  })
}

export function saveDevopsTektonResourceApi(data: TektonResourceForm) {
  return request.post<string>({
    url: `${DEVOPS_PROJECT_BASE}/tekton/resource`,
    data
  })
}

export function deleteDevopsTektonResourceApi(params: TektonResourceDeleteParams) {
  return request.del<string>({
    url: `${DEVOPS_PROJECT_BASE}/tekton/resource/detail/${encodeURIComponent(params.name)}`,
    params: {
      projectId: params.projectId,
      bindingId: params.bindingId,
      resourceType: params.resourceType
    }
  })
}

export function describeDevopsTektonResourceApi(params: TektonResourceDeleteParams) {
  return request.get<{ content: string }>({
    url: `${DEVOPS_PROJECT_BASE}/tekton/resource/describe/${encodeURIComponent(params.name)}`,
    params: {
      projectId: params.projectId,
      bindingId: params.bindingId,
      resourceType: params.resourceType
    }
  })
}

export function logDevopsTektonResourceApi(params: TektonResourceLogParams) {
  return request.get<TektonResourceLogResponse>({
    url: `${DEVOPS_PROJECT_BASE}/tekton/resource/log/${encodeURIComponent(params.name)}`,
    params: {
      projectId: params.projectId,
      bindingId: params.bindingId,
      resourceType: params.resourceType,
      container: params.container,
      tailLines: params.tailLines
    }
  })
}

export function actionDevopsTektonResourceApi(params: TektonResourceActionParams) {
  return request.post<string>({
    url: `${DEVOPS_PROJECT_BASE}/tekton/resource/action/${encodeURIComponent(params.name)}`,
    params: {
      projectId: params.projectId,
      bindingId: params.bindingId,
      resourceType: params.resourceType,
      action: params.action
    }
  })
}

export function getDevopsTektonSecretApi(params: {
  projectId: string
  bindingId: string
  name: string
}) {
  return request.get<TektonSecret>({
    url: `${DEVOPS_PROJECT_BASE}/tekton/secret/detail/${encodeURIComponent(params.name)}`,
    params: {
      projectId: params.projectId,
      bindingId: params.bindingId
    }
  })
}

export function createDevopsTektonSecretApi(data: TektonSecretForm) {
  return request.post<string>({
    url: `${DEVOPS_PROJECT_BASE}/tekton/secret`,
    data
  })
}

export function updateDevopsTektonSecretApi(name: string, data: TektonSecretForm) {
  return request.put<string>({
    url: `${DEVOPS_PROJECT_BASE}/tekton/secret/detail/${encodeURIComponent(name)}`,
    data
  })
}

export function deleteDevopsTektonSecretApi(params: {
  projectId: string
  bindingId: string
  name: string
}) {
  return request.del<string>({
    url: `${DEVOPS_PROJECT_BASE}/tekton/secret/detail/${encodeURIComponent(params.name)}`,
    params: {
      projectId: params.projectId,
      bindingId: params.bindingId
    }
  })
}

export function syncDevopsTektonSecretApi(name: string, data: TektonSecretSyncForm) {
  return request.post<string>({
    url: `${DEVOPS_PROJECT_BASE}/tekton/secret/detail/${encodeURIComponent(name)}/sync`,
    data
  })
}

export function setDevopsProjectMembersApi(id: string, data: SetDevopsProjectMembersForm) {
  return request.put<string>({
    url: `${DEVOPS_PROJECT_BASE}/${id}/member`,
    data
  })
}

export function deleteDevopsProjectMemberApi(id: string) {
  return request.del<string>({
    url: `${DEVOPS_PROJECT_BASE}/member/${id}`
  })
}

export function listDevopsProjectMemberApi(params: DevopsProjectMemberSearchParams) {
  return request.get<DevopsProjectMemberListResponse>({
    url: `${DEVOPS_PROJECT_BASE}/member/list`,
    params
  })
}

export function createDevopsProjectMavenConfigApi(data: DevopsProjectMavenConfigForm) {
  return request.post<{ id: string }>({
    url: `${DEVOPS_PROJECT_BASE}/maven-config`,
    data
  })
}

export function updateDevopsProjectMavenConfigApi(id: string, data: DevopsProjectMavenConfigForm) {
  return request.put<string>({
    url: `${DEVOPS_PROJECT_BASE}/maven-config/${id}`,
    data
  })
}

export function deleteDevopsProjectMavenConfigApi(id: string) {
  return request.del<string>({
    url: `${DEVOPS_PROJECT_BASE}/maven-config/${id}`
  })
}

export function listDevopsProjectMavenConfigApi(params: DevopsProjectMavenConfigSearchParams) {
  return request.get<DevopsProjectMavenConfigListResponse>({
    url: `${DEVOPS_PROJECT_BASE}/maven-config/list`,
    params
  })
}

export function getDevopsProjectMavenConfigApi(id: string) {
  return request.get<DevopsProjectMavenConfig>({
    url: `${DEVOPS_PROJECT_BASE}/maven-config/${id}`
  })
}

export function createDevopsConfigTypeApi(data: DevopsConfigTypeForm) {
  return request.post<{ id: string }>({
    url: `/devops/v1/config/type`,
    data
  })
}

export function updateDevopsConfigTypeApi(id: string, data: DevopsConfigTypeForm) {
  return request.put<string>({
    url: `/devops/v1/config/type/${id}`,
    data
  })
}

export function deleteDevopsConfigTypeApi(id: string) {
  return request.del<string>({
    url: `/devops/v1/config/type/${id}`
  })
}

export function listDevopsConfigTypeApi(params: DevopsConfigTypeSearchParams) {
  return request.get<DevopsConfigTypeListResponse>({
    url: `/devops/v1/config/type/list`,
    params
  })
}

export function treeDevopsConfigTypeApi(params?: { status?: number }) {
  return request.get<DevopsConfigTypeTreeResponse>({
    url: `/devops/v1/config/type/tree`,
    params
  })
}

export function getDevopsConfigTypeApi(id: string) {
  return request.get<DevopsConfigType>({
    url: `/devops/v1/config/type/${id}`
  })
}

export function createDevopsProjectConfigApi(data: DevopsProjectConfigForm) {
  return request.post<{ id: string }>({
    url: `/devops/v1/config/project`,
    data
  })
}

export function updateDevopsProjectConfigApi(id: string, data: DevopsProjectConfigForm) {
  return request.put<string>({
    url: `/devops/v1/config/project/${id}`,
    data
  })
}

export function deleteDevopsProjectConfigApi(id: string) {
  return request.del<string>({
    url: `/devops/v1/config/project/${id}`
  })
}

export function listDevopsProjectConfigApi(params: DevopsProjectConfigSearchParams) {
  return request.get<DevopsProjectConfigListResponse>({
    url: `/devops/v1/config/project/list`,
    params
  })
}

export function getDevopsProjectConfigApi(id: string) {
  return request.get<DevopsProjectConfig>({
    url: `/devops/v1/config/project/${id}`
  })
}
