<template>
  <ElDrawer
    v-model="drawerVisible"
    title="创建发布批次"
    size="min(900px, 96vw)"
    destroy-on-close
    class="release-draft-drawer"
  >
    <div class="release-shell">
      <div class="release-intro">
        <div>
          <span class="eyebrow">RELEASE CONTROL</span>
          <h2>发布到工作流绑定环境</h2>
          <p
            >发布会执行所选工作流的全部节点；包含构建节点时会重新构建。请核对工作流绑定的目标环境。</p
          >
        </div>
        <div class="release-counter"
          ><strong>{{ targetCount }}</strong
          ><span>个发布目标</span></div
        >
      </div>

      <ElSteps :active="step" finish-status="success" align-center class="release-steps">
        <ElStep title="版本" description="工作流、服务与 Revision" />
        <ElStep title="目标" description="环境与发布策略" />
        <ElStep title="确认" description="Diff 与运行计划" />
      </ElSteps>

      <section v-if="step === 0" class="release-section">
        <div class="section-heading"
          ><strong>选择交付版本</strong
          ><span>服务版本固定配置；当前入口尚不支持选择已有产物进行跨环境晋级。</span></div
        >
        <ElForm label-position="top">
          <ElFormItem label="工作流" required>
            <ElSelect
              v-model="form.workflowId"
              filterable
              style="width: 100%"
              @change="loadWorkflow"
            >
              <ElOption
                v-for="workflow in workflows"
                :key="workflow.id"
                :label="workflow.name"
                :value="workflow.id"
              >
                <div class="option-copy"
                  ><strong>{{ workflow.name }}</strong
                  ><small>{{ workflow.code }} · {{ workflow.node_count }} 个节点</small></div
                >
              </ElOption>
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="服务（可批量）" required>
            <ElSelect
              v-model="form.serviceIds"
              multiple
              collapse-tags
              collapse-tags-tooltip
              filterable
              style="width: 100%"
              @change="loadSelectedRevisions"
            >
              <ElOption
                v-for="service in activeServices"
                :key="service.id"
                :label="service.name"
                :value="service.id"
              />
            </ElSelect>
          </ElFormItem>
        </ElForm>
        <div v-if="form.serviceIds.length" class="revision-table">
          <div class="revision-table-heading"
            ><span>服务</span><span>发布 Revision</span><span>Commit</span></div
          >
          <div v-for="serviceId in form.serviceIds" :key="serviceId" class="revision-table-row">
            <div class="service-cell"
              ><span class="service-mark" :class="`is-${serviceById(serviceId)?.delivery_type}`"
                ><component
                  :is="serviceIcon(serviceById(serviceId)?.delivery_type)"
                  :size="14" /></span
              ><strong>{{ serviceById(serviceId)?.name || `服务 #${serviceId}` }}</strong></div
            >
            <ElSelect
              v-model="form.revisionIds[serviceId]"
              filterable
              :loading="Boolean(revisionLoading[serviceId])"
              style="width: 100%"
            >
              <ElOption
                v-for="revision in revisions[serviceId] || []"
                :key="revision.id"
                :label="`R${revision.revision_number}`"
                :value="revision.id"
              />
            </ElSelect>
            <span class="commit-text">{{
              revisionById(serviceId)?.source_commit?.slice(0, 10) || '本地定义'
            }}</span>
          </div>
        </div>
        <ElAlert
          v-if="validationError"
          type="warning"
          :closable="false"
          show-icon
          :title="validationError"
        />
      </section>

      <section v-else-if="step === 1" class="release-section">
        <div class="section-heading"
          ><strong>选择发布目标</strong
          ><span>目标必须已绑定并通过连接验证，生产环境自动进入审批。</span></div
        >
        <div v-if="!availableEnvironments.length" class="empty-target"
          ><ElEmpty :image-size="62" description="没有可发布的环境，请先完成环境绑定和验证。"
        /></div>
        <template v-else>
          <div class="environment-grid">
            <label
              v-for="environment in availableEnvironments"
              :key="environment.id"
              class="environment-option"
              :class="{ 'is-selected': form.environmentIds.includes(environment.id) }"
            >
              <ElCheckbox
                :model-value="form.environmentIds.includes(environment.id)"
                class="environment-check"
                @change="toggleEnvironment(environment.id, $event)"
                ><span
              /></ElCheckbox>
              <span class="environment-option-copy"
                ><strong>{{ environment.name }}</strong
                ><small>{{ targetLabel(environment) }}</small></span
              >
              <ElTag
                size="small"
                effect="plain"
                :type="environment.is_production ? 'danger' : 'success'"
                >{{ environment.is_production ? '生产' : '非生产' }}</ElTag
              >
            </label>
          </div>
          <div class="strategy-panel">
            <div class="strategy-heading"
              ><strong>发布策略</strong
              ><span>策略记录在发布计划中；主机服务的批次参数继续由服务定义控制。</span></div
            >
            <ElRadioGroup v-model="form.strategy" class="strategy-group">
              <ElRadioButton label="rolling"
                ><span class="strategy-copy"
                  ><strong>滚动发布</strong><small>逐步替换，适合常规服务</small></span
                ></ElRadioButton
              >
              <ElRadioButton label="canary" disabled
                ><span class="strategy-copy"
                  ><strong>金丝雀</strong><small>暂未支持</small></span
                ></ElRadioButton
              >
              <ElRadioButton label="bluegreen" disabled
                ><span class="strategy-copy"
                  ><strong>蓝绿</strong><small>暂未支持</small></span
                ></ElRadioButton
              >
            </ElRadioGroup>
            <div class="strategy-suboptions">
              <ElFormItem label="集群执行顺序"
                ><ElSelect v-model="form.executionMode" style="width: 180px"
                  ><ElOption label="并行" value="parallel" /><ElOption
                    label="按环境顺序"
                    value="ordered" /></ElSelect
              ></ElFormItem>
              <ElFormItem label="健康检查超时（秒）"
                ><ElInputNumber
                  v-model="form.timeoutSeconds"
                  :min="60"
                  :max="3600"
                  controls-position="right"
              /></ElFormItem>
            </div>
          </div>
          <ElAlert
            v-if="approvalRequired"
            type="warning"
            :closable="false"
            show-icon
            title="包含生产或审批环境，提交后会停在审批节点。"
          />
        </template>
      </section>

      <section v-else class="release-section">
        <div class="section-heading"
          ><strong>确认发布计划</strong
          ><span>确认后按选定执行模式创建独立工作流运行记录。</span></div
        >
        <div class="plan-summary">
          <div
            ><span>工作流</span><strong>{{ selectedWorkflow?.name || '-' }}</strong></div
          >
          <div
            ><span>服务</span><strong>{{ form.serviceIds.length }} 个</strong></div
          >
          <div
            ><span>目标环境</span><strong>{{ form.environmentIds.length }} 个</strong></div
          >
          <div
            ><span>策略</span><strong>{{ strategyLabel }}</strong></div
          >
        </div>
        <div class="target-plan">
          <div class="target-plan-heading"
            ><strong>目标任务</strong><span>{{ targetCount }} 个目标将生成对应 Job</span></div
          >
          <div v-for="target in targets" :key="target.key" class="target-plan-row">
            <span class="target-state" :class="target.ready ? 'is-ready' : 'is-warning'"
              ><CheckCircle2 v-if="target.ready" :size="15" /><AlertTriangle v-else :size="15"
            /></span>
            <span class="target-plan-copy"
              ><strong>{{ target.serviceName }} · {{ target.environmentName }}</strong
              ><small>{{ target.target }}</small></span
            >
            <ElTag size="small" effect="plain" :type="target.ready ? 'success' : 'warning'">{{
              target.ready ? '已校验' : '待检查'
            }}</ElTag>
          </div>
        </div>
        <div v-if="previewLoading" class="preview-loading"><ElSkeleton :rows="4" animated /></div>
        <template v-else-if="previewRows.length">
          <div class="preview-heading"
            ><strong>资源变更预览</strong
            ><span>取第一个服务目标生成 Server-Side Diff，其他目标会在运行时独立校验。</span></div
          >
          <div class="preview-metrics"
            ><span class="is-create"
              ><strong>{{ previewSummary.create }}</strong
              >新增</span
            ><span class="is-update"
              ><strong>{{ previewSummary.update }}</strong
              >更新</span
            ><span class="is-delete"
              ><strong>{{ previewSummary.delete }}</strong
              >删除</span
            ><span
              ><strong>{{ previewSummary.unchanged }}</strong
              >未变化</span
            ></div
          >
          <ElTable :data="previewRows" size="small" max-height="205"
            ><ElTableColumn label="资源" min-width="230"
              ><template #default="{ row }">{{ row.kind }}/{{ row.name }}</template></ElTableColumn
            ><ElTableColumn prop="namespace" label="Namespace" width="150" /><ElTableColumn
              label="动作"
              width="90"
              ><template #default="{ row }"
                ><ElTag size="small" :type="changeType(row.action)">{{
                  changeLabel(row.action)
                }}</ElTag></template
              ></ElTableColumn
            ></ElTable
          >
        </template>
        <ElAlert v-else-if="previewError" type="warning" :closable="false" :title="previewError" />
        <ElAlert
          v-if="approvalRequired"
          type="warning"
          :closable="false"
          show-icon
          title="审批环境不会立即执行部署，运行记录会保持待审批状态。"
        />
      </section>
    </div>

    <template #footer>
      <div class="release-footer"
        ><span>{{ step + 1 }} / 3</span
        ><div
          ><ElButton @click="drawerVisible = false">取消</ElButton
          ><ElButton v-if="step > 0" @click="step -= 1">上一步</ElButton
          ><ElButton
            v-if="step < 2"
            type="primary"
            :loading="validating"
            :disabled="!canNext"
            @click="nextStep"
            >下一步</ElButton
          ><ElButton
            v-else
            type="primary"
            :icon="Rocket"
            :loading="running"
            :disabled="!canRun"
            @click="run"
            >确认并创建运行</ElButton
          ></div
        ></div
      >
    </template>
  </ElDrawer>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue'
  import { ElMessage } from 'element-plus'
  import {
    AlertTriangle,
    CheckCircle2,
    Container,
    FileCode2,
    PackageOpen,
    Rocket
  } from 'lucide-vue-next'
  import {
    getWorkflowApi,
    listServiceRevisionsApi,
    previewEnvironmentResourcesApi,
    runWorkflowApi,
    type AxiOpsCluster,
    type AxiOpsDeploymentPreview,
    type AxiOpsEnvironment,
    type AxiOpsService,
    type AxiOpsServiceRevision,
    type AxiOpsWorkflow,
    type DeliveryType
  } from '@/api/axiops'

  const props = defineProps<{
    visible: boolean
    deliveryType: DeliveryType
    services: AxiOpsService[]
    workflows: AxiOpsWorkflow[]
    environments: AxiOpsEnvironment[]
    clusters?: AxiOpsCluster[]
  }>()
  const emit = defineEmits<{ 'update:visible': [value: boolean]; executed: [runId: number] }>()
  const drawerVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })
  const step = ref(0)
  const validating = ref(false)
  const running = ref(false)
  const validationError = ref('')
  const previewLoading = ref(false)
  const previewError = ref('')
  const workflowDetail = ref<AxiOpsWorkflow | null>(null)
  const revisions = ref<Record<number, AxiOpsServiceRevision[]>>({})
  const revisionLoading = ref<Record<number, boolean>>({})
  const previewRows = ref<AxiOpsDeploymentPreview['changes']>([])
  const previewSummary = reactive({ create: 0, update: 0, delete: 0, unchanged: 0 })
  const form = reactive<{
    workflowId: number
    serviceIds: number[]
    revisionIds: Record<number, number>
    environmentIds: number[]
    strategy: 'rolling' | 'canary' | 'bluegreen'
    executionMode: 'parallel' | 'ordered'
    timeoutSeconds: number
  }>({
    workflowId: 0,
    serviceIds: [],
    revisionIds: {},
    environmentIds: [],
    strategy: 'rolling',
    executionMode: 'parallel',
    timeoutSeconds: 900
  })
  const availableEnvironments = computed(() =>
    props.environments.filter(
      (item) =>
        item.id === selectedWorkflow.value?.target_environment_id &&
        item.status === 'active' &&
        item.binding_status === 'validated' &&
        (props.deliveryType === 'host'
          ? item.environment_type === 'host_group'
          : item.environment_type === 'k8s_namespace')
    )
  )
  const activeServices = computed(() => props.services.filter((item) => item.status === 'active'))
  const selectedWorkflow = computed(() =>
    props.workflows.find((item) => item.id === form.workflowId)
  )
  const approvalRequired = computed(() =>
    form.environmentIds.some((id) => {
      const item = availableEnvironments.value.find((environment) => environment.id === id)
      return Boolean(item?.is_production || item?.deploy_approval)
    })
  )
  const targetCount = computed(() => form.serviceIds.length * form.environmentIds.length)
  const selectedService = computed(() => serviceById(form.serviceIds[0]))
  const strategyLabel = computed(
    () => ({ rolling: '滚动发布', canary: '金丝雀发布', bluegreen: '蓝绿发布' })[form.strategy]
  )
  const canNext = computed(() =>
    step.value === 0
      ? Boolean(
          form.workflowId &&
            form.serviceIds.length &&
            form.serviceIds.every((id) => form.revisionIds[id])
        )
      : Boolean(form.environmentIds.length && targetCount.value)
  )
  const canRun = computed(() =>
    Boolean(!previewError.value && targetCount.value && form.workflowId)
  )
  const serviceById = (id: number) => activeServices.value.find((item) => item.id === id)
  const serviceIcon = (type?: string) =>
    ({ k8s_yaml: FileCode2, helm: PackageOpen, host: Container })[type || ''] || FileCode2
  const revisionById = (serviceId: number) =>
    (revisions.value[serviceId] || []).find((item) => item.id === form.revisionIds[serviceId])
  const targetLabel = (environment: AxiOpsEnvironment) =>
    environment.environment_type === 'host_group'
      ? environment.host_group_ref || '主机组未绑定'
      : `${props.clusters?.find((cluster) => cluster.id === environment.cluster_id)?.name || `集群 #${environment.cluster_id}`} / ${environment.namespace || '-'}`
  const toggleEnvironment = (environmentId: number, checked: string | number | boolean) => {
    if (checked) {
      if (!form.environmentIds.includes(environmentId)) form.environmentIds.push(environmentId)
    } else {
      form.environmentIds = form.environmentIds.filter((id) => id !== environmentId)
    }
  }
  const reset = () => {
    step.value = 0
    validationError.value = ''
    previewError.value = ''
    previewRows.value = []
    Object.assign(previewSummary, { create: 0, update: 0, delete: 0, unchanged: 0 })
    form.workflowId = props.workflows[0]?.id || 0
    form.serviceIds = activeServices.value[0]?.id ? [activeServices.value[0].id] : []
    form.revisionIds = {}
    form.environmentIds = availableEnvironments.value[0]?.id
      ? [availableEnvironments.value[0].id]
      : []
    form.strategy = 'rolling'
    form.executionMode = 'parallel'
    form.timeoutSeconds = 900
    revisions.value = {}
    workflowDetail.value = null
    if (form.serviceIds.length) void loadSelectedRevisions()
  }
  const loadWorkflow = async () => {
    if (!form.workflowId) return
    validationError.value = ''
    try {
      workflowDetail.value = await getWorkflowApi(form.workflowId)
      form.environmentIds = form.environmentIds.filter(
        (id) => id === workflowDetail.value?.target_environment_id
      )
    } catch (error: any) {
      validationError.value = error?.message || '工作流加载失败'
    }
  }
  const loadSelectedRevisions = async () => {
    const next: Record<number, AxiOpsServiceRevision[]> = {}
    for (const serviceId of form.serviceIds) {
      revisionLoading.value[serviceId] = true
      try {
        next[serviceId] = (await listServiceRevisionsApi(serviceId)) || []
        form.revisionIds[serviceId] = next[serviceId][0]?.id || 0
      } finally {
        revisionLoading.value[serviceId] = false
      }
    }
    revisions.value = { ...revisions.value, ...next }
  }
  const validateWorkflow = async () => {
    await loadWorkflow()
    const detail = workflowDetail.value
    if (!detail) return false
    const deployNodes =
      detail.nodes?.filter((node) =>
        ['yaml_deploy', 'helm_deploy', 'host_deploy'].includes(node.type)
      ) || []
    if (!detail.target_environment_id)
      validationError.value = '工作流尚未绑定环境，请先在编排中绑定'
    else if (form.environmentIds.some((id) => id !== detail.target_environment_id))
      validationError.value = '发布目标与工作流绑定环境不一致'
    else if (detail.status !== 'active') validationError.value = '所选工作流未启用'
    else if (detail.delivery_type !== props.deliveryType)
      validationError.value = '工作流交付类型与项目不匹配'
    else if (deployNodes.length !== 1)
      validationError.value = '发布批次要求工作流恰好包含一个部署节点'
    else if (approvalRequired.value && !detail.nodes?.some((node) => node.type === 'approval'))
      validationError.value = '审批环境需要包含人工审批节点的工作流'
    return !validationError.value
  }
  const nextStep = async () => {
    validating.value = true
    try {
      if (step.value === 0) {
        if (!canNext.value || !(await validateWorkflow())) return
        step.value = 1
      } else if (canNext.value) {
        step.value = 2
        await loadPreview()
      }
    } finally {
      validating.value = false
    }
  }
  const loadPreview = async () => {
    previewLoading.value = true
    previewError.value = ''
    previewRows.value = []
    Object.assign(previewSummary, { create: 0, update: 0, delete: 0, unchanged: 0 })
    if (props.deliveryType === 'host') {
      previewLoading.value = false
      return
    }
    const environmentId = form.environmentIds[0]
    const serviceId = form.serviceIds[0]
    try {
      const preview = await previewEnvironmentResourcesApi(environmentId, {
        service_id: serviceId,
        service_revision_id: form.revisionIds[serviceId],
        release_name: selectedService.value?.code
      })
      previewRows.value = preview.changes || []
      Object.assign(previewSummary, preview.summary)
    } catch (error: any) {
      previewError.value = error?.message || '资源预览失败，请检查集群连接和 Revision'
    } finally {
      previewLoading.value = false
    }
  }
  const targets = computed(() =>
    form.serviceIds.flatMap((serviceId) =>
      form.environmentIds.map((environmentId) => {
        const service = serviceById(serviceId)
        const environment = availableEnvironments.value.find((item) => item.id === environmentId)
        return {
          key: `${serviceId}-${environmentId}`,
          serviceName: service?.name || `服务 #${serviceId}`,
          environmentName: environment?.name || `环境 #${environmentId}`,
          target: environment ? targetLabel(environment) : '-',
          ready: Boolean(service && form.revisionIds[serviceId] && environment)
        }
      })
    )
  )
  const run = async () => {
    if (!canRun.value || running.value) return
    running.value = true
    const ids: number[] = []
    try {
      const jobs = targets.value.map((target) => async () => {
        const [serviceID, environmentID] = target.key.split('-').map(Number)
        const result = await runWorkflowApi(form.workflowId, {
          environment_id: environmentID,
          trigger_type: 'manual',
          service_id: serviceID,
          service_revision_id: form.revisionIds[serviceID],
          deployment_strategy: form.strategy
        })
        ids.push(result.id)
        return result.id
      })
      if (form.executionMode === 'ordered') for (const job of jobs) await job()
      else await Promise.all(jobs.map((job) => job()))
      if (ids[0]) {
        ElMessage.success(`已创建 ${ids.length} 个工作流运行`)
        emit('executed', ids[0])
        drawerVisible.value = false
      }
    } catch (error: any) {
      ElMessage.error(error?.message || '发布批次创建失败，已创建的运行可在运行记录中查看')
    } finally {
      running.value = false
    }
  }
  const changeLabel = (action: string) =>
    ({ create: '新增', update: '更新', delete: '删除', unchanged: '未变化' })[action] || action
  const changeType = (action: string) =>
    ({ create: 'success', update: 'warning', delete: 'danger', unchanged: 'info' })[action] as
      | 'success'
      | 'warning'
      | 'danger'
      | 'info'
  watch(
    () => [props.visible, props.services.length, props.workflows.length],
    ([visible]) => {
      if (visible) reset()
    }
  )
