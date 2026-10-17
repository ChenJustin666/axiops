<template>
  <div class="pod-monitor-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <!-- 返回按钮 -->
        <ElButton class="back-btn" @click="handleGoBack">
          <ElIcon><ArrowLeft /></ElIcon>
          <span>返回</span>
        </ElButton>
        <div class="header-info">
          <div class="breadcrumb">
            <span class="breadcrumb-item">{{ namespace }}</span>
            <span class="breadcrumb-separator">/</span>
            <span class="breadcrumb-item current">{{ podName }}</span>
          </div>
        </div>
      </div>
      <div class="header-right">
        <ElTag v-if="podStatus" :type="getStatusType(podStatus)" size="large">
          {{ podStatus }}
        </ElTag>
        <span v-if="lastUpdateTime" class="update-time">
          <ElIcon><Clock /></ElIcon>
          {{ formatUpdateTime(lastUpdateTime) }}
        </span>
      </div>
    </div>

    <!-- 工具栏 -->
    <MonitorToolbar
      ref="toolbarRef"
      :loading="refreshing && !isSilentRefresh"
      :containers="containerList"
      :selected-container="selectedContainer"
      @refresh="handleToolbarRefresh"
      @export="handleExport"
      @container-change="handleContainerChange"
    />

    <!-- Tab 导航 - 简洁风格 -->
    <div class="monitor-tabs">
      <div
        v-for="tab in tabList"
        :key="tab.name"
        class="tab-item"
        :class="{ active: activeTab === tab.name }"
        @click="handleTabClick(tab.name)"
      >
        <ElIcon class="tab-icon"><component :is="tab.icon" /></ElIcon>
        <span class="tab-text">{{ tab.label }}</span>
      </div>
    </div>

    <!-- Tab 内容区域 -->
    <div class="tab-content">
      <!-- 概览 Tab -->
      <div v-show="activeTab === 'overview'" class="tab-panel">
        <div class="metrics-container">
          <MonitorOverview
            :overview-data="overviewData"
            :loading="overviewLoading && !isSilentRefresh"
          />
          <PodStatus
            :status-data="statusData"
            :restart-data="restartData"
            :loading="statusLoading && !isSilentRefresh"
          />
        </div>
      </div>

      <!-- CPU & 内存 Tab -->
      <div v-show="activeTab === 'compute'" class="tab-panel">
        <div class="metrics-container">
          <CPUMonitor
            :cpu-data="cpuData"
            :loading="(cpuLoading && !isSilentRefresh) || containerSwitching"
            :container-name="selectedContainer === 'all' ? undefined : selectedContainer"
          />
          <MemoryMonitor
            :memory-data="memoryData"
            :loading="(memoryLoading && !isSilentRefresh) || containerSwitching"
            :container-name="selectedContainer === 'all' ? undefined : selectedContainer"
          />
        </div>
      </div>

      <!-- 网络 & 磁盘 Tab -->
      <div v-show="activeTab === 'io'" class="tab-panel">
        <div class="metrics-container">
          <NetworkMonitor
            :network-data="networkData"
            :loading="networkLoading && !isSilentRefresh"
          />
          <DiskMonitor :disk-data="diskData" :loading="diskLoading && !isSilentRefresh" />
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <ElEmpty v-if="showEmptyState" description="暂无监控数据，请刷新重试" :image-size="200">
      <ElButton type="primary" @click="handleManualRefresh">
        <ElIcon><RefreshRight /></ElIcon>
        刷新数据
      </ElButton>
    </ElEmpty>
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { Clock, RefreshRight, ArrowLeft, Monitor, Cpu, Connection } from '@element-plus/icons-vue'
  import MonitorToolbar from './modules/MonitorToolbar.vue'
  import MonitorOverview from './modules/monitor-overview.vue'
  import CPUMonitor from './modules/cpu-monitor.vue'
  import MemoryMonitor from './modules/memory-monitor.vue'
  import NetworkMonitor from './modules/network-monitor.vue'
  import DiskMonitor from './modules/disk-monitor.vue'
  import PodStatus from './modules/pod-status.vue'
  import { usePodMonitor, type ContainerInfo } from './composables/usePodMonitor'

  defineOptions({ name: 'PodMonitor' })

  const route = useRoute()
  const router = useRouter()

  // ==================== 路由参数 ====================
  const clusterUuid = ref(route.query.clusterUuid as string)
  const namespace = ref(route.query.namespace as string)
  const podName = ref(route.query.podName as string)

  // 参数验证
  if (!clusterUuid.value || !namespace.value || !podName.value) {
    ElMessageBox.alert('缺少必需的参数，请检查 URL 参数', '参数错误', {
      type: 'error',
      callback: () => {
        router.back()
      }
    })
  }

  // ==================== Tab 配置 ====================
  const tabList = [
    { name: 'overview', label: '概览', icon: Monitor },
    { name: 'compute', label: 'CPU & 内存', icon: Cpu },
    { name: 'io', label: '网络 & 磁盘', icon: Connection }
  ]

  // ==================== 状态管理 ====================
  const toolbarRef = ref()
  const activeTab = ref<string>('overview')
  const containerList = ref<ContainerInfo[]>([])
  const selectedContainer = ref('all')
  const refreshing = ref(false)
  const isSilentRefresh = ref(false)
  const containerSwitching = ref(false)
  const lastUpdateTime = ref<Date>()
  const hasInitialized = ref(false)

  // 当前时间参数
  const currentStart = ref<string>()
  const currentEnd = ref<string>()

  // Tab 懒加载状态管理
  const tabLoadedMap = ref<Record<string, boolean>>({
    overview: false,
    compute: false,
    io: false
  })

  const tabLoadingMap = ref<Record<string, boolean>>({
    overview: false,
    compute: false,
    io: false
  })

  // ==================== 使用监控数据 ====================
  const {
    overviewData,
    overviewLoading,
    cpuData,
    cpuLoading,
    memoryData,
    memoryLoading,
    networkData,
    networkLoading,
    diskData,
    diskLoading,
    statusData,
    restartData,
    statusLoading,
    loadOverviewData,
    loadCPUData,
    loadMemoryData,
    loadNetworkData,
    loadDiskData,
    loadStatusData,
    extractContainersFromStatus
  } = usePodMonitor()

  // ==================== 计算属性 ====================

  /** Pod 当前状态 */
  const podStatus = computed(() => statusData.value?.current?.phase)

  /** 是否显示空状态 */
  const showEmptyState = computed(() => {
    return (
      activeTab.value === 'overview' &&
      hasInitialized.value &&
      !overviewLoading.value &&
      !statusLoading.value &&
      !overviewData.value &&
      !statusData.value
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

  /** 获取状态类型 */
  const getStatusType = (phase?: string) => {
    switch (phase) {
      case 'Running':
        return 'success'
      case 'Pending':
        return 'warning'
      case 'Failed':
        return 'danger'
      default:
        return 'info'
    }
  }

  // ==================== 数据加载方法 ====================

  /**
   * 获取基础查询参数
   */
  const getBaseParams = () => {
    if (!currentStart.value || !currentEnd.value) {
      return null
    }
    return {
      clusterUuid: clusterUuid.value,
      namespace: namespace.value,
      podName: podName.value,
      start: currentStart.value,
      end: currentEnd.value
    }
  }

  /**
   * 加载概览 Tab 数据
   */
  const loadOverviewTabData = async () => {
    const params = getBaseParams()
    if (!params) return

    // 先加载状态数据，用于获取容器列表
    await loadStatusData(params)

    // 从状态数据中提取容器列表
    const containers = extractContainersFromStatus()
    if (containers.length > 0) {
      containerList.value = containers
    }

    // 加载概览数据
    await loadOverviewData(params)
  }

  /**
   * 加载 CPU 和内存数据（支持按容器查看）
   */
  const loadComputeTabData = async () => {
    const params = getBaseParams()
    if (!params) return

    containerSwitching.value = true

    try {
      if (selectedContainer.value === 'all') {
        await Promise.allSettled([loadCPUData(params), loadMemoryData(params)])
      } else {
        await Promise.allSettled([
          loadCPUData({ ...params, containerName: selectedContainer.value }),
          loadMemoryData({ ...params, containerName: selectedContainer.value })
        ])
      }
    } finally {
      containerSwitching.value = false
    }
  }

  /**
   * 加载网络 & 磁盘 Tab 数据
   */
  const loadIOTabData = async () => {
    const params = getBaseParams()
    if (!params) return

    await Promise.allSettled([loadNetworkData(params), loadDiskData(params)])
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
        case 'overview':
          await loadOverviewTabData()
          break
        case 'compute':
          await loadComputeTabData()
          break
        case 'io':
          await loadIOTabData()
          break
      }

      tabLoadedMap.value[tabName] = true
      console.log(`✅ Tab [${tabName}] 数据加载完成`)
    } catch (error) {
      console.error(`❌ Tab [${tabName}] 数据加载失败:`, error)
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
   * 返回上一页
   */
  const handleGoBack = () => {
    router.back()
  }

  /**
   * 处理 Tab 点击
   */
  const handleTabClick = async (tabName: string) => {
    if (activeTab.value === tabName) return
    activeTab.value = tabName
    console.log(`📌 切换到 Tab: ${tabName}`)
    await loadTabData(tabName, true)
    lastUpdateTime.value = new Date()
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

      lastUpdateTime.value = new Date()
      hasInitialized.value = true
    } catch (error) {
      console.error('❌ 刷新失败:', error)
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
   * 处理容器切换
   */
  const handleContainerChange = async (container: string) => {
    selectedContainer.value = container

    // 如果当前在 compute Tab，重新加载数据
    if (activeTab.value === 'compute') {
      await loadComputeTabData()
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
          cluster: clusterUuid.value,
          namespace: namespace.value,
          pod: podName.value,
          container: selectedContainer.value,
          containerType: selectedContainer.value === 'all' ? 'Pod级别' : '容器级别',
          timeRange: {
            start: currentStart.value,
            end: currentEnd.value
          }
        },
        metrics: {
          overview: overviewData.value,
          cpu: cpuData.value,
          memory: memoryData.value,
          network: networkData.value,
          disk: diskData.value,
          status: statusData.value,
          restart: restartData.value
        }
      }

      const blob = new Blob([JSON.stringify(exportData, null, 2)], {
        type: 'application/json'
      })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url

      const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, -5)
      const containerSuffix = selectedContainer.value === 'all' ? 'pod' : selectedContainer.value
      a.download = `pod-monitor_${podName.value}_${containerSuffix}_${timestamp}.json`

      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)

      ElMessage.success('数据导出成功')
    } catch (error) {
      console.error('❌ 导出失败:', error)
    }
  }

  // ==================== 监听容器变化 ====================
  watch(
    () => containerList.value,
    (newList) => {
      if (
        selectedContainer.value !== 'all' &&
        !newList.find((c) => c.name === selectedContainer.value)
      ) {
        selectedContainer.value = 'all'
      }
    }
  )

  // ==================== 生命周期 ====================
  onMounted(() => {})

  onUnmounted(() => {})
