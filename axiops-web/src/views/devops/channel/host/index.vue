<template>
  <div class="devops-host-page art-full-height">
    <ArtSearchBar
      v-show="showSearchBar"
      ref="searchRef"
      v-model="searchForm"
      :items="searchItems"
      @search="handleSearch"
      @reset="handleReset"
    />

    <ElCard
      class="art-table-card"
      shadow="never"
      :style="{ 'margin-top': showSearchBar ? '12px' : '0' }"
    >
      <ArtTableHeader
        v-model:columns="columns"
        v-model:showSearchBar="showSearchBar"
        :loading="loading"
        :showZebra="true"
        :showBorder="true"
        :showHeaderBackground="true"
        :fullClass="'art-page-view'"
        :layout="'search,refresh,size,fullscreen,columns,settings'"
        @refresh="refreshAll"
      >
        <template #left>
          <ElButton type="primary" @click="openDialog('add')">新增</ElButton>
        </template>
      </ArtTableHeader>

      <ArtTable
        :loading="loading"
        :data="tableData"
        :columns="columns"
        :pagination="pagination"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      />
    </ElCard>

    <ElDialog
      v-model="dialogVisible"
      :title="dialogType === 'add' ? '新增主机' : '编辑主机'"
      width="760px"
      align-center
      class="host-form-dialog"
      :close-on-click-modal="false"
      @close="resetForm"
    >
      <ElForm ref="formRef" :model="formData" :rules="rules" label-width="124px">
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="名称" prop="name">
              <template #label>
                <span class="form-label-help">
                  名称
                  <ElTooltip :content="formHelpText('name')" placement="top">
                    <ElIcon class="field-help-icon"><QuestionFilled /></ElIcon>
                  </ElTooltip>
                </span>
              </template>
              <ElInput v-model="formData.name" :disabled="submitLoading" placeholder="请输入名称" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="IP 地址" prop="ip">
              <template #label>
                <span class="form-label-help">
                  IP 地址
                  <ElTooltip :content="formHelpText('ip')" placement="top">
                    <ElIcon class="field-help-icon"><QuestionFilled /></ElIcon>
                  </ElTooltip>
                </span>
              </template>
              <ElInput
                v-model="formData.ip"
                :disabled="submitLoading"
                placeholder="请输入 IP 地址"
              />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="端口" prop="port">
              <template #label>
                <span class="form-label-help">
                  端口
                  <ElTooltip :content="formHelpText('port')" placement="top">
                    <ElIcon class="field-help-icon"><QuestionFilled /></ElIcon>
                  </ElTooltip>
                </span>
              </template>
              <ElInputNumber
                v-model="formData.port"
                :disabled="submitLoading"
                :min="1"
                :max="65535"
                style="width: 100%"
              />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="状态" prop="status">
              <ElRadioGroup v-model="formData.status" :disabled="submitLoading">
                <ElRadio :value="1">启用</ElRadio>
                <ElRadio :value="0">停用</ElRadio>
              </ElRadioGroup>
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="16">
            <ElFormItem label="登录凭据" prop="credentialId">
              <template #label>
                <span class="form-label-help">
                  登录凭据
                  <ElTooltip :content="formHelpText('credentialId')" placement="top">
                    <ElIcon class="field-help-icon"><QuestionFilled /></ElIcon>
                  </ElTooltip>
                </span>
              </template>
              <ElSelect
                v-model="formData.credentialId"
                :disabled="submitLoading"
                clearable
                filterable
                placeholder="请选择主机凭据"
                style="width: 100%"
                @visible-change="handleCredentialVisibleChange"
              >
                <ElOption
                  v-for="item in hostCredentialOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
                <template #empty>
                  <div class="credential-empty">
                    <span>暂无主机凭据</span>
                    <ElButton link type="primary" @click="openCredentialDialog">新增凭据</ElButton>
                  </div>
                </template>
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="8">
            <ElButton class="credential-add-btn" :icon="Plus" @click="openCredentialDialog">
              新增凭据
            </ElButton>
          </ElCol>
        </ElRow>
        <div
          v-if="inlineTestResult"
          :class="['host-test-result', inlineTestResult.success ? 'is-success' : 'is-error']"
        >
          <div class="test-result-head">
            <div>
              <strong>{{ inlineTestResult.success ? '连接正常' : '连接异常' }}</strong>
              <p :title="inlineTestResult.message">{{ inlineTestResult.message }}</p>
            </div>
            <ElTag :type="inlineTestResult.success ? 'success' : 'danger'">
              {{ inlineTestResult.success ? '正常' : '异常' }}
            </ElTag>
          </div>
          <div class="test-result-grid">
            <div>
              <span>认证用户</span>
              <strong :title="inlineTestResult.authUser || '-'">
                {{ inlineTestResult.authUser || '-' }}
              </strong>
            </div>
            <div>
              <span>系统信息</span>
              <strong :title="inlineTestResult.system || '-'">
                {{ inlineTestResult.system || '-' }}
              </strong>
            </div>
            <div>
              <span>检测时间</span>
              <strong :title="formatTimestamp(inlineTestResult.checkedAt)">
                {{ formatTimestamp(inlineTestResult.checkedAt) }}
              </strong>
            </div>
          </div>
        </div>
        <ElFormItem label="标签" prop="labels">
          <template #label>
            <span class="form-label-help">
              标签
              <ElTooltip :content="formHelpText('labels')" placement="top">
                <ElIcon class="field-help-icon"><QuestionFilled /></ElIcon>
              </ElTooltip>
            </span>
          </template>
          <ElInput
            v-model="formData.labels"
            :disabled="submitLoading"
            type="textarea"
            :rows="3"
            placeholder="请输入 JSON 标签"
          />
        </ElFormItem>
        <ElFormItem label="描述" prop="description">
          <ElInput
            v-model="formData.description"
            :disabled="submitLoading"
            type="textarea"
            :rows="3"
            placeholder="请输入主机描述"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton :disabled="submitLoading" @click="dialogVisible = false">取消</ElButton>
        <ElButton :loading="testConfigLoading" :disabled="submitLoading" @click="handleTestConfig">
          测试连接
        </ElButton>
        <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
          {{ dialogType === 'add' ? '新增' : '保存' }}
        </ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="testResultDialogVisible"
      title="连接测试结果"
      width="560px"
      align-center
      class="host-test-dialog"
    >
      <div
        v-if="standaloneTestResult"
        :class="['host-test-result', standaloneTestResult.success ? 'is-success' : 'is-error']"
      >
        <div class="test-result-head">
          <div>
            <strong>{{ standaloneTestResult.success ? '连接正常' : '连接异常' }}</strong>
            <p :title="standaloneTestResult.message">{{ standaloneTestResult.message }}</p>
          </div>
          <ElTag :type="standaloneTestResult.success ? 'success' : 'danger'">
            {{ standaloneTestResult.success ? '正常' : '异常' }}
          </ElTag>
        </div>
        <div class="test-result-grid">
          <div>
            <span>认证用户</span>
            <strong :title="standaloneTestResult.authUser || '-'">
              {{ standaloneTestResult.authUser || '-' }}
            </strong>
          </div>
          <div>
            <span>系统信息</span>
            <strong :title="standaloneTestResult.system || '-'">
              {{ standaloneTestResult.system || '-' }}
            </strong>
          </div>
          <div>
            <span>检测时间</span>
            <strong :title="formatTimestamp(standaloneTestResult.checkedAt)">
              {{ formatTimestamp(standaloneTestResult.checkedAt) }}
            </strong>
          </div>
        </div>
      </div>
      <template #footer>
        <ElButton type="primary" @click="testResultDialogVisible = false">确定</ElButton>
      </template>
    </ElDialog>

    <CredentialCreateDialog
      v-model="credentialDialogVisible"
      title="新增主机凭据"
      label-width="96px"
      :type-options="hostCredentialTypeOptions"
      default-type="username_password"
      @created="handleCredentialCreated"
    />
  </div>
