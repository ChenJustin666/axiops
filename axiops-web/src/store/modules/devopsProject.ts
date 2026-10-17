import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'
import { listDevopsProjectApi, type DevopsProject } from '@/api/devops/project'
import { useProjectStore } from '@/store/modules/project'
import { usePlatformStore } from '@/store/modules/platform'

export const useDevopsProjectStore = defineStore('devopsProject', () => {
  const projectStore = useProjectStore()
  const platformStore = usePlatformStore()
  const projects = ref<DevopsProject[]>([])
  const loading = ref(false)
  const initialized = ref(false)
  let pendingFetch: Promise<DevopsProject[]> | null = null

  const selectedProject = computed(() => {
    const portalProjectUuid = projectStore.selectedProject?.uuid || ''
    if (!portalProjectUuid) {
      return null
    }
    return projects.value.find((item) => item.portalProjectUuid === portalProjectUuid) || null
  })
  const selectedProjectId = computed(() => selectedProject.value?.id || '')

  const fetchProjects = async (force = false) => {
    if (!force && initialized.value && projects.value.length > 0) {
      return projects.value
    }
    if (pendingFetch) {
      return pendingFetch
    }
    loading.value = true
    pendingFetch = (async () => {
      if (!projectStore.initialized) {
        await projectStore.initializeStore()
      }
      if (!platformStore.currentPlatformId) {
        projects.value = []
        initialized.value = true
        return projects.value
      }
      const res = await listDevopsProjectApi({ page: 1, pageSize: 200, status: 1 })
      projects.value = res.items || []
      initialized.value = true
      return projects.value
    })()
    try {
      return await pendingFetch
    } finally {
      pendingFetch = null
      loading.value = false
    }
  }

  const setSelectedProject = (projectId: string) => {
    const selectedDevopsProject = projects.value.find((item) => item.id === projectId) || null
    const project =
      projectStore.projects.find(
        (item) => item.uuid === selectedDevopsProject?.portalProjectUuid
      ) || null
    if (project) {
      projectStore.selectProject(project)
    }
  }

  const clear = () => {
    projects.value = []
    initialized.value = false
  }

  watch(
    () => platformStore.currentPlatformId,
    (platformId, previousPlatformId) => {
      if (!platformId) {
        clear()
        return
      }
      if (previousPlatformId && previousPlatformId !== platformId) {
        projects.value = []
        initialized.value = false
      }
    }
  )

  return {
    projects,
    selectedProject,
    selectedProjectId,
    loading,
    initialized,
    fetchProjects,
    setSelectedProject,
    clear
  }
})
