import request from '@/utils/http'

export interface AxiOpsCurrentUser {
  id: number
  username: string
  nickname: string
  email: string
  avatar: string
  role_ids: number[]
  roles: string[]
}

export interface AxiOpsLoginResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  user_info: AxiOpsCurrentUser
}

export interface AxiOpsRefreshResponse {
  access_token: string
  expires_in: number
}

export function loginAxiOpsApi(data: { username: string; password: string }) {
  return request.post<AxiOpsLoginResponse>({ url: '/platform/v1/auth/login', data })
}

export function refreshAxiOpsTokenApi(refresh_token: string) {
  return request.post<AxiOpsRefreshResponse>({
    url: '/platform/v1/auth/refresh',
    data: { refresh_token }
  })
}

export function getCurrentAxiOpsUserApi() {
  return request.get<AxiOpsCurrentUser>({ url: '/platform/v1/auth/current-user' })
}

export function logoutAxiOpsApi() {
  return request.post<void>({
    url: '/platform/v1/auth/logout',
    showErrorMessage: false
  })
}
