<template>
  <div class="log-center-page">
    <ElCard shadow="never" class="panel-card center-toolbar">
      <div class="toolbar-title">
        <strong>日志告警中心</strong>
        <span v-if="selectedClusterUuid">
          {{ currentClusterName }} · 默认后端 {{ backendTypeText(clusterDefaultBackend) }}
        </span>
        <span v-else>请选择集群</span>
      </div>
      <div class="panel-actions">
        <ElSelect
          v-model="selectedClusterId"
          placeholder="请选择集群"
          filterable
          clearable
          class="cluster-select"
          :loading="clusterLoading"
          @change="handleClusterChange"
        >
          <ElOption v-for="item in clusters" :key="item.id" :label="item.name" :value="item.id" />
        </ElSelect>
        <ElButton :icon="Refresh" :loading="loading" @click="refreshData">刷新</ElButton>
      </div>
    </ElCard>

    <ElCard v-if="!selectedClusterUuid" shadow="never">
      <ElEmpty description="请先选择集群，再查看该集群的日志告警中心。" />
    </ElCard>

    <template v-else>
      <ArtSearchBar
        v-show="showSearchBar"
        :model-value="filters"
        :items="eventSearchItems"
        @search="applyFilters"
        @reset="resetFilters"
      />

      <ElCard
        shadow="never"
        class="art-table-card"
        :style="{ 'margin-top': showSearchBar ? '12px' : '0' }"
      >
        <ArtTableHeader
          v-model:columns="eventColumns"
          v-model:showSearchBar="showSearchBar"
          :loading="loading"
          :showZebra="true"
          :showBorder="true"
          :showHeaderBackground="true"
          :fullClass="'art-page-view'"
          :layout="'search,refresh,size,fullscreen,columns,settings'"
          @refresh="refreshData"
        >
          <template #left>
            <ElSpace wrap>
              <div>
                <div class="card-header">事件列表</div>
                <div class="card-subtitle">
                  展示集群、项目、命名空间、应用、Pod、原文样本和人工处置记录。
                </div>
              </div>
              <ElTag effect="plain" type="info">当前 {{ pagination.total }} 条</ElTag>
              <ElButton text type="warning" @click="quickFilter('failed')">仅看失败</ElButton>
              <ElButton text type="danger" @click="quickFilter('dead')">仅看死信</ElButton>
              <ElButton text type="primary" @click="goDashboard">返回大盘</ElButton>
            </ElSpace>
          </template>
        </ArtTableHeader>
        <ArtTable
          :data="displayEvents"
          :loading="loading"
          :columns="visibleEventColumns"
          :pagination="{
            current: pagination.page,
            size: pagination.pageSize,
            total: pagination.total
          }"
          :pagination-options="{
            align: 'right',
            pageSizes: [10, 20, 50],
            layout: 'total, sizes, prev, pager, next'
          }"
          :show-table-header="false"
          empty-text="当前筛选条件下没有事件。"
          :row-class-name="eventRowClassName"
          @pagination:size-change="handlePageSizeChange"
          @pagination:current-change="handleCurrentChange"
        >
          <template #rule="{ row }">
            <div class="meta-block">
              <div class="strong">{{ row.ruleName }}</div>
              <div class="muted">{{ backendTypeText(row.backendType) }}</div>
            </div>
          </template>
          <template #cluster="{ row }">
            <div class="meta-block">
              <div class="strong">{{ row.clusterName || row.clusterUuid }}</div>
              <div class="muted">{{ row.projectName || '-' }}</div>
            </div>
          </template>
          <template #scope="{ row }">
            <div class="meta-block">
              <div class="strong">{{ row.namespace || '-' }}</div>
              <div class="muted">
                {{ [row.podName, row.containerName, row.host].filter(Boolean).join(' / ') || '-' }}
              </div>
            </div>
          </template>
          <template #sample="{ row }">
            <div class="sample-preview">
              <div class="sample-time">{{ formatTime(row.sampleTimestamp || row.createdAt) }}</div>
              <div class="sample-message">{{ row.sampleMessage || '-' }}</div>
            </div>
          </template>
          <template #eventStatus="{ row }">
            <ElTag :type="eventStatusTagType(row.eventStatus)">{{ row.eventStatus }}</ElTag>
          </template>
          <template #notifyStatus="{ row }">
            <ElTag :type="notifyStatusTagType(row.notifyStatus)">{{ row.notifyStatus }}</ElTag>
          </template>
          <template #reason="{ row }">
            {{ row.closeReason || row.manualReason || row.notifyError || '-' }}
          </template>
          <template #operation="{ row }">
            <ElSpace class="row-actions" wrap>
              <ElButton link type="primary" @click="openDetail(row)">详情</ElButton>
              <ElButton link type="primary" @click="jumpToSearch(row)">查看原文</ElButton>
              <ElButton link type="primary" @click="fillReason(row)">补原因</ElButton>
              <ElButton
                link
                type="warning"
                :disabled="row.eventStatus !== 'firing'"
                @click="closeEvent(row)"
              >
                手动关闭
              </ElButton>
              <ElButton
                link
                type="primary"
                :disabled="row.notifyStatus !== 'failed' && row.notifyStatus !== 'dead'"
                @click="manualRetry(row.id)"
              >
                补偿重试
              </ElButton>
              <ElPopconfirm title="确认删除这条事件吗？" @confirm="deleteEvent(row.id)">
                <template #reference>
                  <ElButton link type="danger">删除</ElButton>
                </template>
              </ElPopconfirm>
            </ElSpace>
          </template>
        </ArtTable>
      </ElCard>

      <ElDrawer v-model="detailVisible" title="事件详情" size="920px" class="center-detail-drawer">
        <template v-if="currentEvent">
          <div class="event-detail">
            <section class="detail-summary-card">
              <div class="detail-title-block">
                <strong>{{ currentEvent.ruleName }}</strong>
                <span>
                  {{ currentEvent.clusterName || currentEvent.clusterUuid }} ·
                  {{ currentEvent.projectName || '-' }} · {{ currentEvent.namespace || '-' }}
                </span>
              </div>
              <div class="detail-summary">
                <ElTag :type="eventStatusTagType(currentEvent.eventStatus)">
                  {{ currentEvent.eventStatus }}
                </ElTag>
                <ElTag :type="notifyStatusTagType(currentEvent.notifyStatus)">
                  {{ currentEvent.notifyStatus }}
                </ElTag>
                <ElTag effect="plain">{{ backendTypeText(currentEvent.backendType) }}</ElTag>
              </div>
            </section>

            <div class="detail-toolbar">
              <ElButton type="primary" @click="jumpToSearch(currentEvent)">查看原文</ElButton>
              <ElButton @click="copyText(currentEvent.sampleMessage || '')">复制样本</ElButton>
              <ElButton @click="copyText(getRawDisplayText(currentEvent))">复制原文</ElButton>
              <ElButton @click="fillReason(currentEvent)">补原因</ElButton>
              <ElButton
                type="warning"
                :disabled="currentEvent.eventStatus !== 'firing'"
                @click="closeEvent(currentEvent)"
              >
                手动关闭
              </ElButton>
            </div>

            <section class="detail-section">
              <div class="detail-section-head">
                <span>事件上下文</span>
              </div>
              <div class="detail-info-grid">
                <div class="detail-info-item">
                  <span>集群</span>
                  <strong>{{ currentEvent.clusterName || currentEvent.clusterUuid }}</strong>
                </div>
                <div class="detail-info-item">
                  <span>项目</span>
                  <strong>{{ currentEvent.projectName || '-' }}</strong>
                </div>
                <div class="detail-info-item">
                  <span>工作空间</span>
                  <strong>{{ currentEvent.workspaceName || '-' }}</strong>
                </div>
                <div class="detail-info-item">
                  <span>命名空间</span>
                  <strong>{{ currentEvent.namespace || '-' }}</strong>
                </div>
                <div class="detail-info-item wide">
                  <span>Pod / 容器 / 节点</span>
                  <strong>
                    {{
                      [currentEvent.podName, currentEvent.containerName, currentEvent.host]
                        .filter(Boolean)
                        .join(' / ') || '-'
                    }}
                  </strong>
                </div>
                <div class="detail-info-item">
                  <span>样本时间</span>
                  <strong>{{
                    formatTime(currentEvent.sampleTimestamp || currentEvent.createdAt)
                  }}</strong>
                </div>
                <div class="detail-info-item">
                  <span>关闭原因</span>
                  <strong>{{ currentEvent.closeReason || '-' }}</strong>
                </div>
                <div class="detail-info-item">
                  <span>人工原因</span>
                  <strong>{{ currentEvent.manualReason || '-' }}</strong>
                </div>
              </div>
            </section>

            <section class="detail-section">
              <div class="detail-section-head">
                <span>规则定义</span>
              </div>
              <div class="detail-info-grid">
                <div class="detail-info-item wide">
                  <span>规则描述</span>
                  <strong>{{ currentRule?.description || '-' }}</strong>
                </div>
                <div class="detail-info-item wide">
                  <span>查询内容</span>
                  <div class="detail-value-row">
                    <strong>{{ currentRule?.queryText || '-' }}</strong>
                    <ElButton
                      v-if="currentRule?.queryText"
                      text
                      size="small"
                      @click="copyText(currentRule?.queryText || '')"
                    >
                      复制
                    </ElButton>
                  </div>
                </div>
                <div class="detail-info-item">
                  <span>条件类型</span>
                  <strong>{{ conditionTypeText(currentRule?.conditionType || '') }}</strong>
                </div>
                <div class="detail-info-item">
                  <span>阈值 / 窗口</span>
                  <strong
                    >{{ currentRule?.threshold || '-' }} / {{ currentRule?.window || '-' }}</strong
                  >
                </div>
                <div class="detail-info-item wide">
                  <span>评估策略</span>
                  <strong>
                    评估 {{ currentRule?.evalInterval || '-' }} · 持续
                    {{ currentRule?.forDuration || '-' }} · 静默
                    {{ currentRule?.silenceWindow || '-' }}
                  </strong>
                </div>
              </div>
            </section>

            <section v-if="shouldShowSampleBlock(currentEvent)" class="detail-section">
              <div class="detail-section-head">
                <span>命中样本</span>
              </div>
              <pre class="detail-pre">{{ currentEvent.sampleMessage || '-' }}</pre>
            </section>

            <section class="detail-section raw-section">
              <div class="detail-section-head">
                <span>日志原文</span>
              </div>
              <pre class="detail-pre raw">{{ getRawDisplayText(currentEvent) }}</pre>
            </section>

            <section
              v-if="Object.keys(currentEvent.sampleLabels || {}).length > 0"
              class="detail-section"
            >
              <div class="detail-section-head">
                <span>日志标签</span>
              </div>
              <div class="label-group">
                <ElTag
                  v-for="(value, key) in currentEvent.sampleLabels"
                  :key="key"
                  size="small"
                  effect="plain"
                >
                  {{ key }}: {{ value }}
                </ElTag>
              </div>
            </section>
          </div>
        </template>
      </ElDrawer>
    </template>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import dayjs from 'dayjs'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { Refresh } from '@element-plus/icons-vue'
  import {
    closeLogAlertEventApi,
    deleteLogAlertEventApi,
    getLogAlertRuleApi,
    getLogAlertEngineStatsApi,
    getLogClusterConfigApi,
    manualRetryLogAlertEventApi,
    searchLogAlertEventsApi,
    updateLogAlertEventReasonApi,
    type LogAlertEvent,
    type LogAlertRule,
    type LogClusterConfigResponse
  } from '@/api/manager/log'
  import { useUserStore } from '@/store/modules/user'
  import { useLogClusters } from '../shared/useLogClusters'

  defineOptions({ name: 'ClusterLogCenterPage' })

  const router = useRouter()
  const userStore = useUserStore()
  const {
    clusters,
    loading: clusterLoading,
    selectedClusterId,
    selectedClusterUuid,
    loadClusters,
    setCluster
  } = useLogClusters('cluster-log-center-cluster')

  const loading = ref(false)
  const showSearchBar = ref(true)
  const events = ref<LogAlertEvent[]>([])
  const detailVisible = ref(false)
  const currentEvent = ref<LogAlertEvent | null>(null)
  const currentRule = ref<LogAlertRule | null>(null)
  const clusterDefaultBackend = ref('')
  const ruleCache = reactive<Record<number, LogAlertRule>>({})

  const config = reactive<LogClusterConfigResponse>({
    clusterUuid: '',
    defaultBackend: '',
    backends: []
  })
  const stats = reactive({
    clusterUuid: '',
    totalRules: 0,
    enabledRules: 0,
    disabledRules: 0,
    successRules: 0,
    failedRules: 0,
    pendingRules: 0,
    activeFiringEvents: 0,
    failedNotifyEvents: 0,
    deadNotifyEvents: 0,
    criticalRules: 0,
    warningRules: 0,
    infoRules: 0,
    hotRules: 0,
    warmRules: 0,
    coldRules: 0
  })

  const filters = reactive({
    eventStatus: '',
    notifyStatus: '',
    severity: '',
    keyword: ''
  })
  const eventColumns = ref([
    { prop: 'rule', label: '规则 / 后端', minWidth: 220, useSlot: true },
    { prop: 'cluster', label: '集群 / 项目', minWidth: 220, useSlot: true },
    { prop: 'scope', label: '范围', minWidth: 240, useSlot: true },
    { prop: 'sample', label: '样本', minWidth: 320, useSlot: true, showOverflowTooltip: true },
    { prop: 'eventStatus', label: '状态', width: 120, useSlot: true },
    { prop: 'notifyStatus', label: '通知', width: 120, useSlot: true },
    { prop: 'reason', label: '原因', minWidth: 220, useSlot: true, showOverflowTooltip: true },
    { prop: 'operation', label: '操作', fixed: 'right' as const, width: 320, useSlot: true }
  ])
  const visibleEventColumns = computed(() =>
    eventColumns.value.filter((column: any) => column.visible !== false)
  )

  const pagination = reactive({
    page: 1,
    pageSize: 10,
    total: 0
  })

  const currentClusterName = computed(() => {
    const current = clusters.value.find((item) => item.id === selectedClusterId.value)
    return current?.name || '未选择集群'
  })

  const isSuperAdminUser = computed(() =>
    Array.isArray(userStore.info?.roles)
      ? userStore.info.roles.some((item) => String(item).toLowerCase() === 'super_admin')
      : false
  )

  const displayEvents = computed(() => {
    const keyword = filters.keyword.trim().toLowerCase()
    if (!keyword) return events.value
    return events.value.filter((item) =>
      [
        item.ruleName,
        item.clusterName,
        item.projectName,
        item.namespace,
        item.application,
        item.resourceName,
        item.podName,
        item.containerName,
        item.host,
        item.sampleMessage,
        item.sampleRaw,
        item.closeReason,
        item.manualReason
      ]
        .join(' ')
        .toLowerCase()
        .includes(keyword)
    )
  })
  const eventSearchItems = computed(() => [
    {
      label: '事件状态',
      key: 'eventStatus',
      type: 'select',
      props: {
        clearable: true,
        options: [
          { label: 'firing', value: 'firing' },
          { label: 'resolved', value: 'resolved' },
          { label: 'closed', value: 'closed' }
        ]
      }
    },
    {
      label: '通知状态',
      key: 'notifyStatus',
      type: 'select',
      props: {
        clearable: true,
        options: [
          { label: 'success', value: 'success' },
          { label: 'failed', value: 'failed' },
          { label: 'dead', value: 'dead' },
          { label: 'pending', value: 'pending' }
        ]
      }
    },
    {
      label: '级别',
      key: 'severity',
      type: 'select',
      props: {
        clearable: true,
        options: [
          { label: 'critical', value: 'critical' },
          { label: 'warning', value: 'warning' },
          { label: 'info', value: 'info' }
        ]
      }
    },
    {
      label: '关键字',
      key: 'keyword',
      type: 'input',
      props: {
        clearable: true,
        placeholder: '规则 / 项目 / Pod / 原文'
      }
    }
  ])

  async function refreshData() {
    if (!selectedClusterUuid.value) return
    loading.value = true
    try {
      const [configResp, statsResp, eventsResp] = await Promise.all([
        getLogClusterConfigApi(selectedClusterUuid.value),
        getLogAlertEngineStatsApi(selectedClusterUuid.value),
        searchLogAlertEventsApi({
          clusterUuid: selectedClusterUuid.value,
          eventStatus: filters.eventStatus || undefined,
          notifyStatus: filters.notifyStatus || undefined,
          page: pagination.page,
          pageSize: pagination.pageSize
        })
      ])
      Object.assign(config, configResp)
      clusterDefaultBackend.value = configResp.defaultBackend || ''
      Object.assign(stats, statsResp)
      events.value = (eventsResp.data || []).filter((item) =>
        filters.severity ? item.severity === filters.severity : true
      )
      pagination.total = eventsResp.total || 0
      void hydrateRuleCache(events.value)
    } finally {
      loading.value = false
    }
  }

  async function hydrateRuleCache(rows: LogAlertEvent[]) {
    const ids = [...new Set(rows.map((item) => item.ruleId).filter(Boolean))]
    const missing = ids.filter((id) => !ruleCache[id])
    if (missing.length === 0) return
    await Promise.all(
      missing.map(async (id) => {
        try {
          ruleCache[id] = await getLogAlertRuleApi(id)
        } catch {
          // ignore
        }
      })
    )
  }

  function handleClusterChange(value: number | null) {
    setCluster(value)
    pagination.page = 1
    refreshData()
  }

  function applyFilters() {
    pagination.page = 1
    refreshData()
  }

  function resetFilters() {
    filters.eventStatus = ''
    filters.notifyStatus = ''
    filters.severity = ''
    filters.keyword = ''
    pagination.page = 1
    refreshData()
  }

  function quickFilter(type: 'failed' | 'dead') {
    filters.notifyStatus = type
    filters.eventStatus = ''
    pagination.page = 1
    refreshData()
  }

  function handlePageSizeChange() {
    pagination.page = 1
    refreshData()
  }

  function handleCurrentChange(page: number) {
    pagination.page = page
    refreshData()
  }

  async function openDetail(row: LogAlertEvent) {
    currentEvent.value = row
    currentRule.value = ruleCache[row.ruleId] || null
    detailVisible.value = true
    if (!currentRule.value) {
      try {
        currentRule.value = await getLogAlertRuleApi(row.ruleId)
        ruleCache[row.ruleId] = currentRule.value
      } catch {
        currentRule.value = null
      }
    }
  }

  async function copyText(value: string) {
    const text = value?.trim()
    if (!text) {
      ElMessage.info('当前内容为空')
      return
    }
    await navigator.clipboard.writeText(text)
    ElMessage.success('复制成功')
  }

  async function manualRetry(id: number) {
    await manualRetryLogAlertEventApi(id)
    ElMessage.success('补偿执行完成')
    await refreshData()
  }

  async function closeEvent(row: LogAlertEvent) {
    const { value } = await ElMessageBox.prompt('请输入手动关闭原因', '手动关闭事件', {
      confirmButtonText: '关闭',
      cancelButtonText: '取消',
      inputType: 'textarea',
      inputPlaceholder: '例如：确认业务已恢复，人工结束本次告警',
      inputValidator: (val) => (val?.trim() ? true : '关闭原因不能为空')
    })
    await closeLogAlertEventApi(row.id, { reason: value })
    ElMessage.success('事件已手动关闭')
    await refreshData()
  }

  async function fillReason(row: LogAlertEvent) {
    const { value } = await ElMessageBox.prompt('请输入补充原因', '补充事件原因', {
      confirmButtonText: '保存',
      cancelButtonText: '取消',
      inputType: 'textarea',
      inputPlaceholder: '例如：已定位到某版本配置错误，待发布修复',
      inputValue: row.manualReason || '',
      inputValidator: (val) => (val?.trim() ? true : '原因不能为空')
    })
    await updateLogAlertEventReasonApi(row.id, { reason: value })
    ElMessage.success('原因已更新')
    await refreshData()
  }

  async function deleteEvent(id: number) {
    await deleteLogAlertEventApi(id)
    ElMessage.success('事件已删除')
    await refreshData()
  }

  async function jumpToSearch(row: LogAlertEvent) {
    if (!isSuperAdminUser.value) {
      ElMessage.warning('当前角色请到 应用中心-日志查询 中手动检查日志')
      return
    }
    const rule = ruleCache[row.ruleId] || currentRule.value || (await loadRuleCache(row.ruleId))
    const searchText = rule?.queryText?.trim() || row.sampleMessage || ''
    router.push({
      path: '/log/search',
      query: {
        fromAlert: '1',
        alertJumpTs: String(Date.now()),
        backendType: row.backendType || clusterDefaultBackend.value,
        clusterUuid: row.clusterUuid,
        projectUuid: row.projectUuid,
        workspaceId: String(row.workspaceId || ''),
        namespace: row.namespace,
        application: row.application,
        resourceName: row.resourceName,
        podName: row.podName,
        containerName: row.containerName,
        host: row.host,
        message: searchText,
        queryText: searchText,
        sampleTimestamp: row.sampleTimestamp || row.createdAt
      }
    })
  }

  async function loadRuleCache(ruleId: number) {
    if (!ruleId) return null
    if (ruleCache[ruleId]) return ruleCache[ruleId]
    try {
      ruleCache[ruleId] = await getLogAlertRuleApi(ruleId)
      return ruleCache[ruleId]
    } catch {
      return null
    }
  }

  function goDashboard() {
    router.push('/log/dashboard')
  }

  function backendTypeText(value?: string) {
    if (!value) return '-'
    if (value === 'elasticsearch' || value === 'es') return 'Elasticsearch'
    if (value === 'loki') return 'Loki'
    return value
  }

  function conditionTypeText(value: string) {
    const mapping: Record<string, string> = {
      count_gt: '命中数大于阈值',
      count_gte: '命中数大于等于阈值',
      count_lt: '命中数小于阈值',
      count_lte: '命中数小于等于阈值',
      match_any: '命中即告警',
      no_data: '无匹配日志',
      flatline: '日志停滞（无任何日志）'
    }
    return mapping[value] || value || '-'
  }

  function getRawDisplayText(row: LogAlertEvent) {
    const raw = row.sampleRaw?.trim()
    if (!raw) return row.sampleMessage || '-'
    if ((raw.startsWith('{') || raw.startsWith('[')) && raw.includes('"message"')) {
      try {
        const parsed = JSON.parse(raw)
        if (typeof parsed?.message === 'string' && parsed.message.trim()) {
          return parsed.message.trim()
        }
        if (typeof parsed?.log === 'string' && parsed.log.trim()) {
          return parsed.log.trim()
        }
      } catch {
        return raw
      }
    }
    return raw
  }

  function shouldShowSampleBlock(row: LogAlertEvent) {
    const sample = row.sampleMessage?.trim()
    const raw = getRawDisplayText(row).trim()
    if (!sample) return false
    return sample !== raw
  }

  function formatTime(value?: number) {
    if (!value) return '-'
    const normalized = value > 1_000_000_000_000 ? value : value * 1000
    return dayjs(normalized).format('YYYY-MM-DD HH:mm:ss')
  }

  function notifyStatusTagType(value: string) {
    if (value === 'success') return 'success'
    if (value === 'failed') return 'danger'
    if (value === 'dead') return 'warning'
    return 'info'
  }

  function eventStatusTagType(value: string) {
    if (value === 'firing') return 'danger'
    if (value === 'resolved') return 'success'
    if (value === 'closed') return 'warning'
    return 'info'
  }

  function eventRowClassName({ row }: { row: LogAlertEvent }) {
    if (row.notifyStatus === 'failed') return 'event-row-failed'
    if (row.notifyStatus === 'dead') return 'event-row-dead'
    if (row.eventStatus === 'firing') return 'event-row-firing'
    return ''
  }

  onMounted(async () => {
    await loadClusters()
    if (selectedClusterUuid.value) {
      await refreshData()
    }
  })
