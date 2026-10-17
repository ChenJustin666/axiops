<template>
  <div class="billing-statement-page">
    <!-- 搜索栏 -->
    <StatementSearch v-model="searchForm" @search="handleSearch" @reset="resetSearchParams" />

    <!-- 统计卡片 - 优化为更紧凑的单行布局 -->
    <div class="stats-cards">
      <div class="stat-card">
        <div class="stat-icon total-icon">
          <el-icon><Document /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">账单总数</div>
          <div class="stat-value">{{ formatNumber(summary.totalCount) }}</div>
        </div>
      </div>
      <div class="stat-card highlight-purple">
        <div class="stat-icon purple-icon">
          <el-icon><Money /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">总费用</div>
          <div class="stat-value">{{ formatCurrency(summary.totalAmount) }}</div>
        </div>
      </div>
      <div class="stat-card highlight-blue">
        <div class="stat-icon blue-icon">
          <el-icon><Cpu /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">CPU费用</div>
          <div class="stat-value">{{ formatCurrency(summary.cpuCost) }}</div>
        </div>
      </div>
      <div class="stat-card highlight-green">
        <div class="stat-icon green-icon">
          <el-icon><Histogram /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">内存费用</div>
          <div class="stat-value">{{ formatCurrency(summary.memoryCost) }}</div>
        </div>
      </div>
      <div class="stat-card highlight-orange">
        <div class="stat-icon orange-icon">
          <el-icon><Files /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">存储费用</div>
          <div class="stat-value">{{ formatCurrency(summary.storageCost) }}</div>
        </div>
      </div>
      <div class="stat-card highlight-violet">
        <div class="stat-icon violet-icon">
          <el-icon><VideoCamera /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">GPU费用</div>
          <div class="stat-value">{{ formatCurrency(summary.gpuCost) }}</div>
        </div>
      </div>
      <div class="stat-card highlight-cyan">
        <div class="stat-icon cyan-icon">
          <el-icon><Box /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">Pod费用</div>
          <div class="stat-value">{{ formatCurrency(summary.podCost) }}</div>
        </div>
      </div>
      <div class="stat-card highlight-pink">
        <div class="stat-icon pink-icon">
          <el-icon><Management /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">管理费</div>
          <div class="stat-value">{{ formatCurrency(summary.managementFee) }}</div>
        </div>
      </div>
    </div>

    <!-- 表格卡片 - 使用flex:1撑满剩余空间 -->
    <ElCard class="table-card" shadow="never">
      <!-- 表格头部 -->
      <div class="table-header">
        <div class="table-title">
          <span class="title-text">账单明细</span>
          <span class="title-badge">{{ formatNumber(total) }} 条</span>
        </div>
        <div class="table-actions">
          <ElButton type="primary" plain :loading="exportLoading" @click="handleExport">
            <ElIcon v-if="!exportLoading" class="mr-1"><Download /></ElIcon>
            {{ exportLoading ? '导出中...' : '导出Excel' }}
          </ElButton>
        </div>
      </div>

      <!-- 数据表格 - 设置固定高度，自动滚动 -->
      <div class="table-container">
        <ArtTable
          :loading="loading"
          :data="data"
          :columns="columns"
          :height="tableHeight"
          @sort-change="handleSortChange"
        />
      </div>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <span class="total-text">共 {{ formatNumber(total) }} 条记录</span>
        <ElPagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="total"
          :page-sizes="[10, 20, 50, 100]"
          layout="sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </ElCard>

    <!-- 账单详情弹窗 -->
    <StatementDetail v-model="detailVisible" :statement="currentStatement" />
  </div>
</template>

