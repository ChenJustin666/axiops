<template>
  <div class="pods-management-optimized user-page">
    <!-- 搜索栏 -->
    <ArtSearchBar
      v-show="showSearchBar"
      v-model="searchForm"
      :items="searchFormItems"
      :showExpand="false"
      @reset="handleReset"
      @search="handleSearch"
    />

    <!-- 工具栏 -->
    <ArtTableHeader
      :loading="loading"
      v-model:showSearchBar="showSearchBar"
      v-model:columns="columns"
      :showZebra="true"
      :showBorder="true"
      :showHeaderBackground="true"
      :fullClass="'pods-container'"
      :layout="'search,refresh,size,fullscreen,columns,settings'"
      @refresh="handleRefresh"
    >
      <template #left>
        <!-- 刷新按钮（保持独立） -->
        <ElDropdown @command="handleRefreshCommand" trigger="click">
          <ElButton :icon="RefreshCw" :loading="loading" v-ripple>
            {{ refreshButtonText }}
            <ChevronDown :size="14" style="margin-left: 4px" />
          </ElButton>
          <template #dropdown>
            <ElDropdownMenu>
              <ElDropdownItem command="manual">
                <RefreshCw :size="14" style="margin-right: 6px" />
                手动刷新
              </ElDropdownItem>
              <ElDropdownItem divided command="disable">
                <span :style="{ color: !autoRefreshEnabled ? '#409eff' : undefined }">
                  {{ !autoRefreshEnabled ? '✓ ' : '' }}关闭自动刷新
                </span>
              </ElDropdownItem>
              <ElDropdownItem command="1">
                <span :style="{ color: autoRefreshInterval === 1 ? '#409eff' : undefined }">
                  {{ autoRefreshInterval === 1 ? '✓ ' : '' }}每1秒刷新
                </span>
              </ElDropdownItem>
              <ElDropdownItem command="3">
                <span :style="{ color: autoRefreshInterval === 3 ? '#409eff' : undefined }">
                  {{ autoRefreshInterval === 3 ? '✓ ' : '' }}每3秒刷新
                </span>
              </ElDropdownItem>
              <ElDropdownItem command="5">
                <span :style="{ color: autoRefreshInterval === 5 ? '#409eff' : undefined }">
                  {{ autoRefreshInterval === 5 ? '✓ ' : '' }}每5秒刷新
                </span>
              </ElDropdownItem>
              <ElDropdownItem command="10">
                <span :style="{ color: autoRefreshInterval === 10 ? '#409eff' : undefined }">
                  {{ autoRefreshInterval === 10 ? '✓ ' : '' }}每10秒刷新
                </span>
              </ElDropdownItem>
              <ElDropdownItem command="30">
                <span :style="{ color: autoRefreshInterval === 30 ? '#409eff' : undefined }">
                  {{ autoRefreshInterval === 30 ? '✓ ' : '' }}每30秒刷新
                </span>
              </ElDropdownItem>
            </ElDropdownMenu>
          </template>
        </ElDropdown>

        <!-- 资源操作下拉菜单 -->
        <ElDropdown @command="handleOperationCommand" trigger="click">
          <ElButton :icon="Settings" v-ripple>
            资源操作
            <ChevronDown :size="14" style="margin-left: 4px" />
          </ElButton>
          <template #dropdown>
            <ElDropdownMenu>
              <!-- 查看类操作 -->
              <template v-if="operationConfig.yaml || operationConfig.detail">
                <ElDropdownItem
                  v-if="operationConfig.yaml"
                  command="yaml"
                  :disabled="resourceYamlLoading"
                >
                  <FileCode :size="14" style="margin-right: 6px" />
                  {{ resourceYamlLoading ? '加载中...' : '查看 YAML' }}
                </ElDropdownItem>
                <ElDropdownItem
                  v-if="operationConfig.detail"
                  command="detail"
                  :disabled="resourceDetailLoading"
                >
                  <FileSearch :size="14" style="margin-right: 6px" />
                  {{ resourceDetailLoading ? '加载中...' : '查看详情' }}
                </ElDropdownItem>
              </template>

              <!-- 变更控制操作（仅 Deployment） -->
              <template v-if="operationConfig.pauseRollout || operationConfig.resumeRollout">
                <ElDropdownItem divided />
                <ElDropdownItem
                  v-if="operationConfig.pauseRollout"
                  command="pauseRollout"
                  :disabled="pauseLoading"
                >
                  <PauseCircle :size="14" style="margin-right: 6px" />
                  {{ pauseLoading ? '处理中...' : '停止变更' }}
                </ElDropdownItem>
                <ElDropdownItem
                  v-if="operationConfig.resumeRollout"
                  command="resumeRollout"
                  :disabled="resumeLoading"
                >
                  <PlayCircle :size="14" style="margin-right: 6px; color: #67c23a" />
                  {{ resumeLoading ? '处理中...' : '恢复变更' }}
                </ElDropdownItem>
              </template>

              <!-- 手动触发（仅 CronJob） -->
              <template v-if="operationConfig.triggerJob">
                <ElDropdownItem divided />
                <ElDropdownItem command="triggerJob" :disabled="triggerLoading">
                  <Zap :size="14" style="margin-right: 6px; color: #409eff" />
                  {{ triggerLoading ? '触发中...' : '手动触发一次' }}
                </ElDropdownItem>
              </template>

              <!-- 生命周期操作 -->
              <template
                v-if="operationConfig.restart || operationConfig.stop || operationConfig.start"
              >
                <ElDropdownItem divided />
                <ElDropdownItem
                  v-if="operationConfig.restart"
                  command="restart"
                  :disabled="restartLoading"
                >
                  <RotateCw :size="14" style="margin-right: 6px; color: #e6a23c" />
                  {{ restartLoading ? '重启中...' : '重启资源' }}
                </ElDropdownItem>
                <ElDropdownItem v-if="operationConfig.stop" command="stop" :disabled="stopLoading">
                  <StopCircle :size="14" style="margin-right: 6px; color: #f56c6c" />
                  {{ stopLoading ? '停止中...' : '停止资源' }}
                </ElDropdownItem>
                <ElDropdownItem
                  v-if="operationConfig.start"
                  command="start"
                  :disabled="startLoading"
                >
                  <PlayCircle :size="14" style="margin-right: 6px; color: #67c23a" />
                  {{ startLoading ? '启动中...' : '启动资源' }}
                </ElDropdownItem>
              </template>
            </ElDropdownMenu>
          </template>
        </ElDropdown>
      </template>
    </ArtTableHeader>

    <!-- 表格 -->
    <ArtTable
      ref="tableRef"
      rowKey="name"
      :loading="loading"
      :columns="columns"
      :data="filteredPods"
      :stripe="true"
      :border="true"
    />

    <!-- 容器列表查看对话框 -->
    <ContainerViewer
      v-model="containerDialogVisible"
      :pod-name="selectedPod?.name || ''"
      :container-data="currentContainerData"
      :loading="containerDialogLoading"
      @retry="handleViewContainers(selectedPod!)"
    />

    <!-- 统一的 YAML 查看对话框 -->
    <ElDialog
      v-model="yamlDialog.visible"
      :title="yamlDialog.title"
      width="85%"
      top="5vh"
      class="yaml-dialog"
    >
      <YamlEditorPro
        v-model="yamlDialog.content"
        height="650px"
        :readonly="true"
        :show-status-bar="true"
        :validate-on-change="true"
        :filename="yamlDialog.filename"
      />
    </ElDialog>

    <!-- 日志查看组件 -->
    <PodLogViewer
      v-model="logDialogVisible"
      :workload-id="workspace.id"
      :pod-name="selectedPod?.name || ''"
    />

    <!-- 注入临时容器对话框 -->
    <InjectEphemeral
      v-model="ephemeralDialogVisible"
      :pod-name="selectedPod?.name || ''"
      :submitting="injectingEphemeral"
      :version="version"
      :application="application"
      :cluster="cluster"
      :workspace="workspace"
      @submit="handleInjectConfirm"
    />
  </div>
