<template>
  <section v-if="service" class="service-workbench">
    <ElAlert v-if="loadError" :title="loadError" type="error" :closable="false"
      ><ElButton text @click="loadRevisions">Reload</ElButton></ElAlert
    >
    <ElAlert
      v-if="service.delivery_type === 'helm'"
      title="Helm values are read-only"
      type="info"
      :closable="false"
    />
    <div
      class="workbench-grid"
      :style="{
        gridTemplateColumns: `${editorWidth}% minmax(320px, 1fr)`,
        '--editor-width': `${editorWidth}%`
      }"
    >
      <main class="definition-stage">
        <div class="stage-toolbar">
          <div class="stage-title"
            ><FileCode2 :size="14" />{{ service.code }} · YAML definition</div
          >
          <div class="stage-toolbar-actions">
            <ElTooltip content="格式化 YAML（Shift+Alt+F）" placement="bottom">
              <ElButton
                circle
                text
                size="small"
                :icon="AlignLeft"
                aria-label="格式化 YAML"
                :disabled="editorReadonly || !editorContent"
                @click="yamlEditor?.formatYaml()"
              />
            </ElTooltip>
            <ElButton
              v-if="selectedRevision || (environmentMode && environmentConfig)"
              text
              size="small"
              @click="previewMode = !previewMode"
              >{{ previewMode ? '编辑 YAML' : '预览渲染结果' }}</ElButton
            >
          </div>
        </div>
        <div
          v-if="environmentMode && environmentConfig && !environmentConfig.bound"
          class="environment-binding-notice"
        >
          <div>
            <strong>此环境尚未配置服务</strong>
            <span
              >手动方式可直接粘贴原生 YAML；模板方式可选择模板或参考已有环境，保存后再部署。</span
            >
          </div>
          <ElButton
            v-if="canEdit && service.source_type !== 'manual'"
            size="small"
            type="primary"
            @click="emit('join-environment')"
          >
            配置此环境
          </ElButton>
        </div>
        <div class="definition-panel">
          <div v-if="serviceFiles.length" class="definition-file-index" aria-label="YAML files">
            <span class="definition-file-label">文件</span>
            <button
              v-for="file in serviceFiles"
              :key="file"
              type="button"
              class="definition-file-chip"
              :class="{ 'is-active': file === activeFileName }"
              @click="focusFile(file)"
            >
              {{ file }}
            </button>
          </div>
          <div
            v-if="(environmentMode && environmentConfig) || selectedRevision || editorContent"
            class="editor-frame"
            :class="{ 'is-locked': editorReadonly }"
          >
            <YamlEditorPro
              ref="yamlEditor"
              v-model="editorDisplayContent"
              :filename="definitionFilename"
              :readonly="editorReadonly"
              theme="light"
              :show-title-bar="false"
              :show-toolbar="false"
              :show-line-numbers="true"
              :show-status-bar="false"
              :enable-highlight="false"
              :auto-format="false"
              :placeholder="definitionPlaceholder"
              @save="saveEditorShortcut"
            />
            <button
              v-if="editorReadonly && canUnlockEditor"
              type="button"
              class="editor-unlock-button"
              title="解锁编辑"
              aria-label="解锁编辑"
              @click="unlockEditor"
            >
              <Unlock :size="14" />
            </button>
          </div>
          <div
            v-if="editorReadonly && !canUnlockEditor && !loading && !loadError"
            class="editor-readonly-hint"
          >
            <Lock :size="14" />
            <span>{{ editorReadonlyReason }}</span>
          </div>
          <div
            v-if="!((environmentMode && environmentConfig) || selectedRevision || editorContent)"
            class="empty-definition"
            ><FileCode2 :size="30" /><strong>服务尚未关联定义</strong
            ><p>可从代码库、模板或手动编辑创建 YAML 定义。</p>
            <ElButton v-if="canImport" type="primary" :icon="RefreshCw" @click="emit('import')"
              >同步服务定义</ElButton
            >
          </div>
        </div>
        <div
          v-if="
            !loadError &&
            (editing || selectedRevision || editorContent) &&
            service.delivery_type !== 'host'
          "
          class="save-bar"
        >
          <span class="save-state" :class="{ 'is-dirty': dirty }">{{
            dirty ? '有未保存的修改' : '内容已保存'
          }}</span>
          <div class="save-actions">
            <ElButton
              v-if="service.delivery_type !== 'helm'"
              size="small"
              plain
              :icon="CheckCircle2"
              :disabled="loading || saving || !!loadError || !editorContent.trim()"
              @click="validateVariables(false, !environmentMode)"
              >语法校验</ElButton
            >
            <ElButton
              v-if="editing && service.delivery_type !== 'helm' && !environmentMode"
              size="small"
              :disabled="loading || saving || !dirty"
              @click="toggleEditing"
              >撤销修改</ElButton
            >
            <ElButton
              v-if="editing && service.delivery_type !== 'helm'"
              size="small"
              type="primary"
              :icon="Save"
              :loading="saving"
              :disabled="!dirty"
              @click="environmentMode ? saveEnvironment() : saveRevision()"
              >{{ environmentMode ? '保存环境配置' : '保存配置' }}</ElButton
            >
            <ElButton
              v-if="environments.length && !environmentMode"
              size="small"
              plain
              :icon="Rocket"
              :disabled="loading || saving || !selectedRevision"
              @click="emit('environment-config')"
              >加入环境</ElButton
            >
            <ElButton
              v-if="environmentMode && environments.length > 1"
              size="small"
              plain
              :icon="Rocket"
              :disabled="loading || saving || !environmentConfig?.bound"
              @click="emit('environment-config')"
              >同步配置到其他环境</ElButton
            >
          </div>
        </div>
      </main>

      <button
        type="button"
        class="workbench-resizer"
        aria-label="调整编辑区宽度"
        title="拖动调整编辑区宽度"
        @mousedown="startResize"
      />

      <aside class="variables-rail">
        <div class="rail-heading">
          <div
            ><span class="section-eyebrow">SERVICE VARIABLES</span><strong>组件与变量</strong></div
          >
          <ElButton
            v-if="service.delivery_type !== 'helm' && usesTemplateParameters"
            text
            size="small"
            :icon="Plus"
            :disabled="editorReadonly"
            @click="addVariable"
            >添加变量</ElButton
          >
        </div>
        <div class="variables-scroll">
          <section class="variable-section">
            <div class="variable-section-title"
              ><strong>服务组件</strong><small>服务与镜像构建配置</small></div
            >
            <div class="component-row">
              <span class="component-kind"><FileCode2 :size="14" /></span>
              <span class="component-copy"
                ><strong>{{ service.name }}</strong
                ><small>{{ service.image_name || '构建产物镜像' }}</small></span
              >
              <ElButton v-if="canViewBuild" text size="small" @click="emit('build-config')"
                >构建配置</ElButton
              >
            </div>
          </section>
          <section class="variable-section">
            <div class="variable-section-title"
              ><strong>系统变量</strong><small>由项目、服务和环境注入</small></div
            >
            <dl class="system-variable-list">
              <div
                ><dt>$Product$</dt><dd>{{ projectName || '当前项目' }}</dd></div
              >
              <div
                ><dt>$Service$</dt><dd>{{ service.code }}</dd></div
              >
              <div><dt v-text="'{{.Image}}'"></dt><dd>构建或晋级的镜像</dd></div>
              <div><dt>$Namespace$</dt><dd>按环境覆盖</dd></div>
              <div><dt>$EnvName$</dt><dd>按环境注入</dd></div>
              <div><dt>$ClusterName$</dt><dd>按集群注入</dd></div>
              <div
                ><dt>${{ service.code }}-image$</dt
                ><dd>{{ service.image_name || '构建产物' }}</dd></div
              >
            </dl>
            <small
              >手动 YAML 同样支持
              <span v-text="'{{.Service}}、{{.Image}}、{{.Namespace}}'"></span
              >，也可直接填写原生值。</small
            >
          </section>
          <section
            v-if="service.delivery_type !== 'helm' && usesTemplateParameters"
            class="variable-section"
          >
            <div class="variable-section-title"
              ><strong>运行参数</strong><small>默认值可编辑，环境可覆盖</small></div
            >
            <div v-for="group in variableGroups" :key="group.key" class="parameter-group">
              <div class="parameter-group-heading"
                ><strong>{{ group.label }}</strong
                ><small>{{ group.hint }}</small></div
              >
              <div v-if="group.key === 'probe'" class="probe-summary">
                <ElSelect
                  :model-value="findVariableValue('healthCheckType') || 'none'"
                  :disabled="editorReadonly"
                  aria-label="健康检查方式"
                  @update:model-value="setHealthCheckMode"
                >
                  <ElOption value="none" label="不生成探针" />
                  <ElOption value="tcp" label="TCP · 检查端口" />
                  <ElOption value="http" label="HTTP · 检查健康接口" />
                </ElSelect>
                <ElCheckbox
                  v-if="(findVariableValue('healthCheckType') || 'none') !== 'none'"
                  :disabled="editorReadonly"
                  :model-value="
                    !findVariableValue('healthPort') || findVariableValue('healthPort') === '0'
                  "
                  @change="setProbePortFollowService(!!$event)"
                  >跟随应用端口</ElCheckbox
                >
              </div>
              <div
                v-if="
                  group.key === 'probe' &&
                  (findVariableValue('healthCheckType') || 'none') !== 'none'
                "
                class="probe-options"
              >
                <ElCheckbox
                  v-for="check in probeChecks"
                  :key="check.key"
                  :model-value="findVariableValue(check.key) !== 'false'"
                  :disabled="editorReadonly"
                  @change="setVariableValue(check.key, String(!!$event))"
                  >{{ check.label }}</ElCheckbox
                >
              </div>
              <p v-if="group.key === 'probe'" class="probe-help">
                就绪失败停止接流量；启动、存活持续失败会重启。TCP 无需路径，HTTP 使用应用实际接口。
                <span v-if="hasNativeProbes"
                  >当前 YAML 已有原生探针，以 YAML 为准；关闭或调整这些探针请直接修改 YAML。</span
                >
                <span v-else-if="(findVariableValue('healthCheckType') || 'none') === 'none'"
                  >未启用应用探针；发布仍检查 Kubernetes 资源就绪状态。</span
                >
              </p>
              <table
                v-if="group.key !== 'probe' || visibleGroupItems(group).length"
                class="variable-table"
                ><thead
                  ><tr><th>参数</th><th>当前值</th><th>操作</th></tr></thead
                >
                <tbody
                  ><tr v-for="variable in visibleGroupItems(group)" :key="variable.id">
                    <td
                      ><ElInput
                        v-if="group.key === 'custom' && variable.editing"
                        v-model="variable.key"
                        size="small"
                        :disabled="editorReadonly"
                        placeholder="变量名"
                      /><span v-else class="parameter-label">{{ variable.label }}</span
                      ><small>{{ variable.description }}</small></td
                    >
                    <td
                      ><ElInput
                        v-if="variable.editing"
                        v-model="variable.value"
                        size="small"
                        :disabled="editorReadonly"
                      /><span v-else class="variable-value">{{
                        variable.value || '未设置'
                      }}</span></td
                    >
                    <td class="variable-actions"
                      ><ElButton
                        text
                        size="small"
                        :icon="Edit3"
                        title="编辑"
                        :disabled="editorReadonly"
                        @click="variable.editing = !variable.editing"
                    /></td> </tr
                ></tbody>
              </table>
            </div>
            <p v-if="unsetVariables.length" class="variable-warning"
              >未保存默认值：{{ unsetVariables.join(', ') }}，请填写并保存后再执行。</p
            >
          </section>
        </div>
      </aside>
    </div>
    <ElDialog
      v-model="newFileVisible"
      title="鏂板缓 YAML 鏂囦欢"
      width="680px"
      append-to-body
      :close-on-click-modal="false"
    >
      <p>Paste a base YAML file to create a template.</p>
      <ElForm label-position="top">
        <ElFormItem label="File name">
          <ElInput v-model="newFileName" placeholder="e.g. configmap.yaml" />
        </ElFormItem>
        <ElFormItem label="YAML content">
          <ElInput
            v-model="newFileContent"
            type="textarea"
            :rows="14"
            spellcheck="false"
            placeholder="Kubernetes YAML"
        /></ElFormItem>
      </ElForm>
      <p>Secret values must use encrypted references.</p>
      <template #footer>
        <ElButton @click="newFileVisible = false">Cancel</ElButton>
        <ElButton type="primary" @click="createManualFile">Add file</ElButton>
      </template>
    </ElDialog>
  </section>
  <section v-else-if="false" class="no-service-state"
    ><Package :size="30" /><strong>Select a service</strong> ><p
      >This area shows YAML, variables and deployment controls.</p
    ></section
  >
