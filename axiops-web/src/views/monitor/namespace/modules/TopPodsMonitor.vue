<template>
  <div class="namespace-top-pods-monitor" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="monitor-card">
        <!-- 卡片头部 -->
        <div class="card-header">
          <div class="header-info">
            <div class="icon-badge top-pods">
              <svg viewBox="0 0 24 24" width="22" height="22">
                <path
                  fill="currentColor"
                  d="M16,17V19H2V17S2,13 9,13 16,17 16,17M12.5,7.5A3.5,3.5 0 0,1 9,11A3.5,3.5 0 0,1 5.5,7.5A3.5,3.5 0 0,1 9,4A3.5,3.5 0 0,1 12.5,7.5M15.94,13A5.32,5.32 0 0,1 18,17V19H22V17S22,13.37 15.94,13M15,4A3.39,3.39 0 0,0 13.07,4.59A5,5 0 0,1 13.07,10.41A3.39,3.39 0 0,0 15,11A3.5,3.5 0 0,0 18.5,7.5A3.5,3.5 0 0,0 15,4Z"
                />
              </svg>
            </div>
            <div>
              <h3>Top Pods</h3>
              <p>资源使用 Top 排行榜</p>
            </div>
          </div>
          <!-- Tab 切换 -->
          <div class="header-tabs">
            <button
              v-for="tab in tabOptions"
              :key="tab.value"
              class="tab-btn"
              :class="{ active: activeTab === tab.value }"
              @click="activeTab = tab.value"
            >
              <span class="tab-icon" :class="tab.value">
                <svg v-if="tab.value === 'cpu'" viewBox="0 0 24 24" width="14" height="14">
                  <path
                    fill="currentColor"
                    d="M17,17H7V7H17M21,11V9H19V7C19,5.89 18.1,5 17,5H15V3H13V5H11V3H9V5H7C5.89,5 5,5.89 5,7V9H3V11H5V13H3V15H5V17C5,18.1 5.9,19 7,19H9V21H11V19H13V21H15V19H17C18.1,19 19,18.1 19,17V15H21V13H19V11M15,15H9V9H15V15Z"
                  />
                </svg>
                <svg v-else-if="tab.value === 'memory'" viewBox="0 0 24 24" width="14" height="14">
                  <path
                    fill="currentColor"
                    d="M17,9H7V7H17M17,13H7V11H17M14,17H7V15H14M12,3A1,1 0 0,1 13,4A1,1 0 0,1 12,5A1,1 0 0,1 11,4A1,1 0 0,1 12,3M19,3H14.82C14.4,1.84 13.3,1 12,1C10.7,1 9.6,1.84 9.18,3H5A2,2 0 0,0 3,5V19A2,2 0 0,0 5,21H19A2,2 0 0,0 21,19V5A2,2 0 0,0 19,3Z"
                  />
                </svg>
                <svg v-else viewBox="0 0 24 24" width="14" height="14">
                  <path
                    fill="currentColor"
                    d="M12,21L15.6,16.2C14.6,15.45 13.35,15 12,15C10.65,15 9.4,15.45 8.4,16.2L12,21M12,3C7.95,3 4.21,4.34 1.2,6.6L3,9C5.5,7.12 8.62,6 12,6C15.38,6 18.5,7.12 21,9L22.8,6.6C19.79,4.34 16.05,3 12,3M12,9C9.3,9 6.81,9.89 4.8,11.4L6.6,13.8C8.1,12.67 9.97,12 12,12C14.03,12 15.9,12.67 17.4,13.8L19.2,11.4C17.19,9.89 14.7,9 12,9Z"
                  />
                </svg>
              </span>
              {{ tab.label }}
            </button>
          </div>
        </div>

        <!-- 内容区域 -->
        <div class="content-wrapper">
          <!-- CPU Top Pods -->
          <div v-if="activeTab === 'cpu'" class="top-list">
            <div class="section-header">
              <div class="section-icon cpu">
                <svg viewBox="0 0 24 24" width="18" height="18">
                  <path
                    fill="currentColor"
                    d="M3.5 18.5l6-6 4 4L22 6.92 20.59 5.5l-7.09 8-4-4L2 17l1.5 1.5z"
                  />
                </svg>
              </div>
              <span class="section-title">CPU 使用排行</span>
              <span class="section-count">Top {{ topCPUPods.length }}</span>
            </div>

            <div v-if="topCPUPods.length > 0" class="pods-list">
              <div
                v-for="(pod, index) in topCPUPods"
                :key="`cpu-${pod.name}`"
                class="pod-item"
                :class="{ highlight: index < 3 }"
              >
                <div class="pod-rank" :class="getRankClass(index)">
                  <span v-if="index < 3" class="rank-medal">{{ getRankMedal(index) }}</span>
                  <span v-else class="rank-number">{{ index + 1 }}</span>
                </div>

                <div class="pod-info">
                  <div class="pod-name">
                    <svg viewBox="0 0 24 24" width="14" height="14">
                      <path
                        fill="currentColor"
                        d="M21 16.5c0 .38-.21.71-.53.88l-7.9 4.44c-.16.12-.36.18-.57.18s-.41-.06-.57-.18l-7.9-4.44A.991.991 0 0 1 3 16.5v-9c0-.38.21-.71.53-.88l7.9-4.44c.16-.12.36-.18.57-.18s.41.06.57.18l7.9 4.44c.32.17.53.5.53.88v9z"
                      />
                    </svg>
                    <span class="name-text">{{ pod.name }}</span>
                  </div>
                </div>

                <div class="pod-value cpu">
                  <span class="value">{{ formatCPU(pod.value) }}</span>
                  <span class="unit">{{ pod.unit }}</span>
                </div>

                <div class="pod-progress">
                  <div class="progress-bar">
                    <div
                      class="progress-fill cpu"
                      :style="{ width: getPercentage(pod.value, topCPUPods[0]?.value) + '%' }"
                    ></div>
                  </div>
                </div>
              </div>
            </div>

            <div v-else class="empty-list">
              <svg viewBox="0 0 24 24" width="40" height="40">
                <path
                  fill="currentColor"
                  d="M17,17H7V7H17M21,11V9H19V7C19,5.89 18.1,5 17,5H15V3H13V5H11V3H9V5H7C5.89,5 5,5.89 5,7V9H3V11H5V13H3V15H5V17C5,18.1 5.9,19 7,19H9V21H11V19H13V21H15V19H17C18.1,19 19,18.1 19,17V15H21V13H19V11"
                />
              </svg>
              <span>暂无 CPU 使用数据</span>
            </div>
          </div>

          <!-- Memory Top Pods -->
          <div v-if="activeTab === 'memory'" class="top-list">
            <div class="section-header">
              <div class="section-icon memory">
                <svg viewBox="0 0 24 24" width="18" height="18">
                  <path
                    fill="currentColor"
                    d="M9 17H7v-7h2v7zm4 0h-2V7h2v10zm4 0h-2v-4h2v4zm2 2H5V5h14v14zm0-16H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2z"
                  />
                </svg>
              </div>
              <span class="section-title">内存使用排行</span>
              <span class="section-count">Top {{ topMemoryPods.length }}</span>
            </div>

            <div v-if="topMemoryPods.length > 0" class="pods-list">
              <div
                v-for="(pod, index) in topMemoryPods"
                :key="`memory-${pod.name}`"
                class="pod-item"
                :class="{ highlight: index < 3 }"
              >
                <div class="pod-rank" :class="getRankClass(index)">
                  <span v-if="index < 3" class="rank-medal">{{ getRankMedal(index) }}</span>
                  <span v-else class="rank-number">{{ index + 1 }}</span>
                </div>

                <div class="pod-info">
                  <div class="pod-name">
                    <svg viewBox="0 0 24 24" width="14" height="14">
                      <path
                        fill="currentColor"
                        d="M21 16.5c0 .38-.21.71-.53.88l-7.9 4.44c-.16.12-.36.18-.57.18s-.41-.06-.57-.18l-7.9-4.44A.991.991 0 0 1 3 16.5v-9c0-.38.21-.71.53-.88l7.9-4.44c.16-.12.36-.18.57-.18s.41.06.57.18l7.9 4.44c.32.17.53.5.53.88v9z"
                      />
                    </svg>
                    <span class="name-text">{{ pod.name }}</span>
                  </div>
                </div>

                <div class="pod-value memory">
                  <span class="value">{{ formatBytes(pod.value) }}</span>
                </div>

                <div class="pod-progress">
                  <div class="progress-bar">
                    <div
                      class="progress-fill memory"
                      :style="{ width: getPercentage(pod.value, topMemoryPods[0]?.value) + '%' }"
                    ></div>
                  </div>
                </div>
              </div>
            </div>

            <div v-else class="empty-list">
              <svg viewBox="0 0 24 24" width="40" height="40">
                <path
                  fill="currentColor"
                  d="M17,9H7V7H17M17,13H7V11H17M14,17H7V15H14M12,3A1,1 0 0,1 13,4A1,1 0 0,1 12,5A1,1 0 0,1 11,4A1,1 0 0,1 12,3M19,3H14.82C14.4,1.84 13.3,1 12,1C10.7,1 9.6,1.84 9.18,3H5A2,2 0 0,0 3,5V19A2,2 0 0,0 5,21H19A2,2 0 0,0 21,19V5A2,2 0 0,0 19,3Z"
                />
              </svg>
              <span>暂无内存使用数据</span>
            </div>
          </div>

          <!-- Network Top Pods -->
          <div v-if="activeTab === 'network'" class="top-list">
            <div class="section-header">
              <div class="section-icon network">
                <svg viewBox="0 0 24 24" width="18" height="18">
                  <path
                    fill="currentColor"
                    d="M12,21L15.6,16.2C14.6,15.45 13.35,15 12,15C10.65,15 9.4,15.45 8.4,16.2L12,21M12,3C7.95,3 4.21,4.34 1.2,6.6L3,9C5.5,7.12 8.62,6 12,6C15.38,6 18.5,7.12 21,9L22.8,6.6C19.79,4.34 16.05,3 12,3M12,9C9.3,9 6.81,9.89 4.8,11.4L6.6,13.8C8.1,12.67 9.97,12 12,12C14.03,12 15.9,12.67 17.4,13.8L19.2,11.4C17.19,9.89 14.7,9 12,9Z"
                  />
                </svg>
              </div>
              <span class="section-title">网络使用排行</span>
              <span class="section-count">Top {{ topNetworkPods.length }}</span>
            </div>

            <div v-if="topNetworkPods.length > 0" class="pods-list">
              <div
                v-for="(pod, index) in topNetworkPods"
                :key="`network-${pod.name}`"
                class="pod-item"
                :class="{ highlight: index < 3 }"
              >
                <div class="pod-rank" :class="getRankClass(index)">
                  <span v-if="index < 3" class="rank-medal">{{ getRankMedal(index) }}</span>
                  <span v-else class="rank-number">{{ index + 1 }}</span>
                </div>

                <div class="pod-info">
                  <div class="pod-name">
                    <svg viewBox="0 0 24 24" width="14" height="14">
                      <path
                        fill="currentColor"
                        d="M21 16.5c0 .38-.21.71-.53.88l-7.9 4.44c-.16.12-.36.18-.57.18s-.41-.06-.57-.18l-7.9-4.44A.991.991 0 0 1 3 16.5v-9c0-.38.21-.71.53-.88l7.9-4.44c.16-.12.36-.18.57-.18s.41.06.57.18l7.9 4.44c.32.17.53.5.53.88v9z"
                      />
                    </svg>
                    <span class="name-text">{{ pod.name }}</span>
                  </div>
                </div>

                <div class="pod-value network">
                  <span class="value">{{ formatBytes(pod.value) }}</span>
                </div>

                <div class="pod-progress">
                  <div class="progress-bar">
                    <div
                      class="progress-fill network"
                      :style="{ width: getPercentage(pod.value, topNetworkPods[0]?.value) + '%' }"
                    ></div>
                  </div>
                </div>
              </div>
            </div>

            <div v-else class="empty-list">
              <svg viewBox="0 0 24 24" width="40" height="40">
                <path
                  fill="currentColor"
                  d="M12,21L15.6,16.2C14.6,15.45 13.35,15 12,15C10.65,15 9.4,15.45 8.4,16.2L12,21M12,3C7.95,3 4.21,4.34 1.2,6.6L3,9C5.5,7.12 8.62,6 12,6C15.38,6 18.5,7.12 21,9L22.8,6.6C19.79,4.34 16.05,3 12,3M12,9C9.3,9 6.81,9.89 4.8,11.4L6.6,13.8C8.1,12.67 9.97,12 12,12C14.03,12 15.9,12.67 17.4,13.8L19.2,11.4C17.19,9.89 14.7,9 12,9Z"
                />
              </svg>
              <span>暂无网络使用数据</span>
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
              d="M16,17V19H2V17S2,13 9,13 16,17 16,17M12.5,7.5A3.5,3.5 0 0,1 9,11A3.5,3.5 0 0,1 5.5,7.5A3.5,3.5 0 0,1 9,4A3.5,3.5 0 0,1 12.5,7.5M15.94,13A5.32,5.32 0 0,1 18,17V19H22V17S22,13.37 15.94,13M15,4A3.39,3.39 0 0,0 13.07,4.59A5,5 0 0,1 13.07,10.41A3.39,3.39 0 0,0 15,11A3.5,3.5 0 0,0 18.5,7.5A3.5,3.5 0 0,0 15,4Z"
            />
          </svg>
        </div>
        <h3>暂无 Top Pods 数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { ref, computed } from 'vue'
  import type { ResourceRanking } from '@/api/console/namespace-monitor'
  import { formatBytes } from '@/utils/format'

  interface Props {
    topCPUPods?: ResourceRanking[]
    topMemoryPods?: ResourceRanking[]
    topNetworkPods?: ResourceRanking[]
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    topCPUPods: () => [],
    topMemoryPods: () => [],
    topNetworkPods: () => [],
    loading: false
  })

  // 当前激活的 Tab
  const activeTab = ref<'cpu' | 'memory' | 'network'>('cpu')

  // Tab 配置选项
  const tabOptions = [
    { value: 'cpu' as const, label: 'CPU' },
    { value: 'memory' as const, label: '内存' },
    { value: 'network' as const, label: '网络' }
  ]

  // 判断是否有数据
  const hasData = computed(() => {
    return (
      (props.topCPUPods && props.topCPUPods.length > 0) ||
      (props.topMemoryPods && props.topMemoryPods.length > 0) ||
      (props.topNetworkPods && props.topNetworkPods.length > 0)
    )
  })

  // 格式化 CPU 值显示
  const formatCPU = (value: number): string => {
    // 小于 0.001 核时转换为毫核显示
    if (value < 0.001) return (value * 1000).toFixed(0)
    return value.toFixed(3)
  }

  // 计算相对于最大值的百分比
  const getPercentage = (value: number, max?: number): number => {
    if (!max || max === 0) return 0
    return Math.min((value / max) * 100, 100)
  }

  // 获取排名样式类
  const getRankClass = (index: number): string => {
    if (index === 0) return 'rank-1'
    if (index === 1) return 'rank-2'
    if (index === 2) return 'rank-3'
    return 'rank-other'
  }

  // 获取排名奖牌标识
  const getRankMedal = (index: number): string => {
    const medals = ['🥇', '🥈', '🥉']
    return medals[index] || ''
  }
