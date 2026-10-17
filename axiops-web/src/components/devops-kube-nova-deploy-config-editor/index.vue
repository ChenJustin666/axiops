<template>
  <div class="kube-nova-deploy-config-editor">
    <div class="deploy-config-summary" :class="{ empty: !hasConfig }">
      <div class="summary-text">
        <strong>{{ summaryTitle }}</strong>
        <span>{{ summaryDescription }}</span>
      </div>
      <ElButton :disabled="disabled || !projectId" @click="openDialog">
        {{ hasConfig ? '重新配置' : '配置' }}
      </ElButton>
    </div>

    <ElDialog
      v-model="visible"
      title="kube-nova 部署配置"
      width="920px"
      append-to-body
      align-center
      class="kube-nova-deploy-dialog"
      :close-on-click-modal="false"
    >
      <div class="deploy-dialog-body">
        <div class="deploy-dialog-layout">
          <aside class="deploy-side">
            <div class="deploy-overview">
              <div class="overview-main">
                <strong>{{ dialogOverviewTitle }}</strong>
                <span>{{ dialogOverviewDescription }}</span>
              </div>
              <div
                class="credential-state"
                :class="{
                  ready: form.credentialReady,
                  warning: !!form.channelBindingId && !form.credentialReady
                }"
              >
                {{ credentialStateText }}
              </div>
            </div>

            <div class="deploy-flow">
              <div
                v-for="(node, index) in flowNodes"
                :key="node.key"
                class="deploy-flow-node"
                :class="{ active: node.active, done: node.done }"
              >
                <span>{{ index + 1 }}</span>
                <strong>{{ node.label }}</strong>
              </div>
            </div>

            <div class="target-grid">
              <div v-for="item in selectedTargetItems" :key="item.label" class="target-item">
                <span>{{ item.label }}</span>
                <strong :title="item.value">{{ item.value || '未选择' }}</strong>
              </div>
            </div>
          </aside>
          <div class="deploy-main">
            <ElForm label-width="96px" class="deploy-form">
              <section class="form-section">
                <div class="section-title">
                  <strong>实例与认证</strong>
                  <span>实例来自当前 DevOps 项目绑定的 kube-nova 渠道，token 自动使用绑定凭据</span>
                </div>

                <ElFormItem label="实例" required>
                  <div class="field-stack">
                    <div class="select-action-row">
                      <ElSelect
                        v-model="form.channelBindingId"
                        filterable
                        remote
                        clearable
                        remote-show-suffix
                        popper-class="devops-deploy-option-dropdown"
                        :teleported="true"
                        fit-input-width
                        placeholder="请选择 kube-nova 实例"
                        :loading="loading.instance"
                        loading-text="加载中..."
                        :no-data-text="errors.instance || instanceEmptyText"
                        :remote-method="queryInstances"
                        style="width: 100%"
                        @visible-change="(opened: boolean) => opened && queryInstances()"
                        @change="handleInstanceChange"
                      >
                        <template #loading>
                          <div class="devops-select-loading">
                            <span class="devops-select-loading__icon"></span>
                            <span>加载中...</span>
                          </div>
                        </template>
                        <ElOption
                          v-for="option in instanceOptions"
                          :key="option.value"
                          :label="option.label"
                          :value="option.value"
                        >
                          <div class="deploy-option">
                            <span>{{ option.label }}</span>
                            <small v-if="option.metadata?.endpoint">
                              {{ option.metadata.endpoint }}
                            </small>
                          </div>
                        </ElOption>
                      </ElSelect>
                      <ElButton
                        :loading="loading.instance"
                        :disabled="!projectId"
                        @click="queryInstances()"
                      >
                        刷新
                      </ElButton>
                    </div>
                    <div
                      class="credential-hint"
                      :class="{ warning: form.channelBindingId && !form.credentialReady }"
                    >
                      {{ credentialHint }}
                    </div>
                  </div>
                </ElFormItem>
              </section>

              <section class="form-section">
                <div class="section-title">
                  <strong>资源定位</strong>
                  <span>按项目、集群、工作空间、应用、版本、容器逐级选择镜像替换目标</span>
                </div>

                <ElFormItem label="项目" required>
                  <div class="select-action-row">
                    <ElSelect
                      popper-class="devops-deploy-option-dropdown"
                      v-model="form.projectId"
                      filterable
                      remote
                      clearable
                      remote-show-suffix
                      :disabled="!canRequestKubeNova"
                      :placeholder="canRequestKubeNova ? '请选择项目' : '请先选择可用实例'"
                      :loading="loading.project"
                      loading-text="加载中..."
                      :no-data-text="errors.project || '暂无可选项目'"
                      :remote-method="queryProjects"
                      style="width: 100%"
                      @visible-change="(opened: boolean) => opened && queryProjects()"
                      @change="handleProjectChange"
                    >
                      <template #loading>
                        <div class="devops-select-loading">
                          <span class="devops-select-loading__icon"></span>
                          <span>加载中...</span>
                        </div>
                      </template>
                      <ElOption
                        v-for="option in projectOptions"
                        :key="option.value"
                        :label="option.label"
                        :value="option.value"
                      />
                    </ElSelect>
                    <ElButton
                      :loading="loading.project"
                      :disabled="!canRequestKubeNova"
                      @click="queryProjects()"
                    >
                      刷新
                    </ElButton>
                  </div>
                </ElFormItem>

                <ElFormItem label="集群" required>
                  <div class="select-action-row">
                    <ElSelect
                      v-model="form.projectClusterId"
                      filterable
                      remote
                      clearable
                      remote-show-suffix
                      popper-class="devops-deploy-option-dropdown"
                      :teleported="true"
                      fit-input-width
                      :disabled="!form.projectId || !canRequestKubeNova"
                      :placeholder="form.projectId ? '请选择集群' : '请先选择项目'"
                      :loading="loading.cluster"
                      loading-text="加载中..."
                      :no-data-text="errors.cluster || '暂无可选集群'"
                      :remote-method="queryClusters"
                      style="width: 100%"
                      @visible-change="(opened: boolean) => opened && queryClusters()"
                      @change="handleClusterChange"
                    >
                      <template #loading>
                        <div class="devops-select-loading">
                          <span class="devops-select-loading__icon"></span>
                          <span>加载中...</span>
                        </div>
                      </template>
                      <ElOption
                        v-for="option in clusterOptions"
                        :key="option.value"
                        :label="option.label"
                        :value="option.value"
                      >
                        <div class="deploy-option">
                          <span>{{ option.label }}</span>
                          <small v-if="option.metadata?.clusterUuid">
                            {{ option.metadata.clusterUuid }}
                          </small>
                        </div>
                      </ElOption>
                    </ElSelect>
                    <ElButton
                      :loading="loading.cluster"
                      :disabled="!form.projectId || !canRequestKubeNova"
                      @click="queryClusters()"
                    >
                      刷新
                    </ElButton>
                  </div>
                </ElFormItem>

                <ElFormItem label="工作空间" required>
                  <div class="select-action-row">
                    <ElSelect
                      v-model="form.workspaceId"
                      filterable
                      remote
                      clearable
                      remote-show-suffix
                      popper-class="devops-deploy-option-dropdown"
                      :teleported="true"
                      fit-input-width
                      :disabled="!form.projectClusterId || !canRequestKubeNova"
                      :placeholder="form.projectClusterId ? '请选择工作空间' : '请先选择集群'"
                      :loading="loading.workspace"
                      loading-text="加载中..."
                      :no-data-text="errors.workspace || '暂无可选工作空间'"
                      :remote-method="queryWorkspaces"
                      style="width: 100%"
                      @visible-change="(opened: boolean) => opened && queryWorkspaces()"
                      @change="handleWorkspaceChange"
                    >
                      <template #loading>
                        <div class="devops-select-loading">
                          <span class="devops-select-loading__icon"></span>
                          <span>加载中...</span>
                        </div>
                      </template>
                      <ElOption
                        v-for="option in workspaceOptions"
                        :key="option.value"
                        :label="option.label"
                        :value="option.value"
                      >
                        <div class="deploy-option">
                          <span>{{ option.label }}</span>
                          <small v-if="option.metadata?.namespace">{{
                            option.metadata.namespace
                          }}</small>
                        </div>
                      </ElOption>
                    </ElSelect>
                    <ElButton
                      :loading="loading.workspace"
                      :disabled="!form.projectClusterId || !canRequestKubeNova"
                      @click="queryWorkspaces()"
                    >
                      刷新
                    </ElButton>
                  </div>
                </ElFormItem>

                <ElFormItem label="应用" required>
                  <div class="select-action-row">
                    <ElSelect
                      popper-class="devops-deploy-option-dropdown"
                      v-model="form.applicationId"
                      filterable
                      remote
                      clearable
                      remote-show-suffix
                      :disabled="!form.workspaceId || !canRequestKubeNova"
                      :placeholder="form.workspaceId ? '请选择应用' : '请先选择工作空间'"
                      :loading="loading.application"
                      loading-text="加载中..."
                      :no-data-text="errors.application || '暂无可选应用'"
                      :remote-method="queryApplications"
                      style="width: 100%"
                      @visible-change="(opened: boolean) => opened && queryApplications()"
                      @change="handleApplicationChange"
                    >
                      <template #loading>
                        <div class="devops-select-loading">
                          <span class="devops-select-loading__icon"></span>
                          <span>加载中...</span>
                        </div>
                      </template>
                      <ElOption
                        v-for="option in applicationOptions"
                        :key="option.value"
                        :label="option.label"
                        :value="option.value"
                      />
                    </ElSelect>
                    <ElButton
                      :loading="loading.application"
                      :disabled="!form.workspaceId || !canRequestKubeNova"
                      @click="queryApplications()"
                    >
                      刷新
                    </ElButton>
                  </div>
                </ElFormItem>

                <ElFormItem label="版本" required>
                  <div class="select-action-row">
                    <ElSelect
                      popper-class="devops-deploy-option-dropdown"
                      v-model="form.versionId"
                      filterable
                      remote
                      clearable
                      remote-show-suffix
                      :disabled="!form.applicationId || !canRequestKubeNova"
                      :placeholder="form.applicationId ? '请选择版本' : '请先选择应用'"
                      :loading="loading.version"
                      loading-text="加载中..."
                      :no-data-text="errors.version || '暂无可选版本'"
                      :remote-method="queryVersions"
                      style="width: 100%"
                      @visible-change="(opened: boolean) => opened && queryVersions()"
                      @change="handleVersionChange"
                    >
                      <template #loading>
                        <div class="devops-select-loading">
                          <span class="devops-select-loading__icon"></span>
                          <span>加载中...</span>
                        </div>
                      </template>
                      <ElOption
                        v-for="option in versionOptions"
                        :key="option.value"
                        :label="option.label"
                        :value="option.value"
                      />
                    </ElSelect>
                    <ElButton
                      :loading="loading.version"
                      :disabled="!form.applicationId || !canRequestKubeNova"
                      @click="queryVersions()"
                    >
                      刷新
                    </ElButton>
                  </div>
                </ElFormItem>

                <ElFormItem label="容器" required>
                  <div class="select-action-row">
                    <ElSelect
                      v-model="form.containerName"
                      filterable
                      clearable
                      popper-class="devops-deploy-option-dropdown"
                      :teleported="true"
                      fit-input-width
                      :disabled="!form.versionId || !canRequestKubeNova"
                      :placeholder="form.versionId ? '请选择容器' : '请先选择版本'"
                      :loading="loading.container"
                      loading-text="加载中..."
                      :no-data-text="errors.container || '暂无可选容器'"
                      style="width: 100%"
                      @visible-change="(opened: boolean) => opened && queryContainers()"
                      @change="handleContainerChange"
                    >
                      <template #loading>
                        <div class="devops-select-loading">
                          <span class="devops-select-loading__icon"></span>
                          <span>加载中...</span>
                        </div>
                      </template>
                      <ElOption
                        v-for="option in containerOptions"
                        :key="option.value"
                        :label="option.label"
                        :value="option.value"
                      >
                        <div class="deploy-option">
                          <span>{{ option.label }}</span>
                          <small v-if="option.metadata?.image">{{ option.metadata.image }}</small>
                        </div>
                      </ElOption>
                    </ElSelect>
                    <ElButton
                      :loading="loading.container"
                      :disabled="!form.versionId || !canRequestKubeNova"
                      @click="queryContainers()"
                    >
                      刷新
                    </ElButton>
                  </div>
                </ElFormItem>
              </section>

              <section class="form-section">
                <div class="section-title">
                  <strong>执行设置</strong>
                  <span>镜像地址由流水线参数传入，这里只保存部署目标和执行控制项</span>
                </div>

                <ElFormItem label="请求超时">
                  <ElInputNumber
                    v-model="form.timeoutSeconds"
                    :min="1"
                    :max="3600"
                    controls-position="right"
                    style="width: 100%"
                  />
                </ElFormItem>
              </section>
            </ElForm>
          </div>
        </div>
      </div>

      <template #footer>
        <ElButton @click="visible = false">取消</ElButton>
        <ElButton type="primary" :disabled="confirmDisabled" @click="confirmConfig">确定</ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, reactive, ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import { listDevopsDynamicParamOptionsApi, type DynamicParamOption } from '@/api/devops/pipeline'

  defineOptions({ name: 'DevopsKubeNovaDeployConfigEditor' })

  type SelectOption = {
    label: string
    value: string
    metadata?: Record<string, any>
  }

  type DeployForm = {
    channelBindingId: string
    channelName: string
    baseUrl: string
    credentialReady: boolean
    credentialScope: string
    credentialMessage: string
    allowUseGlobalCredential: boolean
    projectId: string
    projectName: string
    projectClusterId: string
    clusterUuid: string
    clusterName: string
    workspaceId: string
    workspaceName: string
    namespace: string
    applicationId: string
    applicationName: string
    applicationCode: string
    versionId: string
    versionName: string
    resourceName: string
    containerName: string
    containerType: string
    timeoutSeconds: number
  }

  type FlowNode = {
    key: string
    label: string
    done: boolean
    active: boolean
  }

  const props = withDefaults(
    defineProps<{
      modelValue?: string
      projectId?: string
      systemId?: string
      environmentId?: string
      disabled?: boolean
    }>(),
    {
      modelValue: '',
      projectId: '',
      systemId: '',
      environmentId: '',
      disabled: false
    }
  )

  const emit = defineEmits<{
    'update:modelValue': [value: string]
    change: [value: string]
  }>()

  const visible = ref(false)
  const instanceOptions = ref<SelectOption[]>([])
  const projectOptions = ref<SelectOption[]>([])
  const clusterOptions = ref<SelectOption[]>([])
  const workspaceOptions = ref<SelectOption[]>([])
  const applicationOptions = ref<SelectOption[]>([])
  const versionOptions = ref<SelectOption[]>([])
  const containerOptions = ref<SelectOption[]>([])
  const kubeNovaProjectParamType = 'kubeNovaProject'
  const kubeNovaClusterParamType = 'kubeNovaCluster'
  const kubeNovaWorkspaceParamType = 'kubeNovaWorkspace'
  const kubeNovaApplicationParamType = 'kubeNovaApplication'
  const kubeNovaVersionParamType = 'kubeNovaVersion'
  const kubeNovaContainerParamType = 'kubeNovaContainer'
  const loading = reactive({
    instance: false,
    project: false,
    cluster: false,
    workspace: false,
    application: false,
    version: false,
    container: false
  })
  const errors = reactive({
    instance: '',
    project: '',
    cluster: '',
    workspace: '',
    application: '',
    version: '',
    container: ''
  })
  const form = reactive<DeployForm>(emptyForm())

  const parsedModel = computed(() => parseDeployConfig(props.modelValue))
  const hasConfig = computed(() => !!String(parsedModel.value?.versionId || '').trim())
  const canRequestKubeNova = computed(() => !!form.channelBindingId && form.credentialReady)
  const instanceEmptyText = computed(() =>
    props.projectId ? '当前项目未绑定 kube-nova 实例' : '请选择 DevOps 项目'
  )
  const credentialHint = computed(() => {
    if (!form.channelBindingId) return '选择实例后自动使用项目绑定的 token 凭据，页面不展示 token'
    if (form.credentialMessage) return form.credentialMessage
    if (form.credentialReady) return '已自动使用实例关联 token 凭据，页面不展示 token'
    return '当前实例未配置可用 token 凭据，请先在项目渠道绑定中配置'
  })
  const credentialStateText = computed(() => {
    if (!form.channelBindingId) return '待选择实例'
    if (form.credentialReady) {
      if (form.credentialScope === 'project') return '项目凭据已就绪'
      if (form.credentialScope === 'global') return '全局凭据已就绪'
      return '凭据已就绪'
    }
    return '凭据不可用'
  })
  const dialogOverviewTitle = computed(() => {
    if (!form.channelBindingId) return '选择 kube-nova 部署目标'
    return form.containerName
      ? `目标容器：${form.containerName}`
      : form.channelName || '选择 kube-nova 部署目标'
  })
  const dialogOverviewDescription = computed(() => {
    const parts = [
      form.channelName || form.baseUrl,
      form.projectName,
      form.clusterName || form.clusterUuid,
      form.workspaceName || form.namespace,
      form.applicationName || form.applicationCode,
      form.versionName,
      form.containerName
    ].filter(Boolean)
    return parts.length ? parts.join(' / ') : '按项目、集群、工作空间、应用、版本、容器逐级定位'
  })
  const selectedTargetItems = computed(() => [
    { label: '实例', value: form.channelName || form.baseUrl },
    { label: '项目', value: form.projectName },
    { label: '集群', value: form.clusterName || form.clusterUuid },
    { label: '工作空间', value: form.workspaceName || form.namespace },
    { label: '应用', value: form.applicationName || form.applicationCode },
    { label: '版本', value: form.versionName || form.versionId },
    { label: '容器', value: form.containerName }
  ])
  const confirmDisabled = computed(() => !!requiredFieldMessage())
  const summaryTitle = computed(() => {
    if (!hasConfig.value) return '未配置 kube-nova 部署目标'
    return parsedModel.value?.containerName
      ? `容器：${parsedModel.value.containerName}`
      : '已配置 kube-nova 部署目标'
  })
  const summaryDescription = computed(() => {
    const data = parsedModel.value
    if (!data) return '选择实例、项目、集群、工作空间、应用、版本和容器'
    const parts = [
      data.channelName || data.baseUrl,
      data.projectName,
      data.clusterName || data.clusterUuid,
      data.workspaceName || data.namespace,
      data.applicationName || data.applicationCode,
      data.versionName || data.versionId
    ].filter(Boolean)
    return parts.length ? parts.join(' / ') : '选择实例、项目、集群、工作空间、应用、版本和容器'
  })
  const flowNodes = computed<FlowNode[]>(() => {
    const nodes = [
      { key: 'instance', label: '实例', done: !!form.channelBindingId },
      { key: 'project', label: '项目', done: !!form.projectId },
      { key: 'cluster', label: '集群', done: !!form.projectClusterId },
      { key: 'workspace', label: '工作空间', done: !!form.workspaceId },
      { key: 'application', label: '应用', done: !!form.applicationId },
      { key: 'version', label: '版本', done: !!form.versionId },
      { key: 'container', label: '容器', done: !!form.containerName }
    ]
    const activeIndex = nodes.findIndex((item) => !item.done)
    return nodes.map((item, index) => ({
      ...item,
      active: activeIndex === index
    }))
  })

  function emptyForm(): DeployForm {
    return {
      channelBindingId: '',
      channelName: '',
      baseUrl: '',
      credentialReady: false,
      credentialScope: '',
      credentialMessage: '',
      allowUseGlobalCredential: false,
      projectId: '',
      projectName: '',
      projectClusterId: '',
      clusterUuid: '',
      clusterName: '',
      workspaceId: '',
      workspaceName: '',
      namespace: '',
      applicationId: '',
      applicationName: '',
      applicationCode: '',
      versionId: '',
      versionName: '',
      resourceName: '',
      containerName: '',
      containerType: '',
      timeoutSeconds: 60
    }
  }

  function openDialog() {
    if (props.disabled) return
    resetFormFromModel()
    visible.value = true
    seedOptionsFromForm()
    queryInstances().catch(() => undefined)
  }

  function resetFormFromModel() {
    Object.assign(form, emptyForm())
    const data = parsedModel.value
    if (!data) {
      return
    }
    Object.assign(form, {
      channelBindingId: stringValue(data.channelBindingId),
      channelName: stringValue(data.channelName),
      baseUrl: stringValue(data.baseUrl || data.endpoint),
      credentialReady:
        data.credentialReady === true || !!stringValue(data.tokenBase64 || data.authTokenBase64),
      credentialScope: stringValue(data.credentialScope),
      credentialMessage: stringValue(data.credentialMessage),
      allowUseGlobalCredential: data.allowUseGlobalCredential === true,
      projectId: stringValue(data.projectId),
      projectName: stringValue(data.projectName),
      projectClusterId: stringValue(data.projectClusterId),
      clusterUuid: stringValue(data.clusterUuid),
      clusterName: stringValue(data.clusterName),
      workspaceId: stringValue(data.workspaceId),
      workspaceName: stringValue(data.workspaceName),
      namespace: stringValue(data.namespace),
      applicationId: stringValue(data.applicationId),
      applicationName: stringValue(data.applicationName),
      applicationCode: stringValue(data.applicationCode),
      versionId: stringValue(data.versionId),
      versionName: stringValue(data.versionName),
      resourceName: stringValue(data.resourceName),
      containerName: stringValue(data.containerName),
      containerType: stringValue(data.containerType),
      timeoutSeconds: normalizeTimeout(data.timeoutSeconds)
    })
  }

  function seedOptionsFromForm() {
    if (
      form.channelBindingId &&
      !instanceOptions.value.some((item) => item.value === form.channelBindingId)
    ) {
      instanceOptions.value = [
        {
          label: form.channelName || form.baseUrl || form.channelBindingId,
          value: form.channelBindingId,
          metadata: {
            endpoint: form.baseUrl,
            credentialReady: form.credentialReady,
            credentialScope: form.credentialScope,
            credentialMessage: form.credentialMessage,
            allowUseGlobalCredential: form.allowUseGlobalCredential
          }
        },
        ...instanceOptions.value
      ]
    }
    seedOption(projectOptions, form.projectId, form.projectName || form.projectId)
    seedOption(
      clusterOptions,
      form.projectClusterId,
      form.clusterName || form.clusterUuid || form.projectClusterId,
      {
        clusterUuid: form.clusterUuid,
        clusterName: form.clusterName
      }
    )
    seedOption(
      workspaceOptions,
      form.workspaceId,
      form.workspaceName || form.namespace || form.workspaceId,
      {
        projectClusterId: form.projectClusterId,
        clusterUuid: form.clusterUuid,
        clusterName: form.clusterName,
        namespace: form.namespace
      }
    )
    seedOption(
      applicationOptions,
      form.applicationId,
      form.applicationName || form.applicationCode || form.applicationId
    )
    seedOption(versionOptions, form.versionId, form.versionName || form.versionId, {
      resourceName: form.resourceName
    })
    seedOption(containerOptions, form.containerName, form.containerName, {
      containerType: form.containerType
    })
  }

  function seedOption(
    target: typeof projectOptions,
    value: string,
    label: string,
    metadata?: Record<string, any>
  ) {
    if (!value || target.value.some((item) => item.value === value)) return
    target.value = [{ label, value, metadata }, ...target.value]
  }

  async function queryInstances(keyword = '') {
    if (!props.projectId) {
      errors.instance = '请选择 DevOps 项目'
      return
    }
    loading.instance = true
    errors.instance = ''
    try {
      const res = await listDevopsDynamicParamOptionsApi({
        projectId: props.projectId,
        systemId: props.systemId,
        environmentId: props.environmentId,
        paramType: 'channelGroupCode',
        configTypeCode: 'deploy_target',
        componentValue: 'kube-nova',
        mappingField: 'endpoint',
        keyword,
        page: 1,
        pageSize: 50
      })
      instanceOptions.value = (res.items || []).map(mapDynamicOption)
      refreshInstanceStateFromOptions()
      seedOptionsFromForm()
    } catch (error) {
      errors.instance = formatError(error)
    } finally {
      loading.instance = false
    }
  }

  async function queryProjects(keyword = '') {
    if (!canRequestKubeNova.value) return
    await loadOptions('project', async () => {
      projectOptions.value = await requestKubeNovaOptions(kubeNovaProjectParamType, { keyword })
      seedOptionsFromForm()
    })
  }

  async function queryClusters(keyword = '') {
    if (!form.projectId || !canRequestKubeNova.value) return
    await loadOptions('cluster', async () => {
      clusterOptions.value = await requestKubeNovaOptions(kubeNovaClusterParamType, {
        projectValue: form.projectId,
        keyword
      })
      seedOptionsFromForm()
    })
  }

  async function queryWorkspaces(keyword = '') {
    if (!form.projectClusterId || !canRequestKubeNova.value) return
    await loadOptions('workspace', async () => {
      workspaceOptions.value = await requestKubeNovaOptions(kubeNovaWorkspaceParamType, {
        projectValue: form.projectClusterId,
        keyword
      })
      seedOptionsFromForm()
    })
  }

  async function queryApplications(keyword = '') {
    if (!form.workspaceId || !canRequestKubeNova.value) return
    await loadOptions('application', async () => {
      applicationOptions.value = await requestKubeNovaOptions(kubeNovaApplicationParamType, {
        projectValue: form.workspaceId,
        keyword
      })
      seedOptionsFromForm()
    })
  }

  async function queryVersions(keyword = '') {
    if (!form.applicationId || !canRequestKubeNova.value) return
    await loadOptions('version', async () => {
      versionOptions.value = await requestKubeNovaOptions(kubeNovaVersionParamType, {
        projectValue: form.applicationId,
        keyword
      })
      seedOptionsFromForm()
    })
  }

  async function queryContainers() {
    if (!form.versionId || !canRequestKubeNova.value) return
    await loadOptions('container', async () => {
      const items = await requestKubeNovaOptions(kubeNovaContainerParamType, {
        projectValue: form.versionId
      })
      containerOptions.value = items.map((item) => ({
        ...item,
        metadata: {
          ...item.metadata,
          containerType: normalizedKubeNovaContainerType(item.metadata?.containerType)
        }
      }))
      seedOptionsFromForm()
    })
  }

  async function requestKubeNovaOptions(paramType: string, extra: Record<string, any> = {}) {
    const res = await listDevopsDynamicParamOptionsApi({
      projectId: props.projectId,
      systemId: props.systemId,
      environmentId: props.environmentId,
      channelBindingId: form.channelBindingId,
      paramType,
      page: 1,
      pageSize: 100,
      ...extra
    })
    return normalizeArray(res.items || res.data?.items).map(mapDynamicOption)
  }

  async function loadOptions(key: keyof typeof loading, runner: () => Promise<void>) {
    if (loading[key]) return
    loading[key] = true
    errors[key] = ''
    try {
      await runner()
    } catch (error) {
      errors[key] = formatError(error)
    } finally {
      loading[key] = false
    }
  }

  function handleInstanceChange(value: string) {
    const selected = instanceOptions.value.find((item) => item.value === value)
    applyInstanceState(selected)
    clearFrom('project')
    if (!form.baseUrl) {
      errors.instance = '实例地址为空'
      return
    }
    if (!form.credentialReady) {
      errors.instance = form.credentialMessage || '实例未配置可用 token 凭据'
      return
    }
    errors.instance = ''
    queryProjects().catch(() => undefined)
  }

  function refreshInstanceStateFromOptions() {
    if (!form.channelBindingId) return
    const selected = instanceOptions.value.find((item) => item.value === form.channelBindingId)
    if (selected) {
      applyInstanceState(selected)
    }
  }

  function applyInstanceState(selected?: SelectOption) {
    form.channelName = selected?.label || ''
    form.baseUrl = normalizeBaseUrl(stringValue(selected?.metadata?.endpoint))
    form.credentialReady = selected?.metadata?.credentialReady === true
    form.credentialScope = stringValue(selected?.metadata?.credentialScope)
    form.credentialMessage = stringValue(selected?.metadata?.credentialMessage)
    form.allowUseGlobalCredential = selected?.metadata?.allowUseGlobalCredential === true
  }

  function handleProjectChange(value: string) {
    const selected = projectOptions.value.find((item) => item.value === value)
    form.projectName = selected?.label || ''
    clearFrom('cluster')
    if (form.projectId) queryClusters().catch(() => undefined)
  }

  function handleClusterChange(value: string) {
    const selected = clusterOptions.value.find((item) => item.value === value)
    form.clusterName = selected?.label || ''
    form.clusterUuid = stringValue(selected?.metadata?.clusterUuid)
    clearFrom('workspace')
    if (form.projectClusterId) queryWorkspaces().catch(() => undefined)
  }

  function handleWorkspaceChange(value: string) {
    const selected = workspaceOptions.value.find((item) => item.value === value)
    form.workspaceName = selected?.label || ''
    form.clusterUuid = stringValue(selected?.metadata?.clusterUuid, form.clusterUuid)
    form.clusterName = stringValue(selected?.metadata?.clusterName, form.clusterName)
    form.namespace = stringValue(selected?.metadata?.namespace)
    clearFrom('application')
    if (form.workspaceId) queryApplications().catch(() => undefined)
  }

  function handleApplicationChange(value: string) {
    const selected = applicationOptions.value.find((item) => item.value === value)
    form.applicationName = selected?.label || ''
    form.applicationCode = stringValue(selected?.metadata?.nameEn)
    clearFrom('version')
    if (form.applicationId) queryVersions().catch(() => undefined)
  }

  function handleVersionChange(value: string) {
    const selected = versionOptions.value.find((item) => item.value === value)
    form.versionName = selected?.label || ''
    form.resourceName = stringValue(selected?.metadata?.resourceName)
    clearFrom('container')
    if (form.versionId) queryContainers().catch(() => undefined)
  }

  function handleContainerChange(value: string) {
    const selected = containerOptions.value.find((item) => item.value === value)
    form.containerType = normalizedKubeNovaContainerType(selected?.metadata?.containerType)
  }

  function clearFrom(
    level: 'project' | 'cluster' | 'workspace' | 'application' | 'version' | 'container'
  ) {
    if (level === 'project') {
      form.projectId = ''
      form.projectName = ''
      projectOptions.value = []
    }
    if (level === 'project' || level === 'cluster') {
      form.projectClusterId = ''
      form.clusterUuid = ''
      form.clusterName = ''
      clusterOptions.value = []
    }
    if (level === 'project' || level === 'cluster' || level === 'workspace') {
      form.workspaceId = ''
      form.workspaceName = ''
      form.namespace = ''
      workspaceOptions.value = []
    }
    if (
      level === 'project' ||
      level === 'cluster' ||
      level === 'workspace' ||
      level === 'application'
    ) {
      form.applicationId = ''
      form.applicationName = ''
      form.applicationCode = ''
      applicationOptions.value = []
    }
    if (
      level === 'project' ||
      level === 'cluster' ||
      level === 'workspace' ||
      level === 'application' ||
      level === 'version'
    ) {
      form.versionId = ''
      form.versionName = ''
      form.resourceName = ''
      versionOptions.value = []
    }
    form.containerName = ''
    form.containerType = ''
    containerOptions.value = []
  }

  function confirmConfig() {
    const missing = requiredFieldMessage()
    if (missing) {
      ElMessage.warning(missing)
      return
    }
    const payload = {
      channelBindingId: form.channelBindingId,
      channelName: form.channelName,
      baseUrl: normalizeBaseUrl(form.baseUrl),
      credentialReady: form.credentialReady,
      credentialScope: form.credentialScope,
      credentialMessage: form.credentialMessage,
      allowUseGlobalCredential: form.allowUseGlobalCredential,
      projectId: toPositiveNumber(form.projectId),
      projectName: form.projectName,
      projectClusterId: toPositiveNumber(form.projectClusterId),
      clusterUuid: form.clusterUuid,
      clusterName: form.clusterName,
      workspaceId: toPositiveNumber(form.workspaceId),
      workspaceName: form.workspaceName,
      namespace: form.namespace,
      applicationId: toPositiveNumber(form.applicationId),
      applicationName: form.applicationName,
      applicationCode: form.applicationCode,
      versionId: toPositiveNumber(form.versionId),
      versionName: form.versionName,
      resourceName: form.resourceName,
      containerName: form.containerName,
      containerType: form.containerType,
      insecureSkipTls: false,
      timeoutSeconds: normalizeTimeout(form.timeoutSeconds)
    }
    const value = JSON.stringify(payload)
    emit('update:modelValue', value)
    emit('change', value)
    visible.value = false
  }

  function requiredFieldMessage() {
    if (!form.channelBindingId || !form.baseUrl) return '请选择 kube-nova 实例'
    if (!form.credentialReady) return '当前 kube-nova 实例未配置可用 token 凭据'
    if (!form.projectId) return '请选择项目'
    if (!form.projectClusterId) return '请选择集群'
    if (!form.workspaceId) return '请选择工作空间'
    if (!form.applicationId) return '请选择应用'
    if (!form.versionId) return '请选择版本'
    if (!form.containerName) return '请选择容器'
    if (!form.containerType) return '请选择带容器类型的容器'
    return ''
  }

  function mapDynamicOption(option: DynamicParamOption): SelectOption {
    return {
      label: option.label,
      value: option.value,
      metadata: parseMetadata(option.metadata)
    }
  }

  function normalizedKubeNovaContainerType(value: any) {
    return stringValue(value, 'main')
  }

  function normalizeArray<T>(value: T[] | undefined) {
    return Array.isArray(value) ? value : []
  }

  function parseDeployConfig(raw?: string) {
    const text = String(raw || '').trim()
    if (!text) return undefined
    try {
      const parsed = JSON.parse(text)
      return parsed && typeof parsed === 'object' ? parsed : undefined
    } catch {
      return undefined
    }
  }

  function parseMetadata(raw?: string) {
    if (!raw) return undefined
    try {
      return JSON.parse(raw)
    } catch {
      return undefined
    }
  }

  function normalizeBaseUrl(value: string) {
    return String(value || '')
      .trim()
      .replace(/\/+$/, '')
  }

  function normalizeTimeout(value: any) {
    const num = Number(value || 60)
    if (!Number.isFinite(num) || num < 1) return 60
    return Math.min(Math.floor(num), 3600)
  }

  function toPositiveNumber(value: string) {
    const num = Number(value)
    return Number.isFinite(num) && num > 0 ? num : undefined
  }

  function stringValue(value: any, defaultValue = '') {
    const text = value === undefined || value === null ? '' : String(value).trim()
    return text || defaultValue
  }

  function formatError(error: unknown) {
    if (error instanceof Error && error.message) return error.message
    if (typeof error === 'object' && error && 'message' in error) {
      return String((error as { message?: unknown }).message || '加载失败')
    }
    return '加载失败'
  }
