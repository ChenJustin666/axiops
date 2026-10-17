<template>
  <div class="toleration-config">
    <div class="section-header">
      <Shield :size="18" />
      <h4>污点容忍配置</h4>
    </div>

    <ElAlert type="info" :closable="false" show-icon>
      <template #title>配置容忍规则，允许 Pod 调度到带有特定污点的节点</template>
    </ElAlert>

    <div class="action-bar">
      <ElButton v-if="editing" type="primary" size="small" @click="openTaintPicker">
        <Database :size="14" />
        从节点污点选择
      </ElButton>
      <ElButton v-if="editing" type="success" size="small" @click="addToleration">
        <Plus :size="14" />
        自定义添加
      </ElButton>
    </div>

    <div v-if="tolerationsValue.length === 0" class="empty-tip">暂无容忍配置</div>

    <div v-else class="tolerations-list">
      <div v-for="(toleration, index) in tolerationsValue" :key="index" class="toleration-item">
        <div class="item-header">
          <span class="item-number">#{{ index + 1 }}</span>
          <ElButton
            v-if="editing"
            type="danger"
            text
            circle
            size="small"
            @click="removeToleration(index)"
          >
            <Trash2 :size="16" />
          </ElButton>
        </div>

        <div class="item-content">
          <div class="form-row">
            <label>污点键 (Key)</label>
            <ElInput
              v-model="toleration.key"
              :readonly="!editing"
              placeholder="例如: node.kubernetes.io/disk-pressure"
            />
          </div>

          <div class="form-row">
            <label>操作符 (Operator)</label>
            <ElSelect v-model="toleration.operator" :disabled="!editing">
              <ElOption label="Equal（等于：键和值都匹配）" value="Equal" />
              <ElOption label="Exists（存在：只匹配键）" value="Exists" />
            </ElSelect>
          </div>

          <div v-if="toleration.operator === 'Equal'" class="form-row">
            <label>污点值 (Value)</label>
            <ElInput v-model="toleration.value" :readonly="!editing" placeholder="污点值" />
          </div>

          <div class="form-row">
            <label>效果 (Effect)</label>
            <ElSelect v-model="toleration.effect" :disabled="!editing">
              <ElOption label="NoSchedule（不调度新 Pod）" value="NoSchedule" />
              <ElOption label="PreferNoSchedule（尽量不调度）" value="PreferNoSchedule" />
              <ElOption label="NoExecute（驱逐已有 Pod）" value="NoExecute" />
            </ElSelect>
          </div>

          <div v-if="toleration.effect === 'NoExecute'" class="form-row">
            <label>容忍时间 (秒)</label>
            <ElInputNumber
              v-model="toleration.tolerationSeconds"
              :disabled="!editing"
              :min="0"
              placeholder="永久容忍（留空）"
              style="width: 100%"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- 常用配置模板 -->
    <ElCard v-if="editing" class="templates-card">
      <template #header>
        <div class="card-header">
          <Zap :size="14" />
          <span>常用模板</span>
        </div>
      </template>
      <div class="templates-grid">
        <div
          v-for="template in templates"
          :key="template.name"
          class="template-item"
          @click="applyTemplate(template)"
        >
          <div class="template-icon">
            <component :is="template.icon" :size="20" />
          </div>
          <div class="template-info">
            <div class="template-name">{{ template.name }}</div>
            <div class="template-desc">{{ template.desc }}</div>
          </div>
        </div>
      </div>
    </ElCard>

    <!-- ==================== 从节点污点选择对话框 ==================== -->
    <ElDialog
      v-model="taintPickerVisible"
      title="从节点污点选择"
      width="750px"
      :close-on-click-modal="false"
      destroy-on-close
      @open="handleTaintPickerOpen"
    >
      <div class="taint-picker-content">
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

        <!-- 第二步：选择污点 -->
        <div class="picker-section">
          <div class="section-title">
            <span class="step-number">2</span>
            <span>选择污点</span>
            <span v-if="pickerSelectedNode" class="taint-count">
              （共 {{ currentNodeTaints.length }} 个污点）
            </span>
          </div>

          <!-- 污点搜索 -->
          <div v-if="pickerSelectedNode" class="taint-search-row">
            <ElInput
              v-model="taintFilterKeyword"
              placeholder="搜索污点..."
              clearable
              style="width: 250px"
            >
              <template #prefix>
                <Search :size="14" />
              </template>
            </ElInput>
          </div>

          <!-- 污点列表 -->
          <div v-if="!pickerSelectedNode" class="empty-tip small"> 请先选择一个节点 </div>
          <div v-else-if="pickerTaintLoading" v-loading="true" class="loading-tip"></div>
          <div v-else-if="filteredTaints.length === 0" class="empty-tip small">
            {{ taintFilterKeyword ? '没有匹配的污点' : '该节点没有污点' }}
          </div>
          <div v-else class="taints-list">
            <div
              v-for="taint in filteredTaints"
              :key="`${taint.key}-${taint.effect}`"
              class="taint-card"
              :class="{ selected: isTaintSelected(taint) }"
              @click="toggleTaintSelection(taint)"
            >
              <div class="taint-checkbox">
                <ElCheckbox :model-value="isTaintSelected(taint)" @click.stop />
              </div>
              <div class="taint-info">
                <div class="taint-key-value">
                  <span class="taint-key">{{ taint.key }}</span>
                  <span v-if="taint.value" class="taint-eq">=</span>
                  <span v-if="taint.value" class="taint-value">{{ taint.value }}</span>
                </div>
                <div class="taint-effect">
                  <ElTag :type="getEffectTagType(taint.effect)" size="small">
                    {{ taint.effect }}
                  </ElTag>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 已选污点汇总 -->
        <div v-if="selectedTaints.length > 0" class="selection-summary">
          <div class="summary-header">
            <Info :size="14" />
            <span>已选择 {{ selectedTaints.length }} 个污点，将生成对应的容忍配置</span>
            <ElButton type="danger" link size="small" @click="selectedTaints = []"> 清空 </ElButton>
          </div>
          <div class="selected-taints-list">
            <ElTag
              v-for="taint in selectedTaints"
              :key="`${taint.key}-${taint.effect}`"
              closable
              size="small"
              :type="getEffectTagType(taint.effect)"
              @close="removeTaintSelection(taint)"
            >
              {{ taint.key }}{{ taint.value ? `=${taint.value}` : '' }}:{{ taint.effect }}
            </ElTag>
          </div>
        </div>
      </div>

      <template #footer>
        <ElButton @click="taintPickerVisible = false">取消</ElButton>
        <ElButton
          type="primary"
          :disabled="selectedTaints.length === 0"
          @click="confirmTaintSelection"
        >
          生成容忍配置 ({{ selectedTaints.length }})
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { ref, computed } from 'vue'
  import { ElMessage } from 'element-plus'
  import {
    Shield,
    Plus,
    Trash2,
    Zap,
    Server,
    Cpu,
    HardDrive,
    Database,
    Search,
    Info,
    Monitor
  } from 'lucide-vue-next'
  import { getNodeListApi, getNodeTaintsApi, type ClusterNodeInfo, type NodeTaint } from '@/api'

  // ==================== Props ====================
  interface Props {
    editing: boolean
    clusterUuid?: string
    cluster?: { clusterUuid?: string } | null
    version?: { clusterUuid?: string } | null
  }

  const props = defineProps<Props>()

  // ==================== Model ====================
  const tolerationsValue = defineModel<any[]>('tolerations', { default: () => [] })

  // ==================== 工具方法 ====================

  /** 获取 clusterUuid（支持多种来源） */
  const getClusterUuid = (): string | null => {
    return props.clusterUuid || props.cluster?.clusterUuid || props.version?.clusterUuid || null
  }

  // ==================== 污点选择对话框状态 ====================
  const taintPickerVisible = ref(false)

  // 节点选择
  const pickerNodeLoading = ref(false)
  const pickerNodeList = ref<ClusterNodeInfo[]>([])
  const pickerNodeTotal = ref(0)
  const pickerNodePage = ref(1)
  const pickerNodePageSize = ref(20)
  const pickerNodeSearchKeyword = ref('')
  const pickerSelectedNodeId = ref<number | null>(null)
  const pickerSelectedNode = ref<ClusterNodeInfo | null>(null)
  let pickerNodeSearchTimer: ReturnType<typeof setTimeout> | null = null

  // 污点选择
  const pickerTaintLoading = ref(false)
  const currentNodeTaints = ref<NodeTaint[]>([])
  const taintFilterKeyword = ref('')
  const selectedTaints = ref<NodeTaint[]>([])

  // ==================== 模板配置 ====================
  const templates = [
    {
      name: 'Master 节点',
      desc: '允许调度到 Master 节点',
      icon: Server,
      config: {
        key: 'node-role.kubernetes.io/master',
        operator: 'Exists',
        value: '',
        effect: 'NoSchedule'
      }
    },
    {
      name: 'Control Plane',
      desc: '允许调度到控制平面节点',
      icon: Server,
      config: {
        key: 'node-role.kubernetes.io/control-plane',
        operator: 'Exists',
        value: '',
        effect: 'NoSchedule'
      }
    },
    {
      name: 'GPU 节点',
      desc: '允许调度到 GPU 节点',
      icon: Cpu,
      config: {
        key: 'nvidia.com/gpu',
        operator: 'Exists',
        value: '',
        effect: 'NoSchedule'
      }
    },
    {
      name: '磁盘压力',
      desc: '容忍磁盘压力污点',
      icon: HardDrive,
      config: {
        key: 'node.kubernetes.io/disk-pressure',
        operator: 'Exists',
        value: '',
        effect: 'NoSchedule'
      }
    },
    {
      name: '内存压力',
      desc: '容忍内存压力污点',
      icon: HardDrive,
      config: {
        key: 'node.kubernetes.io/memory-pressure',
        operator: 'Exists',
        value: '',
        effect: 'NoSchedule'
      }
    },
    {
      name: '不可调度',
      desc: '容忍节点不可调度状态',
      icon: Server,
      config: {
        key: 'node.kubernetes.io/unschedulable',
        operator: 'Exists',
        value: '',
        effect: 'NoSchedule'
      }
    }
  ]

  // ==================== 计算属性 ====================

  /** 过滤后的污点列表 */
  const filteredTaints = computed(() => {
    if (!taintFilterKeyword.value) {
      return currentNodeTaints.value
    }
    const keyword = taintFilterKeyword.value.toLowerCase()
    return currentNodeTaints.value.filter(
      (taint) =>
        taint.key.toLowerCase().includes(keyword) ||
        (taint.value && taint.value.toLowerCase().includes(keyword)) ||
        taint.effect.toLowerCase().includes(keyword)
    )
  })

  // ==================== 基础方法 ====================

  const addToleration = () => {
    tolerationsValue.value = [
      ...tolerationsValue.value,
      {
        key: '',
        operator: 'Equal',
        value: '',
        effect: 'NoSchedule'
      }
    ]
  }

  const removeToleration = (index: number) => {
    tolerationsValue.value = tolerationsValue.value.filter((_, i) => i !== index)
  }

  const applyTemplate = (template: any) => {
    tolerationsValue.value = [...tolerationsValue.value, { ...template.config }]
    ElMessage.success(`已添加「${template.name}」容忍配置`)
  }

  const getEffectTagType = (effect: string): string => {
    switch (effect) {
      case 'NoSchedule':
        return 'danger'
      case 'PreferNoSchedule':
        return 'warning'
      case 'NoExecute':
        return ''
      default:
        return 'info'
    }
  }

  // ==================== 污点选择对话框方法 ====================

  /** 打开污点选择对话框 */
  const openTaintPicker = () => {
    taintPickerVisible.value = true
  }

  /** 对话框打开时初始化 */
  const handleTaintPickerOpen = () => {
    // 重置状态
    pickerNodeList.value = []
    pickerNodeTotal.value = 0
    pickerNodePage.value = 1
    pickerNodeSearchKeyword.value = ''
    pickerSelectedNodeId.value = null
    pickerSelectedNode.value = null
    currentNodeTaints.value = []
    taintFilterKeyword.value = ''
    selectedTaints.value = []

    // 加载节点列表
    loadPickerNodes()
  }

  /** 节点下拉框展开/收起（数据已在对话框打开时加载，这里不需要重复加载） */
  const handlePickerNodeDropdownChange = (_visible: boolean) => {
    // 数据在 handleTaintPickerOpen 中已经加载，这里无需处理
  }

  /** 搜索节点 */
  const handlePickerNodeSearch = (query: string) => {
    pickerNodeSearchKeyword.value = query
    if (pickerNodeSearchTimer) clearTimeout(pickerNodeSearchTimer)
    pickerNodeSearchTimer = setTimeout(() => {
      pickerNodePage.value = 1
      loadPickerNodes()
    }, 300)
  }

  /** 加载节点列表 */
  const loadPickerNodes = async () => {
    const clusterUuid = getClusterUuid()
    if (!clusterUuid) {
      console.warn('[TolerationConfig] 无法获取 clusterUuid，请检查 props')
      ElMessage.warning('无法获取集群信息，请检查配置')
      return
    }

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
      console.error('[TolerationConfig] 加载节点失败:', error)
    } finally {
      pickerNodeLoading.value = false
    }
  }

  /** 加载更多节点 */
  const loadMorePickerNodes = () => {
    if (pickerNodeList.value.length >= pickerNodeTotal.value) return
    pickerNodePage.value++
    loadPickerNodes()
  }

  /** 选择节点后加载该节点的污点 */
  const handlePickerNodeChange = async (nodeId: number) => {
    const node = pickerNodeList.value.find((n) => n.id === nodeId)
    pickerSelectedNode.value = node || null

    if (!node) {
      currentNodeTaints.value = []
      return
    }

    // 加载该节点的污点
    pickerTaintLoading.value = true
    taintFilterKeyword.value = ''

    try {
      const taints = await getNodeTaintsApi(node.id)
      currentNodeTaints.value = (taints || []).filter((taint) => !taint.isDelete)
    } catch (error) {
      console.error('[TolerationConfig] 加载污点失败:', error)
      ElMessage.error('加载节点污点失败')
      currentNodeTaints.value = []
    } finally {
      pickerTaintLoading.value = false
    }
  }

  /** 检查污点是否已选中 */
  const isTaintSelected = (taint: NodeTaint): boolean => {
    return selectedTaints.value.some((t) => t.key === taint.key && t.effect === taint.effect)
  }

  /** 切换污点选中状态 */
  const toggleTaintSelection = (taint: NodeTaint) => {
    const index = selectedTaints.value.findIndex(
      (t) => t.key === taint.key && t.effect === taint.effect
    )
    if (index >= 0) {
      selectedTaints.value.splice(index, 1)
    } else {
      selectedTaints.value.push({ ...taint })
    }
  }

  /** 移除已选污点 */
  const removeTaintSelection = (taint: NodeTaint) => {
    const index = selectedTaints.value.findIndex(
      (t) => t.key === taint.key && t.effect === taint.effect
    )
    if (index >= 0) {
      selectedTaints.value.splice(index, 1)
    }
  }

  /** 确认污点选择，生成容忍配置 */
  const confirmTaintSelection = () => {
    if (selectedTaints.value.length === 0) return

    // 将选中的污点转换为容忍配置
    const newTolerations = selectedTaints.value.map((taint) => ({
      key: taint.key,
      operator: taint.value ? 'Equal' : 'Exists',
      value: taint.value || '',
      effect: taint.effect
    }))

    tolerationsValue.value = [...tolerationsValue.value, ...newTolerations]

    ElMessage.success(`已生成 ${selectedTaints.value.length} 条容忍配置`)
    taintPickerVisible.value = false
  }
