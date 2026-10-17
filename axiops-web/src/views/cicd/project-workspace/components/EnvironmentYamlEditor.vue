<template>
  <section class="environment-yaml-editor">
    <nav class="file-index" aria-label="环境 YAML 文件">
      <button
        v-for="file in files"
        :key="file.name"
        type="button"
        :class="{ active: activeFile === file.name }"
        @click="locate(file.name)"
        >{{ file.name }}</button
      >
    </nav>
    <YamlEditorPro
      ref="editor"
      :model-value="modelValue"
      :readonly="readonly"
      height="min(65vh, 720px)"
      theme="light"
      :show-title-bar="false"
      :show-toolbar="false"
      :show-status-bar="false"
      :show-line-numbers="true"
      :enable-highlight="false"
      :auto-format="false"
      @update:model-value="emit('update:modelValue', $event)"
    />
    <ElDialog v-model="newFileVisible" title="新增当前环境 YAML" width="640px" append-to-body>
      <ElInput v-model="newName" placeholder="例如 configmap.yaml" maxlength="128" />
      <ElInput
        v-model="newContent"
        class="new-file-content"
        type="textarea"
        :rows="12"
        placeholder="粘贴完整 YAML；只添加到当前环境草稿"
      />
      <p class="hint">保留已有文件，保存前会一起预览和校验。</p>
      <template #footer
        ><ElButton @click="newFileVisible = false">取消</ElButton
        ><ElButton type="primary" @click="addFile">加入草稿</ElButton></template
      >
    </ElDialog>
  </section>
</template>

<script setup lang="ts">
  import { computed, nextTick, ref, watch } from 'vue'
  import { ElMessage } from 'element-plus'
  import YamlEditorPro from '@/components/yaml-editor-pro/index.vue'
  const props = defineProps<{ modelValue: string; readonly?: boolean; activeFile?: string }>()
  const emit = defineEmits<{
    'update:modelValue': [value: string]
    'file-change': [value: string]
  }>()
  const editor = ref<InstanceType<typeof YamlEditorPro>>()
  const newFileVisible = ref(false),
    newName = ref(''),
    newContent = ref('')
  const files = computed(() => {
    const matches = [...props.modelValue.matchAll(/^# AxiOps-File: (.+)$/gm)]
    return matches.length
      ? matches.map((m) => ({ name: m[1].trim(), start: m.index || 0 }))
      : [{ name: 'deployment.yaml', start: 0 }]
  })
  async function locate(name: string, notify = true) {
    const i = files.value.findIndex((f) => f.name === name)
    if (i < 0) return
    const file = files.value[i],
      end = files.value[i + 1]?.start ?? props.modelValue.length
    const part = props.modelValue.slice(file.start, end),
      kind = part.search(/^kind\s*:/m)
    const offset = file.start + Math.max(0, kind)
    await nextTick()
    editor.value?.gotoLine(props.modelValue.slice(0, offset).split('\n').length, { focus: true })
    if (notify) emit('file-change', name)
  }
  function addFile() {
    const name = newName.value.trim()
    if (
      !/^[A-Za-z0-9][A-Za-z0-9_.-]*\.ya?ml$/.test(name) ||
      files.value.some((f) => f.name === name)
    )
      return void ElMessage.warning('请输入未使用的 YAML 文件名')
    if (!newContent.value.trim()) return void ElMessage.warning('请填写 YAML 内容')
    const original = /^# AxiOps-File:/m.test(props.modelValue)
      ? props.modelValue.trimEnd()
      : `# AxiOps-File: deployment.yaml\n${props.modelValue.trimEnd()}`
    emit(
      'update:modelValue',
      `${original}\n---\n# AxiOps-File: ${name}\n${newContent.value.trim()}\n`
    )
    newFileVisible.value = false
    void nextTick(() => locate(name))
  }
  watch(
    () => props.activeFile,
    (name) => {
      if (name) void locate(name, false)
    }
  )
  defineExpose({
    openNewYaml() {
      if (!props.readonly) {
        newName.value = ''
        newContent.value = ''
        newFileVisible.value = true
      }
    }
  })
</script>

<style scoped>
  .file-index {
    display: flex;
    gap: 4px;
    overflow-x: auto;
    padding: 4px 0 10px;
  }
  .file-index button {
    white-space: nowrap;
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 4px;
    padding: 5px 9px;
    color: #475569;
    cursor: pointer;
    font-size: 12px;
  }
  .file-index button.active {
    color: #2563eb;
    border-color: #bfdbfe;
    background: #eff6ff;
  }
  .new-file-content {
    margin-top: 12px;
  }
  .hint {
    color: #64748b;
    font-size: 12px;
  }
</style>
