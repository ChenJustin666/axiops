<template>
  <ElDialog
    v-model="visible"
    title="告警详情"
    width="900px"
    align-center
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    destroy-on-close
  >
    <div v-loading="loading" class="alert-detail">
      <template v-if="!loading && alertData">
        <!-- 头部 -->
        <div class="detail-header">
          <div class="header-left">
            <h3>{{ alertData.alertName }}</h3>
            <span class="alert-id">ID: {{ alertData.id }}</span>
          </div>
          <div class="header-right">
            <ElTag :type="getSeverityType(alertData.severity)" effect="dark">
              {{ AlertInstanceSeverityNames[alertData.severity] || alertData.severity }}
            </ElTag>
            <ElTag :type="getStatusType(alertData.status)" effect="dark">
              {{ AlertStatusNames[alertData.status] || alertData.status }}
            </ElTag>
          </div>
        </div>

        <!-- 核心信息 -->
        <div class="core-info">
          <div class="info-cards">
            <div class="info-card" :class="getSeverityType(alertData.severity)">
              <span class="card-label">严重级别</span>
              <span class="card-value">{{
                AlertInstanceSeverityNames[alertData.severity] || alertData.severity
              }}</span>
            </div>
            <div class="info-card" :class="getStatusType(alertData.status)">
              <span class="card-label">告警状态</span>
              <span class="card-value">{{
                AlertStatusNames[alertData.status] || alertData.status
              }}</span>
            </div>
            <div class="info-card">
              <span class="card-label">持续时长</span>
              <span class="card-value">{{ formatDuration(alertData.duration) }}</span>
            </div>
            <div class="info-card">
              <span class="card-label">重复次数</span>
              <span class="card-value">{{ alertData.repeatCount || 0 }}</span>
            </div>
          </div>

          <div class="timeline-box">
            <div class="box-title">时间轴</div>
            <div class="timeline-list">
              <div class="timeline-row" :class="{ active: alertData.startsAt }">
                <span class="dot"></span>
                <span class="label">告警开始</span>
                <span class="time">{{ formatTimestamp(alertData.startsAt) }}</span>
              </div>
              <div class="timeline-row" :class="{ active: alertData.endsAt }">
                <span class="dot"></span>
                <span class="label">告警结束</span>
                <span class="time">{{ formatTimestamp(alertData.endsAt) }}</span>
              </div>
              <div class="timeline-row success" :class="{ active: alertData.resolvedAt }">
                <span class="dot"></span>
                <span class="label">告警恢复</span>
                <span class="time">{{ formatTimestamp(alertData.resolvedAt) }}</span>
              </div>
              <div class="timeline-row" :class="{ active: alertData.lastNotifiedAt }">
                <span class="dot"></span>
                <span class="label">最后通知</span>
                <span class="time">{{ formatTimestamp(alertData.lastNotifiedAt) }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 告警概要 -->
        <div class="section" v-if="hasSummaryInfo">
          <div class="section-title">告警概要</div>
          <div class="summary-items">
            <div class="summary-item" v-if="parsedAnnotations.summary">
              <div class="item-label">概要说明</div>
              <div class="item-value">{{ parsedAnnotations.summary }}</div>
            </div>
            <div class="summary-item" v-if="parsedAnnotations.description">
              <div class="item-label">详细描述</div>
              <div class="item-value">{{ parsedAnnotations.description }}</div>
            </div>
            <div class="summary-item highlight" v-if="parsedAnnotations.value">
              <div class="item-label">触发值</div>
              <div class="item-value trigger">{{ parsedAnnotations.value }}</div>
            </div>
          </div>
        </div>

        <!-- 基本信息 -->
        <div class="section">
          <div class="section-title">基本信息</div>
          <div class="info-rows">
            <div class="info-row">
              <span class="row-label">告警实例</span>
              <span class="row-value copy" @click="copyText(alertData.instance)">{{
                alertData.instance
              }}</span>
            </div>
            <div class="info-row">
              <span class="row-label">告警指纹</span>
              <span class="row-value copy" @click="copyText(alertData.fingerprint)">{{
                alertData.fingerprint
              }}</span>
            </div>
            <div class="info-row" v-if="alertData.generatorUrl">
              <span class="row-label">生成器URL</span>
              <a :href="alertData.generatorUrl" target="_blank" class="row-link">{{
                alertData.generatorUrl
              }}</a>
            </div>
          </div>
        </div>

        <!-- 关联资源 -->
        <div class="section">
          <div class="section-title">关联资源</div>
          <div class="resource-list">
            <div class="resource-row">
              <span class="res-type">集群</span>
              <span class="res-name">{{ alertData.clusterName }}</span>
              <span class="res-id">{{ alertData.clusterUuid }}</span>
            </div>
            <div class="resource-row">
              <span class="res-type">项目</span>
              <span class="res-name">{{ alertData.projectName }}</span>
              <span class="res-id">ID: {{ alertData.projectId }}</span>
            </div>
            <div class="resource-row">
              <span class="res-type">工作空间</span>
              <span class="res-name">{{ alertData.workspaceName }}</span>
              <span class="res-id">ID: {{ alertData.workspaceId }}</span>
            </div>
          </div>
        </div>

        <!-- 通知信息 -->
        <div class="section" v-if="alertData.notificationCount > 0">
          <div class="section-title">通知信息</div>
          <div class="notify-info">
            <span class="notify-count"
              >通知次数：<strong>{{ alertData.notificationCount }}</strong></span
            >
            <div class="notify-groups" v-if="notifiedGroupIds.length">
              <span>已通知：</span>
              <ElTag v-for="id in notifiedGroupIds" :key="id" size="small">组 {{ id }}</ElTag>
            </div>
          </div>
        </div>

        <!-- 标签 -->
        <div class="section" v-if="Object.keys(filteredLabels).length">
          <div class="section-title"
            >标签 <span class="badge">{{ Object.keys(filteredLabels).length }}</span></div
          >
          <div class="labels-list">
            <span v-for="(val, key) in filteredLabels" :key="key" class="label-tag">
              <b>{{ key }}:</b> {{ val }}
            </span>
          </div>
        </div>

        <!-- 其他注解 -->
        <div class="section" v-if="Object.keys(otherAnnotations).length">
          <div class="section-title">其他注解</div>
          <div class="anno-list">
            <div v-for="(val, key) in otherAnnotations" :key="key" class="anno-item">
              <div class="anno-key">{{ key }}</div>
              <div class="anno-val">{{ val }}</div>
            </div>
          </div>
        </div>

        <!-- 底部元数据 -->
        <div class="meta-bar">
          <span
            >创建：{{ alertData.createdBy || '-' }} ·
            {{ formatTimestamp(alertData.createdAt) }}</span
          >
          <span
            >更新：{{ alertData.updatedBy || '-' }} ·
            {{ formatTimestamp(alertData.updatedAt) }}</span
          >
        </div>
      </template>

      <ElEmpty v-else-if="!loading && !alertData" description="未找到告警数据" />
    </div>

    <template #footer>
      <ElButton @click="handleClose" :disabled="loading">关闭</ElButton>
      <ElButton type="danger" @click="handleDelete" :loading="deleteLoading" :disabled="loading"
        >删除告警</ElButton
      >
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, computed, watch } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { useClipboard } from '@vueuse/core'
  import {
    getAlertInstanceByIdApi,
    deleteAlertInstanceApi,
    formatTimestamp,
    formatDuration,
    formatLabels,
    formatAnnotations,
    getNotifiedGroupIds,
    AlertInstanceSeverityNames,
    AlertStatusNames,
    type AlertInstance
  } from '@/api/portal/alert'

  interface Props {
    modelValue: boolean
    alertId?: number
  }

  interface Emits {
    (e: 'update:modelValue', value: boolean): void
    (e: 'refresh'): void
  }

  const props = withDefaults(defineProps<Props>(), {
    modelValue: false,
    alertId: undefined
  })

  const emit = defineEmits<Emits>()
  const { copy } = useClipboard()

  const visible = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  const loading = ref(false)
  const deleteLoading = ref(false)
  const alertData = ref<AlertInstance | null>(null)

  const parsedLabels = computed(() => {
    if (!alertData.value?.labels) return {}
    return formatLabels(alertData.value.labels)
  })

  const filteredLabels = computed(() => {
    const labels = parsedLabels.value
    const exclude = ['__name__', 'job', 'instance']
    const result: Record<string, string> = {}
    for (const [k, v] of Object.entries(labels)) {
      if (!exclude.includes(k)) result[k] = v
    }
    return result
  })

  const parsedAnnotations = computed(() => {
    if (!alertData.value?.annotations) return {}
    return formatAnnotations(alertData.value.annotations)
  })

  const hasSummaryInfo = computed(() => {
    const a = parsedAnnotations.value
    return !!(a.summary || a.description || a.value)
  })

  const otherAnnotations = computed(() => {
    const a = parsedAnnotations.value
    const keys = ['summary', 'description', 'value']
    const result: Record<string, string> = {}
    for (const [k, v] of Object.entries(a)) {
      if (!keys.includes(k)) result[k] = v
    }
    return result
  })

  const notifiedGroupIds = computed(() => {
    if (!alertData.value?.notifiedGroups) return []
    return getNotifiedGroupIds(alertData.value.notifiedGroups)
  })

  const getSeverityType = (s: string) =>
    ({ critical: 'danger', warning: 'warning', info: 'info' })[s] || 'info'
  const getStatusType = (s: string) => ({ firing: 'danger', resolved: 'success' })[s] || 'info'

  const copyText = async (text: string) => {
    try {
      await copy(text)
      ElMessage.success('已复制')
    } catch {
      ElMessage.error('复制失败')
    }
  }

  const loadData = async () => {
    if (!props.alertId) return
    loading.value = true
    try {
      alertData.value = await getAlertInstanceByIdApi(props.alertId)
    } finally {
      loading.value = false
    }
  }

  const handleDelete = async () => {
    if (!alertData.value) return
    try {
      await ElMessageBox.confirm(`确定删除告警"${alertData.value.alertName}"？`, '确认删除', {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'error'
      })
      deleteLoading.value = true
      await deleteAlertInstanceApi(alertData.value.id)
      ElMessage.success('删除成功')
      emit('refresh')
      handleClose()
    } catch (e: any) {
      if (e !== 'cancel') {
        /* 全局处理 */
      }
    } finally {
      deleteLoading.value = false
    }
  }

  const handleClose = () => {
    if (loading.value || deleteLoading.value) return
    visible.value = false
    alertData.value = null
  }

  watch(
    () => props.modelValue,
    (val) => {
      if (val && props.alertId) loadData()
    }
  )
