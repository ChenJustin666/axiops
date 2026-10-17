// src/store/modules/project.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { listProjectsApi } from '@/api/portal/project'
import { usePlatformStore } from '@/store/modules/platform'
import { useUserStore } from '@/store/modules/user'

// ========== 类型定义 ==========
export interface Project {
  id: number
  name: string
  uuid: string
  isSystem: number
}

// ========== 存储键名 ==========
const STORAGE_KEY = 'art-selected-project'
const contextStorage = sessionStorage

// ========== Pinia Store ==========
export const useProjectStore = defineStore('project', () => {
  // 状态
  const projects = ref<Project[]>([])
  const selectedProject = ref<Project | null>(null)
  const loading = ref(false)
  const initialized = ref(false) // 添加初始化标志

  // 计算属性
  const hasSelectedProject = computed(() => !!selectedProject.value)
  const selectedProjectId = computed(() => selectedProject.value?.id)

  const pickDefaultProject = () => {
    return projects.value.find((item) => item.isSystem !== 1) || projects.value[0] || null
  }

  const selectDefaultProject = () => {
    const project = pickDefaultProject()
    selectProject(project)
    return project
  }

  // 初始化 - 从当前标签页存储加载
  const initializeStore = async () => {
    if (initialized.value) return // 防止重复初始化

    try {
      const saved = contextStorage.getItem(STORAGE_KEY)
      if (saved) {
        const savedProject = JSON.parse(saved)
        // 获取项目列表以验证保存的项目是否仍然存在
        await fetchProjects()

        // 验证保存的项目是否在列表中
        const exists = projects.value.find((p) => p.id === savedProject.id)
        if (exists) {
          selectedProject.value = exists
        } else {
          selectDefaultProject()
        }
      } else {
        await fetchProjects()
        selectDefaultProject()
      }
    } catch (error) {
      console.error('Failed to initialize project store:', error)
      contextStorage.removeItem(STORAGE_KEY)
      selectedProject.value = null
    } finally {
      initialized.value = true
    }
  }

  // 获取项目列表
  const fetchProjects = async (name?: string, platformId?: number | null) => {
    loading.value = true
    try {
      const platformStore = usePlatformStore()
      const currentPlatformId = platformId ?? platformStore.currentPlatformId
      if (!currentPlatformId) {
        projects.value = []
        selectedProject.value = null
        contextStorage.removeItem(STORAGE_KEY)
        return []
      }
      const isSuperAdmin = Array.isArray(useUserStore().info?.roles)
        ? useUserStore().info!.roles!.some((role) => String(role).toUpperCase() === 'SUPER_ADMIN')
        : false
      const response = await listProjectsApi({
        page: 1,
        pageSize: 1000,
        name,
        platformId: currentPlatformId || undefined,
        isSystem: isSuperAdmin ? -1 : 0
      })
      projects.value = (response.data || []).map((item) => ({
        id: item.id,
        name: item.name,
        uuid: item.uuid,
        isSystem: item.isSystem
      }))

      // 验证已选择的项目是否仍存在
      if (selectedProject.value) {
        const exists = projects.value.find((p) => p.id === selectedProject.value!.id)
        if (!exists) {
          selectDefaultProject()
        } else {
          selectedProject.value = exists
          // 更新当前标签页存储为最新数据
          contextStorage.setItem(STORAGE_KEY, JSON.stringify(exists))
        }
      } else {
        selectDefaultProject()
      }

      return projects.value
    } catch (error) {
      console.error('获取项目列表失败:', error)
      projects.value = []
      return []
    } finally {
      loading.value = false
    }
  }

  // 选择项目
  const selectProject = (project: Project | null) => {
    selectedProject.value = project

    // 持久化到当前标签页存储
    if (project) {
      contextStorage.setItem(STORAGE_KEY, JSON.stringify(project))
    } else {
      contextStorage.removeItem(STORAGE_KEY)
    }
  }

  // 清空选择
  const clearSelection = () => {
    selectedProject.value = null
    contextStorage.removeItem(STORAGE_KEY)
  }

  // 确保store已初始化
  const ensureInitialized = async () => {
    if (!initialized.value) {
      await initializeStore()
    }
    return selectedProject.value
  }

  // 初始化时立即从当前标签页存储恢复
  const savedProject = contextStorage.getItem(STORAGE_KEY)
  if (savedProject) {
    try {
      selectedProject.value = JSON.parse(savedProject)
    } catch (error) {
      console.error('Failed to parse saved project:', error)
    }
  }
  const clearAllStoredData = () => {
    selectedProject.value = null
    projects.value = []
    loading.value = false
    initialized.value = false
    contextStorage.removeItem(STORAGE_KEY)
    console.log('[项目Store] 已清空所有项目数据')
  }

  const reloadForPlatform = async (platformId?: number | null) => {
    initialized.value = false
    await fetchProjects(undefined, platformId)
    initialized.value = true
    return selectedProject.value
  }

  return {
    // 状态
    projects,
    selectedProject,
    loading,
    initialized,

    // 计算属性
    hasSelectedProject,
    selectedProjectId,

    // 方法
    fetchProjects,
    selectProject,
    clearSelection,
    initializeStore,
    ensureInitialized,
    reloadForPlatform,
    selectDefaultProject,
    clearAllStoredData
  }
})
