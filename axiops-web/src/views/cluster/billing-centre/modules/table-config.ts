import { ref, computed, h } from 'vue'
import type { VNode } from 'vue'
import { ElTag, ElButton, ElIcon, ElTooltip, ElSpace } from 'element-plus'
import { View, Delete } from '@element-plus/icons-vue'
import type { BillingStatement } from '@/api/manager/billing'

// 账单类型映射
const STATEMENT_TYPE_MAP: Record<
  string,
  { label: string; type: 'primary' | 'success' | 'warning' | 'danger' | 'info' }
> = {
  daily: { label: 'daily', type: 'primary' },
  initial: { label: 'initial', type: 'success' },
  config_change: { label: 'config_change', type: 'warning' }
}

// 格式化时间戳为日期时间字符串
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

// 格式化金额
const formatAmount = (amount: number): string => {
  return `¥${amount.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}

// 格式化计费时长
const formatHours = (hours: number): string => {
  if (hours >= 24) {
    const days = Math.floor(hours / 24)
    const remainHours = hours % 24
    if (remainHours > 0) {
      return `${days}d ${remainHours.toFixed(1)}h`
    }
    return `${days}d`
  }
  return `${hours.toFixed(2)}h`
}

export function useStatementTableConfig() {
  const loading = ref(true)
  const deleteLoadingMap = ref<Record<number, boolean>>({})

  const createTableColumns = (handlers: {
    viewDetail: (row: BillingStatement) => void
    deleteStatement: (row: BillingStatement) => void
  }) => {
    const { viewDetail, deleteStatement } = handlers

    const allColumns = [
      {
        prop: 'statementNo',
        label: '账单编号',
        minWidth: 180,
        fixed: 'left' as const,
        visible: true,
        showOverflowTooltip: true
      },
      {
        prop: 'statementType',
        label: '类型',
        width: 120,
        visible: true,
        formatter: (row: Record<string, any>) => {
          const statement = row as BillingStatement
          const typeInfo = STATEMENT_TYPE_MAP[statement.statementType] || {
            label: statement.statementType,
            type: 'info' as const
          }
          return h(ElTag, { type: typeInfo.type, size: 'small' }, () => typeInfo.label)
        }
      },
      {
        prop: 'projectName',
        label: '项目',
        minWidth: 140,
        visible: true,
        showOverflowTooltip: true
      },
      {
        prop: 'clusterName',
        label: '集群',
        minWidth: 140,
        visible: true,
        showOverflowTooltip: true
      },
      {
        prop: 'billingPeriod',
        label: '计费时段',
        minWidth: 180,
        visible: true,
        formatter: (row: Record<string, any>) => {
          const statement = row as BillingStatement
          const start = formatTimestamp(statement.billingStartTime)
          const end = formatTimestamp(statement.billingEndTime)
          return h('div', { class: 'billing-period' }, [
            h('div', {}, start),
            h('div', { style: 'color: #909399' }, `- ${end}`)
          ])
        }
      },
      {
        prop: 'billingHours',
        label: '时长',
        width: 100,
        visible: true,
        formatter: (row: Record<string, any>) => {
          const statement = row as BillingStatement
          return formatHours(statement.billingHours)
        }
      },
      {
        prop: 'cpuCost',
        label: 'CPU费用',
        width: 110,
        visible: true,
        align: 'right' as const,
        formatter: (row: Record<string, any>) => {
          const statement = row as BillingStatement
          return formatAmount(statement.cpuCost)
        }
      },
      {
        prop: 'memoryCost',
        label: '内存费用',
        width: 110,
        visible: true,
        align: 'right' as const,
        formatter: (row: Record<string, any>) => {
          const statement = row as BillingStatement
          return formatAmount(statement.memoryCost)
        }
      },
      {
        prop: 'storageCost',
        label: '存储费用',
        width: 110,
        visible: true,
        align: 'right' as const,
        formatter: (row: Record<string, any>) => {
          const statement = row as BillingStatement
          return formatAmount(statement.storageCost)
        }
      },
      {
        prop: 'gpuCost',
        label: 'GPU费用',
        width: 110,
        visible: true,
        align: 'right' as const,
        formatter: (row: Record<string, any>) => {
          const statement = row as BillingStatement
          return formatAmount(statement.gpuCost)
        }
      },
      {
        prop: 'podCost',
        label: 'Pod费用',
        width: 110,
        visible: true,
        align: 'right' as const,
        formatter: (row: Record<string, any>) => {
          const statement = row as BillingStatement
          return formatAmount(statement.podCost)
        }
      },
      {
        prop: 'managementFee',
        label: '管理费',
        width: 110,
        visible: true,
        align: 'right' as const,
        formatter: (row: Record<string, any>) => {
          const statement = row as BillingStatement
          return formatAmount(statement.managementFee)
        }
      },
      {
        prop: 'totalAmount',
        label: '总费用',
        width: 120,
        visible: true,
        align: 'right' as const,
        formatter: (row: Record<string, any>) => {
          const statement = row as BillingStatement
          return h(
            'span',
            { style: 'color: #722ed1; font-weight: 600' },
            formatAmount(statement.totalAmount)
          )
        }
      },
      {
        prop: 'cpuCapacity',
        label: 'CPU配额',
        width: 100,
        visible: false
      },
      {
        prop: 'memCapacity',
        label: '内存配额',
        width: 100,
        visible: false
      },
      {
        prop: 'storageLimit',
        label: '存储配额',
        width: 100,
        visible: false
      },
      {
        prop: 'workspaceCount',
        label: '工作空间数',
        width: 100,
        visible: false
      },
      {
        prop: 'applicationCount',
        label: '应用数',
        width: 100,
        visible: false
      },
      {
        prop: 'createdBy',
        label: '创建人',
        width: 100,
        visible: false
      },
      {
        prop: 'operation',
        label: '操作',
        width: 100,
        fixed: 'right' as const,
        visible: true,
        formatter: (row: Record<string, any>): VNode => {
          const statement = row as BillingStatement
          const isDeleting = deleteLoadingMap.value[statement.id] || false

          return h(ElSpace, { size: 4 }, () => [
            h(ElTooltip, { content: '查看详情', placement: 'top' }, () =>
              h(
                ElButton,
                {
                  type: 'primary',
                  link: true,
                  size: 'small',
                  onClick: () => viewDetail(statement)
                },
                () => h(ElIcon, { size: 16 }, () => h(View))
              )
            ),
            h(ElTooltip, { content: '删除', placement: 'top' }, () =>
              h(
                ElButton,
                {
                  type: 'danger',
                  link: true,
                  size: 'small',
                  loading: isDeleting,
                  onClick: () => deleteStatement(statement)
                },
                () => h(ElIcon, { size: 16 }, () => h(Delete))
              )
            )
          ])
        }
      }
    ]

    const columnChecks = ref<Record<string, boolean>>(
      allColumns.reduce(
        (acc, col) => {
          if (col.prop) acc[col.prop] = col.visible
          return acc
        },
        {} as Record<string, boolean>
      )
    )

    const columns = computed(() =>
      allColumns.filter((col) => !col.prop || columnChecks.value[col.prop] !== false)
    )

    return { columnChecks, columns }
  }

  return {
    loading,
    deleteLoadingMap,
    createTableColumns
  }
}
