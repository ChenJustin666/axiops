<template>
  <div class="etcd-monitor" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="monitor-card">
        <!-- 卡片头部 -->
        <div class="card-header">
          <div class="header-left">
            <div class="icon-wrapper">
              <svg viewBox="0 0 24 24" width="22" height="22">
                <path
                  fill="currentColor"
                  d="M12,3C7.58,3 4,4.79 4,7C4,9.21 7.58,11 12,11C16.42,11 20,9.21 20,7C20,4.79 16.42,3 12,3M4,9V12C4,14.21 7.58,16 12,16C16.42,16 20,14.21 20,12V9C20,11.21 16.42,13 12,13C7.58,13 4,11.21 4,9M4,14V17C4,19.21 7.58,21 12,21C16.42,21 20,19.21 20,17V14C20,16.21 16.42,18 12,18C7.58,18 4,16.21 4,14Z"
                />
              </svg>
            </div>
            <div class="header-title">
              <h3>Etcd 监控</h3>
              <p>分布式键值存储 · 集群状态与性能</p>
            </div>
          </div>
          <div class="header-right">
            <ElTag :type="etcdStatus.type" effect="dark" round>
              <span class="status-dot"></span>
              {{ etcdStatus.text }}
            </ElTag>
          </div>
        </div>

        <!-- 核心指标卡片 - 统一大小 -->
        <div class="metrics-grid">
          <div class="metric-card">
            <div class="metric-icon db-icon">
              <svg viewBox="0 0 24 24" width="20" height="20">
                <path
                  fill="currentColor"
                  d="M6,2H18A2,2 0 0,1 20,4V20A2,2 0 0,1 18,22H6A2,2 0 0,1 4,20V4A2,2 0 0,1 6,2M6,4V8H18V4H6M6,10V14H18V10H6M6,16V20H18V16H6M8,6H10V6.5H8V6M8,12H10V12.5H8V12M8,18H10V18.5H8V18Z"
                />
              </svg>
            </div>
            <div class="metric-body">
              <span class="metric-label">数据库大小</span>
              <span class="metric-value">{{ formatBytes(etcdData?.dbSizeBytes) }}</span>
              <span class="metric-sub">使用中: {{ formatBytes(etcdData?.dbSizeInUse) }}</span>
            </div>
            <div v-if="etcdData?.dbSizeLimit && etcdData.dbSizeLimit > 0" class="metric-progress">
              <div class="progress-track">
                <div
                  class="progress-fill"
                  :style="{ width: Math.min(getDBUsagePercent(), 100) + '%' }"
                  :class="{ warning: getDBUsagePercent() > 80 }"
                ></div>
              </div>
              <span class="progress-text">{{ getDBUsagePercent().toFixed(1) }}%</span>
            </div>
          </div>

          <div
            class="metric-card"
            :class="{ 'status-warning': (etcdData?.commitLatency || 0) > 0.025 }"
          >
            <div class="metric-icon latency-icon">
              <svg viewBox="0 0 24 24" width="20" height="20">
                <path
                  fill="currentColor"
                  d="M12,20A8,8 0 0,0 20,12A8,8 0 0,0 12,4A8,8 0 0,0 4,12A8,8 0 0,0 12,20M12,2A10,10 0 0,1 22,12A10,10 0 0,1 12,22C6.47,22 2,17.5 2,12A10,10 0 0,1 12,2M12.5,7V12.25L17,14.92L16.25,16.15L11,13V7H12.5Z"
                />
              </svg>
            </div>
            <div class="metric-body">
              <span class="metric-label">提交延迟</span>
              <span class="metric-value">{{ formatLatency(etcdData?.commitLatency) }}</span>
              <span class="metric-badge" :class="getLatencyLevel(etcdData?.commitLatency, 0.025)">
                {{ getLatencyLevelText(etcdData?.commitLatency, 0.025) }}
              </span>
            </div>
          </div>

          <div class="metric-card">
            <div class="metric-icon cluster-icon">
              <svg viewBox="0 0 24 24" width="20" height="20">
                <path
                  fill="currentColor"
                  d="M12,5.5A3.5,3.5 0 0,1 15.5,9A3.5,3.5 0 0,1 12,12.5A3.5,3.5 0 0,1 8.5,9A3.5,3.5 0 0,1 12,5.5M5,8C5.56,8 6.08,8.15 6.53,8.42C6.38,9.85 6.8,11.27 7.66,12.38C7.16,13.34 6.16,14 5,14A3,3 0 0,1 2,11A3,3 0 0,1 5,8M19,8A3,3 0 0,1 22,11A3,3 0 0,1 19,14C17.84,14 16.84,13.34 16.34,12.38C17.2,11.27 17.62,9.85 17.47,8.42C17.92,8.15 18.44,8 19,8M5.5,18.25C5.5,16.18 8.41,14.5 12,14.5C15.59,14.5 18.5,16.18 18.5,18.25V20H5.5V18.25M0,20V18.5C0,17.11 1.89,15.94 4.45,15.6C3.86,16.28 3.5,17.22 3.5,18.25V20H0M24,20H20.5V18.25C20.5,17.22 20.14,16.28 19.55,15.6C22.11,15.94 24,17.11 24,18.5V20Z"
                />
              </svg>
            </div>
            <div class="metric-body">
              <span class="metric-label">集群成员</span>
              <span class="metric-value">{{ formatInteger(etcdData?.memberCount) }} 个</span>
              <span class="metric-badge" :class="etcdData?.hasLeader ? 'good' : 'critical'">
                {{ etcdData?.hasLeader ? '有 Leader' : '无 Leader' }}
              </span>
            </div>
          </div>

          <div class="metric-card">
            <div class="metric-icon keys-icon">
              <svg viewBox="0 0 24 24" width="20" height="20">
                <path
                  fill="currentColor"
                  d="M12.65,10C11.83,7.67 9.61,6 7,6A6,6 0 0,0 1,12A6,6 0 0,0 7,18C9.61,18 11.83,16.33 12.65,14H17V18H21V14H23V10H12.65M7,14A2,2 0 0,1 5,12A2,2 0 0,1 7,10A2,2 0 0,1 9,12A2,2 0 0,1 7,14Z"
                />
              </svg>
            </div>
            <div class="metric-body">
              <span class="metric-label">键总数</span>
              <span class="metric-value">{{ formatInteger(etcdData?.keyTotal) }}</span>
              <span class="metric-sub">Keys</span>
            </div>
          </div>
        </div>

        <!-- 集群状态条 -->
        <div class="cluster-bar">
          <div class="bar-item" :class="{ active: etcdData?.hasLeader }">
            <span class="bar-icon">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M12,2A10,10 0 0,1 22,12A10,10 0 0,1 12,22A10,10 0 0,1 2,12A10,10 0 0,1 12,2M12,4A8,8 0 0,0 4,12A8,8 0 0,0 12,20A8,8 0 0,0 20,12A8,8 0 0,0 12,4M11,16.5L6.5,12L7.91,10.59L11,13.67L16.59,8.09L18,9.5L11,16.5Z"
                />
              </svg>
            </span>
            <span class="bar-label">Leader 状态</span>
            <span class="bar-value">{{ etcdData?.hasLeader ? '正常' : '异常' }}</span>
          </div>
          <div class="bar-divider"></div>
          <div class="bar-item" :class="{ warning: (etcdData?.leaderChanges || 0) > 0 }">
            <span class="bar-icon">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M12,6V9L16,5L12,1V4A8,8 0 0,0 4,12C4,13.57 4.46,15.03 5.24,16.26L6.7,14.8C6.25,13.97 6,13 6,12A6,6 0 0,1 12,6M18.76,7.74L17.3,9.2C17.74,10.04 18,11 18,12A6,6 0 0,1 12,18V15L8,19L12,23V20A8,8 0 0,0 20,12C20,10.43 19.54,8.97 18.76,7.74Z"
                />
              </svg>
            </span>
            <span class="bar-label">Leader 变更</span>
            <span class="bar-value">{{ formatInteger(etcdData?.leaderChanges) }} 次/h</span>
          </div>
          <div class="bar-divider"></div>
          <div class="bar-item">
            <span class="bar-icon">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M12,4A4,4 0 0,1 16,8A4,4 0 0,1 12,12A4,4 0 0,1 8,8A4,4 0 0,1 12,4M12,14C16.42,14 20,15.79 20,18V20H4V18C4,15.79 7.58,14 12,14Z"
                />
              </svg>
            </span>
            <span class="bar-label">节点角色</span>
            <span class="bar-value">{{ etcdData?.isLearner ? 'Learner' : 'Voting Member' }}</span>
          </div>
        </div>

        <!-- 详细指标面板 - 移到图表上方 -->
        <div class="detail-metrics">
          <!-- 性能延迟 -->
          <div class="detail-group">
            <div class="group-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M12,20A8,8 0 0,0 20,12A8,8 0 0,0 12,4A8,8 0 0,0 4,12A8,8 0 0,0 12,20M12,2A10,10 0 0,1 22,12A10,10 0 0,1 12,22C6.47,22 2,17.5 2,12A10,10 0 0,1 12,2M12.5,7V12.25L17,14.92L16.25,16.15L11,13V7H12.5Z"
                />
              </svg>
              <span>性能延迟</span>
            </div>
            <div class="group-items">
              <div class="item">
                <span class="item-label">Commit</span>
                <span class="item-value">{{ formatLatency(etcdData?.commitLatency) }}</span>
              </div>
              <div class="item">
                <span class="item-label">WAL Fsync</span>
                <span class="item-value">{{ formatLatency(etcdData?.walFsyncLatency) }}</span>
              </div>
              <div class="item">
                <span class="item-label">Apply</span>
                <span class="item-value">{{ formatLatency(etcdData?.applyLatency) }}</span>
              </div>
              <div class="item">
                <span class="item-label">Snapshot</span>
                <span class="item-value">{{ formatLatency(etcdData?.snapshotLatency) }}</span>
              </div>
              <div class="item">
                <span class="item-label">Compact</span>
                <span class="item-value">{{ formatLatency(etcdData?.compactLatency) }}</span>
              </div>
              <div class="item">
                <span class="item-label">Peer RTT</span>
                <span class="item-value">{{ formatLatency(etcdData?.peerRTT) }}</span>
              </div>
            </div>
          </div>

          <!-- Proposal 状态 -->
          <div class="detail-group">
            <div class="group-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M19,3H14.82C14.4,1.84 13.3,1 12,1C10.7,1 9.6,1.84 9.18,3H5A2,2 0 0,0 3,5V19A2,2 0 0,0 5,21H19A2,2 0 0,0 21,19V5A2,2 0 0,0 19,3M12,3A1,1 0 0,1 13,4A1,1 0 0,1 12,5A1,1 0 0,1 11,4A1,1 0 0,1 12,3M7,7H17V5H19V19H5V5H7V7M7.5,13.5L9,12L11,14L15.5,9.5L17,11L11,17L7.5,13.5Z"
                />
              </svg>
              <span>Proposal 状态</span>
            </div>
            <div class="group-items">
              <div class="item">
                <span class="item-label">待处理</span>
                <span class="item-value">{{ formatInteger(etcdData?.proposalsPending) }}</span>
              </div>
              <div class="item">
                <span class="item-label">已应用</span>
                <span class="item-value">{{ formatInteger(etcdData?.proposalsApplied) }}</span>
              </div>
              <div class="item">
                <span class="item-label">已提交</span>
                <span class="item-value">{{ formatInteger(etcdData?.proposalsCommitted) }}</span>
              </div>
              <div class="item">
                <span class="item-label">失败</span>
                <span class="item-value" :class="{ danger: (etcdData?.proposalsFailed || 0) > 0 }">
                  {{ formatInteger(etcdData?.proposalsFailed) }}
                </span>
              </div>
            </div>
          </div>

          <!-- 操作速率 -->
          <div class="detail-group">
            <div class="group-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M13,2.05V4.05C17.39,4.59 20.5,8.58 19.96,12.97C19.5,16.61 16.64,19.5 13,19.93V21.93C18.5,21.38 22.5,16.5 21.95,11C21.5,6.25 17.73,2.5 13,2.03V2.05M5.67,19.74C7.18,21 9.04,21.79 11,22V20C9.58,19.82 8.23,19.25 7.1,18.37L5.67,19.74M7.1,5.74C8.22,4.84 9.57,4.26 11,4.06V2.06C9.05,2.25 7.19,3 5.67,4.26L7.1,5.74M5.69,7.1L4.26,5.67C3,7.19 2.25,9.04 2.05,11H4.05C4.24,9.58 4.8,8.23 5.69,7.1M4.06,13H2.06C2.26,14.96 3.03,16.81 4.27,18.33L5.69,16.9C4.81,15.77 4.24,14.42 4.06,13M10,16.5L16,12L10,7.5V16.5Z"
                />
              </svg>
              <span>操作速率</span>
            </div>
            <div class="group-items">
              <div class="item">
                <span class="item-label">GET</span>
                <span class="item-value">{{ formatNumber(etcdData?.getRate) }} /s</span>
              </div>
              <div class="item">
                <span class="item-label">PUT</span>
                <span class="item-value">{{ formatNumber(etcdData?.putRate) }} /s</span>
              </div>
              <div class="item">
                <span class="item-label">DELETE</span>
                <span class="item-value">{{ formatNumber(etcdData?.deleteRate) }} /s</span>
              </div>
            </div>
          </div>

          <!-- 网络指标 -->
          <div class="detail-group">
            <div class="group-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M12,21L15.6,16.2C14.6,15.45 13.35,15 12,15C10.65,15 9.4,15.45 8.4,16.2L12,21M12,3C7.95,3 4.21,4.34 1.2,6.6L3,9C5.5,7.12 8.62,6 12,6C15.38,6 18.5,7.12 21,9L22.8,6.6C19.79,4.34 16.05,3 12,3M12,9C9.3,9 6.81,9.89 4.8,11.4L6.6,13.8C8.1,12.67 9.97,12 12,12C14.03,12 15.9,12.67 17.4,13.8L19.2,11.4C17.19,9.89 14.7,9 12,9Z"
                />
              </svg>
              <span>网络指标</span>
            </div>
            <div class="group-items">
              <div class="item">
                <span class="item-label">发送速率</span>
                <span class="item-value">{{ formatBytes(etcdData?.networkSendRate) }}/s</span>
              </div>
              <div class="item">
                <span class="item-label">接收速率</span>
                <span class="item-value">{{ formatBytes(etcdData?.networkRecvRate) }}/s</span>
              </div>
            </div>
          </div>

          <!-- 异常统计 -->
          <div class="detail-group">
            <div class="group-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path fill="currentColor" d="M13,14H11V10H13M13,18H11V16H13M1,21H23L12,2L1,21Z" />
              </svg>
              <span>异常与慢操作</span>
            </div>
            <div class="group-items">
              <div class="item">
                <span class="item-label">慢 Apply</span>
                <span class="item-value" :class="{ danger: (etcdData?.slowApplies || 0) > 0 }">
                  {{ formatInteger(etcdData?.slowApplies) }}
                </span>
              </div>
              <div class="item">
                <span class="item-label">慢 Commit</span>
                <span class="item-value" :class="{ danger: (etcdData?.slowCommits || 0) > 0 }">
                  {{ formatInteger(etcdData?.slowCommits) }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- 图表区域 -->
        <div class="chart-section">
          <div class="chart-header">
            <div class="chart-title">
              <svg viewBox="0 0 24 24" width="18" height="18">
                <path fill="currentColor" d="M22,21H2V3H4V19H6V10H10V19H12V6H16V19H18V14H22V21Z" />
              </svg>
              <span>Etcd 性能趋势</span>
            </div>
            <div class="chart-legend">
              <span class="legend-item">
                <i class="legend-dot db"></i>
                DB 大小
              </span>
              <span class="legend-item">
                <i class="legend-dot latency"></i>
                提交延迟
              </span>
              <span class="legend-item">
                <i class="legend-dot failed"></i>
                失败提案
              </span>
            </div>
          </div>
          <div ref="etcdChartRef" class="chart"></div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" width="64" height="64">
            <path
              fill="currentColor"
              d="M12,3C7.58,3 4,4.79 4,7C4,9.21 7.58,11 12,11C16.42,11 20,9.21 20,7C20,4.79 16.42,3 12,3M4,9V12C4,14.21 7.58,16 12,16C16.42,16 20,14.21 20,12V9C20,11.21 16.42,13 12,13C7.58,13 4,11.21 4,9M4,14V17C4,19.21 7.58,21 12,21C16.42,21 20,19.21 20,17V14C20,16.21 16.42,18 12,18C7.58,18 4,16.21 4,14Z"
            />
          </svg>
        </div>
        <p class="empty-text">暂无 Etcd 监控数据</p>
        <p class="empty-sub">请检查 Etcd 服务连接状态</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, onUnmounted, nextTick, computed } from 'vue'
  import * as echarts from 'echarts'
  import type { EtcdMetrics } from '@/api/console/cluster-monitor'

  interface Props {
    etcdData?: EtcdMetrics
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    etcdData: undefined,
    loading: false
  })

  const etcdChartRef = ref<HTMLElement>()
  let etcdChart: echarts.ECharts | null = null
  let resizeTimer: NodeJS.Timeout | null = null

  const hasData = computed(() => {
    return !!(props.etcdData && Object.keys(props.etcdData).length > 0)
  })

  // ==================== 格式化函数 ====================
  const formatNumber = (value?: number): string => {
    if (value === undefined || value === null || isNaN(value)) return '0'
    return value.toFixed(2)
  }

  const formatInteger = (value?: number): string => {
    if (value === undefined || value === null || isNaN(value)) return '0'
    return Math.round(value).toString()
  }

  const formatLatency = (value?: number): string => {
    if (value === undefined || value === null || isNaN(value)) return '0 ms'
    const ms = value * 1000
    if (ms < 1) return `${(ms * 1000).toFixed(2)} µs`
    if (ms < 1000) return `${ms.toFixed(2)} ms`
    return `${(ms / 1000).toFixed(2)} s`
  }

  const formatBytes = (bytes?: number): string => {
    if (!bytes || bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
  }

  const getLatencyLevel = (latency?: number, threshold?: number) => {
    if (!latency || !threshold) return 'good'
    if (latency > threshold * 2) return 'critical'
    if (latency > threshold) return 'warning'
    return 'good'
  }

  const getLatencyLevelText = (latency?: number, threshold?: number) => {
    if (!latency || !threshold) return '正常'
    if (latency > threshold * 2) return '严重'
    if (latency > threshold) return '偏高'
    return '正常'
  }

  const getDBUsagePercent = (): number => {
    if (!props.etcdData?.dbSizeLimit || props.etcdData.dbSizeLimit === 0) return 0
    return (props.etcdData.dbSizeBytes / props.etcdData.dbSizeLimit) * 100
  }

  const etcdStatus = computed(() => {
    const data = props.etcdData
    if (!data) return { type: 'info' as const, text: '未知' }

    if (!data.hasLeader) return { type: 'danger' as const, text: '无 Leader' }
    if (data.proposalsFailed > 0 || data.slowCommits > 0 || data.slowApplies > 0) {
      return { type: 'danger' as const, text: '异常' }
    }
    if (data.commitLatency > 0.025 || data.walFsyncLatency > 0.01) {
      return { type: 'warning' as const, text: '需关注' }
    }
    return { type: 'success' as const, text: '正常' }
  })

  // ==================== 图表初始化 ====================
  const initChart = () => {
    if (!etcdChartRef.value) return
    etcdChart = echarts.init(etcdChartRef.value)

    const option: echarts.EChartsOption = {
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(255, 255, 255, 0.96)',
        borderColor: '#e2e8f0',
        borderWidth: 1,
        padding: [12, 16],
        textStyle: {
          color: '#1e293b',
          fontSize: 13
        },
        axisPointer: {
          type: 'cross',
          crossStyle: {
            color: '#94a3b8'
          }
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
        axisLine: {
          lineStyle: {
            color: '#e2e8f0'
          }
        },
        axisLabel: {
          color: '#64748b',
          fontSize: 11,
          formatter: (value: number) =>
            new Date(value).toLocaleTimeString('zh-CN', {
              hour: '2-digit',
              minute: '2-digit'
            })
        },
        splitLine: {
          show: true,
          lineStyle: {
            color: '#f1f5f9',
            type: 'dashed'
          }
        }
      },
      yAxis: [
        {
          type: 'value',
          name: 'DB 大小',
          position: 'left',
          nameTextStyle: {
            color: '#64748b',
            fontSize: 11,
            padding: [0, 0, 0, 40]
          },
          axisLine: {
            show: true,
            lineStyle: {
              color: '#e2e8f0'
            }
          },
          axisLabel: {
            color: '#64748b',
            fontSize: 11,
            formatter: (value: number) => formatBytes(value)
          },
          splitLine: {
            lineStyle: {
              color: '#f1f5f9',
              type: 'dashed'
            }
          }
        },
        {
          type: 'value',
          name: '延迟(ms) / 失败数',
          position: 'right',
          nameTextStyle: {
            color: '#64748b',
            fontSize: 11,
            padding: [0, 50, 0, 0]
          },
          axisLine: {
            show: true,
            lineStyle: {
              color: '#e2e8f0'
            }
          },
          axisLabel: {
            color: '#64748b',
            fontSize: 11
          },
          splitLine: {
            show: false
          }
        }
      ],
      series: []
    }

    etcdChart.setOption(option)
  }

  const updateChart = () => {
    if (!etcdChart || !props.etcdData?.trend) return

    const trend = props.etcdData.trend
    if (trend.length === 0) return

    const dbSizeData = trend.map((item) => [item.timestamp * 1000, item.dbSizeBytes || 0])
    const latencyData = trend.map((item) => [
      item.timestamp * 1000,
      (item.commitLatency || 0) * 1000
    ])
    const failedData = trend.map((item) => [item.timestamp * 1000, item.proposalsFailed || 0])

    etcdChart.setOption({
      series: [
        {
          name: 'DB 大小',
          type: 'line',
          yAxisIndex: 0,
          data: dbSizeData,
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2.5, color: '#3b82f6' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(59, 130, 246, 0.25)' },
              { offset: 1, color: 'rgba(59, 130, 246, 0.02)' }
            ])
          }
        },
        {
          name: '提交延迟',
          type: 'line',
          yAxisIndex: 1,
          data: latencyData,
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2.5, color: '#ef4444' }
        },
        {
          name: '失败提案',
          type: 'line',
          yAxisIndex: 1,
          data: failedData,
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2.5, color: '#f59e0b' }
        }
      ]
    })
  }

  const handleResize = () => {
    if (resizeTimer) clearTimeout(resizeTimer)
    resizeTimer = setTimeout(() => {
      etcdChart?.resize()
    }, 300)
  }

  watch(
    () => props.etcdData,
    () => {
      nextTick(() => {
        if (hasData.value) {
          if (!etcdChart) initChart()
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
    etcdChart?.dispose()
  })
</script>

<style lang="scss" scoped>
  .etcd-monitor {
    font-family:
      -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;

    .monitor-card {
      background: #ffffff;
      border-radius: 16px;
      box-shadow:
        0 1px 3px rgba(0, 0, 0, 0.04),
        0 4px 12px rgba(0, 0, 0, 0.06);
      overflow: hidden;
    }

    // ========== 头部 ==========
    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 20px 24px;
      background: linear-gradient(135deg, #ecfdf5 0%, #d1fae5 100%);
      border-bottom: 1px solid #a7f3d0;

      .header-left {
        display: flex;
        align-items: center;
        gap: 14px;

        .icon-wrapper {
          width: 42px;
          height: 42px;
          border-radius: 12px;
          display: flex;
          align-items: center;
          justify-content: center;
          background: linear-gradient(135deg, #10b981 0%, #059669 100%);
          color: white;
          box-shadow: 0 4px 12px rgba(16, 185, 129, 0.35);
        }

        .header-title {
          h3 {
            margin: 0;
            font-size: 17px;
            font-weight: 600;
            color: #064e3b;
            letter-spacing: -0.02em;
          }

          p {
            margin: 3px 0 0;
            font-size: 12px;
            color: #047857;
            opacity: 0.85;
          }
        }
      }

      .header-right {
        :deep(.el-tag) {
          font-weight: 500;
          padding: 6px 14px;
          border: none;

          .status-dot {
            display: inline-block;
            width: 6px;
            height: 6px;
            border-radius: 50%;
            background: currentColor;
            margin-right: 6px;
            animation: pulse 2s infinite;
          }
        }
      }
    }

    // ========== 核心指标网格 ==========
    .metrics-grid {
      display: grid;
      grid-template-columns: repeat(4, 1fr);
      gap: 16px;
      padding: 20px 24px;
      background: #fafbfc;

      @media (max-width: 1200px) {
        grid-template-columns: repeat(2, 1fr);
      }

      @media (max-width: 640px) {
        grid-template-columns: 1fr;
      }

      .metric-card {
        background: #ffffff;
        border-radius: 14px;
        padding: 18px;
        border: 1px solid #e5e7eb;
        display: flex;
        flex-direction: column;
        gap: 12px;
        transition: all 0.25s ease;
        min-height: 130px;

        &:hover {
          border-color: #d1d5db;
          box-shadow: 0 4px 16px rgba(0, 0, 0, 0.06);
          transform: translateY(-2px);
        }

        &.status-warning {
          border-color: #fbbf24;
          background: linear-gradient(180deg, #fffbeb 0%, #ffffff 100%);
        }

        .metric-icon {
          width: 40px;
          height: 40px;
          border-radius: 10px;
          display: flex;
          align-items: center;
          justify-content: center;
          flex-shrink: 0;

          &.db-icon {
            background: linear-gradient(135deg, #dbeafe 0%, #bfdbfe 100%);
            color: #2563eb;
          }

          &.latency-icon {
            background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
            color: #d97706;
          }

          &.cluster-icon {
            background: linear-gradient(135deg, #d1fae5 0%, #a7f3d0 100%);
            color: #059669;
          }

          &.keys-icon {
            background: linear-gradient(135deg, #ede9fe 0%, #ddd6fe 100%);
            color: #7c3aed;
          }
        }

        .metric-body {
          display: flex;
          flex-direction: column;
          gap: 4px;
          flex: 1;

          .metric-label {
            font-size: 12px;
            color: #6b7280;
            font-weight: 500;
          }

          .metric-value {
            font-size: 24px;
            font-weight: 700;
            color: #111827;
            letter-spacing: -0.02em;
            line-height: 1.2;
          }

          .metric-sub {
            font-size: 12px;
            color: #9ca3af;
          }

          .metric-badge {
            display: inline-flex;
            align-items: center;
            padding: 3px 10px;
            border-radius: 20px;
            font-size: 11px;
            font-weight: 600;
            width: fit-content;
            margin-top: 2px;

            &.good {
              background: #dcfce7;
              color: #15803d;
            }
            &.warning {
              background: #fef3c7;
              color: #b45309;
            }
            &.critical {
              background: #fee2e2;
              color: #dc2626;
            }
          }
        }

        .metric-progress {
          display: flex;
          align-items: center;
          gap: 10px;

          .progress-track {
            flex: 1;
            height: 6px;
            background: #e5e7eb;
            border-radius: 3px;
            overflow: hidden;

            .progress-fill {
              height: 100%;
              background: linear-gradient(90deg, #3b82f6 0%, #60a5fa 100%);
              border-radius: 3px;
              transition: width 0.5s ease;

              &.warning {
                background: linear-gradient(90deg, #f59e0b 0%, #fbbf24 100%);
              }
            }
          }

          .progress-text {
            font-size: 11px;
            color: #6b7280;
            font-weight: 500;
            min-width: 40px;
            text-align: right;
          }
        }
      }
    }

    // ========== 集群状态条 ==========
    .cluster-bar {
      display: flex;
      align-items: center;
      margin: 0 24px;
      padding: 14px 20px;
      background: #f8fafc;
      border-radius: 12px;
      border: 1px solid #e2e8f0;

      @media (max-width: 768px) {
        flex-direction: column;
        gap: 12px;

        .bar-divider {
          display: none;
        }
      }

      .bar-item {
        flex: 1;
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 0 16px;

        @media (max-width: 768px) {
          width: 100%;
          padding: 10px 16px;
          background: white;
          border-radius: 8px;
        }

        &.active {
          .bar-icon {
            color: #10b981;
          }
          .bar-value {
            color: #059669;
          }
        }

        &.warning {
          .bar-icon {
            color: #f59e0b;
          }
          .bar-value {
            color: #d97706;
          }
        }

        .bar-icon {
          color: #64748b;
          display: flex;
          align-items: center;
        }

        .bar-label {
          font-size: 12px;
          color: #64748b;
        }

        .bar-value {
          font-size: 13px;
          font-weight: 600;
          color: #1e293b;
          margin-left: auto;
        }
      }

      .bar-divider {
        width: 1px;
        height: 28px;
        background: #e2e8f0;
      }
    }

    // ========== 详细指标面板 ==========
    .detail-metrics {
      display: grid;
      grid-template-columns: repeat(5, 1fr);
      gap: 16px;
      padding: 20px 24px;
      margin-top: 16px;

      @media (max-width: 1400px) {
        grid-template-columns: repeat(3, 1fr);
      }

      @media (max-width: 900px) {
        grid-template-columns: repeat(2, 1fr);
      }

      @media (max-width: 600px) {
        grid-template-columns: 1fr;
      }

      .detail-group {
        background: #f8fafc;
        border-radius: 12px;
        padding: 14px 16px;
        border: 1px solid #e2e8f0;

        .group-header {
          display: flex;
          align-items: center;
          gap: 8px;
          padding-bottom: 10px;
          margin-bottom: 10px;
          border-bottom: 1px solid #e2e8f0;
          color: #475569;

          span {
            font-size: 12px;
            font-weight: 600;
            letter-spacing: 0.02em;
          }

          svg {
            opacity: 0.7;
          }
        }

        .group-items {
          display: flex;
          flex-direction: column;
          gap: 8px;

          .item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 6px 0;

            .item-label {
              font-size: 12px;
              color: #64748b;
            }

            .item-value {
              font-size: 13px;
              font-weight: 600;
              color: #1e293b;
              font-variant-numeric: tabular-nums;

              &.danger {
                color: #ef4444;
              }
            }
          }
        }
      }
    }

    // ========== 图表区域 ==========
    .chart-section {
      margin: 20px 24px 24px;
      background: #ffffff;
      border-radius: 14px;
      border: 1px solid #e5e7eb;

      .chart-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 16px;
        padding-bottom: 14px;
        border-bottom: 1px solid #f1f5f9;

        .chart-title {
          display: flex;
          align-items: center;
          gap: 10px;
          color: #1e293b;

          svg {
            color: #64748b;
          }

          span {
            font-size: 14px;
            font-weight: 600;
          }
        }

        .chart-legend {
          display: flex;
          gap: 20px;

          .legend-item {
            display: flex;
            align-items: center;
            gap: 6px;
            font-size: 12px;
            color: #64748b;

            .legend-dot {
              width: 10px;
              height: 10px;
              border-radius: 3px;

              &.db {
                background: #3b82f6;
              }
              &.latency {
                background: #ef4444;
              }
              &.failed {
                background: #f59e0b;
              }
            }
          }
        }
      }

      .chart {
        height: 300px;
        width: 100%;
      }
    }

    // ========== 空状态 ==========
    .empty-state {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      min-height: 400px;
      background: #ffffff;
      border-radius: 16px;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);

      .empty-icon {
        color: #d1d5db;
        margin-bottom: 16px;
      }

      .empty-text {
        font-size: 16px;
        color: #374151;
        font-weight: 500;
        margin: 0 0 6px;
      }

      .empty-sub {
        font-size: 13px;
        color: #9ca3af;
        margin: 0;
      }
    }
  }

  // ========== 动画 ==========
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
