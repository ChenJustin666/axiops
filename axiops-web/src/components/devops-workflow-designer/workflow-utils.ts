import type { WorkflowNode } from './types'

const defaultStartNodeId = 'workflow-start'

function activeParallelBranchMap<T extends WorkflowNode>(items: T[], startNodeId: string) {
  const grouped = new Map<string, T[]>()
  items.forEach((item) => {
    const parentId = item.parentNodeId || startNodeId
    if (!grouped.has(parentId)) grouped.set(parentId, [])
    grouped.get(parentId)?.push(item)
  })

  const activeIds = new Set<string>()
  const parentById = new Map<string, string>()
  grouped.forEach((children, parentId) => {
    const parallelChildren = children.filter((child) => child.branchType === 'parallel')
    if (parallelChildren.length < 2) return
    parallelChildren.forEach((child) => {
      if (!child.id) return
      activeIds.add(child.id)
      parentById.set(child.id, parentId)
    })
  })

  return { activeIds, parentById }
}

function nodeSortValue(value?: number) {
  return typeof value === 'number' ? value : Number.MAX_SAFE_INTEGER
}

function compareWorkflowNode(left: WorkflowNode, right: WorkflowNode) {
  const sortDiff = nodeSortValue(left.sortOrder) - nodeSortValue(right.sortOrder)
  if (sortDiff !== 0) return sortDiff
  const xDiff = nodeSortValue(left.x) - nodeSortValue(right.x)
  if (xDiff !== 0) return xDiff
  return nodeSortValue(left.y) - nodeSortValue(right.y)
}

function createsCycle<T extends WorkflowNode>(
  byID: Map<string, T>,
  childId: string,
  nextParentId: string
) {
  let cursor = nextParentId
  const visited = new Set<string>()
  while (cursor && !visited.has(cursor)) {
    if (cursor === childId) return true
    visited.add(cursor)
    cursor = byID.get(cursor)?.parentNodeId || ''
  }
  return false
}

function linearizeNextChildren<T extends WorkflowNode>(items: T[], startNodeId: string): T[] {
  const result = items.map((item) => ({ ...item })) as T[]
  for (let pass = 0; pass < result.length; pass += 1) {
    const byID = new Map<string, T>()
    const grouped = new Map<string, T[]>()
    result.forEach((item) => {
      if (item.id) byID.set(item.id, item)
      const parentId = item.parentNodeId || startNodeId
      if (!grouped.has(parentId)) grouped.set(parentId, [])
      grouped.get(parentId)?.push(item)
    })

    let changed = false
    grouped.forEach((children) => {
      const nextChildren = children
        .filter((child) => child.id && child.branchType !== 'parallel')
        .sort(compareWorkflowNode)
      if (nextChildren.length < 2) return
      nextChildren.forEach((child, index) => {
        if (index === 0 || !child.id) return
        const previous = nextChildren[index - 1]
        if (!previous?.id || createsCycle(byID, child.id, previous.id)) return
        const current = byID.get(child.id)
        if (!current || current.parentNodeId === previous.id) return
        current.parentNodeId = previous.id
        current.branchType = 'next'
        changed = true
      })
    })
    if (!changed) break
  }
  return result
}

export function normalizeWorkflowBranches<T extends WorkflowNode>(
  items: T[],
  startNodeId = defaultStartNodeId
): T[] {
  const normalized = items.map((item) => ({
    ...item,
    branchType: item.branchType === 'parallel' ? 'parallel' : 'next'
  })) as T[]

  const active = activeParallelBranchMap(normalized, startNodeId)
  const lifted = normalized.map((item) => {
    const parentId = item.parentNodeId || startNodeId
    const parallelParentId = active.parentById.get(parentId)
    if (!parallelParentId || item.branchType === 'parallel') return item
    return {
      ...item,
      parentNodeId: parallelParentId,
      branchType: 'next'
    }
  }) as T[]

  const latest = activeParallelBranchMap(lifted, startNodeId)
  const cleanBranches = lifted.map((item) => ({
    ...item,
    branchType: item.id && latest.activeIds.has(item.id) ? 'parallel' : 'next'
  })) as T[]
  return linearizeNextChildren(cleanBranches, startNodeId)
}
