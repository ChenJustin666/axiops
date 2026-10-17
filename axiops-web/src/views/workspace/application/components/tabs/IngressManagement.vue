<template>
  <div class="ingress-management-modern">
    <!-- 表格头部 -->
    <ArtTableHeader
      v-model:columns="columns"
      :loading="loading"
      :showZebra="true"
      :showBorder="true"
      :showHeaderBackground="true"
      :fullClass="'ingress-management-modern'"
      :layout="'refresh,size,fullscreen,columns,settings'"
      @refresh="handleRefresh"
    >
      <template #left>
        <ElSpace wrap>
          <ElButton type="primary" :icon="Plus" @click="handleCreate"> 创建 Ingress </ElButton>
        </ElSpace>
      </template>
    </ArtTableHeader>

    <!-- 数据表格 -->
    <ArtTable :loading="loading" :data="ingresses" :columns="columns" />

    <ApplicationIngressDialog
      v-if="dialogVisible"
      :visible="dialogVisible"
      :dialog-type="dialogType"
      :ingress-data="currentIngress"
      :workspace="workspace"
      :application="application"
      @close="handleDialogClose"
      @success="handleSuccess"
    />

    <!-- 查看详情对话框 -->
    <IngressDetailDialog
      v-if="detailVisible"
      :visible="detailVisible"
      :ingress-name="currentIngressName"
      :workspace="workspace"
      @close="detailVisible = false"
    />

    <!-- YAML 查看弹窗 -->
    <ElDialog
      v-model="yamlViewVisible"
      :title="`查看 Ingress YAML - ${currentIngressName}`"
      width="900px"
      top="5vh"
      :close-on-click-modal="false"
      class="yaml-view-dialog"
    >
      <YamlEditorPro
        v-model="currentYamlContent"
        :readonly="true"
        :height="'calc(100vh - 300px)'"
        :show-toolbar="true"
        :show-line-numbers="true"
        :show-status-bar="true"
        :filename="`${currentIngressName}.yaml`"
      />
      <template #footer>
        <ElButton @click="yamlViewVisible = false" size="default">关闭</ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, h } from 'vue'
  import type { VNode } from 'vue'
  import {
    ElMessage,
    ElMessageBox,
    ElTag,
    ElDropdown,
    ElDropdownMenu,
    ElDropdownItem,
    ElButton
  } from 'element-plus'
  import {
    Plus,
    Globe,
    Eye,
    Edit,
    Trash2,
    Download,
    MoreVertical,
    Clock,
    AlertCircle,
    CheckCircle,
    FileCode
  } from 'lucide-vue-next'
  import {
    getApplicationIngressListApi,
    deleteIngressApi,
    getIngressYamlApi,
    type IngressListItem,
    type OnecProjectApplication,
    type ProjectCluster,
    type ProjectWorkspace
  } from '@/api'
  import YamlEditorPro from '@/components/yaml-editor-pro/index.vue'
  import ApplicationIngressDialog from './components/ApplicationIngressDialog.vue'
  import IngressDetailDialog from '../../../core/ingress/components/IngressDetailDialog.vue'

  interface Props {
    application: OnecProjectApplication
    cluster: ProjectCluster | null
    workspace: ProjectWorkspace | null
    refreshTrigger: number
  }

  const props = defineProps<Props>()

  const ingresses = ref<IngressListItem[]>([])
  const loading = ref(false)
  const dialogVisible = ref(false)
  const detailVisible = ref(false)
  const yamlViewVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const currentIngress = ref<IngressListItem | undefined>(undefined)
  const currentIngressName = ref('')
  const currentYamlContent = ref('')
  const deleteLoadingMap = ref<Record<string, boolean>>({})
  const downloadLoadingMap = ref<Record<string, boolean>>({})
  const lastIngressRequestKey = ref('')

  // 格式化时长
  const formatAge = (ageStr: string): string => {
    if (!ageStr) return '-'
    // 移除小数点后的数字，例如 33m34.682814s => 33m34s
    return ageStr.replace(/(\d+)\.(\d+)([a-z]+)/g, '$1$3')
  }

  const handleCommand = (command: string, row: IngressListItem) => {
    switch (command) {
      case 'view':
        handleView(row)
        break
      case 'edit':
        handleEdit(row)
        break
      case 'view-yaml':
        handleViewYaml(row)
        break
      case 'yaml':
        handleDownloadYaml(row)
        break
      case 'delete':
        handleDelete(row)
        break
    }
  }

  const handleView = (row: IngressListItem) => {
    currentIngressName.value = row.name
    detailVisible.value = true
  }

  const handleEdit = (row: IngressListItem) => {
    dialogType.value = 'edit'
    currentIngress.value = row
    dialogVisible.value = true
  }

  const handleViewYaml = async (row: IngressListItem) => {
    try {
      currentIngressName.value = row.name
      const yaml = await getIngressYamlApi({
        workloadId: props.workspace?.id || 0,
        name: row.name
      })
      currentYamlContent.value = yaml
      yamlViewVisible.value = true
    } catch (error) {
      console.error('获取YAML失败:', error)
      ElMessage.error('获取YAML失败')
    }
  }

  const handleDownloadYaml = async (row: IngressListItem) => {
    try {
      downloadLoadingMap.value[row.name] = true
      const yaml = await getIngressYamlApi({
        workloadId: props.workspace?.id || 0,
        name: row.name
      })

      const blob = new Blob([yaml], { type: 'text/yaml' })
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `${row.name}.yaml`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)

      ElMessage.success('下载成功')
    } catch (error) {
      console.error('下载YAML失败:', error)
    } finally {
      downloadLoadingMap.value[row.name] = false
    }
  }

  const handleDelete = async (row: IngressListItem) => {
    try {
      await ElMessageBox.confirm(
        `确定要删除 Ingress "${row.name}" 吗？此操作不可恢复！`,
        '删除确认',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      deleteLoadingMap.value[row.name] = true
      await deleteIngressApi({
        workloadId: props.workspace?.id || 0,
        name: row.name
      })

      ElMessage.success('删除成功')
      await loadIngresses()
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('删除失败:', error)
      }
    } finally {
      deleteLoadingMap.value[row.name] = false
    }
  }

  /// 创建表格列配置
  const createTableColumns = () => {
    const allColumns = [
      {
        prop: 'name',
        label: 'Ingress 名称',
        minWidth: 180,
        fixed: 'left' as const,
        visible: true,
        formatter: (row: Record<string, any>): VNode => {
          return h('span', { style: 'font-weight: 500;' }, row.name)
        }
      },
      {
        prop: 'ingressClass',
        label: 'Ingress Class',
        width: 140,
        align: 'center' as const,
        visible: true,
        formatter: (row: Record<string, any>): VNode => {
          return h(ElTag, { type: 'info', size: 'default' }, () => row.ingressClass || 'default')
        }
      },
      {
        prop: 'hosts',
        label: '主机/规则',
        minWidth: 220,
        visible: true,
        formatter: (row: Record<string, any>): VNode => {
          const hosts = row.hosts || []
          if (hosts.length === 0) {
            return h('span', { style: 'color: #909399' }, '* (所有域名)')
          }
          return h('div', { class: 'hosts-cell' }, [
            h('div', { class: 'hosts-tags' }, [
              ...hosts
                .slice(0, 2)
                .map((host: string) => h(ElTag, { type: 'success', size: 'small' }, () => host)),
              hosts.length > 2
                ? h(ElTag, { type: 'info', size: 'small' }, () => `+${hosts.length - 2}`)
                : null
            ])
          ])
        }
      },
      {
        prop: 'address',
        label: 'LoadBalancer 地址',
        width: 180,
        align: 'center' as const,
        visible: true,
        formatter: (row: Record<string, any>): VNode => {
          if (!row.address || row.address === '-') {
            return h('span', { style: 'color: #e6a23c;' }, '待分配')
          }
          return h('span', { style: 'color: #67c23a;' }, row.address)
        }
      },
      {
        prop: 'ports',
        label: '端口',
        width: 120,
        align: 'center' as const,
        visible: true,
        formatter: (row: Record<string, any>): VNode => {
          return h('span', row.ports || '-')
        }
      },
      {
        prop: 'age',
        label: '创建时长',
        width: 140,
        align: 'center' as const,
        visible: true,
        formatter: (row: Record<string, any>): VNode => {
          return h('span', formatAge(row.age))
        }
      },
      {
        prop: 'operation',
        label: '操作',
        width: 80,
        align: 'center' as const,
        fixed: 'right' as const,
        visible: true,
        formatter: (row: Record<string, any>): VNode => {
          return h('div', [
            h(
              ElDropdown,
              {
                trigger: 'click',
                onCommand: (cmd: string) => handleCommand(cmd, row)
              },
              {
                default: () =>
                  h(ElButton, { type: 'primary', link: true }, () => h(MoreVertical, { size: 16 })),
                dropdown: () =>
                  h(ElDropdownMenu, {}, () => [
                    h(ElDropdownItem, { command: 'view' }, () => [
                      h(Eye, { size: 14, style: 'margin-right: 6px' }),
                      '查看详情'
                    ]),
                    h(ElDropdownItem, { command: 'edit' }, () => [
                      h(Edit, { size: 14, style: 'margin-right: 6px' }),
                      '编辑'
                    ]),
                    h(ElDropdownItem, { command: 'view-yaml' }, () => [
                      h(FileCode, { size: 14, style: 'margin-right: 6px' }),
                      '查看 YAML'
                    ]),
                    h(ElDropdownItem, { command: 'yaml' }, () => [
                      h(Download, { size: 14, style: 'margin-right: 6px' }),
                      '下载 YAML'
                    ]),
                    h('li', { class: 'el-dropdown-menu__item--divided', role: 'separator' }),
                    h(ElDropdownItem, { command: 'delete' }, () => [
                      h(Trash2, { size: 14, style: 'margin-right: 6px; color: #f56c6c' }),
                      h('span', { style: 'color: #f56c6c' }, '删除')
                    ])
                  ])
              }
            )
          ])
        }
      }
    ]

    const columns = ref(allColumns)

    return {
      columns
    }
  }

  const { columns } = createTableColumns()

  const getIngressRequestKey = (refreshToken = props.refreshTrigger) => {
    const clusterUuid = props.cluster?.clusterUuid || props.cluster?.uuid || 'no-cluster'
    return `${props.application.id}:${props.workspace?.id || 0}:${clusterUuid}:${refreshToken}`
  }

  const loadIngresses = async (
    options: {
      force?: boolean
      requestKey?: string
    } = {}
  ) => {
    const { force = false, requestKey = getIngressRequestKey() } = options

    if (loading.value || (!force && requestKey === lastIngressRequestKey.value)) return

    if (!props.workspace || !props.application) {
      ingresses.value = []
      return
    }

    lastIngressRequestKey.value = requestKey
    loading.value = true
    try {
      const result = await getApplicationIngressListApi({
        workloadId: props.workspace.id,
        applicationId: props.application.id
      })
      ingresses.value = result || []
    } catch (error) {
      console.error('加载Ingress失败:', error)
      ingresses.value = []
    } finally {
      loading.value = false
    }
  }

  const handleCreate = () => {
    if (!props.workspace) {
      return
    }
    dialogType.value = 'add'
    currentIngress.value = undefined
    dialogVisible.value = true
  }

  const handleRefresh = async () => {
    await loadIngresses({ force: true })
  }

  const handleDialogClose = () => {
    dialogVisible.value = false
  }

  const handleSuccess = () => {
    dialogVisible.value = false
    loadIngresses()
  }

  watch(
    () => props.refreshTrigger,
    (newVal, oldVal) => {
      if (newVal > 0 && newVal !== oldVal) {
        loadIngresses({ requestKey: getIngressRequestKey(newVal) })
      }
    }
  )

  watch(
    [() => props.application.id, () => props.workspace?.id, () => props.cluster?.clusterUuid],
    (_newValues, oldValues) => {
      const hasOldValues = oldValues.some((value) => value !== undefined)
      if (hasOldValues) {
        loadIngresses({ force: true, requestKey: getIngressRequestKey() })
      }
    }
  )

  onMounted(() => {
    loadIngresses()
  })
