<template>
  <div class="devops-environment-page">
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
      :data="environments"
      :pagination="pagination"
      empty-text="暂无流水线环境"
      @add="openDialog('add')"
      @refresh="fetchEnvironments"
      @pagination:size-change="handleSizeChange"
      @pagination:current-change="handleCurrentChange"
    >
      <template #default="{ item }">
        <article class="env-card">
          <div class="card-head">
            <span class="env-icon" :style="{ color: item.iconColor || '#2563eb' }">
              <ArtSvgIcon :icon="item.icon || 'ri:server-line'" />
            </span>
            <div class="env-main">
              <strong :title="item.name">{{ item.name }}</strong>
              <em :title="item.code">{{ item.code }}</em>
            </div>
          </div>
          <p :title="item.description || '暂无描述'">{{ item.description || '暂无描述' }}</p>
          <div class="card-meta">
            <ElTag size="small" :type="item.status === 1 ? 'success' : 'info'">
              {{ item.status === 1 ? '启用' : '停用' }}
            </ElTag>
            <span>排序 {{ item.sortOrder }}</span>
            <span class="card-time" :title="`创建时间：${smartFormatTime(item.createdAt)}`">
              创建：{{ smartFormatTime(item.createdAt) }}
            </span>
          </div>
          <div class="card-actions">
            <ElButton text type="primary" @click="openDialog('edit', item)">编辑</ElButton>
            <ElButton text type="danger" @click="deleteEnvironment(item)">删除</ElButton>
          </div>
        </article>
      </template>
    </ArtCard>

    <ElDialog
      v-model="dialogVisible"
      :title="dialogType === 'add' ? '新增环境' : '编辑环境'"
      width="620px"
      align-center
      :close-on-click-modal="false"
      @close="resetForm"
    >
      <ElForm ref="formRef" :model="formData" :rules="rules" label-width="120px">
        <ElFormItem label="名称" prop="name">
          <ElInput v-model="formData.name" placeholder="请输入名称" />
        </ElFormItem>
        <ElFormItem label="编码" prop="code">
          <ElInput
            v-model="formData.code"
            :disabled="dialogType === 'edit'"
            placeholder="例如 dev、prod"
          />
        </ElFormItem>
        <ElFormItem label="环境图标">
          <div class="icon-picker">
            <button
              v-for="item in iconOptions"
              :key="item.icon"
              type="button"
              :class="{ active: formData.icon === item.icon }"
              :style="{ color: item.color }"
              :title="item.label"
              @click="pickIcon(item.icon, item.color)"
            >
              <ArtSvgIcon :icon="item.icon" />
            </button>
          </div>
        </ElFormItem>
        <ElFormItem label="排序">
          <ElInputNumber v-model="formData.sortOrder" :min="0" :max="999" />
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
            placeholder="请输入环境描述"
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
  import { onMounted, reactive, ref } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import { smartFormatTime } from '@/utils/format'
  import {
    createDevopsPipelineEnvironmentApi,
    deleteDevopsPipelineEnvironmentApi,
    listDevopsPipelineEnvironmentApi,
    updateDevopsPipelineEnvironmentApi,
    type DevopsPipelineEnvironment,
    type DevopsPipelineEnvironmentForm
  } from '@/api/devops/pipeline'

  defineOptions({ name: 'DevopsPipelineEnvironment' })

  const loading = ref(false)
  const submitLoading = ref(false)
  const showSearchBar = ref(true)
  const dialogVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const formRef = ref<FormInstance>()
  const currentEnvironment = ref<DevopsPipelineEnvironment | null>(null)
  const environments = ref<DevopsPipelineEnvironment[]>([])
  const searchForm = ref({ name: undefined, code: undefined, status: undefined })
  const pagination = reactive({ current: 1, size: 15, total: 0 })
  const searchItems = [
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
  ]
  const iconOptions = [
    { label: '开发', icon: 'ri:code-s-slash-line', color: '#2563eb' },
    { label: '测试', icon: 'ri:test-tube-line', color: '#0f766e' },
    { label: '准生产', icon: 'ri:rocket-2-line', color: '#7c3aed' },
    { label: '灾备', icon: 'ri:shield-flash-line', color: '#ea580c' },
    { label: '生产', icon: 'ri:building-4-line', color: '#dc2626' },
    { label: '灰度', icon: 'ri:git-branch-line', color: '#0891b2' }
  ]
  const defaultForm = (): DevopsPipelineEnvironmentForm => ({
    name: '',
    code: '',
    description: '',
    icon: 'ri:code-s-slash-line',
    iconColor: '#2563eb',
    sortOrder: 0,
    status: 1
  })
  const formData = reactive<DevopsPipelineEnvironmentForm>(defaultForm())
  const rules: FormRules = {
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    code: [{ required: true, message: '请输入编码', trigger: 'blur' }]
  }

  let fetchPromise: Promise<void> | null = null
  let fetchSeq = 0
  const fetchEnvironments = async (force = false) => {
    if (fetchPromise && !force) return fetchPromise
    const currentSeq = ++fetchSeq
    fetchPromise = (async () => {
      loading.value = true
      try {
        const res = await listDevopsPipelineEnvironmentApi({
          page: pagination.current,
          pageSize: pagination.size,
          ...searchForm.value
        })
        if (currentSeq !== fetchSeq) return
        environments.value = res.items || []
        pagination.total = res.total || 0
      } finally {
        if (currentSeq === fetchSeq) {
          loading.value = false
        }
      }
    })()
    try {
      await fetchPromise
    } finally {
      if (currentSeq === fetchSeq) {
        fetchPromise = null
      }
    }
  }

  const openDialog = (type: 'add' | 'edit', row?: DevopsPipelineEnvironment) => {
    dialogType.value = type
    currentEnvironment.value = row || null
    Object.assign(formData, defaultForm(), row || {})
    dialogVisible.value = true
  }

  const resetForm = () => {
    formRef.value?.clearValidate()
    Object.assign(formData, defaultForm())
  }

  const pickIcon = (icon: string, color: string) => {
    formData.icon = icon
    formData.iconColor = color
  }

  const submitForm = async () => {
    await formRef.value?.validate()
    submitLoading.value = true
    try {
      if (dialogType.value === 'add') {
        await createDevopsPipelineEnvironmentApi(formData)
        pagination.current = 1
      } else if (currentEnvironment.value) {
        await updateDevopsPipelineEnvironmentApi(currentEnvironment.value.id, formData)
      }
      ElMessage.success('保存成功')
      dialogVisible.value = false
      await fetchEnvironments(true)
    } finally {
      submitLoading.value = false
    }
  }

  const deleteEnvironment = async (row: DevopsPipelineEnvironment) => {
    await ElMessageBox.confirm(`确认删除环境「${row.name}」？`, '删除确认', { type: 'warning' })
    await deleteDevopsPipelineEnvironmentApi(row.id)
    ElMessage.success('删除成功')
    await fetchEnvironments(true)
  }

  const handleSearch = () => {
    pagination.current = 1
    fetchEnvironments()
  }

  const handleReset = () => {
    searchForm.value = { name: undefined, code: undefined, status: undefined }
    handleSearch()
  }

  const handleSizeChange = (size: number) => {
    pagination.size = size
    pagination.current = 1
    fetchEnvironments()
  }

  const handleCurrentChange = (page: number) => {
    pagination.current = page
    fetchEnvironments()
  }

  onMounted(fetchEnvironments)
