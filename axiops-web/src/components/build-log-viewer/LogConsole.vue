<template>
  <section class="log-console">
    <div class="log-toolbar">
      <div class="log-caption">
        <strong>{{ title }}</strong>
        <div v-if="canSummarize" class="log-mode" role="group" aria-label="日志视图">
          <button type="button" :aria-pressed="!detailed" @click="emit('detail', false)"
            >简洁</button
          >
          <button type="button" :aria-pressed="!!detailed" @click="emit('detail', true)"
            >详细诊断</button
          >
        </div>
        <span class="follow-state" :class="{ paused: !following }" aria-live="polite">
          {{ following ? '跟随最新输出' : '已暂停跟随' }}
        </span>
      </div>
      <div class="log-actions">
        <ElButton size="small" :icon="ArrowUpToLine" @click="jumpTop">顶部</ElButton>
        <ElButton size="small" type="primary" :icon="ArrowDownToLine" @click="jumpEnd"
          >末尾</ElButton
        >
        <ElCheckbox :model-value="following" @change="setFollowing(Boolean($event))"
          >跟随输出</ElCheckbox
        >
        <ElCheckbox v-model="wrap" @change="layoutChanged">自动换行</ElCheckbox>
        <ElButton size="small" :icon="RefreshCw" :loading="loading" @click="emit('refresh')"
          >刷新</ElButton
        >
        <ElButton size="small" :icon="expanded ? Minimize2 : Maximize2" @click="emit('expand')">
          {{ expanded ? '退出全屏' : '全屏' }}
        </ElButton>
      </div>
    </div>
    <div
      ref="viewport"
      class="log-scroll"
      tabindex="0"
      role="region"
      :aria-label="`${title}，Ctrl+End 跳到末尾，Ctrl+Home 跳到顶部`"
      @scroll.passive="onScroll"
      @keydown.ctrl.end.prevent="jumpEnd"
      @keydown.ctrl.home.prevent="jumpTop"
    >
      <pre :class="{ 'wrap-lines': wrap, 'empty-log': !text }">{{ text || emptyText }}</pre>
    </div>
    <div class="log-footer">
      <span
        >{{ lineCount.toLocaleString() }} 行{{
          canSummarize && !detailed ? '摘要（重复检查已合并）' : '已加载'
        }}
        · {{ loading ? '正在刷新…' : '每 3 秒刷新' }}</span
      >
      <ElButton v-if="!following" size="small" type="primary" @click="jumpEnd">
        {{ unread ? '有新输出 · 跳到末尾' : '跳到末尾并继续跟随' }}
      </ElButton>
      <span v-else class="keyboard-hint">向上滚动可暂停跟随 · Ctrl+End 到末尾</span>
    </div>
  </section>
</template>

<script setup lang="ts">
  import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
  import { ArrowUpToLine, ArrowDownToLine, Maximize2, Minimize2, RefreshCw } from 'lucide-vue-next'
  import type { LogViewState } from './types'

  const props = defineProps<{
    text: string
    canSummarize?: boolean
    detailed?: boolean
    title: string
    emptyText: string
    loading: boolean
    expanded: boolean
    initialState: LogViewState
  }>()
  const emit = defineEmits<{
    refresh: []
    expand: []
    detail: [value: boolean]
    state: [value: LogViewState]
  }>()
  const viewport = ref<HTMLElement | null>(null)
  const following = ref(props.initialState.following)
  const wrap = ref(props.initialState.wrap)
  const unread = ref(props.initialState.unread)
  let lastTop = props.initialState.top
  let pendingRestoreTop: number | null = props.initialState.following
    ? null
    : props.initialState.top
  let observer: ResizeObserver | undefined
  let disposed = false
  const lineCount = computed(() => {
    if (!props.text) return 0
    let count = 1
    for (let index = 0; index < props.text.length; index++) {
      if (props.text.charCodeAt(index) === 10 && index < props.text.length - 1) count++
    }
    return count
  })
  const saveState = () =>
    emit('state', {
      following: following.value,
      wrap: wrap.value,
      top: pendingRestoreTop ?? viewport.value?.scrollTop ?? lastTop,
      unread: unread.value
    })
  function scrollTo(top: number) {
    if (disposed || !viewport.value) return
    viewport.value.scrollTop = top
    lastTop = viewport.value.scrollTop
    saveState()
  }
  function jumpTop() {
    pendingRestoreTop = null
    following.value = false
    scrollTo(0)
  }
  function jumpEnd() {
    pendingRestoreTop = null
    following.value = true
    unread.value = false
    scrollTo(viewport.value?.scrollHeight || 0)
  }
  function setFollowing(value: boolean) {
    following.value = value
    if (value) jumpEnd()
    else saveState()
  }
  function onScroll() {
    const element = viewport.value
    if (!element) return
    // Ignore horizontal scrolling and events from our own scroll/restore.
    if (Math.abs(element.scrollTop - lastTop) < 1) return
    pendingRestoreTop = null
    lastTop = element.scrollTop
    following.value = element.scrollHeight - element.clientHeight - element.scrollTop <= 24
    if (following.value) unread.value = false
    saveState()
  }
  async function layoutChanged() {
    await nextTick()
    if (disposed) return
    if (following.value) jumpEnd()
    else saveState()
  }
  function restoreLayout() {
    if (following.value) jumpEnd()
    else if (pendingRestoreTop !== null && viewport.value && viewport.value.clientHeight > 0) {
      const top = pendingRestoreTop
      pendingRestoreTop = null
      scrollTo(top)
    }
  }
  watch(
    () => props.text,
    async (value, previous) => {
      if (value === previous) return
      const top = viewport.value?.scrollTop ?? lastTop
      if (!following.value) unread.value = true
      await nextTick()
      if (disposed) return
      // Check the current choice: the user may have scrolled during the update.
      if (following.value) jumpEnd()
      else scrollTo(lastTop === top ? top : lastTop)
    }
  )
  onMounted(async () => {
    await nextTick()
    if (disposed) return
    restoreLayout()
    if (typeof ResizeObserver !== 'undefined' && viewport.value) {
      observer = new ResizeObserver(restoreLayout)
      observer.observe(viewport.value)
    }
  })
  onBeforeUnmount(() => {
    saveState()
    disposed = true
    observer?.disconnect()
  })
