import { parse, parseAllDocuments, stringify } from 'yaml'

export type TektonParamType = 'string' | 'array' | 'object'

export interface TektonTaskParam {
  name: string
  type: TektonParamType
  default?: string
  description?: string
  enum?: string[]
  properties?: Record<string, { type?: TektonParamType }>
  required?: boolean
}

export interface TektonWorkspace {
  name: string
  description?: string
  optional?: boolean
  readOnly?: boolean
  mountPath?: string
}

export interface TektonTaskStep {
  name: string
  image: string
  refText?: string
  script?: string
  commandText?: string
  argsText?: string
  workingDir?: string
  advancedRaw?: string
}

export interface TektonTaskResult {
  name: string
  type?: TektonParamType
  properties?: Record<string, { type?: TektonParamType }>
  description?: string
  value?: string
}

export interface TektonTaskForm {
  apiVersion: string
  name: string
  displayName?: string
  description?: string
  metadataLabelsRaw?: string
  metadataAnnotationsRaw?: string
  stepTemplateRaw?: string
  sidecarsRaw?: string
  volumesRaw?: string
  params: TektonTaskParam[]
  workspaces: TektonWorkspace[]
  results: TektonTaskResult[]
  steps: TektonTaskStep[]
}

export interface TaskParseResult {
  resource: Record<string, any>
  form: TektonTaskForm
  hasAdvancedFields: boolean
  normalizedYaml: string
}

export type TaskParseResponse = { ok: true; data: TaskParseResult } | { ok: false; error: string }

export type TaskYamlMutationResponse =
  | { ok: true; content: string; images: string[]; changed: boolean }
  | { ok: false; error: string }

const supportedApiVersions = ['tekton.dev/v1', 'tekton.dev/v1beta1']
const dnsLabelPattern = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/
const tektonDurationPattern = /^([0-9]+(ms|h|m|s))+$/i

export const isPlainObject = (value: unknown): value is Record<string, any> =>
  Boolean(value) && typeof value === 'object' && !Array.isArray(value)

export const cloneValue = <T>(value: T): T => JSON.parse(JSON.stringify(value || {}))

export const normalizeTektonName = (value?: string) => {
  let name = String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9-]+/g, '-')
    .replace(/^-+|-+$/g, '')
  if (name.length > 63) {
    name = name.slice(0, 63).replace(/^-+|-+$/g, '')
  }
  return name || 'tekton-step'
}

const splitLines = (value?: string) =>
  String(value || '')
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean)

const isDnsLabel = (value: string) => value.length <= 63 && dnsLabelPattern.test(value)

const preserveEmptyObjectKeys = new Set([
  'emptyDir',
  'securityContext',
  'computeResources',
  'resources',
  'stdoutConfig',
  'stderrConfig',
  'default',
  'value',
  'podTemplate',
  'taskRunTemplate'
])

const preserveEmptyArrayKeys = new Set(['default', 'value'])

const cleanObject = (input: any): any => {
  if (Array.isArray(input)) {
    return input.map(cleanObject).filter((item) => {
      if (Array.isArray(item)) return item.length > 0
      if (item && typeof item === 'object') return Object.keys(item).length > 0
      return item !== undefined && item !== null && item !== ''
    })
  }
  if (!input || typeof input !== 'object') return input
  return Object.entries(input).reduce<Record<string, any>>((acc, [key, value]) => {
    const next = cleanObject(value)
    if (next === undefined || next === null) return acc
    if (next === '' && !preserveEmptyObjectKeys.has(key)) return acc
    if (Array.isArray(next) && next.length === 0 && !preserveEmptyArrayKeys.has(key)) return acc
    if (
      typeof next === 'object' &&
      !Array.isArray(next) &&
      Object.keys(next).length === 0 &&
      !preserveEmptyObjectKeys.has(key)
    ) {
      return acc
    }
    acc[key] = next
    return acc
  }, {})
}

const listByName = (items: any[]) => {
  const result = new Map<string, Record<string, any>>()
  items.forEach((item) => {
    if (!isPlainObject(item)) return
    const name = String(item.name || '').trim()
    if (name) result.set(name, item)
  })
  return result
}

const specList = (resource: Record<string, any>, key: string) => {
  const spec = isPlainObject(resource.spec) ? resource.spec : {}
  const items = spec[key]
  return Array.isArray(items) ? items : []
}

const yamlErrorText = (error: any) => {
  const line = error?.linePos?.[0]
  const position = line ? `第 ${line.line} 行第 ${line.col} 列：` : ''
  return `${position}${error?.message || 'YAML 解析失败'}`
}

