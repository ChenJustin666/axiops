<template>
  <div class="log-dashboard-page">
    <ElCard shadow="never" class="panel-card dashboard-toolbar">
      <div class="toolbar-title">
        <strong>日志告警大盘</strong>
        <span>{{ eventClusterName || '全部集群' }} · 当前 {{ events.length }} 条最近事件</span>
      </div>
      <div class="panel-actions">
        <ElSelect
          v-model="eventClusterId"
          placeholder="最近事件按集群查看"
          filterable
          clearable
          class="cluster-select"
          :loading="clusterLoading"
          @change="refreshRecentEvents"
        >
          <ElOption v-for="item in clusters" :key="item.id" :label="item.name" :value="item.id" />
        </ElSelect>
        <ElButton :icon="Refresh" :loading="loading" @click="refreshData">刷新</ElButton>
      </div>
    </ElCard>

    <div class="kpi-grid" v-loading="loading">
      <div class="kpi-card">
        <span>纳管集群</span>
        <strong>{{ clusters.length }}</strong>
        <em>当前日志集群</em>
      </div>
      <div class="kpi-card">
        <span>规则总数</span>
        <strong>{{ stats.totalRules }}</strong>
        <em>启用 {{ stats.enabledRules }} / 停用 {{ stats.disabledRules }}</em>
      </div>
      <div class="kpi-card">
        <span>启用规则</span>
        <strong>{{ stats.enabledRules }}</strong>
        <em>正在参与扫描</em>
      </div>
      <div class="kpi-card">
        <span>同步成功</span>
        <strong>{{ stats.successRules }}</strong>
        <em>失败 {{ stats.failedRules }}</em>
      </div>
      <div class="kpi-card">
        <span>待处理</span>
        <strong>{{ stats.pendingRules }}</strong>
        <em>同步队列</em>
      </div>
      <div class="kpi-card">
        <span>规则级别</span>
        <strong>{{ stats.criticalRules }}</strong>
        <em>Warning {{ stats.warningRules }} / Info {{ stats.infoRules }}</em>
      </div>
    </div>

    <div class="dashboard-grid">
      <ElCard shadow="never" class="panel-card summary-card">
        <template #header>
          <div class="card-header-row">
            <div class="card-title">全局态势</div>
            <ElTag effect="plain">稳定集群 {{ stableClusterCount }}</ElTag>
          </div>
        </template>
        <ElDescriptions :column="2" border>
          <ElDescriptionsItem label="同步成功">{{ stats.successRules }}</ElDescriptionsItem>
          <ElDescriptionsItem label="同步失败">{{ stats.failedRules }}</ElDescriptionsItem>
          <ElDescriptionsItem label="待处理">{{ stats.pendingRules }}</ElDescriptionsItem>
          <ElDescriptionsItem label="Critical">{{ stats.criticalRules }}</ElDescriptionsItem>
          <ElDescriptionsItem label="Warning">{{ stats.warningRules }}</ElDescriptionsItem>
          <ElDescriptionsItem label="Hot 规则">{{ stats.hotRules }}</ElDescriptionsItem>
          <ElDescriptionsItem label="Cold 规则">{{ stats.coldRules }}</ElDescriptionsItem>
          <ElDescriptionsItem label="最近事件范围">
            {{ eventClusterName || '全部集群' }}
          </ElDescriptionsItem>
        </ElDescriptions>
      </ElCard>

      <ElCard shadow="never" class="panel-card summary-card">
        <template #header>
          <div class="card-title">集群排名</div>
        </template>
        <div class="cluster-spotlight-list">
          <div
            v-for="(item, index) in rankedClusters"
            :key="item.clusterUuid"
            class="cluster-spotlight-item"
          >
            <div class="rank-badge">{{ index + 1 }}</div>
            <div>
              <div class="backend-title">{{ item.clusterName }}</div>
              <div class="backend-meta">
                默认后端 {{ backendText(item.defaultBackend) }} · 启用 {{ item.enabledRules }} ·
                活跃 {{ item.activeFiringEvents }} · 风险 {{ item.riskScore }}
              </div>
              <div class="rank-progress">
                <div class="rank-progress-bar" :style="{ width: `${item.progress}%` }" />
              </div>
            </div>
            <ElTag
              :type="item.failedNotifyEvents + item.deadNotifyEvents > 0 ? 'danger' : 'success'"
            >
              {{ item.failedNotifyEvents + item.deadNotifyEvents > 0 ? '需关注' : '稳定' }}
            </ElTag>
          </div>
        </div>
      </ElCard>
    </div>

    <div class="quick-grid">
      <button class="quick-card" @click="goRoute('/log/alert')">
        <div class="quick-title">规则管理</div>
        <div class="quick-desc">新建、编辑和启停日志告警规则。</div>
      </button>
      <button class="quick-card" @click="goRoute('/log/engine')">
        <div class="quick-title">日志告警中心</div>
        <div class="quick-desc">查看事件、补原因、手动关闭和删除事件。</div>
      </button>
      <button class="quick-card" @click="goRoute('/log/search')">
        <div class="quick-title">日志查询</div>
        <div class="quick-desc">带着上下文跳到日志检索页做进一步排查。</div>
      </button>
      <button class="quick-card" @click="goRoute('/log/config')">
        <div class="quick-title">后端配置</div>
        <div class="quick-desc">查看默认后端、同步状态和后端实例详情。</div>
      </button>
    </div>

    <ElCard shadow="never" class="table-card">
      <ArtTableHeader
        v-model:columns="eventColumns"
        :loading="loading"
        :showZebra="true"
        :showBorder="true"
        :showHeaderBackground="true"
        :fullClass="'art-page-view'"
        :layout="'refresh,size,fullscreen,columns,settings'"
        @refresh="refreshData"
      >
        <template #left>
          <span class="card-title">最近事件</span>
        </template>
        <template #right>
          <ElButton text type="primary" @click="goRoute('/log/engine')">进入告警中心</ElButton>
        </template>
      </ArtTableHeader>
      <ArtTable
        :loading="loading"
        :data="events.slice(0, 6)"
        :columns="visibleEventColumns"
        :show-table-header="false"
        empty-text="暂无事件"
      >
        <template #scope="{ row }">
          {{ [row.projectName, row.namespace, row.podName].filter(Boolean).join(' / ') || '-' }}
        </template>
        <template #sample="{ row }">{{ row.sampleMessage || '-' }}</template>
        <template #createdAt="{ row }">{{ formatTime(row.createdAt) }}</template>
      </ArtTable>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import dayjs from 'dayjs'
  import { Refresh } from '@element-plus/icons-vue'
  import {
    getLogAlertEngineStatsApi,
    getLogClusterConfigApi,
    searchLogAlertEventsApi,
    type LogAlertEvent
  } from '@/api/manager/log'
  import { useLogClusters } from '../shared/useLogClusters'

  defineOptions({ name: 'ClusterLogDashboardPage' })

  const router = useRouter()
  const {
    clusters,
    loading: clusterLoading,
    loadClusters,
    selectedClusterId,
    setCluster
  } = useLogClusters('cluster-log-dashboard-event-cluster')

  const loading = ref(false)
  const eventClusterId = ref<number | null>(selectedClusterId.value)
  const events = ref<LogAlertEvent[]>([])
  const stats = reactive({
    clusterUuid: '',
    totalRules: 0,
    enabledRules: 0,
    disabledRules: 0,
    successRules: 0,
    failedRules: 0,
    pendingRules: 0,
    activeFiringEvents: 0,
    failedNotifyEvents: 0,
    deadNotifyEvents: 0,
    criticalRules: 0,
    warningRules: 0,
    infoRules: 0,
    hotRules: 0,
    warmRules: 0,
    coldRules: 0
  })
  const clusterSummaries = ref<
    Array<{
      clusterUuid: string
      clusterName: string
      defaultBackend: string
      enabledRules: number
      activeFiringEvents: number
      failedNotifyEvents: number
      deadNotifyEvents: number
      riskScore: number
      progress: number
    }>
  >([])
  const eventColumns = ref([
    { prop: 'clusterName', label: '集群', minWidth: 180 },
    { prop: 'ruleName', label: '规则', minWidth: 220 },
    { prop: 'scope', label: '范围', minWidth: 220, useSlot: true },
    { prop: 'sample', label: '样本', minWidth: 280, useSlot: true, showOverflowTooltip: true },
    { prop: 'eventStatus', label: '事件状态', width: 110 },
    { prop: 'notifyStatus', label: '通知状态', width: 110 },
    { prop: 'createdAt', label: '时间', width: 180, useSlot: true }
  ])
  const visibleEventColumns = computed(() =>
    eventColumns.value.filter((column: any) => column.visible !== false)
  )
  const eventClusterName = computed(
    () => clusters.value.find((item) => item.id === eventClusterId.value)?.name || ''
  )
  const stableClusterCount = computed(
    () =>
      clusterSummaries.value.filter((item) => item.failedNotifyEvents + item.deadNotifyEvents === 0)
        .length
  )
  const rankedClusters = computed(() =>
    [...clusterSummaries.value].sort(
      (a, b) => b.riskScore - a.riskScore || b.activeFiringEvents - a.activeFiringEvents
    )
  )

  async function refreshData() {
    loading.value = true
    try {
      const statsResp = await getLogAlertEngineStatsApi()
      Object.assign(stats, statsResp)
      await Promise.all([loadClusterSummaries(), refreshRecentEvents()])
    } finally {
      loading.value = false
    }
  }

  async function loadClusterSummaries() {
    const summaries = await Promise.all(
      clusters.value.map(async (item) => {
        const [configResp, statsResp] = await Promise.all([
          getLogClusterConfigApi(item.uuid),
          getLogAlertEngineStatsApi(item.uuid)
        ])
        return {
          clusterUuid: item.uuid,
          clusterName: item.name,
          defaultBackend: configResp.defaultBackend || '',
          enabledRules: statsResp.enabledRules || 0,
          activeFiringEvents: statsResp.activeFiringEvents || 0,
          failedNotifyEvents: statsResp.failedNotifyEvents || 0,
          deadNotifyEvents: statsResp.deadNotifyEvents || 0,
          riskScore:
            (statsResp.failedNotifyEvents || 0) * 5 +
            (statsResp.deadNotifyEvents || 0) * 8 +
            (statsResp.activeFiringEvents || 0) * 3 +
            (statsResp.criticalRules || 0),
          progress: 0
        }
      })
    )
    const maxRisk = Math.max(...summaries.map((item) => item.riskScore), 1)
    clusterSummaries.value = summaries.map((item) => ({
      ...item,
      progress: Math.max(10, Math.round((item.riskScore / maxRisk) * 100))
    }))
  }

  async function refreshRecentEvents() {
    setCluster(eventClusterId.value)
    const clusterUuid =
      clusters.value.find((item) => item.id === eventClusterId.value)?.uuid || undefined
    const eventsResp = await searchLogAlertEventsApi({ clusterUuid, page: 1, pageSize: 20 })
    events.value = eventsResp.data || []
  }

  function goRoute(path: string) {
    router.push(path)
  }

  function backendText(value?: string) {
    if (!value) return '-'
    if (value === 'elasticsearch' || value === 'es') return 'Elasticsearch'
    if (value === 'loki') return 'Loki'
    return value
  }

  function formatTime(value?: number) {
    return value ? dayjs(value * 1000).format('YYYY-MM-DD HH:mm:ss') : '-'
  }

  onMounted(async () => {
    await loadClusters()
    await refreshData()
  })
