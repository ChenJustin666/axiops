<template>
  <div class="values-sources">
    <ElButton text size="small" :disabled="disabled" @click="open">导入 values</ElButton>
    <template v-if="modelValue.source">
      <small :title="`${modelValue.source.repo_path} · ${modelValue.source.path}`"
        >Git · {{ modelValue.source.path }} @ {{ modelValue.source.commit_id.slice(0, 8) }}</small
      >
      <ElButton text size="small" :disabled="disabled" @click="refresh">检查更新</ElButton>
    </template>
    <ElDialog
      v-model="visible"
      title="导入当前环境 values"
      width="min(960px, 94vw)"
      append-to-body
      :close-on-click-modal="!working"
      :show-close="!working"
    >
      <ElAlert v-if="error" :title="error" type="error" :closable="false" />
      <p class="note"
        >只修改当前草稿，预览并保存后生效。Chart 来源与 values 来源独立；敏感信息请引用安全凭据。</p
      >
      <ElTabs v-model="mode">
        <ElTabPane label="Chart 文件" name="chart">
          <ElSelect
            v-model="filePath"
            placeholder="选择固定 Chart 中的 values 文件"
            style="width: 100%"
            @change="chooseFile"
          >
            <ElOption
              v-for="file in valueFiles"
              :key="file.path"
              :value="file.path"
              :label="file.path"
            />
          </ElSelect>
        </ElTabPane>
        <ElTabPane label="代码仓库" name="git">
          <div class="source-form">
            <ElSelect v-model="form.code_source_id" placeholder="代码源" :disabled="working"
              ><ElOption v-for="s in sources" :key="s.id" :value="s.id" :label="s.name"
            /></ElSelect>
            <ElInput
              v-model="form.repo_path"
              placeholder="仓库路径，如 team/config"
              :disabled="working"
            />
            <ElInput v-model="form.ref" placeholder="分支 / Tag / Commit" :disabled="working" />
            <ElInput
              v-model="form.path"
              placeholder="如 deploy/values-prod.yaml"
              :disabled="working"
            />
          </div>
          <ElButton :loading="working" :disabled="!form.code_source_id" @click="readGit"
            >读取并比较</ElButton
          >
        </ElTabPane>
        <ElTabPane v-if="environments.length" label="参考已有环境" name="environment">
          <ElSelect
            v-model="referenceId"
            placeholder="选择参考环境"
            style="width: 100%"
            :disabled="working"
            @change="readEnvironment"
            ><ElOption
              v-for="env in environments"
              :key="env.id"
              :value="env.id"
              :label="`${env.name} · ${env.namespace}`"
          /></ElSelect>
          <p class="note"
            >复制服务的环境覆盖，目标环境共享配置和凭据独立。请核对目标环境的域名、配置中心、数据地址和
            Secret 引用。</p
          >
        </ElTabPane>
      </ElTabs>
      <template v-if="candidate !== undefined">
        <p v-if="mergeResult.conflicts.length" class="note"
          >以下参数在平台和 Git 都有修改，请逐项选择；数组按整项处理。</p
        >
        <div v-for="conflict in mergeResult.conflicts" :key="conflict.path" class="conflict">
          <strong>{{ valuesPathLabel(conflict.path) || '全部 values' }}</strong>
          <div class="compare">
            <pre>平台：{{ conflict.local }}</pre
            ><pre>Git：{{ conflict.remote }}</pre>
          </div>
          <ElRadioGroup v-model="choices[conflict.path]"
            ><ElRadio value="local">保留平台</ElRadio
            ><ElRadio value="remote">采用 Git</ElRadio></ElRadioGroup
          >
        </div>
        <p class="note">导入后的环境 values</p>
        <YamlEditorPro
          :model-value="mergeResult.values"
          readonly
          height="min(42vh, 460px)"
          :show-title-bar="false"
          :show-toolbar="false"
          :auto-format="false"
        />
      </template>
      <template #footer
        ><ElButton :disabled="working" @click="visible = false">取消</ElButton
        ><ElButton
          type="primary"
          :disabled="disabled || working || candidate === undefined || unresolved"
          @click="apply"
          >应用到草稿</ElButton
        ></template
      >
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue'
  import {
    getServiceEnvironmentConfigApi,
    listCodeSourcesApi,
    readHelmValuesApi,
    type AxiOpsCodeSource,
    type AxiOpsEnvironment,
    type AxiOpsService,
    type HelmEnvironmentSettings,
    type HelmValuesSource
  } from '@/api/axiops'
  import {
    mergeHelmValuesSource,
    valuesObject,
    valuesPathLabel,
    type ChartFile
  } from '@/utils/helm-service-config'
  import YamlEditorPro from '@/components/yaml-editor-pro/index.vue'
  const props = defineProps<{
    modelValue: HelmEnvironmentSettings
    service: AxiOpsService
    files: ChartFile[]
    environments: AxiOpsEnvironment[]
    disabled?: boolean
  }>()
  const emit = defineEmits<{
    'update:modelValue': [value: HelmEnvironmentSettings]
    'busy-change': [busy: boolean]
  }>()
  const visible = ref(false),
    mode = ref('chart'),
    working = ref(false),
    error = ref(''),
    filePath = ref(''),
    referenceId = ref<number>(),
    candidate = ref<string>(),
    remote = ref<HelmValuesSource>(),
    local = ref(''),
    baseline = ref(''),
    threeWay = ref(false)
  const choices = reactive<Record<string, 'local' | 'remote'>>({}),
    sources = ref<AxiOpsCodeSource[]>([])
  const form = reactive({ code_source_id: 0, repo_path: '', ref: 'main', path: 'values.yaml' })
  const valueFiles = computed(() =>
    props.files.filter(
      (f) =>
        /\.ya?ml$/i.test(f.path) &&
        !/(^|\/)templates\//.test(f.path) &&
        !/(^|\/)(Chart|Chart.lock)\./.test(f.path)
    )
  )
  const mergeResult = computed(() =>
    threeWay.value && candidate.value !== undefined
      ? mergeHelmValuesSource(baseline.value, local.value, candidate.value, choices)
      : { values: candidate.value || '{}\n', conflicts: [] }
  )
  const unresolved = computed(() => mergeResult.value.conflicts.some((c) => !choices[c.path]))
  let generation = 0,
    disposed = false
  function resetPreview() {
    candidate.value = undefined
    remote.value = undefined
    threeWay.value = false
    error.value = ''
    Object.keys(choices).forEach((k) => delete choices[k])
  }
  async function open() {
    if (props.disabled) return
    const token = ++generation
    resetPreview()
    local.value = props.modelValue.values
    mode.value = 'chart'
    filePath.value = ''
    referenceId.value = undefined
    const src = props.modelValue.source
    let repo = props.service.repo_url || ''
    try {
      repo = new URL(repo).pathname.replace(/^\/+|\/+$/g, '').replace(/\.git$/, '')
    } catch {
      /* provider path */
    }
    Object.assign(
      form,
      src || {
        code_source_id: props.service.code_source_id || 0,
        repo_path: props.service.source_ref?.helm_repository?.repo_path || repo,
        ref: props.service.repo_branch || 'main',
        path: 'values.yaml'
      }
    )
    visible.value = true
    try {
      const list = await listCodeSourcesApi(props.service.project_id)
      if (token === generation && visible.value) sources.value = list
    } catch {
      if (token === generation) error.value = '代码源列表加载失败，仍可从 Chart 或已有环境导入'
    }
  }
  function chooseFile() {
    try {
      candidate.value = props.files.find((f) => f.path === filePath.value)?.content
      if (candidate.value !== undefined) valuesObject(candidate.value)
    } catch (e: any) {
      candidate.value = undefined
      error.value = e.message
    }
  }
  async function readGit() {
    if (working.value || props.disabled) return
    const token = ++generation,
      input = JSON.stringify(form)
    resetPreview()
    working.value = true
    try {
      const result = await readHelmValuesApi(props.service.project_id, props.service.id, {
        ...form
      })
      if (token !== generation || !visible.value || input !== JSON.stringify(form)) return
      const previous = props.modelValue.source
      const useMerge =
        !!previous &&
        ['code_source_id', 'repo_path', 'ref', 'path'].every(
          (k) => previous[k as keyof HelmValuesSource] === result[k as keyof HelmValuesSource]
        )
      const base = previous?.base || '{}\n'
      if (useMerge) mergeHelmValuesSource(base, local.value, result.base)
      valuesObject(result.base)
      baseline.value = base
      threeWay.value = useMerge
      remote.value = result
      candidate.value = result.base
    } catch (e: any) {
      if (token === generation) error.value = e.message || '读取失败'
    } finally {
      if (token === generation) working.value = false
    }
  }
  async function refresh() {
    await open()
    if (!props.modelValue.source) return
    mode.value = 'git'
    await nextTick()
    await readGit()
  }
  async function readEnvironment() {
    if (!referenceId.value || working.value) return
    const token = ++generation
    resetPreview()
    working.value = true
    try {
      const config = await getServiceEnvironmentConfigApi(referenceId.value, props.service.id)
      if (token === generation && visible.value) {
        if (!config.bound) throw Error('此服务尚未配置到参考环境')
        candidate.value = config.helm?.values || '{}\n'
      }
    } catch (e: any) {
      if (token === generation) error.value = e.message
    } finally {
      if (token === generation) working.value = false
    }
  }
  function apply() {
    if (
      disposed ||
      props.disabled ||
      working.value ||
      unresolved.value ||
      candidate.value === undefined
    )
      return
    if (props.modelValue.values !== local.value) {
      error.value = '当前草稿已变化，请重新导入比较'
      return
    }
    const next = { ...props.modelValue, values: mergeResult.value.values }
    delete next.source
    if (remote.value) next.source = { ...remote.value }
    emit('update:modelValue', next)
    visible.value = false
  }
  watch(form, resetPreview)
  watch(mode, () => {
    generation++
    working.value = false
    resetPreview()
  })
  watch(working, (v) => emit('busy-change', v))
  watch(visible, (v) => {
    if (!v) {
      generation++
      working.value = false
    }
  })
  watch(
    () => props.service.id,
    () => {
      generation++
      visible.value = false
      working.value = false
    }
  )
  onBeforeUnmount(() => {
    generation++
    disposed = true
    emit('busy-change', false)
  })
</script>

<style scoped>
  .values-sources {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }
  .values-sources small,
  .note {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    line-height: 1.7;
  }
  .source-form {
    display: grid;
    grid-template-columns: 1fr 2fr;
    gap: 12px;
    margin-bottom: 16px;
  }
  .compare {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }
  .compare pre {
    white-space: pre-wrap;
    overflow: auto;
    background: var(--el-fill-color-lighter);
    padding: 12px;
    max-height: 180px;
  }
  .conflict {
    border-bottom: 1px solid var(--el-border-color-light);
    padding: 12px 0;
  }
  @media (max-width: 700px) {
    .source-form,
    .compare {
      grid-template-columns: 1fr;
    }
  }
</style>
