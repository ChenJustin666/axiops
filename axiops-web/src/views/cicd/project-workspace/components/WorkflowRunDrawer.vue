<template>
  <ElDrawer
    v-model="drawerVisible"
    :title="`工作流运行 #${run?.run_number || '—'}`"
    size="min(720px, 100vw)"
    destroy-on-close
  >
    <div v-loading="loading" class="run-detail">
      <div v-if="run" class="run-summary">
        <div>
          <span class="eyebrow">运行状态</span>
          <div class="run-status"
            ><span class="status-dot" :class="run.status" />{{ statusLabel(run.status) }}</div
          >
        </div>
        <div class="run-meta">
          <span>环境 #{{ run.environment_id }}</span>
          <span>定义 v{{ run.definition_version }}</span>
          <span>{{ formatDateTime(run.created_at) }}</span>
          <span>总耗时 {{ taskDuration(run) }}</span>
          <span>执行人 {{ workflowActorLabel(run) }}</span>
        </div>
      </div>

      <section v-if="run" class="run-progress" aria-label="工作流进度">
        <div class="progress-heading">
          <span
            ><strong>交付进度</strong><small>{{ progressCaption }}</small></span
          >
          <strong>{{ progressPercent }}%</strong>
        </div>
        <div class="progress-track"><span :style="{ width: `${progressPercent}%` }" /></div>
        <div class="progress-steps">
          <div
            v-for="task in run.tasks || []"
            :key="task.id"
            :class="['progress-step', task.status]"
          >
            <span class="progress-step-dot" />
            <small>{{ task.node_snapshot?.name || task.node_key }}</small>
          </div>
        </div>
      </section>

      <details v-if="run" class="target-section">
        <summary class="target-summary">
          <strong>{{ runEnvironment?.name || `环境 #${run.environment_id}` }}</strong>
          <span class="target-binding" :title="targetDescription">{{ targetDescription }}</span>
          <ElTag
            v-if="runEnvironment"
            size="small"
            effect="plain"
            :type="runEnvironment.is_production ? 'danger' : 'success'"
            >{{ runEnvironment.is_production ? '生产环境' : '非生产环境' }}</ElTag
          >
          <span class="target-label">详细目标</span>
        </summary>
        <div class="target-grid">
          <div>
            <span class="target-label">环境</span>
            <strong>{{ runEnvironment?.name || `环境 #${run.environment_id}` }}</strong>
          </div>
          <div>
            <span class="target-label">资源绑定</span>
            <span>{{ targetDescription }}</span>
          </div>
          <div>
            <span class="target-label">运行方式</span>
            <span>{{
              run.operation_type === 'rollback'
                ? '回滚'
                : run.operation_type === 'restart'
                  ? '滚动重启'
                  : run.trigger_type || '手动'
            }}</span>
          </div>
          <div v-if="deploymentStrategy">
            <span class="target-label">发布策略</span>
            <span>{{ deploymentStrategy }}</span>
          </div>
        </div>
      </details>

      <ElAlert
        v-if="run?.error_message"
        :title="run.error_message"
        type="error"
        :closable="false"
        show-icon
      />
      <div v-if="run" class="run-actions">
        <ElButton
          v-if="canApprove"
          type="warning"
          :icon="ShieldCheck"
          :loading="actionLoading"
          @click="approve"
          >审批继续</ElButton
        >
        <ElButton
          v-if="canCancel"
          type="danger"
          plain
          :icon="Ban"
          :loading="actionLoading"
          @click="cancel"
          >取消运行</ElButton
        >
        <ElButton :icon="RefreshCw" circle title="刷新运行状态" :loading="loading" @click="load" />
      </div>

      <section v-if="run" class="node-section">
        <div class="section-title"
          ><h3>执行节点</h3
          ><span>{{ completedCount }}/{{ run.tasks?.length || 0 }} 已完成</span></div
        >
        <div class="node-list">
          <div v-for="task in run.tasks || []" :key="task.id" class="node-row">
            <span class="node-icon" :class="task.status"
              ><component :is="statusIcon(task.status)" :size="16"
            /></span>
            <div class="node-main">
              <div class="node-heading"
                ><strong>{{ task.node_snapshot?.name || task.node_key }}</strong
                ><ElTag size="small" effect="plain">{{ nodeLabel(task.node_type) }}</ElTag></div
              >
              <small
                >{{ task.error_message || taskStatusLabel(task.status) }} ·
                {{ taskDuration(task) }}</small
              >
              <small v-if="task.retry_count"
                >已重试 {{ task.retry_count }} 次<template
                  v-if="task.status === 'pending' && task.next_retry_at"
                >
                  · 下次尝试 {{ formatDateTime(task.next_retry_at) }}</template
                ></small
              >
              <details v-if="task.retry_history?.length">
                <summary>查看之前失败的构建</summary>
                <div v-for="(previous, index) in task.retry_history" :key="index">
                  <ElButton
                    text
                    type="primary"
                    @click="openLogs({ ...task, provider_run_id: previous.provider_run_id })"
                    >构建 ID {{ buildId(previous.provider_run_id) }} 日志</ElButton
                  >
                  <small>{{ previous.error_message }}</small>
                </div>
              </details>
              <small v-if="task.started_at"
                >{{ formatDateTime(task.started_at)
                }}<template v-if="task.finished_at">
                  - {{ formatDateTime(task.finished_at) }}</template
                ></small
              >
              <small v-if="task.provider_run_id && !isBuildTask(task)"
                >执行标识 {{ task.provider_run_id }}</small
              >
              <div v-if="isBuildTask(task) && buildDetail(task)" class="build-summary">
                <ElAlert
                  v-if="buildDetail(task)?.error_message"
                  :title="buildDetail(task)?.error_message"
                  :type="
                    ['failed', 'timeout'].includes(buildDetail(task)?.status || '')
                      ? 'error'
                      : 'warning'
                  "
                  :closable="false"
                  show-icon
                />
                <div class="build-summary-line">
                  <strong>{{ buildServiceName(task) }}</strong>
                  <span
                    >构建 #{{
                      buildDetail(task)?.build_number || buildId(task.provider_run_id)
                    }}</span
                  >
                  <ElTag
                    size="small"
                    effect="plain"
                    :type="buildStatusType(buildDetail(task)?.status)"
                  >
                    {{ buildStatusLabel(buildDetail(task)?.status) }}
                  </ElTag>
                </div>
                <BuildCommitLink
                  :commit="buildDetail(task)?.git_commit"
                  :subject="buildDetail(task)?.git_message"
                  :url="buildDetail(task)?.git_commit_url"
                  :author="buildDetail(task)?.git_author"
                />
                <details @toggle="onChangesToggle($event, task)"
                  ><summary>代码变更</summary
                  ><BuildChanges
                    v-if="expandedChanges.includes(buildId(task.provider_run_id))"
                    :build-id="buildId(task.provider_run_id)"
                /></details>
                <div class="build-summary-line build-summary-muted">
                  <span v-if="buildDetail(task)?.git_branch"
                    >分支 {{ buildDetail(task)?.git_branch }}</span
                  >
                </div>
                <div
                  v-if="buildDetail(task)?.image_url || buildDetail(task)?.image_digest"
                  class="build-summary-line build-summary-muted"
                >
                  <span v-if="buildDetail(task)?.image_url" class="build-image"
                    >镜像 {{ buildDetail(task)?.image_url }}</span
                  >
                  <span v-if="buildDetail(task)?.image_digest"
                    >Digest {{ buildDetail(task)?.image_digest }}</span
                  >
                </div>
                <div
                  v-if="
                    buildDetail(task)?.job_name ||
                    buildDetail(task)?.pod_name ||
                    buildDetail(task)?.node_name
                  "
                  class="build-summary-line build-summary-muted"
                >
                  <span v-if="buildDetail(task)?.job_name"
                    >Job {{ buildDetail(task)?.job_name }}</span
                  >
                  <span v-if="buildDetail(task)?.pod_name"
                    >Pod {{ buildDetail(task)?.pod_name }}</span
                  >
                  <span v-if="buildDetail(task)?.node_name"
                    >节点 {{ buildDetail(task)?.node_name }}</span
                  >
                </div>
                <div v-if="buildStages(task).length" class="build-stages">
                  <span
                    v-for="stage in buildStages(task)"
                    :key="stage.name"
                    class="build-stage"
                    :class="`is-${stage.status}`"
                    >{{ stageLabel(stage.name) }} · {{ stageStatusLabel(stage.status) }}</span
                  >
                </div>
              </div>
            </div>
            <ElButton
              v-if="!isBuildTask(task) || buildId(task.provider_run_id)"
              text
              type="primary"
              :icon="FileText"
              @click="openLogs(task)"
              >日志</ElButton
            >
          </div>
        </div>
      </section>
      <ElEmpty v-else-if="!loading" description="暂无运行数据" />

      <ElDialog
        v-model="logsVisible"
        :title="`${logTitle} · 每 3 秒刷新`"
        width="min(1500px, 96vw)"
        top="3vh"
        class="axiops-workflow-build-log"
        destroy-on-close
        append-to-body
      >
        <ElAlert
          v-if="buildDetails[logBuildId]?.error_message"
          :title="buildDetails[logBuildId]?.error_message"
          type="warning"
          :closable="false"
        />
        <ElAlert v-if="logError" :title="logError" type="error" :closable="false" />
        <BuildLogViewer
          v-if="logsVisible"
          :key="`${logTaskId}:${logBuildId}`"
          :text="displayLogText"
          :summary-text="logSummary"
          :title="logTitle"
          :empty-text="
            logsLoading
              ? '正在读取日志…'
              : logTask?.status === 'pending'
                ? '任务尚未开始，开始后显示执行步骤和检查结果。'
                : '尚未保存执行日志；运行中将自动更新。'
          "
          :loading="logsLoading"
          fill
          @refresh="refreshLogs"
        />
      </ElDialog>
    </div>
  </ElDrawer>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, ref, watch } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import {
    Ban,
    CheckCircle2,
    CircleAlert,
    Clock3,
    FileText,
    LoaderCircle,
    RefreshCw,
    ShieldCheck,
    XCircle
  } from 'lucide-vue-next'
  import {
    approveWorkflowRunApi,
    cancelWorkflowRunApi,
    getBuildRunApi,
    getBuildLogsApi,
    getWorkflowRunApi,
    type AxiOpsCluster,
    type AxiOpsCodeSource,
    type AxiOpsEnvironment,
    type AxiOpsService,
    type AxiOpsTaskRun,
    type AxiOpsWorkflowRun
  } from '@/api/axiops'
  import { useAxiOpsAccess } from '@/composables/useAxiOpsAccess'
  import BuildLogViewer from '@/components/build-log-viewer/index.vue'
  import BuildChanges from '@/components/build-changes/index.vue'
  import BuildCommitLink from '@/components/build-commit-link/index.vue'
  import { summarizeDeploymentLog } from '@/utils/deployment-log'
  import { formatExecutionDuration } from '@/utils/execution-duration'
  import { workflowActorLabel } from '@/utils/workflow-actor'

  const props = defineProps<{
    visible: boolean
    runId: number | null
    readOnly?: boolean
    environments?: AxiOpsEnvironment[]
    clusters?: AxiOpsCluster[]
    services?: AxiOpsService[]
    codeSources?: AxiOpsCodeSource[]
  }>()
  const emit = defineEmits<{ 'update:visible': [value: boolean]; changed: [] }>()
  const access = useAxiOpsAccess()
  const drawerVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })
  const run = ref<AxiOpsWorkflowRun | null>(null)
  const loading = ref(false)
  const actionLoading = ref(false)
  const logsLoading = ref(false)
  const logsVisible = ref(false)
  const logText = ref('')
  const logError = ref('')
  const logBuildId = ref(0)
  const logTaskId = ref(0)
  const logTask = computed(() => run.value?.tasks?.find((task) => task.id === logTaskId.value))
  const logTitle = computed(() =>
    logBuildId.value
      ? `构建 #${logBuildId.value} 日志`
      : `${logTask.value?.node_snapshot?.name || logTask.value?.node_key || '任务'} · 执行日志`
  )
  const displayLogText = computed(() =>
    logBuildId.value ? logText.value : logTask.value?.log_text || logTask.value?.error_message || ''
  )
  const logSummary = computed(() =>
    ['yaml_deploy', 'yaml_rollback', 'workload_restart'].includes(logTask.value?.node_type || '') &&
    !logBuildId.value
      ? summarizeDeploymentLog(
          displayLogText.value,
          logTask.value?.status,
          logTask.value?.error_message
        )
      : undefined
  )
  const clockNow = ref(Date.now())
  let clockTimer: number | null = null
  const taskDuration = (task: Pick<AxiOpsTaskRun, 'started_at' | 'finished_at'>) =>
    formatExecutionDuration(task.started_at, task.finished_at, clockNow.value)
  let logRequestInFlight = false
  type BuildRunDetail = Awaited<ReturnType<typeof getBuildRunApi>>
  const buildDetails = ref<Record<number, BuildRunDetail>>({})
  const expandedChanges = ref<number[]>([])
  const onChangesToggle = (event: Event, task: AxiOpsTaskRun) => {
    const id = buildId(task.provider_run_id)
    expandedChanges.value = expandedChanges.value.filter((value) => value !== id)
    if ((event.target as HTMLDetailsElement).open) expandedChanges.value.push(id)
  }
  let refreshTimer: number | null = null
  const activeStatuses = new Set(['pending', 'pending_approval', 'running', 'cancelling'])
  const completedCount = computed(
    () =>
      (run.value?.tasks || []).filter((task) =>
        ['succeeded', 'failed', 'cancelled'].includes(task.status)
      ).length
  )
  const progressPercent = computed(() => {
    const total = run.value?.tasks?.length || 0
    if (!total) return run.value?.status === 'succeeded' ? 100 : 0
    return Math.round((completedCount.value / total) * 100)
  })
  const progressCaption = computed(() => {
    const runningTask = run.value?.tasks?.find((task) => task.status === 'running')
    if (runningTask) return `正在执行：${runningTask.node_snapshot?.name || runningTask.node_key}`
    if (run.value?.status === 'pending_approval') return '等待授权后继续部署'
    if (run.value?.status === 'succeeded')
      return run.value.operation_type === 'restart'
        ? '滚动重启完成，工作负载已就绪'
        : '所有阶段已完成，产物已进入目标环境'
    if (run.value?.status === 'failed') return '执行已暂停，请查看失败节点和日志'
    return `${completedCount.value}/${run.value?.tasks?.length || 0} 个阶段已完成`
  })
  const canCancel = computed(() =>
    Boolean(
      !props.readOnly &&
        run.value &&
        activeStatuses.has(run.value.status) &&
        access.can('workflow', 'cancel', {
          type: 'workflow',
          id: run.value.workflow_id,
          project_id: run.value.project_id,
          environment_id: run.value.environment_id,
          environment: props.environments?.find((item) => item.id === run.value?.environment_id)
            ?.code
        })
    )
  )
  const canApprove = computed(() =>
    Boolean(
      !props.readOnly &&
        run.value?.status === 'pending_approval' &&
        access.can('workflow', 'approve', {
          type: 'workflow',
          id: run.value.workflow_id,
          project_id: run.value.project_id,
          environment_id: run.value.environment_id,
          environment: props.environments?.find((item) => item.id === run.value?.environment_id)
            ?.code
        })
    )
  )
  const runEnvironment = computed(() =>
    props.environments?.find((item) => item.id === run.value?.environment_id)
  )
  const runCluster = computed(() =>
    runEnvironment.value
      ? props.clusters?.find((item) => item.id === runEnvironment.value?.cluster_id)
      : undefined
  )
  const targetDescription = computed(() => {
    const environment = runEnvironment.value
    if (!environment) return `环境 #${run.value?.environment_id || '-'}`
    if (environment.environment_type === 'host_group')
      return environment.host_group_ref || '主机组未配置'
    const cluster = runCluster.value?.name || `集群 #${environment.cluster_id}`
    return `${cluster} / ${environment.namespace || 'Namespace 未配置'}`
  })
  const deploymentStrategy = computed(() => {
    const value = String(run.value?.context_snapshot?.deployment_strategy || '')
    return { rolling: '滚动发布', canary: '金丝雀发布', bluegreen: '蓝绿发布' }[value] || ''
  })

  let loadGeneration = 0
  const load = async () => {
    if (!props.runId || !props.visible) return
    const id = props.runId,
      current = ++loadGeneration
    loading.value = true
    try {
      const nextRun = await getWorkflowRunApi(id)
      if (current !== loadGeneration || !props.visible || props.runId !== id) return
      if (nextRun.id !== id) throw Error('执行记录返回范围不一致')
      run.value = nextRun
      await loadBuildDetails(nextRun, current)
    } finally {
      if (current === loadGeneration) loading.value = false
    }
  }
  const loadBuildDetails = async (value: AxiOpsWorkflowRun, current: number) => {
    const ids = Array.from(
      new Set(
        (value.tasks || [])
          .filter((task) => isBuildTask(task))
          .map((task) => buildId(task.provider_run_id))
          .filter((id) => id > 0)
      )
    )
    const next: Record<number, BuildRunDetail> = {}
    await Promise.all(
      ids.map(async (id) => {
        try {
          next[id] = await getBuildRunApi(id)
        } catch {
          // The build row can be created just after the workflow task; the
          // next polling cycle will retry without interrupting run details.
        }
      })
    )
    if (current === loadGeneration) buildDetails.value = next
  }
  const startPolling = () => {
    if (refreshTimer) clearInterval(refreshTimer)
    refreshTimer = window.setInterval(async () => {
      if (run.value && activeStatuses.has(run.value.status)) await load()
      if (logsVisible.value && logBuildId.value) await refreshLogs()
    }, 3000)
  }
  const approve = async () => {
    if (!run.value) return
    actionLoading.value = true
    try {
      await approveWorkflowRunApi(run.value.id)
      ElMessage.success('审批已通过')
      await load()
      emit('changed')
    } finally {
      actionLoading.value = false
    }
  }
  const cancel = async () => {
    if (!run.value) return
    try {
      await ElMessageBox.confirm(
        '停止后续节点并取消运行中的构建。已提交的部署会保留资源并等待状态确认，不会自动删除服务或回滚。确定取消？',
        '取消运行',
        {
          type: 'warning',
          confirmButtonText: '取消运行',
          cancelButtonText: '返回'
        }
      )
    } catch {
      return
    }
    actionLoading.value = true
    try {
      await cancelWorkflowRunApi(run.value.id)
      ElMessage.success('已提交取消请求')
      await load()
      emit('changed')
    } finally {
      actionLoading.value = false
    }
  }
  const openLogs = async (task: AxiOpsTaskRun) => {
    const id = isBuildTask(task) ? buildId(task.provider_run_id) : 0
    logTaskId.value = task.id
    logsVisible.value = true
    logBuildId.value = id
    logsLoading.value = true
    logText.value = ''
    logError.value = ''
    await refreshLogs()
  }
  const refreshLogs = async () => {
    const id = logBuildId.value
    if (!id && logTaskId.value) {
      try {
        await load()
        logError.value = ''
      } catch (error: any) {
        logError.value = error?.message || '执行日志读取失败，请重试'
      } finally {
        logsLoading.value = false
      }
      return
    }
    if (!id || logRequestInFlight) return
    logRequestInFlight = true
    try {
      const logs = await getBuildLogsApi(id)
      if (id !== logBuildId.value) return
      logText.value = (logs || [])
        .map((item) => item.log_text || '')
        .filter(Boolean)
        .join('\n')
      logError.value = ''
    } catch (error: any) {
      if (id === logBuildId.value) logError.value = error?.message || '日志加载失败，请稍后重试'
    } finally {
      logRequestInFlight = false
      logsLoading.value = false
    }
  }
  const buildId = (value: string) => {
    const match = /^build:(\d+)$/.exec(value || '')
    return match ? Number(match[1]) : 0
  }
  const isBuildTask = (task: AxiOpsTaskRun) => ['build', 'test'].includes(task.node_type)
  const buildDetail = (task: AxiOpsTaskRun) => {
    const id = buildId(task.provider_run_id)
    return id ? buildDetails.value[id] : undefined
  }
  const buildServiceName = (task: AxiOpsTaskRun) => {
    const detail = buildDetail(task)
    if (detail?.service_id) {
      return (
        props.services?.find((item) => item.id === detail.service_id)?.name ||
        `服务 #${detail.service_id}`
      )
    }
    return '未知服务'
  }
  const buildStages = (task: AxiOpsTaskRun) =>
    (buildDetail(task)?.stages_parsed || []) as Array<{ name: string; status: string }>
  const buildStatusLabel = (status?: string) =>
    ({
      queued: '排队中',
      running: '执行中',
      success: '成功',
      failed: '失败',
      timeout: '超时',
      cancelled: '已取消',
      cancelling: '取消中'
    })[status || ''] ||
    status ||
    '未知'
  type BuildStatusTag = 'success' | 'danger' | 'warning' | 'primary' | 'info'
  const buildStatusType = (status?: string): BuildStatusTag =>
    (({
      success: 'success',
      failed: 'danger',
      timeout: 'danger',
      running: 'primary',
      cancelling: 'warning',
      cancelled: 'info'
    })[status || ''] || 'info') as BuildStatusTag
  const stageLabel = (name: string) =>
    ({ git_clone: '拉取代码', build: '构建', image_build: '制作镜像', image_push: '推送镜像' })[
      name
    ] || name
  const stageStatusLabel = (status: string) =>
    ({ pending: '等待', running: '执行中', success: '完成', failed: '失败', skipped: '跳过' })[
      status
    ] || status
  const statusIcon = (status: string) =>
    ({
      succeeded: CheckCircle2,
      failed: XCircle,
      cancelled: CircleAlert,
      running: LoaderCircle,
      pending_approval: ShieldCheck
    })[status] || Clock3
  const statusLabel = (status: string) =>
    ({
      pending: '排队中',
      pending_approval: '等待审批',
      running: '执行中',
      succeeded: '成功',
      failed: '失败',
      cancelling: '取消中',
      cancelled: '已取消'
    })[status] || status
  const taskStatusLabel = (status: string) => statusLabel(status)
  const nodeLabel = (type: string) =>
    ({
      build: '构建',
      yaml_deploy: 'K8s YAML 部署',
      workload_restart: '滚动重启',
      helm_deploy: 'Helm 部署',
      host_deploy: '主机部署',
      test: '测试',
      approval: '审批',
      notify: '通知'
    })[type] || type
  const formatDateTime = (value?: string) =>
    value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'
  watch(
    () => [props.visible, props.runId],
    async ([visible]) => {
      const id = props.runId
      loadGeneration++
      run.value = null
      buildDetails.value = {}
      if (refreshTimer) {
        clearInterval(refreshTimer)
        refreshTimer = null
      }
      logsVisible.value = false
      logTaskId.value = 0
      if (clockTimer) {
        clearInterval(clockTimer)
        clockTimer = null
      }
      if (visible) {
        clockNow.value = Date.now()
        clockTimer = window.setInterval(() => {
          clockNow.value = Date.now()
        }, 1000)
        await load()
        if (props.visible && props.runId === id) startPolling()
      } else if (refreshTimer) {
        clearInterval(refreshTimer)
        refreshTimer = null
      }
    },
    { immediate: true }
  )
  onBeforeUnmount(() => {
    loadGeneration++
    if (refreshTimer) clearInterval(refreshTimer)
    if (clockTimer) clearInterval(clockTimer)
  })
