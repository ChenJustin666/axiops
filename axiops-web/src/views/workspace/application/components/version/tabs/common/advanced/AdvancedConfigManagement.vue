<template>
  <div class="advanced-config-management">
    <!-- 顶部操作栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <ElRadioGroup v-model="activeSection" size="small">
          <ElRadioButton value="pod">
            <Box :size="14" />
            <span>Pod 配置</span>
          </ElRadioButton>
          <ElRadioButton value="container">
            <Container :size="14" />
            <span>容器配置</span>
          </ElRadioButton>
        </ElRadioGroup>

        <!-- 容器选择器 - 仅在容器配置时显示 -->
        <div v-if="activeSection === 'container'" class="container-selector">
          <span class="selector-label">容器：</span>
          <ElSelect
            v-model="selectedContainer"
            placeholder="选择容器"
            style="width: 200px"
            :loading="loading"
            @change="handleContainerChange"
          >
            <ElOptionGroup v-if="initContainers.length > 0" label="Init 容器">
              <ElOption
                v-for="container in initContainers"
                :key="container.containerName"
                :label="container.containerName"
                :value="container.containerName"
              >
                <div class="container-option">
                  <span>{{ container.containerName }}</span>
                  <ElTag size="small" type="warning">Init</ElTag>
                </div>
              </ElOption>
            </ElOptionGroup>
            <ElOptionGroup v-if="mainContainers.length > 0" label="Main 容器">
              <ElOption
                v-for="container in mainContainers"
                :key="container.containerName"
                :label="container.containerName"
                :value="container.containerName"
              >
                <div class="container-option">
                  <span>{{ container.containerName }}</span>
                  <ElTag size="small" type="primary">Main</ElTag>
                </div>
              </ElOption>
            </ElOptionGroup>
          </ElSelect>
        </div>
      </div>

      <!-- 操作按钮 -->
      <div class="toolbar-actions">
        <template v-if="!editing">
          <ElButton type="primary" size="small" @click="startEdit">
            <Edit :size="14" />
            编辑
          </ElButton>
        </template>
        <template v-else>
          <ElButton type="primary" size="small" :loading="saving" @click="handleSave">
            <Save :size="14" />
            保存
          </ElButton>
          <ElButton size="small" @click="cancelEdit">取消</ElButton>
        </template>
        <!-- 刷新按钮 -->
        <ElButton
          size="small"
          :loading="loading"
          :disabled="editing"
          @click="handleRefresh"
          title="刷新数据"
        >
          <RefreshCw :size="14" />
        </ElButton>
      </div>
    </div>

    <!-- 编辑模式提示 -->
    <div v-if="editing" class="edit-mode-banner">
      <AlertCircle :size="16" />
      <span>当前处于编辑模式，修改后请点击"保存"按钮提交更改</span>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" v-loading="loading" class="loading-wrapper"></div>

    <!-- 主内容区域 -->
    <div v-else class="config-content">
      <!-- Pod 配置部分 -->
      <div v-show="activeSection === 'pod'" class="section-content">
        <PodConfigForm
          ref="podConfigRef"
          :config="podConfig"
          :editing="editing"
          @update:config="handlePodConfigUpdate"
        />
      </div>

      <!-- 容器配置部分 -->
      <div v-show="activeSection === 'container'" class="section-content">
        <ContainerConfigForm
          v-if="selectedContainer && currentContainerConfig"
          ref="containerConfigRef"
          :config="currentContainerConfig"
          :editing="editing"
          @update:config="handleContainerConfigUpdate"
        />
        <ElEmpty v-else description="请选择一个容器" :image-size="80" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, watch, onMounted } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { Box, Container, Edit, Save, AlertCircle, RefreshCw } from 'lucide-vue-next'
  import {
    getAdvancedConfigApi,
    updateAdvancedConfigApi,
    type OnecProjectVersion,
    type PodAdvancedConfig,
    type ContainerAdvancedConfig,
    type UpdateAdvancedConfigRequest
  } from '@/api'
  import PodConfigForm from './advanced-config/PodConfigForm.vue'
  import ContainerConfigForm from './advanced-config/ContainerConfigForm.vue'

  interface Props {
    version: OnecProjectVersion
    refreshTrigger?: number
  }

  const props = defineProps<Props>()
  const emit = defineEmits<{ success: [] }>()

  // ==================== 状态定义 ====================
  const loading = ref(false)
  const saving = ref(false)
  const editing = ref(false)
  const activeSection = ref<'pod' | 'container'>('pod')

  // 数据
  const podConfig = ref<PodAdvancedConfig>({})
  const containers = ref<ContainerAdvancedConfig[]>([])
  const selectedContainer = ref('')

  // 原始数据备份（用于取消编辑时恢复）
  const originalData = ref<{
    podConfig: PodAdvancedConfig
    containers: ContainerAdvancedConfig[]
  } | null>(null)

  // 组件引用
  const podConfigRef = ref<InstanceType<typeof PodConfigForm>>()
  const containerConfigRef = ref<InstanceType<typeof ContainerConfigForm>>()

  // ==================== 计算属性 ====================

  /** Init 容器列表 */
  const initContainers = computed(() => {
    return containers.value.filter((c) => c.containerType === 'init')
  })

  /** Main 容器列表 */
  const mainContainers = computed(() => {
    return containers.value.filter((c) => c.containerType === 'main')
  })

  /** 当前选中的容器配置 */
  const currentContainerConfig = computed(() => {
    if (!selectedContainer.value) return null
    return containers.value.find((c) => c.containerName === selectedContainer.value) || null
  })

  // ==================== 核心方法 ====================

  /** 加载高级配置 */
  const loadAdvancedConfig = async () => {
    // 🔧 修复：防止重复加载
    if (loading.value) return

    loading.value = true
    try {
      const response = await getAdvancedConfigApi(props.version.id)

      // 初始化 podConfig
      podConfig.value = response.podConfig || {}

      // 初始化容器配置
      containers.value = response.containers || []

      // 默认选中第一个容器（优先选择 main 容器）
      if (containers.value.length > 0 && !selectedContainer.value) {
        const firstMain = mainContainers.value[0]
        const firstInit = initContainers.value[0]
        selectedContainer.value = firstMain?.containerName || firstInit?.containerName || ''
      }

      // 备份原始数据
      originalData.value = {
        podConfig: JSON.parse(JSON.stringify(podConfig.value)),
        containers: JSON.parse(JSON.stringify(containers.value))
      }
    } catch (error) {
      console.error('[高级配置] 加载失败:', error)
      ElMessage.error('加载高级配置失败')
    } finally {
      loading.value = false
    }
  }

  /** 处理 Pod 配置更新 */
  const handlePodConfigUpdate = (newConfig: PodAdvancedConfig) => {
    podConfig.value = newConfig
  }

  /** 处理容器配置更新 */
  const handleContainerConfigUpdate = (newConfig: ContainerAdvancedConfig) => {
    const index = containers.value.findIndex((c) => c.containerName === newConfig.containerName)
    if (index !== -1) {
      containers.value[index] = newConfig
    }
  }

  /** 切换容器 */
  const handleContainerChange = () => {
    // 容器切换时不需要特殊处理，computed 会自动更新
  }

  // ==================== 编辑操作 ====================

  const startEdit = () => {
    // 进入编辑模式前先备份当前数据
    originalData.value = {
      podConfig: JSON.parse(JSON.stringify(podConfig.value)),
      containers: JSON.parse(JSON.stringify(containers.value))
    }
    editing.value = true
  }

  const cancelEdit = () => {
    // 恢复原始数据
    if (originalData.value) {
      podConfig.value = JSON.parse(JSON.stringify(originalData.value.podConfig))
      containers.value = JSON.parse(JSON.stringify(originalData.value.containers))
    }
    editing.value = false
    ElMessage.info('已取消编辑')
  }

  /** 刷新数据 */
  const handleRefresh = async () => {
    await loadAdvancedConfig()
    ElMessage.success('刷新成功')
  }

  /** 清理对象，移除空值但保留 0 和 false */
  const cleanObject = (obj: any): any => {
    if (obj === null || obj === undefined) return undefined
    if (Array.isArray(obj)) {
      const cleaned = obj.map((item) => cleanObject(item)).filter((item) => item !== undefined)
      return cleaned.length > 0 ? cleaned : undefined
    }
    if (typeof obj === 'object') {
      const cleaned: any = {}
      for (const [key, value] of Object.entries(obj)) {
        const cleanedValue = cleanObject(value)
        // 保留数字 0 和布尔值 false，但移除空字符串、undefined、空数组、空对象
        if (cleanedValue !== undefined && cleanedValue !== '') {
          if (Array.isArray(cleanedValue) && cleanedValue.length === 0) continue
          if (
            typeof cleanedValue === 'object' &&
            !Array.isArray(cleanedValue) &&
            Object.keys(cleanedValue).length === 0
          )
            continue
          cleaned[key] = cleanedValue
        }
      }
      return Object.keys(cleaned).length > 0 ? cleaned : undefined
    }
    // 保留数字 0 和布尔值 false
    if (obj === '') return undefined
    return obj
  }

  /** 构建提交数据 */
  const buildSubmitData = (): UpdateAdvancedConfigRequest => {
    // 清理 Pod 配置
    const cleanedPodConfig = cleanObject(podConfig.value) || {}

    // 清理容器配置，确保每个容器都有 containerName 和 containerType
    const cleanedContainers = containers.value.map((container) => {
      const cleaned = cleanObject(container) || {}
      // 确保必要字段存在
      cleaned.containerName = container.containerName
      cleaned.containerType = container.containerType
      return cleaned
    })

    return {
      podConfig: cleanedPodConfig,
      containers: cleanedContainers
    }
  }

  const handleSave = async () => {
    try {
      await ElMessageBox.confirm('确定要保存高级配置吗？', '确认保存', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      })

      saving.value = true

      const requestData = buildSubmitData()
      console.log('[高级配置] 提交数据:', JSON.stringify(requestData, null, 2))

      await updateAdvancedConfigApi(props.version.id, requestData)

      ElMessage.success('高级配置保存成功')
      editing.value = false
      await loadAdvancedConfig()
      emit('success')
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('[高级配置] 保存失败:', error)
        ElMessage.error('保存失败：' + (error.message || '未知错误'))
      }
    } finally {
      saving.value = false
    }
  }

  // ==================== 生命周期 ====================

  watch(
    () => props.refreshTrigger,
    (newVal) => {
      if (newVal && newVal > 0) {
        editing.value = false
        loadAdvancedConfig()
      }
    }
  )

  onMounted(() => {
    loadAdvancedConfig()
  })

  // ==================== 暴露给父组件 ====================
  defineExpose({
    editing,
    hasUnsavedChanges: () => {
      if (!editing.value) return false
      const current = JSON.stringify({ podConfig: podConfig.value, containers: containers.value })
      const original = JSON.stringify(originalData.value)
      return current !== original
    }
  })
