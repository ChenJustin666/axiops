import { parse, stringify } from 'yaml'
import type {
  TektonDagConfig,
  TektonDagEdge,
  TektonDagPipelineParam,
  TektonDagNode,
  TektonDagTaskParam,
  TektonDagWorkspace,
  TektonNameValue
} from './types'

const clean = (value: any): any => {
  if (Array.isArray(value)) {
    return value.map(clean).filter((item) => {
      if (Array.isArray(item)) return item.length > 0
      if (item && typeof item === 'object') return Object.keys(item).length > 0
      return item !== undefined && item !== null && item !== ''
    })
  }
  if (!value || typeof value !== 'object') return value
  return Object.entries(value).reduce<Record<string, any>>((acc, [key, item]) => {
    const next = clean(item)
    if (next === undefined || next === null) return acc
    if (next === '' && key !== 'value') return acc
    if (Array.isArray(next) && next.length === 0 && !preserveEmptyArrayKeys.has(key)) return acc
    if (
      typeof next === 'object' &&
      !Array.isArray(next) &&
      Object.keys(next).length === 0 &&
      !preserveEmptyObjectKeys.has(key)
    )
      return acc
    acc[key] = next
    return acc
  }, {})
}

const preserveEmptyObjectKeys = new Set([
  'emptyDir',
  'securityContext',
  'podTemplate',
  'taskRunTemplate',
  'computeResources',
  'resources',
  'stdoutConfig',
  'stderrConfig',
  'default',
  'value'
])

const preserveEmptyArrayKeys = new Set(['default', 'value'])

export function renderTektonDagPipelineYAML(config: TektonDagConfig) {
  const activeNodeIds = new Set(
    config.nodes
      .filter((node) => node.enabled !== false && node.type !== 'note')
      .map((node) => node.id)
  )
  const runtimeEdges = config.edges.filter(
    (edge) =>
      ['runAfter', 'result', 'when'].includes(edge.type) &&
      activeNodeIds.has(edge.source) &&
      activeNodeIds.has(edge.target)
  )
  let tasks = config.nodes
    .filter((node) => node.enabled !== false && node.type !== 'finally' && node.type !== 'note')
    .map((node) => renderTask(node, config, runtimeEdges))
  let finallyTasks = config.nodes
    .filter((node) => node.enabled !== false && node.type === 'finally')
    .map((node) => renderTask(node, config, runtimeEdges))
  const cleanupWorkspaces = cleanupWorkspaceMappings(config, tasks, finallyTasks)
  if (config.runPolicy?.cleanBeforeRun && cleanupWorkspaces.length > 0) {
    const cleanupName = uniqueGeneratedTaskName('kube-nova-clean-before-run', tasks, finallyTasks)
    tasks = [
      renderCleanupTask(cleanupName, cleanupWorkspaces, config.runPolicy?.cleanupImage),
      ...tasks
    ]
    addRunAfter(tasks.slice(1), cleanupName)
  }
  if (config.runPolicy?.cleanAfterRun && cleanupWorkspaces.length > 0) {
    finallyTasks = [
      ...finallyTasks,
      renderCleanupTask(
        uniqueGeneratedTaskName('kube-nova-clean-after-run', tasks, finallyTasks),
        cleanupWorkspaces,
        config.runPolicy?.cleanupImage
      )
    ]
  }
  const spec: Record<string, any> = {
    displayName: config.pipeline.displayName,
    description: config.pipeline.description,
    params: renderPipelineParams(config),
    results: renderPipelineResults(config),
    workspaces: renderPipelineWorkspaces(config),
    tasks,
    finally: finallyTasks
  }

  return dump({
    apiVersion: 'tekton.dev/v1',
    kind: 'Pipeline',
    metadata: {
      name: config.pipeline.name,
      namespace: config.pipeline.namespace,
      labels: nameValueListToMap(mergeNameValueLists(config.labels, config.pipeline.labels)),
      annotations: nameValueListToMap(
        mergeNameValueLists(config.annotations, config.pipeline.annotations)
      )
    },
    spec
  })
}

export function renderTektonDagPipelineRunYAML(
  config: TektonDagConfig,
  runName = `${config.pipeline.name}-manual`,
  paramValues: Record<string, string | undefined> = {}
) {
  const runPolicy = config.runPolicy || {}
  const spec: Record<string, any> = {
    pipelineRef: { name: config.pipeline.name },
    params: mergeDeclaredPipelineParams(config).map((item) => ({
      name: item.name,
      value: paramValue(item, paramValues[item.name])
    })),
    workspaces: renderPipelineRunWorkspaces(config, paramValues)
  }
  applyPipelineRunPolicySpec(spec, config, runName, true)
  const labels = nameValueListToMap(runPolicy.labels)
  const annotations = nameValueListToMap(runPolicy.annotations)

  return dump({
    apiVersion: 'tekton.dev/v1',
    kind: 'PipelineRun',
    metadata: {
      name: runName,
      namespace: config.pipeline.namespace,
      labels: clean({
        'kube-nova.io/engine': 'tekton',
        ...(labels || {})
      }),
      annotations: annotations || undefined
    },
    spec
  })
}

