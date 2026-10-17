<template>
  <div class="alert-page">
    <!-- 顶部导航区 -->
    <div class="page-nav">
      <div class="nav-breadcrumb">
        <ElBreadcrumb separator=">">
          <ElBreadcrumbItem
            :class="{ clickable: !!selectedFile }"
            @click="handleBreadcrumbClick('file')"
          >
            <ElIcon><Document /></ElIcon>
            {{ selectedFile ? selectedFile.fileName : '选择规则文件' }}
          </ElBreadcrumbItem>
          <ElBreadcrumbItem
            v-if="selectedFile"
            :class="{ clickable: !!selectedGroup }"
            @click="handleBreadcrumbClick('group')"
          >
            <ElIcon><FolderOpened /></ElIcon>
            {{ selectedGroup ? selectedGroup.groupName : '选择分组' }}
          </ElBreadcrumbItem>
          <ElBreadcrumbItem v-if="selectedGroup">
            <ElIcon><Bell /></ElIcon>
            告警规则
          </ElBreadcrumbItem>
        </ElBreadcrumb>
      </div>
      <div class="nav-actions">
        <ElDropdown trigger="click" @command="handleMoreAction">
          <ElButton>
            更多
            <ElIcon class="arrow"><ArrowDown /></ElIcon>
          </ElButton>
          <template #dropdown>
            <ElDropdownMenu>
              <ElDropdownItem command="import">
                <ElIcon><Upload /></ElIcon>
                批量导入
              </ElDropdownItem>
              <ElDropdownItem command="exportAll">
                <ElIcon><Download /></ElIcon>
                导出全部
              </ElDropdownItem>
              <ElDropdownItem divided command="downloadConfig">
                <ElIcon><Download /></ElIcon>
                下载配置
              </ElDropdownItem>
              <ElDropdownItem command="configAlert">
                <ElIcon><Setting /></ElIcon>
                配置 Alertmanager
              </ElDropdownItem>
            </ElDropdownMenu>
          </template>
        </ElDropdown>
        <ElButton type="primary" @click="handleDeployAll">
          <ElIcon><Position /></ElIcon>
          部署全部
        </ElButton>
      </div>
    </div>

    <!-- 主内容区 -->
    <div class="page-main">
      <!-- 未选择文件 -->
      <div v-if="!selectedFile" class="file-select-view">
        <div class="view-header">
          <div class="header-left">
            <h2>规则文件列表</h2>
            <span class="count">{{ fileList.length }} 个文件</span>
          </div>
          <ElButton type="primary" @click="handleAddFile">
            <ElIcon><Plus /></ElIcon>
            新建文件
          </ElButton>
        </div>

        <!-- 文件搜索 -->
        <div class="search-bar">
          <ElInput
            v-model="fileSearchKeyword"
            placeholder="搜索文件名称或代码..."
            clearable
            :prefix-icon="Search"
            class="search-input"
          />
        </div>

        <!-- 文件卡片列表 -->
        <div v-loading="fileLoading" class="file-card-list">
          <div
            v-for="file in filteredFileList"
            :key="file.id"
            class="file-card"
            @click="handleFileSelect(file)"
          >
            <div class="card-header">
              <div class="card-title">
                <ElIcon class="card-icon"><Document /></ElIcon>
                <span class="file-name">{{ file.fileName }}</span>
              </div>
              <ElTag :type="file.isEnabled ? 'success' : 'info'" size="small">
                {{ file.isEnabled ? '启用' : '禁用' }}
              </ElTag>
            </div>
            <div class="card-body">
              <div class="card-info">
                <span class="label">文件代码</span>
                <code class="value">{{ file.fileCode }}</code>
              </div>
              <div class="card-info">
                <span class="label">命名空间</span>
                <span class="value">{{ file.namespace || 'monitoring' }}</span>
              </div>
              <div v-if="file.description" class="card-info full">
                <span class="label">描述</span>
                <span class="value">{{ file.description }}</span>
              </div>
            </div>
            <div class="card-footer">
              <div class="card-stats">
                <span class="stat">
                  <ElIcon><Folder /></ElIcon>
                  {{ file.groupCount || 0 }} 分组
                </span>
                <span class="stat">
                  <ElIcon><Bell /></ElIcon>
                  {{ file.ruleCount || 0 }} 规则
                </span>
              </div>
              <div class="card-actions" @click.stop>
                <ElTooltip content="编辑">
                  <ElButton size="small" @click="handleEditFile(file)">
                    <ElIcon><Edit /></ElIcon>
                  </ElButton>
                </ElTooltip>
                <ElTooltip content="导出">
                  <ElButton size="small" @click="handleExportFile(file)">
                    <ElIcon><Download /></ElIcon>
                  </ElButton>
                </ElTooltip>
                <ElTooltip content="部署">
                  <ElButton size="small" type="success" @click="handleDeployFile(file)">
                    <ElIcon><Position /></ElIcon>
                  </ElButton>
                </ElTooltip>
                <ElTooltip content="删除">
                  <ElButton size="small" type="danger" @click="handleDeleteFile(file)">
                    <ElIcon><Delete /></ElIcon>
                  </ElButton>
                </ElTooltip>
              </div>
            </div>
          </div>

          <div v-if="filteredFileList.length === 0 && !fileLoading" class="empty-state">
            <ElIcon class="empty-icon"><DocumentCopy /></ElIcon>
            <p>暂无规则文件</p>
            <ElButton type="primary" @click="handleAddFile">
              <ElIcon><Plus /></ElIcon>
              创建第一个文件
            </ElButton>
          </div>
        </div>
      </div>

      <!-- 选择了文件，未选择分组 -->
      <div v-else-if="!selectedGroup" class="group-select-view">
        <div class="view-header">
          <div class="header-left">
            <ElButton link @click="handleBackToFile">
              <ElIcon><ArrowLeft /></ElIcon>
              返回
            </ElButton>
            <h2>{{ selectedFile.fileName }}</h2>
            <span class="count">{{ groupPagination.total }} 个分组</span>
          </div>
          <ElButton type="primary" @click="handleAddGroup">
            <ElIcon><Plus /></ElIcon>
            新建分组
          </ElButton>
        </div>

        <!-- 分组搜索 -->
        <div class="search-bar">
          <ElInput
            v-model="groupSearchKeyword"
            placeholder="搜索分组名称..."
            clearable
            :prefix-icon="Search"
            class="search-input"
            @input="handleGroupSearchInput"
          />
        </div>

        <!-- 分组卡片列表 -->
        <div v-loading="groupLoading" class="group-card-list">
          <div
            v-for="group in groupList"
            :key="group.id"
            class="group-card"
            @click="handleGroupSelect(group)"
          >
            <div class="card-header">
              <div class="card-title">
                <ElIcon class="card-icon"><FolderOpened /></ElIcon>
                <span class="group-name">{{ group.groupName }}</span>
              </div>
              <ElTag size="small" type="info">{{ group.ruleCount || 0 }} 条规则</ElTag>
            </div>
            <div class="card-body">
              <div class="card-info">
                <span class="label">分组代码</span>
                <code class="value">{{ group.groupCode }}</code>
              </div>
              <div class="card-info">
                <span class="label">评估间隔</span>
                <span class="value">{{ group.interval }}</span>
              </div>
              <div v-if="group.description" class="card-info full">
                <span class="label">描述</span>
                <span class="value">{{ group.description }}</span>
              </div>
            </div>
            <div class="card-footer">
              <span class="card-status">{{ group.isEnabled ? '启用' : '禁用' }}</span>
              <div class="card-actions" @click.stop>
                <ElTooltip content="编辑">
                  <ElButton size="small" @click="handleEditGroup(group)">
                    <ElIcon><Edit /></ElIcon>
                  </ElButton>
                </ElTooltip>
                <ElTooltip content="删除">
                  <ElButton size="small" type="danger" @click="handleDeleteGroup(group)">
                    <ElIcon><Delete /></ElIcon>
                  </ElButton>
                </ElTooltip>
              </div>
            </div>
          </div>

          <div v-if="groupList.length === 0 && !groupLoading" class="empty-state">
            <ElIcon class="empty-icon"><FolderOpened /></ElIcon>
            <p>暂无分组</p>
            <ElButton type="primary" @click="handleAddGroup">
              <ElIcon><Plus /></ElIcon>
              创建第一个分组
            </ElButton>
          </div>
        </div>

        <!-- 分页 -->
        <div v-if="groupPagination.total > groupPagination.size" class="pagination-bar">
          <ElPagination
            v-model:current-page="groupPagination.current"
            :total="groupPagination.total"
            :page-size="groupPagination.size"
            layout="total, prev, pager, next"
            @current-change="loadGroupList"
          />
        </div>
      </div>

      <!-- 选择了分组，显示规则列表 -->
      <div v-else class="rule-view">
        <div class="view-header">
          <div class="header-left">
            <ElButton link @click="handleBackToGroup">
              <ElIcon><ArrowLeft /></ElIcon>
              返回
            </ElButton>
            <h2>{{ selectedGroup.groupName }}</h2>
            <span class="count">{{ rulePagination.total }} 条规则</span>
          </div>
          <div class="header-actions">
            <ElButton @click="handleImport">
              <ElIcon><Upload /></ElIcon>
              导入
            </ElButton>
            <ElButton @click="handleExport">
              <ElIcon><Download /></ElIcon>
              导出
            </ElButton>
            <ElButton type="primary" @click="handleAddRule">
              <ElIcon><Plus /></ElIcon>
              添加规则
            </ElButton>
          </div>
        </div>

        <!-- 规则筛选 -->
        <div class="rule-filter-bar">
          <ElInput
            v-model="ruleSearchForm.ruleNameCn"
            placeholder="搜索规则名称..."
            clearable
            :prefix-icon="Search"
            class="filter-input"
          />
          <ElInput
            v-model="ruleSearchForm.alertName"
            placeholder="搜索告警名称..."
            clearable
            :prefix-icon="Search"
            class="filter-input"
          />
          <ElSelect
            v-model="ruleSearchForm.severity"
            placeholder="严重程度"
            clearable
            class="filter-select"
          >
            <ElOption label="严重" value="critical">
              <span class="severity-item critical"><span class="dot"></span>严重</span>
            </ElOption>
            <ElOption label="警告" value="warning">
              <span class="severity-item warning"><span class="dot"></span>警告</span>
            </ElOption>
            <ElOption label="提示" value="info">
              <span class="severity-item info"><span class="dot"></span>提示</span>
            </ElOption>
          </ElSelect>
          <ElSelect
            v-model="ruleSearchForm.isEnabled"
            placeholder="状态"
            clearable
            class="filter-select"
          >
            <ElOption label="已启用" :value="true" />
            <ElOption label="已禁用" :value="false" />
          </ElSelect>
        </div>

        <!-- 规则表格 -->
        <div v-loading="ruleLoading" class="rule-table-wrapper">
          <ElTable
            :data="ruleList"
            stripe
            border
            style="width: 100%"
            :header-cell-style="{ background: '#f8fafc', color: '#334155', fontWeight: 600 }"
          >
            <ElTableColumn type="selection" width="50" />
            <ElTableColumn prop="ruleNameCn" label="规则名称" min-width="160" fixed="left">
              <template #default="{ row }">
                <div class="rule-name-cell">
                  <span>{{ row.ruleNameCn }}</span>
                  <ElTag v-if="!row.isEnabled" size="small" type="info">禁用</ElTag>
                </div>
              </template>
            </ElTableColumn>
            <ElTableColumn prop="alertName" label="告警名称" min-width="180">
              <template #default="{ row }">
                <code class="alert-name-code">{{ row.alertName }}</code>
              </template>
            </ElTableColumn>
            <ElTableColumn prop="severity" label="严重程度" width="100" align="center">
              <template #default="{ row }">
                <span :class="['severity-badge', row.severity]">
                  <span class="dot"></span>
                  {{ getSeverityLabel(row.severity) }}
                </span>
              </template>
            </ElTableColumn>
            <ElTableColumn prop="forDuration" label="持续时间" width="90" align="center">
              <template #default="{ row }">
                <span class="duration-text">{{ row.forDuration }}</span>
              </template>
            </ElTableColumn>
            <ElTableColumn prop="expr" label="PromQL 表达式" min-width="300">
              <template #default="{ row }">
                <div class="expr-cell">
                  <code class="expr-code">{{ row.expr }}</code>
                  <ElButton size="small" text @click="handleCopyExpr(row.expr)">
                    <ElIcon><CopyDocument /></ElIcon>
                  </ElButton>
                </div>
              </template>
            </ElTableColumn>
            <ElTableColumn prop="summary" label="告警摘要" min-width="200">
              <template #default="{ row }">
                <span class="summary-text">{{ row.summary }}</span>
              </template>
            </ElTableColumn>
            <ElTableColumn label="操作" width="160" fixed="right" align="center">
              <template #default="{ row }">
                <ElButton link type="primary" size="small" @click="handleViewRule(row)">
                  查看
                </ElButton>
                <ElButton link type="primary" size="small" @click="handleEditRule(row)">
                  编辑
                </ElButton>
                <ElButton link type="danger" size="small" @click="handleDeleteRule(row)">
                  删除
                </ElButton>
              </template>
            </ElTableColumn>
          </ElTable>
        </div>

        <!-- 规则分页 -->
        <div class="pagination-bar">
          <span class="total-info">共 {{ rulePagination.total }} 条规则</span>
          <ElPagination
            v-model:current-page="rulePagination.current"
            v-model:page-size="rulePagination.size"
            :page-sizes="[10, 20, 50, 100]"
            :total="rulePagination.total"
            layout="sizes, prev, pager, next, jumper"
            @size-change="handleRuleSizeChange"
            @current-change="handleRulePageChange"
          />
        </div>
      </div>
    </div>

    <!-- 弹窗组件 -->
    <FileForm
      v-model="fileFormVisible"
      :form-data="currentFile"
      :is-edit="isEditFile"
      @success="handleFileFormSuccess"
    />
    <GroupForm
      v-model="groupFormVisible"
      :form-data="currentGroup"
      :is-edit="isEditGroup"
      :file-id="selectedFile?.id"
      @success="handleGroupFormSuccess"
    />
    <RuleForm
      v-model="ruleFormVisible"
      :form-data="currentRule"
      :is-edit="isEditRule"
      :group-id="selectedGroup?.id"
      @success="handleRuleFormSuccess"
    />
    <ImportDialog v-model="importVisible" @success="handleImportSuccess" />
    <DeployDialog
      v-model="deployVisible"
      :file-id="selectedFile?.id"
      :deploy-all="isDeployAll"
      @success="handleDeploySuccess"
    />
    <DownloadConfigDialog v-model="downloadConfigVisible" />
    <ConfigAlertDialog v-model="configAlertVisible" @success="handleConfigAlertSuccess" />
  </div>
