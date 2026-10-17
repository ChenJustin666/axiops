<template>
  <div class="inspection-page">
    <ArtSearchBar
      v-show="showSearchBar"
      :model-value="filters"
      :items="recordSearchItems"
      @search="searchRecords"
      @reset="resetRecordFilters"
    />

    <ElCard
      shadow="never"
      class="table-card art-table-card"
      :style="{ 'margin-top': showSearchBar ? '12px' : '0' }"
    >
      <ArtTableHeader
        v-model:columns="recordColumns"
        v-model:showSearchBar="showSearchBar"
        :loading="loading"
        :showZebra="true"
        :showBorder="true"
        :showHeaderBackground="true"
        :fullClass="'art-page-view'"
        :layout="'search,refresh,size,fullscreen,columns,settings'"
        @refresh="loadRecords"
      >
        <template #left>
          <ElSpace wrap>
            <span class="card-title">巡检报告</span>
            <ElTag effect="plain" type="info">当前 {{ records.length }} 条</ElTag>
          </ElSpace>
        </template>
      </ArtTableHeader>
      <ArtTable
        :loading="loading"
        :data="records"
        :columns="visibleRecordColumns"
        :pagination="{
          current: pagination.page,
          size: pagination.pageSize,
          total: pagination.total
        }"
        :pagination-options="{ align: 'right' }"
        :show-table-header="false"
        empty-text="暂无巡检报告"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      >
        <template #task="{ row }">{{ taskName(row.taskId) }}</template>
        <template #cluster="{ row }">{{
          row.clusterName || clusterName(row.clusterUuid)
        }}</template>
        <template #scope="{ row }">{{ scopeText(row.scopeType) }}</template>
        <template #status="{ row }">
          <ElTag :type="statusTagType(row.status)">{{ statusText(row.status) }}</ElTag>
        </template>
        <template #health="{ row }">
          <ElTag :type="healthTagType(row.healthLevel)">{{ healthText(row.healthLevel) }}</ElTag>
        </template>
        <template #score="{ row }">
          <strong>{{ row.score.toFixed(1) }}</strong>
        </template>
        <template #resultStats="{ row }">
          <div class="result-tags">
            <ElTag size="small" effect="plain">总数 {{ row.totalCount }}</ElTag>
            <ElTag size="small" type="success">成功 {{ row.successCount }}</ElTag>
            <ElTag size="small" type="warning">异常 {{ abnormalCount(row) }}</ElTag>
          </div>
        </template>
        <template #duration="{ row }">{{ formatDuration(row.durationMs) }}</template>
        <template #startedAt="{ row }">{{ formatTime(row.startedAt || row.createdAt) }}</template>
        <template #operation="{ row }">
          <ElSpace class="row-actions">
            <ElButton link type="primary" @click="openReport(row.id)">报告</ElButton>
            <ElButton link type="primary" @click="openLogDialog(row)">日志</ElButton>
            <ElButton
              link
              type="warning"
              :disabled="!canFinishRecord(row)"
              :loading="finishingMap[row.id]"
              @click="finishRecord(row)"
            >
              终止
            </ElButton>
            <ElDropdown trigger="click" @command="(command) => handleRecordCommand(command, row)">
              <ElButton
                link
                type="primary"
                :icon="MoreFilled"
                title="更多操作"
                aria-label="更多操作"
              />
              <template #dropdown>
                <ElDropdownMenu>
                  <ElDropdownItem command="export">导出</ElDropdownItem>
                  <ElDropdownItem command="delete" divided>删除</ElDropdownItem>
                </ElDropdownMenu>
              </template>
            </ElDropdown>
          </ElSpace>
        </template>
      </ArtTable>
    </ElCard>

    <ElDrawer v-model="reportVisible" title="巡检报告" size="72%">
      <div v-loading="reportLoading" class="report-detail">
        <template v-if="currentReport">
          <div class="report-summary">
            <div class="report-score" :class="currentReport.record.healthLevel">
              <strong>{{ currentReport.record.score.toFixed(1) }}</strong>
              <span>{{ healthText(currentReport.record.healthLevel) }}</span>
            </div>
            <ElDescriptions :column="3" border>
              <ElDescriptionsItem label="报告编号">{{
                currentReport.record.recordNo
              }}</ElDescriptionsItem>
              <ElDescriptionsItem label="集群">
                {{
                  currentReport.record.clusterName || clusterName(currentReport.record.clusterUuid)
                }}
              </ElDescriptionsItem>
              <ElDescriptionsItem label="状态">
                {{ statusText(currentReport.record.status) }}
              </ElDescriptionsItem>
              <ElDescriptionsItem label="巡检计划">
                {{ taskName(currentReport.record.taskId) }}
              </ElDescriptionsItem>
              <ElDescriptionsItem label="触发类型">
                {{ currentReport.record.triggerType || '-' }}
              </ElDescriptionsItem>
              <ElDescriptionsItem label="耗时">
                {{ formatDuration(currentReport.record.durationMs) }}
              </ElDescriptionsItem>
              <ElDescriptionsItem label="开始时间">
                {{ formatTime(currentReport.record.startedAt) }}
              </ElDescriptionsItem>
              <ElDescriptionsItem label="结束时间">
                {{ formatTime(currentReport.record.finishedAt) }}
              </ElDescriptionsItem>
              <ElDescriptionsItem label="异常信息">
                {{ currentReport.record.errorMessage || '-' }}
              </ElDescriptionsItem>
            </ElDescriptions>
          </div>

          <div class="report-kpis">
            <div>
              <span>检查总数</span>
              <strong>{{ currentReport.record.totalCount }}</strong>
            </div>
            <div>
              <span>成功数</span>
              <strong>{{ currentReport.record.successCount }}</strong>
            </div>
            <div>
              <span>异常数</span>
              <strong>{{ abnormalCount(currentReport.record) }}</strong>
            </div>
            <div>
              <span>通过率</span>
              <strong>{{ passRate(currentReport.record) }}</strong>
            </div>
          </div>

          <ElDivider content-position="left">结果明细</ElDivider>
          <div class="detail-view-switch">
            <ElRadioGroup v-model="resultView" size="small">
              <ElRadioButton label="detail">执行明细</ElRadioButton>
              <ElRadioButton label="target">目标汇总</ElRadioButton>
              <ElRadioButton label="rule">规则汇总</ElRadioButton>
            </ElRadioGroup>
            <ElSpace>
              <ElTag effect="plain" type="info">明细 {{ resultPagination.total }} 条</ElTag>
              <ElButton
                v-if="canFinishRecord(currentReport.record)"
                size="small"
                type="warning"
                plain
                :loading="finishingMap[currentReport.record.id]"
                @click="finishRecord(currentReport.record)"
              >
                终止
              </ElButton>
              <ElButton
                size="small"
                type="danger"
                plain
                :loading="deletingMap[currentReport.record.id]"
                @click="deleteRecord(currentReport.record)"
              >
                删除
              </ElButton>
            </ElSpace>
          </div>
          <div class="detail-toolbar">
            <ElInput
              v-model="resultFilters.keyword"
              clearable
              :placeholder="resultSearchPlaceholder"
              :prefix-icon="Search"
            />
            <ElSelect v-model="resultFilters.status" clearable placeholder="全部状态">
              <ElOption label="成功" value="success" />
              <ElOption label="警告" value="warning" />
              <ElOption label="严重" value="critical" />
              <ElOption label="失败" value="failed" />
            </ElSelect>
            <ElSelect v-model="resultFilters.severity" clearable placeholder="全部级别">
              <ElOption label="严重" value="critical" />
              <ElOption label="警告" value="warning" />
              <ElOption label="提示" value="info" />
            </ElSelect>
            <ElSelect v-model="resultFilters.sortBy" placeholder="排序方式">
              <ElOption label="风险优先" value="risk" />
              <ElOption label="项目 / 工作空间" value="scope" />
              <ElOption label="分类 / 规则" value="category" />
              <ElOption label="得分升序" value="score" />
            </ElSelect>
          </div>
          <ArtTable
            v-if="resultView === 'detail'"
            :data="filteredReportResults"
            :columns="resultColumns"
            :show-table-header="false"
            :height="520"
            empty-text="暂无巡检结果"
          >
            <template #item="{ row }">
              <div class="meta-block">
                <strong>{{ row.itemName }}</strong>
                <span>{{ row.category || '-' }} · {{ row.itemCode }}</span>
              </div>
            </template>
            <template #target="{ row }">
              <div class="meta-block">
                <strong>{{ targetPrimaryText(row) }}</strong>
                <span>{{ targetSecondaryText(row) }}</span>
              </div>
            </template>
            <template #scope="{ row }">
              <div class="meta-block">
                <strong>{{ detailScopeText(row) }}</strong>
                <span>{{ detailResourceText(row) }}</span>
              </div>
            </template>
            <template #detail="{ row }">
              <div v-if="detailTags(row).length" class="detail-tag-list">
                <ElTag v-for="item in detailTags(row)" :key="item.key" size="small" effect="plain">
                  {{ item.label }}：{{ item.value }}
                </ElTag>
              </div>
              <span v-else>-</span>
            </template>
            <template #status="{ row }">
              <ElTag :type="statusTagType(row.status)">{{ statusText(row.status) }}</ElTag>
            </template>
            <template #severity="{ row }">
              <ElTag :type="healthTagType(row.severity)">{{ row.severity || '-' }}</ElTag>
            </template>
            <template #actual="{ row }">
              <div class="actual-result">
                <strong>{{ actualResultText(row) }}</strong>
                <span v-if="actualResultMeta(row)">{{ actualResultMeta(row) }}</span>
              </div>
            </template>
            <template #suggestion="{ row }">{{ row.suggestion || '-' }}</template>
          </ArtTable>
          <ArtTable
            v-else-if="resultView === 'target'"
            :data="filteredTargetSummaries"
            :columns="targetSummaryColumns"
            :show-table-header="false"
            :height="520"
            empty-text="暂无目标汇总"
          >
            <template #target="{ row }">
              <div class="meta-block">
                <strong>{{ row.targetName }}</strong>
                <span>{{ targetTypeText(row.targetType) }}</span>
              </div>
            </template>
            <template #stats="{ row }">
              <div class="result-tags">
                <ElTag size="small" effect="plain">总数 {{ row.totalCount }}</ElTag>
                <ElTag size="small" type="success">成功 {{ row.successCount }}</ElTag>
                <ElTag size="small" type="warning">异常 {{ row.abnormalCount }}</ElTag>
              </div>
            </template>
            <template #status="{ row }">
              <ElTag :type="statusTagType(row.status)">{{ statusText(row.status) }}</ElTag>
            </template>
            <template #items="{ row }">{{ row.abnormalItems || '-' }}</template>
          </ArtTable>
          <ArtTable
            v-else
            :data="filteredRuleSummaries"
            :columns="ruleSummaryColumns"
            :show-table-header="false"
            :height="520"
            empty-text="暂无规则汇总"
          >
            <template #rule="{ row }">
              <div class="meta-block">
                <strong>{{ row.itemName }}</strong>
                <span>{{ row.category || '-' }} · {{ row.itemCode }}</span>
              </div>
            </template>
            <template #stats="{ row }">
              <div class="result-tags">
                <ElTag size="small" effect="plain">目标 {{ row.totalCount }}</ElTag>
                <ElTag size="small" type="success">成功 {{ row.successCount }}</ElTag>
                <ElTag size="small" type="warning">异常 {{ row.abnormalCount }}</ElTag>
              </div>
            </template>
            <template #status="{ row }">
              <ElTag :type="statusTagType(row.status)">{{ statusText(row.status) }}</ElTag>
            </template>
            <template #targets="{ row }">{{ row.abnormalTargets || '-' }}</template>
          </ArtTable>
          <div class="result-pagination">
            <ElPagination
              v-model:current-page="resultPagination.page"
              v-model:page-size="resultPagination.pageSize"
              background
              layout="total, sizes, prev, pager, next, jumper"
              :page-sizes="[20, 50, 100, 200]"
              :total="resultPagination.total"
              @size-change="handleResultSizeChange"
              @current-change="handleResultCurrentChange"
            />
          </div>
        </template>
      </div>
    </ElDrawer>

    <ElDialog
      v-model="logDialogVisible"
      width="960px"
      class="inspection-log-dialog"
      destroy-on-close
      @closed="handleLogDialogClosed"
    >
      <template #header>
        <div class="terminal-dialog-header">
          <div class="terminal-window-controls">
            <span class="danger"></span>
            <span class="warning"></span>
            <span class="success"></span>
          </div>
          <div class="terminal-dialog-title">
            <strong>巡检执行日志</strong>
            <span>{{ currentLogRecord?.recordNo || '-' }}</span>
          </div>
          <div class="terminal-dialog-tags">
            <ElTag
              v-if="currentLogRecord"
              size="small"
              :type="statusTagType(currentLogRecord.status)"
            >
              {{ statusText(currentLogRecord.status) }}
            </ElTag>
            <ElTag size="small" effect="plain" :type="logStreamTagType">
              {{ logStreamText }}
            </ElTag>
          </div>
        </div>
      </template>
      <div class="terminal-shell">
        <div class="terminal-toolbar">
          <div class="terminal-session">
            <strong>{{ logClusterLabel }}</strong>
            <span>{{ logTaskLabel }} · {{ logScopeLabel }} · {{ logRunModeText }}</span>
          </div>
          <ElButton
            size="small"
            type="primary"
            plain
            :disabled="!currentLogRecord"
            @click="openReportFromLogDialog"
          >
            查看报告
          </ElButton>
        </div>
        <div ref="logContainer" class="terminal-body">
          <div class="terminal-banner">
            <span class="terminal-command"
              >$ inspection follow --record {{ currentLogRecord?.recordNo || '-' }}</span
            >
            <span class="terminal-hint">{{ logStreamHint }}</span>
          </div>
          <div
            v-for="(item, index) in runLogs"
            :key="index"
            class="terminal-row"
            :class="item.level"
          >
            <span>{{ formatTime(item.time) }}</span>
            <strong>{{ logLevelText(item.level) }}</strong>
            <em>{{ item.message }}</em>
          </div>
          <div v-if="!runLogs.length" class="terminal-empty">
            <span>$ waiting for inspection output...</span>
            <em>{{ logEmptyText }}</em>
          </div>
        </div>
        <ElAlert
          v-if="logBannerText"
          class="terminal-alert"
          :title="logBannerText"
          :type="logBannerType"
          show-icon
          :closable="false"
        />
      </div>
      <template #footer>
        <div class="terminal-footer">
          <span>{{ logFooterText }}</span>
          <ElSpace class="terminal-footer-actions">
            <ElButton
              v-if="currentLogRecord && canFinishRecord(currentLogRecord)"
              type="warning"
              plain
              :loading="finishingMap[currentLogRecord.id]"
              @click="finishRecord(currentLogRecord)"
            >
              终止
            </ElButton>
            <ElButton @click="logDialogVisible = false">关闭</ElButton>
            <ElButton type="primary" :disabled="!currentLogRecord" @click="openReportFromLogDialog">
              查看报告
            </ElButton>
          </ElSpace>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
  import { useRoute } from 'vue-router'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { MoreFilled, Search } from '@element-plus/icons-vue'
  import {
    deleteInspectionRecordApi,
    exportInspectionReportApi,
    finishInspectionRecordApi,
    getInspectionRecordLogStreamUrl,
    getInspectionReportApi,
    searchInspectionRecordsApi,
    searchInspectionTasksApi,
    type InspectionRunLog,
    type InspectionRecord,
    type InspectionReportResponse,
    type InspectionResult,
    type InspectionTask
  } from '@/api/manager/inspection'
  import {
    formatTime,
    healthTagType,
    healthText,
    scopeText,
    statusTagType,
    statusText
  } from '../shared/constants'
  import { usePollingClusters } from '../shared/usePollingClusters'

  defineOptions({ name: 'InspectionReportCenter' })

  const route = useRoute()

  const {
    clusters,
    loading: clusterLoading,
    loadClusters
  } = usePollingClusters('inspection-report-cluster')

  const loading = ref(false)
  const reportLoading = ref(false)
  const showSearchBar = ref(true)
  const records = ref<InspectionRecord[]>([])
  const tasks = ref<InspectionTask[]>([])
  const currentReport = ref<InspectionReportResponse | null>(null)
  const currentLogRecord = ref<InspectionRecord | null>(null)
  const reportVisible = ref(false)
  const logDialogVisible = ref(false)
  const resultView = ref<'detail' | 'target' | 'rule'>('detail')
  const exportingMap = ref<Record<number, boolean>>({})
  const finishingMap = ref<Record<number, boolean>>({})
  const deletingMap = ref<Record<number, boolean>>({})
  const runLogs = ref<InspectionRunLog[]>([])
  const logMessage = ref('')
  const logContainer = ref<HTMLElement | null>(null)
  const eventSource = ref<EventSource | null>(null)
  const logStreamState = ref<'idle' | 'connecting' | 'streaming' | 'reconnecting' | 'finished'>(
    'idle'
  )
  let resultFilterTimer: number | undefined
  const pagination = reactive({ page: 1, pageSize: 20, total: 0 })
  const resultPagination = reactive({ page: 1, pageSize: 50, total: 0 })
  const filters = reactive<{
    clusterUuid: string
    taskId?: number
    status: string
    healthLevel: string
    timeRange?: string[]
  }>({
    clusterUuid: '',
    taskId: undefined,
    status: '',
    healthLevel: '',
    timeRange: undefined
  })
  const resultFilters = reactive({
    keyword: '',
    status: '',
    severity: '',
    sortBy: 'risk'
  })
  type InspectionDetail = Record<string, string>
  type InspectionResultRow = InspectionResult & {
    detail: InspectionDetail | null
    detailIndex: number
  }
  interface InspectionSummaryRow {
    key: string
    targetType?: string
    targetName?: string
    itemName?: string
    itemCode?: string
    category?: string
    status: string
    totalCount: number
    successCount: number
    warningCount: number
    criticalCount: number
    failedCount: number
    abnormalCount: number
    abnormalItems?: string
    abnormalTargets?: string
  }

  const recordColumns = ref([
    { prop: 'recordNo', label: '报告编号', minWidth: 190, showOverflowTooltip: true },
    { prop: 'task', label: '巡检计划', minWidth: 180, useSlot: true },
    { prop: 'cluster', label: '集群', minWidth: 180, useSlot: true },
    { prop: 'scope', label: '范围', width: 110, useSlot: true },
    { prop: 'status', label: '状态', width: 110, useSlot: true },
    { prop: 'health', label: '健康度', width: 110, useSlot: true },
    { prop: 'score', label: '得分', width: 90, useSlot: true, sortable: true },
    { prop: 'resultStats', label: '结果统计', minWidth: 230, useSlot: true },
    { prop: 'duration', label: '耗时', width: 110, useSlot: true },
    { prop: 'startedAt', label: '开始时间', width: 180, useSlot: true },
    { prop: 'operation', label: '操作', fixed: 'right' as const, width: 176, useSlot: true }
  ])
  const visibleRecordColumns = computed(() =>
    recordColumns.value.filter((column: any) => column.visible !== false)
  )
  const baseResultColumns = [
    { prop: 'item', label: '分类 / 巡检项', minWidth: 240, useSlot: true },
    { prop: 'target', label: '影响对象', minWidth: 220, useSlot: true },
    { prop: 'scope', label: '归属范围', minWidth: 260, useSlot: true },
    { prop: 'detail', label: '明细标签', minWidth: 280, useSlot: true },
    { prop: 'status', label: '状态', width: 110, useSlot: true },
    { prop: 'severity', label: '级别', width: 100, useSlot: true },
    { prop: 'score', label: '得分', width: 90, sortable: true },
    { prop: 'actual', label: '实际结果', minWidth: 280, useSlot: true, showOverflowTooltip: true },
    {
      prop: 'suggestion',
      label: '处理建议',
      minWidth: 280,
      useSlot: true,
      showOverflowTooltip: true
    }
  ]
  const resultColumns = computed(() => {
    const rows = filteredReportResults.value
    return baseResultColumns.filter((column) => {
      switch (column.prop) {
        case 'scope':
          return (
            hasAnyResultValue(rows, detailScopeText) || hasAnyResultValue(rows, detailResourceText)
          )
        case 'detail':
          return hasAnyResultValue(rows, detailTagsText)
        case 'suggestion':
          return hasAnyResultValue(rows, (row) => row.suggestion)
        default:
          return true
      }
    })
  })
  const targetSummaryColumns = [
    { prop: 'target', label: '执行目标', minWidth: 260, useSlot: true },
    { prop: 'status', label: '状态', width: 110, useSlot: true },
    { prop: 'stats', label: '检查统计', minWidth: 240, useSlot: true },
    { prop: 'items', label: '异常巡检项', minWidth: 420, useSlot: true, showOverflowTooltip: true }
  ]
  const ruleSummaryColumns = [
    { prop: 'rule', label: '巡检项', minWidth: 280, useSlot: true },
    { prop: 'status', label: '状态', width: 110, useSlot: true },
    { prop: 'stats', label: '目标统计', minWidth: 240, useSlot: true },
    { prop: 'targets', label: '异常目标', minWidth: 420, useSlot: true, showOverflowTooltip: true }
  ]
  const recordSearchItems = computed(() => [
    {
      label: '集群',
      key: 'clusterUuid',
      type: 'select',
      props: {
        clearable: true,
        filterable: true,
        loading: clusterLoading.value,
        placeholder: '全部集群',
        options: clusters.value.map((item) => ({
          label: item.name,
          value: item.uuid
        }))
      }
    },
    {
      label: '巡检计划',
      key: 'taskId',
      type: 'select',
      props: {
        clearable: true,
        filterable: true,
        options: tasks.value.map((item) => ({
          label: item.name,
          value: item.id
        }))
      }
    },
    {
      label: '执行状态',
      key: 'status',
      type: 'select',
      props: {
        clearable: true,
        options: [
          { label: '成功', value: 'success' },
          { label: '失败', value: 'failed' },
          { label: '执行中', value: 'running' },
          { label: '部分成功', value: 'partial' }
        ]
      }
    },
    {
      label: '健康度',
      key: 'healthLevel',
      type: 'select',
      props: {
        clearable: true,
        options: [
          { label: '健康', value: 'healthy' },
          { label: '警告', value: 'warning' },
          { label: '严重', value: 'critical' },
          { label: '未知', value: 'unknown' }
        ]
      }
    },
    {
      label: '时间',
      key: 'timeRange',
      type: 'datetimerange',
      span: 8,
      props: {
        type: 'datetimerange',
        rangeSeparator: '至',
        startPlaceholder: '开始时间',
        endPlaceholder: '结束时间',
        valueFormat: 'x'
      }
    }
  ])

  const reportResultRows = computed<InspectionResultRow[]>(() => {
    const source = currentReport.value?.results || []
    return source.flatMap((item) => {
      const details = detailList(item.detailJson)
      if (!details.length) {
        return [toResultRow(item, null, 0)]
      }
      return details.map((detail, index) => toResultRow(item, detail, index))
    })
  })

  const filteredReportResults = computed(() => {
    return reportResultRows.value.slice().sort(compareInspectionResult)
  })
  const targetSummaryRows = computed(() =>
    buildSummaryRows(reportResultRows.value, (row) => `${row.targetType}:${targetPrimaryText(row)}`)
  )
  const ruleSummaryRows = computed(() =>
    buildSummaryRows(reportResultRows.value, (row) => `${row.itemCode}:${row.itemId}`, 'rule')
  )
  const filteredTargetSummaries = computed(() => filterSummaryRows(targetSummaryRows.value))
  const filteredRuleSummaries = computed(() => filterSummaryRows(ruleSummaryRows.value))
  const resultSearchPlaceholder = computed(() => {
    if (resultView.value === 'target') return '搜索目标、主机、实例、异常项'
    if (resultView.value === 'rule') return '搜索规则、分类、异常目标'
    return '搜索规则、实例、主机、项目、工作空间、应用'
  })

  const logClusterLabel = computed(() => {
    if (!currentLogRecord.value) return '未选择巡检记录'
    return currentLogRecord.value.clusterName || clusterName(currentLogRecord.value.clusterUuid)
  })
  const logTaskLabel = computed(() =>
    currentLogRecord.value ? taskName(currentLogRecord.value.taskId) : '-'
  )
  const logScopeLabel = computed(() =>
    currentLogRecord.value ? scopeText(currentLogRecord.value.scopeType) : '-'
  )
  const logRunModeText = computed(() =>
    canFinishRecord(currentLogRecord.value) ? '实时巡检会话' : '历史巡检会话'
  )
  const logStreamText = computed(() => {
    switch (logStreamState.value) {
      case 'connecting':
        return '连接中'
      case 'streaming':
        return canFinishRecord(currentLogRecord.value) ? '实时中' : '历史日志'
      case 'reconnecting':
        return '重连中'
      case 'finished':
        return '已结束'
      default:
        return currentLogRecord.value?.status && !canFinishRecord(currentLogRecord.value)
          ? '已结束'
          : '未连接'
    }
  })
  const logStreamTagType = computed(() => {
    switch (logStreamState.value) {
      case 'streaming':
        return 'success'
      case 'reconnecting':
        return 'warning'
      case 'finished':
        return currentLogRecord.value?.status === 'failed' ? 'danger' : 'info'
      default:
        return 'info'
    }
  })
  const logStreamHint = computed(() => {
    if (canFinishRecord(currentLogRecord.value)) {
      if (logStreamState.value === 'reconnecting') {
        return '日志连接短暂波动，系统正在自动重试'
      }
      return '后台任务执行中，日志会实时推送到当前窗口'
    }
    if (runLogs.value.length) {
      return '该任务已结束，当前展示最近一次执行日志'
    }
    return '该任务已执行完成，可直接查看巡检报告'
  })
  const logEmptyText = computed(() =>
    canFinishRecord(currentLogRecord.value)
      ? '等待服务返回实时输出...'
      : '当前任务已结束，如需结果请打开巡检报告'
  )
  const logBannerText = computed(() => {
    if (logMessage.value) return logMessage.value
    if (currentLogRecord.value && !canFinishRecord(currentLogRecord.value)) {
      return buildFinishedMessage(
        currentLogRecord.value.status,
        currentLogRecord.value.errorMessage
      )
    }
    return ''
  })
  const logBannerType = computed(() => {
    if (currentLogRecord.value?.status === 'failed') return 'error'
    if (currentLogRecord.value?.status === 'partial') return 'warning'
    if (canFinishRecord(currentLogRecord.value) && logMessage.value) return 'warning'
    return logBannerText.value ? 'success' : 'info'
  })
  const logFooterText = computed(() => {
    if (canFinishRecord(currentLogRecord.value)) {
      return logStreamState.value === 'reconnecting'
        ? '日志连接正在自动恢复，后台巡检不会中断。'
        : '日志实时推送中，关闭窗口不会影响后台巡检执行。'
    }
    return '任务已执行完成，请查看报告。'
  })

  async function loadTasks() {
    const resp = await searchInspectionTasksApi({
      page: 1,
      pageSize: 200,
      orderField: 'id',
      isAsc: false
    })
    tasks.value = resp.data || []
  }

  async function loadRecords() {
    loading.value = true
    try {
      const [startTime, endTime] = parseTimeRange(filters.timeRange)
      const resp = await searchInspectionRecordsApi({
        page: pagination.page,
        pageSize: pagination.pageSize,
        orderField: 'id',
        isAsc: false,
        clusterUuid: filters.clusterUuid || undefined,
        taskId: filters.taskId,
        status: filters.status || undefined,
        healthLevel: filters.healthLevel || undefined,
        startTime,
        endTime
      })
      records.value = resp.data || []
      pagination.total = resp.total || 0
    } finally {
      loading.value = false
    }
  }

  function searchRecords() {
    pagination.page = 1
    loadRecords()
  }

  function resetRecordFilters() {
    Object.assign(filters, {
      clusterUuid: '',
      taskId: undefined,
      status: '',
      healthLevel: '',
      timeRange: undefined
    })
    searchRecords()
  }

  function handleSizeChange(size: number) {
    pagination.pageSize = size
    pagination.page = 1
    loadRecords()
  }

  function handleCurrentChange(page: number) {
    pagination.page = page
    loadRecords()
  }

  async function openReport(recordId: number) {
    reportVisible.value = true
    resultPagination.page = 1
    currentReport.value = null
    Object.assign(resultFilters, {
      keyword: '',
      status: '',
      severity: '',
      sortBy: 'risk'
    })
    resultView.value = 'detail'
    await loadReport(recordId)
  }

  async function loadReport(recordId: number) {
    reportLoading.value = true
    try {
      const report = await getInspectionReportApi(recordId, {
        page: resultPagination.page,
        pageSize: resultPagination.pageSize,
        keyword: resultFilters.keyword.trim() || undefined,
        status: resultFilters.status || undefined,
        severity: resultFilters.severity || undefined
      })
      currentReport.value = report
      const resultsTotal = Number(report.resultsTotal)
      resultPagination.total = Number.isFinite(resultsTotal) ? resultsTotal : report.results.length
    } finally {
      reportLoading.value = false
    }
  }

  function handleResultSizeChange(size: number) {
    resultPagination.pageSize = size
    resultPagination.page = 1
    reloadCurrentReportPage()
  }

  function handleResultCurrentChange(page: number) {
    resultPagination.page = page
    reloadCurrentReportPage()
  }

  function reloadCurrentReportPage() {
    const recordId = currentReport.value?.record.id
    if (!recordId) return
    void loadReport(recordId)
  }

  function openLogDialog(record: InspectionRecord) {
    currentLogRecord.value = record
    logDialogVisible.value = true
    startLogStream(record)
    void syncCurrentLogRecord(record.id)
  }

  function startLogStream(record: InspectionRecord) {
    stopLogStream()
    runLogs.value = []
    logMessage.value = ''
    logStreamState.value = 'connecting'
    appendLocalLog(
      'info',
      canFinishRecord(record) ? '正在建立实时日志连接...' : '正在同步历史执行日志...'
    )
    const source = new EventSource(getInspectionRecordLogStreamUrl(record.id))
    eventSource.value = source
    source.onopen = () => {
      logStreamState.value = 'streaming'
      if (canFinishRecord(currentLogRecord.value)) {
        logMessage.value = ''
      }
    }
    source.addEventListener('log', (event) => {
      appendRunLog(parseLogEvent((event as MessageEvent).data))
    })
    source.addEventListener('done', async (event) => {
      const payload = parseEventPayload((event as MessageEvent).data)
      updateCurrentLogRecord(payload)
      logMessage.value =
        payload.message ||
        buildFinishedMessage(
          payload.status || currentLogRecord.value?.status,
          payload.errorMessage || currentLogRecord.value?.errorMessage
        )
      logStreamState.value = 'finished'
      appendLocalLog(
        statusToLogLevel(payload.status || currentLogRecord.value?.status),
        logMessage.value
      )
      stopLogStream()
      await loadRecords()
      await syncCurrentLogRecord(record.id)
    })
    source.addEventListener('error', () => {
      if (!eventSource.value || logStreamState.value === 'finished') {
        return
      }
      if (!canFinishRecord(currentLogRecord.value)) {
        logMessage.value = buildFinishedMessage(
          currentLogRecord.value?.status,
          currentLogRecord.value?.errorMessage
        )
        logStreamState.value = 'finished'
        stopLogStream()
        return
      }
      logStreamState.value = 'reconnecting'
      logMessage.value = '执行日志连接异常，系统正在自动重试'
    })
  }

  function stopLogStream() {
    if (eventSource.value) {
      eventSource.value.close()
      eventSource.value = null
    }
  }

  function handleLogDialogClosed() {
    stopLogStream()
    currentLogRecord.value = null
    runLogs.value = []
    logMessage.value = ''
    logStreamState.value = 'idle'
  }

  function appendRunLog(log: InspectionRunLog | null) {
    if (!log) return
    runLogs.value.push(log)
    if (runLogs.value.length > 300) {
      runLogs.value = runLogs.value.slice(-300)
    }
    nextTick(() => {
      const el = logContainer.value
      if (el) {
        el.scrollTop = el.scrollHeight
      }
    })
  }

  function appendLocalLog(level: InspectionRunLog['level'], message: string) {
    appendRunLog({
      time: Math.floor(Date.now() / 1000),
      level: level || 'info',
      message
    })
  }

  function parseLogEvent(raw: string): InspectionRunLog | null {
    const payload = parseEventPayload(raw)
    if (!payload.message) return null
    return {
      time: Number(payload.time) || Math.floor(Date.now() / 1000),
      level: payload.level || 'info',
      message: payload.message
    }
  }

  function parseEventPayload(raw: string): Record<string, any> {
    try {
      const parsed = JSON.parse(raw)
      return parsed && typeof parsed === 'object' ? parsed : {}
    } catch {
      return {}
    }
  }

  async function syncCurrentLogRecord(recordId: number) {
    try {
      const latestReport = await getInspectionReportApi(recordId, { page: 1, pageSize: 1 })
      currentLogRecord.value = latestReport.record
      if (currentReport.value?.record.id === recordId) {
        currentReport.value = {
          ...currentReport.value,
          record: latestReport.record
        }
      }
      if (!canFinishRecord(latestReport.record) && logStreamState.value !== 'finished') {
        logStreamState.value = 'finished'
      }
    } catch {
      // 忽略同步失败，避免影响日志查看主流程
    }
  }

  function updateCurrentLogRecord(payload: Record<string, any>) {
    if (!currentLogRecord.value) return
    currentLogRecord.value = {
      ...currentLogRecord.value,
      status: payload.status || currentLogRecord.value.status,
      healthLevel: payload.healthLevel || currentLogRecord.value.healthLevel,
      finishedAt: Number(payload.finishedAt) || currentLogRecord.value.finishedAt,
      errorMessage: payload.errorMessage || currentLogRecord.value.errorMessage
    }
  }

  function statusToLogLevel(status?: string): InspectionRunLog['level'] {
    if (status === 'failed') return 'error'
    if (status === 'partial') return 'warning'
    return 'info'
  }

  function buildFinishedMessage(status?: string, errorMessage?: string) {
    if (status === 'failed') {
      return errorMessage ? `任务执行失败：${errorMessage}` : '任务执行失败，请查看报告'
    }
    if (status === 'partial') {
      return errorMessage ? `任务执行完成：${errorMessage}` : '任务执行完成，存在异常项，请查看报告'
    }
    return '任务执行完成，请查看报告'
  }

  function openReportFromLogDialog() {
    if (!currentLogRecord.value) return
    const recordId = currentLogRecord.value.id
    logDialogVisible.value = false
    window.setTimeout(() => {
      void openReport(recordId)
    }, 180)
  }

  function handleRecordCommand(command: string, row: InspectionRecord) {
    if (command === 'export') {
      void exportReport(row)
      return
    }
    if (command === 'delete') {
      void deleteRecord(row)
    }
  }

  function canFinishRecord(row?: InspectionRecord | null) {
    return isRunningStatus(row?.status)
  }

  function isRunningStatus(status?: string | null) {
    return (
      String(status || '')
        .trim()
        .toLowerCase() === 'running'
    )
  }

  async function finishRecord(row: InspectionRecord) {
    if (!canFinishRecord(row)) {
      ElMessage.info('该巡检报告当前不是运行中状态，无需终止')
      return
    }
    try {
      await ElMessageBox.confirm(
        '确定终止这条运行中的巡检吗？终止后报告会标记为失败。',
        '终止巡检',
        {
          type: 'warning',
          confirmButtonText: '终止',
          cancelButtonText: '取消'
        }
      )
    } catch {
      return
    }

    finishingMap.value[row.id] = true
    try {
      const resp = await finishInspectionRecordApi(row.id, { message: '手动终止巡检任务' })
      updateRecordSnapshot(resp.record)
      if (currentLogRecord.value?.id === row.id) {
        currentLogRecord.value = resp.record
        logMessage.value = buildFinishedMessage(resp.record.status, resp.record.errorMessage)
        logStreamState.value = 'finished'
        appendLocalLog('warning', '已手动终止巡检任务')
        stopLogStream()
      }
      if (currentReport.value?.record.id === row.id) {
        currentReport.value = {
          ...currentReport.value,
          record: resp.record
        }
      }
      await loadRecords()
      ElMessage.success('已终止巡检')
    } finally {
      delete finishingMap.value[row.id]
    }
  }

  async function deleteRecord(row: InspectionRecord) {
    const message = canFinishRecord(row)
      ? '该报告仍在运行中，删除会硬删除报告、明细和执行日志，并停止后续写入。确定删除吗？'
      : '删除后会硬删除报告、明细和执行日志，无法恢复。确定删除吗？'
    try {
      await ElMessageBox.confirm(message, '删除巡检报告', {
        type: 'warning',
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        confirmButtonClass: 'el-button--danger'
      })
    } catch {
      return
    }

    deletingMap.value[row.id] = true
    try {
      await deleteInspectionRecordApi(row.id)
      if (currentReport.value?.record.id === row.id) {
        currentReport.value = null
        reportVisible.value = false
      }
      if (currentLogRecord.value?.id === row.id) {
        logDialogVisible.value = false
        handleLogDialogClosed()
      }
      if (records.value.length === 1 && pagination.page > 1) {
        pagination.page -= 1
      }
      await loadRecords()
      ElMessage.success('删除成功')
    } finally {
      delete deletingMap.value[row.id]
    }
  }

  function updateRecordSnapshot(record: InspectionRecord) {
    const index = records.value.findIndex((item) => item.id === record.id)
    if (index >= 0) {
      records.value.splice(index, 1, record)
    }
  }

  async function exportReport(row: InspectionRecord) {
    exportingMap.value[row.id] = true
    try {
      const blob = await exportInspectionReportApi(row.id)
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `巡检报告_${row.recordNo || row.id}.xlsx`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)
      ElMessage.success('导出成功')
    } finally {
      exportingMap.value[row.id] = false
    }
  }

  function taskName(id: number) {
    return tasks.value.find((item) => item.id === id)?.name || (id ? `#${id}` : '-')
  }

  function clusterName(uuid: string) {
    if (!uuid) return '全部集群'
    return clusters.value.find((item) => item.uuid === uuid)?.name || uuid
  }

  function formatDuration(ms: number) {
    if (!ms) return '-'
    if (ms < 1000) return `${ms}ms`
    return `${(ms / 1000).toFixed(1)}s`
  }

  function abnormalCount(
    row: Pick<InspectionRecord, 'warningCount' | 'criticalCount' | 'failedCount'>
  ) {
    return (row.warningCount || 0) + (row.criticalCount || 0) + (row.failedCount || 0)
  }

  function passRate(row: Pick<InspectionRecord, 'successCount' | 'totalCount'>) {
    if (!row.totalCount) return '0%'
    return `${Math.round((row.successCount / row.totalCount) * 1000) / 10}%`
  }

  function logLevelText(level?: string) {
    if (level === 'error') return '异常'
    if (level === 'warning') return '警告'
    return '信息'
  }

  function parseTimeRange(range?: string[]) {
    if (!range || range.length !== 2) return [undefined, undefined] as const
    return [Math.floor(Number(range[0]) / 1000), Math.floor(Number(range[1]) / 1000)] as const
  }

  function toResultRow(
    result: InspectionResult,
    detail: InspectionDetail | null,
    detailIndex: number
  ): InspectionResultRow {
    return {
      ...result,
      detail,
      detailIndex
    }
  }

  function detailScopeText(row: InspectionResultRow) {
    const detail = row.detail
    if (!detail) return '-'
    return (
      [detail.projectName, detail.workspaceName || detail.namespace].filter(Boolean).join(' / ') ||
      '-'
    )
  }

  function detailResourceText(row: InspectionResultRow) {
    const detail = row.detail
    if (!detail) return '-'
    const resource = [detail.resourceType, detail.resourceName].filter(Boolean).join(' / ')
    return (
      [detail.applicationName, resource, detail.podName || detail.pod]
        .filter(Boolean)
        .join(' / ') || '-'
    )
  }

  function targetPrimaryText(row: InspectionResultRow) {
    const detail = row.detail || {}
    switch (row.targetType) {
      case 'host':
      case 'node':
        return firstNonEmpty(
          detail.nodeName,
          detail.node,
          detail.hostName,
          detail.host,
          detail.hostname,
          detail.instance,
          detail.name,
          row.targetName
        )
      case 'pod':
        return firstNonEmpty(
          detail.podName,
          detail.pod,
          detail.name,
          detail.resourceName,
          detail.instance,
          row.targetName
        )
      case 'workload':
        return firstNonEmpty(
          detail.resourceName,
          detail.deployment,
          detail.statefulset,
          detail.daemonset,
          detail.job,
          detail.podName,
          detail.instance,
          row.targetName
        )
      case 'prometheus':
      case 'middleware':
      case 'database':
      case 'etcd':
      case 'mysql':
      case 'redis':
      case 'mongodb':
      case 'postgresql':
      case 'kafka':
      case 'rabbitmq':
      case 'zookeeper':
        return firstNonEmpty(
          detail.instance,
          detail.pod,
          detail.podName,
          detail.service,
          detail.job,
          row.targetName
        )
      case 'cluster':
        return firstNonEmpty(
          detail.clusterName,
          currentReport.value?.record.clusterName,
          clusterName(currentReport.value?.record.clusterUuid || ''),
          row.targetName
        )
      default:
        return firstNonEmpty(
          detail.instance,
          detail.resourceName,
          detail.podName,
          detail.name,
          row.targetName
        )
    }
  }

  function targetSecondaryText(row: InspectionResultRow) {
    const detail = row.detail || {}
    const parts = [
      targetTypeText(row.targetType),
      detail.targetKind,
      detail.resourceType,
      detail.metricName,
      detail.job,
      detail.service
    ].filter(Boolean)
    return parts.join(' / ') || '-'
  }

  function actualResultText(row: InspectionResultRow) {
    const valueText = [row.value, row.unit].filter(Boolean).join(' ')
    if (valueText) return `指标值 ${valueText}`
    return row.actual || row.message || '-'
  }

  function actualResultMeta(row: InspectionResultRow) {
    const parts = []
    if (row.expected) parts.push(`期望 ${row.expected}`)
    const actual = String(row.actual || '').trim()
    if (actual && (!row.value || !actual.includes(String(row.value)))) {
      parts.push(actual)
    }
    const tags = detailTags(row)
      .slice(0, 3)
      .map((item) => `${item.label}:${item.value}`)
    parts.push(...tags)
    return parts.join(' / ')
  }

  function detailTags(row: InspectionResultRow) {
    const detail = row.detail
    if (!detail) return []
    const hidden = new Set([
      'clusterUuid',
      'clusterName',
      'projectId',
      'projectUuid',
      'projectName',
      'workspaceId',
      'workspaceName',
      'namespace',
      'applicationId',
      'applicationName',
      'applicationEn',
      'resourceType',
      'resourceName',
      'versionId',
      'version',
      'podName',
      'pod',
      'nodeName',
      'node',
      'hostName',
      'host',
      'hostname',
      'instance',
      'endpoint',
      'name',
      'targetType',
      'targetKind',
      'detailPromql',
      'detailOperator',
      'detailThreshold',
      'detailExpected',
      'conditionSource',
      'hasTargetLabels',
      'prometheusInstance',
      'seriesCount'
    ])
    return Object.entries(detail)
      .filter(([key, value]) => value && !hidden.has(key))
      .map(([key, value]) => ({
        key,
        label: detailLabelText(key),
        value
      }))
  }

  function detailTagsText(row: InspectionResultRow) {
    return detailTags(row)
      .map((item) => `${item.label}:${item.value}`)
      .join(' ')
  }

  function buildSummaryRows(
    rows: InspectionResultRow[],
    keyGetter: (row: InspectionResultRow) => string,
    mode: 'target' | 'rule' = 'target'
  ): InspectionSummaryRow[] {
    const stats = new Map<string, InspectionSummaryRow & { abnormalSet: Set<string> }>()
    for (const row of rows) {
      const key = keyGetter(row)
      let item = stats.get(key)
      if (!item) {
        item = {
          key,
          targetType: row.targetType,
          targetName: targetPrimaryText(row),
          itemName: row.itemName,
          itemCode: row.itemCode,
          category: row.category,
          status: 'success',
          totalCount: 0,
          successCount: 0,
          warningCount: 0,
          criticalCount: 0,
          failedCount: 0,
          abnormalCount: 0,
          abnormalSet: new Set<string>()
        }
        stats.set(key, item)
      }
      item.totalCount += 1
      if (row.status === 'success') item.successCount += 1
      else if (row.status === 'critical') item.criticalCount += 1
      else if (row.status === 'failed') item.failedCount += 1
      else item.warningCount += 1
      item.abnormalCount = item.warningCount + item.criticalCount + item.failedCount
      if (statusWeight(row.status, row.severity) < statusWeight(item.status)) {
        item.status = row.status
      }
      if (row.status !== 'success') {
        item.abnormalSet.add(mode === 'target' ? row.itemName : targetPrimaryText(row))
      }
    }
    return Array.from(stats.values())
      .map((item) => ({
        ...item,
        abnormalItems: Array.from(item.abnormalSet).join('、'),
        abnormalTargets: Array.from(item.abnormalSet).join('、')
      }))
      .sort((left, right) => {
        const risk = statusWeight(left.status) - statusWeight(right.status)
        if (risk !== 0) return risk
        return compareText(
          left.targetName || left.itemName || '',
          right.targetName || right.itemName || ''
        )
      })
  }

  function filterSummaryRows(rows: InspectionSummaryRow[]) {
    return rows
  }

  function compareInspectionResult(left: InspectionResultRow, right: InspectionResultRow) {
    switch (resultFilters.sortBy) {
      case 'scope':
        return compareText(resultSearchScope(left), resultSearchScope(right))
      case 'category':
        return compareText(
          `${left.category}-${left.itemName}`,
          `${right.category}-${right.itemName}`
        )
      case 'score':
        return left.score - right.score
      default:
        return statusWeight(left.status, left.severity) - statusWeight(right.status, right.severity)
    }
  }

  function resultSearchScope(row: InspectionResultRow) {
    return `${detailScopeText(row)}-${detailResourceText(row)}-${targetPrimaryText(row)}`
  }

  function compareText(left: string, right: string) {
    return left.localeCompare(right, 'zh-CN')
  }

  function statusWeight(status?: string, severity?: string) {
    const normalizedStatus = String(status || '')
      .trim()
      .toLowerCase()
    const normalizedSeverity = String(severity || '')
      .trim()
      .toLowerCase()
    if (
      normalizedStatus === 'failed' ||
      normalizedStatus === 'critical' ||
      normalizedSeverity === 'critical'
    ) {
      return 0
    }
    if (normalizedStatus === 'warning' || normalizedSeverity === 'warning') return 1
    if (isRunningStatus(normalizedStatus)) return 2
    if (normalizedStatus === 'success') return 3
    return 4
  }

  function detailList(raw?: string): InspectionDetail[] {
    if (!raw) return []
    try {
      const parsed: unknown = JSON.parse(raw)
      if (Array.isArray(parsed)) {
        return parsed
          .map((item) => normalizeDetail(item))
          .filter((item): item is InspectionDetail => Boolean(item))
      }
      const detail = normalizeDetail(parsed)
      return detail ? [detail] : []
    } catch {
      return []
    }
  }

  function normalizeDetail(value: unknown): InspectionDetail | null {
    if (!value || typeof value !== 'object' || Array.isArray(value)) return null
    return Object.fromEntries(
      Object.entries(value as Record<string, unknown>).map(([key, item]) => [
        key,
        item === undefined || item === null ? '' : String(item)
      ])
    )
  }

  function firstNonEmpty(...values: unknown[]) {
    for (const value of values) {
      const text = String(value ?? '').trim()
      if (text) return text
    }
    return '-'
  }

  function targetTypeText(value?: string) {
    switch (value) {
      case 'cluster':
        return '集群'
      case 'host':
      case 'node':
        return '主机'
      case 'pod':
        return 'Pod'
      case 'workload':
        return '工作负载'
      case 'prometheus':
        return 'Prometheus'
      case 'middleware':
        return '中间件'
      case 'database':
        return '数据库'
      case 'etcd':
        return 'etcd'
      case 'mysql':
        return 'MySQL'
      case 'redis':
        return 'Redis'
      case 'mongodb':
        return 'MongoDB'
      case 'postgresql':
        return 'PostgreSQL'
      case 'kafka':
        return 'Kafka'
      case 'rabbitmq':
        return 'RabbitMQ'
      case 'zookeeper':
        return 'Zookeeper'
      default:
        return value || '-'
    }
  }

  function detailLabelText(key: string) {
    const labelMap: Record<string, string> = {
      status: '状态',
      type: '类型',
      condition: '条件',
      phase: '阶段',
      reason: '原因',
      message: '信息',
      value: '指标值',
      unit: '单位',
      metricName: '指标',
      job: '任务',
      service: '服务',
      endpoint: '端点',
      device: '设备',
      mountpoint: '挂载点',
      container: '容器'
    }
    return labelMap[key] || key
  }

  function hasAnyResultValue(
    rows: InspectionResultRow[],
    getter: (row: InspectionResultRow) => unknown
  ) {
    return rows.some((row) => {
      const value = getter(row)
      return value !== undefined && value !== null && String(value).trim() !== '' && value !== '-'
    })
  }

  watch(
    () => [resultFilters.keyword, resultFilters.status, resultFilters.severity],
    () => {
      if (!currentReport.value) return
      resultPagination.page = 1
      window.clearTimeout(resultFilterTimer)
      resultFilterTimer = window.setTimeout(() => {
        reloadCurrentReportPage()
      }, 300)
    }
  )

  onMounted(async () => {
    await Promise.all([loadClusters(), loadTasks()])
    applyRouteQuery()
    await loadRecords()
    const recordId = Number(route.query.recordId || 0)
    if (recordId > 0) {
      let record = records.value.find((item) => item.id === recordId)
      if (route.query.log === '1') {
        if (!record) {
          const report = await getInspectionReportApi(recordId, { page: 1, pageSize: 1 })
          record = report.record
        }
        if (record) {
          openLogDialog(record)
        }
      } else {
        await openReport(recordId)
      }
    }
  })

  onBeforeUnmount(() => {
    window.clearTimeout(resultFilterTimer)
    stopLogStream()
  })

  function applyRouteQuery() {
    const taskId = Number(route.query.taskId || 0)
    if (taskId > 0) {
      filters.taskId = taskId
    }
    const status = String(route.query.status || '')
    if (status) {
      filters.status = status
    }
  }
