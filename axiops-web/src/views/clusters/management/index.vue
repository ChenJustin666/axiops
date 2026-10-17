<template>
  <div v-loading="loading" class="cluster-page">
    <header class="page-header">
      <div>
        <span class="eyebrow">KUBERNETES</span>
        <h1>集群管理</h1>
        <p>接入测试与生产 Kubernetes 集群，统一查看连接状态和资源容量。</p>
      </div>
      <div class="header-actions">
        <ElButton :icon="RefreshCw" circle title="刷新" @click="loadClusters" />
        <ElButton v-if="canCreate" type="primary" :icon="Plus" @click="openCreate">
          接入集群
        </ElButton>
      </div>
    </header>

    <div class="summary-strip">
      <div
        ><span>集群总数</span><strong>{{ clusters.length }}</strong></div
      >
      <div
        ><span>已连接</span><strong class="success-text">{{ connectedCount }}</strong></div
      >
      <div
        ><span>待检查</span><strong class="warning-text">{{ pendingCount }}</strong></div
      >
      <div
        ><span>节点总数</span><strong>{{ nodeCount }}</strong></div
      >
    </div>

    <div class="table-toolbar">
      <ElInput v-model="keyword" clearable :prefix-icon="Search" placeholder="搜索集群名称或编码" />
      <ElSelect v-model="statusFilter" clearable placeholder="连接状态">
        <ElOption label="已连接" value="connected" />
        <ElOption label="连接中" value="pending" />
        <ElOption label="不可达" value="disconnected" />
        <ElOption label="错误" value="error" />
      </ElSelect>
      <span>{{ filteredClusters.length }} 个集群</span>
    </div>

    <ElTable :data="filteredClusters" row-key="id" class="cluster-table">
      <ElTableColumn label="集群" min-width="245">
        <template #default="{ row }">
          <div class="identity-cell">
            <span class="cluster-symbol"><Cloud :size="18" /></span>
            <span
              ><strong>{{ row.name }}</strong
              ><small>{{ row.code }}</small></span
            >
          </div>
        </template>
      </ElTableColumn>
      <ElTableColumn label="连接状态" width="130">
        <template #default="{ row }">
          <ElTag :type="statusType(row.status)" effect="plain">{{ statusLabel(row.status) }}</ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn label="版本" width="130">
        <template #default="{ row }">{{ row.version || '待检测' }}</template>
      </ElTableColumn>
      <ElTableColumn label="区域 / API Server" min-width="280" show-overflow-tooltip>
        <template #default="{ row }"
          ><span>{{ row.region || '-' }}</span
          ><small class="api-server">{{ row.api_server || '-' }}</small></template
        >
      </ElTableColumn>
      <ElTableColumn label="资源容量" min-width="190">
        <template #default="{ row }"
          ><span>{{ row.node_count || 0 }} 节点</span
          ><small class="api-server"
            >{{ row.cpu_total || '-' }} · {{ row.memory_total || '-' }}</small
          ></template
        >
      </ElTableColumn>
      <ElTableColumn label="操作" width="250" fixed="right">
        <template #default="{ row }">
          <ElButton v-if="canUpdate(row.id)" text type="primary" @click="openEdit(row)"
            >编辑</ElButton
          >
          <ElButton
            v-if="canTest(row.id)"
            text
            :loading="testingId === row.id"
            @click="testConnection(row)"
            >测试连接</ElButton
          >
          <ElButton v-if="canDelete(row.id)" text type="danger" @click="removeCluster(row)"
            >删除</ElButton
          >
        </template>
      </ElTableColumn>
    </ElTable>
    <ElEmpty
      v-if="!loading && !filteredClusters.length"
      description="暂无可访问的 Kubernetes 集群"
    />

    <ElDialog
      v-model="dialogVisible"
      :title="editingId ? '编辑集群' : '接入 Kubernetes 集群'"
      width="720px"
      destroy-on-close
    >
      <ElAlert
        v-if="editingId"
        title="更新 kubeconfig 后会自动重新测试连接；凭据不会在接口或列表中回显。"
        type="info"
        show-icon
        :closable="false"
        class="form-alert"
      />
      <ElForm ref="formRef" :model="form" :rules="rules" label-position="top">
        <div class="form-grid">
          <ElFormItem label="集群名称" prop="name"
            ><ElInput v-model="form.name" placeholder="测试集群"
          /></ElFormItem>
          <ElFormItem label="集群编码" prop="code"
            ><ElInput v-model="form.code" :disabled="Boolean(editingId)" placeholder="test-cluster"
          /></ElFormItem>
        </div>
        <div class="form-grid">
          <ElFormItem label="云厂商 / 类型"
            ><ElInput v-model="form.provider" placeholder="self / aliyun / tencent"
          /></ElFormItem>
          <ElFormItem label="区域"
            ><ElInput v-model="form.region" placeholder="cn-shanghai"
          /></ElFormItem>
        </div>
        <ElFormItem label="API Server" prop="api_server"
          ><ElInput v-model="form.api_server" placeholder="从 kubeconfig 自动读取，也可手动填写"
        /></ElFormItem>
        <ElFormItem label="kubeconfig" prop="kubeconfig">
          <ElInput
            v-model="form.kubeconfig"
            type="textarea"
            :rows="11"
            :placeholder="editingId ? '留空表示保留原 kubeconfig' : '粘贴完整 kubeconfig YAML'"
          />
          <div class="field-note"
            >只接受包含当前 context、cluster server 和 user 凭据的 kubeconfig。服务端会使用 AxiOps
            加密密钥保存。</div
          >
        </ElFormItem>
        <ElFormItem label="描述"
          ><ElInput
            v-model="form.description"
            type="textarea"
            :rows="2"
            maxlength="240"
            show-word-limit
        /></ElFormItem>
      </ElForm>
      <template #footer
        ><ElButton @click="dialogVisible = false">取消</ElButton
        ><ElButton type="primary" :loading="submitting" @click="submit"
          >保存并测试连接</ElButton
        ></template
      >
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref } from 'vue'
  import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
  import { Cloud, Plus, RefreshCw, Search } from 'lucide-vue-next'
  import { parse as parseYaml } from 'yaml'
  import {
    createClusterApi,
    deleteClusterApi,
    listClustersApi,
    testClusterConnectionApi,
    updateClusterApi,
    type AxiOpsCluster,
    type CreateClusterInput
  } from '@/api/axiops'
  import { useAxiOpsAccess } from '@/composables/useAxiOpsAccess'

  defineOptions({ name: 'AxiOpsClusterManagement' })
  const access = useAxiOpsAccess()
  const loading = ref(false)
  const submitting = ref(false)
  const testingId = ref(0)
  const dialogVisible = ref(false)
  const editingId = ref(0)
  const formRef = ref<FormInstance>()
  const clusters = ref<AxiOpsCluster[]>([])
  const keyword = ref('')
  const statusFilter = ref('')
  const form = reactive<CreateClusterInput>({
    name: '',
    code: '',
    provider: 'self',
    region: '',
    api_server: '',
    kubeconfig: '',
    description: ''
  })
  const canCreate = computed(() => access.can('cluster', 'create'))
  const canUpdate = (id: number) => access.can('cluster', 'update', { type: 'cluster', id })
  const canTest = (id: number) => access.can('cluster', 'test', { type: 'cluster', id })
  const canDelete = (id: number) => access.can('cluster', 'delete', { type: 'cluster', id })
  const connectedCount = computed(
    () => clusters.value.filter((item) => item.status === 'connected').length
  )
  const pendingCount = computed(
    () => clusters.value.filter((item) => ['pending', 'disconnected'].includes(item.status)).length
  )
  const nodeCount = computed(() =>
    clusters.value.reduce((sum, item) => sum + (item.node_count || 0), 0)
  )
  const filteredClusters = computed(() =>
    clusters.value.filter((item) => {
      const text = `${item.name} ${item.code}`.toLowerCase()
      return (
        (!keyword.value || text.includes(keyword.value.trim().toLowerCase())) &&
        (!statusFilter.value || item.status === statusFilter.value)
      )
    })
  )
  const rules: FormRules = {
    name: [{ required: true, message: '请输入集群名称', trigger: 'blur' }],
    code: [
      {
        required: true,
        pattern: /^[a-z][a-z0-9-]{1,63}$/,
        message: '使用小写字母、数字和连字符',
        trigger: 'blur'
      }
    ],
    api_server: [
      { required: true, type: 'url', message: '请输入有效的 API Server 地址', trigger: 'blur' }
    ],
    kubeconfig: [
      {
        validator: (_rule, value, callback) =>
          editingId.value || value ? callback() : callback(new Error('请粘贴 kubeconfig')),
        trigger: 'blur'
      }
    ]
  }
  const statusLabel = (status: string) =>
    ({ connected: '已连接', pending: '连接中', disconnected: '不可达', error: '配置错误' })[
      status
    ] ||
    status ||
    '未知'
  const statusType = (status: string): 'success' | 'warning' | 'danger' | 'info' =>
    status === 'connected'
      ? 'success'
      : status === 'error'
        ? 'danger'
        : status === 'pending'
          ? 'warning'
          : 'info'

  const loadClusters = async () => {
    loading.value = true
    try {
      await access.load(true)
      clusters.value = (await listClustersApi()) || []
    } finally {
      loading.value = false
    }
  }
  const resetForm = () =>
    Object.assign(form, {
      name: '',
      code: '',
      provider: 'self',
      region: '',
      api_server: '',
      kubeconfig: '',
      description: ''
    })
  const openCreate = () => {
    editingId.value = 0
    resetForm()
    dialogVisible.value = true
  }
  const openEdit = (item: AxiOpsCluster) => {
    editingId.value = item.id
    Object.assign(form, {
      name: item.name,
      code: item.code,
      provider: item.provider,
      region: item.region,
      api_server: item.api_server || '',
      kubeconfig: '',
      description: item.description || ''
    })
    dialogVisible.value = true
  }
  const validateKubeconfig = () => {
    if (!form.kubeconfig.trim()) return editingId.value > 0
    let parsed: any
    try {
      parsed = parseYaml(form.kubeconfig)
    } catch {
      ElMessage.warning('kubeconfig 不是有效 YAML')
      return false
    }
    if (
      parsed?.apiVersion !== 'v1' ||
      !Array.isArray(parsed.clusters) ||
      !Array.isArray(parsed.users) ||
      !Array.isArray(parsed.contexts) ||
      !parsed['current-context']
    ) {
      ElMessage.warning('kubeconfig 缺少 clusters、users、contexts 或 current-context')
      return false
    }
    const currentContext = parsed.contexts.find(
      (item: any) => item.name === parsed['current-context']
    )
    const clusterName = currentContext?.context?.cluster
    const userName = currentContext?.context?.user
    const server = parsed.clusters.find((item: any) => item.name === clusterName)?.cluster?.server
    if (!server) {
      ElMessage.warning('kubeconfig 当前 context 没有可用 API Server')
      return false
    }
    if (!userName || !parsed.users.some((item: any) => item.name === userName)) {
      ElMessage.warning('kubeconfig 当前 context 没有匹配的用户凭据')
      return false
    }
    form.api_server = server
    return true
  }
  const submit = async () => {
    // Parse kubeconfig first so its server can satisfy the API Server rule.
    if (!validateKubeconfig() || !(await formRef.value?.validate())) return
    submitting.value = true
    try {
      if (editingId.value)
        await updateClusterApi(editingId.value, {
          name: form.name,
          description: form.description,
          api_server: form.api_server,
          kubeconfig: form.kubeconfig || undefined
        })
      else await createClusterApi(form)
      ElMessage.success('集群已保存，正在测试连接')
      dialogVisible.value = false
      await loadClusters()
    } finally {
      submitting.value = false
    }
  }
  const testConnection = async (item: AxiOpsCluster) => {
    testingId.value = item.id
    try {
      await testClusterConnectionApi(item.id)
      ElMessage.success('连接测试已触发')
      await new Promise((resolve) => setTimeout(resolve, 800))
      await loadClusters()
    } finally {
      testingId.value = 0
    }
  }
  const removeCluster = async (item: AxiOpsCluster) => {
    await ElMessageBox.confirm(
      `确认删除集群“${item.name}”？已被环境或发布记录引用的集群会被系统拒绝删除。`,
      '删除集群',
      { type: 'warning' }
    )
    await deleteClusterApi(item.id)
    ElMessage.success('集群已删除')
    await loadClusters()
  }
  onMounted(loadClusters)