</template>

<script setup lang="ts">
  import { ref, reactive, computed, watch, onMounted, onUnmounted, h } from 'vue'
  import type { VNode } from 'vue'
  import { useRouter } from 'vue-router'
  import {
    ElMessage,
    ElMessageBox,
    ElTag,
    ElButton,
    ElDropdown,
    ElDropdownMenu,
    ElDropdownItem
  } from 'element-plus'
  import {
    RefreshCw,
    FileText,
    Terminal,
    Trash2,
    LogOut,
    Layers,
    Package,
    FolderOpen,
    MoreVertical,
    ChevronDown,
    FileCode,
    FileSearch,
    RotateCw,
    StopCircle,
    PlayCircle,
    PauseCircle,
    Zap,
    Settings,
    Info,
    Activity
  } from 'lucide-vue-next'
  import ArtButtonMore from '@/components/core/forms/art-button-more/index.vue'
  import type { ButtonMoreItem } from '@/components/core/forms/art-button-more/index.vue'
  import YamlEditorPro from '@/components/yaml-editor-pro/index.vue'
  import ContainerViewer from './components/container-viewer.vue'
  import InjectEphemeral from './components/inject-ephemeral.vue'
  import PodLogViewer from './components/pod-log-viewer.vue'
  import {
    getPodListApi,
    getPodContainerListApi,
    evictPodApi,
    deletePodApi,
    getPodYamlApi,
    getPodDetailApi,
    getResourceYamlApi,
    getResourceDetailYamlApi,
    restartResourceApi,
    stopResourceApi,
    startResourceApi,
    triggerCronJobApi,
    injectEphemeralContainerApi,
    type OnecProjectVersion,
    type OnecProjectApplication,
    type ProjectCluster,
    type ProjectWorkspace,
    type PodResourceList,
    type ContainerInfoList
  } from '@/api'
  import {
    getStatusType,
    getStatusIcon,
    canExecTerminal,
    getAllStatusOptions
  } from './pod-status-utils'
  import dayjs from 'dayjs'

  interface Props {
    version: OnecProjectVersion
    application: OnecProjectApplication
    cluster: ProjectCluster | null
    workspace: ProjectWorkspace | null
    refreshTrigger: number
  }

  const props = defineProps<Props>()
  const emit = defineEmits<{ refresh: [] }>()
  const router = useRouter()

  // 状态管理
  const loading = ref(false)
  const showSearchBar = ref(false)
  const tableRef = ref()
  const isInitialized = ref(false)

  // 自动刷新
  const autoRefreshEnabled = ref(false)
  const autoRefreshInterval = ref(0)
  const autoRefreshTimer = ref<NodeJS.Timeout | null>(null)
  const countdown = ref(0)
  const countdownTimer = ref<NodeJS.Timeout | null>(null)

  // Pod 数据
  const pods = ref<PodResourceList[]>([])
  let searchForm = reactive({
    name: '',
    status: ''
  })

  // 对话框
  const containerDialogVisible = ref(false)
  const containerDialogLoading = ref(false)
  const currentContainerData = ref<ContainerInfoList | null>(null)
  const logDialogVisible = ref(false)
  const ephemeralDialogVisible = ref(false)

  // 统一的 YAML 对话框状态
  const yamlDialog = reactive({
    visible: false,
    title: '',
    content: '',
    filename: ''
  })

  // 选中的 Pod
  const selectedPod = ref<PodResourceList | null>(null)

  // Loading 状态
  const injectingEphemeral = ref(false)
  const evictLoadingMap = ref<Record<string, boolean>>({})
  const deleteLoadingMap = ref<Record<string, boolean>>({})
  const resourceYamlLoading = ref(false)
  const resourceDetailLoading = ref(false)
  const restartLoading = ref(false)
  const stopLoading = ref(false)
  const startLoading = ref(false)
  const pauseLoading = ref(false)
  const resumeLoading = ref(false)
  const triggerLoading = ref(false)

  const operationConfig = computed(() => {
    const resourceType = props.application.resourceType.toLowerCase()

    return {
      // 获取YAML和详情 - 所有类型都支持
      yaml: ['deployment', 'daemonset', 'statefulset', 'cronjob', 'pod'].includes(resourceType),
      detail: ['deployment', 'daemonset', 'statefulset', 'cronjob', 'pod'].includes(resourceType),

      // 重启 - deployment, daemonset, statefulset
      restart: ['deployment', 'daemonset', 'statefulset'].includes(resourceType),

      // 停止/启动 - deployment, daemonset, statefulset, cronjob
      stop: ['deployment', 'daemonset', 'statefulset', 'cronjob'].includes(resourceType),
      start: ['deployment', 'daemonset', 'statefulset', 'cronjob'].includes(resourceType),

      // 停止变更/恢复变更 - 仅 deployment
      pauseRollout: resourceType === 'deployment',
      resumeRollout: resourceType === 'deployment',

      // 手动触发 - 仅 cronjob
      triggerJob: resourceType === 'cronjob'
    }
  })

  // 搜索表单配置
  const searchFormItems = computed(() => [
    {
      label: 'Pod名称',
      key: 'name',
      type: 'input',
      props: { clearable: true, placeholder: '请输入Pod名称' }
    },
    {
      label: '状态',
      key: 'status',
      type: 'select',
      props: {
        clearable: true,
        placeholder: '请选择状态',
        options: getAllStatusOptions()
      }
    }
  ])

  // 过滤后的 Pods
  const filteredPods = computed(() => {
    let result = pods.value
    if (searchForm.name) {
      result = result.filter((pod) =>
        pod.name.toLowerCase().includes(searchForm.name.toLowerCase())
      )
    }
    if (searchForm.status) {
      result = result.filter((pod) => pod.status === searchForm.status)
    }
    return result
  })

  // 刷新按钮文本
  const refreshButtonText = computed(() => {
    if (!autoRefreshEnabled.value) {
      return '刷新'
    }
    return `刷新 (${countdown.value}s)`
  })

  const formatTime = (timestamp: number) => {
    return dayjs(timestamp * 1000).format('MM-DD HH:mm')
  }

  // 打开 YAML 对话框的统一方法
  const openYamlDialog = (title: string, content: string, filename: string) => {
    yamlDialog.title = title
    yamlDialog.content = content
    yamlDialog.filename = filename
    yamlDialog.visible = true
  }

  const handleViewMonitor = (pod: PodResourceList) => {
    if (!props.cluster || !props.workspace) {
      return
    }

    try {
      const routeExists = router.hasRoute('podMonitoring')
      if (!routeExists) {
        console.error('路由 podMonitoring 未注册')
        return
      }

      // const routeData = router.resolve({
      //   name: 'PodMonitor',
      //   query: {
      //     clusterUuid: props.cluster.clusterUuid,
      //     namespace: props.workspace.namespace,
      //     podName: pod.name,
      //     clusterId: String(props.cluster.id)
      //   }
      // })

      router.push({
        name: 'podMonitoring',
        query: {
          clusterUuid: props.cluster.clusterUuid,
          namespace: props.workspace.namespace,
          podName: pod.name,
          clusterId: String(props.cluster.id)
        }
      })
    } catch (error) {
      console.error('打开 Pod 监控失败:', error)
    }
  }

  const handleButtonMoreClick = (item: ButtonMoreItem, row: PodResourceList) => {
    switch (item.key) {
      case 'monitor':
        handleViewMonitor(row)
        break
      case 'container':
        handleViewContainers(row)
        break
      case 'yaml':
        handleViewYaml(row)
        break
      case 'detail':
        handleViewPodDetail(row)
        break
      case 'inject':
        handleInjectEphemeral(row)
        break
      case 'evict':
        handleEvict(row)
        break
      case 'delete':
        handleDelete(row)
        break
    }
  }

  // 表格列配置
  const createTableColumns = () => {
    const allColumns = [
      {
        prop: 'name',
        label: 'Pod名称',
        minWidth: 200,
        showOverflowTooltip: true,
        visible: true,
        formatter: (row: PodResourceList) => {
          return h(
            'span',
            {
              class: 'pod-name-link',
              onClick: () => handleViewMonitor(row)
            },
            row.name
          )
        }
      },
      {
        prop: 'status',
        label: '状态',
        width: 120,
        visible: true,
        // ✅ 方案 1: 使用 div + flex（最推荐）
        formatter: (row: PodDetailInfo): VNode => {
          const Icon = getStatusIcon(row.status)
          return h(
            ElTag,
            {
              type: getStatusType(row.status)
            },
            () =>
              h(
                'div',
                {
                  style: {
                    display: 'flex',
                    alignItems: 'center',
                    gap: '6px'
                  }
                },
                [h(Icon, { size: 12 }), h('span', row.status)]
              )
          )
        }
      },
      {
        prop: 'ready',
        label: '就绪',
        width: 80,
        align: 'center' as const,
        visible: true
      },
      {
        prop: 'restarts',
        label: '重启',
        width: 70,
        align: 'center' as const,
        visible: true
      },
      {
        prop: 'node',
        label: '节点',
        width: 130,
        showOverflowTooltip: true,
        visible: true
      },
      {
        prop: 'podIP',
        label: 'Pod IP',
        width: 130,
        visible: true
      },
      {
        prop: 'age',
        label: '运行时长',
        width: 100,
        visible: true
      },
      {
        prop: 'creationTime',
        label: '创建时间',
        width: 140,
        visible: true,
        formatter: (row: PodResourceList) => formatTime(row.creationTime)
      },
      {
        prop: 'operation',
        label: '操作',
        width: 180,
        align: 'center' as const,
        fixed: 'right' as const,
        visible: true,
        formatter: (row: PodResourceList): VNode => {
          const isEvicting = evictLoadingMap.value[row.name] || false
          const isDeleting = deleteLoadingMap.value[row.name] || false
          const isAnyLoading = isEvicting || isDeleting

          return h(
            'div',
            {
              style: {
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
                gap: '0'
              }
            },
            [
              // 日志、终端、文件管理 - 紧凑排列，无图标
              h(
                'div',
                {
                  style: {
                    display: 'flex',
                    gap: '0',
                    marginRight: '2px'
                  }
                },
                [
                  h(
                    ElButton,
                    {
                      size: 'small',
                      type: 'primary',
                      text: true,
                      disabled: isAnyLoading,
                      onClick: () => handleViewLog(row),
                      style: { padding: '0 2px' }
                    },
                    () => '日志'
                  ),
                  h(
                    ElButton,
                    {
                      size: 'small',
                      type: 'success',
                      text: true,
                      disabled: isAnyLoading || !canExecTerminal(row.status),
                      onClick: () => handleTerminal(row),
                      style: { padding: '0 2px' }
                    },
                    () => '终端'
                  ),
                  h(
                    ElButton,
                    {
                      size: 'small',
                      type: 'primary',
                      text: true,
                      disabled: isAnyLoading || !canExecTerminal(row.status),
                      onClick: () => handleFileManager(row),
                      style: { padding: '0 2px' }
                    },
                    () => '文件'
                  )
                ]
              ),
              // 更多操作
              h(ArtButtonMore, {
                trigger: h(
                  ElButton,
                  {
                    size: 'small',
                    text: true,
                    disabled: isAnyLoading
                  },
                  () => h(MoreVertical, { size: 14 })
                ),
                list: [
                  {
                    key: 'monitor',
                    label: '监控',
                    icon: 'lucide:activity',
                    color: '#409eff',
                    disabled: isAnyLoading
                  },
                  {
                    key: 'container',
                    label: '容器列表',
                    icon: 'lucide:layers',
                    color: '#409eff',
                    disabled: isAnyLoading
                  },
                  {
                    key: 'yaml',
                    label: 'YAML',
                    icon: 'lucide:file-text',
                    color: '#67c23a',
                    disabled: isAnyLoading
                  },
                  {
                    key: 'detail',
                    label: '详情',
                    icon: 'lucide:info',
                    color: '#409eff',
                    disabled: isAnyLoading
                  },
                  {
                    key: 'inject',
                    label: '注入临时容器',
                    icon: 'lucide:package',
                    color: '#9333ea',
                    disabled: isAnyLoading
                  },
                  {
                    key: 'evict',
                    label: isEvicting ? '驱逐中...' : '驱逐',
                    icon: 'lucide:log-out',
                    color: '#e6a23c',
                    disabled: isAnyLoading
                  },
                  {
                    key: 'delete',
                    label: isDeleting ? '删除中...' : '删除',
                    icon: 'lucide:trash-2',
                    color: '#f56c6c',
                    disabled: isAnyLoading
                  }
                ] as ButtonMoreItem[],
                onClick: (item: ButtonMoreItem) => {
                  if (!isAnyLoading) {
                    handleButtonMoreClick(item, row)
                  }
                }
              })
            ]
          )
        }
      }
    ]

    const columns = ref(allColumns)

    return {
      columns
    }
  }

  const { columns } = createTableColumns()

  // 无感更新 Pod 列表
  const updatePodsSmooth = (newPods: PodResourceList[]) => {
    const oldPodsMap = new Map(pods.value.map((p) => [p.name, p]))
    const updatedPods: PodResourceList[] = []

    for (const newPod of newPods) {
      const oldPod = oldPodsMap.get(newPod.name)
      if (!oldPod) {
        updatedPods.push(newPod)
      } else {
        const hasChanged =
          oldPod.status !== newPod.status ||
          oldPod.ready !== newPod.ready ||
          oldPod.restarts !== newPod.restarts ||
          oldPod.podIP !== newPod.podIP ||
          oldPod.age !== newPod.age
        updatedPods.push(hasChanged ? newPod : oldPod)
      }
    }

    pods.value = updatedPods
  }

  // 加载 Pod 列表
  const loadPods = async (silent = false) => {
    if (!silent) {
      loading.value = true
    }

    try {
      const response = await getPodListApi(props.version.id)
      if (silent) {
        updatePodsSmooth(response || [])
      } else {
        pods.value = response || []
      }

      // ✅ 刷新 Pod 后，静默通知父组件刷新版本状态
      emit('refresh')
    } catch (error) {
      console.error('加载Pod列表失败:', error)
      if (!silent) {
        pods.value = []
      }
    } finally {
      if (!silent) {
        loading.value = false
      }
    }
  }

  // 启动自动刷新
  const startAutoRefresh = (interval: number) => {
    stopAutoRefresh()

    if (interval <= 0) {
      return
    }

    autoRefreshInterval.value = interval
    autoRefreshEnabled.value = true
    countdown.value = interval

    countdownTimer.value = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0) {
        countdown.value = interval
      }
    }, 1000)

    autoRefreshTimer.value = setInterval(() => {
      loadPods(true)
    }, interval * 1000)
  }

  // 停止自动刷新
  const stopAutoRefresh = () => {
    if (autoRefreshTimer.value) {
      clearInterval(autoRefreshTimer.value)
      autoRefreshTimer.value = null
    }
    if (countdownTimer.value) {
      clearInterval(countdownTimer.value)
      countdownTimer.value = null
    }
    autoRefreshEnabled.value = false
    autoRefreshInterval.value = 0
  }

  // 处理刷新命令
  const handleRefreshCommand = (command: string) => {
    if (command === 'manual') {
      handleRefresh()
    } else if (command === 'disable') {
      stopAutoRefresh()
      ElMessage.success('已关闭自动刷新')
    } else {
      const interval = parseInt(command)
      if (!isNaN(interval)) {
        startAutoRefresh(interval)
        ElMessage.success(`已开启自动刷新，每${interval}秒刷新一次`)
      }
    }
  }

  const handleOperationCommand = (command: string) => {
    switch (command) {
      case 'yaml':
        handleViewResourceYaml()
        break
      case 'detail':
        handleViewResourceDetail()
        break
      case 'restart':
        handleRestartResource()
        break
      case 'stop':
        handleStopResource()
        break
      case 'start':
        handleStartResource()
        break
      case 'pauseRollout':
        handlePauseRollout()
        break
      case 'resumeRollout':
        handleResumeRollout()
        break
      case 'triggerJob':
        handleTriggerCronJob()
        break
    }
  }

  // 查看资源 YAML
  const handleViewResourceYaml = async () => {
    openYamlDialog('资源 YAML', '# 加载中...', `${props.version.name || 'resource'}.yaml`)
    resourceYamlLoading.value = true

    try {
      const response = await getResourceYamlApi(props.version.id)
      yamlDialog.content = response || '# 获取失败'
    } catch (error) {
      console.error('获取资源YAML失败:', error)
      yamlDialog.content = '# 获取失败'
    } finally {
      resourceYamlLoading.value = false
    }
  }

  // 查看资源详情
  const handleViewResourceDetail = async () => {
    openYamlDialog('资源详情', '# 加载中...', `${props.version.name || 'resource'}-detail.yaml`)
    resourceDetailLoading.value = true

    try {
      const response = await getResourceDetailYamlApi(props.version.id)
      yamlDialog.content = response || '# 获取失败'
    } catch (error) {
      console.error('获取资源详情失败:', error)
      yamlDialog.content = '# 获取失败'
    } finally {
      resourceDetailLoading.value = false
    }
  }

  // 重启资源
  const handleRestartResource = async () => {
    try {
      await ElMessageBox.confirm('确定要重启此资源吗？这将导致所有 Pod 重启。', '重启确认', {
        type: 'warning',
        confirmButtonText: '确定',
        cancelButtonText: '取消'
      })

      restartLoading.value = true
      await restartResourceApi(props.version.id)
      ElMessage.success('重启成功')
      emit('refresh')
      loadPods(true)
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('重启失败:', error)
      }
    } finally {
      restartLoading.value = false
    }
  }

  // 停止资源
  const handleStopResource = async () => {
    try {
      await ElMessageBox.confirm('确定要停止此资源吗？这将停止所有 Pod。', '停止确认', {
        type: 'warning',
        confirmButtonText: '确定',
        cancelButtonText: '取消'
      })

      stopLoading.value = true
      await stopResourceApi(props.version.id)
      ElMessage.success('停止成功')
      emit('refresh')
      loadPods(true)
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('停止失败:', error)
      }
    } finally {
      stopLoading.value = false
    }
  }

  // 启动资源
  const handleStartResource = async () => {
    try {
      await ElMessageBox.confirm('确定要启动此资源吗？', '启动确认', {
        type: 'info',
        confirmButtonText: '确定',
        cancelButtonText: '取消'
      })

      startLoading.value = true
      await startResourceApi(props.version.id)
      ElMessage.success('启动成功')
      emit('refresh')
      loadPods(true)
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('启动失败:', error)
      }
    } finally {
      startLoading.value = false
    }
  }

  const handlePauseRollout = async () => {
    try {
      await ElMessageBox.confirm(
        '确定要停止此 Deployment 的变更吗？这将暂停滚动更新。',
        '停止变更确认',
        {
          type: 'warning',
          confirmButtonText: '确定',
          cancelButtonText: '取消'
        }
      )

      pauseLoading.value = true
      // 假设 API 方法为 pauseRolloutApi，需要在 @/api 中添加
      // await pauseRolloutApi(props.version.id)
      ElMessage.success('停止变更成功')
      emit('refresh')
      loadPods(true)
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('停止变更失败:', error)
      }
    } finally {
      pauseLoading.value = false
    }
  }

  const handleResumeRollout = async () => {
    try {
      await ElMessageBox.confirm(
        '确定要恢复此 Deployment 的变更吗？这将继续滚动更新。',
        '恢复变更确认',
        {
          type: 'info',
          confirmButtonText: '确定',
          cancelButtonText: '取消'
        }
      )

      resumeLoading.value = true
      // 假设 API 方法为 resumeRolloutApi，需要在 @/api 中添加
      // await resumeRolloutApi(props.version.id)
      ElMessage.success('恢复变更成功')
      emit('refresh')
      loadPods(true)
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('恢复变更失败:', error)
      }
    } finally {
      resumeLoading.value = false
    }
  }

  const handleTriggerCronJob = async () => {
    try {
      await ElMessageBox.confirm(
        '确定要手动触发此 CronJob 吗？这将立即创建一个 Job。',
        '手动触发确认',
        {
          type: 'info',
          confirmButtonText: '确定',
          cancelButtonText: '取消'
        }
      )

      triggerLoading.value = true
      await triggerCronJobApi(props.version.id)
      ElMessage.success('触发成功')
      emit('refresh')
      loadPods(true)
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('触发失败:', error)
      }
    } finally {
      triggerLoading.value = false
    }
  }

  // 查看容器信息
  const handleViewContainers = async (pod: PodResourceList) => {
    selectedPod.value = pod
    containerDialogVisible.value = true
    containerDialogLoading.value = true
    currentContainerData.value = null

    try {
      const response = await getPodContainerListApi(props.version.id, { podName: pod.name })
      currentContainerData.value = response || {
        initContainers: [],
        containers: [],
        ephemeralContainers: []
      }
    } catch (error) {
      console.error('加载容器信息失败:', error)
    } finally {
      containerDialogLoading.value = false
    }
  }

  // 查看 Pod YAML
  const handleViewYaml = async (pod: PodResourceList) => {
    selectedPod.value = pod
    openYamlDialog('Pod YAML', '# 加载中...', `${pod.name}.yaml`)

    try {
      const response = await getPodYamlApi(props.version.id, { podName: pod.name })
      yamlDialog.content = response || '# 获取失败'
    } catch (error) {
      console.error('获取YAML失败:', error)
      yamlDialog.content = '# 获取失败'
    }
  }

  // 查看 Pod 详情
  const handleViewPodDetail = async (pod: PodResourceList) => {
    selectedPod.value = pod
    openYamlDialog('Pod 详情', '# 加载中...', `${pod.name}-detail.yaml`)

    try {
      const response = await getPodDetailApi(props.version.id, { podName: pod.name })
      yamlDialog.content = response || '# 获取失败'
    } catch (error) {
      console.error('获取Pod详情失败:', error)
      yamlDialog.content = '# 获取失败'
    }
  }

  // 查看日志
  const handleViewLog = (pod: PodResourceList) => {
    selectedPod.value = pod
    logDialogVisible.value = true
  }

  // 打开终端
  const handleTerminal = (pod: PodResourceList) => {
    if (!canExecTerminal(pod.status)) {
      return
    }

    if (!props.workspace?.id || !pod.name) {
      return
    }

    try {
      const routeExists = router.hasRoute('TerminalManager')
      if (!routeExists) {
        console.error('路由 TerminalManager 未注册')
        return
      }

      const routeData = router.resolve({
        name: 'TerminalManager',
        query: {
          workloadId: String(props.workspace.id),
          podName: pod.name
        }
      })

      window.open(routeData.href, '_blank', 'noopener,noreferrer')
      ElMessage.success({ message: '终端已在新标签页打开', duration: 2000 })
    } catch (error) {
      console.error('打开终端失败:', error)
    }
  }

  // 文件管理
  const handleFileManager = (pod: PodResourceList) => {
    if (!canExecTerminal(pod.status)) {
      return
    }

    if (!props.workspace?.id || !pod.name) {
      return
    }

    try {
      const routeExists = router.hasRoute('FileManager')
      if (!routeExists) {
        console.error('路由 FileManager 未注册')
        return
      }

      const routeData = router.resolve({
        name: 'FileManager',
        query: {
          workspaceId: String(props.workspace.id),
          podName: pod.name
        }
      })

      window.open(routeData.href, '_blank', 'noopener,noreferrer')
      ElMessage.success({ message: '文件管理器已在新标签页打开', duration: 2000 })
    } catch (error) {
      console.error('打开文件管理器失败:', error)
    }
  }

  // 驱逐 Pod
  const handleEvict = async (pod: PodResourceList) => {
    try {
      await ElMessageBox.confirm(`确定要驱逐Pod "${pod.name}" 吗？`, '驱逐确认', {
        type: 'warning',
        confirmButtonText: '确定',
        cancelButtonText: '取消'
      })

      evictLoadingMap.value[pod.name] = true
      await evictPodApi(props.version.id, { podName: pod.name })
      ElMessage.success('驱逐成功')
      loadPods(true)
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('驱逐失败:', error)
      }
    } finally {
      evictLoadingMap.value[pod.name] = false
    }
  }

  // 删除 Pod
  const handleDelete = async (pod: PodResourceList) => {
    try {
      await ElMessageBox.confirm(`确定要删除Pod "${pod.name}" 吗？`, '删除确认', {
        type: 'warning',
        confirmButtonText: '确定',
        cancelButtonText: '取消'
      })

      deleteLoadingMap.value[pod.name] = true
      await deletePodApi(props.version.id, { podName: pod.name })
      ElMessage.success('删除成功')
      loadPods(true)
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('删除失败:', error)
      }
    } finally {
      deleteLoadingMap.value[pod.name] = false
    }
  }

  // 注入临时容器
  const handleInjectEphemeral = (pod: PodResourceList) => {
    selectedPod.value = pod
    ephemeralDialogVisible.value = true
  }

  // 处理注入确认
  const handleInjectConfirm = async (formData: any) => {
    if (!selectedPod.value) return

    injectingEphemeral.value = true
    try {
      const payload: any = {
        podName: selectedPod.value.name,
        containerName: '',
        image: '',
        command: [],
        args: []
      }

      if (
        formData.containerName &&
        typeof formData.containerName === 'string' &&
        formData.containerName.trim()
      ) {
        payload.containerName = formData.containerName
      }

      if (formData.image && typeof formData.image === 'string' && formData.image.trim()) {
        payload.image = formData.image
      }

      if (Array.isArray(formData.command) && formData.command.length > 0) {
        payload.command = formData.command
      }

      if (Array.isArray(formData.args) && formData.args.length > 0) {
        payload.args = formData.args
      }

      await injectEphemeralContainerApi(props.version.id, payload)
      ElMessage.success('临时容器注入成功')
      ephemeralDialogVisible.value = false
      loadPods(true)
    } catch (error: any) {
      console.error('注入失败:', error)
      let errorMsg = '注入临时容器失败'
      if (error.response?.data?.message) {
        errorMsg = error.response.data.message
      } else if (error.message) {
        errorMsg = error.message
      }
    } finally {
      injectingEphemeral.value = false
    }
  }

  const handleReset = () => {
    searchForm.name = ''
    searchForm.status = ''
  }

  const handleSearch = () => {}

  const handleRefresh = () => {
    loadPods(false)
  }

  watch(
    () => props.refreshTrigger,
    (newVal) => {
      if (newVal > 0 && isInitialized.value) {
        loadPods(false)
      }
    }
  )

  onMounted(() => {
    loadPods(false).then(() => {
      isInitialized.value = true
    })
  })

  onUnmounted(() => {
    stopAutoRefresh()
  })
</script>

<style lang="scss" scoped>
  .user-page {
    padding-bottom: 15px;
  }
  .art-table-card .el-card__body {
    height: 100%;
    min-height: 650px;
    overflow: hidden;
  }
  .pods-management-optimized {
    :deep(.pod-name-link) {
      color: #409eff;
      cursor: pointer;
      transition: all 0.2s;
      font-weight: 500;

      &:hover {
        color: #66b1ff;
        text-decoration: underline;
      }
    }

    .yaml-dialog {
      .yaml-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 16px;
        padding: 16px 24px 0;

        .yaml-info {
          display: flex;
          align-items: center;
          gap: 8px;

          .yaml-title {
            font-size: 14px;
            font-weight: 600;
            color: #303133;
            font-family: var(--ax-font-mono);
          }
        }
      }
    }
  }
</style>
