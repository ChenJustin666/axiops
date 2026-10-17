<template>
  <div class="billing-dashboard">
    <!-- 页面头部 -->
    <div class="dashboard-header">
      <div>
        <h1 class="page-title">费用中心</h1>
        <p class="page-subtitle">账单费用统计与分析总览</p>
      </div>
      <div class="header-actions">
        <!-- 快捷时间选择 -->
        <ElRadioGroup v-model="quickTimeRange" size="default" @change="handleQuickTimeChange">
          <ElRadioButton value="last30days">最近30天</ElRadioButton>
          <ElRadioButton value="thisMonth">本月</ElRadioButton>
          <ElRadioButton value="lastMonth">上月</ElRadioButton>
          <ElRadioButton value="thisYear">本年</ElRadioButton>
          <ElRadioButton value="custom">自定义</ElRadioButton>
        </ElRadioGroup>
        <!-- 自定义日期范围选择器 -->
        <ElDatePicker
          v-if="quickTimeRange === 'custom'"
          v-model="filterForm.dateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          format="YYYY-MM-DD"
          value-format="YYYY-MM-DD"
          :clearable="false"
          style="width: 260px"
          @change="handleDateRangeChange"
        />
        <ElSelect
          v-model="filterForm.clusterUuid"
          placeholder="全部集群"
          clearable
          style="width: 160px"
        >
          <ElOption label="全部集群" value="" />
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
        >
          <ElOption label="全部项目" :value="0" />
          <ElOption
            v-for="project in projectList"
            :key="project.id"
            :label="project.name"
            :value="project.id"
          />
        </ElSelect>
        <ElButton type="primary" @click="handleQuery">
          <ArtSvgIcon icon="ri:search-line" class="mr-1" />
          查询
        </ElButton>
        <ElButton @click="refreshAllData">
          <ArtSvgIcon icon="ri:refresh-line" class="mr-1" />
          刷新
        </ElButton>
      </div>
    </div>

    <!-- 统计卡片区域 -->
    <div class="stats-grid">
      <!-- 总费用 -->
      <div class="stat-card">
        <div class="stat-card-header">
          <div class="stat-card-icon stat-icon-blue">
            <ArtSvgIcon icon="ri:money-cny-circle-line" />
          </div>
          <div class="stat-card-mom" :class="getMomClass(statsData.totalCostMom)">
            {{ formatMomChange(statsData.totalCostMom) }}
          </div>
        </div>
        <div class="stat-card-label">{{ getTimeRangeLabel() }}总费用</div>
        <div class="stat-card-value">¥ {{ formatAmount(statsData.totalCost) }}</div>
        <div class="stat-card-sub"
          >同比上期：¥
          {{ formatAmount(getLastPeriodCost(statsData.totalCost, statsData.totalCostMom)) }}</div
        >
      </div>

      <!-- 资源费用 -->
      <div class="stat-card">
        <div class="stat-card-header">
          <div class="stat-card-icon stat-icon-green">
            <ArtSvgIcon icon="ri:server-line" />
          </div>
          <div class="stat-card-mom" :class="getMomClass(statsData.resourceCostMom)">
            {{ formatMomChange(statsData.resourceCostMom) }}
          </div>
        </div>
        <div class="stat-card-label">资源费用</div>
        <div class="stat-card-value">¥ {{ formatAmount(statsData.resourceCost) }}</div>
        <div class="stat-card-sub">CPU+内存+存储+GPU+Pod</div>
      </div>

      <!-- 管理费 -->
      <div class="stat-card">
        <div class="stat-card-header">
          <div class="stat-card-icon stat-icon-orange">
            <ArtSvgIcon icon="ri:settings-3-line" />
          </div>
          <div class="stat-card-mom" :class="getMomClass(statsData.managementCostMom)">
            {{ formatMomChange(statsData.managementCostMom) }}
          </div>
        </div>
        <div class="stat-card-label">管理费</div>
        <div class="stat-card-value">¥ {{ formatAmount(statsData.managementCost) }}</div>
        <div class="stat-card-sub">运维服务管理费用</div>
      </div>

      <!-- 账单数量 -->
      <div class="stat-card">
        <div class="stat-card-header">
          <div class="stat-card-icon stat-icon-purple">
            <ArtSvgIcon icon="ri:file-list-3-line" />
          </div>
        </div>
        <div class="stat-card-label">账单数量</div>
        <div class="stat-card-value">
          <ArtCountTo :target="statsData.statementCount" :duration="1000" />
        </div>
        <div class="stat-card-sub">涉及 {{ statsData.projectCount }} 个项目</div>
      </div>
    </div>

    <!-- 费用构成 + 费用趋势 -->
    <ElRow :gutter="20" class="mb-5">
      <!-- 费用构成 -->
      <ElCol :span="10">
        <div class="art-card chart-card">
          <div class="card-header">
            <ArtSvgIcon icon="ri:pie-chart-line" class="header-icon text-blue-500" />
            <span class="header-title">费用构成</span>
          </div>
          <div class="chart-container" ref="compositionChartRef"></div>
          <div class="composition-legend">
            <div
              v-for="(item, index) in compositionData.items"
              :key="item.costName"
              class="legend-item"
            >
              <span class="legend-dot" :style="{ background: getCostColor(index) }"></span>
              <span class="legend-name">{{ item.costName }}</span>
              <span class="legend-percent">{{ item.percentage.toFixed(1) }}%</span>
            </div>
          </div>
        </div>
      </ElCol>

      <!-- 费用趋势 -->
      <ElCol :span="14">
        <div class="art-card chart-card">
          <div class="card-header">
            <ArtSvgIcon icon="ri:line-chart-line" class="header-icon text-green-500" />
            <span class="header-title">费用趋势（近{{ trendMonths }}个月）</span>
            <div class="header-actions">
              <ElRadioGroup v-model="trendMonths" size="small" @change="loadTrendData">
                <ElRadioButton :value="3">3个月</ElRadioButton>
                <ElRadioButton :value="6">6个月</ElRadioButton>
                <ElRadioButton :value="12">12个月</ElRadioButton>
              </ElRadioGroup>
            </div>
          </div>
          <div class="chart-container" ref="trendChartRef"></div>
        </div>
      </ElCol>
    </ElRow>

    <!-- 费用排行榜 -->
    <ElRow :gutter="20">
      <!-- 项目费用 TOP -->
      <ElCol :span="12">
        <div class="art-card ranking-card">
          <div class="card-header">
            <ArtSvgIcon icon="ri:folder-chart-line" class="header-icon text-purple-500" />
            <span class="header-title">项目费用 TOP{{ topN }}</span>
            <div class="header-actions">
              <ElSelect v-model="topN" size="small" style="width: 100px" @change="loadRankingData">
                <ElOption label="TOP 5" :value="5" />
                <ElOption label="TOP 10" :value="10" />
                <ElOption label="TOP 20" :value="20" />
              </ElSelect>
            </div>
          </div>
          <div class="ranking-list">
            <template v-if="projectTopData.items.length > 0">
              <div
                v-for="(item, index) in projectTopData.items"
                :key="item.projectId"
                class="ranking-item"
              >
                <div class="ranking-rank" :class="getRankClass(index + 1)">{{ index + 1 }}</div>
                <div class="ranking-info">
                  <div class="ranking-name">{{ item.projectName }}</div>
                  <div class="ranking-bar-wrapper">
                    <div
                      class="ranking-bar ranking-bar-purple"
                      :style="{ width: getBarWidth(item.totalCost, projectMaxCost) }"
                    ></div>
                  </div>
                </div>
                <div class="ranking-value">¥ {{ formatAmount(item.totalCost) }}</div>
              </div>
            </template>
            <div v-else class="ranking-empty">
              <ArtSvgIcon icon="ri:inbox-line" class="empty-icon" />
              <span>暂无数据</span>
            </div>
          </div>
        </div>
      </ElCol>

      <!-- 集群费用 TOP -->
      <ElCol :span="12">
        <div class="art-card ranking-card">
          <div class="card-header">
            <ArtSvgIcon icon="ri:server-line" class="header-icon text-cyan-500" />
            <span class="header-title">集群费用 TOP{{ topN }}</span>
          </div>
          <div class="ranking-list">
            <template v-if="clusterTopData.items.length > 0">
              <div
                v-for="(item, index) in clusterTopData.items"
                :key="item.clusterUuid"
                class="ranking-item"
              >
                <div class="ranking-rank" :class="getRankClass(index + 1)">{{ index + 1 }}</div>
                <div class="ranking-info">
                  <div class="ranking-name">
                    {{ item.clusterName }}
                    <span class="ranking-tag">{{ item.projectCount }}个项目</span>
                  </div>
                  <div class="ranking-bar-wrapper">
                    <div
                      class="ranking-bar ranking-bar-cyan"
                      :style="{ width: getBarWidth(item.totalCost, clusterMaxCost) }"
                    ></div>
                  </div>
                </div>
                <div class="ranking-value">¥ {{ formatAmount(item.totalCost) }}</div>
              </div>
            </template>
            <div v-else class="ranking-empty">
              <ArtSvgIcon icon="ri:inbox-line" class="empty-icon" />
              <span>暂无数据</span>
            </div>
          </div>
        </div>
      </ElCol>
    </ElRow>
  </div>
