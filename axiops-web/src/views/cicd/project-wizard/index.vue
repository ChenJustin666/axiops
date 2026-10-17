<template>
  <div v-loading="loading" class="project-wizard">
    <header class="wizard-header">
      <div class="brand"><ArtLogo :size="28" /><ArtWordmark :height="21" /></div>
      <div class="project-title">
        <span>创建项目</span>
        <strong>{{ project?.name || '项目初始化' }}</strong>
      </div>
      <ElButton text @click="exitWizard">退出</ElButton>
    </header>

    <main v-if="project" class="wizard-body">
      <nav class="stepper" aria-label="项目初始化步骤">
        <button
          v-for="item in steps"
          :key="item.value"
          type="button"
          :class="[
            'step-item',
            { active: currentStep === item.value, done: currentStep > item.value }
          ]"
          :disabled="saving || item.value > currentStep"
          @click="item.value < currentStep && (currentStep = item.value)"
        >
          <span class="step-number">{{ item.value }}</span>
          <span
            ><strong>{{ item.title }}</strong
            ><small>{{ item.description }}</small></span
          >
        </button>
      </nav>

      <section class="step-content">
        <div class="content-heading">
          <div>
            <span class="eyebrow">PROJECT ONBOARDING</span>
            <h1>{{ steps[currentStep - 1].title }}</h1>
            <p>{{ stepDescription }}</p>
          </div>
          <ElTag effect="plain">{{ deliveryLabel }}</ElTag>
        </div>

        <section v-if="currentStep === 1" class="resource-panel">
          <div class="panel-toolbar">
            <div
              ><strong>服务目录</strong><span>{{ services.length }} 个服务</span></div
            >
            <div class="service-add-actions">
              <ElButton
                v-if="project.delivery_type !== 'helm'"
                type="primary"
                :icon="Plus"
                @click="beginManualDirectory"
                >手动</ElButton
              >
              <ElButton
                :type="project.delivery_type === 'helm' ? 'primary' : 'default'"
                :icon="PackageOpen"
                @click="openImportWizard('template')"
                >模板</ElButton
              >
              <ElButton :icon="FileCode2" @click="openImportWizard('git')">Git</ElButton>
            </div>
          </div>
          <div v-if="manualDirectoryOpen" class="manual-directory-row">
            <ElInput
              v-model="manualDirectoryName"
              size="small"
              placeholder="服务名称，如 risk_server"
              maxlength="128"
              :title="SERVICE_NAME_HINT"
              @keyup.enter="createManualDirectory"
            />
            <ElButton
              type="success"
              size="small"
              :loading="manualDirectorySaving"
              @click="createManualDirectory"
              >创建目录</ElButton
            >
            <ElButton text size="small" @click="cancelManualDirectory">取消</ElButton>
          </div>
          <ElText v-if="manualDirectoryOpen" size="small" type="info">{{
            SERVICE_NAME_HINT
          }}</ElText>
          <div v-if="services.length" class="service-list">
            <div v-for="service in services" :key="service.id" class="resource-row">
              <span class="resource-icon"><Boxes :size="18" /></span>
              <div
                ><strong>{{ service.name }}</strong
                ><small>{{ service.code }} · {{ service.source_type }}</small></div
              >
              <ElTag size="small" type="success" effect="plain">已配置</ElTag>
            </div>
          </div>
          <ElEmpty v-else description="先添加一个服务目录，再继续创建环境" />
        </section>

        <section v-else-if="currentStep === 2" class="resource-panel">
          <div class="panel-toolbar">
            <div
              ><strong>运行环境</strong
              ><span>默认一个测试环境，可按需添加；项目创建后仍可新增环境</span></div
            >
            <ElSelect
              model-value=""
              placeholder="添加预设环境"
              style="width: 170px"
              @change="addEnvironmentPreset"
            >
              <ElOption
                v-for="item in environmentPresets"
                :key="item.code"
                :label="item.name"
                :value="item.code"
                :disabled="environmentDrafts.some((e) => e.code === item.code)"
              />
            </ElSelect>
            <ElButton :icon="Plus" @click="addEnvironment">添加环境</ElButton>
          </div>
          <div class="environment-list">
            <div
              v-for="(environment, index) in environmentDrafts"
              :key="environment.key"
              class="environment-row"
            >
              <ElInput
                v-model="environment.name"
                placeholder="环境名称，如 Dev"
                :disabled="saving"
              />
              <ElInput v-model="environment.code" placeholder="编码，如 dev" :disabled="saving" />
              <ElSelect
                v-model="environment.cluster_id"
                placeholder="选择集群"
                filterable
                :disabled="saving"
              >
                <ElOption
                  v-for="cluster in clusters"
                  :key="cluster.id"
                  :label="cluster.name"
                  :value="cluster.id"
                />
              </ElSelect>
              <ElInput v-model="environment.namespace" placeholder="命名空间" :disabled="saving" />
              <ElCheckbox v-model="environment.is_production" :disabled="saving">生产</ElCheckbox>
              <ElCheckbox v-model="environment.deploy_approval" :disabled="saving">审批</ElCheckbox>
              <ElButton
                text
                type="danger"
                :icon="Delete"
                :disabled="saving"
                @click="removeEnvironment(index)"
              />
              <HelmEnvironmentServices
                v-if="
                  project.delivery_type === 'helm' &&
                  !environments.some((e) => e.id === Number(environment.key))
                "
                v-model="environment.helm_services"
                class="helm-environment-services"
                :services="services"
                :environments="environments"
                :disabled="saving"
                @busy-change="environmentBusy[environment.key] = $event"
              />
            </div>
          </div>
          <ElEmpty v-if="!environmentDrafts.length" description="添加至少一个环境" />
        </section>

        <section v-else class="resource-panel">
          <ElButton type="primary" @click="workflowCreationVisible = true"
            >从模板新建工作流</ElButton
          >
          <div class="panel-toolbar">
            <div><strong>工作流</strong><span>系统按环境生成标准构建、测试、部署流水线</span></div>
            <ElButton type="primary" :loading="workflowGenerating" @click="generateWorkflows"
              >生成工作流</ElButton
            >
          </div>
          <div v-if="workflows.length" class="workflow-list">
            <div v-for="workflow in workflows" :key="workflow.id" class="resource-row">
              <span class="resource-icon workflow"><Workflow :size="18" /></span>
              <div
                ><strong>{{ workflow.name }}</strong
                ><small>{{ workflow.code }} · {{ workflow.node_count }} 个节点</small></div
              >
              <ElButton size="small" @click="runWorkflow(workflow)">运行</ElButton>
            </div>
          </div>
          <ElEmpty v-else description="点击生成工作流，为每个环境创建标准流水线" />
        </section>
      </section>
    </main>

    <footer v-if="project" class="wizard-footer">
      <ElButton :disabled="currentStep === 1" @click="previousStep">上一步</ElButton>
      <div class="footer-progress">第 {{ currentStep }} / 3 步</div>
      <ElButton v-if="currentStep < 3" type="primary" :loading="saving" @click="nextStep"
        >保存并继续</ElButton
      >
      <ElButton v-else type="success" :loading="completing" @click="completeWizard"
        >完成并进入项目</ElButton
      >
    </footer>

    <WorkflowCreationDialog
      v-if="project"
      v-model:visible="workflowCreationVisible"
      :project="project"
      :services="services"
      :environments="environments"
      @created="onWorkflowCreated"
    />
    <ServiceCreationWizard
      v-model:visible="serviceWizardVisible"
      :project-id="projectId"
      :delivery-type="project?.delivery_type || 'k8s_yaml'"
      :service="null"
      :code-sources="codeSources"
      :environments="[]"
      :can-create-workflow="false"
      :can-update-build="canUpdateBuild"
      :create-workflow="false"
      :initial-source="
        wizardSource === 'git' ? 'git' : wizardSource === 'template' ? 'template' : 'manual'
      "
      @saved="handleServiceSaved"
    />
    <HelmEnvironmentReview ref="environmentReview" :project-id="projectId" :services="services" />
  </div>
