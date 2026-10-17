<template>
  <div class="devops-channel-center-page art-full-height">
    <ArtSearchBar
      v-show="showSearchBar"
      ref="searchRef"
      v-model="searchForm"
      :items="searchItems"
      @search="handleSearch"
      @reset="handleReset"
    />

    <ArtCard
      v-model:showSearchBar="showSearchBar"
      :loading="loading"
      :data="channelData"
      :pagination="pagination"
      empty-text="暂无渠道实例"
      :min-card-width="360"
      @add="openDialog('add')"
      @refresh="refreshAll"
      @pagination:size-change="handleSizeChange"
      @pagination:current-change="handleCurrentChange"
    >
      <template #summary>
        <div class="summary-list">
          <div class="summary-item">
            <span class="summary-label">渠道实例</span>
            <strong>{{ summary.total }}</strong>
          </div>
          <div class="summary-item">
            <span class="summary-label">启用实例</span>
            <strong>{{ summary.enabled }}</strong>
          </div>
          <div class="summary-item">
            <span class="summary-label">连通正常</span>
            <strong>{{ summary.healthy }}</strong>
          </div>
          <div class="summary-item">
            <span class="summary-label">连通异常</span>
            <strong>{{ summary.unhealthy }}</strong>
          </div>
        </div>

        <ElAlert
          v-if="errorMessage"
          class="channel-error"
          :title="errorMessage"
          type="error"
          show-icon
          :closable="false"
        />

        <div class="channel-chart-grid">
          <div class="chart-panel">
            <div class="panel-title">渠道健康态势</div>
            <div ref="healthChartRef" class="chart-box" />
          </div>
          <div class="chart-panel">
            <div class="panel-title">类型实例分布</div>
            <div ref="typeChartRef" class="chart-box" />
          </div>
        </div>
      </template>

      <template #empty-actions>
        <ElButton type="primary" :icon="Plus" @click="openDialog('add')">新增</ElButton>
      </template>

      <template #default="{ item }">
        <ElCard class="channel-card" shadow="hover" :body-style="{ padding: '0' }">
          <div class="channel-card-header">
            <div class="channel-icon" :style="{ color: channelColor(item) }">
              <ArtSvgIcon :icon="channelIcon(item)" />
            </div>
            <div class="channel-title">
              <div class="channel-name-row">
                <span class="channel-name" :title="item.name">{{ item.name }}</span>
                <ElTag v-if="item.isSystem" size="small" type="warning">系统</ElTag>
              </div>
              <span class="channel-code" :title="item.code">{{ item.code }}</span>
            </div>
          </div>

          <div class="channel-card-body">
            <div class="channel-tags">
              <ElTag size="small" type="info">{{
                channelTypeLabel(item.channelType, item.groupId)
              }}</ElTag>
              <ElTag size="small" :type="item.status === 1 ? 'success' : 'info'">
                {{ item.status === 1 ? '启用' : '停用' }}
              </ElTag>
              <ElTag size="small" :type="healthTagType(item.healthStatus)">
                {{ item.healthStatus || 'unknown' }}
              </ElTag>
            </div>
            <p class="channel-desc">{{ item.description || '暂无描述' }}</p>
            <div class="channel-meta">
              <span>分组</span>
              <strong :title="groupNameMap[item.groupId] || '-'">
                {{ groupNameMap[item.groupId] || '-' }}
              </strong>
            </div>
            <div class="channel-meta">
              <span>地址</span>
              <strong :title="item.endpoint || '未配置'">{{ item.endpoint || '未配置' }}</strong>
            </div>
            <div class="channel-meta">
              <span>凭据</span>
              <strong :title="credentialLabel(item)">{{ credentialLabel(item) }}</strong>
            </div>
            <div class="channel-meta">
              <span>配置</span>
              <strong :title="configSummary(item)">{{ configSummary(item) }}</strong>
            </div>
            <div class="channel-meta">
              <span>版本</span>
              <strong :title="metadataSummary(item)">{{ metadataSummary(item) }}</strong>
            </div>
            <div class="channel-health-message" :title="item.lastCheckMessage || '未检测'">
              {{ item.lastCheckMessage || '未检测' }}
            </div>
          </div>

          <div class="channel-card-footer">
            <ElButton
              text
              type="success"
              :loading="testingChannelId === item.id"
              @click="handleTestExisting(item)"
            >
              测试连接
            </ElButton>
            <ElButton text type="primary" :icon="Edit" @click="openDialog('edit', item)">
              编辑
            </ElButton>
            <ElButton
              v-if="item.channelType === 'jenkins'"
              text
              type="primary"
              @click="openAgentDialog(item)"
            >
              Agent
            </ElButton>
            <ElButton
              v-if="item.channelType === 'tekton'"
              text
              type="primary"
              :loading="syncingTektonChannelId === item.id"
              @click="syncTektonSteps(item)"
            >
              同步步骤
            </ElButton>
            <ElButton
              text
              type="danger"
              :icon="Delete"
              :disabled="item.isSystem"
              @click="deleteChannel(item)"
            >
              删除
            </ElButton>
          </div>
        </ElCard>
      </template>
    </ArtCard>

    <ElDialog
      v-model="dialogVisible"
      :title="dialogType === 'add' ? '新增渠道' : '编辑渠道'"
      width="760px"
      align-center
      :close-on-click-modal="false"
      @close="resetForm"
    >
      <ElForm ref="formRef" :model="formData" :rules="rules" label-width="150px">
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="名称" prop="name">
              <ElInput v-model="formData.name" :disabled="submitLoading" placeholder="请输入名称" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="编码" prop="code">
              <template #label>
                <span class="form-label-help">
                  编码
                  <ElTooltip :content="formHelpText('code')" placement="top">
                    <ElIcon class="field-help-icon"><QuestionFilled /></ElIcon>
                  </ElTooltip>
                </span>
              </template>
              <ElInput
                v-model="formData.code"
                :disabled="dialogType === 'edit' || submitLoading"
                placeholder="请输入编码"
              />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="渠道分组" prop="groupId">
              <template #label>
                <span class="form-label-help">
                  渠道分组
                  <ElTooltip :content="formHelpText('groupId')" placement="top">
                    <ElIcon class="field-help-icon"><QuestionFilled /></ElIcon>
                  </ElTooltip>
                </span>
              </template>
              <ElSelect
                v-model="formData.groupId"
                :disabled="submitLoading"
                placeholder="请选择渠道分组"
                style="width: 100%"
                @change="handleGroupChange"
                @visible-change="handleBaseSelectVisibleChange"
              >
                <ElOption
                  v-for="item in groupOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="渠道类型" prop="channelType">
              <template #label>
                <span class="form-label-help">
                  渠道类型
                  <ElTooltip :content="formHelpText('channelType')" placement="top">
                    <ElIcon class="field-help-icon"><QuestionFilled /></ElIcon>
                  </ElTooltip>
                </span>
              </template>
              <ElSelect
                v-model="formData.channelType"
                :disabled="submitLoading || !formData.groupId"
                filterable
                placeholder="请选择渠道类型"
                style="width: 100%"
                @change="handleChannelTypeChange"
                @visible-change="handleBaseSelectVisibleChange"
              >
                <ElOption
                  v-for="item in filteredChannelTypeOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElFormItem label="渠道图标">
          <div class="icon-picker">
            <button
              v-for="item in iconOptions"
              :key="item.icon"
              type="button"
              :class="{ active: formData.icon === item.icon }"
              :style="{ color: item.color }"
              @click="pickIcon(item.icon, item.color)"
            >
              <ArtSvgIcon :icon="item.icon" />
            </button>
          </div>
        </ElFormItem>
        <ElFormItem label="访问地址" prop="endpoint">
          <template #label>
            <span class="form-label-help">
              访问地址
              <ElTooltip :content="formHelpText('endpoint')" placement="top">
                <ElIcon class="field-help-icon"><QuestionFilled /></ElIcon>
              </ElTooltip>
            </span>
          </template>
          <ElInput
            v-model="formData.endpoint"
            :disabled="submitLoading"
            placeholder="请输入访问地址"
          />
        </ElFormItem>
        <ElRow :gutter="16">
          <ElCol :span="14">
            <ElFormItem label="凭据" prop="credentialId">
              <template #label>
                <span class="form-label-help">
                  凭据
                  <ElTooltip :content="formHelpText('credentialId')" placement="top">
                    <ElIcon class="field-help-icon"><QuestionFilled /></ElIcon>
                  </ElTooltip>
                </span>
              </template>
              <ElSelect
                v-model="formData.credentialId"
                :disabled="submitLoading"
                clearable
                filterable
                placeholder="请选择凭据"
                style="width: 100%"
                @visible-change="handleCredentialVisibleChange"
              >
                <ElOption
                  v-for="item in filteredCredentialOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="10">
            <ElButton class="credential-add-btn" :icon="Plus" @click="openCredentialDialog">
              新增凭据
            </ElButton>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="禁用 TLS 校验" prop="insecureSkipTls">
              <template #label>
                <span class="form-label-help">
                  禁用 TLS 校验
                  <ElTooltip :content="formHelpText('insecureSkipTls')" placement="top">
                    <ElIcon class="field-help-icon"><QuestionFilled /></ElIcon>
                  </ElTooltip>
                </span>
              </template>
              <ElSwitch
                v-model="formData.insecureSkipTls"
                :disabled="submitLoading"
                inline-prompt
                active-text="是"
                inactive-text="否"
              />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="状态" prop="status">
              <ElRadioGroup v-model="formData.status" :disabled="submitLoading">
                <ElRadio :value="1">启用</ElRadio>
                <ElRadio :value="0">停用</ElRadio>
              </ElRadioGroup>
            </ElFormItem>
          </ElCol>
        </ElRow>

        <div v-if="schemaFields.length > 0" class="dynamic-config">
          <ElFormItem v-for="field in schemaFields" :key="field.key" :required="field.required">
            <template #label>
              <span class="form-label-help">
                {{ field.label }}
                <ElTooltip
                  v-if="fieldHelpText(field)"
                  :content="fieldHelpText(field)"
                  placement="top"
                >
                  <ElIcon class="field-help-icon"><QuestionFilled /></ElIcon>
                </ElTooltip>
              </span>
            </template>
            <ElSwitch
              v-if="field.type === 'switch'"
              v-model="configState[field.key]"
              :disabled="submitLoading"
              @change="syncConfigToForm"
            />
            <ElSelect
              v-else-if="field.type === 'host_select'"
              v-model="configState[field.key]"
              :disabled="submitLoading"
              clearable
              filterable
              placeholder="请选择主机资产"
              style="width: 100%"
              @change="syncConfigToForm"
            >
              <ElOption
                v-for="item in hostOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
            <ElSelect
              v-else-if="field.type === 'host_multi'"
              v-model="configState[field.key]"
              :disabled="submitLoading"
              multiple
              filterable
              collapse-tags
              collapse-tags-tooltip
              placeholder="请选择主机资产"
              style="width: 100%"
              @change="syncConfigToForm"
            >
              <ElOption
                v-for="item in hostOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
            <ElInput
              v-else
              v-model="configState[field.key]"
              :disabled="submitLoading"
              :placeholder="field.placeholder || `请输入${field.label}`"
              @input="syncConfigToForm"
            />
          </ElFormItem>
        </div>

        <ElFormItem label="描述" prop="description">
          <ElInput
            v-model="formData.description"
            :disabled="submitLoading"
            type="textarea"
            :rows="3"
            placeholder="请输入渠道描述"
          />
        </ElFormItem>
        <ElFormItem label="标签" prop="labels">
          <template #label>
            <span class="form-label-help">
              标签
              <ElTooltip :content="formHelpText('labels')" placement="top">
                <ElIcon class="field-help-icon"><QuestionFilled /></ElIcon>
              </ElTooltip>
            </span>
          </template>
          <ElInput
            v-model="formData.labels"
            :disabled="submitLoading"
            type="textarea"
            :rows="3"
            placeholder="请输入 JSON 标签"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton :disabled="submitLoading" @click="dialogVisible = false">取消</ElButton>
        <ElButton :loading="testLoading" :disabled="submitLoading" @click="handleTestConfig">
          测试连接
        </ElButton>
        <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
          {{ dialogType === 'add' ? '新增' : '保存' }}
        </ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="agentDialogVisible"
      :title="`${currentAgentChannel?.name || 'Jenkins'} Agent`"
      width="980px"
      align-center
      class="jenkins-agent-dialog"
      :close-on-click-modal="false"
      @close="resetAgentForm"
    >
      <div class="agent-dialog-shell">
        <div class="agent-dialog-summary">
          <div>
            <strong>{{ currentAgentChannel?.name || 'Jenkins' }}</strong>
            <span :title="currentAgentChannel?.endpoint || currentAgentChannel?.code">
              {{ currentAgentChannel?.endpoint || currentAgentChannel?.code || '-' }}
            </span>
          </div>
          <div class="agent-count-badge">
            <b>{{ jenkinsAgents.length }}</b>
            <em>个 Agent</em>
          </div>
        </div>

        <div class="agent-layout">
          <section class="agent-list-panel" v-loading="agentLoading">
            <div class="agent-list-head">
              <div>
                <strong>Agent 列表</strong>
                <span>按渠道维护运行节点</span>
              </div>
              <ElButton type="primary" size="small" @click="openAgentForm('add')">新增</ElButton>
            </div>
            <div class="agent-list-body">
              <ElEmpty
                v-if="jenkinsAgents.length === 0"
                class="agent-empty"
                description="暂无 Agent"
              />
              <button
                v-for="item in jenkinsAgents"
                :key="item.id"
                type="button"
                class="agent-item"
                :class="{ active: agentFormId === item.id }"
                @click="openAgentForm('edit', item)"
              >
                <span class="agent-item-main">
                  <strong>{{ item.name }}</strong>
                  <em>{{ agentListSummary(item) }}</em>
                </span>
                <span class="agent-item-tags">
                  <ElTag
                    size="small"
                    :type="isDynamicAgentType(item.agentType) ? 'warning' : 'info'"
                  >
                    {{ isDynamicAgentType(item.agentType) ? '动态' : '静态' }}
                  </ElTag>
                  <ElTag size="small" :type="item.status === 1 ? 'success' : 'info'">
                    {{ item.status === 1 ? '启用' : '停用' }}
                  </ElTag>
                </span>
              </button>
            </div>
          </section>

          <section class="agent-form-panel">
            <div class="agent-form-title">
              <div>
                <strong>{{ agentFormId ? '编辑 Agent' : '新增 Agent' }}</strong>
                <span>
                  {{
                    isDynamicAgentType(agentForm.agentType)
                      ? '动态 Kubernetes 节点'
                      : 'Jenkins 静态节点'
                  }}
                </span>
              </div>
              <ElRadioGroup v-model="agentForm.status" size="small">
                <ElRadioButton :value="1">启用</ElRadioButton>
                <ElRadioButton :value="0">停用</ElRadioButton>
              </ElRadioGroup>
            </div>

            <ElForm
              ref="agentFormRef"
              :model="agentForm"
              :rules="agentRules"
              label-position="top"
              class="agent-form"
            >
              <section class="agent-form-section">
                <div class="agent-section-title">
                  <strong>基础信息</strong>
                </div>
                <div class="agent-form-grid two">
                  <ElFormItem label="名称" prop="name">
                    <ElInput v-model="agentForm.name" placeholder="例如 主节点" />
                  </ElFormItem>
                  <ElFormItem label="编码" prop="code">
                    <ElInput v-model="agentForm.code" placeholder="例如 master" />
                  </ElFormItem>
                  <ElFormItem label="Agent 类型" prop="agentType">
                    <ElSegmented
                      v-model="agentForm.agentType"
                      :options="[
                        { label: '静态节点', value: 'static' },
                        { label: '动态节点', value: 'dynamic' }
                      ]"
                      block
                    />
                  </ElFormItem>
                </div>
              </section>

              <section class="agent-form-section">
                <div class="agent-section-title">
                  <strong>匹配规则</strong>
                </div>
                <div class="agent-form-grid two">
                  <ElFormItem label="匹配方式" prop="matchMode">
                    <ElSelect v-model="agentForm.matchMode" style="width: 100%">
                      <ElOption label="节点名称" value="name" />
                      <ElOption label="节点标签" value="label" />
                    </ElSelect>
                  </ElFormItem>
                  <ElFormItem label="匹配值" prop="matchValue">
                    <ElInput v-model="agentForm.matchValue" placeholder="例如 master" />
                  </ElFormItem>
                </div>
              </section>

              <section v-if="isDynamicAgentType(agentForm.agentType)" class="agent-form-section">
                <div class="agent-section-title">
                  <strong>动态节点配置</strong>
                  <ElButton size="small" text type="primary" @click="addAgentContainer">
                    新增容器
                  </ElButton>
                </div>
                <ElFormItem label="K8s 云名称">
                  <ElInput v-model="agentForm.cloud" placeholder="例如 my-k8s-cluster" />
                </ElFormItem>
                <div class="agent-container-list">
                  <div
                    v-for="(container, index) in agentForm.containers"
                    :key="index"
                    class="agent-container-row"
                  >
                    <ElInput v-model="container.name" placeholder="容器名称，例如 maven" />
                    <ElInput v-model="container.image" placeholder="镜像，可选" />
                    <ElButton text type="danger" @click="removeAgentContainer(index)">
                      删除
                    </ElButton>
                  </div>
                </div>
                <ElFormItem label="Pod YAML">
                  <ElInput
                    v-model="agentForm.podYaml"
                    type="textarea"
                    :rows="8"
                    placeholder="可选。填写后 Jenkinsfile 会使用 Kubernetes Agent YAML 模板。"
                  />
                </ElFormItem>
              </section>
            </ElForm>

            <div class="agent-form-actions">
              <ElButton
                v-if="agentFormId"
                type="danger"
                plain
                :loading="agentSubmitLoading"
                @click="deleteAgent"
              >
                删除
              </ElButton>
              <span />
              <ElButton :disabled="agentSubmitLoading" @click="resetAgentForm">重置</ElButton>
              <ElButton type="primary" :loading="agentSubmitLoading" @click="submitAgentForm">
                {{ agentFormId ? '保存' : '新增' }}
              </ElButton>
            </div>
          </section>
        </div>
      </div>
    </ElDialog>

    <ElDialog
      v-model="testResultDialogVisible"
      title="连接测试结果"
      width="560px"
      align-center
      class="channel-test-dialog"
    >
      <div
        v-if="channelTestResult"
        :class="['channel-test-result', channelTestResult.success ? 'is-success' : 'is-error']"
      >
        <div class="test-result-head">
          <div>
            <strong>{{ channelTestResult.success ? '连接正常' : '连接异常' }}</strong>
            <p :title="channelTestResult.message">{{ channelTestResult.message }}</p>
          </div>
          <ElTag :type="channelTestResult.success ? 'success' : 'danger'">
            {{ channelTestResult.success ? '正常' : '异常' }}
          </ElTag>
        </div>
        <div class="test-result-list">
          <div v-for="row in channelTestRows" :key="row.label">
            <span>{{ row.label }}</span>
            <strong :title="row.value">{{ row.value }}</strong>
          </div>
        </div>
      </div>
      <template #footer>
        <ElButton type="primary" @click="testResultDialogVisible = false">确定</ElButton>
      </template>
    </ElDialog>

    <CredentialCreateDialog
      v-model="credentialDialogVisible"
      title="新增凭据"
      :type-options="credentialTypeOptionsForSelected"
      :default-type="firstCredentialType()"
      :channel-group-code="selectedChannelGroupCode"
      :channel-type="formData.channelType"
      @created="handleCredentialCreated"
    />
  </div>
