<template>
  <div class="storage-monitor" v-loading="loading">
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
                    d="M6,2H18A2,2 0 0,1 20,4V20A2,2 0 0,1 18,22H6A2,2 0 0,1 4,20V4A2,2 0 0,1 6,2M12,4A6,6 0 0,0 6,10C6,13.31 8.69,16 12,16A6,6 0 0,0 18,10C18,6.69 15.31,4 12,4M12,9A1,1 0 0,1 13,10A1,1 0 0,1 12,11A1,1 0 0,1 11,10A1,1 0 0,1 12,9M7,18A1,1 0 0,0 6,19A1,1 0 0,0 7,20A1,1 0 0,0 8,19A1,1 0 0,0 7,18Z"
                  />
                </svg>
              </div>
              <div>
                <h2>集群存储</h2>
                <p class="subtitle">PV、PVC、StorageClass 监控与分析</p>
              </div>
            </div>
            <div class="health-badge" :class="storageHealthStatus.status">
              <span class="status-dot"></span>
              <span class="status-text">{{ storageHealthStatus.text }}</span>
              <span class="health-score">{{ storageHealthStatus.score }}/100</span>
            </div>
          </div>

          <!-- 问题告警 -->
          <div v-if="healthIssues.length > 0" class="issues-alert">
            <div class="alert-header">
              <svg viewBox="0 0 24 24" width="18" height="18">
                <path
                  fill="currentColor"
                  d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"
                />
              </svg>
              <span>检测到 {{ healthIssues.length }} 个问题</span>
            </div>
            <div class="issues-list">
              <div v-for="(issue, index) in healthIssues" :key="index" class="issue-item">
                <span class="issue-dot"></span>
                <span>{{ issue }}</span>
              </div>
            </div>
          </div>

          <!-- 快速指标 -->
          <div class="quick-metrics">
            <div class="quick-metric">
              <div class="metric-icon pv">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M2 20h20v-4H2v4zm2-3h2v2H4v-2zM2 4v4h20V4H2zm4 3H4V5h2v2zm-4 7h20v-4H2v4zm2-3h2v2H4v-2z"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">PV 总数</span>
                <span class="metric-value">{{ formatNumber(storageData?.pvTotal) }}</span>
                <span class="metric-sub">已绑定 {{ formatNumber(storageData?.pvBound) }}</span>
              </div>
            </div>
            <div class="quick-metric">
              <div class="metric-icon pvc">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-5 14H7v-2h7v2zm3-4H7v-2h10v2zm0-4H7V7h10v2z"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">PVC 总数</span>
                <span class="metric-value">{{ formatNumber(storageData?.pvcTotal) }}</span>
                <span class="metric-sub">已绑定 {{ formatNumber(storageData?.pvcBound) }}</span>
              </div>
            </div>
            <div class="quick-metric">
              <div class="metric-icon sc">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">StorageClass</span>
                <span class="metric-value">{{ storageData?.storageClasses?.length || 0 }}</span>
                <span class="metric-sub">存储类型</span>
              </div>
            </div>
          </div>
        </div>

        <!-- PV 监控卡片 -->
        <div class="monitor-card">
          <div class="card-header">
            <div class="header-info">
              <div class="icon-badge pv">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M2 20h20v-4H2v4zm2-3h2v2H4v-2zM2 4v4h20V4H2zm4 3H4V5h2v2zm-4 7h20v-4H2v4zm2-3h2v2H4v-2z"
                  />
                </svg>
              </div>
              <div>
                <h3>PersistentVolume</h3>
                <p>持久卷资源状态</p>
              </div>
            </div>
            <div class="status-tag" :class="pvStatus.class">{{ pvStatus.text }}</div>
          </div>

          <div class="metrics-grid">
            <div class="metric-box">
              <div class="box-header">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path
                    fill="currentColor"
                    d="M4 8h4V4H4v4zm6 12h4v-4h-4v4zm-6 0h4v-4H4v4zm0-6h4v-4H4v4zm6 0h4v-4h-4v4zm6-10v4h4V4h-4zm-6 4h4V4h-4v4zm6 6h4v-4h-4v4zm0 6h4v-4h-4v4z"
                  />
                </svg>
                <span>总数</span>
              </div>
              <div class="box-value">
                <span class="value">{{ formatNumber(storageData?.pvTotal) }}</span>
                <span class="unit">个</span>
              </div>
            </div>

            <div class="metric-box success">
              <div class="box-header">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path fill="currentColor" d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                </svg>
                <span>已绑定</span>
              </div>
              <div class="box-value">
                <span class="value success">{{ formatNumber(storageData?.pvBound) }}</span>
                <span class="unit">个</span>
              </div>
              <div class="box-progress"
                ><div class="progress-fill success" :style="{ width: pvBoundPercent + '%' }"></div
              ></div>
            </div>

            <div class="metric-box primary">
              <div class="box-header">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path
                    fill="currentColor"
                    d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V5h14v14z"
                  />
                </svg>
                <span>可用</span>
              </div>
              <div class="box-value">
                <span class="value primary">{{ formatNumber(storageData?.pvAvailable) }}</span>
                <span class="unit">个</span>
              </div>
            </div>

            <div class="metric-box" :class="{ danger: (storageData?.pvFailed || 0) > 0 }">
              <div class="box-header">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path
                    fill="currentColor"
                    d="M12 2C6.47 2 2 6.47 2 12s4.47 10 10 10 10-4.47 10-10S17.53 2 12 2zm5 13.59L15.59 17 12 13.41 8.41 17 7 15.59 10.59 12 7 8.41 8.41 7 12 10.59 15.59 7 17 8.41 13.41 12 17 15.59z"
                  />
                </svg>
                <span>失败</span>
              </div>
              <div class="box-value">
                <span class="value" :class="{ danger: (storageData?.pvFailed || 0) > 0 }">{{
                  formatNumber(storageData?.pvFailed)
                }}</span>
                <span class="unit">个</span>
              </div>
            </div>
          </div>

          <div class="chart-area">
            <div class="chart-header">
              <span class="chart-title">PV 状态分布</span>
              <div class="chart-legend">
                <span class="legend-item"><i class="dot bound"></i>已绑定</span>
                <span class="legend-item"><i class="dot available"></i>可用</span>
                <span class="legend-item"><i class="dot failed"></i>失败</span>
              </div>
            </div>
            <div ref="pvChartRef" class="chart-container"></div>
          </div>
        </div>

        <!-- PVC 监控卡片 -->
        <div class="monitor-card">
          <div class="card-header">
            <div class="header-info">
              <div class="icon-badge pvc">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-5 14H7v-2h7v2zm3-4H7v-2h10v2zm0-4H7V7h10v2z"
                  />
                </svg>
              </div>
              <div>
                <h3>PersistentVolumeClaim</h3>
                <p>持久卷声明状态</p>
              </div>
            </div>
            <div class="status-tag" :class="pvcStatus.class">{{ pvcStatus.text }}</div>
          </div>

          <div class="metrics-grid">
            <div class="metric-box">
              <div class="box-header">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path
                    fill="currentColor"
                    d="M4 8h4V4H4v4zm6 12h4v-4h-4v4zm-6 0h4v-4H4v4zm0-6h4v-4H4v4zm6 0h4v-4h-4v4zm6-10v4h4V4h-4zm-6 4h4V4h-4v4zm6 6h4v-4h-4v4zm0 6h4v-4h-4v4z"
                  />
                </svg>
                <span>总数</span>
              </div>
              <div class="box-value">
                <span class="value">{{ formatNumber(storageData?.pvcTotal) }}</span>
                <span class="unit">个</span>
              </div>
            </div>

            <div class="metric-box success">
              <div class="box-header">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path fill="currentColor" d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                </svg>
                <span>已绑定</span>
              </div>
              <div class="box-value">
                <span class="value success">{{ formatNumber(storageData?.pvcBound) }}</span>
                <span class="unit">个</span>
              </div>
              <div class="box-progress"
                ><div class="progress-fill success" :style="{ width: pvcBoundPercent + '%' }"></div
              ></div>
            </div>

            <div class="metric-box" :class="{ warning: (storageData?.pvcPending || 0) > 0 }">
              <div class="box-header">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path
                    fill="currentColor"
                    d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10 10-4.5 10-10S17.5 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67V7z"
                  />
                </svg>
                <span>待绑定</span>
              </div>
              <div class="box-value">
                <span class="value" :class="{ warning: (storageData?.pvcPending || 0) > 0 }">{{
                  formatNumber(storageData?.pvcPending)
                }}</span>
                <span class="unit">个</span>
              </div>
            </div>

            <div class="metric-box" :class="{ danger: (storageData?.pvcLost || 0) > 0 }">
              <div class="box-header">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path
                    fill="currentColor"
                    d="M12 2C6.47 2 2 6.47 2 12s4.47 10 10 10 10-4.47 10-10S17.53 2 12 2zm5 13.59L15.59 17 12 13.41 8.41 17 7 15.59 10.59 12 7 8.41 8.41 7 12 10.59 15.59 7 17 8.41 13.41 12 17 15.59z"
                  />
                </svg>
                <span>丢失</span>
              </div>
              <div class="box-value">
                <span class="value" :class="{ danger: (storageData?.pvcLost || 0) > 0 }">{{
                  formatNumber(storageData?.pvcLost)
                }}</span>
                <span class="unit">个</span>
              </div>
            </div>
          </div>

          <div class="chart-area">
            <div class="chart-header">
              <span class="chart-title">PVC 状态分布</span>
              <div class="chart-legend">
                <span class="legend-item"><i class="dot bound"></i>已绑定</span>
                <span class="legend-item"><i class="dot pending"></i>待绑定</span>
                <span class="legend-item"><i class="dot lost"></i>丢失</span>
              </div>
            </div>
            <div ref="pvcChartRef" class="chart-container"></div>
          </div>
        </div>

        <!-- StorageClass 使用情况 -->
        <div
          v-if="storageData?.storageClasses && storageData.storageClasses.length > 0"
          class="monitor-card"
        >
          <div class="card-header">
            <div class="header-info">
              <div class="icon-badge sc">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"
                  />
                </svg>
              </div>
              <div>
                <h3>StorageClass 使用情况</h3>
                <p>各存储类资源分配统计</p>
              </div>
            </div>
            <div class="stats-badge">
              <span class="badge-item">
                <span class="badge-label">类型数</span>
                <span class="badge-value">{{ storageData.storageClasses.length }}</span>
              </span>
            </div>
          </div>

          <div class="sc-list">
            <div
              v-for="(sc, index) in storageData.storageClasses"
              :key="sc.storageClass"
              class="sc-item"
            >
              <div class="sc-rank" :class="getRankClass(index)">{{ index + 1 }}</div>
              <div class="sc-info">
                <div class="sc-name">
                  <svg viewBox="0 0 24 24" width="16" height="16">
                    <path
                      fill="currentColor"
                      d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"
                    />
                  </svg>
                  <span :title="sc.storageClass">{{ sc.storageClass }}</span>
                </div>
                <div class="sc-bar">
                  <div
                    class="sc-bar-fill"
                    :style="{ width: getScBarWidth(sc.totalBytes) + '%' }"
                  ></div>
                </div>
              </div>
              <div class="sc-stats">
                <div class="sc-stat">
                  <span class="sc-stat-value">{{ sc.pvcCount }}</span>
                  <span class="sc-stat-label">PVC</span>
                </div>
                <div class="sc-stat">
                  <span class="sc-stat-value">{{ formatBytesCompact(sc.totalBytes) }}</span>
                  <span class="sc-stat-label">容量</span>
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
              d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14h-2V9h2v8zm0-10h-2V5h2v2z"
            />
          </svg>
        </div>
        <h3>暂无存储监控数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, onUnmounted, nextTick, computed } from 'vue'
  import * as echarts from 'echarts'
  import type { ClusterStorageMetrics } from '@/api/console/cluster-monitor'
  import { formatBytes } from '@/utils/format'

  interface Props {
    storageData?: ClusterStorageMetrics
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    storageData: undefined,
    loading: false
  })

  const pvChartRef = ref<HTMLElement>()
  const pvcChartRef = ref<HTMLElement>()
  let pvChart: echarts.ECharts | null = null
  let pvcChart: echarts.ECharts | null = null
  let resizeTimer: NodeJS.Timeout | null = null

  const hasData = computed(() => {
    return !!(props.storageData && Object.keys(props.storageData).length > 0)
  })

  // 格式化数字
  const formatNumber = (value?: number): string => {
    if (value === undefined || value === null || isNaN(value)) return '0'
    if (value < 1000) return String(value)
    if (value < 10000) return `${(value / 1000).toFixed(1)}K`
    if (value < 1000000) return `${Math.round(value / 1000)}K`
    return `${(value / 1000000).toFixed(1)}M`
  }

  // 格式化字节（紧凑）
  const formatBytesCompact = (bytes?: number): string => {
    if (!bytes || bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    const value = bytes / Math.pow(k, i)
    return `${value >= 100 ? Math.round(value) : value.toFixed(1)} ${sizes[i]}`
  }

  // PV 绑定百分比
  const pvBoundPercent = computed(() => {
    const total = props.storageData?.pvTotal || 0
    const bound = props.storageData?.pvBound || 0
    if (total === 0) return 0
    return (bound / total) * 100
  })

  // PVC 绑定百分比
  const pvcBoundPercent = computed(() => {
    const total = props.storageData?.pvcTotal || 0
    const bound = props.storageData?.pvcBound || 0
    if (total === 0) return 0
    return (bound / total) * 100
  })

  // 最大存储容量（用于进度条计算）
  const maxStorageBytes = computed(() => {
    if (!props.storageData?.storageClasses) return 0
    return Math.max(...props.storageData.storageClasses.map((sc) => sc.totalBytes || 0), 1)
  })

  // 获取 SC 进度条宽度
  const getScBarWidth = (bytes?: number): number => {
    if (!bytes || maxStorageBytes.value === 0) return 0
    return (bytes / maxStorageBytes.value) * 100
  }

  // 获取排名样式
  const getRankClass = (index: number): string => {
    if (index === 0) return 'gold'
    if (index === 1) return 'silver'
    if (index === 2) return 'bronze'
    return ''
  }

  // 存储健康状态
  const storageHealthStatus = computed(() => {
    if (!props.storageData) return { status: 'unknown', text: '未知', score: 0 }
    let score = 100

    const pvFailed = props.storageData.pvFailed || 0
    const pvcPending = props.storageData.pvcPending || 0
    const pvcLost = props.storageData.pvcLost || 0

    if (pvFailed > 0) score -= 20
    if (pvcPending > 5) score -= 15
    else if (pvcPending > 0) score -= 5
    if (pvcLost > 0) score -= 25

    score = Math.max(0, score)
    let status: 'healthy' | 'warning' | 'critical' | 'unknown' = 'healthy'
    let text = '健康'
    if (score < 60) {
      status = 'critical'
      text = '严重异常'
    } else if (score < 80) {
      status = 'warning'
      text = '需要关注'
    }
    return { status, text, score: Math.round(score) }
  })

  // 健康问题列表
  const healthIssues = computed(() => {
    const issues: string[] = []
    if (!props.storageData) return issues

    if (props.storageData.pvFailed && props.storageData.pvFailed > 0) {
      issues.push(`有 ${props.storageData.pvFailed} 个 PV 处于失败状态`)
    }
    if (props.storageData.pvcLost && props.storageData.pvcLost > 0) {
      issues.push(`有 ${props.storageData.pvcLost} 个 PVC 已丢失`)
    }
    if (props.storageData.pvcPending && props.storageData.pvcPending > 5) {
      issues.push(`有 ${props.storageData.pvcPending} 个 PVC 待绑定`)
    }
    return issues
  })

  // PV 状态
  const pvStatus = computed(() => {
    const failed = props.storageData?.pvFailed || 0
    if (failed > 0) return { class: 'danger', text: '有失败' }
    return { class: 'success', text: '正常' }
  })

  // PVC 状态
  const pvcStatus = computed(() => {
    const lost = props.storageData?.pvcLost || 0
    const pending = props.storageData?.pvcPending || 0
    if (lost > 0) return { class: 'danger', text: '有丢失' }
    if (pending > 5) return { class: 'warning', text: '待绑定' }
    return { class: 'success', text: '正常' }
  })

  // 初始化 PV 图表
  const initPVChart = () => {
    if (!pvChartRef.value) return
    pvChart = echarts.init(pvChartRef.value)
    pvChart.setOption({
      tooltip: {
        trigger: 'item',
        formatter: '{b}: {c} ({d}%)',
        backgroundColor: 'rgba(255,255,255,0.98)',
        borderColor: '#eaecef',
        borderWidth: 1,
        textStyle: { color: '#1f2937', fontSize: 12 }
      },
      legend: {
        show: false
      },
      series: [
        {
          type: 'pie',
          radius: ['50%', '75%'],
          center: ['50%', '50%'],
          avoidLabelOverlap: false,
          itemStyle: {
            borderRadius: 6,
            borderColor: '#fff',
            borderWidth: 2
          },
          label: { show: false },
          emphasis: {
            label: { show: true, fontSize: 14, fontWeight: 'bold' }
          },
          data: [],
          color: ['#10b981', '#3b82f6', '#ef4444']
        }
      ]
    })
  }

  // 更新 PV 图表
  const updatePVChart = () => {
    if (!pvChart || !props.storageData) return

    const data = [
      { value: props.storageData.pvBound || 0, name: '已绑定' },
      { value: props.storageData.pvAvailable || 0, name: '可用' },
      { value: props.storageData.pvFailed || 0, name: '失败' }
    ].filter((item) => item.value > 0)

    pvChart.setOption({ series: [{ data }] })
  }

  // 初始化 PVC 图表
  const initPVCChart = () => {
    if (!pvcChartRef.value) return
    pvcChart = echarts.init(pvcChartRef.value)
    pvcChart.setOption({
      tooltip: {
        trigger: 'item',
        formatter: '{b}: {c} ({d}%)',
        backgroundColor: 'rgba(255,255,255,0.98)',
        borderColor: '#eaecef',
        borderWidth: 1,
        textStyle: { color: '#1f2937', fontSize: 12 }
      },
      legend: {
        show: false
      },
      series: [
        {
          type: 'pie',
          radius: ['50%', '75%'],
          center: ['50%', '50%'],
          avoidLabelOverlap: false,
          itemStyle: {
            borderRadius: 6,
            borderColor: '#fff',
            borderWidth: 2
          },
          label: { show: false },
          emphasis: {
            label: { show: true, fontSize: 14, fontWeight: 'bold' }
          },
          data: [],
          color: ['#10b981', '#f59e0b', '#ef4444']
        }
      ]
    })
  }

  // 更新 PVC 图表
  const updatePVCChart = () => {
    if (!pvcChart || !props.storageData) return

    const data = [
      { value: props.storageData.pvcBound || 0, name: '已绑定' },
      { value: props.storageData.pvcPending || 0, name: '待绑定' },
      { value: props.storageData.pvcLost || 0, name: '丢失' }
    ].filter((item) => item.value > 0)

    pvcChart.setOption({ series: [{ data }] })
  }

  const handleResize = () => {
    if (resizeTimer) clearTimeout(resizeTimer)
    resizeTimer = setTimeout(() => {
      pvChart?.resize()
      pvcChart?.resize()
    }, 300)
  }

  watch(
    () => props.storageData,
    () => {
      nextTick(() => {
        if (hasData.value) {
          if (!pvChart) initPVChart()
          if (!pvcChart) initPVCChart()
          updatePVChart()
          updatePVCChart()
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
    pvChart?.dispose()
    pvcChart?.dispose()
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

  .storage-monitor {
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
        flex-wrap: wrap;
        gap: 16px;

        .header-left {
          display: flex;
          align-items: center;
          gap: 14px;

          .header-icon {
            width: 44px;
            height: 44px;
            border-radius: $radius-sm;
            background: linear-gradient(135deg, #fef3c7, #fde68a);
            display: flex;
            align-items: center;
            justify-content: center;
            color: $warning;
            flex-shrink: 0;
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

          .health-score {
            margin-left: 6px;
            font-weight: 700;
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

          &.critical {
            background: rgba($danger, 0.1);
            color: #dc2626;
            .status-dot {
              background: $danger;
            }
          }
        }
      }

      .issues-alert {
        background: rgba($danger, 0.05);
        border: 1px solid rgba($danger, 0.2);
        border-radius: $radius-sm;
        padding: 14px 18px;
        margin-bottom: 20px;

        .alert-header {
          display: flex;
          align-items: center;
          gap: 10px;
          margin-bottom: 10px;
          font-weight: 600;
          font-size: 14px;
          color: $danger;

          svg {
            color: $danger;
          }
        }

        .issues-list {
          display: flex;
          flex-direction: column;
          gap: 6px;

          .issue-item {
            display: flex;
            align-items: center;
            gap: 8px;
            font-size: 13px;
            color: $gray-600;

            .issue-dot {
              width: 4px;
              height: 4px;
              border-radius: 50%;
              background: $danger;
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

            &.pv {
              background: linear-gradient(135deg, #e0e7ff, #c7d2fe);
              color: #4f46e5;
            }

            &.pvc {
              background: linear-gradient(135deg, #d1fae5, #a7f3d0);
              color: $success;
            }

            &.sc {
              background: linear-gradient(135deg, #fef3c7, #fde68a);
              color: $warning;
            }
          }

          .metric-info {
            min-width: 0;

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
        margin-bottom: 24px;
        padding-bottom: 16px;
        border-bottom: 1px solid $gray-200;
        flex-wrap: wrap;
        gap: 12px;

        .header-info {
          display: flex;
          align-items: center;
          gap: 14px;
          min-width: 0;

          .icon-badge {
            width: 46px;
            height: 46px;
            border-radius: $radius-sm;
            display: flex;
            align-items: center;
            justify-content: center;
            color: #fff;
            flex-shrink: 0;

            &.pv {
              background: linear-gradient(135deg, #4f46e5, #6366f1);
            }

            &.pvc {
              background: linear-gradient(135deg, $success, #34d399);
            }

            &.sc {
              background: linear-gradient(135deg, $warning, #fbbf24);
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
          flex-shrink: 0;

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

        .stats-badge {
          display: flex;
          gap: 12px;

          .badge-item {
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 2px;
            padding: 8px 16px;
            background: $gray-50;
            border-radius: $radius-xs;
            border: 1px solid $gray-200;

            .badge-label {
              font-size: 11px;
              color: $gray-500;
            }

            .badge-value {
              font-size: 16px;
              font-weight: 700;
              color: $gray-800;
            }
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

        &.success {
          background: rgba($success, 0.04);
          border-color: rgba($success, 0.15);
        }

        &.primary {
          background: rgba($primary, 0.04);
          border-color: rgba($primary, 0.15);
        }

        &.warning {
          background: rgba($warning, 0.06);
          border-color: rgba($warning, 0.25);
        }

        &.danger {
          background: rgba($danger, 0.06);
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
            font-size: 22px;
            font-weight: 700;
            color: $gray-800;

            &.success {
              color: #059669;
            }
            &.primary {
              color: $primary;
            }
            &.warning {
              color: #d97706;
            }
            &.danger {
              color: #dc2626;
            }
          }

          .unit {
            font-size: 12px;
            color: $gray-400;
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
            transition: width 0.4s ease;

            &.success {
              background: $success;
            }
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
                height: 8px;
                border-radius: 50%;

                &.bound {
                  background: $success;
                }
                &.available {
                  background: $primary;
                }
                &.failed,
                &.lost {
                  background: $danger;
                }
                &.pending {
                  background: $warning;
                }
              }
            }
          }
        }

        .chart-container {
          height: 200px;
          width: 100%;
        }
      }

      // StorageClass 列表样式
      .sc-list {
        display: flex;
        flex-direction: column;
        gap: 12px;

        .sc-item {
          display: flex;
          align-items: center;
          gap: 16px;
          padding: 16px;
          background: $gray-50;
          border-radius: $radius-sm;
          border: 1px solid $gray-200;
          transition: all 0.2s ease;

          &:hover {
            border-color: $gray-300;
            transform: translateX(4px);
          }

          .sc-rank {
            width: 32px;
            height: 32px;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 14px;
            font-weight: 700;
            background: $gray-200;
            color: $gray-600;
            flex-shrink: 0;

            &.gold {
              background: linear-gradient(135deg, #fef3c7, #fde68a);
              color: #b45309;
            }

            &.silver {
              background: linear-gradient(135deg, #f3f4f6, #e5e7eb);
              color: #4b5563;
            }

            &.bronze {
              background: linear-gradient(135deg, #fce7f3, #fbcfe8);
              color: #9d174d;
            }
          }

          .sc-info {
            flex: 1;
            min-width: 0;

            .sc-name {
              display: flex;
              align-items: center;
              gap: 8px;
              margin-bottom: 8px;

              svg {
                color: $warning;
                flex-shrink: 0;
              }

              span {
                font-size: 14px;
                font-weight: 600;
                color: $gray-800;
                white-space: nowrap;
                overflow: hidden;
                text-overflow: ellipsis;
              }
            }

            .sc-bar {
              height: 6px;
              background: $gray-200;
              border-radius: 3px;
              overflow: hidden;

              .sc-bar-fill {
                height: 100%;
                background: linear-gradient(90deg, $warning, #fbbf24);
                border-radius: 3px;
                transition: width 0.4s ease;
              }
            }
          }

          .sc-stats {
            display: flex;
            gap: 20px;
            flex-shrink: 0;

            .sc-stat {
              display: flex;
              flex-direction: column;
              align-items: flex-end;
              gap: 2px;

              .sc-stat-value {
                font-size: 16px;
                font-weight: 700;
                color: $gray-800;
              }

              .sc-stat-label {
                font-size: 11px;
                color: $gray-500;
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
