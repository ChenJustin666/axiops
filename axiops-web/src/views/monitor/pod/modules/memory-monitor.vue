<template>
  <div class="memory-monitor" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="monitor-card">
        <!-- 卡片头部 -->
        <div class="card-header">
          <div class="header-info">
            <div class="icon-badge memory">
              <svg viewBox="0 0 24 24" width="22" height="22">
                <path
                  fill="currentColor"
                  d="M17,9H7V7H17M17,13H7V11H17M14,17H7V15H14M12,3A1,1 0 0,1 13,4A1,1 0 0,1 12,5A1,1 0 0,1 11,4A1,1 0 0,1 12,3M19,3H14.82C14.4,1.84 13.3,1 12,1C10.7,1 9.6,1.84 9.18,3H5A2,2 0 0,0 3,5V19A2,2 0 0,0 5,21H19A2,2 0 0,0 21,19V5A2,2 0 0,0 19,3Z"
                />
              </svg>
            </div>
            <div>
              <h3>内存使用情况</h3>
              <p v-if="containerName">容器: {{ containerName }}</p>
              <p v-else>Pod 级别监控</p>
            </div>
          </div>
          <div class="status-tag" :class="memoryStatusClass">{{ memoryStatusText }}</div>
        </div>

        <!-- 指标网格 -->
        <div class="metrics-grid">
          <div class="metric-box">
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M16,13V11H21V13H16M16,18V16H21V18H16M16,8V6H21V8H16M3,18H13V16H3V18M3,13H13V11H3V13M3,8H13V6H3V8Z"
                />
              </svg>
              <span>平均使用</span>
            </div>
            <div class="box-value">
              <span class="value">{{ formatBytes(memoryData?.summary?.avgUsageBytes || 0) }}</span>
            </div>
          </div>

          <div class="metric-box" :class="{ warning: isPeakHigh }">
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M3,13H5V11H3V13M3,17H5V15H3V17M3,9H5V7H3V9M7,13H21V11H7V13M7,17H21V15H7V17M7,7V9H21V7H7Z"
                />
              </svg>
              <span>峰值</span>
            </div>
            <div class="box-value">
              <span class="value">{{ formatBytes(memoryData?.summary?.maxUsageBytes || 0) }}</span>
            </div>
          </div>

          <div class="metric-box">
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M12,20A8,8 0 0,1 4,12A8,8 0 0,1 12,4A8,8 0 0,1 20,12A8,8 0 0,1 12,20M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2Z"
                />
              </svg>
              <span>最小值</span>
            </div>
            <div class="box-value">
              <span class="value">{{ formatBytes(memoryData?.summary?.minUsageBytes || 0) }}</span>
            </div>
          </div>

          <div class="metric-box primary">
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M13,2.05V5.08C16.39,5.57 19,8.47 19,12C19,15.87 15.87,19 12,19C8.47,19 5.57,16.39 5.08,13H2.05C2.56,17.95 6.81,21.86 11.95,21.99C17.19,22.14 21.53,17.94 21.96,12.77C22.4,7.47 18.36,2.8 13,2.05Z"
                />
              </svg>
              <span>平均使用率</span>
            </div>
            <div class="box-value">
              <span class="value">{{
                memoryData?.summary?.avgUsagePercent?.toFixed(2) || '0.00'
              }}</span>
              <span class="unit">%</span>
            </div>
            <div
              class="box-badge"
              :class="getUsageLevelClass(memoryData?.summary?.avgUsagePercent)"
            >
              {{ getUsageLevelText(memoryData?.summary?.avgUsagePercent) }}
            </div>
            <div class="box-progress">
              <div
                class="progress-fill"
                :style="{ width: `${Math.min(memoryData?.summary?.avgUsagePercent || 0, 100)}%` }"
                :class="getProgressClass(memoryData?.summary?.avgUsagePercent)"
              ></div>
            </div>
          </div>
        </div>

        <!-- 资源限制统计 -->
        <div class="stats-bar">
          <div class="stats-item info">
            <span class="stats-label">Request</span>
            <span class="stats-value">{{
              formatBytes(memoryData?.limits?.requestBytes || 0)
            }}</span>
          </div>
          <div class="stats-item warning">
            <span class="stats-label">Limit</span>
            <span class="stats-value">
              {{
                memoryData?.limits?.hasLimit
                  ? formatBytes(memoryData?.limits?.limitBytes || 0)
                  : '无限制'
              }}
            </span>
          </div>
          <div
            v-if="memoryData?.summary?.oomKills && memoryData.summary.oomKills > 0"
            class="stats-item danger"
          >
            <span class="stats-label">OOM Kill</span>
            <span class="stats-value">{{ memoryData.summary.oomKills }} 次</span>
          </div>
        </div>

        <!-- 图表区域 -->
        <div class="chart-area">
          <div class="chart-header">
            <span class="chart-title">内存使用趋势</span>
            <div class="chart-legend">
              <span class="legend-item"><i class="dot usage"></i>使用量</span>
              <span class="legend-item"><i class="dot workingset"></i>WorkingSet</span>
              <span class="legend-item"><i class="dot percent"></i>使用率 (%)</span>
            </div>
            <div v-if="memoryData?.trend" class="chart-info">
              共 {{ memoryData.trend.length }} 个数据点
            </div>
          </div>
          <div ref="memoryChartRef" class="chart-container"></div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" width="64" height="64">
            <path
              fill="currentColor"
              d="M17,9H7V7H17M17,13H7V11H17M14,17H7V15H14M12,3A1,1 0 0,1 13,4A1,1 0 0,1 12,5A1,1 0 0,1 11,4A1,1 0 0,1 12,3M19,3H14.82C14.4,1.84 13.3,1 12,1C10.7,1 9.6,1.84 9.18,3H5A2,2 0 0,0 3,5V19A2,2 0 0,0 5,21H19A2,2 0 0,0 21,19V5A2,2 0 0,0 19,3Z"
            />
          </svg>
        </div>
        <h3>暂无内存监控数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, onUnmounted, nextTick, computed } from 'vue'
  import * as echarts from 'echarts'
  import type { PodMemoryMetrics, ContainerMemoryMetrics } from '@/api/console/pod-monitor'

  interface Props {
    memoryData?: PodMemoryMetrics | ContainerMemoryMetrics
    loading?: boolean
    containerName?: string
  }

  const props = withDefaults(defineProps<Props>(), {
    memoryData: undefined,
    loading: false,
    containerName: undefined
  })

  // 图表相关
  const memoryChartRef = ref<HTMLElement>()
  let memoryChart: echarts.ECharts | null = null
  let resizeObserver: ResizeObserver | null = null

  // 数据状态
  const hasData = computed(() => {
    return !!(props.memoryData && Object.keys(props.memoryData).length > 0)
  })

  // 峰值是否过高
  const isPeakHigh = computed(() => {
    const peak = props.memoryData?.summary?.maxUsageBytes || 0
    const limit = props.memoryData?.limits?.limitBytes || 0
    return limit > 0 && peak > limit * 0.8
  })

  // 内存状态
  const memoryStatusClass = computed(() => {
    const percent = props.memoryData?.summary?.avgUsagePercent || 0
    const oomKills = props.memoryData?.summary?.oomKills || 0
    if (oomKills > 0 || percent > 90) return 'danger'
    if (percent > 70) return 'warning'
    return 'success'
  })

  const memoryStatusText = computed(() => {
    const percent = props.memoryData?.summary?.avgUsagePercent || 0
    const oomKills = props.memoryData?.summary?.oomKills || 0
    if (oomKills > 0 || percent > 90) return '异常'
    if (percent > 70) return '需关注'
    return '正常'
  })

  // 使用率等级
  const getUsageLevelClass = (percent?: number): string => {
    if (!percent) return 'low'
    if (percent < 50) return 'low'
    if (percent < 70) return 'normal'
    if (percent < 90) return 'high'
    return 'very-high'
  }

  const getUsageLevelText = (percent?: number): string => {
    if (!percent) return '低负载'
    if (percent < 50) return '低负载'
    if (percent < 70) return '正常'
    if (percent < 90) return '偏高'
    return '过高'
  }

  const getProgressClass = (percent?: number): string => {
    if (!percent) return ''
    if (percent < 50) return 'low'
    if (percent < 70) return 'normal'
    if (percent < 90) return 'high'
    return 'danger'
  }

  // 字节格式化
  const formatBytes = (bytes: number): string => {
    if (!bytes || bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
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

  // 初始化图表
  const initChart = () => {
    if (!memoryChartRef.value) return

    memoryChart = echarts.init(memoryChartRef.value)

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
              item.seriesName === '使用率 (%)'
                ? `${item.value[1].toFixed(2)}%`
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
      yAxis: [
        {
          type: 'value',
          name: '内存',
          position: 'left',
          nameTextStyle: { color: '#6b7280', fontSize: 11 },
          axisLine: { show: false },
          axisTick: { show: false },
          axisLabel: {
            color: '#6b7280',
            fontSize: 11,
            formatter: (value: number) => formatBytes(value)
          },
          splitLine: { lineStyle: { color: '#f3f4f6', type: 'dashed' } }
        },
        {
          type: 'value',
          name: '使用率 (%)',
          position: 'right',
          nameTextStyle: { color: '#6b7280', fontSize: 11 },
          axisLine: { show: false },
          axisTick: { show: false },
          axisLabel: { color: '#6b7280', fontSize: 11, formatter: '{value}%' },
          min: 0,
          splitLine: { show: false }
        }
      ],
      series: [
        {
          name: '使用量',
          type: 'line',
          smooth: true,
          data: [],
          symbol: 'none',
          lineStyle: { width: 2, color: '#f5576c' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(245, 87, 108, 0.2)' },
              { offset: 1, color: 'rgba(245, 87, 108, 0.02)' }
            ])
          }
        },
        {
          name: 'WorkingSet',
          type: 'line',
          smooth: true,
          data: [],
          symbol: 'none',
          lineStyle: { width: 2, color: '#fa8c16' }
        },
        {
          name: '使用率 (%)',
          type: 'line',
          smooth: true,
          yAxisIndex: 1,
          data: [],
          symbol: 'none',
          lineStyle: { width: 2, color: '#667eea' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(102, 126, 234, 0.2)' },
              { offset: 1, color: 'rgba(102, 126, 234, 0.02)' }
            ])
          }
        }
      ]
    }

    memoryChart.setOption(option)
  }

  // 更新图表数据
  const updateChart = () => {
    if (!memoryChart || !props.memoryData?.trend || props.memoryData.trend.length === 0) return

    const usageData = props.memoryData.trend.map((item) => [item.timestamp, item.usageBytes])
    const workingSetData = props.memoryData.trend.map((item) => [
      item.timestamp,
      item.workingSetBytes
    ])
    const percentData = props.memoryData.trend.map((item) => [item.timestamp, item.usagePercent])

    // 计算使用率 Y 轴范围
    const percentValues = percentData.map((d) => d[1] as number).filter((v) => !isNaN(v) && v >= 0)
    const maxPercent = percentValues.length > 0 ? Math.max(...percentValues) : 1

    let percentMax: number
    if (maxPercent <= 0.1) percentMax = 0.5
    else if (maxPercent <= 1) percentMax = 1
    else if (maxPercent <= 5) percentMax = 5
    else if (maxPercent <= 10) percentMax = 10
    else if (maxPercent <= 20) percentMax = 20
    else if (maxPercent <= 50) percentMax = 50
    else percentMax = 100

    memoryChart.setOption({
      yAxis: [
        { min: 'dataMin', max: 'dataMax' },
        { min: 0, max: percentMax }
      ],
      series: [{ data: usageData }, { data: workingSetData }, { data: percentData }]
    })
  }

  // 窗口尺寸变化处理
  const handleResize = () => {
    memoryChart?.resize()
  }

  // 监听数据变化
  watch(
    () => props.memoryData,
    (newData) => {
      if (newData && newData.trend && newData.trend.length > 0) {
        nextTick(() => {
          if (!memoryChart) initChart()
          updateChart()
        })
      }
    },
    { deep: true }
  )

  onMounted(() => {
    if (hasData.value) {
      initChart()
      updateChart()
    }
    window.addEventListener('resize', handleResize)

    // 监听容器尺寸变化
    if (memoryChartRef.value) {
      resizeObserver = new ResizeObserver(handleResize)
      resizeObserver.observe(memoryChartRef.value)
    }
  })

  onUnmounted(() => {
    window.removeEventListener('resize', handleResize)
    resizeObserver?.disconnect()
    memoryChart?.dispose()
  })
