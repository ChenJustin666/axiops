<template>
  <section class="helm-panel">
    <header class="helm-toolbar">
      <div class="helm-identity"
        ><PackageOpen :size="17" /><strong>{{ service.code }}</strong
        ><span>Helm Release · {{ environment.namespace }}</span
        ><ElTag v-if="dirty" size="small" type="warning">未保存</ElTag></div
      >
      <div class="helm-actions">
        <ElButton
          v-if="canEdit"
          size="small"
          :disabled="busy || !revisionId"
          :loading="working"
          @click="preview"
          >预览与保存</ElButton
        >
      </div>
    </header>
    <ElAlert v-if="error" :title="error" type="error" :closable="false" />
    <div class="helm-context">
      <label
        >Chart 版本
        <ElSelect
          :model-value="revisionId"
          size="small"
          :disabled="!canEdit || busy || chartEditing"
          @change="selectRevision"
          ><ElOption
            v-for="rev in revisions"
            :key="rev.id"
            :value="rev.id"
            :label="`#${rev.revision_number}${rev.id === config.revision_id ? ' · 当前环境' : ''}`" /></ElSelect
      ></label>
      <span>{{
        config.bound ? '当前环境独立配置，保存后发布生效' : '配置此环境，保存不会自动部署'
      }}</span>
      <ElButton
        v-if="canEdit && config.bound && environments.length > 1"
        text
        size="small"
        :disabled="busy || dirty"
        @click="openSync"
        >同步到其他环境</ElButton
      >
    </div>
    <ElCollapse v-show="tab === 'chart'" class="helm-advanced"
      ><ElCollapseItem title="Chart 来源与模板更新" name="source">
        <HelmTemplateUpgrade
          v-if="!service.source_ref?.helm_repository && service.source_type !== 'git'"
          class="helm-template-source"
          :service="service"
          :environment="environment"
          :config="config"
          :settings="settings"
          :revision-id="revisionId"
          :can-edit="canEdit && canEditChart"
          :disabled="busy || dirty || chartEditing"
          @busy-change="upgradeWorking = $event"
          @saved="emit('saved')"
        />
        <HelmRepositorySync
          ref="repositorySync"
          :service="service"
          :latest-revision="latestRevision"
          :can-edit="canEditChart"
          :disabled="busy || dirty || chartEditing"
          @busy-change="repositoryWorking = $event"
          @imported="repositoryImported"
        /> </ElCollapseItem
    ></ElCollapse>
    <ElTabs v-model="tab" class="helm-tabs">
      <ElTabPane label="当前环境 values" name="values">
        <div class="editor-heading"
          ><span>{{ environment.name }} · values.yaml</span
          ><small
            >填写副本、CPU、内存、env、路由等 Chart 支持的参数。凭据使用 Secret 引用。</small
          ></div
        >
        <div class="values-tools">
          <HelmValuesSources
            :key="environment.id"
            v-model="settings"
            :service="service"
            :files="files"
            :environments="environments.filter((e) => e.id !== environment.id)"
            :disabled="!canEdit || busy"
            @busy-change="valuesWorking = $event"
          />
          <HelmSharedValuesEditor
            :environment="environment"
            :can-edit="canEdit"
            :disabled="busy || dirty"
            @busy-change="sharedWorking = $event"
            @saved="emit('saved')"
          />
        </div>
        <p class="helm-note"
          >可只填写差异，空对象 {} 使用 Chart 默认值。预览可查看合并后的最终 values。</p
        >
        <YamlEditorPro
          v-model="settings.values"
          height="min(65vh, 720px)"
          :readonly="!canEdit || busy"
          :show-title-bar="false"
          :show-toolbar="false"
          :show-status-bar="false"
          :enable-highlight="false"
          :auto-format="false"
        />
        <HelmCredentialBindings
          v-model="credentialEdits"
          :config="config"
          :project-id="service.project_id"
          :disabled="!canEdit || busy"
        />
        <ElCollapse class="helm-advanced">
          <ElCollapseItem title="镜像映射与继承参数" name="advanced">
            <p
              >构建成功后的镜像按下列 values 路径注入，并验证渲染结果确实引用该镜像。普通
              repository:tag 模板也使用固定 digest。</p
            >
            <ElRadioGroup
              :model-value="imageMode"
              :disabled="!canEdit || busy"
              @change="changeImageMode"
              ><ElRadio value="split">repository + tag</ElRadio
              ><ElRadio value="ref">完整镜像引用</ElRadio></ElRadioGroup
            >
            <div class="image-mapping" v-if="imageMode === 'split'"
              ><ElInput
                v-model="settings.image.repository_path"
                :disabled="!canEdit || busy"
                placeholder="image.repository"
                aria-label="镜像 repository 路径" /><ElInput
                v-model="settings.image.tag_path"
                :disabled="!canEdit || busy"
                placeholder="image.tag"
                aria-label="镜像 tag 路径"
            /></div>
            <ElInput
              v-else
              v-model="settings.image.ref_path"
              :disabled="!canEdit || busy"
              placeholder="image"
              aria-label="完整镜像引用路径"
            />
            <p v-if="config.variables.length"
              >以下为已有项目、服务或环境的路径参数；当前 values
              中的普通值优先，凭据引用单独管理。</p
            >
            <div v-for="v in config.variables" :key="`${v.origin}-${v.key}`" class="inherited-value"
              ><code>{{ v.key }}</code
              ><span>{{ v.secret ? '凭据引用（隐藏）' : v.value }}</span></div
            >
          </ElCollapseItem>
        </ElCollapse>
      </ElTabPane>
      <ElTabPane label="Chart 源码" name="chart">
        <div class="editor-heading"
          ><span>服务 Chart · #{{ selectedRevision?.revision_number || '新版本' }}</span
          ><div
            ><ElButton
              v-if="canEditChart && !chartEditing"
              text
              size="small"
              :disabled="busy"
              @click="chartEditing = true"
              >编辑 Chart</ElButton
            ><template v-if="chartEditing"
              ><ElButton text size="small" :disabled="busy" @click="cancelChart">撤销</ElButton
              ><ElButton text size="small" type="primary" :loading="working" @click="saveChart"
                >保存为自定义版本</ElButton
              ></template
            ></div
          ></div
        >
        <p class="helm-note"
          >Chart 为服务公共版本；保存新版本不会更新其他环境。此环境需选择版本并预览保存。</p
        >
        <div class="chart-layout"
          ><nav aria-label="Chart 文件"
            ><ElTree
              :data="fileTree"
              node-key="path"
              :current-node-key="activePath"
              default-expand-all
              highlight-current
              :expand-on-click-node="true"
              @node-click="(node) => node.file && selectFile(node.path)"
            >
              <template #default="{ data }"
                ><span class="chart-tree-label" :title="data.path"
                  ><component :is="data.file ? FileCode2 : Folder" :size="13" />{{
                    data.label
                  }}</span
                ></template
              > </ElTree
            ><template v-if="chartEditing"
              ><ElInput
                v-model="newFile"
                size="small"
                placeholder="templates/ingress.yaml"
                @keyup.enter="addFile"
              /><ElButton text size="small" @click="addFile">添加文件</ElButton>
              <ElButton
                text
                size="small"
                type="danger"
                :disabled="['Chart.yaml', 'values.yaml'].includes(activePath)"
                @click="removeFile"
                >删除当前文件</ElButton
              ></template
            ></nav
          ><div class="chart-source"
            ><YamlEditorPro
              :model-value="activeContent"
              :validate-on-change="
                !/(^|\/)templates\//.test(activePath) && /\.ya?ml$/.test(activePath)
              "
              :readonly="!chartEditing || busy"
              height="min(65vh, 720px)"
              :show-title-bar="false"
              :show-toolbar="false"
              :show-status-bar="false"
              :enable-highlight="false"
              :auto-format="false"
              @update:model-value="updateFile" /></div
        ></div>
      </ElTabPane>
    </ElTabs>
    <ElDialog
      v-model="previewVisible"
      :title="`配置预览 · ${previewEnvironment?.name || environment.name}`"
      width="min(1120px, 95vw)"
      append-to-body
      :close-on-click-modal="!working"
      :close-on-press-escape="!working"
      :show-close="!working"
    >
      <ElAlert v-if="error" :title="error" type="error" :closable="false" />
      <p class="helm-note"
        >目标：{{ previewEnvironment?.name || environment.name }} ·
        {{ previewEnvironment?.namespace || environment.namespace }}。此操作只保存配置；Secret
        内容已隐藏。镜像为配置校验占位值，发布时由工作流注入选定制品。</p
      >
      <div v-if="syncTarget" class="sync-comparison"
        ><div
          ><strong>目标环境当前 values</strong><pre>{{ syncBefore }}</pre></div
        ><div
          ><strong>将保存的 values</strong><pre>{{ pending?.helm?.values }}</pre>
        </div></div
      >
      <ElTabs v-model="previewTab"
        ><ElTabPane label="最终 values" name="values" /><ElTabPane label="渲染 YAML" name="yaml"
      /></ElTabs>
      <YamlEditorPro
        :model-value="previewTab === 'values' ? finalValues : rendered"
        readonly
        height="min(55vh, 640px)"
        :show-title-bar="false"
        :show-toolbar="false"
        :show-status-bar="false"
        :enable-highlight="false"
        :auto-format="false"
      />
      <ElCheckbox
        v-if="previewEnvironment?.is_production"
        v-model="productionConfirmed"
        :disabled="working"
        >确认保存到生产环境 {{ previewEnvironment.name }}</ElCheckbox
      >
      <template #footer
        ><ElButton :disabled="working" @click="previewVisible = false">返回修改</ElButton
        ><ElButton
          type="primary"
          :loading="working"
          :disabled="!!previewEnvironment?.is_production && !productionConfirmed"
          @click="commit"
          >保存配置</ElButton
        ></template
      >
    </ElDialog>
    <ElDialog v-model="syncVisible" title="同步 Helm 配置" width="min(500px, 92vw)" append-to-body>
      <ElAlert v-if="error" :title="error" type="error" :closable="false" />
      <p class="helm-note"
        >仅覆盖勾选的 values
        配置项，未选中的域名、地址等保持目标环境配置；数组整项替换。保存不会部署，镜像路径映射和凭据引用保持目标环境配置。</p
      >
      <ElSelect v-model="targetId" :disabled="busy" placeholder="选择目标环境" style="width: 100%"
        ><ElOption
          v-for="env in environments.filter((e) => e.id !== environment.id)"
          :key="env.id"
          :value="env.id"
          :label="`${env.name}${env.is_production ? ' · 生产' : ''} · ${env.namespace}`"
      /></ElSelect>
      <ElCheckboxGroup v-model="syncKeys" :disabled="busy" class="sync-keys"
        ><ElCheckbox v-for="key in syncOptions" :key="key" :value="key">{{
          valuesPathLabel(key)
        }}</ElCheckbox></ElCheckboxGroup
      >
      <ElCheckbox v-model="syncChart" :disabled="busy">同时使用当前 Chart 版本</ElCheckbox>
      <template #footer
        ><ElButton @click="syncVisible = false">取消</ElButton
        ><ElButton
          type="primary"
          :disabled="!targetId || (!syncKeys.length && !syncChart)"
          :loading="working"
          @click="previewSync"
          >查看差异与预览</ElButton
        ></template
      >
    </ElDialog>
  </section>
