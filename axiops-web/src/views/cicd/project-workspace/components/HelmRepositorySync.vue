<template>
  <div v-if="binding || service.source_type === 'git' || visible" class="repository-source">
    <span :title="binding ? `${binding.repo_path} · ${binding.chart_path}` : service.repo_url">
      <GitBranch :size="14" />
      {{ binding ? `${binding.ref} · ${binding.commit_id.slice(0, 8)}` : 'Git Chart' }}
      <small v-if="binding && latestRevision?.id !== binding.revision_id">平台自定义版本</small>
    </span>
    <ElButton
      v-if="canEdit && service.code_source_id"
      text
      size="small"
      :disabled="disabled"
      @click="open"
      >检查仓库更新</ElButton
    >
    <ElDialog
      v-model="visible"
      title="从 Git 更新 Chart"
      width="min(1180px, 95vw)"
      append-to-body
      :close-on-click-modal="!working"
      :close-on-press-escape="!working"
      :show-close="!working"
    >
      <ElAlert v-if="error" :title="error" type="error" :closable="false" />
      <p class="note"
        >读取原生 Chart，比较后另存一个版本。各环境的
        values、当前使用版本和运行资源保留；后续选择新版本并预览发布。</p
      >
      <div class="repository-inputs">
        <ElFormItem label="仓库路径"
          ><ElInput v-model="form.repo_path" :disabled="working" placeholder="team/repository"
        /></ElFormItem>
        <ElFormItem label="分支 / Tag / Commit"
          ><ElInput v-model="form.ref" :disabled="working" placeholder="main"
        /></ElFormItem>
        <ElFormItem label="Chart 目录"
          ><ElInput
            v-model="form.chart_path"
            :disabled="working"
            placeholder="deploy/chart；根目录 ."
        /></ElFormItem>
      </div>
      <template v-if="preview">
        <div class="comparison-summary"
          ><strong>{{ preview.metadata?.name }} · {{ preview.metadata?.version }}</strong
          ><span
            >Commit {{ preview.commit_id.slice(0, 10) }} · {{ preview.file_count }} 个文件 ·
            {{ changes }} 项变更</span
          ></div
        >
        <div class="comparison">
          <nav aria-label="仓库 Chart 文件差异"
            ><button
              v-for="item in differences"
              :key="item.path"
              :class="{ active: activePath === item.path }"
              @click="activePath = item.path"
              ><span>{{ item.path }}</span
              ><small>{{ item.change }}</small></button
            ></nav
          >
          <div class="diff"
            ><section
              ><strong>平台最新 Chart</strong
              ><pre>{{ activeDiff?.before ?? '（不存在）' }}</pre></section
            ><section
              ><strong>Git Chart</strong><pre>{{ activeDiff?.after ?? '（不存在）' }}</pre>
            </section></div
          >
        </div>
        <ElCheckbox v-if="preview.local_changes" v-model="acceptLocal" :disabled="working"
          >已检查平台改动，使用 Git 内容另存新版本；平台旧版本继续保留</ElCheckbox
        >
      </template>
      <template #footer>
        <ElButton :disabled="working" @click="visible = false">取消</ElButton>
        <ElButton :disabled="working || !valid" :loading="working" @click="check"
          >读取并比较</ElButton
        >
        <ElButton
          type="primary"
          :disabled="working || !preview || !changes || (preview.local_changes && !acceptLocal)"
          :loading="working"
          @click="save"
          >导入为新版本</ElButton
        >
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
  import { ElMessage } from 'element-plus'
  import { GitBranch } from 'lucide-vue-next'
  import {
    importServiceRevisionApi,
    previewServiceImportApi,
    type AxiOpsService,
    type AxiOpsServiceRevision,
    type ServiceImportPreview,
    type ServiceImportRequest
  } from '@/api/axiops'
  import { chartFiles, chartDifferences } from '@/utils/helm-service-config'
  const props = defineProps<{
    service: AxiOpsService
    latestRevision?: AxiOpsServiceRevision
    canEdit: boolean
    disabled: boolean
  }>()
  const emit = defineEmits<{
    imported: [revision: AxiOpsServiceRevision]
    'busy-change': [busy: boolean]
  }>()
  const binding = computed(() => props.service.source_ref?.helm_repository)
  const visible = ref(false),
    working = ref(false),
    error = ref(''),
    acceptLocal = ref(false),
    activePath = ref('')
  const form = reactive({ repo_path: '', ref: 'main', chart_path: '.' })
  const preview = ref<ServiceImportPreview>()
  const checked = ref<ServiceImportRequest>()
  let generation = 0
  const valid = computed(() => Object.values(form).every((s) => s.trim()))
  const differences = computed(() => {
    try {
      return chartDifferences(
        chartFiles(props.latestRevision?.definition || ''),
        preview.value?.files || []
      )
    } catch {
      return []
    }
  })
  const changes = computed(() => differences.value.filter((f) => f.change !== '未变').length)
  const activeDiff = computed(() => differences.value.find((f) => f.path === activePath.value))
  function open() {
    if (props.disabled || !props.canEdit || !props.service.code_source_id) return
    const source = binding.value
    let repo = props.service.repo_url
    try {
      repo = new URL(repo).pathname.replace(/^\/+|\/+$/g, '').replace(/\.git$/, '')
    } catch {
      /* provider path */
    }
    let chartPath = '.'
    try {
      chartPath = JSON.parse(props.latestRevision?.definition || '{}').chart_path || '.'
    } catch {
      /* incomplete Chart */
    }
    Object.assign(form, {
      repo_path: source?.repo_path || repo,
      ref: source?.ref || props.service.repo_branch || 'main',
      chart_path: source?.chart_path || chartPath
    })
    preview.value = undefined
    checked.value = undefined
    error.value = ''
    acceptLocal.value = false
    visible.value = true
  }
  async function check() {
    if (!valid.value || working.value || !props.canEdit) return
    const token = ++generation
    working.value = true
    error.value = ''
    preview.value = undefined
    acceptLocal.value = false
    try {
      const request = {
        code_source_id: props.service.code_source_id,
        repo_path: form.repo_path.trim(),
        ref: form.ref.trim(),
        chart_path: form.chart_path.trim(),
        paths: []
      }
      const result = await previewServiceImportApi(props.service, request)
      if (token !== generation) return
      if (!result.files?.length || result.current_revision_id === undefined)
        throw Error('Chart 预览不完整，请更新 CI/CD API 后重试')
      if (result.current_revision_id !== (props.latestRevision?.id || 0))
        throw Error('平台 Chart 已更新，请关闭后刷新再比较')
      preview.value = result
      checked.value = request
      activePath.value =
        differences.value.find((f) => f.change !== '未变')?.path || differences.value[0]?.path || ''
    } catch (e: any) {
      if (token === generation) error.value = e.message || 'Chart 读取失败，请核对目录、依赖和权限'
    } finally {
      if (token === generation) working.value = false
    }
  }
  async function save() {
    if (
      working.value ||
      !props.canEdit ||
      !preview.value ||
      !checked.value ||
      !changes.value ||
      (preview.value.local_changes && !acceptLocal.value)
    )
      return
    const token = generation
    working.value = true
    error.value = ''
    try {
      const result = await importServiceRevisionApi(props.service, {
        ...checked.value,
        expected_commit: preview.value.commit_id,
        expected_revision_id: preview.value.current_revision_id,
        accept_local_changes: acceptLocal.value
      })
      if (token !== generation) return
      visible.value = false
      ElMessage.success('已导入新 Chart 版本，请选择版本并预览保存环境配置')
      emit('imported', result)
    } catch (e: any) {
      if (token === generation) {
        error.value = e.message || '导入失败，请重新比较'
        preview.value = undefined
        checked.value = undefined
      }
    } finally {
      if (token === generation) working.value = false
    }
  }
  watch(
    () => [form.repo_path, form.ref, form.chart_path],
    () => {
      generation++
      preview.value = undefined
      checked.value = undefined
      acceptLocal.value = false
    }
  )
  watch(
    () => props.service.id,
    () => {
      generation++
      visible.value = false
      working.value = false
      preview.value = undefined
    }
  )
  watch(visible, (value) => {
    if (!value) {
      generation++
      working.value = false
    }
    emit('busy-change', value)
  })
  onBeforeUnmount(() => {
    generation++
    emit('busy-change', false)
  })
  defineExpose({ open })
