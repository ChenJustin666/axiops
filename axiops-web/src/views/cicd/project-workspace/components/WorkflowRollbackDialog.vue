<template>
  <ElDialog
    :model-value="visible"
    title="回滚服务"
    width="min(1080px, 95vw)"
    top="5vh"
    :close-on-click-modal="false"
    :close-on-press-escape="!submitting"
    :show-close="!submitting"
    destroy-on-close
    class="workflow-rollback-dialog"
    @update:model-value="close"
  >
    <div v-loading="loading" class="rollback-panel">
      <p class="rollback-intro"
        >选择一个服务及其成功版本，预览后回滚。只处理所选服务，不重新构建。</p
      >
      <ElAlert v-if="error" :title="error" type="error" :closable="false" show-icon />
      <ElEmpty
        v-if="!loading && !services.length && !error"
        description="此次运行没有可回滚的服务，或当前账号无回滚权限"
      />
      <div v-if="services.length" class="rollback-selectors">
        <label
          ><span>服务</span
          ><ElSelect
            v-model="selectedServiceId"
            :disabled="submitting || uncertain"
            placeholder="选择要回滚的服务"
            @change="selectService"
            ><ElOption
              v-for="service in services"
              :key="service.service_id"
              :value="service.service_id"
              :label="service.name" /></ElSelect
        ></label>
        <label
          ><span>恢复到成功版本</span
          ><ElSelect
            v-model="snapshotId"
            :disabled="!selectedServiceId || submitting || uncertain"
            placeholder="请选择历史成功版本"
            @change="loadPreview"
            ><ElOption
              v-for="version in versions"
              :key="version.snapshot_id"
              :value="version.snapshot_id"
              :label="versionLabel(version)" /></ElSelect
        ></label>
      </div>
      <p v-if="selectedServiceId && !versions.length && !loading" class="rollback-empty"
        >该服务暂无可用的成功部署快照。构建成功不等于部署成功。</p
      >
      <div v-loading="previewLoading" class="rollback-preview">
        <template v-if="preview">
          <div class="rollback-target"
            ><strong>{{ environmentName || `环境 #${preview.environment_id}` }}</strong
            ><span>集群 #{{ preview.cluster_id }} / {{ preview.namespace }}</span
            ><span>{{ preview.service_name }}</span></div
          >
          <ElAlert
            v-if="preview.blocked_reason"
            :title="preview.blocked_reason"
            type="warning"
            :closable="false"
            show-icon
          />
          <div class="rollback-versions"
            ><article
              ><small
                >当前部署记录{{
                  preview.current.status !== 'applied' ? ' · 未完整成功' : ''
                }}</small
              ><strong>{{ versionLabel(preview.current) }}</strong
              ><code v-for="image in preview.current.images" :key="image">{{ image }}</code
              ><span v-if="preview.current.helm_revision"
                >Helm revision {{ preview.current.helm_revision }}</span
              ></article
            ><span class="rollback-arrow">→</span
            ><article
              ><small>恢复目标 · 已成功</small><strong>{{ versionLabel(preview.target) }}</strong
              ><code v-for="image in preview.target.images" :key="image">{{ image }}</code
              ><span v-if="preview.target.helm_revision"
                >Helm revision {{ preview.target.helm_revision }}</span
              ></article
            ></div
          >
          <details v-if="changes.length" class="rollback-diff rollback-change-list" open
            ><summary>变更字段 · {{ changes.length }} 项</summary
            ><div
              ><p v-for="change in changes.slice(0, 100)" :key="change.path"
                ><strong>{{ change.path }}</strong
                ><span
                  ><del>{{ change.before }}</del> → <ins>{{ change.after }}</ins></span
                ></p
              ><p v-if="changes.length > 100">仅列出前 100 项，完整配置见下方。</p></div
            ></details
          >
          <details class="rollback-diff"
            ><summary>部署配置对照</summary
            ><div class="rollback-diff-columns"
              ><section
                ><span>当前部署快照</span><pre>{{ preview.current_manifest }}</pre></section
              ><section
                ><span>目标成功快照</span><pre>{{ preview.target_manifest }}</pre>
              </section></div
            ></details
          >
          <p v-for="warning in preview.warnings" :key="warning" class="rollback-note">{{
            warning
          }}</p>
          <p v-if="preview.requires_approval" class="rollback-note"
            >此环境按现有策略需要审批，提交后进入待审批状态。</p
          >
          <ElCheckbox
            v-model="confirmed"
            :disabled="!!preview.blocked_reason || submitting || uncertain"
            >确认将 {{ preview.service_name }} 恢复到所选版本，不执行其他服务的部署</ElCheckbox
          >
        </template>
      </div>
      <ElAlert
        v-if="uncertain"
        title="提交结果尚未确认，请先查看运行记录；也可用原请求标识核对，避免重复回滚。"
        type="warning"
        :closable="false"
        show-icon
      />
    </div>
    <template #footer
      ><ElButton :disabled="submitting" @click="close(false)">关闭</ElButton
      ><ElButton v-if="uncertain" :loading="submitting" @click="submit">核对提交结果</ElButton
      ><ElButton
        v-else
        type="primary"
        :loading="submitting"
        :disabled="!canSubmit"
        @click="submit"
        >{{ preview?.requires_approval ? '提交回滚审批' : '确认回滚' }}</ElButton
      ></template
    >
  </ElDialog>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue'
  import {
    loadPendingRollback,
    savePendingRollback,
    rollbackResultUncertain,
    rollbackConfigChanges,
    type PendingRollback
  } from '@/utils/workflow-rollback'
  import {
    getRollbackOptionsApi,
    previewRollbackApi,
    submitRollbackApi,
    type RollbackPreview,
    type RollbackService,
    type DeploymentVersion
  } from '@/api/axiops'

  const props = defineProps<{
    visible: boolean
    runId: number
    serviceId?: number
    environmentName?: string
    serviceNames?: Record<number, string>
  }>()
  const emit = defineEmits<{
    'update:visible': [value: boolean]
    submitted: [runId: number, workflowId: number]
  }>()
  const services = ref<RollbackService[]>([]),
    selectedServiceId = ref<number>(),
    snapshotId = ref<number>(),
    preview = ref<RollbackPreview | null>(null)
  const loading = ref(false),
    previewLoading = ref(false),
    submitting = ref(false),
    confirmed = ref(false),
    uncertain = ref(false),
    error = ref('')
  const pending = ref<PendingRollback | null>(null)
  let generation = 0,
    previewGeneration = 0,
    requestKey = ''
  const versions = computed(
    () => services.value.find((s) => s.service_id === selectedServiceId.value)?.versions || []
  )
  const changes = computed(() =>
    preview.value
      ? rollbackConfigChanges(preview.value.current_manifest, preview.value.target_manifest)
      : []
  )
  const versionLabel = (v: DeploymentVersion) =>
    `${v.workflow_id === preview.value?.workflow_id || !preview.value ? '' : `工作流 ${v.workflow_id} · `}#${v.run_number || '—'} · ${new Date(v.created_at).toLocaleString('zh-CN', { hour12: false })} · 配置 #${v.revision_id}`
  const canSubmit = computed(
    () =>
      !!preview.value &&
      !previewLoading.value &&
      !loading.value &&
      !preview.value.blocked_reason &&
      confirmed.value &&
      !submitting.value &&
      !uncertain.value
  )
  const message = (e: unknown) =>
    e instanceof Error ? e.message : (e as any)?.message || '请求失败，请重试'
  const close = (value: boolean) => {
    if (!submitting.value) emit('update:visible', value)
  }
  const selectService = () => {
    ++previewGeneration
    snapshotId.value = undefined
    preview.value = null
    confirmed.value = false
    error.value = ''
    requestKey = ''
    previewLoading.value = false
  }
  const loadPreview = async () => {
    const ticket = ++previewGeneration
    preview.value = null
    confirmed.value = false
    error.value = ''
    requestKey = ''
    if (!selectedServiceId.value || !snapshotId.value) return
    previewLoading.value = true
    try {
      const result = await previewRollbackApi(
        props.runId,
        selectedServiceId.value,
        snapshotId.value
      )
      if (ticket === previewGeneration && props.visible) preview.value = result
    } catch (e) {
      if (ticket === previewGeneration) error.value = message(e)
    } finally {
      if (ticket === previewGeneration) previewLoading.value = false
    }
  }
  const submit = async () => {
    if (
      submitting.value ||
      (!uncertain.value && !canSubmit.value) ||
      (!preview.value && !pending.value)
    )
      return
    const selected = preview.value
    if (!pending.value && selected) {
      requestKey ||= `rollback-${crypto.randomUUID()}`
      pending.value = {
        key: requestKey,
        snapshotId: selected.target.snapshot_id,
        workflowId: selected.workflow_id,
        request: {
          origin_run_id: selected.origin_run_id,
          service_id: selected.service_id,
          expected_token: selected.token
        }
      }
      savePendingRollback(props.runId, pending.value)
    }
    const operation = pending.value!
    submitting.value = true
    error.value = ''
    try {
      const result = await submitRollbackApi(operation.snapshotId, operation.request, operation.key)
      uncertain.value = false
      pending.value = null
      savePendingRollback(props.runId, null)
      emit('submitted', result.workflow_run_id, operation.workflowId)
      emit('update:visible', false)
    } catch (e: any) {
      error.value = message(e)
      uncertain.value = rollbackResultUncertain(e)
      if (!uncertain.value) {
        pending.value = null
        savePendingRollback(props.runId, null)
        preview.value = null
        confirmed.value = false
        requestKey = ''
      }
    } finally {
      submitting.value = false
    }
  }
  watch(
    () => [props.visible, props.runId, props.serviceId] as const,
    async ([visible]) => {
      const ticket = ++generation
      ++previewGeneration
      if (!visible) return
      services.value = []
      selectedServiceId.value = undefined
      snapshotId.value = undefined
      preview.value = null
      confirmed.value = false
      uncertain.value = false
      previewLoading.value = false
      error.value = ''
      requestKey = ''
      pending.value = loadPendingRollback(props.runId)
      if (pending.value) uncertain.value = true
      loading.value = true
      try {
        const result = await getRollbackOptionsApi(props.runId)
        if (ticket !== generation || !props.visible) return
        services.value = result.map((s) => ({
          ...s,
          name: props.serviceNames?.[s.service_id] || s.name
        }))
        selectedServiceId.value = result.some((s) => s.service_id === props.serviceId)
          ? props.serviceId
          : result.length === 1
            ? result[0].service_id
            : undefined
        if (pending.value) {
          selectedServiceId.value = pending.value.request.service_id
          snapshotId.value = pending.value.snapshotId
        }
      } catch (e) {
        if (ticket === generation) error.value = message(e)
      } finally {
        if (ticket === generation) loading.value = false
      }
    },
    { immediate: true }
  )
