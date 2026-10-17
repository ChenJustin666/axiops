<template>
  <div class="namespace-events-monitor" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="monitor-card">
        <!-- 卡片头部 -->
        <div class="card-header">
          <div class="header-info">
            <div class="icon-badge events">
              <svg viewBox="0 0 24 24" width="22" height="22">
                <path
                  fill="currentColor"
                  d="M13,9H11V7H13M13,17H11V11H13M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2Z"
                />
              </svg>
            </div>
            <div>
              <h3>事件监控</h3>
              <p>Namespace 事件记录与分析</p>
            </div>
          </div>
          <div class="header-stats">
            <div class="stat-badge total">
              <span class="label">总事件</span>
              <span class="value">{{ eventsData?.totalEvents || 0 }}</span>
            </div>
            <div class="stat-badge normal">
              <span class="label">正常</span>
              <span class="value">{{ eventsData?.normalEvents || 0 }}</span>
            </div>
            <div class="stat-badge warning" v-if="(eventsData?.warningEvents || 0) > 0">
              <span class="label">警告</span>
              <span class="value">{{ eventsData?.warningEvents || 0 }}</span>
            </div>
          </div>
        </div>

        <!-- 内容区域 -->
        <div class="content-wrapper">
          <!-- 事件类型筛选 -->
          <div class="filter-section">
            <div class="filter-label">事件类型</div>
            <div class="filter-buttons">
              <button
                v-for="filter in filterOptions"
                :key="filter.value"
                class="filter-btn"
                :class="{ active: eventTypeFilter === filter.value }"
                @click="eventTypeFilter = filter.value"
              >
                <span class="btn-icon" :class="filter.iconClass">
                  <svg v-if="filter.value === 'all'" viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M3 13h2v-2H3v2zm0 4h2v-2H3v2zm0-8h2V7H3v2zm4 4h14v-2H7v2zm0 4h14v-2H7v2zM7 7v2h14V7H7z"
                    />
                  </svg>
                  <svg
                    v-else-if="filter.value === 'Normal'"
                    viewBox="0 0 24 24"
                    width="14"
                    height="14"
                  >
                    <path
                      fill="currentColor"
                      d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"
                    />
                  </svg>
                  <svg v-else viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"
                    />
                  </svg>
                </span>
                {{ filter.label }}
                <span class="btn-count">{{ filter.count }}</span>
              </button>
            </div>
          </div>

          <!-- 事件列表 -->
          <div v-if="filteredEvents.length > 0" class="events-list">
            <div
              v-for="(event, index) in displayedEvents"
              :key="`${event.involvedObject.name}-${event.timestamp}-${index}`"
              class="event-item"
              :class="{ warning: event.type === 'Warning' }"
            >
              <div class="event-icon" :class="event.type === 'Warning' ? 'warning' : 'normal'">
                <svg v-if="event.type === 'Warning'" viewBox="0 0 24 24" width="20" height="20">
                  <path
                    fill="currentColor"
                    d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"
                  />
                </svg>
                <svg v-else viewBox="0 0 24 24" width="20" height="20">
                  <path fill="currentColor" d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                </svg>
              </div>

              <div class="event-content">
                <div class="event-header">
                  <span
                    class="event-type-tag"
                    :class="event.type === 'Warning' ? 'warning' : 'success'"
                  >
                    {{ event.type }}
                  </span>
                  <span class="event-reason">{{ event.reason }}</span>
                  <span v-if="event.count > 1" class="event-count">
                    <svg viewBox="0 0 24 24" width="12" height="12">
                      <path
                        fill="currentColor"
                        d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6s-2.69 6-6 6-6-2.69-6-6H4c0 4.42 3.58 8 8 8s8-3.58 8-8-3.58-8-8-8z"
                      />
                    </svg>
                    {{ event.count }} 次
                  </span>
                </div>

                <div class="event-message">{{ event.message }}</div>

                <div class="event-meta">
                  <span class="meta-item object">
                    <svg viewBox="0 0 24 24" width="12" height="12">
                      <path
                        fill="currentColor"
                        d="M21 16.5c0 .38-.21.71-.53.88l-7.9 4.44c-.16.12-.36.18-.57.18s-.41-.06-.57-.18l-7.9-4.44A.991.991 0 0 1 3 16.5v-9c0-.38.21-.71.53-.88l7.9-4.44c.16-.12.36-.18.57-.18s.41.06.57.18l7.9 4.44c.32.17.53.5.53.88v9z"
                      />
                    </svg>
                    {{ event.involvedObject.kind }}/{{ event.involvedObject.name }}
                  </span>
                  <span class="meta-item time">
                    <svg viewBox="0 0 24 24" width="12" height="12">
                      <path
                        fill="currentColor"
                        d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10 10-4.5 10-10S17.5 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67V7z"
                      />
                    </svg>
                    {{ formatTime(event.timestamp) }}
                  </span>
                </div>
              </div>
            </div>

            <!-- 加载更多 -->
            <div v-if="filteredEvents.length > displayLimit" class="load-more">
              <button class="load-more-btn" @click="loadMore">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path
                    fill="currentColor"
                    d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6s-2.69 6-6 6-6-2.69-6-6H4c0 4.42 3.58 8 8 8s8-3.58 8-8-3.58-8-8-8z"
                  />
                </svg>
                加载更多 (剩余 {{ filteredEvents.length - displayLimit }} 条)
              </button>
            </div>
          </div>

          <!-- 空状态 -->
          <div v-else class="empty-events">
            <div class="empty-icon">
              <svg viewBox="0 0 24 24" width="48" height="48">
                <path fill="currentColor" d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
              </svg>
            </div>
            <span class="empty-text">暂无事件记录</span>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" width="64" height="64">
            <path
              fill="currentColor"
              d="M13,9H11V7H13M13,17H11V11H13M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2Z"
            />
          </svg>
        </div>
        <h3>暂无事件数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { ref, computed } from 'vue'
  import type { NamespaceEvents } from '@/api/console/namespace-monitor'

  interface Props {
    eventsData?: NamespaceEvents
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    eventsData: undefined,
    loading: false
  })

  // 筛选状态
  const eventTypeFilter = ref<'all' | 'Normal' | 'Warning'>('all')
  // 显示数量限制
  const displayLimit = ref(10)

  // 判断是否有数据
  const hasData = computed(() => {
    return !!(props.eventsData && props.eventsData.events && props.eventsData.events.length > 0)
  })

  // 筛选选项配置
  const filterOptions = computed(() => [
    {
      value: 'all' as const,
      label: '全部',
      iconClass: 'all',
      count: props.eventsData?.totalEvents || 0
    },
    {
      value: 'Normal' as const,
      label: '正常',
      iconClass: 'normal',
      count: props.eventsData?.normalEvents || 0
    },
    {
      value: 'Warning' as const,
      label: '警告',
      iconClass: 'warning',
      count: props.eventsData?.warningEvents || 0
    }
  ])

  // 根据类型筛选事件
  const filteredEvents = computed(() => {
    if (!props.eventsData?.events) return []
    if (eventTypeFilter.value === 'all') return props.eventsData.events
    return props.eventsData.events.filter((event) => event.type === eventTypeFilter.value)
  })

  // 当前显示的事件列表
  const displayedEvents = computed(() => {
    return filteredEvents.value.slice(0, displayLimit.value)
  })

  // 格式化时间显示
  const formatTime = (timestamp: string): string => {
    if (!timestamp) return ''
    const date = new Date(timestamp)
    const now = new Date()
    const diff = now.getTime() - date.getTime()

    // 小于1分钟显示"刚刚"
    if (diff < 60000) {
      return '刚刚'
    }
    // 小于1小时显示分钟数
    if (diff < 3600000) {
      return `${Math.floor(diff / 60000)} 分钟前`
    }
    // 小于1天显示小时数
    if (diff < 86400000) {
      return `${Math.floor(diff / 3600000)} 小时前`
    }
    // 大于1天显示具体日期
    return date.toLocaleString('zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  // 加载更多事件
  const loadMore = () => {
    displayLimit.value += 10
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

  .namespace-events-monitor {
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
      background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
      border-bottom: 1px solid #bae6fd;
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

          &.events {
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

          &.normal {
            background: linear-gradient(135deg, #d1fae5 0%, #a7f3d0 100%);
            color: #065f46;
          }

          &.warning {
            background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
            color: #92400e;
          }
        }
      }
    }

    // 内容区域
    .content-wrapper {
      padding: 20px;
    }

    // 筛选区域
    .filter-section {
      margin-bottom: 20px;

      .filter-label {
        font-size: 13px;
        font-weight: 600;
        color: $gray-600;
        margin-bottom: 10px;
      }

      .filter-buttons {
        display: flex;
        gap: 8px;
        flex-wrap: wrap;

        .filter-btn {
          display: flex;
          align-items: center;
          gap: 6px;
          padding: 8px 14px;
          border: 1px solid $gray-200;
          border-radius: $radius-xs;
          background: #fff;
          color: $gray-600;
          font-size: 13px;
          cursor: pointer;
          transition: all 0.2s ease;

          &:hover {
            border-color: $gray-300;
            background: $gray-50;
          }

          &.active {
            background: linear-gradient(135deg, $primary, #2563eb);
            border-color: $primary;
            color: #fff;

            .btn-count {
              background: rgba(255, 255, 255, 0.2);
              color: #fff;
            }
          }

          .btn-icon {
            display: flex;
            align-items: center;
            justify-content: center;
          }

          .btn-count {
            padding: 2px 6px;
            border-radius: 10px;
            background: $gray-100;
            font-size: 11px;
            font-weight: 600;
          }
        }
      }
    }

    // 事件列表
    .events-list {
      display: flex;
      flex-direction: column;
      gap: 12px;

      .event-item {
        display: flex;
        gap: 14px;
        padding: 16px;
        background: $gray-50;
        border-radius: $radius-sm;
        border: 1px solid $gray-200;
        border-left: 4px solid $success;
        transition: all 0.2s ease;

        &.warning {
          border-left-color: $warning;
          background: linear-gradient(135deg, #fffbeb 0%, #fef3c7 100%);
        }

        &:hover {
          border-color: $gray-300;
          transform: translateX(4px);
        }

        .event-icon {
          width: 40px;
          height: 40px;
          border-radius: $radius-xs;
          display: flex;
          align-items: center;
          justify-content: center;
          flex-shrink: 0;

          &.normal {
            background: linear-gradient(135deg, #d1fae5, #a7f3d0);
            color: #059669;
          }

          &.warning {
            background: linear-gradient(135deg, #fef3c7, #fde68a);
            color: #d97706;
          }
        }

        .event-content {
          flex: 1;
          min-width: 0;

          .event-header {
            display: flex;
            align-items: center;
            gap: 8px;
            margin-bottom: 8px;
            flex-wrap: wrap;

            .event-type-tag {
              padding: 3px 8px;
              border-radius: 4px;
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
            }

            .event-reason {
              font-size: 14px;
              font-weight: 600;
              color: $gray-700;
            }

            .event-count {
              display: flex;
              align-items: center;
              gap: 4px;
              padding: 3px 8px;
              background: $gray-100;
              border-radius: 4px;
              font-size: 11px;
              color: $gray-500;
            }
          }

          .event-message {
            font-size: 13px;
            color: $gray-600;
            line-height: 1.6;
            margin-bottom: 10px;
          }

          .event-meta {
            display: flex;
            align-items: center;
            gap: 16px;
            flex-wrap: wrap;

            .meta-item {
              display: flex;
              align-items: center;
              gap: 4px;
              font-size: 12px;
              color: $gray-400;

              &.object {
                color: $primary;
              }

              &.time {
                color: $gray-400;
              }
            }
          }
        }
      }

      .load-more {
        display: flex;
        justify-content: center;
        padding: 16px;

        .load-more-btn {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 10px 20px;
          border: 1px solid $gray-200;
          border-radius: $radius-xs;
          background: #fff;
          color: $primary;
          font-size: 13px;
          font-weight: 500;
          cursor: pointer;
          transition: all 0.2s ease;

          &:hover {
            background: $gray-50;
            border-color: $primary;
          }
        }
      }
    }

    // 事件为空状态
    .empty-events {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 60px 20px;
      color: $gray-400;

      .empty-icon {
        margin-bottom: 16px;
      }

      .empty-text {
        font-size: 14px;
      }
    }

    // 数据空状态
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

  // 响应式适配
  @media (max-width: 768px) {
    .namespace-events-monitor {
      .card-header {
        padding: 16px 18px;
      }

      .event-item {
        padding: 12px;

        .event-icon {
          width: 32px;
          height: 32px;
        }

        .event-meta {
          flex-direction: column;
          align-items: flex-start;
          gap: 6px;
        }
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