</script>

<style scoped>
  .log-dashboard-page {
    min-height: 100%;
    padding: 0;
    background: var(--el-fill-color-extra-light);
  }

  .panel-card,
  .table-card,
  .kpi-card,
  .quick-card {
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    box-shadow: 0 1px 2px rgb(15 23 42 / 3%);
  }

  .dashboard-toolbar {
    margin-bottom: 16px;
  }

  .dashboard-toolbar :deep(.el-card__body) {
    display: flex;
    gap: 16px;
    align-items: center;
    justify-content: space-between;
    padding: 16px;
  }

  .toolbar-title {
    min-width: 0;
  }

  .toolbar-title strong,
  .toolbar-title span,
  .kpi-card span,
  .kpi-card strong,
  .kpi-card em {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .toolbar-title strong {
    font-size: 16px;
    color: var(--el-text-color-primary);
  }

  .toolbar-title span {
    margin-top: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .panel-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    align-items: center;
    justify-content: flex-end;
  }

  .panel-actions :deep(.el-button + .el-button) {
    margin-left: 0;
  }

  .cluster-select {
    width: 260px;
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
  }

  .kpi-card span {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .kpi-card strong {
    margin-top: 8px;
    font-size: 24px;
    line-height: 1;
    color: var(--el-text-color-primary);
  }

  .kpi-card em {
    margin-top: 8px;
    font-size: 12px;
    font-style: normal;
    color: var(--el-text-color-secondary);
  }

  .dashboard-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.1fr) minmax(360px, 0.9fr);
    gap: 16px;
    margin-bottom: 16px;
  }

  .panel-card :deep(.el-card__body),
  .table-card :deep(.el-card__body) {
    padding: 16px;
  }

  .panel-card :deep(.el-card__header),
  .table-card :deep(.el-card__header) {
    padding: 14px 16px;
    border-bottom: 1px solid var(--el-border-color-extra-light);
  }

  .card-header-row {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
  }

  .card-title {
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  .cluster-spotlight-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .cluster-spotlight-item {
    display: grid;
    grid-template-columns: 32px minmax(0, 1fr) auto;
    gap: 12px;
    align-items: center;
    padding: 12px 14px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-extra-light);
    border-radius: 8px;
  }

  .rank-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    font-size: 13px;
    font-weight: 700;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-radius: 8px;
  }

  .backend-title {
    overflow: hidden;
    font-weight: 700;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .backend-meta {
    margin-top: 4px;
    overflow: hidden;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .rank-progress {
    width: 100%;
    height: 6px;
    margin-top: 10px;
    overflow: hidden;
    background: var(--el-fill-color);
    border-radius: 999px;
  }

  .rank-progress-bar {
    height: 100%;
    background: var(--el-color-primary);
    border-radius: 999px;
  }

  .quick-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 12px;
    margin-bottom: 16px;
  }

  .quick-card {
    min-height: 92px;
    padding: 14px;
    text-align: left;
    cursor: pointer;
    transition:
      border-color 0.16s ease,
      box-shadow 0.16s ease,
      transform 0.16s ease;
  }

  .quick-card:hover {
    border-color: var(--el-color-primary-light-6);
    box-shadow: 0 8px 18px rgb(15 23 42 / 6%);
    transform: translateY(-1px);
  }

  .quick-title {
    font-size: 15px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  .quick-desc {
    display: -webkit-box;
    margin-top: 8px;
    overflow: hidden;
    font-size: 12px;
    line-height: 1.5;
    color: var(--el-text-color-secondary);
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }

  @media (width <= 1400px) {
    .kpi-grid {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }

    .quick-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
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

    .dashboard-grid {
      grid-template-columns: 1fr;
    }
  }

  @media (width <= 768px) {
    .cluster-select {
      width: 100%;
    }

    .kpi-grid,
    .quick-grid {
      grid-template-columns: 1fr;
    }

    .cluster-spotlight-item {
      grid-template-columns: 32px minmax(0, 1fr);
    }

    .cluster-spotlight-item > .el-tag {
      grid-column: 2;
      justify-self: start;
    }
  }
</style>
