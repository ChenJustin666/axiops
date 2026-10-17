import { ref, h } from 'vue'
import type { VNode } from 'vue'
import { ElTag, ElButton, ElTooltip } from 'element-plus'

interface Project {
  id: number
  name: string
  uuid: string
  description: string
  isSystem: number
  createdAt: number
}

interface TableConfigHandlers {
  handleEditProject: (row: Project) => void
  handleDeleteProject: (id: number) => void
  handleProjectMembers: (row: Project) => void
  handleProjectPlatforms: (row: Project) => void
}

export function useProjectTableConfig() {
  const loading = ref(false)
  const showSearchBar = ref(true)
  const deleteLoadingMap = ref<Record<number, boolean>>({})
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
    const { handleEditProject, handleDeleteProject, handleProjectMembers, handleProjectPlatforms } =
      handlers

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
        formatter: (row: Record<string, any>) => {
          return (row as Project).uuid || '-'
        }
      },
      {
        prop: 'description',
        label: '描述',
        minWidth: 200,
        showOverflowTooltip: true,
        visible: true,
        formatter: (row: Record<string, any>) => {
          return (row as Project).description || '-'
        }
      },
      {
        prop: 'createdAt',
        label: '创建时间',
        width: 180,
        visible: true,
        formatter: (row: Record<string, any>) => {
          return formatTimestamp((row as Project).createdAt)
        }
      },
      {
        prop: 'operation',
        label: '操作',
        width: 280,
        fixed: 'right' as const,
        visible: true,
        formatter: (row: Record<string, any>): VNode => {
          const project = row as Project
          const isDeleting = deleteLoadingMap.value[project.id] || false
          const isSystemProject = project.isSystem === 1
          const isAnyLoading = isDeleting || submitLoading.value

          return h('div', { class: 'table-operation-buttons' }, [
            h(
              ElButton,
              {
                type: 'primary',
                size: 'small',
                disabled: isAnyLoading || isSystemProject,
                onClick: () => handleProjectMembers(project)
              },
              () => '成员'
            ),
            h(
              ElButton,
              {
                type: 'success',
                size: 'small',
                disabled: isAnyLoading || isSystemProject,
                onClick: () => handleProjectPlatforms(project),
                style: 'margin-left: 8px;'
              },
              () => '平台'
            ),
            h(
              ElTooltip,
              {
                content: isSystemProject ? '系统项目不可编辑' : '编辑项目',
                placement: 'top',
                disabled: !isSystemProject
              },
              () =>
                h(
                  ElButton,
                  {
                    type: 'warning',
                    size: 'small',
                    disabled: isAnyLoading || isSystemProject,
                    onClick: () => handleEditProject(project),
                    style: 'margin-left: 8px;'
                  },
                  () => '编辑'
                )
            ),
            h(
              ElTooltip,
              {
                content: isSystemProject ? '系统项目不可删除' : '删除项目',
                placement: 'top',
                disabled: !isSystemProject
              },
              () =>
                h(
                  ElButton,
                  {
                    type: 'danger',
                    size: 'small',
                    loading: isDeleting,
                    disabled: isAnyLoading || isSystemProject,
                    onClick: () => handleDeleteProject(project.id),
                    style: 'margin-left: 8px;'
                  },
                  () => (isDeleting ? '删除中' : '删除')
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
    deleteLoadingMap,
    submitLoading,
    createTableColumns,
    formatTimestamp
  }
}