</template>

<script setup lang="ts">
  import {
    computed,
    nextTick,
    onActivated,
    onBeforeUnmount,
    onMounted,
    reactive,
    ref,
    watch
  } from 'vue'
  import { storeToRefs } from 'pinia'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import { Delete, Edit, Plus, QuestionFilled } from '@element-plus/icons-vue'
  import { echarts, type EChartsOption } from '@/plugins/echarts'
  import { formatTimestamp } from '@/utils/format'
  import { useDevopsProjectStore } from '@/store/modules/devopsProject'
  import { useUserStore } from '@/store/modules/user'
  import {
    createDevopsChannelApi,
    createDevopsJenkinsAgentApi,
    deleteDevopsChannelApi,
    deleteDevopsJenkinsAgentApi,
    listDevopsChannelApi,
    listDevopsChannelGroupApi,
    listDevopsChannelTypeApi,
    listDevopsCredentialApi,
    listDevopsHostApi,
    listDevopsJenkinsAgentApi,
    syncDevopsTektonChannelStepsApi,
    testDevopsChannelApi,
    testExistingDevopsChannelApi,
    updateDevopsChannelApi,
    updateDevopsJenkinsAgentApi,
    type DevopsChannel,
    type DevopsChannelForm,
    type DevopsChannelGroup,
    type DevopsChannelType,
    type DevopsCredential,
    type DevopsHost,
    type DevopsJenkinsAgent,
    type DevopsJenkinsAgentForm
  } from '@/api/devops/channel'
  import CredentialCreateDialog from '@/views/devops/components/CredentialCreateDialog.vue'

  defineOptions({ name: 'DevopsChannelCenter' })

  interface SchemaField {
    key: string
    label: string
    type?: string
    required?: boolean
    placeholder?: string
    defaultValue?: any
  }

  interface ChannelTestResultView {
    success: boolean
    message: string
    checkedAt: number
    productName?: string
    version?: string
    productVersion?: string
    engineVersion?: string
    apiVersion?: string
    authUser?: string
    capabilities?: string
  }

  type ChannelCardItem = DevopsChannel | Record<string, any>

  const asDevopsChannel = (channel: ChannelCardItem) => channel as DevopsChannel

  const loading = ref(false)
  const submitLoading = ref(false)
  const testLoading = ref(false)
  const testingChannelId = ref('')
  const syncingTektonChannelId = ref('')
  const showSearchBar = ref(true)
  const errorMessage = ref('')
  const searchRef = ref()
  const formRef = ref<FormInstance>()
  const agentFormRef = ref<FormInstance>()
  const dialogVisible = ref(false)
  const credentialDialogVisible = ref(false)
  const testResultDialogVisible = ref(false)
  const agentDialogVisible = ref(false)
  const agentLoading = ref(false)
  const agentSubmitLoading = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const currentId = ref('')
  const currentAgentChannel = ref<DevopsChannel | null>(null)
  const agentFormId = ref('')
  const channelData = ref<DevopsChannel[]>([])
  const jenkinsAgents = ref<DevopsJenkinsAgent[]>([])
  const groupData = ref<DevopsChannelGroup[]>([])
  const channelTypes = ref<DevopsChannelType[]>([])
  const credentials = ref<DevopsCredential[]>([])
  const hosts = ref<DevopsHost[]>([])
  const searchParams = ref<Record<string, any>>({})
  const channelTestResult = ref<ChannelTestResultView | null>(null)
  const configState = reactive<Record<string, any>>({})
  const healthChartRef = ref<HTMLDivElement>()
  const typeChartRef = ref<HTMLDivElement>()
  let healthChart: ReturnType<typeof echarts.init> | null = null
  let typeChart: ReturnType<typeof echarts.init> | null = null
  const userStore = useUserStore()
  const devopsProjectStore = useDevopsProjectStore()
  const { selectedProjectId } = storeToRefs(devopsProjectStore)

  const searchForm = ref({
    name: undefined,
    code: undefined,
    groupId: undefined,
    channelType: undefined,
    status: undefined
  })

  const pagination = ref({
    current: 1,
    size: 15,
    total: 0
  })

  const defaultForm = (): DevopsChannelForm => ({
    groupId: '',
    name: '',
    code: '',
    channelType: '',
    endpoint: '',
    description: '',
    globalCredentialId: '',
    credentialId: '',
    config: '{}',
    labels: '',
    authType: 'none',
    username: '',
    password: '',
    token: '',
    insecureSkipTls: false,
    icon: '',
    iconColor: '',
    status: 1
  })

  const formData = reactive<DevopsChannelForm>(defaultForm())

  const normalizeAgentType = (type?: string): DevopsJenkinsAgentForm['agentType'] =>
    type === 'dynamic' || type === 'pod' ? 'dynamic' : 'static'

  const isDynamicAgentType = (type?: string) => normalizeAgentType(type) === 'dynamic'

  const defaultAgentForm = (): DevopsJenkinsAgentForm => ({
    channelId: '',
    name: '',
    code: '',
    agentType: 'static',
    matchMode: 'name',
    matchValue: '',
    cloud: '',
    podYaml: '',
    containers: [{ name: 'jnlp' }, { name: 'git' }, { name: 'go' }, { name: 'maven' }],
    status: 1
  })

  const agentForm = reactive<DevopsJenkinsAgentForm>(defaultAgentForm())

  const rules: FormRules = {
    groupId: [{ required: true, message: '请选择渠道分组', trigger: 'change' }],
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    code: [{ required: true, message: '请输入编码', trigger: 'blur' }],
    channelType: [{ required: true, message: '请选择渠道类型', trigger: 'change' }],
    endpoint: [{ required: true, message: '请输入访问地址', trigger: 'blur' }]
  }

  const agentRules: FormRules = {
    name: [{ required: true, message: '请输入 Agent 名称', trigger: 'blur' }],
    agentType: [{ required: true, message: '请选择 Agent 类型', trigger: 'change' }],
    matchMode: [{ required: true, message: '请选择匹配方式', trigger: 'change' }],
    matchValue: [{ required: true, message: '请输入匹配值', trigger: 'blur' }]
  }

  const statusOptions = [
    { label: '启用', value: 1 },
    { label: '停用', value: 0 }
  ]

  const credentialTypeOptions = [
    { label: '用户名密码', value: 'username_password' },
    { label: 'Token', value: 'token' },
    { label: 'SSH Key', value: 'ssh_key' },
    { label: 'Kubeconfig', value: 'kubeconfig' },
    { label: 'Secret Text', value: 'secret_text' },
    { label: '证书', value: 'certificate' },
    { label: 'JSON', value: 'json' }
  ]
  const baseChannelCredentialTypes = ['username_password', 'token', 'json']

  const channelVisualMap: Record<string, { icon: string; color: string }> = {
    jenkins: { icon: 'simple-icons:jenkins', color: '#d24939' },
    tekton: { icon: 'ri:flow-chart', color: '#2563eb' },
    buildkit: { icon: 'simple-icons:docker', color: '#0369a1' },
    gitlab: { icon: 'simple-icons:gitlab', color: '#fc6d26' },
    github: { icon: 'simple-icons:github', color: '#24292f' },
    gitee: { icon: 'simple-icons:gitee', color: '#c71d23' },
    svn: { icon: 'ri:git-repository-line', color: '#64748b' },
    nexus: { icon: 'ri:archive-stack-line', color: '#16a34a' },
    jfrog: { icon: 'simple-icons:jfrog', color: '#0f9f6e' },
    harbor: { icon: 'simple-icons:harbor', color: '#0f766e' },
    registry: { icon: 'simple-icons:docker', color: '#0369a1' },
    aliyun_registry: { icon: 'ri:cloud-line', color: '#ff6a00' },
    sonarqube: { icon: 'simple-icons:sonarqube', color: '#4b9fd5' },
    spotbugs: { icon: 'ri:bug-line', color: '#b45309' },
    trivy: { icon: 'ri:shield-check-line', color: '#0f766e' },
    kube_bench: { icon: 'ri:shield-keyhole-line', color: '#7c3aed' },
    kubernetes: { icon: 'simple-icons:kubernetes', color: '#326ce5' },
    host: { icon: 'ri:server-line', color: '#475569' },
    host_group: { icon: 'ri:server-line', color: '#7c3aed' },
    'kube-nova': { icon: 'ri:cloud-line', color: '#0891b2' },
    kubenova: { icon: 'ri:cloud-line', color: '#0891b2' },
    argocd: { icon: 'simple-icons:argo', color: '#ef7b4d' }
  }

  const iconOptions = [
    ...Object.values(channelVisualMap),
    { icon: 'simple-icons:npm', color: '#cb3837' },
    { icon: 'skill-icons:python-dark', color: '#3776ab' },
    { icon: 'skill-icons:golang', color: '#00add8' },
    { icon: 'skill-icons:java-dark', color: '#f89820' },
    { icon: 'ri:terminal-box-line', color: '#475569' },
    { icon: 'ri:tools-line', color: '#7c3aed' }
  ]

  const isSuperAdmin = computed(() =>
    (userStore.getUserInfo.roles || []).some((role) => String(role).toUpperCase() === 'SUPER_ADMIN')
  )

  const groupOptions = computed(() =>
    groupData.value
      .filter((item) => item.status === 1)
      .map((item) => ({ label: item.name, value: item.id }))
  )

  const groupNameMap = computed(() =>
    groupData.value.reduce(
      (acc, item) => {
        acc[item.id] = item.name
        return acc
      },
      {} as Record<string, string>
    )
  )

  const groupCodeById = (id?: string) => groupData.value.find((item) => item.id === id)?.code || ''
  const selectedSearchGroupCode = computed(() => groupCodeById(searchForm.value.groupId))
  const selectedGroup = computed(() => groupData.value.find((item) => item.id === formData.groupId))
  const selectedChannelGroupCode = computed(() => selectedGroup.value?.code || '')

  const selectedChannelType = computed(() =>
    channelTypes.value.find(
      (item) =>
        item.code === formData.channelType && item.groupCode === selectedChannelGroupCode.value
    )
  )

  const searchChannelTypeOptions = computed(() =>
    channelTypes.value
      .filter((item) => item.status === 1 && item.groupCode === selectedSearchGroupCode.value)
      .map((item) => ({ label: item.name, value: item.code }))
  )

  const filteredChannelTypeOptions = computed(() => {
    const group = selectedGroup.value
    if (!group) return []
    const allowed = group.allowedChannelTypes || []
    return channelTypes.value
      .filter((item) => item.status === 1 && item.groupCode === group.code)
      .filter((item) => allowed.length === 0 || allowed.includes(item.code))
      .map((item) => ({ label: item.name, value: item.code }))
  })

  const credentialTypeOptionsForSelected = computed(() => {
    const allowed = channelCredentialTypes(selectedChannelType.value?.credentialTypes)
    const options = allowed.length
      ? credentialTypeOptions.filter((item) => allowed.includes(item.value))
      : credentialTypeOptions
    return options.length ? options : credentialTypeOptions
  })

  const filteredCredentialOptions = computed(() => {
    const allowed = channelCredentialTypes(selectedChannelType.value?.credentialTypes)
    return credentials.value
      .filter((item) => item.status === 1)
      .filter((item) => allowed.length === 0 || allowed.includes(item.credentialType))
      .map((item) => ({
        label: `${item.name}（${credentialTypeLabel(item.credentialType)}）`,
        value: item.id
      }))
  })

  const hostOptions = computed(() =>
    hosts.value
      .filter((item) => item.status === 1)
      .map((item) => ({ label: `${item.name}（${item.ip}:${item.port}）`, value: item.id }))
  )

  const schemaFields = computed<SchemaField[]>(() => parseSchema(selectedChannelType.value))

  const summary = computed(() => ({
    total: pagination.value.total,
    enabled: channelData.value.filter((item) => item.status === 1).length,
    healthy: channelData.value.filter((item) => item.healthStatus === 'healthy').length,
    unhealthy: channelData.value.filter((item) => item.healthStatus === 'unhealthy').length
  }))

  const channelTestRows = computed(() => {
    const result = channelTestResult.value
    if (!result) return []
    return [
      { label: '产品名称', value: result.productName || '-' },
      { label: '版本', value: result.version || '-' },
      { label: 'API 版本', value: result.apiVersion || '-' },
      { label: '认证用户', value: result.authUser || '-' },
      { label: '能力信息', value: result.capabilities || '-' },
      { label: '检测时间', value: formatTimestamp(result.checkedAt) }
    ]
  })

  const searchItems = computed(() => [
    { label: '名称', key: 'name', type: 'input', props: { clearable: true } },
    { label: '编码', key: 'code', type: 'input', props: { clearable: true } },
    {
      label: '渠道分组',
      key: 'groupId',
      type: 'select',
      props: {
        clearable: true,
        options: groupOptions.value,
        onChange: handleSearchGroupChange,
        onVisibleChange: handleSearchGroupVisibleChange
      }
    },
    {
      label: '渠道类型',
      key: 'channelType',
      type: 'select',
      props: {
        clearable: true,
        disabled: !searchForm.value.groupId,
        options: searchChannelTypeOptions.value,
        onVisibleChange: handleSearchChannelTypeVisibleChange
      }
    },
    {
      label: '状态',
      key: 'status',
      type: 'select',
      props: { clearable: true, options: statusOptions }
    }
  ])

  function firstCredentialType() {
    return (
      channelCredentialTypes(selectedChannelType.value?.credentialTypes)[0] || 'username_password'
    )
  }

  const channelCredentialTypes = (types?: string[]) => {
    const values = (types || []).filter(Boolean)
    return Array.from(new Set(values.length > 0 ? values : baseChannelCredentialTypes))
  }

  const tektonTaskNamespaceField: SchemaField = {
    key: 'taskNamespace',
    label: 'Task 命名空间',
    type: 'input',
    required: true,
    defaultValue: 'tekton',
    placeholder: '请输入已存在的 Task 命名空间'
  }

  function parseSchema(channelType?: DevopsChannelType): SchemaField[] {
    if (!channelType?.configSchema) {
      return channelType?.code === 'tekton' ? [tektonTaskNamespaceField] : []
    }
    try {
      const parsed = JSON.parse(channelType.configSchema)
      const fields = Array.isArray(parsed.fields) ? parsed.fields : []
      if (channelType.code === 'tekton' && !fields.some((item) => item.key === 'taskNamespace')) {
        return [tektonTaskNamespaceField, ...fields]
      }
      return fields
    } catch {
      return channelType.code === 'tekton' ? [tektonTaskNamespaceField] : []
    }
  }

  function parseConfig(config?: string) {
    if (!config) return {}
    try {
      return JSON.parse(config)
    } catch {
      return {}
    }
  }

  const formHelpText = (key: string) => {
    const texts: Record<string, string> = {
      code: '渠道实例唯一编码，创建后用于项目绑定和自动化引用，建议使用小写英文、数字和下划线。',
      groupId: '渠道实例必须挂在一个渠道分组下，分组会限制可选渠道类型。',
      channelType: '渠道类型决定动态配置表单、凭据类型和连接测试逻辑。',
      endpoint:
        formData.channelType === 'tekton'
          ? 'Tekton 通过 Kubernetes API 访问，Kubeconfig 凭据可自动读取地址，Token 凭据需要填写 API Server 地址。'
          : '服务访问地址，例如 Jenkins、GitLab、Nexus、Kubernetes API Server 地址。',
      credentialId:
        formData.channelType === 'tekton'
          ? '连接该渠道使用的凭据，Tekton 建议使用 Kubeconfig 或 Token 凭据。'
          : '连接该渠道使用的凭据，敏感信息保存在凭据中心，渠道只保存凭据引用。',
      insecureSkipTls: '仅在自签名证书或内网测试环境使用，开启后会跳过 HTTPS 证书校验。',
      labels: 'JSON 标签，用于后续检索、分组和自动化匹配，例如 {"env":"dev"}。'
    }
    return texts[key] || ''
  }

  const fieldHelpText = (field: SchemaField) => {
    const texts: Record<string, string> = {
      jobName: 'Jenkins 默认任务名，后续触发构建时用于定位 Job。',
      crumbIssuer:
        'Jenkins 开启 CSRF 防护时，写接口需要先获取 Crumb 并随请求提交；仅测试连接通常可关闭。',
      apiPath: '接口基础路径，GitLab 默认是 /api/v4。',
      repository: '默认仓库名称，用于制品仓库或镜像仓库的后续操作。',
      namespace: '默认命名空间，用于 Kubernetes、镜像仓库或部署目标。',
      taskNamespace:
        'Tekton taskNamespace。平台维护的 Task 会统一同步到这里，流水线通过 cluster resolver 引用。',
      projectKey: '扫描服务中的项目唯一标识，例如 SonarQube Project Key。',
      hostId: '选择一个主机资产作为当前主机渠道实例。',
      hostIds: '选择多个主机资产组成主机组渠道。'
    }
    return texts[field.key] || field.placeholder || ''
  }

  const channelTypeOption = (type: string, groupId?: string) =>
    channelTypes.value.find(
      (item) => item.code === type && (!groupId || item.groupCode === groupCodeById(groupId))
    )

  const channelTypeLabel = (type: string, groupId?: string) =>
    channelTypeOption(type, groupId)?.name || type
  const channelIcon = (channel: ChannelCardItem) => {
    const data = asDevopsChannel(channel)
    return (
      data.icon ||
      channelTypeOption(data.channelType, data.groupId)?.icon ||
      channelVisualMap[data.channelType]?.icon ||
      'ri:apps-2-line'
    )
  }
  const channelColor = (channel: ChannelCardItem) => {
    const data = asDevopsChannel(channel)
    return (
      data.iconColor ||
      channelTypeOption(data.channelType, data.groupId)?.iconColor ||
      channelVisualMap[data.channelType]?.color ||
      '#475569'
    )
  }
  const credentialTypeLabel = (type: string) =>
    credentialTypeOptions.find((item) => item.value === type)?.label || type || '无凭据'

  const healthTagType = (status: string) => {
    if (status === 'healthy') return 'success'
    if (status === 'unhealthy') return 'danger'
    return 'info'
  }

  const credentialLabel = (channel: ChannelCardItem) => {
    const data = asDevopsChannel(channel)
    const credential = credentials.value.find((item) => item.id === data.credentialId)
    if (credential) return `${credential.name}（${credentialTypeLabel(credential.credentialType)}）`
    return data.authType && data.authType !== 'none'
      ? `兼容字段（${credentialTypeLabel(data.authType)}）`
      : '无凭据'
  }

  const configSummary = (channel: ChannelCardItem) => {
    const data = asDevopsChannel(channel)
    const parsed = parseConfig(data.config)
    if (data.channelType === 'tekton') return `Task NS: ${parsed.taskNamespace || '-'}`
    if (Array.isArray(parsed.hostIds)) return `${parsed.hostIds.length} 台主机`
    if (parsed.hostId) return hostName(parsed.hostId)
    const keys = Object.keys(parsed)
    return keys.length ? keys.slice(0, 2).join('、') : '默认'
  }

  const metadataSummary = (channel: ChannelCardItem) => {
    const data = asDevopsChannel(channel)
    const metadata = parseConfig(data.metadata)
    if (isGitLabWithoutProductVersion(metadata)) return '待获取'
    return metadataVersion(metadata) || '待检测'
  }

  const isGitLabWithoutProductVersion = (metadata: Record<string, any>) =>
    String(metadata.productName || '').toLowerCase() === 'gitlab' && !metadata.productVersion

  const apiVersionAsVersion = (metadata: Record<string, any>) =>
    ['docker registry', 'github', 'gitee', 'tekton'].includes(
      String(metadata.productName || '').toLowerCase()
    )
      ? metadata.apiVersion
      : ''

  const metadataVersion = (metadata: Record<string, any>) =>
    metadata.version ||
    metadata.productVersion ||
    metadata.engineVersion ||
    metadata.schemaVersion ||
    metadata.benchmarkProfile ||
    apiVersionAsVersion(metadata)

  const hostName = (id: string) => {
    const host = hosts.value.find((item) => item.id === id)
    return host ? `${host.name} ${host.ip}:${host.port}` : id
  }

  const pickParams = () =>
    Object.entries(searchForm.value).reduce(
      (acc, [key, value]) => {
        if (value !== undefined && value !== null && value !== '') acc[key] = value
        return acc
      },
      {} as Record<string, any>
    )

  const getGroups = async () => {
    const res = await listDevopsChannelGroupApi({ page: 1, pageSize: 200, status: -1 })
    groupData.value = res.items || []
  }

  const mergeChannelTypes = (items: DevopsChannelType[]) => {
    const data = new Map(channelTypes.value.map((item) => [`${item.groupCode}:${item.code}`, item]))
    items.forEach((item) => data.set(`${item.groupCode}:${item.code}`, item))
    channelTypes.value = Array.from(data.values())
  }

  const getChannelTypesByGroup = async (groupCode?: string) => {
    const code = String(groupCode || '').trim()
    if (!code) return []
    const res = await listDevopsChannelTypeApi({
      page: 1,
      pageSize: 200,
      status: -1,
      groupCode: code
    })
    const items = res.items || []
    mergeChannelTypes(items)
    return items
  }

  const getChannelTypesForChannels = async (items: DevopsChannel[]) => {
    const groupCodes = Array.from(
      new Set(items.map((item) => groupCodeById(item.groupId)).filter(Boolean))
    )
    await Promise.all(groupCodes.map((groupCode) => getChannelTypesByGroup(groupCode)))
  }

  const getCredentials = async () => {
    const baseParams = { page: 1, pageSize: 200, status: -1 }
    if (!selectedChannelGroupCode.value || !formData.channelType) {
      credentials.value = []
      return
    }
    const channelParams = {
      channelGroupCode: selectedChannelGroupCode.value,
      channelType: formData.channelType
    }
    if (!isSuperAdmin.value && !selectedProjectId.value) {
      credentials.value = []
      return
    }
    const requests = isSuperAdmin.value
      ? [
          listDevopsCredentialApi({ ...baseParams, ...channelParams, scope: 'system' }),
          ...(selectedProjectId.value
            ? [
                listDevopsCredentialApi({
                  ...baseParams,
                  ...channelParams,
                  scope: 'project',
                  projectId: selectedProjectId.value
                })
              ]
            : [])
        ]
      : [
          listDevopsCredentialApi({
            ...baseParams,
            ...channelParams,
            scope: 'project',
            projectId: selectedProjectId.value
          })
        ]
    const results = await Promise.all(requests)
    const merged = new Map<string, DevopsCredential>()
    results.forEach((res) => {
      ;(res.items || []).forEach((item) => merged.set(item.id, item))
    })
    credentials.value = Array.from(merged.values())
  }

  const getHosts = async () => {
    const res = await listDevopsHostApi({ page: 1, pageSize: 200, status: -1 })
    hosts.value = res.items || []
  }

  const getChannels = async () => {
    loading.value = true
    errorMessage.value = ''
    try {
      const res = await listDevopsChannelApi({
        page: pagination.value.current,
        pageSize: pagination.value.size,
        ...searchParams.value
      })
      const rawItems = res.items || []
      const items = rawItems.filter((item) => !item.isSystem)
      channelData.value = items
      pagination.value.total = rawItems.length === items.length ? res.total || 0 : items.length
      await getChannelTypesForChannels(items)
      renderCharts()
    } catch (error: any) {
      errorMessage.value = error?.message || '渠道列表加载失败'
      channelData.value = []
      pagination.value.total = 0
    } finally {
      loading.value = false
    }
  }

  const refreshAll = async () => {
    if (!devopsProjectStore.initialized) {
      await devopsProjectStore.fetchProjects()
    }
    await Promise.all([getGroups(), getHosts()])
    await getChannels()
    await getCredentials()
  }

  const handleSearch = async () => {
    await searchRef.value?.validate?.()
    searchParams.value = pickParams()
    pagination.value.current = 1
    getChannels()
  }

  const handleReset = () => {
    searchParams.value = {}
    pagination.value.current = 1
    getChannels()
  }

  const handleSearchGroupChange = async () => {
    searchForm.value.channelType = undefined
    await getChannelTypesByGroup(selectedSearchGroupCode.value)
  }

  const handleSearchGroupVisibleChange = async (visible: boolean) => {
    if (visible) {
      await getGroups()
    }
  }

  const handleSearchChannelTypeVisibleChange = async (visible: boolean) => {
    if (visible) {
      await getChannelTypesByGroup(selectedSearchGroupCode.value)
    }
  }

  const handleSizeChange = (size: number) => {
    pagination.value.size = size
    pagination.value.current = 1
    getChannels()
  }

  const handleCurrentChange = (current: number) => {
    pagination.value.current = current
    getChannels()
  }

  const handleGroupChange = async () => {
    await getChannelTypesByGroup(selectedChannelGroupCode.value)
    const first = filteredChannelTypeOptions.value[0]
    if (!first) {
      formData.channelType = ''
      credentials.value = []
      return
    }
    if (!filteredChannelTypeOptions.value.some((item) => item.value === formData.channelType)) {
      formData.channelType = first.value
      handleChannelTypeChange()
    }
    getCredentials()
  }

  const handleChannelTypeChange = () => {
    formData.credentialId = ''
    if (!formData.endpoint || dialogType.value === 'add') {
      formData.endpoint = defaultEndpoint(formData.channelType)
    }
    if (dialogType.value === 'add') {
      const typeInfo = selectedChannelType.value
      const fallback = channelVisualMap[formData.channelType]
      formData.icon = typeInfo?.icon || fallback?.icon || ''
      formData.iconColor = typeInfo?.iconColor || fallback?.color || ''
    }
    syncConfigState()
    getCredentials()
  }

  const defaultEndpoint = (type: string) => {
    if (type === 'host') return 'system://host'
    if (type === 'host_group') return 'system://host-group'
    if (type === 'trivy') return 'system://trivy'
    if (type === 'spotbugs') return 'system://spotbugs'
    if (type === 'kube_bench') return 'system://kube-bench'
    if (type === 'tekton') return 'system://tekton'
    if (type === 'kubernetes') return 'https://'
    return 'https://'
  }

  const syncTektonSteps = async (row: ChannelCardItem) => {
    const channel = asDevopsChannel(row)
    if (!channel.id) return
    syncingTektonChannelId.value = channel.id
    try {
      await syncDevopsTektonChannelStepsApi(channel.id)
      ElMessage.success('Tekton 步骤同步已完成')
      await getChannels()
    } finally {
      syncingTektonChannelId.value = ''
    }
  }

  const syncConfigState = () => {
    Object.keys(configState).forEach((key) => delete configState[key])
    const parsed = parseConfig(formData.config)
    schemaFields.value.forEach((field) => {
      if (parsed[field.key] !== undefined) {
        configState[field.key] = parsed[field.key]
        return
      }
      if (field.type === 'host_multi') {
        configState[field.key] = []
      } else if (field.type === 'switch') {
        configState[field.key] = Boolean(field.defaultValue)
      } else {
        configState[field.key] = field.defaultValue ?? ''
      }
    })
    syncConfigToForm()
  }

  const syncConfigToForm = () => {
    formData.config = JSON.stringify(configState)
  }

  const validateDynamicConfig = () => {
    const emptyField = schemaFields.value.find(
      (field) =>
        field.required &&
        (configState[field.key] === undefined ||
          configState[field.key] === null ||
          configState[field.key] === '' ||
          (Array.isArray(configState[field.key]) && configState[field.key].length === 0))
    )
    if (emptyField) {
      ElMessage.warning(`请填写${emptyField.label}`)
      return false
    }
    return true
  }

  const openDialog = async (type: 'add' | 'edit', row?: ChannelCardItem) => {
    const channel = row ? asDevopsChannel(row) : undefined
    dialogType.value = type
    currentId.value = channel?.id || ''
    Object.assign(formData, defaultForm(), channel || {})
    await Promise.all([getGroups(), getHosts()])
    await getChannelTypesByGroup(selectedChannelGroupCode.value || groupCodeById(formData.groupId))
    await getCredentials()
    if (type === 'add' && !formData.groupId && groupOptions.value.length > 0) {
      formData.groupId = groupOptions.value[0].value
      await handleGroupChange()
    }
    dialogVisible.value = true
    await nextTick()
    syncConfigState()
  }

  const resetForm = () => {
    currentId.value = ''
    Object.assign(formData, defaultForm())
    Object.keys(configState).forEach((key) => delete configState[key])
    formRef.value?.clearValidate()
  }

  const handleSubmit = async () => {
    await formRef.value?.validate()
    syncConfigToForm()
    if (!validateDynamicConfig()) return
    submitLoading.value = true
    try {
      if (dialogType.value === 'add') {
        await createDevopsChannelApi(formData)
        ElMessage.success('新增成功')
      } else {
        await updateDevopsChannelApi(currentId.value, formData)
        ElMessage.success('保存成功')
      }
      dialogVisible.value = false
      await getChannels()
    } finally {
      submitLoading.value = false
    }
  }

  const loadJenkinsAgents = async () => {
    if (!currentAgentChannel.value?.id) {
      jenkinsAgents.value = []
      return
    }
    agentLoading.value = true
    try {
      const res = await listDevopsJenkinsAgentApi({
        page: 1,
        pageSize: 200,
        channelId: currentAgentChannel.value.id,
        status: -1
      })
      jenkinsAgents.value = res.items || []
    } finally {
      agentLoading.value = false
    }
  }

  const agentListSummary = (item: DevopsJenkinsAgent) => {
    const matchLabel = item.matchMode === 'name' ? '名称' : '标签'
    const containerCount = isDynamicAgentType(item.agentType)
      ? ` / ${item.containers?.length || 0} 容器`
      : ''
    return `${matchLabel}: ${item.matchValue || '-'}${containerCount}`
  }

  const openAgentDialog = async (row: ChannelCardItem) => {
    currentAgentChannel.value = asDevopsChannel(row)
    agentDialogVisible.value = true
    resetAgentForm()
    await loadJenkinsAgents()
  }

  const openAgentForm = (type: 'add' | 'edit', row?: DevopsJenkinsAgent) => {
    agentFormId.value = type === 'edit' ? row?.id || '' : ''
    Object.assign(agentForm, defaultAgentForm(), {
      channelId: currentAgentChannel.value?.id || '',
      ...(row || {})
    })
    agentForm.agentType = normalizeAgentType(agentForm.agentType)
    if (
      isDynamicAgentType(agentForm.agentType) &&
      (!agentForm.containers || agentForm.containers.length === 0)
    ) {
      agentForm.containers = defaultAgentForm().containers
    }
    nextTick(() => agentFormRef.value?.clearValidate())
  }

  const resetAgentForm = () => {
    agentFormId.value = ''
    Object.assign(agentForm, defaultAgentForm(), {
      channelId: currentAgentChannel.value?.id || ''
    })
    agentFormRef.value?.clearValidate()
  }

  const addAgentContainer = () => {
    if (!agentForm.containers) agentForm.containers = []
    agentForm.containers.push({ name: '', image: '' })
  }

  const removeAgentContainer = (index: number) => {
    agentForm.containers?.splice(index, 1)
  }

  const normalizedAgentForm = (): DevopsJenkinsAgentForm => {
    const agentType = normalizeAgentType(agentForm.agentType)
    return {
      ...agentForm,
      agentType,
      channelId: currentAgentChannel.value?.id || agentForm.channelId,
      containers: isDynamicAgentType(agentType)
        ? (agentForm.containers || []).filter((item) => String(item.name || '').trim())
        : []
    }
  }

  const submitAgentForm = async () => {
    await agentFormRef.value?.validate()
    agentSubmitLoading.value = true
    try {
      const payload = normalizedAgentForm()
      if (agentFormId.value) {
        await updateDevopsJenkinsAgentApi(agentFormId.value, payload)
        ElMessage.success('Agent 已保存')
      } else {
        await createDevopsJenkinsAgentApi(payload)
        ElMessage.success('Agent 已新增')
      }
      resetAgentForm()
      await loadJenkinsAgents()
    } finally {
      agentSubmitLoading.value = false
    }
  }

  const deleteAgent = async () => {
    if (!agentFormId.value) return
    await ElMessageBox.confirm(`确定删除 Agent「${agentForm.name}」吗？`, '删除确认', {
      type: 'warning'
    })
    agentSubmitLoading.value = true
    try {
      await deleteDevopsJenkinsAgentApi(agentFormId.value)
      ElMessage.success('Agent 已删除')
      resetAgentForm()
      await loadJenkinsAgents()
    } finally {
      agentSubmitLoading.value = false
    }
  }

  const buildTestPayload = () => {
    syncConfigToForm()
    return {
      id: currentId.value || undefined,
      groupId: formData.groupId,
      name: formData.name,
      code: formData.code,
      channelType: formData.channelType,
      endpoint: formData.endpoint,
      authType: formData.authType,
      username: formData.username,
      password: formData.password,
      token: formData.token,
      insecureSkipTls: formData.insecureSkipTls,
      config: formData.config,
      credentialId: formData.credentialId
    }
  }

  const handleTestConfig = async () => {
    if (!formData.channelType || !formData.endpoint) {
      ElMessage.warning('请先填写渠道类型和访问地址')
      return
    }
    syncConfigToForm()
    if (!validateDynamicConfig()) return
    testLoading.value = true
    try {
      const result = await testDevopsChannelApi(buildTestPayload())
      await showTestResult(result)
    } finally {
      testLoading.value = false
    }
  }

  const handleTestExisting = async (row: ChannelCardItem) => {
    const channel = asDevopsChannel(row)
    testingChannelId.value = channel.id
    try {
      const result = await testExistingDevopsChannelApi(channel.id)
      await showTestResult(result)
      await getChannels()
    } finally {
      testingChannelId.value = ''
    }
  }

  const normalizeTestResult = (result: {
    success: boolean
    message: string
    checkedAt: number
    metadata?: string
  }): ChannelTestResultView => {
    const metadata = parseConfig(result.metadata)
    let capabilities = ''
    if (Array.isArray(metadata.capabilities)) {
      capabilities = metadata.capabilities.join('、')
    } else if (metadata.capabilities && typeof metadata.capabilities === 'object') {
      capabilities =
        metadata.capabilities.system ||
        Object.entries(metadata.capabilities)
          .map(([key, value]) => `${key}: ${value}`)
          .join('、')
    } else {
      capabilities = metadata.capabilities || ''
    }
    return {
      success: result.success,
      message: result.message,
      checkedAt: result.checkedAt,
      productName: metadata.productName,
      version: isGitLabWithoutProductVersion(metadata) ? '待获取' : metadataVersion(metadata),
      productVersion: metadata.productVersion,
      engineVersion: metadata.engineVersion,
      apiVersion: metadata.apiVersion,
      authUser: metadata.authUser,
      capabilities
    }
  }

  const showTestResult = (result: {
    success: boolean
    message: string
    checkedAt: number
    metadata?: string
  }) => {
    channelTestResult.value = normalizeTestResult(result)
    testResultDialogVisible.value = true
  }

  const deleteChannel = async (row: ChannelCardItem) => {
    const channel = asDevopsChannel(row)
    if (channel.isSystem) {
      ElMessage.warning('系统渠道不能删除')
      return
    }
    await ElMessageBox.confirm(`确定删除渠道"${channel.name}"吗？`, '删除确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteDevopsChannelApi(channel.id)
    ElMessage.success('删除成功')
    await getChannels()
  }

  const openCredentialDialog = () => {
    if (!isSuperAdmin.value && !selectedProjectId.value) {
      ElMessage.warning('请选择项目')
      return
    }
    credentialDialogVisible.value = true
  }

  const handleCredentialCreated = async (credential: DevopsCredential) => {
    await getCredentials()
    formData.credentialId = credential.id
  }

  const handleCredentialVisibleChange = async (visible: boolean) => {
    if (visible) {
      await getCredentials()
    }
  }

  const handleBaseSelectVisibleChange = async (visible: boolean) => {
    if (visible) {
      await Promise.all([getGroups(), getHosts()])
      await getChannelTypesByGroup(selectedChannelGroupCode.value)
    }
  }

  const pickIcon = (icon: string, color: string) => {
    formData.icon = icon
    formData.iconColor = color
  }

  const renderCharts = async () => {
    await nextTick()
    if (healthChartRef.value) {
      healthChart ||= echarts.init(healthChartRef.value)
      healthChart.setOption(healthChartOption(), true)
    }
    if (typeChartRef.value) {
      typeChart ||= echarts.init(typeChartRef.value)
      typeChart.setOption(typeChartOption(), true)
    }
  }

  const healthChartOption = (): EChartsOption => ({
    tooltip: { trigger: 'item' },
    legend: { bottom: 4 },
    graphic:
      channelData.value.length === 0
        ? {
            type: 'text',
            left: 'center',
            top: '42%',
            style: { text: '暂无实例', fill: '#909399', fontSize: 13 }
          }
        : undefined,
    series: [
      {
        type: 'pie',
        radius: ['38%', '58%'],
        center: ['50%', '40%'],
        data: [
          {
            name: '正常',
            value: channelData.value.filter((item) => item.healthStatus === 'healthy').length,
            itemStyle: { color: '#16a34a' }
          },
          {
            name: '异常',
            value: channelData.value.filter((item) => item.healthStatus === 'unhealthy').length,
            itemStyle: { color: '#dc2626' }
          },
          {
            name: '未知',
            value: channelData.value.filter(
              (item) => !['healthy', 'unhealthy'].includes(item.healthStatus)
            ).length,
            itemStyle: { color: '#64748b' }
          }
        ],
        label: { show: false },
        labelLine: { show: false },
        emphasis: { label: { show: false } }
      }
    ]
  })

  const typeChartOption = (): EChartsOption => {
    const counts = channelData.value.reduce(
      (acc, item) => {
        const name = channelTypeLabel(item.channelType, item.groupId)
        acc[name] = (acc[name] || 0) + 1
        return acc
      },
      {} as Record<string, number>
    )
    return {
      tooltip: { trigger: 'axis' },
      grid: { left: 24, right: 18, top: 20, bottom: 46, containLabel: true },
      graphic:
        Object.keys(counts).length === 0
          ? {
              type: 'text',
              left: 'center',
              top: '42%',
              style: { text: '暂无实例', fill: '#909399', fontSize: 13 }
            }
          : undefined,
      xAxis: {
        type: 'category',
        data: Object.keys(counts),
        axisLabel: { interval: 0, overflow: 'truncate', width: 86, margin: 12 }
      },
      yAxis: { type: 'value', minInterval: 1 },
      series: [
        {
          type: 'bar',
          barWidth: 18,
          data: Object.values(counts),
          itemStyle: { borderRadius: [5, 5, 0, 0], color: '#0f766e' }
        }
      ]
    }
  }

  watch(channelData, renderCharts, { deep: true })
  watch(selectedProjectId, () => {
    getCredentials()
  })

  let activatedOnce = false
  onMounted(refreshAll)
  onActivated(() => {
    if (!activatedOnce) {
      activatedOnce = true
      return
    }
    getCredentials()
  })

  onBeforeUnmount(() => {
    healthChart?.dispose()
    typeChart?.dispose()
  })
</script>

<style lang="scss" scoped>
  .devops-channel-center-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .channel-error {
    margin-bottom: 12px;
  }

  .channel-chart-grid {
    flex-shrink: 0;
    display: grid;
    grid-template-columns: minmax(300px, 0.8fr) minmax(0, 1.2fr);
    gap: 12px;
    margin-bottom: 18px;
  }

  .chart-panel {
    padding: 14px;
    overflow: hidden;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .panel-title {
    margin-bottom: 8px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .chart-box {
    height: 188px;
    overflow: hidden;
  }

  .summary-list {
    display: grid;
    flex: 1;
    grid-template-columns: repeat(4, minmax(120px, 1fr));
    gap: 10px;
    min-width: 0;
    margin-bottom: 12px;
  }

  .summary-item {
    min-height: 64px;
    padding: 12px 14px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .summary-label {
    display: block;
    margin-bottom: 8px;
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }

  .summary-item strong {
    font-size: 22px;
    line-height: 1;
    color: var(--el-text-color-primary);
  }

  .channel-card {
    min-width: 0;
    overflow: hidden;
    border-radius: 8px;
  }

  .channel-card-header {
    display: flex;
    gap: 12px;
    min-height: 78px;
    padding: 16px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .channel-icon {
    display: flex;
    flex: 0 0 44px;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    font-size: 26px;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .channel-title {
    flex: 1;
    min-width: 0;
  }

  .channel-name-row {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .channel-name {
    overflow: hidden;
    font-weight: 600;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .channel-code {
    display: block;
    margin-top: 6px;
    overflow: hidden;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .channel-card-body {
    display: flex;
    flex-direction: column;
    gap: 10px;
    min-height: 228px;
    padding: 14px 16px;
  }

  .channel-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .channel-desc {
    display: -webkit-box;
    min-height: 40px;
    margin: 0;
    overflow: hidden;
    color: var(--el-text-color-regular);
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }

  .channel-meta {
    display: grid;
    grid-template-columns: 48px minmax(0, 1fr);
    gap: 8px;
    font-size: 13px;
  }

  .channel-meta span {
    color: var(--el-text-color-secondary);
  }

  .channel-meta strong {
    overflow: hidden;
    font-weight: 500;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .channel-health-message {
    min-height: 30px;
    padding: 6px 8px;
    overflow: hidden;
    font-size: 12px;
    line-height: 18px;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
    background: var(--el-fill-color-lighter);
    border-radius: 6px;
  }

  .channel-card-footer {
    display: flex;
    flex-wrap: nowrap;
    justify-content: flex-end;
    gap: 6px;
    padding: 10px 14px;
    background: var(--el-fill-color-extra-light);
    border-top: 1px solid var(--el-border-color-lighter);

    :deep(.el-button) {
      display: inline-flex;
      flex: 1 1 0;
      align-items: center;
      justify-content: center;
      min-width: 0;
      height: 32px;
      padding: 0 6px;
      margin-left: 0;
      border-radius: 6px;
    }
  }

  :deep(.jenkins-agent-dialog) {
    display: flex;
    flex-direction: column;
    max-height: min(86vh, 820px);
    overflow: hidden;
    border-radius: 8px;
  }

  :deep(.jenkins-agent-dialog .el-dialog__body) {
    flex: 1;
    min-height: 0;
    max-height: none;
    padding: 16px 20px 20px;
    overflow: hidden;
  }

  .agent-dialog-shell {
    display: flex;
    flex-direction: column;
    gap: 14px;
    height: min(70vh, 680px);
    min-height: 520px;
    overflow: hidden;
  }

  .agent-dialog-summary {
    display: flex;
    flex: none;
    gap: 14px;
    align-items: center;
    justify-content: space-between;
    padding: 14px 16px;
    background: linear-gradient(180deg, #f8fbff 0%, #f4f7fc 100%);
    border: 1px solid #dfe7f3;
    border-radius: 8px;

    > div {
      min-width: 0;
    }

    strong,
    span {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      font-size: 16px;
      color: var(--el-text-color-primary);
    }

    span {
      margin-top: 4px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .agent-dialog-summary .agent-count-badge {
    display: flex;
    flex: none;
    gap: 4px;
    align-items: center;
    justify-content: center;
    min-width: 112px;
    height: 36px;
    padding: 0 14px;
    margin: 0;
    line-height: 1;
    white-space: nowrap;
    background: var(--el-color-primary-light-9);
    border: 1px solid var(--el-color-primary-light-5);
    border-radius: 6px;

    b,
    em {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      height: 100%;
      margin: 0;
      line-height: 1;
      color: var(--el-color-primary);
      white-space: nowrap;
    }

    b {
      font-size: 15px;
      font-weight: 700;
    }

    em {
      font-size: 13px;
      font-style: normal;
      font-weight: 600;
    }
  }

  .agent-layout {
    display: grid;
    grid-template-columns: 300px minmax(0, 1fr);
    flex: 1;
    gap: 14px;
    min-height: 0;
    overflow: hidden;
  }

  .agent-list-panel,
  .agent-form-panel {
    display: flex;
    flex-direction: column;
    min-width: 0;
    min-height: 0;
    overflow: hidden;
    background: var(--el-bg-color);
    border: 1px solid #dfe7f3;
    border-radius: 8px;
  }

  .agent-list-panel {
    background: #f8fafc;
  }

  .agent-list-head {
    display: flex;
    flex: none;
    gap: 10px;
    align-items: center;
    justify-content: space-between;
    padding: 14px;
    border-bottom: 1px solid #e5edf8;

    > div {
      min-width: 0;
    }

    strong,
    span {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      font-size: 15px;
      color: var(--el-text-color-primary);
    }

    span {
      margin-top: 4px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .agent-list-body {
    flex: 1;
    min-height: 0;
    padding: 12px;
    overflow: auto;
  }

  .agent-empty {
    min-height: 260px;
  }

  .agent-item {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 12px;
    align-items: center;
    width: 100%;
    min-height: 72px;
    padding: 12px 14px;
    margin-bottom: 10px;
    text-align: left;
    cursor: pointer;
    background: var(--el-bg-color);
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    transition:
      border-color 0.2s ease,
      box-shadow 0.2s ease,
      transform 0.2s ease;

    &:hover {
      border-color: #b8c7e6;
      box-shadow: 0 8px 18px rgb(15 23 42 / 6%);
      transform: translateY(-1px);
    }
  }

  .agent-item.active {
    background: #eef4ff;
    border-color: var(--el-color-primary);
    box-shadow: 0 0 0 3px var(--el-color-primary-light-9);
  }

  .agent-item-main {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    min-width: 0;
  }

  .agent-item-main strong,
  .agent-item-main em {
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .agent-item-main strong {
    font-size: 14px;
    color: var(--el-text-color-primary);
  }

  .agent-item-main em {
    margin-top: 4px;
    font-size: 12px;
    font-style: normal;
    color: var(--el-text-color-secondary);
  }

  .agent-item-tags {
    display: flex;
    flex-direction: column;
    gap: 6px;
    align-items: flex-end;
  }

  .agent-form-title {
    display: flex;
    flex: none;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    padding: 14px 16px;
    border-bottom: 1px solid #e5edf8;

    strong,
    span {
      display: block;
    }

    strong {
      font-size: 16px;
      color: var(--el-text-color-primary);
    }

    span {
      margin-top: 4px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .agent-form {
    flex: 1;
    min-height: 0;
    padding: 14px 16px 4px;
    overflow: auto;

    :deep(.el-segmented) {
      width: 100%;
      min-height: 40px;
      padding: 3px;
      border-radius: 8px;
    }

    :deep(.el-segmented__item) {
      display: inline-flex;
      flex: 1 1 0;
      align-items: center;
      justify-content: center;
      min-width: 0;
      height: 34px;
      padding: 0 12px;
      border-radius: 6px;
    }

    :deep(.el-segmented__item-label) {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 100%;
      min-width: 0;
      overflow: visible;
      line-height: 1;
      text-align: center;
      white-space: nowrap;
    }
  }

  .agent-form-section {
    padding: 14px;
    margin-bottom: 14px;
    background: #fbfcff;
    border: 1px solid #e3e9f5;
    border-radius: 8px;
  }

  .agent-section-title {
    display: flex;
    gap: 10px;
    align-items: center;
    justify-content: space-between;
    min-height: 28px;
    margin-bottom: 12px;

    strong {
      font-size: 14px;
      color: var(--el-text-color-primary);
    }
  }

  .agent-form-grid {
    display: grid;
    gap: 14px;

    &.two {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }

  .agent-form-grid :deep(.el-form-item:last-child:nth-child(odd)) {
    grid-column: 1 / -1;
  }

  .agent-container-list {
    display: grid;
    gap: 10px;
    margin-bottom: 14px;
  }

  .agent-container-row {
    display: grid;
    grid-template-columns: minmax(0, 190px) minmax(0, 1fr) 48px;
    gap: 10px;
    align-items: center;
  }

  .agent-form-actions {
    display: flex;
    flex: none;
    gap: 10px;
    align-items: center;
    justify-content: space-between;
    padding-top: 12px;
    padding: 12px 16px 14px;
    border-top: 1px solid #e5edf8;
  }

  .credential-add-btn {
    width: 100%;
  }

  .icon-picker {
    display: grid;
    grid-template-columns: repeat(auto-fill, 36px);
    gap: 8px;
    width: 100%;
  }

  .icon-picker button {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    font-size: 21px;
    cursor: pointer;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .icon-picker button.active {
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary);
  }

  .dynamic-config {
    padding: 12px 0 2px;
    margin-bottom: 10px;
    border-top: 1px solid var(--el-border-color-lighter);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .form-label-help {
    display: inline-flex;
    flex-wrap: nowrap;
    gap: 4px;
    align-items: center;
    min-width: 0;
    white-space: nowrap;
  }

  .field-help-icon {
    flex: 0 0 auto;
    font-size: 14px;
    color: var(--el-text-color-secondary);
    cursor: help;
  }

  :deep(.el-dialog__body) {
    max-height: calc(100vh - 220px);
    overflow-y: auto;
  }

  .channel-test-result {
    padding: 14px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    background: var(--el-fill-color-extra-light);
  }

  .channel-test-result.is-success {
    border-color: var(--el-color-success-light-5);
    background: var(--el-color-success-light-9);
  }

  .channel-test-result.is-error {
    border-color: var(--el-color-danger-light-5);
    background: var(--el-color-danger-light-9);
  }

  .test-result-head {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 12px;
  }

  .test-result-head strong {
    font-size: 16px;
    color: var(--el-text-color-primary);
  }

  .test-result-head p {
    max-width: 360px;
    margin: 6px 0 0;
    overflow: hidden;
    line-height: 20px;
    color: var(--el-text-color-regular);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .test-result-list {
    display: grid;
    grid-template-columns: 1fr;
    gap: 8px;
  }

  .test-result-list div {
    display: grid;
    grid-template-columns: 86px minmax(0, 1fr);
    gap: 12px;
    align-items: center;
    min-width: 0;
    padding: 9px 12px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
  }

  .test-result-list span {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .test-result-list strong {
    overflow: hidden;
    font-weight: 500;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  @media (max-width: 768px) {
    .channel-chart-grid {
      grid-template-columns: 1fr;
    }

    .summary-list {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .test-result-list div {
      grid-template-columns: 72px minmax(0, 1fr);
    }
  }
</style>
