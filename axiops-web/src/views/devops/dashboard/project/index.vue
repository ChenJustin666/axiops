<template>
  <div class="devops-dashboard-page">
    <PipelineDashboardView
      scope="project"
      :project-id="projectStore.selectedProjectId"
      :project-name="projectStore.selectedProject?.name || ''"
    />
  </div>
</template>

<script setup lang="ts">
  import { onMounted } from 'vue'
  import { useDevopsProjectStore } from '@/store/modules/devopsProject'
  import PipelineDashboardView from '../components/PipelineDashboardView.vue'

  defineOptions({ name: 'DevOpsDashboardProject' })

  const projectStore = useDevopsProjectStore()

  onMounted(() => {
    if (!projectStore.initialized) {
      projectStore.fetchProjects()
    }
  })
</script>

<style scoped lang="scss">
  .devops-dashboard-page {
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }
</style>
