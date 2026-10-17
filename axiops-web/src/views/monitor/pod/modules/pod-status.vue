<template>
  <div class="pod-status-monitor" v-loading="loading">
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
                    d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"
                  />
                </svg>
              </div>
              <div>
                <h2>Pod 状态概览</h2>
                <p class="subtitle">运行状态和容器健康监控</p>
              </div>
            </div>
            <div class="health-badge" :class="overallHealthStatus">
              <span class="status-dot"></span>
              <span class="status-text">{{ overallHealthText }}</span>
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
              <div class="metric-icon" :class="phaseIconClass">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2M12,4A8,8 0 0,1 20,12A8,8 0 0,1 12,20A8,8 0 0,1 4,12A8,8 0 0,1 12,4Z"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">运行阶段</span>
                <span class="metric-value">{{ statusData?.current?.phase || 'Unknown' }}</span>
              </div>
            </div>
            <div class="quick-metric">
              <div class="metric-icon" :class="statusData?.current?.ready ? 'ready' : 'not-ready'">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M21,7L9,19L3.5,13.5L4.91,12.09L9,16.17L19.59,5.59L21,7Z"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">就绪状态</span>
                <span class="metric-value">{{
                  statusData?.current?.ready ? '就绪' : '未就绪'
                }}</span>
              </div>
            </div>
            <div class="quick-metric">
              <div class="metric-icon" :class="totalRestarts > 0 ? 'warning' : 'success'">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M12,4C14.1,4 16.1,4.8 17.6,6.3C20.7,9.4 20.7,14.5 17.6,17.6C15.8,19.5 13.3,20.2 10.9,19.9L11.4,17.9C13.1,18.1 14.9,17.5 16.2,16.2C18.5,13.9 18.5,10.1 16.2,7.7C15.1,6.6 13.5,6 12,6V10.6L7,5.6L12,0.6V4M6.3,17.6C3.7,15 3.3,11 5.1,7.9L6.6,9.4C5.5,11.6 5.9,14.4 7.8,16.2C8.3,16.7 8.9,17.1 9.6,17.4L9,19.4C8,19 7.1,18.4 6.3,17.6Z"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">总重启次数</span>
                <span class="metric-value" :class="{ 'text-danger': totalRestarts > 0 }">{{
                  totalRestarts
                }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="content-grid">
          <!-- Pod 状态信息 -->
          <div class="monitor-card">
            <div class="card-header">
              <div class="header-info">
                <div class="icon-badge status">
                  <svg viewBox="0 0 24 24" width="22" height="22">
                    <path
                      fill="currentColor"
                      d="M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2M12,4A8,8 0 0,1 20,12A8,8 0 0,1 12,20A8,8 0 0,1 4,12A8,8 0 0,1 12,4M11,9H13V15H11V9M12,17A1,1 0 0,1 13,18A1,1 0 0,1 12,19A1,1 0 0,1 11,18A1,1 0 0,1 12,17Z"
                    />
                  </svg>
                </div>
                <div>
                  <h3>当前状态</h3>
                  <p>Pod 运行状态详情</p>
                </div>
              </div>
              <div class="status-tag" :class="getPhaseClass(statusData?.current?.phase)">
                {{ statusData?.current?.phase || 'Unknown' }}
              </div>
            </div>

            <div class="status-content">
              <div class="status-grid">
                <div class="status-item">
                  <span class="label">运行阶段</span>
                  <span class="value">
                    <span class="status-badge" :class="getPhaseClass(statusData?.current?.phase)">
                      {{ statusData?.current?.phase || 'Unknown' }}
                    </span>
                  </span>
                </div>
                <div class="status-item">
                  <span class="label">就绪状态</span>
                  <span class="value">
                    <span
                      class="status-badge"
                      :class="statusData?.current?.ready ? 'success' : 'danger'"
                    >
                      {{ statusData?.current?.ready ? '就绪' : '未就绪' }}
                    </span>
                  </span>
                </div>
                <div v-if="statusData?.current?.reason" class="status-item full">
                  <span class="label">原因</span>
                  <span class="value reason">{{ statusData.current.reason }}</span>
                </div>
                <div v-if="statusData?.current?.message" class="status-item full">
                  <span class="label">消息</span>
                  <span class="value message">{{ statusData.current.message }}</span>
                </div>
              </div>
            </div>

            <!-- 容器状态 -->
            <div v-if="statusData?.current?.containerStates?.length" class="containers-section">
              <div class="section-header">
                <h4>容器状态</h4>
                <span class="count-badge">{{ statusData.current.containerStates.length }}</span>
              </div>
              <div class="container-grid">
                <div
                  class="container-item"
                  v-for="container in statusData.current.containerStates"
                  :key="container.containerName"
                >
                  <div class="container-header">
                    <div class="container-name">
                      <svg viewBox="0 0 24 24" width="16" height="16">
                        <path
                          fill="currentColor"
                          d="M21,16.5C21,16.88 20.79,17.21 20.47,17.38L12.57,21.82C12.41,21.94 12.21,22 12,22C11.79,22 11.59,21.94 11.43,21.82L3.53,17.38C3.21,17.21 3,16.88 3,16.5V7.5C3,7.12 3.21,6.79 3.53,6.62L11.43,2.18C11.59,2.06 11.79,2 12,2C12.21,2 12.41,2.06 12.57,2.18L20.47,6.62C20.79,6.79 21,7.12 21,7.5V16.5Z"
                        />
                      </svg>
                      {{ container.containerName }}
                    </div>
                    <span class="state-badge" :class="container.ready ? 'success' : 'warning'">
                      {{ container.state }}
                    </span>
                  </div>
                  <div class="container-stats">
                    <div class="stat">
                      <span class="stat-label">重启</span>
                      <span
                        class="stat-value"
                        :class="{ 'text-danger': container.restartCount > 0 }"
                      >
                        {{ container.restartCount }}
                      </span>
                    </div>
                    <div v-if="container.reason" class="stat reason">
                      <span class="stat-label">原因</span>
                      <span class="stat-value">{{ container.reason }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 重启信息 -->
          <div class="monitor-card">
            <div class="card-header">
              <div class="header-info">
                <div class="icon-badge restart">
                  <svg viewBox="0 0 24 24" width="22" height="22">
                    <path
                      fill="currentColor"
                      d="M12,4C14.1,4 16.1,4.8 17.6,6.3C20.7,9.4 20.7,14.5 17.6,17.6C15.8,19.5 13.3,20.2 10.9,19.9L11.4,17.9C13.1,18.1 14.9,17.5 16.2,16.2C18.5,13.9 18.5,10.1 16.2,7.7C15.1,6.6 13.5,6 12,6V10.6L7,5.6L12,0.6V4M6.3,17.6C3.7,15 3.3,11 5.1,7.9L6.6,9.4C5.5,11.6 5.9,14.4 7.8,16.2C8.3,16.7 8.9,17.1 9.6,17.4L9,19.4C8,19 7.1,18.4 6.3,17.6Z"
                    />
                  </svg>
                </div>
                <div>
                  <h3>重启统计</h3>
                  <p>容器重启记录</p>
                </div>
              </div>
              <div class="status-tag" :class="totalRestarts > 0 ? 'warning' : 'success'">
                {{ totalRestarts > 0 ? '有重启' : '无重启' }}
              </div>
            </div>

            <div class="restart-summary">
              <div class="summary-card" :class="{ warning: totalRestarts > 0 }">
                <div class="summary-icon">
                  <svg viewBox="0 0 24 24" width="32" height="32">
                    <path
                      fill="currentColor"
                      d="M12,4C14.1,4 16.1,4.8 17.6,6.3C20.7,9.4 20.7,14.5 17.6,17.6C15.8,19.5 13.3,20.2 10.9,19.9L11.4,17.9C13.1,18.1 14.9,17.5 16.2,16.2C18.5,13.9 18.5,10.1 16.2,7.7C15.1,6.6 13.5,6 12,6V10.6L7,5.6L12,0.6V4M6.3,17.6C3.7,15 3.3,11 5.1,7.9L6.6,9.4C5.5,11.6 5.9,14.4 7.8,16.2C8.3,16.7 8.9,17.1 9.6,17.4L9,19.4C8,19 7.1,18.4 6.3,17.6Z"
                    />
                  </svg>
                </div>
                <div class="summary-content">
                  <span class="summary-label">总重启次数</span>
                  <span class="summary-value" :class="{ 'text-danger': totalRestarts > 0 }">{{
                    totalRestarts
                  }}</span>
                </div>
              </div>
            </div>

            <!-- 按容器统计 -->
            <div v-if="restartData?.byContainer?.length" class="restart-by-container">
              <div class="section-header">
                <h4>按容器统计</h4>
              </div>
              <div class="container-restart-list">
                <div
                  class="restart-item"
                  v-for="container in restartData.byContainer"
                  :key="container.containerName"
                >
                  <div class="item-left">
                    <svg viewBox="0 0 24 24" width="16" height="16">
                      <path
                        fill="currentColor"
                        d="M21,16.5C21,16.88 20.79,17.21 20.47,17.38L12.57,21.82C12.41,21.94 12.21,22 12,22C11.79,22 11.59,21.94 11.43,21.82L3.53,17.38C3.21,17.21 3,16.88 3,16.5V7.5C3,7.12 3.21,6.79 3.53,6.62L11.43,2.18C11.59,2.06 11.79,2 12,2C12.21,2 12.41,2.06 12.57,2.18L20.47,6.62C20.79,6.79 21,7.12 21,7.5V16.5Z"
                      />
                    </svg>
                    <span>{{ container.containerName }}</span>
                  </div>
                  <span
                    class="restart-count"
                    :class="container.restartCount > 0 ? 'warning' : 'success'"
                  >
                    {{ container.restartCount }} 次
                  </span>
                </div>
              </div>
            </div>

            <!-- 最近重启事件 -->
            <div v-if="restartData?.recentRestarts?.length" class="recent-restarts">
              <div class="section-header">
                <h4>最近重启事件</h4>
                <span class="count-badge warning">{{ restartData.recentRestarts.length }}</span>
              </div>
              <div class="restart-events">
                <div
                  class="restart-event"
                  v-for="(event, index) in restartData.recentRestarts.slice(0, 5)"
                  :key="index"
                >
                  <div class="event-time">
                    <svg viewBox="0 0 24 24" width="14" height="14">
                      <path
                        fill="currentColor"
                        d="M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2M12,4A8,8 0 0,1 20,12A8,8 0 0,1 12,20A8,8 0 0,1 4,12A8,8 0 0,1 12,4M12.5,7V12.25L17,14.92L16.25,16.15L11,13V7H12.5Z"
                      />
                    </svg>
                    {{ formatTime(event.timestamp) }}
                  </div>
                  <div class="event-detail">
                    <span class="event-container">{{ event.containerName }}</span>
                    <span class="event-reason">{{ event.reason }}</span>
                    <span class="exit-code">退出码: {{ event.exitCode }}</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- 无重启事件时显示 -->
            <div v-if="!restartData?.recentRestarts?.length" class="no-restarts">
              <svg viewBox="0 0 24 24" width="48" height="48">
                <path
                  fill="currentColor"
                  d="M12,2A10,10 0 0,1 22,12A10,10 0 0,1 12,22A10,10 0 0,1 2,12A10,10 0 0,1 12,2M12,4A8,8 0 0,0 4,12A8,8 0 0,0 12,20A8,8 0 0,0 20,12A8,8 0 0,0 12,4M11,16.5L6.5,12L7.91,10.59L11,13.67L16.59,8.09L18,9.5L11,16.5Z"
                />
              </svg>
              <span>暂无重启记录</span>
            </div>
          </div>
        </div>

        <!-- 状态历史时间线 -->
        <div v-if="statusData?.transitions?.length" class="monitor-card timeline-card">
          <div class="card-header">
            <div class="header-info">
              <div class="icon-badge timeline">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M12,20A8,8 0 0,0 20,12A8,8 0 0,0 12,4A8,8 0 0,0 4,12A8,8 0 0,0 12,20M12,2A10,10 0 0,1 22,12A10,10 0 0,1 12,22C6.47,22 2,17.5 2,12A10,10 0 0,1 12,2M12.5,7V12.25L17,14.92L16.25,16.15L11,13V7H12.5Z"
                  />
                </svg>
              </div>
              <div>
                <h3>状态变更历史</h3>
                <p>Pod 状态转换记录</p>
              </div>
            </div>
            <span class="count-badge">{{ statusData.transitions.length }}</span>
          </div>

          <div class="timeline">
            <div
              class="timeline-item"
              v-for="(transition, index) in statusData.transitions.slice(0, 10)"
              :key="index"
            >
              <div class="timeline-marker" :class="getPhaseClass(transition.toPhase)"></div>
              <div class="timeline-content">
                <div class="transition-info">
                  <span class="phase-badge" :class="getPhaseClass(transition.fromPhase)">
                    {{ transition.fromPhase }}
                  </span>
                  <svg viewBox="0 0 24 24" width="16" height="16" class="arrow">
                    <path
                      fill="currentColor"
                      d="M4,11V13H16L10.5,18.5L11.92,19.92L19.84,12L11.92,4.08L10.5,5.5L16,11H4Z"
                    />
                  </svg>
                  <span class="phase-badge" :class="getPhaseClass(transition.toPhase)">
                    {{ transition.toPhase }}
                  </span>
                  <span v-if="transition.reason" class="reason">({{ transition.reason }})</span>
                </div>
                <div class="transition-time">{{ formatTime(transition.timestamp) }}</div>
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
              d="M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2M12,4A8,8 0 0,1 20,12A8,8 0 0,1 12,20A8,8 0 0,1 4,12A8,8 0 0,1 12,4Z"
            />
          </svg>
        </div>
        <h3>暂无状态数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import type { PodStatusMetrics, RestartMetrics } from '@/api/console/pod-monitor'

  interface Props {
    statusData?: PodStatusMetrics
    restartData?: RestartMetrics
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    statusData: undefined,
    restartData: undefined,
    loading: false
  })

  // 数据状态
  const hasData = computed(() => {
    return !!(props.statusData || props.restartData)
  })

  // 总重启次数
  const totalRestarts = computed(() => {
    return props.restartData?.totalRestarts || 0
  })

  // 整体健康状态
  const overallHealthStatus = computed(() => {
    const phase = props.statusData?.current?.phase
    const ready = props.statusData?.current?.ready
    const restarts = totalRestarts.value

    if (phase === 'Failed' || restarts > 5) return 'critical'
    if (!ready || phase === 'Pending' || restarts > 0) return 'warning'
    if (phase === 'Running' && ready) return 'healthy'
    return 'unknown'
  })

  const overallHealthText = computed(() => {
    const status = overallHealthStatus.value
    if (status === 'healthy') return '健康'
    if (status === 'warning') return '需关注'
    if (status === 'critical') return '异常'
    return '未知'
  })

  // 运行阶段图标样式
  const phaseIconClass = computed(() => {
    const phase = props.statusData?.current?.phase
    if (phase === 'Running') return 'running'
    if (phase === 'Pending') return 'pending'
    if (phase === 'Failed') return 'failed'
    if (phase === 'Succeeded') return 'succeeded'
    return 'unknown'
  })

  // 健康问题列表
  const healthIssues = computed(() => {
    const issues: string[] = []

    if (!props.statusData?.current?.ready) {
      issues.push('Pod 未处于就绪状态')
    }

    if (props.statusData?.current?.phase === 'Pending') {
      issues.push('Pod 处于等待状态')
    }

    if (props.statusData?.current?.phase === 'Failed') {
      issues.push('Pod 运行失败')
    }

    if (totalRestarts.value > 0) {
      issues.push(`容器已重启 ${totalRestarts.value} 次`)
    }

    const containerStates = props.statusData?.current?.containerStates || []
    containerStates.forEach((container) => {
      if (!container.ready) {
        issues.push(`容器 ${container.containerName} 未就绪`)
      }
      if (container.restartCount > 3) {
        issues.push(`容器 ${container.containerName} 频繁重启`)
      }
    })

    return issues
  })

  // 获取阶段样式类
  const getPhaseClass = (phase?: string): string => {
    switch (phase) {
      case 'Running':
        return 'success'
      case 'Pending':
        return 'warning'
      case 'Failed':
        return 'danger'
      case 'Succeeded':
        return 'success'
      default:
        return 'info'
    }
  }

  // 格式化时间
  const formatTime = (timestamp: string): string => {
    if (!timestamp) return ''
    const date = new Date(timestamp)
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
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

  .pod-status-monitor {
    font-family: var(--ax-font-sans);

    .monitor-container {
      display: flex;
      flex-direction: column;
      gap: 24px;
    }

    .text-danger {
      color: $danger !important;
    }

    // 总览卡片
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

            &.running,
            &.ready,
            &.success {
              background: linear-gradient(135deg, #d1fae5, #a7f3d0);
              color: $success;
            }

            &.pending,
            &.warning {
              background: linear-gradient(135deg, #fef3c7, #fde68a);
              color: $warning;
            }

            &.failed,
            &.not-ready {
              background: linear-gradient(135deg, #fee2e2, #fecaca);
              color: $danger;
            }

            &.succeeded {
              background: linear-gradient(135deg, #dbeafe, #bfdbfe);
              color: $primary;
            }

            &.unknown {
              background: $gray-100;
              color: $gray-500;
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
          }
        }
      }
    }

    .content-grid {
      display: grid;
      grid-template-columns: repeat(2, 1fr);
      gap: 24px;

      @media (max-width: 1200px) {
        grid-template-columns: 1fr;
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

            &.status {
              background: linear-gradient(135deg, $primary, #2563eb);
            }

            &.restart {
              background: linear-gradient(135deg, $warning, #d97706);
            }

            &.timeline {
              background: linear-gradient(135deg, $gray-500, $gray-600);
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

          &.info {
            background: rgba($primary, 0.1);
            color: #2563eb;
          }
        }

        .count-badge {
          padding: 4px 12px;
          background: $gray-100;
          border-radius: 12px;
          font-size: 12px;
          font-weight: 600;
          color: $gray-600;
        }
      }

      .status-content {
        .status-grid {
          display: grid;
          grid-template-columns: repeat(2, 1fr);
          gap: 12px;

          .status-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 14px 16px;
            background: $gray-50;
            border-radius: $radius-xs;
            border: 1px solid $gray-200;

            &.full {
              grid-column: span 2;
              flex-direction: column;
              align-items: flex-start;
              gap: 8px;
            }

            .label {
              font-size: 12px;
              color: $gray-500;
              font-weight: 500;
            }

            .value {
              font-size: 14px;
              color: $gray-800;
              font-weight: 600;

              &.reason {
                color: $warning;
              }

              &.message {
                font-weight: 400;
                font-size: 13px;
                line-height: 1.5;
                color: $gray-600;
              }
            }

            .status-badge {
              padding: 4px 10px;
              border-radius: 12px;
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

              &.info {
                background: rgba($primary, 0.1);
                color: #2563eb;
              }
            }
          }
        }
      }

      .containers-section,
      .restart-by-container,
      .recent-restarts {
        margin-top: 24px;

        .section-header {
          display: flex;
          align-items: center;
          gap: 10px;
          margin-bottom: 16px;
          padding-bottom: 10px;
          border-bottom: 1px solid $gray-200;

          h4 {
            margin: 0;
            font-size: 14px;
            font-weight: 600;
            color: $gray-700;
          }

          .count-badge {
            padding: 2px 8px;
            background: $gray-100;
            border-radius: 10px;
            font-size: 11px;
            font-weight: 600;
            color: $gray-600;

            &.warning {
              background: rgba($warning, 0.1);
              color: #d97706;
            }
          }
        }
      }

      .container-grid {
        display: flex;
        flex-direction: column;
        gap: 12px;

        .container-item {
          padding: 16px;
          background: $gray-50;
          border-radius: $radius-xs;
          border: 1px solid $gray-200;
          border-left: 3px solid $primary;
          transition: all 0.2s ease;

          &:hover {
            border-color: $gray-300;
            box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
          }

          .container-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 12px;

            .container-name {
              display: flex;
              align-items: center;
              gap: 8px;
              font-weight: 600;
              font-size: 14px;
              color: $gray-800;

              svg {
                color: $gray-400;
              }
            }

            .state-badge {
              padding: 4px 10px;
              border-radius: 10px;
              font-size: 11px;
              font-weight: 600;

              &.success {
                background: rgba($success, 0.1);
                color: #059669;
              }

              &.warning {
                background: rgba($warning, 0.1);
                color: #d97706;
              }
            }
          }

          .container-stats {
            display: flex;
            gap: 20px;

            .stat {
              display: flex;
              align-items: center;
              gap: 6px;
              font-size: 12px;

              .stat-label {
                color: $gray-500;
              }

              .stat-value {
                font-weight: 600;
                color: $gray-700;
              }

              &.reason {
                .stat-value {
                  color: $warning;
                }
              }
            }
          }
        }
      }

      .restart-summary {
        margin-bottom: 20px;

        .summary-card {
          display: flex;
          align-items: center;
          gap: 20px;
          padding: 24px;
          background: linear-gradient(135deg, #d1fae5, #a7f3d0);
          border-radius: $radius-sm;

          &.warning {
            background: linear-gradient(135deg, #fee2e2, #fecaca);

            .summary-icon {
              color: $danger;
            }

            .summary-value {
              color: $danger;
            }
          }

          .summary-icon {
            color: $success;
          }

          .summary-content {
            .summary-label {
              font-size: 13px;
              color: $gray-600;
              display: block;
              margin-bottom: 4px;
            }

            .summary-value {
              font-size: 32px;
              font-weight: 700;
              color: $success;
            }
          }
        }
      }

      .container-restart-list {
        display: flex;
        flex-direction: column;
        gap: 10px;

        .restart-item {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 12px 16px;
          background: $gray-50;
          border-radius: $radius-xs;
          border: 1px solid $gray-200;

          .item-left {
            display: flex;
            align-items: center;
            gap: 10px;
            font-size: 13px;
            font-weight: 500;
            color: $gray-700;

            svg {
              color: $gray-400;
            }
          }

          .restart-count {
            padding: 4px 10px;
            border-radius: 10px;
            font-size: 11px;
            font-weight: 600;

            &.success {
              background: rgba($success, 0.1);
              color: #059669;
            }

            &.warning {
              background: rgba($warning, 0.1);
              color: #d97706;
            }
          }
        }
      }

      .restart-events {
        display: flex;
        flex-direction: column;
        gap: 12px;

        .restart-event {
          padding: 14px 16px;
          background: linear-gradient(135deg, #fef2f2, #fff);
          border-radius: $radius-xs;
          border-left: 3px solid $danger;

          .event-time {
            display: flex;
            align-items: center;
            gap: 6px;
            font-size: 11px;
            color: $gray-500;
            margin-bottom: 8px;
          }

          .event-detail {
            display: flex;
            align-items: center;
            gap: 12px;
            flex-wrap: wrap;
            font-size: 13px;

            .event-container {
              font-weight: 600;
              color: $gray-800;
            }

            .event-reason {
              color: $gray-600;
            }

            .exit-code {
              padding: 2px 8px;
              background: rgba($danger, 0.1);
              border-radius: 8px;
              font-size: 11px;
              font-weight: 600;
              color: $danger;
            }
          }
        }
      }

      .no-restarts {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 40px 20px;
        color: $success;

        svg {
          margin-bottom: 12px;
        }

        span {
          font-size: 14px;
          color: $gray-500;
        }
      }
    }

    .timeline-card {
      .timeline {
        display: flex;
        flex-direction: column;
        gap: 0;

        .timeline-item {
          display: flex;
          gap: 16px;
          padding-bottom: 20px;
          position: relative;

          &:not(:last-child)::before {
            content: '';
            position: absolute;
            left: 7px;
            top: 20px;
            bottom: 0;
            width: 2px;
            background: $gray-200;
          }

          .timeline-marker {
            width: 16px;
            height: 16px;
            border-radius: 50%;
            flex-shrink: 0;
            margin-top: 2px;

            &.success {
              background: $success;
            }

            &.warning {
              background: $warning;
            }

            &.danger {
              background: $danger;
            }

            &.info {
              background: $primary;
            }
          }

          .timeline-content {
            flex: 1;

            .transition-info {
              display: flex;
              align-items: center;
              gap: 10px;
              flex-wrap: wrap;
              margin-bottom: 6px;

              .phase-badge {
                padding: 4px 10px;
                border-radius: 10px;
                font-size: 11px;
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

                &.info {
                  background: rgba($primary, 0.1);
                  color: #2563eb;
                }
              }

              .arrow {
                color: $gray-400;
              }

              .reason {
                font-size: 12px;
                color: $gray-500;
              }
            }

            .transition-time {
              font-size: 11px;
              color: $gray-400;
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

  @media (max-width: 768px) {
    .pod-status-monitor {
      .overview-card {
        padding: 16px;

        .overview-header {
          flex-direction: column;
          align-items: flex-start;
          gap: 16px;
        }
      }

      .monitor-card {
        padding: 16px;

        .card-header {
          flex-direction: column;
          align-items: flex-start;
          gap: 12px;
        }

        .status-content .status-grid {
          grid-template-columns: 1fr;

          .status-item.full {
            grid-column: span 1;
          }
        }
      }
    }
  }
</style>
