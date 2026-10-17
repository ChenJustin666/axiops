<template>
  <div class="scheduling-management">
    <!-- 顶部操作栏 -->
    <div class="toolbar">
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
          <ElButton size="small" @click="handleCancel">取消</ElButton>
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

    <!-- 主内容区 -->
    <div v-else class="content-wrapper">
      <!-- 标签导航栏 -->
      <div class="tab-nav-wrapper">
        <div class="tab-nav">
          <div
            v-for="cat in categories"
            :key="cat.key"
            class="tab-item"
            :class="{
              active: activeCategory === cat.key,
              'has-config': getCategoryBadge(cat.key) > 0
            }"
            @click="activeCategory = cat.key"
          >
            <component :is="cat.icon" :size="14" />
            <span>{{ cat.label }}</span>
            <span v-if="getCategoryBadge(cat.key) > 0" class="config-dot"></span>
          </div>
        </div>
      </div>

      <!-- 编辑状态提示 -->
      <ElAlert
        v-if="editing && hasUnsavedChanges()"
        type="warning"
        :closable="false"
        show-icon
        class="unsaved-tip"
      >
        <template #title> 您有未保存的修改，请在完成所有配置后点击「保存」按钮统一提交 </template>
      </ElAlert>

      <!-- 内容区域 -->
      <div class="config-content">
        <!-- 主机调度 -->
        <div v-show="activeCategory === 'host'" class="config-section">
          <HostSchedulingConfig
            :editing="editing"
            :version="version"
            :cluster="cluster"
            :workspace="workspace"
            v-model:mode="hostSchedulingMode"
            v-model:node-name="nodeName"
            v-model:node-selector="nodeSelector"
            v-model:node-affinity-required="nodeAffinityRequired"
            v-model:node-affinity-preferred="nodeAffinityPreferred"
          />
        </div>

        <!-- 容忍配置 -->
        <div v-show="activeCategory === 'toleration'" class="config-section">
          <TolerationConfig
            :editing="editing"
            :version="version"
            :cluster="cluster"
            v-model:tolerations="tolerations"
          />
        </div>

        <!-- 服务亲和性 -->
        <div v-show="activeCategory === 'service-affinity'" class="config-section">
          <ServiceAffinityConfig
            :editing="editing"
            :version="version"
            :cluster="cluster"
            :workspace="workspace"
            v-model:pod-affinity="podAffinity"
            v-model:pod-anti-affinity="podAntiAffinity"
          />
        </div>

        <!-- 拓扑分布 -->
        <div v-show="activeCategory === 'topology'" class="config-section">
          <TopologySpreadConfig
            :editing="editing"
            v-model:constraints="topologySpreadConstraints"
          />
        </div>

        <!-- 高级配置 -->
        <div v-show="activeCategory === 'advanced'" class="config-section">
          <AdvancedConfig
            :editing="editing"
            v-model:scheduler-name="schedulerName"
            v-model:priority-class="priorityClassName"
            v-model:priority="priority"
            v-model:runtime-class="runtimeClassName"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, markRaw } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { Server, Shield, Users, Layers, Settings, Edit, RefreshCw, Save } from 'lucide-vue-next'
  import {
    getSchedulingConfigApi,
    updateSchedulingConfigApi,
    type OnecProjectVersion,
    type ProjectCluster,
    type ProjectWorkspace,
    type SchedulingConfig,
    type UpdateSchedulingConfigRequest
  } from '@/api'

  import HostSchedulingConfig from './scheduling/HostSchedulingConfig.vue'
  import TolerationConfig from './scheduling/TolerationConfig.vue'
  import ServiceAffinityConfig from './scheduling/ServiceAffinityConfig.vue'
  import TopologySpreadConfig from './scheduling/TopologySpreadConfig.vue'
  import AdvancedConfig from './scheduling/AdvancedConfig.vue'

  interface Props {
    version: OnecProjectVersion
    cluster: ProjectCluster | null
    workspace: ProjectWorkspace | null
    refreshTrigger?: number
  }

  const props = defineProps<Props>()
  const emit = defineEmits<{ success: [] }>()

  // ==================== 状态变量 ====================
  const loading = ref(false)
  const saving = ref(false)
  const editing = ref(false)
  const activeCategory = ref('host')

  // 分类配置
  const categories = [
    { key: 'host', label: '主机调度', icon: markRaw(Server) },
    { key: 'toleration', label: '容忍配置', icon: markRaw(Shield) },
    { key: 'service-affinity', label: 'Pod 亲和性', icon: markRaw(Users) },
    { key: 'topology', label: '拓扑分布', icon: markRaw(Layers) },
    { key: 'advanced', label: '高级配置', icon: markRaw(Settings) }
  ]

  // ==================== 主机调度配置 ====================
  const hostSchedulingMode = ref<'auto' | 'nodeName' | 'nodeSelector' | 'nodeAffinity'>('auto')
  const nodeName = ref('')
  const nodeSelector = ref<Record<string, string>>({})
  const nodeAffinityRequired = ref<any[]>([])
  const nodeAffinityPreferred = ref<any[]>([])

  // ==================== 容忍配置 ====================
  const tolerations = ref<any[]>([])

  // ==================== 服务亲和性配置 ====================
  const podAffinity = ref<any[]>([])
  const podAntiAffinity = ref<any[]>([])

  // ==================== 拓扑分布配置 ====================
  const topologySpreadConstraints = ref<any[]>([])

  // ==================== 高级配置 ====================
  const schedulerName = ref('')
  const priorityClassName = ref('')
  const priority = ref(0)
  const runtimeClassName = ref('')

  // ==================== 原始配置（用于对比是否有修改） ====================
  const originalConfig = ref<SchedulingConfig | null>(null)

  // ==================== 工具方法 ====================

  /**
   * 清空主机调度相关的配置数据
   * 根据当前选择的模式，清空其他模式的配置
   */
  const clearHostSchedulingDataByMode = (currentMode: string) => {
    switch (currentMode) {
      case 'auto':
        nodeName.value = ''
        nodeSelector.value = {}
        nodeAffinityRequired.value = []
        nodeAffinityPreferred.value = []
        break
      case 'nodeName':
        nodeSelector.value = {}
        nodeAffinityRequired.value = []
        nodeAffinityPreferred.value = []
        break
      case 'nodeSelector':
        nodeName.value = ''
        nodeAffinityRequired.value = []
        nodeAffinityPreferred.value = []
        break
      case 'nodeAffinity':
        nodeName.value = ''
        nodeSelector.value = {}
        break
    }
  }

  /**
   * 监听主机调度模式变化
   * 当用户在编辑状态下切换模式时，自动清空其他模式的配置数据
   */
  watch(hostSchedulingMode, (newMode, oldMode) => {
    if (editing.value && newMode !== oldMode) {
      clearHostSchedulingDataByMode(newMode)
    }
  })

  /**
   * 获取分类的徽章数量
   */
  const getCategoryBadge = (key: string): number => {
    switch (key) {
      case 'host':
        if (hostSchedulingMode.value === 'nodeName' && nodeName.value) return 1
        if (
          hostSchedulingMode.value === 'nodeSelector' &&
          Object.keys(nodeSelector.value).length > 0
        ) {
          return Object.keys(nodeSelector.value).length
        }
        if (hostSchedulingMode.value === 'nodeAffinity') {
          return nodeAffinityRequired.value.length + nodeAffinityPreferred.value.length
        }
        return 0
      case 'toleration':
        return tolerations.value.length
      case 'service-affinity':
        return podAffinity.value.length + podAntiAffinity.value.length
      case 'topology':
        return topologySpreadConstraints.value.length
      case 'advanced': {
        let count = 0
        if (schedulerName.value) count++
        if (priorityClassName.value) count++
        if (priority.value > 0) count++
        if (runtimeClassName.value) count++
        return count
      }
      default:
        return 0
    }
  }

  /**
   * 获取当前配置的快照（用于对比）
   */
  const getCurrentConfigSnapshot = () => {
    return {
      hostSchedulingMode: hostSchedulingMode.value,
      nodeName: nodeName.value,
      nodeSelector: JSON.parse(JSON.stringify(nodeSelector.value)),
      nodeAffinityRequired: JSON.parse(JSON.stringify(nodeAffinityRequired.value)),
      nodeAffinityPreferred: JSON.parse(JSON.stringify(nodeAffinityPreferred.value)),
      tolerations: JSON.parse(JSON.stringify(tolerations.value)),
      podAffinity: JSON.parse(JSON.stringify(podAffinity.value)),
      podAntiAffinity: JSON.parse(JSON.stringify(podAntiAffinity.value)),
      topologySpreadConstraints: JSON.parse(JSON.stringify(topologySpreadConstraints.value)),
      schedulerName: schedulerName.value,
      priorityClassName: priorityClassName.value,
      priority: priority.value,
      runtimeClassName: runtimeClassName.value
    }
  }

  /**
   * 检查是否有未保存的修改
   */
  const hasUnsavedChanges = (): boolean => {
    if (!editing.value || !originalConfig.value) return false

    const current = JSON.stringify(getCurrentConfigSnapshot())
    const original = JSON.stringify(originalConfig.value)

    return current !== original
  }

  // ==================== 数据加载 ====================

  const loadConfig = async () => {
    // 🔧 修复：防止重复加载
    if (loading.value) return

    loading.value = true
    try {
      const response = await getSchedulingConfigApi(props.version.id)

      // 解析主机调度模式
      if (response.nodeName) {
        hostSchedulingMode.value = 'nodeName'
        nodeName.value = response.nodeName
        nodeSelector.value = {}
        nodeAffinityRequired.value = []
        nodeAffinityPreferred.value = []
      } else if (response.nodeSelector && Object.keys(response.nodeSelector).length > 0) {
        hostSchedulingMode.value = 'nodeSelector'
        nodeSelector.value = response.nodeSelector
        nodeName.value = ''
        nodeAffinityRequired.value = []
        nodeAffinityPreferred.value = []
      } else if (response.affinity?.nodeAffinity) {
        hostSchedulingMode.value = 'nodeAffinity'
        parseNodeAffinity(response.affinity.nodeAffinity)
        nodeName.value = ''
        nodeSelector.value = {}
      } else {
        hostSchedulingMode.value = 'auto'
        nodeName.value = ''
        nodeSelector.value = {}
        nodeAffinityRequired.value = []
        nodeAffinityPreferred.value = []
      }

      // 解析容忍配置
      tolerations.value = (response.tolerations || []).map((t) => ({
        key: t.key || '',
        operator: t.operator || 'Equal',
        value: t.value || '',
        effect: t.effect || 'NoSchedule',
        tolerationSeconds: t.tolerationSeconds
      }))

      // 解析Pod亲和性
      podAffinity.value = []
      podAntiAffinity.value = []
      if (response.affinity?.podAffinity) {
        parsePodAffinity(response.affinity.podAffinity, 'affinity')
      }
      if (response.affinity?.podAntiAffinity) {
        parsePodAffinity(response.affinity.podAntiAffinity, 'antiAffinity')
      }

      // 解析拓扑分布
      topologySpreadConstraints.value = response.topologySpreadConstraints || []

      // 解析高级配置
      schedulerName.value = response.schedulerName || ''
      priorityClassName.value = response.priorityClassName || ''
      priority.value = response.priority || 0
      runtimeClassName.value = response.runtimeClassName || ''

      // 保存原始配置用于对比（必须在所有数据加载完成后）
      originalConfig.value = getCurrentConfigSnapshot() as any
    } catch (error) {
      console.error('[调度管理] 加载失败:', error)
      ElMessage.error('加载调度配置失败')
    } finally {
      loading.value = false
    }
  }

  const parseNodeAffinity = (nodeAffinity: any) => {
    nodeAffinityRequired.value = []
    nodeAffinityPreferred.value = []

    if (nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution) {
      const required = nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution
      if (required.nodeSelectorTerms) {
        nodeAffinityRequired.value = required.nodeSelectorTerms.flatMap((term: any) =>
          (term.matchExpressions || []).map((expr: any) => ({
            key: expr.key || '',
            operator: expr.operator || 'In',
            values: (expr.values || []).join(',')
          }))
        )
      }
    }

    if (nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution) {
      nodeAffinityPreferred.value =
        nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution.flatMap((item: any) =>
          (item.preference?.matchExpressions || []).map((expr: any) => ({
            weight: item.weight || 50,
            key: expr.key || '',
            operator: expr.operator || 'In',
            values: (expr.values || []).join(',')
          }))
        )
    }
  }

  const parsePodAffinity = (affinity: any, type: 'affinity' | 'antiAffinity') => {
    const required = affinity.requiredDuringSchedulingIgnoredDuringExecution || []
    const preferred = affinity.preferredDuringSchedulingIgnoredDuringExecution || []

    const rules = [
      ...required.map((term: any) => {
        const labelSelector = term.labelSelector || {}
        return {
          type: 'required',
          configMode: 'manual', // 从后端加载的数据默认为手动模式
          applicationId: null,
          versionId: null,
          labelSelector: labelSelector.matchLabels || {},
          matchExpressions: labelSelector.matchExpressions || [],
          topologyKey: term.topologyKey || 'kubernetes.io/hostname',
          namespaces: term.namespaces || []
        }
      }),
      ...preferred.map((item: any) => {
        const podAffinityTerm = item.podAffinityTerm || {}
        const labelSelector = podAffinityTerm.labelSelector || {}

        return {
          type: 'preferred',
          weight: item.weight || 50,
          configMode: 'manual', // 从后端加载的数据默认为手动模式
          applicationId: null,
          versionId: null,
          labelSelector: labelSelector.matchLabels || {},
          matchExpressions: labelSelector.matchExpressions || [],
          topologyKey: podAffinityTerm.topologyKey || 'kubernetes.io/hostname',
          namespaces: podAffinityTerm.namespaces || []
        }
      })
    ]

    if (type === 'affinity') {
      podAffinity.value = rules
    } else {
      podAntiAffinity.value = rules
    }
  }

  // ==================== 刷新操作 ====================

  /** 🔧 新增：手动刷新 */
  const handleRefresh = async () => {
    await loadConfig()
    ElMessage.success('刷新成功')
  }

  // ==================== 编辑操作 ====================

  const startEdit = () => {
    // 进入编辑模式前，保存当前配置作为原始配置
    originalConfig.value = getCurrentConfigSnapshot() as any
    editing.value = true
  }

  /**
   * 取消编辑
   * 如果有未保存的修改，弹出确认框
   */
  const handleCancel = async () => {
    if (hasUnsavedChanges()) {
      try {
        await ElMessageBox.confirm('当前有未保存的修改，确定要取消吗？所有修改将丢失。', '提示', {
          confirmButtonText: '确定取消',
          cancelButtonText: '继续编辑',
          type: 'warning'
        })
      } catch {
        // 用户选择继续编辑
        return
      }
    }

    editing.value = false
    // 重新加载数据，恢复到原始状态
    await loadConfig()
  }

  // ==================== 保存操作 ====================

  const handleSave = async () => {
    try {
      await ElMessageBox.confirm('确定要保存调度配置吗？', '确认保存', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      })

      saving.value = true

      // 构建请求数据
      const data: UpdateSchedulingConfigRequest = {
        nodeName: '',
        nodeSelector: {},
        affinity: {}
      }

      // ========== 主机调度配置 ==========
      switch (hostSchedulingMode.value) {
        case 'auto':
          // 自动调度：不设置任何主机调度配置
          break

        case 'nodeName':
          if (!nodeName.value) {
            ElMessage.warning('请输入节点名称')
            saving.value = false
            return
          }
          data.nodeName = nodeName.value
          break

        case 'nodeSelector':
          if (Object.keys(nodeSelector.value).length === 0) {
            ElMessage.warning('请添加至少一个节点标签')
            saving.value = false
            return
          }
          data.nodeSelector = nodeSelector.value
          break

        case 'nodeAffinity':
          if (nodeAffinityRequired.value.length === 0 && nodeAffinityPreferred.value.length === 0) {
            ElMessage.warning('请添加至少一条节点亲和性规则')
            saving.value = false
            return
          }
          data.affinity!.nodeAffinity = buildNodeAffinity()
          break
      }

      // ========== 容忍配置 ==========
      if (tolerations.value.length > 0) {
        data.tolerations = tolerations.value.map((t) => ({
          key: t.key,
          operator: t.operator,
          value: t.value,
          effect: t.effect,
          ...(t.tolerationSeconds ? { tolerationSeconds: t.tolerationSeconds } : {})
        }))
      } else {
        data.tolerations = []
      }

      // ========== Pod亲和性配置 ==========
      if (podAffinity.value.length > 0) {
        data.affinity!.podAffinity = buildPodAffinity(podAffinity.value)
      }
      if (podAntiAffinity.value.length > 0) {
        data.affinity!.podAntiAffinity = buildPodAffinity(podAntiAffinity.value)
      }

      // 如果 affinity 对象为空，保留空对象让后端清空
      if (
        data.affinity &&
        !data.affinity.nodeAffinity &&
        !data.affinity.podAffinity &&
        !data.affinity.podAntiAffinity
      ) {
        data.affinity = {}
      }

      // ========== 拓扑分布配置 ==========
      if (topologySpreadConstraints.value.length > 0) {
        data.topologySpreadConstraints = topologySpreadConstraints.value
      } else {
        data.topologySpreadConstraints = []
      }

      // ========== 高级配置 ==========
      data.schedulerName = schedulerName.value || ''
      data.priorityClassName = priorityClassName.value || ''
      data.priority = priority.value || 0
      data.runtimeClassName = runtimeClassName.value || ''

      // 提交到服务器
      await updateSchedulingConfigApi(props.version.id, data)

      ElMessage.success('调度配置保存成功')
      editing.value = false

      // 重新加载配置，更新原始配置
      await loadConfig()
      emit('success')
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('[调度管理] 保存失败:', error)
        ElMessage.error('保存失败：' + (error.message || '未知错误'))
      }
    } finally {
      saving.value = false
    }
  }

  // ==================== 构建数据方法 ====================

  const buildNodeAffinity = () => {
    const nodeAffinity: any = {}

    if (nodeAffinityRequired.value.length > 0) {
      const validRules = nodeAffinityRequired.value.filter((r) => r.key)
      if (validRules.length > 0) {
        nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution = {
          nodeSelectorTerms: validRules.map((rule) => ({
            matchExpressions: [
              {
                key: rule.key,
                operator: rule.operator,
                ...(['In', 'NotIn'].includes(rule.operator) && rule.values
                  ? {
                      values: rule.values
                        .split(',')
                        .map((v: string) => v.trim())
                        .filter(Boolean)
                    }
                  : {})
              }
            ]
          }))
        }
      }
    }

    if (nodeAffinityPreferred.value.length > 0) {
      const validRules = nodeAffinityPreferred.value.filter((r) => r.key)
      if (validRules.length > 0) {
        nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution = validRules.map((rule) => ({
          weight: rule.weight || 50,
          preference: {
            matchExpressions: [
              {
                key: rule.key,
                operator: rule.operator,
                ...(['In', 'NotIn'].includes(rule.operator) && rule.values
                  ? {
                      values: rule.values
                        .split(',')
                        .map((v: string) => v.trim())
                        .filter(Boolean)
                    }
                  : {})
              }
            ]
          }
        }))
      }
    }

    return nodeAffinity
  }

  const buildPodAffinity = (rules: any[]) => {
    const required = rules.filter(
      (r) => r.type === 'required' && Object.keys(r.labelSelector).length > 0
    )
    const preferred = rules.filter(
      (r) => r.type === 'preferred' && Object.keys(r.labelSelector).length > 0
    )

    const result: any = {}

    if (required.length > 0) {
      result.requiredDuringSchedulingIgnoredDuringExecution = required.map((rule) => ({
        labelSelector: { matchLabels: rule.labelSelector },
        topologyKey: rule.topologyKey,
        ...(rule.namespaces && rule.namespaces.length > 0 ? { namespaces: rule.namespaces } : {})
      }))
    }

    if (preferred.length > 0) {
      result.preferredDuringSchedulingIgnoredDuringExecution = preferred.map((rule) => ({
        weight: rule.weight || 50,
        podAffinityTerm: {
          labelSelector: { matchLabels: rule.labelSelector },
          topologyKey: rule.topologyKey,
          ...(rule.namespaces && rule.namespaces.length > 0 ? { namespaces: rule.namespaces } : {})
        }
      }))
    }

    return result
  }

  // ==================== 生命周期 ====================

  watch(
    () => props.refreshTrigger,
    (newVal) => {
      if (newVal && newVal > 0) {
        editing.value = false
        loadConfig()
      }
    }
  )

  onMounted(() => {
    loadConfig()
  })

  // ==================== 暴露给父组件 ====================
  defineExpose({
    editing,
    hasUnsavedChanges
  })
