<template>
  <section class="template-configuration-form">
    <ElForm label-position="top" :disabled="disabled">
      <div class="parameter-grid">
        <ElFormItem
          v-for="parameter in primary"
          :key="parameter.key"
          :label="parameter.label || parameterLabels[parameter.key] || parameter.key"
          :title="parameter.description"
        >
          <ElInput
            :model-value="values[parameter.key]"
            @update:model-value="setParameter(parameter.key, $event)"
            :placeholder="parameter.description || parameter.key"
          />
        </ElFormItem>
      </div>
      <details
        v-for="section in parameterSections"
        :key="section.key"
        class="folded-fields"
        :class="'parameters-' + section.key"
      >
        <summary
          >{{ section.label
          }}<template v-if="section.key !== 'health'">
            · {{ section.parameters.length }}</template
          ></summary
        >
        <div class="parameter-grid">
          <ElFormItem
            v-for="parameter in section.parameters"
            :key="parameter.key"
            :label="parameter.label || parameterLabels[parameter.key] || parameter.key"
            :title="parameter.description"
            v-show="showParameter(parameter)"
          >
            <ElSelect
              v-if="parameter.key === 'healthCheckType'"
              :model-value="values[parameter.key]"
              @update:model-value="setParameter(parameter.key, $event)"
            >
              <ElOption value="none" label="不生成探针" /><ElOption
                value="tcp"
                label="TCP · 检查端口"
              /><ElOption value="http" label="HTTP · 检查健康接口" />
            </ElSelect>
            <ElSwitch
              v-else-if="probeToggleKeys.includes(parameter.key)"
              :model-value="values[parameter.key] !== 'false'"
              @update:model-value="setParameter(parameter.key, String($event))"
            />
            <ElInputNumber
              v-else-if="healthNumberBounds[parameter.key]"
              :model-value="Number(values[parameter.key])"
              :min="healthNumberBounds[parameter.key][0]"
              :max="healthNumberBounds[parameter.key][1]"
              :step="1"
              :precision="0"
              controls-position="right"
              @update:model-value="setParameter(parameter.key, String($event ?? ''))"
            />
            <ElInput
              v-else
              :model-value="values[parameter.key]"
              @update:model-value="setParameter(parameter.key, $event)"
            />
          </ElFormItem>
        </div>
        <p v-if="section.key === 'health' && hasHealthType" class="optional-description"
          >可以不生成探针。TCP 只检查端口，HTTP 检查实际健康接口；端口 0 跟随应用端口。
          三种检查分别选择；存活检查请确认可靠后启用。原生 YAML 中已有探针优先，需直接在 YAML
          中修改或删除。</p
        >
      </details>
      <details
        v-if="supportsTemplateOptions(definition)"
        class="folded-fields optional-configuration"
      >
        <summary
          >可选资源与健康检查
          <span>{{ enabledCount ? '已启用 ' + enabledCount + ' 项' : '按需展开' }}</span></summary
        >
        <p class="optional-description"
          >路由、存储、配置引用、伸缩、探针与监控；模板已有资源保留。</p
        >
        <div v-if="completedCapabilities.length" class="completed-options">
          <div v-for="item in completedCapabilities" :key="item.id" class="completed-option">
            <div
              ><strong>{{ item.label }}</strong
              ><span>{{ optionSummary(item.key, item.options) }}</span></div
            >
            <ElButton
              text
              type="primary"
              :disabled="disabled || !!activeOption"
              @click="beginOption(item.key, item.id)"
              >编辑</ElButton
            >
            <ElButton
              text
              :disabled="disabled || !!activeOption"
              @click="removeOption(item.key, item.id)"
              >删除</ElButton
            >
          </div>
        </div>
        <div class="option-checks">
          <ElButton
            v-for="item in availableCapabilities"
            :key="item.key"
            :disabled="disabled || !!activeOption"
            @click="beginOption(item.key, 0)"
            >+ {{ item.label }}</ElButton
          >
        </div>
        <div v-if="activeOption === 'ingress'" class="option-section">
          <strong>路由 Ingress</strong
          ><div class="parameter-grid">
            <ElFormItem label="域名" required
              ><ElInput v-model="optionDraft.host" placeholder="例如 api.example.com"
            /></ElFormItem>
            <ElFormItem label="路径" required><ElInput v-model="optionDraft.path" /></ElFormItem>
          </div>
          <p class="optional-description"
            >后端自动使用当前模板的 Service 名称与 Service 端口，可在预览中确认。</p
          >
          <IngressRouteFields v-model="ingressDraft" />
        </div>
        <div v-if="activeOption === 'pvc'" class="option-section">
          <strong>持久存储 PVC</strong
          ><div class="parameter-grid">
            <ElFormItem label="已有 PVC 名称（留空创建）"
              ><ElInput v-model="optionDraft.existingClaim"
            /></ElFormItem>
            <ElFormItem label="挂载目录" required
              ><ElInput v-model="optionDraft.mountPath"
            /></ElFormItem>
            <ElFormItem v-if="!optionDraft.existingClaim" label="容量" required
              ><ElInput v-model="optionDraft.capacity" placeholder="例如 10Gi；至少 1Mi"
            /></ElFormItem>
            <ElFormItem v-if="!optionDraft.existingClaim" label="Storage Class"
              ><ElInput
                v-model="optionDraft.storageClass"
                placeholder="留空使用集群默认 StorageClass"
            /></ElFormItem>
            <ElFormItem v-if="!optionDraft.existingClaim" label="访问模式">
              <ElSelect v-model="optionDraft.accessMode">
                <ElOption value="ReadWriteOnce" label="RWO · 单节点读写" />
                <ElOption value="ReadWriteMany" label="RWX · 多节点读写" />
                <ElOption value="ReadOnlyMany" label="ROX · 多节点只读" />
                <ElOption value="ReadWriteOncePod" label="RWOP · 单 Pod 读写（需 CSI 支持）" />
              </ElSelect>
            </ElFormItem>
            <ElFormItem label="只读挂载"><ElSwitch v-model="optionDraft.readOnly" /></ElFormItem>
          </div>
          <p class="optional-description"
            >多副本跨节点共享存储通常需要 RWX 和支持该模式的存储类；RWO 不保证跨节点同时挂载。</p
          >
        </div>
        <div v-if="activeOption === 'secret'" class="option-section">
          <strong>Secret 引用</strong
          ><p>引用目标环境已有 Secret；平台凭据可在创建后通过“配套配置”安全创建和绑定。</p>
          <div class="parameter-grid">
            <ElFormItem label="Secret 名称" required
              ><ElInput v-model="optionDraft.secretName"
            /></ElFormItem>
            <ElFormItem label="Secret key" required
              ><ElInput v-model="optionDraft.secretKey"
            /></ElFormItem>
            <ElFormItem label="使用方式"
              ><ElSelect v-model="optionDraft.secretUsage"
                ><ElOption value="env" label="环境变量" /><ElOption
                  value="file"
                  label="文件挂载" /></ElSelect
            ></ElFormItem>
            <ElFormItem v-if="optionDraft.secretUsage !== 'file'" label="注入 Env 名称" required
              ><ElInput v-model="optionDraft.secretEnv"
            /></ElFormItem>
            <ElFormItem v-else label="容器内文件完整路径" required
              ><ElInput v-model="optionDraft.secretMount" placeholder="/app/secrets/token"
            /></ElFormItem>
          </div>
        </div>
        <div v-if="activeOption === 'configmap'" class="option-section">
          <strong>配置文件 ConfigMap</strong
          ><div class="parameter-grid">
            <ElFormItem label="已有 ConfigMap（留空创建）"
              ><ElInput v-model="optionDraft.configName"
            /></ElFormItem>
            <ElFormItem label="文件名" required
              ><ElInput v-model="optionDraft.filename"
            /></ElFormItem>
            <ElFormItem label="挂载文件完整路径" required
              ><ElInput v-model="optionDraft.configMount"
            /></ElFormItem>
          </div>
          <ElFormItem v-if="!optionDraft.configName" label="配置内容（密码请用 Secret）"
            ><ElInput v-model="optionDraft.content" type="textarea" :rows="4"
          /></ElFormItem>
        </div>
        <div v-if="activeOption === 'hpa'" class="option-section">
          <strong>自动伸缩 HPA</strong
          ><div class="parameter-grid">
            <ElFormItem label="最小副本"
              ><ElInputNumber v-model="optionDraft.minReplicas" :min="1"
            /></ElFormItem>
            <ElFormItem label="最大副本"
              ><ElInputNumber v-model="optionDraft.maxReplicas" :min="optionDraft.minReplicas"
            /></ElFormItem>
            <ElFormItem label="CPU 目标利用率 (%)"
              ><ElInputNumber v-model="optionDraft.cpuPercent" :min="1" :max="1000"
            /></ElFormItem>
          </div>
        </div>
        <div v-if="activeOption === 'probe'" class="option-section">
          <ElCheckbox v-model="optionDraft.probeReadinessEnabled"
            >就绪检查 · 失败停止接流量</ElCheckbox
          >
          <ElCheckbox v-model="optionDraft.probeStartupEnabled">启动检查 · 持续失败重启</ElCheckbox>
          <ElCheckbox v-model="optionDraft.probeLivenessEnabled"
            >存活检查 · 持续失败重启</ElCheckbox
          >
          <strong>健康探针</strong
          ><div class="parameter-grid">
            <ElFormItem label="类型"
              ><ElSelect v-model="optionDraft.probeType"
                ><ElOption value="tcp" label="TCP" /><ElOption
                  value="http"
                  label="HTTP" /></ElSelect
            ></ElFormItem>
            <ElFormItem label="端口（留空使用服务端口）"
              ><ElInput v-model="optionDraft.probePort"
            /></ElFormItem>
            <ElFormItem v-if="optionDraft.probeType === 'http'" label="HTTP 路径" required
              ><ElInput v-model="optionDraft.probePath"
            /></ElFormItem>
          </div>
        </div>
        <div v-if="activeOption === 'monitor'" class="option-section">
          <strong>Prometheus 监控注解</strong
          ><div class="parameter-grid">
            <ElFormItem label="指标路径"><ElInput v-model="optionDraft.metricsPath" /></ElFormItem>
            <ElFormItem label="端口（留空使用服务端口）"
              ><ElInput v-model="optionDraft.metricsPort"
            /></ElFormItem> </div
          ><p>集群需已配置按 Pod 注解采集指标。</p>
        </div>
        <ElAlert
          v-if="optionError"
          :title="optionError"
          type="error"
          :closable="false"
          class="option-error"
        />
        <div v-if="activeOption" class="option-actions">
          <span>完成当前项后，可继续添加其他配置。</span>
          <ElButton :disabled="disabled" @click="cancelOption">取消本项</ElButton>
          <ElButton type="primary" :disabled="disabled" @click="completeOption">{{
            editingId ? '完成修改' : '完成添加'
          }}</ElButton>
        </div>
      </details>
    </ElForm>
  </section>