</script>

<style scoped lang="scss">
  .cluster-page {
    min-height: 100%;
    padding: 22px;
    background: var(--art-main-bg-color);
    color: var(--art-text-gray-900);
  }
  .page-header {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 20px;
    margin-bottom: 22px;
  }
  .eyebrow {
    color: var(--el-color-primary);
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 1.2px;
  }
  h1 {
    margin: 6px 0 4px;
    font-size: 25px;
    letter-spacing: 0;
  }
  .page-header p {
    margin: 0;
    color: var(--art-text-gray-600);
    font-size: 13px;
  }
  .header-actions {
    display: flex;
    gap: 9px;
  }
  .summary-strip {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    margin-bottom: 20px;
    border: 1px solid var(--art-border-color);
    border-radius: 6px;
  }
  .summary-strip > div {
    min-height: 88px;
    padding: 16px 20px;
    border-right: 1px solid var(--art-border-color);
  }
  .summary-strip > div:last-child {
    border-right: 0;
  }
  .summary-strip span,
  .summary-strip strong {
    display: block;
  }
  .summary-strip span {
    color: var(--art-text-gray-500);
    font-size: 12px;
  }
  .summary-strip strong {
    margin-top: 8px;
    font-size: 25px;
  }
  .success-text {
    color: var(--el-color-success);
  }
  .warning-text {
    color: var(--el-color-warning);
  }
  .table-toolbar {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 12px;
  }
  .table-toolbar .el-input {
    width: 300px;
  }
  .table-toolbar .el-select {
    width: 150px;
  }
  .table-toolbar > span {
    margin-left: auto;
    color: var(--art-text-gray-500);
    font-size: 12px;
  }
  .cluster-table {
    border-top: 1px solid var(--art-border-color);
  }
  .identity-cell {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .identity-cell > span:last-child {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .identity-cell small,
  .api-server {
    display: block;
    margin-top: 4px;
    color: var(--art-text-gray-500);
    font-size: 12px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .cluster-symbol {
    width: 34px;
    height: 34px;
    display: grid;
    place-items: center;
    border-radius: 6px;
    color: #2563eb;
    background: #e9f1ff;
  }
  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 14px;
  }
  .form-alert {
    margin-bottom: 14px;
  }
  .field-note {
    margin-top: 6px;
    color: var(--art-text-gray-500);
    font-size: 12px;
    line-height: 1.5;
  }
  @media (max-width: 800px) {
    .cluster-page {
      padding: 14px;
    }
    .page-header {
      align-items: flex-start;
      flex-direction: column;
    }
    .summary-strip {
      grid-template-columns: 1fr 1fr;
    }
    .summary-strip > div:nth-child(2) {
      border-right: 0;
    }
    .table-toolbar {
      flex-wrap: wrap;
    }
    .table-toolbar .el-input,
    .table-toolbar .el-select {
      width: 100%;
    }
    .table-toolbar > span {
      width: 100%;
      margin-left: 0;
    }
    .form-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