</script>

<style lang="scss" scoped>
  .ingress-management-modern {
    padding-top: 10px;
    padding-bottom: 15px;
    margin: 2px;

    .name-cell {
      display: flex;
      align-items: center;
      gap: 6px;
      font-weight: 500;
    }

    .hosts-cell {
      .hosts-tags {
        display: flex;
        flex-wrap: wrap;
        gap: 4px;
        align-items: center;
      }
    }

    .address-pending {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 4px;
      color: #e6a23c;
      font-size: 13px;

      svg {
        flex-shrink: 0;
      }
    }

    .address-ready {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 4px;
      color: #67c23a;
      font-size: 13px;

      svg {
        flex-shrink: 0;
      }

      span {
        font-family: var(--ax-font-mono);
        font-weight: 500;
      }
    }

    .info-cell {
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

    :deep(.el-table__empty-block) {
      min-height: 400px;
      display: flex;
      align-items: center;
      justify-content: center;
    }

    :deep(.el-table__empty-text) {
      width: 100%;
    }
  }
</style>
<style lang="scss">
  /* 强制覆盖 Dialog body 的 padding */
  .service-dialog.el-dialog .el-dialog__body {
    padding: 0 !important;
    margin: 0 !important;
  }

  // YAML 查看弹窗样式
  .yaml-view-dialog.el-dialog .el-dialog__body {
    padding: 0 !important;
    margin: 0 !important;
  }

  .yaml-view-dialog.el-dialog .el-dialog__header {
    padding: 16px 20px !important;
    margin: 0 !important;
  }

  .yaml-view-dialog.el-dialog .el-dialog__footer {
    padding: 12px 20px !important;
    border-top: 1px solid #e4e7ed;
    margin: 0 !important;
  }
</style>
