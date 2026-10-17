<template>
  <div class="workload-metrics" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="monitor-card">
        <!-- 卡片头部 -->
        <div class="card-header">
          <div class="header-info">
            <div class="icon-badge workload">
              <svg viewBox="0 0 24 24" width="22" height="22">
                <path
                  fill="currentColor"
                  d="M19.14,12.94c0.04-0.3,0.06-0.61,0.06-0.94c0-0.32-0.02-0.64-0.07-0.94l2.03-1.58c0.18-0.14,0.23-0.41,0.12-0.61 l-1.92-3.32c-0.12-0.22-0.37-0.29-0.59-0.22l-2.39,0.96c-0.5-0.38-1.03-0.7-1.62-0.94L14.4,2.81c-0.04-0.24-0.24-0.41-0.48-0.41 h-3.84c-0.24,0-0.43,0.17-0.47,0.41L9.25,5.35C8.66,5.59,8.12,5.92,7.63,6.29L5.24,5.33c-0.22-0.08-0.47,0-0.59,0.22L2.74,8.87 C2.62,9.08,2.66,9.34,2.86,9.48l2.03,1.58C4.84,11.36,4.8,11.69,4.8,12s0.02,0.64,0.07,0.94l-2.03,1.58 c-0.18,0.14-0.23,0.41-0.12,0.61l1.92,3.32c0.12,0.22,0.37,0.29,0.59,0.22l2.39-0.96c0.5,0.38,1.03,0.7,1.62,0.94l0.36,2.54 c0.05,0.24,0.24,0.41,0.48,0.41h3.84c0.24,0,0.44-0.17,0.47-0.41l0.36-2.54c0.59-0.24,1.13-0.56,1.62-0.94l2.39,0.96 c0.22,0.08,0.47,0,0.59-0.22l1.92-3.32c0.12-0.22,0.07-0.47-0.12-0.61L19.14,12.94z M12,15.6c-1.98,0-3.6-1.62-3.6-3.6 s1.62-3.6,3.6-3.6s3.6,1.62,3.6,3.6S13.98,15.6,12,15.6z"
                />
              </svg>
            </div>
            <div>
              <h3>工作负载统计</h3>
              <p>Deployments, StatefulSets, DaemonSets, Jobs</p>
            </div>
          </div>
          <div class="header-stats">
            <div class="stat-badge total">
              <span class="label">总工作负载</span>
              <span class="value">{{ totalWorkloads }}</span>
            </div>
            <div class="stat-badge running">
              <span class="label">可用</span>
              <span class="value">{{ totalAvailable }}</span>
            </div>
            <div class="stat-badge warning" v-if="totalUnavailable > 0">
              <span class="label">不可用</span>
              <span class="value">{{ totalUnavailable }}</span>
            </div>
          </div>
        </div>

        <!-- 内容区域 -->
        <div class="content-wrapper">
          <!-- Deployments 区块 -->
          <div class="workload-section">
            <div class="section-header">
              <div class="section-icon deployment">
                <svg viewBox="0 0 24 24" width="18" height="18">
                  <path
                    fill="currentColor"
                    d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14h-2V9h2v8zm0-10h-2V5h2v2z"
                  />
                </svg>
              </div>
              <span class="section-title">Deployments</span>
              <span class="section-count">{{ workloadData?.deployments?.total || 0 }} 个</span>
            </div>
            <div class="metrics-grid">
              <div class="metric-box">
                <div class="box-header">
                  <span>总数</span>
                </div>
                <div class="box-value">
                  <span class="value">{{ workloadData?.deployments?.total || 0 }}</span>
                </div>
              </div>
              <div class="metric-box">
                <div class="box-header">
                  <span>期望副本</span>
                </div>
                <div class="box-value">
                  <span class="value">{{ workloadData?.deployments?.desiredReplicas || 0 }}</span>
                </div>
              </div>
              <div class="metric-box success">
                <div class="box-header">
                  <span>可用副本</span>
                </div>
                <div class="box-value">
                  <span class="value success">{{
                    workloadData?.deployments?.availableReplicas || 0
                  }}</span>
                </div>
              </div>
              <div
                class="metric-box"
                :class="{ danger: (workloadData?.deployments?.unavailableReplicas || 0) > 0 }"
              >
                <div class="box-header">
                  <span>不可用副本</span>
                </div>
                <div class="box-value">
                  <span
                    class="value"
                    :class="{ danger: (workloadData?.deployments?.unavailableReplicas || 0) > 0 }"
                  >
                    {{ workloadData?.deployments?.unavailableReplicas || 0 }}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <!-- StatefulSets 区块 -->
          <div class="workload-section">
            <div class="section-header">
              <div class="section-icon statefulset">
                <svg viewBox="0 0 24 24" width="18" height="18">
                  <path
                    fill="currentColor"
                    d="M2 20h20v-4H2v4zm2-3h2v2H4v-2zM2 4v4h20V4H2zm4 3H4V5h2v2zm-4 7h20v-4H2v4zm2-3h2v2H4v-2z"
                  />
                </svg>
              </div>
              <span class="section-title">StatefulSets</span>
              <span class="section-count">{{ workloadData?.statefulSets?.total || 0 }} 个</span>
            </div>
            <div class="metrics-grid three-cols">
              <div class="metric-box">
                <div class="box-header">
                  <span>总数</span>
                </div>
                <div class="box-value">
                  <span class="value">{{ workloadData?.statefulSets?.total || 0 }}</span>
                </div>
              </div>
              <div class="metric-box">
                <div class="box-header">
                  <span>期望副本</span>
                </div>
                <div class="box-value">
                  <span class="value">{{ workloadData?.statefulSets?.desiredReplicas || 0 }}</span>
                </div>
              </div>
              <div class="metric-box success">
                <div class="box-header">
                  <span>就绪副本</span>
                </div>
                <div class="box-value">
                  <span class="value success">{{
                    workloadData?.statefulSets?.readyReplicas || 0
                  }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- DaemonSets 区块 -->
          <div class="workload-section">
            <div class="section-header">
              <div class="section-icon daemonset">
                <svg viewBox="0 0 24 24" width="18" height="18">
                  <path
                    fill="currentColor"
                    d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93s3.05-7.44 7-7.93V2.05c-5.05.5-9 4.76-9 9.95s3.95 9.45 9 9.95v-2.02z"
                  />
                </svg>
              </div>
              <span class="section-title">DaemonSets</span>
              <span class="section-count">{{ workloadData?.daemonSets?.total || 0 }} 个</span>
            </div>
            <div class="metrics-grid three-cols">
              <div class="metric-box">
                <div class="box-header">
                  <span>总数</span>
                </div>
                <div class="box-value">
                  <span class="value">{{ workloadData?.daemonSets?.total || 0 }}</span>
                </div>
              </div>
              <div class="metric-box">
                <div class="box-header">
                  <span>期望调度</span>
                </div>
                <div class="box-value">
                  <span class="value">{{ workloadData?.daemonSets?.desiredScheduled || 0 }}</span>
                </div>
              </div>
              <div class="metric-box success">
                <div class="box-header">
                  <span>当前调度</span>
                </div>
                <div class="box-value">
                  <span class="value success">{{
                    workloadData?.daemonSets?.currentScheduled || 0
                  }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- Jobs 区块 -->
          <div class="workload-section">
            <div class="section-header">
              <div class="section-icon job">
                <svg viewBox="0 0 24 24" width="18" height="18">
                  <path
                    fill="currentColor"
                    d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-5 14H7v-2h7v2zm3-4H7v-2h10v2zm0-4H7V7h10v2z"
                  />
                </svg>
              </div>
              <span class="section-title">Jobs</span>
              <span class="section-count"
                >{{
                  (workloadData?.jobs?.active || 0) +
                  (workloadData?.jobs?.succeeded || 0) +
                  (workloadData?.jobs?.failed || 0)
                }}
                个</span
              >
            </div>
            <div class="metrics-grid">
              <div class="metric-box">
                <div class="box-header">
                  <span>活跃</span>
                </div>
                <div class="box-value">
                  <span class="value">{{ workloadData?.jobs?.active || 0 }}</span>
                </div>
              </div>
              <div class="metric-box success">
                <div class="box-header">
                  <span>成功</span>
                </div>
                <div class="box-value">
                  <span class="value success">{{ workloadData?.jobs?.succeeded || 0 }}</span>
                </div>
              </div>
              <div class="metric-box" :class="{ danger: (workloadData?.jobs?.failed || 0) > 0 }">
                <div class="box-header">
                  <span>失败</span>
                </div>
                <div class="box-value">
                  <span class="value" :class="{ danger: (workloadData?.jobs?.failed || 0) > 0 }">
                    {{ workloadData?.jobs?.failed || 0 }}
                  </span>
                </div>
              </div>
              <div class="metric-box">
                <div class="box-header">
                  <span>CronJobs</span>
                </div>
                <div class="box-value">
                  <span class="value">{{ workloadData?.jobs?.cronJobsTotal || 0 }}</span>
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
              d="M19.14,12.94c0.04-0.3,0.06-0.61,0.06-0.94c0-0.32-0.02-0.64-0.07-0.94l2.03-1.58c0.18-0.14,0.23-0.41,0.12-0.61 l-1.92-3.32c-0.12-0.22-0.37-0.29-0.59-0.22l-2.39,0.96c-0.5-0.38-1.03-0.7-1.62-0.94L14.4,2.81c-0.04-0.24-0.24-0.41-0.48-0.41 h-3.84c-0.24,0-0.43,0.17-0.47,0.41L9.25,5.35C8.66,5.59,8.12,5.92,7.63,6.29L5.24,5.33c-0.22-0.08-0.47,0-0.59,0.22L2.74,8.87 C2.62,9.08,2.66,9.34,2.86,9.48l2.03,1.58C4.84,11.36,4.8,11.69,4.8,12s0.02,0.64,0.07,0.94l-2.03,1.58 c-0.18,0.14-0.23,0.41-0.12,0.61l1.92,3.32c0.12,0.22,0.37,0.29,0.59,0.22l2.39-0.96c0.5,0.38,1.03,0.7,1.62,0.94l0.36,2.54 c0.05,0.24,0.24,0.41,0.48,0.41h3.84c0.24,0,0.44-0.17,0.47-0.41l0.36-2.54c0.59-0.24,1.13-0.56,1.62-0.94l2.39,0.96 c0.22,0.08,0.47,0,0.59-0.22l1.92-3.32c0.12-0.22,0.07-0.47-0.12-0.61L19.14,12.94z"
            />
          </svg>
        </div>
        <h3>暂无工作负载数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import type { NamespaceWorkloadMetrics } from '@/api/console/namespace-monitor'

  interface Props {
    workloadData?: NamespaceWorkloadMetrics
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    workloadData: undefined,
    loading: false
  })

  // 判断是否有数据
  const hasData = computed(() => {
    return !!(props.workloadData && Object.keys(props.workloadData).length > 0)
  })

  // 计算总工作负载数量
  const totalWorkloads = computed(() => {
    if (!props.workloadData) return 0
    return (
      (props.workloadData.deployments?.total || 0) +
      (props.workloadData.statefulSets?.total || 0) +
      (props.workloadData.daemonSets?.total || 0)
    )
  })

  // 计算总可用数量
  const totalAvailable = computed(() => {
    if (!props.workloadData) return 0
    return (
      (props.workloadData.deployments?.availableReplicas || 0) +
      (props.workloadData.statefulSets?.readyReplicas || 0) +
      (props.workloadData.daemonSets?.currentScheduled || 0)
    )
  })

  // 计算总不可用数量
  const totalUnavailable = computed(() => {
    if (!props.workloadData) return 0
    return props.workloadData.deployments?.unavailableReplicas || 0
  })
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

  .workload-metrics {
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
      background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
      border-bottom: 1px solid #fcd34d;
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

          &.workload {
            background: linear-gradient(135deg, $warning, #d97706);
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
            background: rgba(255, 255, 255, 0.9);
            color: #92400e;
          }

          &.running {
            background: linear-gradient(135deg, #d1fae5 0%, #a7f3d0 100%);
            color: #065f46;
          }

          &.warning {
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

    // 工作负载区块
    .workload-section {
      margin-bottom: 20px;
      padding: 18px;
      background: $gray-50;
      border-radius: $radius-sm;
      border: 1px solid $gray-200;

      &:last-child {
        margin-bottom: 0;
      }

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
          display: flex;
          align-items: center;
          justify-content: center;

          &.deployment {
            background: linear-gradient(135deg, #dbeafe, #bfdbfe);
            color: $primary;
          }

          &.statefulset {
            background: linear-gradient(135deg, #d1fae5, #a7f3d0);
            color: $success;
          }

          &.daemonset {
            background: linear-gradient(135deg, #fef3c7, #fde68a);
            color: $warning;
          }

          &.job {
            background: linear-gradient(135deg, #e5e7eb, #d1d5db);
            color: $gray-600;
          }
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
          background: #fff;
          padding: 4px 10px;
          border-radius: 12px;
        }
      }

      // 指标网格
      .metrics-grid {
        display: grid;
        grid-template-columns: repeat(4, 1fr);
        gap: 12px;

        &.three-cols {
          grid-template-columns: repeat(3, 1fr);
        }

        @media (max-width: 1200px) {
          grid-template-columns: repeat(2, 1fr);
        }

        @media (max-width: 640px) {
          grid-template-columns: 1fr;
        }
      }

      .metric-box {
        padding: 14px;
        border-radius: $radius-xs;
        background: #fff;
        border: 1px solid $gray-200;
        transition: all 0.2s ease;

        &:hover {
          border-color: $gray-300;
        }

        &.success {
          background: rgba($success, 0.04);
          border-color: rgba($success, 0.15);
        }

        &.danger {
          background: rgba($danger, 0.06);
          border-color: rgba($danger, 0.25);
        }

        .box-header {
          margin-bottom: 8px;

          span {
            font-size: 12px;
            font-weight: 500;
            color: $gray-500;
          }
        }

        .box-value {
          .value {
            font-size: 22px;
            font-weight: 700;
            color: $gray-800;

            &.success {
              color: $success;
            }

            &.danger {
              color: $danger;
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
