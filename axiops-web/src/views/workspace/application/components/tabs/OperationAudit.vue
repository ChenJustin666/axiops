<template>
  <div class="operation-audit-timeline">
    <!-- 头部区域 -->
    <div class="audit-header">
      <div class="header-left">
        <div class="icon-wrapper">
          <History :size="20" class="header-icon" />
        </div>
        <div class="title-group">
          <h2 class="header-title">操作审计</h2>
          <span class="header-subtitle">记录服务的变更历史与操作轨迹</span>
        </div>
        <ElTag v-if="pagination.total > 0" type="info" effect="plain" round class="count-tag">
          {{ pagination.total }} 条记录
        </ElTag>
      </div>
      <div class="header-right">
        <ElButton :icon="RefreshCw" circle @click="handleRefresh" :loading="loading" />
      </div>
    </div>

    <!-- 内容区域 -->
    <div class="timeline-wrapper" v-loading="loading">
      <ElEmpty v-if="!canLoadData" description="请先选择项目、集群和工作空间" :image-size="100" />

      <ElEmpty
        v-else-if="!loading && auditLogs.length === 0"
        description="暂无审计记录"
        :image-size="100"
      />

      <template v-else>
        <!-- 时间线滚动区域 -->
        <div class="timeline-scroll-area">
          <ElTimeline>
            <ElTimelineItem
              v-for="log in auditLogs"
              :key="log.id"
              :timestamp="formatTime(log.createdAt)"
              placement="top"
              :color="getTimelineColor(log.status)"
              :hollow="log.status !== 1"
            >
              <div class="timeline-card">
                <div class="card-content">
                  <!-- 左侧主体内容 -->
                  <div class="card-main">
                    <div class="card-header">
                      <div class="title-row">
                        <div :class="['action-icon-wrapper', `action-${getIconType(log.title)}`]">
                          <component :is="getActionIcon(log.title)" :size="14" />
                        </div>
                        <h3 class="card-title">{{ log.title }}</h3>
                        <ElTag :type="getStatusTagType(log.status)" size="small" effect="light">
                          {{ formatStatus(log.status) }}
                        </ElTag>
                      </div>

                      <div class="meta-row">
                        <div class="meta-item">
                          <User :size="12" />
                          <span>{{ log.operatorName }}</span>
                        </div>
                        <ElDivider direction="vertical" />
                        <div class="meta-item" v-if="log.applicationName">
                          <Package :size="12" />
                          <span>{{ log.applicationName }}</span>
                        </div>
                      </div>
                    </div>

                    <div class="card-body">
                      <div class="action-detail-box">
                        {{ formatDetailPreview(log.actionDetail) }}
                      </div>
                    </div>
                  </div>

                  <!-- 右侧操作按钮 -->
                  <div class="card-actions">
                    <ElTooltip content="查看详情" placement="top">
                      <ElButton
                        type="primary"
                        text
                        bg
                        circle
                        size="small"
                        :icon="Eye"
                        @click="handleView(log)"
                      />
                    </ElTooltip>
                  </div>
                </div>
              </div>
            </ElTimelineItem>
          </ElTimeline>
        </div>

        <!-- 分页固定在底部 -->
        <div class="pagination-footer" v-if="pagination.total > 0">
          <ElPagination
            v-model:current-page="pagination.current"
            v-model:page-size="pagination.size"
            :page-sizes="[10, 20, 50, 100]"
            :total="pagination.total"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="handleSizeChange"
            @current-change="handleCurrentChange"
            background
            small
          />
        </div>
      </template>
    </div>

    <!-- 详情弹窗 -->
    <ElDialog v-model="detailVisible" title="审计详情" width="650px" align-center destroy-on-close>
      <div class="detail-content" v-if="currentAudit">
        <ElDescriptions :column="2" border size="default">
          <ElDescriptionsItem label="操作人" label-class-name="desc-label">
            <div class="desc-value"><User :size="14" /> {{ currentAudit.operatorName }}</div>
          </ElDescriptionsItem>
          <ElDescriptionsItem label="操作状态" label-class-name="desc-label">
            <ElTag :type="getStatusTagType(currentAudit.status)" size="small">
              {{ formatStatus(currentAudit.status) }}
            </ElTag>
          </ElDescriptionsItem>
          <ElDescriptionsItem label="操作时间" label-class-name="desc-label">
            {{ formatTime(currentAudit.createdAt) }}
          </ElDescriptionsItem>
          <ElDescriptionsItem label="应用名称" label-class-name="desc-label">
            {{ currentAudit.applicationName || '-' }}
          </ElDescriptionsItem>
          <ElDescriptionsItem label="工作空间" label-class-name="desc-label">
            {{ currentAudit.workspaceName || '-' }}
          </ElDescriptionsItem>
          <ElDescriptionsItem label="集群" label-class-name="desc-label">
            {{ currentAudit.clusterName || '-' }}
          </ElDescriptionsItem>
        </ElDescriptions>

        <div class="detail-json-wrapper">
          <div class="detail-json-header">操作详情数据</div>
          <div class="detail-json-content">
            {{ currentAudit.actionDetail }}
          </div>
        </div>
      </div>
      <template #footer>
        <ElButton @click="detailVisible = false">关闭</ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { ref, reactive, watch, onMounted, computed } from 'vue'
  import { ElMessage } from 'element-plus'
  import {
    History,
    RefreshCw,
    Eye,
    User,
    Package,
    Plus,
    Edit,
    X,
    Zap,
    RotateCw,
    Terminal
  } from 'lucide-vue-next'
  import { useProjectStore } from '@/store/modules/project'
  import { useApplicationManagementStore } from '@/store/modules/applicationManagement'
  import {
    searchProjectAuditLogApi,
    getAuditStatusInfo,
    type ProjectAuditLog,
    type OnecProjectApplication,
    type ProjectWorkspace
  } from '@/api/manager/audit'
  import dayjs from 'dayjs'

  interface Props {
    application: OnecProjectApplication
    workspace: ProjectWorkspace | null
    refreshTrigger: number
  }

  const props = defineProps<Props>()

  const projectStore = useProjectStore()
  const managementStore = useApplicationManagementStore()

  const selectedProject = computed(() => projectStore.selectedProject)
  const currentCluster = computed(() => managementStore.selectedCluster)

  const canLoadData = computed(() => {
    return (
      !!selectedProject.value && !!currentCluster.value && !!props.workspace && !!props.application
    )
  })

  const auditLogs = ref<ProjectAuditLog[]>([])
  const loading = ref(false)
  const detailVisible = ref(false)
  const currentAudit = ref<ProjectAuditLog | null>(null)
  const lastAuditRequestKey = ref('')

  const pagination = reactive({
    current: 1,
    size: 20,
    total: 0
  })

  // 样式逻辑
  const getTimelineColor = (status: number) => (status === 1 ? '#10b981' : '#ef4444')

  const getIconType = (title?: string) => {
    if (!title) return 'default'
    if (title.includes('创建')) return 'create'
    if (title.includes('更新') || title.includes('修改')) return 'update'
    if (title.includes('删除')) return 'delete'
    if (title.includes('扩容') || title.includes('缩容')) return 'scale'
    if (title.includes('重启')) return 'restart'
    return 'default'
  }

  const getActionIcon = (title?: string) => {
    const type = getIconType(title)
    const map: Record<string, any> = {
      create: Plus,
      update: Edit,
      delete: X,
      scale: Zap,
      restart: RotateCw,
      default: Terminal
    }
    return map[type]
  }

  const getStatusTagType = (status: number) => getAuditStatusInfo(status).type as any
  const formatStatus = (status: number) => getAuditStatusInfo(status).label
  const formatTime = (timestamp?: number) =>
    timestamp ? dayjs(timestamp * 1000).format('YYYY-MM-DD HH:mm:ss') : '-'

  const formatDetailPreview = (detail: string) => {
    if (!detail) return '无详细信息'
    return detail.length > 100 ? detail.substring(0, 100) + '...' : detail
  }

  // 数据操作
  const handleView = (audit: ProjectAuditLog) => {
    currentAudit.value = audit
    detailVisible.value = true
  }

  const getAuditRequestKey = (refreshToken = props.refreshTrigger) => {
    return [
      selectedProject.value?.id || 0,
      currentCluster.value?.clusterUuid || 'no-cluster',
      props.workspace?.id || 0,
      props.application.id,
      pagination.current,
      pagination.size,
      refreshToken
    ].join(':')
  }

  const loadAuditLogs = async (
    options: {
      force?: boolean
      requestKey?: string
    } = {}
  ) => {
    const { force = false, requestKey = getAuditRequestKey() } = options

    if (!canLoadData.value) return

    if (loading.value || (!force && requestKey === lastAuditRequestKey.value)) return

    lastAuditRequestKey.value = requestKey

    loading.value = true
    try {
      const params = {
        page: pagination.current,
        pageSize: pagination.size,
        projectId: selectedProject.value!.id,
        clusterUuid: currentCluster.value!.clusterUuid,
        workspaceId: props.workspace!.id,
        applicationId: props.application.id,
        status: -1
      }

      const response = await searchProjectAuditLogApi(params as any)
      auditLogs.value = response.items || []
      pagination.total = response.total || 0
    } catch (error) {
      console.error('加载审计日志失败:', error)
    } finally {
      loading.value = false
    }
  }

  const handleRefresh = () => {
    pagination.current = 1
    loadAuditLogs({ force: true, requestKey: getAuditRequestKey() })
  }

  const handleSizeChange = (size: number) => {
    pagination.size = size
    pagination.current = 1
    loadAuditLogs({ force: true, requestKey: getAuditRequestKey() })
  }

  const handleCurrentChange = (current: number) => {
    pagination.current = current
    loadAuditLogs({ force: true, requestKey: getAuditRequestKey() })
  }

  watch(
    () => props.refreshTrigger,
    (newVal, oldVal) => {
      if (newVal > 0 && newVal !== oldVal) {
        loadAuditLogs({ requestKey: getAuditRequestKey(newVal) })
      }
    }
  )

  watch(
    [
      () => selectedProject.value?.id,
      () => currentCluster.value?.id,
      () => props.workspace?.id,
      () => props.application.id
    ],
    (newValues, oldValues) => {
      const hasOldValues = oldValues.some((v) => v !== undefined)
      if (hasOldValues) {
        pagination.current = 1
        loadAuditLogs({ force: true, requestKey: getAuditRequestKey() })
      }
    }
  )

  onMounted(() => {
    loadAuditLogs()
  })
