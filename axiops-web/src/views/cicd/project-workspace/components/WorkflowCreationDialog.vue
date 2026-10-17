<template>
  <ElDialog
    v-model="open"
    title="从模板新建工作流"
    width="640px"
    :close-on-click-modal="false"
    :close-on-press-escape="!saving"
    :show-close="!saving"
  >
    <ElAlert v-if="error" :title="error" type="error" :closable="false" />
    <ElForm v-loading="loading" label-position="top">
      <ElFormItem label="流程模板" required>
        <ElSelect
          v-model="templateId"
          style="width: 100%"
          :disabled="saving"
          placeholder="选择模板库中的流程"
        >
          <ElOption
            v-for="item in compatibleTemplates"
            :key="item.id"
            :label="`${item.name} · v${item.revision}`"
            :value="item.id"
          />
        </ElSelect>
      </ElFormItem>
      <p v-if="selectedTemplate">{{ selectedTemplate.description }}</p>
      <ElFormItem v-if="serviceNodes.length" label="默认服务（可逐节点调整）">
        <ElSelect v-model="serviceId" filterable style="width: 100%" :disabled="saving">
          <ElOption
            v-for="service in services"
            :key="service.id"
            :label="service.name"
            :value="service.id"
          />
        </ElSelect>
      </ElFormItem>
      <details v-if="serviceNodes.length > 1">
        <summary>按节点选择服务</summary>
        <ElFormItem v-for="node in serviceNodes" :key="node.key" :label="node.name">
          <ElSelect
            v-model="nodeBindings[node.key]"
            clearable
            filterable
            :disabled="saving"
            placeholder="使用默认服务"
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
      </details>
      <ElFormItem v-if="deploying" label="目标环境（创建后固定）" required>
        <ElSelect
          v-model="environmentId"
          style="width: 100%"
          :disabled="saving"
          placeholder="明确选择目标环境"
        >
          <ElOption
            v-for="env in environments"
            :key="env.id"
            :value="env.id"
            :label="`${env.name} · ${env.namespace || env.host_group_ref}${env.is_production ? ' · 生产' : ''}`"
            :disabled="env.status !== 'active' || env.binding_status !== 'validated'"
          />
        </ElSelect>
      </ElFormItem>
      <ElFormItem label="工作流名称" required
        ><ElInput
          v-model="name"
          maxlength="128"
          :disabled="saving"
          placeholder="例如 risk 测试发布"
      /></ElFormItem>
      <ElFormItem label="工作流编码" required
        ><ElInput
          v-model="code"
          maxlength="64"
          :disabled="saving"
          placeholder="小写字母开头，可包含数字和连字符"
      /></ElFormItem>
      <p>{{
        deploying
          ? '部署使用所选环境独立配置。保存只创建工作流，不立即执行。'
          : '此流程不部署到环境。构建和测试节点使用所选服务的代码。'
      }}</p>
      <p>创建后可继续编排节点。模板更新不会改动已创建的工作流。</p>
    </ElForm>
    <template #footer
      ><ElButton :disabled="saving" @click="open = false">取消</ElButton
      ><ElButton
        type="primary"
        :loading="saving"
        :disabled="loading || !selectedTemplate"
        @click="create"
        >创建并编排</ElButton
      ></template
    >
  </ElDialog>
