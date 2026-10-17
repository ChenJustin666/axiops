import { parse } from 'yaml'
import type { TektonDagConfig, TektonDagNode, TektonDagValidationIssue } from './types'

export interface TektonDagValidationOptions {
  paramValues?: Record<string, string | undefined>
  workspaceValues?: Record<string, string | undefined>
  requireWorkspaceValues?: boolean
}

const namePattern = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/
const paramPattern = /^[A-Za-z_][A-Za-z0-9_.-]*$/
const taskKindPattern = /^[A-Za-z][A-Za-z0-9.]*$/
const apiVersionPattern = /^[A-Za-z0-9_.-]+(\/[A-Za-z0-9_.-]+)?$/
const tektonDurationPattern = /^([0-9]+(ms|h|m|s))+$/i
const supportedWorkspaceBindings = new Set([
  'emptyDir',
  'persistentVolumeClaim',
  'volumeClaimTemplate',
  'secret',
  'generatedSecret',
  'configMap',
  'projected'
])
const supportedWorkspaceSources = new Set([
  'kubernetesResource',
  'platformCredential',
  'configCenter'
])
const supportedPipelinePlatformTypes = new Set([
  'string',
  'text',
  'booleanParam',
  'number',
  'password',
  'file',
  'choice',
  'objectList',
  'list',
  'channelGroupCode',
  'voucherModel',
  'channelCredential',
  'configCenter',
  'hostGroupTargets',
  'hostConfig',
  'hostGroupConfig',
  'gitProject',
  'gitBranch',
  'gitTag',
  'gitRepositoryUrl',
  'harborProject',
  'harborImage',
  'harborImageTag',
  'harborProjectUrl',
  'harborImageUrl',
  'harborImageTagUrl',
  'registryRepository',
  'registryImage',
  'registryImageTag',
  'registryRepositoryUrl',
  'registryImageUrl',
  'registryImageTagUrl',
  'nexusRepository',
  'nexusArtifactName',
  'nexusArtifactVersion',
  'nexusArtifact',
  'nexusRepositoryUrl',
  'nexusArtifactUrl',
  'nexusArtifactVersionUrl',
  'jfrogRepository',
  'jfrogArtifactName',
  'jfrogArtifactVersion',
  'jfrogArtifact',
  'jfrogRepositoryUrl',
  'jfrogArtifactUrl',
  'jfrogArtifactVersionUrl',
  'sonarProjectName',
  'sonarProjectKey',
  'sonarProjectUrl',
  'sonarProjectKeyUrl',
  'kubernetesNamespace',
  'kubernetesWorkloadType',
  'kubernetesResource',
  'kubernetesContainer',
  'kubernetesImage',
  'kubeNovaProject',
  'kubeNovaCluster',
  'kubeNovaWorkspace',
  'kubeNovaApplication',
  'kubeNovaVersion',
  'kubeNovaContainer',
  'kubeNovaDeployConfig',
  'kubernetesDeployConfig',
  'kubernetesSecretName',
  'kubernetesPvcName'
])

function tektonTypeForPlatformParamType(paramType?: string) {
  switch (String(paramType || '').trim()) {
    case 'array':
    case 'list':
      return 'array'
    case 'object':
    case 'objectList':
      return 'object'
    case '':
      return ''
    default:
      return 'string'
  }
}

const isBlank = (value?: string) => !String(value || '').trim()
const isNonEmptyStringValue = (value: unknown) =>
  typeof value === 'string' && String(value || '').trim() !== ''

const parseParamValue = (value?: string) => {
  try {
    return parse(String(value || ''))
  } catch {
    return undefined
  }
}

function hasListValue(value?: string) {
  const text = String(value || '').trim()
  if (!text) return false
  const parsed = parseParamValue(text)
  if (Array.isArray(parsed)) return true
  return (
    text
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean).length > 0
  )
}

function hasObjectValue(value?: string) {
  const text = String(value || '').trim()
  if (!text) return false
  const parsed = parseParamValue(text)
  if (Array.isArray(parsed)) return false
  if (parsed && typeof parsed === 'object') return true
  return false
}

function hasPipelineParamValue(param: TektonDagConfig['params'][number], value?: string) {
  if (param.type === 'array' || param.paramType === 'list') return hasListValue(value)
  if (param.type === 'object' || param.paramType === 'objectList') return hasObjectValue(value)
  return !isBlank(value)
}

export const validateTektonName = (value?: string) => {
  const name = String(value || '').trim()
  return name.length <= 63 && namePattern.test(name)
}
const validateParamName = (value?: string) => paramPattern.test(String(value || '').trim())
const validateScheduleTimezone = (value?: string) =>
  !String(value || '').trim() ||
  /^[A-Za-z_]+\/[A-Za-z0-9_+-]+(?:\/[A-Za-z0-9_+-]+)*$/.test(String(value || '').trim())

export function validateTektonDagConfig(
  config: TektonDagConfig,
  options: TektonDagValidationOptions = {}
): TektonDagValidationIssue[] {
  const issues: TektonDagValidationIssue[] = []
  validatePipeline(config, issues)
  validatePipelineParams(config, issues, options)
  validatePipelineResults(config, issues)
  validatePipelineWorkspaces(config, issues, options)
  validateNodes(config, issues, options)
  validateEdges(config, issues)
  validateOptionalPolicies(config, issues)

  detectCycles(config).forEach((nodeId) => {
    issues.push({ level: 'error', target: nodeId, message: 'DAG 存在循环依赖' })
  })

  return issues
}

function validateOptionalPolicies(config: TektonDagConfig, issues: TektonDagValidationIssue[]) {
  validateTimeoutMap(config.pipeline.timeouts, 'Pipeline timeouts', 'pipeline', issues)
  validateDuration(config.pipeline.timeout, 'Pipeline Timeout', 'pipeline', issues)
  const runPolicy = config.runPolicy
  if (runPolicy) {
    ;[...(runPolicy.labels || []), ...(runPolicy.annotations || [])].forEach((item) => {
      if (isBlank(item.name) || isBlank(item.value)) {
        issues.push({
          level: 'error',
          target: 'runPolicy',
          message: 'PipelineRun 元数据必须同时填写 Key 和 Value'
        })
      }
    })
    validateTimeoutMap(runPolicy.timeouts, 'PipelineRun timeouts', 'runPolicy', issues)
    validateDuration(runPolicy.timeout, 'PipelineRun Timeout', 'runPolicy', issues)
    ;(runPolicy.podTemplateEnv || []).forEach((item) => {
      if (isBlank(item.name) || isBlank(item.value)) {
        issues.push({
          level: 'error',
          target: 'runPolicy',
          message: 'podTemplate 环境变量必须同时填写名称和值'
        })
      }
    })
    validateYamlObjectRaw(runPolicy.podTemplateRaw, 'podTemplate', issues)
    validateYamlObjectRaw(runPolicy.taskRunTemplateRaw, 'taskRunTemplate', issues)
    validateTaskRunSpecsRaw(config, runPolicy.taskRunSpecsRaw, issues)
    validateTaskRunSpecsItems(config, runPolicy.taskRunSpecs, issues)
  }

  const trigger = config.triggerConfig
  if (trigger?.enabled) {
    const required = [
      ['Namespace', trigger.namespace],
      ['EventListener', trigger.eventListenerName],
      ['Trigger', trigger.triggerName],
      ['TriggerBinding', trigger.triggerBindingName],
      ['TriggerTemplate', trigger.triggerTemplateName]
    ]
    required.forEach(([label, value]) => {
      if (isBlank(value)) {
        issues.push({ level: 'error', target: 'trigger', message: `Trigger 必须配置 ${label}` })
      } else if (!validateTektonName(value)) {
        issues.push({
          level: 'error',
          target: 'trigger',
          message: `Trigger ${label} 名称不符合 Kubernetes 命名规范`
        })
      }
    })
    if (!isBlank(trigger.serviceAccountName) && !validateTektonName(trigger.serviceAccountName)) {
      issues.push({
        level: 'error',
        target: 'trigger',
        message: 'Trigger ServiceAccount 名称不符合 Kubernetes 命名规范'
      })
    }
    if (trigger.interceptorType === 'cel' && isBlank(trigger.celFilter)) {
      issues.push({
        level: 'error',
        target: 'trigger',
        message: 'CEL interceptor 必须配置过滤表达式'
      })
    }
    ;(trigger.bindingParams || []).forEach((item) => {
      if (isBlank(item.name) || isBlank(item.value)) {
        issues.push({
          level: 'error',
          target: 'trigger',
          message: 'TriggerBinding 参数必须同时填写名称和值'
        })
      }
    })
    ;(trigger.templateParams || []).forEach((item) => {
      if (isBlank(item.name)) {
        issues.push({
          level: 'error',
          target: 'trigger',
          message: 'TriggerTemplate 参数必须填写名称'
        })
      }
    })
    ;[...(trigger.labels || []), ...(trigger.annotations || [])].forEach((item) => {
      if (isBlank(item.name) || isBlank(item.value)) {
        issues.push({
          level: 'error',
          target: 'trigger',
          message: 'Trigger 资源元数据必须同时填写 Key 和 Value'
        })
      }
    })
  }

  const schedule = config.scheduleConfig
  if (schedule?.enabled) {
    if (isBlank(schedule.cron) || isBlank(schedule.timezone)) {
      issues.push({
        level: 'error',
        target: 'schedule',
        message: '平台定时必须配置 Cron 和时区'
      })
    }
    const cron = String(schedule.cron || '').trim()
    if (cron && !cron.startsWith('@') && cron.split(/\s+/).length !== 5) {
      issues.push({
        level: 'error',
        target: 'schedule',
        message: '平台定时 Cron 需要使用 5 段标准表达式'
      })
    }
    if (!validateScheduleTimezone(schedule.timezone)) {
      issues.push({
        level: 'error',
        target: 'schedule',
        message: '平台定时时区格式不合法，例如 Asia/Shanghai'
      })
    }
  }

  const pruner = config.prunerPolicy
  if (pruner?.enabled) {
    if (!pruner.resourceTypes?.length) {
      issues.push({ level: 'error', target: 'pruner', message: 'Pruner 必须选择清理资源类型' })
    }
    if (pruner.clusterResourceCleanup === false && pruner.platformRecordCleanup === false) {
      issues.push({ level: 'error', target: 'pruner', message: 'Pruner 至少需要启用一种清理动作' })
    }
    if (
      pruner.nativePrunerMode &&
      !['disabled', 'namespaceConfigMap', 'globalConfigMap'].includes(pruner.nativePrunerMode)
    ) {
      issues.push({ level: 'error', target: 'pruner', message: 'Tekton 原生 Pruner 模式不支持' })
    }
    if (
      pruner.nativePrunerMode !== 'disabled' &&
      !isBlank(pruner.nativePrunerTargetNamespace) &&
      !validateTektonName(pruner.nativePrunerTargetNamespace)
    ) {
      issues.push({
        level: 'error',
        target: 'pruner',
        message: 'Tekton 原生 Pruner 配置 Namespace 不合法'
      })
    }
    ;[...(pruner.selectorLabels || []), ...(pruner.selectorAnnotations || [])].forEach((item) => {
      if (isBlank(item.name) || isBlank(item.value)) {
        issues.push({
          level: 'error',
          target: 'pruner',
          message: 'Pruner 选择器必须同时填写 Key 和 Value'
        })
      }
    })
  }
}

