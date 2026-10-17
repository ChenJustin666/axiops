<template>
  <ElButton text size="small" :disabled="disabled" @click="open">环境共享 values</ElButton>
  <ElDialog
    v-model="visible"
    :title="`${environment.name} · 共享 values`"
    width="min(1080px, 95vw)"
    append-to-body
    :close-on-click-modal="!working"
    :close-on-press-escape="!working"
    :show-close="!working"
  >
    <ElAlert v-if="error" :title="error" type="error" :closable="false" />
    <p class="note"
      >仅作用于 {{ environment.name }}。优先级：Chart 默认值 ＜ 继承路径参数 ＜ 环境共享 values ＜
      服务环境 values。凭据单独绑定，保存不会部署。</p
    >
    <YamlEditorPro
      v-model="values"
      :readonly="working || !canEdit || !!preview"
      height="min(45vh, 480px)"
      :show-title-bar="false"
      :show-toolbar="false"
      :auto-format="false"
    />
    <template v-if="preview">
      <p class="note"
        >已校验 {{ preview.services.length }} 个服务，{{
          preview.services.filter((s) => s.changed).length
        }}
        个最终配置发生变化。自身覆盖了共享值的服务可能不受影响。</p
      >
      <ElCollapse
        ><ElCollapseItem
          v-for="s in preview.services"
          :key="s.service_id"
          :name="s.service_id"
          :title="`${s.name} · ${s.changed ? '配置将变化' : '配置未变化'}`"
          ><div class="comparison">
            <pre>{{ s.before || '原配置无法渲染' }}</pre
            ><pre>{{ s.after }}</pre>
          </div></ElCollapseItem
        ></ElCollapse
      >
      <ElCheckbox v-if="environment.is_production" v-model="confirmed" :disabled="working"
        >确认保存到生产环境 {{ environment.name }}</ElCheckbox
      >
    </template>
    <template #footer>
      <ElButton :disabled="working" @click="preview ? (preview = undefined) : (visible = false)">{{
        preview ? '返回修改' : '关闭'
      }}</ElButton>
      <ElButton
        v-if="canEdit"
        type="primary"
        :loading="working"
        :disabled="!loaded || (!!preview && environment.is_production && !confirmed)"
        @click="preview ? save() : check()"
        >{{ preview ? '保存共享配置' : '校验影响范围' }}</ElButton
      >
    </template>
  </ElDialog>
</template>
<script setup lang="ts">
  import { ref, watch } from 'vue'
  import { ElMessage } from 'element-plus'
  import {
    getHelmSharedValuesApi,
    saveHelmSharedValuesApi,
    type AxiOpsEnvironment,
    type HelmSharedValues
  } from '@/api/axiops'
  import YamlEditorPro from '@/components/yaml-editor-pro/index.vue'
  const props = defineProps<{
    environment: AxiOpsEnvironment
    canEdit: boolean
    disabled?: boolean
  }>()
  const emit = defineEmits<{ saved: []; 'busy-change': [busy: boolean] }>()
  const visible = ref(false),
    working = ref(false),
    values = ref(''),
    error = ref(''),
    confirmed = ref(false),
    loaded = ref<HelmSharedValues>(),
    preview = ref<HelmSharedValues>()
  let generation = 0,
    checked = ''
  const request = () => ({
    values: values.value,
    revision: loaded.value!.revision,
    tokens: Object.fromEntries(loaded.value!.services.map((s) => [s.service_id, s.token]))
  })
  async function open() {
    if (props.disabled || working.value) return
    const token = ++generation
    working.value = true
    visible.value = true
    loaded.value = undefined
    preview.value = undefined
    error.value = ''
    values.value = ''
    confirmed.value = false
    try {
      const result = await getHelmSharedValuesApi(props.environment.id)
      if (token === generation) {
        loaded.value = result
        values.value = result.values
      }
    } catch (e: any) {
      if (token === generation) error.value = e.message || '共享配置读取失败'
    } finally {
      if (token === generation) working.value = false
    }
  }
  async function check() {
    if (!loaded.value || working.value || !props.canEdit) return
    const token = generation
    working.value = true
    error.value = ''
    confirmed.value = false
    try {
      const input = request()
      const result = await saveHelmSharedValuesApi(props.environment.id, input, true)
      if (token === generation) {
        preview.value = result
        checked = JSON.stringify(input)
      }
    } catch (e: any) {
      if (token === generation) error.value = e.message || '共享配置校验失败'
    } finally {
      if (token === generation) working.value = false
    }
  }
  async function save() {
    if (
      !preview.value ||
      !loaded.value ||
      !props.canEdit ||
      working.value ||
      (props.environment.is_production && !confirmed.value)
    )
      return
    const token = generation
    working.value = true
    error.value = ''
    try {
      const input = request()
      if (checked !== JSON.stringify(input)) throw Error('配置已变化，请重新校验')
      const result = await saveHelmSharedValuesApi(props.environment.id, input)
      if (token === generation) {
        if (!result.saved || !result.validated) throw Error('保存结果未确认')
        visible.value = false
        ElMessage.success('共享配置已保存，发布后生效')
        emit('saved')
      }
    } catch (e: any) {
      if (token === generation) error.value = e.message || '保存失败'
    } finally {
      if (token === generation) working.value = false
    }
  }
  watch(working, (v) => emit('busy-change', v))
  watch(
    () => props.environment.id,
    () => {
      generation++
      visible.value = false
      working.value = false
    }
  )
</script>
<style scoped>
  .note {
    font-size: 12px;
    line-height: 1.7;
    color: var(--el-text-color-secondary);
  }
  .comparison {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }
  .comparison pre {
    white-space: pre-wrap;
    overflow: auto;
    max-height: 360px;
    background: var(--el-fill-color-lighter);
    padding: 12px;
  }
  @media (max-width: 700px) {
    .comparison {
      grid-template-columns: 1fr;
    }
  }
</style>
