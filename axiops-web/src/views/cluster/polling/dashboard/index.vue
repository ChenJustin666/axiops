<template>
  <div class="inspection-page">
    <ElCard shadow="never" class="panel-card dashboard-toolbar">
      <div class="toolbar-title">
        <strong>巡检大盘</strong>
        <span>{{ clusterTitle }} · {{ daysText }}</span>
      </div>
      <div class="panel-actions">
        <ElSelect
          v-model="selectedClusterId"
          filterable
          clearable
          placeholder="全部集群"
          style="width: 220px"
          :loading="clusterLoading"
          @change="handleClusterChange"
        >
          <ElOption v-for="item in clusters" :key="item.id" :label="item.name" :value="item.id" />
        </ElSelect>
        <ElSelect v-model="days" style="width: 118px" @change="refreshData">
          <ElOption label="近 7 天" :value="7" />
          <ElOption label="近 14 天" :value="14" />
          <ElOption label="近 30 天" :value="30" />
        </ElSelect>
        <ElButton type="primary" @click="goReport">报告中心</ElButton>
      </div>
    </ElCard>

    <div class="kpi-grid" v-loading="loading">
      <div class="kpi-card">
        <span>报告总数</span>
        <strong>{{ dashboard.totalRecords }}</strong>
        <em>成功 {{ dashboard.successRecords }} / 失败 {{ dashboard.failedRecords }}</em>
      </div>
      <div class="kpi-card">
        <span>平均得分</span>
        <strong>{{ dashboard.avgScore.toFixed(1) }}</strong>
        <em>最新 {{ healthText(dashboard.latestHealthLevel) }}</em>
      </div>
      <div class="kpi-card">
        <span>严重报告</span>
        <strong>{{ dashboard.criticalRecords }}</strong>
        <em>占比 {{ percent(dashboard.criticalRecords, dashboard.totalRecords) }}</em>
      </div>
      <div class="kpi-card">
        <span>检查总数</span>
        <strong>{{ latestRecord?.totalCount || 0 }}</strong>
        <em>最新报告</em>
      </div>
      <div class="kpi-card">
        <span>成功数</span>
        <strong>{{ latestRecord?.successCount || 0 }}</strong>
        <em>通过率 {{ latestPassRate }}</em>
      </div>
      <div class="kpi-card">
        <span>异常数</span>
        <strong>{{ latestRiskCount }}</strong>
        <em
          >警告 {{ latestRecord?.warningCount || 0 }} / 严重
          {{ latestRecord?.criticalCount || 0 }}</em
        >
      </div>
    </div>

    <div class="chart-grid">
      <ElCard shadow="never" class="panel-card chart-card">
        <template #header>
          <div class="card-header-row">
            <div class="card-title">得分趋势</div>
            <ElTag effect="plain">{{ trendRecords.length }} 次</ElTag>
          </div>
        </template>
        <div ref="scoreTrendChartRef" class="chart-box"></div>
      </ElCard>

      <ElCard shadow="never" class="panel-card chart-card">
        <template #header>
          <div class="card-title">状态分布</div>
        </template>
        <div ref="statusChartRef" class="chart-box"></div>
      </ElCard>

      <ElCard shadow="never" class="panel-card chart-card">
        <template #header>
          <div class="card-title">分类风险</div>
        </template>
        <div ref="categoryChartRef" class="chart-box"></div>
      </ElCard>

      <ElCard shadow="never" class="panel-card chart-card">
        <template #header>
          <div class="card-title">异常趋势</div>
        </template>
        <div ref="riskTrendChartRef" class="chart-box"></div>
      </ElCard>
    </div>

    <ElCard shadow="never" class="table-card">
      <ArtTableHeader
        v-model:columns="recordColumns"
        :loading="loading"
        :showZebra="true"
        :showBorder="true"
        :showHeaderBackground="true"
        :fullClass="'art-page-view'"
        :layout="'refresh,size,fullscreen,columns,settings'"
        @refresh="refreshData"
      >
        <template #left>
          <ElSpace wrap>
            <span class="card-title">最近巡检报告</span>
            <ElTag effect="plain" type="info">当前 {{ records.length }} 条</ElTag>
          </ElSpace>
        </template>
      </ArtTableHeader>
      <ArtTable
        :loading="loading"
        :data="records"
        :columns="visibleRecordColumns"
        :show-table-header="false"
        :height="360"
        empty-text="暂无报告"
      >
        <template #cluster="{ row }">{{
          row.clusterName || clusterName(row.clusterUuid)
        }}</template>
        <template #status="{ row }">
          <ElTag :type="statusTagType(row.status)">{{ statusText(row.status) }}</ElTag>
        </template>
        <template #health="{ row }">
          <ElTag :type="healthTagType(row.healthLevel)">{{ healthText(row.healthLevel) }}</ElTag>
        </template>
        <template #score="{ row }">{{ row.score.toFixed(1) }}</template>
        <template #risk="{ row }">{{ riskCount(row) }}</template>
        <template #createdAt="{ row }">{{ formatTime(row.createdAt) }}</template>
      </ArtTable>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import * as echarts from 'echarts'
  import {
    getInspectionDashboardApi,
    searchInspectionRecordsApi,
    type InspectionDashboardResponse,
    type InspectionRecord
  } from '@/api/manager/inspection'
  import {
    formatPercent,
    formatTime,
    healthText,
    statusTagType,
    statusText,
    healthTagType
  } from '../shared/constants'
  import { usePollingClusters } from '../shared/usePollingClusters'

  defineOptions({ name: 'InspectionDashboard' })

  const router = useRouter()
  const {
    clusters,
    loading: clusterLoading,
    selectedClusterId,
    selectedClusterUuid,
    loadClusters,
    setCluster
  } = usePollingClusters('inspection-dashboard-cluster')

  const loading = ref(false)
  const days = ref(7)
  const records = ref<InspectionRecord[]>([])
  const scoreTrendChartRef = ref<HTMLElement>()
  const statusChartRef = ref<HTMLElement>()
  const categoryChartRef = ref<HTMLElement>()
  const riskTrendChartRef = ref<HTMLElement>()
  let scoreTrendChart: echarts.ECharts | null = null
  let statusChart: echarts.ECharts | null = null
  let categoryChart: echarts.ECharts | null = null
  let riskTrendChart: echarts.ECharts | null = null

  const dashboard = reactive<InspectionDashboardResponse>({
    totalRecords: 0,
    successRecords: 0,
    failedRecords: 0,
    criticalRecords: 0,
    avgScore: 0,
    latestHealthLevel: 'unknown',
    latestRecordId: 0,
    categoryStats: []
  })
  const recordColumns = ref([
    { prop: 'recordNo', label: '报告编号', minWidth: 190, showOverflowTooltip: true },
    { prop: 'cluster', label: '集群', minWidth: 180, useSlot: true },
    { prop: 'status', label: '状态', width: 110, useSlot: true },
    { prop: 'health', label: '健康度', width: 110, useSlot: true },
    { prop: 'score', label: '得分', width: 90, useSlot: true },
    { prop: 'risk', label: '异常项', width: 110, useSlot: true },
    { prop: 'createdAt', label: '时间', width: 180, useSlot: true }
  ])
  const visibleRecordColumns = computed(() =>
    recordColumns.value.filter((column: any) => column.visible !== false)
  )

  const latestRecord = computed(() => records.value[0] || null)
  const latestRiskCount = computed(() => (latestRecord.value ? riskCount(latestRecord.value) : 0))
  const statusDistributionRows = computed(() => {
    const record = latestRecord.value
    if (!record) {
      return []
    }
    const rows = [
      { name: '成功', value: record.successCount, color: '#16a34a' },
      { name: '警告', value: record.warningCount, color: '#f59e0b' },
      { name: '严重', value: record.criticalCount, color: '#dc2626' },
      { name: '失败', value: record.failedCount, color: '#64748b' }
    ]
    const total = rows.reduce((sum, item) => sum + Number(item.value || 0), 0)
    let usedPercent = 0
    const lastValueIndex = rows.reduce(
      (last, item, index) => (Number(item.value || 0) > 0 ? index : last),
      -1
    )
    return rows.map((item, index) => {
      const isLast = index === lastValueIndex
      const percent = total
        ? index > lastValueIndex
          ? 0
          : isLast
            ? Math.max(0, Math.round((100 - usedPercent) * 10) / 10)
            : Math.round((Number(item.value || 0) / total) * 1000) / 10
        : 0
      usedPercent += percent
      return { ...item, percent }
    })
  })
  const latestPassRate = computed(() => {
    const record = latestRecord.value
    if (!record || !record.totalCount) return '0%'
    return `${Math.round((record.successCount / record.totalCount) * 1000) / 10}%`
  })
  const trendRecords = computed(() => records.value.slice().reverse())
  const clusterTitle = computed(() => {
    if (!selectedClusterUuid.value) return '全部集群'
    return clusterName(selectedClusterUuid.value)
  })
  const daysText = computed(() => `近 ${days.value} 天`)

  async function refreshData() {
    loading.value = true
    try {
      const [dashboardResp, recordResp] = await Promise.all([
        getInspectionDashboardApi({
          clusterUuid: selectedClusterUuid.value || undefined,
          days: days.value
        }),
        searchInspectionRecordsApi({
          clusterUuid: selectedClusterUuid.value || undefined,
          page: 1,
          pageSize: 30,
          orderField: 'id',
          isAsc: false
        })
      ])
      Object.assign(dashboard, dashboardResp)
      records.value = recordResp.data || []
      await nextTick()
      updateCharts()
    } finally {
      loading.value = false
    }
  }

  function handleClusterChange(id: number | null) {
    setCluster(id)
    refreshData()
  }

  function clusterName(uuid: string) {
    if (!uuid) return '全部集群'
    return clusters.value.find((item) => item.uuid === uuid)?.name || uuid
  }

  function riskCount(row: InspectionRecord) {
    return row.warningCount + row.criticalCount + row.failedCount
  }

  function percent(value: number, total: number) {
    return formatPercent(Math.min(Math.max(value || 0, 0), Math.max(total || 0, 0)), total)
  }

  function goReport() {
    router.push('/cluster/polling/report')
  }

  function initCharts() {
    if (scoreTrendChartRef.value && !scoreTrendChart) {
      scoreTrendChart = echarts.init(scoreTrendChartRef.value)
    }
    if (statusChartRef.value && !statusChart) {
      statusChart = echarts.init(statusChartRef.value)
    }
    if (categoryChartRef.value && !categoryChart) {
      categoryChart = echarts.init(categoryChartRef.value)
    }
    if (riskTrendChartRef.value && !riskTrendChart) {
      riskTrendChart = echarts.init(riskTrendChartRef.value)
    }
  }

  function updateCharts() {
    initCharts()
    updateScoreTrendChart()
    updateStatusChart()
    updateCategoryChart()
    updateRiskTrendChart()
  }

  function updateScoreTrendChart() {
    const rows = trendRecords.value
    scoreTrendChart?.setOption({
      color: ['#2563eb'],
      tooltip: { trigger: 'axis' },
      grid: { left: 36, right: 16, top: 24, bottom: 42 },
      xAxis: { type: 'category', data: rows.map((item) => shortTime(item.createdAt)) },
      yAxis: { type: 'value', min: 0, max: 100 },
      series: [
        {
          name: '得分',
          type: 'line',
          smooth: true,
          symbolSize: 6,
          data: rows.map((item) => Number(item.score.toFixed(1))),
          areaStyle: { color: 'rgba(37, 99, 235, 0.08)' }
        }
      ]
    })
  }

  function updateStatusChart() {
    const rows = statusDistributionRows.value
    statusChart?.setOption({
      color: rows.map((item) => item.color),
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter(params: any) {
          const items = Array.isArray(params) ? params : [params]
          return items
            .map((item) => {
              const row = rows.find((target) => target.name === item.seriesName)
              return `${item.marker}${item.seriesName}: ${row?.value || 0} (${row?.percent || 0}%)`
            })
            .join('<br/>')
        }
      },
      legend: { bottom: 0 },
      grid: { left: 24, right: 24, top: 42, bottom: 54, containLabel: true },
      xAxis: {
        type: 'value',
        min: 0,
        max: 100,
        axisLabel: { formatter: '{value}%' }
      },
      yAxis: { type: 'category', data: ['最新报告'] },
      series: rows.map((item) => ({
        name: item.name,
        type: 'bar',
        stack: 'total',
        barWidth: 30,
        data: [item.percent],
        label: {
          show: item.percent >= 8,
          formatter: `${item.percent}%`,
          color: '#fff',
          fontWeight: 600
        }
      }))
    })
  }

  function updateCategoryChart() {
    const rows = dashboard.categoryStats.slice().sort((left, right) => {
      const leftRisk = left.warningCount + left.criticalCount + left.failedCount
      const rightRisk = right.warningCount + right.criticalCount + right.failedCount
      return rightRisk - leftRisk
    })
    categoryChart?.setOption({
      color: ['#16a34a', '#f59e0b', '#dc2626'],
      tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
      legend: { top: 0 },
      grid: { left: 88, right: 18, top: 34, bottom: 24 },
      xAxis: { type: 'value' },
      yAxis: { type: 'category', data: rows.map((item) => item.category || '默认分类') },
      series: [
        { name: '成功', type: 'bar', stack: 'total', data: rows.map((item) => item.successCount) },
        { name: '警告', type: 'bar', stack: 'total', data: rows.map((item) => item.warningCount) },
        {
          name: '严重/失败',
          type: 'bar',
          stack: 'total',
          data: rows.map((item) => item.criticalCount + item.failedCount)
        }
      ]
    })
  }

  function updateRiskTrendChart() {
    const rows = trendRecords.value
    riskTrendChart?.setOption({
      color: ['#dc2626', '#f59e0b'],
      tooltip: { trigger: 'axis' },
      legend: { top: 0 },
      grid: { left: 36, right: 16, top: 34, bottom: 42 },
      xAxis: { type: 'category', data: rows.map((item) => shortTime(item.createdAt)) },
      yAxis: { type: 'value', minInterval: 1 },
      series: [
        {
          name: '异常项',
          type: 'bar',
          data: rows.map((item) => riskCount(item))
        },
        {
          name: '严重项',
          type: 'line',
          smooth: true,
          data: rows.map((item) => item.criticalCount + item.failedCount)
        }
      ]
    })
  }

  function shortTime(value?: number) {
    if (!value) return '-'
    const timestamp = value < 1000000000000 ? value * 1000 : value
    const date = new Date(timestamp)
    return `${date.getMonth() + 1}/${date.getDate()} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
  }

  function resizeCharts() {
    scoreTrendChart?.resize()
    statusChart?.resize()
    categoryChart?.resize()
    riskTrendChart?.resize()
  }

  function disposeCharts() {
    window.removeEventListener('resize', resizeCharts)
    scoreTrendChart?.dispose()
    statusChart?.dispose()
    categoryChart?.dispose()
    riskTrendChart?.dispose()
    scoreTrendChart = null
    statusChart = null
    categoryChart = null
    riskTrendChart = null
  }

  onMounted(async () => {
    window.addEventListener('resize', resizeCharts)
    await loadClusters()
    await refreshData()
  })

  onBeforeUnmount(() => {
    disposeCharts()
  })
</script>

<style scoped lang="scss">
  @use '../shared/polling-ui' as *;

  .dashboard-toolbar {
    margin-bottom: 16px;

    :deep(.el-card__body) {
      display: flex;
      gap: 16px;
      align-items: center;
      justify-content: space-between;
    }
  }

  .toolbar-title {
    min-width: 0;

    strong,
    span {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      font-size: 16px;
      color: var(--el-text-color-primary);
    }

    span {
      margin-top: 4px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .panel-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    align-items: center;
    justify-content: flex-end;
  }

  .kpi-grid {
    display: grid;
    grid-template-columns: repeat(6, minmax(0, 1fr));
    gap: 12px;
    margin-bottom: 16px;
  }

  .kpi-card {
    min-width: 0;
    padding: 14px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    box-shadow: 0 1px 2px rgb(15 23 42 / 3%);

    span,
    strong,
    em {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    span {
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }

    strong {
      margin-top: 8px;
      font-size: 24px;
      line-height: 1;
      color: var(--el-text-color-primary);
    }

    em {
      margin-top: 8px;
      font-size: 12px;
      font-style: normal;
      color: var(--el-text-color-secondary);
    }
  }

  .chart-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
    margin-bottom: 16px;
  }

  .chart-card {
    min-width: 0;
  }

  .chart-box {
    width: 100%;
    height: 300px;
  }

  @media (width <= 1400px) {
    .kpi-grid {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
  }

  @media (width <= 1200px) {
    .dashboard-toolbar :deep(.el-card__body) {
      flex-direction: column;
      align-items: stretch;
    }

    .panel-actions {
      justify-content: flex-start;
    }

    .chart-grid {
      grid-template-columns: 1fr;
    }
  }

  @media (width <= 768px) {
    .kpi-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