function validateTimeoutMap(
  timeouts: Partial<Record<'pipeline' | 'tasks' | 'finally', string>> | undefined,
  label: string,
  target: string,
  issues: TektonDagValidationIssue[]
) {
  if (!timeouts) return
  Object.entries(timeouts).forEach(([key, value]) => {
    if (!['pipeline', 'tasks', 'finally'].includes(key)) {
      issues.push({
        level: 'error',
        target,
        message: `${label} 只支持 pipeline、tasks、finally`
      })
      return
    }
    validateDuration(value, `${label}.${key}`, target, issues)
  })
}

function validatePipeline(config: TektonDagConfig, issues: TektonDagValidationIssue[]) {
  if (!validateTektonName(config.pipeline.name)) {
    issues.push({
      level: 'error',
      target: 'pipeline',
      message: 'Pipeline 名称不符合 Kubernetes 命名规范'
    })
  }
  if (!isBlank(config.pipeline.namespace) && !validateTektonName(config.pipeline.namespace)) {
    issues.push({
      level: 'error',
      target: 'pipeline',
      message: 'Namespace 不符合 Kubernetes 命名规范'
    })
  }
  if (
    !isBlank(config.pipeline.serviceAccountName) &&
    !validateTektonName(config.pipeline.serviceAccountName)
  ) {
    issues.push({
      level: 'error',
      target: 'pipeline',
      message: 'ServiceAccount 不符合 Kubernetes 命名规范'
    })
  }
  if (
    config.nodes.filter(
      (item) => item.enabled !== false && item.type !== 'note' && item.type !== 'finally'
    ).length === 0
  ) {
    issues.push({ level: 'error', target: 'nodes', message: 'Tekton DAG 至少需要一个启用 Task' })
  }
  ;[...(config.pipeline.labels || []), ...(config.pipeline.annotations || [])].forEach((item) => {
    if (isBlank(item.name) || isBlank(item.value)) {
      issues.push({
        level: 'error',
        target: 'pipeline',
        message: 'Pipeline 元数据必须同时填写 Key 和 Value'
      })
    }
  })
  validateDuration(config.pipeline.timeout, 'Pipeline Timeout', 'pipeline', issues)
  validateDuration(config.runPolicy?.pipelineTimeout, 'Pipeline Timeout', 'runPolicy', issues)
  validateDuration(config.runPolicy?.tasksTimeout, 'Tasks Timeout', 'runPolicy', issues)
  validateDuration(config.runPolicy?.finallyTimeout, 'Finally Timeout', 'runPolicy', issues)
}

function validatePipelineParams(
  config: TektonDagConfig,
  issues: TektonDagValidationIssue[],
  options: TektonDagValidationOptions
) {
  const names = new Set<string>()
  declaredPipelineParams(config).forEach((param) => {
    if (!validateParamName(param.name)) {
      issues.push({ level: 'error', target: 'params', message: 'Pipeline 参数名不合法' })
    }
    if (names.has(param.name)) {
      issues.push({ level: 'error', target: 'params', message: `Pipeline 参数重复：${param.name}` })
    }
    names.add(param.name)
    validatePipelineParamDefaultValue(param, issues)
    if (param.enum?.some((item) => isBlank(item))) {
      issues.push({
        level: 'error',
        target: 'params',
        message: `Pipeline 参数「${param.name}」枚举值不能为空`
      })
    }
    if (param.properties) {
      Object.entries(param.properties).forEach(([key, property]) => {
        if (!validateParamName(key)) {
          issues.push({
            level: 'error',
            target: 'params',
            message: `Pipeline 参数「${param.name}」对象属性名不合法：${key}`
          })
        }
        if (property.type && !['string', 'array', 'object'].includes(property.type)) {
          issues.push({
            level: 'error',
            target: 'params',
            message: `Pipeline 参数「${param.name}」对象属性类型不支持：${key}`
          })
        }
      })
      if (param.type !== 'object') {
        issues.push({
          level: 'error',
          target: 'params',
          message: `Pipeline 参数「${param.name}」配置了 properties，但 Tekton 类型不是 object`
        })
      }
    }
    if (param.type === 'object' && Object.keys(param.properties || {}).length === 0) {
      issues.push({
        level: 'error',
        target: 'params',
        message: `Pipeline 参数「${param.name}」类型为 object 时必须配置 properties`
      })
    }
    if (param.type === 'object' && String(param.name || '').includes('.')) {
      issues.push({
        level: 'error',
        target: 'params',
        message: `Pipeline 参数「${param.name}」类型为 object 时名称不能包含点号`
      })
    }
    if (param.paramType && !supportedPipelinePlatformTypes.has(param.paramType)) {
      issues.push({
        level: 'error',
        target: 'params',
        message: `Pipeline 参数「${param.name}」平台类型已废弃或不支持：${param.paramType}`
      })
    }
    const mappedTektonType = tektonTypeForPlatformParamType(param.paramType)
    if (mappedTektonType && mappedTektonType !== param.type) {
      issues.push({
        level: 'error',
        target: 'params',
        message: `Pipeline 参数「${param.name}」平台类型与 Tekton 类型不匹配`
      })
    }
    validatePipelineParamPlatformConfig(config, param, issues)
    if (param.required && !isPipelineParamValueReady(param, options)) {
      issues.push({
        level: 'error',
        target: 'params',
        message: `必填参数「${param.name}」必须配置默认值、当前值或设为运行时输入`
      })
    }
  })
}

function validatePipelineParamDefaultValue(
  param: TektonDagConfig['params'][number],
  issues: TektonDagValidationIssue[]
) {
  const value = String(param.defaultValue || '').trim()
  if (!value) return
  if (param.type !== 'array' && param.type !== 'object') return
  const error = validateTypedTextValue(
    value,
    param.type,
    `Pipeline ${param.type === 'array' ? '数组' : '对象'}参数「${param.name}」默认值`
  )
  if (error) {
    issues.push({ level: 'error', target: 'params', message: error })
  }
}

