<template>
  <div class="alert-dashboard">
    <div class="dashboard-header">
      <div>
        <h1 class="page-title">告警仪表盘</h1>
        <p class="page-subtitle">实时监控告警状态和趋势分析</p>
      </div>
      <div class="header-actions">
        <ElSelect
          v-model="filterForm.clusterUuid"
          placeholder="全部集群"
          clearable
          style="width: 160px"
          @change="handleFilterChange"
        >
          <ElOption
            v-for="cluster in clusterList"
            :key="cluster.uuid"
            :label="cluster.name"
            :value="cluster.uuid"
          />
        </ElSelect>
        <ElSelect
          v-model="filterForm.projectId"
          placeholder="全部项目"
          clearable
          style="width: 160px"
          @change="handleFilterChange"
        >
          <ElOption
            v-for="project in projectList"
            :key="project.id"
            :label="project.name"
            :value="project.id"
          />
        </ElSelect>
        <ElButton type="primary" @click="refreshAllData">
          <ArtSvgIcon icon="ri:refresh-line" class="mr-1" />
          刷新
        </ElButton>
      </div>
    </div>

    <div class="top-stats-grid">
      <div class="stat-card stat-card-violet">
        <div class="stat-card-icon"><ArtSvgIcon icon="ri:alarm-warning-line" /></div>
        <div class="stat-card-content">
          <span class="stat-card-label">告警总数</span>
          <ArtCountTo class="stat-card-value" :target="overviewData.totalCount" :duration="1000" />
          <span class="stat-card-sub">
            较昨日
            <span
              :class="overviewData.compareYesterday >= 0 ? 'text-rose-200' : 'text-emerald-200'"
            >
              {{ formatChange(overviewData.compareYesterday) }}
            </span>
          </span>
        </div>
      </div>

      <div class="stat-card stat-card-orange">
        <div class="stat-card-icon">
          <span class="pulse-dot"></span>
          <ArtSvgIcon icon="ri:error-warning-line" />
        </div>
        <div class="stat-card-content">
          <span class="stat-card-label">触发中</span>
          <ArtCountTo class="stat-card-value" :target="overviewData.firingCount" :duration="1000" />
          <span class="stat-card-sub">需要立即处理</span>
        </div>
      </div>

      <div class="stat-card stat-card-green">
        <div class="stat-card-icon"><ArtSvgIcon icon="ri:checkbox-circle-line" /></div>
        <div class="stat-card-content">
          <span class="stat-card-label">已恢复</span>
          <ArtCountTo
            class="stat-card-value"
            :target="overviewData.resolvedCount"
            :duration="1000"
          />
          <span class="stat-card-sub">恢复率 {{ overviewData.resolvedRate.toFixed(1) }}%</span>
        </div>
      </div>

      <div class="stat-card stat-card-cyan">
        <div class="stat-card-icon"><ArtSvgIcon icon="ri:time-line" /></div>
        <div class="stat-card-content">
          <span class="stat-card-label">今日新增</span>
          <ArtCountTo
            class="stat-card-value"
            :target="overviewData.todayNewCount"
            :duration="1000"
          />
          <span class="stat-card-sub">今日已恢复 {{ overviewData.todayResolvedCount }}</span>
        </div>
      </div>
    </div>

    <div class="severity-stats-grid">
      <div class="severity-card severity-critical">
        <div class="severity-header">
          <div class="severity-icon-wrapper severity-icon-critical">
            <ArtSvgIcon icon="ri:fire-line" />
          </div>
          <div class="severity-detail">
            <div class="severity-label">Critical 严重</div>
            <div class="severity-value">{{ getSeverityData('critical').totalCount }}</div>
          </div>
          <div class="severity-firing">
            <span class="firing-tag firing-tag-critical">触发中</span>
            <span class="firing-count firing-count-critical">{{
              getSeverityData('critical').firingCount
            }}</span>
          </div>
        </div>
        <div class="severity-progress">
          <div
            class="severity-progress-bar severity-progress-critical"
            :style="{ width: getProgressWidth('critical') + '%' }"
          ></div>
        </div>
      </div>

      <div class="severity-card severity-warning">
        <div class="severity-header">
          <div class="severity-icon-wrapper severity-icon-warning">
            <ArtSvgIcon icon="ri:alert-line" />
          </div>
          <div class="severity-detail">
            <div class="severity-label">Warning 警告</div>
            <div class="severity-value">{{ getSeverityData('warning').totalCount }}</div>
          </div>
          <div class="severity-firing">
            <span class="firing-tag firing-tag-warning">触发中</span>
            <span class="firing-count firing-count-warning">{{
              getSeverityData('warning').firingCount
            }}</span>
          </div>
        </div>
        <div class="severity-progress">
          <div
            class="severity-progress-bar severity-progress-warning"
            :style="{ width: getProgressWidth('warning') + '%' }"
          ></div>
        </div>
      </div>

      <div class="severity-card severity-info">
        <div class="severity-header">
          <div class="severity-icon-wrapper severity-icon-info">
            <ArtSvgIcon icon="ri:information-line" />
          </div>
          <div class="severity-detail">
            <div class="severity-label">Info 信息</div>
            <div class="severity-value">{{ getSeverityData('info').totalCount }}</div>
          </div>
          <div class="severity-firing">
            <span class="firing-tag firing-tag-info">触发中</span>
            <span class="firing-count firing-count-info">{{
              getSeverityData('info').firingCount
            }}</span>
          </div>
        </div>
        <div class="severity-progress">
          <div
            class="severity-progress-bar severity-progress-info"
            :style="{ width: getProgressWidth('info') + '%' }"
          ></div>
        </div>
      </div>

      <div class="severity-card severity-none">
        <div class="severity-header">
          <div class="severity-icon-wrapper severity-icon-none">
            <ArtSvgIcon icon="ri:question-line" />
          </div>
          <div class="severity-detail">
            <div class="severity-label">None 无级别</div>
            <div class="severity-value">{{ getSeverityData('none').totalCount }}</div>
          </div>
          <div class="severity-firing">
            <span class="firing-tag firing-tag-none">触发中</span>
            <span class="firing-count firing-count-none">{{
              getSeverityData('none').firingCount
            }}</span>
          </div>
        </div>
        <div class="severity-progress">
          <div
            class="severity-progress-bar severity-progress-none"
            :style="{ width: getProgressWidth('none') + '%' }"
          ></div>
        </div>
      </div>
    </div>

    <ElRow :gutter="20" class="mb-5">
      <ElCol :sm="24" :md="24" :lg="16">
        <div class="art-card chart-card">
          <div class="card-header">
            <div class="header-left">
              <ArtSvgIcon icon="ri:line-chart-line" class="header-icon text-indigo-500" />
              <span class="header-title">告警趋势</span>
            </div>
            <div class="btn-group">
              <button
                v-for="day in [7, 14, 30]"
                :key="day"
                class="btn-item"
                :class="{ active: trendDays === day }"
                @click="setTrendDays(day)"
              >
                {{ day }}天
              </button>
            </div>
          </div>
          <div class="chart-container">
            <AlertTrendChart ref="trendChartRef" :filter="filterForm" :days="trendDays" />
          </div>
        </div>
      </ElCol>
      <ElCol :sm="24" :md="24" :lg="8">
        <div class="art-card chart-card">
          <div class="card-header">
            <div class="header-left">
              <ArtSvgIcon icon="ri:pie-chart-line" class="header-icon text-purple-500" />
              <span class="header-title">级别分布</span>
            </div>
          </div>
          <div class="chart-container">
            <SeverityPieChart :data="severityData" />
          </div>
        </div>
      </ElCol>
    </ElRow>

    <ElRow :gutter="20" class="mb-5">
      <ElCol :sm="24" :md="12" :lg="12">
        <div class="art-card table-card">
          <div class="card-header">
            <div class="header-left">
              <ArtSvgIcon icon="ri:server-line" class="header-icon text-blue-500" />
              <span class="header-title">集群告警统计</span>
            </div>
            <span class="top-badge">Top 10</span>
          </div>
          <div class="table-container">
            <ClusterStatsTable :filter="filterForm" ref="clusterStatsRef" />
          </div>
        </div>
      </ElCol>
      <ElCol :sm="24" :md="12" :lg="12">
        <div class="art-card table-card">
          <div class="card-header">
            <div class="header-left">
              <ArtSvgIcon icon="ri:folder-line" class="header-icon text-purple-500" />
              <span class="header-title">项目告警统计</span>
            </div>
            <span class="top-badge">Top 10</span>
          </div>
          <div class="table-container">
            <ProjectStatsTable :filter="filterForm" ref="projectStatsRef" />
          </div>
        </div>
      </ElCol>
    </ElRow>

    <ElRow :gutter="20">
      <ElCol :sm="24" :md="12" :lg="12">
        <div class="art-card table-card">
          <div class="card-header">
            <div class="header-left">
              <ArtSvgIcon icon="ri:bar-chart-grouped-line" class="header-icon text-orange-500" />
              <span class="header-title">告警规则 Top 10</span>
            </div>
          </div>
          <div class="table-container">
            <RuleRankingList :filter="filterForm" ref="ruleRankingRef" />
          </div>
        </div>
      </ElCol>
      <ElCol :sm="24" :md="12" :lg="12">
        <div class="art-card table-card">
          <div class="card-header">
            <div class="header-left">
              <ArtSvgIcon icon="ri:layout-grid-line" class="header-icon text-cyan-500" />
              <span class="header-title">工作空间告警统计</span>
            </div>
            <span class="top-badge">Top 10</span>
          </div>
          <div class="table-container">
            <WorkspaceStatsTable :filter="filterForm" ref="workspaceStatsRef" />
          </div>
        </div>
      </ElCol>
    </ElRow>
  </div>
