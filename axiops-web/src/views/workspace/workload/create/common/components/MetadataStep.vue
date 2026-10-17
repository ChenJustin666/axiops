<template>
  <div class="metadata-step">
    <ElForm ref="formRef" :model="formData" :rules="rules" label-width="85px" class="step-form">
      <!-- 命名空间显示 -->
      <ElRow :gutter="24">
        <ElCol :span="22">
          <ElFormItem>
            <template #label>
              <span class="label-text">
                目标位置
                <ElTooltip
                  v-if="!isCreateNewServiceMode"
                  content="当前服务所属的命名空间，不可修改"
                  placement="top"
                >
                  <Info :size="13" class="hint-icon" />
                </ElTooltip>
                <ElTooltip v-else content="选择服务要部署到的集群和工作空间" placement="top">
                  <Info :size="13" class="hint-icon" />
                </ElTooltip>
              </span>
            </template>

            <!-- createNewService 模式下显示集群和工作空间选择器 -->
            <div v-if="isCreateNewServiceMode" class="namespace-selector">
              <ElSelect
                v-model="selectedClusterId"
                placeholder="选择集群"
                filterable
                style="width: 200px; margin-right: 8px"
                :loading="loadingClusters"
                @change="handleClusterChange"
              >
                <ElOption
                  v-for="cluster in clusters"
                  :key="cluster.id"
                  :label="cluster.clusterName"
                  :value="cluster.id"
                />
              </ElSelect>

              <ElSelect
                v-model="selectedWorkspaceId"
                placeholder="选择工作空间"
                filterable
                style="flex: 1"
                :loading="loadingWorkspaces"
                :disabled="!selectedClusterId || clusters.length === 0"
                @change="handleWorkspaceChange"
              >
                <ElOption
                  v-for="ws in workspaces"
                  :key="ws.id"
                  :label="`${ws.name} (${ws.namespace})`"
                  :value="ws.id"
                />
              </ElSelect>
            </div>

            <!-- 其他模式下显示只读的 namespace -->
            <ElInput v-else :value="namespace" disabled>
              <template #prefix>
                <Folder :size="14" />
              </template>
            </ElInput>
          </ElFormItem>
        </ElCol>
      </ElRow>

      <!-- 创建新服务模式的提示信息 -->
      <ElRow v-if="isCreateNewServiceMode && sourceVersionInfo" :gutter="24">
        <ElCol :span="22">
          <ElAlert type="info" :closable="false" show-icon class="source-info-alert">
            <template #title>
              <span class="info-text">
                基于
                {{ sourceVersionInfo.applicationNameCn || sourceVersionInfo.applicationNameEn }} 的
                {{ sourceVersionInfo.version }} 版本创建新服务
              </span>
            </template>
          </ElAlert>
        </ElCol>
      </ElRow>

      <!-- 第一行 -->
      <ElRow :gutter="24">
        <ElCol :span="11">
          <ElFormItem prop="nameCn" required>
            <template #label>
              <span class="label-text">
                中文名
                <ElTooltip content="服务的中文显示名称，用于界面展示" placement="top">
                  <Info :size="13" class="hint-icon" />
                </ElTooltip>
              </span>
            </template>
            <ElInput
              v-model="formData.nameCn"
              placeholder="请输入服务中文名"
              clearable
              :disabled="!isFieldEditableLocal('nameCn')"
              @input="handleInputChange"
              @blur="handleFieldBlur('nameCn')"
            />
          </ElFormItem>
        </ElCol>

        <ElCol :span="11">
          <ElFormItem prop="nameEn" required>
            <template #label>
              <span class="label-text">
                英文名
                <ElTooltip
                  content="必须以英文字母开头，只能包含小写字母、数字和中横线，不能以中横线结尾"
                  placement="top"
                >
                  <Info :size="13" class="hint-icon" />
                </ElTooltip>
              </span>
            </template>
            <ElInput
              v-model="formData.nameEn"
              placeholder="请输入服务英文名"
              :disabled="!isFieldEditableLocal('nameEn')"
              @input="handleEnglishNameInput"
              @blur="handleFieldBlur('nameEn')"
              clearable
            />
          </ElFormItem>
        </ElCol>
      </ElRow>

      <!-- 第二行 -->
      <ElRow :gutter="24">
        <ElCol :span="11">
          <ElFormItem prop="version" required>
            <template #label>
              <span class="label-text">
                版本
                <ElTooltip
                  content="版本必须以英文字母开头，只能包含英文字母、数字和中横线"
                  placement="top"
                >
                  <Info :size="13" class="hint-icon" />
                </ElTooltip>
              </span>
            </template>
            <ElInput
              v-model="formData.version"
              placeholder="请输入版本号（如: v1-0-0）"
              clearable
              :disabled="!isFieldEditableLocal('version')"
              @input="handleVersionInput"
              @blur="handleFieldBlur('version')"
            />
          </ElFormItem>
        </ElCol>

        <ElCol :span="11">
          <ElFormItem prop="resourceName" required>
            <template #label>
              <span class="label-text">
                资源名称
                <ElTooltip
                  content="Kubernetes 资源的 metadata.name，默认为：英文名-版本"
                  placement="top"
                >
                  <Info :size="13" class="hint-icon" />
                </ElTooltip>
              </span>
            </template>
            <ElInput
              v-model="formData.resourceName"
              placeholder="自动生成：英文名-版本"
              :disabled="!isFieldEditableLocal('resourceName')"
              @input="handleResourceNameInput"
              @blur="handleFieldBlur('resourceName')"
              clearable
            >
              <template #append>
                <ElButton
                  v-if="isResourceNameCustomized"
                  @click="resetResourceName"
                  :icon="RefreshCw"
                  size="small"
                  title="重置为自动生成"
                >
                  重置
                </ElButton>
              </template>
            </ElInput>
          </ElFormItem>
        </ElCol>
      </ElRow>

      <!-- 资源名称警告提示 -->
      <ElRow v-if="isResourceNameCustomized" :gutter="24">
        <ElCol :span="22">
          <ElAlert type="warning" :closable="false" show-icon class="resource-name-warning">
            <template #title>
              <span class="warning-text">
                ⚠️ 不推荐自定义资源名称，建议使用默认的"英文名-版本"格式以避免资源名冲突
              </span>
            </template>
          </ElAlert>
        </ElCol>
      </ElRow>

      <!-- 描述 -->
      <ElRow :gutter="24">
        <ElCol :span="22">
          <ElFormItem>
            <template #label>
              <span class="label-text">
                描述
                <ElTooltip content="Pod的描述信息，用于说明其用途" placement="top">
                  <Info :size="13" class="hint-icon" />
                </ElTooltip>
              </span>
            </template>
            <ElInput
              v-model="formData.desc"
              type="textarea"
              :rows="2"
              placeholder="请输入Pod描述（选填）"
              :disabled="!isFieldEditableLocal('desc')"
              @input="handleInputChange"
            />
          </ElFormItem>
        </ElCol>
      </ElRow>

      <!-- 标签部分 -->
      <ElDivider content-position="left">
        <span class="divider-title">
          标签配置
          <ElTooltip content="标签用于标识和选择Pod，格式为键值对" placement="top">
            <Info :size="12" class="divider-hint" />
          </ElTooltip>
        </span>
      </ElDivider>

      <div class="config-section">
        <ElButton type="primary" size="small" @click="addLabel" class="add-btn">
          <Plus :size="14" />
          添加标签
        </ElButton>

        <div class="items-list">
          <div v-for="(label, index) in labels" :key="index" class="item-row">
            <ElInput
              v-model="label.key"
              placeholder="标签键"
              class="item-key"
              @input="handleInputChange"
            />
            <span class="equals">=</span>
            <ElInput
              v-model="label.value"
              placeholder="标签值"
              class="item-value"
              @input="handleInputChange"
            />
            <ElButton type="danger" circle size="small" @click="removeLabel(index)">
              <Trash2 :size="14" />
            </ElButton>
          </div>
          <ElEmpty v-if="labels.length === 0" description="暂无标签" :image-size="60" />
        </div>
      </div>

      <!-- 注解部分 -->
      <ElDivider content-position="left">
        <span class="divider-title">
          注解配置
          <ElTooltip content="注解用于存储Pod的非标识性元数据" placement="top">
            <Info :size="12" class="divider-hint" />
          </ElTooltip>
        </span>
      </ElDivider>

      <div class="config-section">
        <ElButton type="primary" size="small" @click="addAnnotation" class="add-btn">
          <Plus :size="14" />
          添加注解
        </ElButton>

        <div class="items-list">
          <div v-for="(annotation, index) in annotations" :key="index" class="item-row">
            <ElInput
              v-model="annotation.key"
              placeholder="注解键"
              class="item-key"
              @input="handleInputChange"
            />
            <span class="equals">=</span>
            <ElInput
              v-model="annotation.value"
              placeholder="注解值"
              class="item-value"
              @input="handleInputChange"
            />
            <ElButton type="danger" circle size="small" @click="removeAnnotation(index)">
              <Trash2 :size="14" />
            </ElButton>
          </div>
          <ElEmpty v-if="annotations.length === 0" description="暂无注解" :image-size="60" />
        </div>
      </div>
    </ElForm>
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, reactive, watch, onMounted, inject, type ComputedRef } from 'vue'
  import { useMetadataStore } from '@/store/workload'
  import { isFieldEditable, type WorkloadMode } from '../utils/modeHandler'
  import { Info, Folder, Plus, Trash2, RefreshCw } from 'lucide-vue-next'
  import type { FormInstance, FormRules } from 'element-plus'
  import { searchProjectClusterApi, searchProjectWorkspaceApi } from '@/api'
  import type { ProjectCluster, ProjectWorkspace } from '@/api'

  // Props
  interface Props {
    mode: WorkloadMode
    // createNewService 模式下需要传入的额外参数
    sourceVersionInfo?: {
      applicationNameCn: string
      applicationNameEn: string
      version: string
    } | null
  }

  const props = defineProps<Props>()

  // Emits
  const emit = defineEmits<{
    (e: 'validate', result: { valid: boolean; errors: string[] }): void
  }>()

  // Store
  const metadataStore = useMetadataStore()

  // 从父组件注入的参数
  const appProjectId = inject<ComputedRef<number>>('appProjectId')
  const injectedClusterId = inject<ComputedRef<number>>('resourceClusterId')
  const injectedClusterUuid = inject<ComputedRef<string>>('clusterUuid')
  const injectedWorkspaceId = inject<ComputedRef<number>>('workspaceId')

  // Form ref
  const formRef = ref<FormInstance>()

  // 表单数据
  const formData = reactive({
    nameCn: '',
    nameEn: '',
    version: '',
    resourceName: '',
    desc: ''
  })

  // 标签和注解（使用数组形式）
  const labels = ref<Array<{ key: string; value: string }>>([])
  const annotations = ref<Array<{ key: string; value: string }>>([])

  // ========== createNewService 模式下的集群和工作空间选择 ==========
  const clusters = ref<ProjectCluster[]>([])
  const workspaces = ref<ProjectWorkspace[]>([])
  const selectedClusterId = ref<number | null>(null)
  const selectedWorkspaceId = ref<number | null>(null)
  const loadingClusters = ref(false)
  const loadingWorkspaces = ref(false)

  // 判断是否为创建新服务模式
  const isCreateNewServiceMode = computed(() => props.mode === 'createNewService')

  // 加载集群列表
  const loadClusters = async () => {
    const projectId = appProjectId?.value
    if (!projectId || projectId === 0) {
      return
    }

    loadingClusters.value = true
    try {
      const response = await searchProjectClusterApi({ projectId })
      clusters.value = response || []
    } catch (error) {
      console.error('加载集群列表失败:', error)
    } finally {
      loadingClusters.value = false
    }
  }

  // 加载工作空间列表
  const loadWorkspaces = async (clusterId: number) => {
    loadingWorkspaces.value = true
    try {
      const response = await searchProjectWorkspaceApi({ projectClusterId: clusterId })
      workspaces.value = response || []
    } catch (error) {
      console.error('加载工作空间列表失败:', error)
    } finally {
      loadingWorkspaces.value = false
    }
  }

  // 集群变化处理
  const handleClusterChange = (clusterId: number | null) => {
    selectedWorkspaceId.value = null
    workspaces.value = []

    if (clusterId) {
      loadWorkspaces(clusterId)
    }

    // 更新 store 中的目标位置
    updateTargetLocation()
  }

  // 工作空间变化处理
  const handleWorkspaceChange = () => {
    updateTargetLocation()
  }

  // 更新目标位置到 store
  const updateTargetLocation = () => {
    if (!selectedClusterId.value || !selectedWorkspaceId.value) {
      metadataStore.setTargetLocation(null)
      return
    }

    const cluster = clusters.value.find((c) => c.id === selectedClusterId.value)
    const workspace = workspaces.value.find((w) => w.id === selectedWorkspaceId.value)

    if (cluster && workspace) {
      metadataStore.setTargetLocation({
        clusterId: cluster.id,
        clusterUuid: cluster.clusterUuid,
        clusterName: cluster.clusterName,
        workspaceId: workspace.id,
        workspaceName: workspace.name,
        namespace: workspace.namespace
      })
    }
  }

  // 从 store 同步到本地
  watch(
    () => metadataStore.metadata,
    (newVal) => {
      formData.nameCn = newVal.nameCn
      formData.nameEn = newVal.nameEn
      formData.version = newVal.version
      formData.resourceName = newVal.resourceName
      formData.desc = newVal.desc
    },
    { deep: true }
  )

  // 初始化时同步标签和注解
  function initFromStore() {
    const {
      nameCn,
      nameEn,
      version,
      resourceName,
      desc,
      labels: storeLabels,
      annotations: storeAnnotations
    } = metadataStore.metadata

    formData.nameCn = nameCn
    formData.nameEn = nameEn
    formData.version = version
    formData.resourceName = resourceName
    formData.desc = desc

    // 同步标签
    labels.value = Object.entries(storeLabels).map(([key, value]) => ({ key, value }))

    // 同步注解
    annotations.value = Object.entries(storeAnnotations).map(([key, value]) => ({ key, value }))
  }

  // 命名空间
  const namespace = computed(() => metadataStore.namespace)

  // 检查 resourceName 是否为自定义
  const isResourceNameCustomized = computed(() => metadataStore.isResourceNameCustomized())

  // 表单验证规则
  const rules: FormRules = {
    nameCn: [
      { required: true, message: '请输入中文名', trigger: 'blur' },
      { min: 2, max: 50, message: '长度在 2 到 50 个字符', trigger: 'blur' }
    ],
    nameEn: [
      { required: true, message: '请输入英文名', trigger: 'blur' },
      {
        pattern: /^[a-z][a-z0-9-]*[a-z0-9]$/,
        message: '必须以小写字母开头，只能包含小写字母、数字和中横线，不能以中横线结尾',
        trigger: 'blur'
      },
      { min: 2, max: 63, message: '长度在 2 到 63 个字符', trigger: 'blur' }
    ],
    version: [
      { required: true, message: '请输入版本号', trigger: 'blur' },
      {
        pattern: /^[a-zA-Z][a-zA-Z0-9-]*$/,
        message: '必须以字母开头，只能包含字母、数字和中横线',
        trigger: 'blur'
      },
      { min: 1, max: 30, message: '长度在 1 到 30 个字符', trigger: 'blur' }
    ],
    resourceName: [
      { required: true, message: '请输入资源名称', trigger: 'blur' },
      {
        pattern: /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/,
        message: '资源名称格式不正确（只能包含小写字母、数字和连字符，必须以字母或数字开头和结尾）',
        trigger: 'blur'
      },
      { max: 253, message: '资源名称长度不能超过 253 个字符', trigger: 'blur' }
    ]
  }

  // 检查字段是否可编辑
  function isFieldEditableLocal(
    field: 'nameCn' | 'nameEn' | 'version' | 'resourceName' | 'desc'
  ): boolean {
    return isFieldEditable(props.mode, field)
  }

  // 处理输入变化
  function handleInputChange() {
    syncToStore()
    emitValidation()
  }

  // 处理英文名输入（自动转小写）
  function handleEnglishNameInput(value: string) {
    formData.nameEn = value.toLowerCase()
    syncToStore()
    emitValidation()
  }

  // 处理版本号输入
  function handleVersionInput(value: string) {
    formData.version = value
    syncToStore()
    emitValidation()
  }

  // 处理资源名称输入
  function handleResourceNameInput(value: string) {
    formData.resourceName = value.toLowerCase()
    metadataStore.setResourceName(formData.resourceName)
    emitValidation()
  }

  // 重置资源名称为自动生成
  function resetResourceName() {
    metadataStore.resetResourceNameToAuto()
    formData.resourceName = metadataStore.metadata.resourceName
    ElMessage.success('已重置为自动生成模式')
    emitValidation()
  }

  // 处理字段失焦
  function handleFieldBlur(field: string) {
    formRef.value?.validateField(field)
  }

  // 同步到 store
  function syncToStore() {
    metadataStore.updateMetadata({
      nameCn: formData.nameCn,
      nameEn: formData.nameEn,
      version: formData.version,
      resourceName: formData.resourceName,
      desc: formData.desc
    })

    // 同步标签（包括空的，让用户能正常输入）
    const allLabels: Record<string, string> = {}
    labels.value.forEach((label) => {
      // 只要有 key 就保存，允许 value 为空（正在输入中）
      if (label.key) {
        allLabels[label.key] = label.value || ''
      }
    })
    metadataStore.setLabels(allLabels)

    // 同步注解（包括空的，让用户能正常输入）
    const allAnnotations: Record<string, string> = {}
    annotations.value.forEach((annotation) => {
      // 只要有 key 就保存，允许 value 为空（正在输入中）
      if (annotation.key) {
        allAnnotations[annotation.key] = annotation.value || ''
      }
    })
    metadataStore.setAnnotations(allAnnotations)
  }

  // 标签操作
  function addLabel() {
    labels.value.push({ key: '', value: '' })
  }

  function removeLabel(index: number) {
    labels.value.splice(index, 1)
    syncToStore()
    emitValidation()
  }

  // 注解操作
  function addAnnotation() {
    annotations.value.push({ key: '', value: '' })
  }

  function removeAnnotation(index: number) {
    annotations.value.splice(index, 1)
    syncToStore()
    emitValidation()
  }

  // 发送验证结果
  function emitValidation() {
    const validation = metadataStore.validate()
    emit('validate', {
      valid: validation.valid,
      errors: validation.errors
    })
  }

  // 表单验证
  async function validate(): Promise<boolean> {
    if (!formRef.value) {
      return false
    }

    try {
      await formRef.value.validate()
      const storeValidation = metadataStore.validate()

      if (!storeValidation.valid) {
        return false
      }

      return true
    } catch (error) {
      return false
    }
  }

  // 重置表单
  function resetForm() {
    formRef.value?.resetFields()
    labels.value = []
    annotations.value = []
  }

  // 标记是否已初始化集群和工作空间
  const locationInitialized = ref(false)

  // 初始化集群和工作空间选择
  const initLocationSelector = async () => {
    if (!isCreateNewServiceMode.value) {
      return
    }

    const projectId = appProjectId?.value
    const defaultClusterId = injectedClusterId?.value
    const defaultWorkspaceId = injectedWorkspaceId?.value

    // 必须等待所有必要的参数都准备好
    if (!projectId || projectId === 0) {
      return
    }

    // 首次初始化
    if (locationInitialized.value) {
      return
    }

    await loadClusters()

    if (defaultClusterId && defaultClusterId > 0 && clusters.value.length > 0) {
      // 使用路由参数的默认值
      const cluster = clusters.value.find((c) => c.id === defaultClusterId)
      if (cluster) {
        selectedClusterId.value = defaultClusterId
        await loadWorkspaces(defaultClusterId)

        // 设置默认选中的工作空间
        if (defaultWorkspaceId && defaultWorkspaceId > 0 && workspaces.value.length > 0) {
          const workspace = workspaces.value.find((w) => w.id === defaultWorkspaceId)
          if (workspace) {
            selectedWorkspaceId.value = defaultWorkspaceId
            // 保存到 store，这样表单/YAML 切换时可以恢复
            updateTargetLocation()
          }
        }
        locationInitialized.value = true
      }
    } else if (clusters.value.length > 0) {
      // 如果没有默认值但有集群列表，选择第一个集群
      const firstCluster = clusters.value[0]
      selectedClusterId.value = firstCluster.id
      await loadWorkspaces(firstCluster.id)
      if (workspaces.value.length > 0) {
        selectedWorkspaceId.value = workspaces.value[0].id
        updateTargetLocation()
      }
      locationInitialized.value = true
    }
  }

  // 初始化
  onMounted(async () => {
    initFromStore()
    emitValidation()

    // createNewService 模式下初始化集群和工作空间选择
    if (isCreateNewServiceMode.value) {
      // 检查是否已有 targetLocation（表单/YAML 切换时恢复）
      const existingTarget = metadataStore.getTargetLocation()
      if (existingTarget && existingTarget.clusterId && existingTarget.workspaceId) {
        // 已有 targetLocation，直接恢复选择器的值
        const restored = await restoreLocationFromStore()
        if (!restored) {
          // 恢复失败（可能 projectId 还没准备好），等待 watch 触发
        }
      } else {
        // 首次初始化，尝试加载
        initLocationSelector()
      }
    }
  })

  // 从 store 恢复集群和工作空间选择（表单/YAML 切换时使用）
  const restoreLocationFromStore = async () => {
    const existingTarget = metadataStore.getTargetLocation()
    if (!existingTarget || !existingTarget.clusterId || !existingTarget.workspaceId) {
      return false
    }

    const projectId = appProjectId?.value
    if (!projectId || projectId === 0) {
      return false
    }

    await loadClusters()
    if (clusters.value.length > 0) {
      selectedClusterId.value = existingTarget.clusterId
      await loadWorkspaces(existingTarget.clusterId)
      selectedWorkspaceId.value = existingTarget.workspaceId
    }
    locationInitialized.value = true
    return true
  }

  // 监听所有必要参数的变化，当父组件设置好路由参数后再初始化
  watch(
    [() => appProjectId?.value, () => injectedClusterId?.value, () => injectedWorkspaceId?.value],
    async ([projectId, clusterId, workspaceId]) => {
      // 如果已经初始化过，不再重复初始化
      if (locationInitialized.value) {
        return
      }
      // 只有当所有必要参数都准备好时才初始化
      if (
        isCreateNewServiceMode.value &&
        projectId &&
        projectId > 0 &&
        clusterId &&
        clusterId > 0 &&
        workspaceId &&
        workspaceId > 0
      ) {
        // 检查是否已有 targetLocation（表单/YAML 切换时恢复）
        const existingTarget = metadataStore.getTargetLocation()
        if (existingTarget && existingTarget.clusterId && existingTarget.workspaceId) {
          await restoreLocationFromStore()
        } else {
          await initLocationSelector()
        }
      }
    },
    { immediate: true }
  )

  // 监听源版本信息变化
  watch(
    () => props.sourceVersionInfo,
    (newInfo) => {
      if (newInfo) {
        // 可以在这里做一些处理
      }
    },
    { immediate: true }
  )

  // 导出供父组件使用
  defineExpose({
    validate,
    resetForm,
    formRef
  })
