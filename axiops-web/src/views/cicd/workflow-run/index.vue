<template>
  <div v-loading="loading" class="workflow-run-page">
    <ProjectNavBar :project-id="projectId" :project="project" active="workflows" compact>
      <template #crumb>
        <button type="button" class="crumb-link" @click="goWorkspace">工作流</button>
        <ChevronRight :size="14" class="crumb-sep" />
        <ArtDeliveryIcon :type="workflow?.delivery_type" :size="14" class="crumb-type" />
        <button type="button" class="crumb-link" @click="goJob">{{
          workflow?.name || 'Job'
        }}</button>
        <ChevronRight :size="14" class="crumb-sep" />
        <span class="crumb-current" :title="`运行记录 ID：${runId}`"
          >#{{ run?.run_number || '—' }}</span
        >
      </template>
    </ProjectNavBar>

    <header class="run-header">
      <div class="run-heading">
        <strong>{{ workflow?.name || '工作流' }}</strong>
        <small>{{ environmentName }} · {{ formatDate(run?.created_at) }}</small>
        <small>执行人 {{ workflowActorLabel(run) }}</small>
        <small v-if="run?.operation_type === 'rollback'"
          >回滚 {{ run.context_snapshot?.service_name || '' }} → 原运行 #{{
            run.context_snapshot?.target_run_number || '—'
          }}</small
        >
        <small v-if="run?.started_at" class="run-duration">总耗时 {{ taskDuration(run) }}</small>
      </div>
      <ElTag :type="statusType(run?.status)" effect="plain">{{ statusLabel(run?.status) }}</ElTag>
      <ElButton v-if="run?.environment_id" text @click="openRollback()">回滚…</ElButton>
    </header>
    <ElAlert
      v-if="promotionSource"
      type="info"
      :closable="false"
      :title="`使用已有镜像 · 源构建 #${promotionSource.build_number} · 本次未重新构建`"
      :description="`${promotionSource.image_url}@${promotionSource.image_digest}；使用当前目标环境的配置。`"
    />

    <ElAlert
      v-if="run?.context_snapshot?.execution_mode === 'config_only'"
      type="info"
      :closable="false"
      title="仅更新当前环境配置 · 保留原容器镜像 · 本次未构建"
      description="本次使用提交时冻结的环境配置，发布步骤和健康检查结果见下方。"
    />
    <ElAlert
      v-if="run?.context_snapshot?.execution_mode === 'selected_image'"
      type="info"
      :closable="false"
      title="使用已有镜像 · 本次不重新构建"
      description="镜像摘要和目标环境配置已固定，执行结果见下方日志。"
    />
    <ElAlert
      v-if="run?.context_snapshot?.execution_mode === 'verified_image'"
      type="info"
      :closable="false"
      title="生产发布 · 使用已验收的同一份镜像 · 本次不重新构建"
      description="验收来源和镜像摘要已冻结；使用生产环境自己的配置，执行结果见下方日志。"
    />
    <div
      v-for="version in productionVersions"
      :key="version.service_id"
      class="production-provenance"
    >
      <strong
        >{{ version.name }} · b{{ version.build_number }}-{{ version.commit?.slice(0, 8) }}</strong
      >
      <p v-if="version.method !== 'build'"
        >来源运行 #{{ version.run_number }} · 人工验收 {{ version.username }} ·
        {{ formatDate(version.verified_at) }}</p
      >
      <p v-else
        >构建 #{{ version.build_number }} · {{ version.branch || '未知分支' }} ·
        {{ formatDate(version.built_at) }}</p
      >
      <p>{{ version.note }}</p>
      <ElButton
        v-if="version.run_id && version.workflow_id"
        link
        type="primary"
        @click="
          router.push(
            `/cicd/projects/${projectId}/workflows/${version.workflow_id}/runs/${version.run_id}`
          )
        "
        >查看来源部署</ElButton
      >
    </div>
    <DeploymentVerificationPanel
      v-if="run && environments.some((e) => e.id === run?.environment_id && !e.is_production)"
      :run-id="runId"
      :project-id="projectId"
      :environment-id="run.environment_id"
      :refresh-key="`${run.status}:${run.tasks?.map((t) => t.status).join(',')}`"
    />
    <div class="run-tabs"
      ><button :class="{ 'is-active': activeTab === 'workflow' }" @click="activeTab = 'workflow'"
        >工作流</button
      ><button :class="{ 'is-active': activeTab === 'snapshot' }" @click="activeTab = 'snapshot'"
        >本次配置</button
      ><button :class="{ 'is-active': activeTab === 'changes' }" @click="activeTab = 'changes'"
        >代码变更</button
      ></div
    >

    <main v-if="activeTab === 'changes'" class="run-content">
      <template
        v-for="task in run?.tasks?.filter((item) => item.node_type === 'build') || []"
        :key="task.id"
      >
        <strong>{{ taskLabel(task) }}</strong>
        <BuildChanges v-if="buildTaskId(task)" :build-id="buildTaskId(task)" />
        <p v-else>构建尚未提交，代码检出后显示修改记录。</p>
      </template>
      <BuildChanges v-if="promotionSource" :build-id="promotionSource.id" />
      <section v-for="version in productionVersions" :key="version.service_id">
        <strong>{{ version.name }} · 源构建 #{{ version.build_number }}</strong>
        <BuildChanges :build-id="version.build_id" />
      </section>
      <p
        v-if="
          !promotionSource &&
          !productionVersions.length &&
          !run?.tasks?.some((item) => item.node_type === 'build')
        "
        >此次运行没有构建节点。</p
      >
    </main>
    <main v-if="activeTab === 'snapshot'" class="run-content">
      <ElAlert
        title="以下为本次运行绑定的配置版本；保存新配置不会自动更新已运行的服务。"
        type="info"
        :closable="false"
      />
      <p
        >环境 {{ environmentName }} · 工作流定义 v{{ run?.definition_version }} · 执行人
        {{ workflowActorLabel(run) }}</p
      >
      <section v-for="task in run?.tasks || []" :key="task.id" class="task-detail">
        <strong>{{ taskLabel(task) }}</strong>
        <p>服务 {{ taskService(task) }} · {{ statusLabel(task.status) }}</p>
        <p v-if="task.node_snapshot?.config?.service_revision_id"
          >配置版本 #{{ task.node_snapshot.config.service_revision_id }}</p
        >
        <p v-if="task.node_snapshot?.config?.branch">分支 {{ task.node_snapshot.config.branch }}</p>
        <p v-if="task.node_snapshot?.config?.commit_id"
          >Commit {{ task.node_snapshot.config.commit_id }}</p
        >
        <p v-if="task.provider_run_id">执行记录 {{ task.provider_run_id }}</p>
        <ElButton text type="primary" @click="showTaskFromSnapshot(task)"
          >查看执行结果和日志</ElButton
        >
      </section>
    </main>
    <main v-show="activeTab === 'workflow'" class="run-content">
      <ElAlert v-if="pageError" :title="pageError" type="error" :closable="false" show-icon />
      <section class="flow-panel">
        <div class="flow-toolbar"
          ><strong>执行流程</strong
          ><span>{{ (run?.tasks || []).length }} 个任务 · {{ flowSummary }}</span
          ><div class="flow-legend">
            <span v-for="item in flowLegend" :key="item.state" class="legend-item">
              <i class="legend-dot" :class="item.state" />{{ item.label }}
            </span>
          </div>
          <ElButton v-if="activeTask" size="small" text @click="flowExpanded = !flowExpanded">
            {{ flowExpanded ? '收起流程图' : '展开流程图' }}
          </ElButton>
        </div>
        <div v-show="!activeTask || flowExpanded" class="flow-canvas">
          <div class="flow-endpoint">
            <span class="endpoint-badge is-start"><Play :size="13" /></span>
            <span class="endpoint-label">开始</span>
          </div>
          <template v-for="(stage, index) in stages" :key="stage.key">
            <span class="flow-line" :class="lineState(index)" />
            <section class="stage-card" :class="`is-${stageState(stage)}`">
              <header class="stage-head">
                <span class="stage-no">{{ index + 1 }}</span>
                <h3>{{ stage.label }}</h3>
                <span class="stage-count">{{ stage.tasks.length }}</span>
                <span class="stage-flag">{{ stageStateLabel(stageState(stage)) }}</span>
              </header>
              <div class="stage-body">
                <button
                  v-for="task in stage.tasks"
                  :key="task.id"
                  type="button"
                  class="task-card"
                  :class="[taskState(task.status), { selected: task.id === activeTask?.id }]"
                  :title="`${taskLabel(task)} · ${statusLabel(task.status)}`"
                  :aria-pressed="task.id === activeTask?.id"
                  @click="toggleTask(task)"
                >
                  <span class="task-state">
                    <component :is="stateIcon(task.status)" :size="12" :stroke-width="3" />
                  </span>
                  <span class="task-copy"
                    ><strong>{{ taskLabel(task) }}</strong
                    ><small>{{ taskService(task) }}</small></span
                  >
                  <small class="task-duration">{{ taskDuration(task) }}</small>
                </button>
                <span v-if="!stage.tasks.length" class="empty-task">暂无任务</span>
              </div>
            </section>
          </template>
          <span class="flow-line" :class="lineState(stages.length)" />
          <div class="flow-endpoint">
            <span class="endpoint-badge is-end"><Flag :size="13" /></span>
            <span class="endpoint-label">结束</span>
          </div>
        </div>
        <div v-if="activeTask" class="compact-tasks">
          <button
            v-for="task in run?.tasks || []"
            :key="task.id"
            type="button"
            class="task-chip"
            :class="[taskState(task.status), { selected: task.id === activeTask.id }]"
            :aria-pressed="task.id === activeTask.id"
            :title="`${taskLabel(task)} · ${statusLabel(task.status)}`"
            @click="toggleTask(task)"
          >
            <i class="legend-dot" :class="taskState(task.status)" />{{ taskLabel(task) }}
            <small>{{ taskDuration(task) }}</small>
          </button>
        </div>
      </section>

      <section v-if="activeTask" class="task-detail">
        <div class="task-detail-header"
          ><strong>{{ taskLabel(activeTask) }}</strong
          ><span>{{ taskService(activeTask) }} · {{ taskDuration(activeTask) }}</span
          ><div class="detail-actions">
            <ElButton
              v-if="
                ['yaml_deploy', 'helm_deploy', 'yaml_rollback', 'helm_rollback'].includes(
                  activeTask.node_type
                )
              "
              size="small"
              text
              type="primary"
              @click="openRollback(activeTask)"
              >回滚该服务</ElButton
            >
            <ElButton size="small" text @click="metadataExpanded = !metadataExpanded">
              {{ metadataExpanded ? '收起信息' : '详细信息' }}
            </ElButton>
            <ElButton size="small" text @click="closeTask">关闭日志</ElButton>
          </div>
        </div>
        <div v-if="!metadataExpanded" class="task-meta-summary">
          <span :title="environmentName"
            >环境 <strong>{{ environmentName }}</strong></span
          >
          <span :title="`${taskMeta.branch} → ${taskMeta.commit}`"
            >分支 <strong>{{ taskMeta.branch }}</strong></span
          >
          <span :title="taskMeta.image"
            >镜像 <strong>{{ taskMeta.image }}</strong></span
          >
          <ElTag size="small" :type="statusType(activeTask.status)" effect="plain">{{
            statusLabel(activeTask.status)
          }}</ElTag>
        </div>
        <div v-else class="task-meta-grid">
          <div
            ><span>代码仓库</span><strong>{{ taskMeta.repo }}</strong></div
          >
          <div
            ><span>分支</span><strong>{{ taskMeta.branch }} → {{ taskMeta.commit }}</strong></div
          >
          <div
            ><span>镜像名称</span><strong>{{ taskMeta.image }}</strong></div
          >
          <div
            ><span>节点状态</span><strong>{{ statusLabel(activeTask.status) }}</strong></div
          >
        </div>
        <ElAlert v-if="logError" :title="logError" type="error" :closable="false" show-icon />
        <BuildCommitLink
          v-if="activeTask.node_type === 'build' && buildRun?.git_commit"
          :commit="buildRun.git_commit"
          :subject="buildRun.git_message"
          :url="buildRun.git_commit_url"
          :author="buildRun.git_author"
        />
        <ElAlert
          v-if="buildRun?.error_message || activeTask.error_message"
          :title="taskErrorTitle"
          type="warning"
          :closable="false"
          show-icon
        />
        <BuildLogViewer
          :key="activeTask.id"
          :text="taskLogText"
          :summary-text="deploymentSummary"
          :title="taskLogTitle"
          :empty-text="emptyLogText"
          :loading="logLoading"
          @refresh="refresh"
        />
      </section>
    </main>
    <WorkflowRollbackDialog
      v-model:visible="rollbackVisible"
      :run-id="runId"
      :service-id="rollbackServiceId"
      :environment-name="environmentName"
      @submitted="
        (id, wid) => router.push(`/cicd/projects/${projectId}/workflows/${wid}/runs/${id}`)
      "
    />
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, onBeforeUnmount, ref } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import { ElAlert, ElTag } from 'element-plus'
  import { Check, ChevronRight, Clock, Flag, Loader2, Minus, Play, X } from 'lucide-vue-next'
  import BuildLogViewer from '@/components/build-log-viewer/index.vue'
  import BuildChanges from '@/components/build-changes/index.vue'
  import BuildCommitLink from '@/components/build-commit-link/index.vue'
  import WorkflowRollbackDialog from '../project-workspace/components/WorkflowRollbackDialog.vue'
  import ProjectNavBar from '../project-workspace/components/ProjectNavBar.vue'
  import ArtDeliveryIcon from '@/components/core/base/art-delivery-icon/index.vue'
  import DeploymentVerificationPanel from '../project-workspace/components/DeploymentVerificationPanel.vue'
  import { summarizeDeploymentLog } from '@/utils/deployment-log'
  import { formatExecutionDuration } from '@/utils/execution-duration'
  import { workflowActorLabel } from '@/utils/workflow-actor'
  import {
    getBuildLogsApi,
    getBuildRunApi,
    getProjectApi,
    getWorkflowApi,
    getWorkflowRunApi,
    listEnvironmentsApi,
    type AxiOpsBuildRun,
    type AxiOpsEnvironment,
    type AxiOpsProject,
    type AxiOpsTaskRun,
    type AxiOpsWorkflow,
    type AxiOpsWorkflowRun
  } from '@/api/axiops'

  const route = useRoute(),
    router = useRouter()
  const projectId = Number(route.params.projectId),
    workflowId = Number(route.params.workflowId),
    runId = Number(route.params.runId)
  const loading = ref(false),
    run = ref<AxiOpsWorkflowRun | null>(null),
    workflow = ref<AxiOpsWorkflow | null>(null),
    project = ref<AxiOpsProject | null>(null),
    environments = ref<AxiOpsEnvironment[]>([]),
    activeTask = ref<AxiOpsTaskRun | null>(null),
    logs = ref<string[]>([])
  const buildRun = ref<AxiOpsBuildRun | null>(null)
  const promotionSource = computed(() => {
    const context = run.value?.context_snapshot
    return context?.execution_mode === 'existing_image'
      ? (context.source_build as
          | { id: number; build_number: number; image_url: string; image_digest: string }
          | undefined)
      : undefined
  })
  const productionVersions = computed(() => {
    const context = run.value?.context_snapshot
    if (
      !context ||
      !['verified_image', 'selected_image'].includes(String(context.execution_mode || ''))
    )
      return []
    return (
      (context.production_guards || []) as { version: import('@/api/axiops').ProductionCandidate }[]
    )
      .map((g) => g.version)
      .filter(Boolean)
  })
  const isContainerTask = (task: AxiOpsTaskRun | null) =>
    !!task && ['build', 'test'].includes(task.node_type)
  const activeTab = ref('workflow')
  const rollbackVisible = ref(false),
    rollbackServiceId = ref<number>()
  const openRollback = (task?: AxiOpsTaskRun) => {
    rollbackServiceId.value = Number(task?.node_snapshot?.config?.service_id) || undefined
    rollbackVisible.value = true
  }
  const goWorkspace = () => {
    void router.push({ path: `/cicd/projects/${projectId}`, query: { tab: 'workflows' } })
  }
  const goJob = () => {
    void router.push(`/cicd/projects/${projectId}/workflows/${workflowId}`)
  }
  const flowExpanded = ref(false)
  const metadataExpanded = ref(false)
  const pageError = ref(''),
    logError = ref(''),
    logLoading = ref(false)
  let timer: ReturnType<typeof setTimeout> | undefined
  const clockNow = ref(Date.now())
  let clockTimer: ReturnType<typeof setInterval> | undefined
  let refreshing = false,
    refreshAgain = false,
    disposed = false,
    selectionVersion = 0
  const errorText = (error: unknown) => (error instanceof Error ? error.message : String(error))
  const emptyLogText = computed(() => {
    if (logLoading.value) return '正在读取日志…'
    if (logError.value) return '日志读取失败，系统将自动重试；也可点击刷新。'
    if (!isContainerTask(activeTask.value)) {
      if (activeTask.value?.error_message) return activeTask.value.error_message
      if (activeTask.value?.status === 'pending')
        return '任务尚未开始，开始后显示执行步骤和检查结果。'
      if (activeTask.value?.status === 'running')
        return isYAMLTask.value
          ? '部署执行中，等待提交或 Kubernetes 检查结果；每 3 秒更新。'
          : '任务执行中，尚无执行日志。'
      return '该任务已结束，未保存执行检查日志；历史记录不会补写为当前资源状态。'
    }
    if (
      ['success', 'succeeded', 'failed', 'timeout', 'cancelled', 'skipped'].includes(
        buildRun.value?.status || activeTask.value?.status || ''
      )
    )
      return '任务已结束，未保存容器输出。'
    return '等待构建任务提交、调度或容器启动；状态与日志每 3 秒更新。'
  })
  const isYAMLTask = computed(() =>
    ['yaml_deploy', 'yaml_rollback', 'workload_restart'].includes(activeTask.value?.node_type || '')
  )
  const taskLogText = computed(() =>
    isContainerTask(activeTask.value)
      ? [activeTask.value?.log_text, ...logs.value].filter(Boolean).join('\n')
      : activeTask.value?.log_text || activeTask.value?.error_message || ''
  )
  const deploymentSummary = computed(() =>
    isYAMLTask.value
      ? summarizeDeploymentLog(
          taskLogText.value,
          activeTask.value?.status,
          activeTask.value?.error_message
        )
      : undefined
  )
  const taskErrorTitle = computed(() => {
    const text = buildRun.value?.error_message || activeTask.value?.error_message || ''
    return isYAMLTask.value ? text.split('\n')[0] : text
  })
  const taskLogTitle = computed(() => {
    const task = activeTask.value
    if (!task) return '执行日志'
    const label =
      task.node_type === 'build' ? '构建日志' : isYAMLTask.value ? '部署检查' : '执行日志'
    return `${taskLabel(task)} · ${label}`
  })
  const environmentName = computed(
    () => environments.value.find((item) => item.id === run.value?.environment_id)?.name || '-'
  )
  const statusType = (status?: string) =>
    ['succeeded', 'success', 'completed'].includes(status || '')
      ? 'success'
      : ['failed', 'cancelled'].includes(status || '')
        ? 'danger'
        : 'warning'
  const statusLabel = (status?: string) =>
    (
      ({
        succeeded: '成功',
        success: '成功',
        completed: '成功',
        failed: '失败',
        cancelled: '已取消',
        timeout: '超时',
        skipped: '已跳过',
        running: '执行中',
        pending: '排队中',
        pending_approval: '待审批',
        cancelling: '正在停止后续任务',
        queued: '排队中',
        dispatching: '提交中'
      }) as Record<string, string>
    )[status || ''] ||
    status ||
    '未知'
  const formatDate = (value?: string) =>
    value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'
  const taskState = (status?: string) => {
    const value = status || ''
    if (['succeeded', 'success', 'completed'].includes(value)) return 'success'
    if (['failed', 'timeout'].includes(value)) return 'error'
    if (['cancelled', 'cancelling'].includes(value)) return 'warning'
    if (['running', 'dispatching'].includes(value)) return 'running'
    if (value === 'skipped') return 'skipped'
    return 'pending'
  }
  const stateIcon = (status?: string) =>
    (
      ({
        success: Check,
        error: X,
        running: Loader2,
        skipped: Minus
      }) as Record<string, typeof Check>
    )[taskState(status)] || Clock
  const stageState = (stage: { tasks: AxiOpsTaskRun[] }) => {
    const states = stage.tasks.map((task) => taskState(task.status))
    if (!states.length) return 'pending'
    if (states.includes('error')) return 'error'
    if (states.includes('running')) return 'running'
    if (states.includes('warning')) return 'warning'
    if (states.every((item) => item === 'success' || item === 'skipped')) return 'success'
    return 'pending'
  }
  const stageStateLabel = (state: string) =>
    (
      ({
        success: '已完成',
        error: '失败',
        warning: '已停止',
        running: '进行中'
      }) as Record<string, string>
    )[state] || '未开始'
  const lineState = (index: number) => {
    if (index === 0) return 'flowing'
    const previous = stages.value[index - 1]
    return previous ? stageState(previous) : 'pending'
  }
  const flowLegend = [
    { state: 'success', label: '成功' },
    { state: 'running', label: '执行中' },
    { state: 'error', label: '失败' },
    { state: 'pending', label: '排队' }
  ]
  const flowSummary = computed(() => {
    const states = (run.value?.tasks || []).map((task) => taskState(task.status))
    if (!states.length) return '暂无任务'
    if (states.includes('error')) return '存在失败任务'
    if (states.includes('running')) return '正在执行'
    if (states.every((item) => item === 'success' || item === 'skipped')) return '全部完成'
    return '等待执行'
  })
  const taskDuration = (task: Pick<AxiOpsTaskRun, 'started_at' | 'finished_at'>) =>
    formatExecutionDuration(task.started_at, task.finished_at, clockNow.value)
  const taskLabel = (task: AxiOpsTaskRun) => task.node_snapshot?.name || task.node_key
  const buildTaskId = (task: AxiOpsTaskRun) =>
    /^build:\d+$/.test(task.provider_run_id || '') ? Number(task.provider_run_id.slice(6)) : 0
  const taskService = (task: AxiOpsTaskRun) =>
    String(
      (task.node_snapshot?.config as any)?.service_name ||
        (task.node_snapshot?.config as any)?.service_id ||
        task.node_type
    )
  const stages = computed(() =>
    [
      { key: 'build', label: '构建', types: ['build'] },
      {
        key: 'deploy',
        label: '部署',
        types: [
          'yaml_deploy',
          'helm_deploy',
          'host_deploy',
          'yaml_rollback',
          'helm_rollback',
          'workload_restart'
        ]
      },
      { key: 'test', label: '测试', types: ['test'] }
    ].map((stage) => ({
      ...stage,
      tasks: (run.value?.tasks || []).filter((task) => stage.types.includes(task.node_type))
    }))
  )
  const taskMeta = computed(() => {
    const config = (activeTask.value?.node_snapshot?.config || {}) as any
    const buildTasks = (run.value?.tasks || []).filter(
      (task) =>
        task.node_type === 'build' &&
        (task.node_snapshot?.config as any)?.service_id === config.service_id
    )
    const buildConfig =
      buildTasks.length === 1 ? (buildTasks[0].node_snapshot?.config as any) : undefined
    return {
      repo: config.repo_url || config.repository || '-',
      branch:
        buildRun.value?.git_branch ||
        config.branch ||
        buildConfig?.branch ||
        run.value?.context_snapshot?.branch ||
        run.value?.git_branch ||
        '-',
      commit: buildRun.value?.git_commit_short || run.value?.git_commit_short || '-',
      image: buildRun.value?.image_url || config.image || config.image_name || '-'
    }
  })
  const closeTask = () => {
    selectionVersion++
    activeTask.value = null
    buildRun.value = null
    logs.value = []
    logError.value = ''
    logLoading.value = false
    flowExpanded.value = false
    metadataExpanded.value = false
  }
  const toggleTask = (task: AxiOpsTaskRun) => {
    if (activeTask.value?.id === task.id) {
      closeTask()
      return
    }
    closeTask()
    activeTask.value = task
    logLoading.value = isContainerTask(task)
    void refresh()
  }
  const showTaskFromSnapshot = (task: AxiOpsTaskRun) => {
    activeTab.value = 'workflow'
    closeTask()
    toggleTask(task)
  }
  const refresh = async () => {
    if (disposed) return
    if (timer) clearTimeout(timer)
    if (refreshing) {
      refreshAgain = true
      return
    }
    refreshing = true
    try {
      try {
        const latest = await getWorkflowRunApi(runId)
        if (disposed) return
        run.value = latest
        pageError.value = ''
        if (activeTask.value) {
          const updated = latest.tasks?.find((task) => task.id === activeTask.value?.id) || null
          if (updated?.provider_run_id !== activeTask.value.provider_run_id) {
            selectionVersion++
            logs.value = []
            buildRun.value = null
            logError.value = ''
          }
          activeTask.value = updated
        }
      } catch (error) {
        if (!disposed) pageError.value = `工作流状态读取失败：${errorText(error)}`
      }
      const task = activeTask.value,
        version = selectionVersion
      const buildId = isContainerTask(task)
        ? Number(String(task?.provider_run_id || '').replace(/^build:/, ''))
        : 0
      if (!buildId || disposed) {
        logLoading.value = false
        return
      }
      const [detail, output] = await Promise.allSettled([
        getBuildRunApi(buildId),
        getBuildLogsApi(buildId)
      ])
      if (disposed || version !== selectionVersion || activeTask.value?.id !== task?.id) return
      const errors: string[] = []
      if (detail.status === 'fulfilled') buildRun.value = detail.value
      else errors.push(`构建状态读取失败：${errorText(detail.reason)}`)
      if (output.status === 'fulfilled')
        logs.value = (output.value || []).map((item) => item.log_text || '').filter(Boolean)
      else errors.push(`日志读取失败：${errorText(output.reason)}`)
      logError.value = errors.join('；')
      logLoading.value = false
    } finally {
      refreshing = false
      if (!disposed) {
        timer = setTimeout(
          () => {
            void refresh()
          },
          refreshAgain ? 0 : 3000
        )
        refreshAgain = false
      }
    }
  }
  onMounted(async () => {
    clockTimer = setInterval(() => {
      clockNow.value = Date.now()
    }, 1000)
    loading.value = true
    const [metadata] = await Promise.allSettled([
      Promise.all([
        getWorkflowApi(workflowId),
        listEnvironmentsApi(projectId),
        getProjectApi(projectId).catch(() => null)
      ]),
      refresh()
    ])
    if (disposed) return
    if (metadata.status === 'fulfilled')
      [workflow.value, environments.value, project.value] = metadata.value
    else pageError.value = `工作流信息读取失败：${errorText(metadata.reason)}`
    loading.value = false
  })
  onBeforeUnmount(() => {
    disposed = true
    selectionVersion++
    if (timer) clearTimeout(timer)
    if (clockTimer) clearInterval(clockTimer)
  })
