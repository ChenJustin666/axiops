<template>
  <div class="memory-monitor" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="monitor-card">
        <!-- 卡片头部区域 -->
        <div class="card-header">
          <div class="header-info">
            <div class="icon-badge memory">
              <svg viewBox="0 0 24 24" width="22" height="22">
                <path
                  fill="currentColor"
                  d="M2 20h20v-4H2v4zm2-3h2v2H4v-2zM2 4v4h20V4H2zm4 3H4V5h2v2zm-4 7h20v-4H2v4zm2-3h2v2H4v-2z"
                />
              </svg>
            </div>
            <div>
              <h3>内存使用情况</h3>
              <p>内存与交换空间监控</p>
            </div>
          </div>
          <div class="status-tag" :class="memoryStatusClass">{{ memoryStatusText }}</div>
        </div>

        <!-- 核心指标网格 -->
        <div class="metrics-grid">
          <div class="metric-box primary">
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M13 2.05v2.02c3.95.49 7 3.85 7 7.93s-3.05 7.44-7 7.93v2.02c5.05-.5 9-4.76 9-9.95s-3.95-9.45-9-9.95zm-2 0c-5.05.5-9 4.76-9 9.95s3.95 9.45 9 9.95v-2.02c-3.95-.49-7-3.85-7-7.93s3.05-7.44 7-7.93V2.05z"
                />
              </svg>
              <span>平均使用率</span>
            </div>
            <div class="box-value">
              <span class="value">{{
                formatPercentageValue(memoryData?.summary?.avgUsagePercent)
              }}</span>
              <span class="unit">%</span>
            </div>
            <div class="box-badge" :class="getUsageLevel(memoryData?.summary?.avgUsagePercent)">
              {{ getUsageLevelText(memoryData?.summary?.avgUsagePercent) }}
            </div>
            <div class="box-progress">
              <div
                class="progress-fill"
                :style="{ width: (memoryData?.summary?.avgUsagePercent || 0) + '%' }"
              ></div>
            </div>
          </div>

          <div
            class="metric-box"
            :class="{ warning: (memoryData?.summary?.maxUsagePercent || 0) > 85 }"
          >
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path fill="currentColor" d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z" />
              </svg>
              <span>峰值使用率</span>
            </div>
            <div class="box-value">
              <span class="value">{{
                formatPercentageValue(memoryData?.summary?.maxUsagePercent)
              }}</span>
              <span class="unit">%</span>
            </div>
          </div>

          <div class="metric-box">
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M19 3H5c-1.11 0-2 .9-2 2v14c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14h-2v-2h2v2zm0-4h-2V7h2v6z"
                />
              </svg>
              <span>已用内存</span>
            </div>
            <div class="box-value">
              <span class="value">{{ formatBytesCompact(memoryData?.current?.usedBytes) }}</span>
            </div>
          </div>

          <div class="metric-box">
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M17 3H7c-1.1 0-2 .9-2 2v16l7-3 7 3V5c0-1.1-.9-2-2-2zm0 15l-5-2.18L7 18V5h10v13z"
                />
              </svg>
              <span>总内存</span>
            </div>
            <div class="box-value">
              <span class="value">{{ formatBytesCompact(memoryData?.current?.totalBytes) }}</span>
            </div>
          </div>
        </div>

        <!-- 内存统计条 -->
        <div class="stats-bar">
          <div class="stats-item success">
            <span class="stats-label">可用</span>
            <span class="stats-value">{{ formatBytes(memoryData?.current?.availableBytes) }}</span>
          </div>
          <div class="stats-item">
            <span class="stats-label">Free</span>
            <span class="stats-value">{{ formatBytes(memoryData?.current?.freeBytes) }}</span>
          </div>
          <div class="stats-item info">
            <span class="stats-label">Cached</span>
            <span class="stats-value">{{ formatBytes(memoryData?.current?.cachedBytes) }}</span>
          </div>
          <div
            v-if="memoryData?.current?.swapUsagePercent && memoryData.current.swapUsagePercent > 0"
            class="stats-item danger"
          >
            <span class="stats-label">Swap 使用</span>
            <span class="stats-value">{{
              formatPercentage(memoryData.current.swapUsagePercent)
            }}</span>
          </div>
        </div>

        <!-- 趋势图表区域 -->
        <div class="chart-area">
          <div class="chart-header">
            <span class="chart-title">使用趋势</span>
            <div class="chart-legend">
              <span class="legend-item"><i class="dot used"></i>已用内存</span>
              <span class="legend-item"><i class="dot available"></i>可用内存</span>
              <span class="legend-item"><i class="dot percent"></i>使用率 (%)</span>
            </div>
          </div>
          <div ref="memoryChartRef" class="chart-container"></div>
        </div>

        <!-- 可折叠的详细信息面板 -->
        <div class="detail-panel">
          <div class="panel-toggle" @click="toggleDetail">
            <span>内存详细指标</span>
            <svg :class="{ rotated: showDetail }" viewBox="0 0 24 24" width="16" height="16">
              <path
                fill="currentColor"
                d="M7.41 8.59L12 13.17l4.59-4.58L18 10l-6 6-6-6 1.41-1.41z"
              />
            </svg>
          </div>
          <Transition name="slide">
            <div v-if="showDetail" class="panel-content">
              <div class="detail-section">
                <div class="section-title">内存分布</div>
                <div class="detail-grid">
                  <div class="detail-item">
                    <span class="label">Used</span>
                    <span class="value">{{ formatBytes(memoryData?.current?.usedBytes) }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="label">Free</span>
                    <span class="value">{{ formatBytes(memoryData?.current?.freeBytes) }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="label">Available</span>
                    <span class="value">{{
                      formatBytes(memoryData?.current?.availableBytes)
                    }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="label">Buffers</span>
                    <span class="value">{{ formatBytes(memoryData?.current?.buffersBytes) }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="label">Cached</span>
                    <span class="value">{{ formatBytes(memoryData?.current?.cachedBytes) }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="label">Active</span>
                    <span class="value">{{ formatBytes(memoryData?.current?.activeBytes) }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="label">Inactive</span>
                    <span class="value">{{ formatBytes(memoryData?.current?.inactiveBytes) }}</span>
                  </div>
                </div>
              </div>

              <div v-if="memoryData?.current?.swapTotalBytes" class="detail-section">
                <div class="section-title">Swap 信息</div>
                <div class="detail-grid">
                  <div class="detail-item">
                    <span class="label">Swap Total</span>
                    <span class="value">{{ formatBytes(memoryData.current.swapTotalBytes) }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="label">Swap Used</span>
                    <span
                      class="value"
                      :class="{ warning: (memoryData.current.swapUsedBytes || 0) > 0 }"
                    >
                      {{ formatBytes(memoryData.current.swapUsedBytes) }}
                    </span>
                  </div>
                  <div class="detail-item">
                    <span class="label">Swap Free</span>
                    <span class="value">{{ formatBytes(memoryData.current.swapFreeBytes) }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="label">Swap In Rate</span>
                    <span class="value">{{ formatBytes(memoryData.current.swapInRate) }}/s</span>
                  </div>
                  <div class="detail-item">
                    <span class="label">Swap Out Rate</span>
                    <span class="value">{{ formatBytes(memoryData.current.swapOutRate) }}/s</span>
                  </div>
                </div>
              </div>
            </div>
          </Transition>
        </div>
      </div>

      <!-- 空状态展示 -->
      <div v-else class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" width="64" height="64">
            <path
              fill="currentColor"
              d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14h-2V9h2v8zm0-10h-2V5h2v2z"
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
  import { formatBytes, formatPercentage, smartFormatTime } from '@/utils/format'
  import type { NodeMemoryMetrics } from '@/api/console/node-monitor'

  interface Props {
    memoryData?: NodeMemoryMetrics
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    memoryData: undefined,
    loading: false
  })

  // 图表 DOM 引用
  const memoryChartRef = ref<HTMLElement>()
  // ECharts 实例
  let memoryChart: echarts.ECharts | null = null
  // 详情面板展开状态
  const showDetail = ref(false)
  // 图表初始化标记
  let chartInitialized = false

  // 计算属性：判断是否有有效数据
  const hasData = computed(() => {
    return !!(props.memoryData && Object.keys(props.memoryData).length > 0)
  })

  // 计算属性：根据内存使用率返回状态样式类
  const memoryStatusClass = computed(() => {
    const usage = props.memoryData?.summary?.avgUsagePercent || 0
    if (usage > 90) return 'danger'
    if (usage > 75) return 'warning'
    return 'success'
  })

  // 计算属性：根据内存使用率返回状态文本
  const memoryStatusText = computed(() => {
    const usage = props.memoryData?.summary?.avgUsagePercent || 0
    if (usage > 90) return '内存紧张'
    if (usage > 75) return '内存较高'
    return '正常'
  })

  // 切换详情面板展开/收起状态
  const toggleDetail = () => {
    showDetail.value = !showDetail.value
  }

  // 格式化百分比数值，返回不带 % 符号的字符串
  const formatPercentageValue = (value?: number): string => {
    if (value === undefined || value === null || isNaN(value)) return '0.00'
    return value.toFixed(2)
  }

  // 格式化字节数为紧凑格式（用于标题展示）
  const formatBytesCompact = (bytes?: number): string => {
    if (!bytes || bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
  }

  // 根据使用率返回对应的等级样式类
  const getUsageLevel = (value?: number): string => {
    if (!value) return 'low'
    if (value < 50) return 'low'
    if (value < 75) return 'normal'
    if (value < 90) return 'high'
    return 'critical'
  }

  // 根据使用率返回对应的等级文本
  const getUsageLevelText = (value?: number): string => {
    if (!value) return '充裕'
    if (value < 50) return '充裕'
    if (value < 75) return '正常'
    if (value < 90) return '较高'
    return '紧张'
  }

  // 将各种格式的时间戳统一转换为毫秒级时间戳
  const convertTimestamp = (timestamp: any): number => {
    if (typeof timestamp === 'string') {
      return new Date(timestamp).getTime()
    } else if (typeof timestamp === 'number') {
      // 判断是否为秒级时间戳
      if (timestamp < 10000000000) {
        return timestamp * 1000
      }
      return timestamp
    }
    return Date.now()
  }

  // 初始化 ECharts 图表实例
  const initChart = () => {
    if (!memoryChartRef.value || chartInitialized) return

    memoryChart = echarts.init(memoryChartRef.value)
    chartInitialized = true

    const option: echarts.EChartsOption = {
      tooltip: {
        trigger: 'axis',
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
            const value =
              item.seriesName === '使用率 (%)'
                ? `${item.value[1].toFixed(2)}%`
                : formatBytes(item.value[1])
            result += `<div style="margin: 4px 0; display: flex; align-items: center;">
              <span style="display: inline-block; width: 8px; height: 3px; border-radius: 2px; background: ${item.color}; margin-right: 8px;"></span>
              <span style="color: #6b7280;">${item.seriesName}: </span>
              <span style="font-weight: 600; margin-left: 4px;">${value}</span>
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
          formatter: (value: number) => smartFormatTime(value, 'HH:mm:ss')
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
            formatter: (value: number) => formatBytes(value),
            color: '#6b7280',
            fontSize: 11
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
          axisLabel: { formatter: '{value}%', color: '#6b7280', fontSize: 11 },
          min: 0,
          max: 100,
          splitLine: { show: false }
        }
      ],
      series: [
        {
          name: '已用内存',
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
          name: '可用内存',
          type: 'line',
          smooth: true,
          data: [],
          symbol: 'none',
          lineStyle: { width: 2, color: '#10b981' }
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
    if (!memoryChart || !props.memoryData?.trend || props.memoryData.trend.length === 0) {
      return
    }

    const usedData = props.memoryData.trend.map((item) => [
      convertTimestamp(item.timestamp),
      item.usedBytes
    ])
    const availableData = props.memoryData.trend.map((item) => [
      convertTimestamp(item.timestamp),
      item.availableBytes
    ])
    const percentData = props.memoryData.trend.map((item) => [
      convertTimestamp(item.timestamp),
      item.usagePercent
    ])

    memoryChart.setOption({
      series: [{ data: usedData }, { data: availableData }, { data: percentData }]
    })
  }

  // 响应窗口大小变化
  const handleResize = () => {
    memoryChart?.resize()
  }

  // 监听数据变化
  watch(
    () => props.memoryData,
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
    { deep: true }
  )

  onMounted(() => {
    // 组件挂载后，若已有数据则初始化图表
    if (hasData.value) {
      nextTick(() => {
        initChart()
        updateChart()
      })
    }
    window.addEventListener('resize', handleResize)
  })

  onUnmounted(() => {
    // 组件卸载时清理资源
    window.removeEventListener('resize', handleResize)
    memoryChart?.dispose()
    memoryChart = null
    chartInitialized = false
  })
</script>

<style lang="scss" scoped>
  // 主题颜色变量
  $primary: #667eea;
  $success: #10b981;
  $warning: #f59e0b;
  $danger: #ef4444;
  $memory: #f5576c;

  // 灰度色阶
  $gray-50: #f9fafb;
  $gray-100: #f3f4f6;
  $gray-200: #e5e7eb;
  $gray-300: #d1d5db;
  $gray-400: #9ca3af;
  $gray-500: #6b7280;
  $gray-600: #4b5563;
  $gray-700: #374151;
  $gray-800: #1f2937;

  // 圆角变量
  $radius: 14px;
  $radius-sm: 10px;
  $radius-xs: 6px;

  .memory-monitor {
    font-family: var(--ax-font-sans);

    .monitor-card {
      background: #fff;
      border-radius: $radius;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
      overflow: hidden;
    }

    // 卡片头部样式
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

          &.memory {
            background: linear-gradient(135deg, $memory, #f093fb);
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

    // 指标卡片网格布局
    .metrics-grid {
      display: grid;
      grid-template-columns: repeat(4, 1fr);
      gap: 14px;
      padding: 20px 24px;

      @media (max-width: 1200px) {
        grid-template-columns: repeat(2, 1fr);
      }
      @media (max-width: 640px) {
        grid-template-columns: 1fr;
      }
    }

    // 单个指标卡片样式
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
        background: rgba($memory, 0.04);
        border-color: rgba($memory, 0.15);
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
          font-size: 22px;
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
        &.critical {
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
          background: $memory;
          transition: width 0.4s ease;
        }
      }
    }

    // 统计条样式
    .stats-bar {
      display: flex;
      gap: 12px;
      padding: 0 24px 20px;

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

        &.success {
          border-color: rgba($success, 0.3);
        }
        &.info {
          border-color: rgba($primary, 0.3);
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
          font-size: 16px;
          font-weight: 700;
          color: $gray-800;
        }
      }
    }

    // 图表区域样式
    .chart-area {
      padding: 18px 24px;
      background: $gray-50;
      margin: 0 24px 20px;
      border-radius: $radius-sm;

      .chart-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 14px;

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

              &.used {
                background: $memory;
              }
              &.available {
                background: $success;
              }
              &.percent {
                background: $primary;
              }
            }
          }
        }
      }

      .chart-container {
        height: 280px;
        width: 100%;
      }
    }

    // 详情面板样式
    .detail-panel {
      border-top: 1px solid $gray-200;
      margin: 0 24px 20px;
      padding-top: 14px;

      .panel-toggle {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px 16px;
        background: $gray-50;
        border-radius: $radius-xs;
        cursor: pointer;
        transition: all 0.2s ease;
        user-select: none;

        &:hover {
          background: $gray-100;
        }

        span {
          font-size: 12px;
          font-weight: 600;
          color: $gray-600;
        }

        svg {
          color: $gray-400;
          transition: transform 0.3s ease;

          &.rotated {
            transform: rotate(180deg);
          }
        }
      }

      .panel-content {
        padding: 16px 0;

        .detail-section {
          margin-bottom: 18px;

          &:last-child {
            margin-bottom: 0;
          }

          .section-title {
            font-size: 12px;
            font-weight: 600;
            color: $gray-600;
            margin-bottom: 12px;
            padding-bottom: 8px;
            border-bottom: 1px solid $gray-200;
          }

          .detail-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 10px;

            .detail-item {
              display: flex;
              justify-content: space-between;
              align-items: center;
              padding: 10px 14px;
              background: $gray-50;
              border-radius: $radius-xs;

              .label {
                font-size: 11px;
                color: $gray-500;
              }
              .value {
                font-size: 13px;
                font-weight: 600;
                color: $gray-700;

                &.warning {
                  color: $warning;
                }
                &.danger {
                  color: $danger;
                }
              }
            }
          }
        }
      }
    }

    // 空状态样式
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

  // 淡入淡出过渡动画
  .fade-enter-active,
  .fade-leave-active {
    transition: opacity 0.3s ease;
  }
  .fade-enter-from,
  .fade-leave-to {
    opacity: 0;
  }

  // 滑动展开/收起过渡动画
  .slide-enter-active,
  .slide-leave-active {
    transition: all 0.3s ease;
    max-height: 800px;
    overflow: hidden;
  }
  .slide-enter-from,
  .slide-leave-to {
    max-height: 0;
    opacity: 0;
  }
</style>