function validateTypedTextValue(
  value: string | undefined,
  type: string | undefined,
  label: string
) {
  const text = String(value || '').trim()
  if (!text) return ''
  if (type === 'array') {
    const parsed = parseParamValue(text)
    if (Array.isArray(parsed) && parsed.some((item) => typeof item !== 'string')) {
      return `${label} 必须是字符串数组`
    }
    const hasCommaList =
      text
        .split(',')
        .map((item) => item.trim())
        .filter(Boolean).length > 0
    if (!Array.isArray(parsed) && !hasCommaList) return `${label} 必须是 YAML 数组或逗号分隔列表`
  }
  if (type === 'object') {
    const parsed = parseParamValue(text)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return `${label} 必须是 YAML 对象`
    }
  }
  return ''
}

function validatePipelineParamPlatformConfig(
  config: TektonDagConfig,
  param: TektonDagConfig['params'][number],
  issues: TektonDagValidationIssue[]
) {
  const paramByName = new Map(declaredPipelineParams(config).map((item) => [item.name, item]))
  const itemConfig = param.config || {}
  const name = param.name || '-'
  switch (param.paramType) {
    case 'channelGroupCode': {
      if (isBlank(itemConfig.channelGroupCode)) {
        issues.push({
          level: 'error',
          target: 'params',
          message: `渠道变量「${name}」必须配置渠道分组`
        })
      }
      if (isBlank(itemConfig.channelTypeFilter)) {
        issues.push({
          level: 'error',
          target: 'params',
          message: `渠道变量「${name}」必须配置渠道类型`
        })
      }
      if (isBlank(itemConfig.mappingField)) {
        issues.push({
          level: 'error',
          target: 'params',
          message: `渠道变量「${name}」必须配置映射字段`
        })
      }
      const channelParamCode = String(itemConfig.channelParamCode || '').trim()
      if (channelParamCode) {
        const channelParam = paramByName.get(channelParamCode)
        if (!channelParam || !sameChannelVariableDependency(itemConfig, 'endpoint', channelParam)) {
          issues.push({
            level: 'error',
            target: 'params',
            message: `渠道变量「${name}」渠道实例不匹配`
          })
        }
      }
      const dependencyFields =
        isCodeRepoGitRefConfig(itemConfig) &&
        hasSameChannelVariableDependency(itemConfig, 'address.projectUrl', paramByName)
          ? []
          : requiredChannelVariableDependencies(itemConfig)
      dependencyFields.forEach((field) => {
        const depCode = (itemConfig.dependencyParamCodes || []).find(
          (dep) => dep.field === field
        )?.paramCode
        if (!String(depCode || '').trim()) return
        const depParam = paramByName.get(String(depCode || '').trim())
        if (!depParam) {
          issues.push({
            level: 'error',
            target: 'params',
            message: `渠道变量「${name}」必须选择依赖字段 ${field}`
          })
        } else if (!sameChannelVariableDependency(itemConfig, field, depParam)) {
          issues.push({
            level: 'error',
            target: 'params',
            message: `渠道变量「${name}」依赖字段 ${field} 不匹配`
          })
        }
      })
      break
    }
    case 'voucherModel':
      if (isBlank(itemConfig.voucherModel)) {
        issues.push({
          level: 'error',
          target: 'params',
          message: `平台凭证「${name}」必须配置凭证类型`
        })
      }
      if (credentialMode(itemConfig) === 'field' && isBlank(itemConfig.mappingField)) {
        issues.push({
          level: 'error',
          target: 'params',
          message: `平台凭证「${name}」必须配置凭证字段`
        })
      }
      break
    case 'channelCredential': {
      const sourceCode = String(itemConfig.credentialSourceParamCode || '').trim()
      const sourceParam = paramByName.get(sourceCode)
      if (!sourceParam) {
        issues.push({
          level: 'error',
          target: 'params',
          message: `渠道凭证「${name}」必须选择渠道来源参数`
        })
      } else if (!isCredentialSourceParam(sourceParam)) {
        issues.push({
          level: 'error',
          target: 'params',
          message: `渠道凭证「${name}」来源参数类型不支持`
        })
      }
      if (credentialMode(itemConfig) === 'field' && isBlank(itemConfig.mappingField)) {
        issues.push({
          level: 'error',
          target: 'params',
          message: `渠道凭证「${name}」必须配置凭证字段`
        })
      }
      break
    }
    case 'configCenter':
      if (isBlank(itemConfig.configTypeId) && isBlank(itemConfig.configTypeCode)) {
        issues.push({
          level: 'error',
          target: 'params',
          message: `配置中心「${name}」必须配置配置类型`
        })
      }
      break
    case 'hostGroupTargets': {
      const renderMode = String(itemConfig.renderMode || 'json').trim()
      if (
        !['normal', 'inventory', 'json', 'jsonBase64', 'ansible', 'ansibleBase64'].includes(
          renderMode
        )
      ) {
        issues.push({
          level: 'error',
          target: 'params',
          message: `主机组目标「${name}」输出模式不支持`
        })
      }
      break
    }
  }
}

function credentialMode(config: NonNullable<TektonDagConfig['params'][number]['config']>) {
  const mode = String(config.credentialMode || '').trim()
  return ['field', 'jsonData', 'jsonBase64'].includes(mode) ? mode : 'field'
}

function requiredChannelVariableDependencies(
  config: NonNullable<TektonDagConfig['params'][number]['config']>
) {
  const channelType = String(config.channelTypeFilter || '').trim()
  switch (String(config.mappingField || '').trim()) {
    case 'dynamic.branch':
    case 'dynamic.tag':
    case 'dynamic.cluster':
      return ['dynamic.project']
    case 'dynamic.artifact':
    case 'dynamic.artifactName':
      return ['dynamic.repository']
    case 'dynamic.artifactVersion':
    case 'dynamic.artifactTag':
      return ['dynamic.repository', 'dynamic.artifactName']
    case 'dynamic.image':
      return ['dynamic.repository']
    case 'dynamic.imageTag':
      return ['dynamic.repository', 'dynamic.image']
    case 'dynamic.workspace':
      return ['dynamic.project', 'dynamic.cluster']
    case 'dynamic.application':
      return ['dynamic.workspace']
    case 'dynamic.version':
      return ['dynamic.application']
    case 'dynamic.container':
    case 'dynamic.containerName':
      return channelType === 'kubernetes'
        ? ['dynamic.namespace']
        : ['dynamic.application', 'dynamic.version']
    case 'dynamic.workloadType':
      return []
    case 'dynamic.resourceName':
      return ['dynamic.namespace', 'dynamic.workloadType']
    case 'dynamic.imageName':
      return [
        'dynamic.namespace',
        'dynamic.workloadType',
        'dynamic.resourceName',
        'dynamic.containerName'
      ]
    case 'dynamic.deployment':
    case 'dynamic.statefulSet':
    case 'dynamic.daemonSet':
    case 'dynamic.cronJob':
      return ['dynamic.namespace']
    default:
      return []
  }
}

function isCodeRepoGitRefConfig(config: NonNullable<TektonDagConfig['params'][number]['config']>) {
  return (
    String(config.channelGroupCode || '').trim() === 'code_repo' &&
    ['dynamic.branch', 'dynamic.tag'].includes(String(config.mappingField || '').trim())
  )
}

function hasSameChannelVariableDependency(
  source: NonNullable<TektonDagConfig['params'][number]['config']>,
  field: string,
  paramByName: Map<string, TektonDagConfig['params'][number]>
) {
  if (String(source.projectParamCode || '').trim()) {
    const target = paramByName.get(String(source.projectParamCode || '').trim())
    if (target && sameChannelVariableDependency(source, field, target)) return true
  }
  return (source.dependencyParamCodes || []).some((dep) => {
    const target = paramByName.get(String(dep.paramCode || '').trim())
    return !!target && sameChannelVariableDependency(source, field, target)
  })
}

function sameChannelVariableDependency(
  source: NonNullable<TektonDagConfig['params'][number]['config']>,
  field: string,
  target: TektonDagConfig['params'][number]
) {
  const targetConfig = target.config || {}
  return (
    target.paramType === 'channelGroupCode' &&
    String(targetConfig.channelGroupCode || '').trim() ===
      String(source.channelGroupCode || '').trim() &&
    String(targetConfig.channelTypeFilter || '').trim() ===
      String(source.channelTypeFilter || '').trim() &&
    String(targetConfig.mappingField || '').trim() === field
  )
}

