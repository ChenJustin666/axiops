import request from '@/utils/http'

export interface BuildRegistry {
  id: number
  name: string
  type: string
  url: string
  namespaces?: string[]
  credential_id: number
  project_id: number
  status: string
}

export function listBuildRegistriesApi(projectId = 0) {
  return request.get<BuildRegistry[]>({
    url: '/cicd/v1/build-registries',
    params: { project_id: projectId }
  })
}

export function saveBuildRegistryApi(data: BuildRegistry) {
  return request.post<{ id: number }>({ url: '/cicd/v1/build-registries', data })
}

export function testBuildRegistryApi(
  data: Pick<BuildRegistry, 'url' | 'type' | 'namespaces' | 'credential_id'>
) {
  return request.post<{ status: 'verified' }>({ url: '/cicd/v1/build-registries/test', data })
}