</script>

<style scoped>
  .repository-source,
  .repository-source > span,
  .comparison-summary {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
  }
  .repository-source {
    padding: 0 18px 12px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
  .repository-source small {
    color: var(--el-color-warning);
  }
  .note,
  .comparison-summary span {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.8;
  }
  .repository-inputs {
    display: grid;
    grid-template-columns: 1.2fr 1fr 1.2fr;
    gap: 16px;
    margin-top: 16px;
  }
  .comparison-summary {
    margin: 10px 0;
  }
  .comparison {
    display: grid;
    grid-template-columns: 220px minmax(0, 1fr);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    overflow: hidden;
    margin: 12px 0;
  }
  nav {
    max-height: 460px;
    overflow: auto;
    border-right: 1px solid var(--el-border-color-light);
    padding: 6px;
  }
  nav button {
    display: flex;
    width: 100%;
    justify-content: space-between;
    align-items: center;
    gap: 8px;
    padding: 9px;
    text-align: left;
    border: 0;
    background: transparent;
    color: var(--el-text-color-primary);
    cursor: pointer;
  }
  nav button span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  nav button.active {
    background: var(--el-color-primary-light-9);
    color: var(--el-color-primary);
    border-radius: 5px;
  }
  nav small {
    flex-shrink: 0;
    font-size: 11px;
  }
  .diff {
    display: grid;
    grid-template-columns: 1fr 1fr;
    min-width: 0;
  }
  .diff section {
    min-width: 0;
    border-right: 1px solid var(--el-border-color-lighter);
  }
  .diff strong {
    display: block;
    padding: 10px 12px;
    font-size: 12px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  pre {
    margin: 0;
    padding: 12px;
    height: 380px;
    overflow: auto;
    font: 12px/1.7 var(--ax-font-mono, monospace);
    tab-size: 2;
  }
  @media (max-width: 760px) {
    .repository-inputs {
      grid-template-columns: 1fr;
      gap: 0;
    }
    .comparison {
      grid-template-columns: 1fr;
    }
    nav {
      max-height: 150px;
    }
    .diff {
      grid-template-columns: 1fr;
    }
    pre {
      height: 240px;
    }
  }
</style>
