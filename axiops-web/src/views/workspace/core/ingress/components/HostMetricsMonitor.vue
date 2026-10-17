<template>
  <ElDialog
    :model-value="visible"
    :title="host"
    width="90%"
    top="5vh"
    :close-on-click-modal="false"
    @close="handleClose"
    destroy-on-close
  >
    <div v-loading="loading" class="monitor-page">
      <ElEmpty v-if="!hostMetrics" description="暂无监控数据" />

      <template v-else>
        <!-- 工具栏 -->
        <div class="toolbar">
          <ElRadioGroup v-model="timeRange" size="small" @change="loadHostMetrics">
            <ElRadioButton label="1h">1小时</ElRadioButton>
            <ElRadioButton label="6h">6小时</ElRadioButton>
            <ElRadioButton label="24h">24小时</ElRadioButton>
            <ElRadioButton label="7d">7天</ElRadioButton>
          </ElRadioGroup>
          <ElSpace>
            <ElButton size="small" @click="loadHostMetrics">刷新</ElButton>
          </ElSpace>
        </div>

        <!-- 核心指标 -->
        <ElRow :gutter="16" class="metrics-row">
          <ElCol :span="6">
            <div class="metric-box primary">
              <div class="label">实时请求率</div>
              <div class="value">{{ formatQPS(hostMetrics.current.requestsPerSecond) }}</div>
              <div class="sub">峰值: {{ formatQPS(hostMetrics.summary.maxRequestsPerSec) }}</div>
            </div>
          </ElCol>
          <ElCol :span="6">
            <div class="metric-box success">
              <div class="label">活跃连接</div>
              <div class="value">{{ hostMetrics.current.activeConnections }}</div>
              <div class="sub"
                >入站: {{ formatBytes(hostMetrics.current.ingressBytesPerSec) }}/s</div
              >
            </div>
          </ElCol>
          <ElCol :span="6">
            <div class="metric-box warning">
              <div class="label">P95 延迟</div>
              <div class="value">{{ hostMetrics.latency.p95.toFixed(0) }} ms</div>
              <div class="sub">P99: {{ hostMetrics.latency.p99.toFixed(0) }} ms</div>
            </div>
          </ElCol>
          <ElCol :span="6">
            <div class="metric-box danger">
              <div class="label">总错误率</div>
              <div class="value"
                >{{ (hostMetrics.errorRate.totalErrorRate * 100).toFixed(2) }}%</div
              >
              <div class="sub"
                >5xx: {{ (hostMetrics.errorRate.error5xxRate * 100).toFixed(2) }}%</div
              >
            </div>
          </ElCol>
        </ElRow>

        <!-- 汇总统计 -->
        <div class="section">
          <div class="section-title">汇总统计</div>
          <ElRow :gutter="12">
            <ElCol :span="4" v-for="item in summaryList" :key="item.label">
              <div class="summary-item">
                <div class="label">{{ item.label }}</div>
                <div class="value">{{ item.value }}</div>
              </div>
            </ElCol>
          </ElRow>
        </div>

        <!-- 流量趋势 -->
        <div class="section">
          <div class="section-title">流量趋势</div>
          <TrendChart :data="hostMetrics.trend" height="200px" />
        </div>

        <!-- 延迟分布 和 状态码分布 - 并排，固定高度 -->
        <ElRow :gutter="16">
          <ElCol :span="12">
            <div class="section fixed-height">
              <div class="section-title">延迟分布</div>
              <div class="latency-list">
                <div v-for="item in latencyList" :key="item.label" class="latency-item">
                  <div class="header">
                    <span>{{ item.label }}</span>
                    <span>{{ item.value }} ms</span>
                  </div>
                  <ElProgress :percentage="item.percent" :show-text="false" :stroke-width="8" />
                </div>
              </div>
            </div>
          </ElCol>

          <ElCol :span="12">
            <div class="section fixed-height">
              <div class="section-title">状态码分布</div>
              <div class="status-list">
                <div v-for="group in statusGroups" :key="group.type" class="status-group">
                  <div class="group-title">{{ group.label }}</div>
                  <ElSpace wrap>
                    <ElTag
                      v-for="(count, code) in group.codes"
                      :key="code"
                      :type="group.tagType"
                      size="small"
                    >
                      {{ code }}: {{ formatNumber(count) }}
                    </ElTag>
                  </ElSpace>
                </div>
              </div>
            </div>
          </ElCol>
        </ElRow>

        <!-- Top 错误 和 路径统计 - 并排，固定最小高度 -->
        <ElRow :gutter="16">
          <ElCol :span="12" v-if="hostMetrics.errorRate.topErrors.length">
            <div class="section min-height">
              <div class="section-title">Top 错误</div>
              <div class="scroll-list">
                <div
                  v-for="error in hostMetrics.errorRate.topErrors"
                  :key="error.statusCode"
                  class="list-item"
                >
                  <div class="header">
                    <ElTag :type="getStatusCodeType(error.statusCode)" size="small">
                      {{ error.statusCode }}
                    </ElTag>
                    <span>{{ formatNumber(error.count) }} 次</span>
                  </div>
                  <ElProgress :percentage="error.percent" :show-text="true" :stroke-width="6" />
                </div>
              </div>
            </div>
          </ElCol>

          <ElCol :span="12" v-if="hostMetrics.byPath.length">
            <div class="section min-height">
              <div class="section-title">路径统计 (Top 10)</div>
              <div class="scroll-list">
                <div
                  v-for="path in hostMetrics.byPath.slice(0, 10)"
                  :key="path.path"
                  class="list-item"
                >
                  <div class="header">
                    <span class="path-name" :title="path.path">{{ path.path || '/' }}</span>
                    <ElTag
                      size="small"
                      :type="
                        path.errorRate < 0.05
                          ? 'success'
                          : path.errorRate < 0.1
                            ? 'warning'
                            : 'danger'
                      "
                    >
                      {{ (path.errorRate * 100).toFixed(2) }}%
                    </ElTag>
                  </div>
                  <div class="metrics">
                    <span>QPS: {{ formatQPS(path.requestsPerSecond) }}</span>
                    <span>P95: {{ path.p95Latency.toFixed(0) }} ms</span>
                  </div>
                </div>
              </div>
            </div>
          </ElCol>
        </ElRow>
      </template>
    </div>

    <template #footer>
      <ElButton @click="handleClose">关闭</ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, watch, computed } from 'vue'
  import { ElMessage } from 'element-plus'
  import type { HostMetrics } from '@/api'
  import { getHostMetricsApi } from '@/api'
  import TrendChart from './TrendChart.vue'

  interface Props {
    visible: boolean
    clusterUuid: string
    host: string
  }

  const props = defineProps<Props>()
  const emit = defineEmits<{ close: [] }>()

  const loading = ref(false)
  const hostMetrics = ref<HostMetrics>()
  const timeRange = ref<string>('1h')

  const loadHostMetrics = async () => {
    if (!props.visible || !props.clusterUuid || !props.host) return

    loading.value = true
    try {
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

      const response = await getHostMetricsApi({
        clusterUuid: props.clusterUuid,
        host: props.host,
        start: start.toString(),
        end: end.toString()
      })

      if (response.data) {
        hostMetrics.value = response.data
      }
    } finally {
      loading.value = false
    }
  }

  const handleClose = () => emit('close')

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

  const getStatusCodeType = (code: string): string => {
    if (code.startsWith('2')) return 'success'
    if (code.startsWith('4')) return 'warning'
    if (code.startsWith('5')) return 'danger'
    return 'info'
  }

  const summaryList = computed(() => {
    if (!hostMetrics.value) return []
    const s = hostMetrics.value.summary
    return [
      { label: '总请求数', value: formatNumber(s.totalRequests) },
      { label: '入站流量', value: formatBytes(s.totalIngressBytes) },
      { label: '出站流量', value: formatBytes(s.totalEgressBytes) },
      { label: '平均QPS', value: formatQPS(s.avgRequestsPerSec) },
      { label: '请求大小', value: formatBytes(s.avgRequestSize) },
      { label: '响应大小', value: formatBytes(s.avgResponseSize) }
    ]
  })

  const latencyList = computed(() => {
    if (!hostMetrics.value) return []
    const l = hostMetrics.value.latency
    const max = l.max || 1
    return [
      { label: 'P50', value: l.p50.toFixed(2), percent: (l.p50 / max) * 100 },
      { label: 'P95', value: l.p95.toFixed(2), percent: (l.p95 / max) * 100 },
      { label: 'P99', value: l.p99.toFixed(2), percent: (l.p99 / max) * 100 },
      { label: '平均', value: l.avg.toFixed(2), percent: (l.avg / max) * 100 }
    ]
  })

  const statusGroups = computed(() => {
    if (!hostMetrics.value) return []
    const s = hostMetrics.value.statusCodes
    return [
      { type: '2xx', label: '2xx 成功', tagType: 'success', codes: s.status2xx },
      { type: '3xx', label: '3xx 重定向', tagType: 'info', codes: s.status3xx },
      { type: '4xx', label: '4xx 错误', tagType: 'warning', codes: s.status4xx },
      { type: '5xx', label: '5xx 错误', tagType: 'danger', codes: s.status5xx }
    ]
  })

  watch(
    () => props.visible,
    (newVal) => {
      if (newVal) loadHostMetrics()
    }
  )
