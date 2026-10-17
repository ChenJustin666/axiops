<template>
  <div class="cpu-monitor" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="monitor-card">
        <!-- 卡片头部区域 -->
        <div class="card-header">
          <div class="header-info">
            <div class="icon-badge cpu">
              <svg viewBox="0 0 24 24" width="22" height="22">
                <path
                  fill="currentColor"
                  d="M6 18h2v-2H6v2zm8-12V4h-4v2h4zm2 12h2v-2h-2v2zm4-12h2V4h-2v2zm-6 8h2v-2h-2v2zm0-8h2V4h-2v2zm0 4h2V8h-2v2zm6 4h2v-2h-2v2zm0-4h2V8h-2v2zm-6 8h2v-2h-2v2zm-4-4h2v-2H8v2zm8 0h2v-2h-2v2zm-8-4h2V8H8v2zm-2 4H4v2h2v-2zM4 8h2v2H4V8zm2-2H4v2h2V6zm0 12H4v2h2v-2zm-2-4h2v-2H4v2z"
                />
              </svg>
            </div>
            <div>
              <h3>CPU 使用情况</h3>
              <p>处理器负载与性能</p>
            </div>
          </div>
          <div class="status-tag" :class="cpuStatusClass">{{ cpuStatusText }}</div>
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
                formatPercentageValue(cpuData?.summary?.avgUsagePercent)
              }}</span>
              <span class="unit">%</span>
            </div>
            <div class="box-badge" :class="getUsageLevel(cpuData?.summary?.avgUsagePercent)">
              {{ getUsageLevelText(cpuData?.summary?.avgUsagePercent) }}
            </div>
            <div class="box-progress">
              <div
                class="progress-fill"
                :style="{ width: (cpuData?.summary?.avgUsagePercent || 0) + '%' }"
              ></div>
            </div>
          </div>

          <div
            class="metric-box"
            :class="{ warning: (cpuData?.summary?.maxUsagePercent || 0) > 80 }"
          >
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path fill="currentColor" d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z" />
              </svg>
              <span>峰值使用率</span>
            </div>
            <div class="box-value">
              <span class="value">{{
                formatPercentageValue(cpuData?.summary?.maxUsagePercent)
              }}</span>
              <span class="unit">%</span>
            </div>
          </div>

          <div class="metric-box">
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10 10-4.5 10-10S17.5 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67V7z"
                />
              </svg>
              <span>平均负载 (5m)</span>
            </div>
            <div class="box-value">
              <span class="value">{{ (cpuData?.summary?.avgLoad5 || 0).toFixed(2) }}</span>
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
              <span>总核心数</span>
            </div>
            <div class="box-value">
              <span class="value">{{ cpuData?.current?.totalCores || 0 }}</span>
              <span class="unit">cores</span>
            </div>
          </div>
        </div>

        <!-- 负载指标统计条 -->
        <div class="stats-bar">
          <div class="stats-item">
            <span class="stats-label">Load1</span>
            <span class="stats-value">{{ (cpuData?.current?.load1 || 0).toFixed(2) }}</span>
          </div>
          <div class="stats-item">
            <span class="stats-label">Load5</span>
            <span class="stats-value">{{ (cpuData?.current?.load5 || 0).toFixed(2) }}</span>
          </div>
          <div class="stats-item">
            <span class="stats-label">Load15</span>
            <span class="stats-value">{{ (cpuData?.current?.load15 || 0).toFixed(2) }}</span>
          </div>
          <div class="stats-item info">
            <span class="stats-label">上下文切换</span>
            <span class="stats-value"
              >{{ formatCompactNumber(cpuData?.current?.contextSwitchRate) }}/s</span
            >
          </div>
        </div>

        <!-- 趋势图表区域 -->
        <div class="chart-area">
          <div class="chart-header">
            <span class="chart-title">使用趋势</span>
            <div class="chart-legend">
              <span class="legend-item"><i class="dot usage"></i>使用率 (%)</span>
              <span class="legend-item"><i class="dot load"></i>Load5</span>
            </div>
          </div>
          <div ref="cpuChartRef" class="chart-container"></div>
        </div>

        <!-- 可折叠的详细信息面板 -->
        <div class="detail-panel">
          <div class="panel-toggle" @click="toggleDetail">
            <span>CPU 详细指标</span>
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
                <div class="section-title">CPU 时间分布</div>
                <div class="detail-grid">
                  <div class="detail-item">
                    <span class="label">User</span>
                    <span class="value">{{ formatPercentage(cpuData?.current?.userPercent) }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="label">System</span>
                    <span class="value">{{
                      formatPercentage(cpuData?.current?.systemPercent)
                    }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="label">IOWait</span>
                    <span
                      class="value"
                      :class="{ warning: (cpuData?.current?.iowaitPercent || 0) > 10 }"
                    >
                      {{ formatPercentage(cpuData?.current?.iowaitPercent) }}
                    </span>
                  </div>
                  <div class="detail-item">
                    <span class="label">Steal</span>
                    <span
                      class="value"
                      :class="{ danger: (cpuData?.current?.stealPercent || 0) > 5 }"
                    >
                      {{ formatPercentage(cpuData?.current?.stealPercent) }}
                    </span>
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
        <h3>暂无 CPU 监控数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, onUnmounted, nextTick, computed } from 'vue'
  import * as echarts from 'echarts'
  import { formatPercentage, formatCompactNumber, smartFormatTime } from '@/utils/format'
  import type { NodeCPUMetrics } from '@/api/console/node-monitor'

  interface Props {
    cpuData?: NodeCPUMetrics
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    cpuData: undefined,
    loading: false
  })

  // 图表 DOM 引用
  const cpuChartRef = ref<HTMLElement>()
  // ECharts 实例
  let cpuChart: echarts.ECharts | null = null
  // 详情面板展开状态
  const showDetail = ref(false)
  // 图表初始化标记，避免重复初始化
  let chartInitialized = false

  // 计算属性：判断是否有有效数据
  const hasData = computed(() => {
    return !!(props.cpuData && Object.keys(props.cpuData).length > 0)
  })

  // 计算属性：根据 CPU 使用率返回状态样式类
  const cpuStatusClass = computed(() => {
    const usage = props.cpuData?.summary?.avgUsagePercent || 0
    if (usage > 90) return 'danger'
    if (usage > 70) return 'warning'
    return 'success'
  })

  // 计算属性：根据 CPU 使用率返回状态文本
  const cpuStatusText = computed(() => {
    const usage = props.cpuData?.summary?.avgUsagePercent || 0
    if (usage > 90) return '负载过高'
    if (usage > 70) return '负载较高'
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

  // 根据使用率返回对应的等级样式类
  const getUsageLevel = (value?: number): string => {
    if (!value) return 'low'
    if (value < 50) return 'low'
    if (value < 70) return 'normal'
    if (value < 90) return 'high'
    return 'critical'
  }

  // 根据使用率返回对应的等级文本
  const getUsageLevelText = (value?: number): string => {
    if (!value) return '空闲'
    if (value < 50) return '空闲'
    if (value < 70) return '正常'
    if (value < 90) return '较高'
    return '过高'
  }

  // 将各种格式的时间戳统一转换为毫秒级时间戳
  const convertTimestamp = (timestamp: any): number => {
    if (typeof timestamp === 'string') {
      return new Date(timestamp).getTime()
    } else if (typeof timestamp === 'number') {
      // 判断是否为秒级时间戳（10位数字）
      if (timestamp < 10000000000) {
        return timestamp * 1000
      }
      return timestamp
    }
    return Date.now()
  }

  // 初始化 ECharts 图表实例
  const initChart = () => {
    if (!cpuChartRef.value || chartInitialized) return

    cpuChart = echarts.init(cpuChartRef.value)
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
              item.seriesName === 'Load5'
                ? item.value[1].toFixed(2)
                : `${item.value[1].toFixed(2)}%`
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
          name: '使用率 (%)',
          position: 'left',
          nameTextStyle: { color: '#6b7280', fontSize: 11 },
          axisLine: { show: false },
          axisTick: { show: false },
          axisLabel: { formatter: '{value}%', color: '#6b7280', fontSize: 11 },
          splitLine: { lineStyle: { color: '#f3f4f6', type: 'dashed' } },
          min: 0,
          max: 100
        },
        {
          type: 'value',
          name: 'Load',
          position: 'right',
          nameTextStyle: { color: '#6b7280', fontSize: 11 },
          axisLine: { show: false },
          axisTick: { show: false },
          axisLabel: { color: '#6b7280', fontSize: 11 },
          splitLine: { show: false }
        }
      ],
      series: [
        {
          name: '使用率 (%)',
          type: 'line',
          smooth: true,
          data: [],
          symbol: 'none',
          lineStyle: { width: 2, color: '#667eea' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(102, 126, 234, 0.2)' },
              { offset: 1, color: 'rgba(102, 126, 234, 0.02)' }
            ])
          }
        },
        {
          name: 'Load5',
          type: 'line',
          smooth: true,
          yAxisIndex: 1,
          data: [],
          symbol: 'none',
          lineStyle: { width: 2, color: '#f5576c' }
        }
      ]
    }

    cpuChart.setOption(option)
  }

  // 更新图表数据
  const updateChart = () => {
    if (!cpuChart || !props.cpuData?.trend || props.cpuData.trend.length === 0) {
      return
    }

    const usageData = props.cpuData.trend.map((item) => [
      convertTimestamp(item.timestamp),
      item.usagePercent
    ])
    const loadData = props.cpuData.trend.map((item) => [
      convertTimestamp(item.timestamp),
      item.load5
    ])

    cpuChart.setOption({
      series: [{ data: usageData }, { data: loadData }]
    })
  }

  // 响应窗口大小变化，调整图表尺寸
  const handleResize = () => {
    cpuChart?.resize()
  }

  // 监听数据变化，当数据更新时刷新图表
  watch(
    () => props.cpuData,
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
    cpuChart?.dispose()
    cpuChart = null
    chartInitialized = false
  })
</script>

<style lang="scss" scoped>
  // 主题颜色变量
  $primary: #667eea;
  $success: #10b981;
  $warning: #f59e0b;
  $danger: #ef4444;

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

  .cpu-monitor {
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

          &.cpu {
            background: linear-gradient(135deg, $primary, #764ba2);
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
          background: $primary;
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

        &.info {
          border-color: rgba($primary, 0.3);
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

              &.usage {
                background: $primary;
              }
              &.load {
                background: #f5576c;
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
    max-height: 500px;
    overflow: hidden;
  }
  .slide-enter-from,
  .slide-leave-to {
    max-height: 0;
    opacity: 0;
  }
</style>
