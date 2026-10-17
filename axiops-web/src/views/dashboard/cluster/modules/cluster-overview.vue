<template>
  <div class="cluster-overview" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="overview-container">
        <!-- 总览卡片 -->
        <div class="overview-card">
          <div class="overview-header">
            <div class="header-left">
              <div class="header-icon">
                <svg viewBox="0 0 24 24" width="24" height="24">
                  <path
                    fill="currentColor"
                    d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"
                  />
                </svg>
              </div>
              <div>
                <h2>集群概览</h2>
                <p class="subtitle">实时资源使用与运行状态监控</p>
              </div>
            </div>
            <div class="health-badge" :class="clusterHealthStatus.status">
              <span class="status-dot"></span>
              <span class="status-text">{{ clusterHealthStatus.text }}</span>
            </div>
          </div>
        </div>

        <!-- 指标卡片网格 -->
        <div class="metrics-grid">
          <!-- CPU 卡片 -->
          <div class="metric-card">
            <div class="card-header">
              <div class="header-info">
                <div class="icon-badge cpu">
                  <svg viewBox="0 0 24 24" width="22" height="22">
                    <path
                      fill="currentColor"
                      d="M9 3V1H7v2H5v2H3v2h2v2H3v2h2v2H3v2h2v2h2v2h2v-2h2v2h2v-2h2v2h2v-2h2v-2h2v-2h-2v-2h2v-2h-2V9h2V7h-2V5h-2V3h-2v2h-2V3h-2v2H9V3zm0 4h6v2h2v6h-2v2H9v-2H7V9h2V7z"
                    />
                  </svg>
                </div>
                <div class="header-text">
                  <h3>CPU 使用率</h3>
                  <p>集群计算资源</p>
                </div>
              </div>
              <div
                class="status-tag"
                :class="getUsageLevel(overviewData?.resources?.cpu?.usagePercent)"
              >
                {{ getUsageLevelText(overviewData?.resources?.cpu?.usagePercent) }}
              </div>
            </div>

            <div class="metric-body">
              <div class="metric-value-row">
                <span class="main-value">{{
                  formatPercentage(overviewData?.resources?.cpu?.usagePercent)
                }}</span>
              </div>
              <div class="metric-progress">
                <div class="progress-bar">
                  <div
                    class="progress-fill"
                    :class="getUsageLevelClass(overviewData?.resources?.cpu?.usagePercent)"
                    :style="{
                      width: `${Math.min(overviewData?.resources?.cpu?.usagePercent || 0, 100)}%`
                    }"
                  ></div>
                </div>
              </div>
              <!-- 修复：垂直排列统计项 -->
              <div class="metric-stats-vertical">
                <div class="stat-row">
                  <span class="stat-label">使用</span>
                  <span class="stat-value">{{
                    formatCoresWithUnit(overviewData?.resources?.cpu?.usage)
                  }}</span>
                </div>
                <div class="stat-row">
                  <span class="stat-label">可分配</span>
                  <span class="stat-value">{{
                    formatCoresWithUnit(overviewData?.resources?.cpu?.allocatable)
                  }}</span>
                </div>
                <div class="stat-row">
                  <span class="stat-label">容量</span>
                  <span class="stat-value">{{
                    formatCoresWithUnit(overviewData?.resources?.cpu?.capacity)
                  }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- 内存卡片 -->
          <div class="metric-card">
            <div class="card-header">
              <div class="header-info">
                <div class="icon-badge memory">
                  <svg viewBox="0 0 24 24" width="22" height="22">
                    <path
                      fill="currentColor"
                      d="M2 7v10h20V7H2zm18 8H4V9h16v6zM6 11h2v2H6v-2zm3 0h2v2H9v-2zm3 0h2v2h-2v-2zm3 0h2v2h-2v-2z"
                    />
                  </svg>
                </div>
                <div class="header-text">
                  <h3>内存使用率</h3>
                  <p>集群内存资源</p>
                </div>
              </div>
              <div
                class="status-tag"
                :class="getUsageLevel(overviewData?.resources?.memory?.usagePercent)"
              >
                {{ getUsageLevelText(overviewData?.resources?.memory?.usagePercent) }}
              </div>
            </div>

            <div class="metric-body">
              <div class="metric-value-row">
                <span class="main-value">{{
                  formatPercentage(overviewData?.resources?.memory?.usagePercent)
                }}</span>
              </div>
              <div class="metric-progress">
                <div class="progress-bar">
                  <div
                    class="progress-fill memory"
                    :class="getUsageLevelClass(overviewData?.resources?.memory?.usagePercent)"
                    :style="{
                      width: `${Math.min(overviewData?.resources?.memory?.usagePercent || 0, 100)}%`
                    }"
                  ></div>
                </div>
              </div>
              <!-- 修复：垂直排列统计项 -->
              <div class="metric-stats-vertical">
                <div class="stat-row">
                  <span class="stat-label">使用</span>
                  <span class="stat-value">{{
                    formatBytes(overviewData?.resources?.memory?.usage)
                  }}</span>
                </div>
                <div class="stat-row">
                  <span class="stat-label">可分配</span>
                  <span class="stat-value">{{
                    formatBytes(overviewData?.resources?.memory?.allocatable)
                  }}</span>
                </div>
                <div class="stat-row">
                  <span class="stat-label">容量</span>
                  <span class="stat-value">{{
                    formatBytes(overviewData?.resources?.memory?.capacity)
                  }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- 节点卡片 -->
          <div class="metric-card">
            <div class="card-header">
              <div class="header-info">
                <div class="icon-badge node">
                  <svg viewBox="0 0 24 24" width="22" height="22">
                    <path
                      fill="currentColor"
                      d="M20 18c1.1 0 1.99-.9 1.99-2L22 6c0-1.1-.9-2-2-2H4c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2H0v2h24v-2h-4zM4 6h16v10H4V6z"
                    />
                  </svg>
                </div>
                <div class="header-text">
                  <h3>集群节点</h3>
                  <p>节点运行状态</p>
                </div>
              </div>
              <div class="status-tag" :class="nodeHealthStatus.class">
                {{ nodeHealthStatus.text }}
              </div>
            </div>

            <div class="metric-body">
              <div class="node-overview">
                <div class="node-main-stat">
                  <span class="main-number" :title="String(overviewData?.nodes?.total || 0)">{{
                    formatCompactNumber(overviewData?.nodes?.total)
                  }}</span>
                  <span class="main-label">总节点</span>
                </div>
                <div class="node-ring">
                  <svg viewBox="0 0 100 100">
                    <circle cx="50" cy="50" r="40" fill="none" stroke="#e5e7eb" stroke-width="8" />
                    <circle
                      cx="50"
                      cy="50"
                      r="40"
                      fill="none"
                      :stroke="nodeHealthStatus.class === 'success' ? '#10b981' : '#ef4444'"
                      stroke-width="8"
                      :stroke-dasharray="`${nodeReadyPercent * 2.51} 251`"
                      stroke-linecap="round"
                      transform="rotate(-90 50 50)"
                    />
                  </svg>
                  <span class="ring-percent">{{ nodeReadyPercent.toFixed(0) }}%</span>
                </div>
              </div>
              <!-- 修复：节点状态垂直排列，每个占一行 -->
              <div class="node-stats-vertical">
                <div class="node-stat-row ready">
                  <div class="stat-icon">
                    <svg viewBox="0 0 24 24" width="18" height="18">
                      <path
                        fill="currentColor"
                        d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"
                      />
                    </svg>
                  </div>
                  <span class="stat-label">就绪</span>
                  <span class="stat-value" :title="String(overviewData?.nodes?.ready || 0)">{{
                    formatCompactNumber(overviewData?.nodes?.ready)
                  }}</span>
                </div>
                <div class="node-stat-row not-ready">
                  <div class="stat-icon">
                    <svg viewBox="0 0 24 24" width="18" height="18">
                      <path
                        fill="currentColor"
                        d="M12 2C6.47 2 2 6.47 2 12s4.47 10 10 10 10-4.47 10-10S17.53 2 12 2zm5 13.59L15.59 17 12 13.41 8.41 17 7 15.59 10.59 12 7 8.41 8.41 7 12 10.59 15.59 7 17 8.41 13.41 12 17 15.59z"
                      />
                    </svg>
                  </div>
                  <span class="stat-label">未就绪</span>
                  <span class="stat-value" :title="String(overviewData?.nodes?.notReady || 0)">{{
                    formatCompactNumber(overviewData?.nodes?.notReady)
                  }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- Pod 卡片 -->
          <div class="metric-card">
            <div class="card-header">
              <div class="header-info">
                <div class="icon-badge pod">
                  <svg viewBox="0 0 24 24" width="22" height="22">
                    <path
                      fill="currentColor"
                      d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"
                    />
                  </svg>
                </div>
                <div class="header-text">
                  <h3>Pod 状态</h3>
                  <p>工作负载运行</p>
                </div>
              </div>
              <div class="status-tag" :class="podHealthStatus.class">
                {{ podHealthStatus.text }}
              </div>
            </div>

            <div class="metric-body">
              <div class="pod-overview">
                <div class="pod-main-stat">
                  <span
                    class="main-number"
                    :title="String(overviewData?.workloads?.pods?.running || 0)"
                    >{{ formatCompactNumber(overviewData?.workloads?.pods?.running) }}</span
                  >
                  <span class="main-label">运行中</span>
                </div>
                <div class="pod-total">
                  <span class="total-label">总计</span>
                  <span
                    class="total-value"
                    :title="String(overviewData?.workloads?.pods?.total || 0)"
                    >{{ formatCompactNumber(overviewData?.workloads?.pods?.total) }}</span
                  >
                </div>
              </div>
              <div class="pod-stats-grid">
                <div class="pod-stat-item running">
                  <div class="stat-icon">
                    <svg viewBox="0 0 24 24" width="14" height="14">
                      <path fill="currentColor" d="M8 5v14l11-7z" />
                    </svg>
                  </div>
                  <span
                    class="stat-num"
                    :title="String(overviewData?.workloads?.pods?.running || 0)"
                    >{{ formatCompactNumber(overviewData?.workloads?.pods?.running) }}</span
                  >
                  <span class="stat-text">运行</span>
                </div>
                <div class="pod-stat-item pending">
                  <div class="stat-icon">
                    <svg viewBox="0 0 24 24" width="14" height="14">
                      <path
                        fill="currentColor"
                        d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10 10-4.5 10-10S17.5 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67V7z"
                      />
                    </svg>
                  </div>
                  <span
                    class="stat-num"
                    :title="String(overviewData?.workloads?.pods?.pending || 0)"
                    >{{ formatCompactNumber(overviewData?.workloads?.pods?.pending) }}</span
                  >
                  <span class="stat-text">待调度</span>
                </div>
                <div class="pod-stat-item failed">
                  <div class="stat-icon">
                    <svg viewBox="0 0 24 24" width="14" height="14">
                      <path
                        fill="currentColor"
                        d="M12 2C6.47 2 2 6.47 2 12s4.47 10 10 10 10-4.47 10-10S17.53 2 12 2zm5 13.59L15.59 17 12 13.41 8.41 17 7 15.59 10.59 12 7 8.41 8.41 7 12 10.59 15.59 7 17 8.41 13.41 12 17 15.59z"
                      />
                    </svg>
                  </div>
                  <span
                    class="stat-num"
                    :title="String(overviewData?.workloads?.pods?.failed || 0)"
                    >{{ formatCompactNumber(overviewData?.workloads?.pods?.failed) }}</span
                  >
                  <span class="stat-text">失败</span>
                </div>
                <div class="pod-stat-item total">
                  <div class="stat-icon">
                    <svg viewBox="0 0 24 24" width="14" height="14">
                      <path
                        fill="currentColor"
                        d="M4 8h4V4H4v4zm6 12h4v-4h-4v4zm-6 0h4v-4H4v4zm0-6h4v-4H4v4zm6 0h4v-4h-4v4zm6-10v4h4V4h-4zm-6 4h4V4h-4v4zm6 6h4v-4h-4v4zm0 6h4v-4h-4v4z"
                      />
                    </svg>
                  </div>
                  <span
                    class="stat-num"
                    :title="String(overviewData?.workloads?.pods?.total || 0)"
                    >{{ formatCompactNumber(overviewData?.workloads?.pods?.total) }}</span
                  >
                  <span class="stat-text">总计</span>
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
        <h3>暂无概览数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import type { ClusterOverview } from '@/api/console/cluster-monitor'
  import { formatBytes, formatCores, formatPercentage } from '@/utils/format'

  interface Props {
    overviewData?: ClusterOverview
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    overviewData: undefined,
    loading: false
  })

  const hasData = computed(() => {
    return !!(props.overviewData && Object.keys(props.overviewData).length > 0)
  })

  /**
   * 格式化CPU核心数，带"核"单位
   * 例如：0.99 cores → 0.99 核，20 cores → 20 核
   */
  const formatCoresWithUnit = (value?: number): string => {
    if (value === undefined || value === null || isNaN(value)) return '0 核'
    // 如果小于10，保留2位小数；否则保留1位或整数
    if (value < 10) {
      return `${value.toFixed(2)} 核`
    } else if (value < 100) {
      return `${value.toFixed(1)} 核`
    }
    return `${Math.round(value)} 核`
  }

  /**
   * 格式化紧凑数字 - 处理大数字显示问题
   */
  const formatCompactNumber = (value?: number): string => {
    if (value === undefined || value === null || isNaN(value)) return '0'
    if (value < 1000) return String(value)
    if (value < 10000) return `${(value / 1000).toFixed(1)}K`
    if (value < 1000000) return `${Math.round(value / 1000)}K`
    if (value < 10000000) return `${(value / 1000000).toFixed(1)}M`
    return `${Math.round(value / 1000000)}M`
  }

  /**
   * 计算节点就绪百分比
   */
  const nodeReadyPercent = computed(() => {
    const total = props.overviewData?.nodes?.total || 0
    const ready = props.overviewData?.nodes?.ready || 0
    if (total === 0) return 100
    return (ready / total) * 100
  })

  /**
   * 集群健康状态计算
   */
  const clusterHealthStatus = computed(() => {
    if (!props.overviewData) return { status: 'unknown', text: '未知' }

    const hasNodeIssue = (props.overviewData.nodes?.notReady || 0) > 0
    const hasPodIssue = (props.overviewData.workloads?.pods?.failed || 0) > 0
    const cpuHigh = (props.overviewData.resources?.cpu?.usagePercent || 0) > 80
    const memoryHigh = (props.overviewData.resources?.memory?.usagePercent || 0) > 80

    if (hasNodeIssue || hasPodIssue) {
      return { status: 'critical', text: '异常' }
    }
    if (cpuHigh || memoryHigh) {
      return { status: 'warning', text: '需关注' }
    }
    return { status: 'healthy', text: '健康' }
  })

  /**
   * 节点健康状态
   */
  const nodeHealthStatus = computed(() => {
    const notReady = props.overviewData?.nodes?.notReady || 0
    if (notReady > 0) {
      return { class: 'danger', text: '异常' }
    }
    return { class: 'success', text: '健康' }
  })

  /**
   * Pod 健康状态
   */
  const podHealthStatus = computed(() => {
    const failed = props.overviewData?.workloads?.pods?.failed || 0
    const pending = props.overviewData?.workloads?.pods?.pending || 0
    if (failed > 0) {
      return { class: 'danger', text: '有失败' }
    }
    if (pending > 5) {
      return { class: 'warning', text: '待调度' }
    }
    return { class: 'success', text: '正常' }
  })

  const getUsageLevel = (percent?: number): string => {
    if (!percent) return 'success'
    if (percent < 60) return 'success'
    if (percent < 80) return 'warning'
    return 'danger'
  }

  const getUsageLevelClass = (percent?: number): string => {
    if (!percent) return 'level-normal'
    if (percent < 60) return 'level-normal'
    if (percent < 80) return 'level-warning'
    return 'level-danger'
  }

  const getUsageLevelText = (percent?: number): string => {
    if (!percent) return '正常'
    if (percent < 60) return '正常'
    if (percent < 80) return '偏高'
    return '过高'
  }
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

  .cluster-overview {
    background: $gray-50;
    font-family: var(--ax-font-sans);

    .overview-container {
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
            background: linear-gradient(135deg, #dbeafe, #bfdbfe);
            display: flex;
            align-items: center;
            justify-content: center;
            color: $primary;
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

          &.unknown {
            background: $gray-100;
            color: $gray-500;
            .status-dot {
              background: $gray-400;
            }
          }
        }
      }
    }

    .metrics-grid {
      display: grid;
      grid-template-columns: repeat(4, 1fr);
      gap: 20px;

      @media (max-width: 1400px) {
        grid-template-columns: repeat(2, 1fr);
      }

      @media (max-width: 768px) {
        grid-template-columns: 1fr;
      }
    }

    .metric-card {
      background: #fff;
      border-radius: $radius;
      padding: 20px;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
      transition: all 0.3s ease;
      min-width: 0;
      overflow: hidden;
      display: flex;
      flex-direction: column;

      &:hover {
        transform: translateY(-2px);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
      }

      .card-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        margin-bottom: 16px;
        padding-bottom: 14px;
        border-bottom: 1px solid $gray-200;
        gap: 12px;

        .header-info {
          display: flex;
          align-items: center;
          gap: 12px;
          min-width: 0;
          flex: 1;

          .icon-badge {
            width: 46px;
            height: 46px;
            border-radius: $radius-sm;
            display: flex;
            align-items: center;
            justify-content: center;
            color: #fff;
            flex-shrink: 0;

            &.cpu {
              background: linear-gradient(135deg, #667eea, #764ba2);
            }

            &.memory {
              background: linear-gradient(135deg, #f093fb, #f5576c);
            }

            &.node {
              background: linear-gradient(135deg, #4facfe, #00f2fe);
            }

            &.pod {
              background: linear-gradient(135deg, #43e97b, #38f9d7);
            }
          }

          .header-text {
            min-width: 0;

            h3 {
              margin: 0;
              font-size: 15px;
              font-weight: 600;
              color: $gray-800;
              white-space: nowrap;
              overflow: hidden;
              text-overflow: ellipsis;
            }

            p {
              margin: 2px 0 0;
              font-size: 12px;
              color: $gray-500;
              white-space: nowrap;
              overflow: hidden;
              text-overflow: ellipsis;
            }
          }
        }

        .status-tag {
          padding: 6px 14px;
          border-radius: 16px;
          font-size: 12px;
          font-weight: 600;
          flex-shrink: 0;
          white-space: nowrap;

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

      .metric-body {
        flex: 1;
        display: flex;
        flex-direction: column;

        .metric-value-row {
          margin-bottom: 12px;

          .main-value {
            font-size: 32px;
            font-weight: 700;
            color: $gray-800;
            line-height: 1.2;
          }
        }

        .metric-progress {
          margin-bottom: 14px;

          .progress-bar {
            height: 6px;
            background: $gray-200;
            border-radius: 3px;
            overflow: hidden;

            .progress-fill {
              height: 100%;
              border-radius: 3px;
              transition: width 0.6s ease;

              &.level-normal {
                background: linear-gradient(90deg, $success, #34d399);
              }

              &.level-warning {
                background: linear-gradient(90deg, $warning, #fbbf24);
              }

              &.level-danger {
                background: linear-gradient(90deg, $danger, #f87171);
              }
            }
          }
        }

        // 修复：CPU/内存卡片统计项垂直排列
        .metric-stats-vertical {
          display: flex;
          flex-direction: column;
          gap: 8px;

          .stat-row {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 10px 14px;
            background: $gray-50;
            border-radius: $radius-xs;
            border: 1px solid $gray-200;

            .stat-label {
              font-size: 13px;
              color: $gray-500;
              font-weight: 500;
            }

            .stat-value {
              font-size: 14px;
              font-weight: 600;
              color: $gray-700;
            }
          }
        }

        // 节点卡片样式
        .node-overview {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 14px;
          padding: 14px;
          background: $gray-50;
          border-radius: $radius-sm;

          .node-main-stat {
            display: flex;
            flex-direction: column;
            gap: 4px;

            .main-number {
              font-size: 36px;
              font-weight: 700;
              color: $gray-800;
              line-height: 1;
            }

            .main-label {
              font-size: 13px;
              color: $gray-500;
            }
          }

          .node-ring {
            position: relative;
            width: 70px;
            height: 70px;
            flex-shrink: 0;

            svg {
              width: 100%;
              height: 100%;
            }

            .ring-percent {
              position: absolute;
              top: 50%;
              left: 50%;
              transform: translate(-50%, -50%);
              font-size: 14px;
              font-weight: 700;
              color: $gray-700;
            }
          }
        }

        // 修复：节点状态垂直排列，每个占一行
        .node-stats-vertical {
          display: flex;
          flex-direction: column;
          gap: 8px;

          .node-stat-row {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 12px 14px;
            border-radius: $radius-xs;

            &.ready {
              background: rgba($success, 0.06);
              border: 1px solid rgba($success, 0.15);

              .stat-icon {
                background: rgba($success, 0.12);
                color: $success;
              }

              .stat-value {
                color: #059669;
              }
            }

            &.not-ready {
              background: rgba($danger, 0.06);
              border: 1px solid rgba($danger, 0.15);

              .stat-icon {
                background: rgba($danger, 0.12);
                color: $danger;
              }

              .stat-value {
                color: #dc2626;
              }
            }

            .stat-icon {
              width: 32px;
              height: 32px;
              border-radius: $radius-xs;
              display: flex;
              align-items: center;
              justify-content: center;
              flex-shrink: 0;
            }

            .stat-label {
              flex: 1;
              font-size: 13px;
              color: $gray-600;
              font-weight: 500;
            }

            .stat-value {
              font-size: 18px;
              font-weight: 700;
            }
          }
        }

        // Pod 卡片样式
        .pod-overview {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 14px;
          padding: 14px;
          background: linear-gradient(135deg, rgba($success, 0.06), rgba($success, 0.02));
          border-radius: $radius-sm;
          border: 1px solid rgba($success, 0.15);

          .pod-main-stat {
            display: flex;
            flex-direction: column;
            gap: 4px;

            .main-number {
              font-size: 36px;
              font-weight: 700;
              color: #059669;
              line-height: 1;
            }

            .main-label {
              font-size: 13px;
              color: $gray-500;
            }
          }

          .pod-total {
            display: flex;
            flex-direction: column;
            align-items: flex-end;
            gap: 2px;

            .total-label {
              font-size: 11px;
              color: $gray-400;
            }

            .total-value {
              font-size: 20px;
              font-weight: 600;
              color: $gray-600;
            }
          }
        }

        .pod-stats-grid {
          display: grid;
          grid-template-columns: repeat(2, 1fr);
          gap: 8px;

          .pod-stat-item {
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 4px;
            padding: 10px 8px;
            border-radius: $radius-xs;
            min-width: 0;

            &.running {
              background: rgba($success, 0.06);
              border: 1px solid rgba($success, 0.15);

              .stat-icon {
                color: $success;
              }
              .stat-num {
                color: #059669;
              }
            }

            &.pending {
              background: rgba($warning, 0.06);
              border: 1px solid rgba($warning, 0.15);

              .stat-icon {
                color: $warning;
              }
              .stat-num {
                color: #d97706;
              }
            }

            &.failed {
              background: rgba($danger, 0.06);
              border: 1px solid rgba($danger, 0.15);

              .stat-icon {
                color: $danger;
              }
              .stat-num {
                color: #dc2626;
              }
            }

            &.total {
              background: rgba($primary, 0.06);
              border: 1px solid rgba($primary, 0.15);

              .stat-icon {
                color: $primary;
              }
              .stat-num {
                color: $primary;
              }
            }

            .stat-icon {
              width: 22px;
              height: 22px;
              display: flex;
              align-items: center;
              justify-content: center;
            }

            .stat-num {
              font-size: 16px;
              font-weight: 700;
              line-height: 1;
            }

            .stat-text {
              font-size: 11px;
              color: $gray-500;
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
