<template>
  <div v-loading="loading" class="ops-dashboard">
    <header class="dashboard-header">
      <div>
        <span class="eyebrow">AXIOPS DELIVERY CONTROL</span>
        <h1>交付驾驶舱</h1>
        <p>在一个工作面掌握可访问项目、环境目标和最近交付活动。</p>
      </div>
      <div class="header-actions">
        <ElButton
          :icon="RefreshCw"
          circle
          title="刷新驾驶舱"
          :loading="loading"
          @click="loadDashboard"
        />
        <ElButton v-if="canCreateProject" type="primary" :icon="FolderPlus" @click="goToProjects">
          新建项目
        </ElButton>
      </div>
    </header>

    <section class="dashboard-toolbar" aria-label="驾驶舱筛选">
      <ElInput
        v-model="filters.keyword"
        clearable
        :prefix-icon="Search"
        placeholder="搜索项目、服务或编码"
        class="search-input"
      />
      <ElSelect v-model="filters.deliveryType" clearable placeholder="交付类型" class="type-input">
        <ElOption
          v-for="item in deliveryOptions"
          :key="item.value"
          :label="item.label"
          :value="item.value"
        />
      </ElSelect>
      <ElSelect v-model="filters.status" clearable placeholder="项目状态" class="status-input">
        <ElOption label="启用" value="active" />
        <ElOption label="停用" value="disabled" />
      </ElSelect>
      <span class="toolbar-hint">{{ filteredProjects.length }} 个可访问项目</span>
    </section>

    <section class="metric-strip" aria-label="交付统计">
      <article class="metric-item">
        <span class="metric-icon is-blue"><FolderKanban :size="18" /></span>
        <span
          ><small>可访问项目</small><strong>{{ filteredProjects.length }}</strong></span
        >
      </article>
      <article class="metric-item">
        <span class="metric-icon is-teal"><Boxes :size="18" /></span>
        <span
          ><small>当前项目服务</small><strong>{{ services.length }}</strong></span
        >
      </article>
      <article class="metric-item">
        <span class="metric-icon is-orange"><Server :size="18" /></span>
        <span
          ><small>已配置环境</small><strong>{{ environments.length }}</strong></span
        >
      </article>
      <article class="metric-item">
        <span class="metric-icon is-red"><Activity :size="18" /></span>
        <span
          ><small>进行中的运行</small><strong>{{ runningCount }}</strong></span
        >
      </article>
    </section>

    <div class="dashboard-layout">
      <aside class="project-panel">
        <div class="panel-heading">
          <div><span class="panel-eyebrow">PROJECTS</span><strong>项目目录</strong></div>
          <span class="panel-count">{{ filteredProjects.length }}</span>
        </div>
        <div class="project-list">
          <button
            v-for="project in filteredProjects"
            :key="project.id"
            type="button"
            :class="['project-item', { 'is-active': project.id === selectedProjectId }]"
            @click="selectProject(project.id)"
          >
            <span class="project-type" :class="`is-${project.delivery_type}`"
              ><component :is="deliveryMeta(project.delivery_type).icon" :size="16"
            /></span>
            <span class="project-copy"
              ><strong>{{ project.name }}</strong
              ><small>{{ project.code }}</small></span
            >
            <span class="project-state" :class="project.status"
              ><span class="state-dot" />{{ project.status === 'active' ? '启用' : '停用' }}</span
            >
          </button>
          <ElEmpty v-if="!filteredProjects.length" :image-size="46" description="暂无可访问项目" />
        </div>
        <div class="panel-footnote"
          ><ShieldCheck :size="14" /><span>项目、服务和环境均按授权范围过滤。</span></div
        >
      </aside>

      <main class="delivery-panel">
        <template v-if="selectedProject">
          <header class="delivery-heading">
            <div class="delivery-identity">
              <span class="project-type large" :class="`is-${selectedProject.delivery_type}`"
                ><component :is="deliveryMeta(selectedProject.delivery_type).icon" :size="20"
              /></span>
              <div
                ><div class="title-line"
                  ><h2>{{ selectedProject.name }}</h2
                  ><ElTag
                    size="small"
                    effect="plain"
                    :type="deliveryMeta(selectedProject.delivery_type).tag"
                    >{{ deliveryMeta(selectedProject.delivery_type).label }}</ElTag
                  ></div
                ><span>{{ selectedProject.description || '暂无项目描述' }}</span></div
              >
            </div>
            <ElButton
              text
              type="primary"
              :icon="ArrowUpRight"
              @click="openProject(selectedProject.id)"
              >进入工作台</ElButton
            >
          </header>

          <div class="board-toolbar">
            <div><strong>服务交付矩阵</strong><span>服务基线与环境目标一屏对照</span></div>
            <div class="board-actions">
              <ElButton
                size="small"
                :icon="Rocket"
                :disabled="!services.length"
                @click="openProject(selectedProject.id)"
                >进入发布</ElButton
              >
              <ElButton
                size="small"
                text
                :icon="RefreshCw"
                :loading="projectLoading"
                @click="loadProjectData(selectedProject.id)"
                >刷新</ElButton
              >
            </div>
          </div>

          <div class="matrix-wrap">
            <ElTable :data="serviceRows" row-key="id" height="100%" class="delivery-table">
              <ElTableColumn label="服务" min-width="210" fixed>
                <template #default="{ row }">
                  <div class="service-cell"
                    ><span class="service-type" :class="`is-${row.delivery_type}`"
                      ><component :is="serviceIcon(row.delivery_type)" :size="15" /></span
                    ><span
                      ><strong>{{ row.name }}</strong
                      ><small>{{ row.code }}</small></span
                    ></div
                  >
                </template>
              </ElTableColumn>
              <ElTableColumn label="当前版本" min-width="190">
                <template #default="{ row }">
                  <div class="version-cell"
                    ><span
                      ><ElTag size="small" effect="plain">{{
                        row.current_revision_id ? `R${row.current_revision_id}` : '未定义'
                      }}</ElTag
                      ><BuildCommitLink
                        :commit="latestBuild(row.id)?.git_commit"
                        :url="latestBuild(row.id)?.git_commit_url"
                        :subject="latestBuild(row.id)?.git_message"
                        :author="latestBuild(row.id)?.git_author" /></span
                    ><small>{{
                      latestBuild(row.id)
                        ? `构建 #${latestBuild(row.id)?.build_number}`
                        : '尚无构建产物'
                    }}</small></div
                  >
                </template>
              </ElTableColumn>
              <ElTableColumn label="构建产物" min-width="190">
                <template #default="{ row }"
                  ><div class="artifact-cell"
                    ><code>{{
                      latestBuild(row.id)?.image_digest ||
                      latestBuild(row.id)?.image_url ||
                      row.image_name ||
                      '等待构建输出'
                    }}</code
                    ><small>{{ buildStatusLabel(latestBuild(row.id)?.status) }}</small></div
                  ></template
                >
              </ElTableColumn>
              <ElTableColumn
                v-for="environment in environments"
                :key="environment.id"
                :label="environment.name"
                min-width="156"
              >
                <template #header
                  ><div class="environment-heading"
                    ><span>{{ environment.name }}</span
                    ><ElTag
                      v-if="environment.is_production"
                      size="small"
                      type="danger"
                      effect="plain"
                      >生产</ElTag
                    ></div
                  ></template
                >
                <template #default="{ row }"
                  ><button
                    type="button"
                    :class="['environment-cell', environmentState(row, environment).className]"
                    :title="environmentState(row, environment).hint"
                    @click="openProject(selectedProject!.id)"
                    ><span
                      ><i class="state-dot" /><strong>{{
                        environmentState(row, environment).label
                      }}</strong></span
                    ><small>{{ environmentTarget(environment) }}</small></button
                  ></template
                >
              </ElTableColumn>
              <ElTableColumn label="最近更新" width="150" fixed="right"
                ><template #default="{ row }">{{
                  formatDate(row.updated_at)
                }}</template></ElTableColumn
              >
            </ElTable>
            <ElEmpty
              v-if="!projectLoading && !serviceRows.length"
              :image-size="60"
              description="项目尚未配置服务"
            />
          </div>
        </template>
        <div v-else class="empty-selection"
          ><div class="empty-icon"><FolderKanban :size="24" /></div><strong>选择一个项目开始</strong
          ><p>左侧项目目录会根据当前账号权限过滤，选择后可查看服务、环境和运行状态。</p></div
        >
      </main>

      <aside class="activity-panel">
        <div class="panel-heading"
          ><div><span class="panel-eyebrow">OPERATIONS</span><strong>操作与活动</strong></div></div
        >
        <div class="quick-actions">
          <button
            type="button"
            class="quick-action is-primary"
            @click="selectedProject ? openProject(selectedProject.id) : goToProjects()"
            ><span><Rocket :size="17" /></span
            ><strong>{{ selectedProject ? '进入快速发布' : '选择项目发布' }}</strong
            ><small>从工作台选择服务、Revision 和环境</small><ArrowUpRight :size="15"
          /></button>
          <button type="button" class="quick-action" @click="goToCodeSources"
            ><span><GitBranch :size="17" /></span><strong>管理代码源</strong
            ><small>统一维护 Gitea、GitLab、GitHub 等连接</small><ArrowUpRight :size="15"
          /></button>
          <button type="button" class="quick-action" @click="goToClusters"
            ><span><Cloud :size="17" /></span><strong>检查集群</strong
            ><small>查看连接状态、版本和资源容量</small><ArrowUpRight :size="15"
          /></button>
        </div>

        <div class="activity-section">
          <div class="section-heading"
            ><strong>最近运行</strong><span>{{ workflowRuns.length }} 条</span></div
          >
          <div class="run-list">
            <button
              v-for="run in recentRuns"
              :key="run.id"
              type="button"
              class="run-item"
              @click="openProject(selectedProject!.id)"
              ><span class="run-status" :class="run.status"
                ><component :is="runIcon(run.status)" :size="14" /></span
              ><span
                ><strong>#{{ run.run_number || '—' }} · {{ workflowName(run.workflow_id) }}</strong
                ><small
                  >{{ environmentName(run.environment_id) }} ·
                  {{ formatDate(run.created_at) }}</small
                ></span
              ><ElTag size="small" effect="plain" :type="runStatusType(run.status)">{{
                runStatusLabel(run.status)
              }}</ElTag></button
            ><ElEmpty v-if="!recentRuns.length" :image-size="40" description="暂无运行记录" />
          </div>
        </div>

        <div class="activity-section failures"
          ><div class="section-heading"
            ><strong>需要关注</strong><span>{{ failedRuns.length }} 条失败</span></div
          ><div v-if="failedRuns.length" class="failure-list"
            ><div v-for="run in failedRuns.slice(0, 4)" :key="run.id" class="failure-item"
              ><CircleAlert :size="15" /><span
                ><strong>运行 #{{ run.run_number || '—' }} 未成功</strong
                ><small>{{ run.error_message || '打开运行详情查看失败节点和日志' }}</small></span
              ></div
            ></div
          ><div v-else class="all-clear"
            ><CheckCircle2 :size="17" /><span>当前项目没有失败运行</span></div
          ></div
        >
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
  import BuildCommitLink from '@/components/build-commit-link/index.vue'
  import { computed, onMounted, reactive, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import {
    Activity,
    ArrowUpRight,
    Boxes,
    CheckCircle2,
    CircleAlert,
    Cloud,
    Container,
    FileCode2,
    FolderKanban,
    FolderPlus,
    GitBranch,
    PackageOpen,
    RefreshCw,
    Rocket,
    Search,
    Server,
    ShieldCheck,
    Timer
  } from 'lucide-vue-next'
  import {
    listAllServicesApi,
    listBuildRunsApi,
    listEnvironmentsApi,
    listProjectsApi,
    listWorkflowRunsApi,
    listWorkflowsApi,
    type AxiOpsBuildRun,
    type AxiOpsEnvironment,
    type AxiOpsProject,
    type AxiOpsService,
    type AxiOpsWorkflow,
    type AxiOpsWorkflowRun,
    type DeliveryType
  } from '@/api/axiops'
  import { useAxiOpsAccess } from '@/composables/useAxiOpsAccess'

  defineOptions({ name: 'CICDDashboard' })
  const router = useRouter()
  const access = useAxiOpsAccess()
  const loading = ref(false)
  const projectLoading = ref(false)
  const projects = ref<AxiOpsProject[]>([])
  const selectedProjectId = ref(0)
  const services = ref<AxiOpsService[]>([])
  const environments = ref<AxiOpsEnvironment[]>([])
  const workflows = ref<AxiOpsWorkflow[]>([])
  const workflowRuns = ref<AxiOpsWorkflowRun[]>([])
  const buildRuns = ref<AxiOpsBuildRun[]>([])
  const filters = reactive({ keyword: '', deliveryType: '' as DeliveryType | '', status: '' })
  const canCreateProject = computed(() => access.can('project', 'create'))

  const deliveryOptions = [
    { value: 'k8s_yaml' as const, label: 'K8s YAML', icon: FileCode2, tag: 'primary' as const },
    { value: 'helm' as const, label: 'Helm Chart', icon: PackageOpen, tag: 'warning' as const },
    { value: 'host' as const, label: '主机应用', icon: Container, tag: 'success' as const }
  ]
  const filteredProjects = computed(() => {
    const keyword = filters.keyword.trim().toLowerCase()
    return projects.value.filter((item) => {
      const matchesKeyword = !keyword || `${item.name} ${item.code}`.toLowerCase().includes(keyword)
      const matchesType = !filters.deliveryType || item.delivery_type === filters.deliveryType
      const matchesStatus = !filters.status || item.status === filters.status
      return matchesKeyword && matchesType && matchesStatus
    })
  })
  const selectedProject = computed(() =>
    projects.value.find((item) => item.id === selectedProjectId.value)
  )
  const runningCount = computed(
    () =>
      workflowRuns.value.filter((run) =>
        ['pending', 'pending_approval', 'running', 'cancelling'].includes(run.status)
      ).length
  )
  const failedRuns = computed(() => workflowRuns.value.filter((run) => run.status === 'failed'))
  const recentRuns = computed(() =>
    [...workflowRuns.value]
      .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
      .slice(0, 8)
  )
  const serviceRows = computed(() => services.value)
  const buildByService = computed(() => {
    const result = new Map<number, AxiOpsBuildRun>()
    for (const run of [...buildRuns.value].sort(
      (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    )) {
      if (!result.has(run.service_id)) result.set(run.service_id, run)
    }
    return result
  })

  const deliveryMeta = (type: string) =>
    deliveryOptions.find((item) => item.value === type) || {
      label: '待确认',
      icon: FileCode2,
      tag: 'info' as const
    }
  const serviceIcon = (type: string) =>
    type === 'helm' ? PackageOpen : type === 'host' ? Container : FileCode2
  const latestBuild = (serviceId: number) => buildByService.value.get(serviceId)
  const environmentTarget = (environment: AxiOpsEnvironment) =>
    environment.environment_type === 'host_group'
      ? environment.host_group_ref || '主机组未配置'
      : `集群 #${environment.cluster_id} / ${environment.namespace || 'Namespace 未配置'}`
  const environmentState = (service: AxiOpsService, environment: AxiOpsEnvironment) => {
    if (!service.current_revision_id)
      return { label: '待定义', className: 'is-pending', hint: '服务尚未创建可发布 Revision' }
    if (environment.binding_status !== 'validated')
      return { label: '待验证', className: 'is-pending', hint: '环境目标尚未完成连接验证' }
    if (environment.is_production)
      return { label: '生产目标', className: 'is-production', hint: '发布到生产环境需要审批' }
    return { label: '可发布', className: 'is-ready', hint: '环境目标已验证，可以进入发布流程' }
  }
  const formatDate = (value?: string) =>
    value
      ? new Date(value).toLocaleString('zh-CN', {
          hour12: false,
          month: '2-digit',
          day: '2-digit',
          hour: '2-digit',
          minute: '2-digit'
        })
      : '-'
  const buildStatusLabel = (status?: string) =>
    ({
      success: '构建成功',
      failed: '构建失败',
      running: '构建中',
      queued: '排队中',
      cancelled: '已取消',
      timeout: '已超时'
    })[status || ''] || (status ? status : '暂无构建')
  const runStatusLabel = (status: string) =>
    ({
      pending: '排队中',
      pending_approval: '待审批',
      running: '执行中',
      succeeded: '成功',
      failed: '失败',
      cancelling: '取消中',
      cancelled: '已取消'
    })[status] || status
  const runStatusType = (status: string): 'success' | 'danger' | 'warning' | 'primary' | 'info' =>
    (({
      succeeded: 'success',
      failed: 'danger',
      pending_approval: 'warning',
      running: 'primary',
      pending: 'info',
      cancelling: 'warning',
      cancelled: 'info'
    })[status] || 'info') as 'success' | 'danger' | 'warning' | 'primary' | 'info'
  const runIcon = (status: string) =>
    status === 'succeeded'
      ? CheckCircle2
      : status === 'failed'
        ? CircleAlert
        : status === 'running'
          ? Activity
          : Timer
  const workflowName = (id: number) =>
    workflows.value.find((item) => item.id === id)?.name || `工作流 #${id}`
  const environmentName = (id: number) =>
    environments.value.find((item) => item.id === id)?.name || `环境 #${id}`

  const loadProjectData = async (projectId: number) => {
    selectedProjectId.value = projectId
    projectLoading.value = true
    services.value = []
    environments.value = []
    workflows.value = []
    workflowRuns.value = []
    buildRuns.value = []
    try {
      const [serviceResult, environmentResult, workflowResult, buildResult] =
        await Promise.allSettled([
          listAllServicesApi(projectId),
          listEnvironmentsApi(projectId),
          listWorkflowsApi(projectId),
          listBuildRunsApi(projectId)
        ])
      if (serviceResult.status === 'fulfilled') services.value = serviceResult.value || []
      if (environmentResult.status === 'fulfilled')
        environments.value = environmentResult.value || []
      if (workflowResult.status === 'fulfilled') workflows.value = workflowResult.value || []
      if (buildResult.status === 'fulfilled') buildRuns.value = buildResult.value.list || []
      const runResults = await Promise.allSettled(
        workflows.value
          .slice(0, 12)
          .map((workflow) => listWorkflowRunsApi(workflow.id, { page: 1, page_size: 20 }))
      )
      workflowRuns.value = runResults.flatMap((result) =>
        result.status === 'fulfilled' ? result.value.list || [] : []
      )
    } finally {
      projectLoading.value = false
    }
  }
  const loadDashboard = async () => {
    loading.value = true
    try {
      const result = await listProjectsApi({ page: 1, page_size: 100 })
      projects.value = result.list || []
      const current = projects.value.some((item) => item.id === selectedProjectId.value)
        ? selectedProjectId.value
        : filteredProjects.value[0]?.id || 0
      if (current) await loadProjectData(current)
      else {
        selectedProjectId.value = 0
        services.value = []
        environments.value = []
        workflows.value = []
        workflowRuns.value = []
        buildRuns.value = []
      }
    } finally {
      loading.value = false
    }
  }
  const selectProject = (id: number) => {
    void loadProjectData(id)
  }
  const openProject = (id: number) => {
    void router.push(`/cicd/projects/${id}`)
  }
  const goToProjects = () => {
    void router.push('/cicd/projects')
  }
  const goToCodeSources = () => {
    void router.push('/cicd/code-sources')
  }
  const goToClusters = () => {
    void router.push('/clusters/management')
  }

  onMounted(async () => {
    await access.load()
    await loadDashboard()
  })
</script>

<style scoped lang="scss">
  .ops-dashboard {
    min-height: 100%;
    padding: 22px 24px 28px;
    background: var(--art-main-bg-color);
    color: var(--art-text-gray-900);
  }
  .dashboard-header,
  .header-actions,
  .dashboard-toolbar,
  .delivery-heading,
  .delivery-identity,
  .title-line,
  .board-toolbar,
  .board-actions,
  .panel-heading,
  .section-heading,
  .service-cell,
  .version-cell span,
  .environment-heading,
  .run-item,
  .failure-item,
  .quick-action {
    display: flex;
    align-items: center;
  }
  .dashboard-header {
    justify-content: space-between;
    gap: 20px;
    margin-bottom: 18px;
  }
  .eyebrow,
  .panel-eyebrow {
    color: var(--el-color-primary);
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.14em;
  }
  .dashboard-header h1 {
    margin: 6px 0 4px;
    font-size: 25px;
    line-height: 32px;
    letter-spacing: 0;
  }
  .dashboard-header p {
    margin: 0;
    color: var(--art-text-gray-600);
    font-size: 13px;
  }
  .header-actions {
    gap: 9px;
  }
  .dashboard-toolbar {
    gap: 10px;
    padding: 12px 0;
    border-top: 1px solid var(--art-border-color);
    border-bottom: 1px solid var(--art-border-color);
  }
  .search-input {
    width: min(420px, 42vw);
  }
  .type-input {
    width: 150px;
  }
  .status-input {
    width: 130px;
  }
  .toolbar-hint {
    margin-left: auto;
    color: var(--art-text-gray-500);
    font-size: 12px;
  }
  .metric-strip {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    margin: 18px 0;
    border: 1px solid var(--art-border-color);
    border-radius: 7px;
    background: var(--art-bg-color);
  }
  .metric-item {
    min-height: 86px;
    display: flex;
    align-items: center;
    gap: 11px;
    padding: 15px 18px;
    border-right: 1px solid var(--art-border-color);
  }
  .metric-item:last-child {
    border-right: 0;
  }
  .metric-item > span:last-child {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .metric-item small {
    color: var(--art-text-gray-500);
    font-size: 11px;
  }
  .metric-item strong {
    font-size: 24px;
    line-height: 28px;
  }
  .metric-icon,
  .project-type,
  .service-type,
  .empty-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }
  .metric-icon {
    width: 35px;
    height: 35px;
    border-radius: 8px;
  }
  .metric-icon.is-blue {
    color: #2563eb;
    background: #eaf1ff;
  }
  .metric-icon.is-teal {
    color: #0f766e;
    background: #e6f7f4;
  }
  .metric-icon.is-orange {
    color: #c2410c;
    background: #fff2e8;
  }
  .metric-icon.is-red {
    color: #dc2626;
    background: #fff0f0;
  }
  .dashboard-layout {
    min-height: calc(100vh - 300px);
    display: grid;
    grid-template-columns: 230px minmax(520px, 1fr) 310px;
    overflow: hidden;
    border: 1px solid var(--art-border-color);
    border-radius: 7px;
    background: var(--art-bg-color);
  }
  .project-panel,
  .activity-panel {
    min-width: 0;
    display: flex;
    flex-direction: column;
    background: #fbfcfe;
  }
  .project-panel {
    border-right: 1px solid var(--art-border-color);
  }
  .activity-panel {
    border-left: 1px solid var(--art-border-color);
  }
  .panel-heading {
    min-height: 57px;
    justify-content: space-between;
    padding: 0 13px;
    border-bottom: 1px solid var(--art-border-color);
  }
  .panel-heading > div {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .panel-heading strong {
    font-size: 13px;
  }
  .panel-count {
    min-width: 22px;
    padding: 2px 6px;
    border-radius: 10px;
    background: #edf3ff;
    color: var(--el-color-primary);
    font-size: 10px;
    text-align: center;
  }
  .project-list {
    flex: 1;
    min-height: 300px;
    padding: 7px;
    overflow: auto;
  }
  .project-item {
    width: 100%;
    display: grid;
    grid-template-columns: 29px minmax(0, 1fr) auto;
    align-items: center;
    gap: 8px;
    padding: 9px 7px;
    border: 1px solid transparent;
    border-radius: 5px;
    background: transparent;
    color: inherit;
    text-align: left;
    cursor: pointer;
  }
  .project-item:hover {
    background: var(--art-gray-100);
  }
  .project-item.is-active {
    border-color: #b9d0ff;
    background: #eef4ff;
  }
  .project-type {
    width: 27px;
    height: 27px;
    border-radius: 6px;
    color: #fff;
  }
  .project-type.large {
    width: 38px;
    height: 38px;
    flex: 0 0 38px;
    border-radius: 7px;
  }
  .is-k8s_yaml {
    background: #2563eb;
  }
  .is-helm {
    background: #d97706;
  }
  .is-host {
    background: #16805b;
  }
  .project-copy {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .project-copy strong,
  .project-copy small {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .project-copy strong {
    font-size: 12px;
  }
  .project-copy small {
    color: var(--art-text-gray-500);
    font-size: 10px;
  }
  .project-state {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    color: var(--art-text-gray-500);
    font-size: 9px;
    white-space: nowrap;
  }
  .project-state.active {
    color: #15803d;
  }
  .state-dot {
    width: 7px;
    height: 7px;
    flex: 0 0 7px;
    display: inline-block;
    border-radius: 50%;
    background: #94a3b8;
  }
  .project-state.active .state-dot,
  .environment-cell.is-ready .state-dot {
    background: #16a34a;
  }
  .panel-footnote {
    display: flex;
    gap: 7px;
    padding: 10px 12px;
    border-top: 1px solid var(--art-border-color);
    color: var(--art-text-gray-500);
    font-size: 10px;
    line-height: 15px;
  }
  .panel-footnote svg {
    flex: 0 0 auto;
    color: #16805b;
  }
  .delivery-panel {
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .delivery-heading {
    min-height: 75px;
    justify-content: space-between;
    gap: 14px;
    padding: 0 17px;
    border-bottom: 1px solid var(--art-border-color);
  }
  .delivery-identity {
    min-width: 0;
    gap: 10px;
  }
  .delivery-identity > div {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .title-line {
    gap: 7px;
  }
  .title-line h2 {
    margin: 0;
    overflow: hidden;
    font-size: 17px;
    line-height: 23px;
    text-overflow: ellipsis;
    white-space: nowrap;
    letter-spacing: 0;
  }
  .delivery-identity > div > span {
    overflow: hidden;
    color: var(--art-text-gray-500);
    font-size: 11px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .board-toolbar {
    min-height: 58px;
    justify-content: space-between;
    gap: 10px;
    padding: 0 14px;
    border-bottom: 1px solid var(--art-border-color);
  }
  .board-toolbar > div:first-child {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .board-toolbar strong {
    font-size: 13px;
  }
  .board-toolbar span {
    color: var(--art-text-gray-500);
    font-size: 10px;
  }
  .board-actions {
    gap: 5px;
  }
  .matrix-wrap {
    flex: 1;
    min-height: 280px;
    overflow: hidden;
  }
  .delivery-table {
    height: 100%;
  }
  .delivery-table :deep(.el-table__inner-wrapper::before) {
    display: none;
  }
  .service-cell {
    min-width: 0;
    gap: 8px;
  }
  .service-type {
    width: 27px;
    height: 27px;
    flex: 0 0 27px;
    border-radius: 6px;
    color: #fff;
  }
  .service-cell > span:last-child,
  .version-cell,
  .artifact-cell {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .service-cell strong,
  .service-cell small,
  .artifact-cell code {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .service-cell strong {
    font-size: 12px;
  }
  .service-cell small,
  .version-cell small,
  .artifact-cell small {
    color: var(--art-text-gray-500);
    font-size: 10px;
  }
  .version-cell span {
    min-width: 0;
    gap: 7px;
  }
  .version-cell code,
  .artifact-cell code {
    overflow: hidden;
    color: var(--art-text-gray-700);
    font-family: var(--ax-font-mono);
    font-size: 10px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .environment-heading {
    gap: 5px;
    font-size: 11px;
  }
  .environment-cell {
    width: 100%;
    min-height: 46px;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    justify-content: center;
    gap: 4px;
    padding: 5px 7px;
    border: 1px solid transparent;
    border-radius: 5px;
    background: transparent;
    color: inherit;
    text-align: left;
    cursor: pointer;
  }
  .environment-cell:hover {
    border-color: #b9d0ff;
    background: #eef4ff;
  }
  .environment-cell > span {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
  .environment-cell strong {
    font-size: 11px;
    font-weight: 600;
  }
  .environment-cell small {
    max-width: 100%;
    overflow: hidden;
    color: var(--art-text-gray-500);
    font-size: 9px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .environment-cell.is-production .state-dot {
    background: #d97706;
  }
  .environment-cell.is-production strong {
    color: #b45309;
  }
  .environment-cell.is-pending .state-dot {
    background: #94a3b8;
  }
  .activity-panel .panel-heading {
    flex: 0 0 auto;
  }
  .quick-actions {
    display: flex;
    flex-direction: column;
    gap: 7px;
    padding: 12px;
    border-bottom: 1px solid var(--art-border-color);
  }
  .quick-action {
    position: relative;
    width: 100%;
    gap: 8px;
    padding: 9px 8px;
    border: 1px solid var(--art-border-color);
    border-radius: 5px;
    background: var(--art-bg-color);
    color: inherit;
    text-align: left;
    cursor: pointer;
  }
  .quick-action:hover {
    border-color: #b9d0ff;
    background: #f5f8ff;
  }
  .quick-action > span {
    width: 27px;
    height: 27px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex: 0 0 27px;
    border-radius: 6px;
    color: #2563eb;
    background: #eaf1ff;
  }
  .quick-action.is-primary > span {
    color: #fff;
    background: #2563eb;
  }
  .quick-action > strong,
  .quick-action > small {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .quick-action > strong {
    flex: 1;
    font-size: 11px;
  }
  .quick-action > small {
    position: absolute;
    left: 43px;
    right: 22px;
    bottom: 7px;
    transform: translateY(100%);
    color: var(--art-text-gray-500);
    font-size: 9px;
    opacity: 0;
  }
  .quick-action:hover > small {
    opacity: 1;
  }
  .activity-section {
    padding: 13px 12px 0;
  }
  .section-heading {
    justify-content: space-between;
    margin-bottom: 8px;
  }
  .section-heading strong {
    font-size: 12px;
  }
  .section-heading span {
    color: var(--art-text-gray-500);
    font-size: 10px;
  }
  .run-list {
    border-top: 1px solid var(--art-border-color);
  }
  .run-item {
    width: 100%;
    gap: 7px;
    min-height: 53px;
    padding: 7px 0;
    border: 0;
    border-bottom: 1px solid var(--art-border-color);
    background: transparent;
    color: inherit;
    text-align: left;
    cursor: pointer;
  }
  .run-item:hover {
    background: var(--art-gray-100);
  }
  .run-item > span:nth-child(2) {
    min-width: 0;
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 3px;
  }
  .run-item strong,
  .run-item small {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .run-item strong {
    font-size: 10px;
  }
  .run-item small {
    color: var(--art-text-gray-500);
    font-size: 9px;
  }
  .run-status {
    width: 24px;
    height: 24px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex: 0 0 24px;
    border-radius: 50%;
    color: #64748b;
    background: #f1f5f9;
  }
  .run-status.succeeded {
    color: #15803d;
    background: #eaf8ef;
  }
  .run-status.failed {
    color: #dc2626;
    background: #fff0f0;
  }
  .run-status.running,
  .run-status.pending_approval {
    color: #2563eb;
    background: #edf3ff;
  }
  .failures {
    padding-bottom: 14px;
  }
  .failure-list {
    border-top: 1px solid #fee2e2;
  }
  .failure-item {
    gap: 8px;
    padding: 9px 0;
    border-bottom: 1px solid #fee2e2;
    color: #b91c1c;
  }
  .failure-item span {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .failure-item strong {
    font-size: 10px;
  }
  .failure-item small {
    overflow: hidden;
    color: #9f1239;
    font-size: 9px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .all-clear {
    display: flex;
    align-items: center;
    gap: 7px;
    padding: 12px 0;
    color: #15803d;
    font-size: 10px;
  }
  .empty-selection {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 430px;
    padding: 30px;
    text-align: center;
  }
  .empty-icon {
    width: 52px;
    height: 52px;
    border: 1px solid var(--art-border-color);
    border-radius: 12px;
    color: var(--el-color-primary);
    background: var(--art-bg-color);
  }
  .empty-selection strong {
    margin-top: 12px;
    font-size: 15px;
  }
  .empty-selection p {
    max-width: 310px;
    margin: 7px 0;
    color: var(--art-text-gray-500);
    font-size: 11px;
    line-height: 18px;
  }
  @media (max-width: 1280px) {
    .dashboard-layout {
      grid-template-columns: 210px minmax(460px, 1fr);
    }
    .activity-panel {
      grid-column: 1 / -1;
      border-top: 1px solid var(--art-border-color);
      border-left: 0;
      display: grid;
      grid-template-columns: 210px minmax(0, 1fr) minmax(0, 1fr);
    }
    .activity-panel .panel-heading {
      min-height: auto;
      padding: 13px;
    }
    .quick-actions {
      border-bottom: 0;
      border-right: 1px solid var(--art-border-color);
    }
    .activity-section {
      padding-top: 13px;
    }
  }
  @media (max-width: 900px) {
    .ops-dashboard {
      padding: 16px;
    }
    .metric-strip {
      grid-template-columns: 1fr 1fr;
    }
    .metric-item:nth-child(2) {
      border-right: 0;
    }
    .dashboard-layout {
      grid-template-columns: 1fr;
    }
    .project-panel {
      border-right: 0;
      border-bottom: 1px solid var(--art-border-color);
    }
    .project-list {
      max-height: 210px;
    }
    .activity-panel {
      grid-column: auto;
      grid-template-columns: 1fr;
    }
    .activity-panel .panel-heading,
    .quick-actions,
    .activity-section {
      border-right: 0;
      border-bottom: 1px solid var(--art-border-color);
    }
    .quick-actions {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
    }
    .quick-action > small {
      display: none;
    }
    .search-input {
      width: min(100%, 360px);
    }
  }
  @media (max-width: 620px) {
    .dashboard-header {
      align-items: flex-start;
      flex-direction: column;
    }
    .header-actions {
      width: 100%;
      justify-content: flex-end;
    }
    .dashboard-toolbar {
      align-items: stretch;
      flex-wrap: wrap;
    }
    .search-input,
    .type-input,
    .status-input {
      width: 100%;
    }
    .toolbar-hint {
      width: 100%;
      margin-left: 0;
    }
    .metric-item {
      min-height: 76px;
      padding: 12px;
    }
    .metric-item strong {
      font-size: 21px;
    }
    .delivery-heading {
      align-items: flex-start;
      flex-direction: column;
      padding: 14px;
    }
    .delivery-heading > .el-button {
      align-self: flex-end;
    }
    .board-toolbar {
      align-items: flex-start;
      flex-direction: column;
      padding: 11px 14px;
    }
    .board-actions {
      align-self: flex-end;
    }
    .quick-actions {
      grid-template-columns: 1fr;
    }
  }
</style>
