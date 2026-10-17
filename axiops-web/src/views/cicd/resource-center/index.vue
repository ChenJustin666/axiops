<template>
  <div class="resource-center">
    <header class="page-header">
      <div>
        <h1>资源配置</h1>
        <p>统一管理 CI/CD 使用的代码源、镜像仓库、构建资源与集群节点。</p>
      </div>
    </header>

    <div class="summary-grid">
      <ElCard v-for="item in summaries" :key="item.title" shadow="never" class="summary-card">
        <div class="summary-icon" :class="item.tone"><component :is="item.icon" :size="20" /></div>
        <div><span>{{ item.title }}</span><strong>{{ item.value }}</strong></div>
      </ElCard>
    </div>

    <ElCard shadow="never" class="resource-card">
      <div class="resource-tabs">
        <button v-for="tab in tabs" :key="tab.key" :class="{ active: active === tab.key }" @click="select(tab.key)">
          <component :is="tab.icon" :size="17" /><span>{{ tab.label }}</span>
        </button>
      </div>
      <div class="tab-content">
        <div v-if="active === 'overview'" class="overview">
          <h2>平台资源</h2>
          <p>代码源负责拉取源码，镜像仓库负责制品推送，构建资源负责执行和缓存。项目与服务只选择这里已授权的资源。</p>
          <div class="quick-grid">
            <button v-for="tab in tabs.slice(1)" :key="tab.key" class="quick-item" @click="select(tab.key)">
              <component :is="tab.icon" :size="20" /><span>{{ tab.label }}</span><ChevronRight :size="16" />
            </button>
          </div>
        </div>
        <component v-else-if="currentTab.component" :is="currentTab.component" class="embedded-page" />
        <div v-else class="redirect-panel">
          <component :is="currentTab.icon" :size="28" />
          <h2>{{ currentTab.label }}</h2>
          <p>进入独立管理页面维护该类资源，配置会被项目和工作流复用。</p>
          <ElButton type="primary" @click="openCurrent">进入管理</ElButton>
        </div>
      </div>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElButton, ElCard } from 'element-plus'
import { GitBranch, Boxes, Cpu, Server, KeyRound, ChevronRight } from 'lucide-vue-next'
import CodeSources from '@/views/cicd/code-sources/index.vue'
import BuildSettings from '@/views/cicd/build-settings/index.vue'
import Repositories from './BuildRegistries.vue'
import Credentials from '@/views/devops/settings/credential/index.vue'
import ClusterManagement from '@/views/clusters/management/index.vue'

const router = useRouter()
const active = ref('overview')
const tabs = [
  { key: 'overview', label: '资源总览', icon: Boxes, path: '' },
  { key: 'code', label: '代码源', icon: GitBranch, component: CodeSources, path: '/cicd/code-sources' },
  { key: 'registry', label: '镜像仓库', icon: Boxes, component: Repositories, path: '/cluster/repositories' },
  { key: 'build', label: 'CI 构建资源', icon: Cpu, component: BuildSettings, path: '/cicd/build-settings' },
  { key: 'cluster', label: '集群与节点', icon: Server, component: ClusterManagement, path: '/clusters/management' },
  { key: 'credentials', label: '凭据授权', icon: KeyRound, component: Credentials, path: '/devops/settings/credential' }
]
const summaries = [
  { title: '代码源', value: '统一管理', icon: GitBranch, tone: 'green' },
  { title: '镜像仓库', value: '平台授权', icon: Boxes, tone: 'blue' },
  { title: '构建资源池', value: '可用', icon: Cpu, tone: 'orange' },
  { title: '集群与节点', value: '集中配置', icon: Server, tone: 'purple' }
]
const currentTab = computed(() => tabs.find((tab) => tab.key === active.value) || tabs[0])
function select(key: string) { active.value = key }
function openCurrent() { if (currentTab.value.path) router.push(currentTab.value.path) }
</script>

<style scoped>
.resource-center { padding: 24px; background: #f5f7fa; min-height: 100%; color: #1f2937; }
.page-header h1 { margin: 0; font-size: 24px; font-weight: 650; }
.page-header p { margin: 8px 0 22px; color: #64748b; }
.summary-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; margin-bottom: 18px; }
.summary-card :deep(.el-card__body) { display: flex; align-items: center; gap: 12px; padding: 18px; }
.summary-card span { display: block; color: #64748b; font-size: 13px; }.summary-card strong { display: block; margin-top: 5px; font-size: 16px; }
.summary-icon { width: 38px; height: 38px; border-radius: 9px; display: grid; place-items: center; }.green { color:#16a34a; background:#dcfce7 }.blue { color:#2563eb; background:#dbeafe }.orange { color:#ea580c; background:#ffedd5 }.purple { color:#7c3aed; background:#ede9fe }
.resource-card :deep(.el-card__body) { padding: 0; }.resource-tabs { display:flex; gap: 4px; padding: 0 18px; border-bottom:1px solid #e5e7eb; }
.resource-tabs button { display:flex; align-items:center; gap:7px; padding:15px 14px; border:0; background:transparent; color:#64748b; cursor:pointer; border-bottom:2px solid transparent; }.resource-tabs button.active { color:#16a34a; border-bottom-color:#16a34a; font-weight:600; }
.tab-content { min-height: 310px; padding: 28px; }.overview h2,.redirect-panel h2 { margin:0 0 8px; font-size:18px; }.overview p,.redirect-panel p { color:#64748b; }.quick-grid { display:grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap:12px; margin-top:24px; }.quick-item { display:flex; align-items:center; gap:10px; padding:16px; border:1px solid #e5e7eb; background:#fff; border-radius:8px; color:#334155; cursor:pointer; text-align:left; }.quick-item svg:last-child { margin-left:auto; color:#94a3b8; }.redirect-panel { text-align:center; padding:42px 0; color:#64748b; }.redirect-panel h2 { color:#1f2937; }
.embedded-page { min-height: 520px; }
@media (max-width: 900px) { .summary-grid { grid-template-columns: repeat(2,1fr); } }.@media (max-width: 600px) { .resource-center { padding:16px; }.summary-grid,.quick-grid { grid-template-columns:1fr; }.resource-tabs { overflow:auto; } }
</style>
