<template>
  <div class="template-select">
    <ElSelect
      v-model="selected"
      :loading="loading || choosing"
      filterable
      placeholder="选择模板库中的模板"
      :disabled="disabled || choosing"
      @change="choose"
    >
      <ElOption
        v-for="item in templates"
        :key="item.id"
        :value="item.id"
        :label="`${item.name} · v${item.revision}`"
      />
    </ElSelect>
    <ElButton text size="small" :loading="loading" :disabled="choosing" @click="load"
      >刷新</ElButton
    >
    <RouterLink :to="{ path: '/cicd/templates', query: { kind } }" target="_blank"
      >管理模板</RouterLink
    >
    <small v-if="error" class="error">{{ error }}，可刷新重试</small>
    <small v-else-if="!loading && !templates.length">暂无可用模板，请先在模板库创建</small>
    <small v-else-if="current && showDescription !== false">{{ current.description }}</small>
  </div>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, ref, watch } from 'vue'
  import { listDeliveryTemplatesApi, type DeliveryTemplate, type TemplateKind } from '@/api/axiops'
  const props = defineProps<{
    kind: TemplateKind
    autoSelect?: boolean
    disabled?: boolean
    showDescription?: boolean
    apply?: (template: DeliveryTemplate) => Promise<boolean>
  }>()
  const emit = defineEmits<{ select: [template: DeliveryTemplate] }>()
  const templates = ref<DeliveryTemplate[]>([]),
    selected = ref<number>(),
    loading = ref(false),
    error = ref('')
  const choosing = ref(false)
  let applied: number | undefined
  const current = computed(() => templates.value.find((item) => item.id === selected.value))
  let version = 0,
    initialized = false
  async function choose(id: number) {
    const item = templates.value.find((item) => item.id === id)
    if (!item) return
    const token = version
    const copy = JSON.parse(JSON.stringify(item))
    choosing.value = true
    try {
      const accepted = !props.apply || (await props.apply(copy))
      if (token !== version) return
      selected.value = accepted ? id : applied
      if (accepted) {
        applied = id
        emit('select', copy)
      }
    } catch (e: any) {
      if (token === version) {
        selected.value = applied
        error.value = e.message || '应用模板失败'
      }
    } finally {
      if (token === version) choosing.value = false
    }
  }
  async function load() {
    const token = ++version
    loading.value = true
    error.value = ''
    try {
      const rows = await listDeliveryTemplatesApi(props.kind)
      if (token !== version) return
      templates.value = rows || []
      if (!initialized && props.autoSelect && templates.value.length) {
        selected.value = (templates.value.find((item) => item.builtin_key) || templates.value[0]).id
        await choose(selected.value)
      }
      initialized = true
    } catch (e: any) {
      if (token === version) {
        templates.value = []
        error.value = e.message || '模板读取失败'
      }
    } finally {
      if (token === version) loading.value = false
    }
  }
  watch(
    () => props.kind,
    () => {
      initialized = false
      selected.value = undefined
      applied = undefined
      choosing.value = false
      templates.value = []
      void load()
    },
    { immediate: true }
  )
  onBeforeUnmount(() => {
    version++
  })
</script>

<style scoped>
  .template-select {
    display: flex;
    gap: 6px;
    align-items: center;
    flex-wrap: wrap;
    width: 100%;
  }
  .template-select .el-select {
    flex: 1;
    min-width: 180px;
  }
  .template-select a {
    font-size: 12px;
    color: #5265ff;
    white-space: nowrap;
  }
  .template-select small {
    width: 100%;
    color: #8793a5;
    font-size: 12px;
  }
  .template-select .error {
    color: #dc2626;
  }
</style>
