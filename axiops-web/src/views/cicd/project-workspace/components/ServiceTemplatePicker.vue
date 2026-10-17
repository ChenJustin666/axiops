<template>
  <ElDialog
    :model-value="visible"
    :title="target ? `更换模板 · ${target.name}` : `选择 ${kindLabel} 模板`"
    width="min(1160px, 94vw)"
    class="configured-service-dialog"
    append-to-body
    destroy-on-close
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    :show-close="false"
    @update:model-value="setVisible"
  >
    <template #header>
      <div class="page-heading">
        <div class="page-title">
          <span class="heading-icon"><Layers :size="20" /></span>
          <div
            ><span class="page-eyebrow"
              >服务与配置 / {{ stage === 'select' ? '模板库' : '环境配置' }}</span
            >
            <h2>{{
              stage === 'select' ? '选择服务模板' : target ? '配置服务环境' : '创建服务配置'
            }}</h2>
          </div>
        </div>
        <ElButton text :disabled="saving" aria-label="关闭模板配置" @click="setVisible(false)"
          ><X :size="20"
        /></ElButton>
      </div>
    </template>
    <section
      class="service-template-picker"
      :aria-label="target ? '为已有服务更换模板' : '从模板库创建服务'"
    >
      <div v-show="stage === 'select'" class="selection-content">
        <p class="selection-description"
          >选择一个模板开始配置。参数和完整 YAML 都可以按当前环境调整。</p
        >
        <div class="selection-context">
          <ElSelect
            v-if="kind === 'k8s_yaml'"
            v-model="environmentId"
            placeholder="选择目标环境"
            :disabled="saving"
            aria-label="创建配置的目标环境"
          >
            <ElOption
              v-for="env in environments"
              :key="env.id"
              :value="env.id"
              :label="env.name + (env.is_production ? ' · 生产' : '') + ' · ' + env.namespace"
            />
          </ElSelect>
          <ElSelect
            v-if="target && kind === 'k8s_yaml'"
            :model-value="referenceId"
            @update:model-value="changeReference"
            :disabled="saving"
            aria-label="参考配置来源"
          >
            <ElOption :value="0" label="选择模板库中的模板" />
            <ElOption
              v-for="env in environments?.filter((e) => e.id !== environmentId)"
              :key="env.id"
              :value="env.id"
              :label="'参考 ' + env.name + ' 的完整 YAML'"
            />
          </ElSelect>
        </div>
        <ElAlert
          v-if="kind === 'k8s_yaml' && !environments?.length"
          title="请先在环境页面创建环境并绑定集群和命名空间。"
          type="info"
          :closable="false"
        />
        <div v-if="!referenceId" class="picker-heading"
          ><span
            >可用模板 <small>{{ filtered.length }}</small></span
          ><ElButton text :disabled="saving" @click="load">刷新</ElButton></div
        >
        <ElInput
          v-if="!referenceId"
          v-model="search"
          clearable
          placeholder="搜索模板名称"
          :disabled="saving || !!createdId"
        />
        <ElAlert v-if="error" :title="error" type="error" :closable="false" />
        <div v-if="!referenceId" v-loading="loading" class="picker-list">
          <button
            v-for="item in filtered"
            :key="item.id"
            type="button"
            :class="['picker-item', { selected: selected?.id === item.id }]"
            :disabled="saving || !!createdId"
            @click="chooseTemplate(item)"
          >
            <span class="template-card-icon"><FileCode2 :size="22" /></span>
            <span class="template-card-title"
              ><strong>{{ item.name }}</strong
              ><small>v{{ item.revision }}</small></span
            >
            <p>{{ item.description || '从标准配置开始，按环境调整服务参数。' }}</p>
            <span class="template-card-footer"
              >{{ kindLabel
              }}<span
                >{{ selected?.id === item.id ? '已选择' : '选择模板' }}
                <ArrowRight :size="14" /></span
            ></span>
            <Check v-if="selected?.id === item.id" :size="16" class="selection-mark" />
          </button>
          <ElEmpty
            v-if="!loading && !filtered.length"
            :image-size="60"
            :description="
              error
                ? '模板读取失败，请刷新重试'
                : search
                  ? '没有匹配的模板'
                  : '暂无模板，请先在模板库创建'
            "
          />
        </div>
      </div>
      <ElAlert
        v-if="error && stage === 'configure'"
        :title="error"
        type="error"
        :closable="false"
        class="page-error"
      />
      <div v-if="selected || reference" v-show="stage === 'configure'" class="configuration-layout">
        <main class="configuration-main" v-loading="loading">
          <div class="configuration-context">
            <span>{{
              reference
                ? '参考环境：' + environments?.find((e) => e.id === referenceId)?.name
                : selected?.name + ' · v' + selected?.revision
            }}</span>
            <span v-if="kind === 'k8s_yaml'"
              >Namespace：{{ targetEnvironment?.namespace || '未绑定' }}</span
            >
          </div>
          <section class="configuration-section identity-section">
            <div class="section-heading"
              ><span class="section-number">01</span><div><h3>基本信息</h3></div></div
            >
            <div class="identity-grid">
              <div
                ><label class="service-name-label">{{ target ? '目标服务' : '新服务名称' }}</label>
                <ElInput
                  v-model="name"
                  placeholder="例如 risk_server"
                  :disabled="saving || !!createdId || !!target"
                  :title="SERVICE_NAME_HINT"
                  maxlength="128"
                />
              </div>
              <div v-if="kind === 'k8s_yaml'"
                ><label class="service-name-label">目标环境</label>
                <ElSelect v-model="environmentId" :disabled="saving" aria-label="配置页目标环境">
                  <ElOption
                    v-for="env in environments"
                    :key="env.id"
                    :value="env.id"
                    :label="env.name + (env.is_production ? ' · 生产' : '') + ' · ' + env.namespace"
                  />
                </ElSelect>
              </div>
            </div>
            <small class="picker-hint">{{
              target
                ? '仅保存所选环境的配置，其他服务和环境不会跟随修改。'
                : '复制模板的已保存版本；以后修改模板库不会自动改变当前配置。'
            }}</small>
            <ElButton
              v-if="conflictingService"
              text
              type="primary"
              :disabled="saving"
              @click="useExistingService"
              >使用已有服务「{{ conflictingService.name }}」更换模板</ElButton
            >
          </section>
          <section v-if="kind === 'k8s_yaml' && preparedDefinition" class="configuration-section">
            <div class="section-heading"
              ><span class="section-number">02</span><div><h3>模板参数</h3></div></div
            >
            <TemplateConfigurationForm
              ref="configurationForm"
              :definition="preparedDefinition"
              :parameters="parameters"
              v-model:values="parameterValues"
              :service-code="target?.code || serviceCodeFromName(name.trim())"
              :disabled="saving || loading"
              :reference-mode="!!reference"
              @editing-change="optionalEditing = $event"
              @configuration-change="previewCompletedOptions"
            />
          </section>
          <ElAlert
            v-if="reference"
            type="warning"
            :closable="false"
            title="参考完整 YAML，保留手写资源；配套配置和凭据不跨环境复制。请核对域名、Namespace、配置中心、PVC、Secret 引用。"
          />
          <div v-if="reference || targetEnvironment?.is_production" class="review-card">
            <ElCheckbox v-model="targetReviewed" :disabled="saving">
              已核对目标环境及上述差异，保存不会自动部署
            </ElCheckbox>
          </div>
          <details
            v-if="kind === 'k8s_yaml'"
            ref="previewElement"
            class="template-preview configuration-section"
            :open="previewOpen"
          >
            <summary
              >最终 YAML 预览 · {{ targetEnvironment?.name }} ·
              {{ targetEnvironment?.namespace }}</summary
            >
            <p v-if="previewManifest" class="yaml-description"
              >已通过后端渲染校验，包含所有已完成的可选配置。保存不会部署；Image
              为校验占位值，实际部署使用工作流镜像。</p
            >
            <ElInput
              v-if="previewManifest"
              :model-value="previewManifest"
              type="textarea"
              :rows="20"
              readonly
              spellcheck="false"
            />
            <p v-else class="yaml-description">{{
              previewing
                ? '正在生成当前环境的最终 YAML…'
                : '点击“预览完整配置”生成当前配置；添加或删除资源后会自动更新。预览不保存、不部署。'
            }}</p>
          </details>
          <details class="template-source configuration-section">
            <summary>{{ kind === 'k8s_yaml' ? '高级：编辑模板 YAML' : '查看模板内容' }}</summary>
            <p class="yaml-description"
              >此处为模板源码，用于特殊配置。填写参数和添加资源后的最终结果请查看上方“最终 YAML
              预览”。</p
            >
            <ElInput
              v-if="kind === 'k8s_yaml'"
              v-model="editableTemplateBody"
              type="textarea"
              :rows="22"
              :disabled="saving"
              spellcheck="false"
            />
            <pre v-else>{{ selected?.files.map((file) => file.content).join('\n---\n') }}</pre>
          </details>
        </main>
      </div>
    </section>
    <template #footer>
      <div class="picker-actions"
        ><span class="footer-note"
          ><span class="step-dot" />{{
            stage === 'select'
              ? '选择模板 → 配置参数 → 校验保存'
              : '保存前将校验配置 · 不会自动部署'
          }}</span
        >
        <ElButton :disabled="saving" @click="setVisible(false)">取消</ElButton>
        <ElButton
          v-if="stage === 'configure'"
          :disabled="saving || !!createdId"
          @click="stage = 'select'"
          >上一步</ElButton
        >
        <ElButton
          v-if="stage === 'select'"
          type="primary"
          :disabled="!canContinue"
          :loading="loading"
          @click="nextStep"
          >下一步</ElButton
        >
        <ElButton
          v-if="stage === 'configure' && kind === 'k8s_yaml'"
          :loading="previewing"
          :disabled="saving || loading || !name.trim() || !environmentId || optionalEditing"
          @click="create(true)"
          >预览完整配置</ElButton
        >
        <ElButton
          v-if="stage === 'configure'"
          type="primary"
          :loading="saving && !previewing"
          :disabled="
            (!selected && !reference) ||
            !name.trim() ||
            loading ||
            optionalEditing ||
            (kind === 'k8s_yaml' && !environmentId) ||
            ((reference || targetEnvironment?.is_production) && !targetReviewed)
          "
          @click="create()"
          >{{ target ? '保存当前环境' : createdId ? '重试保存配置' : '创建服务' }}</ElButton
        >
      </div>
    </template>
  </ElDialog>
