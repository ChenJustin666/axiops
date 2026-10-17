import request from '@/utils/http'

// 定义基础路径
const PLATFORM_BASE_PATH = '/portal/v1/platform'

// ========== 类型定义 ==========

/** 平台信息 */
export interface PlatformSysPlatform {
  id: number // 自增主键
  platformCode: string // 平台编码（唯一标识）
  platformName: string // 平台名称
  platformDesc: string // 平台描述
  platformIcon: string // 平台图标
  sort: number // 排序
  isEnable: number // 是否启用（0:禁用 1:启用）
  isDefault: number // 是否为默认平台（0:否 1:是）
  createdBy: string // 创建人
  updatedBy: string // 更新人
  createdAt: number // 创建时间
  updatedAt: number // 更新时间
}

/** 添加平台请求 */
export interface PlatformAddRequest {
  platformCode: string // 平台编码，必填
  platformName: string // 平台名称，必填
  platformDesc?: string // 平台描述
  platformIcon?: string // 平台图标
  sort?: number // 排序
  isEnable?: number // 是否启用
  isDefault?: number // 是否为默认平台
}

/** 更新平台请求 */
export interface PlatformUpdateRequest {
  id: number // 平台ID，必填
  platformCode?: string // 平台编码
  platformName?: string // 平台名称
  platformDesc?: string // 平台描述
  platformIcon?: string // 平台图标
  sort?: number // 排序
  isEnable?: number // 是否启用
  isDefault?: number // 是否为默认平台
}

/** 搜索平台请求 */
export interface PlatformSearchRequest {
  page?: number // 当前页码
  pageSize?: number // 每页条数
  orderStr?: string // 排序字段
  isAsc?: boolean // 是否升序
  platformCode?: string // 平台编码
  platformName?: string // 平台名称
  isEnable?: number // 是否启用
  isDefault?: number // 是否为默认平台
}

/** 搜索平台响应 */
export interface PlatformSearchResponse {
  items: PlatformSysPlatform[] // 平台列表
  total: number // 总数
}

/** 获取平台下用户请求 */
export interface PlatformGetUsersRequest {
  platformId: number // 平台ID
  page?: number // 当前页码
  pageSize?: number // 每页条数
}

/** 获取平台下用户响应 */
export interface PlatformGetUsersResponse {
  userIds: number[] // 用户ID列表
  total: number // 总数
}

/** 获取用户可访问平台请求 */
export interface PlatformGetUserPlatformsRequest {
  projectId?: number // 项目ID
}

// ========== API 接口 ==========

/**
 * 添加平台
 * @param data 添加平台请求参数
 * @returns 操作结果
 */
export async function addPlatformApi(data: PlatformAddRequest) {
  return request.post<string>({
    url: `${PLATFORM_BASE_PATH}`,
    data
  })
}

/**
 * 更新平台
 * @param data 更新平台请求参数
 * @returns 操作结果
 */
export async function updatePlatformApi(data: PlatformUpdateRequest) {
  const { id, ...params } = data
  return request.put<string>({
    url: `${PLATFORM_BASE_PATH}/${id}`,
    data: params
  })
}

/**
 * 删除平台
 * @param id 平台ID
 * @returns 操作结果
 */
export async function deletePlatformApi(id: number) {
  return request.del<string>({
    url: `${PLATFORM_BASE_PATH}/${id}`
  })
}

/**
 * 根据ID获取平台
 * @param id 平台ID
 * @returns 平台信息
 */
export async function getPlatformByIdApi(id: number) {
  return request.get<PlatformSysPlatform>({
    url: `${PLATFORM_BASE_PATH}/${id}`
  })
}

/**
 * 搜索平台（支持分页）
 * @param params 搜索参数
 * @returns 平台列表
 */
export async function searchPlatformApi(params?: PlatformSearchRequest) {
  return request.get<PlatformSearchResponse>({
    url: `${PLATFORM_BASE_PATH}/list`,
    params
  })
}

/**
 * 启用平台
 * @param id 平台ID
 * @returns 操作结果
 */
export async function enablePlatformApi(id: number) {
  return request.put<string>({
    url: `${PLATFORM_BASE_PATH}/enable/${id}`
  })
}

/**
 * 禁用平台
 * @param id 平台ID
 * @returns 操作结果
 */
export async function disablePlatformApi(id: number) {
  return request.put<string>({
    url: `${PLATFORM_BASE_PATH}/disable/${id}`
  })
}

/**
 * 设置为默认平台
 * @param id 平台ID
 * @returns 操作结果
 */
export async function setDefaultPlatformApi(id: number) {
  return request.put<string>({
    url: `${PLATFORM_BASE_PATH}/default/${id}`
  })
}

/**
 * 取消默认平台
 * @param id 平台ID
 * @returns 操作结果
 */
export async function unsetDefaultPlatformApi(id: number) {
  return request.put<string>({
    url: `${PLATFORM_BASE_PATH}/undefault/${id}`
  })
}

/**
 * 获取默认平台
 * @returns 默认平台信息
 */
export async function getDefaultPlatformApi() {
  return request.get<PlatformSysPlatform>({
    url: `${PLATFORM_BASE_PATH}/default`
  })
}

/**
 * 获取所有启用的平台
 * @returns 启用的平台列表
 */
export async function getAllEnabledPlatformsApi() {
  return request.get<PlatformSysPlatform[]>({
    url: `${PLATFORM_BASE_PATH}/enabled`
  })
}

/**
 * 获取用户可访问的平台列表
 * @param params 查询参数
 * @returns 用户可访问的平台列表
 */
export async function getUserPlatformsApi(params?: PlatformGetUserPlatformsRequest) {
  return request.get<PlatformSysPlatform[]>({
    url: `${PLATFORM_BASE_PATH}/user/platforms`,
    params
  })
}

/**
 * 获取平台下的用户列表
 * @param params 请求参数
 * @returns 用户列表
 */
export async function getPlatformUsersApi(params: PlatformGetUsersRequest) {
  return request.get<PlatformGetUsersResponse>({
    url: `${PLATFORM_BASE_PATH}/users`,
    params
  })
}
