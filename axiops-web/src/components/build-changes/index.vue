<template>
  <section class="build-changes" aria-label="构建代码变更">
    <header
      ><strong>构建 #{{ detail?.build_number || buildId }} · 代码变更</strong
      ><button type="button" :disabled="loading" @click="load">{{
        loading ? '读取中…' : '刷新'
      }}</button></header
    >
    <p v-if="error" role="alert" class="error">{{ error }}</p>
    <template v-if="detail">
      <div class="change-meta"
        ><span>分支 {{ detail.git_branch || '-' }}</span
        ><BuildCommitLink
          :commit="detail.git_commit"
          :url="detail.git_commit_url"
          :subject="detail.git_message"
          :author="detail.git_author"
      /></div>
      <template v-if="changes">
        <p class="change-hint">{{ baselineLabel }}</p>
        <p v-if="changes.status === 'first_build'" class="change-hint"
          >首次构建，没有同仓库、同分支的成功构建基准；展示当前提交。</p
        >
        <p v-if="changes.status === 'unavailable'" class="change-hint"
          >比较历史暂不可用（可能是浅克隆、历史已改写或网络超时），仅展示已采集的当前提交，不代表没有修改。</p
        >
        <ol v-if="changes.commits?.length" class="commit-list">
          <li v-for="commit in changes.commits" :key="commit.commit">
            <BuildCommitLink
              :commit="commit.commit"
              :url="commit.url"
              :subject="commit.subject"
              :author="commit.author"
            />
            <small>{{ commit.author || '未知作者' }} · {{ date(commit.authored_at) }}</small>
          </li>
        </ol>
        <p
          v-else-if="changes.status === 'complete' && !changes.commits_truncated"
          class="change-hint"
          >与基准构建相同，没有新增提交。</p
        >
        <p v-if="changes.commits_truncated" class="change-hint"
          >提交列表已截断，最多展示最近 100 条；完整差异请在仓库中查看。</p
        >
        <details v-if="changes.files?.length || changes.files_truncated" class="changed-files">
          <summary
            >变更文件 · {{ changes.files?.length || 0
            }}{{ changes.files_truncated ? '+' : '' }}</summary
          >
          <ul
            ><li v-for="file in changes.files" :key="`${file.action}:${file.path}`"
              ><span :class="`action-${file.action}`">{{ file.action }}</span
              ><code>{{ file.path }}</code></li
            ></ul
          >
          <small v-if="changes.files_truncated">文件列表已截断，最多展示 200 项。</small>
        </details>
        <a
          v-if="compareLink"
          :href="compareLink"
          target="_blank"
          rel="noopener noreferrer"
          class="compare-link"
          >在仓库查看完整差异 ↗</a
        >
      </template>
      <p v-else class="change-hint">{{ emptyLabel }}</p>
    </template>
  </section>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, ref, watch } from 'vue'
  import { getBuildRunApi, type AxiOpsBuildRun } from '@/api/axiops'
  import BuildCommitLink from '@/components/build-commit-link/index.vue'
  const props = defineProps<{ buildId: number }>()
  const detail = ref<AxiOpsBuildRun | null>(null),
    error = ref(''),
    loading = ref(false)
  let version = 0,
    disposed = false,
    timer: ReturnType<typeof setTimeout> | undefined
  const changes = computed(() => detail.value?.git_changes)
  const terminal = computed(() =>
    ['success', 'failed', 'timeout', 'cancelled'].includes(detail.value?.status || '')
  )
  const emptyLabel = computed(() =>
    terminal.value
      ? detail.value?.git_commit_url
        ? '未保存本次变更明细，可点击上方 Commit 到仓库查看该提交。'
        : '此次构建未保存代码变更记录（历史构建或检出未完成）；这不代表没有修改。'
      : '等待代码检出和变更采集，稍后自动更新。'
  )
  const baselineLabel = computed(() =>
    changes.value?.base_build_number
      ? `比较基准：同服务 / 仓库 / 分支的上次成功构建 #${changes.value.base_build_number} · ${changes.value.base_commit?.slice(0, 8)}`
      : '构建时的代码快照'
  )
  const date = (value: string) => new Date(value).toLocaleString('zh-CN', { hour12: false })
  const compareLink = computed(() => {
    if (!changes.value?.base_commit || !changes.value.compare_url) return ''
    try {
      const url = new URL(changes.value.compare_url)
      return ['http:', 'https:'].includes(url.protocol) && !url.username && !url.password
        ? url.href
        : ''
    } catch {
      return ''
    }
  })
  const load = async () => {
    const request = ++version
    if (timer) clearTimeout(timer)
    loading.value = true
    try {
      const result = await getBuildRunApi(props.buildId)
      if (disposed || request !== version) return
      detail.value = result
      error.value = ''
    } catch (e) {
      if (!disposed && request === version)
        error.value = e instanceof Error ? e.message : '代码变更读取失败，请重试'
    } finally {
      if (!disposed && request === version) {
        loading.value = false
        if (!terminal.value && !error.value) timer = setTimeout(load, 4000)
      }
    }
  }
  watch(
    () => props.buildId,
    () => {
      detail.value = null
      error.value = ''
      void load()
    },
    { immediate: true }
  )
  onBeforeUnmount(() => {
    disposed = true
    version++
    if (timer) clearTimeout(timer)
  })
</script>

<style scoped>
  .build-changes {
    padding: 14px 16px;
    border: 1px solid #e2e8f0;
    border-radius: 7px;
    background: #fff;
    margin: 10px 0;
    min-width: 0;
  }
  header,
  .change-meta {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
  }
  header strong {
    color: #334155;
    font-size: 13px;
  }
  header button {
    border: 0;
    background: transparent;
    color: #2563eb;
    cursor: pointer;
    font-size: 12px;
  }
  .change-meta {
    justify-content: flex-start;
    margin-top: 10px;
    font-size: 12px;
    color: #64748b;
  }
  .change-hint {
    font-size: 12px;
    color: #64748b;
    line-height: 1.7;
  }
  .commit-list {
    list-style: none;
    padding: 0;
    margin: 8px 0;
  }
  .commit-list li {
    padding: 9px 0;
    border-top: 1px solid #f1f5f9;
  }
  .commit-list small {
    display: block;
    margin-top: 5px;
    font-size: 11px;
    color: #64748b;
  }
  .commit-list :deep(.build-commit-link a span),
  .commit-list :deep(.build-commit-link span span) {
    white-space: normal;
    overflow-wrap: anywhere;
  }
  .changed-files {
    margin: 12px 0;
    font-size: 12px;
    color: #475569;
  }
  summary {
    cursor: pointer;
  }
  .changed-files ul {
    max-height: 240px;
    overflow: auto;
    list-style: none;
    padding: 0;
  }
  .changed-files li {
    display: flex;
    gap: 9px;
    margin: 6px 0;
    overflow-wrap: anywhere;
  }
  .action-A {
    color: #15803d;
  }
  .action-D {
    color: #b91c1c;
  }
  .action-M {
    color: #b45309;
  }
  .compare-link {
    font-size: 12px;
    color: #2563eb;
  }
  .error {
    color: #b91c1c;
    font-size: 12px;
  }
</style>