const valueToFormString = (value: unknown) => {
  if (value === undefined || value === null) return ''
  if (Array.isArray(value) || isPlainObject(value)) return stringify(value).trim()
  return String(value)
}

const normalizeDefaultValue = (base: Record<string, any> | undefined, item: TektonTaskParam) => {
  const current = item.default ?? ''
  if (base && Object.prototype.hasOwnProperty.call(base, 'default')) {
    const oldText = valueToFormString(base.default)
    if (oldText === current) return base.default
  }
  return parseTypedDefaultValue(item.type, current, `Task 参数 ${item.name} 默认值`)
}

const normalizeTaskResultValue = (
  base: Record<string, any> | undefined,
  item: TektonTaskResult
) => {
  const current = item.value ?? ''
  if (base && Object.prototype.hasOwnProperty.call(base, 'value')) {
    const oldText = valueToFormString(base.value)
    if (oldText === current) return base.value
  }
  return parseTypedDefaultValue(item.type || 'string', current, `Task Result ${item.name} value`)
}

const parseTypedDefaultValue = (type: TektonParamType, value: string, label: string) => {
  const text = String(value || '').trim()
  if (!text) return undefined
  if (type === 'array') {
    let parsed: any
    try {
      parsed = parse(text)
    } catch {
      parsed = undefined
    }
    if (Array.isArray(parsed)) {
      if (parsed.some((item) => typeof item !== 'string')) {
        throw new Error(`${label} 必须是字符串数组`)
      }
      return parsed
    }
    const parts = text
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean)
    if (parts.length > 0) return parts
    throw new Error(`${label} 必须是 YAML 数组或逗号分隔列表`)
  }
  if (type === 'object') {
    let parsed: any
    try {
      parsed = parse(text)
    } catch {
      throw new Error(`${label} 必须是 YAML 对象`)
    }
    if (!isPlainObject(parsed)) throw new Error(`${label} 必须是 YAML 对象`)
    return parsed
  }
  return value
}

const normalizeStepRefValue = (base: Record<string, any> | undefined, item: TektonTaskStep) => {
  const current = String(item.refText || '').trim()
  if (!current) return undefined
  if (base && Object.prototype.hasOwnProperty.call(base, 'ref')) {
    const oldText = valueToFormString(base.ref)
    if (oldText === current) return base.ref
  }
  try {
    const parsed = parse(current)
    if (isPlainObject(parsed)) return parsed
  } catch {
    return current
  }
  return current
}

const parseOptionalYamlField = (
  raw: string | undefined,
  type: 'object' | 'array',
  label: string
) => {
  const text = String(raw || '').trim()
  if (!text) return undefined
  let parsed: any
  try {
    parsed = parse(text)
  } catch {
    throw new Error(`${label} 不是合法 YAML`)
  }
  if (type === 'object') {
    if (!isPlainObject(parsed)) throw new Error(`${label} 必须是 YAML 对象`)
    return parsed
  }
  if (!Array.isArray(parsed)) throw new Error(`${label} 必须是 YAML 数组`)
  if (parsed.some((item) => !isPlainObject(item))) {
    throw new Error(`${label} 数组项必须是对象`)
  }
  return parsed
}

const parseTaskSidecarsRaw = (raw: string | undefined) => {
  const sidecars = parseOptionalYamlField(raw, 'array', 'Task spec.sidecars')
  validateTaskSidecars(sidecars)
  return sidecars
}

const parseTaskVolumesRaw = (raw: string | undefined) => {
  const volumes = parseOptionalYamlField(raw, 'array', 'Task spec.volumes')
  validateTaskVolumes(volumes)
  return volumes
}

const validateTaskSidecars = (sidecars: any[] | undefined) => {
  if (!sidecars?.length) return
  const names = new Set<string>()
  for (const item of sidecars) {
    const name = String(item.name || '').trim()
    if (!name) throw new Error('Task spec.sidecars.name 不能为空')
    if (!isDnsLabel(name)) throw new Error(`Task spec.sidecars.name 不符合 DNS_LABEL 规范：${name}`)
    if (names.has(name)) throw new Error(`Task spec.sidecars.name 不能重复：${name}`)
    names.add(name)
    if (!String(item.image || '').trim()) {
      throw new Error(`Task spec.sidecars ${name} image 不能为空`)
    }
  }
}

