<script setup lang="ts">
  import { ref, onMounted } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import {
    ArrowLeft,
    Server,
    Shield,
    Activity,
    FileText,
    RefreshCw,
    Layers,
    Network
  } from 'lucide-vue-next'
  import { getClusterDetailApi, syncClusterApi, type ClusterDetail } from '@/api'
  import { ElMessage } from 'element-plus'
  import { environmentOptions } from '../constants'

  // 正常导入组件
  import BasicInfo from './tabs/BasicInfo.vue'
  import AuthConfig from './tabs/AuthConfig.vue'
  import MiddlewareManage from './tabs/MiddlewareManage.vue'
  import ResourceInfo from './tabs/ResourceInfo.vue'
  import OperationLog from './tabs/OperationLog.vue'
  import NetworkInfo from './tabs/NetworkInfo.vue'

  const route = useRoute()
  const router = useRouter()
  const clusterId = ref(Number(route.params.id))
  const activeTab = ref('basic')
  const loading = ref(false)
  const syncLoading = ref(false)
  const backLoading = ref(false)
  const avatarLoadFailed = ref(false)
  const clusterDetail = ref<ClusterDetail>()

  // 标签页配置
  const tabs = [
    {
      name: 'basic',
      label: '基本信息',
      icon: Server,
      gradient: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
      component: BasicInfo
    },
    {
      name: 'network',
      label: '网络信息',
      icon: Network,
      gradient: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
      component: NetworkInfo
    },
    {
      name: 'auth',
      label: '认证配置',
      icon: Shield,
      gradient: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
      component: AuthConfig
    },
    {
      name: 'middleware',
      label: '中间件管理',
      icon: Layers,
      gradient: 'linear-gradient(135deg, #ffa502 0%, #ff6348 100%)',
      component: MiddlewareManage
    },
    {
      name: 'resources',
      label: '资源信息',
      icon: Activity,
      gradient: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
      component: ResourceInfo
    },
    {
      name: 'logs',
      label: '操作日志',
      icon: FileText,
      gradient: 'linear-gradient(135deg, #fa709a 0%, #fee140 100%)',
      component: OperationLog
    }
  ]

  const healthStatusTextMap: Record<number, string> = {
    1: '健康',
    2: '异常',
    3: '降级',
    4: '未知'
  }

  // 获取环境配置
  const getEnvironmentConfig = (environment: string) => {
    return (
      environmentOptions.find((env) => env.value === environment) || {
        label: environment,
        shortLabel: environment.toUpperCase(),
        gradient: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)'
      }
    )
  }

  const getHealthStatusLabel = (healthStatus?: number) => {
    if (!healthStatus) return healthStatusTextMap[4]
    return healthStatusTextMap[healthStatus] || healthStatusTextMap[4]
  }

  // 获取集群详情
  const fetchClusterDetail = async () => {
    loading.value = true
    try {
      avatarLoadFailed.value = false
      clusterDetail.value = await getClusterDetailApi(clusterId.value)
    } finally {
      loading.value = false
    }
  }

  const handleAvatarError = () => {
    avatarLoadFailed.value = true
  }

  // 同步集群
  const handleSync = async () => {
    if (!clusterDetail.value) return
    syncLoading.value = true
    try {
      await syncClusterApi(clusterDetail.value.id)
      ElMessage.success('集群同步任务已触发')
      await fetchClusterDetail()
    } finally {
      syncLoading.value = false
    }
  }

  // 返回
  const goBack = async () => {
    backLoading.value = true
    try {
      await new Promise((resolve) => setTimeout(resolve, 200))
      router.replace('/cluster/list')
    } finally {
      backLoading.value = false
    }
  }

  onMounted(() => {
    fetchClusterDetail()
  })
</script>

