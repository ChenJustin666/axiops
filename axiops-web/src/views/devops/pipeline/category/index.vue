<template>
  <div class="devops-step-category-page">
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
      empty-text="暂无步骤分类"
      @add="openDialog('add')"
      @refresh="fetchList"
      @pagination:size-change="handleSizeChange"
      @pagination:current-change="handleCurrentChange"
    >
      <template #default="{ item }">
        <div class="category-card">
          <div class="card-head">
            <div class="category-icon" :style="{ color: item.iconColor || '#64748b' }">
              <ArtSvgIcon :icon="item.icon || 'ri:folder-2-line'" />
            </div>
            <div class="title-wrap">
              <span class="title" :title="item.name">{{ item.name }}</span>
              <span class="code" :title="item.code">{{ item.code }}</span>
            </div>
            <ElTag size="small" :type="item.status === 1 ? 'success' : 'info'">
              {{ item.status === 1 ? '启用' : '停用' }}
            </ElTag>
          </div>
          <p class="desc" :title="item.description || '暂无描述'">{{
            item.description || '暂无描述'
          }}</p>
          <div class="meta-row">
            <span>排序</span>
            <strong>{{ item.sortOrder }}</strong>
          </div>
          <div class="card-actions">
            <ElButton text type="primary" @click="openDialog('edit', item)">配置</ElButton>
            <ElButton text type="danger" @click="handleDelete(item)">删除</ElButton>
          </div>
        </div>
      </template>
    </ArtCard>

    <ElDialog
      v-model="dialogVisible"
      :title="dialogType === 'add' ? '新增步骤分类' : '配置步骤分类'"
      width="620px"
      align-center
      :close-on-click-modal="false"
      @close="resetForm"
    >
      <ElForm ref="formRef" :model="formData" :rules="rules" label-width="110px">
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="名称" prop="name">
              <ElInput v-model="formData.name" placeholder="请输入名称" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="编码" prop="code">
              <ElInput
                v-model="formData.code"
                :disabled="dialogType === 'edit'"
                placeholder="例如 build"
              />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="排序" prop="sortOrder">
              <ElInputNumber v-model="formData.sortOrder" :min="0" style="width: 100%" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="状态" prop="status">
              <ElRadioGroup v-model="formData.status">
                <ElRadio :value="1">启用</ElRadio>
                <ElRadio :value="0">停用</ElRadio>
              </ElRadioGroup>
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElFormItem label="分类图标">
          <div class="icon-picker">
            <button
              v-for="item in iconOptions"
              :key="item.icon"
              type="button"
              :class="{ active: formData.icon === item.icon }"
              :style="{ color: item.color }"
              @click="pickIcon(item.icon, item.color)"
            >
              <ArtSvgIcon :icon="item.icon" />
            </button>
          </div>
        </ElFormItem>
        <ElFormItem label="描述" prop="description">
          <ElInput
            v-model="formData.description"
            type="textarea"
            :rows="3"
            placeholder="请输入分类描述"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="dialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
          {{ dialogType === 'add' ? '新增' : '保存' }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onActivated, onMounted, reactive, ref } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import {
    createDevopsStepCategoryApi,
    deleteDevopsStepCategoryApi,
    listDevopsStepCategoryApi,
    updateDevopsStepCategoryApi,
    type DevopsStepCategory,
    type DevopsStepCategoryForm
  } from '@/api/devops/pipeline'

  defineOptions({ name: 'DevopsPipelineCategory' })

  const loading = ref(false)
  const submitLoading = ref(false)
  const showSearchBar = ref(true)
  const searchRef = ref()
  const formRef = ref<FormInstance>()
  const dialogVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const currentId = ref('')
  const tableData = ref<DevopsStepCategory[]>([])
  const searchParams = ref<Record<string, any>>({})

  const searchForm = ref({ name: undefined, code: undefined, status: undefined })
  const pagination = ref({ current: 1, size: 15, total: 0 })

  const iconOptions = [
    { icon: 'ri:git-repository-line', color: '#f97316' },
    { icon: 'ri:hammer-line', color: '#2563eb' },
    { icon: 'simple-icons:docker', color: '#0ea5e9' },
    { icon: 'ri:scan-line', color: '#4b9fd5' },
    { icon: 'ri:rocket-line', color: '#7c3aed' },
    { icon: 'ri:user-follow-line', color: '#d97706' },
    { icon: 'ri:tools-line', color: '#64748b' }
  ]

  const defaultForm = (): DevopsStepCategoryForm => ({
    name: '',
    code: '',
    description: '',
    icon: 'ri:folder-2-line',
    iconColor: '#64748b',
    sortOrder: 0,
    status: 1
  })

  const formData = reactive<DevopsStepCategoryForm>(defaultForm())
  const rules: FormRules = {
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    code: [{ required: true, message: '请输入编码', trigger: 'blur' }]
  }

  const searchItems = computed(() => [
    { label: '名称', key: 'name', type: 'input', props: { clearable: true } },
    { label: '编码', key: 'code', type: 'input', props: { clearable: true } },
    {
      label: '状态',
      key: 'status',
      type: 'select',
      props: {
        clearable: true,
        options: [
          { label: '启用', value: 1 },
          { label: '停用', value: 0 }
        ]
      }
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

  const fetchList = async () => {
    loading.value = true
    try {
      const res = await listDevopsStepCategoryApi({
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

  const handleSearch = () => {
    searchParams.value = pickParams()
    pagination.value.current = 1
    fetchList()
  }

  const handleReset = () => {
    searchParams.value = {}
    pagination.value.current = 1
    fetchList()
  }

  const handleSizeChange = (size: number) => {
    pagination.value.size = size
    pagination.value.current = 1
    fetchList()
  }

  const handleCurrentChange = (page: number) => {
    pagination.value.current = page
    fetchList()
  }

  const pickIcon = (icon: string, color: string) => {
    formData.icon = icon
    formData.iconColor = color
  }

  const openDialog = (type: 'add' | 'edit', row?: DevopsStepCategory) => {
    dialogType.value = type
    currentId.value = row?.id || ''
    Object.assign(formData, defaultForm(), row || {})
    dialogVisible.value = true
  }

  const resetForm = () => {
    formRef.value?.clearValidate()
    Object.assign(formData, defaultForm())
  }

  const handleSubmit = async () => {
    await formRef.value?.validate()
    submitLoading.value = true
    try {
      if (dialogType.value === 'add') {
        await createDevopsStepCategoryApi(formData)
        ElMessage.success('新增成功')
      } else {
        await updateDevopsStepCategoryApi(currentId.value, formData)
        ElMessage.success('保存成功')
      }
      dialogVisible.value = false
      await fetchList()
    } finally {
      submitLoading.value = false
    }
  }

  const handleDelete = async (row: DevopsStepCategory) => {
    await ElMessageBox.confirm(`确认删除步骤分类 ${row.name}？`, '删除确认', { type: 'warning' })
    await deleteDevopsStepCategoryApi(row.id)
    ElMessage.success('删除成功')
    await fetchList()
  }

  let activatedOnce = false
  onMounted(fetchList)
  onActivated(() => {
    if (!activatedOnce) {
      activatedOnce = true
      return
    }
    fetchList()
  })
</script>

<style scoped lang="scss">
  .devops-step-category-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .devops-step-category-page :deep(.summary-list),
  .devops-step-category-page :deep(.summary-item) {
    display: none !important;
  }

  .category-card {
    display: flex;
    flex-direction: column;
    min-height: 178px;
    overflow: hidden;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    box-shadow: 0 8px 22px rgb(15 23 42 / 4%);

    &:hover {
      border-color: var(--el-color-primary-light-5);
      box-shadow: 0 12px 28px rgb(15 23 42 / 8%);
    }
  }

  .card-head {
    display: grid;
    grid-template-columns: 42px minmax(0, 1fr) auto;
    gap: 10px;
    align-items: center;
    padding: 12px 14px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .category-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 42px;
    height: 42px;
    font-size: 24px;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .title-wrap {
    min-width: 0;
  }

  .title,
  .code,
  .desc {
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .title {
    display: block;
    font-size: 16px;
    font-weight: 700;
    white-space: nowrap;
  }

  .code {
    display: block;
    margin-top: 4px;
    color: var(--el-text-color-secondary);
    white-space: nowrap;
  }

  .desc {
    display: -webkit-box;
    flex: 1;
    padding: 12px 14px;
    margin: 0;
    line-height: 1.6;
    color: var(--el-text-color-regular);
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }

  .meta-row {
    display: flex;
    justify-content: space-between;
    padding: 0 14px 10px;
    color: var(--el-text-color-secondary);
  }

  .card-actions {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    padding: 10px 14px;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .icon-picker {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;

    button {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 38px;
      height: 38px;
      font-size: 22px;
      cursor: pointer;
      background: var(--el-fill-color-light);
      border: 1px solid var(--el-border-color-light);
      border-radius: 8px;

      &.active {
        border-color: var(--el-color-primary);
        box-shadow: 0 0 0 2px var(--el-color-primary-light-8);
      }
    }
  }
</style>