export function renderTektonDagTriggerYAML(config: TektonDagConfig, meta: TektonPreviewMeta = {}) {
  const trigger = config.triggerConfig
  if (!trigger?.enabled) return '# Trigger 未启用\n'

  const apiVersion = trigger.apiVersion || 'triggers.tekton.dev/v1beta1'
  const namespace = trigger.namespace || config.pipeline.namespace || 'default'
  const pipelineName = config.pipeline.name || meta.pipelineName || 'tekton-pipeline'
  const bindingName = trigger.triggerBindingName || tektonName(pipelineName, 'binding')
  const templateName = trigger.triggerTemplateName || tektonName(pipelineName, 'template')
  const triggerName = trigger.triggerName || tektonName(pipelineName, 'trigger')
  const eventListenerName = trigger.eventListenerName || tektonName(pipelineName, 'el')
  const triggerServiceAccountName = String(
    trigger.serviceAccountName ||
      config.runPolicy?.serviceAccountName ||
      config.pipeline.serviceAccountName ||
      ''
  ).trim()
  const labels = clean({
    'kube-nova.io/pipeline-id': meta.pipelineId,
    'kube-nova.io/engine': 'tekton',
    ...(nameValueListToMap(trigger.labels) || {})
  })
  const annotations = nameValueListToMap(trigger.annotations)
  const bindingParams = triggerBindingParams(config)
  const templateParams = trigger.templateParams?.length ? trigger.templateParams : bindingParams
  const triggerRunSpec: Record<string, any> = {
    pipelineRef: { name: pipelineName },
    params: templateParams.map((item) => ({
      name: item.name,
      value: `$(tt.params.${item.name})`
    })),
    workspaces: renderPipelineRunWorkspaces(config)
  }
  applyPipelineRunPolicySpec(triggerRunSpec, config, `${pipelineName}-webhook`, false)
  const docs: Record<string, any>[] = [
    {
      apiVersion,
      kind: 'TriggerBinding',
      metadata: { name: bindingName, namespace, labels, annotations },
      spec: {
        params: bindingParams.map((item) => ({ name: item.name, value: item.value }))
      }
    },
    {
      apiVersion,
      kind: 'TriggerTemplate',
      metadata: { name: templateName, namespace, labels, annotations },
      spec: {
        params: templateParams.map((item) => ({
          name: item.name,
          description: item.description,
          default: item.value && !item.value.startsWith('$(') ? item.value : undefined
        })),
        resourcetemplates: [
          {
            apiVersion: 'tekton.dev/v1',
            kind: 'PipelineRun',
            metadata: {
              generateName: `${pipelineName}-`,
              labels: clean({
                'kube-nova.io/pipeline-id': meta.pipelineId,
                'kube-nova.io/pipeline-code': meta.pipelineCode,
                'kube-nova.io/engine': 'tekton',
                'kube-nova.io/trigger-type': 'webhook'
              })
            },
            spec: clean(triggerRunSpec)
          }
        ]
      }
    },
    {
      apiVersion,
      kind: 'Trigger',
      metadata: { name: triggerName, namespace, labels, annotations },
      spec: clean({
        bindings: [{ ref: bindingName }],
        template: { ref: templateName },
        interceptors: renderTriggerInterceptors(config)
      })
    },
    {
      apiVersion,
      kind: 'EventListener',
      metadata: { name: eventListenerName, namespace, labels, annotations },
      spec: clean({
        serviceAccountName: triggerServiceAccountName,
        resources: {
          kubernetesResource: {
            serviceType: trigger.serviceType || 'ClusterIP'
          }
        },
        triggers: [{ triggerRef: triggerName }]
      })
    }
  ]
  const ingress = renderTriggerIngress(config, namespace, labels, annotations)
  if (ingress) docs.push(ingress)
  return dumpAll(docs)
}

export function renderTektonDagSchedulePolicyYAML(
  config: TektonDagConfig,
  meta: TektonPreviewMeta = {}
) {
  const schedule = config.scheduleConfig
  if (!schedule?.enabled) return '# 平台定时未启用\n'
  return dump({
    platformSchedulePolicy: {
      name: tektonName(meta.pipelineCode || config.pipeline.name || 'pipeline', 'schedule'),
      pipelineId: meta.pipelineId,
      pipelineName: config.pipeline.name || meta.pipelineName,
      namespace: config.pipeline.namespace,
      enabled: schedule.enabled === true,
      cron: schedule.cron || '',
      timezone: schedule.timezone || 'Asia/Shanghai',
      concurrencyPolicy: schedule.concurrencyPolicy || 'skip',
      parameterSnapshot: schedule.parameterSnapshot !== false,
      paramsSnapshot: schedule.parameterSnapshot === false ? {} : schedule.paramsSnapshot || {}
    }
  })
}

