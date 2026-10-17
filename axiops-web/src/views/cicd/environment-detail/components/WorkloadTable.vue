<template>
  <ElTable v-loading="loading" :data="rows" style="width: 100%">
    <ElTableColumn prop="name" label="名称" min-width="200">
      <template #default="{ row }">
        <span class="mono">{{ row.name }}</span>
      </template>
    </ElTableColumn>

    <ElTableColumn v-if="showsReplicas || kind === 'daemonset'" label="就绪" width="100">
      <template #default="{ row }">{{ row.ready }}</template>
    </ElTableColumn>

    <ElTableColumn v-if="kind === 'cronjob'" label="调度" width="140">
      <template #default="{ row }">{{ row.schedule }}</template>
    </ElTableColumn>

    <ElTableColumn v-if="kind !== 'job'" label="状态" min-width="180">
      <template #default="{ row }">
        <ElTooltip v-if="row.status_reasons?.length" placement="top">
          <template #content>
            <div v-for="item in row.status_reasons" :key="item.reason">
              {{ item.reason }} · {{ item.pods }} 个 Pod
            </div>
          </template>
          <ElTag size="small" :type="statusTagType(row.status)">{{ row.status || '未知' }}</ElTag>
        </ElTooltip>
        <ElTag v-else size="small" :type="statusTagType(row.status)">{{
          row.status || '未知'
        }}</ElTag>
      </template>
    </ElTableColumn>

    <ElTableColumn v-if="kind === 'job'" label="完成数" width="110">
      <template #default="{ row }">{{ row.completions }}</template>
    </ElTableColumn>

    <ElTableColumn v-if="kind === 'job'" label="状态" width="110">
      <template #default="{ row }">
        <ElTag size="small" :type="statusTagType(row.status)">{{ row.status }}</ElTag>
      </template>
    </ElTableColumn>

    <ElTableColumn label="镜像" min-width="260">
      <template #default="{ row }">
        <div class="images">
          <span v-for="img in row.images" :key="img" class="mono image-tag">{{ img }}</span>
        </div>
      </template>
    </ElTableColumn>

    <ElTableColumn label="创建时间" width="180">
      <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
    </ElTableColumn>

    <ElTableColumn label="操作" width="220" fixed="right">
      <template #default="{ row }">
        <ElButton text type="primary" size="small" @click="$emit('view-yaml', row)">
          YAML / 事件
        </ElButton>
        <ElButton
          v-if="showsReplicas && !readOnly"
          text
          type="primary"
          size="small"
          @click="$emit('scale', row)"
        >
          改副本
        </ElButton>
        <ElButton
          v-if="canRestart && row.restart_allowed"
          text
          type="primary"
          size="small"
          @click="$emit('restart', row)"
        >
          重启
        </ElButton>
        <ElButton
          v-if="kind === 'cronjob' && !readOnly"
          text
          type="primary"
          size="small"
          @click="$emit('toggle-suspend', row)"
        >
          {{ row.suspend ? '恢复' : '暂停' }}
        </ElButton>
        <ElButton
          v-if="kind === 'cronjob' && !readOnly"
          text
          type="primary"
          size="small"
          @click="$emit('trigger', row)"
        >
          立即执行
        </ElButton>
      </template>
    </ElTableColumn>
  </ElTable>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import type { WorkloadKind, EnvWorkloadItem } from '@/api/axiops/env-resource'

  const props = defineProps<{
    loading: boolean
    kind: WorkloadKind
    rows: EnvWorkloadItem[]
    readOnly?: boolean
  }>()

  defineEmits<{
    (e: 'view-yaml', row: EnvWorkloadItem): void
    (e: 'scale', row: EnvWorkloadItem): void
    (e: 'restart', row: EnvWorkloadItem): void
    (e: 'toggle-suspend', row: EnvWorkloadItem): void
    (e: 'trigger', row: EnvWorkloadItem): void
  }>()

  // 只有 Deployment / StatefulSet 有副本语义
  const showsReplicas = computed(() => props.kind === 'deployment' || props.kind === 'statefulset')
  // DaemonSet / Deployment / StatefulSet 支持重启；CronJob / Job 不支持
  const canRestart = computed(() => ['deployment', 'statefulset', 'daemonset'].includes(props.kind))

  function statusTagType(status?: string) {
    if (status === 'Completed' || status === 'Ready') return 'success'
    if (status === 'Failed' || status === 'Degraded') return 'danger'
    if (status === 'NotReady' || status === 'Progressing') return 'warning'
    return 'info'
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
