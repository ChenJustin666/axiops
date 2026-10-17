import { ref, h } from 'vue'
import { ElTag } from 'element-plus'
import { Shield } from 'lucide-vue-next'
import ArtButtonMore from '../../../../components/core/forms/art-button-more/index.vue'
import { ButtonMoreItem } from '../../../../components/core/forms/art-button-more/index.vue'
import type { NetworkPolicyListItem } from '../../api/workload/networkpolicy'
import { formatK8sAge } from '../../core/utils/format'

export function useNetworkPolicyTableConfig() {
  const loading = ref(true)
  const showSearchBar = ref(false)
  const deleteLoadingMap = ref<Record<string, boolean>>({})
  const downloadLoadingMap = ref<Record<string, boolean>>({})

  const createTableColumns = (handlers: {
    handleView: (row: NetworkPolicyListItem) => void
    handleEdit: (row: NetworkPolicyListItem) => void
    handleDownloadYaml: (row: NetworkPolicyListItem) => void
    handleDelete: (row: NetworkPolicyListItem) => void
  }) => {
    const { handleView, handleEdit, handleDownloadYaml, handleDelete } = handlers

    const allColumns = [
      {
        prop: 'name',
        label: '名称',
        minWidth: 220,
        fixed: 'left' as const,
        showOverflowTooltip: true,
        visible: true,
        formatter: (row: Record<string, any>) => {
          const npRow = row as NetworkPolicyListItem
          return h('div', { style: 'display: flex; align-items: center; gap: 8px;' }, [
            h(Shield, { size: 16, style: 'color: #409eff; flex-shrink: 0;' }),
            h('span', { style: 'font-weight: 500;' }, npRow.name)
          ])
        }
      },
      {
        prop: 'namespace',
        label: '命名空间',
        minWidth: 140,
        showOverflowTooltip: true,
        visible: true
      },
      {
        prop: 'labels',
        label: '标签',
        minWidth: 200,
        showOverflowTooltip: true,
        visible: true,
        formatter: (row: Record<string, any>) => {
          const npRow = row as NetworkPolicyListItem
          const labels = npRow.labels || {}

          if (Object.keys(labels).length === 0) {
            return h('span', { style: 'color: #909399;' }, '-')
          }

          const labelEntries = Object.entries(labels).slice(0, 2)
          return h('div', { style: 'display: flex; flex-wrap: wrap; gap: 4px;' }, [
            ...labelEntries.map(([key, value]) =>
              h(ElTag, { size: 'small', type: 'info' }, () => `${key}=${value}`)
            ),
            Object.keys(labels).length > 2
              ? h(
                  ElTag,
                  { size: 'small', type: 'info' },
                  () => `+${Object.keys(labels).length - 2}`
                )
              : null
          ])
        }
      },
      {
        prop: 'age',
        label: '运行时长',
        width: 140,
        align: 'center' as const,
        visible: true,
        formatter: (row: Record<string, any>) => {
          const npRow = row as NetworkPolicyListItem
          return formatK8sAge(npRow.age)
        }
      },
      {
        prop: 'creationTimestamp',
        label: '创建时间',
        minWidth: 160,
        visible: false,
        formatter: (row: Record<string, any>) => {
          const npRow = row as NetworkPolicyListItem
          return npRow.creationTimestamp
            ? new Date(npRow.creationTimestamp * 1000).toLocaleString('zh-CN')
            : '-'
        }
      },
      {
        prop: 'operation',
        label: '操作',
        width: 80,
        fixed: 'right' as const,
        align: 'center' as const,
        visible: true,
        formatter: (row: Record<string, any>) => {
          const npRow = row as NetworkPolicyListItem
          const isDeleting = deleteLoadingMap.value[npRow.name] || false
          const isDownloading = downloadLoadingMap.value[npRow.name] || false
          const isAnyLoading = isDeleting || isDownloading

          return h(ArtButtonMore, {
            list: [
              { key: 'view', label: '查看详情', icon: 'lucide:eye', disabled: isAnyLoading },
              { key: 'edit', label: '编辑', icon: 'lucide:edit', disabled: isAnyLoading },
              {
                key: 'download',
                label: isDownloading ? '下载中...' : '下载 YAML',
                icon: 'lucide:download',
                disabled: isAnyLoading
              },
              {
                key: 'delete',
                label: isDeleting ? '删除中...' : '删除',
                icon: 'lucide:trash-2',
                color: '#f56c6c',
                disabled: isAnyLoading
              }
            ] as ButtonMoreItem[],
            onClick: (item: ButtonMoreItem) => {
              if (!isAnyLoading) {
                switch (item.key) {
                  case 'view':
                    handleView(npRow)
                    break
                  case 'edit':
                    handleEdit(npRow)
                    break
                  case 'download':
                    handleDownloadYaml(npRow)
                    break
                  case 'delete':
                    handleDelete(npRow)
                    break
                }
              }
            }
          })
        }
      }
    ]

    const columns = ref(allColumns)

    return {
      columns
    }
  }

  return {
    loading,
    showSearchBar,
    deleteLoadingMap,
    downloadLoadingMap,
    createTableColumns
  }
}
