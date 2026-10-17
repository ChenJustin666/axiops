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
                  d="M12,2A10,10 0 0,1 22,12A10,10 0 0,1 12,22A10,10 0 0,1 2,12A10,10 0 0,1 12,2M12,4A8,8 0 0,0 4,12A8,8 0 0,0 12,20A8,8 0 0,0 20,12A8,8 0 0,0 12,4M12,6A6,6 0 0,1 18,12A6,6 0 0,1 12,18A6,6 0 0,1 6,12A6,6 0 0,1 12,6M12,8A4,4 0 0,0 8,12A4,4 0 0,0 12,16A4,4 0 0,0 16,12A4,4 0 0,0 12,8Z"
                />
              </svg>
            </div>
            <div>
              <h3>网络 I/O</h3>
              <p>网络接口流量监控</p>
            </div>
          </div>
          <div class="header-controls">
            <div class="interface-select">
              <select :value="selectedInterface" @change="handleInterfaceChange($event)">
                <option value="all">全部接口</option>
                <option v-for="iface in interfaces" :key="iface" :value="iface">{{ iface }}</option>
              </select>
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M7.41,8.58L12,13.17L16.59,8.58L18,10L12,16L6,10L7.41,8.58Z"
                />
              </svg>
            </div>
          </div>
        </div>

        <!-- 统计信息区域 -->
        <div class="stats-section">
          <div class="stat-item receive">
            <div class="stat-icon">
              <svg viewBox="0 0 24 24" width="18" height="18">
                <path
                  fill="currentColor"
                  d="M11,4H13V16L18.5,10.5L19.92,11.92L12,19.84L4.08,11.92L5.5,10.5L11,16V4Z"
                />
              </svg>
            </div>
            <div class="stat-info">
              <span class="stat-label">平均接收速率</span>
              <span class="stat-value">{{ formatBitRate(getAverageReceiveRate()) }}</span>
            </div>
          </div>
          <div class="stat-item transmit">
            <div class="stat-icon">
              <svg viewBox="0 0 24 24" width="18" height="18">
                <path
                  fill="currentColor"
                  d="M13,20H11V8L5.5,13.5L4.08,12.08L12,4.16L19.92,12.08L18.5,13.5L13,8V20Z"
                />
              </svg>
            </div>
            <div class="stat-info">
              <span class="stat-label">平均发送速率</span>
              <span class="stat-value">{{ formatBitRate(getAverageTransmitRate()) }}</span>
            </div>
          </div>
          <div class="stat-item" :class="{ error: getTotalReceiveErrors() > 0 }">
            <div class="stat-icon error">
              <svg viewBox="0 0 24 24" width="18" height="18">
                <path
                  fill="currentColor"
                  d="M12,2C17.53,2 22,6.47 22,12C22,17.53 17.53,22 12,22C6.47,22 2,17.53 2,12C2,6.47 6.47,2 12,2M15.59,7L12,10.59L8.41,7L7,8.41L10.59,12L7,15.59L8.41,17L12,13.41L15.59,17L17,15.59L13.41,12L17,8.41L15.59,7Z"
                />
              </svg>
            </div>
            <div class="stat-info">
              <span class="stat-label">接收错误</span>
              <span class="stat-value" :class="{ danger: getTotalReceiveErrors() > 0 }">{{
                getTotalReceiveErrors()
              }}</span>
            </div>
          </div>
          <div class="stat-item" :class="{ error: getTotalTransmitErrors() > 0 }">
            <div class="stat-icon error">
              <svg viewBox="0 0 24 24" width="18" height="18">
                <path
                  fill="currentColor"
                  d="M12,2C17.53,2 22,6.47 22,12C22,17.53 17.53,22 12,22C6.47,22 2,17.53 2,12C2,6.47 6.47,2 12,2M15.59,7L12,10.59L8.41,7L7,8.41L10.59,12L7,15.59L8.41,17L12,13.41L15.59,17L17,15.59L13.41,12L17,8.41L15.59,7Z"
                />
              </svg>
            </div>
            <div class="stat-info">
              <span class="stat-label">发送错误</span>
              <span class="stat-value" :class="{ danger: getTotalTransmitErrors() > 0 }">{{
                getTotalTransmitErrors()
              }}</span>
            </div>
          </div>
        </div>

        <!-- 图表区域 -->
        <div class="chart-area">
          <div class="chart-header">
            <span class="chart-title">网络速率趋势</span>
            <div class="chart-legend">
              <span class="legend-item"><i class="dot receive"></i>接收</span>
              <span class="legend-item"><i class="dot transmit"></i>发送</span>
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
              d="M12,2A10,10 0 0,1 22,12A10,10 0 0,1 12,22A10,10 0 0,1 2,12A10,10 0 0,1 12,2M12,4A8,8 0 0,0 4,12A8,8 0 0,0 12,20A8,8 0 0,0 20,12A8,8 0 0,0 12,4Z"
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
  import { formatBitRate, smartFormatTime } from '@/utils/format'
  import type { NodeNetworkMetrics, NodeNetworkInterfaceMetrics } from '@/api/console/node-monitor'

  interface Props {
    networkData?: NodeNetworkMetrics | NodeNetworkInterfaceMetrics
    loading?: boolean
    interfaces?: string[]
    selectedInterface?: string
  }

  const props = withDefaults(defineProps<Props>(), {
    networkData: undefined,
    loading: false,
    interfaces: () => [],
    selectedInterface: 'all'
  })

  interface Emits {
    (e: 'interface-change', interfaceName: string): void
  }

  const emit = defineEmits<Emits>()

  // 图表容器引用
  const networkChartRef = ref<HTMLElement>()
  // 图表实例
  let networkChart: echarts.ECharts | null = null
  // 标记图表是否已初始化
  let chartInitialized = false

  // 是否有数据
  const hasData = computed(() => {
    return !!(props.networkData && Object.keys(props.networkData).length > 0)
  })

  // 转换时间戳为毫秒
  const convertTimestamp = (timestamp: any): number => {
    if (typeof timestamp === 'string') {
      return new Date(timestamp).getTime()
    } else if (typeof timestamp === 'number') {
      if (timestamp < 10000000000) {
        return timestamp * 1000
      }
      return timestamp
    }
    return Date.now()
  }

  // 处理接口切换
  const handleInterfaceChange = (event: Event) => {
    const target = event.target as HTMLSelectElement
    emit('interface-change', target.value)
  }

  // 获取平均接收速率
  const getAverageReceiveRate = (): number => {
    if (!props.networkData) return 0
    if ('interfaces' in props.networkData) {
      return props.networkData.interfaces.reduce(
        (sum, i) => sum + i.summary.avgReceiveBytesPerSec,
        0
      )
    }
    return (props.networkData as NodeNetworkInterfaceMetrics).summary.avgReceiveBytesPerSec || 0
  }

  // 获取平均发送速率
  const getAverageTransmitRate = (): number => {
    if (!props.networkData) return 0
    if ('interfaces' in props.networkData) {
      return props.networkData.interfaces.reduce(
        (sum, i) => sum + i.summary.avgTransmitBytesPerSec,
        0
      )
    }
    return (props.networkData as NodeNetworkInterfaceMetrics).summary.avgTransmitBytesPerSec || 0
  }

  // 获取接收错误总数
  const getTotalReceiveErrors = (): number => {
    if (!props.networkData) return 0
    if ('interfaces' in props.networkData) {
      return props.networkData.interfaces.reduce((sum, i) => sum + i.summary.totalReceiveErrors, 0)
    }
    return (props.networkData as NodeNetworkInterfaceMetrics).summary.totalReceiveErrors || 0
  }

  // 获取发送错误总数
  const getTotalTransmitErrors = (): number => {
    if (!props.networkData) return 0
    if ('interfaces' in props.networkData) {
      return props.networkData.interfaces.reduce((sum, i) => sum + i.summary.totalTransmitErrors, 0)
    }
    return (props.networkData as NodeNetworkInterfaceMetrics).summary.totalTransmitErrors || 0
  }

  // 初始化图表
  const initChart = () => {
    if (!networkChartRef.value || chartInitialized) return

    networkChart = echarts.init(networkChartRef.value)
    chartInitialized = true

    const option: echarts.EChartsOption = {
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'cross' },
        backgroundColor: 'rgba(255,255,255,0.98)',
        borderColor: '#eaecef',
        borderWidth: 1,
        padding: [12, 16],
        textStyle: { color: '#1f2937', fontSize: 12 },
        formatter: (params: any) => {
          if (!Array.isArray(params)) return ''
          const timestamp = params[0].value[0]
          let result = `<div style="font-weight: 600; margin-bottom: 8px;">${smartFormatTime(timestamp)}</div>`
          params.forEach((item: any) => {
            result += `<div style="margin: 4px 0; display: flex; align-items: center;">
              <span style="display: inline-block; width: 8px; height: 8px; border-radius: 50%; background: ${item.color}; margin-right: 8px;"></span>
              <span style="color: #6b7280;">${item.seriesName}: </span>
              <span style="font-weight: 600; margin-left: 4px;">${formatBitRate(item.value[1])}</span>
            </div>`
          })
          return result
        }
      },
      legend: {
        show: false
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
          formatter: (value: number) => formatBitRate(value),
          color: '#6b7280',
          fontSize: 11
        },
        splitLine: { lineStyle: { color: '#f3f4f6', type: 'dashed' } }
      },
      series: []
    }

    networkChart.setOption(option)
  }

  // 更新图表数据
  const updateChart = () => {
    if (!networkChart || !props.networkData) return

    let receiveData: Array<[number, number]> = []
    let transmitData: Array<[number, number]> = []

    if ('interfaces' in props.networkData) {
      // 聚合所有接口数据
      const allTimestamps = new Set<string>()
      props.networkData.interfaces.forEach((iface) => {
        iface.trend.forEach((point) => allTimestamps.add(point.timestamp))
      })

      Array.from(allTimestamps)
        .sort()
        .forEach((timestamp) => {
          let totalReceive = 0
          let totalTransmit = 0
          props.networkData.interfaces.forEach((iface) => {
            const point = iface.trend.find((p) => p.timestamp === timestamp)
            if (point) {
              totalReceive += point.receiveBytesPerSec
              totalTransmit += point.transmitBytesPerSec
            }
          })
          receiveData.push([convertTimestamp(timestamp), totalReceive])
          transmitData.push([convertTimestamp(timestamp), totalTransmit])
        })
    } else {
      // 单个接口数据
      receiveData = props.networkData.trend.map((item) => [
        convertTimestamp(item.timestamp),
        item.receiveBytesPerSec
      ])
      transmitData = props.networkData.trend.map((item) => [
        convertTimestamp(item.timestamp),
        item.transmitBytesPerSec
      ])
    }

    networkChart.setOption({
      series: [
        {
          name: '接收',
          type: 'line',
          smooth: true,
          data: receiveData,
          symbol: 'none',
          lineStyle: { width: 2, color: '#0ea5e9' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(14, 165, 233, 0.2)' },
              { offset: 1, color: 'rgba(14, 165, 233, 0.02)' }
            ])
          }
        },
        {
          name: '发送',
          type: 'line',
          smooth: true,
          data: transmitData,
          symbol: 'none',
          lineStyle: { width: 2, color: '#f59e0b' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(245, 158, 11, 0.2)' },
              { offset: 1, color: 'rgba(245, 158, 11, 0.02)' }
            ])
          }
        }
      ]
    })
  }

  // 监听数据变化
  watch(
    () => props.networkData,
    () => {
      nextTick(() => {
        if (hasData.value) {
          if (!chartInitialized) {
            initChart()
          }
          updateChart()
        }
      })
    },
    { deep: true, immediate: true }
  )

  // 窗口大小变化时调整图表
  const handleResize = () => {
    networkChart?.resize()
  }

  onMounted(() => {
    window.addEventListener('resize', handleResize)
  })

  onUnmounted(() => {
    window.removeEventListener('resize', handleResize)
    if (networkChart) {
      networkChart.dispose()
      networkChart = null
    }
    chartInitialized = false
  })
