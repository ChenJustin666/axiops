<template>
  <div class="monitor-overview" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="overview-container">
        <!-- CPU 卡片 -->
        <div
          class="metric-card"
          :class="{
            warning: getUsageLevel(overviewData?.cpu?.usagePercent) === 'warning',
            danger: getUsageLevel(overviewData?.cpu?.usagePercent) === 'danger'
          }"
        >
          <div class="card-icon cpu">
            <img src="@/assets/img/monitoring/CPU.png" alt="CPU" />
          </div>
          <div class="card-body">
            <div class="card-header">
              <span class="card-title">CPU 使用率</span>
              <span class="card-badge" :class="getUsageLevel(overviewData?.cpu?.usagePercent)">
                {{ getUsageLevelText(overviewData?.cpu?.usagePercent) }}
              </span>
            </div>
            <div class="card-value">
              <span class="value">{{ formatPercentage(overviewData?.cpu?.usagePercent) }}</span>
            </div>
            <div class="card-progress">
              <div
                class="progress-fill"
                :style="{ width: `${overviewData?.cpu?.usagePercent || 0}%` }"
                :class="getUsageLevelClass(overviewData?.cpu?.usagePercent)"
              ></div>
            </div>
            <div class="card-detail">
              Load5: {{ overviewData?.cpu?.load5?.toFixed(2) || '0.00' }} /
              {{ overviewData?.cpu?.cores || 0 }} cores
            </div>
          </div>
        </div>

        <!-- 内存卡片 -->
        <div
          class="metric-card"
          :class="{
            warning: getUsageLevel(overviewData?.memory?.usagePercent) === 'warning',
            danger: getUsageLevel(overviewData?.memory?.usagePercent) === 'danger'
          }"
        >
          <div class="card-icon memory">
            <img src="@/assets/img/monitoring/memory.png" alt="Memory" />
          </div>
          <div class="card-body">
            <div class="card-header">
              <span class="card-title">内存使用</span>
              <span class="card-badge" :class="getUsageLevel(overviewData?.memory?.usagePercent)">
                {{ getUsageLevelText(overviewData?.memory?.usagePercent) }}
              </span>
            </div>
            <div class="card-value">
              <span class="value">{{ formatPercentage(overviewData?.memory?.usagePercent) }}</span>
            </div>
            <div class="card-progress">
              <div
                class="progress-fill"
                :style="{ width: `${overviewData?.memory?.usagePercent || 0}%` }"
                :class="getUsageLevelClass(overviewData?.memory?.usagePercent)"
              ></div>
            </div>
            <div class="card-detail">
              {{ formatBytes(overviewData?.memory?.usedBytes) }} /
              {{ formatBytes(overviewData?.memory?.totalBytes) }}
            </div>
            <div v-if="overviewData?.memory?.swapUsagePercent" class="card-extra warning">
              <svg viewBox="0 0 24 24" width="12" height="12">
                <path
                  fill="currentColor"
                  d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"
                />
              </svg>
              <span>Swap: {{ formatPercentage(overviewData.memory.swapUsagePercent) }}</span>
            </div>
          </div>
        </div>

        <!-- 磁盘卡片 -->
        <div
          class="metric-card"
          :class="{
            warning: getUsageLevel(overviewData?.disk?.usagePercent) === 'warning',
            danger: getUsageLevel(overviewData?.disk?.usagePercent) === 'danger'
          }"
        >
          <div class="card-icon disk">
            <img src="@/assets/img/monitoring/disk.png" alt="Disk" />
          </div>
          <div class="card-body">
            <div class="card-header">
              <span class="card-title">磁盘使用</span>
              <span class="card-badge" :class="getUsageLevel(overviewData?.disk?.usagePercent)">
                {{ getUsageLevelText(overviewData?.disk?.usagePercent) }}
              </span>
            </div>
            <div class="card-value">
              <span class="value">{{ formatPercentage(overviewData?.disk?.usagePercent) }}</span>
            </div>
            <div class="card-progress">
              <div
                class="progress-fill"
                :style="{ width: `${overviewData?.disk?.usagePercent || 0}%` }"
                :class="getUsageLevelClass(overviewData?.disk?.usagePercent)"
              ></div>
            </div>
            <div class="card-detail">
              {{ formatBytes(overviewData?.disk?.usedBytes) }} /
              {{ formatBytes(overviewData?.disk?.totalBytes) }}
            </div>
          </div>
        </div>

        <!-- 网络卡片 -->
        <div class="metric-card network">
          <div class="card-icon network">
            <img src="@/assets/img/monitoring/nonetwork.png" alt="Network" />
          </div>
          <div class="card-body">
            <div class="card-header">
              <span class="card-title">网络流量</span>
            </div>
            <div class="network-stats">
              <div class="stat-row receive">
                <svg viewBox="0 0 24 24" width="14" height="14">
                  <path
                    fill="currentColor"
                    d="M11,4H13V16L18.5,10.5L19.92,11.92L12,19.84L4.08,11.92L5.5,10.5L11,16V4Z"
                  />
                </svg>
                <span class="label">接收</span>
                <span class="value">{{
                  formatBitRate(overviewData?.network?.receiveBytesPerSec)
                }}</span>
              </div>
              <div class="stat-row transmit">
                <svg viewBox="0 0 24 24" width="14" height="14">
                  <path
                    fill="currentColor"
                    d="M13,20H11V8L5.5,13.5L4.08,12.08L12,4.16L19.92,12.08L18.5,13.5L13,8V20Z"
                  />
                </svg>
                <span class="label">发送</span>
                <span class="value">{{
                  formatBitRate(overviewData?.network?.transmitBytesPerSec)
                }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Pod 状态卡片 -->
        <div class="metric-card pods">
          <div class="card-icon pods">
            <img src="@/assets/img/monitoring/pod.png" alt="Pod" />
          </div>
          <div class="card-body">
            <div class="card-header">
              <span class="card-title">Pod 状态</span>
              <span class="card-badge" :class="overviewData?.ready ? 'normal' : 'warning'">
                {{ overviewData?.ready ? '健康' : '异常' }}
              </span>
            </div>
            <div class="pods-grid">
              <div class="pod-stat">
                <span class="stat-value">{{ overviewData?.pods?.total || 0 }}</span>
                <span class="stat-label">Total</span>
              </div>
              <div class="pod-stat running">
                <span class="stat-value">{{ overviewData?.pods?.running || 0 }}</span>
                <span class="stat-label">Running</span>
              </div>
              <div v-if="overviewData?.pods?.pending" class="pod-stat pending">
                <span class="stat-value">{{ overviewData.pods.pending }}</span>
                <span class="stat-label">Pending</span>
              </div>
              <div v-if="overviewData?.pods?.failed" class="pod-stat failed">
                <span class="stat-value">{{ overviewData.pods.failed }}</span>
                <span class="stat-label">Failed</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" width="48" height="48">
            <path
              fill="currentColor"
              d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14h-2V9h2v8zm0-10h-2V5h2v2z"
            />
          </svg>
        </div>
        <p>暂无概览数据</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import { formatBytes, formatBitRate, formatPercentage } from '@/utils/format'

  interface Props {
    overviewData?: any
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    overviewData: undefined,
    loading: false
  })

  // 判断是否有数据
  const hasData = computed(() => {
    return !!(props.overviewData && Object.keys(props.overviewData).length > 0)
  })

  // 获取使用率级别
  const getUsageLevel = (percent?: number): string => {
    if (!percent) return 'normal'
    if (percent < 50) return 'normal'
    if (percent < 80) return 'warning'
    return 'danger'
  }

  // 获取使用率级别样式类
  const getUsageLevelClass = (percent?: number): string => {
    if (!percent) return 'level-normal'
    if (percent < 50) return 'level-normal'
    if (percent < 80) return 'level-warning'
    return 'level-danger'
  }

  // 获取使用率级别文本
  const getUsageLevelText = (percent?: number): string => {
    if (!percent) return '正常'
    if (percent < 50) return '正常'
    if (percent < 80) return '偏高'
    return '过高'
  }
