<template>
  <ElDialog
    v-model="dialogVisible"
    :title="`${environment?.name || '环境'} · 环境配置`"
    width="min(980px, 94vw)"
    destroy-on-close
    :before-close="requestClose"
  >
    <ElTabs v-model="activeTab">
      <ElTabPane
        v-if="environment && services.some((s) => s.delivery_type === 'helm')"
        label="Helm 共享 values"
        name="helm-shared"
        ><HelmSharedValuesEditor
          :environment="environment"
          :can-edit="editable"
          @saved="emit('changed')"
      /></ElTabPane>
      <ElTabPane label="环境变量" name="variables">
        <div v-loading="variableLoading" class="config-panel">
          <div class="panel-heading">
            <span>
              <strong>{{ environment?.name }} · 环境共享变量</strong>
              <small>仅此环境共享；同名服务变量优先，不影响其他环境。保存后需重新提交部署。</small>
            </span>
            <div class="heading-actions">
              <ElTag v-if="variableSet" size="small" effect="plain">
                Revision {{ variableSet.revision }}
              </ElTag>
              <ElButton
                v-if="editable"
                text
                type="primary"
                :icon="Plus"
                :disabled="variableLoading || variableSaving || !!variableError"
                @click="addVariable"
              >
                添加变量
              </ElButton>
            </div>
          </div>

          <ElAlert v-if="variableError" :title="variableError" type="error" :closable="false"
            ><ElButton text @click="loadVariables">重新加载</ElButton></ElAlert
          >
          <p class="variable-help"
            >YAML 中用 <code v-pre>{{.变量名}}</code> 引用普通变量。应用进程需要的环境变量，须在容器
            env / envFrom 中声明。Image、Namespace 等系统值无需填写。</p
          >
          <p class="variable-help"
            >凭据引用目前用于 Helm values；YAML 密码请引用目标环境的 Kubernetes Secret（secretKeyRef
            / envFrom），不会把这里的凭据自动注入容器。</p
          >

          <div v-if="!variableError && variableItems.length" class="variable-list">
            <div class="variable-header">
              <span>变量名</span><span>类型</span><span>值或凭据</span><span />
            </div>
            <div v-for="(item, index) in variableItems" :key="index" class="variable-row">
              <ElInput
                v-model="item.key"
                :disabled="!editable || variableLoading || variableSaving || !!variableError"
                placeholder="VARIABLE_NAME"
              />
              <ElSelect
                v-model="item.is_secret"
                :disabled="!editable || variableLoading || variableSaving || !!variableError"
              >
                <ElOption label="普通变量" :value="false" />
                <ElOption label="Secret" :value="true" />
              </ElSelect>
              <ElSelect
                v-if="item.is_secret"
                v-model="item.credential_id"
                :disabled="!editable || variableLoading || variableSaving || !!variableError"
                filterable
                placeholder="选择 Secret 凭据"
              >
                <ElOption
                  v-for="credential in secretCredentials"
                  :key="credential.id"
                  :label="credential.name"
                  :value="credential.id"
                />
                <ElOption
                  v-if="item.credential_id && !credentialExists(item.credential_id)"
                  :label="`凭据 #${item.credential_id}`"
                  :value="item.credential_id"
                />
              </ElSelect>
              <ElInput
                v-else
                v-model="item.value"
                :disabled="!editable || variableLoading || variableSaving || !!variableError"
                placeholder="变量值"
              />
              <ElButton
                v-if="editable"
                :icon="Trash2"
                text
                circle
                title="删除变量"
                :disabled="variableSaving || variableLoading"
                @click="variableItems.splice(index, 1)"
              />
              <span v-else />
            </div>
          </div>
          <ElEmpty
            v-else-if="!variableError && !variableLoading"
            :image-size="64"
            description="暂无环境共享变量"
          />

          <details v-if="variableChanges.length" class="variable-changes" open>
            <summary>待保存变更 · {{ variableChanges.length }} 项</summary>
            <ElTable :data="variableChanges" max-height="230">
              <ElTableColumn prop="key" label="变量" min-width="140" />
              <ElTableColumn prop="before" label="保存前" min-width="170" show-overflow-tooltip />
              <ElTableColumn prop="after" label="保存后" min-width="170" show-overflow-tooltip />
            </ElTable>
            <small>仅引用这些变量、且没有同名服务覆盖的配置受影响；此处不执行部署。</small>
          </details>

          <div v-if="editable" class="panel-footer">
            <ElButton
              :disabled="!variableChanges.length || variableSaving"
              @click="restoreVariables"
              >撤销修改</ElButton
            >
            <ElButton
              type="primary"
              :loading="variableSaving"
              :disabled="variableLoading || !!variableError || !variableChanges.length"
              @click="saveVariables"
            >
              保存配置，待部署
            </ElButton>
          </div>
        </div>
      </ElTabPane>

      <ElTabPane
        v-if="!selectedService || selectedService.delivery_type === 'k8s_yaml'"
        label="环境资源"
        name="resources"
      >
        <div class="config-panel resource-panel">
          <div class="panel-heading">
            <span
              ><strong>环境专属资源</strong
              ><small
                >Ingress、ConfigMap、Secret、PVC 仅保存到当前环境；重新提交 YAML
                部署后应用到集群。应用是否重载配置取决于服务本身。</small
              ></span
            >
            <ElButton v-if="editable" type="primary" plain :icon="Plus" @click="addResource"
              >添加资源</ElButton
            >
          </div>
          <div v-if="resources.length" class="resource-list">
            <div v-for="resource in resources" :key="resource.id" class="resource-row">
              <div class="resource-row-head"
                ><ElSelect
                  v-model="resource.kind"
                  size="small"
                  :disabled="!editable"
                  style="width: 180px"
                  ><ElOption label="Ingress" value="Ingress" /><ElOption
                    label="ConfigMap"
                    value="ConfigMap" /><ElOption label="Secret" value="Secret" /><ElOption
                    label="PVC"
                    value="PersistentVolumeClaim" /></ElSelect
                ><ElInput
                  v-model="resource.name"
                  size="small"
                  :disabled="!editable"
                  placeholder="资源名称"
                  style="width: 220px" />
                ><span>Revision {{ resource.revision }}</span
                ><ElButton
                  v-if="editable"
                  text
                  type="danger"
                  :icon="Trash2"
                  @click="removeResource(resource)"
              /></div>
              <ElInput
                v-model="resource.manifest"
                type="textarea"
                :rows="7"
                :disabled="!editable"
                spellcheck="false"
                class="resource-editor"
              />
              <div v-if="editable" class="resource-actions"
                ><ElButton
                  type="primary"
                  size="small"
                  :loading="resourceSavingId === resource.id"
                  @click="saveResource(resource)"
                  >保存资源</ElButton
                ></div
              >
            </div>
          </div>
          <ElEmpty v-else description="当前环境暂无专属资源" />
        </div>
      </ElTabPane>

      <ElTabPane v-if="false" label="服务覆盖" name="overrides">
        <div class="config-panel">
          <div class="panel-heading">
            <span>
              <strong>环境内服务目标状态</strong>
              <small>固定不可变服务版本，可按环境覆盖镜像与副本数</small>
            </span>
          </div>
          <ElForm label-position="top">
            <ElFormItem label="服务">
              <ElSelect
                v-model="selectedServiceId"
                filterable
                placeholder="选择服务"
                style="width: 100%"
              >
                <ElOption
                  v-for="service in services"
                  :key="service.id"
                  :label="service.name"
                  :value="service.id"
                />
              </ElSelect>
            </ElFormItem>
          </ElForm>

          <div v-loading="overrideLoading">
            <ElForm v-if="selectedServiceId" label-position="top">
              <ElAlert
                type="info"
                :closable="false"
                show-icon
                class="override-note"
                title="环境覆盖只影响当前环境，不会修改服务基线或触发重新构建。"
              />
              <div class="override-summary">
                <div>
                  <span>当前目标</span>
                  <strong>{{ environmentTarget }}</strong>
                </div>
                <div>
                  <span>基线 Revision</span>
                  <strong>{{ baselineRevisionLabel }}</strong>
                </div>
                <div>
                  <span>环境覆盖 Revision</span>
                  <strong>{{
                    overrideExists ? `#${overrideForm.expected_revision}` : '未保存'
                  }}</strong>
                </div>
              </div>
              <div class="override-diff" v-if="overrideDiffRows.length">
                <div class="override-diff-heading">
                  <strong>配置差异</strong>
                  <small>空值表示继承服务基线或运行时构建产物</small>
                </div>
                <div class="override-diff-row" v-for="row in overrideDiffRows" :key="row.key">
                  <span>{{ row.label }}</span>
                  <code>{{ row.baseline }}</code>
                  <span class="diff-arrow">→</span>
                  <code :class="{ 'is-overridden': row.overridden }">{{ row.override }}</code>
                </div>
              </div>
              <ElFormItem label="服务版本" required>
                <ElSelect
                  v-model="overrideForm.service_revision_id"
                  :disabled="!editable"
                  filterable
                  placeholder="选择已导入版本"
                  style="width: 100%"
                >
                  <ElOption
                    v-for="revision in serviceRevisions"
                    :key="revision.id"
                    :label="revisionLabel(revision)"
                    :value="revision.id"
                  />
                </ElSelect>
              </ElFormItem>
              <div class="form-grid">
                <ElFormItem label="镜像覆盖">
                  <ElInput
                    v-model="overrideForm.image_ref"
                    :disabled="!editable"
                    placeholder="留空则使用构建产物或服务默认镜像"
                  />
                </ElFormItem>
                <ElFormItem label="副本数覆盖">
                  <ElInputNumber
                    v-model="overrideForm.replicas"
                    :disabled="!editable || environment?.environment_type === 'host_group'"
                    :min="0"
                    :max="10000"
                    controls-position="right"
                    placeholder="不覆盖"
                    style="width: 100%"
                  />
                </ElFormItem>
              </div>
              <div class="override-meta">
                <ElTag size="small" effect="plain">
                  Revision {{ overrideForm.expected_revision || 0 }}
                </ElTag>
                <span v-if="!overrideExists">尚未保存环境覆盖</span>
              </div>
              <div v-if="editable" class="panel-footer">
                <ElButton
                  type="primary"
                  :loading="overrideSaving"
                  :disabled="!overrideForm.service_revision_id"
                  @click="saveOverride"
                >
                  保存服务覆盖
                </ElButton>
              </div>
              <ElCollapse
                v-if="baselineDefinition"
                v-model="advancedPanels"
                class="baseline-collapse"
              >
                <ElCollapseItem name="definition" title="查看基线定义（只读）">
                  <pre class="baseline-definition">{{ baselineDefinition }}</pre>
                  <small class="field-hint"
                    >需要修改资源结构时，请在服务详情中创建新的 Revision，再为环境选择版本。</small
                  >
                </ElCollapseItem>
              </ElCollapse>
            </ElForm>
            <ElEmpty v-else :image-size="64" description="请选择服务" />
          </div>
        </div>
      </ElTabPane>

      <ElTabPane
        v-if="false && environment?.environment_type === 'k8s_namespace'"
        label="资源预览"
        name="preview"
      >
        <div class="config-panel">
          <div class="panel-heading">
            <span>
              <strong>发布前资源差异</strong>
              <small>基于目标集群 Server-Side Apply dry-run 生成</small>
            </span>
            <ElButton
              type="primary"
              :icon="FileSearch"
              :loading="previewLoading"
              :disabled="!selectedServiceId || !previewRevisionId"
              @click="generatePreview"
            >
              生成预览
            </ElButton>
          </div>

          <ElForm label-position="top">
            <div class="preview-form-grid">
              <ElFormItem label="服务">
                <ElSelect
                  v-model="selectedServiceId"
                  filterable
                  placeholder="选择服务"
                  style="width: 100%"
                >
                  <ElOption
                    v-for="service in services"
                    :key="service.id"
                    :label="service.name"
                    :value="service.id"
                  />
                </ElSelect>
              </ElFormItem>
              <ElFormItem label="服务版本">
                <ElSelect
                  v-model="previewRevisionId"
                  filterable
                  placeholder="选择已导入版本"
                  style="width: 100%"
                >
                  <ElOption
                    v-for="revision in serviceRevisions"
                    :key="revision.id"
                    :label="revisionLabel(revision)"
                    :value="revision.id"
                  />
                </ElSelect>
              </ElFormItem>
              <ElFormItem v-if="selectedService?.delivery_type === 'helm'" label="Release 名称">
                <ElInput v-model="previewReleaseName" placeholder="Helm Release 名称" />
              </ElFormItem>
            </div>
          </ElForm>

          <template v-if="previewResult">
            <div class="diff-summary">
              <span class="diff-create"
                ><strong>{{ previewResult?.summary.create }}</strong> 新建</span
              >
              <span class="diff-update"
                ><strong>{{ previewResult?.summary.update }}</strong> 更新</span
              >
              <span class="diff-delete"
                ><strong>{{ previewResult?.summary.delete }}</strong> 删除</span
              >
              <span class="diff-unchanged"
                ><strong>{{ previewResult?.summary.unchanged }}</strong> 无变化</span
              >
            </div>
            <ElTable :data="previewResult?.changes || []" max-height="310" class="preview-table">
              <ElTableColumn prop="kind" label="Kind" min-width="130" />
              <ElTableColumn prop="namespace" label="Namespace" min-width="150" />
              <ElTableColumn prop="name" label="资源名称" min-width="180" />
              <ElTableColumn label="变化" width="100" align="center">
                <template #default="{ row }">
                  <ElTag :type="actionTagType(row.action)" effect="light" size="small">
                    {{ actionLabel(row.action) }}
                  </ElTag>
                </template>
              </ElTableColumn>
            </ElTable>
            <div v-if="previewValueRows.length" class="values-preview">
              <div class="values-title">Helm Values</div>
              <div v-for="item in previewValueRows" :key="item.key" class="value-row">
                <code>{{ item.key }}</code
                ><span>{{ item.value }}</span>
              </div>
            </div>
          </template>
          <ElEmpty v-else :image-size="64" description="尚未生成资源预览" />
        </div>
      </ElTabPane>
    </ElTabs>

    <template #footer>
      <ElButton @click="requestClose()">关闭</ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue'
  import HelmSharedValuesEditor from './HelmSharedValuesEditor.vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { FileSearch, Plus, Trash2 } from 'lucide-vue-next'
  import * as yaml from 'js-yaml'
  import { HttpError } from '@/utils/http/error'
  import {
    getEnvironmentServiceOverrideApi,
    getEnvironmentVariableSetApi,
    listCredentialsApi,
    listEnvironmentResourcesApi,
    saveEnvironmentResourceApi,
    deleteEnvironmentResourceApi,
    listServiceRevisionsApi,
    previewEnvironmentResourcesApi,
    saveEnvironmentServiceOverrideApi,
    updateEnvironmentVariableSetApi,
    type AxiOpsCredential,
    type AxiOpsDeploymentPreview,
    type AxiOpsEnvironment,
    type AxiOpsEnvironmentServiceOverride,
    type AxiOpsEnvironmentResource,
    type AxiOpsService,
    type AxiOpsServiceRevision,
    type AxiOpsVariableSet
  } from '@/api/axiops'

  interface VariableFormItem {
    key: string
    value: string
    credential_id: number
    is_secret: boolean
  }

  const props = defineProps<{
    initialTab?: 'variables' | 'resources'
    visible: boolean
    environment: AxiOpsEnvironment | null
    services: AxiOpsService[]
    editable: boolean
  }>()
  const emit = defineEmits<{ 'update:visible': [value: boolean]; changed: [] }>()

  const dialogVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })
  const activeTab = ref('variables')
  const variableLoading = ref(false)
  const variableSaving = ref(false)
  const overrideLoading = ref(false)
  const overrideSaving = ref(false)
  const previewLoading = ref(false)
  const resourceSavingId = ref(0)
  const resources = ref<AxiOpsEnvironmentResource[]>([])
  const variableSet = ref<AxiOpsVariableSet | null>(null)
  const variableItems = ref<VariableFormItem[]>([])
  const variableError = ref('')
  const savedVariableItems = ref<VariableFormItem[]>([])
  let variableGeneration = 0
  const variableChanges = computed(() => {
    const before = new Map(savedVariableItems.value.map((item) => [item.key, item]))
    const after = new Map(variableItems.value.map((item) => [item.key.trim(), item]))
    const show = (item?: VariableFormItem) =>
      !item ? '未配置' : item.is_secret ? `凭据引用 #${item.credential_id}` : item.value
    return [...new Set([...before.keys(), ...after.keys()])]
      .filter((key) => JSON.stringify(before.get(key)) !== JSON.stringify(after.get(key)))
      .map((key) => ({
        key: key || '（未命名）',
        before: show(before.get(key)),
        after: show(after.get(key))
      }))
  })
  function restoreVariables() {
    variableItems.value = savedVariableItems.value.map((item) => ({ ...item }))
  }
  async function requestClose(done?: () => void) {
    if (variableSaving.value) return
    if (variableChanges.value.length) {
      try {
        await ElMessageBox.confirm('环境共享变量有未保存修改，是否放弃？', '未保存修改', {
          confirmButtonText: '放弃修改',
          cancelButtonText: '继续编辑',
          type: 'warning'
        })
      } catch {
        return
      }
    }
    if (done) done()
    else dialogVisible.value = false
  }
  const secretCredentials = ref<AxiOpsCredential[]>([])
  const selectedServiceId = ref(0)
  const serviceRevisions = ref<AxiOpsServiceRevision[]>([])
  const overrideExists = ref(false)
  const previewRevisionId = ref(0)
  const previewReleaseName = ref('')
  const previewResult = ref<AxiOpsDeploymentPreview | null>(null)
  const advancedPanels = ref<string[]>([])
  const overrideForm = reactive({
    expected_revision: 0,
    service_revision_id: 0,
    image_ref: '',
    replicas: undefined as number | undefined
  })
  const selectedService = computed(() =>
    props.services.find((item) => item.id === selectedServiceId.value)
  )
  const previewValueRows = computed(() => flattenValues(previewResult.value?.values || {}))
  const selectedRevision = computed(() =>
    serviceRevisions.value.find((revision) => revision.id === overrideForm.service_revision_id)
  )
  const baselineRevisionLabel = computed(() => {
    if (!selectedRevision.value) return '未选择'
    return `#${selectedRevision.value.revision_number} · ${selectedRevision.value.source_commit?.slice(0, 10) || '本地定义'}`
  })
  const environmentTarget = computed(() => {
    if (!props.environment) return '-'
    return props.environment.environment_type === 'host_group'
      ? props.environment.host_group_ref || '-'
      : `集群 #${props.environment.cluster_id} / ${props.environment.namespace}`
  })
  const baselineDefinition = computed(() => {
    const definition = selectedRevision.value?.definition || ''
    if (!definition || selectedService.value?.delivery_type !== 'helm') return definition
    try {
      return JSON.stringify(JSON.parse(definition), null, 2)
    } catch {
      return definition
    }
  })
  const baselineReplicas = computed(() => {
    if (!selectedRevision.value?.definition || selectedService.value?.delivery_type !== 'k8s_yaml')
      return '-'
    try {
      const resources: unknown[] = []
      yaml.loadAll(selectedRevision.value.definition, (document) => resources.push(document))
      const values = resources
        .filter((item): item is Record<string, any> => Boolean(item && typeof item === 'object'))
        .filter((item) => ['Deployment', 'StatefulSet', 'ReplicaSet'].includes(String(item.kind)))
        .map((item) => item.spec?.replicas)
        .filter((value) => typeof value === 'number')
      return values.length ? values.join(', ') : '-'
    } catch {
      return '-'
    }
  })
  const overrideDiffRows = computed(() => {
    const baselineImage = selectedService.value?.image_name || '-'
    const overrideImage = overrideForm.image_ref.trim() || '继承构建产物/基线'
    const overrideReplicas =
      overrideForm.replicas === undefined ? '继承基线' : String(overrideForm.replicas)
    return [
      {
        key: 'revision',
        label: '服务版本',
        baseline: selectedService.value?.current_revision_id
          ? `#${selectedService.value.current_revision_id}`
          : '-',
        override: baselineRevisionLabel.value,
        overridden: Boolean(selectedRevision.value)
      },
      {
        key: 'image',
        label: '镜像引用',
        baseline: baselineImage,
        override: overrideImage,
        overridden: Boolean(overrideForm.image_ref.trim())
      },
      {
        key: 'replicas',
        label: '副本数',
        baseline: baselineReplicas.value,
        override: overrideReplicas,
        overridden: overrideForm.replicas !== undefined
      }
    ]
  })

  const resetOverride = () => {
    overrideExists.value = false
    Object.assign(overrideForm, {
      expected_revision: 0,
      service_revision_id: 0,
      image_ref: '',
      replicas: undefined
    })
  }

  const applyOverride = (override: AxiOpsEnvironmentServiceOverride) => {
    overrideExists.value = true
    Object.assign(overrideForm, {
      expected_revision: override.revision,
      service_revision_id: override.service_revision_id,
      image_ref: override.image_ref || '',
      replicas: override.replicas
    })
  }

  const loadVariables = async () => {
    if (!props.environment) return
    const generation = ++variableGeneration,
      environmentId = props.environment.id
    const current = () =>
      generation === variableGeneration && props.visible && props.environment?.id === environmentId
    variableSet.value = null
    variableItems.value = []
    savedVariableItems.value = []
    variableError.value = ''
    variableLoading.value = true
    try {
      const result = await getEnvironmentVariableSetApi(environmentId)
      if (!current()) return
      if (
        !result ||
        result.environment_id !== environmentId ||
        result.service_id !== 0 ||
        !Array.isArray(result.items)
      )
        throw Error('变量返回范围与当前环境不一致')
      variableSet.value = result
      variableItems.value = (result.items || []).map((item) => ({
        key: item.key,
        value: item.value || '',
        credential_id: item.credential_id || 0,
        is_secret: item.is_secret
      }))
      savedVariableItems.value = variableItems.value.map((item) => ({ ...item }))
    } catch (e: any) {
      if (current()) variableError.value = e?.message || '环境变量读取失败'
    } finally {
      if (current()) variableLoading.value = false
    }
  }

  const loadCredentials = async () => {
    if (!props.environment || !props.editable) return
    try {
      secretCredentials.value =
        (await listCredentialsApi({
          type: 'secret',
          project_id: props.environment.project_id
        })) || []
    } catch {
      secretCredentials.value = []
    }
  }

  const loadOverride = async () => {
    if (!props.environment || !selectedServiceId.value) return
    overrideLoading.value = true
    resetOverride()
    try {
      const revisions = await listServiceRevisionsApi(selectedServiceId.value)
      serviceRevisions.value = revisions || []
      let override: AxiOpsEnvironmentServiceOverride | null = null
      try {
        override = await getEnvironmentServiceOverrideApi(
          props.environment.id,
          selectedServiceId.value
        )
      } catch (error) {
        // A missing override is the valid initial state; other errors must remain visible.
        if (!(error instanceof HttpError && error.code === 50005)) {
          ElMessage.error(error instanceof Error ? error.message : '服务覆盖加载失败')
          return
        }
      }
      if (override) applyOverride(override)
      else {
        const service = props.services.find((item) => item.id === selectedServiceId.value)
        overrideForm.service_revision_id = service?.current_revision_id || revisions[0]?.id || 0
      }
      previewRevisionId.value = overrideForm.service_revision_id
    } finally {
      overrideLoading.value = false
    }
  }

  const addVariable = () =>
    variableItems.value.push({ key: '', value: '', credential_id: 0, is_secret: false })

  const addResource = () => {
    resources.value.unshift({
      id: 0,
      environment_id: props.environment?.id || 0,
      name: `resource-${resources.value.length + 1}`,
      kind: 'ConfigMap',
      manifest:
        'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: example\ndata:\n  KEY: value\n',
      revision: 0,
      updated_at: ''
    })
  }
  const loadResources = async () => {
    if (!props.environment) return
    resources.value = (await listEnvironmentResourcesApi(props.environment.id)) || []
  }
  const saveResource = async (resource: AxiOpsEnvironmentResource) => {
    if (!props.environment || !resource.name.trim() || !resource.manifest.trim())
      return void ElMessage.warning('资源名称和 YAML 不能为空')
    resourceSavingId.value = resource.id || -1
    try {
      const saved = await saveEnvironmentResourceApi(props.environment.id, {
        id: resource.id || undefined,
        name: resource.name.trim(),
        kind: resource.kind,
        manifest: resource.manifest,
        expected_revision: resource.revision || undefined
      })
      Object.assign(resource, saved)
      ElMessage.success('环境资源配置已保存，重新提交 YAML 部署后应用')
    } finally {
      resourceSavingId.value = 0
    }
  }
  const removeResource = async (resource: AxiOpsEnvironmentResource) => {
    if (!resource.id || !props.environment) {
      resources.value = resources.value.filter((item) => item !== resource)
      return
    }
    await deleteEnvironmentResourceApi(props.environment.id, resource.id)
    resources.value = resources.value.filter((item) => item.id !== resource.id)
    ElMessage.success('环境资源已删除')
  }

  const credentialExists = (id: number) =>
    secretCredentials.value.some((credential) => credential.id === id)

  const saveVariables = async () => {
    if (
      !props.editable ||
      variableSaving.value ||
      variableLoading.value ||
      variableError.value ||
      !variableSet.value ||
      !variableChanges.value.length ||
      variableSet.value.environment_id !== props.environment?.id
    )
      return
    const environmentId = props.environment.id,
      set = variableSet.value,
      generation = variableGeneration
    const items = variableItems.value.map((item) => ({
      key: item.key.trim(),
      value: item.is_secret ? '' : item.value,
      credential_id: item.is_secret ? item.credential_id : 0,
      is_secret: item.is_secret
    }))
    if (items.some((item) => !item.key)) return ElMessage.warning('变量名不能为空')
    if (new Set(items.map((item) => item.key)).size !== items.length)
      return ElMessage.warning('变量名不能重复')
    if (items.some((item) => item.is_secret && !item.credential_id))
      return ElMessage.warning('Secret 变量必须选择凭据')
    if (
      items.some(
        (item) =>
          ['Product', 'Service', 'Namespace', 'EnvName', 'ClusterName', 'Image'].includes(
            item.key
          ) || item.key.endsWith('-image')
      )
    )
      return ElMessage.warning('系统变量由平台提供，请删除同名自定义变量后保存')

    variableSaving.value = true
    try {
      if (props.environment.is_production) {
        try {
          await ElMessageBox.confirm(
            `将修改 ${props.environment.name} 的 ${variableChanges.value.length} 项共享变量，影响该环境引用这些变量的服务；保存后不自动部署。`,
            '核对生产环境变更',
            { confirmButtonText: '确认保存', cancelButtonText: '返回检查', type: 'warning' }
          )
        } catch {
          return
        }
      }
      if (
        generation !== variableGeneration ||
        environmentId !== props.environment?.id ||
        !props.visible
      )
        return
      const result = await updateEnvironmentVariableSetApi(set.id, {
        revision: set.revision,
        items
      })
      if (
        generation !== variableGeneration ||
        environmentId !== props.environment?.id ||
        !props.visible
      )
        return
      if (
        !result ||
        result.id !== set.id ||
        result.environment_id !== environmentId ||
        result.service_id !== 0 ||
        result.revision <= set.revision
      )
        throw Error('无法确认保存结果，请重新加载核对')
      ElMessage.success('当前环境共享变量已保存，重新提交部署后生效')
      previewResult.value = null
      await loadVariables()
    } catch (e: any) {
      if (generation === variableGeneration)
        variableError.value = e?.message || '保存失败，请重新加载核对'
    } finally {
      variableSaving.value = false
    }
  }

  const saveOverride = async () => {
    if (!props.environment || !selectedServiceId.value || !overrideForm.service_revision_id) return
    overrideSaving.value = true
    try {
      const result = await saveEnvironmentServiceOverrideApi(
        props.environment.id,
        selectedServiceId.value,
        {
          expected_revision: overrideForm.expected_revision,
          service_revision_id: overrideForm.service_revision_id,
          image_ref: overrideForm.image_ref.trim(),
          replicas: overrideForm.replicas ?? null
        }
      )
      applyOverride(result)
      previewRevisionId.value = result.service_revision_id
      previewResult.value = null
      ElMessage.success('服务覆盖已保存')
    } finally {
      overrideSaving.value = false
    }
  }

  const revisionLabel = (revision: AxiOpsServiceRevision) => {
    const commit = revision.source_commit ? ` · ${revision.source_commit.slice(0, 8)}` : ''
    return `Revision ${revision.revision_number}${commit}`
  }

  const generatePreview = async () => {
    if (!props.environment || !selectedServiceId.value || !previewRevisionId.value) return
    if (selectedService.value?.delivery_type === 'helm' && !previewReleaseName.value.trim())
      return ElMessage.warning('Helm Release 名称不能为空')
    previewLoading.value = true
    try {
      previewResult.value = await previewEnvironmentResourcesApi(props.environment.id, {
        service_id: selectedServiceId.value,
        service_revision_id: previewRevisionId.value,
        release_name:
          selectedService.value?.delivery_type === 'helm'
            ? previewReleaseName.value.trim()
            : undefined
      })
    } finally {
      previewLoading.value = false
    }
  }

  const actionLabel = (action: string) =>
    ({ create: '新建', update: '更新', delete: '删除', unchanged: '无变化' })[action] || action

  const actionTagType = (action: string) => {
    if (action === 'create') return 'success'
    if (action === 'update') return 'warning'
    if (action === 'delete') return 'danger'
    return 'info'
  }

  const flattenValues = (
    value: Record<string, unknown>,
    prefix = ''
  ): Array<{ key: string; value: string }> =>
    Object.entries(value).flatMap(([key, item]) => {
      const path = prefix ? `${prefix}.${key}` : key
      if (item && typeof item === 'object' && !Array.isArray(item)) {
        const record = item as Record<string, unknown>
        if (typeof record.credential_id === 'number')
          return [{ key: path, value: `凭据 #${record.credential_id}` }]
        return flattenValues(record, path)
      }
      return [{ key: path, value: typeof item === 'string' ? item : JSON.stringify(item) }]
    })

  watch(
    () => [props.visible, props.environment?.id],
    ([visible]) => {
      variableGeneration++
      variableSet.value = null
      variableItems.value = []
      savedVariableItems.value = []
      variableError.value = ''
      if (!visible) return
      activeTab.value = props.initialTab || 'variables'
      previewResult.value = null
      previewRevisionId.value = 0
      previewReleaseName.value = ''
      advancedPanels.value = []
      selectedServiceId.value = 0
      selectedServiceId.value = props.services[0]?.id || 0
      void Promise.all([loadVariables(), loadCredentials(), loadResources()])
    }
  )
  watch(
    selectedServiceId,
    () => {
      previewResult.value = null
      const service = props.services.find((item) => item.id === selectedServiceId.value)
      previewReleaseName.value = service?.delivery_type === 'helm' ? service.code : ''
      if (props.visible && selectedServiceId.value) void loadOverride()
    },
    { flush: 'sync' }
  )
