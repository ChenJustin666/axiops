<template>
  <div class="retention-policy">
    <!-- 工具栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <ElSelect
          v-model="selectedProject"
          placeholder="选择项目"
          style="width: 300px"
          filterable
          @change="handleProjectChange"
        >
          <ElOption
            v-for="project in projects"
            :key="project.name"
            :label="project.name"
            :value="project.name"
          />
        </ElSelect>
      </div>

      <div class="toolbar-right">
        <!-- 【修复】使用 policyExists 计算属性判断按钮显示 -->
        <ElButton
          v-if="policyExists"
          type="primary"
          :icon="Play"
          :disabled="!canExecute"
          @click="handleExecutePolicy"
        >
          执行策略
        </ElButton>
        <ElButton v-if="policyExists" :icon="Edit" @click="handleEditPolicy">编辑策略</ElButton>
        <ElButton v-else type="primary" :icon="Plus" @click="handleCreatePolicy">
          创建策略
        </ElButton>
        <ElButton :icon="RefreshCw" @click="fetchPolicy">刷新</ElButton>
      </div>
    </div>

    <!-- 策略信息 - 【修复】使用 policyExists 计算属性判断显示 -->
    <ElCard
      v-if="policyExists && currentPolicy"
      shadow="never"
      class="policy-card"
      v-loading="loading"
    >
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <Archive :size="20" />
            <span>保留策略</span>
          </div>
          <ElTag type="success" size="small">已配置</ElTag>
        </div>
      </template>

      <ElDescriptions :column="3" border size="small">
        <ElDescriptionsItem label="策略 ID">
          <ElTag size="small">{{ currentPolicy.id }}</ElTag>
        </ElDescriptionsItem>
        <ElDescriptionsItem label="算法">
          <ElTag :type="currentPolicy.algorithm === 'or' ? 'primary' : 'warning'" size="small">
            {{ currentPolicy.algorithm === 'or' ? 'OR (满足任一)' : 'AND (满足所有)' }}
          </ElTag>
        </ElDescriptionsItem>
        <ElDescriptionsItem label="触发方式">
          {{ parseTrigger(currentPolicy.trigger) }}
        </ElDescriptionsItem>
        <ElDescriptionsItem label="创建时间">
          {{ formatDateTime(currentPolicy.createTime) }}
        </ElDescriptionsItem>
        <ElDescriptionsItem label="更新时间">
          {{ formatDateTime(currentPolicy.updateTime) }}
        </ElDescriptionsItem>
        <ElDescriptionsItem label="规则数量">
          <ElTag size="small">{{ currentPolicy.rules?.length || 0 }} 条</ElTag>
        </ElDescriptionsItem>
      </ElDescriptions>

      <!-- 规则列表 -->
      <div class="rules-section">
        <div class="section-header">
          <div class="section-title">
            <Shield :size="16" />
            <span>保留规则</span>
          </div>
        </div>
        <ElTable :data="currentPolicy.rules || []" style="width: 100%" stripe>
          <ElTableColumn label="优先级" width="80" align="center">
            <template #default="{ row }">
              <ElTag size="small" type="info">{{ row.priority }}</ElTag>
            </template>
          </ElTableColumn>

          <ElTableColumn label="规则说明" min-width="200">
            <template #default="{ row }">
              <div class="rule-desc">
                <span class="rule-icon">📋</span>
                {{ getRuleDescription(row) }}
              </div>
            </template>
          </ElTableColumn>

          <ElTableColumn label="标签匹配" min-width="120">
            <template #default="{ row }">
              <ElTag size="small" effect="plain">
                {{ parseTagSelectors(row.tagSelectors) }}
              </ElTag>
            </template>
          </ElTableColumn>

          <ElTableColumn label="状态" width="80" align="center">
            <template #default="{ row }">
              <ElTag :type="row.disabled ? 'info' : 'success'" size="small">
                {{ row.disabled ? '禁用' : '启用' }}
              </ElTag>
            </template>
          </ElTableColumn>
        </ElTable>
      </div>

      <!-- 执行历史 -->
      <div class="history-section">
        <div class="section-header">
          <div class="section-title">
            <Clock :size="16" />
            <span>执行历史</span>
          </div>
          <ElButton size="small" text :icon="RefreshCw" @click="fetchExecutions">刷新</ElButton>
        </div>
        <ElTable :data="executions" style="width: 100%" v-loading="loadingExecutions" stripe>
          <ElTableColumn label="执行 ID" width="80" align="center">
            <template #default="{ row }">
              <ElTag size="small" type="info">{{ row.id }}</ElTag>
            </template>
          </ElTableColumn>

          <ElTableColumn label="状态" width="100" align="center">
            <template #default="{ row }">
              <ElTag :type="getExecutionStatusType(row.status)" size="small">
                {{ getExecutionStatusText(row.status) }}
              </ElTag>
            </template>
          </ElTableColumn>

          <ElTableColumn label="触发方式" width="100" align="center">
            <template #default="{ row }">
              {{ row.trigger === 'manual' ? '手动' : '定时' }}
            </template>
          </ElTableColumn>

          <ElTableColumn label="模式" width="80" align="center">
            <template #default="{ row }">
              <ElTag :type="row.dryRun ? 'warning' : 'primary'" size="small">
                {{ row.dryRun ? '预演' : '正式' }}
              </ElTag>
            </template>
          </ElTableColumn>

          <ElTableColumn label="开始时间" width="170">
            <template #default="{ row }">{{ formatDateTime(row.startTime) }}</template>
          </ElTableColumn>

          <ElTableColumn label="结束时间" width="170">
            <template #default="{ row }">
              {{ row.endTime ? formatDateTime(row.endTime) : '-' }}
            </template>
          </ElTableColumn>
        </ElTable>

        <!-- 执行历史分页 -->
        <div v-if="executions.length > 0" class="pagination-wrapper">
          <ElPagination
            v-model:current-page="executionParams.page"
            v-model:page-size="executionParams.pageSize"
            :total="executionTotal"
            :page-sizes="[10, 20, 50]"
            layout="total, sizes, prev, pager, next"
            small
            @size-change="handleExecutionSizeChange"
            @current-change="handleExecutionPageChange"
          />
        </div>
      </div>
    </ElCard>

    <!-- 空状态 - 项目已选但策略不存在 -->
    <ElCard v-else-if="selectedProject && !loading" shadow="never" class="empty-card">
      <ElEmpty description="该项目暂无保留策略" :image-size="100">
        <ElButton type="primary" :icon="Plus" @click="handleCreatePolicy">创建策略</ElButton>
      </ElEmpty>
    </ElCard>

    <!-- 空状态 - 未选择项目 -->
    <ElCard v-else-if="!selectedProject" shadow="never" class="empty-card">
      <ElEmpty description="请先选择项目" :image-size="100">
        <template #image>
          <FolderOpen :size="64" class="empty-icon" />
        </template>
      </ElEmpty>
    </ElCard>

    <!-- 创建/编辑策略对话框 -->
    <ElDialog
      v-model="policyDialogVisible"
      :title="policyMode === 'create' ? '创建保留策略' : '编辑保留策略'"
      width="800px"
      :close-on-click-modal="false"
      destroy-on-close
      class="policy-dialog"
    >
      <ElForm ref="policyFormRef" :model="policyForm" :rules="policyRules" label-width="100px">
        <!-- 基本配置 -->
        <div class="form-section">
          <div class="form-section-title">基本配置</div>
          <ElFormItem label="算法" prop="algorithm">
            <ElRadioGroup v-model="policyForm.algorithm">
              <ElRadioButton value="or">
                <div class="radio-content">
                  <span class="radio-title">OR 模式</span>
                  <span class="radio-desc">满足任一规则即保留</span>
                </div>
              </ElRadioButton>
              <ElRadioButton value="and">
                <div class="radio-content">
                  <span class="radio-title">AND 模式</span>
                  <span class="radio-desc">满足所有规则才保留</span>
                </div>
              </ElRadioButton>
            </ElRadioGroup>
          </ElFormItem>

          <ElFormItem label="定时调度">
            <ElInput
              v-model="policyForm.schedule"
              placeholder="Cron 表达式（可选），例如: 0 0 2 * * ?"
              clearable
            >
              <template #prepend>
                <Clock :size="16" />
              </template>
            </ElInput>
            <div class="form-tip"
              >留空表示仅手动触发，定时规则: 秒=0, 分=0, 时=0, 日=*, 月=*, 周=*</div
            >
          </ElFormItem>
        </div>

        <!-- 保留规则 -->
        <div class="form-section">
          <div class="form-section-header">
            <div class="form-section-title">保留规则</div>
            <ElButton type="primary" size="small" :icon="Plus" @click="handleAddRule">
              添加规则
            </ElButton>
          </div>

          <!-- 规则列表容器 - 固定高度可滚动 -->
          <div class="rules-container" v-if="policyForm.rules.length > 0">
            <TransitionGroup name="rule-list" tag="div">
              <div v-for="(rule, index) in policyForm.rules" :key="rule._key" class="rule-card">
                <div class="rule-card-header">
                  <div class="rule-card-title">
                    <ElTag size="small" type="primary">规则 {{ index + 1 }}</ElTag>
                    <ElSwitch
                      v-model="rule.enabled"
                      size="small"
                      active-text="启用"
                      inactive-text="禁用"
                      inline-prompt
                    />
                  </div>
                  <ElButton
                    type="danger"
                    size="small"
                    :icon="Trash2"
                    circle
                    plain
                    @click="handleRemoveRule(index)"
                  />
                </div>

                <div class="rule-card-body">
                  <ElRow :gutter="16">
                    <ElCol :span="8">
                      <ElFormItem label="优先级" label-width="70px" size="small">
                        <ElInputNumber
                          v-model="rule.priority"
                          :min="0"
                          :max="99"
                          controls-position="right"
                          style="width: 100%"
                        />
                      </ElFormItem>
                    </ElCol>
                    <ElCol :span="16">
                      <ElFormItem label="规则模板" label-width="80px" size="small">
                        <ElSelect
                          v-model="rule.template"
                          placeholder="选择规则模板"
                          style="width: 100%"
                          @change="handleTemplateChange(rule)"
                        >
                          <ElOption
                            v-for="tmpl in ruleTemplates"
                            :key="tmpl.value"
                            :label="tmpl.label"
                            :value="tmpl.value"
                          >
                            <div class="template-option">
                              <span class="template-icon">{{ tmpl.icon }}</span>
                              <div class="template-info">
                                <span class="template-name">{{ tmpl.label }}</span>
                                <span class="template-desc">{{ tmpl.description }}</span>
                              </div>
                            </div>
                          </ElOption>
                        </ElSelect>
                      </ElFormItem>
                    </ElCol>
                  </ElRow>

                  <!-- 动态参数输入 -->
                  <ElRow :gutter="16">
                    <ElCol :span="12">
                      <ElFormItem
                        v-if="rule.template === 'latestPushedK'"
                        label="保留数量"
                        label-width="70px"
                        size="small"
                      >
                        <ElInputNumber
                          v-model="rule.latestPushedK"
                          :min="1"
                          :max="100"
                          controls-position="right"
                          style="width: 100%"
                          placeholder="保留最新推送的 N 个"
                        />
                      </ElFormItem>

                      <ElFormItem
                        v-else-if="rule.template === 'latestPulledN'"
                        label="保留数量"
                        label-width="70px"
                        size="small"
                      >
                        <ElInputNumber
                          v-model="rule.latestPulledN"
                          :min="1"
                          :max="100"
                          controls-position="right"
                          style="width: 100%"
                          placeholder="保留最新拉取的 N 个"
                        />
                      </ElFormItem>

                      <ElFormItem
                        v-else-if="rule.template === 'nDaysSinceLastPush'"
                        label="保留天数"
                        label-width="70px"
                        size="small"
                      >
                        <ElInputNumber
                          v-model="rule.nDaysSinceLastPush"
                          :min="1"
                          :max="365"
                          controls-position="right"
                          style="width: 100%"
                          placeholder="保留最近 N 天内推送的"
                        />
                      </ElFormItem>

                      <ElFormItem
                        v-else-if="rule.template === 'nDaysSinceLastPull'"
                        label="保留天数"
                        label-width="70px"
                        size="small"
                      >
                        <ElInputNumber
                          v-model="rule.nDaysSinceLastPull"
                          :min="1"
                          :max="365"
                          controls-position="right"
                          style="width: 100%"
                          placeholder="保留最近 N 天内拉取的"
                        />
                      </ElFormItem>

                      <ElFormItem
                        v-else-if="rule.template === 'always'"
                        label="说明"
                        label-width="70px"
                        size="small"
                      >
                        <ElTag type="success" size="small">始终保留匹配的制品</ElTag>
                      </ElFormItem>
                    </ElCol>

                    <ElCol :span="12">
                      <ElFormItem label="标签匹配" label-width="70px" size="small">
                        <ElSelect
                          v-model="rule.tagPattern"
                          placeholder="选择或输入"
                          filterable
                          allow-create
                          style="width: 100%"
                        >
                          <ElOption label="所有标签 (**)" value="**" />
                          <ElOption label="以 v 开头 (v*)" value="v*" />
                          <ElOption label="以 release 开头 (release*)" value="release*" />
                          <ElOption label="latest 标签" value="latest" />
                        </ElSelect>
                      </ElFormItem>
                    </ElCol>
                  </ElRow>

                  <ElRow :gutter="16">
                    <ElCol :span="12">
                      <ElFormItem label="匹配方式" label-width="70px" size="small">
                        <ElSelect v-model="rule.decoration" style="width: 100%">
                          <ElOption label="匹配 (matches)" value="matches" />
                          <ElOption label="排除 (excludes)" value="excludes" />
                        </ElSelect>
                      </ElFormItem>
                    </ElCol>
                    <ElCol :span="12">
                      <ElFormItem label="仓库范围" label-width="70px" size="small">
                        <ElSelect
                          v-model="rule.repoPattern"
                          placeholder="选择或输入"
                          filterable
                          allow-create
                          style="width: 100%"
                        >
                          <ElOption label="所有仓库 (**)" value="**" />
                        </ElSelect>
                      </ElFormItem>
                    </ElCol>
                  </ElRow>
                </div>
              </div>
            </TransitionGroup>
          </div>

          <!-- 空状态 -->
          <div v-else class="rules-empty">
            <ElEmpty description="暂无规则，请添加保留规则" :image-size="60">
              <ElButton type="primary" :icon="Plus" @click="handleAddRule">添加规则</ElButton>
            </ElEmpty>
          </div>
        </div>

        <!-- 规则说明 -->
        <ElAlert type="info" :closable="false" show-icon class="rule-tips">
          <template #title>规则说明</template>
          <div class="tips-content">
            <p>• 优先级数字越小，优先级越高（0 为最高）</p>
            <p>• OR 算法：制品满足任一规则即保留；AND 算法：必须满足所有规则才保留</p>
            <p>• 标签选择器支持通配符：<code>**</code> 匹配所有，<code>v*</code> 匹配以 v 开头</p>
          </div>
        </ElAlert>
      </ElForm>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="policyDialogVisible = false">取消</ElButton>
          <ElButton type="primary" @click="handleSubmitPolicy" :loading="submitting">
            {{ policyMode === 'create' ? '创建策略' : '保存修改' }}
          </ElButton>
        </div>
      </template>
    </ElDialog>

    <!-- 执行策略对话框 -->
    <ElDialog
      v-model="executeDialogVisible"
      title="执行保留策略"
      width="480px"
      :close-on-click-modal="false"
    >
      <div class="execute-form">
        <div class="execute-mode">
          <div
            class="mode-card"
            :class="{ active: executeForm.dryRun }"
            @click="executeForm.dryRun = true"
          >
            <div class="mode-icon">🔍</div>
            <div class="mode-info">
              <div class="mode-title">预演模式</div>
              <div class="mode-desc">仅模拟执行，不实际删除制品</div>
            </div>
            <ElIcon v-if="executeForm.dryRun" class="mode-check"><Check /></ElIcon>
          </div>
          <div
            class="mode-card"
            :class="{ active: !executeForm.dryRun }"
            @click="executeForm.dryRun = false"
          >
            <div class="mode-icon">⚡</div>
            <div class="mode-info">
              <div class="mode-title">正式执行</div>
              <div class="mode-desc">根据规则删除制品，操作不可恢复</div>
            </div>
            <ElIcon v-if="!executeForm.dryRun" class="mode-check"><Check /></ElIcon>
          </div>
        </div>

        <ElAlert
          :type="executeForm.dryRun ? 'info' : 'warning'"
          :closable="false"
          show-icon
          class="execute-warning"
        >
          <template #title>{{ executeForm.dryRun ? '预演说明' : '警告' }}</template>
          <p v-if="executeForm.dryRun"
            >预演模式将模拟执行过程，您可以查看哪些制品会被删除，但不会实际删除任何内容。</p
          >
          <p v-else
            >正式执行将根据保留规则删除不符合条件的制品，此操作<strong>不可恢复</strong>，请确保规则配置正确。</p
          >
        </ElAlert>
      </div>

      <template #footer>
        <ElButton @click="executeDialogVisible = false">取消</ElButton>
        <ElButton
          :type="executeForm.dryRun ? 'primary' : 'danger'"
          @click="handleSubmitExecute"
          :loading="executing"
        >
          {{ executeForm.dryRun ? '开始预演' : '正式执行' }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { ref, reactive, computed, onMounted } from 'vue'
  import { ElMessage } from 'element-plus'
  import { Check } from '@element-plus/icons-vue'
  import type { FormInstance, FormRules } from 'element-plus'
  import {
    Archive,
    Plus,
    Edit,
    Play,
    RefreshCw,
    Clock,
    Trash2,
    Shield,
    FolderOpen
  } from 'lucide-vue-next'
  import {
    listRegistryProjectsApi,
    getRetentionPolicyApi,
    createRetentionPolicyApi,
    updateRetentionPolicyApi,
    executeRetentionPolicyApi,
    listRetentionExecutionsApi,
    type RegistryProject,
    type RetentionPolicy,
    type RetentionRule,
    type RetentionExecution,
    type GetRetentionPolicyResponse
  } from '@/api'

  interface Props {
    registryUuid: string
  }

  const props = defineProps<Props>()

  // 生成唯一 key
  let ruleKeyCounter = 0
  const generateRuleKey = () => `rule_${++ruleKeyCounter}_${Date.now()}`

  // 规则模板定义
  const ruleTemplates = [
    {
      value: 'latestPushedK',
      label: '保留最新推送的 N 个',
      description: '按推送时间保留最新的指定数量制品',
      icon: '📤'
    },
    {
      value: 'latestPulledN',
      label: '保留最新拉取的 N 个',
      description: '按拉取时间保留最新的指定数量制品',
      icon: '📥'
    },
    {
      value: 'nDaysSinceLastPush',
      label: '保留最近 N 天推送的',
      description: '保留指定天数内推送的所有制品',
      icon: '📅'
    },
    {
      value: 'nDaysSinceLastPull',
      label: '保留最近 N 天拉取的',
      description: '保留指定天数内被拉取过的制品',
      icon: '🗓️'
    },
    {
      value: 'always',
      label: '始终保留',
      description: '始终保留匹配的制品，不会被删除',
      icon: '🔒'
    }
  ]

  const loading = ref(false)
  const projects = ref<RegistryProject[]>([])
  const selectedProject = ref('')

  // 【修复】策略数据和存在标识分开管理
  const currentPolicy = ref<RetentionPolicy | null>(null)
  const policyResponseExists = ref(false) // 存储 API 响应的 exists 字段

  // 【新增】计算属性：判断策略是否真正存在
  const policyExists = computed(() => {
    // 1. 检查 API 响应的 exists 字段
    if (!policyResponseExists.value) {
      return false
    }
    // 2. 检查 currentPolicy 是否存在且 id > 0
    if (!currentPolicy.value || !currentPolicy.value.id || currentPolicy.value.id <= 0) {
      return false
    }
    return true
  })

  // 【新增】计算属性：判断是否可以执行策略
  const canExecute = computed(() => {
    return policyExists.value && currentPolicy.value && currentPolicy.value.id > 0
  })

  // 执行历史
  const loadingExecutions = ref(false)
  const executions = ref<RetentionExecution[]>([])
  const executionTotal = ref(0)
  const executionParams = reactive({
    page: 1,
    pageSize: 10
  })

  // 策略对话框
  const policyDialogVisible = ref(false)
  const policyMode = ref<'create' | 'edit'>('create')
  const policyFormRef = ref<FormInstance>()
  const submitting = ref(false)

  interface RuleForm {
    _key: string
    priority: number
    enabled: boolean
    template: string
    latestPushedK?: number
    latestPulledN?: number
    nDaysSinceLastPush?: number
    nDaysSinceLastPull?: number
    tagPattern: string
    decoration: string
    repoPattern: string
  }

  const policyForm = reactive({
    algorithm: 'or',
    schedule: '',
    rules: [] as RuleForm[]
  })

  const policyRules: FormRules = {
    algorithm: [{ required: true, message: '请选择算法', trigger: 'change' }]
  }

  // 执行对话框
  const executeDialogVisible = ref(false)
  const executing = ref(false)
  const executeForm = reactive({
    dryRun: true
  })

  // 获取项目列表
  const fetchProjects = async () => {
    try {
      const response = await listRegistryProjectsApi({
        registryUuid: props.registryUuid,
        page: 1,
        pageSize: 100
      })
      projects.value = response.items || []
    } catch (error) {
      console.error('获取项目列表失败:', error)
    }
  }

  // 【修复】获取策略 - 正确处理 exists 字段和空响应
  const fetchPolicy = async () => {
    if (!selectedProject.value) return

    loading.value = true
    // 重置状态
    currentPolicy.value = null
    policyResponseExists.value = false
    executions.value = []
    executionTotal.value = 0

    try {
      const response = await getRetentionPolicyApi({
        registryUuid: props.registryUuid,
        projectName: selectedProject.value
      })

      console.log('获取保留策略响应:', response)

      // 【关键修复】正确处理响应
      // 类型断言，兼容后端修复前后的响应格式
      const resp = response as GetRetentionPolicyResponse | { data: RetentionPolicy }

      // 情况1：后端已修复，返回 { exists: boolean, data: RetentionPolicy | null }
      if ('exists' in resp) {
        policyResponseExists.value = resp.exists === true
        if (resp.exists && resp.data && resp.data.id > 0) {
          currentPolicy.value = resp.data
        } else {
          currentPolicy.value = null
        }
      }
      // 情况2：后端未修复，返回 { data: RetentionPolicy }（旧格式）
      else if ('data' in resp && resp.data) {
        // 检查 data.id 是否有效
        if (resp.data.id && resp.data.id > 0) {
          policyResponseExists.value = true
          currentPolicy.value = resp.data
        } else {
          // id 为 0 或不存在，视为策略不存在
          policyResponseExists.value = false
          currentPolicy.value = null
        }
      }
      // 情况3：响应为空或无效
      else {
        policyResponseExists.value = false
        currentPolicy.value = null
      }

      console.log('策略是否存在:', policyExists.value, 'currentPolicy:', currentPolicy.value)

      // 【修复】只有策略真正存在时才获取执行历史
      if (policyExists.value) {
        await fetchExecutions()
      }
    } catch (error: any) {
      console.error('获取保留策略失败:', error)
      // 404 或其他错误都视为策略不存在
      policyResponseExists.value = false
      currentPolicy.value = null
    } finally {
      loading.value = false
    }
  }

  // 【修复】获取执行历史 - 添加 policyId 校验
  const fetchExecutions = async () => {
    // 【关键修复】检查策略是否存在且 ID 有效
    if (!policyExists.value || !currentPolicy.value?.id || currentPolicy.value.id <= 0) {
      console.log('策略不存在或 ID 无效，跳过获取执行历史')
      executions.value = []
      executionTotal.value = 0
      return
    }

    loadingExecutions.value = true
    try {
      const response = await listRetentionExecutionsApi({
        registryUuid: props.registryUuid,
        policyId: currentPolicy.value.id,
        page: executionParams.page,
        pageSize: executionParams.pageSize
      })
      executions.value = response.items || []
      executionTotal.value = response.total || 0
    } catch (error: any) {
      console.error('获取执行历史失败:', error)
      executions.value = []
      executionTotal.value = 0
    } finally {
      loadingExecutions.value = false
    }
  }

  // 项目切换
  const handleProjectChange = () => {
    currentPolicy.value = null
    policyResponseExists.value = false
    executions.value = []
    executionTotal.value = 0
    fetchPolicy()
  }

  // 创建策略
  const handleCreatePolicy = () => {
    policyMode.value = 'create'
    policyForm.algorithm = 'or'
    policyForm.schedule = ''
    policyForm.rules = []
    policyDialogVisible.value = true
  }

  // 【修复】编辑策略 - 检查策略是否存在
  const handleEditPolicy = () => {
    if (!policyExists.value || !currentPolicy.value) {
      ElMessage.warning('请先创建保留策略')
      return
    }

    policyMode.value = 'edit'
    policyForm.algorithm = currentPolicy.value.algorithm
    policyForm.schedule = ''
    policyForm.rules = (currentPolicy.value.rules || []).map((rule) => {
      const ruleForm: RuleForm = {
        _key: generateRuleKey(),
        priority: rule.priority,
        enabled: !rule.disabled,
        template: rule.template,
        tagPattern: '**',
        decoration: 'matches',
        repoPattern: '**'
      }

      // 解析参数
      if (rule.params) {
        if (rule.params.latestPushedK) {
          ruleForm.latestPushedK = parseInt(rule.params.latestPushedK)
        }
        if (rule.params.latestPulledN) {
          ruleForm.latestPulledN = parseInt(rule.params.latestPulledN)
        }
        if (rule.params.nDaysSinceLastPush) {
          ruleForm.nDaysSinceLastPush = parseInt(rule.params.nDaysSinceLastPush)
        }
        if (rule.params.nDaysSinceLastPull) {
          ruleForm.nDaysSinceLastPull = parseInt(rule.params.nDaysSinceLastPull)
        }
      }

      // 解析 tagSelectors
      if (rule.tagSelectors) {
        try {
          const selectors = JSON.parse(rule.tagSelectors)
          if (Array.isArray(selectors) && selectors.length > 0) {
            ruleForm.tagPattern = selectors[0].pattern || '**'
            ruleForm.decoration = selectors[0].decoration || 'matches'
          }
        } catch {
          ruleForm.tagPattern = '**'
        }
      }

      return ruleForm
    })
    policyDialogVisible.value = true
  }

  // 添加规则
  const handleAddRule = () => {
    policyForm.rules.push({
      _key: generateRuleKey(),
      priority: policyForm.rules.length,
      enabled: true,
      template: 'latestPushedK',
      latestPushedK: 10,
      tagPattern: '**',
      decoration: 'matches',
      repoPattern: '**'
    })
  }

  // 移除规则
  const handleRemoveRule = (index: number) => {
    policyForm.rules.splice(index, 1)
    // 重新调整优先级
    policyForm.rules.forEach((rule, i) => {
      rule.priority = i
    })
  }

  // 模板变更
  const handleTemplateChange = (rule: RuleForm) => {
    // 重置参数
    delete rule.latestPushedK
    delete rule.latestPulledN
    delete rule.nDaysSinceLastPush
    delete rule.nDaysSinceLastPull

    // 设置默认值
    switch (rule.template) {
      case 'latestPushedK':
        rule.latestPushedK = 10
        break
      case 'latestPulledN':
        rule.latestPulledN = 10
        break
      case 'nDaysSinceLastPush':
        rule.nDaysSinceLastPush = 30
        break
      case 'nDaysSinceLastPull':
        rule.nDaysSinceLastPull = 30
        break
    }
  }

  // 构建 tag_selectors JSON
  const buildTagSelectors = (rule: RuleForm): string => {
    const selectors = [
      {
        kind: 'doublestar',
        decoration: rule.decoration || 'matches',
        pattern: rule.tagPattern || '**'
      }
    ]
    return JSON.stringify(selectors)
  }

  // 构建 scope_selectors JSON
  const buildScopeSelectors = (rule: RuleForm): string => {
    const selectors = {
      repository: [
        {
          kind: 'doublestar',
          decoration: 'repoMatches',
          pattern: rule.repoPattern || '**'
        }
      ]
    }
    return JSON.stringify(selectors)
  }

  // 【修复】提交策略 - 支持创建或更新
  const handleSubmitPolicy = async () => {
    if (!policyFormRef.value) return

    try {
      await policyFormRef.value.validate()

      if (policyForm.rules.length === 0) {
        ElMessage.warning('请至少添加一条保留规则')
        return
      }

      // 转换规则格式
      const rules: RetentionRule[] = policyForm.rules.map((rule) => {
        const params: Record<string, string> = {}

        if (rule.latestPushedK !== undefined) {
          params.latestPushedK = rule.latestPushedK.toString()
        }
        if (rule.latestPulledN !== undefined) {
          params.latestPulledN = rule.latestPulledN.toString()
        }
        if (rule.nDaysSinceLastPush !== undefined) {
          params.nDaysSinceLastPush = rule.nDaysSinceLastPush.toString()
        }
        if (rule.nDaysSinceLastPull !== undefined) {
          params.nDaysSinceLastPull = rule.nDaysSinceLastPull.toString()
        }

        return {
          priority: rule.priority,
          disabled: !rule.enabled,
          action: 'retain',
          template: rule.template,
          params,
          tagSelectors: buildTagSelectors(rule),
          scopeSelectors: buildScopeSelectors(rule)
        }
      })

      submitting.value = true

      if (policyMode.value === 'create') {
        await createRetentionPolicyApi({
          registryUuid: props.registryUuid,
          projectName: selectedProject.value,
          algorithm: policyForm.algorithm,
          rules,
          schedule: policyForm.schedule || undefined
        })
        ElMessage.success('创建成功')
      } else {
        // 【修复】更新时传递 projectName，支持后端在 policyId 无效时创建新策略
        await updateRetentionPolicyApi({
          registryUuid: props.registryUuid,
          projectName: selectedProject.value,
          policyId: currentPolicy.value?.id || 0,
          algorithm: policyForm.algorithm,
          rules,
          schedule: policyForm.schedule || undefined
        })
        ElMessage.success('保存成功')
      }

      policyDialogVisible.value = false
      fetchPolicy()
    } catch (error: any) {
      console.error('操作失败:', error)
      ElMessage.error(error?.message || '操作失败')
    } finally {
      submitting.value = false
    }
  }

  // 【修复】执行策略 - 检查策略是否存在
  const handleExecutePolicy = () => {
    // 【关键修复】检查策略是否存在且 ID 有效
    if (!canExecute.value) {
      ElMessage.warning('请先配置保留策略')
      return
    }
    executeForm.dryRun = true
    executeDialogVisible.value = true
  }

  // 【修复】提交执行 - 检查策略 ID
  const handleSubmitExecute = async () => {
    // 【关键修复】检查策略是否存在且 ID 有效
    if (!canExecute.value) {
      ElMessage.error('策略ID无效，请先配置保留策略')
      executeDialogVisible.value = false
      return
    }

    executing.value = true
    try {
      const response = await executeRetentionPolicyApi({
        registryUuid: props.registryUuid,
        policyId: currentPolicy.value!.id,
        dryRun: executeForm.dryRun
      })

      ElMessage.success({
        message: `策略已${executeForm.dryRun ? '开始预演' : '开始执行'}，执行 ID: ${response.executionId}`,
        duration: 5000,
        showClose: true
      })
      executeDialogVisible.value = false

      setTimeout(() => {
        fetchExecutions()
      }, 1000)
    } catch (error: any) {
      console.error('执行策略失败:', error)
      ElMessage.error(error?.message || '执行策略失败')
    } finally {
      executing.value = false
    }
  }

  // 执行历史分页
  const handleExecutionSizeChange = () => {
    executionParams.page = 1
    fetchExecutions()
  }

  const handleExecutionPageChange = () => {
    fetchExecutions()
  }

  // 执行状态
  const getExecutionStatusType = (status: string) => {
    const typeMap: Record<string, any> = {
      InProgress: 'warning',
      Running: 'warning',
      Succeed: 'success',
      Success: 'success',
      Failed: 'danger',
      Stopped: 'info'
    }
    return typeMap[status] || 'info'
  }

  const getExecutionStatusText = (status: string) => {
    const textMap: Record<string, string> = {
      InProgress: '执行中',
      Running: '执行中',
      Succeed: '成功',
      Success: '成功',
      Failed: '失败',
      Stopped: '已停止'
    }
    return textMap[status] || status
  }

  // 解析触发器
  const parseTrigger = (trigger: string) => {
    if (!trigger) return '手动触发'
    try {
      const triggerObj = JSON.parse(trigger)
      if (triggerObj.kind === 'Schedule' && triggerObj.settings?.cron) {
        return `定时: ${triggerObj.settings.cron}`
      }
      return triggerObj.kind || '手动触发'
    } catch {
      return '手动触发'
    }
  }

  // 解析标签选择器
  const parseTagSelectors = (tagSelectors: string) => {
    if (!tagSelectors) return '**'
    try {
      const selectors = JSON.parse(tagSelectors)
      if (Array.isArray(selectors) && selectors.length > 0) {
        return selectors[0].pattern || '**'
      }
      return '**'
    } catch {
      return tagSelectors
    }
  }

  // 获取规则描述
  const getRuleDescription = (rule: RetentionRule) => {
    const template = ruleTemplates.find((t) => t.value === rule.template)
    if (!template) return rule.template

    let desc = template.label
    if (rule.params) {
      if (rule.params.latestPushedK) {
        desc = `保留最新推送的 ${rule.params.latestPushedK} 个制品`
      } else if (rule.params.latestPulledN) {
        desc = `保留最新拉取的 ${rule.params.latestPulledN} 个制品`
      } else if (rule.params.nDaysSinceLastPush) {
        desc = `保留最近 ${rule.params.nDaysSinceLastPush} 天内推送的制品`
      } else if (rule.params.nDaysSinceLastPull) {
        desc = `保留最近 ${rule.params.nDaysSinceLastPull} 天内拉取的制品`
      }
    }
    return desc
  }

  // 格式化日期时间
  const formatDateTime = (timestamp: number) => {
    if (!timestamp) return '-'
    return new Date(timestamp * 1000).toLocaleString('zh-CN')
  }

  onMounted(() => {
    fetchProjects()
  })
</script>

<style lang="scss" scoped>
  .retention-policy {
    .toolbar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 16px;
      padding: 12px 16px;
      background: var(--el-bg-color);
      border-radius: 8px;
      border: 1px solid var(--el-border-color-lighter);

      .toolbar-right {
        display: flex;
        gap: 8px;
      }
    }

    .policy-card {
      .card-header {
        display: flex;
        justify-content: space-between;
        align-items: center;

        .header-left {
          display: flex;
          align-items: center;
          gap: 8px;
          font-size: 16px;
          font-weight: 600;
        }
      }

      .rules-section,
      .history-section {
        margin-top: 24px;

        .section-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 12px;
        }

        .section-title {
          display: flex;
          align-items: center;
          gap: 6px;
          font-size: 14px;
          font-weight: 500;
          color: var(--el-text-color-primary);
        }

        .pagination-wrapper {
          display: flex;
          justify-content: flex-end;
          margin-top: 12px;
        }
      }

      .rule-desc {
        display: flex;
        align-items: center;
        gap: 8px;

        .rule-icon {
          font-size: 14px;
        }
      }
    }

    .empty-card {
      padding: 40px 0;

      .empty-icon {
        color: var(--el-text-color-placeholder);
      }
    }
  }

  // 策略对话框样式
  .policy-dialog {
    .form-section {
      margin-bottom: 20px;
      padding: 16px;
      background: var(--el-fill-color-lighter);
      border-radius: 8px;

      .form-section-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 16px;
      }

      .form-section-title {
        font-size: 14px;
        font-weight: 600;
        color: var(--el-text-color-primary);
        margin-bottom: 16px;

        .form-section-header & {
          margin-bottom: 0;
        }
      }

      .form-tip {
        font-size: 12px;
        color: var(--el-text-color-secondary);
        margin-top: 4px;
      }
    }

    // 规则容器 - 固定高度可滚动
    .rules-container {
      max-height: 360px;
      overflow-y: auto;
      padding-right: 8px;

      &::-webkit-scrollbar {
        width: 6px;
      }

      &::-webkit-scrollbar-thumb {
        background: var(--el-border-color);
        border-radius: 3px;
      }
    }

    .rule-card {
      background: var(--el-bg-color);
      border: 1px solid var(--el-border-color-lighter);
      border-radius: 8px;
      margin-bottom: 12px;
      transition: all 0.3s;

      &:hover {
        border-color: var(--el-color-primary-light-5);
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
      }

      .rule-card-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px 16px;
        border-bottom: 1px solid var(--el-border-color-lighter);
        background: var(--el-fill-color-light);
        border-radius: 8px 8px 0 0;

        .rule-card-title {
          display: flex;
          align-items: center;
          gap: 12px;
        }
      }

      .rule-card-body {
        padding: 16px;
      }
    }

    .rules-empty {
      padding: 24px;
      text-align: center;
    }

    .rule-tips {
      margin-top: 16px;

      .tips-content {
        p {
          margin: 4px 0;
          font-size: 12px;
          line-height: 1.6;

          code {
            background: var(--el-fill-color);
            padding: 2px 6px;
            border-radius: 4px;
            font-family: var(--ax-font-mono);
          }
        }
      }
    }

    // 单选按钮样式
    :deep(.el-radio-group) {
      display: flex;
      gap: 12px;

      .el-radio-button {
        .el-radio-button__inner {
          padding: 12px 20px;
          border-radius: 8px !important;
          border: 1px solid var(--el-border-color) !important;
          box-shadow: none !important;

          .radio-content {
            display: flex;
            flex-direction: column;
            align-items: flex-start;
            gap: 2px;

            .radio-title {
              font-weight: 500;
            }

            .radio-desc {
              font-size: 12px;
              color: var(--el-text-color-secondary);
            }
          }
        }

        &.is-active .el-radio-button__inner {
          border-color: var(--el-color-primary) !important;
          background: var(--el-color-primary-light-9);
        }
      }
    }
  }

  // 模板选项样式
  .template-option {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 4px 0;

    .template-icon {
      font-size: 18px;
    }

    .template-info {
      display: flex;
      flex-direction: column;

      .template-name {
        font-size: 13px;
        font-weight: 500;
      }

      .template-desc {
        font-size: 11px;
        color: var(--el-text-color-secondary);
      }
    }
  }

  // 执行对话框样式
  .execute-form {
    .execute-mode {
      display: flex;
      gap: 12px;
      margin-bottom: 16px;

      .mode-card {
        flex: 1;
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 16px;
        border: 2px solid var(--el-border-color);
        border-radius: 8px;
        cursor: pointer;
        transition: all 0.3s;
        position: relative;

        &:hover {
          border-color: var(--el-color-primary-light-3);
        }

        &.active {
          border-color: var(--el-color-primary);
          background: var(--el-color-primary-light-9);
        }

        .mode-icon {
          font-size: 28px;
        }

        .mode-info {
          flex: 1;

          .mode-title {
            font-size: 14px;
            font-weight: 600;
            margin-bottom: 2px;
          }

          .mode-desc {
            font-size: 12px;
            color: var(--el-text-color-secondary);
          }
        }

        .mode-check {
          position: absolute;
          top: 8px;
          right: 8px;
          color: var(--el-color-primary);
          font-size: 16px;
        }
      }
    }

    .execute-warning {
      p {
        margin: 0;
        font-size: 13px;
        line-height: 1.6;

        strong {
          color: var(--el-color-danger);
        }
      }
    }
  }

  // 动画
  .rule-list-enter-active,
  .rule-list-leave-active {
    transition: all 0.3s ease;
  }

  .rule-list-enter-from,
  .rule-list-leave-to {
    opacity: 0;
    transform: translateX(-20px);
  }

  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }
</style>
