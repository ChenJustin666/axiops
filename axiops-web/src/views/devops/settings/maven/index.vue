<template>
  <div class="devops-maven-page art-full-height">
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
        empty-height="420px"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      />
    </ElCard>

    <ElDialog
      v-model="dialogVisible"
      :title="dialogType === 'add' ? '新增 Maven 配置' : '编辑 Maven 配置'"
      width="860px"
      align-center
      :close-on-click-modal="false"
      @close="resetForm"
    >
      <ElForm ref="formRef" :model="formData" :rules="rules" label-width="112px">
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
        <ElFormItem label="所属项目">
          <ElInput :model-value="currentProjectLabel" disabled />
        </ElFormItem>
        <ElFormItem class="maven-content-form-item" label="配置内容" prop="content">
          <XmlEditor
            v-model="formData.content"
            filename="settings.xml"
            height="420px"
            :readonly="submitLoading"
            placeholder="请输入 settings.xml 内容"
            @valid-change="xmlValid = $event"
          />
        </ElFormItem>
        <ElFormItem label="状态" prop="status">
          <ElRadioGroup v-model="formData.status" :disabled="submitLoading">
            <ElRadio :value="1">启用</ElRadio>
            <ElRadio :value="0">停用</ElRadio>
          </ElRadioGroup>
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
  </div>
</template>

