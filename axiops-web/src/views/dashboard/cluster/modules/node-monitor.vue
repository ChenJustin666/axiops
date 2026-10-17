<template>
  <div class="node-monitor" v-loading="loading">
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
                    d="M20 18c1.1 0 1.99-.9 1.99-2L22 6c0-1.1-.9-2-2-2H4c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2H0v2h24v-2h-4zM4 6h16v10H4V6z"
                  />
                </svg>
              </div>
              <div>
                <h2>节点状态</h2>
                <p class="subtitle">集群节点健康与资源监控</p>
              </div>
            </div>
            <div class="health-badge" :class="overallNodeHealth.status">
              <span class="status-dot"></span>
              <span class="status-text">{{ overallNodeHealth.text }}</span>
            </div>
          </div>

          <!-- 快速指标 -->
          <div class="quick-metrics">
            <div class="quick-metric">
              <div class="metric-icon total">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M4 8h4V4H4v4zm6 12h4v-4h-4v4zm-6 0h4v-4H4v4zm0-6h4v-4H4v4zm6 0h4v-4h-4v4zm6-10v4h4V4h-4zm-6 4h4V4h-4v4zm6 6h4v-4h-4v4zm0 6h4v-4h-4v4z"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">总节点</span>
                <span class="metric-value">{{ nodeData?.total || 0 }}</span>
              </div>
            </div>
            <div class="quick-metric">
              <div class="metric-icon ready">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">就绪节点</span>
                <span class="metric-value">{{ nodeData?.ready || 0 }}</span>
              </div>
            </div>
            <div class="quick-metric">
              <div class="metric-icon not-ready">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M12 2C6.47 2 2 6.47 2 12s4.47 10 10 10 10-4.47 10-10S17.53 2 12 2zm5 13.59L15.59 17 12 13.41 8.41 17 7 15.59 10.59 12 7 8.41 8.41 7 12 10.59 15.59 7 17 8.41 13.41 12 17 15.59z"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">未就绪</span>
                <span class="metric-value" :class="{ danger: (nodeData?.notReady || 0) > 0 }">{{
                  nodeData?.notReady || 0
                }}</span>
              </div>
            </div>
            <div class="quick-metric">
              <div class="metric-icon cpu">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M9 3V1H7v2H5v2H3v2h2v2H3v2h2v2H3v2h2v2h2v2h2v-2h2v2h2v-2h2v2h2v-2h2v-2h2v-2h-2v-2h2v-2h-2V9h2V7h-2V5h-2V3h-2v2h-2V3h-2v2H9V3zm0 4h6v2h2v6h-2v2H9v-2H7V9h2V7z"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">平均 CPU</span>
                <span class="metric-value">{{ formatPercent3(nodeData?.avgCPUUsage) }}</span>
              </div>
            </div>
            <div class="quick-metric">
              <div class="metric-icon memory">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M2 7v10h20V7H2zm18 8H4V9h16v6zM6 11h2v2H6v-2zm3 0h2v2H9v-2zm3 0h2v2h-2v-2zm3 0h2v2h-2v-2z"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">平均内存</span>
                <span class="metric-value">{{ formatPercent3(nodeData?.avgMemoryUsage) }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 图表区域 -->
        <div class="charts-row">
          <div class="chart-card">
            <div class="chart-header">
              <span class="chart-title">节点状态趋势</span>
              <div class="chart-legend">
                <span class="legend-item"><i class="dot ready"></i>就绪节点</span>
                <span class="legend-item"><i class="dot not-ready"></i>未就绪节点</span>
              </div>
            </div>
            <div ref="nodeStatusChartRef" class="chart-container"></div>
          </div>

          <div class="chart-card">
            <div class="chart-header">
              <span class="chart-title">平均资源使用趋势</span>
              <div class="chart-legend">
                <span class="legend-item"><i class="dot cpu"></i>平均 CPU</span>
                <span class="legend-item"><i class="dot memory"></i>平均内存</span>
              </div>
            </div>
            <div ref="nodeResourceChartRef" class="chart-container"></div>
          </div>
        </div>

        <!-- 节点列表 -->
        <div v-if="nodeData?.nodeList && nodeData.nodeList.length > 0" class="nodes-section">
          <div class="section-header">
            <div class="header-left">
              <div class="section-icon">
                <svg viewBox="0 0 24 24" width="20" height="20">
                  <path
                    fill="currentColor"
                    d="M3 13h2v-2H3v2zm0 4h2v-2H3v2zm0-8h2V7H3v2zm4 4h14v-2H7v2zm0 4h14v-2H7v2zM7 7v2h14V7H7z"
                  />
                </svg>
              </div>
              <span class="section-title">节点详情</span>
            </div>
            <div class="section-badge">{{ nodeData.nodeList.length }} 个节点</div>
          </div>

          <div class="nodes-grid">
            <div
              v-for="node in nodeData.nodeList"
              :key="node.nodeName"
              class="node-item"
              :class="{ 'not-ready': !node.ready }"
            >
              <div class="node-header">
                <div class="node-name">
                  <svg viewBox="0 0 24 24" width="18" height="18">
                    <path
                      fill="currentColor"
                      d="M20 18c1.1 0 1.99-.9 1.99-2L22 6c0-1.1-.9-2-2-2H4c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2H0v2h24v-2h-4zM4 6h16v10H4V6z"
                    />
                  </svg>
                  <span>{{ node.nodeName }}</span>
                </div>
                <div class="node-status" :class="node.ready ? 'ready' : 'not-ready'">
                  {{ node.ready ? '就绪' : '未就绪' }}
                </div>
              </div>

              <div class="node-metrics">
                <div class="metric-row">
                  <span class="metric-label">CPU 使用率</span>
                  <span class="metric-value">{{ node.cpuUsage?.toFixed(2) || 0 }}%</span>
                </div>
                <div class="metric-progress">
                  <div
                    class="progress-fill"
                    :class="getProgressClass(node.cpuUsage)"
                    :style="{ width: `${node.cpuUsage || 0}%` }"
                  ></div>
                </div>

                <div class="metric-row">
                  <span class="metric-label">内存使用率</span>
                  <span class="metric-value">{{ node.memoryUsage?.toFixed(2) || 0 }}%</span>
                </div>
                <div class="metric-progress">
                  <div
                    class="progress-fill"
                    :class="getProgressClass(node.memoryUsage)"
                    :style="{ width: `${node.memoryUsage || 0}%` }"
                  ></div>
                </div>

                <div class="metric-row pods">
                  <span class="metric-label">Pod 数量</span>
                  <span class="metric-value">{{ node.podsCount || 0 }}</span>
                </div>
              </div>

              <div v-if="node.memoryPressure || node.diskPressure" class="node-warnings">
                <div v-if="node.memoryPressure" class="warning-tag memory">
                  <svg viewBox="0 0 24 24" width="12" height="12">
                    <path
                      fill="currentColor"
                      d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"
                    />
                  </svg>
                  内存压力
                </div>
                <div v-if="node.diskPressure" class="warning-tag disk">
                  <svg viewBox="0 0 24 24" width="12" height="12">
                    <path
                      fill="currentColor"
                      d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"
                    />
                  </svg>
                  磁盘压力
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" width="64" height="64">
            <path
              fill="currentColor"
              d="M20 18c1.1 0 1.99-.9 1.99-2L22 6c0-1.1-.9-2-2-2H4c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2H0v2h24v-2h-4zM4 6h16v10H4V6z"
            />
          </svg>
        </div>
        <h3>暂无节点监控数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, onUnmounted, nextTick, computed } from 'vue'
  import * as echarts from 'echarts'
  import type { ClusterNodeMetrics } from '@/api/console/cluster-monitor'

  interface Props {
    nodeData?: ClusterNodeMetrics
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    nodeData: undefined,
    loading: false
  })

  const nodeStatusChartRef = ref<HTMLElement>()
  const nodeResourceChartRef = ref<HTMLElement>()
  let nodeStatusChart: echarts.ECharts | null = null
  let nodeResourceChart: echarts.ECharts | null = null
  let resizeTimer: NodeJS.Timeout | null = null

  const hasData = computed(() => {
    return !!(props.nodeData && Object.keys(props.nodeData).length > 0)
  })

  // 整体节点健康状态
  const overallNodeHealth = computed(() => {
    if (!props.nodeData) return { status: 'unknown', text: '未知' }
    const notReady = props.nodeData.notReady || 0
    const total = props.nodeData.total || 0

    if (notReady > 0) {
      return { status: 'critical', text: `${notReady} 个异常` }
    }
    if (total === 0) {
      return { status: 'unknown', text: '无节点' }
    }
    return { status: 'healthy', text: '全部健康' }
  })

  // 格式化百分比：保留3位小数
  const formatPercent3 = (value?: number) => {
    if (value === undefined || value === null) return '0.000%'
    return `${value.toFixed(3)}%`
  }

  const getProgressClass = (value?: number): string => {
    if (!value) return 'normal'
    if (value < 60) return 'normal'
    if (value < 80) return 'warning'
    return 'danger'
  }

  const initNodeStatusChart = () => {
    if (!nodeStatusChartRef.value) return
    nodeStatusChart = echarts.init(nodeStatusChartRef.value)

    const option: echarts.EChartsOption = {
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(255, 255, 255, 0.98)',
        borderColor: '#eaecef',
        borderWidth: 1,
        padding: [12, 16],
        textStyle: { color: '#1f2937', fontSize: 12 },
        formatter: (params: any) => {
          const date = new Date(params[0].axisValue).toLocaleString('zh-CN', {
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
          })
          let html = `<div style="font-weight: 600; margin-bottom: 8px;">${date}</div>`
          params.forEach((param: any) => {
            const value = param.value[1]
            html += `
            <div style="display: flex; align-items: center; gap: 8px; margin: 4px 0;">
              <span style="display: inline-block; width: 10px; height: 10px; border-radius: 50%; background: ${param.color};"></span>
              <span>${param.seriesName}:</span>
              <span style="font-weight: 600;">${value}</span>
            </div>`
          })
          return html
        }
      },
      legend: {
        show: false
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '8%',
        top: '18%', // 修复：增加顶部空间，确保Y轴标签"节点数"完整显示
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
        name: '节点数',
        nameTextStyle: {
          color: '#6b7280',
          fontSize: 12,
          padding: [0, 0, 8, 0] // 修复：增加底部内边距，避免与图表重叠
        },
        nameGap: 15, // 修复：增加名称与轴线的距离
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: { color: '#6b7280', fontSize: 11 },
        splitLine: { lineStyle: { color: '#f3f4f6', type: 'dashed' } }
      },
      series: [
        {
          name: '就绪节点',
          type: 'line',
          smooth: true,
          symbol: 'none',
          data: [],
          lineStyle: { width: 2, color: '#10b981' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(16, 185, 129, 0.2)' },
              { offset: 1, color: 'rgba(16, 185, 129, 0.02)' }
            ])
          }
        },
        {
          name: '未就绪节点',
          type: 'line',
          smooth: true,
          symbol: 'none',
          data: [],
          lineStyle: { width: 2, color: '#ef4444' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(239, 68, 68, 0.2)' },
              { offset: 1, color: 'rgba(239, 68, 68, 0.02)' }
            ])
          }
        }
      ]
    }

    nodeStatusChart.setOption(option)
  }

  const initNodeResourceChart = () => {
    if (!nodeResourceChartRef.value) return
    nodeResourceChart = echarts.init(nodeResourceChartRef.value)

    const option: echarts.EChartsOption = {
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(255, 255, 255, 0.98)',
        borderColor: '#eaecef',
        borderWidth: 1,
        padding: [12, 16],
        textStyle: { color: '#1f2937', fontSize: 12 },
        formatter: (params: any) => {
          const date = new Date(params[0].axisValue).toLocaleString('zh-CN', {
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
          })
          let html = `<div style="font-weight: 600; margin-bottom: 8px;">${date}</div>`
          params.forEach((param: any) => {
            const value = param.value[1]
            const formattedValue = value.toFixed(3) + '%'
            html += `
            <div style="display: flex; align-items: center; gap: 8px; margin: 4px 0;">
              <span style="display: inline-block; width: 10px; height: 10px; border-radius: 50%; background: ${param.color};"></span>
              <span>${param.seriesName}:</span>
              <span style="font-weight: 600;">${formattedValue}</span>
            </div>`
          })
          return html
        }
      },
      legend: {
        show: false
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '8%',
        top: '18%', // 修复：增加顶部空间，确保Y轴标签"使用率 (%)"完整显示
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
        name: '使用率 (%)',
        nameTextStyle: {
          color: '#6b7280',
          fontSize: 12,
          padding: [0, 0, 8, 0] // 修复：增加底部内边距，避免与图表重叠
        },
        nameGap: 15, // 修复：增加名称与轴线的距离
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: {
          color: '#6b7280',
          fontSize: 11,
          formatter: (value: number) => value.toFixed(1) + '%'
        },
        splitLine: { lineStyle: { color: '#f3f4f6', type: 'dashed' } }
      },
      series: [
        {
          name: '平均 CPU 使用率',
          type: 'line',
          smooth: true,
          symbol: 'none',
          data: [],
          lineStyle: { width: 2, color: '#3b82f6' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(59, 130, 246, 0.2)' },
              { offset: 1, color: 'rgba(59, 130, 246, 0.02)' }
            ])
          }
        },
        {
          name: '平均内存使用率',
          type: 'line',
          smooth: true,
          symbol: 'none',
          data: [],
          lineStyle: { width: 2, color: '#a855f7' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(168, 85, 247, 0.2)' },
              { offset: 1, color: 'rgba(168, 85, 247, 0.02)' }
            ])
          }
        }
      ]
    }

    nodeResourceChart.setOption(option)
  }

  const updateCharts = () => {
    if (!props.nodeData) return

    if (nodeStatusChart && props.nodeData.trend) {
      const readyData = props.nodeData.trend.map((item) => [item.timestamp * 1000, item.readyNodes])
      const notReadyData = props.nodeData.trend.map((item) => [
        item.timestamp * 1000,
        item.notReadyNodes
      ])

      nodeStatusChart.setOption({
        series: [{ data: readyData }, { data: notReadyData }]
      })
    }

    if (nodeResourceChart && props.nodeData.trend) {
      const cpuData = props.nodeData.trend.map((item) => [item.timestamp * 1000, item.avgCPUUsage])
      const memoryData = props.nodeData.trend.map((item) => [
        item.timestamp * 1000,
        item.avgMemoryUsage
      ])

      nodeResourceChart.setOption({
        series: [{ data: cpuData }, { data: memoryData }]
      })
    }
  }

  const handleResize = () => {
    if (resizeTimer) clearTimeout(resizeTimer)
    resizeTimer = setTimeout(() => {
      nodeStatusChart?.resize()
      nodeResourceChart?.resize()
    }, 300)
  }

  watch(
    () => props.nodeData,
    () => {
      nextTick(() => {
        if (hasData.value) {
          if (!nodeStatusChart) initNodeStatusChart()
          if (!nodeResourceChart) initNodeResourceChart()
          updateCharts()
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
    nodeStatusChart?.dispose()
    nodeResourceChart?.dispose()
  })
</script>

<style lang="scss" scoped>
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

  .node-monitor {
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
            background: linear-gradient(135deg, #dbeafe, #bfdbfe);
            display: flex;
            align-items: center;
            justify-content: center;
            color: $primary;
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

          &.critical {
            background: rgba($danger, 0.1);
            color: #dc2626;
            .status-dot {
              background: $danger;
            }
          }

          &.unknown {
            background: $gray-100;
            color: $gray-500;
            .status-dot {
              background: $gray-400;
            }
          }
        }
      }

      .quick-metrics {
        display: grid;
        grid-template-columns: repeat(5, 1fr);
        gap: 16px;

        @media (max-width: 1200px) {
          grid-template-columns: repeat(3, 1fr);
        }

        @media (max-width: 768px) {
          grid-template-columns: repeat(2, 1fr);
        }

        .quick-metric {
          display: flex;
          align-items: center;
          gap: 14px;
          padding: 16px;
          border-radius: $radius-sm;
          background: $gray-50;
          border: 1px solid $gray-200;
          transition: all 0.2s ease;

          &:hover {
            border-color: $gray-300;
            transform: translateY(-1px);
          }

          .metric-icon {
            width: 44px;
            height: 44px;
            border-radius: $radius-xs;
            display: flex;
            align-items: center;
            justify-content: center;
            flex-shrink: 0;

            &.total {
              background: linear-gradient(135deg, #e0e7ff, #c7d2fe);
              color: #4338ca;
            }

            &.ready {
              background: linear-gradient(135deg, #d1fae5, #a7f3d0);
              color: #059669;
            }

            &.not-ready {
              background: linear-gradient(135deg, #fee2e2, #fecaca);
              color: #dc2626;
            }

            &.cpu {
              background: linear-gradient(135deg, #dbeafe, #bfdbfe);
              color: $primary;
            }

            &.memory {
              background: linear-gradient(135deg, #f3e8ff, #e9d5ff);
              color: #7c3aed;
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
              font-size: 18px;
              font-weight: 700;
              color: $gray-800;
              display: block;

              &.danger {
                color: $danger;
              }
            }
          }
        }
      }
    }

    .charts-row {
      display: grid;
      grid-template-columns: repeat(2, 1fr);
      gap: 20px;

      @media (max-width: 1200px) {
        grid-template-columns: 1fr;
      }

      .chart-card {
        background: #fff;
        border-radius: $radius;
        padding: 20px;
        box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);

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

                &.ready {
                  background: $success;
                }

                &.not-ready {
                  background: $danger;
                }

                &.cpu {
                  background: $primary;
                }

                &.memory {
                  background: #a855f7;
                }
              }
            }
          }
        }

        .chart-container {
          height: 260px; // 修复：增加图表高度，为Y轴标签留出更多空间
          width: 100%;
        }
      }
    }

    .nodes-section {
      background: #fff;
      border-radius: $radius;
      padding: 24px;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);

      .section-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 20px;
        padding-bottom: 16px;
        border-bottom: 1px solid $gray-200;

        .header-left {
          display: flex;
          align-items: center;
          gap: 10px;

          .section-icon {
            width: 32px;
            height: 32px;
            border-radius: $radius-xs;
            background: $gray-100;
            display: flex;
            align-items: center;
            justify-content: center;
            color: $gray-600;
          }

          .section-title {
            font-size: 15px;
            font-weight: 600;
            color: $gray-700;
          }
        }

        .section-badge {
          padding: 5px 12px;
          border-radius: 12px;
          font-size: 12px;
          font-weight: 500;
          background: rgba($primary, 0.1);
          color: $primary;
        }
      }

      .nodes-grid {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 16px;

        @media (max-width: 1200px) {
          grid-template-columns: repeat(2, 1fr);
        }

        @media (max-width: 768px) {
          grid-template-columns: 1fr;
        }

        .node-item {
          padding: 18px;
          border-radius: $radius-sm;
          background: $gray-50;
          border: 1px solid $gray-200;
          transition: all 0.3s ease;

          &:hover {
            border-color: $primary;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
          }

          &.not-ready {
            border-color: rgba($danger, 0.3);
            background: rgba($danger, 0.02);

            &:hover {
              border-color: $danger;
            }
          }

          .node-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 16px;

            .node-name {
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

            .node-status {
              padding: 4px 10px;
              border-radius: 10px;
              font-size: 11px;
              font-weight: 600;

              &.ready {
                background: rgba($success, 0.1);
                color: #059669;
              }

              &.not-ready {
                background: rgba($danger, 0.1);
                color: #dc2626;
              }
            }
          }

          .node-metrics {
            .metric-row {
              display: flex;
              justify-content: space-between;
              align-items: center;
              margin-bottom: 6px;

              &.pods {
                margin-top: 12px;
                padding-top: 12px;
                border-top: 1px solid $gray-200;
              }

              .metric-label {
                font-size: 12px;
                color: $gray-500;
              }

              .metric-value {
                font-size: 13px;
                font-weight: 600;
                color: $gray-700;
              }
            }

            .metric-progress {
              height: 6px;
              background: $gray-200;
              border-radius: 3px;
              overflow: hidden;
              margin-bottom: 12px;

              .progress-fill {
                height: 100%;
                border-radius: 3px;
                transition: width 0.4s ease;

                &.normal {
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

          .node-warnings {
            display: flex;
            gap: 8px;
            margin-top: 12px;
            padding-top: 12px;
            border-top: 1px solid $gray-200;
            flex-wrap: wrap;

            .warning-tag {
              display: flex;
              align-items: center;
              gap: 4px;
              padding: 4px 10px;
              border-radius: 10px;
              font-size: 11px;
              font-weight: 500;

              &.memory {
                background: rgba($warning, 0.1);
                color: #d97706;
              }

              &.disk {
                background: rgba($danger, 0.1);
                color: #dc2626;
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
</style>
