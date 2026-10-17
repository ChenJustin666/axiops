<template>
  <div class="devops-system-page">
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
      :data="systems"
      :pagination="pagination"
      empty-text="暂无应用"
      @add="openDialog('add')"
      @refresh="fetchSystems"
      @pagination:size-change="handleSizeChange"
      @pagination:current-change="handleCurrentChange"
    >
      <template #default="{ item }">
        <article class="system-card">
          <div class="card-head">
            <span class="system-icon"><ArtSvgIcon icon="ri:apps-2-line" /></span>
            <div class="system-main">
              <strong :title="item.name">{{ item.name }}</strong>
              <span class="system-code" :title="item.code">{{ item.code }}</span>
            </div>
            <ElTag size="small" :type="item.status === 1 ? 'success' : 'info'">
              {{ item.status === 1 ? '启用' : '停用' }}
            </ElTag>
          </div>
          <p :title="item.description || '暂无描述'">{{ item.description || '暂无描述' }}</p>
          <div class="card-meta">
            <span>负责人 {{ item.ownerUserIds?.length || 0 }} 人</span>
            <span class="card-time" :title="`创建时间：${smartFormatTime(item.createdAt)}`">
              创建：{{ smartFormatTime(item.createdAt) }}
            </span>
          </div>
          <div class="card-actions">
            <ElButton text type="primary" @click="openDialog('edit', item)">编辑</ElButton>
            <ElButton text type="danger" @click="deleteSystem(item)">删除</ElButton>
          </div>
        </article>
      </template>
    </ArtCard>

    <ElDialog
      v-model="dialogVisible"
      :title="dialogType === 'add' ? '新增应用' : '编辑应用'"
      width="620px"
      align-center
      :close-on-click-modal="false"
      @close="resetForm"
    >
      <ElForm ref="formRef" :model="formData" :rules="rules" label-width="110px">
        <ElFormItem label="所属项目">
          <ElInput :model-value="projectStore.selectedProject?.name || '请选择项目'" disabled />
        </ElFormItem>
        <ElFormItem label="名称" prop="name">
          <ElInput v-model="formData.name" placeholder="请输入名称" />
        </ElFormItem>
        <ElFormItem label="编码" prop="code">
          <ElInput
            v-model="formData.code"
            :disabled="dialogType === 'edit'"
            placeholder="例如 order-service"
          />
        </ElFormItem>
        <ElFormItem label="负责人 ID">
          <ElSelect
            v-model="formData.ownerUserIds"
            multiple
            filterable
            allow-create
            default-first-option
            placeholder="输入用户 ID 后回车"
            style="width: 100%"
          />
        </ElFormItem>
        <ElFormItem label="状态">
          <ElRadioGroup v-model="formData.status">
            <ElRadio :value="1">启用</ElRadio>
            <ElRadio :value="0">停用</ElRadio>
          </ElRadioGroup>
        </ElFormItem>
        <ElFormItem label="描述">
          <ElInput
            v-model="formData.description"
            type="textarea"
            :rows="3"
            placeholder="请输入应用描述"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="dialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="submitLoading" @click="submitForm">保存</ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref, watch } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import { useDevopsProjectStore } from '@/store/modules/devopsProject'
  import { smartFormatTime } from '@/utils/format'
  import {
    createDevopsSystemApi,
    deleteDevopsSystemApi,
    listDevopsSystemApi,
    updateDevopsSystemApi,
    type DevopsSystem,
    type DevopsSystemForm
  } from '@/api/devops/pipeline'

  defineOptions({ name: 'DevopsProjectSystem' })

  const projectStore = useDevopsProjectStore()
  const loading = ref(false)
  const submitLoading = ref(false)
  const showSearchBar = ref(true)
  const dialogVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const formRef = ref<FormInstance>()
  const currentSystem = ref<DevopsSystem | null>(null)
  const systems = ref<DevopsSystem[]>([])
  const searchForm = ref({ name: undefined, code: undefined, status: undefined })
  const pagination = reactive({ current: 1, size: 15, total: 0 })
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
  const defaultForm = (): DevopsSystemForm => ({
    projectId: projectStore.selectedProjectId,
    name: '',
    code: '',
    description: '',
    ownerUserIds: [],
    status: 1
  })
  const formData = reactive<DevopsSystemForm>(defaultForm())
  const rules: FormRules = {
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    code: [{ required: true, message: '请输入编码', trigger: 'blur' }]
  }

  let fetchPromise: Promise<void> | null = null
  const fetchSystems = async () => {
    if (fetchPromise) return fetchPromise
    fetchPromise = (async () => {
      await projectStore.fetchProjects()
      if (!projectStore.selectedProjectId) {
        systems.value = []
        pagination.total = 0
        return
      }
      loading.value = true
      try {
        const res = await listDevopsSystemApi({
          page: pagination.current,
          pageSize: pagination.size,
          projectId: projectStore.selectedProjectId,
          ...searchForm.value
        })
        systems.value = res.items || []
        pagination.total = res.total || 0
      } finally {
        loading.value = false
      }
    })()
    try {
      await fetchPromise
    } finally {
      fetchPromise = null
    }
  }

  const openDialog = (type: 'add' | 'edit', row?: DevopsSystem) => {
    dialogType.value = type
    currentSystem.value = row || null
    Object.assign(formData, defaultForm(), row || {})
    dialogVisible.value = true
  }

  const resetForm = () => {
    formRef.value?.clearValidate()
    Object.assign(formData, defaultForm())
  }

  const submitForm = async () => {
    if (!projectStore.selectedProjectId) {
      ElMessage.warning('请选择项目')
      return
    }
    await formRef.value?.validate()
    submitLoading.value = true
    try {
      formData.projectId = projectStore.selectedProjectId
      if (dialogType.value === 'add') {
        await createDevopsSystemApi(formData)
      } else if (currentSystem.value) {
        await updateDevopsSystemApi(currentSystem.value.id, formData)
      }
      ElMessage.success('保存成功')
      dialogVisible.value = false
      await fetchSystems()
    } finally {
      submitLoading.value = false
    }
  }

  const deleteSystem = async (row: DevopsSystem) => {
    await ElMessageBox.confirm(`确认删除应用「${row.name}」？`, '删除确认', { type: 'warning' })
    await deleteDevopsSystemApi(row.id)
    ElMessage.success('删除成功')
    await fetchSystems()
  }

  const handleSearch = () => {
    pagination.current = 1
    fetchSystems()
  }

  const handleReset = () => {
    searchForm.value = { name: undefined, code: undefined, status: undefined }
    handleSearch()
  }

  const handleSizeChange = (size: number) => {
    pagination.size = size
    pagination.current = 1
    fetchSystems()
  }

  const handleCurrentChange = (page: number) => {
    pagination.current = page
    fetchSystems()
  }

  watch(() => projectStore.selectedProjectId, fetchSystems)
  onMounted(fetchSystems)