export function renderTektonDagPrunerPolicyYAML(
  config: TektonDagConfig,
  meta: TektonPreviewMeta = {}
) {
  const pruner = config.prunerPolicy
  if (!pruner?.enabled) return '# Pruner 未启用\n'
  const nativeManifest = renderTektonNativePrunerManifest(config, meta)
  return dump({
    platformPrunerPolicy: {
      name: tektonName(meta.pipelineCode || config.pipeline.name || 'pipeline', 'pruner'),
      pipelineId: meta.pipelineId,
      pipelineName: config.pipeline.name || meta.pipelineName,
      namespace: config.pipeline.namespace,
      scope: pruner.scope || 'pipeline',
      enabled: pruner.enabled === true,
      clusterResourceCleanup: pruner.clusterResourceCleanup !== false,
      platformRecordCleanup: pruner.platformRecordCleanup !== false,
      resources: (pruner.resourceTypes || ['PipelineRun', 'TaskRun']).map((item) =>
        item === 'TaskRun' ? 'taskruns.tekton.dev' : 'pipelineruns.tekton.dev'
      ),
      historyLimit: numberValue(pruner.historyLimit),
      successfulHistoryLimit: numberValue(pruner.successfulHistoryLimit),
      failedHistoryLimit: numberValue(pruner.failedHistoryLimit),
      ttlSecondsAfterFinished: numberValue(pruner.ttlSecondsAfterFinished),
      platformRecordRetentionDays: numberValue(pruner.platformRecordRetentionDays),
      platformRecordLimit: numberValue(pruner.platformRecordLimit),
      selectorLabels: normalizeNameValues(pruner.selectorLabels),
      selectorAnnotations: normalizeNameValues(pruner.selectorAnnotations)
    },
    tektonNativePrunerManifest: nativeManifest
  })
}

function renderTektonNativePrunerManifest(config: TektonDagConfig, meta: TektonPreviewMeta = {}) {
  const pruner = config.prunerPolicy
  if (!pruner?.enabled || !pruner.nativePrunerMode || pruner.nativePrunerMode === 'disabled') {
    return {
      enabled: false,
      note: '未启用 Tekton 原生 Pruner 配置，当前由平台清理器执行'
    }
  }
  const namespace = config.pipeline.namespace || 'default'
  const targetNamespace =
    pruner.nativePrunerMode === 'globalConfigMap'
      ? pruner.nativePrunerTargetNamespace || 'tekton-pipelines'
      : pruner.nativePrunerTargetNamespace || namespace
  const prunerSpec = nativePrunerSpec(config, meta)
  if (pruner.nativePrunerMode === 'globalConfigMap') {
    return {
      enabled: true,
      mode: 'globalConfigMap',
      applyTarget: `${targetNamespace}/tekton-pruner-default-spec`,
      manifest: {
        apiVersion: 'v1',
        kind: 'ConfigMap',
        metadata: {
          name: 'tekton-pruner-default-spec',
          namespace: targetNamespace,
          labels: {
            'app.kubernetes.io/part-of': 'tekton-pruner',
            'pruner.tekton.dev/config-type': 'global',
            'kube-nova.io/managed-by': 'kube-nova'
          }
        },
        data: {
          'global-config': dump(
            clean({
              enforcedConfigLevel: pruner.nativePrunerConfigLevel || 'namespace',
              ...prunerSpec
            })
          )
        }
      }
    }
  }
  return {
    enabled: true,
    mode: 'namespaceConfigMap',
    applyTarget: `${targetNamespace}/tekton-pruner-namespace-spec`,
    manifest: {
      apiVersion: 'v1',
      kind: 'ConfigMap',
      metadata: {
        name: 'tekton-pruner-namespace-spec',
        namespace: targetNamespace,
        labels: {
          'app.kubernetes.io/part-of': 'tekton-pruner',
          'pruner.tekton.dev/config-type': 'namespace',
          'kube-nova.io/managed-by': 'kube-nova'
        }
      },
      data: {
        'ns-config': dump(prunerSpec)
      }
    }
  }
}

function nativePrunerSpec(config: TektonDagConfig, meta: TektonPreviewMeta = {}) {
  return clean(nativeResourceGroups(config, meta))
}

function nativeResourceGroups(config: TektonDagConfig, meta: TektonPreviewMeta = {}) {
  const pruner = config.prunerPolicy || {}
  const ownerLabels = meta.pipelineId ? { 'kube-nova.io/pipeline-id': meta.pipelineId } : {}
  const group =
    normalizeNameValues(pruner.selectorLabels).length ||
    normalizeNameValues(pruner.selectorAnnotations).length
      ? {
          selector: [
            clean({
              matchLabels: {
                ...(nameValueListToMap(pruner.selectorLabels) || {}),
                ...ownerLabels
              },
              matchAnnotations: nameValueListToMap(pruner.selectorAnnotations)
            })
          ],
          historyLimit: numberValue(pruner.historyLimit),
          successfulHistoryLimit: numberValue(pruner.successfulHistoryLimit),
          failedHistoryLimit: numberValue(pruner.failedHistoryLimit),
          ttlSecondsAfterFinished: numberValue(pruner.ttlSecondsAfterFinished)
        }
      : undefined
  if (!group && pruner.scope !== 'pipeline') return {}
  const scopedGroup =
    group ||
    clean({
      selector: [
        {
          matchLabels: {
            'tekton.dev/pipeline': config.pipeline.name || meta.pipelineName,
            ...ownerLabels
          }
        }
      ],
      historyLimit: numberValue(pruner.historyLimit),
      successfulHistoryLimit: numberValue(pruner.successfulHistoryLimit),
      failedHistoryLimit: numberValue(pruner.failedHistoryLimit),
      ttlSecondsAfterFinished: numberValue(pruner.ttlSecondsAfterFinished)
    })
  const result: Record<string, any> = {}
  ;(pruner.resourceTypes || ['PipelineRun', 'TaskRun']).forEach((item) => {
    if (item === 'TaskRun') result.taskRuns = [scopedGroup]
    if (item === 'PipelineRun') result.pipelineRuns = [scopedGroup]
  })
  return result
}

