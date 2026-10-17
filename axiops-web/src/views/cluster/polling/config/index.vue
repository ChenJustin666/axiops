<template>
  <div class="inspection-page">
    <ArtSearchBar
      v-show="showSearchBar"
      :model-value="filters"
      :items="taskSearchItems"
      @search="searchTasks"
      @reset="resetTaskFilters"
    />

    <ElCard
      shadow="never"
      class="table-card art-table-card"
      :style="{ 'margin-top': showSearchBar ? '12px' : '0' }"
    >
      <ArtTableHeader
        v-model:columns="taskColumns"
        v-model:showSearchBar="showSearchBar"
        :loading="loading"
        :showZebra="true"
        :showBorder="true"
        :showHeaderBackground="true"
        :fullClass="'art-page-view'"
        :layout="'search,refresh,size,fullscreen,columns,settings'"
        @refresh="loadTasks"
      >
        <template #left>
          <ElSpace wrap>
            <span class="card-title">巡检计划</span>
            <ElTag effect="plain" type="info">当前 {{ tasks.length }} 条</ElTag>
            <ElButton type="primary" :icon="Plus" @click="openTaskDialog()">
              新建巡检计划
            </ElButton>
          </ElSpace>
        </template>
      </ArtTableHeader>
      <ArtTable
        :loading="loading"
        :data="tasks"
        :columns="visibleTaskColumns"
        :pagination="{
          current: pagination.page,
          size: pagination.pageSize,
          total: pagination.total
        }"
        :pagination-options="{ align: 'right' }"
        :show-table-header="false"
        empty-text="暂无巡检计划"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      >
        <template #name="{ row }">
          <div class="meta-block">
            <strong>{{ row.name }}</strong>
            <span>{{ row.description || '暂无描述' }}</span>
          </div>
        </template>
        <template #template="{ row }">{{ templateName(row.templateId) }}</template>
        <template #scope="{ row }">
          <ElTag effect="plain">{{ scopeText(row.scopeType) }}</ElTag>
        </template>
        <template #cluster="{ row }">{{ taskTargetText(row) }}</template>
        <template #schedule="{ row }">{{ scheduleText(row.scheduleType) }}</template>
        <template #cron="{ row }">{{ row.cronExpr || '-' }}</template>
        <template #lastStatus="{ row }">
          <ElTag :type="statusTagType(row.lastStatus)">{{ statusText(row.lastStatus) }}</ElTag>
        </template>
        <template #lastRunAt="{ row }">{{ formatTime(row.lastRunAt) }}</template>
        <template #nextRunAt="{ row }">{{ formatTime(row.nextRunAt) }}</template>
        <template #enabled="{ row }">
          <ElTag :type="row.enabled ? 'success' : 'info'">
            {{ row.enabled ? '启用' : '禁用' }}
          </ElTag>
        </template>
        <template #operation="{ row }">
          <ElSpace class="row-actions">
            <ElButton link type="primary" :loading="runningMap[row.id]" @click="runTask(row)">
              巡检
            </ElButton>
            <ElButton link type="primary" @click="openTaskDialog(row)">编辑</ElButton>
            <ElDropdown trigger="click" @command="(command) => handleTaskCommand(command, row)">
              <ElButton
                link
                type="primary"
                :icon="MoreFilled"
                title="更多操作"
                aria-label="更多操作"
              />
              <template #dropdown>
                <ElDropdownMenu>
                  <ElDropdownItem command="delete">删除</ElDropdownItem>
                </ElDropdownMenu>
              </template>
            </ElDropdown>
          </ElSpace>
        </template>
      </ArtTable>
    </ElCard>

    <ElDialog
      v-model="taskDialogVisible"
      :title="taskDialogMode === 'create' ? '新建巡检计划' : '编辑巡检计划'"
      width="760px"
    >
      <ElForm :model="taskForm" label-width="110px">
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="计划名称" required>
              <ElInput v-model="taskForm.name" placeholder="如 生产集群每日巡检" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="规则组">
              <ElSelect v-model="taskForm.templateId" filterable placeholder="请选择规则组">
                <ElOption
                  v-for="item in templates"
                  :key="item.id"
                  :label="item.name"
                  :value="item.id"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="巡检范围">
              <ElRadioGroup v-model="taskForm.scopeType">
                <ElRadioButton label="cluster">单集群</ElRadioButton>
                <ElRadioButton label="global">全局</ElRadioButton>
              </ElRadioGroup>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem
              label="目标集群"
              :required="taskForm.scopeType === 'cluster' && !taskForm.prometheusEnabled"
            >
              <ElSelect
                v-model="taskForm.clusterUuid"
                filterable
                clearable
                :placeholder="
                  taskForm.prometheusEnabled ? '可选；自定义 Prometheus 可不选' : '选择集群'
                "
                :disabled="taskForm.scopeType === 'global'"
                :loading="clusterLoading"
              >
                <ElOption
                  v-for="item in clusters"
                  :key="item.uuid"
                  :label="item.name"
                  :value="item.uuid"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="调度类型">
              <ElRadioGroup v-model="taskForm.scheduleType">
                <ElRadioButton label="manual">手动</ElRadioButton>
                <ElRadioButton label="cron">定时</ElRadioButton>
              </ElRadioGroup>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="启用状态">
              <ElSwitch v-model="taskForm.enabled" />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElFormItem label="巡检实例">
          <ElInput
            v-model="taskForm.instancesText"
            type="textarea"
            :rows="3"
            placeholder="填写后只巡检这些实例；留空按规则库模式自动发现。每行一个，如主机名、IP:端口、namespace/pod"
          />
        </ElFormItem>
        <ElFormItem v-if="taskForm.scheduleType === 'cron'" label="Cron 表达式" required>
          <ElInput v-model="taskForm.cronExpr" placeholder="如 0 0 2 * * ?" />
          <div class="field-tip">支持秒级可选 Cron，分布式执行由后端 Redis 锁控制。</div>
        </ElFormItem>
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="最大并发">
              <ElInputNumber v-model="taskForm.maxConcurrency" :min="1" :max="20" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="超时秒数">
              <ElInputNumber v-model="taskForm.timeoutSec" :min="30" :max="7200" />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElCollapse class="task-advanced-collapse">
          <ElCollapseItem name="prometheus" title="Prometheus 配置">
            <ElFormItem label="自定义地址">
              <ElSwitch
                v-model="taskForm.prometheusEnabled"
                active-text="计划级 Prometheus"
                inactive-text="集群默认"
              />
            </ElFormItem>
            <template v-if="taskForm.prometheusEnabled">
              <ElFormItem label="访问地址" required>
                <ElInput
                  v-model="taskForm.prometheusEndpoint"
                  placeholder="如 https://prometheus.example.com:9090"
                />
              </ElFormItem>
              <ElRow :gutter="16">
                <ElCol :span="12">
                  <ElFormItem label="启用认证">
                    <ElSwitch v-model="taskForm.prometheusAuthEnabled" />
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="认证类型">
                    <ElSelect
                      v-model="taskForm.prometheusAuthType"
                      :disabled="!taskForm.prometheusAuthEnabled"
                    >
                      <ElOption label="Basic" value="basic" />
                      <ElOption label="Bearer Token" value="token" />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <ElRow
                v-if="taskForm.prometheusAuthEnabled && taskForm.prometheusAuthType === 'basic'"
                :gutter="16"
              >
                <ElCol :span="12">
                  <ElFormItem label="用户名">
                    <ElInput v-model="taskForm.prometheusUsername" />
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="密码">
                    <ElInput v-model="taskForm.prometheusPassword" type="password" show-password />
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <ElFormItem
                v-if="taskForm.prometheusAuthEnabled && taskForm.prometheusAuthType === 'token'"
                label="Token"
              >
                <ElInput v-model="taskForm.prometheusToken" type="textarea" :rows="2" />
              </ElFormItem>
              <ElRow :gutter="16">
                <ElCol :span="12">
                  <ElFormItem label="TLS 证书">
                    <ElSwitch v-model="taskForm.prometheusTlsEnabled" />
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="忽略校验">
                    <ElSwitch v-model="taskForm.prometheusInsecureSkipVerify" />
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <template v-if="taskForm.prometheusTlsEnabled">
                <ElFormItem label="CA 证书">
                  <ElInput v-model="taskForm.prometheusCaCert" type="textarea" :rows="2" />
                </ElFormItem>
                <ElRow :gutter="16">
                  <ElCol :span="12">
                    <ElFormItem label="客户端证书">
                      <ElInput v-model="taskForm.prometheusClientCert" type="textarea" :rows="2" />
                    </ElFormItem>
                  </ElCol>
                  <ElCol :span="12">
                    <ElFormItem label="客户端私钥">
                      <ElInput v-model="taskForm.prometheusClientKey" type="textarea" :rows="2" />
                    </ElFormItem>
                  </ElCol>
                </ElRow>
              </template>
            </template>
          </ElCollapseItem>
          <ElCollapseItem name="advanced" title="高级配置">
            <ElFormItem label="扩展 JSON">
              <ElInput v-model="taskForm.configJson" type="textarea" :rows="3" placeholder="JSON" />
            </ElFormItem>
          </ElCollapseItem>
        </ElCollapse>
        <ElFormItem label="计划描述">
          <ElInput v-model="taskForm.description" type="textarea" :rows="3" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <div class="dialog-footer-actions">
          <ElButton @click="taskDialogVisible = false">取消</ElButton>
          <ElButton type="primary" :loading="saving" @click="saveTask">保存</ElButton>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref, watch } from 'vue'
  import { useRouter } from 'vue-router'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { MoreFilled, Plus } from '@element-plus/icons-vue'
  import {
    createInspectionTaskApi,
    deleteInspectionTaskApi,
    runInspectionTaskApi,
    searchInspectionTasksApi,
    searchInspectionTemplatesApi,
    updateInspectionTaskApi,
    type InspectionTask,
    type InspectionTemplate
  } from '@/api/manager/inspection'
  import {
    formatTime,
    scheduleText,
    scopeText,
    statusTagType,
    statusText
  } from '../shared/constants'
  import { usePollingClusters } from '../shared/usePollingClusters'

  defineOptions({ name: 'InspectionConfigCenter' })

  const router = useRouter()

  const {
    clusters,
    loading: clusterLoading,
    loadClusters
  } = usePollingClusters('inspection-config-cluster')

  const loading = ref(false)
  const saving = ref(false)
  const showSearchBar = ref(true)
  const tasks = ref<InspectionTask[]>([])
  const templates = ref<InspectionTemplate[]>([])
  const runningMap = ref<Record<number, boolean>>({})
  const pagination = reactive({ page: 1, pageSize: 20, total: 0 })
  const taskColumns = ref([
    { prop: 'name', label: '计划名称', minWidth: 230, useSlot: true },
    { prop: 'template', label: '规则组', minWidth: 180, useSlot: true },
    { prop: 'scope', label: '范围', width: 120, useSlot: true },
    { prop: 'cluster', label: '集群', minWidth: 180, useSlot: true },
    { prop: 'schedule', label: '调度', width: 110, useSlot: true },
    { prop: 'cron', label: 'Cron', minWidth: 180, useSlot: true, showOverflowTooltip: true },
    { prop: 'lastStatus', label: '最近状态', width: 120, useSlot: true },
    { prop: 'lastRunAt', label: '运行时间', width: 180, useSlot: true },
    { prop: 'nextRunAt', label: '下次执行', width: 180, useSlot: true },
    { prop: 'enabled', label: '状态', width: 90, useSlot: true },
    { prop: 'operation', label: '操作', fixed: 'right' as const, width: 168, useSlot: true }
  ])
  const visibleTaskColumns = computed(() =>
    taskColumns.value.filter((column: any) => column.visible !== false)
  )
  const filters = reactive<{
    name: string
    scopeType: string
    clusterUuid: string
    scheduleType: string
    enabled?: boolean
  }>({
    name: '',
    scopeType: '',
    clusterUuid: '',
    scheduleType: '',
    enabled: undefined
  })
  const taskSearchItems = computed(() => [
    {
      label: '计划名称',
      key: 'name',
      type: 'input',
      props: { clearable: true, placeholder: '按名称查询' }
    },
    {
      label: '范围',
      key: 'scopeType',
      type: 'select',
      props: {
        clearable: true,
        options: [
          { label: '全局', value: 'global' },
          { label: '单集群', value: 'cluster' }
        ]
      }
    },
    {
      label: '集群',
      key: 'clusterUuid',
      type: 'select',
      props: {
        clearable: true,
        filterable: true,
        loading: clusterLoading.value,
        placeholder: '全部集群',
        options: clusters.value.map((item) => ({
          label: item.name,
          value: item.uuid
        }))
      }
    },
    {
      label: '调度',
      key: 'scheduleType',
      type: 'select',
      props: {
        clearable: true,
        options: [
          { label: '手动', value: 'manual' },
          { label: '定时', value: 'cron' }
        ]
      }
    },
    {
      label: '状态',
      key: 'enabled',
      type: 'select',
      props: {
        clearable: true,
        options: [
          { label: '启用', value: true },
          { label: '禁用', value: false }
        ]
      }
    }
  ])

  const taskDialogVisible = ref(false)
  const taskDialogMode = ref<'create' | 'edit'>('create')
  const taskForm = reactive({
    id: 0,
    name: '',
    description: '',
    templateId: 0,
    scopeType: 'cluster',
    clusterUuid: '',
    scheduleType: 'manual',
    cronExpr: '',
    enabled: true,
    maxConcurrency: 2,
    timeoutSec: 600,
    instancesText: '',
    prometheusEnabled: false,
    prometheusEndpoint: '',
    prometheusAuthEnabled: false,
    prometheusAuthType: 'basic',
    prometheusUsername: '',
    prometheusPassword: '',
    prometheusToken: '',
    prometheusTlsEnabled: false,
    prometheusInsecureSkipVerify: false,
    prometheusCaCert: '',
    prometheusClientCert: '',
    prometheusClientKey: '',
    configJson: ''
  })

  watch(
    () => taskForm.scopeType,
    (value) => {
      if (value === 'global') {
        taskForm.clusterUuid = ''
      }
    }
  )

  async function loadTemplates() {
    const resp = await searchInspectionTemplatesApi({
      page: 1,
      pageSize: 200,
      enabled: true,
      orderField: 'id',
      isAsc: false
    })
    templates.value = resp.data || []
  }

  async function loadTasks() {
    loading.value = true
    try {
      const resp = await searchInspectionTasksApi({
        page: pagination.page,
        pageSize: pagination.pageSize,
        orderField: 'id',
        isAsc: false,
        name: filters.name || undefined,
        scopeType: filters.scopeType || undefined,
        clusterUuid: filters.clusterUuid || undefined,
        scheduleType: filters.scheduleType || undefined,
        enabled: filters.enabled
      })
      tasks.value = resp.data || []
      pagination.total = resp.total || 0
    } finally {
      loading.value = false
    }
  }

  function searchTasks() {
    pagination.page = 1
    loadTasks()
  }

  function resetTaskFilters() {
    Object.assign(filters, {
      name: '',
      scopeType: '',
      clusterUuid: '',
      scheduleType: '',
      enabled: undefined
    })
    searchTasks()
  }

  function handleSizeChange(size: number) {
    pagination.pageSize = size
    pagination.page = 1
    loadTasks()
  }

  function handleCurrentChange(page: number) {
    pagination.page = page
    loadTasks()
  }

  function templateName(id: number) {
    return templates.value.find((item) => item.id === id)?.name || (id ? `#${id}` : '-')
  }

  function clusterName(uuid: string) {
    if (!uuid) return '全部集群'
    return clusters.value.find((item) => item.uuid === uuid)?.name || uuid
  }

  function taskTargetText(row: InspectionTask) {
    if (row.prometheusEnabled) {
      return row.clusterUuid
        ? `${clusterName(row.clusterUuid)} / 计划级 Prometheus`
        : '计划级 Prometheus'
    }
    return clusterName(row.clusterUuid)
  }

  function parseTaskConfig(raw: string) {
    try {
      const parsed = JSON.parse(raw || '{}')
      return {
        instances: normalizeStringList(parsed.instances)
      }
    } catch {
      return { instances: [] as string[] }
    }
  }

  function buildTaskConfigJson() {
    let config: Record<string, any> = {}
    try {
      config = taskForm.configJson ? JSON.parse(taskForm.configJson) : {}
    } catch {
      config = {}
    }
    const instances = parseInstancesText(taskForm.instancesText)
    if (instances.length) {
      config.instances = instances
    } else {
      delete config.instances
    }
    taskForm.configJson = JSON.stringify(config)
    return taskForm.configJson
  }

  function parseInstancesText(value: string) {
    return Array.from(
      new Set(
        String(value || '')
          .split(/[\n,，]+/)
          .map((item) => item.trim())
          .filter(Boolean)
      )
    )
  }

  function normalizeStringList(value: unknown): string[] {
    if (!Array.isArray(value)) return []
    return Array.from(new Set(value.map((item) => String(item || '').trim()).filter(Boolean)))
  }

  function openTaskDialog(row?: InspectionTask) {
    const taskConfig = parseTaskConfig(row?.configJson || '')
    taskDialogMode.value = row ? 'edit' : 'create'
    Object.assign(taskForm, {
      id: row?.id || 0,
      name: row?.name || '',
      description: row?.description || '',
      templateId: row?.templateId || templates.value[0]?.id || 0,
      scopeType: row?.scopeType || 'cluster',
      clusterUuid: row?.clusterUuid || '',
      scheduleType: row?.scheduleType || 'manual',
      cronExpr: row?.cronExpr || '',
      enabled: row?.enabled ?? true,
      maxConcurrency: row?.maxConcurrency || 2,
      timeoutSec: row?.timeoutSec || 600,
      instancesText: taskConfig.instances.join('\n'),
      prometheusEnabled: row?.prometheusEnabled ?? false,
      prometheusEndpoint: row?.prometheusEndpoint || '',
      prometheusAuthEnabled: row?.prometheusAuthEnabled ?? false,
      prometheusAuthType: row?.prometheusAuthType || 'basic',
      prometheusUsername: row?.prometheusUsername || '',
      prometheusPassword: row?.prometheusPassword || '',
      prometheusToken: row?.prometheusToken || '',
      prometheusTlsEnabled: row?.prometheusTlsEnabled ?? false,
      prometheusInsecureSkipVerify: row?.prometheusInsecureSkipVerify ?? false,
      prometheusCaCert: row?.prometheusCaCert || '',
      prometheusClientCert: row?.prometheusClientCert || '',
      prometheusClientKey: row?.prometheusClientKey || '',
      configJson: row?.configJson || ''
    })
    taskDialogVisible.value = true
  }

  async function saveTask() {
    if (!taskForm.name) {
      ElMessage.warning('计划名称不能为空')
      return
    }
    if (taskForm.scopeType === 'cluster' && !taskForm.clusterUuid && !taskForm.prometheusEnabled) {
      ElMessage.warning('单集群巡检必须选择目标集群')
      return
    }
    if (taskForm.scheduleType === 'cron' && !taskForm.cronExpr) {
      ElMessage.warning('定时巡检必须填写 Cron 表达式')
      return
    }
    if (taskForm.prometheusEnabled && !taskForm.prometheusEndpoint) {
      ElMessage.warning('启用计划级 Prometheus 时必须填写访问地址')
      return
    }
    saving.value = true
    try {
      const payload = {
        name: taskForm.name,
        description: taskForm.description,
        templateId: taskForm.templateId,
        scopeType: taskForm.scopeType,
        clusterUuid: taskForm.scopeType === 'global' ? '' : taskForm.clusterUuid,
        scheduleType: taskForm.scheduleType,
        cronExpr: taskForm.scheduleType === 'cron' ? taskForm.cronExpr : '',
        enabled: taskForm.enabled,
        maxConcurrency: taskForm.maxConcurrency,
        timeoutSec: taskForm.timeoutSec,
        configJson: buildTaskConfigJson(),
        prometheusEnabled: taskForm.prometheusEnabled,
        prometheusEndpoint: taskForm.prometheusEnabled ? taskForm.prometheusEndpoint : '',
        prometheusAuthEnabled: taskForm.prometheusEnabled && taskForm.prometheusAuthEnabled,
        prometheusAuthType: taskForm.prometheusAuthEnabled ? taskForm.prometheusAuthType : '',
        prometheusUsername: taskForm.prometheusAuthEnabled ? taskForm.prometheusUsername : '',
        prometheusPassword: taskForm.prometheusAuthEnabled ? taskForm.prometheusPassword : '',
        prometheusToken: taskForm.prometheusAuthEnabled ? taskForm.prometheusToken : '',
        prometheusTlsEnabled: taskForm.prometheusEnabled && taskForm.prometheusTlsEnabled,
        prometheusInsecureSkipVerify:
          taskForm.prometheusEnabled && taskForm.prometheusInsecureSkipVerify,
        prometheusCaCert:
          taskForm.prometheusEnabled && taskForm.prometheusTlsEnabled
            ? taskForm.prometheusCaCert
            : '',
        prometheusClientCert:
          taskForm.prometheusEnabled && taskForm.prometheusTlsEnabled
            ? taskForm.prometheusClientCert
            : '',
        prometheusClientKey:
          taskForm.prometheusEnabled && taskForm.prometheusTlsEnabled
            ? taskForm.prometheusClientKey
            : ''
      }
      if (taskDialogMode.value === 'edit') {
        await updateInspectionTaskApi(taskForm.id, payload)
      } else {
        await createInspectionTaskApi(payload)
      }
      ElMessage.success('保存成功')
      taskDialogVisible.value = false
      await loadTasks()
    } finally {
      saving.value = false
    }
  }

  async function runTask(row: InspectionTask) {
    await ElMessageBox.confirm(`确认立即执行「${row.name}」吗？`, '立即巡检', { type: 'info' })
    runningMap.value = { ...runningMap.value, [row.id]: true }
    try {
      const resp = await runInspectionTaskApi(row.id, {
        clusterUuid: row.scopeType === 'cluster' && row.clusterUuid ? row.clusterUuid : undefined
      })
      const firstRecordId = resp.records?.[0]?.id
      ElMessage.success('巡检任务已提交，请在报告中心查看执行进度')
      await loadTasks()
      await router.push({
        path: '/cluster/polling/report',
        query: {
          taskId: String(row.id),
          ...(firstRecordId ? { recordId: String(firstRecordId), log: '1' } : { status: 'running' })
        }
      })
    } finally {
      runningMap.value = { ...runningMap.value, [row.id]: false }
    }
  }

  function handleTaskCommand(command: string, row: InspectionTask) {
    if (command === 'delete') {
      void removeTask(row)
    }
  }

  async function removeTask(row: InspectionTask) {
    try {
      await ElMessageBox.confirm(`确认删除巡检计划「${row.name}」吗？`, '删除巡检计划', {
        type: 'warning',
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        confirmButtonClass: 'el-button--danger'
      })
    } catch {
      return
    }
    await deleteInspectionTaskApi(row.id)
    ElMessage.success('删除成功')
    await loadTasks()
  }

  onMounted(async () => {
    await Promise.all([loadTemplates(), loadClusters()])
    await loadTasks()
  })
</script>

<style scoped lang="scss">
  @use '../shared/polling-ui' as *;
</style>
