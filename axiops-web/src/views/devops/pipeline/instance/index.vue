<template>
  <div
    class="devops-pipeline-instance-page"
    :class="{
      'editor-active': isEditorMode,
      'records-active': isRecordsMode,
      'process-active': isProcessMode
    }"
  >
    <template v-if="editorMode === 'browse'">
      <ArtSearchBar
        v-show="showSearchBar"
        v-model="searchForm"
        :items="searchItems"
        @search="handleSearch"
        @reset="handleReset"
      />

      <section class="pipeline-content">
        <div class="pipeline-main-panel">
          <ElCard class="art-table-card pipeline-table-card" shadow="never">
            <ArtTableHeader
              v-model:columns="columns"
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
                <ElButton type="primary" :icon="Plus" @click="openCreateMainDialog">新增</ElButton>
              </template>
            </ArtTableHeader>

            <ArtTable
              :loading="loading"
              :data="pipelines"
              :columns="columns"
              :pagination="pagination"
              :pagination-options="{ align: 'center' }"
              empty-text="暂无流水线"
              @pagination:size-change="handleSizeChange"
              @pagination:current-change="handleCurrentChange"
            />
          </ElCard>
        </div>
      </section>
    </template>

    <section v-else-if="isRecordsMode" class="pipeline-records-workbench">
      <div class="records-card">
        <div class="records-head">
          <ElButton @click="exitRunRecords">返回</ElButton>
          <div class="records-title">
            <strong>执行记录</strong>
            <span>
              {{ currentRunPipeline?.name || '-' }}
              <template v-if="currentRunPipeline?.environmentName">
                / {{ currentRunPipeline.environmentName }}
              </template>
            </span>
          </div>
          <div class="records-actions">
            <ElButton :loading="runLoading" @click="fetchRuns">刷新</ElButton>
          </div>
        </div>
        <ArtTable
          class="records-table"
          :loading="runLoading"
          :data="runs"
          :columns="runColumns"
          :pagination="runPagination"
          :pagination-options="{ align: 'center', layout: 'total, prev, pager, next' }"
          :show-table-header="false"
          empty-text="暂无执行记录"
          @pagination:size-change="handleRunSizeChange"
          @pagination:current-change="handleRunCurrentChange"
        />
      </div>
    </section>

    <section v-else-if="isProcessMode" class="pipeline-process-workbench">
      <div class="process-graph-panel">
        <div class="process-graph-head">
          <div class="process-graph-title">
            <ElButton @click="exitRunProcess">返回</ElButton>
            <div class="process-title">
              <strong>
                {{ currentRun?.pipelineName || currentRunPipeline?.name || '执行过程' }}
                <template v-if="currentRun?.engineType === 'tekton' && currentRun?.buildId">
                  #{{ currentRun.buildId }}
                </template>
                <template v-else-if="currentRun?.jenkinsBuildNumber">
                  #{{ currentRun.jenkinsBuildNumber }}
                </template>
              </strong>
              <span>
                Graph / {{ displayRunStages.length }} 个阶段
                <template v-if="parallelGroupCount > 0">
                  / {{ parallelGroupCount }} 个并行组
                </template>
                / {{ runStatusText(currentRun?.status) }}
                <template v-if="currentRun?.triggerUsername">
                  / {{ currentRun.triggerUsername }}
                </template>
              </span>
            </div>
          </div>
          <div class="process-actions">
            <ElButton :loading="logLoading || inputLoading" @click="refreshGraphWorkbench">
              刷新
            </ElButton>
            <ElButton @click="openProcessDetail('full')">全量日志</ElButton>
            <ElButton
              v-if="hasInputWaitState"
              type="primary"
              plain
              @click="openProcessDetail('input')"
            >
              确认操作
            </ElButton>
            <ElButton @click="openProcessDetail('artifact')">制品归档</ElButton>
            <ElButton @click="openProcessDetail('quality')">质量报告</ElButton>
            <ElButton @click="copyRunLog">复制日志</ElButton>
            <ElButton @click="downloadRunLog">下载日志</ElButton>
            <ElButton
              v-if="currentRun?.jenkinsBuildUrl"
              @click="openJenkinsBuild(currentRun.jenkinsBuildUrl)"
            >
              Jenkins
            </ElButton>
            <ElButton
              v-if="currentRun && isRunActive(currentRun.status)"
              type="danger"
              :loading="runActionLoading"
              @click="stopRun(currentRun.id)"
            >
              停止
            </ElButton>
            <ElButton
              v-else-if="currentRun"
              type="danger"
              plain
              :loading="runActionLoading"
              @click="deleteRun(currentRun)"
            >
              删除
            </ElButton>
          </div>
        </div>
        <div ref="processGraphScrollRef" class="process-graph-scroll">
          <div
            v-if="processStatusHint"
            class="process-status-banner"
            :class="processStatusHint.type"
          >
            <strong>{{ processStatusHint.title }}</strong>
            <span>{{ processStatusHint.description }}</span>
          </div>
          <div v-if="processStateLoading" class="process-status-overlay">
            <div class="process-status-card">
              <strong>正在同步执行状态</strong>
              <span>{{
                processStatusHint?.description || '正在获取构建号、阶段和日志信息，请稍候。'
              }}</span>
            </div>
          </div>
          <VueFlow
            v-if="displayRunStages.length > 0"
            :nodes="processGraphNodes"
            :edges="processGraphEdges"
            :style="processGraphStyle"
            :nodes-draggable="false"
            :nodes-connectable="false"
            :elements-selectable="false"
            :default-viewport="processGraphInitialViewport"
            :fit-view-on-init="false"
            :min-zoom="0.72"
            :max-zoom="1.28"
            :pan-on-drag="true"
            :pan-on-scroll="true"
            :zoom-on-scroll="false"
            :zoom-on-pinch="false"
            :zoom-on-double-click="false"
            :only-render-visible-elements="true"
            class="process-flow"
            @init="handleProcessGraphInit"
            @node-click="handleProcessNodeClick"
          >
            <template #node-processGroup="{ data }">
              <div
                class="process-parallel-group"
                :class="[`is-${data.status}`, { 'is-post': data.rootType === 'post' }]"
              >
                <span>{{ data.name }}</span>
              </div>
            </template>
            <template #node-processStage="{ data }">
              <div class="process-stage-node-wrap">
                <Handle type="target" :position="Position.Left" :connectable="false" />
                <button
                  type="button"
                  class="process-stage-node"
                  :class="[
                    `is-${data.status}`,
                    {
                      active: data.stage.id === currentStageId,
                      'is-parallel': data.parallelGroup,
                      'needs-input': data.needsInput,
                      'is-post': data.isPost,
                      'is-disabled': !isStageLogAvailable(data.stage) && !data.needsInput
                    }
                  ]"
                  :aria-label="processStageNodeTitle(data)"
                  @click.stop="selectStage(data.stage)"
                >
                  <span class="stage-node-dot">{{ data.symbol }}</span>
                  <span class="stage-node-name">{{ data.stage.stageName }}</span>
                  <span class="stage-node-duration">{{ data.duration }}</span>
                  <span v-if="data.needsInput" class="stage-node-branch is-input">待确认</span>
                  <span v-else-if="data.parallelGroup" class="stage-node-branch">
                    {{ data.parallelGroup }}
                  </span>
                </button>
                <Handle type="source" :position="Position.Right" :connectable="false" />
              </div>
            </template>
          </VueFlow>
          <div v-if="displayRunStages.length > 0" class="process-graph-tools">
            <button type="button" title="放大" @click="zoomProcessGraphIn">+</button>
            <button type="button" title="缩小" @click="zoomProcessGraphOut">-</button>
            <button type="button" title="居中" @click="resetProcessGraphView">居中</button>
          </div>
          <ElEmpty
            v-if="displayRunStages.length === 0"
            class="process-graph-empty"
            description="暂无阶段"
          />
        </div>
      </div>
    </section>

    <section v-else class="pipeline-editor-workbench">
      <div v-if="isTektonDetailView" class="tekton-detail-shell">
        <div class="tekton-detail-head">
          <div class="tekton-detail-title">
            <ElButton @click="exitEditor">返回</ElButton>
            <div>
              <strong>{{ currentPipeline?.name || formData.name || 'Tekton Pipeline' }}</strong>
              <span>
                {{ tektonDagConfig.pipeline.namespace || '-' }} /
                {{ tektonDagConfig.pipeline.name || '-' }}
              </span>
            </div>
          </div>
          <div class="tekton-detail-actions">
            <ElDropdown
              trigger="click"
              @command="(command) => handleTektonAdvancedCommand(command, true)"
            >
              <ElButton>
                PipelineRun 高级配置
                <ElIcon class="el-icon--right"><ArrowDown /></ElIcon>
              </ElButton>
              <template #dropdown>
                <ElDropdownMenu>
                  <ElDropdownItem command="trigger">Trigger/EventListener</ElDropdownItem>
                  <ElDropdownItem command="schedule">定时触发</ElDropdownItem>
                  <ElDropdownItem command="runPolicy">PipelineRun 配置</ElDropdownItem>
                  <ElDropdownItem command="pruner">Pruner 清理</ElDropdownItem>
                  <ElDropdownItem command="yaml">资源预览</ElDropdownItem>
                </ElDropdownMenu>
              </template>
            </ElDropdown>
            <ElButton type="success" :disabled="!currentPipeline" @click="runCurrentPipeline">
              运行
            </ElButton>
            <ElButton type="primary" :disabled="!currentPipeline" @click="editCurrentPipeline">
              编辑
            </ElButton>
          </div>
        </div>

        <ElTabs
          v-model="tektonDetailActiveTab"
          class="tekton-detail-tabs"
          @tab-change="handleTektonDetailTabChange"
        >
          <ElTabPane label="概览" name="overview">
            <section class="tekton-overview-grid">
              <div class="tekton-overview-item">
                <span>引擎</span>
                <strong>Tekton</strong>
              </div>
              <div class="tekton-overview-item">
                <span>Task</span>
                <strong>{{ tektonDagSummary.taskCount }}</strong>
              </div>
              <div class="tekton-overview-item">
                <span>Finally</span>
                <strong>{{ tektonDagSummary.finallyCount }}</strong>
              </div>
              <div class="tekton-overview-item">
                <span>Matrix</span>
                <strong>{{ tektonDagSummary.matrixCount }}</strong>
              </div>
              <div class="tekton-overview-item">
                <span>Workspace</span>
                <strong>{{ tektonDagSummary.workspaceCount }}</strong>
              </div>
              <div class="tekton-overview-item">
                <span>运行配置</span>
                <strong>{{ tektonDagSummary.runtimeParamCount }}</strong>
              </div>
              <div class="tekton-overview-item">
                <span>最近运行</span>
                <strong>{{ runStatusText(currentPipeline?.lastRunStatus) }}</strong>
              </div>
              <div class="tekton-overview-item">
                <span>同步状态</span>
                <strong>{{ syncStatusText(currentPipeline?.syncStatus) }}</strong>
              </div>
            </section>
            <section class="tekton-detail-section">
              <div class="tekton-section-head">
                <strong>运行资源</strong>
              </div>
              <div class="tekton-info-grid">
                <span>Namespace</span>
                <strong>{{ tektonDagConfig.pipeline.namespace || '-' }}</strong>
                <span>Pipeline</span>
                <strong>{{ tektonDagConfig.pipeline.name || '-' }}</strong>
                <span>ServiceAccount</span>
                <strong>{{ tektonDagConfig.pipeline.serviceAccountName || '默认' }}</strong>
                <span>Timeout</span>
                <strong>{{ tektonDagConfig.pipeline.timeout || '-' }}</strong>
              </div>
            </section>
          </ElTabPane>
          <ElTabPane label="DAG" name="dag">
            <div class="tekton-detail-dag">
              <DevopsTektonDagDesigner
                :key="`detail-${workflowCanvasKey}`"
                v-model="tektonDagConfig"
                v-model:category-id="stepFilter.categoryId"
                :library-items="tektonDagLibraryItems"
                :category-options="tektonCategoryOptions"
                :library-loading="stepLoading"
                :service-account-options="tektonServiceAccountOptions"
                :storage-class-options="tektonStorageClassOptions"
                :secret-options="tektonSecretOptions"
                :pvc-options="tektonPvcOptions"
                :config-map-options="tektonConfigMapOptions"
                readonly
                @refresh-library="refreshTektonTaskLibrary"
              />
            </div>
          </ElTabPane>
          <ElTabPane label="执行记录" name="records">
            <ArtTable
              class="records-table"
              :loading="runLoading"
              :data="runs"
              :columns="runColumns"
              :pagination="runPagination"
              :pagination-options="{ align: 'center', layout: 'total, prev, pager, next' }"
              empty-text="暂无执行记录"
              @pagination:size-change="handleRunSizeChange"
              @pagination:current-change="handleRunCurrentChange"
            />
          </ElTabPane>
          <ElTabPane label="Trigger / 定时" name="trigger">
            <section class="tekton-detail-section">
              <div class="tekton-section-head">
                <strong>EventListener</strong>
                <ElButton link type="primary" @click="openTektonConfigDialog('trigger', true)">
                  配置
                </ElButton>
              </div>
              <div class="tekton-info-grid">
                <span>状态</span>
                <strong>{{ tektonDagConfig.triggerConfig?.enabled ? '启用' : '关闭' }}</strong>
                <span>Provider</span>
                <strong>{{ tektonDagConfig.triggerConfig?.provider || '-' }}</strong>
                <span>EventListener</span>
                <strong>{{ tektonDagConfig.triggerConfig?.eventListenerName || '-' }}</strong>
                <span>Trigger</span>
                <strong>{{ tektonDagConfig.triggerConfig?.triggerName || '-' }}</strong>
                <span>Webhook</span>
                <strong>{{ tektonWebhookUrl }}</strong>
              </div>
            </section>
            <section class="tekton-detail-section">
              <div class="tekton-section-head">
                <strong>定时触发</strong>
                <ElButton link type="primary" @click="openTektonConfigDialog('schedule', true)">
                  配置
                </ElButton>
              </div>
              <div class="tekton-info-grid">
                <span>状态</span>
                <strong>{{ tektonDagConfig.scheduleConfig?.enabled ? '启用' : '关闭' }}</strong>
                <span>Cron</span>
                <strong>{{ tektonDagConfig.scheduleConfig?.cron || '-' }}</strong>
                <span>时区</span>
                <strong>{{ tektonDagConfig.scheduleConfig?.timezone || '-' }}</strong>
                <span>并发策略</span>
                <strong>{{
                  tektonConcurrencyPolicyText(tektonDagConfig.scheduleConfig?.concurrencyPolicy)
                }}</strong>
              </div>
            </section>
          </ElTabPane>
          <ElTabPane label="PipelineRun" name="runPolicy">
            <section class="tekton-detail-section">
              <div class="tekton-section-head">
                <strong>PipelineRun</strong>
                <ElButton link type="primary" @click="openTektonConfigDialog('runPolicy', true)">
                  配置
                </ElButton>
              </div>
              <div class="tekton-info-grid">
                <span>Workspace 策略</span>
                <strong>{{
                  tektonWorkspaceStrategyText(tektonDagConfig.runPolicy?.workspaceStrategy)
                }}</strong>
                <span>并发策略</span>
                <strong>{{
                  tektonConcurrencyPolicyText(tektonDagConfig.runPolicy?.concurrencyPolicy)
                }}</strong>
                <span>执行前清理</span>
                <strong>{{ tektonDagConfig.runPolicy?.cleanBeforeRun ? '开启' : '关闭' }}</strong>
                <span>执行后清理</span>
                <strong>{{ tektonDagConfig.runPolicy?.cleanAfterRun ? '开启' : '关闭' }}</strong>
                <span>Pipeline Timeout</span>
                <strong>{{ tektonDagConfig.runPolicy?.pipelineTimeout || '-' }}</strong>
                <span>Tasks Timeout</span>
                <strong>{{ tektonDagConfig.runPolicy?.tasksTimeout || '-' }}</strong>
              </div>
            </section>
          </ElTabPane>
          <ElTabPane label="清理策略" name="pruner">
            <section class="tekton-detail-section">
              <div class="tekton-section-head">
                <strong>清理策略</strong>
                <ElButton link type="primary" @click="openTektonConfigDialog('pruner', true)">
                  配置
                </ElButton>
              </div>
              <div class="tekton-info-grid">
                <span>状态</span>
                <strong>{{ tektonDagConfig.prunerPolicy?.enabled ? '启用' : '关闭' }}</strong>
                <span>总历史</span>
                <strong>{{ tektonDagConfig.prunerPolicy?.historyLimit ?? '-' }}</strong>
                <span>成功历史</span>
                <strong>{{ tektonDagConfig.prunerPolicy?.successfulHistoryLimit ?? '-' }}</strong>
                <span>失败历史</span>
                <strong>{{ tektonDagConfig.prunerPolicy?.failedHistoryLimit ?? '-' }}</strong>
                <span>TTL</span>
                <strong>{{ tektonDagConfig.prunerPolicy?.ttlSecondsAfterFinished ?? '-' }}</strong>
              </div>
            </section>
          </ElTabPane>
          <ElTabPane label="审计" name="audit">
            <section class="tekton-detail-section">
              <div class="tekton-info-grid">
                <span>创建人</span>
                <strong>{{ currentPipeline?.createdBy || '-' }}</strong>
                <span>更新人</span>
                <strong>{{ currentPipeline?.updatedBy || '-' }}</strong>
                <span>创建时间</span>
                <strong>{{ formatTime(currentPipeline?.createdAt) }}</strong>
                <span>更新时间</span>
                <strong>{{ formatTime(currentPipeline?.updatedAt) }}</strong>
              </div>
            </section>
          </ElTabPane>
        </ElTabs>
      </div>

      <div v-else class="editor-workflow-board">
        <DevopsWorkflowDesigner
          v-if="formData.engineType !== 'tekton'"
          :key="workflowCanvasKey"
          ref="workflowDesignerRef"
          v-model="currentSteps"
          v-model:category-id="stepFilter.categoryId"
          :library-items="workflowLibraryItems"
          :template-items="workflowTemplateItems"
          :category-options="categoryOptions"
          :post-category-id="postCategoryId"
          :library-loading="stepLoading"
          :engine-type="formData.engineType"
          empty-text="暂无可用步骤"
          :readonly="isViewMode"
          :container-options="agentContainerOptions"
          :container-editable="agentContainerEditable"
          :enable-post-entry="formData.engineType === 'jenkins'"
          :auto-fit-on-mount="false"
          :dynamic-project-id="formData.projectId || projectStore.selectedProjectId"
          :dynamic-system-id="formData.systemId"
          :dynamic-environment-id="formData.environmentId"
          :dynamic-build-channel-binding-id="formData.buildChannelBindingId"
          @refresh-library="fetchStepLibrary"
          @save="savePipeline"
        >
          <template #left-actions>
            <ElButton @click="isViewMode ? exitEditor() : handleEditorCancel()">返回</ElButton>
          </template>
          <template #actions>
            <ElDropdown trigger="click" @command="handleEditorConfigCommand">
              <ElButton>
                配置
                <ElIcon class="el-icon--right"><ArrowDown /></ElIcon>
              </ElButton>
              <template #dropdown>
                <ElDropdownMenu>
                  <ElDropdownItem command="main">主配置</ElDropdownItem>
                  <ElDropdownItem v-if="formData.engineType === 'jenkins'" command="jenkins">
                    Jenkins 配置
                  </ElDropdownItem>
                  <ElDropdownItem command="params">参数配置</ElDropdownItem>
                  <ElDropdownItem command="trigger">定时配置</ElDropdownItem>
                </ElDropdownMenu>
              </template>
            </ElDropdown>
            <ElButton
              v-if="!isViewMode"
              type="primary"
              :loading="submitLoading"
              @click="savePipeline"
            >
              保存
            </ElButton>
          </template>
        </DevopsWorkflowDesigner>
        <div v-else class="tekton-editor-shell">
          <div class="tekton-editor-head">
            <div class="tekton-editor-title">
              <ElButton @click="handleEditorCancel">返回</ElButton>
              <div>
                <strong>{{ formData.name || 'Tekton Pipeline' }}</strong>
                <span>
                  {{ tektonDagConfig.pipeline.namespace || '-' }} /
                  {{ tektonDagConfig.pipeline.name || '-' }}
                </span>
              </div>
            </div>
            <div class="tekton-editor-actions">
              <ElButton @click="openMainConfigInEditor">平台配置</ElButton>
              <ElButton @click="openTektonWorkspaceConfig">Workspace 配置</ElButton>
              <ElDropdown trigger="click" @command="handleTektonAdvancedCommand">
                <ElButton>
                  PipelineRun 高级配置
                  <ElIcon class="el-icon--right"><ArrowDown /></ElIcon>
                </ElButton>
                <template #dropdown>
                  <ElDropdownMenu>
                    <ElDropdownItem command="trigger">Trigger/EventListener</ElDropdownItem>
                    <ElDropdownItem command="schedule">定时触发</ElDropdownItem>
                    <ElDropdownItem command="runPolicy">PipelineRun 配置</ElDropdownItem>
                    <ElDropdownItem command="pruner">Pruner 清理</ElDropdownItem>
                    <ElDropdownItem command="yaml">资源预览</ElDropdownItem>
                  </ElDropdownMenu>
                </template>
              </ElDropdown>
              <ElButton type="primary" :loading="submitLoading" @click="savePipeline">
                保存
              </ElButton>
            </div>
          </div>
          <DevopsTektonDagDesigner
            :key="workflowCanvasKey"
            v-model="tektonDagConfig"
            v-model:category-id="stepFilter.categoryId"
            :library-items="tektonDagLibraryItems"
            :category-options="tektonCategoryOptions"
            :library-loading="stepLoading"
            :service-account-options="tektonServiceAccountOptions"
            :storage-class-options="tektonStorageClassOptions"
            :secret-options="tektonSecretOptions"
            :pvc-options="tektonPvcOptions"
            :config-map-options="tektonConfigMapOptions"
            :readonly="isViewMode || isTektonTemplateLocked"
            @refresh-library="refreshTektonTaskLibrary"
            @save="savePipeline"
          />
        </div>
      </div>
    </section>

    <ElDialog
      v-model="mainConfigVisible"
      :title="mainConfigTitle"
      width="860px"
      align-center
      class="pipeline-config-dialog main-config-dialog"
      :close-on-click-modal="false"
    >
      <div
        v-loading="mainConfigLoading"
        element-loading-text="加载中..."
        class="main-config-content"
      >
        <ElAlert
          v-if="systems.length === 0"
          title="当前项目还没有应用，请先到应用管理创建应用"
          type="warning"
          :closable="false"
          class="dialog-alert"
        />
        <ElForm
          ref="mainFormRef"
          :model="formData"
          :rules="rules"
          label-width="96px"
          :disabled="mainConfigReadonly || mainConfigLoading"
        >
          <div class="dialog-form-grid">
            <ElFormItem label="所属项目">
              <ElInput :model-value="projectStore.selectedProject?.name || '请选择项目'" disabled />
            </ElFormItem>
            <ElFormItem label="应用" prop="systemId">
              <ElSelect
                v-model="formData.systemId"
                placeholder="请选择应用"
                filterable
                :disabled="mainConfigLocked"
                style="width: 100%"
                @visible-change="(visible: boolean) => visible && fetchSystems()"
              >
                <ElOption
                  v-for="item in systems"
                  :key="item.id"
                  :label="`${item.name}（${item.code}）`"
                  :value="item.id"
                />
              </ElSelect>
            </ElFormItem>
            <ElFormItem label="名称" prop="name">
              <ElInput
                v-model="formData.name"
                :disabled="mainConfigLocked"
                placeholder="请输入名称"
              />
            </ElFormItem>
            <ElFormItem label="编码" prop="code">
              <ElInput
                v-model="formData.code"
                :disabled="
                  mainConfigLocked || (editorMode !== 'add' && mainConfigContext !== 'create')
                "
                placeholder="例如 build_deploy"
              />
            </ElFormItem>
            <ElFormItem label="归属环境" prop="environmentId">
              <ElSelect
                v-model="formData.environmentId"
                placeholder="请选择环境"
                filterable
                :disabled="mainConfigLocked"
                style="width: 100%"
                @visible-change="(visible: boolean) => visible && fetchEnvironments()"
              >
                <ElOption
                  v-for="item in environments"
                  :key="item.id"
                  :label="`${item.name}（${item.code}）`"
                  :value="item.id"
                />
              </ElSelect>
            </ElFormItem>
            <ElFormItem label="引擎" prop="engineType">
              <ElSelect
                v-model="formData.engineType"
                :disabled="
                  mainConfigLocked || (editorMode !== 'add' && mainConfigContext !== 'create')
                "
                style="width: 100%"
                @change="handleEngineChange"
              >
                <ElOption label="Jenkins" value="jenkins" />
                <ElOption label="Tekton" value="tekton" />
              </ElSelect>
            </ElFormItem>
            <ElFormItem label="构建渠道" prop="buildChannelBindingId">
              <ElSelect
                v-model="formData.buildChannelBindingId"
                :placeholder="`请选择 ${engineTypeName(formData.engineType)} 构建渠道`"
                filterable
                style="width: 100%"
                @visible-change="(visible: boolean) => visible && fetchBuildBindings()"
                @change="handleBuildChannelChange"
              >
                <ElOption
                  v-for="item in buildBindings"
                  :key="item.id"
                  :label="buildBindingLabel(item)"
                  :value="item.id"
                />
              </ElSelect>
            </ElFormItem>
            <div
              v-if="formData.engineType === 'tekton' && selectedBuildBinding"
              class="form-grid-wide tekton-runtime-panel"
            >
              <div class="tekton-runtime-item">
                <span>Namespace</span>
                <strong>{{ selectedTektonBindingConfig.namespace || '-' }}</strong>
              </div>
              <div class="tekton-runtime-item">
                <span>Task 命名空间</span>
                <strong>{{ selectedTektonChannelConfig.taskNamespace || '-' }}</strong>
              </div>
              <div class="tekton-runtime-item">
                <span>ServiceAccount</span>
                <strong>{{ selectedTektonBindingConfig.serviceAccountName || '默认' }}</strong>
              </div>
            </div>
            <ElFormItem label="状态">
              <ElRadioGroup v-model="formData.status" :disabled="mainConfigLocked">
                <ElRadio :value="1">启用</ElRadio>
                <ElRadio :value="0">停用</ElRadio>
              </ElRadioGroup>
            </ElFormItem>
          </div>
        </ElForm>
      </div>
      <template #footer>
        <ElButton @click="mainConfigVisible = false">取消</ElButton>
        <ElButton
          v-if="!mainConfigReadonly"
          type="primary"
          :loading="submitLoading"
          :disabled="mainConfigLoading"
          @click="submitMainConfig"
        >
          保存
        </ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="templateSourceVisible"
      title="选择流水线来源"
      width="820px"
      align-center
      class="pipeline-config-dialog template-source-dialog"
      :close-on-click-modal="false"
    >
      <div class="template-source-content">
        <div class="template-source-head">
          <div>
            <strong>创建来源</strong>
            <span>从空白流程或模板快照创建流水线。</span>
          </div>
          <ElButton :icon="RefreshRight" :loading="templateLoading" @click="fetchTemplates">
            刷新
          </ElButton>
        </div>

        <div class="template-source-toolbar">
          <ElInput
            v-model="templateKeyword"
            class="template-source-search"
            :prefix-icon="Search"
            clearable
            placeholder="搜索模板名称、编码、描述或步骤"
          />
          <ElSegmented
            v-model="templateSourceFilter"
            class="template-source-filter"
            :options="templateSourceFilterOptions"
          />
        </div>

        <div class="template-source-result-bar">
          <span>可用来源</span>
          <em>{{ templateSourceResultCount }} 项</em>
        </div>

        <div class="template-source-list">
          <button
            v-if="showCustomTemplateSource"
            type="button"
            class="template-source-card"
            :class="{ active: isCustomTemplateSource }"
            @click="selectTemplateSource(customTemplateSourceKey)"
          >
            <div class="template-source-card-main">
              <div class="template-source-card-head">
                <strong>自定义流水线</strong>
                <code>custom</code>
              </div>
              <div class="template-source-card-desc">从空白流程开始配置步骤和参数。</div>
            </div>
            <div class="template-source-card-meta">
              <span>Jenkins</span>
              <span>空白创建</span>
            </div>
          </button>

          <button
            v-for="item in filteredTemplateSourceItems"
            :key="item.id"
            type="button"
            class="template-source-card"
            :class="{ active: templateSourceKey === item.id }"
            @click="selectTemplateSource(item.id)"
          >
            <div class="template-source-card-main">
              <div class="template-source-card-head">
                <strong>{{ item.name }}</strong>
                <code>{{ item.code }}</code>
              </div>
              <div class="template-source-card-desc">
                {{ item.description || '暂无描述' }}
              </div>
              <div class="template-source-card-steps">
                <span
                  v-for="(step, index) in templateStepPreview(item)"
                  :key="`${item.id}-${step.name}-${index}`"
                  :title="step.name"
                >
                  {{ step.name }}
                </span>
              </div>
            </div>
            <div class="template-source-card-meta">
              <span>{{ item.steps?.length || 0 }} 步</span>
              <span>{{ item.engineType === 'tekton' ? 'Tekton' : 'Jenkins' }}</span>
              <span>{{ item.scope === 'project' ? '项目模板' : '系统模板' }}</span>
            </div>
          </button>

          <ElEmpty
            v-if="!templateLoading && templateSourceResultCount === 0"
            description="暂无匹配来源"
          />
        </div>
      </div>
      <template #footer>
        <div class="template-source-footer">
          <div class="template-source-current">
            <span>当前选择</span>
            <strong>
              {{
                isCustomTemplateSource ? '自定义流水线' : selectedTemplateChoice?.name || '未选择'
              }}
            </strong>
            <code v-if="isCustomTemplateSource || selectedTemplateChoice">
              {{ isCustomTemplateSource ? 'custom' : selectedTemplateChoice?.code }}
            </code>
          </div>
          <div class="template-source-actions">
            <ElButton @click="templateSourceVisible = false">取消</ElButton>
            <ElButton
              type="primary"
              :disabled="templateSourceConfirmDisabled"
              @click="startSelectedTemplateSource"
            >
              {{
                isCustomTemplateSource ? '开始自定义' : isTektonEngine() ? '选择模板' : '载入模板'
              }}
            </ElButton>
          </div>
        </div>
      </template>
    </ElDialog>

    <ElDialog
      v-model="jenkinsConfigVisible"
      title="Jenkins 配置"
      width="820px"
      align-center
      class="pipeline-config-dialog jenkins-config-dialog"
      :close-on-click-modal="false"
    >
      <ElForm class="config-dialog-form" label-width="96px" :disabled="isViewMode">
        <section class="config-panel">
          <div class="config-panel-head">
            <div>
              <strong>运行 Agent</strong>
              <span>节点列表来自当前 Jenkins 构建渠道。</span>
            </div>
          </div>
          <ElFormItem label="运行节点">
            <ElSelect
              :model-value="selectedAgentValue"
              placeholder="请选择运行节点"
              filterable
              :loading="agentLoading"
              style="width: 100%"
              @update:model-value="
                (value: string | number | boolean | Record<string, any>) =>
                  handleAgentSelectChange(String(value || ''))
              "
              @visible-change="(visible: boolean) => visible && fetchJenkinsAgents(true)"
            >
              <ElOption label="任意可用节点" value="any" />
              <ElOption label="不分配全局 Agent" value="none" />
              <ElOption
                v-for="item in jenkinsAgents"
                :key="item.id"
                :label="`${item.name || item.displayName}（${agentTypeName(item.agentType)}）`"
                :value="item.id"
              />
            </ElSelect>
          </ElFormItem>
          <div v-if="selectedPlatformAgent" class="agent-preview-card">
            <div class="agent-preview-main">
              <strong>{{ selectedPlatformAgent.name }}</strong>
              <span>
                {{ agentMatchText(selectedPlatformAgent) }}
              </span>
            </div>
            <div v-if="agentContainerOptions.length > 0" class="agent-container-tags">
              <ElTag v-for="item in agentContainerOptions" :key="item" size="small">
                {{ item }}
              </ElTag>
            </div>
          </div>
        </section>

        <section class="config-panel">
          <div class="mini-list-head">
            <span>工具配置</span>
            <ElButton size="small" text type="primary" :disabled="isViewMode" @click="addTool">
              新增
            </ElButton>
          </div>
          <div class="mini-list">
            <ElEmpty v-if="formData.tools.length === 0" description="暂无工具配置" />
            <div v-for="(item, index) in formData.tools" :key="index" class="mini-row tool-row">
              <ElSelect v-model="item.type" placeholder="工具类型">
                <ElOption label="JDK" value="jdk" />
                <ElOption label="Maven" value="maven" />
                <ElOption label="Gradle" value="gradle" />
                <ElOption label="Node.js" value="nodejs" />
                <ElOption label="Go" value="go" />
              </ElSelect>
              <ElInput v-model="item.name" placeholder="Jenkins 工具名" />
              <ElInput v-model="item.version" placeholder="版本，可选" />
              <ElButton text type="danger" :disabled="isViewMode" @click="removeTool(index)">
                删除
              </ElButton>
            </div>
          </div>
        </section>

        <section class="config-panel">
          <div class="mini-list-head">
            <span>流水线选项</span>
            <ElButton size="small" text type="primary" :disabled="isViewMode" @click="resetOptions">
              恢复默认
            </ElButton>
          </div>
          <div class="option-grid">
            <div v-for="item in optionDefinitions" :key="item.name" class="option-card">
              <div class="option-main">
                <strong>{{ item.title }}</strong>
                <span>{{ item.description }}</span>
              </div>
              <ElSwitch
                :model-value="optionEnabled(item.name)"
                :disabled="isViewMode"
                @change="
                  (enabled: string | number | boolean) =>
                    setOptionEnabled(item.name, enabled === true, item.defaultValue)
                "
              />
              <ElInput
                v-if="optionEnabled(item.name) && item.valueLabel"
                :model-value="optionValue(item.name)"
                :placeholder="item.defaultValue"
                :disabled="isViewMode"
                @update:model-value="(value: string) => setOptionValue(item.name, value)"
              >
                <template #prepend>{{ item.valueLabel }}</template>
              </ElInput>
            </div>
          </div>
        </section>
      </ElForm>
      <template #footer>
        <ElButton type="primary" @click="jenkinsConfigVisible = false">确定</ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="paramConfigVisible"
      :title="isTektonEngine() ? 'Pipeline 参数配置' : '参数配置'"
      width="920px"
      align-center
      class="pipeline-config-dialog param-config-dialog"
      :close-on-click-modal="false"
    >
      <div class="config-dialog-shell">
        <div class="param-toolbar">
          <div>
            <strong>流水线参数</strong>
            <span>{{ sortedPipelineParams.length }} 个参数</span>
          </div>
          <ElButton :disabled="isViewMode" @click="syncParamsFromCurrentDesigner(true)">
            {{ formData.engineType === 'tekton' ? '同步 DAG 参数' : '同步步骤参数' }}
          </ElButton>
        </div>
        <section class="config-panel param-config-panel">
          <div class="param-list dialog-param-list">
            <ElEmpty v-if="sortedPipelineParams.length === 0" description="暂无参数" />
            <div
              v-for="item in sortedPipelineParams"
              :key="item.code"
              class="param-row"
              :class="{ 'is-full-row': isFullRowParam(item) }"
            >
              <div class="param-row-head">
                <div class="param-row-title">
                  <strong :title="item.name">{{ item.name }}</strong>
                  <span :title="item.code">{{ item.code }}</span>
                  <span v-if="pipelineParamStepName(item)" :title="pipelineParamStepName(item)">
                    {{ pipelineParamStepName(item) }}
                  </span>
                </div>
                <div class="param-tags">
                  <ElTag size="small" effect="plain">{{ paramTypeLabel(item) }}</ElTag>
                  <ElTag
                    size="small"
                    effect="plain"
                    :type="effectiveParamRuntimeMode(item) === 'env' ? 'warning' : 'success'"
                  >
                    {{
                      effectiveParamRuntimeMode(item) === 'env'
                        ? runtimeModeLabel(item, 'env')
                        : runtimeModeLabel(item, 'params')
                    }}
                  </ElTag>
                  <ElTag v-if="item.required" size="small" type="danger">必填</ElTag>
                  <ElTag v-if="item.readonly" size="small" type="info">只读</ElTag>
                  <ElTag v-if="isRuntimeConfigParam(item)" size="small" type="success">
                    运行时配置
                  </ElTag>
                </div>
              </div>
              <DevopsKubeNovaDeployConfigEditor
                v-if="isKubeNovaDeployConfigParam(item)"
                v-model="item.currentValue"
                :project-id="formData.projectId || projectStore.selectedProjectId"
                :system-id="formData.systemId"
                :environment-id="formData.environmentId"
                :channel-group-code="item.config?.channelGroupCode || item.channelGroupCode || ''"
                :channel-type="item.config?.channelTypeFilter || ''"
                :mapping-field="item.config?.mappingField || ''"
                :disabled="isViewMode || item.readonly"
                @change="() => handlePipelineConfigParamChange(item)"
              />
              <DevopsKubernetesDeployConfigEditor
                v-else-if="isKubernetesDeployConfigParam(item)"
                v-model="item.currentValue"
                :project-id="formData.projectId || projectStore.selectedProjectId"
                :system-id="formData.systemId"
                :environment-id="formData.environmentId"
                :disabled="isViewMode || item.readonly"
                @change="() => handlePipelineConfigParamChange(item)"
              />
              <ElInput
                v-else-if="item.paramType === 'text'"
                v-model="item.currentValue"
                type="textarea"
                :rows="5"
                resize="vertical"
                :disabled="isViewMode || item.readonly"
                :placeholder="item.defaultValue || '请输入参数值'"
              />
              <ElSwitch
                v-else-if="item.paramType === 'booleanParam'"
                v-model="item.currentValue"
                :disabled="isViewMode || item.readonly"
                active-value="true"
                inactive-value="false"
              />
              <ElInputNumber
                v-else-if="item.paramType === 'number'"
                :model-value="numberParamValue(item.currentValue)"
                :disabled="isViewMode || item.readonly"
                controls-position="right"
                style="width: 100%"
                placeholder="请输入数字"
                @update:model-value="
                  (value: number | undefined) => setPipelineNumberValue(item, value)
                "
              />
              <div v-else-if="item.paramType === 'objectList'" class="object-list-value-editor">
                <div class="object-list-value-head">
                  <span>{{ objectListFields(item).length }} 个字段</span>
                  <ElButton
                    size="small"
                    type="primary"
                    plain
                    :disabled="isViewMode || item.readonly || objectListFields(item).length === 0"
                    @click="addPipelineObjectListRow(item)"
                  >
                    新增一行
                  </ElButton>
                </div>
                <ElEmpty
                  v-if="objectListFields(item).length === 0"
                  description="未配置对象列表字段"
                />
                <ElTable
                  v-else
                  :data="pipelineObjectListRows(item)"
                  size="small"
                  border
                  empty-text="暂无数据"
                >
                  <ElTableColumn
                    v-for="field in objectListFields(item)"
                    :key="field.code"
                    :label="field.name || field.code"
                    :min-width="160"
                  >
                    <template #default="{ row, $index }">
                      <component
                        :is="objectListFieldComponent(field)"
                        :model-value="objectListFieldValue(row, field)"
                        v-bind="objectListFieldProps(field, isViewMode || item.readonly)"
                        @update:model-value="
                          (value: any) =>
                            updatePipelineObjectListFieldValue(item, $index, field.code, value)
                        "
                      >
                        <template v-if="objectListFieldIsSelect(field)" #default>
                          <ElOption
                            v-for="option in objectListFieldOptions(field)"
                            :key="option.value"
                            :label="option.label || option.value"
                            :value="option.value"
                          />
                        </template>
                      </component>
                    </template>
                  </ElTableColumn>
                  <ElTableColumn label="操作" width="88" fixed="right">
                    <template #default="{ $index }">
                      <ElButton
                        text
                        type="danger"
                        :disabled="isViewMode || item.readonly"
                        @click="removePipelineObjectListRow(item, $index)"
                      >
                        删除
                      </ElButton>
                    </template>
                  </ElTableColumn>
                </ElTable>
              </div>
              <ElInput
                v-else-if="item.paramType === 'file'"
                v-model="item.currentValue"
                :disabled="isViewMode || item.readonly"
                placeholder="请输入文件路径或制品地址"
              />
              <DevopsPlatformAddressPicker
                v-else-if="isCompositePipelineParam(item)"
                v-model="item.currentValue"
                :param-type="platformAddressParamType(item)"
                :project-id="formData.projectId || projectStore.selectedProjectId"
                :system-id="formData.systemId"
                :environment-id="formData.environmentId"
                :channel-group-code="item.config?.channelGroupCode || item.channelGroupCode || ''"
                :channel-type="item.config?.channelTypeFilter || ''"
                :mapping-field="item.config?.mappingField || ''"
                :disabled="isViewMode || item.readonly"
                :placeholder="item.defaultValue || ''"
                @change="() => handlePipelineConfigParamChange(item)"
              />
              <div v-else-if="isChannelCredentialPipelineParam(item)" class="derived-param-preview">
                <span>来源：{{ item.config?.credentialSourceParamCode || '-' }}</span>
                <span>{{ channelCredentialOutputLabel(item) }}</span>
              </div>
              <div v-else-if="isPipelineDynamicConfigParam(item)" class="dynamic-select-control">
                <ElSelect
                  v-model="item.currentValue"
                  class="platform-param-select"
                  popper-class="devops-platform-param-dropdown"
                  :teleported="true"
                  fit-input-width
                  filterable
                  clearable
                  allow-create
                  default-first-option
                  remote-show-suffix
                  :disabled="isViewMode || item.readonly"
                  :remote="isRemotePipelineSelectParam(item)"
                  :remote-method="
                    (keyword: string) => queryPipelineDynamicParamOptions(item, keyword)
                  "
                  :loading="pipelineConfigParamLoading[item.code]"
                  loading-text="加载中..."
                  :no-data-text="dynamicEmptyText(item.paramType)"
                  :placeholder="pipelineDynamicPlaceholder(item)"
                  @visible-change="
                    (visible: boolean) => handlePipelineDynamicParamVisible(item, visible)
                  "
                  @change="() => handlePipelineConfigParamChange(item)"
                >
                  <template #loading>
                    <div class="devops-select-loading">
                      <span class="devops-select-loading__icon"></span>
                      <span>加载中...</span>
                    </div>
                  </template>
                  <div
                    v-if="pipelineConfigParamError[item.code]"
                    class="dynamic-option-tip is-error"
                  >
                    {{ pipelineConfigParamError[item.code] }}
                  </div>
                  <div
                    v-else-if="
                      pipelineConfigParamLoaded[item.code] &&
                      !pipelineConfigParamLoading[item.code] &&
                      !(pipelineConfigParamOptions[item.code] || []).length
                    "
                    class="dynamic-option-tip"
                  >
                    {{ dynamicEmptyText(item.paramType) }}
                  </div>
                  <ElOption
                    v-for="option in pipelineConfigParamOptions[item.code] || []"
                    :key="option.value"
                    :label="option.label"
                    :value="option.value"
                  >
                    <div class="dynamic-option-item">
                      <span>{{ option.label }}</span>
                      <small v-if="dynamicOptionSecondary(option)">
                        {{ dynamicOptionSecondary(option) }}
                      </small>
                    </div>
                  </ElOption>
                </ElSelect>
                <ElButton
                  class="dynamic-select-refresh"
                  :icon="RefreshRight"
                  :loading="pipelineConfigParamLoading[item.code]"
                  :disabled="isViewMode || item.readonly"
                  title="刷新选项"
                  aria-label="刷新选项"
                  @click.stop="refreshPipelineDynamicParamOptions(item)"
                >
                  刷新
                </ElButton>
              </div>
              <ElSelect
                v-else-if="item.paramType === 'choice' || item.paramType === 'list'"
                v-model="item.currentValue"
                class="platform-param-select"
                popper-class="devops-platform-param-dropdown"
                :teleported="true"
                fit-input-width
                filterable
                clearable
                :disabled="isViewMode || item.readonly"
                allow-create
                default-first-option
                :placeholder="item.defaultValue || '请选择或输入'"
              >
                <ElOption
                  v-for="option in item.config?.options?.length
                    ? item.config.options
                    : item.selectList || []"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </ElSelect>
              <ElInput
                v-else
                v-model="item.currentValue"
                :disabled="isViewMode || item.readonly"
                :type="item.paramType === 'password' ? 'password' : 'text'"
                :show-password="item.paramType === 'password'"
                :placeholder="item.defaultValue || '请输入参数值'"
              />
            </div>
          </div>
        </section>
      </div>
      <template #footer>
        <ElButton type="primary" @click="paramConfigVisible = false">确定</ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="tektonWorkspaceConfigVisible"
      title="Workspace 配置"
      width="900px"
      align-center
      class="pipeline-config-dialog run-param-dialog"
      :close-on-click-modal="false"
    >
      <div class="run-param-shell">
        <div class="run-param-toolbar">
          <strong>项目流水线 Workspace</strong>
          <span class="run-param-summary">保存为项目级稳定值，运行时只做覆盖</span>
        </div>
        <section class="run-param-card">
          <ElForm class="run-param-form" label-position="top">
            <ElEmpty
              v-if="tektonWorkspaceConfigItems.length === 0"
              class="run-param-empty"
              description="当前模板未声明 Workspace"
            />
            <div v-else class="run-param-list">
              <div
                v-for="(item, index) in tektonWorkspaceConfigItems"
                :key="item.name"
                class="run-param-field is-full-row"
                :class="{ required: !item.optional }"
              >
                <div class="run-param-meta">
                  <label class="run-param-label" :title="item.name">
                    <span>{{ item.name }}</span>
                  </label>
                  <div class="run-param-tags">
                    <ElTag size="small" effect="plain">Workspace</ElTag>
                    <ElTag size="small" :type="item.optional ? 'info' : 'danger'">
                      {{ item.optional ? '可选' : '必填' }}
                    </ElTag>
                  </div>
                </div>
                <div class="workspace-config-summary-card">
                  <div class="workspace-config-summary-main">
                    <strong>{{ tektonWorkspaceBindingText(item.bindingType) }}</strong>
                    <span :title="projectWorkspaceResourceText(item)">
                      {{ projectWorkspaceResourceText(item) }}
                    </span>
                  </div>
                  <div class="workspace-config-summary-tags">
                    <ElTag size="small" effect="plain">
                      {{ item.runtimeConfig ? '允许运行覆盖' : '项目固定配置' }}
                    </ElTag>
                    <ElTag
                      v-for="option in projectWorkspaceBindingOptions(item)"
                      :key="option.value"
                      size="small"
                      type="info"
                      effect="plain"
                    >
                      可用 {{ option.label }}
                    </ElTag>
                  </div>
                  <ElButton size="small" @click="openTektonWorkspaceItemEditor(index)">
                    配置绑定
                  </ElButton>
                </div>
              </div>
            </div>
          </ElForm>
        </section>
      </div>
      <template #footer>
        <ElButton @click="tektonWorkspaceConfigVisible = false">取消</ElButton>
        <ElButton type="primary" @click="saveTektonWorkspaceConfig">确定</ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="tektonWorkspaceItemEditorVisible"
      title="Workspace 绑定配置"
      width="860px"
      append-to-body
      class="pipeline-config-dialog workspace-item-editor-dialog"
      :close-on-click-modal="false"
      @close="handleTektonWorkspaceItemEditorClosed"
    >
      <ElForm
        v-if="editingTektonWorkspaceItem"
        label-width="120px"
        class="workspace-item-editor-form"
      >
        <div class="workspace-item-editor-head">
          <strong>{{ editingTektonWorkspaceItem.name }}</strong>
          <span>{{
            editingTektonWorkspaceItem.description || '选择项目流水线稳定使用的 Workspace 资源'
          }}</span>
        </div>
        <div class="workspace-item-editor-grid">
          <ElFormItem label="绑定方式" required>
            <ElSelect
              v-model="editingTektonWorkspaceItem.bindingType"
              placeholder="绑定方式"
              @change="handleProjectWorkspaceBindingTypeChange(editingTektonWorkspaceItem)"
            >
              <ElOption
                v-for="option in projectWorkspaceBindingOptions(editingTektonWorkspaceItem)"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="运行时覆盖">
            <ElSwitch
              v-model="editingTektonWorkspaceItem.runtimeConfig"
              active-text="允许"
              inactive-text="不允许"
            />
          </ElFormItem>

          <ElFormItem
            v-if="editingTektonWorkspaceItem.bindingType === 'persistentVolumeClaim'"
            label="PVC"
            required
            class="workspace-item-editor-wide"
          >
            <ElSelect
              v-model="editingTektonWorkspaceItem.claimName"
              placeholder="选择 PVC"
              filterable
              clearable
              @visible-change="(visible: boolean) => visible && fetchTektonRuntimeResources()"
            >
              <ElOption
                v-for="option in tektonPvcOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem
            v-else-if="editingTektonWorkspaceItem.bindingType === 'secret'"
            label="Secret"
            required
            class="workspace-item-editor-wide"
          >
            <ElSelect
              v-model="editingTektonWorkspaceItem.secretName"
              placeholder="选择 Secret"
              filterable
              clearable
              @visible-change="(visible: boolean) => visible && fetchTektonRuntimeResources()"
            >
              <ElOption
                v-for="option in tektonSecretOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem
            v-else-if="editingTektonWorkspaceItem.bindingType === 'configMap'"
            label="ConfigMap"
            required
            class="workspace-item-editor-wide"
          >
            <ElSelect
              v-model="editingTektonWorkspaceItem.configMapName"
              placeholder="选择 ConfigMap"
              filterable
              clearable
              @visible-change="(visible: boolean) => visible && fetchTektonRuntimeResources()"
            >
              <ElOption
                v-for="option in tektonConfigMapOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </ElSelect>
          </ElFormItem>

          <template v-else-if="editingTektonWorkspaceItem.bindingType === 'volumeClaimTemplate'">
            <ElFormItem label="容量" required>
              <ElInput v-model="editingTektonWorkspaceItem.storage" placeholder="例如 10Gi" />
            </ElFormItem>
            <ElFormItem label="StorageClass">
              <ElSelect
                v-model="editingTektonWorkspaceItem.storageClassName"
                placeholder="StorageClass"
                filterable
                clearable
                @visible-change="(visible: boolean) => visible && fetchTektonRuntimeResources()"
              >
                <ElOption
                  v-for="option in tektonStorageClassOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </ElSelect>
            </ElFormItem>
            <ElFormItem label="访问模式">
              <ElSelect v-model="editingTektonWorkspaceItem.accessMode" placeholder="访问模式">
                <ElOption label="ReadWriteOnce" value="ReadWriteOnce" />
                <ElOption label="ReadOnlyMany" value="ReadOnlyMany" />
                <ElOption label="ReadWriteMany" value="ReadWriteMany" />
              </ElSelect>
            </ElFormItem>
            <ElFormItem label="卷模式">
              <ElSelect
                v-model="editingTektonWorkspaceItem.volumeMode"
                clearable
                placeholder="卷模式"
              >
                <ElOption label="Filesystem" value="Filesystem" />
                <ElOption label="Block" value="Block" />
              </ElSelect>
            </ElFormItem>
          </template>

          <template v-else-if="editingTektonWorkspaceItem.bindingType === 'generatedSecret'">
            <ElFormItem label="来源类型" required>
              <ElSelect
                v-model="editingTektonWorkspaceItem.sourceType"
                placeholder="来源类型"
                @change="handleGeneratedSecretSourceChange(editingTektonWorkspaceItem)"
              >
                <ElOption
                  v-for="option in generatedSecretSourceOptions(editingTektonWorkspaceItem)"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </ElSelect>
            </ElFormItem>
            <ElFormItem
              v-if="editingTektonWorkspaceItem.sourceType === 'configCenter'"
              label="配置中心"
              required
            >
              <ElSelect
                v-model="editingTektonWorkspaceItem.resourceId"
                placeholder="选择配置"
                filterable
                clearable
              >
                <ElOption
                  v-for="option in tektonProjectConfigOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </ElSelect>
            </ElFormItem>
            <ElFormItem v-else label="凭证" required>
              <ElSelect
                v-model="editingTektonWorkspaceItem.resourceId"
                placeholder="选择凭证"
                filterable
                clearable
              >
                <ElOption
                  v-for="option in tektonCredentialOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </ElSelect>
            </ElFormItem>
            <ElFormItem
              v-if="editingTektonWorkspaceItem.sourceType !== 'configCenter'"
              label="凭证类型"
            >
              <ElSelect v-model="editingTektonWorkspaceItem.platformType" placeholder="凭证类型">
                <ElOption
                  v-for="option in generatedSecretPlatformTypeOptions(editingTektonWorkspaceItem)"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </ElSelect>
            </ElFormItem>
          </template>

          <ElFormItem
            v-else-if="editingTektonWorkspaceItem.bindingType === 'projected'"
            label="Projected"
            required
            class="workspace-item-editor-wide"
          >
            <ElInput
              v-model="editingTektonWorkspaceItem.config.projectedYaml"
              type="textarea"
              :rows="8"
              placeholder="sources:\n  - secret:\n      name: docker-auth"
            />
          </ElFormItem>

          <ElFormItem
            v-if="editingTektonWorkspaceItem.bindingType !== 'emptyDir'"
            label="subPath"
            class="workspace-item-editor-wide"
          >
            <ElInput
              v-model="editingTektonWorkspaceItem.subPath"
              placeholder="可选，挂载到 Workspace 的子路径"
            />
          </ElFormItem>
        </div>
      </ElForm>
      <template #footer>
        <ElButton @click="cancelTektonWorkspaceItemEditor">取消</ElButton>
        <ElButton type="primary" @click="saveTektonWorkspaceItemEditor">确定</ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="runParamVisible"
      :title="`运行配置：${runTargetPipeline?.name || '流水线'}`"
      width="900px"
      class="pipeline-config-dialog run-param-dialog"
      align-center
      :close-on-click-modal="false"
      @closed="handleRunParamDialogClosed"
    >
      <div class="run-param-shell">
        <div class="run-param-toolbar">
          <strong>运行配置</strong>
          <span class="run-param-summary">
            <template v-if="runParamInitializing">正在初始化...</template>
            <template v-else>
              参数 {{ runParamItems.length }} 个 / Workspace {{ runWorkspaceItems.length }} 个
            </template>
          </span>
        </div>

        <section
          v-loading="runParamInitializing"
          class="run-param-card"
          element-loading-text="正在初始化流水线参数表单..."
        >
          <div v-if="runParamInitializing" aria-hidden="true" />
          <div v-else-if="runParamInitError" class="run-param-init-error">
            <strong>初始化流水线参数表单失败</strong>
            <span>{{ runParamInitError }}</span>
            <ElButton
              v-if="runTargetPipeline"
              size="small"
              type="primary"
              plain
              @click="runPipeline(runTargetPipeline)"
            >
              重新初始化
            </ElButton>
          </div>
          <ElForm v-else class="run-param-form" label-position="top">
            <ElEmpty
              v-if="runParamItems.length === 0 && runWorkspaceItems.length === 0"
              class="run-param-empty"
              description="当前流水线无运行配置"
            />
            <div v-else class="run-param-list">
              <div v-if="runParamItems.length > 0" class="run-param-section-title">
                Pipeline 参数
              </div>
              <div
                v-for="item in runParamItems"
                :key="item.code"
                class="run-param-field"
                :class="{ required: item.required, 'is-full-row': isFullRowParam(item) }"
              >
                <div class="run-param-meta">
                  <label class="run-param-label" :title="item.name">
                    <span>{{ item.name }}</span>
                  </label>
                  <div class="run-param-tags">
                    <ElTag size="small" effect="plain">{{ paramTypeLabel(item) }}</ElTag>
                    <ElTag
                      size="small"
                      :type="effectiveParamRuntimeMode(item) === 'env' ? 'warning' : 'success'"
                    >
                      {{
                        effectiveParamRuntimeMode(item) === 'env'
                          ? runtimeModeLabel(item, 'env')
                          : runtimeModeLabel(item, 'params')
                      }}
                    </ElTag>
                  </div>
                </div>
                <div class="run-param-code" :title="item.code">{{ item.code }}</div>
                <ElFormItem class="run-param-control">
                  <DevopsKubeNovaDeployConfigEditor
                    v-if="isKubeNovaDeployConfigParam(item)"
                    v-model="runParamForm[item.code]"
                    :project-id="runTargetPipeline?.projectId || projectStore.selectedProjectId"
                    :system-id="runTargetPipeline?.systemId || ''"
                    :environment-id="runTargetPipeline?.environmentId || ''"
                    @change="() => handleRunParamChange(item)"
                  />
                  <DevopsKubernetesDeployConfigEditor
                    v-else-if="isKubernetesDeployConfigParam(item)"
                    v-model="runParamForm[item.code]"
                    :project-id="runTargetPipeline?.projectId || projectStore.selectedProjectId"
                    :system-id="runTargetPipeline?.systemId || ''"
                    :environment-id="runTargetPipeline?.environmentId || ''"
                    @change="() => handleRunParamChange(item)"
                  />
                  <ElInput
                    v-else-if="item.paramType === 'text'"
                    v-model="runParamForm[item.code]"
                    type="textarea"
                    :rows="5"
                    resize="vertical"
                    :placeholder="item.defaultValue || '请输入参数值'"
                  />
                  <ElSwitch
                    v-else-if="item.paramType === 'booleanParam'"
                    v-model="runParamForm[item.code]"
                    active-value="true"
                    inactive-value="false"
                  />
                  <ElInputNumber
                    v-else-if="item.paramType === 'number'"
                    :model-value="numberParamValue(runParamForm[item.code])"
                    controls-position="right"
                    style="width: 100%"
                    placeholder="请输入数字"
                    @update:model-value="
                      (value: number | undefined) => setRunNumberValue(item.code, value)
                    "
                  />
                  <div v-else-if="item.paramType === 'objectList'" class="object-list-value-editor">
                    <div class="object-list-value-head">
                      <span>{{ objectListFields(item).length }} 个字段</span>
                      <ElButton
                        size="small"
                        type="primary"
                        plain
                        :disabled="objectListFields(item).length === 0"
                        @click="addRunObjectListRow(item)"
                      >
                        新增一行
                      </ElButton>
                    </div>
                    <ElEmpty
                      v-if="objectListFields(item).length === 0"
                      description="未配置对象列表字段"
                    />
                    <ElTable
                      v-else
                      :data="runObjectListRows(item)"
                      size="small"
                      border
                      empty-text="暂无数据"
                    >
                      <ElTableColumn
                        v-for="field in objectListFields(item)"
                        :key="field.code"
                        :label="field.name || field.code"
                        :min-width="160"
                      >
                        <template #default="{ row, $index }">
                          <component
                            :is="objectListFieldComponent(field)"
                            :model-value="objectListFieldValue(row, field)"
                            v-bind="objectListFieldProps(field, false)"
                            @update:model-value="
                              (value: any) =>
                                updateRunObjectListFieldValue(item, $index, field.code, value)
                            "
                          >
                            <template v-if="objectListFieldIsSelect(field)" #default>
                              <ElOption
                                v-for="option in objectListFieldOptions(field)"
                                :key="option.value"
                                :label="option.label || option.value"
                                :value="option.value"
                              />
                            </template>
                          </component>
                        </template>
                      </ElTableColumn>
                      <ElTableColumn label="操作" width="88" fixed="right">
                        <template #default="{ $index }">
                          <ElButton
                            text
                            type="danger"
                            @click="removeRunObjectListRow(item, $index)"
                          >
                            删除
                          </ElButton>
                        </template>
                      </ElTableColumn>
                    </ElTable>
                  </div>
                  <ElInput
                    v-else-if="item.paramType === 'file'"
                    v-model="runParamForm[item.code]"
                    placeholder="请输入文件路径或制品地址"
                  />
                  <DevopsPlatformAddressPicker
                    v-else-if="isCompositePipelineParam(item)"
                    v-model="runParamForm[item.code]"
                    :param-type="platformAddressParamType(item)"
                    :project-id="runTargetPipeline?.projectId || projectStore.selectedProjectId"
                    :system-id="runTargetPipeline?.systemId || ''"
                    :environment-id="runTargetPipeline?.environmentId || ''"
                    :channel-group-code="
                      item.config?.channelGroupCode || item.channelGroupCode || ''
                    "
                    :channel-type="item.config?.channelTypeFilter || ''"
                    :mapping-field="item.config?.mappingField || ''"
                    :placeholder="item.defaultValue || ''"
                    @change="() => handleRunParamChange(item)"
                  />
                  <div
                    v-else-if="isRunSelectParam(item)"
                    :class="{ 'dynamic-select-control': isDynamicRunParam(item) }"
                  >
                    <ElSelect
                      v-model="runParamForm[item.code]"
                      class="platform-param-select"
                      popper-class="devops-platform-param-dropdown"
                      :teleported="true"
                      fit-input-width
                      filterable
                      clearable
                      allow-create
                      default-first-option
                      :remote-show-suffix="
                        isDynamicRunParam(item) || isCredentialPipelineParam(item)
                      "
                      :remote="isDynamicRunParam(item) || isCredentialPipelineParam(item)"
                      :remote-method="(keyword: string) => queryRunParamOptions(item, keyword)"
                      :loading="runParamOptionLoading[item.code]"
                      loading-text="加载中..."
                      :no-data-text="dynamicEmptyText(item.paramType)"
                      :placeholder="runDynamicPlaceholder(item)"
                      @visible-change="
                        (visible: boolean) => handleRunParamVisibleChange(item, visible)
                      "
                      @change="() => handleRunParamChange(item)"
                    >
                      <template #loading>
                        <div class="devops-select-loading">
                          <span class="devops-select-loading__icon"></span>
                          <span>加载中...</span>
                        </div>
                      </template>
                      <div
                        v-if="dynamicRunParamError[item.code]"
                        class="dynamic-option-tip is-error"
                      >
                        {{ dynamicRunParamError[item.code] }}
                      </div>
                      <div
                        v-else-if="
                          dynamicRunParamLoaded[item.code] &&
                          !runParamOptionLoading[item.code] &&
                          !runParamOptions(item).length
                        "
                        class="dynamic-option-tip"
                      >
                        {{ dynamicEmptyText(item.paramType) }}
                      </div>
                      <ElOption
                        v-for="option in runParamOptions(item)"
                        :key="option.value"
                        :label="option.label"
                        :value="option.value"
                      >
                        <div class="dynamic-option-item">
                          <span>{{ option.label }}</span>
                          <small v-if="dynamicOptionSecondary(option)">
                            {{ dynamicOptionSecondary(option) }}
                          </small>
                        </div>
                      </ElOption>
                    </ElSelect>
                    <ElButton
                      v-if="isDynamicRunParam(item)"
                      class="dynamic-select-refresh"
                      :icon="RefreshRight"
                      :loading="runParamOptionLoading[item.code]"
                      title="刷新选项"
                      aria-label="刷新选项"
                      @click.stop="refreshRunParamOptions(item)"
                    >
                      刷新
                    </ElButton>
                  </div>
                  <ElInput
                    v-else
                    v-model="runParamForm[item.code]"
                    :type="item.paramType === 'password' ? 'password' : 'text'"
                    :show-password="item.paramType === 'password'"
                    :placeholder="item.defaultValue || '请输入参数值'"
                  />
                </ElFormItem>
              </div>
              <div v-if="runWorkspaceItems.length > 0" class="run-param-section-title">
                Workspace 绑定
              </div>
              <div
                v-for="item in runWorkspaceItems"
                :key="item.name"
                class="run-param-field"
                :class="{ required: !item.optional }"
              >
                <div class="run-param-meta">
                  <label class="run-param-label" :title="item.label">
                    <span>{{ item.label }}</span>
                  </label>
                  <div class="run-param-tags">
                    <ElTag size="small" effect="plain">Workspace</ElTag>
                    <ElTag size="small" type="success">{{ item.bindingLabel }}</ElTag>
                  </div>
                </div>
                <div class="run-param-code" :title="item.name">{{ item.name }}</div>
                <ElFormItem class="run-param-control">
                  <ElSelect
                    v-if="runWorkspaceOptions(item).length > 0"
                    v-model="runWorkspaceForm[item.name]"
                    class="platform-param-select"
                    popper-class="devops-platform-param-dropdown"
                    :teleported="true"
                    fit-input-width
                    :placeholder="item.placeholder"
                    filterable
                    clearable
                    allow-create
                    default-first-option
                    style="width: 100%"
                    @visible-change="
                      (visible: boolean) =>
                        visible && fetchTektonRuntimeResources(runTargetPipeline || undefined)
                    "
                  >
                    <ElOption
                      v-for="option in runWorkspaceOptions(item)"
                      :key="option.value"
                      :label="option.label"
                      :value="option.value"
                    />
                  </ElSelect>
                  <ElInput
                    v-else
                    v-model="runWorkspaceForm[item.name]"
                    :placeholder="item.placeholder"
                    clearable
                  />
                </ElFormItem>
              </div>
            </div>
          </ElForm>
        </section>
      </div>
      <template #footer>
        <ElButton :disabled="runSubmitLoading" @click="runParamVisible = false">取消</ElButton>
        <ElButton
          type="primary"
          :disabled="runParamInitializing || !!runParamInitError"
          :loading="runSubmitLoading"
          @click="submitRunPipeline"
        >
          运行
        </ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="triggerConfigVisible"
      title="定时配置"
      width="760px"
      align-center
      class="pipeline-config-dialog trigger-config-dialog"
      :close-on-click-modal="false"
    >
      <div class="config-dialog-shell">
        <section class="config-panel">
          <div class="mini-list-head">
            <span>触发规则</span>
            <ElButton size="small" text type="primary" :disabled="isViewMode" @click="addTrigger">
              新增定时
            </ElButton>
          </div>
          <div class="mini-list trigger-list">
            <ElEmpty v-if="formData.triggers.length === 0" description="暂无定时配置" />
            <div
              v-for="(item, index) in formData.triggers"
              :key="index"
              class="mini-row trigger-row"
            >
              <ElSwitch v-model="item.enabled" :disabled="isViewMode" />
              <ElSelect v-model="item.type" placeholder="触发类型" :disabled="isViewMode">
                <ElOption
                  v-for="trigger in triggerDefinitions"
                  :key="trigger.type"
                  :label="trigger.title"
                  :value="trigger.type"
                />
              </ElSelect>
              <ElInput v-model="item.spec" :disabled="isViewMode" placeholder="例如 H 2 * * *" />
              <ElButton text type="danger" :disabled="isViewMode" @click="removeTrigger(index)">
                删除
              </ElButton>
              <p class="trigger-help">{{ triggerDescription(item.type) }}</p>
            </div>
          </div>
        </section>
      </div>
      <template #footer>
        <ElButton v-if="triggerConfigFromCard" @click="triggerConfigVisible = false">取消</ElButton>
        <ElButton
          type="primary"
          :loading="submitLoading"
          @click="
            triggerConfigFromCard ? submitCardTriggerConfig() : (triggerConfigVisible = false)
          "
        >
          保存
        </ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="tektonConfigVisible"
      title="Tekton 运行配置"
      width="980px"
      align-center
      class="pipeline-config-dialog tekton-config-dialog"
      :close-on-click-modal="false"
      @closed="tektonConfigFromCard = false"
    >
      <div class="tekton-config-intro">
        <div>
          <strong>{{ tektonConfigIntro.title }}</strong>
          <span>{{ tektonConfigIntro.description }}</span>
        </div>
        <ElTag effect="plain">{{ tektonConfigIntro.scope }}</ElTag>
      </div>
      <ElTabs v-model="tektonConfigActiveTab" class="tekton-config-tabs">
        <ElTabPane label="Trigger" name="trigger">
          <ElForm label-width="132px" :disabled="isViewMode && !tektonConfigFromCard">
            <ElFormItem label="启用 Trigger">
              <ElSwitch v-model="tektonDagConfig.triggerConfig.enabled" />
            </ElFormItem>
            <div class="tekton-config-grid two">
              <ElFormItem label="API 版本">
                <ElSelect v-model="tektonDagConfig.triggerConfig.apiVersion" style="width: 100%">
                  <ElOption
                    label="triggers.tekton.dev/v1beta1"
                    value="triggers.tekton.dev/v1beta1"
                  />
                  <ElOption
                    label="triggers.tekton.dev/v1alpha1"
                    value="triggers.tekton.dev/v1alpha1"
                  />
                </ElSelect>
              </ElFormItem>
              <ElFormItem label="Provider">
                <ElSelect
                  v-model="tektonDagConfig.triggerConfig.provider"
                  clearable
                  filterable
                  style="width: 100%"
                >
                  <ElOption label="Generic" value="generic" />
                  <ElOption label="GitLab" value="gitlab" />
                  <ElOption label="GitHub" value="github" />
                  <ElOption label="Gitee" value="gitee" />
                </ElSelect>
              </ElFormItem>
              <ElFormItem label="Namespace">
                <ElInput
                  v-model="tektonDagConfig.triggerConfig.namespace"
                  placeholder="默认使用 Pipeline Namespace"
                />
              </ElFormItem>
              <ElFormItem label="ServiceAccount">
                <ElInput
                  v-model="tektonDagConfig.triggerConfig.serviceAccountName"
                  placeholder="EventListener ServiceAccount，可选"
                />
              </ElFormItem>
              <ElFormItem label="EventListener">
                <ElInput
                  v-model="tektonDagConfig.triggerConfig.eventListenerName"
                  placeholder="例如 build-listener"
                />
              </ElFormItem>
              <ElFormItem label="Trigger">
                <ElInput
                  v-model="tektonDagConfig.triggerConfig.triggerName"
                  placeholder="例如 build-trigger"
                />
              </ElFormItem>
              <ElFormItem label="TriggerBinding">
                <ElInput
                  v-model="tektonDagConfig.triggerConfig.triggerBindingName"
                  placeholder="例如 build-binding"
                />
              </ElFormItem>
              <ElFormItem label="TriggerTemplate">
                <ElInput
                  v-model="tektonDagConfig.triggerConfig.triggerTemplateName"
                  placeholder="例如 build-template"
                />
              </ElFormItem>
              <ElFormItem label="Service 类型">
                <ElSelect v-model="tektonDagConfig.triggerConfig.serviceType" style="width: 100%">
                  <ElOption label="ClusterIP" value="ClusterIP" />
                  <ElOption label="NodePort" value="NodePort" />
                  <ElOption label="LoadBalancer" value="LoadBalancer" />
                </ElSelect>
              </ElFormItem>
              <ElFormItem label="Webhook 地址">
                <ElInput
                  v-model="tektonDagConfig.triggerConfig.webhookPath"
                  placeholder="/tekton/webhook/build"
                >
                  <template #append>
                    <ElButton @click="copyTektonWebhookPath">复制</ElButton>
                  </template>
                </ElInput>
              </ElFormItem>
              <ElFormItem label="Ingress Host">
                <ElInput
                  v-model="tektonDagConfig.triggerConfig.ingressHost"
                  placeholder="hooks.example.com"
                />
              </ElFormItem>
              <ElFormItem label="Ingress Path">
                <ElInput
                  v-model="tektonDagConfig.triggerConfig.ingressPath"
                  placeholder="/hook/build"
                />
              </ElFormItem>
              <p class="tekton-config-note">
                Webhook 地址中的路径优先用于 EventListener 和 Ingress；未填写时再使用 Ingress Path。
              </p>
              <ElFormItem label="IngressClass">
                <ElInput
                  v-model="tektonDagConfig.triggerConfig.ingressClassName"
                  placeholder="nginx"
                />
              </ElFormItem>
              <ElFormItem label="TLS Secret">
                <ElInput
                  v-model="tektonDagConfig.triggerConfig.tlsSecretName"
                  placeholder="hooks-tls"
                />
              </ElFormItem>
              <ElFormItem label="Interceptor">
                <ElSelect
                  v-model="tektonDagConfig.triggerConfig.interceptorType"
                  style="width: 100%"
                >
                  <ElOption label="不使用" value="none" />
                  <ElOption label="GitHub" value="github" />
                  <ElOption label="GitLab" value="gitlab" />
                  <ElOption label="CEL" value="cel" />
                </ElSelect>
              </ElFormItem>
              <ElFormItem label="事件类型">
                <ElSelect
                  v-model="tektonDagConfig.triggerConfig.eventTypes"
                  multiple
                  filterable
                  allow-create
                  default-first-option
                  style="width: 100%"
                >
                  <ElOption label="push" value="push" />
                  <ElOption label="pull_request" value="pull_request" />
                  <ElOption label="Push Hook" value="Push Hook" />
                  <ElOption label="Merge Request Hook" value="Merge Request Hook" />
                </ElSelect>
              </ElFormItem>
              <ElFormItem label="Secret 名称">
                <ElInput
                  v-model="tektonDagConfig.triggerConfig.secretName"
                  placeholder="GitHub/GitLab 签名 Secret"
                />
              </ElFormItem>
              <ElFormItem label="Secret Key">
                <ElInput v-model="tektonDagConfig.triggerConfig.secretKey" placeholder="token" />
              </ElFormItem>
              <ElFormItem label="CEL 过滤">
                <ElInput
                  v-model="tektonDagConfig.triggerConfig.celFilter"
                  placeholder="body.ref == 'refs/heads/main'"
                />
              </ElFormItem>
              <ElFormItem label="Webhook URL">
                <ElInput
                  v-model="tektonDagConfig.triggerConfig.webhookUrl"
                  placeholder="外部访问地址，可选"
                />
              </ElFormItem>
            </div>
            <div class="tekton-config-section">
              <div class="tekton-section-head compact">
                <strong>TriggerBinding 参数</strong>
                <ElButton text type="primary" @click="addTektonTriggerParam">新增参数</ElButton>
              </div>
              <div class="tekton-param-list">
                <ElEmpty
                  v-if="!(tektonDagConfig.triggerConfig.bindingParams || []).length"
                  description="暂无参数映射"
                />
                <div
                  v-for="(item, index) in tektonDagConfig.triggerConfig.bindingParams"
                  :key="index"
                  class="tekton-param-row four"
                >
                  <ElInput v-model="item.name" placeholder="参数名，如 revision" />
                  <ElInput v-model="item.value" placeholder="$(body.head_commit.id)" />
                  <ElInput v-model="item.description" placeholder="说明" />
                  <ElButton text type="danger" @click="removeTektonTriggerParam(index)"
                    >删除</ElButton
                  >
                </div>
              </div>
            </div>
            <div class="tekton-config-section">
              <div class="tekton-section-head compact">
                <strong>TriggerTemplate 参数</strong>
                <ElButton text type="primary" @click="addTektonTriggerTemplateParam">
                  新增参数
                </ElButton>
              </div>
              <div class="tekton-param-list">
                <ElEmpty
                  v-if="!(tektonDagConfig.triggerConfig.templateParams || []).length"
                  description="默认复用 TriggerBinding 参数"
                />
                <div
                  v-for="(item, index) in tektonDagConfig.triggerConfig.templateParams"
                  :key="index"
                  class="tekton-param-row four"
                >
                  <ElInput v-model="item.name" placeholder="参数名，如 revision" />
                  <ElInput v-model="item.value" placeholder="默认值，可选" />
                  <ElInput v-model="item.description" placeholder="说明" />
                  <ElButton text type="danger" @click="removeTektonTriggerTemplateParam(index)">
                    删除
                  </ElButton>
                </div>
              </div>
            </div>
            <div class="tekton-config-section">
              <div class="tekton-section-head compact">
                <strong>资源元数据</strong>
                <div>
                  <ElButton text type="primary" @click="addTektonTriggerLabel">
                    新增 Label
                  </ElButton>
                  <ElButton text type="primary" @click="addTektonTriggerAnnotation">
                    新增 Annotation
                  </ElButton>
                </div>
              </div>
              <div class="tekton-param-list">
                <div
                  v-for="(item, index) in tektonDagConfig.triggerConfig.labels"
                  :key="`trigger-label-${index}`"
                  class="tekton-param-row three"
                >
                  <ElInput v-model="item.name" placeholder="Label Key" />
                  <ElInput v-model="item.value" placeholder="Label Value" />
                  <ElButton text type="danger" @click="removeTektonTriggerLabel(index)">
                    删除
                  </ElButton>
                </div>
                <div
                  v-for="(item, index) in tektonDagConfig.triggerConfig.annotations"
                  :key="`trigger-annotation-${index}`"
                  class="tekton-param-row three"
                >
                  <ElInput v-model="item.name" placeholder="Annotation Key" />
                  <ElInput v-model="item.value" placeholder="Annotation Value" />
                  <ElButton text type="danger" @click="removeTektonTriggerAnnotation(index)">
                    删除
                  </ElButton>
                </div>
                <ElEmpty
                  v-if="
                    !tektonDagConfig.triggerConfig.labels?.length &&
                    !tektonDagConfig.triggerConfig.annotations?.length
                  "
                  description="未配置额外元数据"
                />
              </div>
            </div>
          </ElForm>
        </ElTabPane>
        <ElTabPane label="定时运行" name="schedule">
          <ElForm label-width="132px" :disabled="isViewMode && !tektonConfigFromCard">
            <div class="tekton-config-section">
              <div class="tekton-section-head compact">
                <strong>调度规则</strong>
              </div>
              <div class="tekton-config-grid two">
                <ElFormItem label="启用定时">
                  <ElSwitch v-model="tektonDagConfig.scheduleConfig.enabled" />
                </ElFormItem>
                <ElFormItem label="Cron">
                  <ElInput v-model="tektonDagConfig.scheduleConfig.cron" placeholder="0 2 * * *" />
                </ElFormItem>
                <ElFormItem label="时区">
                  <ElInput
                    v-model="tektonDagConfig.scheduleConfig.timezone"
                    placeholder="Asia/Shanghai"
                  />
                </ElFormItem>
                <ElFormItem label="并发策略">
                  <ElSelect v-model="tektonDagConfig.scheduleConfig.concurrencyPolicy">
                    <ElOption label="允许并发" value="allow" />
                    <ElOption label="跳过新运行" value="skip" />
                    <ElOption label="替换运行" value="replace" />
                  </ElSelect>
                </ElFormItem>
                <ElFormItem label="参数快照">
                  <ElSwitch v-model="tektonDagConfig.scheduleConfig.parameterSnapshot" />
                </ElFormItem>
              </div>
              <p class="tekton-config-note">
                定时触发由平台调度器创建 PipelineRun，不伪装为 Tekton 原生 Cron 资源。
              </p>
            </div>
            <div class="tekton-config-section">
              <div class="tekton-section-head compact">
                <strong>运行配置快照</strong>
                <ElButton
                  text
                  type="primary"
                  :disabled="tektonDagConfig.scheduleConfig.parameterSnapshot === false"
                  @click="refreshTektonScheduleParamSnapshot"
                >
                  刷新快照
                </ElButton>
              </div>
              <div class="tekton-param-list">
                <template v-if="tektonDagConfig.scheduleConfig.parameterSnapshot !== false">
                  <ElEmpty
                    v-if="
                      Object.keys(tektonDagConfig.scheduleConfig.paramsSnapshot || {}).length === 0
                    "
                    description="暂无运行配置"
                  />
                  <div
                    v-for="(_, key) in tektonDagConfig.scheduleConfig.paramsSnapshot"
                    :key="key"
                    class="tekton-param-row two"
                  >
                    <span class="snapshot-key" :title="String(key)">{{ key }}</span>
                    <ElInput v-model="tektonDagConfig.scheduleConfig.paramsSnapshot![key]" />
                  </div>
                </template>
                <ElEmpty v-else description="已关闭参数快照" />
              </div>
            </div>
          </ElForm>
        </ElTabPane>
        <ElTabPane label="PipelineRun" name="runPolicy">
          <ElForm label-width="132px" :disabled="isViewMode && !tektonConfigFromCard">
            <div class="tekton-config-section">
              <div class="tekton-section-head compact">
                <strong>运行策略</strong>
              </div>
              <div class="tekton-config-grid two">
                <ElFormItem label="Workspace 策略">
                  <ElSelect v-model="tektonDagConfig.runPolicy.workspaceStrategy">
                    <ElOption label="保持 EmptyDir" value="emptyDir" />
                    <ElOption label="统一使用 PVC" value="persistentVolumeClaim" />
                    <ElOption label="统一使用动态 PVC" value="volumeClaimTemplate" />
                  </ElSelect>
                </ElFormItem>
                <ElFormItem label="ServiceAccount">
                  <ElSelect
                    v-model="tektonDagConfig.runPolicy.serviceAccountName"
                    clearable
                    filterable
                    allow-create
                    default-first-option
                    placeholder="默认使用 Pipeline 或渠道配置"
                  >
                    <ElOption
                      v-for="item in tektonServiceAccountOptions"
                      :key="item.value"
                      :label="item.label"
                      :value="item.value"
                    />
                  </ElSelect>
                </ElFormItem>
                <ElFormItem label="ManagedBy">
                  <ElInput
                    v-model="tektonDagConfig.runPolicy.managedBy"
                    clearable
                    placeholder="默认由 Tekton Pipelines 控制器处理"
                  />
                </ElFormItem>
                <ElFormItem label="并发策略">
                  <ElSelect v-model="tektonDagConfig.runPolicy.concurrencyPolicy">
                    <ElOption label="允许并发" value="allow" />
                    <ElOption label="跳过新运行" value="skip" />
                    <ElOption label="替换运行" value="replace" />
                  </ElSelect>
                </ElFormItem>
                <ElFormItem label="Pipeline Timeout">
                  <ElInput v-model="tektonDagConfig.runPolicy.pipelineTimeout" placeholder="1h" />
                </ElFormItem>
                <ElFormItem label="Tasks Timeout">
                  <ElInput v-model="tektonDagConfig.runPolicy.tasksTimeout" placeholder="30m" />
                </ElFormItem>
                <ElFormItem label="Finally Timeout">
                  <ElInput v-model="tektonDagConfig.runPolicy.finallyTimeout" placeholder="10m" />
                </ElFormItem>
                <ElFormItem label="清理镜像">
                  <ElInput
                    v-model="tektonDagConfig.runPolicy.cleanupImage"
                    clearable
                    placeholder="默认 busybox:1.36"
                  />
                </ElFormItem>
                <ElFormItem label="执行前清理">
                  <ElSwitch v-model="tektonDagConfig.runPolicy.cleanBeforeRun" />
                </ElFormItem>
                <ElFormItem label="执行后清理">
                  <ElSwitch v-model="tektonDagConfig.runPolicy.cleanAfterRun" />
                </ElFormItem>
              </div>
              <p class="tekton-config-note">
                Workspace 策略只作用于仍为 EmptyDir 的 Pipeline Workspace；统一使用 PVC 时必须在
                运行配置里为对应 Workspace 填写 PVC 名称，统一使用动态 PVC 时未填写容量会默认 10Gi。
                开启清理时如未指定清理镜像，默认使用 `busybox:1.36`。
              </p>
            </div>
            <div class="tekton-config-section">
              <div class="tekton-section-head compact">
                <strong>podTemplate 环境变量</strong>
                <ElButton text type="primary" @click="addTektonRunPolicyEnv"> 新增变量 </ElButton>
              </div>
              <div class="tekton-param-list">
                <div
                  v-for="(item, index) in tektonDagConfig.runPolicy.podTemplateEnv"
                  :key="`run-env-${index}`"
                  class="tekton-param-row three"
                >
                  <ElInput v-model="item.name" placeholder="变量名，如 HTTP_PROXY" />
                  <ElInput v-model="item.value" placeholder="变量值" />
                  <ElButton text type="danger" @click="removeTektonRunPolicyEnv(index)">
                    删除
                  </ElButton>
                </div>
                <ElEmpty
                  v-if="!tektonDagConfig.runPolicy.podTemplateEnv?.length"
                  description="平台会自动注入 buildId 相关变量"
                />
              </div>
              <p class="tekton-config-note">
                BUILD_ID、buildId、KUBE_NOVA_BUILD_ID、KUBE_NOVA_RUN_ID、KUBE_NOVA_PIPELINE_RUN_NAME
                由平台自动注入，用户配置同名变量时以平台值为准。
              </p>
            </div>
            <div class="tekton-config-section">
              <div class="tekton-section-head compact">
                <strong>PipelineRun 元数据</strong>
                <div>
                  <ElButton text type="primary" @click="addTektonRunPolicyLabel">
                    新增 Label
                  </ElButton>
                  <ElButton text type="primary" @click="addTektonRunPolicyAnnotation">
                    新增 Annotation
                  </ElButton>
                </div>
              </div>
              <div class="tekton-param-list">
                <div
                  v-for="(item, index) in tektonDagConfig.runPolicy.labels"
                  :key="`run-label-${index}`"
                  class="tekton-param-row three"
                >
                  <ElInput v-model="item.name" placeholder="Label Key" />
                  <ElInput v-model="item.value" placeholder="Label Value" />
                  <ElButton text type="danger" @click="removeTektonRunPolicyLabel(index)">
                    删除
                  </ElButton>
                </div>
                <div
                  v-for="(item, index) in tektonDagConfig.runPolicy.annotations"
                  :key="`run-annotation-${index}`"
                  class="tekton-param-row three"
                >
                  <ElInput v-model="item.name" placeholder="Annotation Key" />
                  <ElInput v-model="item.value" placeholder="Annotation Value" />
                  <ElButton text type="danger" @click="removeTektonRunPolicyAnnotation(index)">
                    删除
                  </ElButton>
                </div>
                <ElEmpty
                  v-if="
                    !tektonDagConfig.runPolicy.labels?.length &&
                    !tektonDagConfig.runPolicy.annotations?.length
                  "
                  description="未配置额外元数据"
                />
              </div>
            </div>
            <div class="tekton-config-section">
              <div class="tekton-section-head compact">
                <strong>TaskRun 高级配置</strong>
              </div>
              <div class="tekton-yaml-field">
                <div class="tekton-yaml-title">
                  <strong>podTemplate</strong>
                  <span>写入 PipelineRun spec.taskRunTemplate.podTemplate</span>
                </div>
                <ElInput
                  v-model="tektonDagConfig.runPolicy.podTemplateRaw"
                  type="textarea"
                  :rows="5"
                  placeholder="nodeSelector:&#10;  kubernetes.io/os: linux"
                />
                <p class="tekton-config-note">
                  只填写 YAML 对象，平台会继续追加 buildId 相关环境变量。
                </p>
              </div>
              <div class="tekton-yaml-field">
                <div class="tekton-yaml-title">
                  <strong>taskRunTemplate</strong>
                  <span>配置所有 TaskRun 共享的模板</span>
                </div>
                <ElInput
                  v-model="tektonDagConfig.runPolicy.taskRunTemplateRaw"
                  type="textarea"
                  :rows="5"
                  placeholder="serviceAccountName: build-sa&#10;podTemplate:&#10;  securityContext: {}"
                />
                <p class="tekton-config-note">
                  只填写 YAML 对象，对应官方 PipelineRun spec.taskRunTemplate。
                </p>
              </div>
              <div class="tekton-yaml-field">
                <div class="tekton-yaml-title">
                  <strong>taskRunSpecs</strong>
                  <span>按 PipelineTask 覆盖单个 TaskRun</span>
                </div>
                <ElInput
                  v-model="tektonDagConfig.runPolicy.taskRunSpecsRaw"
                  type="textarea"
                  :rows="6"
                  placeholder="- pipelineTaskName: build&#10;  serviceAccountName: build-sa"
                />
                <p class="tekton-config-note">
                  支持 YAML 数组或单个对象，每一项必须填写已存在的 pipelineTaskName，字段按
                  PipelineRun v1 格式渲染。
                </p>
              </div>
            </div>
          </ElForm>
        </ElTabPane>
        <ElTabPane label="清理策略" name="pruner">
          <ElForm label-width="132px" :disabled="isViewMode && !tektonConfigFromCard">
            <ElFormItem label="启用 Pruner">
              <ElSwitch v-model="tektonDagConfig.prunerPolicy.enabled" />
            </ElFormItem>
            <div class="tekton-config-grid two">
              <ElFormItem label="集群资源清理">
                <ElSwitch v-model="tektonDagConfig.prunerPolicy.clusterResourceCleanup" />
              </ElFormItem>
              <ElFormItem label="清理平台记录">
                <ElSwitch v-model="tektonDagConfig.prunerPolicy.platformRecordCleanup" />
              </ElFormItem>
              <ElFormItem label="资源类型">
                <ElCheckboxGroup v-model="tektonDagConfig.prunerPolicy.resourceTypes">
                  <ElCheckbox label="PipelineRun">PipelineRun</ElCheckbox>
                  <ElCheckbox label="TaskRun">TaskRun</ElCheckbox>
                </ElCheckboxGroup>
              </ElFormItem>
              <ElFormItem label="清理范围">
                <ElSelect v-model="tektonDagConfig.prunerPolicy.scope" style="width: 100%">
                  <ElOption label="当前 Pipeline" value="pipeline" />
                  <ElOption label="当前 Namespace" value="namespace" />
                </ElSelect>
              </ElFormItem>
              <ElFormItem label="原生 Pruner">
                <ElSelect
                  v-model="tektonDagConfig.prunerPolicy.nativePrunerMode"
                  style="width: 100%"
                >
                  <ElOption label="不生成原生配置" value="disabled" />
                  <ElOption label="Namespace ConfigMap" value="namespaceConfigMap" />
                  <ElOption label="Global ConfigMap" value="globalConfigMap" />
                </ElSelect>
              </ElFormItem>
              <ElFormItem label="原生 Namespace">
                <ElInput
                  v-model="tektonDagConfig.prunerPolicy.nativePrunerTargetNamespace"
                  placeholder="默认使用 Pipeline Namespace，Global 默认 tekton-pipelines"
                />
              </ElFormItem>
              <ElFormItem
                v-if="tektonDagConfig.prunerPolicy.nativePrunerMode === 'globalConfigMap'"
                label="Global 级别"
              >
                <ElSelect
                  v-model="tektonDagConfig.prunerPolicy.nativePrunerConfigLevel"
                  style="width: 100%"
                >
                  <ElOption label="namespace" value="namespace" />
                  <ElOption label="global" value="global" />
                </ElSelect>
              </ElFormItem>
            </div>
            <div class="tekton-config-section">
              <div class="tekton-section-head compact">
                <strong>保留规则</strong>
              </div>
              <div class="tekton-config-grid three">
                <ElFormItem label="总历史">
                  <ElInputNumber
                    v-model="tektonDagConfig.prunerPolicy.historyLimit"
                    :min="0"
                    :max="500"
                    controls-position="right"
                  />
                </ElFormItem>
                <ElFormItem label="成功历史">
                  <ElInputNumber
                    v-model="tektonDagConfig.prunerPolicy.successfulHistoryLimit"
                    :min="0"
                    :max="500"
                    controls-position="right"
                  />
                </ElFormItem>
                <ElFormItem label="失败历史">
                  <ElInputNumber
                    v-model="tektonDagConfig.prunerPolicy.failedHistoryLimit"
                    :min="0"
                    :max="500"
                    controls-position="right"
                  />
                </ElFormItem>
                <ElFormItem label="完成后 TTL">
                  <ElInputNumber
                    v-model="tektonDagConfig.prunerPolicy.ttlSecondsAfterFinished"
                    :min="0"
                    :max="2592000"
                    controls-position="right"
                  />
                </ElFormItem>
                <ElFormItem label="平台保留天数">
                  <ElInputNumber
                    v-model="tektonDagConfig.prunerPolicy.platformRecordRetentionDays"
                    :min="0"
                    :max="3650"
                    controls-position="right"
                  />
                </ElFormItem>
                <ElFormItem label="平台记录上限">
                  <ElInputNumber
                    v-model="tektonDagConfig.prunerPolicy.platformRecordLimit"
                    :min="0"
                    :max="10000"
                    controls-position="right"
                  />
                </ElFormItem>
              </div>
            </div>
            <div class="tekton-config-section">
              <div class="tekton-section-head compact">
                <strong>选择器</strong>
                <div>
                  <ElButton text type="primary" @click="addTektonPrunerSelectorLabel">
                    新增 Label
                  </ElButton>
                  <ElButton text type="primary" @click="addTektonPrunerSelectorAnnotation">
                    新增 Annotation
                  </ElButton>
                </div>
              </div>
              <div class="tekton-param-list">
                <div
                  v-for="(item, index) in tektonDagConfig.prunerPolicy.selectorLabels"
                  :key="`label-${index}`"
                  class="tekton-param-row three"
                >
                  <ElInput v-model="item.name" placeholder="Label Key" />
                  <ElInput v-model="item.value" placeholder="Label Value" />
                  <ElButton text type="danger" @click="removeTektonPrunerSelectorLabel(index)">
                    删除
                  </ElButton>
                </div>
                <div
                  v-for="(item, index) in tektonDagConfig.prunerPolicy.selectorAnnotations"
                  :key="`annotation-${index}`"
                  class="tekton-param-row three"
                >
                  <ElInput v-model="item.name" placeholder="Annotation Key" />
                  <ElInput v-model="item.value" placeholder="Annotation Value" />
                  <ElButton text type="danger" @click="removeTektonPrunerSelectorAnnotation(index)">
                    删除
                  </ElButton>
                </div>
                <ElEmpty
                  v-if="
                    !tektonDagConfig.prunerPolicy.selectorLabels?.length &&
                    !tektonDagConfig.prunerPolicy.selectorAnnotations?.length
                  "
                  description="默认按当前 Pipeline 或 Namespace 清理"
                />
              </div>
              <p class="tekton-config-note">
                集群清理删除 Kubernetes 中的 Tekton 资源；清理平台记录只影响平台执行记录。启用原生
                Pruner 后，平台会同步 Tekton Pruner ConfigMap 到目标 Namespace。
              </p>
            </div>
          </ElForm>
        </ElTabPane>
      </ElTabs>
      <template #footer>
        <ElButton @click="tektonConfigVisible = false">关闭</ElButton>
        <ElButton type="primary" @click="submitTektonConfigDialog">确定</ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="pipelineYamlPreviewVisible"
      :title="pipelineYamlPreviewTitle"
      width="860px"
      top="6vh"
      class="pipeline-yaml-preview-dialog"
    >
      <ElTabs v-model="pipelineYamlPreviewTab">
        <ElTabPane label="Pipeline" name="pipeline">
          <pre class="yaml-preview-content">{{ tektonPipelineYamlPreview || '暂无 YAML' }}</pre>
        </ElTabPane>
        <ElTabPane label="PipelineRun" name="pipelineRun">
          <pre class="yaml-preview-content">{{ tektonPipelineRunYamlPreview || '暂无 YAML' }}</pre>
        </ElTabPane>
        <ElTabPane label="Trigger" name="trigger">
          <pre class="yaml-preview-content">{{ tektonTriggerYamlPreview || '暂无 YAML' }}</pre>
        </ElTabPane>
        <ElTabPane label="定时策略" name="schedule">
          <p class="tekton-config-note">
            这是平台调度器配置预览，不是 Kubernetes CRD，不能直接 kubectl apply。
          </p>
          <pre class="yaml-preview-content">{{ tektonScheduleYamlPreview || '暂无 YAML' }}</pre>
        </ElTabPane>
        <ElTabPane label="清理策略" name="pruner">
          <p class="tekton-config-note">
            这是平台清理策略预览，由平台后台执行，不是 Kubernetes CRD，不能直接 kubectl apply。
          </p>
          <pre class="yaml-preview-content">{{ tektonPrunerYamlPreview || '暂无 YAML' }}</pre>
        </ElTabPane>
      </ElTabs>
      <template #footer>
        <ElButton @click="pipelineYamlPreviewVisible = false">关闭</ElButton>
        <ElButton :loading="tektonYamlLoading" @click="loadTektonYamlPreview(true)">刷新</ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="logDrawerVisible"
      width="86%"
      class="pipeline-log-dialog"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
    >
      <template #header>
        <div class="log-dialog-head">
          <div>
            <strong>{{ currentRun?.pipelineName || '流水线日志' }}</strong>
            <span
              v-if="currentRun?.engineType === 'tekton' && currentRun?.buildId"
              class="log-build-number"
            >
              #{{ currentRun.buildId }}
            </span>
            <span v-else-if="currentRun?.jenkinsBuildNumber" class="log-build-number">
              #{{ currentRun.jenkinsBuildNumber }}
            </span>
          </div>
          <div class="log-actions">
            <ElButton :loading="logLoading" @click="refreshCurrentLog">刷新</ElButton>
            <ElButton @click="copyRunLog">复制</ElButton>
            <ElButton @click="downloadRunLog">下载</ElButton>
            <ElButton
              v-if="currentRun?.jenkinsBuildUrl"
              @click="openJenkinsBuild(currentRun.jenkinsBuildUrl)"
            >
              Jenkins
            </ElButton>
          </div>
        </div>
      </template>
      <ElTabs v-model="activeLogTab" class="log-tabs" @tab-change="handleLogTabChange">
        <ElTabPane label="阶段日志" name="stage">
          <div class="stage-log-layout">
            <div class="stage-list">
              <div class="section-title">阶段</div>
              <button
                v-for="stage in displayRunStages"
                :key="stage.id"
                class="stage-item"
                :class="{
                  active: stage.id === currentStageId,
                  disabled: !isStageLogAvailable(stage) && !isStageWaitingInput(stage),
                  running: isProcessActiveStatus(stage.status)
                }"
                type="button"
                :aria-disabled="!isStageLogAvailable(stage) && !isStageWaitingInput(stage)"
                @click="selectStage(stage)"
              >
                <span class="stage-name" :title="stage.stageName">{{ stage.stageName }}</span>
                <span class="stage-meta">
                  <ElTag size="small" :type="runStatusType(stage.status)">
                    {{ runStatusText(stage.status) }}
                  </ElTag>
                  <span
                    v-if="isTektonRunContext && (stage.tektonTaskRuns?.length || 0) > 1"
                    class="stage-duration"
                  >
                    {{ tektonTaskRunSummary(stage) }}
                  </span>
                  <span class="stage-duration">{{ formatStageDuration(stage) }}</span>
                </span>
              </button>
              <ElEmpty v-if="displayRunStages.length === 0" description="暂无阶段" />
            </div>
            <div class="log-pane">
              <div class="log-section-head">
                <div class="section-title">运行日志</div>
              </div>
              <div v-if="isTektonStageLogSelected" class="tekton-log-runtime">
                <div class="tekton-log-runtime-item">
                  <span>TaskRun</span>
                  <ElSelect
                    v-if="tektonLogTaskRunOptions.length > 1"
                    v-model="tektonLogTaskRunName"
                    filterable
                    :placeholder="currentSelectedStage?.tektonTaskRunName || '请选择 TaskRun'"
                    style="width: 100%"
                    @change="handleTektonTaskRunChange"
                  >
                    <ElOption
                      v-for="item in tektonLogTaskRunOptions"
                      :key="item.name"
                      :label="tektonTaskRunOptionLabel(item)"
                      :value="item.name"
                    />
                  </ElSelect>
                  <strong v-else :title="currentTektonLogTaskRunName || '-'">
                    {{ currentTektonLogTaskRunName || '-' }}
                  </strong>
                </div>
                <div class="tekton-log-runtime-item">
                  <span>Pod</span>
                  <strong :title="currentTektonLogPodName || '-'">
                    {{ currentTektonLogPodName || '-' }}
                  </strong>
                </div>
                <div class="tekton-log-runtime-item container">
                  <span>Container</span>
                  <ElSelect
                    v-model="tektonLogContainerName"
                    clearable
                    filterable
                    allow-create
                    default-first-option
                    :placeholder="currentSelectedStage?.containerName || '全部容器'"
                    style="width: 100%"
                    @change="startLogStream(true)"
                    @clear="startLogStream(true)"
                  >
                    <ElOption
                      v-for="item in tektonLogContainerOptions"
                      :key="item"
                      :label="item"
                      :value="item"
                    />
                  </ElSelect>
                </div>
              </div>
              <div
                ref="stageLogBodyRef"
                v-loading="logLoading"
                element-loading-text="加载中..."
                class="terminal-log"
              >
                <div v-if="logLines.length" class="terminal-content">
                  <div v-for="(line, index) in logLines" :key="index" class="log-line">
                    <span class="line-no">{{ logLineStartNo + index }}</span>
                    <span class="line-text">{{ line || ' ' }}</span>
                  </div>
                </div>
                <div v-if="!runLog && !logLoading" class="terminal-empty">
                  <ElEmpty :description="isCurrentLogExecuting ? '暂无新日志' : '暂无日志'" />
                </div>
              </div>
              <div class="log-bottom-bar">
                <span></span>
                <div v-if="isCurrentLogExecuting" class="log-live-pill">
                  <span class="live-dots" aria-hidden="true">
                    <i></i>
                    <i></i>
                    <i></i>
                  </span>
                  <span>{{ logExecutingText }}</span>
                </div>
                <ElCheckbox v-model="autoScrollLog" class="auto-scroll-check">自动滚动</ElCheckbox>
              </div>
            </div>
          </div>
        </ElTabPane>
        <ElTabPane label="全量日志" name="full">
          <div class="full-log-pane">
            <div class="log-section-head">
              <div class="section-title">运行日志</div>
            </div>
            <div
              ref="fullLogBodyRef"
              v-loading="logLoading"
              element-loading-text="加载中..."
              class="terminal-log full"
            >
              <div v-if="logLines.length" class="terminal-content">
                <div v-for="(line, index) in logLines" :key="index" class="log-line">
                  <span class="line-no">{{ logLineStartNo + index }}</span>
                  <span class="line-text">{{ line || ' ' }}</span>
                </div>
              </div>
              <div v-if="!runLog && !logLoading" class="terminal-empty">
                <ElEmpty :description="isCurrentLogExecuting ? '暂无新日志' : '暂无日志'" />
              </div>
            </div>
            <div class="log-bottom-bar">
              <span></span>
              <div v-if="isCurrentLogExecuting" class="log-live-pill">
                <span class="live-dots" aria-hidden="true">
                  <i></i>
                  <i></i>
                  <i></i>
                </span>
                <span>{{ logExecutingText }}</span>
              </div>
              <ElCheckbox v-model="autoScrollLog" class="auto-scroll-check">自动滚动</ElCheckbox>
            </div>
          </div>
        </ElTabPane>
        <ElTabPane label="确认操作" name="input">
          <div v-loading="inputLoading" element-loading-text="加载中..." class="input-pane">
            <div v-if="runInputs.length > 0" class="input-attention">
              <strong>流水线等待人工确认</strong>
              <span>请确认下方信息，处理后流水线会继续刷新执行状态。</span>
            </div>
            <ElEmpty
              v-if="runInputs.length === 0"
              :description="
                hasInputWaitState ? '正在获取 Jenkins 确认表单，请稍候刷新' : '暂无待确认事项'
              "
            />
            <div v-for="item in runInputs" v-else :key="item.id" class="input-card">
              <div class="input-card-head">
                <strong>{{ item.name || '人工确认' }}</strong>
                <span>{{ item.message || '请确认后继续流水线' }}</span>
              </div>
              <ElForm label-width="120px" class="input-form">
                <ElFormItem v-for="param in item.params" :key="param.name" :label="param.name">
                  <ElSelect
                    v-if="param.options?.length"
                    v-model="ensureInputForm(item.id)[param.name]"
                    clearable
                    filterable
                    style="width: 100%"
                  >
                    <ElOption
                      v-for="option in param.options"
                      :key="option.value"
                      :label="option.label"
                      :value="option.value"
                    />
                  </ElSelect>
                  <ElSwitch
                    v-else-if="param.type === 'boolean'"
                    v-model="ensureInputForm(item.id)[param.name]"
                    active-value="true"
                    inactive-value="false"
                  />
                  <ElInput
                    v-else-if="param.type === 'text'"
                    v-model="ensureInputForm(item.id)[param.name]"
                    type="textarea"
                    :rows="4"
                  />
                  <ElInput
                    v-else
                    v-model="ensureInputForm(item.id)[param.name]"
                    :type="param.type === 'password' ? 'password' : 'text'"
                    :show-password="param.type === 'password'"
                  />
                  <div v-if="param.description" class="input-param-help">
                    {{ param.description }}
                  </div>
                </ElFormItem>
              </ElForm>
              <div class="input-card-actions">
                <ElButton
                  :loading="inputAbortLoadingMap[item.id]"
                  :disabled="inputProceedLoadingMap[item.id]"
                  @click="abortRunInput(item)"
                >
                  终止
                </ElButton>
                <ElButton
                  type="primary"
                  :loading="inputProceedLoadingMap[item.id]"
                  :disabled="inputAbortLoadingMap[item.id]"
                  @click="proceedRunInput(item)"
                >
                  继续
                </ElButton>
              </div>
            </div>
          </div>
        </ElTabPane>
        <ElTabPane label="制品归档" name="artifact">
          <div v-loading="artifactLoading" element-loading-text="加载中..." class="artifact-pane">
            <ElEmpty v-if="runArtifacts.length === 0" description="暂无产物" />
            <div v-else class="artifact-list">
              <div v-for="item in runArtifacts" :key="item.id" class="artifact-row">
                <div class="artifact-main">
                  <strong :title="item.name || item.objectKey">{{
                    item.name || item.objectKey
                  }}</strong>
                  <span :title="item.objectKey">{{ item.objectKey }}</span>
                </div>
                <ElTag size="small" :type="item.status === 'success' ? 'success' : 'info'">
                  {{ item.status || '-' }}
                </ElTag>
                <ElButton link type="primary" :disabled="!item.url" @click="openArtifact(item)">
                  打开
                </ElButton>
              </div>
            </div>
          </div>
        </ElTabPane>
        <ElTabPane label="质量报告" name="quality">
          <div v-loading="qualityLoading" element-loading-text="加载中..." class="quality-pane">
            <ElEmpty v-if="!qualitySummary" description="暂无质量报告" />
            <template v-else>
              <div class="quality-run-metrics">
                <div class="quality-run-metric">
                  <span>报告数</span>
                  <strong>{{ qualitySummary.overview.reportCount }}</strong>
                </div>
                <div class="quality-run-metric">
                  <span>问题数</span>
                  <strong>{{ qualitySummary.overview.issueCount }}</strong>
                </div>
                <div class="quality-run-metric danger">
                  <span>严重</span>
                  <strong>{{ qualitySummary.overview.criticalCount }}</strong>
                </div>
                <div class="quality-run-metric warning">
                  <span>高危</span>
                  <strong>{{ qualitySummary.overview.highCount }}</strong>
                </div>
                <div class="quality-run-metric">
                  <span>通过率</span>
                  <strong>{{ qualitySummary.overview.gatePassRate }}%</strong>
                </div>
                <div
                  class="quality-run-metric"
                  :class="{ danger: qualitySummary.gateStatus === 'failed' }"
                >
                  <span>门禁</span>
                  <strong>{{ qualityGateText(qualitySummary.gateStatus) }}</strong>
                </div>
              </div>
              <ElTabs v-model="activeQualityTab" class="quality-run-tabs">
                <ElTabPane label="报告" name="reports">
                  <ElEmpty v-if="qualitySummary.reports.length === 0" description="暂无报告" />
                  <div v-else class="quality-report-list">
                    <div
                      v-for="report in qualitySummary.reports"
                      :key="report.id"
                      class="quality-report-row"
                    >
                      <div class="quality-report-main">
                        <strong :title="report.targetName || report.reportPath">
                          {{ report.targetName || report.reportPath || report.tool }}
                        </strong>
                        <span :title="report.reportPath || report.objectKey">
                          {{ report.tool }} / {{ report.reportFormat || '-' }}
                        </span>
                      </div>
                      <ElTag size="small" :type="reportStatusType(report.status)">
                        {{ report.status || '-' }}
                      </ElTag>
                      <ElTag size="small" :type="gateStatusType(report.gateStatus)">
                        {{ qualityGateText(report.gateStatus) }}
                      </ElTag>
                      <ElButton link type="primary" @click="previewQualityReport(report.id)">
                        预览
                      </ElButton>
                      <ElButton link type="primary" @click="openQualityReport(report.id)">
                        下载
                      </ElButton>
                    </div>
                  </div>
                </ElTabPane>
                <ElTabPane label="静态扫描" name="static">
                  <ElEmpty
                    v-if="qualitySummary.staticCodeIssues.length === 0"
                    description="暂无静态扫描问题"
                  />
                  <div v-else class="quality-issue-list">
                    <div
                      v-for="issue in qualitySummary.staticCodeIssues"
                      :key="issue.id"
                      class="quality-issue-row"
                    >
                      <ElTag size="small" :type="qualitySeverityType(issue.severity)">
                        {{ issue.severity || '-' }}
                      </ElTag>
                      <div class="quality-issue-main">
                        <strong :title="issue.title">{{
                          issue.title || issue.ruleId || '-'
                        }}</strong>
                        <span :title="issue.filePath">
                          {{ issue.filePath || issue.component || '-' }}
                        </span>
                      </div>
                      <span class="quality-issue-tool">{{ issue.tool || '-' }}</span>
                    </div>
                  </div>
                </ElTabPane>
                <ElTabPane label="镜像扫描" name="image">
                  <ElEmpty
                    v-if="
                      qualitySummary.images.length === 0 && qualitySummary.imageIssues.length === 0
                    "
                    description="暂无镜像扫描结果"
                  />
                  <div v-else class="quality-image-list">
                    <div
                      v-for="image in qualitySummary.images"
                      :key="image.imageRef || image.imageDigest"
                      class="quality-image-row"
                    >
                      <div class="quality-image-main">
                        <strong :title="image.imageRef">{{ image.imageRef || '-' }}</strong>
                        <span :title="image.imageDigest">{{ image.imageDigest || '-' }}</span>
                      </div>
                      <ElTag size="small" type="danger">严重 {{ image.criticalCount }}</ElTag>
                      <ElTag size="small" type="warning">高危 {{ image.highCount }}</ElTag>
                      <ElTag size="small" type="info">可修复 {{ image.fixableCount }}</ElTag>
                    </div>
                    <div
                      v-for="issue in qualitySummary.imageIssues"
                      :key="issue.id"
                      class="quality-issue-row"
                    >
                      <ElTag size="small" :type="qualitySeverityType(issue.severity)">
                        {{ issue.severity || '-' }}
                      </ElTag>
                      <div class="quality-issue-main">
                        <strong :title="issue.title || issue.cveId">
                          {{ issue.title || issue.cveId || issue.packageName || '-' }}
                        </strong>
                        <span :title="issue.imageRef">
                          {{ issue.packageName || '-' }} /
                          {{ issue.fixedVersion || '暂无修复版本' }}
                        </span>
                      </div>
                      <span class="quality-issue-tool">{{ issue.tool || '-' }}</span>
                    </div>
                  </div>
                </ElTabPane>
                <ElTabPane label="门禁" name="gate">
                  <div class="quality-gate-list">
                    <div
                      class="quality-gate-row"
                      :class="{ danger: qualitySummary.gateStatus === 'failed' }"
                    >
                      <span>门禁状态</span>
                      <strong>{{ qualityGateText(qualitySummary.gateStatus) }}</strong>
                    </div>
                    <div
                      v-for="item in qualitySummary.missingReports"
                      :key="`${item.tool}-${item.stepId}-${item.reportPath}`"
                      class="quality-gate-row warning"
                    >
                      <span>缺失报告</span>
                      <strong>
                        {{ item.tool }} / {{ item.reportFormat }} / {{ item.reportPath || '-' }}
                      </strong>
                    </div>
                    <div
                      v-for="report in qualitySummary.rejectedReports"
                      :key="report.id"
                      class="quality-gate-row danger"
                    >
                      <span>拒绝报告</span>
                      <strong
                        >{{ report.tool }} / {{ report.rejectReason || report.status }}</strong
                      >
                    </div>
                    <div
                      v-for="entry in qualitySummary.rejectedEntries"
                      :key="entry.id"
                      class="quality-gate-row danger"
                    >
                      <span>拒绝条目</span>
                      <strong>{{ entry.entryPath || entry.reason || entry.status }}</strong>
                    </div>
                  </div>
                </ElTabPane>
              </ElTabs>
            </template>
          </div>
        </ElTabPane>
      </ElTabs>
    </ElDialog>

    <ElDialog
      v-model="qualityPreviewVisible"
      :title="qualityPreviewTitle"
      width="80%"
      top="6vh"
      @closed="clearQualityPreview"
    >
      <div
        class="quality-preview-dialog-body"
        v-loading="qualityPreviewLoading"
        element-loading-text="加载中..."
      >
        <iframe
          v-if="qualityPreviewUrl"
          class="quality-preview-frame"
          :src="qualityPreviewUrl"
          sandbox=""
        ></iframe>
        <ElEmpty v-else description="暂无可预览内容" />
      </div>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import {
    computed,
    defineAsyncComponent,
    h,
    nextTick,
    onActivated,
    onBeforeUnmount,
    onMounted,
    reactive,
    ref,
    watch
  } from 'vue'
  import {
    ElButton,
    ElDropdown,
    ElDropdownItem,
    ElDropdownMenu,
    ElMessage,
    ElMessageBox,
    ElTag
  } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import { ArrowDown, Plus, RefreshRight, Search } from '@element-plus/icons-vue'
  import {
    Handle,
    MarkerType,
    Position,
    VueFlow,
    type Edge,
    type Node,
    type ViewportTransform,
    type VueFlowStore
  } from '@vue-flow/core'
  import { parse as parseYaml } from 'yaml'
  import DevopsPlatformAddressPicker from '@/components/devops-platform-address-picker/index.vue'
  import { isCompositePlatformParamType } from '@/components/devops-platform-address-picker/defs'
  import type {
    TektonDagConfig,
    TektonDagLibraryItem,
    TektonResourceOption
  } from '@/components/devops-tekton-dag-designer/types'
  import {
    renderTektonDagPrunerPolicyYAML,
    renderTektonDagPipelineRunYAML,
    renderTektonDagPipelineYAML,
    renderTektonDagSchedulePolicyYAML,
    renderTektonDagTriggerYAML
  } from '@/components/devops-tekton-dag-designer/render-yaml'
  import { validateTektonDagConfig } from '@/components/devops-tekton-dag-designer/validate'
  import type {
    WorkflowLibraryItem,
    WorkflowNode
  } from '@/components/devops-workflow-designer/types'
  import { normalizeWorkflowBranches } from '@/components/devops-workflow-designer/workflow-utils'
  import {
    applyProcessStageDependencyGuard as applyProcessStageDependencyGuardCore,
    buildProcessGraphLayout,
    compareProcessStep,
    processDependencyReadyStatuses,
    processStepRootType,
    type ProcessGraphLayout,
    type ProcessParallelGroupNodeData
  } from './processGraph'
  import { smartFormatTime } from '@/utils/format'
  import { useDevopsProjectStore } from '@/store/modules/devopsProject'
  import '@vue-flow/core/dist/style.css'
  import '@vue-flow/core/dist/theme-default.css'

  import {
    listDevopsTektonPvcApi,
    listDevopsTektonSecretApi,
    listDevopsProjectChannelBindingApi,
    listDevopsProjectConfigApi,
    type DevopsProjectChannelBinding,
    type DevopsProjectConfig
  } from '@/api/devops/project'
  import { listDevopsCredentialApi, type DevopsCredential } from '@/api/devops/channel'
  import {
    abortDevopsPipelineRunInputApi,
    createDevopsPipelineApi,
    createDevopsPipelineRunLogStream,
    deleteDevopsPipelineApi,
    deleteDevopsPipelineRunApi,
    getDevopsPipelineApi,
    getDevopsPipelineRunYamlSnapshotApi,
    getDevopsPipelineTemplateApi,
    listDevopsJenkinsStepApi,
    listDevopsTektonStepApi,
    listDevopsTektonRuntimeResourceApi,
    listDevopsDynamicParamOptionsApi,
    listDevopsPipelineApi,
    listDevopsPipelineEnvironmentApi,
    listDevopsPipelineJenkinsAgentApi,
    listDevopsPipelineRunArtifactApi,
    listDevopsPipelineRunApi,
    listDevopsPipelineRunInputApi,
    listDevopsPipelineRunStageApi,
    listDevopsPipelineTemplateApi,
    listDevopsPipelineTemplateVariablesApi,
    listDevopsStepCategoryApi,
    listDevopsSystemApi,
    previewDevopsPipelineApi,
    proceedDevopsPipelineRunInputApi,
    runDevopsPipelineApi,
    stopDevopsPipelineRunApi,
    updateDevopsPipelineApi,
    type CompoundParamField,
    type DevopsPipeline,
    type DevopsPipelineEnvironment,
    type DevopsPipelineForm,
    type DevopsPipelineRun,
    type DevopsPipelineRunArtifact,
    type DevopsPipelineRunInput,
    type DevopsPipelineRunStage,
    type DevopsPipelineTemplate,
    type DevopsStepCategory,
    type DevopsStepParam,
    type DevopsStepTemplate,
    type DevopsSystem,
    type JenkinsOption,
    type JenkinsAgentOption,
    type JenkinsTool,
    type JenkinsTrigger,
    type DynamicParamOption,
    type DynamicParamOptionMetadata,
    type DynamicSelectOption,
    type ListTektonRuntimeResourceParams,
    type PipelineParam,
    type PipelineScanPlanItem,
    type PipelineTemplateStep,
    type PipelineTemplateVariable,
    type StepParamConfig,
    type TektonTaskRunSnapshot,
    type TektonRuntimeResource
  } from '@/api/devops/pipeline'
  import {
    downloadQualityScanReportApi,
    getQualityRunSummaryApi,
    previewQualityScanReportApi,
    type QualityRunSummaryResponse
  } from '@/api/devops/quality'

  const DevopsKubeNovaDeployConfigEditor = defineAsyncComponent(
    () => import('@/components/devops-kube-nova-deploy-config-editor/index.vue')
  )
  const DevopsKubernetesDeployConfigEditor = defineAsyncComponent(
    () => import('@/components/devops-kubernetes-deploy-config-editor/index.vue')
  )
  const DevopsTektonDagDesigner = defineAsyncComponent(
    () => import('@/components/devops-tekton-dag-designer/index.vue')
  )
  const DevopsWorkflowDesigner = defineAsyncComponent(
    () => import('@/components/devops-workflow-designer/index.vue')
  )

  defineOptions({ name: 'DevopsPipelineInstance' })

  type EditorMode = 'browse' | 'add' | 'edit' | 'view' | 'records' | 'process'
  type MainConfigContext = 'create' | 'card' | 'editor'
  type TektonDagWorkspaceItem = TektonDagConfig['workspaces'][number]
  type RunWorkspaceBindingType =
    | 'persistentVolumeClaim'
    | 'secret'
    | 'generatedSecret'
    | 'configMap'
    | 'projected'
  type ProjectWorkspaceBindingType =
    | 'emptyDir'
    | 'persistentVolumeClaim'
    | 'volumeClaimTemplate'
    | 'secret'
    | 'generatedSecret'
    | 'configMap'
    | 'projected'
  interface TektonRunWorkspaceItem {
    name: string
    label: string
    bindingType: RunWorkspaceBindingType
    bindingLabel: string
    placeholder: string
    optional?: boolean
    defaultValue?: string
  }
  interface TektonProjectWorkspaceItem extends TektonDagWorkspaceItem {
    bindingType: ProjectWorkspaceBindingType
  }
  type TektonDetailTab =
    | 'overview'
    | 'dag'
    | 'records'
    | 'trigger'
    | 'runPolicy'
    | 'pruner'
    | 'audit'
  type TektonConfigTab = 'trigger' | 'schedule' | 'runPolicy' | 'pruner'
  type PipelineYamlPreviewTab = 'pipeline' | 'pipelineRun' | 'trigger' | 'schedule' | 'pruner'
  type PipelineSearchForm = {
    name?: string
    systemId?: string
    environmentId?: string
  }
  type ProcessStageNodeData = {
    stage: DevopsPipelineRunStage
    status: string
    symbol: string
    duration: string
    parallelGroup: string
    needsInput: boolean
    isPost: boolean
  }
  const clonePlain = <T,>(value: T): T => JSON.parse(JSON.stringify(value || {}))
  const projectStore = useDevopsProjectStore()
  const mainFormRef = ref<FormInstance>()
  const loading = ref(false)
  const stepLoading = ref(false)
  const submitLoading = ref(false)
  const runLoading = ref(false)
  const runParamInitializing = ref(false)
  const runParamInitError = ref('')
  const runSubmitLoading = ref(false)
  const processOpeningRunId = ref('')
  const runActionLoading = ref(false)
  const logLoading = ref(false)
  const logDrawerVisible = ref(false)
  const artifactLoading = ref(false)
  const qualityLoading = ref(false)
  const qualityPreviewLoading = ref(false)
  const qualityPreviewVisible = ref(false)
  const qualityPreviewUrl = ref('')
  const qualityPreviewTitle = ref('报告预览')
  const inputLoading = ref(false)
  const runParamVisible = ref(false)
  const tektonWorkspaceConfigVisible = ref(false)
  const tektonWorkspaceItemEditorVisible = ref(false)
  const editingTektonWorkspaceIndex = ref(-1)
  const tektonWorkspaceItemEditorSnapshot = ref<TektonProjectWorkspaceItem | null>(null)
  const mainConfigVisible = ref(false)
  const mainConfigLoading = ref(false)
  const templateSourceVisible = ref(false)
  const templateLoading = ref(false)
  const currentEngineType = computed(() => formData.engineType || 'jenkins')
  const jenkinsConfigVisible = ref(false)
  const paramConfigVisible = ref(false)
  const triggerConfigVisible = ref(false)
  const triggerConfigFromCard = ref(false)
  const tektonConfigVisible = ref(false)
  const tektonConfigFromCard = ref(false)
  const tektonConfigActiveTab = ref<TektonConfigTab>('trigger')
  const tektonConfigIntro = computed(() => {
    const map: Record<TektonConfigTab, { title: string; description: string; scope: string }> = {
      trigger: {
        title: 'Trigger / EventListener',
        description: '配置 Tekton Triggers 资源、Webhook 地址、参数映射和鉴权方式。',
        scope: 'Tekton Triggers'
      },
      schedule: {
        title: '定时触发',
        description: '由平台调度器按 Cron 创建 PipelineRun，不伪装成 Tekton 原生 Cron 资源。',
        scope: '平台能力'
      },
      runPolicy: {
        title: 'PipelineRun 配置',
        description:
          '配置 PipelineRun 的 ServiceAccount、Workspace 策略、元数据、环境变量和超时时间。',
        scope: 'PipelineRun'
      },
      pruner: {
        title: 'Pruner 清理',
        description: '区分集群 Tekton 资源清理和平台执行记录保留，避免策略混淆。',
        scope: 'Tekton / 平台'
      }
    }
    return map[tektonConfigActiveTab.value]
  })
  const tektonDetailActiveTab = ref<TektonDetailTab>('overview')
  const pipelineYamlPreviewVisible = ref(false)
  const pipelineYamlPreviewTab = ref<PipelineYamlPreviewTab>('pipeline')
  const pipelineYamlPreviewTitle = ref('Tekton YAML')
  const tektonPipelineYamlPreview = ref('')
  const tektonPipelineRunYamlPreview = ref('')
  const tektonTriggerYamlPreview = ref('')
  const tektonScheduleYamlPreview = ref('')
  const tektonPrunerYamlPreview = ref('')
  const tektonYamlLoading = ref(false)
  const editorMode = ref<EditorMode>('browse')
  const mainConfigContext = ref<MainConfigContext>('create')
  const showSearchBar = ref(true)
  const currentPipeline = ref<DevopsPipeline | null>(null)
  const currentRunPipeline = ref<DevopsPipeline | null>(null)
  const runTargetPipeline = ref<DevopsPipeline | null>(null)
  const runTargetPipelineTektonConfig = ref<TektonDagConfig | null>(null)
  const currentRun = ref<DevopsPipelineRun | null>(null)
  const pipelines = ref<DevopsPipeline[]>([])
  const systems = ref<DevopsSystem[]>([])
  const environments = ref<DevopsPipelineEnvironment[]>([])
  const templates = ref<DevopsPipelineTemplate[]>([])
  const categories = ref<DevopsStepCategory[]>([])
  const stepLibrary = ref<DevopsStepTemplate[]>([])
  const allStepLibrary = ref<DevopsStepTemplate[]>([])
  const activeStepLibraryCategoryId = ref('')
  const tektonRuntimeServiceAccounts = ref<TektonRuntimeResource[]>([])
  const tektonRuntimeStorageClasses = ref<TektonRuntimeResource[]>([])
  const tektonRuntimeTasks = ref<TektonRuntimeResource[]>([])
  const tektonRuntimeSecrets = ref<TektonRuntimeResource[]>([])
  const tektonRuntimePvcs = ref<TektonRuntimeResource[]>([])
  const tektonRuntimeConfigMaps = ref<TektonRuntimeResource[]>([])
  const tektonWorkspaceCredentials = ref<DevopsCredential[]>([])
  const tektonWorkspaceProjectConfigs = ref<DevopsProjectConfig[]>([])
  const buildBindings = ref<DevopsProjectChannelBinding[]>([])
  const pipelineChannelBindings = ref<DevopsProjectChannelBinding[]>([])
  const pipelineChannelBindingLoadedGroups = reactive<Record<string, boolean>>({})
  const pipelineConfigParamOptions = reactive<Record<string, DynamicSelectOption[]>>({})
  const pipelineConfigParamLoading = reactive<Record<string, boolean>>({})
  const pipelineConfigParamLoaded = reactive<Record<string, boolean>>({})
  const pipelineConfigParamError = reactive<Record<string, string>>({})
  const pipelineConfigParamQueryKey = reactive<Record<string, string>>({})
  const jenkinsAgents = ref<JenkinsAgentOption[]>([])
  const agentLoading = ref(false)
  const workflowDesignerRef = ref<{
    fitCurrentView?: () => void
    fitCurrentViewAfterLayout?: () => Promise<void>
  }>()
  const customTemplateSourceKey = '__custom__'
  type TemplateSourceFilter = 'all' | 'custom' | 'project' | 'system'
  const templateSourceKey = ref('')
  const templateChoiceId = ref('')
  const templateKeyword = ref('')
  const templateSourceFilter = ref<TemplateSourceFilter>('all')
  const isTektonTemplateSource = computed(() => currentEngineType.value === 'tekton')
  const isCustomTemplateSource = computed(() => templateSourceKey.value === customTemplateSourceKey)
  const selectedTemplateChoice = computed(
    () => templates.value.find((item) => item.id === templateSourceKey.value) || null
  )
  const templateSourceConfirmDisabled = computed(
    () => !isCustomTemplateSource.value && !selectedTemplateChoice.value
  )
  const normalizeTemplateSteps = (steps?: PipelineTemplateStep[]) => {
    return [...(steps || [])]
      .sort((a, b) => (a.sortOrder || 0) - (b.sortOrder || 0))
      .map((step) => ({
        name: step.nodeName || step.stepName || step.stepCode || step.stepId || '未命名步骤'
      }))
  }
  const templateStepPreview = (item: DevopsPipelineTemplate) =>
    normalizeTemplateSteps(item.steps).slice(0, 4)
  const templateKeywordText = computed(() => templateKeyword.value.trim().toLowerCase())
  const templateSearchText = (item: DevopsPipelineTemplate) =>
    [
      item.name,
      item.code,
      item.description,
      item.scope,
      item.engineType,
      ...normalizeTemplateSteps(item.steps).map((step) => step.name)
    ]
      .filter(Boolean)
      .join(' ')
      .toLowerCase()
  const templateMatchesKeyword = (item: DevopsPipelineTemplate) => {
    const keyword = templateKeywordText.value
    return !keyword || templateSearchText(item).includes(keyword)
  }
  const customTemplateMatchesKeyword = computed(() => {
    const keyword = templateKeywordText.value
    if (!keyword) return true
    return ['自定义流水线', 'custom', '空白创建', 'jenkins'].some((text) =>
      text.toLowerCase().includes(keyword)
    )
  })
  const templateSourceFilterOptions = computed(() => {
    const options = [
      { label: '全部', value: 'all' },
      { label: '自定义', value: 'custom' },
      { label: '项目模板', value: 'project' },
      { label: '系统模板', value: 'system' }
    ]
    return isTektonTemplateSource.value
      ? options.filter((item) => item.value !== 'custom')
      : options
  })
  const showCustomTemplateSource = computed(
    () =>
      !isTektonTemplateSource.value &&
      customTemplateMatchesKeyword.value &&
      ['all', 'custom'].includes(templateSourceFilter.value)
  )
  const filteredTemplateSourceItems = computed(() => {
    return templates.value.filter((item) => {
      if (templateSourceFilter.value === 'custom') return false
      if (templateSourceFilter.value === 'project' && item.scope !== 'project') return false
      if (templateSourceFilter.value === 'system' && item.scope === 'project') return false
      return templateMatchesKeyword(item)
    })
  })
  const templateSourceResultCount = computed(
    () => filteredTemplateSourceItems.value.length + (showCustomTemplateSource.value ? 1 : 0)
  )
  const selectedTemplateVisibleInList = computed(() => {
    if (isCustomTemplateSource.value) return showCustomTemplateSource.value
    if (!templateSourceKey.value) return false
    return filteredTemplateSourceItems.value.some((item) => item.id === templateSourceKey.value)
  })
  const selectTemplateSource = (value: string) => {
    templateSourceKey.value = value
    templateChoiceId.value = value === customTemplateSourceKey ? '' : value
  }
  const currentSteps = ref<WorkflowNode[]>([])
  const runs = ref<DevopsPipelineRun[]>([])
  const runStages = ref<DevopsPipelineRunStage[]>([])
  const runArtifacts = ref<DevopsPipelineRunArtifact[]>([])
  const qualitySummary = ref<QualityRunSummaryResponse | null>(null)
  const runInputs = ref<DevopsPipelineRunInput[]>([])
  const processPipelineSteps = ref<PipelineTemplateStep[]>([])
  const processGraphInstance = ref<VueFlowStore>()
  const processGraphNeedInitialCenter = ref(false)
  const processGraphZoom = ref(1)
  const processGraphInitialViewport = ref<ViewportTransform>({ x: 0, y: 0, zoom: 1 })
  const processGraphViewportWidth = ref(0)
  const activeLogTab = ref<'stage' | 'full' | 'input' | 'artifact' | 'quality'>('stage')
  const activeQualityTab = ref<'reports' | 'static' | 'image' | 'gate'>('reports')
  const currentStageId = ref('')
  const tektonLogTaskRunName = ref('')
  const tektonLogContainerName = ref('')
  const runParamItems = ref<PipelineParam[]>([])
  const runWorkspaceItems = ref<TektonRunWorkspaceItem[]>([])
  const tektonWorkspaceConfigItems = ref<TektonProjectWorkspaceItem[]>([])
  const editingTektonWorkspaceItem = computed(() =>
    editingTektonWorkspaceIndex.value >= 0
      ? tektonWorkspaceConfigItems.value[editingTektonWorkspaceIndex.value]
      : undefined
  )
  const runLog = ref('')
  const processStateLoading = ref(false)
  const processGraphScrollRef = ref<HTMLElement>()
  const stageLogBodyRef = ref<HTMLElement>()
  const fullLogBodyRef = ref<HTMLElement>()
  const autoScrollLog = ref(true)
  const stageRuntimeTimer = ref<number>()
  const processStatePollTimer = ref<number>()
  const stageRuntimeTick = ref(Date.now())
  const workflowCanvasKey = ref(0)
  const logContentCache = reactive<Record<string, string>>({})
  let logRequestSeq = 0
  let lastInputFetchAt = 0
  let lastInputFetchRunId = ''
  let logEventSource: EventSource | null = null
  let processGraphResizeObserver: ResizeObserver | null = null
  let processGraphSyncFrame = 0
  let runParamRequestSeq = 0
  const runParamForm = reactive<Record<string, any>>({})
  const runWorkspaceForm = reactive<Record<string, string>>({})
  const inputParamForm = reactive<Record<string, Record<string, string>>>({})
  const inputProceedLoadingMap = reactive<Record<string, boolean>>({})
  const inputAbortLoadingMap = reactive<Record<string, boolean>>({})
  const dynamicRunParamOptions = reactive<Record<string, DynamicSelectOption[]>>({})
  const runParamOptionLoading = reactive<Record<string, boolean>>({})
  const dynamicRunParamLoaded = reactive<Record<string, boolean>>({})
  const dynamicRunParamError = reactive<Record<string, string>>({})
  const runParamQueryKey = reactive<Record<string, string>>({})
  const stepFilter = reactive({ categoryId: '' })
  const pagination = reactive({ current: 1, size: 12, total: 0 })
  const runPagination = reactive({ current: 1, size: 12, total: 0 })
  const defaultJenkinsOptions = (): JenkinsOption[] => [
    { name: 'timeout', value: "time: 1, unit: 'HOURS'" },
    { name: 'buildDiscarder', value: "logRotator(numToKeepStr: '10')" },
    { name: 'disableConcurrentBuilds', value: '' },
    { name: 'skipDefaultCheckout', value: 'false' }
  ]
  const optionDefinitions = [
    {
      name: 'timeout',
      title: '流水线超时时间',
      description: '默认 1 小时，超时后 Jenkins 自动终止本次构建。',
      defaultValue: "time: 1, unit: 'HOURS'",
      valueLabel: '参数'
    },
    {
      name: 'timestamps',
      title: '日志时间戳',
      description: '默认关闭，开启后控制台日志每行显示时间。',
      defaultValue: '',
      valueLabel: ''
    },
    {
      name: 'ansiColor',
      title: 'ANSI 彩色日志',
      description: '默认关闭，开启后需要 Jenkins 安装 AnsiColor 插件。',
      defaultValue: "'xterm'",
      valueLabel: '颜色映射'
    },
    {
      name: 'buildDiscarder',
      title: '构建记录保留',
      description: '默认保留最近 10 次构建，避免 Jenkins 磁盘持续膨胀。',
      defaultValue: "logRotator(numToKeepStr: '10')",
      valueLabel: '策略'
    },
    {
      name: 'disableConcurrentBuilds',
      title: '禁止并发构建',
      description: '开启后同一流水线不会同时运行多个构建。',
      defaultValue: '',
      valueLabel: ''
    },
    {
      name: 'skipDefaultCheckout',
      title: '跳过默认检出',
      description: '默认 false，Jenkins 仍按声明式流水线默认行为处理代码检出。',
      defaultValue: 'false',
      valueLabel: '布尔值'
    }
  ]
  const triggerDefinitions = [
    { type: 'cron', title: '定时触发', description: '按 Cron 表达式周期运行，例如 H 2 * * *。' },
    { type: 'pollSCM', title: '代码变更轮询', description: '按周期检查代码仓库变化后触发。' },
    { type: 'upstream', title: '上游任务触发', description: '由上游 Jenkins Job 完成后触发。' }
  ]
  const searchForm = ref<PipelineSearchForm>({
    name: undefined,
    systemId: undefined,
    environmentId: undefined
  })
  const pipelineSystemStoragePrefix = 'devops-pipeline-selected-system'
  const pipelineSystemStorageKey = (projectId = projectStore.selectedProjectId) =>
    `${pipelineSystemStoragePrefix}:${projectId}`
  const readStoredPipelineSystemId = () => {
    if (!projectStore.selectedProjectId) return ''
    try {
      return sessionStorage.getItem(pipelineSystemStorageKey()) || ''
    } catch {
      return ''
    }
  }
  const persistPipelineSystemId = (systemId?: string) => {
    if (!projectStore.selectedProjectId) return
    try {
      const key = pipelineSystemStorageKey()
      if (systemId) {
        sessionStorage.setItem(key, systemId)
      } else {
        sessionStorage.removeItem(key)
      }
    } catch {
      // 忽略浏览器存储异常
    }
  }
  const hasSystem = (systemId?: string) =>
    !!systemId && systems.value.some((item) => item.id === systemId)
  const resolvePipelineSearchSystemId = () => {
    if (hasSystem(searchForm.value.systemId)) return searchForm.value.systemId || ''
    const storedSystemId = readStoredPipelineSystemId()
    if (hasSystem(storedSystemId)) return storedSystemId
    return systems.value[0]?.id || ''
  }
  const channelParamTypes = ['repositoryChannel', 'nexusChannel', 'harborChannel', 'jfrogChannel']
  const kubeNovaDeployConfigParamType = 'kubeNovaDeployConfig'
  const kubernetesDeployConfigParamType = 'kubernetesDeployConfig'
  const dynamicParamTypes = [
    'gitProject',
    'gitBranch',
    'gitTag',
    'nexusRepository',
    'nexusArtifactName',
    'nexusArtifactVersion',
    'nexusArtifact',
    'harborProject',
    'harborImage',
    'harborImageTag',
    'registryRepository',
    'registryImage',
    'registryImageTag',
    'jfrogRepository',
    'jfrogArtifactName',
    'jfrogArtifactVersion',
    'jfrogArtifact',
    'kubeNovaProject',
    'kubeNovaCluster',
    'kubeNovaWorkspace',
    'kubeNovaApplication',
    'kubeNovaVersion',
    'kubeNovaContainer',
    'kubernetesNamespace',
    'kubernetesWorkloadType',
    'kubernetesResource',
    'kubernetesContainer',
    'kubernetesImage',
    'configCenter',
    'mavenConfig',
    'sonarAddress',
    'sonarProjectName',
    'sonarProjectKey',
    'hostGroupTargets',
    'hostConfig',
    'hostGroupConfig'
  ]
  const dynamicParamSpec: Record<
    string,
    {
      channelParamType: string
      projectParamType?: string
      projectParamTypes?: string[]
      projectLabel?: string
    }
  > = {
    gitProject: { channelParamType: 'repositoryChannel' },
    gitBranch: {
      channelParamType: 'repositoryChannel',
      projectParamType: 'gitProject',
      projectParamTypes: ['gitProject', 'gitRepositoryUrl'],
      projectLabel: 'Git 项目或仓库地址'
    },
    gitTag: {
      channelParamType: 'repositoryChannel',
      projectParamType: 'gitProject',
      projectParamTypes: ['gitProject', 'gitRepositoryUrl'],
      projectLabel: 'Git 项目或仓库地址'
    },
    nexusRepository: { channelParamType: 'nexusChannel' },
    nexusArtifactName: {
      channelParamType: 'nexusChannel',
      projectParamType: 'nexusRepository',
      projectParamTypes: [
        'nexusRepository',
        'nexusRepositoryUrl',
        'nexusArtifactUrl',
        'nexusArtifactVersionUrl'
      ],
      projectLabel: 'Nexus 仓库或制品地址'
    },
    nexusArtifactVersion: {
      channelParamType: 'nexusChannel',
      projectParamType: 'nexusRepository',
      projectParamTypes: [
        'nexusRepository',
        'nexusRepositoryUrl',
        'nexusArtifactUrl',
        'nexusArtifactVersionUrl'
      ],
      projectLabel: 'Nexus 仓库或制品地址'
    },
    nexusArtifact: {
      channelParamType: 'nexusChannel',
      projectParamType: 'nexusRepository',
      projectParamTypes: [
        'nexusRepository',
        'nexusRepositoryUrl',
        'nexusArtifactUrl',
        'nexusArtifactVersionUrl'
      ],
      projectLabel: 'Nexus 仓库或制品地址'
    },
    harborProject: { channelParamType: 'harborChannel' },
    harborImage: {
      channelParamType: 'harborChannel',
      projectParamType: 'harborProject',
      projectParamTypes: [
        'harborProject',
        'harborProjectUrl',
        'harborImageUrl',
        'harborImageTagUrl'
      ],
      projectLabel: '镜像项目或镜像地址'
    },
    harborImageTag: {
      channelParamType: 'harborChannel',
      projectParamType: 'harborProject',
      projectParamTypes: [
        'harborProject',
        'harborProjectUrl',
        'harborImageUrl',
        'harborImageTagUrl'
      ],
      projectLabel: '镜像项目或镜像地址'
    },
    registryRepository: { channelParamType: 'channelGroupCode' },
    registryImage: {
      channelParamType: 'channelGroupCode',
      projectParamType: 'registryRepository',
      projectParamTypes: [
        'registryRepository',
        'registryRepositoryUrl',
        'registryImageUrl',
        'registryImageTagUrl'
      ],
      projectLabel: '镜像项目或镜像地址'
    },
    registryImageTag: {
      channelParamType: 'channelGroupCode',
      projectParamType: 'registryRepository',
      projectParamTypes: [
        'registryRepository',
        'registryRepositoryUrl',
        'registryImageUrl',
        'registryImageTagUrl'
      ],
      projectLabel: '镜像项目或镜像地址'
    },
    jfrogRepository: { channelParamType: 'jfrogChannel' },
    jfrogArtifactName: {
      channelParamType: 'jfrogChannel',
      projectParamType: 'jfrogRepository',
      projectParamTypes: [
        'jfrogRepository',
        'jfrogRepositoryUrl',
        'jfrogArtifactUrl',
        'jfrogArtifactVersionUrl'
      ],
      projectLabel: 'JFrog 仓库或制品地址'
    },
    jfrogArtifactVersion: {
      channelParamType: 'jfrogChannel',
      projectParamType: 'jfrogRepository',
      projectParamTypes: [
        'jfrogRepository',
        'jfrogRepositoryUrl',
        'jfrogArtifactUrl',
        'jfrogArtifactVersionUrl'
      ],
      projectLabel: 'JFrog 仓库或制品地址'
    },
    jfrogArtifact: {
      channelParamType: 'jfrogChannel',
      projectParamType: 'jfrogRepository',
      projectParamTypes: [
        'jfrogRepository',
        'jfrogRepositoryUrl',
        'jfrogArtifactUrl',
        'jfrogArtifactVersionUrl'
      ],
      projectLabel: 'JFrog 仓库或制品地址'
    },
    kubernetesNamespace: { channelParamType: 'channelGroupCode' },
    kubernetesWorkloadType: { channelParamType: 'channelGroupCode' },
    kubernetesResource: {
      channelParamType: 'channelGroupCode',
      projectParamType: 'kubernetesNamespace',
      projectLabel: '命名空间'
    },
    kubernetesContainer: {
      channelParamType: 'channelGroupCode',
      projectParamType: 'kubernetesResource',
      projectLabel: '部署资源'
    },
    kubernetesImage: {
      channelParamType: 'channelGroupCode',
      projectParamType: 'kubernetesContainer',
      projectLabel: '容器'
    },
    configCenter: { channelParamType: '' },
    mavenConfig: { channelParamType: '' },
    sonarAddress: { channelParamType: '' },
    sonarProjectName: { channelParamType: 'sonarAddress' },
    sonarProjectKey: { channelParamType: 'sonarAddress' },
    hostGroupTargets: { channelParamType: '' },
    hostConfig: { channelParamType: '' },
    hostGroupConfig: { channelParamType: '' }
  }
  const tektonPlatformParamTypes = [
    'channelGroupCode',
    'voucherModel',
    'channelCredential',
    'configCenter',
    'hostGroupTargets'
  ]
  const isTektonEngine = () => formData.engineType === 'tekton'
  const isTektonTemplateLocked = computed(() => isTektonEngine())
  const isPlatformPipelineParam = (item: PipelineParam) =>
    item.mode === 'paas' ||
    (isTektonEngine() &&
      (tektonPlatformParamTypes.includes(item.paramType) ||
        isCompositePlatformParamType(item.paramType)))
  const isCompositePipelineParam = (item: PipelineParam) =>
    isPlatformPipelineParam(item) && isCompositePlatformParamType(platformAddressParamType(item))
  const isFullRowParam = (item: PipelineParam) =>
    item.config?.fullRow === true ||
    item.paramType === 'objectList' ||
    isKubeNovaDeployConfigParam(item) ||
    isKubernetesDeployConfigParam(item)
  const numberParamValue = (value?: string | number) => {
    if (value === undefined || value === null || value === '') return undefined
    const parsed = Number(value)
    return Number.isNaN(parsed) ? undefined : parsed
  }
  const setPipelineNumberValue = (item: PipelineParam, value?: number) => {
    item.currentValue = value === undefined || value === null ? '' : String(value)
  }
  const setRunNumberValue = (code: string, value?: number) => {
    runParamForm[code] = value === undefined || value === null ? '' : String(value)
  }
  const objectListFields = (item: PipelineParam): CompoundParamField[] =>
    Array.isArray(item.config?.compoundFields) ? item.config.compoundFields : []
  const objectListFieldOptions = (field: CompoundParamField) =>
    field.config?.options?.length ? field.config.options : field.selectList || []
  const objectListFieldIsSelect = (field: CompoundParamField) =>
    ['choice', 'list'].includes(field.paramType) || objectListFieldOptions(field).length > 0
  const objectListFieldComponent = (field: CompoundParamField) => {
    if (field.paramType === 'booleanParam') return 'ElSwitch'
    if (field.paramType === 'number') return 'ElInputNumber'
    if (field.paramType === 'text') return 'ElInput'
    if (objectListFieldIsSelect(field)) return 'ElSelect'
    return 'ElInput'
  }
  const objectListFieldProps = (field: CompoundParamField, disabled: boolean) => {
    if (field.paramType === 'booleanParam') return { disabled }
    if (field.paramType === 'number') {
      return {
        disabled,
        controlsPosition: 'right',
        placeholder: field.defaultValue || '请输入数字'
      }
    }
    if (field.paramType === 'text') {
      return {
        disabled,
        type: 'textarea',
        rows: 2,
        resize: 'vertical',
        placeholder: field.defaultValue || '请输入'
      }
    }
    if (objectListFieldIsSelect(field)) {
      return {
        class: 'platform-param-select',
        popperClass: 'devops-platform-param-dropdown',
        teleported: true,
        fitInputWidth: true,
        disabled,
        filterable: true,
        clearable: true,
        allowCreate: true,
        defaultFirstOption: true,
        placeholder: field.defaultValue || '请选择'
      }
    }
    return {
      disabled,
      type: field.paramType === 'password' ? 'password' : 'text',
      showPassword: field.paramType === 'password',
      placeholder: field.defaultValue || '请输入'
    }
  }
  const objectListFieldValue = (row: Record<string, any>, field: CompoundParamField) => {
    const value = row?.[field.code]
    if (field.paramType === 'booleanParam') {
      return value === true || String(value).toLowerCase() === 'true'
    }
    if (field.paramType === 'number' && value !== undefined && value !== '') return Number(value)
    return value ?? ''
  }
  const normalizeObjectListFieldValue = (field: CompoundParamField, value: any) => {
    if (field.paramType === 'booleanParam') {
      return value === true || String(value).toLowerCase() === 'true'
    }
    if (field.paramType === 'number') {
      return value === undefined || value === null || value === '' ? '' : String(value)
    }
    return value ?? ''
  }
  const parseObjectListRows = (raw?: unknown): Record<string, any>[] => {
    if (Array.isArray(raw)) {
      return raw.filter((item) => item && typeof item === 'object') as Record<string, any>[]
    }
    const text = String(raw || '').trim()
    if (!text) return []
    try {
      const parsed = JSON.parse(text)
      return Array.isArray(parsed)
        ? (parsed.filter((item) => item && typeof item === 'object') as Record<string, any>[])
        : []
    } catch {
      return []
    }
  }
  const stringifyObjectListRows = (rows: Record<string, any>[]) =>
    rows.length > 0 ? JSON.stringify(rows) : ''
  const pipelineObjectListRows = (item: PipelineParam) =>
    parseObjectListRows(item.currentValue || item.defaultValue)
  const runObjectListRows = (item: PipelineParam) => parseObjectListRows(runParamForm[item.code])
  const blankObjectListRow = (item: PipelineParam) =>
    objectListFields(item).reduce<Record<string, any>>((acc, field) => {
      acc[field.code] =
        field.paramType === 'booleanParam'
          ? String(field.defaultValue || '').toLowerCase() === 'true'
          : field.defaultValue || ''
      return acc
    }, {})
  const setPipelineObjectListRows = (item: PipelineParam, rows: Record<string, any>[]) => {
    item.currentValue = stringifyObjectListRows(rows)
  }
  const setRunObjectListRows = (item: PipelineParam, rows: Record<string, any>[]) => {
    runParamForm[item.code] = stringifyObjectListRows(rows)
  }
  const addPipelineObjectListRow = (item: PipelineParam) => {
    setPipelineObjectListRows(item, [...pipelineObjectListRows(item), blankObjectListRow(item)])
  }
  const removePipelineObjectListRow = (item: PipelineParam, index: number) => {
    const rows = [...pipelineObjectListRows(item)]
    rows.splice(index, 1)
    setPipelineObjectListRows(item, rows)
  }
  const updatePipelineObjectListFieldValue = (
    item: PipelineParam,
    rowIndex: number,
    fieldCode: string,
    value: any
  ) => {
    const rows = [...pipelineObjectListRows(item)]
    const field = objectListFields(item).find((target) => target.code === fieldCode)
    if (!rows[rowIndex]) rows[rowIndex] = {}
    rows[rowIndex] = {
      ...rows[rowIndex],
      [fieldCode]: field ? normalizeObjectListFieldValue(field, value) : value
    }
    setPipelineObjectListRows(item, rows)
  }
  const addRunObjectListRow = (item: PipelineParam) => {
    setRunObjectListRows(item, [...runObjectListRows(item), blankObjectListRow(item)])
  }
  const removeRunObjectListRow = (item: PipelineParam, index: number) => {
    const rows = [...runObjectListRows(item)]
    rows.splice(index, 1)
    setRunObjectListRows(item, rows)
  }
  const updateRunObjectListFieldValue = (
    item: PipelineParam,
    rowIndex: number,
    fieldCode: string,
    value: any
  ) => {
    const rows = [...runObjectListRows(item)]
    const field = objectListFields(item).find((target) => target.code === fieldCode)
    if (!rows[rowIndex]) rows[rowIndex] = {}
    rows[rowIndex] = {
      ...rows[rowIndex],
      [fieldCode]: field ? normalizeObjectListFieldValue(field, value) : value
    }
    setRunObjectListRows(item, rows)
  }
  const validateObjectListRows = (item: PipelineParam, raw: unknown) => {
    if (objectListFields(item).length === 0) return `对象列表「${item.name}」未配置字段`
    const text = String(raw || '').trim()
    if (text) {
      try {
        if (!Array.isArray(JSON.parse(text))) {
          return `对象列表「${item.name}」必须是 JSON 数组`
        }
      } catch {
        return `对象列表「${item.name}」必须是合法 JSON 数组`
      }
    }
    const rows = parseObjectListRows(raw)
    if (item.required && rows.length === 0) return `请配置必填对象列表：${item.name}`
    for (const row of rows) {
      for (const field of objectListFields(item)) {
        const value = row[field.code]
        const empty = value === undefined || value === null || String(value).trim() === ''
        if (field.required && empty && !String(field.defaultValue || '').trim()) {
          return `对象列表「${item.name}」缺少必填字段：${field.name || field.code}`
        }
        if (empty) continue
        if (field.paramType === 'number' && Number.isNaN(Number(value))) {
          return `对象列表「${item.name}」字段「${field.name || field.code}」必须是合法数字`
        }
      }
    }
    return ''
  }
  const orderedParams = <T extends { sortOrder?: number }>(items: T[]) =>
    items
      .map((item, index) => ({
        item,
        index,
        order: Number(item.sortOrder || 0) > 0 ? Number(item.sortOrder) : index + 1
      }))
      .sort((left, right) => left.order - right.order || left.index - right.index)
      .map(({ item }) => item)
  const orderedRuntimeParams = (items: PipelineParam[], steps: PipelineTemplateStep[]) => {
    const stepOrder = new Map<string, number>()
    steps
      .map((step, index) => ({
        step,
        index,
        order: Number(step.sortOrder || 0) > 0 ? Number(step.sortOrder) : index + 1
      }))
      .sort((left, right) => left.order - right.order || left.index - right.index)
      .forEach(({ step }, index) => {
        if (step.id) stepOrder.set(step.id, index)
      })
    return items
      .map((item, index) => ({
        item,
        index,
        stepOrder: item.stepNodeId
          ? (stepOrder.get(item.stepNodeId) ?? Number.MAX_SAFE_INTEGER)
          : Number.MAX_SAFE_INTEGER,
        paramOrder: Number(item.sortOrder || 0) > 0 ? Number(item.sortOrder) : index + 1
      }))
      .sort(
        (left, right) =>
          left.stepOrder - right.stepOrder ||
          left.paramOrder - right.paramOrder ||
          left.index - right.index
      )
      .map(({ item }) => item)
  }

  const defaultForm = (): DevopsPipelineForm => ({
    projectId: projectStore.selectedProjectId,
    systemId: resolvePipelineSearchSystemId(),
    environmentId: '',
    name: '',
    code: '',
    engineType: 'jenkins',
    templateId: '',
    buildChannelBindingId: '',
    steps: [],
    params: [],
    agent: { type: 'any' },
    tools: [],
    options: defaultJenkinsOptions(),
    triggers: [],
    triggerMode: 'manual',
    scanEnabled: false,
    scanMode: 'strict',
    rejectUnexpectedReports: true,
    enforce: false,
    scanItems: [],
    tektonDagConfig: '',
    tektonRunPolicy: '',
    tektonTriggerConfig: '',
    tektonPrunerPolicyRef: '',
    tektonTemplateId: '',
    tektonTemplateSnapshot: '',
    tektonParamBindings: '',
    tektonWorkspaceBindings: '',
    tektonResourceBindings: '',
    status: 1
  })

  const formData = reactive<DevopsPipelineForm>(defaultForm())
  watch(
    [filteredTemplateSourceItems, showCustomTemplateSource],
    () => {
      if (selectedTemplateVisibleInList.value) return
      if (showCustomTemplateSource.value) {
        selectTemplateSource(customTemplateSourceKey)
        return
      }
      selectTemplateSource(filteredTemplateSourceItems.value[0]?.id || '')
    },
    { flush: 'post' }
  )
  const defaultTektonDagConfig = (): TektonDagConfig => ({
    version: 'tekton-dag/v1',
    pipeline: {
      name: formData.code || 'tekton-pipeline',
      namespace: '',
      serviceAccountName: '',
      timeout: ''
    },
    params: [],
    workspaces: [{ name: 'source', bindingType: 'emptyDir' }],
    nodes: [],
    edges: [],
    runPolicy: {
      managedBy: '',
      workspaceStrategy: 'emptyDir',
      concurrencyPolicy: 'allow',
      cleanupImage: '',
      labels: [],
      annotations: [],
      podTemplateEnv: [],
      podTemplateRaw: '',
      taskRunTemplateRaw: '',
      taskRunSpecsRaw: ''
    },
    triggerConfig: {
      enabled: false,
      apiVersion: 'triggers.tekton.dev/v1beta1',
      provider: 'generic',
      serviceType: 'ClusterIP',
      interceptorType: 'none',
      secretKey: 'token',
      eventTypes: [],
      bindingParams: []
    },
    prunerPolicy: {
      enabled: false,
      clusterResourceCleanup: true,
      platformRecordCleanup: true,
      nativePrunerMode: 'disabled',
      nativePrunerTargetNamespace: '',
      nativePrunerConfigLevel: 'namespace',
      resourceTypes: ['PipelineRun', 'TaskRun'],
      scope: 'pipeline',
      historyLimit: 20,
      successfulHistoryLimit: 10,
      failedHistoryLimit: 10,
      ttlSecondsAfterFinished: 604800,
      platformRecordRetentionDays: 30,
      platformRecordLimit: 200,
      selectorLabels: [],
      selectorAnnotations: []
    },
    scheduleConfig: {
      enabled: false,
      timezone: 'Asia/Shanghai',
      concurrencyPolicy: 'skip',
      parameterSnapshot: true,
      paramsSnapshot: {}
    }
  })
  const tektonDagConfig = ref<TektonDagConfig>(defaultTektonDagConfig())
  const normalizeJenkinsAgentType = (type?: string) => {
    const value = String(type || '').trim()
    if (value === 'pod') return 'dynamic'
    return value || 'any'
  }
  const isDynamicJenkinsAgent = (type?: string) => normalizeJenkinsAgentType(type) === 'dynamic'
  const normalizeFormAgent = (agent?: DevopsPipelineForm['agent']): DevopsPipelineForm['agent'] => {
    if (!agent?.type) return { type: 'any' }
    const type = normalizeJenkinsAgentType(agent.type)
    return {
      ...agent,
      type,
      agentType: agent.agentType ? normalizeJenkinsAgentType(agent.agentType) : type
    }
  }
  const rules: FormRules = {
    systemId: [{ required: true, message: '请选择应用', trigger: 'change' }],
    environmentId: [{ required: true, message: '请选择归属环境', trigger: 'change' }],
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    code: [{ required: true, message: '请输入编码', trigger: 'blur' }],
    engineType: [{ required: true, message: '请选择引擎', trigger: 'change' }],
    buildChannelBindingId: [{ required: true, message: '请选择构建渠道', trigger: 'change' }]
  }

  const isEditorMode = computed(() => editorMode.value !== 'browse')
  const isRecordsMode = computed(() => editorMode.value === 'records')
  const isProcessMode = computed(() => editorMode.value === 'process')
  const isViewMode = computed(() => editorMode.value === 'view')
  const selectedPlatformAgent = computed(() =>
    formData.agent?.id
      ? jenkinsAgents.value.find((item) => item.id === formData.agent.id) || null
      : null
  )
  const agentContainerOptions = computed(() => {
    const selectedContainers = selectedPlatformAgent.value?.containers || []
    const storedContainers = formData.agent?.containers || []
    const containers = selectedContainers.length > 0 ? selectedContainers : storedContainers
    return isDynamicJenkinsAgent(formData.agent?.type)
      ? containers.map((item) => item.name).filter(Boolean)
      : []
  })
  const agentContainerEditable = computed(() => isDynamicJenkinsAgent(formData.agent?.type))
  const selectedAgentValue = computed(() => formData.agent?.id || formData.agent?.type || 'any')
  const mainConfigReadonly = computed(
    () => mainConfigContext.value === 'editor' && isViewMode.value
  )
  const mainConfigLocked = computed(
    () => mainConfigReadonly.value || mainConfigContext.value !== 'create'
  )
  const mainConfigTitle = computed(() => {
    if (mainConfigContext.value === 'create') return '创建流水线主配置'
    if (mainConfigReadonly.value) return '查看流水线主配置'
    return '编辑流水线主配置'
  })
  const sortedPipelineParams = computed(() => orderedParams(formData.params))
  const pipelineParamStepName = (item: PipelineParam) => {
    if (!item.stepNodeId) return ''
    const step = currentSteps.value.find((node) => node.id === item.stepNodeId)
    return step?.nodeName || step?.stepName || ''
  }
  const selectedSystem = computed(() => systems.value.find((item) => item.id === formData.systemId))
  const categoryOptions = computed(() =>
    categories.value.map((item) => ({
      label: `${item.name}（${item.code}）`,
      value: item.id,
      code: item.code
    }))
  )
  const tektonRuntimeTaskCategoryId = 'tekton-runtime-task'
  const tektonCategoryOptions = computed(() => {
    const categoryIds = new Set(
      allStepLibrary.value
        .filter((item) => item.engineType === 'tekton' && item.status === 1)
        .map((item) => item.categoryId)
        .filter(Boolean)
    )
    const options = categories.value
      .filter((item) => item.code !== 'post')
      .filter((item) => categoryIds.size === 0 || categoryIds.has(item.id))
      .map((item) => ({
        label: item.name,
        value: item.id,
        code: item.code
      }))
    if (formData.engineType !== 'tekton') return options
    return [
      ...options,
      {
        label: '渠道 Task',
        value: tektonRuntimeTaskCategoryId,
        code: tektonRuntimeTaskCategoryId
      }
    ]
  })
  const postCategoryId = computed(
    () => categories.value.find((item) => item.code === 'post')?.id || ''
  )
  const firstNormalCategoryId = computed(
    () => categories.value.find((item) => item.code !== 'post')?.id || ''
  )
  const firstTektonCategoryId = computed(
    () => tektonCategoryOptions.value[0]?.value || firstNormalCategoryId.value
  )
  const engineTypeName = (engineType?: string) => (engineType === 'tekton' ? 'Tekton' : 'Jenkins')
  const buildBindingLabel = (item: DevopsProjectChannelBinding) => {
    const name = item.channelName || item.channelCode || item.id
    const code = item.channelCode || item.channelName || item.id
    return name === code ? name : `${name}（${code}）`
  }
  const selectedBuildBinding = computed(() =>
    buildBindings.value.find((item) => item.id === formData.buildChannelBindingId)
  )
  const selectedTektonBindingConfig = computed(() =>
    parseTektonBindingConfig(selectedBuildBinding.value?.bindingConfig)
  )
  const selectedTektonChannelConfig = computed(() =>
    parseTektonChannelConfig(selectedBuildBinding.value?.channelConfig)
  )
  const parseTektonBindingConfig = (content?: string) => {
    try {
      const parsed = JSON.parse(content || '{}')
      return {
        namespace: String(parsed.namespace || ''),
        serviceAccountName: String(parsed.serviceAccountName || ''),
        pipelinePrefix: String(parsed.pipelinePrefix || '')
      }
    } catch {
      return { namespace: '', serviceAccountName: '', pipelinePrefix: '' }
    }
  }
  const parseTektonChannelConfig = (content?: string) => {
    try {
      const parsed = JSON.parse(content || '{}')
      return {
        taskNamespace: String(parsed.taskNamespace || '')
      }
    } catch {
      return { taskNamespace: '' }
    }
  }
  const parseJsonRecord = (content?: string) => {
    try {
      const parsed = JSON.parse(content || '{}')
      return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
        ? (parsed as Record<string, any>)
        : {}
    } catch {
      return {}
    }
  }
  const parseTektonDagConfig = (content?: string): TektonDagConfig => {
    try {
      const parsed = JSON.parse(content || '{}') as TektonDagConfig
      if (parsed?.version === 'tekton-dag/v1') {
        return normalizeTektonDagCompat({
          ...defaultTektonDagConfig(),
          ...parsed,
          pipeline: { ...defaultTektonDagConfig().pipeline, ...(parsed.pipeline || {}) },
          params: parsed.params || [],
          results: parsed.results || [],
          workspaces: parsed.workspaces || [],
          nodes: parsed.nodes || [],
          edges: parsed.edges || [],
          runPolicy: { ...defaultTektonDagConfig().runPolicy, ...(parsed.runPolicy || {}) },
          triggerConfig: {
            ...defaultTektonDagConfig().triggerConfig,
            ...(parsed.triggerConfig || {})
          },
          prunerPolicy: {
            ...defaultTektonDagConfig().prunerPolicy,
            ...(parsed.prunerPolicy || {})
          },
          scheduleConfig: {
            ...defaultTektonDagConfig().scheduleConfig,
            ...(parsed.scheduleConfig || {})
          }
        })
      }
    } catch {
      // 忽略旧数据，使用默认 DAG。
    }
    return defaultTektonDagConfig()
  }
  const normalizeTektonNameValueList = (value: unknown) => {
    if (Array.isArray(value)) {
      return value
        .map((item) => ({
          name: trimTektonText((item as any)?.name),
          value: trimTektonText((item as any)?.value)
        }))
        .filter((item) => item.name && item.value)
    }
    if (value && typeof value === 'object') {
      return Object.entries(value as Record<string, unknown>)
        .map(([name, itemValue]) => ({
          name: trimTektonText(name),
          value: trimTektonText(itemValue)
        }))
        .filter((item) => item.name && item.value)
    }
    return []
  }
  const normalizeTektonTriggerParamList = (value: unknown) => {
    if (Array.isArray(value)) {
      return value
        .map((item) => ({
          name: trimTektonText((item as any)?.name),
          value: trimTektonText((item as any)?.value),
          description: trimTektonText((item as any)?.description)
        }))
        .filter((item) => item.name)
    }
    if (value && typeof value === 'object') {
      return Object.entries(value as Record<string, unknown>)
        .map(([name, itemValue]) => {
          if (itemValue && typeof itemValue === 'object' && !Array.isArray(itemValue)) {
            return {
              name: trimTektonText(name),
              value: trimTektonText((itemValue as any).value),
              description: trimTektonText((itemValue as any).description)
            }
          }
          return {
            name: trimTektonText(name),
            value: trimTektonText(itemValue),
            description: ''
          }
        })
        .filter((item) => item.name)
    }
    return []
  }
  const normalizeTektonNodeType = (type?: string) => {
    switch (trimTektonText(type).toLowerCase()) {
      case 'finallytask':
        return 'finally'
      case 'clustertask':
        return 'clusterTask'
      case 'resolvertask':
        return 'resolverTask'
      case 'finally':
      case 'task':
      case 'matrix':
      case 'note':
        return trimTektonText(type) || 'task'
      default:
        return trimTektonText(type) || 'task'
    }
  }
  const normalizeTektonTaskRefKind = (kind?: string, nodeType?: string) => {
    const value = trimTektonText(kind).toLowerCase()
    if (value === 'clustertask') return 'ClusterTask'
    if (value === 'task') return 'Task'
    if (!value && nodeType === 'clusterTask') return 'ClusterTask'
    if (!value) return 'Task'
    return trimTektonText(kind)
  }
  const normalizeTektonTextValue = (value: unknown) => {
    if (value === undefined || value === null) return ''
    if (typeof value === 'string') return value
    if (Array.isArray(value) || (value && typeof value === 'object')) {
      return JSON.stringify(value, null, 2)
    }
    return String(value)
  }
  const normalizeLegacyResultRef = (ref: unknown) => {
    if (typeof ref === 'string') {
      const parts = ref
        .split('.')
        .map((item) => trimTektonText(item))
        .filter(Boolean)
      if (parts.length >= 2) return { taskName: parts[0], resultName: parts[1] }
    }
    if (ref && typeof ref === 'object' && !Array.isArray(ref)) {
      const value = ref as Record<string, unknown>
      const taskName = trimTektonText(value.taskName || value.sourceTask || value.task)
      const resultName = trimTektonText(value.resultName || value.result)
      if (taskName && resultName) return { taskName, resultName }
    }
    return null
  }
  const normalizeTektonNodeParams = (raw: unknown) => {
    const toParam = (source: any, fallbackName = '') => {
      if (!source || typeof source !== 'object' || Array.isArray(source)) {
        if (!fallbackName) return null
        return {
          name: fallbackName,
          source: 'fixed',
          value: normalizeTektonTextValue(source)
        }
      }
      const item = source as Record<string, unknown>
      const name = trimTektonText(item.name || item.code || fallbackName)
      if (!name) return null
      const rawSource = trimTektonText(item.source).toLowerCase()
      const legacyResult = normalizeLegacyResultRef(item.resultRef)
      let nextSource = rawSource
      if (legacyResult) nextSource = 'result'
      if (nextSource === 'param') nextSource = 'pipelineparam'
      if (!['pipelineparam', 'runtime', 'result', 'dynamic'].includes(nextSource)) {
        nextSource = 'fixed'
      }
      return {
        ...item,
        name,
        source:
          nextSource === 'pipelineparam'
            ? 'pipelineParam'
            : nextSource === 'runtime'
              ? 'runtime'
              : nextSource === 'result'
                ? 'result'
                : nextSource === 'dynamic'
                  ? 'dynamic'
                  : 'fixed',
        pipelineParam: trimTektonText(item.pipelineParam || item.paramName || item.sourceParam),
        value:
          item.value !== undefined
            ? normalizeTektonTextValue(item.value)
            : normalizeTektonTextValue(item.default),
        resultNodeId: trimTektonText(item.resultNodeId || legacyResult?.taskName),
        resultName: trimTektonText(item.resultName || legacyResult?.resultName),
        defaultValue: normalizeTektonTextValue(item.defaultValue || item.default)
      }
    }
    if (Array.isArray(raw)) return raw.map((item) => toParam(item)).filter(Boolean)
    if (raw && typeof raw === 'object') {
      return Object.keys(raw as Record<string, unknown>)
        .sort()
        .map((key) => toParam((raw as Record<string, unknown>)[key], key))
        .filter(Boolean)
    }
    return []
  }
  const normalizeTektonNodeWorkspaces = (raw: unknown) => {
    const toWorkspace = (source: any, fallbackName = '') => {
      if (!source || typeof source !== 'object' || Array.isArray(source)) {
        const taskWorkspace = trimTektonText(fallbackName)
        const pipelineWorkspace = trimTektonText(source)
        if (!taskWorkspace || !pipelineWorkspace) return null
        return { taskWorkspace, pipelineWorkspace }
      }
      const item = source as Record<string, unknown>
      const taskWorkspace = trimTektonText(item.taskWorkspace || item.name || fallbackName)
      const pipelineWorkspace = trimTektonText(
        item.pipelineWorkspace || item.workspace || item.workspaceName
      )
      if (!taskWorkspace || !pipelineWorkspace) return null
      return {
        ...item,
        taskWorkspace,
        pipelineWorkspace,
        subPath: trimTektonText(item.subPath),
        bindingParamCode: trimTektonText(item.bindingParamCode)
      }
    }
    if (Array.isArray(raw)) return raw.map((item) => toWorkspace(item)).filter(Boolean)
    if (raw && typeof raw === 'object') {
      return Object.keys(raw as Record<string, unknown>)
        .sort()
        .map((key) => toWorkspace((raw as Record<string, unknown>)[key], key))
        .filter(Boolean)
    }
    return []
  }
  const normalizeTektonEdges = (edges: TektonDagConfig['edges'] = []) =>
    (edges || []).map((edge) => ({
      ...edge,
      targetParam: trimTektonText(
        (edge as any).targetParam || (edge as any).targetParamName || (edge as any).paramName
      ),
      whenInput: trimTektonText((edge as any).whenInput || (edge as any).condition),
      whenOperator: trimTektonText((edge as any).whenOperator) === 'notin' ? 'notin' : 'in',
      whenValues: Array.isArray((edge as any).whenValues)
        ? (edge as any).whenValues.map((item: unknown) => trimTektonText(item)).filter(Boolean)
        : trimTektonText((edge as any).whenValues)
            .split(',')
            .map((item) => item.trim())
            .filter(Boolean)
    }))
  const resolveTektonLegacyResultNodeRefs = (nodes: TektonDagConfig['nodes']) => {
    const idSet = new Set(nodes.map((node) => trimTektonText(node.id)).filter(Boolean))
    const nameToId = new Map(
      nodes
        .map((node) => [trimTektonText(node.name), trimTektonText(node.id)] as const)
        .filter(([name, id]) => name && id)
    )
    nodes.forEach((node) => {
      node.params = (node.params || []).map((param) => {
        if (param.source !== 'result') return param
        const current = trimTektonText(param.resultNodeId)
        if (!current || idSet.has(current)) return param
        const resolved = nameToId.get(current)
        return resolved ? { ...param, resultNodeId: resolved } : param
      })
    })
    return nodes
  }
  const mergeTektonItemsByName = <T extends { name: string }>(
    primary: T[] = [],
    secondary: T[] = []
  ) => {
    const merged = new Map<string, T>()
    ;[...primary, ...secondary].forEach((item) => {
      const name = trimTektonText(item?.name)
      if (!name || merged.has(name)) return
      merged.set(name, { ...item, name })
    })
    return Array.from(merged.values())
  }
  const mergeTektonNameValueGroups = (...groups: unknown[]) => {
    const merged = new Map<string, { name: string; value: string }>()
    groups.forEach((group) => {
      normalizeTektonNameValueList(group).forEach((item) => {
        merged.set(item.name, item)
      })
    })
    return Array.from(merged.values())
  }
  const normalizeTektonDagCompat = (config: TektonDagConfig): TektonDagConfig => {
    const next = clonePlain(config) as TektonDagConfig
    const legacyFinallyNodes = (next.finallyNodes || []).map((node) => ({
      ...node,
      type: 'finally'
    }))
    const mergedNodes = new Map<string, TektonDagConfig['nodes'][number]>()
    ;[...(next.nodes || []), ...legacyFinallyNodes].forEach((node) => {
      const key = trimTektonText(node?.id || node?.name)
      if (!key || mergedNodes.has(key)) return
      mergedNodes.set(key, {
        ...node,
        name: trimTektonText(node?.name || (node as any)?.taskName),
        taskCode: trimTektonText((node as any)?.taskCode || (node as any)?.stepCode),
        type: normalizeTektonNodeType(node?.type) as TektonDagConfig['nodes'][number]['type'],
        taskRef: {
          ...node.taskRef,
          kind: normalizeTektonTaskRefKind(node.taskRef?.kind, normalizeTektonNodeType(node?.type))
        },
        params: normalizeTektonNodeParams((node as any).params),
        workspaces: normalizeTektonNodeWorkspaces((node as any).workspaces)
      })
    })
    next.nodes = resolveTektonLegacyResultNodeRefs(Array.from(mergedNodes.values()))
    next.finallyNodes = []
    next.edges = normalizeTektonEdges(next.edges || [])
    next.params = mergeTektonItemsByName(next.pipeline.params || [], next.params || [])
    next.results = mergeTektonItemsByName(next.pipeline.results || [], next.results || [])
    next.workspaces = mergeTektonItemsByName(
      next.pipeline.workspaces || [],
      mergeTektonItemsByName(next.workspaces || [], next.runPolicy?.workspaces || [])
    )
    next.pipeline.params = []
    next.pipeline.results = []
    next.pipeline.workspaces = []
    if (next.runPolicy) next.runPolicy.workspaces = []
    next.pipeline.labels = mergeTektonNameValueGroups(next.labels, next.pipeline.labels)
    next.pipeline.annotations = mergeTektonNameValueGroups(
      next.annotations,
      next.pipeline.annotations
    )
    next.labels = []
    next.annotations = []
    if (next.runPolicy) {
      next.runPolicy.labels = normalizeTektonNameValueList(next.runPolicy.labels)
      next.runPolicy.annotations = normalizeTektonNameValueList(next.runPolicy.annotations)
      next.runPolicy.podTemplateEnv = normalizeTektonNameValueList(next.runPolicy.podTemplateEnv)
    }
    if (next.triggerConfig) {
      next.triggerConfig.labels = normalizeTektonNameValueList(next.triggerConfig.labels)
      next.triggerConfig.annotations = normalizeTektonNameValueList(next.triggerConfig.annotations)
      next.triggerConfig.bindingParams = normalizeTektonTriggerParamList(
        next.triggerConfig.bindingParams
      )
      next.triggerConfig.templateParams = normalizeTektonTriggerParamList(
        next.triggerConfig.templateParams
      )
    }
    if (next.prunerPolicy) {
      next.prunerPolicy.selectorLabels = normalizeTektonNameValueList(
        next.prunerPolicy.selectorLabels
      )
      next.prunerPolicy.selectorAnnotations = normalizeTektonNameValueList(
        next.prunerPolicy.selectorAnnotations
      )
    }
    next.pipeline.timeout =
      trimTektonText(next.pipeline.timeout) ||
      trimTektonText(next.runPolicy?.timeout) ||
      trimTektonText(next.runPolicy?.pipelineTimeout) ||
      trimTektonText(next.pipeline.timeouts?.pipeline) ||
      trimTektonText(next.runPolicy?.timeouts?.pipeline)
    if (next.runPolicy) {
      next.runPolicy.timeout =
        trimTektonText(next.runPolicy.timeout) || trimTektonText(next.runPolicy.timeouts?.pipeline)
      next.runPolicy.pipelineTimeout =
        trimTektonText(next.runPolicy.pipelineTimeout) ||
        trimTektonText(next.runPolicy.timeout) ||
        trimTektonText(next.runPolicy.timeouts?.pipeline)
      next.runPolicy.tasksTimeout =
        trimTektonText(next.runPolicy.tasksTimeout) ||
        trimTektonText(next.runPolicy.timeouts?.tasks) ||
        trimTektonText(next.pipeline.timeouts?.tasks)
      next.runPolicy.finallyTimeout =
        trimTektonText(next.runPolicy.finallyTimeout) ||
        trimTektonText(next.runPolicy.timeouts?.finally) ||
        trimTektonText(next.pipeline.timeouts?.finally)
      if (
        !trimTektonText(next.runPolicy.taskRunSpecsRaw) &&
        Array.isArray(next.runPolicy.taskRunSpecs) &&
        next.runPolicy.taskRunSpecs.length > 0
      ) {
        next.runPolicy.taskRunSpecsRaw = JSON.stringify(next.runPolicy.taskRunSpecs, null, 2)
      }
    }
    return next
  }
  const parseTektonDagConfigFromPipeline = (row: DevopsPipeline): TektonDagConfig => {
    const config = parseTektonDagConfig(row.tektonDagConfig)
    const triggerRecord = parseJsonRecord(row.tektonTriggerConfig)
    const scheduleRecord =
      triggerRecord.scheduleConfig &&
      typeof triggerRecord.scheduleConfig === 'object' &&
      !Array.isArray(triggerRecord.scheduleConfig)
        ? (triggerRecord.scheduleConfig as Record<string, any>)
        : {}
    const triggerOnlyRecord = { ...triggerRecord }
    delete triggerOnlyRecord.scheduleConfig
    config.runPolicy = {
      ...(config.runPolicy || {}),
      ...parseJsonRecord(row.tektonRunPolicy)
    }
    config.triggerConfig = {
      ...(config.triggerConfig || {}),
      ...triggerOnlyRecord
    }
    config.scheduleConfig = {
      ...(config.scheduleConfig || {}),
      ...scheduleRecord
    }
    config.prunerPolicy = {
      ...(config.prunerPolicy || {}),
      ...parseJsonRecord(row.tektonPrunerPolicyRef)
    }
    return normalizeTektonDagCompat(config)
  }
  const tektonResourceName = (value?: string) => {
    const normalized = String(value || '')
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9-]+/g, '-')
      .replace(/^-+|-+$/g, '')
      .replace(/-{2,}/g, '-')
    return normalized || 'tekton-pipeline'
  }
  const isTektonResourceName = (value?: string) =>
    /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/.test(String(value || '').trim())
  const isValidTektonScheduleCron = (value?: string) => {
    const cron = trimTektonText(value)
    return !cron || cron.startsWith('@') || cron.split(/\s+/).length === 5
  }
  const isValidTektonScheduleTimezone = (value?: string) => {
    const timezone = trimTektonText(value)
    return !timezone || /^[A-Za-z_]+\/[A-Za-z0-9_+-]+(?:\/[A-Za-z0-9_+-]+)*$/.test(timezone)
  }
  const applyTektonRuntimeDefaults = () => {
    if (formData.engineType !== 'tekton') return
    tektonDagConfig.value.pipeline.name = tektonResourceName(
      formData.code || tektonDagConfig.value.pipeline.name
    )
    tektonDagConfig.value.pipeline.namespace =
      selectedTektonBindingConfig.value.namespace || tektonDagConfig.value.pipeline.namespace
    tektonDagConfig.value.pipeline.serviceAccountName =
      selectedTektonBindingConfig.value.serviceAccountName ||
      tektonDagConfig.value.pipeline.serviceAccountName
    const pipelineTimeout = tektonDagConfig.value.runPolicy?.pipelineTimeout
    if (pipelineTimeout && !tektonDagConfig.value.pipeline.timeout) {
      tektonDagConfig.value.pipeline.timeout = pipelineTimeout
    }
  }
  const ensureTektonConfigDefaults = () => {
    const pipelineName = tektonDagConfig.value.pipeline?.name || tektonResourceName(formData.code)
    const triggerBase = tektonResourceName(pipelineName)
    tektonDagConfig.value.runPolicy = {
      workspaceStrategy: 'emptyDir',
      managedBy: '',
      concurrencyPolicy: 'allow',
      cleanupImage: '',
      labels: [],
      annotations: [],
      podTemplateEnv: [],
      podTemplateRaw: '',
      taskRunTemplateRaw: '',
      taskRunSpecsRaw: '',
      ...(tektonDagConfig.value.runPolicy || {})
    }
    tektonDagConfig.value.triggerConfig = {
      enabled: false,
      apiVersion: 'triggers.tekton.dev/v1beta1',
      provider: 'generic',
      serviceType: 'ClusterIP',
      interceptorType: 'none',
      secretKey: 'token',
      eventTypes: [],
      bindingParams: [],
      templateParams: [],
      labels: [],
      annotations: [],
      ...(tektonDagConfig.value.triggerConfig || {})
    }
    tektonDagConfig.value.triggerConfig.namespace =
      tektonDagConfig.value.triggerConfig.namespace ||
      tektonDagConfig.value.pipeline?.namespace ||
      ''
    tektonDagConfig.value.triggerConfig.eventListenerName =
      tektonDagConfig.value.triggerConfig.eventListenerName ||
      tektonResourceName(`${triggerBase}-el`)
    tektonDagConfig.value.triggerConfig.triggerName =
      tektonDagConfig.value.triggerConfig.triggerName ||
      tektonResourceName(`${triggerBase}-trigger`)
    tektonDagConfig.value.triggerConfig.triggerBindingName =
      tektonDagConfig.value.triggerConfig.triggerBindingName ||
      tektonResourceName(`${triggerBase}-binding`)
    tektonDagConfig.value.triggerConfig.triggerTemplateName =
      tektonDagConfig.value.triggerConfig.triggerTemplateName ||
      tektonResourceName(`${triggerBase}-template`)
    tektonDagConfig.value.prunerPolicy = {
      enabled: false,
      clusterResourceCleanup: true,
      platformRecordCleanup: true,
      nativePrunerMode: 'disabled',
      nativePrunerTargetNamespace: '',
      nativePrunerConfigLevel: 'namespace',
      resourceTypes: ['PipelineRun', 'TaskRun'],
      scope: 'pipeline',
      historyLimit: 20,
      successfulHistoryLimit: 10,
      failedHistoryLimit: 10,
      ttlSecondsAfterFinished: 604800,
      platformRecordRetentionDays: 30,
      platformRecordLimit: 200,
      selectorLabels: [],
      selectorAnnotations: [],
      ...(tektonDagConfig.value.prunerPolicy || {})
    }
    tektonDagConfig.value.scheduleConfig = {
      enabled: false,
      timezone: 'Asia/Shanghai',
      concurrencyPolicy: 'skip',
      parameterSnapshot: true,
      paramsSnapshot: {},
      ...(tektonDagConfig.value.scheduleConfig || {})
    }
  }
  const trimTektonText = (value?: unknown) => String(value || '').trim()
  const isValidTektonTaskRunSpecsRaw = (value?: string) => {
    const raw = trimTektonText(value)
    if (!raw) return true
    try {
      const parsed = parseYaml(raw)
      return Array.isArray(parsed) || (parsed && typeof parsed === 'object')
    } catch {
      return false
    }
  }
  const isValidTektonYamlObjectRaw = (value?: string) => {
    const raw = trimTektonText(value)
    if (!raw) return true
    try {
      const parsed = parseYaml(raw)
      return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
    } catch {
      return false
    }
  }
  const ensureTektonTriggerParamDefaults = () => {
    ensureTektonConfigDefaults()
    const triggerConfig = tektonDagConfig.value.triggerConfig!
    const current = triggerConfig.bindingParams || []
    const seen = new Set(current.map((item) => trimTektonText(item.name)).filter(Boolean))
    const dagParams = (tektonDagConfig.value.params || []).map((item) => ({
      name: item.name,
      description: item.description
    }))
    const formParams = (formData.params || []).map((item) => ({
      name: item.code,
      description: item.name || item.description
    }))
    ;[...dagParams, ...formParams].forEach((item) => {
      const name = trimTektonText(item.name)
      if (!name || seen.has(name)) return
      seen.add(name)
      current.push({
        name,
        value: `$(body.${name})`,
        description: trimTektonText(item.description)
      })
    })
    triggerConfig.bindingParams = current
  }
  const validateTektonPolicyConfigBeforeSave = () => {
    ensureTektonConfigDefaults()
    ensureTektonScheduleParamSnapshot()
    if (tektonDagConfig.value.triggerConfig?.enabled) {
      ensureTektonTriggerParamDefaults()
      const trigger = tektonDagConfig.value.triggerConfig
      const required = [
        ['Namespace', trigger.namespace],
        ['EventListener', trigger.eventListenerName],
        ['Trigger', trigger.triggerName],
        ['TriggerBinding', trigger.triggerBindingName],
        ['TriggerTemplate', trigger.triggerTemplateName]
      ]
      const missing = required.find((item) => !trimTektonText(item[1]))
      if (missing) {
        tektonConfigActiveTab.value = 'trigger'
        ElMessage.warning(`请配置 ${missing[0]}`)
        return false
      }
      const invalidResource = required.find((item) => !isTektonResourceName(String(item[1] || '')))
      if (invalidResource) {
        tektonConfigActiveTab.value = 'trigger'
        ElMessage.warning(`Trigger ${invalidResource[0]} 名称不符合 Kubernetes 命名规范`)
        return false
      }
      if (
        trimTektonText(trigger.serviceAccountName) &&
        !isTektonResourceName(trigger.serviceAccountName)
      ) {
        tektonConfigActiveTab.value = 'trigger'
        ElMessage.warning('Trigger ServiceAccount 名称不符合 Kubernetes 命名规范')
        return false
      }
      if (trigger.interceptorType === 'cel' && !trimTektonText(trigger.celFilter)) {
        tektonConfigActiveTab.value = 'trigger'
        ElMessage.warning('请配置 CEL 过滤表达式')
        return false
      }
      if (
        (trigger.bindingParams || []).some(
          (item) => !trimTektonText(item.name) || !trimTektonText(item.value)
        )
      ) {
        tektonConfigActiveTab.value = 'trigger'
        ElMessage.warning('TriggerBinding 参数必须同时填写名称和值')
        return false
      }
      if ((trigger.templateParams || []).some((item) => !trimTektonText(item.name))) {
        tektonConfigActiveTab.value = 'trigger'
        ElMessage.warning('TriggerTemplate 参数必须填写名称')
        return false
      }
      if (
        [...(trigger.labels || []), ...(trigger.annotations || [])].some(
          (item) => !trimTektonText(item.name) || !trimTektonText(item.value)
        )
      ) {
        tektonConfigActiveTab.value = 'trigger'
        ElMessage.warning('Trigger 资源元数据必须同时填写 Key 和 Value')
        return false
      }
    }
    if (
      tektonDagConfig.value.scheduleConfig?.enabled &&
      (!trimTektonText(tektonDagConfig.value.scheduleConfig.cron) ||
        !trimTektonText(tektonDagConfig.value.scheduleConfig.timezone))
    ) {
      tektonConfigActiveTab.value = 'schedule'
      ElMessage.warning('请配置平台定时 Cron 和时区')
      return false
    }
    if (
      tektonDagConfig.value.scheduleConfig?.enabled &&
      !isValidTektonScheduleCron(tektonDagConfig.value.scheduleConfig.cron)
    ) {
      tektonConfigActiveTab.value = 'schedule'
      ElMessage.warning('Cron 表达式需使用 5 段格式，例如 0 2 * * *')
      return false
    }
    if (
      tektonDagConfig.value.scheduleConfig?.enabled &&
      !isValidTektonScheduleTimezone(tektonDagConfig.value.scheduleConfig.timezone)
    ) {
      tektonConfigActiveTab.value = 'schedule'
      ElMessage.warning('时区格式不合法，例如 Asia/Shanghai')
      return false
    }
    if (
      [
        ...(tektonDagConfig.value.runPolicy?.labels || []),
        ...(tektonDagConfig.value.runPolicy?.annotations || []),
        ...(tektonDagConfig.value.runPolicy?.podTemplateEnv || [])
      ].some((item) => !trimTektonText(item.name) || !trimTektonText(item.value))
    ) {
      tektonConfigActiveTab.value = 'runPolicy'
      ElMessage.warning('PipelineRun 元数据和环境变量必须同时填写 Key 和 Value')
      return false
    }
    if (!isValidTektonTaskRunSpecsRaw(tektonDagConfig.value.runPolicy?.taskRunSpecsRaw)) {
      tektonConfigActiveTab.value = 'runPolicy'
      ElMessage.warning('taskRunSpecs 必须是合法 YAML 数组或对象')
      return false
    }
    if (!isValidTektonYamlObjectRaw(tektonDagConfig.value.runPolicy?.podTemplateRaw)) {
      tektonConfigActiveTab.value = 'runPolicy'
      ElMessage.warning('podTemplate 必须是合法 YAML 对象')
      return false
    }
    if (!isValidTektonYamlObjectRaw(tektonDagConfig.value.runPolicy?.taskRunTemplateRaw)) {
      tektonConfigActiveTab.value = 'runPolicy'
      ElMessage.warning('taskRunTemplate 必须是合法 YAML 对象')
      return false
    }
    if (
      tektonDagConfig.value.prunerPolicy?.enabled &&
      !tektonDagConfig.value.prunerPolicy.resourceTypes?.length
    ) {
      tektonConfigActiveTab.value = 'pruner'
      ElMessage.warning('请选择 Pruner 清理资源类型')
      return false
    }
    if (
      tektonDagConfig.value.prunerPolicy?.enabled &&
      tektonDagConfig.value.prunerPolicy.clusterResourceCleanup === false &&
      tektonDagConfig.value.prunerPolicy.platformRecordCleanup === false
    ) {
      tektonConfigActiveTab.value = 'pruner'
      ElMessage.warning('Pruner 至少需要启用一种清理动作')
      return false
    }
    if (
      tektonDagConfig.value.prunerPolicy?.enabled &&
      tektonDagConfig.value.prunerPolicy.nativePrunerMode !== 'disabled' &&
      trimTektonText(tektonDagConfig.value.prunerPolicy.nativePrunerTargetNamespace) &&
      !isTektonResourceName(tektonDagConfig.value.prunerPolicy.nativePrunerTargetNamespace)
    ) {
      tektonConfigActiveTab.value = 'pruner'
      ElMessage.warning('Tekton 原生 Pruner 配置 Namespace 不合法')
      return false
    }
    if (
      tektonDagConfig.value.prunerPolicy?.enabled &&
      [
        ...(tektonDagConfig.value.prunerPolicy.selectorLabels || []),
        ...(tektonDagConfig.value.prunerPolicy.selectorAnnotations || [])
      ].some((item) => !trimTektonText(item.name) || !trimTektonText(item.value))
    ) {
      tektonConfigActiveTab.value = 'pruner'
      ElMessage.warning('Pruner 选择器必须同时填写 Key 和 Value')
      return false
    }
    return true
  }
  const tektonDagSummary = computed(() => {
    const nodes = tektonDagConfig.value.nodes || []
    return {
      taskCount: nodes.filter((item) => item.type === 'task' && item.enabled !== false).length,
      finallyCount: nodes.filter((item) => item.type === 'finally' && item.enabled !== false)
        .length,
      matrixCount: nodes.filter((item) => item.matrix && item.enabled !== false).length,
      workspaceCount: tektonDagConfig.value.workspaces?.length || 0,
      runtimeParamCount: (tektonDagConfig.value.params || []).filter((item) => item.runtimeConfig)
        .length
    }
  })
  const tektonWebhookPath = computed(
    () =>
      tektonDagConfig.value.triggerConfig?.webhookPath ||
      `/tekton/webhook/${tektonDagConfig.value.pipeline.name || tektonResourceName(formData.code)}`
  )
  const tektonWebhookUrl = computed(() => {
    const trigger = tektonDagConfig.value.triggerConfig
    if (trigger?.webhookUrl) return trigger.webhookUrl
    const path = tektonWebhookPath.value
    const normalizedPath = path.startsWith('/') ? path : `/${path}`
    if (trigger?.ingressHost) return `https://${trigger.ingressHost}${normalizedPath}`
    const eventListenerName = trigger?.eventListenerName
    const namespace = trigger?.namespace || tektonDagConfig.value.pipeline.namespace
    if (!eventListenerName || !namespace) return normalizedPath
    return `http://el-${eventListenerName}.${namespace}.svc.cluster.local:8080${normalizedPath}`
  })
  const isTektonDetailView = computed(
    () => isViewMode.value && formData.engineType === 'tekton' && Boolean(currentPipeline.value)
  )
  const isTektonRunContext = computed(
    () =>
      (currentRunPipeline.value?.engineType || runTargetPipeline.value?.engineType || '') ===
      'tekton'
  )
  const openTektonConfigDialog = (tab: TektonConfigTab, fromCard = false) => {
    applyTektonRuntimeDefaults()
    ensureTektonConfigDefaults()
    syncParamsFromTektonDag(false)
    if (tab === 'trigger') ensureTektonTriggerParamDefaults()
    if (tab === 'schedule') refreshTektonScheduleParamSnapshot(false)
    tektonConfigActiveTab.value = tab
    tektonConfigFromCard.value = fromCard
    tektonConfigVisible.value = true
  }
  async function openTektonYamlPreview() {
    pipelineYamlPreviewTitle.value = 'Tekton 资源预览'
    pipelineYamlPreviewTab.value = 'pipeline'
    pipelineYamlPreviewVisible.value = true
    await loadTektonYamlPreview()
  }
  const handleTektonAdvancedCommand = (command: string | number | object, fromCard = false) => {
    const value = String(command)
    if (value === 'yaml') {
      void openTektonYamlPreview()
      return
    }
    openTektonConfigDialog(value as TektonConfigTab, fromCard)
  }
  const submitTektonConfigDialog = async () => {
    ensureTektonConfigDefaults()
    if (tektonConfigFromCard.value && !validateTektonDagBeforeSave()) return
    if (!validateTektonPolicyConfigBeforeSave()) return
    if (tektonConfigFromCard.value && currentPipeline.value) {
      submitLoading.value = true
      try {
        await updateDevopsPipelineApi(currentPipeline.value.id, pipelinePayload())
        const detail = await getDevopsPipelineApi(currentPipeline.value.id)
        currentPipeline.value = detail
        currentRunPipeline.value = detail
        fillFormFromPipeline(detail)
        ElMessage.success('Tekton 配置已保存')
        await fetchPipelines()
      } finally {
        submitLoading.value = false
      }
    }
    tektonConfigVisible.value = false
    tektonConfigFromCard.value = false
  }
  const copyTektonWebhookPath = async () => {
    await navigator.clipboard.writeText(tektonWebhookUrl.value)
    ElMessage.success('已复制')
  }
  const addTektonTriggerParam = () => {
    ensureTektonConfigDefaults()
    tektonDagConfig.value.triggerConfig!.bindingParams = [
      ...(tektonDagConfig.value.triggerConfig?.bindingParams || []),
      { name: '', value: '', description: '' }
    ]
  }
  const removeTektonTriggerParam = (index: number) => {
    ensureTektonConfigDefaults()
    tektonDagConfig.value.triggerConfig!.bindingParams = (
      tektonDagConfig.value.triggerConfig?.bindingParams || []
    ).filter((_, itemIndex) => itemIndex !== index)
  }
  const addTektonTriggerTemplateParam = () => {
    ensureTektonConfigDefaults()
    tektonDagConfig.value.triggerConfig!.templateParams = [
      ...(tektonDagConfig.value.triggerConfig?.templateParams || []),
      { name: '', value: '', description: '' }
    ]
  }
  const removeTektonTriggerTemplateParam = (index: number) => {
    ensureTektonConfigDefaults()
    tektonDagConfig.value.triggerConfig!.templateParams = (
      tektonDagConfig.value.triggerConfig?.templateParams || []
    ).filter((_, itemIndex) => itemIndex !== index)
  }
  const addTektonTriggerLabel = () => {
    ensureTektonConfigDefaults()
    tektonDagConfig.value.triggerConfig!.labels = [
      ...(tektonDagConfig.value.triggerConfig?.labels || []),
      { name: '', value: '' }
    ]
  }
  const removeTektonTriggerLabel = (index: number) => {
    ensureTektonConfigDefaults()
    tektonDagConfig.value.triggerConfig!.labels = (
      tektonDagConfig.value.triggerConfig?.labels || []
    ).filter((_, itemIndex) => itemIndex !== index)
  }
  const addTektonTriggerAnnotation = () => {
    ensureTektonConfigDefaults()
    tektonDagConfig.value.triggerConfig!.annotations = [
      ...(tektonDagConfig.value.triggerConfig?.annotations || []),
      { name: '', value: '' }
    ]
  }
  const removeTektonTriggerAnnotation = (index: number) => {
    ensureTektonConfigDefaults()
    tektonDagConfig.value.triggerConfig!.annotations = (
      tektonDagConfig.value.triggerConfig?.annotations || []
    ).filter((_, itemIndex) => itemIndex !== index)
  }
  const addTektonRunPolicyEnv = () => {
    ensureTektonConfigDefaults()
    tektonDagConfig.value.runPolicy!.podTemplateEnv = [
      ...(tektonDagConfig.value.runPolicy?.podTemplateEnv || []),
      { name: '', value: '' }
    ]
  }
  const removeTektonRunPolicyEnv = (index: number) => {
    ensureTektonConfigDefaults()
    tektonDagConfig.value.runPolicy!.podTemplateEnv = (
      tektonDagConfig.value.runPolicy?.podTemplateEnv || []
    ).filter((_, itemIndex) => itemIndex !== index)
  }
  const addTektonRunPolicyLabel = () => {
    ensureTektonConfigDefaults()
    tektonDagConfig.value.runPolicy!.labels = [
      ...(tektonDagConfig.value.runPolicy?.labels || []),
      { name: '', value: '' }
    ]
  }
  const removeTektonRunPolicyLabel = (index: number) => {
    ensureTektonConfigDefaults()
    tektonDagConfig.value.runPolicy!.labels = (
      tektonDagConfig.value.runPolicy?.labels || []
    ).filter((_, itemIndex) => itemIndex !== index)
  }
  const addTektonRunPolicyAnnotation = () => {
    ensureTektonConfigDefaults()
    tektonDagConfig.value.runPolicy!.annotations = [
      ...(tektonDagConfig.value.runPolicy?.annotations || []),
      { name: '', value: '' }
    ]
  }
  const removeTektonRunPolicyAnnotation = (index: number) => {
    ensureTektonConfigDefaults()
    tektonDagConfig.value.runPolicy!.annotations = (
      tektonDagConfig.value.runPolicy?.annotations || []
    ).filter((_, itemIndex) => itemIndex !== index)
  }
  const refreshTektonScheduleParamSnapshot = (notify = true) => {
    ensureTektonConfigDefaults()
    syncParamsFromTektonDag(false)
    if (tektonDagConfig.value.scheduleConfig?.parameterSnapshot === false) {
      tektonDagConfig.value.scheduleConfig!.paramsSnapshot = {}
      return
    }
    const current = tektonDagConfig.value.scheduleConfig?.paramsSnapshot || {}
    const values = tektonPipelineParamValueMap(formData.params)
    const next: Record<string, string> = {}
    ;(tektonDagConfig.value.params || []).forEach((item) => {
      if (!item.name) return
      next[item.name] = current[item.name] ?? values[item.name] ?? item.defaultValue ?? ''
    })
    tektonDagConfig.value.scheduleConfig!.paramsSnapshot = next
    if (notify) ElMessage.success('运行配置快照已刷新')
  }
  const normalizeTektonScheduleParamSnapshot = () => {
    const schedule = tektonDagConfig.value.scheduleConfig
    if (!schedule) return
    if (schedule.parameterSnapshot === false) {
      schedule.paramsSnapshot = {}
    }
  }
  const ensureTektonScheduleParamSnapshot = () => {
    normalizeTektonScheduleParamSnapshot()
    const schedule = tektonDagConfig.value.scheduleConfig
    if (!schedule?.enabled || schedule.parameterSnapshot === false) return
    refreshTektonScheduleParamSnapshot(false)
  }
  const addTektonPrunerSelectorLabel = () => {
    ensureTektonConfigDefaults()
    tektonDagConfig.value.prunerPolicy!.selectorLabels = [
      ...(tektonDagConfig.value.prunerPolicy?.selectorLabels || []),
      { name: '', value: '' }
    ]
  }
  const removeTektonPrunerSelectorLabel = (index: number) => {
    ensureTektonConfigDefaults()
    tektonDagConfig.value.prunerPolicy!.selectorLabels = (
      tektonDagConfig.value.prunerPolicy?.selectorLabels || []
    ).filter((_, itemIndex) => itemIndex !== index)
  }
  const addTektonPrunerSelectorAnnotation = () => {
    ensureTektonConfigDefaults()
    tektonDagConfig.value.prunerPolicy!.selectorAnnotations = [
      ...(tektonDagConfig.value.prunerPolicy?.selectorAnnotations || []),
      { name: '', value: '' }
    ]
  }
  const removeTektonPrunerSelectorAnnotation = (index: number) => {
    ensureTektonConfigDefaults()
    tektonDagConfig.value.prunerPolicy!.selectorAnnotations = (
      tektonDagConfig.value.prunerPolicy?.selectorAnnotations || []
    ).filter((_, itemIndex) => itemIndex !== index)
  }
  const loadTektonYamlPreview = async (showMessage = false) => {
    if (formData.engineType !== 'tekton') return
    ensureTektonConfigDefaults()
    applyTektonRuntimeDefaults()
    syncParamsFromTektonDag(false)
    refreshTektonScheduleParamSnapshot(false)
    tektonYamlLoading.value = true
    try {
      tektonPipelineYamlPreview.value = renderTektonDagPipelineYAML(tektonDagConfig.value)
      tektonPipelineRunYamlPreview.value = renderTektonDagPipelineRunYAML(
        tektonDagConfig.value,
        undefined,
        tektonPipelineParamValueMap(formData.params)
      )
      const meta = {
        pipelineId: currentPipeline.value?.id,
        pipelineCode: formData.code || currentPipeline.value?.code,
        pipelineName: formData.name || currentPipeline.value?.name
      }
      tektonTriggerYamlPreview.value = renderTektonDagTriggerYAML(tektonDagConfig.value, meta)
      tektonScheduleYamlPreview.value = renderTektonDagSchedulePolicyYAML(
        tektonDagConfig.value,
        meta
      )
      tektonPrunerYamlPreview.value = renderTektonDagPrunerPolicyYAML(tektonDagConfig.value, meta)
      if (currentRun.value?.engineType === 'tekton') {
        try {
          const snapshot = await getDevopsPipelineRunYamlSnapshotApi(currentRun.value.id)
          if (snapshot?.tektonPipelineYamlSnapshot) {
            tektonPipelineYamlPreview.value = snapshot.tektonPipelineYamlSnapshot
          }
          if (snapshot?.tektonPipelineRunYamlSnapshot || snapshot?.pipelineSnapshot) {
            tektonPipelineRunYamlPreview.value =
              snapshot.tektonPipelineRunYamlSnapshot || snapshot.pipelineSnapshot
          }
        } catch {
          // 快照读取失败时保留当前配置预览。
        }
      }
      if (isViewMode.value && currentPipeline.value?.id) {
        try {
          const res = await previewDevopsPipelineApi(currentPipeline.value.id)
          if (res?.pipeline) tektonPipelineYamlPreview.value = res.pipeline
        } catch {
          // 预览失败时保留本地生成结果。
        }
      }
      if (showMessage) ElMessage.success('YAML 已刷新')
    } finally {
      tektonYamlLoading.value = false
    }
  }
  const editCurrentPipeline = async () => {
    if (!currentPipeline.value) return
    await enterEditor('edit', currentPipeline.value)
  }
  const runCurrentPipeline = async () => {
    if (!currentPipeline.value) return
    await runPipeline(currentPipeline.value)
  }
  const handleTektonDetailTabChange = async (tab: string) => {
    if (tab === 'records') {
      if (currentPipeline.value) currentRunPipeline.value = currentPipeline.value
      runPagination.current = 1
      await fetchRuns()
      return
    }
    if (tab === 'dag') {
      workflowCanvasKey.value += 1
    }
  }
  const categoryCodeMap = computed(() =>
    categories.value.reduce<Record<string, string>>((acc, item) => {
      acc[item.id] = item.code
      return acc
    }, {})
  )
  const visibleStepLibrary = computed(() => stepLibrary.value)
  const stepTemplateToWorkflowItem = (item: DevopsStepTemplate): WorkflowLibraryItem => ({
    id: item.id,
    name: item.name,
    code: item.code,
    categoryId: item.categoryId,
    categoryCode: categoryCodeMap.value[item.categoryId] || '',
    icon: item.icon,
    iconColor: item.iconColor,
    type: item.type,
    stageContent: item.stageContent,
    paramsCount:
      item.engineType === 'tekton' ? item.taskParams?.length || 0 : item.params?.length || 0,
    params: item.params || []
  })
  const workflowLibraryItems = computed<WorkflowLibraryItem[]>(() =>
    visibleStepLibrary.value.map(stepTemplateToWorkflowItem)
  )
  const workflowTemplateItems = computed<WorkflowLibraryItem[]>(() =>
    allStepLibrary.value.map(stepTemplateToWorkflowItem)
  )
  const tektonStepLibraryItems = computed<TektonDagLibraryItem[]>(() =>
    visibleStepLibrary.value.map((item) => ({
      ...stepTemplateToWorkflowItem(item),
      categoryName: item.categoryName,
      taskParams: item.taskParams || [],
      taskResults: item.taskResults || [],
      taskWorkspaces: item.taskWorkspaces || [],
      source: 'platform' as const,
      raw: item
    }))
  )
  const tektonRuntimeTaskLibraryItems = computed<TektonDagLibraryItem[]>(() =>
    tektonRuntimeTasks.value.map((item) => ({
      id: `${item.kind || 'Task'}:${item.namespace || 'cluster'}:${item.name}`,
      name: item.description || item.name,
      code: item.name,
      categoryId: tektonRuntimeTaskCategoryId,
      categoryName: item.namespace || 'cluster',
      categoryCode: tektonRuntimeTaskCategoryId,
      stageContent: tektonRuntimeTaskYaml(item),
      params: [],
      icon: item.kind === 'ClusterTask' ? 'simple-icons:kubernetes' : 'ri:flow-chart',
      iconColor: item.kind === 'ClusterTask' ? '#326ce5' : '#10b981',
      source: item.kind === 'ClusterTask' ? ('clusterTask' as const) : ('cluster' as const)
    }))
  )
  const tektonDagLibraryItems = computed<TektonDagLibraryItem[]>(() => [
    ...tektonStepLibraryItems.value,
    ...tektonRuntimeTaskLibraryItems.value
  ])
  const tektonServiceAccountOptions = computed<TektonResourceOption[]>(() =>
    tektonRuntimeServiceAccounts.value.map((item) => ({
      label: item.namespace ? `${item.name}（${item.namespace}）` : item.name,
      value: item.name,
      description: item.namespace
    }))
  )
  const tektonStorageClassOptions = computed<TektonResourceOption[]>(() =>
    tektonRuntimeStorageClasses.value.map((item) => ({
      label: item.provisioner ? `${item.name}（${item.provisioner}）` : item.name,
      value: item.name,
      description: item.provisioner
    }))
  )
  const tektonSecretOptions = computed<TektonResourceOption[]>(() =>
    tektonRuntimeSecrets.value.map((item) => ({
      label: item.description ? `${item.name}（${item.description}）` : item.name,
      value: item.name,
      description: item.namespace
    }))
  )
  const tektonPvcOptions = computed<TektonResourceOption[]>(() =>
    tektonRuntimePvcs.value.map((item) => ({
      label: item.description ? `${item.name}（${item.description}）` : item.name,
      value: item.name,
      description: item.namespace
    }))
  )
  const tektonConfigMapOptions = computed<TektonResourceOption[]>(() =>
    tektonRuntimeConfigMaps.value.map((item) => ({
      label: item.description ? `${item.name}（${item.description}）` : item.name,
      value: item.name,
      description: item.namespace
    }))
  )
  const tektonCredentialOptions = computed<TektonResourceOption[]>(() =>
    tektonWorkspaceCredentials.value.map((item) => ({
      label: `${item.name}（${item.credentialType}）`,
      value: item.id,
      description: item.code
    }))
  )
  const tektonProjectConfigOptions = computed<TektonResourceOption[]>(() =>
    tektonWorkspaceProjectConfigs.value.map((item) => ({
      label: `${item.name}（${item.typeName || item.typeCode || '配置'}）`,
      value: item.id,
      description: item.code
    }))
  )
  const workspaceBindingOptionList: Array<{ label: string; value: ProjectWorkspaceBindingType }> = [
    { label: 'EmptyDir', value: 'emptyDir' },
    { label: '已有 PVC', value: 'persistentVolumeClaim' },
    { label: '动态 PVC', value: 'volumeClaimTemplate' },
    { label: '已有 Secret', value: 'secret' },
    { label: '平台生成 Secret', value: 'generatedSecret' },
    { label: 'ConfigMap', value: 'configMap' },
    { label: 'Projected', value: 'projected' }
  ]
  const generatedSecretSourceOptionList = [
    { label: '平台凭证', value: 'platformCredential' },
    { label: '配置中心', value: 'configCenter' }
  ]
  const credentialTypeOptionList = [
    { label: '用户名密码', value: 'username_password' },
    { label: 'Token', value: 'token' },
    { label: 'SSH Key', value: 'ssh_key' },
    { label: 'Kubeconfig', value: 'kubeconfig' },
    { label: 'Secret Text', value: 'secret_text' },
    { label: '证书', value: 'certificate' },
    { label: 'JSON', value: 'json' }
  ]
  const tektonRuntimeTaskYaml = (item: TektonRuntimeResource) => {
    if (item.yaml) return item.yaml
    const namespace = item.namespace ? `  namespace: ${item.namespace}\n` : ''
    const params = (item.params || []).map((name) => `    - name: ${name}`).join('\n')
    const workspaces = (item.workspaces || []).map((name) => `    - name: ${name}`).join('\n')
    const results = (item.results || []).map((name) => `    - name: ${name}`).join('\n')
    return [
      `apiVersion: ${item.apiVersion || 'tekton.dev/v1'}`,
      `kind: ${item.kind || 'Task'}`,
      'metadata:',
      `  name: ${item.name}`,
      namespace.trimEnd(),
      'spec:',
      params ? `  params:\n${params}` : '',
      workspaces ? `  workspaces:\n${workspaces}` : '',
      results ? `  results:\n${results}` : ''
    ]
      .filter(Boolean)
      .join('\n')
  }
  type StepLibraryRefreshPayload = {
    categoryId?: string
  }
  const LOG_RENDER_LIMIT = 1000
  const sanitizedRunLog = computed(() =>
    runLog.value
      .replace(/\r\n/g, '\n')
      .replace(/\[Print Message\]\n?/g, '')
      .replace(/<span[^>]*style=["'][^"']*display\s*:\s*none[^"']*["'][^>]*>.*?<\/span>/gi, '')
      .replace(/<[^>]+>/g, '')
  )
  const logLineWindow = computed(() => {
    const text = sanitizedRunLog.value
    if (!text) return { startLine: 1, lines: [] as string[] }

    let start = 0
    let lineCount = 0
    for (let index = text.length - 1; index >= 0; index -= 1) {
      if (text.charCodeAt(index) !== 10) continue
      lineCount += 1
      if (lineCount >= LOG_RENDER_LIMIT) {
        start = index + 1
        break
      }
    }

    const startLine = start > 0 ? (text.slice(0, start).match(/\n/g)?.length || 0) + 1 : 1
    return {
      startLine,
      lines: text.slice(start).split('\n')
    }
  })
  const logLines = computed(() => logLineWindow.value.lines)
  const logLineStartNo = computed(() => logLineWindow.value.startLine)
  const searchItems = computed(() => [
    {
      label: '应用',
      key: 'systemId',
      type: 'select',
      props: {
        filterable: true,
        placeholder: '请选择应用',
        options: systems.value.map((item) => ({ label: item.name, value: item.id }))
      }
    },
    { label: '名称', key: 'name', type: 'input', props: { clearable: true } },
    {
      label: '环境',
      key: 'environmentId',
      type: 'select',
      props: {
        clearable: true,
        filterable: true,
        options: environments.value.map((item) => ({
          label: `${item.name}（${item.code}）`,
          value: item.id
        }))
      }
    }
  ])

  const columns = computed(() => [
    {
      prop: 'name',
      label: '名称',
      minWidth: 150,
      fixed: 'left' as const,
      showOverflowTooltip: true
    },
    { prop: 'code', label: '编码', minWidth: 140, showOverflowTooltip: true },
    { prop: 'systemName', label: '应用', minWidth: 140, showOverflowTooltip: true },
    { prop: 'environmentName', label: '环境', width: 120, showOverflowTooltip: true },
    {
      prop: 'jobFullName',
      label: '运行目标',
      minWidth: 220,
      showOverflowTooltip: true,
      formatter: (row: DevopsPipeline) => {
        if (row.engineType === 'tekton') {
          return row.code || row.name || '-'
        }
        return row.jobFullName || '-'
      }
    },
    {
      prop: 'triggerMode',
      label: '触发方式',
      width: 100,
      formatter: (row: DevopsPipeline) =>
        h(ElTag, { size: 'small', type: triggerModeType(resolveTriggerMode(row)) }, () =>
          triggerModeText(resolveTriggerMode(row))
        )
    },
    {
      prop: 'syncStatus',
      label: '同步',
      width: 110,
      formatter: (row: DevopsPipeline) =>
        h(ElTag, { size: 'small', type: syncStatusType(row.syncStatus) }, () =>
          syncStatusText(row.syncStatus)
        )
    },
    {
      prop: 'lastRunStatus',
      label: '最近运行',
      width: 120,
      formatter: (row: DevopsPipeline) =>
        h(ElTag, { size: 'small', type: runStatusType(row.lastRunStatus) }, () =>
          runStatusText(row.lastRunStatus)
        )
    },
    {
      prop: 'lastRunBuildNumber',
      label: '最近实例',
      width: 120,
      formatter: (row: DevopsPipeline) => {
        if (row.engineType === 'tekton') {
          return row.lastRunStatus
            ? h(ElTag, { size: 'small', type: runStatusType(row.lastRunStatus) }, () =>
                runStatusText(row.lastRunStatus)
              )
            : h('span', { class: 'muted-text' }, '未运行')
        }
        if (!row.lastRunBuildNumber) {
          return h('span', { class: 'muted-text' }, '未运行')
        }
        return h(
          ElButton,
          { link: true, type: 'primary', onClick: () => openJenkinsBuild(row.lastRunBuildUrl) },
          () => `#${row.lastRunBuildNumber}`
        )
      }
    },
    {
      prop: 'status',
      label: '状态',
      width: 90,
      formatter: (row: DevopsPipeline) =>
        h(ElTag, { size: 'small', type: row.status === 1 ? 'success' : 'info' }, () =>
          row.status === 1 ? '启用' : '停用'
        )
    },
    {
      prop: 'updatedAt',
      label: '更新时间',
      width: 180,
      formatter: (row: DevopsPipeline) => formatTime(row.updatedAt)
    },
    {
      prop: 'operation',
      label: '操作',
      width: 176,
      fixed: 'right' as const,
      formatter: (row: DevopsPipeline) =>
        h('div', { class: 'table-actions' }, [
          h(
            ElButton,
            { link: true, type: 'primary', onClick: () => enterEditor('view', row) },
            () => '查看'
          ),
          h(
            ElButton,
            {
              link: true,
              type: 'success',
              disabled: row.status !== 1,
              onClick: () => runPipeline(row)
            },
            () => '运行'
          ),
          h(
            ElDropdown,
            { trigger: 'click', onCommand: (cmd: string) => handlePipelineCommand(cmd, row) },
            {
              default: () => h(ElButton, { link: true, type: 'primary' }, () => '更多'),
              dropdown: () =>
                h(ElDropdownMenu, null, () => [
                  h(ElDropdownItem, { command: 'main' }, () => '主配置'),
                  h(ElDropdownItem, { command: 'workflow' }, () => '流水线配置'),
                  h(ElDropdownItem, { command: 'schedule' }, () =>
                    row.engineType === 'tekton' ? '高级配置' : '定时配置'
                  ),
                  h(ElDropdownItem, { command: 'records' }, () => '执行记录'),
                  h(ElDropdownItem, { command: 'delete', divided: true }, () => '删除')
                ])
            }
          )
        ])
    }
  ])

  const runColumns = computed(() => [
    { prop: 'pipelineName', label: '流水线', minWidth: 130, showOverflowTooltip: true },
    { prop: 'environmentName', label: '环境', minWidth: 120, showOverflowTooltip: true },
    {
      prop: 'status',
      label: '状态',
      width: 96,
      formatter: (row: DevopsPipelineRun) =>
        h(ElTag, { size: 'small', type: runStatusType(row.status) }, () =>
          runStatusText(row.status)
        )
    },
    {
      prop: 'jenkinsBuildNumber',
      label: isTektonRunContext.value ? 'BuildId / PipelineRun' : '构建号',
      minWidth: isTektonRunContext.value ? 220 : 96,
      formatter: (row: DevopsPipelineRun) => {
        if (isTektonRunContext.value) {
          const buildText = row.buildId ? `#${row.buildId}` : '等待中'
          const runName = row.tektonPipelineRunName || row.tektonPipelineName || ''
          return h(
            'span',
            {
              class: 'muted-text',
              title: runName
            },
            runName ? `${buildText} / ${runName}` : buildText
          )
        }
        if (row.jenkinsBuildNumber > 0) {
          return h(
            ElButton,
            { link: true, type: 'primary', onClick: () => openJenkinsBuild(row.jenkinsBuildUrl) },
            () => `#${row.jenkinsBuildNumber}`
          )
        }
        return h('span', { class: 'muted-text' }, '等待中')
      }
    },
    { prop: 'triggerUsername', label: '执行人', minWidth: 110, showOverflowTooltip: true },
    {
      prop: 'triggerType',
      label: '触发方式',
      width: 96,
      formatter: (row: DevopsPipelineRun) =>
        h(
          ElTag,
          { size: 'small', type: row.triggerType === 'scheduled' ? 'warning' : 'success' },
          () => runTriggerTypeName(row.triggerType)
        )
    },
    {
      prop: 'startedAt',
      label: '开始时间',
      minWidth: 168,
      showOverflowTooltip: true,
      formatter: (row: DevopsPipelineRun) => formatRunTime(row.startedAt)
    },
    {
      prop: 'durationSeconds',
      label: '耗时',
      width: 86,
      formatter: (row: DevopsPipelineRun) => `${row.durationSeconds || 0}s`
    },
    {
      prop: 'operation',
      label: '操作',
      width: 190,
      fixed: 'right' as const,
      formatter: (row: DevopsPipelineRun) => {
        const active = isRunActive(row.status)
        return h(
          'div',
          { class: 'table-actions run-table-actions' },
          active
            ? [
                h(
                  ElButton,
                  {
                    link: true,
                    type: 'primary',
                    loading: processOpeningRunId.value === row.id,
                    onClick: () => enterRunProcess(row, 'stage')
                  },
                  () => '运行日志'
                ),
                h(
                  ElButton,
                  {
                    link: true,
                    type: 'danger',
                    loading: currentRun.value?.id === row.id && runActionLoading.value,
                    onClick: () => stopRun(row.id)
                  },
                  () => '停止'
                )
              ]
            : [
                h(
                  ElButton,
                  {
                    link: true,
                    type: 'primary',
                    loading: processOpeningRunId.value === row.id,
                    onClick: () => enterRunProcess(row, 'full')
                  },
                  () => '查看日志'
                ),
                h(
                  ElButton,
                  {
                    link: true,
                    type: 'danger',
                    loading: currentRun.value?.id === row.id && runActionLoading.value,
                    onClick: () => deleteRun(row)
                  },
                  () => '删除'
                )
              ]
        )
      }
    }
  ])

  const fetchSystems = async () => {
    if (!projectStore.selectedProjectId) {
      systems.value = []
      searchForm.value.systemId = undefined
      formData.systemId = ''
      return
    }
    const res = await listDevopsSystemApi({
      page: 1,
      pageSize: 200,
      projectId: projectStore.selectedProjectId,
      status: 1
    })
    systems.value = res.items || []
    const searchSystemId = resolvePipelineSearchSystemId()
    searchForm.value.systemId = searchSystemId || undefined
    if (searchSystemId) {
      persistPipelineSystemId(searchSystemId)
    }
    if (!hasSystem(formData.systemId)) {
      formData.systemId = searchSystemId
    }
  }

  const fetchEnvironments = async () => {
    const res = await listDevopsPipelineEnvironmentApi({
      page: 1,
      pageSize: 200,
      status: 1
    })
    environments.value = res.items || []
  }

  const fetchTemplates = async () => {
    templateLoading.value = true
    try {
      const res = await listDevopsPipelineTemplateApi({
        page: 1,
        pageSize: 200,
        engineType: formData.engineType || 'jenkins',
        projectId: projectStore.selectedProjectId,
        status: 1
      })
      templates.value = res.items || []
      if (isTektonTemplateSource.value && templateSourceFilter.value === 'custom') {
        templateSourceFilter.value = 'all'
      }
      if (
        templateSourceKey.value &&
        templateSourceKey.value !== customTemplateSourceKey &&
        !templates.value.some((item) => item.id === templateSourceKey.value)
      ) {
        templateSourceKey.value = ''
      }
      if (!templateSourceKey.value) {
        templateSourceKey.value = isTektonEngine()
          ? templates.value[0]?.id || ''
          : customTemplateSourceKey
      }
      templateChoiceId.value = selectedTemplateChoice.value?.id || ''
    } finally {
      templateLoading.value = false
    }
  }

  const fetchCategories = async () => {
    const res = await listDevopsStepCategoryApi({ page: 1, pageSize: 200, status: 1 })
    categories.value = res.items || []
  }

  const fetchStepLibrary = async (payload?: StepLibraryRefreshPayload) => {
    if (categories.value.length === 0) {
      await fetchCategories()
    }
    const defaultCategoryId =
      formData.engineType === 'tekton' ? firstTektonCategoryId.value : firstNormalCategoryId.value
    const categoryId = payload?.categoryId || stepFilter.categoryId || defaultCategoryId
    if (categoryId && stepFilter.categoryId !== categoryId) stepFilter.categoryId = categoryId
    activeStepLibraryCategoryId.value = categoryId || ''
    if (formData.engineType === 'tekton') {
      stepLoading.value = true
      try {
        if (categoryId === tektonRuntimeTaskCategoryId) {
          stepLibrary.value = []
          await fetchTektonRuntimeResources()
          return
        }
        tektonRuntimeTasks.value = []
        const res = await listDevopsTektonStepApi({
          page: 1,
          pageSize: 200,
          status: 1,
          engineType: formData.engineType,
          categoryId: categoryId || undefined
        })
        stepLibrary.value = res.items || []
      } finally {
        stepLoading.value = false
      }
      return
    }
    stepLoading.value = true
    try {
      const stepApi =
        formData.engineType === 'tekton' ? listDevopsTektonStepApi : listDevopsJenkinsStepApi
      const res = await stepApi({
        page: 1,
        pageSize: 200,
        status: 1,
        engineType: formData.engineType,
        categoryId: categoryId || undefined
      })
      stepLibrary.value = res.items || []
    } finally {
      stepLoading.value = false
    }
  }

  const fetchAllSteps = async () => {
    const stepApi =
      formData.engineType === 'tekton' ? listDevopsTektonStepApi : listDevopsJenkinsStepApi
    const res = await stepApi({
      page: 1,
      pageSize: 200,
      status: 1,
      engineType: formData.engineType
    })
    allStepLibrary.value = res.items || []
  }

  const clearTektonRuntimeResources = () => {
    tektonRuntimeServiceAccounts.value = []
    tektonRuntimeStorageClasses.value = []
    tektonRuntimeTasks.value = []
    tektonRuntimeSecrets.value = []
    tektonRuntimePvcs.value = []
    tektonRuntimeConfigMaps.value = []
    tektonWorkspaceCredentials.value = []
    tektonWorkspaceProjectConfigs.value = []
  }

  const fetchTektonWorkspacePlatformResources = async (projectId: string) => {
    if (!projectId) {
      tektonWorkspaceCredentials.value = []
      tektonWorkspaceProjectConfigs.value = []
      return
    }
    const credentialBase = { page: 1, pageSize: 200, status: 1 }
    const [projectCredentials, systemCredentials, configs] = await Promise.all([
      listDevopsCredentialApi({ ...credentialBase, scope: 'project', projectId }).catch(() => ({
        items: []
      })),
      listDevopsCredentialApi({ ...credentialBase, scope: 'system' }).catch(() => ({ items: [] })),
      listDevopsProjectConfigApi({ page: 1, pageSize: 200, status: 1, projectId }).catch(() => ({
        items: []
      }))
    ])
    const credentialMap = new Map<string, DevopsCredential>()
    ;[...(projectCredentials.items || []), ...(systemCredentials.items || [])].forEach((item) => {
      if (item.id) credentialMap.set(item.id, item)
    })
    tektonWorkspaceCredentials.value = Array.from(credentialMap.values())
    tektonWorkspaceProjectConfigs.value = configs.items || []
  }

  const fetchTektonRuntimeResources = async (source?: DevopsPipeline | null) => {
    const engineType = source?.engineType || formData.engineType
    const projectId = source?.projectId || projectStore.selectedProjectId
    const buildChannelBindingId = source?.buildChannelBindingId || formData.buildChannelBindingId
    if (engineType !== 'tekton' || !projectId || !buildChannelBindingId) {
      clearTektonRuntimeResources()
      return
    }
    await fetchTektonWorkspacePlatformResources(projectId)
    const baseParams = {
      projectId,
      systemId: source?.systemId || formData.systemId || undefined,
      environmentId: source?.environmentId || formData.environmentId || undefined,
      buildChannelBindingId
    }
    const loadRuntimeResource = async (
      resourceType: ListTektonRuntimeResourceParams['resourceType']
    ) => {
      try {
        const res = await listDevopsTektonRuntimeResourceApi(
          { ...baseParams, resourceType },
          { showErrorMessage: false }
        )
        return res.items || []
      } catch {
        return []
      }
    }
    const loadSecrets = async () => {
      try {
        const res = await listDevopsTektonSecretApi({
          projectId,
          bindingId: buildChannelBindingId
        })
        return (res.items || []).map<TektonRuntimeResource>((item) => ({
          name: item.name,
          namespace: item.namespace,
          kind: 'Secret',
          apiVersion: 'v1',
          description: item.type || 'Opaque',
          params: [],
          workspaces: [],
          results: [],
          provisioner: '',
          reclaimPolicy: '',
          volumeBindingMode: '',
          allowVolumeExpansion: false,
          secrets: [],
          imagePullSecrets: [],
          createdAt: item.createdAt,
          yaml: ''
        }))
      } catch {
        return []
      }
    }
    const loadPvcs = async () => {
      try {
        const res = await listDevopsTektonPvcApi({
          projectId,
          bindingId: buildChannelBindingId
        })
        return (res.items || []).map<TektonRuntimeResource>((item) => ({
          name: item.name,
          namespace: item.namespace,
          kind: 'PersistentVolumeClaim',
          apiVersion: 'v1',
          description: item.storage || item.status || 'PVC',
          params: [],
          workspaces: [],
          results: [],
          provisioner: item.storageClassName,
          reclaimPolicy: '',
          volumeBindingMode: item.volumeMode,
          allowVolumeExpansion: false,
          secrets: [],
          imagePullSecrets: [],
          createdAt: item.createdAt,
          yaml: ''
        }))
      } catch {
        return []
      }
    }
    const [tasks, serviceAccounts, storageClasses, configMaps, secrets, pvcs] = await Promise.all([
      loadRuntimeResource('task'),
      loadRuntimeResource('serviceAccount'),
      loadRuntimeResource('storageClass'),
      loadRuntimeResource('configMap'),
      loadSecrets(),
      loadPvcs()
    ])
    tektonRuntimeTasks.value = tasks
    tektonRuntimeServiceAccounts.value = serviceAccounts
    tektonRuntimeStorageClasses.value = storageClasses
    tektonRuntimeConfigMaps.value = configMaps.map((item) => ({
      ...item,
      description: item.keys?.length ? item.keys.join(', ') : 'ConfigMap'
    }))
    tektonRuntimeSecrets.value = secrets
    tektonRuntimePvcs.value = pvcs
  }

  const refreshTektonTaskLibrary = async () => {
    await fetchStepLibrary({ categoryId: stepFilter.categoryId || firstTektonCategoryId.value })
  }

  const fetchBuildBindings = async () => {
    if (!projectStore.selectedProjectId) {
      buildBindings.value = []
      return
    }
    const res = await listDevopsProjectChannelBindingApi({
      page: 1,
      pageSize: 200,
      projectId: projectStore.selectedProjectId,
      channelType: formData.engineType || 'jenkins',
      status: 1
    })
    buildBindings.value = res.items || []
  }

  const fetchPipelineChannelBindings = async (groupCode?: string) => {
    const code = String(groupCode || '').trim()
    if (!projectStore.selectedProjectId || !code) return
    if (pipelineChannelBindingLoadedGroups[code]) return
    const res = await listDevopsProjectChannelBindingApi({
      page: 1,
      pageSize: 200,
      projectId: projectStore.selectedProjectId,
      channelGroupCode: code,
      status: 1
    })
    const exists = new Set(pipelineChannelBindings.value.map((item) => item.id))
    ;(res.items || []).forEach((item) => {
      if (!exists.has(item.id)) {
        pipelineChannelBindings.value.push(item)
        exists.add(item.id)
      }
    })
    pipelineChannelBindingLoadedGroups[code] = true
  }

  const isServiceScanItem = (item: PipelineScanPlanItem) =>
    item.toolMode === 'service' || item.toolMode === 'service_api'

  const pipelineChannelParamOptions = (item: PipelineParam) => {
    const groupCode = String(item.config?.channelGroupCode || item.channelGroupCode || '').trim()
    const channelType = String(item.config?.channelTypeFilter || '').trim()
    if (!groupCode) return []
    return pipelineChannelBindings.value.filter(
      (binding) =>
        binding.channelGroupCode === groupCode &&
        (!channelType || binding.channelType === channelType)
    )
  }

  const channelBindingEndpoint = (binding: DevopsProjectChannelBinding) => {
    const config = parseJsonRecord(binding.channelConfig)
    const metadata = parseJsonRecord(binding.metadata)
    return String(config.endpoint || metadata.endpoint || '').trim()
  }

  const channelBindingDynamicOption = (binding: DevopsProjectChannelBinding) => ({
    label: `${binding.channelName}（${binding.channelCode}）`,
    value: binding.id,
    metadata: {
      endpoint: channelBindingEndpoint(binding)
    }
  })

  const handlePipelineChannelParamVisible = async (item: PipelineParam, visible: boolean) => {
    if (!visible) return
    await fetchPipelineChannelBindings(item.config?.channelGroupCode || item.channelGroupCode)
  }

  const isLegacyChannelPipelineParam = (item: PipelineParam) =>
    isPlatformPipelineParam(item) && item.paramType === 'channelGroupCode'

  const dynamicChannelParamType = (item: PipelineParam) => {
    const field = String(item.config?.mappingField || '').trim()
    const channelType = String(item.config?.channelTypeFilter || '').trim()
    const groupCode = String(item.config?.channelGroupCode || item.channelGroupCode || '').trim()
    if (field === 'dynamic.project') {
      if (['gitlab', 'gitee', 'github', 'svn'].includes(channelType)) return 'gitProject'
      if (channelType === 'harbor') return 'harborProject'
      if (channelType === 'registry' || channelType === 'aliyun_registry')
        return 'registryRepository'
      if (channelType === 'nexus') return 'nexusRepository'
      if (channelType === 'jfrog') return 'jfrogRepository'
      if (channelType === 'kube-nova') return 'kubeNovaProject'
    }
    if (field === 'address.projectUrl') {
      if (groupCode === 'code_repo') return 'gitRepositoryUrl'
      if (channelType === 'harbor') return 'harborProjectUrl'
      if (channelType === 'registry' || channelType === 'aliyun_registry')
        return 'registryRepositoryUrl'
      if (channelType === 'sonarqube') return 'sonarProjectUrl'
    }
    if (field === 'address.registryUrl') {
      if (channelType === 'harbor') return 'harborProjectUrl'
      if (channelType === 'registry' || channelType === 'aliyun_registry')
        return 'registryRepositoryUrl'
    }
    if (field === 'dynamic.branch') return 'gitBranch'
    if (field === 'dynamic.tag') {
      if (groupCode === 'image_registry') {
        return channelType === 'harbor' ? 'harborImageTag' : 'registryImageTag'
      }
      return 'gitTag'
    }
    if (field === 'dynamic.registry') {
      if (channelType === 'harbor') return 'harborProject'
      if (channelType === 'registry' || channelType === 'aliyun_registry')
        return 'registryRepository'
    }
    if (field === 'dynamic.repository') {
      if (channelType === 'harbor') return 'harborProject'
      if (channelType === 'registry' || channelType === 'aliyun_registry')
        return 'registryRepository'
      if (channelType === 'jfrog') return 'jfrogRepository'
      return 'nexusRepository'
    }
    if (field === 'dynamic.image') return channelType === 'harbor' ? 'harborImage' : 'registryImage'
    if (field === 'dynamic.imageTag')
      return channelType === 'harbor' ? 'harborImageTag' : 'registryImageTag'
    if (field === 'address.imageUrl')
      return channelType === 'harbor' ? 'harborImageUrl' : 'registryImageUrl'
    if (field === 'address.imageTagUrl')
      return channelType === 'harbor' ? 'harborImageTagUrl' : 'registryImageTagUrl'
    if (field === 'address.repositoryUrl')
      return channelType === 'jfrog' ? 'jfrogRepositoryUrl' : 'nexusRepositoryUrl'
    if (field === 'dynamic.artifact' || field === 'dynamic.artifactName')
      return channelType === 'jfrog' ? 'jfrogArtifactName' : 'nexusArtifactName'
    if (field === 'dynamic.artifactVersion' || field === 'dynamic.artifactTag')
      return channelType === 'jfrog' ? 'jfrogArtifactVersion' : 'nexusArtifactVersion'
    if (field === 'address.artifactUrl')
      return channelType === 'jfrog' ? 'jfrogArtifactUrl' : 'nexusArtifactUrl'
    if (field === 'address.artifactVersionUrl' || field === 'address.artifactTagUrl')
      return channelType === 'jfrog' ? 'jfrogArtifactVersionUrl' : 'nexusArtifactVersionUrl'
    if (field === 'dynamic.projectName') return 'sonarProjectName'
    if (field === 'dynamic.projectKey') return 'sonarProjectKey'
    if (field === 'address.projectKeyUrl') return 'sonarProjectKeyUrl'
    if (field === 'dynamic.namespace') return 'kubernetesNamespace'
    if (field === 'dynamic.workloadType') return 'kubernetesWorkloadType'
    if (
      [
        'dynamic.deployment',
        'dynamic.statefulSet',
        'dynamic.daemonSet',
        'dynamic.cronJob'
      ].includes(field)
    )
      return 'kubernetesResource'
    if (field === 'dynamic.resourceName') return 'kubernetesResource'
    if (field === 'dynamic.cluster') return 'kubeNovaCluster'
    if (field === 'dynamic.workspace') return 'kubeNovaWorkspace'
    if (field === 'dynamic.application') return 'kubeNovaApplication'
    if (field === 'dynamic.version') return 'kubeNovaVersion'
    if (field === 'dynamic.container' || field === 'dynamic.containerName')
      return channelType === 'kubernetes' ? 'kubernetesContainer' : 'kubeNovaContainer'
    if (field === 'dynamic.imageName') return 'kubernetesImage'
    if (
      field === 'dynamic.deployConfig' ||
      field === 'address.deployConfig' ||
      field === 'address.deploymentConfig'
    )
      return channelType === 'kubernetes'
        ? kubernetesDeployConfigParamType
        : kubeNovaDeployConfigParamType
    if (field === 'address.hostConfig') return 'hostConfig'
    if (field === 'address.groupConfig') return 'hostGroupConfig'
    return ''
  }

  const isDynamicChannelAddressParam = (item: PipelineParam) =>
    isCompositePlatformParamType(dynamicChannelParamType(item))

  const platformAddressParamType = (item: PipelineParam) =>
    isDynamicChannelAddressParam(item) ? dynamicChannelParamType(item) : item.paramType

  const isChannelDeployConfigParam = (item: PipelineParam, channelType: string) =>
    item.paramType === 'channelGroupCode' &&
    ['dynamic.deployConfig', 'address.deployConfig', 'address.deploymentConfig'].includes(
      String(item.config?.mappingField || '').trim()
    ) &&
    String(item.config?.channelTypeFilter || '').trim() === channelType

  const isKubeNovaDeployConfigParam = (item: PipelineParam) =>
    item.paramType === kubeNovaDeployConfigParamType ||
    isChannelDeployConfigParam(item, 'kube-nova')

  const isKubernetesDeployConfigParam = (item: PipelineParam) =>
    item.paramType === kubernetesDeployConfigParamType ||
    isChannelDeployConfigParam(item, 'kubernetes')

  const isDynamicChannelPipelineParam = (item: PipelineParam) =>
    isLegacyChannelPipelineParam(item) &&
    !!dynamicChannelParamType(item) &&
    !isDynamicChannelAddressParam(item) &&
    !isKubeNovaDeployConfigParam(item) &&
    !isKubernetesDeployConfigParam(item)

  const effectiveDynamicParamType = (item: PipelineParam) =>
    dynamicChannelParamType(item) || item.paramType

  const isChannelPipelineParam = (item: PipelineParam) =>
    isPlatformPipelineParam(item) && channelParamTypes.includes(item.paramType)

  const isDynamicPipelineParam = (item: PipelineParam) =>
    isPlatformPipelineParam(item) && dynamicParamTypes.includes(item.paramType)

  const isManagedSecretPipelineParam = (item: PipelineParam) =>
    item.paramType === 'kubernetesSecretName' && !!String(item.config?.voucherModel || '').trim()

  const isExternalSecretPipelineParam = (item: PipelineParam) =>
    item.paramType === 'kubernetesSecretName' && !isManagedSecretPipelineParam(item)

  const isPVCPipelineParam = (item: PipelineParam) => item.paramType === 'kubernetesPvcName'

  const isCredentialPipelineParam = (item: PipelineParam) =>
    (isPlatformPipelineParam(item) && item.paramType === 'voucherModel') ||
    isManagedSecretPipelineParam(item)

  const isChannelCredentialPipelineParam = (item: PipelineParam) =>
    item.paramType === 'channelCredential'

  const credentialMappingFieldLabel = (field?: string) => {
    const labelMap: Record<string, string> = {
      username: '用户名',
      password: '密码',
      token: 'Token',
      privateKey: '私钥',
      passphrase: '私钥口令',
      kubeconfig: 'Kubeconfig',
      secretText: 'Secret Text',
      certificate: '证书',
      jsonData: 'JSON',
      jsonBase64: 'JSON Base64'
    }
    const value = String(field || '').trim()
    return value ? `${labelMap[value] || value}（${value}）` : '凭证字段'
  }

  const normalizeCredentialMode = (mode?: string, mappingField?: string) => {
    const value = String(mode || '').trim()
    if (['field', 'jsonData', 'jsonBase64', 'jenkinsCredentialId'].includes(value)) return value
    const legacyField = String(mappingField || '').trim()
    if (legacyField === 'credentialId') return 'jenkinsCredentialId'
    if (legacyField === 'jsonData' || legacyField === 'jsonBase64') return legacyField
    return 'field'
  }

  const channelCredentialOutputLabel = (item: PipelineParam) => {
    const mode = normalizeCredentialMode(item.config?.credentialMode, item.config?.mappingField)
    if (mode === 'jenkinsCredentialId') return '输出：Jenkins credentialId'
    if (mode === 'jsonData') return '输出：JSON'
    if (mode === 'jsonBase64') return '输出：JSON Base64'
    return `输出：${credentialMappingFieldLabel(item.config?.mappingField)}`
  }

  const isPipelineDynamicConfigParam = (item: PipelineParam) =>
    (isLegacyChannelPipelineParam(item) && !isDynamicChannelPipelineParam(item)) ||
    isDynamicChannelPipelineParam(item) ||
    isChannelPipelineParam(item) ||
    isDynamicPipelineParam(item) ||
    isExternalSecretPipelineParam(item) ||
    isPVCPipelineParam(item) ||
    isCredentialPipelineParam(item)

  const isRemotePipelineSelectParam = (item: PipelineParam) =>
    isDynamicChannelPipelineParam(item) ||
    isDynamicPipelineParam(item) ||
    isCredentialPipelineParam(item) ||
    isExternalSecretPipelineParam(item) ||
    isPVCPipelineParam(item)

  const dynamicRequestParamType = (item: PipelineParam) =>
    isManagedSecretPipelineParam(item) ? 'voucherModel' : effectiveDynamicParamType(item)

  const dynamicComponentValue = (item: PipelineParam, readValue: (code?: string) => string) =>
    isCredentialPipelineParam(item)
      ? item.config?.voucherModel || ''
      : item.paramType === 'channelGroupCode'
        ? (isImageRegistryDynamicType(effectiveDynamicParamType(item)) ||
            isArtifactRepositoryDynamicType(effectiveDynamicParamType(item))) &&
          item.config?.componentParamCode
          ? readValue(item.config.componentParamCode)
          : ['kubernetesResource', 'kubernetesContainer', 'kubernetesImage'].includes(
                effectiveDynamicParamType(item)
              )
            ? readValue(
                (item.config?.dependencyParamCodes || []).find(
                  (dep: { field?: string; paramCode?: string }) =>
                    String(dep.field || '').trim() === 'dynamic.workloadType'
                )?.paramCode
              ) ||
              item.config?.channelTypeFilter ||
              ''
            : item.config?.channelTypeFilter || ''
        : readValue(item.config?.componentParamCode)

  const pipelineParamValue = (code?: string) => {
    if (!code) return ''
    const param = formData.params.find((item) => item.code === code)
    return String(param?.currentValue ?? param?.defaultValue ?? '').trim()
  }

  const channelDependencyValues = (item: PipelineParam, readValue: (code?: string) => string) => {
    const result: Record<string, string> = {}
    ;(item.config?.dependencyParamCodes || []).forEach(
      (dep: { field?: string; paramCode?: string }) => {
        const field = String(dep.field || '').trim()
        const value = readValue(dep.paramCode)
        if (field && value) result[field] = value
      }
    )
    return result
  }

  const channelDependencyValuesText = (values: Record<string, string>) =>
    Object.keys(values).length ? JSON.stringify(values) : ''

  const parseDynamicOptionMetadata = (raw?: string): DynamicParamOptionMetadata | undefined => {
    if (!raw) return undefined
    try {
      return JSON.parse(raw) as DynamicParamOptionMetadata
    } catch {
      return undefined
    }
  }

  const dynamicOptionSecondary = (option: DynamicSelectOption) => {
    const metadata = option.metadata || {}
    const value = String(
      metadata.endpoint ||
        metadata.url ||
        metadata.httpUrlToRepo ||
        metadata.sshUrlToRepo ||
        metadata.fullName ||
        metadata.image ||
        metadata.namespace ||
        metadata.clusterUuid ||
        metadata.storageClassName ||
        ''
    ).trim()
    if (value) return value
    const rawValue = String(option.value || '').trim()
    return rawValue && rawValue !== option.label && rawValue.includes('://') ? rawValue : ''
  }

  const dynamicOptionMatchesValue = (option: DynamicSelectOption, value: string) => {
    const text = value.trim()
    if (!text) return false
    if (String(option.value || '').trim() === text) return true
    return String(option.metadata?.bindingId || '').trim() === text
  }

  const normalizeSelectedDynamicOptions = (
    options: DynamicSelectOption[],
    currentValue: unknown
  ) => {
    const value = String(currentValue ?? '').trim()
    if (!value || options.some((option) => String(option.value || '').trim() === value)) {
      return options
    }
    const matched = options.find((option) => dynamicOptionMatchesValue(option, value))
    return matched ? [{ ...matched, value }, ...options] : options
  }

  const isGitRepositoryURLValue = (value: string) => {
    const text = value.trim()
    return /^https?:\/\//.test(text) || /^ssh:\/\//.test(text) || /^[^@\s]+@[^:\s]+:.+/.test(text)
  }

  const isImageRegistryAddressValue = (value: string) => {
    const text = value.trim()
    if (!text || !text.includes('/')) return false
    return /^https?:\/\//.test(text) || text.includes('.') || text.includes(':')
  }

  function isGitRefDynamicType(paramType: string) {
    return ['gitBranch', 'gitTag'].includes(paramType)
  }

  const isImageRegistryDynamicType = (paramType: string) =>
    ['harborImage', 'harborImageTag', 'registryImage', 'registryImageTag'].includes(paramType)

  const imageRegistryProjectDependencyFields = [
    'address.imageTagUrl',
    'address.imageUrl',
    'address.projectUrl',
    'address.registryUrl',
    'dynamic.project',
    'dynamic.repository',
    'dynamic.registry'
  ]

  const isArtifactRepositoryAddressValue = (value: string) => {
    const text = value.trim()
    if (!text || !text.includes('/')) return false
    return /^https?:\/\//.test(text) || text.includes('.') || text.includes(':')
  }

  const isArtifactRepositoryDynamicType = (paramType: string) =>
    [
      'nexusArtifactName',
      'nexusArtifactVersion',
      'jfrogArtifactName',
      'jfrogArtifactVersion'
    ].includes(paramType)

  function gitRefProjectDependencyParamCode(item: PipelineParam, field?: string) {
    if (!isGitRefDynamicType(effectiveDynamicParamType(item))) return ''
    const fields = field ? [field] : ['address.projectUrl', 'dynamic.project']
    return (
      (item.config?.dependencyParamCodes || []).find(
        (dep: { field?: string; paramCode?: string }) =>
          fields.includes(String(dep.field || '').trim())
      )?.paramCode || ''
    )
  }

  const gitRefProjectDependencyValue = (
    item: PipelineParam,
    readValue: (code?: string) => string
  ) => {
    for (const field of ['address.projectUrl', 'dynamic.project']) {
      const value = readValue(gitRefProjectDependencyParamCode(item, field))
      if (value) return value
    }
    return item.config?.projectParamCode ? readValue(item.config.projectParamCode) : ''
  }

  const hasMissingChannelDependency = (
    item: PipelineParam,
    readValue: (code?: string) => string
  ) =>
    isGitRefDynamicType(effectiveDynamicParamType(item)) &&
    (item.config?.dependencyParamCodes || []).some((dep: { field?: string }) =>
      ['dynamic.project', 'address.projectUrl'].includes(String(dep.field || '').trim())
    )
      ? !gitRefProjectDependencyValue(item, readValue)
      : (item.config?.dependencyParamCodes || []).some(
          (dep: { paramCode?: string }) =>
            !!String(dep.paramCode || '').trim() && !readValue(dep.paramCode)
        )

  const pipelineDynamicProjectValue = (item: PipelineParam) => {
    if (isGitRefDynamicType(effectiveDynamicParamType(item))) {
      return gitRefProjectDependencyValue(item, pipelineParamValue)
    }
    if (isImageRegistryDynamicType(effectiveDynamicParamType(item))) {
      for (const field of imageRegistryProjectDependencyFields) {
        const paramCode = (item.config?.dependencyParamCodes || []).find(
          (dep: { field?: string; paramCode?: string }) => String(dep.field || '').trim() === field
        )?.paramCode
        const value = pipelineParamValue(paramCode)
        if (value) return value
      }
    }
    if (isArtifactRepositoryDynamicType(effectiveDynamicParamType(item))) {
      for (const field of [
        'address.artifactVersionUrl',
        'address.artifactUrl',
        'address.repositoryUrl'
      ]) {
        const paramCode = (item.config?.dependencyParamCodes || []).find(
          (dep: { field?: string; paramCode?: string }) => String(dep.field || '').trim() === field
        )?.paramCode
        const value = pipelineParamValue(paramCode)
        if (value) return value
      }
    }
    if (
      ['kubernetesResource', 'kubernetesContainer', 'kubernetesImage'].includes(
        effectiveDynamicParamType(item)
      )
    ) {
      const namespaceParamCode = (item.config?.dependencyParamCodes || []).find(
        (dep: { field?: string; paramCode?: string }) =>
          String(dep.field || '').trim() === 'dynamic.namespace'
      )?.paramCode
      const value = pipelineParamValue(namespaceParamCode)
      if (value) return value
    }
    const projectCode = item.config?.projectParamCode || gitRefProjectDependencyParamCode(item)
    if (!projectCode) return ''
    return pipelineParamValue(projectCode)
  }

  const pipelineDynamicChannelValue = (item: PipelineParam) => {
    const channelCode = item.config?.channelParamCode
    const value =
      String(item.config?.channelBindingId || '').trim() ||
      (channelCode ? pipelineParamValue(channelCode) : '')
    if (value) return value
    const effectiveType = effectiveDynamicParamType(item)
    const projectValue = pipelineDynamicProjectValue(item)
    if (isGitRefDynamicType(effectiveType) && isGitRepositoryURLValue(projectValue)) return ''
    if (isImageRegistryDynamicType(effectiveType) && isImageRegistryAddressValue(projectValue))
      return ''
    if (
      isArtifactRepositoryDynamicType(effectiveType) &&
      isArtifactRepositoryAddressValue(projectValue)
    )
      return ''
    return value
  }

  const dynamicEmptyText = (paramType: string) => {
    if (paramType === 'voucherModel') return '暂无该类型平台凭证'
    if (paramType === 'kubernetesSecretName') return '暂无 Kubernetes Secret'
    if (paramType === 'kubernetesPvcName') return '暂无 Kubernetes PVC'
    if (paramType === 'gitBranch') return '该 Git 项目暂无可选分支'
    if (paramType === 'gitTag') return '该 Git 项目暂无可选 Tag'
    return '暂无可选项'
  }

  const dynamicErrorFallback = (paramType: string) => {
    if (paramType === 'voucherModel') return '获取平台凭证失败'
    if (paramType === 'gitBranch') return '获取 Git 分支失败'
    if (paramType === 'gitTag') return '获取 Git Tag 失败'
    if (paramType === 'gitProject') return '获取 Git 项目失败'
    if (paramType === 'sonarProjectKey') return '获取 Sonar 项目 Key 失败'
    if (paramType === 'kubernetesSecretName') return '获取 Kubernetes Secret 失败'
    if (paramType === 'kubernetesPvcName') return '获取 Kubernetes PVC 失败'
    return '获取选项失败'
  }

  const extractErrorMessage = (error: unknown) => {
    if (error instanceof Error) return error.message
    if (typeof error === 'object' && error && 'message' in error) {
      return String((error as { message?: unknown }).message || '')
    }
    return String(error || '')
  }

  const formatDynamicError = (paramType: string, error: unknown) => {
    const fallback = dynamicErrorFallback(paramType)
    const message = extractErrorMessage(error).trim()
    if (!message || message === '服务器内部错误') {
      return `${fallback}，请检查渠道凭证或项目权限`
    }
    return message.includes(fallback) ? message : `${fallback}：${message}`
  }

  const dynamicQueryKey = (
    item: PipelineParam,
    channelBindingId: string,
    projectValue: string,
    keyword: string,
    dependencyValues = ''
  ) =>
    [
      effectiveDynamicParamType(item),
      dynamicComponentValue(item, pipelineParamValue),
      item.paramType === 'configCenter'
        ? item.config?.configTypeId || item.config?.configTypeCode || ''
        : '',
      channelBindingId,
      projectValue,
      keyword.trim(),
      dependencyValues
    ].join('|')

  const pipelineDynamicPlaceholder = (item: PipelineParam) => {
    if (pipelineConfigParamError[item.code]) return pipelineConfigParamError[item.code]
    if (
      pipelineConfigParamLoaded[item.code] &&
      !pipelineConfigParamLoading[item.code] &&
      !(pipelineConfigParamOptions[item.code] || []).length
    ) {
      return dynamicEmptyText(item.paramType)
    }
    if (isLegacyChannelPipelineParam(item) && !isDynamicChannelPipelineParam(item)) {
      return item.defaultValue || '请选择渠道实例'
    }
    const effectiveType = effectiveDynamicParamType(item)
    const spec = dynamicParamSpec[effectiveType]
    if (
      item.paramType === 'channelGroupCode' &&
      hasMissingChannelDependency(item, pipelineParamValue) &&
      !isGitRepositoryURLValue(pipelineDynamicProjectValue(item)) &&
      !isImageRegistryAddressValue(pipelineDynamicProjectValue(item)) &&
      !isArtifactRepositoryAddressValue(pipelineDynamicProjectValue(item))
    ) {
      return '可手动输入，选择依赖后可自动获取'
    }
    if (spec?.projectParamType && !pipelineDynamicProjectValue(item)) {
      return `可手动输入，选择${spec.projectLabel || '上游资源'}后可自动获取`
    }
    if (
      spec?.channelParamType &&
      !pipelineDynamicChannelValue(item) &&
      !isGitRepositoryURLValue(pipelineDynamicProjectValue(item)) &&
      !isImageRegistryAddressValue(pipelineDynamicProjectValue(item)) &&
      !isArtifactRepositoryAddressValue(pipelineDynamicProjectValue(item))
    ) {
      return '可手动输入，选择渠道实例后可自动获取'
    }
    if (isChannelPipelineParam(item)) return item.defaultValue || '请选择渠道实例'
    if (isExternalSecretPipelineParam(item)) return item.defaultValue || '请选择 Kubernetes Secret'
    if (isPVCPipelineParam(item)) return item.defaultValue || '请选择 Kubernetes PVC'
    if (isCredentialPipelineParam(item)) return item.defaultValue || '请选择凭证'
    return item.defaultValue || '请选择'
  }

  const queryPipelineDynamicParamOptions = async (
    item: PipelineParam,
    keyword = '',
    force = false
  ) => {
    if (!isPipelineDynamicConfigParam(item)) return
    const projectId = formData.projectId || projectStore.selectedProjectId
    if (!projectId) return
    if (isExternalSecretPipelineParam(item) || isPVCPipelineParam(item)) {
      const bindingId = String(formData.buildChannelBindingId || '').trim()
      if (!bindingId) {
        pipelineConfigParamOptions[item.code] = []
        pipelineConfigParamLoaded[item.code] = false
        pipelineConfigParamError[item.code] = ''
        return
      }
      if (pipelineConfigParamLoading[item.code]) return
      const queryKey = dynamicQueryKey(item, bindingId, '', keyword)
      if (
        !force &&
        !keyword.trim() &&
        pipelineConfigParamLoaded[item.code] &&
        pipelineConfigParamQueryKey[item.code] === queryKey &&
        !pipelineConfigParamError[item.code]
      ) {
        return
      }
      pipelineConfigParamQueryKey[item.code] = queryKey
      pipelineConfigParamLoading[item.code] = true
      pipelineConfigParamError[item.code] = ''
      try {
        if (isPVCPipelineParam(item)) {
          const res = await listDevopsTektonPvcApi({ projectId, bindingId, keyword })
          pipelineConfigParamOptions[item.code] = normalizeSelectedDynamicOptions(
            (res.items || []).map((pvc) => ({
              label: `${pvc.name}（${pvc.storage || pvc.status || 'PVC'}）`,
              value: pvc.name,
              metadata: {
                namespace: pvc.namespace,
                storageClassName: pvc.storageClassName,
                accessModes: pvc.accessModes || []
              }
            })),
            item.currentValue
          )
        } else {
          const res = await listDevopsTektonSecretApi({
            projectId,
            bindingId,
            keyword
          })
          pipelineConfigParamOptions[item.code] = normalizeSelectedDynamicOptions(
            (res.items || []).map((secret) => ({
              label: `${secret.name}（${secret.type || 'Opaque'}）`,
              value: secret.name,
              metadata: { namespace: secret.namespace, keys: secret.keys || [] }
            })),
            item.currentValue
          )
        }
        pipelineConfigParamLoaded[item.code] = true
      } catch (error) {
        pipelineConfigParamOptions[item.code] = []
        pipelineConfigParamLoaded[item.code] = true
        pipelineConfigParamError[item.code] = formatDynamicError(item.paramType, error)
      } finally {
        pipelineConfigParamLoading[item.code] = false
      }
      return
    }
    const projectValue = pipelineDynamicProjectValue(item)
    const effectiveType = effectiveDynamicParamType(item)
    if (dynamicParamSpec[effectiveType]?.projectParamType && !projectValue) {
      pipelineConfigParamOptions[item.code] = []
      pipelineConfigParamLoaded[item.code] = false
      pipelineConfigParamError[item.code] = ''
      return
    }
    const channelBindingId = pipelineDynamicChannelValue(item)
    if (
      item.paramType === 'channelGroupCode' &&
      dynamicParamSpec[effectiveType]?.channelParamType &&
      !channelBindingId &&
      !isGitRepositoryURLValue(projectValue) &&
      !isImageRegistryAddressValue(projectValue) &&
      !isArtifactRepositoryAddressValue(projectValue)
    ) {
      pipelineConfigParamOptions[item.code] = []
      pipelineConfigParamLoaded[item.code] = false
      pipelineConfigParamError[item.code] = ''
      return
    }
    if (
      (isDynamicPipelineParam(item) || isDynamicChannelPipelineParam(item)) &&
      dynamicParamSpec[effectiveType]?.channelParamType &&
      !channelBindingId &&
      !isGitRepositoryURLValue(projectValue) &&
      !isImageRegistryAddressValue(projectValue) &&
      !isArtifactRepositoryAddressValue(projectValue)
    ) {
      pipelineConfigParamOptions[item.code] = []
      pipelineConfigParamLoaded[item.code] = false
      pipelineConfigParamError[item.code] = ''
      return
    }
    if (pipelineConfigParamLoading[item.code]) return
    const dependencyValues = channelDependencyValuesText(
      channelDependencyValues(item, pipelineParamValue)
    )
    const queryKey = dynamicQueryKey(
      item,
      channelBindingId,
      projectValue,
      keyword,
      dependencyValues
    )
    if (
      !force &&
      !keyword.trim() &&
      pipelineConfigParamLoaded[item.code] &&
      pipelineConfigParamQueryKey[item.code] === queryKey &&
      !pipelineConfigParamError[item.code]
    ) {
      return
    }
    pipelineConfigParamQueryKey[item.code] = queryKey
    pipelineConfigParamLoading[item.code] = true
    pipelineConfigParamError[item.code] = ''
    try {
      const res = await listDevopsDynamicParamOptionsApi({
        projectId,
        systemId: formData.systemId,
        environmentId: formData.environmentId,
        paramType: dynamicRequestParamType(item),
        channelBindingId,
        projectValue,
        componentValue: dynamicComponentValue(item, pipelineParamValue),
        configTypeId: item.config?.configTypeId || '',
        configTypeCode:
          item.paramType === 'channelGroupCode'
            ? item.config?.channelGroupCode || ''
            : item.config?.configTypeCode || pipelineParamValue(item.config?.componentParamCode),
        mappingField: item.config?.mappingField || '',
        dependencyValues,
        keyword,
        page: 1,
        pageSize: 50
      })
      pipelineConfigParamOptions[item.code] = normalizeSelectedDynamicOptions(
        (res.items || []).map((option: DynamicParamOption) => ({
          label: option.label,
          value: option.value,
          metadata: parseDynamicOptionMetadata(option.metadata)
        })),
        item.currentValue
      )
      pipelineConfigParamLoaded[item.code] = true
    } catch (error) {
      pipelineConfigParamOptions[item.code] = []
      pipelineConfigParamLoaded[item.code] = true
      pipelineConfigParamError[item.code] = formatDynamicError(item.paramType, error)
    } finally {
      pipelineConfigParamLoading[item.code] = false
    }
  }

  const handlePipelineDynamicParamVisible = async (item: PipelineParam, visible: boolean) => {
    if (!visible) return
    if (isLegacyChannelPipelineParam(item) && !isDynamicChannelPipelineParam(item)) {
      await handlePipelineChannelParamVisible(item, visible)
      pipelineConfigParamOptions[item.code] = normalizeSelectedDynamicOptions(
        pipelineChannelParamOptions(item).map(channelBindingDynamicOption),
        item.currentValue
      )
      pipelineConfigParamLoaded[item.code] = true
      pipelineConfigParamError[item.code] = ''
      return
    }
    await queryPipelineDynamicParamOptions(item, '', true)
  }

  const refreshPipelineDynamicParamOptions = async (item: PipelineParam) => {
    if (isViewMode.value || item.readonly) return
    if (isLegacyChannelPipelineParam(item) && !isDynamicChannelPipelineParam(item)) {
      const groupCode = String(item.config?.channelGroupCode || item.channelGroupCode || '').trim()
      if (groupCode) delete pipelineChannelBindingLoadedGroups[groupCode]
    }
    await queryPipelineDynamicParamOptions(item, '', true)
  }

  const handlePipelineConfigParamChange = (item: PipelineParam) => {
    formData.params.forEach((target) => {
      const dependsOn =
        target.config?.channelParamCode === item.code ||
        target.config?.projectParamCode === item.code ||
        target.config?.credentialSourceParamCode === item.code ||
        (target.config?.dependencyParamCodes || []).some(
          (dep: { paramCode?: string }) => dep.paramCode === item.code
        )
      if (!dependsOn) return
      if (isKnownDynamicSelectValue(target.currentValue, pipelineConfigParamOptions[target.code])) {
        target.currentValue = ''
      }
      pipelineConfigParamOptions[target.code] = []
      pipelineConfigParamLoaded[target.code] = false
      pipelineConfigParamError[target.code] = ''
      pipelineConfigParamQueryKey[target.code] = ''
    })
  }

  const resetPipelineParamOptionState = () => {
    Object.keys(pipelineConfigParamOptions).forEach((key) => delete pipelineConfigParamOptions[key])
    Object.keys(pipelineConfigParamLoading).forEach((key) => delete pipelineConfigParamLoading[key])
    Object.keys(pipelineConfigParamLoaded).forEach((key) => delete pipelineConfigParamLoaded[key])
    Object.keys(pipelineConfigParamError).forEach((key) => delete pipelineConfigParamError[key])
    Object.keys(pipelineConfigParamQueryKey).forEach(
      (key) => delete pipelineConfigParamQueryKey[key]
    )
  }

  let agentFetchKey = ''
  const getAgentFetchKey = () =>
    [projectStore.selectedProjectId, formData.buildChannelBindingId].join('|')
  const resetAgentCache = () => {
    agentFetchKey = ''
    jenkinsAgents.value = []
  }
  const fetchJenkinsAgents = async (force = false) => {
    if (formData.engineType !== 'jenkins') {
      resetAgentCache()
      return
    }
    if (!projectStore.selectedProjectId || !formData.buildChannelBindingId) {
      resetAgentCache()
      return
    }
    const currentKey = getAgentFetchKey()
    if (!force && agentFetchKey === currentKey && jenkinsAgents.value.length > 0) return
    agentLoading.value = true
    try {
      const res = await listDevopsPipelineJenkinsAgentApi({
        projectId: projectStore.selectedProjectId,
        buildChannelBindingId: formData.buildChannelBindingId
      })
      jenkinsAgents.value = res.items || []
      agentFetchKey = currentKey
    } catch {
      resetAgentCache()
    } finally {
      agentLoading.value = false
    }
  }

  const fetchPipelines = async (showLoading = true) => {
    if (!projectStore.selectedProjectId) {
      pipelines.value = []
      pagination.total = 0
      return
    }
    if (showLoading) loading.value = true
    try {
      const res = await listDevopsPipelineApi({
        page: pagination.current,
        pageSize: pagination.size,
        projectId: projectStore.selectedProjectId,
        ...searchForm.value
      })
      pipelines.value = res.items || []
      pagination.total = res.total || 0
    } finally {
      if (showLoading) loading.value = false
    }
  }

  const runSequential = async (tasks: Array<() => Promise<unknown>>) => {
    for (const task of tasks) {
      await task()
    }
  }

  let refreshPromise: Promise<void> | null = null
  const refreshAll = async () => {
    if (refreshPromise) return refreshPromise
    loading.value = true
    refreshPromise = (async () => {
      await projectStore.fetchProjects()
      await Promise.all([fetchSystems(), fetchEnvironments(), fetchBuildBindings()])
      await fetchPipelines(false)
    })()
    try {
      await refreshPromise
    } finally {
      refreshPromise = null
      loading.value = false
    }
  }

  const normalizeSteps = (steps: WorkflowNode[] = []) =>
    normalizeWorkflowBranches(
      steps.map((step, index) => ({
        ...step,
        id: step.id || `node-${index + 1}`,
        stepType: step.stepType || 'task',
        enabled: step.enabled !== false
      }))
    )

  const fillFormFromPipeline = (row: DevopsPipeline) => {
    Object.assign(formData, defaultForm(), {
      projectId: row.projectId,
      systemId: row.systemId,
      environmentId: row.environmentId,
      name: row.name,
      code: row.code,
      engineType: row.engineType || 'jenkins',
      templateId: row.templateId,
      buildChannelBindingId: row.buildChannelBindingId,
      steps: row.steps || [],
      params: row.params || [],
      agent: normalizeFormAgent(row.agent),
      tools: row.tools || [],
      options: row.options?.length ? row.options : defaultJenkinsOptions(),
      triggers: row.triggers || [],
      triggerMode: resolveTriggerMode(row),
      scanEnabled: Boolean(row.scanEnabled),
      scanMode: row.scanMode || 'strict',
      rejectUnexpectedReports: row.rejectUnexpectedReports ?? true,
      enforce: Boolean(row.enforce),
      scanItems: (row.scanItems || []).map((item) => ({ ...item })),
      tektonDagConfig: row.tektonDagConfig || '',
      tektonRunPolicy: row.tektonRunPolicy || '',
      tektonTriggerConfig: row.tektonTriggerConfig || '',
      tektonPrunerPolicyRef: row.tektonPrunerPolicyRef || '',
      tektonTemplateId: row.tektonTemplateId || '',
      tektonTemplateSnapshot: row.tektonTemplateSnapshot || '',
      tektonParamBindings: row.tektonParamBindings || '',
      tektonWorkspaceBindings: row.tektonWorkspaceBindings || '',
      tektonResourceBindings: row.tektonResourceBindings || '',
      status: row.status ?? 1
    })
    currentSteps.value = normalizeSteps(row.steps || [])
    tektonDagConfig.value =
      row.engineType === 'tekton' ? parseTektonDagConfigFromPipeline(row) : defaultTektonDagConfig()
    ensureTektonConfigDefaults()
    ensureBuildBindingOption(row)
    applyTektonRuntimeDefaults()
    resetAgentCache()
    resetPipelineParamOptionState()
  }

  const ensureBuildBindingOption = (row: DevopsPipeline) => {
    if (!row.buildChannelBindingId) return
    if (buildBindings.value.some((item) => item.id === row.buildChannelBindingId)) return
    buildBindings.value.push({
      id: row.buildChannelBindingId,
      projectId: row.projectId,
      channelId: '',
      channelGroupCode: row.buildChannelGroupCode || '',
      channelName: row.buildChannelName || row.buildChannelBindingId,
      channelCode: row.buildChannelName || row.buildChannelBindingId,
      channelType: row.engineType || 'jenkins',
      usageScope: 'build',
      isDefault: false,
      allowUseGlobalCredential: true,
      projectCredentialId: '',
      bindingConfig: '',
      status: 1,
      healthStatus: '',
      lastCheckAt: 0,
      lastCheckMessage: '',
      metadata: '',
      createdBy: '',
      updatedBy: '',
      createdAt: 0,
      updatedAt: 0
    })
  }

  const waitFrame = () =>
    new Promise<void>((resolve) => {
      window.requestAnimationFrame(() => resolve())
    })

  const fitWorkflowCanvas = async () => {
    await nextTick()
    await waitFrame()
    await workflowDesignerRef.value?.fitCurrentViewAfterLayout?.()
  }

  const enterEditor = async (
    mode: Exclude<EditorMode, 'browse' | 'records' | 'process'>,
    row?: DevopsPipeline
  ) => {
    if (mode === 'add') {
      await openCreateMainDialog()
      return
    }
    loading.value = true
    try {
      mainFormRef.value?.clearValidate()
      stepFilter.categoryId = ''
      const detailPromise = row ? getDevopsPipelineApi(row.id) : Promise.resolve(null)
      await runSequential([
        fetchSystems,
        fetchEnvironments,
        fetchBuildBindings,
        fetchTemplates,
        fetchCategories
      ])
      const detail = await detailPromise
      if (detail) {
        currentPipeline.value = detail
        fillFormFromPipeline(detail)
        if (mode === 'view' && detail.engineType === 'tekton') {
          currentRunPipeline.value = detail
          tektonDetailActiveTab.value = 'overview'
          tektonPipelineYamlPreview.value = ''
          tektonPipelineRunYamlPreview.value = ''
          tektonTriggerYamlPreview.value = ''
          tektonScheduleYamlPreview.value = ''
          tektonPrunerYamlPreview.value = ''
        }
      }
      await fetchAllSteps()
      await runSequential([fetchStepLibrary, () => fetchJenkinsAgents()])
      if (formData.engineType === 'tekton') {
        await fetchTektonRuntimeResources()
      }
      workflowCanvasKey.value += 1
      editorMode.value = mode
      await fitWorkflowCanvas()
    } finally {
      loading.value = false
    }
  }

  const exitEditor = () => {
    editorMode.value = 'browse'
    currentPipeline.value = null
    currentSteps.value = []
    Object.assign(formData, defaultForm())
    tektonDagConfig.value = defaultTektonDagConfig()
    mainFormRef.value?.clearValidate()
  }

  const exitRunRecords = () => {
    editorMode.value = 'browse'
    currentRunPipeline.value = null
    runs.value = []
    runStages.value = []
    runLog.value = ''
    void fetchPipelines(false)
  }

  const exitRunProcess = () => {
    const shouldBackRecords = Boolean(currentRunPipeline.value)
    handleLogDialogClosed()
    editorMode.value = shouldBackRecords ? 'records' : 'browse'
    if (shouldBackRecords) {
      void fetchRuns()
      void fetchPipelines(false)
    } else {
      void fetchPipelines(false)
    }
  }

  const validateMainFields = () => {
    if (!projectStore.selectedProjectId) {
      ElMessage.warning('请选择项目')
      return false
    }
    if (!formData.systemId) {
      ElMessage.warning('请选择应用')
      return false
    }
    if (!formData.environmentId) {
      ElMessage.warning('请选择归属环境')
      return false
    }
    if (!String(formData.name || '').trim()) {
      ElMessage.warning('请输入名称')
      return false
    }
    if (!String(formData.code || '').trim()) {
      ElMessage.warning('请输入编码')
      return false
    }
    if (!formData.buildChannelBindingId) {
      ElMessage.warning(`请选择 ${engineTypeName(formData.engineType)} 构建渠道`)
      return false
    }
    if (!validateScanConfig()) return false
    return true
  }

  const validateScanConfig = () => {
    if (!formData.scanEnabled) return true
    const items = formData.scanItems || []
    if (items.length === 0) {
      ElMessage.warning('请至少配置一个扫描工具')
      return false
    }
    for (const item of items) {
      if (!item.tool) {
        ElMessage.warning('请选择扫描工具')
        return false
      }
      if (!item.toolMode) {
        ElMessage.warning('请选择扫描模式')
        return false
      }
      if (!item.reportFormat) {
        ElMessage.warning('请输入报告格式')
        return false
      }
      if (isServiceScanItem(item) && !item.toolBindingId) {
        ElMessage.warning('服务型扫描工具必须选择项目渠道')
        return false
      }
      if (
        item.toolMode === 'service_api' &&
        !String(item.targetName || '').trim() &&
        !String(item.targetParamCode || '').trim()
      ) {
        ElMessage.warning('服务 API 型扫描工具必须配置目标或目标参数')
        return false
      }
      if (!isServiceScanItem(item) && item.toolBindingId) {
        ElMessage.warning('本地扫描工具不能绑定项目渠道')
        return false
      }
    }
    return true
  }

  const normalizedAgent = (): DevopsPipelineForm['agent'] => {
    const type = normalizeJenkinsAgentType(formData.agent?.type)
    if (type === 'static' || type === 'dynamic') {
      return {
        type,
        id: formData.agent.id || '',
        name: formData.agent.name || '',
        agentType: normalizeJenkinsAgentType(formData.agent.agentType || type),
        matchMode: formData.agent.matchMode || 'label',
        matchValue: formData.agent.matchValue || '',
        cloud: formData.agent.cloud || '',
        podYaml: formData.agent.podYaml || '',
        containers: formData.agent.containers || []
      }
    }
    return { type }
  }

  const normalizedPipelineSteps = () =>
    normalizeWorkflowBranches(currentSteps.value).map((step) => ({
      ...step,
      paramValues: sanitizePipelineStepParamValues(step),
      containerName: agentContainerEditable.value ? step.containerName || '' : ''
    }))

  const tektonTriggerPayload = () => ({
    ...(tektonDagConfig.value.triggerConfig || {}),
    scheduleConfig: tektonDagConfig.value.scheduleConfig || {}
  })

  const pipelinePayload = (): DevopsPipelineForm => {
    const isTekton = isTektonEngine()
    if (isTekton) {
      ensureTektonConfigDefaults()
      syncParamsFromTektonDag(false)
      ensureTektonScheduleParamSnapshot()
      normalizeTektonScheduleParamSnapshot()
    }
    const dagContent = isTekton ? JSON.stringify(tektonDagConfig.value) : ''
    const triggerMode = isTekton
      ? tektonDagConfig.value.scheduleConfig?.enabled
        ? 'scheduled'
        : 'manual'
      : formData.triggers.some((item) => item.enabled && item.spec)
        ? 'scheduled'
        : 'manual'

    return {
      ...formData,
      projectId: projectStore.selectedProjectId,
      triggerMode,
      steps: isTekton ? [] : normalizedPipelineSteps(),
      params: formData.params.map((item) => ({
        ...item,
        sourceCode: item.sourceCode || item.code,
        stepNodeId: item.stepNodeId || '',
        defaultValue:
          item.defaultValue === undefined || item.defaultValue === null
            ? ''
            : String(item.defaultValue),
        currentValue: isChannelCredentialPipelineParam(item)
          ? ''
          : isCredentialPipelineParam(item) && !isLikelyObjectId(String(item.currentValue || ''))
            ? ''
            : item.currentValue === undefined || item.currentValue === null
              ? ''
              : String(item.currentValue),
        runtimeMode: effectiveParamRuntimeMode(item),
        runtimeConfig: isChannelCredentialPipelineParam(item) ? false : isRuntimeConfigParam(item),
        readonly: isChannelCredentialPipelineParam(item) ? true : item.readonly
      })),
      agent: isTekton ? { type: 'any' } : normalizedAgent(),
      tools: isTekton ? [] : formData.tools,
      options: isTekton ? [] : formData.options,
      triggers: isTekton ? [] : formData.triggers,
      scanMode: formData.scanMode || 'strict',
      scanItems: formData.scanEnabled
        ? (formData.scanItems || []).map((item) => ({ ...item }))
        : [],
      tektonDagConfig: dagContent,
      tektonRunPolicy: isTekton ? JSON.stringify(tektonDagConfig.value.runPolicy || {}) : '',
      tektonTriggerConfig: isTekton ? JSON.stringify(tektonTriggerPayload()) : '',
      tektonPrunerPolicyRef: isTekton
        ? JSON.stringify(tektonDagConfig.value.prunerPolicy || {})
        : '',
      tektonTemplateId: isTekton ? formData.tektonTemplateId || formData.templateId || '' : '',
      tektonTemplateSnapshot: isTekton ? formData.tektonTemplateSnapshot || '' : '',
      tektonParamBindings: isTekton ? formData.tektonParamBindings || '' : '',
      tektonWorkspaceBindings: isTekton ? formData.tektonWorkspaceBindings || '' : '',
      tektonResourceBindings: isTekton ? formData.tektonResourceBindings || '' : ''
    }
  }

  const normalizePipelineParamFormValues = () => {
    formData.params = formData.params.map((item) => {
      const currentValue =
        item.paramType === 'objectList'
          ? stringifyObjectListRows(parseObjectListRows(item.currentValue))
          : isChannelCredentialPipelineParam(item)
            ? ''
            : isCredentialPipelineParam(item) && !isLikelyObjectId(String(item.currentValue || ''))
              ? ''
              : item.currentValue === undefined || item.currentValue === null
                ? ''
                : String(item.currentValue)
      return {
        ...item,
        sourceCode: item.sourceCode || item.code,
        stepNodeId: item.stepNodeId || '',
        defaultValue:
          item.defaultValue === undefined || item.defaultValue === null
            ? ''
            : String(item.defaultValue),
        currentValue,
        runtimeMode: effectiveParamRuntimeMode(item),
        runtimeConfig: isChannelCredentialPipelineParam(item) ? false : isRuntimeConfigParam(item),
        readonly: isChannelCredentialPipelineParam(item) ? true : item.readonly
      }
    })
  }

  const isLikelyObjectId = (value: string) => /^[0-9a-fA-F]{24}$/.test(value.trim())

  const sanitizePipelineStepParamValues = (step: WorkflowNode) => {
    const values = parseNodeParamValues(step.paramValues)
    if (!Object.keys(values).length) return step.paramValues || ''
    const libraryStep = allStepLibrary.value.find(
      (item) => item.id === step.stepId || item.code === step.stepCode
    )
    ;(libraryStep?.params || []).forEach((param) => {
      const managedSecret =
        param.paramType === 'kubernetesSecretName' &&
        !!String(param.config?.voucherModel || '').trim()
      if (param.paramType !== 'voucherModel' && !managedSecret) return
      const value = String(values[param.code] || '').trim()
      if (value && !isLikelyObjectId(value)) {
        delete values[param.code]
      }
    })
    return Object.keys(values).length ? JSON.stringify(values, null, 2) : ''
  }

  const runTriggerTypeName = (triggerType?: string) => {
    if (triggerType === 'scheduled') return '定时'
    if (triggerType === 'manual') return '手动'
    if (triggerType) return triggerType
    return '-'
  }

  const tektonWorkspaceStrategyText = (strategy?: string) => {
    if (strategy === 'persistentVolumeClaim') return '统一使用 PVC'
    if (strategy === 'volumeClaimTemplate') return '统一使用动态 PVC'
    if (strategy === 'emptyDir') return '保持 EmptyDir'
    return '-'
  }

  const tektonWorkspaceBindingText = (type?: string) => {
    if (type === 'persistentVolumeClaim') return 'PVC'
    if (type === 'volumeClaimTemplate') return '动态 PVC'
    if (type === 'secret') return 'Secret'
    if (type === 'generatedSecret') return '平台生成 Secret'
    if (type === 'configMap') return 'ConfigMap'
    if (type === 'projected') return 'Projected'
    return 'EmptyDir'
  }

  const effectiveRunWorkspaceBindingType = (
    workspace: TektonDagWorkspaceItem,
    config: TektonDagConfig
  ) => {
    const bindingType = workspace.bindingType || 'emptyDir'
    const strategy = config.runPolicy?.workspaceStrategy
    return bindingType === 'emptyDir' && strategy ? strategy : bindingType
  }

  const runWorkspaceDefaultValue = (workspace: TektonDagWorkspaceItem, bindingType: string) => {
    if (bindingType === 'persistentVolumeClaim') return runtimeWorkspaceDefault(workspace.claimName)
    if (bindingType === 'secret') return runtimeWorkspaceDefault(workspace.secretName)
    if (bindingType === 'generatedSecret') return runtimeWorkspaceDefault(workspace.resourceId)
    if (bindingType === 'configMap') return runtimeWorkspaceDefault(workspace.configMapName)
    if (bindingType === 'projected') return ''
    return ''
  }

  const runtimeWorkspaceDefault = (value?: string) => {
    const text = String(value || '').trim()
    return text.startsWith('$(params.') ? '' : text
  }

  const runWorkspacePlaceholder = (bindingType: RunWorkspaceBindingType) => {
    if (bindingType === 'persistentVolumeClaim') return '请输入 PVC 名称'
    if (bindingType === 'secret') return '请输入 Secret 名称'
    if (bindingType === 'generatedSecret') return '请选择平台资源'
    if (bindingType === 'projected') return '运行时不覆盖 projected'
    return '请输入 ConfigMap 名称'
  }

  const buildRunWorkspaceItems = (config: TektonDagConfig): TektonRunWorkspaceItem[] =>
    (config.workspaces || [])
      .map((workspace) => {
        if (workspace.runtimeConfig !== true) return null
        const bindingType = effectiveRunWorkspaceBindingType(workspace, config)
        if (
          bindingType !== 'persistentVolumeClaim' &&
          bindingType !== 'secret' &&
          bindingType !== 'generatedSecret' &&
          bindingType !== 'configMap'
        ) {
          return null
        }
        const runtimeBindingType = bindingType as RunWorkspaceBindingType
        return {
          name: workspace.name,
          label: `${workspace.name} ${tektonWorkspaceBindingText(runtimeBindingType)}`,
          bindingType: runtimeBindingType,
          bindingLabel: tektonWorkspaceBindingText(runtimeBindingType),
          placeholder: runWorkspacePlaceholder(runtimeBindingType),
          optional: workspace.optional,
          defaultValue: runWorkspaceDefaultValue(workspace, runtimeBindingType)
        }
      })
      .filter((item): item is TektonRunWorkspaceItem => Boolean(item?.name))

  const runWorkspaceOptions = (item: TektonRunWorkspaceItem): TektonResourceOption[] => {
    if (item.bindingType === 'persistentVolumeClaim') return tektonPvcOptions.value
    if (item.bindingType === 'secret') return tektonSecretOptions.value
    if (item.bindingType === 'generatedSecret') {
      const workspace = runTargetPipelineTektonConfig.value?.workspaces.find(
        (value) => value.name === item.name
      )
      return workspace?.sourceType === 'configCenter'
        ? tektonProjectConfigOptions.value
        : tektonCredentialOptions.value
    }
    if (item.bindingType === 'configMap') return tektonConfigMapOptions.value
    return []
  }

  const buildTektonWorkspaceConfigItems = (config: TektonDagConfig): TektonProjectWorkspaceItem[] =>
    (config.workspaces || []).map((workspace) => {
      const itemConfig = workspace.config || {}
      return {
        ...workspace,
        bindingType: (workspace.bindingType || 'emptyDir') as ProjectWorkspaceBindingType,
        sourceType: workspace.sourceType || 'platformCredential',
        platformType: workspace.platformType || 'username_password',
        config: {
          ...itemConfig,
          projectedYaml:
            itemConfig.projectedYaml ||
            (workspace.projected ? JSON.stringify(workspace.projected, null, 2) : '')
        },
        runtimeConfig: workspace.runtimeConfig === true,
        storage: workspace.storage || '10Gi',
        accessMode: workspace.accessMode || 'ReadWriteOnce',
        volumeMode: workspace.volumeMode || ''
      }
    })

  const projectWorkspaceAllowedBindingTypes = (item: TektonProjectWorkspaceItem) => {
    const allowed = Array.isArray(item.allowedBindingTypes)
      ? item.allowedBindingTypes.filter(Boolean)
      : []
    return allowed.length ? allowed : [item.bindingType || 'emptyDir']
  }

  const projectWorkspaceBindingOptions = (item: TektonProjectWorkspaceItem) => {
    const allowed = new Set(projectWorkspaceAllowedBindingTypes(item))
    return workspaceBindingOptionList.filter((option) => allowed.has(option.value))
  }

  const generatedSecretSourceOptions = (item: TektonProjectWorkspaceItem) => {
    const allowed = Array.isArray(item.allowedSourceTypes)
      ? item.allowedSourceTypes.filter(Boolean)
      : []
    const enabled = allowed.length
      ? new Set(allowed)
      : new Set(['platformCredential', 'configCenter'])
    return generatedSecretSourceOptionList.filter((option) => enabled.has(option.value))
  }

  const generatedSecretPlatformTypeOptions = (item: TektonProjectWorkspaceItem) => {
    const allowed = Array.isArray(item.allowedPlatformTypes)
      ? item.allowedPlatformTypes.filter(Boolean)
      : []
    if (allowed.length === 0) return credentialTypeOptionList
    const allowedSet = new Set(allowed)
    return credentialTypeOptionList.filter((option) => allowedSet.has(option.value))
  }

  const ensureProjectWorkspaceAllowedValue = (item: TektonProjectWorkspaceItem) => {
    const options = projectWorkspaceBindingOptions(item)
    if (!options.some((option) => option.value === item.bindingType)) {
      item.bindingType = (options[0]?.value || 'emptyDir') as ProjectWorkspaceBindingType
    }
    if (item.bindingType === 'generatedSecret') {
      const sourceOptions = generatedSecretSourceOptions(item)
      if (!sourceOptions.some((option) => option.value === item.sourceType)) {
        item.sourceType = sourceOptions[0]?.value || 'platformCredential'
      }
      const platformOptions = generatedSecretPlatformTypeOptions(item)
      if (
        item.sourceType !== 'configCenter' &&
        !platformOptions.some((option) => option.value === item.platformType)
      ) {
        item.platformType = platformOptions[0]?.value || 'username_password'
      }
    }
  }

  const handleProjectWorkspaceBindingTypeChange = (item: TektonProjectWorkspaceItem) => {
    ensureProjectWorkspaceAllowedValue(item)
    item.claimName = ''
    item.secretName = ''
    item.configMapName = ''
    item.resourceId = ''
    item.resourceName = ''
    if (item.bindingType !== 'volumeClaimTemplate') {
      item.storageClassName = ''
      item.volumeMode = ''
    } else {
      item.storage = item.storage || '10Gi'
      item.accessMode = item.accessMode || 'ReadWriteOnce'
    }
    if (item.bindingType !== 'projected') {
      item.projected = undefined
      if (item.config) item.config.projectedYaml = ''
    }
  }

  const handleGeneratedSecretSourceChange = (item: TektonProjectWorkspaceItem) => {
    item.resourceId = ''
    item.resourceName = ''
    ensureProjectWorkspaceAllowedValue(item)
  }

  const projectWorkspaceResourceText = (item: TektonProjectWorkspaceItem) => {
    if (item.bindingType === 'persistentVolumeClaim') return item.claimName || '未选择 PVC'
    if (item.bindingType === 'secret') return item.secretName || '未选择 Secret'
    if (item.bindingType === 'configMap') return item.configMapName || '未选择 ConfigMap'
    if (item.bindingType === 'projected') {
      return item.projected ? 'Projected sources 已配置' : '未配置 Projected'
    }
    if (item.bindingType === 'volumeClaimTemplate') {
      return `动态 PVC ${item.storage || '10Gi'} / ${item.storageClassName || '默认 StorageClass'} / ${item.accessMode || 'ReadWriteOnce'} / ${item.volumeMode || '默认卷模式'}`
    }
    if (item.bindingType === 'generatedSecret') {
      const sourceLabel =
        generatedSecretSourceOptions(item).find((option) => option.value === item.sourceType)
          ?.label || '平台资源'
      return `${sourceLabel} / ${generatedSecretResourceName(item) || '未选择资源'}`
    }
    return '运行时使用临时目录'
  }

  const projectWorkspaceMissingReason = (item: TektonProjectWorkspaceItem) => {
    if (item.optional) return ''
    if (item.bindingType === 'persistentVolumeClaim' && !item.claimName) return '请选择 PVC'
    if (item.bindingType === 'secret' && !item.secretName) return '请选择 Secret'
    if (item.bindingType === 'configMap' && !item.configMapName) return '请选择 ConfigMap'
    if (item.bindingType === 'volumeClaimTemplate' && !item.storage) return '请填写动态 PVC 容量'
    if (item.bindingType === 'generatedSecret' && !item.resourceId) return '请选择平台资源'
    if (item.bindingType === 'projected' && !projectedWorkspaceSpec(item)) {
      return '请填写 Projected 配置'
    }
    return ''
  }

  const projectedWorkspaceSpec = (item: TektonProjectWorkspaceItem) => {
    const raw = String(item.config?.projectedYaml || '').trim()
    if (!raw) return item.projected || null
    try {
      const parsed = parseYaml(raw)
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
        return parsed as Record<string, any>
      }
    } catch {
      return null
    }
    return null
  }

  const openTektonWorkspaceItemEditor = (index: number) => {
    const item = tektonWorkspaceConfigItems.value[index]
    if (!item) return
    ensureProjectWorkspaceAllowedValue(item)
    editingTektonWorkspaceIndex.value = index
    tektonWorkspaceItemEditorSnapshot.value = clonePlain(item)
    tektonWorkspaceItemEditorVisible.value = true
  }

  const restoreTektonWorkspaceItemEditorSnapshot = () => {
    if (!tektonWorkspaceItemEditorSnapshot.value || editingTektonWorkspaceIndex.value < 0) return
    const item = tektonWorkspaceConfigItems.value[editingTektonWorkspaceIndex.value]
    if (item) Object.assign(item, clonePlain(tektonWorkspaceItemEditorSnapshot.value))
  }

  const handleTektonWorkspaceItemEditorClosed = () => {
    restoreTektonWorkspaceItemEditorSnapshot()
    tektonWorkspaceItemEditorSnapshot.value = null
    editingTektonWorkspaceIndex.value = -1
  }

  const cancelTektonWorkspaceItemEditor = () => {
    tektonWorkspaceItemEditorVisible.value = false
  }

  const saveTektonWorkspaceItemEditor = () => {
    const item = editingTektonWorkspaceItem.value
    if (!item) return
    ensureProjectWorkspaceAllowedValue(item)
    const reason = projectWorkspaceMissingReason(item)
    if (reason) {
      ElMessage.warning(`${item.name}：${reason}`)
      return
    }
    if (item.bindingType === 'projected') {
      item.projected = projectedWorkspaceSpec(item) || undefined
    }
    tektonWorkspaceItemEditorSnapshot.value = null
    tektonWorkspaceItemEditorVisible.value = false
  }

  const openTektonWorkspaceConfig = async () => {
    tektonWorkspaceConfigItems.value = buildTektonWorkspaceConfigItems(tektonDagConfig.value)
    tektonWorkspaceConfigItems.value.forEach(ensureProjectWorkspaceAllowedValue)
    await fetchTektonRuntimeResources()
    tektonWorkspaceConfigVisible.value = true
  }

  const generatedSecretResourceName = (item: TektonProjectWorkspaceItem) => {
    if (item.sourceType === 'configCenter') {
      return (
        tektonWorkspaceProjectConfigs.value.find((option) => option.id === item.resourceId)?.name ||
        ''
      )
    }
    return (
      tektonWorkspaceCredentials.value.find((option) => option.id === item.resourceId)?.name || ''
    )
  }

  const generatedSecretCredentialType = (item: TektonProjectWorkspaceItem) =>
    tektonWorkspaceCredentials.value.find((option) => option.id === item.resourceId)
      ?.credentialType ||
    item.platformType ||
    'username_password'

  const generatedSecretWorkspaceConfig = (item: TektonProjectWorkspaceItem) => {
    if (item.sourceType === 'configCenter') {
      const config = tektonWorkspaceProjectConfigs.value.find(
        (option) => option.id === item.resourceId
      )
      return {
        typeId: config?.typeId || '',
        typeCode: config?.typeCode || '',
        secretKey: config?.code || item.name || 'content'
      }
    }
    return {
      credentialType: generatedSecretCredentialType(item),
      provider: ''
    }
  }

  const saveTektonWorkspaceConfig = () => {
    const items = tektonWorkspaceConfigItems.value.map((item) => {
      const projected = item.bindingType === 'projected' ? projectedWorkspaceSpec(item) : null
      return {
        name: item.name,
        description: item.description || '',
        optional: item.optional === true,
        bindingType: item.bindingType,
        allowedBindingTypes: item.allowedBindingTypes || [],
        allowedSourceTypes: item.allowedSourceTypes || [],
        allowedPlatformTypes: item.allowedPlatformTypes || [],
        runtimeConfig: item.runtimeConfig === true,
        claimName: item.bindingType === 'persistentVolumeClaim' ? item.claimName || '' : '',
        secretName: item.bindingType === 'secret' ? item.secretName || '' : '',
        configMapName: item.bindingType === 'configMap' ? item.configMapName || '' : '',
        sourceType:
          item.bindingType === 'generatedSecret' ? item.sourceType || 'platformCredential' : '',
        platformType:
          item.bindingType === 'generatedSecret' && item.sourceType !== 'configCenter'
            ? generatedSecretCredentialType(item)
            : '',
        resourceId: item.bindingType === 'generatedSecret' ? item.resourceId || '' : '',
        resourceName:
          item.bindingType === 'generatedSecret'
            ? generatedSecretResourceName(item) || item.resourceName || ''
            : '',
        config:
          item.bindingType === 'generatedSecret'
            ? generatedSecretWorkspaceConfig(item)
            : item.bindingType === 'projected' && projected
              ? { projected }
              : {},
        projected: projected || undefined,
        storageClassName:
          item.bindingType === 'volumeClaimTemplate' ? item.storageClassName || '' : '',
        storage: item.bindingType === 'volumeClaimTemplate' ? item.storage || '10Gi' : '',
        accessMode:
          item.bindingType === 'volumeClaimTemplate' ? item.accessMode || 'ReadWriteOnce' : '',
        volumeMode: item.bindingType === 'volumeClaimTemplate' ? item.volumeMode || '' : '',
        subPath: item.subPath || ''
      }
    })
    const missing = items.find((item) => {
      if (item.optional) return false
      if (item.bindingType === 'persistentVolumeClaim') return !item.claimName
      if (item.bindingType === 'secret') return !item.secretName
      if (item.bindingType === 'configMap') return !item.configMapName
      if (item.bindingType === 'volumeClaimTemplate') return !item.storage
      if (item.bindingType === 'generatedSecret') return !item.resourceId
      if (item.bindingType === 'projected') return !item.projected
      return false
    })
    if (missing) {
      ElMessage.warning(`请完成 Workspace 绑定：${missing.name}`)
      return
    }
    tektonDagConfig.value.workspaces = items
    formData.tektonWorkspaceBindings = JSON.stringify(items)
    tektonWorkspaceConfigVisible.value = false
    ElMessage.success('Workspace 配置已保存')
  }

  const runWorkspaceValueMap = () =>
    runWorkspaceItems.value.reduce<Record<string, string>>((acc, item) => {
      acc[item.name] = String(runWorkspaceForm[item.name] || '').trim()
      return acc
    }, {})

  const tektonWorkspaceStableValueMap = (config: TektonDagConfig) =>
    (config.workspaces || []).reduce<Record<string, string>>((acc, item) => {
      const bindingType = effectiveRunWorkspaceBindingType(item, config)
      if (bindingType === 'persistentVolumeClaim') acc[item.name] = item.claimName || ''
      if (bindingType === 'secret') acc[item.name] = item.secretName || ''
      if (bindingType === 'generatedSecret') acc[item.name] = item.resourceId || ''
      if (bindingType === 'configMap') acc[item.name] = item.configMapName || ''
      return acc
    }, {})

  const tektonConcurrencyPolicyText = (policy?: string) => {
    if (policy === 'allow') return '允许并发'
    if (policy === 'skip') return '跳过新运行'
    if (policy === 'replace') return '替换运行'
    return '-'
  }

  const openCreateMainDialog = async () => {
    if (!projectStore.selectedProjectId) {
      ElMessage.warning('请选择项目')
      return
    }
    currentPipeline.value = null
    currentSteps.value = []
    templateSourceKey.value = ''
    templateChoiceId.value = ''
    templateKeyword.value = ''
    templateSourceFilter.value = 'all'
    Object.assign(formData, defaultForm())
    tektonDagConfig.value = defaultTektonDagConfig()
    mainConfigContext.value = 'create'
    mainConfigVisible.value = true
    mainConfigLoading.value = true
    try {
      await Promise.all([
        fetchSystems(),
        fetchEnvironments(),
        fetchBuildBindings(),
        fetchCategories()
      ])
      if (systems.value.length === 0) {
        ElMessage.warning('当前项目没有应用，请先到应用管理创建应用')
        mainConfigVisible.value = false
        return
      }
      formData.systemId = resolvePipelineSearchSystemId()
      formData.environmentId = searchForm.value.environmentId || environments.value[0]?.id || ''
    } finally {
      mainConfigLoading.value = false
    }
  }

  const openMainConfigInEditor = async () => {
    mainConfigContext.value = 'editor'
    mainConfigVisible.value = true
    mainConfigLoading.value = true
    try {
      await fetchBuildBindings()
    } finally {
      mainConfigLoading.value = false
    }
  }

  const openMainConfigFromCard = async (row: DevopsPipeline) => {
    mainConfigContext.value = 'card'
    currentPipeline.value = row
    fillFormFromPipeline(row)
    mainConfigVisible.value = true
    mainConfigLoading.value = true
    try {
      const [detail] = await Promise.all([
        getDevopsPipelineApi(row.id),
        fetchSystems(),
        fetchEnvironments(),
        fetchBuildBindings()
      ])
      currentPipeline.value = detail
      fillFormFromPipeline(detail)
    } finally {
      mainConfigLoading.value = false
    }
  }

  const submitMainConfig = async () => {
    if (systems.value.length === 0) {
      ElMessage.warning('当前项目没有应用，请先到应用管理创建应用')
      return
    }
    await mainFormRef.value?.validate()
    if (!validateMainFields()) return
    if (mainConfigContext.value === 'create') {
      mainConfigVisible.value = false
      if (formData.engineType === 'jenkins') {
        await fetchJenkinsAgents(true)
        await fetchTemplates()
        templateSourceVisible.value = true
        return
      } else {
        resetAgentCache()
        await fetchTemplates()
        templateSourceVisible.value = true
        return
      }
    }
    if (mainConfigContext.value === 'editor') {
      mainConfigVisible.value = false
      return
    }
    if (!currentPipeline.value) return
    submitLoading.value = true
    try {
      await updateDevopsPipelineApi(currentPipeline.value.id, pipelinePayload())
      ElMessage.success('主配置已保存')
      mainConfigVisible.value = false
      await fetchPipelines()
    } finally {
      submitLoading.value = false
    }
  }

  const startCustomPipeline = async () => {
    if (formData.engineType === 'tekton') {
      ElMessage.warning('Tekton 流水线必须选择 Pipeline 模板')
      templateSourceVisible.value = true
      return
    }
    templateSourceVisible.value = false
    loading.value = true
    formData.templateId = ''
    currentSteps.value = []
    formData.params = []
    if (formData.engineType === 'tekton') {
      tektonDagConfig.value = defaultTektonDagConfig()
      applyTektonRuntimeDefaults()
    }
    try {
      await runSequential([fetchAllSteps, fetchStepLibrary])
      workflowCanvasKey.value += 1
      editorMode.value = 'add'
      void fetchTektonRuntimeResources()
      await fitWorkflowCanvas()
    } finally {
      loading.value = false
    }
  }

  const startTemplatePipeline = async () => {
    const templateId = selectedTemplateChoice.value?.id || ''
    if (!templateId) {
      ElMessage.warning('请选择流水线模板')
      return
    }
    templateSourceVisible.value = false
    loading.value = true
    templateChoiceId.value = templateId
    formData.templateId = templateId
    try {
      await handleTemplateChange()
      await runSequential([fetchAllSteps, fetchStepLibrary])
      workflowCanvasKey.value += 1
      editorMode.value = 'add'
      void fetchTektonRuntimeResources()
      await fitWorkflowCanvas()
    } finally {
      loading.value = false
    }
  }

  const startSelectedTemplateSource = async () => {
    if (isCustomTemplateSource.value) {
      await startCustomPipeline()
      return
    }
    await startTemplatePipeline()
  }

  const openTriggerConfigInEditor = () => {
    triggerConfigFromCard.value = false
    triggerConfigVisible.value = true
  }

  const handleEditorConfigCommand = (cmd: string) => {
    if (cmd === 'main') return openMainConfigInEditor()
    if (cmd === 'jenkins') {
      fetchJenkinsAgents()
      jenkinsConfigVisible.value = true
      return
    }
    if (cmd === 'params') {
      if (isTektonEngine()) syncParamsFromTektonDag(false)
      paramConfigVisible.value = true
      return
    }
    if (cmd === 'trigger') return openTriggerConfigInEditor()
  }

  const openTriggerConfigFromCard = async (row: DevopsPipeline) => {
    const detail = await getDevopsPipelineApi(row.id)
    currentPipeline.value = detail
    fillFormFromPipeline(detail)
    triggerConfigFromCard.value = true
    triggerConfigVisible.value = true
  }

  const openTektonConfigFromCard = async (row: DevopsPipeline, tab: TektonConfigTab) => {
    const detail = await getDevopsPipelineApi(row.id)
    currentPipeline.value = detail
    currentRunPipeline.value = detail
    fillFormFromPipeline(detail)
    openTektonConfigDialog(tab, true)
  }

  const submitCardTriggerConfig = async () => {
    if (!currentPipeline.value) return
    submitLoading.value = true
    try {
      await updateDevopsPipelineApi(currentPipeline.value.id, pipelinePayload())
      ElMessage.success('定时配置已保存')
      triggerConfigVisible.value = false
      triggerConfigFromCard.value = false
      await fetchPipelines()
    } finally {
      submitLoading.value = false
    }
  }

  const handlePipelineCommand = async (cmd: string, row: DevopsPipeline) => {
    if (cmd === 'main') return openMainConfigFromCard(row)
    if (cmd === 'workflow') return enterEditor('edit', row)
    if (cmd === 'schedule') {
      if (row.engineType === 'tekton') return openTektonConfigFromCard(row, 'schedule')
      return openTriggerConfigFromCard(row)
    }
    if (cmd === 'records') return enterRunRecords(row)
    if (cmd === 'delete') return deletePipeline(row)
  }

  const handleEditorCancel = async () => {
    if (isViewMode.value) {
      exitEditor()
      return
    }
    try {
      await ElMessageBox.confirm('确认取消当前配置？未保存内容将丢失。', '取消确认', {
        type: 'warning'
      })
      exitEditor()
    } catch {
      return
    }
  }

  const handleEngineChange = async () => {
    if (isViewMode.value) return
    formData.templateId = ''
    formData.buildChannelBindingId = ''
    currentSteps.value = []
    formData.params = []
    if (formData.engineType === 'tekton') {
      tektonDagConfig.value = defaultTektonDagConfig()
    }
    await runSequential([fetchTemplates, fetchBuildBindings, fetchStepLibrary, fetchAllSteps])
    clearTektonRuntimeResources()
    resetAgentCache()
  }

  const handleBuildChannelChange = async () => {
    resetAgentCache()
    formData.params.forEach((item) => {
      if (!isExternalSecretPipelineParam(item) && !isPVCPipelineParam(item)) return
      item.currentValue = ''
      pipelineConfigParamOptions[item.code] = []
      pipelineConfigParamLoaded[item.code] = false
      pipelineConfigParamError[item.code] = ''
      pipelineConfigParamQueryKey[item.code] = ''
    })
    formData.agent = { type: 'any' }
    currentSteps.value = currentSteps.value.map((step) => ({ ...step, containerName: '' }))
    if (jenkinsConfigVisible.value) {
      await fetchJenkinsAgents(true)
    }
    applyTektonRuntimeDefaults()
    if (editorMode.value !== 'browse' && formData.engineType === 'tekton') {
      void fetchTektonRuntimeResources()
    } else {
      clearTektonRuntimeResources()
    }
  }

  const agentTypeName = (type?: string) => (isDynamicJenkinsAgent(type) ? '动态节点' : '静态节点')

  const agentMatchText = (agent: JenkinsAgentOption) => {
    if (isDynamicJenkinsAgent(agent.agentType)) {
      if (agent.cloud) return `Kubernetes 云：${agent.cloud}`
      return agent.matchValue ? `节点匹配：${agent.matchValue}` : 'Kubernetes 动态 Agent'
    }
    return `${agent.matchMode === 'name' ? '节点名称' : '节点标签'}：${agent.matchValue || '-'}`
  }

  const handleAgentSelectChange = (value: string) => {
    if (value === 'any' || value === 'none') {
      formData.agent = { type: value }
      currentSteps.value = currentSteps.value.map((step) => ({ ...step, containerName: '' }))
      return
    }
    const selected = jenkinsAgents.value.find((item) => item.id === value)
    if (!selected) return
    const agentType = normalizeJenkinsAgentType(selected.agentType)
    formData.agent = {
      type: agentType,
      id: selected.id,
      name: selected.name || selected.displayName,
      agentType,
      matchMode: selected.matchMode,
      matchValue: selected.matchValue || selected.label,
      cloud: selected.cloud,
      podYaml: selected.podYaml,
      containers: selected.containers || []
    }
    if (!isDynamicJenkinsAgent(agentType)) {
      currentSteps.value = currentSteps.value.map((step) => ({ ...step, containerName: '' }))
    }
  }

  const normalizeParamRuntimeMode = (runtimeMode?: string, mode?: string) => {
    if (mode === 'env') return 'env'
    if (mode === 'params') return 'params'
    if (runtimeMode === 'params' || runtimeMode === 'env') return runtimeMode
    return 'env'
  }

  const effectiveParamRuntimeMode = (item: PipelineParam) => {
    const mode = normalizeParamRuntimeMode(item.runtimeMode, item.mode)
    if (item.runtimeConfig === true && (item.mode === 'params' || item.mode === 'paas')) {
      return 'params'
    }
    return mode
  }

  const runtimeModeLabel = (item: PipelineParam, runtimeMode?: string) => {
    const mode = runtimeMode || effectiveParamRuntimeMode(item)
    if (isPlatformPipelineParam(item)) return mode === 'params' ? '执行变量' : '静态变量'
    return mode === 'params' ? '执行变量' : '环境变量'
  }

  const isRuntimeConfigParam = (item: PipelineParam) =>
    item.paramType !== 'channelCredential' &&
    item.runtimeConfig === true &&
    effectiveParamRuntimeMode(item) === 'params'

  const isDynamicRunParam = (item: PipelineParam) =>
    isManagedSecretPipelineParam(item) ||
    isExternalSecretPipelineParam(item) ||
    isPVCPipelineParam(item) ||
    (isPlatformPipelineParam(item) &&
      (channelParamTypes.includes(item.paramType) ||
        dynamicParamTypes.includes(item.paramType) ||
        (item.paramType === 'channelGroupCode' && !isDynamicChannelAddressParam(item)) ||
        item.paramType === 'voucherModel'))

  const isRunSelectParam = (item: PipelineParam) =>
    item.paramType === 'choice' || item.paramType === 'list' || isDynamicRunParam(item)

  const runParamOptions = (item: PipelineParam) => {
    if (isDynamicRunParam(item)) return dynamicRunParamOptions[item.code] || []
    return item.config?.options?.length ? item.config.options : item.selectList || []
  }

  const isKnownDynamicSelectValue = (
    value: unknown,
    options: DynamicSelectOption[] | undefined
  ) => {
    const text = String(value ?? '').trim()
    return !!text && !!options?.some((option) => dynamicOptionMatchesValue(option, text))
  }

  const preloadPipelineDynamicParamOptions = async () => {
    const items = formData.params.filter(
      (item) =>
        isPipelineDynamicConfigParam(item) && String(item.currentValue ?? '').trim()
    )
    await Promise.allSettled(
      items.map(async (item) => {
        if (isLegacyChannelPipelineParam(item) && !isDynamicChannelPipelineParam(item)) {
          await handlePipelineChannelParamVisible(item, true)
          pipelineConfigParamOptions[item.code] = normalizeSelectedDynamicOptions(
            pipelineChannelParamOptions(item).map(channelBindingDynamicOption),
            item.currentValue
          )
          pipelineConfigParamLoaded[item.code] = true
          pipelineConfigParamError[item.code] = ''
          return
        }
        await queryPipelineDynamicParamOptions(item, '', true)
        if (!isKnownDynamicSelectValue(item.currentValue, pipelineConfigParamOptions[item.code])) {
          await queryPipelineDynamicParamOptions(item, String(item.currentValue || ''), true)
        }
      })
    )
  }

  const runDynamicChannelValue = (item: PipelineParam) => {
    const channelCode = item.config?.channelParamCode
    const value = String(
      item.config?.channelBindingId ||
        (channelCode ? runParamForm[channelCode] || runPipelineParamValue(channelCode) : '')
    ).trim()
    if (value) return value
    const effectiveType = effectiveDynamicParamType(item)
    const projectValue = dynamicProjectValue(item)
    if (isGitRefDynamicType(effectiveType) && isGitRepositoryURLValue(projectValue)) return ''
    if (isImageRegistryDynamicType(effectiveType) && isImageRegistryAddressValue(projectValue))
      return ''
    if (
      isArtifactRepositoryDynamicType(effectiveType) &&
      isArtifactRepositoryAddressValue(projectValue)
    )
      return ''
    return value
  }

  const runParamValue = (code?: string) =>
    String(code ? runParamForm[code] || runPipelineParamValue(code) : '').trim()

  const preloadRunDynamicParamOptions = async () => {
    const items = runParamItems.value.filter(
      (item) => isDynamicRunParam(item) && String(runParamForm[item.code] ?? '').trim()
    )
    await Promise.allSettled(
      items.map(async (item) => {
        await queryRunParamOptions(item, '', true)
        if (!isKnownDynamicSelectValue(runParamForm[item.code], dynamicRunParamOptions[item.code])) {
          await queryRunParamOptions(item, String(runParamForm[item.code] || ''), true)
        }
      })
    )
  }

  const runDynamicPlaceholder = (item: PipelineParam) => {
    if (!isDynamicRunParam(item)) return item.defaultValue || '请选择'
    if (dynamicRunParamError[item.code]) return dynamicRunParamError[item.code]
    if (
      dynamicRunParamLoaded[item.code] &&
      !runParamOptionLoading[item.code] &&
      !runParamOptions(item).length
    ) {
      return dynamicEmptyText(item.paramType)
    }
    if (isLegacyChannelPipelineParam(item) && !isDynamicChannelPipelineParam(item)) {
      return item.defaultValue || '请选择渠道实例'
    }
    const effectiveType = effectiveDynamicParamType(item)
    const spec = dynamicParamSpec[effectiveType]
    if (
      item.paramType === 'channelGroupCode' &&
      hasMissingChannelDependency(item, runParamValue) &&
      !isGitRepositoryURLValue(dynamicProjectValue(item)) &&
      !isImageRegistryAddressValue(dynamicProjectValue(item)) &&
      !isArtifactRepositoryAddressValue(dynamicProjectValue(item))
    ) {
      return '可手动输入，选择依赖后可自动获取'
    }
    if (spec?.projectParamType && !dynamicProjectValue(item)) {
      return `可手动输入，选择${spec.projectLabel || '上游资源'}后可自动获取`
    }
    if (
      (dynamicParamTypes.includes(item.paramType) || isDynamicChannelPipelineParam(item)) &&
      dynamicParamSpec[effectiveType]?.channelParamType &&
      !runDynamicChannelValue(item) &&
      !isGitRepositoryURLValue(dynamicProjectValue(item)) &&
      !isImageRegistryAddressValue(dynamicProjectValue(item)) &&
      !isArtifactRepositoryAddressValue(dynamicProjectValue(item))
    ) {
      return '可手动输入，选择渠道实例后可自动获取'
    }
    if (channelParamTypes.includes(item.paramType)) {
      return item.defaultValue || '请选择渠道实例'
    }
    if (isExternalSecretPipelineParam(item)) return item.defaultValue || '请选择 Kubernetes Secret'
    if (isPVCPipelineParam(item)) return item.defaultValue || '请选择 Kubernetes PVC'
    if (isCredentialPipelineParam(item)) return item.defaultValue || '请选择凭证'
    return item.defaultValue || '请选择'
  }

  const paramTypeLabel = (item: PipelineParam) => {
    switch (item.paramType) {
      case 'text':
        return '多行文本'
      case 'booleanParam':
        return '布尔'
      case 'number':
        return '数字'
      case 'password':
        return '密码'
      case 'file':
        return '文件上传'
      case 'choice':
        return '单选'
      case 'objectList':
        return '对象列表'
      case 'list':
        return isPlatformPipelineParam(item) ? '平台列表' : '列表'
      case 'repositoryChannel':
        return '代码仓库渠道'
      case 'gitProject':
        return 'Git 项目'
      case 'gitRepositoryUrl':
        return 'Git 仓库地址'
      case 'gitBranch':
        return 'Git 分支'
      case 'gitTag':
        return 'Git Tag'
      case 'nexusChannel':
        return 'Nexus 渠道'
      case 'nexusRepository':
        return 'Nexus 仓库'
      case 'nexusRepositoryUrl':
        return 'Nexus 仓库地址'
      case 'harborChannel':
        return 'Harbor 渠道'
      case 'harborProject':
        return 'Harbor 项目'
      case 'harborProjectUrl':
        return 'Harbor 项目地址'
      case 'harborImage':
        return 'Harbor 镜像'
      case 'harborImageUrl':
        return 'Harbor 镜像地址'
      case 'harborImageTagUrl':
        return 'Harbor 镜像地址带 Tag'
      case 'jfrogChannel':
        return 'JFrog 渠道'
      case 'jfrogRepository':
        return 'JFrog 仓库'
      case 'jfrogRepositoryUrl':
        return 'JFrog 仓库地址'
      case 'kubeNovaDeployConfig':
        return 'kube-nova 配置'
      case 'kubernetesDeployConfig':
        return 'Kubernetes 配置'
      case 'mavenConfig':
      case 'configCenter':
        return '配置中心'
      case 'sonarAddress':
        return 'Sonar 地址'
      case 'sonarProjectKey':
        return 'Sonar 项目 Key'
      case 'hostGroupTargets':
        return '主机组目标'
      case 'voucherModel':
        return '凭证'
      case 'kubernetesSecretName':
        return isManagedSecretPipelineParam(item) ? '平台凭证 Secret' : 'Kubernetes Secret'
      case 'kubernetesPvcName':
        return 'Kubernetes PVC'
      case 'channelCredential':
        return '渠道凭证'
      default:
        return '字符串'
    }
  }

  const dynamicProjectValue = (item: PipelineParam) => {
    if (isGitRefDynamicType(effectiveDynamicParamType(item))) {
      return gitRefProjectDependencyValue(item, runParamValue)
    }
    if (isImageRegistryDynamicType(effectiveDynamicParamType(item))) {
      for (const field of imageRegistryProjectDependencyFields) {
        const paramCode = (item.config?.dependencyParamCodes || []).find(
          (dep: { field?: string; paramCode?: string }) => String(dep.field || '').trim() === field
        )?.paramCode
        const value = runParamValue(paramCode)
        if (value) return value
      }
    }
    if (isArtifactRepositoryDynamicType(effectiveDynamicParamType(item))) {
      for (const field of [
        'address.artifactVersionUrl',
        'address.artifactUrl',
        'address.repositoryUrl'
      ]) {
        const paramCode = (item.config?.dependencyParamCodes || []).find(
          (dep: { field?: string; paramCode?: string }) => String(dep.field || '').trim() === field
        )?.paramCode
        const value = runParamValue(paramCode)
        if (value) return value
      }
    }
    if (
      ['kubernetesResource', 'kubernetesContainer', 'kubernetesImage'].includes(
        effectiveDynamicParamType(item)
      )
    ) {
      const namespaceParamCode = (item.config?.dependencyParamCodes || []).find(
        (dep: { field?: string; paramCode?: string }) =>
          String(dep.field || '').trim() === 'dynamic.namespace'
      )?.paramCode
      const value = runParamValue(namespaceParamCode)
      if (value) return value
    }
    const projectCode = item.config?.projectParamCode || gitRefProjectDependencyParamCode(item)
    if (!projectCode) return ''
    return String(runParamForm[projectCode] || runPipelineParamValue(projectCode)).trim()
  }

  const runPipelineParamValue = (code?: string) => {
    if (!code || !runTargetPipeline.value) return ''
    const param = (runTargetPipeline.value.params || []).find((item) => item.code === code)
    return String(param?.currentValue || param?.defaultValue || '').trim()
  }

  const queryRunParamOptions = async (item: PipelineParam, keyword = '', force = false) => {
    if (!isDynamicRunParam(item) || !runTargetPipeline.value) return
    if (isExternalSecretPipelineParam(item) || isPVCPipelineParam(item)) {
      const projectId = runTargetPipeline.value.projectId || projectStore.selectedProjectId
      const bindingId = String(runTargetPipeline.value.buildChannelBindingId || '').trim()
      if (!projectId || !bindingId) {
        dynamicRunParamOptions[item.code] = []
        dynamicRunParamLoaded[item.code] = false
        dynamicRunParamError[item.code] = ''
        return
      }
      if (runParamOptionLoading[item.code]) return
      const queryKey = dynamicQueryKey(item, bindingId, '', keyword)
      if (
        !force &&
        !keyword.trim() &&
        dynamicRunParamLoaded[item.code] &&
        runParamQueryKey[item.code] === queryKey &&
        !dynamicRunParamError[item.code]
      ) {
        return
      }
      runParamQueryKey[item.code] = queryKey
      runParamOptionLoading[item.code] = true
      dynamicRunParamError[item.code] = ''
      try {
        if (isPVCPipelineParam(item)) {
          const res = await listDevopsTektonPvcApi({ projectId, bindingId, keyword })
          dynamicRunParamOptions[item.code] = normalizeSelectedDynamicOptions(
            (res.items || []).map((pvc) => ({
              label: `${pvc.name}（${pvc.storage || pvc.status || 'PVC'}）`,
              value: pvc.name,
              metadata: {
                namespace: pvc.namespace,
                storageClassName: pvc.storageClassName,
                accessModes: pvc.accessModes || []
              }
            })),
            runParamForm[item.code]
          )
        } else {
          const res = await listDevopsTektonSecretApi({
            projectId,
            bindingId,
            keyword
          })
          dynamicRunParamOptions[item.code] = normalizeSelectedDynamicOptions(
            (res.items || []).map((secret) => ({
              label: `${secret.name}（${secret.type || 'Opaque'}）`,
              value: secret.name,
              metadata: { namespace: secret.namespace, keys: secret.keys || [] }
            })),
            runParamForm[item.code]
          )
        }
        dynamicRunParamLoaded[item.code] = true
      } catch (error) {
        dynamicRunParamOptions[item.code] = []
        dynamicRunParamLoaded[item.code] = true
        dynamicRunParamError[item.code] = formatDynamicError(item.paramType, error)
      } finally {
        runParamOptionLoading[item.code] = false
      }
      return
    }
    if (isLegacyChannelPipelineParam(item) && !isDynamicChannelPipelineParam(item)) {
      if (runParamOptionLoading[item.code]) return
      runParamOptionLoading[item.code] = true
      dynamicRunParamError[item.code] = ''
      try {
        await fetchPipelineChannelBindings(item.config?.channelGroupCode || item.channelGroupCode)
        dynamicRunParamOptions[item.code] = normalizeSelectedDynamicOptions(
          pipelineChannelParamOptions(item).map(channelBindingDynamicOption),
          runParamForm[item.code]
        )
        dynamicRunParamLoaded[item.code] = true
      } catch (error) {
        dynamicRunParamOptions[item.code] = []
        dynamicRunParamLoaded[item.code] = true
        dynamicRunParamError[item.code] = formatDynamicError(item.paramType, error)
      } finally {
        runParamOptionLoading[item.code] = false
      }
      return
    }
    const projectValue = dynamicProjectValue(item)
    const effectiveType = effectiveDynamicParamType(item)
    if (dynamicParamSpec[effectiveType]?.projectParamType && !projectValue) {
      dynamicRunParamOptions[item.code] = []
      dynamicRunParamLoaded[item.code] = false
      dynamicRunParamError[item.code] = ''
      return
    }
    const channelBindingId = runDynamicChannelValue(item)
    if (
      item.paramType === 'channelGroupCode' &&
      dynamicParamSpec[effectiveType]?.channelParamType &&
      !channelBindingId &&
      !isGitRepositoryURLValue(projectValue) &&
      !isImageRegistryAddressValue(projectValue) &&
      !isArtifactRepositoryAddressValue(projectValue)
    ) {
      dynamicRunParamOptions[item.code] = []
      dynamicRunParamLoaded[item.code] = false
      dynamicRunParamError[item.code] = ''
      return
    }
    if (
      (dynamicParamTypes.includes(item.paramType) || isDynamicChannelPipelineParam(item)) &&
      dynamicParamSpec[effectiveType]?.channelParamType &&
      !channelBindingId &&
      !isGitRepositoryURLValue(projectValue) &&
      !isImageRegistryAddressValue(projectValue) &&
      !isArtifactRepositoryAddressValue(projectValue)
    ) {
      dynamicRunParamOptions[item.code] = []
      dynamicRunParamLoaded[item.code] = false
      dynamicRunParamError[item.code] = ''
      return
    }
    if (runParamOptionLoading[item.code]) return
    const dependencyValues = channelDependencyValuesText(
      channelDependencyValues(item, runParamValue)
    )
    const queryKey = dynamicQueryKey(
      item,
      channelBindingId,
      projectValue,
      keyword,
      dependencyValues
    )
    if (
      !force &&
      !keyword.trim() &&
      dynamicRunParamLoaded[item.code] &&
      runParamQueryKey[item.code] === queryKey &&
      !dynamicRunParamError[item.code]
    ) {
      return
    }
    runParamQueryKey[item.code] = queryKey
    runParamOptionLoading[item.code] = true
    dynamicRunParamError[item.code] = ''
    try {
      const res = await listDevopsDynamicParamOptionsApi({
        projectId: runTargetPipeline.value.projectId || projectStore.selectedProjectId,
        systemId: runTargetPipeline.value.systemId,
        environmentId: runTargetPipeline.value.environmentId,
        paramType: dynamicRequestParamType(item),
        channelBindingId,
        projectValue,
        componentValue: dynamicComponentValue(item, runParamValue),
        configTypeId: item.config?.configTypeId || '',
        configTypeCode:
          item.paramType === 'channelGroupCode'
            ? item.config?.channelGroupCode || ''
            : item.config?.configTypeCode || runParamValue(item.config?.componentParamCode),
        mappingField: item.config?.mappingField || '',
        dependencyValues,
        keyword,
        page: 1,
        pageSize: 50
      })
      dynamicRunParamOptions[item.code] = normalizeSelectedDynamicOptions(
        (res.items || []).map((option: DynamicParamOption) => ({
          label: option.label,
          value: option.value,
          metadata: parseDynamicOptionMetadata(option.metadata)
        })),
        runParamForm[item.code]
      )
      dynamicRunParamLoaded[item.code] = true
    } catch (error) {
      dynamicRunParamOptions[item.code] = []
      dynamicRunParamLoaded[item.code] = true
      dynamicRunParamError[item.code] = formatDynamicError(item.paramType, error)
    } finally {
      runParamOptionLoading[item.code] = false
    }
  }

  const handleRunParamVisibleChange = async (item: PipelineParam, visible: boolean) => {
    if (visible) await queryRunParamOptions(item, '', true)
  }

  const refreshRunParamOptions = async (item: PipelineParam) => {
    if (isLegacyChannelPipelineParam(item) && !isDynamicChannelPipelineParam(item)) {
      const groupCode = String(item.config?.channelGroupCode || item.channelGroupCode || '').trim()
      if (groupCode) delete pipelineChannelBindingLoadedGroups[groupCode]
    }
    await queryRunParamOptions(item, '', true)
  }

  const handleRunParamChange = (item: PipelineParam) => {
    runParamItems.value.forEach((target) => {
      if (
        target.config?.projectParamCode === item.code ||
        target.config?.credentialSourceParamCode === item.code ||
        (target.config?.dependencyParamCodes || []).some(
          (dep: { paramCode?: string }) => dep.paramCode === item.code
        )
      ) {
        if (
          isKnownDynamicSelectValue(runParamForm[target.code], dynamicRunParamOptions[target.code])
        ) {
          runParamForm[target.code] = ''
        }
        dynamicRunParamOptions[target.code] = []
        dynamicRunParamLoaded[target.code] = false
        dynamicRunParamError[target.code] = ''
        runParamQueryKey[target.code] = ''
      }
      if (target.config?.channelParamCode === item.code) {
        if (
          isKnownDynamicSelectValue(runParamForm[target.code], dynamicRunParamOptions[target.code])
        ) {
          runParamForm[target.code] = ''
        }
        dynamicRunParamOptions[target.code] = []
        dynamicRunParamLoaded[target.code] = false
        dynamicRunParamError[target.code] = ''
        runParamQueryKey[target.code] = ''
      }
    })
  }

  const handleTemplateChange = async () => {
    if (isViewMode.value) return
    if (!formData.templateId) {
      currentSteps.value = []
      formData.params = []
      if (isTektonEngine()) {
        formData.tektonTemplateId = ''
        formData.tektonTemplateSnapshot = ''
        tektonDagConfig.value = defaultTektonDagConfig()
      }
      return
    }
    const template = await getDevopsPipelineTemplateApi(formData.templateId)
    if (isTektonEngine()) {
      if (!template.tektonDagConfig) {
        ElMessage.warning('Tekton 模板缺少 Pipeline DAG 配置')
        return
      }
      formData.tektonTemplateId = template.id
      formData.templateId = template.id
      formData.tektonTemplateSnapshot = JSON.stringify(template)
      formData.tektonRunPolicy = template.tektonRunPolicy || ''
      formData.tektonTriggerConfig = template.tektonTriggerConfig || ''
      formData.tektonPrunerPolicyRef = template.tektonPrunerPolicyRef || ''
      tektonDagConfig.value = parseTektonDagConfig(template.tektonDagConfig)
      applyTektonRuntimeDefaults()
      const res = await listDevopsPipelineTemplateVariablesApi(formData.templateId)
      formData.params = orderedParams((res.items || []).map(variableToParam))
      resetPipelineParamOptionState()
      return
    }
    currentSteps.value = normalizeSteps(template.steps || [])
    const res = await listDevopsPipelineTemplateVariablesApi(formData.templateId)
    formData.params = (res.items || []).map(variableToParam)
    formData.params = orderedParams(formData.params)
    resetPipelineParamOptionState()
  }

  const variableToParam = (item: PipelineTemplateVariable): PipelineParam => ({
    name: item.name,
    code: item.code,
    sourceCode: item.sourceCode || item.code,
    stepNodeId: item.stepNodeId || item.stepId,
    defaultValue: item.defaultValue,
    currentValue:
      item.paramType === 'channelCredential'
        ? ''
        : suggestParamValue(item.code, item.currentValue, item.defaultValue),
    paramType: item.paramType,
    mode: item.mode,
    required: item.required,
    readonly: item.paramType === 'channelCredential' ? true : item.readonly,
    description: item.description,
    selectList: item.selectList || [],
    allowCustom: item.allowCustom,
    sortOrder: item.sortOrder || 0,
    runtimeMode:
      item.runtimeConfig === true
        ? 'params'
        : normalizeParamRuntimeMode(item.runtimeMode, item.mode),
    runtimeConfig:
      item.paramType === 'channelCredential'
        ? false
        : item.runtimeConfig === true && effectiveParamRuntimeMode(item) === 'params',
    config: item.config || {}
  })

  const suggestParamValue = (code: string, currentValue?: string, defaultValue?: string) => {
    if (String(currentValue || '').trim()) return currentValue || ''
    if (String(defaultValue || '').trim()) return defaultValue || ''
    if (String(code || '').toUpperCase() === 'APP_NAME') {
      return selectedSystem.value?.code || selectedSystem.value?.name || ''
    }
    return ''
  }

  const paramSourceCode = (item: PipelineParam) => item.sourceCode || item.code
  const paramInstanceKey = (stepNodeId?: string, sourceCode?: string) =>
    `${stepNodeId || ''}\u0000${sourceCode || ''}`

  const runtimeParamCode = (sourceCode: string, codeMap: Record<string, string>) =>
    codeMap[sourceCode] || sourceCode

  const canUseRuntimeParamCode = (
    code: string,
    sourceCode: string,
    used: Set<string>,
    reserved: Set<string>
  ) => Boolean(code) && !used.has(code) && (!reserved.has(code) || code === sourceCode)

  const nextRuntimeParamCode = (
    sourceCode: string,
    occurrence: Record<string, number>,
    used: Set<string>,
    reserved: Set<string>
  ) => {
    occurrence[sourceCode] = (occurrence[sourceCode] || 0) + 1
    if (occurrence[sourceCode] === 1 && !used.has(sourceCode)) {
      used.add(sourceCode)
      return sourceCode
    }
    for (let index = Math.max(2, occurrence[sourceCode]); ; index++) {
      const code = `${sourceCode}_${index}`
      if (used.has(code)) continue
      if (reserved.has(code) && code !== sourceCode) continue
      used.add(code)
      return code
    }
  }

  const remapParamConfig = (
    config: StepParamConfig | undefined,
    codeMap: Record<string, string>
  ) => ({
    ...(config || {}),
    channelParamCode: runtimeParamCode(config?.channelParamCode || '', codeMap),
    projectParamCode: runtimeParamCode(config?.projectParamCode || '', codeMap),
    componentParamCode: runtimeParamCode(config?.componentParamCode || '', codeMap),
    credentialSourceParamCode: runtimeParamCode(config?.credentialSourceParamCode || '', codeMap)
  })

  const tektonDagParamType = (type?: string) => {
    if (type === 'array') return 'list'
    if (type === 'object') return 'objectList'
    return 'string'
  }

  const buildPipelineParamsFromTektonDag = (
    config: TektonDagConfig,
    existingParams: PipelineParam[] = []
  ): PipelineParam[] => {
    const existingByCode = new Map(existingParams.map((item) => [item.code, item]))
    return orderedParams(
      (config.params || []).map<PipelineParam>((item, index) => {
        const old = existingByCode.get(item.name)
        const defaultValue = item.defaultValue ?? ''
        const paramType = item.paramType || tektonDagParamType(item.type)
        return {
          name: item.displayName || item.name,
          code: item.name,
          sourceCode: item.sourceCode || item.name,
          stepNodeId: '',
          defaultValue,
          currentValue:
            paramType === 'channelCredential' ? '' : (old?.currentValue ?? defaultValue),
          paramType,
          mode: 'params',
          required: item.required === true,
          readonly: paramType === 'channelCredential' ? true : item.readonly === true,
          description: item.description || '',
          selectList: item.selectList || item.options || [],
          allowCustom: item.allowCustom ?? true,
          sortOrder: index + 1,
          runtimeMode: 'params',
          runtimeConfig: paramType === 'channelCredential' ? false : item.runtimeConfig === true,
          config: item.config || {
            options: item.options || item.selectList || []
          }
        }
      })
    )
  }

  const syncParamsFromTektonDag = (showMessage = false) => {
    formData.params = buildPipelineParamsFromTektonDag(tektonDagConfig.value, formData.params)
    resetPipelineParamOptionState()
    if (showMessage) ElMessage.success('Tekton Pipeline 参数已同步')
  }

  const tektonPipelineParamValueMap = (params: PipelineParam[]) =>
    params.reduce<Record<string, string>>((acc, item) => {
      if (!item.code) return acc
      acc[item.code] = String(item.currentValue ?? item.defaultValue ?? '')
      return acc
    }, {})

  const syncParamsFromSteps = async (showMessage = false) => {
    await fetchAllSteps()
    const stepMap = new Map(allStepLibrary.value.map((item) => [item.id, item]))
    const existingByKey = new Map(
      formData.params
        .filter((item) => item.stepNodeId && paramSourceCode(item))
        .map((item) => [paramInstanceKey(item.stepNodeId, paramSourceCode(item)), item])
    )
    const legacyByCode = formData.params.reduce<Record<string, PipelineParam[]>>((acc, item) => {
      if (item.stepNodeId) return acc
      const sourceCode = paramSourceCode(item)
      if (!acc[sourceCode]) acc[sourceCode] = []
      acc[sourceCode].push(item)
      return acc
    }, {})
    const reserved = new Set<string>()
    currentSteps.value.forEach((node) => {
      const step = stepMap.get(node.stepId)
      ;(step?.params || []).forEach((param) => {
        if (param.code) reserved.add(param.code)
      })
    })
    const occurrence: Record<string, number> = {}
    const used = new Set<string>()
    const codeMaps: Record<string, Record<string, string>> = {}
    currentSteps.value.forEach((node) => {
      const step = stepMap.get(node.stepId)
      const stepNodeId = node.id || ''
      codeMaps[stepNodeId] = {}
      ;(step?.params || []).forEach((param) => {
        const sourceCode = param.code
        const old = existingByKey.get(paramInstanceKey(stepNodeId, sourceCode))
        let code = ''
        if (old && canUseRuntimeParamCode(old.code, sourceCode, used, reserved)) {
          code = old.code
          used.add(code)
          occurrence[sourceCode] = (occurrence[sourceCode] || 0) + 1
        }
        if (!code) {
          code = nextRuntimeParamCode(sourceCode, occurrence, used, reserved)
        }
        codeMaps[stepNodeId][sourceCode] = code
      })
    })
    const legacyIndex: Record<string, number> = {}
    const next: PipelineParam[] = []
    currentSteps.value.forEach((node) => {
      const step = stepMap.get(node.stepId)
      const nodeValues = parseNodeParamValues(node.paramValues)
      const stepNodeId = node.id || ''
      ;(step?.params || []).forEach((param) => {
        const sourceCode = param.code
        const code = runtimeParamCode(sourceCode, codeMaps[stepNodeId] || {})
        let old = existingByKey.get(paramInstanceKey(stepNodeId, sourceCode))
        if (!old) {
          const items = legacyByCode[sourceCode] || []
          const index = legacyIndex[sourceCode] || 0
          old = items[index]
          legacyIndex[sourceCode] = index + 1
        }
        const nodeValue = nodeValues[param.code]
        const currentValue =
          param.paramType === 'objectList' && Array.isArray(nodeValue)
            ? JSON.stringify(nodeValue, null, 2)
            : Array.isArray(nodeValue)
              ? nodeValue.join(',')
              : String(nodeValue ?? '').trim()
        const runtimeMode =
          param.runtimeConfig === true
            ? 'params'
            : normalizeParamRuntimeMode(param.runtimeMode, param.mode)
        next.push({
          name: param.name,
          code,
          sourceCode,
          stepNodeId,
          defaultValue: param.defaultValue || '',
          currentValue:
            param.paramType === 'channelCredential'
              ? ''
              : suggestParamValue(code, currentValue || old?.currentValue, param.defaultValue),
          paramType: param.paramType,
          mode: param.mode,
          required: param.required,
          readonly: param.paramType === 'channelCredential' ? true : param.readonly,
          description: param.description,
          selectList: param.selectList || [],
          allowCustom: param.allowCustom,
          sortOrder: param.sortOrder || 0,
          runtimeMode,
          runtimeConfig:
            param.paramType === 'channelCredential'
              ? false
              : param.runtimeConfig === true && runtimeMode === 'params',
          config: remapParamConfig(param.config, codeMaps[stepNodeId] || {})
        })
      })
    })
    formData.params = next
    formData.params = orderedParams(formData.params)
    resetPipelineParamOptionState()
    if (showMessage) ElMessage.success('参数已同步')
  }

  const syncParamsFromCurrentDesigner = async (showMessage = false) => {
    if (isTektonEngine()) {
      syncParamsFromTektonDag(showMessage)
      return
    }
    await syncParamsFromSteps(showMessage)
  }

  const validatePipelineParams = () => {
    for (const item of formData.params) {
      if (item.paramType !== 'objectList') continue
      const message = validateObjectListRows(item, item.currentValue || item.defaultValue)
      if (message) {
        ElMessage.warning(message)
        paramConfigVisible.value = true
        return false
      }
    }
    const missing = formData.params.find(
      (item) =>
        item.mode === 'env' &&
        item.required &&
        !String(item.currentValue || item.defaultValue || '').trim()
    )
    if (missing) {
      ElMessage.warning(`请配置必填环境变量：${missing.name}`)
      paramConfigVisible.value = true
      return false
    }
    return true
  }

  const parseNodeParamValues = (raw?: string) => {
    if (!String(raw || '').trim()) return {}
    try {
      const parsed = JSON.parse(raw || '{}')
      return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
        ? (parsed as Record<string, any>)
        : {}
    } catch {
      return {}
    }
  }

  const isEmptyNodeParamValue = (value: any) => {
    if (Array.isArray(value)) return value.length === 0
    return !String(value ?? '').trim()
  }

  const findMissingStepParam = (params: DevopsStepParam[], values: Record<string, any>) =>
    params.find(
      (item) =>
        item.required &&
        item.paramType !== 'channelCredential' &&
        isEmptyNodeParamValue(values[item.code]) &&
        isEmptyNodeParamValue(item.defaultValue)
    )

  const validateStepParamValues = () => {
    const stepMap = new Map(allStepLibrary.value.map((item) => [item.id, item]))
    const missingNode = currentSteps.value.find((node) => {
      const step = stepMap.get(node.stepId)
      const missing = findMissingStepParam(
        step?.params || [],
        parseNodeParamValues(node.paramValues)
      )
      return Boolean(missing)
    })
    if (!missingNode) return true

    const step = stepMap.get(missingNode.stepId)
    const missing = findMissingStepParam(
      step?.params || [],
      parseNodeParamValues(missingNode.paramValues)
    )
    ElMessage.warning(
      `请配置阶段「${missingNode.nodeName || missingNode.stepName}」参数：${missing?.name || missing?.code}`
    )
    return false
  }

  const validateTektonDagBeforeSave = () => {
    ensureTektonConfigDefaults()
    applyTektonRuntimeDefaults()
    syncParamsFromTektonDag(false)
    const issues = validateTektonDagConfig(tektonDagConfig.value, {
      paramValues: tektonPipelineParamValueMap(formData.params)
    })
    const firstError = issues.find((item) => item.level === 'error')
    if (firstError) {
      ElMessage.warning(firstError.message)
      return false
    }
    return true
  }

  const validateTektonDagBeforeRun = (row: DevopsPipeline) => {
    if (row.engineType !== 'tekton') return true
    const config = parseTektonDagConfigFromPipeline(row)
    const params = buildPipelineParamsFromTektonDag(config, row.params || [])
    const issues = validateTektonDagConfig(config, {
      paramValues: tektonPipelineParamValueMap(params)
    })
    const firstError = issues.find((item) => item.level === 'error')
    if (firstError) {
      ElMessage.warning(firstError.message)
      return false
    }
    return true
  }

  const savePipeline = async () => {
    if (isViewMode.value) return
    if (!projectStore.selectedProjectId) {
      ElMessage.warning('请选择项目')
      return
    }
    if (!validateMainFields()) return
    const isTekton = isTektonEngine()
    if (isTekton) {
      if (!validateTektonDagBeforeSave()) return
      if (!validateTektonPolicyConfigBeforeSave()) return
      syncParamsFromTektonDag(false)
    } else {
      await syncParamsFromSteps(false)
    }
    normalizePipelineParamFormValues()
    if (!isTekton && currentSteps.value.length === 0) {
      ElMessage.warning('请配置流水线步骤')
      return
    }
    if (!isTekton && !validateStepParamValues()) return
    if (!validatePipelineParams()) return
    try {
      await ElMessageBox.confirm(
        `确认保存流水线？保存后将同步 ${engineTypeName(formData.engineType)} 资源。`,
        '保存确认',
        {
          type: 'warning'
        }
      )
    } catch {
      return
    }
    submitLoading.value = true
    try {
      const payload = pipelinePayload()
      if (editorMode.value === 'add') {
        const res = await createDevopsPipelineApi(payload)
        const detail = await getDevopsPipelineApi(res.id)
        currentPipeline.value = detail
        fillFormFromPipeline(detail)
        ElMessage.success('新增成功')
      } else if (currentPipeline.value) {
        await updateDevopsPipelineApi(currentPipeline.value.id, payload)
        const detail = await getDevopsPipelineApi(currentPipeline.value.id)
        currentPipeline.value = detail
        fillFormFromPipeline(detail)
        ElMessage.success('保存成功')
      }
      await fetchPipelines()
      if (isTekton && currentPipeline.value) {
        editorMode.value = 'view'
        tektonDetailActiveTab.value = 'overview'
        workflowCanvasKey.value += 1
        return
      }
      exitEditor()
    } catch {
      return
    } finally {
      submitLoading.value = false
    }
  }

  const deletePipeline = async (row: DevopsPipeline) => {
    await ElMessageBox.confirm(`确认删除流水线「${row.name}」？`, '删除确认', { type: 'warning' })
    await deleteDevopsPipelineApi(row.id)
    ElMessage.success('删除成功')
    await fetchPipelines()
  }

  const resetRunParamState = () => {
    runTargetPipelineTektonConfig.value = null
    runParamInitError.value = ''
    runParamItems.value = []
    runWorkspaceItems.value = []
    Object.keys(runParamForm).forEach((key) => delete runParamForm[key])
    Object.keys(runWorkspaceForm).forEach((key) => delete runWorkspaceForm[key])
    Object.keys(dynamicRunParamOptions).forEach((key) => delete dynamicRunParamOptions[key])
    Object.keys(runParamOptionLoading).forEach((key) => delete runParamOptionLoading[key])
    Object.keys(dynamicRunParamLoaded).forEach((key) => delete dynamicRunParamLoaded[key])
    Object.keys(dynamicRunParamError).forEach((key) => delete dynamicRunParamError[key])
    Object.keys(runParamQueryKey).forEach((key) => delete runParamQueryKey[key])
  }

  const fillRunParamState = (
    runDetail: DevopsPipeline,
    tektonRunConfig: TektonDagConfig | null
  ) => {
    runParamInitError.value = ''
    runTargetPipeline.value = runDetail
    runTargetPipelineTektonConfig.value = tektonRunConfig
    runParamItems.value = orderedRuntimeParams(
      (runDetail.params || []).filter(isRuntimeConfigParam),
      runDetail.steps || []
    )
    runWorkspaceItems.value = tektonRunConfig ? buildRunWorkspaceItems(tektonRunConfig) : []
    runParamItems.value.forEach((item) => {
      if (item.paramType === 'objectList') {
        runParamForm[item.code] = stringifyObjectListRows(
          parseObjectListRows(item.currentValue || item.defaultValue)
        )
        return
      }
      runParamForm[item.code] = String(item.currentValue || item.defaultValue || '')
      if (item.paramType === 'booleanParam' && !runParamForm[item.code]) {
        runParamForm[item.code] = 'false'
      }
      if (item.paramType === 'number' && runParamForm[item.code] !== '') {
        runParamForm[item.code] = Number(runParamForm[item.code])
      }
      if (isDynamicRunParam(item)) {
        dynamicRunParamOptions[item.code] = []
        dynamicRunParamLoaded[item.code] = false
        dynamicRunParamError[item.code] = ''
      }
    })
    runWorkspaceItems.value.forEach((item) => {
      runWorkspaceForm[item.name] = item.defaultValue || ''
    })
  }

  const runPipeline = async (row: DevopsPipeline) => {
    if (row.status !== 1) {
      ElMessage.warning('流水线已停用，不能运行')
      return
    }
    const requestSeq = ++runParamRequestSeq
    runTargetPipeline.value = row
    resetRunParamState()
    runParamVisible.value = true
    runParamInitializing.value = true
    await nextTick()
    try {
      const detail = await getDevopsPipelineApi(row.id)
      if (requestSeq !== runParamRequestSeq) return
      if (detail.status !== 1) {
        ElMessage.warning('流水线已停用，不能运行')
        runParamVisible.value = false
        await fetchPipelines()
        return
      }
      if (!validateTektonDagBeforeRun(detail)) {
        runParamVisible.value = false
        return
      }
      const tektonRunConfig =
        detail.engineType === 'tekton' ? parseTektonDagConfigFromPipeline(detail) : null
      if (tektonRunConfig) {
        await fetchTektonRuntimeResources(detail)
        if (requestSeq !== runParamRequestSeq) return
      }
      const runDetail = tektonRunConfig
        ? {
            ...detail,
            params: buildPipelineParamsFromTektonDag(tektonRunConfig, detail.params || [])
          }
        : detail
      fillRunParamState(runDetail, tektonRunConfig)
    } catch (error) {
      if (requestSeq === runParamRequestSeq) {
        runParamInitError.value =
          extractErrorMessage(error).trim() || '初始化流水线参数表单失败，请稍后重试'
      }
    } finally {
      if (requestSeq === runParamRequestSeq) {
        runParamInitializing.value = false
      }
    }
  }

  const handleRunParamDialogClosed = () => {
    runParamRequestSeq += 1
    runParamInitializing.value = false
  }

  const validateRunParams = () => {
    for (const item of runParamItems.value) {
      if (item.paramType !== 'objectList') continue
      const message = validateObjectListRows(item, runParamForm[item.code])
      if (message) {
        ElMessage.warning(message)
        return false
      }
    }
    const missing = runParamItems.value.find(
      (item) => item.required && !String(runParamForm[item.code] ?? '').trim()
    )
    if (missing) {
      ElMessage.warning(`请填写运行配置：${missing.name}`)
      return false
    }
    const missingWorkspace = runWorkspaceItems.value.find(
      (item) => !item.optional && !String(runWorkspaceForm[item.name] ?? '').trim()
    )
    if (missingWorkspace) {
      ElMessage.warning(`请填写 Workspace 绑定：${missingWorkspace.label}`)
      return false
    }
    if (runTargetPipeline.value?.engineType === 'tekton') {
      const config = parseTektonDagConfigFromPipeline(runTargetPipeline.value)
      const issues = validateTektonDagConfig(config, {
        paramValues: {
          ...tektonPipelineParamValueMap(runTargetPipeline.value.params || []),
          ...runParamItems.value.reduce<Record<string, string>>((acc, item) => {
            acc[item.code] = String(runParamForm[item.code] ?? '')
            return acc
          }, {})
        },
        workspaceValues: {
          ...tektonWorkspaceStableValueMap(config),
          ...runWorkspaceValueMap()
        },
        requireWorkspaceValues: true
      })
      const firstError = issues.find((item) => item.level === 'error')
      if (firstError) {
        ElMessage.warning(firstError.message)
        return false
      }
    }
    return true
  }

  const submitRunPipeline = async () => {
    if (!runTargetPipeline.value) return
    if (!validateRunParams()) return
    await ElMessageBox.confirm(`确认运行流水线「${runTargetPipeline.value.name}」？`, '运行确认', {
      type: 'warning'
    })
    runSubmitLoading.value = true
    try {
      const params = runParamItems.value.reduce(
        (acc, item) => {
          acc[item.code] =
            item.paramType === 'number' && runParamForm[item.code] !== undefined
              ? String(runParamForm[item.code])
              : (runParamForm[item.code] ?? '')
          return acc
        },
        {} as Record<string, string>
      )
      const workspaces = runWorkspaceItems.value.reduce(
        (acc, item) => {
          const value = String(runWorkspaceForm[item.name] || '').trim()
          if (value) acc[item.name] = value
          return acc
        },
        {} as Record<string, string>
      )
      const res = await runDevopsPipelineApi(runTargetPipeline.value.id, {
        triggerType: 'manual',
        params: JSON.stringify(params),
        workspaces: JSON.stringify(workspaces)
      })
      ElMessage.success('已触发运行')
      runParamVisible.value = false
      currentRunPipeline.value = runTargetPipeline.value
      runPagination.current = 1
      const pendingRun = createPendingRunRecord(runTargetPipeline.value, res.id)
      runs.value = [pendingRun, ...runs.value.filter((item) => item.id !== pendingRun.id)]
      await enterRunProcess(pendingRun, 'stage')
      void fetchRuns()
      void fetchPipelines()
    } finally {
      runSubmitLoading.value = false
    }
  }

  const createPendingRunRecord = (pipeline: DevopsPipeline, runId: string): DevopsPipelineRun => ({
    id: runId,
    projectId: pipeline.projectId,
    pipelineId: pipeline.id,
    pipelineName: pipeline.name,
    pipelineCode: pipeline.code,
    projectCode: pipeline.projectCode,
    projectName: pipeline.projectName,
    systemId: pipeline.systemId,
    systemCode: pipeline.systemCode,
    systemName: pipeline.systemName,
    environmentId: pipeline.environmentId,
    environmentCode: pipeline.environmentCode,
    environmentName: pipeline.environmentName,
    engineType: pipeline.engineType,
    jenkinsJobName: pipeline.jobName,
    jenkinsJobFullName: pipeline.jobFullName,
    jenkinsBuildNumber: 0,
    jenkinsQueueId: '',
    buildId: 0,
    tektonNamespace: pipeline.tektonNamespace || '',
    tektonPipelineName: pipeline.tektonPipelineName || '',
    tektonPipelineRunName: '',
    triggerType: 'manual',
    triggerUserId: 0,
    triggerUsername: '',
    status: 'preparing',
    startedAt: new Date().toISOString(),
    finishedAt: '',
    durationSeconds: 0,
    errorMessage: '',
    logStorageType: '',
    logObjectKey: '',
    logSize: 0,
    logArchivedAt: ''
  })

  const openJenkinsBuild = (url?: string) => {
    if (!url) return
    window.open(url, '_blank', 'noopener,noreferrer')
  }

  const enterRunRecords = async (row: DevopsPipeline) => {
    currentRunPipeline.value = row
    runPagination.current = 1
    editorMode.value = 'records'
    await fetchRuns()
  }

  const fetchRuns = async () => {
    if (!currentRunPipeline.value) return
    runLoading.value = true
    try {
      const res = await listDevopsPipelineRunApi({
        page: runPagination.current,
        pageSize: runPagination.size,
        projectId: projectStore.selectedProjectId,
        pipelineId: currentRunPipeline.value.id
      })
      runs.value = res.items || []
      runPagination.total = res.total || 0
    } finally {
      runLoading.value = false
    }
  }

  const refreshLogRelatedLists = async () => {
    const tasks: Promise<unknown>[] = []
    if (currentRunPipeline.value) tasks.push(fetchRuns())
    if (projectStore.selectedProjectId) tasks.push(fetchPipelines(false))
    if (tasks.length === 0) return
    await Promise.allSettled(tasks)
  }

  const handleRunSizeChange = (size: number) => {
    runPagination.size = size
    runPagination.current = 1
    fetchRuns()
  }

  const handleRunCurrentChange = (page: number) => {
    runPagination.current = page
    fetchRuns()
  }

  const virtualRunStagePrefix = 'virtual-stage-'

  const isVirtualRunStage = (stage?: DevopsPipelineRunStage) =>
    Boolean(stage?.id.startsWith(virtualRunStagePrefix))

  const applyProcessStageDependencyGuard = (stages: DevopsPipelineRunStage[]) => {
    return applyProcessStageDependencyGuardCore({
      stages,
      pipelineSteps: processPipelineSteps.value,
      runActive: Boolean(currentRun.value && isRunActive(currentRun.value.status)),
      normalizeRunStatus
    })
  }

  const displayRunStages = computed(() => {
    const stages = [...runStages.value]
    if (!currentRun.value || processPipelineSteps.value.length === 0) return stages

    const stageNodeIds = new Set(stages.map((stage) => stage.nodeId).filter(Boolean))
    const stepById = new Map<string, PipelineTemplateStep>()
    processPipelineSteps.value.forEach((step) => {
      if (step.id) stepById.set(step.id, step)
    })
    const activeStatuses = new Set(['preparing', 'running', 'queued', 'paused'])
    const currentStatus = normalizeRunStatus(currentRun.value.status)
    const virtualStatus = activeStatuses.has(currentStatus) ? 'queued' : 'skipped'
    const virtualStages = processPipelineSteps.value
      .filter((step) => step.id && step.enabled !== false && !stageNodeIds.has(step.id))
      .sort(compareProcessStep)
      .map<DevopsPipelineRunStage>((step) => {
        const rootType = processStepRootType(step, stepById)
        return {
          id: `${virtualRunStagePrefix}${step.id}`,
          runId: currentRun.value?.id || '',
          pipelineId: currentRun.value?.pipelineId || currentRunPipeline.value?.id || '',
          nodeId: step.id || '',
          stageName: step.nodeName || step.stepName || step.stepCode || '阶段',
          stageType: rootType === 'post' ? 'post' : 'stage',
          status: virtualStatus,
          startedAt: '',
          finishedAt: '',
          durationSeconds: 0,
          jenkinsNodeId: '',
          tektonTaskRunName: '',
          tektonPodName: '',
          containerName: '',
          logObjectKey: '',
          errorMessage: ''
        }
      })
    const stepOrder = new Map<string, number>()
    ;[...processPipelineSteps.value].sort(compareProcessStep).forEach((step, index) => {
      if (step.id) stepOrder.set(step.id, index)
    })
    const sortedStages = stages.concat(virtualStages).sort((left, right) => {
      const leftOrder = stepOrder.get(left.nodeId) ?? Number.MAX_SAFE_INTEGER
      const rightOrder = stepOrder.get(right.nodeId) ?? Number.MAX_SAFE_INTEGER
      if (leftOrder !== rightOrder) return leftOrder - rightOrder
      return 0
    })
    return applyProcessStageDependencyGuard(sortedStages)
  })

  const stageLogStatuses = new Set([
    'running',
    'paused',
    'success',
    'failed',
    'failure',
    'aborted',
    'unstable',
    'skipped',
    'finished',
    'not_built',
    'notbuilt'
  ])

  const isStageLogAvailable = (stage: DevopsPipelineRunStage) =>
    stageLogStatuses.has(normalizeRunStatus(stage.status)) &&
    normalizeRunStatus(stage.status) !== 'queued' &&
    !isVirtualRunStage(stage)

  const firstAvailableStage = () => displayRunStages.value.find((item) => isStageLogAvailable(item))

  const resetLogCache = () => {
    Object.keys(logContentCache).forEach((key) => {
      delete logContentCache[key]
    })
  }

  const resetInputParamForm = () => {
    Object.keys(inputParamForm).forEach((key) => {
      delete inputParamForm[key]
    })
    Object.keys(inputProceedLoadingMap).forEach((key) => {
      delete inputProceedLoadingMap[key]
    })
    Object.keys(inputAbortLoadingMap).forEach((key) => {
      delete inputAbortLoadingMap[key]
    })
  }

  const loadProcessPipelineSteps = async (row: DevopsPipelineRun) => {
    if (currentRunPipeline.value?.id === row.pipelineId && currentRunPipeline.value.steps?.length) {
      processPipelineSteps.value = normalizeWorkflowBranches(
        currentRunPipeline.value.steps.filter((item) => item.enabled !== false)
      )
      return
    }
    try {
      const detail = await getDevopsPipelineApi(row.pipelineId)
      currentRunPipeline.value = detail
      processPipelineSteps.value = normalizeWorkflowBranches(
        (detail.steps || []).filter((item) => item.enabled !== false)
      )
    } catch {
      processPipelineSteps.value = []
    }
  }

  const currentLogCacheKey = () => {
    const stage = displayRunStages.value.find((item) => item.id === currentStageId.value)
    if (activeLogTab.value === 'stage' && currentStageId.value && !isVirtualRunStage(stage)) {
      const container =
        tektonLogContainerName.value ||
        currentSelectedTektonTaskRun.value?.containerName ||
        stage?.containerName ||
        ''
      if (!isTektonRunContext.value) {
        return `stage:${currentStageId.value || '-'}:${container || '-'}`
      }
      const taskRun = currentTektonLogTaskRunName.value
      return `stage:${currentStageId.value || '-'}:${taskRun || '-'}:${container || '-'}`
    }
    return 'full'
  }

  const applyCurrentLogCache = () => {
    runLog.value = logContentCache[currentLogCacheKey()] || ''
  }

  const closeLogStream = () => {
    if (logEventSource) {
      logEventSource.close()
      logEventSource = null
    }
  }

  const syncCurrentStageSelection = () => {
    if (activeLogTab.value !== 'stage') return
    const selected = displayRunStages.value.find((item) => item.id === currentStageId.value)
    if (selected && isStageLogAvailable(selected)) {
      syncTektonLogTaskRunSelection()
      return
    }
    currentStageId.value = firstAvailableStage()?.id || ''
    syncTektonLogTaskRunSelection()
  }

  const enterRunProcess = async (
    row: DevopsPipelineRun,
    tab: 'stage' | 'full' | 'input' | 'artifact' | 'quality' = 'stage'
  ) => {
    const openingRunId = row.id
    processOpeningRunId.value = openingRunId
    closeLogStream()
    stopStageRuntimeTimer()
    currentRun.value = row
    activeLogTab.value = tab
    currentStageId.value = ''
    tektonLogTaskRunName.value = ''
    tektonLogContainerName.value = ''
    runLog.value = ''
    runArtifacts.value = []
    qualitySummary.value = null
    runInputs.value = []
    resetInputParamForm()
    resetLogCache()
    try {
      await loadProcessPipelineSteps(row)
      await fetchRunStages()
      if (tab === 'artifact' || !isRunActive(row.status)) {
        await fetchRunArtifacts()
      }
      if (tab === 'quality') {
        await fetchQualitySummary()
      }
      if (tab === 'input' || hasInputWaitState.value || Number(row.jenkinsBuildNumber || 0) > 0) {
        await fetchRunInputs()
      }
      if (hasInputWaitState.value) {
        openPendingInputDialog(runInputs.value.length > 0)
      }
      syncCurrentStageSelection()
      processGraphInstance.value = undefined
      processGraphNeedInitialCenter.value = true
      editorMode.value = 'process'
      await nextTick()
      await syncProcessGraphHeight()
      await centerProcessGraphIfNeeded()
      applyCurrentLogCache()
      startLogStream()
      startStageRuntimeTimer()
      ensureProcessStatePolling()
    } finally {
      if (processOpeningRunId.value === openingRunId) processOpeningRunId.value = ''
    }
  }

  const fetchRunStages = async () => {
    if (!currentRun.value) return
    const stageRes = await listDevopsPipelineRunStageApi(currentRun.value.id)
    runStages.value = stageRes.items || []
    syncCurrentStageSelection()
    await syncProcessGraphHeight()
  }

  const stopProcessStatePolling = () => {
    if (processStatePollTimer.value) {
      window.clearInterval(processStatePollTimer.value)
      processStatePollTimer.value = undefined
    }
  }

  const shouldPollProcessState = () => {
    if (!currentRun.value || !isProcessMode.value) return false
    const runStatus = normalizeRunStatus(currentRun.value.status)
    if (!['queued', 'preparing', 'running', 'paused'].includes(runStatus)) return false
    if (isTektonRunContext.value) {
      if (!currentRun.value.tektonPipelineRunName) return true
    } else if (!currentRun.value.jenkinsBuildNumber) return true
    if (runStatus === 'queued' || runStatus === 'preparing') return true
    if (runStatus === 'paused') return runInputs.value.length === 0
    if (displayRunStages.value.length === 0) return true
    if (runStatus === 'running' && !hasActiveStage.value) return true
    return false
  }

  const syncProcessState = async (silent = false) => {
    if (!currentRun.value) return
    if (!silent) processStateLoading.value = true
    try {
      await refreshCurrentRunState(silent)
      if (currentRun.value && !logEventSource) {
        startLogStream(true)
      }
    } finally {
      processStateLoading.value = false
    }
  }

  const ensureProcessStatePolling = () => {
    stopProcessStatePolling()
    if (!shouldPollProcessState()) return
    processStatePollTimer.value = window.setInterval(() => {
      if (!shouldPollProcessState()) {
        stopProcessStatePolling()
        return
      }
      void syncProcessState(true)
    }, 1500)
  }

  const fetchRunArtifacts = async () => {
    if (!currentRun.value) return
    artifactLoading.value = true
    try {
      const res = await listDevopsPipelineRunArtifactApi(currentRun.value.id)
      runArtifacts.value = res.items || []
    } finally {
      artifactLoading.value = false
    }
  }

  const fetchQualitySummary = async () => {
    if (!currentRun.value) return
    qualityLoading.value = true
    try {
      qualitySummary.value = await getQualityRunSummaryApi(currentRun.value.id)
    } finally {
      qualityLoading.value = false
    }
  }

  const inputDefaultValue = (param: DevopsPipelineRunInput['params'][number]) => {
    if (param.defaultValue) return param.defaultValue
    if (param.options?.length) return param.options[0].value
    if (param.type === 'boolean') return 'false'
    return ''
  }

  const ensureInputForm = (inputId: string) => {
    if (!inputParamForm[inputId]) inputParamForm[inputId] = {}
    return inputParamForm[inputId]
  }

  const openPendingInputDialog = (notify = true) => {
    const shouldNotify = activeLogTab.value !== 'input' || !logDrawerVisible.value
    focusPendingInputStage()
    activeLogTab.value = 'input'
    logDrawerVisible.value = true
    if (notify && shouldNotify) ElMessage.warning('流水线等待人工确认，请处理确认框')
  }

  const applyRunInputs = (items: DevopsPipelineRunInput[], notify = true) => {
    const nextInputIds = new Set(items.map((item) => item.id))
    Object.keys(inputParamForm).forEach((key) => {
      if (!nextInputIds.has(key)) delete inputParamForm[key]
    })
    items.forEach((item) => {
      const form = ensureInputForm(item.id)
      ;(item.params || []).forEach((param) => {
        form[param.name] = form[param.name] ?? inputDefaultValue(param)
      })
    })
    runInputs.value = items
    if (items.length > 0) {
      openPendingInputDialog(notify)
    }
  }

  const fetchRunInputs = async (options: { silent?: boolean; minIntervalMs?: number } = {}) => {
    if (!currentRun.value || (!isRunActive(currentRun.value.status) && !hasInputWaitState.value)) {
      runInputs.value = []
      resetInputParamForm()
      inputLoading.value = false
      return
    }
    const runId = currentRun.value.id
    const now = Date.now()
    if (
      options.minIntervalMs &&
      lastInputFetchRunId === runId &&
      now - lastInputFetchAt < options.minIntervalMs
    ) {
      return
    }
    lastInputFetchRunId = runId
    lastInputFetchAt = now
    if (!options.silent) inputLoading.value = true
    try {
      const res = await listDevopsPipelineRunInputApi(currentRun.value.id)
      applyRunInputs(res.items || [], !options.silent)
      if ((res.items || []).length === 0 && hasInputWaitState.value) {
        openPendingInputDialog(false)
      }
    } finally {
      if (!options.silent) inputLoading.value = false
    }
  }

  const openArtifact = (item: DevopsPipelineRunArtifact) => {
    if (item.url) window.open(item.url, '_blank', 'noopener,noreferrer')
  }

  const openQualityReport = async (id: string) => {
    if (!id) return
    const blob = await downloadQualityScanReportApi(id)
    downloadBlob(blob, `quality-report-${id}`)
  }

  const previewQualityReport = async (id: string) => {
    if (!id) return
    clearQualityPreview()
    qualityPreviewVisible.value = true
    qualityPreviewLoading.value = true
    qualityPreviewTitle.value = `报告预览 - ${id}`
    try {
      const blob = await previewQualityScanReportApi(id)
      qualityPreviewUrl.value = URL.createObjectURL(blob)
    } finally {
      qualityPreviewLoading.value = false
    }
  }

  const clearQualityPreview = () => {
    if (qualityPreviewUrl.value) {
      URL.revokeObjectURL(qualityPreviewUrl.value)
      qualityPreviewUrl.value = ''
    }
  }

  const downloadBlob = (blob: Blob, fileName: string) => {
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = fileName || 'download'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
  }

  const qualityGateText = (status?: string) =>
    ({ passed: '通过', failed: '失败', pending: '待评估', created: '待评估' })[status || ''] ||
    status ||
    '-'

  const gateStatusType = (status?: string) => {
    if (status === 'passed') return 'success'
    if (status === 'failed') return 'danger'
    return 'info'
  }

  const reportStatusType = (status?: string) => {
    if (status === 'accepted' || status === 'success') return 'success'
    if (status === 'rejected' || status === 'failed') return 'danger'
    return 'info'
  }

  const qualitySeverityType = (severity?: string) => {
    const value = String(severity || '').toLowerCase()
    if (value === 'critical' || value === 'high') return 'danger'
    if (value === 'medium') return 'warning'
    return 'info'
  }

  const processStatusHint = computed(() => {
    if (!currentRun.value) return null
    const isTekton = isTektonRunContext.value
    const runResourceName = isTekton
      ? currentRun.value.tektonPipelineRunName || currentRun.value.tektonPipelineName || ''
      : currentRun.value.jenkinsBuildNumber
    if (processStateLoading.value) {
      return {
        type: 'info' as const,
        title: '正在同步执行状态',
        description: isTekton
          ? '正在获取 PipelineRun、TaskRun 和阶段信息。'
          : '正在获取构建号、Jenkins 排队状态和阶段信息。'
      }
    }
    if (isTekton && !runResourceName) {
      return {
        type: 'warning' as const,
        title: '等待创建 PipelineRun',
        description: '任务已提交，平台正在同步 Tekton PipelineRun。'
      }
    }
    if (!isTekton && !currentRun.value.jenkinsBuildNumber) {
      if (currentRun.value.jenkinsQueueId) {
        return {
          type: 'warning' as const,
          title: '任务排队中',
          description: `已进入 Jenkins 队列，队列号 ${currentRun.value.jenkinsQueueId}，等待分配构建号。`
        }
      }
      return {
        type: 'warning' as const,
        title: '等待分配构建号',
        description: '任务已提交，Jenkins 还未返回构建号，请稍候。'
      }
    }
    if (normalizeRunStatus(currentRun.value.status) === 'queued') {
      return {
        type: 'warning' as const,
        title: '等待执行',
        description: isTekton
          ? `PipelineRun ${runResourceName} 已创建，当前仍在等待执行。`
          : `构建号 #${currentRun.value.jenkinsBuildNumber} 已创建，当前仍在排队等待执行。`
      }
    }
    if (normalizeRunStatus(currentRun.value.status) === 'preparing') {
      return {
        type: 'info' as const,
        title: '准备执行中',
        description: isTekton
          ? `PipelineRun ${runResourceName} 已创建，正在准备 TaskRun 和 Workspace。`
          : `构建号 #${currentRun.value.jenkinsBuildNumber} 已创建，正在准备工作区和步骤上下文。`
      }
    }
    if (normalizeRunStatus(currentRun.value.status) === 'running' && !hasActiveStage.value) {
      return {
        type: 'info' as const,
        title: '执行已开始',
        description: isTekton
          ? `PipelineRun ${runResourceName} 已启动，正在等待 TaskRun 状态返回。`
          : `构建号 #${currentRun.value.jenkinsBuildNumber} 已启动，正在等待阶段信息返回。`
      }
    }
    return null
  })

  const startLogStream = (force = false) => {
    closeLogStream()
    if (!currentRun.value) return
    if (activeLogTab.value === 'artifact' || activeLogTab.value === 'quality') {
      logLoading.value = false
      return
    }
    if (activeLogTab.value === 'input') {
      logLoading.value = false
    }
    if (
      !force &&
      !isRunActive(currentRun.value.status) &&
      logContentCache[currentLogCacheKey()]
    ) {
      applyCurrentLogCache()
      logLoading.value = false
      return
    }
    syncTektonLogTaskRunSelection()
    const requestSeq = ++logRequestSeq
    const cacheKey = currentLogCacheKey()
    const selectedStage = displayRunStages.value.find((item) => item.id === currentStageId.value)
    const selectedTaskRun = currentSelectedTektonTaskRun.value
    const params =
      activeLogTab.value === 'stage' && currentStageId.value && !isVirtualRunStage(selectedStage)
        ? {
            stageId: currentStageId.value,
            full: false,
            taskRunName: isTektonRunContext.value
              ? selectedTaskRun?.name || selectedStage?.tektonTaskRunName || ''
              : '',
            podName: isTektonRunContext.value
              ? selectedTaskRun?.podName || selectedStage?.tektonPodName || ''
              : '',
            containerName: isTektonRunContext.value
              ? tektonLogContainerName.value ||
                selectedTaskRun?.containerName ||
                selectedStage?.containerName ||
                ''
              : ''
          }
        : { full: true }

    applyCurrentLogCache()
    logLoading.value = activeLogTab.value === 'input' ? false : !runLog.value

    const source = createDevopsPipelineRunLogStream(currentRun.value.id, params)
    logEventSource = source

    source.addEventListener('stages', (event) => {
      if (requestSeq !== logRequestSeq) return
      const previousStageId = currentStageId.value
      const payload = JSON.parse((event as MessageEvent).data) as {
        items?: DevopsPipelineRunStage[]
      }
      runStages.value = payload.items || []
      syncCurrentStageSelection()
      if (runInputs.value.length > 0) focusPendingInputStage()
      scheduleProcessGraphSync()
      if (currentRun.value && isRunActive(currentRun.value.status)) {
        currentRun.value.status = activeRunStatusFromStages()
      }
      if (hasInputWaitState.value) {
        openPendingInputDialog(false)
        void fetchRunInputs({ silent: true, minIntervalMs: 5000 })
      }
      if (
        activeLogTab.value === 'stage' &&
        currentStageId.value &&
        currentStageId.value !== previousStageId
      ) {
        startLogStream(true)
      }
      ensureProcessStatePolling()
    })

    source.addEventListener('log', async (event) => {
      if (requestSeq !== logRequestSeq || cacheKey !== currentLogCacheKey()) return
      if (activeLogTab.value === 'input') {
        logLoading.value = false
        return
      }
      const payload = JSON.parse((event as MessageEvent).data) as {
        content?: string
        changed?: boolean
      }
      if (payload.changed === false) {
        logLoading.value = false
        return
      }
      logContentCache[cacheKey] = payload.content || ''
      runLog.value = logContentCache[cacheKey]
      logLoading.value = false
      await nextTick()
      scrollLogToBottom()
    })

    source.addEventListener('input', (event) => {
      if (requestSeq !== logRequestSeq) return
      try {
        const payload = JSON.parse((event as MessageEvent).data) as {
          items?: DevopsPipelineRunInput[]
        }
        applyRunInputs(payload.items || [])
      } catch {
        void fetchRunInputs()
      }
    })

    source.addEventListener('log-error', (event) => {
      if (requestSeq !== logRequestSeq) return
      const message = (event as MessageEvent).data
      if (message) {
        try {
          const payload = JSON.parse(message) as { message?: string }
          if (payload.message && !logContentCache[cacheKey]) {
            logContentCache[cacheKey] = payload.message
          }
        } catch {
          if (!logContentCache[cacheKey]) logContentCache[cacheKey] = String(message)
        }
      }
      applyCurrentLogCache()
      logLoading.value = false
    })

    source.addEventListener('end', () => {
      if (requestSeq !== logRequestSeq) return
      logLoading.value = false
      nextTick(scrollLogToBottom)
      closeLogStream()
      void (async () => {
        try {
          if (currentRun.value) {
            await fetchRunStages()
            const finalStatus = finalRunStatusFromStages(displayRunStages.value)
            if (finalStatus && currentRun.value) currentRun.value.status = finalStatus
            scheduleProcessGraphSync()
          }
        } finally {
          ensureProcessStatePolling()
          await refreshLogRelatedLists()
        }
      })()
    })

    source.addEventListener('reconnect', () => {
      if (requestSeq !== logRequestSeq) return
      closeLogStream()
      if (logDrawerVisible.value || isProcessMode.value) {
        window.setTimeout(() => {
          if (requestSeq === logRequestSeq && (logDrawerVisible.value || isProcessMode.value)) {
            startLogStream()
          }
        }, 300)
      }
    })

    source.onerror = () => {
      if (requestSeq === logRequestSeq) {
        logLoading.value = false
      }
    }
  }

  const refreshCurrentLog = async () => {
    if (activeLogTab.value === 'artifact') {
      await fetchRunArtifacts()
      return
    }
    if (activeLogTab.value === 'quality') {
      await fetchQualitySummary()
      return
    }
    if (activeLogTab.value === 'input') {
      await Promise.all([fetchRunStages(), fetchRunInputs()])
      return
    }
    await fetchRunStages()
    applyCurrentLogCache()
    startLogStream(true)
  }

  const handleLogTabChange = async () => {
    syncCurrentStageSelection()
    applyCurrentLogCache()
    await nextTick()
    scrollLogToBottom()
    if (activeLogTab.value === 'artifact' && runArtifacts.value.length === 0) {
      await fetchRunArtifacts()
    }
    if (activeLogTab.value === 'quality') {
      await fetchQualitySummary()
      startLogStream()
      return
    }
    if (activeLogTab.value === 'input') {
      await fetchRunInputs()
      startLogStream()
      return
    }
    startLogStream()
  }

  const selectStage = async (stage: DevopsPipelineRunStage) => {
    if (isStageWaitingInput(stage)) {
      currentStageId.value = stage.id
      activeLogTab.value = 'input'
      logDrawerVisible.value = true
      await fetchRunInputs()
      await nextTick()
      return
    }
    if (isVirtualRunStage(stage) && !isRunActive(currentRun.value?.status)) {
      activeLogTab.value = 'full'
      logDrawerVisible.value = true
      applyCurrentLogCache()
      await nextTick()
      scrollLogToBottom()
      startLogStream()
      return
    }
    if (!isStageLogAvailable(stage)) {
      ElMessage.warning('阶段暂未执行')
      return
    }
    currentStageId.value = stage.id
    tektonLogTaskRunName.value =
      stage.tektonTaskRuns?.find((item) => item.name === stage.tektonTaskRunName)?.name ||
      stage.tektonTaskRuns?.[0]?.name ||
      ''
    tektonLogContainerName.value = ''
    activeLogTab.value = 'stage'
    logDrawerVisible.value = true
    applyCurrentLogCache()
    await nextTick()
    scrollLogToBottom()
    startLogStream()
  }

  const openProcessDetail = async (tab: 'stage' | 'full' | 'input' | 'artifact' | 'quality') => {
    activeLogTab.value = tab
    logDrawerVisible.value = true
    if (tab === 'stage') {
      syncCurrentStageSelection()
    }
    await handleLogTabChange()
  }

  const scrollLogToBottom = () => {
    if (!autoScrollLog.value) return
    if (
      activeLogTab.value === 'input' ||
      activeLogTab.value === 'artifact' ||
      activeLogTab.value === 'quality'
    ) {
      return
    }
    const logBody = activeLogTab.value === 'stage' ? stageLogBodyRef.value : fullLogBodyRef.value
    if (!logBody) return
    logBody.scrollTop = logBody.scrollHeight
  }

  const normalizeRunStatus = (status?: string) => {
    const value = (status || '').toLowerCase()
    if (value === 'failure') return 'failed'
    if (value === 'not_built' || value === 'notbuilt') return 'skipped'
    if (!value) return 'queued'
    return value
  }

  const hasPendingInput = computed(() => runInputs.value.length > 0)

  const hasInputWaitState = computed(
    () =>
      hasPendingInput.value ||
      normalizeRunStatus(currentRun.value?.status) === 'paused' ||
      displayRunStages.value.some((stage) => normalizeRunStatus(stage.status) === 'paused')
  )

  const pendingInputStageCandidate = () => {
    const pausedStage = displayRunStages.value.find(
      (stage) => normalizeRunStatus(stage.status) === 'paused'
    )
    if (pausedStage) return pausedStage
    const activeStage = displayRunStages.value.find((stage) => isProcessActiveStatus(stage.status))
    if (activeStage) return activeStage
    return (
      displayRunStages.value.find(
        (stage) =>
          !['success', 'failed', 'aborted', 'unstable', 'skipped', 'finished'].includes(
            normalizeRunStatus(stage.status)
          )
      ) || displayRunStages.value[0]
    )
  }

  const focusPendingInputStage = () => {
    const stage = pendingInputStageCandidate()
    if (stage) currentStageId.value = stage.id
  }

  const pendingInputStageIds = computed(() => {
    const result = new Set<string>()
    if (!hasInputWaitState.value) return result
    const stage = pendingInputStageCandidate()
    if (stage) result.add(stage.id)
    return result
  })

  const isStageWaitingInput = (stage: DevopsPipelineRunStage) =>
    pendingInputStageIds.value.has(stage.id)

  const isRunActive = (status?: string) =>
    ['preparing', 'running', 'queued', 'paused'].includes(normalizeRunStatus(status))

  const isProcessActiveStatus = (status?: string) =>
    ['preparing', 'running', 'paused'].includes(normalizeRunStatus(status))

  const isRunningStageStatus = (status?: string) =>
    ['running', 'paused', 'preparing'].includes(normalizeRunStatus(status))

  const stageStatusSymbol = (status?: string) => {
    const value = normalizeRunStatus(status)
    const map: Record<string, string> = {
      success: 'OK',
      failed: 'FAIL',
      unstable: 'WARN',
      running: 'RUN',
      queued: 'WAIT',
      paused: 'WAIT',
      aborted: 'STOP',
      skipped: 'SKIP',
      finished: 'DONE'
    }
    return map[value] || 'WAIT'
  }

  const processStageNodeTitle = (data: ProcessStageNodeData) => {
    if (data.needsInput) return `${data.stage.stageName}，点击处理确认操作`
    if (isVirtualRunStage(data.stage)) return `${data.stage.stageName}，阶段未产生独立日志`
    if (!isStageLogAvailable(data.stage)) {
      return `${data.stage.stageName}，阶段暂未执行`
    }
    return `${data.stage.stageName}，点击查看详情`
  }

  const currentSelectedStage = computed(() =>
    displayRunStages.value.find((item) => item.id === currentStageId.value)
  )

  const tektonLogTaskRunOptions = computed(() => currentSelectedStage.value?.tektonTaskRuns || [])

  const currentSelectedTektonTaskRun = computed(() => {
    const items = tektonLogTaskRunOptions.value
    if (items.length === 0) return null
    const selected = items.find((item) => item.name === tektonLogTaskRunName.value)
    return (
      selected ||
      items.find((item) => item.name === currentSelectedStage.value?.tektonTaskRunName) ||
      items[0]
    )
  })

  const currentTektonLogTaskRunName = computed(
    () =>
      currentSelectedTektonTaskRun.value?.name ||
      currentSelectedStage.value?.tektonTaskRunName ||
      ''
  )

  const currentTektonLogPodName = computed(
    () =>
      currentSelectedTektonTaskRun.value?.podName || currentSelectedStage.value?.tektonPodName || ''
  )

  const tektonLogContainerOptions = computed(() => {
    const stage = currentSelectedStage.value
    const taskRun = currentSelectedTektonTaskRun.value
    const names = [
      ...(taskRun?.containerNames || []),
      taskRun?.containerName || '',
      ...(stage?.containerNames || []),
      stage?.containerName || ''
    ]
    return names.filter((item, index) => item && names.indexOf(item) === index)
  })

  const tektonTaskRunOptionLabel = (item: TektonTaskRunSnapshot) => {
    const status = runStatusText(item.status)
    return `${item.name}${status ? ` / ${status}` : ''}`
  }

  const tektonTaskRunSummary = (stage: DevopsPipelineRunStage) => {
    const items = stage.tektonTaskRuns || []
    const success = items.filter((item) => normalizeRunStatus(item.status) === 'success').length
    const failed = items.filter((item) => normalizeRunStatus(item.status) === 'failed').length
    if (failed > 0) return `${failed} 失败 / ${items.length}`
    return `${success}/${items.length}`
  }

  const syncTektonLogTaskRunSelection = () => {
    const stage = currentSelectedStage.value
    const items = stage?.tektonTaskRuns || []
    if (items.length === 0) {
      tektonLogTaskRunName.value = ''
      return
    }
    if (items.some((item) => item.name === tektonLogTaskRunName.value)) return
    tektonLogTaskRunName.value =
      items.find((item) => item.name === stage?.tektonTaskRunName)?.name || items[0].name
  }

  const handleTektonTaskRunChange = () => {
    tektonLogContainerName.value = ''
    startLogStream(true)
  }

  const isTektonStageLogSelected = computed(
    () =>
      isTektonRunContext.value &&
      activeLogTab.value === 'stage' &&
      Boolean(currentSelectedStage.value)
  )

  const hasActiveStage = computed(() =>
    displayRunStages.value.some((stage) => isProcessActiveStatus(stage.status))
  )

  const isCurrentLogExecuting = computed(() => {
    if (
      activeLogTab.value === 'artifact' ||
      activeLogTab.value === 'input' ||
      activeLogTab.value === 'quality'
    ) {
      return false
    }
    if (activeLogTab.value === 'stage') {
      if (currentSelectedStage.value) {
        return isProcessActiveStatus(currentSelectedStage.value.status)
      }
      return hasActiveStage.value || isRunActive(currentRun.value?.status)
    }
    return hasActiveStage.value || isRunActive(currentRun.value?.status)
  })

  const logExecutingText = computed(() => {
    if (activeLogTab.value === 'stage') {
      const stage = currentSelectedStage.value
      if (!stage) return '阶段排队中，等待调度'
      if (normalizeRunStatus(stage.status) === 'paused') return '阶段等待人工确认'
      if (normalizeRunStatus(stage.status) === 'queued') return '阶段暂未执行'
      return '阶段正在执行，等待新日志'
    }
    if (normalizeRunStatus(currentRun.value?.status) === 'paused') return '流水线等待人工确认'
    return '流水线正在执行，等待新日志'
  })

  const finalRunStatusFromStages = (stages: DevopsPipelineRunStage[]) => {
    if (stages.length === 0) return ''
    const statuses = stages.map((stage) => normalizeRunStatus(stage.status))
    if (statuses.some((status) => ['preparing', 'running', 'queued', 'paused'].includes(status))) {
      return ''
    }
    if (statuses.some((status) => status === 'failed')) return 'failed'
    if (statuses.some((status) => status === 'aborted')) return 'aborted'
    if (statuses.some((status) => status === 'unstable')) return 'unstable'
    if (statuses.every((status) => ['success', 'skipped', 'finished'].includes(status))) {
      return 'success'
    }
    return ''
  }

  const activeRunStatusFromStages = () => {
    const statuses = displayRunStages.value.map((stage) => normalizeRunStatus(stage.status))
    if (statuses.some((status) => status === 'paused')) return 'paused'
    if (statuses.some((status) => status === 'running')) return 'running'
    if (statuses.some((status) => status === 'queued')) return 'queued'
    if (statuses.some((status) => status === 'preparing')) return 'preparing'
    return currentRun.value?.status || ''
  }

  const stopLogAutoRefresh = () => {
    closeLogStream()
  }

  const startStageRuntimeTimer = () => {
    stopStageRuntimeTimer()
    stageRuntimeTick.value = Date.now()
    stageRuntimeTimer.value = window.setInterval(() => {
      stageRuntimeTick.value = Date.now()
    }, 1000)
  }

  const stopStageRuntimeTimer = () => {
    if (stageRuntimeTimer.value) {
      window.clearInterval(stageRuntimeTimer.value)
      stageRuntimeTimer.value = undefined
    }
  }

  const handleLogDialogClosed = () => {
    logDrawerVisible.value = false
    const shouldRefresh = Boolean(currentRun.value)
    stopLogAutoRefresh()
    stopStageRuntimeTimer()
    stopProcessStatePolling()
    runLog.value = ''
    runStages.value = []
    processGraphInstance.value = undefined
    disconnectProcessGraphSize()
    processGraphNeedInitialCenter.value = false
    runArtifacts.value = []
    qualitySummary.value = null
    runInputs.value = []
    processPipelineSteps.value = []
    resetInputParamForm()
    currentStageId.value = ''
    currentRun.value = null
    resetLogCache()
    logRequestSeq += 1
    if (shouldRefresh) void refreshLogRelatedLists()
  }

  const copyRunLog = async () => {
    if (!runLog.value) return
    await navigator.clipboard.writeText(runLog.value)
    ElMessage.success('复制成功')
  }

  const downloadRunLog = () => {
    if (!runLog.value) return
    const runNo =
      currentRun.value?.engineType === 'tekton'
        ? currentRun.value?.buildId || currentRun.value?.tektonPipelineRunName
        : currentRun.value?.jenkinsBuildNumber
    const name = runNo
      ? `${currentRun.value.pipelineCode || 'pipeline'}-${runNo}`
      : currentRun.value?.pipelineCode || 'pipeline'
    const blob = new Blob([runLog.value], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${name}-${activeLogTab.value}.log`
    link.click()
    URL.revokeObjectURL(url)
  }

  const refreshCurrentRunState = async (silent = false) => {
    const currentRunId = currentRun.value?.id
    await Promise.allSettled([
      fetchRunStages(),
      fetchRunInputs({ silent, minIntervalMs: silent ? 5000 : 0 }),
      fetchRunArtifacts(),
      refreshLogRelatedLists()
    ])
    const fresh = currentRunId ? runs.value.find((item) => item.id === currentRunId) || null : null
    if (fresh) currentRun.value = fresh
    if (hasInputWaitState.value) {
      openPendingInputDialog(runInputs.value.length > 0)
    }
    ensureProcessStatePolling()
  }

  const refreshGraphWorkbench = async () => {
    if (!currentRun.value) return
    await Promise.allSettled([fetchRunStages(), fetchRunInputs(), fetchRunArtifacts()])
    startLogStream(true)
  }

  const stopRun = async (runId: string) => {
    const targetRun =
      currentRun.value?.id === runId
        ? currentRun.value
        : runs.value.find((item) => item.id === runId)
    if (!targetRun || !isRunActive(targetRun.status)) {
      ElMessage.warning('流水线已结束，不能停止')
      return
    }
    await ElMessageBox.confirm('确认停止当前流水线执行？', '停止确认', { type: 'warning' })
    runActionLoading.value = true
    try {
      await stopDevopsPipelineRunApi(runId)
      ElMessage.success('已提交停止请求')
      await refreshCurrentRunState()
      startLogStream(true)
    } finally {
      runActionLoading.value = false
    }
  }

  const deleteRun = async (row: DevopsPipelineRun) => {
    if (isRunActive(row.status)) {
      ElMessage.warning('流水线未执行完成，不能删除')
      return
    }
    await ElMessageBox.confirm(`确认删除执行记录「${row.pipelineName || row.id}」？`, '删除确认', {
      type: 'warning'
    })
    runActionLoading.value = true
    try {
      await deleteDevopsPipelineRunApi(row.id)
      ElMessage.success('删除成功')
      if (currentRun.value?.id === row.id) {
        exitRunProcess()
      } else {
        await refreshLogRelatedLists()
      }
    } finally {
      runActionLoading.value = false
    }
  }

  const proceedRunInput = async (item: DevopsPipelineRunInput) => {
    if (!currentRun.value) return
    if (inputProceedLoadingMap[item.id] || inputAbortLoadingMap[item.id]) return
    const requiredEmpty = item.params?.find((param) => {
      if (param.type === 'boolean') return false
      return !String(inputParamForm[item.id]?.[param.name] ?? '').trim()
    })
    if (requiredEmpty) {
      ElMessage.warning(`请填写确认参数：${requiredEmpty.name}`)
      return
    }
    inputProceedLoadingMap[item.id] = true
    try {
      await proceedDevopsPipelineRunInputApi(currentRun.value.id, item.id, {
        params: JSON.stringify(inputParamForm[item.id] || {})
      })
      ElMessage.success('已继续执行')
      await refreshCurrentRunState()
      activeLogTab.value = 'stage'
      startLogStream(true)
    } finally {
      inputProceedLoadingMap[item.id] = false
    }
  }

  const abortRunInput = async (item: DevopsPipelineRunInput) => {
    if (!currentRun.value) return
    if (inputProceedLoadingMap[item.id] || inputAbortLoadingMap[item.id]) return
    await ElMessageBox.confirm('确认终止当前人工确认？', '终止确认', { type: 'warning' })
    inputAbortLoadingMap[item.id] = true
    try {
      await abortDevopsPipelineRunInputApi(currentRun.value.id, item.id)
      ElMessage.success('已提交终止请求')
      await refreshCurrentRunState()
      startLogStream(true)
    } finally {
      inputAbortLoadingMap[item.id] = false
    }
  }

  const addTool = () => {
    formData.tools.push({ type: 'maven', name: '', version: '' } as JenkinsTool)
  }
  const removeTool = (index: number) => {
    formData.tools.splice(index, 1)
  }
  const optionEnabled = (name: string) => formData.options.some((item) => item.name === name)
  const optionValue = (name: string) =>
    formData.options.find((item) => item.name === name)?.value || ''
  const setOptionEnabled = (name: string, enabled: boolean, defaultValue: string) => {
    const index = formData.options.findIndex((item) => item.name === name)
    if (!enabled) {
      if (index >= 0) formData.options.splice(index, 1)
      return
    }
    if (index < 0) {
      formData.options.push({ name, value: defaultValue } as JenkinsOption)
    }
  }
  const setOptionValue = (name: string, value: string) => {
    const item = formData.options.find((option) => option.name === name)
    if (item) item.value = value
  }
  const resetOptions = () => {
    formData.options = defaultJenkinsOptions()
  }
  const resolveTriggerMode = (
    row: Pick<DevopsPipeline, 'triggerMode' | 'triggers'> & {
      engineType?: string
      tektonDagConfig?: string
    }
  ) => {
    if (row.triggerMode === 'manual' || row.triggerMode === 'scheduled') return row.triggerMode
    if (row.engineType === 'tekton') {
      const config = parseJsonRecord(row.tektonDagConfig)
      const scheduleConfig =
        config.scheduleConfig && typeof config.scheduleConfig === 'object'
          ? (config.scheduleConfig as Record<string, any>)
          : {}
      return scheduleConfig.enabled ? 'scheduled' : 'manual'
    }
    return (row.triggers || []).some((item) => item.enabled && item.spec) ? 'scheduled' : 'manual'
  }
  const triggerModeText = (mode?: string) => (mode === 'scheduled' ? '定时' : '手动')
  const triggerModeType = (mode?: string) => (mode === 'scheduled' ? 'warning' : 'info')
  const syncTriggerModeFromTriggers = () => {
    formData.triggerMode = formData.triggers.some((item) => item.enabled && item.spec)
      ? 'scheduled'
      : 'manual'
  }
  const triggerDescription = (type: string) =>
    triggerDefinitions.find((item) => item.type === type)?.description ||
    '使用 Jenkins 原生触发配置。'
  const addTrigger = () => {
    formData.triggers.push({ type: 'cron', spec: 'H 2 * * *', enabled: true } as JenkinsTrigger)
    syncTriggerModeFromTriggers()
  }
  const removeTrigger = (index: number) => {
    formData.triggers.splice(index, 1)
    syncTriggerModeFromTriggers()
  }

  const handleSearch = () => {
    persistPipelineSystemId(searchForm.value.systemId)
    pagination.current = 1
    fetchPipelines()
  }
  const handleReset = () => {
    const systemId = resolvePipelineSearchSystemId()
    searchForm.value = {
      name: undefined,
      systemId: systemId || undefined,
      environmentId: undefined
    }
    persistPipelineSystemId(systemId)
    handleSearch()
  }
  const handleSizeChange = (size: number) => {
    pagination.size = size
    pagination.current = 1
    fetchPipelines()
  }
  const handleCurrentChange = (page: number) => {
    pagination.current = page
    fetchPipelines()
  }

  const syncStatusText = (status?: string) => {
    const map: Record<string, string> = { synced: '已同步', pending: '待同步', failed: '同步失败' }
    return map[status || ''] || '未知'
  }
  const syncStatusType = (status?: string) => {
    if (status === 'synced') return 'success'
    if (status === 'failed') return 'danger'
    return 'info'
  }
  const runStatusText = (status?: string) => {
    const map: Record<string, string> = {
      never: '未运行',
      preparing: '准备中',
      queued: '排队中',
      running: '运行中',
      success: '成功',
      failure: '失败',
      failed: '失败',
      unstable: '不稳定',
      skipped: '已跳过',
      paused: '已暂停',
      finished: '已结束',
      aborted: '已停止'
    }
    return map[status || ''] || status || '未知'
  }
  const runStatusType = (status?: string) => {
    if (status === 'success') return 'success'
    if (status === 'failed' || status === 'failure') return 'danger'
    if (status === 'unstable') return 'warning'
    if (status === 'running' || status === 'queued' || status === 'preparing') return 'warning'
    if (
      status === 'aborted' ||
      status === 'skipped' ||
      status === 'paused' ||
      status === 'finished'
    ) {
      return 'info'
    }
    return 'info'
  }
  const formatTime = (value?: number) => {
    if (!value) return '-'
    return new Date(value * 1000).toLocaleString()
  }
  const formatRunTime = (value?: string) => {
    if (!value) return '-'
    if (/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(value)) return value
    return smartFormatTime(value)
  }
  const formatDuration = (seconds?: number, allowZero = false) => {
    const value = Number(seconds ?? 0)
    if (value < 0) return '-'
    if (value === 0) return allowZero ? '0s' : '-'
    if (value < 60) return `${value}s`
    const min = Math.floor(value / 60)
    const sec = value % 60
    return sec > 0 ? `${min}m${sec}s` : `${min}m`
  }
  const parseTimeValue = (value?: string) => {
    if (!value) return 0
    const normalized = value.includes('T') ? value : value.replace(/-/g, '/')
    const time = new Date(normalized).getTime()
    return Number.isNaN(time) ? 0 : time
  }
  const formatStageDuration = (stage: DevopsPipelineRunStage) => {
    const finalStatus = isFinalStageStatus(stage.status)
    if (normalizeRunStatus(stage.status) === 'running') {
      const started = parseTimeValue(stage.startedAt)
      if (started > 0) {
        const seconds = Math.max(1, Math.floor((stageRuntimeTick.value - started) / 1000))
        return formatDuration(seconds)
      }
    }
    if (!stage.durationSeconds) {
      const started = parseTimeValue(stage.startedAt)
      const finished = parseTimeValue(stage.finishedAt)
      if (started > 0 && finished > started) {
        return formatDuration(Math.max(1, Math.floor((finished - started) / 1000)))
      }
    }
    if (typeof stage.durationSeconds === 'number') {
      return formatDuration(stage.durationSeconds, finalStatus)
    }
    return finalStatus ? '0s' : '-'
  }

  const isFinalStageStatus = (status?: string) =>
    ['success', 'failed', 'aborted', 'unstable', 'skipped', 'finished'].includes(
      normalizeRunStatus(status)
    )

  const processGraphLayout = computed<ProcessGraphLayout>(() =>
    buildProcessGraphLayout({
      stages: displayRunStages.value,
      pipelineSteps: processPipelineSteps.value,
      runActive: Boolean(currentRun.value && isRunActive(currentRun.value.status)),
      normalizeRunStatus,
      isRunningStageStatus
    })
  )

  const parallelGroupCount = computed(() => processGraphLayout.value.parallelGroups)

  const processGraphMinHeight = computed(() => Math.max(520, processGraphLayout.value.height + 120))

  const processGraphStyle = computed(() => ({
    width: `${Math.max(processGraphLayout.value.width, processGraphViewportWidth.value, 720)}px`,
    height: `max(${processGraphMinHeight.value}px, var(--process-graph-min-height, 520px))`
  }))

  const syncProcessGraphHeight = async () => {
    await nextTick()
    const body = processGraphScrollRef.value
    if (!body) return
    const height = Math.max(520, body.clientHeight)
    const width = body.clientWidth
    if (processGraphViewportWidth.value !== width) {
      processGraphViewportWidth.value = width
    }
    const minHeight = `${height}px`
    if (body.style.getPropertyValue('--process-graph-min-height') !== minHeight) {
      body.style.setProperty('--process-graph-min-height', minHeight)
    }
  }

  const scheduleProcessGraphSync = () => {
    if (processGraphSyncFrame) return
    processGraphSyncFrame = window.requestAnimationFrame(() => {
      processGraphSyncFrame = 0
      void syncProcessGraphHeight()
    })
  }

  const observeProcessGraphSize = () => {
    const body = processGraphScrollRef.value
    if (!body || typeof ResizeObserver === 'undefined') return
    processGraphResizeObserver?.disconnect()
    processGraphResizeObserver = new ResizeObserver(() => scheduleProcessGraphSync())
    processGraphResizeObserver.observe(body)
  }

  const disconnectProcessGraphSize = () => {
    processGraphResizeObserver?.disconnect()
    processGraphResizeObserver = null
    if (processGraphSyncFrame) {
      window.cancelAnimationFrame(processGraphSyncFrame)
      processGraphSyncFrame = 0
    }
  }

  const computeProcessGraphCenterViewport = (): ViewportTransform => {
    const viewportWidth = Math.max(processGraphViewportWidth.value, 720)
    const graphWidth = processGraphLayout.value.width * processGraphZoom.value
    const graphHeight = processGraphLayout.value.height * processGraphZoom.value
    const viewportHeight = Math.max(processGraphScrollRef.value?.clientHeight || 0, 520)
    return {
      x: (viewportWidth - graphWidth) / 2,
      y: (viewportHeight - graphHeight) / 2,
      zoom: processGraphZoom.value
    }
  }

  const centerProcessGraph = async () => {
    await syncProcessGraphHeight()
    const viewport = computeProcessGraphCenterViewport()
    processGraphInitialViewport.value = viewport
    if (processGraphInstance.value) {
      await processGraphInstance.value.setViewport(viewport, { duration: 0 })
    }
  }

  const centerProcessGraphIfNeeded = async () => {
    if (!processGraphNeedInitialCenter.value || !processGraphInstance.value) return
    await centerProcessGraph()
    processGraphNeedInitialCenter.value = false
  }

  const handleProcessGraphInit = async (instance: VueFlowStore) => {
    processGraphInstance.value = instance
    observeProcessGraphSize()
    await centerProcessGraphIfNeeded()
  }

  const setProcessGraphZoom = async (zoom: number) => {
    processGraphZoom.value = Math.min(1.28, Math.max(0.72, Number(zoom.toFixed(2))))
    const current = processGraphInstance.value?.getViewport()
    if (current && processGraphInstance.value) {
      await processGraphInstance.value.setViewport(
        { x: current.x, y: current.y, zoom: processGraphZoom.value },
        { duration: 0 }
      )
    } else {
      processGraphInitialViewport.value = {
        ...processGraphInitialViewport.value,
        zoom: processGraphZoom.value
      }
    }
  }

  const zoomProcessGraphIn = () => void setProcessGraphZoom(processGraphZoom.value + 0.12)
  const zoomProcessGraphOut = () => void setProcessGraphZoom(processGraphZoom.value - 0.12)
  const resetProcessGraphView = async () => {
    processGraphZoom.value = 1
    await centerProcessGraph()
  }

  const processGraphNodes = computed<Node<ProcessStageNodeData | ProcessParallelGroupNodeData>[]>(
    () => {
      const groupNodes: Node<ProcessStageNodeData | ProcessParallelGroupNodeData>[] =
        processGraphLayout.value.groups.map((group) => ({
          id: group.id,
          type: 'processGroup',
          position: { x: group.x, y: group.y },
          data: group,
          selectable: false,
          draggable: false,
          zIndex: 0,
          style: {
            width: `${group.width}px`,
            height: `${group.height}px`
          }
        }))
      const stageNodes: Node<ProcessStageNodeData | ProcessParallelGroupNodeData>[] =
        processGraphLayout.value.items.map((item) => {
          const status = normalizeRunStatus(item.stage.status)
          return {
            id: item.stage.id,
            type: 'processStage',
            position: { x: item.x, y: item.y },
            sourcePosition: Position.Right,
            targetPosition: Position.Left,
            data: {
              stage: item.stage,
              status,
              symbol: stageStatusSymbol(item.stage.status),
              duration: formatStageDuration(item.stage),
              parallelGroup: item.parallelGroup,
              needsInput: isStageWaitingInput(item.stage),
              isPost: item.rootType === 'post' || item.stage.stageType === 'post'
            },
            selectable: false,
            draggable: false,
            zIndex: 2
          }
        })
      return [...groupNodes, ...stageNodes]
    }
  )

  const processGraphEdges = computed<Edge[]>(() => {
    const layout = processGraphLayout.value
    const itemByStageId = new Map(layout.items.map((item) => [item.stage.id, item]))
    const statusByStageId = new Map(
      displayRunStages.value.map((stage) => [stage.id, normalizeRunStatus(stage.status)])
    )
    const activeParallelGroups = new Set(
      layout.groups
        .filter((group) => normalizeRunStatus(group.status) === 'running')
        .map((group) => group.name)
    )
    return layout.edges.map((edge) => {
      const sourceStatus = statusByStageId.get(edge.source) || ''
      const targetStatus = statusByStageId.get(edge.target) || ''
      const sourceItem = itemByStageId.get(edge.source)
      const targetItem = itemByStageId.get(edge.target)
      const targetActive = isRunningStageStatus(targetStatus)
      const parallelGroupExitActive =
        Boolean(sourceItem?.parallelGroup) &&
        sourceItem?.parallelGroup !== targetItem?.parallelGroup &&
        activeParallelGroups.has(sourceItem?.parallelGroup || '') &&
        processDependencyReadyStatuses.has(sourceStatus)
      const active =
        targetActive &&
        (processDependencyReadyStatuses.has(sourceStatus) || parallelGroupExitActive)
      const failed = ['failed', 'failure', 'aborted'].includes(targetStatus)
      const success = processDependencyReadyStatuses.has(targetStatus)
      const color = active ? '#2563eb' : failed ? '#ef4444' : success ? '#16a34a' : '#94a3b8'
      const edgeClass = active
        ? 'process-edge-active'
        : failed
          ? 'process-edge-failed'
          : success
            ? 'process-edge-success'
            : 'process-edge-static'
      return {
        id: edge.id,
        source: edge.source,
        target: edge.target,
        sourceHandle: null,
        targetHandle: null,
        animated: false,
        class: edgeClass,
        type: 'default',
        interactionWidth: 18,
        zIndex: active ? 1 : 0,
        ariaLabel: '阶段连接线',
        pathOptions: {
          curvature: 0.18
        },
        style: {
          stroke: color,
          strokeWidth: active ? 2.3 : success || failed ? 2 : 1.65
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color,
          width: active ? 9 : 8,
          height: active ? 9 : 8,
          strokeWidth: 1.2
        }
      }
    })
  })

  const handleProcessNodeClick = (event: {
    node: Node<ProcessStageNodeData | ProcessParallelGroupNodeData>
  }) => {
    if (!('stage' in event.node.data)) return
    void selectStage(event.node.data.stage)
  }

  watch(
    () => projectStore.selectedProjectId,
    () => {
      if (!isEditorMode.value) refreshAll()
      if (formData.projectId !== projectStore.selectedProjectId) {
        formData.projectId = projectStore.selectedProjectId
      }
      pipelineChannelBindings.value = []
      Object.keys(pipelineChannelBindingLoadedGroups).forEach((key) => {
        delete pipelineChannelBindingLoadedGroups[key]
      })
      resetAgentCache()
    }
  )

  watch(
    () => searchForm.value.systemId,
    (systemId) => {
      persistPipelineSystemId(systemId)
    }
  )

  watch(jenkinsConfigVisible, (visible) => {
    if (visible) fetchJenkinsAgents(true)
  })

  watch(currentStageId, () => {
    tektonLogContainerName.value = ''
  })

  watch(
    () => formData.triggers,
    () => syncTriggerModeFromTriggers(),
    { deep: true }
  )

  let activatedOnce = false
  onMounted(refreshAll)
  onBeforeUnmount(() => {
    stopLogAutoRefresh()
    stopStageRuntimeTimer()
    stopProcessStatePolling()
    disconnectProcessGraphSize()
    clearQualityPreview()
  })
  onActivated(() => {
    if (!activatedOnce) {
      activatedOnce = true
      return
    }
    if (isProcessMode.value) {
      observeProcessGraphSize()
      scheduleProcessGraphSync()
      ensureProcessStatePolling()
      return
    }
    if (!isEditorMode.value) refreshAll()
  })
</script>

<style scoped lang="scss">
  .devops-pipeline-instance-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    overflow: hidden;

    &.editor-active {
      height: calc(100vh - 176px);
      min-height: 620px;
    }

    &.process-active {
      height: calc(100vh - 176px);
      min-height: 560px;
    }
  }

  :deep(.art-search-bar .el-form-item__label) {
    white-space: nowrap;
  }

  .pipeline-content {
    display: flex;
    flex: 1;
    min-height: 0;
    margin-top: 12px;
  }

  .pipeline-main-panel {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
    overflow: hidden;
  }

  .pipeline-table-card {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
  }

  .pipeline-table-card :deep(.el-card__body) {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
  }

  .table-actions {
    display: inline-flex;
    flex-wrap: nowrap;
    gap: 8px;
    align-items: center;
    justify-content: flex-start;
    min-width: max-content;
    white-space: nowrap;
  }

  .table-actions :deep(.el-button) {
    display: inline-flex;
    flex: none;
    align-items: center;
    justify-content: center;
    min-width: 44px;
    padding: 0 4px;
    margin-left: 0;
    text-align: center;
    white-space: nowrap;
  }

  .table-actions :deep(.el-dropdown) {
    display: inline-flex;
    flex: 0 0 44px;
    align-items: center;
    justify-content: center;
    width: 44px;
  }

  .table-actions :deep(.el-dropdown .el-button) {
    flex: 0 0 44px;
    width: 44px;
  }

  .run-table-actions {
    width: 100%;
    min-width: 136px;
  }

  .run-table-actions :deep(.el-button:first-child) {
    min-width: 72px;
  }

  .pagination-wrap,
  .run-pagination {
    display: flex;
    justify-content: center;
    padding-bottom: 14px;
    margin-top: 12px;
  }

  .pipeline-editor-workbench {
    display: flex;
    flex: 1;
    width: 100%;
    height: 100%;
    min-height: 620px;
    overflow: hidden;
  }

  .tekton-editor-shell,
  .tekton-detail-shell {
    display: flex;
    flex: 1;
    flex-direction: column;
    width: 100%;
    min-width: 0;
    min-height: 0;
    overflow: hidden;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .tekton-editor-head,
  .tekton-detail-head {
    display: flex;
    flex: none;
    gap: 14px;
    align-items: center;
    justify-content: space-between;
    min-height: 64px;
    padding: 10px 16px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .tekton-editor-title,
  .tekton-detail-title {
    display: flex;
    flex: 1;
    gap: 12px;
    align-items: center;
    min-width: 0;

    div {
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
      color: var(--el-text-color-primary);
      font-size: 16px;
    }

    span {
      margin-top: 4px;
      color: var(--el-text-color-secondary);
      font-size: 12px;
    }
  }

  .tekton-editor-actions,
  .tekton-detail-actions {
    display: flex;
    flex: none;
    flex-wrap: wrap;
    gap: 8px;
    justify-content: flex-end;
  }

  .tekton-editor-shell :deep(.tekton-dag-designer) {
    flex: 1;
    min-height: 0;
    border: 0;
    border-radius: 0;
  }

  .tekton-detail-tabs {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
    padding: 0 14px 14px;

    :deep(.el-tabs__content) {
      flex: 1;
      min-height: 0;
      overflow: hidden;
    }

    :deep(.el-tab-pane) {
      height: 100%;
      min-height: 0;
      overflow: auto;
    }
  }

  .tekton-overview-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 12px;
    padding: 14px 0;
  }

  .tekton-overview-item {
    display: grid;
    gap: 10px;
    min-height: 76px;
    padding: 12px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;

    span {
      color: var(--el-text-color-secondary);
      font-size: 12px;
    }

    strong {
      overflow: hidden;
      color: var(--el-text-color-primary);
      font-size: 20px;
      line-height: 1;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .tekton-detail-section {
    display: grid;
    gap: 12px;
    padding: 14px;
    margin-bottom: 12px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .tekton-section-head {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    min-height: 32px;

    strong {
      color: var(--el-text-color-primary);
      font-size: 15px;
    }
  }

  .tekton-info-grid {
    display: grid;
    grid-template-columns: 132px minmax(0, 1fr);
    gap: 10px 14px;
    align-items: center;

    span {
      color: var(--el-text-color-secondary);
      font-size: 12px;
    }

    strong {
      min-width: 0;
      overflow: hidden;
      color: var(--el-text-color-primary);
      font-size: 13px;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .tekton-detail-dag {
    height: 100%;
    min-height: 560px;
    overflow: hidden;
  }

  .tekton-detail-dag :deep(.tekton-dag-designer) {
    height: 100%;
  }

  .pipeline-records-workbench {
    display: flex;
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }

  .pipeline-process-workbench {
    display: flex;
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }

  .process-graph-panel {
    display: flex;
    flex: 1;
    flex-direction: column;
    width: 100%;
    min-width: 0;
    min-height: 0;
    overflow: hidden;
    background: rgb(255 255 255 / 92%);
    border: 1px solid rgb(205 215 229 / 88%);
    border-radius: 8px;
    box-shadow: 0 16px 36px rgb(35 48 74 / 7%);
  }

  .process-title {
    flex: 1;
    min-width: 0;

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
      margin-top: 3px;
      font-size: 13px;
      color: var(--el-text-color-secondary);
    }
  }

  .process-actions {
    display: inline-flex;
    flex: none;
    flex-wrap: wrap;
    gap: 8px;
    justify-content: flex-end;

    :deep(.el-button) {
      height: 32px;
      padding: 0 14px;
    }
  }

  .process-graph-head {
    display: flex;
    flex: none;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    min-height: 54px;
    padding: 10px 14px;
    color: var(--el-text-color-secondary);
    background: rgb(255 255 255 / 88%);
    border-bottom: 1px solid rgb(220 226 236 / 82%);

    strong {
      font-size: 15px;
      color: var(--el-text-color-primary);
    }
  }

  .process-graph-title {
    display: flex;
    flex: 1;
    gap: 12px;
    align-items: center;
    min-width: 0;
  }

  .process-graph-scroll {
    position: relative;
    display: block;
    flex: 1;
    min-height: 0;
    max-height: 100%;
    padding: 0;
    overflow: auto;
    scrollbar-gutter: stable;
    --process-graph-min-height: 500px;
    -webkit-overflow-scrolling: touch;
  }

  .process-status-banner {
    position: sticky;
    top: 0;
    z-index: 6;
    display: grid;
    gap: 4px;
    padding: 12px 16px;
    margin: 0 0 8px;
    background: rgb(255 255 255 / 92%);
    border-bottom: 1px solid rgb(226 232 240 / 88%);
    backdrop-filter: blur(10px);

    strong {
      font-size: 13px;
      font-weight: 700;
      line-height: 1.2;
    }

    span {
      font-size: 12px;
      line-height: 1.4;
    }

    &.info {
      strong,
      span {
        color: #1d4ed8;
      }
    }

    &.warning {
      strong,
      span {
        color: #b45309;
      }
    }
  }

  .process-status-overlay {
    position: absolute;
    inset: 72px 20px 20px;
    z-index: 4;
    display: flex;
    align-items: flex-start;
    justify-content: center;
    pointer-events: none;
  }

  .process-status-card {
    display: grid;
    gap: 6px;
    width: min(480px, calc(100% - 24px));
    padding: 16px 18px;
    color: #1d4ed8;
    background: rgb(255 255 255 / 94%);
    border: 1px solid rgb(191 219 254 / 92%);
    border-radius: 14px;
    box-shadow: 0 18px 36px rgb(37 99 235 / 10%);
    backdrop-filter: blur(10px);

    strong {
      font-size: 14px;
      font-weight: 700;
      line-height: 1.2;
    }

    span {
      font-size: 12px;
      line-height: 1.45;
    }
  }

  .process-flow {
    min-width: 100%;
    min-height: 100%;
    contain: layout paint style;
    background:
      linear-gradient(90deg, rgb(100 116 139 / 9%) 1px, transparent 1px),
      linear-gradient(180deg, rgb(100 116 139 / 9%) 1px, transparent 1px),
      linear-gradient(90deg, rgb(100 116 139 / 4%) 1px, transparent 1px),
      linear-gradient(180deg, rgb(100 116 139 / 4%) 1px, transparent 1px),
      linear-gradient(180deg, #fbfdff 0%, #f8fbff 100%);
    background-size:
      112px 112px,
      112px 112px,
      28px 28px,
      28px 28px,
      100% 100%;
  }

  .process-flow :deep(.vue-flow__pane) {
    cursor: grab;
  }

  .process-flow :deep(.vue-flow__renderer) {
    cursor: grab;
  }

  .process-flow :deep(.vue-flow__transformationpane) {
    will-change: transform;
  }

  .process-flow :deep(.vue-flow__pane.dragging),
  .process-flow :deep(.vue-flow__renderer.dragging) {
    cursor: grabbing;
  }

  .process-flow :deep(.vue-flow__node) {
    width: 156px;
    height: 92px;
    background: transparent;
    border: 0;
    box-shadow: none;
  }

  .process-flow :deep(.vue-flow__node-processGroup) {
    pointer-events: none;
  }

  .process-parallel-group {
    position: relative;
    width: 100%;
    height: 100%;
    overflow: hidden;
    background:
      linear-gradient(180deg, rgb(255 255 255 / 84%), rgb(239 246 255 / 56%)),
      rgb(248 251 255 / 78%);
    border: 1px solid rgb(96 165 250 / 34%);
    border-radius: 16px;
    box-shadow:
      inset 0 0 0 1px rgb(255 255 255 / 78%),
      0 20px 52px rgb(30 64 175 / 10%);

    &::before {
      position: absolute;
      inset: 10px;
      pointer-events: none;
      content: '';
      border: 1px dashed rgb(59 130 246 / 34%);
      border-radius: 12px;
    }

    span {
      position: absolute;
      top: 9px;
      left: 16px;
      max-width: calc(100% - 28px);
      overflow: hidden;
      font-size: 12px;
      font-weight: 700;
      line-height: 1;
      color: #315d9b;
      text-overflow: ellipsis;
      white-space: nowrap;
      background: rgb(248 251 255 / 92%);
    }

    &.is-running {
      border-color: rgb(245 158 11 / 46%);

      &::before {
        border-color: rgb(245 158 11 / 44%);
      }

      span {
        color: #946200;
      }
    }

    &.is-failed {
      border-color: rgb(239 68 68 / 42%);

      &::before {
        border-color: rgb(239 68 68 / 38%);
      }

      span {
        color: #a52222;
      }
    }

    &.is-post {
      background:
        linear-gradient(180deg, rgb(255 255 255 / 84%), rgb(255 247 237 / 58%)),
        rgb(255 251 247 / 78%);
      border-color: rgb(251 146 60 / 38%);

      &::before {
        border-color: rgb(251 146 60 / 35%);
      }

      span {
        color: #9a4b12;
      }
    }
  }

  .process-flow :deep(.vue-flow__handle) {
    width: 8px;
    height: 8px;
    pointer-events: none;
    background: transparent;
    border: 0;
    opacity: 0;
  }

  .process-flow :deep(.vue-flow__edge-path) {
    stroke-linecap: round;
    stroke-linejoin: round;
    transition:
      stroke 0.18s ease,
      stroke-width 0.18s ease,
      opacity 0.18s ease;
  }

  .process-flow :deep(.process-edge-static .vue-flow__edge-path) {
    opacity: 0.62;
  }

  .process-flow :deep(.process-edge-active .vue-flow__edge-path) {
    filter: drop-shadow(0 0 3px rgb(37 99 235 / 16%));
    opacity: 0.98;
    stroke-dasharray: 7 12;
    animation: process-flow-dash 1.45s linear infinite;
  }

  .process-flow :deep(.process-edge-success .vue-flow__edge-path) {
    filter: drop-shadow(0 0 2px rgb(22 163 74 / 13%));
    opacity: 0.92;
  }

  .process-flow :deep(.process-edge-failed .vue-flow__edge-path) {
    filter: drop-shadow(0 0 3px rgb(239 68 68 / 16%));
    opacity: 0.94;
    stroke-dasharray: 5 6;
  }

  .process-flow :deep(.vue-flow__edges) {
    z-index: 0;
  }

  .process-flow :deep(.vue-flow__nodes) {
    z-index: 2;
  }

  .process-graph-tools {
    position: sticky;
    right: 12px;
    bottom: 8px;
    z-index: 5;
    display: inline-flex;
    gap: 6px;
    float: right;
    align-items: center;
    justify-content: flex-end;
    width: max-content;
    min-height: 28px;
    padding: 2px 4px;
    margin-top: -36px;
    margin-left: auto;
    pointer-events: auto;
    background: rgb(255 255 255 / 92%);
    border: 1px solid rgb(176 194 220 / 55%);
    border-radius: 999px;
    box-shadow: 0 8px 24px rgb(15 23 42 / 10%);
    backdrop-filter: blur(10px);

    button {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      min-width: 26px;
      height: 24px;
      padding: 0 6px;
      font: inherit;
      font-size: 12px;
      line-height: 1;
      color: #5b78a6;
      cursor: pointer;
      background: transparent;
      border: 0;

      &:hover {
        color: var(--el-color-primary);
        background: #f2f6ff;
        border-radius: 999px;
      }
    }
  }

  .process-graph-empty {
    width: 100%;
  }

  .process-stage-node-wrap {
    position: relative;
    width: 156px;
    height: 92px;
  }

  .process-stage-node {
    position: relative;
    display: grid;
    grid-template-rows: 30px minmax(32px, auto) 18px;
    gap: 4px;
    place-items: center;
    width: 156px;
    min-width: 100%;
    height: 92px;
    min-height: 100%;
    padding: 4px 10px 5px;
    overflow: hidden;
    font: inherit;
    color: var(--el-text-color-primary);
    text-align: center;
    cursor: pointer;
    background: linear-gradient(180deg, #fff 0%, #f8fbff 100%);
    border: 1px solid rgb(203 213 225 / 94%);
    border-radius: 8px;
    box-shadow:
      0 10px 24px rgb(15 23 42 / 6%),
      inset 0 1px 0 rgb(255 255 255 / 92%);
    transition:
      background-color 0.2s ease,
      border-color 0.2s ease,
      color 0.2s ease,
      box-shadow 0.2s ease;
    -webkit-font-smoothing: antialiased;
    text-rendering: optimizeLegibility;

    &::before {
      position: absolute;
      top: 0;
      left: 14px;
      right: 14px;
      height: 3px;
      pointer-events: none;
      content: '';
      background: #cbd5e1;
      border-radius: 0 0 999px 999px;
      opacity: 0.9;
    }

    &.active,
    &:hover {
      color: var(--el-color-primary);
      border-color: #60a5fa;
      box-shadow:
        0 0 0 3px rgb(37 99 235 / 9%),
        0 14px 30px rgb(37 99 235 / 12%),
        inset 0 1px 0 rgb(255 255 255 / 92%);
    }

    &.is-running,
    &.is-preparing,
    &.is-paused {
      color: #2563eb;
      background: linear-gradient(180deg, #fff 0%, #eef6ff 100%);
      border-color: #93c5fd;
      box-shadow:
        0 16px 34px rgb(37 99 235 / 14%),
        inset 0 1px 0 rgb(255 255 255 / 92%);

      &::before {
        background: #2563eb;
      }
    }

    &.is-queued,
    &.is-skipped {
      color: #7b8794;
    }

    &.needs-input {
      color: #1d4ed8;

      .stage-node-dot {
        box-shadow: 0 0 0 6px rgb(59 130 246 / 14%);
      }
    }

    &.is-success {
      color: #116b34;
      background: linear-gradient(180deg, #fff 0%, #f4fff7 100%);
      border-color: rgb(134 239 172 / 78%);

      &::before {
        background: #22c55e;
      }
    }

    &.is-failed,
    &.is-aborted {
      color: #a52222;
      background: linear-gradient(180deg, #fff 0%, #fff6f6 100%);
      border-color: rgb(252 165 165 / 72%);

      &::before {
        background: #ef4444;
      }
    }

    &.is-post {
      color: #9a4b12;
      background: linear-gradient(180deg, #fff 0%, #fff8ed 100%);
      border-color: rgb(253 186 116 / 68%);

      &::before {
        background: #f97316;
      }
    }

    &.is-disabled {
      cursor: not-allowed;
      opacity: 0.72;

      &:hover {
        color: inherit;
        box-shadow:
          0 10px 24px rgb(15 23 42 / 6%),
          inset 0 1px 0 rgb(255 255 255 / 92%);
      }
    }
  }

  .stage-node-dot {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    font-family: var(--ax-font-mono);
    font-size: 10px;
    font-weight: 700;
    line-height: 1;
    color: #7b8794;
    background: #fff;
    border: 1px solid #d7dee8;
    border-radius: 50%;
    box-shadow: inset 0 1px 0 rgb(255 255 255 / 90%);
  }

  .process-stage-node.is-success .stage-node-dot {
    color: #12833d;
    background: #f0fdf4;
    border-color: #86efac;
    box-shadow:
      0 0 0 4px rgb(34 197 94 / 8%),
      inset 0 1px 0 rgb(255 255 255 / 90%);
  }

  .process-stage-node.is-running .stage-node-dot,
  .process-stage-node.is-preparing .stage-node-dot,
  .process-stage-node.is-paused .stage-node-dot {
    color: #2563eb;
    background: #eff6ff;
    border-color: #93c5fd;
    box-shadow:
      0 0 0 4px rgb(37 99 235 / 11%),
      inset 0 1px 0 rgb(255 255 255 / 90%);
    animation: process-node-pulse 1.35s ease-in-out infinite;
  }

  .process-stage-node.is-queued .stage-node-dot,
  .process-stage-node.is-skipped .stage-node-dot {
    color: #7b8794;
    background: #f8fafc;
    border-color: #d7dee8;
  }

  .process-stage-node.is-failed .stage-node-dot,
  .process-stage-node.is-aborted .stage-node-dot {
    color: #dc2626;
    background: #fef2f2;
    border-color: #fca5a5;
  }

  .process-stage-node.is-post .stage-node-dot {
    color: #9a4b12;
    background: #fff7ed;
    border-color: #fdba74;
  }

  .stage-node-name,
  .stage-node-duration {
    max-width: 100%;
    overflow: hidden;
  }

  .stage-node-name {
    display: -webkit-box;
    font-size: 13px;
    font-weight: 700;
    line-height: 1.25;
    color: currentcolor;
    overflow-wrap: anywhere;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }

  .stage-node-duration {
    font-size: 11px;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .stage-node-branch {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    max-width: 100%;
    min-height: 16px;
    padding: 2px 8px;
    overflow: hidden;
    font-size: 10px;
    line-height: 1;
    color: #52627a;
    text-overflow: ellipsis;
    white-space: nowrap;
    background: #eef4ff;
    border: 1px solid #d7e4fb;
    border-radius: 999px;
  }

  .stage-node-branch.is-input {
    color: #264da0;
    background: #edf3ff;
    border-color: #adc2ff;
  }

  @keyframes process-flow-dash {
    to {
      stroke-dashoffset: -19;
    }
  }

  @keyframes process-node-pulse {
    0%,
    100% {
      box-shadow:
        0 0 0 4px rgb(37 99 235 / 9%),
        inset 0 1px 0 rgb(255 255 255 / 90%);
    }

    50% {
      box-shadow:
        0 0 0 7px rgb(37 99 235 / 15%),
        inset 0 1px 0 rgb(255 255 255 / 90%);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .process-flow :deep(.process-edge-active .vue-flow__edge-path),
    .process-stage-node.is-running .stage-node-dot,
    .process-stage-node.is-preparing .stage-node-dot,
    .process-stage-node.is-paused .stage-node-dot {
      animation: none;
    }
  }

  .input-pane {
    height: 100%;
    min-height: 0;
    padding: 14px 0 0;
    overflow: auto;
  }

  .input-attention {
    display: grid;
    gap: 4px;
    padding: 12px 14px;
    margin-bottom: 12px;
    color: #284a92;
    background: #eef4ff;
    border: 1px solid #b7c9ff;
    border-radius: 6px;

    strong {
      color: #1d3f85;
    }

    span {
      font-size: 13px;
      color: #52627a;
    }
  }

  .input-card {
    display: grid;
    gap: 12px;
    padding: 14px;
    margin-bottom: 12px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .input-card-head {
    min-width: 0;

    strong,
    span {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      color: var(--el-text-color-primary);
    }

    span {
      margin-top: 4px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .input-form {
    :deep(.el-form-item:last-child) {
      margin-bottom: 0;
    }
  }

  .input-param-help {
    width: 100%;
    margin-top: 4px;
    font-size: 12px;
    line-height: 1.5;
    color: var(--el-text-color-secondary);
  }

  .input-card-actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
  }

  .records-card {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-width: 0;
    min-height: 0;
    overflow: hidden;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .records-head {
    display: flex;
    flex: none;
    gap: 14px;
    align-items: center;
    justify-content: flex-start;
    min-height: 64px;
    padding: 0 16px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .records-title {
    flex: 1;
    min-width: 0;

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
      font-size: 13px;
      color: var(--el-text-color-secondary);
    }
  }

  .records-actions {
    display: inline-flex;
    flex: none;
    gap: 8px;
    align-items: center;
  }

  .records-table {
    flex: 1;
    min-height: 0;
  }

  :deep(.pipeline-config-dialog) {
    --el-dialog-padding-primary: 0;

    display: flex;
    flex-direction: column;
    max-height: min(86vh, 820px);
    overflow: hidden;
    border-radius: 8px;
  }

  :deep(.pipeline-config-dialog .el-dialog__header) {
    flex: none;
    padding: 20px 28px;
    margin-right: 0;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  :deep(.pipeline-config-dialog .el-dialog__title) {
    font-size: 18px;
    font-weight: 700;
    line-height: 1.2;
    color: var(--el-text-color-primary);
  }

  :deep(.pipeline-config-dialog .el-dialog__headerbtn) {
    top: 16px;
    right: 20px;
  }

  :deep(.pipeline-config-dialog .el-dialog__body) {
    flex: 1;
    min-height: 0;
    padding: 22px 28px;
    overflow: auto;
  }

  :deep(.main-config-dialog .el-dialog__body) {
    flex: none;
    min-height: auto;
    padding: 0;
    overflow: hidden;
  }

  :deep(.template-source-dialog) {
    width: min(720px, calc(100vw - 32px));
    max-height: calc(100vh - 40px);
  }

  :deep(.template-source-dialog .el-dialog__body) {
    padding: 0;
    overflow: visible;
    background: var(--el-bg-color);
  }

  :deep(.template-source-dialog .el-dialog__footer) {
    flex: none;
    padding: 12px 24px;
    background: var(--el-bg-color);
    border-top: 1px solid var(--el-border-color-lighter);
  }

  :deep(.jenkins-config-dialog .el-dialog__body),
  :deep(.param-config-dialog .el-dialog__body),
  :deep(.trigger-config-dialog .el-dialog__body) {
    overflow: hidden;
    background: #f8fafc;
  }

  :deep(.run-param-dialog .el-dialog__body) {
    overflow: hidden;
  }

  :deep(.pipeline-config-dialog .el-dialog__footer) {
    flex: none;
    padding: 16px 28px 20px;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  :deep(.pipeline-config-dialog .dialog-footer),
  :deep(.pipeline-config-dialog .el-dialog__footer) {
    text-align: right;
  }

  .tekton-config-tabs {
    min-height: 420px;

    :deep(.el-tabs__header) {
      margin-bottom: 14px;
    }
  }

  .tekton-config-intro {
    display: flex;
    gap: 14px;
    align-items: center;
    justify-content: space-between;
    padding: 12px 14px;
    margin-bottom: 14px;
    background: #f8fafc;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;

    div {
      display: grid;
      min-width: 0;
      gap: 4px;
    }

    strong,
    span {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      color: var(--el-text-color-primary);
      font-size: 14px;
    }

    span {
      color: var(--el-text-color-secondary);
      font-size: 12px;
    }
  }

  .tekton-config-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0 14px;

    &.three {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }

    :deep(.el-form-item) {
      margin-bottom: 14px;
    }
  }

  .tekton-config-section {
    display: grid;
    gap: 10px;
    padding: 12px;
    margin-top: 6px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .tekton-section-head.compact {
    min-height: 28px;

    strong {
      font-size: 13px;
    }
  }

  .tekton-param-list {
    display: grid;
    gap: 8px;
  }

  .tekton-param-row {
    display: grid;
    gap: 8px;
    align-items: center;

    &.two {
      grid-template-columns: minmax(120px, 180px) minmax(0, 1fr);
    }

    &.three {
      grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) auto;
    }

    &.four {
      grid-template-columns: minmax(0, 1fr) minmax(0, 1.4fr) minmax(0, 1fr) auto;
    }
  }

  .snapshot-key {
    min-width: 0;
    overflow: hidden;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .tekton-config-note {
    margin: 0;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.5;
  }

  .tekton-yaml-field {
    display: grid;
    gap: 8px;
    min-width: 0;
  }

  .tekton-yaml-title {
    display: flex;
    gap: 10px;
    align-items: baseline;
    min-width: 0;

    strong {
      flex: 0 0 auto;
      color: var(--el-text-color-primary);
      font-size: 13px;
    }

    span {
      min-width: 0;
      overflow: hidden;
      color: var(--el-text-color-secondary);
      font-size: 12px;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .yaml-preview-content {
    min-height: 520px;
    max-height: 62vh;
    padding: 14px;
    margin: 0;
    overflow: auto;
    color: #dce7ff;
    white-space: pre-wrap;
    word-break: break-word;
    background: #1f2937;
    border: 1px solid #334155;
    border-radius: 8px;
  }

  .config-dialog-form,
  .config-dialog-shell {
    display: flex;
    flex-direction: column;
    gap: 12px;
    min-height: 0;
    max-height: min(62vh, 600px);
    overflow: auto;
    scrollbar-gutter: stable;
  }

  .config-panel {
    padding: 14px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;

    :deep(.el-form-item:last-child) {
      margin-bottom: 0;
    }
  }

  .config-panel-head {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;

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

  .mini-list-head,
  .param-toolbar {
    display: flex;
    gap: 10px;
    align-items: center;
    justify-content: space-between;
  }

  .mini-list-head {
    min-height: 32px;
    margin-bottom: 10px;
    font-weight: 600;
  }

  .mini-list {
    display: grid;
    gap: 8px;
  }

  .mini-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) minmax(0, 1fr) auto;
    gap: 8px;
    align-items: center;
    padding: 10px;
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .tool-row {
    grid-template-columns: minmax(120px, 0.8fr) minmax(0, 1.2fr) minmax(0, 1fr) auto;
  }

  .trigger-row {
    grid-template-columns: 64px minmax(150px, 0.9fr) minmax(0, 1.3fr) auto;
    align-items: start;
  }

  .trigger-help {
    grid-column: 2 / 4;
    margin: 0;
    font-size: 12px;
    line-height: 1.5;
    color: var(--el-text-color-secondary);
  }

  .option-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
  }

  .agent-preview-card {
    display: flex;
    gap: 14px;
    align-items: center;
    justify-content: space-between;
    padding: 12px 14px;
    margin-top: -2px;
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .agent-preview-main {
    min-width: 0;

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

  .agent-container-tags {
    display: flex;
    flex: none;
    flex-wrap: wrap;
    gap: 6px;
    justify-content: flex-end;
    max-width: 50%;
  }

  .option-card {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 10px;
    align-items: center;
    min-height: 84px;
    padding: 12px 14px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;

    :deep(.el-input) {
      grid-column: 1 / 3;
    }
  }

  .option-main {
    min-width: 0;

    strong,
    span {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    span {
      margin-top: 4px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .param-toolbar {
    padding: 0 2px 2px;

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

  .param-list {
    display: grid;
    gap: 12px;
  }

  .param-config-panel {
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }

  .param-row {
    display: grid;
    grid-template-columns: minmax(190px, 0.42fr) minmax(0, 1fr);
    gap: 14px;
    align-items: center;
    padding: 14px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .param-row.is-full-row {
    grid-template-columns: 1fr;
    align-items: stretch;

    .param-row-head {
      padding-bottom: 2px;
    }
  }

  .param-row-head {
    display: flex;
    gap: 10px;
    align-items: flex-start;
    justify-content: space-between;
    min-width: 0;
  }

  .param-row-title {
    min-width: 0;

    strong {
      max-width: 100%;
      overflow: hidden;
      font-size: 15px;
      color: var(--el-text-color-primary);
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    span {
      max-width: 100%;
      overflow: hidden;
      font-size: 12px;
      color: var(--el-text-color-secondary);
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .param-tags {
    display: flex;
    flex: none;
    flex-wrap: wrap;
    gap: 6px;
    align-items: center;
    justify-content: flex-end;
  }

  .run-param-shell {
    display: flex;
    flex-direction: column;
    min-width: 0;
    height: min(68vh, 620px);
    min-height: 320px;
    max-height: min(68vh, 620px);
  }

  .run-param-card {
    position: relative;
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
    padding: 0;
    overflow: hidden;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .run-param-init-error {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 8px;
    align-items: center;
    justify-content: center;
    min-height: 220px;
    padding: 28px;
    text-align: center;

    strong {
      font-size: 15px;
      color: var(--el-text-color-primary);
    }

    span {
      font-size: 13px;
      color: var(--el-text-color-secondary);
    }
  }

  .run-param-init-error {
    strong {
      color: var(--el-color-danger);
    }
  }

  .run-param-toolbar {
    display: flex;
    flex: none;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    padding: 0 4px;
    margin-bottom: 12px;

    strong {
      font-size: 16px;
      color: var(--el-text-color-primary);
    }
  }

  .run-param-summary {
    font-size: 12px;
    font-weight: 400;
    color: var(--el-text-color-secondary);
  }

  .run-param-form {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
    overflow: hidden;
  }

  .run-param-empty {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 220px;
  }

  .run-param-list {
    display: grid;
    flex: 1;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
    align-content: start;
    min-height: 0;
    padding: 16px 18px 18px;
    overflow: auto;
    overscroll-behavior: contain;
    scrollbar-gutter: stable;
    -webkit-overflow-scrolling: touch;
  }

  .run-param-section-title {
    grid-column: 1 / -1;
    padding: 2px 2px 0;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    font-weight: 600;
  }

  .run-param-field {
    min-width: 0;
    padding: 14px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .run-param-field.is-full-row {
    grid-column: 1 / -1;

    .run-param-code {
      margin-bottom: 12px;
    }
  }

  .run-param-meta {
    display: flex;
    gap: 10px;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 4px;
  }

  .run-param-tags {
    display: flex;
    flex: none;
    flex-wrap: wrap;
    gap: 6px;
    align-items: center;
    justify-content: flex-end;
  }

  .run-param-label {
    display: flex;
    align-items: center;
    min-width: 0;
    min-height: 24px;
    font-weight: 600;
    color: var(--el-text-color-primary);

    span {
      display: block;
      min-width: 0;
      overflow: hidden;
      font-size: 15px;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .run-param-code {
    margin-bottom: 10px;
    overflow: hidden;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .derived-param-preview {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    align-items: center;
    min-height: 34px;
    padding: 8px 10px;
    font-size: 12px;
    line-height: 1.4;
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
  }

  .run-param-field.required .run-param-label span::before {
    margin-right: 4px;
    color: var(--el-color-danger);
    content: '*';
  }

  .object-list-value-editor {
    display: grid;
    width: 100%;
    min-width: 0;
    gap: 8px;

    :deep(.el-input),
    :deep(.el-select),
    :deep(.el-input-number),
    :deep(.el-textarea) {
      width: 100%;
    }
  }

  .object-list-value-head {
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: space-between;
    min-width: 0;

    span {
      overflow: hidden;
      color: var(--el-text-color-secondary);
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .run-param-control {
    min-width: 0;
    margin-bottom: 0;

    :deep(.el-form-item__content) {
      display: flex;
      align-items: center;
      min-height: 40px;
      margin-left: 0 !important;
    }

    :deep(.el-input),
    :deep(.el-select),
    :deep(.el-textarea) {
      width: 100%;
    }

    :deep(.el-input__wrapper),
    :deep(.el-textarea__inner),
    :deep(.el-select__wrapper) {
      background: var(--el-bg-color);
      border-color: var(--el-border-color-light);
      border-radius: 8px;
    }

    :deep(.el-input__inner),
    :deep(.el-textarea__inner),
    :deep(.el-select__selected-item),
    :deep(.el-select__selected-item span),
    :deep(.el-select__input) {
      font-weight: 500;
      color: var(--el-text-color-primary);
      -webkit-text-fill-color: var(--el-text-color-primary);
    }

    :deep(.el-input__inner::placeholder),
    :deep(.el-textarea__inner::placeholder),
    :deep(.el-select__placeholder) {
      font-weight: 400;
      color: var(--el-text-color-placeholder);
      -webkit-text-fill-color: var(--el-text-color-placeholder);
    }

    :deep(.el-select__placeholder.is-transparent) {
      font-weight: 500;
      color: var(--el-text-color-primary);
      -webkit-text-fill-color: var(--el-text-color-primary);
      opacity: 1;
    }

    :deep(.el-input.is-disabled .el-input__inner),
    :deep(.el-textarea.is-disabled .el-textarea__inner),
    :deep(.el-select.is-disabled .el-select__selected-item),
    :deep(.el-select.is-disabled .el-select__selected-item span),
    :deep(.el-select.is-disabled .el-select__input),
    :deep(.el-select.is-disabled .el-select__placeholder.is-transparent) {
      color: var(--el-text-color-disabled);
      -webkit-text-fill-color: var(--el-text-color-disabled);
    }

    :deep(.el-switch) {
      min-height: 32px;
    }
  }

  .workspace-config-summary-card {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 10px;
    align-items: center;
    width: 100%;
    padding: 10px 12px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .workspace-config-summary-main {
    min-width: 0;

    strong,
    span {
      display: block;
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      color: var(--el-text-color-primary);
      font-size: 14px;
    }

    span {
      margin-top: 4px;
      color: var(--el-text-color-secondary);
      font-size: 12px;
    }
  }

  .workspace-config-summary-tags {
    display: flex;
    grid-column: 1 / -1;
    flex-wrap: wrap;
    gap: 6px;
    min-width: 0;
  }

  .workspace-item-editor-form {
    display: grid;
    gap: 14px;
  }

  .workspace-item-editor-head {
    display: grid;
    gap: 4px;
    padding: 10px 12px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;

    strong,
    span {
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      color: var(--el-text-color-primary);
      font-size: 15px;
    }

    span {
      color: var(--el-text-color-secondary);
      font-size: 12px;
    }
  }

  .workspace-item-editor-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(260px, 1fr));
    gap: 12px 16px;
    align-items: end;

    :deep(.el-form-item) {
      margin-bottom: 0;
    }
  }

  .workspace-item-editor-wide {
    grid-column: 1 / -1;
  }

  @media (width <= 960px) {
    .param-row {
      grid-template-columns: 1fr;
      align-items: stretch;
    }

    .param-row-head {
      flex-direction: column;
      align-items: flex-start;
    }

    .param-tags {
      justify-content: flex-start;
    }

    .run-param-list {
      grid-template-columns: 1fr;
      padding: 16px 16px 0;
    }

    .run-param-toolbar {
      padding: 0 16px;
    }

    .tekton-overview-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .tekton-editor-head,
    .tekton-detail-head {
      align-items: flex-start;
      flex-direction: column;
    }

    .tekton-editor-actions,
    .tekton-detail-actions {
      justify-content: flex-start;
    }

    .run-param-meta {
      flex-direction: column;
      align-items: flex-start;
    }

    .run-param-tags {
      justify-content: flex-start;
    }
  }

  .editor-workflow-board {
    display: flex;
    flex-direction: column;
    flex: 1;
    width: 100%;
    height: 100%;
    min-height: 620px;
  }

  .dialog-alert {
    margin-bottom: 12px;
  }

  .main-config-content,
  .template-source-content {
    box-sizing: border-box;
    width: 100%;
  }

  .main-config-content {
    display: flex;
    align-items: flex-start;
    padding: 28px 28px 18px;

    :deep(.el-form) {
      width: 100%;
    }

    :deep(.el-form-item__label) {
      padding-right: 12px;
      font-weight: 500;
      color: var(--el-text-color-regular);
    }

    :deep(.el-input__wrapper),
    :deep(.el-select__wrapper) {
      min-height: 36px;
      border-radius: 6px;
    }
  }

  .dialog-form-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px 24px;

    :deep(.el-form-item) {
      margin-bottom: 6px;
    }

    :deep(.el-input),
    :deep(.el-select) {
      width: 100%;
    }
  }

  .form-grid-wide {
    grid-column: 1 / -1;
  }

  .scan-config-panel {
    display: flex;
    flex-direction: column;
    gap: 10px;
    min-width: 0;
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    padding: 12px;
  }

  .tekton-runtime-panel {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 18px;
    padding: 14px 16px;
    margin: 4px 0 2px;
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .tekton-runtime-item {
    min-width: 0;
  }

  .tekton-runtime-item span,
  .tekton-runtime-item strong {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .tekton-runtime-item span {
    margin-bottom: 6px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .tekton-runtime-item strong {
    font-size: 13px;
    color: var(--el-text-color-primary);
  }

  .scan-config-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 13px;
    color: var(--el-text-color-primary);
  }

  .scan-config-row {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 8px;
    align-items: center;
    min-width: 0;
  }

  .template-source-content {
    display: grid;
    gap: 12px;
    padding: 18px 24px 16px;
    background: var(--el-bg-color);
  }

  .template-source-head {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;

    > div {
      display: grid;
      gap: 4px;
      min-width: 0;
    }
  }

  .template-source-head strong {
    font-size: 14px;
    line-height: 1.35;
    color: var(--el-text-color-primary);
  }

  .template-source-head span {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .template-source-toolbar {
    display: grid;
    grid-template-columns: minmax(280px, 1fr) auto;
    gap: 10px;
    align-items: center;
    min-width: 0;
  }

  .template-source-search {
    min-width: 0;
  }

  .template-source-filter {
    min-width: 380px;

    :deep(.el-segmented) {
      width: 100%;
      padding: 3px;
      background: var(--el-fill-color-light);
      border: 1px solid var(--el-border-color-light);
      border-radius: 8px;
    }

    :deep(.el-segmented__item) {
      min-width: 84px;
      height: 30px;
      border-radius: 6px;
    }

    :deep(.el-segmented__item-selected) {
      background: var(--el-bg-color);
      box-shadow: none;
    }

    :deep(.el-segmented__item-label) {
      overflow: hidden;
      font-size: 12px;
      font-weight: 500;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .template-source-result-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    min-height: 22px;
    font-size: 12px;
    color: var(--el-text-color-secondary);

    span {
      font-weight: 600;
      color: var(--el-text-color-regular);
    }

    em {
      font-style: normal;
    }
  }

  .template-source-footer {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    min-width: 0;
  }

  .template-source-current {
    display: flex;
    flex: 1;
    gap: 8px;
    align-items: center;
    min-width: 0;
    font-size: 12px;
    color: var(--el-text-color-secondary);

    strong,
    code {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      max-width: 260px;
      font-size: 13px;
      color: var(--el-text-color-primary);
    }

    code {
      max-width: 220px;
      padding: 2px 6px;
      font-family: inherit;
      color: var(--el-text-color-regular);
      background: var(--el-fill-color-light);
      border: 1px solid var(--el-border-color-light);
      border-radius: 4px;
    }
  }

  .template-source-actions {
    display: flex;
    flex: none;
    gap: 8px;
    align-items: center;
    justify-content: flex-end;
  }

  :deep(.template-source-dialog .el-dialog__footer) {
    flex: none;
    padding: 12px 24px;
    background: var(--el-bg-color);
    border-top: 1px solid var(--el-border-color-light);
  }

  .template-source-list {
    display: grid;
    gap: 8px;
    align-content: start;
    max-height: min(430px, calc(100vh - 390px));
    min-height: 0;
    padding: 8px;
    overflow: auto;
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .template-source-card {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 150px;
    gap: 12px;
    align-items: center;
    align-self: start;
    min-width: 0;
    min-height: 86px;
    padding: 12px 14px;
    font: inherit;
    text-align: left;
    cursor: pointer;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    transition:
      background 0.18s ease,
      border-color 0.18s ease,
      box-shadow 0.18s ease;

    &:hover {
      background: var(--el-fill-color-extra-light);
      border-color: var(--el-border-color-hover);
      box-shadow: 0 8px 18px rgb(15 23 42 / 8%);
    }

    &.active {
      background: var(--el-bg-color);
      border-color: var(--el-color-primary);
      box-shadow: 0 0 0 2px var(--el-color-primary-light-9);
    }
  }

  .template-source-card-main {
    display: grid;
    gap: 7px;
    min-width: 0;
  }

  .template-source-card-head {
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: space-between;
    min-width: 0;

    strong,
    code {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      font-size: 14px;
      color: var(--el-text-color-primary);
    }

    code {
      flex: none;
      max-width: 180px;
      font-family: inherit;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .template-source-card-desc {
    display: -webkit-box;
    font-size: 12px;
    line-height: 1.5;
    color: var(--el-text-color-secondary);
    overflow: hidden;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }

  .template-source-card-meta,
  .template-source-card-steps {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    min-width: 0;
  }

  .template-source-card-meta {
    justify-content: flex-end;
    max-width: 150px;

    span {
      display: inline-flex;
      align-items: center;
      height: 24px;
      padding: 0 8px;
      font-size: 12px;
      color: var(--el-text-color-regular);
      background: var(--el-fill-color-light);
      border: 1px solid var(--el-border-color-light);
      border-radius: 4px;
    }
  }

  .template-source-card-steps {
    display: flex;
    flex-wrap: nowrap;
    gap: 6px;
    align-items: center;
    min-width: 0;
    overflow: hidden;
  }

  .template-source-card-steps span {
    display: inline-flex;
    flex: none;
    align-items: center;
    max-width: 132px;
    height: 22px;
    padding: 0 8px;
    font-size: 12px;
    line-height: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    border-radius: 4px;
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-light);
  }

  @media (width <= 960px) {
    .template-source-content {
      padding: 16px;
    }

    .template-source-toolbar {
      grid-template-columns: 1fr;
    }

    .template-source-filter {
      min-width: 0;
    }

    .template-source-card {
      grid-template-columns: 1fr;
    }

    .template-source-card-meta {
      justify-content: flex-start;
      max-width: none;
    }

    .template-source-footer {
      align-items: flex-start;
      flex-direction: column;
    }
  }

  .dynamic-option-tip {
    padding: 8px 12px;
    font-size: 12px;
    line-height: 1.5;
    color: var(--el-text-color-secondary);
    white-space: normal;
  }

  .dynamic-option-tip.is-error {
    color: var(--el-color-danger);
  }

  .dynamic-select-control {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 74px;
    gap: 8px;
    align-items: center;
    width: 100%;
    min-width: 0;
    overflow: visible;
  }

  .dynamic-select-refresh {
    width: 74px;
    height: 34px;
    padding: 0;
  }

  .platform-param-select {
    width: 100%;
    min-width: 0;

    :deep(.el-select__wrapper) {
      position: relative;
      min-height: 42px;
      padding: 10px 12px;
      background: var(--el-bg-color);
      border-radius: 14px;
      box-shadow: 0 0 0 1px #e2e8f0 inset;
      transition:
        box-shadow 0.18s ease,
        background-color 0.18s ease;
    }

    :deep(.el-select__placeholder) {
      color: var(--el-text-color-placeholder);
      -webkit-text-fill-color: var(--el-text-color-placeholder);
    }

    :deep(.el-select__selection) {
      min-width: 0;
    }

    :deep(.el-select__selected-item),
    :deep(.el-select__selected-item span),
    :deep(.el-select__input),
    :deep(.el-select__placeholder.is-transparent) {
      min-width: 0;
      font-weight: 500;
      color: var(--el-text-color-primary);
      -webkit-text-fill-color: var(--el-text-color-primary);
      opacity: 1;
    }

    :deep(.el-select__selected-item span) {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    :deep(.el-select.is-disabled .el-select__selected-item),
    :deep(.el-select.is-disabled .el-select__selected-item span),
    :deep(.el-select.is-disabled .el-select__input),
    :deep(.el-select.is-disabled .el-select__placeholder.is-transparent) {
      color: var(--el-text-color-disabled);
      -webkit-text-fill-color: var(--el-text-color-disabled);
    }

    :deep(.el-select__caret) {
      width: 16px;
      height: 16px;
      color: #334155;
    }

    :deep(.el-select__caret svg) {
      width: 14px;
      height: 14px;
    }

    :deep(.el-select__wrapper:hover) {
      box-shadow: 0 0 0 1px #94a3b8 inset;
    }

    :deep(.el-select__wrapper.is-focused) {
      box-shadow:
        0 0 0 1px #2563eb inset,
        0 0 0 4px rgb(37 99 235 / 14%);
    }
  }

  :global(.el-popper.devops-platform-param-dropdown) {
    --devops-platform-dropdown-height: 260px;

    max-width: min(620px, calc(100vw - 32px));
    padding: 0;
    background: #ffffff !important;
    border: 1px solid #e2e8f0;
    border-radius: 14px;
    box-shadow: 0 16px 34px rgb(15 23 42 / 12%);
    overflow: hidden !important;

    .el-select-dropdown,
    .el-scrollbar,
    .el-scrollbar__view {
      background: #ffffff !important;
    }

    .el-scrollbar,
    .el-scrollbar__view {
      width: 100% !important;
    }

    .el-select-dropdown {
      overflow: hidden !important;
      transform: none !important;
    }

    .el-scrollbar {
      overflow: hidden !important;
    }

    .el-select-dropdown__wrap {
      max-height: var(--devops-platform-dropdown-height) !important;
      background: #ffffff !important;
      overflow-y: auto !important;
    }

    .el-select-dropdown__list {
      width: 100% !important;
      padding: 0 !important;
      margin: 0 !important;
      background: #ffffff !important;
      box-sizing: border-box;
      text-align: left !important;
      transform: none !important;
    }

    .el-select-dropdown__item {
      display: flex !important;
      align-items: center !important;
      justify-content: flex-start !important;
      width: 100% !important;
      height: auto !important;
      min-height: 44px;
      padding: 10px 12px;
      overflow: visible !important;
      line-height: 1.35 !important;
      color: #334155 !important;
      border-radius: 0 !important;
      background: #ffffff !important;
      margin-bottom: 0 !important;
      transform: none !important;
      text-align: left !important;
      white-space: normal !important;
      writing-mode: horizontal-tb !important;
      transition:
        background-color 0.16s ease,
        color 0.16s ease;
    }

    .el-select-dropdown__item.is-hovering,
    .el-select-dropdown__item.hover,
    .el-select-dropdown__item:hover {
      color: #1d4ed8 !important;
      background: #eff6ff !important;
      box-shadow: inset 4px 0 0 #3b82f6;
    }

    .el-select-dropdown__item.is-selected {
      font-weight: 600;
      color: #0f172a !important;
      background: #ffffff !important;
    }

    .el-select-dropdown__empty {
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 18px 12px;
      color: var(--el-text-color-secondary);
    }
  }

  :global(.el-popper.devops-platform-param-dropdown[aria-hidden='true']) {
    display: none !important;
  }

  :global(.el-popper.devops-platform-param-dropdown .el-select-dropdown__loading) {
    min-height: var(--devops-platform-dropdown-height, 260px);
    padding: 0 !important;
    border-left: 0 !important;
    box-shadow: none !important;
  }

  :global(.el-popper.devops-platform-param-dropdown .devops-select-loading) {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: var(--devops-platform-dropdown-height, 260px);
    gap: 8px;
    border-left: 0 !important;
    box-shadow: none !important;
    color: var(--el-color-primary);
    font-size: 13px;
    line-height: 1;
  }

  :global(.el-popper.devops-platform-param-dropdown .devops-select-loading__icon) {
    width: 24px;
    height: 24px;
    border: 2px solid var(--el-color-primary-light-7);
    border-top-color: var(--el-color-primary);
    border-radius: 50%;
    animation: devops-select-loading-rotate 0.9s linear infinite;
  }

  :global(
    .el-popper.devops-platform-param-dropdown
      .el-select-dropdown:has(.el-select-dropdown__loading)
      .el-scrollbar
  ) {
    display: none !important;
  }

  :global(
    .el-popper.devops-platform-param-dropdown
      .el-select-dropdown:has(.el-select-dropdown__loading)
      .el-select-dropdown__item
  ) {
    border-left: 0 !important;
    box-shadow: none !important;
  }

  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown
  ) {
    padding: 0 !important;
    margin: 0 !important;
    border: 0 !important;
    box-shadow: none !important;
  }

  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__wrap
  ) {
    max-height: var(--devops-platform-dropdown-height) !important;
    overflow-y: auto !important;
  }

  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__list
  ) {
    padding: 0 !important;
    margin: 0 !important;
    background: #ffffff !important;
    transform: none !important;
  }

  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__list
      .el-select-dropdown__item
  ) {
    display: flex !important;
    align-items: center !important;
    justify-content: flex-start !important;
    width: 100% !important;
    height: auto !important;
    min-height: 44px !important;
    padding: 10px 12px !important;
    overflow: visible !important;
    line-height: 1.35 !important;
    color: #334155 !important;
    text-overflow: clip !important;
    white-space: normal !important;
    background: #ffffff !important;
    border-radius: 0 !important;
    margin-bottom: 0 !important;
    transform: none !important;
    text-align: left !important;
    writing-mode: horizontal-tb !important;
  }

  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__list
      .el-select-dropdown__item.is-hovering
  ),
  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__list
      .el-select-dropdown__item:hover
  ) {
    color: #1d4ed8 !important;
    background: #eff6ff !important;
    box-shadow: inset 4px 0 0 #3b82f6;
  }

  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__list
      .el-select-dropdown__item.is-selected
  ) {
    font-weight: 600 !important;
    color: #0f172a !important;
    background: #ffffff !important;
    margin-bottom: 0 !important;
  }

  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__list
      .el-select-dropdown__item.is-selected.is-hovering
  ),
  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__list
      .el-select-dropdown__item.is-selected:hover
  ) {
    color: #1d4ed8 !important;
    background: #eff6ff !important;
    box-shadow: inset 4px 0 0 #3b82f6;
  }

  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__item:hover
      ~ .is-selected
  ),
  :global(
    .devops-platform-param-dropdown.el-select__popper:not(.el-tree-select__popper)
      .el-select-dropdown__item.is-selected:has(~ .el-select-dropdown__item:hover)
  ) {
    background: #ffffff !important;
  }

  @keyframes devops-select-loading-rotate {
    to {
      transform: rotate(360deg);
    }
  }

  .dynamic-option-item {
    display: flex;
    flex-direction: column;
    justify-content: center;
    min-height: 44px;
    overflow: visible;
  }

  .dynamic-option-item span,
  .dynamic-option-item small {
    overflow: visible;
    line-height: 1.35;
    text-overflow: clip;
    white-space: normal;
    word-break: break-word;
  }

  .dynamic-option-item small {
    color: var(--el-text-color-secondary);
  }

  .dialog-param-list {
    max-height: 520px;
    overflow: auto;
  }

  .muted-text {
    color: var(--el-text-color-secondary);
  }

  :global(.pipeline-log-dialog) {
    display: flex;
    flex-direction: column;
    height: min(780px, calc(100vh - 72px));
    min-height: 560px;
    max-height: calc(100vh - 64px);
    margin-top: 4vh !important;
  }

  :global(.pipeline-log-dialog .el-dialog__header) {
    flex: none;
    padding: 16px 16px 10px;
    margin-right: 0;
  }

  :global(.pipeline-log-dialog .el-dialog__body) {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
    padding: 0 16px 16px;
    overflow: hidden;
  }

  .log-dialog-head {
    display: flex;
    gap: 14px;
    align-items: flex-start;
    justify-content: space-between;
    min-width: 0;

    strong {
      display: inline-flex;
      max-width: 520px;
      overflow: hidden;
      font-size: 16px;
      line-height: 1.3;
      color: var(--el-text-color-primary);
      text-overflow: ellipsis;
      white-space: nowrap;
      vertical-align: middle;
    }
  }

  .log-build-number {
    margin-left: 6px;
    font-size: 13px;
    font-weight: 600;
    color: var(--el-text-color-secondary);
  }

  .log-actions {
    display: inline-flex;
    flex: none;
    flex-wrap: wrap;
    gap: 8px;
    justify-content: flex-end;
    padding-right: 32px;

    :deep(.el-button) {
      height: 32px;
      margin-left: 0;
    }
  }

  .log-tabs {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;

    :deep(.el-tabs__header) {
      flex: none;
      margin: 0 0 12px;
    }

    :deep(.el-tabs__content) {
      flex: 1;
      min-height: 0;
      overflow: hidden;
    }

    :deep(.el-tab-pane) {
      height: 100%;
      min-height: 0;
      overflow: hidden;
    }
  }

  .stage-log-layout {
    display: grid;
    grid-template-columns: 280px minmax(0, 1fr);
    gap: 14px;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .stage-list {
    display: flex;
    grid-column: 1;
    grid-row: 1;
    flex-direction: column;
    min-width: 0;
    min-height: 0;
    max-height: 100%;
    padding: 12px;
    overflow-x: hidden;
    overflow-y: auto;
    scrollbar-gutter: stable;
    background: #f8fafc;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .stage-list .section-title {
    position: sticky;
    top: -12px;
    z-index: 1;
    padding-top: 12px;
    padding-bottom: 8px;
    margin: -12px 0 10px;
    background: #f8fafc;
  }

  .stage-item {
    display: grid;
    gap: 7px;
    width: 100%;
    min-height: 60px;
    padding: 10px 12px;
    margin: 0 0 8px;
    font: inherit;
    text-align: left;
    cursor: pointer;
    background: #fff;
    border: 1px solid rgb(218 226 237 / 92%);
    border-radius: 8px;
    box-shadow: 0 6px 14px rgb(38 50 74 / 4%);

    &.active {
      border-color: #60a5fa;
      box-shadow:
        0 0 0 3px rgb(37 99 235 / 9%),
        0 10px 24px rgb(37 99 235 / 10%);
    }

    &.running {
      background: linear-gradient(180deg, #fff 0%, #eef6ff 100%);
      border-color: #93c5fd;
    }

    &.disabled {
      cursor: not-allowed;
      opacity: 0.58;
    }
  }

  .stage-name {
    display: block;
    overflow: hidden;
    font-size: 13px;
    font-weight: 700;
    line-height: 1.25;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .stage-meta {
    display: inline-flex;
    gap: 8px;
    align-items: center;
    min-width: 0;
    overflow: hidden;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .stage-duration {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .log-pane,
  .full-log-pane {
    display: flex;
    flex-direction: column;
    min-width: 0;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .log-pane {
    grid-column: 2;
    grid-row: 1;
  }

  .section-title {
    position: relative;
    padding-left: 10px;
    margin-bottom: 12px;
    font-weight: 600;
    color: var(--el-text-color-primary);

    &::before {
      position: absolute;
      top: 3px;
      bottom: 3px;
      left: 0;
      width: 3px;
      content: '';
      background: var(--el-color-primary);
      border-radius: 999px;
    }
  }

  .log-section-head {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    min-height: 24px;
    margin-bottom: 12px;

    .section-title {
      margin-bottom: 0;
    }
  }

  .tekton-log-runtime {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) minmax(180px, 240px);
    gap: 10px;
    margin-bottom: 12px;
  }

  .tekton-log-runtime-item {
    display: flex;
    min-width: 0;
    min-height: 52px;
    padding: 8px 10px;
    overflow: hidden;
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    flex-direction: column;

    span {
      margin-bottom: 4px;
      font-size: 12px;
      line-height: 1;
      color: var(--el-text-color-secondary);
    }

    strong {
      min-width: 0;
      overflow: hidden;
      font-size: 13px;
      font-weight: 600;
      line-height: 1.4;
      color: var(--el-text-color-primary);
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    &.container {
      :deep(.el-input__wrapper) {
        min-height: 30px;
      }
    }
  }

  .log-live-pill {
    display: inline-flex;
    flex: none;
    gap: 8px;
    align-items: center;
    justify-self: center;
    width: max-content;
    max-width: calc(100% - 24px);
    min-height: 24px;
    padding: 5px 35px;
    font-family: var(--ax-font-mono);
    font-size: 12px;
    line-height: 1.2;
    color: #5d6f9f;
    white-space: nowrap;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 999px;
  }

  .log-bottom-bar {
    display: grid;
    flex: none;
    grid-template-columns: minmax(0, 1fr) max-content minmax(0, 1fr);
    gap: 12px;
    align-items: center;
    min-height: 36px;
    padding-bottom: 2px;
    margin-top: 10px;
    position: relative;
    z-index: 2;
  }

  .artifact-pane {
    height: 100%;
    min-height: 0;
    padding: 14px 0;
    overflow: auto;
  }

  .artifact-list {
    display: grid;
    gap: 10px;
  }

  .artifact-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto auto;
    gap: 12px;
    align-items: center;
    min-height: 54px;
    padding: 10px 12px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .artifact-main {
    min-width: 0;

    strong,
    span {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    span {
      margin-top: 4px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .quality-pane {
    height: 100%;
    min-height: 0;
    padding: 14px 0;
    overflow: auto;
  }

  .quality-run-metrics {
    display: grid;
    grid-template-columns: repeat(6, minmax(0, 1fr));
    gap: 12px;
    margin-bottom: 12px;
  }

  .quality-run-metric {
    min-height: 72px;
    padding: 12px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    background: var(--el-fill-color-extra-light);
    display: flex;
    flex-direction: column;
    justify-content: space-between;

    span {
      color: var(--el-text-color-secondary);
      font-size: 12px;
    }

    strong {
      color: var(--el-text-color-primary);
      font-size: 22px;
      line-height: 1;
    }

    &.danger strong {
      color: var(--el-color-danger);
    }

    &.warning strong {
      color: var(--el-color-warning);
    }
  }

  .quality-run-tabs {
    min-height: 0;
  }

  .quality-report-list,
  .quality-issue-list,
  .quality-image-list,
  .quality-gate-list {
    display: grid;
    gap: 10px;
  }

  .quality-report-row,
  .quality-issue-row,
  .quality-image-row,
  .quality-gate-row {
    display: grid;
    gap: 12px;
    align-items: center;
    min-height: 52px;
    padding: 10px 12px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .quality-report-row {
    grid-template-columns: minmax(0, 1fr) auto auto auto auto;
  }

  .quality-preview-dialog-body {
    height: calc(100vh - 180px);
    min-height: 360px;
  }

  .quality-preview-frame {
    width: 100%;
    height: 100%;
    border: 1px solid var(--el-border-color-light);
    border-radius: 4px;
    background: #fff;
  }

  .quality-issue-row {
    grid-template-columns: auto minmax(0, 1fr) 96px;
  }

  .quality-image-row {
    grid-template-columns: minmax(0, 1fr) auto auto auto;
  }

  .quality-gate-row {
    grid-template-columns: 110px minmax(0, 1fr);

    &.danger strong {
      color: var(--el-color-danger);
    }

    &.warning strong {
      color: var(--el-color-warning);
    }
  }

  .quality-report-main,
  .quality-issue-main,
  .quality-image-main {
    min-width: 0;

    strong,
    span {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    span {
      margin-top: 4px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .quality-issue-tool {
    overflow: hidden;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .terminal-log {
    position: relative;
    flex: 1;
    min-height: 260px;
    overflow: auto;
    font-family: var(--ax-font-mono);
    font-size: 13px;
    line-height: 1.6;
    color: #f8f8f2;
    background:
      linear-gradient(90deg, rgb(24 25 34 / 64%) 0 54px, #44475a 54px 55px, transparent 55px),
      linear-gradient(180deg, rgb(255 255 255 / 2%), transparent 42px), #282a36;
    border: 1px solid #3b3f54;
    border-radius: 10px;
    box-shadow:
      inset 0 0 0 1px rgb(255 255 255 / 4%),
      inset 0 18px 40px rgb(0 0 0 / 16%),
      0 10px 24px rgb(15 23 42 / 8%);

    :deep(.el-loading-mask) {
      background: rgb(40 42 54 / 78%);
      border-radius: 10px;
    }

    :deep(.el-empty__image) {
      width: 104px;
    }

    :deep(.el-empty__description p) {
      color: #bdc0d0;
    }
  }

  .terminal-content {
    position: relative;
    z-index: 1;
    min-height: 100%;
  }

  .live-dots {
    display: inline-flex;
    gap: 7px;
    align-items: center;

    i {
      width: 5px;
      height: 5px;
      background: #8aa4ff;
      border-radius: 50%;
      animation: live-dot-pulse 1.2s ease-in-out infinite;

      &:nth-child(2) {
        animation-delay: 0.18s;
      }

      &:nth-child(3) {
        animation-delay: 0.36s;
      }
    }
  }

  @keyframes live-dot-pulse {
    0%,
    80%,
    100% {
      opacity: 0.35;
      transform: translateY(0);
    }

    40% {
      opacity: 1;
      transform: translateY(-3px);
    }
  }

  .terminal-empty {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    pointer-events: none;
  }

  .log-line {
    display: grid;
    grid-template-columns: 54px minmax(0, 1fr);
    min-height: 22px;
    white-space: pre-wrap;
  }

  .line-no {
    padding-right: 10px;
    color: #7d859f;
    text-align: right;
    user-select: none;
  }

  .line-text {
    min-width: 0;
    padding: 0 12px;
    color: #f8f8f2;
    word-break: break-word;
  }

  .auto-scroll-check {
    justify-self: end;
  }
</style>
