<template>
  <div class="pipeline-dashboard-view">
    <ElEmpty
      v-if="scope === 'project' && !projectId"
      class="dashboard-empty"
      description="请选择顶部 DevOps 项目"
    />

    <div v-else v-loading="loading" class="dashboard-content">
      <section class="overview-panel panel">
        <div class="overview-copy">
          <span class="eyebrow">{{ pageMeta.eyebrow }}</span>
          <h2>{{ pageMeta.title }}</h2>
          <p>{{ pageMeta.description }}</p>
          <div class="hero-tags">
            <span v-for="item in pageMeta.tags" :key="item">{{ item }}</span>
          </div>
        </div>

        <div class="health-card">
          <div class="score-ring" :style="{ '--score': deliveryHealth }">
            <span>{{ formatRate(deliveryHealth) }}</span>
            <em>交付健康度</em>
          </div>
          <div class="health-stats">
            <div>
              <strong>{{ formatNumber(trendTotal) }}</strong>
              <span>近 14 天构建</span>
            </div>
            <div>
              <strong>{{ formatDuration(overview.avgDurationSeconds) }}</strong>
              <span>平均耗时</span>
            </div>
          </div>
        </div>
      </section>

      <section class="kpi-grid">
        <div v-for="item in kpiItems" :key="item.label" class="kpi-card panel" :class="item.tone">
          <span class="kpi-icon">
            <ArtSvgIcon :icon="item.icon" />
          </span>
          <div class="kpi-copy">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
            <em>{{ item.hint }}</em>
          </div>
        </div>
      </section>

      <section class="chart-grid">
        <article class="panel trend-panel">
          <header>
            <div>
              <strong>近 14 天交付趋势</strong>
              <span>构建量、成功数、失败数与平均时长</span>
            </div>
            <span class="panel-count">{{ formatNumber(trendTotal) }} 次</span>
          </header>
          <div ref="trendChartRef" class="chart-box"></div>
        </article>

        <article class="panel result-panel">
          <header>
            <div>
              <strong>结果分布</strong>
              <span>成功、失败、取消占比</span>
            </div>
          </header>
          <div v-if="hasStatusData" ref="statusChartRef" class="chart-box"></div>
          <div v-else class="chart-empty">
            <ArtSvgIcon icon="ri:pie-chart-line" />
            <span>暂无结果数据</span>
          </div>
        </article>
      </section>

      <section class="insight-grid">
        <article class="panel asset-panel">
          <header>
            <div>
              <strong>{{ scope === 'project' ? '项目交付资产' : '平台交付资产' }}</strong>
              <span>{{
                scope === 'project'
                  ? '应用、流水线、环境与渠道承载'
                  : '项目、应用、流水线、渠道与环境规模'
              }}</span>
            </div>
          </header>
          <div class="asset-board">
            <div v-for="(item, index) in flowItems" :key="item.label" class="asset-node">
              <span class="node-index">{{ index + 1 }}</span>
              <div>
                <strong>{{ item.value }}</strong>
                <span>{{ item.label }}</span>
              </div>
            </div>
          </div>
        </article>

        <article class="panel risk-panel">
          <header>
            <div>
              <strong>{{ scope === 'project' ? '风险流水线' : '风险项目' }}</strong>
              <span>按失败率与失败次数定位优先治理对象</span>
            </div>
            <span class="panel-count danger">{{ formatRate(failurePressure) }}</span>
          </header>
          <div v-if="riskRankItems.length" class="risk-list">
            <div
              v-for="(item, index) in riskRankItems.slice(0, 4)"
              :key="`risk-${item.id}-${index}`"
              class="risk-row"
            >
              <span class="risk-no">{{ index + 1 }}</span>
              <div class="risk-main">
                <strong :title="item.name || item.code">{{ item.name || item.code || '-' }}</strong>
                <span
                  >失败 {{ formatNumber(item.failureCount) }} 次 / 构建
                  {{ formatNumber(item.buildCount) }} 次</span
                >
              </div>
              <em>{{ formatRate(item.failureRate) }}</em>
            </div>
          </div>
          <div v-else class="rank-empty">
            <ArtSvgIcon icon="ri:shield-check-line" />
            <span>暂无风险数据</span>
          </div>
        </article>
      </section>

      <section class="rank-grid">
        <article
          v-for="section in rankSections"
          :key="section.key"
          class="panel rank-panel"
          :class="`rank-${section.tone}`"
        >
          <header>
            <div>
              <strong>{{ section.title }}</strong>
              <span>{{ section.subtitle }}</span>
            </div>
            <span class="panel-count">{{ section.items.length }} 个</span>
          </header>
          <div v-if="section.items.length" class="rank-list">
            <div
              v-for="(item, index) in section.items"
              :key="`${section.key}-${item.id}-${index}`"
              class="rank-row"
            >
              <span class="rank-no">{{ index + 1 }}</span>
              <div class="rank-info">
                <div class="rank-title">
                  <strong :title="item.name || item.code">{{
                    item.name || item.code || '-'
                  }}</strong>
                  <span>{{ formatNumber(item.buildCount) }} 次</span>
                </div>
                <div class="rank-track">
                  <span :style="rankBarStyle(item, section.items, section.tone)"></span>
                </div>
              </div>
              <div class="rank-rate">
                <strong>{{ rankMetricValue(section.key, item) }}</strong>
                <span>{{ rankMetricLabel(section.key) }}</span>
              </div>
            </div>
          </div>
          <div v-else class="rank-empty">
            <ArtSvgIcon icon="ri:bar-chart-box-line" />
            <span>暂无数据</span>
          </div>
        </article>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
  import * as echarts from 'echarts'
  import {
    getDevopsPipelineDashboardApi,
    type PipelineDashboardMetric,
    type PipelineDashboardResponse
  } from '@/api/devops/pipeline'

  type RankTone = 'blue' | 'green' | 'amber' | 'red' | 'slate'

  const props = withDefaults(
    defineProps<{
      scope: 'global' | 'project'
      projectId?: string
      projectName?: string
    }>(),
    {
      projectId: '',
      projectName: ''
    }
  )

  const emptyDashboard = (): PipelineDashboardResponse => ({
    overview: {
      buildCount: 0,
      successCount: 0,
      failureCount: 0,
      abortedCount: 0,
      totalDurationSeconds: 0,
      successRate: 0,
      failureRate: 0,
      avgDurationSeconds: 0,
      projectCount: 0,
      systemCount: 0,
      environmentCount: 0,
      pipelineCount: 0,
      channelCount: 0
    },
    trends: [],
    projectRanks: [],
    systemRanks: [],
    pipelineRanks: [],
    environmentRanks: [],
    channelRanks: []
  })

  const loading = ref(false)
  const dashboard = ref<PipelineDashboardResponse>(emptyDashboard())
  const trendChartRef = ref<HTMLElement>()
  const statusChartRef = ref<HTMLElement>()
  let trendChart: echarts.ECharts | null = null
  let statusChart: echarts.ECharts | null = null

  const overview = computed(() => dashboard.value.overview)
  const scope = computed(() => props.scope)
  const projectId = computed(() => props.projectId || '')
  const trendTotal = computed(() =>
    dashboard.value.trends.reduce((sum, item) => sum + Number(item.buildCount || 0), 0)
  )
  const hasStatusData = computed(
    () =>
      overview.value.successCount > 0 ||
      overview.value.failureCount > 0 ||
      overview.value.abortedCount > 0
  )
  const deliveryHealth = computed(() => clampRate(overview.value.successRate))
  const failurePressure = computed(() => clampRate(overview.value.failureRate))

  const pageMeta = computed(() => {
    if (props.scope === 'project') {
      const name = props.projectName || '当前项目'
      return {
        eyebrow: '项目交付视图',
        title: `${name} 项目交付大盘`,
        description: '聚焦单项目内应用、流水线、环境与渠道，清晰呈现交付吞吐、稳定性和失败风险。',
        tags: ['项目视角', '流水线稳定性', '环境承载', '风险治理']
      }
    }
    return {
      eyebrow: '全局交付视图',
      title: 'DevOps 交付驾驶舱',
      description:
        '统一观察项目、应用、流水线、渠道和环境，帮助平台团队快速判断交付健康度与治理优先级。',
      tags: ['全局态势', '双引擎交付', '渠道承载', '趋势洞察']
    }
  })

  const kpiItems = computed(() => [
    {
      label: '构建总数',
      value: formatNumber(overview.value.buildCount),
      hint: `成功 ${formatNumber(overview.value.successCount)} / 失败 ${formatNumber(overview.value.failureCount)}`,
      icon: 'ri:rocket-line',
      tone: 'tone-blue'
    },
    {
      label: '成功率',
      value: formatRate(overview.value.successRate),
      hint: `失败率 ${formatRate(overview.value.failureRate)}`,
      icon: 'ri:shield-check-line',
      tone: 'tone-green'
    },
    {
      label: '累计时长',
      value: formatDuration(overview.value.totalDurationSeconds),
      hint: `平均 ${formatDuration(overview.value.avgDurationSeconds)}`,
      icon: 'ri:time-line',
      tone: 'tone-amber'
    },
    {
      label: props.scope === 'project' ? '应用数量' : '项目数量',
      value: formatNumber(
        props.scope === 'project' ? overview.value.systemCount : overview.value.projectCount
      ),
      hint: `流水线 ${formatNumber(overview.value.pipelineCount)} / 渠道 ${formatNumber(overview.value.channelCount)}`,
      icon: props.scope === 'project' ? 'ri:apps-2-line' : 'ri:folder-3-line',
      tone: 'tone-slate'
    }
  ])

  const flowItems = computed(() => {
    if (props.scope === 'project') {
      return [
        { label: '应用', value: formatNumber(overview.value.systemCount) },
        { label: '流水线', value: formatNumber(overview.value.pipelineCount) },
        { label: '环境', value: formatNumber(overview.value.environmentCount) },
        { label: '渠道', value: formatNumber(overview.value.channelCount) },
        { label: '构建', value: formatNumber(overview.value.buildCount) }
      ]
    }
    return [
      { label: '项目', value: formatNumber(overview.value.projectCount) },
      { label: '应用', value: formatNumber(overview.value.systemCount) },
      { label: '流水线', value: formatNumber(overview.value.pipelineCount) },
      { label: '渠道', value: formatNumber(overview.value.channelCount) },
      { label: '环境', value: formatNumber(overview.value.environmentCount) }
    ]
  })

  const successRankItems = computed(() => sortRankItems(dashboard.value.pipelineRanks, 'success'))
  const riskRankItems = computed(() =>
    sortRankItems(
      props.scope === 'global' ? dashboard.value.projectRanks : dashboard.value.pipelineRanks,
      'risk'
    )
  )

  const rankSections = computed(() => {
    const sections: Array<{
      key: string
      title: string
      subtitle: string
      items: PipelineDashboardMetric[]
      tone: RankTone
    }> = []
    if (props.scope === 'global') {
      sections.push({
        key: 'project',
        title: '项目交付排行',
        subtitle: '按累计构建次数排序',
        items: dashboard.value.projectRanks,
        tone: 'blue'
      })
    }
    sections.push(
      {
        key: 'system',
        title: '应用活跃排行',
        subtitle: '应用交付吞吐与稳定性',
        items: dashboard.value.systemRanks,
        tone: 'green'
      },
      {
        key: 'pipeline',
        title: '流水线执行排行',
        subtitle: '高频流水线执行观察',
        items: dashboard.value.pipelineRanks,
        tone: 'amber'
      },
      {
        key: 'channel',
        title: '构建渠道排行',
        subtitle: '渠道承载构建量',
        items: dashboard.value.channelRanks,
        tone: 'slate'
      },
      {
        key: 'environment',
        title: '环境分布排行',
        subtitle: '不同环境构建分布',
        items: dashboard.value.environmentRanks,
        tone: 'green'
      }
    )
    if (props.scope === 'project') {
      sections.push({
        key: 'success',
        title: '稳定流水线排行',
        subtitle: '按成功率与构建量排序',
        items: successRankItems.value,
        tone: 'blue'
      })
    }
    sections.push({
      key: 'risk',
      title: '失败风险排行',
      subtitle: props.scope === 'global' ? '项目失败率排序' : '流水线失败率排序',
      items: riskRankItems.value,
      tone: 'red'
    })
    return sections
  })

  async function fetchDashboard() {
    if (props.scope === 'project' && !props.projectId) {
      dashboard.value = emptyDashboard()
      return
    }
    loading.value = true
    try {
      dashboard.value = await getDevopsPipelineDashboardApi({
        projectId: props.scope === 'project' ? props.projectId : undefined
      })
      await nextTick()
      updateCharts()
    } finally {
      loading.value = false
    }
  }

  function initCharts() {
    if (trendChartRef.value && !trendChart) {
      trendChart = echarts.init(trendChartRef.value)
    }
    if (hasStatusData.value && statusChartRef.value && !statusChart) {
      statusChart = echarts.init(statusChartRef.value)
    }
  }

  function updateCharts() {
    initCharts()
    updateTrendChart()
    updateStatusChart()
  }

  function updateTrendChart() {
    const rows = dashboard.value.trends || []
    trendChart?.setOption({
      backgroundColor: 'transparent',
      color: ['#4f6bff', '#16a34a', '#ef4444', '#f59e0b'],
      tooltip: {
        trigger: 'axis',
        backgroundColor: '#ffffff',
        borderColor: '#dbe4f0',
        borderWidth: 1,
        padding: [10, 12],
        textStyle: { color: '#334155' }
      },
      legend: {
        type: 'scroll',
        top: 2,
        right: 8,
        width: 360,
        itemGap: 16,
        icon: 'roundRect',
        textStyle: { fontSize: 12, color: '#64748b' },
        pageIconSize: 10
      },
      grid: { left: 38, right: 58, top: 52, bottom: 32, containLabel: true },
      xAxis: {
        type: 'category',
        boundaryGap: true,
        data: rows.map((item) => item.date.slice(5)),
        axisTick: { show: false },
        axisLine: { lineStyle: { color: '#dbe4f0' } },
        axisLabel: { color: '#64748b' }
      },
      yAxis: [
        {
          type: 'value',
          minInterval: 1,
          name: '次数',
          nameTextStyle: { color: '#94a3b8' },
          axisLine: { show: false },
          axisLabel: { color: '#64748b' },
          splitLine: { lineStyle: { color: '#eef2f7' } }
        },
        {
          type: 'value',
          nameTextStyle: { color: '#94a3b8' },
          axisLine: { show: false },
          axisLabel: {
            color: '#64748b',
            formatter: (value: number) => formatShortDuration(value)
          },
          splitLine: { show: false }
        }
      ],
      series: [
        {
          name: '构建',
          type: 'bar',
          barWidth: 16,
          itemStyle: {
            borderRadius: [6, 6, 0, 0],
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: '#5b7cff' },
              { offset: 1, color: 'rgba(91, 124, 255, 0.18)' }
            ])
          },
          data: rows.map((item) => item.buildCount)
        },
        {
          name: '成功',
          type: 'line',
          smooth: true,
          symbolSize: 6,
          lineStyle: { width: 3 },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(22, 163, 74, 0.18)' },
              { offset: 1, color: 'rgba(22, 163, 74, 0)' }
            ])
          },
          data: rows.map((item) => item.successCount)
        },
        {
          name: '失败',
          type: 'line',
          smooth: true,
          symbolSize: 6,
          lineStyle: { width: 3 },
          data: rows.map((item) => item.failureCount)
        },
        {
          name: '平均时长',
          type: 'line',
          yAxisIndex: 1,
          smooth: true,
          symbolSize: 6,
          lineStyle: { width: 2, type: 'dashed' },
          data: rows.map((item) => safeAverage(item.durationSeconds, item.buildCount))
        }
      ]
    })
  }

  function updateStatusChart() {
    if (!hasStatusData.value) {
      statusChart?.clear()
      return
    }
    const rows = [
      { name: '成功', value: overview.value.successCount },
      { name: '失败', value: overview.value.failureCount },
      { name: '取消', value: overview.value.abortedCount }
    ]
    statusChart?.setOption({
      backgroundColor: 'transparent',
      color: ['#16a34a', '#ef4444', '#94a3b8'],
      tooltip: {
        trigger: 'item',
        formatter: '{b}: {c} ({d}%)',
        backgroundColor: '#ffffff',
        borderColor: '#dbe4f0',
        borderWidth: 1,
        textStyle: { color: '#334155' }
      },
      legend: {
        bottom: 4,
        itemGap: 18,
        textStyle: { color: '#64748b' }
      },
      graphic: [
        {
          type: 'text',
          left: 'center',
          top: '39%',
          style: {
            text: formatRate(overview.value.successRate),
            fill: '#1e293b',
            fontSize: 24,
            fontWeight: 700,
            textAlign: 'center'
          }
        },
        {
          type: 'text',
          left: 'center',
          top: '51%',
          style: {
            text: '成功率',
            fill: '#64748b',
            fontSize: 12,
            textAlign: 'center'
          }
        }
      ],
      series: [
        {
          name: '结果分布',
          type: 'pie',
          radius: ['58%', '78%'],
          center: ['50%', '44%'],
          avoidLabelOverlap: true,
          label: { show: false },
          labelLine: { show: false },
          itemStyle: {
            borderColor: '#ffffff',
            borderWidth: 4
          },
          data: rows
        }
      ]
    })
  }

  function resizeCharts() {
    trendChart?.resize()
    statusChart?.resize()
  }

  function disposeCharts() {
    window.removeEventListener('resize', resizeCharts)
    trendChart?.dispose()
    statusChart?.dispose()
    trendChart = null
    statusChart = null
  }

  function formatNumber(value: number) {
    return new Intl.NumberFormat('zh-CN').format(Math.round(Number(value || 0)))
  }

  function formatRate(value: number) {
    return `${Number(value || 0).toFixed(1)}%`
  }

  function formatDuration(seconds: number) {
    const value = Math.max(0, Math.round(Number(seconds || 0)))
    if (value >= 3600) {
      return `${(value / 3600).toFixed(1)} 小时`
    }
    if (value >= 60) {
      return `${Math.round(value / 60)} 分钟`
    }
    return `${value} 秒`
  }

  function formatShortDuration(seconds: number) {
    const value = Math.max(0, Math.round(Number(seconds || 0)))
    if (value >= 3600) return `${Math.round(value / 3600)}h`
    if (value >= 60) return `${Math.round(value / 60)}m`
    return `${value}s`
  }

  function safeAverage(total: number, count: number) {
    if (!count) return 0
    return Math.round(Number(total || 0) / Number(count || 1))
  }

  function clampRate(value: number) {
    return Math.min(100, Math.max(0, Number(value || 0)))
  }

  function sortRankItems(items: PipelineDashboardMetric[], mode: 'success' | 'risk') {
    return [...items]
      .filter((item) => Number(item.buildCount || 0) > 0)
      .sort((prev, next) => {
        if (mode === 'success') {
          return (
            Number(next.successRate || 0) - Number(prev.successRate || 0) ||
            Number(next.buildCount || 0) - Number(prev.buildCount || 0)
          )
        }
        return (
          Number(next.failureRate || 0) - Number(prev.failureRate || 0) ||
          Number(next.failureCount || 0) - Number(prev.failureCount || 0)
        )
      })
      .slice(0, 8)
  }

  function rankMetricLabel(sectionKey: string) {
    return sectionKey === 'risk' ? '失败率' : '成功率'
  }

  function rankMetricValue(sectionKey: string, item: PipelineDashboardMetric) {
    return sectionKey === 'risk' ? formatRate(item.failureRate) : formatRate(item.successRate)
  }

  function rankBarStyle(
    item: PipelineDashboardMetric,
    items: PipelineDashboardMetric[],
    tone: RankTone
  ) {
    const max = Math.max(...items.map((target) => Number(target.buildCount || 0)), 1)
    const width = Math.max(6, Math.round((Number(item.buildCount || 0) / max) * 100))
    const toneMap: Record<RankTone, string> = {
      blue: 'linear-gradient(90deg, #5b7cff, #3b82f6)',
      green: 'linear-gradient(90deg, #16a34a, #22c55e)',
      amber: 'linear-gradient(90deg, #f59e0b, #facc15)',
      red: 'linear-gradient(90deg, #ef4444, #f97316)',
      slate: 'linear-gradient(90deg, #64748b, #3b82f6)'
    }
    return { width: `${width}%`, background: toneMap[tone] }
  }

  watch(
    () => [props.scope, props.projectId],
    () => {
      fetchDashboard()
    },
    { immediate: true }
  )

  onMounted(() => {
    window.addEventListener('resize', resizeCharts)
    nextTick(updateCharts)
  })

  onBeforeUnmount(() => {
    disposeCharts()
  })
