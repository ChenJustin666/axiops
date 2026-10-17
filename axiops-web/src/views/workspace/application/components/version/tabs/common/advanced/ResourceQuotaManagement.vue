<template>
  <div class="resource-quota-management">
    <!-- 顶部操作栏 -->
    <div class="toolbar">
      <div class="container-selector">
        <span class="selector-label">容器：</span>
        <ElSelect
          v-model="selectedContainer"
          placeholder="选择容器"
          style="width: 220px"
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
        <ElTag
          v-if="currentContainerType"
          :type="currentContainerType === 'init' ? 'warning' : 'primary'"
          size="small"
        >
          {{ currentContainerType === 'init' ? 'Init 容器' : 'Main 容器' }}
        </ElTag>
      </div>

      <div class="toolbar-actions">
        <ElButton v-if="!editing" type="primary" size="small" @click="startEdit">
          <Edit :size="14" />
          编辑
        </ElButton>
        <template v-else>
          <ElButton type="primary" size="small" :loading="saving" @click="handleSaveAll">
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

    <!-- 加载状态 -->
    <div v-if="loading" v-loading="loading" class="loading-wrapper"></div>

    <!-- 主内容 -->
    <div v-else-if="selectedContainer" class="config-wrapper">
      <ElTable :data="quotaTableData" border size="small" class="quota-table">
        <ElTableColumn prop="resource" label="资源类型" width="100" align="center">
          <template #default="{ row }">
            <div class="resource-cell">
              <Cpu v-if="row.type === 'cpu'" :size="15" class="icon-cpu" />
              <HardDrive v-else :size="15" class="icon-memory" />
              <span>{{ row.resource }}</span>
            </div>
          </template>
        </ElTableColumn>

        <ElTableColumn label="Requests（最小保证）" min-width="200">
          <template #default="{ row }">
            <ElInput
              v-model="row.requests"
              :placeholder="row.type === 'cpu' ? '如: 100m, 0.5, 1' : '如: 128Mi, 1Gi'"
              :disabled="!editing"
              size="small"
              clearable
            />
          </template>
        </ElTableColumn>

        <ElTableColumn label="Limits（最大限制）" min-width="200">
          <template #default="{ row }">
            <ElInput
              v-model="row.limits"
              :placeholder="row.type === 'cpu' ? '如: 200m, 1, 2' : '如: 256Mi, 2Gi'"
              :disabled="!editing"
              size="small"
              clearable
            />
          </template>
        </ElTableColumn>

        <ElTableColumn v-if="editing" label="操作" width="70" align="center">
          <template #default="{ row }">
            <ElButton
              size="small"
              text
              type="danger"
              :disabled="!row.requests && !row.limits"
              @click="clearRow(row.type)"
            >
              <Trash2 :size="14" />
            </ElButton>
          </template>
        </ElTableColumn>
      </ElTable>

      <div class="tips">
        <span><strong>Requests</strong>：调度保证的最小资源</span>
        <span class="divider">|</span>
        <span><strong>Limits</strong>：CPU 超限限流，内存超限 OOM Kill</span>
      </div>
    </div>

    <ElEmpty v-else description="请选择一个容器" :image-size="80" />
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, watch, onMounted } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { Cpu, HardDrive, Edit, Save, Trash2, RefreshCw } from 'lucide-vue-next'
  import {
    getResourceQuotaApi,
    updateResourceQuotaApi,
    type OnecProjectVersion,
    type ContainerResources,
    type ResourceRequirements
  } from '@/api'

  interface Props {
    version: OnecProjectVersion
    refreshTrigger?: number
  }

  const props = defineProps<Props>()
  const emit = defineEmits<{ success: [] }>()

  const loading = ref(false)
  const saving = ref(false)
  const editing = ref(false)

  const containers = ref<ContainerResources[]>([])
  const selectedContainer = ref('')
  const originalData = ref<ContainerResources[]>([])

  // 表格数据
  const cpuRow = ref({ type: 'cpu', resource: 'CPU', requests: '', limits: '' })
  const memoryRow = ref({ type: 'memory', resource: '内存', requests: '', limits: '' })
  const quotaTableData = computed(() => [cpuRow.value, memoryRow.value])

  const initContainers = computed(() => containers.value.filter((c) => c.containerType === 'init'))
  const mainContainers = computed(() => containers.value.filter((c) => c.containerType === 'main'))
  const currentContainerType = computed(() => {
    const container = containers.value.find((c) => c.containerName === selectedContainer.value)
    return container?.containerType || ''
  })

  const loadResourceQuota = async () => {
    // 🔧 修复：防止重复加载
    if (loading.value) return

    loading.value = true
    try {
      const response = await getResourceQuotaApi(props.version.id)
      containers.value = response.containers || []

      if (containers.value.length > 0 && !selectedContainer.value) {
        const firstMain = mainContainers.value[0]
        const firstInit = initContainers.value[0]
        selectedContainer.value = firstMain?.containerName || firstInit?.containerName || ''
        updateCurrentQuota()
      }

      originalData.value = JSON.parse(JSON.stringify(containers.value))
    } catch (error) {
      console.error('[资源配额] 加载失败:', error)
      ElMessage.error('加载资源配额配置失败')
    } finally {
      loading.value = false
    }
  }

  const updateCurrentQuota = () => {
    const container = containers.value.find((c) => c.containerName === selectedContainer.value)
    if (container) {
      cpuRow.value.requests = container.resources.requests?.cpu || ''
      cpuRow.value.limits = container.resources.limits?.cpu || ''
      memoryRow.value.requests = container.resources.requests?.memory || ''
      memoryRow.value.limits = container.resources.limits?.memory || ''
    }
  }

  const syncCurrentQuota = () => {
    const container = containers.value.find((c) => c.containerName === selectedContainer.value)
    if (!container) return

    const resources: ResourceRequirements = {}

    if (cpuRow.value.requests || memoryRow.value.requests) {
      resources.requests = {}
      if (cpuRow.value.requests) resources.requests.cpu = cpuRow.value.requests
      if (memoryRow.value.requests) resources.requests.memory = memoryRow.value.requests
    }

    if (cpuRow.value.limits || memoryRow.value.limits) {
      resources.limits = {}
      if (cpuRow.value.limits) resources.limits.cpu = cpuRow.value.limits
      if (memoryRow.value.limits) resources.limits.memory = memoryRow.value.limits
    }

    container.resources = resources
  }

  const handleContainerChange = () => {
    if (editing.value) syncCurrentQuota()
    updateCurrentQuota()
  }

  const startEdit = () => {
    editing.value = true
  }

  const cancelEdit = () => {
    containers.value = JSON.parse(JSON.stringify(originalData.value))
    updateCurrentQuota()
    editing.value = false
    ElMessage.info('已取消编辑')
  }

  /** 刷新数据 */
  const handleRefresh = async () => {
    await loadResourceQuota()
    ElMessage.success('刷新成功')
  }

  const clearRow = (type: string) => {
    if (type === 'cpu') {
      cpuRow.value.requests = ''
      cpuRow.value.limits = ''
    } else {
      memoryRow.value.requests = ''
      memoryRow.value.limits = ''
    }
  }

  const handleSaveAll = async () => {
    syncCurrentQuota()

    try {
      await ElMessageBox.confirm('确定要保存所有容器的资源配额配置吗？', '确认保存', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      })

      saving.value = true

      const requestData = {
        containers: containers.value.map((container) => ({
          containerName: container.containerName,
          containerType: container.containerType,
          resources: container.resources
        }))
      }

      await updateResourceQuotaApi(props.version.id, requestData)

      ElMessage.success('资源配额保存成功')
      editing.value = false
      await loadResourceQuota()
      emit('success')
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('[资源配额] 保存失败:', error)
        ElMessage.error('保存失败：' + (error.message || '未知错误'))
      }
    } finally {
      saving.value = false
    }
  }

  watch(
    () => props.refreshTrigger,
    (newVal) => {
      if (newVal && newVal > 0) {
        editing.value = false
        loadResourceQuota()
      }
    }
  )

  onMounted(() => {
    loadResourceQuota()
  })

  defineExpose({
    editing,
    hasUnsavedChanges: () => {
      if (!editing.value) return false
      syncCurrentQuota()
      return JSON.stringify(containers.value) !== JSON.stringify(originalData.value)
    }
  })
</script>

<style lang="scss" scoped>
  .resource-quota-management {
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

      .container-selector {
        display: flex;
        align-items: center;
        gap: 10px;

        .selector-label {
          font-size: 13px;
          color: #606266;
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

    .container-option {
      display: flex;
      justify-content: space-between;
      align-items: center;
      width: 100%;
    }

    .loading-wrapper {
      flex: 1;
      min-height: 150px;
    }

    .config-wrapper {
      flex: 1;
      padding: 16px;
      overflow-y: auto;

      .quota-table {
        .resource-cell {
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 6px;
          font-weight: 500;
          font-size: 13px;

          .icon-cpu {
            color: #0ea5e9;
          }

          .icon-memory {
            color: #22c55e;
          }
        }
      }

      .tips {
        margin-top: 12px;
        padding: 8px 12px;
        background: #f5f7fa;
        border-radius: 4px;
        font-size: 12px;
        color: #606266;

        strong {
          color: #303133;
        }

        .divider {
          color: #dcdfe6;
          margin: 0 8px;
        }
      }
    }

    :deep(.el-empty) {
      padding: 40px 0;
    }
  }
</style>
