<template>
  <div class="cicd-projects">
    <ArtPageHeader title="持续交付" subtitle="项目、服务、环境与工作流">
      <template #actions>
        <ElButton v-if="canCreate" type="primary" :icon="Plus" @click="openCreate">
          新建项目
        </ElButton>
      </template>
    </ArtPageHeader>

    <div class="toolbar">
      <div class="project-tabs" role="tablist" aria-label="项目分类">
        <button
          v-for="tab in projectTabs"
          :key="tab.value"
          type="button"
          :class="['project-tab', { 'is-active': projectCategory === tab.value }]"
          @click="projectCategory = tab.value"
        >
          {{ tab.label }}
        </button>
      </div>
      <ElInput
        v-model="filters.keyword"
        clearable
        :prefix-icon="Search"
        placeholder="搜索项目名称或编码"
        @keyup.enter="fetchProjects"
        @clear="fetchProjects"
      />
      <ElSelect v-model="filters.deliveryType" clearable placeholder="交付类型">
        <ElOption
          v-for="item in deliveryOptions"
          :key="item.value"
          :label="item.label"
          :value="item.value"
        />
      </ElSelect>
      <ElSelect v-model="filters.status" clearable placeholder="状态" @change="fetchProjects">
        <ElOption label="启用" value="active" />
        <ElOption label="停用" value="disabled" />
        <ElOption label="已删除" value="deleted" />
      </ElSelect>
      <div class="view-toggle" role="group" aria-label="展示方式">
        <button
          v-for="v in [
            { value: 'card', icon: LayoutGrid, label: '卡片' },
            { value: 'list', icon: List, label: '列表' }
          ]"
          :key="v.value"
          type="button"
          :class="['view-toggle-btn', { 'is-active': viewMode === v.value }]"
          :title="v.label"
          @click="setViewMode(v.value)"
        >
          <component :is="v.icon" :size="14" />
          <span>{{ v.label }}</span>
        </button>
      </div>
      <ElButton :icon="RefreshCw" :loading="loading" circle title="刷新" @click="fetchProjects" />
    </div>

    <div v-loading="loading" class="project-list" :class="`is-${viewMode}`">
      <template v-if="viewMode === 'card'">
        <button
          v-for="project in filteredProjects"
          :key="project.id"
          class="project-card"
          type="button"
          @click="openProject(project.id)"
        >
        <span class="project-card-head">
          <span class="type-mark" :class="`is-${project.delivery_type}`">
            <ArtDeliveryIcon :type="project.delivery_type" :size="20" />
          </span>
          <span class="project-card-title">
            <strong>{{ project.name }}</strong>
            <small>{{ project.code }}</small>
          </span>
          <ElDropdown class="project-card-menu" trigger="click" @click.stop>
            <ElButton text :icon="MoreVertical" aria-label="项目操作" @click.stop />
            <template #dropdown>
              <ElDropdownMenu>
                <ElDropdownItem @click="openProject(project.id)">进入项目</ElDropdownItem>
                <ElDropdownItem
                  v-if="canDeleteProject(project.id)"
                  divided
                  :icon="Trash2"
                  @click="openDelete(project)"
                  >{{ project.status === 'deleted' ? '彻底删除' : '删除项目' }}</ElDropdownItem
                >
              </ElDropdownMenu>
            </template>
          </ElDropdown>
        </span>
        <span class="project-card-description">{{ project.description || '暂无描述' }}</span>
        <span class="project-card-tags">
          <ElTag
            v-if="initStatusLabel(project)"
            size="small"
            type="warning"
            effect="plain"
            class="init-tag"
          >
            {{ initStatusLabel(project) }}
          </ElTag>
          <ElTag size="small" effect="plain" :type="deliveryMeta(project.delivery_type).tag">
            {{ deliveryMeta(project.delivery_type).label }}
          </ElTag>
          <ElTag size="small" effect="plain" :type="statusMeta(project.status).tag">
            {{ statusMeta(project.status).label }}
          </ElTag>
        </span>
        <span class="project-card-footer">
          <span><UserRound :size="13" /> {{ ownerLabel(project) }}</span>
          <span><Clock3 :size="13" /> {{ formatDate(project.updated_at) }}</span>
        </span>
      </button>
      </template>

      <template v-else-if="viewMode === 'list'">
        <button
          v-for="project in filteredProjects"
          :key="project.id"
          class="project-row"
          type="button"
          @click="openProject(project.id)"
        >
          <span class="type-mark" :class="`is-${project.delivery_type}`">
            <ArtDeliveryIcon :type="project.delivery_type" :size="20" />
          </span>
          <span class="project-row-main">
            <strong>{{ project.name }}</strong>
            <small>{{ project.code }}</small>
            <span class="project-row-desc">{{ project.description || '暂无描述' }}</span>
          </span>
          <span class="project-row-tags">
            <ElTag
              v-if="initStatusLabel(project)"
              size="small"
              type="warning"
              effect="plain"
              class="init-tag"
            >
              {{ initStatusLabel(project) }}
            </ElTag>
            <ElTag size="small" effect="plain" :type="deliveryMeta(project.delivery_type).tag">
              {{ deliveryMeta(project.delivery_type).label }}
            </ElTag>
            <ElTag size="small" effect="plain" :type="statusMeta(project.status).tag">
              {{ statusMeta(project.status).label }}
            </ElTag>
          </span>
          <span class="project-row-meta">
            <span><UserRound :size="13" /> {{ ownerLabel(project) }}</span>
            <span><Clock3 :size="13" /> {{ formatDate(project.updated_at) }}</span>
          </span>
        </button>
      </template>

      <ElEmpty v-if="!loading && filteredProjects.length === 0" description="暂无可访问项目" />
    </div>

    <footer class="page-footer">
      <span>共 {{ pagination.total }} 个项目</span>
      <ElPagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        layout="prev, pager, next"
        :total="pagination.total"
        @current-change="fetchProjects"
      />
    </footer>

    <ElDialog
      v-model="deleteVisible"
      :title="preserveHistory ? '删除项目并保留历史' : '彻底删除项目'"
      width="480px"
      :close-on-click-modal="false"
      :close-on-press-escape="!deleting"
      :show-close="!deleting"
      :before-close="closeDelete"
      class="project-delete-dialog"
    >
      <div v-loading="deleteChecking" class="project-delete-content">
        <p class="project-delete-identity">
          <strong>{{ deleteTarget?.name }}</strong>
          <span>{{ deleteTarget?.code }}</span>
        </p>
        <template v-if="deleteCheck">
          <ElAlert
            v-if="!deleteCheck.can_delete"
            type="warning"
            :closable="false"
            show-icon
            title="此项目暂不能删除"
            description="请等待以下任务结束，或取消任务后重新检查。已结束的记录不会阻止删除。"
          />
          <ElAlert
            v-else
            type="warning"
            :closable="false"
            show-icon
            :title="preserveHistory ? '停止使用项目，保留历史' : '彻底删除后无法恢复'"
            :description="
              preserveHistory
                ? '配置、执行记录和日志保留在“已删除”中，项目编码可以重新使用。'
                : '清除本项目的服务、环境配置、执行记录和保存的日志，名称和编码可以重新使用。'
            "
          />
          <p class="project-delete-boundary">
            不删除集群中运行的应用、命名空间、PVC 数据、Git 仓库和镜像；已部署应用请先下线。
          </p>
          <ElCheckbox
            v-if="deleteTarget?.status !== 'deleted' && deleteCheck.can_delete"
            v-model="preserveHistory"
            :disabled="deleting"
            @change="deleteName = ''"
            >保留配置与执行历史</ElCheckbox
          >
          <ElTable
            :data="deleteCheck.can_delete ? deleteCheck.items : deleteCheck.blockers"
            size="small"
            max-height="220"
          >
            <ElTableColumn
              prop="name"
              :label="
                deleteCheck.can_delete
                  ? preserveHistory
                    ? '保留的数据'
                    : '清除的数据'
                  : '未结束任务'
              "
            />
            <ElTableColumn prop="count" label="数量" width="70" align="right" />
          </ElTable>
          <ElForm v-if="deleteCheck.can_delete" label-position="top" @submit.prevent="submitDelete">
            <ElFormItem label="输入完整项目名称确认删除">
              <ElInput
                v-model="deleteName"
                :disabled="deleting"
                :placeholder="deleteCheck.project_name"
                autocomplete="off"
              />
            </ElFormItem>
          </ElForm>
        </template>
        <ElAlert
          v-else-if="!deleteChecking"
          type="error"
          :closable="false"
          title="无法完成删除检查，请重试"
        />
      </div>
      <template #footer>
        <ElButton :disabled="deleting" @click="deleteVisible = false">取消</ElButton>
        <ElButton
          v-if="!deleteCheck || !deleteCheck.can_delete"
          :icon="RefreshCw"
          :loading="deleteChecking"
          @click="refreshDeletionCheck"
          >重新检查</ElButton
        >
        <ElButton
          v-else
          type="danger"
          :icon="Trash2"
          :loading="deleting"
          :disabled="!deleteConfirmed"
          @click="submitDelete"
          >{{ preserveHistory ? '删除并保留历史' : '彻底删除' }}</ElButton
        >
      </template>
    </ElDialog>

    <ElDialog
      v-model="createVisible"
      fullscreen
      :show-close="false"
      destroy-on-close
      class="create-project-dialog"
    >
      <template #header>
        <div class="create-project-header">
          <div class="create-project-brand"><ArtLogo :size="30" /><strong>AxiOps</strong></div>
          <button type="button" aria-label="关闭" @click="createVisible = false">
            <X :size="20" />
          </button>
        </div>
      </template>

      <main class="create-project-page">
        <div class="create-project-content">
          <h1>开始新建项目</h1>
          <ElForm ref="formRef" :model="form" :rules="rules" label-position="top">
            <ElFormItem label="项目名称" prop="name">
              <ElInput
                v-model="form.name"
                maxlength="64"
                placeholder="请输入项目名称"
                @input="syncProjectCode"
              />
            </ElFormItem>

            <ElFormItem prop="code">
              <template #label>
                <span class="project-key-label">
                  项目主键
                  <ElTooltip content="项目资源的全局唯一标识，创建后不可修改" placement="top">
                    <CircleHelp :size="14" />
                  </ElTooltip>
                  <ElButton text type="primary" @click="codeEditable = !codeEditable">
                    {{ codeEditable ? '完成' : '编辑' }}
                  </ElButton>
                </span>
              </template>
              <ElInput
                v-model="form.code"
                :disabled="!codeEditable"
                maxlength="64"
                placeholder="根据项目名称自动生成"
              />
            </ElFormItem>

            <ElFormItem label="描述信息">
              <ElInput
                v-model="form.description"
                type="textarea"
                :rows="3"
                maxlength="500"
                placeholder="请输入描述信息"
              />
            </ElFormItem>

            <ElFormItem label="项目类型" prop="delivery_type" class="project-types-item">
              <div class="delivery-selector" role="radiogroup" aria-label="项目类型">
                <button
                  v-for="option in deliveryOptions"
                  :key="option.value"
                  type="button"
                  :class="['delivery-option', { selected: form.delivery_type === option.value }]"
                  role="radio"
                  :aria-checked="form.delivery_type === option.value"
                  @click="form.delivery_type = option.value"
                >
                  <span class="delivery-option-icon"
                    ><ArtDeliveryIcon :type="option.value" :size="21"
                  /></span>
                  <span>
                    <strong>{{ option.label }}项目</strong>
                    <small>{{ option.description }}</small>
                  </span>
                  <span class="delivery-option-check">{{
                    form.delivery_type === option.value ? '✓' : ''
                  }}</span>
                </button>
              </div>
            </ElFormItem>
          </ElForm>
        </div>
      </main>

      <template #footer>
        <div class="create-project-footer">
          <ElButton @click="createVisible = false">取消</ElButton>
          <ElButton type="primary" :loading="submitting" @click="submitCreate">立即创建</ElButton>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onActivated, onMounted, reactive, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
  import {
    CircleHelp,
    Clock3,
    Container,
    FileCode2,
    LayoutGrid,
    List,
    MoreVertical,
    PackageOpen,
    Plus,
    RefreshCw,
    Search,
    Trash2,
    UserRound,
    X
  } from 'lucide-vue-next'
  import {
    createProjectApi,
    checkProjectDeletionApi,
    deleteProjectApi,
    listProjectsApi,
    type AxiOpsProject,
    type CreateProjectInput,
    type ProjectDeletionCheck,
    type DeliveryType
  } from '@/api/axiops'
  import { useAxiOpsAccess } from '@/composables/useAxiOpsAccess'
  import ArtPageHeader from '@/components/core/base/art-page-header/index.vue'
  import ArtDeliveryIcon from '@/components/core/base/art-delivery-icon/index.vue'

  defineOptions({ name: 'CICDProjects' })

  const router = useRouter()
  const access = useAxiOpsAccess()
  const loading = ref(false)
  const submitting = ref(false)
  const createVisible = ref(false)
  const codeEditable = ref(false)
  const deleteVisible = ref(false)
  const deleting = ref(false)
  const deleteChecking = ref(false)
  const deleteTarget = ref<AxiOpsProject>()
  const deleteCheck = ref<ProjectDeletionCheck>()
  const deleteName = ref('')
  const preserveHistory = ref(false)
  let deleteRequest = 0
  const deleteConfirmed = computed(
    () =>
      !deleteChecking.value &&
      deleteCheck.value?.can_delete === true &&
      deleteName.value !== '' &&
      deleteName.value === deleteCheck.value.project_name
  )
  const formRef = ref<FormInstance>()
  const projects = ref<AxiOpsProject[]>([])
  const projectCategory = ref('all')
  const filters = reactive({ keyword: '', deliveryType: '' as DeliveryType | '', status: '' })
  const pagination = reactive({ page: 1, pageSize: 20, total: 0 })
  type ProjectView = 'card' | 'list'
  const viewMode = ref<ProjectView>('card')
  try {
    if (localStorage.getItem('axiops-projects-view') === 'list') viewMode.value = 'list'
  } catch {
    // Browsers can disable storage; the project list must still load.
  }
  const setViewMode = (mode: ProjectView) => {
    viewMode.value = mode
    try {
      localStorage.setItem('axiops-projects-view', mode)
    } catch {
      // Keep the selected view usable for this visit without persistence.
    }
  }
  let listRequest = 0
  let mounted = false

  const deliveryOptions = [
    {
      value: 'k8s_yaml' as const,
      label: 'K8s YAML',
      description: 'Git YAML 服务与命名空间环境',
      icon: FileCode2,
      tag: 'primary' as const
    },
    {
      value: 'helm' as const,
      label: 'Helm Chart',
      description: 'Chart、分层 values 与 release',
      icon: PackageOpen,
      tag: 'warning' as const
    },
    {
      value: 'host' as const,
      label: '主机应用',
      description: '二进制制品、主机组与滚动发布',
      icon: Container,
      tag: 'success' as const
    }
  ]
  const projectTabs = [
    { value: 'all', label: '全部' },
    { value: 'k8s_yaml', label: 'K8s YAML' },
    { value: 'helm', label: 'Helm Chart' },
    { value: 'host', label: '主机应用' }
  ]

  const form = reactive<CreateProjectInput>({
    name: '',
    code: '',
    description: '',
    delivery_type: 'k8s_yaml'
  })
  const rules: FormRules = {
    name: [{ required: true, message: '请输入项目名称', trigger: 'blur' }],
    code: [
      { required: true, message: '请输入项目编码', trigger: 'blur' },
      {
        pattern: /^[a-z][a-z0-9-]{1,63}$/,
        message: '使用小写字母、数字和连字符，并以字母开头',
        trigger: 'blur'
      }
    ],
    delivery_type: [{ required: true, message: '请选择交付类型', trigger: 'change' }]
  }

  const canCreate = computed(() => access.can('project', 'create'))
  const canDeleteProject = (id: number) => access.can('project', 'delete', { type: 'project', id })

  const closeDelete = (done: () => void) => {
    if (!deleting.value) done()
  }
  const refreshDeletionCheck = async () => {
    const target = deleteTarget.value
    if (!target || deleting.value) return
    const request = ++deleteRequest
    deleteChecking.value = true
    deleteCheck.value = undefined
    deleteName.value = ''
    try {
      const result = await checkProjectDeletionApi(target.id)
      if (request === deleteRequest && deleteTarget.value?.id === target.id)
        deleteCheck.value = result
    } catch {
      /* Request errors are displayed by the HTTP client. */
    } finally {
      if (request === deleteRequest) deleteChecking.value = false
    }
  }
  const openDelete = (project: AxiOpsProject) => {
    if (deleting.value || !canDeleteProject(project.id)) return
    deleteTarget.value = project
    preserveHistory.value = false
    deleteVisible.value = true
    void refreshDeletionCheck()
  }
  const submitDelete = async () => {
    if (deleting.value || !deleteConfirmed.value || !deleteTarget.value) return
    const id = deleteTarget.value.id
    deleting.value = true
    try {
      await deleteProjectApi(id, deleteName.value, !preserveHistory.value)
      deleteVisible.value = false
      projects.value = projects.value.filter((project) => project.id !== id)
      pagination.total = Math.max(0, pagination.total - 1)
      if (!projects.value.length && pagination.page > 1) pagination.page--
      ElMessage.success(
        preserveHistory.value
          ? '项目已删除，历史已保留'
          : '项目已彻底删除，可使用原名称和编码重新创建'
      )
    } catch {
      deleting.value = false
      await refreshDeletionCheck()
      return
    } finally {
      deleting.value = false
    }
    await fetchProjects()
  }
  const filteredProjects = computed(() =>
    projects.value.filter((item) => {
      const matchesFilter = filters.deliveryType
        ? item.delivery_type === filters.deliveryType
        : true
      const matchesCategory =
        projectCategory.value === 'all' || item.delivery_type === projectCategory.value
      return matchesFilter && matchesCategory
    })
  )

  const deliveryMeta = (type: string) =>
    deliveryOptions.find((item) => item.value === type) || {
      label: '待确认',
      icon: FileCode2,
      tag: 'info' as const
    }
  const statusMeta = (status: string) => {
    if (status === 'deleted') return { label: '已删除 · 查看历史', tag: 'info' as const }
    if (status === 'active') return { label: '启用', tag: 'success' as const }
    return { label: '停用', tag: 'info' as const }
  }
  const initStatusLabel = (project: AxiOpsProject) => {
    if (project.status === 'deleted') return ''
    if (project.initialization_status && project.initialization_status !== 'completed') return '待初始化'
    return ''
  }
  const ownerLabel = (project: AxiOpsProject) => {
    const value = project.owner_user_id
    if (value == null || value === '' || /^\d+$/.test(String(value))) return '平台管理员'
    return String(value)
  }

  const fetchProjects = async () => {
    const request = ++listRequest
    loading.value = true
    try {
      const result = await listProjectsApi({
        keyword: filters.keyword || undefined,
        status: filters.status || undefined,
        page: pagination.page,
        page_size: pagination.pageSize
      })
      if (request !== listRequest) return
      projects.value = result.list || []
      pagination.total = result.total || 0
    } finally {
      if (request === listRequest) loading.value = false
    }
  }

  const openCreate = () => {
    Object.assign(form, { name: '', code: '', description: '', delivery_type: 'k8s_yaml' })
    codeEditable.value = false
    createVisible.value = true
  }

  const syncProjectCode = (value: string) => {
    if (codeEditable.value) return
    form.code = value
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '')
      .slice(0, 64)
  }

  const submitCreate = async () => {
    if (submitting.value) return
    if (!(await formRef.value?.validate())) return
    submitting.value = true
    try {
      const result = await createProjectApi(form)
      ElMessage.success('项目已创建')
      createVisible.value = false
      pagination.page = 1
      filters.keyword = ''
      filters.status = ''
      filters.deliveryType = ''
      projectCategory.value = 'all'
      try {
        await fetchProjects()
      } catch {
        /* Creation succeeded; returning to the list retries the read. */
      }
      await router.push(`/cicd/projects/${result.id}?init=1&tab=services`)
    } finally {
      submitting.value = false
    }
  }

  const openProject = (id: number) => router.push(`/cicd/projects/${id}`)
  const formatDate = (value: string) => (value ? new Date(value).toLocaleDateString('zh-CN') : '-')

  onMounted(async () => {
    try {
      await access.load()
      await fetchProjects()
    } finally {
      mounted = true
    }
  })
  onActivated(() => {
    if (mounted) void fetchProjects().catch(() => {})
  })