function isCredentialSourceParam(param: TektonDagConfig['params'][number]) {
  if (param.paramType !== 'channelGroupCode') return false
  const field = String(param.config?.mappingField || '').trim()
  return field === 'endpoint' || field.startsWith('address.')
}

function validatePipelineResults(config: TektonDagConfig, issues: TektonDagValidationIssue[]) {
  const names = new Set<string>()
  const resultByTaskName = new Map<string, Set<string>>()
  config.nodes
    .filter((node) => node.enabled !== false && node.type !== 'note')
    .forEach((node) => {
      const taskName = String(node.name || '').trim()
      if (!taskName) return
      resultByTaskName.set(taskName, new Set(node.results || []))
    })
  declaredPipelineResults(config).forEach((result) => {
    const name = String(result.name || '').trim()
    if (!validateParamName(name)) {
      issues.push({ level: 'error', target: 'results', message: 'Pipeline Result 名称不合法' })
    }
    if (names.has(name)) {
      issues.push({
        level: 'error',
        target: 'results',
        message: `Pipeline Result 重复：${name}`
      })
    }
    names.add(name)
    if (result.type && !['string', 'array', 'object'].includes(result.type)) {
      issues.push({
        level: 'error',
        target: 'results',
        message: `Pipeline Result「${name || '-'}」类型不支持`
      })
    }
    if (result.properties && result.type !== 'object') {
      issues.push({
        level: 'error',
        target: 'results',
        message: `Pipeline Result「${name || '-'}」配置了 properties，但 Tekton 类型不是 object`
      })
    }
    if (result.type === 'object' && Object.keys(result.properties || {}).length === 0) {
      issues.push({
        level: 'error',
        target: 'results',
        message: `Pipeline Result「${name || '-'}」类型为 object 时必须配置 properties`
      })
    }
    if (result.properties) {
      Object.entries(result.properties).forEach(([key, property]) => {
        if (!validateParamName(key)) {
          issues.push({
            level: 'error',
            target: 'results',
            message: `Pipeline Result「${name || '-'}」properties 名称不合法：${key}`
          })
        }
        if (property.type && !['string', 'array', 'object'].includes(property.type)) {
          issues.push({
            level: 'error',
            target: 'results',
            message: `Pipeline Result「${name || '-'}」properties 类型不支持：${key}`
          })
        }
      })
    }
    const value = String(result.value || '').trim()
    if (!value) {
      issues.push({
        level: 'error',
        target: 'results',
        message: `Pipeline Result「${name || '-'}」必须配置 value`
      })
      return
    }
    extractPipelineResultRefs(value).forEach((ref) => {
      const taskResults = resultByTaskName.get(ref.taskName)
      if (!taskResults || !taskResults.has(ref.resultName)) {
        issues.push({
          level: 'error',
          target: 'results',
          message: `Pipeline Result「${name || '-'}」引用的 Task result 不存在：${ref.taskName}.${ref.resultName}`
        })
      }
    })
  })
}

function extractPipelineResultRefs(value: string) {
  const refs: Array<{ taskName: string; resultName: string }> = []
  const pattern = /\$\((tasks|finally)\.([^.()]+)\.results\.([^)]+)\)/g
  let match = pattern.exec(value)
  while (match) {
    refs.push({ taskName: match[2], resultName: match[3] })
    match = pattern.exec(value)
  }
  return refs
}

function validatePipelineWorkspaces(
  config: TektonDagConfig,
  issues: TektonDagValidationIssue[],
  options: TektonDagValidationOptions
) {
  const names = new Set<string>()
  const pipelineParamByName = new Map(
    declaredPipelineParams(config)
      .filter((param) => param.name)
      .map((param) => [param.name, param])
  )
  const workspaceStrategy = config.runPolicy?.workspaceStrategy
  const declaredWorkspaces = declaredPipelineWorkspaces(config)
  declaredWorkspaces.forEach((workspace) => {
    const bindingType = effectiveWorkspaceBindingType(workspace.bindingType, workspaceStrategy)
    if (!validateTektonName(workspace.name)) {
      issues.push({ level: 'error', target: 'workspaces', message: 'Workspace 名称不合法' })
    }
    if (names.has(workspace.name)) {
      issues.push({
        level: 'error',
        target: 'workspaces',
        message: `Workspace 重复：${workspace.name}`
      })
    }
    names.add(workspace.name)
    if (bindingType && !supportedWorkspaceBindings.has(bindingType)) {
      issues.push({
        level: 'error',
        target: 'workspaces',
        message: `Workspace「${workspace.name || '-'}」绑定类型不支持：${bindingType}`
      })
    }
    ;(workspace.allowedBindingTypes || []).forEach((item) => {
      if (!supportedWorkspaceBindings.has(item)) {
        issues.push({
          level: 'error',
          target: 'workspaces',
          message: `Workspace「${workspace.name || '-'}」允许绑定类型不支持：${item}`
        })
      }
    })
    if (
      workspace.allowedBindingTypes?.length &&
      bindingType &&
      !workspace.allowedBindingTypes.includes(bindingType)
    ) {
      issues.push({
        level: 'error',
        target: 'workspaces',
        message: `Workspace「${workspace.name || '-'}」默认绑定类型不在允许范围内`
      })
    }
    ;(workspace.allowedSourceTypes || []).forEach((item) => {
      if (!supportedWorkspaceSources.has(item)) {
        issues.push({
          level: 'error',
          target: 'workspaces',
          message: `Workspace「${workspace.name || '-'}」平台来源类型不支持：${item}`
        })
      }
      if (workspaceAllowsGeneratedSecret(workspace) && item === 'kubernetesResource') {
        issues.push({
          level: 'error',
          target: 'workspaces',
          message: `Workspace「${workspace.name || '-'}」平台生成 Secret 只能来自平台凭证或配置中心`
        })
      }
    })
    if (
      options.requireWorkspaceValues &&
      bindingType === 'persistentVolumeClaim' &&
      isBlank(options.workspaceValues?.[workspace.name])
    ) {
      issues.push({
        level: 'error',
        target: 'workspaces',
        message: `PVC Workspace「${workspace.name || '-'}」必须在运行配置中填写 PVC 名称`
      })
    }
    if (
      options.requireWorkspaceValues &&
      (bindingType === 'secret' || bindingType === 'generatedSecret') &&
      isBlank(options.workspaceValues?.[workspace.name])
    ) {
      issues.push({
        level: 'error',
        target: 'workspaces',
        message:
          bindingType === 'generatedSecret'
            ? `平台生成 Secret Workspace「${workspace.name || '-'}」必须选择平台资源`
            : `Secret Workspace「${workspace.name || '-'}」必须在运行配置中填写 Secret 名称`
      })
    }
    if (
      options.requireWorkspaceValues &&
      bindingType === 'configMap' &&
      isBlank(options.workspaceValues?.[workspace.name])
    ) {
      issues.push({
        level: 'error',
        target: 'workspaces',
        message: `ConfigMap Workspace「${workspace.name || '-'}」必须在运行配置中填写 ConfigMap 名称`
      })
    }
    if (bindingType === 'volumeClaimTemplate' && isBlank(workspace.storage)) {
      issues.push({
        level: 'warning',
        target: 'workspaces',
        message: `动态 PVC Workspace「${workspace.name || '-'}」未填写容量时将默认使用 10Gi`
      })
    }
    if (
      bindingType === 'volumeClaimTemplate' &&
      String(workspace.volumeMode || '').trim() &&
      !['Filesystem', 'Block'].includes(String(workspace.volumeMode || '').trim())
    ) {
      issues.push({
        level: 'error',
        target: 'workspaces',
        message: `动态 PVC Workspace「${workspace.name || '-'}」volumeMode 只支持 Filesystem 或 Block`
      })
    }
    if (
      bindingType === 'projected' &&
      !workspace.projected &&
      !workspace.config?.projected &&
      options.requireWorkspaceValues
    ) {
      issues.push({
        level: 'error',
        target: 'workspaces',
        message: `Projected Workspace「${workspace.name || '-'}」必须配置 projected 内容`
      })
    }
    validateWorkspaceParamRefs(workspace, bindingType, pipelineParamByName, issues, options)
  })
}

function workspaceAllowsGeneratedSecret(workspace: TektonDagConfig['workspaces'][number]) {
  const allowed = Array.isArray(workspace.allowedBindingTypes)
    ? workspace.allowedBindingTypes.filter(Boolean)
    : []
  return allowed.length
    ? allowed.includes('generatedSecret')
    : workspace.bindingType === 'generatedSecret'
}