</script>

<style lang="scss" scoped>
  $primary: #667eea;
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

  .memory-monitor {
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

          &.memory {
            background: linear-gradient(135deg, #f093fb, #f5576c);
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

        &.warning {
          background: rgba($warning, 0.1);
          color: #d97706;
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
        background: rgba(#f093fb, 0.04);
        border-color: rgba(#f093fb, 0.15);
      }

      &.warning {
        background: rgba($warning, 0.06);
        border-color: rgba($warning, 0.25);
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
        }

        .unit {
          font-size: 12px;
          color: $gray-400;
        }
      }

      .box-badge {
        display: inline-block;
        padding: 4px 10px;
        border-radius: 12px;
        font-size: 11px;
        font-weight: 600;
        margin-top: 8px;

        &.low {
          background: rgba($primary, 0.1);
          color: #2563eb;
        }

        &.normal {
          background: rgba($success, 0.1);
          color: #059669;
        }

        &.high {
          background: rgba($warning, 0.1);
          color: #d97706;
        }

        &.very-high {
          background: rgba($danger, 0.1);
          color: #dc2626;
        }
      }

      .box-progress {
        margin-top: 10px;
        height: 4px;
        background: $gray-200;
        border-radius: 2px;
        overflow: hidden;

        .progress-fill {
          height: 100%;
          background: #f093fb;
          transition: width 0.4s ease;

          &.low {
            background: $primary;
          }

          &.normal {
            background: $success;
          }

          &.high {
            background: $warning;
          }

          &.danger {
            background: $danger;
          }
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

        &.info {
          border-color: rgba($primary, 0.3);
        }

        &.warning {
          border-color: rgba($warning, 0.3);
        }

        &.danger {
          border-color: rgba($danger, 0.3);
          background: rgba($danger, 0.04);
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

              &.usage {
                background: #f5576c;
              }

              &.workingset {
                background: #fa8c16;
              }

              &.percent {
                background: $primary;
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
    .memory-monitor {
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

        .chart-container {
          height: 220px;
        }
      }
    }
  }
</style>