</template>

<script setup lang="ts">
  import { computed, h, onActivated, onMounted, reactive, ref, watch } from 'vue'
  import { storeToRefs } from 'pinia'
  import { ElButton, ElMessage, ElMessageBox, ElSwitch, ElTag } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import { Plus, QuestionFilled } from '@element-plus/icons-vue'
  import { formatTimestamp } from '@/utils/format'
  import { useDevopsProjectStore } from '@/store/modules/devopsProject'
  import { useUserStore } from '@/store/modules/user'
  import {
    createDevopsHostApi,
    deleteDevopsHostApi,
    listDevopsCredentialApi,
    listDevopsHostApi,
    testDevopsHostConfigApi,
    testDevopsHostApi,
    updateDevopsHostApi,
    type DevopsCredential,
    type DevopsHost,
    type DevopsHostForm
  } from '@/api/devops/channel'
  import CredentialCreateDialog from '@/views/devops/components/CredentialCreateDialog.vue'

  defineOptions({ name: 'DevopsHostAsset' })

  interface HostTestResultView {
    success: boolean
    message: string
    checkedAt: number
    authUser?: string
    system?: string
  }

  const loading = ref(false)
  const submitLoading = ref(false)
  const testConfigLoading = ref(false)
  const testLoadingId = ref('')
  const showSearchBar = ref(true)
  const searchRef = ref()
  const formRef = ref<FormInstance>()
  const dialogVisible = ref(false)
  const credentialDialogVisible = ref(false)
  const testResultDialogVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const currentId = ref('')
  const tableData = ref<DevopsHost[]>([])
  const credentials = ref<DevopsCredential[]>([])
  const searchParams = ref<Record<string, any>>({})
  const inlineTestResult = ref<HostTestResultView | null>(null)
  const standaloneTestResult = ref<HostTestResultView | null>(null)
  const userStore = useUserStore()
  const devopsProjectStore = useDevopsProjectStore()
  const { selectedProjectId } = storeToRefs(devopsProjectStore)

  const searchForm = ref({
    name: undefined,
    ip: undefined,
    status: undefined
  })

  const pagination = ref({
    current: 1,
    size: 20,
    total: 0
  })

  const defaultForm = (): DevopsHostForm => ({
    name: '',
    ip: '',
    port: 22,
    credentialId: '',
    labels: '',
    description: '',
    status: 1
  })

  const formData = reactive<DevopsHostForm>(defaultForm())

  const rules: FormRules = {
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    ip: [{ required: true, message: '请输入 IP 地址', trigger: 'blur' }],
    port: [{ required: true, message: '请输入端口', trigger: 'change' }]
  }

  const statusOptions = [
    { label: '启用', value: 1 },
    { label: '停用', value: 0 }
  ]

  const credentialTypeLabel = (type: string) => {
    const labels: Record<string, string> = {
      username_password: '用户名密码',
      ssh_key: 'SSH Key'
    }
    return labels[type] || type
  }

  const hostCredentialTypeOptions = [
    { label: '用户名密码', value: 'username_password' },
    { label: 'SSH Key', value: 'ssh_key' }
  ]

  const isSuperAdmin = computed(() =>
    (userStore.getUserInfo.roles || []).some((role) => String(role).toUpperCase() === 'SUPER_ADMIN')
  )

  const hostCredentialOptions = computed(() =>
    credentials.value
      .filter((item) => item.status === 1)
      .filter((item) => ['username_password', 'ssh_key'].includes(item.credentialType))
      .map((item) => ({
        label: `${item.name}（${credentialTypeLabel(item.credentialType)}）`,
        value: item.id
      }))
  )

  const credentialName = (id: string) => {
    const credential = credentials.value.find((item) => item.id === id)
    return credential
      ? `${credential.name}（${credentialTypeLabel(credential.credentialType)}）`
      : '无凭据'
  }

  const searchItems = computed(() => [
    { label: '名称', key: 'name', type: 'input', props: { clearable: true } },
    { label: 'IP 地址', key: 'ip', type: 'input', props: { clearable: true } },
    {
      label: '状态',
      key: 'status',
      type: 'select',
      props: { clearable: true, options: statusOptions }
    }
  ])

  const healthTagType = (status: string) => {
    if (status === 'healthy') return 'success'
    if (status === 'unhealthy') return 'danger'
    return 'info'
  }

  const formHelpText = (key: string) => {
    const texts: Record<string, string> = {
      name: '主机资产名称，用于主机组选择和渠道实例绑定时识别。',
      ip: '主机 SSH 或 TCP 连接地址，支持内网 IP 或可解析域名。',
      port: '主机连接端口，SSH 默认 22。',
      credentialId: '主机登录凭据，支持用户名密码或 SSH Key，可在当前弹窗内新增。',
      labels: 'JSON 标签，用于环境、用途和主机组筛选，例如 {"env":"dev"}。'
    }
    return texts[key] || ''
  }

  const columns = computed(() => [
    { prop: 'name', label: '名称', minWidth: 150, fixed: 'left' as const },
    { prop: 'ip', label: 'IP 地址', minWidth: 150, showOverflowTooltip: true },
    { prop: 'port', label: '端口', width: 90 },
    {
      prop: 'credentialId',
      label: '凭据',
      minWidth: 180,
      formatter: (row: DevopsHost) => credentialName(row.credentialId)
    },
    {
      prop: 'healthStatus',
      label: '连接性',
      width: 110,
      formatter: (row: DevopsHost) =>
        h(ElTag, { type: healthTagType(row.healthStatus) }, () => row.healthStatus || 'unknown')
    },
    {
      prop: 'status',
      label: '状态',
      width: 100,
      formatter: (row: DevopsHost) =>
        h(ElSwitch, {
          modelValue: row.status === 1,
          onUpdateModelValue: (val: boolean) => handleStatusChange(row, val)
        })
    },
    {
      prop: 'lastCheckAt',
      label: '最后检测',
      width: 180,
      formatter: (row: DevopsHost) => (row.lastCheckAt ? formatTimestamp(row.lastCheckAt) : '-')
    },
    { prop: 'lastCheckMessage', label: '检测结果', minWidth: 220, showOverflowTooltip: true },
    {
      prop: 'operation',
      label: '操作',
      width: 210,
      fixed: 'right' as const,
      formatter: (row: DevopsHost) =>
        h('div', { class: 'table-actions' }, [
          h(
            ElButton,
            {
              link: true,
              type: 'success',
              loading: testLoadingId.value === row.id,
              onClick: () => handleTest(row)
            },
            () => '测试'
          ),
          h(
            ElButton,
            { link: true, type: 'primary', onClick: () => openDialog('edit', row) },
            () => '编辑'
          ),
          h(
            ElButton,
            { link: true, type: 'danger', onClick: () => handleDelete(row) },
            () => '删除'
          )
        ])
    }
  ])

  const pickParams = () =>
    Object.entries(searchForm.value).reduce(
      (acc, [key, value]) => {
        if (value !== undefined && value !== null && value !== '') acc[key] = value
        return acc
      },
      {} as Record<string, any>
    )

  const getCredentials = async () => {
    const baseParams = { page: 1, pageSize: 200, status: -1 }
    if (!isSuperAdmin.value && !selectedProjectId.value) {
      credentials.value = []
      return
    }
    const requests = isSuperAdmin.value
      ? [
          listDevopsCredentialApi({ ...baseParams, scope: 'system' }),
          ...(selectedProjectId.value
            ? [
                listDevopsCredentialApi({
                  ...baseParams,
                  scope: 'project',
                  projectId: selectedProjectId.value
                })
              ]
            : [])
        ]
      : [
          listDevopsCredentialApi({
            ...baseParams,
            scope: 'project',
            projectId: selectedProjectId.value
          })
        ]
    const results = await Promise.all(requests)
    const merged = new Map<string, DevopsCredential>()
    results.forEach((res) => {
      ;(res.items || []).forEach((item) => merged.set(item.id, item))
    })
    credentials.value = Array.from(merged.values())
    if (formData.credentialId && !merged.has(formData.credentialId)) {
      formData.credentialId = ''
    }
  }

  const getData = async () => {
    loading.value = true
    try {
      const res = await listDevopsHostApi({
        page: pagination.value.current,
        pageSize: pagination.value.size,
        ...searchParams.value
      })
      tableData.value = res.items || []
      pagination.value.total = res.total || 0
    } finally {
      loading.value = false
    }
  }

  const refreshAll = async () => {
    if (!devopsProjectStore.initialized) {
      await devopsProjectStore.fetchProjects()
    }
    await getCredentials()
    await getData()
  }

  const handleSearch = async () => {
    await searchRef.value?.validate?.()
    searchParams.value = pickParams()
    pagination.value.current = 1
    getData()
  }

  const handleReset = () => {
    searchParams.value = {}
    pagination.value.current = 1
    getData()
  }

  const handleSizeChange = (size: number) => {
    pagination.value.size = size
    pagination.value.current = 1
    getData()
  }

  const handleCurrentChange = (current: number) => {
    pagination.value.current = current
    getData()
  }

  const openDialog = async (type: 'add' | 'edit', row?: DevopsHost) => {
    dialogType.value = type
    currentId.value = row?.id || ''
    inlineTestResult.value = null
    Object.assign(formData, defaultForm(), row || {})
    await getCredentials()
    dialogVisible.value = true
  }

  const resetForm = () => {
    currentId.value = ''
    inlineTestResult.value = null
    Object.assign(formData, defaultForm())
    formRef.value?.clearValidate()
  }

  const handleSubmit = async () => {
    await formRef.value?.validate()
    submitLoading.value = true
    try {
      if (dialogType.value === 'add') {
        await createDevopsHostApi(formData)
        ElMessage.success('新增成功')
      } else {
        await updateDevopsHostApi(currentId.value, formData)
        ElMessage.success('保存成功')
      }
      dialogVisible.value = false
      await getData()
    } finally {
      submitLoading.value = false
    }
  }

  const handleTestConfig = async () => {
    await formRef.value?.validate()
    testConfigLoading.value = true
    try {
      const result = await testDevopsHostConfigApi({
        id: currentId.value || undefined,
        name: formData.name,
        ip: formData.ip,
        port: formData.port,
        credentialId: formData.credentialId
      })
      inlineTestResult.value = normalizeTestResult(result)
    } finally {
      testConfigLoading.value = false
    }
  }

  const handleStatusChange = async (row: DevopsHost, val: boolean) => {
    await updateDevopsHostApi(row.id, { ...row, status: val ? 1 : 0 })
    row.status = val ? 1 : 0
    ElMessage.success('状态更新成功')
    await getData()
  }

  const handleTest = async (row: DevopsHost) => {
    testLoadingId.value = row.id
    try {
      const result = await testDevopsHostApi(row.id)
      standaloneTestResult.value = normalizeTestResult(result)
      testResultDialogVisible.value = true
      await getData()
    } finally {
      testLoadingId.value = ''
    }
  }

  const normalizeTestResult = (result: {
    success: boolean
    message: string
    checkedAt: number
    metadata?: string
  }): HostTestResultView => {
    const metadata = parseMetadata(result.metadata)
    return {
      success: result.success,
      message: result.message,
      checkedAt: result.checkedAt,
      authUser: metadata.authUser,
      system: metadata.capabilities?.system
    }
  }

  const parseMetadata = (metadata?: string) => {
    if (!metadata) return {} as Record<string, any>
    try {
      return JSON.parse(metadata)
    } catch {
      return {} as Record<string, any>
    }
  }

  const openCredentialDialog = () => {
    if (!isSuperAdmin.value && !selectedProjectId.value) {
      ElMessage.warning('请选择项目')
      return
    }
    credentialDialogVisible.value = true
  }

  const handleCredentialCreated = async (credential: DevopsCredential) => {
    await getCredentials()
    formData.credentialId = credential.id
  }

  const handleCredentialVisibleChange = async (visible: boolean) => {
    if (visible) {
      await getCredentials()
    }
  }

  const handleDelete = async (row: DevopsHost) => {
    await ElMessageBox.confirm(`确定删除主机"${row.name}"吗？`, '删除确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteDevopsHostApi(row.id)
    ElMessage.success('删除成功')
    await getData()
  }

  watch(selectedProjectId, () => {
    getCredentials()
  })

  let activatedOnce = false
  onMounted(refreshAll)
  onActivated(() => {
    if (!activatedOnce) {
      activatedOnce = true
      return
    }
    getCredentials()
  })
</script>

<style lang="scss" scoped>
  .devops-host-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .art-table-card {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
  }

  :deep(.art-table-card > .el-card__body) {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
  }

  .table-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .credential-add-btn {
    width: 100%;
  }

  .credential-empty {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    color: var(--el-text-color-secondary);
  }

  .host-test-result {
    padding: 14px;
    margin-bottom: 18px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    background: var(--el-fill-color-extra-light);
  }

  .host-test-result.is-success {
    border-color: var(--el-color-success-light-5);
    background: var(--el-color-success-light-9);
  }

  .host-test-result.is-error {
    border-color: var(--el-color-danger-light-5);
    background: var(--el-color-danger-light-9);
  }

  .test-result-head {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 12px;
  }

  .test-result-head strong {
    font-size: 16px;
    color: var(--el-text-color-primary);
  }

  .test-result-head p {
    max-width: 360px;
    margin: 6px 0 0;
    overflow: hidden;
    line-height: 20px;
    color: var(--el-text-color-regular);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .test-result-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: 8px;
  }

  .test-result-grid div {
    display: grid;
    grid-template-columns: 86px minmax(0, 1fr);
    gap: 12px;
    align-items: center;
    min-width: 0;
    padding: 9px 12px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
  }

  .test-result-grid span {
    display: block;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .test-result-grid strong {
    display: block;
    overflow: hidden;
    font-weight: 500;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .form-label-help {
    display: inline-flex;
    flex-wrap: nowrap;
    gap: 4px;
    align-items: center;
    min-width: 0;
    white-space: nowrap;
  }

  .field-help-icon {
    flex: 0 0 auto;
    font-size: 14px;
    color: var(--el-text-color-secondary);
    cursor: help;
  }

  :deep(.el-dialog__body) {
    max-height: calc(100vh - 220px);
    overflow-y: auto;
  }

  @media (max-width: 768px) {
    .test-result-grid div {
      grid-template-columns: 72px minmax(0, 1fr);
    }
  }
</style>
