<template>
  <div class="template-source">
    <span v-if="status?.template"
      >{{ status.name || '共享模板' }} · v{{ status.template.revision
      }}{{ status.mode === 'custom' ? ' · 已自定义' : ''
      }}{{ status.archived ? ' · 已归档' : '' }}</span
    >
    <span v-else>自定义 Chart</span>
    <ElButton v-if="canEdit" text size="small" :disabled="disabled" @click="open">{{
      status?.update_available ? '有新模板版本' : '选择模板版本'
    }}</ElButton>
    <ElDialog
      v-model="visible"
      title="更新环境 Chart"
      width="min(1180px, 95vw)"
      append-to-body
      :close-on-click-modal="!working"
      :close-on-press-escape="!working"
      :show-close="!working"
    >
      <ElAlert v-if="error" :title="error" type="error" :closable="false" />
      <p class="note"
        >{{ environment.name }} · {{ environment.namespace }}。保留此环境的
        values、镜像映射和凭据引用；保存只更新配置，不会部署。</p
      >
      <div class="selectors">
        <ElSelect
          :model-value="templateId"
          placeholder="选择共享模板"
          :disabled="working || loading || !!rendered"
          @change="selectTemplate"
          ><ElOption v-for="t in templates" :key="t.id" :value="t.id" :label="t.name"
        /></ElSelect>
        <ElSelect
          :model-value="version"
          placeholder="选择固定版本"
          :disabled="working || loading || !!rendered"
          @change="selectVersion"
          ><ElOption
            v-for="v in versions"
            :key="v.revision"
            :value="v.revision"
            :label="`v${v.revision} · ${v.name}`"
        /></ElSelect>
      </div>
      <p v-if="versions.length === 200" class="note">显示最近 200 个版本。</p>
      <template v-if="!rendered">
        <div v-loading="loading" class="comparison">
          <nav aria-label="模板文件差异"
            ><button
              v-for="f in differences"
              :key="f.path"
              :class="{ active: activePath === f.path }"
              @click="activePath = f.path"
              ><span>{{ f.path }}</span
              ><small>{{ f.change }}</small></button
            ></nav
          >
          <div class="file-diff"
            ><div
              ><strong>当前 Chart</strong><pre>{{ activeDiff?.before || '（不存在）' }}</pre></div
            ><div
              ><strong>选择的模板版本</strong><pre>{{ activeDiff?.after || '（不存在）' }}</pre>
            </div></div
          >
        </div>
        <p v-if="selected && !differences.length" class="note"
          >Chart 文件内容一致，可保持当前版本。</p
        >
        <ElCheckbox v-if="replacement" v-model="replaceConfirmed" :disabled="working"
          >确认使用所选模板替换当前自定义 Chart 或其他模板，当前环境 values 将重新校验</ElCheckbox
        >
      </template>
      <template v-else>
        <p class="note"
          >模板 v{{ version }} 已使用目标环境 values 渲染通过。以下为实际资源预览，Secret
          内容已隐藏。</p
        >
        <YamlEditorPro
          :model-value="rendered"
          readonly
          height="min(57vh, 650px)"
          :show-title-bar="false"
          :show-toolbar="false"
          :show-status-bar="false"
          :enable-highlight="false"
          :auto-format="false"
        />
        <ElCheckbox
          v-if="environment.is_production"
          v-model="productionConfirmed"
          :disabled="working"
          >确认保存到生产环境 {{ environment.name }}</ElCheckbox
        >
      </template>
      <template #footer>
        <ElButton :disabled="working" @click="rendered ? resetPreview() : (visible = false)">{{
          rendered ? '返回比较' : '取消'
        }}</ElButton>
        <ElButton
          v-if="!rendered"
          type="primary"
          :loading="working"
          :disabled="loading || !selected || (replacement && !replaceConfirmed)"
          @click="preview"
          >校验与预览</ElButton
        >
        <ElButton
          v-else
          type="primary"
          :loading="working"
          :disabled="environment.is_production && !productionConfirmed"
          @click="save"
          >保存到此环境</ElButton
        >
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, ref, watch } from 'vue'
  import { ElMessage } from 'element-plus'
  import {
    getHelmTemplateStatusApi,
    listDeliveryTemplatesApi,
    listDeliveryTemplateVersionsApi,
    getDeliveryTemplateVersionApi,
    saveServiceEnvironmentConfigApi,
    type AxiOpsService,
    type AxiOpsEnvironment,
    type ServiceEnvironmentConfig,
    type HelmEnvironmentSettings,
    type HelmTemplateStatus,
    type DeliveryTemplate,
    type DeliveryTemplateVersion,
    type SaveServiceEnvironmentConfig
  } from '@/api/axiops'
  import { chartFiles } from '@/utils/helm-service-config'
  import YamlEditorPro from '@/components/yaml-editor-pro/index.vue'
  const props = defineProps<{
    service: AxiOpsService
    environment: AxiOpsEnvironment
    config: ServiceEnvironmentConfig
    settings: HelmEnvironmentSettings
    revisionId: number
    canEdit: boolean
    disabled: boolean
  }>()
  const emit = defineEmits<{ saved: []; 'busy-change': [value: boolean] }>()
  const status = ref<HelmTemplateStatus>(),
    visible = ref(false),
    loading = ref(false),
    working = ref(false),
    error = ref('')
  const templates = ref<DeliveryTemplate[]>([]),
    versions = ref<DeliveryTemplateVersion[]>([]),
    templateId = ref<number>(),
    version = ref<number>(),
    selected = ref<DeliveryTemplate>(),
    activePath = ref('')
  const replaceConfirmed = ref(false),
    productionConfirmed = ref(false),
    rendered = ref(''),
    pending = ref<SaveServiceEnvironmentConfig>(),
    previewStamp = ref('')
  let generation = 0,
    selection = 0
  const stamp = () =>
    JSON.stringify([props.service.id, props.config.token, props.revisionId, props.settings])
  const differences = computed(() => {
    if (!selected.value) return []
    const before = new Map(chartFiles(props.config.definition).map((f) => [f.path, f.content])),
      after = new Map(selected.value.files.map((f) => [f.path, f.content]))
    return [...new Set([...before.keys(), ...after.keys()])]
      .sort()
      .filter((p) => before.get(p) !== after.get(p))
      .map((path) => ({
        path,
        before: before.get(path),
        after: after.get(path),
        change: !before.has(path) ? '新增' : !after.has(path) ? '删除' : '修改'
      }))
  })
  const activeDiff = computed(() => differences.value.find((f) => f.path === activePath.value))
  const replacement = computed(
    () => status.value?.mode !== 'template' || status.value.template?.id !== templateId.value
  )
  function resetPreview() {
    rendered.value = ''
    pending.value = undefined
    productionConfirmed.value = false
  }
  async function loadStatus() {
    const current = ++generation
    ++selection
    visible.value = false
    loading.value = false
    working.value = false
    status.value = undefined
    resetPreview()
    try {
      const result = await getHelmTemplateStatusApi(props.service.id, props.revisionId)
      if (current === generation) status.value = result
    } catch {
      /* Library access is optional for an existing Chart. */
    }
  }
  async function open() {
    if (props.disabled || !props.canEdit || loading.value || working.value) return
    const current = generation
    visible.value = true
    error.value = ''
    loading.value = true
    selected.value = undefined
    versions.value = []
    templateId.value = undefined
    version.value = undefined
    replaceConfirmed.value = false
    resetPreview()
    try {
      const result = await listDeliveryTemplatesApi('helm')
      if (current !== generation) return
      templates.value = result || []
      const first =
        templates.value.find((t) => t.id === status.value?.template?.id) || templates.value[0]
      if (first) await selectTemplate(first.id)
    } catch (e: any) {
      if (current === generation) error.value = e?.message || '模板读取失败'
    } finally {
      if (current === generation) loading.value = false
    }
  }
  async function selectTemplate(id: number) {
    const current = generation,
      request = ++selection
    templateId.value = id
    version.value = undefined
    selected.value = undefined
    versions.value = []
    replaceConfirmed.value = false
    resetPreview()
    loading.value = true
    error.value = ''
    try {
      const list = await listDeliveryTemplateVersionsApi(id)
      if (current !== generation || request !== selection) return
      versions.value = list || []
      if (versions.value[0]) await selectVersion(versions.value[0].revision)
    } catch (e: any) {
      if (current === generation && request === selection)
        error.value = e?.message || '版本读取失败'
    } finally {
      if (current === generation && request === selection) loading.value = false
    }
  }
  async function selectVersion(revision: number) {
    if (!templateId.value) return
    const id = templateId.value,
      current = generation,
      request = ++selection
    version.value = revision
    selected.value = undefined
    loading.value = true
    resetPreview()
    error.value = ''
    try {
      const result = await getDeliveryTemplateVersionApi(id, revision)
      if (current !== generation || request !== selection) return
      selected.value = result.template
      activePath.value = differences.value[0]?.path || ''
    } catch (e: any) {
      if (current === generation && request === selection)
        error.value = e?.message || '模板版本读取失败'
    } finally {
      if (current === generation && request === selection) loading.value = false
    }
  }
  async function preview() {
    if (
      !props.canEdit ||
      props.disabled ||
      !selected.value ||
      !templateId.value ||
      !version.value ||
      working.value ||
      loading.value ||
      (replacement.value && !replaceConfirmed.value)
    )
      return
    const current = generation
    working.value = true
    error.value = ''
    const input = stamp()
    try {
      const request: SaveServiceEnvironmentConfig = {
        template: { id: templateId.value, revision: version.value },
        helm: JSON.parse(JSON.stringify(props.settings)),
        targets: [{ environment_id: props.config.environment_id, token: props.config.token }],
        revision_id: 0,
        variables: {}
      }
      const result = await saveServiceEnvironmentConfigApi(props.service.id, request, true)
      if (current !== generation) return
      if (input !== stamp()) throw Error('配置已变化，请重新比较和预览')
      const target = result.configs?.find(
        (c) => c.environment_id === props.config.environment_id && c.service_id === props.service.id
      )
      if (!result.validated || result.saved || !target?.manifest)
        throw Error('预览结果无效，请更新 CI/CD API 后重试')
      rendered.value = target.manifest
      pending.value = request
      previewStamp.value = input
    } catch (e: any) {
      if (current === generation) error.value = e?.message || '模板与环境参数不兼容'
    } finally {
      if (current === generation) working.value = false
    }
  }
  async function save() {
    if (
      !props.canEdit ||
      props.disabled ||
      !pending.value ||
      working.value ||
      (props.environment.is_production && !productionConfirmed.value)
    )
      return
    const current = generation
    working.value = true
    error.value = ''
    try {
      if (previewStamp.value !== stamp()) throw Error('配置已变化，请重新预览')
      const result = await saveServiceEnvironmentConfigApi(props.service.id, pending.value)
      if (current !== generation) return
      if (
        !result.saved ||
        !result.validated ||
        !result.configs?.some(
          (c) =>
            c.environment_id === props.config.environment_id && c.service_id === props.service.id
        )
      )
        throw Error('保存结果未确认，请重新加载查看')
      visible.value = false
      ElMessage.success('此环境已更新 Chart，其他环境保持不变')
      emit('saved')
    } catch (e: any) {
      if (current === generation) error.value = e?.message || '模板更新失败'
    } finally {
      if (current === generation) working.value = false
    }
  }
  watch(() => [props.service.id, props.config.token, props.revisionId], loadStatus, {
    immediate: true
  })
  watch(working, (value) => emit('busy-change', value))
  onBeforeUnmount(() => {
    emit('busy-change', false)
    generation++
    selection++
  })
