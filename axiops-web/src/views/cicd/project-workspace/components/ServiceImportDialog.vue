<template>
  <ElDialog
    v-model="dialogVisible"
    :title="`${service?.name || '服务'} · 导入定义`"
    width="720px"
    destroy-on-close
  >
    <ElAlert
      v-if="!service?.code_source_id"
      title="服务未绑定代码源，请重新创建或在服务设置中绑定代码源。"
      type="warning"
      :closable="false"
      show-icon
    />
    <ElForm v-if="service" label-position="top" class="import-form">
      <div class="form-grid">
        <ElFormItem label="仓库路径"
          ><ElInput v-model="form.repo_path" placeholder="team/repository"
        /></ElFormItem>
        <ElFormItem label="分支、Tag 或 Commit"
          ><ElInput v-model="form.ref" placeholder="main"
        /></ElFormItem>
      </div>
      <ElFormItem v-if="service.delivery_type === 'helm'" label="Chart 目录"
        ><ElInput v-model="form.chart_path" placeholder="deploy/chart"
      /></ElFormItem>
      <ElFormItem v-if="service.delivery_type === 'host'" label="主机服务定义"
        ><ElInput v-model="form.spec_path" placeholder="deploy/host-service.yaml"
      /></ElFormItem>
      <ElFormItem label="需要读取的文件"
        ><ElInput
          v-model="form.paths"
          type="textarea"
          :rows="6"
          :placeholder="pathPlaceholder"
        /><small
          >每行一个仓库相对路径，最多 50 个文件。预览和正式导入会固定到同一个 Commit。</small
        ></ElFormItem
      >
      <section class="repository-browser">
        <div class="browser-heading">
          <span>仓库目录</span><small>{{ browsePath || '/' }}</small>
          <ElButton size="small" :loading="filesLoading" @click="loadFiles">浏览目录</ElButton>
          <ElButton
            v-if="repoFiles.some((item) => item.type === 'file')"
            size="small"
            type="success"
            @click="addCurrentDirectoryFiles"
            >加入当前目录 YAML</ElButton
          >
        </div>
        <ElTable v-if="repoFiles.length" :data="repoFiles" size="small" max-height="190">
          <ElTableColumn label="名称" min-width="260">
            <template #default="{ row }">
              <ElButton text type="primary" @click="selectFile(row)">
                {{ row.type === 'dir' ? '目录：' : '文件：' }}{{ row.name }}
              </ElButton>
            </template>
          </ElTableColumn>
          <ElTableColumn prop="type" label="类型" width="90" />
          <ElTableColumn label="操作" width="90">
            <template #default="{ row }">
              <ElButton v-if="row.type === 'dir'" text @click="openDirectory(row.path)"
                >进入</ElButton
              >
              <ElButton v-else text type="success" @click="selectFile(row)">加入</ElButton>
            </template>
          </ElTableColumn>
        </ElTable>
        <ElEmpty v-else-if="filesLoaded" :image-size="42" description="当前目录没有文件" />
      </section>
    </ElForm>

    <section v-if="preview" class="preview-panel">
      <div class="preview-heading"
        ><strong>预览结果</strong
        ><ElTag size="small" type="success" effect="plain">{{
          shortCommit(preview.commit_id)
        }}</ElTag></div
      >
      <ElDescriptions :column="2" border>
        <ElDescriptionsItem label="内容摘要">{{
          preview.content_hash.slice(0, 16)
        }}</ElDescriptionsItem>
        <ElDescriptionsItem label="组件数量">{{ preview.component_count }}</ElDescriptionsItem>
        <ElDescriptionsItem v-if="preview.resource_count !== undefined" label="资源数量">{{
          preview.resource_count
        }}</ElDescriptionsItem>
        <ElDescriptionsItem v-if="preview.metadata" label="Chart"
          >{{ preview.metadata.name }} {{ preview.metadata.version }}</ElDescriptionsItem
        >
        <ElDescriptionsItem v-if="preview.os" label="操作系统">{{ preview.os }}</ElDescriptionsItem>
        <ElDescriptionsItem v-if="preview.artifact_source" label="制品来源">{{
          preview.artifact_source
        }}</ElDescriptionsItem>
      </ElDescriptions>
      <div v-if="preview.components?.length" class="tag-row"
        ><span>组件</span
        ><ElTag v-for="item in preview.components" :key="item" size="small" effect="plain">{{
          item
        }}</ElTag></div
      >
      <div v-if="preview.images?.length" class="tag-row"
        ><span>镜像</span
        ><ElTag
          v-for="item in preview.images"
          :key="item"
          size="small"
          type="info"
          effect="plain"
          >{{ item }}</ElTag
        ></div
      >
      <div v-if="preview.scripts?.length" class="script-list"
        ><div v-for="item in preview.scripts" :key="item.phase"
          ><strong>{{ item.phase }}</strong
          ><span>{{ item.path }}</span
          ><small>{{ item.bytes }} bytes</small></div
        ></div
      >
    </section>

    <template #footer>
      <ElButton @click="dialogVisible = false">取消</ElButton>
      <ElButton :loading="previewing" :disabled="!canSubmit" @click="loadPreview">预览</ElButton>
      <ElButton type="primary" :loading="importing" :disabled="!preview" @click="confirmImport"
        >创建 Revision</ElButton
      >
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue'
  import { ElMessage } from 'element-plus'
  import {
    importServiceRevisionApi,
    listRepositoryFilesApi,
    previewServiceImportApi,
    type AxiOpsService,
    type ServiceImportPreview,
    type ServiceImportRequest
  } from '@/api/axiops'

  const props = defineProps<{ visible: boolean; service: AxiOpsService | null }>()
  const emit = defineEmits<{ 'update:visible': [value: boolean]; imported: [] }>()
  const dialogVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })
  const previewing = ref(false)
  const importing = ref(false)
  const preview = ref<ServiceImportPreview | null>(null)
  const form = reactive({ repo_path: '', ref: 'main', paths: '', chart_path: '', spec_path: '' })
  const repoFiles = ref<Array<{ name: string; path: string; type: 'file' | 'dir'; size: number }>>(
    []
  )
  const browsePath = ref('')
  const filesLoading = ref(false)
  const filesLoaded = ref(false)
  const canSubmit = computed(() =>
    Boolean(
      props.service?.code_source_id &&
        form.repo_path.trim() &&
        form.ref.trim() &&
        pathList().length &&
        (props.service.delivery_type !== 'helm' || form.chart_path.trim()) &&
        (props.service.delivery_type !== 'host' || form.spec_path.trim())
    )
  )
  const pathPlaceholder = computed(() =>
    props.service?.delivery_type === 'helm'
      ? 'deploy/chart/Chart.yaml\ndeploy/chart/values.yaml\ndeploy/chart/templates/deployment.yaml'
      : props.service?.delivery_type === 'host'
        ? 'deploy/host-service.yaml\nscripts/deploy.sh\nscripts/health.sh\nscripts/rollback.sh'
        : 'deploy/deployment.yaml\ndeploy/service.yaml'
  )
  const pathList = () =>
    Array.from(
      new Set(
        form.paths
          .split(/\r?\n/)
          .map((item) => item.trim())
          .filter(Boolean)
      )
    )
  const repositoryPathFromURL = (value: string) => {
    try {
      const path = new URL(value).pathname.replace(/^\/+|\/+$/g, '')
      return path.replace(/\.git$/i, '')
    } catch {
      return value.replace(/^\/+|\/+$/g, '').replace(/\.git$/i, '')
    }
  }
  const requestData = (): ServiceImportRequest => ({
    code_source_id: props.service?.code_source_id || 0,
    repo_path: form.repo_path.trim(),
    ref: form.ref.trim(),
    paths: pathList(),
    ...(props.service?.delivery_type === 'helm' ? { chart_path: form.chart_path.trim() } : {}),
    ...(props.service?.delivery_type === 'host' ? { spec_path: form.spec_path.trim() } : {})
  })
  const loadFiles = async () => {
    if (!props.service?.code_source_id || !form.repo_path.trim() || !form.ref.trim()) return
    filesLoading.value = true
    try {
      repoFiles.value =
        (await listRepositoryFilesApi(props.service.code_source_id, {
          repo_path: form.repo_path.trim(),
          ref: form.ref.trim(),
          path: browsePath.value
        })) || []
      filesLoaded.value = true
    } finally {
      filesLoading.value = false
    }
  }
  const openDirectory = async (path: string) => {
    browsePath.value = path
    await loadFiles()
  }
  const selectFile = (file: { path: string; type: 'file' | 'dir' }) => {
    if (file.type === 'dir') return openDirectory(file.path)
    const paths = pathList()
    if (!paths.includes(file.path)) form.paths = [...paths, file.path].join('\n')
  }
  const addCurrentDirectoryFiles = () => {
    const selected = pathList()
    const yamlFiles = repoFiles.value
      .filter((item) => item.type === 'file' && /\.(ya?ml)$/i.test(item.path))
      .map((item) => item.path)
    form.paths = Array.from(new Set([...selected, ...yamlFiles])).join('\n')
  }
  const loadPreview = async () => {
    if (!props.service || !canSubmit.value) return
    previewing.value = true
    try {
      preview.value = await previewServiceImportApi(props.service, requestData())
    } finally {
      previewing.value = false
    }
  }
  const confirmImport = async () => {
    if (!props.service || !preview.value) return
    importing.value = true
    try {
      const revision = await importServiceRevisionApi(props.service, {
        ...requestData(),
        expected_commit: preview.value.commit_id
      })
      ElMessage.success(`Revision #${revision.revision_number} 已创建`)
      dialogVisible.value = false
      emit('imported')
    } finally {
      importing.value = false
    }
  }
  const shortCommit = (value: string) => value?.slice(0, 10) || '-'
  watch(
    () => [props.visible, props.service?.id],
    ([visible]) => {
      if (!visible) return
      preview.value = null
      Object.assign(form, {
        repo_path: '',
        ref: props.service?.repo_branch || 'main',
        paths: '',
        chart_path: '',
        spec_path: ''
      })
      if (props.service?.repo_url) form.repo_path = repositoryPathFromURL(props.service.repo_url)
      browsePath.value = ''
      repoFiles.value = []
      filesLoaded.value = false
    }
  )