</script>

<style scoped lang="scss">
  .release-shell {
    min-height: calc(100vh - 165px);
    padding: 4px 2px 20px;
    color: var(--art-text-gray-900);
  }
  .release-intro {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    padding-bottom: 18px;
  }
  .eyebrow {
    color: var(--el-color-primary);
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.12em;
  }
  .release-intro h2 {
    margin: 6px 0 4px;
    font-size: 21px;
    letter-spacing: 0;
  }
  .release-intro p {
    margin: 0;
    color: var(--art-text-gray-500);
    font-size: 12px;
    line-height: 18px;
  }
  .release-counter {
    min-width: 90px;
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 2px;
    padding: 8px 12px;
    border-left: 2px solid var(--el-color-primary);
    background: #f8fbff;
  }
  .release-counter strong {
    color: var(--el-color-primary);
    font-size: 24px;
    line-height: 28px;
  }
  .release-counter span,
  .section-heading span,
  .strategy-heading span,
  .target-plan-heading span,
  .preview-heading span {
    color: var(--art-text-gray-500);
    font-size: 10px;
  }
  .release-steps {
    margin: 3px 0 24px;
  }
  .release-section {
    min-height: 420px;
  }
  .section-heading {
    display: flex;
    flex-direction: column;
    gap: 3px;
    margin-bottom: 15px;
  }
  .section-heading strong {
    font-size: 15px;
  }
  .option-copy {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .option-copy small {
    color: var(--art-text-gray-500);
    font-size: 10px;
  }
  .revision-table {
    margin-top: 16px;
    border: 1px solid var(--art-border-color);
    border-radius: 7px;
    overflow: hidden;
  }
  .revision-table-heading,
  .revision-table-row {
    display: grid;
    grid-template-columns: minmax(170px, 1fr) minmax(190px, 1fr) 115px;
    align-items: center;
    gap: 12px;
    padding: 9px 12px;
  }
  .revision-table-heading {
    color: var(--art-text-gray-500);
    background: #f8fafc;
    font-size: 10px;
  }
  .revision-table-row {
    min-height: 55px;
    border-top: 1px solid var(--art-border-color);
  }
  .service-cell {
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 7px;
  }
  .service-cell strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 12px;
  }
  .service-mark {
    width: 24px;
    height: 24px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex: 0 0 24px;
    border-radius: 5px;
    color: #fff;
  }
  .service-mark.is-k8s_yaml {
    background: #2563eb;
  }
  .service-mark.is-helm {
    background: #d97706;
  }
  .service-mark.is-host {
    background: #16805b;
  }
  .commit-text {
    overflow: hidden;
    color: var(--art-text-gray-500);
    font-family: var(--ax-font-mono);
    font-size: 10px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .environment-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 9px;
  }
  .environment-option {
    min-height: 68px;
    display: grid;
    grid-template-columns: 25px minmax(0, 1fr) auto;
    align-items: center;
    gap: 8px;
    padding: 9px 11px;
    border: 1px solid var(--art-border-color);
    border-radius: 7px;
    cursor: pointer;
    transition:
      border-color 0.18s,
      background 0.18s;
  }
  .environment-option.is-selected {
    border-color: var(--el-color-primary);
    background: #f5f8ff;
    box-shadow: inset 0 0 0 1px var(--el-color-primary);
  }
  .environment-check :deep(.el-checkbox__label) {
    display: none;
  }
  .environment-option-copy {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .environment-option-copy strong,
  .environment-option-copy small {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .environment-option-copy strong {
    font-size: 12px;
  }
  .environment-option-copy small {
    color: var(--art-text-gray-500);
    font-size: 10px;
  }
  .strategy-panel {
    margin-top: 20px;
    padding: 14px;
    border: 1px solid var(--art-border-color);
    border-radius: 7px;
    background: #fbfcfe;
  }
  .strategy-heading {
    display: flex;
    flex-direction: column;
    gap: 3px;
    margin-bottom: 12px;
  }
  .strategy-group {
    display: flex;
    width: 100%;
  }
  .strategy-group :deep(.el-radio-button) {
    flex: 1;
  }
  .strategy-group :deep(.el-radio-button__inner) {
    width: 100%;
    padding: 10px 8px;
  }
  .strategy-copy {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .strategy-copy small {
    color: var(--art-text-gray-500);
    font-size: 10px;
  }
  .strategy-suboptions {
    display: flex;
    gap: 18px;
    margin-top: 15px;
  }
  .strategy-suboptions :deep(.el-form-item) {
    margin: 0;
  }
  .plan-summary {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 1px;
    margin-bottom: 18px;
    border: 1px solid var(--art-border-color);
    background: var(--art-border-color);
  }
  .plan-summary > div {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 11px 13px;
    background: var(--art-bg-color);
  }
  .plan-summary span {
    color: var(--art-text-gray-500);
    font-size: 10px;
  }
  .plan-summary strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 12px;
  }
  .target-plan {
    border: 1px solid var(--art-border-color);
    border-radius: 7px;
    overflow: hidden;
  }
  .target-plan-heading,
  .target-plan-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 12px;
  }
  .target-plan-heading {
    justify-content: space-between;
    background: #f8fafc;
  }
  .target-plan-row {
    min-height: 51px;
    border-top: 1px solid var(--art-border-color);
  }
  .target-state {
    width: 23px;
    height: 23px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex: 0 0 23px;
    border-radius: 50%;
  }
  .target-state.is-ready {
    color: #15803d;
    background: #eaf8ef;
  }
  .target-state.is-warning {
    color: #b45309;
    background: #fff5e6;
  }
  .target-plan-copy {
    min-width: 0;
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 3px;
  }
  .target-plan-copy strong,
  .target-plan-copy small {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .target-plan-copy strong {
    font-size: 12px;
  }
  .target-plan-copy small {
    color: var(--art-text-gray-500);
    font-size: 10px;
  }
  .preview-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin: 18px 0 8px;
  }
  .preview-metrics {
    display: flex;
    gap: 15px;
    margin-bottom: 8px;
    color: var(--art-text-gray-500);
    font-size: 10px;
  }
  .preview-metrics span {
    display: inline-flex;
    align-items: baseline;
    gap: 4px;
  }
  .preview-metrics strong {
    color: var(--art-text-gray-800);
    font-size: 16px;
  }
  .preview-metrics .is-create strong {
    color: #15803d;
  }
  .preview-metrics .is-update strong {
    color: #b45309;
  }
  .preview-metrics .is-delete strong {
    color: #dc2626;
  }
  .preview-loading {
    padding-top: 14px;
  }
  .release-footer {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    color: var(--art-text-gray-500);
    font-size: 11px;
  }
  @media (max-width: 680px) {
    .environment-grid,
    .plan-summary {
      grid-template-columns: 1fr;
    }
    .revision-table-heading,
    .revision-table-row {
      grid-template-columns: 1fr;
      gap: 6px;
    }
    .revision-table-heading {
      display: none;
    }
    .strategy-group {
      flex-direction: column;
    }
    .strategy-suboptions {
      flex-direction: column;
      gap: 8px;
    }
  }
</style>