</template>

<script setup lang="ts">
  import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { PackageOpen, FileCode2, Folder } from 'lucide-vue-next'
  import {
    createServiceRevisionApi,
    validateHelmChartApi,
    listServiceRevisionsApi,
    getServiceEnvironmentConfigApi,
    saveServiceEnvironmentConfigApi,
    type AxiOpsEnvironment,
    type AxiOpsService,
    type AxiOpsServiceRevision,
    type HelmEnvironmentSettings,
    type SaveServiceEnvironmentConfig,
    type ServiceEnvironmentConfig
  } from '@/api/axiops'
  import {
    chartFiles,
    environmentHelmSettings,
    initialHelmValues,
    validateHelmSettings,
    syncHelmValues,
    chartFileTree,
    helmValuesPaths,
    valuesPathLabel,
    type ChartFile
  } from '@/utils/helm-service-config'
  import YamlEditorPro from '@/components/yaml-editor-pro/index.vue'
  import HelmTemplateUpgrade from './HelmTemplateUpgrade.vue'
  import HelmCredentialBindings from './HelmCredentialBindings.vue'
  import HelmRepositorySync from './HelmRepositorySync.vue'
  import HelmValuesSources from './HelmValuesSources.vue'
  import HelmSharedValuesEditor from './HelmSharedValuesEditor.vue'
  const props = defineProps<{
    service: AxiOpsService
    environment: AxiOpsEnvironment
    environments: AxiOpsEnvironment[]
    config: ServiceEnvironmentConfig
    canEdit: boolean
    canEditChart: boolean
    activeFile?: string
  }>()
  const emit = defineEmits<{
    saved: []
    'file-change': [file: string]
    'files-change': [files: string[]]
  }>()
  const settings = ref<HelmEnvironmentSettings>(environmentHelmSettings(props.config))
  const imageMode = ref(settings.value.image.ref_path ? 'ref' : 'split')
  const revisions = ref<AxiOpsServiceRevision[]>([]),
    revisionId = ref(props.config.revision_id),
    tab = ref('values'),
    files = ref<ChartFile[]>([]),
    activePath = ref('values.yaml'),
    newFile = ref(''),
    chartEditing = ref(false)
  const upgradeWorking = ref(false)
  const repositoryWorking = ref(false)
  const valuesWorking = ref(false),
    sharedWorking = ref(false)
  const repositorySync = ref<InstanceType<typeof HelmRepositorySync>>()
  const working = ref(false),
    loading = ref(false),
    error = ref(''),
    savedState = ref(''),
    savedFiles = ref('')
  const previewVisible = ref(false),
    previewTab = ref('yaml'),
    finalValues = ref(''),
    rendered = ref(''),
    pending = ref<SaveServiceEnvironmentConfig | null>(null),
    productionConfirmed = ref(false)
  const syncVisible = ref(false),
    targetId = ref<number>(),
    syncTarget = ref<ServiceEnvironmentConfig | null>(null),
    syncBefore = ref('')
  const previewEnvironment = ref<AxiOpsEnvironment>()
  const syncKeys = ref<string[]>([]),
    syncChart = ref(false),
    previewInput = ref('')
  const syncOptions = computed(() => {
    try {
      return helmValuesPaths(settings.value.values, [
        ...Object.values(settings.value.image),
        ...Object.keys(settings.value.credential_keys || {})
      ])
    } catch {
      return []
    }
  })
  const credentialEdits = ref<NonNullable<SaveServiceEnvironmentConfig['helm_credentials']>>({})
  let generation = 0
  const snapshot = () => JSON.stringify([revisionId.value, settings.value, credentialEdits.value])
  const dirty = computed(
    () =>
      (!!savedState.value && snapshot() !== savedState.value) ||
      (chartEditing.value && JSON.stringify(files.value) !== savedFiles.value)
  )
  const busy = computed(
    () =>
      working.value ||
      loading.value ||
      upgradeWorking.value ||
      repositoryWorking.value ||
      valuesWorking.value ||
      sharedWorking.value
  )
  const fileTree = computed(() => chartFileTree(files.value))
  const selectedRevision = computed(() => revisions.value.find((r) => r.id === revisionId.value))
  const latestRevision = computed(() =>
    revisions.value.reduce<AxiOpsServiceRevision | undefined>(
      (latest, item) => (!latest || item.id > latest.id ? item : latest),
      undefined
    )
  )
  const activeContent = computed(
    () => files.value.find((f) => f.path === activePath.value)?.content || ''
  )
  function selectFile(path: string) {
    activePath.value = path
    tab.value = 'chart'
    emit('file-change', path)
  }
  function repositoryImported(revision: AxiOpsServiceRevision) {
    revisions.value = [revision, ...revisions.value.filter((r) => r.id !== revision.id)]
    emit('saved')
  }
  function updateFile(content: string) {
    const file = files.value.find((f) => f.path === activePath.value)
    if (file && chartEditing.value) file.content = content
  }
  function addFile() {
    const path = newFile.value.trim()
    if (
      !/^[A-Za-z0-9_.-]+(?:\/[A-Za-z0-9_.-]+)*$/.test(path) ||
      path.split('/').some((p) => p === '..' || p === '.') ||
      files.value.some((f) => f.path === path)
    )
      return void ElMessage.warning('请输入未使用的 Chart 相对文件路径')
    files.value.push({ path, content: '' })
    selectFile(path)
    newFile.value = ''
  }
  function changeImageMode(mode: string | number | boolean | undefined) {
    imageMode.value = mode === 'ref' ? 'ref' : 'split'
    settings.value.image =
      mode === 'ref'
        ? { ref_path: 'image', repository_path: '', tag_path: '' }
        : { ref_path: '', repository_path: 'image.repository', tag_path: 'image.tag' }
  }
  async function removeFile() {
    if (
      !chartEditing.value ||
      busy.value ||
      ['Chart.yaml', 'values.yaml'].includes(activePath.value)
    )
      return
    try {
      await ElMessageBox.confirm(
        `删除 Chart 文件 ${activePath.value}？保存新版本后生效。`,
        '删除文件',
        { type: 'warning' }
      )
      files.value = files.value.filter((f) => f.path !== activePath.value)
      activePath.value = files.value[0]?.path || ''
    } catch {
      /* cancelled */
    }
  }
  function cancelChart() {
    files.value = JSON.parse(savedFiles.value)
    chartEditing.value = false
  }
  function selectRevision(id: number) {
    if (!props.canEdit || busy.value || chartEditing.value) return
    revisionId.value = id
    files.value = chartFiles(selectedRevision.value?.definition || props.config.definition)
    savedFiles.value = JSON.stringify(files.value)
    if (!files.value.some((f) => f.path === activePath.value))
      activePath.value = files.value[0]?.path || ''
  }
  async function load() {
    const current = ++generation
    loading.value = true
    error.value = ''
    previewVisible.value = false
    syncVisible.value = false
    pending.value = null
    chartEditing.value = false
    files.value = []
    revisions.value = []
    revisionId.value = 0
    savedState.value = ''
    savedFiles.value = ''
    try {
      credentialEdits.value = {}
      settings.value = environmentHelmSettings(props.config)
      imageMode.value = settings.value.image.ref_path ? 'ref' : 'split'
      settings.value.values = initialHelmValues(props.config)
      revisionId.value = props.config.revision_id
      savedState.value = snapshot()
      const list = await listServiceRevisionsApi(props.service.id)
      if (current !== generation) return
      revisions.value = list || []
      files.value = chartFiles(props.config.definition)
      if (!files.value.length) {
        throw Error('尚未导入 Chart，请从代码仓库导入或选择模板')
      }
      savedFiles.value = JSON.stringify(files.value)
      tab.value = props.activeFile === '@environment-values' ? 'values' : tab.value
      activePath.value = files.value.some((f) => f.path === props.activeFile)
        ? props.activeFile!
        : files.value[0]?.path || ''
      if (props.activeFile && files.value.some((f) => f.path === props.activeFile))
        tab.value = 'chart'
    } catch (e: any) {
      if (current === generation) error.value = e?.message || 'Chart 读取失败，请重新加载'
    } finally {
      if (current === generation) loading.value = false
    }
  }
  function makeRequest(config = props.config): SaveServiceEnvironmentConfig {
    validateHelmSettings(settings.value)
    for (const [path, ref] of Object.entries(credentialEdits.value)) {
      if (
        !/^[A-Za-z0-9_-]+(?:\.[A-Za-z0-9_-]+)*$/.test(path) ||
        (ref && (!ref.credential_id || !ref.key)) ||
        path.startsWith('__incomplete') ||
        path.startsWith('__invalid')
      )
        throw Error('请填写有效且不重复的凭据路径、凭据和字段')
    }
    return {
      targets: [{ environment_id: config.environment_id, token: config.token }],
      revision_id: revisionId.value,
      variables: {},
      helm: JSON.parse(JSON.stringify(settings.value)),
      ...(Object.keys(credentialEdits.value).length
        ? { helm_credentials: JSON.parse(JSON.stringify(credentialEdits.value)) }
        : {})
    }
  }
  async function preview() {
    if (!props.canEdit || busy.value || !revisionId.value || chartEditing.value) return
    const current = generation
    error.value = ''
    working.value = true
    pending.value = null
    syncTarget.value = null
    previewEnvironment.value = props.environment
    productionConfirmed.value = false
    try {
      const request = makeRequest()
      const result = await saveServiceEnvironmentConfigApi(props.service.id, request, true)
      if (current !== generation) return
      rendered.value = result.configs[0].manifest || ''
      finalValues.value = result.configs[0].final_values || ''
      if (!rendered.value) throw Error('预览没有返回渲染结果，请更新 CI/CD API')
      pending.value = request
      previewInput.value = snapshot()
      previewVisible.value = true
    } catch (e: any) {
      if (current === generation) error.value = e?.message || 'Chart/values 校验失败'
    } finally {
      if (current === generation) working.value = false
    }
  }
  async function commit() {
    if (
      !pending.value ||
      !props.canEdit ||
      busy.value ||
      (previewEnvironment.value?.is_production && !productionConfirmed.value)
    )
      return
    const current = generation
    const request = JSON.parse(JSON.stringify(pending.value))
    working.value = true
    try {
      if (
        previewInput.value !== snapshot() ||
        (!syncTarget.value && JSON.stringify(makeRequest()) !== JSON.stringify(request))
      )
        throw Error('配置已变化，请重新预览')
      const result = await saveServiceEnvironmentConfigApi(props.service.id, request)
      if (current !== generation) return
      if (
        !result.saved ||
        !result.validated ||
        !result.configs?.some(
          (c) =>
            c.service_id === props.service.id &&
            c.environment_id === request.targets[0].environment_id
        )
      )
        throw Error('保存结果未确认，请重新加载查看')
      previewVisible.value = false
      ElMessage.success('环境配置已保存，发布后生效')
      savedState.value = snapshot()
      emit('saved')
    } catch (e: any) {
      if (current === generation) error.value = e?.message || '保存失败，请重新加载核对'
    } finally {
      if (current === generation) working.value = false
    }
  }
  async function saveChart() {
    if (!props.canEditChart || busy.value || !chartEditing.value) return
    const current = generation
    working.value = true
    error.value = ''
    try {
      const { definition } = await validateHelmChartApi(props.service.id, files.value)
      if (current !== generation) return
      const latest = Math.max(
        props.service.current_revision_id || 0,
        ...revisions.value.map((r) => r.id)
      )
      const result = await createServiceRevisionApi(props.service.id, {
        source_commit: 'manual-chart-edit',
        expected_revision_id: latest,
        definition
      })
      if (current !== generation) return
      revisions.value = [result, ...revisions.value]
      revisionId.value = result.id
      chartEditing.value = false
      savedFiles.value = JSON.stringify(files.value)
      ElMessage.success('Chart 新版本已保存，请预览保存当前环境配置')
    } catch (e: any) {
      if (current === generation) error.value = e?.message || 'Chart 校验或保存失败'
    } finally {
      if (current === generation) working.value = false
    }
  }
  function openSync() {
    targetId.value = undefined
    syncKeys.value = []
    syncChart.value = false
    syncVisible.value = true
  }
  async function previewSync() {
    if (
      !props.canEdit ||
      !targetId.value ||
      busy.value ||
      dirty.value ||
      (!syncKeys.value.length && !syncChart.value)
    )
      return
    const current = generation
    working.value = true
    error.value = ''
    pending.value = null
    productionConfirmed.value = false
    try {
      const target = await getServiceEnvironmentConfigApi(targetId.value, props.service.id)
      if (current !== generation) return
      if (target.environment_id !== targetId.value || target.service_id !== props.service.id)
        throw Error('目标环境配置不匹配')
      const request = makeRequest(target)
      request.revision_id = syncChart.value ? revisionId.value : target.revision_id
      if (!request.revision_id)
        throw Error('目标环境尚未选择 Chart，请勾选“同时使用当前 Chart 版本”')
      const targetSettings = environmentHelmSettings(target)
      delete request.helm_credentials
      request.helm = {
        credential_keys: targetSettings.credential_keys,
        source: targetSettings.source,
        image: targetSettings.image,
        values: syncHelmValues(settings.value.values, targetSettings.values, syncKeys.value)
      }
      request.source = { environment_id: props.config.environment_id, token: props.config.token }
      const result = await saveServiceEnvironmentConfigApi(props.service.id, request, true)
      if (current !== generation) return
      syncTarget.value = target
      syncBefore.value = environmentHelmSettings(target).values
      previewEnvironment.value = props.environments.find((e) => e.id === target.environment_id)
      pending.value = request
      previewInput.value = snapshot()
      rendered.value = result.configs[0].manifest || ''
      finalValues.value = result.configs[0].final_values || ''
      if (!rendered.value) throw Error('目标环境预览为空')
      syncVisible.value = false
      previewVisible.value = true
    } catch (e: any) {
      if (current === generation) error.value = e?.message || '同步预览失败'
    } finally {
      if (current === generation) working.value = false
    }
  }
  async function confirmLeave() {
    if (busy.value) return false
    if (!dirty.value) return true
    try {
      await ElMessageBox.confirm('有未保存的 Helm 配置，离开将丢弃修改。', '未保存的修改', {
        confirmButtonText: '放弃修改',
        cancelButtonText: '继续编辑',
        type: 'warning'
      })
      return true
    } catch {
      return false
    }
  }
  async function openRepositorySync() {
    if (
      !props.canEditChart ||
      working.value ||
      upgradeWorking.value ||
      repositoryWorking.value ||
      dirty.value ||
      chartEditing.value
    )
      return
    if (!props.service.code_source_id) {
      ElMessage.warning('请先在服务设置中关联代码仓库，再导入 Chart')
      return
    }
    const serviceId = props.service.id
    await load()
    tab.value = 'chart'
    await nextTick()
    if (serviceId === props.service.id && !busy.value) repositorySync.value?.open()
  }
  watch(() => [props.service.id, props.config.token], load, { immediate: true })
  watch(tab, (value) =>
    emit('file-change', value === 'values' ? '@environment-values' : activePath.value)
  )
  watch(
    () => files.value.map((file) => file.path),
    (value) => emit('files-change', value)
  )
  watch(
    () => props.activeFile,
    (path) => {
      if (path === '@environment-values') {
        tab.value = 'values'
        return
      }
      if (path && files.value.some((f) => f.path === path)) {
        activePath.value = path
        tab.value = 'chart'
      }
    }
  )
  onBeforeUnmount(() => generation++)
  defineExpose({ dirty, busy, confirmLeave, openRepositorySync })
