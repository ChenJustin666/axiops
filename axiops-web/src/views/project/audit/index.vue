<template>
  <div class="audit-log-page art-full-height">
    <!-- 未选择项目时显示引导提示 -->
    <template v-if="!selectedProject">
      <ElCard shadow="never" class="empty-project-card">
        <SelectionGuideState
          level="project"
          :project="selectedProject"
          resource-name="审计日志"
          :show-progress="false"
        />
      </ElCard>
    </template>

    <!-- 已选择项目 -->
    <template v-else>
      <!-- 搜索栏 -->
      <AuditSearch
        v-show="showSearchBar"
        v-model="searchForm"
        :project-id="selectedProject.id"
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
          @refresh="refreshData"
        >
          <template #left>
            <ElSpace wrap>
              <ElButton
                type="danger"
                :disabled="selectedIds.length === 0 || batchDeleting"
                :loading="batchDeleting"
                @click="handleBatchDelete"
              >
                {{ batchDeleting ? '批量删除中...' : `批量删除 (${selectedIds.length})` }}
              </ElButton>
            </ElSpace>
          </template>
        </ArtTableHeader>

        <!-- 数据表格 -->
        <ArtTable
          :loading="loading"
          :data="data"
          :columns="columns"
          :pagination="pagination"
          :show-selection="true"
          @selection-change="handleSelectionChange"
          @pagination:size-change="handleSizeChange"
          @pagination:current-change="handleCurrentChange"
        />
      </ElCard>

      <!-- 详情弹窗 -->
      <ElDialog
        v-model="detailDialogVisible"
        title="审计日志详情"
        width="800px"
        align-center
        :close-on-click-modal="false"
      >
        <div class="detail-content" v-if="currentAuditLog">
          <!-- 顶部状态栏 -->
          <div class="detail-status-bar">
            <div class="status-left">
              <span class="log-id">ID: {{ currentAuditLog.id }}</span>
            </div>
            <ElTag :type="getAuditStatusInfo(currentAuditLog.status).type" size="default">
              {{ getAuditStatusInfo(currentAuditLog.status).label }}
            </ElTag>
          </div>

          <!-- 操作信息 -->
          <div class="detail-section">
            <div class="section-title">
              <Activity :size="18" />
              <span>操作信息</span>
            </div>
            <div class="section-body">
              <div class="info-row">
                <div class="info-item">
                  <label>操作简称</label>
                  <div class="info-value highlight">{{ currentAuditLog.title }}</div>
                </div>
              </div>
              <div class="info-row">
                <div class="info-item full-width">
                  <label>操作详情</label>
                  <div class="info-value detail-text">
                    {{ currentAuditLog.actionDetail }}
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 资源信息 -->
          <div class="detail-section">
            <div class="section-title">
              <Server :size="18" />
              <span>资源信息</span>
            </div>
            <div class="section-body">
              <div class="info-row">
                <div class="info-item">
                  <label>
                    <Database :size="14" />
                    集群名称
                  </label>
                  <div class="info-value">{{ currentAuditLog.clusterName }}</div>
                </div>
                <div class="info-item">
                  <label>
                    <Hash :size="14" />
                    集群 UUID
                  </label>
                  <div class="info-value code">{{ currentAuditLog.clusterUuid }}</div>
                </div>
              </div>
              <div class="info-row">
                <div class="info-item">
                  <label>
                    <FolderKanban :size="14" />
                    项目名称
                  </label>
                  <div class="info-value">{{ currentAuditLog.projectName }}</div>
                </div>
                <div class="info-item">
                  <label>
                    <Hash :size="14" />
                    项目 ID
                  </label>
                  <div class="info-value code">{{ currentAuditLog.projectId }}</div>
                </div>
              </div>
              <div class="info-row">
                <div class="info-item">
                  <label>
                    <Layers :size="14" />
                    工作空间
                  </label>
                  <div class="info-value">
                    {{
                      getWorkspaceDisplayName(
                        currentAuditLog.workspaceName,
                        currentAuditLog.workspaceId
                      )
                    }}
                    <span class="info-badge">ID: {{ currentAuditLog.workspaceId }}</span>
                  </div>
                </div>
                <div class="info-item">
                  <label>
                    <Box :size="14" />
                    应用名称
                  </label>
                  <div class="info-value">
                    <template v-if="currentAuditLog.applicationName">
                      {{ currentAuditLog.applicationName }}
                      <span class="info-badge">ID: {{ currentAuditLog.applicationId }}</span>
                    </template>
                    <span v-else class="empty-value">-</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 操作人信息 -->
          <div class="detail-section">
            <div class="section-title">
              <User :size="18" />
              <span>操作人信息</span>
            </div>
            <div class="section-body">
              <div class="info-row">
                <div class="info-item">
                  <label>
                    <UserCircle :size="14" />
                    操作人
                  </label>
                  <div class="info-value">
                    <span class="operator-name">{{ currentAuditLog.operatorName }}</span>
                  </div>
                </div>
                <div class="info-item">
                  <label>
                    <Hash :size="14" />
                    操作人 ID
                  </label>
                  <div class="info-value code">{{ currentAuditLog.operatorId }}</div>
                </div>
              </div>
            </div>
          </div>

          <!-- 时间信息 -->
          <div class="detail-section">
            <div class="section-title">
              <Clock :size="18" />
              <span>时间信息</span>
            </div>
            <div class="section-body">
              <div class="info-row">
                <div class="info-item">
                  <label>
                    <Calendar :size="14" />
                    操作时间
                  </label>
                  <div class="info-value time">
                    {{ formatTimestamp(currentAuditLog.createdAt) }}
                  </div>
                </div>
                <div class="info-item">
                  <label>
                    <RefreshCw :size="14" />
                    更新时间
                  </label>
                  <div class="info-value time">
                    {{ formatTimestamp(currentAuditLog.updatedAt) }}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <template #footer>
          <ElButton @click="detailDialogVisible = false">关闭</ElButton>
        </template>
      </ElDialog>
    </template>
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, onMounted, watch } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import {
    Activity,
    Server,
    Database,
    Hash,
    FolderKanban,
    Layers,
    Box,
    User,
    UserCircle,
    Clock,
    Calendar,
    RefreshCw
  } from 'lucide-vue-next'
  import { useProjectStore } from '@/store/modules/project'
  import {
    searchProjectAuditLogApi,
    deleteProjectAuditLogApi,
    batchDeleteProjectAuditLogApi,
    getAuditStatusInfo,
    getWorkspaceDisplayName,
    type ProjectAuditLog
  } from '@/api/manager/audit'
  import { formatTimestamp } from '@/utils/format'
  import AuditSearch from './modules/audit-search.vue'
  import { useAuditLogTableConfig } from './modules/table-config'
  import SelectionGuideState from '@/components/SelectionGuideState/index.vue'

  defineOptions({ name: 'ProjectAuditLog' })

  // 获取当前选中的项目
  const projectStore = useProjectStore()
  const selectedProject = computed(() => projectStore.selectedProject)

  // 使用抽离的表格配置逻辑
  const { loading, showSearchBar, deleteLoadingMap, batchDeleting, createTableColumns } =
    useAuditLogTableConfig()

  // 表格数据
  const data = ref<ProjectAuditLog[]>([])

  // 已选择的行
  const selectedIds = ref<number[]>([])

  // 搜索表单
  const searchForm = ref({
    clusterUuid: undefined,
    workspaceId: -1,
    applicationId: -1,
    title: undefined,
    operatorName: undefined,
    status: -1,
    timeRange: undefined
  })

  // 搜索参数
  const searchParams = ref<any>({})

  // 分页配置
  const pagination = ref({
    current: 1,
    size: 20,
    total: 0
  })

  // 详情弹窗
  const detailDialogVisible = ref(false)
  const currentAuditLog = ref<ProjectAuditLog | null>(null)

  // 查看详情
  const handleViewDetail = (row: ProjectAuditLog) => {
    currentAuditLog.value = row
    detailDialogVisible.value = true
  }

  // 删除单条记录
  const handleDelete = async (row: ProjectAuditLog) => {
    try {
      deleteLoadingMap.value[row.id] = true
      await deleteProjectAuditLogApi(row.id)
      ElMessage.success('删除成功')
      await getData()
    } catch (error) {
    } finally {
      deleteLoadingMap.value[row.id] = false
    }
  }

  // 批量删除
  const handleBatchDelete = async () => {
    if (selectedIds.value.length === 0) {
      ElMessage.warning('请先选择要删除的记录')
      return
    }

    if (selectedIds.value.length > 100) {
      ElMessage.warning('一次最多只能删除100条记录')
      return
    }

    try {
      await ElMessageBox.confirm(
        `确定要删除选中的 ${selectedIds.value.length} 条审计日志吗?此操作不可恢复!`,
        '批量删除确认',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      batchDeleting.value = true
      await batchDeleteProjectAuditLogApi({ ids: selectedIds.value })
      ElMessage.success('批量删除成功')
      selectedIds.value = []
      await getData()
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('批量删除失败:', error)
      }
    } finally {
      batchDeleting.value = false
    }
  }

  // 处理表格选择变化
  const handleSelectionChange = (selection: ProjectAuditLog[]) => {
    selectedIds.value = selection.map((item) => item.id)
  }

  // 创建表格列配置
  const { columns } = createTableColumns({
    handleDelete,
    handleViewDetail
  })

  // 获取数据
  const getData = async () => {
    if (!selectedProject.value) {
      data.value = []
      pagination.value.total = 0
      return
    }

    try {
      loading.value = true
      const params = {
        page: pagination.value.current,
        pageSize: pagination.value.size,
        orderField: 'id',
        isAsc: false,
        ...searchParams.value
      }

      const response = await searchProjectAuditLogApi(params)
      data.value = response.items || []
      pagination.value.total = response.total || 0
    } finally {
      loading.value = false
    }
  }

  // 刷新数据
  const refreshData = () => {
    getData()
  }

  // 搜索处理
  const handleSearch = (params: Record<string, any>) => {
    if (!params.projectId) {
      ElMessage.warning('请先选择项目')
      return
    }

    searchParams.value = params
    pagination.value.current = 1
    getData()
  }

  // 重置搜索参数
  const resetSearchParams = () => {
    if (!selectedProject.value) {
      searchParams.value = {}
    } else {
      searchParams.value = {
        projectId: selectedProject.value.id,
        workspaceId: -1,
        applicationId: -1,
        status: -1
      }
    }
    pagination.value.current = 1
    getData()
  }

  // 分页大小改变
  const handleSizeChange = (size: number) => {
    pagination.value.size = size
    pagination.value.current = 1
    getData()
  }

  // 当前页改变
  const handleCurrentChange = (current: number) => {
    pagination.value.current = current
    getData()
  }

  // 监听项目切换
  watch(
    selectedProject,
    (newProject, oldProject) => {
      if (newProject?.id !== oldProject?.id) {
        if (newProject) {
          searchParams.value = {
            projectId: newProject.id,
            workspaceId: -1,
            applicationId: -1,
            status: -1
          }
        } else {
          searchParams.value = {}
        }

        pagination.value.current = 1
        selectedIds.value = []

        if (newProject) {
          getData()
        } else {
          data.value = []
          pagination.value.total = 0
        }
      }
    },
    { immediate: false }
  )

  // 初始化
  onMounted(() => {
    if (selectedProject.value) {
      searchParams.value = {
        projectId: selectedProject.value.id,
        workspaceId: -1,
        applicationId: -1,
        status: -1
      }
      getData()
    }
  })
