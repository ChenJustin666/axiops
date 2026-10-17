<template>
  <details class="environment-comparison" @toggle="onToggle">
    <summary>对比其他环境的配置（只读）</summary>
    <template v-if="opened">
      <div class="comparison-controls">
        <ElSelect v-model="targetId" clearable placeholder="选择对比环境" style="width: 260px">
          <ElOption
            v-for="env in environments.filter((e) => e.id !== config.environment_id)"
            :key="env.id"
            :value="env.id"
            :label="`${env.name}${env.is_production ? ' · 生产' : ''}`"
          />
        </ElSelect>
        <ElCheckbox v-model="differencesOnly">只看差异</ElCheckbox>
        <ElButton v-if="targetId" size="small" :disabled="loading" @click="readTarget"
          >刷新对比</ElButton
        >
      </div>
      <p
        >比较已保存配置；本地未保存修改不参与。环境地址和文件建议在目标环境明确覆盖；对比不会保存或部署。</p
      >
      <ElAlert v-if="error" :title="error" type="error" :closable="false" />
      <div v-loading="loading">
        <template v-if="other && !error">
          <p
            >{{ current?.name }}：{{ current?.namespace || current?.host_group_ref }} ↔
            {{ target?.name }}：{{ target?.namespace || target?.host_group_ref }}</p
          >
          <ElTable :data="rows" max-height="440" empty-text="没有配置差异">
            <ElTableColumn prop="key" label="配置项" width="180" />
            <ElTableColumn :label="current?.name || '当前环境'"
              ><template #default="{ row }"
                ><small>{{ row.beforeOrigin }}</small
                ><pre>{{ row.before }}</pre>
              </template></ElTableColumn
            >
            <ElTableColumn :label="target?.name || '对比环境'"
              ><template #default="{ row }"
                ><small>{{ row.afterOrigin }}</small
                ><pre>{{ row.after }}</pre>
              </template></ElTableColumn
            >
          </ElTable>
          <details v-if="config.definition !== other.definition"
            ><summary>查看模板 YAML 差异</summary
            ><div class="definitions">
              <pre>{{ config.definition }}</pre
              ><pre>{{ other.definition }}</pre>
            </div></details
          >
        </template>
      </div>
    </template>
  </details>
</template>
<script setup lang="ts">
  import { computed, onBeforeUnmount, ref, watch } from 'vue'
  import {
    getServiceEnvironmentConfigApi,
    type AxiOpsEnvironment,
    type ServiceEnvironmentConfig
  } from '@/api/axiops'
  import { environmentComparison } from '@/utils/environment-comparison'
  const props = defineProps<{
    config: ServiceEnvironmentConfig
    environments: AxiOpsEnvironment[]
  }>()
  const targetId = ref<number | undefined>(),
    opened = ref(false),
    differencesOnly = ref(true),
    loading = ref(false),
    error = ref('')
  const other = ref<ServiceEnvironmentConfig | null>(null)
  let generation = 0
  const onToggle = (event: Event) => {
    opened.value = (event.target as HTMLDetailsElement).open
  }
  const current = computed(() =>
    props.environments.find((e) => e.id === props.config.environment_id)
  )
  const target = computed(() => props.environments.find((e) => e.id === targetId.value))
  const rows = computed(() =>
    other.value
      ? environmentComparison(props.config, other.value).filter(
          (r) => !differencesOnly.value || r.changed
        )
      : []
  )
  async function readTarget() {
    const request = ++generation,
      id = targetId.value,
      sid = props.config.service_id
    other.value = null
    error.value = ''
    loading.value = false
    if (!opened.value || !id || !target.value || id === props.config.environment_id) return
    loading.value = true
    try {
      const result = await getServiceEnvironmentConfigApi(id, sid)
      if (request !== generation) return
      if (result.environment_id !== id || result.service_id !== sid)
        throw Error('返回配置与选择的环境不符')
      environmentComparison(props.config, result)
      other.value = result
    } catch (e: any) {
      if (request === generation) error.value = e?.message || '环境配置读取失败'
    } finally {
      if (request === generation) loading.value = false
    }
  }
  watch([targetId, opened], readTarget, { flush: 'sync' })
  watch(
    () => [props.config.service_id, props.config.environment_id, props.config.token],
    () => {
      generation++
      other.value = null
      targetId.value = undefined
      error.value = ''
      loading.value = false
    },
    { flush: 'sync' }
  )
  onBeforeUnmount(() => generation++)
</script>
<style scoped>
  .environment-comparison {
    margin: 12px 16px;
    border: 1px solid var(--el-border-color-lighter);
    padding: 12px;
    border-radius: 6px;
  }
  summary {
    cursor: pointer;
  }
  .comparison-controls {
    display: flex;
    gap: 12px;
    margin-top: 12px;
    align-items: center;
  }
  p,
  small {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
  pre {
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    max-height: 260px;
    overflow: auto;
    font: 12px/1.6 var(--ax-font-mono);
  }
  .definitions {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }
</style>
