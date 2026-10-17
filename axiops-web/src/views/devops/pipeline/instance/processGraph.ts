import type { DevopsPipelineRunStage, PipelineTemplateStep } from '@/api/devops/pipeline'

export type ProcessGraphRootType = 'main' | 'post'

export type ProcessParallelGroupNodeData = {
  id: string
  name: string
  status: string
  rootType: ProcessGraphRootType
  x: number
  y: number
  width: number
  height: number
}

export type ProcessGraphLayoutItem = {
  stage: DevopsPipelineRunStage
  x: number
  y: number
  parallelGroup: string
  rootType: ProcessGraphRootType
}

export type ProcessGraphLayoutEdge = {
  id: string
  source: string
  target: string
  status: string
}

export type ProcessGraphLayout = {
  items: ProcessGraphLayoutItem[]
  edges: ProcessGraphLayoutEdge[]
  groups: ProcessParallelGroupNodeData[]
  parallelGroups: number
  width: number
  height: number
}

export const workflowStartNodeId = 'workflow-start'
export const workflowPostNodeId = 'workflow-post'

const processNodeWidth = 156
const processNodeHeight = 92
const processColumnGap = 236
const processRowGap = 136
const processGraphPaddingX = 54
const processGraphPaddingY = 54
const processParallelGroupPaddingX = 40
const processParallelGroupPaddingY = 40

export const processDependencyReadyStatuses = new Set([
  'success',
  'unstable',
  'skipped',
  'finished'
])

type NormalizeRunStatus = (status?: string) => string
type IsRunningStageStatus = (status?: string) => boolean

export function compareProcessStep(left: PipelineTemplateStep, right: PipelineTemplateStep) {
  const leftOrder = typeof left.sortOrder === 'number' ? left.sortOrder : Number.MAX_SAFE_INTEGER
  const rightOrder = typeof right.sortOrder === 'number' ? right.sortOrder : Number.MAX_SAFE_INTEGER
  if (leftOrder !== rightOrder) return leftOrder - rightOrder
  const leftX = typeof left.x === 'number' ? left.x : Number.MAX_SAFE_INTEGER
  const rightX = typeof right.x === 'number' ? right.x : Number.MAX_SAFE_INTEGER
  if (leftX !== rightX) return leftX - rightX
  const leftY = typeof left.y === 'number' ? left.y : Number.MAX_SAFE_INTEGER
  const rightY = typeof right.y === 'number' ? right.y : Number.MAX_SAFE_INTEGER
  return leftY - rightY
}

export function processStepRootType(
  step: PipelineTemplateStep,
  stepById: Map<string, PipelineTemplateStep>
): ProcessGraphRootType {
  let parentId = step.parentNodeId || workflowStartNodeId
  const visited = new Set<string>()
  while (parentId && parentId !== workflowStartNodeId) {
    if (parentId === workflowPostNodeId) return 'post'
    if (visited.has(parentId)) break
    visited.add(parentId)
    const parent = stepById.get(parentId)
    if (!parent) break
    parentId = parent.parentNodeId || workflowStartNodeId
  }
  return 'main'
}

function uniqueProcessNodeIds(items: string[]) {
  return Array.from(new Set(items.filter(Boolean)))
}