</script>

<style scoped lang="scss">
  .devops-environment-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .env-card {
    display: flex;
    flex-direction: column;
    min-height: 210px;
    padding: 16px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    background: var(--el-bg-color);
  }

  .card-head {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .env-icon {
    display: grid;
    flex: 0 0 44px;
    width: 44px;
    height: 44px;
    place-items: center;
    background: var(--el-fill-color-light);
    border-radius: 8px;
  }

  .env-main {
    flex: 1;
    min-width: 0;
  }

  .env-main strong,
  .env-main em,
  .env-card p {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .env-main em,
  .card-meta {
    color: var(--el-text-color-secondary);
  }

  .env-main em {
    font-style: normal;
  }

  .env-card p {
    min-height: 24px;
    margin: 16px 0;
  }

  .card-meta {
    display: flex;
    flex-wrap: nowrap;
    gap: 10px;
    align-items: center;
    margin-top: auto;
    font-size: 13px;
  }

  .card-meta span,
  .card-time {
    white-space: nowrap;
  }

  .card-actions {
    display: flex;
    justify-content: flex-end;
    padding-top: 12px;
    margin-top: 12px;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .icon-picker {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .icon-picker button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 42px;
    height: 42px;
    font-size: 20px;
    cursor: pointer;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color);
    border-radius: 8px;
  }

  .icon-picker button.active {
    border-color: var(--el-color-primary);
    box-shadow: 0 0 0 2px var(--el-color-primary-light-8);
  }
</style>