function validateWorkspaceParamRefs(
  workspace: TektonDagConfig['workspaces'][number],
  bindingType: TektonDagConfig['workspaces'][number]['bindingType'],
  pipelineParamByName: Map<string, TektonDagConfig['params'][number]>,
  issues: TektonDagValidationIssue[],
  options: TektonDagValidationOptions
) {
  const fields = [
    workspace.claimName,
    workspace.secretName,
    workspace.configMapName,
    workspace.storageClassName,
    workspace.storage,
    workspace.accessMode,
    workspace.subPath,
    workspace.resourceId
  ]
  fields.flatMap(extractPipelineParamRefs).forEach((paramName) => {
    if (!pipelineParamByName.has(paramName)) {
      issues.push({
        level: 'error',
        target: 'workspaces',
        message: `Workspace「${workspace.name || '-'}」引用了不存在的 Pipeline 参数：${paramName}`
      })
    }
  })
  const requiredBindingFields =
    bindingType === 'persistentVolumeClaim'
      ? [workspace.claimName]
      : bindingType === 'secret'
        ? [workspace.secretName]
        : bindingType === 'generatedSecret'
          ? [workspace.resourceId]
          : bindingType === 'configMap'
            ? [workspace.configMapName]
            : []
  requiredBindingFields.flatMap(extractPipelineParamRefs).forEach((paramName) => {
    const pipelineParam = pipelineParamByName.get(paramName)
    if (pipelineParam && !isPipelineParamValueReady(pipelineParam, options)) {
      issues.push({
        level: 'error',
        target: 'params',
        message: `Workspace「${workspace.name || '-'}」必填绑定字段引用的 Pipeline 参数「${paramName}」必须配置默认值、当前值或设为运行时输入`
      })
    }
  })
}

function isPipelineParamValueReady(
  param: TektonDagConfig['params'][number],
  options: TektonDagValidationOptions
) {
  if (param.paramType === 'channelCredential') return true
  return (
    param.runtimeConfig === true ||
    hasPipelineParamValue(param, param.defaultValue) ||
    hasPipelineParamValue(param, options.paramValues?.[param.name])
  )
}

function extractPipelineParamRefs(value?: string) {
  const text = String(value || '')
  const refs: string[] = []
  const pattern = /\$\(params\.([A-Za-z_][A-Za-z0-9_.-]*)\)/g
  let match = pattern.exec(text)
  while (match) {
    refs.push(match[1])
    match = pattern.exec(text)
  }
  return refs
}

function declaredPipelineParams(config: TektonDagConfig) {
  const merged = new Map<string, TektonDagConfig['params'][number]>()
  ;[...(config.pipeline.params || []), ...(config.params || [])].forEach((item) => {
    const name = String(item?.name || '').trim()
    if (!name || merged.has(name)) return
    merged.set(name, { ...item, name })
  })
  return Array.from(merged.values())
}

function declaredPipelineResults(config: TektonDagConfig) {
  const merged = new Map<string, NonNullable<TektonDagConfig['results']>[number]>()
  ;[...(config.pipeline.results || []), ...(config.results || [])].forEach((item) => {
    const name = String(item?.name || '').trim()
    if (!name || merged.has(name)) return
    merged.set(name, { ...item, name })
  })
  return Array.from(merged.values())
}

function declaredPipelineWorkspaces(config: TektonDagConfig) {
  const merged = new Map<string, TektonDagConfig['workspaces'][number]>()
  ;[
    ...(config.pipeline.workspaces || []),
    ...(config.workspaces || []),
    ...(config.runPolicy?.workspaces || [])
  ].forEach((item) => {
    const name = String(item?.name || '').trim()
    if (!name || merged.has(name)) return
    merged.set(name, { ...item, name })
  })
  return Array.from(merged.values())
}

function validateNodes(
  config: TektonDagConfig,
  issues: TektonDagValidationIssue[],
  options: TektonDagValidationOptions
) {
  const names = new Map<string, string>()
  const workspaceNames = new Set(
    declaredPipelineWorkspaces(config)
      .map((item) => item.name)
      .filter(Boolean)
  )
  const pipelineParamNames = new Set(
    declaredPipelineParams(config)
      .map((item) => item.name)
      .filter(Boolean)
  )

  config.nodes.forEach((node) => {
    if (node.type === 'note' || node.enabled === false) return
    validateNodeIdentity(node, names, issues)
    validateTaskRef(node, pipelineParamNames, issues)
    validateNodeParams(config, node, issues, options)
    validateWorkspaceMappings(node, workspaceNames, config, issues)
    validateWhenExpressions(node, pipelineParamNames, issues)
    validateMatrix(node, issues)
  })
}

function validateNodeIdentity(
  node: TektonDagNode,
  names: Map<string, string>,
  issues: TektonDagValidationIssue[]
) {
  if (!validateTektonName(node.name)) {
    issues.push({
      level: 'error',
      target: node.id,
      message: `节点「${node.displayName || node.name}」名称不合法`
    })
  }
  const duplicate = names.get(node.name)
  if (duplicate) {
    issues.push({ level: 'error', target: node.id, message: `Task 名称重复：${node.name}` })
  } else {
    names.set(node.name, node.id)
  }
  if (node.retries !== undefined && node.retries < 0) {
    issues.push({
      level: 'error',
      target: node.id,
      message: `节点「${node.displayName || node.name}」重试次数不能小于 0`
    })
  }
  if (node.onError && !['continue', 'stopAndFail'].includes(node.onError)) {
    issues.push({
      level: 'error',
      target: node.id,
      message: `节点「${node.displayName || node.name}」失败策略只支持 continue 或 stopAndFail`
    })
  }
  validateDuration(node.timeout, `节点「${node.displayName || node.name}」Timeout`, node.id, issues)
}

function validateDuration(
  value: string | undefined,
  label: string,
  target: string,
  issues: TektonDagValidationIssue[]
) {
  const duration = String(value || '').trim()
  if (!duration) return
  if (!tektonDurationPattern.test(duration)) {
    issues.push({
      level: 'error',
      target,
      message: `${label} 格式不合法，请使用 30m、1h、1h30m 这类 Tekton duration`
    })
  }
}