</script>

<style lang="scss" scoped>
  .alert-detail {
    max-height: 70vh;
    overflow-y: auto;
  }

  .detail-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    padding-bottom: 16px;
    margin-bottom: 16px;
    border-bottom: 1px solid #ebeef5;

    .header-left {
      h3 {
        margin: 0 0 4px;
        font-size: 18px;
        font-weight: 600;
      }
      .alert-id {
        font-size: 12px;
        color: #909399;
        font-family: var(--ax-font-mono);
      }
    }
    .header-right {
      display: flex;
      gap: 8px;
    }
  }

  .core-info {
    display: grid;
    grid-template-columns: 240px 1fr;
    gap: 16px;
    margin-bottom: 16px;

    @media (max-width: 700px) {
      grid-template-columns: 1fr;
    }
  }

  .info-cards {
    display: flex;
    flex-direction: column;
    gap: 10px;

    .info-card {
      padding: 12px 14px;
      border-radius: 8px;
      border-left: 4px solid #dcdfe6;
      background: #f9fafb;

      .card-label {
        display: block;
        font-size: 12px;
        color: #909399;
        margin-bottom: 4px;
      }
      .card-value {
        font-size: 16px;
        font-weight: 600;
        color: #303133;
      }

      &.danger {
        border-left-color: #f56c6c;
        background: #fef0f0;
        .card-value {
          color: #f56c6c;
        }
      }
      &.warning {
        border-left-color: #e6a23c;
        background: #fdf6ec;
        .card-value {
          color: #e6a23c;
        }
      }
      &.success {
        border-left-color: #67c23a;
        background: #f0f9eb;
        .card-value {
          color: #67c23a;
        }
      }
      &.info {
        border-left-color: #909399;
        background: #f4f4f5;
      }
    }
  }

  .timeline-box {
    background: #f9fafb;
    border: 1px solid #ebeef5;
    border-radius: 8px;
    padding: 14px 16px;

    .box-title {
      font-size: 14px;
      font-weight: 600;
      margin-bottom: 12px;
      padding-bottom: 8px;
      border-bottom: 1px dashed #e4e7ed;
    }

    .timeline-list {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }

    .timeline-row {
      display: flex;
      align-items: center;
      gap: 10px;
      opacity: 0.4;

      &.active {
        opacity: 1;
      }

      .dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        background: #c0c4cc;
        flex-shrink: 0;
      }

      &.success .dot {
        background: #67c23a;
      }

      .label {
        font-size: 12px;
        color: #909399;
        width: 60px;
      }
      .time {
        font-size: 13px;
        color: #303133;
        font-weight: 500;
      }
    }
  }

  .section {
    background: #fff;
    border: 1px solid #ebeef5;
    border-radius: 8px;
    padding: 14px 16px;
    margin-bottom: 12px;

    .section-title {
      font-size: 14px;
      font-weight: 600;
      margin-bottom: 12px;
      padding-bottom: 8px;
      border-bottom: 1px dashed #e4e7ed;
      display: flex;
      align-items: center;
      gap: 8px;

      .badge {
        font-size: 11px;
        padding: 2px 8px;
        background: #ecf5ff;
        color: #409eff;
        border-radius: 10px;
      }
    }
  }

  .summary-items {
    .summary-item {
      padding: 10px 12px;
      background: #f9fafb;
      border-radius: 6px;
      border-left: 3px solid #409eff;
      margin-bottom: 10px;

      &:last-child {
        margin-bottom: 0;
      }
      &.highlight {
        background: #fffbeb;
        border-left-color: #e6a23c;
      }

      .item-label {
        font-size: 12px;
        color: #909399;
        margin-bottom: 4px;
      }
      .item-value {
        font-size: 14px;
        color: #303133;
        line-height: 1.5;
      }
      .item-value.trigger {
        font-size: 18px;
        font-weight: 700;
        color: #e6a23c;
      }
    }
  }

  .info-rows {
    .info-row {
      display: flex;
      align-items: flex-start;
      padding: 10px 0;
      border-bottom: 1px solid #f2f3f5;

      &:last-child {
        border-bottom: none;
      }

      .row-label {
        width: 80px;
        font-size: 13px;
        color: #909399;
        flex-shrink: 0;
      }
      .row-value {
        flex: 1;
        font-size: 13px;
        color: #303133;
        word-break: break-all;
        font-family: var(--ax-font-mono);

        &.copy {
          cursor: pointer;
          padding: 4px 8px;
          background: #f5f7fa;
          border-radius: 4px;
          &:hover {
            background: #ecf5ff;
          }
        }
      }
      .row-link {
        flex: 1;
        font-size: 13px;
        color: #409eff;
        word-break: break-all;
        text-decoration: none;
        &:hover {
          text-decoration: underline;
        }
      }
    }
  }

  .resource-list {
    .resource-row {
      display: flex;
      align-items: center;
      gap: 10px;
      padding: 8px 12px;
      background: #f9fafb;
      border-radius: 6px;
      margin-bottom: 8px;
      font-size: 13px;

      &:last-child {
        margin-bottom: 0;
      }

      .res-type {
        color: #909399;
        width: 60px;
      }
      .res-name {
        font-weight: 500;
        color: #303133;
      }
      .res-id {
        margin-left: auto;
        font-size: 11px;
        color: #c0c4cc;
        font-family: var(--ax-font-mono);
      }
    }
  }

  .notify-info {
    display: flex;
    align-items: center;
    gap: 16px;
    flex-wrap: wrap;

    .notify-count {
      font-size: 13px;
      color: #606266;
      strong {
        font-size: 18px;
        color: #e6a23c;
        margin-left: 4px;
      }
    }

    .notify-groups {
      display: flex;
      align-items: center;
      gap: 6px;
      font-size: 13px;
      color: #909399;
    }
  }

  .labels-list {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;

    .label-tag {
      padding: 4px 10px;
      background: #f5f7fa;
      border: 1px solid #e4e7ed;
      border-radius: 4px;
      font-size: 12px;
      font-family: var(--ax-font-mono);

      b {
        color: #409eff;
        font-weight: 500;
      }
    }
  }

  .anno-list {
    .anno-item {
      padding: 10px 12px;
      background: #f9fafb;
      border-left: 3px solid #409eff;
      border-radius: 4px;
      margin-bottom: 8px;

      &:last-child {
        margin-bottom: 0;
      }

      .anno-key {
        font-size: 12px;
        font-weight: 500;
        color: #409eff;
        margin-bottom: 4px;
        font-family: var(--ax-font-mono);
      }
      .anno-val {
        font-size: 13px;
        color: #303133;
        line-height: 1.5;
        white-space: pre-wrap;
        word-break: break-word;
      }
    }
  }

  .meta-bar {
    display: flex;
    justify-content: space-between;
    padding: 10px 12px;
    background: #f5f7fa;
    border-radius: 6px;
    font-size: 12px;
    color: #909399;
    flex-wrap: wrap;
    gap: 10px;
  }
</style>