<script setup lang="ts">
  import { computed, h, onActivated, onMounted, reactive, ref, watch } from 'vue'
  import { storeToRefs } from 'pinia'
  import { ElButton, ElMessage, ElMessageBox, ElTag } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import { formatTimestamp } from '@/utils/format'
  import XmlEditor from './components/XmlEditor.vue'
  import { useDevopsProjectStore } from '@/store/modules/devopsProject'
  import {
    createDevopsProjectMavenConfigApi,
    deleteDevopsProjectMavenConfigApi,
    listDevopsProjectMavenConfigApi,
    updateDevopsProjectMavenConfigApi,
    type DevopsProjectMavenConfig,
    type DevopsProjectMavenConfigForm
  } from '@/api/devops/project'

  defineOptions({ name: 'DevOpsMavenManagement' })

  const devopsProjectStore = useDevopsProjectStore()
  const { selectedProject, selectedProjectId } = storeToRefs(devopsProjectStore)

  const loading = ref(false)
  const submitLoading = ref(false)
  const showSearchBar = ref(true)
  const searchBarRef = ref()
  const formRef = ref<FormInstance>()
  const tableData = ref<DevopsProjectMavenConfig[]>([])
  const dialogVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const currentId = ref('')
  const xmlValid = ref(true)
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

  const currentProjectLabel = computed(() => {
    if (!selectedProject.value) return '未选择项目'
    return `${selectedProject.value.name}（${selectedProject.value.code}）`
  })

  const defaultForm = (): DevopsProjectMavenConfigForm => ({
    projectId: '',
    name: '',
    code: '',
    content: '',
    description: '',
    status: 1
  })

  const formData = reactive<DevopsProjectMavenConfigForm>(defaultForm())

  const rules: FormRules = {
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    code: [{ required: true, message: '请输入编码', trigger: 'blur' }],
    content: [{ required: true, message: '请输入 settings.xml 内容', trigger: 'blur' }],
    status: [{ required: true, message: '请选择状态', trigger: 'change' }]
  }

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
      formatter: (row: DevopsProjectMavenConfig) =>
        h(ElTag, { type: row.status === 1 ? 'success' : 'info', size: 'small' }, () =>
          row.status === 1 ? '启用' : '停用'
        )
    },
    {
      prop: 'content',
      label: '配置内容',
      minWidth: 280,
      showOverflowTooltip: true,
      formatter: (row: DevopsProjectMavenConfig) => formatContentPreview(row.content)
    },
    { prop: 'description', label: '描述', minWidth: 220, showOverflowTooltip: true },
    {
      prop: 'createdBy',
      label: '创建人',
      width: 120,
      showOverflowTooltip: true,
      formatter: (row: DevopsProjectMavenConfig) => row.createdBy || '-'
    },
    {
      prop: 'updatedBy',
      label: '更新人',
      width: 120,
      showOverflowTooltip: true,
      formatter: (row: DevopsProjectMavenConfig) => row.updatedBy || '-'
    },
    {
      prop: 'createdAt',
      label: '创建时间',
      width: 180,
      formatter: (row: DevopsProjectMavenConfig) => formatTimestamp(row.createdAt)
    },
    {
      prop: 'updatedAt',
      label: '更新时间',
      width: 180,
      formatter: (row: DevopsProjectMavenConfig) => formatTimestamp(row.updatedAt)
    },
    {
      prop: 'operation',
      label: '操作',
      width: 150,
      fixed: 'right' as const,
      formatter: (row: DevopsProjectMavenConfig) =>
        h('div', { class: 'table-actions' }, [
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

  const pickSearchParams = () =>
    Object.entries(searchForm.value).reduce(
      (acc, [key, value]) => {
        if (value !== undefined && value !== null && value !== '') acc[key] = value
        return acc
      },
      {} as Record<string, any>
    )

  const getData = async () => {
    if (!selectedProjectId.value) {
      tableData.value = []
      pagination.value.total = 0
      ElMessage.warning('请选择项目')
      return
    }
    loading.value = true
    try {
      const res = await listDevopsProjectMavenConfigApi({
        page: pagination.value.current,
        pageSize: pagination.value.size,
        projectId: selectedProjectId.value,
        ...searchParams.value
      })
      tableData.value = res.items || []
      pagination.value.total = res.total || 0
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

  const openDialog = (type: 'add' | 'edit', row?: DevopsProjectMavenConfig) => {
    if (!selectedProjectId.value) {
      ElMessage.warning('请选择项目')
      return
    }
    dialogType.value = type
    currentId.value = row?.id || ''
    xmlValid.value = true
    Object.assign(formData, defaultForm(), row || {}, { projectId: selectedProjectId.value })
    dialogVisible.value = true
  }

  const resetForm = () => {
    Object.assign(formData, defaultForm())
    currentId.value = ''
    xmlValid.value = true
    formRef.value?.clearValidate()
  }

  const handleSubmit = async () => {
    if (!selectedProjectId.value) {
      ElMessage.warning('请选择项目')
      return
    }
    formData.projectId = selectedProjectId.value
    await formRef.value?.validate()
    if (!xmlValid.value) {
      ElMessage.warning('请先修正 settings.xml XML 语法')
      return
    }
    submitLoading.value = true
    try {
      if (dialogType.value === 'add') {
        await createDevopsProjectMavenConfigApi(formData)
        ElMessage.success('新增成功')
      } else {
        await updateDevopsProjectMavenConfigApi(currentId.value, formData)
        ElMessage.success('保存成功')
      }
      dialogVisible.value = false
      await getData()
    } finally {
      submitLoading.value = false
    }
  }

  const handleDelete = async (row: DevopsProjectMavenConfig) => {
    await ElMessageBox.confirm(`确定删除 Maven 配置"${row.name}"吗？`, '删除确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteDevopsProjectMavenConfigApi(row.id)
    ElMessage.success('删除成功')
    await getData()
  }

  watch(selectedProjectId, () => {
    if (dialogVisible.value && dialogType.value === 'add') {
      formData.projectId = selectedProjectId.value
    }
    pagination.value.current = 1
    getData()
  })

  const refreshPage = async () => {
    if (!devopsProjectStore.initialized) {
      await devopsProjectStore.fetchProjects()
    }
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

<style scoped lang="scss">
  .devops-maven-page {
    height: 100%;
    min-height: 0;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .art-table-card {
    flex: 1;
    min-height: 520px;
    overflow: hidden;
    display: flex;
    flex-direction: column;

    :deep(.el-card__body) {
      flex: 1;
      min-height: 0;
      display: flex;
      flex-direction: column;
    }

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

  :deep(.maven-content-form-item .el-form-item__content) {
    min-width: 0;
  }
</style>