<template>
  <div class="management-page">
    <div class="page-shell">
      <div class="cluster-header" v-if="clusterDetail">
        <div class="header-back">
          <el-button text @click="goBack" class="back-btn" :loading="backLoading">
            <ArrowLeft v-if="!backLoading" />
            返回列表
          </el-button>
        </div>

        <div class="header-content">
          <div class="cluster-avatar">
            <img
              v-if="clusterDetail.avatar && !avatarLoadFailed"
              :src="clusterDetail.avatar"
              alt="集群图标"
              @error="handleAvatarError"
            />
            <div v-else class="avatar-placeholder">
              <Server />
            </div>
          </div>

          <div class="cluster-info">
            <div class="info-heading">
              <div class="title-block">
                <div class="title-row">
                  <h1 class="cluster-name">{{ clusterDetail.name }}</h1>
                  <span class="page-tag">集群管理</span>
                </div>
                <p class="cluster-desc" v-if="clusterDetail.description">
                  {{ clusterDetail.description }}
                </p>
              </div>

              <div class="header-actions">
                <el-button @click="handleSync" :loading="syncLoading" class="sync-btn">
                  <RefreshCw v-if="!syncLoading" :class="{ 'animate-spin': syncLoading }" />
                  {{ syncLoading ? '同步中...' : '同步集群' }}
                </el-button>
              </div>
            </div>

            <div class="cluster-meta">
              <div class="meta-item">
                <span class="meta-label">环境</span>
                <span
                  class="meta-value env-tag"
                  :class="`env-${clusterDetail.environment || 'default'}`"
                >
                  {{ getEnvironmentConfig(clusterDetail.environment).label }}
                </span>
              </div>
              <div class="meta-item">
                <span class="meta-label">类型</span>
                <span class="meta-value">{{ clusterDetail.clusterType || '-' }}</span>
              </div>
              <div class="meta-item" v-if="clusterDetail.version">
                <span class="meta-label">版本</span>
                <span class="meta-value version">{{ clusterDetail.version }}</span>
              </div>
              <div class="meta-item">
                <span class="meta-label">状态</span>
                <span
                  class="meta-value status-tag"
                  :class="{
                    healthy: clusterDetail.healthStatus === 1,
                    unhealthy: clusterDetail.healthStatus === 2,
                    degraded: clusterDetail.healthStatus === 3,
                    unknown: clusterDetail.healthStatus === 4
                  }"
                >
                  <span class="status-dot"></span>
                  {{ getHealthStatusLabel(clusterDetail.healthStatus) }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="main-content">
        <div class="tab-container">
          <div class="tab-nav">
            <div
              v-for="tab in tabs"
              :key="tab.name"
              class="tab-item"
              :class="{ active: activeTab === tab.name }"
              @click="activeTab = tab.name"
            >
              <div class="tab-icon">
                <component :is="tab.icon" />
              </div>
              <span class="tab-label">{{ tab.label }}</span>
            </div>
          </div>

          <div class="tab-panels" v-loading="loading">
            <template v-for="tab in tabs" :key="tab.name">
              <component
                v-show="activeTab === tab.name"
                :is="tab.component"
                v-if="clusterDetail"
                :cluster-detail="clusterDetail"
                :cluster-id="clusterId"
                :cluster-uuid="clusterDetail.uuid"
                :is-active="activeTab === tab.name"
                @refresh="fetchClusterDetail"
              />
            </template>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
  .management-page {
    min-height: 100vh;
    background: #fff;
  }

  .page-shell {
    max-width: 1440px;
    margin: 0 auto;
    padding: 0;
  }

  .cluster-header {
    margin-bottom: 16px;
    background: #ffffff;
    border: 1px solid #e2e8f0;
    border-radius: 14px;
    box-shadow: 0 8px 24px rgba(15, 23, 42, 0.04);

    .header-back {
      display: flex;
      align-items: center;
      padding: 12px 20px 0;

      .back-btn {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 0;
        font-size: 14px;
        color: #475569;
        transition: color 0.2s ease;

        &:hover:not(:disabled) {
          color: #2563eb;
        }

        &:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        svg {
          width: 16px;
          height: 16px;
        }
      }
    }

    .header-content {
      display: flex;
      align-items: center;
      gap: 16px;
      padding: 18px 20px;

      .cluster-avatar {
        width: 56px;
        height: 56px;
        border-radius: 12px;
        overflow: hidden;
        background: #f8fafc;
        border: 1px solid #e2e8f0;
        flex-shrink: 0;

        img {
          width: 100%;
          height: 100%;
          object-fit: cover;
        }

        .avatar-placeholder {
          width: 100%;
          height: 100%;
          display: flex;
          align-items: center;
          justify-content: center;
          background: #eff6ff;

          svg {
            width: 24px;
            height: 24px;
            color: #2563eb;
          }
        }
      }

      .cluster-info {
        flex: 1;
        min-width: 0;

        .info-heading {
          display: flex;
          align-items: center;
          justify-content: space-between;
          gap: 16px;
        }

        .title-block {
          min-width: 0;
        }

        .title-row {
          display: flex;
          align-items: center;
          gap: 10px;
          flex-wrap: wrap;
          margin-bottom: 6px;
        }

        .page-tag {
          display: inline-flex;
          align-items: center;
          padding: 4px 10px;
          border-radius: 999px;
          background: #eff6ff;
          color: #2563eb;
          font-size: 12px;
          font-weight: 600;
        }

        .cluster-name {
          margin: 0;
          font-size: 22px;
          line-height: 1.2;
          font-weight: 700;
          color: #0f172a;
          word-break: break-word;
        }

        .cluster-desc {
          margin: 0;
          color: #64748b;
          font-size: 13px;
          line-height: 1.6;
        }

        .cluster-meta {
          display: flex;
          align-items: center;
          gap: 10px;
          flex-wrap: wrap;
          margin-top: 14px;

          .meta-item {
            display: inline-flex;
            align-items: center;
            gap: 8px;
            padding: 8px 12px;
            background: #f8fafc;
            border: 1px solid #e2e8f0;
            border-radius: 10px;

            .meta-label {
              font-size: 12px;
              line-height: 1.2;
              color: #94a3b8;
            }

            .meta-value {
              font-size: 14px;
              line-height: 1.4;
              font-weight: 600;
              color: #0f172a;

              &.env-tag,
              &.status-tag,
              &.version {
                display: inline-flex;
                align-items: center;
              }

              &.env-tag {
                padding: 4px 10px;
                border-radius: 999px;
                border: 1px solid #dbeafe;
                background: #eff6ff;
                color: #2563eb;
                font-size: 12px;

                &.env-development {
                  background: #ecfdf5;
                  border-color: #bbf7d0;
                  color: #059669;
                }

                &.env-testing {
                  background: #eff6ff;
                  border-color: #bfdbfe;
                  color: #2563eb;
                }

                &.env-staging {
                  background: #fff7ed;
                  border-color: #fed7aa;
                  color: #ea580c;
                }

                &.env-production {
                  background: #fef2f2;
                  border-color: #fecaca;
                  color: #dc2626;
                }
              }

              &.version {
                padding: 4px 10px;
                border-radius: 8px;
                background: #f1f5f9;
                border: 1px solid #e2e8f0;
                font-family: var(--ax-font-mono);
                font-size: 13px;
                color: #334155;
              }

              &.status-tag {
                gap: 8px;
                padding: 4px 10px;
                border-radius: 999px;
                border: 1px solid transparent;
                font-size: 12px;

                .status-dot {
                  width: 8px;
                  height: 8px;
                  border-radius: 50%;
                }

                &.healthy {
                  background: #ecfdf5;
                  border-color: #bbf7d0;
                  color: #059669;

                  .status-dot {
                    background: #10b981;
                  }
                }

                &.unhealthy {
                  background: #fef2f2;
                  border-color: #fecaca;
                  color: #dc2626;

                  .status-dot {
                    background: #ef4444;
                  }
                }

                &.degraded {
                  background: #fffbeb;
                  border-color: #fde68a;
                  color: #d97706;

                  .status-dot {
                    background: #f59e0b;
                  }
                }

                &.unknown {
                  background: #f8fafc;
                  border-color: #e2e8f0;
                  color: #64748b;

                  .status-dot {
                    background: #94a3b8;
                  }
                }
              }
            }
          }
        }
      }
    }
  }

  .header-actions {
    flex-shrink: 0;

    .sync-btn {
      min-width: 104px;
      height: 36px;
      border-color: #cbd5e1;
      color: #334155;
      background: #ffffff;
      font-weight: 600;
      border-radius: 10px;
      transition:
        border-color 0.2s ease,
        color 0.2s ease,
        background-color 0.2s ease;

      &:hover:not(:disabled) {
        border-color: #93c5fd;
        color: #2563eb;
        background: #eff6ff;
      }

      &:disabled {
        opacity: 0.6;
        cursor: not-allowed;
      }

      svg {
        width: 16px;
        height: 16px;
        margin-right: 6px;
      }
    }
  }

  .main-content {
    .tab-container {
      background: #ffffff;
      border: 1px solid #e2e8f0;
      border-radius: 14px;
      overflow: hidden;
      box-shadow: 0 8px 24px rgba(15, 23, 42, 0.04);

      .tab-nav {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 12px 16px;
        overflow-x: auto;
        border-bottom: 1px solid #eef2f7;
        background: #ffffff;

        .tab-item {
          display: inline-flex;
          align-items: center;
          gap: 10px;
          flex-shrink: 0;
          padding: 8px 12px;
          border: 1px solid transparent;
          border-radius: 10px;
          cursor: pointer;
          color: #475569;
          transition: all 0.2s ease;

          .tab-icon {
            width: 24px;
            height: 24px;
            border-radius: 8px;
            display: flex;
            align-items: center;
            justify-content: center;
            background: #f8fafc;
            border: 1px solid #e2e8f0;

            svg {
              width: 13px;
              height: 13px;
              color: #64748b;
            }
          }

          .tab-label {
            font-size: 13px;
            font-weight: 600;
            line-height: 1;
            white-space: nowrap;
          }

          &:hover {
            background: #f8fafc;
            border-color: #e2e8f0;
            color: #0f172a;
          }

          &.active {
            background: #eff6ff;
            border-color: #bfdbfe;
            color: #1d4ed8;

            .tab-icon {
              background: #dbeafe;
              border-color: #bfdbfe;

              svg {
                color: #2563eb;
              }
            }
          }
        }
      }

      .tab-panels {
        min-height: 560px;
        padding: 20px;
        background: #ffffff;
      }
    }
  }

  @keyframes spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }

  .animate-spin {
    animation: spin 1s linear infinite;
  }

  .management-page {
    :deep(
      .basic-info-panel,
      .network-info-panel,
      .auth-config-panel,
      .middleware-panel,
      .resource-info-panel,
      .cluster-audit-panel
    ) {
      background: transparent;
    }

    :deep(.panel-header) {
      display: flex;
      align-items: center !important;
      justify-content: space-between !important;
      gap: 16px !important;
      margin-bottom: 16px !important;
      padding: 16px 18px !important;
       background: #ffffff !important;
      border: 1px solid #e2e8f0 !important;
      border-radius: 12px !important;
      box-shadow: none !important;
    }

    :deep(.panel-header .header-title) {
      gap: 12px;
    }

    :deep(.panel-header .header-title > svg) {
      width: 18px;
      height: 18px;
      color: #2563eb;
    }

    :deep(.panel-header .header-title h2),
    :deep(.panel-header .header-title span),
    :deep(.panel-header .header-text h2),
    :deep(.panel-header .header-text h3) {
      color: #0f172a;
      font-size: 16px;
      font-weight: 600;
      margin: 0;
    }

    :deep(.panel-header .header-title p),
    :deep(.panel-header .header-text p) {
      margin-top: 4px;
      color: #64748b;
      font-size: 13px;
    }

    :deep(.panel-header .header-left),
    :deep(.panel-header .header-right),
    :deep(.panel-header .header-text) {
      display: flex;
      align-items: center;
      gap: 12px;
    }

    :deep(.panel-header .header-actions) {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
    }

    :deep(.panel-header .header-actions .el-button) {
      height: 36px !important;
      border-radius: 10px !important;
    }

    :deep(.info-group) {
      margin-bottom: 16px !important;
      padding: 20px !important;
      border: 1px solid #e2e8f0 !important;
      border-radius: 12px !important;
      background: #f8fafc !important;
      box-shadow: none !important;
    }

    :deep(.group-title) {
      margin-bottom: 16px;
      color: #0f172a;
      font-size: 15px;
      font-weight: 600;
    }

    :deep(.group-title .title-icon) {
      width: 28px;
      height: 28px;
      border: 1px solid #dbeafe;
      border-radius: 8px;
      background: #eff6ff !important;
    }

    :deep(.group-title .title-icon svg) {
      width: 14px;
      height: 14px;
      color: #2563eb !important;
    }

    :deep(.card-title),
    :deep(.section-title) {
      color: #0f172a;
    }

    :deep(.empty-state) {
      padding: 40px 20px !important;
      border: 1px dashed #dbe3ee !important;
      border-radius: 12px !important;
      background: #f8fafc !important;
    }

    :deep(.auth-status .el-alert),
    :deep(.auth-form-container),
    :deep(.incluster-alert),
    :deep(.security-alert) {
      border-radius: 12px !important;
      border: 1px solid #e2e8f0 !important;
      background: #f8fafc !important;
      box-shadow: none !important;
    }

    :deep(.auth-content .auth-type-selector) {
      margin-bottom: 20px;
      gap: 14px;
    }

    :deep(.auth-content .auth-type-card) {
      border: 1px solid #e2e8f0 !important;
      border-radius: 12px !important;
      padding: 16px !important;
      box-shadow: none !important;
      background: #ffffff !important;
    }

    :deep(.auth-content .auth-type-card .card-gradient) {
      height: 2px !important;
    }

    :deep(.auth-content .auth-type-card:hover) {
      background: #f8fafc !important;
      border-color: #cbd5e1 !important;
    }

    :deep(.auth-content .auth-type-card.active) {
      border-color: #bfdbfe !important;
      background: #eff6ff !important;
      box-shadow: none !important;
    }

    :deep(.resource-overview) {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: 14px;
      margin-bottom: 20px;
    }

    :deep(.resource-card),
    :deep(.detailed-info),
    :deep(.info-section),
    :deep(.filter-toolbar),
    :deep(.log-container),
    :deep(.pagination-bar) {
      border-radius: 12px;
    }

    :deep(.resource-card) {
      border: 1px solid #e2e8f0 !important;
      box-shadow: none !important;
    }

    :deep(.resource-card:hover) {
      transform: translateY(-2px) !important;
      box-shadow: 0 10px 24px rgba(15, 23, 42, 0.08) !important;
    }

    :deep(.resource-card .card-icon.cpu),
    :deep(.resource-card .card-icon.memory),
    :deep(.resource-card .card-icon.storage),
    :deep(.resource-card .card-icon.pod),
    :deep(.resource-card .card-icon.gpu) {
      background: #eff6ff !important;
      color: #2563eb !important;
    }

    :deep(.resource-card .card-icon svg) {
      color: currentColor;
    }

    :deep(.detailed-info),
    :deep(.info-section),
    :deep(.filter-toolbar) {
      border: 1px solid #e2e8f0 !important;
      background: #f8fafc !important;
      box-shadow: none !important;
    }

    :deep(.log-container) {
      padding: 16px !important;
      border: 1px solid #eef2f7 !important;
      background: #ffffff !important;
    }

    :deep(.pagination-bar) {
      margin-top: 16px !important;
      padding: 14px 16px 0 !important;
      border-top: 1px solid #eef2f7 !important;
    }
  }

  @media (max-width: 1024px) {
    .cluster-header {
      .header-content {
        padding: 16px;
      }
    }

    .main-content {
      .tab-container {
        .tab-panels {
          padding: 20px;
        }
      }
    }
  }

  @media (max-width: 768px) {
    .page-shell {
      padding: 0;
    }

    .cluster-header {
      .header-back {
        padding: 12px 16px 0;
      }

      .header-content {
        flex-direction: column;
        align-items: flex-start;
      }
    }

    .cluster-info {
      width: 100%;

      .info-heading {
        flex-direction: column;
      }
    }

    .header-actions {
      width: 100%;

      .sync-btn {
        width: 100%;
      }
    }

    .main-content {
      .tab-container {
        .tab-nav {
          padding: 12px;
        }

        .tab-panels {
          min-height: auto;
          padding: 16px;
        }
      }
    }

    .management-page {
      :deep(.panel-header),
      :deep(.filter-toolbar),
      :deep(.dialog-footer) {
        flex-direction: column;
        align-items: flex-start;
      }

      :deep(.auth-content .auth-type-selector) {
        grid-template-columns: 1fr;
      }
    }
  }
</style>
