<template>
  <div class="template-library">
    <header class="library-header"
      ><strong>模板库</strong><span>{{ currentKind.label }}</span
      ><small>创建时使用模板副本，修改模板不会自动更新已有服务或工作流</small></header
    >
    <nav class="kind-tabs" aria-label="模板分类">
      <button
        v-for="item in kinds"
        :key="item.value"
        :class="{ active: kind === item.value }"
        @click="changeKind(item.value)"
        ><component :is="item.icon" :size="16" />{{ item.label }}</button
      >
    </nav>
    <ElAlert v-if="error" :title="error" type="error" :closable="false" show-icon />
    <TemplateHistory
      v-if="historyTemplate"
      v-model="historyVisible"
      :template="historyTemplate"
      :can-edit="editable"
      @restore="restoreVersion"
    />
    <div class="library-body" v-loading="loading">
      <aside class="template-list">
        <div class="list-toolbar"
          ><ElInput v-model="search" placeholder="搜索模板" clearable size="small" /><ElButton
            v-if="canCreate"
            :icon="Plus"
            size="small"
            title="新建模板"
            @click="createDraft"
        /></div>
        <button
          v-for="item in filtered"
          :key="item.id"
          class="template-row"
          :class="{ selected: draft?.id === item.id }"
          @click="select(item)"
          ><FileCode2 :size="15" /><span>{{ item.name }}</span
          ><small>v{{ item.revision }}</small></button
        >
        <ElEmpty v-if="!loading && !filtered.length" description="暂无模板" :image-size="45" />
        <ElButton text size="small" @click="reload">刷新列表</ElButton>
      </aside>
      <section v-if="draft" class="template-editor">
        <div class="editor-heading"
          ><ElInput
            v-model="draft.name"
            :readonly="!editable"
            placeholder="模板名称"
            maxlength="128"
          /><ElTag size="small" effect="plain">{{
            draft.id ? `v${draft.revision}` : '未保存'
          }}</ElTag
          ><span v-if="dirty" class="unsaved">有修改</span></div
        >
        <ElInput
          v-model="draft.description"
          :readonly="!editable"
          class="template-description"
          placeholder="用途和使用说明"
          maxlength="512"
        />
        <div class="file-tabs"
          ><button
            v-for="(file, index) in draft.files"
            :key="index"
            :class="{ active: fileIndex === index }"
            @click="fileIndex = index"
            >{{ file.path }}</button
          ><ElButton
            v-if="editable && kind !== 'build' && kind !== 'workflow'"
            text
            size="small"
            :icon="Plus"
            @click="addFile"
            >文件</ElButton
          ></div
        >
        <div v-if="currentFile" class="file-tools"
          ><ElInput
            v-model="currentFile.path"
            :readonly="!editable"
            size="small"
            placeholder="文件路径"
          /><ElButton text size="small" @click="downloadFile">下载文件</ElButton
          ><ElButton
            v-if="editable && draft.files.length > 1"
            text
            type="danger"
            size="small"
            @click="removeFile"
            >移除</ElButton
          ></div
        >
        <YamlEditorPro
          v-if="currentFile"
          v-model="currentFile.content"
          :filename="currentFile.path"
          height="100%"
          theme="light"
          :readonly="!editable"
          :show-title-bar="false"
          :show-toolbar="false"
          :validate-on-change="false"
          :auto-format="false"
          class="code-editor"
        />
        <footer class="editor-actions"
          ><ElButton v-if="editable" type="primary" :loading="saving" @click="save">保存</ElButton
          ><ElButton v-if="draft.id" :disabled="saving" @click="openHistory"
            >版本与使用情况</ElButton
          ><ElButton :loading="validating" @click="validate">校验</ElButton
          ><ElButton v-if="canCreate" @click="copyDraft">另存为</ElButton
          ><ElButton v-if="draft.id && canDelete" type="danger" plain @click="archive"
            >归档</ElButton
          ></footer
        >
      </section>
      <ElEmpty v-else class="empty-editor" description="选择左侧模板，或新建一个模板" />
      <aside v-if="draft" class="variables-panel">
        <h3>{{ kind === 'k8s_yaml' ? '变量列表' : '使用说明' }}</h3>
        <template v-if="kind === 'k8s_yaml'">
          <h4>系统内置变量</h4>
          <dl
            ><div v-for="item in builtinVariables" :key="item.key"
              ><dt v-text="variableToken(item.key)" /><dd>{{ item.label }}</dd></div
            ></dl
          >
          <h4>自定义参数 <small>创建时填写，已有服务不自动联动</small></h4>
          <div v-for="(variable, index) in draft.variables" :key="index" class="variable-row"
            ><ElInput
              v-model="variable.key"
              :readonly="!editable"
              size="small"
              placeholder="变量名" /><ElInput
              v-model="variable.value"
              :readonly="!editable"
              size="small"
              placeholder="默认值" /><ElButton
              v-if="editable"
              text
              type="danger"
              :icon="Trash2"
              size="small"
              title="删除变量"
              @click="draft.variables.splice(index, 1)" /><ElInput
              v-model="variable.description"
              :readonly="!editable"
              size="small"
              class="variable-description"
              placeholder="说明，如应用实际监听端口" /><ElInput
              v-model="variable.label"
              :readonly="!editable"
              size="small"
              placeholder="表单名称（如 Nacos 地址）" />
            <ElSelect
              v-model="variable.group"
              :disabled="!editable"
              size="small"
              placeholder="表单分组"
              ><ElOption value="" label="常用参数" /><ElOption
                value="env"
                label="Env / JVM（折叠）" /><ElOption
                value="advanced"
                label="其他参数（折叠）" /></ElSelect
          ></div>
          <ElButton
            v-if="editable"
            text
            type="primary"
            :icon="Plus"
            @click="draft.variables.push({ key: '', value: '', description: '' })"
            >添加变量</ElButton
          >
          <p
            >镜像由构建产物注入。端口必须与应用一致；健康探针、资源配置按应用需要添加，不替换镜像自身的启动命令。</p
          >
        </template>
        <p v-else-if="kind === 'helm'"
          >维护完整 Chart 文件。参数在 values.yaml 中声明，templates/ 中使用 Helm 语法。创建 Helm
          服务时选择这里的 Chart，保存为固定内容的服务版本。</p
        >
        <p v-else-if="kind === 'dockerfile'"
          >维护可复用的 Dockerfile。在服务构建配置中选择“平台模板 /
          编辑”可直接使用；也可以下载后提交到代码仓库并选择文件路径。模板保存不会修改仓库，已保存的构建配置不会随模板库更新自动变化。</p
        >
        <p v-else-if="kind === 'workflow'">
          维护 workflow.json 流程定义，支持构建、测试、通知及 YAML / Helm / 主机部署节点。
          service_id 和 service_revision_id 留为
          0，创建工作流时选择具体服务与环境。审批功能后续完善。金丝雀、数据库执行和 Android
          专用打包尚未提供执行器。
        </p>
        <p v-else
          >构建模板包含 build_image、build_command 和可选 cache_dir。在服务构建配置的 JAR
          镜像或编译产物模式中选择并应用；仓库、凭据和目标环境不放入模板。</p
        >
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref } from 'vue'
  import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { Boxes, FileCode2, Package, Hammer, Plus, Trash2, Workflow } from 'lucide-vue-next'
  import YamlEditorPro from '@/components/yaml-editor-pro/index.vue'
  import TemplateHistory from './TemplateHistory.vue'
  import { useAxiOpsAccess } from '@/composables/useAxiOpsAccess'
  import {
    archiveDeliveryTemplateApi,
    listDeliveryTemplatesApi,
    previewDeliveryTemplateApi,
    saveDeliveryTemplateApi,
    type DeliveryTemplate,
    type TemplateKind
  } from '@/api/axiops'

  const access = useAxiOpsAccess(),
    router = useRouter(),
    route = useRoute()
  const kinds = [
    { value: 'k8s_yaml', label: 'K8s YAML', icon: Boxes },
    { value: 'helm', label: 'Helm Chart', icon: Package },
    { value: 'dockerfile', label: 'Dockerfile', icon: FileCode2 },
    { value: 'build', label: '构建', icon: Hammer },
    { value: 'workflow', label: '工作流', icon: Workflow }
  ] as const
  const kind = ref<TemplateKind>(
    kinds.some((item) => item.value === route.query.kind)
      ? (route.query.kind as TemplateKind)
      : 'k8s_yaml'
  )
  const currentKind = computed(() => kinds.find((item) => item.value === kind.value)!)
  const items = ref<DeliveryTemplate[]>([]),
    draft = ref<DeliveryTemplate | null>(null),
    baseline = ref(''),
    search = ref(''),
    fileIndex = ref(0)
  const loading = ref(false),
    saving = ref(false),
    validating = ref(false),
    error = ref('')
  const historyVisible = ref(false),
    historyTemplate = ref<DeliveryTemplate>()
  async function openHistory() {
    if (!draft.value?.id || !(await discard())) return
    historyTemplate.value = items.value.find((t) => t.id === draft.value?.id)
    historyVisible.value = true
  }
  function restoreVersion(item: DeliveryTemplate) {
    const current = items.value.find((t) => t.id === item.id)
    if (!current) return
    setDraft(current)
    draft.value = clone(item)
  }
  let loadVersion = 0
  const canCreate = computed(() => access.can('build_template', 'create', { type: 'platform' }))
  const canDelete = computed(() => access.can('build_template', 'delete', { type: 'platform' }))
  const editable = computed(
    () =>
      !saving.value &&
      (draft.value?.id
        ? access.can('build_template', 'update', { type: 'platform' })
        : canCreate.value)
  )
  const dirty = computed(() => !!draft.value && JSON.stringify(draft.value) !== baseline.value)
  const filtered = computed(() =>
    items.value.filter((item) => item.name.toLowerCase().includes(search.value.toLowerCase()))
  )
  const currentFile = computed(() => draft.value?.files[fileIndex.value])
  const variableToken = (key: string) => '{{.' + key + '}}'
  const builtinVariables = [
    { key: 'Service', label: '服务编码（合法 Kubernetes 名称）' },
    { key: 'Image', label: '本次构建镜像' },
    { key: 'Product', label: '项目编码' },
    { key: 'Namespace', label: '目标命名空间' },
    { key: 'EnvName', label: '目标环境' }
  ]
  const clone = (item: DeliveryTemplate) => JSON.parse(JSON.stringify(item)) as DeliveryTemplate
  function setDraft(item: DeliveryTemplate | null) {
    draft.value = item ? clone(item) : null
    baseline.value = JSON.stringify(draft.value)
    fileIndex.value = 0
  }
  async function discard() {
    if (saving.value) return false
    if (!dirty.value) return true
    return ElMessageBox.confirm('当前模板有未保存修改，确定放弃？', '未保存的修改', {
      type: 'warning',
      confirmButtonText: '放弃修改',
      cancelButtonText: '继续编辑'
    })
      .then(() => true)
      .catch(() => false)
  }
  async function load() {
    const version = ++loadVersion
    loading.value = true
    error.value = ''
    try {
      const result = await listDeliveryTemplatesApi(kind.value)
      if (version !== loadVersion) return
      items.value = result || []
      setDraft(items.value.find((item) => item.id === draft.value?.id) || items.value[0] || null)
    } catch (e: any) {
      if (version === loadVersion) {
        items.value = []
        setDraft(null)
        error.value = e.message || '模板列表加载失败'
      }
    } finally {
      if (version === loadVersion) loading.value = false
    }
  }
  async function reload() {
    if (await discard()) await load()
  }
  async function changeKind(value: TemplateKind) {
    if (value === kind.value || !(await discard())) return
    kind.value = value
    search.value = ''
    setDraft(null)
    await router.replace({ query: { kind: value } })
    await load()
  }
  async function select(item: DeliveryTemplate) {
    if (item.id !== draft.value?.id && (await discard())) setDraft(item)
  }
  async function createDraft() {
    if (!(await discard())) return
    const initial = items.value.find((item) => item.builtin_key) || items.value[0]
    const file =
      kind.value === 'workflow'
        ? {
            path: 'workflow.json',
            content: JSON.stringify(
              {
                delivery_type: 'k8s_yaml',
                nodes: [
                  {
                    key: 'build',
                    name: '构建',
                    type: 'build',
                    config: { service_id: 0 },
                    timeout_seconds: 3600,
                    retry_limit: 0
                  }
                ]
              },
              null,
              2
            )
          }
        : kind.value === 'dockerfile'
          ? { path: 'Dockerfile', content: 'FROM eclipse-temurin:17-jre\n' }
          : kind.value === 'build'
            ? {
                path: 'build.yaml',
                content:
                  'build_image: maven:3.9-eclipse-temurin-17\nbuild_command: mvn -B clean package\n'
              }
            : { path: kind.value === 'helm' ? 'Chart.yaml' : 'deployment.yaml', content: '' }
    setDraft(
      initial
        ? { ...clone(initial), id: 0, name: '', revision: 0, builtin_key: '' }
        : {
            id: 0,
            kind: kind.value,
            name: '',
            description: '',
            revision: 0,
            files: [file],
            variables: []
          }
    )
  }
  function copyDraft() {
    if (!draft.value || saving.value) return
    draft.value = {
      ...clone(draft.value),
      id: 0,
      revision: 0,
      builtin_key: '',
      name: `${draft.value.name} 副本`
    }
    baseline.value = ''
  }
  function addFile() {
    if (!draft.value) return
    draft.value.files.push({ path: `file-${draft.value.files.length + 1}.yaml`, content: '' })
    fileIndex.value = draft.value.files.length - 1
  }
  function removeFile() {
    draft.value?.files.splice(fileIndex.value, 1)
    fileIndex.value = 0
  }
  function downloadFile() {
    if (!currentFile.value) return
    const url = URL.createObjectURL(
      new Blob([currentFile.value.content], { type: 'text/plain;charset=utf-8' })
    )
    const a = document.createElement('a')
    a.href = url
    a.download = currentFile.value.path.split('/').pop()!
    a.click()
    URL.revokeObjectURL(url)
  }
  async function validate() {
    if (!draft.value) return
    validating.value = true
    try {
      await previewDeliveryTemplateApi(draft.value)
      ElMessage.success('模板校验通过')
    } catch (e: any) {
      ElMessage.error(e.message || '校验失败')
    } finally {
      validating.value = false
    }
  }
  async function save() {
    if (!draft.value || saving.value || !editable.value) return
    saving.value = true
    try {
      const saved = await saveDeliveryTemplateApi(draft.value)
      items.value = [...items.value.filter((item) => item.id !== saved.id), saved]
      setDraft(saved)
      ElMessage.success('模板已保存，已有服务保持不变')
    } catch (e: any) {
      ElMessage.error(e.message || '保存失败，本地修改已保留')
    } finally {
      saving.value = false
    }
  }
  async function archive() {
    if (!draft.value?.id || !(await discard())) return
    try {
      await ElMessageBox.confirm('归档后不再供新服务选择；已有服务配置不变。', '归档模板', {
        type: 'warning'
      })
      await archiveDeliveryTemplateApi(draft.value)
      setDraft(null)
      await load()
    } catch (e: any) {
      if (e instanceof Error) ElMessage.error(e.message)
    }
  }
  onBeforeRouteLeave(() => discard())
  onMounted(async () => {
    await access.load()
    await load()
  })
