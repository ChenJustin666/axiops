<template>
  <div class="log-search-page">
    <section class="page-header">
      <div class="page-title-group">
        <div class="eyebrow">日志检索</div>
        <h1>统一检索容器日志与系统日志</h1>
        <p>默认展示核心条件，更多过滤项按需展开，整体更紧凑。</p>
      </div>

      <div class="status-group">
        <span class="status-chip" :class="sourceType">{{ sourceTypeLabel }}</span>
        <span class="status-chip muted">{{ isQuerying ? '查询中' : '待命' }}</span>
        <span class="status-chip muted">{{ autoRefreshText }}</span>
        <span class="status-chip muted">{{ currentClusterName || '未选择集群' }}</span>
      </div>
    </section>

    <section class="search-card">
      <div class="card-topbar">
        <div class="segment-group compact">
          <button
            v-for="item in sourceTypeOptions"
            :key="item.value"
            class="segment-btn mini"
            :class="{ active: sourceType === item.value }"
            @click="handleSourceTypeChange(item.value)"
          >
            {{ item.label }}
          </button>
        </div>

        <div class="segment-group compact">
          <button
            v-for="item in queryModeButtons"
            :key="item.key"
            class="segment-btn mini"
            :class="{ active: item.active, disabled: item.disabled }"
            :disabled="item.disabled"
            @click="item.onClick"
          >
            {{ item.label }}
          </button>
        </div>

        <div class="segment-group compact">
          <button
            v-for="item in editorModeButtons"
            :key="item.key"
            class="segment-btn mini"
            :class="{ active: item.active, disabled: item.disabled }"
            :disabled="item.disabled"
            @click="item.onClick"
          >
            {{ item.label }}
          </button>
        </div>
      </div>

      <ElForm label-position="top" size="small" class="search-form">
        <ElRow v-show="editorMode === 'form'" :gutter="8">
          <template v-if="sourceType === 'container'">
            <ElCol :xs="24" :sm="12" :md="6">
              <ElFormItem label="选择项目" required>
                <ElSelect
                  v-model="form.projectId"
                  placeholder="选择项目"
                  clearable
                  filterable
                  :loading="projectLoading"
                  @change="handleProjectChange"
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

            <ElCol :xs="24" :sm="12" :md="6">
              <ElFormItem label="选择集群" required>
                <ElSelect
                  v-model="form.clusterId"
                  placeholder="选择集群"
                  clearable
                  filterable
                  :disabled="!form.projectId"
                  :loading="clusterLoading"
                  @change="handleClusterChange"
                >
                  <ElOption
                    v-for="item in clusterOptions"
                    :key="item.id"
                    :label="item.name"
                    :value="item.id"
                  />
                </ElSelect>
              </ElFormItem>
            </ElCol>

            <ElCol :xs="24" :sm="12" :md="6">
              <ElFormItem label="选择工作空间" required>
                <ElSelect
                  v-model="form.workspaceId"
                  placeholder="选择工作空间"
                  clearable
                  filterable
                  :disabled="!form.clusterId"
                  :loading="workspaceLoading"
                  @change="handleWorkspaceChange"
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

            <ElCol :xs="24" :sm="12" :md="6">
              <ElFormItem label="选择应用">
                <ElSelect
                  v-model="form.applicationId"
                  placeholder="选择应用"
                  clearable
                  filterable
                  :disabled="!form.workspaceId"
                  :loading="applicationLoading"
                  @change="handleApplicationChange"
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

            <ElCol :xs="24" :sm="12" :md="5">
              <ElFormItem label="选择版本">
                <ElSelect
                  v-model="form.versionId"
                  placeholder="选择版本"
                  clearable
                  filterable
                  :disabled="!form.applicationId"
                  :loading="versionLoading"
                  @change="handleVersionChange"
                >
                  <ElOption
                    v-for="item in versionOptions"
                    :key="item.id"
                    :label="formatVersionLabel(item)"
                    :value="item.id"
                  />
                </ElSelect>
              </ElFormItem>
            </ElCol>

            <ElCol :xs="24" :sm="12" :md="8">
              <ElFormItem label="选择时间范围">
                <ElDatePicker
                  v-model="timeRange"
                  type="datetimerange"
                  format="YYYY-MM-DD HH:mm:ss"
                  start-placeholder="开始时间"
                  end-placeholder="结束时间"
                  range-separator="至"
                  value-format="x"
                  :shortcuts="timeShortcuts"
                  style="width: 100%"
                />
              </ElFormItem>
            </ElCol>

            <ElCol :xs="24" :sm="24" :md="11">
              <ElFormItem label="关键字">
                <ElInput
                  v-model="form.message"
                  placeholder="输入关键字（支持正则）"
                  clearable
                  @keyup.enter="handleSearch"
                />
              </ElFormItem>
            </ElCol>
          </template>

          <template v-else>
            <ElCol :xs="24" :sm="12" :md="6">
              <ElFormItem label="选择集群" required>
                <ElSelect
                  v-model="form.clusterId"
                  placeholder="选择集群"
                  clearable
                  filterable
                  :loading="clusterLoading"
                  @change="handleClusterChange"
                >
                  <ElOption
                    v-for="item in clusterOptions"
                    :key="item.id"
                    :label="item.name"
                    :value="item.id"
                  />
                </ElSelect>
              </ElFormItem>
            </ElCol>

            <ElCol :xs="24" :sm="12" :md="5">
              <ElFormItem label="选择节点" required>
                <ElSelect
                  v-model="form.host"
                  placeholder="选择节点"
                  clearable
                  filterable
                  allow-create
                  :disabled="!form.clusterId"
                  :loading="hostLoading"
                  @visible-change="handleHostDropdownVisible"
                >
                  <ElOption v-for="item in hostOptions" :key="item" :label="item" :value="item" />
                </ElSelect>
              </ElFormItem>
            </ElCol>

            <ElCol :xs="24" :sm="12" :md="7">
              <ElFormItem label="选择时间范围">
                <ElDatePicker
                  v-model="timeRange"
                  type="datetimerange"
                  format="YYYY-MM-DD HH:mm:ss"
                  start-placeholder="开始时间"
                  end-placeholder="结束时间"
                  range-separator="至"
                  value-format="x"
                  :shortcuts="timeShortcuts"
                  style="width: 100%"
                />
              </ElFormItem>
            </ElCol>

            <ElCol :xs="24" :sm="24" :md="12">
              <ElFormItem label="关键字">
                <ElInput
                  v-model="form.message"
                  placeholder="输入关键字检索日志正文"
                  clearable
                  @keyup.enter="handleSearch"
                />
              </ElFormItem>
            </ElCol>
          </template>
        </ElRow>

        <div v-show="editorMode === 'form'" class="more-toggle" @click="showMore = !showMore">
          <span>更多模式</span>
          <span class="more-meta">{{ showMore ? '收起' : '展开' }}</span>
        </div>

        <ElCollapseTransition>
          <div v-show="editorMode === 'form' && showMore" class="more-panel">
            <template v-if="sourceType === 'container'">
              <ElRow :gutter="8">
                <ElCol :xs="24" :sm="12" :md="4">
                  <ElFormItem label="选择 Pod">
                    <ElSelect
                      v-model="form.podName"
                      placeholder="选择或输入 Pod 名称"
                      clearable
                      filterable
                      allow-create
                      default-first-option
                      :loading="podNameLoading"
                      @visible-change="handlePodNameDropdownVisible"
                    >
                      <ElOption
                        v-for="item in podNameOptions"
                        :key="item"
                        :label="item"
                        :value="item"
                      />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>

                <ElCol :xs="24" :sm="12" :md="4">
                  <ElFormItem label="选择容器">
                    <ElSelect
                      v-model="form.containerName"
                      placeholder="选择或输入容器名称"
                      clearable
                      filterable
                      allow-create
                      default-first-option
                      :loading="containerNameLoading"
                      @visible-change="handleContainerNameDropdownVisible"
                    >
                      <ElOption
                        v-for="item in containerNameOptions"
                        :key="item"
                        :label="item"
                        :value="item"
                      />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>

                <ElCol :xs="24" :sm="12" :md="4">
                  <ElFormItem label="选择 Pod IP">
                    <ElSelect
                      v-model="form.podIp"
                      placeholder="选择或输入 Pod IP"
                      clearable
                      filterable
                      allow-create
                      default-first-option
                      :loading="podIpLoading"
                      @visible-change="handlePodIpDropdownVisible"
                    >
                      <ElOption
                        v-for="item in podIpOptions"
                        :key="item"
                        :label="item"
                        :value="item"
                      />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>

                <ElCol :xs="24" :sm="12" :md="4">
                  <ElFormItem label="选择节点">
                    <ElSelect
                      v-model="form.host"
                      placeholder="选择或输入节点"
                      clearable
                      filterable
                      allow-create
                      :loading="hostLoading"
                      @visible-change="handleHostDropdownVisible"
                    >
                      <ElOption
                        v-for="item in hostOptions"
                        :key="item"
                        :label="item"
                        :value="item"
                      />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>

                <ElCol :xs="24" :sm="12" :md="6">
                  <ElFormItem label="排序">
                    <ElSelect v-model="form.direction" placeholder="请选择排序方向">
                      <ElOption label="最新优先" value="backward" />
                      <ElOption label="最早优先" value="forward" />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>

                <ElCol :xs="24" :sm="12" :md="6">
                  <ElFormItem label="日志级别">
                    <ElSelect
                      v-model="form.level"
                      placeholder="选择日志级别"
                      clearable
                      filterable
                      allow-create
                      :loading="levelLoading"
                      @visible-change="handleLevelDropdownVisible"
                    >
                      <ElOption
                        v-for="item in levelOptions"
                        :key="item"
                        :label="item"
                        :value="item"
                      />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>

                <ElCol :xs="24" :sm="12" :md="6">
                  <ElFormItem label="返回条数">
                    <ElSlider v-model="form.maxLines" :min="20" :max="1000" :step="20" show-stops />
                  </ElFormItem>
                </ElCol>
              </ElRow>

              <div class="label-section">
                <div class="section-head compact-head">
                  <div>
                    <div class="section-title">Labels</div>
                    <div class="section-subtitle">按需添加标签过滤，避免重复条件。</div>
                  </div>
                  <div class="section-actions">
                    <ElButton size="small" plain @click="openLabelDialog">添加标签</ElButton>
                    <ElButton
                      size="small"
                      :disabled="labelFilters.length === 0"
                      @click="clearLabels"
                      >清空</ElButton
                    >
                  </div>
                </div>

                <div v-if="labelFilters.length > 0" class="label-list">
                  <ElTag
                    v-for="item in labelFilters"
                    :key="item.id"
                    closable
                    effect="plain"
                    @close="removeLabel(item.id)"
                  >
                    {{ item.key }} {{ item.operator }} {{ item.value }}
                  </ElTag>
                </div>
                <div v-else class="empty-hint">暂无标签条件</div>
              </div>
            </template>

            <template v-else>
              <ElRow :gutter="8">
                <ElCol :xs="24" :sm="12" :md="12">
                  <ElFormItem label="排序">
                    <ElSelect v-model="form.direction" placeholder="请选择排序方向">
                      <ElOption label="最新优先" value="backward" />
                      <ElOption label="最早优先" value="forward" />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>

                <ElCol :xs="24" :sm="12" :md="12">
                  <ElFormItem label="日志级别">
                    <ElSelect
                      v-model="form.level"
                      placeholder="选择日志级别"
                      clearable
                      filterable
                      allow-create
                      :loading="levelLoading"
                      @visible-change="handleLevelDropdownVisible"
                    >
                      <ElOption
                        v-for="item in levelOptions"
                        :key="item"
                        :label="item"
                        :value="item"
                      />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>

                <ElCol :xs="24" :sm="12" :md="12">
                  <ElFormItem label="返回条数">
                    <ElSlider v-model="form.maxLines" :min="20" :max="1000" :step="20" show-stops />
                  </ElFormItem>
                </ElCol>
              </ElRow>
            </template>
          </div>
        </ElCollapseTransition>

        <div v-show="editorMode === 'code' && canUseCodeMode" class="grafana-panel">
          <div class="grafana-head">
            <div>
              <div class="section-title">Code 模式</div>
              <div class="section-subtitle">选择 Loki 或 ES 后，手动切换到这里编辑。</div>
            </div>
            <div class="section-actions">
              <ElButton size="small" plain @click="syncQueryTextFromForm">从表单生成</ElButton>
              <ElButton size="small" :disabled="!form.queryText" @click="form.queryText = ''"
                >清空</ElButton
              >
              <ElButton size="small" plain @click="copyCurrentQuery">复制</ElButton>
            </div>
          </div>

          <div class="mode-hint-row">
            <span class="mode-hint">{{ queryMode === 'loki' ? 'Loki / LogQL' : 'ES / DSL' }}</span>
            <span class="mode-hint muted">{{ queryTextPlaceholder }}</span>
          </div>

          <ElInput
            v-model="form.queryText"
            type="textarea"
            :rows="6"
            class="grafana-editor"
            :placeholder="queryTextPlaceholder"
          />

          <div class="snippet-row">
            <ElTag
              v-for="snippet in querySnippets"
              :key="snippet"
              effect="plain"
              class="snippet-tag"
              @click="insertSnippet(snippet)"
            >
              {{ snippet }}
            </ElTag>
          </div>
        </div>

        <div v-if="activeFilterChips.length > 0" class="active-chip-row">
          <ElTag
            v-for="chip in activeFilterChips"
            :key="chip"
            size="small"
            effect="plain"
            class="active-chip"
          >
            {{ chip }}
          </ElTag>
        </div>

        <div class="bottom-bar">
          <div class="query-summary">
            <span class="summary-label">当前条件</span>
            <span class="summary-text">{{ querySummary || searchHintText }}</span>
          </div>
          <div class="section-actions">
            <ElButton @click="handleReset">重置</ElButton>
            <ElDropdown
              split-button
              type="primary"
              :disabled="isQuerying || !canSearch"
              @click="handleSearch"
              @command="handleQueryCommand"
            >
              查询
              <template #dropdown>
                <ElDropdownMenu>
                  <ElDropdownItem command="once">仅查询一次（默认）</ElDropdownItem>
                  <ElDropdownItem command="3">自动刷新 3s</ElDropdownItem>
                  <ElDropdownItem command="5">自动刷新 5s</ElDropdownItem>
                  <ElDropdownItem command="15">自动刷新 15s</ElDropdownItem>
                  <ElDropdownItem command="30">自动刷新 30s</ElDropdownItem>
                  <ElDropdownItem command="off" :disabled="autoRefreshSeconds === null">
                    关闭自动刷新
                  </ElDropdownItem>
                </ElDropdownMenu>
              </template>
            </ElDropdown>
          </div>
        </div>
      </ElForm>
    </section>

    <section class="result-card" v-loading="isQuerying" element-loading-text="日志查询中...">
      <div class="card-head">
        <div>
          <div class="card-title">查询结果</div>
          <div class="card-subtitle">{{ resultMeta }}</div>
        </div>

        <div class="result-actions">
          <ElDropdown
            trigger="click"
            :disabled="logs.length === 0 || isQuerying"
            @command="handleExportCommand"
          >
            <ElButton size="small" :disabled="logs.length === 0 || isQuerying">
              导出内容{{ selectedLogRows.length > 0 ? `（已选 ${selectedLogRows.length}）` : '' }}
            </ElButton>
            <template #dropdown>
              <ElDropdownMenu>
                <ElDropdownItem command="all-txt">导出全部 TXT</ElDropdownItem>
                <ElDropdownItem command="all-excel-desc">导出全部 Excel（时间降序）</ElDropdownItem>
                <ElDropdownItem command="all-excel-asc">导出全部 Excel（时间升序）</ElDropdownItem>
                <ElDropdownItem
                  divided
                  command="selected-txt"
                  :disabled="selectedLogRows.length === 0"
                >
                  导出选中 TXT
                </ElDropdownItem>
                <ElDropdownItem
                  command="selected-excel-desc"
                  :disabled="selectedLogRows.length === 0"
                >
                  导出选中 Excel（时间降序）
                </ElDropdownItem>
                <ElDropdownItem
                  command="selected-excel-asc"
                  :disabled="selectedLogRows.length === 0"
                >
                  导出选中 Excel（时间升序）
                </ElDropdownItem>
              </ElDropdownMenu>
            </template>
          </ElDropdown>
          <ElButton size="small" :disabled="logs.length === 0 || isQuerying" @click="selectAllRows"
            >全选</ElButton
          >
          <ElButton
            size="small"
            :disabled="selectedLogRows.length === 0 || isQuerying"
            @click="clearSelectedRows"
            >取消选择</ElButton
          >
          <ElButton size="small" :disabled="logs.length === 0 || isQuerying" @click="clearResult"
            >清空结果</ElButton
          >
          <ElButton size="small" :disabled="!nextToken || isQuerying" @click="loadMore"
            >加载更多</ElButton
          >
        </div>
      </div>

      <ElTable
        v-if="logs.length > 0"
        ref="logTableRef"
        :data="logs"
        row-key="id"
        class="log-table"
        height="500"
        stripe
        @selection-change="handleSelectionChange"
      >
        <ElTableColumn type="selection" width="42" fixed="left" :reserve-selection="true" />
        <ElTableColumn label="时间" width="198">
          <template #default="{ row }">
            <span class="time-cell" :title="formatTime(row.timestamp)">{{
              formatTime(row.timestamp)
            }}</span>
          </template>
        </ElTableColumn>

        <ElTableColumn label="正文" min-width="420">
          <template #default="{ row }">
            <div class="message-wrapper" @click="openLogDetail(row)">
              <div
                class="message-cell"
                :class="{ expanded: expandedMessageMap[row.id] }"
                v-html="getHighlightedDisplayMessage(row)"
              />
            </div>
          </template>
        </ElTableColumn>

        <ElTableColumn label="Pod名字" width="220">
          <template #default="{ row }">
            <ElTag
              v-if="getRowPodName(row)"
              size="small"
              effect="plain"
              class="clickable-tag"
              @click.stop="applyQuickFilter('pod_name', getRowPodName(row))"
            >
              {{ getRowPodName(row) }}
            </ElTag>
            <span v-else class="table-empty">-</span>
          </template>
        </ElTableColumn>

        <ElTableColumn label="容器名字" width="180">
          <template #default="{ row }">
            <ElTag
              v-if="getRowContainerName(row)"
              size="small"
              effect="plain"
              class="clickable-tag"
              @click.stop="applyQuickFilter('container_name', getRowContainerName(row))"
            >
              {{ getRowContainerName(row) }}
            </ElTag>
            <span v-else class="table-empty">-</span>
          </template>
        </ElTableColumn>

        <ElTableColumn label="节点" width="180">
          <template #default="{ row }">
            <ElTag
              v-if="getRowNodeName(row)"
              size="small"
              effect="plain"
              class="clickable-tag"
              @click.stop="applyQuickFilter(nodeFieldKey, getRowNodeName(row))"
            >
              {{ getRowNodeName(row) }}
            </ElTag>
            <span v-else class="table-empty">-</span>
          </template>
        </ElTableColumn>
      </ElTable>

      <div v-else class="empty-state">
        <ElEmpty description="暂无日志，设置条件后点击查询" />
      </div>
    </section>

    <ElDialog v-model="labelDialogVisible" title="编辑标签条件" width="640px">
      <div class="label-editor">
        <div v-for="(item, index) in labelDraftRows" :key="item.id" class="label-row">
          <ElSelect
            v-model="item.key"
            placeholder="label key"
            clearable
            filterable
            allow-create
            default-first-option
            :loading="labelKeyLoading"
            @change="() => handleDraftLabelKeyChange(item)"
            @visible-change="(visible) => visible && loadLabelKeys()"
          >
            <ElOption v-for="key in labelKeyOptions" :key="key" :label="key" :value="key" />
          </ElSelect>
          <ElSelect v-model="item.operator" class="operator-select">
            <ElOption v-for="op in labelOperators" :key="op" :label="op" :value="op" />
          </ElSelect>
          <ElSelect
            v-model="item.value"
            placeholder="label value"
            clearable
            filterable
            allow-create
            default-first-option
            :loading="labelValueLoadingMap[item.id]"
            @visible-change="(visible) => visible && loadDraftLabelValues(item)"
          >
            <ElOption
              v-for="value in getLabelValueOptions(item.key)"
              :key="`${item.id}-${value}`"
              :label="value"
              :value="value"
            />
          </ElSelect>
          <ElButton text circle @click="removeDraftRow(index)">删除</ElButton>
        </div>

        <ElButton class="add-label-btn" plain @click="addDraftRow">添加一行</ElButton>
      </div>

      <template #footer>
        <ElButton @click="labelDialogVisible = false">取消</ElButton>
        <ElButton type="primary" @click="applyLabels">确认</ElButton>
      </template>
    </ElDialog>

    <ElDialog v-model="logDetailVisible" title="日志详情" width="900px">
      <template v-if="currentLogDetail">
        <div class="detail-meta">
          <span>{{ formatTime(currentLogDetail.timestamp) }}</span>
          <span>{{ currentLogDetail.backendType || 'unknown' }}</span>
        </div>

        <div
          v-if="Object.keys(currentLogDetail.labels || {}).length > 0"
          class="log-labels detail-labels"
        >
          <ElTag
            v-for="(value, key) in currentLogDetail.labels"
            :key="`detail-${key}`"
            size="small"
            effect="plain"
            class="clickable-tag"
            @click="applyQuickFilter(key, value)"
          >
            {{ key }}: {{ value }}
          </ElTag>
        </div>

        <div class="detail-block">
          <div class="detail-head">
            <span>正文</span>
            <ElButton size="small" text @click="copyLogText(getRowDisplayMessage(currentLogDetail))"
              >复制</ElButton
            >
          </div>
          <pre class="log-message" v-html="getHighlightedDisplayMessage(currentLogDetail)" />
        </div>

        <div class="detail-block">
          <div class="detail-head">
            <span>原文</span>
            <ElButton
              size="small"
              text
              @click="copyLogText(currentLogDetail.raw || currentLogDetail.message)"
            >
              复制
            </ElButton>
          </div>
          <pre class="log-raw">{{ currentLogDetail.raw }}</pre>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
  import dayjs from 'dayjs'
  import * as XLSX from 'xlsx'
  import { ElMessage } from 'element-plus'
  import { getLogQueryModeOptionsApi, searchClusterApi, type Cluster } from '@/api/manager'
  import { getNodeListApi, type ClusterNodeInfo } from '@/api/manager/node'
  import {
    listLogLabelKeysApi,
    listLogLabelValuesApi,
    queryLogsApi,
    type LogQueryMode,
    type LogSourceType,
    type LogQueryRequest,
    type LogRecord
  } from '@/api/manager/log'
  import {
    searchProjectApi,
    searchProjectClusterApi,
    searchProjectWorkspaceApi,
    type ProjectCluster,
    type ProjectWorkspace
  } from '@/api/manager/project'
  import {
    searchApplicationApi,
    searchVersionApi,
    type OnecProjectApplication,
    type OnecProjectVersion
  } from '@/api/workload/application'
  import { buildLogCodeText, normalizeQueryModeByCapability } from './utils'

  defineOptions({ name: 'ClusterLogSearch' })

  type SourceType = 'container' | 'system'
  type QueryMode = 'platform' | 'loki' | 'es'
  type EditorMode = 'form' | 'code'
  type LabelOperator = '=' | '!=' | '=~' | '!~'

  interface LogLabelRow {
    id: string
    key: string
    operator: LabelOperator
    value: string
  }

  interface SearchForm {
    projectId?: number
    clusterId?: number
    workspaceId?: number
    applicationId?: number
    versionId?: number
    host: string
    podName: string
    containerName: string
    podIp: string
    level: string
    message: string
    direction: 'backward' | 'forward'
    maxLines: number
    queryText: string
  }

  interface ClusterItem {
    id: number
    name: string
    uuid: string
  }

  interface WorkspaceItem {
    id: number
    label: string
    namespace: string
  }

  interface LogViewItem extends LogRecord {
    id: string
  }

  const sourceTypeOptions: Array<{ value: SourceType; label: string }> = [
    { value: 'container', label: '容器日志' },
    { value: 'system', label: '系统日志' }
  ]

  const labelOperators: LabelOperator[] = ['=', '!=', '=~', '!~']
  const nodeFieldKey = 'node_name'
  const timeShortcuts = [
    { text: '最近 5 分钟', value: () => [dayjs().subtract(5, 'minute').valueOf(), Date.now()] },
    { text: '最近 15 分钟', value: () => [dayjs().subtract(15, 'minute').valueOf(), Date.now()] },
    { text: '最近 1 小时', value: () => [dayjs().subtract(1, 'hour').valueOf(), Date.now()] },
    { text: '最近 6 小时', value: () => [dayjs().subtract(6, 'hour').valueOf(), Date.now()] },
    { text: '最近 24 小时', value: () => [dayjs().subtract(24, 'hour').valueOf(), Date.now()] }
  ]

  const sourceType = ref<SourceType>('container')
  const queryMode = ref<QueryMode>('platform')
  const editorMode = ref<EditorMode>('form')
  const capability = ref({ platform: true, loki: false, es: false })
  const showMore = ref(false)
  const isQuerying = ref(false)
  const nextToken = ref('')
  const queryHistory = ref('')
  const autoRefreshSeconds = ref<number | null>(null)
  const labelDialogVisible = ref(false)
  const refreshTimer = ref<number | null>(null)

  const form = reactive<SearchForm>({
    host: '',
    podName: '',
    containerName: '',
    podIp: '',
    level: '',
    message: '',
    direction: 'backward',
    maxLines: 200,
    queryText: ''
  })

  const timeRange = ref<[number, number]>([dayjs().subtract(30, 'minute').valueOf(), Date.now()])
  const projectOptions = ref<ClusterItem[]>([])
  const clusterOptions = ref<ClusterItem[]>([])
  const workspaceOptions = ref<WorkspaceItem[]>([])
  const applicationOptions = ref<OnecProjectApplication[]>([])
  const versionOptions = ref<OnecProjectVersion[]>([])
  const nodeOptions = ref<string[]>([])
  const podNameOptions = ref<string[]>([])
  const containerNameOptions = ref<string[]>([])
  const levelOptions = ref<string[]>([])
  const logs = ref<LogViewItem[]>([])
  const projectLoading = ref(false)
  const clusterLoading = ref(false)
  const workspaceLoading = ref(false)
  const applicationLoading = ref(false)
  const versionLoading = ref(false)
  const nodeLoading = ref(false)
  const podNameLoading = ref(false)
  const containerNameLoading = ref(false)
  const podIpLoading = ref(false)
  const hostLoading = ref(false)
  const levelLoading = ref(false)

  const labelFilters = ref<LogLabelRow[]>([])
  const labelKeyOptions = ref<string[]>([])
  const labelValueOptions = reactive<Record<string, string[]>>({})
  const labelKeyLoading = ref(false)
  const labelValueLoadingMap = reactive<Record<string, boolean>>({})
  const podIpOptions = ref<string[]>([])
  const hostLabelOptions = ref<string[]>([])
  const labelDraftRows = ref<LogLabelRow[]>([
    { id: crypto.randomUUID(), key: '', operator: '=', value: '' }
  ])
  const expandedMessageMap = reactive<Record<string, boolean>>({})
  const selectedLogRows = ref<LogViewItem[]>([])
  const logTableRef = ref()
  const logDetailVisible = ref(false)
  const currentLogDetail = ref<LogViewItem | null>(null)
  const lastBackendType = ref('')
  const lastQueryMode = ref('')
  const lastDataStream = ref('')
  const lastIndexPattern = ref('')

  const sourceTypeLabel = computed(
    () => sourceTypeOptions.find((item) => item.value === sourceType.value)?.label || '日志'
  )
  const currentProject = computed(() =>
    projectOptions.value.find((item) => item.id === form.projectId)
  )
  const currentCluster = computed(() =>
    clusterOptions.value.find((item) => item.id === form.clusterId)
  )
  const currentClusterName = computed(() => currentCluster.value?.name || '')
  const currentNamespace = computed(() => {
    const workspace = workspaceOptions.value.find((item) => item.id === form.workspaceId)
    return workspace?.namespace || ''
  })
  const currentVersion = computed(() =>
    versionOptions.value.find((item) => item.id === form.versionId)
  )
  const hostOptions = computed(() => mergeUniqueStrings(nodeOptions.value, hostLabelOptions.value))

  const queryModeButtons = computed(() => [
    {
      key: 'platform',
      label: '平台',
      active: queryMode.value === 'platform',
      disabled: !capability.value.platform,
      onClick: () => handleQueryModeChange('platform')
    },
    {
      key: 'loki',
      label: 'Loki',
      active: queryMode.value === 'loki',
      disabled: !capability.value.loki,
      onClick: () => handleQueryModeChange('loki')
    },
    {
      key: 'es',
      label: 'ES',
      active: queryMode.value === 'es',
      disabled: !capability.value.es,
      onClick: () => handleQueryModeChange('es')
    }
  ])

  const editorModeButtons = computed(() => [
    {
      key: 'form',
      label: '表单',
      active: editorMode.value === 'form',
      disabled: false,
      onClick: () => handleEditorModeChange('form')
    },
    {
      key: 'code',
      label: 'Code',
      active: editorMode.value === 'code',
      disabled: !canUseCodeMode.value,
      onClick: () => handleEditorModeChange('code')
    }
  ])

  const canUseCodeMode = computed(
    () => queryMode.value !== 'platform' && (capability.value.loki || capability.value.es)
  )

  const queryTextPlaceholder = computed(() =>
    queryMode.value === 'loki'
      ? '{log_type="container", namespace_name="default", node_name="node-1"} |= "error"'
      : 'log_type:"container" AND node_name:"node-1" AND message:"error"'
  )

  const querySnippets = computed(() =>
    queryMode.value === 'loki'
      ? ['|= "error"', '|~ "warn|error"', 'level="error"', `node_name="node-1"`]
      : ['message:"error"', 'level:"error"', `node_name:"node-1"`, 'AND', 'OR']
  )

  const querySummary = computed(() => {
    const parts: string[] = []
    if (sourceType.value === 'container') {
      if (currentProject.value?.name) parts.push(`项目 ${currentProject.value.name}`)
      if (currentCluster.value?.name) parts.push(`集群 ${currentCluster.value.name}`)
      if (currentNamespace.value) parts.push(`空间 ${currentNamespace.value}`)
      if (currentVersion.value?.version) parts.push(`版本 ${currentVersion.value.version}`)
      if (form.podName) parts.push(`Pod ${form.podName}`)
      if (form.containerName) parts.push(`容器 ${form.containerName}`)
      if (form.host) parts.push(`节点 ${form.host}`)
    } else {
      if (currentCluster.value?.name) parts.push(`集群 ${currentCluster.value.name}`)
      if (form.host) parts.push(`节点 ${form.host}`)
    }

    if (form.level) parts.push(`级别 ${form.level}`)
    if (form.message) parts.push(`正文 ${form.message}`)
    if (labelFilters.value.length > 0) parts.push(`Labels ${labelFilters.value.length}`)
    if (queryMode.value !== 'platform') parts.push(queryMode.value.toUpperCase())
    if (editorMode.value === 'code' && queryMode.value !== 'platform') parts.push('Code')

    return parts.join(' · ')
  })

  const activeFilterChips = computed(() => {
    const chips: string[] = []
    if (queryMode.value !== 'platform') chips.push(`模式 ${queryMode.value.toUpperCase()}`)
    if (sourceType.value === 'container' && currentNamespace.value)
      chips.push(`空间 ${currentNamespace.value}`)
    if (form.host) chips.push(`节点 ${form.host}`)
    if (form.podName) chips.push(`Pod ${form.podName}`)
    if (form.containerName) chips.push(`容器 ${form.containerName}`)
    if (form.level) chips.push(`级别 ${form.level}`)
    if (form.message) chips.push(`正文 ${form.message}`)
    if (labelFilters.value.length > 0) chips.push(`标签 ${labelFilters.value.length}`)
    return chips
  })

  const resultMeta = computed(() => {
    const total = logs.value.length
    const selected = selectedLogRows.value.length
    const modeText =
      lastQueryMode.value || (queryMode.value === 'platform' ? 'platform' : queryMode.value)
    const capabilityText =
      queryMode.value === 'platform'
        ? '当前使用平台查询'
        : capability.value[queryMode.value]
          ? `${modeText.toUpperCase()} 已启用`
          : `${modeText.toUpperCase()} 不可用`
    const backendText = lastBackendType.value ? `后端 ${lastBackendType.value}` : ''
    const targetText = lastDataStream.value
      ? `数据流 ${lastDataStream.value}`
      : lastIndexPattern.value
        ? `索引 ${lastIndexPattern.value}`
        : ''
    const lastQueryText = queryHistory.value ? `最近查询 ${queryHistory.value}` : ''

    return [
      total > 0 ? `${total} 条结果` : '暂无结果',
      selected > 0 ? `已选 ${selected}` : '',
      capabilityText,
      backendText,
      targetText,
      lastQueryText,
      nextToken.value ? '可继续加载更多' : ''
    ]
      .filter(Boolean)
      .join(' · ')
  })

  const autoRefreshText = computed(() =>
    autoRefreshSeconds.value === null ? '自动刷新关闭' : `自动刷新 ${autoRefreshSeconds.value}s`
  )
  const canSearch = computed(() => {
    if (sourceType.value === 'system') {
      return Boolean(form.clusterId && form.host)
    }
    return Boolean(form.projectId && form.clusterId && form.workspaceId)
  })
  const searchHintText = computed(() =>
    sourceType.value === 'system'
      ? '请选择集群、节点后执行查询'
      : '请选择项目、集群、工作空间后执行查询'
  )

  const queryChangedByCode = ref(false)

  function formatTime(timestamp: number) {
    return dayjs(timestamp).format('YYYY-MM-DD HH:mm:ss')
  }

  function mergeUniqueStrings(...lists: string[][]) {
    const seen = new Set<string>()
    const result: string[] = []
    for (const list of lists) {
      for (const item of list) {
        const value = item?.trim()
        if (!value || seen.has(value)) continue
        seen.add(value)
        result.push(value)
      }
    }
    return result
  }

  function getRowLabelValue(labels: Record<string, string> | undefined, ...keys: string[]) {
    if (!labels) return ''
    for (const key of keys) {
      const value = labels[key]
      if (value) return value
    }
    return ''
  }

  function getRowNodeName(row: LogViewItem | LogRecord) {
    return getRowLabelValue(row.labels, nodeFieldKey, 'host', 'kubernetes.host')
  }

  function getRowPodName(row: LogViewItem | LogRecord) {
    return getRowLabelValue(row.labels, 'pod_name', 'kubernetes.pod_name')
  }

  function getRowContainerName(row: LogViewItem | LogRecord) {
    return getRowLabelValue(row.labels, 'container_name', 'kubernetes.container_name')
  }

  function parseStructuredLogPayload(value: string) {
    const text = value?.trim()
    if (!text || (!text.startsWith('{') && !text.startsWith('['))) return null
    try {
      return JSON.parse(text) as Record<string, unknown>
    } catch {
      return null
    }
  }

  function extractStructuredLogMessage(payload: Record<string, unknown> | null) {
    if (!payload) return ''
    if (typeof payload.message === 'string' && payload.message.trim()) {
      return payload.message.trim()
    }
    if (typeof payload.log === 'string' && payload.log.trim()) {
      return payload.log.trim()
    }
    return ''
  }

  function getRowDisplayMessage(row: LogViewItem | LogRecord | null | undefined) {
    if (!row) return '-'

    const directMessage = row.message?.trim()
    const directPayload = parseStructuredLogPayload(directMessage || '')
    const structuredMessage = extractStructuredLogMessage(directPayload)
    if (structuredMessage) return structuredMessage
    if (directMessage && !directPayload) return directMessage

    const rawMessage = row.raw?.trim()
    const rawPayload = parseStructuredLogPayload(rawMessage || '')
    const structuredRawMessage = extractStructuredLogMessage(rawPayload)
    if (structuredRawMessage) return structuredRawMessage
    if (rawMessage) return rawMessage

    const labelMessage = getRowLabelValue(row.labels, 'message', 'log')
    if (labelMessage) return labelMessage

    return '-'
  }

  function escapeHtml(value: string) {
    return value
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;')
  }

  function escapeRegex(value: string) {
    return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  }

  function normalizeMessageSearchPattern(input: string) {
    const trimmed = input.trim()
    if (!trimmed || trimmed === '*') return ''
    try {
      new RegExp(trimmed, 'i')
      return trimmed
    } catch {
      return escapeRegex(trimmed).replace(/\\\*/g, '.*').replace(/\\\?/g, '.')
    }
  }

  function buildMessageHighlightRegex() {
    const pattern = normalizeMessageSearchPattern(form.message)
    if (!pattern) return null
    try {
      return new RegExp(pattern, 'gi')
    } catch {
      return null
    }
  }

  function highlightMessageContent(content: string) {
    const text = content || '-'
    const matcher = buildMessageHighlightRegex()
    if (!matcher) {
      return escapeHtml(text)
    }

    const matches = Array.from(text.matchAll(matcher))
    if (matches.length === 0) {
      return escapeHtml(text)
    }

    let result = ''
    let lastIndex = 0
    for (const match of matches) {
      const raw = match[0]
      const index = match.index ?? 0
      if (!raw) continue
      result += escapeHtml(text.slice(lastIndex, index))
      result += `<mark class="message-highlight">${escapeHtml(raw)}</mark>`
      lastIndex = index + raw.length
    }
    result += escapeHtml(text.slice(lastIndex))
    return result
  }

  function getHighlightedDisplayMessage(row: LogViewItem | LogRecord | null | undefined) {
    return highlightMessageContent(getRowDisplayMessage(row))
  }

  function upsertLabelFilter(key: string, value: string) {
    const normalizedKey = key.trim()
    const normalizedValue = value.trim()
    if (!normalizedKey || !normalizedValue) return

    const exists = labelFilters.value.some(
      (item) =>
        item.key === normalizedKey && item.operator === '=' && item.value === normalizedValue
    )
    if (exists) return

    labelFilters.value = [
      ...labelFilters.value,
      { id: crypto.randomUUID(), key: normalizedKey, operator: '=', value: normalizedValue }
    ]
  }

  function applyQuickFilter(key: string, value: string) {
    const normalizedValue = value?.trim()
    if (!normalizedValue) return

    switch (key) {
      case nodeFieldKey:
      case 'host':
        form.host = normalizedValue
        break
      case 'pod_name':
        if (sourceType.value === 'container') form.podName = normalizedValue
        break
      case 'container_name':
        if (sourceType.value === 'container') form.containerName = normalizedValue
        break
      case 'level':
        form.level = normalizedValue
        break
      default:
        upsertLabelFilter(key, normalizedValue)
        break
    }

    if (queryMode.value !== 'platform' && editorMode.value === 'code') {
      syncQueryTextFromForm()
    }
    ElMessage.success(`已应用字段 ${key}`)
  }

  function getResolvedPodName() {
    if (sourceType.value !== 'container') return undefined
    if (form.podName) return form.podName
    if (currentVersion.value?.resourceName) return `${currentVersion.value.resourceName}*`
    return undefined
  }

  function getCodeParams() {
    return {
      sourceType: sourceType.value,
      clusterUuid: getClusterUuid(),
      projectUuid: currentProject.value?.uuid,
      namespaceName: getSelectedNamespace(),
      podName: getResolvedPodName(),
      containerName: sourceType.value === 'container' ? form.containerName || undefined : undefined,
      podIp: sourceType.value === 'container' ? form.podIp || undefined : undefined,
      host: form.host || undefined,
      level: form.level || undefined,
      message: form.message || undefined,
      from: timeRange.value[0],
      to: timeRange.value[1],
      maxLines: form.maxLines,
      queryMode: queryMode.value as LogQueryMode
    }
  }

  function syncQueryTextFromForm() {
    form.queryText = buildLogCodeText(getCodeParams(), capability.value, labelFilters.value)
    queryChangedByCode.value = false
  }

  function insertSnippet(text: string) {
    form.queryText = form.queryText ? `${form.queryText}\n${text}` : text
    queryChangedByCode.value = true
  }

  function handleSourceTypeChange(next: SourceType) {
    if (sourceType.value === next) return
    sourceType.value = next
    form.clusterId = undefined
    form.podName = ''
    form.containerName = ''
    form.podIp = ''
    form.host = ''
    form.level = ''
    form.applicationId = undefined
    form.versionId = undefined
    form.workspaceId = undefined
    form.message = ''
    form.queryText = ''
    labelFilters.value = []
    nextToken.value = ''
    logs.value = []
    showMore.value = false
    editorMode.value = 'form'
    queryChangedByCode.value = false
    podNameOptions.value = []
    containerNameOptions.value = []
    podIpOptions.value = []
    hostLabelOptions.value = []
    levelOptions.value = []
    labelKeyOptions.value = []
    clusterOptions.value = []
    resetLabelValueCache()
    if (next === 'system') {
      void loadSystemClusterList()
    }
    if (queryMode.value !== 'platform') {
      void loadLabelKeys()
      void preloadPrimaryFieldOptions()
    }
  }

  function handleQueryModeChange(next: QueryMode) {
    const normalized = normalizeQueryModeByCapability(
      next as LogQueryMode,
      capability.value
    ) as QueryMode
    queryMode.value = normalized

    if (normalized === 'platform') {
      editorMode.value = 'form'
      form.queryText = ''
      labelKeyOptions.value = []
      podNameOptions.value = []
      containerNameOptions.value = []
      podIpOptions.value = []
      hostLabelOptions.value = []
      levelOptions.value = []
      resetLabelValueCache()
      return
    }

    void loadLabelKeys()
    void preloadPrimaryFieldOptions()

    if (editorMode.value === 'code') {
      syncQueryTextFromForm()
    }
  }

  function handleEditorModeChange(next: EditorMode) {
    if (next === 'code') {
      if (!canUseCodeMode.value) {
        ElMessage.warning('请先选择 Loki 或 ES')
        return
      }
      syncQueryTextFromForm()
      editorMode.value = 'code'
      showMore.value = true
      return
    }

    editorMode.value = 'form'
  }

  async function loadProjectList() {
    projectLoading.value = true
    try {
      const resp = await searchProjectApi({ page: 1, pageSize: 200 })
      projectOptions.value = (resp?.items || []).map((item) => ({
        id: item.id,
        name: item.name,
        uuid: item.uuid
      }))
    } finally {
      projectLoading.value = false
    }
  }

  async function loadClusterList(projectId?: number) {
    clusterOptions.value = []
    workspaceOptions.value = []
    applicationOptions.value = []
    versionOptions.value = []
    nodeOptions.value = []

    if (!projectId) return

    clusterLoading.value = true
    try {
      const resp = await searchProjectClusterApi({ projectId })
      const items = (resp || []) as ProjectCluster[]
      clusterOptions.value = items.map((item) => ({
        id: item.id,
        name: item.clusterName,
        uuid: item.clusterUuid
      }))
    } finally {
      clusterLoading.value = false
    }
  }

  async function loadSystemClusterList() {
    clusterOptions.value = []
    clusterLoading.value = true
    try {
      const resp = await searchClusterApi({
        page: 1,
        pageSize: 100,
        name: '',
        environment: ''
      })
      const items = (resp?.items || []) as Cluster[]
      clusterOptions.value = items.map((item) => ({
        id: item.id,
        name: item.name,
        uuid: item.uuid
      }))
    } finally {
      clusterLoading.value = false
    }
  }

  async function loadWorkspaceList(clusterId?: number) {
    workspaceOptions.value = []
    applicationOptions.value = []
    versionOptions.value = []

    if (!clusterId) return

    workspaceLoading.value = true
    try {
      const resp = await searchProjectWorkspaceApi({ projectClusterId: clusterId })
      const items = (resp || []) as ProjectWorkspace[]
      workspaceOptions.value = items.map((item) => ({
        id: item.id,
        label: `${item.name} (${item.namespace})`,
        namespace: item.namespace
      }))
    } finally {
      workspaceLoading.value = false
    }
  }

  async function loadApplicationList(workspaceId?: number) {
    applicationOptions.value = []
    versionOptions.value = []

    if (!workspaceId) return

    applicationLoading.value = true
    try {
      const resp = await searchApplicationApi({ workspaceId })
      applicationOptions.value = (resp || []) as OnecProjectApplication[]
    } finally {
      applicationLoading.value = false
    }
  }

  async function loadVersionList(applicationId?: number): Promise<OnecProjectVersion[]> {
    versionOptions.value = []

    if (!applicationId) return []

    versionLoading.value = true
    try {
      const resp = await searchVersionApi({ applicationId })
      versionOptions.value = (resp || []) as OnecProjectVersion[]
      return versionOptions.value
    } finally {
      versionLoading.value = false
    }
  }

  async function loadNodeList(clusterUuid?: string) {
    nodeOptions.value = []
    if (!clusterUuid) return

    nodeLoading.value = true
    try {
      const resp = await getNodeListApi({ page: 1, pageSize: 200, clusterUuid })
      nodeOptions.value = (resp.items || [])
        .map((item: ClusterNodeInfo) => item.nodeName)
        .filter(Boolean)
    } finally {
      nodeLoading.value = false
    }
  }

  async function loadQueryCapability(clusterUuid?: string) {
    capability.value = { platform: true, loki: false, es: false }
    if (!clusterUuid) return

    try {
      const resp = await getLogQueryModeOptionsApi(clusterUuid)
      capability.value = {
        platform: true,
        loki: !!resp?.loki,
        es: !!resp?.elasticsearch
      }
    } catch {
      capability.value = { platform: true, loki: false, es: false }
    }

    queryMode.value = normalizeQueryModeByCapability(
      queryMode.value as LogQueryMode,
      capability.value
    ) as QueryMode
    if (queryMode.value === 'platform') {
      editorMode.value = 'form'
      form.queryText = ''
      labelKeyOptions.value = []
      podNameOptions.value = []
      containerNameOptions.value = []
      podIpOptions.value = []
      hostLabelOptions.value = []
      levelOptions.value = []
      resetLabelValueCache()
      return
    }

    await loadLabelKeys()
    await preloadPrimaryFieldOptions()
  }

  function getClusterUuid() {
    return clusterOptions.value.find((item) => item.id === form.clusterId)?.uuid || ''
  }

  function getSelectedNamespace() {
    return workspaceOptions.value.find((item) => item.id === form.workspaceId)?.namespace || ''
  }

  function getCurrentLogSourceType(): LogSourceType {
    return sourceType.value === 'container' ? 'container' : 'system'
  }

  function getCurrentLogQueryMode(): LogQueryMode | undefined {
    if (queryMode.value === 'platform') return undefined
    return queryMode.value
  }

  function getMetadataQueryMode(): LogQueryMode | undefined {
    const directMode = getCurrentLogQueryMode()
    if (directMode) return directMode
    if (capability.value.loki) return 'loki'
    if (capability.value.es) return 'es'
    return undefined
  }

  function getLabelScopeRequest() {
    const clusterUuid = getClusterUuid()
    const resolvedMode = getMetadataQueryMode()
    if (!clusterUuid || !resolvedMode) return null

    return {
      sourceType: getCurrentLogSourceType(),
      clusterUuid,
      queryMode: resolvedMode,
      workspaceId: form.workspaceId,
      namespace: getSelectedNamespace() || undefined,
      application:
        applicationOptions.value.find((item) => item.id === form.applicationId)?.nameEn ||
        undefined,
      resourceName: currentVersion.value?.resourceName || undefined,
      hosts: form.host ? [form.host] : undefined
    }
  }

  function resetLabelValueCache() {
    Object.keys(labelValueOptions).forEach((key) => {
      delete labelValueOptions[key]
    })
    Object.keys(labelValueLoadingMap).forEach((key) => {
      delete labelValueLoadingMap[key]
    })
  }

  async function loadLabelKeys() {
    const scope = getLabelScopeRequest()
    if (!scope) {
      labelKeyOptions.value = []
      return
    }

    labelKeyLoading.value = true
    try {
      const resp = await listLogLabelKeysApi(scope)
      labelKeyOptions.value = resp?.keys || []
    } catch {
      labelKeyOptions.value = []
    } finally {
      labelKeyLoading.value = false
    }
  }

  async function loadLabelValuesByKey(
    labelKey: string,
    options: { silent?: boolean; force?: boolean } = {}
  ) {
    const key = labelKey.trim()
    if (!key) return [] as string[]

    if (!options.force && labelValueOptions[key]) {
      return labelValueOptions[key]
    }

    const scope = getLabelScopeRequest()
    if (!scope) {
      labelValueOptions[key] = []
      return []
    }

    if (!options.silent) labelValueLoadingMap[key] = true
    try {
      const resp = await listLogLabelValuesApi({ ...scope, labelKey: key })
      const values = resp?.values || []
      labelValueOptions[key] = values
      return values
    } catch {
      labelValueOptions[key] = []
      return []
    } finally {
      if (!options.silent) labelValueLoadingMap[key] = false
    }
  }

  async function preloadPrimaryFieldOptions() {
    if (queryMode.value === 'platform' && !capability.value.loki && !capability.value.es) return
    podNameLoading.value = sourceType.value === 'container'
    containerNameLoading.value = sourceType.value === 'container'
    podIpLoading.value = sourceType.value === 'container'
    hostLoading.value = true
    levelLoading.value = true
    try {
      const tasks: Array<Promise<string[]>> = []
      if (sourceType.value === 'container') {
        tasks.push(loadLabelValuesByKey('pod_name', { silent: true }))
        tasks.push(loadLabelValuesByKey('container_name', { silent: true }))
        tasks.push(loadLabelValuesByKey('pod_ip', { silent: true }))
      } else {
        tasks.push(Promise.resolve([]))
        tasks.push(Promise.resolve([]))
        tasks.push(Promise.resolve([]))
      }
      tasks.push(loadLabelValuesByKey(nodeFieldKey, { silent: true }))
      tasks.push(loadLabelValuesByKey('level', { silent: true }))

      const [pods, containers, podIps, hosts, levels] = await Promise.all(tasks)
      podNameOptions.value = pods
      containerNameOptions.value = containers
      podIpOptions.value = podIps
      hostLabelOptions.value = hosts
      levelOptions.value = levels
    } finally {
      podNameLoading.value = false
      containerNameLoading.value = false
      podIpLoading.value = false
      hostLoading.value = false
      levelLoading.value = false
    }
  }

  async function handlePodNameDropdownVisible(visible: boolean) {
    if (!visible) return
    podNameLoading.value = true
    try {
      podNameOptions.value = await loadLabelValuesByKey('pod_name')
    } finally {
      podNameLoading.value = false
    }
  }

  async function handleContainerNameDropdownVisible(visible: boolean) {
    if (!visible) return
    containerNameLoading.value = true
    try {
      containerNameOptions.value = await loadLabelValuesByKey('container_name')
    } finally {
      containerNameLoading.value = false
    }
  }

  async function handlePodIpDropdownVisible(visible: boolean) {
    if (!visible) return
    podIpLoading.value = true
    try {
      podIpOptions.value = await loadLabelValuesByKey('pod_ip')
    } finally {
      podIpLoading.value = false
    }
  }

  async function handleHostDropdownVisible(visible: boolean) {
    if (!visible) return
    hostLoading.value = true
    try {
      const dynamicHosts = await loadLabelValuesByKey(nodeFieldKey)
      hostLabelOptions.value = dynamicHosts
    } finally {
      hostLoading.value = false
    }
  }

  async function handleLevelDropdownVisible(visible: boolean) {
    if (!visible) return
    levelLoading.value = true
    try {
      levelOptions.value = await loadLabelValuesByKey('level')
    } finally {
      levelLoading.value = false
    }
  }

  function getLabelValueOptions(labelKey: string) {
    return labelValueOptions[labelKey?.trim()] || []
  }

  function handleDraftLabelKeyChange(item: LogLabelRow) {
    item.value = ''
    const key = item.key?.trim()
    if (!key) return
    void loadLabelValuesByKey(key)
  }

  async function loadDraftLabelValues(item: LogLabelRow) {
    const key = item.key?.trim()
    if (!key) return
    labelValueLoadingMap[item.id] = true
    try {
      await loadLabelValuesByKey(key)
    } finally {
      labelValueLoadingMap[item.id] = false
    }
  }

  function handleVersionChange() {
    if (editorMode.value === 'code') syncQueryTextFromForm()
  }

  function buildRequest(append = false): LogQueryRequest {
    if (sourceType.value === 'container' && !form.projectId) {
      throw new Error('请选择项目')
    }
    const clusterUuid = getClusterUuid()
    if (!clusterUuid) {
      throw new Error('请选择集群')
    }
    if (sourceType.value === 'container' && !form.workspaceId) {
      throw new Error('请选择工作空间')
    }
    if (sourceType.value === 'system' && !form.host) {
      throw new Error('请选择节点')
    }

    const resolvedMode = getCurrentLogQueryMode()
    const request: LogQueryRequest = {
      sourceType: sourceType.value,
      logType: sourceType.value,
      clusterUuid,
      start: String(timeRange.value[0]),
      end: String(timeRange.value[1]),
      limit: form.maxLines,
      direction: form.direction,
      mode: 'form'
    }

    if (currentProject.value?.uuid) request.projectUuid = currentProject.value.uuid

    if (sourceType.value === 'container') {
      const namespace = getSelectedNamespace()
      if (namespace) request.namespace = namespace
      const podName = getResolvedPodName()
      if (podName) request.podName = podName
      if (form.containerName) request.containerName = form.containerName
      if (form.podIp) request.podIp = form.podIp
      if (form.host) {
        request.host = form.host
        request.hosts = [form.host]
      }
    } else if (form.host) {
      request.host = form.host
      request.hosts = [form.host]
    }

    if (form.message) request.message = form.message
    if (form.level) request.level = form.level
    if (resolvedMode) request.queryMode = resolvedMode

    if (resolvedMode && editorMode.value === 'code') {
      request.mode = 'code'
      request.expr =
        form.queryText?.trim() ||
        buildLogCodeText(getCodeParams(), capability.value, labelFilters.value)
    } else if (resolvedMode && labelFilters.value.length > 0) {
      request.mode = 'code'
      request.expr = buildLogCodeText(getCodeParams(), capability.value, labelFilters.value)
    }

    if (append && nextToken.value) request.nextToken = nextToken.value

    return request
  }

  async function executeQuery(append = false) {
    if (isQuerying.value) return

    try {
      isQuerying.value = true
      const params = buildRequest(append)
      const resp = await queryLogsApi(params)

      const list = (resp?.list || []).map((item, index) => ({
        ...item,
        id: `${item.timestamp}-${index}-${crypto.randomUUID()}`
      }))

      if (!append) {
        Object.keys(expandedMessageMap).forEach((key) => {
          delete expandedMessageMap[key]
        })
        selectedLogRows.value = []
        if (logTableRef.value?.clearSelection) {
          logTableRef.value.clearSelection()
        }
      }
      logs.value = append ? [...logs.value, ...list] : list
      nextToken.value = resp?.nextToken || ''
      queryHistory.value = dayjs().format('HH:mm:ss')
      lastBackendType.value = resp?.backendType || ''
      lastQueryMode.value = resp?.queryMode || ''
      lastDataStream.value = resp?.dataStream || ''
      lastIndexPattern.value = resp?.indexPattern || ''

      if (queryMode.value !== 'platform' && editorMode.value === 'code' && !form.queryText) {
        syncQueryTextFromForm()
      }

      ElMessage.success(`已获取 ${list.length} 条日志`)
    } catch (error: any) {
      ElMessage.error(error?.message || '查询失败')
    } finally {
      isQuerying.value = false
    }
  }

  function handleSearch() {
    if (sourceType.value === 'container' && !form.projectId) {
      ElMessage.warning('请选择项目')
      return
    }
    if (!form.clusterId) {
      ElMessage.warning('请选择集群')
      return
    }
    if (sourceType.value === 'container' && !form.workspaceId) {
      ElMessage.warning('请选择工作空间')
      return
    }
    if (sourceType.value === 'system' && !form.host) {
      ElMessage.warning('请选择节点')
      return
    }

    if (queryMode.value !== 'platform' && editorMode.value === 'code' && !form.queryText) {
      syncQueryTextFromForm()
    }

    nextToken.value = ''
    executeQuery(false)
  }

  function handleQueryCommand(command: string | number) {
    if (command === 'once' || command === 'off') {
      stopAutoRefresh()
      autoRefreshSeconds.value = null
      ElMessage.success(command === 'once' ? '已切换为单次查询模式' : '已关闭自动刷新')
      return
    }

    const seconds = Number(command)
    if (!Number.isFinite(seconds) || seconds <= 0) return

    autoRefreshSeconds.value = seconds
    startAutoRefresh(seconds)
    ElMessage.success(`已开启自动刷新（${seconds}s）`)
  }

  function loadMore() {
    if (!nextToken.value) return
    executeQuery(true)
  }

  function clearResult() {
    logs.value = []
    nextToken.value = ''
    currentLogDetail.value = null
    lastBackendType.value = ''
    lastQueryMode.value = ''
    lastDataStream.value = ''
    lastIndexPattern.value = ''
    selectedLogRows.value = []
    Object.keys(expandedMessageMap).forEach((key) => {
      delete expandedMessageMap[key]
    })
    if (logTableRef.value?.clearSelection) {
      logTableRef.value.clearSelection()
    }
  }

  function handleReset() {
    form.projectId = undefined
    form.clusterId = undefined
    form.workspaceId = undefined
    form.applicationId = undefined
    form.versionId = undefined
    form.host = ''
    form.podName = ''
    form.containerName = ''
    form.podIp = ''
    form.level = ''
    form.message = ''
    form.queryText = ''
    form.direction = 'backward'
    form.maxLines = 200
    labelFilters.value = []
    labelKeyOptions.value = []
    podNameOptions.value = []
    containerNameOptions.value = []
    podIpOptions.value = []
    hostLabelOptions.value = []
    levelOptions.value = []
    resetLabelValueCache()
    showMore.value = false
    editorMode.value = 'form'
    queryChangedByCode.value = false
    clearResult()
  }

  function handleProjectChange() {
    form.clusterId = undefined
    form.workspaceId = undefined
    form.applicationId = undefined
    form.versionId = undefined
    form.host = ''
    form.podName = ''
    form.containerName = ''
    form.podIp = ''
    form.level = ''
    labelKeyOptions.value = []
    podNameOptions.value = []
    containerNameOptions.value = []
    podIpOptions.value = []
    hostLabelOptions.value = []
    levelOptions.value = []
    resetLabelValueCache()
    clearResult()
    loadClusterList(form.projectId)
    if (editorMode.value === 'code') syncQueryTextFromForm()
  }

  async function handleClusterChange() {
    form.workspaceId = undefined
    form.applicationId = undefined
    form.versionId = undefined
    form.host = ''
    form.podName = ''
    form.containerName = ''
    form.podIp = ''
    form.level = ''
    resetLabelValueCache()
    clearResult()

    const clusterUuid = getClusterUuid()
    const tasks = [loadNodeList(clusterUuid), loadQueryCapability(clusterUuid)]
    if (sourceType.value === 'container') {
      tasks.unshift(loadWorkspaceList(form.clusterId))
    } else {
      workspaceOptions.value = []
    }
    await Promise.all(tasks)
    if (editorMode.value === 'code') syncQueryTextFromForm()
  }

  async function handleWorkspaceChange() {
    form.applicationId = undefined
    form.versionId = undefined
    applicationOptions.value = []
    versionOptions.value = []
    resetLabelValueCache()
    if (sourceType.value === 'container') {
      await loadApplicationList(form.workspaceId)
    }
    if (queryMode.value !== 'platform') {
      await loadLabelKeys()
      await preloadPrimaryFieldOptions()
    }
    if (editorMode.value === 'code') syncQueryTextFromForm()
  }

  async function handleApplicationChange() {
    form.versionId = undefined
    resetLabelValueCache()
    const versions = await loadVersionList(form.applicationId)
    if (versions.length > 0) {
      form.versionId = versions[0].id
    }
    if (queryMode.value !== 'platform') {
      await loadLabelKeys()
      await preloadPrimaryFieldOptions()
    }
    if (editorMode.value === 'code') syncQueryTextFromForm()
  }

  async function openLabelDialog() {
    labelDraftRows.value =
      labelFilters.value.length > 0
        ? labelFilters.value.map((item) => ({ ...item }))
        : [{ id: crypto.randomUUID(), key: '', operator: '=', value: '' }]
    await loadLabelKeys()
    labelDialogVisible.value = true
  }

  function addDraftRow() {
    labelDraftRows.value.push({ id: crypto.randomUUID(), key: '', operator: '=', value: '' })
  }

  function removeDraftRow(index: number) {
    labelDraftRows.value.splice(index, 1)
    if (labelDraftRows.value.length === 0) addDraftRow()
  }

  function applyLabels() {
    labelFilters.value = labelDraftRows.value.filter((item) => item.key.trim() && item.value.trim())
    if (editorMode.value === 'code') {
      syncQueryTextFromForm()
    }
    labelDialogVisible.value = false
  }

  function removeLabel(id: string) {
    labelFilters.value = labelFilters.value.filter((item) => item.id !== id)
  }

  function clearLabels() {
    labelFilters.value = []
    if (editorMode.value === 'code') {
      syncQueryTextFromForm()
    }
  }

  async function copyCurrentQuery() {
    const text =
      editorMode.value === 'code' && form.queryText
        ? form.queryText
        : queryMode.value !== 'platform' && labelFilters.value.length > 0
          ? buildLogCodeText(getCodeParams(), capability.value, labelFilters.value)
          : querySummary.value
    if (!text) {
      ElMessage.info('没有可复制的查询内容')
      return
    }
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  }

  async function copyLogText(text: string) {
    const content = text?.trim()
    if (!content) {
      ElMessage.info('当前内容为空')
      return
    }
    await navigator.clipboard.writeText(content)
    ElMessage.success('复制成功')
  }

  function handleExportCommand(command: string | number) {
    switch (command) {
      case 'all-txt':
        exportLogsAsTxt('all')
        return
      case 'all-excel-desc':
        exportLogsAsExcel('all', 'desc')
        return
      case 'all-excel-asc':
        exportLogsAsExcel('all', 'asc')
        return
      case 'selected-txt':
        exportLogsAsTxt('selected')
        return
      case 'selected-excel-desc':
        exportLogsAsExcel('selected', 'desc')
        return
      case 'selected-excel-asc':
        exportLogsAsExcel('selected', 'asc')
        return
      default:
    }
  }

  function buildExportBaseName() {
    return `日志内容_${dayjs().format('YYYYMMDD_HHmmss')}`
  }

  function handleSelectionChange(rows: LogViewItem[]) {
    selectedLogRows.value = rows
  }

  function selectAllRows() {
    if (!logTableRef.value || logs.value.length === 0) return
    for (const row of logs.value) {
      logTableRef.value.toggleRowSelection(row, true)
    }
  }

  function clearSelectedRows() {
    if (!logTableRef.value) return
    logTableRef.value.clearSelection()
    selectedLogRows.value = []
  }

  function getExportRows(scope: 'all' | 'selected') {
    if (scope === 'selected') {
      if (selectedLogRows.value.length === 0) {
        ElMessage.info('请先选择需要导出的日志')
        return [] as LogViewItem[]
      }
      return selectedLogRows.value
    }
    if (logs.value.length === 0) {
      ElMessage.info('暂无可导出的内容')
      return [] as LogViewItem[]
    }
    return logs.value
  }

  function getLabelValue(labels: Record<string, string> | undefined, ...keys: string[]) {
    if (!labels) return ''
    for (const key of keys) {
      const value = labels[key]
      if (value) return value
    }
    return ''
  }

  function normalizeExportText(value: string, maxLen = 12000) {
    const text = (value || '').replace(/\0/g, '').trim()
    if (text.length <= maxLen) return text
    return `${text.slice(0, maxLen)}...(已截断)`
  }

  function splitTextForExcel(value: string, chunkSize = 88) {
    const text = normalizeExportText(value)
    if (!text) return ''
    const lines: string[] = []
    for (const rawLine of text.split('\n')) {
      if (rawLine.length <= chunkSize) {
        lines.push(rawLine)
        continue
      }
      for (let i = 0; i < rawLine.length; i += chunkSize) {
        lines.push(rawLine.slice(i, i + chunkSize))
      }
    }
    return lines.join('\n')
  }

  function sortRowsByTimestamp(rows: LogViewItem[], order: 'asc' | 'desc') {
    const cloned = [...rows]
    cloned.sort((a, b) => {
      if (order === 'asc') return a.timestamp - b.timestamp
      return b.timestamp - a.timestamp
    })
    return cloned
  }

  function buildWorksheetRowsMeta(rows: Array<Record<string, string | number>>) {
    const heights = new Array(rows.length + 1).fill(null)
    heights[0] = { hpt: 22 }
    for (let i = 0; i < rows.length; i += 1) {
      const message = String(rows[i].正文 || '')
      const raw = String(rows[i].原文 || '')
      const lines = Math.max(
        1,
        Math.ceil(message.length / 88),
        Math.ceil(raw.length / 116),
        message.split('\n').length,
        raw.split('\n').length
      )
      heights[i + 1] = { hpt: Math.min(88, Math.max(20, 16 + lines * 4)) }
    }
    return heights
  }

  function exportLogsAsTxt(scope: 'all' | 'selected') {
    const rows = getExportRows(scope)
    if (rows.length === 0) return

    try {
      const content = rows
        .map((item, index) => {
          const labels = Object.entries(item.labels || {})
            .map(([key, value]) => `${key}=${value}`)
            .join(', ')
          return [
            `#${index + 1}`,
            `时间: ${formatTime(item.timestamp)}`,
            `来源: ${item.backendType || 'unknown'}`,
            `正文: ${getRowDisplayMessage(item)}`,
            `标签: ${labels || '-'}`,
            `原文: ${item.raw || ''}`
          ].join('\n')
        })
        .join('\n\n------------------------------\n\n')

      const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${buildExportBaseName()}_${scope === 'selected' ? '选中' : '全部'}.txt`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
      ElMessage.success('TXT 导出成功')
    } catch (error: any) {
      ElMessage.error(error?.message || 'TXT 导出失败')
    }
  }

  function exportLogsAsExcel(scope: 'all' | 'selected', order: 'asc' | 'desc') {
    const rows = getExportRows(scope)
    if (rows.length === 0) return

    try {
      const sortedRows = sortRowsByTimestamp(rows, order)
      const dataRows = sortedRows.map((item, index) => ({
        序号: index + 1,
        时间: formatTime(item.timestamp),
        来源: item.backendType || 'unknown',
        项目: normalizeExportText(
          getLabelValue(item.labels, 'project_uuid', 'kubernetes.project_uuid'),
          256
        ),
        集群: normalizeExportText(
          getLabelValue(item.labels, 'cluster_uuid', 'kubernetes.cluster_uuid'),
          256
        ),
        命名空间: normalizeExportText(
          getLabelValue(item.labels, 'namespace_name', 'kubernetes.namespace_name'),
          256
        ),
        Pod: normalizeExportText(
          getLabelValue(item.labels, 'pod_name', 'kubernetes.pod_name'),
          1024
        ),
        容器: normalizeExportText(
          getLabelValue(item.labels, 'container_name', 'kubernetes.container_name'),
          1024
        ),
        PodIP: normalizeExportText(getLabelValue(item.labels, 'pod_ip', 'kubernetes.pod_ip'), 128),
        节点: normalizeExportText(
          getLabelValue(item.labels, nodeFieldKey, 'host', 'kubernetes.host'),
          512
        ),
        级别: normalizeExportText(getLabelValue(item.labels, 'level'), 64),
        日志类型: normalizeExportText(getLabelValue(item.labels, 'log_type'), 64),
        正文: splitTextForExcel(getRowDisplayMessage(item), 88),
        标签: Object.entries(item.labels || {})
          .map(([key, value]) => `${key}=${value}`)
          .join(', '),
        标签JSON: normalizeExportText(JSON.stringify(item.labels || {}), 20000),
        原文: splitTextForExcel(item.raw || '', 116)
      }))

      const worksheet = XLSX.utils.json_to_sheet(dataRows)
      worksheet['!cols'] = [
        { wch: 6 },
        { wch: 20 },
        { wch: 10 },
        { wch: 20 },
        { wch: 38 },
        { wch: 20 },
        { wch: 30 },
        { wch: 22 },
        { wch: 16 },
        { wch: 18 },
        { wch: 10 },
        { wch: 12 },
        { wch: 48 },
        { wch: 60 },
        { wch: 44 },
        { wch: 72 }
      ]
      worksheet['!rows'] = buildWorksheetRowsMeta(dataRows)
      worksheet['!autofilter'] = { ref: `A1:P${dataRows.length + 1}` }
      // 尝试冻结首行（兼容多数查看器，不支持时会自动忽略）
      ;(worksheet as any)['!freeze'] = { xSplit: 0, ySplit: 1 }

      // 头部单元格强化；部分查看器可能忽略样式，但不影响导出可用性。
      const range = XLSX.utils.decode_range(worksheet['!ref'] || 'A1')
      for (let c = range.s.c; c <= range.e.c; c += 1) {
        const addr = XLSX.utils.encode_cell({ r: 0, c })
        if (!worksheet[addr]) continue
        ;(worksheet[addr] as any).s = {
          font: { bold: true },
          alignment: { horizontal: 'center', vertical: 'center', wrapText: true }
        }
      }

      const labelRows = sortedRows.flatMap((item, index) => {
        const labels = item.labels || {}
        const entries = Object.entries(labels)
        if (entries.length === 0) {
          return [
            {
              序号: index + 1,
              时间: formatTime(item.timestamp),
              来源: item.backendType || 'unknown',
              标签键: '-',
              标签值: '-'
            }
          ]
        }
        return entries.map(([key, value]) => ({
          序号: index + 1,
          时间: formatTime(item.timestamp),
          来源: item.backendType || 'unknown',
          标签键: normalizeExportText(key, 256),
          标签值: normalizeExportText(value, 4000)
        }))
      })
      const labelSheet = XLSX.utils.json_to_sheet(labelRows)
      labelSheet['!cols'] = [{ wch: 8 }, { wch: 22 }, { wch: 12 }, { wch: 28 }, { wch: 72 }]
      labelSheet['!autofilter'] = { ref: `A1:E${labelRows.length + 1}` }
      ;(labelSheet as any)['!freeze'] = { xSplit: 0, ySplit: 1 }

      const metaRows = [
        { 字段: '导出时间', 值: dayjs().format('YYYY-MM-DD HH:mm:ss') },
        { 字段: '导出范围', 值: scope === 'selected' ? '选中日志' : '全部日志' },
        { 字段: '导出排序', 值: order === 'asc' ? '时间升序' : '时间降序' },
        { 字段: '日志类型', 值: sourceTypeLabel.value },
        {
          字段: '查询模式',
          值: queryMode.value === 'platform' ? '平台' : queryMode.value.toUpperCase()
        },
        { 字段: '筛选摘要', 值: querySummary.value || '-' },
        { 字段: '导出条数', 值: String(rows.length) }
      ]
      const metaSheet = XLSX.utils.json_to_sheet(metaRows)
      metaSheet['!cols'] = [{ wch: 16 }, { wch: 84 }]

      const workbook = XLSX.utils.book_new()
      XLSX.utils.book_append_sheet(workbook, worksheet, '日志内容')
      XLSX.utils.book_append_sheet(workbook, labelSheet, '标签明细')
      XLSX.utils.book_append_sheet(workbook, metaSheet, '导出信息')
      XLSX.writeFile(
        workbook,
        `${buildExportBaseName()}_${scope === 'selected' ? '选中' : '全部'}_${order === 'asc' ? '升序' : '降序'}.xlsx`
      )
      ElMessage.success('Excel 导出成功')
    } catch (error: any) {
      ElMessage.error(error?.message || 'Excel 导出失败')
    }
  }

  function openLogDetail(item: LogViewItem) {
    currentLogDetail.value = item
    logDetailVisible.value = true
  }

  function formatVersionLabel(item: OnecProjectVersion) {
    return item.version || item.resourceName || `版本 ${item.id}`
  }

  watch(queryMode, (mode) => {
    if (mode === 'platform') {
      editorMode.value = 'form'
      form.queryText = ''
      queryChangedByCode.value = false
      labelKeyOptions.value = []
      podNameOptions.value = []
      containerNameOptions.value = []
      podIpOptions.value = []
      hostLabelOptions.value = []
      levelOptions.value = []
      resetLabelValueCache()
      return
    }

    void loadLabelKeys()
    void preloadPrimaryFieldOptions()
  })

  function startAutoRefresh(seconds: number) {
    if (refreshTimer.value) clearInterval(refreshTimer.value)
    if (!Number.isFinite(seconds) || seconds <= 0) return
    refreshTimer.value = window.setInterval(() => {
      if (queryHistory.value) {
        executeQuery(false)
      }
    }, seconds * 1000)
  }

  function stopAutoRefresh() {
    if (refreshTimer.value) {
      clearInterval(refreshTimer.value)
      refreshTimer.value = null
    }
  }

  onMounted(async () => {
    await loadProjectList()
  })

  onBeforeUnmount(() => {
    stopAutoRefresh()
  })
</script>

<style scoped lang="scss">
  .log-search-page {
    padding: 6px;
  }

  .page-header,
  .search-card,
  .result-card {
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 14px;
    background: var(--el-bg-color);
    box-shadow: 0 8px 24px rgba(15, 23, 42, 0.04);
  }

  .page-header {
    display: flex;
    justify-content: space-between;
    gap: 10px;
    align-items: flex-start;
    padding: 8px 10px;
    margin-bottom: 6px;
  }

  .page-title-group {
    min-width: 0;

    .eyebrow {
      font-size: 11px;
      color: var(--el-color-primary);
      letter-spacing: 0.16em;
      text-transform: uppercase;
      margin-bottom: 4px;
    }

    h1 {
      margin: 0;
      font-size: 16px;
      line-height: 1.2;
      color: var(--el-text-color-primary);
    }

    p {
      margin: 3px 0 0;
      color: var(--el-text-color-secondary);
      font-size: 12px;
      line-height: 1.35;
    }
  }

  .status-group {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .status-chip {
    padding: 4px 10px;
    border-radius: 999px;
    background: var(--el-color-primary-light-9);
    color: var(--el-color-primary);
    font-size: 12px;
    font-weight: 600;

    &.muted {
      background: var(--el-fill-color-light);
      color: var(--el-text-color-secondary);
    }

    &.container {
      background: rgba(64, 158, 255, 0.12);
    }

    &.system {
      background: rgba(103, 194, 58, 0.12);
      color: #2e7d32;
    }
  }

  .search-card,
  .result-card {
    padding: 8px 10px;
    margin-bottom: 6px;
  }

  .card-topbar,
  .card-head,
  .grafana-head,
  .bottom-bar,
  .aux-row,
  .log-item-head,
  .code-switch-row {
    display: flex;
    justify-content: space-between;
    gap: 10px;
    align-items: flex-start;
  }

  .card-topbar {
    align-items: center;
    margin-bottom: 6px;
  }

  .card-title,
  .section-title,
  .code-switch-label {
    font-size: 14px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  .card-subtitle,
  .section-subtitle {
    margin-top: 2px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }

  .segment-group,
  .result-actions,
  .section-actions,
  .snippet-row,
  .label-list,
  .capability-list,
  .action-inline {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .segment-group {
    padding: 2px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 12px;
    background: var(--el-fill-color-lighter);
  }

  .segment-btn {
    border: 0;
    background: transparent;
    color: var(--el-text-color-secondary);
    padding: 7px 12px;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.18s ease;

    &.mini {
      padding: 5px 9px;
      font-size: 11px;
    }

    &:hover:not(:disabled) {
      color: var(--el-color-primary);
      background: rgba(64, 158, 255, 0.08);
    }

    &.active {
      color: var(--el-color-primary);
      background: var(--el-bg-color);
      box-shadow: 0 4px 12px rgba(15, 23, 42, 0.06);
    }

    &.disabled {
      opacity: 0.45;
      cursor: not-allowed;
    }
  }

  .search-form {
    :deep(.el-form-item) {
      margin-bottom: 8px;
    }

    :deep(.el-form-item__label) {
      padding-bottom: 2px;
      color: var(--el-text-color-regular);
      font-weight: 600;
      font-size: 12px;
    }

    :deep(.el-input__wrapper),
    :deep(.el-select__wrapper) {
      min-height: 30px;
    }

    :deep(.el-date-editor--datetimerange .el-range-input),
    :deep(.el-date-editor--datetimerange .el-range-separator) {
      font-size: 12px;
    }

    :deep(.el-date-editor--datetimerange .el-range-separator) {
      padding: 0 4px;
    }
  }

  .more-toggle {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    margin-top: 0;
    padding: 4px 8px;
    border-radius: 999px;
    background: var(--el-fill-color-light);
    color: var(--el-text-color-secondary);
    cursor: pointer;
    user-select: none;
    font-size: 11px;
  }

  .more-meta {
    padding: 1px 6px;
    border-radius: 999px;
    background: var(--el-color-primary-light-8);
    color: var(--el-color-primary);
  }

  .more-panel {
    padding-top: 6px;
    border-top: 1px solid var(--el-border-color-lighter);
    margin-top: 6px;
  }

  .code-switch-row {
    align-items: center;
    margin-top: 6px;
    padding-top: 8px;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .label-section,
  .grafana-panel,
  .query-summary-row.compact-summary {
    margin-top: 6px;
    padding-top: 6px;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .compact-head {
    margin-bottom: 6px;
  }

  .empty-hint {
    color: var(--el-text-color-placeholder);
    font-size: 12px;
    padding: 4px 0 0;
  }

  .grafana-panel {
    padding: 10px 12px;
    border-radius: 12px;
    background: linear-gradient(180deg, #101827 0%, #0b1220 100%);
    border: 1px solid rgba(148, 163, 184, 0.18);
  }

  .grafana-head {
    align-items: center;
    margin-bottom: 6px;

    .section-title,
    .section-subtitle {
      color: #e5e7eb;
    }

    .section-subtitle {
      opacity: 0.72;
    }
  }

  .mode-hint-row {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    align-items: center;
    margin-bottom: 6px;
  }

  .mode-hint {
    padding: 4px 8px;
    border-radius: 999px;
    background: rgba(148, 163, 184, 0.16);
    color: #e5e7eb;
    font-size: 12px;

    &.muted {
      color: #b6c2d9;
    }
  }

  .grafana-editor {
    :deep(.el-textarea__inner) {
      min-height: 96px;
      background: #0f172a;
      color: #e5e7eb;
      border: 1px solid rgba(148, 163, 184, 0.18);
      border-radius: 10px;
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
      font-size: 13px;
      line-height: 1.65;
      box-shadow: none;
    }
  }

  .snippet-tag {
    cursor: pointer;
  }

  .bottom-bar {
    align-items: center;
    margin-top: 6px;
    padding-top: 6px;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .active-chip-row {
    margin-top: 8px;
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .active-chip {
    border-radius: 999px;
  }

  .query-summary {
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
  }

  .summary-label {
    flex: none;
    font-size: 11px;
    color: var(--el-text-color-placeholder);
  }

  .summary-text {
    color: var(--el-text-color-regular);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 11px;
  }

  .log-table {
    margin-top: 6px;

    :deep(.el-table__row) {
      cursor: pointer;
    }

    :deep(.el-table__cell) {
      padding-top: 8px;
      padding-bottom: 8px;
    }
  }

  .time-cell {
    display: inline-block;
    font-size: 12px;
    font-weight: 500;
    color: var(--el-text-color-secondary);
    white-space: nowrap;
  }

  .table-empty {
    color: var(--el-text-color-placeholder);
    font-size: 12px;
  }

  .backend-cell {
    display: grid;
    gap: 3px;
  }

  .backend-meta {
    font-size: 12px;
    color: var(--el-text-color-placeholder);
  }

  .message-cell {
    font-size: 13px;
    color: var(--el-text-color-primary);
    line-height: 1.5;
    display: -webkit-box;
    overflow: hidden;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
    word-break: break-word;
  }

  :deep(.message-highlight) {
    background: rgba(255, 208, 75, 0.55);
    color: inherit;
    padding: 0 2px;
    border-radius: 4px;
  }

  .message-wrapper {
    display: grid;
    gap: 6px;
  }

  .message-meta-row {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }

  .message-cell.expanded {
    display: block;
    -webkit-line-clamp: unset;
    white-space: pre-wrap;
  }

  .operation-cell {
    display: flex;
    align-items: center;
    justify-content: flex-start;
    gap: 4px;
  }

  .more-action {
    display: inline-block;
    cursor: pointer;
    color: var(--el-color-primary);
    font-size: 12px;
    padding: 0 4px;
  }

  .log-time {
    font-size: 12px;
    font-weight: 500;
    color: var(--el-text-color-secondary);
  }

  .log-backend {
    display: inline-flex;
    align-items: center;
    width: fit-content;
    padding: 2px 8px;
    border-radius: 999px;
    background: var(--el-fill-color-light);
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }

  .log-labels {
    display: flex;
    flex-wrap: wrap;
    gap: 5px;
  }

  .clickable-tag {
    cursor: pointer;
  }

  .log-message,
  .log-raw {
    margin: 0;
    white-space: pre-wrap;
    word-break: break-word;
    border-radius: 10px;
    padding: 10px 12px;
    line-height: 1.6;
  }

  .log-message {
    background: var(--el-bg-color);
    color: var(--el-text-color-primary);
    font-size: 13px;
  }

  .log-raw {
    margin-top: 8px;
    background: rgba(15, 23, 42, 0.04);
    color: var(--el-text-color-secondary);
    font-size: 12px;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  }

  .empty-state {
    padding: 14px 0 8px;
  }

  .label-editor {
    display: grid;
    gap: 12px;
  }

  .detail-meta {
    display: flex;
    gap: 10px;
    margin-bottom: 8px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }

  .detail-labels {
    margin-bottom: 10px;
  }

  .detail-block + .detail-block {
    margin-top: 10px;
  }

  .detail-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 6px;
    color: var(--el-text-color-primary);
    font-weight: 600;
  }

  .label-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 112px minmax(0, 1fr) auto;
    gap: 8px;
    align-items: center;
  }

  .operator-select {
    width: 112px;
  }

  .add-label-btn {
    justify-self: flex-start;
  }

  @media (max-width: 768px) {
    .log-search-page {
      padding: 8px;
    }

    .page-header,
    .search-card,
    .result-card {
      padding: 10px;
    }

    .page-header,
    .card-topbar,
    .card-head,
    .grafana-head,
    .bottom-bar,
    .aux-row,
    .log-item-head,
    .code-switch-row {
      flex-direction: column;
      align-items: stretch;
    }

    .label-row {
      grid-template-columns: 1fr;
    }
  }
</style>