</script>

<style scoped lang="scss">
  @use '../shared/polling-ui' as *;

  :deep(.inspection-log-dialog .el-dialog__header) {
    padding: 16px 18px 0;
    margin-right: 0;
  }

  :deep(.inspection-log-dialog .el-dialog__body) {
    padding: 12px 18px 0;
  }

  :deep(.inspection-log-dialog .el-dialog__footer) {
    padding: 14px 18px 18px;
  }

  .result-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .detail-tag-list {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .actual-result {
    display: flex;
    flex-direction: column;
    gap: 4px;

    strong {
      font-weight: 600;
      color: var(--el-text-color-primary);
    }

    span {
      font-size: 12px;
      line-height: 1.4;
      color: var(--el-text-color-secondary);
    }
  }

  .detail-toolbar {
    display: grid;
    grid-template-columns: minmax(260px, 1fr) 132px 132px 150px;
    gap: 10px;
    margin-bottom: 12px;
  }

  .detail-view-switch {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    align-items: center;
    justify-content: space-between;
    margin: -2px 0 10px;
  }

  .result-pagination {
    display: flex;
    justify-content: flex-end;
    margin-top: 12px;
  }

  .report-kpis {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 10px;
    margin: 16px 0 2px;

    div {
      padding: 12px;
      background: var(--el-fill-color-extra-light);
      border: 1px solid var(--el-border-color-extra-light);
      border-radius: 6px;
    }

    span,
    strong {
      display: block;
    }

    span {
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }

    strong {
      margin-top: 6px;
      font-size: 20px;
      color: var(--el-text-color-primary);
    }
  }

  .terminal-dialog-header {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr) auto;
    gap: 14px;
    align-items: center;
  }

  .report-summary {
    display: grid;
    grid-template-columns: 160px minmax(0, 1fr);
    gap: 16px;
    align-items: stretch;
  }

  .terminal-window-controls,
  .terminal-dialog-tags {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .terminal-window-controls span {
    width: 10px;
    height: 10px;
    background: #475569;
    border-radius: 999px;

    &.danger {
      background: #fb7185;
    }

    &.warning {
      background: #fbbf24;
    }

    &.success {
      background: #34d399;
    }
  }

  .terminal-dialog-title {
    min-width: 0;

    strong,
    span {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      font-size: 15px;
      font-weight: 700;
    }

    span {
      margin-top: 4px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .terminal-shell {
    overflow: hidden;
    background: #101318;
    border: 1px solid #252a33;
    border-radius: 6px;
    box-shadow: inset 0 1px 0 rgb(255 255 255 / 2%);
  }

  .terminal-toolbar {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    padding: 10px 12px;
    background: #171b22;
    border-bottom: 1px solid #252a33;
  }

  .terminal-session {
    min-width: 0;

    strong,
    span {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      font-size: 14px;
      color: #f8fafc;
    }

    span {
      margin-top: 4px;
      font-size: 12px;
      color: #94a3b8;
    }
  }

  .terminal-body {
    height: 420px;
    min-height: 260px;
    max-height: 52vh;
    padding: 12px 14px;
    overflow: auto;
    font-family: var(--ax-font-mono);
    font-size: 12px;
    line-height: 1.65;
    color: #d6dde8;
  }

  .terminal-banner {
    display: flex;
    flex-wrap: wrap;
    gap: 8px 16px;
    align-items: center;
    justify-content: space-between;
    padding-bottom: 10px;
    margin-bottom: 12px;
    border-bottom: 1px dashed rgb(148 163 184 / 20%);
  }

  .terminal-command {
    color: #a5f3fc;
  }

  .terminal-hint {
    font-size: 12px;
    color: #94a3b8;
  }

  .terminal-row {
    display: grid;
    grid-template-columns: 164px 52px minmax(0, 1fr);
    gap: 10px;
    padding: 2px 0;

    span {
      color: #768499;
    }

    strong {
      font-weight: 600;
      color: #7dd3fc;
    }

    em {
      font-style: normal;
      color: #e2e8f0;
      word-break: break-word;
    }

    &.warning strong {
      color: #facc15;
    }

    &.error strong {
      color: #fb7185;
    }
  }

  .terminal-empty {
    display: flex;
    flex-direction: column;
    gap: 4px;
    color: #94a3b8;

    span {
      color: #a5f3fc;
    }

    em {
      font-style: normal;
    }
  }

  .terminal-alert {
    border-width: 1px 0 0;
    border-radius: 0;
  }

  .terminal-footer {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;

    span {
      font-size: 13px;
      color: var(--el-text-color-secondary);
    }
  }

  .terminal-footer-actions {
    margin-left: auto;
  }

  .report-score {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 138px;
    background: var(--el-color-success-light-9);
    border-radius: 6px;

    &.warning {
      background: var(--el-color-warning-light-9);
    }

    &.critical {
      background: var(--el-color-danger-light-9);
    }

    strong {
      font-size: 34px;
    }

    span {
      color: var(--el-text-color-secondary);
    }
  }

  @media (width <= 1200px) {
    .report-summary {
      grid-template-columns: 1fr;
    }

    .detail-toolbar,
    .report-kpis {
      grid-template-columns: 1fr 1fr;
    }

    .terminal-dialog-header,
    .terminal-footer {
      display: flex;
      flex-direction: column;
      align-items: flex-end;
    }

    .terminal-row {
      grid-template-columns: 1fr;
      gap: 2px;
      padding: 6px 0;
    }
  }

  @media (width <= 768px) {
    .detail-toolbar,
    .report-kpis {
      grid-template-columns: 1fr;
    }
  }
</style>
