// store/workload/metadata.ts
import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export interface MetadataConfig {
  nameCn: string // 中文名
  nameEn: string // 英文名
  version: string // 版本
  resourceName: string // 资源名称 (K8s metadata.name)
  desc: string // 描述
  labels: Record<string, string> // 自定义标签
  annotations: Record<string, string> // 自定义注解
}

// createNewService 模式下用户选择的目标位置
export interface TargetLocation {
  clusterId: number
  clusterUuid: string
  clusterName: string
  workspaceId: number
  workspaceName: string
  namespace: string
}

export const useMetadataStore = defineStore('workload-metadata', () => {
  // 元数据配置
  const metadata = ref<MetadataConfig>({
    nameCn: '',
    nameEn: '',
    version: '',
    resourceName: '',
    desc: '',
    labels: {},
    annotations: {}
  })

  // 命名空间（只读，由路由参数决定）
  const namespace = ref('default')

  // 资源类型（pod/deployment/statefulset等）
  const resourceType = ref<'pod' | 'deployment' | 'statefulset' | 'daemonset' | 'job' | 'cronjob'>(
    'pod'
  )

  // 是否手动修改过 resourceName（用于判断是否需要自动更新）
  const resourceNameManuallySet = ref<boolean>(false)

  // createNewService 模式下用户选择的目标位置
  const targetLocation = ref<TargetLocation | null>(null)

  // 监听 nameEn 和 version 变化，自动更新 resourceName
  watch(
    () => [metadata.value.nameEn, metadata.value.version],
    ([newNameEn, newVersion]) => {
      // 只有在没有手动设置过的情况下才自动更新
      if (!resourceNameManuallySet.value && newNameEn && newVersion) {
        metadata.value.resourceName = `${newNameEn}-${newVersion}`
        console.log(`自动更新 resourceName: ${metadata.value.resourceName}`)
      }
    },
    { immediate: true }
  )

  // 更新元数据
  function updateMetadata(data: Partial<MetadataConfig>) {
    // 如果更新中包含 resourceName，标记为手动设置
    if (data.resourceName !== undefined && data.resourceName !== metadata.value.resourceName) {
      resourceNameManuallySet.value = true
      console.log(`resourceName 已手动设置为: ${data.resourceName}`)
    }
    metadata.value = { ...metadata.value, ...data }
  }

  // 设置资源名称
  function setResourceName(name: string) {
    metadata.value.resourceName = name
    // 标记为手动设置，阻止自动更新
    if (name && name.trim()) {
      resourceNameManuallySet.value = true
      console.log(`✋ resourceName 已手动设置为: ${name}`)
    }
  }

  // 重置资源名称为自动模式
  function resetResourceNameToAuto() {
    resourceNameManuallySet.value = false
    if (metadata.value.nameEn && metadata.value.version) {
      metadata.value.resourceName = `${metadata.value.nameEn}-${metadata.value.version}`
      console.log(`🔄 resourceName 已重置为自动模式: ${metadata.value.resourceName}`)
    }
  }

  // 检查 resourceName 是否为自定义
  function isResourceNameCustomized(): boolean {
    return resourceNameManuallySet.value
  }

  // 设置命名空间
  function setNamespace(ns: string) {
    namespace.value = ns
    console.log(`命名空间已设置为: ${ns}`)
  }

  // 设置资源类型
  function setResourceType(type: typeof resourceType.value) {
    resourceType.value = type
  }

  // 设置 createNewService 模式下的目标位置
  function setTargetLocation(location: TargetLocation | null) {
    targetLocation.value = location
    if (location) {
      namespace.value = location.namespace
    }
  }

  // 获取 createNewService 模式下的目标位置
  function getTargetLocation(): TargetLocation | null {
    return targetLocation.value
  }

  // 添加标签
  function addLabel(key: string, value: string) {
    metadata.value.labels = { ...metadata.value.labels, [key]: value }
  }

  // 移除标签
  function removeLabel(key: string) {
    const newLabels = { ...metadata.value.labels }
    delete newLabels[key]
    metadata.value.labels = newLabels
  }

  // 更新所有标签
  function setLabels(labels: Record<string, string>) {
    metadata.value.labels = { ...labels }
  }

  // 添加注解
  function addAnnotation(key: string, value: string) {
    metadata.value.annotations = { ...metadata.value.annotations, [key]: value }
  }

  // 移除注解
  function removeAnnotation(key: string) {
    const newAnnotations = { ...metadata.value.annotations }
    delete newAnnotations[key]
    metadata.value.annotations = newAnnotations
  }

  // 更新所有注解
  function setAnnotations(annotations: Record<string, string>) {
    metadata.value.annotations = { ...annotations }
  }

  // 重置状态
  function reset() {
    metadata.value = {
      nameCn: '',
      nameEn: '',
      version: '',
      resourceName: '',
      desc: '',
      labels: {},
      annotations: {}
    }
    namespace.value = 'default'
    resourceType.value = 'pod'
    resourceNameManuallySet.value = false
    targetLocation.value = null
  }

  // 验证元数据
  function validate() {
    const errors: string[] = []
    const warnings: string[] = []

    if (!metadata.value.nameCn?.trim()) {
      errors.push('请输入中文名称')
    }

    if (!metadata.value.nameEn?.trim()) {
      errors.push('请输入英文名称')
    } else if (!/^[a-z][a-z0-9-]*[a-z0-9]$/.test(metadata.value.nameEn)) {
      errors.push('英文名称必须以小写字母开头，只能包含小写字母、数字和连字符，且不能以连字符结尾')
    }

    if (!metadata.value.version?.trim()) {
      errors.push('请输入版本号')
    } else if (!/^[a-zA-Z0-9][a-zA-Z0-9.-]*$/.test(metadata.value.version)) {
      errors.push('版本号必须以字母或数字开头，只能包含字母、数字、点和连字符')
    }

    // 验证资源名称（K8s metadata.name）
    if (!metadata.value.resourceName?.trim()) {
      errors.push('资源名称不能为空')
    } else {
      // K8s 资源名称规则：DNS子域名格式
      if (!/^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/.test(metadata.value.resourceName)) {
        errors.push(
          '资源名称格式不正确（只能包含小写字母、数字和连字符，必须以字母或数字开头和结尾）'
        )
      }
      if (metadata.value.resourceName.length > 253) {
        errors.push('资源名称长度不能超过 253 个字符')
      }
    }

    // 如果资源名称是自定义的，给出警告
    if (resourceNameManuallySet.value) {
      warnings.push(
        '⚠️ 您已自定义资源名称，不推荐此操作。建议使用默认的"英文名-版本"格式以避免资源名冲突'
      )
    }

    // 验证标签格式
    for (const [key, value] of Object.entries(metadata.value.labels)) {
      if (!key || !value) {
        errors.push('标签的键和值不能为空')
        break
      }
      if (
        !/^([a-z0-9]([-a-z0-9.]*[a-z0-9])?\/)?[a-z0-9A-Z]([-a-z0-9A-Z._]*[a-z0-9A-Z])?$/.test(key)
      ) {
        errors.push(`标签键 "${key}" 格式不正确`)
        break
      }
    }

    return {
      valid: errors.length === 0,
      errors,
      warnings
    }
  }

  // 从现有数据加载（用于编辑模式）
  function loadFromExisting(data: {
    nameCn: string
    nameEn: string
    version: string
    resourceName?: string
    desc?: string
    labels?: Record<string, string>
    annotations?: Record<string, string>
  }) {
    // 先设置基础字段
    metadata.value = {
      nameCn: data.nameCn,
      nameEn: data.nameEn,
      version: data.version,
      resourceName: data.resourceName || `${data.nameEn}-${data.version}`,
      desc: data.desc || '',
      labels: data.labels || {},
      annotations: data.annotations || {}
    }

    // 判断 resourceName 是否为自定义
    const expectedResourceName = `${data.nameEn}-${data.version}`
    if (data.resourceName && data.resourceName !== expectedResourceName) {
      resourceNameManuallySet.value = true
      console.log(`✋ 加载时检测到自定义 resourceName: ${data.resourceName}`)
    } else {
      resourceNameManuallySet.value = false
    }
  }

  return {
    // State
    metadata,
    namespace,
    resourceType,
    resourceNameManuallySet,
    targetLocation,

    // Actions
    updateMetadata,
    setResourceName,
    resetResourceNameToAuto,
    isResourceNameCustomized,
    setNamespace,
    setResourceType,
    setTargetLocation,
    getTargetLocation,
    addLabel,
    removeLabel,
    setLabels,
    addAnnotation,
    removeAnnotation,
    setAnnotations,
    reset,
    validate,
    loadFromExisting
  }
})
