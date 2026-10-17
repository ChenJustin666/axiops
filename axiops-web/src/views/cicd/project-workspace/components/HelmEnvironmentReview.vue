<template>
  <ElDialog
    v-model="visible"
    :title="`创建环境预览 · ${input?.name}`"
    width="min(1120px, 95vw)"
    append-to-body
    @closed="!visible && finish(false)"
  >
    <p
      >{{ input?.namespace }} ·
      仅创建环境和服务配置，不会部署。预览中的镜像为校验占位值，发布时由工作流注入选定制品。</p
    >
    <ElSelect v-model="active" style="width: 100%"
      ><ElOption
        v-for="c in configs"
        :key="c.service_id"
        :value="c.service_id"
        :label="services.find((s) => s.id === c.service_id)?.name || `服务 #${c.service_id}`"
    /></ElSelect>
    <ElTabs v-model="tab"
      ><ElTabPane label="最终 values" name="values" /><ElTabPane label="渲染 YAML" name="yaml"
    /></ElTabs>
    <YamlEditorPro
      :model-value="tab === 'values' ? selected?.final_values || '' : selected?.manifest || ''"
      readonly
      height="min(55vh, 620px)"
      :show-title-bar="false"
      :show-toolbar="false"
      :auto-format="false"
    />
    <ElCheckbox v-if="production" v-model="confirmed"
      >已核对生产环境的域名、配置中心、数据地址及 Secret 引用</ElCheckbox
    >
    <template #footer
      ><ElButton @click="finish(false)">返回修改</ElButton
      ><ElButton type="primary" :disabled="production && !confirmed" @click="finish(true)"
        >确认创建环境</ElButton
      ></template
    >
  </ElDialog>
</template>
<script setup lang="ts">
  import { computed, onBeforeUnmount, ref } from 'vue'
  import {
    createEnvironmentApi,
    type AxiOpsService,
    type CreateEnvironmentInput,
    type ServiceEnvironmentConfig
  } from '@/api/axiops'
  import YamlEditorPro from '@/components/yaml-editor-pro/index.vue'
  const props = defineProps<{ projectId: number; services: AxiOpsService[] }>()
  const visible = ref(false),
    input = ref<CreateEnvironmentInput>(),
    configs = ref<ServiceEnvironmentConfig[]>([]),
    active = ref(0),
    tab = ref('values'),
    confirmed = ref(false)
  const selected = computed(() => configs.value.find((c) => c.service_id === active.value))
  const production = computed(
    () => input.value?.is_production || ['prod', 'production'].includes(input.value?.code || '')
  )
  let resolve: ((confirmed: boolean) => void) | undefined
  async function review(data: CreateEnvironmentInput) {
    if (resolve) throw Error('正在确认其他环境')
    input.value = JSON.parse(JSON.stringify(data))
    confirmed.value = false
    const result = await createEnvironmentApi(props.projectId, input.value!, true)
    if (
      result.saved ||
      result.configs?.length !== data.helm_services?.length ||
      result.configs?.some((c) => !c.manifest || !c.final_values)
    )
      throw Error('环境预览不完整，请更新 API 后重试')
    configs.value = result.configs || []
    active.value = configs.value[0]?.service_id || 0
    tab.value = 'values'
    visible.value = true
    return new Promise<boolean>((done) => {
      resolve = done
    })
  }
  function finish(confirmed: boolean) {
    const done = resolve
    resolve = undefined
    visible.value = false
    done?.(confirmed)
  }
  onBeforeUnmount(() => finish(false))
  defineExpose({ review })
</script>
