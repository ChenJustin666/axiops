<template>
  <div class="peer-config-item">
    <div class="peer-header">
      <ElRadioGroup
        v-model="internalState.selectorType"
        size="small"
        @change="handleSelectorTypeChange"
      >
        <ElRadio value="selector">选择器</ElRadio>
        <ElRadio value="ipblock">IP 块</ElRadio>
      </ElRadioGroup>
      <ElButton type="danger" :icon="Trash2" circle size="small" @click="$emit('remove')" />
    </div>

    <!-- 选择器模式 -->
    <div v-if="internalState.selectorType === 'selector'" class="selector-mode">
      <!-- Pod 选择器 -->
      <div class="selector-section">
        <div class="selector-header">
          <ElCheckbox v-model="internalState.podSelectorEnabled" @change="handlePodSelectorToggle">
            Pod 选择器
          </ElCheckbox>
          <ElTooltip content="选择满足标签条件的 Pod">
            <ElIcon :size="14" style="color: #909399"><HelpCircle /></ElIcon>
          </ElTooltip>
        </div>

        <template v-if="internalState.podSelectorEnabled">
          <div class="pod-mode-selector">
            <ElRadioGroup
              v-model="internalState.podMode"
              size="small"
              @change="handlePodModeChange"
            >
              <ElRadio value="current">当前命名空间</ElRadio>
              <ElRadio value="specific">指定命名空间</ElRadio>
            </ElRadioGroup>
          </div>

          <div v-if="internalState.podMode === 'specific'" class="namespace-select">
            <ElSelect
              v-model="internalState.podNamespace"
              placeholder="选择命名空间"
              filterable
              size="small"
              style="width: 100%"
              @change="handlePodNamespaceChange"
            >
              <ElOption v-for="ns in allNamespaces" :key="ns" :label="ns" :value="ns" />
            </ElSelect>

            <ElAlert
              v-if="
                !internalState.namespaceSelectorEnabled &&
                internalState.podNamespace &&
                internalState.podNamespace !== currentNamespace
              "
              type="warning"
              :closable="false"
              show-icon
              style="margin-top: 8px"
            >
              <template #title> 选择的命名空间与当前不同，建议同时启用 Namespace 选择器 </template>
            </ElAlert>
          </div>

          <div v-if="loadingLabels" class="loading-container">
            <ElIcon class="is-loading" :size="18"><Loader2 /></ElIcon>
            <span>加载中...</span>
          </div>

          <div v-else class="label-editor">
            <div class="key-value-list">
              <div
                v-for="(label, index) in internalState.podLabels"
                :key="'pod-label-' + index"
                class="key-value-item"
              >
                <ElInput v-model="label.key" placeholder="键" size="small" class="key-input" />
                <span class="separator">=</span>
                <ElInput v-model="label.value" placeholder="值" size="small" class="value-input" />
                <ElButton
                  type="danger"
                  :icon="Trash2"
                  circle
                  size="small"
                  @click="removePodLabel(index)"
                />
              </div>
            </div>

            <div class="label-actions">
              <ElButton size="small" @click="addPodLabel">添加标签</ElButton>
              <ElButton type="primary" size="small" plain @click="openPodSelectorDialog">
                从资源选择
              </ElButton>
            </div>
          </div>
        </template>
      </div>

      <!-- Namespace 选择器 -->
      <div class="selector-section">
        <div class="selector-header">
          <ElCheckbox
            v-model="internalState.namespaceSelectorEnabled"
            @change="handleNamespaceSelectorToggle"
          >
            Namespace 选择器
          </ElCheckbox>
          <ElTooltip content="选择命名空间（可与 Pod 选择器组合使用）">
            <ElIcon :size="14" style="color: #909399"><HelpCircle /></ElIcon>
          </ElTooltip>
        </div>

        <template v-if="internalState.namespaceSelectorEnabled">
          <div class="namespace-selector-display">
            <div v-if="internalState.namespaceNamespace" class="selected-namespace">
              <span class="namespace-label">命名空间:</span>
              <ElTag size="small" type="primary">{{ internalState.namespaceNamespace }}</ElTag>
              <ElButton
                type="danger"
                :icon="Trash2"
                circle
                size="small"
                @click="clearNamespaceSelector"
              />
            </div>
            <div v-else class="empty-namespace">
              <span class="empty-text">未选择命名空间</span>
              <ElButton type="primary" size="small" plain @click="openNamespaceLabelSelectorDialog">
                选择命名空间和标签
              </ElButton>
            </div>
          </div>

          <div class="label-editor">
            <div class="key-value-list">
              <div
                v-for="(label, index) in internalState.namespaceLabels"
                :key="'ns-label-' + index"
                class="key-value-item"
              >
                <ElInput v-model="label.key" placeholder="键" size="small" class="key-input" />
                <span class="separator">=</span>
                <ElInput v-model="label.value" placeholder="值" size="small" class="value-input" />
                <ElButton
                  type="danger"
                  :icon="Trash2"
                  circle
                  size="small"
                  @click="removeNamespaceLabel(index)"
                />
              </div>
            </div>

            <div class="label-actions">
              <ElButton size="small" @click="addNamespaceLabel">添加标签</ElButton>
              <ElButton type="primary" size="small" plain @click="openNamespaceLabelSelectorDialog">
                重新选择
              </ElButton>
            </div>
          </div>
        </template>
      </div>

      <!-- 选择器说明 -->
      <ElAlert type="info" :closable="false" show-icon style="margin-top: 12px">
        <template #title>
          <span v-if="internalState.podSelectorEnabled && internalState.namespaceSelectorEnabled">
            同时启用两个选择器时，将选择指定命名空间中满足 Pod 标签条件的 Pod
          </span>
          <span v-else-if="internalState.podSelectorEnabled">
            仅使用 Pod 选择器，作用于当前命名空间或指定命名空间
          </span>
          <span v-else-if="internalState.namespaceSelectorEnabled">
            仅使用 Namespace 选择器，选择整个命名空间
          </span>
          <span v-else>请至少启用一个选择器，否则该配置将不起作用</span>
        </template>
      </ElAlert>
    </div>

    <!-- IPBlock 模式 -->
    <div v-else class="ipblock-mode">
      <ElAlert type="warning" :closable="false" show-icon style="margin-bottom: 12px">
        <template #title>IP 块模式与选择器模式互斥，无法同时使用</template>
      </ElAlert>

      <div class="ipblock-config">
        <div class="cidr-input">
          <ElInput
            v-model="internalState.ipBlockCidr"
            placeholder="CIDR (如: 10.0.0.0/8, 192.168.1.0/24)"
            size="small"
          />
        </div>

        <div class="except-list">
          <div class="except-header">
            <span class="except-label">排除的 IP 段</span>
            <ElButton size="small" :icon="Plus" @click="addExcept">添加</ElButton>
          </div>
          <div
            v-for="(except, index) in internalState.ipBlockExcept"
            :key="'except-' + index"
            class="except-item"
          >
            <ElInput
              v-model="except.value"
              placeholder="CIDR (如: 10.0.1.0/24)"
              size="small"
              class="value-input-full"
            />
            <ElButton
              type="danger"
              :icon="Trash2"
              circle
              size="small"
              @click="removeExcept(index)"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- Pod 标签选择弹窗 -->
    <PodSelectorDialog
      v-model="podSelectorDialogVisible"
      :cluster="cluster"
      :all-namespaces="allNamespaces"
      :current-namespace="currentNamespace"
      :initial-namespace="
        internalState.podMode === 'specific' ? internalState.podNamespace : undefined
      "
      @confirm="handlePodSelectorConfirm"
    />

    <!-- Namespace 标签选择弹窗 -->
    <NamespaceLabelSelectorDialog
      v-model="namespaceLabelSelectorDialogVisible"
      :cluster="cluster"
      :all-namespaces="allNamespaces"
      :initial-namespace="internalState.namespaceNamespace"
      @confirm="handleNamespaceLabelSelectorConfirm"
    />
  </div>
