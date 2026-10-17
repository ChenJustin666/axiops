<template>
  <div class="monitor-overview" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="overview-grid">
        <!-- CPU 卡片 -->
        <div class="metric-card" :class="{ warning: isCpuWarning, danger: isCpuDanger }">
          <div class="card-row">
            <div class="card-icon cpu">
              <svg viewBox="0 0 24 24" width="18" height="18">
                <path
                  fill="currentColor"
                  d="M17,17H7V7H17M21,11V9H19V7C19,5.89 18.1,5 17,5H15V3H13V5H11V3H9V5H7C5.89,5 5,5.89 5,7V9H3V11H5V13H3V15H5V17C5,18.1 5.9,19 7,19H9V21H11V19H13V21H15V19H17C18.1,19 19,18.1 19,17V15H21V13H19V11M15,15H9V9H15V15Z"
                />
              </svg>
            </div>
            <span class="card-title">CPU 使用</span>
            <span class="card-badge" :class="getCpuStatusClass">{{ getCpuStatusText }}</span>
          </div>
          <div class="card-value">{{ overviewData?.cpu?.usagePercent?.toFixed(1) || '0.0' }}%</div>
          <div class="card-detail">
            {{ overviewData?.cpu?.usageCores?.toFixed(3) || '0.000' }} /
            {{ overviewData?.cpu?.limitCores?.toFixed(2) || 'N/A' }} cores
          </div>
          <div class="card-progress">
            <div
              class="progress-bar"
              :class="getCpuStatusClass"
              :style="{ width: `${Math.min(overviewData?.cpu?.usagePercent || 0, 100)}%` }"
            ></div>
          </div>
        </div>

        <!-- 内存卡片 -->
        <div class="metric-card" :class="{ warning: isMemoryWarning, danger: isMemoryDanger }">
          <div class="card-row">
            <div class="card-icon memory">
              <svg viewBox="0 0 24 24" width="18" height="18">
                <path
                  fill="currentColor"
                  d="M17,9H7V7H17M17,13H7V11H17M14,17H7V15H14M12,3A1,1 0 0,1 13,4A1,1 0 0,1 12,5A1,1 0 0,1 11,4A1,1 0 0,1 12,3M19,3H14.82C14.4,1.84 13.3,1 12,1C10.7,1 9.6,1.84 9.18,3H5A2,2 0 0,0 3,5V19A2,2 0 0,0 5,21H19A2,2 0 0,0 21,19V5A2,2 0 0,0 19,3Z"
                />
              </svg>
            </div>
            <span class="card-title">内存使用</span>
            <span class="card-badge" :class="getMemoryStatusClass">{{ getMemoryStatusText }}</span>
          </div>
          <div class="card-value"
            >{{ overviewData?.memory?.usagePercent?.toFixed(1) || '0.0' }}%</div
          >
          <div class="card-detail">
            {{ formatBytes(overviewData?.memory?.usageBytes) }} /
            {{ formatBytes(overviewData?.memory?.limitBytes) }}
          </div>
          <div class="card-progress">
            <div
              class="progress-bar"
              :class="getMemoryStatusClass"
              :style="{ width: `${Math.min(overviewData?.memory?.usagePercent || 0, 100)}%` }"
            ></div>
          </div>
        </div>

        <!-- 网络卡片 -->
        <div class="metric-card network">
          <div class="card-row">
            <div class="card-icon network">
              <svg viewBox="0 0 24 24" width="18" height="18">
                <path
                  fill="currentColor"
                  d="M17,3A2,2 0 0,1 19,5V15A2,2 0 0,1 17,17H13V19H14A1,1 0 0,1 15,20H22V22H15A1,1 0 0,1 14,23H10A1,1 0 0,1 9,22H2V20H9A1,1 0 0,1 10,19H11V17H7C5.89,17 5,16.1 5,15V5A2,2 0 0,1 7,3H17M17,5H7V15H17V5Z"
                />
              </svg>
            </div>
            <span class="card-title">网络流量</span>
            <span v-if="hasNetworkErrors" class="card-badge danger">错误</span>
          </div>
          <div class="network-rows">
            <div class="network-row">
              <svg viewBox="0 0 24 24" width="12" height="12" class="icon-receive">
                <path fill="currentColor" d="M9,4H15V12H19.84L12,19.84L4.16,12H9V4Z" />
              </svg>
              <span class="network-label">接收</span>
              <span class="network-value receive">{{
                formatBytes(overviewData?.network?.receiveBytes)
              }}</span>
            </div>
            <div class="network-row">
              <svg viewBox="0 0 24 24" width="12" height="12" class="icon-transmit">
                <path fill="currentColor" d="M15,20H9V12H4.16L12,4.16L19.84,12H15V20Z" />
              </svg>
              <span class="network-label">发送</span>
              <span class="network-value transmit">{{
                formatBytes(overviewData?.network?.transmitBytes)
              }}</span>
            </div>
          </div>
        </div>

        <!-- 状态卡片 - 使用 Pod/船 图标 -->
        <div class="metric-card status">
          <div class="card-row">
            <div class="card-icon status" :class="getPhaseIconClass">
              <!-- Pod/船 图标 -->
              <svg viewBox="0 0 24 24" width="18" height="18">
                <path
                  fill="currentColor"
                  d="M6,6C6,4.89 6.89,4 8,4H9V2H15V4H16C17.11,4 18,4.89 18,6V9H6V6M20,13V22H4V13C4,11.89 4.89,11 6,11H7V9H17V11H18C19.11,11 20,11.89 20,13M8,14V16H10V14H8M14,14V16H16V14H14M8,17V19H10V17H8M14,17V19H16V17H14Z"
                />
              </svg>
            </div>
            <span class="card-title">Pod 状态</span>
            <span class="card-badge" :class="getHealthBadgeClass">{{ getHealthText }}</span>
          </div>
          <div class="status-row">
            <span class="status-badge" :class="getPhaseClass">{{
              overviewData?.status?.phase || 'Unknown'
            }}</span>
          </div>
          <div class="status-info">
            <span class="info-item">
              <svg viewBox="0 0 24 24" width="11" height="11">
                <path
                  fill="currentColor"
                  d="M12,4C14.1,4 16.1,4.8 17.6,6.3C20.7,9.4 20.7,14.5 17.6,17.6C15.8,19.5 13.3,20.2 10.9,19.9L11.4,17.9C13.1,18.1 14.9,17.5 16.2,16.2C18.5,13.9 18.5,10.1 16.2,7.7C15.1,6.6 13.5,6 12,6V10.6L7,5.6L12,0.6V4M6.3,17.6C3.7,15 3.3,11 5.1,7.9L6.6,9.4C5.5,11.6 5.9,14.4 7.8,16.2C8.3,16.7 8.9,17.1 9.6,17.4L9,19.4C8,19 7.1,18.4 6.3,17.6Z"
                />
              </svg>
              重启: {{ overviewData?.restartCount || 0 }}
            </span>
            <span class="info-item" :class="overviewData?.status?.ready ? 'ready' : 'not-ready'">
              <svg v-if="overviewData?.status?.ready" viewBox="0 0 24 24" width="11" height="11">
                <path
                  fill="currentColor"
                  d="M21,7L9,19L3.5,13.5L4.91,12.09L9,16.17L19.59,5.59L21,7Z"
                />
              </svg>
              <svg v-else viewBox="0 0 24 24" width="11" height="11">
                <path
                  fill="currentColor"
                  d="M12,2C17.53,2 22,6.47 22,12C22,17.53 17.53,22 12,22C6.47,22 2,17.53 2,12C2,6.47 6.47,2 12,2M15.59,7L12,10.59L8.41,7L7,8.41L10.59,12L7,15.59L8.41,17L12,13.41L15.59,17L17,15.59L13.41,12L17,8.41L15.59,7Z"
                />
              </svg>
              {{ overviewData?.status?.ready ? '就绪' : '未就绪' }}
            </span>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <svg viewBox="0 0 24 24" width="32" height="32">
          <path
            fill="currentColor"
            d="M13,13H11V7H13M13,17H11V15H13M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2Z"
          />
        </svg>
        <span>暂无概览数据</span>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import type { PodOverview } from '@/api/console/pod-monitor'

  interface Props {
    overviewData?: PodOverview
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    overviewData: undefined,
    loading: false
  })

  // 数据状态
  const hasData = computed(() => {
    return !!(props.overviewData && Object.keys(props.overviewData).length > 0)
  })

  // CPU 状态判断
  const isCpuWarning = computed(() => {
    const percent = props.overviewData?.cpu?.usagePercent || 0
    return percent >= 70 && percent < 90
  })

  const isCpuDanger = computed(() => {
    const percent = props.overviewData?.cpu?.usagePercent || 0
    return percent >= 90
  })

  const getCpuStatusClass = computed(() => {
    if (isCpuDanger.value) return 'danger'
    if (isCpuWarning.value) return 'warning'
    return 'success'
  })

  const getCpuStatusText = computed(() => {
    if (isCpuDanger.value) return '过高'
    if (isCpuWarning.value) return '偏高'
    return '正常'
  })

  // 内存状态判断
  const isMemoryWarning = computed(() => {
    const percent = props.overviewData?.memory?.usagePercent || 0
    return percent >= 70 && percent < 90
  })

  const isMemoryDanger = computed(() => {
    const percent = props.overviewData?.memory?.usagePercent || 0
    return percent >= 90
  })

  const getMemoryStatusClass = computed(() => {
    if (isMemoryDanger.value) return 'danger'
    if (isMemoryWarning.value) return 'warning'
    return 'success'
  })

  const getMemoryStatusText = computed(() => {
    if (isMemoryDanger.value) return '过高'
    if (isMemoryWarning.value) return '偏高'
    return '正常'
  })

  // 网络错误
  const hasNetworkErrors = computed(() => {
    return (
      (props.overviewData?.network?.receiveErrors || 0) > 0 ||
      (props.overviewData?.network?.transmitErrors || 0) > 0
    )
  })

  // 状态相关
  const getPhaseIconClass = computed(() => {
    const phase = props.overviewData?.status?.phase
    switch (phase) {
      case 'Running':
        return 'running'
      case 'Pending':
        return 'pending'
      case 'Failed':
        return 'failed'
      default:
        return ''
    }
  })

  const getPhaseClass = computed(() => {
    const phase = props.overviewData?.status?.phase
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
  })

  const getHealthBadgeClass = computed(() => {
    const phase = props.overviewData?.status?.phase
    const ready = props.overviewData?.status?.ready
    const restarts = props.overviewData?.restartCount || 0

    if (phase === 'Failed' || restarts > 5) return 'danger'
    if (!ready || phase === 'Pending' || restarts > 0) return 'warning'
    if (phase === 'Running' && ready) return 'success'
    return 'info'
  })

  const getHealthText = computed(() => {
    const cls = getHealthBadgeClass.value
    if (cls === 'success') return '健康'
    if (cls === 'warning') return '需关注'
    if (cls === 'danger') return '异常'
    return '未知'
  })

  // 格式化字节
  const formatBytes = (bytes?: number): string => {
    if (!bytes || bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
  }
</script>

<style lang="scss" scoped>
  $primary: #667eea;
  $success: #10b981;
  $warning: #f59e0b;
  $danger: #ef4444;
  $receive: #4facfe;
  $transmit: #fa8c16;
  $gray-50: #f9fafb;
  $gray-100: #f3f4f6;
  $gray-200: #e5e7eb;
  $gray-300: #d1d5db;
  $gray-400: #9ca3af;
  $gray-500: #6b7280;
  $gray-600: #4b5563;
  $gray-700: #374151;
  $gray-800: #1f2937;
  $radius: 12px;
  $radius-sm: 8px;
  $radius-xs: 4px;

  .monitor-overview {
    background: #fff;
    border-radius: $radius;
    padding: 16px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
    margin-bottom: 20px;
    font-family: var(--ax-font-sans);
  }

  .overview-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 12px;

    @media (max-width: 1200px) {
      grid-template-columns: repeat(2, 1fr);
    }

    @media (max-width: 640px) {
      grid-template-columns: 1fr;
    }
  }

  .metric-card {
    padding: 14px 16px;
    border: 1px solid $gray-200;
    border-radius: $radius-sm;
    background: $gray-50;
    transition: all 0.2s ease;
    position: relative;
    overflow: hidden;

    &::before {
      content: '';
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      height: 3px;
      background: $success;
    }

    &.warning::before {
      background: $warning;
    }

    &.danger::before {
      background: $danger;
    }

    &:hover {
      border-color: $gray-300;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
    }

    .card-row {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 8px;
    }

    .card-icon {
      width: 28px;
      height: 28px;
      border-radius: 6px;
      display: flex;
      align-items: center;
      justify-content: center;
      flex-shrink: 0;

      &.cpu {
        background: linear-gradient(135deg, #ede9fe, #ddd6fe);
        color: #7c3aed;
      }

      &.memory {
        background: linear-gradient(135deg, #fce7f3, #fbcfe8);
        color: #db2777;
      }

      &.network {
        background: linear-gradient(135deg, #dbeafe, #bfdbfe);
        color: #2563eb;
      }

      &.status {
        background: linear-gradient(135deg, #ffecd2, #fcb69f);
        color: #c2410c;

        &.running {
          background: linear-gradient(135deg, #d1fae5, #a7f3d0);
          color: $success;
        }

        &.pending {
          background: linear-gradient(135deg, #fef3c7, #fde68a);
          color: $warning;
        }

        &.failed {
          background: linear-gradient(135deg, #fee2e2, #fecaca);
          color: $danger;
        }
      }
    }

    .card-title {
      font-size: 12px;
      color: $gray-500;
      font-weight: 500;
      flex: 1;
    }

    .card-badge {
      padding: 2px 8px;
      border-radius: 10px;
      font-size: 10px;
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

    .card-value {
      font-size: 22px;
      font-weight: 700;
      color: $gray-800;
      line-height: 1.2;
      margin-bottom: 2px;
    }

    .card-detail {
      font-size: 11px;
      color: $gray-400;
      margin-bottom: 8px;
    }

    .card-progress {
      height: 3px;
      background: $gray-200;
      border-radius: 2px;
      overflow: hidden;

      .progress-bar {
        height: 100%;
        border-radius: 2px;
        transition: width 0.4s ease;

        &.success {
          background: $success;
        }

        &.warning {
          background: $warning;
        }

        &.danger {
          background: $danger;
        }
      }
    }

    // 网络卡片样式
    &.network {
      &::before {
        background: $receive;
      }
    }

    .network-rows {
      display: flex;
      flex-direction: column;
      gap: 6px;
      margin-top: 4px;
    }

    .network-row {
      display: flex;
      align-items: center;
      gap: 6px;
      font-size: 12px;

      .icon-receive {
        color: $receive;
      }

      .icon-transmit {
        color: $transmit;
      }

      .network-label {
        color: $gray-500;
        font-weight: 500;
      }

      .network-value {
        margin-left: auto;
        font-weight: 700;
        font-size: 13px;

        &.receive {
          color: #0284c7;
        }

        &.transmit {
          color: #c2410c;
        }
      }
    }

    // 状态卡片样式
    &.status {
      &::before {
        background: $success;
      }
    }

    .status-row {
      margin-bottom: 8px;
    }

    .status-badge {
      display: inline-block;
      padding: 3px 10px;
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

    .status-info {
      display: flex;
      gap: 12px;
      font-size: 11px;

      .info-item {
        display: flex;
        align-items: center;
        gap: 4px;
        color: $gray-500;

        &.ready {
          color: $success;
        }

        &.not-ready {
          color: $danger;
        }
      }
    }
  }

  .empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    padding: 24px;
    color: $gray-400;

    span {
      font-size: 13px;
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
    .monitor-overview {
      padding: 12px;
    }

    .metric-card {
      padding: 12px 14px;

      .card-value {
        font-size: 20px;
      }
    }
  }
</style>
