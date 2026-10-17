<template>
  <ElCard class="error-metrics-card" shadow="never">
    <template #header>
      <div class="card-header">
        <span class="card-title">错误分析</span>
      </div>
    </template>
    <div v-loading="loading" class="card-content">
      <ElEmpty v-if="!errorData" description="暂无数据" :image-size="100" />
      <div v-else class="error-detail">
        <!-- 整体错误率 -->
        <div class="section">
          <h4 class="section-title">整体错误率</h4>
          <div class="error-rate-grid">
            <div class="error-rate-item">
              <span class="error-rate-label">总错误率</span>
              <span
                class="error-rate-value"
                :class="getErrorRateClass(errorData.overall.totalErrorRate)"
              >
                {{ (errorData.overall.totalErrorRate * 100).toFixed(2) }}%
              </span>
            </div>
            <div class="error-rate-item">
              <span class="error-rate-label">4xx 错误率</span>
              <span
                class="error-rate-value"
                :class="getErrorRateClass(errorData.overall.error4xxRate)"
              >
                {{ (errorData.overall.error4xxRate * 100).toFixed(2) }}%
              </span>
            </div>
            <div class="error-rate-item">
              <span class="error-rate-label">5xx 错误率</span>
              <span
                class="error-rate-value"
                :class="getErrorRateClass(errorData.overall.error5xxRate)"
              >
                {{ (errorData.overall.error5xxRate * 100).toFixed(2) }}%
              </span>
            </div>
          </div>
        </div>

        <!-- 状态码分布 -->
        <div class="section">
          <h4 class="section-title">状态码分布</h4>
          <div class="status-code-grid">
            <!-- 2xx 状态码 -->
            <div v-if="hasStatusCodes(errorData.statusCodes.status2xx)" class="status-code-group">
              <h5 class="group-title success">2xx 成功</h5>
              <div class="status-code-list">
                <div
                  v-for="(count, code) in errorData.statusCodes.status2xx"
                  :key="code"
                  class="status-code-item success"
                >
                  <span class="code">{{ code }}</span>
                  <span class="count">{{ formatNumber(count) }}</span>
                </div>
              </div>
            </div>

            <!-- 3xx 状态码 -->
            <div v-if="hasStatusCodes(errorData.statusCodes.status3xx)" class="status-code-group">
              <h5 class="group-title info">3xx 重定向</h5>
              <div class="status-code-list">
                <div
                  v-for="(count, code) in errorData.statusCodes.status3xx"
                  :key="code"
                  class="status-code-item info"
                >
                  <span class="code">{{ code }}</span>
                  <span class="count">{{ formatNumber(count) }}</span>
                </div>
              </div>
            </div>

            <!-- 4xx 状态码 -->
            <div v-if="hasStatusCodes(errorData.statusCodes.status4xx)" class="status-code-group">
              <h5 class="group-title warning">4xx 客户端错误</h5>
              <div class="status-code-list">
                <div
                  v-for="(count, code) in errorData.statusCodes.status4xx"
                  :key="code"
                  class="status-code-item warning"
                >
                  <span class="code">{{ code }}</span>
                  <span class="count">{{ formatNumber(count) }}</span>
                </div>
              </div>
            </div>

            <!-- 5xx 状态码 -->
            <div v-if="hasStatusCodes(errorData.statusCodes.status5xx)" class="status-code-group">
              <h5 class="group-title danger">5xx 服务端错误</h5>
              <div class="status-code-list">
                <div
                  v-for="(count, code) in errorData.statusCodes.status5xx"
                  :key="code"
                  class="status-code-item danger"
                >
                  <span class="code">{{ code }}</span>
                  <span class="count">{{ formatNumber(count) }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 按Host错误统计 -->
        <div v-if="errorData.byHost && errorData.byHost.length > 0" class="section">
          <h4 class="section-title">按 Host 错误统计</h4>
          <div class="host-list">
            <div v-for="item in errorData.byHost" :key="item.name" class="host-item">
              <div class="host-info">
                <span class="host-name">{{ item.name }}</span>
                <span v-if="item.namespace" class="host-namespace">{{ item.namespace }}</span>
              </div>
              <div class="host-errors">
                <span class="error-metric">
                  <span class="label">总错误率:</span>
                  <span class="value" :class="getErrorRateClass(item.errorRate.totalErrorRate)">
                    {{ (item.errorRate.totalErrorRate * 100).toFixed(2) }}%
                  </span>
                </span>
                <span class="error-metric">
                  <span class="label">4xx:</span>
                  <span class="value" :class="getErrorRateClass(item.errorRate.error4xxRate)">
                    {{ (item.errorRate.error4xxRate * 100).toFixed(2) }}%
                  </span>
                </span>
                <span class="error-metric">
                  <span class="label">5xx:</span>
                  <span class="value" :class="getErrorRateClass(item.errorRate.error5xxRate)">
                    {{ (item.errorRate.error5xxRate * 100).toFixed(2) }}%
                  </span>
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- 按Path错误统计 -->
        <div v-if="errorData.byPath && errorData.byPath.length > 0" class="section">
          <h4 class="section-title">按 Path 错误统计</h4>
          <div class="path-list">
            <div v-for="item in errorData.byPath" :key="item.path" class="path-item">
              <div class="path-info">
                <span class="path-path">{{ item.path }}</span>
                <span v-if="item.namespace" class="path-namespace">{{ item.namespace }}</span>
              </div>
              <div class="path-errors">
                <span class="error-metric">
                  <span class="label">总错误率:</span>
                  <span class="value" :class="getErrorRateClass(item.errorRate.totalErrorRate)">
                    {{ (item.errorRate.totalErrorRate * 100).toFixed(2) }}%
                  </span>
                </span>
                <span class="error-metric">
                  <span class="label">4xx:</span>
                  <span class="value" :class="getErrorRateClass(item.errorRate.error4xxRate)">
                    {{ (item.errorRate.error4xxRate * 100).toFixed(2) }}%
                  </span>
                </span>
                <span class="error-metric">
                  <span class="label">5xx:</span>
                  <span class="value" :class="getErrorRateClass(item.errorRate.error5xxRate)">
                    {{ (item.errorRate.error5xxRate * 100).toFixed(2) }}%
                  </span>
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </ElCard>
</template>

<script setup lang="ts">
  import type { IngressErrorMetrics } from '@/api'

  interface Props {
    errorData?: IngressErrorMetrics
    loading?: boolean
  }

  defineProps<Props>()

  const formatNumber = (num: number): string => {
    if (num >= 1000000) {
      return (num / 1000000).toFixed(2) + 'M'
    }
    if (num >= 1000) {
      return (num / 1000).toFixed(2) + 'K'
    }
    return num.toString()
  }

  const hasStatusCodes = (statusCodes: Record<string, number>): boolean => {
    return Object.keys(statusCodes).length > 0
  }

  const getErrorRateClass = (errorRate: number): string => {
    if (errorRate < 0.01) return 'success'
    if (errorRate < 0.05) return 'warning'
    return 'danger'
  }
</script>

<style lang="scss" scoped>
  .error-metrics-card {
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

      .error-detail {
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

          .error-rate-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 16px;

            .error-rate-item {
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

              .error-rate-label {
                font-size: 13px;
                color: #909399;
              }

              .error-rate-value {
                font-size: 24px;
                font-weight: 600;
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

          .status-code-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 16px;

            .status-code-group {
              background: #f5f7fa;
              border-radius: 8px;
              padding: 16px;
              transition: all 0.3s;

              &:hover {
                background: #ecf5ff;
              }

              .group-title {
                font-size: 13px;
                font-weight: 600;
                margin-bottom: 12px;

                &.success {
                  color: #67c23a;
                }

                &.info {
                  color: #909399;
                }

                &.warning {
                  color: #e6a23c;
                }

                &.danger {
                  color: #f56c6c;
                }
              }

              .status-code-list {
                display: flex;
                flex-direction: column;
                gap: 6px;

                .status-code-item {
                  display: flex;
                  justify-content: space-between;
                  align-items: center;
                  padding: 8px 10px;
                  background: #fff;
                  border-radius: 4px;
                  border-left: 3px solid #dcdfe6;

                  &.success {
                    border-left-color: #67c23a;
                  }

                  &.info {
                    border-left-color: #909399;
                  }

                  &.warning {
                    border-left-color: #e6a23c;
                  }

                  &.danger {
                    border-left-color: #f56c6c;
                  }

                  .code {
                    font-weight: 600;
                    color: #303133;
                    font-size: 13px;
                  }

                  .count {
                    font-size: 12px;
                    color: #909399;
                  }
                }
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

              .host-errors,
              .path-errors {
                display: flex;
                gap: 16px;

                .error-metric {
                  display: flex;
                  gap: 4px;
                  font-size: 13px;

                  .label {
                    color: #909399;
                  }

                  .value {
                    font-weight: 500;

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
        }
      }
    }
  }
</style>