</template>

<script setup lang="ts">
  import { ref, reactive, watch, nextTick } from 'vue'
  import { Trash2, HelpCircle, Plus, Loader2 } from 'lucide-vue-next'
  import { getNamespaceLabelsApi } from '../../../../api/workload/selector'
  import PodSelectorDialog from './PodSelectorDialog.vue'
  import NamespaceLabelSelectorDialog from './NamespaceLabelSelectorDialog.vue'

  interface KeyValue {
    key: string
    value: string
  }

  interface PodSelectorConfig {
    enabled: boolean
    mode: 'current' | 'specific'
    namespace?: string
    labels: KeyValue[]
  }

  interface NamespaceSelectorConfig {
    enabled: boolean
    namespace?: string
    labels: KeyValue[]
  }

  interface IPBlockConfig {
    cidr?: string
    except?: { value: string }[]
  }

  interface NetworkPolicyPeer {
    selectorType: 'selector' | 'ipblock'
    podSelector?: PodSelectorConfig
    namespaceSelector?: NamespaceSelectorConfig
    ipBlock?: IPBlockConfig
  }

  interface Props {
    peer: NetworkPolicyPeer
    allNamespaces: string[]
    currentNamespace?: string
    cluster: any
  }

  interface Emits {
    (e: 'update:peer', value: NetworkPolicyPeer): void
    (e: 'remove'): void
  }

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  // 加载状态
  const loadingLabels = ref(false)

  // 内部状态 - 从 props 初始化
  const internalState = reactive({
    selectorType: props.peer.selectorType,
    podSelectorEnabled: props.peer.podSelector?.enabled ?? false,
    podMode: (props.peer.podSelector?.mode ?? 'current') as 'current' | 'specific',
    podNamespace: props.peer.podSelector?.namespace ?? '',
    podLabels: props.peer.podSelector?.labels ? [...props.peer.podSelector.labels] : [],
    namespaceSelectorEnabled: props.peer.namespaceSelector?.enabled ?? false,
    namespaceNamespace: props.peer.namespaceSelector?.namespace ?? '',
    namespaceLabels: props.peer.namespaceSelector?.labels
      ? [...props.peer.namespaceSelector.labels]
      : [],
    ipBlockCidr: props.peer.ipBlock?.cidr ?? '',
    ipBlockExcept: props.peer.ipBlock?.except ? [...props.peer.ipBlock.except] : []
  })

  // 弹窗状态
  const podSelectorDialogVisible = ref(false)
  const namespaceLabelSelectorDialogVisible = ref(false)

  // 控制循环更新的标志
  let isEmitting = false
  let isUpdatingFromProps = false
  let pendingUpdateTimer: ReturnType<typeof setTimeout> | null = null

  // 同步到父组件（使用防抖避免频繁更新）
  const syncToParent = () => {
    if (isUpdatingFromProps) return // 如果正在从 props 更新，则不同步回去

    if (pendingUpdateTimer) clearTimeout(pendingUpdateTimer)
    pendingUpdateTimer = setTimeout(() => {
      isEmitting = true
      const peer: NetworkPolicyPeer = {
        selectorType: internalState.selectorType,
        podSelector: {
          enabled: internalState.podSelectorEnabled,
          mode: internalState.podMode,
          namespace: internalState.podNamespace || undefined,
          labels: [...internalState.podLabels]
        },
        namespaceSelector: {
          enabled: internalState.namespaceSelectorEnabled,
          namespace: internalState.namespaceNamespace || undefined,
          labels: [...internalState.namespaceLabels]
        },
        ipBlock: {
          cidr: internalState.ipBlockCidr || undefined,
          except: [...internalState.ipBlockExcept]
        }
      }
      emit('update:peer', peer)

      // 延迟重置标志，确保 emit 处理完成
      nextTick(() => {
        setTimeout(() => {
          isEmitting = false
        }, 50)
      })
    }, 150)
  }

  // 监听 props.peer 变化（仅同步到本地状态）
  // 使用浅比较避免不必要的更新
  let lastPeerSnapshot = JSON.stringify(props.peer)

  watch(
    () => props.peer,
    (newPeer) => {
      // 如果正在 emit，则跳过这次更新
      if (isEmitting) {
        return
      }

      const newSnapshot = JSON.stringify(newPeer)
      if (newSnapshot === lastPeerSnapshot) {
        return
      }

      lastPeerSnapshot = newSnapshot
      isUpdatingFromProps = true

      // 更新内部状态
      internalState.selectorType = newPeer.selectorType
      internalState.podSelectorEnabled = newPeer.podSelector?.enabled ?? false
      internalState.podMode = newPeer.podSelector?.mode ?? 'current'
      internalState.podNamespace = newPeer.podSelector?.namespace ?? ''

      if (newPeer.podSelector?.labels) {
        internalState.podLabels = [...newPeer.podSelector.labels]
      } else {
        internalState.podLabels = []
      }

      internalState.namespaceSelectorEnabled = newPeer.namespaceSelector?.enabled ?? false
      internalState.namespaceNamespace = newPeer.namespaceSelector?.namespace ?? ''

      if (newPeer.namespaceSelector?.labels) {
        internalState.namespaceLabels = [...newPeer.namespaceSelector.labels]
      } else {
        internalState.namespaceLabels = []
      }

      internalState.ipBlockCidr = newPeer.ipBlock?.cidr ?? ''
      if (newPeer.ipBlock?.except) {
        internalState.ipBlockExcept = [...newPeer.ipBlock.except]
      } else {
        internalState.ipBlockExcept = []
      }

      // 延迟重置标志
      setTimeout(() => {
        isUpdatingFromProps = false
      }, 100)
    },
    { deep: true }
  )

  // 处理选择器类型切换
  const handleSelectorTypeChange = () => {
    if (internalState.selectorType === 'ipblock') {
      // 切换到 IPBlock 模式，清空选择器
      internalState.podSelectorEnabled = false
      internalState.namespaceSelectorEnabled = false
    }
    syncToParent()
  }

  // 处理 Pod 选择器开关
  const handlePodSelectorToggle = () => {
    syncToParent()
  }

  // 处理 Pod 模式切换
  const handlePodModeChange = () => {
    if (internalState.podMode === 'current') {
      internalState.podNamespace = ''
      internalState.podLabels = []
    }
    syncToParent()
  }

  // 处理 Pod 命名空间变化 - 自动配置 Namespace 选择器
  const handlePodNamespaceChange = async () => {
    const ns = internalState.podNamespace
    if (!ns || !props.cluster) {
      syncToParent()
      return
    }

    // 清空现有 Pod 标签
    internalState.podLabels = []

    // 如果选择的命名空间与当前不同，自动启用并配置 Namespace 选择器
    if (ns !== props.currentNamespace) {
      internalState.namespaceSelectorEnabled = true
      internalState.namespaceNamespace = ns
      internalState.namespaceLabels = []

      // 自动请求该命名空间的标签
      try {
        const res = await getNamespaceLabelsApi({
          clusterUuid: props.cluster.clusterUuid,
          namespace: ns
        })

        // 自动添加 kubernetes.io/metadata.name 标签
        const metaKey = 'kubernetes.io/metadata.name'
        internalState.namespaceLabels = [{ key: metaKey, value: ns }]

        // 如果命名空间有其他标签，也可以自动添加
        if (res.labels && Object.keys(res.labels).length > 0) {
          for (const [key, value] of Object.entries(res.labels)) {
            if (key !== metaKey) {
              internalState.namespaceLabels.push({ key, value })
            }
          }
        }
      } catch (error) {
        console.error('加载 Namespace 标签失败:', error)
        // 即使加载失败，也至少添加 metadata.name 标签
        internalState.namespaceLabels = [{ key: 'kubernetes.io/metadata.name', value: ns }]
      }
    } else {
      // 选择的是当前命名空间，不需要 Namespace 选择器
      internalState.namespaceSelectorEnabled = false
      internalState.namespaceNamespace = ''
      internalState.namespaceLabels = []
    }

    syncToParent()

    // 打开 Pod 标签选择弹窗让用户选择 Pod 标签
    openPodSelectorDialog()
  }

  // 处理 Namespace 选择器开关
  const handleNamespaceSelectorToggle = () => {
    if (!internalState.namespaceSelectorEnabled) {
      internalState.namespaceNamespace = ''
      internalState.namespaceLabels = []
    }
    syncToParent()
  }

  // 清空 Namespace 选择器
  const clearNamespaceSelector = () => {
    internalState.namespaceNamespace = ''
    internalState.namespaceLabels = []
    syncToParent()
  }

  // 打开 Namespace 标签选择弹窗
  const openNamespaceLabelSelectorDialog = () => {
    namespaceLabelSelectorDialogVisible.value = true
  }

  // 处理 Namespace 标签选择器确认
  const handleNamespaceLabelSelectorConfirm = (namespace: string, labels: KeyValue[]) => {
    internalState.namespaceNamespace = namespace
    internalState.namespaceLabels = labels
    syncToParent()
  }

  // 打开 Pod 选择器弹窗
  const openPodSelectorDialog = () => {
    podSelectorDialogVisible.value = true
  }

  // 处理 Pod 选择器确认
  const handlePodSelectorConfirm = (labels: KeyValue[]) => {
    for (const label of labels) {
      const exists = internalState.podLabels.some((l) => l.key === label.key)
      if (!exists) {
        internalState.podLabels.push({ ...label })
      }
    }
    syncToParent()
  }

  // Pod 标签操作
  const addPodLabel = () => {
    internalState.podLabels.push({ key: '', value: '' })
    syncToParent()
  }

  const removePodLabel = (index: number) => {
    internalState.podLabels.splice(index, 1)
    syncToParent()
  }

  // Namespace 标签操作
  const addNamespaceLabel = () => {
    internalState.namespaceLabels.push({ key: '', value: '' })
    syncToParent()
  }

  const removeNamespaceLabel = (index: number) => {
    internalState.namespaceLabels.splice(index, 1)
    syncToParent()
  }

  // IPBlock 操作
  const addExcept = () => {
    internalState.ipBlockExcept.push({ value: '' })
    syncToParent()
  }

  const removeExcept = (index: number) => {
    internalState.ipBlockExcept.splice(index, 1)
    syncToParent()
  }