</script>

<style lang="scss" scoped>
  .advanced-config-management {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: #fff;

    .toolbar {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 12px 16px;
      border-bottom: 1px solid var(--el-border-color-lighter);
      flex-shrink: 0;
      background: #fff;

      .toolbar-left {
        display: flex;
        align-items: center;
        gap: 16px;

        :deep(.el-radio-group) {
          .el-radio-button__inner {
            display: inline-flex;
            align-items: center;
            gap: 4px;
          }
        }

        .container-selector {
          display: flex;
          align-items: center;
          gap: 8px;
          padding-left: 16px;
          border-left: 1px solid var(--el-border-color);

          .selector-label {
            font-size: 13px;
            color: #606266;
            flex-shrink: 0;
          }
        }
      }

      .toolbar-actions {
        display: flex;
        gap: 8px;

        :deep(.el-button) {
          display: inline-flex;
          align-items: center;
          gap: 4px;
        }
      }
    }

    .edit-mode-banner {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 10px 16px;
      background: linear-gradient(135deg, #fdf6ec 0%, #fef0e6 100%);
      border-bottom: 1px solid #f5dab1;
      color: #e6a23c;
      font-size: 13px;
      font-weight: 500;
    }

    .container-option {
      display: flex;
      justify-content: space-between;
      align-items: center;
      width: 100%;
    }

    .loading-wrapper {
      flex: 1;
      display: flex;
      align-items: center;
      justify-content: center;
      min-height: 200px;
    }

    .config-content {
      flex: 1;
      overflow-y: auto;
      background: #f5f7fa;

      .section-content {
        min-height: 100%;
      }
    }

    :deep(.el-empty) {
      padding: 60px 0;
    }
  }
</style>
