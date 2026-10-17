<template>
  <ElCard class="certificate-management-card" shadow="never">
    <template #header>
      <div class="card-header">
        <span class="card-title">证书管理</span>
        <ElTag
          v-if="certificateData"
          :type="certificateData.httpsRequestPercent > 50 ? 'success' : 'warning'"
          size="large"
        >
          HTTPS: {{ certificateData.httpsRequestPercent.toFixed(1) }}%
        </ElTag>
      </div>
    </template>
    <div v-loading="loading" class="card-content">
      <ElEmpty
        v-if="
          !certificateData ||
          !certificateData.certificates ||
          certificateData.certificates.length === 0
        "
        description="暂无证书数据"
        :image-size="100"
      />
      <div v-else class="certificate-detail">
        <!-- 证书统计 -->
        <div class="section">
          <h4 class="section-title">证书统计</h4>
          <div class="certificate-stats">
            <div class="stat-item">
              <span class="stat-label">证书总数</span>
              <span class="stat-value">{{ certificateData.certificates.length }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">即将过期</span>
              <span class="stat-value warning">{{ certificateData.expiringCount }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">已过期</span>
              <span class="stat-value danger">{{ certificateData.expiredCount }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">HTTPS 请求占比</span>
              <span class="stat-value">{{ certificateData.httpsRequestPercent.toFixed(1) }}%</span>
            </div>
          </div>
        </div>

        <!-- 证书列表 -->
        <div class="section">
          <h4 class="section-title">证书列表</h4>
          <div class="certificate-list">
            <div
              v-for="cert in sortedCertificates"
              :key="`${cert.namespace}-${cert.name}`"
              class="certificate-item"
              :class="{ 'is-expired': cert.isExpired, 'is-expiring': cert.isExpiring }"
            >
              <div class="certificate-info">
                <div class="certificate-header">
                  <span class="certificate-name">{{ cert.name }}</span>
                  <div class="certificate-tags">
                    <ElTag v-if="cert.isExpired" type="danger" size="small"> 已过期 </ElTag>
                    <ElTag v-else-if="cert.isExpiring" type="warning" size="small">
                      即将过期 ({{ cert.daysRemaining }} 天)
                    </ElTag>
                    <ElTag v-else type="success" size="small">
                      有效 ({{ cert.daysRemaining }} 天)
                    </ElTag>
                  </div>
                </div>
                <div class="certificate-meta">
                  <span class="meta-item">
                    <span class="meta-label">命名空间:</span>
                    <span class="meta-value">{{ cert.namespace }}</span>
                  </span>
                  <span class="meta-item">
                    <span class="meta-label">过期时间:</span>
                    <span class="meta-value">{{ formatExpirationTime(cert.expirationTime) }}</span>
                  </span>
                  <span class="meta-item">
                    <span class="meta-label">剩余天数:</span>
                    <span
                      class="meta-value"
                      :class="getDaysRemainingClass(cert.daysRemaining, cert.isExpired)"
                    >
                      {{ cert.isExpired ? '已过期' : `${cert.daysRemaining} 天` }}
                    </span>
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </ElCard>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import type { IngressCertificateMetrics } from '@/api'

  interface Props {
    certificateData?: IngressCertificateMetrics
    loading?: boolean
  }

  const props = defineProps<Props>()

  // 按剩余天数排序，即将过期的排在前面
  const sortedCertificates = computed(() => {
    if (!props.certificateData?.certificates) return []
    return [...props.certificateData.certificates].sort((a, b) => {
      // 已过期的排在最前面
      if (a.isExpired && !b.isExpired) return -1
      if (!a.isExpired && b.isExpired) return 1
      // 都已过期或都没过期，按剩余天数排序
      return a.daysRemaining - b.daysRemaining
    })
  })

  const formatExpirationTime = (timestamp: number): string => {
    return new Date(timestamp).toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  const getDaysRemainingClass = (days: number, isExpired: boolean): string => {
    if (isExpired) return 'danger'
    if (days < 7) return 'danger'
    if (days < 30) return 'warning'
    return 'success'
  }
</script>

<style lang="scss" scoped>
  .certificate-management-card {
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

      .certificate-detail {
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

          .certificate-stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 16px;

            .stat-item {
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

              .stat-label {
                font-size: 13px;
                color: #909399;
              }

              .stat-value {
                font-size: 24px;
                font-weight: 600;
                color: #303133;

                &.warning {
                  color: #e6a23c;
                }

                &.danger {
                  color: #f56c6c;
                }
              }
            }
          }

          .certificate-list {
            display: flex;
            flex-direction: column;
            gap: 12px;

            .certificate-item {
              padding: 16px;
              background: #f5f7fa;
              border-radius: 8px;
              border-left: 4px solid #67c23a;
              transition: all 0.3s;

              &:hover {
                background: #ecf5ff;
              }

              &.is-expiring {
                border-left-color: #e6a23c;
                background: linear-gradient(135deg, #fff7e6 0%, #ffe7ba 100%);
              }

              &.is-expired {
                border-left-color: #f56c6c;
                background: linear-gradient(135deg, #fff1f0 0%, #ffccc7 100%);
              }

              .certificate-info {
                .certificate-header {
                  display: flex;
                  justify-content: space-between;
                  align-items: center;
                  margin-bottom: 12px;

                  .certificate-name {
                    font-size: 15px;
                    font-weight: 600;
                    color: #303133;
                  }

                  .certificate-tags {
                    display: flex;
                    gap: 8px;
                  }
                }

                .certificate-meta {
                  display: flex;
                  flex-wrap: wrap;
                  gap: 16px;

                  .meta-item {
                    display: flex;
                    gap: 4px;
                    font-size: 13px;

                    .meta-label {
                      color: #909399;
                    }

                    .meta-value {
                      color: #303133;
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
  }
</style>
