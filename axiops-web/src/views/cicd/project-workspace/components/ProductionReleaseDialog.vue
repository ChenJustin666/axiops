<template>
  <ElDialog
    :model-value="visible"
    title="执行工作流"
    width="min(960px, 95vw)"
    top="6vh"
    :close-on-click-modal="false"
    :close-on-press-escape="!submitting"
    :show-close="!submitting"
    class="production-release-dialog"
    @update:model-value="close"
  >
    <div v-loading="loading" class="production-body">
      <ElAlert v-if="error" :title="error" type="error" :closable="false" show-icon />
      <template v-if="options">
        <div class="production-heading"
          ><div
            ><small>目标环境</small><strong>{{ options.environment_name }}</strong></div
          >
          <ElTag size="small" type="danger" effect="plain">生产环境</ElTag></div
        >
        <div class="production-controls">
          <label>部署目标</label
          ><span class="production-target"
            >集群 #{{ options.cluster_id }} / {{ options.namespace }}</span
          >
          <label>发布服务</label>
          <ElSelect
            v-model="selectedIds"
            multiple
            :disabled="locked || !!verificationId"
            placeholder="选择服务"
            @change="selectServices"
          >
            <ElOption
              v-for="item in options.services"
              :key="item.service_id"
              :value="item.service_id"
              :label="item.name"
            />
          </ElSelect>
          <label>发布方式</label>
          <ElRadioGroup v-model="mode" :disabled="locked" @change="modeChanged">
            <ElRadioButton value="build">构建并部署</ElRadioButton>
            <ElRadioButton value="image">使用已有镜像</ElRadioButton>
          </ElRadioGroup>
        </div>
        <template v-if="mode === 'image'">
          <div class="production-tools">
            <ElButton
              text
              size="small"
              :disabled="locked"
              :aria-expanded="advanced"
              @click="advanced = !advanced"
              >高级</ElButton
            >
            <ElButton text size="small" :disabled="locked || loading" @click="refreshVersions"
              >刷新镜像</ElButton
            >
          </div>
          <div v-if="advanced" class="production-source"
            ><label>镜像来源</label>
            <ElSelect
              v-model="sourceId"
              :disabled="locked || !!verificationId"
              @change="sourceChanged"
            >
              <ElOption
                v-for="source in options.image_sources"
                :key="source.id"
                :value="source.id"
                :label="source.name"
              />
              <ElOption :value="0" label="全部来源" />
            </ElSelect>
          </div>
          <section v-for="sid in selectedIds" :key="sid" class="production-service-card">
            <div class="production-card-heading"
              ><strong>{{ service(sid)?.name }}</strong>
              <span v-if="selectedIds.length > 1"
                ><ElButton
                  text
                  size="small"
                  :disabled="locked || selectedIds[0] === sid"
                  @click="move(sid, -1)"
                  >上移</ElButton
                >
                <ElButton
                  text
                  size="small"
                  :disabled="locked || selectedIds[selectedIds.length - 1] === sid"
                  @click="move(sid, 1)"
                  >下移</ElButton
                ></span
              ></div
            >
            <ElSelect
              v-model="chosen[sid]"
              class="production-image-select"
              popper-class="production-image-dropdown"
              :fit-input-width="true"
              filterable
              :disabled="locked || !!verificationId || !images(sid).length"
              :placeholder="images(sid).length ? '选择镜像版本' : '此来源暂无成功构建的镜像'"
              @change="loadPreview(sid)"
            >
              <ElOption
                v-for="version in images(sid)"
                :key="version.build_id"
                :value="version.build_id"
                :label="optionLabel(version, sid)"
                class="production-image-option"
              >
                <div
                  ><strong>{{ imageTag(version) }}</strong
                  ><span>{{
                    version.build_id === recommended(sid)?.build_id
                      ? version.deployment_status === 'succeeded'
                        ? '推荐 · 已部署成功'
                        : '最新构建'
                      : '构建成功'
                  }}</span></div
                >
                <small
                  >{{ date(version.built_at) }} · {{ version.branch || '未知分支' }} ·
                  {{ version.commit.slice(0, 8) || '未知提交' }}</small
                >
              </ElOption>
            </ElSelect>
            <p v-if="!images(sid).length" class="production-note"
              >暂无镜像，可在“高级”中切换来源，或选择“构建并部署”。</p
            >
            <div v-if="candidate(sid)" class="production-metadata">
              <ElTag size="small" type="success">构建成功</ElTag>
              <ElTag
                v-if="candidate(sid)?.deployment_status === 'succeeded'"
                size="small"
                type="success"
                effect="plain"
                >来源环境部署成功</ElTag
              >
              <span>{{ date(candidate(sid)?.built_at) }}</span
              ><span>{{ candidate(sid)?.branch || '未知分支' }}</span>
              <a
                v-if="safeURL(candidate(sid)?.commit_url)"
                :href="safeURL(candidate(sid)?.commit_url)"
                target="_blank"
                rel="noopener noreferrer"
                >{{ candidate(sid)?.commit.slice(0, 8) }}</a
              >
              <span v-else>{{ candidate(sid)?.commit.slice(0, 8) || '未知提交' }}</span>
              <a
                v-if="safeURL(candidate(sid)?.compare_url)"
                :href="safeURL(candidate(sid)?.compare_url)"
                target="_blank"
                rel="noopener noreferrer"
                title="与该构建的上一版本比较，不代表与当前生产比较"
                >本次构建变更</a
              >
              <ElButton
                v-if="candidate(sid)?.run_id"
                link
                type="primary"
                @click="openOrigin(candidate(sid)!)"
                >来源记录</ElButton
              >
            </div>
            <div v-loading="previewLoading[sid]" class="production-preview">
              <ElAlert
                v-if="previewErrors[sid]"
                :title="previewErrors[sid]"
                type="error"
                :closable="false"
              />
              <ElButton
                v-if="previewErrors[sid] && candidate(sid)"
                text
                :disabled="locked"
                @click="loadPreview(sid)"
                >重新检查</ElButton
              >
              <ElButton
                v-if="previewErrors[sid]"
                text
                type="primary"
                :disabled="locked"
                @click="configure(sid)"
                >配置目标环境</ElButton
              >
              <template v-if="previews[sid]">
                <p class="production-target"
                  >集群 #{{ previews[sid].cluster_id }} / {{ previews[sid].namespace }}</p
                >
                <ElAlert
                  v-if="previews[sid].blocked_reason"
                  :title="previews[sid].blocked_reason"
                  type="warning"
                  :closable="false"
                />
                <details class="production-diff"
                  ><summary>镜像与目标配置 · {{ changes(sid).length }} 项变更</summary>
                  <div class="production-images"
                    ><div
                      ><small>当前镜像</small
                      ><code>{{ previews[sid].current_image || '首次发布' }}</code></div
                    >
                    <div
                      ><small>本次镜像</small
                      ><code>{{ previews[sid].candidate.image_ref }}</code></div
                    ></div
                  >
                  <p v-for="change in changes(sid).slice(0, 100)" :key="change.path"
                    ><strong>{{ change.path }}</strong
                    ><span
                      ><del>{{ change.before }}</del> → <ins>{{ change.after }}</ins></span
                    ></p
                  >
                  <details
                    ><summary>完整部署 YAML</summary><pre>{{ previews[sid].target_manifest }}</pre>
                  </details>
                </details>
              </template>
            </div>
          </section>
          <p class="production-note"
            >使用所选镜像原始摘要与当前目标环境配置；来源部署成功不等同于业务测试通过。</p
          >
          <p v-if="requiresApproval" class="production-note"
            >此环境沿用现有审批策略，提交后进入待审批状态。</p
          >
        </template>
        <div v-else class="production-build-info">
          <strong>构建新镜像，并部署到「{{ options.environment_name }}」</strong>
          <p>下一步选择各服务分支，核对目标集群与命名空间后执行。只有一个环境也可以直接使用。</p>
        </div>
        <p v-if="releaseHint" class="production-note" role="status">{{ releaseHint }}</p>
      </template>
      <ElAlert
        v-if="uncertain"
        title="提交结果尚未确认，请用原请求核对结果，避免重复发布。"
        type="warning"
        :closable="false"
      />
    </div>
    <template #footer>
      <ElButton :disabled="submitting" @click="close(false)">取消</ElButton>
      <ElButton v-if="!options && !loading" @click="refreshVersions">重新加载</ElButton>
      <ElButton v-if="uncertain" :loading="submitting" @click="submit">核对提交结果</ElButton>
      <ElButton
        v-else-if="mode === 'build' && options"
        type="primary"
        :disabled="locked || !selectedIds.length"
        @click="openBuild"
        >下一步：选择分支</ElButton
      >
      <ElButton
        v-else-if="options"
        type="primary"
        :disabled="!canSubmit"
        :loading="submitting"
        @click="submit"
        >{{ requiresApproval ? '提交发布' : '发布' }}</ElButton
      >
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue'
  import { useRouter } from 'vue-router'
  import {
    getProductionOptionsApi,
    getProductionImagePreviewApi,
    submitProductionApi,
    type ProductionOptions,
    type ProductionCandidate,
    type ProductionPreview,
    type ProductionPick
  } from '@/api/axiops'
  import { rollbackConfigChanges, rollbackResultUncertain } from '@/utils/workflow-rollback'

  const props = defineProps<{
    visible: boolean
    workflowId: number
    projectId: number
    serviceId?: number
    initialServiceIds?: number[]
    verificationId?: number
  }>()
  const emit = defineEmits<{
    'update:visible': [value: boolean]
    submitted: [runId: number]
    build: [serviceIds: number[]]
    configure: [serviceId: number, environmentId: number]
  }>()
  const router = useRouter()
  const options = ref<ProductionOptions | null>(null),
    selectedIds = ref<number[]>([]),
    sourceId = ref(0),
    mode = ref<'image' | 'build'>('image'),
    advanced = ref(false),
    loading = ref(false),
    submitting = ref(false),
    uncertain = ref(false),
    error = ref('')
  const chosen = reactive<Record<number, number>>({}),
    previews = reactive<Record<number, ProductionPreview>>({}),
    previewErrors = reactive<Record<number, string>>({}),
    previewLoading = reactive<Record<number, boolean>>({}),
    versions = reactive<Record<number, number>>({})
  const locked = computed(() => submitting.value || uncertain.value)
  let generation = 0
  const pending = ref<{ key: string; picks: ProductionPick[] } | null>(null)
  const storageKey = () => `axiops:production:${props.workflowId}`
  const storePending = () => {
    try {
      if (pending.value) sessionStorage.setItem(storageKey(), JSON.stringify(pending.value))
      else sessionStorage.removeItem(storageKey())
    } catch {
      /* In-memory identity remains available. */
    }
  }
  const service = (sid: number) => options.value?.services.find((s) => s.service_id === sid)
  const images = (sid: number) => {
    const byBuild = new Map<number, ProductionCandidate>()
    for (const version of service(sid)?.images || []) {
      if (sourceId.value && version.environment_id !== sourceId.value) continue
      const old = byBuild.get(version.build_id)
      if (
        !old ||
        (old.deployment_status !== 'succeeded' && version.deployment_status === 'succeeded')
      )
        byBuild.set(version.build_id, version)
    }
    return [...byBuild.values()].sort((a, b) => b.build_id - a.build_id)
  }
  const candidate = (sid: number) => images(sid).find((v) => v.build_id === chosen[sid])
  const recommended = (sid: number) =>
    images(sid).find((v) => v.deployment_status === 'succeeded') || images(sid)[0]
  const date = (value?: string) =>
    value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '时间未知'
  const imageTag = (v: ProductionCandidate) =>
    v.image_ref.split('@')[0].split('/').pop() || `构建 #${v.build_number}`
  const optionLabel = (v: ProductionCandidate, sid: number) =>
    `${imageTag(v)} · ${date(v.built_at)} · ${v.branch || '未知分支'} · ${v.commit.slice(0, 8) || '未知提交'}${v.build_id === recommended(sid)?.build_id && v.deployment_status === 'succeeded' ? ' · 推荐' : ''}`
  const safeURL = (value?: string) => (/^https?:\/\//i.test(value || '') ? value : undefined)
  const changes = (sid: number) =>
    previews[sid]
      ? rollbackConfigChanges(previews[sid].current_manifest, previews[sid].target_manifest)
      : []
  const previewReady = computed(
    () =>
      selectedIds.value.length > 0 &&
      selectedIds.value.every(
        (sid) =>
          !!candidate(sid) &&
          previews[sid]?.candidate.build_id === chosen[sid] &&
          !previewLoading[sid] &&
          !previewErrors[sid] &&
          !previews[sid]?.blocked_reason
      )
  )
  const requiresApproval = computed(() =>
    selectedIds.value.some((sid) => previews[sid]?.requires_approval)
  )
  const canSubmit = computed(
    () => mode.value === 'image' && !loading.value && !locked.value && previewReady.value
  )
  const releaseHint = computed(() => {
    if (loading.value || uncertain.value || mode.value === 'build') return ''
    if (!selectedIds.value.length) return '请选择要发布的服务。'
    if (selectedIds.value.some((sid) => previewLoading[sid])) return '正在检查镜像与目标环境配置…'
    if (selectedIds.value.some((sid) => previewErrors[sid]))
      return '配置检查未通过，请按服务下方提示处理。'
    if (selectedIds.value.some((sid) => !chosen[sid])) return '请为每个服务选择镜像版本。'
    return ''
  })
  const message = (e: unknown) => (e as { message?: string })?.message || '请求失败，请重试'
  const close = (value: boolean) => {
    if (!submitting.value) {
      generation++
      emit('update:visible', value)
    }
  }
  const openOrigin = (v: ProductionCandidate) => {
    if (locked.value || !v.run_id || !v.workflow_id) return
    close(false)
    void router.push(
      `/cicd/projects/${props.projectId}/workflows/${v.workflow_id}/runs/${v.run_id}`
    )
  }
  const move = (sid: number, offset: number) => {
    if (locked.value) return
    const i = selectedIds.value.indexOf(sid),
      j = i + offset
    if (i < 0 || j < 0 || j >= selectedIds.value.length) return
    const ids = [...selectedIds.value]
    ;[ids[i], ids[j]] = [ids[j], ids[i]]
    selectedIds.value = ids
  }
  const loadPreview = async (sid: number) => {
    if (locked.value) return
    const gen = generation,
      version = (versions[sid] || 0) + 1,
      id = chosen[sid]
    versions[sid] = version
    delete previews[sid]
    previewErrors[sid] = ''
    previewLoading[sid] = false
    if (!id || !candidate(sid)) return
    previewLoading[sid] = true
    try {
      const result = await getProductionImagePreviewApi(props.workflowId, id)
      if (gen === generation && version === versions[sid] && chosen[sid] === id)
        previews[sid] = result
    } catch (e) {
      if (gen === generation && version === versions[sid]) previewErrors[sid] = message(e)
    } finally {
      if (gen === generation && version === versions[sid]) previewLoading[sid] = false
    }
  }
  const selectServices = async () => {
    if (locked.value) return
    for (const sid of selectedIds.value) {
      if (!chosen[sid]) {
        const version = recommended(sid)
        if (version) chosen[sid] = version.build_id
      }
      if (chosen[sid] && !previews[sid]) await loadPreview(sid)
    }
  }
  const sourceChanged = async () => {
    if (locked.value) return
    generation++
    for (const state of [chosen, previews, previewErrors, previewLoading, versions])
      for (const key of Object.keys(state)) delete state[Number(key)]
    await selectServices()
  }
  const modeChanged = () => {
    if (mode.value === 'image') void selectServices()
    else openBuild()
  }
  const configure = (sid: number) => {
    if (locked.value || !options.value) return
    const environmentId = options.value.environment_id
    close(false)
    emit('configure', sid, environmentId)
  }
  const openBuild = () => {
    if (!locked.value && selectedIds.value.length) {
      close(false)
      emit('build', [...selectedIds.value])
    }
  }
  const load = async (preserve = false) => {
    if (submitting.value) return
    if (!preserve) advanced.value = false
    const retained =
      preserve && options.value
        ? {
            ids: [...selectedIds.value],
            chosen: { ...chosen },
            source: sourceId.value,
            mode: mode.value
          }
        : null
    const gen = ++generation
    loading.value = true
    error.value = ''
    options.value = null
    selectedIds.value = []
    pending.value = null
    uncertain.value = false
    mode.value = 'image'
    for (const state of [chosen, previews, previewErrors, previewLoading, versions])
      for (const key of Object.keys(state)) delete state[Number(key)]
    try {
      const saved = sessionStorage.getItem(storageKey())
      if (saved) {
        const value = JSON.parse(saved)
        if (
          typeof value.key === 'string' &&
          Array.isArray(value.picks) &&
          value.picks.length &&
          value.picks.every(
            (p: ProductionPick) =>
              ((p.build_id || 0) > 0 || (p.verification_id || 0) > 0) &&
              p.expected_token?.length === 64
          )
        ) {
          pending.value = value
          uncertain.value = true
        }
      }
    } catch {
      /* Storage may be unavailable. */
    }
    try {
      const result = await getProductionOptionsApi(props.workflowId)
      if (gen !== generation) return
      options.value = result
      sourceId.value = retained
        ? retained.source
        : result.image_sources.find((s) => ['qa', 'test', 'testing'].includes(s.code || ''))?.id ||
          result.image_sources.find((s) => s.id === result.policy.source_environment_id)?.id ||
          0
      if (uncertain.value) return
      if (retained) {
        mode.value = retained.mode
        selectedIds.value = retained.ids
        for (const sid of retained.ids) {
          if (gen !== generation) return
          const id = retained.chosen[sid]
          if (!service(sid)) {
            previewErrors[sid] = '原选服务已不可用，请重新选择服务。'
            continue
          }
          if (id && images(sid).some((v) => v.build_id === id)) {
            chosen[sid] = id
            await loadPreview(sid)
          } else if (id)
            previewErrors[sid] = '原选版本已失效，请重新选择镜像；系统不会替换成其他版本。'
        }
        return
      }
      if (props.initialServiceIds?.length && !props.verificationId) {
        if (!props.initialServiceIds.every((sid) => service(sid))) {
          error.value = '所选服务已变化，请重新选择服务。'
          return
        }
        selectedIds.value = [...new Set(props.initialServiceIds)]
        await selectServices()
        return
      }
      const first = props.serviceId
        ? result.services.find((s) => s.service_id === props.serviceId)
        : result.services[0]
      if (props.serviceId && !first) {
        error.value = '此工作流不包含来源服务，请检查编排。'
        return
      }
      if (props.verificationId) {
        const verified = first?.candidates.find((v) => v.id === props.verificationId)
        if (!verified || !first?.images.some((v) => v.build_id === verified.build_id)) {
          error.value = '指定版本已失效，请返回来源记录核对。'
          return
        }
        sourceId.value = 0
        chosen[first.service_id] = verified.build_id
      }
      if (first) {
        selectedIds.value = [first.service_id]
        await selectServices()
      }
    } catch (e) {
      if (gen === generation) error.value = message(e)
    } finally {
      if (gen === generation) loading.value = false
    }
  }
  const refreshVersions = () => load(true)
  const submit = async () => {
    if (submitting.value || (!uncertain.value && !canSubmit.value)) return
    if (!pending.value) {
      pending.value = {
        key: `production-${crypto.randomUUID()}`,
        picks: selectedIds.value.map((sid) => ({
          build_id: chosen[sid],
          expected_token: previews[sid].token
        }))
      }
      storePending()
    }
    submitting.value = true
    error.value = ''
    try {
      const result = await submitProductionApi(
        props.workflowId,
        pending.value.picks,
        pending.value.key
      )
      pending.value = null
      storePending()
      uncertain.value = false
      emit('submitted', result.workflow_run_id)
      emit('update:visible', false)
    } catch (e) {
      error.value = message(e)
      if (rollbackResultUncertain(e as any)) uncertain.value = true
      else {
        pending.value = null
        storePending()
        uncertain.value = false
        for (const sid of selectedIds.value) delete previews[sid]
      }
    } finally {
      submitting.value = false
    }
  }
  watch(
    () => [props.visible, props.workflowId],
    () => {
      if (props.visible && props.workflowId) void load()
      else generation++
    },
    { immediate: true }
  )
</script>

<style scoped>
  .production-body {
    max-height: 69vh;
    overflow-y: auto;
    padding: 0 4px;
  }
  .production-heading,
  .production-card-heading {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
    margin: 8px 0 18px;
  }
  .production-heading > div {
    display: flex;
    align-items: center;
    gap: 14px;
  }
  .production-heading strong {
    font-size: 18px;
  }
  .production-heading small,
  .production-note,
  .production-target {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
  .production-controls {
    display: grid;
    grid-template-columns: 80px 1fr;
    align-items: center;
    gap: 18px 16px;
    margin: 24px 0;
  }
  .production-controls label,
  .production-source label {
    color: var(--el-text-color-regular);
    font-size: 13px;
  }
  .production-source {
    display: flex;
    align-items: center;
    gap: 16px;
    margin: 0 0 12px;
    padding: 12px;
    background: var(--el-fill-color-light);
    border-radius: 6px;
  }
  .production-tools {
    display: flex;
    justify-content: flex-end;
    gap: 4px;
    margin-top: -12px;
  }
  .production-source .el-select {
    width: 180px;
  }
  .production-source label {
    width: 80px;
  }
  .production-service-card {
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    padding: 18px;
    margin: 12px 0;
  }
  .production-card-heading {
    margin: 0 0 14px;
  }
  .production-image-select {
    width: 100%;
  }
  .production-metadata {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 10px;
    font-size: 12px;
    margin-top: 12px;
    color: var(--el-text-color-secondary);
  }
  .production-metadata a {
    color: var(--el-color-primary);
  }
  .production-preview {
    min-height: 24px;
    margin-top: 8px;
  }
  .production-diff {
    font-size: 12px;
  }
  .production-diff summary {
    padding: 8px 0;
    cursor: pointer;
    color: var(--el-text-color-secondary);
  }
  .production-diff p {
    overflow-wrap: anywhere;
  }
  .production-diff p strong,
  .production-diff p span {
    display: block;
  }
  .production-diff del {
    color: var(--el-color-danger);
  }
  .production-diff ins {
    color: var(--el-color-success);
  }
  .production-diff pre {
    max-height: 320px;
    overflow: auto;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    background: var(--el-fill-color-light);
    padding: 12px;
  }
  .production-images {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
    background: var(--el-fill-color-light);
    padding: 12px;
    border-radius: 6px;
  }
  .production-images small,
  .production-images code {
    display: block;
    overflow-wrap: anywhere;
    margin-bottom: 8px;
  }
  .production-build-info {
    padding: 20px;
    background: var(--el-fill-color-light);
    border-radius: 8px;
    font-size: 13px;
    line-height: 1.8;
  }
  @media (max-width: 640px) {
    .production-controls {
      grid-template-columns: 1fr;
      gap: 10px;
    }
    .production-source {
      gap: 8px;
    }
    .production-source label {
      width: auto;
    }
    .production-source .el-select {
      width: 140px;
    }
    .production-service-card {
      padding: 12px;
    }
    .production-images {
      grid-template-columns: 1fr;
    }
  }
</style>
<style>
  .el-select__popper.production-image-dropdown
    .el-select-dropdown__list
    .el-select-dropdown__item.production-image-option {
    height: auto !important;
    min-height: 62px;
    padding-top: 8px !important;
    padding-bottom: 8px !important;
    line-height: 20px !important;
  }
  .production-image-option > div {
    display: flex;
    justify-content: space-between;
    gap: 18px;
  }
  .production-image-option strong {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .production-image-option > div span {
    flex-shrink: 0;
  }
  .production-image-option > div span {
    color: var(--el-color-success);
    font-size: 12px;
  }
  .production-image-option small {
    display: block;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
</style>
