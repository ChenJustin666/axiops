<template>
  <div class="network-monitor" v-loading="loading">
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
              <h3>网络 I/O</h3>
              <p>网络流量与速率监控</p>
            </div>
          </div>
          <div class="status-tag" :class="networkStatusClass">{{ networkStatusText }}</div>
        </div>

        <!-- 指标网格 -->
        <div class="metrics-grid">
          <div class="metric-box primary">
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M12,3C7.58,3 4,4.79 4,7C4,9.21 7.58,11 12,11C16.42,11 20,9.21 20,7C20,4.79 16.42,3 12,3M4,9V12C4,14.21 7.58,16 12,16C16.42,16 20,14.21 20,12V9C20,11.21 16.42,13 12,13C7.58,13 4,11.21 4,9M4,14V17C4,19.21 7.58,21 12,21C16.42,21 20,19.21 20,17V14C20,16.21 16.42,18 12,18C7.58,18 4,16.21 4,14Z"
                />
              </svg>
              <span>总流量</span>
            </div>
            <div class="box-value">
              <span class="value">{{ formatBytes(totalTraffic) }}</span>
            </div>
          </div>

          <div class="metric-box">
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path fill="currentColor" d="M9,4H15V12H19.84L12,19.84L4.16,12H9V4Z" />
              </svg>
              <span>总接收</span>
            </div>
            <div class="box-value">
              <span class="value receive">{{
                formatBytes(networkData?.summary?.totalReceiveBytes || 0)
              }}</span>
            </div>
          </div>

          <div class="metric-box">
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path fill="currentColor" d="M15,20H9V12H4.16L12,4.16L19.84,12H15V20Z" />
              </svg>
              <span>总发送</span>
            </div>
            <div class="box-value">
              <span class="value transmit">{{
                formatBytes(networkData?.summary?.totalTransmitBytes || 0)
              }}</span>
            </div>
          </div>

          <div class="metric-box" :class="{ danger: hasErrors }">
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M12,2C17.53,2 22,6.47 22,12C22,17.53 17.53,22 12,22C6.47,22 2,17.53 2,12C2,6.47 6.47,2 12,2M15.59,7L12,10.59L8.41,7L7,8.41L10.59,12L7,15.59L8.41,17L12,13.41L15.59,17L17,15.59L13.41,12L17,8.41L15.59,7Z"
                />
              </svg>
              <span>总错误</span>
            </div>
            <div class="box-value">
              <span class="value" :class="{ danger: hasErrors }">{{
                networkData?.summary?.totalErrors || 0
              }}</span>
              <span class="unit">次</span>
            </div>
          </div>
        </div>

        <!-- 速率统计 -->
        <div class="stats-bar">
          <div class="stats-item receive">
            <span class="stats-label">平均接收速率</span>
            <span class="stats-value">{{
              formatBitRate(networkData?.summary?.avgReceiveBytesPerSec || 0)
            }}</span>
          </div>
          <div class="stats-item transmit">
            <span class="stats-label">平均发送速率</span>
            <span class="stats-value">{{
              formatBitRate(networkData?.summary?.avgTransmitBytesPerSec || 0)
            }}</span>
          </div>
        </div>

        <!-- 图表区域 -->
        <div class="chart-area">
          <div class="chart-header">
            <span class="chart-title">网络速率趋势</span>
            <div class="chart-controls">
              <div class="control-group">
                <button
                  class="control-btn"
                  :class="{ active: chartDisplayMode === 'rate' }"
                  @click="chartDisplayMode = 'rate'"
                >
                  速率
                </button>
                <button
                  class="control-btn"
                  :class="{ active: chartDisplayMode === 'traffic' }"
                  @click="chartDisplayMode = 'traffic'"
                >
                  累计流量
                </button>
              </div>
            </div>
            <div class="chart-legend">
              <span class="legend-item"><i class="dot receive"></i>接收</span>
              <span class="legend-item"><i class="dot transmit"></i>发送</span>
            </div>
            <div v-if="networkData?.trend" class="chart-info">
              共 {{ networkData.trend.length }} 个数据点
            </div>
          </div>
          <div ref="networkChartRef" class="chart-container"></div>
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
  import type { NetworkMetrics } from '@/api/console/pod-monitor'

  interface Props {
    networkData?: NetworkMetrics
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    networkData: undefined,
    loading: false
  })

  // 图表相关
  const networkChartRef = ref<HTMLElement>()
  let networkChart: echarts.ECharts | null = null
  let resizeObserver: ResizeObserver | null = null

  // 显示模式
  const chartDisplayMode = ref<'rate' | 'traffic'>('rate')

  // 数据状态
  const hasData = computed(() => {
    return !!(props.networkData && Object.keys(props.networkData).length > 0)
  })

  // 总流量
  const totalTraffic = computed(() => {
    const receive = props.networkData?.summary?.totalReceiveBytes || 0
    const transmit = props.networkData?.summary?.totalTransmitBytes || 0
    return receive + transmit
  })

  // 是否有错误
  const hasErrors = computed(() => {
    return (props.networkData?.summary?.totalErrors || 0) > 0
  })

  // 网络状态
  const networkStatusClass = computed(() => {
    if (hasErrors.value) return 'danger'
    return 'success'
  })

  const networkStatusText = computed(() => {
    if (hasErrors.value) return '有错误'
    return '正常'
  })

  // 字节格式化
  const formatBytes = (bytes: number): string => {
    if (!bytes || bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
  }

  // 比特率格式化
  const formatBitRate = (bytesPerSec: number): string => {
    if (!bytesPerSec || bytesPerSec === 0) return '0 bit/s'
    const bitsPerSec = bytesPerSec * 8
    const k = 1000
    const sizes = ['bit/s', 'Kbit/s', 'Mbit/s', 'Gbit/s', 'Tbit/s']
    const i = Math.min(Math.floor(Math.log(bitsPerSec) / Math.log(k)), sizes.length - 1)
    return `${(bitsPerSec / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
  }

  // 字节速率格式化
  const formatByteRate = (bytesPerSec: number): string => {
    if (!bytesPerSec || bytesPerSec === 0) return '0 B/s'
    const k = 1024
    const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s', 'TB/s']
    const i = Math.min(Math.floor(Math.log(bytesPerSec) / Math.log(k)), sizes.length - 1)
    return `${(bytesPerSec / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
  }

  // 时间格式化
  const formatTime = (timestamp: any): string => {
    try {
      let date: Date
      if (typeof timestamp === 'number') {
        date = timestamp < 10000000000 ? new Date(timestamp * 1000) : new Date(timestamp)
      } else if (typeof timestamp === 'string') {
        date = new Date(timestamp)
      } else if (timestamp instanceof Date) {
        date = timestamp
      } else {
        return '-'
      }
      if (isNaN(date.getTime())) return '-'
      return date.toLocaleString('zh-CN', {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false
      })
    } catch {
      return '-'
    }
  }

  // 计算速率数据
  const calculateRateData = (type: 'receive' | 'transmit'): Array<[string, number]> => {
    if (!props.networkData?.trend || props.networkData.trend.length < 2) return []

    const trend = props.networkData.trend
    const rateData: Array<[string, number]> = []

    for (let i = 1; i < trend.length; i++) {
      const currentPoint = trend[i]
      const prevPoint = trend[i - 1]
      const currentBytes =
        type === 'receive' ? currentPoint.receiveBytes : currentPoint.transmitBytes
      const prevBytes = type === 'receive' ? prevPoint.receiveBytes : prevPoint.transmitBytes
      const timeDiff =
        (new Date(currentPoint.timestamp).getTime() - new Date(prevPoint.timestamp).getTime()) /
        1000

      if (timeDiff > 0) {
        const rate = Math.max(0, (currentBytes - prevBytes) / timeDiff)
        rateData.push([currentPoint.timestamp, rate])
      }
    }

    return rateData
  }

  // 初始化图表
  const initChart = () => {
    if (!networkChartRef.value) return

    networkChart = echarts.init(networkChartRef.value)

    const option: echarts.EChartsOption = {
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(255, 255, 255, 0.98)',
        borderColor: '#eaecef',
        borderWidth: 1,
        padding: [12, 16],
        textStyle: { color: '#1f2937', fontSize: 12 },
        axisPointer: { type: 'cross', crossStyle: { color: '#999' } },
        formatter: (params: any) => {
          if (!Array.isArray(params)) return ''
          let result = `<div style="font-weight: 600; margin-bottom: 8px; font-size: 13px;">${formatTime(params[0].value[0])}</div>`
          params.forEach((item: any) => {
            const value =
              chartDisplayMode.value === 'rate'
                ? formatBitRate(item.value[1])
                : formatBytes(item.value[1])
            result += `<div style="margin: 4px 0; display: flex; align-items: center; font-size: 13px;">
            <span style="display: inline-block; width: 10px; height: 10px; border-radius: 50%; background: ${item.color}; margin-right: 8px;"></span>
            <span style="flex: 1;">${item.seriesName}:</span>
            <span style="font-weight: 600; margin-left: 12px;">${value}</span>
          </div>`
          })
          return result
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
          formatter: (v: number) =>
            new Date(v).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
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
          formatter: (value: number) => {
            return chartDisplayMode.value === 'rate' ? formatBitRate(value) : formatBytes(value)
          }
        },
        splitLine: { lineStyle: { color: '#f3f4f6', type: 'dashed' } }
      },
      series: [
        {
          name: '接收',
          type: 'line',
          smooth: true,
          data: [],
          symbol: 'none',
          lineStyle: { width: 2, color: '#4facfe' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(79, 172, 254, 0.2)' },
              { offset: 1, color: 'rgba(79, 172, 254, 0.02)' }
            ])
          }
        },
        {
          name: '发送',
          type: 'line',
          smooth: true,
          data: [],
          symbol: 'none',
          lineStyle: { width: 2, color: '#fa8c16' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(250, 140, 22, 0.2)' },
              { offset: 1, color: 'rgba(250, 140, 22, 0.02)' }
            ])
          }
        }
      ]
    }

    networkChart.setOption(option)
  }

  // 更新图表数据
  const updateChart = () => {
    if (!networkChart || !props.networkData?.trend || props.networkData.trend.length === 0) return

    let receiveData: Array<[string, number]>
    let transmitData: Array<[string, number]>
    let yAxisName: string

    if (chartDisplayMode.value === 'rate') {
      receiveData = calculateRateData('receive')
      transmitData = calculateRateData('transmit')
      yAxisName = '速率'
    } else {
      receiveData = props.networkData.trend.map((item) => [item.timestamp, item.receiveBytes])
      transmitData = props.networkData.trend.map((item) => [item.timestamp, item.transmitBytes])
      yAxisName = '累计流量'
    }

    // 计算 Y 轴范围
    const allValues = [
      ...receiveData.map((d) => d[1] as number),
      ...transmitData.map((d) => d[1] as number)
    ].filter((v) => !isNaN(v) && v >= 0)

    if (allValues.length === 0) return

    const maxValue = Math.max(...allValues)
    const minValue = Math.min(...allValues)
    const valueRange = maxValue - minValue

    let yMax: number
    let yMin: number

    if (valueRange > 0) {
      yMax = maxValue + valueRange * 0.1
      yMin = Math.max(0, minValue - valueRange * 0.1)
    } else {
      if (maxValue < 1024) {
        yMax = maxValue * 2 || 1024
        yMin = 0
      } else {
        yMax = maxValue * 1.2
        yMin = Math.max(0, maxValue * 0.8)
      }
    }

    networkChart.setOption({
      yAxis: { name: yAxisName, min: yMin, max: yMax },
      series: [{ data: receiveData }, { data: transmitData }]
    })
  }

  // 窗口尺寸变化处理
  const handleResize = () => {
    networkChart?.resize()
  }

  // 监听数据变化
  watch(
    () => props.networkData,
    () => {
      nextTick(() => {
        if (hasData.value) {
          if (!networkChart) initChart()
          updateChart()
        }
      })
    },
    { deep: true }
  )

  // 监听显示模式变化
  watch(chartDisplayMode, () => {
    if (hasData.value && networkChart) {
      updateChart()
    }
  })

  onMounted(() => {
    if (hasData.value) {
      initChart()
      updateChart()
    }
    window.addEventListener('resize', handleResize)

    if (networkChartRef.value) {
      resizeObserver = new ResizeObserver(handleResize)
      resizeObserver.observe(networkChartRef.value)
    }
  })

  onUnmounted(() => {
    window.removeEventListener('resize', handleResize)
    resizeObserver?.disconnect()
    networkChart?.dispose()
  })
