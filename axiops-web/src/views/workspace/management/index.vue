<template>
  <div class="workspace-container">
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
            size="default"
            :disabled="!selectedProject"
            :loading="loadingClusters"
            @change="handleClusterChange"
            @clear="handleClusterClear"
            popper-class="workspace-mgmt-cluster-dropdown"
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
      </div>

      <!-- 操作按钮 -->
      <div class="breadcrumb-actions">
        <ElTooltip content="创建工作空间" placement="bottom">
          <ElButton
            type="primary"
            :icon="Plus"
            circle
            size="default"
            :disabled="!selectedCluster"
            @click="handleCreate"
          />
        </ElTooltip>
        <ElTooltip content="刷新数据" placement="bottom">
          <ElButton
            :icon="RefreshCw"
            circle
            size="default"
            :loading="loading"
            @click="refreshData"
          />
        </ElTooltip>
      </div>
    </div>

    <!-- 工作空间内容 -->
    <template v-if="selectedCluster">
      <!-- 顶部信息栏 -->
      <ElCard class="info-bar" shadow="never">
        <div class="info-items">
          <div class="info-item">
            <span class="label">当前项目：</span>
            <span class="value">{{ selectedProject?.name }}</span>
          </div>
          <ElDivider direction="vertical" />
          <div class="info-item">
            <span class="label">集群：</span>
            <span class="value">{{ selectedCluster?.clusterName }}</span>
          </div>
          <ElDivider direction="vertical" />
          <div class="info-item">
            <span class="label">资源使用：</span>
            <div class="resource-quick-view">
              <ElTooltip content="CPU使用率" placement="top">
                <ElTag size="small" :type="cpuUsageType"> CPU: {{ cpuUsagePercent }}%</ElTag>
              </ElTooltip>
              <ElTooltip content="内存使用率" placement="top">
                <ElTag size="small" :type="memUsageType"> 内存: {{ memUsagePercent }}%</ElTag>
              </ElTooltip>
              <ElTooltip content="存储使用率" placement="top">
                <ElTag size="small" :type="storageUsageType">
                  存储: {{ storageUsagePercent }}%
                </ElTag>
              </ElTooltip>
            </div>
          </div>
        </div>
        <div class="view-switcher">
          <ElRadioGroup v-model="viewMode" @change="handleViewModeChange">
            <ElRadioButton value="card">
              <Grid :size="14" style="margin-right: 4px" />
              卡片视图
            </ElRadioButton>
            <ElRadioButton value="list">
              <List :size="14" style="margin-right: 4px" />
              列表视图
            </ElRadioButton>
          </ElRadioGroup>
        </div>
      </ElCard>

      <!-- 视图组件 -->
      <component
        :is="viewComponent"
        :workspaces="workspaces"
        :loading="loading"
        @edit="handleEdit"
        @delete="handleDelete"
        @manage="handleManage"
        @create="handleCreate"
        @refresh="handleRefresh"
      />
    </template>

    <!-- 空状态 - 使用 SelectionGuideState -->
    <div v-else class="empty-container">
      <SelectionGuideState
        level="project-cluster"
        :project="selectedProject"
        :cluster="selectedCluster"
        resource-name="工作空间"
      />
    </div>

    <!-- 创建/编辑对话框 -->
    <WorkspaceDialog
      v-if="selectedProject && selectedCluster"
      v-model="dialogVisible"
      :mode="dialogMode"
      :workspace="currentWorkspace"
      :project="selectedProject"
      :cluster="selectedCluster"
      @success="handleDialogSuccess"
    />
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, watch, onMounted, onActivated } from 'vue'
  import { useRouter } from 'vue-router'
  import { ElMessage, ElMessageBox, ElLoading } from 'element-plus'
  import { Server, RefreshCw, Plus, Grid, List } from 'lucide-vue-next'
  import { useProjectStore } from '@/store/modules/project'
  import {
    searchProjectClusterApi,
    searchProjectWorkspaceApi,
    deleteProjectWorkspaceApi,
    type ProjectCluster,
    type ProjectWorkspace
  } from '@/api'
  import CardView from './subpage/card.vue'
  import ListView from './subpage/list.vue'
  import WorkspaceDialog from './subpage/dialog/index.vue'
  import SelectionGuideState from '@/components/SelectionGuideState/index.vue'
  import { parseCpu, parseMemory } from '@/utils/resource'

  defineOptions({ name: 'WorkspaceIndex' })

  const router = useRouter()
  const projectStore = useProjectStore()

  const STORAGE_KEY_CLUSTER = 'workspace-management-selected-cluster'

  function loadFromStorage(key: string): number | null {
    try {
      const value = sessionStorage.getItem(key)
      return value ? Number(value) : null
    } catch (error) {
      console.error('加载存储失败:', error)
      return null
    }
  }

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

  // 项目相关
  const selectedProject = computed(() => projectStore.selectedProject)

  const clusters = ref<ProjectCluster[]>([])
  const selectedClusterId = ref<number | null>(null)
  const selectedCluster = ref<ProjectCluster | null>(null)
  const workspaces = ref<ProjectWorkspace[]>([])
  const loading = ref(false)
  const loadingClusters = ref(false)

  const isInitializing = ref(false)
  const hasCompletedInitialLoad = ref(false)

  // 视图模式
  const viewMode = ref<'card' | 'list'>('card')

  // 对话框状态
  const dialogVisible = ref(false)
  const dialogMode = ref<'create' | 'edit'>('create')
  const currentWorkspace = ref<ProjectWorkspace | null>(null)

  // 计算属性
  const viewComponent = computed(() => {
    return viewMode.value === 'card' ? CardView : ListView
  })

  // 资源使用率
  const cpuUsagePercent = computed(() => {
    if (!selectedCluster.value) return 0
    const capacity = parseCpu(selectedCluster.value.cpuCapacity).value
    const allocated = parseCpu(selectedCluster.value.cpuAllocated).value
    return capacity === 0 ? 0 : Math.round((allocated / capacity) * 100)
  })

  const memUsagePercent = computed(() => {
    if (!selectedCluster.value) return 0
    const capacity = parseMemory(selectedCluster.value.memCapacity).value
    const allocated = parseMemory(selectedCluster.value.memAllocated).value
    return capacity === 0 ? 0 : Math.round((allocated / capacity) * 100)
  })

  const storageUsagePercent = computed(() => {
    if (!selectedCluster.value) return 0
    const limit = parseMemory(selectedCluster.value.storageLimit).value
    const allocated = parseMemory(selectedCluster.value.storageAllocated).value
    return limit === 0 ? 0 : Math.round((allocated / limit) * 100)
  })

  const getUsageType = (percent: number) => {
    if (percent < 60) return 'success'
    if (percent < 80) return 'warning'
    return 'danger'
  }

  const cpuUsageType = computed(() => getUsageType(cpuUsagePercent.value))
  const memUsageType = computed(() => getUsageType(memUsagePercent.value))
  const storageUsageType = computed(() => getUsageType(storageUsagePercent.value))

  watch(selectedProject, async (newProject, oldProject) => {
    // 如果正在初始化，跳过
    if (isInitializing.value) {
      return
    }

    // 如果项目真的发生了变化
    if (newProject?.id !== oldProject?.id) {
      // 清除存储
      saveToStorage(STORAGE_KEY_CLUSTER, null)

      // 清空所有状态
      selectedClusterId.value = null
      selectedCluster.value = null
      workspaces.value = []
      clusters.value = []

      // 只有在项目存在时才加载集群
      if (newProject) {
        await loadClusters()

        // 如果只有一个集群，自动选择
        if (clusters.value.length === 1) {
          selectedClusterId.value = clusters.value[0].id
          selectedCluster.value = clusters.value[0]
          saveToStorage(STORAGE_KEY_CLUSTER, clusters.value[0].id)
          await loadWorkspaces()
        }
      }
    }
  })

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
        if (savedClusterId && typeof savedClusterId === 'number') {
          const cluster = clusters.value.find((c) => c.id === savedClusterId)
          if (cluster) {
            selectedClusterId.value = savedClusterId
            selectedCluster.value = cluster
          } else {
            console.warn('⚠️ 缓存的集群不存在，清空选择')
            saveToStorage(STORAGE_KEY_CLUSTER, null)
          }
        } else if (clusters.value.length === 1) {
          // 没有缓存但只有一个集群，自动选择
          selectedClusterId.value = clusters.value[0].id
          selectedCluster.value = clusters.value[0]
          saveToStorage(STORAGE_KEY_CLUSTER, clusters.value[0].id)
        }
      }

      if (clusters.value.length === 0) {
        // 没有集群时清空
        selectedClusterId.value = null
        selectedCluster.value = null
        workspaces.value = []
        saveToStorage(STORAGE_KEY_CLUSTER, null)
      }
    } catch (error) {
      clusters.value = []
    } finally {
      loadingClusters.value = false
    }
  }

  // 处理集群选择
  const handleClusterChange = async (clusterId: number | null) => {
    saveToStorage(STORAGE_KEY_CLUSTER, clusterId)

    if (clusterId) {
      const cluster = clusters.value.find((c) => c.id === clusterId)
      if (cluster) {
        selectedCluster.value = cluster
        await loadWorkspaces()
      } else {
        console.warn('选择的集群不存在，清空选择')
        selectedClusterId.value = null
        selectedCluster.value = null
        saveToStorage(STORAGE_KEY_CLUSTER, null)
      }
    } else {
      selectedCluster.value = null
      workspaces.value = []
    }
  }

  // 清除集群选择
  const handleClusterClear = () => {
    selectedCluster.value = null
    workspaces.value = []

    saveToStorage(STORAGE_KEY_CLUSTER, null)
  }

  // 加载工作空间
  const loadWorkspaces = async () => {
    if (!selectedCluster.value) {
      workspaces.value = []
      return
    }

    loading.value = true

    try {
      const response = await searchProjectWorkspaceApi({
        projectClusterId: selectedCluster.value.id
      })
      workspaces.value = response || []
    } catch (error) {
      console.error('❌ 加载工作空间失败:', error)
      workspaces.value = []
    } finally {
      loading.value = false
    }
  }

  // 刷新数据
  const refreshData = async () => {
    if (selectedCluster.value) {
      await loadWorkspaces()
      ElMessage.success('刷新成功')
    } else if (selectedProject.value) {
      await loadClusters()
      ElMessage.success('刷新成功')
    }
  }

  // 处理子组件的刷新事件
  const handleRefresh = async () => {
    await refreshData()
  }

  // 处理视图模式切换
  const handleViewModeChange = () => {
    ElMessage.success(`已切换到${viewMode.value === 'card' ? '卡片' : '列表'}视图`)
  }

  // 创建工作空间
  const handleCreate = () => {
    if (!selectedCluster.value) {
      return
    }

    dialogMode.value = 'create'
    currentWorkspace.value = null
    dialogVisible.value = true
  }

  // 编辑工作空间
  const handleEdit = (workspace: ProjectWorkspace) => {
    dialogMode.value = 'edit'
    currentWorkspace.value = workspace
    dialogVisible.value = true
  }

  // 删除工作空间
  const handleDelete = async (workspace: ProjectWorkspace) => {
    try {
      await ElMessageBox.confirm(
        `确定要删除工作空间 "${workspace.name}" 吗？此操作不可恢复。`,
        '删除确认',
        {
          confirmButtonText: '确定删除',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      // 显示全屏 loading
      const loadingInstance = ElLoading.service({
        lock: true,
        text: '正在删除工作空间，请稍候...',
        background: 'rgba(0, 0, 0, 0.7)'
      })

      try {
        await deleteProjectWorkspaceApi(workspace.id)
        ElMessage.success('删除成功')
        await loadWorkspaces()
      } finally {
        loadingInstance.close()
      }
    } catch (error) {
      if (error !== 'cancel') {
        console.error('删除工作空间失败:', error)
      }
    }
  }

  // 管理工作空间
  const handleManage = (workspace: ProjectWorkspace) => {
    router.push({
      name: 'WorkspaceManagement',
      params: { id: workspace.id }
    })
  }

  // 对话框成功回调
  const handleDialogSuccess = async () => {
    await loadWorkspaces()
  }

  const initPage = async () => {
    if (isInitializing.value) {
      return
    }

    isInitializing.value = true

    try {
      // 1. 确保 store 初始化
      await projectStore.ensureInitialized()

      // 2. 检查项目是否存在
      if (!selectedProject.value) {
        console.warn('⚠️ 未选择项目，清空所有状态')
        saveToStorage(STORAGE_KEY_CLUSTER, null)

        selectedClusterId.value = null
        selectedCluster.value = null
        clusters.value = []
        workspaces.value = []
        return
      }

      // 3. 项目存在时加载集群
      await loadClusters()

      // 4. 如果有选中的集群，加载工作空间
      if (selectedClusterId.value) {
        await loadWorkspaces()
      }
    } catch (error) {
      console.error('❌ 页面初始化失败:', error)
    } finally {
      setTimeout(() => {
        isInitializing.value = false
        hasCompletedInitialLoad.value = true
      }, 100)
    }
  }

  const clearAllStorage = () => {
    saveToStorage(STORAGE_KEY_CLUSTER, null)

    selectedClusterId.value = null
    selectedCluster.value = null
    clusters.value = []
    workspaces.value = []
  }

  defineExpose({
    clearAllStorage
  })

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

    if (selectedCluster.value) {
      await loadWorkspaces()
    }
  })
</script>

<style lang="scss" scoped>
  .workspace-container {
    height: 100%;
    padding: 0;
    background: #f5f7fa;

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
            max-width: 350px;
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

    .info-bar {
      margin-bottom: 16px;

      :deep(.el-card__body) {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 16px 20px;
      }

      .info-items {
        display: flex;
        align-items: center;
        gap: 12px;

        .info-item {
          display: flex;
          align-items: center;
          gap: 8px;

          .label {
            color: var(--el-text-color-secondary);
            font-size: 14px;
          }

          .value {
            color: var(--el-text-color-primary);
            font-weight: 500;
            font-size: 14px;
          }

          .resource-quick-view {
            display: flex;
            gap: 8px;
          }
        }
      }

      .view-switcher {
        :deep(.el-radio-button__inner) {
          display: flex;
          align-items: center;
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
  }
</style>

<style lang="scss">
  /* Workspace Management - 集群下拉菜单样式 */
  .workspace-mgmt-cluster-dropdown {
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
</style>
