<template>
  <div class="platform-address-picker">
    <div class="platform-address-input">
      <ElInput
        :model-value="modelValue"
        clearable
        :disabled="disabled"
        :placeholder="resolvedPlaceholder"
        @update:model-value="handleInput"
      >
        <template #append>
          <ElButton :disabled="disabled || !projectId || !spec" @click="openDialog">
            选择资源
          </ElButton>
        </template>
      </ElInput>
    </div>

    <ElDialog
      v-model="visible"
      :title="spec?.title || '选择平台地址'"
      width="860px"
      append-to-body
      class="platform-address-dialog"
      :close-on-click-modal="false"
    >
      <div class="address-dialog-body">
        <div class="address-flow">
          <div
            v-for="(node, index) in flowNodes"
            :key="node.key"
            class="address-flow-node"
            :class="{ active: node.active, done: node.done }"
          >
            <span class="address-flow-index">{{ index + 1 }}</span>
            <span class="address-flow-label">{{ node.label }}</span>
          </div>
        </div>

        <ElForm label-width="112px" class="platform-address-form">
          <ElFormItem
            v-if="spec?.channelTypeSelectable"
            :label="spec.channelTypeLabel || '渠道类型'"
            required
          >
            <div class="select-action-row">
              <ElSelect
                v-model="channelTypeValue"
                filterable
                clearable
                popper-class="devops-platform-param-dropdown"
                :teleported="true"
                fit-input-width
                placeholder="请选择渠道类型"
                :loading="loading.channelType"
                loading-text="加载中..."
                :no-data-text="errors.channelType || '暂无可选渠道类型'"
                style="width: 100%"
                @visible-change="(opened: boolean) => opened && queryChannelTypes()"
                @change="handleChannelTypeChange"
              >
                <template #loading>
                  <div class="devops-select-loading">
                    <span class="devops-select-loading__icon"></span>
                    <span>加载中...</span>
                  </div>
                </template>
                <ElOption
                  v-for="option in channelTypeOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </ElSelect>
              <ElButton
                class="address-refresh-button"
                :icon="RefreshRight"
                :loading="loading.channelType"
                :disabled="disabled"
                @click="queryChannelTypes()"
              >
                刷新
              </ElButton>
            </div>
          </ElFormItem>

          <ElFormItem :label="spec?.channelLabel || '渠道实例'" required>
            <div class="select-action-row">
              <ElSelect
                v-model="channelBindingId"
                filterable
                allow-create
                default-first-option
                remote
                clearable
                remote-show-suffix
                popper-class="devops-platform-param-dropdown"
                :teleported="true"
                fit-input-width
                :disabled="false"
                :placeholder="
                  spec?.channelTypeSelectable && !channelTypeValue
                    ? '请先选择渠道类型'
                    : '请选择渠道实例'
                "
                :loading="loading.channel"
                loading-text="加载中..."
                :no-data-text="errors.channel || '暂无可选渠道实例'"
                style="width: 100%"
                :remote-method="queryChannels"
                @visible-change="(opened: boolean) => opened && queryChannels()"
                @change="handleChannelChange"
              >
                <template #loading>
                  <div class="devops-select-loading">
                    <span class="devops-select-loading__icon"></span>
                    <span>加载中...</span>
                  </div>
                </template>
                <ElOption
                  v-for="option in channelOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                >
                  <div class="address-option">
                    <span>{{ option.label }}</span>
                    <small v-if="optionSecondary(option)">{{ optionSecondary(option) }}</small>
                  </div>
                </ElOption>
              </ElSelect>
              <ElButton
                class="address-refresh-button"
                :icon="RefreshRight"
                :loading="loading.channel"
                :disabled="spec?.channelTypeSelectable && !channelTypeValue"
                @click="queryChannels()"
              >
                刷新
              </ElButton>
            </div>
          </ElFormItem>

          <ElFormItem v-if="spec?.projectParamType" :label="spec.projectLabel || '项目'" required>
            <div class="select-action-row">
              <ElSelect
                v-model="projectValue"
                filterable
                allow-create
                default-first-option
                remote
                clearable
                remote-show-suffix
                popper-class="devops-platform-param-dropdown"
                :teleported="true"
                fit-input-width
                :disabled="!channelBindingId"
                :placeholder="channelBindingId ? '请选择或输入项目' : '请先选择渠道实例'"
                :loading="loading.project"
                loading-text="加载中..."
                :no-data-text="errors.project || '暂无可选项目'"
                style="width: 100%"
                :remote-method="queryProjects"
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
                >
                  <div class="address-option">
                    <span>{{ option.label }}</span>
                    <small v-if="optionSecondary(option)">{{ optionSecondary(option) }}</small>
                  </div>
                </ElOption>
              </ElSelect>
              <ElButton
                class="address-refresh-button"
                :icon="RefreshRight"
                :loading="loading.project"
                :disabled="!channelBindingId"
                @click="queryProjects()"
              >
                刷新
              </ElButton>
            </div>
          </ElFormItem>

          <ElFormItem
            v-if="spec?.repositoryParamType"
            :label="spec.repositoryLabel || '仓库'"
            required
          >
            <div class="select-action-row">
              <ElSelect
                v-model="repositoryValue"
                filterable
                allow-create
                default-first-option
                remote
                clearable
                remote-show-suffix
                popper-class="devops-platform-param-dropdown"
                :teleported="true"
                fit-input-width
                :disabled="!channelBindingId"
                :placeholder="
                  channelBindingId
                    ? `请选择或输入${spec.repositoryLabel || '仓库'}`
                    : '请先选择渠道实例'
                "
                :loading="loading.repository"
                loading-text="加载中..."
                :no-data-text="errors.repository || '暂无可选资源'"
                style="width: 100%"
                :remote-method="queryRepositories"
                @visible-change="(opened: boolean) => opened && queryRepositories()"
                @change="handleRepositoryChange"
              >
                <template #loading>
                  <div class="devops-select-loading">
                    <span class="devops-select-loading__icon"></span>
                    <span>加载中...</span>
                  </div>
                </template>
                <ElOption
                  v-for="option in repositoryOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                >
                  <div class="address-option">
                    <span>{{ option.label }}</span>
                    <small v-if="optionSecondary(option)">{{ optionSecondary(option) }}</small>
                  </div>
                </ElOption>
              </ElSelect>
              <ElButton
                class="address-refresh-button"
                :icon="RefreshRight"
                :loading="loading.repository"
                :disabled="!channelBindingId"
                @click="queryRepositories()"
              >
                刷新
              </ElButton>
            </div>
          </ElFormItem>

          <ElFormItem v-if="spec?.imageParamType" :label="spec.imageLabel || '镜像'" required>
            <div class="select-action-row">
              <ElSelect
                v-model="imageValue"
                filterable
                allow-create
                default-first-option
                remote
                clearable
                remote-show-suffix
                popper-class="devops-platform-param-dropdown"
                :teleported="true"
                fit-input-width
                :disabled="!channelBindingId || !projectValue"
                :placeholder="projectValue ? '请选择或输入镜像' : '请先选择项目'"
                :loading="loading.image"
                loading-text="加载中..."
                :no-data-text="errors.image || '暂无可选镜像'"
                style="width: 100%"
                :remote-method="queryImages"
                @visible-change="(opened: boolean) => opened && queryImages()"
                @change="handleImageChange"
              >
                <template #loading>
                  <div class="devops-select-loading">
                    <span class="devops-select-loading__icon"></span>
                    <span>加载中...</span>
                  </div>
                </template>
                <ElOption
                  v-for="option in imageOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                >
                  <div class="address-option">
                    <span>{{ option.label }}</span>
                    <small v-if="optionSecondary(option)">{{ optionSecondary(option) }}</small>
                  </div>
                </ElOption>
              </ElSelect>
              <ElButton
                class="address-refresh-button"
                :icon="RefreshRight"
                :loading="loading.image"
                :disabled="!channelBindingId || !projectValue"
                @click="queryImages()"
              >
                刷新
              </ElButton>
            </div>
          </ElFormItem>

          <ElFormItem v-if="spec?.tagParamType" :label="spec.tagLabel || 'Tag'" required>
            <div class="select-action-row">
              <ElSelect
                v-model="tagValue"
                filterable
                allow-create
                default-first-option
                remote
                clearable
                remote-show-suffix
                popper-class="devops-platform-param-dropdown"
                :teleported="true"
                fit-input-width
                :disabled="!channelBindingId || !projectValue || !imageValue"
                :placeholder="imageValue ? '请选择或输入 Tag' : '请先选择镜像'"
                :loading="loading.tag"
                loading-text="加载中..."
                :no-data-text="errors.tag || '暂无可选 Tag'"
                style="width: 100%"
                :remote-method="queryTags"
                @visible-change="(opened: boolean) => opened && queryTags()"
                @change="handleTagChange"
              >
                <template #loading>
                  <div class="devops-select-loading">
                    <span class="devops-select-loading__icon"></span>
                    <span>加载中...</span>
                  </div>
                </template>
                <ElOption
                  v-for="option in tagOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                >
                  <div class="address-option">
                    <span>{{ option.label }}</span>
                    <small v-if="optionSecondary(option)">{{ optionSecondary(option) }}</small>
                  </div>
                </ElOption>
              </ElSelect>
              <ElButton
                class="address-refresh-button"
                :icon="RefreshRight"
                :loading="loading.tag"
                :disabled="!channelBindingId || !projectValue || !imageValue"
                @click="queryTags()"
              >
                刷新
              </ElButton>
            </div>
          </ElFormItem>

          <ElFormItem
            v-if="spec?.protocolOptions?.length"
            :label="spec.protocolLabel || '协议'"
            required
          >
            <ElSegmented
              v-model="protocolValue"
              :options="spec.protocolOptions"
              :disabled="!repositoryValue"
              @change="handleProtocolChange"
            />
          </ElFormItem>

          <ElFormItem
            v-if="spec?.outputFormatOptions?.length"
            :label="spec.outputFormatLabel || '输出格式'"
            required
          >
            <ElSegmented
              v-model="outputFormatValue"
              :options="spec.outputFormatOptions"
              :disabled="!repositoryValue"
              @change="handleOutputFormatChange"
            />
          </ElFormItem>

          <ElFormItem v-if="showCredentialSwitch" label="HTTP 凭证">
            <div class="credential-url-box">
              <ElSwitch
                v-model="includeHttpCredential"
                active-text="携带"
                inactive-text="不携带"
                :disabled="!channelBindingId"
                @change="handleCredentialSwitchChange"
              />
              <ElAlert
                v-if="includeHttpCredential"
                class="credential-url-warning"
                type="warning"
                :closable="false"
                title="凭证会写入仓库地址原文"
                description="可能出现在流水线配置、Jenkinsfile 和构建日志中，请确认该仓库确实需要 HTTP 凭证 URL。"
              />
            </div>
          </ElFormItem>

          <ElFormItem v-if="spec?.tagLabel && !spec?.tagParamType" :label="spec.tagLabel" required>
            <ElInput
              v-model="tagValue"
              :disabled="false"
              clearable
              placeholder="请输入 Tag"
              @input="handleTagChange"
            />
          </ElFormItem>
        </ElForm>

        <div class="address-preview">
          <div class="address-preview-label">{{ spec?.resultLabel || '最终地址' }}</div>
          <ElInput :model-value="previewValue" readonly placeholder="选择完成后自动生成" />
        </div>
      </div>

      <template #footer>
        <div class="platform-address-footer">
          <ElButton @click="visible = false">取消</ElButton>
          <ElButton type="primary" :disabled="!previewValue" @click="confirmValue">
            写入地址
          </ElButton>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, reactive, ref } from 'vue'
  import { RefreshRight } from '@element-plus/icons-vue'
  import { listDevopsChannelTypeApi } from '@/api/devops/channel'
  import { listDevopsDynamicParamOptionsApi, type DynamicParamOption } from '@/api/devops/pipeline'
  import { platformAddressSpecs } from './defs'

  type SelectOption = {
    label: string
    value: string
    metadata?: Record<string, any>
  }

  type FlowNode = {
    key: string
    label: string
    done: boolean
    active?: boolean
  }

  const props = withDefaults(
    defineProps<{
      modelValue?: string
      paramType: string
      projectId?: string
      systemId?: string
      environmentId?: string
      channelGroupCode?: string
      channelType?: string
      mappingField?: string
      disabled?: boolean
      placeholder?: string
    }>(),
    {
      modelValue: '',
      projectId: '',
      systemId: '',
      environmentId: '',
      channelGroupCode: '',
      channelType: '',
      mappingField: '',
      disabled: false,
      placeholder: ''
    }
  )

  const emit = defineEmits<{
    'update:modelValue': [value: string]
    change: [value: string]
  }>()

  const visible = ref(false)
  const channelTypeValue = ref('')
  const channelBindingId = ref('')
  const projectValue = ref('')
  const repositoryValue = ref('')
  const imageValue = ref('')
  const tagValue = ref('')
  const protocolValue = ref('http')
  const outputFormatValue = ref('json')
  const includeHttpCredential = ref(false)
  const channelTypeOptions = ref<SelectOption[]>([])
  const channelOptions = ref<SelectOption[]>([])
  const projectOptions = ref<SelectOption[]>([])
  const repositoryOptions = ref<SelectOption[]>([])
  const imageOptions = ref<SelectOption[]>([])
  const tagOptions = ref<SelectOption[]>([])
  const loading = reactive({
    channelType: false,
    channel: false,
    project: false,
    repository: false,
    image: false,
    tag: false
  })
  const errors = reactive({
    channelType: '',
    channel: '',
    project: '',
    repository: '',
    image: '',
    tag: ''
  })
  const spec = computed(() => platformAddressSpecs[props.paramType])
  const resolvedPlaceholder = computed(
    () => props.placeholder || spec.value?.placeholder || '请输入或选择平台地址'
  )
  const selectedChannel = computed(() =>
    channelOptions.value.find((item) => item.value === channelBindingId.value)
  )
  const selectedRepository = computed(() =>
    repositoryOptions.value.find((item) => item.value === repositoryValue.value)
  )
  const selectedImage = computed(() =>
    imageOptions.value.find((item) => item.value === imageValue.value)
  )
  const isSelectionComplete = computed(() => {
    if (!spec.value || !channelBindingId.value) return false
    if (spec.value.projectParamType && !projectValue.value) return false
    if (spec.value.repositoryParamType && !repositoryValue.value) return false
    if (spec.value.imageParamType && !imageValue.value) return false
    if (spec.value.tagLabel && !tagValue.value.trim()) return false
    return true
  })
  const flowNodes = computed(() => {
    const nodes: FlowNode[] = []
    if (spec.value?.channelTypeSelectable) {
      nodes.push({
        key: 'channelType',
        label: spec.value.channelTypeLabel || '渠道类型',
        done: !!channelTypeValue.value
      })
    }
    nodes.push({
      key: 'channel',
      label: spec.value?.channelLabel || '渠道实例',
      done: !!channelBindingId.value
    })
    if (spec.value?.projectParamType) {
      nodes.push({
        key: 'project',
        label: spec.value.projectLabel || '项目',
        done: !!projectValue.value
      })
    }
    if (spec.value?.repositoryParamType) {
      nodes.push({
        key: 'repository',
        label: spec.value.repositoryLabel || '仓库',
        done: !!repositoryValue.value
      })
    }
    if (spec.value?.protocolOptions?.length) {
      nodes.push({
        key: 'protocol',
        label: spec.value.protocolLabel || '协议',
        done: !!protocolValue.value
      })
    }
    if (spec.value?.outputFormatOptions?.length) {
      nodes.push({
        key: 'outputFormat',
        label: spec.value.outputFormatLabel || '输出格式',
        done: !!outputFormatValue.value
      })
    }
    if (spec.value?.imageParamType) {
      nodes.push({
        key: 'image',
        label: spec.value.imageLabel || '镜像',
        done: !!imageValue.value
      })
    }
    if (spec.value?.tagLabel) {
      nodes.push({
        key: 'tag',
        label: spec.value.tagLabel,
        done: !!tagValue.value.trim()
      })
    }
    return nodes.map((node) => ({
      ...node,
      active: !node.done && nodes.findIndex((item) => !item.done) === nodes.indexOf(node)
    }))
  })

  const previewValue = computed(() => {
    if (!spec.value || !isSelectionComplete.value) return ''
    const endpoint = String(
      selectedChannel.value?.metadata?.endpoint || channelBindingId.value
    ).trim()
    if (!endpoint) return ''
    if (props.paramType === 'gitRepositoryUrl') {
      if (protocolValue.value === 'ssh') {
        const sshURL = String(selectedRepository.value?.metadata?.sshUrlToRepo || '').trim()
        return sshURL || gitSshUrl(endpoint, repositoryValue.value)
      }
      const httpURL = String(selectedRepository.value?.metadata?.httpUrlToRepo || '').trim()
      return httpURL || joinUrl(endpoint, repositoryValue.value, { suffix: '.git' })
    }
    if (isArtifactRepositoryUrlParam()) {
      return selectedRepository.value?.metadata?.url
        ? String(selectedRepository.value.metadata.url)
        : joinUrl(endpoint, ...artifactRepositoryFallbackPath(), repositoryValue.value, {
            trailingSlash: spec.value.trailingSlash
          })
    }
    if (props.paramType === 'harborProjectUrl') {
      return joinUrl(endpoint, projectValue.value)
    }
    if (props.paramType === 'harborImageUrl' || props.paramType === 'harborImageTagUrl') {
      const fullName = String(selectedImage.value?.metadata?.fullName || '').trim()
      const imageUrl = joinUrl(endpoint, fullName || joinUrl(projectValue.value, imageValue.value))
      return props.paramType === 'harborImageTagUrl' && tagValue.value.trim()
        ? `${imageUrl}:${tagValue.value.trim()}`
        : imageUrl
    }
    if (props.paramType === 'registryRepositoryUrl') {
      return joinUrl(endpoint, projectValue.value)
    }
    if (props.paramType === 'registryImageUrl' || props.paramType === 'registryImageTagUrl') {
      const fullName = String(selectedImage.value?.metadata?.fullName || '').trim()
      const imageUrl = joinUrl(endpoint, fullName || joinUrl(projectValue.value, imageValue.value))
      return props.paramType === 'registryImageTagUrl' && tagValue.value.trim()
        ? `${imageUrl}:${tagValue.value.trim()}`
        : imageUrl
    }
    if (props.paramType === 'nexusArtifactUrl' || props.paramType === 'jfrogArtifactUrl') {
      return selectedImage.value?.metadata?.url
        ? String(selectedImage.value.metadata.url)
        : joinUrl(
            endpoint,
            ...artifactRepositoryFallbackPath(),
            projectValue.value,
            imageValue.value,
            {
              trailingSlash: spec.value.trailingSlash
            }
          )
    }
    if (
      props.paramType === 'nexusArtifactVersionUrl' ||
      props.paramType === 'jfrogArtifactVersionUrl'
    ) {
      const artifactUrl = selectedImage.value?.metadata?.url
        ? String(selectedImage.value.metadata.url)
        : joinUrl(
            endpoint,
            ...artifactRepositoryFallbackPath(),
            projectValue.value,
            imageValue.value,
            {
              trailingSlash: spec.value.trailingSlash
            }
          )
      return tagValue.value.trim() ? `${artifactUrl}:${tagValue.value.trim()}` : artifactUrl
    }
    if (props.paramType === 'sonarProjectUrl' || props.paramType === 'sonarProjectKeyUrl') {
      return joinUrl(endpoint, `dashboard?id=${encodeURIComponent(projectValue.value)}`)
    }
    if (props.paramType === 'hostConfig' || props.paramType === 'hostGroupConfig') {
      const metadata = selectedRepository.value?.metadata || {}
      return String(
        metadata[outputFormatValue.value] || selectedRepository.value?.value || ''
      ).trim()
    }
    return ''
  })

  function handleInput(value: string) {
    emit('update:modelValue', value)
    emit('change', value)
  }

  function openDialog() {
    if (!spec.value || props.disabled) return
    visible.value = true
    protocolValue.value = spec.value.protocolOptions?.[0]?.value || 'http'
    outputFormatValue.value = spec.value.outputFormatOptions?.[0]?.value || 'json'
    if (spec.value.channelTypeSelectable) {
      channelTypeValue.value = ''
      handleChannelTypeChange()
      queryChannelTypes().catch(() => undefined)
    } else {
      queryChannels().catch(() => undefined)
    }
  }

  async function queryOptions(paramType: string, keyword = '', extra?: Record<string, string>) {
    if (!props.projectId) return []
    const res = await listDevopsDynamicParamOptionsApi({
      projectId: props.projectId,
      systemId: props.systemId,
      environmentId: props.environmentId,
      paramType,
      keyword,
      page: 1,
      pageSize: 50,
      ...extra
    })
    return (res.items || []).map(mapOption)
  }

  async function queryChannels(keyword = '') {
    if (!spec.value || loading.channel) return
    if (spec.value.channelTypeSelectable && !channelTypeValue.value) return
    loading.channel = true
    errors.channel = ''
    try {
      const groupCode = props.channelGroupCode || spec.value.channelGroupCode || ''
      const channelType = spec.value.channelTypeSelectable
        ? channelTypeValue.value
        : props.channelType || spec.value.channelType || ''
      channelOptions.value = await queryOptions(
        groupCode ? 'channelGroupCode' : spec.value.channelParamType,
        keyword,
        {
          configTypeCode: groupCode,
          componentValue: channelType,
          mappingField: 'endpoint'
        }
      )
    } catch (error) {
      channelOptions.value = []
      errors.channel = formatLoadError(error)
    } finally {
      loading.channel = false
    }
  }

  async function queryChannelTypes() {
    if (!spec.value?.channelTypeSelectable || loading.channelType) return
    loading.channelType = true
    errors.channelType = ''
    try {
      const res = await listDevopsChannelTypeApi({
        page: 1,
        pageSize: 100,
        groupCode: props.channelGroupCode || spec.value.channelGroupCode || '',
        status: 1
      })
      const allowed = new Set(spec.value.allowedChannelTypes || [])
      channelTypeOptions.value = (res.items || [])
        .filter((item) => allowed.size === 0 || allowed.has(item.code))
        .map((item) => ({
          label: `${item.name}（${item.code}）`,
          value: item.code
        }))
      if (channelTypeValue.value) {
        queryChannels().catch(() => undefined)
      }
    } catch (error) {
      channelTypeOptions.value = []
      errors.channelType = formatLoadError(error)
    } finally {
      loading.channelType = false
    }
  }

  async function queryProjects(keyword = '') {
    if (!spec.value?.projectParamType || !channelBindingId.value || loading.project) {
      if (!channelBindingId.value) {
        projectOptions.value = []
        errors.project = ''
      }
      return
    }
    loading.project = true
    errors.project = ''
    try {
      projectOptions.value = await queryOptions(resolveProjectParamType(), keyword, {
        channelBindingId: channelBindingId.value
      })
    } catch (error) {
      projectOptions.value = []
      errors.project = formatLoadError(error)
    } finally {
      loading.project = false
    }
  }

  async function queryRepositories(keyword = '') {
    if (!spec.value?.repositoryParamType || !channelBindingId.value || loading.repository) {
      if (!channelBindingId.value) {
        repositoryOptions.value = []
        errors.repository = ''
      }
      return
    }
    loading.repository = true
    errors.repository = ''
    try {
      const extra: Record<string, string> = {
        channelBindingId: channelBindingId.value
      }
      if (props.paramType === 'gitRepositoryUrl' && includeHttpCredential.value) {
        extra.componentValue = 'includeHttpCredential'
      }
      repositoryOptions.value = await queryOptions(resolveRepositoryParamType(), keyword, extra)
    } catch (error) {
      repositoryOptions.value = []
      errors.repository = formatLoadError(error)
    } finally {
      loading.repository = false
    }
  }

  async function queryImages(keyword = '') {
    if (
      !spec.value?.imageParamType ||
      !channelBindingId.value ||
      !projectValue.value ||
      loading.image
    ) {
      if (!channelBindingId.value || !projectValue.value) {
        imageOptions.value = []
        errors.image = ''
      }
      return
    }
    loading.image = true
    errors.image = ''
    try {
      imageOptions.value = await queryOptions(resolveImageParamType(), keyword, {
        channelBindingId: channelBindingId.value,
        projectValue: projectValue.value
      })
    } catch (error) {
      imageOptions.value = []
      errors.image = formatLoadError(error)
    } finally {
      loading.image = false
    }
  }

  async function queryTags(keyword = '') {
    if (
      !spec.value?.tagParamType ||
      !channelBindingId.value ||
      !projectValue.value ||
      !imageValue.value ||
      loading.tag
    ) {
      if (!channelBindingId.value || !projectValue.value || !imageValue.value) {
        tagOptions.value = []
        errors.tag = ''
      }
      return
    }
    loading.tag = true
    errors.tag = ''
    try {
      tagOptions.value = await queryOptions(resolveTagParamType(), keyword, {
        channelBindingId: channelBindingId.value,
        projectValue: projectValue.value,
        componentValue: imageValue.value
      })
    } catch (error) {
      tagOptions.value = []
      errors.tag = formatLoadError(error)
    } finally {
      loading.tag = false
    }
  }

  function handleChannelChange() {
    projectValue.value = ''
    repositoryValue.value = ''
    imageValue.value = ''
    tagValue.value = ''
    projectOptions.value = []
    repositoryOptions.value = []
    imageOptions.value = []
    tagOptions.value = []
    errors.channel = ''
    errors.project = ''
    errors.repository = ''
    errors.image = ''
    errors.tag = ''
    if (spec.value?.projectParamType) queryProjects().catch(() => undefined)
    if (spec.value?.repositoryParamType) queryRepositories().catch(() => undefined)
  }

  function handleChannelTypeChange() {
    channelBindingId.value = ''
    projectValue.value = ''
    repositoryValue.value = ''
    imageValue.value = ''
    tagValue.value = ''
    channelOptions.value = []
    projectOptions.value = []
    repositoryOptions.value = []
    imageOptions.value = []
    tagOptions.value = []
    errors.channelType = ''
    errors.channel = ''
    errors.project = ''
    errors.repository = ''
    errors.image = ''
    errors.tag = ''
    if (channelTypeValue.value) queryChannels().catch(() => undefined)
  }

  function handleCredentialSwitchChange() {
    repositoryValue.value = ''
    repositoryOptions.value = []
    errors.repository = ''
    if (channelBindingId.value && spec.value?.repositoryParamType) {
      queryRepositories().catch(() => undefined)
    }
  }

  function handleProjectChange() {
    imageValue.value = ''
    tagValue.value = ''
    imageOptions.value = []
    tagOptions.value = []
    errors.image = ''
    errors.tag = ''
    if (spec.value?.imageParamType) queryImages().catch(() => undefined)
  }

  function handleRepositoryChange() {
    errors.repository = ''
  }

  function handleProtocolChange() {
    errors.repository = ''
  }

  function handleOutputFormatChange() {
    errors.repository = ''
  }

  function handleImageChange() {
    errors.image = ''
    tagValue.value = ''
    tagOptions.value = []
    errors.tag = ''
    if (spec.value?.tagParamType) queryTags().catch(() => undefined)
  }

  function handleTagChange() {
    errors.tag = ''
  }

  const showCredentialSwitch = computed(() => props.paramType === 'gitRepositoryUrl')

  function confirmValue() {
    if (!previewValue.value) return
    handleInput(previewValue.value)
    visible.value = false
  }

  function mapOption(option: DynamicParamOption): SelectOption {
    return {
      label: option.label,
      value: option.value,
      metadata: parseMetadata(option.metadata)
    }
  }

  function optionSecondary(option: SelectOption) {
    const metadata = option.metadata || {}
    const value = String(
      metadata.endpoint ||
        metadata.url ||
        metadata.httpUrlToRepo ||
        metadata.sshUrlToRepo ||
        metadata.fullName ||
        metadata.image ||
        metadata.host ||
        metadata.namespace ||
        metadata.clusterUuid ||
        (metadata.count ? `${metadata.count} 台` : '') ||
        ''
    ).trim()
    if (value) return value
    const rawValue = String(option.value || '').trim()
    return rawValue && rawValue !== option.label && rawValue.includes('://') ? rawValue : ''
  }

  function selectedImageRegistryType() {
    if ((props.channelGroupCode || spec.value?.channelGroupCode) !== 'image_registry') return ''
    if (spec.value?.channelTypeSelectable) return channelTypeValue.value
    return props.channelType || spec.value?.channelType || ''
  }

  function selectedArtifactRepositoryType() {
    if ((props.channelGroupCode || spec.value?.channelGroupCode) !== 'artifact_repo') return ''
    if (spec.value?.channelTypeSelectable) return channelTypeValue.value
    return props.channelType || spec.value?.channelType || ''
  }

  function isArtifactRepositoryUrlParam() {
    return props.paramType === 'nexusRepositoryUrl' || props.paramType === 'jfrogRepositoryUrl'
  }

  function artifactRepositoryFallbackPath() {
    const channelType = selectedArtifactRepositoryType()
    if (channelType === 'jfrog') return ['artifactory']
    if (channelType === 'nexus') return ['repository']
    return spec.value?.fallbackPath || []
  }

  function resolveRepositoryParamType() {
    const channelType = selectedArtifactRepositoryType()
    if (!channelType) return spec.value?.repositoryParamType || ''
    return channelType === 'jfrog' ? 'jfrogRepository' : 'nexusRepository'
  }

  function resolveProjectParamType() {
    const imageRegistryType = selectedImageRegistryType()
    if (imageRegistryType) {
      return imageRegistryType === 'harbor' ? 'harborProject' : 'registryRepository'
    }
    const artifactRepositoryType = selectedArtifactRepositoryType()
    if (artifactRepositoryType) {
      return artifactRepositoryType === 'jfrog' ? 'jfrogRepository' : 'nexusRepository'
    }
    return spec.value?.projectParamType || spec.value?.repositoryParamType || ''
  }

  function resolveImageParamType() {
    const imageRegistryType = selectedImageRegistryType()
    if (imageRegistryType) {
      return imageRegistryType === 'harbor' ? 'harborImage' : 'registryImage'
    }
    const artifactRepositoryType = selectedArtifactRepositoryType()
    if (artifactRepositoryType) {
      return artifactRepositoryType === 'jfrog' ? 'jfrogArtifactName' : 'nexusArtifactName'
    }
    return spec.value?.imageParamType || ''
  }

  function resolveTagParamType() {
    const imageRegistryType = selectedImageRegistryType()
    if (imageRegistryType) {
      return imageRegistryType === 'harbor' ? 'harborImageTag' : 'registryImageTag'
    }
    const artifactRepositoryType = selectedArtifactRepositoryType()
    if (artifactRepositoryType) {
      return artifactRepositoryType === 'jfrog' ? 'jfrogArtifactVersion' : 'nexusArtifactVersion'
    }
    return spec.value?.tagParamType || ''
  }

  function parseMetadata(raw?: string) {
    if (!raw) return undefined
    try {
      return JSON.parse(raw)
    } catch {
      return undefined
    }
  }

  function formatLoadError(error: unknown) {
    if (error instanceof Error && error.message) return error.message
    if (typeof error === 'object' && error && 'message' in error) {
      return String((error as { message?: unknown }).message || '加载失败，请稍后重试')
    }
    return '加载失败，请稍后重试'
  }

  function joinUrl(
    base: string,
    ...partsAndOptions: Array<string | { trailingSlash?: boolean; suffix?: string }>
  ) {
    const options =
      typeof partsAndOptions[partsAndOptions.length - 1] === 'object'
        ? (partsAndOptions.pop() as { trailingSlash?: boolean; suffix?: string })
        : {}
    const cleanBase = String(base || '').replace(/\/+$/, '')
    const paths = partsAndOptions
      .map((item) =>
        String(item || '')
          .trim()
          .replace(/^\/+|\/+$/g, '')
      )
      .filter(Boolean)
    let value = [cleanBase, ...paths].filter(Boolean).join('/')
    if (options.suffix && value && !value.endsWith(options.suffix)) value += options.suffix
    return options.trailingSlash && value ? `${value}/` : value
  }

  function gitSshUrl(endpoint: string, repositoryPath: string) {
    const host = endpointHost(endpoint)
    const path = String(repositoryPath || '')
      .trim()
      .replace(/^\/+|\/+$/g, '')
      .replace(/\.git$/, '')
    if (!host || !path) return ''
    return `git@${host}:${path}.git`
  }

  function endpointHost(endpoint: string) {
    const value = String(endpoint || '').trim()
    if (!value) return ''
    try {
      const normalized = value.includes('://') ? value : `https://${value}`
      return new URL(normalized).hostname
    } catch {
      return ''
    }
  }