function buildProcessStageDependencyMap(
  stages: DevopsPipelineRunStage[],
  pipelineSteps: PipelineTemplateStep[]
) {
  const stageNodeIds = new Set(stages.map((stage) => stage.nodeId).filter(Boolean))
  const steps = pipelineSteps
    .filter((step) => step.id && step.enabled !== false && stageNodeIds.has(step.id))
    .sort(compareProcessStep)
  const dependencies = new Map<string, string[]>()
  if (steps.length === 0) return dependencies

  const childrenMap = new Map<string, PipelineTemplateStep[]>()
  steps.forEach((step) => {
    if (!step.id) return
    const parentId = step.parentNodeId || workflowStartNodeId
    if (!childrenMap.has(parentId)) childrenMap.set(parentId, [])
    childrenMap.get(parentId)?.push(step)
  })
  childrenMap.forEach((children) => children.sort(compareProcessStep))

  const placed = new Set<string>()
  const placeStep = (step: PipelineTemplateStep, sources: string[]) => {
    if (!step.id || placed.has(step.id)) return sources
    placed.add(step.id)
    dependencies.set(step.id, uniqueProcessNodeIds(sources))
    return [step.id]
  }
  const walk = (parentId: string, sources: string[], stack = new Set<string>()): string[] => {
    if (stack.has(parentId)) return sources
    const nextStack = new Set(stack)
    nextStack.add(parentId)
    const children = (childrenMap.get(parentId) || []).filter((child) => child.id)
    if (children.length === 0) return sources

    let currentSources = sources
    const parallelChildren = children.filter((child) => child.branchType === 'parallel')
    const nextChildren = children.filter((child) => child.branchType !== 'parallel')
    if (parallelChildren.length >= 2) {
      const parallelEndSources: string[] = []
      parallelChildren.forEach((child) => {
        const childSources = placeStep(child, sources)
        parallelEndSources.push(...walk(child.id || '', childSources, nextStack))
      })
      currentSources = uniqueProcessNodeIds(parallelEndSources)
    }
    nextChildren.forEach((child) => {
      const childSources = placeStep(child, currentSources)
      currentSources = walk(child.id || '', childSources, nextStack)
    })
    return currentSources
  }

  walk(workflowStartNodeId, [])
  walk(workflowPostNodeId, [])
  return dependencies
}

export function applyProcessStageDependencyGuard(params: {
  stages: DevopsPipelineRunStage[]
  pipelineSteps: PipelineTemplateStep[]
  runActive: boolean
  normalizeRunStatus: NormalizeRunStatus
}) {
  const { stages, pipelineSteps, runActive, normalizeRunStatus } = params
  if (!runActive || stages.length === 0) return stages
  const dependencies = buildProcessStageDependencyMap(stages, pipelineSteps)
  if (dependencies.size === 0) return stages

  const statusByNodeId = new Map(
    stages.map((stage) => [stage.nodeId, normalizeRunStatus(stage.status)])
  )
  return stages.map((stage) => {
    const deps = dependencies.get(stage.nodeId) || []
    const blocked = deps.some(
      (nodeId) => !processDependencyReadyStatuses.has(statusByNodeId.get(nodeId) || 'queued')
    )
    if (!blocked) {
      statusByNodeId.set(stage.nodeId, normalizeRunStatus(stage.status))
      return stage
    }
    const nextStage =
      normalizeRunStatus(stage.status) === 'queued'
        ? stage
        : {
            ...stage,
            status: 'queued',
            startedAt: '',
            finishedAt: '',
            durationSeconds: 0,
            jenkinsNodeId: ''
          }
    statusByNodeId.set(stage.nodeId, normalizeRunStatus(nextStage.status))
    return nextStage
  })
}

function sequentialProcessGraphLayout(
  stages: DevopsPipelineRunStage[],
  normalizeRunStatus: NormalizeRunStatus
): ProcessGraphLayout {
  const items = stages.map((stage, index) => ({
    stage,
    x: processGraphPaddingX + index * processColumnGap,
    y: processGraphPaddingY,
    parallelGroup: '',
    rootType: stage.stageType === 'post' ? ('post' as const) : ('main' as const)
  }))
  const edges = stages.slice(1).map((stage, index) => ({
    id: `process-edge-${stages[index].id}-${stage.id}`,
    source: stages[index].id,
    target: stage.id,
    status: normalizeRunStatus(stage.status)
  }))
  return {
    items,
    edges,
    groups: [],
    parallelGroups: 0,
    width: Math.max(1, items.length) * processColumnGap + processGraphPaddingX,
    height: processNodeHeight + processGraphPaddingY * 2
  }
}

