<template>
  <div class="log-config-page">
    <ElCard shadow="never" class="panel-card config-toolbar">
      <div class="toolbar-title">
        <strong>日志后端</strong>
        <span>{{ currentClusterName }}</span>
      </div>
      <div class="panel-actions">
        <ElSelect
          v-model="selectedClusterId"
          placeholder="请选择集群"
          filterable
          clearable
          class="cluster-select"
          :loading="clusterLoading"
          @change="handleClusterChange"
        >
          <ElOption v-for="item in clusters" :key="item.id" :label="item.name" :value="item.id" />
        </ElSelect>
        <ElButton :icon="Refresh" @click="refreshData" :loading="loading">刷新</ElButton>
      </div>
    </ElCard>

    <div class="summary-grid">
      <div class="summary-card">
        <div class="label">默认后端</div>
        <div class="value">{{ config.defaultBackend || '-' }}</div>
      </div>
      <div class="summary-card">
        <div class="label">Loki 可用</div>
        <div class="value">{{ modeOptions.loki ? '是' : '否' }}</div>
      </div>
      <div class="summary-card">
        <div class="label">ES 可用</div>
        <div class="value">{{ modeOptions.elasticsearch ? '是' : '否' }}</div>
      </div>
      <div class="summary-card">
        <div class="label">同步健康</div>
        <div class="value">{{ syncStatus.healthStatus || 'unknown' }}</div>
      </div>
    </div>

    <ElCard shadow="never">
      <template #header>
        <div class="card-header">规则同步摘要</div>
      </template>
      <ElDescriptions :column="3" border>
        <ElDescriptionsItem label="规则总数">{{ syncStatus.totalRules }}</ElDescriptionsItem>
        <ElDescriptionsItem label="启用规则">{{ syncStatus.enabledRules }}</ElDescriptionsItem>
        <ElDescriptionsItem label="禁用规则">{{ syncStatus.disabledRules }}</ElDescriptionsItem>
        <ElDescriptionsItem label="同步成功">{{ syncStatus.successRules }}</ElDescriptionsItem>
        <ElDescriptionsItem label="同步失败">{{ syncStatus.failedRules }}</ElDescriptionsItem>
        <ElDescriptionsItem label="待处理">{{ syncStatus.pendingRules }}</ElDescriptionsItem>
        <ElDescriptionsItem label="最近错误" :span="2">
          {{ syncStatus.lastSyncError || '-' }}
        </ElDescriptionsItem>
        <ElDescriptionsItem label="最近同步时间">
          {{ formatTime(syncStatus.lastSyncAt) }}
        </ElDescriptionsItem>
      </ElDescriptions>
    </ElCard>

    <ElCard shadow="never">
      <ArtTableHeader
        v-model:columns="backendColumns"
        :loading="loading"
        :showZebra="true"
        :showBorder="true"
        :showHeaderBackground="true"
        :fullClass="'art-page-view'"
        :layout="'refresh,size,fullscreen,columns,settings'"
        @refresh="refreshData"
      >
        <template #left>
          <span class="card-header">后端实例</span>
        </template>
      </ArtTableHeader>
      <ArtTable
        :loading="loading"
        :data="config.backends"
        :columns="visibleBackendColumns"
        :show-table-header="false"
      >
        <template #address="{ row }">{{ row.appUrl }}:{{ row.port }}</template>
        <template #isDefault="{ row }">
          <ElTag :type="row.isDefault ? 'success' : 'info'">{{
            row.isDefault ? '是' : '否'
          }}</ElTag>
        </template>
        <template #auth="{ row }">
          {{ row.authEnabled ? row.authType || '已启用' : '无' }}
        </template>
        <template #tls="{ row }">
          <ElTag :type="row.tlsEnabled ? 'warning' : 'info'">{{
            row.tlsEnabled ? '开启' : '关闭'
          }}</ElTag>
        </template>
        <template #status="{ row }">
          <ElTag :type="row.status === 1 ? 'success' : 'danger'">{{
            row.status === 1 ? '正常' : '异常'
          }}</ElTag>
        </template>
        <template #updatedAt="{ row }">{{ formatTime(row.updatedAt) }}</template>
      </ArtTable>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref } from 'vue'
  import dayjs from 'dayjs'
  import { Refresh } from '@element-plus/icons-vue'
  import {
    getLogClusterConfigApi,
    getLogClusterSyncStatusApi,
    getLogQueryModeOptionsApi,
    type LogClusterConfigResponse
  } from '@/api/manager/log'
  import { useLogClusters } from '../shared/useLogClusters'

  defineOptions({ name: 'ClusterLogConfigPage' })

  const {
    clusters,
    loading: clusterLoading,
    selectedClusterId,
    selectedClusterUuid,
    loadClusters,
    setCluster
  } = useLogClusters('cluster-log-config-cluster')

  const loading = ref(false)
  const config = reactive<LogClusterConfigResponse>({
    clusterUuid: '',
    defaultBackend: '',
    backends: []
  })
  const syncStatus = reactive({
    clusterUuid: '',
    totalRules: 0,
    enabledRules: 0,
    successRules: 0,
    failedRules: 0,
    pendingRules: 0,
    disabledRules: 0,
    healthStatus: 'unknown',
    lastSyncError: '',
    lastSyncAt: 0
  })
  const modeOptions = reactive({
    loki: false,
    elasticsearch: false
  })
  const backendColumns = ref([
    { prop: 'appName', label: '名称', minWidth: 220 },
    { prop: 'appCode', label: '类型', width: 140 },
    { prop: 'protocol', label: '协议', width: 100 },
    { prop: 'address', label: '地址', minWidth: 220, useSlot: true },
    { prop: 'isDefault', label: '默认', width: 90, useSlot: true },
    { prop: 'auth', label: '认证', width: 120, useSlot: true },
    { prop: 'tls', label: 'TLS', width: 90, useSlot: true },
    { prop: 'status', label: '状态', width: 100, useSlot: true },
    { prop: 'updatedAt', label: '更新时间', width: 180, useSlot: true }
  ])
  const visibleBackendColumns = computed(() =>
    backendColumns.value.filter((column: any) => column.visible !== false)
  )
  const currentClusterName = computed(() => {
    const current = clusters.value.find((item) => item.id === selectedClusterId.value)
    return current?.name || '请选择集群'
  })

  async function refreshData() {
    if (!selectedClusterUuid.value) return
    loading.value = true
    try {
      const [configResp, syncResp, modeResp] = await Promise.all([
        getLogClusterConfigApi(selectedClusterUuid.value),
        getLogClusterSyncStatusApi(selectedClusterUuid.value),
        getLogQueryModeOptionsApi(selectedClusterUuid.value)
      ])
      Object.assign(config, configResp)
      Object.assign(syncStatus, syncResp)
      Object.assign(modeOptions, modeResp)
    } finally {
      loading.value = false
    }
  }

  function handleClusterChange(value: number | null) {
    setCluster(value)
    refreshData()
  }

  function formatTime(value?: number) {
    return value ? dayjs(value * 1000).format('YYYY-MM-DD HH:mm:ss') : '-'
  }

  onMounted(async () => {
    await loadClusters()
    if (selectedClusterUuid.value) {
      await refreshData()
    }
  })