</script>

<style scoped>
  .monitor-page {
    padding: 16px;
  }

  .toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
    padding: 12px;
    background: #f5f7fa;
    border-radius: 4px;
  }

  .metrics-row {
    margin-bottom: 16px;
  }

  .metric-box {
    padding: 16px;
    border-radius: 4px;
    text-align: center;
  }

  .metric-box.primary {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: #fff;
  }

  .metric-box.success {
    background: linear-gradient(135deg, #84fab0 0%, #8fd3f4 100%);
    color: #fff;
  }

  .metric-box.warning {
    background: linear-gradient(135deg, #fccb90 0%, #d57eeb 100%);
    color: #fff;
  }

  .metric-box.danger {
    background: linear-gradient(135deg, #ff9a9e 0%, #fecfef 100%);
    color: #fff;
  }

  .metric-box .label {
    font-size: 12px;
    opacity: 0.9;
    margin-bottom: 8px;
  }

  .metric-box .value {
    font-size: 24px;
    font-weight: bold;
    margin-bottom: 4px;
  }

  .metric-box .sub {
    font-size: 12px;
    opacity: 0.8;
  }

  .section {
    background: #fff;
    border: 1px solid #ebeef5;
    border-radius: 4px;
    padding: 16px;
    margin-bottom: 16px;
  }

  .section.fixed-height {
    height: 380px;
    display: flex;
    flex-direction: column;
  }

  .section.min-height {
    min-height: 380px;
    display: flex;
    flex-direction: column;
  }

  .section-title {
    font-size: 14px;
    font-weight: 600;
    color: #303133;
    margin-bottom: 12px;
    padding-bottom: 8px;
    border-bottom: 1px solid #ebeef5;
    flex-shrink: 0;
  }

  .summary-item {
    padding: 12px;
    background: #f5f7fa;
    border-radius: 4px;
    text-align: center;
  }

  .summary-item .label {
    font-size: 12px;
    color: #909399;
    margin-bottom: 4px;
  }

  .summary-item .value {
    font-size: 16px;
    font-weight: 600;
    color: #303133;
  }

  .latency-list {
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .status-list {
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .scroll-list {
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .latency-item {
    padding: 10px 12px;
    background: #f5f7fa;
    border-radius: 4px;
    flex-shrink: 0;
  }

  .latency-item .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }

  .status-group {
    padding: 10px 12px;
    background: #f5f7fa;
    border-radius: 4px;
    flex-shrink: 0;
  }

  .status-group .group-title {
    font-size: 13px;
    font-weight: 600;
    margin-bottom: 8px;
    color: #303133;
  }

  .list-item {
    padding: 10px 12px;
    background: #f5f7fa;
    border-radius: 4px;
    flex-shrink: 0;
  }

  .list-item .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }

  .path-name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    padding-right: 8px;
  }

  .list-item .metrics {
    display: flex;
    gap: 16px;
    font-size: 12px;
    color: #606266;
  }
</style>