</template>
<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue'
  import IngressRouteFields from './IngressRouteFields.vue'
  import type { ServiceAssets } from '@/api/axiops/service-assets'
  import {
    configureTemplateSelections,
    emptyTemplateOptions,
    parameterLabels,
    supportsTemplateOptions,
    templateHasProbes,
    templateParameterSection,
    type TemplateParameter,
    type TemplateSelection,
    type TemplateOptions
  } from '@/utils/template-configuration'
  const props = defineProps<{
    definition: string
    parameters: TemplateParameter[]
    values: Record<string, string>
    serviceCode: string
    disabled?: boolean
    referenceMode?: boolean
  }>()
  const options = reactive(emptyTemplateOptions())
  const emit = defineEmits<{
    'update:values': [values: Record<string, string>]
    'editing-change': [editing: boolean]
    'configuration-change': []
  }>()
  const probeToggleKeys = ['probeReadinessEnabled', 'probeStartupEnabled', 'probeLivenessEnabled']
  const setParameter = (key: string, value: string) => {
    const next = { ...props.values, [key]: value }
    if (
      key === 'healthCheckType' &&
      value !== 'none' &&
      (!props.values.healthCheckType || props.values.healthCheckType === 'none')
    ) {
      for (const flag of probeToggleKeys)
        next[flag] ??= flag === 'probeLivenessEnabled' ? 'false' : 'true'
    }
    emit('update:values', next)
  }
  const primary = computed(() =>
    props.parameters.filter((p) => templateParameterSection(p) === 'primary')
  )
  const hasHealthType = computed(() => props.parameters.some((p) => p.key === 'healthCheckType'))
  const healthNumberBounds: Record<string, [number, number]> = {
    healthPort: [0, 65535],
    probePeriodSeconds: [1, 3600],
    probeTimeoutSeconds: [1, 60],
    probeFailureThreshold: [1, 100],
    startupFailureThreshold: [1, 1000]
  }
  function showParameter(parameter: TemplateParameter) {
    if (!hasHealthType.value || parameter.key === 'healthCheckType') return true
    if (/^(health|probe|startup)/i.test(parameter.key)) {
      if (!props.values.healthCheckType || props.values.healthCheckType === 'none') return false
      if (parameter.key === 'healthPath' && props.values.healthCheckType !== 'http') return false
    }
    return true
  }
  const parameterSections = computed(() =>
    [
      { key: 'env', label: 'Env / JVM' },
      {
        key: 'health',
        label: props.parameters.some(
          (p) => /^(metrics|prometheus|monitor)/i.test(p.key) || p.group === 'monitor'
        )
          ? '模板健康检查与监控'
          : '模板健康检查'
      },
      { key: 'advanced', label: '其他模板参数' }
    ]
      .map((section) => ({
        ...section,
        parameters: [
          ...props.parameters,
          ...(hasHealthType.value
            ? probeToggleKeys
                .filter((key) => !props.parameters.some((p) => p.key === key))
                .map((key) => ({ key, value: 'true' }))
            : [])
        ].filter((p) => templateParameterSection(p) === section.key)
      }))
      .filter((section) => section.parameters.length)
  )
  type CapabilityKey = 'ingress' | 'pvc' | 'secret' | 'configmap' | 'hpa' | 'probe' | 'monitor'
  const activeOption = ref<CapabilityKey | ''>('')
  const optionDraft = reactive(emptyTemplateOptions())
  const optionError = ref('')
  const selections = ref<TemplateSelection[]>([])
  const editingId = ref(0)
  let selectionSequence = 0
  const repeatable = new Set<CapabilityKey>(['ingress', 'pvc', 'secret', 'configmap'])
  watch(activeOption, (key) => emit('editing-change', !!key), { immediate: true })
  function beginOption(key: CapabilityKey, id?: number) {
    if (
      props.disabled ||
      activeOption.value ||
      !visibleCapabilities.value.some((item) => item.key === key)
    )
      return
    const existing =
      id === 0
        ? undefined
        : selections.value.find((item) => item.key === key && (id === undefined || item.id === id))
    if (!existing && !repeatable.has(key) && selections.value.some((item) => item.key === key))
      return
    editingId.value = existing?.id || 0
    Object.assign(
      optionDraft,
      JSON.parse(JSON.stringify(existing?.options || emptyTemplateOptions()))
    )
    activeOption.value = key
    optionError.value = ''
  }
  function cancelOption() {
    if (props.disabled) return
    activeOption.value = ''
    editingId.value = 0
    optionError.value = ''
  }
  const ingressDraft = computed<ServiceAssets['routes'][number]>({
    get: () => ({
      name: 'route',
      host: optionDraft.host,
      path: optionDraft.path,
      service: '',
      port: 0,
      class: optionDraft.ingressClass,
      path_type: optionDraft.pathType,
      tls_enabled: optionDraft.tlsEnabled,
      tls_hosts: optionDraft.tlsHosts,
      tls_secret: optionDraft.tlsSecret,
      annotations: optionDraft.ingressAnnotations,
      labels: optionDraft.ingressLabels
    }),
    set: (route) =>
      Object.assign(optionDraft, {
        ingressClass: route.class,
        pathType: route.path_type,
        tlsEnabled: route.tls_enabled,
        tlsHosts: route.tls_hosts,
        tlsSecret: route.tls_secret,
        ingressAnnotations: route.annotations,
        ingressLabels: route.labels
      })
  })
  function completeOption() {
    if (props.disabled || !activeOption.value) return
    const entry = {
      id: editingId.value || selectionSequence + 1,
      key: activeOption.value,
      options: JSON.parse(JSON.stringify({ ...optionDraft, [activeOption.value]: true }))
    }
    const candidate = editingId.value
      ? selections.value.map((item) => (item.id === editingId.value ? entry : item))
      : [...selections.value, entry]
    try {
      configureTemplateSelections(props.definition, candidate, props.serviceCode, props.values)
      selections.value = candidate
      selectionSequence = Math.max(selectionSequence, entry.id)
      refreshOptions()
      activeOption.value = ''
      editingId.value = 0
      optionError.value = ''
      emit('configuration-change')
    } catch (e: any) {
      optionError.value = e.message || '请检查当前配置'
    }
  }
  function refreshOptions() {
    Object.assign(options, emptyTemplateOptions())
    for (const item of selections.value) {
      Object.assign(options, item.options)
    }
    for (const item of capabilities)
      options[item.key] = selections.value.some((selected) => selected.key === item.key)
  }
  function removeOption(key: CapabilityKey, id?: number) {
    if (props.disabled || activeOption.value) return
    const selected = selections.value.find(
      (item) => item.key === key && (id === undefined || item.id === id)
    )
    selections.value = selections.value.filter((item) => item !== selected)
    refreshOptions()
    emit('configuration-change')
  }
  function optionSummary(key: CapabilityKey, options: TemplateOptions) {
    switch (key) {
      case 'ingress':
        return options.host + options.path
      case 'pvc':
        return (options.existingClaim || options.capacity) + ' → ' + options.mountPath
      case 'secret':
        return (
          options.secretName +
          '/' +
          options.secretKey +
          ' → ' +
          (options.secretUsage === 'file' ? options.secretMount : options.secretEnv)
        )
      case 'configmap':
        return options.filename + ' → ' + options.configMount
      case 'hpa':
        return (
          options.minReplicas +
          '–' +
          options.maxReplicas +
          ' 副本 · CPU ' +
          options.cpuPercent +
          '%'
        )
      case 'probe':
        return (
          options.probeType.toUpperCase() +
          ' · ' +
          (options.probePort || '服务端口') +
          (options.probeType === 'http' ? ' · ' + options.probePath : '')
        )
      case 'monitor':
        return options.metricsPath + ' · ' + (options.metricsPort || '服务端口')
    }
  }
  const capabilities: Array<{
    key: 'ingress' | 'pvc' | 'secret' | 'configmap' | 'hpa' | 'probe' | 'monitor'
    label: string
  }> = [
    { key: 'ingress', label: '路由 Ingress' },
    { key: 'pvc', label: '存储 PVC' },
    { key: 'secret', label: 'Secret' },
    { key: 'configmap', label: 'ConfigMap' },
    { key: 'hpa', label: 'HPA' },
    { key: 'probe', label: '探针' },
    { key: 'monitor', label: '监控注解' }
  ]
  const visibleCapabilities = computed(() =>
    capabilities.filter(
      (item) =>
        (!props.referenceMode || ['hpa', 'probe', 'monitor'].includes(item.key)) &&
        (item.key !== 'probe' || (!hasHealthType.value && !templateHasProbes(props.definition)))
    )
  )
  const completedCapabilities = computed(() =>
    selections.value.map((item) => ({
      ...item,
      label: capabilities.find((capability) => capability.key === item.key)!.label
    }))
  )
  const availableCapabilities = computed(() =>
    visibleCapabilities.value.filter(
      (item) =>
        repeatable.has(item.key) || !selections.value.some((selected) => selected.key === item.key)
    )
  )
  const enabledCount = computed(() => completedCapabilities.value.length)
  defineExpose({
    getDraftOptions: () => ({
      ...options,
      selections: selections.value,
      activeOption: activeOption.value,
      ...(activeOption.value ? { pending: { ...optionDraft } } : {})
    }),
    getConfigured: () => {
      if (activeOption.value) throw new Error('请先完成或取消当前可选配置，再预览和保存')
      return configureTemplateSelections(
        props.definition,
        selections.value,
        props.serviceCode,
        props.values
      )
    }
  })