</template>

<script setup lang="ts">
  import { ref, reactive, onMounted, onUnmounted } from 'vue'
  import {
    getAlertOverviewApi,
    getAlertSeverityStatsApi,
    type GetAlertOverviewResponse,
    type GetAlertSeverityStatsResponse,
    type SeverityStatItem
  } from '@/api/manager/alert-dashboard'
  import { searchClusterApi, type Cluster } from '@/api/manager/cluster'
  import { getProjectsByUserApi, type Project } from '@/api/manager/project'
  import AlertTrendChart from './modules/alert-trend-chart.vue'
  import SeverityPieChart from './modules/severity-pie-chart.vue'
  import ClusterStatsTable from './modules/cluster-stats-table.vue'
  import ProjectStatsTable from './modules/project-stats-table.vue'
  import RuleRankingList from './modules/rule-ranking-list.vue'
  import WorkspaceStatsTable from './modules/workspace-stats-table.vue'

  defineOptions({ name: 'AlertDashboard' })

  const filterForm = reactive({
    clusterUuid: '',
    projectId: undefined as number | undefined,
    workspaceId: undefined as number | undefined
  })

  const trendDays = ref(7)
  const clusterList = ref<Cluster[]>([])
  const projectList = ref<Project[]>([])

  const overviewData = ref<GetAlertOverviewResponse>({
    totalCount: 0,
    firingCount: 0,
    resolvedCount: 0,
    todayNewCount: 0,
    todayResolvedCount: 0,
    avgDuration: 0,
    resolvedRate: 0,
    compareYesterday: 0
  })

  const severityData = ref<GetAlertSeverityStatsResponse>({
    items: [],
    totalCount: 0
  })

  const trendChartRef = ref()
  const clusterStatsRef = ref()
  const projectStatsRef = ref()
  const ruleRankingRef = ref()
  const workspaceStatsRef = ref()

  let refreshTimer: ReturnType<typeof setInterval> | null = null

  const getSeverityData = (severity: string): SeverityStatItem => {
    const item = severityData.value.items.find((i) => i.severity === severity)
    return (
      item || {
        severity,
        severityCn: '',
        totalCount: 0,
        firingCount: 0,
        resolvedCount: 0,
        percentage: 0
      }
    )
  }

  const getProgressWidth = (severity: string): number => {
    const data = getSeverityData(severity)
    if (data.totalCount === 0) return 0
    return Math.round((data.firingCount / data.totalCount) * 100)
  }

  const formatChange = (value: number): string => {
    const sign = value >= 0 ? '+' : ''
    return `${sign}${value.toFixed(1)}%`
  }

  const setTrendDays = (days: number) => {
    trendDays.value = days
  }

  const loadClusterList = async () => {
    try {
      const res = await searchClusterApi({ pageSize: 100 })
      clusterList.value = res.items || []
    } catch (error) {}
  }

  const loadProjectList = async () => {
    try {
      const res = await getProjectsByUserApi()
      projectList.value = res || []
    } catch (error) {}
  }

  const loadOverviewData = async () => {
    try {
      const res = await getAlertOverviewApi({
        clusterUuid: filterForm.clusterUuid || undefined,
        projectId: filterForm.projectId || undefined,
        workspaceId: filterForm.workspaceId || undefined
      })
      overviewData.value = res
    } catch (error) {}
  }

  const loadSeverityData = async () => {
    try {
      const res = await getAlertSeverityStatsApi({
        clusterUuid: filterForm.clusterUuid || undefined,
        projectId: filterForm.projectId || undefined,
        workspaceId: filterForm.workspaceId || undefined
      })
      severityData.value = res
    } catch (error) {}
  }

  const handleFilterChange = () => {
    refreshAllData()
  }

  const refreshAllData = async () => {
    await Promise.all([loadOverviewData(), loadSeverityData()])
    clusterStatsRef.value?.refresh()
    projectStatsRef.value?.refresh()
    ruleRankingRef.value?.refresh()
    workspaceStatsRef.value?.refresh()
    trendChartRef.value?.refresh()
  }

  const startAutoRefresh = () => {
    refreshTimer = setInterval(() => {
      loadOverviewData()
      loadSeverityData()
    }, 60000)
  }

  onMounted(async () => {
    await Promise.all([loadClusterList(), loadProjectList()])
    await refreshAllData()
    startAutoRefresh()
  })

  onUnmounted(() => {
    if (refreshTimer) {
      clearInterval(refreshTimer)
    }
  })