interface TektonPreviewMeta {
  pipelineId?: string
  pipelineCode?: string
  pipelineName?: string
}

function renderPipelineParams(config: TektonDagConfig) {
  return mergeDeclaredPipelineParams(config).map((item) => {
    const type = tektonPipelineParamType(item)
    return {
      name: item.name,
      type,
      default:
        item.defaultValue === undefined ||
        (item.required && !String(item.defaultValue || '').trim())
          ? undefined
          : paramValue(item),
      description: item.description,
      enum: renderPipelineParamEnum(item),
      properties: type === 'object' ? item.properties : undefined
    }
  })
}

function renderPipelineParamEnum(item: TektonDagPipelineParam) {
  const direct = (item.enum || []).map((value) => String(value || '').trim()).filter(Boolean)
  if (direct.length > 0) return Array.from(new Set(direct))
  if (!['choice', 'list'].includes(String(item.paramType || ''))) return undefined
  const options = item.selectList?.length ? item.selectList : item.options || []
  const values = options.map((option) => String(option.value || '').trim()).filter(Boolean)
  return values.length > 0 ? Array.from(new Set(values)) : undefined
}

function renderPipelineResults(config: TektonDagConfig) {
  return mergeDeclaredPipelineResults(config).map((item) => ({
    name: item.name,
    type: item.type || 'string',
    properties: item.type === 'object' ? item.properties : undefined,
    description: item.description,
    value: item.value
  }))
}

function mergeDeclaredPipelineParams(config: TektonDagConfig) {
  const merged = new Map<string, TektonDagPipelineParam>()
  ;[...(config.pipeline.params || []), ...(config.params || [])].forEach((item) => {
    const name = String(item?.name || '').trim()
    if (!name || merged.has(name)) return
    merged.set(name, { ...item, name })
  })
  return Array.from(merged.values())
}

function mergeDeclaredPipelineResults(config: TektonDagConfig) {
  const merged = new Map<string, NonNullable<TektonDagConfig['results']>[number]>()
  ;[...(config.pipeline.results || []), ...(config.results || [])].forEach((item) => {
    const name = String(item?.name || '').trim()
    if (!name || merged.has(name)) return
    merged.set(name, { ...item, name })
  })
  return Array.from(merged.values())
}

function renderPipelineWorkspaces(config: TektonDagConfig) {
  const items = mergeDeclaredWorkspaces(config, true)
  return items.map((item) => ({
    name: item.name,
    description: item.description,
    optional: item.optional === true ? true : undefined
  }))
}

function renderTask(node: TektonDagNode, config: TektonDagConfig, runtimeEdges: TektonDagEdge[]) {
  const task: Record<string, any> = {
    name: node.name,
    displayName: node.displayName,
    description: node.description,
    params: renderTaskParams(node.params, config, node, runtimeEdges),
    workspaces: node.workspaces.map((item) =>
      clean({
        name: item.taskWorkspace,
        workspace: item.pipelineWorkspace,
        subPath: item.subPath
      })
    ),
    when: [...node.when.map(renderWhenExpression), ...renderWhenEdges(node, runtimeEdges)],
    retries: node.retries,
    timeout: node.timeout,
    onError: node.onError,
    matrix: renderMatrix(node)
  }
  const pipelineRef = renderPipelineRef(node)
  if (hasRecordValue(node.pipelineSpec)) {
    task.pipelineSpec = node.pipelineSpec
  } else if (pipelineRef) {
    task.pipelineRef = pipelineRef
  } else if (hasRecordValue(node.taskSpec)) {
    task.taskSpec = node.taskSpec
  } else {
    task.taskRef = renderTaskRef(node)
  }
  if (node.type !== 'finally') {
    const deps = runtimeEdges
      .filter((edge) => edge.target === node.id)
      .filter((edge) => edge.type === 'runAfter')
      .map((edge) => config.nodes.find((item) => item.id === edge.source)?.name || '')
      .filter(Boolean)
    if (deps.length > 0) task.runAfter = deps
  }
  return task
}

function renderWhenEdges(node: TektonDagNode, runtimeEdges: TektonDagEdge[]) {
  return runtimeEdges
    .filter((edge) => edge.type === 'when' && edge.target === node.id)
    .filter((edge) => edge.whenInput && edge.whenValues?.length)
    .map((edge) => ({
      input: edge.whenInput,
      operator: edge.whenOperator || 'in',
      values: edge.whenValues || []
    }))
}

function renderWhenExpression(item: TektonDagNode['when'][number]) {
  if (item.cel?.trim()) {
    return { cel: item.cel.trim() }
  }
  return {
    input: item.input,
    operator: item.operator || 'in',
    values: item.values || []
  }
}