</script>

<style lang="scss" scoped>
  $primary: #667eea;
  $success: #10b981;
  $warning: #f59e0b;
  $danger: #ef4444;
  $receive: #4facfe;
  $transmit: #fa8c16;
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

  .network-monitor {
    font-family: var(--ax-font-sans);

    .monitor-card {
      background: #fff;
      border-radius: $radius;
      padding: 24px;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
    }

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 24px;
      padding-bottom: 16px;
      border-bottom: 1px solid $gray-200;

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
            background: linear-gradient(135deg, $receive, #00f2fe);
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

      .status-tag {
        padding: 6px 14px;
        border-radius: 16px;
        font-size: 12px;
        font-weight: 600;

        &.success {
          background: rgba($success, 0.1);
          color: #059669;
        }

        &.danger {
          background: rgba($danger, 0.1);
          color: #dc2626;
        }
      }
    }

    .metrics-grid {
      display: grid;
      grid-template-columns: repeat(4, 1fr);
      gap: 14px;
      margin-bottom: 20px;

      @media (max-width: 1200px) {
        grid-template-columns: repeat(2, 1fr);
      }

      @media (max-width: 640px) {
        grid-template-columns: 1fr;
      }
    }

    .metric-box {
      padding: 16px;
      border-radius: $radius-sm;
      background: $gray-50;
      border: 1px solid $gray-200;
      transition: all 0.2s ease;

      &:hover {
        border-color: $gray-300;
      }

      &.primary {
        background: rgba($receive, 0.04);
        border-color: rgba($receive, 0.15);
      }

      &.danger {
        background: rgba($danger, 0.04);
        border-color: rgba($danger, 0.25);
      }

      .box-header {
        display: flex;
        align-items: center;
        gap: 6px;
        margin-bottom: 10px;
        color: $gray-500;

        span {
          font-size: 12px;
          font-weight: 500;
        }
      }

      .box-value {
        display: flex;
        align-items: baseline;
        gap: 4px;

        .value {
          font-size: 20px;
          font-weight: 700;
          color: $gray-800;

          &.receive {
            color: $receive;
          }

          &.transmit {
            color: $transmit;
          }

          &.danger {
            color: $danger;
          }
        }

        .unit {
          font-size: 12px;
          color: $gray-400;
        }
      }
    }

    .stats-bar {
      display: flex;
      gap: 12px;
      margin-bottom: 20px;

      .stats-item {
        flex: 1;
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 4px;
        padding: 12px;
        background: $gray-50;
        border-radius: $radius-xs;
        border: 1px solid $gray-200;

        &.receive {
          border-color: rgba($receive, 0.3);
        }

        &.transmit {
          border-color: rgba($transmit, 0.3);
        }

        .stats-label {
          font-size: 11px;
          color: $gray-500;
        }

        .stats-value {
          font-size: 14px;
          font-weight: 700;
          color: $gray-800;
        }
      }
    }

    .chart-area {
      padding: 18px;
      background: $gray-50;
      border-radius: $radius-sm;

      .chart-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 14px;
        flex-wrap: wrap;
        gap: 12px;

        .chart-title {
          font-size: 13px;
          font-weight: 600;
          color: $gray-600;
        }

        .chart-controls {
          .control-group {
            display: flex;
            background: #fff;
            border-radius: $radius-xs;
            padding: 2px;
            border: 1px solid $gray-200;

            .control-btn {
              padding: 6px 12px;
              font-size: 11px;
              font-weight: 500;
              color: $gray-500;
              background: transparent;
              border: none;
              border-radius: 4px;
              cursor: pointer;
              transition: all 0.2s ease;

              &:hover {
                color: $gray-700;
              }

              &.active {
                background: $primary;
                color: #fff;
              }
            }
          }
        }

        .chart-legend {
          display: flex;
          gap: 14px;

          .legend-item {
            display: flex;
            align-items: center;
            gap: 5px;
            font-size: 11px;
            color: $gray-500;

            .dot {
              width: 8px;
              height: 3px;
              border-radius: 2px;

              &.receive {
                background: $receive;
              }

              &.transmit {
                background: $transmit;
              }
            }
          }
        }

        .chart-info {
          font-size: 11px;
          color: $gray-400;
          padding: 4px 10px;
          background: #fff;
          border-radius: $radius-xs;
        }
      }

      .chart-container {
        height: 280px;
        width: 100%;
      }
    }

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
  }

  .fade-enter-active,
  .fade-leave-active {
    transition: opacity 0.3s ease;
  }

  .fade-enter-from,
  .fade-leave-to {
    opacity: 0;
  }

  @media (max-width: 768px) {
    .network-monitor {
      .monitor-card {
        padding: 16px;
      }

      .card-header {
        flex-direction: column;
        align-items: flex-start;
        gap: 12px;
      }

      .stats-bar {
        flex-wrap: wrap;

        .stats-item {
          min-width: calc(50% - 6px);
        }
      }

      .chart-area {
        padding: 14px;

        .chart-header {
          flex-direction: column;
          align-items: flex-start;
        }

        .chart-container {
          height: 220px;
        }
      }
    }
  }
</style>
