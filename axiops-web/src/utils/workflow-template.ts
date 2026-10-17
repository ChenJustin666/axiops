import type { AxiOpsService, DeliveryType, WorkflowNode } from '@/api/axiops/cicd'
import type { DeliveryTemplate } from '@/api/axiops/templates'

export const isDeploymentNode = (type: string) =>
  ['yaml_deploy', 'helm_deploy', 'host_deploy'].includes(type)
export const nodeNeedsService = (type: string) =>
  ['build', 'test'].includes(type) || isDeploymentNode(type)

export function readWorkflowTemplate(template: DeliveryTemplate) {
  const file = template.files.find((item) => item.path === 'workflow.json')
  if (!file) throw new Error('模板缺少 workflow.json')
  const definition = JSON.parse(file.content) as {
    delivery_type: DeliveryType
    nodes: WorkflowNode[]
  }
  if (!Array.isArray(definition.nodes) || !definition.nodes.length)
    throw new Error('模板没有可执行节点')
  return definition
}

export function instantiateWorkflowTemplate(
  template: DeliveryTemplate,
  delivery: DeliveryType,
  service?: AxiOpsService,
  bindings: Record<string, AxiOpsService> = {}
) {
  const definition = readWorkflowTemplate(template)
  const deploying = definition.nodes.some((node) => isDeploymentNode(node.type))
  if (deploying && definition.delivery_type !== delivery)
    throw new Error('模板与项目交付类型不匹配')
  const nodes: WorkflowNode[] = definition.nodes.map((node) => {
    const config = { ...(node.config || {}) }
    const target = bindings[node.key] || service
    if (nodeNeedsService(node.type)) {
      if (!target) throw new Error(`节点 ${node.name} 请选择服务`)
      config.service_id = target.id
      if (['build', 'test'].includes(node.type) && !config.branch)
        config.branch = target.repo_branch
      if (isDeploymentNode(node.type)) {
        if (!target.current_revision_id)
          throw new Error(`请先保存 ${target.name || target.code} 的服务版本`)
        config.service_revision_id = target.current_revision_id
      }
      if (node.type === 'helm_deploy') config.release_name = target.code
    }
    return { ...node, config, depends_on: [...(node.depends_on || [])] }
  })
  return { delivery_type: delivery, nodes }
}
