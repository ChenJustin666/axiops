import type { WorkflowNode } from '@/api/axiops/cicd'

const deployTypes = new Set(['yaml_deploy', 'helm_deploy', 'host_deploy'])
const serviceTypes = new Set(['build', 'test', ...deployTypes])

export function workflowDeploymentOrder(nodes: WorkflowNode[]): number[] {
  if (!nodes.length || nodes.some((node) => !serviceTypes.has(node.type))) return []
  const byKey = new Map(nodes.map((node) => [node.key, node]))
  if (byKey.size !== nodes.length) return []
  const serviceIds: number[] = []
  const deployByService = new Map<number, WorkflowNode>()
  const predecessors = new Map<number, Set<number>>()
  for (const node of nodes) {
    const id = Number(node.config?.service_id || 0)
    if (!Number.isSafeInteger(id) || id <= 0) return []
    if (!serviceIds.includes(id)) serviceIds.push(id)
    if (deployTypes.has(node.type)) {
      if (deployByService.has(id)) return []
      deployByService.set(id, node)
    }
  }
  if (serviceIds.length < 2 || serviceIds.some((id) => !deployByService.has(id))) return []
  const ranked = serviceIds.map((id) => Number(deployByService.get(id)?.config?.start_order || 0))
  if (ranked.some((rank) => rank > 0)) {
    if (
      new Set(ranked).size !== serviceIds.length ||
      ranked.some((rank) => !Number.isInteger(rank) || rank < 1 || rank > serviceIds.length)
    )
      return []
  }
  for (const node of nodes) {
    const targetId = Number(node.config?.service_id)
    for (const key of node.depends_on || []) {
      const source = byKey.get(key)
      if (!source) return []
      const sourceId = Number(source.config?.service_id)
      if (sourceId === targetId) continue
      if (!deployTypes.has(source.type) || !deployTypes.has(node.type)) return []
      if (!predecessors.has(targetId)) predecessors.set(targetId, new Set())
      predecessors.get(targetId)!.add(sourceId)
    }
  }
  if (ranked.some((rank) => rank > 0)) {
    return [...serviceIds].sort(
      (a, b) => ranked[serviceIds.indexOf(a)] - ranked[serviceIds.indexOf(b)]
    )
  }
  const order: number[] = []
  const remaining = new Set(serviceIds)
  while (remaining.size) {
    const next = serviceIds.find(
      (id) =>
        remaining.has(id) &&
        [...(predecessors.get(id) || [])].every((prior) => !remaining.has(prior))
    )
    if (!next) return []
    order.push(next)
    remaining.delete(next)
  }
  return order
}

export function hasWorkflowDeploymentSequence(nodes: WorkflowNode[]): boolean {
  const order = workflowDeploymentOrder(nodes)
  if (order.length < 2) return false
  const deployByService = new Map(
    nodes
      .filter((node) => deployTypes.has(node.type))
      .map((node) => [Number(node.config?.service_id), node])
  )
  return order.every(
    (id, index) => Number(deployByService.get(id)?.config?.start_order) === index + 1
  )
}

export function hasLegacyWorkflowDeploymentSequence(nodes: WorkflowNode[]): boolean {
  const order = workflowDeploymentOrder(nodes)
  if (order.length < 2 || hasWorkflowDeploymentSequence(nodes)) return false
  const deployByService = new Map(
    nodes
      .filter((node) => deployTypes.has(node.type))
      .map((node) => [Number(node.config?.service_id), node])
  )
  return order
    .slice(1)
    .some((id, index) =>
      (deployByService.get(id)?.depends_on || []).includes(deployByService.get(order[index])!.key)
    )
}

export function withWorkflowDeploymentOrder(
  nodes: WorkflowNode[],
  order: number[]
): WorkflowNode[] {
  const available = workflowDeploymentOrder(nodes)
  if (
    available.length !== order.length ||
    new Set(order).size !== order.length ||
    order.some((id) => !available.includes(id))
  ) {
    throw new Error('工作流服务无法安全调整部署顺序')
  }
  const byKey = new Map(nodes.map((node) => [node.key, node]))
  return nodes.map((node) => {
    if (!deployTypes.has(node.type)) return { ...node, depends_on: [...(node.depends_on || [])] }
    const serviceId = Number(node.config?.service_id)
    const index = order.indexOf(serviceId)
    const ownDependencies = (node.depends_on || []).filter(
      (key) => Number(byKey.get(key)?.config?.service_id) === serviceId
    )
    return {
      ...node,
      config: { ...node.config, start_order: index + 1 },
      depends_on: ownDependencies
    }
  })
}
