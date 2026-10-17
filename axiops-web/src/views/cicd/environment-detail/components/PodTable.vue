<template>
  <ElTable v-loading="loading" :data="rows" style="width: 100%">
    <ElTableColumn prop="name" label="名称" min-width="200">
      <template #default="{ row }"
        ><span class="mono">{{ row.name }}</span></template
      >
    </ElTableColumn>
    <ElTableColumn label="状态" width="110">
      <template #default="{ row }">
        <ElTag size="small" :type="phaseTagType(row.status || row.phase)">{{
          row.status || row.phase
        }}</ElTag>
      </template>
    </ElTableColumn>
    <ElTableColumn label="Ready" width="90">
      <template #default="{ row }">{{ row.ready }}</template>
    </ElTableColumn>
    <ElTableColumn label="重启次数" width="90">
      <template #default="{ row }">{{ row.restarts }}</template>
    </ElTableColumn>
    <ElTableColumn label="镜像地址" min-width="260">
      <template #default="{ row }">
        <div class="images">
          <span v-for="img in row.images" :key="img" class="mono image-tag">{{ img }}</span>
        </div>
      </template>
    </ElTableColumn>
    <ElTableColumn label="节点" width="140">
      <template #default="{ row }">{{ row.node_name || '—' }}</template>
    </ElTableColumn>
    <ElTableColumn label="创建时间" width="180">
      <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
    </ElTableColumn>
    <ElTableColumn label="操作" width="200" fixed="right">
      <template #default="{ row }">
        <ElButton text type="primary" size="small" @click="$emit('view-yaml', row)">
          YAML / 事件
        </ElButton>
        <ElButton text type="primary" size="small" @click="$emit('logs', row)">日志</ElButton>
        <ElButton v-if="!readOnly" text type="primary" size="small" @click="$emit('exec', row)"
          >调试</ElButton
        >
      </template>
    </ElTableColumn>
  </ElTable>
</template>

<script setup lang="ts">
  import type { EnvPodItem } from '@/api/axiops/env-resource'

  defineProps<{ loading: boolean; rows: EnvPodItem[]; readOnly?: boolean }>()
  defineEmits<{
    (e: 'view-yaml', row: EnvPodItem): void
    (e: 'logs', row: EnvPodItem): void
    (e: 'exec', row: EnvPodItem): void
  }>()

  function phaseTagType(phase: string) {
    if (phase === 'Ready' || phase === 'Succeeded') return 'success'
    if (
      [
        'Failed',
        'CrashLoopBackOff',
        'ImagePullBackOff',
        'ErrImagePull',
        'Error',
        'OOMKilled'
      ].includes(phase)
    )
      return 'danger'
    return 'warning'
  }

  function formatTime(ms: number) {
    if (!ms) return '—'
    return new Date(ms).toLocaleString()
  }
</script>

<style scoped lang="scss">
  .mono {
    font-family: var(--ax-font-mono);
    font-size: 12px;
  }
  .images {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .image-tag {
    color: var(--el-text-color-secondary);
  }
</style>
