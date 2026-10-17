<template>
  <div class="devops-channel-type-page art-full-height">
    <ArtSearchBar
      v-show="showSearchBar"
      ref="searchRef"
      v-model="searchForm"
      :items="searchItems"
      @search="handleSearch"
      @reset="handleReset"
    />

    <ArtCard
      v-model:showSearchBar="showSearchBar"
      :loading="loading"
      :data="tableData"
      :pagination="pagination"
      empty-text="暂无渠道类型"
      @add="openDialog('add')"
      @refresh="refreshAll"
      @pagination:size-change="handleSizeChange"
      @pagination:current-change="handleCurrentChange"
    >
      <template #default="{ item }">
        <div class="type-card">
          <div class="type-card-head">
            <div class="type-icon" :style="{ color: item.iconColor || '#64748b' }">
              <ArtSvgIcon :icon="item.icon || 'ri:apps-2-line'" />
            </div>
            <div class="type-title">
              <div class="type-name-row">
                <span class="type-name" :title="item.name">{{ item.name }}</span>
                <ElTag v-if="item.isSystem" size="small" type="warning">系统</ElTag>
              </div>
              <span class="type-code" :title="item.code">{{ item.code }}</span>
            </div>
            <ElSwitch
              :model-value="item.status === 1"
              :disabled="item.isSystem"
              @change="(val: boolean) => handleStatusChange(item, val)"
            />
          </div>

          <p class="type-desc" :title="item.description || getGroupName(item.groupCode)">
            {{ item.description || getGroupName(item.groupCode) }}
          </p>

          <div class="type-meta">
            <div>
              <span>归属分组</span>
              <strong :title="getGroupName(item.groupCode)">{{
                getGroupName(item.groupCode)
              }}</strong>
            </div>
            <div>
              <span>连接模式</span>
              <strong>{{ item.connectionMode || '-' }}</strong>
            </div>
            <div>
              <span>测试策略</span>
              <strong>{{ item.testStrategy || '-' }}</strong>
            </div>
          </div>

          <div class="credential-type-list">
            <span class="credential-section-label">支持凭据类型</span>
            <div class="credential-tags">
              <ElTag v-for="ct in item.credentialTypes || []" :key="ct" size="small" type="info">
                {{ credentialTypeLabel(ct) }}
              </ElTag>
              <span v-if="!(item.credentialTypes || []).length" class="no-data-tip">-</span>
            </div>
          </div>

          <div class="type-actions">
            <ElButton text type="primary" @click="openDialog('edit', item)">编辑</ElButton>
            <ElButton text type="danger" :disabled="item.isSystem" @click="handleDelete(item)">
              删除
            </ElButton>
          </div>
        </div>
      </template>
    </ArtCard>

    <ElDialog
      v-model="dialogVisible"
      :title="dialogType === 'add' ? '新增渠道类型' : '编辑渠道类型'"
      width="720px"
      align-center
      :close-on-click-modal="false"
      @close="resetForm"
    >
      <ElForm ref="formRef" :model="formData" :rules="rules" label-width="136px">
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="名称" prop="name">
              <ElInput v-model="formData.name" :disabled="submitLoading" placeholder="如 Jenkins" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="编码" prop="code">
              <template #label>
                <span class="form-label-help">
                  编码
                  <ElTooltip
                    content="渠道类型唯一标识，如 jenkins。创建后不可修改。"
                    placement="top"
                  >
                    <ElIcon class="field-help-icon"><QuestionFilled /></ElIcon>
                  </ElTooltip>
                </span>
              </template>
              <ElInput
                v-model="formData.code"
                :disabled="dialogType === 'edit' || submitLoading"
                placeholder="如 jenkins"
              />
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="归属分组" prop="groupCode">
              <ElSelect
                v-model="formData.groupCode"
                :disabled="submitLoading"
                filterable
                placeholder="选择分组"
                style="width: 100%"
                @visible-change="handleGroupSelectVisible"
              >
                <ElOption
                  v-for="item in groupOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="连接模式" prop="connectionMode">
              <ElSelect
                v-model="formData.connectionMode"
                :disabled="submitLoading"
                placeholder="选择模式"
                style="width: 100%"
              >
                <ElOption
                  v-for="item in connectionModeOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="测试策略" prop="testStrategy">
              <ElSelect
                v-model="formData.testStrategy"
                :disabled="submitLoading"
                placeholder="选择策略"
                style="width: 100%"
              >
                <ElOption
                  v-for="item in testStrategyOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="元数据策略" prop="metadataStrategy">
              <ElSelect
                v-model="formData.metadataStrategy"
                :disabled="submitLoading"
                placeholder="选择策略"
                style="width: 100%"
              >
                <ElOption
                  v-for="item in metadataStrategyOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElFormItem label="图标">
          <div class="icon-picker">
            <button
              v-for="opt in iconOptions"
              :key="opt.icon"
              type="button"
              :class="{ active: formData.icon === opt.icon }"
              :style="{ color: opt.color }"
              @click="pickIcon(opt.icon, opt.color)"
            >
              <ArtSvgIcon :icon="opt.icon" />
            </button>
          </div>
        </ElFormItem>

        <ElFormItem label="凭据类型" prop="credentialTypes">
          <ElSelect
            v-model="formData.credentialTypes"
            :disabled="submitLoading"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            placeholder="选择支持的凭据类型"
            style="width: 100%"
          >
            <ElOption
              v-for="item in credentialTypeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </ElSelect>
        </ElFormItem>

        <ElFormItem label="配置 Schema">
          <template #label>
            <span class="form-label-help">
              配置 Schema
              <ElTooltip
                content='JSON 格式，定义渠道实例的额外配置字段。如 {"fields":[{"key":"namespace","label":"命名空间","type":"input"}]}'
                placement="top"
              >
                <ElIcon class="field-help-icon"><QuestionFilled /></ElIcon>
              </ElTooltip>
            </span>
          </template>
          <ElInput
            v-model="formData.configSchema"
            :disabled="submitLoading"
            type="textarea"
            :rows="4"
            placeholder='{"fields":[{"key":"namespace","label":"命名空间","type":"input"}]}'
          />
        </ElFormItem>

        <ElFormItem label="映射字段">
          <template #label>
            <span class="form-label-help">
              映射字段
              <ElTooltip
                content='JSON 格式，定义流水线参数可映射的渠道字段。如 {"fields":[{"field":"config.repository","name":"默认仓库"}]}'
                placement="top"
              >
                <ElIcon class="field-help-icon"><QuestionFilled /></ElIcon>
              </ElTooltip>
            </span>
          </template>
          <ElInput
            v-model="formData.mappingFields"
            :disabled="submitLoading"
            type="textarea"
            :rows="4"
            placeholder='{"fields":[{"field":"config.repository","name":"默认仓库"}]}'
          />
        </ElFormItem>

        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="系统类型" prop="isSystem">
              <ElSwitch
                v-model="formData.isSystem"
                :disabled="dialogType === 'edit' || submitLoading"
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
      </ElForm>
      <template #footer>
        <ElButton :disabled="submitLoading" @click="dialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
          {{ dialogType === 'add' ? '新增' : '保存' }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onActivated, onMounted, reactive, ref } from 'vue'
  import { ElButton, ElMessage, ElMessageBox, ElSwitch, ElTag } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import { QuestionFilled } from '@element-plus/icons-vue'
  import {
    createDevopsChannelTypeApi,
    deleteDevopsChannelTypeApi,
    listDevopsChannelGroupApi,
    listDevopsChannelTypeApi,
    updateDevopsChannelTypeApi,
    type DevopsChannelGroup,
    type DevopsChannelType,
    type DevopsChannelTypeForm
  } from '@/api/devops/channel'

  defineOptions({ name: 'DevopsChannelTypeManage' })

  const loading = ref(false)
  const submitLoading = ref(false)
  const showSearchBar = ref(true)
  const dialogVisible = ref(false)
  const formRef = ref<FormInstance>()
  const dialogType = ref<'add' | 'edit'>('add')
  const currentId = ref('')
  const tableData = ref<DevopsChannelType[]>([])
  const groups = ref<DevopsChannelGroup[]>([])
  const groupNameMap = ref<Record<string, string>>({})
  const searchParams = ref<Record<string, any>>({})

  const searchForm = ref({
    name: undefined as string | undefined,
    code: undefined as string | undefined,
    groupCode: undefined as string | undefined,
    status: undefined as number | undefined
  })

  const pagination = ref({ current: 1, size: 15, total: 0 })

  const defaultForm = (): DevopsChannelTypeForm => ({
    code: '',
    name: '',
    groupCode: '',
    credentialTypes: [],
    configSchema: '',
    testStrategy: 'http',
    icon: 'ri:apps-2-line',
    iconColor: '#64748b',
    connectionMode: 'server',
    metadataStrategy: 'product_version',
    mappingFields: '',
    isSystem: false,
    status: 1
  })

  const formData = reactive<DevopsChannelTypeForm>(defaultForm())

  const rules: FormRules = {
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    code: [{ required: true, message: '请输入编码', trigger: 'blur' }],
    groupCode: [{ required: true, message: '请选择归属分组', trigger: 'change' }]
  }

  const statusOptions = [
    { label: '启用', value: 1 },
    { label: '停用', value: 0 }
  ]

  const connectionModeOptions = [
    { label: '服务端 (Server)', value: 'server' },
    { label: 'API', value: 'api' },
    { label: 'Kubernetes', value: 'kubernetes' },
    { label: '工具 (Tool)', value: 'tool' },
    { label: '协议 (Protocol)', value: 'protocol' },
    { label: '主机 (Host)', value: 'host' },
    { label: '主机组 (Host Group)', value: 'host_group' }
  ]

  const testStrategyOptions = [
    { label: 'HTTP', value: 'http' },
    { label: 'TCP', value: 'tcp' },
    { label: 'Kubernetes', value: 'kubernetes' },
    { label: '工具', value: 'tool' },
    { label: '主机', value: 'host' },
    { label: '主机组', value: 'host_group' },
    { label: '无', value: 'none' }
  ]

  const metadataStrategyOptions = [
    { label: '产品版本', value: 'product_version' },
    { label: '引擎版本', value: 'engine_version' },
    { label: 'API 能力', value: 'api_capability' },
    { label: '基准', value: 'benchmark' },
    { label: '能力', value: 'capability' },
    { label: 'API 版本', value: 'api_version' }
  ]

  const credentialTypeOptions = [
    { label: '用户名密码', value: 'username_password' },
    { label: 'Token', value: 'token' },
    { label: 'SSH 密钥', value: 'ssh_key' },
    { label: 'Kubeconfig', value: 'kubeconfig' },
    { label: '证书', value: 'certificate' },
    { label: 'Secret Text', value: 'secret_text' },
    { label: 'JSON', value: 'json' }
  ]

  const iconOptions = [
    { icon: 'ri:flow-chart', color: '#2563eb' },
    { icon: 'ri:git-repository-line', color: '#f97316' },
    { icon: 'ri:archive-stack-line', color: '#16a34a' },
    { icon: 'simple-icons:docker', color: '#0369a1' },
    { icon: 'ri:shield-check-line', color: '#0f766e' },
    { icon: 'simple-icons:kubernetes', color: '#326ce5' },
    { icon: 'ri:server-line', color: '#475569' },
    { icon: 'ri:rocket-line', color: '#7c3aed' },
    { icon: 'ri:apps-2-line', color: '#64748b' },
    { icon: 'simple-icons:jenkins', color: '#d24939' },
    { icon: 'simple-icons:gitlab', color: '#fc6d26' },
    { icon: 'simple-icons:sonarqube', color: '#4b9fd5' },
    { icon: 'simple-icons:harbor', color: '#0f766e' },
    { icon: 'simple-icons:jfrog', color: '#0f9f6e' },
    { icon: 'simple-icons:github', color: '#24292f' },
    { icon: 'simple-icons:gitee', color: '#c71d23' },
    { icon: 'simple-icons:argo', color: '#ef7b4d' },
    { icon: 'ri:cloud-line', color: '#0891b2' },
    { icon: 'ri:bug-line', color: '#b45309' },
    { icon: 'ri:shield-keyhole-line', color: '#7c3aed' }
  ]

  const groupOptions = computed(() =>
    groups.value
      .filter((g) => g.status === 1)
      .map((g) => ({ label: `${g.name}（${g.code}）`, value: g.code }))
  )

  const searchItems = computed(() => [
    { label: '名称', key: 'name', type: 'input', props: { clearable: true } },
    { label: '编码', key: 'code', type: 'input', props: { clearable: true } },
    {
      label: '分组',
      key: 'groupCode',
      type: 'select',
      props: { clearable: true, options: groupOptions.value }
    },
    {
      label: '状态',
      key: 'status',
      type: 'select',
      props: { clearable: true, options: statusOptions }
    }
  ])

  const getGroupName = (code: string) => groupNameMap.value[code] || code || '-'
  const credentialTypeLabel = (v: string) =>
    credentialTypeOptions.find((o) => o.value === v)?.label || v

  const pickParams = () =>
    Object.entries(searchForm.value).reduce(
      (acc, [key, value]) => {
        if (value !== undefined && value !== null && value !== '') acc[key] = value
        return acc
      },
      {} as Record<string, any>
    )

  const getData = async () => {
    loading.value = true
    try {
      const res = await listDevopsChannelTypeApi({
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

  const getGroups = async () => {
    const res = await listDevopsChannelGroupApi({ page: 1, pageSize: 200, status: 1 })
    groups.value = res.items || []
    groupNameMap.value = {}
    for (const g of res.items || []) {
      if (g?.code) groupNameMap.value[g.code] = g.name
    }
  }

  const handleGroupSelectVisible = async (visible: boolean) => {
    if (visible) await getGroups()
  }

  const refreshAll = async () => {
    await getGroups()
    await getData()
  }

  const handleSearch = async () => {
    searchParams.value = pickParams()
    pagination.value.current = 1
    await getData()
  }

  const handleReset = async () => {
    searchParams.value = {}
    pagination.value.current = 1
    await getData()
  }

  const handleSizeChange = async (size: number) => {
    pagination.value.size = size
    pagination.value.current = 1
    await getData()
  }

  const handleCurrentChange = async (current: number) => {
    pagination.value.current = current
    await getData()
  }

  const openDialog = async (type: 'add' | 'edit', row?: DevopsChannelType) => {
    dialogType.value = type
    currentId.value = row?.id || ''
    if (type === 'edit' && row) {
      Object.assign(formData, {
        code: row.code,
        name: row.name,
        groupCode: row.groupCode,
        credentialTypes: row.credentialTypes || [],
        configSchema: row.configSchema || '',
        testStrategy: row.testStrategy || 'http',
        icon: row.icon || 'ri:apps-2-line',
        iconColor: row.iconColor || '#64748b',
        connectionMode: row.connectionMode || 'server',
        metadataStrategy: row.metadataStrategy || 'product_version',
        mappingFields: row.mappingFields || '',
        isSystem: row.isSystem,
        status: row.status
      })
    } else {
      Object.assign(formData, defaultForm())
    }
    dialogVisible.value = true
  }

  const resetForm = () => {
    currentId.value = ''
    Object.assign(formData, defaultForm())
    formRef.value?.clearValidate()
  }

  const pickIcon = (icon: string, color: string) => {
    formData.icon = icon
    formData.iconColor = color
  }

  const handleSubmit = async () => {
    await formRef.value?.validate()
    submitLoading.value = true
    try {
      formData.configSchema = normalizeJsonText(formData.configSchema)
      formData.mappingFields = normalizeJsonText(formData.mappingFields)
      if (dialogType.value === 'add') {
        await createDevopsChannelTypeApi(formData)
        ElMessage.success('新增成功')
      } else {
        await updateDevopsChannelTypeApi(currentId.value, formData)
        ElMessage.success('保存成功')
      }
      dialogVisible.value = false
      await refreshAll()
    } catch (error) {
      if (error instanceof SyntaxError) {
        ElMessage.error('配置 Schema 或映射字段不是合法 JSON')
        return
      }
      throw error
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

  const handleStatusChange = async (row: DevopsChannelType, val: boolean) => {
    await updateDevopsChannelTypeApi(row.id, { ...row, status: val ? 1 : 0 })
    row.status = val ? 1 : 0
    ElMessage.success('状态更新成功')
    await getData()
  }

  const handleDelete = async (row: DevopsChannelType) => {
    if (row.isSystem) {
      ElMessage.warning('系统类型不能删除')
      return
    }
    await ElMessageBox.confirm(`确定删除类型"${row.name}"吗？`, '删除确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteDevopsChannelTypeApi(row.id)
    ElMessage.success('删除成功')
    await getData()
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
</script>

<style lang="scss" scoped>
  .devops-channel-type-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .type-card {
    display: flex;
    flex-direction: column;
    min-width: 0;
    min-height: 280px;
    padding: 16px;
    overflow: hidden;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .type-card-head {
    display: flex;
    gap: 12px;
    align-items: center;
    min-width: 0;
  }

  .type-icon {
    display: flex;
    flex: 0 0 44px;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    font-size: 26px;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .type-title {
    flex: 1;
    min-width: 0;
  }

  .type-name-row {
    display: flex;
    gap: 8px;
    align-items: center;
    min-width: 0;
  }

  .type-name {
    overflow: hidden;
    font-weight: 600;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .type-code {
    display: block;
    margin-top: 5px;
    overflow: hidden;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .type-desc {
    display: -webkit-box;
    min-height: 40px;
    margin: 14px 0;
    overflow: hidden;
    color: var(--el-text-color-regular);
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }

  .type-meta {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
    margin-bottom: 12px;
  }

  .type-meta div {
    min-width: 0;
    padding: 8px;
    background: var(--el-fill-color-lighter);
    border-radius: 6px;
  }

  .type-meta span {
    display: block;
    margin-bottom: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .type-meta strong {
    display: block;
    overflow: hidden;
    font-weight: 500;
    font-size: 13px;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .credential-type-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-height: 52px;
  }

  .credential-section-label {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .credential-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }

  .no-data-tip {
    font-size: 12px;
    color: var(--el-text-color-placeholder);
  }

  .type-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding-top: 12px;
    margin-top: auto;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .icon-picker {
    display: grid;
    grid-template-columns: repeat(auto-fill, 36px);
    gap: 8px;
    width: 100%;
  }

  .icon-picker button {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    font-size: 21px;
    cursor: pointer;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .icon-picker button.active {
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary);
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
</style>