</script>

<style lang="scss" scoped>
  .metadata-step {
    padding: 8px 0;

    .step-form {
      :deep(.el-form-item) {
        margin-bottom: 18px;
      }

      :deep(.el-form-item__label) {
        font-size: 13px;
        color: #606266;
        font-weight: normal;
      }

      .label-text {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        white-space: nowrap;

        .hint-icon {
          color: #909399;
          cursor: help;
          transition: color 0.2s;

          &:hover {
            color: #409eff;
          }
        }
      }

      // 资源名称警告提示样式
      .resource-name-warning {
        margin-bottom: 18px;

        .warning-text {
          font-size: 13px;
          line-height: 1.5;
        }
      }

      .el-divider {
        margin: 24px 0 16px;

        .divider-title {
          display: inline-flex;
          align-items: center;
          gap: 6px;
          font-size: 14px;
          font-weight: 500;
          color: #303133;

          .divider-hint {
            color: #909399;
            cursor: help;

            &:hover {
              color: #409eff;
            }
          }
        }
      }

      .config-section {
        margin-bottom: 20px;

        .add-btn {
          margin-bottom: 12px;
        }

        .items-list {
          border-radius: 4px;
          padding: 12px;
          min-height: 80px;

          .item-row {
            display: flex;
            align-items: center;
            gap: 8px;
            margin-bottom: 8px;
            padding: 8px;
            border-radius: 4px;

            &:last-child {
              margin-bottom: 0;
            }

            .item-key {
              width: 200px;
            }

            .item-value {
              flex: 1;
              max-width: 300px;
            }

            .equals {
              color: #909399;
              font-weight: 500;
              padding: 0 4px;
            }
          }

          :deep(.el-empty) {
            padding: 16px 0;

            .el-empty__description {
              margin-top: 8px;
              font-size: 13px;
            }
          }
        }
      }

      // 命名空间选择器样式
      .namespace-selector {
        display: flex;
        align-items: center;
        width: 100%;
      }

      // 源版本信息提示样式
      .source-info-alert {
        margin-bottom: 18px;

        .info-text {
          font-size: 13px;
          line-height: 1.5;
        }
      }
    }
  }
</style>
