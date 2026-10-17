import request from '@/utils/http'

export type TemplateKind = 'k8s_yaml' | 'helm' | 'dockerfile' | 'build' | 'workflow'
export interface DeliveryTemplate {
  id: number
  kind: TemplateKind
  name: string
  description: string
  files: Array<{ path: string; content: string }>
  variables: Array<{
    key: string
    value: string
    description: string
    label?: string
    group?: string
  }>
  revision: number
  builtin_key?: string
  updated_at?: string
}
export const listDeliveryTemplatesApi = (kind?: string) =>
  request.get<DeliveryTemplate[]>({ url: '/cicd/v1/delivery-templates', params: { kind } })
export const saveDeliveryTemplateApi = (template: DeliveryTemplate) => {
  const { kind, name, description, files, variables, revision } = template
  const data = { kind, name, description, files, variables, revision }
  return template.id
    ? request.put<DeliveryTemplate>({ url: `/cicd/v1/delivery-templates/${template.id}`, data })
    : request.post<DeliveryTemplate>({ url: '/cicd/v1/delivery-templates', data })
}
export const previewDeliveryTemplateApi = (template: DeliveryTemplate) => {
  const { kind, name, description, files, variables } = template
  return request.post<{ definition: string }>({
    url: '/cicd/v1/delivery-templates/preview',
    data: { kind, name, description, files, variables }
  })
}
export const archiveDeliveryTemplateApi = (template: DeliveryTemplate) =>
  request.del<void>({
    url: `/cicd/v1/delivery-templates/${template.id}`,
    params: { revision: template.revision }
  })

export interface DeliveryTemplateVersion {
  revision: number
  name: string
  description: string
  content_hash: string
  created_at: string
}
export interface HelmTemplateStatus {
  mode: 'template' | 'custom' | 'unlinked'
  service_id: number
  revision_id: number
  template?: { id: number; revision: number }
  name?: string
  latest_revision?: number
  archived?: boolean
  available?: boolean
  update_available?: boolean
}
export interface DeliveryTemplateUsage {
  project_id: number
  service_id: number
  service_name: string
  environment_id: number
  environment_name: string
  revision: number
}
export const listDeliveryTemplateVersionsApi = (id: number) =>
  request.get<DeliveryTemplateVersion[]>({ url: `/cicd/v1/delivery-templates/${id}/versions` })
export const getDeliveryTemplateVersionApi = (id: number, revision: number) =>
  request.get<{ template: DeliveryTemplate; definition: string; content_hash: string }>({
    url: `/cicd/v1/delivery-templates/${id}/versions/${revision}`
  })
export const listDeliveryTemplateUsagesApi = (id: number) =>
  request.get<DeliveryTemplateUsage[]>({ url: `/cicd/v1/delivery-templates/${id}/usages` })
export const getHelmTemplateStatusApi = (id: number, revision_id: number) =>
  request.get<HelmTemplateStatus>({
    url: `/cicd/v1/services/${id}/helm-template`,
    params: { revision_id },
    showErrorMessage: false
  })
