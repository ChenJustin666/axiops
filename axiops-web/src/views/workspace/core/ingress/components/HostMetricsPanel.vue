<template>
  <div v-loading="loading" class="host-metrics-panel">
    <!-- 空状态 -->
    <ElEmpty v-if="!hostMetrics && !loading" description="暂无监控数据" :image-size="120" />

    <!-- 监控内容 -->
    <div v-else-if="hostMetrics" class="metrics-content">
      <!-- 时间范围选择 -->
      <div v-if="showTimeRangeSelector" class="time-selector">
        <ElRadioGroup v-model="timeRange" size="small" @change="handleTimeRangeChange">
          <ElRadioButton label="1h">1小时</ElRadioButton>
          <ElRadioButton label="6h">6小时</ElRadioButton>
          <ElRadioButton label="24h">24小时</ElRadioButton>
          <ElRadioButton label="7d">7天</ElRadioButton>
        </ElRadioGroup>
        <ElButton size="small" :icon="RefreshRight" @click="refresh">刷新</ElButton>
      </div>

      <!-- 核心指标卡片 -->
      <div class="metrics-cards">
        <div class="metric-card primary">
          <div class="metric-icon">
            <svg viewBox="0 0 24 24" width="20" height="20">
              <path
                fill="currentColor"
                d="M16 6l2.29 2.29-4.88 4.88-4-4L2 16.59 3.41 18l6-6 4 4 6.3-6.29L22 12V6z"
              />
            </svg>
          </div>
          <div class="metric-content">
            <div class="metric-label">请求率</div>
            <div class="metric-value">{{ formatQPS(hostMetrics.current.requestsPerSecond) }}</div>
          </div>
        </div>

        <div class="metric-card success">
          <div class="metric-icon">
            <svg viewBox="0 0 24 24" width="20" height="20">
              <path
                fill="currentColor"
                d="M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67z"
              />
            </svg>
          </div>
          <div class="metric-content">
            <div class="metric-label">P95延迟</div>
            <div class="metric-value" :class="getLatencyClass(hostMetrics.latency.p95)">
              {{ hostMetrics.latency.p95.toFixed(0) }} ms
            </div>
          </div>
        </div>

        <div class="metric-card warning">
          <div class="metric-icon">
            <svg viewBox="0 0 24 24" width="20" height="20">
              <path
                fill="currentColor"
                d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"
              />
            </svg>
          </div>
          <div class="metric-content">
            <div class="metric-label">错误率</div>
            <div
              class="metric-value"
              :class="getErrorRateClass(hostMetrics.errorRate.totalErrorRate)"
            >
              {{ (hostMetrics.errorRate.totalErrorRate * 100).toFixed(2) }}%
            </div>
          </div>
        </div>

        <div class="metric-card info">
          <div class="metric-icon">
            <svg viewBox="0 0 24 24" width="20" height="20">
              <path
                fill="currentColor"
                d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"
              />
            </svg>
          </div>
          <div class="metric-content">
            <div class="metric-label">活跃连接</div>
            <div class="metric-value">{{
              formatNumber(hostMetrics.current.activeConnections)
            }}</div>
          </div>
        </div>
      </div>

      <!-- 详细统计 -->
      <ElCollapse v-model="activeNames" class="metrics-collapse">
        <ElCollapseItem title="延迟分布" name="latency">
          <div class="latency-stats">
            <div class="latency-row">
              <span class="label">P50</span>
              <ElProgress
                :percentage="(hostMetrics.latency.p50 / hostMetrics.latency.max) * 100"
                :show-text="false"
                :stroke-width="8"
              />
              <span class="value">{{ hostMetrics.latency.p50.toFixed(2) }} ms</span>
            </div>
            <div class="latency-row">
              <span class="label">P95</span>
              <ElProgress
                :percentage="(hostMetrics.latency.p95 / hostMetrics.latency.max) * 100"
                :show-text="false"
                :stroke-width="8"
                color="#e6a23c"
              />
              <span class="value">{{ hostMetrics.latency.p95.toFixed(2) }} ms</span>
            </div>
            <div class="latency-row">
              <span class="label">P99</span>
              <ElProgress
                :percentage="(hostMetrics.latency.p99 / hostMetrics.latency.max) * 100"
                :show-text="false"
                :stroke-width="8"
                color="#f56c6c"
              />
              <span class="value">{{ hostMetrics.latency.p99.toFixed(2) }} ms</span>
            </div>
            <div class="latency-row">
              <span class="label">平均</span>
              <ElProgress
                :percentage="(hostMetrics.latency.avg / hostMetrics.latency.max) * 100"
                :show-text="false"
                :stroke-width="8"
                color="#67c23a"
              />
              <span class="value">{{ hostMetrics.latency.avg.toFixed(2) }} ms</span>
            </div>
          </div>
        </ElCollapseItem>

        <ElCollapseItem title="状态码分布" name="statusCodes">
          <div class="status-codes">
            <div
              v-if="Object.keys(hostMetrics.statusCodes.status2xx).length > 0"
              class="code-group success"
            >
              <div class="group-title">2xx 成功</div>
              <div class="code-list">
                <ElTag
                  v-for="(count, code) in hostMetrics.statusCodes.status2xx"
                  :key="code"
                  size="small"
                  type="success"
                >
                  {{ code }}: {{ formatNumber(count) }}
                </ElTag>
              </div>
            </div>
            <div
              v-if="Object.keys(hostMetrics.statusCodes.status4xx).length > 0"
              class="code-group warning"
            >
              <div class="group-title">4xx 客户端错误</div>
              <div class="code-list">
                <ElTag
                  v-for="(count, code) in hostMetrics.statusCodes.status4xx"
                  :key="code"
                  size="small"
                  type="warning"
                >
                  {{ code }}: {{ formatNumber(count) }}
                </ElTag>
              </div>
            </div>
            <div
              v-if="Object.keys(hostMetrics.statusCodes.status5xx).length > 0"
              class="code-group danger"
            >
              <div class="group-title">5xx 服务器错误</div>
              <div class="code-list">
                <ElTag
                  v-for="(count, code) in hostMetrics.statusCodes.status5xx"
                  :key="code"
                  size="small"
                  type="danger"
                >
                  {{ code }}: {{ formatNumber(count) }}
                </ElTag>
              </div>
            </div>
          </div>
        </ElCollapseItem>

        <ElCollapseItem v-if="hostMetrics.byPath.length > 0" title="按路径统计" name="byPath">
          <div class="path-stats">
            <div v-for="path in hostMetrics.byPath.slice(0, 10)" :key="path.path" class="path-item">
              <div class="path-header">
                <span class="path-name">{{ path.path || '/' }}</span>
                <ElTag
                  size="small"
                  :type="
                    path.errorRate < 0.05 ? 'success' : path.errorRate < 0.1 ? 'warning' : 'danger'
                  "
                >
                  {{ (path.errorRate * 100).toFixed(2) }}%
                </ElTag>
              </div>
              <div class="path-metrics">
                <span>{{ formatQPS(path.requestsPerSecond) }}</span>
                <span>{{ path.p95Latency.toFixed(0) }} ms</span>
              </div>
            </div>
          </div>
        </ElCollapseItem>
      </ElCollapse>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted } from 'vue'
  import { ElMessage } from 'element-plus'
  import { RefreshRight } from '@element-plus/icons-vue'
  import type { HostMetrics } from '@/api'
  import { getHostMetricsApi } from '@/api'

  interface Props {
    clusterUuid: string
    host: string
    showTimeRangeSelector?: boolean
    autoLoad?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    showTimeRangeSelector: true,
    autoLoad: true
  })

  const loading = ref(false)
  const hostMetrics = ref<HostMetrics>()
  const timeRange = ref<string>('1h')
  const activeNames = ref<string[]>([])

  const loadMetrics = async () => {
    if (!props.clusterUuid || !props.host) {
      return
    }

    loading.value = true
    try {
      const { start, end } = getTimeRange()

      const response = await getHostMetricsApi({
        clusterUuid: props.clusterUuid,
        host: props.host,
        start,
        end
      })

      if (response.data) {
        hostMetrics.value = response.data
      } else {
        hostMetrics.value = undefined
      }
    } catch (error: any) {
      console.error('加载 Host 监控数据失败:', error)
      hostMetrics.value = undefined
    } finally {
      loading.value = false
    }
  }

  const getTimeRange = () => {
    const end = Date.now()
    let start = end - 60 * 60 * 1000

    switch (timeRange.value) {
      case '1h':
        start = end - 60 * 60 * 1000
        break
      case '6h':
        start = end - 6 * 60 * 60 * 1000
        break
      case '24h':
        start = end - 24 * 60 * 60 * 1000
        break
      case '7d':
        start = end - 7 * 24 * 60 * 60 * 1000
        break
    }

    return {
      start: start.toString(),
      end: end.toString()
    }
  }

  const handleTimeRangeChange = () => {
    loadMetrics()
  }

  const refresh = () => {
    loadMetrics()
  }

  const formatBytes = (bytes: number): string => {
    if (!bytes || bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const formatNumber = (num: number): string => {
    if (!num) return '0'
    if (num >= 1000000) return (num / 1000000).toFixed(2) + 'M'
    if (num >= 1000) return (num / 1000).toFixed(2) + 'K'
    return num.toString()
  }

  const formatQPS = (qps: number): string => {
    if (!qps) return '0 QPS'
    if (qps >= 1000) return (qps / 1000).toFixed(2) + 'K QPS'
    return qps.toFixed(2) + ' QPS'
  }

  const getErrorRateClass = (errorRate: number): string => {
    if (errorRate < 0.01) return 'success'
    if (errorRate < 0.05) return 'warning'
    return 'danger'
  }

  const getLatencyClass = (latency: number): string => {
    if (latency < 100) return 'success'
    if (latency < 500) return 'warning'
    return 'danger'
  }

  watch(
    () => [props.clusterUuid, props.host],
    () => {
      if (props.autoLoad) {
        loadMetrics()
      }
    }
  )

  onMounted(() => {
    if (props.autoLoad) {
      loadMetrics()
    }
  })

  defineExpose({
    refresh,
    loadMetrics
  })
</script>

<style lang="scss" scoped>
  .host-metrics-panel {
    .metrics-content {
      .time-selector {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 16px;
        padding: 12px;
        background: #f5f7fa;
        border-radius: 6px;
      }

      .metrics-cards {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
        gap: 12px;
        margin-bottom: 16px;

        .metric-card {
          display: flex;
          align-items: center;
          gap: 12px;
          padding: 16px;
          background: #fff;
          border-radius: 8px;
          box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
          transition: all 0.3s;

          &:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
          }

          .metric-icon {
            width: 40px;
            height: 40px;
            display: flex;
            align-items: center;
            justify-content: center;
            border-radius: 8px;
            color: #fff;
          }

          &.primary .metric-icon {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
          }

          &.success .metric-icon {
            background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
          }

          &.warning .metric-icon {
            background: linear-gradient(135deg, #fa709a 0%, #fee140 100%);
          }

          &.info .metric-icon {
            background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
          }

          .metric-content {
            flex: 1;

            .metric-label {
              font-size: 12px;
              color: #909399;
              margin-bottom: 4px;
            }

            .metric-value {
              font-size: 18px;
              font-weight: 600;
              color: #303133;

              &.success {
                color: #67c23a;
              }

              &.warning {
                color: #e6a23c;
              }

              &.danger {
                color: #f56c6c;
              }
            }
          }
        }
      }

      .metrics-collapse {
        :deep(.el-collapse-item__header) {
          font-weight: 500;
        }

        .latency-stats {
          display: flex;
          flex-direction: column;
          gap: 12px;

          .latency-row {
            display: grid;
            grid-template-columns: 40px 1fr 80px;
            gap: 12px;
            align-items: center;

            .label {
              font-size: 13px;
              font-weight: 500;
              color: #606266;
            }

            .value {
              font-size: 13px;
              font-weight: 600;
              color: #303133;
              text-align: right;
            }
          }
        }

        .status-codes {
          display: flex;
          flex-direction: column;
          gap: 12px;

          .code-group {
            padding: 12px;
            background: #f5f7fa;
            border-radius: 6px;
            border-left: 3px solid;

            &.success {
              border-left-color: #67c23a;
            }

            &.warning {
              border-left-color: #e6a23c;
            }

            &.danger {
              border-left-color: #f56c6c;
            }

            .group-title {
              font-size: 13px;
              font-weight: 600;
              color: #303133;
              margin-bottom: 8px;
            }

            .code-list {
              display: flex;
              flex-wrap: wrap;
              gap: 6px;
            }
          }
        }

        .path-stats {
          display: flex;
          flex-direction: column;
          gap: 8px;

          .path-item {
            padding: 12px;
            background: #f5f7fa;
            border-radius: 6px;
            transition: all 0.3s;

            &:hover {
              background: #ecf5ff;
            }

            .path-header {
              display: flex;
              justify-content: space-between;
              align-items: center;
              margin-bottom: 6px;

              .path-name {
                font-size: 14px;
                font-weight: 500;
                color: #303133;
                word-break: break-all;
              }
            }

            .path-metrics {
              display: flex;
              gap: 16px;
              font-size: 13px;
              color: #606266;
            }
          }
        }
      }
    }
  }
</style>