</script>

<style scoped>
  .platform-address-picker {
    width: 100%;
  }

  .platform-address-input {
    width: 100%;

    :deep(.el-input__wrapper) {
      background: var(--el-bg-color);
      box-shadow: 0 0 0 1px var(--el-border-color) inset;
    }

    :deep(.el-input__inner) {
      font-weight: 500;
      color: var(--el-text-color-primary);
      -webkit-text-fill-color: var(--el-text-color-primary);
    }

    :deep(.el-input__inner::placeholder) {
      font-weight: 400;
      color: var(--el-text-color-placeholder);
      -webkit-text-fill-color: var(--el-text-color-placeholder);
    }

    :deep(.el-input.is-disabled .el-input__inner) {
      color: var(--el-text-color-disabled);
      -webkit-text-fill-color: var(--el-text-color-disabled);
    }

    :deep(.el-input-group__append) {
      background: var(--el-fill-color-light);
      box-shadow: 0 0 0 1px var(--el-border-color) inset;
    }

    :deep(.el-input-group__append .el-button) {
      font-weight: 500;
      color: var(--el-text-color-regular);
    }
  }

  .address-dialog-body {
    max-height: calc(100vh - 230px);
    padding: 12px;
    overflow: auto;
    background: var(--el-fill-color-extra-light);
    border-radius: 0 0 8px 8px;
  }

  :global(.platform-address-dialog .el-dialog__body) {
    max-height: calc(100vh - 210px);
    overflow: hidden;
  }

  .address-flow {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(128px, 1fr));
    gap: 4px;
    padding: 8px;
    margin-bottom: 14px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .address-flow-node {
    position: relative;
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    height: 34px;
    padding: 0 10px;
    color: var(--el-text-color-secondary);
    background: transparent;
    border: 0;
    border-radius: 6px;
  }

  .address-flow-node.done {
    color: var(--el-text-color-regular);
    background: transparent;
  }

  .address-flow-node.active {
    color: var(--el-color-primary);
    background: var(--el-fill-color-lighter);
  }

  .address-flow-node.active::before {
    position: absolute;
    bottom: 3px;
    left: 10px;
    right: 10px;
    height: 2px;
    content: '';
    background: var(--el-color-primary);
    border-radius: 999px;
  }

  .address-flow-index {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex: 0 0 22px;
    width: 22px;
    height: 22px;
    font-size: 12px;
    font-weight: 700;
    line-height: 1;
    color: inherit;
    background: transparent;
    border: 1px solid currentcolor;
    border-radius: 50%;
  }

  .address-flow-label {
    overflow: hidden;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .platform-address-form {
    padding: 0 4px;
  }

  .select-action-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 82px;
    gap: 8px;
    align-items: center;
    width: 100%;

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

    :deep(.el-select__selected-item),
    :deep(.el-select__selected-item span),
    :deep(.el-select__input) {
      font-weight: 500;
      color: var(--el-text-color-primary);
      -webkit-text-fill-color: var(--el-text-color-primary);
    }

    :deep(.el-select__placeholder) {
      font-weight: 400;
      color: var(--el-text-color-placeholder);
      -webkit-text-fill-color: var(--el-text-color-placeholder);
    }

    :deep(.el-select__placeholder.is-transparent) {
      font-weight: 500;
      color: var(--el-text-color-primary);
      -webkit-text-fill-color: var(--el-text-color-primary);
      opacity: 1;
    }

    :deep(.el-select.is-disabled .el-select__selected-item),
    :deep(.el-select.is-disabled .el-select__selected-item span),
    :deep(.el-select.is-disabled .el-select__input),
    :deep(.el-select.is-disabled .el-select__placeholder.is-transparent) {
      color: var(--el-text-color-disabled);
      -webkit-text-fill-color: var(--el-text-color-disabled);
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
  }

  .address-refresh-button {
    width: 82px;
    height: 34px;
    padding: 0 12px;
  }

  .address-option {
    display: flex;
    flex-direction: column;
    justify-content: center;
    min-height: 52px;
    overflow: visible;
  }

  .address-option span,
  .address-option small {
    overflow: visible;
    line-height: 1.35;
    text-overflow: clip;
    white-space: normal;
    word-break: break-word;
  }

  .address-option small {
    color: var(--el-text-color-secondary);
  }

  :global(.el-popper.devops-platform-param-dropdown) {
    --devops-platform-dropdown-height: 260px;

    max-width: min(620px, calc(100vw - 32px));
    padding: 0;
    background: #ffffff !important;
    border: 1px solid #e2e8f0;
    border-radius: 14px;
    box-shadow: 0 16px 34px rgb(15 23 42 / 12%);
    overflow: hidden !important;

    .el-select-dropdown,
    .el-scrollbar,
    .el-scrollbar__view {
      background: #ffffff !important;
    }

    .el-scrollbar,
    .el-scrollbar__view {
      width: 100% !important;
    }

    .el-select-dropdown {
      overflow: hidden !important;
      transform: none !important;
    }

    .el-scrollbar {
      overflow: hidden !important;
    }

    .el-select-dropdown__wrap {
      max-height: var(--devops-platform-dropdown-height) !important;
      background: #ffffff !important;
      overflow-y: auto !important;
    }

    .el-select-dropdown__list {
      width: 100% !important;
      padding: 0 !important;
      margin: 0 !important;
      background: #ffffff !important;
      box-sizing: border-box;
      text-align: left !important;
      transform: none !important;
    }

    .el-select-dropdown__item {
      display: flex !important;
      align-items: center !important;
      justify-content: flex-start !important;
      width: 100% !important;
      height: auto !important;
      min-height: 44px;
      padding: 10px 12px;
      overflow: visible !important;
      line-height: 1.35 !important;
      color: #334155 !important;
      border-radius: 0 !important;
      background: #ffffff !important;
      margin-bottom: 0 !important;
      transform: none !important;
      text-align: left !important;
      white-space: normal !important;
      writing-mode: horizontal-tb !important;
    }

    .el-select-dropdown__item.is-hovering {
      color: #1d4ed8 !important;
      background: #eff6ff !important;
      box-shadow: inset 4px 0 0 #3b82f6;
    }

    .el-select-dropdown__item.is-selected {
      font-weight: 600;
      color: #0f172a !important;
      background: #ffffff !important;
    }

    .el-select-dropdown__empty {
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 18px 12px;
      color: var(--el-text-color-secondary);
    }
  }

  :global(.el-popper.devops-platform-param-dropdown[aria-hidden='true']) {
    display: none !important;
  }

  :global(.el-popper.devops-platform-param-dropdown .el-select-dropdown__loading) {
    min-height: var(--devops-platform-dropdown-height, 260px);
    padding: 0 !important;
    border-left: 0 !important;
    box-shadow: none !important;
  }

  :global(.el-popper.devops-platform-param-dropdown .devops-select-loading) {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: var(--devops-platform-dropdown-height, 260px);
    gap: 8px;
    border-left: 0 !important;
    box-shadow: none !important;
    color: var(--el-color-primary);
    font-size: 13px;
    line-height: 1;
  }

  :global(.el-popper.devops-platform-param-dropdown .devops-select-loading__icon) {
    width: 24px;
    height: 24px;
    border: 2px solid var(--el-color-primary-light-7);
    border-top-color: var(--el-color-primary);
    border-radius: 50%;
    animation: devops-select-loading-rotate 0.9s linear infinite;
  }

  :global(
    .el-popper.devops-platform-param-dropdown
      .el-select-dropdown:has(.el-select-dropdown__loading)
      .el-scrollbar
  ) {
    display: none !important;
  }

  :global(
    .el-popper.devops-platform-param-dropdown
      .el-select-dropdown:has(.el-select-dropdown__loading)
      .el-select-dropdown__item
  ) {
    border-left: 0 !important;
    box-shadow: none !important;
  }

  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown
  ) {
    padding: 0 !important;
    margin: 0 !important;
    border: 0 !important;
    box-shadow: none !important;
  }

  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__wrap
  ) {
    max-height: var(--devops-platform-dropdown-height) !important;
    overflow-y: auto !important;
  }

  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__list
  ) {
    padding: 0 !important;
    margin: 0 !important;
    background: #ffffff !important;
    transform: none !important;
  }

  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__list
      .el-select-dropdown__item
  ) {
    display: flex !important;
    align-items: center !important;
    justify-content: flex-start !important;
    width: 100% !important;
    height: auto !important;
    min-height: 44px !important;
    padding: 10px 12px !important;
    overflow: visible !important;
    line-height: 1.35 !important;
    color: #334155 !important;
    text-overflow: clip !important;
    transform: none !important;
    white-space: normal !important;
    background: #ffffff !important;
    border-radius: 0 !important;
    margin-bottom: 0 !important;
    text-align: left !important;
    writing-mode: horizontal-tb !important;
  }

  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__list
      .el-select-dropdown__item.is-hovering
  ),
  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__list
      .el-select-dropdown__item:hover
  ) {
    color: #1d4ed8 !important;
    background: #eff6ff !important;
    box-shadow: inset 4px 0 0 #3b82f6;
  }

  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__list
      .el-select-dropdown__item.is-selected
  ) {
    font-weight: 600 !important;
    color: #0f172a !important;
    background: #ffffff !important;
    margin-bottom: 0 !important;
  }

  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__list
      .el-select-dropdown__item.is-selected.is-hovering
  ),
  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__list
      .el-select-dropdown__item.is-selected:hover
  ) {
    color: #1d4ed8 !important;
    background: #eff6ff !important;
    box-shadow: inset 4px 0 0 #3b82f6;
  }

  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__item:hover
      ~ .is-selected
  ),
  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__item.is-selected:has(~ .el-select-dropdown__item:hover)
  ) {
    background: #ffffff !important;
  }

  @keyframes devops-select-loading-rotate {
    to {
      transform: rotate(360deg);
    }
  }

  .credential-url-box {
    display: flex;
    flex-direction: column;
    gap: 10px;
    width: 100%;
  }

  .credential-url-warning {
    line-height: 1.5;
  }

  .address-preview {
    padding: 14px;
    margin-top: 4px;
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .address-preview-label {
    margin-bottom: 8px;
    font-size: 13px;
    font-weight: 700;
    color: var(--el-text-color-regular);
  }

  .platform-address-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  @media (width <= 720px) {
    .select-action-row {
      grid-template-columns: 1fr;
    }

    .address-refresh-button {
      width: 100%;
    }
  }
</style>
