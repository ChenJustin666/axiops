import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { searchClusterApi, type Cluster } from '@/api/manager'

export function usePollingClusters(storageKey: string) {
  const clusters = ref<Cluster[]>([])
  const loading = ref(false)
  const selectedClusterId = ref<number | null>(loadClusterId(storageKey))

  const selectedCluster = computed(
    () => clusters.value.find((item) => item.id === selectedClusterId.value) || null
  )
  const selectedClusterUuid = computed(() => selectedCluster.value?.uuid || '')
  const selectedClusterName = computed(() => selectedCluster.value?.name || '')

  async function loadClusters() {
    loading.value = true
    try {
      const resp = await searchClusterApi({ page: 1, pageSize: 200 })
      clusters.value = resp.items || []
      if (
        selectedClusterId.value &&
        !clusters.value.some((item) => item.id === selectedClusterId.value)
      ) {
        selectedClusterId.value = null
        saveClusterId(storageKey, null)
      }
    } catch (error: any) {
      ElMessage.error(error?.message || '获取集群列表失败')
    } finally {
      loading.value = false
    }
  }

  function setCluster(id: number | null) {
    selectedClusterId.value = id
    saveClusterId(storageKey, id)
  }

  return {
    clusters,
    loading,
    selectedClusterId,
    selectedCluster,
    selectedClusterUuid,
    selectedClusterName,
    loadClusters,
    setCluster
  }
}

function loadClusterId(key: string): number | null {
  try {
    const raw = sessionStorage.getItem(key)
    return raw ? Number(raw) : null
  } catch {
    return null
  }
}

function saveClusterId(key: string, value: number | null) {
  try {
    if (value === null || value === undefined) {
      sessionStorage.removeItem(key)
      return
    }
    sessionStorage.setItem(key, String(value))
  } catch {
    // ignore
  }
}