const taskVolumeSourceKeys = new Set([
  'hostPath',
  'emptyDir',
  'gcePersistentDisk',
  'awsElasticBlockStore',
  'gitRepo',
  'secret',
  'nfs',
  'iscsi',
  'glusterfs',
  'persistentVolumeClaim',
  'rbd',
  'flexVolume',
  'cinder',
  'cephfs',
  'flocker',
  'downwardAPI',
  'fc',
  'azureFile',
  'configMap',
  'vsphereVolume',
  'quobyte',
  'azureDisk',
  'photonPersistentDisk',
  'projected',
  'portworxVolume',
  'scaleIO',
  'storageos',
  'csi',
  'ephemeral',
  'image'
])

const validateTaskVolumes = (volumes: any[] | undefined) => {
  if (!volumes?.length) return
  const names = new Set<string>()
  for (const item of volumes) {
    const name = String(item.name || '').trim()
    if (!name) throw new Error('Task spec.volumes.name 不能为空')
    if (!isDnsLabel(name)) throw new Error(`Task spec.volumes.name 不符合 DNS_LABEL 规范：${name}`)
    if (names.has(name)) throw new Error(`Task spec.volumes.name 不能重复：${name}`)
    names.add(name)
    const sourceKeys = Object.keys(item).filter((key) => key !== 'name')
    if (sourceKeys.length === 0) throw new Error(`Task spec.volumes ${name} 必须配置卷来源`)
    if (!sourceKeys.some((key) => taskVolumeSourceKeys.has(key))) {
      throw new Error(`Task spec.volumes ${name} 卷来源不支持`)
    }
  }
}

const toTaskParam = (item: any): TektonTaskParam => ({
  name: String(item?.name || '').trim(),
  type: supportedParamType(item?.type),
  default: valueToFormString(item?.default),
  description: String(item?.description || '').trim(),
  enum: Array.isArray(item?.enum)
    ? item.enum.map((value: unknown) => String(value || '').trim()).filter(Boolean)
    : [],
  properties: isPlainObject(item?.properties) ? normalizeParamProperties(item.properties) : {},
  required: !Object.prototype.hasOwnProperty.call(item || {}, 'default')
})

const supportedParamType = (value: unknown): TektonParamType =>
  ['string', 'array', 'object'].includes(String(value || ''))
    ? (value as TektonParamType)
    : 'string'

const validateRawTaskParamContract = (spec: Record<string, any>, field: 'params' | 'results') => {
  const items = spec[field]
  if (!Array.isArray(items)) return ''
  const label = field === 'params' ? 'Task 参数' : 'Task Result'
  for (const item of items) {
    if (!isPlainObject(item)) continue
    const name = String(item.name || '').trim() || '未命名'
    const type = String(item.type || '').trim()
    if (type && !['string', 'array', 'object'].includes(type)) {
      return `${label} ${name} type 只支持 string、array、object`
    }
    const hasProperties = isPlainObject(item.properties) && Object.keys(item.properties).length > 0
    if (hasProperties && type !== 'object') {
      return `${label} ${name} 配置 properties 时类型必须是 object`
    }
    if (type !== 'object') continue
    if (field === 'params' && name.includes('.')) {
      return `${label} ${name} 类型为 object 时名称不能包含点号`
    }
    if (!hasProperties) {
      return `${label} ${name} 类型为 object 时必须配置 properties`
    }
    for (const [propertyName, property] of Object.entries(item.properties)) {
      if (!String(propertyName || '').trim()) return `${label} ${name} 的 properties 名称不能为空`
      if (!isPlainObject(property))
        return `${label} ${name} 的 properties ${propertyName} 必须是对象`
      const propertyType = String(property.type || '').trim()
      if (propertyType && !['string', 'array', 'object'].includes(propertyType)) {
        return `${label} ${name} 的 properties 类型不支持`
      }
    }
  }
  return ''
}

const validateRawTaskObjectNames = (spec: Record<string, any>) => {
  const checks: Array<{ field: string; items: unknown }> = [
    { field: 'steps', items: spec.steps },
    { field: 'sidecars', items: spec.sidecars },
    { field: 'volumes', items: spec.volumes }
  ]
  for (const check of checks) {
    if (!Array.isArray(check.items)) continue
    const names = new Set<string>()
    for (const item of check.items) {
      if (!isPlainObject(item)) continue
      const name = String(item.name || '').trim()
      if (!name) return `Task spec.${check.field}.name 不能为空`
      if (!isDnsLabel(name)) return `Task spec.${check.field}.name 不符合 DNS_LABEL 规范：${name}`
      if (names.has(name)) return `Task spec.${check.field}.name 不能重复：${name}`
      names.add(name)
    }
  }
  return ''
}

