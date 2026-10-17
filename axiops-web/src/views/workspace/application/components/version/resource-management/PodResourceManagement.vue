<template>
  <div class="pod-resource-management-full">
    <ElCard class="management-card">
      <ElTabs v-model="activeTab" @tab-change="handleTabChange">
        <!-- Pod管理 -->
        <ElTabPane label="Pod管理" name="pods">
          <PodsManagement
            v-if="loadedTabs.pods"
            :version="version"
            :application="application"
            :cluster="cluster"
            :workspace="workspace"
            :refresh-trigger="refreshTriggers.pods"
            @refresh="handleSuccess"
          />
        </ElTabPane>

        <!-- 事件管理 -->
        <ElTabPane label="事件管理" name="events">
          <EventsManagement
            v-if="loadedTabs.events"
            :version="version"
            :refresh-trigger="refreshTriggers.events"
          />
        </ElTabPane>
      </ElTabs>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, nextTick } from 'vue'
  import type {
    OnecProjectApplication,
    OnecProjectVersion,
    ProjectCluster,
    ProjectWorkspace
  } from '@/api'
  import PodsManagement from '../tabs/common/PodsManagement.vue'
  import EventsManagement from '../tabs/common/EventsManagement.vue'

  interface Props {
    version: OnecProjectVersion
    application: OnecProjectApplication
    cluster: ProjectCluster | null
    workspace: ProjectWorkspace | null
    refreshTrigger?: number
  }

  const props = defineProps<Props>()
  const emit = defineEmits<{ refresh: [] }>()

  const activeTab = ref('pods')
  const loadedTabs = ref({
    pods: false,
    events: false
  })
  const refreshTriggers = ref({
    pods: 0,
    events: 0
  })

  const isInitialized = ref(false)

  const handleTabChange = (tabName: string) => {
    const tab = tabName as keyof typeof loadedTabs.value

    // 标记为已加载（用于 v-if 懒加载）
    if (!loadedTabs.value[tab]) {
      loadedTabs.value[tab] = true
    } else {
    }

    nextTick(() => {
      if (tab in refreshTriggers.value) {
        refreshTriggers.value[tab]++
      }
    })
  }
  // =====================================================

  watch(
    () => props.refreshTrigger,
    (newVal) => {
      if (newVal && newVal > 0 && isInitialized.value) {
        const tab = activeTab.value as keyof typeof refreshTriggers.value
        if (tab in refreshTriggers.value) {
          refreshTriggers.value[tab]++
        }
      }
    }
  )

  onMounted(() => {
    // 首次显示时自动触发当前 tab 的加载
    const tab = activeTab.value as keyof typeof loadedTabs.value
    if (!loadedTabs.value[tab]) {
      loadedTabs.value[tab] = true

      nextTick(() => {
        if (tab in refreshTriggers.value) {
          refreshTriggers.value[tab]++
        }

        nextTick(() => {
          isInitialized.value = true
        })
      })
    }
  })
</script>

<style lang="scss" scoped>
  .pod-resource-management-full {
    .management-card {
      :deep(.el-card__body) {
        padding: 0;
      }

      :deep(.el-tabs__header) {
        padding: 0 20px;
        margin: 0;
        background: #fafafa;
        border-bottom: 1px solid #e4e7ed;

        .el-tabs__nav-wrap::after {
          display: none;
        }

        .el-tabs__item {
          height: 50px;
          line-height: 50px;
          padding: 0 20px;
          font-size: 14px;
          color: #606266;

          &:hover {
            color: #409eff;
          }

          &.is-active {
            color: #409eff;
            font-weight: 500;
          }
        }

        .el-tabs__active-bar {
          height: 2px;
          background-color: #409eff;
        }
      }

      :deep(.el-tabs__content) {
        padding: 24px;
        background: #fafbfc;
      }
    }
  }
</style>
