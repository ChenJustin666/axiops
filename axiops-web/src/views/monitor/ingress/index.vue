<template>
  <div class="ingress-monitor-page">
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
            <span class="breadcrumb-item current">{{ host }}</span>
          </div>
        </div>
      </div>
      <div class="header-right">
        <ElTag v-if="ingressStatus" :type="getStatusType(ingressStatus)" size="large">
          {{ ingressStatus }}
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
      @refresh="handleToolbarRefresh"
      @export="handleExport"
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
          <ControllerHealth
            :controller-data="controllerData"
            :loading="controllerLoading && !isSilentRefresh"
          />
          <TrafficOverview
            :traffic-data="trafficData"
            :loading="trafficLoading && !isSilentRefresh"
          />
        </div>
      </div>

      <!-- 流量分析 Tab -->
      <div v-show="activeTab === 'traffic'" class="tab-panel">
        <div class="metrics-container">
          <TrafficMetrics
            :traffic-data="trafficData"
            :loading="trafficLoading && !isSilentRefresh"
          />
        </div>
      </div>

      <!-- 性能分析 Tab -->
      <div v-show="activeTab === 'performance'" class="tab-panel">
        <div class="metrics-container">
          <PerformanceMetrics
            :performance-data="performanceData"
            :loading="performanceLoading && !isSilentRefresh"
          />
        </div>
      </div>

      <!-- 错误分析 Tab -->
      <div v-show="activeTab === 'errors'" class="tab-panel">
        <div class="metrics-container">
          <ErrorMetrics :error-data="errorData" :loading="errorLoading && !isSilentRefresh" />
        </div>
      </div>

      <!-- 后端健康 Tab -->
      <div v-show="activeTab === 'backends'" class="tab-panel">
        <div class="metrics-container">
          <BackendHealth
            :backend-data="backendData"
            :loading="backendLoading && !isSilentRefresh"
          />
        </div>
      </div>

      <!-- 证书管理 Tab -->
      <div v-show="activeTab === 'certificates'" class="tab-panel">
        <div class="metrics-container">
          <CertificateManagement
            :certificate-data="certificateData"
            :loading="certificateLoading && !isSilentRefresh"
          />
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
  import { ref, computed, onMounted, onUnmounted } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import {
    Clock,
    RefreshRight,
    ArrowLeft,
    Monitor,
    TrendCharts,
    Timer,
    Warning,
    Connection,
    Lock
  } from '@element-plus/icons-vue'
  import MonitorToolbar from './modules/MonitorToolbar.vue'
  import ControllerHealth from './modules/controller-health.vue'
  import TrafficOverview from './modules/traffic-overview.vue'
  import TrafficMetrics from './modules/traffic-metrics.vue'
  import PerformanceMetrics from './modules/performance-metrics.vue'
  import ErrorMetrics from './modules/error-metrics.vue'
  import BackendHealth from './modules/backend-health.vue'
  import CertificateManagement from './modules/certificate-management.vue'
  import { useIngressMonitor } from './composables/useIngressMonitor'

  defineOptions({ name: 'IngressMonitor' })

  const route = useRoute()
  const router = useRouter()

  // ==================== 路由参数 ====================
  const clusterUuid = ref(route.query.clusterUuid as string)
  const namespace = ref(route.query.namespace as string)
  const host = ref(route.query.host as string)
  const ingressName = ref(route.query.ingressName as string) || host.value

  // 参数验证
  if (!clusterUuid.value || !host.value) {
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
    { name: 'traffic', label: '流量分析', icon: TrendCharts },
    { name: 'performance', label: '性能分析', icon: Timer },
    { name: 'errors', label: '错误分析', icon: Warning },
    { name: 'backends', label: '后端健康', icon: Connection },
    { name: 'certificates', label: '证书管理', icon: Lock }
  ]

  // ==================== 状态管理 ====================
  const toolbarRef = ref()
  const activeTab = ref<string>('overview')
  const refreshing = ref(false)
  const isSilentRefresh = ref(false)
  const lastUpdateTime = ref<Date>()
  const hasInitialized = ref(false)

  // 当前时间参数
  const currentStart = ref<string>()
  const currentEnd = ref<string>()

  // Tab 懒加载状态管理
  const tabLoadedMap = ref<Record<string, boolean>>({
    overview: false,
    traffic: false,
    performance: false,
    errors: false,
    backends: false,
    certificates: false
  })

  const tabLoadingMap = ref<Record<string, boolean>>({
    overview: false,
    traffic: false,
    performance: false,
    errors: false,
    backends: false,
    certificates: false
  })

  // ==================== 使用监控数据 ====================
  const {
    controllerData,
    controllerLoading,
    trafficData,
    trafficLoading,
    performanceData,
    performanceLoading,
    errorData,
    errorLoading,
    backendData,
    backendLoading,
    certificateData,
    certificateLoading,
    loadControllerHealth,
    loadTrafficMetrics,
    loadPerformanceMetrics,
    loadErrorMetrics,
    loadBackendMetrics,
    loadCertificateMetrics
  } = useIngressMonitor()

  // ==================== 计算属性 ====================

  /** Ingress 当前状态 */
  const ingressStatus = computed(() => {
    if (!controllerData.value) return undefined
    if (controllerData.value.readyPods > 0) return '正常'
    return '异常'
  })

  /** 是否显示空状态 */
  const showEmptyState = computed(() => {
    return (
      activeTab.value === 'overview' &&
      hasInitialized.value &&
      !controllerLoading.value &&
      !trafficLoading.value &&
      !controllerData.value &&
      !trafficData.value
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
  const getStatusType = (status?: string) => {
    switch (status) {
      case '正常':
        return 'success'
      case '异常':
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
      host: host.value,
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

    await Promise.allSettled([
      loadControllerHealth({
        clusterUuid: clusterUuid.value,
        start: params.start,
        end: params.end
      }),
      loadTrafficMetrics(params)
    ])
  }

  /**
   * 加载流量分析 Tab 数据
   */
  const loadTrafficTabData = async () => {
    const params = getBaseParams()
    if (!params) return

    await loadTrafficMetrics(params)
  }

  /**
   * 加载性能分析 Tab 数据
   */
  const loadPerformanceTabData = async () => {
    const params = getBaseParams()
    if (!params) return

    await loadPerformanceMetrics(params)
  }

  /**
   * 加载错误分析 Tab 数据
   */
  const loadErrorsTabData = async () => {
    const params = getBaseParams()
    if (!params) return

    await loadErrorMetrics(params)
  }

  /**
   * 加载后端健康 Tab 数据
   */
  const loadBackendsTabData = async () => {
    const params = getBaseParams()
    if (!params) return

    await loadBackendMetrics(params)
  }

  /**
   * 加载证书管理 Tab 数据
   */
  const loadCertificatesTabData = async () => {
    // 证书查询需要 namespace 和 ingressName
    if (!namespace.value) {
      console.warn('[Ingress监控] 证书查询需要 namespace 参数')
      ElMessage.warning('证书查询需要 namespace 参数')
      return
    }

    await loadCertificateMetrics({
      clusterUuid: clusterUuid.value,
      namespace: namespace.value,
      ingressName: ingressName.value
    })
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
        case 'traffic':
          await loadTrafficTabData()
          break
        case 'performance':
          await loadPerformanceTabData()
          break
        case 'errors':
          await loadErrorsTabData()
          break
        case 'backends':
          await loadBackendsTabData()
          break
        case 'certificates':
          await loadCertificatesTabData()
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
          host: host.value,
          timeRange: {
            start: currentStart.value,
            end: currentEnd.value
          }
        },
        metrics: {
          controller: controllerData.value,
          traffic: trafficData.value,
          performance: performanceData.value,
          errors: errorData.value,
          backends: backendData.value,
          certificates: certificateData.value
        }
      }

      const blob = new Blob([JSON.stringify(exportData, null, 2)], {
        type: 'application/json'
      })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url

      const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, -5)
      a.download = `ingress-monitor_${host.value}_${timestamp}.json`

      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)

      ElMessage.success('数据导出成功')
    } catch (error) {
      console.error('❌ 导出失败:', error)
      ElMessage.error('数据导出失败')
    }
  }

  // ==================== 生命周期 ====================
  onMounted(() => {})

  onUnmounted(() => {})
</script>

<style lang="scss" scoped>
  .ingress-monitor-page {
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
    .ingress-monitor-page {
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
