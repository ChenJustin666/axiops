<template>
  <div class="host-scheduling-config">
    <div class="section-header">
      <Server :size="18" />
      <h4>主机调度策略</h4>
    </div>

    <!-- 调度模式选择 -->
    <div class="mode-selector">
      <div
        v-for="item in modes"
        :key="item.value"
        class="mode-card"
        :class="{ active: mode === item.value, disabled: !editing }"
        @click="editing && handleModeChange(item.value)"
      >
        <component :is="item.icon" :size="20" />
        <div class="mode-info">
          <div class="mode-name">{{ item.label }}</div>
          <div class="mode-desc">{{ item.desc }}</div>
        </div>
        <Check v-if="mode === item.value" :size="18" class="check-icon" />
      </div>
    </div>

    <!-- ==================== 指定节点模式 ==================== -->
    <div v-if="mode === 'nodeName'" class="mode-content">
      <ElAlert type="info" :closable="false" show-icon>
        <template #title>选择具体节点运行，跳过调度器检查</template>
      </ElAlert>

      <ElSelect
        v-model="nodeNameValue"
        :disabled="!editing"
        :loading="nodeSelectLoading"
        filterable
        remote
        reserve-keyword
        :remote-method="handleNodeRemoteSearch"
        placeholder="请输入节点名称或IP搜索"
        size="large"
        class="full-width"
        @visible-change="handleNodeSelectVisibleChange"
      >
        <ElOption
          v-for="node in nodeList"
          :key="node.id"
          :label="node.nodeName"
          :value="node.nodeName"
        >
          <div class="node-option">
            <span class="node-name">{{ node.nodeName }}</span>
            <div class="node-meta">
              <ElTag size="small" :type="node.nodeStatus === 'Ready' ? 'success' : 'danger'">
                {{ node.nodeStatus }}
              </ElTag>
              <span class="node-ip">{{ node.nodeIp }}</span>
            </div>
          </div>
        </ElOption>

        <template #footer>
          <div class="select-footer">
            <span class="page-info">已加载 {{ nodeList.length }} / {{ nodeTotalCount }} 个</span>
            <ElButton
              v-if="nodeList.length < nodeTotalCount"
              type="primary"
              link
              size="small"
              :loading="nodeSelectLoading"
              @click.stop="loadMoreNodes"
            >
              加载更多
            </ElButton>
          </div>
        </template>
      </ElSelect>
    </div>

    <!-- ==================== 节点标签选择器模式 ==================== -->
    <div v-if="mode === 'nodeSelector'" class="mode-content">
      <ElAlert type="info" :closable="false" show-icon>
        <template #title>通过标签匹配选择节点，经过调度器检查</template>
      </ElAlert>

      <div class="action-bar">
        <ElButton v-if="editing" type="primary" size="small" @click="openLabelPicker('selector')">
          <Database :size="14" />
          从节点标签选择
        </ElButton>
        <ElButton
          v-if="editing"
          type="success"
          size="small"
          @click="openCustomLabelDialog('selector')"
        >
          <Plus :size="14" />
          自定义标签
        </ElButton>
      </div>

      <div v-if="Object.keys(nodeSelectorValue).length === 0" class="empty-tip">
        暂无标签配置，请添加节点标签
      </div>

      <div v-else class="labels-list">
        <div v-for="(value, key) in nodeSelectorValue" :key="key" class="label-item">
          <div class="label-key">{{ key }}</div>
          <div class="label-eq">=</div>
          <ElInput
            v-model="nodeSelectorValue[key]"
            :readonly="!editing"
            placeholder="标签值"
            size="default"
          />
          <ElButton v-if="editing" type="danger" text circle @click="removeLabel(key)">
            <Trash2 :size="16" />
          </ElButton>
        </div>
      </div>
    </div>

    <!-- ==================== 节点亲和性模式 ==================== -->
    <div v-if="mode === 'nodeAffinity'" class="mode-content">
      <ElAlert type="info" :closable="false" show-icon>
        <template #title>使用节点亲和性规则，支持复杂的选择逻辑</template>
      </ElAlert>

      <ElTabs type="border-card">
        <!-- 必须满足 -->
        <ElTabPane>
          <template #label>
            <div class="tab-label">
              <ShieldCheck :size="14" />
              <span>必须满足</span>
              <ElBadge
                v-if="nodeAffinityRequiredValue.length > 0"
                :value="nodeAffinityRequiredValue.length"
              />
            </div>
          </template>

          <div class="action-bar">
            <ElButton
              v-if="editing"
              type="primary"
              size="small"
              @click="openLabelPicker('affinityRequired')"
            >
              <Database :size="14" />
              从节点标签选择
            </ElButton>
            <ElButton
              v-if="editing"
              type="success"
              size="small"
              @click="addAffinityRule('required')"
            >
              <Plus :size="14" />
              自定义规则
            </ElButton>
          </div>

          <div v-if="nodeAffinityRequiredValue.length === 0" class="empty-tip">暂无规则</div>

          <div v-else class="rules-list">
            <div v-for="(rule, index) in nodeAffinityRequiredValue" :key="index" class="rule-item">
              <ElInput
                v-model="rule.key"
                :readonly="!editing"
                placeholder="标签键"
                style="width: 200px"
              />
              <ElSelect v-model="rule.operator" :disabled="!editing" style="width: 150px">
                <ElOption label="In (包含)" value="In" />
                <ElOption label="NotIn (不包含)" value="NotIn" />
                <ElOption label="Exists (存在)" value="Exists" />
                <ElOption label="DoesNotExist (不存在)" value="DoesNotExist" />
              </ElSelect>
              <ElInput
                v-if="['In', 'NotIn'].includes(rule.operator)"
                v-model="rule.values"
                :readonly="!editing"
                placeholder="值（逗号分隔）"
                style="flex: 1"
              />
              <ElButton
                v-if="editing"
                type="danger"
                text
                circle
                @click="removeAffinityRule('required', index)"
              >
                <X :size="16" />
              </ElButton>
            </div>
          </div>
        </ElTabPane>

        <!-- 优先满足 -->
        <ElTabPane>
          <template #label>
            <div class="tab-label">
              <Star :size="14" />
              <span>优先满足</span>
              <ElBadge
                v-if="nodeAffinityPreferredValue.length > 0"
                :value="nodeAffinityPreferredValue.length"
              />
            </div>
          </template>

          <div class="action-bar">
            <ElButton
              v-if="editing"
              type="primary"
              size="small"
              @click="openLabelPicker('affinityPreferred')"
            >
              <Database :size="14" />
              从节点标签选择
            </ElButton>
            <ElButton
              v-if="editing"
              type="success"
              size="small"
              @click="addAffinityRule('preferred')"
            >
              <Plus :size="14" />
              自定义规则
            </ElButton>
          </div>

          <div v-if="nodeAffinityPreferredValue.length === 0" class="empty-tip">暂无规则</div>

          <div v-else class="rules-list">
            <div v-for="(rule, index) in nodeAffinityPreferredValue" :key="index" class="rule-item">
              <ElInputNumber
                v-model="rule.weight"
                :disabled="!editing"
                :min="1"
                :max="100"
                placeholder="权重"
                style="width: 100px"
              />
              <ElInput
                v-model="rule.key"
                :readonly="!editing"
                placeholder="标签键"
                style="width: 200px"
              />
              <ElSelect v-model="rule.operator" :disabled="!editing" style="width: 150px">
                <ElOption label="In" value="In" />
                <ElOption label="NotIn" value="NotIn" />
                <ElOption label="Exists" value="Exists" />
                <ElOption label="DoesNotExist" value="DoesNotExist" />
              </ElSelect>
              <ElInput
                v-if="['In', 'NotIn'].includes(rule.operator)"
                v-model="rule.values"
                :readonly="!editing"
                placeholder="值（逗号分隔）"
                style="flex: 1"
              />
              <ElButton
                v-if="editing"
                type="danger"
                text
                circle
                @click="removeAffinityRule('preferred', index)"
              >
                <X :size="16" />
              </ElButton>
            </div>
          </div>
        </ElTabPane>
      </ElTabs>
    </div>

    <!-- ==================== 节点标签选择对话框（先选节点，再选标签） ==================== -->
    <ElDialog
      v-model="labelPickerVisible"
      title="从节点标签选择"
      width="750px"
      :close-on-click-modal="false"
      destroy-on-close
      @open="handleLabelPickerOpen"
    >
      <div class="label-picker-content">
        <!-- 第一步：选择节点 -->
        <div class="picker-section">
          <div class="section-title">
            <span class="step-number">1</span>
            <span>选择节点</span>
          </div>
          <div class="node-selector-row">
            <ElSelect
              v-model="pickerSelectedNodeId"
              filterable
              remote
              reserve-keyword
              :remote-method="handlePickerNodeSearch"
              :loading="pickerNodeLoading"
              placeholder="搜索并选择节点..."
              style="flex: 1"
              @change="handlePickerNodeChange"
              @visible-change="handlePickerNodeDropdownChange"
            >
              <ElOption
                v-for="node in pickerNodeList"
                :key="node.id"
                :label="node.nodeName"
                :value="node.id"
              >
                <div class="node-option">
                  <span class="node-name">{{ node.nodeName }}</span>
                  <div class="node-meta">
                    <ElTag size="small" :type="node.nodeStatus === 'Ready' ? 'success' : 'danger'">
                      {{ node.nodeStatus }}
                    </ElTag>
                    <span class="node-ip">{{ node.nodeIp }}</span>
                  </div>
                </div>
              </ElOption>

              <template #footer>
                <div class="select-footer">
                  <span class="page-info">
                    已加载 {{ pickerNodeList.length }} / {{ pickerNodeTotal }} 个
                  </span>
                  <ElButton
                    v-if="pickerNodeList.length < pickerNodeTotal"
                    type="primary"
                    link
                    size="small"
                    :loading="pickerNodeLoading"
                    @click.stop="loadMorePickerNodes"
                  >
                    加载更多
                  </ElButton>
                </div>
              </template>
            </ElSelect>
          </div>

          <!-- 当前选中的节点信息 -->
          <div v-if="pickerSelectedNode" class="selected-node-info">
            <Monitor :size="16" />
            <span>{{ pickerSelectedNode.nodeName }}</span>
            <ElTag
              size="small"
              :type="pickerSelectedNode.nodeStatus === 'Ready' ? 'success' : 'danger'"
            >
              {{ pickerSelectedNode.nodeStatus }}
            </ElTag>
            <span class="node-ip">{{ pickerSelectedNode.nodeIp }}</span>
          </div>
        </div>

        <!-- 第二步：选择标签 -->
        <div class="picker-section">
          <div class="section-title">
            <span class="step-number">2</span>
            <span>选择标签</span>
            <span v-if="pickerSelectedNode" class="label-count">
              （共 {{ currentNodeLabels.length }} 个标签）
            </span>
          </div>

          <!-- 标签搜索 -->
          <div v-if="pickerSelectedNode" class="label-search-row">
            <ElInput
              v-model="labelFilterKeyword"
              placeholder="搜索标签..."
              clearable
              style="width: 250px"
            >
              <template #prefix>
                <Search :size="14" />
              </template>
            </ElInput>
          </div>

          <!-- 标签列表 -->
          <div v-if="!pickerSelectedNode" class="empty-tip small"> 请先选择一个节点 </div>
          <div v-else-if="pickerLabelLoading" v-loading="true" class="loading-tip"></div>
          <div v-else-if="filteredLabels.length === 0" class="empty-tip small">
            {{ labelFilterKeyword ? '没有匹配的标签' : '该节点没有标签' }}
          </div>
          <div v-else class="labels-grid">
            <div
              v-for="label in filteredLabels"
              :key="`${label.key}=${label.value}`"
              class="label-card"
              :class="{ selected: isLabelSelected(label) }"
              @click="toggleLabelSelection(label)"
            >
              <div class="label-checkbox">
                <ElCheckbox :model-value="isLabelSelected(label)" @click.stop />
              </div>
              <div class="label-info">
                <div class="label-key-text">{{ label.key }}</div>
                <div class="label-value-text">{{ label.value || '(空值)' }}</div>
              </div>
            </div>
          </div>
        </div>

        <!-- 已选标签汇总 -->
        <div v-if="selectedLabels.length > 0" class="selection-summary">
          <div class="summary-header">
            <Info :size="14" />
            <span>已选择 {{ selectedLabels.length }} 个标签</span>
            <ElButton type="danger" link size="small" @click="selectedLabels = []"> 清空 </ElButton>
          </div>
          <div class="selected-labels-list">
            <ElTag
              v-for="label in selectedLabels"
              :key="`${label.key}=${label.value}`"
              closable
              size="small"
              @close="removeLabelSelection(label)"
            >
              {{ label.key }}={{ label.value || '(空)' }}
            </ElTag>
          </div>
        </div>
      </div>

      <template #footer>
        <ElButton @click="labelPickerVisible = false">取消</ElButton>
        <ElButton
          type="primary"
          :disabled="selectedLabels.length === 0"
          @click="confirmLabelSelection"
        >
          确认添加 ({{ selectedLabels.length }})
        </ElButton>
      </template>
    </ElDialog>

    <!-- ==================== 自定义标签对话框 ==================== -->
    <ElDialog
      v-model="customLabelDialogVisible"
      title="添加自定义标签"
      width="500px"
      :close-on-click-modal="false"
    >
      <ElForm :model="customLabelForm" label-width="80px">
        <ElFormItem label="标签键" required>
          <ElInput v-model="customLabelForm.key" placeholder="例如: kubernetes.io/os" clearable />
        </ElFormItem>
        <ElFormItem label="标签值">
          <ElInput v-model="customLabelForm.value" placeholder="例如: linux" clearable />
        </ElFormItem>
        <ElFormItem v-if="customLabelTarget === 'affinityPreferred'" label="权重">
          <ElInputNumber v-model="customLabelForm.weight" :min="1" :max="100" style="width: 100%" />
        </ElFormItem>
      </ElForm>

      <template #footer>
        <ElButton @click="customLabelDialogVisible = false">取消</ElButton>
        <ElButton type="primary" :disabled="!customLabelForm.key" @click="confirmCustomLabel">
          确认添加
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { ref, computed } from 'vue'
  import { ElMessage } from 'element-plus'
  import {
    Server,
    Zap,
    Target,
    Filter,
    Magnet,
    Check,
    Plus,
    X,
    Trash2,
    ShieldCheck,
    Star,
    Search,
    Database,
    Info,
    Monitor
  } from 'lucide-vue-next'
  import {
    getNodeListApi,
    getNodeLabelsApi,
    type OnecProjectVersion,
    type ProjectCluster,
    type ProjectWorkspace,
    type ClusterNodeInfo
  } from '@/api'

  // ==================== Props ====================
  interface Props {
    editing: boolean
    version: OnecProjectVersion
    cluster: ProjectCluster | null
    workspace: ProjectWorkspace | null
  }

  const props = defineProps<Props>()

  // ==================== Model 定义 ====================
  const mode = defineModel<'auto' | 'nodeName' | 'nodeSelector' | 'nodeAffinity'>('mode', {
    default: 'auto'
  })
  const nodeNameValue = defineModel<string>('nodeName', { default: '' })
  const nodeSelectorValue = defineModel<Record<string, string>>('nodeSelector', {
    default: () => ({})
  })
  const nodeAffinityRequiredValue = defineModel<any[]>('nodeAffinityRequired', {
    default: () => []
  })
  const nodeAffinityPreferredValue = defineModel<any[]>('nodeAffinityPreferred', {
    default: () => []
  })

  // ==================== 调度模式配置 ====================
  const modes = [
    { value: 'auto', label: '自动调度', desc: 'Kubernetes 自动分配', icon: Zap },
    { value: 'nodeName', label: '指定节点', desc: '选择具体节点运行', icon: Target },
    { value: 'nodeSelector', label: '标签选择', desc: '通过标签筛选节点', icon: Filter },
    { value: 'nodeAffinity', label: '节点亲和性', desc: '使用复杂的亲和规则', icon: Magnet }
  ]

  // ==================== 指定节点模式 - 状态 ====================
  const nodeSelectLoading = ref(false)
  const nodeList = ref<ClusterNodeInfo[]>([])
  const nodeTotalCount = ref(0)
  const nodeCurrentPage = ref(1)
  const nodePageSize = ref(20)
  const nodeSearchKeyword = ref('')
  let nodeSearchTimer: ReturnType<typeof setTimeout> | null = null

  // ==================== 标签选择器对话框 - 状态 ====================
  const labelPickerVisible = ref(false)
  const labelPickerTarget = ref<'selector' | 'affinityRequired' | 'affinityPreferred'>('selector')

  // 对话框内的节点选择
  const pickerNodeLoading = ref(false)
  const pickerNodeList = ref<ClusterNodeInfo[]>([])
  const pickerNodeTotal = ref(0)
  const pickerNodePage = ref(1)
  const pickerNodePageSize = ref(20)
  const pickerNodeSearchKeyword = ref('')
  const pickerSelectedNodeId = ref<number | null>(null)
  const pickerSelectedNode = ref<ClusterNodeInfo | null>(null)
  let pickerNodeSearchTimer: ReturnType<typeof setTimeout> | null = null

  // 对话框内的标签
  const pickerLabelLoading = ref(false)
  const currentNodeLabels = ref<{ key: string; value: string }[]>([])
  const labelFilterKeyword = ref('')
  const selectedLabels = ref<{ key: string; value: string }[]>([])

  // ==================== 自定义标签对话框 - 状态 ====================
  const customLabelDialogVisible = ref(false)
  const customLabelTarget = ref<'selector' | 'affinityRequired' | 'affinityPreferred'>('selector')
  const customLabelForm = ref({
    key: '',
    value: '',
    weight: 50
  })

  // ==================== 计算属性 ====================

  /** 过滤后的标签列表 */
  const filteredLabels = computed(() => {
    if (!labelFilterKeyword.value) {
      return currentNodeLabels.value
    }
    const keyword = labelFilterKeyword.value.toLowerCase()
    return currentNodeLabels.value.filter(
      (label) =>
        label.key.toLowerCase().includes(keyword) || label.value.toLowerCase().includes(keyword)
    )
  })

  // ==================== 工具方法 ====================

  /** 获取 clusterUuid */
  const getClusterUuid = (): string | null => {
    return props.cluster?.clusterUuid || props.version.clusterUuid || null
  }

  // ==================== 指定节点模式 - 方法 ====================

  const handleNodeSelectVisibleChange = (visible: boolean) => {
    if (visible && nodeList.value.length === 0) {
      loadNodes()
    }
  }

  const handleNodeRemoteSearch = (query: string) => {
    nodeSearchKeyword.value = query
    if (nodeSearchTimer) clearTimeout(nodeSearchTimer)
    nodeSearchTimer = setTimeout(() => {
      nodeCurrentPage.value = 1
      loadNodes()
    }, 300)
  }

  const loadNodes = async () => {
    const clusterUuid = getClusterUuid()
    if (!clusterUuid) return

    nodeSelectLoading.value = true
    try {
      const response = await getNodeListApi({
        clusterUuid,
        page: nodeCurrentPage.value,
        pageSize: nodePageSize.value
      })

      let items = response.items || []

      // 前端搜索过滤
      if (nodeSearchKeyword.value) {
        const keyword = nodeSearchKeyword.value.toLowerCase()
        items = items.filter(
          (node) => node.nodeName.toLowerCase().includes(keyword) || node.nodeIp.includes(keyword)
        )
      }

      if (nodeCurrentPage.value === 1) {
        nodeList.value = items
      } else {
        nodeList.value = [...nodeList.value, ...items]
      }
      nodeTotalCount.value = response.total || 0
    } catch (error) {
      console.error('[HostScheduling] 加载节点失败:', error)
    } finally {
      nodeSelectLoading.value = false
    }
  }

  const loadMoreNodes = () => {
    if (nodeList.value.length >= nodeTotalCount.value) return
    nodeCurrentPage.value++
    loadNodes()
  }

  // ==================== 标签选择器对话框 - 方法 ====================

  /** 打开标签选择对话框 */
  const openLabelPicker = (target: 'selector' | 'affinityRequired' | 'affinityPreferred') => {
    labelPickerTarget.value = target
    labelPickerVisible.value = true
  }

  /** 对话框打开时初始化 */
  const handleLabelPickerOpen = () => {
    // 重置状态
    pickerNodeList.value = []
    pickerNodeTotal.value = 0
    pickerNodePage.value = 1
    pickerNodeSearchKeyword.value = ''
    pickerSelectedNodeId.value = null
    pickerSelectedNode.value = null
    currentNodeLabels.value = []
    labelFilterKeyword.value = ''
    selectedLabels.value = []

    // 加载节点列表
    loadPickerNodes()
  }

  /** 对话框内节点下拉框展开/收起 */
  const handlePickerNodeDropdownChange = (visible: boolean) => {
    if (visible && pickerNodeList.value.length === 0) {
      loadPickerNodes()
    }
  }

  /** 对话框内搜索节点 */
  const handlePickerNodeSearch = (query: string) => {
    pickerNodeSearchKeyword.value = query
    if (pickerNodeSearchTimer) clearTimeout(pickerNodeSearchTimer)
    pickerNodeSearchTimer = setTimeout(() => {
      pickerNodePage.value = 1
      loadPickerNodes()
    }, 300)
  }

  /** 加载对话框内的节点列表 */
  const loadPickerNodes = async () => {
    const clusterUuid = getClusterUuid()
    if (!clusterUuid) return

    pickerNodeLoading.value = true
    try {
      const response = await getNodeListApi({
        clusterUuid,
        page: pickerNodePage.value,
        pageSize: pickerNodePageSize.value
      })

      let items = response.items || []

      // 前端搜索过滤
      if (pickerNodeSearchKeyword.value) {
        const keyword = pickerNodeSearchKeyword.value.toLowerCase()
        items = items.filter(
          (node) => node.nodeName.toLowerCase().includes(keyword) || node.nodeIp.includes(keyword)
        )
      }

      if (pickerNodePage.value === 1) {
        pickerNodeList.value = items
      } else {
        pickerNodeList.value = [...pickerNodeList.value, ...items]
      }
      pickerNodeTotal.value = response.total || 0
    } catch (error) {
      console.error('[HostScheduling] 加载节点失败:', error)
    } finally {
      pickerNodeLoading.value = false
    }
  }

  /** 加载更多对话框内的节点 */
  const loadMorePickerNodes = () => {
    if (pickerNodeList.value.length >= pickerNodeTotal.value) return
    pickerNodePage.value++
    loadPickerNodes()
  }

  /** 选择节点后加载该节点的标签 */
  const handlePickerNodeChange = async (nodeId: number) => {
    // 找到选中的节点
    const node = pickerNodeList.value.find((n) => n.id === nodeId)
    pickerSelectedNode.value = node || null

    if (!node) {
      currentNodeLabels.value = []
      return
    }

    // 加载该节点的标签
    pickerLabelLoading.value = true
    labelFilterKeyword.value = ''

    try {
      const labels = await getNodeLabelsApi(node.id)
      currentNodeLabels.value = (labels || [])
        .filter((label) => !label.isDelete)
        .map((label) => ({
          key: label.key,
          value: label.value
        }))
    } catch (error) {
      console.error('[HostScheduling] 加载标签失败:', error)
      ElMessage.error('加载节点标签失败')
      currentNodeLabels.value = []
    } finally {
      pickerLabelLoading.value = false
    }
  }

  /** 检查标签是否已选中 */
  const isLabelSelected = (label: { key: string; value: string }): boolean => {
    return selectedLabels.value.some((l) => l.key === label.key && l.value === label.value)
  }

  /** 切换标签选中状态 */
  const toggleLabelSelection = (label: { key: string; value: string }) => {
    const index = selectedLabels.value.findIndex(
      (l) => l.key === label.key && l.value === label.value
    )
    if (index >= 0) {
      selectedLabels.value.splice(index, 1)
    } else {
      selectedLabels.value.push({ ...label })
    }
  }

  /** 移除已选标签 */
  const removeLabelSelection = (label: { key: string; value: string }) => {
    const index = selectedLabels.value.findIndex(
      (l) => l.key === label.key && l.value === label.value
    )
    if (index >= 0) {
      selectedLabels.value.splice(index, 1)
    }
  }

  /** 确认标签选择 */
  const confirmLabelSelection = () => {
    if (selectedLabels.value.length === 0) return

    switch (labelPickerTarget.value) {
      case 'selector':
        const newSelector = { ...nodeSelectorValue.value }
        selectedLabels.value.forEach((label) => {
          newSelector[label.key] = label.value
        })
        nodeSelectorValue.value = newSelector
        break

      case 'affinityRequired':
        const requiredRules = selectedLabels.value.map((label) => ({
          key: label.key,
          operator: 'In',
          values: label.value
        }))
        nodeAffinityRequiredValue.value = [...nodeAffinityRequiredValue.value, ...requiredRules]
        break

      case 'affinityPreferred':
        const preferredRules = selectedLabels.value.map((label) => ({
          key: label.key,
          operator: 'In',
          values: label.value,
          weight: 50
        }))
        nodeAffinityPreferredValue.value = [...nodeAffinityPreferredValue.value, ...preferredRules]
        break
    }

    ElMessage.success(`已添加 ${selectedLabels.value.length} 个标签`)
    labelPickerVisible.value = false
  }

  // ==================== 自定义标签 - 方法 ====================

  const openCustomLabelDialog = (target: 'selector' | 'affinityRequired' | 'affinityPreferred') => {
    customLabelTarget.value = target
    customLabelForm.value = { key: '', value: '', weight: 50 }
    customLabelDialogVisible.value = true
  }

  const confirmCustomLabel = () => {
    const { key, value, weight } = customLabelForm.value
    if (!key) {
      ElMessage.warning('请输入标签键')
      return
    }

    switch (customLabelTarget.value) {
      case 'selector':
        nodeSelectorValue.value = { ...nodeSelectorValue.value, [key]: value || '' }
        break
      case 'affinityRequired':
        nodeAffinityRequiredValue.value = [
          ...nodeAffinityRequiredValue.value,
          { key, operator: 'In', values: value || '' }
        ]
        break
      case 'affinityPreferred':
        nodeAffinityPreferredValue.value = [
          ...nodeAffinityPreferredValue.value,
          { key, operator: 'In', values: value || '', weight }
        ]
        break
    }

    ElMessage.success('标签添加成功')
    customLabelDialogVisible.value = false
  }

  // ==================== 通用方法 ====================

  const handleModeChange = (value: any) => {
    mode.value = value
    if (value !== 'nodeName') nodeNameValue.value = ''
    if (value !== 'nodeSelector') nodeSelectorValue.value = {}
    if (value !== 'nodeAffinity') {
      nodeAffinityRequiredValue.value = []
      nodeAffinityPreferredValue.value = []
    }
  }

  const removeLabel = (key: string) => {
    const newValue = { ...nodeSelectorValue.value }
    delete newValue[key]
    nodeSelectorValue.value = newValue
  }

  const addAffinityRule = (type: 'required' | 'preferred') => {
    const rule: any = { key: '', operator: 'In', values: '' }
    if (type === 'preferred') rule.weight = 50

    if (type === 'required') {
      nodeAffinityRequiredValue.value = [...nodeAffinityRequiredValue.value, rule]
    } else {
      nodeAffinityPreferredValue.value = [...nodeAffinityPreferredValue.value, rule]
    }
  }

  const removeAffinityRule = (type: 'required' | 'preferred', index: number) => {
    if (type === 'required') {
      nodeAffinityRequiredValue.value = nodeAffinityRequiredValue.value.filter(
        (_, i) => i !== index
      )
    } else {
      nodeAffinityPreferredValue.value = nodeAffinityPreferredValue.value.filter(
        (_, i) => i !== index
      )
    }
  }