</script>

<style scoped lang="scss">
  .template-source {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
  .note {
    font-size: 12px;
    line-height: 1.8;
    color: var(--el-text-color-secondary);
  }
  .selectors {
    display: grid;
    grid-template-columns: 2fr 1fr;
    gap: 12px;
    margin: 16px 0;
  }
  .comparison {
    display: grid;
    grid-template-columns: 220px minmax(0, 1fr);
    height: min(53vh, 600px);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    overflow: hidden;
    margin-bottom: 12px;
  }
  nav {
    overflow: auto;
    border-right: 1px solid var(--el-border-color-lighter);
    padding: 6px;
  }
  nav button {
    display: flex;
    flex-direction: column;
    gap: 4px;
    text-align: left;
    width: 100%;
    border: 0;
    border-radius: 4px;
    background: transparent;
    color: var(--el-text-color-regular);
    padding: 9px;
    cursor: pointer;
    overflow-wrap: anywhere;
    font-size: 12px;
  }
  nav button.active {
    background: var(--el-color-primary-light-9);
    color: var(--el-color-primary);
  }
  nav small {
    color: var(--el-text-color-secondary);
  }
  .file-diff {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    min-height: 0;
  }
  .file-diff > div {
    overflow: auto;
    padding: 12px;
    border-right: 1px solid var(--el-border-color-lighter);
    font-size: 12px;
  }
  pre {
    font-family: var(--el-font-family-monospace, monospace);
    line-height: 1.65;
    tab-size: 2;
  }
  @media (max-width: 760px) {
    .comparison {
      grid-template-columns: 1fr;
      height: auto;
    }
    nav {
      max-height: 130px;
    }
    .file-diff {
      grid-template-columns: 1fr;
    }
    .file-diff > div {
      max-height: 260px;
    }
  }
</style>