</script>

<style lang="scss" scoped>
  // 颜色变量
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

  .monitor-overview {
    margin-bottom: 20px;
  }

  .overview-container {
    display: grid;
    grid-template-columns: repeat(5, 1fr);
    gap: 16px;

    @media (max-width: 1400px) {
      grid-template-columns: repeat(3, 1fr);
    }

    @media (max-width: 992px) {
      grid-template-columns: repeat(2, 1fr);
    }

    @media (max-width: 640px) {
      grid-template-columns: 1fr;
    }
  }

  .metric-card {
    position: relative;
    display: flex;
    gap: 14px;
    padding: 18px;
    background: #fff;
    border-radius: $radius;
    border: 1px solid $gray-200;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
    transition: all 0.2s ease;
    overflow: hidden;

    // 顶部装饰条
    &::before {
      content: '';
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      height: 3px;
      background: $gray-200;
      transition: background 0.3s ease;
    }

    &:hover {
      transform: translateY(-2px);
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
      border-color: $gray-300;

      &::before {
        background: $primary;
      }
    }

    &.warning {
      border-color: rgba($warning, 0.3);

      &::before {
        background: $warning;
      }
    }

    &.danger {
      border-color: rgba($danger, 0.3);

      &::before {
        background: $danger;
      }
    }

    .card-icon {
      width: 48px;
      height: 48px;
      border-radius: $radius-sm;
      display: flex;
      align-items: center;
      justify-content: center;
      flex-shrink: 0;

      img {
        width: 28px;
        height: 28px;
        object-fit: contain;
      }

      &.cpu {
        background: linear-gradient(135deg, #ede9fe, #ddd6fe);
      }

      &.memory {
        background: linear-gradient(135deg, #fce7f3, #fbcfe8);
      }

      &.disk {
        background: linear-gradient(135deg, #d1fae5, #a7f3d0);
      }

      &.network {
        background: linear-gradient(135deg, #dbeafe, #bfdbfe);
      }

      &.pods {
        background: linear-gradient(135deg, #fee2e2, #fecaca);
      }
    }

    .card-body {
      flex: 1;
      min-width: 0;
    }

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 8px;

      .card-title {
        font-size: 12px;
        font-weight: 500;
        color: $gray-500;
      }

      .card-badge {
        padding: 2px 8px;
        border-radius: 10px;
        font-size: 10px;
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

    .card-value {
      margin-bottom: 8px;

      .value {
        font-size: 24px;
        font-weight: 700;
        color: $gray-800;
        line-height: 1.2;
      }
    }

    .card-progress {
      height: 4px;
      background: $gray-100;
      border-radius: 2px;
      overflow: hidden;
      margin-bottom: 8px;

      .progress-fill {
        height: 100%;
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

    .card-detail {
      font-size: 11px;
      color: $gray-500;
      line-height: 1.4;
    }

    .card-extra {
      display: flex;
      align-items: center;
      gap: 4px;
      margin-top: 6px;
      font-size: 10px;

      &.warning {
        color: $warning;
      }
    }

    // 网络卡片样式
    .network-stats {
      display: flex;
      flex-direction: column;
      gap: 6px;

      .stat-row {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 6px 10px;
        border-radius: $radius-xs;
        font-size: 11px;

        &.receive {
          background: linear-gradient(135deg, rgba(#0ea5e9, 0.08), rgba(#0ea5e9, 0.04));

          svg {
            color: #0ea5e9;
          }
        }

        &.transmit {
          background: linear-gradient(135deg, rgba($warning, 0.08), rgba($warning, 0.04));

          svg {
            color: $warning;
          }
        }

        .label {
          color: $gray-500;
          font-weight: 500;
        }

        .value {
          margin-left: auto;
          font-weight: 700;
          color: $gray-700;
        }
      }
    }

    // Pod 卡片样式
    .pods-grid {
      display: grid;
      grid-template-columns: repeat(2, 1fr);
      gap: 6px;

      .pod-stat {
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 8px 6px;
        background: $gray-50;
        border-radius: $radius-xs;

        .stat-value {
          font-size: 16px;
          font-weight: 700;
          color: $gray-800;
        }

        .stat-label {
          font-size: 9px;
          color: $gray-500;
          margin-top: 2px;
        }

        &.running {
          background: linear-gradient(135deg, rgba($success, 0.1), rgba($success, 0.05));

          .stat-value {
            color: #059669;
          }
        }

        &.pending {
          background: linear-gradient(135deg, rgba($warning, 0.1), rgba($warning, 0.05));

          .stat-value {
            color: #d97706;
          }
        }

        &.failed {
          background: linear-gradient(135deg, rgba($danger, 0.1), rgba($danger, 0.05));

          .stat-value {
            color: #dc2626;
          }
        }
      }
    }
  }

  // 空状态
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 150px;
    background: #fff;
    border-radius: $radius;
    padding: 30px;

    .empty-icon {
      color: $gray-300;
      margin-bottom: 12px;
    }

    p {
      margin: 0;
      font-size: 13px;
      color: $gray-400;
    }
  }

  // 动画
  .fade-enter-active,
  .fade-leave-active {
    transition: opacity 0.3s ease;
  }

  .fade-enter-from,
  .fade-leave-to {
    opacity: 0;
  }
</style>