function cleanupWorkspaceMappings(
  config: TektonDagConfig,
  tasks: Record<string, any>[],
  finallyTasks: Record<string, any>[]
) {
  const declared = renderPipelineWorkspaces(config)
  const fallback = pipelineWorkspaceDeclarations([...tasks, ...finallyTasks])
  const workspaces = declared.length > 0 ? declared : fallback
  const configuredWorkspaces = mergeDeclaredWorkspaces(config, true)
  const writable = new Set(
    configuredWorkspaces
      .filter((item) => !['secret', 'configMap'].includes(item.bindingType || ''))
      .map((item) => item.name)
      .filter(Boolean)
  )
  const hasConfiguredWorkspaces = configuredWorkspaces.length > 0
  return workspaces
    .map((item) => String(item.name || '').trim())
    .filter((name) => name && (!hasConfiguredWorkspaces || writable.has(name)))
    .map((name) => ({ name, workspace: name }))
}

function pipelineWorkspaceDeclarations(tasks: Record<string, any>[]) {
  const names = new Set<string>()
  tasks.forEach((task) => {
    ;(task.workspaces || []).forEach((item: Record<string, any>) => {
      const name = String(item.workspace || item.name || '').trim()
      if (name) names.add(name)
    })
  })
  return Array.from(names).map((name) => ({ name }))
}

function renderCleanupTask(
  name: string,
  workspaces: Array<{ name: string; workspace: string }>,
  image?: string
) {
  return {
    name,
    workspaces: workspaces.map((item) => ({ name: item.name, workspace: item.workspace })),
    taskSpec: {
      workspaces: workspaces.map((item) => ({ name: item.name })),
      steps: workspaces.map((item) => ({
        name: tektonName('clean', item.name),
        image: String(image || '').trim() || 'busybox:1.36',
        script: cleanupScript(item.name)
      }))
    }
  }
}

function cleanupScript(workspaceName: string) {
  return `#!/bin/sh
set -eu
TARGET="$(workspaces.${workspaceName}.path)"
if [ -n "$TARGET" ] && [ -d "$TARGET" ]; then
  rm -rf "$TARGET"/* "$TARGET"/.[!.]* "$TARGET"/..?* 2>/dev/null || true
fi
`
}

function addRunAfter(tasks: Record<string, any>[], dependency: string) {
  tasks.forEach((task) => {
    const current = Array.isArray(task.runAfter) ? task.runAfter : []
    task.runAfter = Array.from(new Set([...current, dependency])).sort()
  })
}

function uniqueGeneratedTaskName(base: string, ...groups: Array<Array<Record<string, any>>>) {
  const used = new Set(
    groups
      .flat()
      .map((task) => String(task.name || '').trim())
      .filter(Boolean)
  )
  const name = tektonName(base)
  if (!used.has(name)) return name
  for (let index = 1; index < 20; index += 1) {
    const candidate = tektonName(base, String(index))
    if (!used.has(candidate)) return candidate
  }
  return tektonName(base, 'task')
}

function renderTaskRef(node: TektonDagNode) {
  const taskRef = node.taskRef || { kind: 'Task', name: '' }
  if (taskRef.resolver || taskRef.namespace || taskRef.params?.length) {
    const defaultResolverParams = [
      { name: 'kind', value: taskRef.kind === 'ClusterTask' ? 'clustertask' : 'task' },
      { name: 'name', value: taskRef.name },
      { name: 'namespace', value: taskRef.namespace || '' }
    ].filter((item) => item.value)
    return {
      resolver: taskRef.resolver || 'cluster',
      params: taskRef.params?.length ? taskRef.params : defaultResolverParams,
      apiVersion: taskRef.apiVersion
    }
  }
  const kind = String(taskRef.kind || 'Task').trim()
  return {
    name: taskRef.name,
    kind: kind !== 'Task' || taskRef.apiVersion ? kind : undefined,
    apiVersion: taskRef.apiVersion
  }
}

function renderPipelineRef(node: TektonDagNode) {
  const pipelineRef = node.pipelineRef
  if (!pipelineRef) return undefined
  const hasResolver = Boolean(pipelineRef.resolver || pipelineRef.params?.length)
  if (hasResolver) {
    return {
      resolver: pipelineRef.resolver || 'cluster',
      params: pipelineRef.params || [],
      apiVersion: pipelineRef.apiVersion
    }
  }
  if (!pipelineRef.name) return undefined
  return {
    name: pipelineRef.name,
    apiVersion: pipelineRef.apiVersion
  }
}

function hasRecordValue(value?: Record<string, any>) {
  return Boolean(value && typeof value === 'object' && Object.keys(value).length > 0)
}

function renderTaskParams(
  params: TektonDagTaskParam[],
  config: TektonDagConfig,
  node: TektonDagNode,
  runtimeEdges: TektonDagEdge[]
) {
  const requirements = new Map((node.paramRequirements || []).map((item) => [item.name, item]))
  const rendered = params.map((item) => ({
    name: item.name,
    value: taskParamValue(
      item,
      config,
      node,
      runtimeEdges,
      requirements.get(item.name)?.type || requirements.get(item.name)?.paramType
    )
  }))
  runtimeEdges
    .filter((edge) => edge.type === 'result' && edge.target === node.id && edge.targetParam)
    .forEach((edge) => {
      if (rendered.some((item) => item.name === edge.targetParam)) return
      const sourceName = config.nodes.find((dagNode) => dagNode.id === edge.source)?.name
      if (!sourceName || !edge.resultName) return
      rendered.push({
        name: edge.targetParam || '',
        value: `$(tasks.${sourceName}.results.${edge.resultName})`
      })
    })
  return rendered
}

