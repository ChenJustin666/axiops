<template>
  <section class="environment-runtime">
    <div class="runtime-summary">
      <div class="target-context">
        <strong
          >{{ environment.name }}
          <ElTag v-if="environment.is_production" size="small" type="danger">生产</ElTag></strong
        >
        <span
          >{{ isKubernetes ? '集群' : '主机组' }}
          <b>{{ isKubernetes ? clusterName : environment.host_group_ref || '未绑定' }}</b></span
        >
        <span v-if="isKubernetes"
          >命名空间 <b>{{ environment.namespace || '未绑定' }}</b></span
        >
      </div>
    </div>
    <ElAlert
      v-if="!isKubernetes"
      title="当前为主机环境，Kubernetes 资源视图不适用。请在发布记录中查看主机部署结果。"
      type="info"
      :closable="false"
    />
    <template v-else>
      <div class="runtime-toolbar">
        <div class="resource-tabs" role="tablist" aria-label="运行资源分类">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            role="tab"
            :aria-selected="activeTab === tab.key"
            :class="{ active: activeTab === tab.key }"
            @click="activeTab = tab.key"
            >{{ tab.label }}</button
          >
        </div>
        <div class="refresh-actions">
          <small>{{
            loading
              ? '正在读取'
              : error
                ? '读取失败'
                : lastUpdated
                  ? `最近读取 ${new Date(lastUpdated).toLocaleTimeString()}`
                  : '尚未读取'
          }}</small>
          <ElSwitch v-model="autoRefresh" active-text="每 15 秒刷新" size="small" />
          <ElButton size="small" :loading="loading" @click="loadResources">刷新</ElButton>
        </div>
      </div>
      <div class="resource-filters">
        <ElRadioGroup v-if="activeTab === 'workloads'" v-model="workloadKind" size="small">
          <ElRadioButton v-for="kind in kinds" :key="kind" :value="kind.toLowerCase()">{{
            kind
          }}</ElRadioButton>
        </ElRadioGroup>
        <ElRadioGroup v-if="activeTab === 'config'" v-model="configKind" size="small">
          <ElRadioButton value="configmap">ConfigMap</ElRadioButton
          ><ElRadioButton value="secret">Secret</ElRadioButton>
        </ElRadioGroup>
        <ElInput
          v-model="search"
          placeholder="筛选当前列表名称"
          clearable
          size="small"
          class="resource-search"
        />
      </div>
      <p class="resource-scope"
        >{{ environment.namespace }} 中的实际资源 ·
        {{ error ? '状态未知' : loading ? '读取中' : `当前分类 ${resourceCount} 个`
        }}<span v-if="!loading && !error && resourceCount === 0">
          · 当前分类为空不代表整个环境没有资源</span
        ></p
      >
      <ElAlert v-if="error" type="error" :title="error" :closable="false"
        ><ElButton text @click="loadResources">重新读取</ElButton></ElAlert
      >
      <div v-else class="resource-table">
        <WorkloadTable
          v-if="activeTab === 'workloads'"
          :loading="loading"
          :kind="workloadKind"
          :rows="filtered(workloads)"
          :read-only="true"
          @view-yaml="openDetail(workloadKind, $event.name)"
          @restart="openRestart"
        />
        <PodTable
          v-else-if="activeTab === 'pods'"
          :loading="loading"
          :rows="filtered(pods)"
          :read-only="true"
          @view-yaml="openDetail('pod', $event.name)"
          @logs="openLogs"
        />
        <NetworkTables
          v-else-if="activeTab === 'network'"
          :loading="loading"
          :services="filtered(services)"
          :ingresses="filtered(ingresses)"
          @view-service-yaml="openDetail('service', $event.name)"
          @view-ingress-yaml="openDetail('ingress', $event.name)"
        />
        <StorageTable
          v-else-if="activeTab === 'storage'"
          :loading="loading"
          :rows="filtered(pvcs)"
          @view-yaml="openDetail('persistentvolumeclaim', $event.name)"
        />
        <ConfigTable
          v-else
          :loading="loading"
          :rows="filtered(configs)"
          @view-yaml="openDetail(configKind, $event.name)"
        />
      </div>
    </template>
    <WorkloadRestartDialog
      v-model="restartVisible"
      :environment="environment"
      :workload="restartWorkload"
      @submitted="restartSubmitted"
    />
    <WorkflowRunDrawer
      :visible="restartRunVisible"
      :run-id="restartRunId"
      :environments="[environment]"
      @update:visible="restartRunVisible = $event"
      @changed="loadResources"
    />
    <ElDrawer
      v-model="detailVisible"
      :title="`${environment.name} / ${selectedName}`"
      size="min(1000px, 90vw)"
      destroy-on-close
      @closed="closeDetail"
    >
      <div v-loading="detailLoading" class="resource-detail">
        <ElAlert v-if="detailError" :title="detailError" type="error" :closable="false" />
        <template v-else-if="detail">
          <ElTag size="small">{{ detail.namespace }}</ElTag>
          <ElTabs v-model="detailTab">
            <ElTabPane label="实际 YAML" name="yaml">
              <ElAlert
                v-if="detail.redacted"
                title="Secret 内容已隐藏，仅展示键名和资源信息。"
                type="info"
                :closable="false"
              />
              <pre class="runtime-output">{{ detail.yaml }}</pre>
            </ElTabPane>
            <ElTabPane label="事件" name="events">
              <ElAlert
                v-if="detail.events_error"
                :title="detail.events_error"
                type="warning"
                :closable="false"
              />
              <ElEmpty
                v-else-if="!detail.events.length"
                description="暂无事件，历史事件可能已由集群清理"
                :image-size="48"
              />
              <article v-for="(event, i) in detail.events" :key="i" class="event-row"
                ><strong>{{ event.reason }}</strong
                ><small
                  >{{ event.resource_kind ? `${event.resource_kind}/${event.resource_name} · ` : ''
                  }}{{ event.type }} · {{ event.count }} 次 ·
                  {{ new Date(event.time).toLocaleString() }}</small
                ><p>{{ event.message }}</p></article
              >
            </ElTabPane>
            <ElTabPane v-if="selectedKind === 'pod'" label="容器日志" name="logs">
              <div class="log-controls"
                ><ElSelect v-model="container" placeholder="选择容器" size="small"
                  ><ElOption
                    v-for="name in containers"
                    :key="name"
                    :value="name"
                    :label="name" /></ElSelect
                ><ElCheckbox v-model="previousLogs">上次退出</ElCheckbox
                ><ElButton size="small" :loading="logsLoading" @click="loadLogs"
                  >刷新日志</ElButton
                ></div
              >
              <small>最近 500 行，最多 512 KiB，点击刷新获取最新输出。</small>
              <ElAlert v-if="logsError" :title="logsError" type="error" :closable="false" />
              <pre v-else v-loading="logsLoading" class="runtime-output log-output">{{
                logs || '暂无日志输出'
              }}</pre>
            </ElTabPane>
          </ElTabs>
        </template>
      </div>
    </ElDrawer>
  </section>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
  import type { AxiOpsEnvironment } from '@/api/axiops/cicd'
  import {
    listEnvWorkloadsApi,
    listEnvPodsApi,
    listEnvServicesApi,
    listEnvIngressesApi,
    listEnvPVCsApi,
    listEnvConfigsApi,
    getEnvResourceDetailApi,
    getEnvPodLogsApi,
    type WorkloadKind,
    type EnvWorkloadItem,
    type EnvPodItem,
    type EnvServiceItem,
    type EnvIngressItem,
    type EnvPVCItem,
    type EnvConfigItem,
    type EnvResourceDetail
  } from '@/api/axiops/env-resource'
  import WorkloadTable from '../../environment-detail/components/WorkloadTable.vue'
  import WorkloadRestartDialog from '../../environment-detail/components/WorkloadRestartDialog.vue'
  import WorkflowRunDrawer from './WorkflowRunDrawer.vue'
  import PodTable from '../../environment-detail/components/PodTable.vue'
  import NetworkTables from '../../environment-detail/components/NetworkTables.vue'
  import StorageTable from '../../environment-detail/components/StorageTable.vue'
  import ConfigTable from '../../environment-detail/components/ConfigTable.vue'
  import * as yaml from 'js-yaml'

  const props = defineProps<{
    environment: AxiOpsEnvironment
    clusterName: string
    deliveryType: string
  }>()
  const isKubernetes = computed(() => props.environment.environment_type === 'k8s_namespace')
  const restartVisible = ref(false),
    restartWorkload = ref<EnvWorkloadItem | null>(null),
    restartRunVisible = ref(false),
    restartRunId = ref(0)
  function openRestart(row: EnvWorkloadItem) {
    if (!row.restart_allowed || row.namespace !== props.environment.namespace) return
    restartWorkload.value = row
    restartVisible.value = true
  }
  function restartSubmitted(id: number) {
    restartRunId.value = id
    restartRunVisible.value = true
    void loadResources()
  }
  const activeTab = ref('workloads'),
    workloadKind = ref<WorkloadKind>('deployment'),
    configKind = ref<'configmap' | 'secret'>('configmap')
  const tabs = [
    { key: 'workloads', label: '工作负载' },
    { key: 'pods', label: '容器组' },
    { key: 'network', label: '网络' },
    { key: 'storage', label: '存储' },
    { key: 'config', label: '配置' }
  ]
  const kinds = ['Deployment', 'StatefulSet', 'DaemonSet', 'CronJob', 'Job']
  const loading = ref(false),
    error = ref(''),
    lastUpdated = ref(0),
    search = ref(''),
    autoRefresh = ref(false)
  const workloads = ref<EnvWorkloadItem[]>([]),
    pods = ref<EnvPodItem[]>([]),
    services = ref<EnvServiceItem[]>([]),
    ingresses = ref<EnvIngressItem[]>([]),
    pvcs = ref<EnvPVCItem[]>([]),
    configs = ref<EnvConfigItem[]>([])
  let generation = 0,
    loadedKey = '',
    disposed = false,
    timer: ReturnType<typeof setInterval> | undefined
  const contextKey = () =>
    [
      props.environment.id,
      props.environment.cluster_id,
      props.environment.namespace,
      props.environment.environment_type,
      props.environment.binding_status,
      activeTab.value,
      workloadKind.value,
      configKind.value
    ].join(':')
  const filtered = <T extends { name: string }>(rows: T[]) =>
    rows.filter((row) => row.name.toLowerCase().includes(search.value.trim().toLowerCase()))
  const resourceCount = computed(() =>
    activeTab.value === 'workloads'
      ? workloads.value.length
      : activeTab.value === 'pods'
        ? pods.value.length
        : activeTab.value === 'network'
          ? services.value.length + ingresses.value.length
          : activeTab.value === 'storage'
            ? pvcs.value.length
            : configs.value.length
  )
  function clearRows() {
    workloads.value = []
    pods.value = []
    services.value = []
    ingresses.value = []
    pvcs.value = []
    configs.value = []
  }
  function resourceRows<T extends { namespace: string; name: string }>(
    value: T[],
    namespace: string
  ): T[] {
    if (
      !Array.isArray(value) ||
      value.some((row) => !row || row.namespace !== namespace || typeof row.name !== 'string')
    )
      throw Error('资源返回范围与当前环境不一致，请重新读取')
    return value
  }
  async function loadResources() {
    const current = ++generation,
      key = contextKey(),
      env = { ...props.environment }
    const tab = activeTab.value,
      kind = workloadKind.value,
      configType = configKind.value
    error.value = ''
    if (loadedKey !== key) {
      clearRows()
      lastUpdated.value = 0
      loadedKey = key
    }
    if (!isKubernetes.value) {
      loading.value = false
      return
    }
    if (!env.cluster_id || !env.namespace || env.binding_status !== 'validated') {
      error.value = '环境尚未绑定有效的集群和命名空间'
      loading.value = false
      return
    }
    loading.value = true
    const valid = () => !disposed && current === generation && key === contextKey()
    try {
      if (tab === 'workloads') {
        const rows = await listEnvWorkloadsApi(env.id, kind)
        if (valid()) workloads.value = resourceRows(rows, env.namespace)
      } else if (tab === 'pods') {
        const rows = await listEnvPodsApi(env.id)
        if (valid()) pods.value = resourceRows(rows, env.namespace)
      } else if (tab === 'network') {
        const [s, i] = await Promise.all([listEnvServicesApi(env.id), listEnvIngressesApi(env.id)])
        if (valid()) {
          services.value = resourceRows(s, env.namespace)
          ingresses.value = resourceRows(i, env.namespace)
        }
      } else if (tab === 'storage') {
        const rows = await listEnvPVCsApi(env.id)
        if (valid()) pvcs.value = resourceRows(rows, env.namespace)
      } else {
        const rows = await listEnvConfigsApi(env.id, configType)
        if (valid()) configs.value = resourceRows(rows, env.namespace)
      }
      if (valid()) lastUpdated.value = Date.now()
    } catch (e: any) {
      if (valid()) {
        error.value = e?.message || '资源读取失败，请检查集群连接和访问权限'
        clearRows()
      }
    } finally {
      if (valid()) loading.value = false
    }
  }
  const detailVisible = ref(false),
    detailLoading = ref(false),
    detailError = ref(''),
    detail = ref<EnvResourceDetail | null>(null),
    detailTab = ref('yaml'),
    selectedName = ref(''),
    selectedKind = ref('')
  const containers = ref<string[]>([]),
    container = ref(''),
    previousLogs = ref(false),
    logs = ref(''),
    logsError = ref(''),
    logsLoading = ref(false)
  let detailGeneration = 0,
    logGeneration = 0
  function closeDetail() {
    detailGeneration++
    logGeneration++
    detailVisible.value = false
    detail.value = null
    detailError.value = ''
    detailLoading.value = false
    logs.value = ''
    logsError.value = ''
    logsLoading.value = false
  }
  async function openDetail(kind: string, name: string, tab = 'yaml') {
    closeDetail()
    const current = ++detailGeneration,
      env = { ...props.environment }
    selectedKind.value = kind
    selectedName.value = name
    detailTab.value = tab
    detailVisible.value = true
    detailLoading.value = true
    containers.value = []
    container.value = ''
    previousLogs.value = false
    try {
      const value = await getEnvResourceDetailApi(env.id, kind, name)
      if (disposed || current !== detailGeneration || env.id !== props.environment.id) return
      if (
        !value ||
        value.namespace !== env.namespace ||
        value.name !== name ||
        value.kind !== kind ||
        !Array.isArray(value.events)
      )
        throw Error('详情返回范围与当前资源不一致')
      detail.value = value
      if (kind === 'pod') {
        const pod = yaml.load(value.yaml) as {
          spec?: {
            containers?: Array<{ name: string }>
            initContainers?: Array<{ name: string }>
            ephemeralContainers?: Array<{ name: string }>
          }
        }
        containers.value = [
          ...(pod?.spec?.containers || []),
          ...(pod?.spec?.initContainers || []),
          ...(pod?.spec?.ephemeralContainers || [])
        ].map((item) => item.name)
        container.value = containers.value[0] || ''
      }
    } catch (e: any) {
      if (current === detailGeneration) detailError.value = e?.message || '资源详情读取失败'
    } finally {
      if (current === detailGeneration) detailLoading.value = false
    }
  }
  function openLogs(pod: EnvPodItem) {
    return openDetail('pod', pod.name, 'logs')
  }
  async function loadLogs() {
    const current = ++logGeneration,
      envId = props.environment.id,
      name = selectedName.value,
      selected = container.value
    logs.value = ''
    logsError.value = ''
    logsLoading.value = false
    if (!detailVisible.value || !detail.value || detailTab.value !== 'logs' || !selected) return
    logsLoading.value = true
    try {
      const result = await getEnvPodLogsApi(envId, name, selected, previousLogs.value)
      if (
        !disposed &&
        current === logGeneration &&
        envId === props.environment.id &&
        name === selectedName.value &&
        selected === container.value &&
        detailVisible.value
      ) {
        if (typeof result?.logs !== 'string') throw Error('日志返回格式错误')
        logs.value = result.logs
      }
    } catch (e: any) {
      if (current === logGeneration)
        logsError.value = e?.message || '日志读取失败，容器可能尚未启动或上次日志已清理'
    } finally {
      if (current === logGeneration) logsLoading.value = false
    }
  }
  watch([container, previousLogs, detailTab], () => {
    void loadLogs()
  })
  watch(
    () => [props.environment.id, props.environment.cluster_id, props.environment.namespace],
    () => {
      closeDetail()
      restartVisible.value = false
      restartRunVisible.value = false
      restartWorkload.value = null
      search.value = ''
    },
    { flush: 'sync' }
  )
  watch(
    contextKey,
    () => {
      void loadResources()
    },
    { immediate: true, flush: 'sync' }
  )
  onMounted(() => {
    timer = setInterval(() => {
      if (autoRefresh.value && !loading.value && !document.hidden && !detailVisible.value)
        void loadResources()
    }, 15000)
  })
  onBeforeUnmount(() => {
    disposed = true
    generation++
    closeDetail()
    if (timer) clearInterval(timer)
  })
