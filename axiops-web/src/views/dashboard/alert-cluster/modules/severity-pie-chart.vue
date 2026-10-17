<template>
  <div ref="chartRef" class="w-full h-full"></div>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, onUnmounted } from 'vue'
  import * as echarts from 'echarts'
  import type { GetAlertSeverityStatsResponse } from '@/api/manager/alert-dashboard'

  interface Props {
    data: GetAlertSeverityStatsResponse
  }

  const props = defineProps<Props>()
  const chartRef = ref<HTMLElement>()
  let chartInstance: echarts.ECharts | null = null

  // 级别配置
  const severityConfig: Record<string, { name: string; color: string }> = {
    critical: { name: 'Critical', color: '#ef4444' },
    warning: { name: 'Warning', color: '#f59e0b' },
    info: { name: 'Info', color: '#3b82f6' },
    none: { name: 'None', color: '#9ca3af' }
  }

  // 初始化图表
  const initChart = () => {
    if (!chartRef.value) return
    chartInstance = echarts.init(chartRef.value)
    updateChart()
    window.addEventListener('resize', handleResize)
  }

  // 更新图表
  const updateChart = () => {
    if (!chartInstance) return

    const chartData = props.data.items.map((item) => ({
      name: severityConfig[item.severity]?.name || item.severity,
      value: item.totalCount,
      itemStyle: { color: severityConfig[item.severity]?.color || '#9ca3af' }
    }))

    const option: echarts.EChartsOption = {
      tooltip: {
        trigger: 'item',
        backgroundColor: 'rgba(255, 255, 255, 0.95)',
        borderColor: '#e5e7eb',
        borderWidth: 1,
        padding: [10, 14],
        textStyle: { color: '#374151', fontSize: 13 },
        formatter: '{b}: {c} ({d}%)'
      },
      legend: {
        bottom: 12,
        itemWidth: 10,
        itemHeight: 10,
        itemGap: 16,
        textStyle: { color: '#6b7280', fontSize: 12 }
      },
      series: [
        {
          type: 'pie',
          radius: ['50%', '70%'],
          center: ['50%', '42%'],
          avoidLabelOverlap: false,
          itemStyle: {
            borderRadius: 4,
            borderColor: '#fff',
            borderWidth: 2
          },
          label: {
            show: true,
            position: 'center',
            formatter: () => `{total|${props.data.totalCount}}\n{label|总计}`,
            rich: {
              total: {
                fontSize: 26,
                fontWeight: 'bold',
                color: '#1f2937'
              },
              label: {
                fontSize: 13,
                color: '#9ca3af',
                padding: [6, 0, 0, 0]
              }
            }
          },
          labelLine: { show: false },
          emphasis: { scale: true, scaleSize: 4 },
          data: chartData
        }
      ]
    }

    chartInstance.setOption(option, true)
  }

  // 窗口大小变化
  const handleResize = () => chartInstance?.resize()

  watch(() => props.data, updateChart, { deep: true })

  onMounted(() => initChart())

  onUnmounted(() => {
    window.removeEventListener('resize', handleResize)
    chartInstance?.dispose()
  })
</script>
