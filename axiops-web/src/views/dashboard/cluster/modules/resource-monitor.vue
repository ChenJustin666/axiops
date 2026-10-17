<template>
  <div class="resource-monitor" v-loading="loading">
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
                    d="M19,3H14.82C14.4,1.84 13.3,1 12,1C10.7,1 9.6,1.84 9.18,3H5A2,2 0 0,0 3,5V19A2,2 0 0,0 5,21H19A2,2 0 0,0 21,19V5A2,2 0 0,0 19,3M12,3A1,1 0 0,1 13,4A1,1 0 0,1 12,5A1,1 0 0,1 11,4A1,1 0 0,1 12,3"
                  />
                </svg>
              </div>
              <div>
                <h2>集群资源</h2>
                <p class="subtitle">CPU、内存、Pod、存储资源监控</p>
              </div>
            </div>
            <div class="health-badge" :class="resourceHealthStatus.status">
              <span class="status-dot"></span>
              <span class="status-text">{{ resourceHealthStatus.text }}</span>
              <span class="health-score">{{ resourceHealthStatus.score }}/100</span>
            </div>
          </div>

          <!-- 快速指标 -->
          <div class="quick-metrics">
            <div class="quick-metric">
              <div class="metric-icon cpu">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M9 3V1H7v2H5v2H3v2h2v2H3v2h2v2H3v2h2v2h2v2h2v-2h2v2h2v-2h2v2h2v-2h2v-2h2v-2h-2v-2h2v-2h-2V9h2V7h-2V5h-2V3h-2v2h-2V3h-2v2H9V3zm0 4h6v2h2v6h-2v2H9v-2H7V9h2V7z"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">CPU 使用率</span>
                <span class="metric-value">{{
                  formatPercentage(resourceData?.cpu?.usagePercent)
                }}</span>
                <span class="metric-sub">容量 {{ formatCores(resourceData?.cpu?.capacity) }}</span>
              </div>
            </div>
            <div class="quick-metric">
              <div class="metric-icon memory">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M2 7v10h20V7H2zm18 8H4V9h16v6zM6 11h2v2H6v-2zm3 0h2v2H9v-2zm3 0h2v2h-2v-2zm3 0h2v2h-2v-2z"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">内存使用率</span>
                <span class="metric-value">{{
                  formatPercentage(resourceData?.memory?.usagePercent)
                }}</span>
                <span class="metric-sub"
                  >容量 {{ formatBytes(resourceData?.memory?.capacity) }}</span
                >
              </div>
            </div>
            <div class="quick-metric">
              <div class="metric-icon pod">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">Pod 使用率</span>
                <span class="metric-value">{{
                  formatPercentage(resourceData?.pods?.usagePercent)
                }}</span>
                <span class="metric-sub"
                  >{{ formatLargeNumber(resourceData?.pods?.running) }} /
                  {{ formatLargeNumber(resourceData?.pods?.capacity) }}</span
                >
              </div>
            </div>
          </div>
        </div>

        <!-- 资源详情卡片 -->
        <div class="metrics-grid">
          <!-- CPU 卡片 -->
          <div class="monitor-card">
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
                  <h3>CPU 资源</h3>
                  <p>计算资源使用详情</p>
                </div>
              </div>
              <div class="status-tag" :class="getCPUStatus.class">{{ getCPUStatus.text }}</div>
            </div>

            <div class="metrics-content">
              <div class="metric-box primary">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="16" height="16">
                    <path
                      fill="currentColor"
                      d="M13 2.05v2.02c3.95.49 7 3.85 7 7.93s-3.05 7.44-7 7.93v2.02c5.05-.5 9-4.76 9-9.95s-3.95-9.45-9-9.95z"
                    />
                  </svg>
                  <span>使用率</span>
                </div>
                <div class="box-value">
                  <span class="value">{{ formatPercentage(resourceData?.cpu?.usagePercent) }}</span>
                </div>
                <div class="box-progress">
                  <div
                    class="progress-fill"
                    :class="getUsageLevelClass(resourceData?.cpu?.usagePercent)"
                    :style="{ width: `${Math.min(resourceData?.cpu?.usagePercent || 0, 100)}%` }"
                  ></div>
                </div>
              </div>

              <div class="metric-box">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="16" height="16">
                    <path
                      fill="currentColor"
                      d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10 10-4.5 10-10S17.5 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8z"
                    />
                  </svg>
                  <span>已使用</span>
                </div>
                <div class="box-value">
                  <span class="value">{{ formatCores(resourceData?.cpu?.usage) }}</span>
                </div>
              </div>

              <div class="metric-box">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="16" height="16">
                    <path
                      fill="currentColor"
                      d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V5h14v14z"
                    />
                  </svg>
                  <span>可分配</span>
                </div>
                <div class="box-value">
                  <span class="value">{{ formatCores(resourceData?.cpu?.allocatable) }}</span>
                </div>
              </div>

              <div class="metric-box">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="16" height="16">
                    <path
                      fill="currentColor"
                      d="M4 8h4V4H4v4zm6 12h4v-4h-4v4zm-6 0h4v-4H4v4zm0-6h4v-4H4v4zm6 0h4v-4h-4v4zm6-10v4h4V4h-4zm-6 4h4V4h-4v4zm6 6h4v-4h-4v4zm0 6h4v-4h-4v4z"
                    />
                  </svg>
                  <span>总容量</span>
                </div>
                <div class="box-value">
                  <span class="value">{{ formatCores(resourceData?.cpu?.capacity) }}</span>
                </div>
              </div>
            </div>

            <div class="chart-area">
              <div class="chart-header">
                <span class="chart-title">CPU 使用趋势</span>
                <div class="chart-legend">
                  <span class="legend-item"><i class="dot usage"></i>使用率</span>
                  <span class="legend-item"><i class="dot request"></i>请求率</span>
                </div>
              </div>
              <div ref="cpuChartRef" class="chart-container"></div>
            </div>
          </div>

          <!-- 内存卡片 -->
          <div class="monitor-card">
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
                  <h3>内存资源</h3>
                  <p>内存资源使用详情</p>
                </div>
              </div>
              <div class="status-tag" :class="getMemoryStatus.class">{{
                getMemoryStatus.text
              }}</div>
            </div>

            <div class="metrics-content">
              <div class="metric-box primary">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="16" height="16">
                    <path
                      fill="currentColor"
                      d="M13 2.05v2.02c3.95.49 7 3.85 7 7.93s-3.05 7.44-7 7.93v2.02c5.05-.5 9-4.76 9-9.95s-3.95-9.45-9-9.95z"
                    />
                  </svg>
                  <span>使用率</span>
                </div>
                <div class="box-value">
                  <span class="value">{{
                    formatPercentage(resourceData?.memory?.usagePercent)
                  }}</span>
                </div>
                <div class="box-progress memory">
                  <div
                    class="progress-fill"
                    :class="getUsageLevelClass(resourceData?.memory?.usagePercent)"
                    :style="{ width: `${Math.min(resourceData?.memory?.usagePercent || 0, 100)}%` }"
                  ></div>
                </div>
              </div>

              <div class="metric-box">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="16" height="16">
                    <path
                      fill="currentColor"
                      d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10 10-4.5 10-10S17.5 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8z"
                    />
                  </svg>
                  <span>已使用</span>
                </div>
                <div class="box-value">
                  <span class="value">{{ formatBytes(resourceData?.memory?.usage) }}</span>
                </div>
              </div>

              <div class="metric-box">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="16" height="16">
                    <path
                      fill="currentColor"
                      d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V5h14v14z"
                    />
                  </svg>
                  <span>可分配</span>
                </div>
                <div class="box-value">
                  <span class="value">{{ formatBytes(resourceData?.memory?.allocatable) }}</span>
                </div>
              </div>

              <div class="metric-box">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="16" height="16">
                    <path
                      fill="currentColor"
                      d="M4 8h4V4H4v4zm6 12h4v-4h-4v4zm-6 0h4v-4H4v4zm0-6h4v-4H4v4zm6 0h4v-4h-4v4zm6-10v4h4V4h-4zm-6 4h4V4h-4v4zm6 6h4v-4h-4v4zm0 6h4v-4h-4v4z"
                    />
                  </svg>
                  <span>总容量</span>
                </div>
                <div class="box-value">
                  <span class="value">{{ formatBytes(resourceData?.memory?.capacity) }}</span>
                </div>
              </div>
            </div>

            <div class="chart-area">
              <div class="chart-header">
                <span class="chart-title">内存使用趋势</span>
                <div class="chart-legend">
                  <span class="legend-item"><i class="dot usage"></i>使用率</span>
                  <span class="legend-item"><i class="dot request"></i>请求率</span>
                </div>
              </div>
              <div ref="memoryChartRef" class="chart-container"></div>
            </div>
          </div>
        </div>

        <!-- Pod 和存储卡片 -->
        <div class="metrics-grid">
          <!-- Pod 卡片 -->
          <div class="monitor-card">
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
                  <h3>Pod 资源</h3>
                  <p>Pod 运行状态统计</p>
                </div>
              </div>
              <div class="status-tag" :class="getPodStatus.class">{{ getPodStatus.text }}</div>
            </div>

            <div class="pod-overview">
              <div class="pod-main-stat">
                <div class="main-value">{{ formatLargeNumber(resourceData?.pods?.running) }}</div>
                <div class="main-label">运行中 Pod</div>
                <div class="main-sub"
                  >总计 {{ formatLargeNumber(resourceData?.pods?.capacity) }} 个容量</div
                >
              </div>
              <div class="pod-progress-ring">
                <svg viewBox="0 0 120 120">
                  <circle cx="60" cy="60" r="50" fill="none" stroke="#e5e7eb" stroke-width="10" />
                  <circle
                    cx="60"
                    cy="60"
                    r="50"
                    fill="none"
                    stroke="#10b981"
                    stroke-width="10"
                    :stroke-dasharray="`${(resourceData?.pods?.usagePercent || 0) * 3.14} 314`"
                    stroke-linecap="round"
                    transform="rotate(-90 60 60)"
                  />
                </svg>
                <div class="ring-value">{{
                  formatPercentage(resourceData?.pods?.usagePercent)
                }}</div>
              </div>
            </div>

            <div class="pod-stats-bar">
              <div class="stats-item success">
                <div class="stats-icon"
                  ><svg viewBox="0 0 24 24" width="16" height="16">
                    <path fill="currentColor" d="M8 5v14l11-7z" /></svg
                ></div>
                <div class="stats-info">
                  <span class="stats-value">{{
                    formatLargeNumber(resourceData?.pods?.running)
                  }}</span>
                  <span class="stats-label">运行中</span>
                </div>
              </div>
              <div class="stats-item warning">
                <div class="stats-icon"
                  ><svg viewBox="0 0 24 24" width="16" height="16">
                    <path
                      fill="currentColor"
                      d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10 10-4.5 10-10S17.5 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67V7z"
                    /></svg
                ></div>
                <div class="stats-info">
                  <span class="stats-value">{{
                    formatLargeNumber(resourceData?.pods?.pending)
                  }}</span>
                  <span class="stats-label">待调度</span>
                </div>
              </div>
              <div class="stats-item danger">
                <div class="stats-icon"
                  ><svg viewBox="0 0 24 24" width="16" height="16">
                    <path
                      fill="currentColor"
                      d="M12 2C6.47 2 2 6.47 2 12s4.47 10 10 10 10-4.47 10-10S17.53 2 12 2zm5 13.59L15.59 17 12 13.41 8.41 17 7 15.59 10.59 12 7 8.41 8.41 7 12 10.59 15.59 7 17 8.41 13.41 12 17 15.59z"
                    /></svg
                ></div>
                <div class="stats-info">
                  <span class="stats-value">{{
                    formatLargeNumber(resourceData?.pods?.failed)
                  }}</span>
                  <span class="stats-label">失败</span>
                </div>
              </div>
              <div class="stats-item info">
                <div class="stats-icon"
                  ><svg viewBox="0 0 24 24" width="16" height="16">
                    <path
                      fill="currentColor"
                      d="M4 8h4V4H4v4zm6 12h4v-4h-4v4zm-6 0h4v-4H4v4zm0-6h4v-4H4v4zm6 0h4v-4h-4v4zm6-10v4h4V4h-4zm-6 4h4V4h-4v4zm6 6h4v-4h-4v4zm0 6h4v-4h-4v4z"
                    /></svg
                ></div>
                <div class="stats-info">
                  <span class="stats-value">{{
                    formatLargeNumber(resourceData?.pods?.capacity)
                  }}</span>
                  <span class="stats-label">总容量</span>
                </div>
              </div>
            </div>
          </div>

          <!-- 存储卡片 -->
          <div class="monitor-card">
            <div class="card-header">
              <div class="header-info">
                <div class="icon-badge storage">
                  <svg viewBox="0 0 24 24" width="22" height="22">
                    <path
                      fill="currentColor"
                      d="M6,2H18A2,2 0 0,1 20,4V20A2,2 0 0,1 18,22H6A2,2 0 0,1 4,20V4A2,2 0 0,1 6,2M12,4A6,6 0 0,0 6,10C6,13.31 8.69,16 12,16A6,6 0 0,0 18,10C18,6.69 15.31,4 12,4M12,9A1,1 0 0,1 13,10A1,1 0 0,1 12,11A1,1 0 0,1 11,10A1,1 0 0,1 12,9M7,18A1,1 0 0,0 6,19A1,1 0 0,0 7,20A1,1 0 0,0 8,19A1,1 0 0,0 7,18Z"
                    />
                  </svg>
                </div>
                <div>
                  <h3>存储资源</h3>
                  <p>存储容量使用详情</p>
                </div>
              </div>
              <div class="status-tag" :class="getStorageStatus.class">{{
                getStorageStatus.text
              }}</div>
            </div>

            <div class="storage-overview">
              <div class="storage-main-stat">
                <div class="main-value">{{
                  formatBytes(resourceData?.storage?.totalUsageBytes)
                }}</div>
                <div class="main-label">已使用存储</div>
                <div class="main-sub"
                  >总容量 {{ formatBytes(resourceData?.storage?.totalCapacityBytes) }}</div
                >
              </div>
              <div class="storage-progress-ring">
                <svg viewBox="0 0 120 120">
                  <circle cx="60" cy="60" r="50" fill="none" stroke="#e5e7eb" stroke-width="10" />
                  <circle
                    cx="60"
                    cy="60"
                    r="50"
                    fill="none"
                    stroke="#f59e0b"
                    stroke-width="10"
                    :stroke-dasharray="`${storageUsagePercent * 3.14} 314`"
                    stroke-linecap="round"
                    transform="rotate(-90 60 60)"
                  />
                </svg>
                <div class="ring-value">{{ formatPercentage(storageUsagePercent) }}</div>
              </div>
            </div>

            <div class="storage-stats-bar">
              <div class="stats-item">
                <div class="stats-icon storage-icon"
                  ><svg viewBox="0 0 24 24" width="16" height="16">
                    <path
                      fill="currentColor"
                      d="M2 20h20v-4H2v4zm2-3h2v2H4v-2zM2 4v4h20V4H2zm4 3H4V5h2v2zm-4 7h20v-4H2v4zm2-3h2v2H4v-2z"
                    /></svg
                ></div>
                <div class="stats-info">
                  <span class="stats-value">{{
                    formatBytes(resourceData?.storage?.totalCapacityBytes)
                  }}</span>
                  <span class="stats-label">总容量</span>
                </div>
              </div>
              <div class="stats-item">
                <div class="stats-icon used-icon"
                  ><svg viewBox="0 0 24 24" width="16" height="16">
                    <path
                      fill="currentColor"
                      d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"
                    /></svg
                ></div>
                <div class="stats-info">
                  <span class="stats-value">{{
                    formatBytes(resourceData?.storage?.totalUsageBytes)
                  }}</span>
                  <span class="stats-label">已使用</span>
                </div>
              </div>
              <div class="stats-item">
                <div class="stats-icon free-icon"
                  ><svg viewBox="0 0 24 24" width="16" height="16">
                    <path
                      fill="currentColor"
                      d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V5h14v14z"
                    /></svg
                ></div>
                <div class="stats-info">
                  <span class="stats-value">{{ formatBytes(storageFreeBytes) }}</span>
                  <span class="stats-label">可用</span>
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
              d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14h-2V9h2v8zm0-10h-2V5h2v2z"
            />
          </svg>
        </div>
        <h3>暂无资源监控数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, onUnmounted, nextTick, computed } from 'vue'
  import * as echarts from 'echarts'
  import type { ClusterResourceMetrics } from '@/api/console/cluster-monitor'
  import { formatBytes, formatCores, formatPercentage } from '@/utils/format'

  interface Props {
    resourceData?: ClusterResourceMetrics
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    resourceData: undefined,
    loading: false
  })

  const cpuChartRef = ref<HTMLElement>()
  const memoryChartRef = ref<HTMLElement>()
  let cpuChart: echarts.ECharts | null = null
  let memoryChart: echarts.ECharts | null = null
  let resizeTimer: NodeJS.Timeout | null = null

  const hasData = computed(() => {
    return !!(props.resourceData && Object.keys(props.resourceData).length > 0)
  })

  // 格式化大数字
  const formatLargeNumber = (value?: number): string => {
    if (value === undefined || value === null || isNaN(value)) return '0'
    if (value < 1000) return String(value)
    if (value < 10000) return `${(value / 1000).toFixed(1)}K`
    if (value < 1000000) return `${(value / 1000).toFixed(0)}K`
    if (value < 10000000) return `${(value / 1000000).toFixed(1)}M`
    return `${(value / 1000000).toFixed(0)}M`
  }

  // 存储使用百分比
  const storageUsagePercent = computed(() => {
    const total = props.resourceData?.storage?.totalCapacityBytes || 0
    const used = props.resourceData?.storage?.totalUsageBytes || 0
    if (total === 0) return 0
    return (used / total) * 100
  })

  // 存储可用空间
  const storageFreeBytes = computed(() => {
    const total = props.resourceData?.storage?.totalCapacityBytes || 0
    const used = props.resourceData?.storage?.totalUsageBytes || 0
    return Math.max(0, total - used)
  })

  // 资源健康状态
  const resourceHealthStatus = computed(() => {
    if (!props.resourceData) return { status: 'unknown', text: '未知', score: 0 }
    let score = 100
    const cpuPercent = props.resourceData.cpu?.usagePercent || 0
    const memPercent = props.resourceData.memory?.usagePercent || 0
    const podPercent = props.resourceData.pods?.usagePercent || 0

    if (cpuPercent > 90) score -= 25
    else if (cpuPercent > 80) score -= 15
    else if (cpuPercent > 70) score -= 5

    if (memPercent > 90) score -= 25
    else if (memPercent > 80) score -= 15
    else if (memPercent > 70) score -= 5

    if (podPercent > 90) score -= 15
    else if (podPercent > 80) score -= 10

    score = Math.max(0, score)
    let status: 'healthy' | 'warning' | 'critical' | 'unknown' = 'healthy'
    let text = '健康'
    if (score < 60) {
      status = 'critical'
      text = '严重异常'
    } else if (score < 80) {
      status = 'warning'
      text = '需要关注'
    }
    return { status, text, score: Math.round(score) }
  })

  // CPU状态
  const getCPUStatus = computed(() => {
    const percent = props.resourceData?.cpu?.usagePercent || 0
    if (percent > 80) return { class: 'danger', text: '过高' }
    if (percent > 60) return { class: 'warning', text: '偏高' }
    return { class: 'success', text: '正常' }
  })

  // 内存状态
  const getMemoryStatus = computed(() => {
    const percent = props.resourceData?.memory?.usagePercent || 0
    if (percent > 80) return { class: 'danger', text: '过高' }
    if (percent > 60) return { class: 'warning', text: '偏高' }
    return { class: 'success', text: '正常' }
  })

  // Pod状态
  const getPodStatus = computed(() => {
    const failed = props.resourceData?.pods?.failed || 0
    const pending = props.resourceData?.pods?.pending || 0
    if (failed > 0) return { class: 'danger', text: '有失败' }
    if (pending > 10) return { class: 'warning', text: '待调度' }
    return { class: 'success', text: '正常' }
  })

  // 存储状态
  const getStorageStatus = computed(() => {
    if (storageUsagePercent.value > 85) return { class: 'danger', text: '空间紧张' }
    if (storageUsagePercent.value > 70) return { class: 'warning', text: '空间偏紧' }
    return { class: 'success', text: '正常' }
  })

  const getUsageLevelClass = (percent?: number): string => {
    if (!percent) return 'level-normal'
    if (percent < 60) return 'level-normal'
    if (percent < 80) return 'level-warning'
    return 'level-danger'
  }

  const initCPUChart = () => {
    if (!cpuChartRef.value) return
    cpuChart = echarts.init(cpuChartRef.value)
    cpuChart.setOption({
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(255,255,255,0.98)',
        borderColor: '#eaecef',
        borderWidth: 1,
        padding: [12, 16],
        textStyle: { color: '#1f2937', fontSize: 12 }
      },
      grid: { left: '3%', right: '4%', bottom: '8%', top: '12%', containLabel: true },
      xAxis: {
        type: 'time',
        boundaryGap: false,
        axisLine: { lineStyle: { color: '#e5e7eb' } },
        axisLabel: {
          color: '#6b7280',
          fontSize: 11,
          formatter: (v: number) =>
            new Date(v).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
        },
        splitLine: { show: true, lineStyle: { color: '#f3f4f6', type: 'dashed' } }
      },
      yAxis: {
        type: 'value',
        name: '使用率 (%)',
        max: 100,
        nameTextStyle: { color: '#6b7280', fontSize: 11 },
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: { color: '#6b7280', fontSize: 11, formatter: '{value}%' },
        splitLine: { lineStyle: { color: '#f3f4f6', type: 'dashed' } }
      },
      series: []
    })
  }

  const updateCPUChart = () => {
    if (!cpuChart || !props.resourceData?.cpu?.trend) return
    const trend = props.resourceData.cpu.trend
    if (trend.length === 0) return
    cpuChart.setOption({
      series: [
        {
          name: '使用率',
          type: 'line',
          data: trend.map((i) => [i.timestamp * 1000, i.usagePercent]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#667eea' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(102,126,234,0.2)' },
              { offset: 1, color: 'rgba(102,126,234,0.02)' }
            ])
          }
        },
        {
          name: '请求率',
          type: 'line',
          data: trend.map((i) => [i.timestamp * 1000, i.requestsPercent]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#f5576c', type: 'dashed' }
        }
      ]
    })
  }

  const initMemoryChart = () => {
    if (!memoryChartRef.value) return
    memoryChart = echarts.init(memoryChartRef.value)
    memoryChart.setOption({
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(255,255,255,0.98)',
        borderColor: '#eaecef',
        borderWidth: 1,
        padding: [12, 16],
        textStyle: { color: '#1f2937', fontSize: 12 }
      },
      grid: { left: '3%', right: '4%', bottom: '8%', top: '12%', containLabel: true },
      xAxis: {
        type: 'time',
        boundaryGap: false,
        axisLine: { lineStyle: { color: '#e5e7eb' } },
        axisLabel: {
          color: '#6b7280',
          fontSize: 11,
          formatter: (v: number) =>
            new Date(v).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
        },
        splitLine: { show: true, lineStyle: { color: '#f3f4f6', type: 'dashed' } }
      },
      yAxis: {
        type: 'value',
        name: '使用率 (%)',
        max: 100,
        nameTextStyle: { color: '#6b7280', fontSize: 11 },
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: { color: '#6b7280', fontSize: 11, formatter: '{value}%' },
        splitLine: { lineStyle: { color: '#f3f4f6', type: 'dashed' } }
      },
      series: []
    })
  }

  const updateMemoryChart = () => {
    if (!memoryChart || !props.resourceData?.memory?.trend) return
    const trend = props.resourceData.memory.trend
    if (trend.length === 0) return
    memoryChart.setOption({
      series: [
        {
          name: '使用率',
          type: 'line',
          data: trend.map((i) => [i.timestamp * 1000, i.usagePercent]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#f093fb' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(240,147,251,0.2)' },
              { offset: 1, color: 'rgba(240,147,251,0.02)' }
            ])
          }
        },
        {
          name: '请求率',
          type: 'line',
          data: trend.map((i) => [i.timestamp * 1000, i.requestsPercent]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#f5576c', type: 'dashed' }
        }
      ]
    })
  }

  const handleResize = () => {
    if (resizeTimer) clearTimeout(resizeTimer)
    resizeTimer = setTimeout(() => {
      cpuChart?.resize()
      memoryChart?.resize()
    }, 300)
  }

  watch(
    () => props.resourceData,
    () => {
      nextTick(() => {
        if (hasData.value) {
          if (!cpuChart) initCPUChart()
          if (!memoryChart) initMemoryChart()
          updateCPUChart()
          updateMemoryChart()
        }
      })
    },
    { deep: true, immediate: true }
  )

  onMounted(() => {
    window.addEventListener('resize', handleResize)
  })
  onUnmounted(() => {
    window.removeEventListener('resize', handleResize)
    if (resizeTimer) clearTimeout(resizeTimer)
    cpuChart?.dispose()
    memoryChart?.dispose()
  })
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

  .resource-monitor {
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
        margin-bottom: 20px;
        padding-bottom: 16px;
        border-bottom: 1px solid $gray-200;
        .header-left {
          display: flex;
          align-items: center;
          gap: 14px;
          .header-icon {
            width: 44px;
            height: 44px;
            border-radius: $radius-sm;
            background: linear-gradient(135deg, #e0e7ff, #c7d2fe);
            display: flex;
            align-items: center;
            justify-content: center;
            color: $primary;
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
        .health-badge {
          display: flex;
          align-items: center;
          gap: 10px;
          padding: 10px 18px;
          border-radius: 24px;
          font-weight: 500;
          font-size: 14px;
          .status-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            animation: pulse 2s infinite;
          }
          .health-score {
            margin-left: 6px;
            font-weight: 700;
          }
          &.healthy {
            background: rgba($success, 0.1);
            color: #059669;
            .status-dot {
              background: $success;
            }
          }
          &.warning {
            background: rgba($warning, 0.1);
            color: #d97706;
            .status-dot {
              background: $warning;
            }
          }
          &.critical {
            background: rgba($danger, 0.1);
            color: #dc2626;
            .status-dot {
              background: $danger;
            }
          }
        }
      }
      .quick-metrics {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 16px;
        @media (max-width: 768px) {
          grid-template-columns: 1fr;
        }
        .quick-metric {
          display: flex;
          align-items: center;
          gap: 14px;
          padding: 18px;
          border-radius: $radius-sm;
          background: $gray-50;
          border: 1px solid $gray-200;
          transition: all 0.2s ease;
          &:hover {
            border-color: $gray-300;
          }
          .metric-icon {
            width: 48px;
            height: 48px;
            border-radius: $radius-xs;
            display: flex;
            align-items: center;
            justify-content: center;
            flex-shrink: 0;
            &.cpu {
              background: linear-gradient(135deg, #e0e7ff, #c7d2fe);
              color: #667eea;
            }
            &.memory {
              background: linear-gradient(135deg, #fce7f3, #fbcfe8);
              color: #f093fb;
            }
            &.pod {
              background: linear-gradient(135deg, #d1fae5, #a7f3d0);
              color: $success;
            }
          }
          .metric-info {
            .metric-label {
              font-size: 12px;
              color: $gray-500;
              margin-bottom: 4px;
              display: block;
            }
            .metric-value {
              font-size: 20px;
              font-weight: 700;
              color: $gray-800;
              display: block;
            }
            .metric-sub {
              font-size: 11px;
              color: $gray-400;
              margin-top: 2px;
              display: block;
            }
          }
        }
      }
    }

    .metrics-grid {
      display: grid;
      grid-template-columns: repeat(2, 1fr);
      gap: 20px;
      @media (max-width: 1200px) {
        grid-template-columns: 1fr;
      }
    }

    .monitor-card {
      background: #fff;
      border-radius: $radius;
      padding: 24px;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
      .card-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 20px;
        padding-bottom: 16px;
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
            &.cpu {
              background: linear-gradient(135deg, #667eea, #764ba2);
            }
            &.memory {
              background: linear-gradient(135deg, #f093fb, #f5576c);
            }
            &.pod {
              background: linear-gradient(135deg, #43e97b, #38f9d7);
            }
            &.storage {
              background: linear-gradient(135deg, #f59e0b, #d97706);
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

      .metrics-content {
        display: grid;
        grid-template-columns: repeat(4, 1fr);
        gap: 12px;
        margin-bottom: 20px;
        @media (max-width: 900px) {
          grid-template-columns: repeat(2, 1fr);
        }
      }

      .metric-box {
        padding: 14px;
        border-radius: $radius-sm;
        background: $gray-50;
        border: 1px solid $gray-200;
        transition: all 0.2s ease;
        &:hover {
          border-color: $gray-300;
        }
        &.primary {
          background: rgba($primary, 0.04);
          border-color: rgba($primary, 0.15);
        }
        .box-header {
          display: flex;
          align-items: center;
          gap: 6px;
          margin-bottom: 8px;
          color: $gray-500;
          span {
            font-size: 12px;
            font-weight: 500;
          }
        }
        .box-value {
          display: flex;
          align-items: baseline;
          gap: 4px;
          .value {
            font-size: 20px;
            font-weight: 700;
            color: $gray-800;
          }
        }
        .box-progress {
          margin-top: 10px;
          height: 4px;
          background: $gray-200;
          border-radius: 2px;
          overflow: hidden;
          .progress-fill {
            height: 100%;
            transition: width 0.4s ease;
            &.level-normal {
              background: $success;
            }
            &.level-warning {
              background: $warning;
            }
            &.level-danger {
              background: $danger;
            }
          }
        }
      }

      .chart-area {
        padding: 18px;
        background: $gray-50;
        border-radius: $radius-sm;
        .chart-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 14px;
          .chart-title {
            font-size: 13px;
            font-weight: 600;
            color: $gray-600;
          }
          .chart-legend {
            display: flex;
            gap: 14px;
            .legend-item {
              display: flex;
              align-items: center;
              gap: 5px;
              font-size: 11px;
              color: $gray-500;
              .dot {
                width: 8px;
                height: 3px;
                border-radius: 2px;
                &.usage {
                  background: #667eea;
                }
                &.request {
                  background: #f5576c;
                }
              }
            }
          }
        }
        .chart-container {
          height: 200px;
          width: 100%;
        }
      }

      .pod-overview,
      .storage-overview {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 20px;
        background: $gray-50;
        border-radius: $radius-sm;
        margin-bottom: 20px;
        .pod-main-stat,
        .storage-main-stat {
          .main-value {
            font-size: 36px;
            font-weight: 700;
            color: $gray-800;
            line-height: 1.2;
          }
          .main-label {
            font-size: 14px;
            color: $gray-600;
            margin-top: 4px;
          }
          .main-sub {
            font-size: 12px;
            color: $gray-400;
            margin-top: 2px;
          }
        }
        .pod-progress-ring,
        .storage-progress-ring {
          position: relative;
          width: 100px;
          height: 100px;
          svg {
            width: 100%;
            height: 100%;
          }
          .ring-value {
            position: absolute;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            font-size: 16px;
            font-weight: 700;
            color: $gray-800;
          }
        }
      }

      .pod-stats-bar,
      .storage-stats-bar {
        display: grid;
        grid-template-columns: repeat(4, 1fr);
        gap: 12px;
        @media (max-width: 900px) {
          grid-template-columns: repeat(2, 1fr);
        }
        .stats-item {
          display: flex;
          align-items: center;
          gap: 10px;
          padding: 12px;
          background: $gray-50;
          border-radius: $radius-xs;
          border: 1px solid $gray-200;
          &.success {
            border-color: rgba($success, 0.3);
            .stats-icon {
              background: rgba($success, 0.1);
              color: $success;
            }
          }
          &.warning {
            border-color: rgba($warning, 0.3);
            .stats-icon {
              background: rgba($warning, 0.1);
              color: $warning;
            }
          }
          &.danger {
            border-color: rgba($danger, 0.3);
            .stats-icon {
              background: rgba($danger, 0.1);
              color: $danger;
            }
          }
          &.info {
            border-color: rgba($primary, 0.3);
            .stats-icon {
              background: rgba($primary, 0.1);
              color: $primary;
            }
          }
          .stats-icon {
            width: 32px;
            height: 32px;
            border-radius: $radius-xs;
            display: flex;
            align-items: center;
            justify-content: center;
            background: $gray-100;
            color: $gray-500;
            &.storage-icon {
              background: rgba($primary, 0.1);
              color: $primary;
            }
            &.used-icon {
              background: rgba($warning, 0.1);
              color: $warning;
            }
            &.free-icon {
              background: rgba($success, 0.1);
              color: $success;
            }
          }
          .stats-info {
            .stats-value {
              font-size: 16px;
              font-weight: 700;
              color: $gray-800;
              display: block;
            }
            .stats-label {
              font-size: 11px;
              color: $gray-500;
            }
          }
        }
      }

      .storage-stats-bar {
        grid-template-columns: repeat(3, 1fr);
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

  @keyframes pulse {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0.5;
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
