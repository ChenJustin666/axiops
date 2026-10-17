<template>
  <div class="probes-management">
    <!-- 顶部操作栏 -->
    <div class="toolbar">
      <!-- 容器选择器 - 分组下拉框 -->
      <div class="container-selector">
        <span class="selector-label">容器：</span>
        <ElSelect
          v-model="selectedContainer"
          placeholder="选择容器"
          style="width: 220px"
          :loading="loading"
          @change="handleContainerChange"
        >
          <!-- Init 容器分组 -->
          <ElOptionGroup v-if="initContainers.length > 0" label="Init 容器">
            <ElOption
              v-for="container in initContainers"
              :key="container.containerName"
              :label="container.containerName"
              :value="container.containerName"
            >
              <div class="container-option">
                <span>{{ container.containerName }}</span>
                <div class="probe-badges">
                  <ElTag size="small" type="warning">Init</ElTag>
                  <ElTag v-if="container.livenessProbe" size="small" type="danger">L</ElTag>
                  <ElTag v-if="container.readinessProbe" size="small" type="success">R</ElTag>
                  <ElTag v-if="container.startupProbe" size="small" type="info">S</ElTag>
                </div>
              </div>
            </ElOption>
          </ElOptionGroup>

          <!-- Main 容器分组 -->
          <ElOptionGroup v-if="mainContainers.length > 0" label="Main 容器">
            <ElOption
              v-for="container in mainContainers"
              :key="container.containerName"
              :label="container.containerName"
              :value="container.containerName"
            >
              <div class="container-option">
                <span>{{ container.containerName }}</span>
                <div class="probe-badges">
                  <ElTag size="small" type="primary">Main</ElTag>
                  <ElTag v-if="container.livenessProbe" size="small" type="danger">L</ElTag>
                  <ElTag v-if="container.readinessProbe" size="small" type="success">R</ElTag>
                  <ElTag v-if="container.startupProbe" size="small" type="info">S</ElTag>
                </div>
              </div>
            </ElOption>
          </ElOptionGroup>
        </ElSelect>

        <!-- 当前容器类型标签 -->
        <ElTag
          v-if="currentContainerType"
          :type="currentContainerType === 'init' ? 'warning' : 'primary'"
          size="small"
          class="current-type-tag"
        >
          {{ currentContainerType === 'init' ? 'Init 容器' : 'Main 容器' }}
        </ElTag>
      </div>

      <!-- 操作按钮 -->
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
        <ElButton size="small" :loading="loading" :disabled="editing" @click="handleRefresh">
          <RefreshCw :size="14" />
        </ElButton>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" v-loading="loading" class="loading-wrapper"></div>

    <!-- 主内容 -->
    <div v-else-if="selectedContainer" class="config-wrapper">
      <!-- 探针卡片列表 -->
      <div class="probes-grid">
        <!-- 存活探针 -->
        <div class="probe-card liveness">
          <div class="card-header">
            <div class="header-left">
              <Activity :size="18" class="probe-icon" />
              <span class="probe-title">存活探针</span>
              <ElTag size="small" type="danger">Liveness</ElTag>
            </div>
            <ElSwitch
              v-model="livenessEnabled"
              :disabled="!editing"
              @change="handleLivenessToggle"
            />
          </div>
          <div class="card-body">
            <div v-if="!livenessEnabled" class="empty-state">
              <span>未启用</span>
              <ElButton
                v-if="editing"
                type="primary"
                link
                size="small"
                @click="enableLivenessProbe"
              >
                点击启用
              </ElButton>
            </div>
            <ProbeConfigForm
              v-else
              :editing="editing"
              v-model:probe-type="livenessType"
              v-model:http-config="livenessConfig.httpGet"
              v-model:tcp-config="livenessConfig.tcpSocket"
              v-model:exec-config="livenessConfig.exec"
              v-model:grpc-config="livenessConfig.grpc"
              v-model:params="livenessConfig.params"
            />
          </div>
        </div>

        <!-- 就绪探针 -->
        <div class="probe-card readiness">
          <div class="card-header">
            <div class="header-left">
              <CheckCircle :size="18" class="probe-icon" />
              <span class="probe-title">就绪探针</span>
              <ElTag size="small" type="success">Readiness</ElTag>
            </div>
            <ElSwitch
              v-model="readinessEnabled"
              :disabled="!editing"
              @change="handleReadinessToggle"
            />
          </div>
          <div class="card-body">
            <div v-if="!readinessEnabled" class="empty-state">
              <span>未启用</span>
              <ElButton
                v-if="editing"
                type="primary"
                link
                size="small"
                @click="enableReadinessProbe"
              >
                点击启用
              </ElButton>
            </div>
            <ProbeConfigForm
              v-else
              :editing="editing"
              v-model:probe-type="readinessType"
              v-model:http-config="readinessConfig.httpGet"
              v-model:tcp-config="readinessConfig.tcpSocket"
              v-model:exec-config="readinessConfig.exec"
              v-model:grpc-config="readinessConfig.grpc"
              v-model:params="readinessConfig.params"
            />
          </div>
        </div>

        <!-- 启动探针 -->
        <div class="probe-card startup">
          <div class="card-header">
            <div class="header-left">
              <Zap :size="18" class="probe-icon" />
              <span class="probe-title">启动探针</span>
              <ElTag size="small" type="warning">Startup</ElTag>
            </div>
            <ElSwitch v-model="startupEnabled" :disabled="!editing" @change="handleStartupToggle" />
          </div>
          <div class="card-body">
            <div v-if="!startupEnabled" class="empty-state">
              <span>未启用</span>
              <ElButton v-if="editing" type="primary" link size="small" @click="enableStartupProbe">
                点击启用
              </ElButton>
            </div>
            <ProbeConfigForm
              v-else
              :editing="editing"
              v-model:probe-type="startupType"
              v-model:http-config="startupConfig.httpGet"
              v-model:tcp-config="startupConfig.tcpSocket"
              v-model:exec-config="startupConfig.exec"
              v-model:grpc-config="startupConfig.grpc"
              v-model:params="startupConfig.params"
            />
          </div>
        </div>
      </div>
    </div>

    <ElEmpty v-else description="请选择一个容器" :image-size="80" />
  </div>