</template>
<script setup lang="ts">
  import { computed, ref, watch } from 'vue'
  import { ElMessage } from 'element-plus'
  import {
    createWorkflowApi,
    listDeliveryTemplatesApi,
    previewDeliveryTemplateApi,
    type AxiOpsProject,
    type AxiOpsService,
    type AxiOpsEnvironment,
    type DeliveryTemplate
  } from '@/api/axiops'
  import {
    instantiateWorkflowTemplate,
    readWorkflowTemplate,
    isDeploymentNode,
    nodeNeedsService
  } from '@/utils/workflow-template'
  const props = defineProps<{
    visible: boolean
    project: AxiOpsProject
    services: AxiOpsService[]
    environments: AxiOpsEnvironment[]
  }>()
  const emit = defineEmits<{ 'update:visible': [boolean]; created: [number] }>()
  const open = computed({
    get: () => props.visible,
    set: (value) => {
      if (!saving.value) emit('update:visible', value)
    }
  })
  const loading = ref(false),
    saving = ref(false),
    error = ref('')
  const templates = ref<DeliveryTemplate[]>([]),
    templateId = ref<number>(),
    serviceId = ref<number>(),
    environmentId = ref<number>()
  const name = ref(''),
    code = ref('')
  const nodeBindings = ref<Record<string, number>>({})
  let generation = 0
  const compatibleTemplates = computed(() =>
    templates.value.filter((item) => {
      try {
        const d = readWorkflowTemplate(item)
        return (
          d.delivery_type === props.project.delivery_type ||
          !d.nodes.some((n) => isDeploymentNode(n.type))
        )
      } catch {
        return false
      }
    })
  )
  const selectedTemplate = computed(() =>
    compatibleTemplates.value.find((t) => t.id === templateId.value)
  )
  const serviceNodes = computed(() =>
    selectedTemplate.value
      ? readWorkflowTemplate(selectedTemplate.value).nodes.filter((n) => nodeNeedsService(n.type))
      : []
  )
  watch(templateId, () => {
    nodeBindings.value = {}
  })
  const deploying = computed(
    () =>
      !!selectedTemplate.value &&
      readWorkflowTemplate(selectedTemplate.value).nodes.some((n) => isDeploymentNode(n.type))
  )
  watch(
    () => [props.visible, props.project.id],
    async () => {
      const request = ++generation
      if (!props.visible) return
      templates.value = []
      templateId.value = undefined
      serviceId.value = props.services.length === 1 ? props.services[0].id : undefined
      environmentId.value = undefined
      name.value = ''
      code.value = `flow-${Date.now()}`
      error.value = ''
      loading.value = true
      try {
        const list = await listDeliveryTemplatesApi('workflow')
        if (request === generation) templates.value = list || []
      } catch (e: any) {
        if (request === generation) error.value = e.message || '模板读取失败'
      } finally {
        if (request === generation) loading.value = false
      }
    },
    { immediate: true }
  )
  async function create() {
    if (saving.value || loading.value || !selectedTemplate.value) return
    if (props.project.delivery_type === 'unclassified')
      return ElMessage.warning('请先设置项目交付类型')
    const service = props.services.find((s) => s.id === serviceId.value)
    const env = props.environments.find((e) => e.id === environmentId.value)
    if (!name.value.trim() || !/^[a-z][a-z0-9-]{0,63}$/.test(code.value))
      return ElMessage.warning('请填写名称和有效编码')
    if (deploying.value && (!env || env.status !== 'active' || env.binding_status !== 'validated'))
      return ElMessage.warning('请选择已验证的目标环境')
    const template = selectedTemplate.value
    saving.value = true
    error.value = ''
    try {
      const bindings: Record<string, AxiOpsService> = {}
      for (const node of serviceNodes.value) {
        const selected = props.services.find(
          (s) => s.id === (nodeBindings.value[node.key] || serviceId.value)
        )
        if (!selected) throw new Error(`节点 ${node.name} 请选择服务`)
        bindings[node.key] = selected
      }
      const definition = instantiateWorkflowTemplate(
        template,
        props.project.delivery_type,
        service,
        bindings
      )
      const projectId = props.project.id
      const request = {
        name: name.value.trim(),
        code: code.value,
        description: `来自模板 ${template.name} v${template.revision}`,
        target_environment_id: deploying.value ? env!.id : 0,
        definition
      }
      await previewDeliveryTemplateApi(template)
      const result = await createWorkflowApi(projectId, request)
      emit('created', result.id)
      emit('update:visible', false)
      ElMessage.success('工作流已创建，尚未执行')
    } catch (e: any) {
      error.value = e.message || '创建失败，请查看工作流列表确认结果后重试'
    } finally {
      saving.value = false
    }
  }
</script>
<style scoped>
  p {
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.6;
  }
</style>
