<template>
  <div class="run-trend" :class="{ 'is-compact': compact }">
    <div class="trend-plot">
      <template v-for="(item, index) in slots" :key="item ? item.run.id : `empty-${index}`">
        <button
          v-if="item"
          type="button"
          class="trend-bar"
          :class="`is-${item.tone}`"
          :style="{ height: item.height }"
          :title="item.tip"
          :aria-label="item.tip"
          @click="emit('select', { run: item.run, workflow: item.workflow })"
        >
          <i v-if="item.tone === 'failed'" class="trend-dot" />
        </button>
        <span v-else class="trend-bar is-empty" />
      </template>
    </div>
    <div v-if="!compact" class="trend-scale">
      <span>{{ scaleLabel }}</span>
      <span class="trend-tip">{{ latestLabel }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import type { AxiOpsWorkflow, AxiOpsWorkflowRun } from '@/api/axiops'

  defineOptions({ name: 'ProjectRunTrend' })

  const props = withDefaults(
    defineProps<{
      runs: { run: AxiOpsWorkflowRun; workflow?: AxiOpsWorkflow }[]
      compact?: boolean
    }>(),
    { compact: false }
  )
  const emit = defineEmits<{
    select: [{ run: AxiOpsWorkflowRun; workflow?: AxiOpsWorkflow }]
  }>()

  const successStatuses = ['succeeded', 'success', 'completed']
  const terminalStatuses = new Set([...successStatuses, 'failed', 'cancelled'])
  const statusLabels: Record<string, string> = {
    succeeded: '成功',
    success: '成功',
    completed: '成功',
    failed: '失败',
    cancelled: '已取消',
    running: '执行中',
    pending: '排队中'
  }
  const statusLabel = (status: string) => statusLabels[status] || status || '未知'
  const formatDate = (value?: string) =>
    value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'
  const runSeconds = (run: AxiOpsWorkflowRun) => {
    if (!run.started_at) return 0
    const end = run.finished_at ? new Date(run.finished_at).getTime() : Date.now()
    return Math.max(1, Math.round((end - new Date(run.started_at).getTime()) / 1000))
  }
  const toneOf = (status: string) =>
    successStatuses.includes(status)
      ? 'success'
      : terminalStatuses.has(status)
        ? 'failed'
        : 'pending'

  // 非 compact：全局时间序列，条高按耗时归一化
  // compact：固定 10 个槽位，最近运行靠右对齐，空槽留灰
  const slotCount = computed(() => (props.compact ? 10 : props.runs.length || 1))
  const maxSeconds = computed(() => Math.max(1, ...props.runs.map((item) => runSeconds(item.run))))
  const filledBars = computed(() =>
    props.runs.map(({ run, workflow }) => {
      const ratio = runSeconds(run) / maxSeconds.value
      const tone = toneOf(run.status)
      const percent = props.compact
        ? Math.round(35 + ratio * 65)
        : Math.round(26 + ratio * 74)
      return {
        run,
        workflow,
        tone,
        height: `${tone === 'failed' ? Math.max(percent, props.compact ? 48 : 54) : percent}%`,
        tip: `#${run.run_number || '—'} ${workflow?.name || '工作流'} · ${statusLabel(run.status)} · ${runSeconds(run)} 秒 · ${formatDate(run.created_at)}`
      }
    })
  )
  const slots = computed(() => {
    if (!props.compact) return filledBars.value
    const list = new Array<ReturnType<typeof filledBars.value[number] | null>>(slotCount.value).fill(null)
    const recent = filledBars.value.slice(-slotCount.value)
    recent.forEach((bar, index) => {
      list[slotCount.value - recent.length + index] = bar
    })
    return list
  })
  const scaleLabel = computed(() => `峰值 ${maxSeconds.value} 秒`)
  const latestLabel = computed(() => {
    const last = props.runs[props.runs.length - 1]
    return last ? `最近一次：#${last.run.run_number || '—'} ${statusLabel(last.run.status)}` : ''
  })
</script>

<style scoped lang="scss">
  .run-trend {
    padding: 14px 16px 10px;
  }
  .run-trend.is-compact {
    padding: 0;
  }
  .trend-plot {
    display: flex;
    align-items: flex-end;
    gap: 3px;
    height: 56px;
    background-image: repeating-linear-gradient(
      to top,
      transparent 0 13px,
      #eef2f7 13px 14px
    );
    border-bottom: 1px solid #dbe3ec;
  }
  .is-compact .trend-plot {
    gap: 2px;
    height: 26px;
    background-image: none;
    border-bottom: 0;
  }
  .trend-bar {
    position: relative;
    flex: 1 1 0;
    min-width: 4px;
    padding: 0;
    border: 0;
    border-radius: 2px 2px 0 0;
    cursor: pointer;
    transition: filter 0.15s ease;
  }
  .is-compact .trend-bar {
    min-width: 5px;
    border-radius: 1.5px 1.5px 0 0;
  }
  .trend-bar.is-empty {
    cursor: default;
    background: #f1f5f9;
  }
  .trend-bar:hover:not(.is-empty) {
    filter: brightness(1.12);
  }
  .trend-bar.is-success {
    background: linear-gradient(
      180deg,
      color-mix(in srgb, var(--el-color-success) 88%, transparent),
      color-mix(in srgb, var(--el-color-success) 62%, transparent)
    );
  }
  .is-compact .trend-bar.is-success {
    background: linear-gradient(
      180deg,
      color-mix(in srgb, var(--el-color-success) 80%, transparent),
      color-mix(in srgb, var(--el-color-success) 100%, transparent)
    );
  }
  .trend-bar.is-failed {
    background: color-mix(in srgb, var(--el-color-danger) 30%, transparent);
  }
  .is-compact .trend-bar.is-failed {
    background: color-mix(in srgb, var(--el-color-danger) 35%, transparent);
  }
  .trend-bar.is-pending {
    background: color-mix(in srgb, var(--el-color-warning) 45%, transparent);
  }
  .is-compact .trend-bar.is-pending {
    background: color-mix(in srgb, var(--el-color-warning) 55%, transparent);
  }
  .trend-dot {
    position: absolute;
    top: -3px;
    left: 50%;
    width: 5px;
    height: 5px;
    margin-left: -2.5px;
    border-radius: 50%;
    background: var(--el-color-danger);
    box-shadow: 0 0 0 2px #fff;
  }
  .is-compact .trend-dot {
    top: -2.5px;
    width: 4px;
    height: 4px;
    margin-left: -2px;
  }
  .trend-scale {
    display: flex;
    justify-content: space-between;
    margin-top: 8px;
    color: #94a3b8;
    font: 11px var(--ax-font-mono);
  }
  .trend-tip {
    color: #64748b;
  }
</style>
