import request from '@/utils/http'

const PROJECT_BASE = '/portal/v1/project'

// ========== 类型定义 ==========

/** 项目信息 */
export interface PortalProject {
  id: number
  name: string
  uuid: string
  isSystem: number
  description: string
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

/** 创建项目请求 */
export interface CreateProjectReq {
  name: string
  description?: string
  createdBy?: string
}

/** 更新项目请求 */
export interface UpdateProjectReq {
  id: number
  name: string
  description?: string
  updatedBy?: string
}

/** 删除项目请求 */
export interface DeleteProjectReq {
  id: number
  updatedBy?: string
}

/** 搜索项目请求 */
export interface ListProjectsReq {
  page?: number
  pageSize?: number
  name?: string
  uuid?: string
  isSystem?: number
  userId?: number
  platformId?: number
}

/** 搜索项目响应 */
export interface ListProjectsResp {
  data: PortalProject[]
  total: number
}

export interface PortalProjectMember {
  id: number
  projectId: number
  userId: number
  username: string
  nickname: string
  role: string
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
  platformRoles: PortalProjectMemberPlatformRole[]
}

export interface PortalProjectMemberInput {
  userId: number
  role?: string
  platformRoles?: PortalProjectMemberPlatformRoleInput[]
}

export interface PortalProjectMemberPlatformRole {
  platformId: number
  platformCode: string
  platformName: string
  role: string
}

export interface PortalProjectMemberPlatformRoleInput {
  platformId: number
  role?: string
}

export interface ProjectPlatformBinding {
  id: number
  projectId: number
  platformId: number
  platformCode: string
  platformName: string
  createdBy: string
  createdAt: number
}

// ========== API 方法 ==========

/** 创建项目 */
export function createProjectApi(data: CreateProjectReq) {
  return request.post<{ id: number; uuid: string }>({
    url: `${PROJECT_BASE}/create`,
    data
  })
}

/** 更新项目 */
export function updateProjectApi(data: UpdateProjectReq) {
  return request.put<Record<string, never>>({
    url: `${PROJECT_BASE}/update/${data.id}`,
    data
  })
}

/** 删除项目 */
export function deleteProjectApi(data: DeleteProjectReq) {
  return request.del<Record<string, never>>({
    url: `${PROJECT_BASE}/delete/${data.id}`
  })
}

/** 获取项目详情 */
export function getProjectApi(id: number) {
  return request.get<{ data: PortalProject }>({
    url: `${PROJECT_BASE}/get/${id}`
  })
}

/** 搜索项目列表 */
export function listProjectsApi(params: ListProjectsReq) {
  return request.get<ListProjectsResp>({
    url: `${PROJECT_BASE}/list`,
    params
  })
}

/** 批量获取项目 */
export function batchGetProjectsApi(ids: number[]) {
  return request.get<ListProjectsResp>({
    url: `${PROJECT_BASE}/batch`,
    params: { ids: ids.join(',') }
  })
}

export function setProjectMembersApi(projectId: number, members: PortalProjectMemberInput[]) {
  return request.put<Record<string, never>>({
    url: `${PROJECT_BASE}/${projectId}/members`,
    data: { members }
  })
}

export function listProjectMembersApi(projectId: number) {
  return request.get<{ data: PortalProjectMember[] }>({
    url: `${PROJECT_BASE}/${projectId}/members`
  })
}

export function bindProjectPlatformApi(projectId: number, platformId: number) {
  return request.post<Record<string, never>>({
    url: `${PROJECT_BASE}/${projectId}/platforms`,
    data: { platformId }
  })
}

export function unbindProjectPlatformApi(projectId: number, platformId: number) {
  return request.del<Record<string, never>>({
    url: `${PROJECT_BASE}/${projectId}/platforms/${platformId}`
  })
}

export function getProjectPlatformsApi(projectId: number) {
  return request.get<{ data: ProjectPlatformBinding[] }>({
    url: `${PROJECT_BASE}/${projectId}/platforms`
  })
}

export function getProjectsByPlatformApi(platformId: number) {
  return request.get<ListProjectsResp>({
    url: `${PROJECT_BASE}/platform/${platformId}/projects`
  })
}
