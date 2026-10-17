import { ref, computed, h } from 'vue'
import type { VNode } from 'vue'
import { ElTag } from 'element-plus'
import ArtButtonMore from '@/components/core/forms/art-button-more/index.vue'
import type { ButtonMoreItem } from '@/components/core/forms/art-button-more/index.vue'
import { formatTimestamp } from '@/utils/format'
import {
  type BillingPriceConfig,
  formatCpuPrice,
  formatMemoryPrice,
  formatStoragePrice,
  formatGpuPrice,
  formatPodPrice,
  formatManagementFee
} from '@/api/manager/billing'

export function useBillingTableConfig() {
  const loading = ref(true)
  const showSearchBar = ref(true)
  const deleteLoadingMap = ref<Record<number, boolean>>({})

  const createTableColumns = (handlers: {
    buttonMoreClick: (item: ButtonMoreItem, row: BillingPriceConfig) => void
  }) => {
    const { buttonMoreClick } = handlers

    const allColumns = [
      { prop: 'id', label: '编号', width: 80, fixed: 'left' as const, visible: true },
      {
        prop: 'configName',
        label: '配置名称',
        minWidth: 150,
        fixed: 'left' as const,
        visible: true,
        showOverflowTooltip: true
      },
      {
        prop: 'description',
        label: '配置描述',
        minWidth: 180,
        visible: true,
        showOverflowTooltip: true,
        formatter: (row: Record<string, any>) => {
          const configRow = row as BillingPriceConfig
          return configRow.description || '-'
        }
      },
      {
        prop: 'cpuPrice',
        label: 'CPU 单价',
        minWidth: 150,
        visible: true,
        formatter: (row: Record<string, any>) => {
          const configRow = row as BillingPriceConfig
          return formatCpuPrice(configRow.cpuPrice)
        }
      },
      {
        prop: 'memoryPrice',
        label: '内存单价',
        minWidth: 150,
        visible: true,
        formatter: (row: Record<string, any>) => {
          const configRow = row as BillingPriceConfig
          return formatMemoryPrice(configRow.memoryPrice)
        }
      },
      {
        prop: 'storagePrice',
        label: '存储单价',
        minWidth: 150,
        visible: true,
        formatter: (row: Record<string, any>) => {
          const configRow = row as BillingPriceConfig
          return formatStoragePrice(configRow.storagePrice)
        }
      },
      {
        prop: 'gpuPrice',
        label: 'GPU 单价',
        minWidth: 150,
        visible: true,
        formatter: (row: Record<string, any>) => {
          const configRow = row as BillingPriceConfig
          return formatGpuPrice(configRow.gpuPrice)
        }
      },
      {
        prop: 'podPrice',
        label: 'Pod 单价',
        minWidth: 150,
        visible: true,
        formatter: (row: Record<string, any>) => {
          const configRow = row as BillingPriceConfig
          return formatPodPrice(configRow.podPrice)
        }
      },
      {
        prop: 'managementFee',
        label: '管理费',
        minWidth: 140,
        visible: true,
        formatter: (row: Record<string, any>) => {
          const configRow = row as BillingPriceConfig
          return formatManagementFee(configRow.managementFee)
        }
      },
      {
        prop: 'isSystem',
        label: '系统内置',
        width: 100,
        visible: true,
        formatter: (row: Record<string, any>) => {
          const configRow = row as BillingPriceConfig
          return h(
            ElTag,
            { type: configRow.isSystem === 1 ? 'warning' : 'info', size: 'small' },
            () => (configRow.isSystem === 1 ? '是' : '否')
          )
        }
      },
      {
        prop: 'createdBy',
        label: '创建人',
        width: 120,
        visible: false,
        formatter: (row: Record<string, any>) => {
          const configRow = row as BillingPriceConfig
          return configRow.createdBy || '-'
        }
      },
      {
        prop: 'createdAt',
        label: '创建时间',
        width: 180,
        sortable: true,
        visible: true,
        formatter: (row: Record<string, any>) =>
          formatTimestamp((row as BillingPriceConfig).createdAt)
      },
      {
        prop: 'updatedAt',
        label: '更新时间',
        width: 180,
        sortable: true,
        visible: false,
        formatter: (row: Record<string, any>) =>
          formatTimestamp((row as BillingPriceConfig).updatedAt)
      },
      {
        prop: 'operation',
        label: '操作',
        width: 100,
        fixed: 'right' as const,
        visible: true,
        formatter: (row: Record<string, any>): VNode => {
          const configRow = row as BillingPriceConfig
          const isDeleting = deleteLoadingMap.value[configRow.id] || false
          const isSystem = configRow.isSystem === 1

          const menuItems: ButtonMoreItem[] = [
            {
              key: 'edit',
              label: isSystem ? '系统内置(不可编辑)' : '编辑配置',
              icon: 'lucide:edit',
              color: isSystem ? '#c0c4cc' : undefined,
              disabled: isSystem || isDeleting
            },
            {
              key: 'delete',
              label: isSystem ? '系统内置(不可删除)' : isDeleting ? '删除中...' : '删除配置',
              icon: 'lucide:trash-2',
              color: isSystem ? '#c0c4cc' : '#f56c6c',
              disabled: isSystem || isDeleting
            }
          ]

          return h('div', {}, [
            h(ArtButtonMore, {
              list: menuItems,
              onClick: (item: ButtonMoreItem) => {
                if (!isDeleting) {
                  buttonMoreClick(item, configRow)
                }
              }
            })
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
    showSearchBar,
    deleteLoadingMap,
    createTableColumns
  }
}