function taskParamValue(
  item: TektonDagTaskParam,
  config: TektonDagConfig,
  node: TektonDagNode,
  runtimeEdges: TektonDagEdge[],
  requirementType?: string
) {
  switch (item.source) {
    case 'pipelineParam':
    case 'runtime':
      return `$(params.${item.pipelineParam || item.name})`
    case 'result':
      return resultValue(item, config, node, runtimeEdges)
    default:
      return typedParamValue(item.value || item.defaultValue || '', requirementType, item.paramType)
  }
}

function resultValue(
  item: TektonDagTaskParam,
  config: TektonDagConfig,
  node: TektonDagNode,
  runtimeEdges: TektonDagEdge[]
) {
  const edge = runtimeEdges.find(
    (edgeItem) =>
      edgeItem.type === 'result' &&
      edgeItem.target === node.id &&
      edgeItem.targetParam === item.name
  )
  const resultNodeId = edge?.source || item.resultNodeId
  const resultName = edge?.resultName || item.resultName
  const taskName = config.nodes.find((dagNode) => dagNode.id === resultNodeId)?.name
  if (!taskName || !resultName) return item.value || ''
  return `$(tasks.${taskName}.results.${resultName})`
}

function renderMatrix(node: TektonDagNode) {
  if (!node.matrix) return undefined
  return {
    params: node.matrix.params.map((item) => ({
      name: item.name,
      value: item.values
    })),
    include: node.matrix.include?.map((item) => ({
      name: item.name,
      params: item.params
    }))
  }
}

function renderPipelineRunWorkspaces(
  config: TektonDagConfig,
  paramValues: Record<string, string | undefined> = {}
) {
  return mergeDeclaredWorkspaces(config, true).map((item) =>
    renderPipelineRunWorkspace(item, config, paramValues)
  )
}

function mergeDeclaredWorkspaces(config: TektonDagConfig, includeRunPolicy = false) {
  const items = [
    ...(config.pipeline.workspaces || []),
    ...(config.workspaces || []),
    ...(includeRunPolicy ? config.runPolicy?.workspaces || [] : [])
  ]
  const merged = new Map<string, TektonDagWorkspace>()
  items.forEach((item) => {
    const name = String(item.name || '').trim()
    if (!name || merged.has(name)) return
    merged.set(name, { ...item, name })
  })
  return Array.from(merged.values())
}

function renderPipelineRunWorkspace(
  item: TektonDagWorkspace,
  config?: TektonDagConfig,
  paramValues: Record<string, string | undefined> = {}
) {
  const originalBindingType = item.bindingType || 'emptyDir'
  const policyBindingType = config?.runPolicy?.workspaceStrategy || ''
  const bindingType =
    originalBindingType === 'emptyDir' && policyBindingType
      ? policyBindingType
      : originalBindingType
  const runtimeValue = paramValues[item.name]
  switch (bindingType) {
    case 'persistentVolumeClaim':
      return {
        name: item.name,
        persistentVolumeClaim: { claimName: runtimeValue || item.claimName },
        subPath: item.subPath
      }
    case 'volumeClaimTemplate':
      return {
        name: item.name,
        volumeClaimTemplate: {
          spec: {
            accessModes: [item.accessMode || 'ReadWriteOnce'],
            storageClassName: item.storageClassName,
            volumeMode: item.volumeMode,
            resources: {
              requests: {
                storage: item.storage || '10Gi'
              }
            }
          }
        }
      }
    case 'secret':
      return {
        name: item.name,
        secret: { secretName: runtimeValue || item.secretName },
        subPath: item.subPath
      }
    case 'generatedSecret':
      return {
        name: item.name,
        secret: { secretName: item.secretName || `workspace-secret-${item.name}` },
        subPath: item.subPath
      }
    case 'configMap':
      return {
        name: item.name,
        configMap: { name: runtimeValue || item.configMapName },
        subPath: item.subPath
      }
    case 'projected':
      return {
        name: item.name,
        projected: item.projected || item.config?.projected,
        subPath: item.subPath
      }
    default:
      return { name: item.name, emptyDir: {} }
  }
}

function triggerBindingParams(config: TektonDagConfig) {
  const trigger = config.triggerConfig
  if (trigger?.bindingParams?.length) {
    return trigger.bindingParams.filter((item) => item.name)
  }
  return mergeDeclaredPipelineParams(config)
    .filter((item) => item.name)
    .map((item) => ({
      name: item.name,
      value: `$(body.${item.name})`,
      description: item.description
    }))
}