</script>

<style scoped lang="scss">
  .pipeline-dashboard-view {
    height: 100%;
    min-height: 0;
    overflow: hidden;
    color: #1e293b;
  }

  .dashboard-content {
    display: flex;
    flex-direction: column;
    gap: 16px;
    height: 100%;
    min-height: 0;
    overflow: auto;
    background:
      linear-gradient(180deg, #f5f7fc 0%, #f8fafc 44%, #ffffff 100%),
      linear-gradient(90deg, rgba(79, 107, 255, 0.06) 1px, transparent 1px),
      linear-gradient(rgba(79, 107, 255, 0.05) 1px, transparent 1px);
    background-size:
      auto,
      28px 28px,
      28px 28px;
  }

  .dashboard-empty {
    height: 100%;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .panel {
    min-width: 0;
    background: rgba(255, 255, 255, 0.94);
    border: 1px solid #e5ebf5;
    border-radius: 8px;
    box-shadow: 0 10px 28px rgba(30, 41, 59, 0.06);

    > header {
      display: flex;
      gap: 12px;
      align-items: center;
      justify-content: space-between;
      min-height: 68px;
      padding: 18px 20px 12px;
      border-bottom: 1px solid #edf1f7;

      div {
        min-width: 0;
      }

      > div strong,
      > div span {
        display: block;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      > div strong {
        font-size: 15px;
        color: #1e293b;
      }

      > div span {
        margin-top: 5px;
        font-size: 12px;
        color: #64748b;
      }
    }
  }

  .eyebrow,
  .hero-tags span,
  .panel-count,
  .node-index,
  .risk-no,
  .rank-no {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    box-sizing: border-box;
    line-height: 1;
    text-align: center;
    font-variant-numeric: tabular-nums;
  }

  .overview-panel {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 360px;
    gap: 24px;
    align-items: stretch;
    min-height: 220px;
    padding: 28px;
    background:
      linear-gradient(135deg, #ffffff 0%, #f4f7ff 52%, #edf5ff 100%),
      linear-gradient(90deg, rgba(79, 107, 255, 0.08) 1px, transparent 1px),
      linear-gradient(rgba(79, 107, 255, 0.06) 1px, transparent 1px);
    background-size:
      auto,
      32px 32px,
      32px 32px;
    border-color: #dbe6ff;
  }

  .overview-copy {
    display: flex;
    flex-direction: column;
    justify-content: center;
    min-width: 0;
  }

  .eyebrow {
    width: fit-content;
    min-height: 26px;
    padding: 5px 10px;
    font-size: 12px;
    font-weight: 700;
    color: #335cff;
    background: rgba(79, 107, 255, 0.1);
    border: 1px solid rgba(79, 107, 255, 0.18);
    border-radius: 4px;
  }

  .overview-copy h2 {
    margin: 16px 0 10px;
    overflow-wrap: anywhere;
    font-size: 30px;
    font-weight: 800;
    line-height: 1.2;
    color: #111827;
  }

  .overview-copy p {
    max-width: 760px;
    margin: 0;
    font-size: 14px;
    line-height: 1.8;
    color: #475569;
  }

  .hero-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    margin-top: 22px;

    span {
      height: 28px;
      padding: 0 12px;
      font-size: 12px;
      color: #334155;
      background: #ffffff;
      border: 1px solid #dbe4f0;
      border-radius: 4px;
      box-shadow: 0 4px 12px rgba(30, 41, 59, 0.04);
    }
  }

  .health-card {
    display: grid;
    grid-template-columns: 144px minmax(0, 1fr);
    gap: 16px;
    align-items: center;
    padding: 18px;
    background: rgba(255, 255, 255, 0.84);
    border: 1px solid #dbe6ff;
    border-radius: 8px;
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.8);
  }

  .score-ring {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    width: 144px;
    aspect-ratio: 1;
    background:
      radial-gradient(circle at center, #ffffff 59%, transparent 60%),
      conic-gradient(#4f6bff calc(var(--score) * 1%), #e5ebf5 0);
    border-radius: 50%;

    span,
    em {
      display: block;
      max-width: 108px;
      overflow: hidden;
      text-align: center;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    span {
      font-size: 26px;
      font-weight: 800;
      color: #1e293b;
    }

    em {
      margin-top: 6px;
      font-size: 12px;
      font-style: normal;
      color: #64748b;
    }
  }

  .health-stats {
    display: grid;
    gap: 12px;

    div {
      min-width: 0;
      padding: 14px;
      background: #f8fbff;
      border: 1px solid #e5ebf5;
      border-radius: 6px;
    }

    strong,
    span {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      font-size: 20px;
      color: #111827;
    }

    span {
      margin-top: 6px;
      font-size: 12px;
      color: #64748b;
    }
  }

  .kpi-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 16px;
  }

  .kpi-card {
    display: flex;
    gap: 16px;
    align-items: center;
    min-height: 112px;
    padding: 20px;
    transition:
      transform 0.2s ease,
      border-color 0.2s ease,
      box-shadow 0.2s ease;

    &:hover {
      border-color: rgba(79, 107, 255, 0.28);
      box-shadow: 0 14px 36px rgba(30, 41, 59, 0.09);
      transform: translateY(-2px);
    }

    .kpi-icon {
      display: inline-flex;
      flex: 0 0 50px;
      align-items: center;
      justify-content: center;
      box-sizing: border-box;
      width: 50px;
      height: 50px;
      font-size: 24px;
      line-height: 1;
      color: var(--tone-color);
      background: color-mix(in srgb, var(--tone-color) 10%, white);
      border: 1px solid color-mix(in srgb, var(--tone-color) 18%, white);
      border-radius: 8px;
    }

    .kpi-copy {
      min-width: 0;
    }

    .kpi-copy span,
    .kpi-copy strong,
    .kpi-copy em {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .kpi-copy span {
      font-size: 12px;
      color: #64748b;
    }

    .kpi-copy strong {
      margin-top: 7px;
      font-size: 28px;
      line-height: 1.2;
      color: #111827;
      font-variant-numeric: tabular-nums;
    }

    .kpi-copy em {
      margin-top: 7px;
      font-size: 12px;
      font-style: normal;
      color: #94a3b8;
    }
  }

  .tone-blue {
    --tone-color: #4f6bff;
  }

  .tone-green {
    --tone-color: #16a34a;
  }

  .tone-amber {
    --tone-color: #f59e0b;
  }

  .tone-slate {
    --tone-color: #64748b;
  }

  .chart-grid,
  .insight-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.55fr) minmax(320px, 0.75fr);
    gap: 16px;
    min-height: 360px;
  }

  .chart-box {
    width: 100%;
    height: 300px;
  }

  .chart-empty {
    display: flex;
    flex-direction: column;
    gap: 10px;
    align-items: center;
    justify-content: center;
    height: 300px;
    color: #64748b;

    .art-svg-icon {
      font-size: 38px;
      color: #94a3b8;
    }

    span {
      font-size: 13px;
    }
  }

  .panel-count {
    flex: 0 0 auto;
    min-width: 60px;
    height: 28px;
    padding: 0 12px;
    font-size: 12px;
    font-weight: 700;
    color: #335cff;
    white-space: nowrap;
    background: rgba(79, 107, 255, 0.08);
    border: 1px solid rgba(79, 107, 255, 0.16);
    border-radius: 4px;

    &.danger {
      color: #dc2626;
      background: rgba(239, 68, 68, 0.08);
      border-color: rgba(239, 68, 68, 0.16);
    }
  }

  .asset-board {
    display: grid;
    grid-template-columns: repeat(5, minmax(0, 1fr));
    gap: 12px;
    padding: 20px;
  }

  .asset-node {
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    min-height: 150px;
    padding: 16px;
    background: linear-gradient(180deg, #ffffff 0%, #f8fbff 100%);
    border: 1px solid #e5ebf5;
    border-radius: 8px;
  }

  .node-index {
    width: 28px;
    height: 24px;
    font-size: 12px;
    font-weight: 800;
    color: #335cff;
    background: rgba(79, 107, 255, 0.1);
    border-radius: 4px;
  }

  .asset-node div strong,
  .asset-node div span {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .asset-node strong {
    font-size: 26px;
    color: #111827;
  }

  .asset-node div span {
    margin-top: 8px;
    font-size: 12px;
    color: #64748b;
  }

  .risk-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 18px 20px 20px;
  }

  .risk-row {
    display: grid;
    grid-template-columns: 30px minmax(0, 1fr) 58px;
    gap: 12px;
    align-items: center;
    min-height: 58px;
    padding: 11px 12px;
    background: #fff7f7;
    border: 1px solid #fee2e2;
    border-radius: 8px;
  }

  .risk-no {
    width: 30px;
    height: 30px;
    font-size: 13px;
    font-weight: 800;
    color: #dc2626;
    background: #fee2e2;
    border-radius: 6px;
  }

  .risk-main {
    min-width: 0;

    strong,
    span {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      font-size: 13px;
      color: #1e293b;
    }

    span {
      margin-top: 6px;
      font-size: 12px;
      color: #b45353;
    }
  }

  .risk-row em {
    overflow: hidden;
    font-size: 13px;
    font-style: normal;
    font-weight: 800;
    color: #dc2626;
    text-align: right;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .rank-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 16px;
    padding-bottom: 16px;
  }

  .rank-panel {
    min-height: 294px;
  }

  .rank-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 16px 20px 20px;
  }

  .rank-row {
    display: grid;
    grid-template-columns: 32px minmax(0, 1fr) 76px;
    gap: 12px;
    align-items: center;
    min-height: 52px;
    padding: 10px 12px;
    background: #f8fbff;
    border: 1px solid #edf1f7;
    border-radius: 8px;
    transition:
      background 0.2s ease,
      border-color 0.2s ease;

    &:hover {
      background: #ffffff;
      border-color: rgba(79, 107, 255, 0.24);
    }
  }

  .rank-no {
    width: 32px;
    height: 32px;
    font-size: 14px;
    font-weight: 800;
    color: #335cff;
    background: rgba(79, 107, 255, 0.09);
    border-radius: 6px;
  }

  .rank-info {
    min-width: 0;
  }

  .rank-title {
    display: flex;
    gap: 10px;
    align-items: center;
    justify-content: space-between;
    min-width: 0;

    strong,
    span {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      min-width: 0;
      font-size: 13px;
      color: #1e293b;
    }

    span {
      flex: 0 0 auto;
      font-size: 12px;
      color: #64748b;
    }
  }

  .rank-track {
    height: 6px;
    margin-top: 9px;
    overflow: hidden;
    background: #e5ebf5;
    border-radius: 3px;

    span {
      display: block;
      height: 100%;
      border-radius: inherit;
    }
  }

  .rank-rate {
    text-align: right;

    strong,
    span {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      font-size: 14px;
      color: #1e293b;
    }

    span {
      margin-top: 5px;
      font-size: 11px;
      color: #64748b;
    }
  }

  .rank-red .rank-rate strong {
    color: #dc2626;
  }

  .rank-green .rank-rate strong,
  .rank-blue .rank-rate strong {
    color: #16a34a;
  }

  .rank-empty {
    display: flex;
    flex-direction: column;
    gap: 8px;
    align-items: center;
    justify-content: center;
    height: 180px;
    color: #64748b;

    .art-svg-icon {
      font-size: 30px;
      color: #94a3b8;
    }

    span {
      font-size: 12px;
    }
  }

  @media (max-width: 1360px) {
    .overview-panel {
      grid-template-columns: 1fr;
    }

    .health-card {
      grid-template-columns: 144px minmax(0, 1fr);
    }

    .kpi-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .chart-grid,
    .insight-grid,
    .rank-grid {
      grid-template-columns: 1fr;
    }
  }

  @media (max-width: 900px) {
    .dashboard-content {
      padding: 12px;
    }

    .overview-copy h2 {
      font-size: 26px;
    }

    .health-card,
    .asset-board {
      grid-template-columns: 1fr;
    }

    .score-ring {
      margin: 0 auto;
    }

    .kpi-grid {
      grid-template-columns: 1fr;
    }

    .chart-box {
      height: 260px;
    }

    .risk-row,
    .rank-row {
      grid-template-columns: 30px minmax(0, 1fr);
    }

    .risk-row em,
    .rank-rate {
      grid-column: 2;
      text-align: left;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .kpi-card,
    .rank-row {
      transition: none;
    }

    .kpi-card:hover {
      transform: none;
    }
  }
</style>
