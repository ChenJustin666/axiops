<template>
  <div class="devops-jenkins-credential-page art-full-height">
    <div class="page-context">
      <div>
        <span class="context-label">当前 DevOps 项目</span>
        <strong :title="currentProjectLabel">{{ currentProjectLabel }}</strong>
      </div>
      <div class="jenkins-select">
        <span class="context-label">Jenkins 渠道</span>
        <ElSelect
          v-model="selectedBuildBindingId"
          filterable
          placeholder="请选择 Jenkins"
          style="width: 260px"
          @visible-change="handleBuildBindingVisibleChange"
        >
          <ElOption
            v-for="item in buildBindings"
            :key="item.id"
            :label="`${item.channelName}（${item.channelCode}）`"
            :value="item.id"
          />
        </ElSelect>
      </div>
      <ElTag type="info" size="small">仅管理平台同步到 Jenkins 的凭证副本</ElTag>
    </div>

    <ElCard class="art-table-card" shadow="never">
      <ArtTableHeader
        v-model:columns="columns"
        :loading="loading"
        :showZebra="true"
        :showBorder="true"
        :showHeaderBackground="true"
        :fullClass="'art-page-view'"
        :layout="'refresh,size,fullscreen,columns,settings'"
        @refresh="getData"
      />

      <ElEmpty v-if="!selectedProjectId" description="请选择顶部 DevOps 项目" />
      <ElEmpty v-else-if="!selectedBuildBindingId" description="当前项目未绑定 Jenkins 构建渠道" />
      <ArtTable
        v-else
        :loading="loading"
        :data="tableData"
        :columns="columns"
        :pagination="pagination"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      />
    </ElCard>

    <ElDrawer v-model="referenceDrawerVisible" title="引用流水线" size="640px">
      <div class="reference-drawer">
        <div class="reference-summary">
          <strong :title="currentRecord?.jenkinsCredentialId || ''">
            {{ currentRecord?.jenkinsCredentialId || '-' }}
          </strong>
          <span>{{ currentRecord?.credentialName || '-' }}</span>
        </div>
        <ElTable v-loading="referenceLoading" :data="referenceData" border height="100%">
          <ElTableColumn prop="pipelineName" label="流水线" min-width="160" show-overflow-tooltip />
          <ElTableColumn prop="pipelineCode" label="编码" min-width="150" show-overflow-tooltip />
          <ElTableColumn prop="paramCode" label="参数编码" min-width="140" show-overflow-tooltip />
          <ElTableColumn prop="paramName" label="参数名称" min-width="140" show-overflow-tooltip />
          <ElTableColumn prop="referenceType" label="引用类型" width="130" />
        </ElTable>
        <ElEmpty
          v-if="!referenceLoading && referenceData.length === 0"
          description="暂无引用流水线"
        />
      </div>
    </ElDrawer>
  </div>
</template>

