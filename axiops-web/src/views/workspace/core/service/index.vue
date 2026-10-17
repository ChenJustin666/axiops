<template>
  <div class="service-page art-full-height">
    <!-- 搜索栏 -->
    <ServiceSearch
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
        :data="displayData"
        :columns="columns"
        :pagination="pagination"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      />
    </ElCard>

    <!-- 创建/编辑对话框 -->
    <ServiceDialog
      v-if="dialogVisible"
      :visible="dialogVisible"
      :dialog-type="dialogType"
      :service-data="currentService"
      :workspace="workspace"
      @close="handleDialogClose"
      @success="handleSuccess"
    />

    <!-- 查看详情对话框 -->
    <ServiceDetailDialog
      v-if="detailVisible"
      :visible="detailVisible"
      :service-name="currentServiceName"
      :workspace="workspace"
      @close="detailVisible = false"
    />

    <!-- Endpoints 对话框 -->
    <ServiceEndpointsDialog
      v-if="endpointsVisible"
      :visible="endpointsVisible"
      :service-name="currentServiceName"
      :workspace="workspace"
      @close="endpointsVisible = false"
    />
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, watch, onMounted } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import {
    getServiceListApi,
    deleteServiceApi,
    getServiceYamlApi,
    type ServiceListItem,
    type ProjectWorkspace
  } from '@/api'
  import ServiceSearch from './components/ServiceSearch.vue'
  import ServiceDialog from './components/ServiceDialog.vue'
  import ServiceDetailDialog from './components/ServiceDetailDialog.vue'
  import ServiceEndpointsDialog from './components/ServiceEndpointsDialog.vue'
  import { useServiceTableConfig } from './composables/table-config'

  defineOptions({ name: 'ServiceManagement' })

  interface Props {
    workspace?: ProjectWorkspace
    refreshTrigger?: number
  }

  const props = defineProps<Props>()
  const emit = defineEmits(['loaded'])

  const { loading, showSearchBar, deleteLoadingMap, downloadLoadingMap, createTableColumns } =
    useServiceTableConfig()

  const allData = ref<ServiceListItem[]>([])
  const useServerPagination = ref(true)

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
  const endpointsVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const currentService = ref<ServiceListItem | undefined>(undefined)
  const currentServiceName = ref('')

  const workloadId = computed(() => props.workspace?.id || 0)

  // 用于跟踪是否已完成初始加载，避免重复请求
  const hasInitialLoaded = ref(false)
  // 记录上一次的 refreshTrigger 值，用于判断是否是真正的刷新
  const lastRefreshTrigger = ref(props.refreshTrigger || 0)

  const displayData = computed(() => {
    if (useServerPagination.value) {
      return allData.value
    } else {
      const start = (pagination.value.current - 1) * pagination.value.size
      const end = start + pagination.value.size
      return allData.value.slice(start, end)
    }
  })

  const handleView = (row: ServiceListItem) => {
    currentServiceName.value = row.name
    detailVisible.value = true
  }

  const handleEdit = (row: ServiceListItem) => {
    dialogType.value = 'edit'
    currentService.value = row
    dialogVisible.value = true
  }

  const handleEndpoints = (row: ServiceListItem) => {
    currentServiceName.value = row.name
    endpointsVisible.value = true
  }

  const handleDownloadYaml = async (row: ServiceListItem) => {
    try {
      downloadLoadingMap.value[row.name] = true
      const yaml = await getServiceYamlApi({
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
    } catch (error) {
      console.error('下载YAML失败:', error)
    } finally {
      downloadLoadingMap.value[row.name] = false
    }
  }

  const handleDelete = async (row: ServiceListItem) => {
    try {
      await ElMessageBox.confirm(
        `确定要删除 Service "${row.name}" 吗？此操作不可恢复！`,
        '删除确认',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      deleteLoadingMap.value[row.name] = true
      await deleteServiceApi({
        workloadId: workloadId.value,
        name: row.name
      })

      ElMessage.success('删除成功')
      loadData()
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('删除失败:', error)
      }
    } finally {
      deleteLoadingMap.value[row.name] = false
    }
  }

  const { columns } = createTableColumns({
    handleView,
    handleEdit,
    handleViewEndpoints: handleEndpoints,
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

      const response = await getServiceListApi(params)

      let items: ServiceListItem[] = []

      if (Array.isArray(response)) {
        items = response
        useServerPagination.value = false
      } else if (response && response.items) {
        items = response.items
        if (response.total !== undefined && response.total !== null) {
          useServerPagination.value = true
          pagination.value.total = response.total
        } else {
          useServerPagination.value = false
        }
      } else {
        items = []
      }

      allData.value = items

      if (!useServerPagination.value) {
        pagination.value.total = items.length
      }

      emit('loaded')
    } catch (error: any) {
      console.error('加载Service列表失败:', error)
      allData.value = []
      pagination.value.total = 0
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

    if (useServerPagination.value) {
      loadData()
    }
  }

  const handleCurrentChange = (current: number) => {
    pagination.value.current = current

    if (useServerPagination.value) {
      loadData()
    }
  }

  const handleCreate = () => {
    if (!props.workspace) {
      return
    }
    dialogType.value = 'add'
    currentService.value = undefined
    dialogVisible.value = true
  }

  const handleDialogClose = () => {
    dialogVisible.value = false
  }

  const handleSuccess = () => {
    dialogVisible.value = false
    loadData()
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
  .service-page {
    padding-bottom: 20px;
  }
</style>
