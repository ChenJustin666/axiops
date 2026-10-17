<template>
  <ElButton text size="small" @click="open">发布版本来源</ElButton>
  <ElDialog
    v-model="visible"
    title="生产发布版本来源"
    width="min(560px, 95vw)"
    :close-on-click-modal="false"
    :close-on-press-escape="!saving"
    :show-close="!saving"
  >
    <div v-loading="loading"
      ><p
        >为
        {{ environment.name }}
        选择测试或预发布环境。生产只接收该环境已验收的镜像，运行配置仍使用生产自己的版本。</p
      ><ElAlert v-if="error" :title="error" type="error" :closable="false" /><ElSelect
        v-model="sourceId"
        placeholder="选择来源环境"
        style="width: 100%; margin-top: 16px"
        ><ElOption
          v-for="source in sources"
          :key="source.id"
          :value="source.id"
          :label="source.name" /></ElSelect
    ></div>
    <template #footer
      ><ElButton :disabled="saving" @click="visible = false">取消</ElButton
      ><ElButton
        type="primary"
        :loading="saving"
        :disabled="loading || !sourceId || !loaded"
        @click="save"
        >保存来源</ElButton
      ></template
    >
  </ElDialog>
</template>
<script setup lang="ts">
  import { computed, ref } from 'vue'
  import {
    getProductionPolicyApi,
    saveProductionPolicyApi,
    type AxiOpsEnvironment,
    type ProductionPolicy
  } from '@/api/axiops'
  const props = defineProps<{ environment: AxiOpsEnvironment; environments: AxiOpsEnvironment[] }>()
  const visible = ref(false),
    loading = ref(false),
    saving = ref(false),
    loaded = ref(false),
    error = ref(''),
    sourceId = ref<number>(),
    policy = ref<ProductionPolicy>()
  const sources = computed(() =>
    props.environments.filter(
      (e) =>
        !e.is_production &&
        e.environment_type === 'k8s_namespace' &&
        e.status === 'active' &&
        e.binding_status === 'validated'
    )
  )
  const open = async () => {
    visible.value = true
    loading.value = true
    loaded.value = false
    error.value = ''
    try {
      policy.value = await getProductionPolicyApi(props.environment.id)
      sourceId.value = policy.value.source_environment_id || undefined
      loaded.value = true
    } catch (e) {
      error.value = (e as Error).message
    } finally {
      loading.value = false
    }
  }
  const save = async () => {
    if (!loaded.value || !policy.value || !sourceId.value || saving.value) return
    saving.value = true
    try {
      await saveProductionPolicyApi(props.environment.id, {
        ...policy.value,
        source_environment_id: sourceId.value
      })
      visible.value = false
    } catch (e) {
      error.value = (e as Error).message
    } finally {
      saving.value = false
    }
  }
</script>