const normalizeParamProperties = (properties: Record<string, any>) =>
  Object.entries(properties).reduce<Record<string, { type?: TektonParamType }>>(
    (acc, [key, value]) => {
      const name = String(key || '').trim()
      if (!name || !isPlainObject(value)) return acc
      acc[name] = { type: supportedParamType(value.type) }
      return acc
    },
    {}
  )

const toWorkspace = (item: any): TektonWorkspace => ({
  name: String(item?.name || '').trim(),
  description: String(item?.description || '').trim(),
  optional: item?.optional === true,
  readOnly: item?.readOnly === true,
  mountPath: String(item?.mountPath || '').trim()
})

const toTaskStep = (item: any): TektonTaskStep => ({
  name: String(item?.name || '').trim(),
  image: String(item?.image || '').trim(),
  refText: valueToFormString(item?.ref),
  script: String(item?.script || ''),
  commandText: Array.isArray(item?.command) ? item.command.join('\n') : String(item?.command || ''),
  argsText: Array.isArray(item?.args) ? item.args.join('\n') : String(item?.args || ''),
  workingDir: String(item?.workingDir || '').trim(),
  advancedRaw: stepAdvancedFieldsText(item)
})

const toTaskResult = (item: any): TektonTaskResult => ({
  name: String(item?.name || '').trim(),
  type: supportedParamType(item?.type),
  properties: isPlainObject(item?.properties) ? normalizeParamProperties(item.properties) : {},
  value: valueToFormString(item?.value),
  description: String(item?.description || '').trim()
})

const hasDuplicateName = (items: Array<{ name?: string }>) => {
  const names = new Set<string>()
  for (const item of items) {
    const name = String(item.name || '').trim()
    if (!name) continue
    if (names.has(name)) return true
    names.add(name)
  }
  return false
}

const stepUsesRef = (item: any) => {
  if (!isPlainObject(item)) return false
  const ref = item.ref
  if (typeof ref === 'string') return ref.trim() !== ''
  return isPlainObject(ref) && Object.keys(ref).length > 0
}

const hasFieldValue = (item: Record<string, any>, key: string) => {
  const value = item[key]
  if (value === undefined || value === null) return false
  if (typeof value === 'string') return value.trim() !== ''
  if (Array.isArray(value)) return value.length > 0
  if (isPlainObject(value)) return Object.keys(value).length > 0
  return true
}

const stepSimpleFields = ['name', 'image', 'ref', 'script', 'command', 'args', 'workingDir']

const stepRefConflictFieldNames = [
  'image',
  'command',
  'args',
  'script',
  'workingDir',
  'env',
  'envFrom',
  'volumeMounts',
  'volumeDevices',
  'computeResources',
  'resources',
  'securityContext',
  'timeout',
  'onError',
  'stdoutConfig',
  'stderrConfig',
  'results',
  'imagePullPolicy',
  'ports',
  'livenessProbe',
  'readinessProbe',
  'startupProbe'
]

const stepRefConflictFields = (item: Record<string, any>) =>
  stepRefConflictFieldNames.filter((key) => hasFieldValue(item, key))

const stripStepRefConflictFields = (item: Record<string, any>) => {
  stepRefConflictFields(item).forEach((key) => {
    delete item[key]
  })
}

const hasUnknownKeys = (item: any, allowed: string[]) => {
  if (!isPlainObject(item)) return false
  return Object.keys(item).some((key) => !allowed.includes(key))
}

const stepAdvancedFields = (item: any) => {
  if (!isPlainObject(item)) return {}
  return Object.entries(item).reduce<Record<string, any>>((acc, [key, value]) => {
    if (stepSimpleFields.includes(key)) return acc
    acc[key] = value
    return acc
  }, {})
}

const stepAdvancedFieldsText = (item: any) => {
  const advanced = stepAdvancedFields(item)
  return Object.keys(advanced).length > 0 ? valueToFormString(advanced) : ''
}

const parseStepAdvancedFields = (raw: string | undefined, stepName: string) => {
  const parsed = parseOptionalYamlField(raw, 'object', `Task Step ${stepName || '-'} 高级字段`)
  const advanced = isPlainObject(parsed) ? parsed : {}
  const conflicts = Object.keys(advanced).filter((key) => stepSimpleFields.includes(key))
  if (conflicts.length > 0) {
    throw new Error(
      `Task Step ${stepName || '-'} 高级字段不能配置 ${conflicts.join('、')}，请在基础字段中维护`
    )
  }
  validateStepAdvancedExecutionFields(advanced, stepName)
  return advanced
}

