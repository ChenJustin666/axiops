import { ref, h } from 'vue'
import { ElDropdown, ElDropdownMenu, ElDropdownItem, ElButton, ElTag } from 'element-plus'
import { MoreVertical, Edit2, RefreshCw, Settings, Receipt, Trash2 } from 'lucide-vue-next'
import type { ProjectCluster } from '@/api'

// 表格配置 Hook
export function useResourceTableConfig() {
  const loading = ref(false)
  const showSearchBar = ref(false)
  const submitLoading = ref(false)
  const syncLoadingMap = ref<Record<number, boolean>>({})

  // 格式化日期
  const formatDate = (timestamp: number) => {
    if (!timestamp) return '-'
    const date = new Date(timestamp * 1000)
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    const hours = String(date.getHours()).padStart(2, '0')
    const minutes = String(date.getMinutes()).padStart(2, '0')
    return `${month}/${day} ${hours}:${minutes}`
  }

  // 格式化资源值
  const formatResourceValue = (value: string | number): string => {
    if (typeof value === 'number') return value.toFixed(2)
    if (!value) return '0.00'
    const numStr = String(value).replace(/[^0-9.]/g, '')
    const num = parseFloat(numStr)
    return isNaN(num) ? '0.00' : num.toFixed(2)
  }

  // 创建表格列配置
  const createTableColumns = (handlers: {
    handleEditResource: (resource: ProjectCluster) => void
    handleSyncResource: (resource: ProjectCluster) => void
    handleDeleteResource: (resource: ProjectCluster) => void
    handleBillingConfig: (resource: ProjectCluster) => void
    handleGenerateBilling: (resource: ProjectCluster) => void
    handleCommand: (command: string, resource: ProjectCluster) => void
  }) => {
    const columns = ref([
      {
        prop: 'clusterName',
        label: '集群名称',
        minWidth: 180,
        show: true,
        fixed: 'left'
      },
      {
        prop: 'clusterUuid',
        label: '集群UUID',
        minWidth: 280,
        show: false
      },
      {
        prop: 'cpuCapacity',
        label: 'CPU (已用/容量)',
        minWidth: 140,
        show: true,
        formatter: (row: ProjectCluster) => {
          return `${formatResourceValue(row.cpuAllocated)}/${formatResourceValue(row.cpuCapacity)} 核`
        }
      },
      {
        prop: 'memCapacity',
        label: '内存 (已用/容量)',
        minWidth: 150,
        show: true,
        formatter: (row: ProjectCluster) => {
          return `${formatResourceValue(row.memAllocated)}/${formatResourceValue(row.memCapacity)} GiB`
        }
      },
      {
        prop: 'storageLimit',
        label: '存储 (已用/配额)',
        minWidth: 150,
        show: true,
        formatter: (row: ProjectCluster) => {
          return `${formatResourceValue(row.storageAllocated)}/${formatResourceValue(row.storageLimit)} GiB`
        }
      },
      {
        prop: 'gpuCapacity',
        label: 'GPU (已用/容量)',
        minWidth: 130,
        show: false,
        formatter: (row: ProjectCluster) => {
          return `${formatResourceValue(row.gpuAllocated)}/${formatResourceValue(row.gpuCapacity)}`
        }
      },
      {
        prop: 'podsLimit',
        label: 'Pods (已用/配额)',
        minWidth: 130,
        show: true,
        formatter: (row: ProjectCluster) => {
          return `${row.podsAllocated || 0}/${row.podsLimit || 0}`
        }
      },
      {
        prop: 'cpuOvercommitRatio',
        label: 'CPU超分',
        minWidth: 90,
        show: false,
        formatter: (row: ProjectCluster) => `${row.cpuOvercommitRatio || 1}x`
      },
      {
        prop: 'memOvercommitRatio',
        label: '内存超分',
        minWidth: 90,
        show: false,
        formatter: (row: ProjectCluster) => `${row.memOvercommitRatio || 1}x`
      },
      {
        prop: 'updatedAt',
        label: '更新时间',
        minWidth: 120,
        show: true,
        formatter: (row: ProjectCluster) => formatDate(row.updatedAt)
      },
      {
        prop: 'actions',
        label: '操作',
        width: 100,
        fixed: 'right',
        show: true,
        render: (row: ProjectCluster) => {
          return h(
            ElDropdown,
            {
              trigger: 'click',
              onCommand: (cmd: string) => handlers.handleCommand(cmd, row)
            },
            {
              default: () =>
                h(
                  ElButton,
                  {
                    link: true,
                    type: 'primary'
                  },
                  {
                    default: () => [
                      h('span', { style: { marginRight: '4px' } }, '操作'),
                      h(MoreVertical, { size: 14 })
                    ]
                  }
                ),
              dropdown: () =>
                h(ElDropdownMenu, null, {
                  default: () => [
                    h(
                      ElDropdownItem,
                      { command: 'edit' },
                      {
                        default: () => [
                          h(Edit2, {
                            size: 14,
                            style: { marginRight: '8px', color: '#6b7280' }
                          }),
                          h('span', '编辑配额')
                        ]
                      }
                    ),
                    h(
                      ElDropdownItem,
                      { command: 'sync' },
                      {
                        default: () => [
                          h(RefreshCw, {
                            size: 14,
                            style: { marginRight: '8px', color: '#6b7280' }
                          }),
                          h('span', '同步状态')
                        ]
                      }
                    ),
                    h(
                      ElDropdownItem,
                      { command: 'billing-config', divided: true },
                      {
                        default: () => [
                          h(Settings, {
                            size: 14,
                            style: { marginRight: '8px', color: '#6b7280' }
                          }),
                          h('span', '费用配置')
                        ]
                      }
                    ),
                    h(
                      ElDropdownItem,
                      { command: 'generate-billing' },
                      {
                        default: () => [
                          h(Receipt, {
                            size: 14,
                            style: { marginRight: '8px', color: '#6b7280' }
                          }),
                          h('span', '生成账单')
                        ]
                      }
                    ),
                    h(
                      ElDropdownItem,
                      {
                        command: 'delete',
                        divided: true,
                        style: { color: '#ef4444' }
                      },
                      {
                        default: () => [
                          h(Trash2, {
                            size: 14,
                            style: { marginRight: '8px', color: '#ef4444' }
                          }),
                          h('span', '删除资源池')
                        ]
                      }
                    )
                  ]
                })
            }
          )
        }
      }
    ])

    return { columns }
  }

  return {
    loading,
    showSearchBar,
    submitLoading,
    syncLoadingMap,
    createTableColumns,
    formatDate
  }
}