</script>

<style lang="scss" scoped>
  .pod-monitor-page {
    max-width: 1800px;
    margin: 0 auto;
    min-height: calc(100vh - 100px);
  }

  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
    padding: 20px 24px;
    background: linear-gradient(135deg, #ffffff 0%, #f8fafc 100%);
    border-radius: 16px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.04);
    border: 1px solid rgba(0, 0, 0, 0.04);

    .header-left {
      display: flex;
      align-items: center;
      gap: 16px;

      .back-btn {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 10px 18px;
        border-radius: 10px;
        background: linear-gradient(135deg, #f5f7fa 0%, #e8ecf1 100%);
        border: 1px solid #e4e7ed;
        color: #606266;
        font-size: 14px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.3s ease;

        &:hover {
          background: linear-gradient(135deg, #ecf5ff 0%, #d9ecff 100%);
          border-color: #409eff;
          color: #409eff;
          transform: translateX(-2px);
        }

        .el-icon {
          font-size: 16px;
        }
      }

      .header-info {
        .breadcrumb {
          font-size: 15px;
          color: #909399;
          display: flex;
          align-items: center;
          gap: 10px;

          .breadcrumb-item {
            padding: 6px 12px;
            border-radius: 8px;
            background: #f5f7fa;
            transition: all 0.3s;

            &.current {
              color: #409eff;
              font-weight: 600;
              background: linear-gradient(135deg, #ecf5ff 0%, #d9ecff 100%);
            }
          }

          .breadcrumb-separator {
            color: #dcdfe6;
            font-weight: 300;
          }
        }
      }
    }

    .header-right {
      display: flex;
      align-items: center;
      gap: 16px;

      .el-tag {
        padding: 8px 16px;
        font-size: 13px;
        font-weight: 600;
        border-radius: 20px;
      }

      .update-time {
        font-size: 13px;
        color: #909399;
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 8px 14px;
        background: linear-gradient(135deg, #f5f7fa 0%, #e8ecf1 100%);
        border-radius: 10px;
        border: 1px solid #ebeef5;
      }
    }
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
        font-size: 16px;
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
    .pod-monitor-page {
      padding: 12px;
    }

    .page-header {
      flex-direction: column;
      align-items: flex-start;
      gap: 12px;
      padding: 16px;

      .header-left {
        flex-wrap: wrap;
        width: 100%;

        .back-btn {
          margin-bottom: 8px;
        }
      }

      .header-right {
        width: 100%;
        justify-content: space-between;
      }
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

        .tab-text {
          display: none;
        }
      }
    }
  }
</style>