</script>

<style scoped lang="scss">
  .devops-system-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .system-card {
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    background: var(--el-bg-color);
  }

  .system-code,
  .card-meta {
    color: var(--el-text-color-secondary);
  }

  .system-card {
    display: flex;
    flex-direction: column;
    min-height: 184px;
    padding: 16px;
  }

  .card-head {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .system-icon {
    display: grid;
    flex: 0 0 44px;
    width: 44px;
    height: 44px;
    color: #2563eb;
    place-items: center;
    background: #eff6ff;
    border-radius: 8px;
  }

  .system-main {
    flex: 1;
    min-width: 0;
  }

  .system-main strong,
  .system-code,
  .system-card p {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .system-card p {
    min-height: 24px;
    margin: 14px 0;
  }

  .system-code {
    margin-top: 4px;
    font-size: 13px;
    font-style: normal;
  }

  .card-meta {
    display: flex;
    gap: 8px;
    justify-content: space-between;
    margin-top: auto;
    font-size: 13px;
  }

  .card-meta span {
    white-space: nowrap;
  }

  .card-time {
    flex-shrink: 0;
  }

  .card-actions {
    display: flex;
    justify-content: flex-end;
    padding-top: 12px;
    margin-top: 12px;
    border-top: 1px solid var(--el-border-color-lighter);
  }
</style>
