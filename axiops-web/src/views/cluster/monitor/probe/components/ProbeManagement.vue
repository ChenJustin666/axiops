<template>
  <div class="probe-management art-full-height">
    <!-- 搜索栏 -->
    <ProbeSearch
      v-show="showSearchBar"
      v-model="searchForm"
      :cluster-uuid="clusterUuid"
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
            <ElButton type="primary" @click="handleCreate">新增 Probe</ElButton>
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
    <ProbeDialog
      v-if="dialogVisible"
      :visible="dialogVisible"
      :dialog-type="dialogType"
      :probe-data="currentProbe"
      :cluster-uuid="clusterUuid"
      @close="handleDialogClose"
      @success="handleSuccess"
    />

    <!-- 查看详情对话框 -->
    <ProbeDetailDialog
      v-if="detailVisible"
      :visible="detailVisible"
      :probe-name="currentProbeName"
      :namespace="currentNamespace"
      :cluster-uuid="clusterUuid"
      @close="detailVisible = false"
    />
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, watch, onMounted } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import {
    getProbeListApi,
    deleteProbeApi,
    getProbeYamlApi,
    type ProbeListItem
  } from '@/api/workload/monitor'
  import ProbeSearch from './ProbeSearch.vue'
  import ProbeDialog from './ProbeDialog.vue'
  import ProbeDetailDialog from './ProbeDetailDialog.vue'
  import { useProbeTableConfig } from '../composables/table-config'

  defineOptions({ name: 'ProbeManagement' })

  interface Props {
    clusterUuid: string
    clusterId: number
    refreshTrigger?: number
  }

  const props = defineProps<Props>()
  const emit = defineEmits(['loaded'])

  const { loading, showSearchBar, deleteLoadingMap, downloadLoadingMap, createTableColumns } =
    useProbeTableConfig()

  const allData = ref<ProbeListItem[]>([])
  const useServerPagination = ref(true)

  const searchForm = ref({
    namespace: 'monitoring',
    search: ''
  })

  const searchParams = ref<any>({})

  const pagination = ref({
    current: 1,
    size: 20,
    total: 0
  })

  const dialogVisible = ref(false)
  const detailVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const currentProbe = ref<ProbeListItem | undefined>(undefined)
  const currentProbeName = ref('')
  const currentNamespace = ref('')

  const clusterUuid = computed(() => props.clusterUuid || '')

  const displayData = computed(() => {
    if (useServerPagination.value) {
      return allData.value
    } else {
      const start = (pagination.value.current - 1) * pagination.value.size
      const end = start + pagination.value.size
      return allData.value.slice(start, end)
    }
  })

  const handleView = (row: ProbeListItem) => {
    currentProbeName.value = row.name
    currentNamespace.value = row.namespace
    detailVisible.value = true
  }

  const handleEdit = (row: ProbeListItem) => {
    dialogType.value = 'edit'
    currentProbe.value = row
    dialogVisible.value = true
  }

  const handleDownloadYaml = async (row: ProbeListItem) => {
    try {
      downloadLoadingMap.value[row.name] = true
      const yaml = await getProbeYamlApi({
        clusterUuid: clusterUuid.value,
        namespace: row.namespace,
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

  const handleDelete = async (row: ProbeListItem) => {
    try {
      await ElMessageBox.confirm(
        `确定要删除 Probe "${row.name}" 吗？此操作不可恢复！`,
        '删除确认',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      deleteLoadingMap.value[row.name] = true
      await deleteProbeApi({
        clusterUuid: clusterUuid.value,
        namespace: row.namespace,
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

  const handleMonitor = (row: ProbeListItem) => {
    ElMessage.info('监控功能正在开发中...')
    // TODO: 跳转到监控页面或打开监控对话框
    // 可以跳转到 Grafana 或其他监控平台
    // 例如: window.open(`/monitoring/probe/${row.namespace}/${row.name}`)
  }

  const { columns } = createTableColumns({
    handleView,
    handleEdit,
    handleDownloadYaml,
    handleDelete,
    handleMonitor
  })

  watch(
    () => props.refreshTrigger,
    () => {
      if (props.clusterUuid) {
        resetSearchParams()
        loadData()
      }
    }
  )

  watch(
    () => props.clusterUuid,
    (newUuid) => {
      if (newUuid) {
        resetSearchParams()
        loadData()
      }
    }
  )

  const loadData = async () => {
    if (!clusterUuid.value) {
      return
    }

    loading.value = true
    try {
      const params = {
        clusterUuid: clusterUuid.value,
        namespace: searchParams.value.namespace || searchForm.value.namespace,
        ...searchParams.value
      }

      console.log('🚀 [Probe] 加载数据，参数:', params)

      const response = await getProbeListApi(params)

      let items: ProbeListItem[] = []

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

      console.log('✅ [Probe] 加载成功，共', items.length, '条数据')
      emit('loaded')
    } catch (error: any) {
      console.error('❌ [Probe] 加载列表失败:', error)
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
    searchForm.value = {
      namespace: '',
      search: ''
    }
    pagination.value.current = 1
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
    if (!props.clusterUuid) {
      ElMessage.warning('请先选择集群')
      return
    }
    dialogType.value = 'add'
    currentProbe.value = undefined
    dialogVisible.value = true
  }

  const handleDialogClose = () => {
    dialogVisible.value = false
  }

  const handleSuccess = () => {
    dialogVisible.value = false
    loadData()
  }

  onMounted(() => {
    console.log('🔧 [Probe] 组件挂载, clusterUuid:', props.clusterUuid)
    if (props.clusterUuid) {
      loadData()
    }
  })
</script>

<style lang="scss" scoped>
  .probe-management {
    padding-bottom: 20px;
  }
</style>
