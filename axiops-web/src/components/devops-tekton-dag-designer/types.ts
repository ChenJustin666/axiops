import type {
  DevopsStepParam,
  DevopsStepTemplate,
  DevopsTektonTaskParam,
  DevopsTektonTaskResult,
  DevopsTektonWorkspace,
  StepParamConfig,
  StepParamOption
} from '@/api/devops/pipeline'

export type TektonDagNodeType =
  | 'task'
  | 'clusterTask'
  | 'resolverTask'
  | 'matrix'
  | 'finally'
  | 'note'
export type TektonDagEdgeType = 'runAfter' | 'result' | 'when' | 'finally'
export type TektonParamSource = 'fixed' | 'pipelineParam' | 'runtime' | 'result' | 'dynamic'
export type TektonWorkspaceBindingType =
  | 'emptyDir'
  | 'persistentVolumeClaim'
  | 'volumeClaimTemplate'
  | 'secret'
  | 'generatedSecret'
  | 'configMap'
  | 'projected'
export type TektonWorkspaceSourceType = 'kubernetesResource' | 'platformCredential' | 'configCenter'
export type TektonTaskRefKind = 'Task' | 'ClusterTask' | (string & {})

export interface TektonTaskRef {
  kind: TektonTaskRefKind
  name: string
  apiVersion?: string
  bundle?: string
  resolver?: string
  namespace?: string
  params?: TektonNameValue[]
}

export interface TektonPipelineRef {
  name?: string
  apiVersion?: string
  bundle?: string
  resolver?: string
  params?: TektonNameValue[]
}

export interface TektonNameValue {
  name: string
  value: string
}

export interface TektonDagPipelineParam {
  name: string
  displayName?: string
  type: 'string' | 'array' | 'object'
  enum?: string[]
  properties?: Record<string, { type?: 'string' | 'array' | 'object' }>
  paramType?: string
  defaultValue?: string
  description?: string
  runtimeConfig?: boolean
  required?: boolean
  readonly?: boolean
  selectList?: StepParamOption[]
  allowCustom?: boolean
  options?: StepParamOption[]
  config?: StepParamConfig
  sourceCode?: string
}

export interface TektonDagPipelineResult {
  name: string
  type?: 'string' | 'array' | 'object'
  properties?: Record<string, { type?: 'string' | 'array' | 'object' }>
  description?: string
  value?: string
}

export interface TektonDagWorkspace {
  name: string
  description?: string
  optional?: boolean
  bindingType?: TektonWorkspaceBindingType
  allowedBindingTypes?: TektonWorkspaceBindingType[]
  allowedSourceTypes?: TektonWorkspaceSourceType[]
  allowedPlatformTypes?: string[]
  sourceType?: TektonWorkspaceSourceType
  platformType?: string
  resourceId?: string
  resourceName?: string
  runtimeConfig?: boolean
  config?: Record<string, any>
  claimName?: string
  secretName?: string
  configMapName?: string
  projected?: Record<string, any>
  storageClassName?: string
  storage?: string
  accessMode?: string
  volumeMode?: string
  subPath?: string
}

export interface TektonDagTaskParam {
  name: string
  source: TektonParamSource
  value?: string
  pipelineParam?: string
  resultNodeId?: string
  resultName?: string
  runtimeConfig?: boolean
  sourceCode?: string
  displayName?: string
  defaultValue?: string
  description?: string
  required?: boolean
  readonly?: boolean
  paramType?: string
  options?: StepParamOption[]
  selectList?: StepParamOption[]
  allowCustom?: boolean
  config?: StepParamConfig
}

export interface TektonDagWorkspaceMapping {
  taskWorkspace: string
  pipelineWorkspace: string
  subPath?: string
  optional?: boolean
  description?: string
  requiredBindingType?: TektonWorkspaceBindingType
  bindingParamCode?: string
}

export interface TektonDagWhenExpression {
  input?: string
  operator?: 'in' | 'notin'
  values?: string[]
  cel?: string
}

export interface TektonDagMatrixParam {
  name: string
  values: string[]
}

export interface TektonDagMatrixInclude {
  name?: string
  params: TektonNameValue[]
}

export interface TektonDagMatrix {
  params: TektonDagMatrixParam[]
  include?: TektonDagMatrixInclude[]
}

export interface TektonDagParamRequirement {
  name: string
  type?: 'string' | 'array' | 'object'
  displayName?: string
  exposed?: boolean
  required?: boolean
  defaultValue?: string
  runtimeConfig?: boolean
  sourceCode?: string
  description?: string
  readonly?: boolean
  paramType?: string
  options?: StepParamOption[]
  selectList?: StepParamOption[]
  allowCustom?: boolean
  config?: StepParamConfig
}

export interface TektonDagWorkspaceRequirement {
  name: string
  optional?: boolean
  description?: string
  requiredBindingType?: TektonWorkspaceBindingType
  bindingParamCode?: string
}

export interface TektonDagNode {
  id: string
  type: TektonDagNodeType
  name: string
  displayName?: string
  description?: string
  taskId?: string
  taskCode?: string
  categoryId?: string
  categoryName?: string
  categoryCode?: string
  icon?: string
  iconColor?: string
  taskRef: TektonTaskRef
  taskSpec?: Record<string, any>
  pipelineRef?: TektonPipelineRef
  pipelineSpec?: Record<string, any>
  params: TektonDagTaskParam[]
  paramRequirements?: TektonDagParamRequirement[]
  workspaces: TektonDagWorkspaceMapping[]
  workspaceRequirements?: TektonDagWorkspaceRequirement[]
  when: TektonDagWhenExpression[]
  retries?: number
  timeout?: string
  onError?: 'stopAndFail' | 'continue'
  matrix?: TektonDagMatrix
  results?: string[]
  enabled?: boolean
  position: {
    x: number
    y: number
  }
  sourceTask?: DevopsStepTemplate
}

