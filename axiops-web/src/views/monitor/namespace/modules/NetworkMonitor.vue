<template>
  <div class="namespace-network-monitor" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="monitor-card">
        <!-- 卡片头部 -->
        <div class="card-header">
          <div class="header-info">
            <div class="icon-badge network">
              <svg viewBox="0 0 24 24" width="22" height="22">
                <path
                  fill="currentColor"
                  d="M17,3A2,2 0 0,1 19,5V15A2,2 0 0,1 17,17H13V19H14A1,1 0 0,1 15,20H22V22H15A1,1 0 0,1 14,23H10A1,1 0 0,1 9,22H2V20H9A1,1 0 0,1 10,19H11V17H7C5.89,17 5,16.1 5,15V5A2,2 0 0,1 7,3H17M17,5H7V15H17V5Z"
                />
              </svg>
            </div>
            <div>
              <h3>网络流量</h3>
              <p>入站与出站网络监控</p>
            </div>
          </div>
          <div class="header-stats">
            <div class="stat-badge total">
              <span class="label">总流量</span>
              <span class="value">{{ formatBytes(totalTraffic) }}</span>
            </div>
            <div class="stat-badge ingress">
              <span class="label">入站总量</span>
              <span class="value">{{
                formatBytes(networkData?.summary?.totalReceiveBytes || 0)
              }}</span>
            </div>
            <div class="stat-badge egress">
              <span class="label">出站总量</span>
              <span class="value">{{
                formatBytes(networkData?.summary?.totalTransmitBytes || 0)
              }}</span>
            </div>
          </div>
        </div>

        <!-- 快速指标 -->
        <div class="quick-metrics">
          <div class="quick-metric">
            <div class="metric-icon ingress">
              <svg viewBox="0 0 24 24" width="20" height="20">
                <path
                  fill="currentColor"
                  d="M7.41,8.58L12,13.17L16.59,8.58L18,10L12,16L6,10L7.41,8.58Z"
                />
              </svg>
            </div>
            <div class="metric-info">
              <span class="metric-label">平均入站速率</span>
              <span class="metric-value">{{
                formatBitRate(networkData?.summary?.avgReceiveBytesPerSec || 0)
              }}</span>
            </div>
          </div>
          <div class="quick-metric">
            <div class="metric-icon egress">
              <svg viewBox="0 0 24 24" width="20" height="20">
                <path
                  fill="currentColor"
                  d="M7.41,15.41L12,10.83L16.59,15.41L18,14L12,8L6,14L7.41,15.41Z"
                />
              </svg>
            </div>
            <div class="metric-info">
              <span class="metric-label">平均出站速率</span>
              <span class="metric-value">{{
                formatBitRate(networkData?.summary?.avgTransmitBytesPerSec || 0)
              }}</span>
            </div>
          </div>
        </div>

        <!-- 图表区域 -->
        <div class="chart-area">
          <div class="chart-header">
            <span class="chart-title">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M3.5,18.49L9.5,12.48L13.5,16.48L22,6.92L20.59,5.51L13.5,13.48L9.5,9.48L2,16.99L3.5,18.49Z"
                />
              </svg>
              网络速率趋势
            </span>
            <div v-if="networkData?.trend" class="chart-info">
              共 {{ networkData.trend.length }} 个数据点
            </div>
          </div>
          <div ref="chartRef" class="chart-container"></div>
          <div class="chart-legend">
            <span class="legend-item"> <i class="dot ingress"></i>入站速率 </span>
            <span class="legend-item"> <i class="dot egress"></i>出站速率 </span>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" width="64" height="64">
            <path
              fill="currentColor"
              d="M17,3A2,2 0 0,1 19,5V15A2,2 0 0,1 17,17H13V19H14A1,1 0 0,1 15,20H22V22H15A1,1 0 0,1 14,23H10A1,1 0 0,1 9,22H2V20H9A1,1 0 0,1 10,19H11V17H7C5.89,17 5,16.1 5,15V5A2,2 0 0,1 7,3H17M17,5H7V15H17V5Z"
            />
          </svg>
        </div>
        <h3>暂无网络监控数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, onUnmounted, nextTick, computed } from 'vue'
  import * as echarts from 'echarts'
  import type { NamespaceNetworkMetrics } from '@/api/console/namespace-monitor'
  import { formatBytes, formatBitRate, smartFormatTime } from '@/utils/format'

  // ==================== 类型定义 ====================

  interface Props {
    networkData?: NamespaceNetworkMetrics
    loading?: boolean
  }

  // ==================== 组件属性 ====================

  const props = withDefaults(defineProps<Props>(), {
    networkData: undefined,
    loading: false
  })

  // ==================== 响应式数据 ====================

  const chartRef = ref<HTMLElement>()
  let chart: echarts.ECharts | null = null
  let chartInitialized = false

  // ==================== 计算属性 ====================

  /**
   * 判断是否有数据
   */
  const hasData = computed(() => {
    return !!(props.networkData && Object.keys(props.networkData).length > 0)
  })

  /**
   * 计算总流量
   */
  const totalTraffic = computed(() => {
    const ingress = props.networkData?.summary?.totalReceiveBytes || 0
    const egress = props.networkData?.summary?.totalTransmitBytes || 0
    return ingress + egress
  })

  // ==================== 图表相关方法 ====================

  /**
   * 初始化图表
   */
  const initChart = () => {
    if (!chartRef.value) {
      return
    }

    // 检查容器尺寸
    const width = chartRef.value.clientWidth
    const height = chartRef.value.clientHeight

    if (width === 0 || height === 0) {
      // 延迟重试
      setTimeout(() => {
        initChart()
      }, 200)
      return
    }

    // 销毁已存在的图表实例
    if (chart) {
      chart.dispose()
      chart = null
    }

    chart = echarts.init(chartRef.value)
    chartInitialized = true

    const option: echarts.EChartsOption = {
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'cross' },
        backgroundColor: 'rgba(255, 255, 255, 0.98)',
        borderColor: '#eaecef',
        borderWidth: 1,
        padding: [12, 16],
        textStyle: { color: '#1f2937', fontSize: 12 },
        formatter: (params: any) => {
          if (!Array.isArray(params) || params.length === 0) return ''

          const timestamp = params[0].data[0]
          const formattedTime = smartFormatTime(timestamp, 'YYYY-MM-DD HH:mm:ss')

          let html = `<div style="font-weight: bold; margin-bottom: 8px;">${formattedTime}</div>`

          params.forEach((param: any) => {
            const value = param.data[1]
            const formattedValue = formatBitRate(value)
            html += `
              <div style="display: flex; align-items: center; margin: 4px 0;">
                <span style="display: inline-block; width: 10px; height: 10px; border-radius: 50%; background: ${param.color}; margin-right: 8px;"></span>
                <span style="flex: 1;">${param.seriesName}:</span>
                <span style="font-weight: bold; margin-left: 12px;">${formattedValue}</span>
              </div>
            `
          })

          return html
        }
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '8%',
        top: '12%',
        containLabel: true
      },
      xAxis: {
        type: 'time',
        boundaryGap: false,
        axisLine: { lineStyle: { color: '#e5e7eb' } },
        axisLabel: {
          color: '#6b7280',
          fontSize: 11,
          formatter: (value: number) => smartFormatTime(value, 'HH:mm:ss')
        },
        splitLine: { show: true, lineStyle: { color: '#f3f4f6', type: 'dashed' } }
      },
      yAxis: {
        type: 'value',
        name: '速率',
        nameTextStyle: { color: '#6b7280', fontSize: 11 },
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: {
          color: '#6b7280',
          fontSize: 11,
          formatter: (value: number) => formatBitRate(value)
        },
        splitLine: { lineStyle: { color: '#f3f4f6', type: 'dashed' } }
      },
      series: [
        {
          name: '入站速率',
          type: 'line',
          data: props.networkData?.trend?.map((t) => [t.timestamp, t.receiveBytesPerSec]) || [],
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#4facfe' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(79, 172, 254, 0.25)' },
              { offset: 1, color: 'rgba(79, 172, 254, 0.02)' }
            ])
          }
        },
        {
          name: '出站速率',
          type: 'line',
          data: props.networkData?.trend?.map((t) => [t.timestamp, t.transmitBytesPerSec]) || [],
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#fa8c16' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(250, 140, 22, 0.25)' },
              { offset: 1, color: 'rgba(250, 140, 22, 0.02)' }
            ])
          }
        }
      ]
    }

    chart.setOption(option)
  }

  /**
   * 更新图表数据
   */
  const updateChart = () => {
    if (!chart || !props.networkData?.trend) return

    chart.setOption({
      series: [
        { data: props.networkData.trend.map((t) => [t.timestamp, t.receiveBytesPerSec]) },
        { data: props.networkData.trend.map((t) => [t.timestamp, t.transmitBytesPerSec]) }
      ]
    })
  }

  /**
   * 处理窗口尺寸变化
   */
  const handleResize = () => {
    chart?.resize()
  }

  // ==================== 生命周期钩子 ====================

  watch(
    () => props.networkData,
    async () => {
      if (!hasData.value) return

      await nextTick()

      // 延迟等待 DOM 渲染完成
      setTimeout(() => {
        if (props.networkData?.trend?.length) {
          if (!chart || !chartInitialized) {
            initChart()
          } else {
            updateChart()
          }
        }
      }, 150)
    },
    { deep: true, immediate: true }
  )

  onMounted(() => {
    window.addEventListener('resize', handleResize)
  })

  onUnmounted(() => {
    window.removeEventListener('resize', handleResize)
    chart?.dispose()
    chart = null
    chartInitialized = false
  })
