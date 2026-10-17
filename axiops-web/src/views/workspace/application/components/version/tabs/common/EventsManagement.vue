<template>
  <div class="events-management-optimized">
    <ElCard shadow="never" class="events-card">
      <template #header>
        <div class="header-wrapper">
          <div class="header-left">
            <Activity :size="16" />
            <span class="header-title">事件列表</span>
            <ElTag v-if="pagination.total > 0" type="info" size="small">
              {{ pagination.total }} 条
            </ElTag>
          </div>
          <div class="header-right">
            <ElSelect
              v-model="queryParams.type"
              placeholder="事件类型"
              clearable
              size="default"
              style="width: 130px"
              @change="handleSearch"
            >
              <ElOption label="全部" value="" />
              <ElOption label="正常" value="Normal">
                <div class="option-with-icon">
                  <span>正常</span>
                </div>
              </ElOption>
              <ElOption label="警告" value="Warning">
                <div class="option-with-icon">
                  <span>警告</span>
                </div>
              </ElOption>
            </ElSelect>
            <ElButton circle size="default" :loading="loading" @click="handleRefresh">
              <template #icon>
                <RefreshCw :size="14" />
              </template>
            </ElButton>
          </div>
        </div>
      </template>

      <div class="table-container">
        <!-- 统计卡片 -->
        <div v-if="events.length > 0" class="stats-cards">
          <div class="stat-card normal-card">
            <div class="stat-icon">
              <CheckCircle :size="22" />
            </div>
            <div class="stat-info">
              <div class="stat-label">正常事件</div>
              <div class="stat-value">{{ normalEventsCount }}</div>
            </div>
          </div>
          <div class="stat-card warning-card">
            <div class="stat-icon">
              <AlertTriangle :size="22" />
            </div>
            <div class="stat-info">
              <div class="stat-label">警告事件</div>
              <div class="stat-value">{{ warningEventsCount }}</div>
            </div>
          </div>
          <div class="stat-card total-card">
            <div class="stat-icon">
              <Activity :size="22" />
            </div>
            <div class="stat-info">
              <div class="stat-label">当前页数量</div>
              <div class="stat-value">{{ events.length }}</div>
            </div>
          </div>
        </div>

        <!-- 事件表格区域 -->
        <div class="table-scroll-area">
          <ElTable
            :data="events"
            v-loading="loading"
            stripe
            class="events-table"
            :header-cell-style="{ background: '#fafafa', color: '#606266' }"
          >
            <!-- 类型列 -->
            <ElTableColumn prop="type" label="类型" width="100" align="center">
              <template #default="{ row }">
                <ElTag :type="row.type === 'Normal' ? 'success' : 'warning'" size="default">
                  {{ row.type === 'Normal' ? '正常' : '警告' }}
                </ElTag>
              </template>
            </ElTableColumn>

            <ElTableColumn prop="reason" label="原因" width="180">
              <template #default="{ row }">
                <div class="reason-cell">
                  <Info :size="14" class="icon-muted" />
                  <span>{{ row.reason }}</span>
                </div>
              </template>
            </ElTableColumn>

            <ElTableColumn prop="message" label="消息" min-width="300" show-overflow-tooltip>
              <template #default="{ row }">
                <div class="message-cell">{{ row.message }}</div>
              </template>
            </ElTableColumn>

            <ElTableColumn prop="involvedObjectKind" label="对象类型" width="120">
              <template #default="{ row }">
                <ElTag size="small" type="info">{{ row.involvedObjectKind || '-' }}</ElTag>
              </template>
            </ElTableColumn>

            <ElTableColumn
              prop="involvedObjectName"
              label="对象名称"
              width="180"
              show-overflow-tooltip
            >
              <template #default="{ row }">
                <span class="text-content">{{ row.involvedObjectName || '-' }}</span>
              </template>
            </ElTableColumn>

            <ElTableColumn prop="count" label="次数" width="80" align="center">
              <template #default="{ row }">
                <ElTag size="small" type="info" round>{{ row.count || 1 }}</ElTag>
              </template>
            </ElTableColumn>

            <ElTableColumn prop="lastTimestamp" label="最后发生时间" width="160" align="center">
              <template #default="{ row }">
                <div class="time-cell">
                  <Clock :size="14" class="icon-muted" />
                  <span>{{ formatTime(row.lastTimestamp) }}</span>
                </div>
              </template>
            </ElTableColumn>

            <!-- 表格空状态插槽 -->
            <template #empty>
              <div class="table-empty-state">
                <ElEmpty :image-size="80">
                  <template #description>
                    <div class="empty-content">
                      <p class="empty-title">暂无事件记录</p>
                      <p class="empty-hint">系统还没有记录任何事件</p>
                    </div>
                  </template>
                </ElEmpty>
              </div>
            </template>
          </ElTable>
        </div>

        <!-- 分页固定在底部 -->
        <div v-if="pagination.total > 0" class="pagination-footer">
          <ElPagination
            v-model:current-page="queryParams.page"
            v-model:page-size="queryParams.pageSize"
            :total="pagination.total"
            :page-sizes="[10, 20, 50, 100]"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="handleSizeChange"
            @current-change="handlePageChange"
            background
            small
          />
        </div>
      </div>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { ref, reactive, computed, watch, onMounted } from 'vue'
  import { ElMessage } from 'element-plus'
  import { Activity, RefreshCw, CheckCircle, AlertTriangle, Clock, Info } from 'lucide-vue-next'
  import {
    queryEventsApi,
    type OnecProjectVersion,
    type EventInfo,
    type EventsQueryParams
  } from '@/api'
  import dayjs from 'dayjs'

  interface Props {
    version: OnecProjectVersion
    refreshTrigger?: number
  }

  const props = defineProps<Props>()

  const events = ref<EventInfo[]>([])
  const loading = ref(false)

  const queryParams = reactive<EventsQueryParams>({
    page: 1,
    pageSize: 20,
    type: ''
  })

  const pagination = reactive({
    total: 0
  })

  // ==================== 计算属性 ====================

  const normalEventsCount = computed(() => {
    return events.value.filter((e) => e.type === 'Normal').length
  })

  const warningEventsCount = computed(() => {
    return events.value.filter((e) => e.type === 'Warning').length
  })

  // ==================== 工具函数 ====================

  const formatTime = (timestamp: number) => {
    if (!timestamp) return '-'
    return dayjs(timestamp * 1000).format('MM-DD HH:mm:ss')
  }

  // ==================== 核心方法 ====================

  /**
   * 加载事件列表
   * 🔧 关键修复：添加防重复加载保护，避免 watch 和 onMounted 同时触发导致重复请求
   */
  const loadEvents = async () => {
    // 防止重复加载 - 如果已经在加载中，直接返回
    if (loading.value) return

    loading.value = true
    try {
      const response = await queryEventsApi(props.version.id, queryParams)
      events.value = response.items || []
      pagination.total = response.total || 0
    } catch (error) {
      console.error('[EventsManagement] 加载失败:', error)
      events.value = []
      pagination.total = 0
    } finally {
      loading.value = false
    }
  }

  // ==================== 事件处理 ====================

  /** 搜索/筛选 */
  const handleSearch = () => {
    queryParams.page = 1
    loadEvents()
  }

  /** 页码变化 */
  const handlePageChange = () => {
    loadEvents()
  }

  /** 每页条数变化 */
  const handleSizeChange = () => {
    queryParams.page = 1
    loadEvents()
  }

  /** 手动刷新 */
  const handleRefresh = async () => {
    await loadEvents()
    // 只有加载成功后才提示
    if (!loading.value) {
      ElMessage.success('刷新成功')
    }
  }

  // ==================== 生命周期 ====================

  /**
   * 监听 refreshTrigger 变化
   * 当父组件需要刷新数据时，会改变 refreshTrigger 的值
   */
  watch(
    () => props.refreshTrigger,
    (newVal) => {
      if (newVal && newVal > 0) {
        loadEvents()
      }
    }
  )

  /**
   * 组件挂载时加载数据
   * 注意：由于 loadEvents 内部有防重复加载保护，
   * 即使 watch 和 onMounted 同时触发，也只会执行一次请求
   */
  onMounted(() => {
    loadEvents()
  })