const validateStepAdvancedExecutionFields = (advanced: Record<string, any>, stepName: string) => {
  const name = stepName || '-'
  const onError = String(advanced.onError || '').trim()
  if (onError && onError !== 'continue' && onError !== 'stopAndFail') {
    throw new Error(`Task Step ${name} onError 只支持 continue 或 stopAndFail`)
  }
  const timeout = String(advanced.timeout || '').trim()
  if (timeout && !tektonDurationPattern.test(timeout)) {
    throw new Error(
      `Task Step ${name} timeout 格式不合法，请使用 30m、1h、1h30m 这类 Tekton duration`
    )
  }
}

const detectAdvancedFields = (resource: Record<string, any>) => {
  const metadata = isPlainObject(resource.metadata) ? resource.metadata : {}
  const spec = isPlainObject(resource.spec) ? resource.spec : {}
  if (hasUnknownKeys(resource, ['apiVersion', 'kind', 'metadata', 'spec'])) return true
  if (hasUnknownKeys(metadata, ['name', 'namespace', 'labels', 'annotations'])) return true
  if (
    hasUnknownKeys(spec, [
      'displayName',
      'description',
      'params',
      'workspaces',
      'results',
      'steps',
      'stepTemplate',
      'sidecars',
      'volumes'
    ])
  )
    return true
  if (
    (spec.params || []).some((item: any) =>
      hasUnknownKeys(item, ['name', 'type', 'default', 'description', 'enum', 'properties'])
    )
  ) {
    return true
  }
  if (
    (spec.workspaces || []).some((item: any) =>
      hasUnknownKeys(item, ['name', 'description', 'optional', 'readOnly', 'mountPath'])
    )
  ) {
    return true
  }
  if (
    (spec.results || []).some((item: any) =>
      hasUnknownKeys(item, ['name', 'description', 'type', 'properties', 'value'])
    )
  ) {
    return true
  }
  return (spec.steps || []).some((item: any) =>
    hasUnknownKeys(item, ['name', 'image', 'ref', 'script', 'command', 'args', 'workingDir'])
  )
}

export const dumpTaskYaml = (resource: Record<string, any>) =>
  stringify(resource, { indent: 2, lineWidth: 120 }).trimEnd() + '\n'

export const buildTaskResource = (
  taskForm: TektonTaskForm,
  rawResource: Record<string, any>,
  fallbackName: string
) => {
  const base = cloneValue(rawResource)
  const baseMetadata = isPlainObject(base.metadata) ? cloneValue(base.metadata) : {}
  const baseSpec = isPlainObject(base.spec) ? cloneValue(base.spec) : {}
  const paramMap = listByName(specList(base, 'params'))
  const workspaceMap = listByName(specList(base, 'workspaces'))
  const resultMap = listByName(specList(base, 'results'))
  const stepMap = listByName(specList(base, 'steps'))

  delete baseMetadata.namespace

  return cleanObject({
    ...base,
    apiVersion: taskForm.apiVersion || 'tekton.dev/v1',
    kind: 'Task',
    metadata: {
      ...baseMetadata,
      name: normalizeTektonName(fallbackName),
      labels: parseOptionalYamlField(taskForm.metadataLabelsRaw, 'object', 'Task metadata.labels'),
      annotations: parseOptionalYamlField(
        taskForm.metadataAnnotationsRaw,
        'object',
        'Task metadata.annotations'
      )
    },
    spec: {
      ...baseSpec,
      displayName: String(taskForm.displayName || '').trim(),
      description: String(taskForm.description || '').trim(),
      stepTemplate: parseOptionalYamlField(
        taskForm.stepTemplateRaw,
        'object',
        'Task spec.stepTemplate'
      ),
      sidecars: parseTaskSidecarsRaw(taskForm.sidecarsRaw),
      volumes: parseTaskVolumesRaw(taskForm.volumesRaw),
      params: taskForm.params.map((item) => {
        const name = String(item.name || '').trim()
        const baseParam = paramMap.get(name)
        return {
          ...(baseParam || {}),
          name,
          type: item.type || 'string',
          default: normalizeDefaultValue(baseParam, item),
          description: String(item.description || '').trim(),
          enum: item.enum?.length ? item.enum : undefined,
          properties:
            item.type === 'object' && item.properties && Object.keys(item.properties).length > 0
              ? item.properties
              : undefined
        }
      }),
      workspaces: taskForm.workspaces.map((item) => {
        const name = String(item.name || '').trim()
        return {
          ...(workspaceMap.get(name) || {}),
          name,
          description: String(item.description || '').trim(),
          optional: item.optional || undefined,
          readOnly: item.readOnly || undefined,
          mountPath: String(item.mountPath || '').trim()
        }
      }),
      results: taskForm.results.map((item) => {
        const name = String(item.name || '').trim()
        return {
          ...(resultMap.get(name) || {}),
          name,
          type: item.type || 'string',
          properties:
            item.type === 'object' && item.properties && Object.keys(item.properties).length > 0
              ? item.properties
              : undefined,
          value: normalizeTaskResultValue(resultMap.get(name), item),
          description: String(item.description || '').trim()
        }
      }),
      steps: taskForm.steps.map((item) => {
        const name = String(item.name || '').trim()
        const baseStep = stepMap.get(name)
        const ref = normalizeStepRefValue(baseStep, item)
        const advancedStep = parseStepAdvancedFields(item.advancedRaw, name)
        const next = {
          ...advancedStep,
          name,
          image: String(item.image || '').trim(),
          ref,
          workingDir: String(item.workingDir || '').trim(),
          script: item.script,
          command: splitLines(item.commandText),
          args: splitLines(item.argsText)
        }
        if (ref) {
          stripStepRefConflictFields(next)
        }
        return next
      })
    }
  })
}