function validateTaskRef(
  node: TektonDagNode,
  pipelineParamNames: Set<string>,
  issues: TektonDagValidationIssue[]
) {
  const sources = [
    hasTaskRefValue(node),
    hasPipelineRefValue(node),
    hasRecordObjectValue(node.taskSpec),
    hasRecordObjectValue(node.pipelineSpec)
  ].filter(Boolean).length
  if (sources === 0) {
    issues.push({
      level: 'error',
      target: node.id,
      message: `节点「${node.displayName || node.name}」未选择 Task 或 Pipeline 引用`
    })
    return
  }
  if (sources > 1) {
    issues.push({
      level: 'error',
      target: node.id,
      message: `节点「${node.displayName || node.name}」taskRef、pipelineRef、taskSpec、pipelineSpec 只能配置一个`
    })
    return
  }
  if (hasRecordObjectValue(node.taskSpec)) {
    issues.push({
      level: 'error',
      target: node.id,
      message: `节点「${node.displayName || node.name}」平台默认禁止 Tekton inline taskSpec`
    })
    return
  }
  if (hasRecordObjectValue(node.pipelineSpec)) {
    issues.push({
      level: 'error',
      target: node.id,
      message: `节点「${node.displayName || node.name}」平台默认禁止 Tekton inline pipelineSpec`
    })
    return
  }
  if (node.pipelineRef) {
    validatePipelineRef(node, pipelineParamNames, issues)
    return
  }
  const taskRef = node.taskRef || { kind: 'Task', name: '' }
  if (!isBlank(taskRef.bundle)) {
    issues.push({
      level: 'error',
      target: node.id,
      message: `节点「${node.displayName || node.name}」taskRef.bundle 已废弃，请使用 resolver`
    })
    return
  }
  const isResolverRef =
    !isBlank(taskRef.resolver) || !isBlank(taskRef.namespace) || Boolean(taskRef.params?.length)
  if (isResolverRef) {
    const resolver = String(taskRef.resolver || '').trim() || 'cluster'
    if (!validateTektonName(resolver)) {
      issues.push({
        level: 'error',
        target: node.id,
        message: `节点「${node.displayName || node.name}」Resolver 名称不合法`
      })
    }
    taskRef.params?.forEach((param) => {
      if (!validateParamName(param.name) || isBlank(param.value)) {
        issues.push({
          level: 'error',
          target: node.id,
          message: `节点「${node.displayName || node.name}」Resolver 参数不完整`
        })
      }
      extractPipelineParamRefs(param.value).forEach((paramName) => {
        if (!pipelineParamNames.has(paramName)) {
          issues.push({
            level: 'error',
            target: node.id,
            message: `节点「${node.displayName || node.name}」Resolver 参数引用了不存在的 Pipeline 参数：${paramName}`
          })
        }
      })
    })
    if (taskRef.params?.length) {
      const paramByName = new Map(
        taskRef.params.map((param) => [String(param.name || '').trim(), param])
      )
      if (resolver === 'cluster') {
        ;['kind', 'name'].forEach((name) => {
          const param = paramByName.get(name)
          if (!param || isBlank(param.value)) {
            issues.push({
              level: 'error',
              target: node.id,
              message: `节点「${node.displayName || node.name}」Resolver 必须配置 ${name} 参数`
            })
          }
        })
        const kind = String(paramByName.get('kind')?.value || '').trim()
        const name = String(paramByName.get('name')?.value || '').trim()
        const namespace = String(paramByName.get('namespace')?.value || '').trim()
        if (kind && !['task', 'clustertask'].includes(kind.toLowerCase())) {
          issues.push({
            level: 'error',
            target: node.id,
            message: `节点「${node.displayName || node.name}」cluster resolver kind 只支持 task 或 clustertask`
          })
        }
        if (name && !validateTektonName(name)) {
          issues.push({
            level: 'error',
            target: node.id,
            message: `节点「${node.displayName || node.name}」cluster resolver name 不合法`
          })
        }
        if (namespace && !validateTektonName(namespace)) {
          issues.push({
            level: 'error',
            target: node.id,
            message: `节点「${node.displayName || node.name}」cluster resolver namespace 不合法`
          })
        }
      }
    } else if (isBlank(taskRef.name) || isBlank(taskRef.kind)) {
      issues.push({
        level: 'error',
        target: node.id,
        message: `节点「${node.displayName || node.name}」Resolver 必须配置 Task 名称和类型`
      })
    }
    return
  }
  if (isBlank(taskRef.name)) {
    issues.push({
      level: 'error',
      target: node.id,
      message: `节点「${node.displayName || node.name}」未选择 Task`
    })
  }
  const kind = String(taskRef.kind || 'Task').trim()
  const apiVersion = String(taskRef.apiVersion || '').trim()
  if (!taskKindPattern.test(kind)) {
    issues.push({
      level: 'error',
      target: node.id,
      message: `节点「${node.displayName || node.name}」taskRef.kind 不合法`
    })
  }
  if (apiVersion && !apiVersionPattern.test(apiVersion)) {
    issues.push({
      level: 'error',
      target: node.id,
      message: `节点「${node.displayName || node.name}」taskRef.apiVersion 不合法`
    })
  }
  if (kind !== 'Task' && kind !== 'ClusterTask' && !apiVersion) {
    issues.push({
      level: 'error',
      target: node.id,
      message: `节点「${node.displayName || node.name}」Custom Task 必须配置 apiVersion`
    })
  }
}

function validatePipelineRef(
  node: TektonDagNode,
  pipelineParamNames: Set<string>,
  issues: TektonDagValidationIssue[]
) {
  const ref = node.pipelineRef
  if (!ref) return
  if (!isBlank(ref.bundle)) {
    issues.push({
      level: 'error',
      target: node.id,
      message: `节点「${node.displayName || node.name}」pipelineRef.bundle 已废弃，请使用 resolver`
    })
    return
  }
  if (!isBlank(ref.apiVersion) && !apiVersionPattern.test(String(ref.apiVersion || '').trim())) {
    issues.push({
      level: 'error',
      target: node.id,
      message: `节点「${node.displayName || node.name}」pipelineRef.apiVersion 不合法`
    })
  }
  const isResolverRef = !isBlank(ref.resolver) || Boolean(ref.params?.length)
  if (!isResolverRef) {
    if (isBlank(ref.name)) {
      issues.push({
        level: 'error',
        target: node.id,
        message: `节点「${node.displayName || node.name}」pipelineRef.name 不能为空`
      })
    } else if (!validateTektonName(ref.name || '')) {
      issues.push({
        level: 'error',
        target: node.id,
        message: `节点「${node.displayName || node.name}」pipelineRef.name 不合法`
      })
    }
    return
  }
  const resolver = String(ref.resolver || '').trim() || 'cluster'
  if (!validateTektonName(resolver)) {
    issues.push({
      level: 'error',
      target: node.id,
      message: `节点「${node.displayName || node.name}」pipelineRef resolver 名称不合法`
    })
  }
  if (!ref.params?.length) {
    issues.push({
      level: 'error',
      target: node.id,
      message: `节点「${node.displayName || node.name}」pipelineRef resolver 必须配置 params`
    })
    return
  }
  ref.params.forEach((param) => {
    if (!validateParamName(param.name) || isBlank(param.value)) {
      issues.push({
        level: 'error',
        target: node.id,
        message: `节点「${node.displayName || node.name}」pipelineRef resolver 参数不完整`
      })
    }
    extractPipelineParamRefs(param.value).forEach((paramName) => {
      if (!pipelineParamNames.has(paramName)) {
        issues.push({
          level: 'error',
          target: node.id,
          message: `节点「${node.displayName || node.name}」pipelineRef 参数引用了不存在的 Pipeline 参数：${paramName}`
        })
      }
    })
  })
}

function hasTaskRefValue(node: TektonDagNode) {
  const ref = node.taskRef
  if (!ref) return false
  const kind = String(ref.kind || '').trim()
  return (
    !isBlank(ref.name) ||
    !isBlank(ref.apiVersion) ||
    !isBlank(ref.bundle) ||
    !isBlank(ref.resolver) ||
    !isBlank(ref.namespace) ||
    Boolean(ref.params?.length) ||
    (kind !== '' && kind !== 'Task')
  )
}

function hasPipelineRefValue(node: TektonDagNode) {
  const ref = node.pipelineRef
  return Boolean(
    ref &&
      (!isBlank(ref.name) ||
        !isBlank(ref.apiVersion) ||
        !isBlank(ref.bundle) ||
        !isBlank(ref.resolver) ||
        ref.params?.length)
  )
}

function hasRecordObjectValue(value?: Record<string, any>) {
  return Boolean(value && typeof value === 'object' && Object.keys(value).length > 0)
}