</script>

<style lang="scss" scoped>
  // ==================== 颜色变量 ====================
  $primary: #3b82f6;
  $success: #10b981;
  $warning: #f59e0b;
  $danger: #ef4444;

  $gray-50: #f9fafb;
  $gray-100: #f3f4f6;
  $gray-200: #e5e7eb;
  $gray-300: #d1d5db;
  $gray-400: #9ca3af;
  $gray-500: #6b7280;
  $gray-600: #4b5563;
  $gray-700: #374151;
  $gray-800: #1f2937;

  $radius: 14px;
  $radius-sm: 10px;
  $radius-xs: 6px;

  // ==================== 主容器样式 ====================
  .namespace-network-monitor {
    padding: 20px;
    background: $gray-50;
    font-family: var(--ax-font-sans);

    .monitor-card {
      max-width: 1800px;
      margin: 0 auto;
      background: #fff;
      border-radius: $radius;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
      overflow: hidden;
    }
  }

  // ==================== 卡片头部样式 ====================
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 20px 24px;
    background: linear-gradient(135deg, #e0f7ff 0%, #d0eeff 100%);
    border-bottom: 1px solid #b3e5fc;
    flex-wrap: wrap;
    gap: 14px;

    .header-info {
      display: flex;
      align-items: center;
      gap: 14px;

      .icon-badge {
        width: 46px;
        height: 46px;
        border-radius: $radius-sm;
        display: flex;
        align-items: center;
        justify-content: center;
        color: #fff;

        &.network {
          background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
        }
      }

      h3 {
        margin: 0;
        font-size: 16px;
        font-weight: 600;
        color: $gray-800;
      }

      p {
        margin: 2px 0 0;
        font-size: 12px;
        color: $gray-500;
      }
    }

    .header-stats {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;

      .stat-badge {
        padding: 8px 14px;
        border-radius: $radius-xs;
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 12px;

        &.total {
          background: linear-gradient(135deg, #e0e7ff 0%, #c7d2fe 100%);
          color: #4338ca;
        }

        &.ingress {
          background: linear-gradient(135deg, #d1fae5 0%, #a7f3d0 100%);
          color: #065f46;
        }

        &.egress {
          background: linear-gradient(135deg, #fed7aa 0%, #fbbf24 100%);
          color: #92400e;
        }

        .label {
          opacity: 0.8;
        }

        .value {
          font-weight: 700;
          font-size: 14px;
        }
      }
    }
  }

  // ==================== 快速指标样式 ====================
  .quick-metrics {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 16px;
    padding: 20px 24px;
    border-bottom: 1px solid $gray-200;

    @media (max-width: 640px) {
      grid-template-columns: 1fr;
    }

    .quick-metric {
      display: flex;
      align-items: center;
      gap: 14px;
      padding: 18px;
      border-radius: $radius-sm;
      background: $gray-50;
      border: 1px solid $gray-200;
      transition: all 0.2s ease;

      &:hover {
        border-color: $gray-300;
      }

      .metric-icon {
        width: 48px;
        height: 48px;
        border-radius: $radius-xs;
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;

        &.ingress {
          background: linear-gradient(135deg, #dbeafe, #bfdbfe);
          color: $primary;
        }

        &.egress {
          background: linear-gradient(135deg, #fef3c7, #fde68a);
          color: $warning;
        }
      }

      .metric-info {
        .metric-label {
          font-size: 12px;
          color: $gray-500;
          margin-bottom: 4px;
          display: block;
        }

        .metric-value {
          font-size: 20px;
          font-weight: 700;
          color: $gray-800;
          display: block;
        }
      }
    }
  }

  // ==================== 图表区域样式 ====================
  .chart-area {
    padding: 20px 24px;

    .chart-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 16px;

      .chart-title {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 14px;
        font-weight: 600;
        color: $gray-700;

        svg {
          color: $gray-500;
        }
      }

      .chart-info {
        font-size: 12px;
        color: $gray-400;
        padding: 4px 12px;
        background: $gray-50;
        border-radius: $radius-xs;
      }
    }

    .chart-container {
      height: 320px;
      width: 100%;
    }

    .chart-legend {
      display: flex;
      justify-content: center;
      gap: 24px;
      margin-top: 16px;

      .legend-item {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 12px;
        color: $gray-500;

        .dot {
          width: 10px;
          height: 3px;
          border-radius: 2px;

          &.ingress {
            background: #4facfe;
          }

          &.egress {
            background: #fa8c16;
          }
        }
      }
    }
  }

  // ==================== 空状态样式 ====================
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 350px;
    background: #fff;
    border-radius: $radius;
    padding: 60px 40px;

    .empty-icon {
      color: $gray-300;
      margin-bottom: 20px;
    }

    h3 {
      margin: 0 0 10px;
      font-size: 16px;
      color: $gray-600;
      font-weight: 500;
    }

    p {
      margin: 0;
      font-size: 13px;
      color: $gray-400;
    }
  }

  // ==================== 动画 ====================
  .fade-enter-active,
  .fade-leave-active {
    transition: opacity 0.3s ease;
  }

  .fade-enter-from,
  .fade-leave-to {
    opacity: 0;
  }

  // ==================== 响应式布局 ====================
  @media (max-width: 768px) {
    .namespace-network-monitor {
      padding: 12px;
    }

    .card-header {
      padding: 16px;

      .header-stats {
        width: 100%;
      }
    }

    .chart-area {
      padding: 16px;

      .chart-container {
        height: 280px;
      }
    }
  }
</style>
