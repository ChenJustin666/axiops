<template>
  <div class="pod-metrics" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="monitor-card">
        <!-- 卡片头部 -->
        <div class="card-header">
          <div class="header-info">
            <div class="icon-badge pod">
              <svg viewBox="0 0 24 24" width="22" height="22">
                <path
                  fill="currentColor"
                  d="M12,2A3,3 0 0,1 15,5V11A3,3 0 0,1 12,14A3,3 0 0,1 9,11V5A3,3 0 0,1 12,2M7.5,22A3.5,3.5 0 0,1 4,18.5V17.5A3.5,3.5 0 0,1 7.5,14A3.5,3.5 0 0,1 11,17.5V18.5A3.5,3.5 0 0,1 7.5,22M16.5,22A3.5,3.5 0 0,1 13,18.5V17.5A3.5,3.5 0 0,1 16.5,14A3.5,3.5 0 0,1 20,17.5V18.5A3.5,3.5 0 0,1 16.5,22Z"
                />
              </svg>
            </div>
            <div>
              <h3>Pod 指标</h3>
              <p>Pod 状态与趋势分析</p>
            </div>
          </div>
          <div class="header-stats">
            <div class="stat-badge total">
              <span class="label">总计</span>
              <span class="value">{{ podData?.total || 0 }}</span>
            </div>
            <div class="stat-badge running">
              <span class="label">Running</span>
              <span class="value">{{ podData?.running || 0 }}</span>
            </div>
            <div class="stat-badge pending" v-if="(podData?.pending || 0) > 0">
              <span class="label">Pending</span>
              <span class="value">{{ podData?.pending || 0 }}</span>
            </div>
            <div class="stat-badge failed" v-if="(podData?.failed || 0) > 0">
              <span class="label">Failed</span>
              <span class="value">{{ podData?.failed || 0 }}</span>
            </div>
            <div class="stat-badge restarts" v-if="podData?.totalRestarts">
              <svg viewBox="0 0 24 24" width="14" height="14">
                <path
                  fill="currentColor"
                  d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"
                />
              </svg>
              <span class="value">{{ podData.totalRestarts }} 次重启</span>
            </div>
          </div>
        </div>

        <!-- 内容区域 -->
        <div class="content-wrapper">
          <!-- 状态概览 -->
          <div class="status-overview">
            <div class="overview-title">状态分布</div>
            <div class="status-grid">
              <div class="status-item running">
                <div class="status-icon">
                  <svg viewBox="0 0 24 24" width="24" height="24">
                    <path
                      fill="currentColor"
                      d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"
                    />
                  </svg>
                </div>
                <div class="status-info">
                  <span class="status-label">Running</span>
                  <span class="status-value">{{ podData?.running || 0 }}</span>
                </div>
                <div class="status-progress">
                  <div
                    class="progress-fill"
                    :style="{ width: getPercent(podData?.running) + '%' }"
                  ></div>
                </div>
              </div>

              <div class="status-item pending">
                <div class="status-icon">
                  <svg viewBox="0 0 24 24" width="24" height="24">
                    <path
                      fill="currentColor"
                      d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10 10-4.5 10-10S17.5 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67V7z"
                    />
                  </svg>
                </div>
                <div class="status-info">
                  <span class="status-label">Pending</span>
                  <span class="status-value">{{ podData?.pending || 0 }}</span>
                </div>
                <div class="status-progress">
                  <div
                    class="progress-fill"
                    :style="{ width: getPercent(podData?.pending) + '%' }"
                  ></div>
                </div>
              </div>

              <div class="status-item failed">
                <div class="status-icon">
                  <svg viewBox="0 0 24 24" width="24" height="24">
                    <path
                      fill="currentColor"
                      d="M12 2C6.47 2 2 6.47 2 12s4.47 10 10 10 10-4.47 10-10S17.53 2 12 2zm5 13.59L15.59 17 12 13.41 8.41 17 7 15.59 10.59 12 7 8.41 8.41 7 12 10.59 15.59 7 17 8.41 13.41 12 17 15.59z"
                    />
                  </svg>
                </div>
                <div class="status-info">
                  <span class="status-label">Failed</span>
                  <span class="status-value">{{ podData?.failed || 0 }}</span>
                </div>
                <div class="status-progress">
                  <div
                    class="progress-fill"
                    :style="{ width: getPercent(podData?.failed) + '%' }"
                  ></div>
                </div>
              </div>

              <div class="status-item succeeded">
                <div class="status-icon">
                  <svg viewBox="0 0 24 24" width="24" height="24">
                    <path
                      fill="currentColor"
                      d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"
                    />
                  </svg>
                </div>
                <div class="status-info">
                  <span class="status-label">Succeeded</span>
                  <span class="status-value">{{ podData?.succeeded || 0 }}</span>
                </div>
                <div class="status-progress">
                  <div
                    class="progress-fill"
                    :style="{ width: getPercent(podData?.succeeded) + '%' }"
                  ></div>
                </div>
              </div>

              <div class="status-item unknown">
                <div class="status-icon">
                  <svg viewBox="0 0 24 24" width="24" height="24">
                    <path
                      fill="currentColor"
                      d="M11 18h2v-2h-2v2zm1-16C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm0-14c-2.21 0-4 1.79-4 4h2c0-1.1.9-2 2-2s2 .9 2 2c0 2-3 1.75-3 5h2c0-2.25 3-2.5 3-5 0-2.21-1.79-4-4-4z"
                    />
                  </svg>
                </div>
                <div class="status-info">
                  <span class="status-label">Unknown</span>
                  <span class="status-value">{{ podData?.unknown || 0 }}</span>
                </div>
                <div class="status-progress">
                  <div
                    class="progress-fill"
                    :style="{ width: getPercent(podData?.unknown) + '%' }"
                  ></div>
                </div>
              </div>
            </div>
          </div>

          <!-- 图表区域 -->
          <div v-if="podData?.trend?.length" class="chart-area">
            <div class="chart-header">
              <span class="chart-title">Pod 数量趋势</span>
              <div class="chart-legend">
                <span class="legend-item"><i class="dot total"></i>Total</span>
                <span class="legend-item"><i class="dot running"></i>Running</span>
                <span class="legend-item"><i class="dot pending"></i>Pending</span>
                <span class="legend-item"><i class="dot failed"></i>Failed</span>
              </div>
            </div>
            <div ref="chartRef" class="chart-container"></div>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" width="64" height="64">
            <path
              fill="currentColor"
              d="M12,2A3,3 0 0,1 15,5V11A3,3 0 0,1 12,14A3,3 0 0,1 9,11V5A3,3 0 0,1 12,2M7.5,22A3.5,3.5 0 0,1 4,18.5V17.5A3.5,3.5 0 0,1 7.5,14A3.5,3.5 0 0,1 11,17.5V18.5A3.5,3.5 0 0,1 7.5,22M16.5,22A3.5,3.5 0 0,1 13,18.5V17.5A3.5,3.5 0 0,1 16.5,14A3.5,3.5 0 0,1 20,17.5V18.5A3.5,3.5 0 0,1 16.5,22Z"
            />
          </svg>
        </div>
        <h3>暂无 Pod 数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, onUnmounted, nextTick, computed } from 'vue'
  import * as echarts from 'echarts'
  import type { NamespacePodStatistics } from '@/api/console/namespace-monitor'

  interface Props {
    podData?: NamespacePodStatistics
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    podData: undefined,
    loading: false
  })

  // 图表相关引用
  const chartRef = ref<HTMLElement>()
  let chart: echarts.ECharts | null = null

  // 判断是否有数据
  const hasData = computed(() => {
    return !!(props.podData && Object.keys(props.podData).length > 0)
  })

  // 计算各状态占比
  const getPercent = (value?: number): number => {
    if (!value || !props.podData?.total || props.podData.total === 0) return 0
    return Math.min((value / props.podData.total) * 100, 100)
  }

  // 初始化图表配置
  const initChart = () => {
    if (!chartRef.value || chart) return

    chart = echarts.init(chartRef.value)

    const option: echarts.EChartsOption = {
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(255, 255, 255, 0.98)',
        borderColor: '#eaecef',
        borderWidth: 1,
        padding: [12, 16],
        textStyle: {
          color: '#1f2937',
          fontSize: 12
        },
        axisPointer: {
          type: 'cross'
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
        axisLine: {
          lineStyle: { color: '#e5e7eb' }
        },
        axisLabel: {
          color: '#6b7280',
          fontSize: 11,
          formatter: (v: number) =>
            new Date(v).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
        },
        splitLine: {
          show: true,
          lineStyle: { color: '#f3f4f6', type: 'dashed' }
        }
      },
      yAxis: {
        type: 'value',
        name: 'Pods',
        minInterval: 1,
        nameTextStyle: {
          color: '#6b7280',
          fontSize: 11
        },
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: {
          color: '#6b7280',
          fontSize: 11
        },
        splitLine: {
          lineStyle: { color: '#f3f4f6', type: 'dashed' }
        }
      },
      series: []
    }

    chart.setOption(option)
    updateChartData()
  }

  // 更新图表数据
  const updateChartData = () => {
    if (!chart || !props.podData?.trend) return

    const trend = props.podData.trend
    if (trend.length === 0) return

    chart.setOption({
      series: [
        {
          name: 'Total',
          type: 'line',
          data: trend.map((t) => [t.timestamp, t.total]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#3b82f6' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(59, 130, 246, 0.2)' },
              { offset: 1, color: 'rgba(59, 130, 246, 0.02)' }
            ])
          }
        },
        {
          name: 'Running',
          type: 'line',
          data: trend.map((t) => [t.timestamp, t.running]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#10b981' }
        },
        {
          name: 'Pending',
          type: 'line',
          data: trend.map((t) => [t.timestamp, t.pending]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#f59e0b' }
        },
        {
          name: 'Failed',
          type: 'line',
          data: trend.map((t) => [t.timestamp, t.failed]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#ef4444' }
        }
      ]
    })
  }

  // 处理窗口尺寸变化
  const handleResize = () => {
    chart?.resize()
  }

  // 监听数据变化
  watch(
    () => props.podData,
    () => {
      nextTick(() => {
        if (hasData.value && props.podData?.trend?.length) {
          if (!chart) {
            initChart()
          } else {
            updateChartData()
          }
        }
      })
    },
    { deep: true }
  )

  // 组件挂载时初始化
  onMounted(() => {
    if (hasData.value && props.podData?.trend?.length) {
      nextTick(() => {
        initChart()
      })
    }
    window.addEventListener('resize', handleResize)
  })

  // 组件卸载时清理
  onUnmounted(() => {
    window.removeEventListener('resize', handleResize)
    if (chart) {
      chart.dispose()
      chart = null
    }
  })
</script>

<style lang="scss" scoped>
  // 颜色变量定义
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

  .pod-metrics {
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
      background: linear-gradient(135deg, #dbeafe 0%, #bfdbfe 100%);
      border-bottom: 1px solid #60a5fa;
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

          &.pod {
            background: linear-gradient(135deg, $primary, #2563eb);
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
        gap: 8px;
        flex-wrap: wrap;

        .stat-badge {
          padding: 6px 12px;
          border-radius: 6px;
          display: flex;
          align-items: center;
          gap: 6px;
          font-size: 12px;

          .label {
            opacity: 0.8;
          }

          .value {
            font-weight: 700;
            font-size: 14px;
          }

          &.total {
            background: rgba(255, 255, 255, 0.9);
            color: #1e40af;
          }

          &.running {
            background: linear-gradient(135deg, #d1fae5 0%, #a7f3d0 100%);
            color: #065f46;
          }

          &.pending {
            background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
            color: #92400e;
          }

          &.failed {
            background: linear-gradient(135deg, #fee2e2 0%, #fecaca 100%);
            color: #991b1b;
          }

          &.restarts {
            background: linear-gradient(135deg, #fee2e2 0%, #fecaca 100%);
            color: #991b1b;
            animation: pulse 2s ease-in-out infinite;

            svg {
              flex-shrink: 0;
            }
          }
        }
      }
    }

    // 内容区域
    .content-wrapper {
      padding: 20px;
    }

    // 状态概览区域
    .status-overview {
      margin-bottom: 20px;

      .overview-title {
        font-size: 13px;
        font-weight: 600;
        color: $gray-600;
        margin-bottom: 14px;
      }

      .status-grid {
        display: grid;
        grid-template-columns: repeat(5, 1fr);
        gap: 14px;

        @media (max-width: 1200px) {
          grid-template-columns: repeat(3, 1fr);
        }

        @media (max-width: 768px) {
          grid-template-columns: repeat(2, 1fr);
        }

        @media (max-width: 480px) {
          grid-template-columns: 1fr;
        }
      }

      .status-item {
        display: flex;
        flex-direction: column;
        padding: 16px;
        background: $gray-50;
        border-radius: $radius-sm;
        border: 1px solid $gray-200;
        border-left: 4px solid;
        transition: all 0.2s ease;

        &:hover {
          border-color: $gray-300;
        }

        &.running {
          border-left-color: $success;
          .status-icon {
            color: $success;
          }
          .status-progress .progress-fill {
            background: $success;
          }
        }

        &.pending {
          border-left-color: $warning;
          .status-icon {
            color: $warning;
          }
          .status-progress .progress-fill {
            background: $warning;
          }
        }

        &.failed {
          border-left-color: $danger;
          .status-icon {
            color: $danger;
          }
          .status-progress .progress-fill {
            background: $danger;
          }
        }

        &.succeeded {
          border-left-color: $primary;
          .status-icon {
            color: $primary;
          }
          .status-progress .progress-fill {
            background: $primary;
          }
        }

        &.unknown {
          border-left-color: $gray-400;
          .status-icon {
            color: $gray-400;
          }
          .status-progress .progress-fill {
            background: $gray-400;
          }
        }

        .status-icon {
          margin-bottom: 10px;
        }

        .status-info {
          display: flex;
          flex-direction: column;
          gap: 4px;

          .status-label {
            font-size: 12px;
            color: $gray-500;
          }

          .status-value {
            font-size: 24px;
            font-weight: 700;
            color: $gray-800;
          }
        }

        .status-progress {
          margin-top: 12px;
          height: 4px;
          background: $gray-200;
          border-radius: 2px;
          overflow: hidden;

          .progress-fill {
            height: 100%;
            transition: width 0.4s ease;
          }
        }
      }
    }

    // 图表区域
    .chart-area {
      padding: 18px;
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

              &.total {
                background: $primary;
              }

              &.running {
                background: $success;
              }

              &.pending {
                background: $warning;
              }

              &.failed {
                background: $danger;
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

    // 空状态
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

  // 动画
  @keyframes pulse {
    0%,
    100% {
      transform: scale(1);
    }
    50% {
      transform: scale(1.02);
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
</style>
