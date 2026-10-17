<template>
  <div class="resource-quota-monitor" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="monitor-card">
        <!-- 卡片头部 -->
        <div class="card-header">
          <div class="header-info">
            <div class="icon-badge quota">
              <svg viewBox="0 0 24 24" width="22" height="22">
                <path
                  fill="currentColor"
                  d="M19,3H14.82C14.4,1.84 13.3,1 12,1C10.7,1 9.6,1.84 9.18,3H5C3.9,3 3,3.9 3,5V19C3,20.1 3.9,21 5,21H19C20.1,21 21,20.1 21,19V5C21,3.9 20.1,3 19,3M12,3C12.55,3 13,3.45 13,4C13,4.55 12.55,5 12,5C11.45,5 11,4.55 11,4C11,3.45 11.45,3 12,3M7,7H17V5H19V19H5V5H7V7M12,17L17,12L15.59,10.59L12,14.17L9.91,12.09L8.5,13.5L12,17Z"
                />
              </svg>
            </div>
            <div>
              <h3>资源配额</h3>
              <p>Namespace 配额使用情况</p>
            </div>
          </div>
          <div class="header-stats">
            <div class="stat-badge" :class="getQuotaLevelClass(quotaData?.cpu?.usagePercent)">
              <span class="label">CPU</span>
              <span class="value">{{ formatPercentage(quotaData?.cpu?.usagePercent) }}</span>
            </div>
            <div class="stat-badge" :class="getQuotaLevelClass(quotaData?.memory?.usagePercent)">
              <span class="label">内存</span>
              <span class="value">{{ formatPercentage(quotaData?.memory?.usagePercent) }}</span>
            </div>
            <div class="stat-badge" :class="getQuotaLevelClass(quotaData?.pods?.usagePercent)">
              <span class="label">Pods</span>
              <span class="value">{{ formatPercentage(quotaData?.pods?.usagePercent) }}</span>
            </div>
          </div>
        </div>

        <!-- 内容区域 -->
        <div class="content-wrapper">
          <!-- 主要资源配额卡片 -->
          <div class="quota-grid">
            <!-- CPU 配额 -->
            <div class="quota-item" :class="getQuotaLevel(quotaData?.cpu?.usagePercent)">
              <div class="quota-header">
                <div class="quota-icon cpu">
                  <svg viewBox="0 0 24 24" width="20" height="20">
                    <path
                      fill="currentColor"
                      d="M17,17H7V7H17M21,11V9H19V7C19,5.89 18.1,5 17,5H15V3H13V5H11V3H9V5H7C5.89,5 5,5.89 5,7V9H3V11H5V13H3V15H5V17C5,18.1 5.9,19 7,19H9V21H11V19H13V21H15V19H17C18.1,19 19,18.1 19,17V15H21V13H19V11M15,15H9V9H15V15Z"
                    />
                  </svg>
                </div>
                <span class="quota-name">CPU 配额</span>
                <div class="quota-status" :class="getQuotaLevel(quotaData?.cpu?.usagePercent)">
                  <svg
                    v-if="(quotaData?.cpu?.usagePercent || 0) >= 90"
                    viewBox="0 0 24 24"
                    width="14"
                    height="14"
                  >
                    <path
                      fill="currentColor"
                      d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"
                    />
                  </svg>
                  <svg
                    v-else-if="(quotaData?.cpu?.usagePercent || 0) >= 70"
                    viewBox="0 0 24 24"
                    width="14"
                    height="14"
                  >
                    <path
                      fill="currentColor"
                      d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"
                    />
                  </svg>
                  <svg v-else viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"
                    />
                  </svg>
                  <span>{{ getQuotaStatusText(quotaData?.cpu?.usagePercent) }}</span>
                </div>
              </div>

              <div class="quota-values">
                <div class="value-item">
                  <span class="value-label">已使用</span>
                  <span class="value-number">{{ formatCores(quotaData?.cpu?.used) }}</span>
                </div>
                <div class="value-divider">/</div>
                <div class="value-item">
                  <span class="value-label">总配额</span>
                  <span class="value-number">{{ formatCores(quotaData?.cpu?.hard) }}</span>
                </div>
              </div>

              <div class="quota-progress">
                <div class="progress-bar">
                  <div
                    class="progress-fill"
                    :class="getQuotaLevel(quotaData?.cpu?.usagePercent)"
                    :style="{ width: Math.min(quotaData?.cpu?.usagePercent || 0, 100) + '%' }"
                  ></div>
                </div>
                <span class="progress-text">{{
                  formatPercentage(quotaData?.cpu?.usagePercent)
                }}</span>
              </div>
            </div>

            <!-- 内存配额 -->
            <div class="quota-item" :class="getQuotaLevel(quotaData?.memory?.usagePercent)">
              <div class="quota-header">
                <div class="quota-icon memory">
                  <svg viewBox="0 0 24 24" width="20" height="20">
                    <path
                      fill="currentColor"
                      d="M17,9H7V7H17M17,13H7V11H17M14,17H7V15H14M12,3A1,1 0 0,1 13,4A1,1 0 0,1 12,5A1,1 0 0,1 11,4A1,1 0 0,1 12,3M19,3H14.82C14.4,1.84 13.3,1 12,1C10.7,1 9.6,1.84 9.18,3H5A2,2 0 0,0 3,5V19A2,2 0 0,0 5,21H19A2,2 0 0,0 21,19V5A2,2 0 0,0 19,3Z"
                    />
                  </svg>
                </div>
                <span class="quota-name">内存配额</span>
                <div class="quota-status" :class="getQuotaLevel(quotaData?.memory?.usagePercent)">
                  <svg
                    v-if="(quotaData?.memory?.usagePercent || 0) >= 90"
                    viewBox="0 0 24 24"
                    width="14"
                    height="14"
                  >
                    <path
                      fill="currentColor"
                      d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"
                    />
                  </svg>
                  <svg
                    v-else-if="(quotaData?.memory?.usagePercent || 0) >= 70"
                    viewBox="0 0 24 24"
                    width="14"
                    height="14"
                  >
                    <path
                      fill="currentColor"
                      d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"
                    />
                  </svg>
                  <svg v-else viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"
                    />
                  </svg>
                  <span>{{ getQuotaStatusText(quotaData?.memory?.usagePercent) }}</span>
                </div>
              </div>

              <div class="quota-values">
                <div class="value-item">
                  <span class="value-label">已使用</span>
                  <span class="value-number">{{ formatBytes(quotaData?.memory?.used) }}</span>
                </div>
                <div class="value-divider">/</div>
                <div class="value-item">
                  <span class="value-label">总配额</span>
                  <span class="value-number">{{ formatBytes(quotaData?.memory?.hard) }}</span>
                </div>
              </div>

              <div class="quota-progress">
                <div class="progress-bar">
                  <div
                    class="progress-fill"
                    :class="getQuotaLevel(quotaData?.memory?.usagePercent)"
                    :style="{ width: Math.min(quotaData?.memory?.usagePercent || 0, 100) + '%' }"
                  ></div>
                </div>
                <span class="progress-text">{{
                  formatPercentage(quotaData?.memory?.usagePercent)
                }}</span>
              </div>
            </div>

            <!-- Pods 配额 -->
            <div class="quota-item" :class="getQuotaLevel(quotaData?.pods?.usagePercent)">
              <div class="quota-header">
                <div class="quota-icon pods">
                  <svg viewBox="0 0 24 24" width="20" height="20">
                    <path
                      fill="currentColor"
                      d="M12,2A3,3 0 0,1 15,5V11A3,3 0 0,1 12,14A3,3 0 0,1 9,11V5A3,3 0 0,1 12,2M7.5,22A3.5,3.5 0 0,1 4,18.5V17.5A3.5,3.5 0 0,1 7.5,14A3.5,3.5 0 0,1 11,17.5V18.5A3.5,3.5 0 0,1 7.5,22M16.5,22A3.5,3.5 0 0,1 13,18.5V17.5A3.5,3.5 0 0,1 16.5,14A3.5,3.5 0 0,1 20,17.5V18.5A3.5,3.5 0 0,1 16.5,22Z"
                    />
                  </svg>
                </div>
                <span class="quota-name">Pods 配额</span>
                <div class="quota-status" :class="getQuotaLevel(quotaData?.pods?.usagePercent)">
                  <svg
                    v-if="(quotaData?.pods?.usagePercent || 0) >= 90"
                    viewBox="0 0 24 24"
                    width="14"
                    height="14"
                  >
                    <path
                      fill="currentColor"
                      d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"
                    />
                  </svg>
                  <svg
                    v-else-if="(quotaData?.pods?.usagePercent || 0) >= 70"
                    viewBox="0 0 24 24"
                    width="14"
                    height="14"
                  >
                    <path
                      fill="currentColor"
                      d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"
                    />
                  </svg>
                  <svg v-else viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"
                    />
                  </svg>
                  <span>{{ getQuotaStatusText(quotaData?.pods?.usagePercent) }}</span>
                </div>
              </div>

              <div class="quota-values">
                <div class="value-item">
                  <span class="value-label">已使用</span>
                  <span class="value-number">{{ quotaData?.pods?.used || 0 }}</span>
                </div>
                <div class="value-divider">/</div>
                <div class="value-item">
                  <span class="value-label">总配额</span>
                  <span class="value-number">{{ quotaData?.pods?.hard || 0 }}</span>
                </div>
              </div>

              <div class="quota-progress">
                <div class="progress-bar">
                  <div
                    class="progress-fill"
                    :class="getQuotaLevel(quotaData?.pods?.usagePercent)"
                    :style="{ width: Math.min(quotaData?.pods?.usagePercent || 0, 100) + '%' }"
                  ></div>
                </div>
                <span class="progress-text">{{
                  formatPercentage(quotaData?.pods?.usagePercent)
                }}</span>
              </div>
            </div>
          </div>

          <!-- 其他资源配额 -->
          <div
            v-if="quotaData?.allResources && quotaData.allResources.length > 0"
            class="other-resources"
          >
            <div class="section-header">
              <div class="section-icon">
                <svg viewBox="0 0 24 24" width="18" height="18">
                  <path
                    fill="currentColor"
                    d="M3 13h2v-2H3v2zm0 4h2v-2H3v2zm0-8h2V7H3v2zm4 4h14v-2H7v2zm0 4h14v-2H7v2zM7 7v2h14V7H7z"
                  />
                </svg>
              </div>
              <span class="section-title">其他资源配额</span>
              <span class="section-count">{{ quotaData.allResources.length }} 项</span>
            </div>

            <div class="resources-grid">
              <div
                v-for="resource in quotaData.allResources"
                :key="resource.resource"
                class="resource-item"
              >
                <div class="resource-header">
                  <span class="resource-name">{{ resource.resource }}</span>
                  <span class="resource-usage">{{ resource.used }} / {{ resource.hard }}</span>
                </div>
                <div class="resource-progress">
                  <div class="progress-bar">
                    <div
                      class="progress-fill"
                      :class="getQuotaLevel(resource.percent)"
                      :style="{ width: Math.min(resource.percent, 100) + '%' }"
                    ></div>
                  </div>
                  <span class="progress-text" :class="getQuotaLevel(resource.percent)">
                    {{ resource.percent.toFixed(1) }}%
                  </span>
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
              d="M19,3H14.82C14.4,1.84 13.3,1 12,1C10.7,1 9.6,1.84 9.18,3H5C3.9,3 3,3.9 3,5V19C3,20.1 3.9,21 5,21H19C20.1,21 21,20.1 21,19V5C21,3.9 20.1,3 19,3M12,3C12.55,3 13,3.45 13,4C13,4.55 12.55,5 12,5C11.45,5 11,4.55 11,4C11,3.45 11.45,3 12,3"
            />
          </svg>
        </div>
        <h3>暂无资源配额数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import type { NamespaceQuotaMetrics } from '@/api/console/namespace-monitor'
  import { formatBytes, formatPercentage } from '@/utils/format'

  interface Props {
    quotaData?: NamespaceQuotaMetrics
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    quotaData: undefined,
    loading: false
  })

  // 判断是否有数据
  const hasData = computed(() => {
    return !!(props.quotaData && Object.keys(props.quotaData).length > 0)
  })

  // 格式化 CPU 核心数显示
  const formatCores = (cores?: number): string => {
    if (!cores || cores === 0) return '0'
    // 小于 1 核时显示毫核
    if (cores < 1) return `${(cores * 1000).toFixed(0)}m`
    return `${cores.toFixed(3)}`
  }

  // 获取配额使用级别
  const getQuotaLevel = (percent?: number): string => {
    if (!percent) return 'safe'
    if (percent < 70) return 'safe'
    if (percent < 90) return 'warning'
    return 'critical'
  }

  // 获取配额级别样式类
  const getQuotaLevelClass = (percent?: number): string => {
    const level = getQuotaLevel(percent)
    return `level-${level}`
  }

  // 获取配额状态文本
  const getQuotaStatusText = (percent?: number): string => {
    if (!percent) return '充足'
    if (percent < 70) return '充足'
    if (percent < 90) return '注意'
    return '紧张'
  }
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

  .resource-quota-monitor {
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
      background: linear-gradient(135deg, #fff7ed 0%, #ffedd5 100%);
      border-bottom: 1px solid #fed7aa;
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

          &.quota {
            background: linear-gradient(135deg, #fb923c, #f97316);
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

          &.level-safe {
            background: linear-gradient(135deg, #d1fae5 0%, #a7f3d0 100%);
            color: #065f46;
          }

          &.level-warning {
            background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
            color: #92400e;
          }

          &.level-critical {
            background: linear-gradient(135deg, #fee2e2 0%, #fecaca 100%);
            color: #991b1b;
          }
        }
      }
    }

    // 内容区域
    .content-wrapper {
      padding: 20px;
    }

    // 配额网格
    .quota-grid {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 16px;
      margin-bottom: 24px;

      @media (max-width: 1200px) {
        grid-template-columns: repeat(2, 1fr);
      }

      @media (max-width: 768px) {
        grid-template-columns: 1fr;
      }
    }

    // 配额项
    .quota-item {
      padding: 20px;
      background: $gray-50;
      border-radius: $radius-sm;
      border: 1px solid $gray-200;
      transition: all 0.2s ease;

      &:hover {
        border-color: $gray-300;
      }

      &.safe {
        border-left: 4px solid $success;
      }

      &.warning {
        border-left: 4px solid $warning;
        background: linear-gradient(135deg, #fffbeb 0%, $gray-50 100%);
      }

      &.critical {
        border-left: 4px solid $danger;
        background: linear-gradient(135deg, #fef2f2 0%, $gray-50 100%);
      }

      .quota-header {
        display: flex;
        align-items: center;
        gap: 10px;
        margin-bottom: 16px;

        .quota-icon {
          width: 36px;
          height: 36px;
          border-radius: $radius-xs;
          display: flex;
          align-items: center;
          justify-content: center;
          color: #fff;

          &.cpu {
            background: linear-gradient(135deg, #667eea, #764ba2);
          }

          &.memory {
            background: linear-gradient(135deg, #f093fb, #f5576c);
          }

          &.pods {
            background: linear-gradient(135deg, #4facfe, #00f2fe);
          }
        }

        .quota-name {
          font-size: 14px;
          font-weight: 600;
          color: $gray-700;
          flex: 1;
        }

        .quota-status {
          display: flex;
          align-items: center;
          gap: 4px;
          padding: 4px 10px;
          border-radius: 12px;
          font-size: 11px;
          font-weight: 600;

          &.safe {
            background: rgba($success, 0.1);
            color: #059669;
          }

          &.warning {
            background: rgba($warning, 0.1);
            color: #d97706;
          }

          &.critical {
            background: rgba($danger, 0.1);
            color: #dc2626;
          }
        }
      }

      .quota-values {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 16px;
        padding: 14px;
        background: #fff;
        border-radius: $radius-xs;
        border: 1px solid $gray-200;

        .value-item {
          display: flex;
          flex-direction: column;
          gap: 4px;

          .value-label {
            font-size: 11px;
            color: $gray-400;
          }

          .value-number {
            font-size: 18px;
            font-weight: 700;
            color: $gray-800;
          }
        }

        .value-divider {
          font-size: 20px;
          color: $gray-300;
          font-weight: 300;
        }
      }

      .quota-progress {
        display: flex;
        align-items: center;
        gap: 12px;

        .progress-bar {
          flex: 1;
          height: 8px;
          background: $gray-200;
          border-radius: 4px;
          overflow: hidden;

          .progress-fill {
            height: 100%;
            border-radius: 4px;
            transition: width 0.4s ease;

            &.safe {
              background: linear-gradient(90deg, $success, #34d399);
            }

            &.warning {
              background: linear-gradient(90deg, $warning, #fbbf24);
            }

            &.critical {
              background: linear-gradient(90deg, $danger, #f87171);
            }
          }
        }

        .progress-text {
          font-size: 13px;
          font-weight: 600;
          color: $gray-600;
          min-width: 50px;
          text-align: right;
        }
      }
    }

    // 其他资源配额
    .other-resources {
      .section-header {
        display: flex;
        align-items: center;
        gap: 10px;
        margin-bottom: 16px;
        padding-bottom: 12px;
        border-bottom: 1px solid $gray-200;

        .section-icon {
          width: 32px;
          height: 32px;
          border-radius: $radius-xs;
          background: linear-gradient(135deg, #e0e7ff, #c7d2fe);
          display: flex;
          align-items: center;
          justify-content: center;
          color: #4c51bf;
        }

        .section-title {
          font-size: 14px;
          font-weight: 600;
          color: $gray-700;
        }

        .section-count {
          margin-left: auto;
          font-size: 12px;
          color: $gray-500;
          background: $gray-100;
          padding: 4px 10px;
          border-radius: 12px;
        }
      }

      .resources-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
        gap: 14px;

        .resource-item {
          padding: 14px;
          background: $gray-50;
          border-radius: $radius-xs;
          border: 1px solid $gray-200;
          transition: all 0.2s ease;

          &:hover {
            border-color: $gray-300;
          }

          .resource-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 10px;

            .resource-name {
              font-size: 13px;
              font-weight: 600;
              color: $gray-700;
            }

            .resource-usage {
              font-size: 12px;
              color: $gray-500;
              font-weight: 500;
            }
          }

          .resource-progress {
            display: flex;
            align-items: center;
            gap: 10px;

            .progress-bar {
              flex: 1;
              height: 6px;
              background: $gray-200;
              border-radius: 3px;
              overflow: hidden;

              .progress-fill {
                height: 100%;
                border-radius: 3px;
                transition: width 0.4s ease;

                &.safe {
                  background: $success;
                }

                &.warning {
                  background: $warning;
                }

                &.critical {
                  background: $danger;
                }
              }
            }

            .progress-text {
              font-size: 11px;
              font-weight: 600;
              min-width: 40px;
              text-align: right;

              &.safe {
                color: #059669;
              }

              &.warning {
                color: #d97706;
              }

              &.critical {
                color: #dc2626;
              }
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
      min-height: 300px;
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

  // 过渡动画
  .fade-enter-active,
  .fade-leave-active {
    transition: opacity 0.3s ease;
  }

  .fade-enter-from,
  .fade-leave-to {
    opacity: 0;
  }
</style>