function renderTriggerInterceptors(config: TektonDagConfig) {
  const trigger = config.triggerConfig
  const interceptorType = triggerInterceptorType(config)
  if (!interceptorType || interceptorType === 'none') return undefined
  if (interceptorType === 'cel') {
    if (!trigger?.celFilter) return undefined
    return [
      {
        ref: { name: 'cel' },
        params: [{ name: 'filter', value: trigger.celFilter }]
      }
    ]
  }
  const params: Array<{ name: string; value: unknown }> = []
  if (trigger?.secretName) {
    params.push({
      name: 'secretRef',
      value: {
        secretName: trigger.secretName,
        secretKey: trigger.secretKey || 'token'
      }
    })
  }
  const eventTypes = triggerEventTypes(config)
  if (eventTypes.length > 0) params.push({ name: 'eventTypes', value: eventTypes })
  return [{ ref: { name: interceptorType }, params: params.length ? params : undefined }]
}

function triggerInterceptorType(config: TektonDagConfig) {
  const trigger = config.triggerConfig
  if (trigger?.interceptorType) return trigger.interceptorType
  switch ((trigger?.provider || '').toLowerCase()) {
    case 'github':
      return 'github'
    case 'gitlab':
      return 'gitlab'
    default:
      return 'none'
  }
}

function triggerEventTypes(config: TektonDagConfig) {
  const configured = (config.triggerConfig?.eventTypes || []).filter(Boolean)
  if (configured.length > 0) return configured
  switch ((config.triggerConfig?.provider || '').toLowerCase()) {
    case 'github':
      return ['push']
    case 'gitlab':
      return ['Push Hook']
    default:
      return []
  }
}

function renderTriggerIngress(
  config: TektonDagConfig,
  namespace: string,
  labels?: Record<string, any>,
  annotations?: Record<string, any>
) {
  const trigger = config.triggerConfig
  if (!trigger?.ingressHost) return undefined
  const eventListenerName = trigger.eventListenerName || tektonName(config.pipeline.name, 'el')
  const rawPath = trigger.webhookPath || trigger.ingressPath || '/'
  const path = rawPath.startsWith('/') ? rawPath : `/${rawPath}`
  return {
    apiVersion: 'networking.k8s.io/v1',
    kind: 'Ingress',
    metadata: {
      name: tektonName(eventListenerName, 'ingress'),
      namespace,
      labels,
      annotations
    },
    spec: {
      ingressClassName: trigger.ingressClassName,
      tls: trigger.tlsSecretName
        ? [{ hosts: [trigger.ingressHost], secretName: trigger.tlsSecretName }]
        : undefined,
      rules: [
        {
          host: trigger.ingressHost,
          http: {
            paths: [
              {
                path,
                pathType: 'Prefix',
                backend: {
                  service: {
                    name: `el-${eventListenerName}`,
                    port: { number: 8080 }
                  }
                }
              }
            ]
          }
        }
      ]
    }
  }
}

function renderTimeouts(config: TektonDagConfig) {
  const timeouts = {
    ...(config.pipeline.timeouts || {}),
    ...(config.runPolicy?.timeouts || {}),
    pipeline:
      config.runPolicy?.timeout ||
      config.runPolicy?.pipelineTimeout ||
      config.pipeline.timeout ||
      config.runPolicy?.timeouts?.pipeline ||
      config.pipeline.timeouts?.pipeline,
    tasks: config.runPolicy?.tasksTimeout || config.runPolicy?.timeouts?.tasks,
    finally: config.runPolicy?.finallyTimeout || config.runPolicy?.timeouts?.finally
  }
  return clean(timeouts)
}

function applyPipelineRunPolicySpec(
  spec: Record<string, any>,
  config: TektonDagConfig,
  runName: string,
  includePlatformEnv: boolean
) {
  const runPolicy = config.runPolicy || {}
  const managedBy = String(runPolicy.managedBy || '').trim()
  if (managedBy) spec.managedBy = managedBy
  const timeouts = renderTimeouts(config)
  if (timeouts) spec.timeouts = timeouts
  const taskRunTemplate = renderPipelineRunTaskRunTemplate(config, runName, includePlatformEnv)
  if (taskRunTemplate) spec.taskRunTemplate = taskRunTemplate
  const taskRunSpecs = parseTaskRunSpecs(runPolicy.taskRunSpecsRaw, runPolicy.taskRunSpecs)
  if (taskRunSpecs) spec.taskRunSpecs = taskRunSpecs
}

function renderPipelineRunTaskRunTemplate(
  config: TektonDagConfig,
  runName: string,
  includePlatformEnv: boolean
) {
  const runPolicy = config.runPolicy || {}
  const taskRunTemplate = parseYamlObject(runPolicy.taskRunTemplateRaw) || {}
  const serviceAccountName = String(
    runPolicy.serviceAccountName || config.pipeline.serviceAccountName || ''
  ).trim()
  if (serviceAccountName) taskRunTemplate.serviceAccountName = serviceAccountName
  const podTemplate = {
    ...(taskRunTemplate.podTemplate || {}),
    ...(parseYamlObject(runPolicy.podTemplateRaw) || {})
  }
  const envItems = [...(runPolicy.podTemplateEnv || [])]
  if (includePlatformEnv) {
    envItems.push(
      { name: 'BUILD_ID', value: '$(buildId)' },
      { name: 'buildId', value: '$(buildId)' },
      { name: 'KUBE_NOVA_BUILD_ID', value: '$(buildId)' },
      { name: 'KUBE_NOVA_RUN_ID', value: '$(runId)' },
      { name: 'KUBE_NOVA_PIPELINE_RUN_NAME', value: runName }
    )
  }
  if (envItems.length > 0) {
    podTemplate.env = appendTektonPodEnv(podTemplate.env, envItems, true)
  }
  if (Object.keys(podTemplate).length > 0) taskRunTemplate.podTemplate = podTemplate
  return clean(taskRunTemplate)
}

