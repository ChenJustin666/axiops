<template>
  <ElDrawer
    v-model="drawerVisible"
    :title="''"
    direction="rtl"
    size="100%"
    :with-header="false"
    destroy-on-close
    class="service-wizard-drawer"
  >
    <div class="wizard-shell">
      <header class="onboarding-header">
        <div class="onboarding-breadcrumb">
          <span class="axiops-brand" aria-label="AxiOps">
            <ArtLogo :size="27" class="axiops-brand-logo" />
            <strong>AxiOps</strong>
          </span>
          <span class="breadcrumb-divider" />
          <button type="button" class="back-button" @click="drawerVisible = false">‹</button>
          <span>项目</span>
          <span class="breadcrumb-separator">›</span>
          <strong>{{ editing ? `编辑服务 · ${service?.name || ''}` : '新建服务' }}</strong>
          <ElTag size="small" effect="plain">{{ deliveryLabel }}</ElTag>
        </div>
        <div class="onboarding-envs" aria-label="已配置环境">
          <ElTag
            v-for="environment in environments.slice(0, 4)"
            :key="environment.id"
            size="small"
            type="success"
            effect="plain"
          >
            {{ environment.name }}
          </ElTag>
          <span v-if="!environments.length" class="env-placeholder">环境将在下一步配置</span>
        </div>
      </header>

      <div class="current-step-title">
        <strong>服务配置</strong>
        <span>在服务目录中维护 YAML/Helm 文件、变量和 Revision</span>
      </div>

      <section v-if="step === 0" class="wizard-section">
        <div class="empty-editor-layout">
          <aside class="empty-service-tree">
            <div class="tree-panel-heading"><span>服务列表</span></div>
            <div class="tree-panel-footer">
              <ElInput size="small" placeholder="搜索服务" :prefix-icon="Search" disabled />
            </div>
          </aside>
          <div class="empty-editor-content">
            <FileCode2 :size="58" stroke-width="1" />
            <p>暂无服务，点击创建服务</p>
            <ElButton
              circle
              :icon="Plus"
              aria-label="创建服务目录"
              @click="createManualServiceDirectory"
            />
          </div>
        </div>
      </section>

      <section v-else-if="step === 1" class="wizard-section">
        <div class="section-heading">
          <div>
            <strong>服务身份与定义</strong>
            <span>服务编码创建后固定，用于镜像仓库、环境和权限引用。</span>
          </div>
          <ElTag size="small" effect="plain">{{ deliveryLabel }}</ElTag>
        </div>
        <div class="service-config-layout">
          <aside class="service-tree-panel">
            <div class="tree-panel-heading">
              <div
                ><span>服务配置</span
                ><small>{{ editing ? '编辑当前服务' : '可连续添加多个服务' }}</small></div
              >
              <ElButton
                v-if="deliveryType !== 'helm'"
                text
                circle
                :icon="Plus"
                title="添加服务目录"
                @click="createManualServiceDirectory"
              />
              <ElButton
                text
                size="small"
                :icon="PackageOpen"
                title="模板导入"
                @click="openSourceDialog('template')"
                >模板</ElButton
              >
              <ElButton
                text
                size="small"
                :icon="FileCode2"
                title="Git 导入"
                @click="openSourceDialog('git')"
                >Git</ElButton
              >
            </div>
            <div class="tree-panel-list">
              <button type="button" class="tree-service-item is-active">
                <span class="tree-service-status" />
                <span
                  ><strong>{{ form.name || '未命名服务' }}</strong
                  ><small>{{ form.code || '等待填写编码' }}</small></span
                >
              </button>
              <div class="tree-file-list">
                <button
                  v-for="file in serviceFiles"
                  :key="`tree-${file}`"
                  type="button"
                  :class="['tree-file-item', { 'is-active': activeFileName === file }]"
                  @click="activeFileName = file"
                >
                  <FileCode2 :size="13" />
                  <span>{{ file }}</span>
                </button>
              </div>
            </div>
            <div class="tree-panel-footer">
              <ElInput
                v-model="treeSearch"
                size="small"
                placeholder="搜索服务"
                :prefix-icon="Search"
              />
            </div>
          </aside>

          <div class="service-definition-column">
            <ElForm
              ref="formRef"
              :model="form"
              :rules="rules"
              label-position="top"
              class="wizard-form"
            >
              <div class="definition-files-heading">
                <div
                  ><strong>服务目录 / {{ form.code || 'new-service' }}</strong
                  ><small>在目录中维护 YAML 文件</small></div
                >
                <div class="definition-files-actions">
                  <ElButton
                    v-if="!addingFile"
                    text
                    size="small"
                    :icon="Plus"
                    @click="addingFile = true"
                    >添加文件</ElButton
                  >
                  <template v-else>
                    <ElInput
                      v-model="newFileName"
                      size="small"
                      placeholder="例如：config.yaml"
                      @keyup.enter="confirmAddFile"
                    />
                    <ElButton text size="small" type="primary" @click="confirmAddFile"
                      >确定</ElButton
                    >
                    <ElButton text size="small" @click="addingFile = false">取消</ElButton>
                  </template>
                </div>
              </div>
              <div class="definition-files-list">
                <button
                  v-for="file in serviceFiles"
                  :key="file"
                  type="button"
                  :class="['definition-file-item', { 'is-active': activeFileName === file }]"
                  @click="activeFileName = file"
                >
                  <FileCode2 :size="14" /><span>{{ file }}</span>
                  <Trash2
                    v-if="serviceFiles.length > 1"
                    :size="12"
                    class="file-remove"
                    @click.stop="removeFile(file)"
                  />
                </button>
              </div>
              <div class="form-grid">
                <ElFormItem label="服务名称" prop="name">
                  <ElInput v-model="form.name" maxlength="128" placeholder="例如：risk_server" />
                  <ElText size="small" type="info">{{ SERVICE_NAME_HINT }}</ElText>
                </ElFormItem>
                <ElFormItem label="服务编码" prop="code">
                  <ElInput
                    v-model="form.code"
                    :disabled="editing"
                    maxlength="64"
                    placeholder="order-api"
                  />
                </ElFormItem>
              </div>

              <template v-if="form.source_type === 'git'">
                <div class="form-grid">
                  <ElFormItem label="平台代码源" prop="code_source_id">
                    <ElSelect
                      v-model="form.code_source_id"
                      filterable
                      clearable
                      style="width: 100%"
                      placeholder="选择统一代码源"
                      @change="loadRepositories"
                    >
                      <ElOption
                        v-for="source in codeSources"
                        :key="source.id"
                        :label="`${source.name} · ${source.type}`"
                        :value="source.id"
                      />
                    </ElSelect>
                  </ElFormItem>
                  <ElFormItem label="仓库" prop="repository_id">
                    <ElSelect
                      v-model="form.repository_id"
                      filterable
                      clearable
                      :loading="repositoryLoading"
                      style="width: 100%"
                      placeholder="选择仓库"
                      @change="selectRepository"
                    >
                      <ElOption
                        v-for="repository in repositories"
                        :key="repository.id"
                        :label="repository.full_name || repository.name"
                        :value="repository.id"
                      />
                    </ElSelect>
                  </ElFormItem>
                </div>
                <div class="form-grid">
                  <ElFormItem label="仓库路径" prop="repo_path">
                    <ElInput v-model="form.repo_path" placeholder="team/order-service" />
                  </ElFormItem>
                  <ElFormItem label="分支、Tag 或 Commit" prop="repo_branch">
                    <ElInput v-model="form.repo_branch" placeholder="main" />
                  </ElFormItem>
                </div>
                <ElFormItem v-if="deliveryType === 'helm'" label="Chart 目录" prop="chart_path">
                  <ElInput v-model="form.chart_path" placeholder="deploy/chart；仓库根目录填写 ." />
                </ElFormItem>
                <p v-if="deliveryType === 'helm'" class="field-hint"
                  >选择包含 Chart.yaml 的目录，自动读取 values、templates、辅助文件和已展开的子
                  Chart，无需逐个选择文件。</p
                >
                <ElFormItem v-else label="服务定义文件" prop="paths">
                  <ElInput
                    v-model="form.paths"
                    type="textarea"
                    :rows="4"
                    placeholder="deploy/order/deployment.yaml&#10;deploy/order/service.yaml"
                  />
                  <small class="field-hint"
                    >每行一个仓库相对路径。可以在下方浏览目录并点选文件。</small
                  >
                </ElFormItem>
                <div class="repository-browser">
                  <div class="browser-toolbar">
                    <div>
                      <strong>仓库目录</strong>
                      <small>{{ browsePath || '/' }}</small>
                    </div>
                    <ElButton
                      size="small"
                      :icon="FolderSearch"
                      :loading="filesLoading"
                      @click="loadFiles"
                    >
                      浏览目录
                    </ElButton>
                    <ElButton
                      v-if="
                        deliveryType === 'helm' &&
                        repoFiles.some((file) => file.name === 'Chart.yaml')
                      "
                      size="small"
                      type="primary"
                      @click="form.chart_path = browsePath || '.'"
                      >使用此 Chart 目录</ElButton
                    >
                  </div>
                  <ElTable v-if="repoFiles.length" :data="repoFiles" size="small" max-height="190">
                    <ElTableColumn label="名称" min-width="260">
                      <template #default="{ row }">
                        <ElButton text type="primary" @click="selectFile(row)">
                          <Folder v-if="row.type === 'dir'" :size="14" />
                          <FileCode2 v-else :size="14" />
                          {{ row.name }}
                        </ElButton>
                      </template>
                    </ElTableColumn>
                    <ElTableColumn prop="type" label="类型" width="85" />
                    <ElTableColumn label="操作" width="100">
                      <template #default="{ row }">
                        <ElButton v-if="row.type === 'dir'" text @click="openDirectory(row.path)"
                          >进入</ElButton
                        >
                        <ElButton
                          v-else-if="deliveryType !== 'helm'"
                          text
                          type="success"
                          @click="selectFile(row)"
                          >加入</ElButton
                        >
                      </template>
                    </ElTableColumn>
                  </ElTable>
                  <ElEmpty
                    v-else-if="filesLoaded"
                    :image-size="42"
                    description="当前目录没有文件"
                  />
                </div>
              </template>

              <template v-else>
                <div v-if="deliveryType !== 'host'" class="template-strip">
                  <DeliveryTemplateSelect
                    :key="templatePickerKey"
                    :kind="deliveryType === 'helm' ? 'helm' : 'k8s_yaml'"
                    :auto-select="!editing"
                    :disabled="definitionLoading || submitting"
                    :apply="useLibraryTemplate"
                  />
                </div>
                <ElFormItem label="服务定义" prop="definition">
                  <div class="definition-editor">
                    <YamlEditorPro
                      v-model="activeFileContent"
                      :filename="definitionFilename"
                      height="300px"
                      theme="dark"
                      :show-toolbar="true"
                      :show-status-bar="true"
                      :auto-format="false"
                      :validate-on-change="false"
                      :placeholder="definitionPlaceholder"
                    />
                  </div>
                </ElFormItem>
              </template>
            </ElForm>
          </div>

          <aside class="variable-panel">
            <div class="variable-panel-heading"
              ><span>变量</span><ElButton text size="small">配置全局变量</ElButton></div
            >
            <section class="variable-section">
              <div class="variable-section-title"
                ><strong>系统内置变量</strong><small>由项目和环境自动注入</small></div
              >
              <dl class="variable-list">
                <div><dt>$Product$</dt><dd>当前项目</dd></div>
                <div
                  ><dt>$Service$</dt><dd>{{ form.name || '未命名服务' }}</dd></div
                >
                <div><dt>$Namespace$</dt><dd>下一步配置</dd></div>
                <div><dt>$EnvName$</dt><dd>下一步配置</dd></div>
                <div
                  ><dt>$Image$</dt><dd>{{ form.image_name || '构建产物注入' }}</dd></div
                >
              </dl>
            </section>
            <section class="variable-section">
              <div class="variable-section-title"
                ><strong>服务变量</strong><small>模板参数可在环境中覆盖</small></div
              >
              <div class="service-variable-list">
                <label v-for="key in Object.keys(serviceVariables)" :key="key"
                  ><span>{{ key }}</span
                  ><ElInput v-model="serviceVariables[key]" size="small"
                /></label>
                <small v-if="deliveryType === 'helm'">Helm 参数在 values.yaml 中维护</small>
              </div>
            </section>
          </aside>
        </div>
      </section>
    </div>

    <ElDialog
      v-model="sourceDialogVisible"
      title="新建服务"
      width="680px"
      append-to-body
      destroy-on-close
      :close-on-click-modal="false"
    >
      <p class="source-dialog-tip">请选择服务配置来源</p>
      <div class="mode-grid source-mode-grid" :class="{ 'is-helm': deliveryType === 'helm' }">
        <button
          v-for="option in modeOptions"
          :key="option.value"
          type="button"
          :class="['mode-card', { 'is-active': form.source_type === option.value }]"
          @click="form.source_type = option.value"
        >
          <span class="mode-icon" :class="`is-${option.value}`">
            <component :is="option.icon" :size="20" />
          </span>
          <span class="mode-copy">
            <strong>{{ option.label }}</strong>
            <small>{{ option.description }}</small>
          </span>
          <CheckCircle2 v-if="form.source_type === option.value" :size="17" class="mode-check" />
        </button>
      </div>
      <template #footer>
        <ElButton @click="sourceDialogVisible = false">取消</ElButton>
        <ElButton type="primary" @click="confirmSource">下一步</ElButton>
      </template>
    </ElDialog>

    <template #footer>
      <div class="wizard-footer">
        <span class="step-note">服务定义未保存</span>
        <div>
          <ElButton @click="drawerVisible = false">取消</ElButton>
          <ElButton type="primary" :icon="Check" :loading="submitting" @click="submit"
            >保存服务</ElButton
          >
        </div>
      </div>
    </template>
  </ElDrawer>