</script>

<style scoped lang="scss">
  .log-console {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-width: 0;
    min-height: 0;
    height: 100%;
    overflow: hidden;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    background: var(--el-bg-color);
  }
  .log-toolbar,
  .log-footer {
    display: flex;
    flex: none;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 8px 16px;
    padding: 10px 14px;
  }
  .log-caption,
  .log-actions {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
  }
  .log-caption strong {
    font-size: 14px;
  }
  .log-mode {
    display: inline-flex;
    border: 1px solid var(--el-border-color);
    border-radius: 5px;
    overflow: hidden;
  }
  .log-mode button {
    border: 0;
    padding: 4px 8px;
    background: transparent;
    color: var(--el-text-color-secondary);
    cursor: pointer;
    font: inherit;
    font-size: 12px;
  }
  .log-mode button[aria-pressed='true'] {
    background: var(--el-color-primary-light-9);
    color: var(--el-color-primary);
  }
  .log-mode button:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: -2px;
  }
  .follow-state {
    color: var(--el-color-success);
    font-size: 12px;
  }
  .follow-state.paused {
    color: var(--el-color-warning);
  }
  .log-actions :deep(.el-button + .el-button) {
    margin-left: 0;
  }
  .log-actions :deep(.el-checkbox) {
    margin: 0 4px;
  }
  .log-scroll {
    flex: 1;
    min-height: 0;
    overflow: auto;
    overscroll-behavior: contain;
    overflow-anchor: none;
    scrollbar-gutter: stable;
    scrollbar-color: #718096 #111827;
    background: #0d1525;
    color: #e2e8f0;
  }
  .log-scroll:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: -2px;
  }
  .log-scroll::-webkit-scrollbar {
    width: 14px;
    height: 14px;
  }
  .log-scroll::-webkit-scrollbar-track {
    background: #111827;
  }
  .log-scroll::-webkit-scrollbar-thumb {
    border: 3px solid #111827;
    border-radius: 8px;
    background: #718096;
  }
  .log-scroll::-webkit-scrollbar-thumb:hover {
    background: #a0aec0;
  }
  pre {
    min-width: 100%;
    box-sizing: border-box;
    margin: 0;
    padding: 16px 18px 24px;
    font: 13px/1.7 var(--ax-font-mono);
    tab-size: 4;
    white-space: pre;
    overflow-wrap: normal;
  }
  .wrap-lines {
    white-space: pre-wrap;
    overflow-wrap: anywhere;
  }
  .empty-log {
    color: #94a3b8;
  }
  .log-footer {
    min-height: 28px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
  @media (max-width: 700px) {
    .log-toolbar,
    .log-footer {
      padding: 8px;
    }
    .keyboard-hint {
      display: none;
    }
    pre {
      padding: 12px;
    }
  }
</style>
