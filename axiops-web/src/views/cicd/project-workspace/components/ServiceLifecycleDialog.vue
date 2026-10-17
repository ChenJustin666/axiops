<template>
  <ElDialog
    :model-value="visible"
    :title="title"
    width="680px"
    append-to-body
    destroy-on-close
    :close-on-click-modal="false"
    :close-on-press-escape="!saving"
    :show-close="!saving"
    @update:model-value="close"
  >
    <div v-loading="loading" class="lifecycle-body">
      <ElAlert v-if="error" :title="error" type="error" :closable="false" />
      <template v-if="mode === 'rename' && service">
        <ElForm label-position="top" @submit.prevent="rename">
          <ElFormItem label="服务名称"
            ><ElInput
              v-model="name"
              maxlength="64"
              show-word-limit
              :disabled="saving"
              placeholder="例如 风控网关"
              @keyup.enter="rename"
          /></ElFormItem>
          <ElFormItem label="服务编码"><ElInput :model-value="service.code" disabled /></ElFormItem>
        </ElForm>
        <p class="hint"
          >只修改展示名称。服务编码、镜像路径和已部署资源保持不变，不触发构建或部署。</p
        >
      </template>
      <template v-else-if="mode === 'delete' && service">
        <p
          >删除服务 <strong>{{ service.name }}</strong
          >（{{ service.code }}）？</p
        >
        <ElAlert
          title="只从平台正常列表移除，可在“已删除服务”恢复。保留配置版本、构建与部署日志，不卸载集群应用，也不删除仓库或镜像。"
          type="warning"
          :closable="false"
          show-icon
        />
        <p v-if="check?.deployment_history" class="hint"
          >此服务有
          {{ check.deployment_history }}
          条历史部署记录。集群中的应用可能仍在运行，删除平台配置不会将其停服。</p
        >
        <div v-if="check?.blockers.length" class="references">
          <strong>请先处理以下关联或执行中的任务</strong>
          <div v-for="ref in check.blockers" :key="`${ref.kind}:${ref.id}`" class="reference-row">
            <ElTag size="small">{{ labels[ref.kind] }}</ElTag
            ><span :title="ref.name">{{ ref.name }}</span>
            <ElButton
              v-if="ref.kind === 'environment' && canDetach(ref.id)"
              text
              type="danger"
              size="small"
              :disabled="saving"
              @click="detach(ref)"
              >解除关联</ElButton
            >
            <RouterLink
              v-if="ref.kind === 'workflow'"
              :to="`/cicd/projects/${projectId}/workflows/${ref.id}`"
              target="_blank"
              >查看工作流</RouterLink
            >
          </div>
          <p class="hint"
            >修改工作流，移除引用此服务的节点；未结束任务请等待完成或在运行页取消，然后刷新检查。</p
          >
        </div>
        <p v-else-if="check?.can_delete" class="hint"
          >未发现环境关联、工作流引用或未结束任务，可以删除。</p
        >
      </template>
      <template v-else-if="mode === 'trash'">
        <p class="hint"
          >YAML 和 Helm 服务均可恢复或彻底删除。恢复保留原配置；彻底删除后编码可重新使用。</p
        >
        <ElInput v-model="search" placeholder="搜索已删除服务" clearable :disabled="saving" />
        <ElTable :data="filtered" max-height="360" empty-text="暂无已删除服务">
          <ElTableColumn prop="name" label="服务名称" show-overflow-tooltip />
          <ElTableColumn prop="code" label="编码" show-overflow-tooltip />
          <ElTableColumn label="类型" width="75">
            <template #default="{ row }">{{
              row.delivery_type === 'helm'
                ? 'Helm'
                : row.delivery_type === 'k8s_yaml'
                  ? 'YAML'
                  : '主机'
            }}</template>
          </ElTableColumn>
          <ElTableColumn label="操作" width="160"
            ><template #default="{ row }"
              ><ElButton
                v-if="canManage(row)"
                text
                type="primary"
                :disabled="saving"
                @click="restore(row)"
                >恢复</ElButton
              ><ElButton
                v-if="canManage(row)"
                text
                type="danger"
                :disabled="saving"
                @click="purge(row)"
                >彻底删除</ElButton
              ></template
            ></ElTableColumn
          >
        </ElTable>
      </template>
    </div>
    <template #footer>
      <ElButton v-if="mode !== 'rename'" :disabled="saving || loading" @click="load"
        >刷新{{ mode === 'delete' ? '检查' : '列表' }}</ElButton
      >
      <ElButton :disabled="saving" @click="close(false)">{{
        mode === 'trash' ? '关闭' : '取消'
      }}</ElButton>
      <ElButton
        v-if="mode === 'rename'"
        type="primary"
        :loading="saving"
        :disabled="!name.trim() || loading"
        @click="rename"
        >保存名称</ElButton
      >
      <ElButton
        v-if="mode === 'delete'"
        type="danger"
        :loading="saving"
        :disabled="loading || !check?.can_delete"
        @click="remove"
        >确认删除</ElButton
      >
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, ref, watch } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import {
    checkServiceDeletionApi,
    deleteServiceApi,
    detachServiceEnvironmentApi,
    listAllServicesApi,
    renameServiceApi,
    restoreServiceApi,
    purgeServiceApi,
    type AxiOpsService,
    type AxiOpsEnvironment,
    type ServiceDeletionCheck
  } from '@/api/axiops'
  import { useAxiOpsAccess } from '@/composables/useAxiOpsAccess'
  const props = defineProps<{
    visible: boolean
    projectId: number
    environments?: AxiOpsEnvironment[]
    service: AxiOpsService | null
    mode: 'delete' | 'rename' | 'trash'
  }>()
  const emit = defineEmits<{ 'update:visible': [value: boolean]; changed: [] }>()
  const access = useAxiOpsAccess()
  const title = computed(
    () => ({ delete: '删除服务', rename: '编辑服务名称', trash: '服务垃圾桶' })[props.mode]
  )
  const loading = ref(false),
    saving = ref(false),
    error = ref(''),
    name = ref(''),
    search = ref('')
  const check = ref<ServiceDeletionCheck | null>(null),
    deleted = ref<AxiOpsService[]>([])
  const labels = { environment: '环境', workflow: '工作流', build: '构建', run: '执行任务' }
  const filtered = computed(() =>
    deleted.value.filter((item) =>
      `${item.name} ${item.code}`.toLowerCase().includes(search.value.toLowerCase())
    )
  )
  const canManage = (service: AxiOpsService) =>
    access.can('service', 'delete', {
      type: 'service',
      id: service.id,
      project_id: props.projectId
    })
  const canDetach = (id: number) =>
    access.can('environment', 'update', {
      type: 'environment',
      id,
      project_id: props.projectId,
      environment: props.environments?.find((item) => item.id === id)?.code
    })
  let generation = 0
  function close(value: boolean) {
    if (!saving.value) emit('update:visible', value)
  }
  async function load() {
    const version = ++generation
    loading.value = true
    error.value = ''
    check.value = null
    try {
      if (props.mode === 'trash') {
        const rows = await listAllServicesApi(props.projectId, { status: 'deleted' })
        if (version === generation) deleted.value = rows || []
      } else if (props.mode === 'delete' && props.service) {
        const result = await checkServiceDeletionApi(props.service.id)
        if (version === generation) check.value = result
      }
    } catch (e: any) {
      if (version === generation) error.value = e.message || '读取失败，请重试'
    } finally {
      if (version === generation) loading.value = false
    }
  }
  async function remove() {
    if (saving.value || loading.value || !props.service || !check.value?.can_delete) return
    saving.value = true
    error.value = ''
    try {
      await deleteServiceApi(props.service.id)
      ElMessage.success('服务已删除，可在“已删除服务”恢复')
      emit('changed')
      emit('update:visible', false)
    } catch (e: any) {
      await load()
      error.value = e.message || '删除失败，请重新检查引用'
    } finally {
      saving.value = false
    }
  }
  async function rename() {
    if (saving.value || !props.service || !name.value.trim()) return
    saving.value = true
    error.value = ''
    try {
      await renameServiceApi(props.service.id, name.value.trim())
      ElMessage.success('服务名称已更新')
      emit('changed')
      emit('update:visible', false)
    } catch (e: any) {
      error.value = e.message || '名称更新失败'
    } finally {
      saving.value = false
    }
  }
  async function restore(service: AxiOpsService) {
    if (saving.value) return
    saving.value = true
    try {
      await ElMessageBox.confirm(
        `恢复服务“${service.name}”？保留原配置与记录，不会自动部署。`,
        '恢复服务',
        { confirmButtonText: '恢复', cancelButtonText: '取消' }
      )
      await restoreServiceApi(service.id)
      ElMessage.success('服务已恢复')
      emit('changed')
      await load()
    } catch (e: any) {
      if (e !== 'cancel' && e !== 'close') error.value = e.message || '恢复失败'
    } finally {
      saving.value = false
    }
  }
  async function purge(service: AxiOpsService) {
    if (saving.value || !canManage(service)) return
    saving.value = true
    error.value = ''
    try {
      const result = await checkServiceDeletionApi(service.id)
      if (!result.can_purge) {
        throw new Error(
          result.blockers.length
            ? `请先处理关联或任务：${result.blockers.map((item) => item.name).join('、')}`
            : '服务已恢复或状态已变化，请刷新列表'
        )
      }
      const { value } = await ElMessageBox.prompt(
        `永久删除“${service.name}”及其配置版本、构建配置和专属变量，无法恢复。历史执行记录保留，集群应用不会被卸载。请输入服务编码 ${service.code} 确认。`,
        '彻底删除服务',
        {
          type: 'warning',
          confirmButtonText: '彻底删除',
          cancelButtonText: '取消',
          inputPlaceholder: service.code,
          inputValidator: (value: string) => value === service.code || '请输入完整服务编码',
          closeOnClickModal: false
        }
      )
      if (value !== service.code) return
      await purgeServiceApi(service.id, value)
      ElMessage.success('服务已彻底删除，可以重新使用此编码')
      emit('changed')
      await load()
    } catch (e: any) {
      if (e !== 'cancel' && e !== 'close') error.value = e.message || '彻底删除失败，请刷新检查'
    } finally {
      saving.value = false
    }
  }
  async function detach(ref: ServiceDeletionCheck['blockers'][number]) {
    if (saving.value || !props.service) return
    const serviceId = props.service.id
    saving.value = true
    try {
      await ElMessageBox.confirm(
        `解除与“${ref.name}”的关联？移除该环境的服务版本、镜像和副本覆盖配置，保留变量与历史记录，不卸载运行中的应用。`,
        '解除环境关联',
        { type: 'warning', confirmButtonText: '解除关联', cancelButtonText: '取消' }
      )
      await detachServiceEnvironmentApi(serviceId, ref.id, ref.revision)
      emit('changed')
      await load()
    } catch (e: any) {
      if (e !== 'cancel' && e !== 'close') {
        await load()
        error.value = e.message || '解除失败'
      }
    } finally {
      saving.value = false
    }
  }
  watch(
    () => [props.visible, props.service?.id, props.mode, props.projectId],
    () => {
      generation++
      check.value = null
      error.value = ''
      deleted.value = []
      search.value = ''
      if (!props.visible) return
      name.value = props.service?.name || ''
      void load()
    },
    { immediate: true }
  )
  onBeforeUnmount(() => generation++)
</script>

<style scoped>
  .lifecycle-body {
    display: flex;
    flex-direction: column;
    gap: 14px;
    min-height: 120px;
  }
  .hint {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.8;
    margin: 0;
  }
  .references {
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    padding: 12px;
    max-height: 320px;
    overflow: auto;
  }
  .reference-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 0;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  .reference-row > span:not(.el-tag) {
    flex: 1;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .reference-row a {
    font-size: 12px;
    color: var(--el-color-primary);
  }
</style>