</template>

<script setup lang="ts">
  import { SERVICE_NAME_HINT, serviceNameError, serviceCodeFromName } from '@/utils/service-name'
  import { computed, onMounted, ref } from 'vue'
  import type { InitialHelmService } from '@/api/axiops'
  import HelmEnvironmentServices from '../project-workspace/components/HelmEnvironmentServices.vue'
  import HelmEnvironmentReview from '../project-workspace/components/HelmEnvironmentReview.vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { useRoute, useRouter } from 'vue-router'
  import { Boxes, Delete, FileCode2, PackageOpen, Plus, Workflow } from 'lucide-vue-next'
  import {
    createEnvironmentApi,
    createServiceApi,
    createServiceRevisionApi,
    createWorkflowApi,
    getProjectApi,
    getServiceEnvironmentConfigApi,
    listAllServicesApi,
    listClustersApi,
    listCodeSourcesApi,
    listEnvironmentsApi,
    listWorkflowsApi,
    updateProjectApi,
    type AxiOpsCluster,
    type AxiOpsCodeSource,
    type AxiOpsEnvironment,
    type AxiOpsProject,
    type AxiOpsService,
    type AxiOpsWorkflow,
    type WorkflowNode
  } from '@/api/axiops'
  import { useAxiOpsAccess } from '@/composables/useAxiOpsAccess'
  import ServiceCreationWizard from '../project-workspace/components/ServiceCreationWizard.vue'
  import WorkflowCreationDialog from '../project-workspace/components/WorkflowCreationDialog.vue'

  defineOptions({ name: 'AxiOpsProjectInitializationWizard' })
  const route = useRoute()
  const router = useRouter()
  const access = useAxiOpsAccess()
  const projectId = Number(route.params.projectId)
  const loading = ref(false)
  const saving = ref(false)
  const environmentBusy = ref<Record<string, boolean>>({})
  const environmentReview = ref<InstanceType<typeof HelmEnvironmentReview>>()
  const completing = ref(false)
  const workflowGenerating = ref(false)
  const workflowCreationVisible = ref(false)
  const onWorkflowCreated = async () => {
    workflows.value = await listWorkflowsApi(projectId, { silent: true })
  }
  const currentStep = ref(1)
  const project = ref<AxiOpsProject | null>(null)
  const services = ref<AxiOpsService[]>([])
  const environments = ref<AxiOpsEnvironment[]>([])
  const clusters = ref<AxiOpsCluster[]>([])
  const codeSources = ref<AxiOpsCodeSource[]>([])
  const workflows = ref<AxiOpsWorkflow[]>([])
  const serviceWizardVisible = ref(false)
  const wizardSource = ref<'manual' | 'template' | 'git'>('manual')
  const manualDirectoryOpen = ref(false)
  const manualDirectoryName = ref('')
  const manualDirectorySaving = ref(false)
  const environmentDrafts = ref<
    Array<{
      key: string
      helm_services?: InitialHelmService[]
      name: string
      code: string
      cluster_id: number
      namespace: string
      is_production: boolean
      deploy_approval: boolean
    }>
  >([])
  const steps = [
    { value: 1, title: '新建服务', description: '添加服务目录和定义' },
    { value: 2, title: '创建环境', description: '绑定集群和命名空间' },
    { value: 3, title: '确认工作流', description: '按环境生成交付流水线' }
  ]
  const deliveryLabel = computed(
    () =>
      ({ k8s_yaml: 'K8s YAML', helm: 'Helm Chart', host: '主机应用', unclassified: '未设置' })[
        project.value?.delivery_type || 'k8s_yaml'
      ] || 'K8s YAML'
  )
  const stepDescription = computed(() => steps[currentStep.value - 1]?.description || '')
  const canUpdateBuild = computed(() =>
    access.can('build', 'update', { type: 'project', project_id: projectId })
  )

  const loadData = async () => {
    loading.value = true
    try {
      project.value = await getProjectApi(projectId)
      if (project.value.initialization_status === 'completed') {
        await router.replace(`/cicd/projects/${projectId}?tab=overview`)
        return
      }
      currentStep.value = Math.min(Math.max(project.value.initialization_step || 1, 1), 3)
      const [serviceList, environmentList, clusterList, sourceList, workflowList] =
        await Promise.all([
          listAllServicesApi(projectId, {}, { silent: true }),
          listEnvironmentsApi(projectId, { silent: true }),
          listClustersApi(undefined, { silent: true }),
          listCodeSourcesApi(projectId, { silent: true }),
          listWorkflowsApi(projectId, { silent: true })
        ])
      services.value = serviceList || []
      environments.value = environmentList || []
      clusters.value = clusterList || []
      codeSources.value = sourceList || []
      workflows.value = workflowList || []
      environmentDrafts.value = environments.value.map((item) => ({
        key: String(item.id),
        name: item.name,
        code: item.code,
        cluster_id: item.cluster_id,
        namespace: item.namespace,
        is_production: item.is_production,
        deploy_approval: item.deploy_approval
      }))
      if (!environmentDrafts.value.length) {
        const clusterId = clusters.value[0]?.id || 0
        environmentDrafts.value = [
          {
            key: 'new-qa',
            name: '测试环境',
            code: 'qa',
            cluster_id: clusterId,
            namespace: `${project.value.code}-qa`,
            is_production: false,
            deploy_approval: false
          }
        ]
      }
    } catch (error: any) {
      ElMessage.error(error?.message || '加载项目失败')
      await router.replace('/cicd/projects')
    } finally {
      loading.value = false
    }
  }
  const persistStep = async (step: number) => {
    await updateProjectApi(projectId, {
      initialization_status: step >= 4 ? 'in_progress' : 'in_progress',
      initialization_step: step
    })
  }
  const nextStep = async () => {
    if (currentStep.value === 1 && !services.value.length)
      return ElMessage.warning('请先添加至少一个服务')
    if (currentStep.value === 2) {
      if (Object.values(environmentBusy.value).some(Boolean))
        return ElMessage.warning('请等待 values 文件读取完成')
      if (!environmentDrafts.value.length) return ElMessage.warning('请先添加至少一个环境')
      for (const item of environmentDrafts.value) {
        if (!item.name.trim() || !item.code.trim() || !item.cluster_id || !item.namespace.trim())
          return ElMessage.warning('请完整填写环境名称、编码、集群和命名空间')
      }
      saving.value = true
      try {
        for (const item of environmentDrafts.value) {
          if (environments.value.some((existing) => existing.id === Number(item.key))) continue
          const input = {
            ...(project.value?.delivery_type === 'helm'
              ? { helm_services: item.helm_services || [] }
              : {}),
            name: item.name.trim(),
            code: item.code.trim(),
            cluster_id: item.cluster_id,
            namespace: item.namespace.trim(),
            is_production: item.is_production,
            deploy_approval: item.deploy_approval
          }
          if (input.helm_services?.length) {
            const snapshot = JSON.stringify(input)
            if (!environmentReview.value || !(await environmentReview.value.review(input))) return
            if (snapshot !== JSON.stringify({ ...input, helm_services: item.helm_services || [] }))
              throw Error('环境配置已变化，请重新预览')
          }
          const result = await createEnvironmentApi(projectId, input)
          item.key = String(result.id)
        }
        environments.value = await listEnvironmentsApi(projectId, { silent: true })
      } finally {
        saving.value = false
      }
    }
    if (currentStep.value < 3) {
      saving.value = true
      try {
        await persistStep(currentStep.value + 1)
        currentStep.value += 1
      } finally {
        saving.value = false
      }
    }
  }
  const previousStep = async () => {
    if (currentStep.value <= 1) return
    currentStep.value -= 1
    await persistStep(currentStep.value)
  }
  const addEnvironment = () =>
    environmentDrafts.value.push({
      key: `new-${Date.now()}`,
      name: '',
      code: '',
      cluster_id: 0,
      namespace: '',
      is_production: false,
      deploy_approval: false
    })
  const environmentPresets = [
    { name: '开发环境', code: 'dev' },
    { name: '测试环境', code: 'qa' },
    { name: '预发布环境', code: 'staging' },
    { name: '生产环境', code: 'prod' }
  ]
  const addEnvironmentPreset = (code: string) => {
    const preset = environmentPresets.find((e) => e.code === code)
    if (!preset || environmentDrafts.value.some((e) => e.code === code)) return
    environmentDrafts.value.push({
      key: `new-${code}`,
      ...preset,
      cluster_id: 0,
      namespace: `${project.value?.code}-${code}`,
      is_production: code === 'prod',
      deploy_approval: code === 'prod'
    })
  }
  const removeEnvironment = (index: number) => environmentDrafts.value.splice(index, 1)
  const handleServiceSaved = async () => {
    serviceWizardVisible.value = false
    services.value = await listAllServicesApi(projectId, {}, { silent: true })
  }
  const beginManualDirectory = () => {
    if (project.value?.delivery_type === 'helm') {
      openImportWizard('template')
      return
    }
    wizardSource.value = 'manual'
    manualDirectoryName.value = ''
    manualDirectoryOpen.value = true
  }
  const cancelManualDirectory = () => {
    manualDirectoryOpen.value = false
    manualDirectoryName.value = ''
  }
  const openImportWizard = (source: 'template' | 'git') => {
    wizardSource.value = source
    serviceWizardVisible.value = true
  }
  const createManualDirectory = async () => {
    if (project.value?.delivery_type === 'helm') return
    const name = manualDirectoryName.value.trim()
    const nameError = serviceNameError(name)
    if (nameError) return ElMessage.warning(nameError)
    const code = serviceCodeFromName(name)
    if (services.value.some((item) => item.code === code))
      return ElMessage.warning('服务目录已存在')
    manualDirectorySaving.value = true
    try {
      const created = await createServiceApi(projectId, {
        name,
        code,
        description: '',
        source_type: 'manual',
        repo_url: '',
        repo_branch: 'main',
        language: '',
        framework: '',
        image_name: ''
      })
      await createServiceRevisionApi(created.id, {
        source_commit: 'manual',
        definition:
          '# AxiOps-Variables: [{"key":"replicas","value":"1"},{"key":"port","value":"8080"}]\n# AxiOps-File: deployment.yaml\napiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: {{.Service}}\nspec:\n  replicas: {{.replicas}}\n  template:\n    spec:\n      containers:\n        - name: {{.Service}}\n          image: {{.Image}}\n          ports:\n            - containerPort: {{.port}}\n\n---\n# AxiOps-File: service.yaml\napiVersion: v1\nkind: Service\nmetadata:\n  name: {{.Service}}\nspec:\n  ports:\n    - port: {{.port}}'
      })
      cancelManualDirectory()
      services.value = await listAllServicesApi(projectId, {}, { silent: true })
      ElMessage.success('服务目录已创建')
    } catch (error: any) {
      ElMessage.error(error?.message || '服务目录创建失败')
    } finally {
      manualDirectorySaving.value = false
    }
  }
  const generateWorkflows = async () => {
    const deliveryType = project.value?.delivery_type
    if (!deliveryType || deliveryType === 'unclassified')
      return ElMessage.warning('请先设置交付类型')
    if (!services.value.length || !environments.value.length)
      return ElMessage.warning('请先完成服务和环境配置')
    workflowGenerating.value = true
    try {
      const deployType =
        project.value?.delivery_type === 'helm'
          ? 'helm_deploy'
          : project.value?.delivery_type === 'host'
            ? 'host_deploy'
            : 'yaml_deploy'
      for (const environment of environments.value) {
        if (
          workflows.value.some(
            (item) => item.code === `${project.value?.code}-${environment.code}`.slice(0, 64)
          )
        )
          continue
        let service = services.value[0]
        let serviceRevisionId = service.current_revision_id
        if (deliveryType === 'helm') {
          const selected =
            environmentDrafts.value.find((item) => Number(item.key) === environment.id)
              ?.helm_services || []
          const preferred = [
            ...selected
              .map((entry) => services.value.find((s) => s.id === entry.service_id))
              .filter((s): s is AxiOpsService => !!s),
            ...services.value.filter((s) => !selected.some((entry) => entry.service_id === s.id))
          ]
          let found = false
          for (const candidate of preferred) {
            const config = await getServiceEnvironmentConfigApi(environment.id, candidate.id)
            if (config.bound && config.revision_id > 0) {
              service = candidate
              serviceRevisionId = config.revision_id
              found = true
              break
            }
          }
          if (!found)
            throw Error(`${environment.name} 尚未配置 Helm 服务，请先选择此环境并保存服务 values`)
        }
        const nodes: WorkflowNode[] = [
          {
            key: 'build',
            name: '构建',
            type: 'build' as const,
            config: { service_id: service.id, branch: service.repo_branch },
            timeout_seconds: 3600,
            retry_limit: 1
          },
          {
            key: 'deploy',
            name: '部署',
            type: deployType,
            depends_on: ['build'],
            config: {
              service_id: service.id,
              service_revision_id: serviceRevisionId,
              ...(deliveryType === 'helm' ? { release_name: service.code } : {})
            },
            timeout_seconds: 1800,
            retry_limit: 0
          }
        ]
        if (environment.is_production || environment.deploy_approval) {
          nodes.splice(1, 0, {
            key: 'approval',
            name: '发布审批',
            type: 'approval',
            config: {},
            timeout_seconds: 3600,
            retry_limit: 0
          })
          nodes.find((node) => node.key === 'deploy')!.depends_on!.push('approval')
        }
        await createWorkflowApi(projectId, {
          target_environment_id: environment.id,
          name: `${project.value?.code}-${environment.code}`,
          code: `${project.value?.code}-${environment.code}`.slice(0, 64),
          description: `环境 ${environment.name} 的标准交付流水线`,
          definition: { delivery_type: deliveryType, nodes }
        })
      }
      workflows.value = await listWorkflowsApi(projectId, { silent: true })
      ElMessage.success('工作流已生成')
    } catch (error: any) {
      ElMessage.error(error?.message || '生成工作流失败')
    } finally {
      workflowGenerating.value = false
    }
  }
  const runWorkflow = async (workflow: AxiOpsWorkflow) => {
    await router.push({
      path: `/cicd/projects/${projectId}`,
      query: { tab: 'workflows', execute: String(workflow.id) }
    })
  }
  const completeWizard = async () => {
    if (!workflows.value.length) await generateWorkflows()
    if (!workflows.value.length) return
    if (
      workflows.value.some(
        (workflow) =>
          !workflow.target_environment_id &&
          (workflow.requires_environment ||
            workflow.nodes?.some((node) =>
              ['yaml_deploy', 'helm_deploy', 'host_deploy'].includes(node.type)
            ))
      )
    )
      return ElMessage.warning('请先在工作流编排中绑定所有部署工作流的目标环境')
    try {
      await ElMessageBox.confirm(
        '完成后将进入项目日常工作区，仍可继续修改服务、环境和工作流。',
        '完成项目初始化',
        { type: 'success' }
      )
    } catch {
      return
    }
    completing.value = true
    try {
      await updateProjectApi(projectId, {
        initialization_status: 'completed',
        initialization_step: 3
      })
      await router.replace(`/cicd/projects/${projectId}?tab=overview`)
    } finally {
      completing.value = false
    }
  }
  const exitWizard = async () => {
    await router.replace('/cicd/projects')
  }
  onMounted(async () => {
    await access.load()
    await loadData()
  })
