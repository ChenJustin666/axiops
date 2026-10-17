<template>
  <div class="application-log-search">
    <ElCard class="filter-card" shadow="never">
      <template #header>
        <div class="filter-header">
          <div class="filter-header-main">
            <span class="filter-title">功能选择</span>
            <div class="filter-status">
              <span class="status-chip">命名空间 {{ workspace?.namespace || '-' }}</span>
              <span class="status-chip">模式 {{ resolvedQueryMode.toUpperCase() }}</span>
            </div>
          </div>

          <div class="filter-actions">
            <div class="limit-group">
              <span class="limit-label">返回行数</span>
              <ElSelect v-model="form.maxLines" size="small" class="limit-select">
                <ElOption label="100 行" :value="100" />
                <ElOption label="200 行" :value="200" />
                <ElOption label="500 行" :value="500" />
                <ElOption label="1000 行" :value="1000" />
              </ElSelect>
            </div>
            <ElButton @click="handleReset">重置</ElButton>
            <ElButton
              type="primary"
              :loading="querying"
              :disabled="!canSearch"
              @click="handleSearch"
            >
              查询日志
            </ElButton>
          </div>
        </div>
      </template>

      <ElForm label-position="top" size="small" class="filter-form">
        <ElRow :gutter="12">
          <ElCol :xs="24" :sm="12" :md="6">
            <ElFormItem label="应用">
              <ElSelect
                v-model="form.applicationId"
                placeholder="选择应用"
                clearable
                filterable
                :loading="applicationLoading"
                @change="handleApplicationChange"
              >
                <ElOption
                  v-for="item in applicationOptions"
                  :key="item.id"
                  :label="item.nameCn || item.nameEn"
                  :value="item.id"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>

          <ElCol :xs="24" :sm="12" :md="6">
            <ElFormItem label="版本">
              <ElSelect
                v-model="form.versionId"
                placeholder="选择版本"
                clearable
                filterable
                :loading="versionLoading"
                :disabled="!form.applicationId"
                @change="handleVersionChange"
              >
                <ElOption
                  v-for="item in versionOptions"
                  :key="item.id"
                  :label="item.version || item.resourceName"
                  :value="item.id"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>

          <ElCol :xs="24" :sm="12" :md="12">
            <ElFormItem label="时间范围">
              <ElDatePicker
                v-model="timeRange"
                type="datetimerange"
                format="YYYY-MM-DD HH:mm:ss"
                value-format="x"
                start-placeholder="开始时间"
                end-placeholder="结束时间"
                range-separator="至"
                :shortcuts="timeShortcuts"
                style="width: 100%"
              />
            </ElFormItem>
          </ElCol>

          <ElCol :xs="24" :sm="12" :md="8">
            <ElFormItem label="Pod">
              <ElSelect
                v-model="form.podName"
                placeholder="选择 Pod"
                clearable
                filterable
                :loading="podLoading"
                :disabled="!form.versionId"
                @change="handlePodChange"
              >
                <ElOption v-for="item in podOptions" :key="item" :label="item" :value="item" />
              </ElSelect>
            </ElFormItem>
          </ElCol>

          <ElCol :xs="24" :sm="12" :md="8">
            <ElFormItem label="Container">
              <ElSelect
                v-model="form.containerName"
                placeholder="选择 Container"
                clearable
                filterable
                :loading="containerLoading"
                :disabled="!form.podName"
              >
                <ElOption
                  v-for="item in containerOptions"
                  :key="item"
                  :label="item"
                  :value="item"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>

          <ElCol :xs="24" :sm="24" :md="8">
            <ElFormItem label="日志正文">
              <ElInput
                v-model="form.message"
                placeholder="输入关键字检索日志正文"
                clearable
                @keyup.enter="handleSearch"
              />
            </ElFormItem>
          </ElCol>
        </ElRow>
      </ElForm>

      <div class="filter-hint">
        <span>当前命名空间：{{ workspace?.namespace || '-' }}</span>
        <span v-if="resolvedQueryMode !== 'platform'"
          >查询模式：{{ resolvedQueryMode.toUpperCase() }}</span
        >
        <span>返回行数：{{ form.maxLines }}</span>
        <span v-if="form.versionId && !form.podName && podOptions.length > 0">
          未选 Pod 时会按当前版本关联的 Pod 集合查询
        </span>
      </div>
    </ElCard>

    <ElCard class="result-card" shadow="never">
      <template #header>
        <div class="result-header">
          <div class="result-meta">
            <span class="result-title">查询结果</span>
            <span class="result-count">{{ logs.length }} 条</span>
            <span v-if="selectedLogs.length > 0" class="result-selected">
              已选 {{ selectedLogs.length }} 条
            </span>
            <span v-if="lastQueryAt" class="result-time">最近查询：{{ lastQueryAt }}</span>
            <span class="result-mode">{{ resolvedQueryMode.toUpperCase() }}</span>
            <span class="result-lines">返回 {{ form.maxLines }} 行</span>
            <span class="result-hint">点击正文可查看完整日志</span>
          </div>

          <div class="result-actions">
            <ElButton @click="selectAllRows" :disabled="logs.length === 0">全选</ElButton>
            <ElButton @click="clearSelectedRows" :disabled="selectedLogs.length === 0"
              >清空选择</ElButton
            >
            <ElDropdown
              v-if="capability.export"
              @command="handleExportCommand"
              :disabled="logs.length === 0"
            >
              <ElButton type="primary" plain> 导出日志 </ElButton>
              <template #dropdown>
                <ElDropdownMenu>
                  <ElDropdownItem command="txt-all">导出全部 TXT</ElDropdownItem>
                  <ElDropdownItem command="txt-selected">导出选中 TXT</ElDropdownItem>
                  <ElDropdownItem command="excel-all">导出全部 Excel</ElDropdownItem>
                  <ElDropdownItem command="excel-selected">导出选中 Excel</ElDropdownItem>
                </ElDropdownMenu>
              </template>
            </ElDropdown>
          </div>
        </div>
      </template>

      <div class="result-table-shell">
        <ElTable
          ref="tableRef"
          v-loading="querying"
          :data="logs"
          stripe
          border
          height="100%"
          row-key="id"
          empty-text="暂无日志，设置条件后点击查询"
          @selection-change="handleSelectionChange"
        >
          <ElTableColumn type="selection" width="48" align="center" />

          <ElTableColumn label="时间" width="180" sortable>
            <template #default="{ row }">
              {{ formatTimestamp(row.timestamp) }}
            </template>
          </ElTableColumn>

          <ElTableColumn label="日志内容" min-width="620">
            <template #default="{ row }">
              <div class="message-cell" @click="openLogDetail(row)">
                <div
                  class="message-preview"
                  v-html="highlightMessage(row.message || row.raw || '-')"
                />
              </div>
            </template>
          </ElTableColumn>

          <ElTableColumn label="Pod" min-width="180">
            <template #default="{ row }">
              <span class="meta-chip">
                {{ getRowLabelValue(row.labels, 'pod_name', 'kubernetes.pod_name') || '-' }}
              </span>
            </template>
          </ElTableColumn>

          <ElTableColumn label="Container" min-width="160">
            <template #default="{ row }">
              <span class="meta-chip meta-chip--container">
                {{
                  getRowLabelValue(row.labels, 'container_name', 'kubernetes.container_name') || '-'
                }}
              </span>
            </template>
          </ElTableColumn>
        </ElTable>
      </div>
    </ElCard>

    <ElDialog v-model="detailVisible" title="日志详情" width="920px">
      <div v-if="currentLog" class="log-detail">
        <div class="detail-meta">
          <div class="meta-row">
            <span class="meta-label">时间</span>
            <span>{{ formatTimestamp(currentLog.timestamp) }}</span>
          </div>
          <div class="meta-row">
            <span class="meta-label">Pod</span>
            <span>{{
              getRowLabelValue(currentLog.labels, 'pod_name', 'kubernetes.pod_name') || '-'
            }}</span>
          </div>
          <div class="meta-row">
            <span class="meta-label">Container</span>
            <span>
              {{
                getRowLabelValue(
                  currentLog.labels,
                  'container_name',
                  'kubernetes.container_name'
                ) || '-'
              }}
            </span>
          </div>
        </div>

        <div class="detail-actions">
          <ElButton
            type="primary"
            plain
            @click="copyLogContent(currentLog.message || currentLog.raw || '')"
          >
            复制正文
          </ElButton>
        </div>

        <div
          class="detail-content"
          v-html="highlightMessage(currentLog.message || currentLog.raw || '', true)"
        />
      </div>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref, watch } from 'vue'
  import dayjs from 'dayjs'
  import { ElMessage } from 'element-plus'
  import * as XLSX from 'xlsx'
  import { useProjectStore } from '@/store/modules/project'
  import { getLogQueryModeOptionsApi } from '@/api/manager/cluster'
  import { queryLogsApi, type LogRecord } from '@/api/manager/log'
  import {
    searchApplicationApi,
    searchVersionApi,
    type OnecProjectApplication,
    type OnecProjectVersion
  } from '@/api/workload/application'
  import { getPodContainerListApi, getPodListApi } from '@/api/workload/wiekload'
  import type { ProjectCluster, ProjectWorkspace } from '@/api'

  interface Props {
    application: OnecProjectApplication
    workspace: ProjectWorkspace | null
    cluster: ProjectCluster | null
    refreshTrigger: number
  }

  interface LogItem extends LogRecord {
    id: string
  }

  interface SearchForm {
    applicationId?: number
    versionId?: number
    podName: string
    containerName: string
    message: string
    maxLines: number
  }

  const props = defineProps<Props>()
  const projectStore = useProjectStore()

  const form = reactive<SearchForm>({
    podName: '',
    containerName: '',
    message: '',
    maxLines: 200
  })

  const applicationLoading = ref(false)
  const versionLoading = ref(false)
  const podLoading = ref(false)
  const containerLoading = ref(false)
  const querying = ref(false)
  const detailVisible = ref(false)
  const currentLog = ref<LogItem | null>(null)
  const lastQueryAt = ref('')
  const tableRef = ref()
  const selectedLogs = ref<LogItem[]>([])
  const applicationOptions = ref<OnecProjectApplication[]>([])
  const versionOptions = ref<OnecProjectVersion[]>([])
  const podOptions = ref<string[]>([])
  const containerOptions = ref<string[]>([])
  const logs = ref<LogItem[]>([])
  const capability = ref({ loki: false, es: false, export: false })
  const lastContextKey = ref('')

  const timeRange = ref<[number, number]>([dayjs().subtract(30, 'minute').valueOf(), Date.now()])

  const timeShortcuts = [
    { text: '最近 15 分钟', value: () => [dayjs().subtract(15, 'minute').valueOf(), Date.now()] },
    { text: '最近 1 小时', value: () => [dayjs().subtract(1, 'hour').valueOf(), Date.now()] },
    { text: '最近 6 小时', value: () => [dayjs().subtract(6, 'hour').valueOf(), Date.now()] },
    { text: '最近 24 小时', value: () => [dayjs().subtract(24, 'hour').valueOf(), Date.now()] }
  ]

  const clusterUuid = computed(() => {
    return props.cluster?.clusterUuid || props.cluster?.uuid || ''
  })

  const projectUuid = computed(() => projectStore.selectedProject?.uuid || '')

  const resolvedQueryMode = computed<'platform' | 'loki' | 'es'>(() => {
    if (capability.value.loki) return 'loki'
    if (capability.value.es) return 'es'
    return 'platform'
  })
  const canSearch = computed(() => {
    return (
      !!clusterUuid.value &&
      !!props.workspace?.namespace &&
      !!form.applicationId &&
      !!form.versionId
    )
  })

  const selectedVersion = computed(() => {
    return versionOptions.value.find((item) => item.id === form.versionId)
  })

  function formatTimestamp(timestamp: number) {
    return timestamp ? dayjs(timestamp).format('YYYY-MM-DD HH:mm:ss') : '-'
  }

  function getRowLabelValue(
    labels: Record<string, string>,
    primaryKey: string,
    fallbackKey?: string
  ) {
    return labels?.[primaryKey] || (fallbackKey ? labels?.[fallbackKey] : '') || ''
  }

  function openLogDetail(item: LogItem) {
    currentLog.value = item
    detailVisible.value = true
  }

  async function copyLogContent(text: string) {
    if (!text) {
      ElMessage.warning('暂无可复制内容')
      return
    }
    try {
      await navigator.clipboard.writeText(text)
      ElMessage.success('日志正文已复制')
    } catch {
      ElMessage.error('复制失败')
    }
  }

  function escapeHtml(value: string) {
    return value
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;')
  }

  function highlightMessage(text: string, preserveLineBreak = false) {
    const content = escapeHtml(text || '-')
    const keyword = form.message.trim()
    let html = content
    if (keyword) {
      const escapedKeyword = keyword.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
      html = html.replace(
        new RegExp(escapedKeyword, 'gi'),
        (matched) => `<mark class="log-highlight">${matched}</mark>`
      )
    }
    if (preserveLineBreak) {
      return html.replace(/\n/g, '<br />')
    }
    return html
  }

  function clearResult() {
    logs.value = []
    selectedLogs.value = []
    lastQueryAt.value = ''
    currentLog.value = null
    if (tableRef.value?.clearSelection) {
      tableRef.value.clearSelection()
    }
  }

  function handleSelectionChange(rows: LogItem[]) {
    selectedLogs.value = rows
  }

  function selectAllRows() {
    if (!tableRef.value || logs.value.length === 0) return
    logs.value.forEach((row) => {
      tableRef.value.toggleRowSelection(row, true)
    })
  }

  function clearSelectedRows() {
    if (!tableRef.value) return
    tableRef.value.clearSelection()
    selectedLogs.value = []
  }

  function getExportRows(scope: 'all' | 'selected') {
    if (scope === 'selected') {
      if (selectedLogs.value.length === 0) {
        ElMessage.info('请先勾选需要导出的日志')
        return [] as LogItem[]
      }
      return selectedLogs.value
    }
    if (logs.value.length === 0) {
      ElMessage.info('暂无可导出的日志')
      return [] as LogItem[]
    }
    return logs.value
  }

  function buildExportBaseName() {
    const app = applicationOptions.value.find((item) => item.id === form.applicationId)
    const appName = app?.nameEn || 'application-log'
    return `${appName}_${dayjs().format('YYYYMMDD_HHmmss')}`
  }

  function exportAsTxt(scope: 'all' | 'selected') {
    const rows = getExportRows(scope)
    if (rows.length === 0) return

    const content = rows
      .map((row, index) => {
        return [
          `#${index + 1}`,
          `时间: ${formatTimestamp(row.timestamp)}`,
          `Pod: ${getRowLabelValue(row.labels, 'pod_name', 'kubernetes.pod_name') || '-'}`,
          `Container: ${getRowLabelValue(row.labels, 'container_name', 'kubernetes.container_name') || '-'}`,
          `正文: ${row.message || row.raw || ''}`
        ].join('\n')
      })
      .join('\n\n------------------------------\n\n')

    const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = `${buildExportBaseName()}_${scope === 'selected' ? 'selected' : 'all'}.txt`
    document.body.appendChild(anchor)
    anchor.click()
    document.body.removeChild(anchor)
    URL.revokeObjectURL(url)
    ElMessage.success('TXT 导出成功')
  }

  function exportAsExcel(scope: 'all' | 'selected') {
    const rows = getExportRows(scope)
    if (rows.length === 0) return

    const worksheetRows = rows.map((row, index) => ({
      序号: index + 1,
      时间: formatTimestamp(row.timestamp),
      正文: row.message || row.raw || '',
      Pod: getRowLabelValue(row.labels, 'pod_name', 'kubernetes.pod_name') || '-',
      Container: getRowLabelValue(row.labels, 'container_name', 'kubernetes.container_name') || '-'
    }))

    const worksheet = XLSX.utils.json_to_sheet(worksheetRows)
    worksheet['!cols'] = [{ wch: 6 }, { wch: 22 }, { wch: 100 }, { wch: 28 }, { wch: 24 }]

    const workbook = XLSX.utils.book_new()
    XLSX.utils.book_append_sheet(workbook, worksheet, '容器日志')
    XLSX.writeFile(
      workbook,
      `${buildExportBaseName()}_${scope === 'selected' ? 'selected' : 'all'}.xlsx`
    )
    ElMessage.success('Excel 导出成功')
  }

  function handleExportCommand(command: string) {
    if (!capability.value.export) return
    switch (command) {
      case 'txt-all':
        exportAsTxt('all')
        break
      case 'txt-selected':
        exportAsTxt('selected')
        break
      case 'excel-all':
        exportAsExcel('all')
        break
      case 'excel-selected':
        exportAsExcel('selected')
        break
    }
  }

  async function loadCapability() {
    capability.value = { loki: false, es: false, export: false }
    if (!clusterUuid.value) return
    try {
      const resp = await getLogQueryModeOptionsApi(clusterUuid.value, props.workspace?.namespace)
      capability.value = {
        loki: !!resp?.rawQuery && !!resp?.loki,
        es: !!resp?.rawQuery && !!resp?.elasticsearch,
        export: !!resp?.export
      }
    } catch {
      capability.value = { loki: false, es: false, export: false }
    }
  }

  async function loadApplications() {
    if (!props.workspace?.id) {
      applicationOptions.value = []
      return
    }
    applicationLoading.value = true
    try {
      const resp = await searchApplicationApi({ workspaceId: props.workspace.id })
      applicationOptions.value = resp || []
    } finally {
      applicationLoading.value = false
    }
  }

  async function loadVersions(applicationId?: number) {
    versionOptions.value = []
    if (!applicationId) return
    versionLoading.value = true
    try {
      const resp = await searchVersionApi({ applicationId })
      versionOptions.value = resp || []
    } finally {
      versionLoading.value = false
    }
  }

  async function loadPods(versionId?: number) {
    podOptions.value = []
    containerOptions.value = []
    form.podName = ''
    form.containerName = ''
    if (!versionId) return
    podLoading.value = true
    try {
      const resp = await getPodListApi(versionId)
      podOptions.value = (resp || []).map((item) => item.name).filter(Boolean)
    } finally {
      podLoading.value = false
    }
  }

  async function loadContainers(versionId?: number, podName?: string) {
    containerOptions.value = []
    form.containerName = ''
    if (!versionId || !podName) return
    containerLoading.value = true
    try {
      const resp = await getPodContainerListApi(versionId, { podName })
      containerOptions.value = [
        ...(resp?.initContainers || []).map((item) => item.name),
        ...(resp?.containers || []).map((item) => item.name),
        ...((resp?.ephemeralContainers || []).map((item) => item.name) || [])
      ].filter(Boolean)
    } finally {
      containerLoading.value = false
    }
  }

  async function hydrateFilters(options: { force?: boolean; syncTopApplication?: boolean } = {}) {
    const { force = false, syncTopApplication = false } = options
    const contextKey = `${props.workspace?.id || 0}:${props.application?.id || 0}`
    if (!force && contextKey === lastContextKey.value) {
      return
    }

    lastContextKey.value = contextKey
    await Promise.all([loadCapability(), loadApplications()])

    const matchedApplication = applicationOptions.value.find(
      (item) => item.id === props.application.id
    )
    const canKeepCurrentApplication = applicationOptions.value.some(
      (item) => item.id === form.applicationId
    )
    if (syncTopApplication || !canKeepCurrentApplication) {
      form.applicationId = matchedApplication?.id || applicationOptions.value[0]?.id
    }

    if (form.applicationId) {
      await loadVersions(form.applicationId)
      if (!versionOptions.value.some((item) => item.id === form.versionId)) {
        form.versionId = versionOptions.value[0]?.id
      }
      if (form.versionId) {
        await loadPods(form.versionId)
      }
    }
  }

  function buildPodRegex(pods: string[]) {
    return pods.map((item) => item.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('|')
  }

  function buildQueryExpression() {
    const terms: string[] = []
    const namespace = props.workspace?.namespace || ''
    const pods = form.podName ? [form.podName] : podOptions.value
    const keyword = form.message.trim()

    if (resolvedQueryMode.value === 'loki') {
      const selector = [`log_type="container"`]
      if (namespace) selector.push(`namespace_name="${namespace}"`)
      let expr = `{${selector.join(', ')}}`
      expr += ` | json | cluster_uuid="${clusterUuid.value}"`
      if (projectUuid.value) expr += ` | project_uuid="${projectUuid.value}"`
      if (pods.length > 0) {
        expr += ` | pod_name=~"${buildPodRegex(pods)}"`
      }
      if (form.containerName) {
        expr += ` | container_name="${form.containerName}"`
      }
      if (keyword) {
        expr += ` |= "${keyword.replace(/\\/g, '\\\\').replace(/"/g, '\\"')}"`
      }
      return expr
    }

    terms.push(`cluster_uuid:"${clusterUuid.value}"`)
    terms.push(`log_type:"container"`)
    if (projectUuid.value) {
      terms.push(`project_uuid:"${projectUuid.value}"`)
    }
    if (namespace) {
      terms.push(`namespace_name:"${namespace}"`)
    }
    if (pods.length > 0) {
      const podExpr = pods.map((item) => `pod_name:"${item}"`).join(' OR ')
      terms.push(`(${podExpr})`)
    }
    if (form.containerName) {
      terms.push(`container_name:"${form.containerName}"`)
    }
    if (keyword) {
      terms.push(`message:"${keyword}"`)
    }
    return terms.join(' AND ')
  }

  async function handleSearch() {
    if (!clusterUuid.value || !props.workspace?.namespace) {
      ElMessage.warning('请先在顶部选择集群和工作空间')
      return
    }
    if (!form.applicationId) {
      ElMessage.warning('请选择应用')
      return
    }
    if (!form.versionId) {
      ElMessage.warning('请选择版本')
      return
    }
    if (!form.podName && podOptions.value.length === 0) {
      ElMessage.warning('当前版本下暂无可查询的 Pod')
      return
    }

    querying.value = true
    try {
      const request = {
        sourceType: 'container' as const,
        logType: 'container' as const,
        clusterUuid: clusterUuid.value,
        projectUuid: projectUuid.value || undefined,
        namespace: props.workspace.namespace,
        start: String(timeRange.value[0]),
        end: String(timeRange.value[1]),
        limit: form.maxLines,
        direction: 'backward' as const,
        message: resolvedQueryMode.value === 'platform' ? form.message || undefined : undefined,
        podName: resolvedQueryMode.value === 'platform' ? form.podName || undefined : undefined,
        containerName:
          resolvedQueryMode.value === 'platform' ? form.containerName || undefined : undefined,
        queryMode: resolvedQueryMode.value === 'platform' ? undefined : resolvedQueryMode.value,
        mode: resolvedQueryMode.value === 'platform' ? 'form' : 'code',
        expr: resolvedQueryMode.value === 'platform' ? undefined : buildQueryExpression()
      }

      const resp = await queryLogsApi(request)
      logs.value = (resp?.list || []).map((item, index) => ({
        ...item,
        id: `${item.timestamp}-${index}-${crypto.randomUUID()}`
      }))
      selectedLogs.value = []
      lastQueryAt.value = dayjs().format('HH:mm:ss')
      ElMessage.success(`已获取 ${logs.value.length} 条日志`)
    } catch (error: any) {
      ElMessage.error(error?.message || '查询日志失败')
    } finally {
      querying.value = false
    }
  }

  function handleReset() {
    form.message = ''
    form.maxLines = 200
    timeRange.value = [dayjs().subtract(30, 'minute').valueOf(), Date.now()]
    if (applicationOptions.value.some((item) => item.id === props.application.id)) {
      form.applicationId = props.application.id
    }
    form.versionId = versionOptions.value[0]?.id
    form.podName = ''
    form.containerName = ''
    containerOptions.value = []
    clearResult()
  }

  async function handleApplicationChange() {
    form.versionId = undefined
    form.podName = ''
    form.containerName = ''
    clearResult()
    await loadVersions(form.applicationId)
    form.versionId = versionOptions.value[0]?.id
    await loadPods(form.versionId)
  }

  async function handleVersionChange() {
    clearResult()
    await loadPods(form.versionId)
  }

  async function handlePodChange() {
    clearResult()
    await loadContainers(form.versionId, form.podName)
  }

  watch([() => props.workspace?.id, () => props.application.id], () => {
    void hydrateFilters({ force: true, syncTopApplication: true })
  })

  watch(
    () => props.refreshTrigger,
    () => {
      void hydrateFilters({ force: true, syncTopApplication: false })
    }
  )

  onMounted(async () => {
    await hydrateFilters({ force: true, syncTopApplication: true })
  })
</script>

<style lang="scss" scoped>
  .application-log-search {
    display: flex;
    flex-direction: column;
    gap: 16px;
    padding: 16px;
    min-height: 100%;

    .filter-card,
    .result-card {
      border: 1px solid #e5eaf3;
      border-radius: 16px;
      background: #fff;
      box-shadow: 0 8px 20px rgba(15, 23, 42, 0.04);
    }

    .filter-card {
      overflow: hidden;

      :deep(.el-card__header) {
        padding: 14px 18px;
        border-bottom: 1px solid #eef2f7;
        background: #fbfcfe;
      }

      .filter-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 16px;

        .filter-header-main {
          min-width: 0;
          display: flex;
          align-items: center;
          gap: 14px;
          flex-wrap: wrap;

          .filter-title {
            font-size: 16px;
            font-weight: 700;
            color: #1f2937;
          }

          .filter-status {
            display: flex;
            flex-wrap: wrap;
            gap: 8px;
          }
        }

        .status-chip {
          display: inline-flex;
          align-items: center;
          min-height: 28px;
          padding: 4px 10px;
          border-radius: 999px;
          background: #f3f6fb;
          color: #475467;
          font-size: 12px;
        }

        .filter-actions {
          display: flex;
          align-items: center;
          flex-wrap: wrap;
          gap: 10px;
          flex-shrink: 0;

          .limit-select {
            width: 116px;
          }

          .limit-group {
            display: inline-flex;
            align-items: center;
            gap: 10px;
            padding: 8px 12px;
            border-radius: 12px;
            background: #f8fafc;
            border: 1px solid #e5eaf3;

            .limit-label {
              font-size: 12px;
              font-weight: 600;
              color: #667085;
              white-space: nowrap;
            }
          }
        }
      }

      :deep(.el-card__body) {
        padding: 0;
      }

      .filter-form {
        padding: 14px 18px 2px;
      }

      .filter-hint {
        display: flex;
        flex-wrap: wrap;
        gap: 12px;
        padding: 0 18px 12px;
        color: #667085;
        font-size: 12px;
      }
    }

    .result-card {
      :deep(.el-card__header) {
        padding: 14px 18px;
        border-bottom: 1px solid #eef2f7;
        background: #fbfcfe;
      }

      :deep(.el-card__body) {
        padding: 0 16px 16px;
        overflow: hidden;
      }

      .result-table-shell {
        height: clamp(460px, calc(100vh - 430px), 720px);
        min-height: 460px;
        max-height: 720px;
        overflow: hidden;
      }

      .result-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 16px;

        .result-meta {
          display: flex;
          align-items: center;
          flex-wrap: wrap;
          gap: 12px;

          .result-title {
            font-weight: 700;
            color: #1f2937;
          }

          .result-count,
          .result-time,
          .result-mode,
          .result-selected,
          .result-lines,
          .result-hint {
            color: #667085;
            font-size: 13px;
          }

          .result-mode,
          .result-selected,
          .result-lines {
            padding: 4px 10px;
            border-radius: 999px;
            background: #f3f6fb;
          }
        }

        .result-actions {
          display: flex;
          align-items: center;
          gap: 10px;
          flex-shrink: 0;
        }
      }

      :deep(.el-table) {
        height: 100%;
      }

      :deep(.el-table__inner-wrapper) {
        height: 100%;
      }

      :deep(.el-table__body-wrapper),
      :deep(.el-scrollbar),
      :deep(.el-scrollbar__wrap) {
        height: 100% !important;
        overflow-y: auto !important;
      }

      .message-cell {
        cursor: pointer;
        color: #344054;

        &:hover {
          color: #2563eb;
        }

        .message-preview {
          line-height: 1.7;
          display: -webkit-box;
          -webkit-line-clamp: 3;
          -webkit-box-orient: vertical;
          overflow: hidden;
        }
      }

      .meta-chip {
        display: inline-flex;
        align-items: center;
        min-height: 28px;
        padding: 4px 10px;
        border-radius: 999px;
        background: #f5f7fa;
        color: #475467;
        font-size: 12px;
        line-height: 1.4;

        &.meta-chip--container {
          background: #eef4ff;
          color: #2563eb;
        }
      }
    }

    .log-detail {
      display: flex;
      flex-direction: column;
      gap: 16px;

      .detail-meta {
        display: grid;
        grid-template-columns: repeat(3, minmax(0, 1fr));
        gap: 12px;

        .meta-row {
          padding: 12px 14px;
          background: #f8fafc;
          border: 1px solid #e5eaf3;
          border-radius: 12px;

          .meta-label {
            display: block;
            margin-bottom: 6px;
            font-size: 12px;
            color: #98a2b3;
          }
        }
      }

      .detail-actions {
        display: flex;
        justify-content: flex-end;
      }

      .detail-content {
        max-height: 520px;
        overflow: auto;
        padding: 16px 18px;
        border-radius: 14px;
        background: #0f172a;
        color: #e2e8f0;
        font-family: var(--ax-font-mono);
        font-size: 13px;
        line-height: 1.7;
        border: 1px solid rgba(148, 163, 184, 0.24);
        white-space: normal;
        word-break: break-word;
      }
    }
  }

  :deep(.log-highlight) {
    background: linear-gradient(90deg, rgba(255, 225, 126, 0.9), rgba(255, 201, 71, 0.9));
    color: #7c2d12;
    padding: 0 2px;
    border-radius: 4px;
    font-weight: 600;
  }

  @media (max-width: 1200px) {
    .application-log-search {
      .filter-card {
        .filter-header {
          flex-direction: column;
          align-items: stretch;

          .filter-actions {
            width: 100%;
          }
        }
      }

      .result-card {
        min-height: 460px;

        .result-header {
          flex-direction: column;
          align-items: stretch;
        }
      }
    }
  }

  @media (max-width: 768px) {
    .application-log-search {
      padding: 12px;

      .log-detail {
        .detail-meta {
          grid-template-columns: repeat(1, minmax(0, 1fr));
        }
      }
    }
  }
</style>
