<template>
  <section class="archived-history">
    <ElAlert title="项目已删除 · 只读历史" type="info" :closable="false" />
    <ElAlert v-if="error" :title="error" type="error" :closable="false" />
    <ElTabs v-model="tab" @tab-change="resetRecords">
      <ElTabPane label="工作流记录" name="workflows" />
      <ElTabPane label="构建记录" name="builds" />
    </ElTabs>
    <div class="history-toolbar">
      <ElSelect
        v-if="tab === 'workflows'"
        v-model="workflowId"
        placeholder="选择工作流"
        @change="resetRecords"
      >
        <ElOption v-for="item in workflows" :key="item.id" :label="item.name" :value="item.id" />
      </ElSelect>
      <ElButton
        :icon="RefreshCw"
        circle
        title="刷新历史"
        :loading="loading"
        @click="refreshHistory"
      />
    </div>
    <ElTable v-loading="loading" :data="records" empty-text="暂无历史记录">
      <ElTableColumn label="编号" width="90">
        <template #default="{ row }">#{{ row.run_number || row.build_number || row.id }}</template>
      </ElTableColumn>
      <ElTableColumn label="状态" width="120">
        <template #default="{ row }">{{ statusLabel(row.status) }}</template>
      </ElTableColumn>
      <ElTableColumn v-if="tab === 'workflows'" label="环境" width="160">
        <template #default="{ row }">{{
          environments.find((e) => e.id === row.environment_id)?.name ||
          (row.environment_id ? `环境 #${row.environment_id}` : '-')
        }}</template>
      </ElTableColumn>
      <ElTableColumn v-else label="服务" width="160">
        <template #default="{ row }">{{
          services.find((s) => s.id === row.service_id)?.name || `服务 #${row.service_id}`
        }}</template>
      </ElTableColumn>
      <ElTableColumn prop="git_branch" label="分支" min-width="100" />
      <ElTableColumn label="时间" min-width="180">
        <template #default="{ row }">{{
          new Date(row.created_at).toLocaleString('zh-CN')
        }}</template>
      </ElTableColumn>
      <ElTableColumn label="" width="110" fixed="right">
        <template #default="{ row }"
          ><ElButton text type="primary" :icon="FileText" @click="openRecord(row)"
            >查看日志</ElButton
          ></template
        >
      </ElTableColumn>
    </ElTable>
    <ElPagination
      v-model:current-page="page"
      :page-size="20"
      :total="total"
      layout="total, prev, pager, next"
      @current-change="loadRecords"
    />
    <WorkflowRunDrawer
      v-model:visible="runVisible"
      :run-id="runId"
      :environments="environments"
      :services="services"
      read-only
    />
    <ElDialog
      v-model="logsVisible"
      title="历史构建日志"
      width="min(1500px, 96vw)"
      top="3vh"
      destroy-on-close
    >
      <BuildLogViewer
        v-if="logsVisible"
        :text="logText"
        :loading="logsLoading"
        empty-text="此构建没有保存容器日志"
        @refresh="loadLogs"
      />
    </ElDialog>
  </section>
</template>