</script>

<style scoped lang="scss">
  .variable-help {
    margin: 8px 0;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.7;
  }
  .variable-changes {
    margin-top: 18px;
    padding: 12px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
  }
  .variable-changes summary {
    cursor: pointer;
    margin-bottom: 8px;
    font-weight: 600;
  }
  .variable-changes small {
    display: block;
    margin-top: 8px;
    color: var(--el-text-color-secondary);
  }
  .config-panel {
    min-height: 360px;
  }

  .panel-heading {
    display: flex;
    gap: 16px;
    align-items: flex-start;
    justify-content: space-between;
    min-height: 58px;
    margin-bottom: 12px;
    border-bottom: 1px solid var(--art-border-color);
  }

  .panel-heading > span {
    display: flex;
    flex-direction: column;
  }

  .panel-heading small {
    margin-top: 5px;
    color: var(--art-text-gray-500);
  }

  .heading-actions {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .variable-header,
  .variable-row {
    display: grid;
    grid-template-columns: minmax(160px, 0.8fr) 120px minmax(220px, 1.4fr) 34px;
    gap: 8px;
    align-items: center;
  }

  .variable-header {
    padding: 0 0 8px;
    font-size: 12px;
    color: var(--art-text-gray-500);
  }

  .variable-row {
    margin-bottom: 8px;
  }

  .form-grid {
    display: grid;
    grid-template-columns: 1.5fr 1fr;
    gap: 14px;
  }

  .override-meta {
    display: flex;
    gap: 10px;
    align-items: center;
    font-size: 12px;
    color: var(--art-text-gray-500);
  }

  .override-note {
    margin-bottom: 14px;
  }

  .override-summary {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 10px;
    margin-bottom: 14px;
  }

  .override-summary > div {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 10px 12px;
    border: 1px solid var(--art-border-color);
    border-radius: 4px;
    background: var(--art-main-bg-color);
  }

  .override-summary span,
  .override-summary strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .override-summary span {
    color: var(--art-text-gray-500);
    font-size: 12px;
  }

  .override-summary strong {
    font-size: 13px;
  }

  .override-diff {
    margin-bottom: 16px;
    border-top: 1px solid var(--art-border-color);
    border-bottom: 1px solid var(--art-border-color);
  }

  .override-diff-heading {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 12px;
    padding: 10px 0 8px;
  }

  .override-diff-heading small {
    color: var(--art-text-gray-500);
    font-size: 12px;
  }

  .override-diff-row {
    display: grid;
    grid-template-columns: 100px minmax(0, 1fr) 20px minmax(0, 1fr);
    gap: 8px;
    align-items: center;
    min-height: 34px;
    border-top: 1px solid var(--art-border-color);
    font-size: 12px;
  }

  .override-diff-row > span:first-child {
    color: var(--art-text-gray-500);
  }

  .override-diff-row code {
    overflow: hidden;
    padding: 3px 5px;
    border-radius: 3px;
    background: var(--el-fill-color-light);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .override-diff-row code.is-overridden {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }

  .diff-arrow {
    color: var(--art-text-gray-500);
    text-align: center;
  }

  .baseline-collapse {
    margin-top: 18px;
    border-top: 1px solid var(--art-border-color);
  }

  .baseline-definition {
    max-height: 260px;
    overflow: auto;
    margin: 0;
    padding: 12px;
    border: 1px solid var(--art-border-color);
    border-radius: 4px;
    background: #0f172a;
    color: #dbeafe;
    font: 12px/1.6 var(--ax-font-mono);
    white-space: pre;
  }

  .field-hint {
    display: block;
    margin-top: 7px;
    color: var(--art-text-gray-500);
    font-size: 12px;
  }

  .panel-footer {
    display: flex;
    justify-content: flex-end;
    margin-top: 20px;
  }

  .preview-form-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 14px;
  }

  .diff-summary {
    display: flex;
    flex-wrap: wrap;
    gap: 18px;
    padding: 10px 0 14px;
    font-size: 13px;
    border-bottom: 1px solid var(--art-border-color);
  }

  .diff-summary span {
    display: inline-flex;
    gap: 5px;
    align-items: baseline;
  }

  .diff-summary strong {
    font-size: 19px;
  }

  .diff-create {
    color: var(--el-color-success);
  }

  .diff-update {
    color: var(--el-color-warning);
  }

  .diff-delete {
    color: var(--el-color-danger);
  }

  .diff-unchanged {
    color: var(--art-text-gray-500);
  }

  .preview-table {
    margin-top: 12px;
  }

  .values-preview {
    margin-top: 16px;
    border-top: 1px solid var(--art-border-color);
  }

  .values-title {
    padding: 12px 0 8px;
    font-weight: 600;
  }

  .value-row {
    display: grid;
    grid-template-columns: minmax(180px, 0.8fr) minmax(200px, 1.2fr);
    gap: 12px;
    padding: 7px 0;
    overflow-wrap: anywhere;
    border-bottom: 1px solid var(--art-border-color);
  }

  .value-row span {
    color: var(--art-text-gray-600);
  }

  @media (width <= 700px) {
    .variable-header {
      display: none;
    }

    .variable-row {
      grid-template-columns: 1fr 110px 34px;
    }

    .variable-row > :nth-child(3) {
      grid-row: 2;
      grid-column: 1 / 3;
    }

    .form-grid {
      grid-template-columns: 1fr;
    }

    .override-summary {
      grid-template-columns: 1fr;
    }

    .override-diff-row {
      grid-template-columns: 80px minmax(0, 1fr);
      padding: 6px 0;
    }

    .override-diff-row .diff-arrow {
      display: none;
    }

    .override-diff-row code:last-child {
      grid-column: 2;
    }

    .preview-form-grid,
    .value-row {
      grid-template-columns: 1fr;
    }
  }
</style>
