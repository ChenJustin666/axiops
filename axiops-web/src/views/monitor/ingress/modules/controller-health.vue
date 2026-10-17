<template>
  <ElCard class="controller-health-card" shadow="never">
    <template #header>
      <div class="card-header">
        <span class="card-title">Controller 健康状态</span>
      </div>
    </template>
    <div v-loading="loading" class="card-content">
      <ElEmpty v-if="!controllerData" description="暂无数据" :image-size="100" />
      <div v-else class="health-metrics">
        <div class="metric-item">
          <span class="metric-label">运行 Pods</span>
          <span class="metric-value"
            >{{ controllerData.runningPods }} / {{ controllerData.totalPods }}</span
          >
        </div>
        <div class="metric-item">
          <span class="metric-label">就绪 Pods</span>
          <span class="metric-value">{{ controllerData.readyPods }}</span>
        </div>
        <div class="metric-item">
          <span class="metric-label">CPU 使用率</span>
          <span class="metric-value">{{ controllerData.cpuUsage.toFixed(2) }}%</span>
        </div>
        <div class="metric-item">
          <span class="metric-label">内存使用率</span>
          <span class="metric-value">{{ controllerData.memoryUsage.toFixed(2) }}%</span>
        </div>
        <div class="metric-item">
          <span class="metric-label">活跃连接</span>
          <span class="metric-value">{{ controllerData.connections.active }}</span>
        </div>
        <div class="metric-item">
          <span class="metric-label">重载成功率</span>
          <span class="metric-value">{{ controllerData.lastReloadSuccess ? '成功' : '失败' }}</span>
        </div>
      </div>
    </div>
  </ElCard>
</template>

<script setup lang="ts">
  import type { IngressControllerHealth } from '@/api'

  interface Props {
    controllerData?: IngressControllerHealth
    loading?: boolean
  }

  defineProps<Props>()
</script>

<style lang="scss" scoped>
  .controller-health-card {
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

      .health-metrics {
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