</script>

<style scoped lang="scss">
  .helm-panel {
    min-width: 0;
    border: 1px solid var(--el-border-color-light);
    border-radius: 10px;
    overflow: hidden;
  }
  .values-tools {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 12px;
  }
  .helm-toolbar,
  .helm-context,
  .editor-heading,
  .helm-identity,
  .helm-actions {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }
  .helm-toolbar {
    justify-content: space-between;
    padding: 14px 18px;
    background: var(--el-fill-color-lighter);
  }
  .helm-identity span,
  .helm-context > span,
  .editor-heading small,
  .helm-note {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.7;
  }
  .helm-context {
    padding: 12px 18px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  .helm-context label {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
  }
  .helm-template-source {
    padding: 0 18px 12px;
  }
  .helm-context .el-select {
    width: 180px;
  }
  .helm-tabs {
    padding: 0 18px 16px;
  }
  .editor-heading {
    justify-content: space-between;
    margin-bottom: 12px;
    font-size: 13px;
  }
  .helm-advanced {
    margin-top: 16px;
  }
  .helm-advanced p {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
  .image-mapping {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
    margin: 12px 0;
  }
  .chart-layout {
    display: grid;
    grid-template-columns: 210px minmax(0, 1fr);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    overflow: hidden;
  }
  .chart-layout nav {
    padding: 8px;
    border-right: 1px solid var(--el-border-color-lighter);
  }
  .chart-tree-label {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
    font-size: 12px;
  }
  .chart-layout nav button {
    display: block;
    width: 100%;
    padding: 9px 8px;
    border: 0;
    border-radius: 4px;
    text-align: left;
    background: transparent;
    color: var(--el-text-color-regular);
    font-size: 12px;
    overflow-wrap: anywhere;
    cursor: pointer;
  }
  .chart-layout nav button.active {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
  .chart-source {
    min-width: 0;
  }
  .inherited-value {
    display: flex;
    gap: 12px;
    padding: 6px 0;
    font-size: 12px;
  }
  .sync-comparison {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
    margin: 12px 0;
  }
  .sync-comparison pre {
    max-height: 200px;
    overflow: auto;
    font-size: 12px;
    background: var(--el-fill-color-lighter);
    padding: 12px;
  }
  @media (max-width: 760px) {
    .helm-toolbar,
    .helm-context {
      padding: 12px;
    }
    .helm-tabs {
      padding: 0 10px 10px;
    }
    .chart-layout {
      grid-template-columns: 1fr;
    }
    .chart-layout nav {
      border-right: 0;
      display: flex;
      flex-wrap: wrap;
      gap: 6px;
    }
    .chart-layout nav button {
      width: auto;
    }
    .image-mapping,
    .sync-comparison {
      grid-template-columns: 1fr;
    }
    .helm-identity span {
      width: 100%;
    }
  }
</style>