</script>

<style scoped>
  .log-config-page {
    min-height: 100%;
    padding: 0;
    background: var(--el-fill-color-extra-light);
  }

  .panel-card,
  .summary-card {
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    box-shadow: 0 1px 2px rgb(15 23 42 / 3%);
  }

  .config-toolbar {
    margin-bottom: 16px;
  }

  .config-toolbar :deep(.el-card__body) {
    display: flex;
    gap: 16px;
    align-items: center;
    justify-content: space-between;
    padding: 16px;
  }

  .toolbar-title {
    min-width: 0;
  }

  .toolbar-title strong,
  .toolbar-title span {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .toolbar-title strong {
    font-size: 16px;
    color: var(--el-text-color-primary);
  }

  .toolbar-title span {
    margin-top: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .panel-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    align-items: center;
    justify-content: flex-end;
  }

  .panel-actions :deep(.el-button + .el-button) {
    margin-left: 0;
  }

  .cluster-select {
    width: 260px;
  }

  .summary-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 12px;
    margin-bottom: 16px;
  }

  .summary-card {
    min-width: 0;
    padding: 14px;
  }

  .summary-card .label {
    overflow: hidden;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .summary-card .value {
    margin-top: 8px;
    overflow: hidden;
    font-size: 24px;
    font-weight: 700;
    line-height: 1;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .card-header {
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  :deep(.el-card) {
    margin-bottom: 16px;
    border-radius: 8px;
  }

  @media (width <= 1200px) {
    .config-toolbar :deep(.el-card__body) {
      flex-direction: column;
      align-items: stretch;
    }

    .panel-actions {
      justify-content: flex-start;
    }

    .cluster-select {
      width: 100%;
    }

    .summary-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }

  @media (width <= 768px) {
    .summary-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