export const normalizeTaskYamlForSave = (content: string, code: string): TaskParseResponse => {
  const parsed = parseTaskYaml(content)
  if (!parsed.ok) return parsed
  const resource = cloneValue(parsed.data.resource)
  const metadata = isPlainObject(resource.metadata) ? cloneValue(resource.metadata) : {}
  metadata.name = normalizeTektonName(code)
  delete metadata.namespace
  resource.metadata = metadata
  const normalized = parseTaskYaml(dumpTaskYaml(resource))
  return normalized.ok ? normalized : parsed
}

export const parseTaskYaml = (content: string): TaskParseResponse => {
  if (!String(content || '').trim()) {
    return { ok: false, error: 'YAML 不能为空' }
  }
  const docs = parseAllDocuments(content, { prettyErrors: true, uniqueKeys: true })
  const errorDoc = docs.find((item) => item.errors.length > 0)
  if (errorDoc) {
    return { ok: false, error: yamlErrorText(errorDoc.errors[0]) }
  }
  const resources = docs
    .map((doc) => doc.toJS({ maxAliasCount: 100 }) as any)
    .filter((item) => item !== undefined && item !== null && item !== '')
  if (resources.length === 0) {
    return { ok: false, error: 'YAML 不能为空' }
  }
  const taskResources = resources.filter((item) => isPlainObject(item) && item.kind === 'Task')
  if (taskResources.length === 0) {
    return { ok: false, error: resources.length === 1 ? 'YAML 必须是 Task' : 'YAML 中未找到 Task' }
  }
  if (taskResources.length > 1) {
    return { ok: false, error: 'YAML 只能包含一个 Task' }
  }
  const parsed = taskResources[0]
  if (!isPlainObject(parsed)) {
    return { ok: false, error: 'YAML 必须是对象' }
  }
  const apiVersion = String(parsed.apiVersion || '')
  if (!supportedApiVersions.includes(apiVersion)) {
    return { ok: false, error: 'apiVersion 只支持 tekton.dev/v1 或 tekton.dev/v1beta1' }
  }
  const spec = isPlainObject(parsed.spec) ? parsed.spec : {}
  const metadata = isPlainObject(parsed.metadata) ? parsed.metadata : {}
  const rawParamError = validateRawTaskParamContract(spec, 'params')
  if (rawParamError) return { ok: false, error: rawParamError }
  const rawResultError = validateRawTaskParamContract(spec, 'results')
  if (rawResultError) return { ok: false, error: rawResultError }
  const rawNameError = validateRawTaskObjectNames(spec)
  if (rawNameError) return { ok: false, error: rawNameError }
  const form: TektonTaskForm = {
    apiVersion,
    name: String(metadata.name || ''),
    displayName: String(spec.displayName || '').trim(),
    description: String(spec.description || '').trim(),
    metadataLabelsRaw: valueToFormString(metadata.labels),
    metadataAnnotationsRaw: valueToFormString(metadata.annotations),
    stepTemplateRaw: valueToFormString(spec.stepTemplate),
    sidecarsRaw: valueToFormString(spec.sidecars),
    volumesRaw: valueToFormString(spec.volumes),
    params: Array.isArray(spec.params) ? spec.params.map(toTaskParam) : [],
    workspaces: Array.isArray(spec.workspaces) ? spec.workspaces.map(toWorkspace) : [],
    results: Array.isArray(spec.results) ? spec.results.map(toTaskResult) : [],
    steps: Array.isArray(spec.steps) ? spec.steps.map(toTaskStep) : []
  }
  if (!form.name) {
    return { ok: false, error: 'metadata.name 不能为空' }
  }
  if (!isDnsLabel(form.name)) {
    return { ok: false, error: `metadata.name 不符合 DNS_LABEL 规范：${form.name}` }
  }
  for (const step of Array.isArray(spec.steps) ? spec.steps : []) {
    if (!stepUsesRef(step)) continue
    const conflicts = stepRefConflictFields(step)
    if (conflicts.length > 0) {
      return {
        ok: false,
        error: `Task Step 引用 ref 时不能同时配置 ${conflicts.join('、')}`
      }
    }
  }
  return {
    ok: true,
    data: {
      resource: parsed,
      form,
      hasAdvancedFields: detectAdvancedFields(parsed),
      normalizedYaml: dumpTaskYaml(parsed)
    }
  }
}