</script>

<style lang="scss" scoped>
  .operation-audit-timeline {
    height: 100%;
    min-height: 400px;
    max-height: 800px;
    display: flex;
    flex-direction: column;
    background: #fff;
    border-radius: 8px;
    overflow: hidden;

    // 头部区域
    .audit-header {
      flex-shrink: 0;
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 16px 20px;
      background: linear-gradient(to right, #f8fafc, #fff);
      border-bottom: 1px solid #eef2f6;

      .header-left {
        display: flex;
        align-items: center;
        gap: 12px;

        .icon-wrapper {
          width: 36px;
          height: 36px;
          border-radius: 10px;
          background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
          display: flex;
          align-items: center;
          justify-content: center;
          box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);

          .header-icon {
            color: #fff;
          }
        }

        .title-group {
          display: flex;
          flex-direction: column;

          .header-title {
            margin: 0;
            font-size: 16px;
            font-weight: 600;
            color: #1e293b;
          }

          .header-subtitle {
            font-size: 12px;
            color: #94a3b8;
            margin-top: 2px;
          }
        }

        .count-tag {
          margin-left: 4px;
          font-weight: 500;
        }
      }
    }

    // 内容包裹层
    .timeline-wrapper {
      flex: 1;
      display: flex;
      flex-direction: column;
      min-height: 0;
      overflow: hidden;
    }

    // 时间线滚动区域
    .timeline-scroll-area {
      flex: 1;
      overflow-y: auto;
      padding: 20px 24px;

      // 自定义滚动条
      &::-webkit-scrollbar {
        width: 6px;
      }

      &::-webkit-scrollbar-track {
        background: #f1f5f9;
        border-radius: 3px;
      }

      &::-webkit-scrollbar-thumb {
        background: #cbd5e1;
        border-radius: 3px;

        &:hover {
          background: #94a3b8;
        }
      }

      :deep(.el-timeline) {
        padding-left: 0;
      }

      :deep(.el-timeline-item__timestamp) {
        font-family: var(--ax-font-mono);
        font-size: 12px;
        color: #94a3b8;
      }

      :deep(.el-timeline-item__wrapper) {
        padding-left: 24px;
      }
    }

    // 时间线卡片
    .timeline-card {
      background: #fff;
      border: 1px solid #e2e8f0;
      border-radius: 10px;
      padding: 14px 16px;
      margin-top: 6px;
      transition: all 0.2s ease;

      &:hover {
        border-color: #cbd5e1;
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
      }

      .card-content {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 16px;

        .card-main {
          flex: 1;
          min-width: 0;

          .card-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 10px;
            flex-wrap: wrap;
            gap: 8px;

            .title-row {
              display: flex;
              align-items: center;
              gap: 10px;

              .action-icon-wrapper {
                width: 28px;
                height: 28px;
                border-radius: 6px;
                display: flex;
                align-items: center;
                justify-content: center;

                &.action-create {
                  background: #dcfce7;
                  color: #16a34a;
                }

                &.action-update {
                  background: #dbeafe;
                  color: #2563eb;
                }

                &.action-delete {
                  background: #fee2e2;
                  color: #dc2626;
                }

                &.action-scale {
                  background: #ffedd5;
                  color: #ea580c;
                }

                &.action-restart {
                  background: #fef3c7;
                  color: #d97706;
                }

                &.action-default {
                  background: #f1f5f9;
                  color: #64748b;
                }
              }

              .card-title {
                margin: 0;
                font-size: 14px;
                font-weight: 600;
                color: #334155;
              }
            }

            .meta-row {
              display: flex;
              align-items: center;
              gap: 8px;

              .meta-item {
                display: flex;
                align-items: center;
                gap: 4px;
                font-size: 12px;
                color: #64748b;
              }

              :deep(.el-divider--vertical) {
                margin: 0;
                height: 12px;
              }
            }
          }

          .card-body {
            .action-detail-box {
              background: #f8fafc;
              border: 1px solid #e2e8f0;
              border-radius: 6px;
              padding: 10px 12px;
              font-family: var(--ax-font-mono);
              font-size: 12px;
              color: #475569;
              line-height: 1.5;
              word-break: break-all;
            }
          }
        }

        .card-actions {
          flex-shrink: 0;
          display: flex;
          align-items: center;
        }
      }
    }

    // 分页固定在底部
    .pagination-footer {
      flex-shrink: 0;
      padding: 12px 20px;
      background: #fafbfc;
      border-top: 1px solid #eef2f6;
      display: flex;
      justify-content: center;
    }

    // 详情弹窗
    .detail-content {
      :deep(.desc-label) {
        width: 90px;
        color: #64748b;
        font-weight: 500;
        background: #f8fafc;
      }

      .desc-value {
        display: flex;
        align-items: center;
        gap: 6px;
      }

      .detail-json-wrapper {
        margin-top: 16px;
        border: 1px solid #e2e8f0;
        border-radius: 8px;
        overflow: hidden;

        .detail-json-header {
          padding: 10px 14px;
          background: #f8fafc;
          font-size: 13px;
          font-weight: 600;
          color: #475569;
          border-bottom: 1px solid #e2e8f0;
        }

        .detail-json-content {
          padding: 14px;
          background: #1e293b;
          color: #e2e8f0;
          font-family: var(--ax-font-mono);
          font-size: 12px;
          line-height: 1.6;
          white-space: pre-wrap;
          word-break: break-all;
          max-height: 350px;
          overflow-y: auto;
        }
      }
    }
  }
</style>
