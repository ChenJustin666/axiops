import { ref, h } from 'vue'
import type { VNode } from 'vue'
import { ElTag, ElButton, ElTooltip } from 'element-plus'

interface Project {
  id: number
  name: string
  uuid: string
  description: string
  isSystem: number
  adminCount: number
  resourceCount: number
  createdAt: number
}

interface TableConfigHandlers {
  handleAssignMember: (row: Project) => void
  handleViewDetail: (row: Project) => void
}

export function useProjectTableConfig() {
  const loading = ref(false)
  const showSearchBar = ref(true)
  const submitLoading = ref(false)

  const formatTimestamp = (timestamp: number) => {
    if (!timestamp) return '-'
    return new Date(timestamp * 1000).toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
  }

  const createTableColumns = (handlers: TableConfigHandlers) => {
    const { handleAssignMember, handleViewDetail } = handlers

    const allColumns = [
      {
        prop: 'name',
        label: '项目名称',
        minWidth: 150,
        showOverflowTooltip: true,
        visible: true,
        formatter: (row: Record<string, any>) => {
          const project = row as Project
          return h('div', { class: 'flex items-center gap-2' }, [
            h('span', { class: 'font-medium' }, project.name || '-'),
            project.isSystem === 1 && h(ElTag, { type: 'info', size: 'small' }, () => '系统')
          ])
        }
      },
      {
        prop: 'uuid',
        label: '项目标识',
        minWidth: 200,
        showOverflowTooltip: true,
        visible: true,
        formatter: (row: Record<string, any>) => (row as Project).uuid || '-'
      },
      {
        prop: 'description',
        label: '描述',
        minWidth: 200,
        showOverflowTooltip: true,
        visible: true,
        formatter: (row: Record<string, any>) => (row as Project).description || '-'
      },
      {
        prop: 'adminCount',
        label: '管理员',
        width: 100,
        align: 'center' as const,
        visible: true,
        formatter: (row: Record<string, any>) => {
          const count = (row as Project).adminCount || 0
          return h('div', { class: 'flex items-center justify-center gap-1' }, [
            h('i', {
              class: 'iconfont-sys iconsys-user',
              style: 'font-size: 14px; color: #606266;'
            }),
            h('span', count)
          ])
        }
      },
      {
        prop: 'resourceCount',
        label: '资源池',
        width: 100,
        align: 'center' as const,
        visible: true,
        formatter: (row: Record<string, any>) => {
          const count = (row as Project).resourceCount || 0
          return h(
            ElTag,
            { type: count > 0 ? 'success' : 'info', size: 'small' },
            () => count + ' 个'
          )
        }
      },
      {
        prop: 'createdAt',
        label: '创建时间',
        width: 180,
        visible: true,
        formatter: (row: Record<string, any>) => formatTimestamp((row as Project).createdAt)
      },
      {
        prop: 'operation',
        label: '操作',
        width: 180,
        fixed: 'right' as const,
        visible: true,
        formatter: (row: Record<string, any>): VNode => {
          const project = row as Project
          const isAnyLoading = submitLoading.value

          return h('div', { class: 'table-operation-buttons' }, [
            h(ElTooltip, { content: '分配成员', placement: 'top' }, () =>
              h(
                ElButton,
                {
                  type: 'primary',
                  size: 'small',
                  disabled: isAnyLoading,
                  onClick: () => handleAssignMember(project)
                },
                () => [
                  h('i', { class: 'iconfont-sys iconsys-user-add', style: 'margin-right: 4px;' }),
                  '成员'
                ]
              )
            ),
            h(ElTooltip, { content: '查看项目详情', placement: 'top' }, () =>
              h(
                ElButton,
                {
                  type: 'info',
                  size: 'small',
                  disabled: isAnyLoading,
                  onClick: () => handleViewDetail(project),
                  style: 'margin-left: 8px;'
                },
                () => [
                  h('i', { class: 'iconfont-sys iconsys-eye', style: 'margin-right: 4px;' }),
                  '详情'
                ]
              )
            )
          ])
        }
      }
    ]

    const columns = ref(allColumns)
    return { columns }
  }

  return {
    loading,
    showSearchBar,
    submitLoading,
    createTableColumns,
    formatTimestamp
  }
}
