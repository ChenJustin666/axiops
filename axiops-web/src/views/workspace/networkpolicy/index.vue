<template>
  <div class="networkpolicy-page art-full-height">
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
            size="default"
            :disabled="!selectedProject"
            :loading="loadingClusters"
            filterable
            @change="handleClusterChange"
            @clear="handleClusterClear"
            popper-class="np-cluster-dropdown"
            class="breadcrumb-select"
          >
            <ElOption
              v-for="cluster in clusters"
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
            size="default"
            filterable
            :disabled="!selectedClusterId"
            :loading="loadingWorkspaces"
            @change="handleWorkspaceChange"
            @clear="handleWorkspaceClear"
            popper-class="np-workspace-dropdown"
            class="breadcrumb-select"
          >
            <ElOption
              v-for="workspace in workspaces"
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
      </div>

      <!-- 操作按钮 -->
      <div class="breadcrumb-actions">
        <ElTooltip content="刷新数据" placement="bottom">
          <ElButton
            :icon="RefreshCw"
            circle
            size="default"
            :disabled="!isWorkspaceReady"
            @click="handleRefresh"
          />
        </ElTooltip>
      </div>
    </div>

    <!-- 主要内容区 -->
    <ElCard class="main-content-card" shadow="never">
      <!-- 初始化加载状态 -->
      <div v-if="isInitializing" class="loading-state">
        <ElSkeleton :rows="8" animated />
      </div>

      <!-- 未完成选择时显示引导提示 -->
      <div v-else-if="!isWorkspaceReady" class="empty-state">
        <SelectionGuideState
          level="project-cluster-workspace"
          :project="selectedProject"
          :cluster="selectedCluster"
          :workspace="selectedWorkspace"
          resource-name="NetworkPolicy"
        />
      </div>

      <!-- 表格内容 -->
      <div v-else class="table-container">
        <!-- 搜索栏 -->
        <NetworkPolicySearch
          v-show="showSearchBar"
          v-model="searchForm"
          @search="handleSearch"
          @reset="resetSearchParams"
        />

        <ElCard
          class="art-table-card"
          shadow="never"
          :style="{ 'margin-top': showSearchBar ? '12px' : '0' }"
        >
          <!-- 表格头部 -->
          <ArtTableHeader
            v-model:columns="columns"
            v-model:showSearchBar="showSearchBar"
            :loading="loading"
            :showZebra="true"
            :showBorder="true"
            :showHeaderBackground="true"
            :fullClass="'art-page-view'"
            :layout="'search,refresh,size,fullscreen,columns,settings'"
            @refresh="loadData"
          >
            <template #left>
              <ElSpace wrap>
                <ElButton type="primary" @click="handleCreate">新 增</ElButton>
              </ElSpace>
            </template>
          </ArtTableHeader>

          <!-- 数据表格 -->
          <ArtTable
            :loading="loading"
            :data="data"
            :columns="columns"
            :pagination="pagination"
            @pagination:size-change="handleSizeChange"
            @pagination:current-change="handleCurrentChange"
          />
        </ElCard>
      </div>
    </ElCard>

    <!-- 新增/编辑弹窗 -->
    <NetworkPolicyDialog
      v-model="dialogVisible"
      :dialog-type="dialogType"
      :network-policy-data="currentNetworkPolicy"
      :workspace="selectedWorkspace"
      :cluster="selectedCluster"
      @success="loadData"
    />

    <!-- 详情查看弹窗 -->
    <NetworkPolicyDetailDialog
      v-model="detailVisible"
      :network-policy-name="currentNetworkPolicyName"
      :workspace="selectedWorkspace"
      :cluster="selectedCluster"
    />
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, watch, onMounted } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { RefreshCw, Server, Box, ChevronRight } from 'lucide-vue-next'
  import { useProjectStore } from '../../../store/modules/project'
  import { searchProjectClusterApi, searchProjectWorkspaceApi } from '../../../api/manager/project'
  import type { ProjectCluster, ProjectWorkspace } from '../../../api/manager/project'
  import {
    getNetworkPolicyListApi,
    deleteNetworkPolicyApi,
    getNetworkPolicyYamlApi,
    type NetworkPolicyListItem,
    type NetworkPolicyListRequest,
    type NetworkPolicyNameRequest
  } from '../../../api/workload/networkpolicy'
  import { useNetworkPolicyTableConfig } from './modules/table-config'
  import NetworkPolicySearch from './modules/NetworkPolicySearch.vue'
  import NetworkPolicyDialog from './modules/NetworkPolicyDialog.vue'
  import NetworkPolicyDetailDialog from './modules/NetworkPolicyDetailDialog.vue'
  import SelectionGuideState from '../../../components/SelectionGuideState/index.vue'

  defineOptions({ name: 'NetworkPolicyManagement' })

  // LocalStorage 键名
  const STORAGE_KEY_CLUSTER = 'networkpolicy-selected-cluster'
  const STORAGE_KEY_WORKSPACE = 'networkpolicy-selected-workspace'

  const projectStore = useProjectStore()

  const selectedProject = computed(() => projectStore.selectedProject)

  // 集群和工作空间
  const clusters = ref<ProjectCluster[]>([])
  const workspaces = ref<ProjectWorkspace[]>([])
  const selectedClusterId = ref<number | null>(null)
  const selectedWorkspaceId = ref<number | null>(null)

  const selectedCluster = computed(
    () => clusters.value.find((c) => c.id === selectedClusterId.value) || null
  )
  const selectedWorkspace = computed(
    () => workspaces.value.find((w) => w.id === selectedWorkspaceId.value) || null
  )

  // 获取 clusterUuid
  const clusterUuid = computed(() => selectedCluster.value?.clusterUuid || '')

  const loadingClusters = ref(false)
  const loadingWorkspaces = ref(false)
  const isInitializing = ref(false)
  const hasCompletedInitialLoad = ref(false)

  const isWorkspaceReady = computed(() => {
    return !!(
      selectedClusterId.value &&
      selectedWorkspaceId.value &&
      selectedCluster.value &&
      selectedWorkspace.value
    )
  })

  // 表格相关
  const { loading, showSearchBar, deleteLoadingMap, downloadLoadingMap, createTableColumns } =
    useNetworkPolicyTableConfig()

  const data = ref<NetworkPolicyListItem[]>([])
  const searchForm = ref({
    search: '',
    labelSelector: ''
  })
  const searchParams = ref<any>({})
  const pagination = ref({
    current: 1,
    size: 20,
    total: 0
  })

  // 弹窗相关
  const dialogVisible = ref(false)
  const detailVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const currentNetworkPolicy = ref<NetworkPolicyListItem | undefined>(undefined)
  const currentNetworkPolicyName = ref('')

  // 表格列配置
  const handleView = (row: NetworkPolicyListItem) => {
    currentNetworkPolicyName.value = row.name
    detailVisible.value = true
  }

  const handleEdit = (row: NetworkPolicyListItem) => {
    dialogType.value = 'edit'
    currentNetworkPolicy.value = row
    dialogVisible.value = true
  }

  const handleDownloadYaml = async (row: NetworkPolicyListItem) => {
    if (!selectedCluster.value || !selectedWorkspace.value) return

    downloadLoadingMap.value[row.name] = true
    try {
      const params: NetworkPolicyNameRequest = {
        clusterUuid: selectedCluster.value.clusterUuid,
        namespace: selectedWorkspace.value.namespace,
        name: row.name
      }
      const yaml = await getNetworkPolicyYamlApi(params)

      const blob = new Blob([yaml], { type: 'text/yaml' })
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `${row.name}.yaml`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)

      ElMessage.success('下载成功')
    } finally {
      downloadLoadingMap.value[row.name] = false
    }
  }

  const handleDelete = async (row: NetworkPolicyListItem) => {
    if (!selectedCluster.value || !selectedWorkspace.value) return

    try {
      await ElMessageBox.confirm(
        `确定要删除 NetworkPolicy "${row.name}" 吗？此操作不可恢复！`,
        '删除确认',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      deleteLoadingMap.value[row.name] = true
      const params: NetworkPolicyNameRequest = {
        clusterUuid: selectedCluster.value.clusterUuid,
        namespace: selectedWorkspace.value.namespace,
        name: row.name
      }
      await deleteNetworkPolicyApi(params)

      ElMessage.success('删除成功')
      loadData()
    } catch (error: any) {
      // 用户取消操作不做处理
      if (error === 'cancel') return
    } finally {
      deleteLoadingMap.value[row.name] = false
    }
  }

  const { columns } = createTableColumns({
    handleView,
    handleEdit,
    handleDownloadYaml,
    handleDelete
  })

  // LocalStorage 操作
  function loadFromStorage(key: string): number | string | null {
    try {
      const value = sessionStorage.getItem(key)
      return value || null
    } catch {
      return null
    }
  }

  function saveToStorage(key: string, value: number | string | null) {
    try {
      if (value !== null && value !== undefined) {
        sessionStorage.setItem(key, String(value))
      } else {
        sessionStorage.removeItem(key)
      }
    } catch {
      // sessionStorage 操作失败时静默处理
    }
  }

  function clearStorage() {
    try {
      sessionStorage.removeItem(STORAGE_KEY_CLUSTER)
      sessionStorage.removeItem(STORAGE_KEY_WORKSPACE)
    } catch {
      // 静默处理
    }
  }

  // 集群操作
  const loadClusters = async () => {
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

      if (isInitializing.value) {
        const savedClusterId = loadFromStorage(STORAGE_KEY_CLUSTER)
        if (savedClusterId) {
          const cluster = clusters.value.find((c) => c.id === Number(savedClusterId))
          if (cluster) {
            selectedClusterId.value = cluster.id
          } else {
            // 缓存的集群不存在，清空选择
            saveToStorage(STORAGE_KEY_CLUSTER, null)
            saveToStorage(STORAGE_KEY_WORKSPACE, null)
          }
        }
      }
    } catch {
      clusters.value = []
    } finally {
      loadingClusters.value = false
    }
  }

  const handleClusterChange = async (clusterId: number | null) => {
    saveToStorage(STORAGE_KEY_CLUSTER, clusterId)

    if (clusterId) {
      await loadWorkspaces()
    } else {
      selectedWorkspaceId.value = null
      workspaces.value = []
      data.value = []
      pagination.value.total = 0
      saveToStorage(STORAGE_KEY_WORKSPACE, null)
    }
  }

  const handleClusterClear = () => {
    selectedClusterId.value = null
    selectedWorkspaceId.value = null
    workspaces.value = []
    data.value = []
    pagination.value.total = 0

    saveToStorage(STORAGE_KEY_CLUSTER, null)
    saveToStorage(STORAGE_KEY_WORKSPACE, null)
  }

  // 工作空间操作
  const loadWorkspaces = async () => {
    if (!selectedClusterId.value) {
      workspaces.value = []
      return
    }

    if (loadingWorkspaces.value) {
      return
    }

    loadingWorkspaces.value = true

    try {
      const response = await searchProjectWorkspaceApi({
        projectClusterId: selectedClusterId.value
      })
      workspaces.value = response || []

      if (isInitializing.value) {
        const savedWorkspaceId = loadFromStorage(STORAGE_KEY_WORKSPACE)
        if (savedWorkspaceId) {
          const workspace = workspaces.value.find((w) => w.id === Number(savedWorkspaceId))
          if (workspace) {
            selectedWorkspaceId.value = workspace.id
            // 加载 NetworkPolicy 数据
            loadData()
          } else {
            // 缓存的工作空间不存在，清空选择
            saveToStorage(STORAGE_KEY_WORKSPACE, null)
          }
        }
      }
    } catch {
      workspaces.value = []
    } finally {
      loadingWorkspaces.value = false
    }
  }

  const handleWorkspaceChange = (workspaceId: number | null) => {
    saveToStorage(STORAGE_KEY_WORKSPACE, workspaceId)

    if (workspaceId) {
      loadData()
    } else {
      data.value = []
      pagination.value.total = 0
    }
  }

  const handleWorkspaceClear = () => {
    selectedWorkspaceId.value = null
    data.value = []
    pagination.value.total = 0

    saveToStorage(STORAGE_KEY_WORKSPACE, null)
  }

  // 数据加载
  const loadData = async () => {
    if (!selectedCluster.value || !selectedWorkspace.value) {
      return
    }

    loading.value = true
    try {
      const params: NetworkPolicyListRequest = {
        clusterUuid: selectedCluster.value.clusterUuid,
        namespace: selectedWorkspace.value.namespace,
        search: searchParams.value.search,
        labelSelector: searchParams.value.labelSelector
      }

      const response = await getNetworkPolicyListApi(params)

      data.value = response.items || []
      pagination.value.total = response.total || 0
      hasCompletedInitialLoad.value = true
    } finally {
      loading.value = false
    }
  }

  // 搜索
  const handleSearch = (params: Record<string, any>) => {
    searchParams.value = params
    pagination.value.current = 1
    loadData()
  }

  const resetSearchParams = () => {
    searchParams.value = {}
    pagination.value.current = 1
    loadData()
  }

  // 分页
  const handleSizeChange = (size: number) => {
    pagination.value.size = size
    pagination.value.current = 1
    loadData()
  }

  const handleCurrentChange = (current: number) => {
    pagination.value.current = current
    loadData()
  }

  // 刷新
  const handleRefresh = () => {
    loadData()
  }

  // 新增
  const handleCreate = () => {
    if (!isWorkspaceReady.value) {
      return
    }
    dialogType.value = 'add'
    currentNetworkPolicy.value = undefined
    dialogVisible.value = true
  }

  // 监听项目变化
  watch(selectedProject, async (newProject, oldProject) => {
    if (isInitializing.value) {
      return
    }

    if (newProject?.id !== oldProject?.id) {
      clearStorage()

      selectedClusterId.value = null
      selectedWorkspaceId.value = null
      clusters.value = []
      workspaces.value = []
      data.value = []
      pagination.value.total = 0

      if (newProject) {
        await loadClusters()
      }
    }
  })

  // 初始化
  const initPage = async () => {
    if (isInitializing.value) {
      return
    }

    isInitializing.value = true

    try {
      if (!selectedProject.value) {
        clearStorage()
        return
      }

      await loadClusters()

      if (selectedClusterId.value) {
        await loadWorkspaces()
      }
    } finally {
      setTimeout(() => {
        isInitializing.value = false
        hasCompletedInitialLoad.value = true
      }, 100)
    }
  }

  onMounted(() => {
    initPage()
  })

  // 暴露清理方法供退出登录时调用
  defineExpose({
    clearStorage
  })
