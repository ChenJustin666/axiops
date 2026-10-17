<template>
  <div v-loading="loading" class="workflow-job-page">
    <ProjectNavBar :project-id="projectId" :project="project" active="workflows" compact>
      <template #crumb>
        <button type="button" class="crumb-link" @click="goWorkspace">工作流</button>
        <ChevronRight :size="14" class="crumb-sep" />
        <ArtDeliveryIcon :type="workflow?.delivery_type" :size="14" class="crumb-type" />
        <span class="crumb-current" :title="workflow?.name || ''">{{
          workflow?.name || 'Job'
        }}</span>
        <span class="crumb-meta">{{ workflow?.code || '-' }} · {{ environmentName }}</span>
      </template>
      <template #actions>
        <ElButton
          :icon="Play"
          type="primary"
          :disabled="loading || !workflow"
          @click="openExecuteDialog"
          >执行</ElButton
        >
        <ElButton
          text
          :icon="Settings"
          :disabled="!workflow"
          title="查看或编辑工作流编排"
          @click="designerVisible = true"
          >编排</ElButton
        >
        <ElButton plain :loading="cloning" :disabled="!workflow" @click="cloneWorkflow"
          >克隆</ElButton
        >
      </template>
    </ProjectNavBar>

    <nav class="job-tabs" aria-label="Job视图">
      <button
        v-for="tab in tabs"
        :key="tab.value"
        type="button"
        :class="{ active: activeTab === tab.value }"
        @click="activeTab = tab.value"
        >{{ tab.label }}</button
      >
    </nav>
    <ElAlert v-if="runsError" :title="runsError" type="error" :closable="false" show-icon />

    <main v-if="activeTab === 'runs'" class="job-content ax-panel">
      <div class="panel-head"
        ><strong>执行记录</strong><span>{{ totalRuns }} 次运行</span
        ><ElButton text :icon="RefreshCw" :loading="loadingRuns" @click="loadRuns"
          >刷新</ElButton
        ></div
      >
      <div class="runs-table">
        <div class="run-row run-header"
          ><span>运行</span><span>状态</span><span>耗时</span><span>执行人 / 时间</span
          ><span>分支 / 最新 Commit</span><span>实际目标环境</span><span>操作</span></div
        >
        <div v-for="run in runList" :key="run.id" class="run-row">
          <ElButton
            link
            type="primary"
            class="run-no"
            :title="`记录 ID：${run.id}`"
            @click="openRun(run.id)"
            >#{{ run.run_number || '—' }}</ElButton
          >
          <span
            ><ElTag size="small" effect="plain" :type="statusType(run.status)">{{
              statusLabel(run.status)
            }}</ElTag></span
          >
          <span class="run-dur">{{ duration(run) }}</span>
          <span class="runner"
            ><strong>{{ runner(run) }}</strong
            ><small>{{ formatDate(run.created_at) }}</small></span
          >
          <div class="run-commits">
            <div v-for="commit in run.build_commits || []" :key="commit.build_id">
              <small
                >{{ run.build_commits!.length > 1 ? `${serviceName(commit.service_id)} · ` : ''
                }}{{ commit.branch }}</small
              >
              <BuildCommitLink
                :commit="commit.commit"
                :subject="commit.subject"
                :url="commit.url"
                :author="commit.author"
              />
            </div>
            <span v-if="!run.build_commits?.length">{{
              run.context_snapshot?.branch || '尚无可查看的构建提交'
            }}</span>
          </div>
          <span>{{ runEnvironmentLabel(run) }}</span>
          <span class="run-actions"
            ><ElButton link type="primary" @click="openRun(run.id)">进度与日志</ElButton
            ><ElButton
              v-if="run.environment_id"
              link
              type="primary"
              @click="openRollbackRun(run.id)"
              >回滚…</ElButton
            ></span
          >
        </div>
        <ElEmpty
          v-if="!runList.length && !loadingRuns"
          description="暂无执行记录，点击右上角执行创建 Job"
        />
      </div>
    </main>
    <main v-else-if="activeTab === 'changes'" class="job-content ax-panel">
      <div class="panel-head"
        ><strong>修改记录</strong><span>{{ totalRuns }} 次运行</span
        ><ElButton text :icon="RefreshCw" :loading="loadingRuns" @click="loadRuns"
          >刷新</ElButton
        ></div
      >
      <section v-for="run in runList" :key="run.id" class="change-run">
        <div class="change-run-heading"
          ><ElButton link type="primary" @click="openRun(run.id)"
            >#{{ run.run_number || '—' }} · {{ formatDate(run.created_at) }}</ElButton
          ><ElTag size="small" effect="plain" :type="statusType(run.status)">{{
            statusLabel(run.status)
          }}</ElTag></div
        >
        <div v-for="commit in run.build_commits || []" :key="commit.build_id">
          <div class="change-commit-heading"
            ><small>{{ serviceName(commit.service_id) }} · {{ commit.branch }}</small
            ><BuildCommitLink
              :commit="commit.commit"
              :subject="commit.subject"
              :url="commit.url"
              :author="commit.author"
            /><ElButton link type="primary" @click="toggleChanges(commit.build_id)">{{
              expandedBuilds.includes(commit.build_id) ? '收起' : '展开本次修改'
            }}</ElButton></div
          >
          <BuildChanges
            v-if="expandedBuilds.includes(commit.build_id)"
            :build-id="commit.build_id"
          />
        </div>
        <p v-if="!run.build_commits?.length"
          >此次运行暂无可查看的构建提交；可进入运行详情查看执行状态。</p
        >
      </section>
      <ElEmpty v-if="!runList.length && !loadingRuns" description="暂无运行记录" />
    </main>
    <main v-else class="job-content ax-panel">
      <div class="panel-head"
        ><strong>节点配置</strong><span>{{ visibleNodes.length }} 个节点</span></div
      >
      <p class="panel-note">以下为已保存的节点配置，执行结果请查看运行记录。</p>
      <ElTable :data="visibleNodes" empty-text="此工作流未配置这类节点">
        <ElTableColumn prop="name" label="节点" />
        <ElTableColumn label="服务"
          ><template #default="{ row }">{{
            services.find((item) => item.id === Number(row.config?.service_id))?.name || '未指定'
          }}</template></ElTableColumn
        >
        <ElTableColumn label="依赖"
          ><template #default="{ row }">{{
            row.depends_on?.join('、') || '无前置节点'
          }}</template></ElTableColumn
        >
      </ElTable>
    </main>
    <ElPagination
      v-if="['runs', 'changes'].includes(activeTab) && totalRuns > pageSize"
      class="run-pagination"
      v-model:current-page="runPage"
      :page-size="pageSize"
      :total="totalRuns"
      layout="prev, pager, next"
      @current-change="changeRunPage"
    />
    <WorkflowDesignerDialog
      v-model:visible="designerVisible"
      :workflow="workflow"
      :project="project"
      :services="services"
      :editable="canEditWorkflow"
      @saved="reloadWorkflow"
    />

    <WorkflowRunDrawer
      v-model:visible="runDrawerVisible"
      :run-id="selectedRunId"
      :environments="environments"
      :clusters="clusters"
      :services="services"
      :code-sources="codeSources"
      @changed="loadRuns"
    />
    <WorkflowRollbackDialog
      v-model:visible="rollbackVisible"
      :run-id="rollbackRunId"
      :environment-name="environmentName"
      :service-names="Object.fromEntries(services.map((s) => [s.id, s.name]))"
      @submitted="
        (id, wid) => router.push(`/cicd/projects/${projectId}/workflows/${wid}/runs/${id}`)
      "
    />
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import { useRoute, useRouter } from 'vue-router'
  import { ChevronRight, Play, RefreshCw, Settings } from 'lucide-vue-next'
  import {
    createWorkflowApi,
    getProjectApi,
    listAllServicesApi,
    listClustersApi,
    listCodeSourcesApi,
    listEnvironmentsApi,
    listWorkflowRunsApi,
    getWorkflowApi,
    type AxiOpsCluster,
    type AxiOpsCodeSource,
    type AxiOpsEnvironment,
    type AxiOpsProject,
    type AxiOpsService,
    type AxiOpsWorkflow,
    type AxiOpsWorkflowRun
  } from '@/api/axiops'
  import WorkflowRunDrawer from '../project-workspace/components/WorkflowRunDrawer.vue'
  import ProjectNavBar from '../project-workspace/components/ProjectNavBar.vue'
  import ArtDeliveryIcon from '@/components/core/base/art-delivery-icon/index.vue'
  import WorkflowRollbackDialog from '../project-workspace/components/WorkflowRollbackDialog.vue'
  import BuildChanges from '@/components/build-changes/index.vue'
  import BuildCommitLink from '@/components/build-commit-link/index.vue'
  import { formatExecutionDuration } from '@/utils/execution-duration'
  import { workflowActorLabel } from '@/utils/workflow-actor'
  import WorkflowDesignerDialog from '../project-workspace/components/WorkflowDesignerDialog.vue'
  import { useAxiOpsAccess } from '@/composables/useAxiOpsAccess'

  const route = useRoute(),
    router = useRouter()
  const access = useAxiOpsAccess()
  const rollbackVisible = ref(false),
    rollbackRunId = ref(0)
  const openRollbackRun = (id: number) => {
    rollbackRunId.value = id
    rollbackVisible.value = true
  }
  const designerVisible = ref(false),
    cloning = ref(false)
  const canEditWorkflow = computed(() =>
    access.can('workflow', 'update', {
      type: 'workflow',
      project_id: projectId,
      id: workflowId,
      environment_id: workflow.value?.target_environment_id,
      environment: environments.value.find(
        (item) => item.id === workflow.value?.target_environment_id
      )?.code
    })
  )
  const projectId = Number(route.params.projectId),
    workflowId = Number(route.params.workflowId)
  const loading = ref(false),
    loadingRuns = ref(false),
    runsError = ref('')
  let runsRequest = 0
  const project = ref<AxiOpsProject | null>(null),
    workflow = ref<AxiOpsWorkflow | null>(null)
  const environments = ref<AxiOpsEnvironment[]>([]),
    clusters = ref<AxiOpsCluster[]>([]),
    services = ref<AxiOpsService[]>([]),
    codeSources = ref<AxiOpsCodeSource[]>([]),
    runList = ref<AxiOpsWorkflowRun[]>([])
  const activeTab = ref('runs'),
    selectedRunId = ref<number | null>(null),
    runDrawerVisible = ref(false)
  const expandedBuilds = ref<number[]>([])
  const runPage = ref(1),
    totalRuns = ref(0),
    pageSize = 20
  const serviceName = (id: number) =>
    services.value.find((item) => item.id === id)?.name || `服务 #${id}`
  const toggleChanges = (id: number) => {
    expandedBuilds.value = expandedBuilds.value.includes(id)
      ? expandedBuilds.value.filter((value) => value !== id)
      : [...expandedBuilds.value, id]
  }
  const tabs = [
    { value: 'runs', label: '执行记录' },
    { value: 'changes', label: '修改记录' },
    { value: 'build', label: '构建' },
    { value: 'deploy', label: '部署' },
    { value: 'test', label: '测试' }
  ]
  const visibleNodes = computed(() =>
    (workflow.value?.nodes || []).filter((node) =>
      activeTab.value === 'deploy'
        ? ['yaml_deploy', 'helm_deploy', 'host_deploy'].includes(node.type)
        : node.type === activeTab.value
    )
  )
  const reloadWorkflow = async () => {
    workflow.value = await getWorkflowApi(workflowId)
  }
  const environment = computed(() =>
    environments.value.find((item) => workflow.value?.target_environment_id === item.id)
  )
  const environmentName = computed(
    () =>
      environment.value?.name ||
      (workflow.value?.target_environment_id
        ? `环境 #${workflow.value.target_environment_id}`
        : workflow.value?.nodes?.some((node) =>
              ['yaml_deploy', 'helm_deploy', 'host_deploy'].includes(node.type)
            )
          ? '未绑定目标环境'
          : '不部署到环境')
  )
  const runEnvironmentLabel = (run: AxiOpsWorkflowRun) => {
    const target = environments.value.find((item) => item.id === run.environment_id)
    return target
      ? `${target.name} · ${target.namespace || target.host_group_ref}`
      : run.environment_id
        ? `环境 #${run.environment_id}`
        : '不部署到环境'
  }
  const formatDate = (value?: string) =>
    value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'
  const duration = (run: AxiOpsWorkflowRun) => {
    return formatExecutionDuration(run.started_at, run.finished_at)
  }
  const isSuccessful = (status: string) => ['succeeded', 'success', 'completed'].includes(status)
  const statusType = (status: string) =>
    isSuccessful(status)
      ? 'success'
      : ['failed', 'cancelled'].includes(status)
        ? 'danger'
        : 'warning'
  const statusLabel = (status: string) =>
    (
      ({
        succeeded: '成功',
        success: '成功',
        completed: '成功',
        failed: '失败',
        cancelled: '取消',
        running: '执行中',
        pending: '排队中'
      }) as Record<string, string>
    )[status] ||
    status ||
    '未知'
  const runner = workflowActorLabel
  const loadRuns = async () => {
    const request = ++runsRequest
    loadingRuns.value = true
    try {
      const result = await listWorkflowRunsApi(workflowId, {
        page: runPage.value,
        page_size: pageSize
      })
      if (request !== runsRequest) return
      runList.value = result.list || []
      totalRuns.value = result.total || 0
      expandedBuilds.value = expandedBuilds.value.filter((id) =>
        runList.value.some((run) => run.build_commits?.some((commit) => commit.build_id === id))
      )
      runsError.value = ''
    } catch (error) {
      if (request === runsRequest) {
        runList.value = []
        runsError.value = error instanceof Error ? error.message : '运行记录读取失败，请刷新重试'
      }
    } finally {
      if (request === runsRequest) loadingRuns.value = false
    }
  }
  const changeRunPage = () => {
    expandedBuilds.value = []
    void loadRuns()
  }
  const openRun = (id: number) => {
    void router.push(`/cicd/projects/${projectId}/workflows/${workflowId}/runs/${id}`)
  }
  const goWorkspace = () => {
    void router.push({ path: `/cicd/projects/${projectId}`, query: { tab: 'workflows' } })
  }
  const openExecuteDialog = () => {
    void router.push({
      path: `/cicd/projects/${projectId}`,
      query: { tab: 'workflows', execute: String(workflowId) }
    })
  }
  const cloneWorkflow = async () => {
    if (!workflow.value || !project.value || cloning.value) return
    cloning.value = true
    try {
      const result = await createWorkflowApi(projectId, {
        name: `${workflow.value.name}-副本`,
        code: `${workflow.value.code}-copy-${Date.now()}`.slice(0, 64),
        description: workflow.value.description || '',
        target_environment_id: workflow.value.target_environment_id || 0,
        definition: {
          delivery_type: workflow.value.delivery_type,
          nodes: workflow.value.nodes || []
        }
      })
      ElMessage.success('工作流已克隆')
      await router.push(`/cicd/projects/${projectId}/workflows/${result.id}`)
    } catch (error: any) {
      ElMessage.error(error?.message || '克隆失败')
    } finally {
      cloning.value = false
    }
  }
  onMounted(async () => {
    await access.load()
    loading.value = true
    try {
      const [
        projectResult,
        workflowResult,
        environmentResult,
        clusterResult,
        serviceResult,
        sourceResult
      ] = await Promise.all([
        getProjectApi(projectId),
        getWorkflowApi(workflowId),
        listEnvironmentsApi(projectId),
        listClustersApi(projectId),
        listAllServicesApi(projectId),
        listCodeSourcesApi(projectId)
      ])
      project.value = projectResult
      workflow.value = workflowResult
      environments.value = environmentResult
      clusters.value = clusterResult
      services.value = serviceResult
      codeSources.value = sourceResult
      await loadRuns()
      if (route.query.configure === '1') designerVisible.value = true
      if (route.query.execute === '1') openExecuteDialog()
    } finally {
      loading.value = false
    }
  })