</script>
<style scoped>
  .parameter-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px 24px;
  }
  .folded-fields {
    margin: 8px 0 12px;
    padding: 16px 20px;
    border-radius: 8px;
    background: var(--el-fill-color-lighter);
  }
  .folded-fields summary {
    font-size: 13px;
    cursor: pointer;
    color: var(--el-text-color-regular);
  }
  .folded-fields[open] > summary {
    margin-bottom: 16px;
  }
  .folded-fields summary span {
    margin-left: 12px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
  .optional-description {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    line-height: 1.7;
  }
  .option-checks {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 12px;
    margin: 20px 0;
  }
  .option-checks :deep(.el-button) {
    margin-left: 0;
    height: 40px;
  }
  .completed-options {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 16px;
  }
  .completed-option {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 16px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    background: var(--el-bg-color);
  }
  .completed-option > div {
    min-width: 0;
    flex: 1;
  }
  .completed-option strong {
    display: block;
    font-size: 13px;
  }
  .completed-option span {
    display: block;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    margin-top: 4px;
    overflow-wrap: anywhere;
  }
  .completed-option :deep(.el-button) {
    margin-left: 0;
    padding: 6px;
  }
  .option-actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 16px;
  }
  .option-actions span {
    margin-right: auto;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
  .option-error {
    margin-top: 16px;
  }
  .option-section {
    padding: 24px;
    margin-top: 16px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }
  .option-section strong {
    display: block;
    font-size: 14px;
    margin-bottom: 20px;
  }
  .option-section p {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    line-height: 1.6;
  }
  @media (max-width: 600px) {
    .parameter-grid,
    .option-checks {
      grid-template-columns: 1fr;
    }
  }
</style>