</script>

<style scoped lang="scss">
  .project-wizard {
    min-height: 100%;
    background: #f6f8fb;
    color: #1f2937;
  }
  .wizard-header {
    height: 64px;
    display: flex;
    align-items: center;
    gap: 24px;
    padding: 0 28px;
    background: #fff;
    border-bottom: 1px solid #e5e7eb;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 9px;
    color: #0f766e;
    font-size: 18px;
  }
  .project-title {
    display: flex;
    align-items: baseline;
    gap: 12px;
    color: #64748b;
  }
  .project-title strong {
    color: #111827;
    font-size: 15px;
  }
  .wizard-header > :last-child {
    margin-left: auto;
  }
  .wizard-body {
    display: grid;
    grid-template-columns: 250px minmax(0, 1fr);
    gap: 28px;
    max-width: 1320px;
    margin: 0 auto;
    padding: 34px 28px 100px;
  }
  .stepper {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .step-item {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    width: 100%;
    padding: 13px 12px;
    text-align: left;
    border: 0;
    border-radius: 8px;
    background: transparent;
    color: #64748b;
    cursor: pointer;
  }
  .step-item:disabled {
    cursor: default;
  }
  .step-item.active {
    background: #e8f5f2;
    color: #0f766e;
  }
  .step-item.done {
    color: #15803d;
  }
  .step-number {
    display: grid;
    place-items: center;
    width: 25px;
    height: 25px;
    border: 1px solid currentColor;
    border-radius: 50%;
    font-size: 12px;
  }
  .step-item strong,
  .step-item small {
    display: block;
  }
  .step-item strong {
    font-size: 14px;
  }
  .step-item small {
    margin-top: 3px;
    font-size: 11px;
    color: #94a3b8;
  }
  .step-content {
    min-width: 0;
  }
  .content-heading {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 20px;
  }
  .eyebrow {
    color: #0f766e;
    font-size: 11px;
    letter-spacing: 0.08em;
  }
  h1 {
    margin: 6px 0;
    font-size: 24px;
  }
  .content-heading p {
    margin: 0;
    color: #64748b;
  }
  .form-panel,
  .resource-panel {
    min-height: 430px;
    padding: 24px;
    background: #fff;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
  }
  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 18px;
  }
  .info-strip {
    padding: 12px 14px;
    color: #475569;
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    font-size: 13px;
  }
  .panel-toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
  }
  .service-add-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .manual-directory-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto auto;
    gap: 8px;
    align-items: center;
    margin: -4px 0 16px;
    padding: 10px;
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
  }
  .panel-toolbar strong,
  .panel-toolbar span {
    display: block;
  }
  .panel-toolbar span {
    margin-top: 4px;
    color: #94a3b8;
    font-size: 12px;
  }
  .resource-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 14px 0;
    border-bottom: 1px solid #eef2f7;
  }
  .resource-row > div {
    flex: 1;
  }
  .resource-row strong,
  .resource-row small {
    display: block;
  }
  .resource-row small {
    margin-top: 4px;
    color: #94a3b8;
    font-size: 12px;
  }
  .resource-icon {
    display: grid;
    place-items: center;
    width: 34px;
    height: 34px;
    color: #0f766e;
    background: #e8f5f2;
    border-radius: 8px;
  }
  .resource-icon.workflow {
    color: #2563eb;
    background: #eff6ff;
  }
  .environment-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .environment-row {
    display: grid;
    grid-template-columns: 1fr 1fr 1.2fr 1fr auto auto auto;
    gap: 8px;
    align-items: center;
  }
  .helm-environment-services {
    grid-column: 1 / -1;
  }
  .wizard-footer {
    position: fixed;
    right: 0;
    bottom: 0;
    left: 0;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 28px;
    background: #fff;
    border-top: 1px solid #e5e7eb;
  }
  .footer-progress {
    color: #64748b;
    font-size: 13px;
  }
  @media (max-width: 1000px) {
    .wizard-body {
      grid-template-columns: 1fr;
    }
    .stepper {
      display: grid;
      grid-template-columns: repeat(4, 1fr);
    }
    .step-item {
      padding: 8px;
    }
    .step-item small {
      display: none;
    }
    .environment-row {
      grid-template-columns: 1fr 1fr;
    }
  }
</style>