</script>

<style lang="scss" scoped>
  .scheduling-management {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: #fff;

    .toolbar {
      display: flex;
      align-items: center;
      justify-content: flex-end;
      padding: 12px 16px;
      border-bottom: 1px solid var(--el-border-color-lighter);
      flex-shrink: 0;
      background: #fff;

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

    .loading-wrapper {
      flex: 1;
      display: flex;
      align-items: center;
      justify-content: center;
      min-height: 200px;
    }

    .content-wrapper {
      flex: 1;
      overflow-y: auto;
      display: flex;
      flex-direction: column;
      background: #fff;

      .tab-nav-wrapper {
        padding: 16px 20px 0;
        background: #fff;
        flex-shrink: 0;
      }

      .tab-nav {
        display: flex;
        gap: 4px;
        overflow-x: auto;
        padding-bottom: 0;
        border-bottom: 1px solid #e4e7ed;

        &::-webkit-scrollbar {
          height: 0;
        }

        .tab-item {
          display: inline-flex;
          align-items: center;
          gap: 6px;
          padding: 10px 20px;
          font-size: 13px;
          color: #606266;
          cursor: pointer;
          transition: all 0.2s;
          white-space: nowrap;
          position: relative;
          border-bottom: 2px solid transparent;
          margin-bottom: -1px;

          &:hover {
            color: #409eff;
          }

          &.active {
            color: #409eff;
            font-weight: 500;
            border-bottom-color: #409eff;
          }

          &.has-config .config-dot {
            width: 6px;
            height: 6px;
            background: #67c23a;
            border-radius: 50%;
            margin-left: 2px;
          }
        }
      }

      .unsaved-tip {
        margin: 16px 20px 0;
      }

      .config-content {
        flex: 1;
        overflow-y: auto;
        background: #fff;

        .config-section {
          min-height: 200px;
        }
      }
    }
  }
</style>