</template>
<script setup lang="ts">
  import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { Check, Layers, FileCode2, ArrowRight, X } from 'lucide-vue-next'
  import {
    listDeliveryTemplatesApi,
    previewDeliveryTemplateApi,
    createServiceApi,
    createServiceRevisionApi,
    type DeliveryTemplate,
    createConfiguredServiceApi,
    getServiceEnvironmentConfigApi,
    saveServiceEnvironmentConfigApi,
    type AxiOpsEnvironment,
    type ServiceEnvironmentConfig,
    type SaveServiceEnvironmentConfig,
    type AxiOpsService
  } from '@/api/axiops'
  import TemplateConfigurationForm from './TemplateConfigurationForm.vue'
  import { configValues } from '@/utils/service-environment-config'
  import { emptyServiceAssets } from '@/api/axiops/service-assets'
  import {
    templateYAMLBody,
    updateTemplateYAMLBody,
    type TemplateParameter
  } from '@/utils/template-configuration'
  import { SERVICE_NAME_HINT, serviceCodeFromName, serviceNameError } from '@/utils/service-name'

  const props = defineProps<{
    visible: boolean
    projectId: number
    kind: 'k8s_yaml' | 'helm'
    environments?: AxiOpsEnvironment[]
    preferredEnvironmentId?: number
    existingCodes: string[]
    service?: AxiOpsService | null
    replacementCandidates?: AxiOpsService[]
  }>()
  const emit = defineEmits<{
    'update:visible': [value: boolean]
    saved: [serviceId: number, replaced?: boolean, environmentId?: number]
  }>()
  const target = ref<AxiOpsService | null>(null)
  const stage = ref<'select' | 'configure'>('select')
  const initialDraft = ref('')
  const optionalEditing = ref(false)
  const previewing = ref(false)
  const previewManifest = ref('')
  const previewOpen = ref(false)
  const previewElement = ref<HTMLElement>()
  const draftSignature = () =>
    JSON.stringify({
      name: name.value,
      definition: preparedDefinition.value,
      values: parameterValues.value,
      options: configurationForm.value?.getDraftOptions()
    })
  async function chooseTemplate(item: DeliveryTemplate) {
    if (saving.value || createdId.value) return
    if (selected.value?.id === item.id && !referenceId.value) {
      return
    }
    if (!(await confirmDiscard())) return
    referenceId.value = 0
    selected.value = item
  }
  const canContinue = computed(
    () =>
      !saving.value &&
      !loading.value &&
      !!preparedDefinition.value &&
      !!(reference.value || selected.value) &&
      (props.kind !== 'k8s_yaml' || !!targetEnvironment.value)
  )
  function nextStep() {
    if (canContinue.value) stage.value = 'configure'
  }
  async function confirmDiscard() {
    if (!initialDraft.value || draftSignature() === initialDraft.value) return true
    const version = generation
    try {
      await ElMessageBox.confirm('当前配置有未保存的修改，是否放弃？', '未保存的配置', {
        confirmButtonText: '放弃修改',
        cancelButtonText: '继续编辑',
        type: 'warning'
      })
      return version === generation && props.visible
    } catch {
      return false
    }
  }
  async function changeReference(value: number) {
    if (saving.value || value === referenceId.value || !(await confirmDiscard())) return
    referenceId.value = value
  }
  const conflictingService = computed(() =>
    !target.value && !createdId.value
      ? props.replacementCandidates?.find(
          (item) =>
            item.code === serviceCodeFromName(name.value.trim()) &&
            item.status === 'active' &&
            item.delivery_type === props.kind
        )
      : undefined
  )
  function useExistingService() {
    if (saving.value || !conflictingService.value) return
    target.value = conflictingService.value
    name.value = target.value.name
    error.value = ''
  }
  const kindLabel = computed(() => (props.kind === 'helm' ? 'Helm Chart' : 'K8s YAML'))
  const templates = ref<DeliveryTemplate[]>([]),
    selected = ref<DeliveryTemplate | null>(null)
  const name = ref(''),
    search = ref(''),
    error = ref('')
  const loading = ref(false),
    saving = ref(false),
    createdId = ref(0)
  const filtered = computed(() =>
    templates.value.filter((item) => item.name.toLowerCase().includes(search.value.toLowerCase()))
  )
  let generation = 0
  const preparedDefinition = ref('')
  const editableTemplateBody = computed({
    get: () => templateYAMLBody(preparedDefinition.value),
    set: (body: string) => {
      preparedDefinition.value = updateTemplateYAMLBody(preparedDefinition.value, body)
    }
  })
  async function previewCompletedOptions() {
    await nextTick()
    if (
      props.kind === 'k8s_yaml' &&
      props.visible &&
      stage.value === 'configure' &&
      name.value.trim() &&
      environmentId.value &&
      !optionalEditing.value
    )
      await create(true)
  }
  const environmentId = ref(0),
    referenceId = ref(0),
    targetReviewed = ref(false)
  const targetEnvironment = computed(() =>
    props.environments?.find((e) => e.id === environmentId.value)
  )
  const reference = ref<ServiceEnvironmentConfig | null>(null)
  const parameters = ref<TemplateParameter[]>([]),
    parameterValues = ref<Record<string, string>>({})
  const configurationForm = ref<InstanceType<typeof TemplateConfigurationForm>>()
  const configurationSignature = () =>
    JSON.stringify([draftSignature(), environmentId.value, referenceId.value, target.value?.id])
  watch(configurationSignature, () => {
    previewManifest.value = ''
    previewOpen.value = false
  })
  let preparation = 0
  async function prepare() {
    if (saving.value || !props.visible) return
    const request = ++preparation
    loading.value = true
    error.value = ''
    targetReviewed.value = false
    preparedDefinition.value = ''
    optionalEditing.value = false
    reference.value = null
    try {
      let definition = '',
        fields: TemplateParameter[] = []
      if (referenceId.value && target.value) {
        const source = await getServiceEnvironmentConfigApi(referenceId.value, target.value.id)
        if (request !== preparation) return
        if (source.service_id !== target.value.id || source.environment_id !== referenceId.value)
          throw new Error('参考配置与当前选择不一致，请重新加载')
        if (!source.bound) throw new Error('参考环境尚未配置此服务，请选择其他环境或模板')
        reference.value = source
        definition = source.definition
        const match = definition.match(/^# AxiOps-Variables: (.+)$/m)
        fields = match ? JSON.parse(match[1]) : []
        const actual = configValues(source)
        fields = fields.map((field) => ({
          ...field,
          value: actual[field.key]?.secret ? '' : (actual[field.key]?.value ?? field.value)
        }))
      } else if (selected.value) {
        definition = (await previewDeliveryTemplateApi(selected.value)).definition
        fields = selected.value.variables.map((field) => ({ ...field }))
      }
      if (request !== preparation) return
      parameters.value = fields
      parameterValues.value = Object.fromEntries(fields.map((field) => [field.key, field.value]))
      preparedDefinition.value = definition
      await nextTick()
      if (request === preparation) initialDraft.value = draftSignature()
    } catch (e: any) {
      if (request === preparation) error.value = e.message || '配置准备失败'
    } finally {
      if (request === preparation) loading.value = false
    }
  }
  watch(
    () => [selected.value?.id, referenceId.value],
    () => void prepare()
  )
  watch(environmentId, () => {
    targetReviewed.value = false
    if (referenceId.value && referenceId.value === environmentId.value) {
      referenceId.value = 0
      stage.value = 'select'
    }
  })

  async function setVisible(value: boolean) {
    if (saving.value) return
    if (!value && !(await confirmDiscard())) return
    if (!value) initialDraft.value = ''
    // If revision creation failed, leave the created service visible to the user.
    if (!value && createdId.value) emit('saved', createdId.value)
    emit('update:visible', value)
  }
  async function load() {
    if (saving.value || createdId.value) return
    if (initialDraft.value && draftSignature() !== initialDraft.value && !(await confirmDiscard()))
      return
    const version = ++generation
    loading.value = true
    error.value = ''
    selected.value = null
    try {
      const rows = await listDeliveryTemplatesApi(props.kind)
      if (version === generation)
        templates.value = (rows || []).filter((item) => item.kind === props.kind)
    } catch (e: any) {
      if (version === generation) {
        templates.value = []
        error.value = e.message || '模板读取失败'
      }
    } finally {
      if (version === generation) loading.value = false
    }
  }
  async function create(previewOnly = false) {
    if (saving.value || loading.value || (!selected.value && !reference.value) || !props.visible)
      return
    const problem = target.value ? '' : serviceNameError(name.value)
    if (problem) {
      error.value = problem
      return
    }
    const code = serviceCodeFromName(name.value.trim())
    if (!target.value && !createdId.value && props.existingCodes.includes(code)) {
      error.value = '服务名称已存在，请使用原服务更换模板，无需删除重建'
      return
    }
    if (
      target.value &&
      (target.value.project_id !== props.projectId ||
        target.value.delivery_type !== props.kind ||
        target.value.status !== 'active')
    ) {
      error.value = '目标服务不可用；已删除的服务请先恢复'
      return
    }
    saving.value = true
    previewing.value = previewOnly
    error.value = ''
    const version = generation
    const sourceSignature = configurationSignature()
    const template = selected.value
    const projectId = props.projectId
    const serviceName = name.value.trim()
    const replacing = target.value
    try {
      if (props.kind === 'k8s_yaml') {
        if (!environmentId.value || !preparedDefinition.value)
          throw new Error('请先选择目标环境和模板')
        if (
          !previewOnly &&
          (reference.value || targetEnvironment.value?.is_production) &&
          !targetReviewed.value
        )
          throw new Error('请核对目标环境差异')
        const configured = configurationForm.value?.getConfigured() || {
          definition: preparedDefinition.value,
          assets: emptyServiceAssets()
        }
        const targetConfig = replacing
          ? await getServiceEnvironmentConfigApi(environmentId.value, replacing.id)
          : null
        if (version !== generation) return
        if (
          targetConfig &&
          (targetConfig.service_id !== replacing?.id ||
            targetConfig.environment_id !== environmentId.value)
        )
          throw new Error('目标配置与当前选择不一致，请重新加载')
        const assets = targetConfig?.assets
          ? {
              ...targetConfig.assets,
              files: [...(targetConfig.assets.files || []), ...configured.assets.files],
              secrets: [...(targetConfig.assets.secrets || []), ...configured.assets.secrets],
              routes: [...(targetConfig.assets.routes || []), ...configured.assets.routes],
              volumes: [...(targetConfig.assets.volumes || []), ...configured.assets.volumes]
            }
          : configured.assets
        const data = {
          name: serviceName,
          code,
          definition: configured.definition,
          variables: { ...parameterValues.value },
          assets
        }
        const saveRequest: SaveServiceEnvironmentConfig | null = targetConfig
          ? {
              targets: [{ environment_id: environmentId.value, token: targetConfig.token }],
              revision_id: 0,
              definition: data.definition,
              variables: data.variables,
              assets,
              ...(reference.value
                ? {
                    source: {
                      environment_id: reference.value.environment_id,
                      token: reference.value.token
                    }
                  }
                : {})
            }
          : null
        // Source + assets is intentionally rejected by the API. No source
        // resources/credentials are copied; retain target assets instead.
        if (saveRequest?.source) {
          if (
            configured.assets.files.length ||
            configured.assets.routes.length ||
            configured.assets.volumes.length ||
            configured.assets.secrets.length
          )
            throw new Error(
              '参考已有环境时请先保存 YAML；配套配置请在目标环境单独设置，避免复制域名、存储或凭据'
            )
          delete saveRequest.assets
        }
        const preview = replacing
          ? await saveServiceEnvironmentConfigApi(replacing.id, saveRequest!, true)
          : await createConfiguredServiceApi(environmentId.value, data, true)
        if (version !== generation) return
        if (!preview.validated || !preview.configs?.[0]?.manifest)
          throw new Error('后端未返回完整配置校验结果')
        if (previewOnly) {
          if (sourceSignature !== configurationSignature()) return
          previewManifest.value = templateYAMLBody(preview.configs[0].manifest)
          previewOpen.value = true
          await nextTick()
          previewElement.value?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
          return
        }
        await ElMessageBox.confirm(
          '目标：' +
            targetEnvironment.value?.name +
            ' · ' +
            targetEnvironment.value?.namespace +
            '。配置校验通过，保存后不会自动部署。',
          replacing ? '保存当前环境配置' : '创建服务配置',
          {
            confirmButtonText: '确认保存',
            cancelButtonText: '返回修改',
            type: targetEnvironment.value?.is_production ? 'warning' : 'info'
          }
        )
        if (version !== generation) return
        const result = replacing
          ? await saveServiceEnvironmentConfigApi(replacing.id, saveRequest!)
          : await createConfiguredServiceApi(environmentId.value, data)
        ElMessage.success('当前环境配置已保存，待部署')
        initialDraft.value = ''
        emit('saved', replacing?.id || (result as any).service_id, !!replacing, environmentId.value)
        emit('update:visible', false)
        return
      }
      if (!template) return
      if (replacing) {
        const snapshot = await previewDeliveryTemplateApi(template)
        if (version !== generation || !props.visible) return
        await createServiceRevisionApi(replacing.id, {
          template:
            props.kind === 'helm' ? { id: template.id, revision: template.revision } : undefined,
          source_commit:
            props.kind === 'helm'
              ? 'template-copy'
              : `template:${template.id}@${template.revision}`,
          definition: snapshot.definition,
          expected_revision_id: replacing.current_revision_id || 0
        })
        if (version !== generation) return
        ElMessage.success('服务 Chart 已保存；到目标环境预览并保存后再发布')
        initialDraft.value = ''
        emit('saved', replacing.id, true)
        emit('update:visible', false)
        return
      }
      if (!createdId.value) {
        preparedDefinition.value = (await previewDeliveryTemplateApi(template)).definition
        if (version !== generation) return
        const created = await createServiceApi(projectId, {
          name: serviceName,
          code,
          source_type: 'template',
          description: '',
          repo_url: '',
          repo_branch: 'main',
          language: '',
          framework: '',
          image_name: ''
        })
        createdId.value = created.id
      }
      await createServiceRevisionApi(createdId.value, {
        template:
          props.kind === 'helm' ? { id: template.id, revision: template.revision } : undefined,
        source_commit:
          props.kind === 'helm' ? 'template-copy' : `template:${template.id}@${template.revision}`,
        definition: preparedDefinition.value
      })
      ElMessage.success('已使用模板创建服务')
      initialDraft.value = ''
      emit('saved', createdId.value)
      emit('update:visible', false)
    } catch (e: any) {
      if (e === 'cancel' || e === 'close' || e?.message === 'cancel') return
      error.value =
        (createdId.value ? '服务已创建，配置保存失败，可重试。' : '') + (e.message || '创建失败')
    } finally {
      saving.value = false
      previewing.value = false
    }
  }
  watch(
    () => [props.visible, props.projectId, props.kind, props.service?.id],
    () => {
      generation++
      preparation++
      if (!props.visible) return
      templates.value = []
      stage.value = 'select'
      initialDraft.value = ''
      optionalEditing.value = false
      previewManifest.value = ''
      previewOpen.value = false
      selected.value = null
      target.value = props.service || null
      name.value = target.value?.name || ''
      search.value = ''
      error.value = ''
      createdId.value = 0
      preparedDefinition.value = ''
      referenceId.value = 0
      reference.value = null
      parameters.value = []
      parameterValues.value = {}
      environmentId.value = props.environments?.some((e) => e.id === props.preferredEnvironmentId)
        ? props.preferredEnvironmentId!
        : props.environments?.[0]?.id || 0
      void load()
    },
    { immediate: true }
  )
  onBeforeUnmount(() => {
    generation++
    preparation++
  })
</script>

<style scoped>
  :global(.configured-service-dialog) {
    display: flex;
    flex-direction: column;
    margin: 4vh auto;
    height: 92dvh;
    max-height: 92dvh;
    padding: 0;
    border-radius: 12px !important;
    overflow: hidden;
  }
  :global(.configured-service-dialog .el-dialog__header) {
    margin: 0;
    padding: 24px 32px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  :global(.configured-service-dialog .el-dialog__body) {
    padding: 24px 32px !important;
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    overscroll-behavior: contain;
  }
  :global(.configured-service-dialog .el-dialog__footer) {
    padding: 18px 32px;
    border-top: 1px solid var(--el-border-color-lighter);
  }
  .page-heading,
  .page-title {
    display: flex;
    align-items: center;
    gap: 14px;
  }
  .page-heading {
    justify-content: space-between;
    max-width: 1440px;
    margin: auto;
  }
  .heading-icon,
  .template-card-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 42px;
    height: 42px;
    border-radius: 10px;
    background: var(--el-color-primary-light-9);
    color: var(--el-color-primary);
    flex-shrink: 0;
  }
  .page-eyebrow {
    display: block;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    margin-bottom: 5px;
  }
  h2,
  h3,
  p {
    margin: 0;
  }
  .page-title h2 {
    font-size: 20px;
    font-weight: 600;
    line-height: 1.4;
  }
  .selection-description {
    color: var(--el-text-color-secondary);
    line-height: 1.7;
    margin-bottom: 24px;
  }
  .selection-context {
    display: flex;
    gap: 16px;
    margin-bottom: 20px;
  }
  .selection-context > * {
    flex: 1;
  }
  .picker-heading {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin: 20px 0 8px;
    font-weight: 600;
  }
  .picker-heading small {
    color: var(--el-text-color-secondary);
    margin-left: 8px;
    font-weight: 400;
  }
  .picker-list {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
    padding: 20px 0 4px;
    min-height: 100px;
  }
  .picker-list :deep(.el-empty) {
    grid-column: 1 / -1;
  }
  .picker-item {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    min-height: 210px;
    padding: 24px;
    text-align: left;
    border: 1px solid var(--el-border-color);
    border-radius: 10px;
    background: var(--el-bg-color);
    color: var(--el-text-color-primary);
    cursor: pointer;
    transition:
      border-color 0.15s,
      box-shadow 0.15s;
  }
  .picker-item:hover,
  .picker-item:focus-visible {
    border-color: var(--el-color-primary);
    box-shadow: 0 4px 16px rgb(0 0 0 / 5%);
  }
  .picker-item.selected {
    border-color: var(--el-color-primary);
  }
  .template-card-title {
    display: flex;
    align-items: baseline;
    gap: 8px;
    margin: 16px 0 8px;
  }
  .template-card-title strong {
    font-size: 15px;
  }
  .template-card-title small {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
  .picker-item p {
    font-size: 13px;
    color: var(--el-text-color-secondary);
    line-height: 1.7;
    margin-bottom: 24px;
  }
  .template-card-footer {
    display: flex;
    width: 100%;
    margin-top: auto;
    justify-content: space-between;
    align-items: center;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
  .template-card-footer > span {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    color: var(--el-color-primary);
  }
  .selection-mark {
    position: absolute;
    top: 24px;
    right: 24px;
    color: var(--el-color-primary);
  }
  .configuration-context {
    display: flex;
    flex-wrap: wrap;
    gap: 8px 24px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    overflow-wrap: anywhere;
  }
  .configuration-layout {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    max-width: 1440px;
    margin: auto;
    align-items: start;
  }
  .configuration-main {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .configuration-section,
  .review-card {
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 10px;
    padding: 20px 24px;
  }
  .section-heading {
    display: flex;
    gap: 12px;
    margin-bottom: 18px;
  }
  .section-number {
    display: flex;
    justify-content: center;
    align-items: center;
    width: 30px;
    height: 30px;
    flex-shrink: 0;
    border-radius: 8px;
    background: var(--el-fill-color-light);
    color: var(--el-text-color-secondary);
    font-size: 12px;
    font-weight: 600;
  }
  .section-heading h3 {
    font-size: 15px;
    font-weight: 600;
    line-height: 1.6;
  }
  .identity-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 24px;
  }
  .service-name-label {
    display: block;
    font-size: 13px;
    color: var(--el-text-color-regular);
    margin-bottom: 10px;
  }
  .identity-grid .el-select {
    width: 100%;
  }
  .picker-hint {
    display: block;
    margin-top: 14px;
    color: var(--el-text-color-secondary);
    line-height: 1.7;
    font-size: 12px;
  }
  .review-card {
    padding: 24px;
  }
  .review-card {
    border-color: var(--el-color-warning-light-5);
  }
  .review-card :deep(.el-checkbox) {
    height: auto;
    align-items: flex-start;
    white-space: normal;
  }
  .review-card :deep(.el-checkbox__label) {
    font-size: 12px;
    line-height: 1.7;
  }
  .review-card :deep(.el-checkbox__input) {
    margin-top: 3px;
  }
  .template-preview summary,
  .template-source summary {
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
    cursor: pointer;
  }
  .yaml-description {
    margin: 18px 0 16px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    line-height: 1.7;
  }
  .template-preview :deep(textarea),
  .template-source :deep(textarea),
  .template-source pre {
    font: 13px/1.8 var(--ax-font-mono);
    padding: 20px;
    background: var(--el-fill-color-lighter);
    tab-size: 2;
  }
  .template-source pre {
    max-height: 600px;
    overflow: auto;
    border-radius: 8px;
  }
  .page-error {
    max-width: 1440px;
    margin: 0 auto 24px;
  }
  .picker-actions {
    display: flex;
    align-items: center;
    gap: 12px;
    max-width: 1440px;
    margin: auto;
  }
  .picker-actions :deep(.el-button + .el-button) {
    margin-left: 0;
  }
  .footer-note {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    margin-right: auto;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
  .step-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--el-color-primary);
  }
  @media (max-width: 760px) {
    :global(.configured-service-dialog .el-dialog__header) {
      padding: 18px 20px;
    }
    :global(.configured-service-dialog .el-dialog__body) {
      padding: 20px 16px !important;
    }
    :global(.configured-service-dialog .el-dialog__footer) {
      padding: 14px 16px;
    }
    .page-title h2 {
      font-size: 17px;
    }
    .configuration-layout,
    .identity-grid,
    .picker-list {
      grid-template-columns: 1fr;
    }
    .configuration-section {
      padding: 20px;
    }
    .selection-context {
      flex-direction: column;
    }
    .footer-note {
      display: none;
    }
    .picker-actions {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .picker-actions :deep(.el-button) {
      width: 100%;
    }
    .picker-item {
      min-height: 190px;
    }
  }
</style>