</script>

<style scoped>
  .log-center-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
    min-height: 100%;
    padding: 0;
    background: var(--el-fill-color-extra-light);
  }

  .panel-card,
  .art-table-card {
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    box-shadow: 0 1px 2px rgb(15 23 42 / 3%);
  }

  .center-toolbar :deep(.el-card__body) {
    display: flex;
    gap: 16px;
    align-items: center;
    justify-content: space-between;
    padding: 16px;
  }

  .toolbar-title {
    min-width: 0;
  }

  .toolbar-title strong,
  .toolbar-title span {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .toolbar-title strong {
    font-size: 16px;
    color: var(--el-text-color-primary);
  }

  .toolbar-title span {
    margin-top: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .panel-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    align-items: center;
    justify-content: flex-end;
  }

  .panel-actions :deep(.el-button + .el-button),
  .row-actions :deep(.el-button + .el-button) {
    margin-left: 0;
  }

  .cluster-select {
    width: 280px;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 12px;
  }

  .stat-card {
    padding: 16px;
    background: linear-gradient(180deg, #fff 0%, #f8fbff 100%);
    border: 1px solid #e4e7ed;
    border-radius: 12px;
    box-shadow: 0 14px 30px rgb(15 23 42 / 5%);
  }

  .label {
    font-size: 13px;
    color: #606266;
  }

  .value {
    margin-top: 8px;
    font-size: 26px;
    font-weight: 700;
  }

  .value.danger {
    color: #f56c6c;
  }

  .value.warning {
    color: #e6a23c;
  }

  .center-banner {
    border-radius: 12px;
    box-shadow: 0 10px 24px rgb(15 23 42 / 5%);
  }

  .table-card-header {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
  }

  .card-header {
    font-weight: 600;
  }

  .card-subtitle {
    margin-top: 4px;
    font-size: 12px;
    color: #909399;
  }

  .filter-form {
    display: grid;
    grid-template-columns: repeat(6, minmax(150px, 1fr));
    gap: 10px 12px;
    align-items: end;
    margin-bottom: 0;
  }

  .filter-form :deep(.el-form-item) {
    margin-right: 0;
    margin-bottom: 0;
  }

  .filter-form :deep(.el-form-item__content) {
    width: 100%;
  }

  .filter-form :deep(.el-input),
  .filter-form :deep(.el-select) {
    width: 100% !important;
  }

  :deep(.el-card) {
    border-radius: 8px;
    box-shadow: 0 1px 2px rgb(15 23 42 / 3%);
  }

  .meta-block {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .strong {
    font-weight: 600;
    color: #111827;
  }

  .muted {
    font-size: 12px;
    color: #6b7280;
  }

  .sample-preview {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .sample-time {
    font-size: 12px;
    color: #6b7280;
  }

  .sample-message {
    line-height: 1.6;
    color: #111827;
  }

  .pagination-bar {
    display: flex;
    justify-content: flex-end;
    padding-top: 16px;
  }

  .detail-summary {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .center-detail-drawer :deep(.el-drawer__body) {
    padding: 16px 20px 24px;
    background: var(--el-fill-color-extra-light);
  }

  .event-detail {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .detail-summary-card,
  .detail-section,
  .detail-toolbar {
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    box-shadow: 0 1px 2px rgb(15 23 42 / 3%);
  }

  .detail-summary-card {
    display: flex;
    gap: 16px;
    align-items: flex-start;
    justify-content: space-between;
    padding: 18px;
  }

  .detail-title-block {
    min-width: 0;
  }

  .detail-title-block strong,
  .detail-title-block span {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .detail-title-block strong {
    font-size: 20px;
    line-height: 1.35;
    color: var(--el-text-color-primary);
  }

  .detail-title-block span {
    margin-top: 8px;
    font-size: 13px;
    line-height: 1.6;
    color: var(--el-text-color-secondary);
  }

  .detail-toolbar {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    padding: 12px 14px;
  }

  .detail-toolbar :deep(.el-button + .el-button) {
    margin-left: 0;
  }

  .detail-section {
    padding: 16px;
  }

  .detail-section-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 10px;
    font-size: 14px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  .detail-info-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }

  .detail-info-item {
    min-width: 0;
    padding: 11px 12px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-extra-light);
    border-radius: 8px;
  }

  .detail-info-item.wide {
    grid-column: 1 / -1;
  }

  .detail-info-item span,
  .detail-info-item strong {
    display: block;
  }

  .detail-info-item span {
    margin-bottom: 6px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .detail-info-item strong {
    font-size: 13px;
    font-weight: 600;
    line-height: 1.6;
    color: var(--el-text-color-primary);
    word-break: break-word;
  }

  .detail-value-row {
    display: flex;
    gap: 8px;
    align-items: flex-start;
    justify-content: space-between;
  }

  .detail-pre {
    padding: 12px;
    margin: 0;
    overflow: auto;
    font-size: 13px;
    line-height: 1.6;
    color: var(--el-text-color-primary);
    word-break: break-word;
    white-space: pre-wrap;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-extra-light);
    border-radius: 8px;
  }

  .detail-pre.raw {
    max-height: 320px;
    overflow: auto;
  }

  .label-group {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  :deep(.event-row-failed) {
    background: #fff7ed;
  }

  :deep(.event-row-dead) {
    background: #fef2f2;
  }

  :deep(.event-row-firing) {
    background: #fff7f7;
  }

  @media (width <= 1200px) {
    .center-toolbar :deep(.el-card__body) {
      flex-direction: column;
      align-items: stretch;
    }

    .panel-actions {
      justify-content: flex-start;
    }
  }

  @media (width <= 768px) {
    .cluster-select {
      width: 100%;
    }

    .filter-form {
      grid-template-columns: 1fr;
    }

    .detail-summary-card {
      flex-direction: column;
    }

    .detail-info-grid {
      grid-template-columns: 1fr;
    }

    .detail-value-row {
      flex-direction: column;
      align-items: flex-start;
    }

    .detail-info-item.wide {
      grid-column: auto;
    }
  }
</style>
