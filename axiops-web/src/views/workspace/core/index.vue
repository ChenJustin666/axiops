<template>
  <div class="core-config-page">
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
            v-model="managementStore.selectedClusterId"
            placeholder="选择集群"
            clearable
            size="default"
            :disabled="!selectedProject"
            :loading="loadingClusters"
            filterable
            @change="handleClusterChange"
            @clear="handleClusterClear"
            popper-class="core-config-cluster-dropdown"
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
            v-model="managementStore.selectedWorkspaceId"
            placeholder="选择工作空间"
            clearable
            size="default"
            filterable
            :disabled="!managementStore.selectedClusterId"
            :loading="loadingWorkspaces"
            @change="handleWorkspaceChange"
            @clear="handleWorkspaceClear"
            popper-class="core-config-workspace-dropdown"
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
          :cluster="managementStore.selectedCluster"
          :workspace="managementStore.selectedWorkspace"
          resource-name="Kubernetes 配置"
        />
      </div>

      <!-- Tab内容 -->
      <div v-else class="tabs-container">
        <!-- 自定义Tab头部 -->
        <div class="custom-tabs-header">
          <div class="tabs-nav">
            <div
              v-for="tab in tabList"
              :key="tab.name"
              class="tab-item"
              :class="{ 'is-active': activeTab === tab.name }"
              @click="handleTabClick(tab.name)"
            >
              <component :is="tab.icon" :size="15" class="tab-icon" />
              <span class="tab-label">{{ tab.label }}</span>
            </div>
          </div>
        </div>

        <!-- Tab内容区 -->
        <div class="tabs-content">
          <!-- ConfigMap -->
          <div v-show="activeTab === 'configmap'" class="tab-pane">
            <ConfigMapManagement
              v-if="loadedTabs.configmap"
              :cluster="managementStore.selectedCluster"
              :workspace="managementStore.selectedWorkspace"
              :refresh-trigger="refreshTriggers.configmap"
            />
          </div>

          <!-- Secret -->
          <div v-show="activeTab === 'secret'" class="tab-pane">
            <SecretManagement
              v-if="loadedTabs.secret"
              :cluster="managementStore.selectedCluster"
              :workspace="managementStore.selectedWorkspace"
              :refresh-trigger="refreshTriggers.secret"
            />
          </div>

          <!-- Service -->
          <div v-show="activeTab === 'service'" class="tab-pane">
            <ServiceManagement
              v-if="loadedTabs.service"
              :cluster="managementStore.selectedCluster"
              :workspace="managementStore.selectedWorkspace"
              :refresh-trigger="refreshTriggers.service"
            />
          </div>

          <!-- Ingress -->
          <div v-show="activeTab === 'ingress'" class="tab-pane">
            <IngressManagement
              v-if="loadedTabs.ingress"
              :cluster="managementStore.selectedCluster"
              :workspace="managementStore.selectedWorkspace"
              :refresh-trigger="refreshTriggers.ingress"
            />
          </div>

          <!-- PVC -->
          <div v-show="activeTab === 'pvc'" class="tab-pane">
            <PVCManagement
              v-if="loadedTabs.pvc"
              :workspace="managementStore.selectedWorkspace"
              :refresh-trigger="refreshTriggers.pvc"
            />
          </div>

          <!-- ServiceAccount -->
          <div v-show="activeTab === 'serviceaccount'" class="tab-pane">
            <ServiceAccountManagement
              v-if="loadedTabs.serviceaccount"
              :workspace="managementStore.selectedWorkspace"
              :refresh-trigger="refreshTriggers.serviceaccount"
            />
          </div>

          <!-- Role -->
          <div v-show="activeTab === 'role'" class="tab-pane">
            <RoleManagement
              v-if="loadedTabs.role"
              :workspace="managementStore.selectedWorkspace"
              :refresh-trigger="refreshTriggers.role"
            />
          </div>

          <!-- RoleBinding -->
          <div v-show="activeTab === 'rolebinding'" class="tab-pane">
            <RoleBindingManagement
              v-if="loadedTabs.rolebinding"
              :workspace="managementStore.selectedWorkspace"
              :refresh-trigger="refreshTriggers.rolebinding"
            />
          </div>
        </div>
      </div>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, watch, onMounted, onActivated, nextTick } from 'vue'
  import {
    RefreshCw,
    Server,
    Box,
    ChevronRight,
    FileText,
    Lock,
    Globe,
    Network,
    HardDrive,
    User,
    Shield,
    Link
  } from 'lucide-vue-next'
  import { useProjectStore } from '@/store/modules/project'
  import { useApplicationManagementStore } from '@/store/modules/applicationManagement'
  import {
    searchProjectClusterApi,
    searchProjectWorkspaceApi,
    type ProjectCluster,
    type ProjectWorkspace
  } from '@/api'
  import ConfigMapManagement from './configmap/index.vue'
  import SecretManagement from './secret/index.vue'
  import ServiceManagement from './service/index.vue'
  import IngressManagement from './ingress/index.vue'
  import PVCManagement from './pvc/index.vue'
  import ServiceAccountManagement from './serviceaccount/index.vue'
  import RoleManagement from './role/index.vue'
  import RoleBindingManagement from './rolebinding/index.vue'
  import SelectionGuideState from '@/components/SelectionGuideState/index.vue'

  defineOptions({ name: 'CoreConfigPage' })

  const projectStore = useProjectStore()
  const managementStore = useApplicationManagementStore()

  const selectedProject = computed(() => projectStore.selectedProject)

  const STORAGE_KEY_CLUSTER = 'core-config-selected-cluster'
  const STORAGE_KEY_WORKSPACE = 'core-config-selected-workspace'
  const STORAGE_KEY_TAB = 'core-config-active-tab'

  // Tab配置列表
  const tabList = [
    { name: 'configmap', label: 'ConfigMap', icon: FileText },
    { name: 'secret', label: 'Secret', icon: Lock },
    { name: 'service', label: 'Service', icon: Globe },
    { name: 'ingress', label: 'Ingress', icon: Network },
    { name: 'pvc', label: 'PVC', icon: HardDrive },
    { name: 'serviceaccount', label: 'ServiceAccount', icon: User },
    { name: 'role', label: 'Role', icon: Shield },
    { name: 'rolebinding', label: 'RoleBinding', icon: Link }
  ]

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

  const clusters = ref<ProjectCluster[]>([])
  const workspaces = ref<ProjectWorkspace[]>([])

  const loadingClusters = ref(false)
  const loadingWorkspaces = ref(false)

  const activeTab = ref('configmap')

  const isInitializing = ref(false)
  const hasCompletedInitialLoad = ref(false)

  const isWorkspaceReady = computed(() => {
    return !!(
      managementStore.selectedClusterId &&
      managementStore.selectedWorkspaceId &&
      managementStore.selectedCluster &&
      managementStore.selectedWorkspace
    )
  })

  // 标签页懒加载标记
  const loadedTabs = ref({
    configmap: false,
    secret: false,
    service: false,
    ingress: false,
    pvc: false,
    serviceaccount: false,
    role: false,
    rolebinding: false
  })

  // 标签页刷新触发器
  const refreshTriggers = ref({
    configmap: 0,
    secret: 0,
    service: 0,
    ingress: 0,
    pvc: 0,
    serviceaccount: 0,
    role: 0,
    rolebinding: 0
  })

  watch(selectedProject, async (newProject, oldProject) => {
    if (isInitializing.value) {
      return
    }

    if (newProject?.id !== oldProject?.id) {
      saveToStorage(STORAGE_KEY_CLUSTER, null)
      saveToStorage(STORAGE_KEY_WORKSPACE, null)
      saveToStorage(STORAGE_KEY_TAB, null)

      managementStore.setCluster(null)
      managementStore.setWorkspace(null)

      clusters.value = []
      workspaces.value = []
      resetLoadedTabs()

      activeTab.value = 'configmap'

      if (newProject) {
        await loadClusters()
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
        if (savedClusterId) {
          const cluster = clusters.value.find((c) => c.id === Number(savedClusterId))
          if (cluster) {
            managementStore.setCluster(cluster, true)
          } else {
            console.warn('缓存的集群不存在，清空选择')
            saveToStorage(STORAGE_KEY_CLUSTER, null)
            saveToStorage(STORAGE_KEY_WORKSPACE, null)
            saveToStorage(STORAGE_KEY_TAB, null)
            managementStore.setCluster(null)
            managementStore.setWorkspace(null)
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
      const cluster = clusters.value.find((c) => c.id === clusterId)
      if (cluster) {
        managementStore.setCluster(cluster)
        await loadWorkspaces()
      }
    } else {
      managementStore.setCluster(null)
      managementStore.setWorkspace(null)
      workspaces.value = []
      resetLoadedTabs()

      saveToStorage(STORAGE_KEY_WORKSPACE, null)
    }
  }

  const handleClusterClear = () => {
    managementStore.setCluster(null)
    managementStore.setWorkspace(null)
    workspaces.value = []
    resetLoadedTabs()

    activeTab.value = 'configmap'

    saveToStorage(STORAGE_KEY_CLUSTER, null)
    saveToStorage(STORAGE_KEY_WORKSPACE, null)
    saveToStorage(STORAGE_KEY_TAB, null)
  }

  const loadWorkspaces = async () => {
    if (!managementStore.selectedClusterId) {
      workspaces.value = []
      return
    }

    if (loadingWorkspaces.value) {
      return
    }

    loadingWorkspaces.value = true

    try {
      const response = await searchProjectWorkspaceApi({
        projectClusterId: managementStore.selectedClusterId
      })
      workspaces.value = response || []

      if (isInitializing.value) {
        const savedWorkspaceId = loadFromStorage(STORAGE_KEY_WORKSPACE)
        if (savedWorkspaceId) {
          const workspace = workspaces.value.find((w) => w.id === Number(savedWorkspaceId))
          if (workspace) {
            managementStore.setWorkspace(workspace, true)
            resetLoadedTabs()

            const savedTab = loadFromStorage(STORAGE_KEY_TAB)
            if (savedTab && typeof savedTab === 'string') {
              activeTab.value = savedTab
            }

            // 初始化时加载当前tab - 只设置 loadedTabs，让子组件的 onMounted 处理加载
            await nextTick()
            loadedTabs.value[activeTab.value as keyof typeof loadedTabs.value] = true
          } else {
            console.warn('缓存的工作空间不存在，清空选择')
            saveToStorage(STORAGE_KEY_WORKSPACE, null)
            managementStore.setWorkspace(null)
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
      const workspace = workspaces.value.find((w) => w.id === workspaceId)
      if (workspace) {
        // 使用 silent=true 避免触发额外的 watch
        managementStore.setWorkspace(workspace, true)
        resetLoadedTabs()
        // 工作空间变化后，设置当前tab为已加载，让子组件的 onMounted 处理加载
        nextTick(() => {
          loadedTabs.value[activeTab.value as keyof typeof loadedTabs.value] = true
        })
      } else {
        console.warn('选择的工作空间不存在，清空选择')
        managementStore.setWorkspace(null, true)
        saveToStorage(STORAGE_KEY_WORKSPACE, null)
      }
    } else {
      managementStore.setWorkspace(null, true)
      resetLoadedTabs()
    }
  }

  const handleWorkspaceClear = () => {
    managementStore.setWorkspace(null)
    resetLoadedTabs()

    saveToStorage(STORAGE_KEY_WORKSPACE, null)
  }

  const resetLoadedTabs = () => {
    loadedTabs.value = {
      configmap: false,
      secret: false,
      service: false,
      ingress: false,
      pvc: false,
      serviceaccount: false,
      role: false,
      rolebinding: false
    }
  }

  /**
   * 处理Tab点击
   * 核心逻辑：
   * 1. 首次进入某个tab时，只设置 loadedTabs = true，由子组件的 onMounted 加载数据
   * 2. 再次进入已加载的tab时，触发 refreshTrigger++ 来刷新数据
   * 这样避免了首次进入时 onMounted 和 watch(refreshTrigger) 同时触发导致的重复请求
   */
  const handleTabClick = (tabName: string) => {
    if (!isWorkspaceReady.value) {
      return
    }

    // 如果点击的是当前tab，直接返回
    if (activeTab.value === tabName) {
      return
    }

    // 切换到新tab
    activeTab.value = tabName
    saveToStorage(STORAGE_KEY_TAB, tabName)

    const tabKey = tabName as keyof typeof loadedTabs.value
    const wasLoaded = loadedTabs.value[tabKey]

    if (!wasLoaded) {
      // 首次加载此tab - 只设置 loadedTabs = true，让子组件的 onMounted 处理数据加载
      loadedTabs.value[tabKey] = true
    } else {
      // 再次切换到已加载的tab - 触发 refreshTrigger 刷新数据
      const trigger = tabName as keyof typeof refreshTriggers.value
      if (trigger in refreshTriggers.value) {
        refreshTriggers.value[trigger]++
      }
    }
  }

  const handleRefresh = () => {
    if (!isWorkspaceReady.value) {
      return
    }

    const tab = activeTab.value as keyof typeof refreshTriggers.value
    if (tab in refreshTriggers.value) {
      refreshTriggers.value[tab]++
    }
  }

  const initPage = async () => {
    if (isInitializing.value) {
      return
    }

    isInitializing.value = true

    try {
      await Promise.all([projectStore.ensureInitialized(), managementStore.ensureInitialized()])

      if (!selectedProject.value) {
        console.warn('未选择项目，清空所有状态')
        saveToStorage(STORAGE_KEY_CLUSTER, null)
        saveToStorage(STORAGE_KEY_WORKSPACE, null)
        saveToStorage(STORAGE_KEY_TAB, null)
        return
      }

      await loadClusters()

      if (managementStore.selectedClusterId) {
        await loadWorkspaces()
      }
    } catch {
      // 全局拦截器处理错误
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

    // 页面被激活时，如果已完成初始化且工作空间已就绪，刷新当前tab
    if (isWorkspaceReady.value && activeTab.value) {
      const tab = activeTab.value as keyof typeof refreshTriggers.value
      if (tab in refreshTriggers.value) {
        refreshTriggers.value[tab]++
      }
    }
  })
</script>

<style lang="scss" scoped>
  .core-config-page {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: #f5f7fa;

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

      .tabs-container {
        height: 100%;
        display: flex;
        flex-direction: column;

        // 自定义Tab头部样式 - 简洁风格
        .custom-tabs-header {
          flex-shrink: 0;
          background: #ffffff;
          border-bottom: 1px solid #f0f0f0;
          padding: 0 20px;

          .tabs-nav {
            display: flex;
            align-items: center;
            gap: 32px;
            overflow-x: auto;
            scrollbar-width: none;
            -ms-overflow-style: none;

            &::-webkit-scrollbar {
              display: none;
            }

            .tab-item {
              display: flex;
              align-items: center;
              gap: 8px;
              padding: 16px 0;
              font-size: 14px;
              font-weight: 400;
              color: #8c8c8c;
              cursor: pointer;
              border-bottom: 2px solid transparent;
              transition: all 0.2s ease;
              white-space: nowrap;
              position: relative;
              margin-bottom: -1px;

              .tab-icon {
                flex-shrink: 0;
                opacity: 0.75;
                transition: all 0.2s ease;
              }

              .tab-label {
                transition: all 0.2s ease;
              }

              &:hover {
                color: #409eff;

                .tab-icon {
                  opacity: 1;
                }
              }

              &.is-active {
                color: #409eff;
                font-weight: 500;
                border-bottom-color: #409eff;

                .tab-icon {
                  opacity: 1;
                }
              }
            }
          }
        }

        // Tab内容区样式
        .tabs-content {
          flex: 1;
          overflow-y: auto;
          //padding: 16px;
          background: #f5f7fa;

          .tab-pane {
            min-height: 400px;
          }
        }
      }
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

    .custom-tabs-header {
      .tabs-nav {
        gap: 24px !important;

        .tab-item {
          padding: 12px 0 !important;
          font-size: 13px !important;
        }
      }
    }
  }
</style>

<style lang="scss">
  /* Core Config 集群下拉菜单样式 */
  .core-config-cluster-dropdown {
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

  /* Core Config 工作空间下拉菜单样式 */
  .core-config-workspace-dropdown {
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
