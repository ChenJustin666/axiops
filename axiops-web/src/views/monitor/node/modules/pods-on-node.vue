<template>
  <div class="pods-on-node" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="monitor-card">
        <!-- 卡片头部 -->
        <div class="card-header">
          <div class="header-info">
            <div class="icon-badge pods">
              <svg viewBox="0 0 24 24" width="22" height="22">
                <path
                  fill="currentColor"
                  d="M21,16.5C21,16.88 20.79,17.21 20.47,17.38L12.57,21.82C12.41,21.94 12.21,22 12,22C11.79,22 11.59,21.94 11.43,21.82L3.53,17.38C3.21,17.21 3,16.88 3,16.5V7.5C3,7.12 3.21,6.79 3.53,6.62L11.43,2.18C11.59,2.06 11.79,2 12,2C12.21,2 12.41,2.06 12.57,2.18L20.47,6.62C20.79,6.79 21,7.12 21,7.5V16.5M12,4.15L6.04,7.5L12,10.85L17.96,7.5L12,4.15M5,15.91L11,19.29V12.58L5,9.21V15.91M19,15.91V9.21L13,12.58V19.29L19,15.91Z"
                />
              </svg>
            </div>
            <div>
              <h3>Node 上的 Pod</h3>
              <p>{{ podsData?.nodeName }} - {{ podsData?.totalPods || 0 }} 个 Pod</p>
            </div>
          </div>
          <div class="header-stats">
            <span class="stat-badge success">Running: {{ podsData?.runningPods || 0 }}</span>
            <span v-if="podsData?.pendingPods" class="stat-badge warning"
              >Pending: {{ podsData.pendingPods }}</span
            >
            <span v-if="podsData?.failedPods" class="stat-badge danger"
              >Failed: {{ podsData.failedPods }}</span
            >
          </div>
        </div>

        <!-- TOP 排行区域 -->
        <div class="top-section">
          <div class="top-grid">
            <!-- CPU TOP -->
            <div class="top-card">
              <div class="top-header">
                <div class="top-icon cpu">
                  <svg viewBox="0 0 24 24" width="18" height="18">
                    <path
                      fill="currentColor"
                      d="M17,17H7V7H17M21,11V9H19V7C19,5.89 18.1,5 17,5H15V3H13V5H11V3H9V5H7C5.89,5 5,5.89 5,7V9H3V11H5V13H3V15H5V17A2,2 0 0,0 7,19H9V21H11V19H13V21H15V19H17A2,2 0 0,0 19,17V15H21V13H19V11M15,15H9V9H15V15Z"
                    />
                  </svg>
                </div>
                <span class="top-title"
                  >CPU 使用 TOP {{ podsData?.topPodsByCPU?.length || 0 }}</span
                >
              </div>
              <div class="top-list">
                <div
                  v-for="(pod, index) in podsData?.topPodsByCPU"
                  :key="'cpu-' + index"
                  class="top-item"
                >
                  <div class="rank" :class="getRankClass(index)">{{ index + 1 }}</div>
                  <div class="pod-info">
                    <span class="pod-name">{{ pod.podName }}</span>
                    <span class="pod-namespace">{{ pod.namespace }}</span>
                  </div>
                  <div class="pod-value cpu">{{ pod.value.toFixed(3) }} {{ pod.unit }}</div>
                </div>
                <div v-if="!podsData?.topPodsByCPU?.length" class="no-data">暂无数据</div>
              </div>
            </div>

            <!-- Memory TOP -->
            <div class="top-card">
              <div class="top-header">
                <div class="top-icon memory">
                  <svg viewBox="0 0 24 24" width="18" height="18">
                    <path
                      fill="currentColor"
                      d="M17,17H7V7H17M21,11V9H19V7C19,5.89 18.1,5 17,5H15V3H13V5H11V3H9V5H7C5.89,5 5,5.89 5,7V9H3V11H5V13H3V15H5V17A2,2 0 0,0 7,19H9V21H11V19H13V21H15V19H17A2,2 0 0,0 19,17V15H21V13H19V11M13,13H11V11H13M15,9H9V15H15V9Z"
                    />
                  </svg>
                </div>
                <span class="top-title"
                  >内存使用 TOP {{ podsData?.topPodsByMem?.length || 0 }}</span
                >
              </div>
              <div class="top-list">
                <div
                  v-for="(pod, index) in podsData?.topPodsByMem"
                  :key="'mem-' + index"
                  class="top-item"
                >
                  <div class="rank" :class="getRankClass(index)">{{ index + 1 }}</div>
                  <div class="pod-info">
                    <span class="pod-name">{{ pod.podName }}</span>
                    <span class="pod-namespace">{{ pod.namespace }}</span>
                  </div>
                  <div class="pod-value memory">{{ formatBytes(pod.value) }}</div>
                </div>
                <div v-if="!podsData?.topPodsByMem?.length" class="no-data">暂无数据</div>
              </div>
            </div>
          </div>
        </div>

        <!-- Pod 列表区域 -->
        <div class="pods-list-section">
          <div class="section-header">
            <div class="section-title">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path fill="currentColor" d="M3,4H21V8H3V4M3,10H21V14H3V10M3,16H21V20H3V16Z" />
              </svg>
              <span>Pod 列表</span>
            </div>
            <div class="section-actions">
              <div class="search-box">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path
                    fill="currentColor"
                    d="M9.5,3A6.5,6.5 0 0,1 16,9.5C16,11.11 15.41,12.59 14.44,13.73L14.71,14H15.5L20.5,19L19,20.5L14,15.5V14.71L13.73,14.44C12.59,15.41 11.11,16 9.5,16A6.5,6.5 0 0,1 3,9.5A6.5,6.5 0 0,1 9.5,3M9.5,5C7,5 5,7 5,9.5C5,12 7,14 9.5,14C12,14 14,12 14,9.5C14,7 12,5 9.5,5Z"
                  />
                </svg>
                <input v-model="searchQuery" type="text" placeholder="搜索 Pod..." />
                <button v-if="searchQuery" class="clear-btn" @click="clearSearch">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M19,6.41L17.59,5L12,10.59L6.41,5L5,6.41L10.59,12L5,17.59L6.41,19L12,13.41L17.59,19L19,17.59L13.41,12L19,6.41Z"
                    />
                  </svg>
                </button>
              </div>
              <div class="filter-select">
                <select v-model="phaseFilter">
                  <option value="">全部状态</option>
                  <option value="Running">Running</option>
                  <option value="Pending">Pending</option>
                  <option value="Failed">Failed</option>
                  <option value="Succeeded">Succeeded</option>
                </select>
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path
                    fill="currentColor"
                    d="M7.41,8.58L12,13.17L16.59,8.58L18,10L12,16L6,10L7.41,8.58Z"
                  />
                </svg>
              </div>
            </div>
          </div>

          <!-- Pod 表格 -->
          <div class="table-wrapper">
            <table class="pods-table">
              <thead>
                <tr>
                  <th>Namespace</th>
                  <th>Pod Name</th>
                  <th>Phase</th>
                  <th>CPU</th>
                  <th>Memory</th>
                  <th>Restarts</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="pod in filteredPods" :key="pod.podName" @click="handlePodClick(pod)">
                  <td>
                    <span class="namespace-tag">{{ pod.namespace }}</span>
                  </td>
                  <td>
                    <span class="pod-link">{{ pod.podName }}</span>
                  </td>
                  <td>
                    <span class="phase-badge" :class="getPhaseClass(pod.phase)">{{
                      pod.phase
                    }}</span>
                  </td>
                  <td>
                    <span class="resource-value">{{ pod.cpuUsage.toFixed(3) }} cores</span>
                  </td>
                  <td>
                    <span class="resource-value">{{ formatBytes(pod.memoryUsage) }}</span>
                  </td>
                  <td>
                    <span class="restart-badge" :class="{ warning: pod.restartCount > 0 }">
                      {{ pod.restartCount }}
                    </span>
                  </td>
                </tr>
                <tr v-if="filteredPods.length === 0">
                  <td colspan="6" class="no-data-row">
                    <div class="no-data-content">
                      <svg viewBox="0 0 24 24" width="32" height="32">
                        <path
                          fill="currentColor"
                          d="M19,3H5C3.89,3 3,3.89 3,5V19A2,2 0 0,0 5,21H19A2,2 0 0,0 21,19V5C21,3.89 20.1,3 19,3M19,5V19H5V5H19Z"
                        />
                      </svg>
                      <span>没有匹配的 Pod</span>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" width="64" height="64">
            <path
              fill="currentColor"
              d="M21,16.5C21,16.88 20.79,17.21 20.47,17.38L12.57,21.82C12.41,21.94 12.21,22 12,22C11.79,22 11.59,21.94 11.43,21.82L3.53,17.38C3.21,17.21 3,16.88 3,16.5V7.5C3,7.12 3.21,6.79 3.53,6.62L11.43,2.18C11.59,2.06 11.79,2 12,2C12.21,2 12.41,2.06 12.57,2.18L20.47,6.62C20.79,6.79 21,7.12 21,7.5V16.5Z"
            />
          </svg>
        </div>
        <h3>暂无 Pod 数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { ref, computed } from 'vue'
  import { useRouter } from 'vue-router'
  import { formatBytes } from '@/utils/format'
  import type { NodePodMetrics, NodePodBrief } from '@/api/console/node-monitor'

  interface Props {
    podsData?: NodePodMetrics
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    podsData: undefined,
    loading: false
  })

  const router = useRouter()
  // 搜索关键词
  const searchQuery = ref('')
  // 状态过滤
  const phaseFilter = ref('')

  // 是否有数据
  const hasData = computed(() => {
    return !!(props.podsData && Object.keys(props.podsData).length > 0)
  })

  // 过滤后的 Pod 列表
  const filteredPods = computed(() => {
    if (!props.podsData?.podList) return []

    let pods = props.podsData.podList

    // 搜索过滤
    if (searchQuery.value) {
      const query = searchQuery.value.toLowerCase()
      pods = pods.filter(
        (pod) =>
          pod.podName.toLowerCase().includes(query) || pod.namespace.toLowerCase().includes(query)
      )
    }

    // 状态过滤
    if (phaseFilter.value) {
      pods = pods.filter((pod) => pod.phase === phaseFilter.value)
    }

    return pods
  })

  // 获取排名样式
  const getRankClass = (index: number): string => {
    if (index === 0) return 'gold'
    if (index === 1) return 'silver'
    if (index === 2) return 'bronze'
    return ''
  }

  // 获取 Phase 样式
  const getPhaseClass = (phase: string): string => {
    switch (phase) {
      case 'Running':
        return 'success'
      case 'Pending':
        return 'warning'
      case 'Failed':
        return 'danger'
      case 'Succeeded':
        return 'info'
      default:
        return ''
    }
  }

  // 清除搜索
  const clearSearch = () => {
    searchQuery.value = ''
  }

  // Pod 点击处理
  const handlePodClick = (pod: NodePodBrief) => {
    router.push({
      path: '/monitor/pod',
      query: {
        clusterUuid: router.currentRoute.value.query.clusterUuid,
        namespace: pod.namespace,
        podName: pod.podName
      }
    })
  }
