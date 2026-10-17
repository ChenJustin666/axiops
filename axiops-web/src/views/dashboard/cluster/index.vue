<template>
  <div class="cluster-monitor-page">
    <!-- 工具栏 - 包含集群选择、时间范围 -->
    <div class="toolbar-wrapper">
      <MonitorToolbar
        :loading="refreshing && !isSilentRefresh"
        :clusters="clusterList"
        :selected-cluster="selectedCluster"
        @refresh="handleToolbarRefresh"
        @export="handleExport"
        @cluster-change="handleClusterChange"
      />
    </div>

    <!-- Tab 导航 - 简洁下划线风格 -->
    <div class="monitor-tabs">
      <div
        v-for="tab in tabList"
        :key="tab.name"
        class="tab-item"
        :class="{ active: activeTab === tab.name }"
        @click="handleTabClick(tab.name)"
      >
        <img :src="tab.icon" class="tab-icon" alt="" />
        <span class="tab-text">{{ tab.label }}</span>
      </div>
    </div>

    <!-- Tab 内容区域 -->
    <div class="tab-content">
      <!-- 集群资源 Tab -->
      <div v-show="activeTab === 'resources'" class="tab-panel">
        <div class="metrics-container">
          <ResourceMonitor
            :resource-data="resourceData"
            :loading="resourceLoading && !isSilentRefresh"
          />
          <NetworkMonitor
            :network-data="networkData"
            :loading="networkLoading && !isSilentRefresh"
          />
          <StorageMonitor
            :storage-data="storageData"
            :loading="storageLoading && !isSilentRefresh"
          />
        </div>
      </div>

      <!-- 控制平面 Tab -->
      <div v-show="activeTab === 'control-plane'" class="tab-panel">
        <div class="metrics-container">
          <ControlPlaneMonitor
            :control-plane-data="controlPlaneDataForDisplay"
            :loading="controlPlaneLoading && !isSilentRefresh"
          />
        </div>
      </div>

      <!-- Etcd Tab -->
      <div v-show="activeTab === 'etcd'" class="tab-panel">
        <div class="metrics-container">
          <EtcdMonitor :etcd-data="etcdData" :loading="etcdLoading && !isSilentRefresh" />
        </div>
      </div>

      <!-- 节点 Tab -->
      <div v-show="activeTab === 'nodes'" class="tab-panel">
        <div class="metrics-container">
          <NodeMonitor :node-data="nodeData" :loading="nodeLoading && !isSilentRefresh" />
        </div>
      </div>

      <!-- 工作负载 Tab -->
      <div v-show="activeTab === 'workloads'" class="tab-panel">
        <div class="metrics-container">
          <WorkloadMonitor
            :workload-data="workloadData"
            :loading="workloadLoading && !isSilentRefresh"
          />
        </div>
      </div>

      <!-- Pod & 命名空间 Tab -->
      <div v-show="activeTab === 'pods'" class="tab-panel">
        <div class="metrics-container">
          <ClusterOverview
            :overview-data="overviewData"
            :loading="overviewLoading && !isSilentRefresh"
          />
          <NamespaceMonitor
            :namespace-data="namespaceData"
            :loading="namespaceLoading && !isSilentRefresh"
          />
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <ElEmpty
      v-if="showEmptyState"
      description="暂无监控数据，请选择集群或刷新重试"
      :image-size="200"
    >
      <ElButton v-if="!selectedCluster" type="primary" @click="handleSelectFirstCluster">
        <ElIcon><Select /></ElIcon>
        选择集群
      </ElButton>
      <ElButton v-else type="primary" @click="handleManualRefresh">
        <ElIcon><RefreshRight /></ElIcon>
        刷新数据
      </ElButton>
    </ElEmpty>
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
  import { ElMessage } from 'element-plus'
  import { Clock, RefreshRight, Select } from '@element-plus/icons-vue'
  import { searchClusterApi, type Cluster } from '@/api/manager/cluster'
  import MonitorToolbar from './modules/MonitorToolbar.vue'
  import ClusterOverview from './modules/cluster-overview.vue'
  import ResourceMonitor from './modules/resource-monitor.vue'
  import NodeMonitor from './modules/node-monitor.vue'
  import ControlPlaneMonitor from './modules/control-plane-monitor.vue'
  import EtcdMonitor from './modules/etcd-monitor.vue'
  import WorkloadMonitor from './modules/workload-monitor.vue'
  import NetworkMonitor from './modules/network-monitor.vue'
  import StorageMonitor from './modules/storage-monitor.vue'
  import NamespaceMonitor from './modules/namespace-monitor.vue'
  import { useClusterMonitor } from './composables/useClusterMonitor'
  import { getClusterHealthScore } from '@/api/console/cluster-monitor'

  // 导入图标
  import monitoringIcon from '@/assets/img/monitoring/monitoring.png'
  import serverIcon from '@/assets/img/monitoring/server.png'
  import diskIcon from '@/assets/img/monitoring/disk.png'
  import podIcon from '@/assets/img/monitoring/pod.png'

  defineOptions({ name: 'ClusterMonitor' })

  // ==================== Tab 配置 ====================
  const tabList = [
    { name: 'resources', label: '集群资源', icon: monitoringIcon },
    { name: 'control-plane', label: '控制平面', icon: serverIcon },
    { name: 'etcd', label: 'Etcd', icon: diskIcon },
    { name: 'nodes', label: '节点', icon: serverIcon },
    { name: 'workloads', label: '工作负载', icon: podIcon },
    { name: 'pods', label: 'Pod & 命名空间', icon: podIcon }
  ]

  // ==================== 状态管理 ====================
  const clusterList = ref<Cluster[]>([])
  const selectedCluster = ref<string>('')
  const selectedClusterName = ref<string>('')
  const refreshing = ref(false)
  const isSilentRefresh = ref(false)
  const lastUpdateTime = ref<Date>()
  const hasInitialized = ref(false)
  const activeTab = ref<string>('resources')

  // 当前时间参数
  const currentStart = ref<string>()
  const currentEnd = ref<string>()

  // Tab 懒加载状态管理
  const tabLoadedMap = ref<Record<string, boolean>>({
    resources: false,
    'control-plane': false,
    etcd: false,
    nodes: false,
    workloads: false,
    pods: false
  })

  const tabLoadingMap = ref<Record<string, boolean>>({
    resources: false,
    'control-plane': false,
    etcd: false,
    nodes: false,
    workloads: false,
    pods: false
  })

  // ==================== 使用监控数据 ====================
  const {
    overviewData,
    overviewLoading,
    resourceData,
    resourceLoading,
    nodeData,
    nodeLoading,
    controlPlaneData,
    controlPlaneLoading,
    workloadData,
    workloadLoading,
    networkData,
    networkLoading,
    storageData,
    storageLoading,
    namespaceData,
    namespaceLoading,
    etcdData,
    etcdLoading,
    loadOverviewData,
    loadResourceData,
    loadNodeData,
    loadControlPlaneData,
    loadWorkloadData,
    loadNetworkData,
    loadStorageData,
    loadNamespaceData,
    loadEtcdData
  } = useClusterMonitor()

  // ==================== 计算属性 ====================

  /** 集群健康分数 */
  const clusterHealth = computed(() => {
    if (!overviewData.value) return null
    return Math.round(getClusterHealthScore(overviewData.value))
  })

  /** 控制平面数据（排除 etcd） */
  const controlPlaneDataForDisplay = computed(() => {
    if (!controlPlaneData.value) return undefined
    const { etcd, ...rest } = controlPlaneData.value
    return rest
  })

  /** 是否显示空状态 */
  const showEmptyState = computed(() => {
    return hasInitialized.value && !isAnyDataLoaded.value
  })

  /** 是否有任何数据已加载 */
  const isAnyDataLoaded = computed(() => {
    return !!(
      overviewData.value ||
      resourceData.value ||
      nodeData.value ||
      controlPlaneData.value ||
      workloadData.value ||
      networkData.value ||
      storageData.value ||
      namespaceData.value ||
      etcdData.value
    )
  })

  /** 是否有任何加载中 */
  const isAnyLoading = computed(() => {
    return (
      overviewLoading.value ||
      resourceLoading.value ||
      nodeLoading.value ||
      controlPlaneLoading.value ||
      workloadLoading.value ||
      networkLoading.value ||
      storageLoading.value ||
      namespaceLoading.value ||
      etcdLoading.value
    )
  })

  // ==================== 工具函数 ====================

  /** 格式化更新时间 */
  const formatUpdateTime = (date: Date) => {
    return date.toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
  }

  /** 获取健康状态类型 */
  const getHealthType = (score: number) => {
    if (score >= 90) return 'success'
    if (score >= 70) return 'warning'
    return 'danger'
  }

  // ==================== 集群列表加载 ====================

  /**
   * 加载集群列表
   */
  const loadClusterList = async () => {
    try {
      const response = await searchClusterApi({
        page: 1,
        pageSize: 100
      })

      if (response.items && response.items.length > 0) {
        clusterList.value = response.items

        if (!selectedCluster.value) {
          const firstCluster = response.items[0]
          selectedCluster.value = firstCluster.uuid
          selectedClusterName.value = firstCluster.name
        }
      } else {
        console.warn('⚠️  未找到任何集群')
      }
    } catch (error) {}
  }

  // ==================== 数据加载方法 ====================

  /**
   * 获取基础查询参数
   */
  const getBaseParams = () => {
    if (!currentStart.value || !currentEnd.value || !selectedCluster.value) {
      return null
    }
    return {
      clusterUuid: selectedCluster.value,
      start: currentStart.value,
      end: currentEnd.value
    }
  }

  /**
   * 加载集群资源 Tab 数据
   */
  const loadResourcesTabData = async () => {
    const params = getBaseParams()
    if (!params) return

    await Promise.allSettled([
      loadResourceData(params),
      loadNetworkData(params),
      loadStorageData({ clusterUuid: params.clusterUuid })
    ])
  }

  /**
   * 加载控制平面 Tab 数据
   */
  const loadControlPlaneTabData = async () => {
    const params = getBaseParams()
    if (!params) return

    await loadControlPlaneData(params)
  }

  /**
   * 加载 Etcd Tab 数据
   */
  const loadEtcdTabData = async () => {
    const params = getBaseParams()
    if (!params) return

    await loadEtcdData(params)
  }

  /**
   * 加载节点 Tab 数据
   */
  const loadNodesTabData = async () => {
    const params = getBaseParams()
    if (!params) return

    await loadNodeData(params)
  }

  /**
   * 加载工作负载 Tab 数据
   */
  const loadWorkloadsTabData = async () => {
    const params = getBaseParams()
    if (!params) return

    await loadWorkloadData(params)
  }

  /**
   * 加载 Pod & 命名空间 Tab 数据
   */
  const loadPodsTabData = async () => {
    const params = getBaseParams()
    if (!params) return

    await Promise.allSettled([loadOverviewData(params), loadNamespaceData(params)])
  }

  /**
   * 根据 Tab 名称加载对应数据
   */
  const loadTabData = async (tabName: string, forceReload: boolean = false) => {
    // 防止重复加载
    if (tabLoadingMap.value[tabName]) {
      console.log(`⏳ Tab [${tabName}] 正在加载中，跳过重复请求`)
      return
    }

    // 如果已经加载过且不是强制刷新，跳过
    if (tabLoadedMap.value[tabName] && !forceReload) {
      console.log(`✅ Tab [${tabName}] 已加载，跳过`)
      return
    }

    tabLoadingMap.value[tabName] = true

    try {
      console.log(`🔄 开始加载 Tab [${tabName}] 数据`)

      switch (tabName) {
        case 'resources':
          await loadResourcesTabData()
          break
        case 'control-plane':
          await loadControlPlaneTabData()
          break
        case 'etcd':
          await loadEtcdTabData()
          break
        case 'nodes':
          await loadNodesTabData()
          break
        case 'workloads':
          await loadWorkloadsTabData()
          break
        case 'pods':
          await loadPodsTabData()
          break
      }

      tabLoadedMap.value[tabName] = true
      lastUpdateTime.value = new Date()
      console.log(`✅ Tab [${tabName}] 数据加载完成`)
    } catch (error) {
    } finally {
      tabLoadingMap.value[tabName] = false
    }
  }

  /**
   * 重置所有 Tab 的加载状态
   */
  const resetAllTabLoadedState = () => {
    Object.keys(tabLoadedMap.value).forEach((key) => {
      tabLoadedMap.value[key] = false
    })
  }

  // ==================== 事件处理 ====================

  /**
   * 处理 Tab 点击
   */
  const handleTabClick = async (tabName: string) => {
    if (activeTab.value === tabName) return
    activeTab.value = tabName
    console.log(`📌 切换到 Tab: ${tabName}`)
    await loadTabData(tabName, true)
  }

  /**
   * 处理工具栏刷新事件
   */
  const handleToolbarRefresh = async (start: string, end: string, silent: boolean = false) => {
    try {
      currentStart.value = start
      currentEnd.value = end
      isSilentRefresh.value = silent

      if (!silent) {
        refreshing.value = true
      }

      // 重置所有 Tab 的加载状态
      resetAllTabLoadedState()

      // 加载当前激活 Tab 的数据
      await loadTabData(activeTab.value, true)

      hasInitialized.value = true
    } catch (error) {
    } finally {
      if (!silent) {
        refreshing.value = false
      }
      isSilentRefresh.value = false
    }
  }

  /**
   * 手动刷新
   */
  const handleManualRefresh = async () => {
    if (!currentStart.value || !currentEnd.value) {
      return
    }
    await handleToolbarRefresh(currentStart.value, currentEnd.value, false)
  }

  /**
   * 选择第一个集群
   */
  const handleSelectFirstCluster = () => {
    if (clusterList.value.length > 0) {
      const firstCluster = clusterList.value[0]
      handleClusterChange(firstCluster.uuid)
    }
  }

  /**
   * 处理集群切换
   */
  const handleClusterChange = async (clusterUuid: string) => {
    const previousCluster = selectedCluster.value

    selectedCluster.value = clusterUuid
    const cluster = clusterList.value.find((c) => c.uuid === clusterUuid)
    if (cluster) {
      selectedClusterName.value = cluster.name
    }

    // 重置所有 Tab 的加载状态
    resetAllTabLoadedState()

    if (currentStart.value && currentEnd.value) {
      await loadTabData(activeTab.value, true)
    }
  }

  /**
   * 导出数据
   */
  const handleExport = () => {
    try {
      const exportData = {
        metadata: {
          exportTime: new Date().toISOString(),
          exportVersion: '1.0.0',
          cluster: selectedCluster.value,
          clusterName: selectedClusterName.value,
          activeTab: activeTab.value,
          timeRange: {
            start: currentStart.value,
            end: currentEnd.value
          }
        },
        metrics: {
          overview: overviewData.value,
          resources: resourceData.value,
          nodes: nodeData.value,
          controlPlane: controlPlaneData.value,
          workloads: workloadData.value,
          network: networkData.value,
          storage: storageData.value,
          namespaces: namespaceData.value,
          etcd: etcdData.value
        }
      }

      const blob = new Blob([JSON.stringify(exportData, null, 2)], {
        type: 'application/json'
      })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url

      const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, -5)
      a.download = `cluster-monitor_${selectedClusterName.value}_${timestamp}.json`

      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)

      ElMessage.success('数据导出成功')
    } catch (error) {}
  }

  // ==================== 生命周期 ====================
  onMounted(async () => {
    await loadClusterList()

    if (selectedCluster.value) {
      await nextTick()

      const end = new Date()
      const start = new Date(end.getTime() - 30 * 60 * 1000)

      currentStart.value = start.toISOString()
      currentEnd.value = end.toISOString()

      // 只加载第一个 tab（集群资源）的数据
      await loadTabData(activeTab.value, true)
      hasInitialized.value = true
    }
  })

  onUnmounted(() => {})
