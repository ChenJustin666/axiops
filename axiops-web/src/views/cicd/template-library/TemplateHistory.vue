<template>
  <ElDialog
    :model-value="modelValue"
    :title="`${template.name} · 版本与使用情况`"
    width="min(1120px, 95vw)"
    append-to-body
    @update:model-value="emit('update:modelValue', $event)"
  >
    <ElAlert v-if="error" :title="error" type="error" :closable="false" />
    <ElTabs v-model="tab" @tab-change="loadUsages">
      <ElTabPane label="版本历史" name="history">
        <div class="version-toolbar"
          ><ElSelect :model-value="revision" :disabled="loading" @change="selectVersion"
            ><ElOption
              v-for="v in versions"
              :key="v.revision"
              :value="v.revision"
              :label="`v${v.revision} · ${v.name}`" /></ElSelect
          ><span>历史版本只读，采用后需另行保存为新版本。</span
          ><ElButton v-if="canEdit" size="small" :disabled="!selected || loading" @click="restore"
            >采用此版本编辑</ElButton
          ></div
        >
        <p v-if="versions.length === 200">显示最近 200 个版本。</p>
        <p v-if="!loading && !versions.length">暂无已保存历史。早期被覆盖的内容无法恢复。</p>
        <div class="history-layout" v-loading="loading"
          ><nav
            ><button
              v-for="f in selected?.files"
              :key="f.path"
              :class="{ active: path === f.path }"
              @click="path = f.path"
              >{{ f.path }}</button
            ></nav
          ><YamlEditorPro
            :model-value="content"
            readonly
            height="min(56vh, 650px)"
            :show-title-bar="false"
            :show-toolbar="false"
            :show-status-bar="false"
            :validate-on-change="false"
            :enable-highlight="false"
            :auto-format="false"
        /></div>
      </ElTabPane>
      <ElTabPane label="使用情况" name="usages">
        <p>仅显示有权查看的服务和环境。模板保存不会自动升级这些使用方。</p>
        <ElTable
          :data="usages"
          v-loading="usageLoading"
          max-height="550"
          empty-text="暂无可见的使用方"
        >
          <ElTableColumn prop="service_name" label="服务" /><ElTableColumn label="项目" width="100"
            ><template #default="{ row }">#{{ row.project_id }}</template></ElTableColumn
          >
          <ElTableColumn label="环境"
            ><template #default="{ row }">{{
              row.environment_name || '服务 Chart（尚未代表环境发布）'
            }}</template></ElTableColumn
          >
          <ElTableColumn label="固定模板版本" width="140"
            ><template #default="{ row }">v{{ row.revision }}</template></ElTableColumn
          >
        </ElTable>
      </ElTabPane>
    </ElTabs>
  </ElDialog>
</template>
<script setup lang="ts">
  import { computed, onBeforeUnmount, ref, watch } from 'vue'
  import {
    listDeliveryTemplateVersionsApi,
    getDeliveryTemplateVersionApi,
    listDeliveryTemplateUsagesApi,
    type DeliveryTemplate,
    type DeliveryTemplateVersion,
    type DeliveryTemplateUsage
  } from '@/api/axiops'
  import YamlEditorPro from '@/components/yaml-editor-pro/index.vue'
  const props = defineProps<{ modelValue: boolean; template: DeliveryTemplate; canEdit: boolean }>()
  const emit = defineEmits<{
    'update:modelValue': [value: boolean]
    restore: [template: DeliveryTemplate]
  }>()
  const tab = ref('history'),
    loading = ref(false),
    usageLoading = ref(false),
    error = ref(''),
    versions = ref<DeliveryTemplateVersion[]>([]),
    selected = ref<DeliveryTemplate>(),
    revision = ref<number>(),
    path = ref(''),
    usages = ref<DeliveryTemplateUsage[]>([])
  const content = computed(
    () => selected.value?.files.find((f) => f.path === path.value)?.content || ''
  )
  let generation = 0,
    selection = 0,
    usagesLoaded = false
  async function load() {
    const current = ++generation
    ++selection
    versions.value = []
    selected.value = undefined
    usages.value = []
    usagesLoaded = false
    usageLoading.value = false
    revision.value = undefined
    error.value = ''
    tab.value = 'history'
    loading.value = props.modelValue
    if (!props.modelValue) return
    try {
      const list = await listDeliveryTemplateVersionsApi(props.template.id)
      if (current !== generation) return
      versions.value = list || []
      if (list?.[0]) await selectVersion(list[0].revision)
    } catch (e: any) {
      if (current === generation) error.value = e?.message || '历史读取失败'
    } finally {
      if (current === generation) loading.value = false
    }
  }
  async function selectVersion(value: number) {
    const current = generation,
      request = ++selection
    revision.value = value
    selected.value = undefined
    loading.value = true
    error.value = ''
    try {
      const v = await getDeliveryTemplateVersionApi(props.template.id, value)
      if (current !== generation || request !== selection) return
      selected.value = v.template
      path.value = v.template.files[0]?.path || ''
    } catch (e: any) {
      if (current === generation && request === selection)
        error.value = e?.message || '版本读取失败'
    } finally {
      if (current === generation && request === selection) loading.value = false
    }
  }
  async function loadUsages() {
    if (tab.value !== 'usages' || usagesLoaded || usageLoading.value) return
    const current = generation
    usageLoading.value = true
    error.value = ''
    try {
      const list = await listDeliveryTemplateUsagesApi(props.template.id)
      if (current !== generation) return
      usages.value = list || []
      usagesLoaded = true
    } catch (e: any) {
      if (current === generation) error.value = e?.message || '使用情况读取失败'
    } finally {
      if (current === generation) usageLoading.value = false
    }
  }
  function restore() {
    if (!selected.value || !props.canEdit || loading.value) return
    emit(
      'restore',
      JSON.parse(JSON.stringify({ ...selected.value, revision: props.template.revision }))
    )
    emit('update:modelValue', false)
  }
  watch(() => [props.modelValue, props.template.id], load, { immediate: true })
  onBeforeUnmount(() => {
    generation++
    selection++
  })
</script>
<style scoped>
  .version-toolbar {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
    margin: 12px 0;
  }
  .version-toolbar .el-select {
    width: 230px;
  }
  p,
  .version-toolbar span {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    line-height: 1.8;
  }
  .history-layout {
    display: grid;
    grid-template-columns: 210px minmax(0, 1fr);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    overflow: hidden;
  }
  nav {
    padding: 8px;
    max-height: 56vh;
    overflow: auto;
    border-right: 1px solid var(--el-border-color-lighter);
  }
  nav button {
    display: block;
    width: 100%;
    text-align: left;
    background: transparent;
    color: var(--el-text-color-regular);
    border: 0;
    padding: 9px;
    border-radius: 4px;
    font-size: 12px;
    cursor: pointer;
    overflow-wrap: anywhere;
  }
  nav button.active {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
  @media (max-width: 760px) {
    .history-layout {
      grid-template-columns: 1fr;
    }
    nav {
      max-height: 130px;
    }
  }
</style>