function validateNodeParams(
  config: TektonDagConfig,
  node: TektonDagNode,
  issues: TektonDagValidationIssue[],
  options: TektonDagValidationOptions
) {
  const nodeParamByName = new Map(node.params.map((param) => [param.name, param]))
  const matrixParamNames = new Set((node.matrix?.params || []).map((param) => param.name))
  const pipelineParamByName = new Map(
    declaredPipelineParams(config).map((param) => [param.name, param])
  )
  const requirementByName = new Map(
    (node.paramRequirements || []).map((param) => [param.name, param])
  )
  ;(node.paramRequirements || []).forEach((requiredParam) => {
    const param = nodeParamByName.get(requiredParam.name)
    if (!param) {
      if (matrixParamNames.has(requiredParam.name)) return
      if (
        requiredParam.required !== true &&
        requiredParam.runtimeConfig !== true &&
        requiredParam.readonly !== true
      ) {
        return
      }
      issues.push({
        level: 'error',
        target: node.id,
        message: `节点「${node.displayName || node.name}」缺少必要 Task 参数映射「${requiredParam.displayName || requiredParam.name}」`
      })
      return
    }
    if (
      requiredParam.required &&
      param.source === 'fixed' &&
      isBlank(param.value) &&
      isBlank(param.defaultValue) &&
      isBlank(requiredParam.defaultValue)
    ) {
      issues.push({
        level: 'error',
        target: node.id,
        message: `节点「${node.displayName || node.name}」必填参数「${requiredParam.displayName || requiredParam.name}」未配置`
      })
    }
    if (
      requiredParam.required &&
      (param.source === 'pipelineParam' || param.source === 'runtime') &&
      param.pipelineParam
    ) {
      const pipelineParam = pipelineParamByName.get(param.pipelineParam)
      if (pipelineParam && !isPipelineParamValueReady(pipelineParam, options)) {
        issues.push({
          level: 'error',
          target: 'params',
          message: `Pipeline 参数「${param.pipelineParam}」被必填 Task 参数「${requiredParam.displayName || requiredParam.name}」引用，必须配置默认值、当前值或设为运行时输入`
        })
      }
    }
  })
  node.params.forEach((param) => {
    if (!validateParamName(param.name)) {
      issues.push({
        level: 'error',
        target: node.id,
        message: `节点「${node.displayName || node.name}」存在非法参数名`
      })
    }
    if (param.source === 'result' && (isBlank(param.resultNodeId) || isBlank(param.resultName))) {
      issues.push({
        level: 'error',
        target: node.id,
        message: `节点「${node.displayName || node.name}」Result 引用不完整`
      })
    }
    if (
      param.source === 'result' &&
      !isBlank(param.resultName) &&
      !validateParamName(param.resultName)
    ) {
      issues.push({
        level: 'warning',
        target: node.id,
        message: `节点「${node.displayName || node.name}」引用的 Result 名称可能不符合 Tekton 命名规范`
      })
    }
    if (
      (param.source === 'pipelineParam' || param.source === 'runtime') &&
      isBlank(param.pipelineParam)
    ) {
      issues.push({
        level: 'error',
        target: node.id,
        message: `节点「${node.displayName || node.name}」Pipeline 参数引用不完整`
      })
    }
    if (
      (param.source === 'pipelineParam' || param.source === 'runtime') &&
      param.pipelineParam &&
      !declaredPipelineParams(config).some((item) => item.name === param.pipelineParam)
    ) {
      issues.push({
        level: 'error',
        target: node.id,
        message: `节点「${node.displayName || node.name}」引用了不存在的 Pipeline 参数`
      })
    }
    if (param.source === 'fixed') {
      const requirement = requirementByName.get(param.name)
      const value = param.value || param.defaultValue || requirement?.defaultValue || ''
      if (String(value || '').includes('$(')) return
      const error = validateTypedTextValue(
        value,
        requirement?.type || requirement?.paramType || param.paramType,
        `节点「${node.displayName || node.name}」参数「${param.displayName || param.name}」固定值`
      )
      if (error) {
        issues.push({ level: 'error', target: node.id, message: error })
      }
    }
    if (
      param.source === 'result' &&
      param.resultNodeId &&
      !config.nodes.some((item) => item.id === param.resultNodeId && item.enabled !== false)
    ) {
      issues.push({
        level: 'error',
        target: node.id,
        message: `节点「${node.displayName || node.name}」引用了不存在的上游 Result`
      })
    }
    if (param.source === 'result' && param.resultNodeId && !isBlank(param.resultName)) {
      const sourceNode = config.nodes.find(
        (item) => item.id === param.resultNodeId && item.enabled !== false
      )
      if (sourceNode?.results?.length && !sourceNode.results.includes(String(param.resultName))) {
        issues.push({
          level: 'warning',
          target: node.id,
          message: `节点「${node.displayName || node.name}」引用的 Result「${param.resultName}」未在上游节点「${sourceNode.displayName || sourceNode.name}」声明`
        })
      }
    }
  })
}

function validateWorkspaceMappings(
  node: TektonDagNode,
  workspaceNames: Set<string>,
  config: TektonDagConfig,
  issues: TektonDagValidationIssue[]
) {
  const seenTaskWorkspaces = new Set<string>()
  const mappingByTaskWorkspace = new Map(
    node.workspaces.map((workspace) => [workspace.taskWorkspace, workspace])
  )
  ;(node.workspaceRequirements || []).forEach((requiredWorkspace) => {
    const mapping = mappingByTaskWorkspace.get(requiredWorkspace.name)
    if (!mapping && !requiredWorkspace.optional) {
      issues.push({
        level: 'error',
        target: node.id,
        message: `节点「${node.displayName || node.name}」缺少 Workspace「${requiredWorkspace.name}」映射`
      })
      return
    }
    if (!mapping || !requiredWorkspace.requiredBindingType) return
    const pipelineWorkspace = declaredPipelineWorkspaces(config).find(
      (item) => item.name === mapping.pipelineWorkspace
    )
    const bindingType = effectiveWorkspaceBindingType(
      pipelineWorkspace?.bindingType,
      config.runPolicy?.workspaceStrategy
    )
    if (bindingType !== requiredWorkspace.requiredBindingType) {
      issues.push({
        level: 'error',
        target: node.id,
        message: `Workspace「${requiredWorkspace.name}」必须绑定为 ${requiredWorkspace.requiredBindingType}`
      })
    }
  })
  node.workspaces.forEach((workspace) => {
    if (isBlank(workspace.taskWorkspace) || isBlank(workspace.pipelineWorkspace)) {
      issues.push({
        level: 'error',
        target: node.id,
        message: `节点「${node.displayName || node.name}」Workspace 映射不完整`
      })
    }
    const taskWorkspace = String(workspace.taskWorkspace || '').trim()
    if (taskWorkspace) {
      if (seenTaskWorkspaces.has(taskWorkspace)) {
        issues.push({
          level: 'error',
          target: node.id,
          message: `节点「${node.displayName || node.name}」Workspace 重复映射：${taskWorkspace}`
        })
      }
      seenTaskWorkspaces.add(taskWorkspace)
    }
    if (workspace.pipelineWorkspace && !workspaceNames.has(workspace.pipelineWorkspace)) {
      issues.push({
        level: 'error',
        target: node.id,
        message: `Workspace「${workspace.pipelineWorkspace}」未在 Pipeline 中声明`
      })
    }
  })
}

function effectiveWorkspaceBindingType(
  bindingType: TektonDagConfig['workspaces'][number]['bindingType'],
  workspaceStrategy?: TektonDagConfig['runPolicy']['workspaceStrategy']
) {
  const current = bindingType || 'emptyDir'
  return current === 'emptyDir' && workspaceStrategy ? workspaceStrategy : current
}

function validateWhenExpressions(
  node: TektonDagNode,
  pipelineParamNames: Set<string>,
  issues: TektonDagValidationIssue[]
) {
  node.when.forEach((item) => {
    if (!isBlank(item.cel)) {
      extractPipelineParamRefs(item.cel).forEach((paramName) => {
        if (!pipelineParamNames.has(paramName)) {
          issues.push({
            level: 'error',
            target: node.id,
            message: `节点「${node.displayName || node.name}」When CEL 引用了不存在的 Pipeline 参数：${paramName}`
          })
        }
      })
      return
    }
    if (
      isBlank(item.input) ||
      !item.values?.length ||
      item.values.some((value) => !isNonEmptyStringValue(value))
    ) {
      issues.push({
        level: 'error',
        target: node.id,
        message: `节点「${node.displayName || node.name}」When 条件必须配置字符串 values`
      })
    }
    if (item.operator && !['in', 'notin'].includes(item.operator)) {
      issues.push({
        level: 'error',
        target: node.id,
        message: `节点「${node.displayName || node.name}」When operator 只支持 in/notin`
      })
    }
    extractPipelineParamRefs(item.input).forEach((paramName) => {
      if (!pipelineParamNames.has(paramName)) {
        issues.push({
          level: 'error',
          target: node.id,
          message: `节点「${node.displayName || node.name}」When 引用了不存在的 Pipeline 参数：${paramName}`
        })
      }
    })
  })
}

function validateYamlObjectRaw(
  raw: string | undefined,
  label: string,
  issues: TektonDagValidationIssue[]
) {
  const text = String(raw || '').trim()
  if (!text) return
  try {
    const parsed = parse(text)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      issues.push({
        level: 'error',
        target: 'runPolicy',
        message: `${label} 必须是合法 YAML 对象`
      })
    }
  } catch {
    issues.push({
      level: 'error',
      target: 'runPolicy',
      message: `${label} 必须是合法 YAML 对象`
    })
  }
}

function validateTaskRunSpecsRaw(
  config: TektonDagConfig,
  raw: string | undefined,
  issues: TektonDagValidationIssue[]
) {
  const text = String(raw || '').trim()
  if (!text) return
  try {
    const parsed = parse(text)
    const validArray =
      Array.isArray(parsed) &&
      parsed.every((item) => item && typeof item === 'object' && !Array.isArray(item))
    const validObject =
      parsed &&
      typeof parsed === 'object' &&
      !Array.isArray(parsed) &&
      Object.keys(parsed).length > 0
    if (!validArray && !validObject) {
      issues.push({
        level: 'error',
        target: 'runPolicy',
        message: 'taskRunSpecs 必须是合法 YAML 数组或对象'
      })
      return
    }
    const specs = Array.isArray(parsed) ? parsed : [parsed]
    validateTaskRunSpecsItems(config, specs, issues)
  } catch {
    issues.push({
      level: 'error',
      target: 'runPolicy',
      message: 'taskRunSpecs 必须是合法 YAML 数组或对象'
    })
  }
}