</template>

<script setup lang="ts">
  import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue'
  import { ElMessage } from 'element-plus'
  import * as yaml from 'js-yaml'
  import {
    AlignLeft,
    CheckCircle2,
    Edit3,
    FileCode2,
    Package,
    Plus,
    RefreshCw,
    Rocket,
    Save,
    Lock,
    Unlock
  } from 'lucide-vue-next'
  import {
    createServiceRevisionApi,
    listServiceRevisionsApi,
    type AxiOpsCluster,
    type AxiOpsEnvironment,
    type AxiOpsService,
    type AxiOpsServiceRevision,
    type ServiceEnvironmentConfig
  } from '@/api/axiops'
  import YamlEditorPro from '@/components/yaml-editor-pro/index.vue'

  const props = withDefaults(
    defineProps<{
      service: AxiOpsService | null
      projectName?: string
      activeFile?: string
      environments: AxiOpsEnvironment[]
      clusters?: AxiOpsCluster[]
      canEdit: boolean
      canImport?: boolean
      canViewBuild: boolean
      environmentMode?: boolean
      environmentConfig?: ServiceEnvironmentConfig | null
    }>(),
    { projectName: '', activeFile: '', clusters: () => [], canImport: false }
  )
  const emit = defineEmits<{
    edit: []
    import: []
    'template-import': []
    'file-change': [file: string]
    'build-config': []
    'environment-config': []
    saved: []
    'environment-save': [payload: { definition: string; variables: Record<string, string> }]
    'join-environment': []
  }>()
  const loading = ref(false)
  const loadError = ref('')
  let revisionRequest = 0
  const savedEditorState = ref('')
  const editorState = () =>
    JSON.stringify([
      editorContent.value,
      customVariables.value.map(({ key, value }) => ({ key, value }))
    ])
  const dirty = computed(() => !!savedEditorState.value && editorState() !== savedEditorState.value)
  const saving = ref(false)
  const revisions = ref<AxiOpsServiceRevision[]>([])
  const selectedRevisionId = ref(0)
  const editorContent = ref('')
  const usesTemplateParameters = computed(
    () =>
      props.service?.source_type !== 'manual' || referencedVariables(editorContent.value).length > 0
  )
  const yamlEditor = ref<{
    gotoLine: (line: number, options?: { focus?: boolean }) => void
    formatYaml: () => Promise<void>
  }>()
  const serviceFiles = ref<string[]>(['deployment.yaml'])
  const fileContents = reactive<Record<string, string>>({})
  const activeFileName = ref('deployment.yaml')
  const newFileVisible = ref(false),
    newFileName = ref(''),
    newFileContent = ref('')
  const editing = ref(false)
  const previewMode = ref(false)
  const canUnlockEditor = computed(
    () =>
      !loading.value &&
      !saving.value &&
      !loadError.value &&
      !!props.canEdit &&
      !previewMode.value &&
      props.service?.delivery_type !== 'helm' &&
      !editing.value
  )
  const editorReadonly = computed(
    () =>
      !props.canEdit ||
      loading.value ||
      saving.value ||
      !!loadError.value ||
      previewMode.value ||
      !editing.value ||
      props.service?.delivery_type === 'helm'
  )
  const editorReadonlyReason = computed(() => {
    if (props.service?.delivery_type === 'helm') return 'Helm 服务使用 values 配置，当前为只读。'
    if (previewMode.value) return '当前为预览模式，点击“编辑 YAML”后可修改。'
    if (!props.canEdit) return '当前账号没有该服务的编辑权限。'
    if (!editing.value) return '当前为只读查看状态，点击“解锁编辑”后可修改。'
    return ''
  })
  const editorWidth = ref(60)
  const resizing = ref(false)
  const customVariables = ref<
    Array<{
      id: number
      key: string
      value: string
      editing?: boolean
      label?: string
      description?: string
    }>
  >([])
  const initialEnvironmentVariables = ref<Record<string, string>>({})
  const variableGroups = computed(() => {
    const find = (key: string) => customVariables.value.find((item) => item.key === key)
    const labels: Record<string, [string, string]> = {
      replicas: ['Replicas', '运行数量'],
      port: ['应用端口', '服务端口'],
      cpuRequest: ['CPU 请求', '最低 CPU'],
      memoryRequest: ['内存请求', '最低内存'],
      cpuLimit: ['CPU 上限', '最高 CPU'],
      memoryLimit: ['内存上限', '最高内存'],
      healthCheckType: ['探针类型', '选择 none、http 或 tcp'],
      healthPath: ['HTTP 路径', '例如 /health'],
      healthPort: ['探针端口', '0 表示使用服务端口'],
      probeReadinessEnabled: ['就绪检查', '失败停止接流量，不重启'],
      probeStartupEnabled: ['启动检查', '持续失败会重启'],
      probeLivenessEnabled: ['存活检查', '持续失败会重启'],
      probePeriodSeconds: ['检查间隔', '每隔多少秒检查一次'],
      probeTimeoutSeconds: ['检查超时', '单次检查最长等待时间'],
      probeFailureThreshold: ['连续失败次数', '达到次数后标记异常'],
      startupFailureThreshold: ['启动失败次数', '启动阶段允许失败的次数']
    }
    const group = (key: string, label: string, hint: string, keys: string[]) => ({
      key,
      label,
      hint,
      items: keys
        .map((name) => {
          const item = find(name)
          if (!item) return null
          // Keep the original reactive variable object so editing the grouped
          // view updates the payload that is saved with the service revision.
          item.label = labels[name][0]
          item.description = labels[name][1]
          return item
        })
        .filter(Boolean)
    })
    const groups = [
      group('resources', '资源配置', '控制容器 CPU、内存和副本数', [
        'replicas',
        'port',
        'cpuRequest',
        'memoryRequest',
        'cpuLimit',
        'memoryLimit'
      ]),
      group('probe', '健康检查', '按需生成；原生 YAML 探针优先', [
        'healthCheckType',
        'healthPath',
        'healthPort',
        'probeReadinessEnabled',
        'probeStartupEnabled',
        'probeLivenessEnabled',
        'probePeriodSeconds',
        'probeTimeoutSeconds',
        'probeFailureThreshold',
        'startupFailureThreshold'
      ])
    ]
    const groupedKeys = new Set(groups.flatMap((entry) => entry.items.map((item) => item!.key)))
    const custom = customVariables.value.filter((item) => !groupedKeys.has(item.key))
    if (custom.length)
      groups.push({
        key: 'custom',
        label: '应用配置',
        hint: '当前服务的配置变量',
        items: custom.map((item) => {
          item.label = item.key
          return item
        })
      })
    return groups
  })
  const findVariableValue = (key: string) =>
    customVariables.value.find((item) => item.key === key)?.value || ''
  const setVariableValue = (key: string, value: string) => {
    const item = customVariables.value.find((entry) => entry.key === key)
    if (item) item.value = value
    else customVariables.value.push({ id: Date.now() + customVariables.value.length, key, value })
  }
  const probeChecks = [
    { key: 'probeReadinessEnabled', label: '就绪检查' },
    { key: 'probeStartupEnabled', label: '启动检查' },
    { key: 'probeLivenessEnabled', label: '存活检查（失败重启）' }
  ]
  const hasNativeProbes = computed(() =>
    /^\s+(?:readiness|liveness|startup)Probe\s*:/m.test(editorContent.value)
  )
  const setHealthCheckMode = (value: string) => {
    if (
      value !== 'none' &&
      (!findVariableValue('healthCheckType') || findVariableValue('healthCheckType') === 'none')
    ) {
      for (const { key } of probeChecks) {
        if (!findVariableValue(key))
          setVariableValue(key, key === 'probeLivenessEnabled' ? 'false' : 'true')
      }
    }
    setVariableValue('healthCheckType', value)
  }
  const setProbePortFollowService = (follow: boolean) => {
    setVariableValue('healthPort', follow ? '0' : findVariableValue('port') || '8080')
  }
  const visibleGroupItems = (group: { key: string; items: any[] }) =>
    group.items.filter((item) => {
      if (group.key !== 'probe') return true
      const mode = findVariableValue('healthCheckType') || 'none'
      if (
        mode === 'none' ||
        item.key === 'healthCheckType' ||
        probeChecks.some((check) => check.key === item.key)
      )
        return false
      if (item.key === 'healthPath' && mode !== 'http') return false
      if (item.key === 'healthPort' && findVariableValue('healthPort') === '0') return false
      return (
        item.key !== 'startupFailureThreshold' ||
        findVariableValue('probeStartupEnabled') !== 'false'
      )
    })
  const systemVariableKeys = computed(() => [
    'Product',
    'Service',
    'Namespace',
    'EnvName',
    'ClusterName',
    'Image',
    `${props.service?.code || 'service'}-image`
  ])
  const referencedVariables = (definition: string) => [
    ...new Set(
      [
        ...definition.matchAll(
          /\{\{\s*\.?([A-Za-z][A-Za-z0-9_]*)\s*\}\}|\$([A-Za-z][A-Za-z0-9_-]*)\$/g
        )
      ]
        .map((match) => match[1] || match[2])
        .filter((key) => !systemVariableKeys.value.includes(key))
    )
  ]
  const unsetVariables = computed(() =>
    referencedVariables(editorContent.value).filter(
      (key) => !customVariables.value.find((item) => item.key.trim() === key)?.value.trim()
    )
  )
  const selectedRevision = computed(() =>
    revisions.value.find((item) => item.id === selectedRevisionId.value)
  )
  const definitionFilename = computed(() => `${props.service?.code || 'service'}/definition.yaml`)
  const definitionPlaceholder = computed(() =>
    props.service?.delivery_type === 'helm'
      ? '# Helm values.yaml\nreplicaCount: 1\nimage:\n  repository: registry.example.com/team/service\n  tag: latest'
      : '# Kubernetes YAML\napiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: service'
  )
  const variableMap = computed(() => ({
    ...Object.fromEntries(customVariables.value.map((item) => [item.key.trim(), item.value])),
    Product: props.projectName || '$Product$',
    Service: props.service?.code || '$Service$',
    Image: props.service?.image_name || '$Image$',
    [`${props.service?.code || 'service'}-image`]: props.service?.image_name || '$Image$'
  }))
  const renderVariables = (value: string) =>
    value.replace(
      /\{\{\s*\.?([A-Za-z][A-Za-z0-9_]*)\s*\}\}|\$([A-Za-z][A-Za-z0-9_-]*)\$/g,
      (match, dotKey, dollarKey) => variableMap.value[dotKey || dollarKey] ?? match
    )
  const renderPreviewProbes = (definition: string) => {
    if (props.service?.delivery_type === 'helm') return definition
    const mode = findVariableValue('healthCheckType').trim().toLowerCase()
    if (!mode || mode === 'none' || !/^(http|tcp)$/.test(mode)) return definition
    if (/\{\{\s*\.?[A-Za-z][A-Za-z0-9_]*\s*\}\}|\$[A-Za-z][A-Za-z0-9_-]*\$/g.test(definition))
      return definition

    const configuredPort = Number(findVariableValue('healthPort') || 0)
    const port = configuredPort > 0 ? configuredPort : Number(findVariableValue('port') || 8080)
    const period = Number(findVariableValue('probePeriodSeconds') || 10)
    const timeout = Number(findVariableValue('probeTimeoutSeconds') || 3)
    const failureThreshold = Number(findVariableValue('probeFailureThreshold') || 3)
    const startupFailureThreshold = Number(findVariableValue('startupFailureThreshold') || 30)
    if (![port, period, timeout, failureThreshold, startupFailureThreshold].every(Number.isFinite))
      return definition

    try {
      const documents: unknown[] = []
      yaml.loadAll(definition, (document) => documents.push(document))
      const rendered = documents.map((document: any) => {
        if (!document || typeof document !== 'object') return document
        const paths = [
          ['spec', 'template', 'spec', 'containers'],
          ['spec', 'jobTemplate', 'spec', 'template', 'spec', 'containers']
        ]
        for (const path of paths) {
          let parent: any = document
          for (const key of path.slice(0, -1)) parent = parent?.[key]
          const containers = parent?.[path[path.length - 1]]
          if (!Array.isArray(containers)) continue
          const check =
            mode === 'http'
              ? { httpGet: { path: findVariableValue('healthPath') || '/health', port } }
              : { tcpSocket: { port } }
          const common = { periodSeconds: period, timeoutSeconds: timeout, failureThreshold }
          containers.forEach((container: any) => {
            if (!container || typeof container !== 'object') return
            if (findVariableValue('probeStartupEnabled') !== 'false')
              container.startupProbe ??= {
                ...check,
                periodSeconds: period,
                timeoutSeconds: timeout,
                failureThreshold: startupFailureThreshold
              }
            if (findVariableValue('probeReadinessEnabled') !== 'false')
              container.readinessProbe ??= { ...check, ...common }
            if (findVariableValue('probeLivenessEnabled') !== 'false')
              container.livenessProbe ??= { ...check, ...common }
          })
        }
        return document
      })
      return rendered
        .map((document) =>
          document == null ? '' : yaml.dump(document, { noRefs: true }).trimEnd()
        )
        .filter(Boolean)
        .join('\n---\n')
    } catch {
      return definition
    }
  }
  const editorDisplayContent = computed({
    get: () => {
      if (!previewMode.value) return editorContent.value
      return renderPreviewProbes(renderVariables(editorContent.value))
    },
    set: (value: string) => {
      if (!previewMode.value) editorContent.value = value
    }
  })
  const parseFiles = (definition: string) => {
    Object.keys(fileContents).forEach((key) => delete fileContents[key])
    if (props.service?.delivery_type === 'helm') {
      const chart = JSON.parse(definition) as { files: Array<{ path: string; content: string }> }
      serviceFiles.value = chart.files.map((file) => file.path)
      chart.files.forEach((file) => {
        fileContents[file.path] = file.content
      })
      activeFileName.value =
        serviceFiles.value.find((file) => file.endsWith('values.yaml')) || serviceFiles.value[0]
      return
    }
    const matches = [...definition.matchAll(/^# AxiOps-File: (.+)$/gm)]
    if (!matches.length) {
      serviceFiles.value = ['deployment.yaml']
      fileContents[serviceFiles.value[0]] = definition
      activeFileName.value = serviceFiles.value[0]
      return
    }
    serviceFiles.value = matches.map((match) => match[1].trim())
    Object.keys(fileContents).forEach((key) => delete fileContents[key])
    matches.forEach((match, index) => {
      const start = (match.index || 0) + match[0].length
      const end =
        index + 1 < matches.length
          ? matches[index + 1].index || definition.length
          : definition.length
      fileContents[match[1].trim()] = definition
        .slice(start, end)
        .replace(/^\r?\n/, '')
        .replace(/\n---\s*$/, '')
        .trimEnd()
    })
    activeFileName.value = serviceFiles.value[0]
  }
  const serializeFiles = () =>
    serviceFiles.value
      .map((file) => `# AxiOps-File: ${file}\n${fileContents[file] || ''}`.trimEnd())
      .join('\n---\n')
  const focusFile = (file: string) => {
    if (!serviceFiles.value.includes(file)) return
    activeFileName.value = file
    if (props.activeFile !== file) emit('file-change', file)
    const marker = `# AxiOps-File: ${file}`
    const position = editorContent.value.indexOf(marker)
    if (position < 0) {
      void nextTick(() => yamlEditor.value?.gotoLine(1))
      return
    }
    const sectionStart = position + marker.length
    const nextMarker = editorContent.value.indexOf('\n# AxiOps-File:', sectionStart)
    const sectionEnd = nextMarker < 0 ? editorContent.value.length : nextMarker
    const section = editorContent.value.slice(sectionStart, sectionEnd)
    const kindOffset = section.search(/^\s*kind\s*:/m)
    const apiVersionOffset = section.search(/^\s*apiVersion\s*:/m)
    const targetOffset = kindOffset >= 0 ? kindOffset : apiVersionOffset
    const targetPosition = sectionStart + (targetOffset >= 0 ? targetOffset : 0)
    const line = editorContent.value.slice(0, targetPosition).split('\n').length
    void nextTick(() => yamlEditor.value?.gotoLine(line))
  }
  const openNewFile = () => {
    if (editorReadonly.value || loading.value || saving.value || loadError.value) return
    newFileName.value = ''
    newFileContent.value = ''
    newFileVisible.value = true
  }
  const createManualFile = () => {
    if (editorReadonly.value || loading.value || saving.value || loadError.value) return
    const name = newFileName.value.trim()
    if (!/^[a-zA-Z0-9][a-zA-Z0-9_.-]*\.ya?ml$/.test(name) || name.length > 128)
      return void ElMessage.warning('请输入有效的 .yaml 或 .yml 文件名')
    if (serviceFiles.value.includes(name))
      return void ElMessage.warning('文件已经存在，请选择它进行编辑')
    if (!newFileContent.value.trim()) return void ElMessage.warning('请填写 YAML 内容')
    // Preserve every existing file; adding is a local draft until Save template.
    serviceFiles.value.push(name)
    fileContents[name] = newFileContent.value
    activeFileName.value = name
    editorContent.value = serializeFiles()
    editing.value = true
    previewMode.value = false
    newFileVisible.value = false
    newFileContent.value = ''
  }
  const toggleEditing = () => {
    if (loading.value || saving.value) return
    syncEditorFromRevision()
    savedEditorState.value = editorState()
  }
  const unlockEditor = () => {
    if (!canUnlockEditor.value) return
    editing.value = true
  }
  const addVariable = () =>
    customVariables.value.push({ id: Date.now(), key: '', value: '', editing: true })
  const saveEditorShortcut = () => {
    if (editorReadonly.value || saving.value || loading.value || !dirty.value) return
    if (props.environmentMode) void saveEnvironment()
    else void saveRevision()
  }
  const stopResize = () => {
    resizing.value = false
    document.body.style.removeProperty('cursor')
    document.body.style.removeProperty('user-select')
    window.removeEventListener('mousemove', resizeEditor)
    window.removeEventListener('mouseup', stopResize)
  }
  const resizeEditor = (event: MouseEvent) => {
    if (!resizing.value) return
    const grid = document.querySelector<HTMLElement>('.service-workbench .workbench-grid')
    if (!grid) return
    const rect = grid.getBoundingClientRect()
    const next = ((event.clientX - rect.left) / rect.width) * 100
    editorWidth.value = Math.min(78, Math.max(42, next))
  }
  const startResize = () => {
    resizing.value = true
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
    window.addEventListener('mousemove', resizeEditor)
    window.addEventListener('mouseup', stopResize)
  }
  onBeforeUnmount(stopResize)
  const validateVariables = (silent = false, allowEnvironmentValues = false) => {
    const keys = new Set(customVariables.value.map((item) => item.key.trim()))
    const placeholders = [
      ...editorContent.value.matchAll(
        /\{\{\s*\.?([A-Za-z][A-Za-z0-9_]*)\s*\}\}|\$([A-Za-z][A-Za-z0-9_-]*)\$/g
      )
    ].map((match) => match[1] || match[2])
    const missing = [...new Set(placeholders)].filter(
      (key) =>
        !keys.has(key) &&
        ![
          'Product',
          'Service',
          'Namespace',
          'EnvName',
          'ClusterName',
          'Image',
          `${props.service?.code || 'service'}-image`
        ].includes(key)
    )
    const fail = (message: string) => {
      if (!silent) ElMessage.warning(message)
      return false
    }
    if (customVariables.value.some((item) => !/^[A-Za-z][A-Za-z0-9_]*$/.test(item.key.trim())))
      return fail('变量名须以字母开头，只能包含字母、数字和下划线')
    if (keys.size !== customVariables.value.length) return fail('变量名不能重复')
    if (missing.length) return fail(`变量未定义：${missing.join(', ')}`)
    if (!allowEnvironmentValues && unsetVariables.value.length)
      return fail(`缺少变量值：${unsetVariables.value.join(', ')}`)
    if (!editorContent.value.includes('apiVersion:') || !editorContent.value.includes('kind:'))
      return fail('YAML 缺少 apiVersion 或 kind')
    try {
      // System values are injected at deployment; empty shared defaults may be
      // supplied by an environment. Do not erase their YAML nodes during validation.
      const rendered = editorContent.value.replace(
        /\{\{\s*\.?([A-Za-z][A-Za-z0-9_]*)\s*\}\}|\$([A-Za-z][A-Za-z0-9_-]*)\$/g,
        (_match, dotKey, dollarKey) =>
          variableMap.value[dotKey || dollarKey]?.trim() || 'axiops-variable'
      )
      const documents: unknown[] = []
      yaml.loadAll(rendered, (document) => documents.push(document))
      const resources = documents.filter((document) => document != null) as Record<string, any>[]
      if (!resources.length) return fail('YAML 中没有可部署的资源')
      for (const resource of resources) {
        if (
          typeof resource !== 'object' ||
          Array.isArray(resource) ||
          typeof resource.apiVersion !== 'string' ||
          !resource.apiVersion.trim() ||
          typeof resource.kind !== 'string' ||
          !resource.kind.trim() ||
          typeof resource.metadata?.name !== 'string' ||
          !resource.metadata.name.trim()
        )
          return fail('每个资源都需要有效的 apiVersion、kind 和 metadata.name；多个资源用 --- 分隔')
        if (resource.kind === 'Secret')
          return fail('Secret 请通过环境凭据管理并在 YAML 中引用，不保存密码正文')
      }
    } catch (error: any) {
      const line = error?.mark?.line
      if (typeof line === 'number') void nextTick(() => yamlEditor.value?.gotoLine(line + 1))
      return fail(
        `YAML 语法错误${typeof line === 'number' ? `（第 ${line + 1} 行）` : ''}：${error?.reason || error?.message || '无法解析'}`
      )
    }
    if (!silent) {
      if (allowEnvironmentValues && unsetVariables.value.length)
        ElMessage.warning(
          `YAML 结构校验通过；待目标环境提供变量：${unsetVariables.value.join(', ')}`
        )
      else ElMessage.success('YAML 语法与变量校验通过；部署时仍会检查目标环境')
    }
    return true
  }
  const syncVariables = (definition: string, legacyDefaults = false) => {
    const keys = [...new Set(referencedVariables(definition))]
    const defaults: Record<string, string> = {
      cpuRequest: '100m',
      memoryRequest: '128Mi',
      cpuLimit: '500m',
      memoryLimit: '512Mi',
      healthCheckType: 'none',
      healthPath: '/health',
      healthPort: '0',
      probePeriodSeconds: '10',
      probeTimeoutSeconds: '3',
      probeFailureThreshold: '3',
      startupFailureThreshold: '30'
    }
    keys.forEach((key) => {
      if (!customVariables.value.some((item) => item.key === key))
        customVariables.value.push({
          id: Date.now() + customVariables.value.length,
          key,
          value:
            defaults[key] ||
            (legacyDefaults ? (key === 'replicas' ? '1' : key === 'port' ? '8080' : '') : ''),
          editing: false
        })
    })
  }
  const syncEditorFromRevision = () => {
    const definition = selectedRevision.value?.definition || ''
    customVariables.value = []
    const match = definition.match(/^# AxiOps-Variables: (.+)$/m)
    if (match) {
      try {
        const values = JSON.parse(match[1]) as Array<{ key?: string; value?: string }>
        if (Array.isArray(values))
          customVariables.value = values
            .filter((item) => item.key)
            .map((item, index) => ({
              id: index + 1,
              key: String(item.key),
              value: String(item.value || ''),
              editing: false
            }))
      } catch {
        throw new Error('服务变量元数据无效，请修复 YAML 后重新加载')
      }
    }
    const clean = definition.replace(/^# AxiOps-Variables: .+\r?\n?/m, '')
    const normalized = clean
    editorContent.value = normalized
    syncVariables(normalized, false)
    parseFiles(normalized)
  }
  const syncEditorFromEnvironmentConfig = () => {
    const config = props.environmentConfig
    customVariables.value = []
    initialEnvironmentVariables.value = {}
    if (!config) {
      editorContent.value = ''
      parseFiles('')
      return
    }
    const metadata = config.definition.match(/^# AxiOps-Variables: (.+)$/m)
    if (metadata) {
      const parameters = JSON.parse(metadata[1])
      if (!Array.isArray(parameters)) throw new Error('模板参数元数据无效')
      parameters.forEach((item, index) => {
        if (item.key)
          customVariables.value.push({
            id: index + 1,
            key: String(item.key),
            value: String(item.value ?? ''),
            editing: false
          })
      })
    }
    if (Array.isArray(config.variables)) {
      config.variables.forEach((item, index) => {
        if (!item?.key || item.secret) return
        const key = String(item.key)
        const value = String(item.value ?? '')
        const existing = customVariables.value.find((item) => item.key === key)
        if (existing) existing.value = value
        else customVariables.value.push({ id: index + 1, key, value, editing: false })
        initialEnvironmentVariables.value[key] = value
      })
    }
    if (config.replicas != null && !customVariables.value.some((item) => item.key === 'replicas')) {
      const value = String(config.replicas)
      customVariables.value.push({ id: Date.now(), key: 'replicas', value, editing: false })
      initialEnvironmentVariables.value.replicas = value
    }
    const clean = config.definition.replace(/^# AxiOps-Variables: .+\r?\n?/m, '')
    const normalized = clean
    editorContent.value = normalized
    syncVariables(normalized, false)
    parseFiles(normalized)
  }
  const loadRevisions = async () => {
    const request = ++revisionRequest,
      serviceID = props.service?.id
    if (!serviceID) return
    loading.value = true
    loadError.value = ''
    savedEditorState.value = ''
    revisions.value = []
    selectedRevisionId.value = 0
    editorContent.value = ''
    customVariables.value = []
    previewMode.value = false
    if (props.environmentMode) {
      revisions.value = []
      selectedRevisionId.value = props.environmentConfig?.environment_yaml_revision_id || 0
      syncEditorFromEnvironmentConfig()
      editing.value = !props.environmentConfig?.bound && props.service?.source_type === 'manual'
      savedEditorState.value = editorState()
      loading.value = false
      return
    }
    try {
      const items = (await listServiceRevisionsApi(serviceID, true)) || []
      if (request !== revisionRequest) return
      revisions.value = [...items].sort((a, b) => b.revision_number - a.revision_number)
      selectedRevisionId.value = revisions.value[0]?.id || 0
      if (revisions.value.length) syncEditorFromRevision()
      else if (props.service?.delivery_type !== 'helm') {
        customVariables.value = []
        const initial = ''
        editorContent.value = initial
        parseFiles(initial)
      }
      editing.value = false
      savedEditorState.value = editorState()
    } catch (error: any) {
      if (request !== revisionRequest) return
      loadError.value = error?.message || 'Revision 鍔犺浇澶辫触锛岄噸鏂拌鍙栨垚鍔熷悗鎵嶈兘缂栬緫'
      editing.value = false
      ElMessage.error(error?.message || 'Revision 鍔犺浇澶辫触')
    } finally {
      if (request === revisionRequest) loading.value = false
    }
  }
  const saveRevision = async () => {
    if (
      loading.value ||
      saving.value ||
      loadError.value ||
      !props.canEdit ||
      props.service?.delivery_type === 'helm'
    )
      return
    if (!props.service?.id || !editorContent.value.trim())
      return ElMessage.warning('服务定义不能为空')
    if (!validateVariables(false, true)) return
    const serviceID = props.service.id,
      request = revisionRequest
    saving.value = true
    try {
      await createServiceRevisionApi(serviceID, {
        source_commit: 'manual-edit',
        expected_revision_id: selectedRevisionId.value,
        definition: `# AxiOps-Variables: ${JSON.stringify(customVariables.value.map(({ key, value }) => ({ key, value })))}\n${editorContent.value.trim()}`
      })
      if (request !== revisionRequest || props.service?.id !== serviceID || props.environmentMode)
        return
      ElMessage.success('当前服务配置已保存')
      const reload = loadRevisions(),
        reloadRequest = revisionRequest
      await reload
      if (
        reloadRequest === revisionRequest &&
        props.service?.id === serviceID &&
        !props.environmentMode
      )
        emit('saved')
    } catch (error: any) {
      if (request === revisionRequest && props.service?.id === serviceID && !props.environmentMode)
        ElMessage.error(error?.message || '服务配置保存失败')
    } finally {
      saving.value = false
    }
  }
  const getEnvironmentDraft = () => {
    if (
      loading.value ||
      saving.value ||
      loadError.value ||
      !props.canEdit ||
      !props.environmentConfig ||
      props.service?.delivery_type === 'helm'
    )
      return
    if (!editorContent.value.trim()) return void ElMessage.warning('环境 YAML 不能为空')
    if (!validateVariables()) return
    const variables: Record<string, string> = {}
    customVariables.value.forEach((item) => {
      const key = item.key.trim()
      if (!key || initialEnvironmentVariables.value[key] === item.value) return
      variables[key] = item.value
    })
    return { definition: editorContent.value.trim(), variables }
  }
  const saveEnvironment = () => {
    const payload = getEnvironmentDraft()
    if (payload) emit('environment-save', payload)
  }
  watch(editorContent, (definition) => syncVariables(definition, false))
  watch(
    () => [
      props.environmentMode,
      props.environmentConfig?.environment_id,
      props.environmentConfig?.environment_yaml_revision_id,
      props.environmentConfig?.definition,
      props.environmentConfig?.variables
    ],
    () => {
      if (props.environmentMode) {
        syncEditorFromEnvironmentConfig()
        editing.value = !props.environmentConfig?.bound && props.service?.source_type === 'manual'
        savedEditorState.value = editorState()
      }
    },
    { deep: true }
  )
  watch(
    () => props.activeFile,
    (file) => {
      if (file && serviceFiles.value.includes(file)) {
        focusFile(file)
      }
    },
    { immediate: true }
  )
  watch(
    serviceFiles,
    () => {
      const file = props.activeFile
      if (file && serviceFiles.value.includes(file)) focusFile(file)
    },
    { deep: true }
  )
  watch(
    () => props.service?.id,
    () => {
      newFileVisible.value = false
      newFileContent.value = ''
      void loadRevisions()
    },
    { immediate: true }
  )
  onBeforeUnmount(() => revisionRequest++)
  defineExpose({
    dirty,
    saving,
    reloadDefinition: loadRevisions,
    revisionId: selectedRevisionId,
    openNewYaml: openNewFile,
    getEnvironmentDraft,
    discardChanges: () => {
      if (props.environmentMode) syncEditorFromEnvironmentConfig()
      else syncEditorFromRevision()
      savedEditorState.value = editorState()
    }
  })
</script>

<style scoped lang="scss">
  .probe-options {
    display: flex;
    flex-wrap: wrap;
    gap: 0 12px;
  }
  .probe-help {
    font-size: 12px;
    line-height: 1.6;
    color: var(--el-text-color-secondary);
  }
  .probe-summary .el-select {
    width: 210px;
  }
  .variable-warning {
    color: #b45309;
    font-size: 13px;
    line-height: 1.6;
  }
  .runtime-defaults {
    display: grid;
    gap: 5px;
    margin-bottom: 10px;
    padding: 8px 10px;
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    font-size: 12px;
  }
  .runtime-defaults div {
    display: flex;
    justify-content: space-between;
    gap: 12px;
  }
  .runtime-defaults code {
    color: #166534;
  }
  .runtime-defaults span {
    color: #64748b;
    text-align: right;
  }
  .service-workbench,
  .no-service-state {
    height: 100%;
    min-height: 0;
    min-width: 0;
    color: #1e293b;
    background: #fff;
  }
  .stage-toolbar-actions,
  .save-bar,
  .save-actions {
    display: flex;
    align-items: center;
  }
  .workbench-grid {
    position: relative;
    width: 100%;
    height: 100%;
    min-width: 0;
    min-height: 0;
    display: grid;
    grid-template-columns: minmax(560px, 1fr) minmax(360px, 420px);
    overflow: visible;
  }
  .workbench-resizer {
    position: absolute;
    top: 0;
    bottom: 0;
    left: calc(var(--editor-width) - 4px);
    z-index: 3;
    width: 8px;
    padding: 0;
    border: 0;
    border-left: 1px solid #e2e8f0;
    border-right: 1px solid transparent;
    background: transparent;
    cursor: col-resize;
  }
  .workbench-resizer:hover,
  .workbench-resizer:focus-visible {
    border-left-color: #60a5fa;
    background: rgba(96, 165, 250, 0.12);
    outline: 0;
  }
  .definition-stage {
    position: relative;
    min-width: 0;
    width: 100%;
    min-height: 0;
    height: 100%;
    display: flex;
    flex-direction: column;
    overflow-y: auto;
    overflow-x: hidden;
    background: #fff;
  }
  .stage-toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 8px;
    padding: 8px 14px;
    border-bottom: 1px solid #e2e8f0;
  }
  .stage-title {
    display: inline-flex;
    align-items: center;
    gap: 7px;
    min-width: 0;
    overflow: hidden;
    color: #334155;
    font-family: var(--ax-font-mono);
    font-size: 13px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .stage-toolbar-actions {
    gap: 6px;
  }
  .definition-panel {
    display: flex;
    flex-direction: column;
    flex: 1 1 auto;
    min-width: 0;
    min-height: 0;
    overflow: visible;
    padding: 0;
  }
  .environment-binding-notice {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 10px 14px;
    border-bottom: 1px solid #dbeafe;
    background: #eff6ff;
  }
  .environment-binding-notice > div {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
  }
  .environment-binding-notice strong {
    color: #1e3a8a;
    font-size: 13px;
    font-weight: 600;
  }
  .environment-binding-notice span {
    color: #475569;
    font-size: 12px;
    line-height: 1.5;
  }
  .definition-file-index {
    display: flex;
    flex: 0 0 auto;
    align-items: center;
    gap: 6px;
    min-height: 34px;
    padding: 6px 14px;
    overflow-x: auto;
    border-bottom: 1px solid #eef2f7;
    background: #fbfcfe;
    white-space: nowrap;
  }
  .definition-file-label {
    color: #94a3b8;
    font-size: 12px;
  }
  .definition-file-chip {
    flex: 0 0 auto;
    padding: 3px 7px;
    border: 1px solid #dbe4f0;
    border-radius: 5px;
    background: #fff;
    color: #475569;
    font-family: var(--ax-font-mono);
    font-size: 12px;
    cursor: pointer;
    transition:
      border-color 0.15s ease,
      background 0.15s ease,
      color 0.15s ease;
  }
  .definition-file-chip:hover,
  .definition-file-chip.is-active {
    border-color: #93c5fd;
    background: #eff6ff;
    color: #2563eb;
  }
  .definition-file-chip:focus-visible {
    outline: 2px solid rgba(59, 130, 246, 0.35);
    outline-offset: 1px;
  }
  .editor-frame {
    position: relative;
    flex: 1 1 auto;
    width: 100%;
    height: auto;
    min-height: min(420px, 52vh);
    min-width: 0;
    overflow: hidden;
    border: 0;
    border-radius: 0;
  }
  .editor-frame.is-locked {
    background: #f8fafc;
  }
  .editor-unlock-button {
    position: absolute;
    right: 12px;
    top: 10px;
    z-index: 3;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    padding: 0;
    color: #475569;
    background: rgba(255, 255, 255, 0.96);
    border: 1px solid #cbd5e1;
    border-radius: 50%;
    box-shadow: 0 2px 8px rgba(15, 23, 42, 0.14);
    cursor: pointer;
    transition:
      color 0.15s ease,
      border-color 0.15s ease,
      box-shadow 0.15s ease;
  }
  .editor-unlock-button:hover {
    color: #2563eb;
    border-color: #93c5fd;
    box-shadow: 0 3px 10px rgba(37, 99, 235, 0.2);
  }
  .editor-readonly-hint {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    min-height: 42px;
    padding: 8px 12px;
    color: #64748b;
    background: #f1f5f9;
    border: 1px solid #e2e8f0;
    border-top: 0;
    font-size: 13px;
  }
  .editor-readonly-hint > span {
    flex: 1 1 auto;
  }
  .editor-frame :deep(.yaml-editor-pro) {
    width: 100% !important;
    height: 100% !important;
    min-width: 0;
    border: 0;
    box-shadow: none;
  }
  .editor-frame :deep(.yaml-editor-pro .editor-body) {
    background: #fff !important;
  }
  .editor-frame :deep(.yaml-editor-pro .editor-wrapper) {
    background: #fff !important;
  }
  .empty-definition,
  .no-service-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 10px;
    color: #64748b;
    text-align: center;
  }
  .empty-definition {
    height: 100%;
    min-height: 320px;
  }
  .empty-definition strong,
  .no-service-state strong {
    color: #334155;
    font-size: 16px;
  }
  .empty-definition p,
  .no-service-state p {
    margin: 0;
    font-size: 13px;
  }
  .save-bar {
    position: sticky;
    bottom: 0;
    z-index: 8;
    flex: 0 0 auto;
    display: flex;
    align-items: center;
    justify-content: flex-start;
    flex-wrap: wrap;
    gap: 12px;
    padding: 9px 14px;
    border-top: 1px solid #e2e8f0;
    box-shadow: 0 -3px 10px rgba(15, 23, 42, 0.06);
    background: #fff;
  }
  .save-bar > span,
  .save-actions {
    gap: 6px;
  }
  .save-state {
    color: #64748b;
    font-size: 12px;
    white-space: nowrap;
  }
  .save-state.is-dirty {
    color: #b45309;
  }
  .save-actions {
    flex: 0 1 auto;
    min-width: 0;
    flex-wrap: wrap;
    justify-content: flex-start;
  }
  .variables-rail {
    min-width: 0;
    display: flex;
    flex-direction: column;
    border-left: 1px solid #e5e7eb;
    background: #fff;
    color: #1e293b;
  }
  .rail-heading {
    min-height: 70px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 0 14px;
    border-bottom: 1px solid #e5e7eb;
  }
  .rail-heading > div {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .rail-heading strong {
    font-size: 15px;
    color: #1e293b !important;
  }
  .section-eyebrow {
    color: #94a3b8;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.08em;
  }
  .variables-scroll {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 12px;
    line-height: 1.6;
  }
  .variable-section {
    padding: 11px 0;
    border-bottom: 1px solid #e5e7eb;
  }
  .variable-section:first-child {
    padding-top: 2px;
  }
  .variable-section-title {
    display: flex;
    flex-direction: column;
    gap: 3px;
    margin-bottom: 9px;
  }
  .variable-section-title strong {
    font-size: 13.5px;
    color: #1e293b !important;
  }
  .variable-section-title small,
  .component-copy small {
    color: #64748b;
    font-size: 12px;
    color: #64748b !important;
  }
  .component-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 9px 10px;
    border: 1px solid #e5e7eb;
    border-radius: 5px;
    background: #f8fafc;
  }
  .component-kind {
    width: 25px;
    height: 25px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border-radius: 5px;
    color: #16a34a;
    background: #dcfce7;
  }
  .component-copy {
    min-width: 0;
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .component-copy strong,
  .component-copy small {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .component-copy strong,
  .system-variable-list dd {
    color: #1e293b !important;
  }
  .component-copy strong {
    font-size: 13.5px;
  }
  .system-variable-list {
    margin: 0;
  }
  .system-variable-list > div {
    display: grid;
    grid-template-columns: 120px minmax(0, 1fr);
    gap: 8px;
    padding: 7px 0;
    border-bottom: 1px solid #f1f5f9;
    font-size: 13px;
  }
  .system-variable-list dt {
    color: #64748b;
    font-family: var(--ax-font-mono);
  }
  .variables-rail :deep(.el-input__wrapper) {
    background: #fff !important;
    box-shadow: 0 0 0 1px #dbe3ec inset !important;
  }
  .variables-rail :deep(.el-input__inner) {
    color: #1e293b !important;
    -webkit-text-fill-color: #1e293b !important;
  }
  .variables-rail :deep(.el-input__inner::placeholder) {
    color: #94a3b8 !important;
    -webkit-text-fill-color: #94a3b8 !important;
  }
  .system-variable-list dd {
    min-width: 0;
    margin: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .variable-table-wrap {
    overflow-x: auto;
  }
  .variable-table {
    width: 100%;
    table-layout: fixed;
    border-collapse: collapse;
    font-size: 13px;
  }
  .variable-table th {
    padding: 8px 7px;
    color: #64748b;
    background: #f8fafc;
    font-weight: 500;
    text-align: left;
  }
  .variable-table td {
    padding: 7px;
    border-bottom: 1px solid #e8edf3;
    vertical-align: middle;
    overflow-wrap: anywhere;
  }
  .variable-table th:last-child {
    width: 46px;
  }
  .variable-table code {
    color: #334155;
    font-family: var(--ax-font-mono);
  }
  .variable-value {
    display: inline-block;
    max-width: 100%;
    padding: 5px 9px;
    border: 1px solid #dbe3ec;
    border-radius: 4px;
    background: #f8fafc;
    color: #1e293b;
    font-size: 13px;
  }
  .parameter-group {
    margin-bottom: 10px;
  }
  .parameter-group-heading {
    display: flex;
    align-items: baseline;
    gap: 6px;
    margin-bottom: 7px;
  }
  .parameter-group-heading strong,
  .probe-summary > span,
  .probe-summary > strong {
    color: #1e293b;
    font-size: 13px;
    font-weight: 600;
  }
  .parameter-group-heading small {
    color: #94a3b8;
    font-size: 11.5px;
    font-weight: 400;
  }
  .variable-table .parameter-label {
    display: block;
    color: #1e293b;
    font-size: 13px;
    font-weight: 600;
  }
  .variable-table td small {
    display: block;
    margin-top: 2px;
    color: #94a3b8;
    font-size: 11.5px;
    line-height: 1.45;
  }
  .probe-summary {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px;
    margin-bottom: 7px;
  }
  .variable-actions {
    width: 62px;
    white-space: nowrap;
  }
  .add-variable {
    margin-top: 5px;
    padding-left: 0;
  }
  @media (max-width: 1180px) {
    .workbench-grid {
      grid-template-columns: minmax(420px, 1fr) minmax(320px, 380px);
    }
  }
  @media (max-width: 860px) {
    .workbench-grid {
      display: block;
      overflow: auto;
    }
    .definition-stage {
      min-height: 420px;
    }
    .workbench-resizer {
      display: none;
    }
    .variables-rail {
      display: none;
    }
  }
</style>
