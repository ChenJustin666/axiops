<template>
  <div class="devops-pipeline-template-page" :class="{ 'editor-active': isEditorMode }">
    <template v-if="!isEditorMode">
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
        :data="templates"
        :pagination="pagination"
        empty-text="暂无流水线模板"
        @add="enterTemplateEditor('add')"
        @refresh="refreshAll"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      >
        <template #default="{ item }">
          <article class="template-card">
            <span class="template-icon" :style="{ color: item.iconColor || '#64748b' }">
              <ArtSvgIcon :icon="item.icon || 'ri:flow-chart'" />
            </span>
            <span class="template-info">
              <strong :title="item.name">{{ item.name }}</strong>
              <em :title="item.code">{{ item.code }}</em>
            </span>
            <p class="template-desc" :title="item.description || '暂无描述'">
              {{ item.description || '暂无描述' }}
            </p>
            <span class="template-tags">
              <ElTag size="small" :type="item.scope === 'system' ? 'warning' : 'success'">
                {{ item.scope === 'system' ? '系统' : '项目' }}
              </ElTag>
              <ElTag size="small" type="info">{{ engineTypeName(item.engineType) }}</ElTag>
              <ElTag size="small">{{ item.steps?.length || 0 }} 个步骤</ElTag>
            </span>
            <div class="template-card-actions">
              <ElButton text @click="enterTemplateEditor('view', item)">查看</ElButton>
              <ElButton
                text
                type="primary"
                :disabled="!canManageTemplate(item)"
                @click="enterTemplateEditor('edit', item)"
              >
                配置
              </ElButton>
              <ElButton
                text
                type="danger"
                :disabled="!canManageTemplate(item)"
                @click="deleteCurrentTemplate(item)"
              >
                删除
              </ElButton>
            </div>
          </article>
        </template>
      </ArtCard>
    </template>

    <section v-else class="template-editor-workbench">
      <ElForm
        ref="formRef"
        class="template-base-config"
        :model="formData"
        :rules="rules"
        label-width="88px"
        :disabled="isViewMode"
      >
        <div class="base-config-title">
          <span class="template-icon large" :style="{ color: formData.iconColor || '#64748b' }">
            <ArtSvgIcon :icon="formData.icon || 'ri:flow-chart'" />
          </span>
          <div>
            <strong>{{ editorTitle }}</strong>
            <span>{{ engineTypeName(formData.engineType) }} 工作流编排</span>
          </div>
        </div>
        <div class="base-config-head">
          <strong>基础信息</strong>
          <span>保存后可复用当前编排</span>
        </div>
        <div class="base-config-grid">
          <ElFormItem label="名称" prop="name">
            <ElInput v-model="formData.name" placeholder="请输入名称" />
          </ElFormItem>
          <ElFormItem label="编码" prop="code">
            <ElInput
              v-model="formData.code"
              :disabled="editorMode !== 'add'"
              placeholder="例如 java_docker"
            />
          </ElFormItem>
          <ElFormItem label="级别" prop="scope">
            <ElSelect
              v-model="formData.scope"
              style="width: 100%"
              @change="handleTemplateScopeChange"
            >
              <ElOption v-if="isSuperAdmin" label="系统级" value="system" />
              <ElOption label="项目级" value="project" />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="引擎" prop="engineType">
            <ElSelect
              v-model="formData.engineType"
              :disabled="editorMode !== 'add' || isRouteEngineLocked"
              style="width: 100%"
              @change="handleTemplateEngineChange"
            >
              <ElOption label="Jenkins" value="jenkins" />
              <ElOption label="Tekton" value="tekton" />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="所属项目">
            <ElInput :model-value="templateProjectName" disabled />
          </ElFormItem>
          <ElFormItem label="状态">
            <ElRadioGroup v-model="formData.status">
              <ElRadio :value="1">启用</ElRadio>
              <ElRadio :value="0">停用</ElRadio>
            </ElRadioGroup>
          </ElFormItem>
          <ElFormItem label="图标" class="base-icon-row">
            <div class="icon-picker compact-picker">
              <button
                v-for="item in iconOptions"
                :key="item.icon"
                type="button"
                :disabled="isViewMode"
                :class="{ active: formData.icon === item.icon }"
                :style="{ color: item.color }"
                @click="pickIcon(item.icon, item.color)"
              >
                <ArtSvgIcon :icon="item.icon" />
              </button>
            </div>
          </ElFormItem>
          <ElFormItem label="描述" class="base-desc-row">
            <ElInput
              v-model="formData.description"
              type="textarea"
              :rows="2"
              placeholder="请输入描述"
            />
          </ElFormItem>
        </div>
      </ElForm>

      <div class="editor-workflow-board">
        <DevopsWorkflowDesigner
          v-model="currentSteps"
          v-model:category-id="stepFilter.categoryId"
          :library-items="workflowLibraryItems"
          :category-options="categoryOptions"
          :post-category-id="postCategoryId"
          :library-loading="stepLoading"
          :engine-type="selectedEngineType"
          empty-text="暂无可用步骤"
          :readonly="isViewMode"
          :dynamic-project-id="projectStore.selectedProjectId"
          :enable-step-param-config="false"
          @refresh-library="fetchStepLibrary"
          @save="saveTemplateEditor"
        >
          <template #left-actions>
            <ElButton @click="handleEditorCancel">{{ isViewMode ? '返回' : '取消' }}</ElButton>
          </template>
          <template #actions>
            <ElButton
              v-if="!isViewMode"
              type="primary"
              :loading="submitLoading"
              @click="saveTemplateEditor"
            >
              保存模板
            </ElButton>
          </template>
        </DevopsWorkflowDesigner>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
  import { computed, defineAsyncComponent, onActivated, onMounted, reactive, ref, watch } from 'vue'
  import { useRoute } from 'vue-router'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import type {
    WorkflowLibraryItem,
    WorkflowNode
  } from '@/components/devops-workflow-designer/types'
  import { normalizeWorkflowBranches } from '@/components/devops-workflow-designer/workflow-utils'
  import { useDevopsProjectStore } from '@/store/modules/devopsProject'
  import { useUserStore } from '@/store/modules/user'
  import {
    createDevopsPipelineTemplateApi,
    deleteDevopsPipelineTemplateApi,
    getDevopsPipelineTemplateApi,
    listDevopsJenkinsStepApi,
    listDevopsTektonStepApi,
    listDevopsPipelineTemplateApi,
    listDevopsStepCategoryApi,
    updateDevopsPipelineTemplateApi,
    type DevopsPipelineTemplate,
    type DevopsPipelineTemplateForm,
    type DevopsStepCategory,
    type DevopsStepTemplate
  } from '@/api/devops/pipeline'

  defineOptions({ name: 'DevopsPipelineTemplate' })

  const DevopsWorkflowDesigner = defineAsyncComponent(
    () => import('@/components/devops-workflow-designer/index.vue')
  )

  const projectStore = useDevopsProjectStore()
  const userStore = useUserStore()
  const route = useRoute()
  const loading = ref(false)
  const stepLoading = ref(false)
  const submitLoading = ref(false)
  const showSearchBar = ref(true)
  const editorMode = ref<'browse' | 'add' | 'edit' | 'view'>('browse')
  const formRef = ref<FormInstance>()
  const currentTemplate = ref<DevopsPipelineTemplate | null>(null)
  const templates = ref<DevopsPipelineTemplate[]>([])
  const categories = ref<DevopsStepCategory[]>([])
  const stepLibrary = ref<DevopsStepTemplate[]>([])
  const currentSteps = ref<WorkflowNode[]>([])
  const searchParams = ref<Record<string, any>>({})
  const searchForm = ref({
    name: undefined,
    code: undefined,
    scope: undefined,
    engineType: undefined,
    status: undefined
  })
  const stepFilter = reactive({ categoryId: '' })
  const pagination = ref({ current: 1, size: 15, total: 0 })
  const isEditorMode = computed(() => editorMode.value !== 'browse')
  const isViewMode = computed(() => editorMode.value === 'view')
  const isSuperAdmin = computed(() =>
    (userStore.getUserInfo.roles || []).some((role) => String(role).toUpperCase() === 'SUPER_ADMIN')
  )
  type TemplateEngineType = 'jenkins' | 'tekton'
  const routeTemplateEngine = computed<TemplateEngineType | ''>(() => {
    const routeName = String(route.name || '')
    const routePath = String(route.path || '')
    if (routeName === 'DevOpsTektonTemplateLibrary' || routePath.includes('/tekton-template')) {
      return 'tekton'
    }
    if (routeName === 'DevOpsJenkinsTemplateLibrary' || routePath.includes('/jenkins-template')) {
      return 'jenkins'
    }
    return ''
  })
  const isRouteEngineLocked = computed(() => routeTemplateEngine.value !== '')
  const defaultTemplateEngine = (): TemplateEngineType => routeTemplateEngine.value || 'jenkins'
  const templateLibraryTitle = computed(() => {
    if (routeTemplateEngine.value === 'tekton') return 'Tekton 模板库'
    if (routeTemplateEngine.value === 'jenkins') return 'Jenkins 模板库'
    return '流水线模板'
  })
  const editorTitle = computed(() => {
    if (editorMode.value === 'add') return `新增${templateLibraryTitle.value}`
    if (editorMode.value === 'view') return `查看${templateLibraryTitle.value}`
    return `配置${templateLibraryTitle.value}`
  })

  const defaultTemplateScope = () => (isSuperAdmin.value ? 'system' : 'project')
  const defaultForm = (): DevopsPipelineTemplateForm => ({
    name: '',
    code: '',
    icon: 'ri:flow-chart',
    iconColor: '#2563eb',
    description: '',
    scope: defaultTemplateScope(),
    projectId: defaultTemplateScope() === 'project' ? projectStore.selectedProjectId : '',
    engineType: defaultTemplateEngine(),
    steps: [],
    status: 1
  })
  const formData = reactive<DevopsPipelineTemplateForm>(defaultForm())
  const selectedProjectName = computed(() => {
    const project = projectStore.selectedProject
    if (!project) return ''
    return project.code ? `${project.name}（${project.code}）` : project.name
  })
  const templateProjectName = computed(() => {
    if (formData.scope !== 'project') return ''
    return selectedProjectName.value || '未选择项目'
  })
  const rules: FormRules = {
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    code: [{ required: true, message: '请输入编码', trigger: 'blur' }],
    scope: [{ required: true, message: '请选择级别', trigger: 'change' }],
    engineType: [{ required: true, message: '请选择引擎', trigger: 'change' }]
  }

  const iconOptions = [
    { icon: 'ri:flow-chart', color: '#2563eb' },
    { icon: 'simple-icons:jenkins', color: '#d24939' },
    { icon: 'simple-icons:tekton', color: '#2563eb' },
    { icon: 'simple-icons:docker', color: '#0ea5e9' },
    { icon: 'simple-icons:kubernetes', color: '#326ce5' },
    { icon: 'simple-icons:java', color: '#b07219' },
    { icon: 'simple-icons:go', color: '#00add8' },
    { icon: 'simple-icons:python', color: '#3776ab' },
    { icon: 'simple-icons:nodedotjs', color: '#339933' },
    { icon: 'simple-icons:vuedotjs', color: '#42b883' }
  ]

  const categoryOptions = computed(() =>
    categories.value.map((item) => ({
      label: `${item.name}（${item.code}）`,
      value: item.id,
      code: item.code
    }))
  )
  const postCategoryId = computed(
    () => categories.value.find((item) => item.code === 'post')?.id || ''
  )
  const firstNormalCategoryId = computed(
    () => categories.value.find((item) => item.code !== 'post')?.id || ''
  )
  const categoryCodeMap = computed(() =>
    categories.value.reduce<Record<string, string>>((acc, item) => {
      acc[item.id] = item.code
      return acc
    }, {})
  )
  const selectedEngineType = computed(() =>
    isEditorMode.value
      ? formData.engineType
      : currentTemplate.value?.engineType || formData.engineType
  )
  const workflowLibraryItems = computed<WorkflowLibraryItem[]>(() =>
    stepLibrary.value.map((item) => ({
      id: item.id,
      name: item.name,
      code: item.code,
      categoryId: item.categoryId,
      categoryCode: categoryCodeMap.value[item.categoryId] || '',
      icon: item.icon,
      iconColor: item.iconColor,
      type: item.type,
      stageContent: item.stageContent,
      paramsCount:
        item.engineType === 'tekton' ? item.taskParams?.length || 0 : item.params?.length || 0,
      params: item.params || [],
      taskParams: item.taskParams || [],
      taskResults: item.taskResults || [],
      taskWorkspaces: item.taskWorkspaces || []
    }))
  )
  type StepLibraryRefreshPayload = {
    categoryId?: string
  }
  const engineTypeName = (engineType?: string) => (engineType === 'tekton' ? 'Tekton' : 'Jenkins')
  const searchItems = computed(() => {
    const items = [
      { label: '名称', key: 'name', type: 'input', props: { clearable: true } },
      { label: '编码', key: 'code', type: 'input', props: { clearable: true } },
      {
        label: '级别',
        key: 'scope',
        type: 'select',
        props: {
          clearable: true,
          options: [
            { label: '系统级', value: 'system' },
            { label: '项目级', value: 'project' }
          ]
        }
      }
    ]
    if (!isRouteEngineLocked.value) {
      items.push({
        label: '引擎',
        key: 'engineType',
        type: 'select',
        props: {
          clearable: true,
          options: [
            { label: 'Jenkins', value: 'jenkins' },
            { label: 'Tekton', value: 'tekton' }
          ]
        }
      })
    }
    items.push({
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
    })
    return items
  })

  const pickParams = () =>
    Object.entries(searchForm.value).reduce(
      (acc, [key, value]) => {
        if (value !== undefined && value !== null && value !== '') acc[key] = value
        return acc
      },
      {} as Record<string, any>
    )

  const fetchCategories = async () => {
    const res = await listDevopsStepCategoryApi({ page: 1, pageSize: 200, status: 1 })
    categories.value = res.items || []
  }

  const fetchStepLibrary = async (payload?: StepLibraryRefreshPayload) => {
    if (categories.value.length === 0) {
      await fetchCategories()
    }
    const categoryId = payload?.categoryId || stepFilter.categoryId || firstNormalCategoryId.value
    if (categoryId && stepFilter.categoryId !== categoryId) stepFilter.categoryId = categoryId
    stepLoading.value = true
    try {
      const stepApi =
        selectedEngineType.value === 'tekton' ? listDevopsTektonStepApi : listDevopsJenkinsStepApi
      const res = await stepApi({
        page: 1,
        pageSize: 200,
        status: 1,
        engineType: selectedEngineType.value,
        categoryId: categoryId || undefined
      })
      stepLibrary.value = res.items || []
    } finally {
      stepLoading.value = false
    }
  }

  const fetchTemplates = async () => {
    loading.value = true
    try {
      const params = { ...searchParams.value }
      if (routeTemplateEngine.value) {
        params.engineType = routeTemplateEngine.value
      }
      if (!isSuperAdmin.value && projectStore.selectedProjectId) {
        params.projectId = projectStore.selectedProjectId
      }
      const res = await listDevopsPipelineTemplateApi({
        page: pagination.value.current,
        pageSize: pagination.value.size,
        ...params
      })
      templates.value = res.items || []
      pagination.value.total = res.total || 0
    } finally {
      loading.value = false
    }
  }

  const refreshAll = async () => {
    await Promise.all([
      projectStore.initialized ? Promise.resolve() : projectStore.fetchProjects(),
      fetchCategories()
    ])
    await fetchTemplates()
  }

  const selectTemplate = (item: DevopsPipelineTemplate) => {
    currentTemplate.value = item
    currentSteps.value = normalizeWorkflowBranches(
      (item.steps || []).map((step, index) => ({
        ...step,
        id: step.id || `node-${index + 1}`,
        stepType: step.stepType || 'task',
        enabled: step.enabled !== false
      }))
    )
    stepFilter.categoryId = ''
  }

  const handleSearch = () => {
    searchParams.value = pickParams()
    pagination.value.current = 1
    fetchTemplates()
  }

  const handleReset = () => {
    searchParams.value = {}
    pagination.value.current = 1
    fetchTemplates()
  }

  const handleSizeChange = (size: number) => {
    pagination.value.size = size
    pagination.value.current = 1
    fetchTemplates()
  }

  const handleCurrentChange = (page: number) => {
    pagination.value.current = page
    fetchTemplates()
  }

  const pickIcon = (icon: string, color: string) => {
    if (isViewMode.value) return
    formData.icon = icon
    formData.iconColor = color
  }

  const enterTemplateEditor = async (
    mode: 'add' | 'edit' | 'view',
    row?: DevopsPipelineTemplate | null
  ) => {
    if (mode === 'edit' && row && !canManageTemplate(row)) {
      ElMessage.warning('只有 SUPER_ADMIN 可以维护系统级模板')
      return
    }
    editorMode.value = mode
    formRef.value?.clearValidate()
    stepFilter.categoryId = ''
    if (mode === 'add') {
      currentTemplate.value = null
      currentSteps.value = []
      Object.assign(formData, defaultForm())
    } else if (row) {
      currentTemplate.value = row
      Object.assign(formData, defaultForm(), row)
      currentSteps.value = normalizeWorkflowBranches(
        (row.steps || []).map((step, index) => ({
          ...step,
          id: step.id || `node-${index + 1}`,
          stepType: step.stepType || 'task',
          enabled: step.enabled !== false
        }))
      )
    }
    if (formData.scope === 'project') {
      formData.projectId = row?.projectId || projectStore.selectedProjectId
    }
    if (categories.value.length === 0) {
      await fetchCategories()
    }
    if (mode !== 'view') {
      await fetchStepLibrary()
    } else {
      stepLibrary.value = []
    }
  }

  const exitTemplateEditor = () => {
    editorMode.value = 'browse'
    formRef.value?.clearValidate()
    currentTemplate.value = null
    currentSteps.value = []
    Object.assign(formData, defaultForm())
  }

  const handleTemplateEngineChange = async () => {
    if (isViewMode.value) return
    if (editorMode.value === 'add') {
      currentSteps.value = []
    }
    stepFilter.categoryId = ''
    await fetchStepLibrary()
  }

  const syncTemplateProject = () => {
    formData.projectId = formData.scope === 'project' ? projectStore.selectedProjectId : ''
  }

  const handleTemplateScopeChange = () => {
    syncTemplateProject()
    if (formData.scope === 'project' && !projectStore.selectedProjectId) {
      ElMessage.warning('请先选择右上角 DevOps 项目')
    }
  }

  const handleEditorCancel = async () => {
    if (isViewMode.value) {
      exitTemplateEditor()
      return
    }
    try {
      await ElMessageBox.confirm('确认取消当前配置？未保存内容将丢失。', '取消确认', {
        type: 'warning'
      })
    } catch {
      return
    }
    exitTemplateEditor()
  }

  const templateStepsPayload = () =>
    normalizeWorkflowBranches(currentSteps.value).map((step) => ({
      ...step,
      paramValues: ''
    }))

  const canManageTemplate = (row: DevopsPipelineTemplate) => {
    if (row.scope === 'system') return isSuperAdmin.value
    return true
  }

  const saveTemplateEditor = async () => {
    if (isViewMode.value) return
    await formRef.value?.validate()
    if (routeTemplateEngine.value) {
      formData.engineType = routeTemplateEngine.value
    }
    if (formData.scope === 'system' && !isSuperAdmin.value) {
      ElMessage.warning('只有 SUPER_ADMIN 可以创建或维护系统级模板')
      return
    }
    if (formData.scope === 'project') {
      if (!projectStore.selectedProjectId) {
        ElMessage.warning('请先选择右上角 DevOps 项目')
        return
      }
      formData.projectId = projectStore.selectedProjectId
    } else {
      formData.projectId = ''
    }
    try {
      await ElMessageBox.confirm('确认保存流水线模板？保存成功后将返回列表。', '保存确认', {
        type: 'warning'
      })
    } catch {
      return
    }
    submitLoading.value = true
    try {
      const payload = { ...formData, steps: templateStepsPayload() }
      if (editorMode.value === 'add') {
        const res = await createDevopsPipelineTemplateApi(payload)
        ElMessage.success('新增成功')
        const created = await getDevopsPipelineTemplateApi(res.id)
        await fetchTemplates()
        selectTemplate(created)
        Object.assign(formData, defaultForm(), created)
        exitTemplateEditor()
      } else if (currentTemplate.value) {
        const templateId = currentTemplate.value.id
        await updateDevopsPipelineTemplateApi(templateId, payload)
        ElMessage.success('保存成功')
        const updated = await getDevopsPipelineTemplateApi(templateId)
        await fetchTemplates()
        selectTemplate(updated)
        Object.assign(formData, defaultForm(), updated)
        exitTemplateEditor()
      }
    } finally {
      submitLoading.value = false
    }
  }

  const deleteCurrentTemplate = async (row?: DevopsPipelineTemplate) => {
    const target = row || currentTemplate.value
    if (!target) return
    if (!canManageTemplate(target)) {
      ElMessage.warning('只有 SUPER_ADMIN 可以删除系统级模板')
      return
    }
    await ElMessageBox.confirm(`确认删除流水线模板 ${target.name}？`, '删除确认', {
      type: 'warning'
    })
    await deleteDevopsPipelineTemplateApi(target.id)
    ElMessage.success('删除成功')
    currentTemplate.value = null
    currentSteps.value = []
    await fetchTemplates()
  }

  watch(
    () => projectStore.selectedProjectId,
    () => {
      if (formData.scope === 'project') {
        formData.projectId = projectStore.selectedProjectId
      }
      if (!isEditorMode.value && !isSuperAdmin.value) {
        fetchTemplates()
      }
    }
  )

  watch(isSuperAdmin, () => {
    if (!isSuperAdmin.value && formData.scope === 'system') {
      formData.scope = 'project'
      formData.projectId = projectStore.selectedProjectId
    }
  })

  watch(
    routeTemplateEngine,
    (engineType) => {
      if (!engineType) return
      searchForm.value.engineType = undefined
      searchParams.value = { ...searchParams.value, engineType }
      if (editorMode.value === 'browse' || editorMode.value === 'add') {
        formData.engineType = engineType
      }
      if (editorMode.value === 'browse') {
        pagination.value.current = 1
        fetchTemplates()
      }
    },
    { immediate: true }
  )

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

