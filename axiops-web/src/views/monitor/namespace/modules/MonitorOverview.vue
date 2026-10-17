<template>
  <div class="namespace-monitor-overview" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="monitor-container">
        <!-- 概览头部 -->
        <div class="overview-header">
          <div class="header-left">
            <div class="header-icon">
              <svg viewBox="0 0 24 24" width="24" height="24">
                <path
                  fill="currentColor"
                  d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14h-2v-4h2v4zm0-6h-2V7h2v4z"
                />
              </svg>
            </div>
            <div>
              <h2>命名空间概览</h2>
              <p class="subtitle">资源使用与工作负载状态监控</p>
            </div>
          </div>
          <div class="health-badge" :class="overallHealthStatus">
            <span class="status-dot"></span>
            <span class="status-text">{{ overallHealthText }}</span>
          </div>
        </div>

        <!-- 资源指标卡片 -->
        <div class="metrics-grid">
          <!-- CPU 卡片 -->
          <div
            class="metric-box"
            :class="getBoxClass(overviewData?.resources?.cpu?.current?.usagePercent)"
          >
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M17,17H7V7H17M21,11V9H19V7C19,5.89 18.1,5 17,5H15V3H13V5H11V3H9V5H7C5.89,5 5,5.89 5,7V9H3V11H5V13H3V15H5V17C5,18.1 5.9,19 7,19H9V21H11V19H13V21H15V19H17C18.1,19 19,18.1 19,17V15H21V13H19V11M15,15H9V9H15V15Z"
                />
              </svg>
              <span>CPU 使用率</span>
            </div>
            <div class="box-value">
              <span class="value">{{
                formatPercentage(overviewData?.resources?.cpu?.current?.usagePercent)
              }}</span>
            </div>
            <div class="box-detail">
              {{ formatCores(overviewData?.resources?.cpu?.current?.usageCores) }} /
              {{ formatCores(overviewData?.resources?.cpu?.limits) }}
            </div>
            <div class="box-progress">
              <div
                class="progress-fill"
                :class="getProgressClass(overviewData?.resources?.cpu?.current?.usagePercent)"
                :style="{ width: `${overviewData?.resources?.cpu?.current?.usagePercent || 0}%` }"
              ></div>
            </div>
            <div
              class="box-badge"
              :class="getUsageLevel(overviewData?.resources?.cpu?.current?.usagePercent)"
            >
              {{ getUsageLevelText(overviewData?.resources?.cpu?.current?.usagePercent) }}
            </div>
          </div>

          <!-- 内存卡片 -->
          <div
            class="metric-box"
            :class="getBoxClass(overviewData?.resources?.memory?.current?.usagePercent)"
          >
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M17,9H7V7H17M17,13H7V11H17M14,17H7V15H14M12,3A1,1 0 0,1 13,4A1,1 0 0,1 12,5A1,1 0 0,1 11,4A1,1 0 0,1 12,3M19,3H14.82C14.4,1.84 13.3,1 12,1C10.7,1 9.6,1.84 9.18,3H5A2,2 0 0,0 3,5V19A2,2 0 0,0 5,21H19A2,2 0 0,0 21,19V5A2,2 0 0,0 19,3Z"
                />
              </svg>
              <span>内存使用</span>
            </div>
            <div class="box-value">
              <span class="value">{{
                formatPercentage(overviewData?.resources?.memory?.current?.usagePercent)
              }}</span>
            </div>
            <div class="box-detail">
              {{ formatBytes(overviewData?.resources?.memory?.current?.workingSetBytes) }} /
              {{ formatBytes(overviewData?.resources?.memory?.limits) }}
            </div>
            <div class="box-progress">
              <div
                class="progress-fill memory"
                :class="getProgressClass(overviewData?.resources?.memory?.current?.usagePercent)"
                :style="{
                  width: `${overviewData?.resources?.memory?.current?.usagePercent || 0}%`
                }"
              ></div>
            </div>
            <div
              class="box-badge"
              :class="getUsageLevel(overviewData?.resources?.memory?.current?.usagePercent)"
            >
              {{ getUsageLevelText(overviewData?.resources?.memory?.current?.usagePercent) }}
            </div>
          </div>

          <!-- 磁盘卡片 -->
          <div class="metric-box" :class="getBoxClass(overviewData?.resources?.disk?.usagePercent)">
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M6,2H18A2,2 0 0,1 20,4V20A2,2 0 0,1 18,22H6A2,2 0 0,1 4,20V4A2,2 0 0,1 6,2M12,4A6,6 0 0,0 6,10C6,13.31 8.69,16 12.1,16L11.22,13.77C10.23,13.4 9.5,12.47 9.5,11.37C9.5,9.93 10.67,8.77 12.11,8.77C13.55,8.77 14.72,9.93 14.72,11.37C14.72,12.47 13.99,13.4 13,13.77L12.12,16C15.41,16 18.1,13.31 18.1,10A6,6 0 0,0 12,4M12,9.5A1.5,1.5 0 0,0 10.5,11A1.5,1.5 0 0,0 12,12.5A1.5,1.5 0 0,0 13.5,11A1.5,1.5 0 0,0 12,9.5Z"
                />
              </svg>
              <span>磁盘使用</span>
            </div>
            <div class="box-value">
              <span class="value">{{
                formatPercentage(overviewData?.resources?.disk?.usagePercent || 0)
              }}</span>
            </div>
            <div class="box-detail">
              {{ formatBytes(overviewData?.resources?.disk?.used || 0) }} /
              {{ formatBytes(overviewData?.resources?.disk?.total || 0) }}
            </div>
            <div class="box-progress">
              <div
                class="progress-fill disk"
                :class="getProgressClass(overviewData?.resources?.disk?.usagePercent)"
                :style="{ width: `${overviewData?.resources?.disk?.usagePercent || 0}%` }"
              ></div>
            </div>
            <div
              class="box-badge"
              :class="getUsageLevel(overviewData?.resources?.disk?.usagePercent)"
            >
              {{ getUsageLevelText(overviewData?.resources?.disk?.usagePercent) }}
            </div>
          </div>

          <!-- 网络卡片 -->
          <div class="metric-box network">
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M17,3A2,2 0 0,1 19,5V15A2,2 0 0,1 17,17H13V19H14A1,1 0 0,1 15,20H22V22H15A1,1 0 0,1 14,23H10A1,1 0 0,1 9,22H2V20H9A1,1 0 0,1 10,19H11V17H7C5.89,17 5,16.1 5,15V5A2,2 0 0,1 7,3H17M17,5H7V15H17V5Z"
                />
              </svg>
              <span>网络流量</span>
            </div>
            <div class="network-stats">
              <div class="stat-row ingress">
                <svg viewBox="0 0 24 24" width="14" height="14">
                  <path
                    fill="currentColor"
                    d="M7.41,8.58L12,13.17L16.59,8.58L18,10L12,16L6,10L7.41,8.58Z"
                  />
                </svg>
                <span class="stat-label">接收</span>
                <span class="stat-value">{{
                  formatBitRate(overviewData?.network?.current?.receiveBytesPerSec || 0)
                }}</span>
              </div>
              <div class="stat-row egress">
                <svg viewBox="0 0 24 24" width="14" height="14">
                  <path
                    fill="currentColor"
                    d="M7.41,15.41L12,10.83L16.59,15.41L18,14L12,8L6,14L7.41,15.41Z"
                  />
                </svg>
                <span class="stat-label">发送</span>
                <span class="stat-value">{{
                  formatBitRate(overviewData?.network?.current?.transmitBytesPerSec || 0)
                }}</span>
              </div>
            </div>
          </div>

          <!-- Pod 状态卡片 -->
          <div class="metric-box pod">
            <div class="box-header">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M21,16.5C21,16.88 20.79,17.21 20.47,17.38L12.57,21.82C12.41,21.94 12.21,22 12,22C11.79,22 11.59,21.94 11.43,21.82L3.53,17.38C3.21,17.21 3,16.88 3,16.5V7.5C3,7.12 3.21,6.79 3.53,6.62L11.43,2.18C11.59,2.06 11.79,2 12,2C12.21,2 12.41,2.06 12.57,2.18L20.47,6.62C20.79,6.79 21,7.12 21,7.5V16.5M12,4.15L6.04,7.5L12,10.85L17.96,7.5L12,4.15Z"
                />
              </svg>
              <span>Pod 状态</span>
            </div>
            <div class="pod-stats">
              <div class="stat-item">
                <span class="stat-count total">{{
                  overviewData?.workloads?.pods?.total || 0
                }}</span>
                <span class="stat-text">总数</span>
              </div>
              <div class="stat-item">
                <span class="stat-count running">{{
                  overviewData?.workloads?.pods?.running || 0
                }}</span>
                <span class="stat-text">运行中</span>
              </div>
            </div>
            <div
              v-if="overviewData?.workloads?.pods?.failed && overviewData.workloads.pods.failed > 0"
              class="box-badge danger"
            >
              异常
            </div>
            <div v-else class="box-badge normal"> 健康 </div>
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
  import type { NamespaceMetrics } from '@/api/console/namespace-monitor'
  import { formatBytes, formatBitRate, formatPercentage } from '@/utils/format'

  // ==================== 类型定义 ====================

  interface Props {
    overviewData?: NamespaceMetrics
    loading?: boolean
  }

  // ==================== 组件属性 ====================

  const props = withDefaults(defineProps<Props>(), {
    overviewData: undefined,
    loading: false
  })

  // ==================== 计算属性 ====================

  /**
   * 判断是否有数据
   */
  const hasData = computed(() => {
    return !!(props.overviewData && Object.keys(props.overviewData).length > 0)
  })

  /**
   * 计算整体健康状态
   */
  const overallHealthStatus = computed(() => {
    if (!props.overviewData) return 'unknown'

    const cpuPercent = props.overviewData.resources?.cpu?.current?.usagePercent || 0
    const memoryPercent = props.overviewData.resources?.memory?.current?.usagePercent || 0
    const diskPercent = props.overviewData.resources?.disk?.usagePercent || 0
    const failedPods = props.overviewData.workloads?.pods?.failed || 0

    if (cpuPercent > 90 || memoryPercent > 90 || diskPercent > 90 || failedPods > 0) {
      return 'critical'
    }
    if (cpuPercent > 70 || memoryPercent > 70 || diskPercent > 70) {
      return 'warning'
    }
    return 'healthy'
  })

  /**
   * 获取健康状态文本
   */
  const overallHealthText = computed(() => {
    const statusMap: Record<string, string> = {
      healthy: '运行正常',
      warning: '需要关注',
      critical: '存在异常',
      unknown: '状态未知'
    }
    return statusMap[overallHealthStatus.value] || '状态未知'
  })

  // ==================== 工具函数 ====================

  /**
   * 格式化 CPU 核心数
   */
  const formatCores = (cores?: number): string => {
    if (!cores || cores === 0) return '0'
    if (cores < 1) return `${(cores * 1000).toFixed(0)}m`
    return `${cores.toFixed(2)} cores`
  }

  /**
   * 获取使用率级别
   */
  const getUsageLevel = (percent?: number): string => {
    if (!percent) return 'normal'
    if (percent < 50) return 'normal'
    if (percent < 80) return 'warning'
    return 'danger'
  }

  /**
   * 获取使用率级别文本
   */
  const getUsageLevelText = (percent?: number): string => {
    if (!percent) return '正常'
    if (percent < 50) return '正常'
    if (percent < 80) return '偏高'
    return '过高'
  }

  /**
   * 获取进度条样式类
   */
  const getProgressClass = (percent?: number): string => {
    if (!percent) return ''
    if (percent < 50) return 'success'
    if (percent < 80) return 'warning'
    return 'danger'
  }

  /**
   * 获取卡片整体样式类
   */
  const getBoxClass = (percent?: number): string => {
    if (!percent) return ''
    if (percent >= 80) return 'danger'
    if (percent >= 50) return 'warning'
    return ''
  }
