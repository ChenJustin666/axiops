<template>
  <div class="devops-project-center-page art-full-height">
    <ArtSearchBar
      v-show="showSearchBar"
      v-model="searchForm"
      :items="searchItems"
      @search="handleSearch"
      @reset="handleReset"
    />

    <ArtCard
      v-model:showSearchBar="showSearchBar"
      :loading="loading"
      :data="projects"
      :pagination="pagination"
      empty-text="暂无 DevOps 项目"
      @add="openDialog('add')"
      @refresh="fetchProjects(true)"
      @pagination:size-change="handleSizeChange"
      @pagination:current-change="handleCurrentChange"
    >
      <template #default="{ item }">
        <article class="project-card">
          <div class="card-head">
            <span class="project-icon">
              <ArtSvgIcon icon="ri:folder-3-line" />
            </span>
            <div class="project-main">
              <strong :title="item.name">{{ item.name }}</strong>
              <span :title="item.code">{{ item.code || '-' }}</span>
            </div>
            <ElTag size="small" :type="item.status === 1 ? 'success' : 'info'">
              {{ item.status === 1 ? '启用' : '停用' }}
            </ElTag>
          </div>

          <p class="project-desc" :title="item.description || '暂无描述'">
            {{ item.description || '暂无描述' }}
          </p>

          <div class="project-meta">
            <span>引擎：{{ item.pipelineEngineType || 'jenkins' }}</span>
            <span :title="`创建时间：${smartFormatTime(item.createdAt)}`">
              创建：{{ smartFormatTime(item.createdAt) }}
            </span>
          </div>

          <div class="project-actions">
            <ElButton text type="primary" @click="openDialog('edit', item)">编辑</ElButton>
            <ElButton text type="danger" @click="handleDelete(item)">删除</ElButton>
          </div>
        </article>
      </template>
    </ArtCard>

    <ElDialog
      v-model="dialogVisible"
      :title="dialogType === 'add' ? '新增 DevOps 项目' : '编辑 DevOps 项目'"
      width="640px"
      align-center
      :close-on-click-modal="false"
      @close="resetForm"
    >
      <ElForm ref="formRef" :model="formData" :rules="rules" label-width="120px">
        <ElFormItem label="名称" prop="name">
          <ElInput v-model="formData.name" placeholder="请输入项目名称" />
        </ElFormItem>

        <ElFormItem label="构建渠道" prop="defaultEngineChannelId">
          <ElSelect
            v-model="formData.defaultEngineChannelId"
            filterable
            placeholder="请选择默认构建渠道"
            style="width: 100%"
            @change="handleDefaultChannelChange"
          >
            <ElOption
              v-for="channel in buildChannels"
              :key="channel.id"
              :label="channelLabel(channel)"
              :value="channel.id"
            />
          </ElSelect>
        </ElFormItem>

        <ElFormItem label="可用构建渠道" prop="buildChannelIds">
          <ElSelect
            v-model="formData.buildChannelIds"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            placeholder="请选择可用构建渠道"
            style="width: 100%"
          >
            <ElOption
              v-for="channel in buildChannels"
              :key="channel.id"
              :label="channelLabel(channel)"
              :value="channel.id"
            />
          </ElSelect>
        </ElFormItem>

        <ElFormItem label="状态">
          <ElRadioGroup v-model="formData.status">
            <ElRadio :value="1">启用</ElRadio>
            <ElRadio :value="0">停用</ElRadio>
          </ElRadioGroup>
        </ElFormItem>

        <ElFormItem label="扩展配置">
          <ElInput
            v-model="formData.extraConfig"
            type="textarea"
            :rows="3"
            placeholder='可选 JSON，例如 {"namespace":"devops-demo"}'
          />
        </ElFormItem>

        <ElFormItem label="描述">
          <ElInput
            v-model="formData.description"
            type="textarea"
            :rows="3"
            placeholder="请输入项目描述"
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
  import { computed, onMounted, reactive, ref } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import { smartFormatTime } from '@/utils/format'
  import { listDevopsChannelApi, type DevopsChannel } from '@/api/devops/channel'
  import {
    createDevopsProjectApi,
    deleteDevopsProjectApi,
    listDevopsProjectApi,
    updateDevopsProjectApi,
    type DevopsProject,
    type DevopsProjectForm
  } from '@/api/devops/project'
  import { useDevopsProjectStore } from '@/store/modules/devopsProject'

  defineOptions({ name: 'DevopsProjectCenter' })

  const devopsProjectStore = useDevopsProjectStore()
  const loading = ref(false)
  const submitLoading = ref(false)
  const showSearchBar = ref(true)
  const dialogVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const formRef = ref<FormInstance>()
  const currentProject = ref<DevopsProject | null>(null)
  const projects = ref<DevopsProject[]>([])
  const buildChannels = ref<DevopsChannel[]>([])
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

  const defaultForm = (): DevopsProjectForm => ({
    name: '',
    description: '',
    defaultEngineChannelId: '',
    buildChannelIds: [],
    status: 1,
    extraConfig: ''
  })

  const formData = reactive<DevopsProjectForm>(defaultForm())
  const rules: FormRules = {
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    defaultEngineChannelId: [{ required: true, message: '请选择默认构建渠道', trigger: 'change' }],
    buildChannelIds: [{ required: true, message: '请选择可用构建渠道', trigger: 'change' }]
  }

  const channelLabel = (channel: DevopsChannel) =>
    `${channel.name || channel.code || channel.id}${channel.channelType ? ` (${channel.channelType})` : ''}`

  const handleDefaultChannelChange = (channelId: string) => {
    if (channelId && !formData.buildChannelIds.includes(channelId)) {
      formData.buildChannelIds = [channelId, ...formData.buildChannelIds]
    }
  }

  const loadBuildChannels = async () => {
    const res = await listDevopsChannelApi({ page: 1, pageSize: 200, status: 1 })
    buildChannels.value = (res.items || []).filter((item) => {
      const groupCode = String((item as DevopsChannel & { groupCode?: string }).groupCode || '')
      return groupCode === 'build_engine' || ['jenkins', 'tekton'].includes(item.channelType)
    })
  }

  const fetchProjects = async (forceStoreRefresh = false) => {
    loading.value = true
    try {
      const res = await listDevopsProjectApi({
        page: pagination.current,
        pageSize: pagination.size,
        ...searchForm.value
      })
      projects.value = res.items || []
      pagination.total = res.total || 0
      devopsProjectStore.clear()
      await devopsProjectStore.fetchProjects(forceStoreRefresh)
    } finally {
      loading.value = false
    }
  }

  const handleSearch = () => {
    pagination.current = 1
    fetchProjects()
  }

  const handleReset = () => {
    searchForm.value = { name: undefined, code: undefined, status: undefined }
    handleSearch()
  }

  const handleSizeChange = (size: number) => {
    pagination.size = size
    pagination.current = 1
    fetchProjects()
  }

  const handleCurrentChange = (current: number) => {
    pagination.current = current
    fetchProjects()
  }

  const openDialog = async (type: 'add' | 'edit', row?: DevopsProject) => {
    dialogType.value = type
    currentProject.value = row || null
    if (!buildChannels.value.length) {
      await loadBuildChannels()
    }
    Object.assign(formData, defaultForm(), {
      name: row?.name || '',
      description: row?.description || '',
      defaultEngineChannelId: row?.defaultEngineChannelId || '',
      buildChannelIds: row?.defaultEngineChannelId ? [row.defaultEngineChannelId] : [],
      status: row?.status ?? 1,
      extraConfig: row?.extraConfig || ''
    })
    dialogVisible.value = true
  }

  const resetForm = () => {
    formRef.value?.clearValidate()
    Object.assign(formData, defaultForm())
  }

  const submitForm = async () => {
    await formRef.value?.validate()
    handleDefaultChannelChange(formData.defaultEngineChannelId)
    submitLoading.value = true
    try {
      if (dialogType.value === 'add') {
        await createDevopsProjectApi(formData)
      } else if (currentProject.value) {
        await updateDevopsProjectApi(currentProject.value.id, formData)
      }
      ElMessage.success('保存成功')
      dialogVisible.value = false
      await fetchProjects(true)
    } finally {
      submitLoading.value = false
    }
  }

  const handleDelete = async (row: DevopsProject) => {
    await ElMessageBox.confirm(`确认删除项目「${row.name}」吗？`, '删除确认', { type: 'warning' })
    await deleteDevopsProjectApi(row.id)
    ElMessage.success('删除成功')
    fetchProjects(true)
  }

  onMounted(async () => {
    await Promise.all([loadBuildChannels(), fetchProjects()])
  })
