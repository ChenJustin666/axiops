<template>
  <section class="environment-services">
    <h4>配置此环境的服务</h4>
    <p class="note"
      >选择服务并填写此环境的 values。可留空先创建环境；选中的服务会一起校验保存，不会自动部署。</p
    >
    <ElSelect
      :model-value="selectedIds"
      multiple
      filterable
      placeholder="选择服务"
      style="width: 100%"
      :disabled="disabled || loading"
      @update:model-value="selectServices"
    >
      <ElOption
        v-for="s in services.filter((s) => s.delivery_type === 'helm')"
        :key="s.id"
        :value="s.id"
        :label="s.name"
      />
    </ElSelect>
    <ElAlert v-if="error" :title="error" type="error" :closable="false" />
    <ElTabs v-if="modelValue.length" v-model="active" class="services-tabs">
      <ElTabPane
        v-for="entry in modelValue"
        :key="entry.service_id"
        :name="String(entry.service_id)"
        :label="service(entry.service_id)?.name || String(entry.service_id)"
      >
        <template v-if="service(entry.service_id)">
          <div class="values-bar"
            ><span
              >独立环境 values · Chart #{{
                revisions[entry.service_id]?.find((r) => r.id === entry.revision_id)
                  ?.revision_number
              }}</span
            ><HelmValuesSources
              :model-value="entry.helm"
              :service="service(entry.service_id)!"
              :files="files(entry)"
              :environments="environments"
              :disabled="disabled || loading"
              @update:model-value="update(entry, $event)"
              @busy-change="sourceBusy = $event"
          /></div>
          <YamlEditorPro
            :model-value="entry.helm.values"
            :readonly="disabled || loading || sourceBusy"
            height="340px"
            :show-title-bar="false"
            :show-toolbar="false"
            :auto-format="false"
            @update:model-value="update(entry, { ...entry.helm, values: $event })"
          />
          <ElCollapse
            ><ElCollapseItem title="查看 Chart 默认 values / 镜像路径" :name="entry.service_id">
              <pre class="defaults">{{
                files(entry).find((f) => f.path === 'values.yaml')?.content || '{}'
              }}</pre>
              <p class="note">构建制品注入到以下路径；原生 Chart 使用不同路径时在此调整。</p>
              <div class="mapping"
                ><ElInput
                  :model-value="entry.helm.image.repository_path"
                  placeholder="image.repository"
                  :disabled="disabled"
                  @update:model-value="
                    update(entry, {
                      ...entry.helm,
                      image: { ...entry.helm.image, repository_path: $event }
                    })
                  " /><ElInput
                  :model-value="entry.helm.image.tag_path"
                  placeholder="image.tag"
                  :disabled="disabled"
                  @update:model-value="
                    update(entry, {
                      ...entry.helm,
                      image: { ...entry.helm.image, tag_path: $event }
                    })
                  "
              /></div>
              <ElInput
                :model-value="entry.helm.image.ref_path"
                placeholder="完整镜像路径（与 repository/tag 二选一，使用时清空上面两项）"
                :disabled="disabled"
                @update:model-value="
                  update(entry, { ...entry.helm, image: { ...entry.helm.image, ref_path: $event } })
                "
              /> </ElCollapseItem
          ></ElCollapse>
        </template>
      </ElTabPane>
    </ElTabs>
  </section>
</template>
<script setup lang="ts">
  import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
  import {
    listServiceRevisionsApi,
    type AxiOpsService,
    type AxiOpsEnvironment,
    type AxiOpsServiceRevision,
    type InitialHelmService,
    type HelmEnvironmentSettings
  } from '@/api/axiops'
  import { chartFiles } from '@/utils/helm-service-config'
  import YamlEditorPro from '@/components/yaml-editor-pro/index.vue'
  import HelmValuesSources from './HelmValuesSources.vue'
  const props = withDefaults(
    defineProps<{
      modelValue?: InitialHelmService[]
      services: AxiOpsService[]
      environments: AxiOpsEnvironment[]
      disabled?: boolean
    }>(),
    { modelValue: () => [] }
  )
  const emit = defineEmits<{
    'update:modelValue': [values: InitialHelmService[]]
    'busy-change': [busy: boolean]
  }>()
  const active = ref(''),
    loading = ref(false),
    sourceBusy = ref(false),
    error = ref(''),
    revisions = ref<Record<number, AxiOpsServiceRevision[]>>({})
  const selectedIds = computed(() => props.modelValue.map((e) => e.service_id))
  let alive = true
  const service = (id: number) => props.services.find((s) => s.id === id)
  const files = (entry: InitialHelmService) => {
    try {
      return chartFiles(
        revisions.value[entry.service_id]?.find((r) => r.id === entry.revision_id)?.definition || ''
      )
    } catch {
      return []
    }
  }
  function update(entry: InitialHelmService, helm: HelmEnvironmentSettings) {
    emit(
      'update:modelValue',
      props.modelValue.map((e) => (e.service_id === entry.service_id ? { ...e, helm } : e))
    )
  }
  async function selectServices(ids: number[]) {
    if (props.disabled || loading.value) return
    loading.value = true
    error.value = ''
    try {
      const next: InitialHelmService[] = []
      for (const id of ids) {
        const existing = props.modelValue.find((e) => e.service_id === id)
        if (existing) {
          next.push(existing)
          continue
        }
        const list = await listServiceRevisionsApi(id)
        if (!alive) return
        revisions.value[id] = list || []
        const revision = list?.find((r) => r.id === service(id)?.current_revision_id) || list?.[0]
        if (!revision) throw Error(`${service(id)?.name || id} 尚未导入有效 Chart`)
        next.push({
          service_id: id,
          revision_id: revision.id,
          helm: {
            values: '{}\n',
            image: { repository_path: 'image.repository', tag_path: 'image.tag', ref_path: '' }
          }
        })
      }
      emit('update:modelValue', next)
      active.value = String(
        next.find((e) => String(e.service_id) === active.value)?.service_id ||
          next[0]?.service_id ||
          ''
      )
    } catch (e: any) {
      error.value = e.message || '服务版本读取失败'
    } finally {
      loading.value = false
    }
  }
  watch(
    () => loading.value || sourceBusy.value,
    (v) => emit('busy-change', v)
  )
  onMounted(async () => {
    if (!props.modelValue.length) return
    loading.value = true
    try {
      for (const entry of props.modelValue) {
        const list = await listServiceRevisionsApi(entry.service_id)
        if (!alive) return
        revisions.value[entry.service_id] = list || []
      }
      active.value = String(props.modelValue[0]?.service_id || '')
    } catch (e: any) {
      if (alive) error.value = e.message || '服务版本读取失败'
    } finally {
      loading.value = false
    }
  })
  onBeforeUnmount(() => {
    alive = false
    emit('busy-change', false)
  })
</script>
<style scoped>
  .environment-services {
    margin-top: 24px;
    border-top: 1px solid var(--el-border-color-light);
    padding-top: 8px;
    min-width: 0;
  }
  .note,
  .values-bar {
    font-size: 12px;
    line-height: 1.7;
    color: var(--el-text-color-secondary);
  }
  .values-bar {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
    margin-bottom: 8px;
  }
  .mapping {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
    margin-bottom: 12px;
  }
  .defaults {
    white-space: pre-wrap;
    max-height: 320px;
    overflow: auto;
    padding: 12px;
    background: var(--el-fill-color-lighter);
  }
</style>
