export interface WorkflowParamOption {
  label: string
  value: string
  metadata?: Record<string, any>
}

export interface WorkflowCompoundParamField {
  name: string
  code: string
  defaultValue?: string
  paramType: string
  mode: string
  required?: boolean
  readonly?: boolean
  description?: string
  selectList?: WorkflowParamOption[]
  allowCustom?: boolean
  runtimeConfig?: boolean
  runtimeMode?: string
  config?: WorkflowParamConfig
}

export interface WorkflowParamConfig {
  fullRow?: boolean
  [key: string]: any
}

export interface WorkflowParamDefinition {
  name: string
  code: string
  defaultValue?: string
  paramType: string
  mode: string
  required?: boolean
  readonly?: boolean
  description?: string
  selectList?: WorkflowParamOption[]
  allowCustom?: boolean
  channelGroupCode?: string
  mappingField?: string
  voucherModel?: string
  credentialMode?: 'field' | 'jsonData' | 'jsonBase64' | 'jenkinsCredentialId'
  credentialSourceParamCode?: string
  runtimeMode?: string
  runtimeConfig?: boolean
  sortOrder?: number
  config?: WorkflowParamConfig
  currentValue?: string
  compoundFields?: WorkflowCompoundParamField[]
}

export interface WorkflowLibraryItem {
  id: string
  name: string
  code: string
  categoryId?: string
  categoryCode?: string
  icon?: string
  iconColor?: string
  type?: string
  stageContent?: string
  paramsCount?: number
  params?: WorkflowParamDefinition[]
  taskParams?: unknown[]
  taskResults?: unknown[]
  taskWorkspaces?: unknown[]
}

export interface WorkflowCategoryOption {
  label: string
  value: string
  code?: string
}

export interface WorkflowNode {
  id?: string
  stepId: string
  stepCode?: string
  stepName?: string
  nodeName?: string
  containerName?: string
  stepType?: string
  icon?: string
  iconColor?: string
  parentNodeId?: string
  branchType?: 'next' | 'parallel'
  sortOrder?: number
  enabled?: boolean
  x?: number
  y?: number
  paramValues?: string
}