</template>

<script setup lang="ts">
  import { ref, reactive, computed, watch, onMounted } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { Activity, CheckCircle, Zap, Edit, Save, RefreshCw } from 'lucide-vue-next'
  import {
    getProbesApi,
    updateProbesApi,
    type OnecProjectVersion,
    type ContainerProbes,
    type Probe
  } from '@/api'
  import ProbeConfigForm from './probes/ProbeConfigForm.vue'

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

  // 容器列表
  const containers = ref<ContainerProbes[]>([])
  const selectedContainer = ref('')

  // 原始数据备份
  const originalData = ref<ContainerProbes[]>([])

  // ==================== 计算属性 ====================

  const initContainers = computed(() => {
    return containers.value.filter((c) => c.containerType === 'init')
  })

  const mainContainers = computed(() => {
    return containers.value.filter((c) => c.containerType === 'main')
  })

  const currentContainerType = computed(() => {
    if (!selectedContainer.value) return ''
    const container = containers.value.find((c) => c.containerName === selectedContainer.value)
    return container?.containerType || ''
  })

  // ==================== 探针配置 ====================
  // 使用工厂函数创建默认配置，避免对象引用问题

  /** 创建默认的 HTTP 配置 */
  const createDefaultHttpConfig = (path: string = '/health') => ({
    path,
    port: 80,
    scheme: 'HTTP',
    host: ''
  })

  /** 创建默认的 TCP 配置 */
  const createDefaultTcpConfig = () => ({
    port: 80,
    host: ''
  })

  /** 创建默认的 Exec 配置 */
  const createDefaultExecConfig = () => ({
    commandStr: ''
  })

  /** 创建默认的 gRPC 配置 */
  const createDefaultGrpcConfig = () => ({
    port: 50051,
    service: ''
  })

  /** 创建默认的探针参数 */
  const createDefaultParams = (probeKind: 'liveness' | 'readiness' | 'startup') => ({
    initialDelaySeconds: probeKind === 'startup' ? 0 : probeKind === 'readiness' ? 5 : 10,
    periodSeconds: 10,
    timeoutSeconds: 1,
    failureThreshold: probeKind === 'startup' ? 30 : 3,
    successThreshold: 1
  })

  // 存活探针
  const livenessEnabled = ref(false)
  const livenessType = ref<'httpGet' | 'tcpSocket' | 'exec' | 'grpc'>('httpGet')
  const livenessConfig = reactive({
    httpGet: createDefaultHttpConfig('/health'),
    tcpSocket: createDefaultTcpConfig(),
    exec: createDefaultExecConfig(),
    grpc: createDefaultGrpcConfig(),
    params: createDefaultParams('liveness')
  })

  // 就绪探针
  const readinessEnabled = ref(false)
  const readinessType = ref<'httpGet' | 'tcpSocket' | 'exec' | 'grpc'>('httpGet')
  const readinessConfig = reactive({
    httpGet: createDefaultHttpConfig('/ready'),
    tcpSocket: createDefaultTcpConfig(),
    exec: createDefaultExecConfig(),
    grpc: createDefaultGrpcConfig(),
    params: createDefaultParams('readiness')
  })

  // 启动探针
  const startupEnabled = ref(false)
  const startupType = ref<'httpGet' | 'tcpSocket' | 'exec' | 'grpc'>('httpGet')
  const startupConfig = reactive({
    httpGet: createDefaultHttpConfig('/startup'),
    tcpSocket: createDefaultTcpConfig(),
    exec: createDefaultExecConfig(),
    grpc: createDefaultGrpcConfig(),
    params: createDefaultParams('startup')
  })

  // ==================== 核心方法 ====================

  /** 加载健康检查配置 */
  const loadProbes = async () => {
    // 防止重复加载
    if (loading.value) return

    loading.value = true
    try {
      const response = await getProbesApi(props.version.id)
      containers.value = response.containers || []

      // 默认选中第一个容器（优先选择 main 容器）
      if (containers.value.length > 0 && !selectedContainer.value) {
        const firstMain = mainContainers.value[0]
        const firstInit = initContainers.value[0]
        selectedContainer.value = firstMain?.containerName || firstInit?.containerName || ''
        updateCurrentProbes()
      }

      // 备份原始数据（深拷贝）
      originalData.value = JSON.parse(JSON.stringify(containers.value))
    } catch (error) {
      console.error('[健康检查] 加载失败:', error)
      ElMessage.error('加载健康检查配置失败')
    } finally {
      loading.value = false
    }
  }

  /** 更新当前容器的探针配置到表单 */
  const updateCurrentProbes = () => {
    const container = containers.value.find((c) => c.containerName === selectedContainer.value)
    if (!container) return

    // 加载存活探针
    if (container.livenessProbe) {
      livenessEnabled.value = true
      loadProbeConfig(container.livenessProbe, 'liveness')
    } else {
      livenessEnabled.value = false
      resetProbeConfig('liveness')
    }

    // 加载就绪探针
    if (container.readinessProbe) {
      readinessEnabled.value = true
      loadProbeConfig(container.readinessProbe, 'readiness')
    } else {
      readinessEnabled.value = false
      resetProbeConfig('readiness')
    }

    // 加载启动探针
    if (container.startupProbe) {
      startupEnabled.value = true
      loadProbeConfig(container.startupProbe, 'startup')
    } else {
      startupEnabled.value = false
      resetProbeConfig('startup')
    }
  }

  /** 将探针数据加载到配置表单 */
  const loadProbeConfig = (probe: Probe, probeKind: 'liveness' | 'readiness' | 'startup') => {
    const config =
      probeKind === 'liveness'
        ? livenessConfig
        : probeKind === 'readiness'
          ? readinessConfig
          : startupConfig
    const typeRef =
      probeKind === 'liveness'
        ? livenessType
        : probeKind === 'readiness'
          ? readinessType
          : startupType

    if (probe.httpGet) {
      typeRef.value = 'httpGet'
      // 使用 Object.assign 保持响应式
      Object.assign(config.httpGet, {
        path: probe.httpGet.path || '/',
        port: probe.httpGet.port || 80,
        scheme: probe.httpGet.scheme || 'HTTP',
        host: probe.httpGet.host || ''
      })
    } else if (probe.tcpSocket) {
      typeRef.value = 'tcpSocket'
      Object.assign(config.tcpSocket, {
        port: probe.tcpSocket.port || 80,
        host: probe.tcpSocket.host || ''
      })
    } else if (probe.exec) {
      typeRef.value = 'exec'
      config.exec.commandStr = (probe.exec.command || []).join('\n')
    } else if (probe.grpc) {
      typeRef.value = 'grpc'
      Object.assign(config.grpc, {
        port: probe.grpc.port || 50051,
        service: probe.grpc.service || ''
      })
    }

    // 更新通用参数
    Object.assign(config.params, {
      initialDelaySeconds: probe.initialDelaySeconds ?? 0,
      periodSeconds: probe.periodSeconds ?? 10,
      timeoutSeconds: probe.timeoutSeconds ?? 1,
      failureThreshold: probe.failureThreshold ?? 3,
      successThreshold: probe.successThreshold ?? 1
    })
  }

  /** 重置探针配置为默认值 */
  const resetProbeConfig = (probeKind: 'liveness' | 'readiness' | 'startup') => {
    const config =
      probeKind === 'liveness'
        ? livenessConfig
        : probeKind === 'readiness'
          ? readinessConfig
          : startupConfig
    const typeRef =
      probeKind === 'liveness'
        ? livenessType
        : probeKind === 'readiness'
          ? readinessType
          : startupType

    // 重置类型
    typeRef.value = 'httpGet'

    // 根据探针类型设置默认路径
    const defaultPath =
      probeKind === 'liveness' ? '/health' : probeKind === 'readiness' ? '/ready' : '/startup'

    // 使用 Object.assign 保持响应式并重置配置
    Object.assign(config.httpGet, createDefaultHttpConfig(defaultPath))
    Object.assign(config.tcpSocket, createDefaultTcpConfig())
    Object.assign(config.exec, createDefaultExecConfig())
    Object.assign(config.grpc, createDefaultGrpcConfig())
    Object.assign(config.params, createDefaultParams(probeKind))
  }

  /** 从表单构建探针对象 */
  const buildProbe = (probeKind: 'liveness' | 'readiness' | 'startup'): Probe | undefined => {
    const config =
      probeKind === 'liveness'
        ? livenessConfig
        : probeKind === 'readiness'
          ? readinessConfig
          : startupConfig
    const type =
      probeKind === 'liveness'
        ? livenessType.value
        : probeKind === 'readiness'
          ? readinessType.value
          : startupType.value
    const enabled =
      probeKind === 'liveness'
        ? livenessEnabled.value
        : probeKind === 'readiness'
          ? readinessEnabled.value
          : startupEnabled.value

    if (!enabled) return undefined

    const probe: Probe = {
      type,
      ...config.params
    }

    if (type === 'httpGet') {
      probe.httpGet = {
        path: config.httpGet.path,
        port: config.httpGet.port,
        scheme: config.httpGet.scheme
      }
      if (config.httpGet.host) probe.httpGet.host = config.httpGet.host
    } else if (type === 'tcpSocket') {
      probe.tcpSocket = { port: config.tcpSocket.port }
      if (config.tcpSocket.host) probe.tcpSocket.host = config.tcpSocket.host
    } else if (type === 'exec') {
      probe.exec = {
        command: config.exec.commandStr
          .split('\n')
          .map((s) => s.trim())
          .filter((s) => s)
      }
    } else if (type === 'grpc') {
      probe.grpc = { port: config.grpc.port }
      if (config.grpc.service) probe.grpc.service = config.grpc.service
    }

    return probe
  }

  /** 同步当前表单配置到 containers 数据 */
  const syncCurrentProbes = () => {
    const container = containers.value.find((c) => c.containerName === selectedContainer.value)
    if (!container) return

    container.livenessProbe = buildProbe('liveness')
    container.readinessProbe = buildProbe('readiness')
    container.startupProbe = buildProbe('startup')
  }

  // ==================== 事件处理 ====================

  const handleContainerChange = () => {
    // 切换前先保存当前容器的配置
    if (editing.value) {
      syncCurrentProbes()
    }
    updateCurrentProbes()
  }

  const startEdit = () => {
    editing.value = true
  }

  const cancelEdit = () => {
    // 恢复原始数据（深拷贝）
    containers.value = JSON.parse(JSON.stringify(originalData.value))
    updateCurrentProbes()
    editing.value = false
    ElMessage.info('已取消编辑')
  }

  /** 手动刷新 */
  const handleRefresh = async () => {
    await loadProbes()
    ElMessage.success('刷新成功')
  }

  /** 启用存活探针 */
  const enableLivenessProbe = () => {
    livenessEnabled.value = true
    resetProbeConfig('liveness')
  }

  /** 启用就绪探针 */
  const enableReadinessProbe = () => {
    readinessEnabled.value = true
    resetProbeConfig('readiness')
  }

  /** 启用启动探针 */
  const enableStartupProbe = () => {
    startupEnabled.value = true
    resetProbeConfig('startup')
  }

  /** 存活探针开关切换处理 */
  const handleLivenessToggle = (enabled: boolean) => {
    if (enabled) {
      // 启用时重置为默认配置
      resetProbeConfig('liveness')
    }
    // 禁用时不需要额外处理，buildProbe 会返回 undefined
  }

  /** 就绪探针开关切换处理 */
  const handleReadinessToggle = (enabled: boolean) => {
    if (enabled) {
      resetProbeConfig('readiness')
    }
  }

  /** 启动探针开关切换处理 */
  const handleStartupToggle = (enabled: boolean) => {
    if (enabled) {
      resetProbeConfig('startup')
    }
  }

  /** 保存所有容器的健康检查配置 */
  const handleSaveAll = async () => {
    // 先同步当前容器的配置
    syncCurrentProbes()

    try {
      await ElMessageBox.confirm('确定要保存所有容器的健康检查配置吗？', '确认保存', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      })

      saving.value = true

      // 构建提交数据 - 提交所有容器
      const requestData = {
        containers: containers.value.map((container) => ({
          containerName: container.containerName,
          containerType: container.containerType,
          livenessProbe: container.livenessProbe,
          readinessProbe: container.readinessProbe,
          startupProbe: container.startupProbe
        }))
      }

      await updateProbesApi(props.version.id, requestData)

      ElMessage.success('健康检查配置保存成功')
      editing.value = false
      await loadProbes()
      emit('success')
    } catch (error: unknown) {
      // 检查是否为用户取消操作
      if (error === 'cancel' || (error instanceof Error && error.message === 'cancel')) {
        return
      }
      console.error('[健康检查] 保存失败:', error)
      const errorMessage = error instanceof Error ? error.message : '未知错误'
      ElMessage.error('保存失败：' + errorMessage)
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
        loadProbes()
      }
    }
  )

  onMounted(() => {
    loadProbes()
  })

  // ==================== 暴露给父组件 ====================
  defineExpose({
    editing,
    hasUnsavedChanges: () => {
      if (!editing.value) return false
      // 同步当前配置再比较
      syncCurrentProbes()
      const current = JSON.stringify(containers.value)
      const original = JSON.stringify(originalData.value)
      return current !== original
    }
  })
