<template>
  <div class="network-monitor" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="monitor-container">
        <!-- 总览卡片 -->
        <div class="overview-card">
          <div class="overview-header">
            <div class="header-left">
              <div class="header-icon">
                <svg viewBox="0 0 24 24" width="24" height="24">
                  <path
                    fill="currentColor"
                    d="M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2M12,4A8,8 0 0,1 20,12A8,8 0 0,1 12,20A8,8 0 0,1 4,12A8,8 0 0,1 12,4M12,6A6,6 0 0,0 6,12A6,6 0 0,0 12,18A6,6 0 0,0 18,12A6,6 0 0,0 12,6M12,8A4,4 0 0,1 16,12A4,4 0 0,1 12,16A4,4 0 0,1 8,12A4,4 0 0,1 12,8Z"
                  />
                </svg>
              </div>
              <div>
                <h2>集群网络</h2>
                <p class="subtitle">网络流量与带宽实时监控</p>
              </div>
            </div>
            <div class="health-badge" :class="networkHealthStatus.status">
              <span class="status-dot"></span>
              <span class="status-text">{{ networkHealthStatus.text }}</span>
            </div>
          </div>

          <!-- 快速指标 -->
          <div class="quick-metrics">
            <div class="quick-metric">
              <div class="metric-icon ingress">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M19,12V15H12V12H19M19,3H5A2,2 0 0,0 3,5V19A2,2 0 0,0 5,21H19A2,2 0 0,0 21,19V5A2,2 0 0,0 19,3M5,7H19V5H5V7M5,9V19H19V9H5Z"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">入流量速率</span>
                <span class="metric-value">{{
                  formatBitRate(networkData?.totalIngressBytesPerSec || 0)
                }}</span>
                <span class="metric-sub">Ingress Traffic</span>
              </div>
            </div>
            <div class="quick-metric">
              <div class="metric-icon egress">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M5,12V9H12V12H5M19,3H5A2,2 0 0,0 3,5V19A2,2 0 0,0 5,21H19A2,2 0 0,0 21,19V5A2,2 0 0,0 19,3M5,7H19V5H5V7M5,9V19H19V9H5Z"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">出流量速率</span>
                <span class="metric-value">{{
                  formatBitRate(networkData?.totalEgressBytesPerSec || 0)
                }}</span>
                <span class="metric-sub">Egress Traffic</span>
              </div>
            </div>
            <div class="quick-metric">
              <div class="metric-icon total">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M12,2A10,10 0 0,1 22,12A10,10 0 0,1 12,22A10,10 0 0,1 2,12A10,10 0 0,1 12,2M12,4A8,8 0 0,0 4,12A8,8 0 0,0 12,20A8,8 0 0,0 20,12A8,8 0 0,0 12,4M12,6A6,6 0 0,1 18,12A6,6 0 0,1 12,18A6,6 0 0,1 6,12A6,6 0 0,1 12,6M12,8A4,4 0 0,0 8,12A4,4 0 0,0 12,16A4,4 0 0,0 16,12A4,4 0 0,0 12,8Z"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">总流量速率</span>
                <span class="metric-value">{{ formatBitRate(totalTrafficRate) }}</span>
                <span class="metric-sub">Total Bandwidth</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 网络监控卡片 -->
        <div class="monitor-card">
          <div class="card-header">
            <div class="header-info">
              <div class="icon-badge network">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2M12,4A8,8 0 0,1 20,12A8,8 0 0,1 12,20A8,8 0 0,1 4,12A8,8 0 0,1 12,4M12,6A6,6 0 0,0 6,12A6,6 0 0,0 12,18A6,6 0 0,0 18,12A6,6 0 0,0 12,6M12,8A4,4 0 0,1 16,12A4,4 0 0,1 12,16A4,4 0 0,1 8,12A4,4 0 0,1 12,8Z"
                  />
                </svg>
              </div>
              <div>
                <h3>网络流量监控</h3>
                <p>入站与出站流量实时趋势</p>
              </div>
            </div>
            <div class="data-info" v-if="networkData?.trend">
              <span class="info-badge">{{ networkData.trend.length }} 个数据点</span>
            </div>
          </div>

          <div class="traffic-stats">
            <div class="traffic-stat ingress">
              <div class="stat-icon">
                <svg viewBox="0 0 24 24" width="20" height="20">
                  <path
                    fill="currentColor"
                    d="M16,13V21H8V13H2L12,3L22,13H16M12,8.41L9.41,11H11V19H13V11H14.59L12,8.41Z"
                  />
                </svg>
              </div>
              <div class="stat-content">
                <div class="stat-label">入站流量</div>
                <div class="stat-value">{{
                  formatBitRate(networkData?.totalIngressBytesPerSec || 0)
                }}</div>
                <div class="stat-bar">
                  <div
                    class="bar-fill ingress"
                    :style="{ width: getTrafficPercent('ingress') + '%' }"
                  ></div>
                </div>
              </div>
            </div>
            <div class="traffic-stat egress">
              <div class="stat-icon">
                <svg viewBox="0 0 24 24" width="20" height="20">
                  <path
                    fill="currentColor"
                    d="M16,11V3H8V11H2L12,21L22,11H16M12,18.59L9.41,16H11V8H13V16H14.59L12,18.59Z"
                  />
                </svg>
              </div>
              <div class="stat-content">
                <div class="stat-label">出站流量</div>
                <div class="stat-value">{{
                  formatBitRate(networkData?.totalEgressBytesPerSec || 0)
                }}</div>
                <div class="stat-bar">
                  <div
                    class="bar-fill egress"
                    :style="{ width: getTrafficPercent('egress') + '%' }"
                  ></div>
                </div>
              </div>
            </div>
          </div>

          <div class="chart-area">
            <div class="chart-header">
              <span class="chart-title">网络流量趋势</span>
              <div class="chart-legend">
                <span class="legend-item"><i class="dot ingress"></i>入流量</span>
                <span class="legend-item"><i class="dot egress"></i>出流量</span>
              </div>
            </div>
            <div ref="networkChartRef" class="chart-container"></div>
          </div>

          <div class="detail-panel">
            <div class="panel-toggle" @click="toggleDetail">
              <span>流量详细统计</span>
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
                  <div class="section-title">流量统计</div>
                  <div class="detail-grid">
                    <div class="detail-item">
                      <span class="label">入站速率峰值</span>
                      <span class="value">{{ formatBitRate(peakIngress) }}</span>
                    </div>
                    <div class="detail-item">
                      <span class="label">出站速率峰值</span>
                      <span class="value">{{ formatBitRate(peakEgress) }}</span>
                    </div>
                    <div class="detail-item">
                      <span class="label">入站平均速率</span>
                      <span class="value">{{ formatBitRate(avgIngress) }}</span>
                    </div>
                    <div class="detail-item">
                      <span class="label">出站平均速率</span>
                      <span class="value">{{ formatBitRate(avgEgress) }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </Transition>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" width="64" height="64">
            <path
              fill="currentColor"
              d="M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2M12,4A8,8 0 0,1 20,12A8,8 0 0,1 12,20A8,8 0 0,1 4,12A8,8 0 0,1 12,4Z"
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
  import type { ClusterNetworkMetrics } from '@/api/console/cluster-monitor'
  import { formatBitRate } from '@/utils/format'

  interface Props {
    networkData?: ClusterNetworkMetrics
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    networkData: undefined,
    loading: false
  })

  const networkChartRef = ref<HTMLElement>()
  let networkChart: echarts.ECharts | null = null
  let resizeTimer: NodeJS.Timeout | null = null
  const showDetail = ref(false)

  const toggleDetail = () => (showDetail.value = !showDetail.value)

  const hasData = computed(() => {
    return !!(props.networkData && Object.keys(props.networkData).length > 0)
  })

  const totalTrafficRate = computed(() => {
    const ingress = props.networkData?.totalIngressBytesPerSec || 0
    const egress = props.networkData?.totalEgressBytesPerSec || 0
    return ingress + egress
  })

  // 网络健康状态
  const networkHealthStatus = computed(() => {
    if (!props.networkData) return { status: 'unknown', text: '未知' }
    const total = totalTrafficRate.value
    if (total > 1000000000) return { status: 'warning', text: '高负载' }
    return { status: 'healthy', text: '正常' }
  })

  // 获取流量百分比
  const getTrafficPercent = (type: 'ingress' | 'egress'): number => {
    const total = totalTrafficRate.value
    if (total === 0) return 0
    const value =
      type === 'ingress'
        ? props.networkData?.totalIngressBytesPerSec || 0
        : props.networkData?.totalEgressBytesPerSec || 0
    return Math.min((value / total) * 100, 100)
  }

  // 峰值和平均值计算
  const peakIngress = computed(() => {
    if (!props.networkData?.trend) return 0
    return Math.max(...props.networkData.trend.map((t) => t.ingressBytesPerSec || 0))
  })

  const peakEgress = computed(() => {
    if (!props.networkData?.trend) return 0
    return Math.max(...props.networkData.trend.map((t) => t.egressBytesPerSec || 0))
  })

  const avgIngress = computed(() => {
    if (!props.networkData?.trend || props.networkData.trend.length === 0) return 0
    const sum = props.networkData.trend.reduce((acc, t) => acc + (t.ingressBytesPerSec || 0), 0)
    return sum / props.networkData.trend.length
  })

  const avgEgress = computed(() => {
    if (!props.networkData?.trend || props.networkData.trend.length === 0) return 0
    const sum = props.networkData.trend.reduce((acc, t) => acc + (t.egressBytesPerSec || 0), 0)
    return sum / props.networkData.trend.length
  })

  const initChart = () => {
    if (!networkChartRef.value) return
    networkChart = echarts.init(networkChartRef.value)
    networkChart.setOption({
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(255,255,255,0.98)',
        borderColor: '#eaecef',
        borderWidth: 1,
        padding: [12, 16],
        textStyle: { color: '#1f2937', fontSize: 12 },
        formatter: (params: any) => {
          if (!Array.isArray(params)) return ''
          let result = `<div style="font-weight: 600; margin-bottom: 8px;">${new Date(params[0].value[0]).toLocaleString()}</div>`
          params.forEach((item: any) => {
            const value = formatBitRate(item.value[1])
            result += `<div style="margin: 4px 0; display: flex; align-items: center;">
              <span style="display: inline-block; width: 10px; height: 10px; border-radius: 50%; background: ${item.color}; margin-right: 8px;"></span>
              <span style="flex: 1;">${item.seriesName}:</span>
              <span style="font-weight: 600; margin-left: 12px;">${value}</span>
            </div>`
          })
          return result
        }
      },
      grid: { left: '3%', right: '4%', bottom: '8%', top: '12%', containLabel: true },
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
        name: '流量速率',
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
      series: []
    })
  }

  const updateChart = () => {
    if (!networkChart || !props.networkData?.trend || props.networkData.trend.length === 0) return

    const ingressData = props.networkData.trend.map((item) => [
      item.timestamp * 1000,
      item.ingressBytesPerSec
    ])
    const egressData = props.networkData.trend.map((item) => [
      item.timestamp * 1000,
      item.egressBytesPerSec
    ])

    networkChart.setOption({
      series: [
        {
          name: '入流量',
          type: 'line',
          smooth: true,
          data: ingressData,
          symbol: 'none',
          lineStyle: { width: 2, color: '#4facfe' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(79, 172, 254, 0.25)' },
              { offset: 1, color: 'rgba(79, 172, 254, 0.05)' }
            ])
          }
        },
        {
          name: '出流量',
          type: 'line',
          smooth: true,
          data: egressData,
          symbol: 'none',
          lineStyle: { width: 2, color: '#fa8c16' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(250, 140, 22, 0.25)' },
              { offset: 1, color: 'rgba(250, 140, 22, 0.05)' }
            ])
          }
        }
      ]
    })
  }

  const handleResize = () => {
    if (resizeTimer) clearTimeout(resizeTimer)
    resizeTimer = setTimeout(() => {
      networkChart?.resize()
    }, 300)
  }

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
    { deep: true, immediate: true }
  )

  onMounted(() => {
    window.addEventListener('resize', handleResize)
  })
  onUnmounted(() => {
    window.removeEventListener('resize', handleResize)
    if (resizeTimer) clearTimeout(resizeTimer)
    networkChart?.dispose()
  })
