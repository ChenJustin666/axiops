<template>
  <ElCard class="performance-metrics-card" shadow="never">
    <template #header>
      <div class="card-header">
        <span class="card-title">性能分析</span>
      </div>
    </template>
    <div v-loading="loading" class="card-content">
      <ElEmpty v-if="!performanceData" description="暂无数据" :image-size="100" />
      <div v-else class="performance-detail">
        <!-- 整体延迟指标 -->
        <div class="section">
          <h4 class="section-title">整体延迟指标</h4>
          <div class="latency-grid">
            <div class="latency-item">
              <span class="latency-label">P50 延迟</span>
              <span class="latency-value">{{ performanceData.overall.p50 }} ms</span>
            </div>
            <div class="latency-item">
              <span class="latency-label">P95 延迟</span>
              <span class="latency-value">{{ performanceData.overall.p95 }} ms</span>
            </div>
            <div class="latency-item">
              <span class="latency-label">P99 延迟</span>
              <span class="latency-value">{{ performanceData.overall.p99 }} ms</span>
            </div>
            <div class="latency-item">
              <span class="latency-label">平均延迟</span>
              <span class="latency-value">{{ performanceData.overall.avg }} ms</span>
            </div>
            <div class="latency-item">
              <span class="latency-label">最大延迟</span>
              <span class="latency-value">{{ performanceData.overall.max }} ms</span>
            </div>
            <div
              class="latency-item latency-grade"
              :class="getLatencyGradeClass(performanceData.overall.p95)"
            >
              <span class="latency-label">性能等级</span>
              <span class="latency-value">{{ getLatencyGrade(performanceData.overall.p95) }}</span>
            </div>
          </div>
        </div>

        <!-- 按Host延迟统计 -->
        <div v-if="performanceData.byHost && performanceData.byHost.length > 0" class="section">
          <h4 class="section-title">按 Host 延迟统计</h4>
          <div class="host-list">
            <div v-for="item in performanceData.byHost" :key="item.name" class="host-item">
              <div class="host-info">
                <span class="host-name">{{ item.name }}</span>
                <span v-if="item.namespace" class="host-namespace">{{ item.namespace }}</span>
              </div>
              <div class="host-latency">
                <span class="latency-metric">
                  <span class="label">P95:</span>
                  <span class="value">{{ item.latency.p95 }} ms</span>
                </span>
                <span class="latency-metric">
                  <span class="label">P99:</span>
                  <span class="value">{{ item.latency.p99 }} ms</span>
                </span>
                <span class="latency-metric">
                  <span class="label">Avg:</span>
                  <span class="value">{{ item.latency.avg }} ms</span>
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- 按Path延迟统计 -->
        <div v-if="performanceData.byPath && performanceData.byPath.length > 0" class="section">
          <h4 class="section-title">按 Path 延迟统计</h4>
          <div class="path-list">
            <div v-for="item in performanceData.byPath" :key="item.path" class="path-item">
              <div class="path-info">
                <span class="path-path">{{ item.path }}</span>
                <span v-if="item.namespace" class="path-namespace">{{ item.namespace }}</span>
              </div>
              <div class="path-latency">
                <span class="latency-metric">
                  <span class="label">P95:</span>
                  <span class="value">{{ item.latency.p95 }} ms</span>
                </span>
                <span class="latency-metric">
                  <span class="label">P99:</span>
                  <span class="value">{{ item.latency.p99 }} ms</span>
                </span>
                <span class="latency-metric">
                  <span class="label">Avg:</span>
                  <span class="value">{{ item.latency.avg }} ms</span>
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- 上游延迟 -->
        <div class="section">
          <h4 class="section-title">上游延迟</h4>
          <div class="upstream-latency">
            <div class="latency-item large">
              <span class="latency-label">P50</span>
              <span class="latency-value">{{ performanceData.upstreamLatency.p50 }} ms</span>
            </div>
            <div class="latency-item large">
              <span class="latency-label">P95</span>
              <span class="latency-value">{{ performanceData.upstreamLatency.p95 }} ms</span>
            </div>
            <div class="latency-item large">
              <span class="latency-label">P99</span>
              <span class="latency-value">{{ performanceData.upstreamLatency.p99 }} ms</span>
            </div>
            <div class="latency-item large">
              <span class="latency-label">平均</span>
              <span class="latency-value">{{ performanceData.upstreamLatency.avg }} ms</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </ElCard>
</template>

<script setup lang="ts">
  import type { IngressPerformanceMetrics } from '@/api'

  interface Props {
    performanceData?: IngressPerformanceMetrics
    loading?: boolean
  }

  defineProps<Props>()

  const getLatencyGrade = (latencyMs: number): string => {
    if (latencyMs < 100) return '优秀'
    if (latencyMs < 500) return '良好'
    if (latencyMs < 1000) return '一般'
    return '较差'
  }

  const getLatencyGradeClass = (latencyMs: number): string => {
    if (latencyMs < 100) return 'excellent'
    if (latencyMs < 500) return 'good'
    if (latencyMs < 1000) return 'fair'
    return 'poor'
  }
</script>

<style lang="scss" scoped>
  .performance-metrics-card {
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

      .performance-detail {
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

          .latency-grid,
          .upstream-latency {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 12px;

            .latency-item {
              display: flex;
              flex-direction: column;
              gap: 8px;
              padding: 14px;
              background: #f5f7fa;
              border-radius: 8px;
              transition: all 0.3s;

              &:hover {
                background: #ecf5ff;
              }

              &.large {
                padding: 18px;
                background: linear-gradient(135deg, #f5f7fa 0%, #e8ecf1 100%);
              }

              &.latency-grade {
                background: linear-gradient(135deg, #e7f7ff 0%, #d4ecff 100%);

                &.excellent {
                  background: linear-gradient(135deg, #e7f7ff 0%, #d4ecff 100%);
                  border: 1px solid #95de64;
                }

                &.good {
                  background: linear-gradient(135deg, #fff7e6 0%, #ffd591 100%);
                  border: 1px solid #ffc069;
                }

                &.fair {
                  background: linear-gradient(135deg, #fff1f0 0%, #ffccc7 100%);
                  border: 1px solid #ffa39e;
                }

                &.poor {
                  background: linear-gradient(135deg, #fff1f0 0%, #ffccc7 100%);
                  border: 1px solid #f5222d;
                }
              }

              .latency-label {
                font-size: 13px;
                color: #909399;
              }

              .latency-value {
                font-size: 18px;
                font-weight: 600;
                color: #303133;
              }
            }
          }

          .host-list,
          .path-list {
            display: flex;
            flex-direction: column;
            gap: 8px;

            .host-item,
            .path-item {
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
              .path-info {
                display: flex;
                align-items: center;
                gap: 8px;
                flex: 1;

                .host-name,
                .path-path {
                  font-weight: 500;
                  color: #303133;
                }

                .host-namespace,
                .path-namespace {
                  font-size: 12px;
                  color: #909399;
                  padding: 2px 6px;
                  background: #e4e7ed;
                  border-radius: 4px;
                }
              }

              .host-latency,
              .path-latency {
                display: flex;
                gap: 16px;

                .latency-metric {
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
        }
      }
    }
  }
</style>
