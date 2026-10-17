<template>
  <ElDialog v-model="dialogVisible" title="批量同步 K8s YAML 服务" width="980px" destroy-on-close>
    <ElAlert
      type="info"
      :closable="false"
      show-icon
      title="选择仓库根目录下的多个服务目录；每个目录会创建一个服务并固定到同一 Commit。"
    />

    <ElForm label-position="top" class="bulk-form">
      <div class="form-grid">
        <ElFormItem label="代码源" required>
          <ElSelect
            v-model="form.sourceId"
            filterable
            style="width: 100%"
            :disabled="!codeSources.length || scanning || importing"
            @change="onSourceChange"
          >
            <ElOption
              v-for="source in codeSources"
              :key="source.id"
              :label="source.name"
              :value="source.id"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="仓库" required>
          <ElSelect
            v-model="form.repoId"
            filterable
            style="width: 100%"
            :loading="repositoriesLoading"
            :disabled="!form.sourceId || scanning || importing"
            @change="resetBrowse"
          >
            <ElOption
              v-for="repository in repositories"
              :key="repository.id"
              :label="repository.full_name || repository.name"
              :value="repository.id"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="分支 / Tag / Commit" required>
          <ElInput v-model="form.ref" :disabled="scanning || importing" placeholder="main" />
        </ElFormItem>
        <ElFormItem label="根目录">
          <ElInput v-model="form.rootPath" :disabled="scanning || importing" placeholder="/" />
        </ElFormItem>
      </div>
      <div class="bulk-actions">
        <ElButton :loading="scanning" :disabled="!canBrowse" @click="browseRoot">浏览目录</ElButton>
        <span class="form-hint">已选择 {{ selectedDirectories.length }} 个服务目录</span>
      </div>
    </ElForm>

    <div v-if="files.length" class="directory-browser">
      <div class="browser-heading">
        <div
          ><strong>仓库目录</strong><small>{{ browsePath || '/' }}</small></div
        >
        <ElButton v-if="browsePath" text type="primary" @click="goParent">返回上级</ElButton>
      </div>
      <ElTable :data="files" size="small" max-height="250" @row-click="onRowClick">
        <ElTableColumn width="54">
          <template #default="{ row }">
            <ElCheckbox
              v-if="row.type === 'dir'"
              :model-value="isSelected(row.path)"
              @change="toggleDirectory(row)"
              @click.stop
            />
          </template>
        </ElTableColumn>
        <ElTableColumn label="名称" min-width="360">
          <template #default="{ row }">
            <span class="file-name"
              ><Folder v-if="row.type === 'dir'" :size="16" /><FileCode2 v-else :size="16" />{{
                row.name
              }}</span
            >
          </template>
        </ElTableColumn>
        <ElTableColumn prop="type" label="类型" width="100" />
        <ElTableColumn prop="size" label="大小" width="110">
          <template #default="{ row }">{{
            row.type === 'file' ? formatSize(row.size) : '-'
          }}</template>
        </ElTableColumn>
        <ElTableColumn label="操作" width="120">
          <template #default="{ row }">
            <ElButton
              v-if="row.type === 'dir'"
              text
              type="primary"
              @click.stop="openDirectory(row.path)"
              >进入</ElButton
            >
          </template>
        </ElTableColumn>
      </ElTable>
    </div>
    <ElEmpty v-else-if="browsed" :image-size="56" description="当前目录没有可浏览内容" />

    <ElAlert
      v-if="selectedDirectories.length"
      :title="SERVICE_NAME_HINT"
      type="info"
      :closable="false"
    />
    <ElTable
      v-if="selectedDirectories.length"
      :data="selectedDirectories"
      class="selected-table"
      size="small"
    >
      <ElTableColumn label="服务名称" min-width="190">
        <template #default="{ row }"
          ><ElInput v-model="row.name" size="small" maxlength="128"
        /></template>
      </ElTableColumn>
      <ElTableColumn label="服务编码" min-width="190">
        <template #default="{ row }"><ElInput v-model="row.code" size="small" /></template>
      </ElTableColumn>
      <ElTableColumn prop="path" label="服务目录" min-width="260" />
      <ElTableColumn label="YAML 文件" width="100" align="center"
        ><template #default="{ row }">{{ row.paths.length }}</template></ElTableColumn
      >
      <ElTableColumn label="状态" width="120">
        <template #default="{ row }"
          ><ElTag :type="statusType(row.status)" size="small">{{
            statusLabel(row.status)
          }}</ElTag></template
        >
      </ElTableColumn>
    </ElTable>
    <ElAlert v-if="errors.length" class="error-list" type="warning" :closable="false">
      <template #title>部分服务未完成</template>
      <div v-for="item in errors" :key="item.path">{{ item.path }}：{{ item.message }}</div>
    </ElAlert>

    <template #footer>
      <ElButton @click="dialogVisible = false">取消</ElButton>
      <ElButton
        type="primary"
        :loading="importing"
        :disabled="!selectedDirectories.length || importing"
        @click="importAll"
        >创建并同步 {{ selectedDirectories.length }} 个服务</ElButton
      >
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { SERVICE_NAME_HINT, serviceNameError, serviceCodeFromName } from '@/utils/service-name'
  import { computed, reactive, ref, watch } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { FileCode2, Folder } from 'lucide-vue-next'
  import {
    createServiceApi,
    importServiceRevisionApi,
    listRepositoriesApi,
    listRepositoryFilesApi,
    previewServiceImportApi,
    type AxiOpsCodeSource,
    type AxiOpsRepository,
    type AxiOpsRepositoryFile,
    type AxiOpsService
  } from '@/api/axiops'

  type DirectoryRow = {
    path: string
    name: string
    code: string
    paths: string[]
    serviceId?: number
    status: 'pending' | 'running' | 'success' | 'failed'
  }
  const props = defineProps<{
    visible: boolean
    projectId: number
    codeSources: AxiOpsCodeSource[]
    existingCodes: string[]
  }>()
  const emit = defineEmits<{ 'update:visible': [value: boolean]; imported: [] }>()
  const dialogVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })
  const form = reactive({ sourceId: 0, repoId: '', ref: 'main', rootPath: '' })
  const repositories = ref<AxiOpsRepository[]>([])
  const files = ref<AxiOpsRepositoryFile[]>([])
  const selectedDirectories = ref<DirectoryRow[]>([])
  const errors = ref<Array<{ path: string; message: string }>>([])
  const batchCommit = ref('')
  const browsePath = ref('')
  const browsed = ref(false)
  const repositoriesLoading = ref(false)
  const scanning = ref(false)
  const importing = ref(false)
  const canBrowse = computed(() => Boolean(form.sourceId && form.repoId && form.ref.trim()))
  const selectedRepository = computed(() =>
    repositories.value.find((item) => item.id === form.repoId)
  )

  const reset = () => {
    form.sourceId = props.codeSources[0]?.id || 0
    form.repoId = ''
    form.ref = 'main'
    form.rootPath = ''
    repositories.value = []
    files.value = []
    selectedDirectories.value = []
    errors.value = []
    batchCommit.value = ''
    browsePath.value = ''
    browsed.value = false
    if (form.sourceId) void loadRepositories()
  }
  watch(
    () => props.visible,
    (visible) => {
      if (visible) reset()
    }
  )
  const loadRepositories = async () => {
    if (!form.sourceId) return
    repositoriesLoading.value = true
    try {
      repositories.value = (await listRepositoriesApi(form.sourceId)) || []
      if (repositories.value.length === 1) {
        form.repoId = repositories.value[0].id
        form.ref = repositories.value[0].default_branch || 'main'
      }
    } finally {
      repositoriesLoading.value = false
    }
  }
  const onSourceChange = async () => {
    form.repoId = ''
    repositories.value = []
    resetBrowse()
    await loadRepositories()
  }
  const resetBrowse = () => {
    files.value = []
    browsePath.value = ''
    browsed.value = false
    selectedDirectories.value = []
  }
  const browse = async (path: string) => {
    if (!canBrowse.value) return
    scanning.value = true
    try {
      files.value =
        (await listRepositoryFilesApi(form.sourceId, {
          repo_path:
            selectedRepository.value?.full_name || selectedRepository.value?.name || form.repoId,
          ref: form.ref.trim(),
          path
        })) || []
      browsePath.value = path
      browsed.value = true
    } finally {
      scanning.value = false
    }
  }
  const browseRoot = () => browse(form.rootPath.trim().replace(/^\/+|\/+$/g, ''))
  const openDirectory = (path: string) => browse(path)
  const goParent = () => browse(browsePath.value.split('/').slice(0, -1).join('/'))
  const isSelected = (path: string) => selectedDirectories.value.some((item) => item.path === path)
  const safeCode = serviceCodeFromName
  const uniqueCode = (value: string) => {
    const taken = new Set([
      ...props.existingCodes,
      ...selectedDirectories.value.map((item) => item.code)
    ])
    let code = safeCode(value)
    let suffix = 2
    while (taken.has(code)) code = `${safeCode(value)}-${suffix++}`
    return code
  }
  const collectYaml = async (path: string, depth = 0): Promise<string[]> => {
    if (depth > 2) return []
    const items =
      (await listRepositoryFilesApi(form.sourceId, {
        repo_path:
          selectedRepository.value?.full_name || selectedRepository.value?.name || form.repoId,
        ref: form.ref.trim(),
        path
      })) || []
    const result = items
      .filter((item) => item.type === 'file' && /\.(ya?ml)$/i.test(item.path))
      .map((item) => item.path)
    for (const item of items.filter((entry) => entry.type === 'dir'))
      result.push(...(await collectYaml(item.path, depth + 1)))
    return Array.from(new Set(result)).slice(0, 50)
  }
  const toggleDirectory = async (row: AxiOpsRepositoryFile) => {
    if (isSelected(row.path)) {
      selectedDirectories.value = selectedDirectories.value.filter((item) => item.path !== row.path)
      return
    }
    scanning.value = true
    try {
      const paths = await collectYaml(row.path)
      if (!paths.length) {
        ElMessage.warning(`${row.path} 下未找到 YAML 文件`)
        return
      }
      const name = row.name.replace(/\/$/, '')
      selectedDirectories.value.push({
        path: row.path,
        name,
        code: uniqueCode(name),
        paths,
        status: 'pending'
      })
    } finally {
      scanning.value = false
    }
  }
  const onRowClick = (row: AxiOpsRepositoryFile) => {
    if (row.type === 'dir') void toggleDirectory(row)
  }
  const formatSize = (value: number) =>
    value > 1024 * 1024
      ? `${(value / 1024 / 1024).toFixed(1)} MB`
      : `${Math.max(1, Math.round(value / 1024))} KB`
  const statusLabel = (status: DirectoryRow['status']) =>
    ({ pending: '待创建', running: '处理中', success: '已完成', failed: '失败' })[status]
  const statusType = (status: DirectoryRow['status']) =>
    ({ pending: 'info', running: 'warning', success: 'success', failed: 'danger' })[status] as
      | 'info'
      | 'warning'
      | 'success'
      | 'danger'
  const importAll = async () => {
    if (!selectedRepository.value || !selectedDirectories.value.length) return
    const invalid = selectedDirectories.value.find(
      (row) => !row.serviceId && serviceNameError(row.name)
    )
    if (invalid)
      return void ElMessage.warning(`${invalid.name || invalid.path}：${SERVICE_NAME_HINT}`)
    try {
      await ElMessageBox.confirm(
        `将创建 ${selectedDirectories.value.length} 个服务并生成 Revision，是否继续？`,
        '批量同步确认',
        { type: 'warning' }
      )
    } catch {
      return
    }
    importing.value = true
    errors.value = []
    const repoPath = selectedRepository.value.full_name || selectedRepository.value.name
    // Resolve the branch once, then pin every service to the same immutable Commit.
    let pinnedCommit = batchCommit.value
    for (const row of selectedDirectories.value) {
      if (row.status === 'success') continue
      row.status = 'running'
      try {
        if (!row.serviceId) {
          const created = await createServiceApi(props.projectId, {
            name: row.name,
            code: row.code,
            description: `从 ${row.path} 批量同步`,
            source_type: 'git',
            repo_url: selectedRepository.value.clone_url,
            repo_branch: form.ref.trim(),
            language: 'kubernetes',
            framework: '',
            image_name: '',
            code_source_id: form.sourceId,
            repository_id: selectedRepository.value.id
          })
          row.serviceId = created.id
        }
        const service = {
          id: row.serviceId,
          project_id: props.projectId,
          name: row.name,
          code: row.code,
          delivery_type: 'k8s_yaml',
          source_type: 'git',
          code_source_id: form.sourceId
        } as AxiOpsService
        const request = {
          code_source_id: form.sourceId,
          repo_path: repoPath,
          ref: pinnedCommit || form.ref.trim(),
          paths: row.paths
        }
        const preview = await previewServiceImportApi(service, request)
        pinnedCommit ||= preview.commit_id
        batchCommit.value ||= preview.commit_id
        await importServiceRevisionApi(service, { ...request, expected_commit: preview.commit_id })
        row.status = 'success'
      } catch (error: any) {
        row.status = 'failed'
        errors.value.push({
          path: row.path,
          message: error?.message || '服务创建或 Revision 导入失败'
        })
      }
    }
    importing.value = false
    if (selectedDirectories.value.every((item) => item.status === 'success')) {
      ElMessage.success('批量同步完成')
      emit('imported')
      dialogVisible.value = false
    } else ElMessage.warning('批量同步部分完成，请处理失败项后重试')
  }
</script>

<style scoped lang="scss">
  .bulk-form {
    margin-top: 18px;
  }
  .form-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0 14px;
  }
  .bulk-actions,
  .browser-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }
  .form-hint,
  .browser-heading small {
    color: var(--art-text-gray-500);
    font-size: 12px;
  }
  .directory-browser {
    margin-top: 14px;
    border: 1px solid var(--art-border-color);
    border-radius: 4px;
    padding: 12px;
  }
  .browser-heading {
    margin-bottom: 8px;
  }
  .browser-heading div {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .file-name {
    display: inline-flex;
    align-items: center;
    gap: 7px;
  }
  .selected-table {
    margin-top: 16px;
  }
  .error-list {
    margin-top: 14px;
  }
  @media (max-width: 700px) {
    .form-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
