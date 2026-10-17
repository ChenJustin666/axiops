<template>
  <div class="job-manager-page art-full-height">
    <!-- 面包屑式选择器 -->
    <div class="breadcrumb-selector">
      <div class="breadcrumb-content">
        <!-- 集群选择 -->
        <div class="breadcrumb-item">
          <span class="breadcrumb-label">
            <Server :size="14" />
            集群
          </span>
          <ElSelect
            v-model="selectedClusterId"
            placeholder="选择集群"
            clearable
            filterable
            :filter-method="filterClusters"
            size="default"
            :disabled="!selectedProject"
            :loading="loadingClusters"
            @change="handleClusterChange"
            @clear="handleClusterClear"
            @visible-change="handleClusterDropdownVisibleChange"
            popper-class="job-cluster-dropdown"
            class="breadcrumb-select"
          >
            <ElOption
              v-for="cluster in filteredClusters"
              :key="cluster.id"
              :label="cluster.clusterName"
              :value="cluster.id"
            >
              <div class="cluster-option">
                <span class="option-name">{{ cluster.clusterName }}</span>
              </div>
            </ElOption>
          </ElSelect>
        </div>

        <div class="breadcrumb-separator">
          <ChevronRight :size="16" />
        </div>

        <!-- 工作空间选择 -->
        <div class="breadcrumb-item">
          <span class="breadcrumb-label">
            <Box :size="14" />
            工作空间
          </span>
          <ElSelect
            v-model="selectedWorkspaceId"
            placeholder="选择工作空间"
            clearable
            filterable
            :filter-method="filterWorkspaces"
            size="default"
            :disabled="!selectedClusterId"
            :loading="loadingWorkspaces"
            @change="handleWorkspaceChange"
            @clear="handleWorkspaceClear"
            @visible-change="handleWorkspaceDropdownVisibleChange"
            popper-class="job-workspace-dropdown"
            class="breadcrumb-select"
          >
            <ElOption
              v-for="workspace in filteredWorkspaces"
              :key="workspace.id"
              :label="workspace.name"
              :value="workspace.id"
            >
              <div class="workspace-option">
                <span class="option-name">{{ workspace.name }}</span>
                <ElTag size="small" type="info">{{ workspace.namespace }}</ElTag>
              </div>
            </ElOption>
          </ElSelect>
        </div>

        <template v-if="viewMode === 'job-pods' && selectedJob">
          <div class="breadcrumb-separator">
            <ChevronRight :size="16" />
          </div>
          <div class="breadcrumb-item">
            <span class="breadcrumb-label">
              <Briefcase :size="14" />
              Job
            </span>
            <ElTag type="primary" size="default">{{ selectedJob.name }}</ElTag>
          </div>
        </template>
      </div>

      <!-- 操作按钮 -->
      <div class="breadcrumb-actions">
        <ElButton v-if="viewMode === 'job-pods'" :icon="ArrowLeft" @click="handleBackToJobList">
          返回 Job 列表
        </ElButton>

        <!-- 自动刷新下拉菜单 -->
        <ElDropdown @command="handleRefreshCommand" trigger="click">
          <ElButton :icon="RefreshCw" :loading="loading">
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
              <ElDropdownItem command="60">
                <span :style="{ color: autoRefreshInterval === 60 ? '#409eff' : undefined }">
                  {{ autoRefreshInterval === 60 ? '✓ ' : '' }}每60秒刷新
                </span>
              </ElDropdownItem>
            </ElDropdownMenu>
          </template>
        </ElDropdown>
      </div>
    </div>

    <!-- 主体内容 -->
    <template v-if="selectedWorkspace">
      <template v-if="viewMode === 'list'">
        <!-- 搜索栏 -->
        <ArtSearchBar
          v-show="showSearchBar"
          v-model="searchForm"
          :items="searchFormItems"
          :showExpand="false"
          @reset="handleReset"
          @search="handleSearch"
        />

        <ElCard
          class="art-table-card"
          shadow="never"
          :style="{ 'margin-top': showSearchBar ? '12px' : '0' }"
        >
          <!-- 工具栏 -->
          <ArtTableHeader
            :loading="loading"
            v-model:showSearchBar="showSearchBar"
            v-model:columns="columns"
            :showZebra="true"
            :showBorder="true"
            :showHeaderBackground="true"
            :fullClass="'art-page-view'"
            :layout="'search,refresh,size,fullscreen,columns,settings'"
            @refresh="handleRefresh"
          />

          <!-- 表格 -->
          <ArtTable
            ref="tableRef"
            rowKey="name"
            :loading="loading"
            :columns="columns"
            :data="tableData"
            :pagination="pagination"
            :stripe="true"
            :border="true"
            @pagination:size-change="handleSizeChange"
            @pagination:current-change="handleCurrentChange"
          />
        </ElCard>
      </template>

      <template v-else-if="viewMode === 'job-pods' && selectedJob">
        <!-- Job 信息卡片 -->
        <ElCard class="job-info-card" shadow="never">
          <div class="job-info-content">
            <div class="job-info-left">
              <div class="job-icon">
                <Briefcase :size="24" />
              </div>
              <div class="job-details">
                <div class="job-name">{{ selectedJob.name }}</div>
                <div class="job-meta">
                  <ElTag :type="getStatusType(selectedJob.status)" size="small">
                    {{ selectedJob.status }}
                  </ElTag>
                  <span class="job-namespace">{{ selectedJob.namespace }}</span>
                  <span class="job-time"
                    >创建于 {{ formatTime(selectedJob.creationTimestamp) }}</span
                  >
                </div>
              </div>
            </div>
            <div class="job-info-right">
              <div class="job-stats">
                <div class="stat-item success">
                  <div class="stat-value">{{ selectedJob.succeeded }}</div>
                  <div class="stat-label">成功</div>
                </div>
                <div class="stat-item danger">
                  <div class="stat-value">{{ selectedJob.failed }}</div>
                  <div class="stat-label">失败</div>
                </div>
                <div class="stat-item primary">
                  <div class="stat-value">{{ selectedJob.active }}</div>
                  <div class="stat-label">运行中</div>
                </div>
                <div class="stat-item">
                  <div class="stat-value">{{ selectedJob.completions }}</div>
                  <div class="stat-label">期望完成</div>
                </div>
              </div>
            </div>
          </div>
        </ElCard>

        <!-- Pod 列表搜索栏 -->
        <ArtSearchBar
          v-show="showPodSearchBar"
          v-model="podSearchForm"
          :items="podSearchFormItems"
          :showExpand="false"
          @reset="handlePodReset"
          @search="handlePodSearch"
        />

        <!-- Pod 列表表格 -->
        <ElCard
          class="art-table-card"
          shadow="never"
          :style="{ 'margin-top': showPodSearchBar ? '12px' : '12px' }"
        >
          <ArtTableHeader
            :loading="podLoading"
            v-model:showSearchBar="showPodSearchBar"
            v-model:columns="podColumns"
            :showZebra="true"
            :showBorder="true"
            :showHeaderBackground="true"
            :fullClass="'art-page-view'"
            :layout="'search,refresh,size,fullscreen,columns,settings'"
            @refresh="loadJobPods"
          />

          <ArtTable
            ref="podTableRef"
            rowKey="name"
            :loading="podLoading"
            :columns="podColumns"
            :data="filteredJobPods"
            :stripe="true"
            :border="true"
          />
        </ElCard>
      </template>
    </template>

    <!-- 空状态：根据当前选择状态显示对应提示 -->
    <div v-else class="empty-container">
      <SelectionGuideState
        level="project-cluster-workspace"
        :project="selectedProject"
        :cluster="selectedCluster"
        :workspace="selectedWorkspace"
        resource-name="Job"
      />
    </div>

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

    <!-- 容器列表查看对话框 -->
    <ContainerViewer
      v-model="containerDialogVisible"
      :pod-name="selectedPod?.name || ''"
      :container-data="currentContainerData"
      :loading="containerDialogLoading"
      @retry="handleViewContainers(selectedPod!)"
    />

    <!-- 日志查看组件 -->
    <PodLogViewer
      v-model="logDialogVisible"
      :workload-id="selectedWorkspace?.id || 0"
      :pod-name="selectedPod?.name || ''"
    />

    <!-- 注入临时容器对话框 -->
    <InjectEphemeral
      v-model="ephemeralDialogVisible"
      :pod-name="selectedPod?.name || ''"
      :submitting="injectingEphemeral"
      :version="dummyVersion"
      :application="dummyApplication"
      :cluster="selectedCluster"
      :workspace="selectedWorkspace"
      @submit="handleInjectConfirm"
    />
  </div>
