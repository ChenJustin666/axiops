<template>
  <ElCard class="backend-health-card" shadow="never">
    <template #header>
      <div class="card-header">
        <span class="card-title">后端健康状态</span>
      </div>
    </template>
    <div v-loading="loading" class="card-content">
      <ElEmpty v-if="!backendData" description="暂无数据" :image-size="100" />
      <div v-else class="backend-detail">
        <!-- 上游延迟 -->
        <div class="section">
          <h4 class="section-title">上游整体延迟</h4>
          <div class="upstream-latency">
            <div class="latency-item">
              <span class="latency-label">P50</span>
              <span class="latency-value">{{ backendData.upstreamLatency }} ms</span>
            </div>
          </div>
        </div>

        <!-- Service Endpoints -->
        <div
          v-if="backendData.endpointsByService && backendData.endpointsByService.length > 0"
          class="section"
        >
          <h4 class="section-title">Service Endpoints</h4>
          <div class="service-list">
            <div
              v-for="item in backendData.endpointsByService"
              :key="`${item.namespace}-${item.serviceName}`"
              class="service-item"
            >
              <div class="service-info">
                <span class="service-name">{{ item.serviceName }}</span>
                <span class="service-namespace">{{ item.namespace }}</span>
              </div>
              <div class="service-endpoints">
                <span class="endpoint-metric">
                  <span class="label">可用端点:</span>
                  <span class="value">{{ item.availableEndpoints }}</span>
                </span>
                <ElTag :type="item.hasEndpoints ? 'success' : 'danger'" size="small">
                  {{ item.hasEndpoints ? '健康' : '异常' }}
                </ElTag>
              </div>
            </div>
          </div>
        </div>

        <!-- 后端健康详情 -->
        <div
          v-if="backendData.backendHealth && backendData.backendHealth.length > 0"
          class="section"
        >
          <h4 class="section-title">后端健康详情</h4>
          <div class="backend-list">
            <div
              v-for="item in backendData.backendHealth"
              :key="item.upstream"
              class="backend-item"
            >
              <div class="backend-info">
                <span class="backend-upstream">{{ item.upstream }}</span>
              </div>
              <div class="backend-metrics">
                <span class="metric">
                  <span class="label">响应延迟:</span>
                  <span class="value">{{ item.responseLatency }} ms</span>
                </span>
                <span class="metric">
                  <span class="label">成功率:</span>
                  <span class="value" :class="getSuccessRateClass(item.successRate)">
                    {{ (item.successRate * 100).toFixed(2) }}%
                  </span>
                </span>
                <span class="metric">
                  <span class="label">活跃连接:</span>
                  <span class="value">{{ item.activeConnections }}</span>
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- 空状态提示 -->
        <div
          v-if="!backendData.endpointsByService?.length && !backendData.backendHealth?.length"
          class="empty-backend"
        >
          <ElEmpty description="暂无后端数据" :image-size="100" />
        </div>
      </div>
    </div>
  </ElCard>
</template>

<script setup lang="ts">
  import type { IngressBackendMetrics } from '@/api'

  interface Props {
    backendData?: IngressBackendMetrics
    loading?: boolean
  }

  defineProps<Props>()

  const getSuccessRateClass = (successRate: number): string => {
    if (successRate >= 0.95) return 'success'
    if (successRate >= 0.8) return 'warning'
    return 'danger'
  }
</script>

<style lang="scss" scoped>
  .backend-health-card {
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
      min-height: 300px;

      .backend-detail {
        .section {
          margin-bottom: 24px;

          &:last-child {
            margin-bottom: 0;
          }

          .section-title {
            font-size: 14px;
            font-weight: 600;
            color: #606266;
            margin-bottom: 12px;
            padding-bottom: 8px;
            border-bottom: 1px solid #ebeef5;
          }

          .upstream-latency {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 16px;

            .latency-item {
              display: flex;
              flex-direction: column;
              gap: 8px;
              padding: 16px;
              background: #f5f7fa;
              border-radius: 8px;
              transition: all 0.3s;

              &:hover {
                background: #ecf5ff;
              }

              .latency-label {
                font-size: 13px;
                color: #909399;
              }

              .latency-value {
                font-size: 24px;
                font-weight: 600;
                color: #303133;
              }
            }
          }

          .service-list,
          .backend-list {
            display: flex;
            flex-direction: column;
            gap: 8px;

            .service-item,
            .backend-item {
              display: flex;
              justify-content: space-between;
              align-items: center;
              padding: 12px;
              background: #f5f7fa;
              border-radius: 6px;
              transition: all 0.3s;

              &:hover {
                background: #ecf5ff;
              }

              .service-info,
              .backend-info {
                display: flex;
                align-items: center;
                gap: 8px;
                flex: 1;

                .service-name,
                .backend-upstream {
                  font-weight: 500;
                  color: #303133;
                }

                .service-namespace {
                  font-size: 12px;
                  color: #909399;
                  padding: 2px 6px;
                  background: #e4e7ed;
                  border-radius: 4px;
                }
              }

              .service-endpoints,
              .backend-metrics {
                display: flex;
                gap: 16px;
                align-items: center;

                .endpoint-metric,
                .metric {
                  display: flex;
                  gap: 4px;
                  font-size: 13px;

                  .label {
                    color: #909399;
                  }

                  .value {
                    font-weight: 500;
                    color: #303133;

                    &.success {
                      color: #67c23a;
                    }

                    &.warning {
                      color: #e6a23c;
                    }

                    &.danger {
                      color: #f56c6c;
                    }
                  }
                }
              }
            }
          }

          .empty-backend {
            padding: 40px 0;
          }
        }
      }
    }
  }
</style>