</script>

<style scoped lang="scss">
  .project-delete-content {
    min-height: 120px;
    display: grid;
    gap: 16px;
  }
  .project-delete-identity {
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
    overflow-wrap: anywhere;
  }
  .project-delete-identity span {
    font-size: 12px;
    color: var(--art-text-gray-600);
  }
  .project-delete-boundary {
    margin: 0;
    color: var(--art-text-gray-600);
    font-size: 12px;
    line-height: 1.6;
  }
  :deep(.project-delete-dialog) {
    max-width: calc(100vw - 32px);
  }
  .cicd-projects {
    min-height: 100%;
    padding: 20px;
    background: var(--art-main-bg-color);
    color: var(--art-text-gray-900);
  }
  .toolbar {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
    padding: 12px 0;
    border-top: 1px solid var(--art-border-color);
    border-bottom: 1px solid var(--art-border-color);
  }
  .project-tabs {
    display: flex;
    align-items: center;
    gap: 4px;
    flex: 1 0 100%;
    order: -1;
  }
  .project-tab {
    min-height: 30px;
    padding: 0 14px;
    color: var(--art-text-gray-600);
    font-size: 12px;
    background: transparent;
    border: 1px solid transparent;
    border-radius: 4px;
    cursor: pointer;
  }
  .project-tab:hover {
    color: var(--el-color-primary);
  }
  .project-tab.is-active {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary-light-7);
  }
  .toolbar > .el-input {
    width: min(100%, 360px);
  }
  .toolbar > .el-select {
    width: 180px;
  }
  .project-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
    min-height: 340px;
    padding: 16px 0 22px;
  }
  .project-list.is-card {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
    gap: 14px;
  }
  .project-card {
    width: 100%;
    min-height: 168px;
    display: flex;
    flex-direction: column;
    align-items: stretch;
    gap: 14px;
    padding: 16px;
    color: inherit;
    text-align: left;
    background: var(--art-bg-color);
    border: 1px solid var(--art-border-color);
    border-radius: 8px;
    cursor: pointer;
    transition:
      border-color 0.18s ease,
      box-shadow 0.18s ease,
      transform 0.18s ease;
  }
  .project-card:hover {
    border-color: var(--el-color-primary-light-5);
    box-shadow: 0 8px 20px rgb(15 23 42 / 8%);
    transform: translateY(-1px);
  }
  .project-card-head {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .project-card-title {
    min-width: 0;
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 3px;
  }
  .project-card-title strong,
  .project-card-title small,
  .project-card-description {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .project-card-title strong {
    font-size: 15px;
  }
  .project-card-title small {
    color: var(--art-text-gray-500);
    font-size: 11px;
  }
  .project-card-menu {
    flex: 0 0 auto;
  }
  .project-card-description {
    min-height: 38px;
    color: var(--art-text-gray-600);
    font-size: 12px;
    line-height: 19px;
    white-space: normal;
    display: -webkit-box;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }
  .project-card-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
  .project-row-tags .init-tag,
  .project-card-tags .init-tag {
    white-space: normal;
    line-height: 1.4;
  }
  .project-card-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    margin-top: auto;
    color: var(--art-text-gray-500);
    font-size: 11px;
  }
  .project-card-footer span {
    display: inline-flex;
    align-items: center;
    min-width: 0;
    gap: 4px;
  }
  .project-card-footer span:last-child {
    white-space: nowrap;
  }
  .view-toggle {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    padding: 3px;
    border: 1px solid var(--art-border-color);
    border-radius: 6px;
    background: var(--art-bg-color);
  }
  .view-toggle-btn {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    height: 30px;
    padding: 0 10px;
    border: 0;
    border-radius: 4px;
    background: transparent;
    color: var(--art-text-gray-500);
    font-size: 12px;
    cursor: pointer;
  }
  .view-toggle-btn:hover {
    color: var(--el-color-primary);
  }
  .view-toggle-btn.is-active {
    background: var(--el-color-primary);
    color: #fff;
  }
  .project-row {
    display: grid;
    grid-template-columns: 44px 1.2fr 1fr auto;
    align-items: center;
    gap: 14px;
    width: 100%;
    padding: 14px 16px;
    border: 1px solid var(--art-border-color);
    border-radius: 8px;
    background: var(--art-bg-color);
    color: inherit;
    text-align: left;
    cursor: pointer;
    transition: border-color 0.18s ease, box-shadow 0.18s ease;
  }
  .project-row:hover {
    border-color: var(--el-color-primary-light-5);
    box-shadow: 0 4px 12px rgb(15 23 42 / 6%);
  }
  .project-row-main {
    display: flex;
    min-width: 0;
    flex-direction: column;
    gap: 2px;
  }
  .project-row-main strong {
    overflow: hidden;
    font-size: 15px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .project-row-main small {
    color: var(--art-text-gray-500);
    font-size: 11px;
  }
  .project-row-desc {
    min-height: 17px;
    color: var(--art-text-gray-600);
    font-size: 12px;
    line-height: 17px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .project-row-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    min-width: 0;
  }
  .project-row-meta {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 130px;
    color: var(--art-text-gray-500);
    font-size: 11px;
  }
  .project-row-meta span {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    white-space: nowrap;
  }
  .type-mark {
    width: 38px;
    height: 38px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: 1px solid rgb(255 255 255 / 28%);
    border-radius: 10px;
    color: #fff;
    box-shadow: 0 5px 12px rgb(15 23 42 / 12%);
  }
  .is-k8s_yaml {
    background: linear-gradient(135deg, #0f766e, #2563eb);
  }
  .is-helm {
    background: linear-gradient(135deg, #7c3aed, #2563eb);
  }
  .is-host {
    background: linear-gradient(135deg, #0891b2, #0f766e);
  }
  .is-unclassified {
    background: linear-gradient(135deg, #64748b, #334155);
  }
  .type-mark.is-mini {
    width: 26px;
    height: 26px;
    border-radius: 6px;
    box-shadow: none;
  }
  .project-identity {
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .project-identity strong,
  .project-identity small,
  .project-description {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .project-identity strong {
    font-size: 14px;
  }
  .project-identity small,
  .updated-at {
    margin-top: 3px;
    color: var(--art-text-gray-500);
    font-size: 12px;
  }
  .project-description {
    color: var(--art-text-gray-700);
    font-size: 13px;
  }
  .row-arrow {
    color: var(--art-text-gray-400);
  }
  .page-footer {
    min-height: 54px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    color: var(--art-text-gray-500);
    font-size: 12px;
  }
  .delivery-selector {
    width: 100%;
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 10px;
  }
  .delivery-option {
    min-height: 88px;
    display: grid;
    grid-template-columns: 38px 1fr 18px;
    align-items: center;
    gap: 10px;
    padding: 12px;
    text-align: left;
    color: inherit;
    background: var(--art-main-bg-color);
    border: 1px solid var(--art-border-color);
    border-radius: 6px;
    cursor: pointer;
  }
  .delivery-option.selected {
    border-color: var(--el-color-primary);
    box-shadow: inset 0 0 0 1px var(--el-color-primary);
  }
  .delivery-option span:nth-child(2) {
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .delivery-option small {
    margin-top: 4px;
    color: var(--art-text-gray-500);
    line-height: 17px;
  }
  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 14px;
  }
  :global(.create-project-dialog .el-dialog__header) {
    padding: 0;
    margin: 0;
    border-bottom: 1px solid #e4e7ed;
  }
  :global(.create-project-dialog .el-dialog__body) {
    height: calc(100vh - 121px);
    padding: 0;
    overflow: auto;
    background: #fff;
  }
  :global(.create-project-dialog .el-dialog__footer) {
    height: 56px;
    padding: 0;
    border-top: 1px solid #e4e7ed;
  }
  .create-project-header {
    height: 64px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 24px;
    background: #fff;
  }
  .create-project-header button {
    width: 34px;
    height: 34px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0;
    color: #606266;
    background: transparent;
    border: 0;
    cursor: pointer;
  }
  .create-project-brand {
    display: flex;
    align-items: center;
    gap: 9px;
    color: #303133;
    font-size: 17px;
  }
  .create-project-page {
    min-height: 100%;
    padding: 54px 24px 72px;
  }
  .create-project-content {
    width: min(100%, 680px);
    margin: 0 auto;
  }
  .create-project-content h1 {
    margin: 0 0 38px;
    color: #303133;
    font-size: 28px;
    font-weight: 400;
    line-height: 40px;
    letter-spacing: 0;
  }
  .create-project-content :deep(.el-form-item) {
    margin-bottom: 25px;
  }
  .project-key-label,
  .feature-label {
    display: inline-flex;
    align-items: center;
    gap: 5px;
  }
  .project-key-label .el-button {
    height: auto;
    padding: 0 3px;
  }
  .project-features :deep(.el-form-item__content) {
    display: block;
  }
  .feature-row {
    min-height: 48px;
    display: grid;
    grid-template-columns: 150px minmax(0, 1fr);
    align-items: center;
    gap: 20px;
    border-bottom: 1px solid #ebeef5;
  }
  .feature-label {
    color: #606266;
    font-size: 13px;
  }
  .create-project-footer {
    height: 56px;
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
    padding: 0 26px;
    background: #fff;
  }
  @media (max-width: 900px) {
    .toolbar {
      align-items: stretch;
    }
    .delivery-selector {
      grid-template-columns: 1fr;
    }
  }
  @media (max-width: 600px) {
    .cicd-projects {
      padding: 14px;
    }
    .page-header {
      align-items: flex-start;
    }
    .page-header h1 {
      font-size: 21px;
    }
    .toolbar,
    .form-grid {
      grid-template-columns: 1fr;
    }
    .toolbar > :last-child {
      display: none;
    }
  .project-list.is-card {
    grid-template-columns: 1fr;
  }
    .create-project-page {
      padding: 28px 18px 54px;
    }
    .create-project-content h1 {
      margin-bottom: 28px;
      font-size: 24px;
    }
    .feature-row {
      grid-template-columns: 1fr;
      gap: 8px;
      padding: 12px 0;
    }
  }
</style>
