<template>
  <ElDialog
    :model-value="visible"
    title="选择 Pod 标签"
    width="650px"
    :close-on-click-modal="false"
    @update:model-value="handleClose"
  >
    <div class="pod-label-selector-dialog">
      <!-- 步骤 1: 选择资源类型 -->
      <div class="selector-step">
        <div class="step-label">
          <span class="step-number">1</span>
          选择资源类型
        </div>
        <ElSelect
          v-model="selectedResourceType"
          placeholder="选择资源类型"
          style="width: 100%"
          @change="handleResourceTypeChange"
        >
          <ElOption label="Deployment" value="deployment" />
          <ElOption label="StatefulSet" value="statefulset" />
          <ElOption label="DaemonSet" value="daemonset" />
          <ElOption label="Job" value="job" />
          <ElOption label="CronJob" value="cronjob" />
        </ElSelect>
      </div>

      <!-- 步骤 2: 选择资源 -->
      <div class="selector-step">
        <div class="step-label">
          <span class="step-number">2</span>
          选择资源
        </div>
        <ElSelect
          v-model="selectedResourceName"
          placeholder="选择资源"
          filterable
          style="width: 100%"
          :disabled="!selectedResourceType"
          :loading="loadingResources"
          @change="handleResourceChange"
        >
          <ElOption
            v-for="resource in resources"
            :key="resource.name"
            :label="resource.name"
            :value="resource.name"
          />
        </ElSelect>
      </div>

      <!-- 步骤 3: 选择要添加的标签 -->
      <div class="selector-step">
        <div class="step-label">
          <span class="step-number">3</span>
          选择要添加的标签
        </div>
        <div v-if="loadingLabels" class="loading-container">
          <ElIcon class="is-loading" :size="24"><Loader2 /></ElIcon>
          <span>加载标签中...</span>
        </div>
        <div v-else-if="Object.keys(availableLabels).length > 0" class="labels-grid">
          <div
            v-for="[key, value] in Object.entries(availableLabels)"
            :key="key"
            class="label-checkbox"
          >
            <ElCheckbox v-model="selectedLabels[key]">
              <span class="label-key">{{ key }}</span>
              <span class="label-value">= {{ value }}</span>
            </ElCheckbox>
          </div>
        </div>
        <ElEmpty
          v-else
          :description="!selectedResourceName ? '请先选择资源以加载标签' : '该资源没有标签'"
          :image-size="80"
        />
      </div>
    </div>

    <template #footer>
      <div class="dialog-actions">
        <ElButton @click="handleClose">取消</ElButton>
        <ElButton
          type="primary"
          :disabled="Object.keys(availableLabels).length === 0"
          @click="handleConfirm"
        >
          确认添加
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, computed, watch } from 'vue'
  import { Loader2 } from 'lucide-vue-next'
  import {
    getWorkloadResourceListApi,
    getWorkloadPodSelectorApi
  } from '../../../../api/workload/selector'

  interface KeyValue {
    key: string
    value: string
  }

  interface Props {
    modelValue: boolean
    cluster: any
    namespace: string
  }

  interface Emits {
    (e: 'update:modelValue', value: boolean): void
    (e: 'confirm', labels: KeyValue[]): void
  }

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  const visible = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  // 加载状态
  const loadingResources = ref(false)
  const loadingLabels = ref(false)

  // 选择状态
  const selectedResourceType = ref<'deployment' | 'statefulset' | 'daemonset' | 'job' | 'cronjob'>()
  const selectedResourceName = ref('')

  // 数据
  const resources = ref<Array<{ name: string }>>([])
  const availableLabels = ref<Record<string, string>>({})
  const selectedLabels = ref<Record<string, boolean>>({})

  // 资源类型变化
  const handleResourceTypeChange = () => {
    selectedResourceName.value = ''
    resources.value = []
    availableLabels.value = {}
    selectedLabels.value = {}
  }

  // 资源变化
  const handleResourceChange = async () => {
    availableLabels.value = {}
    selectedLabels.value = {}

    if (!selectedResourceName.value) return

    loadingLabels.value = true
    try {
      const response = await getWorkloadPodSelectorApi({
        clusterUuid: props.cluster.clusterUuid,
        namespace: props.namespace,
        type: selectedResourceType.value!,
        name: selectedResourceName.value
      })
      availableLabels.value = response.labels || response.matchLabels || {}

      // 默认全选
      selectedLabels.value = {}
      Object.keys(availableLabels.value).forEach((k) => {
        selectedLabels.value[k] = true
      })
    } catch (error) {
      console.error('加载标签失败:', error)
      availableLabels.value = {}
    } finally {
      loadingLabels.value = false
    }
  }

  // 监听资源类型变化，自动加载资源列表
  watch(selectedResourceType, async (type) => {
    if (!type) return

    loadingResources.value = true
    try {
      const response = await getWorkloadResourceListApi({
        clusterUuid: props.cluster.clusterUuid,
        namespace: props.namespace,
        type: type
      })
      resources.value = response.items || []
    } catch (error) {
      console.error('加载资源列表失败:', error)
      resources.value = []
    } finally {
      loadingResources.value = false
    }
  })

  const handleClose = () => {
    visible.value = false
    // 延迟重置状态，等待动画完成
    setTimeout(() => {
      selectedResourceType.value = undefined
      selectedResourceName.value = ''
      resources.value = []
      availableLabels.value = {}
      selectedLabels.value = {}
    }, 300)
  }

  const handleConfirm = () => {
    const labels: KeyValue[] = []
    for (const [key, value] of Object.entries(availableLabels.value)) {
      if (selectedLabels.value[key]) {
        labels.push({ key, value })
      }
    }
    emit('confirm', labels)
    handleClose()
  }
