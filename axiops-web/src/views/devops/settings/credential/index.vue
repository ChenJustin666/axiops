<template>
  <div class="devops-credential-page art-full-height">
    <div class="credential-context">
      <div>
        <span class="context-label">当前 DevOps 项目</span>
        <strong :title="currentProjectLabel">{{ currentProjectLabel }}</strong>
      </div>
      <ElTag :type="isSuperAdmin ? 'warning' : 'info'" size="small">
        {{ isSuperAdmin ? 'SUPER_ADMIN 可查看和维护所有凭证' : '仅查看当前项目凭证' }}
      </ElTag>
    </div>

    <ArtSearchBar
      v-show="showSearchBar"
      ref="searchBarRef"
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
        @refresh="getData"
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
      :title="dialogType === 'add' ? '新增凭证' : '编辑凭证'"
      width="760px"
      align-center
      :close-on-click-modal="false"
      @close="resetForm"
    >
      <ElForm ref="formRef" :model="formData" :rules="rules" label-width="132px">
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="名称" prop="name">
              <ElInput v-model="formData.name" :disabled="submitLoading" placeholder="请输入名称" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="编码" prop="code">
              <ElInput
                v-model="formData.code"
                :disabled="submitLoading || dialogType === 'edit'"
                placeholder="请输入唯一编码"
              />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="凭证类型" prop="credentialType">
              <ElSelect
                v-model="formData.credentialType"
                :disabled="submitLoading"
                style="width: 100%"
                @change="handleCredentialTypeChange"
              >
                <ElOption
                  v-for="item in credentialTypeOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
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
          <ElCol :span="12">
            <ElFormItem label="凭证级别" prop="scope">
              <ElRadioGroup
                v-model="formData.scope"
                :disabled="submitLoading || dialogType === 'edit' || !isSuperAdmin"
                @change="handleScopeChange"
              >
                <ElRadio value="project">项目级</ElRadio>
                <ElRadio v-if="isSuperAdmin" value="system">系统级</ElRadio>
              </ElRadioGroup>
            </ElFormItem>
          </ElCol>
          <ElCol v-if="formData.scope === 'project'" :span="12">
            <ElFormItem label="所属项目" prop="projectId">
              <ElInput :model-value="formProjectLabel" disabled placeholder="请先在顶部选择项目" />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="渠道" prop="channelGroupCode">
              <ElSelect
                v-model="formData.channelGroupCode"
                :disabled="submitLoading"
                clearable
                filterable
                placeholder="请选择渠道"
                style="width: 100%"
                @change="handleFormChannelGroupChange"
                @visible-change="handleChannelGroupVisibleChange"
              >
                <ElOption
                  v-for="item in channelGroupOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="渠道类型" prop="channelType">
              <ElSelect
                v-model="formData.channelType"
                :disabled="submitLoading || !formData.channelGroupCode"
                clearable
                filterable
                placeholder="请选择渠道类型"
                style="width: 100%"
                @change="handleFormChannelTypeChange"
                @visible-change="handleFormChannelTypeVisibleChange"
              >
                <ElOption
                  v-for="item in formChannelTypeOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElRow v-if="formData.credentialType === 'username_password'" :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="用户名" prop="username">
              <ElInput
                v-model="formData.username"
                :disabled="submitLoading"
                placeholder="请输入用户名"
              />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="密码" prop="password">
              <ElInput
                v-model="formData.password"
                :disabled="submitLoading"
                type="password"
                show-password
                placeholder="请输入密码"
              />
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElFormItem v-if="formData.credentialType === 'token'" label="Token" prop="token">
          <ElInput
            v-model="formData.token"
            :disabled="submitLoading"
            type="textarea"
            :rows="4"
            placeholder="请输入 Token"
          />
        </ElFormItem>

        <template v-if="formData.credentialType === 'ssh_key'">
          <ElFormItem label="用户名" prop="username">
            <ElInput
              v-model="formData.username"
              :disabled="submitLoading"
              placeholder="请输入 SSH 用户名"
            />
          </ElFormItem>
          <ElFormItem label="私钥" prop="privateKey">
            <ElInput
              v-model="formData.privateKey"
              :disabled="submitLoading"
              type="textarea"
              :rows="5"
              placeholder="请输入 SSH 私钥"
            />
          </ElFormItem>
          <ElFormItem label="Passphrase" prop="passphrase">
            <ElInput
              v-model="formData.passphrase"
              :disabled="submitLoading"
              type="password"
              show-password
              placeholder="请输入私钥口令，非必填"
            />
          </ElFormItem>
        </template>

        <ElFormItem
          v-if="formData.credentialType === 'kubeconfig'"
          label="Kubeconfig"
          prop="kubeconfig"
        >
          <ElInput
            v-model="formData.kubeconfig"
            :disabled="submitLoading"
            type="textarea"
            :rows="6"
            placeholder="请输入 kubeconfig"
          />
        </ElFormItem>

        <ElFormItem
          v-if="formData.credentialType === 'secret_text'"
          label="Secret"
          prop="secretText"
        >
          <ElInput
            v-model="formData.secretText"
            :disabled="submitLoading"
            type="textarea"
            :rows="4"
            placeholder="请输入密文"
          />
        </ElFormItem>

        <ElFormItem
          v-if="formData.credentialType === 'certificate'"
          label="证书"
          prop="certificate"
        >
          <ElInput
            v-model="formData.certificate"
            :disabled="submitLoading"
            type="textarea"
            :rows="6"
            placeholder="请输入证书内容"
          />
        </ElFormItem>

        <ElFormItem v-if="formData.credentialType === 'json'" label="JSON" prop="jsonData">
          <ElInput
            v-model="formData.jsonData"
            :disabled="submitLoading"
            type="textarea"
            :rows="6"
            placeholder='{"username":"xxx","password":"xxx"}'
          />
        </ElFormItem>

        <ElFormItem label="描述" prop="description">
          <ElInput
            v-model="formData.description"
            :disabled="submitLoading"
            type="textarea"
            :rows="3"
            placeholder="请输入描述"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton :disabled="submitLoading" @click="dialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
          {{ dialogType === 'add' ? '新增' : '保存' }}
        </ElButton>
      </template>
    </ElDialog>

    <ElDrawer v-model="usageVisible" title="查询引用" size="560px">
      <div v-loading="usageLoading" class="usage-drawer">
        <div class="usage-summary">
          <strong :title="currentCredential?.name">{{ currentCredential?.name || '-' }}</strong>
          <span :title="currentCredential?.code">{{ currentCredential?.code || '' }}</span>
        </div>
        <ElTable :data="usageData" border height="calc(100vh - 220px)">
          <ElTableColumn prop="resourceType" label="资源类型" width="120" />
          <ElTableColumn
            prop="resourceName"
            label="资源名称"
            min-width="150"
            show-overflow-tooltip
          />
          <ElTableColumn
            prop="resourceCode"
            label="资源编码"
            min-width="150"
            show-overflow-tooltip
          />
          <ElTableColumn prop="relation" label="引用关系" width="120" show-overflow-tooltip />
        </ElTable>
        <ElEmpty v-if="!usageLoading && usageData.length === 0" description="暂无引用资源" />
      </div>
    </ElDrawer>
  </div>
