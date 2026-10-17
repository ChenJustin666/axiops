<template>
  <ElDialog
    :model-value="visible"
    title="选择 Peer 标签"
    width="600px"
    :close-on-click-modal="false"
    @update:model-value="handleClose"
  >
    <div class="peer-label-selector-dialog">
      <!-- Step 1: Select Namespace -->
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
          @change="handleNamespaceChange"
        >
          <ElOption v-for="ns in allNamespaces" :key="ns" :label="ns" :value="ns" />
        </ElSelect>
      </div>

      <!-- Step 2: Select Service -->
      <div class="selector-step">
        <div class="step-label">
          <span class="step-number">2</span>
          选择服务
        </div>
        <ElSelect
          v-model="selectedApplicationId"
          placeholder="选择服务"
          filterable
          style="width: 100%"
          :disabled="!selectedNamespace"
          @change="handleApplicationChange"
        >
          <ElOption
            v-for="app in applications"
            :key="app.id"
            :label="app.nameCn || app.nameEn"
            :value="app.id"
          >
            <div class="application-option">
              <span class="option-name">{{ app.nameCn || app.nameEn }}</span>
              <ElTag :type="getResourceTypeTag(app.resourceType)" size="small">
                {{ normalizeResourceType(app.resourceType) }}
              </ElTag>
            </div>
          </ElOption>
        </ElSelect>
      </div>

      <!-- Step 3: Select Version -->
      <div class="selector-step">
        <div class="step-label">
          <span class="step-number">3</span>
          选择版本
        </div>
        <ElSelect
          v-model="selectedVersionId"
          placeholder="选择版本"
          filterable
          style="width: 100%"
          :disabled="!selectedApplicationId"
          @change="handleVersionChange"
        >
          <ElOption
            v-for="ver in versions"
            :key="ver.id"
            :label="`${ver.version} (${ver.resourceName})`"
            :value="ver.id"
          />
        </ElSelect>
      </div>

      <!-- Step 4: Choose Labels -->
      <div class="selector-step">
        <div class="step-label">
          <span class="step-number">4</span>
          选择要添加的标签
        </div>
        <div v-if="Object.keys(availableLabels).length > 0" class="labels-grid">
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
        <ElEmpty v-else description="请先选择版本以加载标签" :image-size="80" />
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
  import {
    searchApplicationApi,
    searchVersionApi,
    getVersionLabelsApi,
    type OnecProjectApplication,
    type OnecProjectVersion
  } from '@/api/workload/application'
  import { getClusterNamespaceListApi } from '../../../../api/manager/cluster'
  import { getNamespaceLabelsApi } from '../../../../api/workload/selector'

  interface KeyValue {
    key: string
    value: string
  }

  interface Props {
    modelValue: boolean
    cluster: any // expects cluster object with clusterUuid
  }

  interface Emits {
    (e: 'update:modelValue', value: boolean): void
    (e: 'confirm', labels: KeyValue[]): void
  }

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  const visible = computed({
    get: () => props.modelValue,
    set: (v) => emit('update:modelValue', v)
  })

  // Data
  const allNamespaces = ref<string[]>([])
  const selectedNamespace = ref<string>('')
  const applications = ref<OnecProjectApplication[]>([])
  const selectedApplicationId = ref<number>()
  const versions = ref<OnecProjectVersion[]>([])
  const selectedVersionId = ref<number>()
  const availableLabels = ref<Record<string, string>>({})
  const selectedLabels = ref<Record<string, boolean>>({})

  const loadingNamespaces = ref(false)
  const loadingApplications = ref(false)
  const loadingVersions = ref(false)

  // Helpers
  const normalizeResourceType = (type: string) => type?.toLowerCase() || ''
  const getResourceTypeTag = (type: string) => {
    const rt = normalizeResourceType(type)
    const map: Record<string, string> = {
      pod: 'info',
      deployment: 'success',
      statefulset: 'warning',
      daemonset: 'primary',
      cronjob: 'danger',
      job: ''
    }
    return map[rt] || 'info'
  }

  // Load namespaces on open
  watch(visible, async (val) => {
    if (val) {
      await loadNamespaces()
    }
  })

  const loadNamespaces = async () => {
    if (!props.cluster) return
    loadingNamespaces.value = true
    try {
      const list = await getClusterNamespaceListApi(props.cluster.clusterUuid)
      allNamespaces.value = list
      if (list.length) selectedNamespace.value = list[0]
    } catch (e) {
      console.error('加载命名空间失败', e)
      allNamespaces.value = []
    } finally {
      loadingNamespaces.value = false
    }
  }

  const handleNamespaceChange = async () => {
    // Reset downstream selections
    selectedApplicationId.value = undefined
    applications.value = []
    selectedVersionId.value = undefined
    versions.value = []
    availableLabels.value = {}
    selectedLabels.value = {}
    // Load applications for the selected namespace via workspace -> not implemented, fallback to all applications
    await loadApplications()
  }

  const loadApplications = async () => {
    // In this simplified implementation we load all applications regardless of namespace
    loadingApplications.value = true
    try {
      const resp = await searchApplicationApi({ workspaceId: 0 }) // workspaceId placeholder; adjust as needed
      applications.value = resp || []
    } catch (e) {
      console.error('加载服务失败', e)
      applications.value = []
    } finally {
      loadingApplications.value = false
    }
  }

  const handleApplicationChange = async () => {
    selectedVersionId.value = undefined
    versions.value = []
    availableLabels.value = {}
    selectedLabels.value = {}
    if (!selectedApplicationId.value) return
    loadingVersions.value = true
    try {
      const resp = await searchVersionApi({ applicationId: selectedApplicationId.value })
      versions.value = resp || []
    } catch (e) {
      console.error('加载版本失败', e)
      versions.value = []
    } finally {
      loadingVersions.value = false
    }
  }

  const handleVersionChange = async () => {
    if (!selectedVersionId.value) return
    availableLabels.value = {}
    selectedLabels.value = {}
    try {
      const res = await getVersionLabelsApi(selectedVersionId.value)
      // Assuming response shape { key:string, value:string }[]
      const map: Record<string, string> = {}
      for (const item of res as any) {
        map[item.key] = item.value
      }
      availableLabels.value = map
      // default select all
      for (const k of Object.keys(map)) {
        selectedLabels.value[k] = true
      }
    } catch (e) {
      console.error('加载标签失败', e)
      availableLabels.value = {}
    }
  }

  const handleClose = () => {
    visible.value = false
    // reset all
    selectedNamespace.value = ''
    selectedApplicationId.value = undefined
    selectedVersionId.value = undefined
    applications.value = []
    versions.value = []
    availableLabels.value = {}
    selectedLabels.value = {}
  }

  const handleConfirm = () => {
    const labels: KeyValue[] = []
    for (const [k, v] of Object.entries(availableLabels.value)) {
      if (selectedLabels.value[k]) {
        labels.push({ key: k, value: v })
      }
    }
    emit('confirm', labels)
    handleClose()
  }
