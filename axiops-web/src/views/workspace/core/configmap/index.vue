<template>
  <div class="configmap-page art-full-height">
    <!-- 搜索栏 -->
    <ConfigMapSearch
      v-show="showSearchBar"
      v-model="searchForm"
      @search="handleSearch"
      @reset="resetSearchParams"
    />

    <ElCard
      class="art-table-card"
      shadow="never"
      :style="{ 'margin-top': showSearchBar ? '12px' : '0' }"
    >
      <!-- 表格头部 -->
      <ArtTableHeader
        v-model:columns="columns"
        v-model:showSearchBar="showSearchBar"
        :loading="loading"
        :showZebra="true"
        :showBorder="true"
        :showHeaderBackground="true"
        :fullClass="'art-page-view'"
        :layout="'search,refresh,size,fullscreen,columns,settings'"
        @refresh="loadData"
      >
        <template #left>
          <ElSpace wrap>
            <ElButton type="primary" @click="handleCreate">新 增</ElButton>
          </ElSpace>
        </template>
      </ArtTableHeader>

      <!-- 数据表格 -->
      <ArtTable
        :loading="loading"
        :data="data"
        :columns="columns"
        :pagination="pagination"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      />
    </ElCard>

    <!-- 新增/编辑弹窗 -->
    <ConfigMapDialog
      v-model="dialogVisible"
      :dialog-type="dialogType"
      :configmap-data="currentConfigMap"
      :workspace="workspace"
      @success="loadData"
    />

    <!-- 详情查看弹窗 -->
    <ConfigMapDetailDialog
      v-model="detailVisible"
      :configmap-name="currentConfigMapName"
      :workspace="workspace"
    />

    <!-- 引用查看弹窗 -->
    <ConfigMapUsageDialog
      v-model="usageVisible"
      :configmap-name="currentConfigMapName"
      :workspace="workspace"
    />
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, watch, onMounted } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import {
    getConfigMapListApi,
    deleteConfigMapApi,
    getConfigMapUsageApi,
    getConfigMapYamlApi,
    type ConfigMapListItem,
    type ProjectWorkspace,
    type ProjectCluster
  } from '@/api'
  import ConfigMapSearch from './components/ConfigMapSearch.vue'
  import ConfigMapDialog from './components/ConfigMapDialog.vue'
  import ConfigMapDetailDialog from './components/ConfigMapDetailDialog.vue'
  import ConfigMapUsageDialog from './components/ConfigMapUsageDialog.vue'
  import { useConfigMapTableConfig } from './composables/table-config'

  defineOptions({ name: 'ConfigMapManagement' })

  interface Props {
    cluster: ProjectCluster | null
    workspace: ProjectWorkspace | null
    refreshTrigger?: number
  }

  const props = defineProps<Props>()

  const { loading, showSearchBar, deleteLoadingMap, downloadLoadingMap, createTableColumns } =
    useConfigMapTableConfig()

  const data = ref<ConfigMapListItem[]>([])

  const searchForm = ref({
    search: '',
    labelSelector: ''
  })

  const searchParams = ref<any>({})

  const pagination = ref({
    current: 1,
    size: 20,
    total: 0
  })

  const dialogVisible = ref(false)
  const detailVisible = ref(false)
  const usageVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const currentConfigMap = ref<ConfigMapListItem | undefined>(undefined)
  const currentConfigMapName = ref('')

  const workloadId = computed(() => props.workspace?.id || 0)

  // 用于跟踪是否已完成初始加载，避免重复请求
  const hasInitialLoaded = ref(false)
  // 记录上一次的 refreshTrigger 值，用于判断是否是真正的刷新
  const lastRefreshTrigger = ref(props.refreshTrigger || 0)

  const handleView = (row: ConfigMapListItem) => {
    currentConfigMapName.value = row.name
    detailVisible.value = true
  }

  const handleEdit = (row: ConfigMapListItem) => {
    dialogType.value = 'edit'
    currentConfigMap.value = row
    dialogVisible.value = true
  }

  const handleViewUsage = (row: ConfigMapListItem) => {
    currentConfigMapName.value = row.name
    usageVisible.value = true
  }

  const handleDownloadYaml = async (row: ConfigMapListItem) => {
    downloadLoadingMap.value[row.name] = true
    try {
      const yaml = await getConfigMapYamlApi({
        workloadId: workloadId.value,
        name: row.name
      })

      const blob = new Blob([yaml], { type: 'text/yaml' })
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `${row.name}.yaml`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)

      ElMessage.success('下载成功')
    } finally {
      downloadLoadingMap.value[row.name] = false
    }
  }

  const handleDelete = async (row: ConfigMapListItem) => {
    try {
      const usage = await getConfigMapUsageApi({
        workloadId: workloadId.value,
        name: row.name
      })

      let confirmMessage = `确定要删除 ConfigMap "${row.name}" 吗？此操作不可恢复！`
      if (!usage.canDelete && usage.deleteWarning) {
        confirmMessage = `${usage.deleteWarning}\n\n${confirmMessage}`
      }

      await ElMessageBox.confirm(confirmMessage, '删除确认', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
        dangerouslyUseHTMLString: true
      })

      deleteLoadingMap.value[row.name] = true
      await deleteConfigMapApi({
        workloadId: workloadId.value,
        name: row.name
      })

      ElMessage.success('删除成功')
      loadData()
    } catch (error: any) {
      // 用户取消操作不做处理
      if (error === 'cancel') return
    } finally {
      deleteLoadingMap.value[row.name] = false
    }
  }

  const { columns } = createTableColumns({
    handleView,
    handleEdit,
    handleViewUsage,
    handleDownloadYaml,
    handleDelete
  })

  /**
   * 监听 refreshTrigger 变化
   * 只有当 trigger 值真正变化（增加）时才触发刷新，避免与 onMounted 重复
   */
  watch(
    () => props.refreshTrigger,
    (newTrigger) => {
      // 只有当值真正增加，且已完成初始加载时才触发刷新
      if (
        newTrigger !== undefined &&
        newTrigger > lastRefreshTrigger.value &&
        hasInitialLoaded.value
      ) {
        lastRefreshTrigger.value = newTrigger
        if (props.workspace) {
          loadData()
        }
      }
    }
  )

  // 监听工作空间变化，重置状态并重新加载
  watch(
    () => props.workspace,
    (newWorkspace, oldWorkspace) => {
      if (newWorkspace && newWorkspace.id !== oldWorkspace?.id) {
        hasInitialLoaded.value = false
        resetSearchParams()
      }
    }
  )

  const loadData = async () => {
    if (!workloadId.value) {
      return
    }

    loading.value = true
    try {
      const params = {
        workloadId: workloadId.value,
        ...searchParams.value
      }

      const response = await getConfigMapListApi(params)

      data.value = response.items || []
      pagination.value.total = response.total || 0
    } finally {
      loading.value = false
    }
  }

  const handleSearch = (params: Record<string, any>) => {
    searchParams.value = params
    pagination.value.current = 1
    loadData()
  }

  const resetSearchParams = () => {
    searchParams.value = {}
    pagination.value.current = 1
    loadData()
  }

  const handleSizeChange = (size: number) => {
    pagination.value.size = size
    pagination.value.current = 1
    loadData()
  }

  const handleCurrentChange = (current: number) => {
    pagination.value.current = current
    loadData()
  }

  const handleCreate = () => {
    if (!props.workspace) {
      return
    }
    dialogType.value = 'add'
    currentConfigMap.value = undefined
    dialogVisible.value = true
  }

  /**
   * 组件挂载时加载数据
   * 这是首次加载的唯一入口，避免与 watch(refreshTrigger) 重复
   */
  onMounted(() => {
    if (props.workspace) {
      // 记录当前的 trigger 值，后续 watch 只响应比这个值大的变化
      lastRefreshTrigger.value = props.refreshTrigger || 0
      loadData().then(() => {
        hasInitialLoaded.value = true
      })
    }
  })
</script>

<style lang="scss" scoped>
  .configmap-page {
    padding-bottom: 20px;
  }
</style>