</template>

<script setup lang="ts">
  import { ref, reactive, computed, onMounted, onUnmounted, nextTick } from 'vue'
  import * as echarts from 'echarts'
  import {
    getBillingDashboardStatsApi,
    getBillingCostCompositionApi,
    getBillingCostTrendApi,
    getBillingProjectTopApi,
    getBillingClusterTopApi,
    formatAmount,
    COST_COMPOSITION_COLORS,
    type BillingDashboardStatsResponse,
    type BillingCostCompositionResponse,
    type BillingCostTrendResponse,
    type BillingProjectTopResponse,
    type BillingClusterTopResponse
  } from '@/api/manager/billing'
  import { searchClusterApi, type Cluster } from '@/api/manager/cluster'
  import { getProjectsByUserApi, type Project } from '@/api/manager/project'

  defineOptions({ name: 'BillingDashboard' })

  // 快捷时间范围类型
  type QuickTimeRangeType = 'last30days' | 'thisMonth' | 'lastMonth' | 'thisYear' | 'custom'

  // 快捷时间范围，默认最近30天
  const quickTimeRange = ref<QuickTimeRangeType>('last30days')

  // 获取最近30天的日期范围
  const getLast30DaysRange = (): [string, string] => {
    const end = new Date()
    const start = new Date()
    start.setDate(start.getDate() - 29) // 包含今天，所以是29天前
    return [formatDate(start), formatDate(end)]
  }

  // 获取本月日期范围
  const getThisMonthRange = (): [string, string] => {
    const now = new Date()
    const start = new Date(now.getFullYear(), now.getMonth(), 1)
    const end = new Date(now.getFullYear(), now.getMonth() + 1, 0)
    return [formatDate(start), formatDate(end)]
  }

  // 获取上月日期范围
  const getLastMonthRange = (): [string, string] => {
    const now = new Date()
    const start = new Date(now.getFullYear(), now.getMonth() - 1, 1)
    const end = new Date(now.getFullYear(), now.getMonth(), 0)
    return [formatDate(start), formatDate(end)]
  }

  // 获取本年日期范围
  const getThisYearRange = (): [string, string] => {
    const now = new Date()
    const start = new Date(now.getFullYear(), 0, 1)
    const end = new Date(now.getFullYear(), 11, 31)
    return [formatDate(start), formatDate(end)]
  }

  // 格式化日期为 YYYY-MM-DD
  const formatDate = (date: Date): string => {
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    return `${year}-${month}-${day}`
  }

  // 筛选表单
  const filterForm = reactive({
    dateRange: getLast30DaysRange() as [string, string],
    clusterUuid: '',
    projectId: 0
  })

  // 下拉选项数据
  const clusterList = ref<Cluster[]>([])
  const projectList = ref<Project[]>([])

  // 统计数据
  const statsData = ref<BillingDashboardStatsResponse>({
    totalCost: 0,
    totalCostMom: 0,
    resourceCost: 0,
    resourceCostMom: 0,
    managementCost: 0,
    managementCostMom: 0,
    statementCount: 0,
    projectCount: 0
  })

  // 费用构成数据
  const compositionData = ref<BillingCostCompositionResponse>({
    items: [],
    totalCost: 0
  })

  // 费用趋势数据
  const trendData = ref<BillingCostTrendResponse>({ items: [] })
  const trendMonths = ref(6)

  // 排行榜数据
  const topN = ref(10)
  const projectTopData = ref<BillingProjectTopResponse>({ items: [] })
  const clusterTopData = ref<BillingClusterTopResponse>({ items: [] })

  // 图表实例
  const compositionChartRef = ref<HTMLElement>()
  const trendChartRef = ref<HTMLElement>()
  let compositionChart: echarts.ECharts | null = null
  let trendChart: echarts.ECharts | null = null

  // 计算最大费用（用于进度条宽度）
  const projectMaxCost = computed(() => {
    if (projectTopData.value.items.length === 0) return 1
    return Math.max(...projectTopData.value.items.map((item) => item.totalCost))
  })

  const clusterMaxCost = computed(() => {
    if (clusterTopData.value.items.length === 0) return 1
    return Math.max(...clusterTopData.value.items.map((item) => item.totalCost))
  })

  // 获取颜色
  const getCostColor = (index: number) =>
    COST_COMPOSITION_COLORS[index % COST_COMPOSITION_COLORS.length]

  // 格式化环比变化
  const formatMomChange = (value: number) => {
    const sign = value >= 0 ? '+' : ''
    return `${sign}${value.toFixed(1)}%`
  }

  // 获取环比样式类
  const getMomClass = (value: number) => {
    if (value > 0) return 'mom-up'
    if (value < 0) return 'mom-down'
    return 'mom-flat'
  }

  // 计算上期费用
  const getLastPeriodCost = (current: number, mom: number) => {
    if (mom === 0) return current
    return current / (1 + mom / 100)
  }

  // 排名样式
  const getRankClass = (rank: number) => {
    if (rank === 1) return 'rank-gold'
    if (rank === 2) return 'rank-silver'
    if (rank === 3) return 'rank-bronze'
    return 'rank-default'
  }

  // 计算进度条宽度
  const getBarWidth = (value: number, max: number) => {
    if (max === 0) return '0%'
    const percent = (value / max) * 100
    return `${Math.max(percent, 5)}%`
  }

  // 获取时间范围标签
  const getTimeRangeLabel = (): string => {
    switch (quickTimeRange.value) {
      case 'last30days':
        return '近30天'
      case 'thisMonth':
        return '本月'
      case 'lastMonth':
        return '上月'
      case 'thisYear':
        return '本年'
      case 'custom':
        return '所选期间'
      default:
        return ''
    }
  }

  // 获取查询的时间区间（秒级时间戳）
  const getQueryTimeRange = (): { startTime: number; endTime: number } => {
    const [startStr, endStr] = filterForm.dateRange
    const startDate = new Date(startStr)
    startDate.setHours(0, 0, 0, 0)
    const endDate = new Date(endStr)
    endDate.setHours(23, 59, 59, 999)
    return {
      startTime: Math.floor(startDate.getTime() / 1000),
      endTime: Math.floor(endDate.getTime() / 1000)
    }
  }

  // 处理快捷时间范围切换
  const handleQuickTimeChange = (value: QuickTimeRangeType) => {
    switch (value) {
      case 'last30days':
        filterForm.dateRange = getLast30DaysRange()
        break
      case 'thisMonth':
        filterForm.dateRange = getThisMonthRange()
        break
      case 'lastMonth':
        filterForm.dateRange = getLastMonthRange()
        break
      case 'thisYear':
        filterForm.dateRange = getThisYearRange()
        break
      case 'custom':
        // 自定义模式保持当前日期范围不变
        return
    }
    refreshAllData()
  }

  // 处理自定义日期范围变化
  const handleDateRangeChange = () => {
    if (filterForm.dateRange && filterForm.dateRange.length === 2) {
      refreshAllData()
    }
  }

  // 加载集群列表
  const loadClusterList = async () => {
    try {
      const res = await searchClusterApi({ pageSize: 100 })
      clusterList.value = res.items || []
    } catch (e) {}
  }

  // 加载项目列表
  const loadProjectList = async () => {
    try {
      const res = await getProjectsByUserApi()
      projectList.value = res || []
    } catch (e) {}
  }

  // 加载统计数据
  const loadStatsData = async () => {
    try {
      const { startTime, endTime } = getQueryTimeRange()
      const res = await getBillingDashboardStatsApi({
        startTime,
        endTime,
        clusterUuid: filterForm.clusterUuid || undefined,
        projectId: filterForm.projectId || undefined
      })
      statsData.value = res
    } catch (e) {}
  }

  // 加载费用构成数据
  const loadCompositionData = async () => {
    try {
      const { startTime, endTime } = getQueryTimeRange()
      const res = await getBillingCostCompositionApi({
        startTime,
        endTime,
        clusterUuid: filterForm.clusterUuid || undefined,
        projectId: filterForm.projectId || undefined
      })
      compositionData.value = res
      await nextTick()
      renderCompositionChart()
    } catch (e) {}
  }

  // 加载费用趋势数据
  const loadTrendData = async () => {
    try {
      const res = await getBillingCostTrendApi({
        months: trendMonths.value,
        clusterUuid: filterForm.clusterUuid || undefined,
        projectId: filterForm.projectId || undefined
      })
      trendData.value = res
      await nextTick()
      renderTrendChart()
    } catch (e) {}
  }

  // 加载排行榜数据
  const loadRankingData = async () => {
    try {
      const { startTime, endTime } = getQueryTimeRange()
      const [projectRes, clusterRes] = await Promise.all([
        getBillingProjectTopApi({
          startTime,
          endTime,
          clusterUuid: filterForm.clusterUuid || undefined,
          topN: topN.value
        }),
        getBillingClusterTopApi({
          startTime,
          endTime,
          topN: topN.value
        })
      ])
      projectTopData.value = projectRes
      clusterTopData.value = clusterRes
    } catch (e) {}
  }

  // 渲染费用构成图表
  const renderCompositionChart = () => {
    if (!compositionChartRef.value) return

    if (!compositionChart) {
      compositionChart = echarts.init(compositionChartRef.value)
    }

    const data = compositionData.value.items.map((item, index) => ({
      name: item.costName,
      value: item.costAmount,
      itemStyle: { color: getCostColor(index) }
    }))

    const option: echarts.EChartsOption = {
      tooltip: {
        trigger: 'item',
        formatter: (params: any) => {
          return `${params.name}<br/>¥ ${formatAmount(params.value)} (${params.percent}%)`
        }
      },
      series: [
        {
          type: 'pie',
          radius: ['50%', '75%'],
          center: ['50%', '50%'],
          avoidLabelOverlap: false,
          itemStyle: {
            borderRadius: 6,
            borderColor: '#fff',
            borderWidth: 2
          },
          label: {
            show: false
          },
          emphasis: {
            label: {
              show: true,
              fontSize: 14,
              fontWeight: 'bold',
              formatter: (params: any) => `¥ ${formatAmount(params.value)}`
            },
            itemStyle: {
              shadowBlur: 10,
              shadowOffsetX: 0,
              shadowColor: 'rgba(0, 0, 0, 0.2)'
            }
          },
          data
        }
      ]
    }

    compositionChart.setOption(option, true)
  }

  // 渲染费用趋势图表
  const renderTrendChart = () => {
    if (!trendChartRef.value) return

    if (!trendChart) {
      trendChart = echarts.init(trendChartRef.value)
    }

    const months = trendData.value.items.map((item) => item.month)
    const totalCosts = trendData.value.items.map((item) => item.totalCost)
    const resourceCosts = trendData.value.items.map((item) => item.resourceCost)

    const option: echarts.EChartsOption = {
      tooltip: {
        trigger: 'axis',
        formatter: (params: any) => {
          let result = params[0].axisValue + '<br/>'
          params.forEach((item: any) => {
            result += `${item.marker} ${item.seriesName}: ¥ ${formatAmount(item.value)}<br/>`
          })
          return result
        }
      },
      legend: {
        data: ['总费用', '资源费用'],
        right: 20,
        top: 0
      },
      grid: {
        left: 60,
        right: 20,
        top: 40,
        bottom: 30
      },
      xAxis: {
        type: 'category',
        data: months,
        axisLine: { lineStyle: { color: '#e5e7eb' } },
        axisLabel: { color: '#6b7280' }
      },
      yAxis: {
        type: 'value',
        axisLine: { show: false },
        splitLine: { lineStyle: { color: '#f3f4f6' } },
        axisLabel: {
          color: '#6b7280',
          formatter: (value: number) =>
            `¥${value >= 1000 ? (value / 1000).toFixed(0) + 'k' : value}`
        }
      },
      series: [
        {
          name: '总费用',
          type: 'line',
          data: totalCosts,
          smooth: true,
          symbol: 'circle',
          symbolSize: 8,
          lineStyle: { width: 3, color: '#3b82f6' },
          itemStyle: { color: '#3b82f6' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(59, 130, 246, 0.2)' },
              { offset: 1, color: 'rgba(59, 130, 246, 0)' }
            ])
          }
        },
        {
          name: '资源费用',
          type: 'line',
          data: resourceCosts,
          smooth: true,
          symbol: 'circle',
          symbolSize: 6,
          lineStyle: { width: 2, type: 'dashed', color: '#10b981' },
          itemStyle: { color: '#10b981' }
        }
      ]
    }

    trendChart.setOption(option, true)
  }

  // 处理查询
  const handleQuery = () => {
    refreshAllData()
  }

  // 刷新所有数据
  const refreshAllData = async () => {
    await Promise.all([loadStatsData(), loadCompositionData(), loadTrendData(), loadRankingData()])
  }

  // 窗口大小变化时重绘图表
  const handleResize = () => {
    compositionChart?.resize()
    trendChart?.resize()
  }

  onMounted(async () => {
    // 加载下拉选项
    await Promise.all([loadClusterList(), loadProjectList()])
    // 加载数据
    await refreshAllData()
    // 监听窗口大小变化
    window.addEventListener('resize', handleResize)
  })

  onUnmounted(() => {
    window.removeEventListener('resize', handleResize)
    compositionChart?.dispose()
    trendChart?.dispose()
  })
