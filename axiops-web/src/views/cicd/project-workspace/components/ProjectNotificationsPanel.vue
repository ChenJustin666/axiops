<template>
  <section class="notifications-panel">
    <div class="section-heading">
      <div><h2>通知</h2><p>项目交付结果订阅与投递追踪</p></div>
      <ElButton v-if="canCreate" type="primary" :icon="Plus" @click="createVisible = true">
        新建订阅
      </ElButton>
    </div>

    <ElTabs v-model="activeTab" @tab-change="handleTabChange">
      <ElTabPane label="通知订阅" name="subscriptions">
        <ElTable v-loading="subscriptionLoading" :data="subscriptions" row-key="id">
          <ElTableColumn label="订阅" min-width="190">
            <template #default="{ row }">
              <div class="identity-cell"
                ><strong>{{ row.name }}</strong
                ><small>{{ channelLabel(row.channel) }}</small></div
              >
            </template>
          </ElTableColumn>
          <ElTableColumn label="范围" width="120">
            <template #default="{ row }">{{ row.user_id ? '仅触发人' : '项目全部' }}</template>
          </ElTableColumn>
          <ElTableColumn label="事件" width="170">
            <template #default="{ row }">
              <ElTag v-if="row.on_success" size="small" type="success" effect="plain">成功</ElTag>
              <ElTag v-if="row.on_failure" size="small" type="danger" effect="plain">失败</ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn label="接收目标" min-width="190">
            <template #default="{ row }">{{
              row.recipient || credentialLabel(row.credential_id)
            }}</template>
          </ElTableColumn>
          <ElTableColumn label="状态" width="90">
            <template #default="{ row }">
              <ElTag size="small" :type="row.enabled ? 'success' : 'info'">{{
                row.enabled ? '启用' : '停用'
              }}</ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn v-if="canDelete" label="操作" width="90" fixed="right">
            <template #default="{ row }">
              <ElButton text type="danger" @click="removeSubscription(row)">删除</ElButton>
            </template>
          </ElTableColumn>
        </ElTable>
        <ElEmpty v-if="!subscriptionLoading && !subscriptions.length" description="暂无通知订阅" />
      </ElTabPane>

      <ElTabPane label="投递记录" name="deliveries">
        <div class="delivery-toolbar">
          <ElSelect
            v-model="deliveryStatus"
            clearable
            placeholder="全部状态"
            @change="reloadDeliveries"
          >
            <ElOption label="等待投递" value="pending" />
            <ElOption label="投递中" value="processing" />
            <ElOption label="成功" value="succeeded" />
            <ElOption label="失败终止" value="dead" />
          </ElSelect>
          <ElButton :icon="RefreshCw" circle title="刷新投递记录" @click="reloadDeliveries" />
        </div>
        <ElTable v-loading="deliveryLoading" :data="deliveries" row-key="id">
          <ElTableColumn label="订阅" min-width="170">
            <template #default="{ row }">
              <div class="identity-cell"
                ><strong>{{ row.subscription_name || `#${row.subscription_id}` }}</strong
                ><small>{{ channelLabel(row.channel) }}</small></div
              >
            </template>
          </ElTableColumn>
          <ElTableColumn label="事件" min-width="160">
            <template #default="{ row }">{{ row.event_type }} #{{ row.event_id }}</template>
          </ElTableColumn>
          <ElTableColumn label="状态" width="120">
            <template #default="{ row }"
              ><ElTag size="small" :type="deliveryStatusType(row.status)">{{
                deliveryStatusLabel(row.status)
              }}</ElTag></template
            >
          </ElTableColumn>
          <ElTableColumn prop="attempts" label="尝试" width="80" align="center" />
          <ElTableColumn label="失败原因" min-width="220" show-overflow-tooltip>
            <template #default="{ row }">{{ row.last_error || '-' }}</template>
          </ElTableColumn>
          <ElTableColumn label="创建时间" width="180">
            <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
          </ElTableColumn>
        </ElTable>
        <ElEmpty v-if="!deliveryLoading && !deliveries.length" description="暂无投递记录" />
        <ElPagination
          v-if="deliveryTotal > deliveryPageSize"
          v-model:current-page="deliveryPage"
          :page-size="deliveryPageSize"
          :total="deliveryTotal"
          layout="prev, pager, next, total"
          @current-change="loadDeliveries"
        />
      </ElTabPane>
    </ElTabs>

    <ElDialog v-model="createVisible" title="新建通知订阅" width="680px" destroy-on-close>
      <ElForm label-position="top">
        <div class="form-grid">
          <ElFormItem label="订阅名称" required
            ><ElInput v-model="form.name" placeholder="发布结果通知"
          /></ElFormItem>
          <ElFormItem label="通知渠道" required>
            <ElSelect v-model="form.channel" style="width: 100%" @change="resetCredentialSelection">
              <ElOption
                v-for="item in channelOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
          </ElFormItem>
        </div>
        <ElFormItem label="通知范围">
          <ElSegmented v-model="form.scope" :options="scopeOptions" />
        </ElFormItem>
        <div class="event-row">
          <span><strong>构建/发布成功</strong><small>顺利完成时发送结果</small></span
          ><ElSwitch v-model="form.on_success" />
        </div>
        <div class="event-row">
          <span><strong>构建/发布失败</strong><small>发送失败节点和原因</small></span
          ><ElSwitch v-model="form.on_failure" />
        </div>
        <ElFormItem v-if="form.channel === 'email'" label="收件邮箱" required>
          <ElInput v-model="form.recipient" placeholder="developer@example.com" />
        </ElFormItem>
        <div v-if="canCreateCredential" class="event-row">
          <span><strong>创建新凭据</strong><small>凭据由平台统一加密保存</small></span
          ><ElSwitch v-model="form.create_credential" />
        </div>
        <template v-if="form.create_credential">
          <ElFormItem label="凭据名称" required
            ><ElInput v-model="form.credential_name"
          /></ElFormItem>
          <template v-if="form.channel === 'email'">
            <div class="form-grid thirds">
              <ElFormItem label="SMTP 主机" required
                ><ElInput v-model="form.smtp_host"
              /></ElFormItem>
              <ElFormItem label="端口" required
                ><ElInputNumber v-model="form.smtp_port" :min="1" :max="65535" style="width: 100%"
              /></ElFormItem>
              <ElFormItem label="发件人" required><ElInput v-model="form.smtp_from" /></ElFormItem>
            </div>
            <div class="form-grid">
              <ElFormItem label="用户名"><ElInput v-model="form.smtp_username" /></ElFormItem>
              <ElFormItem label="密码"
                ><ElInput v-model="form.smtp_password" type="password" show-password
              /></ElFormItem>
            </div>
          </template>
          <ElFormItem v-else label="Webhook HTTPS 地址" required>
            <ElInput
              v-model="form.webhook_url"
              placeholder="https://open.example.com/webhook/..."
            />
          </ElFormItem>
        </template>
        <ElFormItem v-else label="加密凭据" required>
          <ElSelect
            v-model="form.credential_id"
            filterable
            style="width: 100%"
            placeholder="选择凭据"
          >
            <ElOption
              v-for="credential in availableCredentials"
              :key="credential.id"
              :label="credential.name"
              :value="credential.id"
            />
          </ElSelect>
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="createVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="submitting" @click="submitSubscription"
          >创建订阅</ElButton
        >
      </template>
    </ElDialog>
  </section>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { Plus, RefreshCw } from 'lucide-vue-next'
  import { useUserStore } from '@/store/modules/user'
  import {
    createCredentialApi,
    createNotificationSubscriptionApi,
    deleteNotificationSubscriptionApi,
    listCredentialsApi,
    listNotificationDeliveriesApi,
    listNotificationSubscriptionsApi,
    type AxiOpsCredential,
    type AxiOpsNotificationDelivery,
    type AxiOpsNotificationSubscription,
    type NotificationChannel
  } from '@/api/axiops'

  const props = defineProps<{
    projectId: number
    canCreate: boolean
    canDelete: boolean
    canCreateCredential: boolean
    canViewCredential: boolean
  }>()
  const userStore = useUserStore()
  const activeTab = ref('subscriptions')
  const subscriptionLoading = ref(false)
  const deliveryLoading = ref(false)
  const submitting = ref(false)
  const createVisible = ref(false)
  const subscriptions = ref<AxiOpsNotificationSubscription[]>([])
  const deliveries = ref<AxiOpsNotificationDelivery[]>([])
  const credentials = ref<AxiOpsCredential[]>([])
  const deliveryStatus = ref('')
  const deliveryPage = ref(1)
  const deliveryPageSize = 20
  const deliveryTotal = ref(0)
  const channelOptions: Array<{ label: string; value: NotificationChannel }> = [
    { label: '飞书', value: 'feishu' },
    { label: '钉钉', value: 'dingtalk' },
    { label: '企业微信', value: 'wecom' },
    { label: '邮件', value: 'email' },
    { label: 'Webhook', value: 'webhook' }
  ]
  const scopeOptions = [
    { label: '项目全部', value: 'project' },
    { label: '仅本人触发', value: 'self' }
  ]
  const form = reactive({
    name: '',
    channel: 'feishu' as NotificationChannel,
    scope: 'project',
    on_success: true,
    on_failure: true,
    recipient: '',
    credential_id: 0,
    create_credential: false,
    credential_name: '',
    webhook_url: '',
    smtp_host: '',
    smtp_port: 465,
    smtp_username: '',
    smtp_password: '',
    smtp_from: ''
  })

  const availableCredentials = computed(() =>
    credentials.value.filter((item) =>
      form.channel === 'email' ? item.type === 'smtp' : item.type === 'notification_webhook'
    )
  )

  const loadSubscriptions = async () => {
    subscriptionLoading.value = true
    try {
      subscriptions.value = (await listNotificationSubscriptionsApi(props.projectId)) || []
    } finally {
      subscriptionLoading.value = false
    }
  }
  const loadCredentials = async () => {
    if (!props.canViewCredential) return
    const [webhooks, smtp] = await Promise.all([
      listCredentialsApi({ type: 'notification_webhook', project_id: props.projectId }),
      listCredentialsApi({ type: 'smtp', project_id: props.projectId })
    ])
    credentials.value = [...(webhooks || []), ...(smtp || [])]
  }
  const loadDeliveries = async () => {
    deliveryLoading.value = true
    try {
      const result = await listNotificationDeliveriesApi(props.projectId, {
        status: deliveryStatus.value || undefined,
        page: deliveryPage.value,
        page_size: deliveryPageSize
      })
      deliveries.value = result.list || []
      deliveryTotal.value = result.total || 0
    } finally {
      deliveryLoading.value = false
    }
  }
  const reloadDeliveries = () => {
    deliveryPage.value = 1
    void loadDeliveries()
  }
  const handleTabChange = (name: string | number) => {
    if (name === 'deliveries') void loadDeliveries()
  }
  const resetCredentialSelection = () => {
    form.credential_id = 0
    form.recipient = ''
  }
  const resetForm = () => {
    Object.assign(form, {
      name: '',
      channel: 'feishu',
      scope: 'project',
      on_success: true,
      on_failure: true,
      recipient: '',
      credential_id: 0,
      create_credential: false,
      credential_name: '',
      webhook_url: '',
      smtp_host: '',
      smtp_port: 465,
      smtp_username: '',
      smtp_password: '',
      smtp_from: ''
    })
  }

  const submitSubscription = async () => {
    if (!form.name.trim()) return ElMessage.warning('请填写订阅名称')
    if (!form.on_success && !form.on_failure) return ElMessage.warning('至少选择一个通知事件')
    if (form.scope === 'self' && !userStore.info.userId)
      return ElMessage.warning('当前用户信息不可用')
    if (form.channel === 'email' && !/^\S+@\S+\.\S+$/.test(form.recipient.trim()))
      return ElMessage.warning('请输入有效收件邮箱')
    submitting.value = true
    try {
      let credentialId = form.credential_id
      if (form.create_credential) {
        if (!form.credential_name.trim()) return ElMessage.warning('请填写凭据名称')
        let data = ''
        if (form.channel === 'email') {
          if (!form.smtp_host.trim() || !form.smtp_from.trim())
            return ElMessage.warning('请填写 SMTP 主机和发件人')
          data = JSON.stringify({
            host: form.smtp_host.trim(),
            port: form.smtp_port,
            username: form.smtp_username,
            password: form.smtp_password,
            from: form.smtp_from.trim()
          })
        } else {
          try {
            const url = new URL(form.webhook_url.trim())
            if (url.protocol !== 'https:') return ElMessage.warning('Webhook 必须使用 HTTPS')
          } catch {
            return ElMessage.warning('请输入有效的 Webhook 地址')
          }
          data = JSON.stringify({ url: form.webhook_url.trim() })
        }
        credentialId = (
          await createCredentialApi({
            name: form.credential_name.trim(),
            type: form.channel === 'email' ? 'smtp' : 'notification_webhook',
            description: `用于 ${form.name.trim()} 通知订阅`,
            data,
            project_id: props.projectId
          })
        ).id
      }
      if (!credentialId) return ElMessage.warning('请选择或创建通知凭据')
      await createNotificationSubscriptionApi(props.projectId, {
        user_id: form.scope === 'self' ? Number(userStore.info.userId) : 0,
        name: form.name.trim(),
        channel: form.channel,
        credential_id: credentialId,
        recipient: form.channel === 'email' ? form.recipient.trim() : '',
        on_success: form.on_success,
        on_failure: form.on_failure
      })
      ElMessage.success('通知订阅已创建')
      createVisible.value = false
      resetForm()
      await Promise.all([loadSubscriptions(), loadCredentials()])
    } finally {
      submitting.value = false
    }
  }

  const removeSubscription = async (item: AxiOpsNotificationSubscription) => {
    await ElMessageBox.confirm(`确认删除通知订阅“${item.name}”？`, '删除订阅', { type: 'warning' })
    await deleteNotificationSubscriptionApi(item.id, props.projectId)
    ElMessage.success('通知订阅已删除')
    await loadSubscriptions()
  }
  const credentialLabel = (id: number) =>
    credentials.value.find((item) => item.id === id)?.name || `凭据 #${id}`
  const channelLabel = (channel: NotificationChannel) =>
    channelOptions.find((item) => item.value === channel)?.label || channel
  const deliveryStatusLabel = (status: AxiOpsNotificationDelivery['status']) =>
    ({ pending: '等待投递', processing: '投递中', succeeded: '成功', dead: '失败终止' })[status]
  const deliveryStatusType = (status: AxiOpsNotificationDelivery['status']) =>
    ({ pending: 'warning', processing: 'primary', succeeded: 'success', dead: 'danger' })[
      status
    ] as 'warning' | 'primary' | 'success' | 'danger'
  const formatDateTime = (value: string) => (value ? new Date(value).toLocaleString() : '-')

  onMounted(() => void Promise.all([loadSubscriptions(), loadCredentials()]))
</script>

<style scoped lang="scss">
  .notifications-panel {
    min-height: 420px;
  }
  .section-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 14px;
  }
  .section-heading h2 {
    margin: 0;
    font-size: 20px;
  }
  .section-heading p {
    margin: 5px 0 0;
    color: var(--art-text-gray-500);
  }
  .identity-cell {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .identity-cell small {
    margin-top: 3px;
    color: var(--art-text-gray-500);
  }
  .delivery-toolbar {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-bottom: 12px;
  }
  .delivery-toolbar .el-select {
    width: 150px;
  }
  .el-pagination {
    justify-content: flex-end;
    margin-top: 14px;
  }
  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 14px;
  }
  .form-grid.thirds {
    grid-template-columns: 1.2fr 0.7fr 1.2fr;
  }
  .event-row {
    min-height: 58px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid var(--art-border-color);
  }
  .event-row span {
    display: flex;
    flex-direction: column;
  }
  .event-row small {
    margin-top: 3px;
    color: var(--art-text-gray-500);
  }
  @media (max-width: 700px) {
    .form-grid,
    .form-grid.thirds {
      grid-template-columns: 1fr;
    }
    .section-heading {
      align-items: flex-start;
      gap: 12px;
    }
  }
</style>