</script>

<style lang="scss" scoped>
  // ==================== 颜色变量 ====================
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

  // ==================== 主容器样式 ====================
  .namespace-monitor-overview {
    background: $gray-50;
    font-family: var(--ax-font-sans);
  }

  .monitor-container {
    max-width: 1800px;
    margin: 0 auto;
    background: #fff;
    border-radius: $radius;
    padding: 24px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  }

  // ==================== 头部样式 ====================
  .overview-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
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

  // ==================== 指标卡片网格 ====================
  .metrics-grid {
    display: grid;
    grid-template-columns: repeat(5, 1fr);
    gap: 14px;

    @media (max-width: 1400px) {
      grid-template-columns: repeat(3, 1fr);
    }

    @media (max-width: 900px) {
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
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
    }

    &.warning {
      background: rgba($warning, 0.04);
      border-color: rgba($warning, 0.2);
    }

    &.danger {
      background: rgba($danger, 0.04);
      border-color: rgba($danger, 0.2);
    }

    .box-header {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 12px;
      color: $gray-500;

      span {
        font-size: 12px;
        font-weight: 500;
      }
    }

    .box-value {
      margin-bottom: 6px;

      .value {
        font-size: 24px;
        font-weight: 700;
        color: $gray-800;
      }
    }

    .box-detail {
      font-size: 11px;
      color: $gray-400;
      margin-bottom: 12px;
    }

    .box-progress {
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

    .box-badge {
      position: absolute;
      top: 12px;
      right: 12px;
      padding: 4px 10px;
      border-radius: 12px;
      font-size: 11px;
      font-weight: 600;

      &.normal {
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

  // ==================== 网络卡片样式 ====================
  .metric-box.network {
    .network-stats {
      display: flex;
      flex-direction: column;
      gap: 8px;
      margin-top: 8px;

      .stat-row {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 8px 12px;
        border-radius: $radius-xs;
        font-size: 12px;

        &.ingress {
          background: linear-gradient(135deg, #dbeafe, #eff6ff);
          color: #1e40af;

          svg {
            color: $primary;
          }
        }

        &.egress {
          background: linear-gradient(135deg, #fef3c7, #fffbeb);
          color: #92400e;

          svg {
            color: $warning;
          }
        }

        .stat-label {
          font-weight: 500;
        }

        .stat-value {
          margin-left: auto;
          font-weight: 700;
        }
      }
    }
  }

  // ==================== Pod 卡片样式 ====================
  .metric-box.pod {
    .pod-stats {
      display: grid;
      grid-template-columns: repeat(2, 1fr);
      gap: 8px;
      margin-top: 8px;

      .stat-item {
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 10px;
        background: #fff;
        border-radius: $radius-xs;
        border: 1px solid $gray-200;

        .stat-count {
          font-size: 20px;
          font-weight: 700;
          margin-bottom: 4px;

          &.total {
            color: $primary;
          }

          &.running {
            color: $success;
          }
        }

        .stat-text {
          font-size: 11px;
          color: $gray-500;
          font-weight: 500;
        }
      }
    }
  }

  // ==================== 空状态样式 ====================
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 200px;
    background: #fff;
    border-radius: $radius;
    padding: 40px;

    .empty-icon {
      color: $gray-300;
      margin-bottom: 16px;
    }

    h3 {
      margin: 0 0 8px;
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
