import request from '@/utils/http'
import type { PageResult } from './cicd'

export interface AccessPolicy {
  resource: string
  action: string
  effect: 'allow' | 'deny'
  scope_type: 'platform' | 'project' | 'service' | 'build' | 'workflow' | 'cluster' | 'environment'
  scope_id: number
  environment: string
}

export interface AxiOpsRole {
  id: number
  name: string
  code: string
  description: string
  type: string
  binding_scope: 'platform' | 'scoped'
}

export interface RolePermission {
  id?: number
  role_id?: number
  resource: string
  action: string
  effect: 'allow' | 'deny'
}

export interface AxiOpsRoleDetail extends AxiOpsRole {
  permissions: RolePermission[]
  menu_ids: number[]
}

export interface AxiOpsMenu {
  id: number
  name: string
  code: string
  parent_id: number
  path: string
  icon: string
  sort_order: number
  type: number
}

export interface AxiOpsUser {
  id: number
  username: string
  nickname: string
  email: string
  status: number
  department_id: number
  role_ids: number[]
  role_names: string[]
}

export interface AxiOpsUserDetail extends AxiOpsUser {
  phone: string
  avatar: string
}

export interface CreateAxiOpsUserInput {
  username: string
  password: string
  nickname: string
  email: string
  phone: string
  department_id: number
  role_ids: number[]
}

export interface RoleBinding {
  id: number
  subject_type: 'user' | 'department'
  subject_id: number
  role_id: number
  scope_type: 'project' | 'service' | 'build' | 'workflow' | 'cluster' | 'environment'
  scope_id: number
  environment: string
  status: number
  expires_at?: string
  created_at: string
}

export interface CreateRoleBindingInput {
  subject_type: 'user' | 'department'
  subject_id: number
  role_id: number
  scope_type: RoleBinding['scope_type']
  scope_id: number
  environment?: string
  expires_at?: string
}

export interface AxiOpsDepartment {
  id: number
  name: string
  parent_id: number
  sort_order: number
  leader_user_id: number
  user_count: number
  created_at: string
  updated_at: string
}

export interface SaveDepartmentInput {
  name: string
  parent_id: number
  sort_order: number
  leader_user_id: number
}

export function getCurrentAccessApi() {
  return request.get<AccessPolicy[]>({ url: '/platform/v1/auth/access' })
}

export function getCurrentMenusApi() {
  return request.get<AxiOpsMenu[]>({ url: '/platform/v1/auth/menus' })
}

export function listRolesApi() {
  return request.get<AxiOpsRole[]>({ url: '/platform/v1/roles' })
}

export function getRoleApi(id: number) {
  return request.get<AxiOpsRoleDetail>({ url: `/platform/v1/roles/${id}` })
}

export function createRoleApi(data: {
  name: string
  code: string
  description: string
  binding_scope: 'scoped'
  permissions: RolePermission[]
  menu_ids: number[]
}) {
  return request.post<{ id: number }>({ url: '/platform/v1/roles', data })
}

export function updateRoleApi(
  id: number,
  data: {
    name: string
    description: string
    binding_scope: 'platform' | 'scoped'
    permissions: RolePermission[]
    menu_ids: number[]
  }
) {
  return request.put<void>({ url: `/platform/v1/roles/${id}`, data })
}

export function deleteRoleApi(id: number) {
  return request.del<void>({ url: `/platform/v1/roles/${id}` })
}

export function listAccessMenusApi() {
  return request.get<AxiOpsMenu[]>({ url: '/platform/v1/access/menus' })
}

export function listUsersApi(keyword = '', status = -1) {
  return request.get<PageResult<AxiOpsUser>>({
    url: '/platform/v1/users',
    params: { keyword, status, page: 1, page_size: 200 }
  })
}

export function getUserApi(id: number) {
  return request.get<AxiOpsUserDetail>({ url: `/platform/v1/users/${id}` })
}

export function createUserApi(data: CreateAxiOpsUserInput) {
  return request.post<{ id: number }>({ url: '/platform/v1/users', data })
}

export function updateUserApi(
  id: number,
  data: { nickname: string; email: string; phone: string; status: number; department_id: number }
) {
  return request.put<void>({ url: `/platform/v1/users/${id}`, data })
}

export function setUserRolesApi(id: number, role_ids: number[]) {
  return request.post<void>({ url: `/platform/v1/users/${id}/roles`, data: { role_ids } })
}

export function deleteUserApi(id: number) {
  return request.del<void>({ url: `/platform/v1/users/${id}` })
}

export function listRoleBindingsApi(params: { subject_type?: string; subject_id?: number } = {}) {
  return request.get<RoleBinding[]>({ url: '/platform/v1/access/bindings', params })
}

export function createRoleBindingApi(data: CreateRoleBindingInput) {
  return request.post<{ id: number }>({ url: '/platform/v1/access/bindings', data })
}

export function updateRoleBindingApi(
  id: number,
  data: { status?: number; expires_at?: string; clear_expires_at?: boolean }
) {
  return request.put<void>({ url: `/platform/v1/access/bindings/${id}`, data })
}

export function deleteRoleBindingApi(id: number) {
  return request.del<void>({ url: `/platform/v1/access/bindings/${id}` })
}

export function listDepartmentsApi() {
  return request.get<AxiOpsDepartment[]>({ url: '/platform/v1/departments' })
}

export function createDepartmentApi(data: SaveDepartmentInput) {
  return request.post<{ id: number }>({ url: '/platform/v1/departments', data })
}

export function updateDepartmentApi(id: number, data: SaveDepartmentInput) {
  return request.put<void>({ url: `/platform/v1/departments/${id}`, data })
}

export function deleteDepartmentApi(id: number) {
  return request.del<void>({ url: `/platform/v1/departments/${id}` })
}