</script>

<style scoped lang="scss">
  .template-library {
    background: var(--art-main-bg-color, #fff);
    color: var(--art-text-gray-900, #334155);
    height: calc(100vh - 110px);
    min-height: 600px;
    display: flex;
    flex-direction: column;
  }
  .library-header {
    display: flex;
    gap: 16px;
    align-items: center;
    padding: 12px 18px;
    border-bottom: 1px solid #e8edf3;
    font-size: 13px;
  }
  .library-header small {
    margin-left: auto;
    color: #8793a5;
  }
  .kind-tabs {
    display: flex;
    gap: 12px;
    padding: 0 14px;
    border-bottom: 1px solid #e8edf3;
  }
  .kind-tabs button,
  .file-tabs button {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 10px;
    background: transparent;
    border: 0;
    border-bottom: 2px solid transparent;
    color: #64748b;
    cursor: pointer;
  }
  .kind-tabs button.active,
  .file-tabs button.active {
    color: #5265ff;
    border-bottom-color: #5265ff;
  }
  .library-body {
    flex: 1;
    display: grid;
    grid-template-columns: 220px minmax(340px, 1fr) minmax(280px, 350px);
    min-height: 0;
  }
  .template-list {
    padding: 12px 8px;
    border-right: 1px solid #e8edf3;
    overflow: auto;
  }
  .list-toolbar {
    display: flex;
    gap: 6px;
    margin-bottom: 12px;
  }
  .template-row {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 10px 6px;
    border: 0;
    text-align: left;
    background: transparent;
    color: inherit;
    cursor: pointer;
  }
  .template-row span {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .template-row small {
    color: #94a3b8;
  }
  .template-row.selected {
    background: #f0f3ff;
    color: #5265ff;
  }
  .template-editor {
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 12px;
  }
  .editor-heading {
    display: flex;
    gap: 10px;
    align-items: center;
  }
  .editor-heading .el-input {
    max-width: 320px;
  }
  .unsaved {
    font-size: 12px;
    color: #b7791f;
  }
  .file-tabs {
    display: flex;
    overflow-x: auto;
    flex-shrink: 0;
  }
  .file-tabs button {
    padding: 8px;
    white-space: nowrap;
    font-size: 12px;
  }
  .file-tools {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .file-tools .el-input {
    flex: 1;
  }
  .code-editor {
    flex: 1;
    min-height: 250px;
  }
  .editor-actions {
    display: flex;
    border-top: 1px solid #e8edf3;
    padding-top: 10px;
  }
  .variables-panel {
    border-left: 1px solid #e8edf3;
    padding: 16px;
    overflow-y: auto;
  }
  .variables-panel h3 {
    font-size: 14px;
    margin: 0 0 24px;
  }
  .variables-panel h4 {
    font-size: 12px;
    color: #7c8798;
    font-weight: normal;
    margin: 20px 0 8px;
  }
  .variables-panel h4 small {
    display: block;
    margin-top: 5px;
  }
  .variables-panel dl {
    margin: 0;
    font-size: 12px;
  }
  .variables-panel dl div {
    display: flex;
    gap: 8px;
    padding: 10px 0;
    border-bottom: 1px solid #eef1f6;
  }
  .variables-panel dt {
    min-width: 105px;
  }
  .variables-panel dd {
    margin: 0;
    color: #768296;
  }
  .variables-panel p {
    font-size: 12px;
    line-height: 1.9;
    color: #8793a5;
  }
  .variable-row {
    display: grid;
    grid-template-columns: 1fr 1fr 24px;
    gap: 6px;
    margin-bottom: 12px;
  }
  .variable-description {
    grid-column: 1 / -1;
  }
  .empty-editor {
    grid-column: 2 / -1;
  }
  @media (max-width: 1100px) {
    .library-body {
      grid-template-columns: 180px minmax(320px, 1fr);
    }
    .variables-panel {
      grid-column: 2;
      max-height: 300px;
      border-top: 1px solid #e8edf3;
    }
    .library-header small {
      display: none;
    }
  }
</style>