export function buildProcessGraphLayout(params: {
  stages: DevopsPipelineRunStage[]
  pipelineSteps: PipelineTemplateStep[]
  runActive: boolean
  normalizeRunStatus: NormalizeRunStatus
  isRunningStageStatus: IsRunningStageStatus
}): ProcessGraphLayout {
  const { stages, pipelineSteps, runActive, normalizeRunStatus, isRunningStageStatus } = params
  const stageByNodeId = new Map<string, DevopsPipelineRunStage>()
  stages.forEach((stage) => {
    if (stage.nodeId) stageByNodeId.set(stage.nodeId, stage)
  })
  const steps = pipelineSteps
    .filter((step) => step.id && step.enabled !== false && stageByNodeId.has(step.id))
    .sort(compareProcessStep)
  if (steps.length === 0) return sequentialProcessGraphLayout(stages, normalizeRunStatus)

  const childrenMap = new Map<string, PipelineTemplateStep[]>()
  steps.forEach((step) => {
    if (!step.id) return
    const parentId = step.parentNodeId || workflowStartNodeId
    if (!childrenMap.has(parentId)) childrenMap.set(parentId, [])
    childrenMap.get(parentId)?.push(step)
  })
  childrenMap.forEach((children) => children.sort(compareProcessStep))

  const items: ProcessGraphLayoutItem[] = []
  const edges: ProcessGraphLayoutEdge[] = []
  const groups: ProcessParallelGroupNodeData[] = []
  const placed = new Set<string>()
  let parallelGroupSeq = 0
  let maxX = processGraphPaddingX
  let maxY = processGraphPaddingY

  const placeItem = (
    step: PipelineTemplateStep,
    x: number,
    y: number,
    parallelGroup: string,
    sources: string[],
    rootType: ProcessGraphRootType
  ): string[] => {
    if (!step.id || placed.has(step.id)) return sources
    const stage = stageByNodeId.get(step.id)
    if (!stage) return sources
    placed.add(step.id)
    items.push({ stage, x, y, parallelGroup, rootType })
    sources.forEach((source) => {
      if (!source) return
      edges.push({
        id: `process-edge-${source}-${stage.id}`,
        source,
        target: stage.id,
        status: normalizeRunStatus(stage.status)
      })
    })
    maxX = Math.max(maxX, x)
    maxY = Math.max(maxY, y)
    return [stage.id]
  }

  const layoutFromParent = (
    parentId: string,
    x: number,
    centerY: number,
    sources: string[],
    rootType: ProcessGraphRootType,
    activeGroup = '',
    stack = new Set<string>()
  ): { nextX: number; endSources: string[] } => {
    if (stack.has(parentId)) return { nextX: x, endSources: sources }
    const nextStack = new Set(stack)
    nextStack.add(parentId)
    const children = (childrenMap.get(parentId) || []).filter((child) => child.id)
    if (children.length === 0) return { nextX: x, endSources: sources }

    let cursorX = x
    let currentSources = sources
    const parallelChildren = children.filter((child) => child.branchType === 'parallel')
    const nextChildren = children.filter((child) => child.branchType !== 'parallel')
    const groupCenterY =
      parallelChildren.length >= 2
        ? Math.max(
            centerY,
            processGraphPaddingY + ((parallelChildren.length - 1) * processRowGap) / 2
          )
        : centerY

    if (parallelChildren.length >= 2) {
      parallelGroupSeq += 1
      const groupName = `并行组-${parallelGroupSeq}`
      const groupStartIndex = items.length
      const startY = groupCenterY - ((parallelChildren.length - 1) * processRowGap) / 2
      const parallelEndSources: string[] = []
      let branchNextX = cursorX + processColumnGap
      parallelChildren.forEach((child, index) => {
        const branchY = Math.max(processGraphPaddingY, startY + index * processRowGap)
        const childSources = placeItem(child, cursorX, branchY, groupName, sources, rootType)
        const childResult = layoutFromParent(
          child.id || '',
          cursorX + processColumnGap,
          branchY,
          childSources,
          rootType,
          groupName,
          nextStack
        )
        branchNextX = Math.max(branchNextX, childResult.nextX)
        parallelEndSources.push(...childResult.endSources)
      })
      const groupItems = items
        .slice(groupStartIndex)
        .filter((item) => item.parallelGroup === groupName)
      if (groupItems.length > 0) {
        const groupMinX = Math.min(...groupItems.map((item) => item.x))
        const groupMinY = Math.min(...groupItems.map((item) => item.y))
        const groupMaxX = Math.max(...groupItems.map((item) => item.x + processNodeWidth))
        const groupMaxY = Math.max(...groupItems.map((item) => item.y + processNodeHeight))
        const groupStatuses = groupItems.map((item) => normalizeRunStatus(item.stage.status))
        const groupStatus = groupStatuses.some((status) => ['failed', 'aborted'].includes(status))
          ? 'failed'
          : groupStatuses.some((status) => isRunningStageStatus(status))
            ? 'running'
            : groupStatuses.every((status) => processDependencyReadyStatuses.has(status))
              ? 'success'
              : runActive
                ? 'queued'
                : 'finished'
        groups.push({
          id: `process-group-${parallelGroupSeq}`,
          name: groupName,
          status: groupStatus,
          rootType,
          x: groupMinX - processParallelGroupPaddingX,
          y: groupMinY - processParallelGroupPaddingY,
          width: groupMaxX - groupMinX + processParallelGroupPaddingX * 2,
          height: groupMaxY - groupMinY + processParallelGroupPaddingY * 2
        })
        maxX = Math.max(maxX, groupMaxX)
        maxY = Math.max(maxY, groupMaxY)
      }
      currentSources = parallelEndSources
      cursorX = branchNextX
    }

    nextChildren.forEach((child) => {
      currentSources = placeItem(
        child,
        cursorX,
        groupCenterY,
        activeGroup,
        currentSources,
        rootType
      )
      const childResult = layoutFromParent(
        child.id || '',
        cursorX + processColumnGap,
        groupCenterY,
        currentSources,
        rootType,
        activeGroup,
        nextStack
      )
      cursorX = Math.max(cursorX + processColumnGap, childResult.nextX)
      currentSources = childResult.endSources
    })

    return { nextX: cursorX, endSources: currentSources }
  }

  layoutFromParent(workflowStartNodeId, processGraphPaddingX, processGraphPaddingY + 12, [], 'main')

  if ((childrenMap.get(workflowPostNodeId) || []).length > 0) {
    const postY = maxY + processNodeHeight + processRowGap
    layoutFromParent(workflowPostNodeId, processGraphPaddingX, postY, [], 'post')
  }

  const orphanStages = stages.filter((stage) => !placed.has(stage.nodeId))
  orphanStages.forEach((stage, index) => {
    const previousStage =
      index === 0 ? stages.find((item) => placed.has(item.nodeId)) : orphanStages[index - 1]
    const x = maxX + processColumnGap * (index + 1)
    const y = processGraphPaddingY + 12
    placed.add(stage.nodeId)
    items.push({
      stage,
      x,
      y,
      parallelGroup: '',
      rootType: stage.stageType === 'post' ? 'post' : 'main'
    })
    if (previousStage?.id) {
      edges.push({
        id: `process-edge-${previousStage.id}-${stage.id}`,
        source: previousStage.id,
        target: stage.id,
        status: normalizeRunStatus(stage.status)
      })
    }
    maxX = Math.max(maxX, x)
    maxY = Math.max(maxY, y)
  })

  return {
    items,
    edges,
    groups,
    parallelGroups: parallelGroupSeq,
    width: maxX + processNodeWidth + processGraphPaddingX,
    height: maxY + processNodeHeight + processGraphPaddingY
  }
}