function validateTaskRunSpecsItems(
  config: TektonDagConfig,
  specs: Array<Record<string, any>> | undefined,
  issues: TektonDagValidationIssue[]
) {
  if (!Array.isArray(specs) || specs.length === 0) return
  const taskNames = new Set(
    config.nodes
      .filter((node) => node.enabled !== false && node.type !== 'note')
      .map((node) => String(node.name || '').trim())
      .filter(Boolean)
  )
  specs.forEach((item: any) => {
    const pipelineTaskName = String(item?.pipelineTaskName || '').trim()
    if (!pipelineTaskName) {
      issues.push({
        level: 'error',
        target: 'runPolicy',
        message: 'taskRunSpecs 每一项必须填写 pipelineTaskName'
      })
      return
    }
    if (!taskNames.has(pipelineTaskName)) {
      issues.push({
        level: 'error',
        target: 'runPolicy',
        message: `taskRunSpecs 引用的 PipelineTask 不存在：${pipelineTaskName}`
      })
    }
  })
}

function validateMatrix(node: TektonDagNode, issues: TektonDagValidationIssue[]) {
  if (!node.matrix) return
  const allowedTaskParams = new Set((node.paramRequirements || []).map((param) => param.name))
  if (!node.matrix.params.length) {
    issues.push({
      level: 'error',
      target: node.id,
      message: `节点「${node.name}」Matrix 未配置参数`
    })
  }
  node.matrix.params.forEach((param) => {
    if (
      !validateParamName(param.name) ||
      !param.values.length ||
      param.values.some((value) => !isNonEmptyStringValue(value))
    ) {
      issues.push({
        level: 'error',
        target: node.id,
        message: `节点「${node.displayName || node.name}」Matrix 参数必须配置字符串 values`
      })
    }
    if (allowedTaskParams.size > 0 && param.name && !allowedTaskParams.has(param.name)) {
      issues.push({
        level: 'error',
        target: node.id,
        message: `节点「${node.displayName || node.name}」Matrix 参数「${param.name}」不属于 Task 参数`
      })
    }
  })
  warnDuplicateNames(
    node.matrix.params.map((param) => param.name),
    node.id,
    `节点「${node.displayName || node.name}」Matrix 参数重复`,
    issues
  )
  const taskParamNames = new Set(
    node.params.map((param) => param.name).filter((name) => validateParamName(name))
  )
  node.matrix.params.forEach((param) => {
    if (taskParamNames.has(param.name)) {
      issues.push({
        level: 'warning',
        target: node.id,
        message: `节点「${node.displayName || node.name}」Matrix 参数「${param.name}」同时存在于普通参数中，请确认最终取值来源`
      })
    }
  })
  node.matrix.include?.forEach((item) => {
    item.params.forEach((param) => {
      if (!validateParamName(param.name) || !isNonEmptyStringValue(param.value)) {
        issues.push({
          level: 'error',
          target: node.id,
          message: `节点「${node.displayName || node.name}」Matrix include 参数必须配置字符串 value`
        })
      }
      if (allowedTaskParams.size > 0 && param.name && !allowedTaskParams.has(param.name)) {
        issues.push({
          level: 'error',
          target: node.id,
          message: `节点「${node.displayName || node.name}」Matrix include 参数「${param.name}」不属于 Task 参数`
        })
      }
      if (taskParamNames.has(param.name)) {
        issues.push({
          level: 'warning',
          target: node.id,
          message: `节点「${node.displayName || node.name}」Matrix include 参数「${param.name}」同时存在于普通参数中，请确认最终取值来源`
        })
      }
    })
    warnDuplicateNames(
      item.params.map((param) => param.name),
      node.id,
      `节点「${node.displayName || node.name}」Matrix include 参数重复`,
      issues
    )
  })
}

function validateEdges(config: TektonDagConfig, issues: TektonDagValidationIssue[]) {
  const nodeIDs = new Set(config.nodes.map((node) => node.id))
  const pipelineParamNames = new Set(
    declaredPipelineParams(config)
      .map((item) => item.name)
      .filter(Boolean)
  )
  config.edges.forEach((edge) => {
    if (!nodeIDs.has(edge.source) || !nodeIDs.has(edge.target)) {
      issues.push({ level: 'error', target: edge.id, message: '存在指向不存在节点的依赖线' })
    }
    const source = config.nodes.find((item) => item.id === edge.source)
    const target = config.nodes.find((item) => item.id === edge.target)
    if (edge.source === edge.target) {
      issues.push({ level: 'error', target: edge.id, message: '依赖线不能指向自身' })
    }
    if (source?.type === 'finally' && target?.type !== 'finally') {
      issues.push({ level: 'error', target: edge.id, message: '普通 Task 不能依赖 Finally Task' })
    }
    if (target?.type === 'finally' && edge.type === 'runAfter') {
      issues.push({
        level: 'warning',
        target: edge.id,
        message: 'Finally 任务在 Tekton 中统一于主流程后执行，不生成 runAfter'
      })
    }
    if (source?.type === 'finally' && target?.type === 'finally') {
      issues.push({
        level: 'warning',
        target: edge.id,
        message: 'Tekton finally 任务并行执行，连线仅作为备注展示'
      })
    }
    if (edge.type === 'result' && (isBlank(edge.resultName) || isBlank(edge.targetParam))) {
      issues.push({
        level: 'error',
        target: edge.id,
        message: 'Result 连线必须选择输出 Result 和目标参数'
      })
    }
    if (
      edge.type === 'result' &&
      !isBlank(edge.resultName) &&
      !validateParamName(edge.resultName)
    ) {
      issues.push({
        level: 'warning',
        target: edge.id,
        message: 'Result 连线的 Result 名称可能不符合 Tekton 命名规范'
      })
    }
    if (
      edge.type === 'result' &&
      source?.results?.length &&
      edge.resultName &&
      !source.results.includes(edge.resultName)
    ) {
      issues.push({
        level: 'warning',
        target: edge.id,
        message: `Result 连线引用的 Result「${edge.resultName}」未在上游节点「${source.displayName || source.name}」声明`
      })
    }
    if (
      edge.type === 'when' &&
      (isBlank(edge.whenInput) ||
        !edge.whenValues?.length ||
        edge.whenValues.some((value) => !isNonEmptyStringValue(value)))
    ) {
      issues.push({
        level: 'error',
        target: edge.id,
        message: 'When 连线必须配置 input 和字符串匹配值'
      })
    }
    if (edge.type === 'when' && edge.whenOperator && !['in', 'notin'].includes(edge.whenOperator)) {
      issues.push({
        level: 'error',
        target: edge.id,
        message: 'When 连线 operator 只支持 in/notin'
      })
    }
    if (edge.type === 'when') {
      extractPipelineParamRefs(edge.whenInput).forEach((paramName) => {
        if (!pipelineParamNames.has(paramName)) {
          issues.push({
            level: 'error',
            target: edge.id,
            message: `When 连线引用了不存在的 Pipeline 参数：${paramName}`
          })
        }
      })
    }
  })
}

function warnDuplicateNames(
  names: string[],
  target: string,
  prefix: string,
  issues: TektonDagValidationIssue[]
) {
  const seen = new Set<string>()
  names
    .map((name) => String(name || '').trim())
    .filter((name) => validateParamName(name))
    .forEach((name) => {
      if (seen.has(name)) {
        issues.push({ level: 'warning', target, message: `${prefix}：${name}` })
      } else {
        seen.add(name)
      }
    })
}

function detectCycles(config: TektonDagConfig): string[] {
  const graph = new Map<string, string[]>()
  config.nodes
    .filter((node) => node.enabled !== false && node.type !== 'note' && node.type !== 'finally')
    .forEach((node) => graph.set(node.id, []))
  config.edges
    .filter(
      (edge) =>
        (edge.type === 'runAfter' || edge.type === 'result' || edge.type === 'when') &&
        graph.has(edge.source) &&
        graph.has(edge.target)
    )
    .forEach((edge) => {
      graph.get(edge.source)?.push(edge.target)
    })

  const visiting = new Set<string>()
  const visited = new Set<string>()
  const cycles = new Set<string>()

  const walk = (nodeId: string) => {
    if (visiting.has(nodeId)) {
      cycles.add(nodeId)
      return
    }
    if (visited.has(nodeId)) return
    visiting.add(nodeId)
    ;(graph.get(nodeId) || []).forEach(walk)
    visiting.delete(nodeId)
    visited.add(nodeId)
  }

  graph.forEach((_, nodeId) => walk(nodeId))
  return Array.from(cycles)
}