</script>

<style lang="scss" scoped>
  @use "sass:color";

  // 颜色变量
  $primary: #3b82f6;
  $success: #10b981;
  $warning: #f59e0b;
  $danger: #ef4444;
  $info: #0ea5e9;
  $purple: #8b5cf6;
  $pink: #ec4899;
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

  .pods-on-node {
    .monitor-card {
      background: #fff;
      border-radius: $radius;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
      overflow: hidden;
    }

    // 卡片头部
    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 20px 24px;
      border-bottom: 1px solid $gray-200;
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

          &.pods {
            background: linear-gradient(135deg, $purple, #7c3aed);
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
        }
      }
    }

    // TOP 排行区域
    .top-section {
      padding: 20px 24px;
      background: $gray-50;
      border-bottom: 1px solid $gray-200;

      .top-grid {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 16px;

        @media (max-width: 768px) {
          grid-template-columns: 1fr;
        }
      }

      .top-card {
        background: #fff;
        border-radius: $radius-sm;
        border: 1px solid $gray-200;
        overflow: hidden;

        .top-header {
          display: flex;
          align-items: center;
          gap: 10px;
          padding: 14px 16px;
          background: $gray-50;
          border-bottom: 1px solid $gray-200;

          .top-icon {
            width: 32px;
            height: 32px;
            border-radius: $radius-xs;
            display: flex;
            align-items: center;
            justify-content: center;

            &.cpu {
              background: linear-gradient(135deg, rgba($purple, 0.15), rgba($purple, 0.08));
              color: $purple;
            }

            &.memory {
              background: linear-gradient(135deg, rgba($pink, 0.15), rgba($pink, 0.08));
              color: $pink;
            }
          }

          .top-title {
            font-size: 13px;
            font-weight: 600;
            color: $gray-700;
          }
        }

        .top-list {
          padding: 8px;

          .top-item {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 10px 12px;
            border-radius: $radius-xs;
            transition: all 0.2s ease;

            &:hover {
              background: $gray-50;
            }

            .rank {
              width: 24px;
              height: 24px;
              border-radius: 50%;
              display: flex;
              align-items: center;
              justify-content: center;
              font-size: 11px;
              font-weight: 700;
              background: $gray-100;
              color: $gray-500;
              flex-shrink: 0;

              &.gold {
                background: linear-gradient(135deg, #fef3c7, #fde68a);
                color: #92400e;
              }

              &.silver {
                background: linear-gradient(135deg, #f3f4f6, #e5e7eb);
                color: $gray-600;
              }

              &.bronze {
                background: linear-gradient(135deg, #fed7aa, #fdba74);
                color: #9a3412;
              }
            }

            .pod-info {
              flex: 1;
              min-width: 0;
              display: flex;
              flex-direction: column;
              gap: 2px;

              .pod-name {
                font-size: 13px;
                font-weight: 600;
                color: $gray-800;
                overflow: hidden;
                text-overflow: ellipsis;
                white-space: nowrap;
              }

              .pod-namespace {
                font-size: 11px;
                color: $gray-400;
              }
            }

            .pod-value {
              font-size: 13px;
              font-weight: 700;
              flex-shrink: 0;

              &.cpu {
                color: $purple;
              }

              &.memory {
                color: $pink;
              }
            }
          }

          .no-data {
            padding: 20px;
            text-align: center;
            font-size: 13px;
            color: $gray-400;
          }
        }
      }
    }

    // Pod 列表区域
    .pods-list-section {
      padding: 20px 24px;

      .section-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 16px;
        flex-wrap: wrap;
        gap: 12px;

        .section-title {
          display: flex;
          align-items: center;
          gap: 8px;
          font-size: 14px;
          font-weight: 600;
          color: $gray-700;

          svg {
            color: $gray-400;
          }
        }

        .section-actions {
          display: flex;
          gap: 10px;
          flex-wrap: wrap;

          .search-box {
            position: relative;
            display: flex;
            align-items: center;

            > svg {
              position: absolute;
              left: 12px;
              color: $gray-400;
              pointer-events: none;
            }

            input {
              padding: 8px 32px 8px 36px;
              width: 220px;
              border: 1px solid $gray-200;
              border-radius: $radius-xs;
              font-size: 13px;
              color: $gray-700;
              background: $gray-50;
              transition: all 0.2s ease;

              &::placeholder {
                color: $gray-400;
              }

              &:hover {
                border-color: $gray-300;
              }

              &:focus {
                outline: none;
                border-color: $primary;
                background: #fff;
                box-shadow: 0 0 0 3px rgba($primary, 0.1);
              }
            }

            .clear-btn {
              position: absolute;
              right: 8px;
              padding: 4px;
              background: none;
              border: none;
              color: $gray-400;
              cursor: pointer;
              display: flex;
              align-items: center;
              justify-content: center;
              border-radius: 50%;
              transition: all 0.2s ease;

              &:hover {
                background: $gray-200;
                color: $gray-600;
              }
            }
          }

          .filter-select {
            position: relative;
            display: flex;
            align-items: center;

            select {
              appearance: none;
              padding: 8px 32px 8px 14px;
              background: $gray-50;
              border: 1px solid $gray-200;
              border-radius: $radius-xs;
              font-size: 13px;
              color: $gray-700;
              cursor: pointer;
              transition: all 0.2s ease;

              &:hover {
                border-color: $gray-300;
              }

              &:focus {
                outline: none;
                border-color: $primary;
                box-shadow: 0 0 0 3px rgba($primary, 0.1);
              }
            }

            svg {
              position: absolute;
              right: 10px;
              pointer-events: none;
              color: $gray-400;
            }
          }
        }
      }

      // 表格样式
      .table-wrapper {
        overflow-x: auto;
        border-radius: $radius-sm;
        border: 1px solid $gray-200;
      }

      .pods-table {
        width: 100%;
        border-collapse: collapse;
        font-size: 13px;

        th,
        td {
          padding: 12px 16px;
          text-align: left;
          border-bottom: 1px solid $gray-100;
        }

        thead {
          background: $gray-50;

          th {
            font-weight: 600;
            color: $gray-600;
            font-size: 12px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            border-bottom-color: $gray-200;
          }
        }

        tbody {
          tr {
            cursor: pointer;
            transition: background 0.2s ease;

            &:hover {
              background: rgba($primary, 0.04);
            }

            &:last-child td {
              border-bottom: none;
            }
          }
        }

        .namespace-tag {
          display: inline-block;
          padding: 2px 8px;
          background: $gray-100;
          border-radius: 4px;
          font-size: 11px;
          color: $gray-600;
        }

        .pod-link {
          color: $primary;
          font-weight: 500;
          transition: color 0.2s ease;

          &:hover {
            color: color.adjust($primary, $lightness: -10%);
            text-decoration: underline;
          }
        }

        .phase-badge {
          display: inline-block;
          padding: 3px 10px;
          border-radius: 12px;
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
            background: rgba($info, 0.1);
            color: #0284c7;
          }
        }

        .resource-value {
          font-family: var(--ax-font-mono);
          font-size: 12px;
          color: $gray-700;
        }

        .restart-badge {
          display: inline-flex;
          align-items: center;
          justify-content: center;
          min-width: 24px;
          padding: 2px 8px;
          background: $gray-100;
          border-radius: 10px;
          font-size: 12px;
          font-weight: 600;
          color: $gray-600;

          &.warning {
            background: rgba($warning, 0.1);
            color: #d97706;
          }
        }

        .no-data-row {
          td {
            padding: 40px;
          }

          .no-data-content {
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 8px;
            color: $gray-400;

            svg {
              color: $gray-300;
            }

            span {
              font-size: 13px;
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

  // 动画
  .fade-enter-active,
  .fade-leave-active {
    transition: opacity 0.3s ease;
  }

  .fade-enter-from,
  .fade-leave-to {
    opacity: 0;
  }

  // 响应式适配
  @media (max-width: 768px) {
    .pods-on-node {
      .card-header {
        flex-direction: column;
        align-items: flex-start;
      }

      .pods-list-section {
        padding: 16px;

        .section-header {
          flex-direction: column;
          align-items: flex-start;

          .section-actions {
            width: 100%;

            .search-box {
              flex: 1;

              input {
                width: 100%;
              }
            }
          }
        }
      }
    }
  }
</style>