</script>

<style scoped>
  .kube-nova-deploy-config-editor {
    width: 100%;
  }

  :global(.kube-nova-deploy-dialog) {
    max-width: calc(100vw - 48px);
  }

  :global(.kube-nova-deploy-dialog .el-dialog__body) {
    padding: 12px 18px 0;
    overflow: hidden;
  }

  :global(.kube-nova-deploy-dialog .el-dialog__footer) {
    padding: 12px 18px 16px;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .deploy-config-summary {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    min-height: 54px;
    padding: 10px 12px;
    border: 1px solid var(--el-border-color);
    border-radius: 6px;
    background: var(--el-fill-color-blank);
  }

  .deploy-config-summary.empty {
    background: var(--el-fill-color-light);
  }

  .summary-text {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .summary-text strong,
  .summary-text span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .summary-text strong {
    font-size: 14px;
    color: var(--el-text-color-primary);
  }

  .summary-text span {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .deploy-dialog-body {
    height: min(68vh, 640px);
    min-height: 500px;
    overflow: hidden;
  }

  .deploy-dialog-layout {
    display: grid;
    grid-template-columns: 260px minmax(0, 1fr);
    gap: 14px;
    height: 100%;
    min-height: 0;
  }

  .deploy-side,
  .deploy-main {
    min-height: 0;
  }

  .deploy-side {
    overflow-y: auto;
    padding-right: 2px;
  }

  .deploy-main {
    overflow: hidden;
  }

  .deploy-overview {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 8px;
    align-items: stretch;
    margin-bottom: 10px;
    padding: 12px;
    border: 1px solid var(--el-border-color);
    border-radius: 6px;
    background: var(--el-fill-color-light);
  }

  .overview-main {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .overview-main strong,
  .overview-main span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .overview-main strong {
    color: var(--el-text-color-primary);
    font-size: 15px;
  }

  .overview-main span {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }

  .credential-state {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: fit-content;
    min-width: 96px;
    height: 28px;
    padding: 0 10px;
    border: 1px solid var(--el-border-color);
    border-radius: 6px;
    background: var(--el-fill-color-blank);
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1;
    white-space: nowrap;
  }

  .credential-state.ready {
    border-color: var(--el-color-success-light-5);
    background: var(--el-color-success-light-9);
    color: var(--el-color-success);
  }

  .credential-state.warning {
    border-color: var(--el-color-warning-light-5);
    background: var(--el-color-warning-light-9);
    color: var(--el-color-warning-dark-2);
  }

  .deploy-flow {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 6px;
    margin-bottom: 10px;
  }

  .deploy-flow-node {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    padding: 7px 9px;
    border: 1px solid var(--el-border-color);
    border-radius: 6px;
    background: var(--el-fill-color-light);
    color: var(--el-text-color-secondary);
  }

  .deploy-flow-node span {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex: 0 0 20px;
    width: 20px;
    height: 20px;
    border-radius: 50%;
    background: var(--el-fill-color-blank);
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1;
  }

  .deploy-flow-node strong {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 12px;
    font-weight: 600;
  }

  .deploy-flow-node.done {
    border-color: var(--el-color-primary-light-5);
    background: var(--el-color-primary-light-9);
    color: var(--el-color-primary);
  }

  .deploy-flow-node.done span {
    background: var(--el-color-primary);
    color: #fff;
  }

  .deploy-flow-node.active {
    border-color: var(--el-color-warning-light-5);
    background: var(--el-color-warning-light-9);
    color: var(--el-color-warning-dark-2);
  }

  .deploy-form {
    height: 100%;
    max-height: none;
    overflow-y: auto;
    padding-right: 8px;
  }

  .target-grid {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 6px;
    margin-bottom: 0;
  }

  .target-item {
    min-width: 0;
    padding: 7px 9px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    background: var(--el-fill-color-blank);
  }

  .target-item span,
  .target-item strong {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .target-item span {
    margin-bottom: 3px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }

  .target-item strong {
    color: var(--el-text-color-primary);
    font-size: 13px;
    font-weight: 600;
  }

  .form-section {
    padding: 12px 12px 0;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    background: var(--el-fill-color-blank);
  }

  .form-section + .form-section {
    margin-top: 10px;
  }

  .section-title {
    display: flex;
    flex-direction: column;
    gap: 3px;
    margin-bottom: 10px;
  }

  .section-title strong {
    color: var(--el-text-color-primary);
    font-size: 14px;
  }

  .section-title span {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }

  .field-stack {
    width: 100%;
  }

  .select-action-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 68px;
    gap: 8px;
    width: 100%;
  }

  .credential-hint {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.4;
  }

  .credential-hint.warning {
    color: var(--el-color-warning-dark-2);
  }

  .deploy-option {
    display: flex;
    flex-direction: column;
    gap: 2px;
    justify-content: center;
    min-height: 52px;
    overflow: visible;
    line-height: 1.3;
  }

  .deploy-option span,
  .deploy-option small {
    overflow: visible;
    text-overflow: clip;
    white-space: normal;
    word-break: break-word;
  }

  .deploy-option small {
    color: var(--el-text-color-secondary);
  }

  :global(.devops-deploy-option-dropdown.el-select__popper:not(.el-tree-select__popper)) {
    overflow: hidden !important;
    background: #ffffff !important;
    border: 1px solid #e2e8f0;
    border-radius: 14px;
    box-shadow: 0 16px 34px rgb(15 23 42 / 12%);
  }

  :global(
    .devops-deploy-option-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__wrap
  ) {
    max-height: 260px !important;
    background: #ffffff !important;
  }

  :global(
    .devops-deploy-option-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__list
  ) {
    padding: 4px 0 8px !important;
    background: #ffffff !important;
  }

  :global(
    .devops-deploy-option-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__list
      .el-select-dropdown__item
  ) {
    height: auto !important;
    min-height: 44px !important;
    padding: 10px 12px !important;
    overflow: visible !important;
    line-height: 1.35 !important;
    color: #334155 !important;
    white-space: normal !important;
    background: #ffffff !important;
    border-radius: 0 !important;
    margin-bottom: 0 !important;
  }

  :global(
    .devops-deploy-option-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__list
      .el-select-dropdown__item.is-hovering
  ),
  :global(
    .devops-deploy-option-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__list
      .el-select-dropdown__item:hover
  ) {
    background: #f8fafc !important;
  }

  :global(
    .devops-deploy-option-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__list
      .el-select-dropdown__item.is-selected
  ) {
    font-weight: 600 !important;
    color: #0f172a !important;
    background: #ffffff !important;
  }

  :deep(.deploy-form .el-form-item) {
    margin-bottom: 12px;
  }

  :deep(.deploy-form .el-select__wrapper) {
    min-height: 42px;
    padding: 10px 12px;
    background: #ffffff;
    border-radius: 14px;
    box-shadow: 0 0 0 1px #e2e8f0 inset;
    transition:
      box-shadow 0.18s ease,
      background-color 0.18s ease;
  }

  :deep(.deploy-form .el-select__wrapper:hover) {
    box-shadow: 0 0 0 1px #94a3b8 inset;
  }

  :deep(.deploy-form .el-select__wrapper.is-focused) {
    box-shadow:
      0 0 0 1px #2563eb inset,
      0 0 0 4px rgb(37 99 235 / 14%);
  }

  :deep(.deploy-form .el-select__selection) {
    min-width: 0;
  }

  :deep(.deploy-form .el-select__caret) {
    width: 16px;
    height: 16px;
    color: #334155;
  }

  :deep(.deploy-form .el-select__caret svg) {
    width: 14px;
    height: 14px;
  }

  @media (max-width: 900px) {
    .deploy-dialog-body {
      height: min(74vh, 720px);
    }

    .deploy-dialog-layout {
      grid-template-columns: minmax(0, 1fr);
    }

    .deploy-side {
      max-height: 240px;
    }

    .target-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .deploy-config-summary {
      align-items: stretch;
      flex-direction: column;
    }

    .select-action-row {
      grid-template-columns: minmax(0, 1fr);
    }
  }
</style>