</script>

<style lang="scss" scoped>
  @use '../styles/network-policy-form.scss' as *;

  .pod-label-selector-dialog {
    max-height: 60vh;
    overflow-y: auto;
    padding-right: 4px;

    &::-webkit-scrollbar {
      width: 6px;
    }

    &::-webkit-scrollbar-thumb {
      background: #dcdfe6;
      border-radius: 3px;

      &:hover {
        background: #c0c4cc;
      }
    }

    &::-webkit-scrollbar-track {
      background: #f5f7fa;
    }

    .selector-step {
      margin-bottom: 20px;
      padding: 16px;
      background: #f5f7fa;
      border-radius: 4px;
      border: 1px solid #e4e7ed;
      transition: border-color 0.2s;

      &:hover {
        border-color: #c0c4cc;
      }

      &:last-child {
        margin-bottom: 0;
      }

      .step-label {
        display: flex;
        align-items: center;
        font-size: 14px;
        font-weight: 500;
        color: #303133;
        margin-bottom: 12px;
        padding-bottom: 10px;
        border-bottom: 1px dashed #e4e7ed;

        .step-number {
          display: inline-flex;
          align-items: center;
          justify-content: center;
          width: 22px;
          height: 22px;
          margin-right: 10px;
          background: #409eff;
          color: #fff;
          border-radius: 50%;
          font-size: 12px;
          font-weight: 600;
        }
      }
    }

    .loading-container {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 10px;
      padding: 24px;
      color: #909399;
      background: #fff;
      border-radius: 4px;
    }

    .labels-grid {
      max-height: 220px;
      overflow-y: auto;
      padding: 10px;
      background: #fff;
      border-radius: 4px;
      border: 1px solid #ebeef5;

      &::-webkit-scrollbar {
        width: 6px;
      }

      &::-webkit-scrollbar-thumb {
        background: #dcdfe6;
        border-radius: 3px;
      }

      .label-checkbox {
        margin-bottom: 6px;
        padding: 8px 10px;
        background: #f5f7fa;
        border-radius: 4px;
        border: 1px solid transparent;
        transition: all 0.2s;

        &:hover {
          border-color: #409eff;
          background: #ecf5ff;
        }

        &:last-child {
          margin-bottom: 0;
        }

        .label-key {
          font-weight: 500;
          color: #303133;
        }

        .label-value {
          color: #606266;
          margin-left: 8px;
        }
      }
    }
  }

  .dialog-actions {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
  }
</style>
