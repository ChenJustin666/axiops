<template>
  <div class="devops-dashboard-cockpit">
    <PipelineDashboardView scope="global" />
  </div>
</template>

<script setup lang="ts">
  import PipelineDashboardView from '../components/PipelineDashboardView.vue'

  defineOptions({ name: 'DevOpsDashboardCockpit' })
</script>

<style scoped lang="scss">
  .devops-dashboard-cockpit {
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }
</style>