</script>

<style scoped>
  .billing-dashboard {
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
    flex-wrap: wrap;
  }

  /* 统计卡片 */
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 20px;
    margin-bottom: 24px;
  }

  .stat-card {
    background: white;
    border-radius: 16px;
    padding: 20px 24px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
    border: 1px solid #f3f4f6;
  }

  .stat-card-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 16px;
  }

  .stat-card-icon {
    width: 44px;
    height: 44px;
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 22px;
  }

  .stat-icon-blue {
    background: linear-gradient(135deg, #dbeafe, #bfdbfe);
    color: #3b82f6;
  }

  .stat-icon-green {
    background: linear-gradient(135deg, #d1fae5, #a7f3d0);
    color: #10b981;
  }

  .stat-icon-orange {
    background: linear-gradient(135deg, #fef3c7, #fde68a);
    color: #f59e0b;
  }

  .stat-icon-purple {
    background: linear-gradient(135deg, #ede9fe, #ddd6fe);
    color: #8b5cf6;
  }

  .stat-card-mom {
    padding: 4px 10px;
    border-radius: 20px;
    font-size: 12px;
    font-weight: 600;
  }

  .mom-up {
    background: #fef2f2;
    color: #ef4444;
  }

  .mom-down {
    background: #ecfdf5;
    color: #10b981;
  }

  .mom-flat {
    background: #f3f4f6;
    color: #6b7280;
  }

  .stat-card-label {
    font-size: 13px;
    color: #6b7280;
    margin-bottom: 8px;
  }

  .stat-card-value {
    font-size: 28px;
    font-weight: 700;
    color: #1f2937;
    line-height: 1.2;
  }

  .stat-card-sub {
    font-size: 12px;
    color: #9ca3af;
    margin-top: 8px;
  }

  /* 图表卡片 */
  .chart-card {
    padding: 24px;
    height: 420px;
    display: flex;
    flex-direction: column;
  }

  .card-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 16px;
    flex-shrink: 0;
  }

  .header-icon {
    font-size: 18px;
  }

  .header-title {
    font-size: 16px;
    font-weight: 600;
    color: #374151;
    flex: 1;
  }

  .chart-container {
    flex: 1;
    min-height: 0;
  }

  .composition-legend {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px;
    margin-top: 16px;
    padding-top: 16px;
    border-top: 1px solid #f3f4f6;
  }

  .legend-item {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
  }

  .legend-dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .legend-name {
    color: #6b7280;
    flex: 1;
  }

  .legend-percent {
    color: #374151;
    font-weight: 500;
  }

  /* 排行榜卡片 */
  .ranking-card {
    padding: 24px;
    min-height: 420px;
    max-height: 420px;
    display: flex;
    flex-direction: column;
  }

  .ranking-list {
    flex: 1;
    overflow-y: auto;
    min-height: 0;
  }

  .ranking-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 0;
    border-bottom: 1px solid #f9fafb;
  }

  .ranking-item:last-child {
    border-bottom: none;
  }

  .ranking-rank {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    font-weight: 600;
    flex-shrink: 0;
  }

  .rank-gold {
    background: linear-gradient(135deg, #fbbf24, #f59e0b);
    color: white;
  }

  .rank-silver {
    background: linear-gradient(135deg, #9ca3af, #6b7280);
    color: white;
  }

  .rank-bronze {
    background: linear-gradient(135deg, #d97706, #b45309);
    color: white;
  }

  .rank-default {
    background: #f3f4f6;
    color: #6b7280;
  }

  .ranking-info {
    flex: 1;
    min-width: 0;
  }

  .ranking-name {
    font-size: 14px;
    font-weight: 500;
    color: #374151;
    margin-bottom: 6px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .ranking-tag {
    display: inline-block;
    padding: 2px 8px;
    background: #f3f4f6;
    color: #6b7280;
    border-radius: 10px;
    font-size: 11px;
    font-weight: normal;
    margin-left: 8px;
  }

  .ranking-bar-wrapper {
    height: 6px;
    background: #f3f4f6;
    border-radius: 3px;
    overflow: hidden;
  }

  .ranking-bar {
    height: 100%;
    border-radius: 3px;
    transition: width 0.3s ease;
  }

  .ranking-bar-purple {
    background: linear-gradient(90deg, #a78bfa, #8b5cf6);
  }

  .ranking-bar-cyan {
    background: linear-gradient(90deg, #22d3ee, #06b6d4);
  }

  .ranking-value {
    font-size: 14px;
    font-weight: 600;
    color: #1f2937;
    white-space: nowrap;
  }

  .ranking-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 280px;
    color: #9ca3af;
    font-size: 14px;
    gap: 12px;
  }

  .empty-icon {
    font-size: 48px;
    color: #d1d5db;
  }

  /* 通用颜色 */
  .text-blue-500 {
    color: #3b82f6;
  }

  .text-green-500 {
    color: #10b981;
  }

  .text-purple-500 {
    color: #8b5cf6;
  }

  .text-cyan-500 {
    color: #06b6d4;
  }

  .mb-5 {
    margin-bottom: 20px;
  }

  .mr-1 {
    margin-right: 4px;
  }
</style>