export const validateTaskForSave = (form: TektonTaskForm) => {
  if (!String(form.name || '').trim()) return 'Task metadata.name 不能为空'
  if (!isDnsLabel(String(form.name || '').trim())) {
    return `Task metadata.name 不符合 DNS_LABEL 规范：${String(form.name || '').trim()}`
  }
  const advancedError = validateTaskAdvancedFields(form)
  if (advancedError) return advancedError
  if (form.steps.length === 0) return 'Task spec.steps 不能为空'
  const invalidStepName = form.steps.find((item) => {
    const name = String(item.name || '').trim()
    return name && !isDnsLabel(name)
  })
  if (invalidStepName) {
    return `Task spec.steps.name 不符合 DNS_LABEL 规范：${String(invalidStepName.name || '').trim()}`
  }
  if (
    form.steps.some(
      (item) =>
        !String(item.name || '').trim() ||
        (!String(item.image || '').trim() && !String(item.refText || '').trim())
    )
  ) {
    return 'Task spec.steps 的 name 和 image/ref 不能为空'
  }
  if (
    form.steps.some((item) => String(item.image || '').trim() && String(item.refText || '').trim())
  ) {
    return 'Task spec.steps 不能同时配置 image 和 ref'
  }
  if (form.params.some((item) => !String(item.name || '').trim())) {
    return 'Task spec.params.name 不能为空'
  }
  if (hasDuplicateName(form.params)) return 'Task spec.params.name 不能重复'
  for (const param of form.params) {
    const name = String(param.name || '').trim()
    if (param.enum?.some((item) => !String(item || '').trim())) {
      return `Task 参数 ${name} 的 enum 不能为空`
    }
    if (param.properties && Object.keys(param.properties).length > 0 && param.type !== 'object') {
      return `Task 参数 ${name} 配置 properties 时类型必须是 object`
    }
    if (param.type === 'object' && name.includes('.')) {
      return `Task 参数 ${name} 类型为 object 时名称不能包含点号`
    }
    if (param.type === 'object' && Object.keys(param.properties || {}).length === 0) {
      return `Task 参数 ${name} 类型为 object 时必须配置 properties`
    }
    try {
      parseTypedDefaultValue(param.type, param.default || '', `Task 参数 ${name} 默认值`)
    } catch (error: any) {
      return error?.message || `Task 参数 ${name} 默认值不合法`
    }
    for (const [propertyName, property] of Object.entries(param.properties || {})) {
      if (!String(propertyName || '').trim()) return `Task 参数 ${name} 的 properties 名称不能为空`
      if (property.type && !['string', 'array', 'object'].includes(property.type)) {
        return `Task 参数 ${name} 的 properties 类型不支持`
      }
    }
  }
  if (form.workspaces.some((item) => !String(item.name || '').trim())) {
    return 'Task spec.workspaces.name 不能为空'
  }
  if (hasDuplicateName(form.workspaces)) return 'Task spec.workspaces.name 不能重复'
  if (form.results.some((item) => !String(item.name || '').trim())) {
    return 'Task spec.results.name 不能为空'
  }
  if (hasDuplicateName(form.results)) return 'Task spec.results.name 不能重复'
  for (const result of form.results) {
    const name = String(result.name || '').trim()
    if (
      result.properties &&
      Object.keys(result.properties).length > 0 &&
      result.type !== 'object'
    ) {
      return `Task Result ${name} 配置 properties 时类型必须是 object`
    }
    if (result.type === 'object' && Object.keys(result.properties || {}).length === 0) {
      return `Task Result ${name} 类型为 object 时必须配置 properties`
    }
    for (const [propertyName, property] of Object.entries(result.properties || {})) {
      if (!String(propertyName || '').trim())
        return `Task Result ${name} 的 properties 名称不能为空`
      if (property.type && !['string', 'array', 'object'].includes(property.type)) {
        return `Task Result ${name} 的 properties 类型不支持`
      }
    }
  }
  if (hasDuplicateName(form.steps)) return 'Task spec.steps.name 不能重复'
  for (const step of form.steps) {
    const name = String(step.name || '').trim()
    let advanced: Record<string, any>
    try {
      advanced = parseStepAdvancedFields(step.advancedRaw, name)
    } catch (error: any) {
      return error?.message || `Task Step ${name || '-'} 高级字段不合法`
    }
    if (String(step.refText || '').trim()) {
      const conflicts = stepRefConflictFields(advanced)
      if (conflicts.length > 0) {
        return `Task Step ${name || '-'} 引用 ref 时不能同时配置 ${conflicts.join('、')}`
      }
    }
  }
  return ''
}

