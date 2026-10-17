<template>
  <ElSelect
    v-model="selectedProjectId"
    class="devops-project-select"
    filterable
    :loading="loading"
    placeholder="选择 DevOps 项目"
    @visible-change="handleVisibleChange"
    @change="handleProjectChange"
  >
    <ElOption
      v-for="project in projects"
      :key="project.id"
      :label="project.name"
      :value="project.id"
    >
      <div class="devops-project-option">
        <span class="project-name">{{ project.name }}</span>
        <span class="project-code">{{ project.code }}</span>
      </div>
    </ElOption>
  </ElSelect>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref, watch } from 'vue'
  import { storeToRefs } from 'pinia'
  import { ElMessage } from 'element-plus'
  import { useDevopsProjectStore } from '@/store/modules/devopsProject'
  import { mittBus } from '@/utils/sys'

  defineOptions({ name: 'ArtDevopsProjectSelector' })

  const devopsProjectStore = useDevopsProjectStore()
  const { projects, selectedProject, loading } = storeToRefs(devopsProjectStore)
  const selectedProjectId = ref('')

  const currentProjectName = computed(() => selectedProject.value?.name || '')

  watch(
    selectedProject,
    (project) => {
      selectedProjectId.value = project?.id || ''
    },
    { immediate: true }
  )

  const handleVisibleChange = async (visible: boolean) => {
    if (visible) {
      await devopsProjectStore.fetchProjects()
    }
  }

  const handleProjectChange = (projectId: string) => {
    devopsProjectStore.setSelectedProject(projectId)
    mittBus.emit('devopsProjectChanged', selectedProject.value)
    if (currentProjectName.value) {
      ElMessage.success(`已切换到「${currentProjectName.value}」项目`)
    }
  }

  onMounted(async () => {
    await devopsProjectStore.fetchProjects()
  })
</script>

<style lang="scss" scoped>
  .devops-project-select {
    width: 220px;
  }

  .devops-project-option {
    display: flex;
    gap: 10px;
    align-items: center;
    justify-content: space-between;
    min-width: 0;
  }

  .project-name,
  .project-code {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .project-name {
    color: var(--el-text-color-primary);
  }

  .project-code {
    flex: 0 0 auto;
    max-width: 130px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
</style>