</script>

<style scoped lang="scss">
  @use '@/views/cicd/styles/cicd-block.scss' as *;
  .workflow-run-page {
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
    color: var(--el-text-color-primary);
    font-family: var(--ax-font-mono);
    font-size: 13px;
    font-weight: 600;
  }
  .run-header {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 16px;
    border-bottom: 1px solid #e2e8f0;
  }
  .run-heading {
    display: flex;
    flex: 1;
    flex-direction: row;
    align-items: baseline;
    flex-wrap: wrap;
    gap: 4px 12px;
  }
  .run-heading strong {
    color: #1e293b;
    font-size: 14px;
  }
  .run-heading small {
    color: #64748b;
    font-size: 12px;
  }
  .run-duration {
    font-family: var(--ax-font-mono);
  }
  .run-tabs {
    @include cicd-tabs;
  }
  .run-content {
    padding: 12px 16px;
  }
  /* ============ 执行流程 ============ */
  .flow-panel {
    @include cicd-block;
  }
  .flow-toolbar {
    @include cicd-block-head;
  }
  .flow-legend {
    display: flex;
    flex: 1;
    align-items: center;
    justify-content: flex-end;
    gap: 14px;
    min-width: 0;
  }
  .legend-item {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    white-space: nowrap;
  }
  .legend-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--el-text-color-secondary);
    flex: none;
  }
  .legend-dot.success {
    background: var(--el-color-success);
  }
  .legend-dot.error {
    background: var(--el-color-error);
  }
  .legend-dot.warning {
    background: var(--el-color-warning);
  }
  .legend-dot.running {
    background: var(--el-color-primary);
    animation: flow-pulse 1.4s ease-in-out infinite;
  }
  .flow-canvas {
    display: flex;
    align-items: stretch;
    gap: 0;
    min-height: 190px;
    padding: 22px 18px;
    overflow-x: auto;
    background: var(--ax-surface-sunken);
  }
  /* 端点：开始 / 结束 */
  .flow-endpoint {
    flex: 0 0 auto;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    width: 56px;
  }
  .endpoint-badge {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    border-radius: 50%;
    border: 1px solid var(--el-border-color);
    background: var(--ax-surface-card);
    color: var(--el-text-color-secondary);
  }
  .endpoint-badge.is-start {
    border-color: color-mix(in srgb, var(--el-color-primary) 40%, transparent);
    background: color-mix(in srgb, var(--el-color-primary) 10%, transparent);
    color: var(--el-color-primary);
  }
  .endpoint-label {
    color: var(--el-text-color-secondary);
    font-size: 13px;
    letter-spacing: 0.02em;
  }
  /* 连接线：带箭头，颜色随前一阶段状态 */
  .flow-line {
    position: relative;
    flex: 0 0 52px;
    align-self: center;
    height: 2px;
    background: var(--el-border-color);
  }
  .flow-line::after {
    content: '';
    position: absolute;
    top: -4px;
    right: -1px;
    border-top: 5px solid transparent;
    border-bottom: 5px solid transparent;
    border-left: 7px solid var(--el-border-color);
  }
  .flow-line.flowing,
  .flow-line.running {
    background: var(--el-color-primary);
  }
  .flow-line.flowing::after,
  .flow-line.running::after {
    border-left-color: var(--el-color-primary);
  }
  .flow-line.success {
    background: var(--el-color-success);
  }
  .flow-line.success::after {
    border-left-color: var(--el-color-success);
  }
  .flow-line.error {
    background: var(--el-color-error);
  }
  .flow-line.error::after {
    border-left-color: var(--el-color-error);
  }
  .flow-line.warning {
    background: var(--el-color-warning);
  }
  .flow-line.warning::after {
    border-left-color: var(--el-color-warning);
  }
  /* 阶段卡 */
  .stage-card {
    flex: 0 0 264px;
    display: flex;
    flex-direction: column;
    padding: 12px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--ax-radius-md);
    background: var(--ax-surface-card);
    box-shadow: var(--ax-shadow-xs);
  }
  .stage-card.is-error {
    border-color: color-mix(in srgb, var(--el-color-error) 45%, transparent);
  }
  .stage-card.is-running {
    border-color: color-mix(in srgb, var(--el-color-primary) 45%, transparent);
  }
  .stage-head {
    display: flex;
    align-items: center;
    gap: 8px;
    padding-bottom: 10px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  .stage-no {
    flex: none;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    border-radius: 50%;
    background: color-mix(in srgb, var(--el-color-primary) 12%, transparent);
    color: var(--el-color-primary);
    font-family: var(--ax-font-mono);
    font-size: 12px;
    font-weight: 600;
  }
  .stage-card h3 {
    flex: 1;
    min-width: 0;
    margin: 0;
    overflow: hidden;
    color: var(--el-text-color-primary);
    font-family: var(--ax-font-display);
    font-size: 15px;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .stage-count {
    flex: none;
    padding: 1px 7px;
    border-radius: var(--ax-radius-xs);
    background: var(--el-fill-color-light);
    color: var(--el-text-color-secondary);
    font-family: var(--ax-font-mono);
    font-size: 12px;
  }
  .stage-flag {
    flex: none;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }
  .stage-card.is-success .stage-flag {
    color: var(--el-color-success);
  }
  .stage-card.is-error .stage-flag {
    color: var(--el-color-error);
  }
  .stage-card.is-running .stage-flag {
    color: var(--el-color-primary);
  }
  .stage-card.is-warning .stage-flag {
    color: var(--el-color-warning);
  }
  .stage-body {
    padding-top: 8px;
  }
  /* 任务节点 */
  .task-card {
    display: grid;
    grid-template-columns: 22px minmax(0, 1fr) auto;
    align-items: center;
    width: 100%;
    gap: 8px;
    padding: 9px 10px;
    margin-top: 6px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--ax-radius-sm);
    background: var(--ax-surface-card);
    text-align: left;
    cursor: pointer;
    transition:
      border-color 0.16s ease,
      background 0.16s ease,
      transform 0.16s ease;
  }
  .task-card:first-child {
    margin-top: 0;
  }
  .task-card:hover {
    border-color: var(--el-color-primary);
    transform: translateY(-1px);
    box-shadow: var(--ax-shadow-sm);
  }
  .task-card.selected {
    border-color: var(--el-color-primary);
    background: color-mix(in srgb, var(--el-color-primary) 7%, var(--ax-surface-card));
  }
  .task-state {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    border-radius: 50%;
    background: var(--el-fill-color-light);
    color: var(--el-text-color-secondary);
  }
  .task-card.success .task-state {
    background: color-mix(in srgb, var(--el-color-success) 14%, transparent);
    color: var(--el-color-success);
  }
  .task-card.error .task-state {
    background: color-mix(in srgb, var(--el-color-error) 14%, transparent);
    color: var(--el-color-error);
  }
  .task-card.running .task-state {
    background: color-mix(in srgb, var(--el-color-primary) 14%, transparent);
    color: var(--el-color-primary);
  }
  .task-card.warning .task-state {
    background: color-mix(in srgb, var(--el-color-warning) 16%, transparent);
    color: var(--el-color-warning);
  }
  .task-card.running .task-state svg {
    animation: flow-spin 1.1s linear infinite;
  }
  .task-copy {
    display: flex;
    min-width: 0;
    flex-direction: column;
    gap: 2px;
  }
  .task-copy strong {
    overflow: hidden;
    color: var(--el-text-color-primary);
    font-size: 14px;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .task-copy small {
    overflow: hidden;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .task-duration,
  .empty-task {
    color: var(--el-text-color-secondary);
    font-family: var(--ax-font-mono);
    font-size: 12px;
    font-variant-numeric: tabular-nums;
  }
  .empty-task {
    display: block;
    margin-top: 14px;
    font-family: var(--ax-font-sans);
    font-size: 13px;
  }
  @keyframes flow-spin {
    to {
      transform: rotate(360deg);
    }
  }
  @keyframes flow-pulse {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0.35;
    }
  }
  /* 任务快捷切换条 */
  .compact-tasks {
    display: flex;
    gap: 6px;
    padding: 10px 14px;
    border-top: 1px solid var(--el-border-color-lighter);
    background: var(--ax-surface-card);
    overflow-x: auto;
  }
  .task-chip {
    flex: none;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 5px 10px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 999px;
    background: var(--ax-surface-card);
    color: var(--el-text-color-regular);
    font-size: 13px;
    cursor: pointer;
    transition:
      border-color 0.16s ease,
      background 0.16s ease;
  }
  .task-chip small {
    color: var(--el-text-color-secondary);
    font-family: var(--ax-font-mono);
    font-size: 12px;
    font-variant-numeric: tabular-nums;
  }
  .task-chip:hover {
    border-color: var(--el-color-primary);
  }
  .task-chip.selected {
    border-color: var(--el-color-primary);
    background: color-mix(in srgb, var(--el-color-primary) 8%, var(--ax-surface-card));
    color: var(--el-color-primary);
    font-weight: 600;
  }
  .task-detail {
    margin-top: 8px;
    @include cicd-block;
  }
  .task-detail-header {
    @include cicd-block-head;

    .detail-actions {
      display: flex;
      flex: none;
      gap: 4px;
    }
  }
  .task-meta-summary {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 6px 20px;
    padding: 10px 16px;
    border-bottom: 1px solid #eef2f7;
    color: #94a3b8;
    font-size: 11px;
  }
  .task-meta-summary > span {
    max-width: min(40vw, 450px);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .task-meta-summary strong {
    margin-left: 4px;
    color: #334155;
    font-family: var(--ax-font-mono);
    font-size: 12px;
    font-weight: 600;
  }
  .task-meta-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 10px;
    padding: 10px 16px;
    border-bottom: 1px solid #eef2f7;
  }
  .task-meta-grid div {
    display: flex;
    min-width: 0;
    flex-direction: column;
    gap: 2px;
  }
  .task-meta-grid span {
    color: #94a3b8;
    font-size: 11px;
  }
  .task-meta-grid strong {
    overflow: hidden;
    color: #334155;
    font-family: var(--ax-font-mono);
    font-size: 12px;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  @media (max-width: 900px) {
    .task-meta-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }
</style>
