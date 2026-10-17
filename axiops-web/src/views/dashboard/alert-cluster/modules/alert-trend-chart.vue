<template>
  <div ref="chartRef" class="w-full h-full"></div>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, onUnmounted } from 'vue'
  import * as echarts from 'echarts'
  import { getAlertTrendApi, type TrendDataPoint } from '@/api/manager/alert-dashboard'

  interface Props {
    filter: {
      clusterUuid?: string
      projectId?: number
      workspaceId?: number
    }
    days?: number
  }

  const props = withDefaults(defineProps<Props>(), {
    days: 7
  })

  const chartRef = ref<HTMLElement>()
  let chartInstance: echarts.ECharts | null = null
  const chartData = ref<TrendDataPoint[]>([])

  // 初始化图表
  const initChart = () => {
    if (!chartRef.value) return
    chartInstance = echarts.init(chartRef.value)
    window.addEventListener('resize', handleResize)
  }

  // 更新图表
  const updateChart = () => {
    if (!chartInstance) return

    const option: echarts.EChartsOption = {
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(255, 255, 255, 0.95)',
        borderColor: '#e5e7eb',
        borderWidth: 1,
        padding: [12, 16],
        textStyle: {
          color: '#374151',
          fontSize: 13
        },
        axisPointer: {
          type: 'line',
          lineStyle: {
            color: '#e5e7eb',
            type: 'dashed'
          }
        }
      },
      legend: {
        data: ['新增告警', '已恢复', '触发中'],
        bottom: 0,
        itemWidth: 20,
        itemHeight: 8,
        itemGap: 24,
        textStyle: {
          color: '#6b7280',
          fontSize: 12
        }
      },
      grid: {
        left: 12,
        right: 20,
        bottom: 36,
        top: 12,
        containLabel: true
      },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: chartData.value.map((item) => item.date.slice(5)),
        axisLine: {
          lineStyle: { color: '#e5e7eb' }
        },
        axisTick: { show: false },
        axisLabel: {
          color: '#9ca3af',
          fontSize: 12,
          margin: 12
        }
      },
      yAxis: {
        type: 'value',
        splitLine: {
          lineStyle: { color: '#f3f4f6', type: 'dashed' }
        },
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: {
          color: '#9ca3af',
          fontSize: 12
        }
      },
      series: [
        {
          name: '新增告警',
          type: 'line',
          smooth: true,
          symbol: 'circle',
          symbolSize: 6,
          showSymbol: false,
          lineStyle: { width: 2, color: '#ef4444' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(239, 68, 68, 0.2)' },
              { offset: 1, color: 'rgba(239, 68, 68, 0.01)' }
            ])
          },
          itemStyle: { color: '#ef4444' },
          emphasis: { showSymbol: true, symbolSize: 8 },
          data: chartData.value.map((item) => item.newCount)
        },
        {
          name: '已恢复',
          type: 'line',
          smooth: true,
          symbol: 'circle',
          symbolSize: 6,
          showSymbol: false,
          lineStyle: { width: 2, color: '#10b981' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(16, 185, 129, 0.2)' },
              { offset: 1, color: 'rgba(16, 185, 129, 0.01)' }
            ])
          },
          itemStyle: { color: '#10b981' },
          emphasis: { showSymbol: true, symbolSize: 8 },
          data: chartData.value.map((item) => item.resolvedCount)
        },
        {
          name: '触发中',
          type: 'line',
          smooth: true,
          symbol: 'circle',
          symbolSize: 6,
          showSymbol: false,
          lineStyle: { width: 2, color: '#f59e0b' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(245, 158, 11, 0.2)' },
              { offset: 1, color: 'rgba(245, 158, 11, 0.01)' }
            ])
          },
          itemStyle: { color: '#f59e0b' },
          emphasis: { showSymbol: true, symbolSize: 8 },
          data: chartData.value.map((item) => item.firingCount)
        }
      ]
    }

    chartInstance.setOption(option, true)
  }

  // 加载数据
  const loadData = async () => {
    try {
      const res = await getAlertTrendApi({
        clusterUuid: props.filter.clusterUuid || undefined,
        projectId: props.filter.projectId || undefined,
        workspaceId: props.filter.workspaceId || undefined,
        days: props.days
      })
      chartData.value = res.dataPoints || []
      updateChart()
    } catch (error) {}
  }

  // 刷新
  const refresh = () => loadData()

  // 窗口大小变化
  const handleResize = () => chartInstance?.resize()

  watch(
    () => props.filter,
    () => loadData(),
    { deep: true }
  )
  watch(
    () => props.days,
    () => loadData()
  )

  onMounted(() => {
    initChart()
    loadData()
  })

  onUnmounted(() => {
    window.removeEventListener('resize', handleResize)
    chartInstance?.dispose()
  })

  defineExpose({ refresh })
</script>
