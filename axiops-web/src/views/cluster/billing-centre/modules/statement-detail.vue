<template>
  <ElDialog
    v-model="visible"
    title="账单详情"
    width="750px"
    align-center
    destroy-on-close
    @close="handleClose"
  >
    <div class="statement-detail" v-if="statement">
      <!-- 顶部摘要卡片 -->
      <div class="summary-card">
        <div class="summary-left">
          <div class="statement-no">{{ statement.statementNo }}</div>
          <div class="statement-meta">
            <ElTag :type="getTypeInfo(statement.statementType).type" size="small">
              {{ getTypeInfo(statement.statementType).label }}
            </ElTag>
            <span class="meta-divider">|</span>
            <span class="meta-text">{{ statement.projectName }}</span>
            <span class="meta-divider">|</span>
            <span class="meta-text">{{ statement.clusterName }}</span>
          </div>
        </div>
        <div class="summary-right">
          <div class="total-label">总费用</div>
          <div class="total-amount">{{ formatAmount(statement.totalAmount) }}</div>
        </div>
      </div>

      <!-- 计费时段 -->
      <div class="time-section">
        <div class="time-item">
          <ElIcon class="time-icon"><Clock /></ElIcon>
          <div class="time-content">
            <div class="time-label">计费周期</div>
            <div class="time-value">
              {{ formatTimestamp(statement.billingStartTime) }} 至
              {{ formatTimestamp(statement.billingEndTime) }}
            </div>
          </div>
        </div>
        <div class="time-item">
          <ElIcon class="time-icon"><Timer /></ElIcon>
          <div class="time-content">
            <div class="time-label">计费时长</div>
            <div class="time-value">{{ formatHours(statement.billingHours) }}</div>
          </div>
        </div>
      </div>

      <!-- 费用明细 -->
      <div class="detail-section">
        <div class="section-header">
          <ElIcon><Money /></ElIcon>
          <span>费用明细</span>
        </div>
        <div class="cost-grid">
          <div class="cost-item">
            <div class="cost-icon cpu"
              ><svg viewBox="0 0 24 24" fill="currentColor">
                <path
                  d="M9 2v2H7v2H5v2H3v2h2v2H3v2h2v2h2v2h2v2h2v-2h2v2h2v-2h2v-2h2v-2h2v-2h-2v-2h2V8h-2V6h-2V4h-2V2h-2v2h-2V2H9zm0 4h6v2h2v6h-2v2H9v-2H7V8h2V6z"
                /></svg
            ></div>
            <div class="cost-info">
              <div class="cost-name">CPU费用</div>
              <div class="cost-value">{{ formatAmount(statement.cpuCost) }}</div>
            </div>
          </div>
          <div class="cost-item">
            <div class="cost-icon memory"
              ><svg viewBox="0 0 24 24" fill="currentColor">
                <path
                  d="M2 7h2v10H2V7zm4 0h1v10H6V7zm2 0h2v10H8V7zm3 0h2v10h-2V7zm3 0h1v10h-1V7zm2 0h2v10h-2V7zm3 0h2v10h-2V7z"
                /></svg
            ></div>
            <div class="cost-info">
              <div class="cost-name">内存费用</div>
              <div class="cost-value">{{ formatAmount(statement.memoryCost) }}</div>
            </div>
          </div>
          <div class="cost-item">
            <div class="cost-icon storage"
              ><svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M4 4h16v4H4V4zm0 6h16v4H4v-4zm0 6h16v4H4v-4z" /></svg
            ></div>
            <div class="cost-info">
              <div class="cost-name">存储费用</div>
              <div class="cost-value">{{ formatAmount(statement.storageCost) }}</div>
            </div>
          </div>
          <div class="cost-item">
            <div class="cost-icon gpu"
              ><svg viewBox="0 0 24 24" fill="currentColor">
                <path
                  d="M5 3h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2zm1 4v2h2V7H6zm4 0v2h2V7h-2zm4 0v2h2V7h-2zm-8 4v2h2v-2H6zm4 0v2h2v-2h-2zm4 0v2h2v-2h-2zm-8 4v2h2v-2H6zm4 0v2h2v-2h-2zm4 0v2h2v-2h-2z"
                /></svg
            ></div>
            <div class="cost-info">
              <div class="cost-name">GPU费用</div>
              <div class="cost-value">{{ formatAmount(statement.gpuCost) }}</div>
            </div>
          </div>
          <div class="cost-item">
            <div class="cost-icon pod"
              ><svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" /></svg
            ></div>
            <div class="cost-info">
              <div class="cost-name">Pod费用</div>
              <div class="cost-value">{{ formatAmount(statement.podCost) }}</div>
            </div>
          </div>
          <div class="cost-item">
            <div class="cost-icon mgmt"
              ><svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4z" /></svg
            ></div>
            <div class="cost-info">
              <div class="cost-name">管理费</div>
              <div class="cost-value">{{ formatAmount(statement.managementFee) }}</div>
            </div>
          </div>
        </div>
        <div class="cost-summary">
          <div class="cost-summary-item">
            <span>资源费用小计</span>
            <span class="summary-value">{{ formatAmount(statement.resourceCostTotal) }}</span>
          </div>
          <div class="cost-summary-item total">
            <span>总费用</span>
            <span class="summary-value">{{ formatAmount(statement.totalAmount) }}</span>
          </div>
        </div>
      </div>

      <!-- 资源配额快照 -->
      <div class="detail-section">
        <div class="section-header">
          <ElIcon><Box /></ElIcon>
          <span>资源配额快照</span>
        </div>
        <div class="info-grid">
          <div class="info-item">
            <span class="info-label">CPU</span>
            <span class="info-value">{{ statement.cpuCapacity }} 核</span>
          </div>
          <div class="info-item">
            <span class="info-label">内存</span>
            <span class="info-value">{{ statement.memCapacity }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">存储</span>
            <span class="info-value">{{ statement.storageLimit }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">GPU</span>
            <span class="info-value">{{ statement.gpuCapacity }} 卡</span>
          </div>
          <div class="info-item">
            <span class="info-label">Pod配额</span>
            <span class="info-value">{{ statement.podsLimit }} 个</span>
          </div>
          <div class="info-item">
            <span class="info-label">工作空间</span>
            <span class="info-value">{{ statement.workspaceCount }} 个</span>
          </div>
          <div class="info-item">
            <span class="info-label">应用数量</span>
            <span class="info-value">{{ statement.applicationCount }} 个</span>
          </div>
          <div class="info-item">
            <span class="info-label">创建人</span>
            <span class="info-value">{{ statement.createdBy || '-' }}</span>
          </div>
        </div>
      </div>

      <!-- 单价配置快照 -->
      <div class="detail-section">
        <div class="section-header">
          <ElIcon><PriceTag /></ElIcon>
          <span>单价配置快照</span>
        </div>
        <div class="price-grid">
          <div class="price-item">
            <span class="price-label">CPU</span>
            <span class="price-value">¥{{ statement.priceCpu }}/核/小时</span>
          </div>
          <div class="price-item">
            <span class="price-label">内存</span>
            <span class="price-value">¥{{ statement.priceMemory }}/GiB/小时</span>
          </div>
          <div class="price-item">
            <span class="price-label">存储</span>
            <span class="price-value">¥{{ statement.priceStorage }}/GiB/小时</span>
          </div>
          <div class="price-item">
            <span class="price-label">GPU</span>
            <span class="price-value">¥{{ statement.priceGpu }}/卡/小时</span>
          </div>
          <div class="price-item">
            <span class="price-label">Pod</span>
            <span class="price-value">¥{{ statement.pricePod }}/个/小时</span>
          </div>
          <div class="price-item">
            <span class="price-label">管理费</span>
            <span class="price-value">¥{{ statement.priceManagementFee }}/小时</span>
          </div>
        </div>
      </div>

      <!-- 备注 -->
      <div class="detail-section" v-if="statement.remark">
        <div class="section-header">
          <ElIcon><Document /></ElIcon>
          <span>备注</span>
        </div>
        <div class="remark-content">{{ statement.remark }}</div>
      </div>
    </div>

    <template #footer>
      <ElButton @click="handleClose">关闭</ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import { Clock, Timer, Money, Box, PriceTag, Document } from '@element-plus/icons-vue'
  import type { BillingStatement } from '@/api/manager/billing'

  interface Props {
    modelValue: boolean
    statement: BillingStatement | null
  }

  interface Emits {
    (e: 'update:modelValue', value: boolean): void
  }

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  // 弹窗控制
  const visible = computed({
    get: () => props.modelValue,
    set: (value) => emit('update:modelValue', value)
  })

  // 账单类型映射
  const STATEMENT_TYPE_MAP: Record<
    string,
    { label: string; type: 'primary' | 'success' | 'warning' | 'danger' | 'info' }
  > = {
    daily: { label: '日常账单', type: 'primary' },
    initial: { label: '初始账单', type: 'success' },
    config_change: { label: '配置变更', type: 'warning' }
  }

  // 获取类型信息
  const getTypeInfo = (type: string) => {
    return STATEMENT_TYPE_MAP[type] || { label: type, type: 'info' as const }
  }

  // 格式化时间戳
  const formatTimestamp = (timestamp: number): string => {
    if (!timestamp) return '-'
    const date = new Date(timestamp * 1000)
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    const hour = String(date.getHours()).padStart(2, '0')
    const minute = String(date.getMinutes()).padStart(2, '0')
    return `${year}-${month}-${day} ${hour}:${minute}`
  }

  // 格式化小时
  const formatHours = (hours: number): string => {
    if (hours >= 24) {
      const days = Math.floor(hours / 24)
      const remainHours = hours % 24
      if (remainHours > 0) {
        return `${days} 天 ${remainHours.toFixed(1)} 小时`
      }
      return `${days} 天`
    }
    return `${hours.toFixed(2)} 小时`
  }

  // 格式化金额
  const formatAmount = (amount: number): string => {
    return `¥${amount.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
  }

  // 关闭
  const handleClose = () => {
    visible.value = false
  }
</script>

<style lang="scss" scoped>
  .statement-detail {
    max-height: 65vh;
    overflow-y: auto;
    padding-right: 8px;

    &::-webkit-scrollbar {
      width: 6px;
    }

    &::-webkit-scrollbar-thumb {
      background: #dcdfe6;
      border-radius: 3px;
    }
  }

  // 顶部摘要卡片
  .summary-card {
    display: flex;
    justify-content: space-between;
    align-items: center;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    border-radius: 12px;
    padding: 20px 24px;
    margin-bottom: 20px;
    color: #fff;

    .summary-left {
      .statement-no {
        font-size: 14px;
        font-family: var(--ax-font-mono);
        opacity: 0.9;
        margin-bottom: 8px;
      }

      .statement-meta {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 13px;

        .meta-divider {
          opacity: 0.5;
        }

        .meta-text {
          opacity: 0.9;
        }

        :deep(.el-tag) {
          border: none;
        }
      }
    }

    .summary-right {
      text-align: right;

      .total-label {
        font-size: 13px;
        opacity: 0.85;
        margin-bottom: 4px;
      }

      .total-amount {
        font-size: 28px;
        font-weight: 700;
      }
    }
  }

  // 计费时段
  .time-section {
    display: flex;
    gap: 24px;
    margin-bottom: 20px;
    padding: 16px;
    background: #f8f9fa;
    border-radius: 8px;

    .time-item {
      display: flex;
      align-items: center;
      gap: 12px;
      flex: 1;

      .time-icon {
        font-size: 20px;
        color: #722ed1;
      }

      .time-content {
        .time-label {
          font-size: 12px;
          color: #909399;
          margin-bottom: 2px;
        }

        .time-value {
          font-size: 14px;
          color: #303133;
          font-weight: 500;
        }
      }
    }
  }

  // 通用分区样式
  .detail-section {
    margin-bottom: 20px;

    &:last-child {
      margin-bottom: 0;
    }

    .section-header {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 15px;
      font-weight: 600;
      color: #303133;
      margin-bottom: 14px;
      padding-bottom: 10px;
      border-bottom: 1px solid #ebeef5;

      .el-icon {
        color: #722ed1;
      }
    }
  }

  // 费用网格
  .cost-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px;
    margin-bottom: 16px;
  }

  .cost-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 14px 16px;
    background: #fafafa;
    border-radius: 8px;
    transition: all 0.2s;

    &:hover {
      background: #f0f2f5;
    }

    .cost-icon {
      width: 36px;
      height: 36px;
      border-radius: 8px;
      display: flex;
      align-items: center;
      justify-content: center;

      svg {
        width: 20px;
        height: 20px;
      }

      &.cpu {
        background: rgba(24, 144, 255, 0.1);
        color: #1890ff;
      }

      &.memory {
        background: rgba(82, 196, 26, 0.1);
        color: #52c41a;
      }

      &.storage {
        background: rgba(250, 140, 22, 0.1);
        color: #fa8c16;
      }

      &.gpu {
        background: rgba(146, 84, 222, 0.1);
        color: #9254de;
      }

      &.pod {
        background: rgba(19, 194, 194, 0.1);
        color: #13c2c2;
      }

      &.mgmt {
        background: rgba(235, 47, 150, 0.1);
        color: #eb2f96;
      }
    }

    .cost-info {
      .cost-name {
        font-size: 12px;
        color: #909399;
        margin-bottom: 2px;
      }

      .cost-value {
        font-size: 15px;
        font-weight: 600;
        color: #303133;
      }
    }
  }

  // 费用汇总
  .cost-summary {
    border-top: 1px dashed #e4e7ed;
    padding-top: 12px;

    .cost-summary-item {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 8px 0;
      font-size: 14px;
      color: #606266;

      .summary-value {
        font-weight: 600;
        color: #303133;
      }

      &.total {
        margin-top: 4px;
        padding-top: 12px;
        border-top: 2px solid #722ed1;

        span:first-child {
          font-weight: 600;
          color: #303133;
        }

        .summary-value {
          font-size: 20px;
          color: #722ed1;
        }
      }
    }
  }

  // 信息网格
  .info-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 12px;
  }

  .info-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 12px;
    background: #fafafa;
    border-radius: 6px;

    .info-label {
      font-size: 12px;
      color: #909399;
    }

    .info-value {
      font-size: 14px;
      color: #303133;
      font-weight: 500;
    }
  }

  // 单价网格
  .price-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px;
  }

  .price-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 14px;
    background: #fafafa;
    border-radius: 6px;

    .price-label {
      font-size: 13px;
      color: #606266;
    }

    .price-value {
      font-size: 13px;
      color: #303133;
      font-weight: 500;
    }
  }

  // 备注
  .remark-content {
    background: #fafafa;
    border-radius: 6px;
    padding: 14px;
    font-size: 13px;
    color: #606266;
    line-height: 1.6;
  }
</style>