</script>

<style lang="scss" scoped>
  .audit-log-page {
    padding-bottom: 15px;
  }

  .empty-project-card {
    height: calc(100vh - 200px);
    display: flex;
    align-items: center;
    justify-content: center;

    :deep(.el-card__body) {
      width: 100%;
      display: flex;
      align-items: center;
      justify-content: center;
    }
  }

  // 详情内容区域
  .detail-content {
    max-height: 65vh;
    overflow-y: auto;

    &::-webkit-scrollbar {
      width: 6px;
    }

    &::-webkit-scrollbar-thumb {
      background: #dcdfe6;
      border-radius: 3px;
    }
  }

  // 顶部状态栏
  .detail-status-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    margin-bottom: 20px;
    background: #f8fafc;
    border-radius: 8px;
    border: 1px solid #e4e7ed;

    .status-left {
      .log-id {
        font-size: 13px;
        color: #64748b;
        font-family: var(--ax-font-mono);
      }
    }
  }

  // 详情分组
  .detail-section {
    margin-bottom: 20px;
    background: #fff;
    border-radius: 8px;
    overflow: hidden;
    border: 1px solid #e4e7ed;

    &:last-child {
      margin-bottom: 0;
    }

    .section-title {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 12px 16px;
      background: #f8fafc;
      border-bottom: 1px solid #e4e7ed;
      font-size: 14px;
      font-weight: 600;
      color: #303133;

      svg {
        color: #409eff;
      }
    }

    .section-body {
      padding: 16px;
    }
  }

  // 信息行
  .info-row {
    display: flex;
    gap: 20px;
    margin-bottom: 14px;

    &:last-child {
      margin-bottom: 0;
    }
  }

  // 信息项
  .info-item {
    flex: 1;
    min-width: 0;

    &.full-width {
      flex: 0 0 100%;
    }

    label {
      display: flex;
      align-items: center;
      gap: 6px;
      font-size: 12px;
      color: #909399;
      margin-bottom: 6px;

      svg {
        color: #c0c4cc;
      }
    }

    .info-value {
      font-size: 14px;
      color: #303133;
      line-height: 1.5;
      word-break: break-all;

      &.highlight {
        font-size: 15px;
        font-weight: 500;
        color: #409eff;
        padding: 10px 12px;
        background: #ecf5ff;
        border-radius: 6px;
        border-left: 3px solid #409eff;
      }

      &.detail-text {
        padding: 12px;
        background: #f5f7fa;
        border-radius: 6px;
        white-space: pre-wrap;
        max-height: 180px;
        overflow-y: auto;
        line-height: 1.7;
        font-size: 13px;

        &::-webkit-scrollbar {
          width: 4px;
        }

        &::-webkit-scrollbar-thumb {
          background: #dcdfe6;
          border-radius: 2px;
        }
      }

      &.code {
        font-family: var(--ax-font-mono);
        font-size: 13px;
        color: #606266;
        background: #f5f7fa;
        padding: 4px 8px;
        border-radius: 4px;
        display: inline-block;
      }

      &.time {
        font-family: var(--ax-font-mono);
        font-size: 13px;
        color: #67c23a;
      }

      .info-badge {
        display: inline-block;
        margin-left: 8px;
        padding: 2px 6px;
        background: #f0f2f5;
        color: #606266;
        border-radius: 4px;
        font-size: 12px;
        font-family: var(--ax-font-mono);
      }

      .operator-name {
        font-weight: 500;
        color: #303133;
      }

      .empty-value {
        color: #c0c4cc;
      }
    }
  }
</style>
