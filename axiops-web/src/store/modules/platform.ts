/**
 * 平台状态管理模块
 *
 * 提供多系统平台相关的状态管理
 *
 * 当前平台按浏览器标签页隔离，避免多个平台页面互相覆盖。
 *
 * @module store/modules/platform
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getUserPlatformsApi, type PlatformSysPlatform } from '@/api/portal/platform'
import { ElMessage } from 'element-plus'

const PLATFORM_VISUAL_MAP: Record<string, { icon: string; shortName: string; accent: string }> = {
  portal: {
    icon: 'ri:shield-user-line',
    shortName: '门',
    accent: '#2563eb'
  },
  container: {
    icon: 'ri:ship-2-line',
    shortName: '容',
    accent: '#3b82f6'
  },
  microservice: {
    icon: 'ri:links-line',
    shortName: '微',
    accent: '#10b981'
  },
  devops: {
    icon: 'ri:rocket-2-line',
    shortName: 'D',
    accent: '#5d87ff'
  }
}

/**
 * 平台 Store
 */
export const usePlatformStore = defineStore(
  'platform',
  () => {
    // ========== 状态定义 ==========

    /** 用户可用的平台列表 */
    const platformList = ref<PlatformSysPlatform[]>([])

    /** 当前项目可用的平台列表 */
    const projectPlatformList = ref<PlatformSysPlatform[]>([])

    /** 当前选中的平台 ID */
    const currentPlatformId = ref<number | null>(null)

    /** 当前选中的平台信息 */
    const currentPlatform = computed(() => {
      if (!currentPlatformId.value) return null
      return platformList.value.find((p) => p.id === currentPlatformId.value) || null
    })

    /** 可用的平台列表（仅 isEnable=1） */
    const enabledPlatforms = computed(() => platformList.value.filter((p) => p.isEnable === 1))

    /** 当前项目可用的平台列表（仅 isEnable=1） */
    const enabledProjectPlatforms = computed(() => {
      return projectPlatformList.value.filter((p) => p.isEnable === 1)
    })

    /** 是否有可用平台 */
    const hasEnabledPlatform = computed(() => {
      return enabledPlatforms.value.length > 0
    })

    // ========== Actions ==========

    /**
     * 获取用户可用的平台列表
     * @param projectId 项目ID
     * @returns 平台列表
     */
    async function fetchUserPlatforms(): Promise<PlatformSysPlatform[]> {
      try {
        const data = await getUserPlatformsApi()
        platformList.value = data || []
        if (currentPlatformId.value && !isPlatformEnabled(currentPlatformId.value)) {
          currentPlatformId.value = null
        }
        return platformList.value
      } catch (error) {
        console.error('获取用户平台列表失败:', error)
        platformList.value = []
        throw error
      }
    }

    async function fetchProjectPlatforms(
      projectId?: number | null
    ): Promise<PlatformSysPlatform[]> {
      if (!projectId) {
        projectPlatformList.value = []
        return []
      }
      try {
        const data = await getUserPlatformsApi({ projectId })
        projectPlatformList.value = data || []
        return projectPlatformList.value
      } catch (error) {
        console.error('获取项目平台列表失败:', error)
        projectPlatformList.value = []
        throw error
      }
    }

    function clearProjectPlatforms(): void {
      projectPlatformList.value = []
    }

    /**
     * 选择默认平台
     * 优先选择 isDefault=1 且 isEnable=1 的平台
     * 如果没有默认平台，选择第一个 isEnable=1 的平台
     * @returns 选中的平台 ID，如果没有可用平台则返回 null
     */
    function selectDefaultPlatform(): number | null {
      // 优先选择默认平台
      const defaultPlatform = platformList.value.find((p) => p.isDefault === 1 && p.isEnable === 1)

      if (defaultPlatform) {
        currentPlatformId.value = defaultPlatform.id
        console.log('选择默认平台:', defaultPlatform.platformName, defaultPlatform.id)
        return defaultPlatform.id
      }

      // 如果没有默认平台，选择第一个启用的平台
      const firstEnabledPlatform = platformList.value.find((p) => p.isEnable === 1)

      if (firstEnabledPlatform) {
        currentPlatformId.value = firstEnabledPlatform.id
        console.log(
          '选择第一个启用的平台:',
          firstEnabledPlatform.platformName,
          firstEnabledPlatform.id
        )
        return firstEnabledPlatform.id
      }

      // 没有可用平台
      currentPlatformId.value = null
      console.warn('没有可用的平台')
      return null
    }

    /**
     * 设置当前平台
     * @param platformId 平台 ID
     */
    function setCurrentPlatform(platformId: number): void {
      const platform = platformList.value.find((p) => p.id === platformId)
      if (!platform) {
        console.error('平台不存在:', platformId)
        return
      }

      if (platform.isEnable !== 1) {
        ElMessage.warning('该平台已被禁用，无法切换')
        return
      }

      currentPlatformId.value = platformId
      console.log('切换到平台:', platform.platformName, platformId)
    }

    /**
     * 验证平台是否可用
     * @param platformId 平台 ID
     * @returns 是否可用
     */
    function isPlatformEnabled(platformId: number): boolean {
      const platform = platformList.value.find((p) => p.id === platformId)
      return platform ? platform.isEnable === 1 : false
    }

    /**
     * 清空平台状态
     */
    function clearPlatformState(): void {
      platformList.value = []
      projectPlatformList.value = []
      currentPlatformId.value = null
      console.log('平台状态已清空')
    }

    /**
     * 设置平台列表（用于测试或特殊场景）
     * @param platforms 平台列表
     */
    function setPlatformList(platforms: PlatformSysPlatform[]): void {
      platformList.value = platforms
      if (currentPlatformId.value && !isPlatformEnabled(currentPlatformId.value)) {
        currentPlatformId.value = null
      }
    }

    function getPlatformVisual(platform?: Partial<PlatformSysPlatform> | null) {
      const code = String(platform?.platformCode || '').toLowerCase()
      const visual = PLATFORM_VISUAL_MAP[code]
      if (visual) {
        return {
          icon: platform?.platformIcon || visual.icon,
          shortName: visual.shortName,
          accent: visual.accent
        }
      }

      const platformName = String(platform?.platformName || '').trim()
      return {
        icon: platform?.platformIcon || 'ri:apps-2-line',
        shortName: platformName.slice(0, 1).toUpperCase() || '平',
        accent: '#5d87ff'
      }
    }

    // ========== 返回 ==========

    return {
      // 状态
      platformList,
      projectPlatformList,
      currentPlatformId,
      currentPlatform,
      enabledPlatforms,
      enabledProjectPlatforms,
      hasEnabledPlatform,

      // Actions
      fetchUserPlatforms,
      fetchProjectPlatforms,
      clearProjectPlatforms,
      selectDefaultPlatform,
      setCurrentPlatform,
      isPlatformEnabled,
      clearPlatformState,
      setPlatformList,
      getPlatformVisual
    }
  },
  {
    // 当前平台必须按标签页隔离，否则多平台同时操作会互相覆盖。
    persist: {
      key: 'platform',
      storage: sessionStorage,
      paths: ['currentPlatformId']
    }
  }
)