function paramValue(item: TektonDagPipelineParam, overrideValue?: string) {
  const value = overrideValue ?? item.defaultValue ?? ''
  return typedParamValue(value, tektonPipelineParamType(item), item.paramType)
}

function typedParamValue(value: string, type?: string, paramType?: string) {
  const tektonType =
    type ||
    (['list', 'array'].includes(String(paramType || ''))
      ? 'array'
      : String(paramType || '') === 'object'
        ? 'object'
        : 'string')
  if (tektonType === 'array') {
    const parsed = parseValue(value)
    if (Array.isArray(parsed)) return parsed
    return value
      .split(',')
      .map((part) => part.trim())
      .filter(Boolean)
  }
  if (tektonType === 'object') {
    const parsed = parseValue(value)
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed : {}
  }
  return value
}

function tektonPipelineParamType(item: TektonDagPipelineParam) {
  if (item.paramType === 'objectList') return 'string'
  return item.type || 'string'
}

function parseValue(value: string) {
  try {
    return parse(value)
  } catch {
    return value
  }
}

function nameValueListToMap(items?: { name: string; value: string }[]) {
  if (!items?.length) return undefined
  return items.reduce<Record<string, string>>((acc, item) => {
    if (item.name) acc[item.name] = item.value
    return acc
  }, {})
}

function mergeNameValueLists(...groups: Array<Array<{ name: string; value: string }> | undefined>) {
  const merged = new Map<string, { name: string; value: string }>()
  groups.forEach((items) => {
    ;(items || []).forEach((item) => {
      const name = String(item?.name || '').trim()
      const value = String(item?.value || '').trim()
      if (!name || !value) return
      merged.set(name, { name, value })
    })
  })
  return Array.from(merged.values())
}

function appendTektonPodEnv(
  current: TektonNameValue[] | undefined,
  additions: TektonNameValue[],
  additionsWin = false
) {
  const override = new Set((additions || []).map((item) => item.name).filter(Boolean))
  const result = (current || []).filter((item) => {
    if (!item?.name) return false
    return additionsWin ? !override.has(item.name) : true
  })
  const seen = new Set(result.map((item) => item.name).filter(Boolean))
  additions.forEach((item) => {
    if (!item?.name || seen.has(item.name)) return
    seen.add(item.name)
    result.push(item)
  })
  return result
}

function parseTaskRunSpecs(raw?: string, directItems?: Record<string, any>[]) {
  if (Array.isArray(directItems) && directItems.length > 0) {
    return clean(normalizeTaskRunSpecs(directItems))
  }
  if (!raw?.trim()) return undefined
  try {
    const parsed = parse(raw)
    if (Array.isArray(parsed)) return clean(normalizeTaskRunSpecs(parsed))
    if (parsed && typeof parsed === 'object') return clean(normalizeTaskRunSpecs([parsed]))
  } catch {
    return undefined
  }
  return undefined
}

const taskRunSpecFieldAliases: Array<[string, string]> = [
  ['taskServiceAccountName', 'serviceAccountName'],
  ['taskPodTemplate', 'podTemplate'],
  ['stepOverrides', 'stepSpecs'],
  ['sidecarOverrides', 'sidecarSpecs']
]

function normalizeTaskRunSpecs(items: any[]) {
  return items.map((item) => {
    if (!item || typeof item !== 'object' || Array.isArray(item)) return item
    const next = { ...item }
    taskRunSpecFieldAliases.forEach(([legacyKey, v1Key]) => {
      if (hasTaskRunSpecValue(next[legacyKey]) && !hasTaskRunSpecValue(next[v1Key])) {
        next[v1Key] = next[legacyKey]
      }
      delete next[legacyKey]
    })
    return next
  })
}

function hasTaskRunSpecValue(value: any) {
  if (value === undefined || value === null) return false
  if (typeof value === 'string') return value.trim() !== ''
  if (Array.isArray(value)) return value.length > 0
  if (typeof value === 'object') return Object.keys(value).length > 0
  return true
}

function parseYamlObject(raw?: string) {
  if (!raw?.trim()) return undefined
  try {
    const parsed = parse(raw)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) return clean(parsed)
  } catch {
    return undefined
  }
  return undefined
}

function normalizeNameValues(items?: { name: string; value: string }[]) {
  return (items || [])
    .map((item) => ({ name: item.name, value: item.value }))
    .filter((item) => item.name)
}

function numberValue(value?: number) {
  return Number(value || 0)
}

function tektonName(...parts: string[]) {
  const value = parts
    .join('-')
    .toLowerCase()
    .replace(/[^a-z0-9-]/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
  return value || 'task'
}

function dump(obj: Record<string, any>) {
  return stringify(clean(obj), { indent: 2, lineWidth: 120 }).trimEnd() + '\n'
}

function dumpAll(items: Record<string, any>[]) {
  return items.map((item) => dump(item).trimEnd()).join('\n---\n') + '\n'
}