</template>

<script setup lang="ts">
  import { SERVICE_NAME_HINT, serviceNameError } from '@/utils/service-name'
  import { computed, reactive, ref, watch } from 'vue'
  import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
  import {
    Check,
    CheckCircle2,
    Container,
    FileCode2,
    Folder,
    FolderSearch,
    PackageOpen,
    Plus,
    Search,
    Trash2
  } from 'lucide-vue-next'
  import {
    createServiceApi,
    createServiceRevisionApi,
    createWorkflowApi,
    importServiceRevisionApi,
    listServiceRevisionsApi,
    listRepositoriesApi,
    listRepositoryFilesApi,
    previewServiceImportApi,
    saveBuildConfigApi,
    updateServiceApi,
    type AxiOpsCodeSource,
    type AxiOpsEnvironment,
    type AxiOpsService,
    type CreateServiceInput,
    type DeliveryType,
    type ServiceImportPreview
  } from '@/api/axiops'
  import YamlEditorPro from '@/components/yaml-editor-pro/index.vue'
  import DeliveryTemplateSelect from '@/components/delivery-template-select/index.vue'
  import { previewDeliveryTemplateApi, type DeliveryTemplate } from '@/api/axiops'

  type SourceType = CreateServiceInput['source_type']
  type Repository = {
    id: string
    name: string
    full_name: string
    clone_url: string
    default_branch: string
    private: boolean
  }
  type RepoFile = { name: string; path: string; type: 'file' | 'dir'; size: number }

  const props = defineProps<{
    visible: boolean
    projectId: number
    deliveryType: DeliveryType | 'unclassified'
    service: AxiOpsService | null
    codeSources: AxiOpsCodeSource[]
    environments: AxiOpsEnvironment[]
    canCreateWorkflow: boolean
    canUpdateBuild: boolean
    initialSource?: SourceType
    createWorkflow?: boolean
  }>()
  const emit = defineEmits<{ 'update:visible': [value: boolean]; saved: [] }>()
  const drawerVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })
  const editing = computed(() => Boolean(props.service?.id))
  const step = ref(0)
  const sourceDialogVisible = ref(false)
  const defaultServiceVariables: Record<string, string> =
    props.deliveryType === 'k8s_yaml' ? { replicas: '1', port: '8080' } : {}
  const serviceVariables = reactive<Record<string, string>>({})
  const templatePickerKey = ref(0)
  const selectedLibraryTemplate = ref<DeliveryTemplate | null>(null)
  const definitionLoadError = ref(false)
  const definitionLoading = ref(false)
  let definitionVersion = 0
  const serviceFiles = ref<string[]>(['deployment.yaml', 'service.yaml'])
  const fileContents = reactive<Record<string, string>>({})
  const activeFileName = ref('deployment.yaml')
  const addingFile = ref(false)
  const newFileName = ref('')
  const treeSearch = ref('')
  const submitting = ref(false)
  const formRef = ref<FormInstance>()
  const repositoryLoading = ref(false)
  const filesLoading = ref(false)
  const filesLoaded = ref(false)
  const browsePath = ref('')
  const repositories = ref<Repository[]>([])
  const repoFiles = ref<RepoFile[]>([])
  const definitionPlaceholder = computed(() =>
    props.deliveryType === 'helm'
      ? 'replicaCount: 1\nimage:\n  repository: registry.example.com/team/service\n  tag: latest'
      : props.deliveryType === 'host'
        ? 'name: service\nartifact: dist/service.tar.gz\nhealth_check:\n  command: systemctl is-active service'
        : 'apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: service'
  )
  const defaultDefinition = () =>
    props.deliveryType === 'host'
      ? 'name: service\nartifact: dist/service.tar.gz\nhealth_check:\n  command: systemctl is-active service\n'
      : ''
  const defaultFileDefinitions = (): Record<string, string> =>
    props.deliveryType === 'host' ? { 'service.yaml': defaultDefinition() } : {}

  const form = reactive({
    name: '',
    code: '',
    description: '',
    source_type: 'git' as SourceType,
    repo_url: '',
    repo_branch: 'main',
    repo_path: '',
    chart_path: '.',
    paths: '',
    code_source_id: undefined as number | undefined,
    repository_id: undefined as string | undefined,
    language: '',
    framework: '',
    image_name: '',
    definition: '',
    init_build: true,
    init_workflow: true,
    include_test: props.deliveryType !== 'helm'
  })
  const sourceModes = [
    {
      value: 'git' as const,
      label: '从代码库同步',
      description: '选择平台代码源、仓库和定义目录，固定 Commit 生成 Revision。',
      icon: FileCode2
    },
    {
      value: 'manual' as const,
      label: '手工输入',
      description: '直接编辑 K8s YAML 或主机服务定义。',
      icon: Container
    },
    {
      value: 'template' as const,
      label: '使用模板',
      description: '选择模板库中的模板，在服务中保存独立配置副本。',
      icon: PackageOpen
    }
  ]
  const modeOptions = computed(() =>
    props.deliveryType === 'helm' ? [sourceModes[2], sourceModes[0]] : sourceModes
  )
  const deliveryLabel = computed(
    () =>
      ({ k8s_yaml: 'K8s YAML', helm: 'Helm Chart', host: '主机应用', unclassified: '未分类' })[
        props.deliveryType
      ] || '未分类'
  )
  const definitionFilename = computed(() => `${form.code || 'service'}/${activeFileName.value}`)
  const activeFileContent = computed({
    get: () =>
      Object.prototype.hasOwnProperty.call(fileContents, activeFileName.value)
        ? fileContents[activeFileName.value]
        : form.definition,
    set: (value: string) => {
      fileContents[activeFileName.value] = value
      form.definition = value
    }
  })
  const rules: FormRules = {
    name: [
      {
        validator: (_rule, value, callback) => {
          const name = String(value || '').trim()
          const error = props.service && name === props.service.name ? '' : serviceNameError(name)
          if (error) callback(new Error(error))
          else callback()
        },
        trigger: 'blur'
      }
    ],
    code: [
      {
        required: true,
        pattern: /^[a-z][a-z0-9-]{0,63}$/,
        message: '使用小写字母、数字和连字符',
        trigger: 'blur'
      }
    ],
    code_source_id: [
      {
        validator: (_rule, value, callback) =>
          form.source_type !== 'git' || value ? callback() : callback(new Error('请选择代码源')),
        trigger: 'change'
      }
    ],
    repository_id: [
      {
        validator: (_rule, value, callback) =>
          form.source_type !== 'git' || value || form.repo_path.trim()
            ? callback()
            : callback(new Error('请选择仓库')),
        trigger: 'change'
      }
    ],
    repo_path: [
      {
        validator: (_rule, value, callback) =>
          form.source_type !== 'git' || String(value || '').trim()
            ? callback()
            : callback(new Error('请输入仓库路径')),
        trigger: 'blur'
      }
    ],
    paths: [
      {
        validator: (_rule, value, callback) =>
          form.source_type !== 'git' ||
          props.deliveryType === 'helm' ||
          String(value || '')
            .split(/\r?\n/)
            .some((item) => item.trim())
            ? callback()
            : callback(new Error('至少选择一个 YAML 或定义文件')),
        trigger: 'blur'
      }
    ],
    chart_path: [
      {
        validator: (_rule, value, callback) =>
          props.deliveryType !== 'helm' || form.source_type !== 'git' || String(value || '').trim()
            ? callback()
            : callback(new Error('请填写 Chart 所在目录，仓库根目录填写 .')),
        trigger: 'blur'
      }
    ],
    definition: [
      {
        validator: (_rule, value, callback) =>
          form.source_type === 'git' || String(value || '').trim()
            ? callback()
            : callback(new Error('请输入服务定义')),
        trigger: 'blur'
      }
    ]
  }
  const reset = () => {
    definitionVersion++
    definitionLoading.value = false
    // This drawer is a service editor only. Project/environment/workflow setup
    // lives in the corresponding project tabs.
    step.value = 1
    Object.keys(serviceVariables).forEach((key) => delete serviceVariables[key])
    Object.assign(serviceVariables, defaultServiceVariables)
    selectedLibraryTemplate.value = null
    definitionLoadError.value = false
    templatePickerKey.value++
    Object.assign(form, {
      name: props.service?.name || '',
      code: props.service?.code || '',
      description: props.service?.description || '',
      source_type: editing.value
        ? props.service?.source_type || 'manual'
        : props.deliveryType === 'helm'
          ? props.initialSource === 'git'
            ? 'git'
            : 'template'
          : props.initialSource || 'manual',
      repo_url: props.service?.repo_url || '',
      repo_branch: props.service?.repo_branch || 'main',
      repo_path: props.service?.repo_url ? repositoryPath(props.service.repo_url) : '',
      chart_path: '.',
      paths: '',
      code_source_id:
        props.service?.code_source_id ||
        (props.codeSources.length === 1 ? props.codeSources[0].id : undefined),
      repository_id: undefined,
      language: props.service?.language || '',
      framework: props.service?.framework || '',
      image_name: props.service?.image_name || '',
      definition: '',
      init_build: true,
      init_workflow: props.createWorkflow ?? !editing.value,
      include_test: props.deliveryType !== 'helm'
    })
    repositories.value = []
    repoFiles.value = []
    browsePath.value = ''
    filesLoaded.value = false
    treeSearch.value = ''
    if (form.code_source_id) void loadRepositories(form.code_source_id)
    serviceFiles.value =
      props.deliveryType === 'helm'
        ? []
        : props.deliveryType === 'host'
          ? ['service.yaml']
          : ['deployment.yaml', 'service.yaml']
    activeFileName.value = serviceFiles.value[0] || 'Chart.yaml'
    Object.keys(fileContents).forEach((key) => delete fileContents[key])
    const defaults = defaultFileDefinitions()
    const definition = form.definition || defaultDefinition()
    serviceFiles.value.forEach((file, index) => {
      fileContents[file] = defaults[file] || (index === 0 ? definition : '')
    })
    addingFile.value = false
    newFileName.value = ''
    // Helm creation always starts from a library template or repository Chart.
    sourceDialogVisible.value = !editing.value && !props.initialSource
    if (!editing.value && props.initialSource) {
      step.value = 1
    }
  }
  const createManualServiceDirectory = () => {
    if (props.deliveryType === 'helm') {
      openSourceDialog('template')
      return
    }
    form.source_type = 'manual'
    if (!form.name) form.name = 'new-service'
    if (!form.code) form.code = 'new-service'
    if (!form.definition) form.definition = defaultDefinition()
    step.value = 1
    sourceDialogVisible.value = false
  }
  const openSourceDialog = (source?: SourceType) => {
    if (source)
      form.source_type = props.deliveryType === 'helm' && source === 'manual' ? 'template' : source
    sourceDialogVisible.value = true
  }
  const confirmAddFile = () => {
    const name = newFileName.value.trim()
    if (!name || serviceFiles.value.includes(name)) return
    serviceFiles.value.push(name)
    fileContents[name] = ''
    activeFileName.value = name
    newFileName.value = ''
    addingFile.value = false
  }
  const removeFile = (file: string) => {
    if (serviceFiles.value.length <= 1) return
    serviceFiles.value = serviceFiles.value.filter((item) => item !== file)
    delete fileContents[file]
    if (activeFileName.value === file) activeFileName.value = serviceFiles.value[0]
  }
  const confirmSource = () => {
    sourceDialogVisible.value = false
    if (form.source_type !== 'git' && !form.definition) form.definition = defaultDefinition()
    step.value = 1
  }
  const repositoryPath = (value: string) => {
    try {
      return new URL(value).pathname.replace(/^\/+|\/+$/g, '').replace(/\.git$/i, '')
    } catch {
      return value.replace(/^\/+|\/+$/g, '').replace(/\.git$/i, '')
    }
  }
  const loadRepositories = async (sourceId?: number) => {
    repositories.value = []
    form.repository_id = undefined
    if (!sourceId) return
    repositoryLoading.value = true
    try {
      repositories.value = (await listRepositoriesApi(sourceId)) || []
    } catch (error: any) {
      ElMessage.error(error?.message || '仓库列表加载失败')
    } finally {
      repositoryLoading.value = false
    }
  }
  const selectRepository = (repositoryId?: string) => {
    const repository = repositories.value.find((item) => item.id === repositoryId)
    if (!repository) return
    form.repo_url = repository.clone_url
    form.repo_path = repositoryPath(repository.clone_url)
    form.repo_branch = repository.default_branch || 'main'
  }
  const loadFiles = async () => {
    if (!form.code_source_id || !form.repo_path.trim() || !form.repo_branch.trim()) return
    filesLoading.value = true
    try {
      repoFiles.value =
        (await listRepositoryFilesApi(form.code_source_id, {
          repo_path: form.repo_path.trim(),
          ref: form.repo_branch.trim(),
          path: browsePath.value
        })) || []
      filesLoaded.value = true
    } catch (error: any) {
      ElMessage.error(error?.message || '仓库目录加载失败')
    } finally {
      filesLoading.value = false
    }
  }
  const openDirectory = async (path: string) => {
    browsePath.value = path
    await loadFiles()
  }
  const selectFile = (file: RepoFile) => {
    if (file.type === 'dir') {
      void openDirectory(file.path)
      return
    }
    if (props.deliveryType === 'helm') {
      if (file.name === 'Chart.yaml')
        form.chart_path = file.path.split('/').slice(0, -1).join('/') || '.'
      return
    }
    const paths = form.paths
      .split(/\r?\n/)
      .map((item) => item.trim())
      .filter(Boolean)
    if (!paths.includes(file.path)) form.paths = [...paths, file.path].join('\n')
  }
  const useLibraryTemplate = async (template: DeliveryTemplate) => {
    if (definitionLoading.value || submitting.value) return false
    if (template.kind !== (props.deliveryType === 'helm' ? 'helm' : 'k8s_yaml')) return false
    const version = definitionVersion
    if (serviceFiles.value.some((file) => fileContents[file]?.trim())) {
      const accepted = await ElMessageBox.confirm(
        '选择模板会替换当前编辑的文件和变量，继续？',
        '应用模板',
        { type: 'warning', confirmButtonText: '应用模板', cancelButtonText: '保留当前内容' }
      )
        .then(() => true)
        .catch(() => false)
      if (!accepted || version !== definitionVersion || !props.visible) return false
    }
    selectedLibraryTemplate.value = template
    Object.keys(fileContents).forEach((key) => delete fileContents[key])
    template.files.forEach((file) => {
      fileContents[file.path] = file.content
    })
    serviceFiles.value = template.files.map((file) => file.path)
    activeFileName.value = serviceFiles.value[0]
    form.definition = fileContents[activeFileName.value] || ''
    Object.keys(serviceVariables).forEach((key) => delete serviceVariables[key])
    template.variables.forEach((variable) => {
      serviceVariables[variable.key] = variable.value
    })
    definitionLoadError.value = false
    return true
  }
  const materializeDefinition = (definition: string) => {
    const values: Record<string, string> = {
      cmd: serviceVariables.cmd,
      cpuLimit: serviceVariables.cpuLimit,
      memoryLimit: serviceVariables.memoryLimit,
      port: serviceVariables.port,
      replicas: serviceVariables.replicas,
      Service: form.code.trim(),
      Image: form.image_name.trim() || 'registry.example.com/team/service:latest',
      service: form.code.trim()
    }
    return definition.replace(
      /\{\{\s*\.?([A-Za-z][A-Za-z0-9_]*)\s*\}\}|\$([A-Za-z][A-Za-z0-9_]*)\$/g,
      (match, dotKey, dollarKey) => {
        const key = dotKey || dollarKey
        return Object.prototype.hasOwnProperty.call(values, key) ? values[key] : match
      }
    )
  }
  const addVariableMetadata = (definition: string) =>
    `# AxiOps-Variables: ${JSON.stringify(
      Object.entries(serviceVariables).map(([key, value]) => ({ key, value }))
    )}\n${definition}`
  const splitStoredDefinition = (definition: string) => {
    const marker = /^# AxiOps-File: (.+)$/gm
    const matches = [...definition.matchAll(marker)]
    if (!matches.length) return null
    const files: Record<string, string> = {}
    matches.forEach((match, index) => {
      const name = match[1].trim()
      const start = (match.index || 0) + match[0].length
      const end =
        index + 1 < matches.length
          ? matches[index + 1].index || definition.length
          : definition.length
      files[name] = definition
        .slice(start, end)
        .replace(/^\r?\n/, '')
        .trimStart()
        .replace(/\n---\s*$/, '')
    })
    return Object.keys(files).length ? files : null
  }
  const serializedFiles = () =>
    serviceFiles.value
      .map((file) => `# AxiOps-File: ${file}\n${fileContents[file] || ''}`.trimEnd())
      .join('\n---\n')
  const hydrateStoredDefinition = (definition: string) => {
    const storedFiles = splitStoredDefinition(definition)
    if (!storedFiles) {
      serviceFiles.value = ['deployment.yaml']
      Object.keys(fileContents).forEach((key) => delete fileContents[key])
      activeFileName.value = serviceFiles.value[0]
      fileContents[activeFileName.value] = definition.replace(/^# AxiOps-Variables:.*\r?\n/gm, '')
      form.definition = fileContents[activeFileName.value]
      return
    }
    serviceFiles.value = Object.keys(storedFiles)
    Object.keys(fileContents).forEach((key) => delete fileContents[key])
    Object.assign(fileContents, storedFiles)
    activeFileName.value = serviceFiles.value[0]
    form.definition = definition
  }
  const loadExistingDefinition = async () => {
    if (!props.service?.id) return
    const version = ++definitionVersion
    definitionLoading.value = true
    try {
      const revisions = await listServiceRevisionsApi(props.service.id)
      if (version !== definitionVersion) return
      const latest = [...(revisions || [])].sort((a, b) => b.revision_number - a.revision_number)[0]
      if (latest?.definition) {
        if (props.deliveryType === 'helm') {
          const chart = JSON.parse(latest.definition)
          serviceFiles.value = chart.files.map((file: { path: string }) => file.path)
          Object.keys(fileContents).forEach((key) => delete fileContents[key])
          chart.files.forEach((file: { path: string; content: string }) => {
            fileContents[file.path] = file.content
          })
          activeFileName.value = serviceFiles.value[0]
          form.definition = fileContents[activeFileName.value]
        } else {
          hydrateStoredDefinition(latest.definition)
          const metadata = latest.definition.match(/^# AxiOps-Variables: (.+)$/m)
          if (metadata)
            JSON.parse(metadata[1]).forEach((v: { key: string; value: string }) => {
              serviceVariables[v.key] = v.value
            })
        }
      }
    } catch (error: any) {
      if (version !== definitionVersion) return
      definitionLoadError.value = true
      ElMessage.error(error?.message || '服务配置读取失败，请关闭后重试，不能用默认配置覆盖原服务')
    } finally {
      if (version === definitionVersion) definitionLoading.value = false
    }
  }
  const validateCurrentStep = async () => {
    if (definitionLoadError.value || definitionLoading.value || submitting.value) return false
    if (
      !editing.value &&
      props.deliveryType === 'helm' &&
      form.source_type !== 'git' &&
      (form.source_type !== 'template' || !selectedLibraryTemplate.value)
    ) {
      ElMessage.warning('请先选择 Helm 模板，或从代码仓库导入 Chart')
      return false
    }
    if (step.value !== 1) return true
    return Boolean(await formRef.value?.validate().catch(() => false))
  }
  const createBuildConfig = async (serviceId: number) => {
    if (!form.init_build || !props.canUpdateBuild) return
    const language = form.language.toLowerCase()
    const presets: Record<string, { image: string; command: string; cache: string }> = {
      go: {
        image: 'golang:1.24-alpine',
        command: 'go test ./... && go build ./...',
        cache: '/go/pkg/mod'
      },
      java: {
        image: 'maven:3.9-eclipse-temurin-21',
        command: 'mvn -B clean verify',
        cache: '/root/.m2/repository'
      },
      'node.js': {
        image: 'node:22-alpine',
        command: 'npm ci && npm test && npm run build',
        cache: '/root/.npm'
      },
      python: {
        image: 'python:3.13-alpine',
        command: 'pip install -r requirements.txt && pytest',
        cache: '/root/.cache/pip'
      }
    }
    const preset = presets[language] || {
      image: 'ubuntu:24.04',
      command: 'echo "configure build command"',
      cache: ''
    }
    await saveBuildConfigApi(serviceId, {
      template_type: language || 'custom',
      build_image: preset.image,
      build_command: preset.command,
      dockerfile_path: 'Dockerfile',
      build_context: '.',
      image_repo: form.image_name.trim(),
      cache_enabled: true,
      cache_dir: preset.cache,
      timeout_seconds: 1800,
      branch_rules: { includes: ['main'], excludes: [] },
      env_vars: [],
      resource_cpu: '',
      resource_memory: '',
      provider: 'kube_job'
    })
  }
  const createWorkflow = async (serviceId: number, revisionId: number) => {
    if (!form.init_workflow || !props.canCreateWorkflow) return
    const deployType =
      props.deliveryType === 'helm'
        ? 'helm_deploy'
        : props.deliveryType === 'host'
          ? 'host_deploy'
          : 'yaml_deploy'
    const nodes: any[] = [
      {
        key: 'build',
        name: '构建',
        type: 'build',
        config: { service_id: serviceId, branch: form.repo_branch },
        timeout_seconds: 3600,
        retry_limit: 1
      }
    ]
    let dependency = 'build'
    if (form.include_test) {
      nodes.push({
        key: 'test',
        name: '自动化测试',
        type: 'test',
        depends_on: ['build'],
        config: { service_id: serviceId, command: 'echo "configure test command"' },
        timeout_seconds: 1800,
        retry_limit: 0
      })
      dependency = 'test'
    }
    if (revisionId)
      nodes.push({
        key: 'deploy',
        name: '部署',
        type: deployType,
        depends_on: [dependency],
        config: {
          service_id: serviceId,
          service_revision_id: revisionId,
          ...(props.deliveryType === 'helm' ? { release_name: form.code } : {})
        },
        timeout_seconds: 1800,
        retry_limit: 0
      })
    await createWorkflowApi(props.projectId, {
      name: `${form.name} · 标准交付`,
      code: `${form.code}-delivery`.slice(0, 64),
      description: '由服务创建向导生成，可在工作流编排中继续调整',
      definition: { delivery_type: props.deliveryType as DeliveryType, nodes }
    })
  }
  const submit = async () => {
    if (!(await validateCurrentStep())) return
    submitting.value = true
    try {
      let preparedDefinition = ''
      if (form.source_type !== 'git' && props.deliveryType !== 'host') {
        preparedDefinition = (
          await previewDeliveryTemplateApi({
            id: 0,
            kind: props.deliveryType === 'helm' ? 'helm' : 'k8s_yaml',
            name: form.name,
            description: '',
            revision: 0,
            files: serviceFiles.value.map((path) => ({ path, content: fileContents[path] || '' })),
            variables:
              props.deliveryType === 'helm'
                ? []
                : Object.entries(serviceVariables).map(([key, value]) => ({
                    key,
                    value,
                    description: ''
                  }))
          })
        ).definition
      }
      const payload: CreateServiceInput = {
        name: form.name.trim(),
        code: form.code.trim(),
        description: form.description.trim(),
        source_type: form.source_type,
        repo_url: form.repo_url.trim(),
        repo_branch: form.repo_branch.trim() || 'main',
        language: form.language.trim(),
        framework: form.framework.trim(),
        image_name: form.image_name.trim(),
        code_source_id: form.code_source_id,
        repository_id: form.repository_id
      }
      let serviceId = props.service?.id || 0
      if (serviceId) await updateServiceApi(serviceId, payload)
      else serviceId = (await createServiceApi(props.projectId, payload)).id
      const serviceForImport =
        props.service ||
        ({
          ...payload,
          delivery_type: props.deliveryType,
          id: serviceId,
          project_id: props.projectId,
          current_revision_id: 0,
          component_count: 0,
          status: 'active',
          updated_at: ''
        } as AxiOpsService)
      let revisionId = props.service?.current_revision_id || 0
      if (
        form.source_type === 'git' &&
        (props.deliveryType === 'helm' || form.paths.trim()) &&
        form.code_source_id
      ) {
        const request = {
          code_source_id: form.code_source_id,
          repo_path: form.repo_path.trim(),
          ref: form.repo_branch.trim() || 'main',
          paths:
            props.deliveryType === 'helm'
              ? []
              : form.paths
                  .split(/\r?\n/)
                  .map((item) => item.trim())
                  .filter(Boolean),
          ...(props.deliveryType === 'helm' ? { chart_path: form.chart_path.trim() } : {}),
          ...(props.deliveryType === 'host'
            ? { spec_path: form.paths.split(/\r?\n/).find(Boolean) }
            : {})
        }
        const preview: ServiceImportPreview = await previewServiceImportApi(
          serviceForImport,
          request
        )
        const revision = await importServiceRevisionApi(serviceForImport, {
          ...request,
          expected_commit: preview.commit_id,
          expected_revision_id: preview.current_revision_id ?? serviceForImport.current_revision_id
        })
        revisionId = revision.id
      } else if (
        form.source_type !== 'git' &&
        serviceFiles.value.some((file) => (fileContents[file] || '').trim())
      ) {
        const library = selectedLibraryTemplate.value
        const unchanged =
          !!library &&
          library.files.length === serviceFiles.value.length &&
          library.files.every((file) => fileContents[file.path] === file.content)
        const revision = await createServiceRevisionApi(serviceId, {
          template:
            props.deliveryType === 'helm' && unchanged
              ? { id: library!.id, revision: library!.revision }
              : undefined,
          source_commit:
            selectedLibraryTemplate.value && props.deliveryType !== 'helm'
              ? `template:${selectedLibraryTemplate.value.id}@${selectedLibraryTemplate.value.revision}`
              : form.source_type,
          definition:
            preparedDefinition || addVariableMetadata(materializeDefinition(serializedFiles()))
        })
        revisionId = revision.id
      }
      await createBuildConfig(serviceId)
      await createWorkflow(serviceId, revisionId)
      ElMessage.success(editing.value ? '服务已更新' : '服务已创建并完成初始化')
      drawerVisible.value = false
      emit('saved')
    } catch (error: any) {
      ElMessage.error(error?.message || '服务保存失败，请检查代码源和定义文件')
    } finally {
      submitting.value = false
    }
  }
  watch(
    () => [props.visible, props.service?.id],
    ([visible]) => {
      if (visible) {
        reset()
        void loadExistingDefinition()
      } else {
        definitionVersion++
        definitionLoading.value = false
      }
    }
  )
</script>

<style scoped lang="scss">
  .wizard-shell {
    min-height: 100vh;
    padding: 0 0 72px;
    background: var(--art-main-bg-color);
    color: var(--art-text-gray-900);
  }
  .onboarding-header {
    min-height: 64px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 20px;
    padding: 0 28px;
    border-bottom: 1px solid var(--art-border-color);
    background: var(--art-bg-color);
  }
  .onboarding-breadcrumb,
  .onboarding-envs,
  .axiops-brand {
    display: flex;
    align-items: center;
  }
  .onboarding-breadcrumb {
    min-width: 0;
    gap: 10px;
    color: var(--art-text-gray-600);
    font-size: 13px;
  }
  .onboarding-breadcrumb strong {
    overflow: hidden;
    color: var(--art-text-gray-900);
    font-size: 14px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .axiops-brand {
    gap: 7px;
    color: var(--art-text-gray-900);
    font-size: 17px;
    letter-spacing: 0;
  }
  .axiops-brand-logo {
    width: 27px;
    height: 27px;
    flex: 0 0 27px;
    overflow: hidden;
    border-radius: 7px;
  }
  .breadcrumb-divider {
    width: 1px;
    height: 22px;
    background: var(--art-border-color);
  }
  .back-button {
    width: 28px;
    height: 28px;
    padding: 0;
    color: var(--art-text-gray-600);
    font-size: 23px;
    line-height: 24px;
    background: transparent;
    border: 0;
    border-radius: 4px;
    cursor: pointer;
  }
  .back-button:hover {
    color: var(--el-color-primary);
    background: var(--art-gray-100);
  }
  .breadcrumb-separator {
    color: var(--art-text-gray-400);
    font-size: 18px;
  }
  .onboarding-envs {
    justify-content: flex-end;
    gap: 7px;
  }
  .env-placeholder {
    color: var(--art-text-gray-500);
    font-size: 11px;
  }
  .onboarding-progress {
    display: grid;
    grid-template-columns: 150px minmax(0, 760px);
    align-items: start;
    gap: 18px;
    padding: 15px 28px 12px;
    background: #f5f7f7;
  }
  .guide-name {
    width: 110px;
    padding: 8px;
    color: #fff;
    font-size: 18px;
    font-weight: 300;
    line-height: 24px;
    text-align: center;
    background: #3289e4;
  }
  .onboarding-progress :deep(.el-steps) {
    width: 100%;
    padding-top: 3px;
  }
  .onboarding-progress :deep(.el-step__icon) {
    width: 18px;
    height: 18px;
    font-size: 11px;
  }
  .onboarding-progress :deep(.el-step__line) {
    top: 8px;
  }
  .onboarding-progress :deep(.el-step__title) {
    font-size: 14px;
    line-height: 28px;
  }
  .current-step-title {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 0 28px 12px;
    background: #f5f7f7;
  }
  .current-step-title strong {
    width: 110px;
    padding: 8px;
    color: #fff;
    font-size: 18px;
    font-weight: 300;
    line-height: 24px;
    text-align: center;
    background: #3289e4;
  }
  .current-step-title span {
    color: #4c4c4c;
    font-size: 13px;
  }
  .wizard-intro {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 16px;
    padding: 24px 28px 18px;
    background: var(--art-bg-color);
  }
  .eyebrow {
    color: var(--el-color-primary);
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.12em;
  }
  .wizard-intro h2 {
    margin: 6px 0 4px;
    font-size: 21px;
    letter-spacing: 0;
  }
  .wizard-intro p {
    margin: 0;
    max-width: 690px;
    color: var(--art-text-gray-500);
    font-size: 12px;
    line-height: 18px;
  }
  .step-counter {
    flex: 0 0 auto;
    padding: 5px 9px;
    color: var(--el-color-primary);
    font-size: 11px;
    background: var(--el-color-primary-light-9);
    border: 1px solid var(--el-color-primary-light-7);
    border-radius: 4px;
  }
  .wizard-steps {
    margin: 4px 0 24px;
  }
  .wizard-section {
    padding: 15px 5px 0;
    background: #f5f7f7;
  }
  .section-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    margin-bottom: 16px;
  }
  .section-heading > div {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .section-heading strong {
    font-size: 15px;
  }
  .section-heading span {
    color: var(--art-text-gray-500);
    font-size: 11px;
  }
  .mode-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px;
    margin-bottom: 18px;
  }
  .source-dialog-tip {
    margin: 0 0 16px;
    color: #606266;
    font-size: 13px;
  }
  .source-mode-grid {
    margin-bottom: 0;
  }
  .source-mode-grid.is-helm {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .template-dialog-layout {
    min-height: 500px;
    display: grid;
    grid-template-columns: 260px minmax(0, 1fr);
    overflow: hidden;
    border: 1px solid #dcdfe6;
    background: #fff;
  }
  .template-catalog {
    padding: 12px 10px;
    background: #f7f8fa;
    border-right: 1px solid #dcdfe6;
  }
  .template-catalog-heading {
    padding: 5px 8px 12px;
    color: #303133;
    font-size: 13px;
    font-weight: 600;
  }
  .template-item {
    width: 100%;
    min-height: 66px;
    display: grid;
    grid-template-columns: 34px minmax(0, 1fr) 16px;
    align-items: center;
    gap: 9px;
    padding: 9px;
    color: #303133;
    text-align: left;
    background: transparent;
    border: 1px solid transparent;
    cursor: pointer;
  }
  .template-item.is-active {
    color: #1989fa;
    background: rgb(25 137 250 / 10%);
    border-color: rgb(25 137 250 / 28%);
  }
  .template-item-icon {
    width: 32px;
    height: 32px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    color: #1989fa;
    background: #fff;
    border: 1px solid #dcdfe6;
  }
  .template-item > span:nth-child(2) {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .template-item strong {
    font-size: 13px;
  }
  .template-item small {
    overflow: hidden;
    color: #909399;
    font-size: 11px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .template-preview {
    min-width: 0;
    padding: 14px;
  }
  .template-preview header {
    height: 44px;
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
  }
  .template-preview header > div {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .template-preview header strong {
    font-size: 14px;
  }
  .template-preview header small {
    color: #909399;
    font-size: 11px;
  }
  .empty-editor-layout {
    min-height: calc(100vh - 300px);
    display: grid;
    grid-template-columns: 250px minmax(0, 1fr);
    background: #fff;
    border-bottom: 1px solid #ccc;
    box-shadow: 0 4px 4px rgb(0 0 0 / 5%);
  }
  .empty-service-tree {
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    border-right: 1px solid #e4e7ed;
  }
  .empty-editor-content {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-direction: column;
    gap: 12px;
    color: #909399;
  }
  .empty-editor-content p {
    margin: 0;
    color: #606266;
    font-size: 15px;
  }

  .service-config-layout {
    display: grid;
    grid-template-columns: 250px minmax(500px, 1fr) minmax(372px, 0.74fr);
    min-height: calc(100vh - 300px);
    border-bottom: 1px solid #ccc;
    background: var(--art-bg-color);
    box-shadow: 0 4px 4px rgb(0 0 0 / 5%);
  }
  .service-tree-panel,
  .variable-panel {
    min-width: 0;
    display: flex;
    flex-direction: column;
    background: #fbfcfe;
  }
  .service-tree-panel {
    border-right: 1px solid var(--art-border-color);
  }
  .variable-panel {
    border-left: 1px solid var(--art-border-color);
  }
  .tree-panel-heading,
  .variable-panel-heading {
    min-height: 58px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 0 14px;
    border-bottom: 1px solid var(--art-border-color);
  }
  .tree-panel-heading > div,
  .variable-panel-heading {
    min-width: 0;
  }
  .tree-panel-heading > div {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .tree-panel-heading span,
  .variable-panel-heading > span {
    font-size: 13px;
    font-weight: 600;
  }
  .tree-panel-heading small {
    color: var(--art-text-gray-500);
    font-size: 10px;
  }
  .tree-panel-list {
    flex: 1;
    padding: 9px;
  }
  .tree-service-item {
    width: 100%;
    display: grid;
    grid-template-columns: 7px minmax(0, 1fr);
    align-items: center;
    gap: 8px;
    padding: 10px 9px;
    text-align: left;
    color: inherit;
    background: transparent;
    border: 1px solid transparent;
    border-radius: 5px;
    cursor: pointer;
  }
  .tree-service-item.is-active {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary-light-7);
  }
  .tree-service-status {
    width: 7px;
    height: 7px;
    display: inline-block;
    border-radius: 50%;
    background: #22c55e;
  }
  .tree-service-item span:last-child {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .tree-service-item strong,
  .tree-service-item small {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .tree-service-item strong {
    font-size: 12px;
  }
  .tree-service-item small {
    color: var(--art-text-gray-500);
    font-size: 10px;
  }
  .tree-panel-footer {
    padding: 10px;
    border-top: 1px solid var(--art-border-color);
  }
  .service-definition-column {
    min-width: 0;
    padding: 18px 20px 12px;
    overflow: hidden;
  }
  .definition-files-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    min-height: 46px;
    padding: 0 2px;
    border-bottom: 1px solid var(--art-border-color);
  }
  .tree-file-list {
    margin: -2px 10px 8px 28px;
    padding-left: 8px;
    border-left: 1px solid var(--art-border-color);
  }
  .tree-file-item {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 6px;
    min-height: 26px;
    padding: 0 7px;
    border: 0;
    border-radius: 4px;
    background: transparent;
    color: var(--art-text-gray-500);
    font-size: 10px;
    text-align: left;
    cursor: pointer;
  }
  .tree-file-item:hover,
  .tree-file-item.is-active {
    color: var(--el-color-primary);
    background: var(--art-gray-100);
  }
  .definition-files-heading > div:first-child {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .definition-files-heading strong {
    font-size: 12px;
  }
  .definition-files-heading small {
    color: var(--art-text-gray-500);
    font-size: 10px;
  }
  .definition-files-actions {
    display: flex;
    align-items: center;
    gap: 4px;
  }
  .definition-files-actions .el-input {
    width: 150px;
  }
  .definition-files-list {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 7px 0;
    overflow-x: auto;
  }
  .definition-file-item {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    min-height: 27px;
    padding: 0 9px;
    border: 1px solid transparent;
    border-radius: 4px;
    background: transparent;
    color: var(--art-text-gray-600);
    font-size: 11px;
    cursor: pointer;
  }
  .definition-file-item:hover {
    color: var(--el-color-primary);
    background: var(--art-gray-100);
  }
  .definition-file-item.is-active {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary-light-7);
  }
  .file-remove {
    margin-left: auto;
    color: var(--art-text-gray-400);
  }
  .file-remove:hover {
    color: var(--el-color-danger);
  }
  .service-definition-column .wizard-form {
    max-width: 920px;
    margin: 0 auto;
  }
  .variable-panel-heading {
    color: var(--art-text-gray-900);
  }
  .variable-panel-heading .el-button {
    max-width: 130px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .variable-section {
    padding: 16px 14px;
    border-bottom: 1px solid var(--art-border-color);
  }
  .variable-section-title {
    display: flex;
    flex-direction: column;
    gap: 3px;
    margin-bottom: 10px;
  }
  .variable-section-title strong {
    font-size: 12px;
  }
  .variable-section-title small {
    color: var(--art-text-gray-500);
    font-size: 10px;
  }
  .variable-list {
    margin: 0;
  }
  .variable-list div {
    display: grid;
    grid-template-columns: 112px minmax(0, 1fr);
    gap: 8px;
    padding: 8px 0;
    border-bottom: 1px solid var(--art-border-color);
  }
  .variable-list div:last-child {
    border-bottom: 0;
  }
  .variable-list dt,
  .variable-list dd {
    overflow: hidden;
    margin: 0;
    font-size: 11px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .variable-list dt {
    color: var(--art-text-gray-600);
  }
  .variable-list dd {
    color: var(--art-text-gray-800);
  }
  .service-variable-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .service-variable-list label {
    display: grid;
    grid-template-columns: 100px minmax(0, 1fr);
    align-items: center;
    gap: 8px;
  }
  .service-variable-list label > span {
    color: var(--art-text-gray-700);
    font-family: var(--ax-font-mono);
    font-size: 11px;
  }
  .mode-card {
    min-height: 144px;
    display: flex;
    align-items: flex-start;
    gap: 11px;
    position: relative;
    padding: 16px;
    border: 1px solid var(--art-border-color);
    border-radius: 8px;
    background: var(--art-bg-color);
    color: inherit;
    text-align: left;
    cursor: pointer;
    transition:
      border-color 0.18s,
      box-shadow 0.18s,
      transform 0.18s;
  }
  .mode-card:hover {
    border-color: var(--el-color-primary-light-4);
    transform: translateY(-1px);
  }
  .mode-card.is-active {
    border-color: var(--el-color-primary);
    box-shadow:
      0 0 0 1px var(--el-color-primary),
      0 10px 24px rgba(37, 99, 235, 0.1);
  }
  .mode-icon,
  .option-icon {
    width: 35px;
    height: 35px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex: 0 0 35px;
    border-radius: 8px;
  }
  .mode-icon.is-git {
    color: #2563eb;
    background: #eaf1ff;
  }
  .mode-icon.is-manual {
    color: #0f766e;
    background: #e7f8f5;
  }
  .mode-icon.is-template {
    color: #c2410c;
    background: #fff2e8;
  }
  .mode-copy {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 7px;
  }
  .mode-copy strong {
    font-size: 13px;
  }
  .mode-copy small {
    color: var(--art-text-gray-500);
    font-size: 11px;
    line-height: 17px;
  }
  .mode-check {
    position: absolute;
    top: 14px;
    right: 14px;
    color: var(--el-color-primary);
  }
  .wizard-form :deep(.el-form-item) {
    margin-bottom: 15px;
  }
  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0 14px;
  }
  .form-grid.thirds {
    grid-template-columns: repeat(3, 1fr);
  }
  .field-hint {
    display: block;
    margin-top: 5px;
    color: var(--art-text-gray-500);
    font-size: 11px;
  }
  .repository-browser {
    margin: 2px 0 14px;
    border: 1px solid var(--art-border-color);
    border-radius: 7px;
    background: #fbfcfe;
  }
  .browser-toolbar {
    min-height: 48px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 0 12px;
    border-bottom: 1px solid var(--art-border-color);
  }
  .browser-toolbar > div {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .browser-toolbar strong {
    font-size: 12px;
  }
  .browser-toolbar small {
    color: var(--art-text-gray-500);
    font-family: var(--ax-font-mono);
    font-size: 10px;
  }
  .repository-browser :deep(.el-table) {
    background: transparent;
  }
  .repository-browser :deep(.el-button) {
    gap: 5px;
  }
  .template-strip {
    display: flex;
    align-items: center;
    gap: 12px;
    min-height: 43px;
    padding: 0 12px;
    margin-bottom: 12px;
    border: 1px solid var(--art-border-color);
    border-radius: 6px;
    background: #fbfcfe;
  }
  .template-strip > span {
    color: var(--art-text-gray-600);
    font-size: 11px;
  }
  .definition-editor {
    overflow: hidden;
    border: 1px solid #1e293b;
    border-radius: 7px;
    box-shadow: 0 8px 22px rgba(15, 23, 42, 0.08);
  }
  .delivery-options {
    margin-top: 3px;
    border-top: 1px solid var(--art-border-color);
  }
  .delivery-option-row {
    min-height: 66px;
    display: grid;
    grid-template-columns: 35px minmax(0, 1fr) auto;
    align-items: center;
    gap: 11px;
    border-bottom: 1px solid var(--art-border-color);
  }
  .delivery-option-row > span:nth-child(2) {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .delivery-option-row strong {
    font-size: 12px;
  }
  .delivery-option-row small {
    color: var(--art-text-gray-500);
    font-size: 10px;
  }
  .option-icon.is-build {
    color: #2563eb;
    background: #eaf1ff;
  }
  .option-icon.is-workflow {
    color: #7c3aed;
    background: #f2edff;
  }
  .option-icon.is-test {
    color: #0f766e;
    background: #e7f8f5;
  }
  .review-panel {
    margin-top: 22px;
    padding: 14px;
    border: 1px solid var(--art-border-color);
    border-radius: 7px;
    background: #f8fafc;
  }
  .review-heading {
    display: flex;
    flex-direction: column;
    gap: 3px;
    margin-bottom: 12px;
  }
  .review-heading strong {
    font-size: 13px;
  }
  .review-heading span {
    color: var(--art-text-gray-500);
    font-size: 10px;
  }
  .review-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px 24px;
    margin: 0;
  }
  .review-grid div {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .review-grid dt {
    color: var(--art-text-gray-500);
    font-size: 10px;
  }
  .review-grid dd {
    min-width: 0;
    margin: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 12px;
  }
  .review-grid code {
    margin-left: 5px;
    color: var(--art-text-gray-500);
    font-size: 10px;
  }
  .wizard-footer {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .step-note {
    color: var(--art-text-gray-500);
    font-size: 11px;
  }
  @media (max-width: 1100px) {
    .service-config-layout {
      grid-template-columns: 220px minmax(420px, 1fr) 300px;
    }
  }
  @media (max-width: 900px) {
    .onboarding-header {
      align-items: flex-start;
      flex-direction: column;
      gap: 10px;
      padding-top: 12px;
      padding-bottom: 12px;
    }
    .onboarding-envs {
      justify-content: flex-start;
    }
    .onboarding-progress {
      grid-template-columns: 110px minmax(0, 1fr);
      padding-right: 16px;
      padding-left: 16px;
    }
    .service-config-layout {
      grid-template-columns: 190px minmax(0, 1fr);
    }
    .variable-panel {
      display: grid;
      grid-column: 1 / -1;
      grid-template-columns: 1fr 1fr;
      border-top: 1px solid var(--art-border-color);
      border-left: 0;
    }
    .variable-panel-heading {
      grid-column: 1 / -1;
    }
  }
  @media (max-width: 720px) {
    .onboarding-progress {
      grid-template-columns: 1fr;
      gap: 9px;
      padding: 12px 16px;
    }
    .guide-name,
    .current-step-title strong {
      width: 96px;
      font-size: 16px;
    }
    .current-step-title {
      align-items: flex-start;
      flex-direction: column;
      gap: 8px;
      padding: 0 16px 12px;
    }
    .service-config-layout {
      grid-template-columns: 1fr;
    }
    .service-tree-panel {
      max-height: 170px;
      border-right: 0;
      border-bottom: 1px solid var(--art-border-color);
    }
    .variable-panel {
      display: block;
    }
    .variable-panel-heading {
      display: flex;
    }
    .mode-grid,
    .source-mode-grid.is-helm,
    .form-grid.thirds {
      grid-template-columns: 1fr;
    }
    .form-grid,
    .review-grid {
      grid-template-columns: 1fr;
    }
    .mode-card {
      min-height: auto;
    }
    .template-dialog-layout {
      grid-template-columns: 1fr;
    }
    .template-catalog {
      border-right: 0;
      border-bottom: 1px solid #dcdfe6;
    }
  }
</style>