<script setup lang="ts">
  import { computed, h, onActivated, onMounted, ref, watch } from 'vue'
  import { storeToRefs } from 'pinia'
  import { ElButton, ElMessage, ElMessageBox, ElTag } from 'element-plus'
  import { formatTimestamp } from '@/utils/format'
  import { useDevopsProjectStore } from '@/store/modules/devopsProject'
  import {
    listDevopsProjectChannelBindingApi,
    type DevopsProjectChannelBinding
  } from '@/api/devops/project'
  import {
    deleteJenkinsCredentialSyncApi,
    listJenkinsCredentialReferencesApi,
    listJenkinsCredentialSyncApi,
    type JenkinsCredentialReference,
    type JenkinsCredentialSyncRecord
  } from '@/api/devops/pipeline'

  defineOptions({ name: 'DevOpsJenkinsCredential' })

  const devopsProjectStore = useDevopsProjectStore()
  const { selectedProject, selectedProjectId } = storeToRefs(devopsProjectStore)

  const loading = ref(false)
  const referenceLoading = ref(false)
  const referenceDrawerVisible = ref(false)
  const buildBindings = ref<DevopsProjectChannelBinding[]>([])
  const selectedBuildBindingId = ref('')
  const tableData = ref<JenkinsCredentialSyncRecord[]>([])
  const referenceData = ref<JenkinsCredentialReference[]>([])
  const currentRecord = ref<JenkinsCredentialSyncRecord | null>(null)

  const pagination = ref({
    current: 1,
    size: 20,
    total: 0
  })

  const currentProjectLabel = computed(() => {
    if (!selectedProject.value) return '未选择项目'
    return `${selectedProject.value.name}（${selectedProject.value.code}）`
  })

  const columns = computed(() => [
    { prop: 'credentialName', label: '平台凭证', minWidth: 160, fixed: 'left' as const },
    { prop: 'credentialCode', label: '凭证编码', minWidth: 160, showOverflowTooltip: true },
    {
      prop: 'credentialType',
      label: '凭证类型',
      width: 130,
      formatter: (row: JenkinsCredentialSyncRecord) => credentialTypeLabel(row.credentialType)
    },
    {
      prop: 'jenkinsCredentialId',
      label: 'Jenkins credentialId',
      minWidth: 240,
      showOverflowTooltip: true
    },
    {
      prop: 'syncStatus',
      label: '同步状态',
      width: 120,
      formatter: (row: JenkinsCredentialSyncRecord) =>
        h(
          ElTag,
          { type: syncStatusType(row.syncStatus), size: 'small', title: row.syncMessage },
          () => syncStatusLabel(row.syncStatus)
        )
    },
    {
      prop: 'referenceCount',
      label: '引用数',
      width: 100,
      formatter: (row: JenkinsCredentialSyncRecord) =>
        h(
          ElButton,
          {
            link: true,
            type: row.referenceCount > 0 ? 'primary' : 'info',
            onClick: () => openReferences(row)
          },
          () => String(row.referenceCount || 0)
        )
    },
    {
      prop: 'lastSyncAt',
      label: '最后同步',
      width: 180,
      formatter: (row: JenkinsCredentialSyncRecord) => formatTimestamp(row.lastSyncAt)
    },
    {
      prop: 'operation',
      label: '操作',
      width: 160,
      fixed: 'right' as const,
      formatter: (row: JenkinsCredentialSyncRecord) =>
        h('div', { class: 'table-actions' }, [
          h(
            ElButton,
            { link: true, type: 'primary', onClick: () => openReferences(row) },
            () => '查看引用'
          ),
          h(
            ElButton,
            {
              link: true,
              type: 'danger',
              disabled: row.referenceCount > 0 || row.syncStatus === 'deleted',
              onClick: () => handleDelete(row)
            },
            () => '删除副本'
          )
        ])
    }
  ])

  const credentialTypeLabel = (type: string) => {
    const map: Record<string, string> = {
      username_password: '用户名密码',
      token: 'Token',
      secret_text: 'Secret Text',
      ssh_key: 'SSH Key'
    }
    return map[type] || type || '-'
  }

  const syncStatusLabel = (status?: string) => {
    if (status === 'synced') return '已同步'
    if (status === 'failed') return '同步失败'
    if (status === 'deleted') return '已删除'
    if (status === 'pending') return '待同步'
    return '未知'
  }

  const syncStatusType = (status?: string) => {
    if (status === 'synced') return 'success'
    if (status === 'failed') return 'danger'
    if (status === 'deleted') return 'info'
    return 'warning'
  }

  const updateSelectedBuildBinding = (nextId: string) => {
    if (selectedBuildBindingId.value === nextId) return false
    selectedBuildBindingId.value = nextId
    return true
  }

  const fetchBuildBindings = async () => {
    if (!selectedProjectId.value) {
      buildBindings.value = []
      return updateSelectedBuildBinding('')
    }
    const res = await listDevopsProjectChannelBindingApi({
      page: 1,
      pageSize: 200,
      projectId: selectedProjectId.value,
      channelType: 'jenkins',
      status: 1
    })
    buildBindings.value = res.items || []
    const current = buildBindings.value.find((item) => item.id === selectedBuildBindingId.value)
    if (current) return false
    const nextId =
      buildBindings.value.find((item) => item.isDefault)?.id || buildBindings.value[0]?.id || ''
    return updateSelectedBuildBinding(nextId)
  }

  const getData = async () => {
    if (!selectedProjectId.value) {
      tableData.value = []
      pagination.value.total = 0
      return
    }
    loading.value = true
    try {
      const bindingChanged = await fetchBuildBindings()
      if (!selectedBuildBindingId.value) {
        tableData.value = []
        pagination.value.total = 0
        return
      }
      if (bindingChanged) return
      const res = await listJenkinsCredentialSyncApi({
        projectId: selectedProjectId.value,
        buildChannelBindingId: selectedBuildBindingId.value,
        page: pagination.value.current,
        pageSize: pagination.value.size
      })
      tableData.value = res.items || []
      pagination.value.total = res.total || 0
    } finally {
      loading.value = false
    }
  }

  const handleSizeChange = (size: number) => {
    pagination.value.size = size
    pagination.value.current = 1
    getData()
  }

  const handleCurrentChange = (current: number) => {
    pagination.value.current = current
    getData()
  }

  const handleBuildBindingVisibleChange = async (visible: boolean) => {
    if (visible) await fetchBuildBindings()
  }

  const openReferences = async (row: JenkinsCredentialSyncRecord) => {
    if (!selectedProjectId.value || !selectedBuildBindingId.value) {
      ElMessage.warning('请选择项目和 Jenkins 渠道')
      return
    }
    currentRecord.value = row
    referenceDrawerVisible.value = true
    referenceLoading.value = true
    try {
      const res = await listJenkinsCredentialReferencesApi({
        projectId: selectedProjectId.value,
        buildChannelBindingId: selectedBuildBindingId.value,
        credentialId: row.credentialId
      })
      referenceData.value = res.items || []
    } finally {
      referenceLoading.value = false
    }
  }

  const handleDelete = async (row: JenkinsCredentialSyncRecord) => {
    if (!selectedProjectId.value || !selectedBuildBindingId.value) {
      ElMessage.warning('请选择项目和 Jenkins 渠道')
      return
    }
    if (row.referenceCount > 0) {
      ElMessage.warning('该 Jenkins 凭证仍被流水线引用，不能删除')
      await openReferences(row)
      return
    }
    await ElMessageBox.confirm(
      `确认删除 Jenkins 凭证副本 ${row.jenkinsCredentialId}？`,
      '删除确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    await deleteJenkinsCredentialSyncApi({
      projectId: selectedProjectId.value,
      buildChannelBindingId: selectedBuildBindingId.value,
      credentialId: row.credentialId
    })
    ElMessage.success('删除成功')
    await getData()
  }

  const refreshPage = async () => {
    if (!devopsProjectStore.initialized) {
      await devopsProjectStore.fetchProjects()
    }
    await getData()
  }

  watch(selectedProjectId, () => {
    selectedBuildBindingId.value = ''
    pagination.value.current = 1
    getData()
  })

  watch(selectedBuildBindingId, () => {
    pagination.value.current = 1
    getData()
  })

  let activatedOnce = false
  onMounted(refreshPage)
  onActivated(() => {
    if (!activatedOnce) {
      activatedOnce = true
      return
    }
    refreshPage()
  })
</script>

<style lang="scss" scoped>
  .devops-jenkins-credential-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .page-context {
    display: flex;
    flex: 0 0 auto;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    padding: 12px 14px;
    margin-bottom: 12px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .page-context strong,
  .context-label {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .page-context strong {
    margin-top: 4px;
    color: var(--el-text-color-primary);
  }

  .context-label {
    margin-bottom: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .jenkins-select {
    min-width: 260px;
  }

  .art-table-card {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
  }

  :deep(.art-table-card > .el-card__body) {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
  }

  .table-actions {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .reference-drawer {
    display: flex;
    flex-direction: column;
    gap: 12px;
    height: 100%;
    min-height: 0;
  }

  .reference-summary {
    padding: 12px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .reference-summary strong,
  .reference-summary span {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .reference-summary span {
    margin-top: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
</style>
