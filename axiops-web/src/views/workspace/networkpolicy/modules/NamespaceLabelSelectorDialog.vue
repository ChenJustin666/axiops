<template>
  <ElDialog
    :model-value="visible"
    title="选择 Namespace 标签"
    width="600px"
    :close-on-click-modal="false"
    @update:model-value="handleClose"
  >
    <div class="namespace-label-selector-dialog">
      <!-- 步骤 1: 选择命名空间 -->
      <div class="selector-step">
        <div class="step-label">
          <span class="step-number">1</span>
          选择命名空间
        </div>
        <ElSelect
          v-model="selectedNamespace"
          placeholder="选择命名空间"
          filterable
          style="width: 100%"
          :loading="loadingLabels"
          @change="handleNamespaceChange"
        >
          <ElOption v-for="ns in allNamespaces" :key="ns" :label="ns" :value="ns" />
        </ElSelect>
      </div>

      <!-- 步骤 2: 选择要添加的标签 -->
      <div v-if="selectedNamespace" class="selector-step">
        <div class="step-label">
          <span class="step-number">2</span>
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
        <ElEmpty v-else description="该命名空间没有标签" :image-size="80" />
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
  import { getNamespaceLabelsApi } from '../../../../api/workload/selector'

  interface KeyValue {
    key: string
    value: string
  }

  interface Props {
    modelValue: boolean
    cluster: any
    allNamespaces: string[]
    initialNamespace?: string
  }

  interface Emits {
    (e: 'update:modelValue', value: boolean): void
    (e: 'confirm', namespace: string, labels: KeyValue[]): void
  }

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  const visible = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  // 加载状态
  const loadingLabels = ref(false)

  // 选择状态
  const selectedNamespace = ref('')
  const availableLabels = ref<Record<string, string>>({})
  const selectedLabels = ref<Record<string, boolean>>({})

  // 监听弹窗打开
  watch(visible, (val) => {
    if (val) {
      // 初始化命名空间选择
      if (props.initialNamespace && props.allNamespaces.includes(props.initialNamespace)) {
        selectedNamespace.value = props.initialNamespace
        loadNamespaceLabels(props.initialNamespace)
      } else if (props.allNamespaces.length > 0) {
        selectedNamespace.value = ''
      }
    } else {
      // 关闭时重置状态
      setTimeout(() => {
        selectedNamespace.value = ''
        availableLabels.value = {}
        selectedLabels.value = {}
      }, 300)
    }
  })

  // 加载命名空间标签
  const loadNamespaceLabels = async (namespace: string) => {
    if (!namespace || !props.cluster) return

    availableLabels.value = {}
    selectedLabels.value = {}

    loadingLabels.value = true
    try {
      const res = await getNamespaceLabelsApi({
        clusterUuid: props.cluster.clusterUuid,
        namespace
      })
      availableLabels.value = res.labels || {}

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

  // 命名空间变化
  const handleNamespaceChange = () => {
    if (selectedNamespace.value) {
      loadNamespaceLabels(selectedNamespace.value)
    }
  }

  const handleClose = () => {
    visible.value = false
  }

  const handleConfirm = () => {
    const labels: KeyValue[] = []
    for (const [key, value] of Object.entries(availableLabels.value)) {
      if (selectedLabels.value[key]) {
        labels.push({ key, value })
      }
    }

    // 自动添加 kubernetes.io/metadata.name 标签
    const metaKey = 'kubernetes.io/metadata.name'
    if (selectedNamespace.value && !selectedLabels.value[metaKey]) {
      labels.push({
        key: metaKey,
        value: selectedNamespace.value
      })
    }

    emit('confirm', selectedNamespace.value, labels)
    handleClose()
  }
</script>

<style lang="scss" scoped>
  @use '../styles/network-policy-form.scss' as *;

  .namespace-label-selector-dialog {
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
