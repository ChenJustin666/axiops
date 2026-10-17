<template>
  <div class="tekton-template-page" :class="{ 'editor-active': isEditorMode }">
    <template v-if="!isEditorMode">
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
        :data="templates"
        :pagination="pagination"
        empty-text="暂无 Tekton Pipeline 模板"
        @add="enterTemplateEditor('add')"
        @refresh="refreshAll"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      >
        <template #default="{ item }">
          <article class="template-card">
            <span class="template-icon" :style="{ color: item.iconColor || '#2563eb' }">
              <ArtSvgIcon :icon="item.icon || 'simple-icons:tekton'" />
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
                {{ item.scope === 'system' ? '全局模板' : '项目模板' }}
              </ElTag>
              <ElTag size="small" type="info">{{ templateSummary(item).taskCount }} 个 Task</ElTag>
              <ElTag size="small" type="info">
                {{ templateSummary(item).paramCount }} 个参数
              </ElTag>
              <ElTag size="small" type="info">
                {{ templateSummary(item).workspaceCount }} 个 Workspace
              </ElTag>
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

    <section v-else class="template-editor">
      <aside class="template-side">
        <ElForm
          ref="formRef"
          class="base-form"
          :model="formData"
          :rules="rules"
          label-width="86px"
          :disabled="isViewMode"
        >
          <div class="side-title">
            <span class="template-icon large" :style="{ color: formData.iconColor || '#2563eb' }">
              <ArtSvgIcon :icon="formData.icon || 'simple-icons:tekton'" />
            </span>
            <span>
              <strong>{{ editorTitle }}</strong>
              <em>Tekton Pipeline 模板</em>
            </span>
          </div>

          <ElFormItem label="名称" prop="name">
            <ElInput v-model="formData.name" placeholder="请输入模板名称" />
          </ElFormItem>
          <ElFormItem label="编码" prop="code">
            <ElInput
              v-model="formData.code"
              :disabled="editorMode !== 'add'"
              placeholder="例如 java_build_deploy"
              @blur="syncPipelineName"
            />
          </ElFormItem>
          <ElFormItem label="级别" prop="scope">
            <ElSelect v-model="formData.scope" style="width: 100%">
              <ElOption v-if="isSuperAdmin" label="全局模板" value="system" />
              <ElOption label="项目模板" value="project" />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="所属项目">
            <ElInput
              :model-value="projectStore.selectedProject?.name || '未选择项目'"
              :disabled="formData.scope !== 'project'"
            />
          </ElFormItem>
          <ElFormItem label="状态">
            <ElRadioGroup v-model="formData.status">
              <ElRadio :value="1">启用</ElRadio>
              <ElRadio :value="0">停用</ElRadio>
            </ElRadioGroup>
          </ElFormItem>
          <ElFormItem label="图标">
            <div class="icon-picker">
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
          <ElFormItem label="描述">
            <ElInput
              v-model="formData.description"
              type="textarea"
              :rows="3"
              placeholder="说明适用场景、输入契约和产物"
            />
          </ElFormItem>
        </ElForm>

        <div class="template-contract">
          <strong>模板契约</strong>
          <span
            >参数和 Workspace 来自 Pipeline 编排，不在这里选择 PVC、Secret 或配置中心资源。</span
          >
          <dl>
            <dt>Task</dt>
            <dd>{{ currentSummary.taskCount }}</dd>
            <dt>参数</dt>
            <dd>{{ currentSummary.paramCount }}</dd>
            <dt>Workspace</dt>
            <dd>{{ currentSummary.workspaceCount }}</dd>
          </dl>
        </div>

        <div class="side-actions">
          <ElButton @click="handleEditorCancel">{{ isViewMode ? '返回' : '取消' }}</ElButton>
          <ElButton
            v-if="!isViewMode"
            type="primary"
            :loading="submitLoading"
            @click="saveTemplateEditor"
          >
            保存模板
          </ElButton>
        </div>
      </aside>

      <main class="template-main">
        <DevopsTektonDagDesigner
          v-model="dagConfig"
          v-model:category-id="taskFilter.categoryId"
          :library-items="tektonLibraryItems"
          :category-options="categoryOptions"
          :library-loading="taskLoading"
          workspace-mode="contract"
          :readonly="isViewMode"
          @refresh-library="fetchTaskLibrary"
          @save="saveTemplateEditor"
        />
      </main>
    </section>
  </div>
</template>

