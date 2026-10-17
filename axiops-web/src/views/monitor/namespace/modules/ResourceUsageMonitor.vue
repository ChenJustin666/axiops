<template>
  <div class="resource-usage-monitor" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="monitor-card">
        <!-- 卡片头部 -->
        <div class="card-header" :class="resourceType">
          <div class="header-info">
            <div class="icon-badge" :class="resourceType">
              <svg v-if="resourceType === 'cpu'" viewBox="0 0 24 24" width="22" height="22">
                <path
                  fill="currentColor"
                  d="M17,17H7V7H17M21,11V9H19V7C19,5.89 18.1,5 17,5H15V3H13V5H11V3H9V5H7C5.89,5 5,5.89 5,7V9H3V11H5V13H3V15H5V17C5,18.1 5.9,19 7,19H9V21H11V19H13V21H15V19H17C18.1,19 19,18.1 19,17V15H21V13H19V11M15,15H9V9H15V15Z"
                />
              </svg>
              <svg v-else viewBox="0 0 24 24" width="22" height="22">
                <path
                  fill="currentColor"
                  d="M17,9H7V7H17M17,13H7V11H17M14,17H7V15H14M12,3A1,1 0 0,1 13,4A1,1 0 0,1 12,5A1,1 0 0,1 11,4A1,1 0 0,1 12,3M19,3H14.82C14.4,1.84 13.3,1 12,1C10.7,1 9.6,1.84 9.18,3H5A2,2 0 0,0 3,5V19A2,2 0 0,0 5,21H19A2,2 0 0,0 21,19V5A2,2 0 0,0 19,3Z"
                />
              </svg>
            </div>
            <div>
              <h3>{{ resourceType === 'cpu' ? 'CPU 使用情况' : '内存使用情况' }}</h3>
              <p>资源使用趋势与统计</p>
            </div>
          </div>
          <div class="status-tag" :class="statusClass">{{ statusText }}</div>
        </div>

        <!-- 指标卡片 -->
        <div class="metrics-grid">
          <div class="metric-box primary">
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10 10-4.5 10-10S17.5 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67V7z"
                />
              </svg>
              <span>当前使用</span>
            </div>
            <div class="box-value">
              <span class="value">{{ formatCurrentValue() }}</span>
            </div>
            <div class="box-badge" :class="usageLevelClass">{{ usageLevelText }}</div>
          </div>

          <div class="metric-box" :class="{ warning: getCurrentPercent() > 70 }">
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M13 2.05v2.02c3.95.49 7 3.85 7 7.93s-3.05 7.44-7 7.93v2.02c5.05-.5 9-4.76 9-9.95s-3.95-9.45-9-9.95zm-2 0c-5.05.5-9 4.76-9 9.95s3.95 9.45 9 9.95v-2.02c-3.95-.49-7-3.85-7-7.93s3.05-7.44 7-7.93V2.05z"
                />
              </svg>
              <span>使用率</span>
            </div>
            <div class="box-value">
              <span class="value">{{ formatPercentage(getCurrentPercent()) }}</span>
            </div>
            <div class="box-progress">
              <div
                class="progress-fill"
                :class="progressClass"
                :style="{ width: getCurrentPercent() + '%' }"
              ></div>
            </div>
          </div>

          <div class="metric-box">
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14h-2v-4h2v4zm0-6h-2V7h2v4z"
                />
              </svg>
              <span>平均值</span>
            </div>
            <div class="box-value">
              <span class="value">{{ formatAvgValue() }}</span>
            </div>
          </div>

          <div class="metric-box">
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M3.5,18.49L9.5,12.48L13.5,16.48L22,6.92L20.59,5.51L13.5,13.48L9.5,9.48L2,16.99L3.5,18.49Z"
                />
              </svg>
              <span>峰值</span>
            </div>
            <div class="box-value">
              <span class="value">{{ formatMaxValue() }}</span>
            </div>
          </div>
        </div>

        <!-- 图表区域 -->
        <div v-if="resourceData?.trend?.length" class="chart-area">
          <div class="chart-header">
            <span class="chart-title">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M3.5,18.49L9.5,12.48L13.5,16.48L22,6.92L20.59,5.51L13.5,13.48L9.5,9.48L2,16.99L3.5,18.49Z"
                />
              </svg>
              使用趋势
            </span>
            <div class="chart-legend">
              <span class="legend-item"> <i class="dot usage"></i>使用量 </span>
              <span class="legend-item"> <i class="dot percent"></i>使用率 </span>
            </div>
          </div>
          <div ref="chartRef" class="chart-container"></div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" width="64" height="64">
            <path
              fill="currentColor"
              d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14h-2V9h2v8zm0-10h-2V5h2v2z"
            />
          </svg>
        </div>
        <h3>暂无{{ resourceType === 'cpu' ? 'CPU' : '内存' }}数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, onUnmounted, nextTick, computed } from 'vue'
  import * as echarts from 'echarts'
  import type { NamespaceCPUMetrics, NamespaceMemoryMetrics } from '@/api/console/namespace-monitor'
  import { formatBytes, formatPercentage, smartFormatTime } from '@/utils/format'

  // ==================== 类型定义 ====================

  interface Props {
    resourceType: 'cpu' | 'memory'
    resourceData?: NamespaceCPUMetrics | NamespaceMemoryMetrics
    loading?: boolean
    active?: boolean
  }

  // ==================== 组件属性 ====================

  const props = withDefaults(defineProps<Props>(), {
    resourceData: undefined,
    loading: false,
    active: true
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
    return !!(props.resourceData && Object.keys(props.resourceData).length > 0)
  })

  /**
   * 获取状态样式类
   */
  const statusClass = computed(() => {
    const percent = getCurrentPercent()
    if (percent >= 80) return 'danger'
    if (percent >= 50) return 'warning'
    return 'success'
  })

  /**
   * 获取状态文本
   */
  const statusText = computed(() => {
    const percent = getCurrentPercent()
    if (percent >= 80) return '负载过高'
    if (percent >= 50) return '负载偏高'
    return '运行正常'
  })

  /**
   * 获取使用级别样式类
   */
  const usageLevelClass = computed(() => {
    const percent = getCurrentPercent()
    if (percent >= 80) return 'critical'
    if (percent >= 50) return 'warning'
    return 'normal'
  })

  /**
   * 获取使用级别文本
   */
  const usageLevelText = computed(() => {
    const percent = getCurrentPercent()
    if (percent >= 80) return '过高'
    if (percent >= 50) return '偏高'
    return '正常'
  })

  /**
   * 获取进度条样式类
   */
  const progressClass = computed(() => {
    const percent = getCurrentPercent()
    if (percent >= 80) return 'danger'
    if (percent >= 50) return 'warning'
    return 'success'
  })

  // ==================== 格式化方法 ====================

  /**
   * 格式化当前使用值
   */
  const formatCurrentValue = (): string => {
    if (!props.resourceData?.current) return '0'

    if (props.resourceType === 'cpu') {
      const cpuData = props.resourceData as NamespaceCPUMetrics
      const value = cpuData.current.usageCores || 0
      return formatCoresValue(value)
    } else {
      const memData = props.resourceData as NamespaceMemoryMetrics
      const value = memData.current.workingSetBytes || 0
      return formatBytes(value)
    }
  }

  /**
   * 获取当前使用百分比
   */
  const getCurrentPercent = (): number => {
    return props.resourceData?.current?.usagePercent || 0
  }

  /**
   * 格式化平均值
   */
  const formatAvgValue = (): string => {
    if (!props.resourceData?.current) return '0'

    if (props.resourceType === 'cpu') {
      const cpuData = props.resourceData as NamespaceCPUMetrics
      const value = cpuData.current.avgCores || 0
      return formatCoresValue(value)
    } else {
      const memData = props.resourceData as NamespaceMemoryMetrics
      const value = memData.current.avgBytes || 0
      return formatBytes(value)
    }
  }

  /**
   * 格式化峰值
   */
  const formatMaxValue = (): string => {
    if (!props.resourceData?.current) return '0'

    if (props.resourceType === 'cpu') {
      const cpuData = props.resourceData as NamespaceCPUMetrics
      const value = cpuData.current.maxCores || 0
      return formatCoresValue(value)
    } else {
      const memData = props.resourceData as NamespaceMemoryMetrics
      const value = memData.current.maxBytes || 0
      return formatBytes(value)
    }
  }

  /**
   * 格式化 CPU 核心数
   */
  const formatCoresValue = (value: number): string => {
    if (value < 0.001) return `${(value * 1000).toFixed(0)}m`
    return `${value.toFixed(3)} cores`
  }

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

    const trendData = props.resourceData?.trend || []

    // 定义颜色配置
    const colorConfig = {
      cpu: { primary: '#667eea', gradient: 'rgba(102, 126, 234, 0.3)' },
      memory: { primary: '#f093fb', gradient: 'rgba(240, 147, 251, 0.3)' }
    }

    const colors = colorConfig[props.resourceType]

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

          params.forEach((param: any, index: number) => {
            const value = param.data[1]
            let formattedValue = ''

            if (index === 0) {
              // 使用量
              if (props.resourceType === 'cpu') {
                formattedValue = formatCoresValue(value)
              } else {
                formattedValue = formatBytes(value)
              }
            } else {
              // 使用率
              formattedValue = `${value.toFixed(2)}%`
            }

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
      yAxis: [
        {
          type: 'value',
          name: props.resourceType === 'cpu' ? 'Cores' : 'Memory',
          nameTextStyle: { color: '#6b7280', fontSize: 11 },
          axisLine: { show: false },
          axisTick: { show: false },
          axisLabel: {
            color: '#6b7280',
            fontSize: 11,
            formatter: (value: number) => {
              if (props.resourceType === 'cpu') {
                if (value < 0.001) return `${(value * 1000).toFixed(0)}m`
                return `${value.toFixed(2)}`
              } else {
                return formatBytes(value)
              }
            }
          },
          splitLine: { lineStyle: { color: '#f3f4f6', type: 'dashed' } }
        },
        {
          type: 'value',
          name: '使用率 (%)',
          position: 'right',
          min: 0,
          max: 100,
          nameTextStyle: { color: '#6b7280', fontSize: 11 },
          axisLine: { show: false },
          axisTick: { show: false },
          axisLabel: {
            color: '#6b7280',
            fontSize: 11,
            formatter: '{value}%'
          },
          splitLine: { show: false }
        }
      ],
      series: [
        {
          name: '使用量',
          type: 'line',
          yAxisIndex: 0,
          data: trendData.map((t: any) => [
            t.timestamp,
            props.resourceType === 'cpu' ? t.usageCores : t.workingSetBytes
          ]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: colors.primary },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: colors.gradient },
              { offset: 1, color: colors.gradient.replace('0.3)', '0.02)') }
            ])
          }
        },
        {
          name: '使用率',
          type: 'line',
          yAxisIndex: 1,
          data: trendData.map((t: any) => [t.timestamp, t.usagePercent]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#e6a23c' }
        }
      ]
    }

    chart.setOption(option)
  }

  /**
   * 更新图表数据
   */
  const updateChart = () => {
    if (!chart || !props.resourceData?.trend) return

    const trendData = props.resourceData.trend

    chart.setOption({
      series: [
        {
          data: trendData.map((t: any) => [
            t.timestamp,
            props.resourceType === 'cpu' ? t.usageCores : t.workingSetBytes
          ])
        },
        {
          data: trendData.map((t: any) => [t.timestamp, t.usagePercent])
        }
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
    () => [props.resourceData, props.active],
    async () => {
      if (!hasData.value || !props.active) return

      await nextTick()

      // 延迟等待 DOM 渲染完成
      setTimeout(() => {
        if (props.resourceData?.trend?.length) {
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
  $cpu-color: #667eea;
  $memory-color: #f093fb;

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
  .resource-usage-monitor {
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
    border-bottom: 1px solid;
    flex-wrap: wrap;
    gap: 14px;

    &.cpu {
      background: linear-gradient(135deg, #ede9fe 0%, #ddd6fe 100%);
      border-bottom-color: #c4b5fd;
    }

    &.memory {
      background: linear-gradient(135deg, #fef5f8 0%, #fee8ef 100%);
      border-bottom-color: #fecdd7;
    }

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
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);

        &.cpu {
          background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        }

        &.memory {
          background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
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

  // ==================== 指标卡片网格 ====================
  .metrics-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 14px;
    padding: 20px 24px;
    border-bottom: 1px solid $gray-200;

    @media (max-width: 1200px) {
      grid-template-columns: repeat(2, 1fr);
    }

    @media (max-width: 640px) {
      grid-template-columns: 1fr;
    }
  }

  // ==================== 指标卡片样式 ====================
  .metric-box {
    position: relative;
    padding: 16px;
    border-radius: $radius-sm;
    background: $gray-50;
    border: 1px solid $gray-200;
    transition: all 0.2s ease;

    &:hover {
      border-color: $gray-300;
    }

    &.primary {
      background: rgba($primary, 0.04);
      border-color: rgba($primary, 0.15);
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
      .value {
        font-size: 22px;
        font-weight: 700;
        color: $gray-800;
      }
    }

    .box-badge {
      display: inline-block;
      padding: 4px 10px;
      border-radius: 12px;
      font-size: 11px;
      font-weight: 600;
      margin-top: 8px;

      &.normal {
        background: rgba($success, 0.1);
        color: #059669;
      }

      &.warning {
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
        border-radius: 2px;
        transition: width 0.4s ease;

        &.success {
          background: linear-gradient(90deg, $success, #34d399);
        }

        &.warning {
          background: linear-gradient(90deg, $warning, #fbbf24);
        }

        &.danger {
          background: linear-gradient(90deg, $danger, #f87171);
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

            &.usage {
              background: $cpu-color;
            }

            &.percent {
              background: #e6a23c;
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
    .resource-usage-monitor {
      padding: 12px;
    }

    .card-header {
      padding: 16px;
    }

    .metrics-grid {
      padding: 16px;
    }

    .chart-area {
      padding: 16px;

      .chart-container {
        height: 280px;
      }
    }
  }
</style>