</script>

<style lang="scss" scoped>
  .probes-management {
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

      .container-selector {
        display: flex;
        align-items: center;
        gap: 10px;

        .selector-label {
          font-size: 13px;
          color: #606266;
          flex-shrink: 0;
        }

        .current-type-tag {
          margin-left: 4px;
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

      .probe-badges {
        display: flex;
        gap: 4px;
      }
    }

    .loading-wrapper {
      flex: 1;
      display: flex;
      align-items: center;
      justify-content: center;
      min-height: 200px;
    }

    .config-wrapper {
      flex: 1;
      overflow-y: auto;
      padding: 16px;

      .probes-grid {
        display: flex;
        flex-direction: column;
        gap: 12px;

        .probe-card {
          border-radius: 8px;
          border: 1px solid #e4e7ed;
          overflow: hidden;

          .card-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 10px 14px;
            background: #f5f7fa;
            border-bottom: 1px solid #e4e7ed;

            .header-left {
              display: flex;
              align-items: center;
              gap: 8px;

              .probe-icon {
                color: #606266;
              }

              .probe-title {
                font-size: 13px;
                font-weight: 600;
                color: #303133;
              }
            }
          }

          .card-body {
            padding: 12px 14px;
            min-height: 60px;

            .empty-state {
              display: flex;
              align-items: center;
              justify-content: center;
              gap: 8px;
              color: #909399;
              font-size: 13px;
              padding: 10px 0;
            }
          }

          &.liveness {
            .card-header {
              background: linear-gradient(135deg, #fef2f2 0%, #fee2e2 100%);
              border-bottom-color: #fecaca;

              .probe-icon {
                color: #ef4444;
              }
            }
          }

          &.readiness {
            .card-header {
              background: linear-gradient(135deg, #f0fdf4 0%, #dcfce7 100%);
              border-bottom-color: #bbf7d0;

              .probe-icon {
                color: #22c55e;
              }
            }
          }

          &.startup {
            .card-header {
              background: linear-gradient(135deg, #fffbeb 0%, #fef3c7 100%);
              border-bottom-color: #fde68a;

              .probe-icon {
                color: #f59e0b;
              }
            }
          }
        }
      }
    }

    :deep(.el-empty) {
      padding: 40px 0;
    }
  }
</style>
