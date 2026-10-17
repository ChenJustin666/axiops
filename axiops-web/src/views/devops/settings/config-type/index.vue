<template>
  <div class="devops-config-type-page art-full-height">
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
          <ElButton type="primary" @click="openDialog('add')">新增配置类型</ElButton>
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
      :title="dialogType === 'add' ? '新增配置类型' : '编辑配置类型'"
      width="560px"
      align-center
      :close-on-click-modal="false"
      @close="resetForm"
    >
      <ElForm ref="formRef" :model="formData" :rules="rules" label-width="96px">
        <ElFormItem label="上级类型" prop="parentId">
          <ElSelect
            v-model="formData.parentId"
            :disabled="submitLoading"
            clearable
            filterable
            placeholder="不选则为根类型"
            style="width: 100%"
            @visible-change="handleParentVisibleChange"
          >
            <ElOption
              v-for="item in parentTypeOptions"
              :key="item.id"
              :label="`${item.name}（${item.code}）`"
              :value="item.id"
              :disabled="item.id === currentType?.id"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="名称" prop="name">
          <ElInput v-model="formData.name" :disabled="submitLoading" placeholder="请输入名称" />
        </ElFormItem>
        <ElFormItem v-if="dialogType === 'add'" label="编码" prop="code">
          <ElInput v-model="formData.code" :disabled="submitLoading" placeholder="请输入唯一编码" />
        </ElFormItem>
        <ElFormItem v-else label="编码">
          <ElInput :model-value="currentType?.code" disabled />
        </ElFormItem>
        <ElFormItem label="存储类型" prop="storageType">
          <ElSelect
            v-model="formData.storageType"
            :disabled="submitLoading"
            placeholder="请选择存储类型"
            style="width: 100%"
          >
            <ElOption
              v-for="item in storageTypeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="排序" prop="sortOrder">
          <ElInputNumber
            v-model="formData.sortOrder"
            :disabled="submitLoading"
            :min="0"
            style="width: 100%"
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
  import { computed, h, onActivated, onMounted, reactive, ref } from 'vue'
  import { ElButton, ElMessage, ElMessageBox, ElTag } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import { formatTimestamp } from '@/utils/format'
  import {
    createDevopsConfigTypeApi,
    deleteDevopsConfigTypeApi,
    listDevopsConfigTypeApi,
    updateDevopsConfigTypeApi,
    type DevopsConfigType,
    type DevopsConfigTypeForm
  } from '@/api/devops/project'

  defineOptions({ name: 'DevOpsConfigType' })

  const formRef = ref<FormInstance>()
  const searchBarRef = ref()
  const tableData = ref<DevopsConfigType[]>([])
  const parentTypeOptions = ref<DevopsConfigType[]>([])
  const loading = ref(false)
  const parentLoading = ref(false)
  const submitLoading = ref(false)
  const showSearchBar = ref(true)
  const dialogVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const currentType = ref<DevopsConfigType>()
  const searchParams = ref<Record<string, any>>({})

  const pagination = ref({
    current: 1,
    size: 20,
    total: 0
  })

  const statusOptions = [
    { label: '启用', value: 1 },
    { label: '停用', value: 0 }
  ]

  const storageTypeOptions = [
    { label: 'JSON', value: 'json' },
    { label: 'XML', value: 'xml' },
    { label: 'YML', value: 'yml' },
    { label: 'TEXT', value: 'text' },
    { label: 'CONFIG', value: 'config' }
  ]

  const defaultForm = (): DevopsConfigTypeForm => ({
    parentId: '',
    name: '',
    code: '',
    storageType: 'text',
    description: '',
    sortOrder: 0,
    status: 1
  })

  const formData = reactive<DevopsConfigTypeForm>(defaultForm())

  const rules: FormRules = {
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    code: [{ required: true, message: '请输入编码', trigger: 'blur' }],
    storageType: [{ required: true, message: '请选择存储类型', trigger: 'change' }],
    sortOrder: [{ required: true, message: '请输入排序', trigger: 'change' }],
    status: [{ required: true, message: '请选择状态', trigger: 'change' }]
  }

  const searchForm = ref({
    name: undefined,
    code: undefined,
    status: undefined
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

  const parentTypeName = (parentId?: string) => {
    if (!parentId) return '根类型'
    const parent =
      parentTypeOptions.value.find((item) => item.id === parentId) ||
      tableData.value.find((item) => item.id === parentId)
    return parent ? `${parent.name}（${parent.code}）` : '-'
  }

  const storageTypeLabel = (value?: string) => {
    const option = storageTypeOptions.find((item) => item.value === (value || 'text'))
    return option?.label || 'TEXT'
  }

  const storageTagType = (value?: string) => {
    const tagMap: Record<string, 'primary' | 'success' | 'warning' | 'info'> = {
      json: 'success',
      xml: 'primary',
      yml: 'warning',
      text: 'info',
      config: 'primary'
    }
    return tagMap[value || 'text'] || 'info'
  }

  const columns = computed(() => [
    { prop: 'name', label: '名称', minWidth: 180, fixed: 'left' as const },
    { prop: 'code', label: '编码', minWidth: 160, showOverflowTooltip: true },
    {
      prop: 'storageType',
      label: '存储类型',
      width: 120,
      formatter: (row: DevopsConfigType) =>
        h(ElTag, { type: storageTagType(row.storageType), size: 'small' }, () =>
          storageTypeLabel(row.storageType)
        )
    },
    {
      prop: 'parentId',
      label: '上级类型',
      minWidth: 180,
      showOverflowTooltip: true,
      formatter: (row: DevopsConfigType) => parentTypeName(row.parentId)
    },
    {
      prop: 'status',
      label: '状态',
      width: 90,
      formatter: (row: DevopsConfigType) =>
        h(ElTag, { type: row.status === 1 ? 'success' : 'info', size: 'small' }, () =>
          row.status === 1 ? '启用' : '停用'
        )
    },
    { prop: 'sortOrder', label: '排序', width: 90 },
    { prop: 'description', label: '描述', minWidth: 220, showOverflowTooltip: true },
    {
      prop: 'createdBy',
      label: '创建人',
      width: 120,
      showOverflowTooltip: true,
      formatter: (row: DevopsConfigType) => row.createdBy || '-'
    },
    {
      prop: 'updatedBy',
      label: '更新人',
      width: 120,
      showOverflowTooltip: true,
      formatter: (row: DevopsConfigType) => row.updatedBy || '-'
    },
    {
      prop: 'updatedAt',
      label: '更新时间',
      width: 180,
      formatter: (row: DevopsConfigType) => formatTimestamp(row.updatedAt)
    },
    {
      prop: 'operation',
      label: '操作',
      width: 190,
      fixed: 'right' as const,
      formatter: (row: DevopsConfigType) =>
        h('div', { class: 'table-actions' }, [
          h(
            ElButton,
            { link: true, type: 'primary', onClick: () => openDialog('add', row) },
            () => '下级'
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

  const pickSearchParams = () =>
    Object.entries(searchForm.value).reduce(
      (acc, [key, value]) => {
        if (value !== undefined && value !== null && value !== '') acc[key] = value
        return acc
      },
      {} as Record<string, any>
    )

  const normalizeConfigType = (item: DevopsConfigType) => ({
    ...item,
    storageType: item.storageType || (item.code === 'MavenSettings' ? 'xml' : 'text')
  })

  const fetchParentOptions = async () => {
    parentLoading.value = true
    try {
      const res = await listDevopsConfigTypeApi({ page: 1, pageSize: 200, status: -1 })
      parentTypeOptions.value = (res.items || []).map(normalizeConfigType)
    } finally {
      parentLoading.value = false
    }
  }

  const getData = async () => {
    loading.value = true
    try {
      const res = await listDevopsConfigTypeApi({
        page: pagination.value.current,
        pageSize: pagination.value.size,
        ...searchParams.value
      })
      tableData.value = (res.items || []).map(normalizeConfigType)
      pagination.value.total = res.total || 0
      if (!parentLoading.value) void fetchParentOptions()
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

  const openDialog = async (type: 'add' | 'edit', row?: DevopsConfigType) => {
    await fetchParentOptions()
    dialogType.value = type
    currentType.value = type === 'edit' ? row : undefined
    Object.assign(formData, defaultForm(), {
      ...(type === 'edit' && row
        ? {
            parentId: row.parentId,
            name: row.name,
            code: row.code,
            storageType: row.storageType || (row.code === 'MavenSettings' ? 'xml' : 'text'),
            description: row.description,
            sortOrder: row.sortOrder,
            status: row.status
          }
        : { parentId: row?.id || '' })
    })
    dialogVisible.value = true
  }

  const resetForm = () => {
    Object.assign(formData, defaultForm())
    currentType.value = undefined
    formRef.value?.clearValidate()
  }

  const handleSubmit = async () => {
    await formRef.value?.validate()
    submitLoading.value = true
    try {
      if (dialogType.value === 'add') {
        await createDevopsConfigTypeApi(formData)
        ElMessage.success('新增成功')
      } else if (currentType.value) {
        await updateDevopsConfigTypeApi(currentType.value.id, formData)
        ElMessage.success('保存成功')
      }
      dialogVisible.value = false
      await getData()
    } finally {
      submitLoading.value = false
    }
  }

  const handleDelete = async (row: DevopsConfigType) => {
    await ElMessageBox.confirm(`确定删除配置类型"${row.name}"吗？`, '删除确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteDevopsConfigTypeApi(row.id)
    ElMessage.success('删除成功')
    await getData()
  }

  const handleParentVisibleChange = async (visible: boolean) => {
    if (visible) await fetchParentOptions()
  }

  let activatedOnce = false
  onMounted(getData)
  onActivated(() => {
    if (!activatedOnce) {
      activatedOnce = true
      return
    }
    getData()
  })
</script>

<style scoped lang="scss">
  .devops-config-type-page {
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
    white-space: nowrap;
  }
</style>