</script>

<style scoped lang="scss">
  :global(.workflow-rollback-dialog) {
    max-height: 90vh;
    display: flex;
    flex-direction: column;
  }
  :global(.workflow-rollback-dialog .el-dialog__body) {
    overflow: auto;
  }
  .rollback-change-list {
    margin-bottom: 12px;
    > div {
      max-height: 200px;
      overflow: auto;
      padding: 0 12px 12px;
    }
    p {
      display: grid;
      gap: 4px;
      font-size: 12px;
      line-height: 1.6;
      overflow-wrap: anywhere;
    }
    strong {
      color: var(--el-text-color-secondary);
      font-weight: 500;
    }
    del {
      color: var(--el-color-danger);
    }
    ins {
      color: var(--el-color-success);
      text-decoration: none;
    }
  }
  .rollback-panel {
    min-height: 180px;
  }
  .rollback-intro,
  .rollback-note,
  .rollback-empty {
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.6;
  }
  .rollback-selectors {
    display: grid;
    grid-template-columns: 1fr 2fr;
    gap: 20px;
    margin: 22px 0;
    label {
      display: grid;
      gap: 8px;
      font-size: 13px;
    }
  }
  .rollback-target {
    display: flex;
    flex-wrap: wrap;
    gap: 18px;
    align-items: center;
    padding: 14px 0;
    font-size: 13px;
  }
  .rollback-versions {
    display: grid;
    grid-template-columns: 1fr auto 1fr;
    gap: 18px;
    align-items: center;
    margin: 18px 0;
    article {
      display: grid;
      gap: 8px;
      min-width: 0;
      padding: 16px;
      border: 1px solid var(--el-border-color-lighter);
      border-radius: 8px;
    }
    small {
      color: var(--el-text-color-secondary);
    }
    strong {
      font-size: 13px;
    }
    code {
      font-size: 11px;
      overflow-wrap: anywhere;
      line-height: 1.6;
    }
  }
  .rollback-diff {
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    summary {
      cursor: pointer;
      padding: 12px;
      font-size: 13px;
    }
  }
  .rollback-diff-columns {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1px;
    background: var(--el-border-color-lighter);
    section {
      min-width: 0;
      background: var(--el-fill-color-blank);
    }
    span {
      display: block;
      padding: 8px 12px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
    pre {
      margin: 0;
      padding: 12px;
      height: min(34vh, 360px);
      overflow: auto;
      font-size: 12px;
      line-height: 1.6;
      white-space: pre;
      background: var(--el-fill-color-light);
    }
  }
  .rollback-panel :deep(.el-checkbox) {
    white-space: normal;
    height: auto;
    margin-top: 10px;
  }
  @media (max-width: 700px) {
    .rollback-selectors,
    .rollback-diff-columns {
      grid-template-columns: 1fr;
    }
    .rollback-versions {
      grid-template-columns: 1fr;
    }
    .rollback-arrow {
      text-align: center;
    }
  }
</style>
