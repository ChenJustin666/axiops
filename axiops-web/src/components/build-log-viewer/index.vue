<template>
  <div class="build-log-viewer" :class="{ 'fill-container': fill }">
    <LogConsole
      v-if="!expanded"
      :text="displayText"
      :can-summarize="summaryText !== undefined"
      :detailed="detailed"
      @detail="detailed = $event"
      :title="title"
      :empty-text="emptyText"
      :loading="loading"
      :initial-state="viewState"
      :expanded="false"
      @state="viewState = $event"
      @refresh="emit('refresh')"
      @expand="expanded = true"
    />
    <ElDialog
      v-model="expanded"
      :title="title"
      fullscreen
      append-to-body
      destroy-on-close
      :close-on-click-modal="false"
      class="axiops-log-fullscreen"
    >
      <LogConsole
        v-if="expanded"
        :text="displayText"
        :can-summarize="summaryText !== undefined"
        :detailed="detailed"
        @detail="detailed = $event"
        :title="title"
        :empty-text="emptyText"
        :loading="loading"
        :initial-state="viewState"
        :expanded="true"
        @state="viewState = $event"
        @refresh="emit('refresh')"
        @expand="expanded = false"
      />
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, ref } from 'vue'
  import LogConsole from './LogConsole.vue'
  import type { LogViewState } from './types'
  const props = withDefaults(
    defineProps<{
      text: string
      summaryText?: string
      title?: string
      emptyText?: string
      loading?: boolean
      fill?: boolean
    }>(),
    { title: '构建日志', emptyText: '等待日志输出…', loading: false, fill: false }
  )
  const emit = defineEmits<{ refresh: [] }>()
  const expanded = ref(false)
  const detailed = ref(false)
  const displayText = computed(() =>
    detailed.value || props.summaryText === undefined ? props.text : props.summaryText
  )
  const viewState = ref<LogViewState>({ following: true, wrap: true, top: 0, unread: false })
</script>

<style scoped lang="scss">
  .build-log-viewer {
    height: 72vh;
    height: 72dvh;
    min-height: 340px;
    display: flex;
    flex-direction: column;
    min-width: 0;
    resize: vertical;
    overflow: hidden;
  }
  .fill-container {
    resize: none;
    flex: 1;
    min-height: 0;
    height: 100%;
  }
  :global(.axiops-log-fullscreen) {
    display: flex;
    flex-direction: column;
    overflow: hidden;
    padding: 12px;
  }
  :global(.axiops-log-fullscreen .el-dialog__body) {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    padding: 0;
  }
  :global(.axiops-log-fullscreen .el-dialog__header) {
    flex: none;
    padding-bottom: 12px;
  }
  @media (max-height: 500px) {
    .build-log-viewer {
      min-height: 230px;
    }
    .fill-container {
      min-height: 0;
    }
  }
</style>
