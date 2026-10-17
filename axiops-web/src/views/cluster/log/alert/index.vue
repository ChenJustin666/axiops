<template>
  <div class="log-alert-page">
    <ElCard shadow="never" class="command-card">
      <div class="command-grid">
        <div class="toolbar-title">
          <strong>日志告警</strong>
          <span v-if="selectedClusterUuid">
            {{ currentClusterName }} · 健康度 {{ healthStatusText }} · 默认后端
            {{ backendTypeText(clusterDefaultBackend) }}
          </span>
          <span v-else>请选择集群</span>
        </div>
        <div class="command-actions">
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
          <ElButton :icon="Refresh" @click="refreshAll" :loading="loading">刷新</ElButton>
          <ElButton
            type="primary"
            :icon="Plus"
            :disabled="!selectedClusterUuid"
            @click="openCreateEntry"
          >
            新建规则
          </ElButton>
        </div>
      </div>
    </ElCard>

    <ElCard v-if="!selectedClusterUuid" shadow="never">
      <div class="empty-shell">
        <ElEmpty description="请先选择集群，再进入该集群的日志告警工作台。" />
        <div class="empty-guides">
          <div class="empty-guide-card">
            <div class="empty-guide-title">推荐先做什么</div>
            <div class="empty-guide-desc"
              >先选集群，再从内置模板创建第一条规则，最后微调项目、命名空间与关键字。</div
            >
          </div>
          <div class="empty-guide-card">
            <div class="empty-guide-title">适合谁来用</div>
            <div class="empty-guide-desc"
              >业务用户先用模板，平台用户再补评估间隔、持续命中和静默窗口。</div
            >
          </div>
        </div>
      </div>
    </ElCard>

    <template v-else>
      <div class="stats-grid">
        <div class="stat-card">
          <div class="label">规则总数</div>
          <div class="value">{{ syncStats.totalRules }}</div>
        </div>
        <div class="stat-card">
          <div class="label">启用规则</div>
          <div class="value">{{ syncStats.enabledRules }}</div>
        </div>
        <div class="stat-card">
          <div class="label">同步成功</div>
          <div class="value success">{{ syncStats.successRules }}</div>
        </div>
        <div class="stat-card">
          <div class="label">同步失败</div>
          <div class="value danger">{{ syncStats.failedRules }}</div>
        </div>
        <div class="stat-card">
          <div class="label">待评估</div>
          <div class="value warning">{{ syncStats.pendingRules }}</div>
        </div>
        <div class="stat-card">
          <div class="label">健康状态</div>
          <div class="value">{{ healthStatusText }}</div>
        </div>
      </div>

      <ArtSearchBar
        v-show="showSearchBar"
        :model-value="filters"
        :items="ruleSearchItems"
        @search="applyFilters"
        @reset="clearFilters"
      />

      <ElCard
        shadow="never"
        class="table-card art-table-card"
        :style="{ 'margin-top': showSearchBar ? '12px' : '0' }"
      >
        <ArtTableHeader
          v-model:columns="ruleColumns"
          v-model:showSearchBar="showSearchBar"
          :loading="loading"
          :showZebra="true"
          :showBorder="true"
          :showHeaderBackground="true"
          :fullClass="'art-page-view'"
          :layout="'search,refresh,size,fullscreen,columns,settings'"
          @refresh="refreshAll"
        >
          <template #left>
            <ElSpace wrap>
              <div>
                <div class="card-header">规则列表</div>
                <div class="card-subtitle">支持模板创建、手工创建、启停和删除。</div>
              </div>
              <ElTag effect="plain" type="info">当前 {{ displayRules.length }} 条</ElTag>
            </ElSpace>
          </template>
        </ArtTableHeader>
        <ArtTable
          :data="displayRules"
          :loading="loading"
          :columns="visibleRuleColumns"
          :show-table-header="false"
          empty-text="暂无规则，可通过右上角新建规则或使用模板快速创建。"
        >
          <template #backend="{ row }">
            <ElTag effect="plain">{{ backendTypeText(row.backendType) }}</ElTag>
          </template>
          <template #severity="{ row }">
            <ElTag :type="severityTagType(row.severity)">{{ severityText(row.severity) }}</ElTag>
          </template>
          <template #scope="{ row }">
            <div class="scope-table-cell">
              <ElTag effect="plain" type="info">{{ ruleScopeTypeText(row) }}</ElTag>
              <span>{{ ruleScopeTargetText(row) }}</span>
            </div>
          </template>
          <template #conditionType="{ row }">
            {{ conditionTypeText(row.conditionType) }}
          </template>
          <template #strategy="{ row }">
            <div class="strategy-table-cell">
              <span>评估 {{ row.evalInterval || '60s' }}</span>
              <span>持续 {{ row.forDuration || '0s' }}</span>
              <span>静默 {{ row.silenceWindow || '15m' }}</span>
            </div>
          </template>
          <template #lastSyncStatus="{ row }">
            <ElTag :type="syncStatusTagType(row.lastSyncStatus)">{{ row.lastSyncStatus }}</ElTag>
          </template>
          <template #lastSyncError="{ row }">{{ row.lastSyncError || '-' }}</template>
          <template #updatedAt="{ row }">{{ formatTime(row.updatedAt) }}</template>
          <template #operation="{ row }">
            <ElSpace class="row-actions" wrap>
              <ElButton link type="primary" @click="openEdit(row)">编辑</ElButton>
              <ElButton link :type="row.enabled ? 'warning' : 'success'" @click="toggleRule(row)">
                {{ row.enabled ? '禁用' : '启用' }}
              </ElButton>
              <ElPopconfirm title="确定删除这条规则吗？" @confirm="removeRule(row.id)">
                <template #reference>
                  <ElButton link type="danger">删除</ElButton>
                </template>
              </ElPopconfirm>
            </ElSpace>
          </template>
        </ArtTable>
      </ElCard>
    </template>

    <ElDialog
      v-model="dialogVisible"
      :title="dialogMode === 'create' ? '新建日志告警' : '编辑日志告警'"
      width="760px"
    >
      <ElForm ref="formRef" :model="form" :rules="rules" label-width="110px">
        <div v-if="selectedTemplate" class="template-source-banner">
          <div>
            <div class="template-source-title"
              >当前正在使用内置模板：{{ selectedTemplate.label }}</div
            >
            <div class="template-source-desc">{{ selectedTemplate.description }}</div>
          </div>
          <ElTag :type="severityTagType(selectedTemplate.payload.severity)" effect="dark">
            {{ severityText(selectedTemplate.payload.severity) }}
          </ElTag>
        </div>

        <div class="form-section-title">告警范围</div>
        <div class="form-section-subtitle"
          >先确定规则作用在哪一层，再继续选择项目、工作空间、服务或版本。</div
        >
        <div class="scope-option-grid">
          <button
            v-for="item in scopeOptions"
            :key="item.value"
            type="button"
            class="scope-option-card"
            :class="{ active: scopeMode === item.value }"
            @click="selectScopeMode(item.value)"
          >
            <div class="scope-option-title">{{ item.label }}</div>
            <div class="scope-option-desc">{{ item.description }}</div>
          </button>
        </div>
        <div class="scope-summary-card">
          <div class="scope-summary-title">{{ scopeSummary }}</div>
          <div class="scope-summary-desc">{{ scopeHelperText }}</div>
        </div>
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="集群" prop="clusterUuid">
              <ElInput :model-value="currentClusterName" disabled />
            </ElFormItem>
          </ElCol>
          <ElCol v-if="scopeMode !== 'cluster'" :span="12">
            <ElFormItem label="项目">
              <ElSelect
                v-model="scopeProjectId"
                filterable
                placeholder="选择项目"
                @change="handleScopeProjectChange"
              >
                <ElOption
                  v-for="item in projectOptions"
                  :key="item.id"
                  :label="item.name"
                  :value="item.id"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElRow v-if="scopeMode !== 'cluster'" :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="工作空间">
              <ElSelect
                :model-value="form.workspaceId || undefined"
                filterable
                placeholder="选择工作空间"
                :disabled="!scopeProjectClusterId"
                :loading="workspaceLoading || scopeClusterLoading"
                @change="handleScopeWorkspaceChange"
              >
                <ElOption
                  v-for="item in workspaceOptions"
                  :key="item.id"
                  :label="item.label"
                  :value="item.id"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="命名空间">
              <ElInput :model-value="selectedWorkspace?.namespace || '-'" disabled />
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElRow v-if="scopeMode === 'service'" :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="服务">
              <ElSelect
                :model-value="selectedApplicationId"
                filterable
                placeholder="选择服务"
                :disabled="!form.workspaceId"
                :loading="applicationLoading"
                @change="handleScopeApplicationChange"
              >
                <ElOption
                  v-for="item in applicationOptions"
                  :key="item.id"
                  :label="item.nameCn || item.nameEn"
                  :value="item.id"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="版本">
              <ElSelect
                :model-value="selectedVersionId"
                clearable
                filterable
                placeholder="不选则覆盖该服务全部版本"
                :disabled="!selectedApplicationId"
                :loading="versionLoading"
                @change="handleScopeVersionChange"
              >
                <ElOption
                  v-for="item in versionOptions"
                  :key="item.id"
                  :label="`${item.version} (${item.resourceName})`"
                  :value="item.id"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElAlert
          v-if="scopeMode === 'cluster'"
          type="warning"
          :closable="false"
          title="当前为集群全局规则，会覆盖当前集群所有命名空间，仅建议 SUPER_ADMIN 配置通用异常。"
          style="margin-bottom: 16px"
        />

        <div class="form-section-title">规则基础信息</div>
        <div class="form-section-subtitle"
          >给规则起一个一眼能看懂的名字，方便后续筛选、联调和排障。</div
        >
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="规则名称" prop="name">
              <ElInput v-model="form.name" placeholder="例如 支付服务 Java 异常告警" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="启用状态">
              <ElSwitch v-model="form.enabled" />
            </ElFormItem>
          </ElCol>
        </ElRow>

        <div class="form-section-title">规则定义</div>
        <div class="form-section-subtitle"
          >先选择触发逻辑，再配置窗口、阈值和降噪策略。无匹配日志与日志停滞的语义已经拆开。</div
        >
        <div class="definition-block">
          <ElRow :gutter="16">
            <ElCol :span="12">
              <ElFormItem label="条件类型" prop="conditionType">
                <ElSelect v-model="form.conditionType">
                  <ElOption
                    v-for="item in conditionOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </ElFormItem>
            </ElCol>
            <ElCol :span="12">
              <ElFormItem label="级别" prop="severity">
                <ElSelect v-model="form.severity">
                  <ElOption
                    v-for="item in severityOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </ElFormItem>
            </ElCol>
          </ElRow>
        </div>

        <div class="definition-block subtle">
          <ElRow :gutter="16">
            <ElCol :span="12">
              <ElFormItem label="阈值" prop="threshold">
                <ElInput
                  v-model="form.threshold"
                  :disabled="form.conditionType === 'no_data' || form.conditionType === 'flatline'"
                />
              </ElFormItem>
              <div class="field-tip">{{ thresholdHint }}</div>
            </ElCol>
            <ElCol :span="12">
              <ElFormItem label="时间窗口" prop="window">
                <ElInput v-model="form.window" placeholder="例如 5m" />
              </ElFormItem>
              <div class="field-tip">建议统一用 `s / m / h / d`，例如 `30s`、`5m`、`1h`。</div>
            </ElCol>
          </ElRow>
        </div>

        <div class="condition-helper">
          <div class="condition-helper-title">判定说明</div>
          <div class="condition-helper-text">{{ conditionHint }}</div>
        </div>

        <div class="definition-block subtle">
          <ElRow :gutter="16">
            <ElCol :span="8">
              <ElFormItem label="评估间隔" prop="evalInterval">
                <ElInput v-model="form.evalInterval" placeholder="例如 30s" />
              </ElFormItem>
              <div class="field-tip">平台每隔多久检查一次该规则，建议不要低于 15s。</div>
            </ElCol>
            <ElCol :span="8">
              <ElFormItem label="持续命中" prop="forDuration">
                <ElInput v-model="form.forDuration" placeholder="例如 90s，0s 表示立即告警" />
              </ElFormItem>
              <div class="field-tip">连续满足条件多久才真正告警，用来降低瞬时抖动误报。</div>
            </ElCol>
            <ElCol :span="8">
              <ElFormItem label="静默窗口" prop="silenceWindow">
                <ElInput v-model="form.silenceWindow" placeholder="例如 15m" />
              </ElFormItem>
              <div class="field-tip">同一告警触发后，在静默窗口内不会重复发通知。</div>
            </ElCol>
          </ElRow>
        </div>

        <div class="condition-helper muted">
          <div class="condition-helper-title">触发策略说明</div>
          <div class="condition-helper-text">{{ strategyHint }}</div>
        </div>

        <div
          v-if="form.conditionType === 'no_data' || form.conditionType === 'flatline'"
          class="condition-example-grid"
        >
          <div v-for="item in conditionExamples" :key="item.title" class="condition-example-card">
            <div class="condition-example-title">{{ item.title }}</div>
            <div class="condition-example-desc">{{ item.description }}</div>
          </div>
        </div>

        <ElFormItem label="规则描述">
          <ElInput v-model="form.description" type="textarea" :rows="2" />
        </ElFormItem>

        <ElAlert
          v-if="form.conditionType === 'no_data' || form.conditionType === 'flatline'"
          type="info"
          :closable="false"
          :title="
            form.conditionType === 'no_data'
              ? '无匹配日志：可填写查询内容，表示窗口内没有匹配到这类日志就报警；不填则表示范围内完全无日志。'
              : '日志停滞：默认检查当前范围内是否完全没有业务日志输出，不需要填写查询内容。'
          "
          style="margin-bottom: 16px"
        />

        <ElFormItem :label="queryTextFieldLabel" prop="queryText">
          <ElInput
            v-model="form.queryText"
            type="textarea"
            :rows="4"
            :disabled="isFlatlineCondition"
            :placeholder="queryTextPlaceholder"
          />
        </ElFormItem>
        <div class="field-tip query-tip">{{ queryTextTip }}</div>

        <ElCard v-if="testResult" shadow="never" class="test-result-card">
          <div class="test-result-header">
            <div class="test-result-title">规则测试结果</div>
            <ElTag :type="testResult.triggered ? 'danger' : 'success'">
              {{ testResult.triggered ? '当前会触发' : '当前不会触发' }}
            </ElTag>
          </div>
          <div class="test-result-grid">
            <div class="test-result-item">
              <span class="test-label">命中日志数</span>
              <strong>{{ testResult.count }}</strong>
            </div>
            <div class="test-result-item">
              <span class="test-label">后端</span>
              <strong>{{ testResult.backendType || '-' }}</strong>
            </div>
            <div class="test-result-item">
              <span class="test-label">查询模式</span>
              <strong>{{ testResult.queryMode || '-' }}</strong>
            </div>
            <div class="test-result-item">
              <span class="test-label">测试时间</span>
              <strong>{{ formatTime(Math.floor(testResult.evaluatedAt / 1000)) }}</strong>
            </div>
          </div>
          <div class="test-summary">
            {{ testSummary }}
          </div>
          <div class="test-sample-block">
            <div class="test-sample-title">命中样本</div>
            <div v-if="testResult.samples.length === 0" class="test-sample-empty">
              当前没有命中样本。
            </div>
            <div v-else class="test-sample-list">
              <div
                v-for="(item, index) in testResult.samples"
                :key="index"
                class="test-sample-item"
              >
                <span class="sample-time">{{ formatTime(Math.floor(item.timestamp / 1000)) }}</span>
                <span class="sample-message">{{ item.message }}</span>
              </div>
            </div>
          </div>
        </ElCard>
      </ElForm>

      <template #footer>
        <ElButton @click="dialogVisible = false">取消</ElButton>
        <ElButton :loading="testing" @click="testRule">规则测试</ElButton>
        <ElButton type="primary" :loading="submitting" @click="submitForm">保存</ElButton>
      </template>
    </ElDialog>

    <ElDialog v-model="createEntryVisible" title="新增日志告警" width="760px">
      <div class="dialog-heading">
        <div class="dialog-heading-title">先选择创建方式</div>
        <div class="dialog-heading-desc">普通用户建议先从模板开始，再微调查询条件和阈值。</div>
      </div>
      <div class="create-entry-grid">
        <div class="entry-card" @click="openCreate(false)">
          <div class="entry-badge">01</div>
          <div class="entry-title">空白新建</div>
          <div class="entry-desc">从空白规则开始，手动填写查询条件、阈值和级别。</div>
          <div class="entry-meta">适合高级用户</div>
        </div>
        <div class="entry-card primary" @click="openTemplateChooser">
          <div class="entry-badge">02</div>
          <div class="entry-title">内置模板</div>
          <div class="entry-desc">从常见异常模板开始，自动填充规则，降低使用门槛。</div>
          <div class="entry-meta">推荐普通用户优先使用</div>
        </div>
      </div>
    </ElDialog>

    <ElDialog v-model="templateChooserVisible" title="选择内置模板" width="860px">
      <div class="dialog-heading compact">
        <div class="dialog-heading-title">选择更接近业务场景的模板</div>
        <div class="dialog-heading-desc">
          模板会自动填充查询内容、条件类型、阈值和级别，保存前仍可继续编辑。
        </div>
      </div>
      <div class="template-toolbar">
        <ElInput v-model="templateKeyword" clearable placeholder="搜索模板，如 Java、OOM、5xx" />
        <ElTag effect="plain" type="info">共 {{ filteredTemplates.length }} 个模板</ElTag>
      </div>
      <div class="template-layout">
        <div class="template-grid">
          <div
            v-for="item in filteredTemplates"
            :key="item.key"
            class="template-item"
            :class="{ active: templatePreviewKey === item.key }"
            @click="templatePreviewKey = item.key"
          >
            <div class="template-title-row">
              <div class="template-title">{{ item.label }}</div>
              <ElTag effect="plain" size="small">{{ item.category }}</ElTag>
            </div>
            <div class="template-desc">{{ item.description }}</div>
            <div class="template-meta">
              <span>{{ conditionTypeText(item.payload.conditionType) }}</span>
              <span>{{ severityText(item.payload.severity) }}</span>
              <span>{{ item.payload.window }}</span>
            </div>
          </div>
        </div>
        <div v-if="templatePreview" class="template-preview">
          <div class="template-preview-header">
            <div>
              <div class="template-preview-title">{{ templatePreview.label }}</div>
              <div class="template-preview-subtitle">{{ templatePreview.description }}</div>
            </div>
            <ElTag effect="dark" :type="severityTagType(templatePreview.payload.severity)">
              {{ severityText(templatePreview.payload.severity) }}
            </ElTag>
          </div>
          <div class="template-preview-grid">
            <div class="template-preview-item">
              <span>类别</span>
              <strong>{{ templatePreview.category }}</strong>
            </div>
            <div class="template-preview-item">
              <span>建议人群</span>
              <strong>{{ templatePreview.audience }}</strong>
            </div>
            <div class="template-preview-item">
              <span>触发条件</span>
              <strong>{{ conditionTypeText(templatePreview.payload.conditionType) }}</strong>
            </div>
            <div class="template-preview-item">
              <span>默认窗口</span>
              <strong>{{ templatePreview.payload.window }}</strong>
            </div>
          </div>
          <div class="preview-query-card">
            <div class="preview-query-label">默认查询</div>
            <div class="preview-query-text">{{ templatePreview.payload.queryText }}</div>
          </div>
          <div class="preview-tip-list">
            <div v-for="tip in templatePreview.tips" :key="tip" class="preview-tip-item">
              {{ tip }}
            </div>
          </div>
          <div class="preview-actions">
            <ElButton @click="templateChooserVisible = false">稍后再选</ElButton>
            <ElButton type="primary" @click="applyTemplate(templatePreview)">使用这个模板</ElButton>
          </div>
        </div>
      </div>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref, watch } from 'vue'
  import dayjs from 'dayjs'
  import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
  import { Plus, Refresh } from '@element-plus/icons-vue'
  import {
    searchProjectApi,
    searchProjectClusterApi,
    searchProjectWorkspaceApi,
    type Project,
    type ProjectCluster,
    type ProjectWorkspace
  } from '@/api/manager/project'
  import {
    createLogAlertRuleApi,
    deleteLogAlertRuleApi,
    disableLogAlertRuleApi,
    getLogClusterConfigApi,
    enableLogAlertRuleApi,
    getLogClusterSyncStatusApi,
    listLogAlertRulesApi,
    queryLogsApi,
    updateLogAlertRuleApi,
    type CreateLogAlertRuleRequest,
    type LogAlertRule
  } from '@/api/manager/log'
  import {
    searchApplicationApi,
    searchVersionApi,
    type OnecProjectApplication,
    type OnecProjectVersion
  } from '@/api/workload/application'
  import { useUserStore } from '@/store/modules/user'
  import { useLogClusters } from '../shared/useLogClusters'

  defineOptions({ name: 'ClusterLogAlertPage' })

  type ScopeMode = 'cluster' | 'namespace' | 'service'

  interface WorkspaceOption {
    id: number
    label: string
    namespace: string
    name: string
  }

  interface TestResult {
    count: number
    triggered: boolean
    backendType: string
    queryMode: string
    evaluatedAt: number
    samples: Array<{
      timestamp: number
      message: string
    }>
  }

  const conditionOptions = [
    { label: '命中数大于阈值', value: 'count_gt' },
    { label: '命中数大于等于阈值', value: 'count_gte' },
    { label: '命中数小于阈值', value: 'count_lt' },
    { label: '命中数小于等于阈值', value: 'count_lte' },
    { label: '命中即告警', value: 'match_any' },
    { label: '无匹配日志', value: 'no_data' },
    { label: '日志停滞（无任何日志）', value: 'flatline' }
  ] as const

  const severityOptions = [
    { label: '严重', value: 'critical' },
    { label: '警告', value: 'warning' },
    { label: '提示', value: 'info' }
  ] as const

  const templates = [
    {
      key: 'java-stacktrace',
      label: 'Java 堆栈异常',
      category: '应用异常',
      audience: 'Java 服务',
      description: '匹配 Exception、Caused by 和常见运行时异常。',
      tips: ['适合 Java / Spring 服务', '建议补充服务或版本范围，减少误报'],
      payload: {
        name: 'Java 堆栈异常',
        description: '匹配 Java 堆栈与异常链路',
        queryText:
          'Exception OR "Caused by:" OR NullPointerException OR RuntimeException OR IllegalStateException',
        conditionType: 'count_gt',
        threshold: '1',
        window: '5m',
        severity: 'warning'
      }
    },
    {
      key: 'contains-error',
      label: 'Error 关键字',
      category: '通用异常',
      audience: '所有服务',
      description: '匹配常见 ERROR / Error / error 关键字异常。',
      tips: ['适合作为第一条通用告警', '建议至少限定到命名空间或服务范围'],
      payload: {
        name: 'Error 关键字异常',
        description: '匹配日志正文中的 error 关键字',
        queryText: 'ERROR OR Error OR error',
        conditionType: 'count_gt',
        threshold: '1',
        window: '5m',
        severity: 'warning'
      }
    },
    {
      key: 'oom',
      label: 'OOM 内存异常',
      category: '资源异常',
      audience: '容器 / JVM',
      description: '匹配 OOMKilled、OutOfMemoryError、Java heap space。',
      tips: ['建议设置为严重', '命中后优先排查资源限制和内存泄漏'],
      payload: {
        name: 'OOM 内存异常',
        description: '匹配容器与 JVM 内存溢出',
        queryText: 'OOMKilled OR OutOfMemoryError OR "Java heap space"',
        conditionType: 'count_gt',
        threshold: '1',
        window: '5m',
        severity: 'critical'
      }
    },
    {
      key: 'http-5xx',
      label: 'HTTP 5xx',
      category: '接口异常',
      audience: '网关 / API 服务',
      description: '匹配 500 / 502 / 503 / 504 服务端异常。',
      tips: ['适合接口层快速发现服务端错误', '建议配合服务或版本范围使用'],
      payload: {
        name: 'HTTP 5xx 异常',
        description: '匹配常见 5xx 服务端错误',
        queryText: '500 OR 502 OR 503 OR 504',
        conditionType: 'count_gt',
        threshold: '5',
        window: '5m',
        severity: 'critical'
      }
    },
    {
      key: 'heartbeat-no-data',
      label: '心跳丢失',
      category: '无数据告警',
      audience: '守护进程 / 心跳任务',
      description: '窗口内没有任何心跳日志时触发。',
      tips: ['适合守护进程、采集器和定时心跳', '该模板按 0 命中处理'],
      payload: {
        name: '心跳日志缺失',
        description: '窗口内没有心跳日志，可能服务停止或链路断开',
        queryText: 'heartbeat',
        conditionType: 'no_data',
        threshold: '0',
        window: '5m',
        severity: 'critical'
      }
    },
    {
      key: 'flatline-running',
      label: '业务日志停滞',
      category: '静默告警',
      audience: '任务型服务',
      description: '窗口内没有业务运行日志时触发静默告警。',
      tips: ['适合消费、调度、任务执行场景', '建议配合较长窗口，避免瞬时波动误报'],
      payload: {
        name: '业务日志停滞',
        description: '窗口内无运行日志，怀疑任务停摆',
        queryText: '',
        conditionType: 'flatline',
        threshold: '0',
        window: '10m',
        severity: 'warning'
      }
    }
  ] as const

  const {
    clusters,
    loading: clusterLoading,
    selectedClusterId,
    selectedClusterUuid,
    loadClusters,
    setCluster
  } = useLogClusters('cluster-log-alert-cluster')

  const userStore = useUserStore()
  const loading = ref(false)
  const submitting = ref(false)
  const testing = ref(false)
  const showSearchBar = ref(true)
  const testResult = ref<TestResult | null>(null)
  const dialogVisible = ref(false)
  const createEntryVisible = ref(false)
  const templateChooserVisible = ref(false)
  const templateKeyword = ref('')
  const templatePreviewKey = ref('')
  const appliedTemplateKey = ref('')
  const dialogMode = ref<'create' | 'edit'>('create')
  const editingId = ref<number | null>(null)
  const rulesData = ref<LogAlertRule[]>([])
  const projectOptions = ref<Project[]>([])
  const workspaceOptions = ref<WorkspaceOption[]>([])
  const applicationOptions = ref<OnecProjectApplication[]>([])
  const versionOptions = ref<OnecProjectVersion[]>([])
  const formRef = ref<FormInstance>()
  const clusterDefaultBackend = ref('')

  const scopeMode = ref<ScopeMode>('namespace')
  const scopeProjectId = ref<number>()
  const scopeProjectClusterId = ref<number>()
  const selectedApplicationId = ref<number>()
  const selectedVersionId = ref<number>()
  const scopeClusterLoading = ref(false)
  const workspaceLoading = ref(false)
  const applicationLoading = ref(false)
  const versionLoading = ref(false)

  const syncStats = reactive({
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

  const filters = reactive({
    projectUuid: '',
    namespace: '',
    enabled: undefined as boolean | undefined,
    keyword: ''
  })
  const ruleColumns = ref([
    { prop: 'name', label: '规则名称', minWidth: 220 },
    { prop: 'backend', label: '后端', width: 110, useSlot: true },
    { prop: 'severity', label: '级别', width: 110, useSlot: true },
    { prop: 'scope', label: '范围', minWidth: 220, useSlot: true },
    { prop: 'conditionType', label: '条件', width: 150, useSlot: true },
    { prop: 'threshold', label: '阈值', width: 100 },
    { prop: 'window', label: '窗口', width: 100 },
    { prop: 'strategy', label: '策略', minWidth: 220, useSlot: true },
    { prop: 'queryText', label: '查询内容', minWidth: 240, showOverflowTooltip: true },
    { prop: 'lastSyncStatus', label: '同步状态', width: 120, useSlot: true },
    {
      prop: 'lastSyncError',
      label: '错误信息',
      minWidth: 220,
      useSlot: true,
      showOverflowTooltip: true
    },
    { prop: 'updatedAt', label: '更新时间', width: 180, useSlot: true },
    { prop: 'operation', label: '操作', fixed: 'right' as const, width: 240, useSlot: true }
  ])
  const visibleRuleColumns = computed(() =>
    ruleColumns.value.filter((column: any) => column.visible !== false)
  )
  const appliedFilters = reactive({
    projectUuid: '',
    namespace: '',
    enabled: undefined as boolean | undefined,
    keyword: ''
  })

  const form = reactive<CreateLogAlertRuleRequest>({
    clusterUuid: '',
    workspaceId: 0,
    projectUuid: '',
    namespace: '',
    application: '',
    resourceName: '',
    name: '',
    description: '',
    queryText: '',
    conditionType: 'count_gt',
    threshold: '1',
    window: '5m',
    severity: 'warning',
    notifyChannels: '[]',
    enabled: true,
    evalInterval: '60s',
    forDuration: '0s',
    silenceWindow: '15m'
  })

  const rules = reactive<FormRules>({
    name: [{ required: true, message: '请输入规则名称', trigger: 'blur' }],
    queryText: [
      {
        validator: (_rule, value, callback) => {
          if (isQueryTextRequired.value && !String(value || '').trim()) {
            callback(new Error('当前条件类型必须填写查询内容'))
            return
          }
          callback()
        },
        trigger: 'blur'
      }
    ],
    conditionType: [{ required: true, message: '请选择条件类型', trigger: 'change' }],
    threshold: [{ required: true, message: '请输入阈值', trigger: 'blur' }],
    window: [{ required: true, message: '请输入时间窗口', trigger: 'blur' }],
    severity: [{ required: true, message: '请选择级别', trigger: 'change' }],
    evalInterval: [{ validator: validateEvalInterval, trigger: 'blur' }],
    forDuration: [{ validator: validateForDuration, trigger: 'blur' }],
    silenceWindow: [{ validator: validateSilenceWindow, trigger: 'blur' }]
  })

  const isSuperAdminUser = computed(() =>
    Array.isArray(userStore.info?.roles)
      ? userStore.info.roles.some((item) => String(item).toLowerCase() === 'super_admin')
      : false
  )

  const scopeOptions = computed(() => {
    const base = [
      {
        value: 'namespace' as ScopeMode,
        label: '命名空间规则',
        description: '限定在当前集群的某个工作空间 / namespace 内生效'
      },
      {
        value: 'service' as ScopeMode,
        label: '服务规则',
        description: '限定到某个服务，版本可选；不选版本则覆盖该服务全部版本'
      }
    ]
    if (!isSuperAdminUser.value) {
      return base
    }
    return [
      {
        value: 'cluster' as ScopeMode,
        label: '全局规则',
        description: '仅 SUPER_ADMIN 可用，对当前集群所有命名空间生效'
      },
      ...base
    ]
  })

  const healthStatusText = computed(() => {
    const mapping: Record<string, string> = {
      healthy: '健康',
      degraded: '异常',
      syncing: '同步中',
      unknown: '未知'
    }
    return mapping[syncStats.healthStatus] || syncStats.healthStatus || '-'
  })

  const currentClusterName = computed(() => {
    const current = clusters.value.find((item) => item.id === selectedClusterId.value)
    return current?.name || '未选择集群'
  })

  const selectedProject = computed(() => {
    return projectOptions.value.find((item) => item.id === scopeProjectId.value) || null
  })

  const selectedWorkspace = computed(() => {
    return workspaceOptions.value.find((item) => item.id === form.workspaceId) || null
  })

  const selectedApplication = computed(() => {
    return applicationOptions.value.find((item) => item.id === selectedApplicationId.value) || null
  })

  const selectedVersion = computed(() => {
    return versionOptions.value.find((item) => item.id === selectedVersionId.value) || null
  })

  const filteredTemplates = computed(() => {
    const keyword = templateKeyword.value.trim().toLowerCase()
    if (!keyword) return templates
    return templates.filter((item) => {
      return [
        item.label,
        item.description,
        item.category,
        item.audience,
        item.payload.queryText,
        item.payload.conditionType
      ].some((text) => String(text).toLowerCase().includes(keyword))
    })
  })

  const templatePreview = computed(() => {
    const currentKey = templatePreviewKey.value || filteredTemplates.value[0]?.key
    return filteredTemplates.value.find((item) => item.key === currentKey) || null
  })

  const selectedTemplate = computed(() => {
    return templates.find((item) => item.key === appliedTemplateKey.value) || null
  })

  const scopeSummary = computed(() => {
    if (scopeMode.value === 'cluster') {
      return `当前集群 ${currentClusterName.value}`
    }
    if (!selectedWorkspace.value) {
      return '尚未选择工作空间'
    }
    if (scopeMode.value === 'namespace') {
      return `命名空间 ${selectedWorkspace.value.namespace}`
    }
    if (!selectedApplication.value) {
      return `命名空间 ${selectedWorkspace.value.namespace} 下的服务`
    }
    if (selectedVersion.value) {
      return `服务 ${selectedApplication.value.nameEn} 的版本 ${selectedVersion.value.version}`
    }
    return `服务 ${selectedApplication.value.nameEn} 的全部版本`
  })

  const scopeHelperText = computed(() => {
    if (scopeMode.value === 'cluster') {
      return '该规则会覆盖当前集群所有命名空间，仅适合平台级通用异常和基础设施异常。'
    }
    if (scopeMode.value === 'namespace') {
      return selectedWorkspace.value
        ? `规则会限定在 ${selectedWorkspace.value.namespace} 命名空间内，不会影响其它工作空间。`
        : '先选择项目和工作空间，系统会自动带出 namespace。'
    }
    if (!selectedWorkspace.value) {
      return '先选择项目和工作空间，再选择需要管控的服务。'
    }
    if (!selectedApplication.value) {
      return `当前会限定在 ${selectedWorkspace.value.namespace} 命名空间内，接下来请选择具体服务。`
    }
    if (selectedVersion.value) {
      return `当前仅针对 ${selectedApplication.value.nameEn} 的版本 ${selectedVersion.value.version} 生效。`
    }
    return `当前会覆盖 ${selectedApplication.value.nameEn} 在 ${selectedWorkspace.value.namespace} 中的全部版本。`
  })

  const conditionHint = computed(() => {
    const threshold = form.threshold || '0'
    const queryText = shortText(form.queryText || '当前查询条件', 42)
    switch (form.conditionType) {
      case 'count_gt':
        return `将在 ${scopeSummary.value} 内，按 ${form.window} 窗口统计“${queryText}”的命中数；当命中数大于 ${threshold} 时触发。`
      case 'count_gte':
      case 'match_any':
        return `将在 ${scopeSummary.value} 内，按 ${form.window} 窗口统计“${queryText}”的命中数；当命中数大于等于 ${threshold} 时触发。`
      case 'count_lt':
        return `将在 ${scopeSummary.value} 内，按 ${form.window} 窗口统计“${queryText}”的命中数；当命中数小于 ${threshold} 时触发。`
      case 'count_lte':
        return `将在 ${scopeSummary.value} 内，按 ${form.window} 窗口统计“${queryText}”的命中数；当命中数小于等于 ${threshold} 时触发。`
      case 'no_data':
        return form.queryText.trim()
          ? `将在 ${scopeSummary.value} 内检查“${queryText}”在 ${form.window} 窗口内是否完全没有出现；若命中数为 0，则触发告警。`
          : `将在 ${scopeSummary.value} 内检查 ${form.window} 窗口内是否完全没有任何日志；若命中数为 0，则触发告警。`
      case 'flatline':
        return `将在 ${scopeSummary.value} 内检查 ${form.window} 窗口内是否完全没有业务日志输出；若命中数归零，则触发日志停滞告警。`
      default:
        return '系统会根据命中日志数、时间窗口和阈值自动判定是否触发。'
    }
  })

  const isQueryTextRequired = computed(() => !['no_data', 'flatline'].includes(form.conditionType))

  const isFlatlineCondition = computed(() => form.conditionType === 'flatline')

  const queryTextFieldLabel = computed(() => {
    return isQueryTextRequired.value ? '查询内容' : '查询内容（可选）'
  })

  const queryTextPlaceholder = computed(() => {
    if (form.conditionType === 'flatline') {
      return '日志停滞默认检测当前范围内是否完全没有业务日志输出，无需填写。'
    }
    if (form.conditionType === 'no_data') {
      return '可选。填写后表示在窗口内没有查询到这类日志就报警；不填则表示范围内完全没有日志。'
    }
    return '输入关键字、布尔表达式或错误特征，例如 ERROR OR Exception'
  })

  const queryTextTip = computed(() => {
    if (form.conditionType === 'flatline') {
      return '日志停滞关注的是“当前范围没有任何业务日志输出”，所以这里默认留空并禁用。'
    }
    if (form.conditionType === 'no_data') {
      return '如果你想监控某类日志是否消失，可以填写关键字；如果你只关心整个范围没有日志，可以保持为空。'
    }
    return '建议填写最能代表异常的关键字或表达式，避免把范围放得过宽。'
  })

  const thresholdHint = computed(() => {
    if (form.conditionType === 'no_data' || form.conditionType === 'flatline') {
      return '这两类条件固定按 0 命中处理，所以阈值会自动锁定为 0。'
    }
    return '阈值建议结合业务量级设置；太小容易误报，太大又可能漏报。'
  })

  const conditionExamples = computed(() => {
    if (form.conditionType === 'no_data') {
      return [
        {
          title: '有查询内容',
          description: '例如填写“支付成功”，表示 5 分钟内没有出现支付成功日志就报警。'
        },
        {
          title: '无查询内容',
          description: '表示当前范围在 5 分钟内完全没有日志输出就报警。'
        }
      ]
    }
    if (form.conditionType === 'flatline') {
      return [
        {
          title: '业务停摆检测',
          description: '适合消费者、调度器、对账、同步任务等“正常应该持续产生日志”的服务。'
        },
        {
          title: '默认语义',
          description: '不关心某一类日志是否缺失，而是关心当前范围有没有任何业务日志输出。'
        }
      ]
    }
    return []
  })

  const strategyHint = computed(() => {
    const interval = form.evalInterval || '60s'
    const forDuration = form.forDuration || '0s'
    const silenceWindow = form.silenceWindow || '15m'
    const forText =
      forDuration === '0s' || forDuration === '0'
        ? '命中后立即进入告警判断'
        : `需要连续满足条件达到 ${forDuration} 才会真正告警`
    return `平台会按 ${interval} 周期扫描该规则；${forText}；单次触发后会静默 ${silenceWindow}，避免短时间重复通知。`
  })

  const testSummary = computed(() => {
    if (!testResult.value) return ''
    return testResult.value.triggered
      ? `本次在 ${scopeSummary.value} 内命中 ${testResult.value.count} 条日志，当前会触发该规则。`
      : `本次在 ${scopeSummary.value} 内命中 ${testResult.value.count} 条日志，当前不会触发该规则。`
  })

  const displayRules = computed(() => {
    return rulesData.value.filter((item) => {
      if (appliedFilters.projectUuid && item.projectUuid !== appliedFilters.projectUuid)
        return false
      if (appliedFilters.namespace && !item.namespace.includes(appliedFilters.namespace))
        return false
      if (typeof appliedFilters.enabled === 'boolean' && item.enabled !== appliedFilters.enabled)
        return false
      if (
        appliedFilters.keyword &&
        !item.name.toLowerCase().includes(appliedFilters.keyword.toLowerCase())
      )
        return false
      return true
    })
  })
  const ruleSearchItems = computed(() => [
    {
      label: '项目',
      key: 'projectUuid',
      type: 'select',
      props: {
        placeholder: '全部项目',
        clearable: true,
        filterable: true,
        options: projectOptions.value.map((item) => ({
          label: item.name,
          value: item.uuid
        }))
      }
    },
    {
      label: '命名空间',
      key: 'namespace',
      type: 'input',
      props: {
        placeholder: '按 namespace 过滤',
        clearable: true
      }
    },
    {
      label: '状态',
      key: 'enabled',
      type: 'select',
      props: {
        placeholder: '全部',
        clearable: true,
        options: [
          { label: '已启用', value: true },
          { label: '已禁用', value: false }
        ]
      }
    },
    {
      label: '关键字',
      key: 'keyword',
      type: 'input',
      props: {
        placeholder: '按名称过滤',
        clearable: true
      }
    }
  ])

  async function refreshAll() {
    if (!selectedClusterUuid.value) return
    loading.value = true
    try {
      const [listResp, syncResp, configResp] = await Promise.all([
        listLogAlertRulesApi({
          clusterUuid: selectedClusterUuid.value,
          page: 1,
          pageSize: 200
        }),
        getLogClusterSyncStatusApi(selectedClusterUuid.value),
        getLogClusterConfigApi(selectedClusterUuid.value)
      ])
      rulesData.value = listResp.data || []
      Object.assign(syncStats, syncResp)
      clusterDefaultBackend.value = configResp.defaultBackend || ''
    } finally {
      loading.value = false
    }
  }

  async function loadProjects() {
    const resp = await searchProjectApi({ page: 1, pageSize: 200 })
    projectOptions.value = resp.items || []
  }

  async function resolveProjectCluster(projectId?: number) {
    if (!projectId || !selectedClusterUuid.value) return null
    const resp = await searchProjectClusterApi({ projectId })
    const items = (resp || []) as ProjectCluster[]
    return items.find((item) => item.clusterUuid === selectedClusterUuid.value) || null
  }

  async function loadWorkspaceOptions(projectClusterId?: number) {
    workspaceOptions.value = []
    if (!projectClusterId) return
    workspaceLoading.value = true
    try {
      const resp = await searchProjectWorkspaceApi({ projectClusterId })
      const items = (resp || []) as ProjectWorkspace[]
      workspaceOptions.value = items.map((item) => ({
        id: item.id,
        label: `${item.name} (${item.namespace})`,
        namespace: item.namespace,
        name: item.name
      }))
    } finally {
      workspaceLoading.value = false
    }
  }

  async function loadApplicationOptions(workspaceId?: number) {
    applicationOptions.value = []
    if (!workspaceId) return
    applicationLoading.value = true
    try {
      const resp = await searchApplicationApi({ workspaceId })
      applicationOptions.value = (resp || []) as OnecProjectApplication[]
    } finally {
      applicationLoading.value = false
    }
  }

  async function loadVersionOptions(applicationId?: number) {
    versionOptions.value = []
    if (!applicationId) return
    versionLoading.value = true
    try {
      const resp = await searchVersionApi({ applicationId })
      versionOptions.value = (resp || []) as OnecProjectVersion[]
    } finally {
      versionLoading.value = false
    }
  }

  function syncFormScope() {
    form.clusterUuid = selectedClusterUuid.value || ''
    if (scopeMode.value === 'cluster') {
      form.workspaceId = 0
      form.projectUuid = ''
      form.namespace = ''
      form.application = ''
      form.resourceName = ''
      return
    }
    form.projectUuid = selectedProject.value?.uuid || ''
    form.workspaceId = selectedWorkspace.value?.id || 0
    form.namespace = selectedWorkspace.value?.namespace || ''
    if (scopeMode.value === 'service') {
      form.application = selectedApplication.value?.nameEn || ''
      form.resourceName = selectedVersion.value?.resourceName || ''
      return
    }
    form.application = ''
    form.resourceName = ''
  }

  function resetScopeSelection(mode: ScopeMode = 'namespace') {
    scopeMode.value = mode
    scopeProjectId.value = undefined
    scopeProjectClusterId.value = undefined
    selectedApplicationId.value = undefined
    selectedVersionId.value = undefined
    workspaceOptions.value = []
    applicationOptions.value = []
    versionOptions.value = []
    form.workspaceId = 0
    form.projectUuid = ''
    form.namespace = ''
    form.application = ''
    form.resourceName = ''
  }

  function handleClusterChange(value: number | null) {
    setCluster(value)
    resetForm()
    refreshAll()
  }

  function applyFilters() {
    appliedFilters.projectUuid = filters.projectUuid
    appliedFilters.namespace = filters.namespace
    appliedFilters.enabled = filters.enabled
    appliedFilters.keyword = filters.keyword
  }

  function clearFilters() {
    filters.projectUuid = ''
    filters.namespace = ''
    filters.enabled = undefined
    filters.keyword = ''
    appliedFilters.projectUuid = ''
    appliedFilters.namespace = ''
    appliedFilters.enabled = undefined
    appliedFilters.keyword = ''
  }

  function openCreateEntry() {
    if (!selectedClusterUuid.value) {
      ElMessage.warning('请先选择集群')
      return
    }
    createEntryVisible.value = true
  }

  function openTemplateChooser() {
    createEntryVisible.value = false
    templateKeyword.value = ''
    if (!templatePreviewKey.value && templates[0]) {
      templatePreviewKey.value = templates[0].key
    }
    templateChooserVisible.value = true
  }

  async function selectScopeMode(mode: ScopeMode) {
    if (scopeMode.value === mode) return
    scopeMode.value = mode
    await handleScopeModeChange()
  }

  function openCreate(keepTemplate = false) {
    if (!selectedClusterUuid.value) {
      ElMessage.warning('请先选择集群')
      return
    }
    if (!keepTemplate) {
      appliedTemplateKey.value = ''
      templatePreviewKey.value = ''
    }
    createEntryVisible.value = false
    templateChooserVisible.value = false
    dialogMode.value = 'create'
    editingId.value = null
    resetForm()
    dialogVisible.value = true
  }

  function applyTemplate(template: (typeof templates)[number]) {
    appliedTemplateKey.value = template.key
    templatePreviewKey.value = template.key
    openCreate(true)
    form.name = template.payload.name
    form.description = template.payload.description
    form.queryText = template.payload.queryText
    form.conditionType = template.payload.conditionType
    form.threshold = template.payload.threshold
    form.window = template.payload.window
    form.severity = template.payload.severity
  }

  async function handleScopeProjectChange(projectId?: number) {
    scopeProjectId.value = projectId
    scopeProjectClusterId.value = undefined
    form.workspaceId = 0
    selectedApplicationId.value = undefined
    selectedVersionId.value = undefined
    workspaceOptions.value = []
    applicationOptions.value = []
    versionOptions.value = []
    if (!projectId) {
      syncFormScope()
      return
    }
    scopeClusterLoading.value = true
    try {
      const projectCluster = await resolveProjectCluster(projectId)
      if (!projectCluster) {
        ElMessage.warning('当前项目未绑定所选集群，请切换项目或集群')
        syncFormScope()
        return
      }
      scopeProjectClusterId.value = projectCluster.id
      await loadWorkspaceOptions(projectCluster.id)
    } finally {
      scopeClusterLoading.value = false
    }
    syncFormScope()
  }

  async function handleScopeWorkspaceChange(workspaceId?: number) {
    form.workspaceId = workspaceId || 0
    selectedApplicationId.value = undefined
    selectedVersionId.value = undefined
    applicationOptions.value = []
    versionOptions.value = []
    if (scopeMode.value === 'service' && workspaceId) {
      await loadApplicationOptions(workspaceId)
    }
    syncFormScope()
  }

  async function handleScopeApplicationChange(applicationId?: number) {
    selectedApplicationId.value = applicationId
    selectedVersionId.value = undefined
    versionOptions.value = []
    if (applicationId) {
      await loadVersionOptions(applicationId)
    }
    syncFormScope()
  }

  function handleScopeVersionChange(versionId?: number) {
    selectedVersionId.value = versionId
    syncFormScope()
  }

  async function handleScopeModeChange() {
    if (scopeMode.value === 'cluster') {
      resetScopeSelection('cluster')
    } else if (scopeMode.value === 'namespace') {
      selectedApplicationId.value = undefined
      selectedVersionId.value = undefined
      applicationOptions.value = []
      versionOptions.value = []
      form.application = ''
      form.resourceName = ''
    } else if (scopeMode.value === 'service' && (form.workspaceId || 0) > 0) {
      await loadApplicationOptions(form.workspaceId || 0)
    }
    syncFormScope()
  }

  function validateScopeSelection() {
    syncFormScope()
    if (scopeMode.value === 'cluster') {
      if (!isSuperAdminUser.value) {
        ElMessage.warning('只有 SUPER_ADMIN 可以创建全局规则')
        return false
      }
      return true
    }
    if (!scopeProjectId.value) {
      ElMessage.warning('请选择项目')
      return false
    }
    if (!scopeProjectClusterId.value) {
      ElMessage.warning('当前项目未绑定所选集群')
      return false
    }
    if (!form.workspaceId) {
      ElMessage.warning('请选择工作空间')
      return false
    }
    if (scopeMode.value === 'service' && !selectedApplicationId.value) {
      ElMessage.warning('请选择服务')
      return false
    }
    return true
  }

  async function openEdit(row: LogAlertRule) {
    dialogMode.value = 'edit'
    editingId.value = row.id
    appliedTemplateKey.value = ''
    templatePreviewKey.value = ''
    resetForm()
    form.name = row.name
    form.description = row.description
    form.queryText = row.conditionType === 'flatline' ? '' : row.queryText
    form.conditionType = row.conditionType
    form.threshold = row.threshold
    form.window = row.window
    form.severity = row.severity
    form.notifyChannels = row.notifyChannels || '[]'
    form.enabled = row.enabled
    form.evalInterval = row.evalInterval || '60s'
    form.forDuration = row.forDuration || '0s'
    form.silenceWindow = row.silenceWindow || '15m'
    form.clusterUuid = row.clusterUuid

    if (!row.projectUuid && !row.namespace) {
      resetScopeSelection('cluster')
      syncFormScope()
      dialogVisible.value = true
      return
    }

    scopeMode.value = row.application ? 'service' : 'namespace'
    const project = projectOptions.value.find((item) => item.uuid === row.projectUuid)
    if (project) {
      await handleScopeProjectChange(project.id)
    }
    if (row.workspaceId > 0) {
      await handleScopeWorkspaceChange(Number(row.workspaceId))
    }
    if (row.application && row.workspaceId > 0) {
      const app = applicationOptions.value.find((item) => item.nameEn === row.application)
      if (app) {
        await handleScopeApplicationChange(app.id)
        const version = versionOptions.value.find((item) => item.resourceName === row.resourceName)
        if (version) {
          handleScopeVersionChange(version.id)
        }
      }
    }
    syncFormScope()
    dialogVisible.value = true
  }

  async function submitForm() {
    if (!validateScopeSelection()) return
    await formRef.value?.validate()
    submitting.value = true
    try {
      const payload = {
        ...form,
        queryText: isFlatlineCondition.value ? '' : form.queryText,
        threshold:
          form.conditionType === 'no_data' || form.conditionType === 'flatline'
            ? form.threshold || '0'
            : form.threshold
      }
      if (dialogMode.value === 'create') {
        await createLogAlertRuleApi(payload)
        ElMessage.success('创建成功')
      } else if (editingId.value) {
        await updateLogAlertRuleApi(editingId.value, payload)
        ElMessage.success('更新成功')
      }
      dialogVisible.value = false
      await refreshAll()
    } finally {
      submitting.value = false
    }
  }

  async function testRule() {
    if (!selectedClusterUuid.value) {
      ElMessage.warning('请先选择集群')
      return
    }
    if (!validateScopeSelection()) return
    await formRef.value?.validate()
    testing.value = true
    try {
      const end = Date.now()
      const start = dayjs(end).subtract(parseWindow(form.window), 'millisecond').valueOf()
      const resp = await queryLogsApi({
        sourceType: 'container',
        logType: 'container',
        clusterUuid: form.clusterUuid || selectedClusterUuid.value,
        projectUuid: form.projectUuid || undefined,
        namespace: form.namespace || undefined,
        application: form.application || undefined,
        resourceName: form.resourceName || undefined,
        queryMode: resolveQueryMode(clusterDefaultBackend.value),
        mode: 'form',
        queryText: isFlatlineCondition.value ? '' : form.queryText,
        start: String(start),
        end: String(end),
        limit: 20,
        direction: 'backward'
      })
      const count = resp.list.length
      const triggered = shouldTriggerByCondition(form.conditionType, form.threshold, count)
      testResult.value = {
        count,
        triggered,
        backendType: resp.backendType || '-',
        queryMode: resp.queryMode || 'es',
        evaluatedAt: Date.now(),
        samples: (resp.list || []).slice(0, 5).map((item) => ({
          timestamp: item.timestamp,
          message: item.message
        }))
      }
      ElMessage.success(
        triggered
          ? `测试完成，当前规则会触发，命中 ${count} 条日志`
          : `测试完成，当前规则不会触发，命中 ${count} 条日志`
      )
    } finally {
      testing.value = false
    }
  }

  async function toggleRule(row: LogAlertRule) {
    if (row.enabled) {
      await disableLogAlertRuleApi(row.id)
      ElMessage.success('已禁用')
    } else {
      await enableLogAlertRuleApi(row.id)
      ElMessage.success('已启用')
    }
    await refreshAll()
  }

  async function removeRule(id: number) {
    await deleteLogAlertRuleApi(id)
    ElMessage.success('删除成功')
    await refreshAll()
  }

  function resetForm() {
    form.clusterUuid = selectedClusterUuid.value || ''
    form.workspaceId = 0
    form.projectUuid = ''
    form.namespace = ''
    form.application = ''
    form.resourceName = ''
    form.name = ''
    form.description = ''
    form.queryText = ''
    form.conditionType = 'count_gt'
    form.threshold = '1'
    form.window = '5m'
    form.severity = 'warning'
    form.notifyChannels = '[]'
    form.enabled = true
    form.evalInterval = '60s'
    form.forDuration = '0s'
    form.silenceWindow = '15m'
    testResult.value = null
    resetScopeSelection('namespace')
  }

  function formatTime(value?: number) {
    return value ? dayjs(value * 1000).format('YYYY-MM-DD HH:mm:ss') : '-'
  }

  function severityTagType(value: string) {
    if (value === 'critical') return 'danger'
    if (value === 'warning') return 'warning'
    return 'info'
  }

  function severityText(value: string) {
    return severityOptions.find((item) => item.value === value)?.label || value || '-'
  }

  function backendTypeText(value?: string) {
    if (!value) return '-'
    if (value === 'elasticsearch' || value === 'es') return 'Elasticsearch'
    if (value === 'loki') return 'Loki'
    return value
  }

  function conditionTypeText(value: string) {
    return conditionOptions.find((item) => item.value === value)?.label || value || '-'
  }

  function syncStatusTagType(value: string) {
    if (value === 'success') return 'success'
    if (value === 'failed') return 'danger'
    if (value === 'pending') return 'warning'
    if (value === 'disabled') return 'info'
    return undefined
  }

  function parseWindow(value: string) {
    const match = String(value || '')
      .trim()
      .match(/^(\d+)([smhd])$/)
    if (!match) return 5 * 60 * 1000
    const amount = Number(match[1])
    const unit = match[2]
    const unitMap: Record<string, number> = {
      s: 1000,
      m: 60 * 1000,
      h: 60 * 60 * 1000,
      d: 24 * 60 * 60 * 1000
    }
    return amount * (unitMap[unit] || 60 * 1000)
  }

  function resolveQueryMode(value?: string) {
    if (value === 'loki') return 'loki'
    if (value === 'elasticsearch' || value === 'es') return 'es'
    return 'platform'
  }

  function shouldTriggerByCondition(conditionType: string, threshold: string, count: number) {
    const thresholdValue = Number(threshold || 0)
    switch (conditionType) {
      case 'count_gt':
        return count > thresholdValue
      case 'count_gte':
      case 'match_any':
        return count >= thresholdValue
      case 'count_lt':
        return count < thresholdValue
      case 'count_lte':
        return count <= thresholdValue
      case 'no_data':
      case 'flatline':
        return count === 0
      default:
        return count > 0
    }
  }

  function ruleScopeTypeText(row: LogAlertRule) {
    if (!row.projectUuid && !row.namespace) return '全局'
    if (row.application && row.resourceName) return '服务版本'
    if (row.application) return '服务'
    return '命名空间'
  }

  function ruleScopeTargetText(row: LogAlertRule) {
    if (!row.projectUuid && !row.namespace) return currentClusterName.value || row.clusterUuid
    if (row.application && row.resourceName) {
      return `${row.namespace} / ${row.application} / ${row.resourceName}`
    }
    if (row.application) {
      return `${row.namespace} / ${row.application}`
    }
    return row.namespace || '-'
  }

  function shortText(value: string, size = 32) {
    const trimmed = String(value || '').trim()
    if (trimmed.length <= size) return trimmed
    return `${trimmed.slice(0, size)}...`
  }

  function isValidDuration(value: string, allowZero = false) {
    const match = String(value || '')
      .trim()
      .match(/^(\d+)([smhd])$/)
    if (!match) return false
    if (!allowZero && Number(match[1]) <= 0) return false
    return allowZero ? Number(match[1]) >= 0 : true
  }

  function validateEvalInterval(_rule: unknown, value: string, callback: (error?: Error) => void) {
    if (!isValidDuration(value, false)) {
      callback(new Error('请使用类似 30s、5m、1h 的时间格式'))
      return
    }
    callback()
  }

  function validateForDuration(_rule: unknown, value: string, callback: (error?: Error) => void) {
    if (!isValidDuration(value, true)) {
      callback(new Error('请使用类似 0s、30s、5m、1h 的时间格式'))
      return
    }
    callback()
  }

  function validateSilenceWindow(_rule: unknown, value: string, callback: (error?: Error) => void) {
    if (!isValidDuration(value, true)) {
      callback(new Error('请使用类似 0s、15m、1h 的时间格式'))
      return
    }
    callback()
  }

  watch(
    () => form.conditionType,
    (value) => {
      if (value === 'no_data' || value === 'flatline') {
        form.threshold = '0'
      }
      if (value === 'flatline') {
        form.queryText = ''
      }
      testResult.value = null
    }
  )

  watch(
    () => [
      form.queryText,
      form.window,
      form.severity,
      form.threshold,
      form.evalInterval,
      form.forDuration,
      form.silenceWindow,
      scopeMode.value,
      form.projectUuid,
      form.namespace,
      form.application,
      form.resourceName
    ],
    () => {
      testResult.value = null
    }
  )

  watch(
    () => templateKeyword.value,
    () => {
      const first = filteredTemplates.value[0]
      if (!first) {
        templatePreviewKey.value = ''
        return
      }
      if (!filteredTemplates.value.some((item) => item.key === templatePreviewKey.value)) {
        templatePreviewKey.value = first.key
      }
    },
    { immediate: true }
  )

  onMounted(async () => {
    await Promise.all([loadClusters(), loadProjects()])
    applyFilters()
    if (selectedClusterUuid.value) {
      await refreshAll()
    }
  })
</script>

<style scoped>
  .log-alert-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
    min-height: 100%;
    padding: 0;
    background: var(--el-fill-color-extra-light);
  }

  .page-header {
    display: flex;
    flex-direction: column;
    gap: 10px;
    align-items: flex-start;
    padding: 8px 4px 2px;
  }

  .eyebrow {
    margin-bottom: 4px;
    font-size: 12px;
    color: #64748b;
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }

  .page-header h1 {
    margin: 0;
    font-size: 34px;
    line-height: 1.2;
    color: #0f172a;
  }

  .page-header p {
    max-width: 860px;
    margin: 8px 0 0;
    font-size: 17px;
    line-height: 1.7;
    color: #64748b;
  }

  .status-group {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    margin-top: 10px;
  }

  .status-chip {
    display: inline-flex;
    align-items: center;
    padding: 6px 12px;
    font-size: 12px;
    font-weight: 600;
    color: #2563eb;
    background: #eef6ff;
    backdrop-filter: blur(10px);
    border-radius: 999px;
  }

  .status-chip.muted {
    color: #64748b;
    background: #f8fafc;
  }

  .header-actions {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .cluster-select {
    width: 280px;
  }

  .command-card {
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    box-shadow: 0 1px 2px rgb(15 23 42 / 3%);
  }

  .command-card :deep(.el-card__body) {
    padding: 16px;
  }

  .command-grid {
    display: flex;
    gap: 16px;
    align-items: center;
    justify-content: space-between;
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

  .command-actions {
    display: flex;
    flex: none;
    flex-wrap: wrap;
    gap: 10px;
    align-items: center;
    justify-content: flex-end;
  }

  .command-actions :deep(.el-button + .el-button),
  .row-actions :deep(.el-button + .el-button) {
    margin-left: 0;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(6, minmax(0, 1fr));
    gap: 12px;
  }

  .stat-card {
    min-width: 0;
    padding: 14px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    box-shadow: 0 1px 2px rgb(15 23 42 / 3%);
  }

  .stat-card .label {
    overflow: hidden;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .stat-card .value {
    margin-top: 8px;
    overflow: hidden;
    font-size: 24px;
    font-weight: 700;
    line-height: 1;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .stat-card .value.success {
    color: #67c23a;
  }

  .stat-card .value.danger {
    color: #f56c6c;
  }

  .stat-card .value.warning {
    color: #e6a23c;
  }

  .filter-card,
  .table-card {
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    box-shadow: 0 1px 2px rgb(15 23 42 / 3%);
  }

  .filter-card {
    background: var(--el-bg-color);
  }

  .template-card {
    border-radius: 12px;
  }

  .card-header {
    font-weight: 600;
  }

  .card-subtitle {
    margin-top: 4px;
    font-size: 12px;
    color: #909399;
  }

  .table-card-header {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
  }

  .form-section-subtitle {
    margin: -4px 0 14px;
    font-size: 13px;
    line-height: 1.6;
    color: #606266;
  }

  .scope-table-cell {
    display: flex;
    gap: 10px;
    align-items: center;
    line-height: 1.5;
    color: #606266;
  }

  .scope-option-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 12px;
    margin-bottom: 16px;
  }

  .scope-option-card {
    display: flex;
    flex-direction: column;
    gap: 8px;
    width: 100%;
    padding: 14px 16px;
    text-align: left;
    cursor: pointer;
    background: linear-gradient(180deg, #fff 0%, #f8fbff 100%);
    border: 1px solid #dcdfe6;
    border-radius: 14px;
    transition:
      border-color 0.2s ease,
      box-shadow 0.2s ease,
      transform 0.2s ease;
  }

  .scope-option-card:hover {
    border-color: #409eff;
    box-shadow: 0 10px 24px rgb(64 158 255 / 12%);
    transform: translateY(-1px);
  }

  .scope-option-card.active {
    background: linear-gradient(180deg, #eef6ff 0%, #fff 100%);
    border-color: #409eff;
    box-shadow: 0 12px 24px rgb(64 158 255 / 14%);
  }

  .scope-option-title {
    font-size: 15px;
    font-weight: 700;
    color: #303133;
  }

  .scope-option-desc {
    font-size: 12px;
    line-height: 1.6;
    color: #606266;
  }

  .dialog-heading {
    margin-bottom: 18px;
  }

  .dialog-heading.compact {
    margin-bottom: 14px;
  }

  .dialog-heading-title {
    font-size: 16px;
    font-weight: 700;
    color: #303133;
  }

  .dialog-heading-desc {
    margin-top: 6px;
    font-size: 13px;
    line-height: 1.6;
    color: #606266;
  }

  .create-entry-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
  }

  .entry-card {
    padding: 20px;
    cursor: pointer;
    background: linear-gradient(180deg, #fff 0%, #f8fbff 100%);
    border: 1px solid #e4e7ed;
    border-radius: 16px;
    transition:
      transform 0.2s ease,
      border-color 0.2s ease,
      box-shadow 0.2s ease;
  }

  .entry-card.primary {
    background: linear-gradient(180deg, #fff 0%, #eef6ff 100%);
    border-color: #c6e2ff;
  }

  .entry-card:hover {
    border-color: #409eff;
    box-shadow: 0 12px 28px rgb(64 158 255 / 14%);
    transform: translateY(-1px);
  }

  .entry-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    font-size: 13px;
    font-weight: 700;
    color: #2563eb;
    background: #eff6ff;
    border-radius: 10px;
  }

  .entry-title {
    margin-top: 16px;
    font-size: 18px;
    font-weight: 700;
    color: #303133;
  }

  .entry-desc {
    min-height: 44px;
    margin-top: 10px;
    line-height: 1.6;
    color: #606266;
  }

  .entry-meta {
    margin-top: 14px;
    font-size: 12px;
    color: #909399;
  }

  .template-toolbar {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 14px;
  }

  .template-layout {
    display: grid;
    grid-template-columns: minmax(0, 1.1fr) minmax(300px, 0.9fr);
    gap: 16px;
    align-items: start;
  }

  .template-grid {
    display: grid;
    grid-template-columns: repeat(1, minmax(0, 1fr));
    gap: 10px;
  }

  .template-item {
    padding: 14px;
    cursor: pointer;
    background: linear-gradient(180deg, #fff 0%, #f8fbff 100%);
    border: 1px solid #e4e7ed;
    border-radius: 12px;
    transition:
      transform 0.2s ease,
      border-color 0.2s ease,
      box-shadow 0.2s ease;
  }

  .template-item.active {
    border-color: #409eff;
    box-shadow: 0 12px 24px rgb(64 158 255 / 12%);
  }

  .template-item:hover {
    border-color: #409eff;
    box-shadow: 0 10px 24px rgb(64 158 255 / 12%);
    transform: translateY(-1px);
  }

  .template-title-row {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    justify-content: space-between;
  }

  .template-title {
    font-weight: 600;
    color: #303133;
  }

  .template-desc {
    min-height: 40px;
    margin-top: 8px;
    font-size: 13px;
    line-height: 1.5;
    color: #606266;
  }

  .template-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-top: 10px;
    font-size: 12px;
    color: #909399;
  }

  .template-preview {
    position: sticky;
    top: 0;
    padding: 18px;
    background: linear-gradient(180deg, #fff 0%, #f7fbff 100%);
    border: 1px solid #dbeafe;
    border-radius: 16px;
  }

  .template-preview-header {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    justify-content: space-between;
  }

  .template-preview-title {
    font-size: 18px;
    font-weight: 700;
    color: #303133;
  }

  .template-preview-subtitle {
    margin-top: 8px;
    font-size: 13px;
    line-height: 1.6;
    color: #606266;
  }

  .template-preview-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
    margin-top: 16px;
  }

  .template-preview-item {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 12px;
    background: #fff;
    border: 1px solid #ebeef5;
    border-radius: 12px;
  }

  .template-preview-item span {
    font-size: 12px;
    color: #909399;
  }

  .preview-query-card {
    padding: 14px;
    margin-top: 16px;
    color: #e2e8f0;
    background: #0f172a;
    border-radius: 12px;
  }

  .preview-query-label {
    font-size: 12px;
    color: #93c5fd;
  }

  .preview-query-text {
    margin-top: 10px;
    font-family: var(--ax-font-mono);
    font-size: 12px;
    line-height: 1.7;
    word-break: break-word;
  }

  .preview-tip-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 16px;
  }

  .preview-tip-item {
    padding: 10px 12px;
    font-size: 13px;
    line-height: 1.6;
    color: #606266;
    background: #fff;
    border: 1px solid #ebeef5;
    border-radius: 10px;
  }

  .preview-actions {
    display: flex;
    gap: 12px;
    justify-content: flex-end;
    margin-top: 18px;
  }

  .filter-form {
    display: grid;
    grid-template-columns: repeat(5, minmax(180px, 1fr));
    gap: 10px 12px;
    align-items: end;
    margin-bottom: 0;
  }

  .filter-form :deep(.el-form-item__label) {
    font-weight: 600;
    color: #475569;
  }

  .filter-form :deep(.el-form-item) {
    margin-right: 0;
    margin-bottom: 0;
  }

  .filter-form :deep(.el-form-item__content) {
    width: 100%;
  }

  .filter-form :deep(.el-input),
  .filter-form :deep(.el-select) {
    width: 100% !important;
  }

  .template-source-banner {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    padding: 14px 16px;
    margin-bottom: 18px;
    background: linear-gradient(180deg, #eff6ff 0%, #fff 100%);
    border: 1px solid #dbeafe;
    border-radius: 12px;
  }

  .template-source-title {
    font-size: 14px;
    font-weight: 700;
    color: #1d4ed8;
  }

  .template-source-desc {
    margin-top: 4px;
    font-size: 13px;
    color: #606266;
  }

  .form-section-title {
    padding-left: 12px;
    margin-bottom: 14px;
    font-size: 15px;
    font-weight: 700;
    color: #303133;
    border-left: 4px solid #409eff;
  }

  .scope-summary-card {
    padding: 14px 16px;
    margin-bottom: 18px;
    background: linear-gradient(180deg, #f8fbff 0%, #fff 100%);
    border: 1px solid #dbeafe;
    border-radius: 12px;
  }

  .scope-summary-title {
    font-size: 14px;
    font-weight: 700;
    color: #1d4ed8;
  }

  .scope-summary-desc {
    margin-top: 6px;
    font-size: 13px;
    line-height: 1.7;
    color: #606266;
  }

  .definition-block {
    padding: 14px 16px 4px;
    margin-bottom: 14px;
    background: linear-gradient(180deg, #fff 0%, #f8fbff 100%);
    border: 1px solid #e4e7ed;
    border-radius: 14px;
  }

  .definition-block.subtle {
    background: #fbfcff;
    border-style: dashed;
  }

  .condition-helper {
    padding: 12px 14px;
    margin-bottom: 18px;
    background: #f8fafc;
    border: 1px solid #e5e7eb;
    border-radius: 12px;
  }

  .condition-helper.muted {
    background: #fcfcfd;
    border-style: dashed;
  }

  .condition-helper-title {
    font-size: 13px;
    font-weight: 700;
    color: #303133;
  }

  .condition-helper-text {
    margin-top: 6px;
    font-size: 13px;
    line-height: 1.6;
    color: #606266;
  }

  .field-tip {
    margin-top: -8px;
    margin-bottom: 10px;
    font-size: 12px;
    line-height: 1.6;
    color: #909399;
  }

  .strategy-table-cell {
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 12px;
    color: #606266;
  }

  .query-tip {
    margin-top: -10px;
    margin-bottom: 16px;
  }

  .condition-example-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
    margin: 0 0 16px;
  }

  .condition-example-card {
    padding: 12px 14px;
    background: linear-gradient(180deg, #fff 0%, #fbfcff 100%);
    border: 1px solid #e5e7eb;
    border-radius: 12px;
  }

  .condition-example-title {
    font-size: 13px;
    font-weight: 700;
    color: #303133;
  }

  .condition-example-desc {
    margin-top: 6px;
    font-size: 13px;
    line-height: 1.6;
    color: #606266;
  }

  .test-result-card {
    margin-top: 8px;
    background: linear-gradient(180deg, #fff 0%, #f8fbff 100%);
    border-radius: 12px;
  }

  .test-result-header {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
  }

  .test-result-title {
    font-weight: 600;
    color: #303133;
  }

  .test-result-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 12px;
    margin-top: 12px;
  }

  .test-result-item {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 12px;
    background: #fff;
    border: 1px solid #ebeef5;
    border-radius: 10px;
  }

  .test-label {
    font-size: 12px;
    color: #909399;
  }

  .test-summary {
    padding: 12px 14px;
    margin-top: 12px;
    font-size: 13px;
    line-height: 1.6;
    color: #606266;
    background: #fff;
    border: 1px solid #ebeef5;
    border-radius: 10px;
  }

  .test-sample-block {
    padding-top: 14px;
    margin-top: 14px;
    border-top: 1px dashed #dcdfe6;
  }

  .test-sample-title {
    margin-bottom: 10px;
    font-weight: 600;
    color: #303133;
  }

  .test-sample-empty {
    font-size: 13px;
    color: #909399;
  }

  .test-sample-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .test-sample-item {
    display: grid;
    grid-template-columns: 168px 1fr;
    gap: 12px;
    padding: 10px 12px;
    background: #fff;
    border: 1px solid #ebeef5;
    border-radius: 10px;
  }

  .sample-time {
    font-size: 12px;
    color: #909399;
  }

  .sample-message {
    font-size: 13px;
    line-height: 1.6;
    color: #303133;
    word-break: break-word;
  }

  @media (width <= 1440px) {
    .stats-grid {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }

    .template-layout,
    .scope-option-grid {
      grid-template-columns: repeat(1, minmax(0, 1fr));
    }

    .filter-form {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }

    .test-result-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .test-sample-item {
      grid-template-columns: 140px 1fr;
    }
  }

  @media (width <= 992px) {
    .page-header,
    .command-grid {
      flex-direction: column;
      align-items: stretch;
    }

    .header-actions,
    .command-actions {
      flex-wrap: wrap;
      justify-content: flex-start;
      width: 100%;
    }

    .cluster-select {
      width: 100%;
    }

    .stats-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .create-entry-grid {
      grid-template-columns: repeat(1, minmax(0, 1fr));
    }

    .test-result-grid {
      grid-template-columns: repeat(1, minmax(0, 1fr));
    }

    .test-sample-item {
      grid-template-columns: repeat(1, minmax(0, 1fr));
    }

    .table-card-header,
    .scope-table-cell,
    .template-source-banner,
    .template-preview-header,
    .template-toolbar,
    .preview-actions {
      flex-direction: column;
      align-items: flex-start;
    }

    .template-preview-grid,
    .condition-example-grid {
      grid-template-columns: repeat(1, minmax(0, 1fr));
    }

    .filter-form {
      grid-template-columns: 1fr;
    }
  }
</style>
