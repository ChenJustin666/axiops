<template>
  <div class="ingress-page art-full-height">
    <!-- 搜索栏 -->
    <IngressSearch
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
            <ElButton type="primary" @click="handleCreate">新增 Ingress</ElButton>
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
    <IngressDialog
      v-if="dialogVisible"
      :visible="dialogVisible"
      :dialog-type="dialogType"
      :ingress-data="currentIngress"
      :workspace="workspace"
      @close="handleDialogClose"
      @success="handleSuccess"
    />

    <!-- 查看详情对话框 -->
    <IngressDetailDialog
      v-if="detailVisible"
      :visible="detailVisible"
      :ingress-name="currentIngressName"
      :workspace="workspace"
      @close="detailVisible = false"
    />

    <!-- Host 监控对话框 -->
    <HostMetricsMonitor
      :visible="monitorVisible"
      :cluster-uuid="workspace?.clusterUuid || ''"
      :host="currentHost"
      @close="handleCloseMonitor"
    />
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, watch, onMounted } from 'vue'
  import { useRouter } from 'vue-router'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import {
    getIngressListApi,
    deleteIngressApi,
    getIngressYamlApi,
    type IngressListItem,
    type ProjectWorkspace
  } from '@/api'
  import IngressSearch from './components/IngressSearch.vue'
  import IngressDialog from './components/IngressDialog.vue'
  import IngressDetailDialog from './components/IngressDetailDialog.vue'
  import HostMetricsMonitor from './components/HostMetricsMonitor.vue'
  import { useIngressTableConfig } from './composables/table-config'

  defineOptions({ name: 'IngressManagement' })

  interface Props {
    workspace?: ProjectWorkspace
    refreshTrigger?: number
  }

  const props = defineProps<Props>()

  const router = useRouter()

  const { loading, showSearchBar, deleteLoadingMap, downloadLoadingMap, createTableColumns } =
    useIngressTableConfig()

  const allData = ref<IngressListItem[]>([])
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
  const monitorVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const currentIngress = ref<IngressListItem | undefined>(undefined)
  const currentIngressName = ref('')
  const currentHost = ref('')

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

  const handleMonitor = (row: IngressListItem) => {
    if (!props.workspace) {
      ElMessage.warning('工作空间信息缺失')
      return
    }

    // 如果有 hosts，使用第一个 host；否则使用 ingressName
    const host = row.hosts && row.hosts.length > 0 ? row.hosts[0] : row.name
    // 打开 Host 监控弹窗
    currentHost.value = host
    monitorVisible.value = true

    // 添加调试日志
    console.log('打开监控弹窗', {
      host,
      clusterUuid: props.workspace?.clusterUuid,
      monitorVisible: monitorVisible.value
    })
  }

  const handleCloseMonitor = () => {
    monitorVisible.value = false
    console.log('关闭监控弹窗')
  }

  const handleView = (row: IngressListItem) => {
    currentIngressName.value = row.name
    detailVisible.value = true
  }

  const handleEdit = (row: IngressListItem) => {
    dialogType.value = 'edit'
    currentIngress.value = row
    dialogVisible.value = true
  }

  const handleDownloadYaml = async (row: IngressListItem) => {
    try {
      downloadLoadingMap.value[row.name] = true
      const yaml = await getIngressYamlApi({
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

  const handleDelete = async (row: IngressListItem) => {
    try {
      await ElMessageBox.confirm(
        `确定要删除 Ingress "${row.name}" 吗？此操作不可恢复！`,
        '删除确认',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      deleteLoadingMap.value[row.name] = true
      await deleteIngressApi({
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
    handleMonitor,
    handleView,
    handleEdit,
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

      const response = await getIngressListApi(params)

      let items: IngressListItem[] = []

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
    } catch (error: any) {
      console.error('加载Ingress列表失败:', error)
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
    currentIngress.value = undefined
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
  .ingress-page {
    padding-bottom: 20px;
  }
</style>