</script>

<style lang="scss" scoped>
  // 颜色变量定义
  $primary: #3b82f6;
  $success: #10b981;
  $warning: #f59e0b;
  $danger: #ef4444;

  $cpu-color: #667eea;
  $memory-color: #f093fb;
  $network-color: #4facfe;

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

  .namespace-top-pods-monitor {
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

          &.top-pods {
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

      .header-tabs {
        display: flex;
        gap: 8px;

        .tab-btn {
          display: flex;
          align-items: center;
          gap: 6px;
          padding: 8px 16px;
          border: 1px solid $gray-200;
          border-radius: $radius-xs;
          background: #fff;
          color: $gray-600;
          font-size: 13px;
          font-weight: 500;
          cursor: pointer;
          transition: all 0.2s ease;

          &:hover {
            border-color: $gray-300;
            background: $gray-50;
          }

          &.active {
            background: linear-gradient(135deg, $warning, #d97706);
            border-color: $warning;
            color: #fff;
          }

          .tab-icon {
            display: flex;
            align-items: center;
            justify-content: center;
          }
        }
      }
    }

    // 内容区域
    .content-wrapper {
      padding: 20px;
    }

    // Top 列表
    .top-list {
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
          color: #fff;

          &.cpu {
            background: linear-gradient(135deg, $cpu-color, #764ba2);
          }

          &.memory {
            background: linear-gradient(135deg, $memory-color, #f5576c);
          }

          &.network {
            background: linear-gradient(135deg, $network-color, #00f2fe);
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
          background: $gray-100;
          padding: 4px 10px;
          border-radius: 12px;
        }
      }

      .pods-list {
        display: flex;
        flex-direction: column;
        gap: 10px;

        .pod-item {
          display: flex;
          align-items: center;
          gap: 14px;
          padding: 14px 16px;
          background: $gray-50;
          border-radius: $radius-sm;
          border: 1px solid $gray-200;
          transition: all 0.2s ease;

          &.highlight {
            background: linear-gradient(135deg, #fffbeb 0%, #fef3c7 100%);
            border-color: #fcd34d;
          }

          &:hover {
            transform: translateX(4px);
            border-color: $gray-300;
          }

          .pod-rank {
            width: 44px;
            height: 44px;
            border-radius: $radius-xs;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 700;
            flex-shrink: 0;

            &.rank-1 {
              background: linear-gradient(135deg, #fde047, #facc15);
              box-shadow: 0 4px 12px rgba(250, 204, 21, 0.4);
            }

            &.rank-2 {
              background: linear-gradient(135deg, #cbd5e1, #94a3b8);
              box-shadow: 0 4px 12px rgba(148, 163, 184, 0.4);
            }

            &.rank-3 {
              background: linear-gradient(135deg, #fdba74, #fb923c);
              box-shadow: 0 4px 12px rgba(251, 146, 60, 0.4);
            }

            &.rank-other {
              background: $gray-100;
              color: $gray-500;
            }

            .rank-medal {
              font-size: 22px;
            }

            .rank-number {
              font-size: 18px;
            }
          }

          .pod-info {
            flex: 1;
            min-width: 0;

            .pod-name {
              display: flex;
              align-items: center;
              gap: 8px;
              font-size: 14px;
              font-weight: 600;
              color: $gray-700;

              svg {
                flex-shrink: 0;
                color: $gray-400;
              }

              .name-text {
                overflow: hidden;
                text-overflow: ellipsis;
                white-space: nowrap;
              }
            }
          }

          .pod-value {
            min-width: 100px;
            text-align: right;
            font-weight: 700;

            &.cpu {
              color: $cpu-color;
            }

            &.memory {
              color: $memory-color;
            }

            &.network {
              color: $network-color;
            }

            .value {
              font-size: 16px;
            }

            .unit {
              font-size: 12px;
              opacity: 0.7;
              margin-left: 2px;
            }
          }

          .pod-progress {
            width: 120px;

            .progress-bar {
              height: 6px;
              background: $gray-200;
              border-radius: 3px;
              overflow: hidden;

              .progress-fill {
                height: 100%;
                border-radius: 3px;
                transition: width 0.4s ease;

                &.cpu {
                  background: linear-gradient(90deg, $cpu-color, #764ba2);
                }

                &.memory {
                  background: linear-gradient(90deg, $memory-color, #f5576c);
                }

                &.network {
                  background: linear-gradient(90deg, $network-color, #00f2fe);
                }
              }
            }
          }
        }
      }

      .empty-list {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 40px 20px;
        color: $gray-400;

        svg {
          margin-bottom: 12px;
        }

        span {
          font-size: 14px;
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