</template>

<script setup lang="ts">
  import { computed, h, onActivated, onMounted, reactive, ref, watch } from 'vue'
  import { storeToRefs } from 'pinia'
  import { ElButton, ElMessage, ElMessageBox, ElTag } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import { formatTimestamp } from '@/utils/format'
  import { useDevopsProjectStore } from '@/store/modules/devopsProject'
  import { useUserStore } from '@/store/modules/user'
  import {
    createDevopsCredentialApi,
    deleteDevopsCredentialApi,
    getDevopsCredentialUsageApi,
    listDevopsChannelGroupApi,
    listDevopsChannelTypeApi,
    listDevopsCredentialApi,
    updateDevopsCredentialApi,
    type DevopsChannelGroup,
    type DevopsChannelType,
    type DevopsCredential,
    type DevopsCredentialForm,
    type DevopsCredentialUsage
  } from '@/api/devops/channel'

  defineOptions({ name: 'DevopsCredentialManagement' })

  const userStore = useUserStore()
  const devopsProjectStore = useDevopsProjectStore()
  const { selectedProject, selectedProjectId, projects } = storeToRefs(devopsProjectStore)

  const loading = ref(false)
  const submitLoading = ref(false)
  const usageLoading = ref(false)
  const showSearchBar = ref(true)
  const searchBarRef = ref()
  const formRef = ref<FormInstance>()
  const tableData = ref<DevopsCredential[]>([])
  const dialogVisible = ref(false)
  const usageVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const currentId = ref('')
  const currentCredential = ref<DevopsCredential | null>(null)
  const usageData = ref<DevopsCredentialUsage[]>([])
  const channelGroups = ref<DevopsChannelGroup[]>([])
  const channelTypes = ref<DevopsChannelType[]>([])
  const searchParams = ref<Record<string, any>>({})

  const searchForm = ref({
    name: undefined,
    code: undefined,
    credentialType: undefined,
    channelGroupCode: undefined,
    channelType: undefined,
    scope: undefined,
    status: undefined
  })

  const pagination = ref({
    current: 1,
    size: 20,
    total: 0
  })

  const credentialTypeOptions = [
    { label: '用户名密码', value: 'username_password' },
    { label: 'Token', value: 'token' },
    { label: 'SSH Key', value: 'ssh_key' },
    { label: 'Kubeconfig', value: 'kubeconfig' },
    { label: 'Secret Text', value: 'secret_text' },
    { label: '证书', value: 'certificate' },
    { label: 'JSON', value: 'json' }
  ]

  const statusOptions = [
    { label: '启用', value: 1 },
    { label: '停用', value: 0 }
  ]

  const scopeOptions = [
    { label: '系统级', value: 'system' },
    { label: '项目级', value: 'project' }
  ]

  const isSuperAdmin = computed(() =>
    (userStore.getUserInfo.roles || []).some((role) => String(role).toUpperCase() === 'SUPER_ADMIN')
  )

  const currentProjectLabel = computed(() => {
    if (!selectedProject.value) return '未选择项目'
    return `${selectedProject.value.name}（${selectedProject.value.code}）`
  })

  const projectNameById = computed(() =>
    projects.value.reduce(
      (acc, item) => {
        acc[item.id] = `${item.name}（${item.code}）`
        return acc
      },
      {} as Record<string, string>
    )
  )

  const channelGroupOptions = computed(() =>
    channelGroups.value
      .filter((item) => item.status === 1)
      .map((item) => ({ label: `${item.name}（${item.code}）`, value: item.code }))
  )

  const searchChannelTypeOptions = computed(() =>
    channelTypes.value
      .filter((item) => item.status === 1)
      .filter((item) => item.groupCode === searchForm.value.channelGroupCode)
      .map((item) => ({ label: `${item.name}（${item.code}）`, value: item.code }))
  )

  const formChannelTypeOptions = computed(() =>
    channelTypes.value
      .filter((item) => item.status === 1)
      .filter((item) => !formData.channelGroupCode || item.groupCode === formData.channelGroupCode)
      .map((item) => ({ label: `${item.name}（${item.code}）`, value: item.code }))
  )

  const channelGroupLabel = (code?: string) =>
    channelGroups.value.find((item) => item.code === code)?.name || code || '-'

  const channelTypeLabel = (code?: string, groupCode?: string) =>
    channelTypes.value.find(
      (item) => item.code === code && (!groupCode || item.groupCode === groupCode)
    )?.name ||
    code ||
    '-'

  const defaultForm = (): DevopsCredentialForm => ({
    name: '',
    code: '',
    credentialType: 'username_password',
    username: '',
    password: '',
    token: '',
    privateKey: '',
    passphrase: '',
    kubeconfig: '',
    secretText: '',
    certificate: '',
    jsonData: '',
    description: '',
    status: 1,
    scope: 'project',
    projectId: '',
    channelGroupCode: '',
    channelType: ''
  })

  const formData = reactive<DevopsCredentialForm>(defaultForm())

  const formProjectLabel = computed(() =>
    formData.projectId ? projectNameById.value[formData.projectId] || formData.projectId : ''
  )

  const rules: FormRules = {
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    code: [{ required: true, message: '请输入编码', trigger: 'blur' }],
    credentialType: [{ required: true, message: '请选择凭证类型', trigger: 'change' }],
    scope: [{ required: true, message: '请选择凭证级别', trigger: 'change' }],
    projectId: [
      {
        validator: (_, value, callback) => {
          if (formData.scope === 'project' && !String(value || '').trim()) {
            callback(new Error('项目级凭证必须选择项目'))
            return
          }
          callback()
        },
        trigger: 'change'
      }
    ],
    username: [
      {
        validator: (_, value, callback) => {
          if (
            ['username_password', 'ssh_key'].includes(formData.credentialType) &&
            !String(value || '').trim()
          ) {
            callback(new Error('请输入用户名'))
            return
          }
          callback()
        },
        trigger: 'blur'
      }
    ],
    password: [
      {
        validator: (_, value, callback) => {
          if (formData.credentialType === 'username_password' && !String(value || '').trim()) {
            callback(new Error('请输入密码'))
            return
          }
          callback()
        },
        trigger: 'blur'
      }
    ],
    token: [
      {
        validator: (_, value, callback) => {
          if (formData.credentialType === 'token' && !String(value || '').trim()) {
            callback(new Error('请输入 Token'))
            return
          }
          callback()
        },
        trigger: 'blur'
      }
    ],
    privateKey: [
      {
        validator: (_, value, callback) => {
          if (formData.credentialType === 'ssh_key' && !String(value || '').trim()) {
            callback(new Error('请输入私钥'))
            return
          }
          callback()
        },
        trigger: 'blur'
      }
    ],
    kubeconfig: [
      {
        validator: (_, value, callback) => {
          if (formData.credentialType === 'kubeconfig' && !String(value || '').trim()) {
            callback(new Error('请输入 kubeconfig'))
            return
          }
          callback()
        },
        trigger: 'blur'
      }
    ],
    secretText: [
      {
        validator: (_, value, callback) => {
          if (formData.credentialType === 'secret_text' && !String(value || '').trim()) {
            callback(new Error('请输入密文'))
            return
          }
          callback()
        },
        trigger: 'blur'
      }
    ],
    certificate: [
      {
        validator: (_, value, callback) => {
          if (formData.credentialType === 'certificate' && !String(value || '').trim()) {
            callback(new Error('请输入证书内容'))
            return
          }
          callback()
        },
        trigger: 'blur'
      }
    ],
    jsonData: [
      {
        validator: (_, value, callback) => {
          if (formData.credentialType !== 'json') {
            callback()
            return
          }
          const text = String(value || '').trim()
          if (!text) {
            callback(new Error('请输入 JSON'))
            return
          }
          try {
            JSON.parse(text)
            callback()
          } catch {
            callback(new Error('请输入合法 JSON'))
          }
        },
        trigger: 'blur'
      }
    ]
  }

  const searchItems = computed(() => [
    { label: '名称', key: 'name', type: 'input', props: { clearable: true } },
    { label: '编码', key: 'code', type: 'input', props: { clearable: true } },
    {
      label: '凭证类型',
      key: 'credentialType',
      type: 'select',
      props: { clearable: true, options: credentialTypeOptions }
    },
    {
      label: '渠道',
      key: 'channelGroupCode',
      type: 'select',
      props: {
        clearable: true,
        options: channelGroupOptions.value,
        onChange: handleSearchChannelGroupChange,
        onVisibleChange: handleChannelGroupVisibleChange
      }
    },
    {
      label: '渠道类型',
      key: 'channelType',
      type: 'select',
      props: {
        clearable: true,
        disabled: !searchForm.value.channelGroupCode,
        options: searchChannelTypeOptions.value,
        onVisibleChange: handleSearchChannelTypeVisibleChange
      }
    },
    ...(isSuperAdmin.value
      ? [
          {
            label: '凭证级别',
            key: 'scope',
            type: 'select',
            props: { clearable: true, options: scopeOptions }
          }
        ]
      : []),
    {
      label: '状态',
      key: 'status',
      type: 'select',
      props: { clearable: true, options: statusOptions }
    }
  ])

  const columns = computed(() => [
    { prop: 'name', label: '名称', minWidth: 160, fixed: 'left' as const },
    { prop: 'code', label: '编码', minWidth: 150, showOverflowTooltip: true },
    {
      prop: 'credentialType',
      label: '凭证类型',
      minWidth: 130,
      formatter: (row: DevopsCredential) => credentialTypeLabel(row.credentialType)
    },
    { prop: 'username', label: '用户名', minWidth: 130, showOverflowTooltip: true },
    {
      prop: 'scope',
      label: '凭证级别',
      width: 120,
      formatter: (row: DevopsCredential) =>
        h(
          ElTag,
          { type: credentialScope(row) === 'system' ? 'warning' : 'success', size: 'small' },
          () => scopeLabel(credentialScope(row))
        )
    },
    {
      prop: 'projectId',
      label: '所属项目',
      minWidth: 170,
      showOverflowTooltip: true,
      formatter: (row: DevopsCredential) =>
        credentialScope(row) === 'system'
          ? '-'
          : projectNameById.value[row.projectId] || row.projectId
    },
    {
      prop: 'channelGroupCode',
      label: '渠道',
      minWidth: 150,
      showOverflowTooltip: true,
      formatter: (row: DevopsCredential) => channelGroupLabel(row.channelGroupCode)
    },
    {
      prop: 'channelType',
      label: '渠道类型',
      minWidth: 150,
      showOverflowTooltip: true,
      formatter: (row: DevopsCredential) => channelTypeLabel(row.channelType, row.channelGroupCode)
    },
    {
      prop: 'status',
      label: '状态',
      width: 90,
      formatter: (row: DevopsCredential) =>
        h(ElTag, { type: row.status === 1 ? 'success' : 'info', size: 'small' }, () =>
          row.status === 1 ? '启用' : '停用'
        )
    },
    {
      prop: 'isSystem',
      label: '系统',
      width: 90,
      formatter: (row: DevopsCredential) =>
        h(ElTag, { type: row.isSystem ? 'warning' : 'info', size: 'small' }, () =>
          row.isSystem ? '是' : '否'
        )
    },
    { prop: 'description', label: '描述', minWidth: 200, showOverflowTooltip: true },
    {
      prop: 'updatedAt',
      label: '更新时间',
      width: 180,
      formatter: (row: DevopsCredential) => formatTimestamp(row.updatedAt)
    },
    {
      prop: 'operation',
      label: '操作',
      width: 220,
      fixed: 'right' as const,
      formatter: (row: DevopsCredential) =>
        h('div', { class: 'table-actions' }, [
          h(
            ElButton,
            { link: true, type: 'success', onClick: () => openUsageDrawer(row) },
            () => '查询引用'
          ),
          h(
            ElButton,
            {
              link: true,
              type: 'primary',
              disabled: !canManageCredential(row),
              onClick: () => openDialog('edit', row)
            },
            () => '编辑'
          ),
          h(
            ElButton,
            {
              link: true,
              type: 'danger',
              disabled: !canManageCredential(row),
              onClick: () => handleDelete(row)
            },
            () => '删除'
          )
        ])
    }
  ])

  const credentialTypeLabel = (type: string) =>
    credentialTypeOptions.find((item) => item.value === type)?.label || type

  const credentialScope = (row: DevopsCredential) => {
    if (row.scope) return row.scope
    return row.isSystem || !row.projectId ? 'system' : 'project'
  }

  const canManageCredential = (row: DevopsCredential) => {
    if (credentialScope(row) === 'system') return isSuperAdmin.value
    return true
  }

  const scopeLabel = (scope: string) => (scope === 'system' ? '系统级' : '项目级')

  const pickSearchParams = () =>
    Object.entries(searchForm.value).reduce(
      (acc, [key, value]) => {
        if (value !== undefined && value !== null && value !== '') acc[key] = value
        return acc
      },
      {} as Record<string, any>
    )

  const buildQueryParams = () => {
    const params = {
      page: pagination.value.current,
      pageSize: pagination.value.size,
      ...searchParams.value
    } as Record<string, any>

    if (!isSuperAdmin.value) {
      params.scope = 'project'
      params.projectId = selectedProjectId.value
      return params
    }

    if (params.scope === 'project' && selectedProjectId.value) {
      params.projectId = selectedProjectId.value
    }
    return params
  }

  const getData = async () => {
    await loadChannelMeta()
    if (!isSuperAdmin.value && !selectedProjectId.value) {
      tableData.value = []
      pagination.value.total = 0
      ElMessage.warning('请选择项目')
      return
    }
    loading.value = true
    try {
      const res = await listDevopsCredentialApi(buildQueryParams())
      tableData.value = res.items || []
      pagination.value.total = res.total || 0
      await loadChannelTypesForCredentials(tableData.value)
    } finally {
      loading.value = false
    }
  }

  const handleSearch = async () => {
    await searchBarRef.value?.validate?.()
    searchParams.value = pickSearchParams()
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

  const loadChannelMeta = async () => {
    const groupRes = await listDevopsChannelGroupApi({ page: 1, pageSize: 200, status: 1 })
    channelGroups.value = groupRes.items || []
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

  const loadChannelTypesForCredentials = async (items: DevopsCredential[]) => {
    const groupCodes = Array.from(
      new Set(items.map((item) => String(item.channelGroupCode || '').trim()).filter(Boolean))
    )
    await Promise.all(groupCodes.map((groupCode) => loadChannelTypesByGroup(groupCode)))
  }

  const openDialog = async (type: 'add' | 'edit', row?: DevopsCredential) => {
    if (row && !canManageCredential(row)) {
      ElMessage.warning('无权维护系统级凭证')
      return
    }
    if (type === 'add' && !isSuperAdmin.value && !selectedProjectId.value) {
      ElMessage.warning('请选择项目')
      return
    }
    dialogType.value = type
    currentId.value = row?.id || ''
    const scope = row ? credentialScope(row) : 'project'
    Object.assign(formData, defaultForm(), row || {}, {
      scope,
      projectId: scope === 'project' ? row?.projectId || selectedProjectId.value : ''
    })
    await loadChannelMeta()
    await loadChannelTypesByGroup(formData.channelGroupCode)
    dialogVisible.value = true
  }

  const handleCredentialTypeChange = () => {
    formRef.value?.clearValidate()
  }

  const handleSearchChannelGroupChange = async () => {
    searchForm.value.channelType = undefined
    await loadChannelTypesByGroup(searchForm.value.channelGroupCode)
  }

  const handleSearchChannelTypeVisibleChange = async (visible: boolean) => {
    if (visible) {
      await loadChannelTypesByGroup(searchForm.value.channelGroupCode)
    }
  }

  const handleChannelGroupVisibleChange = async (visible: boolean) => {
    if (visible) {
      await loadChannelMeta()
    }
  }

  const handleFormChannelTypeVisibleChange = async (visible: boolean) => {
    if (visible) {
      await loadChannelTypesByGroup(formData.channelGroupCode)
    }
  }

  const handleFormChannelGroupChange = async () => {
    formData.channelType = ''
    if (!formData.channelGroupCode) {
      formData.credentialType = 'username_password'
    }
    await loadChannelTypesByGroup(formData.channelGroupCode)
    formRef.value?.clearValidate(['channelGroupCode', 'channelType'])
  }

  const handleFormChannelTypeChange = () => {
    const currentType = channelTypes.value.find(
      (item) => item.code === formData.channelType && item.groupCode === formData.channelGroupCode
    )
    if (
      currentType?.credentialTypes?.length &&
      !currentType.credentialTypes.includes(formData.credentialType)
    ) {
      formData.credentialType = currentType.credentialTypes[0]
    }
    formRef.value?.clearValidate(['channelType'])
  }

  const handleScopeChange = () => {
    formData.projectId = formData.scope === 'project' ? selectedProjectId.value : ''
    formRef.value?.clearValidate(['scope', 'projectId'])
  }

  const resetForm = () => {
    Object.assign(formData, defaultForm())
    currentId.value = ''
    formRef.value?.clearValidate()
  }

  const handleSubmit = async () => {
    formData.channelGroupCode = String(formData.channelGroupCode || '').trim()
    formData.channelType = String(formData.channelType || '').trim()
    if (formData.scope === 'project') {
      if (!selectedProjectId.value && dialogType.value === 'add') {
        ElMessage.warning('请选择项目')
        return
      }
      if (dialogType.value === 'add') {
        formData.projectId = selectedProjectId.value
      }
    } else {
      formData.projectId = ''
    }
    await formRef.value?.validate()
    if (formData.credentialType === 'json') {
      formData.jsonData = normalizeJsonText(formData.jsonData)
    }
    submitLoading.value = true
    try {
      if (dialogType.value === 'add') {
        await createDevopsCredentialApi(formData)
        ElMessage.success('新增成功')
      } else {
        await updateDevopsCredentialApi(currentId.value, formData)
        ElMessage.success('保存成功')
      }
      dialogVisible.value = false
      await getData()
    } finally {
      submitLoading.value = false
    }
  }

  const normalizeJsonText = (content?: string) => {
    const text = String(content || '').trim()
    if (!text) return ''
    JSON.parse(text)
    return text
  }

  const handleDelete = async (row: DevopsCredential) => {
    if (!canManageCredential(row)) {
      ElMessage.warning('无权删除系统级凭证')
      return
    }
    const usage = await getDevopsCredentialUsageApi(row.id)
    if ((usage.total || 0) > 0) {
      ElMessage.warning('凭证正在被资源使用，不能删除')
      openUsageDrawer(row)
      return
    }
    await ElMessageBox.confirm(`确定删除凭证"${row.name}"吗？`, '删除确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteDevopsCredentialApi(row.id)
    ElMessage.success('删除成功')
    await getData()
  }

  const openUsageDrawer = async (row: DevopsCredential) => {
    currentCredential.value = row
    usageVisible.value = true
    usageLoading.value = true
    try {
      const res = await getDevopsCredentialUsageApi(row.id)
      usageData.value = res.items || []
    } finally {
      usageLoading.value = false
    }
  }

  watch(selectedProjectId, async () => {
    if (!isSuperAdmin.value || searchParams.value.scope === 'project' || dialogVisible.value) {
      if (dialogVisible.value && dialogType.value === 'add' && formData.scope === 'project') {
        formData.projectId = selectedProjectId.value
      }
      pagination.value.current = 1
      getData()
    }
  })

  const refreshPage = async () => {
    if (!devopsProjectStore.initialized) {
      await devopsProjectStore.fetchProjects()
    }
    await loadChannelMeta()
    await getData()
  }

  let activatedOnce = false
  onMounted(refreshPage)
  onActivated(() => {
    if (!activatedOnce) {
      activatedOnce = true
      return
    }
    refreshPage()
  })
</script>

<style lang="scss" scoped>
  .devops-credential-page {
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

  .credential-context {
    display: flex;
    flex: 0 0 auto;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    padding: 12px 14px;
    margin-bottom: 12px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .credential-context strong,
  .context-label {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .credential-context strong {
    margin-top: 4px;
    color: var(--el-text-color-primary);
  }

  .context-label {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  :deep(.art-table-card > .el-card__body) {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
  }

  .table-actions {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  :deep(.el-dialog__body) {
    max-height: calc(100vh - 220px);
    overflow-y: auto;
  }

  .usage-drawer {
    display: flex;
    flex-direction: column;
    gap: 12px;
    height: 100%;
    min-height: 0;
  }

  .usage-summary {
    padding: 12px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .usage-summary strong,
  .usage-summary span {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .usage-summary span {
    margin-top: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
</style>
