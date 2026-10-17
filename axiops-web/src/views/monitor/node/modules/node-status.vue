<template>
  <div class="node-status-monitor" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="monitor-card">
        <!-- 卡片头部区域 -->
        <div class="card-header">
          <div class="header-info">
            <div class="icon-badge status">
              <svg viewBox="0 0 24 24" width="22" height="22">
                <path
                  fill="currentColor"
                  d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"
                />
              </svg>
            </div>
            <div>
              <h3>Node 状态</h3>
              <p>Kubernetes 节点信息</p>
            </div>
          </div>
          <div class="status-tag" :class="getReadyStatusClass()">{{ getReadyStatusText() }}</div>
        </div>

        <div class="content-wrapper">
          <!-- 双栏布局：节点状况 + 资源容量 -->
          <div class="two-column-grid">
            <!-- 节点状况区域 -->
            <div class="status-section">
              <div class="section-header">
                <svg viewBox="0 0 24 24" width="18" height="18">
                  <path
                    fill="currentColor"
                    d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"
                  />
                </svg>
                <span>节点状况</span>
              </div>
              <div class="conditions-list">
                <div
                  v-for="condition in statusData?.conditions"
                  :key="condition.type"
                  class="condition-item"
                  :class="{ warning: shouldWarn(condition) }"
                >
                  <div class="condition-header">
                    <span class="condition-type">{{ condition.type }}</span>
                    <span class="condition-badge" :class="condition.status ? 'success' : 'info'">
                      {{ condition.status ? 'True' : 'False' }}
                    </span>
                  </div>
                  <div v-if="condition.reason" class="condition-reason">
                    {{ condition.reason }}
                  </div>
                  <div v-if="condition.message" class="condition-message">
                    {{ condition.message }}
                  </div>
                  <div class="condition-time">
                    Last Transition: {{ smartFormatTime(condition.lastTransitionTime) }}
                  </div>
                </div>
              </div>
            </div>

            <!-- 资源容量区域 -->
            <div class="capacity-section">
              <div class="section-header">
                <svg viewBox="0 0 24 24" width="18" height="18">
                  <path
                    fill="currentColor"
                    d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14h-2V9h2v8zm0-10h-2V5h2v2z"
                  />
                </svg>
                <span>资源容量</span>
              </div>
              <div class="capacity-grid">
                <!-- CPU 资源 -->
                <div class="capacity-item">
                  <div class="capacity-label">CPU</div>
                  <div class="capacity-bars">
                    <div class="bar-row">
                      <span class="bar-label">Capacity</span>
                      <div class="bar-wrapper">
                        <div class="bar capacity-bar">
                          <span class="bar-text">{{ statusData?.capacity?.cpuCores }} cores</span>
                        </div>
                      </div>
                    </div>
                    <div class="bar-row">
                      <span class="bar-label">Allocatable</span>
                      <div class="bar-wrapper">
                        <div
                          class="bar allocatable-bar"
                          :style="{ width: `${getAllocatablePercent('cpu')}%` }"
                        >
                          <span class="bar-text"
                            >{{ statusData?.allocatable?.cpuCores }} cores</span
                          >
                        </div>
                      </div>
                    </div>
                    <div class="bar-row">
                      <span class="bar-label">Allocated</span>
                      <div class="bar-wrapper">
                        <div
                          class="bar allocated-bar"
                          :style="{ width: `${getAllocatedPercent('cpu')}%` }"
                        >
                          <span class="bar-text"
                            >{{ statusData?.allocated?.cpuCores?.toFixed(2) }} cores</span
                          >
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- 内存资源 -->
                <div class="capacity-item">
                  <div class="capacity-label">Memory</div>
                  <div class="capacity-bars">
                    <div class="bar-row">
                      <span class="bar-label">Capacity</span>
                      <div class="bar-wrapper">
                        <div class="bar capacity-bar">
                          <span class="bar-text">{{
                            formatBytes(statusData?.capacity?.memoryBytes)
                          }}</span>
                        </div>
                      </div>
                    </div>
                    <div class="bar-row">
                      <span class="bar-label">Allocatable</span>
                      <div class="bar-wrapper">
                        <div
                          class="bar allocatable-bar"
                          :style="{ width: `${getAllocatablePercent('memory')}%` }"
                        >
                          <span class="bar-text">{{
                            formatBytes(statusData?.allocatable?.memoryBytes)
                          }}</span>
                        </div>
                      </div>
                    </div>
                    <div class="bar-row">
                      <span class="bar-label">Allocated</span>
                      <div class="bar-wrapper">
                        <div
                          class="bar allocated-bar"
                          :style="{ width: `${getAllocatedPercent('memory')}%` }"
                        >
                          <span class="bar-text">{{
                            formatBytes(statusData?.allocated?.memoryBytes)
                          }}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- Pod 资源 -->
                <div class="capacity-item">
                  <div class="capacity-label">Pods</div>
                  <div class="capacity-bars">
                    <div class="bar-row">
                      <span class="bar-label">Capacity</span>
                      <div class="bar-wrapper">
                        <div class="bar capacity-bar">
                          <span class="bar-text">{{ statusData?.capacity?.pods }}</span>
                        </div>
                      </div>
                    </div>
                    <div class="bar-row">
                      <span class="bar-label">Allocated</span>
                      <div class="bar-wrapper">
                        <div
                          class="bar allocated-bar"
                          :style="{ width: `${getAllocatedPercent('pods')}%` }"
                        >
                          <span class="bar-text">{{ statusData?.allocated?.pods }}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 节点信息区域 -->
          <div class="node-info-section">
            <div class="section-header">
              <svg viewBox="0 0 24 24" width="18" height="18">
                <path
                  fill="currentColor"
                  d="M14 2H6c-1.1 0-2 .9-2 2v16c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V8l-6-6zm4 18H6V4h7v5h5v11zM9 15v-1.5h6V15H9zm0-3v-1.5h6V12H9zm0-3V7.5h6V9H9z"
                />
              </svg>
              <span>节点信息</span>
            </div>
            <div class="info-grid">
              <div class="info-item">
                <span class="label">Kubelet Version</span>
                <span class="value">{{ statusData?.nodeInfo?.kubeletVersion || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="label">Container Runtime</span>
                <span class="value">{{
                  statusData?.nodeInfo?.containerRuntimeVersion || '-'
                }}</span>
              </div>
              <div class="info-item">
                <span class="label">Kernel Version</span>
                <span class="value">{{ statusData?.nodeInfo?.kernelVersion || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="label">OS Image</span>
                <span class="value">{{ statusData?.nodeInfo?.osImage || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="label">Architecture</span>
                <span class="value">{{ statusData?.nodeInfo?.architecture || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="label">Uptime</span>
                <span class="value">{{ formatDuration(statusData?.nodeInfo?.uptimeSeconds) }}</span>
              </div>
            </div>
          </div>

          <!-- Kubelet 指标区域 -->
          <div v-if="statusData?.kubeletMetrics" class="kubelet-section">
            <div class="section-header">
              <svg viewBox="0 0 24 24" width="18" height="18">
                <path
                  fill="currentColor"
                  d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zM9 17H7v-7h2v7zm4 0h-2V7h2v10zm4 0h-2v-4h2v4z"
                />
              </svg>
              <span>Kubelet 指标</span>
            </div>
            <div class="kubelet-stats">
              <div class="stat-item">
                <span class="label">Running Pods</span>
                <span class="value">{{ statusData.kubeletMetrics.runningPods }}</span>
              </div>
              <div class="stat-item">
                <span class="label">Running Containers</span>
                <span class="value">{{ statusData.kubeletMetrics.runningContainers }}</span>
              </div>
              <div class="stat-item">
                <span class="label">PLEG Relist Duration</span>
                <span class="value"
                  >{{ statusData.kubeletMetrics.plegRelistDuration.toFixed(2) }}ms</span
                >
              </div>
              <div class="stat-item">
                <span class="label">Runtime Ops Rate</span>
                <span class="value"
                  >{{ statusData.kubeletMetrics.runtimeOpsRate.toFixed(2) }}/s</span
                >
              </div>
              <div class="stat-item">
                <span class="label">Runtime Ops Error Rate</span>
                <span
                  class="value"
                  :class="{ danger: statusData.kubeletMetrics.runtimeOpsErrorRate > 0 }"
                >
                  {{ statusData.kubeletMetrics.runtimeOpsErrorRate.toFixed(2) }}/s
                </span>
              </div>
            </div>
          </div>

          <!-- Taints 区域 -->
          <div v-if="statusData?.taints?.length" class="taints-section">
            <div class="section-header">
              <svg viewBox="0 0 24 24" width="18" height="18">
                <path fill="currentColor" d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z" />
              </svg>
              <span>Taints</span>
              <span class="count-badge">{{ statusData.taints.length }}</span>
            </div>
            <div class="taints-list">
              <span v-for="(taint, index) in statusData.taints" :key="index" class="taint-tag">
                {{ taint.key }}={{ taint.value }}:{{ taint.effect }}
              </span>
            </div>
          </div>

          <!-- Labels 区域 -->
          <div v-if="statusData?.labels" class="labels-section">
            <div class="section-header">
              <svg viewBox="0 0 24 24" width="18" height="18">
                <path
                  fill="currentColor"
                  d="M21.41 11.58l-9-9C12.05 2.22 11.55 2 11 2H4c-1.1 0-2 .9-2 2v7c0 .55.22 1.05.59 1.42l9 9c.36.36.86.58 1.41.58.55 0 1.05-.22 1.41-.59l7-7c.37-.36.59-.86.59-1.41 0-.55-.23-1.06-.59-1.42zM5.5 7C4.67 7 4 6.33 4 5.5S4.67 4 5.5 4 7 4.67 7 5.5 6.33 7 5.5 7z"
                />
              </svg>
              <span>Labels</span>
              <span class="count-badge">{{ Object.keys(statusData.labels).length }}</span>
            </div>
            <div class="labels-list">
              <span v-for="(value, key) in statusData.labels" :key="key" class="label-tag">
                {{ key }}: {{ value }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- 空状态展示 -->
      <div v-else class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" width="64" height="64">
            <path
              fill="currentColor"
              d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14h-2V9h2v8zm0-10h-2V5h2v2z"
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
  import { formatBytes, formatDuration, smartFormatTime } from '@/utils/format'
  import type { NodeK8sStatus } from '@/api/console/node-monitor'

  interface Props {
    statusData?: NodeK8sStatus
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    statusData: undefined,
    loading: false
  })

  // 计算属性：判断是否有有效数据
  const hasData = computed(() => {
    return !!(props.statusData && Object.keys(props.statusData).length > 0)
  })

  // 获取 Ready 状态对应的样式类
  const getReadyStatusClass = (): string => {
    const ready = props.statusData?.conditions?.find((c) => c.type === 'Ready')
    return ready?.status ? 'success' : 'danger'
  }

  // 获取 Ready 状态对应的文本
  const getReadyStatusText = (): string => {
    const ready = props.statusData?.conditions?.find((c) => c.type === 'Ready')
    return ready?.status ? 'Ready' : 'NotReady'
  }

  // 判断某个 condition 是否需要告警样式
  const shouldWarn = (condition: any): boolean => {
    const warningTypes = ['MemoryPressure', 'DiskPressure', 'PIDPressure']
    return warningTypes.includes(condition.type) && condition.status
  }

  // 计算可分配资源占总容量的百分比
  const getAllocatablePercent = (type: 'cpu' | 'memory' | 'pods'): number => {
    if (!props.statusData?.capacity || !props.statusData?.allocatable) return 0

    const capacity =
      type === 'cpu'
        ? props.statusData.capacity.cpuCores
        : type === 'memory'
          ? props.statusData.capacity.memoryBytes
          : props.statusData.capacity.pods

    const allocatable =
      type === 'cpu'
        ? props.statusData.allocatable.cpuCores
        : type === 'memory'
          ? props.statusData.allocatable.memoryBytes
          : props.statusData.allocatable.pods

    return (allocatable / capacity) * 100
  }

  // 计算已分配资源占总容量的百分比
  const getAllocatedPercent = (type: 'cpu' | 'memory' | 'pods'): number => {
    if (!props.statusData?.capacity || !props.statusData?.allocated) return 0

    const capacity =
      type === 'cpu'
        ? props.statusData.capacity.cpuCores
        : type === 'memory'
          ? props.statusData.capacity.memoryBytes
          : props.statusData.capacity.pods

    const allocated =
      type === 'cpu'
        ? props.statusData.allocated.cpuCores
        : type === 'memory'
          ? props.statusData.allocated.memoryBytes
          : props.statusData.allocated.pods

    return (allocated / capacity) * 100
  }
</script>

<style lang="scss" scoped>
  // 主题颜色变量
  $primary: #3b82f6;
  $success: #10b981;
  $warning: #f59e0b;
  $danger: #ef4444;

  // 灰度色阶
  $gray-50: #f9fafb;
  $gray-100: #f3f4f6;
  $gray-200: #e5e7eb;
  $gray-300: #d1d5db;
  $gray-400: #9ca3af;
  $gray-500: #6b7280;
  $gray-600: #4b5563;
  $gray-700: #374151;
  $gray-800: #1f2937;

  // 圆角变量
  $radius: 14px;
  $radius-sm: 10px;
  $radius-xs: 6px;

  .node-status-monitor {
    font-family: var(--ax-font-sans);

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
      }
    }

    // 内容区域
    .content-wrapper {
      padding: 20px 24px;
    }

    // 统一的区块标题样式
    .section-header {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 16px;
      padding-bottom: 12px;
      border-bottom: 1px solid $gray-200;
      color: $gray-600;

      span {
        font-size: 14px;
        font-weight: 600;
      }

      .count-badge {
        padding: 2px 8px;
        border-radius: 10px;
        font-size: 11px;
        font-weight: 600;
        background: rgba($primary, 0.1);
        color: $primary;
      }
    }

    // 双栏布局
    .two-column-grid {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 20px;
      margin-bottom: 20px;

      @media (max-width: 1024px) {
        grid-template-columns: 1fr;
      }
    }

    // 节点状况区域
    .status-section,
    .capacity-section {
      background: $gray-50;
      border-radius: $radius-sm;
      padding: 20px;
    }

    .conditions-list {
      display: flex;
      flex-direction: column;
      gap: 12px;

      .condition-item {
        padding: 14px 16px;
        background: white;
        border-radius: $radius-xs;
        border-left: 4px solid $primary;
        transition: all 0.2s ease;

        &.warning {
          border-left-color: $warning;
          background: rgba($warning, 0.04);
        }

        .condition-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 8px;

          .condition-type {
            font-weight: 600;
            color: $gray-800;
            font-size: 13px;
          }

          .condition-badge {
            padding: 3px 10px;
            border-radius: 10px;
            font-size: 11px;
            font-weight: 600;

            &.success {
              background: rgba($success, 0.1);
              color: #059669;
            }
            &.info {
              background: $gray-100;
              color: $gray-500;
            }
          }
        }

        .condition-reason,
        .condition-message {
          font-size: 12px;
          color: $gray-600;
          margin-bottom: 4px;
        }

        .condition-time {
          font-size: 11px;
          color: $gray-400;
          margin-top: 8px;
        }
      }
    }

    // 资源容量区域
    .capacity-grid {
      display: flex;
      flex-direction: column;
      gap: 18px;

      .capacity-item {
        .capacity-label {
          font-weight: 600;
          color: $gray-700;
          margin-bottom: 10px;
          font-size: 13px;
        }

        .capacity-bars {
          display: flex;
          flex-direction: column;
          gap: 8px;

          .bar-row {
            display: flex;
            align-items: center;
            gap: 10px;

            .bar-label {
              min-width: 80px;
              font-size: 11px;
              color: $gray-500;
            }

            .bar-wrapper {
              flex: 1;
              height: 24px;
              background: $gray-100;
              border-radius: 4px;
              overflow: hidden;
              position: relative;

              .bar {
                height: 100%;
                display: flex;
                align-items: center;
                padding: 0 8px;
                transition: width 0.4s ease;

                &.capacity-bar {
                  width: 100%;
                  background: $gray-200;
                }
                &.allocatable-bar {
                  background: $success;
                }
                &.allocated-bar {
                  background: $primary;
                }

                .bar-text {
                  font-size: 11px;
                  font-weight: 600;
                  color: white;
                  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
                  white-space: nowrap;
                }
              }
            }
          }
        }
      }
    }

    // 节点信息区域
    .node-info-section,
    .kubelet-section,
    .taints-section,
    .labels-section {
      margin-top: 20px;
      padding: 20px;
      background: $gray-50;
      border-radius: $radius-sm;
    }

    .info-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      gap: 12px;

      .info-item {
        padding: 12px 16px;
        background: white;
        border-radius: $radius-xs;
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 12px;

        .label {
          font-size: 12px;
          color: $gray-500;
          font-weight: 500;
        }

        .value {
          font-size: 13px;
          color: $gray-700;
          font-weight: 600;
          text-align: right;
          word-break: break-all;
        }
      }
    }

    // Kubelet 统计区域
    .kubelet-stats {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
      gap: 12px;

      .stat-item {
        padding: 12px 16px;
        background: white;
        border-radius: $radius-xs;
        display: flex;
        justify-content: space-between;
        align-items: center;

        .label {
          font-size: 12px;
          color: $gray-500;
        }

        .value {
          font-size: 15px;
          font-weight: 700;
          color: $gray-800;

          &.danger {
            color: $danger;
          }
        }
      }
    }

    // Taints 和 Labels 列表
    .taints-list,
    .labels-list {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
    }

    .taint-tag {
      display: inline-block;
      padding: 6px 12px;
      border-radius: 14px;
      font-size: 11px;
      font-weight: 500;
      background: rgba($warning, 0.1);
      color: #d97706;
    }

    .label-tag {
      display: inline-block;
      padding: 6px 12px;
      border-radius: 14px;
      font-size: 11px;
      font-weight: 500;
      background: rgba($primary, 0.1);
      color: $primary;
      max-width: 300px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    // 空状态样式
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

  // 淡入淡出过渡动画
  .fade-enter-active,
  .fade-leave-active {
    transition: opacity 0.3s ease;
  }
  .fade-enter-from,
  .fade-leave-to {
    opacity: 0;
  }
</style>