<style scoped lang="scss">
  .devops-pipeline-template-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .workflow-panel,
  .step-library,
  .flow-canvas,
  .node-config {
    min-height: 0;
    overflow: hidden;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .panel-title,
  .board-title {
    display: flex;
    gap: 10px;
    align-items: center;
    justify-content: space-between;
    min-height: 50px;
    padding: 10px 14px;
    border-bottom: 1px solid var(--el-border-color-lighter);

    span {
      color: var(--el-text-color-secondary);
    }
  }

  .template-card,
  .library-step,
  .flow-node {
    width: 100%;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .template-card {
    display: grid;
    grid-template-columns: 42px minmax(0, 1fr);
    gap: 8px 10px;
    align-items: start;
    min-height: 182px;
    padding: 12px;
    text-align: left;
    box-shadow: 0 10px 28px rgb(15 23 42 / 4%);

    &:hover {
      border-color: var(--el-color-primary-light-5);
      box-shadow: 0 14px 34px rgb(15 23 42 / 8%);
    }
  }

  .template-desc {
    display: -webkit-box;
    grid-column: 1 / -1;
    min-height: 36px;
    margin: 4px 0 0;
    overflow: hidden;
    line-height: 1.5;
    color: var(--el-text-color-regular);
    text-overflow: ellipsis;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }

  .template-tags {
    display: flex;
    flex-wrap: wrap;
    grid-column: 1 / -1;
    gap: 5px;
  }

  .template-card-actions {
    display: flex;
    flex-wrap: nowrap;
    grid-column: 1 / -1;
    gap: 6px;
    align-items: center;
    justify-content: flex-end;
    padding-top: 10px;
    margin-top: auto;
    border-top: 1px solid var(--el-border-color-lighter);

    :deep(.el-button) {
      height: 28px;
      padding: 0 8px;
      margin-left: 0;
    }
  }

  .template-icon,
  .step-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .template-icon {
    width: 38px;
    height: 38px;
    font-size: 22px;

    &.large {
      width: 48px;
      height: 48px;
      font-size: 28px;
    }
  }

  .template-info,
  .library-step span,
  .node-text {
    min-width: 0;

    strong,
    em {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    em {
      margin-top: 4px;
      font-style: normal;
      color: var(--el-text-color-secondary);
    }
  }

  .workflow-panel {
    display: flex;
    flex-direction: column;
  }

  .workflow-head {
    display: flex;
    gap: 14px;
    align-items: center;
    justify-content: space-between;
    padding: 14px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .workflow-title {
    display: grid;
    grid-template-columns: 54px minmax(0, 1fr);
    gap: 10px;
    align-items: center;
    min-width: 0;

    h3,
    p {
      margin: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    h3 {
      font-size: 18px;
    }

    p {
      margin-top: 6px;
      color: var(--el-text-color-secondary);
    }
  }

  .workflow-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    justify-content: flex-end;
  }

  .workflow-board {
    flex: 1;
    min-height: 0;
    padding: 12px;
  }

  .template-editor-workbench {
    display: grid;
    flex: 1;
    grid-template-columns: minmax(300px, 320px) minmax(0, 1fr);
    gap: 12px;
    min-height: 0;
    overflow: hidden;
  }

  .template-base-config {
    display: flex;
    flex-direction: column;
    min-height: 0;
    padding: 12px;
    overflow: auto;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    box-shadow: 0 10px 28px rgb(15 23 42 / 4%);
  }

  .base-config-title {
    display: grid;
    grid-template-columns: 48px minmax(0, 1fr);
    gap: 10px;
    align-items: center;
    min-width: 0;
    padding-bottom: 12px;
    margin-bottom: 12px;
    border-bottom: 1px solid var(--el-border-color-lighter);

    strong,
    span {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      font-size: 17px;
    }

    span {
      margin-top: 5px;
      color: var(--el-text-color-secondary);
    }
  }

  .base-config-head {
    display: block;
    margin-bottom: 10px;

    strong {
      display: block;
      font-size: 16px;
    }

    span {
      display: block;
      margin-top: 4px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .base-config-grid {
    display: flex;
    flex-direction: column;

    :deep(.el-form-item) {
      margin-bottom: 10px;
    }
  }

  .base-icon-row,
  .base-desc-row {
    width: 100%;
  }

  .editor-workflow-board {
    height: 100%;
    min-height: 0;
  }

  .step-library,
  .flow-canvas,
  .node-config {
    display: flex;
    flex-direction: column;
  }

  .library-list,
  .flow-scroll {
    flex: 1;
    min-height: 0;
    padding: 12px;
    overflow: auto;
  }

  .library-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .library-step {
    display: grid;
    grid-template-columns: 42px minmax(0, 1fr);
    gap: 10px;
    align-items: center;
    min-height: 70px;
    padding: 10px;
    text-align: left;
  }

  .step-icon {
    width: 42px;
    height: 42px;
    font-size: 24px;
  }

  .flow-chain {
    display: flex;
    flex-direction: column;
    align-items: center;
    min-height: 100%;
    padding: 8px 0;
  }

  .flow-node {
    display: grid;
    grid-template-columns: 42px minmax(0, 1fr) auto;
    gap: 12px;
    align-items: center;
    max-width: 620px;
    min-height: 82px;
    padding: 12px;
    text-align: left;

    &.active {
      border-color: var(--el-color-primary);
      box-shadow: 0 0 0 2px var(--el-color-primary-light-8);
    }

    &.disabled {
      opacity: 0.55;
    }
  }

  .node-index {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 42px;
    height: 42px;
    font-weight: 700;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-radius: 50%;
  }

  .node-actions {
    display: flex;
    flex-direction: column;
  }

  .flow-arrow {
    height: 30px;
    line-height: 30px;
    color: var(--el-text-color-secondary);
  }

  .node-config {
    padding-bottom: 12px;

    :deep(.el-empty) {
      flex: 1;
    }

    .el-form {
      padding: 12px;
    }
  }

  .config-actions {
    display: flex;
    justify-content: flex-end;
    padding: 0 12px;
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

      &:disabled {
        cursor: not-allowed;
        opacity: 0.65;
      }
    }
  }

  .compact-picker {
    max-height: 84px;
    overflow: auto;
  }

  :deep(.template-variable-drawer .el-drawer__header) {
    padding: 16px 20px;
    margin-bottom: 0;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  :deep(.template-variable-drawer .el-drawer__body) {
    min-height: 0;
    padding: 16px 20px;
    overflow: hidden;
  }

  :deep(.template-variable-drawer .el-table) {
    height: 100%;
  }
</style>