export interface TektonDagEdge {
  id: string
  source: string
  target: string
  type: TektonDagEdgeType
  resultName?: string
  targetParam?: string
  whenInput?: string
  whenOperator?: 'in' | 'notin'
  whenValues?: string[]
}

export interface TektonRunPolicy {
  serviceAccountName?: string
  managedBy?: string
  timeout?: string
  timeouts?: Partial<Record<'pipeline' | 'tasks' | 'finally', string>>
  workspaces?: TektonDagWorkspace[]
  workspaceStrategy?: 'emptyDir' | 'persistentVolumeClaim' | 'volumeClaimTemplate'
  cleanBeforeRun?: boolean
  cleanAfterRun?: boolean
  cleanupImage?: string
  concurrencyPolicy?: 'allow' | 'skip' | 'replace'
  pipelineTimeout?: string
  tasksTimeout?: string
  finallyTimeout?: string
  labels?: TektonNameValue[]
  annotations?: TektonNameValue[]
  podTemplateEnv?: TektonNameValue[]
  podTemplateRaw?: string
  taskRunTemplateRaw?: string
  taskRunSpecs?: Record<string, any>[]
  taskRunSpecsRaw?: string
}

export interface TektonTriggerConfig {
  enabled?: boolean
  apiVersion?: string
  namespace?: string
  eventListenerName?: string
  triggerName?: string
  triggerBindingName?: string
  triggerTemplateName?: string
  serviceAccountName?: string
  serviceType?: 'ClusterIP' | 'NodePort' | 'LoadBalancer'
  ingressHost?: string
  ingressPath?: string
  ingressClassName?: string
  tlsSecretName?: string
  webhookPath?: string
  webhookUrl?: string
  provider?: string
  interceptorType?: 'none' | 'github' | 'gitlab' | 'cel'
  celFilter?: string
  eventTypes?: string[]
  secretName?: string
  secretKey?: string
  bindingParams?: TektonTriggerParamBinding[]
  templateParams?: TektonTriggerParamBinding[]
  labels?: TektonNameValue[]
  annotations?: TektonNameValue[]
}

export interface TektonTriggerParamBinding {
  name: string
  value: string
  description?: string
}

export interface TektonPrunerPolicy {
  enabled?: boolean
  clusterResourceCleanup?: boolean
  platformRecordCleanup?: boolean
  nativePrunerMode?: 'disabled' | 'namespaceConfigMap' | 'globalConfigMap'
  nativePrunerTargetNamespace?: string
  nativePrunerConfigLevel?: 'global' | 'namespace'
  resourceTypes?: Array<'PipelineRun' | 'TaskRun'>
  scope?: 'pipeline' | 'namespace'
  historyLimit?: number
  successfulHistoryLimit?: number
  failedHistoryLimit?: number
  ttlSecondsAfterFinished?: number
  platformRecordRetentionDays?: number
  platformRecordLimit?: number
  selectorLabels?: TektonNameValue[]
  selectorAnnotations?: TektonNameValue[]
}

export interface TektonScheduleConfig {
  enabled?: boolean
  cron?: string
  timezone?: string
  concurrencyPolicy?: 'allow' | 'skip' | 'replace'
  parameterSnapshot?: boolean
  paramsSnapshot?: Record<string, string>
}

export interface TektonDagConfig {
  version: 'tekton-dag/v1'
  labels?: TektonNameValue[]
  annotations?: TektonNameValue[]
  pipeline: {
    name: string
    namespace: string
    displayName?: string
    description?: string
    serviceAccountName?: string
    timeout?: string
    timeouts?: Partial<Record<'pipeline' | 'tasks' | 'finally', string>>
    params?: TektonDagPipelineParam[]
    results?: TektonDagPipelineResult[]
    workspaces?: TektonDagWorkspace[]
    labels?: TektonNameValue[]
    annotations?: TektonNameValue[]
  }
  params: TektonDagPipelineParam[]
  results?: TektonDagPipelineResult[]
  workspaces: TektonDagWorkspace[]
  nodes: TektonDagNode[]
  finallyNodes?: TektonDagNode[]
  edges: TektonDagEdge[]
  runPolicy?: TektonRunPolicy
  triggerConfig?: TektonTriggerConfig
  prunerPolicy?: TektonPrunerPolicy
  scheduleConfig?: TektonScheduleConfig
}

export interface TektonResourceOption {
  label: string
  value: string
  description?: string
}

export interface TektonTaskCategoryOption {
  label: string
  value: string
  code?: string
}

export interface TektonDagValidationIssue {
  level: 'error' | 'warning'
  target?: string
  message: string
}

export interface TektonDagLibraryItem {
  id: string
  name: string
  code: string
  categoryId?: string
  categoryName?: string
  categoryCode?: string
  stageContent: string
  params?: DevopsStepParam[]
  taskParams?: DevopsTektonTaskParam[]
  taskResults?: DevopsTektonTaskResult[]
  taskWorkspaces?: DevopsTektonWorkspace[]
  icon?: string
  iconColor?: string
  source?: 'platform' | 'cluster' | 'clusterTask' | 'resolver'
  raw?: DevopsStepTemplate
}