export const validateTaskAdvancedFields = (form: TektonTaskForm) => {
  try {
    parseOptionalYamlField(form.metadataLabelsRaw, 'object', 'Task metadata.labels')
    parseOptionalYamlField(form.metadataAnnotationsRaw, 'object', 'Task metadata.annotations')
    parseOptionalYamlField(form.stepTemplateRaw, 'object', 'Task spec.stepTemplate')
    parseTaskSidecarsRaw(form.sidecarsRaw)
    parseTaskVolumesRaw(form.volumesRaw)
    for (const step of form.steps) {
      const name = String(step.name || '').trim()
      const advanced = parseStepAdvancedFields(step.advancedRaw, name)
      if (String(step.refText || '').trim()) {
        const conflicts = stepRefConflictFields(advanced)
        if (conflicts.length > 0) {
          throw new Error(
            `Task Step ${name || '-'} 引用 ref 时不能同时配置 ${conflicts.join('、')}`
          )
        }
      }
    }
  } catch (error: any) {
    return error?.message || 'Task 高级字段不合法'
  }
  return ''
}

const collectImageValues = (input: unknown, result: Set<string>) => {
  if (Array.isArray(input)) {
    input.forEach((item) => collectImageValues(item, result))
    return
  }
  if (!isPlainObject(input)) return
  Object.entries(input).forEach(([key, value]) => {
    if (key === 'image' && typeof value === 'string' && value.trim()) {
      result.add(value.trim())
      return
    }
    collectImageValues(value, result)
  })
}

const replaceImageValues = (input: unknown, from: string, to: string): boolean => {
  let changed = false
  if (Array.isArray(input)) {
    input.forEach((item) => {
      if (replaceImageValues(item, from, to)) changed = true
    })
    return changed
  }
  if (!isPlainObject(input)) return false
  Object.entries(input).forEach(([key, value]) => {
    if (key === 'image' && value === from) {
      input[key] = to
      changed = true
      return
    }
    if (replaceImageValues(value, from, to)) changed = true
  })
  return changed
}

export const collectTaskImages = (resource: Record<string, any>) => {
  const result = new Set<string>()
  collectImageValues(resource, result)
  return Array.from(result).sort()
}

export const collectTaskImagesFromYaml = (content: string) => {
  const parsed = parseTaskYaml(content)
  return parsed.ok ? collectTaskImages(parsed.data.resource) : []
}

export const replaceTaskImageInYaml = (
  content: string,
  from: string,
  to: string
): TaskYamlMutationResponse => {
  const oldImage = String(from || '').trim()
  const nextImage = String(to || '').trim()
  if (!oldImage || !nextImage) return { ok: false, error: '镜像地址不能为空' }
  const parsed = parseTaskYaml(content)
  if (!parsed.ok) return { ok: false, error: parsed.error }
  const resource = cloneValue(parsed.data.resource)
  const changed = replaceImageValues(resource, oldImage, nextImage)
  return {
    ok: true,
    content: dumpTaskYaml(resource),
    images: collectTaskImages(resource),
    changed
  }
}