<script setup lang="ts">
  import { computed, defineAsyncComponent, onActivated, onMounted, reactive, ref, watch } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import type {
    TektonDagConfig,
    TektonDagLibraryItem
  } from '@/components/devops-tekton-dag-designer/types'
  import { validateTektonDagConfig } from '@/components/devops-tekton-dag-designer/validate'
  import { renderTektonDagPipelineYAML } from '@/components/devops-tekton-dag-designer/render-yaml'
  import { useDevopsProjectStore } from '@/store/modules/devopsProject'
  import { useUserStore } from '@/store/modules/user'
  import {
    createDevopsPipelineTemplateApi,
    deleteDevopsPipelineTemplateApi,
    getDevopsPipelineTemplateApi,
    listDevopsPipelineTemplateApi,
    listDevopsStepCategoryApi,
    listDevopsTektonStepApi,
    updateDevopsPipelineTemplateApi,
    type DevopsPipelineTemplate,
    type DevopsPipelineTemplateForm,
    type DevopsStepCategory,
    type DevopsStepTemplate
  } from '@/api/devops/pipeline'

  defineOptions({ name: 'DevOpsTektonTemplateLibrary' })

  const DevopsTektonDagDesigner = defineAsyncComponent(
    () => import('@/components/devops-tekton-dag-designer/index.vue')
  )

  const projectStore = useDevopsProjectStore()
  const userStore = useUserStore()
  const loading = ref(false)
  const taskLoading = ref(false)
  const submitLoading = ref(false)
  const showSearchBar = ref(true)
  const editorMode = ref<'browse' | 'add' | 'edit' | 'view'>('browse')
  const formRef = ref<FormInstance>()
  const currentTemplate = ref<DevopsPipelineTemplate | null>(null)
  const templates = ref<DevopsPipelineTemplate[]>([])
  const categories = ref<DevopsStepCategory[]>([])
  const taskLibrary = ref<DevopsStepTemplate[]>([])
  const taskFilter = reactive({ categoryId: '' })
  const searchParams = ref<Record<string, any>>({})
  const searchForm = ref({
    name: undefined,
    code: undefined,
    scope: undefined,
    status: undefined
  })
  const pagination = ref({ current: 1, size: 15, total: 0 })
  const isEditorMode = computed(() => editorMode.value !== 'browse')
  const isViewMode = computed(() => editorMode.value === 'view')
  const editorTitle = computed(() => {
    if (editorMode.value === 'add') return '新增模板'
    if (editorMode.value === 'view') return '查看模板'
    return '配置模板'
  })
  const isSuperAdmin = computed(() =>
    (userStore.getUserInfo.roles || []).some((role) => String(role).toUpperCase() === 'SUPER_ADMIN')
  )
  const defaultTemplateScope = () => (isSuperAdmin.value ? 'system' : 'project')

  const normalizeTektonName = (value?: string) => {
    const name = String(value || '')
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9-]+/g, '-')
      .replace(/^-+|-+$/g, '')
    return name || 'tekton-pipeline'
  }

  const defaultTektonDagConfig = (): TektonDagConfig => ({
    version: 'tekton-dag/v1',
    pipeline: {
      name: normalizeTektonName(formData.code || 'tekton-pipeline'),
      namespace: '',
      displayName: '',
      description: '',
      serviceAccountName: '',
      timeout: ''
    },
    params: [],
    results: [],
    workspaces: [{ name: 'source', bindingType: 'emptyDir' }],
    nodes: [],
    edges: [],
    runPolicy: {
      managedBy: '',
      workspaceStrategy: 'emptyDir',
      concurrencyPolicy: 'allow',
      cleanupImage: '',
      labels: [],
      annotations: [],
      podTemplateEnv: [],
      podTemplateRaw: '',
      taskRunTemplateRaw: '',
      taskRunSpecsRaw: ''
    },
    triggerConfig: {
      enabled: false,
      apiVersion: 'triggers.tekton.dev/v1beta1',
      provider: 'generic',
      serviceType: 'ClusterIP',
      interceptorType: 'none',
      secretKey: 'token',
      eventTypes: [],
      bindingParams: [],
      templateParams: [],
      labels: [],
      annotations: []
    },
    prunerPolicy: {
      enabled: false,
      clusterResourceCleanup: true,
      platformRecordCleanup: true,
      nativePrunerMode: 'disabled',
      nativePrunerTargetNamespace: '',
      nativePrunerConfigLevel: 'namespace',
      resourceTypes: ['PipelineRun', 'TaskRun'],
      scope: 'pipeline',
      historyLimit: 20,
      successfulHistoryLimit: 10,
      failedHistoryLimit: 10,
      ttlSecondsAfterFinished: 604800,
      platformRecordRetentionDays: 30,
      platformRecordLimit: 200,
      selectorLabels: [],
      selectorAnnotations: []
    },
    scheduleConfig: {
      enabled: false,
      timezone: 'Asia/Shanghai',
      concurrencyPolicy: 'skip',
      parameterSnapshot: true,
      paramsSnapshot: {}
    }
  })

  const defaultForm = (): DevopsPipelineTemplateForm => ({
    name: '',
    code: '',
    icon: 'simple-icons:tekton',
    iconColor: '#2563eb',
    description: '',
    scope: defaultTemplateScope(),
    projectId: defaultTemplateScope() === 'project' ? projectStore.selectedProjectId : '',
    engineType: 'tekton',
    steps: [],
    tektonDagConfig: '',
    tektonRunPolicy: '',
    tektonTriggerConfig: '',
    tektonPrunerPolicyRef: '',
    tektonPipelineYamlSnapshot: '',
    status: 1
  })

  const formData = reactive<DevopsPipelineTemplateForm>(defaultForm())
  const dagConfig = ref<TektonDagConfig>(defaultTektonDagConfig())
  const rules: FormRules = {
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    code: [{ required: true, message: '请输入编码', trigger: 'blur' }],
    scope: [{ required: true, message: '请选择级别', trigger: 'change' }]
  }

  const iconOptions = [
    { icon: 'simple-icons:tekton', color: '#2563eb' },
    { icon: 'ri:flow-chart', color: '#0f766e' },
    { icon: 'simple-icons:kubernetes', color: '#326ce5' },
    { icon: 'simple-icons:docker', color: '#0ea5e9' },
    { icon: 'simple-icons:go', color: '#00add8' },
    { icon: 'simple-icons:java', color: '#b07219' }
  ]

  const searchItems = [
    { label: '名称', key: 'name', type: 'input', props: { clearable: true } },
    { label: '编码', key: 'code', type: 'input', props: { clearable: true } },
    {
      label: '级别',
      key: 'scope',
      type: 'select',
      props: {
        clearable: true,
        options: [
          { label: '全局模板', value: 'system' },
          { label: '项目模板', value: 'project' }
        ]
      }
    },
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

  const categoryOptions = computed(() =>
    categories.value.map((item) => ({
      label: `${item.name}（${item.code}）`,
      value: item.id,
      code: item.code
    }))
  )
  const categoryCodeMap = computed(() =>
    categories.value.reduce<Record<string, string>>((acc, item) => {
      acc[item.id] = item.code
      return acc
    }, {})
  )
  const firstCategoryId = computed(() => categories.value[0]?.id || '')
  const tektonLibraryItems = computed<TektonDagLibraryItem[]>(() =>
    taskLibrary.value.map((item) => ({
      id: item.id,
      name: item.name,
      code: item.code,
      categoryId: item.categoryId,
      categoryName: item.categoryName,
      categoryCode: categoryCodeMap.value[item.categoryId] || '',
      stageContent: item.stageContent,
      params: item.params || [],
      taskParams: item.taskParams || [],
      taskResults: item.taskResults || [],
      taskWorkspaces: item.taskWorkspaces || [],
      icon: item.icon,
      iconColor: item.iconColor,
      source: 'platform',
      raw: item
    }))
  )
  const currentSummary = computed(() => summarizeDag(dagConfig.value))

  function cloneDag(config: TektonDagConfig): TektonDagConfig {
    return JSON.parse(JSON.stringify(config)) as TektonDagConfig
  }

  function normalizeTektonNameValueList(value: unknown) {
    if (Array.isArray(value)) {
      return value
        .map((item) => ({
          name: String((item as any)?.name || '').trim(),
          value: String((item as any)?.value || '').trim()
        }))
        .filter((item) => item.name && item.value)
    }
    if (value && typeof value === 'object') {
      return Object.entries(value as Record<string, unknown>)
        .map(([name, itemValue]) => ({
          name: String(name || '').trim(),
          value: String(itemValue || '').trim()
        }))
        .filter((item) => item.name && item.value)
    }
    return []
  }

  function normalizeTektonTriggerParamList(value: unknown) {
    if (Array.isArray(value)) {
      return value
        .map((item) => ({
          name: String((item as any)?.name || '').trim(),
          value: String((item as any)?.value || '').trim(),
          description: String((item as any)?.description || '').trim()
        }))
        .filter((item) => item.name)
    }
    if (value && typeof value === 'object') {
      return Object.entries(value as Record<string, unknown>)
        .map(([name, itemValue]) => {
          if (itemValue && typeof itemValue === 'object' && !Array.isArray(itemValue)) {
            return {
              name: String(name || '').trim(),
              value: String((itemValue as any).value || '').trim(),
              description: String((itemValue as any).description || '').trim()
            }
          }
          return {
            name: String(name || '').trim(),
            value: String(itemValue || '').trim(),
            description: ''
          }
        })
        .filter((item) => item.name)
    }
    return []
  }

  function normalizeTektonNodeType(type?: string) {
    switch (
      String(type || '')
        .trim()
        .toLowerCase()
    ) {
      case 'finallytask':
        return 'finally'
      case 'clustertask':
        return 'clusterTask'
      case 'resolvertask':
        return 'resolverTask'
      case 'finally':
      case 'task':
      case 'matrix':
      case 'note':
        return String(type || '').trim() || 'task'
      default:
        return String(type || 'task').trim() || 'task'
    }
  }

  function normalizeTektonTaskRefKind(kind?: string, nodeType?: string) {
    const value = String(kind || '')
      .trim()
      .toLowerCase()
    if (value === 'clustertask') return 'ClusterTask'
    if (value === 'task') return 'Task'
    if (!value && nodeType === 'clusterTask') return 'ClusterTask'
    if (!value) return 'Task'
    return String(kind || '').trim()
  }

  function normalizeTektonTextValue(value: unknown) {
    if (value === undefined || value === null) return ''
    if (typeof value === 'string') return value
    if (Array.isArray(value) || (value && typeof value === 'object')) {
      return JSON.stringify(value, null, 2)
    }
    return String(value)
  }

  function normalizeLegacyResultRef(ref: unknown) {
    if (typeof ref === 'string') {
      const parts = ref
        .split('.')
        .map((item) => String(item || '').trim())
        .filter(Boolean)
      if (parts.length >= 2) return { taskName: parts[0], resultName: parts[1] }
    }
    if (ref && typeof ref === 'object' && !Array.isArray(ref)) {
      const value = ref as Record<string, unknown>
      const taskName = String(value.taskName || value.sourceTask || value.task || '').trim()
      const resultName = String(value.resultName || value.result || '').trim()
      if (taskName && resultName) return { taskName, resultName }
    }
    return null
  }

  function normalizeTektonNodeParams(raw: unknown) {
    const toParam = (source: any, fallbackName = '') => {
      if (!source || typeof source !== 'object' || Array.isArray(source)) {
        if (!fallbackName) return null
        return {
          name: fallbackName,
          source: 'fixed',
          value: normalizeTektonTextValue(source)
        }
      }
      const item = source as Record<string, unknown>
      const name = String(item.name || item.code || fallbackName || '').trim()
      if (!name) return null
      const rawSource = String(item.source || '')
        .trim()
        .toLowerCase()
      const legacyResult = normalizeLegacyResultRef(item.resultRef)
      let nextSource = rawSource
      if (legacyResult) nextSource = 'result'
      if (nextSource === 'param') nextSource = 'pipelineparam'
      if (!['pipelineparam', 'runtime', 'result', 'dynamic'].includes(nextSource)) {
        nextSource = 'fixed'
      }
      return {
        ...item,
        name,
        source:
          nextSource === 'pipelineparam'
            ? 'pipelineParam'
            : nextSource === 'runtime'
              ? 'runtime'
              : nextSource === 'result'
                ? 'result'
                : nextSource === 'dynamic'
                  ? 'dynamic'
                  : 'fixed',
        pipelineParam: String(
          item.pipelineParam || item.paramName || item.sourceParam || ''
        ).trim(),
        value:
          item.value !== undefined
            ? normalizeTektonTextValue(item.value)
            : normalizeTektonTextValue(item.default),
        resultNodeId: String(item.resultNodeId || legacyResult?.taskName || '').trim(),
        resultName: String(item.resultName || legacyResult?.resultName || '').trim(),
        defaultValue: normalizeTektonTextValue(item.defaultValue || item.default)
      }
    }
    if (Array.isArray(raw)) return raw.map((item) => toParam(item)).filter(Boolean)
    if (raw && typeof raw === 'object') {
      return Object.keys(raw as Record<string, unknown>)
        .sort()
        .map((key) => toParam((raw as Record<string, unknown>)[key], key))
        .filter(Boolean)
    }
    return []
  }

  function normalizeTektonNodeWorkspaces(raw: unknown) {
    const toWorkspace = (source: any, fallbackName = '') => {
      if (!source || typeof source !== 'object' || Array.isArray(source)) {
        const taskWorkspace = String(fallbackName || '').trim()
        const pipelineWorkspace = String(source || '').trim()
        if (!taskWorkspace || !pipelineWorkspace) return null
        return { taskWorkspace, pipelineWorkspace }
      }
      const item = source as Record<string, unknown>
      const taskWorkspace = String(item.taskWorkspace || item.name || fallbackName || '').trim()
      const pipelineWorkspace = String(
        item.pipelineWorkspace || item.workspace || item.workspaceName || ''
      ).trim()
      if (!taskWorkspace || !pipelineWorkspace) return null
      return {
        ...item,
        taskWorkspace,
        pipelineWorkspace,
        subPath: String(item.subPath || '').trim(),
        bindingParamCode: String(item.bindingParamCode || '').trim()
      }
    }
    if (Array.isArray(raw)) return raw.map((item) => toWorkspace(item)).filter(Boolean)
    if (raw && typeof raw === 'object') {
      return Object.keys(raw as Record<string, unknown>)
        .sort()
        .map((key) => toWorkspace((raw as Record<string, unknown>)[key], key))
        .filter(Boolean)
    }
    return []
  }

  function normalizeTektonEdges(edges: TektonDagConfig['edges'] = []) {
    return (edges || []).map((edge) => ({
      ...edge,
      targetParam: String(
        (edge as any).targetParam || (edge as any).targetParamName || (edge as any).paramName || ''
      ).trim(),
      whenInput: String((edge as any).whenInput || (edge as any).condition || '').trim(),
      whenOperator: String((edge as any).whenOperator || '').trim() === 'notin' ? 'notin' : 'in',
      whenValues: Array.isArray((edge as any).whenValues)
        ? (edge as any).whenValues.map((item: unknown) => String(item || '').trim()).filter(Boolean)
        : String((edge as any).whenValues || '')
            .split(',')
            .map((item) => item.trim())
            .filter(Boolean)
    }))
  }

  function resolveTektonLegacyResultNodeRefs(nodes: TektonDagConfig['nodes']) {
    const idSet = new Set(nodes.map((node) => String(node.id || '').trim()).filter(Boolean))
    const nameToId = new Map(
      nodes
        .map((node) => [String(node.name || '').trim(), String(node.id || '').trim()] as const)
        .filter(([name, id]) => name && id)
    )
    nodes.forEach((node) => {
      node.params = (node.params || []).map((param) => {
        if (param.source !== 'result') return param
        const current = String(param.resultNodeId || '').trim()
        if (!current || idSet.has(current)) return param
        const resolved = nameToId.get(current)
        return resolved ? { ...param, resultNodeId: resolved } : param
      })
    })
    return nodes
  }

  function mergeTektonItemsByName<T extends { name: string }>(
    primary: T[] = [],
    secondary: T[] = []
  ) {
    const merged = new Map<string, T>()
    ;[...primary, ...secondary].forEach((item) => {
      const name = String(item?.name || '').trim()
      if (!name || merged.has(name)) return
      merged.set(name, { ...item, name })
    })
    return Array.from(merged.values())
  }

  function mergeTektonNameValueGroups(...groups: unknown[]) {
    const merged = new Map<string, { name: string; value: string }>()
    groups.forEach((group) => {
      normalizeTektonNameValueList(group).forEach((item) => {
        merged.set(item.name, item)
      })
    })
    return Array.from(merged.values())
  }

  function normalizeDagCompat(config: TektonDagConfig): TektonDagConfig {
    const next = cloneDag(config)
    const legacyFinallyNodes = (next.finallyNodes || []).map((node) => ({
      ...node,
      type: 'finally'
    }))
    const mergedNodes = new Map<string, TektonDagConfig['nodes'][number]>()
    ;[...(next.nodes || []), ...legacyFinallyNodes].forEach((node) => {
      const key = String(node?.id || node?.name || '').trim()
      if (!key) return
      if (mergedNodes.has(key)) return
      mergedNodes.set(key, {
        ...node,
        name: String(node?.name || (node as any)?.taskName || '').trim(),
        taskCode: String((node as any)?.taskCode || (node as any)?.stepCode || '').trim(),
        type: normalizeTektonNodeType(node?.type) as TektonDagConfig['nodes'][number]['type'],
        taskRef: {
          ...node.taskRef,
          kind: normalizeTektonTaskRefKind(node.taskRef?.kind, normalizeTektonNodeType(node?.type))
        },
        params: normalizeTektonNodeParams((node as any).params),
        workspaces: normalizeTektonNodeWorkspaces((node as any).workspaces)
      })
    })
    next.nodes = resolveTektonLegacyResultNodeRefs(Array.from(mergedNodes.values()))
    next.finallyNodes = []
    next.edges = normalizeTektonEdges(next.edges || [])
    next.params = mergeTektonItemsByName(next.pipeline.params || [], next.params || [])
    next.results = mergeTektonItemsByName(next.pipeline.results || [], next.results || [])
    next.workspaces = mergeTektonItemsByName(
      next.pipeline.workspaces || [],
      mergeTektonItemsByName(next.workspaces || [], next.runPolicy?.workspaces || [])
    )
    next.pipeline.params = []
    next.pipeline.results = []
    next.pipeline.workspaces = []
    if (next.runPolicy) next.runPolicy.workspaces = []
    next.pipeline.labels = mergeTektonNameValueGroups(next.labels, next.pipeline.labels)
    next.pipeline.annotations = mergeTektonNameValueGroups(
      next.annotations,
      next.pipeline.annotations
    )
    next.labels = []
    next.annotations = []
    if (next.runPolicy) {
      next.runPolicy.labels = normalizeTektonNameValueList(next.runPolicy.labels)
      next.runPolicy.annotations = normalizeTektonNameValueList(next.runPolicy.annotations)
      next.runPolicy.podTemplateEnv = normalizeTektonNameValueList(next.runPolicy.podTemplateEnv)
    }
    if (next.triggerConfig) {
      next.triggerConfig.labels = normalizeTektonNameValueList(next.triggerConfig.labels)
      next.triggerConfig.annotations = normalizeTektonNameValueList(next.triggerConfig.annotations)
      next.triggerConfig.bindingParams = normalizeTektonTriggerParamList(
        next.triggerConfig.bindingParams
      )
      next.triggerConfig.templateParams = normalizeTektonTriggerParamList(
        next.triggerConfig.templateParams
      )
    }
    if (next.prunerPolicy) {
      next.prunerPolicy.selectorLabels = normalizeTektonNameValueList(
        next.prunerPolicy.selectorLabels
      )
      next.prunerPolicy.selectorAnnotations = normalizeTektonNameValueList(
        next.prunerPolicy.selectorAnnotations
      )
    }
    next.pipeline.timeout =
      String(next.pipeline.timeout || '').trim() ||
      String(next.runPolicy?.timeout || '').trim() ||
      String(next.runPolicy?.pipelineTimeout || '').trim() ||
      String(next.pipeline.timeouts?.pipeline || '').trim() ||
      String(next.runPolicy?.timeouts?.pipeline || '').trim()
    if (next.runPolicy) {
      next.runPolicy.timeout =
        String(next.runPolicy.timeout || '').trim() ||
        String(next.runPolicy.timeouts?.pipeline || '').trim()
      next.runPolicy.pipelineTimeout =
        String(next.runPolicy.pipelineTimeout || '').trim() ||
        String(next.runPolicy.timeout || '').trim() ||
        String(next.runPolicy.timeouts?.pipeline || '').trim()
      next.runPolicy.tasksTimeout =
        String(next.runPolicy.tasksTimeout || '').trim() ||
        String(next.runPolicy.timeouts?.tasks || '').trim() ||
        String(next.pipeline.timeouts?.tasks || '').trim()
      next.runPolicy.finallyTimeout =
        String(next.runPolicy.finallyTimeout || '').trim() ||
        String(next.runPolicy.timeouts?.finally || '').trim() ||
        String(next.pipeline.timeouts?.finally || '').trim()
      if (
        !String(next.runPolicy.taskRunSpecsRaw || '').trim() &&
        Array.isArray(next.runPolicy.taskRunSpecs) &&
        next.runPolicy.taskRunSpecs.length > 0
      ) {
        next.runPolicy.taskRunSpecsRaw = JSON.stringify(next.runPolicy.taskRunSpecs, null, 2)
      }
    }
    return next
  }

  function parseDagConfig(content?: string): TektonDagConfig {
    if (!String(content || '').trim()) return defaultTektonDagConfig()
    try {
      const parsed = JSON.parse(String(content)) as TektonDagConfig
      return normalizeDagCompat({
        ...defaultTektonDagConfig(),
        ...parsed,
        pipeline: { ...defaultTektonDagConfig().pipeline, ...(parsed.pipeline || {}) },
        params: parsed.params || [],
        results: parsed.results || [],
        workspaces: parsed.workspaces || [],
        nodes: parsed.nodes || [],
        edges: parsed.edges || [],
        runPolicy: { ...defaultTektonDagConfig().runPolicy, ...(parsed.runPolicy || {}) },
        triggerConfig: {
          ...defaultTektonDagConfig().triggerConfig,
          ...(parsed.triggerConfig || {})
        },
        prunerPolicy: { ...defaultTektonDagConfig().prunerPolicy, ...(parsed.prunerPolicy || {}) },
        scheduleConfig: {
          ...defaultTektonDagConfig().scheduleConfig,
          ...(parsed.scheduleConfig || {})
        }
      })
    } catch {
      return defaultTektonDagConfig()
    }
  }

  function summarizeDag(config: TektonDagConfig) {
    return {
      taskCount: (config.nodes || []).filter(
        (item) => item.enabled !== false && item.type !== 'note'
      ).length,
      paramCount: config.params?.length || 0,
      workspaceCount: config.workspaces?.length || 0
    }
  }

  function templateSummary(row: DevopsPipelineTemplate) {
    return summarizeDag(parseDagConfig(row.tektonDagConfig))
  }

  function canManageTemplate(row: DevopsPipelineTemplate) {
    if (row.scope === 'system') return isSuperAdmin.value
    return true
  }

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

  const fetchTaskLibrary = async (payload?: { categoryId?: string }) => {
    if (categories.value.length === 0) await fetchCategories()
    const categoryId = payload?.categoryId || taskFilter.categoryId || firstCategoryId.value
    if (categoryId && taskFilter.categoryId !== categoryId) taskFilter.categoryId = categoryId
    taskLoading.value = true
    try {
      const res = await listDevopsTektonStepApi({
        page: 1,
        pageSize: 200,
        status: 1,
        engineType: 'tekton',
        categoryId: categoryId || undefined
      })
      taskLibrary.value = res.items || []
    } finally {
      taskLoading.value = false
    }
  }

  const fetchTemplates = async () => {
    loading.value = true
    try {
      const res = await listDevopsPipelineTemplateApi({
        page: pagination.value.current,
        pageSize: pagination.value.size,
        projectId:
          !isSuperAdmin.value && projectStore.selectedProjectId
            ? projectStore.selectedProjectId
            : undefined,
        ...searchParams.value,
        engineType: 'tekton'
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

  const syncPipelineName = () => {
    if (isViewMode.value) return
    if (!dagConfig.value.pipeline.name || dagConfig.value.pipeline.name === 'tekton-pipeline') {
      dagConfig.value.pipeline.name = normalizeTektonName(formData.code)
    }
  }

  const enterTemplateEditor = async (
    mode: 'add' | 'edit' | 'view',
    row?: DevopsPipelineTemplate | null
  ) => {
    if (mode === 'edit' && row && !canManageTemplate(row)) {
      ElMessage.warning('只有 SUPER_ADMIN 可以维护全局模板')
      return
    }
    editorMode.value = mode
    formRef.value?.clearValidate()
    taskFilter.categoryId = ''
    if (mode === 'add') {
      currentTemplate.value = null
      Object.assign(formData, defaultForm())
      dagConfig.value = defaultTektonDagConfig()
    } else if (row) {
      const detail = await getDevopsPipelineTemplateApi(row.id)
      currentTemplate.value = detail
      Object.assign(formData, defaultForm(), detail, { engineType: 'tekton', steps: [] })
      dagConfig.value = parseDagConfig(detail.tektonDagConfig)
    }
    if (formData.scope === 'project') {
      formData.projectId = row?.projectId || projectStore.selectedProjectId
    }
    if (mode !== 'view') {
      await fetchTaskLibrary()
    } else {
      taskLibrary.value = []
    }
  }

  const exitTemplateEditor = () => {
    editorMode.value = 'browse'
    formRef.value?.clearValidate()
    currentTemplate.value = null
    Object.assign(formData, defaultForm())
    dagConfig.value = defaultTektonDagConfig()
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

  const saveTemplateEditor = async () => {
    if (isViewMode.value) return
    await formRef.value?.validate()
    syncPipelineName()
    if (formData.scope === 'system' && !isSuperAdmin.value) {
      ElMessage.warning('只有 SUPER_ADMIN 可以创建或维护全局模板')
      return
    }
    if (formData.scope === 'project') {
      if (!projectStore.selectedProjectId) {
        ElMessage.warning('请选择项目')
        return
      }
      formData.projectId = projectStore.selectedProjectId
    } else {
      formData.projectId = ''
    }
    const normalizedDag = normalizeTemplateDagConfig(dagConfig.value)
    const issues = validateTektonDagConfig(normalizedDag)
    const firstError = issues.find((item) => item.level === 'error')
    if (firstError) {
      ElMessage.error(firstError.message)
      return
    }
    submitLoading.value = true
    try {
      dagConfig.value = normalizedDag
      const payload: DevopsPipelineTemplateForm = {
        ...formData,
        engineType: 'tekton',
        steps: [],
        tektonDagConfig: JSON.stringify(normalizedDag),
        tektonRunPolicy: JSON.stringify(normalizedDag.runPolicy || {}),
        tektonTriggerConfig: JSON.stringify({
          ...(normalizedDag.triggerConfig || {}),
          scheduleConfig: normalizedDag.scheduleConfig || {}
        }),
        tektonPrunerPolicyRef: JSON.stringify(normalizedDag.prunerPolicy || {}),
        tektonPipelineYamlSnapshot: renderTektonDagPipelineYAML(normalizedDag)
      }
      if (editorMode.value === 'add') {
        const res = await createDevopsPipelineTemplateApi(payload)
        ElMessage.success('新增成功')
        await fetchTemplates()
        const created = await getDevopsPipelineTemplateApi(res.id)
        await enterTemplateEditor('edit', created)
        exitTemplateEditor()
      } else if (currentTemplate.value) {
        await updateDevopsPipelineTemplateApi(currentTemplate.value.id, payload)
        ElMessage.success('保存成功')
        await fetchTemplates()
        exitTemplateEditor()
      }
    } finally {
      submitLoading.value = false
    }
  }

  function normalizeTemplateDagConfig(config: TektonDagConfig) {
    const next = cloneDag(config)
    next.workspaces = (next.workspaces || []).map((workspace) => ({
      ...workspace,
      claimName: '',
      secretName: '',
      configMapName: '',
      sourceType: undefined,
      platformType: '',
      resourceId: '',
      resourceName: '',
      storageClassName: '',
      storage: '',
      accessMode: '',
      volumeMode: '',
      subPath: '',
      projected: undefined,
      config: undefined
    }))
    return next
  }

  const deleteCurrentTemplate = async (row: DevopsPipelineTemplate) => {
    if (!canManageTemplate(row)) {
      ElMessage.warning('只有 SUPER_ADMIN 可以删除全局模板')
      return
    }
    await ElMessageBox.confirm(`确认删除 Tekton 模板 ${row.name}？`, '删除确认', {
      type: 'warning'
    })
    await deleteDevopsPipelineTemplateApi(row.id)
    ElMessage.success('删除成功')
    await fetchTemplates()
  }

  watch(
    () => projectStore.selectedProjectId,
    () => {
      if (formData.scope === 'project') {
        formData.projectId = projectStore.selectedProjectId
      }
      if (!isEditorMode.value) {
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
  .tekton-template-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .template-card {
    display: grid;
    grid-template-columns: 42px minmax(0, 1fr);
    gap: 8px 10px;
    align-items: start;
    min-height: 184px;
    padding: 12px;
    text-align: left;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    box-shadow: 0 10px 28px rgb(15 23 42 / 4%);

    &:hover {
      border-color: var(--el-color-primary-light-5);
      box-shadow: 0 14px 34px rgb(15 23 42 / 8%);
    }
  }

  .template-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 38px;
    height: 38px;
    font-size: 22px;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;

    &.large {
      width: 48px;
      height: 48px;
      font-size: 28px;
    }
  }

  .template-info {
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

  .template-desc {
    display: -webkit-box;
    grid-column: 1 / -1;
    min-height: 38px;
    margin: 4px 0 0;
    overflow: hidden;
    line-height: 1.5;
    color: var(--el-text-color-regular);
    text-overflow: ellipsis;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }

  .template-tags,
  .template-card-actions {
    display: flex;
    flex-wrap: wrap;
    grid-column: 1 / -1;
    gap: 6px;
  }

  .template-card-actions {
    flex-wrap: nowrap;
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

  .template-editor {
    display: grid;
    flex: 1;
    grid-template-columns: minmax(300px, 332px) minmax(0, 1fr);
    gap: 12px;
    min-height: 0;
    overflow: hidden;
  }

  .template-side,
  .template-main {
    min-height: 0;
    overflow: hidden;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .template-side {
    display: flex;
    flex-direction: column;
  }

  .base-form {
    flex: 1;
    min-height: 0;
    padding: 12px;
    overflow: auto;
  }

  .side-title {
    display: grid;
    grid-template-columns: 48px minmax(0, 1fr);
    gap: 10px;
    align-items: center;
    padding-bottom: 12px;
    margin-bottom: 12px;
    border-bottom: 1px solid var(--el-border-color-lighter);

    strong,
    em {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      font-size: 17px;
    }

    em {
      margin-top: 5px;
      font-style: normal;
      color: var(--el-text-color-secondary);
    }
  }

  .icon-picker {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;

    button {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 32px;
      height: 32px;
      font-size: 18px;
      cursor: pointer;
      background: var(--el-fill-color-light);
      border: 1px solid var(--el-border-color-lighter);
      border-radius: 8px;

      &.active {
        border-color: var(--el-color-primary);
        box-shadow: 0 0 0 2px var(--el-color-primary-light-8);
      }
    }
  }

  .template-contract {
    padding: 12px;
    border-top: 1px solid var(--el-border-color-lighter);

    strong,
    span {
      display: block;
    }

    span {
      margin-top: 6px;
      line-height: 1.5;
      color: var(--el-text-color-secondary);
    }

    dl {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 8px;
      margin: 12px 0 0;
    }

    dt,
    dd {
      margin: 0;
      text-align: center;
    }

    dt {
      color: var(--el-text-color-secondary);
    }

    dd {
      margin-top: 5px;
      font-weight: 700;
    }
  }

  .side-actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
    padding: 12px;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .template-main {
    display: flex;

    :deep(.tekton-dag-designer) {
      flex: 1;
      min-width: 0;
    }
  }

  @media (max-width: 1200px) {
    .template-editor {
      grid-template-columns: 1fr;
    }

    .template-side {
      min-height: 360px;
    }
  }
</style>