</script>

<style scoped lang="scss">
  .import-form {
    margin-top: 14px;
  }
  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 14px;
  }
  .import-form small {
    display: block;
    margin-top: 5px;
    color: var(--art-text-gray-500);
  }
  .preview-panel {
    margin-top: 18px;
    padding-top: 16px;
    border-top: 1px solid var(--art-border-color);
  }
  .repository-browser {
    margin-top: 12px;
    border: 1px solid var(--art-border-color);
    padding: 10px;
  }
  .browser-heading {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 8px;
  }
  .browser-heading small {
    flex: 1;
    color: var(--art-text-gray-500);
    font-family: var(--ax-font-mono);
  }
  .preview-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 10px;
  }
  .tag-row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 7px;
    margin-top: 12px;
  }
  .tag-row > span {
    width: 42px;
    color: var(--art-text-gray-500);
    font-size: 12px;
  }
  .script-list {
    margin-top: 12px;
    border-top: 1px solid var(--art-border-color);
  }
  .script-list > div {
    min-height: 42px;
    display: grid;
    grid-template-columns: 80px 1fr auto;
    align-items: center;
    gap: 10px;
    border-bottom: 1px solid var(--art-border-color);
  }
  .script-list small {
    color: var(--art-text-gray-500);
  }
  @media (max-width: 620px) {
    .form-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