</script>

<style lang="scss" scoped>
  .events-management-optimized {
    height: 100%;
    min-height: 400px;
    max-height: 800px;
    display: flex;
    flex-direction: column;

    .events-card {
      height: 100%;
      display: flex;
      flex-direction: column;
      border-radius: 8px;
      overflow: hidden;

      :deep(.el-card__header) {
        flex-shrink: 0;
        padding: 14px 20px;
        background: linear-gradient(to right, #f8fafc, #fff);
        border-bottom: 1px solid #eef2f6;
      }

      :deep(.el-card__body) {
        flex: 1;
        padding: 0;
        display: flex;
        flex-direction: column;
        min-height: 0;
        overflow: hidden;
      }

      // 头部区域
      .header-wrapper {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 16px;

        .header-left {
          display: flex;
          align-items: center;
          gap: 8px;
          flex: 1;
          min-width: 0;

          .header-title {
            font-size: 15px;
            font-weight: 600;
            color: #303133;
          }
        }

        .header-right {
          display: flex;
          align-items: center;
          gap: 8px;
          flex-shrink: 0;

          .option-with-icon {
            display: flex;
            align-items: center;
            gap: 6px;
          }
        }
      }

      // 表格容器
      .table-container {
        flex: 1;
        display: flex;
        flex-direction: column;
        min-height: 0;
        overflow: hidden;

        // 统计卡片
        .stats-cards {
          flex-shrink: 0;
          display: grid;
          grid-template-columns: repeat(3, 1fr);
          gap: 12px;
          padding: 16px 20px;
          background: #fafbfc;
          border-bottom: 1px solid #eef2f6;

          .stat-card {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 14px 16px;
            border-radius: 8px;
            transition: all 0.2s ease;
            border: 1px solid;

            &:hover {
              transform: translateY(-1px);
              box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
            }

            &.normal-card {
              background: linear-gradient(135deg, #f0fdf4 0%, #dcfce7 100%);
              border-color: #86efac;

              .stat-icon {
                background: rgba(34, 197, 94, 0.15);
                color: #16a34a;
              }
            }

            &.warning-card {
              background: linear-gradient(135deg, #fffbeb 0%, #fef3c7 100%);
              border-color: #fcd34d;

              .stat-icon {
                background: rgba(245, 158, 11, 0.15);
                color: #d97706;
              }
            }

            &.total-card {
              background: linear-gradient(135deg, #eff6ff 0%, #dbeafe 100%);
              border-color: #93c5fd;

              .stat-icon {
                background: rgba(59, 130, 246, 0.15);
                color: #2563eb;
              }
            }

            .stat-icon {
              width: 44px;
              height: 44px;
              display: flex;
              align-items: center;
              justify-content: center;
              border-radius: 10px;
              flex-shrink: 0;
            }

            .stat-info {
              flex: 1;

              .stat-label {
                font-size: 12px;
                color: #6b7280;
                margin-bottom: 2px;
                font-weight: 500;
              }

              .stat-value {
                font-size: 22px;
                font-weight: 700;
                color: #1f2937;
                line-height: 1;
              }
            }
          }
        }

        // 表格滚动区域
        .table-scroll-area {
          flex: 1;
          overflow-y: auto;
          padding: 16px 20px;

          // 自定义滚动条
          &::-webkit-scrollbar {
            width: 6px;
          }

          &::-webkit-scrollbar-track {
            background: #f1f5f9;
            border-radius: 3px;
          }

          &::-webkit-scrollbar-thumb {
            background: #cbd5e1;
            border-radius: 3px;

            &:hover {
              background: #94a3b8;
            }
          }

          // 事件表格
          .events-table {
            width: 100%;

            // 表格单元格样式
            .reason-cell {
              display: flex;
              align-items: center;
              gap: 6px;
              font-weight: 500;
            }

            .message-cell {
              font-size: 13px;
              color: #4b5563;
              line-height: 1.5;
            }

            .text-content {
              color: #606266;
            }

            .time-cell {
              display: flex;
              align-items: center;
              justify-content: center;
              gap: 4px;
              font-size: 13px;
              color: #606266;
            }

            .icon-muted {
              color: #909399;
              flex-shrink: 0;
            }

            // 表格空状态
            .table-empty-state {
              padding: 20px 0;

              .empty-content {
                text-align: center;

                .empty-title {
                  margin: 0 0 4px 0;
                  font-size: 14px;
                  font-weight: 500;
                  color: #606266;
                }

                .empty-hint {
                  margin: 0;
                  font-size: 12px;
                  color: #909399;
                }
              }
            }
          }
        }

        // 分页固定在底部
        .pagination-footer {
          flex-shrink: 0;
          padding: 12px 20px;
          background: #fafbfc;
          border-top: 1px solid #eef2f6;
          display: flex;
          justify-content: flex-end;
        }
      }
    }
  }
</style>