</template>

<script setup lang="ts">
  import SelectionGuideState from '@/components/SelectionGuideState/index.vue'
  import { ref, reactive, computed, watch, onMounted, onActivated, onUnmounted, h } from 'vue'
  import type { VNode } from 'vue'
  import { useRouter } from 'vue-router'
  import {
    ElMessage,
    ElMessageBox,
    ElTag,
    ElButton,
    ElDropdown,
    ElDropdownMenu,
    ElDropdownItem,
    ElDialog
  } from 'element-plus'
  import {
    RefreshCw,
    Server,
    Box,
    ChevronRight,
    CheckCircle,
    XCircle,
    Clock,
    AlertCircle,
    AlertTriangle,
    FileText,
    Terminal,
    MoreVertical,
    ChevronDown,
    Trash2,
    Info,
    List,
    Briefcase,
    ArrowLeft,
    Layers,
    Package,
    LogOut,
    FolderOpen
  } from 'lucide-vue-next'
  import { useProjectStore } from '@/store/modules/project'
  import ArtButtonMore from '@/components/core/forms/art-button-more/index.vue'
  import type { ButtonMoreItem } from '@/components/core/forms/art-button-more/index.vue'
  import YamlEditorPro from '@/components/yaml-editor-pro/index.vue'
  import ContainerViewer from '@/views/workspace/application/components/version/tabs/common/components/container-viewer.vue'
  import InjectEphemeral from '@/views/workspace/application/components/version/tabs/common/components/inject-ephemeral.vue'
  import PodLogViewer from '@/views/workspace/application/components/version/tabs/common/components/pod-log-viewer.vue'
  import {
    searchProjectClusterApi,
    searchProjectWorkspaceApi,
    listJobsApi,
    listJobPodsApi,
    getCronJobJobDetailsApi,
    deleteCronJobJobApi,
    getPodContainerListApi,
    getPodYamlApi,
    getPodDetailApi,
    evictPodApi,
    deletePodApi,
    injectEphemeralContainerApi,
    type ProjectCluster,
    type ProjectWorkspace,
    type JobInfo,
    type PodResourceList,
    type ContainerInfoList,
    type OnecProjectVersion,
    type OnecProjectApplication
  } from '@/api'
  import dayjs from 'dayjs'

  defineOptions({ name: 'JobManager' })

  const router = useRouter()
  const projectStore = useProjectStore()
  const selectedProject = computed(() => projectStore.selectedProject)

  // 当前标签页存储的 key
  const STORAGE_KEY_CLUSTER = 'job-manager-selected-cluster'
  const STORAGE_KEY_WORKSPACE = 'job-manager-selected-workspace'

  // 工具函数：从 sessionStorage 加载
  function loadFromStorage(key: string): number | null {
    try {
      const value = sessionStorage.getItem(key)
      return value ? Number(value) : null
    } catch (error) {
      console.error('加载存储失败:', error)
      return null
    }
  }

  // 工具函数：保存到 sessionStorage
  function saveToStorage(key: string, value: number | null) {
    try {
      if (value !== null) {
        sessionStorage.setItem(key, String(value))
      } else {
        sessionStorage.removeItem(key)
      }
    } catch (error) {
      console.error('保存存储失败:', error)
    }
  }

  // 视图模式：'list' 或 'job-pods'
  const viewMode = ref<'list' | 'job-pods'>('list')

  // 数据状态
  const clusters = ref<ProjectCluster[]>([])
  const workspaces = ref<ProjectWorkspace[]>([])
  const tableData = ref<JobInfo[]>([])
  const jobPods = ref<PodResourceList[]>([])

  // 前端搜索过滤关键词
  const clusterFilterKeyword = ref('')
  const workspaceFilterKeyword = ref('')

  // 过滤后的集群列表
  const filteredClusters = computed(() => {
    if (!clusterFilterKeyword.value) {
      return clusters.value
    }
    const keyword = clusterFilterKeyword.value.toLowerCase()
    return clusters.value.filter((cluster) => cluster.clusterName.toLowerCase().includes(keyword))
  })

  // 过滤后的工作空间列表
  const filteredWorkspaces = computed(() => {
    if (!workspaceFilterKeyword.value) {
      return workspaces.value
    }
    const keyword = workspaceFilterKeyword.value.toLowerCase()
    return workspaces.value.filter(
      (workspace) =>
        workspace.name.toLowerCase().includes(keyword) ||
        workspace.namespace.toLowerCase().includes(keyword)
    )
  })

  // 集群过滤方法
  const filterClusters = (keyword: string) => {
    clusterFilterKeyword.value = keyword
  }

  // 工作空间过滤方法
  const filterWorkspaces = (keyword: string) => {
    workspaceFilterKeyword.value = keyword
  }

  // 集群下拉框显示/隐藏时重置过滤关键词
  const handleClusterDropdownVisibleChange = (visible: boolean) => {
    if (!visible) {
      clusterFilterKeyword.value = ''
    }
  }

  // 工作空间下拉框显示/隐藏时重置过滤关键词
  const handleWorkspaceDropdownVisibleChange = (visible: boolean) => {
    if (!visible) {
      workspaceFilterKeyword.value = ''
    }
  }

  // 选择状态
  const selectedClusterId = ref<number | null>(null)
  const selectedWorkspaceId = ref<number | null>(null)
  const selectedCluster = ref<ProjectCluster | null>(null)
  const selectedWorkspace = ref<ProjectWorkspace | null>(null)
  const selectedJob = ref<JobInfo | null>(null)
  const selectedPod = ref<PodResourceList | null>(null)

  // 初始化标志
  const isInitializing = ref(false)
  const hasCompletedInitialLoad = ref(false)

  // 加载状态
  const loadingClusters = ref(false)
  const loadingWorkspaces = ref(false)
  const loading = ref(false)
  const podLoading = ref(false)
  const showSearchBar = ref(false)
  const showPodSearchBar = ref(false)
  const tableRef = ref()
  const podTableRef = ref()

  // 对话框状态
  const containerDialogVisible = ref(false)
  const containerDialogLoading = ref(false)
  const currentContainerData = ref<ContainerInfoList | null>(null)
  const logDialogVisible = ref(false)
  const ephemeralDialogVisible = ref(false)
  const injectingEphemeral = ref(false)

  // YAML 对话框
  const yamlDialog = reactive({
    visible: false,
    title: '',
    content: '',
    filename: ''
  })

  // Loading 状态
  const deleteLoadingMap = ref<Record<string, boolean>>({})
  const evictLoadingMap = ref<Record<string, boolean>>({})
  const deletePodLoadingMap = ref<Record<string, boolean>>({})

  // 自动刷新
  const autoRefreshEnabled = ref(false)
  const autoRefreshInterval = ref(0)
  const autoRefreshTimer = ref<NodeJS.Timeout | null>(null)
  const countdown = ref(0)
  const countdownTimer = ref<NodeJS.Timeout | null>(null)

  // 搜索表单
  let searchForm = reactive({
    search: '',
    status: '',
    sortBy: 'creationTimestamp',
    sortDesc: true
  })

  // Pod 搜索表单
  let podSearchForm = reactive({
    name: '',
    status: ''
  })
  // 分页
  const pagination = reactive({
    current: 1,
    size: 20,
    total: 0
  })

  // 虚拟的 version 和 application 对象（用于兼容子组件）
  const dummyVersion = computed<OnecProjectVersion>(
    () =>
      ({
        id: selectedWorkspace.value?.id || 0,
        name: selectedWorkspace.value?.name || ''
      }) as OnecProjectVersion
  )

  const dummyApplication = computed<OnecProjectApplication>(
    () =>
      ({
        id: 0,
        nameEn: 'Job Manager',
        nameCn: 'Job 管理',
        resourceType: 'job'
      }) as OnecProjectApplication
  )

  // 刷新按钮文本
  const refreshButtonText = computed(() => {
    if (!autoRefreshEnabled.value) {
      return '刷新'
    }
    return `刷新 (${countdown.value}s)`
  })

  // 过滤后的 Job Pods
  const filteredJobPods = computed(() => {
    let result = jobPods.value
    if (podSearchForm.name) {
      result = result.filter((pod) =>
        pod.name.toLowerCase().includes(podSearchForm.name.toLowerCase())
      )
    }
    if (podSearchForm.status) {
      result = result.filter((pod) => pod.status === podSearchForm.status)
    }
    return result
  })

  // 搜索表单配置
  const searchFormItems = computed(() => [
    {
      label: 'Job名称',
      key: 'search',
      type: 'input',
      props: { clearable: true, placeholder: '请输入Job名称' }
    },
    {
      label: '状态',
      key: 'status',
      type: 'select',
      props: {
        clearable: true,
        placeholder: '请选择状态',
        options: [
          { label: 'Running', value: 'Running' },
          { label: 'Completed', value: 'Completed' },
          { label: 'Failed', value: 'Failed' },
          { label: 'Suspended', value: 'Suspended' }
        ]
      }
    },
    {
      label: '排序字段',
      key: 'sortBy',
      type: 'select',
      props: {
        placeholder: '选择排序字段',
        options: [
          { label: '名称', value: 'name' },
          { label: '创建时间', value: 'creationTimestamp' },
          { label: '状态', value: 'status' }
        ]
      }
    },
    {
      label: '降序',
      key: 'sortDesc',
      type: 'switch',
      props: {}
    }
  ])

  // Pod 搜索表单配置
  const podSearchFormItems = computed(() => [
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
        options: [
          { label: 'Running', value: 'Running' },
          { label: 'Pending', value: 'Pending' },
          { label: 'Succeeded', value: 'Succeeded' },
          { label: 'Failed', value: 'Failed' },
          { label: 'Unknown', value: 'Unknown' }
        ]
      }
    }
  ])

  // 获取状态图标
  const getStatusIcon = (status: string) => {
    const iconMap: Record<string, any> = {
      Running: Clock,
      Completed: CheckCircle,
      Failed: XCircle,
      Suspended: AlertCircle,
      Pending: Clock,
      Succeeded: CheckCircle,
      Unknown: AlertTriangle
    }
    return iconMap[status] || AlertCircle
  }

  // 获取状态类型
  const getStatusType = (status: string) => {
    const typeMap: Record<string, any> = {
      Running: 'primary',
      Completed: 'success',
      Failed: 'danger',
      Suspended: 'warning',
      Pending: 'warning',
      Succeeded: 'success',
      Unknown: 'info'
    }
    return typeMap[status] || 'info'
  }

  // 判断是否可以打开终端
  const canExecTerminal = (status: string) => {
    return status === 'Running'
  }

  // 格式化时间
  const formatTime = (timestamp?: number) => {
    if (!timestamp) return '-'
    return dayjs(timestamp * 1000).format('YYYY-MM-DD HH:mm:ss')
  }

  // 打开 YAML 对话框
  const openYamlDialog = (title: string, content: string, filename: string) => {
    yamlDialog.title = title
    yamlDialog.content = content
    yamlDialog.filename = filename
    yamlDialog.visible = true
  }

  // Job 表格列配置
  const createJobTableColumns = () => {
    const allColumns = [
      {
        type: 'globalIndex',
        label: '序号',
        width: 80,
        align: 'center' as const,
        visible: true
      },
      {
        prop: 'name',
        label: 'Job名称',
        minWidth: 200,
        showOverflowTooltip: true,
        visible: true,
        formatter: (row: JobInfo): VNode => {
          return h(
            'span',
            {
              style: {
                color: '#409eff',
                cursor: 'pointer',
                textDecoration: 'underline'
              },
              onClick: () => handleViewJobPods(row)
            },
            row.name
          )
        }
      },
      {
        prop: 'namespace',
        label: '命名空间',
        width: 150,
        showOverflowTooltip: true,
        visible: true
      },
      {
        prop: 'status',
        label: '状态',
        width: 120,
        visible: true,
        formatter: (row: JobInfo): VNode => {
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
        prop: 'completions',
        label: '期望完成',
        width: 100,
        align: 'center' as const,
        visible: true
      },
      {
        prop: 'parallelism',
        label: '并行度',
        width: 90,
        align: 'center' as const,
        visible: true
      },
      {
        prop: 'succeeded',
        label: '成功',
        width: 80,
        align: 'center' as const,
        visible: true,
        formatter: (row: JobInfo): VNode => {
          return h('span', { style: 'color: #67c23a; font-weight: 500' }, row.succeeded)
        }
      },
      {
        prop: 'failed',
        label: '失败',
        width: 80,
        align: 'center' as const,
        visible: true,
        formatter: (row: JobInfo): VNode => {
          return h('span', { style: 'color: #f56c6c; font-weight: 500' }, row.failed)
        }
      },
      {
        prop: 'active',
        label: '运行中',
        width: 90,
        align: 'center' as const,
        visible: true,
        formatter: (row: JobInfo): VNode => {
          return h('span', { style: 'color: #409eff; font-weight: 500' }, row.active)
        }
      },
      {
        prop: 'duration',
        label: '持续时间',
        width: 120,
        visible: true
      },
      {
        prop: 'startTime',
        label: '开始时间',
        width: 180,
        visible: true,
        formatter: (row: JobInfo): VNode => h('span', formatTime(row.startTime))
      },
      {
        prop: 'creationTimestamp',
        label: '创建时间',
        width: 180,
        visible: true,
        formatter: (row: JobInfo): VNode => h('span', formatTime(row.creationTimestamp))
      },
      {
        prop: 'operation',
        label: '操作',
        width: 120,
        align: 'center' as const,
        fixed: 'right' as const,
        visible: true,
        formatter: (row: JobInfo): VNode => {
          const isDeleting = deleteLoadingMap.value[row.name] || false

          return h(
            'div',
            {
              style: {
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
                gap: '4px'
              }
            },
            [
              h(ArtButtonMore, {
                trigger: h(
                  ElButton,
                  {
                    size: 'small',
                    text: true,
                    disabled: isDeleting
                  },
                  () => h(MoreVertical, { size: 14 })
                ),
                list: [
                  {
                    key: 'pods',
                    label: '查看 Pods',
                    icon: 'lucide:list',
                    color: '#409eff',
                    disabled: isDeleting
                  },
                  {
                    key: 'yaml',
                    label: 'YAML',
                    icon: 'lucide:file-text',
                    color: '#67c23a',
                    disabled: isDeleting
                  },
                  {
                    key: 'detail',
                    label: '详情',
                    icon: 'lucide:info',
                    color: '#409eff',
                    disabled: isDeleting
                  },
                  {
                    key: 'delete',
                    label: isDeleting ? '删除中...' : '删除',
                    icon: 'lucide:trash-2',
                    color: '#f56c6c',
                    disabled: isDeleting
                  }
                ] as ButtonMoreItem[],
                onClick: (item: ButtonMoreItem) => {
                  if (!isDeleting) {
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

  const { columns } = createJobTableColumns()

  // Pod 表格列配置
  const createPodTableColumns = () => {
    const allColumns = [
      {
        type: 'globalIndex',
        label: '序号',
        width: 80,
        align: 'center' as const,
        visible: true
      },
      {
        prop: 'name',
        label: 'Pod名称',
        minWidth: 200,
        showOverflowTooltip: true,
        visible: true
      },
      {
        prop: 'status',
        label: '状态',
        width: 120,
        visible: true,
        formatter: (row: PodResourceList): VNode => {
          const Icon = getStatusIcon(row.status)
          return h(
            ElTag,
            {
              type: getStatusType(row.status),
              size: 'small'
            },
            () => [h(Icon, { size: 12, style: 'margin-right: 4px' }), row.status]
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
        width: 180,
        visible: true,
        formatter: (row: PodResourceList): VNode => h('span', formatTime(row.creationTime))
      },
      {
        prop: 'operation',
        label: '操作',
        width: 240,
        align: 'center' as const,
        fixed: 'right' as const,
        visible: true,
        formatter: (row: PodResourceList): VNode => {
          const isEvicting = evictLoadingMap.value[row.name] || false
          const isDeleting = deletePodLoadingMap.value[row.name] || false
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
              h(
                'div',
                {
                  style: {
                    display: 'flex',
                    gap: '2px',
                    marginRight: '0px'
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
                      onClick: () => handleViewLog(row)
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
                      onClick: () => handleTerminal(row)
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
                      onClick: () => handleFileManager(row)
                    },
                    () => '文件'
                  )
                ]
              ),
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
                    key: 'container',
                    label: '容器列表',
                    icon: Layers,
                    color: '#409eff',
                    disabled: isAnyLoading
                  },
                  {
                    key: 'yaml',
                    label: 'YAML',
                    icon: FileText,
                    color: '#67c23a',
                    disabled: isAnyLoading
                  },
                  {
                    key: 'detail',
                    label: '详情',
                    icon: Info,
                    color: '#409eff',
                    disabled: isAnyLoading
                  },
                  {
                    key: 'inject',
                    label: '注入临时容器',
                    icon: Package,
                    color: '#9333ea',
                    disabled: isAnyLoading
                  },
                  {
                    key: 'evict',
                    label: isEvicting ? '驱逐中...' : '驱逐',
                    icon: LogOut,
                    color: '#e6a23c',
                    disabled: isAnyLoading
                  },
                  {
                    key: 'delete',
                    label: isDeleting ? '删除中...' : '删除',
                    icon: Trash2,
                    color: '#f56c6c',
                    disabled: isAnyLoading
                  }
                ] as ButtonMoreItem[],
                onClick: (item: ButtonMoreItem) => {
                  if (!isAnyLoading) {
                    handlePodButtonMoreClick(item, row)
                  }
                }
              })
            ]
          )
        }
      }
    ]

    const podColumns = ref(allColumns)

    return {
      podColumns
    }
  }

  const { podColumns } = createPodTableColumns()

  // 返回 Job 列表
  const handleBackToJobList = () => {
    viewMode.value = 'list'
    selectedJob.value = null
    jobPods.value = []
    stopAutoRefresh()
  }

  // 查看 Job 的 Pods（进入 Pod 视图）
  const handleViewJobPods = (job: JobInfo) => {
    selectedJob.value = job
    viewMode.value = 'job-pods'
    loadJobPods()
  }

  // 更多操作点击
  const handleButtonMoreClick = (item: ButtonMoreItem, row: JobInfo) => {
    switch (item.key) {
      case 'pods':
        handleViewJobPods(row)
        break
      case 'yaml':
        handleViewYaml(row)
        break
      case 'detail':
        handleViewDetail(row)
        break
      case 'delete':
        handleDelete(row)
        break
    }
  }

  // Pod 更多操作点击
  const handlePodButtonMoreClick = (item: ButtonMoreItem, row: PodResourceList) => {
    switch (item.key) {
      case 'container':
        handleViewContainers(row)
        break
      case 'yaml':
        handleViewPodYaml(row)
        break
      case 'detail':
        handleViewPodDetail(row)
        break
      case 'inject':
        handleInjectEphemeral(row)
        break
      case 'evict':
        handleEvictPod(row)
        break
      case 'delete':
        handleDeletePod(row)
        break
    }
  }

  // 查看容器信息
  const handleViewContainers = async (pod: PodResourceList) => {
    if (!selectedWorkspace.value) return

    selectedPod.value = pod
    containerDialogVisible.value = true
    containerDialogLoading.value = true
    currentContainerData.value = null

    try {
      const response = await getPodContainerListApi(selectedWorkspace.value.id, {
        podName: pod.name
      })
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
  const handleViewPodYaml = async (pod: PodResourceList) => {
    if (!selectedWorkspace.value) return

    selectedPod.value = pod
    openYamlDialog('Pod YAML', '# 加载中...', `${pod.name}.yaml`)

    try {
      const response = await getPodYamlApi(selectedWorkspace.value.id, { podName: pod.name })
      yamlDialog.content = response || '# 获取失败'
    } catch (error) {
      console.error('获取YAML失败:', error)
      yamlDialog.content = '# 获取失败'
    }
  }

  // 查看 Pod 详情
  const handleViewPodDetail = async (pod: PodResourceList) => {
    if (!selectedWorkspace.value) return

    selectedPod.value = pod
    openYamlDialog('Pod 详情', '# 加载中...', `${pod.name}-detail.yaml`)

    try {
      const response = await getPodDetailApi(selectedWorkspace.value.id, { podName: pod.name })
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

    if (!selectedWorkspace.value?.id || !pod.name) {
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
          workloadId: String(selectedWorkspace.value.id),
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

    if (!selectedWorkspace.value?.id || !pod.name) {
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
          workspaceId: String(selectedWorkspace.value.id),
          podName: pod.name
        }
      })

      window.open(routeData.href, '_blank', 'noopener,noreferrer')
      ElMessage.success({ message: '文件管理器已在新标签页打开', duration: 2000 })
    } catch (error) {
      console.error('打开文件管理器失败:', error)
    }
  }

  // 注入临时容器
  const handleInjectEphemeral = (pod: PodResourceList) => {
    selectedPod.value = pod
    ephemeralDialogVisible.value = true
  }

  // 处理注入确认
  const handleInjectConfirm = async (formData: any) => {
    if (!selectedPod.value || !selectedWorkspace.value) return

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

      await injectEphemeralContainerApi(selectedWorkspace.value.id, payload)
      ElMessage.success('临时容器注入成功')
      ephemeralDialogVisible.value = false
      loadJobPods()
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

  // 驱逐 Pod
  const handleEvictPod = async (pod: PodResourceList) => {
    if (!selectedWorkspace.value) return

    try {
      await ElMessageBox.confirm(`确定要驱逐Pod "${pod.name}" 吗？`, '驱逐确认', {
        type: 'warning',
        confirmButtonText: '确定',
        cancelButtonText: '取消'
      })

      evictLoadingMap.value[pod.name] = true
      await evictPodApi(selectedWorkspace.value.id, { podName: pod.name })
      ElMessage.success('驱逐成功')
      loadJobPods()
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('驱逐失败:', error)
      }
    } finally {
      evictLoadingMap.value[pod.name] = false
    }
  }

  // 删除 Pod
  const handleDeletePod = async (pod: PodResourceList) => {
    if (!selectedWorkspace.value) return

    try {
      await ElMessageBox.confirm(`确定要删除Pod "${pod.name}" 吗？`, '删除确认', {
        type: 'warning',
        confirmButtonText: '确定',
        cancelButtonText: '取消'
      })

      deletePodLoadingMap.value[pod.name] = true
      await deletePodApi(selectedWorkspace.value.id, { podName: pod.name })
      ElMessage.success('删除成功')
      loadJobPods()
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('删除失败:', error)
      }
    } finally {
      deletePodLoadingMap.value[pod.name] = false
    }
  }

  // 查看 Job YAML
  const handleViewYaml = async (job: JobInfo) => {
    if (!selectedWorkspace.value) return

    selectedJob.value = job
    openYamlDialog('Job YAML', '# 加载中...', `${job.name}.yaml`)

    try {
      const response = await getCronJobJobDetailsApi(selectedWorkspace.value.id, job.name)
      yamlDialog.content = response || '# 获取失败'
    } catch (error) {
      console.error('获取YAML失败:', error)
      yamlDialog.content = '# 获取失败'
    }
  }

  // 查看 Job 详情
  const handleViewDetail = async (job: JobInfo) => {
    if (!selectedWorkspace.value) return

    selectedJob.value = job
    openYamlDialog('Job 详情', '# 加载中...', `${job.name}-detail.yaml`)

    try {
      const response = await getCronJobJobDetailsApi(selectedWorkspace.value.id, job.name)
      yamlDialog.content = response || '# 获取失败'
    } catch (error) {
      console.error('获取Job详情失败:', error)
      yamlDialog.content = '# 获取失败'
    }
  }

  // 删除 Job
  const handleDelete = async (job: JobInfo) => {
    if (!selectedWorkspace.value) return

    try {
      await ElMessageBox.confirm(`确定要删除Job "${job.name}" 吗？`, '删除确认', {
        type: 'warning',
        confirmButtonText: '确定',
        cancelButtonText: '取消'
      })

      deleteLoadingMap.value[job.name] = true
      await deleteCronJobJobApi(selectedWorkspace.value.id, job.name)
      ElMessage.success('删除成功')
      loadJobs(true)
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('删除失败:', error)
      }
    } finally {
      deleteLoadingMap.value[job.name] = false
    }
  }

  // 加载集群列表
  const loadClusters = async (savedClusterId?: number | null) => {
    if (!selectedProject.value) {
      clusters.value = []
      return
    }

    if (loadingClusters.value) {
      return
    }

    loadingClusters.value = true

    try {
      const response = await searchProjectClusterApi({
        projectId: selectedProject.value.id
      })
      clusters.value = response || []

      // 如果提供了 savedClusterId，尝试恢复选择
      if (savedClusterId !== undefined && savedClusterId !== null) {
        const cluster = clusters.value.find((c) => c.id === savedClusterId)
        if (cluster) {
          selectedClusterId.value = savedClusterId
          selectedCluster.value = cluster
        } else {
          // 找不到保存的集群，清除存储
          saveToStorage(STORAGE_KEY_CLUSTER, null)
          saveToStorage(STORAGE_KEY_WORKSPACE, null)
        }
      }
    } catch (error) {
      console.error('加载集群失败:', error)
      clusters.value = []
    } finally {
      loadingClusters.value = false
    }
  }

  // 加载工作空间列表
  const loadWorkspaces = async (clusterId: number, savedWorkspaceId?: number | null) => {
    if (!clusterId) {
      workspaces.value = []
      return
    }

    if (loadingWorkspaces.value) {
      return
    }

    loadingWorkspaces.value = true

    try {
      const response = await searchProjectWorkspaceApi({
        projectClusterId: clusterId
      })
      workspaces.value = response || []

      // 如果提供了 savedWorkspaceId，尝试恢复选择
      if (savedWorkspaceId !== undefined && savedWorkspaceId !== null) {
        const workspace = workspaces.value.find((w) => w.id === savedWorkspaceId)
        if (workspace) {
          selectedWorkspaceId.value = savedWorkspaceId
          selectedWorkspace.value = workspace
        } else {
          // 找不到保存的工作空间，清除存储
          saveToStorage(STORAGE_KEY_WORKSPACE, null)
        }
      }
    } catch (error) {
      console.error('加载工作空间失败:', error)
      workspaces.value = []
    } finally {
      loadingWorkspaces.value = false
    }
  }

  // 加载Job列表
  const loadJobs = async (silent = false) => {
    if (!selectedWorkspace.value) {
      tableData.value = []
      return
    }

    if (!silent) {
      loading.value = true
    }

    try {
      const response = await listJobsApi(selectedWorkspace.value.id, {
        page: pagination.current,
        pageSize: pagination.size,
        search: searchForm.search || undefined,
        labels: undefined,
        sortBy: searchForm.sortBy || undefined,
        sortDesc: searchForm.sortDesc
      })

      tableData.value = response.items || []
      pagination.total = response.total || 0
    } catch (error) {
      console.error('加载Job列表失败:', error)
      tableData.value = []
      pagination.total = 0
    } finally {
      if (!silent) {
        loading.value = false
      }
    }
  }

  // 加载 Job 的 Pods
  const loadJobPods = async () => {
    if (!selectedWorkspace.value || !selectedJob.value) {
      jobPods.value = []
      return
    }

    podLoading.value = true
    try {
      const response = await listJobPodsApi(selectedWorkspace.value.id, {
        jobName: selectedJob.value.name
      })
      jobPods.value = response || []
    } catch (error) {
      console.error('加载 Job Pod 列表失败:', error)
      jobPods.value = []
    } finally {
      podLoading.value = false
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
      if (viewMode.value === 'list') {
        loadJobs(true)
      } else if (viewMode.value === 'job-pods') {
        loadJobPods()
      }
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

  // 清除工作空间相关状态
  const clearWorkspaceState = () => {
    selectedWorkspaceId.value = null
    selectedWorkspace.value = null
    workspaces.value = []
    tableData.value = []
    pagination.total = 0
    handleBackToJobList()
    saveToStorage(STORAGE_KEY_WORKSPACE, null)
  }

  // 集群变化
  const handleClusterChange = async (clusterId: number | null) => {
    // 先清除工作空间相关状态
    clearWorkspaceState()

    if (clusterId) {
      saveToStorage(STORAGE_KEY_CLUSTER, clusterId)
      const cluster = clusters.value.find((c) => c.id === clusterId)
      if (cluster) {
        selectedCluster.value = cluster
        await loadWorkspaces(clusterId)
      }
    } else {
      selectedCluster.value = null
      saveToStorage(STORAGE_KEY_CLUSTER, null)
    }
  }

  // 清除集群
  const handleClusterClear = () => {
    selectedCluster.value = null
    clearWorkspaceState()
    saveToStorage(STORAGE_KEY_CLUSTER, null)
  }

  // 工作空间变化
  const handleWorkspaceChange = async (workspaceId: number | null) => {
    if (workspaceId) {
      const workspace = workspaces.value.find((w) => w.id === workspaceId)
      if (workspace) {
        saveToStorage(STORAGE_KEY_WORKSPACE, workspaceId)
        selectedWorkspace.value = workspace
        pagination.current = 1
        handleBackToJobList()
        await loadJobs()
      } else {
        // 找不到工作空间，清除选择
        selectedWorkspaceId.value = null
        selectedWorkspace.value = null
        saveToStorage(STORAGE_KEY_WORKSPACE, null)
      }
    } else {
      selectedWorkspace.value = null
      tableData.value = []
      pagination.total = 0
      handleBackToJobList()
      saveToStorage(STORAGE_KEY_WORKSPACE, null)
    }
  }

  // 清除工作空间
  const handleWorkspaceClear = () => {
    selectedWorkspace.value = null
    tableData.value = []
    pagination.total = 0
    handleBackToJobList()
    saveToStorage(STORAGE_KEY_WORKSPACE, null)
  }

  // 搜索
  const handleSearch = () => {
    pagination.current = 1
    loadJobs()
  }

  // 重置
  const handleReset = () => {
    searchForm.search = ''
    searchForm.status = ''
    searchForm.sortBy = 'creationTimestamp'
    searchForm.sortDesc = true
    pagination.current = 1
    loadJobs()
  }

  // Pod 搜索
  const handlePodSearch = () => {
    // Pod 列表不需要重新加载，使用 computed 过滤
  }

  // Pod 重置
  const handlePodReset = () => {
    podSearchForm.name = ''
    podSearchForm.status = ''
  }

  // 刷新
  const handleRefresh = () => {
    if (viewMode.value === 'list') {
      loadJobs()
    } else if (viewMode.value === 'job-pods') {
      loadJobPods()
    }
  }

  // 分页大小变化
  const handleSizeChange = (size: number) => {
    pagination.size = size
    pagination.current = 1
    loadJobs()
  }

  // 页码变化
  const handleCurrentChange = (current: number) => {
    pagination.current = current
    loadJobs()
  }

  // 监听项目切换
  watch(selectedProject, async (newProject, oldProject) => {
    if (isInitializing.value) {
      return
    }

    if (newProject?.id !== oldProject?.id) {
      // 清除所有存储
      saveToStorage(STORAGE_KEY_CLUSTER, null)
      saveToStorage(STORAGE_KEY_WORKSPACE, null)

      // 清空所有选择状态
      selectedClusterId.value = null
      selectedWorkspaceId.value = null
      selectedCluster.value = null
      selectedWorkspace.value = null
      clusters.value = []
      workspaces.value = []
      tableData.value = []
      pagination.total = 0
      handleBackToJobList()

      if (newProject) {
        await loadClusters()
      }
    }
  })

  // 初始化逻辑
  const initPage = async () => {
    if (isInitializing.value) {
      return
    }

    isInitializing.value = true

    try {
      await projectStore.ensureInitialized()

      if (!selectedProject.value) {
        saveToStorage(STORAGE_KEY_CLUSTER, null)
        saveToStorage(STORAGE_KEY_WORKSPACE, null)
        return
      }

      // 先从 sessionStorage 读取保存的值
      const savedClusterId = loadFromStorage(STORAGE_KEY_CLUSTER)
      const savedWorkspaceId = loadFromStorage(STORAGE_KEY_WORKSPACE)

      // 加载集群列表，并尝试恢复选择
      await loadClusters(savedClusterId)

      // 如果集群恢复成功，加载工作空间列表并尝试恢复选择
      if (selectedClusterId.value) {
        await loadWorkspaces(selectedClusterId.value, savedWorkspaceId)

        // 如果工作空间恢复成功，加载 Jobs
        if (selectedWorkspaceId.value) {
          await loadJobs()
        }
      }
    } catch (error) {
      console.error('页面初始化失败:', error)
    } finally {
      setTimeout(() => {
        isInitializing.value = false
        hasCompletedInitialLoad.value = true
      }, 100)
    }
  }

  let initPromise: Promise<void> | null = null

  onMounted(async () => {
    if (!initPromise) {
      initPromise = initPage()
    }
    await initPromise
  })

  onActivated(async () => {
    if (!hasCompletedInitialLoad.value) {
      if (!initPromise) {
        initPromise = initPage()
      }
      await initPromise
      return
    }

    if (selectedWorkspace.value && viewMode.value === 'list') {
      await loadJobs(true)
    } else if (selectedWorkspace.value && viewMode.value === 'job-pods') {
      await loadJobPods()
    }
  })

  onUnmounted(() => {
    stopAutoRefresh()
  })
</script>

<style lang="scss" scoped>
  .job-manager-page {
    padding-bottom: 15px;

    .breadcrumb-selector {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 16px;
      padding: 12px 16px;
      margin-bottom: 16px;
      background: white;
      border-radius: 8px;
      border: 1px solid #e4e7ed;
      box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.03);

      .breadcrumb-content {
        flex: 1;
        display: flex;
        align-items: center;
        gap: 8px;
        min-width: 0;
        flex-wrap: wrap;

        .breadcrumb-item {
          display: flex;
          align-items: center;
          gap: 8px;

          .breadcrumb-label {
            display: flex;
            align-items: center;
            gap: 4px;
            font-size: 13px;
            color: #606266;
            font-weight: 500;
            white-space: nowrap;
          }

          .breadcrumb-select {
            min-width: 200px;
            max-width: 300px;
          }
        }

        .breadcrumb-separator {
          color: #c0c4cc;
          display: flex;
          align-items: center;
        }
      }

      .breadcrumb-actions {
        display: flex;
        gap: 8px;
        flex-shrink: 0;
      }
    }

    .cluster-option,
    .workspace-option {
      display: flex;
      justify-content: space-between;
      align-items: center;
      width: 100%;
      gap: 8px;

      .option-name {
        font-weight: 500;
        color: #303133;
        flex: 1;
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .option-meta {
        font-size: 12px;
        color: #909399;
        flex-shrink: 0;
      }
    }

    .job-info-card {
      margin-bottom: 16px;
      border: 1px solid #e4e7ed;

      .job-info-content {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 24px;

        .job-info-left {
          display: flex;
          align-items: center;
          gap: 16px;
          flex: 1;

          .job-icon {
            width: 48px;
            height: 48px;
            display: flex;
            align-items: center;
            justify-content: center;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border-radius: 12px;
            flex-shrink: 0;
          }

          .job-details {
            flex: 1;
            min-width: 0;

            .job-name {
              font-size: 18px;
              font-weight: 600;
              color: #303133;
              margin-bottom: 8px;
              overflow: hidden;
              text-overflow: ellipsis;
              white-space: nowrap;
            }

            .job-meta {
              display: flex;
              align-items: center;
              gap: 12px;
              flex-wrap: wrap;

              .job-namespace,
              .job-time {
                font-size: 13px;
                color: #909399;
              }
            }
          }
        }

        .job-info-right {
          .job-stats {
            display: flex;
            gap: 24px;

            .stat-item {
              text-align: center;

              .stat-value {
                font-size: 24px;
                font-weight: 600;
                color: #606266;
                margin-bottom: 4px;
              }

              .stat-label {
                font-size: 12px;
                color: #909399;
              }

              &.success .stat-value {
                color: #67c23a;
              }

              &.danger .stat-value {
                color: #f56c6c;
              }

              &.primary .stat-value {
                color: #409eff;
              }
            }
          }
        }
      }
    }

    .empty-container {
      min-height: 400px;
      display: flex;
      align-items: center;
      justify-content: center;
      background: white;
      border-radius: 8px;
      box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
    }

    .yaml-dialog {
      :deep(.el-dialog__body) {
        padding: 20px;
      }
    }
  }
</style>

<style lang="scss">
  .job-cluster-dropdown {
    min-width: 400px !important;

    .el-select-dropdown__wrap {
      min-height: 300px !important;
      max-height: 300px !important;
    }

    .el-select-dropdown__item {
      height: auto !important;
      line-height: normal !important;
      padding: 0 !important;
      margin: 0 !important;

      &.hover {
        background-color: transparent !important;

        .cluster-option {
          background-color: #f5f7fa;
        }
      }

      &.selected {
        background-color: transparent !important;
        font-weight: normal !important;

        .cluster-option {
          background-color: #ecf5ff;

          .option-name {
            color: #409eff;
            font-weight: 500;
          }
        }
      }
    }

    .cluster-option {
      display: flex !important;
      align-items: center !important;
      justify-content: space-between !important;
      width: 100%;
      gap: 12px;
      padding: 10px 16px !important;
      border-radius: 4px;
      transition: background-color 0.2s;
      height: auto !important;
      box-sizing: border-box;

      .option-name {
        font-weight: 500;
        color: #303133;
        flex: 1;
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-size: 14px;
        line-height: 1;
        margin: 0 !important;
        padding: 0 !important;
        vertical-align: middle;
      }

      .option-meta {
        font-size: 12px;
        color: #909399;
        flex-shrink: 0;
        white-space: nowrap;
        line-height: 1;
        margin: 0 !important;
        padding: 0 !important;
        vertical-align: middle;
      }
    }
  }

  .job-workspace-dropdown {
    min-width: 400px !important;

    .el-select-dropdown__wrap {
      min-height: 300px !important;
      max-height: 300px !important;
    }

    .el-select-dropdown__item {
      height: auto !important;
      line-height: normal !important;
      padding: 0 !important;
      margin: 0 !important;

      &.hover {
        background-color: transparent !important;

        .workspace-option {
          background-color: #f5f7fa;
        }
      }

      &.selected {
        background-color: transparent !important;
        font-weight: normal !important;

        .workspace-option {
          background-color: #ecf5ff;

          .option-name {
            color: #409eff;
            font-weight: 500;
          }
        }
      }
    }

    .workspace-option {
      display: flex !important;
      align-items: center !important;
      justify-content: space-between !important;
      width: 100%;
      gap: 12px;
      padding: 10px 16px !important;
      border-radius: 4px;
      transition: background-color 0.2s;
      height: auto !important;
      box-sizing: border-box;

      .option-name {
        font-weight: 500;
        color: #303133;
        flex: 1;
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-size: 14px;
        line-height: 1;
        margin: 0 !important;
        padding: 0 !important;
        vertical-align: middle;
      }

      :deep(.el-tag) {
        margin: 0 !important;
        vertical-align: middle !important;
      }
    }
  }
</style>
