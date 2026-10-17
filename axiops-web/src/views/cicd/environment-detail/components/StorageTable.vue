<template>
  <ElTable v-loading="loading" :data="rows" style="width: 100%">
    <ElTableColumn prop="name" label="名称" min-width="200" />
    <ElTableColumn label="状态" width="110">
      <template #default="{ row }">
        <ElTag size="small" :type="row.status === 'Bound' ? 'success' : 'warning'">
          {{ row.status }}
        </ElTag>
      </template>
    </ElTableColumn>
    <ElTableColumn label="容量" width="110">
      <template #default="{ row }">{{ row.capacity || '—' }}</template>
    </ElTableColumn>
    <ElTableColumn label="访问模式" width="140">
      <template #default="{ row }">{{ row.access_modes || '—' }}</template>
    </ElTableColumn>
    <ElTableColumn label="StorageClass" width="160">
      <template #default="{ row }">{{ row.storage_class || '—' }}</template>
    </ElTableColumn>
    <ElTableColumn label="创建时间" width="180">
      <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
    </ElTableColumn>
    <ElTableColumn label="操作" width="100" fixed="right">
      <template #default="{ row }">
        <ElButton text type="primary" size="small" @click="$emit('view-yaml', row)">
          查看 YAML
        </ElButton>
      </template>
    </ElTableColumn>
  </ElTable>
</template>

<script setup lang="ts">
  import type { EnvPVCItem } from '@/api/axiops/env-resource'

  defineProps<{ loading: boolean; rows: EnvPVCItem[] }>()
  defineEmits<{ (e: 'view-yaml', row: EnvPVCItem): void }>()

  function formatTime(ms: number) {
    if (!ms) return '—'
    return new Date(ms).toLocaleString()
  }
</script>