</script>

<style scoped lang="scss">
  .devops-project-center-page {
    .project-card {
      display: flex;
      flex-direction: column;
      gap: 14px;
      min-height: 190px;
      padding: 18px;
      border: 1px solid var(--art-border-color);
      border-radius: 8px;
      background: var(--art-main-bg-color);
    }

    .card-head {
      display: grid;
      grid-template-columns: 42px minmax(0, 1fr) auto;
      gap: 12px;
      align-items: center;
    }

    .project-icon {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 42px;
      height: 42px;
      color: #2563eb;
      background: rgba(37, 99, 235, 10%);
      border-radius: 8px;
      font-size: 24px;
    }

    .project-main {
      min-width: 0;

      strong,
      span {
        display: block;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      strong {
        color: var(--art-text-gray-900);
        font-size: 16px;
        line-height: 24px;
      }

      span {
        margin-top: 4px;
        color: var(--art-text-gray-500);
        font-size: 12px;
      }
    }

    .project-desc {
      display: -webkit-box;
      min-height: 42px;
      margin: 0;
      overflow: hidden;
      color: var(--art-text-gray-600);
      line-height: 21px;
      -webkit-line-clamp: 2;
      -webkit-box-orient: vertical;
    }

    .project-meta {
      display: flex;
      flex-wrap: wrap;
      gap: 8px 14px;
      color: var(--art-text-gray-500);
      font-size: 12px;
    }

    .project-actions {
      display: flex;
      justify-content: flex-end;
      gap: 8px;
      margin-top: auto;
    }
  }
</style>
