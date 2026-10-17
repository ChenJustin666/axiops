<template>
  <div class="devops-project-channel-page art-full-height">
    <div class="page-toolbar">
      <div class="toolbar-title">
        <h2>渠道绑定</h2>
      </div>
      <div class="summary-list">
        <div class="summary-item">
          <span>分组</span>
          <strong>{{ visibleGroups.length }}</strong>
        </div>
        <div class="summary-item">
          <span>已绑定</span>
          <strong>{{ bindings.length }}</strong>
        </div>
        <div class="summary-item">
          <span>默认渠道</span>
          <strong>{{ defaultBindingCount }}</strong>
        </div>
      </div>
      <div class="toolbar-actions">
        <ElButton :icon="RefreshRight" :loading="loading" @click="refreshAll">刷新</ElButton>
        <ElButton
          type="primary"
          :icon="Link"
          :disabled="!selectedProjectId"
          @click="openAddDialog()"
        >
          新增
        </ElButton>
      </div>
    </div>

    <div v-loading="loading" class="binding-scroll">
      <ElEmpty v-if="!selectedProjectId" description="请选择项目" />
      <ElEmpty v-else-if="visibleGroups.length === 0" description="当前项目暂无绑定渠道">
        <ElButton type="primary" plain @click="openAddDialog()">新增</ElButton>
      </ElEmpty>
      <div v-else class="group-section-list">
        <section v-for="group in visibleGroups" :key="group.id" class="group-section">
          <div class="group-section-head">
            <div class="group-title-wrap">
              <div class="group-icon" :style="{ color: group.iconColor || '#2563eb' }">
                <ArtSvgIcon :icon="group.icon || 'ri:folder-3-line'" />
              </div>
              <div class="group-title">
                <div class="group-name-row">
                  <strong :title="group.name">{{ group.name }}</strong>
                  <ElTag size="small" type="info">{{ group.code }}</ElTag>
                </div>
                <span :title="group.description || '暂无描述'">
                  {{ group.description || '暂无描述' }}
                </span>
              </div>
            </div>
            <div class="group-actions">
              <ElTag size="small" type="success"
                >{{ groupBindings(group.code).length }} 个实例</ElTag
              >
              <ElButton
                size="small"
                type="primary"
                plain
                :disabled="availableChannelCount(group.code) === 0"
                @click="openAddDialog(group.code)"
              >
                新增
              </ElButton>
            </div>
          </div>

          <div v-if="groupBindings(group.code).length > 0" class="binding-card-grid">
            <ElCard
              v-for="item in groupBindings(group.code)"
              :key="item.id"
              class="binding-card"
              shadow="hover"
              :body-style="{ padding: '0' }"
            >
              <div class="binding-card-head">
                <div class="binding-main">
                  <strong :title="bindingDisplayName(item)">{{ bindingDisplayName(item) }}</strong>
                  <span :title="bindingDisplayCode(item)">{{ bindingDisplayCode(item) }}</span>
                </div>
                <ElTag v-if="item.isDefault" size="small" type="success">默认</ElTag>
              </div>
              <div class="binding-card-body">
                <div class="binding-meta">
                  <span>类型</span>
                  <strong>{{ channelTypeLabel(item.channelType, item.channelGroupCode) }}</strong>
                </div>
                <div class="binding-meta">
                  <span>凭据策略</span>
                  <strong :title="bindingCredentialLabel(item)">
                    {{ bindingCredentialLabel(item) }}
                  </strong>
                </div>
                <div class="binding-meta">
                  <span>状态</span>
                  <ElTag size="small" :type="item.status === 1 ? 'success' : 'info'">
                    {{ item.status === 1 ? '启用' : '停用' }}
                  </ElTag>
                </div>
                <div class="binding-meta">
                  <span>更新</span>
                  <strong>{{ formatTimestamp(item.updatedAt) }}</strong>
                </div>
              </div>
              <div class="binding-card-footer">
                <ElButton text type="primary" @click="openEditDialog(item)">编辑</ElButton>
                <ElButton text type="danger" @click="handleDelete(item)">解绑</ElButton>
              </div>
            </ElCard>
          </div>
        </section>
      </div>
    </div>

    <ElDialog
      v-model="dialogVisible"
      :title="dialogMode === 'add' ? '新增渠道绑定' : '编辑渠道绑定'"
      width="720px"
      align-center
      :close-on-click-modal="false"
      @close="resetForm"
    >
      <ElForm
        ref="formRef"
        v-loading="dialogLoading"
        element-loading-text="加载中..."
        :model="formData"
        :rules="rules"
        label-width="132px"
        :disabled="submitLoading || dialogLoading"
      >
        <div class="dialog-context">
          <span>绑定项目</span>
          <strong :title="selectedProjectLabel">{{ selectedProjectLabel || '-' }}</strong>
        </div>

        <div class="form-section">
          <div class="form-section-title">基础配置</div>
          <ElRow :gutter="16">
            <ElCol :span="12">
              <ElFormItem label="渠道分组" prop="channelGroupCode">
                <ElSelect
                  v-model="formData.channelGroupCode"
                  :disabled="submitLoading || dialogMode === 'edit'"
                  filterable
                  placeholder="请选择渠道分组"
                  style="width: 100%"
                  @change="handleDialogGroupChange"
                  @visible-change="handleBindingSourceVisibleChange"
                >
                  <ElOption
                    v-for="item in dialogGroupOptions"
                    :key="item.code"
                    :label="item.label"
                    :value="item.code"
                    :disabled="item.disabled"
                  />
                </ElSelect>
              </ElFormItem>
            </ElCol>
            <ElCol :span="12">
              <ElFormItem v-if="dialogMode === 'add'" label="渠道实例" prop="channelId">
                <ElSelect
                  v-model="formData.channelId"
                  :disabled="submitLoading"
                  filterable
                  placeholder="请选择渠道实例"
                  style="width: 100%"
                  @change="handleChannelChange"
                  @visible-change="handleBindingSourceVisibleChange"
                >
                  <ElOption
                    v-for="item in filteredChannelOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                  <template #empty>
                    <div class="select-empty">该分组暂无可绑定实例</div>
                  </template>
                </ElSelect>
              </ElFormItem>
              <ElFormItem v-else label="渠道实例">
                <ElInput :model-value="editingChannelLabel" disabled placeholder="当前绑定实例" />
              </ElFormItem>
            </ElCol>
          </ElRow>
          <ElAlert
            v-if="dialogMode === 'add' && filteredChannelOptions.length === 0"
            class="bind-form-alert"
            title="该分组下暂无可新增绑定的渠道实例，请先到渠道中心新增实例或检查是否已绑定"
            type="warning"
            show-icon
            :closable="false"
          />
        </div>

        <div class="form-section">
          <div class="form-section-title">凭据策略</div>
          <ElRow :gutter="16">
            <ElCol v-if="isBuildGroup" :span="12">
              <ElFormItem label="默认构建渠道" prop="isDefault">
                <ElSwitch
                  v-model="formData.isDefault"
                  :disabled="submitLoading || (dialogMode === 'edit' && editingBinding?.isDefault)"
                  inline-prompt
                  active-text="是"
                  inactive-text="否"
                />
              </ElFormItem>
            </ElCol>
            <ElCol :span="isBuildGroup ? 12 : 24">
              <ElFormItem label="允许全局凭据" prop="allowUseGlobalCredential">
                <ElSwitch
                  v-model="formData.allowUseGlobalCredential"
                  :disabled="submitLoading"
                  inline-prompt
                  active-text="是"
                  inactive-text="否"
                />
              </ElFormItem>
            </ElCol>
          </ElRow>
          <ElFormItem label="策略说明">
            <div class="form-tip">
              {{
                formData.allowUseGlobalCredential
                  ? '允许直接使用渠道实例的全局凭据'
                  : '禁用全局凭据，项目级凭据请到凭据管理中维护'
              }}
            </div>
          </ElFormItem>
          <ElFormItem label="项目凭据" prop="projectCredentialId">
            <ElSelect
              v-model="formData.projectCredentialId"
              :disabled="submitLoading"
              clearable
              filterable
              placeholder="请选择该渠道类型的项目级凭据"
              style="width: 100%"
              @visible-change="handleCredentialVisibleChange"
            >
              <ElOption
                v-for="item in credentialOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
              <template #empty>
                <div class="select-empty">当前项目暂无匹配凭据，请先到凭证管理新增</div>
              </template>
            </ElSelect>
          </ElFormItem>
        </div>

        <div class="form-section">
          <div class="form-section-title">扩展配置</div>
          <template v-if="isTektonBinding">
            <ElRow :gutter="16">
              <ElCol :span="12">
                <ElFormItem label="流水线 Namespace" prop="tektonConfig.namespace">
                  <ElInput
                    v-model="formData.tektonConfig.namespace"
                    disabled
                    placeholder="按项目编码自动生成"
                  />
                </ElFormItem>
              </ElCol>
              <ElCol :span="12">
                <ElFormItem label="Task 命名空间">
                  <ElInput
                    :model-value="selectedTektonTaskNamespace"
                    disabled
                    placeholder="请先选择 Tekton 渠道实例"
                  />
                </ElFormItem>
              </ElCol>
            </ElRow>
            <ElFormItem label="ServiceAccount">
              <ElInput
                v-model="formData.tektonConfig.serviceAccountName"
                :disabled="submitLoading"
                placeholder="可选，必须已存在"
              />
            </ElFormItem>
            <ElFormItem label="配置说明">
              <div class="form-tip">
                Pipeline 和 PipelineRun 固定使用 devops-项目编码命名空间，平台会自动创建；Task
                命名空间在 Tekton 渠道实例中维护。
              </div>
              <div v-if="!selectedTektonTaskNamespace" class="form-tip is-warning">
                请先到渠道中心配置 Task 命名空间。
              </div>
            </ElFormItem>
          </template>
          <ElFormItem v-else label="绑定配置" prop="bindingConfig">
            <ElInput
              v-model="formData.bindingConfig"
              :disabled="submitLoading"
              type="textarea"
              :rows="4"
              placeholder="请输入 JSON 配置，非必填"
            />
          </ElFormItem>
        </div>
      </ElForm>
      <template #footer>
        <ElButton :disabled="submitLoading || dialogLoading" @click="dialogVisible = false">
          取消
        </ElButton>
        <ElButton
          type="primary"
          :loading="submitLoading"
          :disabled="dialogLoading"
          @click="handleSubmit"
        >
          {{ dialogMode === 'add' ? '新增' : '保存' }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onActivated, onMounted, reactive, ref, watch } from 'vue'
  import { storeToRefs } from 'pinia'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import { Link, RefreshRight } from '@element-plus/icons-vue'
  import { formatTimestamp } from '@/utils/format'
  import {
    addDevopsProjectChannelApi,
    deleteDevopsProjectChannelBindingApi,
    listDevopsProjectChannelBindingApi,
    updateDevopsProjectChannelBindingApi,
    type DevopsProjectChannelBinding
  } from '@/api/devops/project'
  import {
    listDevopsChannelApi,
    listDevopsChannelGroupApi,
    listDevopsChannelTypeApi,
    type DevopsChannel,
    type DevopsChannelGroup,
    type DevopsChannelType,
    listDevopsCredentialApi,
    type DevopsCredential
  } from '@/api/devops/channel'
  import { useDevopsProjectStore } from '@/store/modules/devopsProject'

  defineOptions({ name: 'DevopsProjectChannel' })

  const baseLoading = ref(false)
  const loading = ref(false)
  const submitLoading = ref(false)
  const dialogLoading = ref(false)
  const dialogVisible = ref(false)
  const dialogMode = ref<'add' | 'edit'>('add')
  const formRef = ref<FormInstance>()
  const groups = ref<DevopsChannelGroup[]>([])
  const channels = ref<DevopsChannel[]>([])
  const channelTypes = ref<DevopsChannelType[]>([])
  const credentials = ref<DevopsCredential[]>([])
  const bindings = ref<DevopsProjectChannelBinding[]>([])
  const devopsProjectStore = useDevopsProjectStore()
  const { projects, selectedProject, selectedProjectId } = storeToRefs(devopsProjectStore)
  const editingBinding = ref<DevopsProjectChannelBinding | null>(null)

  const defaultForm = () => ({
    projectId: '',
    channelGroupCode: '',
    channelId: '',
    isDefault: false,
    allowUseGlobalCredential: true,
    projectCredentialId: '',
    bindingConfig: '',
    tektonConfig: {
      namespace: '',
      serviceAccountName: '',
      pipelinePrefix: ''
    },
    status: 1
  })

  const formData = reactive(defaultForm())

  const rules: FormRules = {
    projectId: [{ required: true, message: '请选择项目', trigger: 'change' }],
    channelGroupCode: [{ required: true, message: '请选择渠道分组', trigger: 'change' }],
    channelId: [{ required: true, message: '请选择渠道实例', trigger: 'change' }],
    'tektonConfig.namespace': [{ required: true, message: '请输入 Namespace', trigger: 'blur' }]
  }

  const defaultBindingCount = computed(() => bindings.value.filter((item) => item.isDefault).length)
  const selectedProjectLabel = computed(() => {
    if (!selectedProject.value) return ''
    return `${selectedProject.value.name}（${selectedProject.value.code}）`
  })
  const visibleGroups = computed(() =>
    groups.value.filter((item) => item.status === 1 && groupBindings(item.code).length > 0)
  )
  // 通过 allowedChannelTypes 识别构建渠道分组，不再硬编码组 code
  const buildGroupCodes = computed(() =>
    groups.value
      .filter(
        (g) =>
          g.allowedChannelTypes?.includes('jenkins') || g.allowedChannelTypes?.includes('tekton')
      )
      .map((g) => g.code)
  )
  const isBuildGroup = computed(() => buildGroupCodes.value.includes(formData.channelGroupCode))
  const selectedDialogChannel = computed(() => {
    if (dialogMode.value === 'edit' && editingBinding.value) {
      return channelById(editingBinding.value.channelId)
    }
    return channelById(formData.channelId)
  })
  const isTektonBinding = computed(() => selectedDialogChannel.value?.channelType === 'tekton')
  const selectedTektonTaskNamespace = computed(() =>
    tektonTaskNamespaceByChannel(selectedDialogChannel.value)
  )
  const editingChannelLabel = computed(() => {
    if (!editingBinding.value) return ''
    return `${bindingDisplayName(editingBinding.value)}（${bindingDisplayCode(editingBinding.value)} / ${channelTypeLabel(editingBinding.value.channelType, editingBinding.value.channelGroupCode)}）`
  })

  const availableChannelsByGroup = computed(() => {
    const boundIdsByGroup = bindings.value.reduce(
      (acc, item) => {
        if (!acc[item.channelGroupCode]) acc[item.channelGroupCode] = new Set<string>()
        acc[item.channelGroupCode].add(item.channelId)
        return acc
      },
      {} as Record<string, Set<string>>
    )
    return groups.value.reduce(
      (acc, group) => {
        const boundIds = boundIdsByGroup[group.code] || new Set<string>()
        acc[group.code] = channels.value.filter(
          (item) =>
            item.groupId === group.id &&
            item.status === 1 &&
            !item.isSystem &&
            !boundIds.has(item.id)
        )
        return acc
      },
      {} as Record<string, DevopsChannel[]>
    )
  })

  const dialogGroupOptions = computed(() =>
    groups.value.map((item) => {
      const count = availableChannelsByGroup.value[item.code]?.length || 0
      return {
        code: item.code,
        label: `${item.name}（${item.code} / 可绑定 ${count} 个）`,
        disabled: dialogMode.value === 'add' && count === 0
      }
    })
  )

  const filteredChannelOptions = computed(() => {
    return (availableChannelsByGroup.value[formData.channelGroupCode] || []).map((item) => ({
      label: `${channelDisplayName(item)}（${channelDisplayCode(item)} / ${channelTypeLabel(item.channelType, formData.channelGroupCode)}）`,
      value: item.id
    }))
  })

  const credentialOptions = computed(() =>
    credentials.value
      .filter((item) => item.status === 1)
      .map((item) => ({
        label: `${item.name}（${credentialTypeLabel(item.credentialType)}）`,
        value: item.id
      }))
  )

  const groupBindings = (groupCode: string) =>
    bindings.value.filter((item) => item.channelGroupCode === groupCode)

  const channelTypeLabel = (type: string, groupCode?: string) => {
    const channelType = channelTypes.value.find(
      (item) => item.code === type && (!groupCode || item.groupCode === groupCode)
    )
    if (channelType?.name) return channelType.name
    const labels: Record<string, string> = {
      jenkins: 'Jenkins',
      tekton: 'Tekton',
      gitlab: 'GitLab',
      svn: 'SVN',
      gitee: 'Gitee',
      github: 'GitHub',
      nexus: 'Nexus',
      jfrog: 'JFrog',
      harbor: 'Harbor',
      registry: 'Docker Registry',
      aliyun_registry: '阿里云镜像仓库',
      sonarqube: 'SonarQube',
      spotbugs: 'SpotBugs',
      trivy: 'Trivy',
      kube_bench: 'kube-bench',
      kubernetes: 'Kubernetes',
      host: '主机',
      host_group: '主机组',
      'kube-nova': 'kube-nova',
      kubenova: 'kube-nova',
      argocd: 'Argo CD'
    }
    return labels[type] || type
  }

  const channelDisplayName = (item: DevopsChannel) =>
    item.name || channelTypeLabel(item.channelType, groupCodeById(item.groupId)) || '未命名渠道'

  const channelDisplayCode = (item: DevopsChannel) =>
    item.code || channelTypeLabel(item.channelType, groupCodeById(item.groupId)) || '未配置编码'

  const bindingDisplayName = (item: DevopsProjectChannelBinding) =>
    item.channelName || channelTypeLabel(item.channelType, item.channelGroupCode) || '未命名渠道'

  const bindingDisplayCode = (item: DevopsProjectChannelBinding) =>
    item.channelCode || channelTypeLabel(item.channelType, item.channelGroupCode) || '未配置编码'

  const channelById = (id: string) => channels.value.find((item) => item.id === id)
  const groupCodeById = (id?: string) => groups.value.find((item) => item.id === id)?.code || ''

  const credentialTypeLabel = (type: string) => {
    const labels: Record<string, string> = {
      username_password: '用户名密码',
      token: 'Token',
      ssh_key: 'SSH Key',
      kubeconfig: 'Kubeconfig',
      secret_text: 'Secret Text',
      certificate: '证书',
      json: 'JSON'
    }
    return labels[type] || type || '未知类型'
  }

  const bindingCredentialLabel = (item: DevopsProjectChannelBinding) => {
    const channel = channelById(item.channelId)
    if (item.allowUseGlobalCredential) {
      return channel?.credentialId ? '允许使用全局凭据' : '允许使用全局凭据，渠道暂未配置'
    }
    return item.projectCredentialId ? '禁用全局凭据，已存在项目凭据' : '禁用全局凭据'
  }

  const parseTektonConfig = (content?: string) => {
    try {
      const data = JSON.parse(content || '{}')
      return {
        namespace: String(data.namespace || ''),
        serviceAccountName: String(data.serviceAccountName || ''),
        pipelinePrefix: String(data.pipelinePrefix || '')
      }
    } catch {
      return { namespace: '', serviceAccountName: '', pipelinePrefix: '' }
    }
  }

  const stringifyTektonConfig = (config: ReturnType<typeof parseTektonConfig>) =>
    JSON.stringify({
      namespace: String(config.namespace || '').trim(),
      serviceAccountName: String(config.serviceAccountName || '').trim(),
      pipelinePrefix: String(config.pipelinePrefix || '').trim()
    })

  const parseTektonChannelConfig = (content?: string) => {
    try {
      const data = JSON.parse(content || '{}')
      return {
        taskNamespace: String(data.taskNamespace || '')
      }
    } catch {
      return { taskNamespace: '' }
    }
  }

  const tektonTaskNamespaceByChannel = (channel?: DevopsChannel) =>
    parseTektonChannelConfig(channel?.config).taskNamespace

  const normalizeKubernetesName = (value?: string) => {
    let name = String(value || '')
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9-]+/g, '-')
      .replace(/^-+|-+$/g, '')
    if (name.length > 57) {
      name = name.slice(0, 57).replace(/-+$/g, '')
    }
    return name || 'project'
  }

  const projectTektonNamespace = (projectId?: string) => {
    const project = projects.value.find((item) => item.id === projectId)
    return `devops-${normalizeKubernetesName(project?.code || projectId)}`
  }

  const syncTektonNamespace = () => {
    formData.tektonConfig.namespace = projectTektonNamespace(formData.projectId)
  }

  const availableChannelCount = (groupCode: string) =>
    availableChannelsByGroup.value[groupCode]?.length || 0

  const loadCredentials = async () => {
    const channel = selectedDialogChannel.value
    if (!formData.projectId || !formData.channelGroupCode || !channel?.channelType) {
      credentials.value = []
      return
    }
    await loadChannelTypesByGroup(formData.channelGroupCode)
    const res = await listDevopsCredentialApi({
      page: 1,
      pageSize: 200,
      scope: 'project',
      projectId: formData.projectId,
      channelGroupCode: formData.channelGroupCode,
      channelType: channel.channelType,
      status: 1
    })
    credentials.value = res.items || []
  }

  const mergeChannelTypes = (items: DevopsChannelType[]) => {
    const data = new Map(channelTypes.value.map((item) => [`${item.groupCode}:${item.code}`, item]))
    items.forEach((item) => data.set(`${item.groupCode}:${item.code}`, item))
    channelTypes.value = Array.from(data.values())
  }

  const loadChannelTypesByGroup = async (groupCode?: string) => {
    const code = String(groupCode || '').trim()
    if (!code) return []
    const res = await listDevopsChannelTypeApi({
      page: 1,
      pageSize: 200,
      status: 1,
      groupCode: code
    })
    const items = res.items || []
    mergeChannelTypes(items)
    return items
  }

  const loadChannelTypesForBindings = async (items: DevopsProjectChannelBinding[]) => {
    const groupCodes = Array.from(
      new Set(items.map((item) => String(item.channelGroupCode || '').trim()).filter(Boolean))
    )
    await Promise.all(groupCodes.map((groupCode) => loadChannelTypesByGroup(groupCode)))
  }

  const loadChannelTypesForChannels = async (items: DevopsChannel[]) => {
    const groupCodes = Array.from(
      new Set(items.map((item) => groupCodeById(item.groupId)).filter(Boolean))
    )
    await Promise.all(groupCodes.map((groupCode) => loadChannelTypesByGroup(groupCode)))
  }

  const getBaseData = async () => {
    baseLoading.value = true
    try {
      await devopsProjectStore.fetchProjects(true)
      const [groupRes, channelRes] = await Promise.all([
        listDevopsChannelGroupApi({ page: 1, pageSize: 200, status: 1 }),
        listDevopsChannelApi({ page: 1, pageSize: 200, status: 1 })
      ])
      groups.value = (groupRes.items || []).sort((a, b) => a.sortOrder - b.sortOrder)
      channels.value = (channelRes.items || []).filter((item) => !item.isSystem)
      await loadChannelTypesForChannels(channels.value)
    } finally {
      baseLoading.value = false
    }
  }

  const getBindings = async () => {
    if (!selectedProjectId.value) {
      bindings.value = []
      return
    }
    loading.value = true
    try {
      const bindingRes = await listDevopsProjectChannelBindingApi({
        page: 1,
        pageSize: 200,
        projectId: selectedProjectId.value,
        status: 1
      })
      bindings.value = bindingRes.items || []
      await loadChannelTypesForBindings(bindings.value)
    } finally {
      loading.value = false
    }
  }

  const refreshAll = async () => {
    await getBaseData()
    await getBindings()
  }

  const hasBuildDefault = () =>
    buildGroupCodes.value.some((code) =>
      groupBindings(code).some((item) => item.isDefault && item.status === 1)
    )

  const openAddDialog = async (groupCode?: string) => {
    dialogMode.value = 'add'
    editingBinding.value = null
    Object.assign(formData, defaultForm(), {
      projectId: selectedProjectId.value,
      channelGroupCode: groupCode || ''
    })
    syncTektonNamespace()
    dialogVisible.value = true
    dialogLoading.value = true
    try {
      await Promise.all([getBaseData(), getBindings()])
      const defaultGroupCode =
        groupCode ||
        groups.value.find((item) => availableChannelCount(item.code) > 0)?.code ||
        groups.value[0]?.code ||
        ''
      Object.assign(formData, {
        projectId: selectedProjectId.value,
        channelGroupCode: defaultGroupCode,
        isDefault: buildGroupCodes.value.includes(defaultGroupCode) && !hasBuildDefault()
      })
      syncTektonNamespace()
    } finally {
      dialogLoading.value = false
    }
  }

  const openEditDialog = async (row: DevopsProjectChannelBinding) => {
    dialogMode.value = 'edit'
    editingBinding.value = row
    Object.assign(formData, defaultForm(), {
      projectId: row.projectId,
      channelGroupCode: row.channelGroupCode,
      channelId: row.channelId,
      isDefault: row.isDefault,
      allowUseGlobalCredential: row.allowUseGlobalCredential,
      projectCredentialId: row.projectCredentialId,
      bindingConfig: row.bindingConfig,
      tektonConfig: parseTektonConfig(row.bindingConfig),
      status: row.status || 1
    })
    syncTektonNamespace()
    dialogVisible.value = true
    dialogLoading.value = true
    try {
      await loadCredentials()
    } finally {
      dialogLoading.value = false
    }
  }

  const handleDialogGroupChange = async () => {
    Object.assign(formData, {
      channelId: '',
      isDefault: buildGroupCodes.value.includes(formData.channelGroupCode) && !hasBuildDefault(),
      allowUseGlobalCredential: true,
      projectCredentialId: '',
      bindingConfig: '',
      tektonConfig: {
        namespace: projectTektonNamespace(formData.projectId),
        serviceAccountName: '',
        pipelinePrefix: ''
      }
    })
    credentials.value = []
    await loadChannelTypesByGroup(formData.channelGroupCode)
  }

  const handleChannelChange = async (resetTekton = true) => {
    if (isBuildGroup.value && !hasBuildDefault()) {
      formData.isDefault = true
    }
    if (!resetTekton) return
    formData.projectCredentialId = ''
    Object.assign(formData.tektonConfig, {
      namespace: projectTektonNamespace(formData.projectId),
      serviceAccountName: '',
      pipelinePrefix: ''
    })
    await loadCredentials()
  }

  const resetForm = () => {
    Object.assign(formData, defaultForm())
    editingBinding.value = null
    formRef.value?.clearValidate()
  }

  const handleSubmit = async () => {
    await handleChannelChange(false)
    if (isTektonBinding.value) syncTektonNamespace()
    await formRef.value?.validate()
    if (isTektonBinding.value && !selectedTektonTaskNamespace.value) {
      ElMessage.warning('请先到渠道中心配置 Task 命名空间')
      return
    }
    submitLoading.value = true
    try {
      const projectCredentialId = String(formData.projectCredentialId || '').trim()
      const bindingConfig = isTektonBinding.value
        ? stringifyTektonConfig(formData.tektonConfig)
        : formData.bindingConfig
      if (dialogMode.value === 'add') {
        await addDevopsProjectChannelApi(formData.projectId, {
          channelId: formData.channelId,
          isDefault: isBuildGroup.value ? formData.isDefault : false,
          allowUseGlobalCredential: formData.allowUseGlobalCredential,
          ...(projectCredentialId ? { projectCredentialId } : {}),
          bindingConfig
        })
      } else if (editingBinding.value) {
        await updateDevopsProjectChannelBindingApi(editingBinding.value.id, {
          isDefault: isBuildGroup.value ? formData.isDefault : false,
          allowUseGlobalCredential: formData.allowUseGlobalCredential,
          projectCredentialId,
          bindingConfig,
          status: formData.status
        })
      }
      ElMessage.success(dialogMode.value === 'add' ? '绑定成功' : '保存成功')
      dialogVisible.value = false
      await getBindings()
    } finally {
      submitLoading.value = false
    }
  }

  const handleDelete = async (row: DevopsProjectChannelBinding) => {
    await ElMessageBox.confirm(`确定解绑"${bindingDisplayName(row)}"吗？`, '解绑确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteDevopsProjectChannelBindingApi(row.id)
    ElMessage.success('删除成功')
    await getBindings()
  }

  const handleBindingSourceVisibleChange = async (visible: boolean) => {
    if (visible) {
      await getBaseData()
      await getBindings()
    }
  }

  const handleCredentialVisibleChange = async (visible: boolean) => {
    if (visible) {
      await loadCredentials()
    }
  }

  let activatedOnce = false
  onMounted(refreshAll)
  onActivated(() => {
    if (!activatedOnce) {
      activatedOnce = true
      return
    }
    refreshAll()
  })
  watch(selectedProjectId, () => {
    getBindings()
  })
</script>

<style lang="scss" scoped>
  .devops-project-channel-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .page-toolbar {
    display: grid;
    grid-template-columns: minmax(140px, 1fr) minmax(320px, 440px) auto;
    gap: 16px;
    align-items: center;
    padding: 14px 16px;
    margin-bottom: 12px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .toolbar-title {
    min-width: 0;
  }

  .toolbar-title h2 {
    margin: 0;
    overflow: hidden;
    font-size: 18px;
    font-weight: 600;
    line-height: 24px;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .summary-item {
    min-width: 0;
    padding: 2px 14px;
    border-left: 1px solid var(--el-border-color-lighter);
  }

  .summary-item span {
    display: block;
    margin-bottom: 6px;
    font-size: 12px;
    line-height: 16px;
    color: var(--el-text-color-secondary);
  }

  .summary-list {
    display: grid;
    grid-template-columns: repeat(3, minmax(100px, 1fr));
  }

  .summary-item strong {
    font-size: 20px;
    line-height: 1;
    color: var(--el-text-color-primary);
  }

  .toolbar-actions {
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: flex-end;
  }

  .binding-scroll {
    flex: 1;
    min-height: 0;
    padding-right: 4px;
    overflow-y: auto;
    scrollbar-gutter: stable;
  }

  .group-section-list {
    display: grid;
    gap: 12px;
  }

  .group-section {
    min-width: 0;
    padding: 14px;
    overflow: hidden;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .group-section-head {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 12px;
  }

  .group-title-wrap {
    display: flex;
    gap: 12px;
    align-items: center;
    min-width: 0;
  }

  .group-icon {
    display: flex;
    flex: 0 0 42px;
    align-items: center;
    justify-content: center;
    width: 42px;
    height: 42px;
    font-size: 24px;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .group-title {
    min-width: 0;
  }

  .group-name-row {
    display: flex;
    gap: 8px;
    align-items: center;
    min-width: 0;
  }

  .group-name-row strong {
    overflow: hidden;
    font-size: 16px;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .group-title span {
    display: block;
    max-width: 520px;
    margin-top: 4px;
    overflow: hidden;
    font-size: 13px;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .group-actions {
    display: flex;
    flex: 0 0 auto;
    gap: 8px;
    align-items: center;
  }

  .binding-card-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 12px;
  }

  .binding-card {
    min-width: 0;
    overflow: hidden;
    border-radius: 8px;
  }

  .binding-card-head {
    display: flex;
    gap: 10px;
    align-items: flex-start;
    justify-content: space-between;
    min-height: 70px;
    padding: 14px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .binding-main {
    min-width: 0;
  }

  .binding-main strong,
  .binding-main span {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .binding-main strong {
    color: var(--el-text-color-primary);
  }

  .binding-main span {
    margin-top: 6px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .binding-card-body {
    display: grid;
    gap: 10px;
    min-height: 136px;
    padding: 14px;
  }

  .binding-meta {
    display: grid;
    grid-template-columns: 72px minmax(0, 1fr);
    gap: 8px;
    align-items: center;
    font-size: 13px;
  }

  .binding-meta span {
    color: var(--el-text-color-secondary);
    white-space: nowrap;
  }

  .binding-meta strong {
    overflow: hidden;
    font-weight: 500;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .binding-meta :deep(.el-tag) {
    justify-self: start;
    max-width: 100%;
  }

  .binding-card-footer {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
    padding: 10px 14px;
    background: var(--el-fill-color-extra-light);
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .group-empty {
    padding: 20px 0 8px;
  }

  .bind-form-alert {
    margin: 0 0 12px;
  }

  .dialog-context {
    display: grid;
    grid-template-columns: 72px minmax(0, 1fr);
    gap: 12px;
    align-items: center;
    padding: 10px 12px;
    margin-bottom: 14px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .dialog-context span {
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }

  .dialog-context strong {
    min-width: 0;
    overflow: hidden;
    font-size: 13px;
    font-weight: 500;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .form-section {
    padding: 12px 14px 2px;
    margin-bottom: 14px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .form-section-title {
    margin-bottom: 12px;
    font-size: 14px;
    font-weight: 600;
    line-height: 20px;
    color: var(--el-text-color-primary);
  }

  .form-tip {
    font-size: 12px;
    line-height: 18px;
    color: var(--el-text-color-secondary);
  }

  .form-tip.is-warning {
    color: var(--el-color-warning);
  }

  .select-empty {
    padding: 10px 0;
    font-size: 13px;
    color: var(--el-text-color-secondary);
    text-align: center;
  }

  :deep(.el-dialog__body) {
    max-height: calc(100vh - 220px);
    overflow-y: auto;
  }

  :deep(.el-dialog__body .el-form-item__label) {
    font-weight: 500;
  }

  @media (width <= 960px) {
    .page-toolbar {
      grid-template-columns: 1fr;
    }

    .summary-list {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }

    .summary-item:first-child {
      border-left: 0;
    }

    .group-section-head {
      flex-direction: column;
      align-items: flex-start;
    }
  }

  @media (width <= 720px) {
    :deep(.el-dialog) {
      width: calc(100vw - 24px) !important;
    }

    :deep(.el-col-12) {
      max-width: 100%;
      flex: 0 0 100%;
    }

    .dialog-context {
      grid-template-columns: 1fr;
      gap: 4px;
    }
  }
</style>
