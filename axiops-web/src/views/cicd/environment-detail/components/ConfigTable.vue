<template>
  <ElTable v-loading="loading" :data="rows" style="width: 100%">
    <ElTableColumn prop="name" label="名称" min-width="200" />
    <ElTableColumn label="类型" width="160">
      <template #default="{ row }">{{ row.type || row.kind }}</template>
    </ElTableColumn>
    <ElTableColumn label="键数量" width="100">
      <template #default="{ row }">{{ row.keys }}</template>
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
  import type { EnvConfigItem } from '@/api/axiops/env-resource'

  defineProps<{ loading: boolean; rows: EnvConfigItem[] }>()
  defineEmits<{ (e: 'view-yaml', row: EnvConfigItem): void }>()

  function formatTime(ms: number) {
    if (!ms) return '—'
    return new Date(ms).toLocaleString()
  }
</script>