</script>

<style scoped lang="scss">
  @use '@/views/cicd/styles/cicd-block.scss' as *;
  .run-commits {
    min-width: 0;
  }
  .run-commits small {
    color: #64748b;
    font-size: 11px;
  }
  .change-run {
    padding: 14px 16px;
    border-bottom: 1px solid #eef2f7;
  }
  .change-run-heading,
  .change-commit-heading {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 12px;
    margin-bottom: 8px;
  }
  .change-run p,
  .change-commit-heading small {
    color: #64748b;
    font-size: 12px;
  }
  .run-pagination {
    padding: 12px 16px;
  }
  .workflow-job-page {
    min-height: 100%;
    background: #fff;
    color: #334155;
  }
  .crumb-link {
    padding: 0;
    border: 0;
    background: transparent;
    color: var(--el-text-color-secondary);
    font-family: inherit;
    font-size: 13px;
    cursor: pointer;
  }
  .crumb-link:hover {
    color: #0f766e;
  }
  .crumb-sep {
    flex: 0 0 auto;
    color: var(--el-text-color-placeholder);
  }
  .crumb-type {
    flex: 0 0 auto;
    color: var(--el-text-color-secondary);
    opacity: 0.85;
  }
  .crumb-current {
    overflow: hidden;
    max-width: 260px;
    color: var(--el-text-color-primary);
    font-size: 14px;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .crumb-meta {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    white-space: nowrap;
  }
  .job-tabs {
    @include cicd-tabs;
  }
  .ax-panel {
    @include cicd-block;
  }
  .panel-head {
    @include cicd-block-head;
  }
  .panel-note {
    padding: 10px 16px;
    color: #64748b;
    font-size: 12px;
  }
  .job-content {
    padding: 12px 16px;
  }
  .runs-table {
    width: 100%;
    overflow-x: auto;
  }
  .pipeline-panel {
    margin-bottom: 22px;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    background: #fff;
  }
  .pipeline-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 18px;
    border-bottom: 1px solid #eef2f7;
  }
  .pipeline-toolbar > div {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .pipeline-toolbar span {
    color: #94a3b8;
    font-size: 12px;
  }
  .pipeline-flow {
    display: flex;
    align-items: center;
    gap: 8px;
    overflow-x: auto;
    padding: 26px 20px 30px;
    background: #f8fafc;
  }
  .pipeline-start,
  .pipeline-end {
    flex: 0 0 auto;
    color: #64748b;
    font-size: 12px;
    font-weight: 600;
  }
  .pipeline-line {
    flex: 0 0 26px;
    height: 2px;
    background: #cbd5e1;
  }
  .pipeline-stage {
    position: relative;
    flex: 0 0 210px;
    min-height: 104px;
    padding: 12px;
    border: 1px solid #cbd5e1;
    border-radius: 7px;
    background: #fff;
    color: #334155;
    text-align: left;
    cursor: pointer;
  }
  .pipeline-stage:hover {
    border-color: #2563eb;
    box-shadow: 0 3px 10px rgb(37 99 235 / 12%);
  }
  .pipeline-stage.is-success {
    border-color: #86efac;
  }
  .pipeline-stage.is-failed {
    border-color: #fca5a5;
  }
  .pipeline-stage.is-running {
    border-color: #93c5fd;
  }
  .pipeline-stage-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }
  .pipeline-stage-tasks {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-top: 12px;
    font-size: 12px;
  }
  .pipeline-stage-tasks span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .pipeline-stage-tasks em {
    color: #94a3b8;
    font-style: normal;
  }
  .pipeline-stage > small {
    position: absolute;
    right: 12px;
    bottom: 10px;
    color: #94a3b8;
  }
  .run-row {
    display: grid;
    grid-template-columns: 64px 88px 100px 150px 1.2fr 1fr 150px;
    min-width: 960px;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 11px 16px;
    border: 0;
    border-bottom: 1px solid #eef2f7;
    background: #fff;
    color: #475569;
    text-align: left;
    font-size: 13px;
  }
  button.run-row {
    cursor: pointer;
  }
  button.run-row:hover {
    background: #f8fafc;
  }
  .run-header {
    @include cicd-table-head;
  }
  .run-row:last-child {
    border-bottom: 0;
  }
  // 编号、耗时、时间统一等宽，与概览指标列同一视觉语言
  .run-no,
  .run-dur {
    font-family: var(--ax-font-mono);
  }
  .runner small {
    font-family: var(--ax-font-mono);
    font-size: 11px;
  }
  .run-number,
  .row-action,
  .code-info {
    color: #2563eb;
  }
  .runner,
  .branch,
  .multiline {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }
  .runner small,
  .branch small {
    color: #94a3b8;
  }
  .multiline span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .muted {
    color: #94a3b8;
  }
  .test-result {
    color: #2563eb;
  }
  .row-action {
    cursor: pointer;
  }
  .job-detail-metrics {
    margin-bottom: 20px;
  }
  .execute-stage-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding-bottom: 16px;
    border-bottom: 1px solid #e2e8f0;
    color: #334155;
    font-size: 13px;
  }
  .execute-stage-bar span {
    margin-right: 6px;
    font-weight: 600;
  }
  .execute-section {
    padding: 18px 0;
    border-bottom: 1px solid #eef2f7;
  }
  .execute-section:last-child {
    border-bottom: 0;
  }
  .execute-section-title {
    display: flex;
    align-items: baseline;
    gap: 10px;
    margin-bottom: 13px;
  }
  .execute-section-title strong {
    color: #1e293b;
    font-size: 14px;
  }
  .execute-section-title span,
  .execute-test-note {
    color: #94a3b8;
    font-size: 12px;
  }
  .service-chip-list,
  .execute-service-list {
    display: flex;
    flex-wrap: wrap;
    gap: 8px 20px;
    padding: 10px 12px;
    background: #f8fafc;
    border: 1px solid #e2e8f0;
  }
  .execute-form-row {
    display: flex;
    align-items: center;
    gap: 18px;
    margin-top: 13px;
  }
  .execute-form-row > label {
    width: 48px;
    color: #64748b;
    font-size: 12px;
  }
  .execute-service-list {
    margin: 10px 0 4px 66px;
    flex-direction: column;
    gap: 6px;
  }
  .execute-service-list small {
    margin-left: 8px;
    color: #94a3b8;
  }
  .execute-build-list {
    margin-left: 0;
    width: 100%;
    box-sizing: border-box;
  }
  .execute-service-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    min-height: 38px;
  }
  .execute-service-row .el-checkbox {
    flex: 1;
    min-width: 0;
  }
  .execute-branch-select {
    width: 280px;
  }
  @media (max-width: 900px) {
    .crumb-meta {
      display: none;
    }
  }
</style>
