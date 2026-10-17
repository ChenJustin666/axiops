<template>
  <div class="kubernetes-deploy-config-editor">
    <div class="config-trigger">
      <div class="config-summary" :class="{ empty: !hasConfig }">
        <strong>{{ summaryTitle }}</strong>
        <span>{{ summaryDesc }}</span>
      </div>
      <ElButton size="small" type="primary" plain :disabled="disabled" @click="openDialog">
        {{ hasConfig ? '修改配置' : '选择目标' }}
      </ElButton>
    </div>

    <ElDialog
      v-model="visible"
      title="Kubernetes 部署配置"
      width="920px"
      append-to-body
      align-center
      class="kubernetes-deploy-dialog"
      :close-on-click-modal="false"
      destroy-on-close
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
                  ready: credentialReady,
                  warning: !!form.channelBindingId && !credentialReady
                }"
              >
                {{ credentialStatusText }}
              </div>
            </div>

            <div class="deploy-flow">
              <div
                v-for="step in flowSteps"
                :key="step.key"
                class="deploy-flow-node"
                :class="{ active: step.active, done: step.done }"
              >
                <span>{{ step.index }}</span>
                <strong>{{ step.label }}</strong>
              </div>
            </div>

            <div class="selected-target-grid">
              <div v-for="item in selectedTargetItems" :key="item.label" class="target-item">
                <span>{{ item.label }}</span>
                <strong :title="item.value">{{ item.value || '未选择' }}</strong>
              </div>
            </div>
          </aside>

          <div class="deploy-main">
            <ElAlert
              v-if="dialogError"
              class="deploy-alert"
              type="error"
              :title="dialogError"
              :closable="false"
              show-icon
            />

            <ElAlert
              v-if="form.credentialMessage"
              class="deploy-alert"
              :type="credentialReady ? 'success' : 'warning'"
              :title="form.credentialMessage"
              :closable="false"
              show-icon
            />

            <ElForm label-width="104px" class="deploy-form">
              <section class="form-section">
                <div class="section-title">
                  <strong>集群实例</strong>
                  <span>渠道资源来自当前 DevOps 项目，凭据由后端按项目绑定解析</span>
                </div>
                <ElFormItem label="渠道资源" required>
                  <div class="field-stack">
                    <div class="select-action-row">
                      <ElSelect
                        v-model="form.channelBindingId"
                        filterable
                        clearable
                        remote-show-suffix
                        popper-class="devops-deploy-option-dropdown"
                        :teleported="true"
                        fit-input-width
                        :loading="loading.instance"
                        loading-text="加载中..."
                        :disabled="disabled || !projectId"
                        :placeholder="
                          projectId ? '请选择 Kubernetes 渠道资源' : '请选择 DevOps 项目'
                        "
                        :no-data-text="errors.instance ? '加载失败' : instanceEmptyText"
                        style="width: 100%"
                        @visible-change="(shown: boolean) => shown && loadInstances()"
                        @change="handleInstanceChange"
                      >
                        <template #loading>
                          <div class="devops-select-loading">
                            <span class="devops-select-loading__icon"></span>
                            <span>加载中...</span>
                          </div>
                        </template>
                        <ElOption
                          v-for="item in instanceOptions"
                          :key="item.value"
                          :label="item.label"
                          :value="item.value"
                        >
                          <div class="deploy-option">
                            <span>{{ item.label }}</span>
                            <small>{{ item.metadata?.endpoint || '未配置 API 地址' }}</small>
                          </div>
                        </ElOption>
                      </ElSelect>
                      <ElButton
                        :icon="RefreshRight"
                        :loading="loading.instance"
                        :disabled="disabled || !projectId"
                        @click="loadInstances(true)"
                      >
                        刷新
                      </ElButton>
                    </div>
                    <p v-if="errors.instance" class="field-error">{{ errors.instance }}</p>
                  </div>
                </ElFormItem>
              </section>

              <section class="form-section">
                <div class="section-title">
                  <strong>部署目标</strong>
                  <span>按命名空间、资源类型、资源、容器逐级选择镜像替换目标</span>
                </div>

                <ElFormItem label="命名空间" required>
                  <div class="field-stack">
                    <div class="select-action-row">
                      <ElSelect
                        popper-class="devops-deploy-option-dropdown"
                        v-model="form.namespace"
                        filterable
                        clearable
                        :loading="loading.namespace"
                        loading-text="加载中..."
                        :disabled="selectDisabled('namespace')"
                        :no-data-text="errors.namespace ? '加载失败' : '暂无可选命名空间'"
                        placeholder="请选择命名空间"
                        style="width: 100%"
                        @visible-change="(shown: boolean) => shown && loadNamespaces()"
                        @change="handleNamespaceChange"
                      >
                        <template #loading>
                          <div class="devops-select-loading">
                            <span class="devops-select-loading__icon"></span>
                            <span>加载中...</span>
                          </div>
                        </template>
                        <ElOption
                          v-for="item in namespaceOptions"
                          :key="item.value"
                          :label="item.label"
                          :value="item.value"
                        />
                      </ElSelect>
                      <ElButton
                        :icon="RefreshRight"
                        :loading="loading.namespace"
                        :disabled="selectDisabled('namespace')"
                        @click="loadNamespaces(true)"
                      >
                        刷新
                      </ElButton>
                    </div>
                    <p v-if="errors.namespace" class="field-error">{{ errors.namespace }}</p>
                  </div>
                </ElFormItem>

                <ElFormItem label="资源类型" required>
                  <ElSelect
                    popper-class="devops-deploy-option-dropdown"
                    v-model="form.resourceType"
                    filterable
                    clearable
                    :disabled="selectDisabled('resourceType')"
                    placeholder="请选择资源类型"
                    style="width: 100%"
                    @change="handleResourceTypeChange"
                  >
                    <ElOption
                      v-for="item in resourceTypeOptions"
                      :key="item.value"
                      :label="item.label"
                      :value="item.value"
                    />
                  </ElSelect>
                </ElFormItem>

                <ElFormItem label="资源名称" required>
                  <div class="field-stack">
                    <div class="select-action-row">
                      <ElSelect
                        popper-class="devops-deploy-option-dropdown"
                        v-model="form.resourceName"
                        filterable
                        clearable
                        :loading="loading.resource"
                        loading-text="加载中..."
                        :disabled="selectDisabled('resource')"
                        :no-data-text="errors.resource ? '加载失败' : '暂无可选资源'"
                        placeholder="请选择资源"
                        style="width: 100%"
                        @visible-change="(shown: boolean) => shown && loadResources()"
                        @change="handleResourceChange"
                      >
                        <template #loading>
                          <div class="devops-select-loading">
                            <span class="devops-select-loading__icon"></span>
                            <span>加载中...</span>
                          </div>
                        </template>
                        <ElOption
                          v-for="item in resourceOptions"
                          :key="item.value"
                          :label="item.label"
                          :value="item.value"
                        />
                      </ElSelect>
                      <ElButton
                        :icon="RefreshRight"
                        :loading="loading.resource"
                        :disabled="selectDisabled('resource')"
                        @click="loadResources(true)"
                      >
                        刷新
                      </ElButton>
                    </div>
                    <p v-if="errors.resource" class="field-error">{{ errors.resource }}</p>
                  </div>
                </ElFormItem>

                <ElFormItem label="容器" required>
                  <div class="field-stack">
                    <div class="select-action-row">
                      <ElSelect
                        v-model="form.containerName"
                        filterable
                        clearable
                        popper-class="devops-deploy-option-dropdown"
                        :teleported="true"
                        fit-input-width
                        :loading="loading.container"
                        loading-text="加载中..."
                        :disabled="selectDisabled('container')"
                        :no-data-text="errors.container ? '加载失败' : '暂无可选容器'"
                        placeholder="请选择容器"
                        style="width: 100%"
                        @visible-change="(shown: boolean) => shown && loadContainers()"
                        @change="handleContainerChange"
                      >
                        <template #loading>
                          <div class="devops-select-loading">
                            <span class="devops-select-loading__icon"></span>
                            <span>加载中...</span>
                          </div>
                        </template>
                        <ElOption
                          v-for="item in containerOptions"
                          :key="item.value"
                          :label="item.label"
                          :value="item.value"
                        >
                          <div class="deploy-option">
                            <span>{{ item.label }}</span>
                            <small>{{ item.metadata?.image || '未读取到镜像' }}</small>
                          </div>
                        </ElOption>
                      </ElSelect>
                      <ElButton
                        :icon="RefreshRight"
                        :loading="loading.container"
                        :disabled="selectDisabled('container')"
                        @click="loadContainers(true)"
                      >
                        刷新
                      </ElButton>
                    </div>
                    <p v-if="errors.container" class="field-error">{{ errors.container }}</p>
                  </div>
                </ElFormItem>
              </section>

              <section class="form-section">
                <div class="section-title">
                  <strong>执行选项</strong>
                  <span>镜像地址在 Pipeline 层通过独立变量传入</span>
                </div>
                <div class="runtime-grid">
                  <ElFormItem label="超时秒数">
                    <ElInputNumber
                      v-model="form.rolloutTimeoutSeconds"
                      :min="1"
                      :disabled="disabled"
                      controls-position="right"
                      style="width: 100%"
                    />
                  </ElFormItem>
                  <ElFormItem label="健康检查">
                    <ElSwitch v-model="form.healthCheckEnabled" :disabled="disabled" />
                  </ElFormItem>
                  <ElFormItem label="自动回滚">
                    <ElSwitch v-model="form.rollbackEnabled" :disabled="disabled" />
                  </ElFormItem>
                </div>
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
  import { RefreshRight } from '@element-plus/icons-vue'
  import { listDevopsDynamicParamOptionsApi, type DynamicParamOption } from '@/api/devops/pipeline'

  defineOptions({ name: 'DevopsKubernetesDeployConfigEditor' })

  type SelectOption = {
    label: string
    value: string
    metadata?: Record<string, any>
  }

  type DeployForm = {
    channelBindingId: string
    channelName: string
    endpoint: string
    credentialReady: boolean
    credentialScope: string
    credentialType: string
    credentialMessage: string
    allowUseGlobalCredential: boolean
    insecureSkipTLS: boolean
    namespace: string
    resourceType: string
    resourceName: string
    containerName: string
    containerType: string
    currentImage: string
    healthCheckEnabled: boolean
    rollbackEnabled: boolean
    rolloutTimeoutSeconds: number
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
  const namespaceOptions = ref<SelectOption[]>([])
  const resourceOptions = ref<SelectOption[]>([])
  const containerOptions = ref<SelectOption[]>([])

  const resourceTypeOptions = [
    { label: 'Deployment', value: 'deployment' },
    { label: 'StatefulSet', value: 'statefulset' },
    { label: 'DaemonSet', value: 'daemonset' },
    { label: 'CronJob', value: 'cronjob' }
  ]

  const loading = reactive({
    instance: false,
    namespace: false,
    resource: false,
    container: false
  })
  const errors = reactive({
    instance: '',
    namespace: '',
    resource: '',
    container: ''
  })

  const form = reactive<DeployForm>(defaultForm())

  const hasConfig = computed(() => Boolean(parseDeployConfig(props.modelValue)?.channelBindingId))
  const parsedConfig = computed(() => parseDeployConfig(props.modelValue))
  const credentialReady = computed(() => form.credentialReady === true)
  const instanceEmptyText = computed(() =>
    props.projectId ? '当前项目未绑定 Kubernetes 渠道资源' : '请选择 DevOps 项目'
  )
  const dialogError = computed(
    () => errors.instance || errors.namespace || errors.resource || errors.container
  )
  const summaryTitle = computed(() => {
    const config = parsedConfig.value
    if (!config?.channelBindingId) return '未配置 Kubernetes 部署目标'
    return `${config.namespace || '-'} / ${config.resourceType || '-'} / ${config.resourceName || '-'}`
  })
  const summaryDesc = computed(() => {
    const config = parsedConfig.value
    if (!config?.channelBindingId) return '选择 Kubernetes 渠道、命名空间、资源和容器'
    return `容器：${config.containerName || '-'}，凭证运行时由 Jenkins 注入`
  })
  const dialogOverviewTitle = computed(() => {
    if (!form.channelBindingId) return '选择 Kubernetes 部署目标'
    return form.containerName
      ? `目标容器：${form.containerName}`
      : form.channelName || '选择 Kubernetes 部署目标'
  })
  const dialogOverviewDescription = computed(() => {
    const parts = [
      form.channelName || form.endpoint,
      form.namespace,
      form.resourceType,
      form.resourceName,
      form.containerName
    ].filter(Boolean)
    return parts.length ? parts.join(' / ') : '按命名空间、资源类型、资源、容器逐级定位'
  })
  const selectedTargetItems = computed(() => [
    { label: '实例', value: form.channelName || form.endpoint },
    { label: '命名空间', value: form.namespace },
    { label: '资源类型', value: resourceTypeLabel(form.resourceType) },
    { label: '资源名称', value: form.resourceName },
    { label: '容器', value: form.containerName },
    { label: '当前镜像', value: form.currentImage }
  ])
  const credentialStatusText = computed(() => {
    if (!form.channelBindingId) return '未选择实例'
    if (credentialReady.value) {
      return form.credentialScope === 'project' ? '项目凭据' : '全局凭据'
    }
    return '凭据不可用'
  })
  const flowSteps = computed(() => {
    const items = [
      { key: 'instance', label: '实例', done: Boolean(form.channelBindingId) },
      { key: 'namespace', label: '命名空间', done: Boolean(form.namespace) },
      { key: 'resourceType', label: '类型', done: Boolean(form.resourceType) },
      { key: 'resource', label: '资源', done: Boolean(form.resourceName) },
      { key: 'container', label: '容器', done: Boolean(form.containerName) }
    ]
    const activeIndex = items.findIndex((item) => !item.done)
    return items.map((item, index) => ({
      ...item,
      index: index + 1,
      active: activeIndex === index || (activeIndex === -1 && index === items.length - 1)
    }))
  })
  const confirmDisabled = computed(
    () =>
      props.disabled ||
      !form.channelBindingId ||
      !credentialReady.value ||
      !form.namespace ||
      !form.resourceType ||
      !form.resourceName ||
      !form.containerName ||
      !form.containerType
  )

  function defaultForm(): DeployForm {
    return {
      channelBindingId: '',
      channelName: '',
      endpoint: '',
      credentialReady: false,
      credentialScope: '',
      credentialType: '',
      credentialMessage: '',
      allowUseGlobalCredential: false,
      insecureSkipTLS: true,
      namespace: '',
      resourceType: 'deployment',
      resourceName: '',
      containerName: '',
      containerType: '',
      currentImage: '',
      healthCheckEnabled: true,
      rollbackEnabled: true,
      rolloutTimeoutSeconds: 300
    }
  }

  async function openDialog() {
    Object.assign(form, defaultForm(), parseDeployConfig(props.modelValue) || {})
    clearErrors()
    visible.value = true
    await loadInstances()
    restoreSelectedInstance()
    if (form.channelBindingId && credentialReady.value) {
      await hydrateCascadeOptions()
    }
  }

  async function hydrateCascadeOptions() {
    await loadNamespaces()
    if (form.namespace && form.resourceType) {
      await loadResources()
    }
    if (form.namespace && form.resourceType && form.resourceName) {
      await loadContainers()
    }
  }

  async function loadInstances(force = false) {
    if (!props.projectId || (instanceOptions.value.length && !force)) return
    loading.instance = true
    errors.instance = ''
    try {
      instanceOptions.value = await requestOptions('channelGroupCode', {
        configTypeCode: 'deploy_target',
        componentValue: 'kubernetes',
        mappingField: 'endpoint'
      })
      restoreSelectedInstance()
    } catch (error: unknown) {
      errors.instance = normalizeLoadError(error, '加载 Kubernetes 渠道资源失败')
    } finally {
      loading.instance = false
    }
  }

  async function loadNamespaces(force = false) {
    if (!form.channelBindingId || !credentialReady.value) return
    if (namespaceOptions.value.length && !force) return
    loading.namespace = true
    errors.namespace = ''
    try {
      namespaceOptions.value = await requestOptions('kubernetesNamespace', {
        channelBindingId: form.channelBindingId
      })
    } catch (error: unknown) {
      errors.namespace = normalizeLoadError(error, '加载 Kubernetes 命名空间失败')
    } finally {
      loading.namespace = false
    }
  }

  async function loadResources(force = false) {
    if (!form.channelBindingId || !form.namespace || !form.resourceType || !credentialReady.value) {
      return
    }
    if (resourceOptions.value.length && !force) return
    loading.resource = true
    errors.resource = ''
    try {
      resourceOptions.value = await requestOptions('kubernetesResource', {
        channelBindingId: form.channelBindingId,
        projectValue: form.namespace,
        componentValue: form.resourceType
      })
    } catch (error: unknown) {
      errors.resource = normalizeLoadError(error, '加载 Kubernetes 资源失败')
    } finally {
      loading.resource = false
    }
  }

  async function loadContainers(force = false) {
    if (
      !form.channelBindingId ||
      !form.namespace ||
      !form.resourceType ||
      !form.resourceName ||
      !credentialReady.value
    ) {
      return
    }
    if (containerOptions.value.length && !force) return
    loading.container = true
    errors.container = ''
    try {
      containerOptions.value = await requestOptions('kubernetesContainer', {
        channelBindingId: form.channelBindingId,
        projectValue: form.namespace,
        componentValue: form.resourceType,
        configTypeCode: form.resourceName
      })
    } catch (error: unknown) {
      errors.container = normalizeLoadError(error, '加载 Kubernetes 容器失败')
    } finally {
      loading.container = false
    }
  }

  async function requestOptions(paramType: string, extra: Record<string, any> = {}) {
    const res = await listDevopsDynamicParamOptionsApi({
      projectId: props.projectId,
      systemId: props.systemId,
      environmentId: props.environmentId,
      paramType,
      page: 1,
      pageSize: 100,
      ...extra
    })
    return normalizeArray(res.items || res.data?.items).map(mapDynamicOption)
  }

  function handleInstanceChange(value: string) {
    const option = instanceOptions.value.find((item) => item.value === value)
    applyInstanceOption(option)
    form.namespace = ''
    form.resourceName = ''
    form.containerName = ''
    namespaceOptions.value = []
    resourceOptions.value = []
    containerOptions.value = []
    errors.namespace = ''
    errors.resource = ''
    errors.container = ''
    if (form.channelBindingId && credentialReady.value) {
      loadNamespaces(true)
    }
  }

  function handleNamespaceChange() {
    form.resourceName = ''
    form.containerName = ''
    resourceOptions.value = []
    containerOptions.value = []
    errors.resource = ''
    errors.container = ''
    loadResources(true)
  }

  function handleResourceTypeChange() {
    form.resourceName = ''
    form.containerName = ''
    resourceOptions.value = []
    containerOptions.value = []
    errors.resource = ''
    errors.container = ''
    loadResources(true)
  }

  function handleResourceChange() {
    const option = resourceOptions.value.find((item) => item.value === form.resourceName)
    form.containerName = ''
    form.containerType = ''
    form.currentImage = ''
    containerOptions.value = []
    errors.container = ''
    if (option?.metadata?.resourceType) {
      form.resourceType = stringValue(option.metadata.resourceType, form.resourceType)
    }
    loadContainers(true)
  }

  function handleContainerChange() {
    const option = containerOptions.value.find((item) => item.value === form.containerName)
    form.containerType = stringValue(option?.metadata?.containerType)
    form.currentImage = stringValue(option?.metadata?.image)
  }

  function restoreSelectedInstance() {
    if (!form.channelBindingId) return
    const option = instanceOptions.value.find((item) => item.value === form.channelBindingId)
    applyInstanceOption(option)
  }

  function applyInstanceOption(option?: SelectOption) {
    form.channelName = option?.label || form.channelName || ''
    form.endpoint = stringValue(option?.metadata?.endpoint, form.endpoint)
    form.credentialReady = Boolean(option?.metadata?.credentialReady ?? form.credentialReady)
    form.credentialScope = stringValue(option?.metadata?.credentialScope, form.credentialScope)
    form.credentialType = stringValue(option?.metadata?.credentialType, form.credentialType)
    form.credentialMessage = stringValue(
      option?.metadata?.credentialMessage,
      form.credentialMessage
    )
    form.allowUseGlobalCredential = Boolean(
      option?.metadata?.allowUseGlobalCredential ?? form.allowUseGlobalCredential
    )
    form.insecureSkipTLS = Boolean(option?.metadata?.insecureSkipTLS ?? form.insecureSkipTLS)
  }

  function selectDisabled(type: 'namespace' | 'resourceType' | 'resource' | 'container') {
    if (props.disabled || !form.channelBindingId || !credentialReady.value) return true
    if (type === 'namespace') return false
    if (type === 'resourceType') return !form.namespace
    if (type === 'resource') return !form.namespace || !form.resourceType
    return !form.namespace || !form.resourceType || !form.resourceName
  }

  function confirmConfig() {
    if (confirmDisabled.value) return
    const value = JSON.stringify({
      channelBindingId: form.channelBindingId,
      channelName: form.channelName,
      endpoint: form.endpoint,
      clusterServer: form.endpoint,
      credentialContentType: 'auto',
      credentialScope: form.credentialScope,
      credentialType: form.credentialType,
      namespace: form.namespace,
      resourceType: form.resourceType,
      resourceName: form.resourceName,
      containerName: form.containerName,
      containerType: form.containerType,
      currentImage: form.currentImage,
      healthCheckEnabled: form.healthCheckEnabled,
      rollbackEnabled: form.rollbackEnabled,
      rolloutTimeoutSeconds: form.rolloutTimeoutSeconds || 300,
      insecureSkipTLS: form.insecureSkipTLS
    })
    emit('update:modelValue', value)
    emit('change', value)
    visible.value = false
  }

  function clearErrors() {
    errors.instance = ''
    errors.namespace = ''
    errors.resource = ''
    errors.container = ''
  }

  function normalizeLoadError(error: unknown, fallback: string) {
    const message = stringValue((error as { message?: string })?.message, fallback)
    const lowerMessage = message.toLowerCase()
    if (lowerMessage.includes('namespaces is forbidden')) {
      return '当前 Kubernetes 凭据没有集群级 list namespaces 权限。请给该 ServiceAccount 绑定可读取命名空间的 ClusterRole/ClusterRoleBinding，或更换具备命名空间列表权限的凭据。'
    }
    if (
      message === fallback ||
      message.startsWith(`${fallback}：`) ||
      message.startsWith(`${fallback}:`)
    ) {
      return message
    }
    return `${fallback}：${message}`
  }

  function resourceTypeLabel(value: string) {
    return resourceTypeOptions.find((item) => item.value === value)?.label || value
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

  function mapDynamicOption(option: DynamicParamOption): SelectOption {
    return {
      label: option.label,
      value: option.value,
      metadata: parseMetadata(option.metadata)
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

  function normalizeArray<T>(value: T[] | undefined) {
    return Array.isArray(value) ? value : []
  }

  function stringValue(value: any, defaultValue = '') {
    if (value === undefined || value === null) return defaultValue
    const text = String(value).trim()
    return text || defaultValue
  }
</script>

<style scoped lang="scss">
  .kubernetes-deploy-config-editor {
    width: 100%;
  }

  .config-trigger {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    padding: 10px 12px;
    background: #f8fafc;
    border: 1px solid #dbe3ee;
    border-radius: 8px;
  }

  .config-summary {
    display: flex;
    flex: 1;
    min-width: 0;
    flex-direction: column;
    gap: 4px;

    strong,
    span {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      color: #0f172a;
      font-size: 13px;
      font-weight: 600;
    }

    span {
      color: #64748b;
      font-size: 12px;
    }

    &.empty strong {
      color: #64748b;
    }
  }

  :deep(.kubernetes-deploy-dialog .el-dialog__body) {
    padding: 0 20px;
  }

  :deep(.kubernetes-deploy-dialog .el-dialog__footer) {
    padding: 14px 20px 16px;
    border-top: 1px solid #e5e7eb;
  }

  .deploy-dialog-body {
    max-height: min(72vh, 720px);
    overflow: hidden;
  }

  .deploy-dialog-layout {
    display: grid;
    grid-template-columns: 260px minmax(0, 1fr);
    gap: 16px;
    min-height: 0;
  }

  .deploy-side,
  .deploy-main {
    min-width: 0;
  }

  .deploy-side {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding-bottom: 14px;
  }

  .deploy-main {
    display: flex;
    max-height: min(72vh, 720px);
    padding: 0 4px 14px 0;
    overflow-y: auto;
    flex-direction: column;
    gap: 12px;
  }

  .deploy-overview {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 14px;
    background: #f8fafc;
    border: 1px solid #dbe3ee;
    border-radius: 8px;
  }

  .overview-main {
    display: flex;
    min-width: 0;
    flex-direction: column;
    gap: 4px;

    strong,
    span {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      color: #0f172a;
      font-size: 15px;
      font-weight: 600;
    }

    span {
      color: #64748b;
      font-size: 12px;
    }
  }

  .credential-state {
    display: inline-flex;
    align-items: center;
    width: fit-content;
    min-height: 24px;
    padding: 0 9px;
    color: #475569;
    font-size: 12px;
    line-height: 1;
    background: #ffffff;
    border: 1px solid #cbd5e1;
    border-radius: 8px;

    &.ready {
      color: #047857;
      border-color: #86efac;
    }

    &.warning {
      color: #b45309;
      border-color: #fbbf24;
    }
  }

  .deploy-flow {
    display: grid;
    grid-template-columns: 1fr;
    gap: 8px;
  }

  .deploy-flow-node {
    display: flex;
    gap: 8px;
    align-items: center;
    min-width: 0;
    height: 36px;
    padding: 0 10px;
    color: #64748b;
    background: #ffffff;
    border: 1px solid #dbe3ee;
    border-radius: 8px;

    span {
      display: inline-flex;
      flex: 0 0 22px;
      align-items: center;
      justify-content: center;
      width: 22px;
      height: 22px;
      color: #64748b;
      font-size: 12px;
      line-height: 1;
      background: #f8fafc;
      border: 1px solid #dbe3ee;
      border-radius: 50%;
    }

    strong {
      overflow: hidden;
      font-size: 12px;
      font-weight: 600;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    &.done {
      color: #047857;
      border-color: #a7f3d0;

      span {
        color: #047857;
        border-color: #a7f3d0;
      }
    }

    &.active {
      color: #1d4ed8;
      background: #eff6ff;
      border-color: #93c5fd;
    }
  }

  .selected-target-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: 8px;
  }

  .target-item {
    min-width: 0;
    padding: 9px 10px;
    background: #ffffff;
    border: 1px solid #e2e8f0;
    border-radius: 8px;

    span,
    strong {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    span {
      color: #64748b;
      font-size: 12px;
    }

    strong {
      margin-top: 4px;
      color: #111827;
      font-size: 13px;
      font-weight: 600;
    }
  }

  .deploy-alert {
    margin: 0;
  }

  .deploy-form {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .form-section {
    padding: 16px;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
  }

  .section-title {
    display: flex;
    min-width: 0;
    flex-direction: column;
    gap: 4px;
    margin-bottom: 16px;

    strong {
      color: #0f172a;
      font-size: 14px;
      font-weight: 600;
    }

    span {
      color: #64748b;
      font-size: 12px;
      line-height: 18px;
    }
  }

  .field-stack {
    display: flex;
    width: 100%;
    min-width: 0;
    flex-direction: column;
    gap: 8px;
  }

  .select-action-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 76px;
    gap: 10px;
    width: 100%;
  }

  .runtime-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.2fr) minmax(0, 1fr) minmax(0, 1fr);
    gap: 14px;
    align-items: center;
  }

  :deep(.runtime-grid .el-form-item + .el-form-item) {
    margin-top: 0;
  }

  :deep(.runtime-grid .el-form-item__content) {
    min-height: 36px;
    align-items: center;
  }

  .field-error {
    margin: 0;
    color: #b91c1c;
    font-size: 12px;
    line-height: 18px;
  }

  .deploy-option {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    min-width: 0;
    min-height: 52px;
    overflow: visible;

    span,
    small {
      overflow: visible;
      text-overflow: clip;
      white-space: normal;
      word-break: break-word;
    }

    small {
      color: #94a3b8;
      font-size: 12px;
    }
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

  :deep(.el-form-item) {
    margin-bottom: 0;
  }

  :deep(.form-section .el-form-item + .el-form-item) {
    margin-top: 14px;
  }

  :deep(.form-section .el-form-item__label) {
    padding-right: 12px;
    color: #374151;
    line-height: 32px;
  }

  :deep(.el-input__wrapper),
  :deep(.el-select__wrapper) {
    min-height: 36px;
  }

  :deep(.el-select__wrapper) {
    min-height: 42px;
    padding: 10px 12px;
    background: #ffffff;
    border-radius: 14px;
    box-shadow: 0 0 0 1px #e2e8f0 inset;
    transition:
      box-shadow 0.18s ease,
      background-color 0.18s ease;
  }

  :deep(.el-select__wrapper:hover) {
    box-shadow: 0 0 0 1px #94a3b8 inset;
  }

  :deep(.el-select__wrapper.is-focused) {
    box-shadow:
      0 0 0 1px #2563eb inset,
      0 0 0 4px rgb(37 99 235 / 14%);
  }

  :deep(.el-select__selection) {
    min-width: 0;
  }

  :deep(.el-select__caret) {
    width: 16px;
    height: 16px;
    color: #334155;
  }

  :deep(.el-select__caret svg) {
    width: 14px;
    height: 14px;
  }

  :deep(.el-select),
  :deep(.el-input),
  :deep(.el-input-number) {
    width: 100%;
  }

  @media (max-width: 860px) {
    .deploy-dialog-body {
      max-height: 72vh;
      overflow-y: auto;
    }

    .deploy-dialog-layout,
    .runtime-grid {
      grid-template-columns: 1fr;
    }

    .deploy-main {
      max-height: none;
      overflow: visible;
    }
  }
</style>