</script>

<style lang="scss" scoped>
  @use '../styles/network-policy-form.scss' as *;

  .peer-label-selector-dialog {
    max-height: 60vh;
    overflow-y: auto;

    .selector-step {
      margin-bottom: 16px;
      .step-label {
        display: flex;
        align-items: center;
        font-size: 13px;
        font-weight: 500;
        color: #303133;
        margin-bottom: 8px;
        .step-number {
          display: inline-flex;
          align-items: center;
          justify-content: center;
          width: 20px;
          height: 20px;
          margin-right: 8px;
          background: #409eff;
          color: #fff;
          border-radius: 50%;
          font-size: 12px;
        }
      }
      .application-option {
        display: flex;
        align-items: center;
        justify-content: space-between;
        width: 100%;
        gap: 8px;
        .option-name {
          flex: 1;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }
      }
    }

    .labels-grid {
      max-height: 200px;
      overflow-y: auto;
      padding: 8px;
      background: #f5f7fa;
      border-radius: 4px;
      &::-webkit-scrollbar {
        width: 6px;
      }
      &::-webkit-scrollbar-thumb {
        background: #dcdfe6;
        border-radius: 3px;
      }
      .label-checkbox {
        margin-bottom: 4px;
        padding: 6px 8px;
        background: #fff;
        border-radius: 4px;
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

    .dialog-actions {
      display: flex;
      justify-content: flex-end;
      gap: 8px;
    }
  }
</style>