</script>

<style scoped>
  .environment-runtime {
    min-width: 0;
  }
  .runtime-summary {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 16px;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    margin: 12px 0;
    flex-wrap: wrap;
  }
  .target-context {
    display: flex;
    gap: 12px 24px;
    flex-wrap: wrap;
    align-items: center;
    font-size: 13px;
  }
  .target-context span {
    color: var(--el-text-color-secondary);
  }
  .target-context b {
    margin-left: 6px;
    color: var(--el-text-color-primary);
    font-weight: 500;
  }
  .summary-actions,
  .refresh-actions,
  .log-controls {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }
  .runtime-toolbar {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  .resource-tabs {
    display: flex;
    gap: 6px;
    overflow-x: auto;
  }
  .resource-tabs button {
    padding: 12px 14px;
    white-space: nowrap;
    background: none;
    border: 0;
    border-bottom: 2px solid transparent;
    cursor: pointer;
    color: var(--el-text-color-regular);
  }
  .resource-tabs button.active {
    border-bottom-color: var(--el-color-primary);
    color: var(--el-color-primary);
    font-weight: 600;
  }
  .refresh-actions small,
  .resource-scope {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
  .resource-filters {
    display: flex;
    gap: 12px;
    justify-content: space-between;
    flex-wrap: wrap;
    margin-top: 16px;
  }
  .resource-search {
    width: 230px;
  }
  .resource-table {
    min-height: 340px;
  }
  .runtime-output {
    padding: 16px;
    background: #101827;
    color: #e2e8f0;
    overflow: auto;
    max-height: 65vh;
    min-height: 220px;
    font: 12px/1.7 var(--ax-font-mono);
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    border-radius: 6px;
  }
  .log-output {
    min-height: 55vh;
  }
  .log-controls {
    margin-bottom: 10px;
  }
  .log-controls .el-select {
    width: 220px;
  }
  .event-row {
    padding: 12px 0;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  .event-row small {
    display: block;
    color: var(--el-text-color-secondary);
    margin-top: 4px;
  }
  .event-row p {
    overflow-wrap: anywhere;
  }
</style>
