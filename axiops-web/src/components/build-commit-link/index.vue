<template>
  <span class="build-commit-link">
    <a v-if="link" :href="link" :title="hint" target="_blank" rel="noopener noreferrer" @click.stop>
      <code>{{ commit?.slice(0, 8) }}</code
      ><span v-if="subject">{{ subject }}</span>
    </a>
    <span v-else :title="hint"
      ><code>{{ commit?.slice(0, 8) || '暂无 Commit' }}</code
      ><span v-if="subject">{{ subject }}</span></span
    >
  </span>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  const props = defineProps<{ commit?: string; subject?: string; url?: string; author?: string }>()
  const link = computed(() => {
    if (!props.commit || !props.url) return ''
    try {
      const url = new URL(props.url)
      return ['https:', 'http:'].includes(url.protocol) && !url.username && !url.password
        ? url.href
        : ''
    } catch {
      return ''
    }
  })
  const hint = computed(() =>
    [
      props.subject,
      props.author,
      props.commit,
      link.value ? '打开仓库提交详情' : '未保存可用的仓库链接'
    ]
      .filter(Boolean)
      .join('\n')
  )
</script>

<style scoped>
  .build-commit-link {
    display: block;
    min-width: 0;
    max-width: 100%;
  }
  .build-commit-link > a,
  .build-commit-link > span {
    display: flex;
    align-items: baseline;
    gap: 7px;
    min-width: 0;
    color: #64748b;
    font-size: 12px;
  }
  .build-commit-link > a {
    color: #2563eb;
    text-decoration: none;
  }
  .build-commit-link a:hover {
    text-decoration: underline;
  }
  .build-commit-link code {
    flex-shrink: 0;
    font-size: 11px;
  }
  .build-commit-link span span,
  .build-commit-link a span {
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .build-commit-link a:focus-visible {
    outline: 2px solid #93c5fd;
    outline-offset: 2px;
  }
</style>
