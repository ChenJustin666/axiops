<template>
  <div class="namespace-monitor" v-loading="loading">
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
                    d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"
                  />
                </svg>
              </div>
              <div>
                <h2>命名空间</h2>
                <p class="subtitle">Namespace 资源使用排名</p>
              </div>
            </div>
            <div class="header-badges">
              <div class="stat-badge total">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path
                    fill="currentColor"
                    d="M4 8h4V4H4v4zm6 12h4v-4h-4v4zm-6 0h4v-4H4v4zm0-6h4v-4H4v4zm6 0h4v-4h-4v4zm6-10v4h4V4h-4zm-6 4h4V4h-4v4zm6 6h4v-4h-4v4zm0 6h4v-4h-4v4z"
                  />
                </svg>
                <span class="badge-label">总数</span>
                <span class="badge-value">{{ namespaceData?.total || 0 }}</span>
              </div>
              <div class="stat-badge active">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path
                    fill="currentColor"
                    d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"
                  />
                </svg>
                <span class="badge-label">活跃</span>
                <span class="badge-value">{{ namespaceData?.active || 0 }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 排名卡片网格 -->
        <div class="ranking-grid">
          <!-- CPU 使用排名 -->
          <div class="ranking-card">
            <div class="card-header">
              <div class="header-info">
                <div class="icon-badge cpu">
                  <svg viewBox="0 0 24 24" width="22" height="22">
                    <path
                      fill="currentColor"
                      d="M9 3V1H7v2H5v2H3v2h2v2H3v2h2v2H3v2h2v2h2v2h2v-2h2v2h2v-2h2v2h2v-2h2v-2h2v-2h-2v-2h2v-2h-2V9h2V7h-2V5h-2V3h-2v2h-2V3h-2v2H9V3zm0 4h6v2h2v6h-2v2H9v-2H7V9h2V7z"
                    />
                  </svg>
                </div>
                <div>
                  <h3>CPU 使用 Top 10</h3>
                  <p>按 CPU 使用量排名</p>
                </div>
              </div>
            </div>

            <div
              class="ranking-list"
              v-if="namespaceData?.topByCPU && namespaceData.topByCPU.length > 0"
            >
              <div
                v-for="(item, index) in namespaceData.topByCPU"
                :key="item.namespace"
                class="ranking-item"
              >
                <div class="rank-badge" :class="getRankClass(index)">{{ index + 1 }}</div>
                <div class="rank-content">
                  <div class="rank-info">
                    <div class="namespace-name">
                      <svg viewBox="0 0 24 24" width="14" height="14">
                        <path
                          fill="currentColor"
                          d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"
                        />
                      </svg>
                      <span>{{ item.namespace }}</span>
                    </div>
                    <span class="namespace-value">{{ formatValue(item.value, item.unit) }}</span>
                  </div>
                  <div class="rank-bar">
                    <div
                      class="bar-fill cpu"
                      :style="{ width: getBarWidth(item.value, namespaceData.topByCPU) + '%' }"
                    ></div>
                  </div>
                </div>
              </div>
            </div>

            <div v-else class="empty-ranking">
              <svg viewBox="0 0 24 24" width="40" height="40">
                <path
                  fill="currentColor"
                  d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14h-2V9h2v8zm0-10h-2V5h2v2z"
                />
              </svg>
              <span>暂无数据</span>
            </div>
          </div>

          <!-- 内存使用排名 -->
          <div class="ranking-card">
            <div class="card-header">
              <div class="header-info">
                <div class="icon-badge memory">
                  <svg viewBox="0 0 24 24" width="22" height="22">
                    <path
                      fill="currentColor"
                      d="M2 7v10h20V7H2zm18 8H4V9h16v6zM6 11h2v2H6v-2zm3 0h2v2H9v-2zm3 0h2v2h-2v-2zm3 0h2v2h-2v-2z"
                    />
                  </svg>
                </div>
                <div>
                  <h3>内存使用 Top 10</h3>
                  <p>按内存使用量排名</p>
                </div>
              </div>
            </div>

            <div
              class="ranking-list"
              v-if="namespaceData?.topByMemory && namespaceData.topByMemory.length > 0"
            >
              <div
                v-for="(item, index) in namespaceData.topByMemory"
                :key="item.namespace"
                class="ranking-item"
              >
                <div class="rank-badge" :class="getRankClass(index)">{{ index + 1 }}</div>
                <div class="rank-content">
                  <div class="rank-info">
                    <div class="namespace-name">
                      <svg viewBox="0 0 24 24" width="14" height="14">
                        <path
                          fill="currentColor"
                          d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"
                        />
                      </svg>
                      <span>{{ item.namespace }}</span>
                    </div>
                    <span class="namespace-value">{{ formatValue(item.value, item.unit) }}</span>
                  </div>
                  <div class="rank-bar">
                    <div
                      class="bar-fill memory"
                      :style="{ width: getBarWidth(item.value, namespaceData.topByMemory) + '%' }"
                    ></div>
                  </div>
                </div>
              </div>
            </div>

            <div v-else class="empty-ranking">
              <svg viewBox="0 0 24 24" width="40" height="40">
                <path
                  fill="currentColor"
                  d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14h-2V9h2v8zm0-10h-2V5h2v2z"
                />
              </svg>
              <span>暂无数据</span>
            </div>
          </div>

          <!-- Pod 数量排名 -->
          <div class="ranking-card">
            <div class="card-header">
              <div class="header-info">
                <div class="icon-badge pod">
                  <svg viewBox="0 0 24 24" width="22" height="22">
                    <path
                      fill="currentColor"
                      d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"
                    />
                  </svg>
                </div>
                <div>
                  <h3>Pod 数量 Top 10</h3>
                  <p>按 Pod 数量排名</p>
                </div>
              </div>
            </div>

            <div
              class="ranking-list"
              v-if="namespaceData?.topByPods && namespaceData.topByPods.length > 0"
            >
              <div
                v-for="(item, index) in namespaceData.topByPods"
                :key="item.namespace"
                class="ranking-item"
              >
                <div class="rank-badge" :class="getRankClass(index)">{{ index + 1 }}</div>
                <div class="rank-content">
                  <div class="rank-info">
                    <div class="namespace-name">
                      <svg viewBox="0 0 24 24" width="14" height="14">
                        <path
                          fill="currentColor"
                          d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"
                        />
                      </svg>
                      <span>{{ item.namespace }}</span>
                    </div>
                    <span class="namespace-value">{{ formatValue(item.value, item.unit) }}</span>
                  </div>
                  <div class="rank-bar">
                    <div
                      class="bar-fill pod"
                      :style="{ width: getBarWidth(item.value, namespaceData.topByPods) + '%' }"
                    ></div>
                  </div>
                </div>
              </div>
            </div>

            <div v-else class="empty-ranking">
              <svg viewBox="0 0 24 24" width="40" height="40">
                <path
                  fill="currentColor"
                  d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14h-2V9h2v8zm0-10h-2V5h2v2z"
                />
              </svg>
              <span>暂无数据</span>
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
              d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"
            />
          </svg>
        </div>
        <h3>暂无命名空间监控数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import type { ClusterNamespaceMetrics } from '@/api/console/cluster-monitor'

  interface Props {
    namespaceData?: ClusterNamespaceMetrics
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    namespaceData: undefined,
    loading: false
  })

  const hasData = computed(() => {
    return !!(props.namespaceData && Object.keys(props.namespaceData).length > 0)
  })

  // 获取排名样式类
  const getRankClass = (index: number): string => {
    if (index === 0) return 'gold'
    if (index === 1) return 'silver'
    if (index === 2) return 'bronze'
    return 'normal'
  }

  // 计算进度条宽度
  const getBarWidth = (value: number, list: Array<{ value: number }>): number => {
    if (!list || list.length === 0) return 0
    const maxValue = Math.max(...list.map((item) => item.value))
    if (maxValue === 0) return 0
    return (value / maxValue) * 100
  }

  // 格式化数值
  const formatValue = (value: number, unit?: string) => {
    // 如果单位是 bytes，转换成易读格式
    if (unit === 'bytes') {
      const bytes = value
      if (bytes < 1024) return `${bytes.toFixed(2)} B`
      if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`
      if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
      return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
    }

    // 如果单位是 cores，保留3位小数
    if (unit === 'cores') {
      return `${value.toFixed(3)} cores`
    }

    // 如果单位是 pods，显示整数
    if (unit === 'pods' || !unit) {
      return `${Math.round(value)} ${unit || 'pods'}`
    }

    // 其他情况保留2位小数
    return `${value.toFixed(2)} ${unit || ''}`
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

  .namespace-monitor {
    background: $gray-50;
    font-family: var(--ax-font-sans);

    .monitor-container {
      max-width: 1800px;
      margin: 0 auto;
      display: flex;
      flex-direction: column;
      gap: 24px;
    }

    .overview-card {
      background: #fff;
      border-radius: $radius;
      padding: 24px;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);

      .overview-header {
        display: flex;
        justify-content: space-between;
        align-items: center;

        @media (max-width: 768px) {
          flex-direction: column;
          gap: 16px;
          align-items: flex-start;
        }

        .header-left {
          display: flex;
          align-items: center;
          gap: 14px;

          .header-icon {
            width: 44px;
            height: 44px;
            border-radius: $radius-sm;
            background: linear-gradient(135deg, #fce7f3, #fbcfe8);
            display: flex;
            align-items: center;
            justify-content: center;
            color: #db2777;
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

        .header-badges {
          display: flex;
          gap: 12px;

          .stat-badge {
            display: flex;
            align-items: center;
            gap: 8px;
            padding: 10px 16px;
            border-radius: $radius-sm;
            font-size: 13px;

            svg {
              flex-shrink: 0;
            }

            .badge-label {
              color: inherit;
              opacity: 0.8;
            }

            .badge-value {
              font-weight: 700;
              font-size: 16px;
            }

            &.total {
              background: rgba($primary, 0.08);
              color: $primary;
            }

            &.active {
              background: rgba($success, 0.08);
              color: #059669;
            }
          }
        }
      }
    }

    .ranking-grid {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 20px;

      @media (max-width: 1400px) {
        grid-template-columns: repeat(2, 1fr);
      }

      @media (max-width: 900px) {
        grid-template-columns: 1fr;
      }
    }

    .ranking-card {
      background: #fff;
      border-radius: $radius;
      padding: 20px;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);

      .card-header {
        margin-bottom: 18px;
        padding-bottom: 14px;
        border-bottom: 1px solid $gray-200;

        .header-info {
          display: flex;
          align-items: center;
          gap: 12px;

          .icon-badge {
            width: 46px;
            height: 46px;
            border-radius: $radius-sm;
            display: flex;
            align-items: center;
            justify-content: center;
            color: #fff;
            flex-shrink: 0;

            &.cpu {
              background: linear-gradient(135deg, #667eea, #764ba2);
            }

            &.memory {
              background: linear-gradient(135deg, #f093fb, #f5576c);
            }

            &.pod {
              background: linear-gradient(135deg, #43e97b, #38f9d7);
            }
          }

          h3 {
            margin: 0;
            font-size: 15px;
            font-weight: 600;
            color: $gray-800;
          }

          p {
            margin: 2px 0 0;
            font-size: 12px;
            color: $gray-500;
          }
        }
      }

      .ranking-list {
        display: flex;
        flex-direction: column;
        gap: 10px;

        .ranking-item {
          display: flex;
          align-items: center;
          gap: 12px;
          padding: 12px 14px;
          background: $gray-50;
          border-radius: $radius-xs;
          border: 1px solid $gray-200;
          transition: all 0.2s ease;

          &:hover {
            border-color: $gray-300;
            transform: translateX(4px);
          }

          .rank-badge {
            width: 28px;
            height: 28px;
            border-radius: 8px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 700;
            font-size: 12px;
            color: #fff;
            flex-shrink: 0;

            &.gold {
              background: linear-gradient(135deg, #fbbf24, #f59e0b);
              box-shadow: 0 2px 8px rgba(251, 191, 36, 0.4);
            }

            &.silver {
              background: linear-gradient(135deg, #d1d5db, #9ca3af);
              box-shadow: 0 2px 8px rgba(156, 163, 175, 0.4);
            }

            &.bronze {
              background: linear-gradient(135deg, #fb923c, #f97316);
              box-shadow: 0 2px 8px rgba(249, 115, 22, 0.4);
            }

            &.normal {
              background: linear-gradient(135deg, #60a5fa, #3b82f6);
            }
          }

          .rank-content {
            flex: 1;
            min-width: 0;

            .rank-info {
              display: flex;
              justify-content: space-between;
              align-items: center;
              margin-bottom: 8px;

              .namespace-name {
                display: flex;
                align-items: center;
                gap: 6px;
                font-size: 13px;
                font-weight: 500;
                color: $gray-700;

                svg {
                  color: $gray-400;
                  flex-shrink: 0;
                }

                span {
                  overflow: hidden;
                  text-overflow: ellipsis;
                  white-space: nowrap;
                }
              }

              .namespace-value {
                font-size: 13px;
                font-weight: 600;
                color: $gray-600;
                flex-shrink: 0;
                margin-left: 8px;
              }
            }

            .rank-bar {
              height: 6px;
              background: $gray-200;
              border-radius: 3px;
              overflow: hidden;

              .bar-fill {
                height: 100%;
                border-radius: 3px;
                transition: width 0.6s ease;

                &.cpu {
                  background: linear-gradient(90deg, #667eea, #764ba2);
                }

                &.memory {
                  background: linear-gradient(90deg, #f093fb, #f5576c);
                }

                &.pod {
                  background: linear-gradient(90deg, #43e97b, #38f9d7);
                }
              }
            }
          }
        }
      }

      .empty-ranking {
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
          font-size: 13px;
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

  .fade-enter-active,
  .fade-leave-active {
    transition: opacity 0.3s ease;
  }

  .fade-enter-from,
  .fade-leave-to {
    opacity: 0;
  }
</style>
