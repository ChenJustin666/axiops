<template>
  <div class="system-monitor" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="monitor-card">
        <!-- 卡片头部区域 -->
        <div class="card-header">
          <div class="header-info">
            <div class="icon-badge system">
              <svg viewBox="0 0 24 24" width="22" height="22">
                <path
                  fill="currentColor"
                  d="M19.14 12.94c.04-.31.06-.63.06-.94 0-.31-.02-.63-.06-.94l2.03-1.58c.18-.14.23-.41.12-.61l-1.92-3.32c-.12-.22-.37-.29-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54c-.04-.24-.24-.41-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96c-.22-.08-.47 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.04.31-.06.63-.06.94s.02.63.06.94l-2.03 1.58c-.18.14-.23.41-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z"
                />
              </svg>
            </div>
            <div>
              <h3>系统监控</h3>
              <p>进程与文件描述符</p>
            </div>
          </div>
          <div class="status-tag" :class="systemStatusClass">{{ systemStatusText }}</div>
        </div>

        <div class="content-wrapper">
          <!-- 双栏布局：进程信息 + 文件描述符 -->
          <div class="two-column-grid">
            <!-- 进程信息区域 -->
            <div class="process-section">
              <div class="section-header">
                <svg viewBox="0 0 24 24" width="18" height="18">
                  <path
                    fill="currentColor"
                    d="M4 8h4V4H4v4zm6 12h4v-4h-4v4zm-6 0h4v-4H4v4zm0-6h4v-4H4v4zm6 0h4v-4h-4v4zm6-10v4h4V4h-4zm-6 4h4V4h-4v4zm6 6h4v-4h-4v4zm0 6h4v-4h-4v4z"
                  />
                </svg>
                <span>进程状态</span>
              </div>
              <div class="process-stats">
                <div class="stat-card running">
                  <div class="stat-icon">
                    <svg viewBox="0 0 24 24" width="24" height="24">
                      <path fill="currentColor" d="M8 5v14l11-7z" />
                    </svg>
                  </div>
                  <div class="stat-content">
                    <div class="stat-label">Running</div>
                    <div class="stat-value">{{ systemData?.processes?.running || 0 }}</div>
                  </div>
                </div>
                <div class="stat-card blocked">
                  <div class="stat-icon">
                    <svg viewBox="0 0 24 24" width="24" height="24">
                      <path fill="currentColor" d="M6 19h4V5H6v14zm8-14v14h4V5h-4z" />
                    </svg>
                  </div>
                  <div class="stat-content">
                    <div class="stat-label">Blocked</div>
                    <div class="stat-value">{{ systemData?.processes?.blocked || 0 }}</div>
                  </div>
                </div>
              </div>
            </div>

            <!-- 文件描述符区域 -->
            <div class="fd-section">
              <div class="section-header">
                <svg viewBox="0 0 24 24" width="18" height="18">
                  <path
                    fill="currentColor"
                    d="M14 2H6c-1.1 0-2 .9-2 2v16c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V8l-6-6zm4 18H6V4h7v5h5v11z"
                  />
                </svg>
                <span>文件描述符</span>
              </div>
              <div class="fd-stats">
                <div class="fd-overview">
                  <div class="fd-circle" :class="getFdLevelClass()">
                    <div class="fd-percent">
                      {{ formatPercentage(systemData?.fileDescriptors?.usagePercent) }}
                    </div>
                    <div class="fd-label">使用率</div>
                  </div>
                  <div class="fd-details">
                    <div class="fd-detail-item">
                      <span class="label">已分配</span>
                      <span class="value">{{
                        formatNumber(systemData?.fileDescriptors?.allocated)
                      }}</span>
                    </div>
                    <div class="fd-detail-item">
                      <span class="label">最大值</span>
                      <span class="value">{{
                        formatNumber(systemData?.fileDescriptors?.maximum)
                      }}</span>
                    </div>
                    <div class="fd-detail-item">
                      <span class="label">可用</span>
                      <span class="value">{{ formatNumber(getAvailableFd()) }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 文件描述符趋势图表 -->
          <div v-if="systemData?.fileDescriptors?.trend?.length" class="chart-area">
            <div class="chart-header">
              <span class="chart-title">文件描述符使用趋势</span>
              <div class="chart-legend">
                <span class="legend-item"><i class="dot allocated"></i>已分配</span>
                <span class="legend-item"><i class="dot percent"></i>使用率</span>
              </div>
            </div>
            <div ref="fdChartRef" class="chart-container"></div>
          </div>
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
        <h3>暂无系统监控数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, onUnmounted, nextTick, computed } from 'vue'
  import * as echarts from 'echarts'
  import { formatPercentage, formatNumber, smartFormatTime } from '@/utils/format'
  import type { NodeSystemMetrics } from '@/api/console/node-monitor'

  interface Props {
    systemData?: NodeSystemMetrics
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    systemData: undefined,
    loading: false
  })

  // 图表 DOM 引用
  const fdChartRef = ref<HTMLElement>()
  // ECharts 实例
  let fdChart: echarts.ECharts | null = null
  // 图表初始化标记
  let chartInitialized = false

  // 计算属性：判断是否有有效数据
  const hasData = computed(() => {
    return !!(props.systemData && Object.keys(props.systemData).length > 0)
  })

  // 计算属性：系统状态样式类
  const systemStatusClass = computed(() => {
    const fdUsage = props.systemData?.fileDescriptors?.usagePercent || 0
    const blocked = props.systemData?.processes?.blocked || 0
    if (fdUsage > 90 || blocked > 10) return 'danger'
    if (fdUsage > 70 || blocked > 5) return 'warning'
    return 'success'
  })

  // 计算属性：系统状态文本
  const systemStatusText = computed(() => {
    const fdUsage = props.systemData?.fileDescriptors?.usagePercent || 0
    const blocked = props.systemData?.processes?.blocked || 0
    if (fdUsage > 90 || blocked > 10) return '需要关注'
    if (fdUsage > 70 || blocked > 5) return '轻微压力'
    return '正常'
  })

  // 计算可用的文件描述符数量
  const getAvailableFd = (): number => {
    const allocated = props.systemData?.fileDescriptors?.allocated || 0
    const maximum = props.systemData?.fileDescriptors?.maximum || 0
    return maximum - allocated
  }

  // 根据文件描述符使用率返回对应的等级样式类
  const getFdLevelClass = (): string => {
    const percent = props.systemData?.fileDescriptors?.usagePercent || 0
    if (percent < 50) return 'level-normal'
    if (percent < 80) return 'level-warning'
    return 'level-danger'
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

  // 格式化数量，用于 Y 轴标签
  const formatCount = (value: number | undefined | null): string => {
    if (value === undefined || value === null || isNaN(value)) {
      return '0'
    }
    if (value >= 1000000) {
      return (value / 1000000).toFixed(1) + 'M'
    }
    if (value >= 1000) {
      return (value / 1000).toFixed(1) + 'K'
    }
    return value.toFixed(0)
  }

  // 初始化 ECharts 图表实例
  const initChart = () => {
    if (!fdChartRef.value || chartInitialized) return

    fdChart = echarts.init(fdChartRef.value)
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
            const rawValue = item.value[1]
            const value =
              item.seriesName === '使用率'
                ? `${(rawValue || 0).toFixed(2)}%`
                : formatNumber(rawValue)
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
        left: '70px',
        right: '70px',
        bottom: '8%',
        top: '12%',
        containLabel: false
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
          name: '数量',
          nameLocation: 'middle',
          nameGap: 50,
          nameTextStyle: { color: '#6b7280', fontSize: 12, fontWeight: 500 },
          position: 'left',
          axisLine: { show: true, lineStyle: { color: '#409eff' } },
          axisTick: { show: false },
          axisLabel: {
            formatter: (value: number) => formatCount(value),
            color: '#6b7280',
            fontSize: 11,
            margin: 8
          },
          splitLine: { lineStyle: { color: '#f3f4f6', type: 'dashed' } }
        },
        {
          type: 'value',
          name: '使用率',
          nameLocation: 'middle',
          nameGap: 45,
          nameTextStyle: { color: '#6b7280', fontSize: 12, fontWeight: 500 },
          position: 'right',
          axisLine: { show: true, lineStyle: { color: '#f56c6c' } },
          axisTick: { show: false },
          axisLabel: {
            formatter: '{value}%',
            color: '#6b7280',
            fontSize: 11,
            margin: 8
          },
          min: 0,
          max: 100,
          splitLine: { show: false }
        }
      ],
      series: [
        {
          name: '已分配',
          type: 'line',
          smooth: true,
          data: [],
          symbol: 'none',
          lineStyle: { width: 2, color: '#409eff' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(64, 158, 255, 0.2)' },
              { offset: 1, color: 'rgba(64, 158, 255, 0.02)' }
            ])
          }
        },
        {
          name: '使用率',
          type: 'line',
          smooth: true,
          yAxisIndex: 1,
          data: [],
          symbol: 'none',
          lineStyle: { width: 2, color: '#f56c6c' }
        }
      ]
    }

    fdChart.setOption(option)
  }

  // 更新图表数据
  const updateChart = () => {
    if (!fdChart || !props.systemData?.fileDescriptors?.trend) return

    const allocatedData = props.systemData.fileDescriptors.trend.map((item) => [
      convertTimestamp(item.timestamp),
      item.allocated || 0
    ])
    const percentData = props.systemData.fileDescriptors.trend.map((item) => [
      convertTimestamp(item.timestamp),
      item.usagePercent || 0
    ])

    fdChart.setOption({
      series: [{ data: allocatedData }, { data: percentData }]
    })
  }

  // 响应窗口大小变化
  const handleResize = () => {
    fdChart?.resize()
  }

  // 监听数据变化
  watch(
    () => props.systemData,
    () => {
      nextTick(() => {
        if (hasData.value && props.systemData?.fileDescriptors?.trend?.length) {
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
    // 组件挂载后，若已有趋势数据则初始化图表
    if (hasData.value && props.systemData?.fileDescriptors?.trend?.length) {
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
    fdChart?.dispose()
    fdChart = null
    chartInitialized = false
  })
</script>

<style lang="scss" scoped>
  // 主题颜色变量
  $primary: #409eff;
  $success: #10b981;
  $warning: #f59e0b;
  $danger: #ef4444;
  $system: #f59e0b;

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

  .system-monitor {
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

          &.system {
            background: linear-gradient(135deg, $system, #d97706);
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

    // 内容区域
    .content-wrapper {
      padding: 20px 24px;
    }

    // 统一的区块标题样式
    .section-header {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 16px;
      padding-bottom: 12px;
      border-bottom: 1px solid $gray-200;
      color: $gray-600;

      span {
        font-size: 14px;
        font-weight: 600;
      }
    }

    // 双栏布局
    .two-column-grid {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 20px;
      margin-bottom: 20px;

      @media (max-width: 1024px) {
        grid-template-columns: 1fr;
      }
    }

    // 进程信息区域
    .process-section,
    .fd-section {
      background: $gray-50;
      border-radius: $radius-sm;
      padding: 20px;
    }

    .process-stats {
      display: grid;
      grid-template-columns: repeat(2, 1fr);
      gap: 14px;

      .stat-card {
        padding: 18px;
        background: white;
        border-radius: $radius-xs;
        display: flex;
        align-items: center;
        gap: 14px;
        border-left: 4px solid;
        transition: all 0.2s ease;

        &:hover {
          transform: translateY(-2px);
          box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
        }

        &.running {
          border-left-color: $success;

          .stat-icon {
            background: linear-gradient(135deg, #d1fae5, #a7f3d0);
            color: #065f46;
          }
        }

        &.blocked {
          border-left-color: $warning;

          .stat-icon {
            background: linear-gradient(135deg, #fef3c7, #fde68a);
            color: #92400e;
          }
        }

        .stat-icon {
          width: 44px;
          height: 44px;
          border-radius: $radius-xs;
          display: flex;
          align-items: center;
          justify-content: center;
          flex-shrink: 0;
        }

        .stat-content {
          flex: 1;

          .stat-label {
            font-size: 12px;
            color: $gray-500;
            margin-bottom: 4px;
          }

          .stat-value {
            font-size: 24px;
            font-weight: 700;
            color: $gray-800;
          }
        }
      }
    }

    // 文件描述符区域
    .fd-stats {
      .fd-overview {
        display: flex;
        align-items: center;
        gap: 20px;

        @media (max-width: 640px) {
          flex-direction: column;
        }

        .fd-circle {
          width: 130px;
          height: 130px;
          border-radius: 50%;
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          border: 6px solid;
          transition: all 0.3s ease;
          flex-shrink: 0;

          &.level-normal {
            border-color: $success;
            background: linear-gradient(135deg, #d1fae5, #a7f3d0);
          }

          &.level-warning {
            border-color: $warning;
            background: linear-gradient(135deg, #fef3c7, #fde68a);
          }

          &.level-danger {
            border-color: $danger;
            background: linear-gradient(135deg, #fee2e2, #fecaca);
          }

          .fd-percent {
            font-size: 24px;
            font-weight: 700;
            color: $gray-800;
          }

          .fd-label {
            font-size: 12px;
            color: $gray-600;
            margin-top: 4px;
          }
        }

        .fd-details {
          flex: 1;
          display: flex;
          flex-direction: column;
          gap: 10px;
          width: 100%;

          .fd-detail-item {
            padding: 12px 16px;
            background: white;
            border-radius: $radius-xs;
            display: flex;
            justify-content: space-between;
            align-items: center;

            .label {
              font-size: 12px;
              color: $gray-500;
            }

            .value {
              font-size: 15px;
              font-weight: 700;
              color: $gray-800;
            }
          }
        }
      }
    }

    // 图表区域样式
    .chart-area {
      padding: 18px 24px;
      background: $gray-50;
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

              &.allocated {
                background: $primary;
              }
              &.percent {
                background: #f56c6c;
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
</style>
