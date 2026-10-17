<template>
  <div class="devops-setting-channel-page art-full-height">
    <div class="channel-context">
      <div>
        <span class="context-label">当前 DevOps 项目</span>
        <strong :title="currentProjectLabel">{{ currentProjectLabel }}</strong>
      </div>
      <ElTag type="info" size="small">项目渠道只允许查看、测试和更换凭据</ElTag>
    </div>

    <div class="summary-grid">
      <div class="summary-card">
        <span>绑定渠道</span>
        <strong>{{ bindings.length }}</strong>
      </div>
      <div class="summary-card">
        <span>正常渠道</span>
        <strong>{{ healthyCount }}</strong>
      </div>
      <div class="summary-card">
        <span>项目凭据</span>
        <strong>{{ projectCredentialCount }}</strong>
      </div>
      <div class="summary-card">
        <span>待绑定凭据</span>
        <strong>{{ missingCredentialCount }}</strong>
      </div>
      <div class="summary-action">
        <ElButton :icon="RefreshRight" :loading="loading" @click="refreshAll">刷新</ElButton>
      </div>
    </div>

    <div v-loading="loading" class="channel-scroll">
      <ElEmpty v-if="!selectedProjectId" description="请选择顶部 DevOps 项目" />
      <ElEmpty v-else-if="visibleGroups.length === 0" description="当前项目暂无绑定渠道" />
      <div v-else class="group-list">
        <section v-for="group in visibleGroups" :key="group.id" class="group-section">
          <div class="group-head">
            <div class="group-title">
              <div class="group-icon" :style="{ color: group.iconColor || '#2563eb' }">
                <ArtSvgIcon :icon="group.icon || 'ri:folder-3-line'" />
              </div>
              <div>
                <strong :title="group.name">{{ group.name }}</strong>
                <span :title="group.code">{{ group.code }}</span>
              </div>
            </div>
            <ElTag size="small" type="info">{{ groupBindings(group.code).length }} 个渠道</ElTag>
          </div>

          <div class="channel-card-grid">
            <ElCard
              v-for="item in groupBindings(group.code)"
              :key="item.id"
              class="channel-card"
              shadow="hover"
              :body-style="{ padding: '0' }"
            >
              <div class="channel-card-head">
                <div class="channel-identity">
                  <div class="channel-icon" :style="{ color: channelIcon(item).color }">
                    <ArtSvgIcon :icon="channelIcon(item).icon" />
                  </div>
                  <div class="channel-name">
                    <strong :title="bindingDisplayName(item)">{{
                      bindingDisplayName(item)
                    }}</strong>
                    <span :title="bindingDisplayCode(item)">{{ bindingDisplayCode(item) }}</span>
                  </div>
                </div>
                <ElTag size="small" :type="credentialReady(item) ? 'info' : 'warning'">
                  {{ credentialReady(item) ? '已配置凭据' : '待绑定凭据' }}
                </ElTag>
              </div>

              <div class="channel-card-body">
                <div class="meta-row">
                  <span>类型</span>
                  <strong :title="channelTypeLabel(item.channelType, item.channelGroupCode)">
                    {{ channelTypeLabel(item.channelType, item.channelGroupCode) }}
                  </strong>
                </div>
                <div class="meta-row">
                  <span>地址</span>
                  <strong :title="channelById(item.channelId)?.endpoint || '-'">
                    {{ channelById(item.channelId)?.endpoint || '-' }}
                  </strong>
                </div>
                <div class="meta-row">
                  <span>凭据</span>
                  <strong :title="effectiveCredentialLabel(item)">
                    {{ effectiveCredentialLabel(item) }}
                  </strong>
                </div>
                <div class="meta-row">
                  <span>版本</span>
                  <strong :title="effectiveVersion(item)">{{ effectiveVersion(item) }}</strong>
                </div>
                <div class="meta-row">
                  <span>状态</span>
                  <ElTag size="small" :type="projectHealthState(item).type">
                    {{ projectHealthState(item).label }}
                  </ElTag>
                </div>
              </div>

              <div class="channel-card-footer">
                <ElButton
                  text
                  type="primary"
                  :loading="testingId === item.id"
                  @click="handleTest(item)"
                >
                  测试
                </ElButton>
                <ElButton text type="primary" @click="openCredentialDialog(item)"
                  >更换凭据</ElButton
                >
              </div>
            </ElCard>
          </div>
        </section>
      </div>
    </div>

    <ElDialog
      v-model="credentialDialogVisible"
      title="更换项目渠道凭据"
      width="680px"
      align-center
      :close-on-click-modal="false"
      @close="resetCredentialForm"
    >
      <ElForm
        ref="credentialFormRef"
        :model="credentialForm"
        :rules="credentialRules"
        label-width="132px"
      >
        <ElFormItem label="渠道实例">
          <ElInput :model-value="editingBindingLabel" disabled />
        </ElFormItem>
        <ElFormItem label="全局凭据策略">
          <ElTag :type="editingBinding?.allowUseGlobalCredential ? 'success' : 'warning'">
            {{ editingBinding?.allowUseGlobalCredential ? '允许使用' : '禁止使用' }}
          </ElTag>
          <div class="form-tip">
            开启后未选择项目凭据时使用渠道全局凭据；关闭后不使用全局凭据，可到凭证管理维护项目级凭据。
          </div>
        </ElFormItem>
        <ElFormItem label="项目凭据" prop="projectCredentialId">
          <ElSelect
            v-model="credentialForm.projectCredentialId"
            clearable
            filterable
            placeholder="请选择项目级凭据"
            style="width: 100%"
            @visible-change="handleCredentialVisibleChange"
          >
            <ElOption
              v-for="item in filteredCredentialOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
            <template #empty>
              <div class="select-empty">当前项目暂无匹配凭据，请先到凭证管理新增</div>
            </template>
          </ElSelect>
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton :disabled="submitLoading" @click="credentialDialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="submitLoading" @click="handleSaveCredential">
          保存
        </ElButton>
      </template>
    </ElDialog>

    <ElDialog v-model="testDialogVisible" title="连接测试结果" width="640px" align-center>
      <div
        v-if="testResult"
        class="test-result"
        :class="testResult.success ? 'is-success' : 'is-error'"
      >
        <div class="test-result-head">
          <div>
            <strong>{{ testResult.success ? '连接正常' : '连接异常' }}</strong>
            <span :title="testResult.message">{{ testResult.message }}</span>
          </div>
          <ElTag :type="testResult.success ? 'success' : 'danger'">
            {{ testResult.healthStatus }}
          </ElTag>
        </div>
        <div class="test-result-grid">
          <div v-for="row in testRows" :key="row.label" class="test-result-row">
            <span>{{ row.label }}</span>
            <strong :title="row.value">{{ row.value }}</strong>
          </div>
        </div>
      </div>
      <template #footer>
        <ElButton type="primary" @click="testDialogVisible = false">确定</ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onActivated, onMounted, reactive, ref, watch } from 'vue'
  import { storeToRefs } from 'pinia'
  import { ElMessage } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import { RefreshRight } from '@element-plus/icons-vue'
  import { formatTimestamp } from '@/utils/format'
  import { useDevopsProjectStore } from '@/store/modules/devopsProject'
  import {
    listDevopsProjectChannelBindingApi,
    testDevopsProjectChannelBindingApi,
    updateDevopsProjectChannelBindingApi,
    type DevopsProjectChannelBinding,
    type TestDevopsProjectChannelResponse
  } from '@/api/devops/project'
  import {
    listDevopsChannelApi,
    listDevopsChannelGroupApi,
    listDevopsChannelTypeApi,
    listDevopsCredentialApi,
    type DevopsChannel,
    type DevopsChannelGroup,
    type DevopsChannelType,
    type DevopsCredential
  } from '@/api/devops/channel'

  defineOptions({ name: 'DevopsProjectChannelSettings' })

  type TagType = 'primary' | 'success' | 'warning' | 'info' | 'danger'

  const devopsProjectStore = useDevopsProjectStore()
  const { selectedProject, selectedProjectId } = storeToRefs(devopsProjectStore)

  const loading = ref(false)
  const submitLoading = ref(false)
  const testingId = ref('')
  const credentialDialogVisible = ref(false)
  const testDialogVisible = ref(false)
  const credentialFormRef = ref<FormInstance>()
  const bindings = ref<DevopsProjectChannelBinding[]>([])
  const channels = ref<DevopsChannel[]>([])
  const groups = ref<DevopsChannelGroup[]>([])
  const channelTypes = ref<DevopsChannelType[]>([])
  const credentials = ref<DevopsCredential[]>([])
  const editingBinding = ref<DevopsProjectChannelBinding | null>(null)
  const testResult = ref<TestDevopsProjectChannelResponse | null>(null)

  const credentialForm = reactive({
    projectCredentialId: ''
  })

  const credentialRules: FormRules = {}

  const channelVisualMap: Record<string, { icon: string; color: string }> = {
    jenkins: { icon: 'simple-icons:jenkins', color: '#d24939' },
    tekton: { icon: 'ri:flow-chart', color: '#2563eb' },
    buildkit: { icon: 'simple-icons:docker', color: '#0369a1' },
    gitlab: { icon: 'simple-icons:gitlab', color: '#fc6d26' },
    github: { icon: 'simple-icons:github', color: '#24292f' },
    gitee: { icon: 'simple-icons:gitee', color: '#c71d23' },
    svn: { icon: 'ri:git-repository-line', color: '#64748b' },
    nexus: { icon: 'ri:archive-stack-line', color: '#16a34a' },
    jfrog: { icon: 'simple-icons:jfrog', color: '#0f9f6e' },
    harbor: { icon: 'simple-icons:harbor', color: '#0f766e' },
    registry: { icon: 'simple-icons:docker', color: '#0369a1' },
    aliyun_registry: { icon: 'ri:cloud-line', color: '#ff6a00' },
    sonarqube: { icon: 'simple-icons:sonarqube', color: '#4b9fd5' },
    spotbugs: { icon: 'ri:bug-line', color: '#b45309' },
    trivy: { icon: 'ri:shield-check-line', color: '#0f766e' },
    kube_bench: { icon: 'ri:shield-keyhole-line', color: '#7c3aed' },
    kubernetes: { icon: 'simple-icons:kubernetes', color: '#326ce5' },
    host: { icon: 'ri:server-line', color: '#475569' },
    host_group: { icon: 'ri:server-line', color: '#7c3aed' },
    'kube-nova': { icon: 'ri:cloud-line', color: '#0891b2' },
    kubenova: { icon: 'ri:cloud-line', color: '#0891b2' },
    argocd: { icon: 'simple-icons:argo', color: '#ef7b4d' }
  }

  const currentProjectLabel = computed(() => {
    if (!selectedProject.value) return '未选择项目'
    return `${selectedProject.value.name}（${selectedProject.value.code}）`
  })

  const visibleGroups = computed(() =>
    groups.value.filter((item) => item.status === 1 && groupBindings(item.code).length > 0)
  )

  const healthyCount = computed(
    () => bindings.value.filter((item) => projectHealthState(item).status === 'healthy').length
  )

  const projectCredentialCount = computed(
    () => bindings.value.filter((item) => !!item.projectCredentialId).length
  )

  const missingCredentialCount = computed(
    () => bindings.value.filter((item) => !credentialReady(item)).length
  )

  const editingBindingLabel = computed(() =>
    editingBinding.value
      ? `${bindingDisplayName(editingBinding.value)}（${bindingDisplayCode(editingBinding.value)}）`
      : ''
  )

  const baseChannelCredentialTypes = ['username_password', 'token', 'json']
  const channelCredentialTypes = (types?: string[]) => {
    const values = (types || []).filter(Boolean)
    return Array.from(new Set(values.length > 0 ? values : baseChannelCredentialTypes))
  }

  const currentChannelType = computed(() => {
    if (!editingBinding.value) return null
    return (
      channelTypes.value.find(
        (item) =>
          item.code === editingBinding.value?.channelType &&
          item.groupCode === editingBinding.value?.channelGroupCode
      ) || null
    )
  })

  const filteredCredentialOptions = computed(() => {
    const allowed = channelCredentialTypes(currentChannelType.value?.credentialTypes)
    return credentials.value
      .filter((item) => item.status === 1)
      .filter((item) => allowed.length === 0 || allowed.includes(item.credentialType))
      .map((item) => ({
        label: `${item.name}（${credentialTypeLabel(item.credentialType)}）`,
        value: item.id
      }))
  })

  const testRows = computed(() => {
    if (!testResult.value) return []
    const metadata = parseMetadata(testResult.value.metadata)
    return [
      { label: '产品名称', value: metadata.productName || '-' },
      { label: '版本', value: metadataVersion(metadata) },
      { label: 'API 版本', value: metadata.apiVersion || '-' },
      { label: '认证用户', value: metadata.authUser || '-' },
      { label: '能力信息', value: formatCapabilities(metadata.capabilities) },
      { label: '检测时间', value: formatTimestamp(testResult.value.checkedAt) }
    ]
  })

  const groupBindings = (groupCode: string) =>
    bindings.value.filter((item) => item.channelGroupCode === groupCode)

  const channelById = (id: string) => channels.value.find((item) => item.id === id)

  const credentialById = (id: string) => credentials.value.find((item) => item.id === id)

  const channelIcon = (item: DevopsProjectChannelBinding) => {
    const channel = channelById(item.channelId)
    if (channel?.icon) return { icon: channel.icon, color: channel.iconColor || '#2563eb' }
    return channelVisualMap[item.channelType] || { icon: 'ri:git-branch-line', color: '#2563eb' }
  }

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

  const bindingDisplayName = (item: DevopsProjectChannelBinding) =>
    item.channelName ||
    channelById(item.channelId)?.name ||
    channelTypeLabel(item.channelType, item.channelGroupCode)

  const bindingDisplayCode = (item: DevopsProjectChannelBinding) =>
    item.channelCode || channelById(item.channelId)?.code || '未配置编码'

  const parseMetadata = (raw?: string) => {
    if (!raw) return {} as Record<string, any>
    try {
      return JSON.parse(raw) as Record<string, any>
    } catch {
      return {} as Record<string, any>
    }
  }

  const channelVersion = (item: DevopsProjectChannelBinding) => {
    const metadata = parseMetadata(channelById(item.channelId)?.metadata)
    return metadataVersion(metadata)
  }

  const bindingVersion = (item: DevopsProjectChannelBinding) => {
    const metadata = parseMetadata(item.metadata)
    return metadataVersion(metadata)
  }

  const metadataVersion = (metadata: Record<string, any>) =>
    metadata.version ||
    metadata.productVersion ||
    metadata.engineVersion ||
    metadata.schemaVersion ||
    metadata.benchmarkProfile ||
    apiVersionAsVersion(metadata) ||
    '-'

  const apiVersionAsVersion = (metadata: Record<string, any>) => {
    const productName = String(metadata.productName || '').toLowerCase()
    if (['docker registry', 'github', 'gitee', 'tekton'].includes(productName)) {
      return metadata.apiVersion
    }
    return ''
  }

  const effectiveVersion = (item: DevopsProjectChannelBinding) => {
    if (item.projectCredentialId) return bindingVersion(item)
    if (item.allowUseGlobalCredential) return channelVersion(item)
    return '-'
  }

  const formatCapabilities = (value: any) => {
    if (!value) return '-'
    if (typeof value === 'string') return value
    return Object.entries(value)
      .map(([key, item]) => `${key}: ${item}`)
      .join('、')
  }

  const healthLabel = (status?: string) => {
    if (status === 'healthy') return '正常'
    if (status === 'unhealthy') return '异常'
    return '未知'
  }

  const healthTagType = (status?: string): TagType => {
    if (status === 'healthy') return 'success'
    if (status === 'unhealthy') return 'danger'
    return 'info'
  }

  const projectHealthState = (item: DevopsProjectChannelBinding) => {
    if (!credentialReady(item)) {
      return { status: 'unhealthy', label: '未配置凭据', type: 'warning' as const }
    }
    if (item.projectCredentialId) {
      const status = item.healthStatus || 'unknown'
      return { status, label: healthLabel(status), type: healthTagType(status) }
    }
    if (item.allowUseGlobalCredential) {
      const status = channelById(item.channelId)?.healthStatus || 'unknown'
      return { status, label: healthLabel(status), type: healthTagType(status) }
    }
    const status = item.healthStatus || 'unknown'
    return {
      status,
      label: item.lastCheckMessage ? healthLabel(status) : '未指定项目凭据',
      type: healthTagType(status)
    }
  }

  const credentialReady = (item: DevopsProjectChannelBinding) => {
    if (item.projectCredentialId) return true
    if (item.allowUseGlobalCredential) return !!channelById(item.channelId)?.credentialId
    return false
  }

  const effectiveCredentialLabel = (item: DevopsProjectChannelBinding) => {
    if (item.projectCredentialId) {
      const credential = credentialById(item.projectCredentialId)
      return credential
        ? `项目凭据：${credential.name}（${credentialTypeLabel(credential.credentialType)}）`
        : '项目凭据：已绑定'
    }
    if (item.allowUseGlobalCredential) {
      const credentialId = channelById(item.channelId)?.credentialId || ''
      return credentialId ? '全局凭据：渠道已配置' : '全局凭据：未配置'
    }
    return '不使用全局凭据，未指定项目凭据'
  }

  const loadCredentials = async (binding?: DevopsProjectChannelBinding | null) => {
    if (!selectedProjectId.value) {
      credentials.value = []
      return
    }
    const target = binding || editingBinding.value
    if (!target?.channelGroupCode || !target.channelType) {
      credentials.value = []
      return
    }
    await loadChannelTypesByGroup(target.channelGroupCode)
    const projectRes = await listDevopsCredentialApi({
      page: 1,
      pageSize: 200,
      scope: 'project',
      projectId: selectedProjectId.value,
      channelGroupCode: target.channelGroupCode,
      channelType: target.channelType,
      status: 1
    })
    credentials.value = projectRes.items || []
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

  const refreshAll = async () => {
    if (!devopsProjectStore.initialized) {
      await devopsProjectStore.fetchProjects()
    }
    if (!selectedProjectId.value) {
      bindings.value = []
      ElMessage.warning('请选择项目')
      return
    }
    loading.value = true
    try {
      const [groupRes, channelRes, bindingRes] = await Promise.all([
        listDevopsChannelGroupApi({ page: 1, pageSize: 200, status: 1 }),
        listDevopsChannelApi({ page: 1, pageSize: 200, status: 1 }),
        listDevopsProjectChannelBindingApi({
          page: 1,
          pageSize: 200,
          projectId: selectedProjectId.value,
          status: 1
        })
      ])
      groups.value = (groupRes.items || []).sort((a, b) => a.sortOrder - b.sortOrder)
      channels.value = (channelRes.items || []).filter((item) => !item.isSystem)
      bindings.value = bindingRes.items || []
      await loadChannelTypesForBindings(bindings.value)
      await loadCredentials()
    } finally {
      loading.value = false
    }
  }

  const openCredentialDialog = async (row: DevopsProjectChannelBinding) => {
    editingBinding.value = row
    await loadCredentials(row)
    Object.assign(credentialForm, {
      projectCredentialId: row.projectCredentialId || ''
    })
    credentialDialogVisible.value = true
  }

  const resetCredentialForm = () => {
    editingBinding.value = null
    Object.assign(credentialForm, {
      projectCredentialId: ''
    })
    credentialFormRef.value?.clearValidate()
  }

  const handleSaveCredential = async () => {
    if (!editingBinding.value) return
    await credentialFormRef.value?.validate()
    submitLoading.value = true
    try {
      await updateDevopsProjectChannelBindingApi(editingBinding.value.id, {
        isDefault: editingBinding.value.isDefault,
        allowUseGlobalCredential: editingBinding.value.allowUseGlobalCredential,
        projectCredentialId: credentialForm.projectCredentialId,
        bindingConfig: editingBinding.value.bindingConfig,
        status: editingBinding.value.status || 1
      })
      ElMessage.success('凭据已保存')
      credentialDialogVisible.value = false
      await refreshAll()
    } finally {
      submitLoading.value = false
    }
  }

  const handleTest = async (row: DevopsProjectChannelBinding) => {
    const channel = channelById(row.channelId)
    if (!channel) {
      ElMessage.warning('渠道实例不存在')
      return
    }
    testingId.value = row.id
    try {
      testResult.value = await testDevopsProjectChannelBindingApi(row.id)
      testDialogVisible.value = true
      await refreshAll()
    } finally {
      testingId.value = ''
    }
  }

  const handleCredentialVisibleChange = async (visible: boolean) => {
    if (visible) {
      await loadCredentials()
    }
  }

  watch(selectedProjectId, () => {
    refreshAll()
  })

  let activatedOnce = false
  onMounted(refreshAll)
  onActivated(() => {
    if (!activatedOnce) {
      activatedOnce = true
      return
    }
    refreshAll()
  })
</script>

<style lang="scss" scoped>
  .devops-setting-channel-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .channel-context,
  .summary-card,
  .summary-action,
  .group-section {
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .channel-context {
    display: flex;
    flex: 0 0 auto;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    padding: 12px 14px;
    margin-bottom: 12px;
  }

  .context-label,
  .summary-card span,
  .group-title span,
  .meta-row span,
  .test-result-row span {
    color: var(--el-text-color-secondary);
  }

  .context-label {
    display: block;
    margin-bottom: 4px;
    font-size: 12px;
  }

  .channel-context strong,
  .summary-card strong,
  .group-title strong,
  .channel-name strong,
  .meta-row strong,
  .test-result-row strong {
    min-width: 0;
    overflow: hidden;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .summary-grid {
    display: grid;
    flex: 0 0 auto;
    grid-template-columns: repeat(4, minmax(140px, 1fr)) auto;
    gap: 12px;
    margin-bottom: 12px;
  }

  .summary-card {
    min-width: 0;
    padding: 14px;
  }

  .summary-card span {
    display: block;
    margin-bottom: 8px;
    font-size: 13px;
  }

  .summary-card strong {
    display: block;
    font-size: 24px;
    line-height: 1;
  }

  .summary-action {
    display: flex;
    align-items: center;
    padding: 0 14px;
  }

  .channel-scroll {
    flex: 1;
    min-height: 0;
    padding-right: 4px;
    overflow-y: auto;
    scrollbar-gutter: stable;
  }

  .group-list {
    display: grid;
    gap: 12px;
  }

  .group-section {
    padding: 14px;
    overflow: hidden;
  }

  .group-head {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
  }

  .group-title,
  .channel-identity {
    display: flex;
    gap: 10px;
    align-items: center;
    min-width: 0;
  }

  .group-icon,
  .channel-icon {
    display: flex;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .group-icon {
    width: 40px;
    height: 40px;
    font-size: 22px;
  }

  .channel-icon {
    width: 46px;
    height: 46px;
    font-size: 26px;
  }

  .group-title > div:last-child,
  .channel-name {
    min-width: 0;
  }

  .group-title strong,
  .group-title span,
  .channel-name strong,
  .channel-name span {
    display: block;
  }

  .group-title span,
  .channel-name span {
    margin-top: 4px;
    overflow: hidden;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .channel-card-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: 12px;
  }

  .channel-card {
    min-width: 0;
    overflow: hidden;
    border-radius: 8px;
  }

  .channel-card-head {
    display: flex;
    gap: 10px;
    align-items: flex-start;
    justify-content: space-between;
    min-height: 76px;
    padding: 14px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .channel-card-body {
    display: grid;
    gap: 10px;
    min-height: 168px;
    padding: 14px;
  }

  .meta-row {
    display: grid;
    grid-template-columns: 48px minmax(0, 1fr);
    gap: 8px;
    align-items: center;
    min-width: 0;
    font-size: 13px;
  }

  .meta-row strong {
    display: block;
    font-weight: 500;
  }

  .channel-card-footer {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
    padding: 10px 14px;
    background: var(--el-fill-color-extra-light);
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .form-tip {
    margin-top: 6px;
    overflow: hidden;
    font-size: 12px;
    line-height: 18px;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .select-empty {
    padding: 10px 0;
    font-size: 13px;
    color: var(--el-text-color-secondary);
    text-align: center;
  }

  .test-result {
    padding: 14px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .test-result.is-success {
    background: var(--el-color-success-light-9);
    border-color: var(--el-color-success-light-5);
  }

  .test-result.is-error {
    background: var(--el-color-danger-light-9);
    border-color: var(--el-color-danger-light-5);
  }

  .test-result-head {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 12px;
  }

  .test-result-head strong,
  .test-result-head span {
    display: block;
  }

  .test-result-head span {
    max-width: 460px;
    margin-top: 6px;
    overflow: hidden;
    color: var(--el-text-color-regular);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .test-result-grid {
    display: grid;
    gap: 8px;
  }

  .test-result-row {
    display: grid;
    grid-template-columns: 90px minmax(0, 1fr);
    gap: 12px;
    align-items: center;
    padding: 10px 12px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
  }

  @media (width <= 1100px) {
    .summary-grid {
      grid-template-columns: repeat(2, minmax(140px, 1fr));
    }

    .summary-action {
      min-height: 58px;
    }
  }

  @media (width <= 760px) {
    .channel-context,
    .group-head,
    .channel-card-head {
      flex-direction: column;
      align-items: flex-start;
    }

    .summary-grid,
    .channel-card-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