</script>

<style lang="scss" scoped>
  .toleration-config {
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

    :deep(.el-alert) {
      margin-bottom: 16px;
    }

    .action-bar {
      display: flex;
      gap: 8px;
      margin-bottom: 16px;
    }

    .tolerations-list {
      display: flex;
      flex-direction: column;
      gap: 12px;
      margin-bottom: 20px;

      .toleration-item {
        padding: 16px;
        background: #fafbfc;
        border: 1px solid #e4e7ed;
        border-radius: 8px;

        .item-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 12px;
          padding-bottom: 8px;
          border-bottom: 1px solid #e4e7ed;

          .item-number {
            font-size: 13px;
            font-weight: 600;
            color: #606266;
          }
        }

        .item-content {
          display: flex;
          flex-direction: column;
          gap: 12px;

          .form-row {
            display: flex;
            flex-direction: column;
            gap: 6px;

            label {
              font-size: 13px;
              font-weight: 500;
              color: #606266;
            }
          }
        }
      }
    }

    .templates-card {
      margin-top: 20px;

      .card-header {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 14px;
        font-weight: 600;
      }

      .templates-grid {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 10px;

        .template-item {
          display: flex;
          align-items: center;
          gap: 10px;
          padding: 12px;
          background: #f5f7fa;
          border: 1px solid #e4e7ed;
          border-radius: 6px;
          cursor: pointer;
          transition: all 0.2s;

          &:hover {
            border-color: #409eff;
            background: #ecf5ff;
          }

          .template-icon {
            width: 36px;
            height: 36px;
            display: flex;
            align-items: center;
            justify-content: center;
            background: white;
            border-radius: 6px;
            color: #409eff;
          }

          .template-info {
            flex: 1;

            .template-name {
              font-size: 13px;
              font-weight: 600;
              color: #303133;
              margin-bottom: 2px;
            }

            .template-desc {
              font-size: 11px;
              color: #909399;
            }
          }
        }
      }
    }

    .empty-tip {
      padding: 40px 20px;
      text-align: center;
      font-size: 13px;
      color: #909399;
      background: #fafbfc;
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

  // ==================== 污点选择对话框样式 ====================
  .taint-picker-content {
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

        .taint-count {
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

      .taint-search-row {
        margin-bottom: 12px;
      }

      .taints-list {
        display: flex;
        flex-direction: column;
        gap: 10px;
        max-height: 280px;
        overflow-y: auto;
        padding: 4px;

        .taint-card {
          display: flex;
          align-items: center;
          gap: 12px;
          padding: 14px 16px;
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

          .taint-checkbox {
            flex-shrink: 0;
          }

          .taint-info {
            flex: 1;
            display: flex;
            justify-content: space-between;
            align-items: center;
            min-width: 0;

            .taint-key-value {
              display: flex;
              align-items: center;
              gap: 4px;
              flex-wrap: wrap;

              .taint-key {
                font-size: 14px;
                font-weight: 600;
                color: #303133;
                font-family: var(--ax-font-mono);
                word-break: break-all;
              }

              .taint-eq {
                color: #909399;
              }

              .taint-value {
                font-size: 13px;
                color: #606266;
                font-family: var(--ax-font-mono);
                word-break: break-all;
              }
            }

            .taint-effect {
              flex-shrink: 0;
              margin-left: 12px;
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

      .selected-taints-list {
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
