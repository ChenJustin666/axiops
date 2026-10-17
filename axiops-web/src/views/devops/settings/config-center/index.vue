<template>
  <div class="devops-config-center-page art-full-height">
    <div class="config-center-layout">
      <ElCard class="type-tree-card" shadow="never">
        <div class="type-tree-header">
          <div>
            <strong>配置类型</strong>
            <span>{{ configTypes.length }} 个类型</span>
          </div>
          <div class="type-tree-actions">
            <ElButton size="small" :loading="typeLoading" @click="fetchTypeTree">刷新</ElButton>
          </div>
        </div>
        <div class="type-tree-body">
          <ElEmpty v-if="!typeLoading && !typeTree.length" description="暂无配置类型" />
          <ElTree
            v-else
            ref="typeTreeRef"
            class="type-tree"
            :data="typeTree"
            node-key="id"
            :props="{ label: 'name', children: 'children' }"
            :expand-on-click-node="false"
            :current-node-key="selectedTypeId"
            default-expand-all
            highlight-current
            @node-click="handleTypeSelect"
          >
            <template #default="{ data }">
              <div class="type-node">
                <span class="type-node-main">
                  <span class="type-node-name" :title="data.name">{{ data.name }}</span>
                  <span class="type-node-code" :title="data.code">{{ data.code }}</span>
                </span>
              </div>
            </template>
          </ElTree>
        </div>
      </ElCard>

      <div class="config-table-area">
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
              <ElButton type="primary" :disabled="!selectedType" @click="openConfigDialog('add')">
                新增配置
              </ElButton>
            </template>
          </ArtTableHeader>

          <ArtTable
            :loading="loading"
            :data="tableData"
            :columns="columns"
            :pagination="pagination"
            empty-height="420px"
            @pagination:size-change="handleSizeChange"
            @pagination:current-change="handleCurrentChange"
          />
        </ElCard>
      </div>
    </div>

    <ElDialog
      v-model="configDialogVisible"
      :title="configDialogType === 'add' ? '新增配置' : '编辑配置'"
      class="config-center-dialog"
      width="960px"
      align-center
      :close-on-click-modal="false"
      @close="resetConfigForm"
    >
      <ElForm
        ref="configFormRef"
        class="config-form"
        :model="configForm"
        :rules="configRules"
        label-position="top"
      >
        <div class="config-meta-panel">
          <ElRow :gutter="16">
            <ElCol :span="12">
              <ElFormItem label="名称" prop="name">
                <ElInput
                  v-model="configForm.name"
                  :disabled="submitLoading"
                  placeholder="请输入名称"
                />
              </ElFormItem>
            </ElCol>
            <ElCol :span="12">
              <ElFormItem label="编码" prop="code">
                <ElInput
                  v-model="configForm.code"
                  :disabled="submitLoading || configDialogType === 'edit'"
                  placeholder="请输入唯一编码"
                />
              </ElFormItem>
            </ElCol>
          </ElRow>
          <ElRow :gutter="16">
            <ElCol :span="12">
              <ElFormItem label="所属项目">
                <ElInput :model-value="currentProjectLabel" disabled />
              </ElFormItem>
            </ElCol>
            <ElCol :span="12">
              <ElFormItem label="配置类型">
                <ElInput :model-value="selectedTypeLabel" disabled />
              </ElFormItem>
            </ElCol>
          </ElRow>
          <ElRow :gutter="16">
            <ElCol :span="8">
              <ElFormItem label="状态" prop="status">
                <ElRadioGroup v-model="configForm.status" :disabled="submitLoading">
                  <ElRadio :value="1">启用</ElRadio>
                  <ElRadio :value="0">停用</ElRadio>
                </ElRadioGroup>
              </ElFormItem>
            </ElCol>
            <ElCol :span="16">
              <ElFormItem label="描述" prop="description">
                <ElInput
                  v-model="configForm.description"
                  :disabled="submitLoading"
                  type="textarea"
                  :rows="2"
                  resize="none"
                  placeholder="请输入描述"
                />
              </ElFormItem>
            </ElCol>
          </ElRow>
        </div>
        <ElFormItem class="config-content-form-item" label="配置内容" prop="content">
          <div class="config-editor-wrap">
            <XmlEditor
              v-if="isXmlConfigType"
              v-model="configForm.content"
              :filename="configEditorFilename"
              height="430px"
              theme="dark"
              :readonly="submitLoading"
              placeholder="请输入 XML 配置内容"
              @valid-change="contentValid = $event"
            />
            <ElInput
              v-else
              v-model="configForm.content"
              class="config-dark-textarea"
              :disabled="submitLoading"
              type="textarea"
              :rows="18"
              resize="none"
              placeholder="请输入配置内容"
            />
          </div>
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton :disabled="submitLoading" @click="configDialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="submitLoading" @click="handleConfigSubmit">
          {{ configDialogType === 'add' ? '新增' : '保存' }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, h, nextTick, onActivated, onMounted, reactive, ref, watch } from 'vue'
  import { storeToRefs } from 'pinia'
  import { ElButton, ElMessage, ElMessageBox, ElTag } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import { formatTimestamp } from '@/utils/format'
  import { useDevopsProjectStore } from '@/store/modules/devopsProject'
  import XmlEditor from '../maven/components/XmlEditor.vue'
  import {
    createDevopsProjectConfigApi,
    deleteDevopsProjectConfigApi,
    listDevopsProjectConfigApi,
    treeDevopsConfigTypeApi,
    updateDevopsProjectConfigApi,
    type DevopsConfigType,
    type DevopsProjectConfig,
    type DevopsProjectConfigForm
  } from '@/api/devops/project'

  defineOptions({ name: 'DevOpsConfigCenter' })

  const defaultMavenSettingsTypeCode = 'MavenSettings'
  const devopsProjectStore = useDevopsProjectStore()
  const { selectedProject, selectedProjectId } = storeToRefs(devopsProjectStore)

  const typeTreeRef = ref()
  const configFormRef = ref<FormInstance>()
  const searchBarRef = ref()
  const typeTree = ref<DevopsConfigType[]>([])
  const selectedTypeId = ref('')
  const typeLoading = ref(false)
  const loading = ref(false)
  const submitLoading = ref(false)
  const showSearchBar = ref(true)
  const configDialogVisible = ref(false)
  const configDialogType = ref<'add' | 'edit'>('add')
  const currentConfigId = ref('')
  const contentValid = ref(true)
  const tableData = ref<DevopsProjectConfig[]>([])
  const searchParams = ref<Record<string, any>>({})

  const searchForm = ref({
    name: undefined,
    code: undefined,
    status: undefined
  })

  const pagination = ref({
    current: 1,
    size: 20,
    total: 0
  })

  const statusOptions = [
    { label: '启用', value: 1 },
    { label: '停用', value: 0 }
  ]

  const defaultConfigForm = (): DevopsProjectConfigForm => ({
    projectId: '',
    typeId: '',
    typeCode: '',
    name: '',
    code: '',
    content: '',
    description: '',
    status: 1
  })

  const configForm = reactive<DevopsProjectConfigForm>(defaultConfigForm())

  const configRules: FormRules = {
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    code: [{ required: true, message: '请输入编码', trigger: 'blur' }],
    content: [{ required: true, message: '请输入配置内容', trigger: 'blur' }],
    status: [{ required: true, message: '请选择状态', trigger: 'change' }]
  }

  const configTypes = computed(() => flattenTypes(typeTree.value))
  const selectedType = computed(() =>
    configTypes.value.find((item) => item.id === selectedTypeId.value)
  )
  const isMavenSettingsType = computed(
    () => selectedType.value?.code === defaultMavenSettingsTypeCode
  )
  const configStorageType = computed(() => {
    if (isMavenSettingsType.value) return 'xml'
    return selectedType.value?.storageType || 'text'
  })
  const isXmlConfigType = computed(() => configStorageType.value === 'xml')
  const configEditorFilename = computed(() => {
    if (isMavenSettingsType.value) return 'settings.xml'
    if (configStorageType.value === 'xml') return 'config.xml'
    return `config.${configStorageType.value}`
  })
  const currentProjectLabel = computed(() => {
    if (!selectedProject.value) return '未选择项目'
    return `${selectedProject.value.name}（${selectedProject.value.code}）`
  })
  const selectedTypeLabel = computed(() => {
    if (!selectedType.value) return '未选择类型'
    return `${selectedType.value.name}（${selectedType.value.code}）`
  })

  const searchItems = computed(() => [
    { label: '名称', key: 'name', type: 'input', props: { clearable: true } },
    { label: '编码', key: 'code', type: 'input', props: { clearable: true } },
    {
      label: '状态',
      key: 'status',
      type: 'select',
      props: { clearable: true, options: statusOptions }
    }
  ])

  const formatContentPreview = (content?: string) => {
    const text = content?.replace(/\s+/g, ' ').trim()
    return text || '-'
  }

  const columns = computed(() => [
    { prop: 'name', label: '名称', minWidth: 160, fixed: 'left' as const },
    { prop: 'code', label: '编码', minWidth: 150, showOverflowTooltip: true },
    {
      prop: 'status',
      label: '状态',
      width: 90,
      formatter: (row: DevopsProjectConfig) =>
        h(ElTag, { type: row.status === 1 ? 'success' : 'info', size: 'small' }, () =>
          row.status === 1 ? '启用' : '停用'
        )
    },
    {
      prop: 'content',
      label: '配置内容',
      minWidth: 280,
      showOverflowTooltip: true,
      formatter: (row: DevopsProjectConfig) => formatContentPreview(row.content)
    },
    { prop: 'description', label: '描述', minWidth: 220, showOverflowTooltip: true },
    {
      prop: 'createdBy',
      label: '创建人',
      width: 120,
      showOverflowTooltip: true,
      formatter: (row: DevopsProjectConfig) => row.createdBy || '-'
    },
    {
      prop: 'updatedBy',
      label: '更新人',
      width: 120,
      showOverflowTooltip: true,
      formatter: (row: DevopsProjectConfig) => row.updatedBy || '-'
    },
    {
      prop: 'createdAt',
      label: '创建时间',
      width: 180,
      formatter: (row: DevopsProjectConfig) => formatTimestamp(row.createdAt)
    },
    {
      prop: 'updatedAt',
      label: '更新时间',
      width: 180,
      formatter: (row: DevopsProjectConfig) => formatTimestamp(row.updatedAt)
    },
    {
      prop: 'operation',
      label: '操作',
      width: 150,
      fixed: 'right' as const,
      formatter: (row: DevopsProjectConfig) =>
        h('div', { class: 'table-actions' }, [
          h(
            ElButton,
            { link: true, type: 'primary', onClick: () => openConfigDialog('edit', row) },
            () => '编辑'
          ),
          h(
            ElButton,
            { link: true, type: 'danger', onClick: () => handleConfigDelete(row) },
            () => '删除'
          )
        ])
    }
  ])

  const flattenTypes = (items: DevopsConfigType[]) =>
    items.reduce<DevopsConfigType[]>((acc, item) => {
      acc.push(item)
      if (item.children?.length) acc.push(...flattenTypes(item.children))
      return acc
    }, [])

  const pickSearchParams = () =>
    Object.entries(searchForm.value).reduce(
      (acc, [key, value]) => {
        if (value !== undefined && value !== null && value !== '') acc[key] = value
        return acc
      },
      {} as Record<string, any>
    )

  const fetchTypeTree = async () => {
    typeLoading.value = true
    try {
      const res = await treeDevopsConfigTypeApi({ status: -1 })
      typeTree.value = res.items || []
      const allTypes = flattenTypes(typeTree.value)
      if (!selectedTypeId.value || !allTypes.some((item) => item.id === selectedTypeId.value)) {
        selectedTypeId.value = allTypes[0]?.id || ''
      }
      await nextTick()
      if (selectedTypeId.value) typeTreeRef.value?.setCurrentKey?.(selectedTypeId.value)
      await getData()
    } finally {
      typeLoading.value = false
    }
  }

  const getData = async () => {
    if (!selectedProjectId.value || !selectedType.value) {
      tableData.value = []
      pagination.value.total = 0
      return
    }
    loading.value = true
    try {
      const res = await listDevopsProjectConfigApi({
        page: pagination.value.current,
        pageSize: pagination.value.size,
        projectId: selectedProjectId.value,
        typeId: selectedType.value.id,
        ...searchParams.value
      })
      tableData.value = res.items || []
      pagination.value.total = res.total || 0
    } finally {
      loading.value = false
    }
  }

  const handleTypeSelect = (data: DevopsConfigType) => {
    selectedTypeId.value = data.id
    pagination.value.current = 1
    getData()
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

  const openConfigDialog = (type: 'add' | 'edit', row?: DevopsProjectConfig) => {
    if (!selectedProjectId.value) {
      ElMessage.warning('请选择项目')
      return
    }
    if (!selectedType.value) {
      ElMessage.warning('请选择配置类型')
      return
    }
    configDialogType.value = type
    currentConfigId.value = row?.id || ''
    contentValid.value = true
    Object.assign(configForm, defaultConfigForm(), row || {}, {
      projectId: selectedProjectId.value,
      typeId: selectedType.value.id,
      typeCode: selectedType.value.code
    })
    configDialogVisible.value = true
  }

  const resetConfigForm = () => {
    Object.assign(configForm, defaultConfigForm())
    currentConfigId.value = ''
    contentValid.value = true
    configFormRef.value?.clearValidate()
  }

  const handleConfigSubmit = async () => {
    if (!selectedProjectId.value || !selectedType.value) {
      ElMessage.warning('请选择项目和配置类型')
      return
    }
    configForm.projectId = selectedProjectId.value
    configForm.typeId = selectedType.value.id
    configForm.typeCode = selectedType.value.code
    await configFormRef.value?.validate()
    if (isXmlConfigType.value && !contentValid.value) {
      ElMessage.warning('请先修正 XML 语法')
      return
    }
    submitLoading.value = true
    try {
      if (configDialogType.value === 'add') {
        await createDevopsProjectConfigApi(configForm)
        ElMessage.success('新增成功')
      } else {
        await updateDevopsProjectConfigApi(currentConfigId.value, configForm)
        ElMessage.success('保存成功')
      }
      configDialogVisible.value = false
      await getData()
    } finally {
      submitLoading.value = false
    }
  }

  const handleConfigDelete = async (row: DevopsProjectConfig) => {
    await ElMessageBox.confirm(`确定删除配置"${row.name}"吗？`, '删除确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteDevopsProjectConfigApi(row.id)
    ElMessage.success('删除成功')
    await getData()
  }

  const refreshPage = async () => {
    if (!devopsProjectStore.initialized) {
      await devopsProjectStore.fetchProjects()
    }
    await fetchTypeTree()
  }

  watch(selectedProjectId, () => {
    if (configDialogVisible.value && configDialogType.value === 'add') {
      configForm.projectId = selectedProjectId.value
    }
    pagination.value.current = 1
    getData()
  })

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

<style scoped lang="scss">
  .devops-config-center-page {
    height: 100%;
    min-height: 0;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .config-center-layout {
    flex: 1;
    min-height: 0;
    display: grid;
    grid-template-columns: 300px minmax(0, 1fr);
    gap: 12px;
  }

  .type-tree-card,
  .art-table-card {
    min-height: 0;
    overflow: hidden;
    display: flex;
    flex-direction: column;

    :deep(.el-card__body) {
      flex: 1;
      min-height: 0;
      display: flex;
      flex-direction: column;
    }
  }

  .type-tree-card {
    border-color: #e6ebf2;
    border-radius: 8px;

    :deep(.el-card__body) {
      padding: 0;
    }
  }

  .type-tree-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 16px;
    background: linear-gradient(180deg, #ffffff 0%, #f8fbff 100%);
    border-bottom: 1px solid var(--el-border-color-light);

    strong {
      display: block;
      font-size: 15px;
      line-height: 20px;
      color: var(--el-text-color-primary);
    }

    span {
      display: block;
      margin-top: 2px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .type-tree-actions {
    display: inline-flex;
    gap: 8px;
    flex-shrink: 0;
  }

  .type-tree-body {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 8px 10px 12px;
    background: #fbfcff;
  }

  .type-tree {
    background: transparent;

    :deep(.el-tree-node__content) {
      position: relative;
      height: 46px;
      padding-right: 8px;
      margin: 2px 0;
      border-radius: 7px;
      transition:
        background-color 0.18s ease,
        color 0.18s ease;

      &:hover {
        background: #f1f6ff;
      }
    }

    :deep(.el-tree-node.is-current > .el-tree-node__content) {
      color: #2f6bff;
      background: #edf4ff;

      &::before {
        position: absolute;
        top: 9px;
        bottom: 9px;
        left: 0;
        width: 3px;
        content: '';
        background: #4f7cff;
        border-radius: 999px;
      }
    }
  }

  .type-node {
    width: 100%;
    min-width: 0;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  .type-node-main {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
    line-height: 18px;
  }

  .type-node-name {
    font-size: 13px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .type-node-name,
  .type-node-code {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .type-node-code {
    font-size: 12px;
    color: #8a94a6;
  }

  .config-table-area {
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  .art-table-card {
    flex: 1;
    min-height: 520px;

    :deep(.art-table) {
      flex: 1;
      min-height: 420px;
    }

    :deep(.el-table) {
      min-height: 360px;
    }
  }

  .table-actions {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }

  :deep(.config-content-form-item .el-form-item__content) {
    width: 100%;
    min-width: 0;
  }

  .config-form {
    display: flex;
    flex-direction: column;
    gap: 14px;

    :deep(.el-form-item) {
      margin-bottom: 14px;
    }

    :deep(.el-form-item__label) {
      margin-bottom: 6px;
      font-size: 13px;
      font-weight: 600;
      color: #2f3747;
    }
  }

  .config-meta-panel {
    padding: 14px 16px 0;
    background: #f8fafc;
    border: 1px solid #e6ebf2;
    border-radius: 8px;
  }

  .config-editor-wrap {
    width: 100%;
    min-width: 0;
  }

  .config-dark-textarea {
    height: 430px;

    :deep(.el-textarea__inner) {
      height: 430px;
      min-height: 430px !important;
      max-height: 430px;
      padding: 12px 14px;
      overflow: auto;
      font-family: var(--ax-font-mono);
      font-size: 13px;
      line-height: 21px;
      color: #e5e7eb;
      caret-color: #60a5fa;
      white-space: pre;
      background: #111827;
      border-color: #27364d;
      box-shadow: none;

      &::placeholder {
        color: #64748b;
      }

      &:focus {
        border-color: #3b82f6;
        box-shadow: 0 0 0 1px rgb(59 130 246 / 28%);
      }
    }
  }

  :global(.config-center-dialog) {
    border-radius: 10px;
  }

  :global(.config-center-dialog .el-dialog__header) {
    padding: 18px 22px 10px;
    margin-right: 0;
  }

  :global(.config-center-dialog .el-dialog__body) {
    max-height: calc(100vh - 190px);
    padding: 8px 22px 10px;
    overflow: auto;
  }

  :global(.config-center-dialog .el-dialog__footer) {
    padding: 12px 22px 18px;
    border-top: 1px solid #edf1f7;
  }
</style>
