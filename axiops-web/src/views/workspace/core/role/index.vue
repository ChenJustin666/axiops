<template>
  <div class="role-page art-full-height">
    <!-- 搜索栏 -->
    <RoleSearch
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
            <ElButton type="primary" @click="handleCreate">
              <ShieldCheck :size="16" style="margin-right: 4px" />
              新增 Role
            </ElButton>
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
    <RoleDialog
      v-if="dialogVisible"
      :visible="dialogVisible"
      :dialog-type="dialogType"
      :role-data="currentRole"
      :workspace="workspace"
      @close="handleDialogClose"
      @success="handleSuccess"
    />

    <!-- 查看详情对话框 -->
    <RoleDetailDialog
      v-if="detailVisible"
      :visible="detailVisible"
      :role-name="currentRoleName"
      :workspace="workspace"
      @close="detailVisible = false"
    />

    <!-- 关联信息对话框 -->
    <RoleAssociationDialog
      v-if="associationVisible"
      :visible="associationVisible"
      :role-name="currentRoleName"
      :workspace="workspace"
      @close="associationVisible = false"
    />
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, watch, onMounted } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { ShieldCheck } from 'lucide-vue-next'
  import {
    getRoleListApi,
    deleteRoleApi,
    getRoleYamlApi,
    type RoleListItem,
    type ProjectWorkspace
  } from '@/api/workload/core'
  import RoleSearch from './components/RoleSearch.vue'
  import RoleDialog from './components/RoleDialog.vue'
  import RoleDetailDialog from './components/RoleDetailDialog.vue'
  import RoleAssociationDialog from './components/RoleAssociationDialog.vue'
  import { useRoleTableConfig } from './composables/table-config'

  defineOptions({ name: 'RoleManagement' })

  interface Props {
    workspace?: ProjectWorkspace
    refreshTrigger?: number
  }

  const props = defineProps<Props>()

  const { loading, showSearchBar, deleteLoadingMap, downloadLoadingMap, createTableColumns } =
    useRoleTableConfig()

  const allData = ref<RoleListItem[]>([])
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
  const associationVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const currentRole = ref<RoleListItem | undefined>(undefined)
  const currentRoleName = ref('')

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

  const handleView = (row: RoleListItem) => {
    currentRoleName.value = row.name
    detailVisible.value = true
  }

  const handleViewAssociation = (row: RoleListItem) => {
    currentRoleName.value = row.name
    associationVisible.value = true
  }

  const handleEdit = (row: RoleListItem) => {
    dialogType.value = 'edit'
    currentRole.value = row
    dialogVisible.value = true
  }

  const handleDownloadYaml = async (row: RoleListItem) => {
    downloadLoadingMap.value[row.name] = true
    try {
      const yaml = await getRoleYamlApi({
        clusterUuid: props.workspace!.clusterUuid!,
        namespace: props.workspace!.namespace!,
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

  const handleDelete = async (row: RoleListItem) => {
    try {
      await ElMessageBox.confirm(`确定要删除 Role "${row.name}" 吗？此操作不可恢复！`, '删除确认', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      })

      deleteLoadingMap.value[row.name] = true
      await deleteRoleApi({
        clusterUuid: props.workspace!.clusterUuid!,
        namespace: props.workspace!.namespace!,
        name: row.name
      })

      ElMessage.success('删除成功')
      loadData()
    } catch (error: any) {
      if (error === 'cancel') return
    } finally {
      deleteLoadingMap.value[row.name] = false
    }
  }

  const { columns } = createTableColumns({
    handleView,
    handleViewAssociation,
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
    if (!props.workspace?.clusterUuid || !props.workspace?.namespace) {
      return
    }

    loading.value = true
    try {
      const params = {
        clusterUuid: props.workspace.clusterUuid,
        namespace: props.workspace.namespace,
        ...searchParams.value
      }

      const response = await getRoleListApi(params)

      let items: RoleListItem[] = []

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
      console.error('加载Role列表失败:', error)
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
    currentRole.value = undefined
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
  .role-page {
    padding-bottom: 20px;
  }
</style>