</script>

<script lang="ts">
  export default {
    name: 'PeerConfigItem'
  }
</script>

<style lang="scss" scoped>
  @use '../styles/network-policy-form.scss' as *;

  .peer-config-item {
    padding: 16px;
    background: #fff;
    border-radius: 4px;
    border: 1px solid #e4e7ed;
    transition: border-color 0.2s;

    &:hover {
      border-color: #c0c4cc;
    }

    .peer-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 16px;
      padding-bottom: 12px;
      border-bottom: 1px solid #ebeef5;

      :deep(.el-radio-group) {
        .el-radio {
          margin-right: 16px;
        }
      }
    }

    .selector-mode {
      .selector-section {
        margin-bottom: 16px;
        padding: 14px;
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

        .selector-header {
          display: flex;
          align-items: center;
          gap: 8px;
          margin-bottom: 12px;
          padding-bottom: 10px;
          border-bottom: 1px dashed #e4e7ed;

          :deep(.el-checkbox__label) {
            font-weight: 500;
            color: #303133;
          }
        }

        .pod-mode-selector {
          margin-bottom: 12px;
          padding: 10px;
          background: #fff;
          border-radius: 4px;
        }

        .namespace-select {
          margin-bottom: 12px;
        }

        .namespace-selector-display {
          padding: 12px;
          background: #fff;
          border-radius: 4px;
          border: 1px solid #ebeef5;
          margin-bottom: 12px;

          .selected-namespace {
            display: flex;
            align-items: center;
            gap: 10px;

            .namespace-label {
              font-size: 13px;
              color: #606266;
              font-weight: 500;
            }
          }

          .empty-namespace {
            display: flex;
            align-items: center;
            justify-content: space-between;

            .empty-text {
              font-size: 13px;
              color: #909399;
            }
          }
        }

        .label-editor {
          .key-value-list {
            margin-bottom: 10px;

            .key-value-item {
              display: flex;
              align-items: center;
              gap: 8px;
              margin-bottom: 8px;
              padding: 8px 10px;
              background: #fff;
              border-radius: 4px;
              border: 1px solid #e4e7ed;
              transition: border-color 0.2s;

              &:hover {
                border-color: #409eff;
              }

              .key-input {
                flex: 0 0 160px;
              }

              .value-input {
                flex: 1;
              }

              .separator {
                color: #909399;
                font-weight: 500;
              }
            }
          }

          .label-actions {
            display: flex;
            gap: 8px;
          }
        }
      }
    }

    .ipblock-mode {
      .ipblock-config {
        padding: 14px;
        background: #f5f7fa;
        border-radius: 4px;
        border: 1px solid #e4e7ed;

        .cidr-input {
          margin-bottom: 16px;
        }

        .except-list {
          .except-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 10px;
            padding-bottom: 8px;
            border-bottom: 1px dashed #e4e7ed;

            .except-label {
              font-size: 13px;
              font-weight: 500;
              color: #606266;
            }
          }

          .except-item {
            display: flex;
            align-items: center;
            gap: 10px;
            margin-bottom: 10px;
            padding: 8px 10px;
            background: #fff;
            border-radius: 4px;
            border: 1px solid #e4e7ed;
            transition: border-color 0.2s;

            &:hover {
              border-color: #409eff;
            }

            .value-input-full {
              flex: 1;
            }
          }
        }
      }
    }

    .loading-container {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
      padding: 20px;
      color: #909399;
      font-size: 13px;
      background: #f5f7fa;
      border-radius: 4px;
    }
  }
</style>
