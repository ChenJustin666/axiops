<template>
  <div class="project-nav-bar" :class="{ 'is-compact': compact }">
    <header class="project-nav-head">
      <ElButton
        text
        circle
        :icon="ArrowLeft"
        title="返回项目列表"
        aria-label="返回项目列表"
        @click="router.push('/cicd/projects')"
      />
      <span class="project-icon"><FolderKanban :size="19" /></span>
      <div class="project-title">
        <h1>{{ project?.name || '项目工作区' }}</h1>
        <span class="project-code">{{ project?.code || '-' }}</span>
      </div>
      <div v-if="$slots.crumb" class="project-crumb"><slot name="crumb" /></div>
      <div v-if="!compact" class="head-stats">
        <span>服务 <b>{{ counts?.services ?? 0 }}</b></span>
        <span>环境 <b>{{ counts?.environments ?? 0 }}</b></span>
        <span>工作流 <b>{{ counts?.workflows ?? 0 }}</b></span>
      </div>
      <div class="head-tail">
        <slot name="actions" />
        <ElButton
          v-if="refreshable"
          text
          circle
          :icon="RefreshCw"
          title="刷新"
          aria-label="刷新"
          :loading="refreshing"
          @click="emit('refresh')"
        />
      </div>
    </header>
    <nav v-if="showTabs" class="project-nav" aria-label="项目导航">
      <button
        v-for="item in navItems"
        :key="item.value"
        type="button"
        :class="['nav-item', { 'is-active': active === item.value }]"
        @click="select(item.value)"
      >
        <component :is="item.icon" :size="15" /><span>{{ item.label }}</span
        ><em v-if="item.badge">{{ item.badge }}</em>
      </button>
    </nav>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import { useRouter } from 'vue-router'
  import {
    ArrowLeft,
    Boxes,
    FolderKanban,
    GitCommit,
    LayoutDashboard,
    RefreshCw,
    Server,
    TestTube2,
    Workflow
  } from 'lucide-vue-next'
  import type { AxiOpsProject } from '@/api/axiops'

  const props = withDefaults(
    defineProps<{
      projectId: number
      project?: AxiOpsProject | null
      active?: string
      counts?: { services?: number; environments?: number; workflows?: number }
      compact?: boolean
      showTabs?: boolean
      refreshable?: boolean
      refreshing?: boolean
      autoNavigate?: boolean
    }>(),
    {
      project: null,
      active: 'overview',
      compact: false,
      showTabs: true,
      refreshable: false,
      refreshing: false,
      autoNavigate: true
    }
  )
  const emit = defineEmits<{ select: [string]; refresh: [] }>()
  const router = useRouter()
  const navItems = computed(() => [
    { value: 'overview', label: '概览', icon: LayoutDashboard, badge: 0 },
    { value: 'services', label: '服务配置', icon: Boxes, badge: badge('services') },
    { value: 'environments', label: '环境', icon: Server, badge: badge('environments') },
    { value: 'workflows', label: '工作流', icon: Workflow, badge: badge('workflows') },
    { value: 'tests', label: '测试', icon: TestTube2, badge: 0 },
    { value: 'versions', label: '版本管理', icon: GitCommit, badge: 0 }
  ])
  const badge = (key: 'services' | 'environments' | 'workflows') => {
    if (props.compact || !props.counts) return 0
    return props.counts[key] || 0
  }
  const select = (value: string) => {
    emit('select', value)
    if (!props.autoNavigate) return
    void router.push({ path: `/cicd/projects/${props.projectId}`, query: { tab: value } })
  }
</script>

<style scoped lang="scss">
  .project-nav-bar {
    flex: 0 0 auto;
    background: var(--el-bg-color);
  }
  .project-nav-head {
    display: flex;
    align-items: center;
    gap: 12px;
    height: 68px;
    padding: 0 22px;
    border-bottom: 1px solid var(--ax-line);
  }
  .is-compact .project-nav-head {
    height: 56px;
  }
  .project-icon {
    display: inline-flex;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    width: 34px;
    height: 34px;
    border-radius: 7px;
    color: #fff;
    background: #0f766e;
    box-shadow: 0 4px 10px rgb(15 118 110 / 18%);
  }
  .project-title {
    display: flex;
    align-items: baseline;
    gap: 8px;
    min-width: 0;
  }
  h1 {
    margin: 0;
    overflow: hidden;
    color: var(--el-text-color-primary);
    font-family: var(--ax-font-display);
    font-size: 17px;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .is-compact h1 {
    font-size: 15px;
  }
  .project-code {
    flex: 0 0 auto;
    color: var(--el-text-color-secondary);
    font: 11px var(--ax-font-mono);
  }
  .project-crumb {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
    margin-left: 4px;
    padding-left: 12px;
    border-left: 1px solid var(--ax-line);
    font-size: 13px;
  }
  .head-stats {
    display: flex;
    flex: 0 0 auto;
    gap: 14px;
    margin-left: auto;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
  .head-stats b {
    color: var(--el-text-color-primary);
  }
  .head-tail {
    display: flex;
    flex: 0 0 auto;
    align-items: center;
    gap: 8px;
    margin-left: auto;
  }
  .head-stats + .head-tail {
    margin-left: 12px;
  }
  .project-nav {
    display: flex;
    gap: 2px;
    padding: 0 22px;
    border-bottom: 1px solid var(--ax-line);
    overflow-x: auto;
  }
  .nav-item {
    display: inline-flex;
    align-items: center;
    gap: 7px;
    min-height: 48px;
    padding: 0 14px;
    border: 0;
    border-bottom: 2px solid transparent;
    background: transparent;
    color: var(--el-text-color-secondary);
    font-family: inherit;
    font-size: 15px;
    white-space: nowrap;
    cursor: pointer;
  }
  .is-compact .nav-item {
    min-height: 42px;
  }
  .nav-item:hover,
  .nav-item.is-active {
    color: #0f766e;
  }
  .nav-item.is-active {
    border-bottom-color: #0f766e;
    font-weight: 600;
  }
  .nav-item em {
    padding: 1px 5px;
    border-radius: 8px;
    background: #ecfdf5;
    color: #047857;
    font-size: 10px;
    font-style: normal;
  }
  @media (max-width: 1180px) {
    .head-stats {
      display: none;
    }
  }
</style>
