<template>
  <div class="trend-chart" :style="{ height }">
    <div v-if="!data || data.length === 0" class="empty">暂无数据</div>
    <div v-else class="chart-content">
      <!-- 简单的折线图 -->
      <svg class="line-chart" viewBox="0 0 100 60" preserveAspectRatio="none">
        <!-- 网格线 -->
        <line x1="0" y1="15" x2="100" y2="15" stroke="#f0f0f0" stroke-width="0.3" />
        <line x1="0" y1="30" x2="100" y2="30" stroke="#f0f0f0" stroke-width="0.3" />
        <line x1="0" y1="45" x2="100" y2="45" stroke="#f0f0f0" stroke-width="0.3" />

        <!-- QPS 线条 -->
        <path
          :d="generatePath(data, 'requestsPerSecond')"
          fill="none"
          stroke="#409eff"
          stroke-width="1"
          vector-effect="non-scaling-stroke"
        />

        <!-- 延迟线条 -->
        <path
          :d="generatePath(data, 'p95Latency')"
          fill="none"
          stroke="#e6a23c"
          stroke-width="1"
          vector-effect="non-scaling-stroke"
        />
      </svg>

      <!-- 图例 -->
      <div class="legend">
        <span class="legend-item">
          <i class="dot" style="background: #409eff"></i>
          <span>QPS</span>
        </span>
        <span class="legend-item">
          <i class="dot" style="background: #e6a23c"></i>
          <span>P95延迟</span>
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import type { HostMetricDataPoint } from '@/api'

  interface Props {
    data: HostMetricDataPoint[]
    height?: string
  }

  const props = withDefaults(defineProps<Props>(), {
    height: '240px'
  })

  const getMaxValue = (key: keyof HostMetricDataPoint) => {
    if (!props.data || props.data.length === 0) return 1
    return Math.max(...props.data.map((item) => Number(item[key] || 0)))
  }

  const generatePath = (data: HostMetricDataPoint[], key: keyof HostMetricDataPoint) => {
    if (!data || data.length === 0) return ''

    const max = getMaxValue(key)
    if (max === 0) return ''

    const points = data.map((item, index) => {
      const x = (index / (data.length - 1)) * 100
      // 正常坐标系：底部是0，顶部是最大值
      // SVG的Y轴是从上到下，所以需要用 (viewBox高度 - 实际值对应的高度)
      const normalizedValue = Number(item[key] || 0) / max // 0-1之间
      const y = 55 - normalizedValue * 50 // 底部55，顶部5
      return `${x},${y}`
    })

    return `M ${points.join(' L ')}`
  }
</script>

<style scoped>
  .trend-chart {
    width: 100%;
    background: #fff;
  }

  .empty {
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #909399;
    font-size: 13px;
  }

  .chart-content {
    height: 100%;
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 12px;
  }

  .x-labels {
    display: none;
  }

  .line-chart {
    flex: 1;
    width: 100%;
    min-height: 120px;
  }

  .legend {
    display: flex;
    justify-content: center;
    gap: 20px;
    padding: 4px 0;
  }

  .legend-item {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: #606266;
  }

  .dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
  }
</style>