</script>

<style scoped lang="scss">
  .run-detail {
    min-height: 100%;
  }
  .run-summary {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    padding-bottom: 18px;
    border-bottom: 1px solid var(--art-border-color);
  }
  .eyebrow {
    color: var(--art-text-gray-500);
    font-size: 12px;
  }
  .run-status {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 7px;
    font-size: 18px;
    font-weight: 600;
  }
  .run-meta {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 5px;
    color: var(--art-text-gray-500);
    font-size: 12px;
  }
  .run-progress {
    margin-top: 18px;
    padding: 13px 14px 11px;
    border: 1px solid #d9e5f8;
    border-radius: 7px;
    background: #f8fbff;
  }
  .progress-heading,
  .progress-heading > span,
  .progress-steps,
  .progress-step {
    display: flex;
    align-items: center;
  }
  .progress-heading {
    justify-content: space-between;
    gap: 12px;
  }
  .progress-heading > span {
    min-width: 0;
    flex-direction: column;
    align-items: flex-start;
    gap: 3px;
  }
  .progress-heading strong {
    font-size: 12px;
  }
  .progress-heading > strong {
    color: #2563eb;
    font-size: 15px;
  }
  .progress-heading small {
    max-width: 470px;
    overflow: hidden;
    color: #64748b;
    font-size: 10px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .progress-track {
    height: 5px;
    margin-top: 10px;
    overflow: hidden;
    border-radius: 99px;
    background: #dbeafe;
  }
  .progress-track span {
    display: block;
    height: 100%;
    border-radius: inherit;
    background: linear-gradient(90deg, #2563eb, #14b8a6);
    transition: width 0.35s ease;
  }
  .progress-steps {
    gap: 0;
    margin-top: 11px;
    overflow-x: auto;
  }
  .progress-step {
    min-width: 0;
    flex: 1;
    gap: 5px;
    color: #94a3b8;
  }
  .progress-step + .progress-step::before {
    width: 18px;
    height: 1px;
    margin-right: 5px;
    background: #dbe3ef;
    content: '';
  }
  .progress-step-dot {
    width: 7px;
    height: 7px;
    flex: 0 0 7px;
    border: 2px solid #cbd5e1;
    border-radius: 50%;
    background: #fff;
  }
  .progress-step small {
    max-width: 90px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 10px;
  }
  .progress-step.succeeded {
    color: #15803d;
  }
  .progress-step.succeeded .progress-step-dot {
    border-color: #16a34a;
    background: #16a34a;
  }
  .progress-step.running {
    color: #2563eb;
  }
  .progress-step.running .progress-step-dot {
    border-color: #2563eb;
    background: #bfdbfe;
    box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.12);
  }
  .progress-step.failed {
    color: #dc2626;
  }
  .progress-step.failed .progress-step-dot {
    border-color: #dc2626;
    background: #fecaca;
  }
  .target-section {
    margin-top: 10px;
    padding-bottom: 8px;
    border-bottom: 1px solid var(--art-border-color);
  }
  .target-summary {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 7px 0;
    cursor: pointer;
    font-size: 12px;
  }
  .target-summary strong {
    flex: none;
  }
  .target-binding {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--art-text-gray-500);
  }
  .target-grid {
    display: grid;
    grid-template-columns: 0.8fr 1.4fr 0.8fr;
    gap: 8px;
    padding: 8px 10px;
    font-size: 12px;
    border: 1px solid var(--art-border-color);
    border-radius: 6px;
    background: var(--art-bg-color);
  }
  .target-grid > div {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .target-label {
    color: var(--art-text-gray-500);
    font-size: 11px;
  }
  .target-grid strong,
  .target-grid span:not(.target-label) {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .status-dot {
    width: 9px;
    height: 9px;
    border-radius: 50%;
    background: #94a3b8;
  }
  .status-dot.running {
    background: #2563eb;
  }
  .status-dot.succeeded {
    background: #16805b;
  }
  .status-dot.failed {
    background: #dc2626;
  }
  .status-dot.pending_approval {
    background: #d97706;
  }
  .run-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin: 16px 0;
  }
  .node-section {
    margin-top: 20px;
  }
  .section-title {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 9px;
  }
  .section-title h3 {
    margin: 0;
    font-size: 15px;
  }
  .section-title span {
    color: var(--art-text-gray-500);
    font-size: 12px;
  }
  .node-list {
    border-top: 1px solid var(--art-border-color);
  }
  .node-row {
    display: flex;
    align-items: center;
    gap: 11px;
    min-height: 72px;
    border-bottom: 1px solid var(--art-border-color);
  }
  .node-icon {
    width: 30px;
    height: 30px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
    color: #64748b;
    background: #f1f5f9;
  }
  .node-icon.running {
    color: #2563eb;
    background: #eff6ff;
  }
  .node-icon.succeeded {
    color: #16805b;
    background: #ecfdf5;
  }
  .node-icon.failed {
    color: #dc2626;
    background: #fef2f2;
  }
  .node-icon.pending_approval {
    color: #b45309;
    background: #fffbeb;
  }
  .node-main {
    min-width: 0;
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .node-heading {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .node-main small {
    overflow: hidden;
    color: var(--art-text-gray-500);
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .build-summary {
    display: flex;
    flex-direction: column;
    gap: 5px;
    margin-top: 7px;
    padding: 8px 10px;
    border-left: 2px solid var(--el-color-primary-light-5);
    background: var(--art-bg-color);
    font-size: 12px;
  }
  .build-summary-line {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
  }
  .build-summary-muted {
    color: var(--art-text-gray-500);
  }
  .build-summary a {
    color: var(--el-color-primary);
    text-decoration: none;
  }
  .build-summary a:hover {
    text-decoration: underline;
  }
  .build-image {
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .build-stages {
    display: flex;
    flex-wrap: wrap;
    gap: 5px;
  }
  .build-stage {
    padding: 2px 6px;
    border-radius: 3px;
    color: var(--art-text-gray-500);
    background: #f1f5f9;
    font-size: 11px;
  }
  .build-stage.is-running {
    color: #1d4ed8;
    background: #eff6ff;
  }
  .build-stage.is-success {
    color: #166534;
    background: #ecfdf5;
  }
  .build-stage.is-failed {
    color: #b91c1c;
    background: #fef2f2;
  }
  :global(.axiops-workflow-build-log) {
    height: 92vh;
    height: 92dvh;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    max-width: 96vw;
    min-width: min(600px, 96vw);
  }
  :global(.axiops-workflow-build-log .el-dialog__body) {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
    gap: 8px;
  }
  :global(.axiops-workflow-build-log .el-dialog__header) {
    flex: none;
    padding-bottom: 8px;
  }
  :global(.axiops-workflow-build-log .el-dialog__title) {
    font-size: 14px;
  }
  :global(.axiops-workflow-build-log .el-dialog__body > .el-alert) {
    flex: none;
    max-height: 90px;
    overflow: auto;
  }
  @media (max-width: 520px) {
    .run-summary {
      flex-direction: column;
    }
    .run-meta {
      align-items: flex-start;
    }
    .target-grid {
      grid-template-columns: 1fr;
      gap: 9px;
    }
  }
</style>