<script setup lang="ts">
  import { ref, reactive, onMounted, onUnmounted, computed } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import {
    Download,
    Document,
    Money,
    Cpu,
    Histogram,
    Files,
    VideoCamera,
    Box,
    Management
  } from '@element-plus/icons-vue'
  import {
    exportBillingStatementApi,
    searchBillingStatementApi,
    deleteBillingStatementApi,
    type BillingStatement,
    type BillingStatementSummary,
    type BillingStatementSearchRequest,
    type BillingStatementExportRequest
  } from '@/api/manager/billing'
  import StatementSearch from './modules/statement-search.vue'
  import StatementDetail from './modules/statement-detail.vue'
  import { useStatementTableConfig } from './modules/table-config'

  defineOptions({ name: 'BillingStatement' })
  const exportLoading = ref(false)

  // 表格配置
  const { loading, deleteLoadingMap, createTableColumns } = useStatementTableConfig()

  // 动态计算表格高度 - 这是关键！
  const tableHeight = ref(400)

  const calculateTableHeight = () => {
    // 获取视口高度，减去顶部导航、搜索栏、统计卡片、分页等高度
    const viewportHeight = window.innerHeight
    // 顶部导航约 60px，搜索栏约 80px，统计卡片约 70px，表格头部约 50px，分页约 60px，padding约 40px
    const reservedHeight = 60 + 80 + 70 + 50 + 60 + 40
    tableHeight.value = Math.max(300, viewportHeight - reservedHeight)
  }

  // 数据
  const data = ref<BillingStatement[]>([])
  const total = ref(0)
  const summary = ref<BillingStatementSummary>({
    totalCount: 0,
    totalAmount: 0,
    cpuCost: 0,
    memoryCost: 0,
    storageCost: 0,
    gpuCost: 0,
    podCost: 0,
    managementFee: 0
  })

  // 分页
  const pagination = reactive({
    page: 1,
    pageSize: 20 // 默认改为20条，显示更多数据
  })

  // 排序
  const sortParams = reactive({
    orderField: 'billing_start_time',
    isAsc: false
  })

  // 搜索表单
  const searchForm = ref({
    startTime: undefined,
    endTime: undefined,
    clusterUuid: undefined,
    projectId: undefined,
    statementType: undefined
  })

  // 搜索参数
  const searchParams = ref<BillingStatementSearchRequest>({})

  // 详情弹窗
  const detailVisible = ref(false)
  const currentStatement = ref<BillingStatement | null>(null)

  // 格式化数字
  const formatNumber = (num: number) => {
    return num.toLocaleString('zh-CN')
  }

  // 格式化金额
  const formatCurrency = (amount: number) => {
    return `¥${amount.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
  }

  // 查看详情
  const viewDetail = (row: BillingStatement) => {
    currentStatement.value = row
    detailVisible.value = true
  }

  // 删除账单
  const deleteStatement = async (row: BillingStatement) => {
    try {
      await ElMessageBox.confirm(
        `确定删除账单「${row.statementNo}」吗？此操作不可恢复！`,
        '删除确认',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      deleteLoadingMap.value[row.id] = true
      await deleteBillingStatementApi(row.id)
      ElMessage.success('删除成功')
      getData()
    } catch (error: any) {
      if (error !== 'cancel') {
        deleteLoadingMap.value[row.id] = false
      }
    } finally {
      deleteLoadingMap.value[row.id] = false
    }
  }

  // 创建表格列
  const { columns } = createTableColumns({
    viewDetail,
    deleteStatement
  })

  // 获取数据
  const getData = async () => {
    loading.value = true
    try {
      const params: BillingStatementSearchRequest = {
        ...searchParams.value,
        page: pagination.page,
        pageSize: pagination.pageSize,
        orderField: sortParams.orderField,
        isAsc: sortParams.isAsc
      }
      const response = await searchBillingStatementApi(params)
      data.value = response.list || []
      total.value = response.total || 0
      summary.value = response.summary || {
        totalCount: 0,
        totalAmount: 0,
        cpuCost: 0,
        memoryCost: 0,
        storageCost: 0,
        gpuCost: 0,
        podCost: 0,
        managementFee: 0
      }
    } finally {
      loading.value = false
    }
  }

  // 搜索
  const handleSearch = (params: BillingStatementSearchRequest) => {
    searchParams.value = params
    pagination.page = 1
    getData()
  }

  // 重置
  const resetSearchParams = () => {
    searchParams.value = {}
    pagination.page = 1
    getData()
  }

  // 分页大小变化
  const handleSizeChange = () => {
    pagination.page = 1
    getData()
  }

  // 页码变化
  const handleCurrentChange = () => {
    getData()
  }

  // 排序变化
  const handleSortChange = ({ prop, order }: { prop: string; order: string }) => {
    if (prop && order) {
      sortParams.orderField = prop
      sortParams.isAsc = order === 'ascending'
    } else {
      sortParams.orderField = 'billing_start_time'
      sortParams.isAsc = false
    }
    getData()
  }

  // 导出Excel
  const handleExport = async () => {
    try {
      await ElMessageBox.confirm('确定要导出当前筛选条件下的账单数据吗？', '导出确认', {
        confirmButtonText: '确定导出',
        cancelButtonText: '取消',
        type: 'info'
      })

      exportLoading.value = true

      const params: BillingStatementExportRequest = {
        ...searchParams.value,
        orderField: sortParams.orderField,
        isAsc: sortParams.isAsc
      }

      const response = await exportBillingStatementApi(params)

      const blob = new Blob([response], {
        type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
      })
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      const fileName = `账单明细_${new Date().toISOString().slice(0, 10).replace(/-/g, '')}.xlsx`
      link.download = fileName
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)

      ElMessage.success('导出成功')
    } catch (error: any) {
      if (error !== 'cancel') {
        exportLoading.value = false
      }
    } finally {
      exportLoading.value = false
    }
  }

  // 初始化
  onMounted(() => {
    calculateTableHeight()
    window.addEventListener('resize', calculateTableHeight)
    getData()
  })

  onUnmounted(() => {
    window.removeEventListener('resize', calculateTableHeight)
  })
</script>

<style lang="scss" scoped>
  .billing-statement-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    padding: 0 0 12px 0;
    box-sizing: border-box;
  }

  /* ========== 统计卡片 - 紧凑单行布局 ========== */
  .stats-cards {
    display: flex;
    gap: 10px;
    margin: 10px 0;
    flex-shrink: 0; /* 防止被压缩 */
  }

  .stat-card {
    flex: 1;
    min-width: 0; /* 允许收缩 */
    background: #fff;
    border-radius: 8px;
    padding: 12px 14px;
    border: 1px solid #e4e7ed;
    display: flex;
    align-items: center;
    gap: 10px;
    transition: all 0.3s;

    &:hover {
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
      transform: translateY(-1px);
    }

    .stat-icon {
      width: 36px;
      height: 36px;
      border-radius: 8px;
      display: flex;
      align-items: center;
      justify-content: center;
      flex-shrink: 0;

      .el-icon {
        font-size: 18px;
        color: #fff;
      }
    }

    .stat-content {
      flex: 1;
      min-width: 0;

      .stat-label {
        font-size: 12px;
        color: #909399;
        margin-bottom: 2px;
        white-space: nowrap;
      }

      .stat-value {
        font-size: 16px;
        font-weight: 600;
        color: #303133;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
      }
    }

    /* 图标背景色 */
    .total-icon {
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    }
    .purple-icon {
      background: linear-gradient(135deg, #722ed1 0%, #9254de 100%);
    }
    .blue-icon {
      background: linear-gradient(135deg, #1890ff 0%, #40a9ff 100%);
    }
    .green-icon {
      background: linear-gradient(135deg, #52c41a 0%, #73d13d 100%);
    }
    .orange-icon {
      background: linear-gradient(135deg, #fa8c16 0%, #ffa940 100%);
    }
    .violet-icon {
      background: linear-gradient(135deg, #9254de 0%, #b37feb 100%);
    }
    .cyan-icon {
      background: linear-gradient(135deg, #13c2c2 0%, #36cfc9 100%);
    }
    .pink-icon {
      background: linear-gradient(135deg, #eb2f96 0%, #f759ab 100%);
    }

    /* 数值高亮色 */
    &.highlight-purple .stat-value {
      color: #722ed1;
    }
    &.highlight-blue .stat-value {
      color: #1890ff;
    }
    &.highlight-green .stat-value {
      color: #52c41a;
    }
    &.highlight-orange .stat-value {
      color: #fa8c16;
    }
    &.highlight-violet .stat-value {
      color: #9254de;
    }
    &.highlight-cyan .stat-value {
      color: #13c2c2;
    }
    &.highlight-pink .stat-value {
      color: #eb2f96;
    }
  }

  /* ========== 表格卡片 - 自适应填充 ========== */
  .table-card {
    flex: 1; /* 关键：撑满剩余空间 */
    display: flex;
    flex-direction: column;
    min-height: 0; /* 允许收缩 */

    :deep(.el-card__body) {
      flex: 1;
      display: flex;
      flex-direction: column;
      padding: 16px;
      min-height: 0;
    }
  }

  .table-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    flex-shrink: 0;

    .table-title {
      display: flex;
      align-items: center;
      gap: 8px;

      .title-text {
        font-size: 16px;
        font-weight: 600;
        color: #303133;
      }

      .title-badge {
        font-size: 12px;
        color: #909399;
        background: #f4f4f5;
        padding: 2px 8px;
        border-radius: 10px;
      }
    }
  }

  .table-container {
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }

  .pagination-wrapper {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 12px;
    padding-top: 12px;
    border-top: 1px solid #ebeef5;
    flex-shrink: 0;

    .total-text {
      font-size: 13px;
      color: #909399;
    }
  }

  .mr-1 {
    margin-right: 4px;
  }

  /* ========== 响应式布局 ========== */
  @media (max-width: 1600px) {
    .stats-cards {
      flex-wrap: wrap;

      .stat-card {
        flex: 0 0 calc(25% - 8px);
      }
    }
  }

  @media (max-width: 1200px) {
    .stats-cards .stat-card {
      flex: 0 0 calc(25% - 8px);
    }
  }

  @media (max-width: 768px) {
    .stats-cards .stat-card {
      flex: 0 0 calc(50% - 5px);
    }
  }
</style>