</template>

<script setup lang="ts">
  import { ref, reactive, computed, watch, onMounted } from 'vue'
  import { ElMessage, ElMessageBox, ElBreadcrumb, ElBreadcrumbItem } from 'element-plus'
  import {
    Plus,
    Edit,
    Delete,
    Upload,
    Download,
    Position,
    Document,
    DocumentCopy,
    FolderOpened,
    Folder,
    Bell,
    CopyDocument,
    Search,
    Setting,
    MoreFilled,
    ArrowDown,
    ArrowLeft
  } from '@element-plus/icons-vue'
  import FileForm from './components/FileForm.vue'
  import GroupForm from './components/GroupForm.vue'
  import RuleForm from './components/RuleForm.vue'
  import ImportDialog from './components/ImportDialog.vue'
  import DeployDialog from './components/DeployDialog.vue'
  import DownloadConfigDialog from './components/DownloadConfigDialog.vue'
  import ConfigAlertDialog from './components/ConfigAlertDialog.vue'
  import {
    listAlertRuleFileApi,
    deleteAlertRuleFileApi,
    searchAlertRuleGroupApi,
    deleteAlertRuleGroupApi,
    searchAlertRuleApi,
    deleteAlertRuleApi,
    batchExportAlertRulesApi,
    exportAllAlertRulesApi,
    type AlertRuleFile,
    type AlertRuleGroup,
    type AlertRule
  } from '@/api/manager/alert'

  defineOptions({ name: 'AlertManagement' })

  // ==================== 文件相关 ====================
  const fileList = ref<AlertRuleFile[]>([])
  const fileLoading = ref(false)
  const fileSearchKeyword = ref('')
  const selectedFileId = ref<number | null>(null)
  const selectedFile = computed(
    () => fileList.value.find((f) => f.id === selectedFileId.value) || null
  )

  const fileFormVisible = ref(false)
  const isEditFile = ref(false)
  const currentFile = ref<Partial<AlertRuleFile>>({})

  const filteredFileList = computed(() => {
    if (!fileSearchKeyword.value) return fileList.value
    const keyword = fileSearchKeyword.value.toLowerCase()
    return fileList.value.filter(
      (file) =>
        file.fileName.toLowerCase().includes(keyword) ||
        file.fileCode.toLowerCase().includes(keyword)
    )
  })

  const loadFileList = async () => {
    try {
      fileLoading.value = true
      const res = await listAlertRuleFileApi()
      fileList.value = res || []

      if (selectedFileId.value) {
        const found = fileList.value.find((f) => f.id === selectedFileId.value)
        if (!found) {
          selectedFileId.value = null
          groupList.value = []
          ruleList.value = []
        }
      }
    } catch (error) {
      console.error('加载文件列表失败:', error)
    } finally {
      fileLoading.value = false
    }
  }

  const handleFileSelect = (file: AlertRuleFile) => {
    selectedFileId.value = file.id
    selectedGroupId.value = null
    ruleList.value = []
    groupPagination.current = 1
    loadGroupList()
  }

  const handleBackToFile = () => {
    selectedFileId.value = null
    selectedGroupId.value = null
    ruleList.value = []
  }

  const handleAddFile = () => {
    isEditFile.value = false
    currentFile.value = {}
    fileFormVisible.value = true
  }

  const handleEditFile = (file: AlertRuleFile) => {
    isEditFile.value = true
    currentFile.value = { ...file }
    fileFormVisible.value = true
  }

  const handleExportFile = async (file: AlertRuleFile) => {
    try {
      const res = await batchExportAlertRulesApi({ fileId: file.id })
      downloadYaml(res.yamlStr, `${file.fileCode}_alert_rules`)
      ElMessage.success('导出成功')
    } catch (error) {
      console.error('导出失败:', error)
    }
  }

  const handleDeployFile = (file: AlertRuleFile) => {
    selectedFileId.value = file.id
    isDeployAll.value = false
    deployVisible.value = true
  }

  const handleDeleteFile = async (file: AlertRuleFile) => {
    try {
      await ElMessageBox.confirm(
        `确定要删除文件 "${file.fileName}" 吗？该操作将同时删除文件下的所有分组和规则。`,
        '删除确认',
        { type: 'warning', confirmButtonText: '确定删除', cancelButtonText: '取消' }
      )

      await deleteAlertRuleFileApi(file.id, true)
      ElMessage.success('删除成功')

      if (selectedFileId.value === file.id) {
        selectedFileId.value = null
        groupList.value = []
        ruleList.value = []
      }

      await loadFileList()
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('删除文件失败:', error)
      }
    }
  }

  const handleFileFormSuccess = () => {
    loadFileList()
  }

  // ==================== 分组相关 ====================
  const groupList = ref<AlertRuleGroup[]>([])
  const groupLoading = ref(false)
  const groupSearchKeyword = ref('')
  const selectedGroupId = ref<number | null>(null)
  const selectedGroup = computed(
    () => groupList.value.find((g) => g.id === selectedGroupId.value) || null
  )

  const groupPagination = reactive({
    current: 1,
    size: 20,
    total: 0
  })

  const groupFormVisible = ref(false)
  const isEditGroup = ref(false)
  const currentGroup = ref<Partial<AlertRuleGroup>>({})

  const loadGroupList = async () => {
    if (!selectedFile.value) return

    try {
      groupLoading.value = true
      const res = await searchAlertRuleGroupApi({
        fileId: selectedFile.value.id,
        page: groupPagination.current,
        pageSize: groupPagination.size,
        groupName: groupSearchKeyword.value || undefined
      })

      groupList.value = res.items || []
      groupPagination.total = res.total

      if (selectedGroupId.value) {
        const found = groupList.value.find((g) => g.id === selectedGroupId.value)
        if (!found) {
          selectedGroupId.value = null
          ruleList.value = []
        }
      }
    } catch (error) {
      console.error('加载分组列表失败:', error)
    } finally {
      groupLoading.value = false
    }
  }

  let groupSearchTimer: number | null = null
  const handleGroupSearchInput = () => {
    if (groupSearchTimer) clearTimeout(groupSearchTimer)
    groupSearchTimer = window.setTimeout(() => {
      groupPagination.current = 1
      loadGroupList()
    }, 300)
  }

  const handleGroupSelect = (group: AlertRuleGroup) => {
    selectedGroupId.value = group.id
    rulePagination.current = 1
    loadRuleList()
  }

  const handleBackToGroup = () => {
    selectedGroupId.value = null
    ruleList.value = []
  }

  const handleBreadcrumbClick = (level: string) => {
    if (level === 'file') {
      selectedFileId.value = null
      selectedGroupId.value = null
      ruleList.value = []
    } else if (level === 'group') {
      selectedGroupId.value = null
      ruleList.value = []
    }
  }

  const handleAddGroup = () => {
    if (!selectedFile.value) {
      ElMessage.warning('请先选择规则文件')
      return
    }
    isEditGroup.value = false
    currentGroup.value = {}
    groupFormVisible.value = true
  }

  const handleEditGroup = (group: AlertRuleGroup) => {
    isEditGroup.value = true
    currentGroup.value = { ...group }
    groupFormVisible.value = true
  }

  const handleDeleteGroup = async (group: AlertRuleGroup) => {
    try {
      await ElMessageBox.confirm(
        `确定要删除分组 "${group.groupName}" 吗？该操作将同时删除分组下的所有规则。`,
        '删除确认',
        { type: 'warning', confirmButtonText: '确定删除', cancelButtonText: '取消' }
      )

      await deleteAlertRuleGroupApi(group.id, true)
      ElMessage.success('删除成功')

      if (selectedGroupId.value === group.id) {
        selectedGroupId.value = null
        ruleList.value = []
      }

      await loadGroupList()
      await loadFileList()
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('删除分组失败:', error)
      }
    }
  }

  const handleGroupFormSuccess = () => {
    loadGroupList()
    loadFileList()
  }

  // ==================== 规则相关 ====================
  const ruleList = ref<AlertRule[]>([])
  const ruleLoading = ref(false)

  const rulePagination = reactive({
    current: 1,
    size: 10,
    total: 0
  })

  const ruleSearchForm = ref({
    ruleNameCn: '',
    alertName: '',
    severity: '' as '' | 'critical' | 'warning' | 'info',
    isEnabled: undefined as boolean | undefined
  })

  const ruleFormVisible = ref(false)
  const isEditRule = ref(false)
  const currentRule = ref<Partial<AlertRule>>({})

  // 监听搜索条件变化
  watch(
    () => [
      ruleSearchForm.value.ruleNameCn,
      ruleSearchForm.value.alertName,
      ruleSearchForm.value.severity,
      ruleSearchForm.value.isEnabled
    ],
    () => {
      rulePagination.current = 1
      if (selectedGroup.value) {
        loadRuleList()
      }
    }
  )

  const loadRuleList = async () => {
    if (!selectedGroup.value) return

    try {
      ruleLoading.value = true
      const res = await searchAlertRuleApi({
        groupId: selectedGroup.value.id,
        page: rulePagination.current,
        pageSize: rulePagination.size,
        ruleNameCn: ruleSearchForm.value.ruleNameCn || undefined,
        alertName: ruleSearchForm.value.alertName || undefined,
        severity: ruleSearchForm.value.severity || undefined,
        isEnabled: ruleSearchForm.value.isEnabled
      })

      ruleList.value = res.items || []
      rulePagination.total = res.total
    } catch (error) {
      console.error('加载规则列表失败:', error)
    } finally {
      ruleLoading.value = false
    }
  }

  const handleAddRule = () => {
    if (!selectedGroup.value) {
      ElMessage.warning('请先选择告警分组')
      return
    }
    isEditRule.value = false
    currentRule.value = {}
    ruleFormVisible.value = true
  }

  const handleViewRule = (rule: AlertRule) => {
    isEditRule.value = true
    currentRule.value = { ...rule }
    ruleFormVisible.value = true
  }

  const handleEditRule = (rule: AlertRule) => {
    isEditRule.value = true
    currentRule.value = { ...rule }
    ruleFormVisible.value = true
  }

  const handleDeleteRule = async (rule: AlertRule) => {
    try {
      await ElMessageBox.confirm(`确定要删除规则 "${rule.ruleNameCn}" 吗？`, '删除确认', {
        type: 'warning',
        confirmButtonText: '确定删除',
        cancelButtonText: '取消'
      })

      await deleteAlertRuleApi(rule.id)
      ElMessage.success('删除成功')
      await loadRuleList()
      await loadGroupList()
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('删除规则失败:', error)
      }
    }
  }

  const handleRuleFormSuccess = () => {
    loadRuleList()
    loadGroupList()
  }

  const handleRuleSizeChange = (size: number) => {
    rulePagination.size = size
    rulePagination.current = 1
    loadRuleList()
  }

  const handleRulePageChange = (page: number) => {
    rulePagination.current = page
    loadRuleList()
  }

  const getSeverityLabel = (severity: string) => {
    const labelMap: Record<string, string> = {
      critical: '严重',
      warning: '警告',
      info: '提示'
    }
    return labelMap[severity] || severity
  }

  const handleCopyExpr = async (expr: string) => {
    try {
      await navigator.clipboard.writeText(expr)
      ElMessage.success('已复制到剪贴板')
    } catch {
      const textarea = document.createElement('textarea')
      textarea.value = expr
      document.body.appendChild(textarea)
      textarea.select()
      document.execCommand('copy')
      document.body.removeChild(textarea)
      ElMessage.success('已复制到剪贴板')
    }
  }

  // ==================== 批量操作 ====================
  const importVisible = ref(false)
  const deployVisible = ref(false)
  const isDeployAll = ref(false)
  const downloadConfigVisible = ref(false)
  const configAlertVisible = ref(false)

  const handleMoreAction = (command: string) => {
    switch (command) {
      case 'import':
        importVisible.value = true
        break
      case 'exportAll':
        handleExportAll()
        break
      case 'downloadConfig':
        downloadConfigVisible.value = true
        break
      case 'configAlert':
        configAlertVisible.value = true
        break
    }
  }

  const handleImport = () => {
    importVisible.value = true
  }

  const handleExport = async () => {
    if (!selectedFile.value) {
      ElMessage.warning('请先选择规则文件')
      return
    }

    try {
      const res = await batchExportAlertRulesApi({ fileId: selectedFile.value.id })
      downloadYaml(res.yamlStr, `${selectedFile.value.fileCode}_alert_rules`)
      ElMessage.success('导出成功')
    } catch (error) {
      console.error('导出失败:', error)
    }
  }

  const handleExportAll = async () => {
    try {
      const res = await exportAllAlertRulesApi()
      downloadYaml(res.yamlStr, 'all_alert_rules')
      ElMessage.success('导出全部成功')
    } catch (error) {
      console.error('导出全部失败:', error)
    }
  }

  const downloadYaml = (yamlStr: string, filename: string) => {
    const blob = new Blob([yamlStr], { type: 'text/yaml' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${filename}_${Date.now()}.yaml`
    a.click()
    URL.revokeObjectURL(url)
  }

  const handleDeployAll = () => {
    isDeployAll.value = true
    deployVisible.value = true
  }

  const handleDeploySuccess = () => {
    ElMessage.success('部署成功')
  }

  const handleConfigAlertSuccess = () => {
    ElMessage.success('Alertmanager 配置成功')
  }

  const handleImportSuccess = () => {
    loadFileList()
    if (selectedFile.value) {
      loadGroupList()
    }
  }

  // ==================== 生命周期 ====================
  onMounted(() => {
    loadFileList()
  })
</script>

<style lang="scss" scoped>
  .alert-page {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: #f5f7fa;
    padding: 16px;
    box-sizing: border-box;
    gap: 12px;
  }

  // 顶部导航
  .page-nav {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 20px;
    background: #fff;
    border-radius: 8px;
    border: 1px solid #e4e7ed;

    .nav-breadcrumb {
      :deep(.el-breadcrumb) {
        font-size: 14px;

        .el-breadcrumb__item {
          .el-breadcrumb__inner {
            display: flex;
            align-items: center;
            gap: 6px;
            color: #6b7280;
            font-weight: 500;

            .el-icon {
              font-size: 16px;
            }
          }

          &:last-child .el-breadcrumb__inner {
            color: #1f2937;
          }

          .el-breadcrumb__separator {
            color: #d1d5db;
          }
        }

        .clickable {
          cursor: pointer;

          .el-breadcrumb__inner:hover {
            color: #3b82f6;
          }
        }
      }
    }

    .nav-actions {
      display: flex;
      gap: 10px;

      .arrow {
        margin-left: 4px;
        font-size: 12px;
      }
    }
  }

  // 主内容区
  .page-main {
    flex: 1;
    display: flex;
    flex-direction: column;
    background: #fff;
    border-radius: 8px;
    border: 1px solid #e4e7ed;
    overflow: hidden;
    min-height: 0;
  }

  // 视图通用样式
  .file-select-view,
  .group-select-view,
  .rule-view {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .view-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid #f3f4f6;

    .header-left {
      display: flex;
      align-items: center;
      gap: 12px;

      h2 {
        font-size: 16px;
        font-weight: 600;
        color: #1f2937;
        margin: 0;
      }

      .count {
        font-size: 13px;
        color: #9ca3af;
        background: #f3f4f6;
        padding: 2px 10px;
        border-radius: 12px;
      }
    }

    .header-actions {
      display: flex;
      gap: 10px;
    }
  }

  .search-bar {
    padding: 16px 20px;

    .search-input {
      width: 320px;
    }
  }

  // 文件卡片列表
  .file-card-list {
    flex: 1;
    overflow-y: auto;
    padding: 0 20px 20px;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
    gap: 16px;
    align-content: start;
  }

  .file-card {
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    overflow: hidden;
    cursor: pointer;
    transition: all 0.2s;

    &:hover {
      border-color: #3b82f6;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
    }

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 12px 16px;
      background: #f9fafb;
      border-bottom: 1px solid #e5e7eb;

      .card-title {
        display: flex;
        align-items: center;
        gap: 8px;

        .card-icon {
          font-size: 18px;
          color: #6b7280;
        }

        .file-name {
          font-size: 14px;
          font-weight: 600;
          color: #1f2937;
        }
      }
    }

    .card-body {
      padding: 16px;
      display: grid;
      grid-template-columns: repeat(2, 1fr);
      gap: 12px;

      .card-info {
        display: flex;
        flex-direction: column;
        gap: 4px;

        &.full {
          grid-column: 1 / -1;
        }

        .label {
          font-size: 12px;
          color: #9ca3af;
        }

        .value {
          font-size: 13px;
          color: #374151;
          font-family: inherit;

          &.code {
            font-family: var(--ax-font-mono);
            background: #f3f4f6;
            padding: 2px 8px;
            border-radius: 4px;
          }
        }
      }
    }

    .card-footer {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 12px 16px;
      border-top: 1px solid #e5e7eb;
      background: #f9fafb;

      .card-stats {
        display: flex;
        gap: 16px;

        .stat {
          display: flex;
          align-items: center;
          gap: 4px;
          font-size: 13px;
          color: #6b7280;

          .el-icon {
            font-size: 14px;
          }
        }
      }

      .card-actions {
        display: flex;
        gap: 4px;
        opacity: 0;
        transition: opacity 0.2s;
      }
    }

    &:hover .card-actions {
      opacity: 1;
    }
  }

  // 分组卡片列表
  .group-card-list {
    flex: 1;
    overflow-y: auto;
    padding: 0 20px 20px;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: 16px;
    align-content: start;
  }

  .group-card {
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    overflow: hidden;
    cursor: pointer;
    transition: all 0.2s;

    &:hover {
      border-color: #3b82f6;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
    }

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 12px 16px;
      background: #f9fafb;
      border-bottom: 1px solid #e5e7eb;

      .card-title {
        display: flex;
        align-items: center;
        gap: 8px;

        .card-icon {
          font-size: 18px;
          color: #6b7280;
        }

        .group-name {
          font-size: 14px;
          font-weight: 600;
          color: #1f2937;
        }
      }
    }

    .card-body {
      padding: 16px;
      display: grid;
      grid-template-columns: repeat(2, 1fr);
      gap: 12px;

      .card-info {
        display: flex;
        flex-direction: column;
        gap: 4px;

        &.full {
          grid-column: 1 / -1;
        }

        .label {
          font-size: 12px;
          color: #9ca3af;
        }

        .value {
          font-size: 13px;
          color: #374151;

          &.code {
            font-family: var(--ax-font-mono);
            background: #f3f4f6;
            padding: 2px 8px;
            border-radius: 4px;
          }
        }
      }
    }

    .card-footer {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 12px 16px;
      border-top: 1px solid #e5e7eb;
      background: #f9fafb;

      .card-status {
        font-size: 13px;
        color: #6b7280;
      }

      .card-actions {
        display: flex;
        gap: 4px;
        opacity: 0;
        transition: opacity 0.2s;
      }
    }

    &:hover .card-actions {
      opacity: 1;
    }
  }

  // 空状态
  .empty-state {
    grid-column: 1 / -1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 20px;
    color: #9ca3af;

    .empty-icon {
      font-size: 64px;
      margin-bottom: 16px;
      color: #d1d5db;
    }

    p {
      font-size: 14px;
      margin: 0 0 20px 0;
    }
  }

  // 分页条
  .pagination-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 20px;
    border-top: 1px solid #f3f4f6;

    .total-info {
      font-size: 13px;
      color: #6b7280;
    }
  }

  // 规则筛选栏
  .rule-filter-bar {
    display: flex;
    gap: 10px;
    padding: 16px 20px;
    border-bottom: 1px solid #f3f4f6;
    flex-wrap: wrap;

    .filter-input {
      width: 200px;
    }

    .filter-select {
      width: 140px;
    }
  }

  // 规则表格
  .rule-table-wrapper {
    flex: 1;
    overflow: hidden;
    padding: 0 20px;

    :deep(.el-table) {
      border-radius: 8px;
      overflow: hidden;

      .el-table__header-wrapper th {
        background: #f8fafc !important;
      }
    }

    .rule-name-cell {
      display: flex;
      align-items: center;
      gap: 8px;
      font-weight: 500;
      color: #1f2937;
    }

    .alert-name-code {
      font-family: var(--ax-font-mono);
      font-size: 12px;
      color: #4b5563;
      background: #f3f4f6;
      padding: 4px 8px;
      border-radius: 4px;
    }

    .severity-badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      font-size: 13px;
      font-weight: 500;

      .dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
      }

      &.critical {
        color: #dc2626;

        .dot {
          background: #dc2626;
        }
      }

      &.warning {
        color: #d97706;

        .dot {
          background: #d97706;
        }
      }

      &.info {
        color: #2563eb;

        .dot {
          background: #2563eb;
        }
      }
    }

    .duration-text {
      font-size: 13px;
      color: #6b7280;
    }

    .expr-cell {
      display: flex;
      align-items: center;
      gap: 8px;

      .expr-code {
        flex: 1;
        font-family: var(--ax-font-mono);
        font-size: 12px;
        color: #4b5563;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
    }

    .summary-text {
      font-size: 13px;
      color: #6b7280;
      display: -webkit-box;
      -webkit-line-clamp: 2;
      -webkit-box-orient: vertical;
      overflow: hidden;
    }
  }

  // 严重程度选项
  :deep(.el-select-dropdown__item) {
    .severity-item {
      display: flex;
      align-items: center;
      gap: 8px;

      .dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
      }

      &.critical .dot {
        background: #dc2626;
      }

      &.warning .dot {
        background: #d97706;
      }

      &.info .dot {
        background: #2563eb;
      }
    }
  }
</style>