</script>

<style lang="scss" scoped>
  $primary: #3b82f6;
  $success: #10b981;
  $warning: #f59e0b;
  $danger: #ef4444;
  $ingress: #4facfe;
  $egress: #fa8c16;
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
    background: $gray-50;
    font-family: var(--ax-font-sans);

    .monitor-container {
      max-width: 1800px;
      margin: 0 auto;
      display: flex;
      flex-direction: column;
      gap: 24px;
    }

    .overview-card {
      background: #fff;
      border-radius: $radius;
      padding: 24px;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
      .overview-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 20px;
        padding-bottom: 16px;
        border-bottom: 1px solid $gray-200;
        .header-left {
          display: flex;
          align-items: center;
          gap: 14px;
          .header-icon {
            width: 44px;
            height: 44px;
            border-radius: $radius-sm;
            background: linear-gradient(135deg, #e0f7ff, #bae6fd);
            display: flex;
            align-items: center;
            justify-content: center;
            color: $ingress;
          }
          h2 {
            margin: 0;
            font-size: 18px;
            font-weight: 600;
            color: $gray-800;
          }
          .subtitle {
            margin: 4px 0 0;
            font-size: 13px;
            color: $gray-500;
          }
        }
        .health-badge {
          display: flex;
          align-items: center;
          gap: 10px;
          padding: 10px 18px;
          border-radius: 24px;
          font-weight: 500;
          font-size: 14px;
          .status-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            animation: pulse 2s infinite;
          }
          &.healthy {
            background: rgba($success, 0.1);
            color: #059669;
            .status-dot {
              background: $success;
            }
          }
          &.warning {
            background: rgba($warning, 0.1);
            color: #d97706;
            .status-dot {
              background: $warning;
            }
          }
        }
      }
      .quick-metrics {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 16px;
        @media (max-width: 768px) {
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
              background: linear-gradient(135deg, #e0f7ff, #bae6fd);
              color: $ingress;
            }
            &.egress {
              background: linear-gradient(135deg, #ffedd5, #fed7aa);
              color: $egress;
            }
            &.total {
              background: linear-gradient(135deg, #e0e7ff, #c7d2fe);
              color: $primary;
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
            .metric-sub {
              font-size: 11px;
              color: $gray-400;
              margin-top: 2px;
              display: block;
            }
          }
        }
      }
    }

    .monitor-card {
      background: #fff;
      border-radius: $radius;
      padding: 24px;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
      .card-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 20px;
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
              background: linear-gradient(135deg, $ingress, #00c6fb);
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
        .data-info .info-badge {
          padding: 6px 12px;
          background: $gray-100;
          border-radius: 12px;
          font-size: 12px;
          color: $gray-600;
        }
      }

      .traffic-stats {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 16px;
        margin-bottom: 20px;
        .traffic-stat {
          display: flex;
          align-items: center;
          gap: 14px;
          padding: 18px;
          border-radius: $radius-sm;
          border: 1px solid $gray-200;
          &.ingress {
            background: rgba($ingress, 0.04);
            border-color: rgba($ingress, 0.2);
            .stat-icon {
              background: rgba($ingress, 0.1);
              color: $ingress;
            }
            .bar-fill.ingress {
              background: $ingress;
            }
          }
          &.egress {
            background: rgba($egress, 0.04);
            border-color: rgba($egress, 0.2);
            .stat-icon {
              background: rgba($egress, 0.1);
              color: $egress;
            }
            .bar-fill.egress {
              background: $egress;
            }
          }
          .stat-icon {
            width: 40px;
            height: 40px;
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
              font-size: 22px;
              font-weight: 700;
              color: $gray-800;
              margin-bottom: 8px;
            }
            .stat-bar {
              height: 4px;
              background: $gray-200;
              border-radius: 2px;
              overflow: hidden;
              .bar-fill {
                height: 100%;
                border-radius: 2px;
                transition: width 0.4s ease;
              }
            }
          }
        }
      }

      .chart-area {
        padding: 18px;
        background: $gray-50;
        border-radius: $radius-sm;
        margin-bottom: 16px;
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
                &.ingress {
                  background: $ingress;
                }
                &.egress {
                  background: $egress;
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

      .detail-panel {
        border-top: 1px solid $gray-200;
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
              grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
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
                }
              }
            }
          }
        }
      }
    }

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

  @keyframes pulse {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0.5;
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
