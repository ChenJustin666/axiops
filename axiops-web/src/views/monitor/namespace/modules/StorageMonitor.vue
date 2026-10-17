<template>
  <div class="namespace-storage-monitor" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="monitor-card">
        <!-- 卡片头部 -->
        <div class="card-header">
          <div class="header-info">
            <div class="icon-badge storage">
              <svg viewBox="0 0 24 24" width="22" height="22">
                <path
                  fill="currentColor"
                  d="M6,2H18A2,2 0 0,1 20,4V20A2,2 0 0,1 18,22H6A2,2 0 0,1 4,20V4A2,2 0 0,1 6,2M12,4A6,6 0 0,0 6,10C6,13.31 8.69,16 12,16A6,6 0 0,0 18,10C18,6.69 15.31,4 12,4M12,9A1,1 0 0,1 13,10A1,1 0 0,1 12,11A1,1 0 0,1 11,10A1,1 0 0,1 12,9M7,18A1,1 0 0,0 6,19A1,1 0 0,0 7,20A1,1 0 0,0 8,19A1,1 0 0,0 7,18M12.09,13.27L14.58,19.58L17.17,18.58L12.95,12.77C12.65,12.89 12.33,13 12,13C11.67,13 11.35,12.89 11.05,12.77L6.83,18.58L9.42,19.58L11.91,13.27C11.95,13.28 12,13.28 12,13.28C12.03,13.28 12.06,13.28 12.09,13.27Z"
                />
              </svg>
            </div>
            <div>
              <h3>存储统计</h3>
              <p>PVC 和存储容量监控</p>
            </div>
          </div>
          <div class="header-stats">
            <div class="stat-badge total">
              <span class="label">总 PVC</span>
              <span class="value">{{ storageData?.pvcTotal || 0 }}</span>
            </div>
            <div class="stat-badge bound">
              <span class="label">已绑定</span>
              <span class="value">{{ storageData?.pvcBound || 0 }}</span>
            </div>
            <div class="stat-badge pending" v-if="(storageData?.pvcPending || 0) > 0">
              <span class="label">待绑定</span>
              <span class="value">{{ storageData?.pvcPending || 0 }}</span>
            </div>
            <div class="stat-badge usage">
              <span class="label">总容量</span>
              <span class="value">{{ formatBytes(storageData?.totalRequestedBytes || 0) }}</span>
            </div>
          </div>
        </div>

        <!-- 内容区域 -->
        <div class="content-wrapper">
          <!-- PVC 状态概览 -->
          <div class="pvc-overview">
            <div class="overview-title">PVC 状态分布</div>
            <div class="status-grid">
              <div class="status-item total">
                <div class="status-icon">
                  <svg viewBox="0 0 24 24" width="24" height="24">
                    <path
                      fill="currentColor"
                      d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14h-2V9h2v8zm0-10h-2V5h2v2z"
                    />
                  </svg>
                </div>
                <div class="status-info">
                  <span class="status-label">总 PVC 数量</span>
                  <span class="status-value">{{ storageData?.pvcTotal || 0 }}</span>
                </div>
                <div class="status-progress">
                  <div class="progress-fill" style="width: 100%"></div>
                </div>
              </div>

              <div class="status-item bound">
                <div class="status-icon">
                  <svg viewBox="0 0 24 24" width="24" height="24">
                    <path
                      fill="currentColor"
                      d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"
                    />
                  </svg>
                </div>
                <div class="status-info">
                  <span class="status-label">已绑定</span>
                  <span class="status-value">{{ storageData?.pvcBound || 0 }}</span>
                </div>
                <div class="status-progress">
                  <div class="progress-fill" :style="{ width: getBoundPercent + '%' }"></div>
                </div>
              </div>

              <div class="status-item pending">
                <div class="status-icon">
                  <svg viewBox="0 0 24 24" width="24" height="24">
                    <path
                      fill="currentColor"
                      d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10 10-4.5 10-10S17.5 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67V7z"
                    />
                  </svg>
                </div>
                <div class="status-info">
                  <span class="status-label">待绑定</span>
                  <span class="status-value">{{ storageData?.pvcPending || 0 }}</span>
                </div>
                <div class="status-progress">
                  <div class="progress-fill" :style="{ width: getPendingPercent + '%' }"></div>
                </div>
              </div>
            </div>
          </div>

          <!-- 容量统计 -->
          <div class="capacity-stats">
            <div class="stats-title">容量信息</div>
            <div class="capacity-card">
              <div class="capacity-icon">
                <svg viewBox="0 0 24 24" width="28" height="28">
                  <path
                    fill="currentColor"
                    d="M2 20h20v-4H2v4zm2-3h2v2H4v-2zM2 4v4h20V4H2zm4 3H4V5h2v2zm-4 7h20v-4H2v4zm2-3h2v2H4v-2z"
                  />
                </svg>
              </div>
              <div class="capacity-info">
                <span class="capacity-label">总请求容量</span>
                <span class="capacity-value">{{
                  formatBytes(storageData?.totalRequestedBytes || 0)
                }}</span>
              </div>
            </div>
          </div>

          <!-- 存储类统计 -->
          <div
            v-if="storageData?.byStorageClass && storageData.byStorageClass.length > 0"
            class="storage-classes"
          >
            <div class="section-header">
              <div class="section-icon">
                <svg viewBox="0 0 24 24" width="18" height="18">
                  <path
                    fill="currentColor"
                    d="M14,2H6A2,2 0 0,0 4,4V20A2,2 0 0,0 6,22H18A2,2 0 0,0 20,20V8L14,2M18,20H6V4H13V9H18V20Z"
                  />
                </svg>
              </div>
              <span class="section-title">按存储类统计</span>
              <span class="section-count">{{ storageData.byStorageClass.length }} 个存储类</span>
            </div>

            <div class="storage-class-list">
              <div
                v-for="(sc, index) in storageData.byStorageClass"
                :key="sc.storageClass"
                class="storage-class-item"
              >
                <div class="sc-index">{{ index + 1 }}</div>
                <div class="sc-content">
                  <div class="sc-main">
                    <span class="sc-name">{{ sc.storageClass }}</span>
                    <span class="sc-size">{{ formatBytes(sc.totalBytes) }}</span>
                  </div>
                  <div class="sc-meta">
                    <span class="sc-count">{{ sc.pvcCount }} 个 PVC</span>
                    <div class="sc-bar">
                      <div
                        class="sc-bar-fill"
                        :style="{ width: getStorageClassPercent(sc.totalBytes) + '%' }"
                      ></div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 无存储类数据提示 -->
          <div v-else-if="storageData?.pvcTotal === 0" class="no-storage-class">
            <svg viewBox="0 0 24 24" width="32" height="32">
              <path
                fill="currentColor"
                d="M11 15h2v2h-2v-2zm0-8h2v6h-2V7zm.99-5C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8z"
              />
            </svg>
            <span>暂无存储类数据</span>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" width="64" height="64">
            <path
              fill="currentColor"
              d="M6,2H18A2,2 0 0,1 20,4V20A2,2 0 0,1 18,22H6A2,2 0 0,1 4,20V4A2,2 0 0,1 6,2M12,4A6,6 0 0,0 6,10C6,13.31 8.69,16 12,16A6,6 0 0,0 18,10C18,6.69 15.31,4 12,4M12,9A1,1 0 0,1 13,10A1,1 0 0,1 12,11A1,1 0 0,1 11,10A1,1 0 0,1 12,9"
            />
          </svg>
        </div>
        <h3>暂无存储数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import type { NamespaceStorageMetrics } from '@/api/console/namespace-monitor'
  import { formatBytes } from '@/utils/format'

  interface Props {
    storageData?: NamespaceStorageMetrics
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    storageData: undefined,
    loading: false
  })

  // 判断是否有数据
  const hasData = computed(() => {
    return !!(props.storageData && Object.keys(props.storageData).length > 0)
  })

  // 计算已绑定 PVC 百分比
  const getBoundPercent = computed(() => {
    if (!props.storageData?.pvcTotal || props.storageData.pvcTotal === 0) return 0
    return Math.min(((props.storageData.pvcBound || 0) / props.storageData.pvcTotal) * 100, 100)
  })

  // 计算待绑定 PVC 百分比
  const getPendingPercent = computed(() => {
    if (!props.storageData?.pvcTotal || props.storageData.pvcTotal === 0) return 0
    return Math.min(((props.storageData.pvcPending || 0) / props.storageData.pvcTotal) * 100, 100)
  })

  // 计算存储类容量占比
  const getStorageClassPercent = (bytes: number): number => {
    if (!props.storageData?.totalRequestedBytes || props.storageData.totalRequestedBytes === 0)
      return 0
    return Math.min((bytes / props.storageData.totalRequestedBytes) * 100, 100)
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

  .namespace-storage-monitor {
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
      background: linear-gradient(135deg, #ecfdf5 0%, #d1fae5 100%);
      border-bottom: 1px solid #a7f3d0;
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

          &.storage {
            background: linear-gradient(135deg, $success, #059669);
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

          &.total {
            background: linear-gradient(135deg, #e0e7ff 0%, #c7d2fe 100%);
            color: #4c51bf;
          }

          &.bound {
            background: linear-gradient(135deg, #d1fae5 0%, #a7f3d0 100%);
            color: #065f46;
          }

          &.pending {
            background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
            color: #92400e;
          }

          &.usage {
            background: linear-gradient(135deg, #dbeafe 0%, #bfdbfe 100%);
            color: #1e40af;
          }
        }
      }
    }

    // 内容区域
    .content-wrapper {
      padding: 20px;
    }

    // PVC 状态概览
    .pvc-overview {
      margin-bottom: 20px;

      .overview-title {
        font-size: 13px;
        font-weight: 600;
        color: $gray-600;
        margin-bottom: 14px;
      }

      .status-grid {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 14px;

        @media (max-width: 768px) {
          grid-template-columns: 1fr;
        }
      }

      .status-item {
        display: flex;
        flex-direction: column;
        padding: 16px;
        background: $gray-50;
        border-radius: $radius-sm;
        border: 1px solid $gray-200;
        border-left: 4px solid;
        transition: all 0.2s ease;

        &:hover {
          border-color: $gray-300;
        }

        &.total {
          border-left-color: $primary;
          .status-icon {
            color: $primary;
          }
          .status-progress .progress-fill {
            background: $primary;
          }
        }

        &.bound {
          border-left-color: $success;
          .status-icon {
            color: $success;
          }
          .status-progress .progress-fill {
            background: $success;
          }
        }

        &.pending {
          border-left-color: $warning;
          .status-icon {
            color: $warning;
          }
          .status-progress .progress-fill {
            background: $warning;
          }
        }

        .status-icon {
          margin-bottom: 10px;
        }

        .status-info {
          display: flex;
          flex-direction: column;
          gap: 4px;

          .status-label {
            font-size: 12px;
            color: $gray-500;
          }

          .status-value {
            font-size: 24px;
            font-weight: 700;
            color: $gray-800;
          }
        }

        .status-progress {
          margin-top: 12px;
          height: 4px;
          background: $gray-200;
          border-radius: 2px;
          overflow: hidden;

          .progress-fill {
            height: 100%;
            transition: width 0.4s ease;
          }
        }
      }
    }

    // 容量统计
    .capacity-stats {
      margin-bottom: 20px;

      .stats-title {
        font-size: 13px;
        font-weight: 600;
        color: $gray-600;
        margin-bottom: 14px;
      }

      .capacity-card {
        display: flex;
        align-items: center;
        gap: 16px;
        padding: 20px;
        background: linear-gradient(135deg, $gray-50 0%, #fff 100%);
        border-radius: $radius-sm;
        border: 1px solid $gray-200;

        .capacity-icon {
          width: 56px;
          height: 56px;
          border-radius: $radius-sm;
          background: linear-gradient(135deg, #dbeafe, #bfdbfe);
          display: flex;
          align-items: center;
          justify-content: center;
          color: $primary;
        }

        .capacity-info {
          display: flex;
          flex-direction: column;
          gap: 4px;

          .capacity-label {
            font-size: 12px;
            color: $gray-500;
          }

          .capacity-value {
            font-size: 28px;
            font-weight: 700;
            color: $primary;
          }
        }
      }
    }

    // 存储类统计
    .storage-classes {
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

      .storage-class-list {
        display: flex;
        flex-direction: column;
        gap: 12px;

        .storage-class-item {
          display: flex;
          align-items: center;
          gap: 14px;
          padding: 16px;
          background: $gray-50;
          border-radius: $radius-sm;
          border: 1px solid $gray-200;
          transition: all 0.2s ease;

          &:hover {
            border-color: $gray-300;
            background: #fff;
          }

          .sc-index {
            width: 28px;
            height: 28px;
            border-radius: 50%;
            background: linear-gradient(135deg, $primary, #2563eb);
            color: #fff;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 12px;
            font-weight: 600;
            flex-shrink: 0;
          }

          .sc-content {
            flex: 1;
            min-width: 0;

            .sc-main {
              display: flex;
              justify-content: space-between;
              align-items: center;
              margin-bottom: 8px;

              .sc-name {
                font-size: 14px;
                font-weight: 600;
                color: $gray-700;
                overflow: hidden;
                text-overflow: ellipsis;
                white-space: nowrap;
              }

              .sc-size {
                font-size: 16px;
                font-weight: 700;
                color: $primary;
                flex-shrink: 0;
                margin-left: 12px;
              }
            }

            .sc-meta {
              display: flex;
              align-items: center;
              gap: 12px;

              .sc-count {
                font-size: 12px;
                color: $gray-500;
                flex-shrink: 0;
              }

              .sc-bar {
                flex: 1;
                height: 4px;
                background: $gray-200;
                border-radius: 2px;
                overflow: hidden;

                .sc-bar-fill {
                  height: 100%;
                  background: linear-gradient(90deg, $primary, #60a5fa);
                  transition: width 0.4s ease;
                }
              }
            }
          }
        }
      }
    }

    // 无存储类数据提示
    .no-storage-class {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 10px;
      padding: 30px;
      background: $gray-50;
      border-radius: $radius-sm;
      color: $gray-400;
      font-size: 14px;
    }

    // 空状态
    .empty-state {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      min-height: 350px;
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
