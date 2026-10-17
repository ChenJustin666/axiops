<template>
  <div class="env-detail-page">
    <ElButton text class="back-button" @click="router.back()">返回</ElButton>
    <ElAlert v-if="error" type="error" :title="error" :closable="false"
      ><ElButton text @click="load">重新读取</ElButton></ElAlert
    >
    <ElCard v-else v-loading="loading" shadow="never">
      <EnvironmentRuntimePanel
        v-if="environment"
        :key="environment.id"
        :environment="environment"
        :cluster-name="clusterName"
        delivery-type="k8s_yaml"
      />
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, ref, watch } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import {
    getEnvironmentApi,
    listClustersApi,
    type AxiOpsEnvironment,
    type AxiOpsCluster
  } from '@/api/axiops/cicd'
  import EnvironmentRuntimePanel from '../project-workspace/components/EnvironmentRuntimePanel.vue'
  const route = useRoute(),
    router = useRouter()
  const environment = ref<AxiOpsEnvironment | null>(null),
    clusters = ref<AxiOpsCluster[]>([]),
    loading = ref(false),
    error = ref('')
  let generation = 0
  const clusterName = computed(
    () =>
      clusters.value.find((c) => c.id === environment.value?.cluster_id)?.name ||
      `集群 #${environment.value?.cluster_id || '—'}`
  )
  async function load() {
    const id = Number(route.params.id),
      current = ++generation
    environment.value = null
    error.value = ''
    loading.value = true
    try {
      if (!Number.isSafeInteger(id) || id <= 0) throw Error('环境编号无效')
      const env = await getEnvironmentApi(id)
      if (current !== generation) return
      if (env.id !== id) throw Error('环境返回范围不一致')
      environment.value = env
      try {
        const list = await listClustersApi(undefined, { silent: true })
        if (current === generation) clusters.value = list || []
      } catch {
        if (current === generation) clusters.value = []
      }
    } catch (e) {
      if (current === generation) error.value = e instanceof Error ? e.message : '读取环境失败'
    } finally {
      if (current === generation) loading.value = false
    }
  }
  watch(() => route.params.id, load, { immediate: true })
  onBeforeUnmount(() => {
    generation++
  })
</script>

<style scoped>
  .env-detail-page {
    display: flex;
    flex-direction: column;
    gap: 12px;
    min-width: 0;
  }
  .back-button {
    align-self: flex-start;
  }
</style>