<script setup lang="ts">
  import { onMounted, onBeforeUnmount, ref } from 'vue'
  import { FileText, RefreshCw } from 'lucide-vue-next'
  import {
    listWorkflowsApi,
    listWorkflowRunsApi,
    listBuildRunsApi,
    getBuildLogsApi,
    listEnvironmentsApi,
    listAllServicesApi,
    type AxiOpsWorkflow,
    type AxiOpsWorkflowRun,
    type AxiOpsBuildRun,
    type AxiOpsEnvironment,
    type AxiOpsService
  } from '@/api/axiops'
  import WorkflowRunDrawer from './WorkflowRunDrawer.vue'
  import BuildLogViewer from '@/components/build-log-viewer/index.vue'

  const props = defineProps<{ projectId: number }>()
  const tab = ref('workflows'),
    workflowId = ref(0),
    page = ref(1),
    total = ref(0)
  const workflows = ref<AxiOpsWorkflow[]>([]),
    environments = ref<AxiOpsEnvironment[]>([])
  const services = ref<AxiOpsService[]>([])
  const records = ref<(AxiOpsWorkflowRun | AxiOpsBuildRun)[]>([])
  const loading = ref(false),
    error = ref('')
  const runVisible = ref(false),
    runId = ref<number | null>(null)
  const logsVisible = ref(false),
    logsLoading = ref(false),
    logText = ref(''),
    buildId = ref(0)
  let request = 0,
    logRequest = 0
  let directoryReady = false,
    mounted = true
  const statusLabel = (status: string) =>
    (
      ({
        succeeded: '成功',
        success: '成功',
        failed: '失败',
        cancelled: '已取消',
        timeout: '超时'
      }) as Record<string, string>
    )[status] || status
  const loadRecords = async () => {
    const current = ++request
    loading.value = true
    error.value = ''
    try {
      const result =
        tab.value === 'builds'
          ? await listBuildRunsApi(props.projectId, { page: page.value, page_size: 20 })
          : workflowId.value
            ? await listWorkflowRunsApi(workflowId.value, { page: page.value, page_size: 20 })
            : { list: [], total: 0 }
      if (current !== request) return
      records.value = result.list || []
      total.value = result.total
    } catch {
      if (current === request) {
        records.value = []
        total.value = 0
        error.value = '历史记录读取失败，请刷新重试'
      }
    } finally {
      if (current === request) loading.value = false
    }
  }
  const resetRecords = () => {
    page.value = 1
    void loadRecords()
  }
  const loadLogs = async () => {
    const current = ++logRequest
    logsLoading.value = true
    try {
      const logs = await getBuildLogsApi(buildId.value)
      if (current === logRequest)
        logText.value = (logs || []).map((item) => item.log_text || '').join('\n')
    } catch {
      if (current === logRequest) logText.value = '日志读取失败，请刷新重试'
    } finally {
      if (current === logRequest) logsLoading.value = false
    }
  }
  const openRecord = (row: AxiOpsWorkflowRun | AxiOpsBuildRun) => {
    if (tab.value === 'workflows') {
      runId.value = row.id
      runVisible.value = true
    } else {
      buildId.value = row.id
      logText.value = ''
      logsVisible.value = true
      void loadLogs()
    }
  }
  const refreshHistory = async () => {
    if (directoryReady) {
      await loadRecords()
      return
    }
    loading.value = true
    error.value = ''
    const results = await Promise.allSettled([
      listWorkflowsApi(props.projectId),
      listEnvironmentsApi(props.projectId),
      listAllServicesApi(props.projectId)
    ])
    if (!mounted) return
    if (results[0].status === 'fulfilled') workflows.value = results[0].value || []
    if (results[1].status === 'fulfilled') environments.value = results[1].value || []
    if (results[2].status === 'fulfilled') services.value = results[2].value || []
    workflowId.value = workflows.value[0]?.id || 0
    if (results.some((result) => result.status === 'rejected')) {
      error.value = '历史目录读取失败，请刷新重试'
      loading.value = false
      return
    }
    directoryReady = true
    await loadRecords()
  }
  onMounted(refreshHistory)
  onBeforeUnmount(() => {
    mounted = false
    request++
    logRequest++
  })
</script>

<style scoped>
  .archived-history {
    padding: 20px 0;
    min-width: 0;
  }
  .archived-history > .el-alert {
    margin-bottom: 16px;
  }
  .history-toolbar {
    display: flex;
    gap: 12px;
    align-items: center;
    margin-bottom: 12px;
  }
  .history-toolbar .el-select {
    width: min(320px, calc(100vw - 100px));
  }
  .el-pagination {
    margin-top: 16px;
    justify-content: flex-end;
  }
</style>