</script>

<style scoped>
  .alert-dashboard {
    padding: 0;
  }

  .dashboard-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 24px;
  }

  .page-title {
    font-size: 24px;
    font-weight: 600;
    color: #1f2937;
    margin: 0;
  }

  .page-subtitle {
    font-size: 14px;
    color: #6b7280;
    margin-top: 4px;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .top-stats-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 16px;
    margin-bottom: 20px;
  }

  .stat-card {
    position: relative;
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 20px;
    border-radius: 16px;
    color: white;
    overflow: hidden;
    transition:
      transform 0.2s,
      box-shadow 0.2s;
  }

  .stat-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 12px 24px -8px rgba(0, 0, 0, 0.15);
  }

  .stat-card::before {
    content: '';
    position: absolute;
    top: 0;
    right: 0;
    width: 100px;
    height: 100px;
    background: rgba(255, 255, 255, 0.1);
    border-radius: 50%;
    transform: translate(30%, -30%);
  }

  .stat-card-violet {
    background: linear-gradient(135deg, #8b5cf6, #a78bfa);
  }

  .stat-card-orange {
    background: linear-gradient(135deg, #f97316, #fb923c);
  }

  .stat-card-green {
    background: linear-gradient(135deg, #10b981, #34d399);
  }

  .stat-card-cyan {
    background: linear-gradient(135deg, #06b6d4, #22d3ee);
  }

  .stat-card-icon {
    position: relative;
    width: 48px;
    height: 48px;
    background: rgba(255, 255, 255, 0.2);
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 22px;
    flex-shrink: 0;
  }

  .pulse-dot {
    position: absolute;
    top: -4px;
    right: -4px;
    width: 10px;
    height: 10px;
    background: #fff;
    border-radius: 50%;
    animation: pulse 1.5s ease-in-out infinite;
  }

  @keyframes pulse {
    0%,
    100% {
      opacity: 1;
      transform: scale(1);
    }
    50% {
      opacity: 0.6;
      transform: scale(1.2);
    }
  }

  .stat-card-content {
    position: relative;
    z-index: 1;
    min-width: 0;
  }

  .stat-card-label {
    font-size: 13px;
    opacity: 0.9;
    margin-bottom: 4px;
    display: block;
  }

  .stat-card-value {
    font-size: 28px;
    font-weight: 700;
    line-height: 1.2;
    color: white;
  }

  .stat-card-sub {
    font-size: 12px;
    opacity: 0.85;
    margin-top: 4px;
    display: block;
  }

  .severity-stats-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 16px;
    margin-bottom: 20px;
  }

  .severity-card {
    padding: 20px;
    border-radius: 16px;
    transition:
      transform 0.2s,
      box-shadow 0.2s;
  }

  .severity-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 16px -4px rgba(0, 0, 0, 0.1);
  }

  .severity-critical {
    background: linear-gradient(135deg, rgba(239, 68, 68, 0.08), rgba(239, 68, 68, 0.02));
    border: 1px solid rgba(239, 68, 68, 0.15);
  }

  .severity-warning {
    background: linear-gradient(135deg, rgba(245, 158, 11, 0.08), rgba(245, 158, 11, 0.02));
    border: 1px solid rgba(245, 158, 11, 0.15);
  }

  .severity-info {
    background: linear-gradient(135deg, rgba(59, 130, 246, 0.08), rgba(59, 130, 246, 0.02));
    border: 1px solid rgba(59, 130, 246, 0.15);
  }

  .severity-none {
    background: linear-gradient(135deg, rgba(107, 114, 128, 0.08), rgba(107, 114, 128, 0.02));
    border: 1px solid rgba(107, 114, 128, 0.15);
  }

  .severity-header {
    display: flex;
    align-items: center;
    gap: 14px;
  }

  .severity-icon-wrapper {
    width: 42px;
    height: 42px;
    border-radius: 10px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 18px;
    flex-shrink: 0;
  }

  .severity-icon-critical {
    background: linear-gradient(135deg, #ef4444, #f87171);
    color: white;
  }

  .severity-icon-warning {
    background: linear-gradient(135deg, #f59e0b, #fbbf24);
    color: white;
  }

  .severity-icon-info {
    background: linear-gradient(135deg, #3b82f6, #60a5fa);
    color: white;
  }

  .severity-icon-none {
    background: linear-gradient(135deg, #6b7280, #9ca3af);
    color: white;
  }

  /* 修复：将 .severity-info 改为 .severity-detail 避免类名冲突 */
  .severity-detail {
    flex: 1;
    min-width: 0;
  }

  .severity-label {
    font-size: 13px;
    color: #6b7280;
    margin-bottom: 2px;
  }

  .severity-value {
    font-size: 24px;
    font-weight: 700;
    color: #1f2937;
  }

  .severity-firing {
    text-align: right;
  }

  .firing-tag {
    display: inline-block;
    padding: 3px 10px;
    border-radius: 12px;
    font-size: 11px;
    font-weight: 500;
    margin-bottom: 4px;
  }

  .firing-tag-critical {
    background: rgba(239, 68, 68, 0.1);
    color: #dc2626;
  }

  .firing-tag-warning {
    background: rgba(245, 158, 11, 0.1);
    color: #d97706;
  }

  .firing-tag-info {
    background: rgba(59, 130, 246, 0.1);
    color: #2563eb;
  }

  .firing-tag-none {
    background: rgba(107, 114, 128, 0.1);
    color: #6b7280;
  }

  .firing-count {
    display: block;
    font-size: 18px;
    font-weight: 600;
  }

  .firing-count-critical {
    color: #dc2626;
  }

  .firing-count-warning {
    color: #d97706;
  }

  .firing-count-info {
    color: #2563eb;
  }

  .firing-count-none {
    color: #6b7280;
  }

  .severity-progress {
    height: 4px;
    background: #e5e7eb;
    border-radius: 2px;
    margin-top: 14px;
    overflow: hidden;
  }

  .severity-progress-bar {
    height: 100%;
    border-radius: 2px;
    transition: width 0.5s ease;
  }

  .severity-progress-critical {
    background: linear-gradient(90deg, #ef4444, #f87171);
  }

  .severity-progress-warning {
    background: linear-gradient(90deg, #f59e0b, #fbbf24);
  }

  .severity-progress-info {
    background: linear-gradient(90deg, #3b82f6, #60a5fa);
  }

  .severity-progress-none {
    background: linear-gradient(90deg, #6b7280, #9ca3af);
  }

  .chart-card,
  .table-card {
    padding: 24px;
    margin-bottom: 20px;
  }

  .chart-container {
    height: 280px;
  }

  .table-container {
    height: 320px;
    overflow-y: auto;
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .header-icon {
    font-size: 18px;
  }

  .header-title {
    font-size: 16px;
    font-weight: 600;
    color: #374151;
  }

  .btn-group {
    display: flex;
    background: #f3f4f6;
    border-radius: 8px;
    padding: 4px;
  }

  .btn-item {
    padding: 6px 12px;
    font-size: 13px;
    border: none;
    background: transparent;
    color: #6b7280;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-item:hover {
    color: #374151;
  }

  .btn-item.active {
    background: white;
    color: var(--el-color-primary);
    font-weight: 500;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .top-badge {
    display: inline-block;
    padding: 4px 12px;
    background: linear-gradient(135deg, rgba(59, 130, 246, 0.1), rgba(99, 102, 241, 0.1));
    color: #3b82f6;
    border-radius: 20px;
    font-size: 12px;
    font-weight: 500;
  }

  .text-indigo-500 {
    color: #6366f1;
  }

  .text-purple-500 {
    color: #a855f7;
  }

  .text-blue-500 {
    color: #3b82f6;
  }

  .text-orange-500 {
    color: #f97316;
  }

  .text-cyan-500 {
    color: #06b6d4;
  }

  @media (max-width: 1200px) {
    .top-stats-grid,
    .severity-stats-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (max-width: 768px) {
    .top-stats-grid,
    .severity-stats-grid {
      grid-template-columns: 1fr;
    }

    .dashboard-header {
      flex-direction: column;
      gap: 16px;
    }

    .header-actions {
      width: 100%;
      flex-wrap: wrap;
    }
  }
</style>
