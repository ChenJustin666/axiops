<template>
  <ElCard class="traffic-overview-card" shadow="never">
    <template #header>
      <div class="card-header">
        <span class="card-title">流量概览</span>
      </div>
    </template>
    <div v-loading="loading" class="card-content">
      <ElEmpty v-if="!trafficData" description="暂无数据" :image-size="100" />
      <div v-else class="traffic-metrics">
        <div class="metric-item">
          <span class="metric-label">当前 QPS</span>
          <span class="metric-value">{{ trafficData.current.requestsPerSecond.toFixed(2) }}</span>
        </div>
        <div class="metric-item">
          <span class="metric-label">活跃连接</span>
          <span class="metric-value">{{ trafficData.current.activeConnections }}</span>
        </div>
        <div class="metric-item">
          <span class="metric-label">入站流量</span>
          <span class="metric-value"
            >{{ formatBytes(trafficData.current.ingressBytesPerSec) }}/s</span
          >
        </div>
        <div class="metric-item">
          <span class="metric-label">出站流量</span>
          <span class="metric-value"
            >{{ formatBytes(trafficData.current.egressBytesPerSec) }}/s</span
          >
        </div>
        <div class="metric-item">
          <span class="metric-label">总请求数</span>
          <span class="metric-value">{{ trafficData.summary.totalRequests }}</span>
        </div>
        <div class="metric-item">
          <span class="metric-label">平均 QPS</span>
          <span class="metric-value">{{ trafficData.summary.avgRequestsPerSec.toFixed(2) }}</span>
        </div>
      </div>
    </div>
  </ElCard>
</template>

<script setup lang="ts">
  import type { IngressTrafficMetrics } from '@/api'

  interface Props {
    trafficData?: IngressTrafficMetrics
    loading?: boolean
  }

  defineProps<Props>()

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }
</script>

<style lang="scss" scoped>
  .traffic-overview-card {
    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;

      .card-title {
        font-size: 16px;
        font-weight: 600;
        color: #303133;
      }
    }

    .card-content {
      min-height: 200px;

      .traffic-metrics {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
        gap: 16px;

        .metric-item {
          display: flex;
          flex-direction: column;
          gap: 8px;
          padding: 16px;
          background: #f5f7fa;
          border-radius: 8px;

          .metric-label {
            font-size: 13px;
            color: #909399;
          }

          .metric-value {
            font-size: 20px;
            font-weight: 600;
            color: #303133;
          }
        }
      }
    }
  }
</style>
