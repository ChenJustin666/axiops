<template>
  <ElCard class="traffic-metrics-card" shadow="never">
    <template #header>
      <div class="card-header">
        <span class="card-title">流量详细分析</span>
      </div>
    </template>
    <div v-loading="loading" class="card-content">
      <ElEmpty v-if="!trafficData" description="暂无数据" :image-size="100" />
      <div v-else class="traffic-detail">
        <!-- 按Host统计 -->
        <div class="section">
          <h4 class="section-title">按 Host 统计</h4>
          <div class="host-list">
            <div v-for="item in trafficData.byHost" :key="item.name" class="host-item">
              <div class="host-info">
                <span class="host-name">{{ item.name }}</span>
                <span v-if="item.namespace" class="host-namespace">{{ item.namespace }}</span>
              </div>
              <div class="host-metrics">
                <span class="metric">
                  <span class="label">QPS:</span>
                  <span class="value">{{ item.requestsPerSecond.toFixed(2) }}</span>
                </span>
                <span class="metric">
                  <span class="label">流量:</span>
                  <span class="value">{{ formatBytes(item.bytesPerSecond) }}/s</span>
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- 按Path统计 -->
        <div class="section">
          <h4 class="section-title">按 Path 统计</h4>
          <div class="path-list">
            <div v-for="item in trafficData.byPath" :key="item.path" class="path-item">
              <div class="path-info">
                <span class="path-path">{{ item.path }}</span>
                <span v-if="item.namespace" class="path-namespace">{{ item.namespace }}</span>
              </div>
              <div class="path-metrics">
                <span class="metric">
                  <span class="label">QPS:</span>
                  <span class="value">{{ item.requestsPerSecond.toFixed(2) }}</span>
                </span>
                <span class="metric">
                  <span class="label">流量:</span>
                  <span class="value">{{ formatBytes(item.bytesPerSecond) }}/s</span>
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- 按Service统计 -->
        <div class="section">
          <h4 class="section-title">按 Service 统计</h4>
          <div class="service-list">
            <div v-for="item in trafficData.byService" :key="item.name" class="service-item">
              <div class="service-info">
                <span class="service-name">{{ item.name }}</span>
                <span v-if="item.namespace" class="service-namespace">{{ item.namespace }}</span>
              </div>
              <div class="service-metrics">
                <span class="metric">
                  <span class="label">QPS:</span>
                  <span class="value">{{ item.requestsPerSecond.toFixed(2) }}</span>
                </span>
                <span class="metric">
                  <span class="label">流量:</span>
                  <span class="value">{{ formatBytes(item.bytesPerSecond) }}/s</span>
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- 按方法统计 -->
        <div class="section">
          <h4 class="section-title">按 HTTP 方法统计</h4>
          <div class="method-list">
            <div v-for="item in trafficData.byMethod" :key="item.method" class="method-item">
              <ElTag :type="getMethodType(item.method)" size="large">
                {{ item.method }}
              </ElTag>
              <span class="method-qps">{{ item.requestsPerSecond.toFixed(2) }} QPS</span>
            </div>
          </div>
        </div>

        <!-- 汇总信息 -->
        <div class="section summary-section">
          <h4 class="section-title">流量汇总</h4>
          <div class="summary-grid">
            <div class="summary-item">
              <span class="summary-label">总请求数</span>
              <span class="summary-value">{{
                formatNumber(trafficData.summary.totalRequests)
              }}</span>
            </div>
            <div class="summary-item">
              <span class="summary-label">平均 QPS</span>
              <span class="summary-value">{{
                trafficData.summary.avgRequestsPerSec.toFixed(2)
              }}</span>
            </div>
            <div class="summary-item">
              <span class="summary-label">峰值 QPS</span>
              <span class="summary-value">{{
                trafficData.summary.maxRequestsPerSec.toFixed(2)
              }}</span>
            </div>
            <div class="summary-item">
              <span class="summary-label">平均请求大小</span>
              <span class="summary-value">{{
                formatBytes(trafficData.summary.avgRequestSize)
              }}</span>
            </div>
            <div class="summary-item">
              <span class="summary-label">平均响应大小</span>
              <span class="summary-value">{{
                formatBytes(trafficData.summary.avgResponseSize)
              }}</span>
            </div>
            <div class="summary-item">
              <span class="summary-label">总入站流量</span>
              <span class="summary-value">{{
                formatBytes(trafficData.summary.totalIngressBytes)
              }}</span>
            </div>
            <div class="summary-item">
              <span class="summary-label">总出站流量</span>
              <span class="summary-value">{{
                formatBytes(trafficData.summary.totalEgressBytes)
              }}</span>
            </div>
          </div>
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
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const formatNumber = (num: number): string => {
    if (num >= 1000000) {
      return (num / 1000000).toFixed(2) + 'M'
    }
    if (num >= 1000) {
      return (num / 1000).toFixed(2) + 'K'
    }
    return num.toString()
  }

  const getMethodType = (method: string) => {
    switch (method.toUpperCase()) {
      case 'GET':
        return 'success'
      case 'POST':
        return 'primary'
      case 'PUT':
        return 'warning'
      case 'DELETE':
        return 'danger'
      default:
        return 'info'
    }
  }
</script>

<style lang="scss" scoped>
  .traffic-metrics-card {
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

      .traffic-detail {
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

          .host-list,
          .path-list,
          .service-list {
            display: flex;
            flex-direction: column;
            gap: 8px;

            .host-item,
            .path-item,
            .service-item {
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

              .host-info,
              .path-info,
              .service-info {
                display: flex;
                align-items: center;
                gap: 8px;
                flex: 1;

                .host-name,
                .path-path,
                .service-name {
                  font-weight: 500;
                  color: #303133;
                }

                .host-namespace,
                .path-namespace,
                .service-namespace {
                  font-size: 12px;
                  color: #909399;
                  padding: 2px 6px;
                  background: #e4e7ed;
                  border-radius: 4px;
                }
              }

              .host-metrics,
              .path-metrics,
              .service-metrics {
                display: flex;
                gap: 16px;

                .metric {
                  display: flex;
                  gap: 4px;
                  font-size: 13px;

                  .label {
                    color: #909399;
                  }

                  .value {
                    color: #303133;
                    font-weight: 500;
                  }
                }
              }
            }
          }

          .method-list {
            display: flex;
            flex-wrap: wrap;
            gap: 12px;

            .method-item {
              display: flex;
              align-items: center;
              gap: 8px;
              padding: 8px 12px;
              background: #f5f7fa;
              border-radius: 6px;

              .method-qps {
                font-size: 13px;
                font-weight: 500;
                color: #303133;
              }
            }
          }

          &.summary-section {
            .summary-grid {
              display: grid;
              grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
              gap: 16px;

              .summary-item {
                display: flex;
                flex-direction: column;
                gap: 8px;
                padding: 16px;
                background: linear-gradient(135deg, #f5f7fa 0%, #e8ecf1 100%);
                border-radius: 8px;

                .summary-label {
                  font-size: 13px;
                  color: #909399;
                }

                .summary-value {
                  font-size: 18px;
                  font-weight: 600;
                  color: #303133;
                }
              }
            }
          }
        }
      }
    }
  }
</style>