</script>

<style lang="scss" scoped>
  .cluster-monitor-page {
    max-width: 1800px;
    margin: 0 auto;
    min-height: calc(100vh - 100px);
  }

  /* 工具栏包装器 */
  .toolbar-wrapper {
    position: relative;
    z-index: 10;
    margin-bottom: 20px;
  }

  // Tab 导航 - 简洁下划线风格
  .monitor-tabs {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 24px;
    border-bottom: 1px solid #ebeef5;
    padding-bottom: 0;

    .tab-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 12px 20px;
      font-size: 14px;
      font-weight: 500;
      color: #606266;
      cursor: pointer;
      border-bottom: 2px solid transparent;
      margin-bottom: -1px;
      transition: all 0.25s ease;

      .tab-icon {
        width: 18px;
        height: 18px;
        object-fit: contain;
        opacity: 0.7;
        transition: all 0.25s ease;
      }

      .tab-text {
        transition: all 0.25s ease;
      }

      &:hover {
        color: #409eff;

        .tab-icon {
          opacity: 1;
        }
      }

      &.active {
        color: #409eff;
        border-bottom-color: #409eff;

        .tab-icon {
          opacity: 1;
        }
      }
    }
  }

  // Tab 内容区域
  .tab-content {
    .tab-panel {
      animation: fadeIn 0.3s ease;
    }
  }

  .metrics-container {
    display: flex;
    flex-direction: column;
    gap: 20px;

    > * {
      animation: fadeInUp 0.4s ease-out;
    }
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }

  @keyframes fadeInUp {
    from {
      opacity: 0;
      transform: translateY(15px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @media (max-width: 768px) {
    .cluster-monitor-page {
      padding: 12px;
    }

    .monitor-tabs {
      overflow-x: auto;
      -webkit-overflow-scrolling: touch;

      &::-webkit-scrollbar {
        display: none;
      }

      .tab-item {
        padding: 10px 16px;
        font-size: 13px;
        white-space: nowrap;

        .tab-icon {
          width: 16px;
          height: 16px;
        }

        .tab-text {
          display: none;
        }
      }
    }
  }
</style>