</script>

<style lang="scss" scoped>
  .networkpolicy-page {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: #f5f7fa;
  }

  .breadcrumb-selector {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 10px 16px;
    margin-bottom: 16px;
    background: white;
    border-radius: 4px;
    border: 1px solid #e4e7ed;

    .breadcrumb-content {
      flex: 1;
      display: flex;
      align-items: center;
      gap: 8px;
      min-width: 0;

      .breadcrumb-item {
        display: flex;
        align-items: center;
        gap: 8px;
        min-width: 0;

        .breadcrumb-label {
          display: flex;
          align-items: center;
          gap: 4px;
          font-size: 13px;
          color: #606266;
          font-weight: 500;
          white-space: nowrap;
          flex-shrink: 0;
        }

        .breadcrumb-select {
          min-width: 180px;
          max-width: 280px;

          :deep(.el-input__wrapper) {
            background-color: #f5f7fa;
            border: 1px solid #dcdfe6;
            transition: all 0.2s;

            &:hover {
              border-color: #c0c4cc;
            }
          }

          :deep(.el-input__inner) {
            font-size: 13px;
            font-weight: 500;
          }
        }
      }

      .breadcrumb-separator {
        color: #c0c4cc;
        flex-shrink: 0;
        display: flex;
        align-items: center;
      }
    }

    .breadcrumb-actions {
      flex-shrink: 0;
    }
  }

  .main-content-card {
    flex: 1;
    overflow: hidden;

    :deep(.el-card__body) {
      height: 100%;
      padding: 0;
      display: flex;
      flex-direction: column;
    }

    .loading-state {
      padding: 40px;
    }

    .empty-state {
      flex: 1;
      display: flex;
      align-items: center;
      justify-content: center;
    }

    .table-container {
      height: 100%;
      display: flex;
      flex-direction: column;
    }
  }

  .art-table-card {
    flex: 1;
    overflow: hidden;
    display: flex;
    flex-direction: column;

    :deep(.el-card__body) {
      flex: 1;
      overflow: hidden;
      display: flex;
      flex-direction: column;
    }
  }

  @media (max-width: 1200px) {
    .breadcrumb-selector {
      .breadcrumb-content {
        .breadcrumb-item {
          .breadcrumb-select {
            min-width: 150px;
            max-width: 200px;
          }
        }
      }
    }
  }
</style>

<style lang="scss">
  /* NetworkPolicy 集群下拉菜单样式 */
  .np-cluster-dropdown {
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
    }
  }

  /* NetworkPolicy 工作空间下拉菜单样式 */
  .np-workspace-dropdown {
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
        flex-shrink: 0;
      }
    }
  }
</style>
