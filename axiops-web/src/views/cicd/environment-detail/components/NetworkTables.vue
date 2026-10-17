<template>
  <div class="network-tables">
    <h4>Service</h4>
    <ElTable v-loading="loading" :data="services" style="width: 100%">
      <ElTableColumn prop="name" label="名称" min-width="180" />
      <ElTableColumn label="类型" width="120">
        <template #default="{ row }">{{ row.type }}</template>
      </ElTableColumn>
      <ElTableColumn label="ClusterIP" width="150">
        <template #default="{ row }">{{ row.cluster_ip }}</template>
      </ElTableColumn>
      <ElTableColumn label="端口" min-width="180">
        <template #default="{ row }">{{ row.ports.join(', ') }}</template>
      </ElTableColumn>
      <ElTableColumn label="创建时间" width="180">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </ElTableColumn>
      <ElTableColumn label="操作" width="100" fixed="right">
        <template #default="{ row }">
          <ElButton text type="primary" size="small" @click="$emit('view-service-yaml', row)">
            YAML
          </ElButton>
        </template>
      </ElTableColumn>
    </ElTable>

    <h4 class="mt">Ingress</h4>
    <ElTable v-loading="loading" :data="ingresses" style="width: 100%">
      <ElTableColumn prop="name" label="名称" min-width="180" />
      <ElTableColumn label="域名" min-width="220">
        <template #default="{ row }">{{ row.hosts.join(', ') || '—' }}</template>
      </ElTableColumn>
      <ElTableColumn label="Class" width="140">
        <template #default="{ row }">{{ row.class || '—' }}</template>
      </ElTableColumn>
      <ElTableColumn label="创建时间" width="180">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </ElTableColumn>
      <ElTableColumn label="操作" width="100" fixed="right">
        <template #default="{ row }">
          <ElButton text type="primary" size="small" @click="$emit('view-ingress-yaml', row)">
            YAML
          </ElButton>
        </template>
      </ElTableColumn>
    </ElTable>
  </div>
</template>

<script setup lang="ts">
  import type { EnvServiceItem, EnvIngressItem } from '@/api/axiops/env-resource'

  defineProps<{
    loading: boolean
    services: EnvServiceItem[]
    ingresses: EnvIngressItem[]
  }>()
  defineEmits<{
    (e: 'view-service-yaml', row: EnvServiceItem): void
    (e: 'view-ingress-yaml', row: EnvIngressItem): void
  }>()

  function formatTime(ms: number) {
    if (!ms) return '—'
    return new Date(ms).toLocaleString()
  }
</script>

<style scoped lang="scss">
  .network-tables h4 {
    margin: 0 0 8px;
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }
  .mt {
    margin-top: 20px;
  }
</style>