</script>

<style lang="scss" scoped>
  .host-scheduling-config {
    padding: 20px;

    .section-header {
      display: flex;
      align-items: center;
      gap: 10px;
      margin-bottom: 20px;

      h4 {
        margin: 0;
        font-size: 16px;
        font-weight: 600;
        color: #303133;
      }
    }

    .mode-selector {
      display: grid;
      grid-template-columns: repeat(2, 1fr);
      gap: 12px;
      margin-bottom: 20px;

      .mode-card {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 16px;
        background: #f9fafb;
        border: 2px solid #e4e7ed;
        border-radius: 8px;
        cursor: pointer;
        transition: all 0.2s;

        &:hover:not(.disabled) {
          border-color: #409eff;
          background: #ecf5ff;
        }

        &.active {
          border-color: #409eff;
          background: #e6f4ff;
        }

        &.disabled {
          cursor: not-allowed;
          opacity: 0.6;
        }

        .mode-info {
          flex: 1;

          .mode-name {
            font-size: 14px;
            font-weight: 600;
            color: #303133;
            margin-bottom: 4px;
          }

          .mode-desc {
            font-size: 12px;
            color: #909399;
          }
        }

        .check-icon {
          color: #409eff;
        }
      }
    }

    .mode-content {
      padding: 16px;
      background: #fafbfc;
      border: 1px solid #e4e7ed;
      border-radius: 8px;

      :deep(.el-alert) {
        margin-bottom: 16px;
      }

      .full-width {
        width: 100%;
        margin-top: 12px;
      }

      .node-option {
        display: flex;
        justify-content: space-between;
        align-items: center;
        width: 100%;

        .node-name {
          font-weight: 600;
          color: #303133;
        }

        .node-meta {
          display: flex;
          align-items: center;
          gap: 8px;

          .node-ip {
            font-size: 12px;
            color: #909399;
            font-family: var(--ax-font-mono);
          }
        }
      }

      .select-footer {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 8px 12px;
        border-top: 1px solid #e4e7ed;
        background: #f5f7fa;

        .page-info {
          font-size: 12px;
          color: #909399;
        }
      }

      .action-bar {
        display: flex;
        gap: 8px;
        margin-bottom: 12px;
      }

      .labels-list,
      .rules-list {
        display: flex;
        flex-direction: column;
        gap: 8px;

        .label-item,
        .rule-item {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 12px;
          background: white;
          border: 1px solid #e4e7ed;
          border-radius: 6px;

          .label-key {
            min-width: 180px;
            max-width: 300px;
            padding: 8px 12px;
            background: #f5f7fa;
            border: 1px solid #dcdfe6;
            border-radius: 4px;
            font-size: 13px;
            font-weight: 600;
            font-family: var(--ax-font-mono);
            word-break: break-all;
          }

          .label-eq {
            color: #909399;
            font-weight: 600;
          }
        }
      }

      .tab-label {
        display: flex;
        align-items: center;
        gap: 6px;
      }
    }

    .empty-tip {
      padding: 40px 20px;
      text-align: center;
      font-size: 13px;
      color: #909399;
      background: white;
      border: 1px dashed #e4e7ed;
      border-radius: 6px;

      &.small {
        padding: 30px 16px;
      }
    }

    .loading-tip {
      height: 150px;
    }
  }

  // ==================== 标签选择对话框样式 ====================
  .label-picker-content {
    .picker-section {
      margin-bottom: 20px;

      .section-title {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 12px;
        font-size: 14px;
        font-weight: 600;
        color: #303133;

        .step-number {
          display: inline-flex;
          align-items: center;
          justify-content: center;
          width: 22px;
          height: 22px;
          background: #409eff;
          color: white;
          border-radius: 50%;
          font-size: 12px;
        }

        .label-count {
          font-weight: normal;
          color: #909399;
          font-size: 13px;
        }
      }

      .node-selector-row {
        display: flex;
        gap: 8px;
      }

      .selected-node-info {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-top: 12px;
        padding: 10px 14px;
        background: #f0f9ff;
        border: 1px solid #b3d8ff;
        border-radius: 6px;
        font-size: 13px;
        color: #409eff;

        .node-ip {
          color: #909399;
          font-family: var(--ax-font-mono);
          font-size: 12px;
        }
      }

      .label-search-row {
        margin-bottom: 12px;
      }

      .labels-grid {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 10px;
        max-height: 280px;
        overflow-y: auto;
        padding: 4px;

        .label-card {
          display: flex;
          align-items: flex-start;
          gap: 10px;
          padding: 12px;
          background: #fafbfc;
          border: 2px solid #e4e7ed;
          border-radius: 6px;
          cursor: pointer;
          transition: all 0.2s;

          &:hover {
            border-color: #c0c4cc;
            background: #f5f7fa;
          }

          &.selected {
            border-color: #409eff;
            background: #ecf5ff;
          }

          .label-checkbox {
            padding-top: 2px;
          }

          .label-info {
            flex: 1;
            min-width: 0;

            .label-key-text {
              font-size: 13px;
              font-weight: 600;
              color: #303133;
              word-break: break-all;
              margin-bottom: 4px;
            }

            .label-value-text {
              font-size: 12px;
              color: #606266;
              word-break: break-all;
              font-family: var(--ax-font-mono);
            }
          }
        }
      }
    }

    .selection-summary {
      padding: 12px;
      background: #e6f7ff;
      border: 1px solid #91d5ff;
      border-radius: 6px;

      .summary-header {
        display: flex;
        align-items: center;
        gap: 6px;
        margin-bottom: 8px;
        font-size: 13px;
        color: #1890ff;
      }

      .selected-labels-list {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
      }
    }

    .node-option {
      display: flex;
      justify-content: space-between;
      align-items: center;
      width: 100%;

      .node-name {
        font-weight: 600;
        color: #303133;
      }

      .node-meta {
        display: flex;
        align-items: center;
        gap: 8px;

        .node-ip {
          font-size: 12px;
          color: #909399;
          font-family: var(--ax-font-mono);
        }
      }
    }

    .select-footer {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 8px 12px;
      border-top: 1px solid #e4e7ed;
      background: #f5f7fa;

      .page-info {
        font-size: 12px;
        color: #909399;
      }
    }
  }
</style>