</script>

<style lang="scss" scoped>
  // 颜色变量
  $primary: #3b82f6;
  $success: #10b981;
  $warning: #f59e0b;
  $danger: #ef4444;
  $info: #0ea5e9;
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
    .monitor-card {
      background: #fff;
      border-radius: $radius;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
      overflow: hidden;
    }

    // 卡片头部
    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 20px 24px;
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
            background: linear-gradient(135deg, $info, #0284c7);
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

      .header-controls {
        .interface-select {
          position: relative;
          display: flex;
          align-items: center;

          select {
            appearance: none;
            padding: 8px 32px 8px 14px;
            background: $gray-50;
            border: 1px solid $gray-200;
            border-radius: $radius-xs;
            font-size: 13px;
            color: $gray-700;
            cursor: pointer;
            transition: all 0.2s ease;

            &:hover {
              border-color: $gray-300;
            }

            &:focus {
              outline: none;
              border-color: $primary;
              box-shadow: 0 0 0 3px rgba($primary, 0.1);
            }
          }

          svg {
            position: absolute;
            right: 10px;
            pointer-events: none;
            color: $gray-400;
          }
        }
      }
    }

    // 统计信息区域
    .stats-section {
      display: grid;
      grid-template-columns: repeat(4, 1fr);
      gap: 14px;
      padding: 20px 24px;
      background: $gray-50;
      border-bottom: 1px solid $gray-200;

      @media (max-width: 1200px) {
        grid-template-columns: repeat(2, 1fr);
      }

      @media (max-width: 640px) {
        grid-template-columns: 1fr;
      }

      .stat-item {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 14px 16px;
        background: #fff;
        border-radius: $radius-sm;
        border: 1px solid $gray-200;
        transition: all 0.2s ease;

        &:hover {
          border-color: $gray-300;
        }

        &.receive {
          border-left: 3px solid $info;

          .stat-icon {
            background: rgba($info, 0.1);
            color: $info;
          }
        }

        &.transmit {
          border-left: 3px solid $warning;

          .stat-icon {
            background: rgba($warning, 0.1);
            color: $warning;
          }
        }

        &.error {
          border-left: 3px solid $danger;
        }

        .stat-icon {
          width: 36px;
          height: 36px;
          border-radius: $radius-xs;
          display: flex;
          align-items: center;
          justify-content: center;
          flex-shrink: 0;

          &.error {
            background: rgba($danger, 0.1);
            color: $danger;
          }
        }

        .stat-info {
          display: flex;
          flex-direction: column;
          gap: 4px;

          .stat-label {
            font-size: 12px;
            color: $gray-500;
            font-weight: 500;
          }

          .stat-value {
            font-size: 16px;
            font-weight: 700;
            color: $gray-800;

            &.danger {
              color: $danger;
            }
          }
        }
      }
    }

    // 图表区域
    .chart-area {
      padding: 20px 24px;

      .chart-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 16px;

        .chart-title {
          font-size: 14px;
          font-weight: 600;
          color: $gray-700;
        }

        .chart-legend {
          display: flex;
          gap: 16px;

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

              &.receive {
                background: $info;
              }

              &.transmit {
                background: $warning;
              }
            }
          }
        }
      }

      .chart-container {
        height: 320px;
        width: 100%;
      }
    }

    // 空状态
    .empty-state {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      min-height: 400px;
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

  // 动画
  .fade-enter-active,
  .fade-leave-active {
    transition: opacity 0.3s ease;
  }

  .fade-enter-from,
  .fade-leave-to {
    opacity: 0;
  }

  // 响应式适配
  @media (max-width: 768px) {
    .network-monitor {
      .card-header {
        flex-direction: column;
        align-items: flex-start;
        gap: 14px;
      }

      .chart-area {
        padding: 16px;

        .chart-container {
          height: 260px;
        }
      }
    }
  }
</style>
